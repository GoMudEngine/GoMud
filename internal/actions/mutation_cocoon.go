package actions

import (
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// TriggerCocoon fires the cocoon mutation for any Actor. Self-buff: encases the
// actor in a near-invulnerable chrysalis shell (buff 104) AND drops aggro —
// every room mob currently fixed on the actor loses its target ("vanish from
// threat"). No attack-lock: the cost is the spent move + the shared cooldown.
// Gates: owns "cocoon" + shared special-move cooldown + 10 stamina. Combat not
// required.
func TriggerCocoon(actor Actor, opts MutationOpts) MutationResult {
	pre := mutationPreamble(actor, "cocoon", false /* combatRequired */, 10)
	if !pre.OK {
		return MutationResult{BlockReason: pre.BlockReason}
	}

	actor.AddBuff(104, "cocoon")

	// Drop aggro: mobs in the room fixed on this actor lose their target.
	// Guarded to player actors — a mob actor has UserId 0, which would spuriously
	// match every mob-vs-mob fight (their target UserId is also 0).
	dropped := 0
	actorUserId := actor.GetUserId()
	if room := actor.GetRoom(); room != nil && actorUserId > 0 {
		for _, mobId := range room.GetMobs() {
			m := mobs.GetInstance(mobId)
			if m == nil {
				continue
			}
			if m.Character.IsInCombat() && m.Character.CurrentCombatTarget().UserId == actorUserId {
				m.Character.EndAggro()
				dropped++
			}
		}
	}

	if actor.IsPlayer() {
		actor.SendText(messaging.CategoryMutation,
			`<ansi fg="cyan-bold">You fold inward; a hard shell snaps shut and the fight loses sight of you.</ansi>`)
	}

	if room := actor.GetRoom(); room != nil {
		room.SendTextVisual(messaging.CategoryMutation,
			`<ansi fg="cyan"><ansi fg="username">`+actor.GetName()+
				`</ansi> folds into a hard, translucent shell.</ansi>`,
			actor.GetUserId())
	}

	return MutationResult{Triggered: true, AffectedCount: 1}
}
