package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
)

// Salvage runs a single-tick corpse salvage for the mob. Thin
// wrapper over actions.Salvage with TargetCorpse=true. The action
// handles corpse-finding, yield rolling, material storage, and
// room flavor (when players are present).
//
// Activity transitions are managed here (not in actions.Salvage) to
// maintain the Salvaging state for combat-entry veto logic. Guard
// nil Activity — test mobs constructed outside New() may not have
// the machine initialized yet.
func Salvage(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	// Transition to Salvaging before action.
	if mob.Character.Activity != nil {
		_ = mob.Character.Activity.TransitionToSalvaging(
			activity.SalvagingData{
				ItemUuid:    "corpse-salvage", // placeholder; not used by mob path
				RoundsTotal: 1,
			},
			state.TransitionReason{
				Trigger: activity.TriggerSalvageBegin,
				Actor:   state.ActorRef{MobInstanceId: mob.InstanceId},
			},
		)
	}

	// Run the action.
	actor := actions.NewMobActorInRoom(mob, room)
	_ = actions.Salvage(actor, actions.SalvageOptions{TargetCorpse: true})

	// Return Activity machine to Free — single-tick resolution complete.
	if mob.Character.Activity != nil {
		_ = mob.Character.Activity.TransitionToFree(state.TransitionReason{
			Trigger: activity.TriggerSalvageComplete,
			Actor:   state.ActorRef{MobInstanceId: mob.InstanceId},
		})
	}

	return true, nil
}
