package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/combatphase"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Flee(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// A no-go root (e.g. a Jailed holding-cell buff — 5.1c) pins the player in
	// place; flee must honor it too, or it becomes a jail-escape hole (the
	// directional `go` block alone is bypassable via flee — smoke BUG-02).
	if user.Character.HasBuffFlag(buffs.NoMovement) {
		user.SendText(messaging.CategorySystem, `You're locked in — there's nowhere to flee to.`)
		return true, nil
	}

	// A no-flee state (e.g. the Blood Frenzy mutation state) forbids retreat:
	// you can still move and fight, but you cannot make yourself turn and run.
	if user.Character.HasBuffFlag(buffs.NoFlee) {
		user.SendText(messaging.CategorySystem, `Something in you refuses to turn your back and run — you can only fight.`)
		return true, nil
	}

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
