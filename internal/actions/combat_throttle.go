package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// ThrottleResult holds the outcome of a throttle attempt for the caller to use
// when formatting messages, firing events, and updating UI.
type ThrottleResult struct {
	// Target is the resolved aggro target. Valid only when Executed is true.
	Target AggroTarget

	// MoveResult is the outcome from ExecuteSkillMove. Valid only when Executed
	// is true.
	MoveResult combat.SkillMoveResult

	// Executed reports whether the throttle was actually performed. False when
	// any early-exit condition fired (OnCooldown, NoTarget).
	Executed bool

	// OnCooldown is true when the special-move cooldown blocked the throttle.
	OnCooldown bool

	// NoTarget is true when there is no aggro target or the target is gone.
	NoTarget bool

	// NotFanged is true when the actor's species lacks a fanged natural attack.
	// Reachable via a direct player command or a btree/combatcommands dispatch
	// to a non-fanged mob. Unreachable via the AI path (CanUseThrottle gates it).
	NotFanged bool

	// InterruptedCast is true when the hit triggered the cast-cancel path and
	// the target's spellcast was successfully interrupted.
	InterruptedCast bool

	// BleedDmg is the per-tick bleed magnitude applied on a hit
	// (Strength/10, min 2).
	BleedDmg int
}

// ExecuteThrottle performs the core throttle resolution shared between player
// and mob callers. It handles:
//   - Special-move cooldown (using SpecialMoveCooldown from balance config)
//   - Target resolution via ResolveAggroTarget
//   - ExecuteSkillMove via combat package (UnarmedCombat skill, Dexterity
//     attack stat, Dexterity defense stat, KickDamagePercent, Strength damage
//     stat, no knockdown)
//   - On hit: apply ConditionBleeding (duration 3, magnitude = Strength/10
//     min 2) sourced as "throttle"
//   - On hit: apply Throttled DoT buff (id 89) for stamina drain
//   - On hit: chance (ThrottleInterruptChance) to interrupt an in-progress cast
//     via InterruptTargetCast (engine's standard cast-cancel path)
//   - combat.RecordSpecialMove for analytics + RoundsWaiting = 1
//   - OnSkillUse(UnarmedCombat) on hit for progression
//
// Callers are responsible for all messaging and any combat-initiation logic.
func ExecuteThrottle(actor Actor) ThrottleResult {
	char := actor.GetCharacter()

	// Must be in combat (aggro set) before this function is called.
	if char.Aggro == nil {
		return ThrottleResult{NoTarget: true}
	}

	// Check special-move cooldown using the config value.
	cfg := configs.GetBalanceConfig()
	if !char.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		return ThrottleResult{OnCooldown: true}
	}

	// Resolve the aggro target.
	target := ResolveAggroTarget(char.Aggro)
	if !target.Found {
		return ThrottleResult{NoTarget: true}
	}

	// Anatomy/identity gate (defense-in-depth): only fanged creatures throttle.
	// Unreachable via the AI path (CanUseThrottle gates it) but reachable via a
	// direct player command or a btree/combatcommands dispatch to a non-fanged mob.
	if !combat.SpeciesIsFanged(char) {
		return ThrottleResult{NotFanged: true}
	}

	// Execute the skill move (reuse kick's config for damage percent; no
	// knockdown — the choke deals stamina drain and cast interrupt instead).
	result := combat.ExecuteSkillMove(combat.SkillMoveParams{
		Attacker:        char,
		Defender:        target.Char,
		AttackStat:      char.Stats.Dexterity.ValueAdj,
		AttackSkill:     char.GetSkillLevel(skills.UnarmedCombat),
		DefenseStat:     target.Char.Stats.Dexterity.ValueAdj,
		DefenseSkill:    target.Char.GetCombatSkillLevel(),
		DamagePercent:   float64(cfg.KickDamagePercent),
		KnockdownChance: 0, // No knockdown — choke + stamina drain instead
		SkillRank:       char.GetSkillLevel(skills.UnarmedCombat),
		DamageStat:      char.Stats.Strength.ValueAdj,
	})

	bleedDmg := 0
	interrupted := false

	if result.Hit {
		// Health-over-time: apply a light bleed (duration 3, magnitude = Strength/10,
		// min 2) — weaker than maul; the choke's primary DoT is stamina drain.
		mag := char.Stats.Strength.ValueAdj / 10
		if mag < 2 {
			mag = 2
		}
		target.Char.AddCondition(characters.ConditionBleeding, 3, float64(mag), "throttle")
		bleedDmg = mag

		// Stamina-over-time: apply the Throttled DoT buff (id 89).
		_ = target.Char.AddBuff(89, false)

		// Cast interrupt: fairly high flat chance via ThrottleInterruptChance config.
		// Uses the engine's standard cast-cancel path (50% conviction refund +
		// TriggerCastCancel transition) — no new buff/status required.
		if target.Char.IsCasting() && util.Rand(100) < int(float64(cfg.ThrottleInterruptChance)*100) {
			var attackerRef state.ActorRef
			if actor.IsPlayer() {
				attackerRef = state.ActorRef{UserId: actor.GetUserId()}
			} else {
				attackerRef = state.ActorRef{MobInstanceId: actor.GetMobInstanceId()}
			}
			interrupted = InterruptTargetCast(target.Char, attackerRef)
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

	// Record combat analytics.
	dmgRecorded := 0
	if result.Hit {
		dmgRecorded = result.Damage
	}
	combat.RecordSpecialMove(sourceType, targetType, "throttle", result.Hit, dmgRecorded, char, target.Char, util.GetRoundCount())

	// Consume the combat round.
	if char.Aggro != nil {
		char.Aggro.RoundsWaiting = 1
	}

	// Progression: unarmed-combat on hit.
	if result.Hit {
		actor.OnSkillUse(string(skills.UnarmedCombat))
	}

	return ThrottleResult{
		Target:          target,
		MoveResult:      result,
		Executed:        true,
		BleedDmg:        bleedDmg,
		InterruptedCast: interrupted,
	}
}
