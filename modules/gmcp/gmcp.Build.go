package gmcp

// Build.* GMCP packages — the server side of the admin web room-builder (admin
// web-building 1b). Each package is RoleAdmin-gated in HandleIAC; the routing
// there resolves the admin and forwards to the pure-ish core functions below,
// which mutate room templates through the rooms package and reply with a
// Build.Result plus a refreshed (unfogged) Zone.Map snapshot.
//
// The core (buildRoomCreate/Update/Delete, buildExitAdd/Remove) takes a
// buildDeps seam so it is unit-testable against a fake world; realBuildDeps()
// wires the seam to the live rooms/mapper packages.

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/gamelock"
	"github.com/GoMudEngine/GoMud/internal/mapper"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Placeholders for a freshly ghost-created room. Both fields must be non-empty
// or Room.Validate() rejects the room on its next load.
const (
	untitledRoomTitle          = "Untitled Room"
	placeholderRoomDescription = "This room is under construction and has not yet been described."
)

// BuildResult is echoed to the client after every mutation (as Build.Result).
type BuildResult struct {
	Ok       bool                `json:"ok"`
	Error    string              `json:"error,omitempty"`
	RoomId   int                 `json:"roomId,omitempty"`   // e.g. the newly created room, or the affected room
	ItemId   int                 `json:"itemId,omitempty"`   // e.g. the created/updated item
	Refs     []itemRef           `json:"refs,omitempty"`     // Build.Item.Delete: what still references a blocked item
	MobId    int                 `json:"mobId,omitempty"`    // e.g. the created/updated/spawned mob
	MobRefs  []mobRefEntry       `json:"mobRefs,omitempty"`  // Build.Mob.Delete: what still references a blocked mob
	ZoneRefs []rooms.ZoneBlocker `json:"zoneRefs,omitempty"` // Build.Zone.Delete: what blocks a delete
	Message  string              `json:"message,omitempty"`  // e.g. Build.Mob.Spawn's "<name> spawned in room <id>"
}

func buildErr(format string, args ...any) BuildResult {
	return BuildResult{Error: fmt.Sprintf(format, args...)}
}

// ---- client -> server payloads -----------------------------------------------

type roomCreateReq struct {
	FromRoomId int    `json:"fromRoomId"`
	Dir        string `json:"dir"`
	Plane      int    `json:"plane"`
	X          int    `json:"x"`
	Y          int    `json:"y"`
	Z          int    `json:"z"`
}

type roomUpdateReq struct {
	RoomId        int               `json:"roomId"`
	Title         string            `json:"title"`
	Description   string            `json:"description"`
	Biome         string            `json:"biome"`
	Symbol        string            `json:"symbol"`
	Legend        string            `json:"legend"`
	Music         string            `json:"music"`
	Bank          bool              `json:"bank"`
	Storage       bool              `json:"storage"`
	Pvp           bool              `json:"pvp"`
	CharacterRoom bool              `json:"characterRoom"`
	Nouns         map[string]string `json:"nouns"`
	IdleMessages  []string          `json:"idleMessages"`
}

type roomDeleteReq struct {
	RoomId int `json:"roomId"`
}

type exitAddReq struct {
	RoomId     int    `json:"roomId"`
	Dir        string `json:"dir"`        // spatial (compass); empty for a portal
	PortalName string `json:"portalName"` // portal/door name; empty for a spatial exit
	ToRoomId   int    `json:"toRoomId"`
	Secret     bool   `json:"secret"`
	OneWay     bool   `json:"oneway"`
	Lock       int    `json:"lock"` // lock difficulty (0 = no lock)
	Message    string `json:"message"`
}

type exitRemoveReq struct {
	RoomId   int    `json:"roomId"`
	ExitName string `json:"exitName"`
}

type mapRequestReq struct {
	Zone string `json:"zone"` // optional; defaults to the admin's current zone
}

type roomGetReq struct {
	RoomId int `json:"roomId"`
}

type roomListReq struct {
	Zone string `json:"zone"`
}

// buildRoomListRow is one row of Build.Rooms — the test-spawn room picker's
// data source (admin web-building 3 fix round). Deliberately minimal (id +
// title only): it exists so the mob editor's test-spawn control can offer a
// per-zone room dropdown instead of asking the admin to type a raw room id.
type buildRoomListRow struct {
	Id    int    `json:"id"`
	Title string `json:"title"`
}

type zoneCreateReq struct {
	Name         string `json:"name"`
	Biome        string `json:"biome"`
	Region       string `json:"region"`
	NonCartesian bool   `json:"nonCartesian"`
}

