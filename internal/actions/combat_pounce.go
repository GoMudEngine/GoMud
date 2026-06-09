package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// PounceResult holds the outcome of a pounce attempt for the caller to use
// when formatting messages, firing events, and updating UI.
type PounceResult struct {
	// Target is the resolved aggro target. Valid only when Executed is true.
	Target AggroTarget

	// MoveResult is the outcome from ExecuteSkillMove. Valid only when Executed
	// is true.
	MoveResult combat.SkillMoveResult

	// Executed reports whether the pounce was actually performed. False when
	// any early-exit condition fired (OnCooldown, NoTarget, NotPredator).
	Executed bool

	// OnCooldown is true when the special-move cooldown blocked the pounce.
	OnCooldown bool

	// NoTarget is true when there is no aggro target or the target is gone.
	NoTarget bool

	// NotPredator is true when the actor's species is not a quadruped predator
	// (requires legs AND fanged-or-clawed). Reachable via a direct player
	// command or a btree/combatcommands dispatch to a non-predator mob.
	// Unreachable via the AI path (CanUsePounce gates it).
	NotPredator bool
}

// ExecutePounce performs the core pounce resolution shared between player and
// mob callers. It handles:
//   - Special-move cooldown (using SpecialMoveCooldown from balance config)
//   - Target resolution via ResolveAggroTarget
//   - Compound predator gate: legs AND (fanged OR clawed)
//   - ExecuteSkillMove via combat package (UnarmedCombat skill, Dexterity
//     attack stat, Dexterity defense stat, BashDamagePercent, Strength damage
//     stat, BashKnockdownChance, KnockdownToSupine=true — backward knockdown)
//   - combat.RecordSpecialMove for analytics + RoundsWaiting = 1
//   - OnSkillUse(UnarmedCombat) on hit for progression
//
// Callers are responsible for all messaging and any combat-initiation logic.
func ExecutePounce(actor Actor) PounceResult {
	char := actor.GetCharacter()

	// Must be in combat (aggro set) before this function is called.
	if char.Aggro == nil {
		return PounceResult{NoTarget: true}
	}

	// Check special-move cooldown using the config value.
	cfg := configs.GetBalanceConfig()
	if !char.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		return PounceResult{OnCooldown: true}
	}

	// Resolve the aggro target.
	target := ResolveAggroTarget(char.Aggro)
	if !target.Found {
		return PounceResult{NoTarget: true}
	}

	// Compound predator gate (defense-in-depth): only legged fanged-or-clawed
	// creatures pounce. Unreachable via the AI path (CanUsePounce gates it) but
	// reachable via a direct player command or btree dispatch to a non-predator.
	if !combat.SpeciesIsQuadrupedPredator(char) {
		return PounceResult{NotPredator: true}
	}

	// Execute the skill move — uses bash's damage percent and knockdown chance;
	// KnockdownToSupine=true drives the target backward (face-up). No bleed:
	// pounce is a knockdown opener, not a DoT.
	result := combat.ExecuteSkillMove(combat.SkillMoveParams{
		Attacker:        char,
		Defender:        target.Char,
		AttackStat:      char.Stats.Dexterity.ValueAdj,
		AttackSkill:     char.GetSkillLevel(skills.UnarmedCombat),
		DefenseStat:     target.Char.Stats.Dexterity.ValueAdj,
		DefenseSkill:    target.Char.GetCombatSkillLevel(),
		DamagePercent:   float64(cfg.BashDamagePercent),
		KnockdownChance: int(cfg.BashKnockdownChance),
		KnockdownToSupine: true, // predator leaps and drives target backward
		SkillRank:       char.GetSkillLevel(skills.UnarmedCombat),
		DamageStat:      char.Stats.Strength.ValueAdj,
	})

	// Determine source/target types for analytics.
	sourceType := combat.User
	if !actor.IsPlayer() {
		sourceType = combat.Mob
	}
	targetType := combat.User
	if target.MobInstanceId > 0 {
		targetType = combat.Mob
	}

	// Record combat analytics.
	dmgRecorded := 0
	if result.Hit {
		dmgRecorded = result.Damage
	}
	combat.RecordSpecialMove(sourceType, targetType, "pounce", result.Hit, dmgRecorded, char, target.Char, util.GetRoundCount())

	// Consume the combat round.
	if char.Aggro != nil {
		char.Aggro.RoundsWaiting = 1
	}

	// Progression: unarmed-combat on hit.
	if result.Hit {
		actor.OnSkillUse(string(skills.UnarmedCombat))
	}

	return PounceResult{
		Target:     target,
		MoveResult: result,
		Executed:   true,
	}
}
