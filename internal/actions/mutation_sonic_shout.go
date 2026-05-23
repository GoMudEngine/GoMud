package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// TriggerSonicShout fires the sonic-shout mutation for any Actor (player or
// mob). AoE attack: rolls an opposed Wil+UnarmedCombat vs Wil+CombatSkill
// check against every mob in the room. On hit, each victim takes Willpower-
// scaled damage and has a chance of being knocked prone. The attacker always
// takes a self-deafen (ConditionBlinded, duration 3, magnitude 0.7) after the
// shout regardless of how many targets were hit.
//
// Gates: must have "sonic-shout" mutation + be in combat + pass the shared
// special-move cooldown + have at least 15 stamina. Stamina is only deducted
// on full success.
//
// NOTE: The damage formula (Willpower × 0.08) is inherited from the original
// player command and bypasses the unified combat.CalcRawDamage pipeline. This
// is a pre-existing design debt flagged here for a future balance pass. The
// "stun" effect is implemented as knockdown (TransitionToProne) rather than
// ConditionStunned — also inherited from the original implementation.
//
// Player actors receive descriptive messages; mob actors are silent (MobActor
// SendText is a no-op). Damage amounts are reported via
// combat.GetDamageDescription — no raw numbers per CLAUDE.md.
func TriggerSonicShout(actor Actor, opts MutationOpts) MutationResult {
	pre := mutationPreamble(actor, "sonic-shout", true /* combatRequired */, 15)
	if !pre.OK {
		return MutationResult{BlockReason: pre.BlockReason}
	}

	room := actor.GetRoom()
	if room == nil {
		return MutationResult{BlockReason: "no-room"}
	}

	char := actor.GetCharacter()

	attackerScore := float64(char.GetSkillLevel(skills.UnarmedCombat)) +
		float64(char.Stats.Willpower.ValueAdj)

	// Pre-existing: raw arithmetic, not combat.CalcRawDamage. See NOTE above.
	baseDamage := int(float64(char.Stats.Willpower.ValueAdj) * 0.08)
	if baseDamage < 1 {
		baseDamage = 1
	}

	if actor.IsPlayer() {
		actor.SendText(messaging.CategorySystem,
			`<ansi fg="magenta-bold">You unleash a devastating sonic shout that reverberates through the room!</ansi>`)
	}
	room.SendTextVisual(
		messaging.CategoryMutation,
		fmt.Sprintf(
			`<ansi fg="magenta-bold"><ansi fg="username">%s</ansi> unleashes a devastating sonic shout!</ansi>`,
			actor.GetName(),
		),
		actor.GetUserId(),
	)

	// Determine analytics source type once.
	sourceType := combat.User
	if !actor.IsPlayer() {
		sourceType = combat.Mob
	}

	// AoE: roll against every mob in the room except the attacker.
	affectedCount := 0
	for _, mobInstId := range room.GetMobs(rooms.FindAll) {
		if mobInstId == actor.GetMobInstanceId() {
			continue
		}
		victim := mobs.GetInstance(mobInstId)
		if victim == nil {
			continue
		}

		defenderScore := float64(victim.Character.Stats.Willpower.ValueAdj) +
			float64(victim.Character.GetCombatSkillLevel())
		attackSuccess, _, _, _ := dice.OpposedRollStat(attackerScore, defenderScore)

		shoutDmg := 0
		if attackSuccess {
			victim.Character.Health -= baseDamage
			if victim.Character.Health < 1 {
				victim.Character.Health = 0
			}
			shoutDmg = baseDamage

			// Knockdown chance: this implements the "stun" effect from the
			// original command. A low dice roll knocks the victim prone.
			knockdownRoll := dice.RollStat(50)
			if knockdownRoll.Value < 40 {
				_ = victim.Character.Position.TransitionToProne(
					position.ProneData{MinRecoveryRounds: 2},
					state.TransitionReason{Trigger: position.TriggerKnockdownFaceForward},
				)
			}

			if actor.IsPlayer() {
				actor.SendText(messaging.CategorySystem,
					fmt.Sprintf(
						`The shout staggers <ansi fg="mobname">%s</ansi>! (<ansi fg="damage">%s</ansi>)`,
						victim.Character.Name,
						combat.GetDamageDescription(baseDamage, victim.Character.HealthMax.Value),
					))
			}
			affectedCount++
		}

		combat.RecordSpecialMove(
			sourceType, combat.Mob, "sonic_shout",
			attackSuccess, shoutDmg,
			char, &victim.Character, util.GetRoundCount(),
		)
	}

	// Self-deafen: the echoes ring in the attacker's senses for 3 rounds.
	char.AddCondition(characters.ConditionBlinded, 3, 0.7, "sonic-shout self-deafen")
	if actor.IsPlayer() {
		actor.SendText(messaging.CategorySystem,
			`<ansi fg="yellow">The echoes leave your own senses ringing!</ansi>`)
	}

	actor.OnSkillUse(string(skills.UnarmedCombat))

	if char.Aggro != nil {
		char.Aggro.RoundsWaiting = 1
	}

	return MutationResult{Triggered: true, AffectedCount: affectedCount}
}
