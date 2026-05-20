package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/awareness"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Logout_AwarenessCleanup forces Awareness through Revealing →
// Visible synchronously when a player logs out or disconnects.
// Ensures hidden players don't leave the world Hidden; on
// reconnect they're Visible. Room broadcast fires so observers
// in the room see the leave.
//
// Registered at default priority so it fires before HandleLeave
// (events.Last), while the UserRecord is still in the manager.
//
// Mob despawn is handled by instance destruction — the
// Awareness machine on the Character goes with the instance,
// so no separate cascade is needed.
func init() {
	events.RegisterListener(events.PlayerDespawn{}, onPlayerDespawnForAwareness)
}

func onPlayerDespawnForAwareness(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.PlayerDespawn)
	if !ok {
		return events.Continue
	}
	u := users.GetByUserId(evt.UserId)
	if u == nil {
		return events.Continue
	}
	u.Character.Awareness.ForceVisible(state.TransitionReason{
		Trigger: awareness.TriggerLogout,
		Actor:   state.ActorRef{UserId: evt.UserId},
	})
	return events.Continue
}