// ---- server -> client room detail (Build.Room) -------------------------------
// The Zone.Map snapshot only carries map-rendering fields; the inspector needs
// the full editable template, fetched per room via Build.Room.Get.

type buildExitDetail struct {
	Name     string `json:"name"`
	ToRoomId int    `json:"toRoomId"`
	Dir      string `json:"dir,omitempty"` // set when the exit is spatial (compass)
	Secret   bool   `json:"secret,omitempty"`
	OneWay   bool   `json:"oneway,omitempty"`
	Lock     int    `json:"lock,omitempty"`
	Message  string `json:"message,omitempty"`
}

type buildRoomDetail struct {
	RoomId        int               `json:"roomId"`
	Title         string            `json:"title"`
	Description   string            `json:"description"`
	Zone          string            `json:"zone"`
	Biome         string            `json:"biome"`
	Symbol        string            `json:"symbol"`
	Legend        string            `json:"legend"`
	Music         string            `json:"music"`
	Bank          bool              `json:"bank"`
	Storage       bool              `json:"storage"`
	Pvp           bool              `json:"pvp"`
	CharacterRoom bool              `json:"characterRoom"`
	Nouns         map[string]string `json:"nouns"`
	IdleMessages  []string          `json:"idleMessages"`
	Exits         []buildExitDetail `json:"exits"`
	Plane         int               `json:"plane"`
	X             int               `json:"x"`
	Y             int               `json:"y"`
	Z             int               `json:"z"`
	Biomes        []string          `json:"biomes,omitempty"` // valid biome ids for the dropdown
}

// ---- dependency seam ---------------------------------------------------------

// buildDeps is the seam over the rooms/mapper packages so the build core is
// unit-testable with a fake world. Production wires it via realBuildDeps().
type buildDeps struct {
	loadTemplate   func(id int) *rooms.Room
	save           func(r rooms.Room) error
	deleteTemplate func(id int) error
	validatePlace  func(plane, x, y, z, exclude int) error
	newRoom        func(zone string) *rooms.Room
	allRoomIds     func() []int
	reciprocal     func(dir string) string
	delta          func(dir string) (x, y, z int)
	isCompass      func(dir string) bool
	isNonEuclidean func(plane int) bool
}

func realBuildDeps() buildDeps {
	return buildDeps{
		// LoadRoomTemplate returns authoritative authored coords: the 0.15.0
		// migration wrote flat x/y only for non-origin rooms and re-derives
		// origin rooms as (0,0,0), so a coord-only file always means position
		// (0,0,0). Any inconsistency would fail the panic-mode cartcheck at
		// boot, so a bootable world's template coords are always correct.
		loadTemplate:   rooms.LoadRoomTemplate,
		save:           rooms.SaveRoomTemplate,
		deleteTemplate: rooms.DeleteRoomTemplate,
		validatePlace:  rooms.ValidatePlacement,
		newRoom:        rooms.NewRoom,
		allRoomIds:     rooms.GetAllRoomIds,
		reciprocal:     reciprocalCompass,
		delta:          mapper.GetDelta,
		isCompass:      mapper.IsCompassDirection,
		isNonEuclidean: func(plane int) bool { return rooms.GetPlaneRegistry().IsNonEuclidean(plane) },
	}
}

// ---- admin gate --------------------------------------------------------------

// compassOpposite is the reciprocal for each spatial direction.
//
// mapper.GetReciprocalExit is deterministic as of the 2026-07-25 fix, but this
// table is deliberately NARROWER: it covers only the ten plain compass names
// the builder's grid model can create, and returns "" for everything else.
// buildExitRemove calls reciprocal() UNGATED on whatever exit name it is given,
// so widening this to the mapper's full vocabulary would start deleting
// "-x2"/"-gap" return legs that the builder never authored. Keep it narrow.
var compassOpposite = map[string]string{
	"north": "south", "south": "north",
	"east": "west", "west": "east",
	"northeast": "southwest", "southwest": "northeast",
	"northwest": "southeast", "southeast": "northwest",
	"up": "down", "down": "up",
}

func reciprocalCompass(dir string) string { return compassOpposite[dir] }

func roleIsAdmin(u *users.UserRecord) bool {
	return u != nil && u.Role == users.RoleAdmin
}

// GMCPBuildOp carries a Build.* request from the connection goroutine to
// MainWorker, where all room/mapper mutation must happen (the room manager is
// not safe for concurrent access with the world tick).
type GMCPBuildOp struct {
	ConnectionId uint64
	Command      string
	Payload      []byte
}

func (e GMCPBuildOp) Type() string { return `GMCPBuildOp` }

