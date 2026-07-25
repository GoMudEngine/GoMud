package gmcp

import (
	"sort"

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
	// rename + renameBlockers are Phase 2. renameBlockers is deliberately
	// separate from blockers: the two policies differ. A delete is blocked by
	// rooms, authored content and inbound exits; a rename is blocked only by
	// players, because everything else is rewritten by the rename itself.
	rename         func(oldName, newName string) error
	renameBlockers func(zone string) []rooms.ZoneBlocker
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
		rename:         rooms.RenameZone,
		renameBlockers: rooms.ZoneRenameBlockers,
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

// ---- server -> client payloads ----

type zoneListRow struct {
	Zone      string `json:"zone"`
	RoomCount int    `json:"roomCount"`
	Instanced bool   `json:"instanced"`
}

type zoneListPayload struct {
	Zones []zoneListRow `json:"zones"`
}

type zoneEnums struct {
	Biomes        []string `json:"biomes"`
	DeathPolicies []string `json:"deathPolicies"`
	Regions       []string `json:"regions"` // observed values, as suggestions
}

// zoneUpdateReq is both the client->server update payload and the editable
// half of zoneDetail. Name is NOT editable here — renaming is Phase 2.
type zoneUpdateReq struct {
	Name           string   `json:"name"`
	RoomId         int      `json:"roomId"`
	DefaultBiome   string   `json:"defaultBiome"`
	Region         string   `json:"region"`
	MusicFile      string   `json:"musicFile"`
	IdleMessages   []string `json:"idleMessages"`
	Instanced      bool     `json:"instanced"`
	DeathPolicy    string   `json:"deathPolicy"`
	PortalDuration string   `json:"portalDuration"`
	EntryRoom      int      `json:"entryRoom"`
	AllowRecall    bool     `json:"allowRecall"`
	NonCartesian   bool     `json:"nonCartesian"`
	DefaultPlane   int      `json:"defaultPlane"`
}

type zoneDetail struct {
	zoneUpdateReq
	Enums zoneEnums `json:"enums"`
}

type zoneGetReq struct {
	Zone string `json:"zone"`
}

type zoneDeleteReq struct {
	Zone string `json:"zone"`
}

// buildZoneList reads RoomCount from ZoneConfig.RoomIds — the engine's own
// per-zone room lookup, populated as rooms load and pruned only by a genuine
// delete (idle-room unloading drops the room from roomManager.rooms but leaves
// RoomIds intact, so the count stays right as the server ages).
//
// It deliberately does NOT use the roomIds dep here. That dep's real wiring
// rescans every room in the world with a disk read + YAML parse per room, and
// calling it once per zone made this list cost 49 x 1384 = 67,816 template
// loads (~10s measured) to render a handful of numbers. RoomIds gives the same
// counts — verified identical across all 49 zones — for free.
func buildZoneList(d zoneDeps) zoneListPayload {
	out := zoneListPayload{Zones: []zoneListRow{}}
	for _, z := range d.zoneNames() {
		cfg := d.load(z)
		row := zoneListRow{Zone: z}
		if cfg != nil {
			row.Instanced = cfg.Instanced
			row.RoomCount = len(cfg.RoomIds)
		}
		out.Zones = append(out.Zones, row)
	}
	return out
}

func buildZoneGet(d zoneDeps, zone string) (zoneDetail, bool) {
	cfg := d.load(zone)
	if cfg == nil {
		return zoneDetail{}, false
	}
	return zoneDetail{
		zoneUpdateReq: zoneUpdateReq{
			Name: cfg.Name, RoomId: cfg.RoomId,
			DefaultBiome: cfg.DefaultBiome, Region: cfg.Region,
			MusicFile: cfg.MusicFile, IdleMessages: cfg.IdleMessages,
			Instanced: cfg.Instanced, DeathPolicy: cfg.DeathPolicy,
			PortalDuration: cfg.PortalDuration, EntryRoom: cfg.EntryRoom,
			AllowRecall: cfg.AllowRecall, NonCartesian: cfg.NonCartesian,
			DefaultPlane: cfg.DefaultPlane,
		},
		Enums: collectZoneEnums(d),
	}, true
}

func collectZoneEnums(d zoneDeps) zoneEnums {
	regions := []string{}
	seen := map[string]bool{}
	for _, z := range d.zoneNames() {
		if cfg := d.load(z); cfg != nil && cfg.Region != "" && !seen[cfg.Region] {
			seen[cfg.Region] = true
			regions = append(regions, cfg.Region)
		}
	}
	return zoneEnums{
		Biomes:        zoneBiomeNames(),
		DeathPolicies: []string{"rejoin", "ejected"},
		Regions:       regions,
	}
}

// zoneBiomeNames returns every biome id the engine knows, for the editor's
// dropdown. Server-supplied so a typo cannot reach zone-config.
func zoneBiomeNames() []string {
	out := []string{}
	for _, b := range rooms.GetAllBiomes() {
		bi := b
		out = append(out, bi.Id())
	}
	sort.Strings(out)
	return out
}

func buildZoneUpdate(d zoneDeps, req zoneUpdateReq) BuildResult {
	cfg := d.load(req.Name)
	if cfg == nil {
		return buildErr("zone %q not found", req.Name)
	}

	// A non_cartesian zone marks its plane non-Euclidean, and PlaneRegistry.Mark
	// accumulates with a sticky OR — so on plane 0 it marks the ENTIRE overworld
	// non-Euclidean and silently disables collision + reciprocity enforcement
	// world-wide. Never allow that combination to be saved.
	if req.NonCartesian && req.DefaultPlane == 0 {
		return buildErr("a non-Cartesian zone needs its own plane — plane 0 is the overworld, " +
			"and flagging it non-Euclidean disables map consistency checks for the whole world")
	}

	if req.DeathPolicy != "" && req.DeathPolicy != "rejoin" && req.DeathPolicy != "ejected" {
		return buildErr("death_policy %q invalid; valid: rejoin, ejected", req.DeathPolicy)
	}

	inZone := map[int]bool{}
	for _, id := range d.roomIds(req.Name) {
		inZone[id] = true
	}
	if req.EntryRoom != 0 && !inZone[req.EntryRoom] {
		return buildErr("entry room %d is not in zone %q", req.EntryRoom, req.Name)
	}
	if req.RoomId != 0 && !inZone[req.RoomId] {
		return buildErr("root room %d is not in zone %q", req.RoomId, req.Name)
	}

	// Copy the editable fields onto the loaded config so anything the form does
	// not carry (RoomIds, Mutators) survives the save.
	out := *cfg
	out.RoomId = req.RoomId
	out.DefaultBiome = req.DefaultBiome
	out.Region = req.Region
	out.MusicFile = req.MusicFile
	out.IdleMessages = req.IdleMessages
	out.Instanced = req.Instanced
	out.DeathPolicy = req.DeathPolicy
	out.PortalDuration = req.PortalDuration
	out.EntryRoom = req.EntryRoom
	out.AllowRecall = req.AllowRecall
	out.NonCartesian = req.NonCartesian
	out.DefaultPlane = req.DefaultPlane

	if err := d.save(out); err != nil {
		return buildErr("could not save zone %q: %s", req.Name, err.Error())
	}
	return BuildResult{Ok: true, Message: "zone " + req.Name + " saved"}
}

// ---- senders ----

func sendZoneList(uid int) {
	sendGMCP(uid, `Build.Zones`, buildZoneList(realZoneDeps()))
}

func sendZoneDetail(uid int, d zoneDetail) {
	sendGMCP(uid, `Build.Zone`, d)
}

type zoneRenameReq struct {
	Zone    string `json:"zone"`
	NewName string `json:"newName"`
}

// buildZoneRename refuses before it acts: unknown zone, then blockers, then
// the rename itself. rooms.RenameZone re-validates the new name and re-checks
// the blockers on its own — never trust the caller — so this layer's job is to
// surface a refusal in a form the editor can render.
func buildZoneRename(d zoneDeps, req zoneRenameReq) BuildResult {
	if d.load(req.Zone) == nil {
		return buildErr("zone %q not found", req.Zone)
	}
	if b := d.renameBlockers(req.Zone); len(b) > 0 {
		return BuildResult{
			Error:    "zone cannot be renamed right now — these must clear first",
			ZoneRefs: b,
		}
	}
	if err := d.rename(req.Zone, req.NewName); err != nil {
		return buildErr("could not rename zone %q: %s", req.Zone, err.Error())
	}
	return BuildResult{Ok: true, Message: "zone renamed to " + req.NewName}
}
