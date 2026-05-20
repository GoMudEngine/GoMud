// Wake any Dormant mobs in the destination room when a player enters.
//
// Chunk 5 (Presence): the centralized RoomChange hook supplements the
// rooms.go round-tick path so that mobs wake immediately on player entry
// rather than waiting until the next tick's player-presence scan.
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/presence"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// PresencePlayerEntry wakes Dormant mobs in the destination room
// whenever a player moves into it. Mob-to-mob movement is intentionally
// ignored — only nearby players trigger the wake-up.
func PresencePlayerEntry(e events.Event) events.ListenerReturn {
	evt := e.(events.RoomChange)

	// Only fire on player entry (not mob movement).
	if evt.UserId == 0 {
		return events.Continue
	}
	if users.GetByUserId(evt.UserId) == nil {
		return events.Continue
	}

	room := rooms.LoadRoom(evt.ToRoomId)
	if room == nil {
		return events.Continue
	}

	for _, mobInstId := range room.GetMobs() {
		mob := mobs.GetInstance(mobInstId)
		if mob == nil || mob.Character.Presence == nil {
			continue
		}
		if mob.Character.Presence.State() == presence.Dormant {
			_ = mob.Character.Presence.TransitionTo(presence.Active,
				state.TransitionReason{Trigger: presence.TriggerPlayerEntry})
			mob.Character.LastDormantEntryRound = 0
		}
	}

	return events.Continue
}
