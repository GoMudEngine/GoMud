package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// TripVariant identifies which trip form was executed.
type TripVariant int

const (
	// TripStandard is a regular leg-sweep trip.
	TripStandard TripVariant = iota
	// TripTailsweep is used when the actor has the tail mutation — a tail-powered
	// knockdown with better damage and knockdown chance than the standard form.
	TripTailsweep
)

// TripResult holds the outcome of a trip attempt for the caller to use when
// formatting messages, firing events, and updating UI.
type TripResult struct {
	// Target is the resolved aggro target. Valid only when Executed is true.
	Target AggroTarget

	// MoveResult is the outcome from ExecuteSkillMove. Valid only when Executed is true.
	MoveResult combat.SkillMoveResult

	// Variant reports which trip form was used.
	Variant TripVariant

	// Executed reports whether the trip was actually performed. False when any
	// early-exit condition fired (OnCooldown, NoTarget).
	Executed bool

	// OnCooldown is true when the special-move cooldown blocked the trip.
	OnCooldown bool

	// Crafting is true when the actor is mid-craft and cannot trip.
	Crafting bool

	// NoTarget is true when there is no aggro target or the target is gone.
	NoTarget bool
}

// ExecuteTrip performs the core trip resolution shared between player and mob
// callers. It handles:
//   - Special-move cooldown (using SpecialMoveCooldown from balance config)
//   - Variant detection: tailsweep when actor has "tail" mutation
//   - Target resolution via ResolveAggroTarget
//   - ExecuteSkillMove via combat package with variant-specific params
//   - Tail equipment bonuses applied when tailsweep variant is active
//   - RecordAndWait (analytics + round consumption)
//
// Callers are responsible for all messaging, skill progression events, and
// any combat-initiation logic (e.g. SetAggro for player out-of-combat trip).
func ExecuteTrip(actor Actor) TripResult {
	char := actor.GetCharacter()

	// Don't interrupt any active activity (cast/craft/salvage) to trip someone.
	if char.IsActing() {
		return TripResult{Crafting: true}
	}

	// Must be in combat (aggro set) before this function is called.
	if char.Aggro == nil {
		return TripResult{NoTarget: true}
	}

	// Check special-move cooldown using the config value.
	cfg := configs.GetBalanceConfig()
	cooldownStr := fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)
	if !char.Cooldowns.Try("special-move", cooldownStr) {
		return TripResult{OnCooldown: true}
	}

	// Resolve the aggro target.
	target := ResolveAggroTarget(char.Aggro)
	if !target.Found {
		return TripResult{NoTarget: true}
	}

	// Determine trip variant and associated params.
	variant := TripStandard
	damagePercent := float64(cfg.TripDamagePercent)
	knockdownChance := int(cfg.TripKnockdownChance)

	// Tailsweep: actor has the tail mutation.
	if _, ok := char.Mutations["tail"]; ok {
		variant = TripTailsweep
		damagePercent = 0.40  // Better than regular trip (0.25)
		knockdownChance = 70  // Better than regular trip (60%)

		// Apply tail equipment bonuses if a tail item is equipped.
		if char.Equipment.Tail.ItemId > 0 {
			tailSpec := char.Equipment.Tail.GetSpec()
			if tailSpec.DamageMultiplier > 0 {
				damagePercent += tailSpec.DamageMultiplier / 100.0
			}
			knockdownChance += char.Equipment.Tail.StatMod("knockdown")
		}
	}

	// Execute the skill move.
	result := combat.ExecuteSkillMove(combat.SkillMoveParams{
		Attacker:        char,
		Defender:        target.Char,
		AttackStat:      char.Stats.Dexterity.ValueAdj,
		AttackSkill:     char.GetSkillLevel(skills.UnarmedCombat),
		DefenseStat:     target.Char.Stats.Dexterity.ValueAdj,
		DefenseSkill:    target.Char.GetCombatSkillLevel(),
		DamagePercent:   damagePercent,
		KnockdownChance: knockdownChance,
		SkillRank:       char.GetSkillLevel(skills.UnarmedCombat),
		DamageStat:      char.Stats.Strength.ValueAdj,
	})

	// Choose the move name for analytics based on variant.
	moveName := "trip"
	if variant == TripTailsweep {
		moveName = "tailsweep"
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

	// Record analytics and consume the combat round.
	dmgRecorded := 0
	if result.Hit {
		dmgRecorded = result.Damage
	}
	RecordAndWait(char, moveName, sourceType, target.Char, targetType, result.Hit, dmgRecorded, util.GetRoundCount())

	// Progression: unarmed-combat on hit (moved from user/mob wrappers)
	if result.Hit {
		actor.OnSkillUse(string(skills.UnarmedCombat))
	}

	return TripResult{
		Target:     target,
		MoveResult: result,
		Variant:    variant,
		Executed:   true,
	}
}
