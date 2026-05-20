package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/combatphase"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Flee(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if !user.Character.IsDisengaging() {
		// Fleeing costs stamina
		const fleeStaminaCost = 10
		if !user.Character.DeductStamina(fleeStaminaCost) {
			user.SendText(messaging.CategorySystem, `You're too exhausted to flee! You need to stand and fight.`)
			return true, nil
		}

		user.SendText(messaging.CategorySystem, `You attempt to flee...`)

		// Task 15: use CombatPhase.TransitionToDisengaging instead of the
		// legacy Aggro{Type:Flee} sentinel. The round driver's handlePlayerFlee
		// checks IsDisengaging() which reads CombatPhase state directly.
		// Veto errors (e.g., grappled) are silently ignored here — the grapple
		// check in handlePlayerFlee catches them a moment later.
		// TODO Task 18: no legacy fallback needed; Aggro field is gone.
		if user.Character.CombatPhase != nil {
			_ = user.Character.CombatPhase.TransitionToDisengaging(state.TransitionReason{
				Trigger: combatphase.TriggerFleeCommand,
				Actor:   state.ActorRef{UserId: user.UserId},
			})
		}
	}

	return true, nil
}
