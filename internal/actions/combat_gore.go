package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// GoreResult holds the outcome of a gore attempt for the caller to use
// when formatting messages, firing events, and updating UI.
type GoreResult struct {
	// Target is the resolved aggro target. Valid only when Executed is true.
	Target AggroTarget

	// MoveResult is the outcome from ExecuteSkillMove. Valid only when Executed
	// is true.
	MoveResult combat.SkillMoveResult

	// Executed reports whether the gore was actually performed. False when
	// any early-exit condition fired (OnCooldown, NoTarget, NotHorned).
	Executed bool

	// OnCooldown is true when the special-move cooldown blocked the gore.
	OnCooldown bool

	// NoTarget is true when there is no aggro target or the target is gone.
	NoTarget bool

	// NotHorned is true when the actor's species is not horned (requires
	// a gore natural attack). Reachable via a direct player command or a
	// btree/combatcommands dispatch to a non-horned mob. Unreachable via
	// the AI path (CanUseGore gates it).
	NotHorned bool
}

// ExecuteGore performs the core gore resolution shared between player and
// mob callers. It handles:
//   - Special-move cooldown (using SpecialMoveCooldown from balance config)
//   - Target resolution via ResolveAggroTarget
//   - Horned-species gate: SpeciesIsHorned (gore natural attack)
//   - ExecuteSkillMove via combat package (UnarmedCombat skill, Strength
//     attack stat, Dexterity defense stat, KickDamagePercent,
//     BashKnockdownChance, KnockdownToSupine=true — forward charge knockback)
//   - combat.RecordSpecialMove for analytics + RoundsWaiting = 1
//   - OnSkillUse(UnarmedCombat) on hit for progression
//
// Callers are responsible for all messaging and any combat-initiation logic.
func ExecuteGore(actor Actor) GoreResult {
	char := actor.GetCharacter()

	// Must be in combat (aggro set) before this function is called.
	if char.Aggro == nil {
		return GoreResult{NoTarget: true}
	}

	// Check special-move cooldown using the config value.
	cfg := configs.GetBalanceConfig()
	if !char.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		return GoreResult{OnCooldown: true}
	}

	// Resolve the aggro target.
	target := ResolveAggroTarget(char.Aggro)
	if !target.Found {
		return GoreResult{NoTarget: true}
	}

	// Horned gate (defense-in-depth): only handless horned creatures gore.
	// Unreachable via the AI path (CanUseGore gates it) but reachable via a
	// direct player command or btree dispatch to a non-horned or tool-using mob.
	if char.HasBodyPart("hands") || !combat.SpeciesIsHorned(char) {
		return GoreResult{NotHorned: true}
	}

	// Execute the skill move — uses kick's damage percent for the charge
	// impact and bash's knockdown chance; KnockdownToSupine=true drives the
	// target forward (face-up). No bleed: gore is a knockdown opener.
	result := combat.ExecuteSkillMove(combat.SkillMoveParams{
		Attacker:          char,
		Defender:          target.Char,
		AttackStat:        char.Stats.Strength.ValueAdj,
		AttackSkill:       char.GetSkillLevel(skills.UnarmedCombat),
		DefenseStat:       target.Char.Stats.Dexterity.ValueAdj,
		DefenseSkill:      target.Char.GetCombatSkillLevel(),
		DamagePercent:     float64(cfg.KickDamagePercent),
		KnockdownChance:   int(cfg.BashKnockdownChance),
		KnockdownToSupine: true, // horned charge drives target forward (face-up)
		SkillRank:         char.GetSkillLevel(skills.UnarmedCombat),
		DamageStat:        char.Stats.Strength.ValueAdj,
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
	combat.RecordSpecialMove(sourceType, targetType, "gore", result.Hit, dmgRecorded, char, target.Char, util.GetRoundCount())

	// Consume the combat round.
	if char.Aggro != nil {
		char.Aggro.RoundsWaiting = 1
	}

	// Progression: unarmed-combat on hit.
	if result.Hit {
		actor.OnSkillUse(string(skills.UnarmedCombat))
	}

	return GoreResult{
		Target:     target,
		MoveResult: result,
		Executed:   true,
	}
}
