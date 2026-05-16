package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
)

// Cancel aborts any in-progress activity for the mob (casting,
// crafting, salvaging). Mirrors the player cancel command.
// Casting: refunds 50% of unspent conviction.
// Crafting/Salvaging: no refund.
func Cancel(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	a := mob.Character.Activity
	if a == nil || a.IsFree() {
		return true, nil
	}

	switch a.State() {
	case activity.Casting:
		d, _ := a.CastingData()
		unspent := d.TotalConvictionCost - d.ConvictionSpent
		if unspent > 0 {
			refund := unspent / 2
			mob.Character.Conviction += refund
			if mob.Character.Conviction > mob.Character.ConvictionMax.Value {
				mob.Character.Conviction = mob.Character.ConvictionMax.Value
			}
		}
		_ = a.TransitionToFree(state.TransitionReason{
			Trigger: activity.TriggerCastCancel,
			Actor:   state.ActorRef{MobInstanceId: mob.InstanceId},
		})
		// Legacy parallel-write clear; removed in Task 11.
		mob.Character.CastingState = nil

	case activity.Crafting:
		_ = a.TransitionToFree(state.TransitionReason{
			Trigger: activity.TriggerCraftCancel,
			Actor:   state.ActorRef{MobInstanceId: mob.InstanceId},
		})
		mob.Character.CraftingState = nil

	case activity.Salvaging:
		_ = a.TransitionToFree(state.TransitionReason{
			Trigger: activity.TriggerSalvageCancel,
			Actor:   state.ActorRef{MobInstanceId: mob.InstanceId},
		})
		mob.Character.CraftingState = nil
		delete(mob.Character.MiscData, "salvage_item_uuid")
		delete(mob.Character.MiscData, "salvage_spoiled_potion")
	}
	return true, nil
}
