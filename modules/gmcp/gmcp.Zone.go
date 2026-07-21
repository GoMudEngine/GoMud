package gmcp

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mapper"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

type GMCPZoneModule struct {
	plug *plugins.Plugin
}

// GMCPZoneUpdate asks the system to (re)send the explored-zone map snapshot.
type GMCPZoneUpdate struct {
	UserId int
}

func (g GMCPZoneUpdate) Type() string { return `GMCPZoneUpdate` }

type GMCPZoneModule_Payload struct {
	Zone     string                `json:"zone"`
	CurrentZ int                   `json:"cz"`              // z-level (floor) of the player's current room
	Party    []int                 `json:"party,omitempty"` // room IDs currently holding party members
	Rooms    []mapper.SnapshotRoom `json:"rooms"`
}

func init() {
	g := GMCPZoneModule{
		plug: plugins.New(`gmcp.Zone`, `1.0`),
	}
	events.RegisterListener(events.RoomChange{}, g.roomChangeHandler)
	events.RegisterListener(GMCPZoneUpdate{}, g.buildAndSend)
}

func (g *GMCPZoneModule) roomChangeHandler(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.RoomChange)
	if !ok {
		mudlog.Error("Event", "Expected Type", "RoomChange", "Actual Type", e.Type())
		return events.Cancel
	}
	// Mob-only move (UserId == 0) — no connected user to send a map snapshot to.
	if evt.UserId == 0 {
		return events.Continue
	}
	events.AddToQueue(GMCPZoneUpdate{UserId: evt.UserId})
	return events.Continue
}

func (g *GMCPZoneModule) buildAndSend(e events.Event) events.ListenerReturn {
	evt, ok := e.(GMCPZoneUpdate)
	if !ok || evt.UserId < 1 {
		return events.Continue
	}
	user := users.GetByUserId(evt.UserId)
	if user == nil {
		return events.Continue
	}
	if !isGMCPEnabled(user.ConnectionId()) {
		return events.Cancel
	}

	room := rooms.LoadRoom(user.Character.RoomId)
	if room == nil {
		return events.Continue
	}

	m := mapper.GetMapper(room.RoomId)
	if m == nil {
		return events.Continue
	}

	visited := map[int]struct{}{}
	for _, id := range user.Character.GetVisitedRooms(room.Zone) {
		visited[id] = struct{}{}
	}
	// Always include the current room even before the move-handler marks it.
	visited[room.RoomId] = struct{}{}

	// Current floor (z) so the client shows only the level the player is on.
	_, _, cz, _ := m.GetCoordinates(room.RoomId)

	// Party-member room positions (excluding self), de-duplicated.
	party := []int{}
	if p := parties.Get(evt.UserId); p != nil {
		seen := map[int]bool{}
		for _, uId := range p.GetMembers() {
			if uId == evt.UserId {
				continue
			}
			if mu := users.GetByUserId(uId); mu != nil {
				// Translate an ephemeral instance id to its template id so a
				// party member standing in an instance (tutorial/dungeon) does
				// not leak the raw instance id onto the map; no-op for normal
				// rooms.
				rid, _ := rooms.OriginalRoomId(mu.Character.RoomId)
				if !seen[rid] {
					seen[rid] = true
					party = append(party, rid)
				}
			}
		}
	}

	payload := GMCPZoneModule_Payload{
		Zone:     room.Zone,
		CurrentZ: cz,
		Party:    party,
		Rooms:    m.Snapshot(visited),
	}

	events.AddToQueue(GMCPOut{
		UserId:  evt.UserId,
		Module:  `Zone.Map`,
		Payload: payload,
	})

	mudlog.Debug("gmcp.Zone", "userId", evt.UserId, "zone", room.Zone, "rooms", len(payload.Rooms))
	return events.Continue
}
