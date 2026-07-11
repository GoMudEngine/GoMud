package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mutations"
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

	// A no-flee state (Blood Frenzy, hamstrung, winded, tackled, …) forbids
	// retreat: you can still move and fight, but you can't break off to flee.
	if user.Character.HasBuffFlag(buffs.NoFlee) {
		user.SendText(messaging.CategorySystem, `You can't break off to flee right now — you can only fight.`)
		return true, nil
	}

	if !user.Character.IsDisengaging() {
		// Fleeing costs stamina — a flyer breaks away far more easily (Winged Flight).
		fleeStaminaCost := 10
		if mutations.IsFlying(user.Character.Mutations) {
			fleeStaminaCost = int(float64(fleeStaminaCost) * float64(configs.GetBalanceConfig().FlightFleeStaminaMult))
			if fleeStaminaCost < 1 {
				fleeStaminaCost = 1
			}
		}
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