// handleBuildOp processes a deferred Build.* request on the MainWorker
// goroutine: it resolves + admin-gates the requester, dispatches to the build
// core, and replies with a Build.Result / Build.Room plus a refreshed unfogged
// Zone.Map for the affected zone.
func (g *GMCPModule) handleBuildOp(e events.Event) events.ListenerReturn {
	evt, ok := e.(GMCPBuildOp)
	if !ok {
		return events.Cancel
	}
	uid, ok := requireAdmin(evt.ConnectionId)
	if !ok {
		return events.Continue // non-admins are silently refused
	}
	d := realBuildDeps()

	switch evt.Command {
	case `Build.Room.Create`:
		var req roomCreateReq
		if json.Unmarshal(evt.Payload, &req) != nil {
			sendBuildResult(uid, buildErr("bad Build.Room.Create payload"))
			break
		}
		// Refresh the zone the new room lives in (the source room's zone),
		// which may differ from the zone the admin is standing in.
		refreshZone := zoneOfRoom(req.FromRoomId, uid)
		sendBuildResult(uid, buildRoomCreate(d, req.FromRoomId, req.Dir, req.Plane, req.X, req.Y, req.Z))
		sendZoneMapFull(uid, refreshZone)

	case `Build.Room.Update`:
		var req roomUpdateReq
		if json.Unmarshal(evt.Payload, &req) != nil {
			sendBuildResult(uid, buildErr("bad Build.Room.Update payload"))
			break
		}
		refreshZone := zoneOfRoom(req.RoomId, uid)
		sendBuildResult(uid, buildRoomUpdate(d, req))
		sendZoneMapFull(uid, refreshZone)

	case `Build.Room.Delete`:
		var req roomDeleteReq
		if json.Unmarshal(evt.Payload, &req) != nil {
			sendBuildResult(uid, buildErr("bad Build.Room.Delete payload"))
			break
		}
		refreshZone := zoneOfRoom(req.RoomId, uid) // capture before the room is gone
		sendBuildResult(uid, buildRoomDelete(d, req.RoomId))
		sendZoneMapFull(uid, refreshZone)

	case `Build.Exit.Add`:
		var req exitAddReq
		if json.Unmarshal(evt.Payload, &req) != nil {
			sendBuildResult(uid, buildErr("bad Build.Exit.Add payload"))
			break
		}
		refreshZone := zoneOfRoom(req.RoomId, uid)
		sendBuildResult(uid, buildExitAdd(d, req))
		sendZoneMapFull(uid, refreshZone)

	case `Build.Exit.Remove`:
		var req exitRemoveReq
		if json.Unmarshal(evt.Payload, &req) != nil {
			sendBuildResult(uid, buildErr("bad Build.Exit.Remove payload"))
			break
		}
		refreshZone := zoneOfRoom(req.RoomId, uid)
		sendBuildResult(uid, buildExitRemove(d, req.RoomId, req.ExitName))
		sendZoneMapFull(uid, refreshZone)

	case `Build.Room.Get`:
		var req roomGetReq
		if json.Unmarshal(evt.Payload, &req) != nil {
			sendBuildResult(uid, buildErr("bad Build.Room.Get payload"))
			break
		}
		if detail, found := buildRoomGet(d, req.RoomId); found {
			sendRoomDetail(uid, detail)
		} else {
			sendBuildResult(uid, buildErr("room %d not found", req.RoomId))
		}

	case `Build.Room.List`:
		var req roomListReq
		if json.Unmarshal(evt.Payload, &req) != nil {
			sendBuildResult(uid, buildErr("bad Build.Room.List payload"))
			break
		}
		sendRoomList(uid, req.Zone)

	case `Build.Zone.Create`:
		var req zoneCreateReq
		if json.Unmarshal(evt.Payload, &req) != nil {
			sendBuildResult(uid, buildErr("bad Build.Zone.Create payload"))
			break
		}
		res := buildZoneCreate(req)
		sendBuildResult(uid, res)
		if res.Ok {
			// Jump the builder to the new zone.
			sendZoneMapFull(uid, strings.TrimSpace(req.Name))
		}

	case `Build.Zone.List`:
		sendZoneList(uid)
	case `Build.Zone.Get`:
		var req zoneGetReq
		if json.Unmarshal(evt.Payload, &req) != nil {
			sendBuildResult(uid, buildErr("bad Build.Zone.Get payload"))
			break
		}
		if d, ok := buildZoneGet(realZoneDeps(), req.Zone); ok {
			sendZoneDetail(uid, d)
		} else {
			sendBuildResult(uid, buildErr("zone %q not found", req.Zone))
		}
	case `Build.Zone.Update`:
		var req zoneUpdateReq
		if json.Unmarshal(evt.Payload, &req) != nil {
			sendBuildResult(uid, buildErr("bad Build.Zone.Update payload"))
			break
		}
		sendBuildResult(uid, buildZoneUpdate(realZoneDeps(), req))
		sendZoneList(uid)
	case `Build.Zone.Delete`:
		var req zoneDeleteReq
		if json.Unmarshal(evt.Payload, &req) != nil {
			sendBuildResult(uid, buildErr("bad Build.Zone.Delete payload"))
			break
		}
		res := buildZoneDelete(realZoneDeps(), req.Zone)
		if res.Ok {
			// rooms cannot import mapper, so cache invalidation happens here.
			mapper.ClearCache()
		}
		sendBuildResult(uid, res)
		sendZoneList(uid)

	case `Build.Map.Request`:
		var req mapRequestReq
		if json.Unmarshal(evt.Payload, &req) != nil {
			req = mapRequestReq{}
		}
		zone := req.Zone
		if zone == "" {
			zone = zoneForUser(uid)
		}
		sendZoneMapFull(uid, zone)

	// ---- item editor (admin web-building 2) ----
	case `Build.Item.List`:
		sendItemList(uid)
	case `Build.Item.Get`:
		var req itemGetReq
		if json.Unmarshal(evt.Payload, &req) != nil {
			sendBuildResult(uid, buildErr("bad Build.Item.Get payload"))
			break
		}
		if detail, ok := buildItemGet(realItemDeps(), req.ItemId); ok {
			sendItemDetail(uid, detail)
		} else {
			sendBuildResult(uid, buildErr("item %d not found", req.ItemId))
		}
	case `Build.Item.Create`:
		var req itemCreateReq
		if json.Unmarshal(evt.Payload, &req) != nil {
			sendBuildResult(uid, buildErr("bad Build.Item.Create payload"))
			break
		}
		sendBuildResult(uid, buildItemCreate(realItemDeps(), req.Type))
		sendItemList(uid)
	case `Build.Item.Update`:
		var req itemUpdateReq
		if json.Unmarshal(evt.Payload, &req) != nil {
			sendBuildResult(uid, buildErr("bad Build.Item.Update payload"))
			break
		}
		sendBuildResult(uid, buildItemUpdate(realItemDeps(), req))
		sendItemList(uid)
	case `Build.Item.Delete`:
		var req itemDeleteReq
		if json.Unmarshal(evt.Payload, &req) != nil {
			sendBuildResult(uid, buildErr("bad Build.Item.Delete payload"))
			break
		}
		res := buildItemDelete(realItemDeps(), req.ItemId)
		sendBuildResult(uid, res)
		if res.Ok {
			sendItemList(uid)
		}

	// ---- mob editor (admin web-building 3) ----
	case `Build.Mob.List`:
		sendMobList(uid)
	case `Build.Mob.Get`:
		var req mobGetReq
		if json.Unmarshal(evt.Payload, &req) != nil {
			sendBuildResult(uid, buildErr("bad Build.Mob.Get payload"))
			break
		}
		if d, ok := buildMobGet(realMobDeps(), req.MobId); ok {
			sendMobDetail(uid, d)
		} else {
			sendBuildResult(uid, buildErr("mob %d not found", req.MobId))
		}
	case `Build.Mob.Create`:
		var req mobCreateReq
		if json.Unmarshal(evt.Payload, &req) != nil {
			sendBuildResult(uid, buildErr("bad Build.Mob.Create payload"))
			break
		}
		res := buildMobCreate(realMobDeps(), req.Zone)
		sendBuildResult(uid, res)
		sendMobList(uid)
	case `Build.Mob.Update`:
		var req mobUpdateReq
		req.LLMProfileJSON = "-" // sentinel: field absent from the payload = leave profile untouched
		if json.Unmarshal(evt.Payload, &req) != nil {
			sendBuildResult(uid, buildErr("bad Build.Mob.Update payload"))
			break
		}
		sendBuildResult(uid, buildMobUpdate(realMobDeps(), req))
		sendMobList(uid)
	case `Build.Mob.Delete`:
		var req mobDeleteReq
		if json.Unmarshal(evt.Payload, &req) != nil {
			sendBuildResult(uid, buildErr("bad Build.Mob.Delete payload"))
			break
		}
		res := buildMobDelete(realMobDeps(), req.MobId)
		sendBuildResult(uid, res)
		if res.Ok {
			sendMobList(uid)
		}
	case `Build.Mob.Spawn`:
		var req mobSpawnReq
		if json.Unmarshal(evt.Payload, &req) != nil {
			sendBuildResult(uid, buildErr("bad Build.Mob.Spawn payload"))
			break
		}
		sendBuildResult(uid, buildMobSpawn(realMobDeps(), req))
	}
	return events.Continue
}

