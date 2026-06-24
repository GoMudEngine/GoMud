package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// DrainResult holds the outcome of a drain attempt for the caller to use when
// formatting messages, firing events, and updating UI.
type DrainResult struct {
	// Target is the resolved aggro target. Valid only when Executed is true.
	Target AggroTarget

	// MoveResult is the outcome from ExecuteSkillMove. Valid only when Executed
	// is true.
	MoveResult combat.SkillMoveResult

	// Executed reports whether the drain was actually performed. False when any
	// early-exit condition fired (OnCooldown, NoTarget, NotLifeDrainer).
	Executed bool

	// OnCooldown is true when the special-move cooldown blocked the drain.
	OnCooldown bool

	// NoTarget is true when there is no aggro target or the target is gone.
	NoTarget bool

	// NotLifeDrainer is true when the actor's species lacks the LifeDrain flag.
	// Reachable via a direct player command or a btree/combatcommands dispatch
	// to a non-lifedrain mob. Unreachable via the AI path (CanUseDrain gates it).
	NotLifeDrainer bool

	// Healed is the amount of HP the attacker actually recovered from the
	// lifesteal on a hit (after clamping to HealthMax). Zero on a miss.
	Healed int

	// BleedDmg is the per-tick bleed damage applied to the target on a hit
	// (Strength/12, min 2).
	BleedDmg int
}

// ExecuteDrain performs the core drain resolution shared between player and mob
// callers. It handles:
//   - Special-move cooldown (using SpecialMoveCooldown from balance config)
//   - Target resolution via ResolveAggroTarget
//   - Species identity gate: SpeciesHasLifeDrain
//   - ExecuteSkillMove via combat package (UnarmedCombat skill, Strength
//     attack stat, Dexterity defense stat, TripDamagePercent, Strength damage
//     stat, no knockdown)
//   - On hit: apply ConditionBleeding (duration 4, magnitude = Strength/12
//     min 2) sourced as "drain"; heal the attacker for DrainHealRatio * damage
//   - combat.RecordSpecialMove for analytics + RoundsWaiting = 1
//   - OnSkillUse(UnarmedCombat) on hit for progression
//
// Callers are responsible for all messaging and any combat-initiation logic.
func ExecuteDrain(actor Actor) DrainResult {
	char := actor.GetCharacter()

	// Must be in combat (aggro set) before this function is called.
	if char.Aggro == nil {
		return DrainResult{NoTarget: true}
	}

	// Check special-move cooldown using the config value.
	cfg := configs.GetBalanceConfig()
	if !char.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		return DrainResult{OnCooldown: true}
	}

	// Resolve the aggro target.
	target := ResolveAggroTarget(char.Aggro)
	if !target.Found {
		return DrainResult{NoTarget: true}
	}

	// Identity gate (defense-in-depth): only LifeDrain species can drain.
	// Unreachable via the AI path (CanUseDrain gates it) but reachable via a
	// direct player command or a btree/combatcommands dispatch to a non-lifedrain
	// mob.
	if !combat.SpeciesHasLifeDrain(char) {
		return DrainResult{NotLifeDrainer: true}
	}

	// Execute the skill move. Drain uses TripDamagePercent — the sapping strike
	// is lighter than a full melee blow; the lifesteal makes up the difference.
	// Strength drives both the attack and the damage, reflecting the predatory
	// grip. Dexterity governs the defender's evasion.
	result := combat.ExecuteSkillMove(combat.SkillMoveParams{
		Attacker:        char,
		Defender:        target.Char,
		AttackStat:      char.Stats.Strength.ValueAdj,
		AttackSkill:     char.GetSkillLevel(skills.UnarmedCombat),
		DefenseStat:     target.Char.GetEffectiveDexterity(),
		DefenseSkill:    target.Char.GetCombatSkillLevel(),
		DamagePercent:   float64(cfg.TripDamagePercent),
		KnockdownChance: 0, // No knockdown — the drain itself is the payoff
		SkillRank:       char.GetSkillLevel(skills.UnarmedCombat),
		DamageStat:      char.Stats.Strength.ValueAdj,
	})

	// On hit: bleed the victim and steal their life-force.
	bleedDmg := 0
	healed := 0
	if result.Hit {
		// Bleed condition (duration 4, magnitude = Strength/12, min 2) —
		// lighter than maul's bleed but still meaningful.
		mag := char.Stats.Strength.ValueAdj / 12
		if mag < 2 {
			mag = 2
		}
		target.Char.AddCondition(characters.ConditionBleeding, 4, float64(mag), "drain")
		bleedDmg = mag

		// Lifesteal: heal the attacker for a fraction of damage dealt.
		healAmt := int(float64(result.Damage) * float64(cfg.DrainHealRatio))
		if healAmt < 1 {
			healAmt = 1
		}
		healed = char.Heal(healAmt)
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
	combat.RecordSpecialMove(sourceType, targetType, "drain", result.Hit, dmgRecorded, char, target.Char, util.GetRoundCount())

	// Consume the combat round.
	if char.Aggro != nil {
		char.Aggro.RoundsWaiting = 1
	}

	// Progression: unarmed-combat on hit.
	if result.Hit {
		actor.OnSkillUse(string(skills.UnarmedCombat))
	}

	return DrainResult{
		Target:     target,
		MoveResult: result,
		Executed:   true,
		Healed:     healed,
		BleedDmg:   bleedDmg,
	}
}
