package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
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

// DrainAreaPlayerResult captures the per-player outcome of a single player
// target within an ExecuteDrainArea sweep.
type DrainAreaPlayerResult struct {
	// UserId identifies which player this result belongs to.
	UserId int

	// MoveResult is the outcome from ExecuteSkillMove against this player.
	MoveResult combat.SkillMoveResult

	// BleedDmg is the per-tick bleed magnitude applied to this player on a
	// hit (Strength/12, min 2). Zero on a miss.
	BleedDmg int
}

// DrainAreaResult holds the aggregate outcome of an ExecuteDrainArea sweep.
type DrainAreaResult struct {
	// Executed reports whether the area drain actually ran (false only when
	// there was no room, or no living players in it, to drain).
	Executed bool

	// NoTargets is true when the actor has no room, or the room has no
	// living players to drain.
	NoTargets bool

	// PlayerResults holds the per-player outcome for every player that was
	// swept by the drain (hit or miss).
	PlayerResults []DrainAreaPlayerResult

	// TotalDamage is the sum of damage dealt across all hit players.
	TotalDamage int

	// Healed is the amount of HP the actor actually recovered (after
	// clamping to HealthMax) from the aggregate lifesteal across every hit
	// player. Zero if no player was hit.
	Healed int
}

// ExecuteDrainArea performs an area version of ExecuteDrain: every living
// player in the actor's room is swept for drain damage (mirroring
// ExecuteDrain's per-target math exactly — ExecuteSkillMove with
// TripDamagePercent, a bleed condition on hit, DrainHealRatio lifesteal), and
// the actor is healed once by the aggregate of every hit's lifesteal
// fraction. Summing per-hit heal fractions and healing once at the end is
// mathematically identical to computing the aggregate fraction of the total
// damage up front (both reduce to DrainHealRatio * sum(damage_i)) — the
// per-hit accumulation is what lets this loop also report per-player damage
// for messaging.
//
// Unlike ExecuteDrain, this is NOT gated by SpeciesHasLifeDrain / aggro / the
// special-move cooldown — those are defense-in-depth for the single-target
// player/mob "drain" command entry point. ExecuteDrainArea is a boss-ability
// primitive meant to be invoked once, at the moment a fold-cast spell
// resolves (see resolveMobDrainArea in internal/hooks/spell_resolution.go),
// by a caster (e.g. a construct boss) that has no reason to carry the
// vampire-specific LifeDrain species flag. The caller (the spell-effect
// dispatch) owns cast-completion timing and telegraph/interrupt semantics;
// this function only owns the room-wide damage/heal math.
//
// Callers are responsible for all messaging.
func ExecuteDrainArea(actor Actor) DrainAreaResult {
	char := actor.GetCharacter()
	room := actor.GetRoom()
	if room == nil {
		return DrainAreaResult{NoTargets: true}
	}

	cfg := configs.GetBalanceConfig()
	playerIds := room.GetPlayers(rooms.FindAll)

	result := DrainAreaResult{}
	totalHeal := 0
	for _, uid := range playerIds {
		target := users.GetByUserId(uid)
		if target == nil || target.Character.Health < 1 {
			continue // skip vanished/downed players — already out of the fight
		}

		moveResult := combat.ExecuteSkillMove(combat.SkillMoveParams{
			Attacker:      char,
			Defender:      target.Character,
			AttackStat:    char.Stats.Strength.ValueAdj,
			AttackSkill:   char.GetSkillLevel(skills.UnarmedCombat),
			DefenseStat:   target.Character.GetEffectiveDexterity(),
			DefenseSkill:  target.Character.GetCombatSkillLevel(),
			DamagePercent: float64(cfg.TripDamagePercent),
			SkillRank:     char.GetSkillLevel(skills.UnarmedCombat),
			DamageStat:    char.Stats.Strength.ValueAdj,
		})

		pr := DrainAreaPlayerResult{UserId: uid, MoveResult: moveResult}

		if moveResult.Hit {
			mag := char.Stats.Strength.ValueAdj / 12
			if mag < 2 {
				mag = 2
			}
			target.Character.AddCondition(characters.ConditionBleeding, 4, float64(mag), "drain")
			pr.BleedDmg = mag

			healAmt := int(float64(moveResult.Damage) * float64(cfg.DrainHealRatio))
			if healAmt < 1 {
				healAmt = 1
			}
			totalHeal += healAmt
			result.TotalDamage += moveResult.Damage
		}

		result.PlayerResults = append(result.PlayerResults, pr)
	}

	if len(result.PlayerResults) == 0 {
		return DrainAreaResult{NoTargets: true}
	}

	result.Executed = true
	if totalHeal > 0 {
		result.Healed = char.Heal(totalHeal)
	}

	return result
}