// requireAdmin resolves the connection's active user and returns its id iff the
// user is an admin.
func requireAdmin(connectionId uint64) (int, bool) {
	uid := userIdForConnection(connectionId)
	if uid <= 0 {
		return 0, false
	}
	if !roleIsAdmin(users.GetByUserId(uid)) {
		return 0, false
	}
	return uid, true
}

// ---- core mutations ----------------------------------------------------------

// buildRoomCreate creates a blank room at (plane,x,y,z) as a spatial neighbour
// of fromRoomId in dir, wiring reciprocal exits. The cell is validated first.
func buildRoomCreate(d buildDeps, fromRoomId int, dir string, plane, x, y, z int) BuildResult {
	from := d.loadTemplate(fromRoomId)
	if from == nil {
		return buildErr("source room %d not found", fromRoomId)
	}
	if !d.isCompass(dir) {
		return buildErr("ghost-create needs a compass direction, got %q", dir)
	}
	recip := d.reciprocal(dir)
	if recip == "" {
		return buildErr("no reciprocal direction for %q", dir)
	}
	if err := d.validatePlace(plane, x, y, z, 0); err != nil {
		return buildErr("%s", err.Error())
	}

	nr := d.newRoom(from.Zone) // inherit the source room's zone
	nr.X, nr.Y, nr.Z, nr.Plane = x, y, z, plane
	// Title and Description must be non-empty: Room.Validate() rejects blanks on
	// load, so a room saved with an empty description can never be read back
	// (it would break its own Get/Update). Seed valid placeholders for the
	// admin to overwrite in the inspector.
	nr.Title = untitledRoomTitle
	nr.Description = placeholderRoomDescription
	if err := d.save(*nr); err != nil {
		return buildErr("could not save new room: %s", err.Error())
	}

	// Reciprocal spatial exits (both directions), routed through save so both
	// templates persist.
	if r := applyExit(d, fromRoomId, dir, exit.RoomExit{RoomId: nr.RoomId}); !r.Ok {
		return r
	}
	if r := applyExit(d, nr.RoomId, recip, exit.RoomExit{RoomId: fromRoomId}); !r.Ok {
		return r
	}
	return BuildResult{Ok: true, RoomId: nr.RoomId}
}

