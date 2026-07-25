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
	Zones    []string              `json:"zones,omitempty"`  // all zone names (builder zone-switcher only; empty for the play client)
	Biomes   []string              `json:"biomes,omitempty"` // valid biome ids (builder new-zone form; empty for the play client)
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

// ---- Build.Zone.* (admin web-building 4) --------------------------------
//
// Build.Zone.* GMCP packages — the server side of the admin web zone editor.
// Handlers take a zoneDeps so they unit-test against a fake world; realZoneDeps
// wires the live engine.

type zoneDeps struct {
	load      func(zone string) *rooms.ZoneConfig
	save      func(cfg rooms.ZoneConfig) error
	del       func(zone string) error
	blockers  func(zone string) []rooms.ZoneBlocker
	zoneNames func() []string
	roomIds   func(zone string) []int
}

func realZoneDeps() zoneDeps {
	return zoneDeps{
		load:      rooms.GetZoneConfig,
		save:      func(cfg rooms.ZoneConfig) error { return rooms.SaveZoneConfig(&cfg) },
		del:       rooms.DeleteZone,
		blockers:  rooms.ZoneDeletionBlockers,
		zoneNames: rooms.GetAllZoneNames,
		roomIds: func(zone string) []int {
			out := []int{}
			for _, id := range rooms.GetAllRoomIds() {
				if r := rooms.LoadRoomTemplate(id); r != nil && r.Zone == zone {
					out = append(out, id)
				}
			}
			return out
		},
	}
}

func buildZoneDelete(d zoneDeps, zone string) BuildResult {
	if d.load(zone) == nil {
		return buildErr("zone %q not found", zone)
	}
	if b := d.blockers(zone); len(b) > 0 {
		return BuildResult{
			Error:    "zone is not empty — remove these first",
			ZoneRefs: b,
		}
	}
	if err := d.del(zone); err != nil {
		return buildErr("could not delete zone %q: %s", zone, err.Error())
	}
	return BuildResult{Ok: true, Message: "zone " + zone + " deleted"}
}
