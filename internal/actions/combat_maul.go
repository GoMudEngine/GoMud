package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// MaulResult holds the outcome of a maul attempt for the caller to use when
// formatting messages, firing events, and updating UI.
type MaulResult struct {
	// Target is the resolved aggro target. Valid only when Executed is true.
	Target AggroTarget

	// MoveResult is the outcome from ExecuteSkillMove. Valid only when Executed
	// is true.
	MoveResult combat.SkillMoveResult

	// Executed reports whether the maul was actually performed. False when any
	// early-exit condition fired (OnCooldown, NoTarget).
	Executed bool

	// OnCooldown is true when the special-move cooldown blocked the maul.
	OnCooldown bool

	// NoTarget is true when there is no aggro target or the target is gone.
	NoTarget bool

	// NotFanged is true when the actor's species lacks a fanged natural attack.
	// Reachable via a direct player command or a btree/combatcommands dispatch
	// to a non-fanged mob. Unreachable via the AI path (CanUseMaul gates it).
	NotFanged bool

	// BleedDmg is the per-tick bleed damage applied on a hit
	// (Strength/8, min 3).
	BleedDmg int
}

// ExecuteMaul performs the core maul resolution shared between player and mob
// callers. It handles:
//   - Special-move cooldown (using SpecialMoveCooldown from balance config)
//   - Target resolution via ResolveAggroTarget
//   - ExecuteSkillMove via combat package (UnarmedCombat skill, Dexterity
//     attack stat, Dexterity defense stat, KickDamagePercent, Strength damage
//     stat, no knockdown)
//   - On hit: apply ConditionBleeding (duration 5, magnitude = Strength/8
//     min 3) sourced as "maul"
//   - combat.RecordSpecialMove for analytics + RoundsWaiting = 1
//   - OnSkillUse(UnarmedCombat) on hit for progression
//
// Callers are responsible for all messaging and any combat-initiation logic.
func ExecuteMaul(actor Actor) MaulResult {
	char := actor.GetCharacter()

	// Must be in combat (aggro set) before this function is called.
	if char.Aggro == nil {
		return MaulResult{NoTarget: true}
	}

	// Check special-move cooldown using the config value.
	cfg := configs.GetBalanceConfig()
	if !char.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		return MaulResult{OnCooldown: true}
	}

	// Resolve the aggro target.
	target := ResolveAggroTarget(char.Aggro)
	if !target.Found {
		return MaulResult{NoTarget: true}
	}

	// Anatomy/identity gate (defense-in-depth): only handless fanged creatures
	// maul. Unreachable via the AI path (CanUseMaul gates it) but reachable
	// via a direct player command or a btree/combatcommands dispatch to a
	// non-fanged or tool-using mob.
	if char.HasBodyPart("hands") || !combat.SpeciesIsFanged(char) {
		return MaulResult{NotFanged: true}
	}

	// Execute the skill move (reuse kick's config for damage percent — maul is
	// a heavier strike than rake — no knockdown; the savage tearing deals higher
	// damage and bleeds more severely).
	result := combat.ExecuteSkillMove(combat.SkillMoveParams{
		Attacker:        char,
		Defender:        target.Char,
		AttackStat:      char.Stats.Dexterity.ValueAdj,
		AttackSkill:     char.GetSkillLevel(skills.UnarmedCombat),
		DefenseStat:     target.Char.Stats.Dexterity.ValueAdj,
		DefenseSkill:    target.Char.GetCombatSkillLevel(),
		DamagePercent:   float64(cfg.KickDamagePercent),
		KnockdownChance: 0, // No knockdown — savage bleed instead
		SkillRank:       char.GetSkillLevel(skills.UnarmedCombat),
		DamageStat:      char.Stats.Strength.ValueAdj,
	})

	// On hit: apply bleed condition (duration 5, magnitude = Strength/8,
	// min 3) — stronger bleed than rake.
	bleedDmg := 0
	if result.Hit {
		mag := char.Stats.Strength.ValueAdj / 8
		if mag < 3 {
			mag = 3
		}
		target.Char.AddCondition(characters.ConditionBleeding, 5, float64(mag), "maul")
		bleedDmg = mag
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
	combat.RecordSpecialMove(sourceType, targetType, "maul", result.Hit, dmgRecorded, char, target.Char, util.GetRoundCount())

	// Consume the combat round.
	if char.Aggro != nil {
		char.Aggro.RoundsWaiting = 1
	}

	// Progression: unarmed-combat on hit.
	if result.Hit {
		actor.OnSkillUse(string(skills.UnarmedCombat))
	}

	return MaulResult{
		Target:     target,
		MoveResult: result,
		Executed:   true,
		BleedDmg:   bleedDmg,
	}
}