// buildRoomUpdate loads the room's full template, overwrites the editable
// fields, and re-saves — preserving exits, coordinates, spawn lists, and other
// structure. Always routing through save is what fixes the historical
// biome/other-field non-persistence gaps.
func buildRoomUpdate(d buildDeps, req roomUpdateReq) BuildResult {
	// Room.Validate() rejects blank title/description on load, so refuse to save
	// a room into an unreadable state — surface the error in the inspector.
	if strings.TrimSpace(req.Title) == "" {
		return buildErr("title cannot be empty")
	}
	if strings.TrimSpace(req.Description) == "" {
		return buildErr("description cannot be empty")
	}
	r := d.loadTemplate(req.RoomId)
	if r == nil {
		return buildErr("room %d not found", req.RoomId)
	}
	r.Title = req.Title
	r.Description = req.Description
	r.Biome = req.Biome
	r.MapSymbol = req.Symbol
	r.MapLegend = req.Legend
	r.MusicFile = req.Music
	r.IsBank = req.Bank
	r.IsStorage = req.Storage
	r.Pvp = req.Pvp
	r.IsCharacterRoom = req.CharacterRoom
	r.Nouns = req.Nouns
	r.IdleMessages = req.IdleMessages
	if err := d.save(*r); err != nil {
		return buildErr("could not save room %d: %s", req.RoomId, err.Error())
	}
	return BuildResult{Ok: true, RoomId: req.RoomId}
}

