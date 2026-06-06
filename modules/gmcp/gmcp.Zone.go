package gmcp

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mapper"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
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
	Zone  string                `json:"zone"`
	Rooms []mapper.SnapshotRoom `json:"rooms"`
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
	if !ok || evt.UserId == 0 {
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

	payload := GMCPZoneModule_Payload{
		Zone:  room.Zone,
		Rooms: m.Snapshot(visited),
	}

	events.AddToQueue(GMCPOut{
		UserId:  evt.UserId,
		Module:  `Zone.Map`,
		Payload: payload,
	})

	mudlog.Debug("gmcp.Zone", "userId", evt.UserId, "zone", room.Zone, "rooms", len(payload.Rooms))
	return events.Continue
}
