package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/skills"
)

// TriggerHealingGel fires the healing-gel mutation for any Actor (player or
// mob). Self-only: heals floor(HealthMax * 0.25). Stamina cost 8. Does NOT
// require combat — this is the only mutation in the suite with
// combatRequired=false. Also applies a short ConditionRecoveryPenalty (the
// gel weighs down movement) and fires a Skullduggery skill-use event.
//
// Bug fix (vs. original player command): the original code computed
// int(Vitality * 0.15), which is a stat-derived flat amount that violates the
// no-flat-heal rule (see CLAUDE.md). Corrected to floor(HealthMax * 0.25),
// which scales with the actor's actual health pool.
//
// Gates: must have "healing-gel" mutation + pass the shared special-move
// cooldown + have at least 8 stamina. Combat is NOT required.
func TriggerHealingGel(actor Actor, opts MutationOpts) MutationResult {
	pre := mutationPreamble(actor, "healing-gel", false /* combatRequired=false */, 8)
	if !pre.OK {
		return MutationResult{BlockReason: pre.BlockReason}
	}

	char := actor.GetCharacter()

	// Heal as 25% of HealthMax per the no-flat-heal rule.
	healAmount := char.HealthMax.Value * 25 / 100
	if healAmount < 1 {
		healAmount = 1
	}
	healed := char.Heal(healAmount)

	if actor.IsPlayer() {
		actor.SendText(messaging.CategorySystem,
			fmt.Sprintf(
				`<ansi fg="green-bold">A thick, regenerative gel oozes from your pores,`+
					` knitting wounds as it spreads. (%s)</ansi>`,
				combat.GetHealDescription(healed, char.HealthMax.Value),
			),
		)
	}

	room := actor.GetRoom()
	if room != nil {
		room.SendTextVisual(
			messaging.CategoryMutation,
			fmt.Sprintf(
				`<ansi fg="green">A glistening gel coats <ansi fg="username">%s</ansi>'s`+
					` skin, sealing wounds before your eyes.</ansi>`,
				actor.GetName(),
			),
			actor.GetUserId(),
		)
	}

	// The gel weighs the actor down for 2 rounds.
	char.AddCondition(characters.ConditionRecoveryPenalty, 2, 1.0, "healing-gel movement penalty")
	if actor.IsPlayer() {
		actor.SendText(messaging.CategorySystem,
			`<ansi fg="yellow">The gel weighs you down, slowing your movements.</ansi>`)
	}

	actor.OnSkillUse(string(skills.Skullduggery))

	return MutationResult{Triggered: true, AffectedCount: 1}
}