// buildZoneCreate creates a brand-new zone: a folder + zone-config + a single
// valid entrance room placed on a FRESH authored plane (so its origin cell does
// not collide with the overworld or another zone), then applies the requested
// zone-config options. Returns the entrance room id so the builder can jump to
// it. Runs on MainWorker (via handleBuildOp).
func buildZoneCreate(req zoneCreateReq) BuildResult {
	name := strings.TrimSpace(req.Name)
	if len(name) < 2 {
		return buildErr("zone name must be at least 2 characters")
	}

	// CreateZone makes the zone folder, a zone-config, and a first valid room
	// (non-empty title/description), returning its id. Rejects a duplicate name.
	roomId, err := rooms.CreateZone(name)
	if err != nil {
		return buildErr("%s", err.Error())
	}

	// Put the new zone on its own plane so its (0,0,0) origin can't collide with
	// plane 0's origin at cartcheck. A non-Euclidean zone is exempt from the
	// collision check but still gets a distinct plane for clean rendering.
	plane := rooms.NextFreeAuthoredPlane()
	if r := rooms.LoadRoomTemplate(roomId); r != nil {
		r.Plane = plane
		r.X, r.Y, r.Z = 0, 0, 0
		r.Title = name + " — Entrance"
		r.Description = placeholderRoomDescription
		if req.Biome != "" {
			r.Biome = req.Biome
		}
		rooms.SaveRoomTemplate(*r)
	}

	if cfg := rooms.GetZoneConfig(name); cfg != nil {
		// CreateZone leaves the zone-config root RoomId unset, so GetZoneRoot
		// (used by the map refresh) would return 0 and render an empty zone.
		// Point it at the entrance room.
		cfg.RoomId = roomId
		cfg.DefaultBiome = req.Biome
		cfg.Region = req.Region
		cfg.NonCartesian = req.NonCartesian
		cfg.DefaultPlane = plane
		rooms.SaveZoneConfig(cfg)
	}
	// Register the new plane's Euclidean-ness for ValidatePlacement / mapper.
	rooms.GetPlaneRegistry().Mark(plane, req.NonCartesian, name)

	return BuildResult{Ok: true, RoomId: roomId}
}

// buildRoomDelete removes a room after cleaning every exit that points at it,
// so no dangling exits remain.
func buildRoomDelete(d buildDeps, roomId int) BuildResult {
	if d.loadTemplate(roomId) == nil {
		return buildErr("room %d not found", roomId)
	}
	// Clean inbound exits from every other room.
	for _, id := range d.allRoomIds() {
		if id == roomId {
			continue
		}
		src := d.loadTemplate(id)
		if src == nil {
			continue
		}
		changed := false
		for name, e := range src.Exits {
			if e.RoomId == roomId {
				delete(src.Exits, name)
				changed = true
			}
		}
		if changed {
			if err := d.save(*src); err != nil {
				return buildErr("could not update room %d while deleting %d: %s", id, roomId, err.Error())
			}
		}
	}
	if err := d.deleteTemplate(roomId); err != nil {
		return buildErr("could not delete room %d: %s", roomId, err.Error())
	}
	return BuildResult{Ok: true, RoomId: roomId}
}

// buildRoomGet builds the full editable detail for a room (the inspector's
// data source). Exits are classified spatial (Dir set) vs portal by whether
// the exit key is a compass direction.
func buildRoomGet(d buildDeps, roomId int) (buildRoomDetail, bool) {
	r := d.loadTemplate(roomId)
	if r == nil {
		return buildRoomDetail{}, false
	}
	detail := buildRoomDetail{
		RoomId: r.RoomId, Title: r.Title, Description: r.Description, Zone: r.Zone,
		Biome: r.Biome, Symbol: r.MapSymbol, Legend: r.MapLegend, Music: r.MusicFile,
		Bank: r.IsBank, Storage: r.IsStorage, Pvp: r.Pvp, CharacterRoom: r.IsCharacterRoom,
		Nouns: r.Nouns, IdleMessages: r.IdleMessages,
		Plane: r.Plane, X: r.X, Y: r.Y, Z: r.Z,
	}
	for name, e := range r.Exits {
		ed := buildExitDetail{
			Name: name, ToRoomId: e.RoomId, Secret: e.Secret, OneWay: e.OneWay,
			Lock: int(e.Lock.Difficulty), Message: e.ExitMessage,
		}
		if d.isCompass(name) {
			ed.Dir = name
		}
		detail.Exits = append(detail.Exits, ed)
	}
	return detail, true
}

// buildRoomList returns every room in zone as {id, title}, sorted by id — the
// test-spawn room picker's data source (mobs.js buildTestSpawnRow). Read-only;
// unlike sendZoneMapFull it does not crawl the mapper or care about
// reachability/foreign-room bleed, since the picker only needs a flat,
// existence-guaranteed list of rooms the admin can choose to spawn into.
func buildRoomList(zone string) []buildRoomListRow {
	out := []buildRoomListRow{}
	for _, id := range rooms.GetAllRoomIds() {
		r := rooms.LoadRoom(id)
		if r == nil || r.Zone != zone {
			continue
		}
		out = append(out, buildRoomListRow{Id: r.RoomId, Title: r.Title})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Id < out[j].Id })
	return out
}

