package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// BiteResult holds the outcome of a bite attempt for the caller to use when
// formatting messages, firing events, and updating UI.
type BiteResult struct {
	// Target is the resolved aggro target. Valid only when Executed is true.
	Target AggroTarget

	// MoveResult is the outcome from ExecuteSkillMove. Valid only when Executed is true.
	MoveResult combat.SkillMoveResult

	// Executed reports whether the bite was actually performed. False when any
	// early-exit condition fired (OnCooldown, NoTarget).
	Executed bool

	// OnCooldown is true when the special-move cooldown blocked the bite.
	OnCooldown bool

	// NoTarget is true when there is no aggro target or the target is gone.
	NoTarget bool

	// DrainAmount is the health drained from the target and added to the actor.
	// Zero on a miss or zero-damage hit.
	DrainAmount int
}

// ExecuteBite performs the core bite resolution shared between player and mob
// callers. It handles:
//   - Special-move cooldown (using SpecialMoveCooldown from balance config)
//   - Target resolution via ResolveAggroTarget
//   - ExecuteSkillMove via combat package (UnarmedCombat, Strength attack/damage,
//     Dexterity defense, 0.60 damage percent, no knockdown)
//   - Life drain: on hit with damage > 0, heals the actor for 50% of damage dealt
//     (capped at HealthMax)
//   - RecordAndWait (analytics + round consumption)
//   - OnSkillUse(UnarmedCombat) on hit for progression
//
// Callers are responsible for all messaging and any combat-initiation logic
// (e.g. SetAggro for player out-of-combat bite).
func ExecuteBite(actor Actor) BiteResult {
	char := actor.GetCharacter()

	// Must be in combat (aggro set) before this function is called.
	if char.Aggro == nil {
		return BiteResult{NoTarget: true}
	}

	// Check special-move cooldown using the config value.
	cfg := configs.GetBalanceConfig()
	cooldownStr := fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)
	if !char.Cooldowns.Try("special-move", cooldownStr) {
		return BiteResult{OnCooldown: true}
	}

	// Resolve the aggro target.
	target := ResolveAggroTarget(char.Aggro)
	if !target.Found {
		return BiteResult{NoTarget: true}
	}

	// Execute the skill move.
	result := combat.ExecuteSkillMove(combat.SkillMoveParams{
		Attacker:        char,
		Defender:        target.Char,
		AttackStat:      char.Stats.Strength.ValueAdj,
		AttackSkill:     char.GetSkillLevel(skills.UnarmedCombat),
		DefenseStat:     target.Char.Stats.Dexterity.ValueAdj,
		DefenseSkill:    target.Char.GetCombatSkillLevel(),
		DamagePercent:   0.60,
		KnockdownChance: 0,
		SkillRank:       char.GetSkillLevel(skills.UnarmedCombat),
		DamageStat:      char.Stats.Strength.ValueAdj,
	})

	// Life drain: heal actor for 50% of damage dealt, capped at HealthMax.
	drainAmount := 0
	if result.Hit && result.Damage > 0 {
		drainAmount = int(float64(result.Damage) * 0.50)
		char.Health += drainAmount
		if char.Health > char.HealthMax.Value {
			char.Health = char.HealthMax.Value
		}
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
	RecordAndWait(char, "bite", sourceType, target.Char, targetType, result.Hit, dmgRecorded, util.GetRoundCount())

	// Progression: unarmed-combat on hit.
	if result.Hit {
		actor.OnSkillUse(string(skills.UnarmedCombat))
	}

	return BiteResult{
		Target:      target,
		MoveResult:  result,
		Executed:    true,
		DrainAmount: drainAmount,
	}
}
