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
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/gamelock"
	"github.com/GoMudEngine/GoMud/internal/mapper"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// BuildResult is echoed to the client after every mutation (as Build.Result).
type BuildResult struct {
	Ok     bool   `json:"ok"`
	Error  string `json:"error,omitempty"`
	RoomId int    `json:"roomId,omitempty"` // e.g. the newly created room, or the affected room
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

// compassOpposite is the deterministic reciprocal for each spatial direction.
// We deliberately do NOT use mapper.GetReciprocalExit here: it scans posDeltas
// for a matching inverse delta and returns whichever the (randomly-ordered) map
// iteration hits first, so for e.g. "north" it can return "south-gap" instead
// of "south". Spatial builder exits are always plain compass names, so a fixed
// table is both correct and stable.
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
	nr.Title = "Untitled"
	nr.Description = ""
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
	visited := map[int]struct{}{}
	for _, id := range m.CrawledRoomIds() {
		if r := rooms.LoadRoom(id); r != nil && r.Zone == zone {
			visited[id] = struct{}{}
		}
	}
	payload := GMCPZoneModule_Payload{
		Zone:  zone,
		Rooms: m.Snapshot(visited),
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
