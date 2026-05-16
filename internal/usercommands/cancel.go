package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Cancel aborts any in-progress activity (casting, crafting, salvaging).
// Casting: refunds 50% of unspent conviction.
// Crafting: no refund (materials not consumed until completion).
// Salvaging: no refund (item not consumed until completion).
func Cancel(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	a := user.Character.Activity
	if a == nil || a.IsFree() {
		user.SendText(`You aren't doing anything to cancel.`)
		return true, nil
	}

	switch a.State() {
	case activity.Casting:
		d, _ := a.CastingData()
		// Refund 50% of unspent conviction (existing behavior, preserved).
		unspent := d.TotalConvictionCost - d.ConvictionSpent
		if unspent > 0 {
			refund := unspent / 2
			user.Character.Conviction += refund
			if user.Character.Conviction > user.Character.ConvictionMax.Value {
				user.Character.Conviction = user.Character.ConvictionMax.Value
			}
		}
		_ = a.TransitionToFree(state.TransitionReason{
			Trigger: activity.TriggerCastCancel,
			Actor:   state.ActorRef{UserId: user.UserId},
		})
		user.SendText(`You stop casting.`)

	case activity.Crafting:
		_ = a.TransitionToFree(state.TransitionReason{
			Trigger: activity.TriggerCraftCancel,
			Actor:   state.ActorRef{UserId: user.UserId},
		})
		user.SendText(`You stop crafting.`)

	case activity.Salvaging:
		_ = a.TransitionToFree(state.TransitionReason{
			Trigger: activity.TriggerSalvageCancel,
			Actor:   state.ActorRef{UserId: user.UserId},
		})
		user.SendText(`You stop salvaging.`)
	}
	return true, nil
}