// buildExitAdd adds an exit from req.RoomId. A spatial (compass-direction) exit
// must lead to the adjacent same-plane cell (or into a non-Euclidean plane) and
// auto-wires a reciprocal unless one-way; crossing between two Euclidean planes
// is rejected (use a portal). A portal exit is a named, non-spatial door to any
// room on any plane, with no auto-reciprocal.
func buildExitAdd(d buildDeps, req exitAddReq) BuildResult {
	from := d.loadTemplate(req.RoomId)
	if from == nil {
		return buildErr("room %d not found", req.RoomId)
	}
	to := d.loadTemplate(req.ToRoomId)
	if to == nil {
		return buildErr("destination room %d not found", req.ToRoomId)
	}

	ex := exit.RoomExit{
		RoomId:      req.ToRoomId,
		Secret:      req.Secret,
		OneWay:      req.OneWay,
		ExitMessage: req.Message,
	}
	if req.Lock > 0 {
		ex.Lock = gamelock.Lock{Difficulty: uint8(req.Lock)}
	}

	spatial := req.Dir != ""
	if spatial {
		dir := req.Dir
		if !d.isCompass(dir) {
			return buildErr("%q is not a spatial direction; name it as a portal instead", dir)
		}
		crossPlane := from.Plane != to.Plane
		if crossPlane && !d.isNonEuclidean(from.Plane) && !d.isNonEuclidean(to.Plane) {
			return buildErr("a spatial exit can't cross between Euclidean planes; use a portal/door")
		}
		if !crossPlane {
			dx, dy, dz := d.delta(dir)
			if to.X != from.X+dx || to.Y != from.Y+dy || to.Z != from.Z+dz {
				return buildErr("room %d is not directly %s of room %d; use a portal for non-adjacent links", req.ToRoomId, dir, req.RoomId)
			}
		}
		if r := applyExit(d, req.RoomId, dir, ex); !r.Ok {
			return r
		}
		if !req.OneWay {
			recip := d.reciprocal(dir)
			if recip == "" {
				return buildErr("no reciprocal direction for %q", dir)
			}
			// Reciprocal mirrors only secrecy; message/lock are directional.
			if r := applyExit(d, req.ToRoomId, recip, exit.RoomExit{RoomId: req.RoomId, Secret: req.Secret}); !r.Ok {
				return r
			}
		}
		return BuildResult{Ok: true, RoomId: req.RoomId}
	}

	// Portal / named door.
	name := strings.TrimSpace(req.PortalName)
	if name == "" {
		return buildErr("an exit needs either a spatial direction or a portal name")
	}
	if d.isCompass(name) {
		return buildErr("%q is a compass direction; add it as a spatial exit instead", name)
	}
	if r := applyExit(d, req.RoomId, name, ex); !r.Ok {
		return r
	}
	return BuildResult{Ok: true, RoomId: req.RoomId}
}

// buildExitRemove removes exitName from roomId. For a spatial exit it also
// removes the reciprocal on the destination.
func buildExitRemove(d buildDeps, roomId int, exitName string) BuildResult {
	from := d.loadTemplate(roomId)
	if from == nil {
		return buildErr("room %d not found", roomId)
	}
	e, ok := from.Exits[exitName]
	if !ok {
		return buildErr("room %d has no exit %q", roomId, exitName)
	}
	delete(from.Exits, exitName)
	if err := d.save(*from); err != nil {
		return buildErr("could not save room %d: %s", roomId, err.Error())
	}

	// Remove the matching reciprocal on the destination, if this was a spatial
	// exit whose return leg points back here.
	if recip := d.reciprocal(exitName); recip != "" && !e.OneWay {
		if to := d.loadTemplate(e.RoomId); to != nil {
			if back, ok := to.Exits[recip]; ok && back.RoomId == roomId {
				delete(to.Exits, recip)
				if err := d.save(*to); err != nil {
					return buildErr("could not save room %d: %s", e.RoomId, err.Error())
				}
			}
		}
	}
	return BuildResult{Ok: true, RoomId: roomId}
}

// applyExit loads a room template, sets one exit, and re-saves — the single
// persistence path for every exit write so both legs of a link survive.
func applyExit(d buildDeps, roomId int, exitName string, ex exit.RoomExit) BuildResult {
	r := d.loadTemplate(roomId)
	if r == nil {
		return buildErr("room %d not found", roomId)
	}
	if r.Exits == nil {
		r.Exits = map[string]exit.RoomExit{}
	}
	r.Exits[exitName] = ex
	if err := d.save(*r); err != nil {
		return buildErr("could not save room %d: %s", roomId, err.Error())
	}
	return BuildResult{Ok: true, RoomId: roomId}
}

// ---- server -> client senders ------------------------------------------------

// sendBuildResult emits a Build.Result to the admin.
func sendBuildResult(uid int, res BuildResult) {
	events.AddToQueue(GMCPOut{UserId: uid, Module: "Build.Result", Payload: res})
}

