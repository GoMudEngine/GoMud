package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// ToxicBite is a thin wrapper over actions.TriggerToxicBite.
// All gates, scoring, single-target damage, poison application, and player
// messages live in the action. This wrapper resolves the player's engaged
// combat target (same as the original command, which used EngagedTarget()).
func ToxicBite(rest string, user *users.UserRecord, room *rooms.Room,
	flags events.EventFlag) (bool, error) {

	actor := actions.NewUserActorInRoom(user, room)

	// Resolve the player's current engaged combat target into an Actor.
	// This mirrors the original command's use of EngagedTarget(), which
	// returns the mob/user ID the player is currently fighting.
	var target actions.Actor
	engaged := user.Character.EngagedTarget()
	if engaged.MobInstanceId > 0 {
		if m := mobs.GetInstance(engaged.MobInstanceId); m != nil {
			target = actions.NewMobActorInRoom(m, room)
		}
	} else if engaged.UserId > 0 {
		if u := users.GetByUserId(engaged.UserId); u != nil {
			target = actions.NewUserActorInRoom(u, room)
		}
	}

	_ = actions.TriggerToxicBite(actor, actions.MutationOpts{TargetActor: target})
	return true, nil
}
