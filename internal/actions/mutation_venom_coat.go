package actions

import (
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/skills"
)

// TriggerVenomCoat fires the venom-coat mutation for any Actor (player or mob).
// Self-buff: slicks the actor's weapons in venom (buff 103) for a burst of
// extra bite. Gates: owns "venom-coat" + shared special-move cooldown + 8
// stamina. Combat is NOT required — this is a prep move.
func TriggerVenomCoat(actor Actor, opts MutationOpts) MutationResult {
	pre := mutationPreamble(actor, "venom-coat", false /* combatRequired */, 8)
	if !pre.OK {
		return MutationResult{BlockReason: pre.BlockReason}
	}

	// Event-queue-safe buff application (per the Wave 2 gotcha — routes through
	// ApplyBuffs so start-text + GMCP conditions refresh).
	actor.AddBuff(103, "venom-coat")

	if actor.IsPlayer() {
		actor.SendText(messaging.CategoryMutation,
			`<ansi fg="green-bold">You flex, and venom weeps slick across your weapons.</ansi>`)
	}

	if room := actor.GetRoom(); room != nil {
		room.SendTextVisual(messaging.CategoryMutation,
			`<ansi fg="green"><ansi fg="username">`+actor.GetName()+
				`</ansi>'s weapons glisten with a sudden venomous sheen.</ansi>`,
			actor.GetUserId())
	}

	actor.OnSkillUse(string(skills.WeaponCombat))

	return MutationResult{Triggered: true, AffectedCount: 1}
}