// sendRoomDetail emits a Build.Room (full editable detail) to the admin, with
// the valid biome list attached for the inspector's dropdown.
func sendRoomDetail(uid int, detail buildRoomDetail) {
	detail.Biomes = validBiomeIds()
	events.AddToQueue(GMCPOut{UserId: uid, Module: "Build.Room", Payload: detail})
}

// sendRoomList emits a Build.Rooms to the admin (the test-spawn room picker).
func sendRoomList(uid int, zone string) {
	events.AddToQueue(GMCPOut{UserId: uid, Module: "Build.Rooms", Payload: buildRoomList(zone)})
}

func validBiomeIds() []string {
	bs := rooms.GetAllBiomes()
	out := make([]string, 0, len(bs))
	for _, b := range bs {
		out = append(out, b.BiomeId)
	}
	sort.Strings(out)
	return out
}

// sendZoneMapFull rebuilds the given zone's mapper and pushes an UNFOGGED
// Zone.Map (every room in the zone, all planes) to the admin — the builder must
// see and edit unexplored rooms, unlike the fog-of-war play mapper. Rooms are
// scoped to the target zone so cross-zone crawl spillover isn't rendered.
func sendZoneMapFull(uid int, zone string) {
	rootId, err := rooms.GetZoneRoot(zone)
	if err != nil {
		return
	}
	m := mapper.GetMapper(rootId, true) // force refresh so new rooms/exits show
	if m == nil {
		return
	}
	crawled := m.CrawledRoomIds()

	// Pass 1: the current zone's rooms + their per-plane bounding box.
	type bbox struct{ minX, maxX, minY, maxY int }
	boxes := map[int]*bbox{}
	visited := map[int]struct{}{}
	for _, id := range crawled {
		r := rooms.LoadRoom(id)
		if r == nil || r.Zone != zone {
			continue
		}
		visited[id] = struct{}{}
		if b := boxes[r.Plane]; b == nil {
			boxes[r.Plane] = &bbox{r.X, r.X, r.Y, r.Y}
		} else {
			b.minX, b.maxX = min(b.minX, r.X), max(b.maxX, r.X)
			b.minY, b.maxY = min(b.minY, r.Y), max(b.maxY, r.Y)
		}
	}

	// Pass 2: foreign rooms (other zones, same plane) within the zone's bounding
	// box plus a small margin — added as dimmed context so cross-zone overlaps
	// are visible and ghost cells don't invite building into an occupied
	// neighbour. Same-plane zones share one coordinate frame (the 0.15.0
	// migration keeps a connected component on one plane), so this is exactly
	// the set ValidatePlacement would reject.
	const margin = 2
	for _, id := range crawled {
		if _, ok := visited[id]; ok {
			continue
		}
		r := rooms.LoadRoom(id)
		if r == nil || r.Zone == zone {
			continue
		}
		b := boxes[r.Plane]
		if b == nil {
			continue // current zone doesn't occupy this plane
		}
		if r.X < b.minX-margin || r.X > b.maxX+margin || r.Y < b.minY-margin || r.Y > b.maxY+margin {
			continue
		}
		visited[id] = struct{}{}
	}

	snap := m.Snapshot(visited)
	for i := range snap {
		if r := rooms.LoadRoom(snap[i].RoomId); r != nil && r.Zone != zone {
			snap[i].Foreign = true
			snap[i].OwnerZone = r.Zone
		}
	}

	zoneNames := rooms.GetAllZoneNames()
	sort.Strings(zoneNames)

	payload := GMCPZoneModule_Payload{
		Zone:   zone,
		Rooms:  snap,
		Zones:  zoneNames,       // let the builder switch to any zone
		Biomes: validBiomeIds(), // for the new-zone / inspector biome dropdowns
	}
	events.AddToQueue(GMCPOut{UserId: uid, Module: "Zone.Map", Payload: payload})
}

// zoneForUser returns the zone name of the admin's current room, or "".
func zoneForUser(uid int) string {
	u := users.GetByUserId(uid)
	if u == nil {
		return ""
	}
	if r := rooms.LoadRoom(u.Character.RoomId); r != nil {
		return r.Zone
	}
	return ""
}

// zoneOfRoom returns the zone a room belongs to, falling back to the admin's
// current zone. The builder can view/edit a zone other than the one the admin
// is standing in, so refreshes must target the EDITED room's zone.
func zoneOfRoom(roomId, uid int) string {
	if r := rooms.LoadRoom(roomId); r != nil && r.Zone != "" {
		return r.Zone
	}
	return zoneForUser(uid)
}
