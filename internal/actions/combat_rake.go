package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// RakeResult holds the outcome of a rake attempt for the caller to use when
// formatting messages, firing events, and updating UI.
type RakeResult struct {
	// Target is the resolved aggro target. Valid only when Executed is true.
	Target AggroTarget

	// MoveResult is the outcome from ExecuteSkillMove. Valid only when Executed
	// is true.
	MoveResult combat.SkillMoveResult

	// Executed reports whether the rake was actually performed. False when any
	// early-exit condition fired (OnCooldown, NoTarget).
	Executed bool

	// OnCooldown is true when the special-move cooldown blocked the rake.
	OnCooldown bool

	// NoTarget is true when there is no aggro target or the target is gone.
	NoTarget bool

	// NotClawed is true when the actor's species lacks a clawed natural attack.
	// Reachable via a direct player command or a btree/combatcommands dispatch
	// to a non-clawed mob. Unreachable via the AI path (CanUseRake gates it).
	NotClawed bool

	// BleedDmg is the per-tick bleed damage applied on a hit
	// (Strength/12, min 2).
	BleedDmg int
}

// ExecuteRake performs the core rake resolution shared between player and mob
// callers. It handles:
//   - Special-move cooldown (using SpecialMoveCooldown from balance config)
//   - Target resolution via ResolveAggroTarget
//   - ExecuteSkillMove via combat package (UnarmedCombat skill, Dexterity
//     attack stat, Dexterity defense stat, TripDamagePercent, Strength damage
//     stat, no knockdown)
//   - On hit: apply ConditionBleeding (duration 4, magnitude = Strength/12
//     min 2) sourced as "rake"
//   - combat.RecordSpecialMove for analytics + RoundsWaiting = 1
//   - OnSkillUse(UnarmedCombat) on hit for progression
//
// Callers are responsible for all messaging and any combat-initiation logic.
func ExecuteRake(actor Actor) RakeResult {
	char := actor.GetCharacter()

	// Must be in combat (aggro set) before this function is called.
	if char.Aggro == nil {
		return RakeResult{NoTarget: true}
	}

	// Check special-move cooldown using the config value.
	cfg := configs.GetBalanceConfig()
	if !char.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		return RakeResult{OnCooldown: true}
	}

	// Resolve the aggro target.
	target := ResolveAggroTarget(char.Aggro)
	if !target.Found {
		return RakeResult{NoTarget: true}
	}

	// Anatomy/identity gate (defense-in-depth): only clawed creatures rake.
	// Unreachable via the AI path (CanUseRake gates it) but reachable via a
	// direct player command or a btree/combatcommands dispatch to a non-clawed mob.
	if !combat.SpeciesIsClawed(char) {
		return RakeResult{NotClawed: true}
	}

	// Execute the skill move (reuse trip's config for damage percent, no
	// knockdown — the clawed raking strike deals moderate damage and bleeds
	// rather than felling).
	result := combat.ExecuteSkillMove(combat.SkillMoveParams{
		Attacker:        char,
		Defender:        target.Char,
		AttackStat:      char.Stats.Dexterity.ValueAdj,
		AttackSkill:     char.GetSkillLevel(skills.UnarmedCombat),
		DefenseStat:     target.Char.Stats.Dexterity.ValueAdj,
		DefenseSkill:    target.Char.GetCombatSkillLevel(),
		DamagePercent:   float64(cfg.TripDamagePercent),
		KnockdownChance: 0, // No knockdown — bleed instead
		SkillRank:       char.GetSkillLevel(skills.UnarmedCombat),
		DamageStat:      char.Stats.Strength.ValueAdj,
	})

	// On hit: apply bleed condition (duration 4, magnitude = Strength/12,
	// min 2).
	bleedDmg := 0
	if result.Hit {
		bleedDmg = char.Stats.Strength.ValueAdj / 12
		if bleedDmg < 2 {
			bleedDmg = 2
		}
		target.Char.AddCondition(characters.ConditionBleeding, 4, float64(bleedDmg), "rake")
	}

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
	combat.RecordSpecialMove(sourceType, targetType, "rake", result.Hit, dmgRecorded, char, target.Char, util.GetRoundCount())

	// Consume the combat round.
	if char.Aggro != nil {
		char.Aggro.RoundsWaiting = 1
	}

	// Progression: unarmed-combat on hit.
	if result.Hit {
		actor.OnSkillUse(string(skills.UnarmedCombat))
	}

	return RakeResult{
		Target:     target,
		MoveResult: result,
		Executed:   true,
		BleedDmg:   bleedDmg,
	}
}
