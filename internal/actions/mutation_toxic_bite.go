package actions

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// TriggerToxicBite fires the toxic-bite mutation for any Actor (player or
// mob). Single-target: on a successful opposed Dex+UnarmedCombat vs
// Dex+CombatSkill roll, deals Strength-scaled bite damage and applies
// ConditionPoisoned (duration 9, Vitality-scaled magnitude) to the target.
// On a miss the attacker bites their own tongue and takes half the bite
// damage as self-damage.
//
// Gates: must have "toxic-bite" mutation + be in combat + pass the shared
// special-move cooldown + have at least 12 stamina. Stamina is deducted by
// the preamble on full success; no-target is therefore checked after the
// preamble (matching the original player command ordering where target
// resolution follows the stamina deduction).
//
// NOTE: Bite damage is routed through combat.CalcRawDamage(Str,
// UnarmedCombat rank, 1.0, ChannelPhysical) → ApplyMitigation against the
// victim's physical_mitigation → dice.RollStat for variance. Poison DoT
// magnitude (Vitality × 0.04) still uses raw arithmetic — pipeline routing
// for over-time damage is a separate followup
// (project_poison_dot_magnitude_pipeline).
//
// Player actors receive descriptive messages; mob actors are silent (MobActor
// SendText is a no-op). Damage amounts are reported via
// combat.GetDamageDescription — no raw numbers per CLAUDE.md.
func TriggerToxicBite(actor Actor, opts MutationOpts) MutationResult {
	pre := mutationPreamble(actor, "toxic-bite", true /* combatRequired */, 12)
	if !pre.OK {
		return MutationResult{BlockReason: pre.BlockReason}
	}

	// Target resolution follows the preamble (matching the original player
	// command ordering). Stamina has already been deducted; no refund on
	// no-target (mirrors original behavior).
	if opts.TargetActor == nil || opts.TargetActor.GetCharacter() == nil {
		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem, "You have no target!")
		}
		return MutationResult{BlockReason: "no-target"}
	}

	char := actor.GetCharacter()
	target := opts.TargetActor.GetCharacter()
	targetName := opts.TargetActor.GetName()
	targetUserId := opts.TargetActor.GetUserId()

	attackerScore := float64(char.GetSkillLevel(skills.UnarmedCombat)) +
		float64(char.GetEffectiveDexterity())
	defenderScore := float64(target.GetEffectiveDexterity()) +
		float64(target.GetCombatSkillLevel())

	attackSuccess, _, _, _ := dice.OpposedRollStat(attackerScore, defenderScore)

	// Bite damage: routed through the unified Physical pipeline.
	// Strength is the input stat; UnarmedCombat skill rank scales via the
	// sqrt curve; no item multiplier for mutations.
	rawBiteDmg := combat.CalcRawDamage(
		char.Stats.Strength.ValueAdj,
		char.GetSkillLevel(skills.UnarmedCombat),
		1.0,                    // no item multiplier for mutations
		combat.ChannelPhysical, // bite is physical impact
	)
	mitPct := opts.TargetActor.GetCharacter().GetPhysicalMitigation()
	mitCap := combat.MitigationCap(combat.ChannelPhysical)
	mitigatedBite := combat.ApplyMitigation(rawBiteDmg, mitPct, mitCap)
	biteRoll := dice.RollStat(mitigatedBite)
	biteDamage := int(math.Round(biteRoll.Value))
	if biteDamage < 1 {
		biteDamage = 1
	}

	room := actor.GetRoom()

	// Determine analytics source/target types once.
	sourceType := combat.User
	if !actor.IsPlayer() {
		sourceType = combat.Mob
	}
	targetType := combat.Mob
	if targetUserId > 0 {
		targetType = combat.User
	}

	biteDmgRecorded := 0

	if attackSuccess {
		target.Health -= biteDamage
		if target.Health < 1 {
			target.Health = 0
		}
		biteDmgRecorded = biteDamage

		// Apply poison DoT: magnitude is damage per AutoHeal tick.
		// Pre-existing: raw arithmetic formula, see NOTE above.
		poisonDmg := float64(char.Stats.Vitality.ValueAdj) * 0.04
		if poisonDmg < 1 {
			poisonDmg = 1
		}
		target.AddCondition(characters.ConditionPoisoned, 9, poisonDmg, "toxic-bite")

		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem,
				fmt.Sprintf(
					`<ansi fg="green-bold">You sink your toxic fangs into <ansi fg="mobname">%s</ansi>! Venom courses into the wound. (<ansi fg="damage">%s</ansi>)</ansi>`,
					targetName,
					combat.GetDamageDescription(biteDamage, opts.TargetActor.GetCharacter().HealthMax.Value),
				),
			)
		}

		// Notify the victim if they are a player.
		if targetUserId > 0 {
			if tUser := users.GetByUserId(targetUserId); tUser != nil {
				tUser.SendText(messaging.CategorySystem,
					fmt.Sprintf(
						`<ansi fg="red"><ansi fg="username">%s</ansi> bites you with venomous fangs! You feel poison spreading! (<ansi fg="damage">%s</ansi>)</ansi>`,
						actor.GetName(),
						combat.GetDamageDescription(biteDamage, tUser.Character.HealthMax.Value),
					),
				)
			}
		}

		if room != nil {
			room.SendTextVisual(
				messaging.CategoryMutation,
				fmt.Sprintf(
					`<ansi fg="username">%s</ansi> bites <ansi fg="mobname">%s</ansi> with venomous fangs!`,
					actor.GetName(), targetName,
				),
				actor.GetUserId(), targetUserId,
			)
		}
	} else {
		// Miss path: attacker bites their own tongue for half the bite damage.
		selfDamage := biteDamage / 2
		if selfDamage < 1 {
			selfDamage = 1
		}
		char.Health -= selfDamage
		if char.Health < 1 {
			char.Health = 0
		}

		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem,
				fmt.Sprintf(
					`<ansi fg="red">Your toxic bite misses <ansi fg="mobname">%s</ansi> and you bite your own tongue! (<ansi fg="damage">%s</ansi>)</ansi>`,
					targetName,
					combat.GetDamageDescription(selfDamage, char.HealthMax.Value),
				),
			)
		}

		// Notify the victim if they are a player.
		if targetUserId > 0 {
			if tUser := users.GetByUserId(targetUserId); tUser != nil {
				tUser.SendText(messaging.CategorySystem,
					fmt.Sprintf(
						`<ansi fg="username">%s</ansi> lunges to bite you, but misses!`,
						actor.GetName(),
					),
				)
			}
		}

		if room != nil {
			room.SendTextVisual(
				messaging.CategoryMutation,
				fmt.Sprintf(
					`<ansi fg="username">%s</ansi> lunges to bite <ansi fg="mobname">%s</ansi>, but misses and bites their own tongue!`,
					actor.GetName(), targetName,
				),
				actor.GetUserId(), targetUserId,
			)
		}
	}

	combat.RecordSpecialMove(
		sourceType, targetType, "toxic_bite",
		attackSuccess, biteDmgRecorded,
		char, target, util.GetRoundCount(),
	)

	actor.OnSkillUse(string(skills.UnarmedCombat))

	if char.Aggro != nil {
		char.Aggro.RoundsWaiting = 1
	}

	affectedCount := 0
	if attackSuccess {
		affectedCount = 1
	}
	return MutationResult{Triggered: true, AffectedCount: affectedCount}
}
