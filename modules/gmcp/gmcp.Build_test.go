package gmcp

import (
	"fmt"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/mapper"
	"github.com/GoMudEngine/GoMud/internal/mutators"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// fakeWorld is an in-memory stand-in for the rooms package so the Build.*
// core logic can be unit-tested without a live world or disk I/O.
type fakeWorld struct {
	rooms     map[int]*rooms.Room
	saved     []rooms.Room
	deleted   []int
	nextId    int
	occupied  map[[4]int]int // (plane,x,y,z) -> roomId
	nonEuclid map[int]bool
}

func newFakeWorld() *fakeWorld {
	return &fakeWorld{
		rooms:     map[int]*rooms.Room{},
		occupied:  map[[4]int]int{},
		nonEuclid: map[int]bool{},
		nextId:    1000,
	}
}

func (w *fakeWorld) put(r *rooms.Room) {
	if r.Exits == nil {
		r.Exits = map[string]exit.RoomExit{}
	}
	w.rooms[r.RoomId] = r
	w.occupied[[4]int{r.Plane, r.X, r.Y, r.Z}] = r.RoomId
}

func (w *fakeWorld) deps() buildDeps {
	return buildDeps{
		loadTemplate: func(id int) *rooms.Room { return w.rooms[id] },
		save: func(r rooms.Room) error {
			w.saved = append(w.saved, r)
			cp := r
			w.rooms[r.RoomId] = &cp
			w.occupied[[4]int{r.Plane, r.X, r.Y, r.Z}] = r.RoomId
			return nil
		},
		deleteTemplate: func(id int) error {
			w.deleted = append(w.deleted, id)
			delete(w.rooms, id)
			return nil
		},
		validatePlace: func(plane, x, y, z, exclude int) error {
			if w.nonEuclid[plane] {
				return nil
			}
			if id, ok := w.occupied[[4]int{plane, x, y, z}]; ok && id != exclude {
				return fmt.Errorf("cell (plane %d, %d,%d,%d) already occupied by room %d", plane, x, y, z, id)
			}
			return nil
		},
		newRoom: func(zone string) *rooms.Room {
			w.nextId++
			return &rooms.Room{RoomId: w.nextId, Zone: zone, Exits: map[string]exit.RoomExit{}}
		},
		allRoomIds: func() []int {
			ids := make([]int, 0, len(w.rooms))
			for id := range w.rooms {
				ids = append(ids, id)
			}
			return ids
		},
		reciprocal:     reciprocalCompass,         // deterministic compass inverse
		delta:          mapper.GetDelta,           // real (pure)
		isCompass:      mapper.IsCompassDirection, // real (pure)
		isNonEuclidean: func(plane int) bool { return w.nonEuclid[plane] },
		// The registries are empty in unit tests; the fake says every
		// reference exists so spawn tests exercise the POLICY, not data loading.
		mobExists:  func(int) bool { return true },
		itemExists: func(int) bool { return true },
		buffExists: func(int) bool { return true },
	}
}

func TestRoleIsAdmin(t *testing.T) {
	if !roleIsAdmin(&users.UserRecord{Role: users.RoleAdmin}) {
		t.Error("RoleAdmin user should pass the build gate")
	}
	if roleIsAdmin(&users.UserRecord{Role: "user"}) {
		t.Error("non-admin user must be refused")
	}
	if roleIsAdmin(nil) {
		t.Error("nil user must be refused")
	}
}

func TestBuildRoomCreate_RejectsOccupiedCell(t *testing.T) {
	w := newFakeWorld()
	w.put(&rooms.Room{RoomId: 1, Zone: "Testville", Plane: 0, X: 0, Y: 0, Z: 0})
	// A room already sits north of room 1 (0,-1,0).
	w.put(&rooms.Room{RoomId: 2, Zone: "Testville", Plane: 0, X: 0, Y: -1, Z: 0})

	res := buildRoomCreate(w.deps(), 1, "north", 0, 0, -1, 0)
	if res.Ok {
		t.Fatalf("create into an occupied cell must fail, got %+v", res)
	}
	if len(w.saved) != 0 {
		t.Errorf("no room should be saved when placement is rejected, saved=%d", len(w.saved))
	}
}

func TestBuildRoomCreate_WiresReciprocalExits(t *testing.T) {
	w := newFakeWorld()
	w.put(&rooms.Room{RoomId: 1, Zone: "Testville", Plane: 0, X: 0, Y: 0, Z: 0})

	res := buildRoomCreate(w.deps(), 1, "north", 0, 0, -1, 0)
	if !res.Ok {
		t.Fatalf("create should succeed, got %+v", res)
	}
	newId := res.RoomId
	if newId == 0 {
		t.Fatal("create must return the new room id")
	}

	nr := w.rooms[newId]
	if nr == nil {
		t.Fatal("new room not saved")
	}
	if nr.X != 0 || nr.Y != -1 || nr.Z != 0 || nr.Plane != 0 {
		t.Errorf("new room coords wrong: %+v", nr)
	}
	// Forward exit on room 1, reciprocal exit on the new room.
	if e, ok := w.rooms[1].Exits["north"]; !ok || e.RoomId != newId {
		t.Errorf("room 1 should have a north exit to %d, got %+v", newId, w.rooms[1].Exits)
	}
	if e, ok := nr.Exits["south"]; !ok || e.RoomId != 1 {
		t.Errorf("new room should have a south exit back to 1, got %+v", nr.Exits)
	}
	// A created room must be valid on reload — Room.Validate() rejects blank
	// title/description, which would brick the room's own Get/Update.
	if nr.Title == "" || nr.Description == "" {
		t.Errorf("new room must have non-empty title AND description, got title=%q desc=%q", nr.Title, nr.Description)
	}
	if err := nr.Validate(); err != nil {
		t.Errorf("new room should pass Room.Validate(), got: %v", err)
	}
}

func TestBuildRoomUpdate_RejectsEmptyTitleOrDescription(t *testing.T) {
	w := newFakeWorld()
	w.put(&rooms.Room{RoomId: 5, Zone: "Testville", Title: "Old", Description: "Old desc."})

	if res := buildRoomUpdate(w.deps(), roomUpdateReq{RoomId: 5, Title: "", Description: "d"}); res.Ok {
		t.Error("empty title must be rejected")
	}
	if res := buildRoomUpdate(w.deps(), roomUpdateReq{RoomId: 5, Title: "t", Description: "   "}); res.Ok {
		t.Error("blank description must be rejected")
	}
	if len(w.saved) != 0 {
		t.Errorf("no save should happen when validation fails, got %d", len(w.saved))
	}
}

func TestBuildRoomUpdate_RoundTripsFieldsPreservingStructure(t *testing.T) {
	w := newFakeWorld()
	w.put(&rooms.Room{
		RoomId: 5, Zone: "Testville", Plane: 2, X: 3, Y: 4, Z: 0,
		Title: "Old", Biome: "swamp",
		Exits: map[string]exit.RoomExit{"east": {RoomId: 6}},
	})

	res := buildRoomUpdate(w.deps(), roomUpdateReq{
		RoomId: 5, Title: "New Title", Description: "A fresh desc.",
		Biome: "city", Symbol: "T", Legend: "Town", Music: "town.mp3",
		Bank: true, Pvp: true,
		Nouns:        map[string]string{"statue": "A weathered statue."},
		IdleMessages: []string{"A breeze stirs."},
	})
	if !res.Ok {
		t.Fatalf("update should succeed, got %+v", res)
	}
	if len(w.saved) != 1 {
		t.Fatalf("expected one save, got %d", len(w.saved))
	}
	got := w.saved[0]
	if got.Title != "New Title" || got.Description != "A fresh desc." || got.Biome != "city" {
		t.Errorf("core fields not round-tripped: %+v", got)
	}
	if got.MapSymbol != "T" || got.MapLegend != "Town" || got.MusicFile != "town.mp3" {
		t.Errorf("map/music fields not round-tripped: %+v", got)
	}
	if !got.IsBank || !got.Pvp {
		t.Errorf("flags not round-tripped: %+v", got)
	}
	if got.Nouns["statue"] == "" || len(got.IdleMessages) != 1 {
		t.Errorf("nouns/idle not round-tripped: %+v", got)
	}
	// Structure the edit must NOT clobber:
	if got.Plane != 2 || got.X != 3 || got.Y != 4 {
		t.Errorf("update clobbered authored coords: %+v", got)
	}
	if e, ok := got.Exits["east"]; !ok || e.RoomId != 6 {
		t.Errorf("update clobbered exits: %+v", got.Exits)
	}
}

func TestBuildExitAdd_RejectsCrossEuclideanPlaneSpatial(t *testing.T) {
	w := newFakeWorld()
	w.put(&rooms.Room{RoomId: 1, Zone: "Over", Plane: 0, X: 0, Y: 0, Z: 0})
	w.put(&rooms.Room{RoomId: 2, Zone: "Inn", Plane: 1, X: 0, Y: -1, Z: 0}) // different Euclidean plane

	res := buildExitAdd(w.deps(), exitAddReq{RoomId: 1, Dir: "north", ToRoomId: 2})
	if res.Ok {
		t.Fatalf("a spatial exit crossing Euclidean planes must be rejected, got %+v", res)
	}
}

func TestBuildExitAdd_PortalAllowsCrossPlane(t *testing.T) {
	w := newFakeWorld()
	w.put(&rooms.Room{RoomId: 1, Zone: "Over", Plane: 0, X: 0, Y: 0, Z: 0})
	w.put(&rooms.Room{RoomId: 2, Zone: "Inn", Plane: 1, X: 9, Y: 9, Z: 0})

	res := buildExitAdd(w.deps(), exitAddReq{RoomId: 1, PortalName: "enter inn", ToRoomId: 2, Message: "You step inside."})
	if !res.Ok {
		t.Fatalf("a portal to another plane must be allowed, got %+v", res)
	}
	e, ok := w.rooms[1].Exits["enter inn"]
	if !ok || e.RoomId != 2 || e.ExitMessage != "You step inside." {
		t.Errorf("portal exit not written correctly: %+v", w.rooms[1].Exits)
	}
	// Portals are non-spatial: no auto reciprocal on the target.
	if len(w.rooms[2].Exits) != 0 {
		t.Errorf("portal must not auto-wire a reciprocal, got %+v", w.rooms[2].Exits)
	}
}

func TestBuildExitAdd_SpatialWiresReciprocal(t *testing.T) {
	w := newFakeWorld()
	w.put(&rooms.Room{RoomId: 1, Zone: "Over", Plane: 0, X: 0, Y: 0, Z: 0})
	w.put(&rooms.Room{RoomId: 2, Zone: "Over", Plane: 0, X: 0, Y: -1, Z: 0}) // adjacent north

	res := buildExitAdd(w.deps(), exitAddReq{RoomId: 1, Dir: "north", ToRoomId: 2})
	if !res.Ok {
		t.Fatalf("adjacent spatial exit should succeed, got %+v", res)
	}
	if e, ok := w.rooms[1].Exits["north"]; !ok || e.RoomId != 2 {
		t.Errorf("forward exit missing: %+v", w.rooms[1].Exits)
	}
	if e, ok := w.rooms[2].Exits["south"]; !ok || e.RoomId != 1 {
		t.Errorf("reciprocal exit missing: %+v", w.rooms[2].Exits)
	}
}

func TestBuildExitAdd_RejectsNonAdjacentSpatial(t *testing.T) {
	w := newFakeWorld()
	w.put(&rooms.Room{RoomId: 1, Zone: "Over", Plane: 0, X: 0, Y: 0, Z: 0})
	w.put(&rooms.Room{RoomId: 2, Zone: "Over", Plane: 0, X: 0, Y: -5, Z: 0}) // far north, not adjacent

	res := buildExitAdd(w.deps(), exitAddReq{RoomId: 1, Dir: "north", ToRoomId: 2})
	if res.Ok {
		t.Fatalf("a spatial exit to a non-adjacent cell must be rejected (would be a wrap exit), got %+v", res)
	}
}

func TestBuildExitRemove_RemovesReciprocal(t *testing.T) {
	w := newFakeWorld()
	w.put(&rooms.Room{RoomId: 1, Zone: "Over", Plane: 0, X: 0, Y: 0, Z: 0,
		Exits: map[string]exit.RoomExit{"north": {RoomId: 2}}})
	w.put(&rooms.Room{RoomId: 2, Zone: "Over", Plane: 0, X: 0, Y: -1, Z: 0,
		Exits: map[string]exit.RoomExit{"south": {RoomId: 1}}})

	res := buildExitRemove(w.deps(), 1, "north")
	if !res.Ok {
		t.Fatalf("remove should succeed, got %+v", res)
	}
	if _, ok := w.rooms[1].Exits["north"]; ok {
		t.Error("forward exit not removed")
	}
	if _, ok := w.rooms[2].Exits["south"]; ok {
		t.Error("reciprocal exit not removed")
	}
}

func TestBuildRoomGet_MapsFieldsAndClassifiesExits(t *testing.T) {
	w := newFakeWorld()
	w.put(&rooms.Room{
		RoomId: 7, Zone: "Over", Plane: 0, X: 1, Y: 2, Z: 0,
		Title: "Hall", Description: "A hall.", Biome: "city",
		MapSymbol: "H", MapLegend: "Hall", MusicFile: "hall.mp3",
		IsBank: true, Pvp: true,
		Nouns:        map[string]string{"banner": "A red banner."},
		IdleMessages: []string{"Torches gutter."},
		Exits: map[string]exit.RoomExit{
			"north":     {RoomId: 8},
			"enter inn": {RoomId: 9, ExitMessage: "You duck inside.", Secret: true},
		},
	})

	detail, ok := buildRoomGet(w.deps(), 7)
	if !ok {
		t.Fatal("expected room 7 to be found")
	}
	if detail.Title != "Hall" || detail.Biome != "city" || detail.Symbol != "H" || detail.Music != "hall.mp3" {
		t.Errorf("fields not mapped: %+v", detail)
	}
	if !detail.Bank || !detail.Pvp {
		t.Errorf("flags not mapped: %+v", detail)
	}
	if detail.Plane != 0 || detail.X != 1 || detail.Y != 2 {
		t.Errorf("coords not mapped: %+v", detail)
	}
	byName := map[string]buildExitDetail{}
	for _, e := range detail.Exits {
		byName[e.Name] = e
	}
	if e := byName["north"]; e.Dir != "north" || e.ToRoomId != 8 {
		t.Errorf("spatial exit misclassified: %+v", e)
	}
	if e := byName["enter inn"]; e.Dir != "" || e.ToRoomId != 9 || e.Message != "You duck inside." || !e.Secret {
		t.Errorf("portal exit misclassified: %+v", e)
	}
}

func TestBuildRoomGet_CarriesSpawnList(t *testing.T) {
	w := newFakeWorld()
	r := &rooms.Room{RoomId: 100, Title: "T", Description: "D"}
	r.SpawnInfo = []rooms.SpawnInfo{
		{MobId: 336, RespawnRate: "5 real minutes", Message: "A guard arrives."},
		{ItemId: 40001, Container: "chest"},
	}
	w.rooms[100] = r

	d, ok := buildRoomGet(w.deps(), 100)
	if !ok {
		t.Fatal("expected room detail")
	}
	if len(d.Spawns) != 2 {
		t.Fatalf("expected 2 spawns, got %d", len(d.Spawns))
	}
	if d.Spawns[0].MobId != 336 || d.Spawns[0].RespawnRate != "5 real minutes" {
		t.Errorf("mob spawn not mapped: %+v", d.Spawns[0])
	}
	if d.Spawns[1].ItemId != 40001 || d.Spawns[1].Container != "chest" {
		t.Errorf("item spawn not mapped: %+v", d.Spawns[1])
	}
}

func TestBuildRoomDelete_CleansInboundExits(t *testing.T) {
	w := newFakeWorld()
	w.put(&rooms.Room{RoomId: 1, Zone: "Over", Plane: 0, X: 0, Y: 0, Z: 0,
		Exits: map[string]exit.RoomExit{"north": {RoomId: 2}}})
	w.put(&rooms.Room{RoomId: 2, Zone: "Over", Plane: 0, X: 0, Y: -1, Z: 0,
		Exits: map[string]exit.RoomExit{"south": {RoomId: 1}}})

	res := buildRoomDelete(w.deps(), 2)
	if !res.Ok {
		t.Fatalf("delete should succeed, got %+v", res)
	}
	if _, ok := w.rooms[2]; ok {
		t.Error("room 2 should be deleted")
	}
	if _, ok := w.rooms[1].Exits["north"]; ok {
		t.Error("inbound exit from room 1 should be cleaned")
	}
}

func TestBuildRoomUpdate_SavesSpawnsAndValidates(t *testing.T) {
	w := newFakeWorld()
	r := &rooms.Room{RoomId: 100, Title: "T", Description: "D"}
	r.Containers = map[string]rooms.Container{"chest": {}}
	w.rooms[100] = r

	base := func() roomUpdateReq {
		return roomUpdateReq{RoomId: 100, Title: "T", Description: "D"}
	}

	// Valid list saves and preserves order.
	req := base()
	req.Spawns = []rooms.SpawnInfo{{MobId: 1}, {ItemId: 2, Container: "chest"}}
	res := buildRoomUpdate(w.deps(), req)
	if !res.Ok {
		t.Fatalf("valid spawn list should save: %+v", res)
	}
	saved := w.saved[len(w.saved)-1]
	if len(saved.SpawnInfo) != 2 || saved.SpawnInfo[0].MobId != 1 || saved.SpawnInfo[1].ItemId != 2 {
		t.Errorf("spawn list not saved in order: %+v", saved.SpawnInfo)
	}

	// A contradictory entry is refused and nothing is saved.
	before := len(w.saved)
	bad := base()
	bad.Spawns = []rooms.SpawnInfo{{MobId: 1, ItemId: 2}}
	if res := buildRoomUpdate(w.deps(), bad); res.Ok {
		t.Error("an entry spawning both a mob and an item must be rejected")
	}
	if len(w.saved) != before {
		t.Error("nothing may be saved when a spawn entry is invalid")
	}

	// A container that is not in this room is refused.
	bad2 := base()
	bad2.Spawns = []rooms.SpawnInfo{{ItemId: 2, Container: "barrel"}}
	if res := buildRoomUpdate(w.deps(), bad2); res.Ok {
		t.Error("a container absent from the room must be rejected")
	}
}

// InstanceId/DespawnedRound are runtime tracking the client never holds
// meaningfully. A save must not let a client blank them, or the engine loses
// track of an already-spawned mob and spawns a duplicate.
func TestBuildRoomUpdate_PreservesSpawnRuntimeTracking(t *testing.T) {
	w := newFakeWorld()
	r := &rooms.Room{RoomId: 100, Title: "T", Description: "D"}
	r.SpawnInfo = []rooms.SpawnInfo{{MobId: 1, InstanceId: 4242, DespawnedRound: 99}}
	w.rooms[100] = r

	req := roomUpdateReq{RoomId: 100, Title: "T", Description: "D",
		Spawns: []rooms.SpawnInfo{{MobId: 1}}} // client sends no tracking

	if res := buildRoomUpdate(w.deps(), req); !res.Ok {
		t.Fatalf("update should succeed: %+v", res)
	}
	saved := w.saved[len(w.saved)-1]
	if saved.SpawnInfo[0].InstanceId != 4242 || saved.SpawnInfo[0].DespawnedRound != 99 {
		t.Errorf("runtime tracking must be preserved from the template, got %+v", saved.SpawnInfo[0])
	}
}

// A room save must never be the reason authored content disappears.
//
// On 2026-07-25 a live builder save of room 468 lost a sanctuary mutator and a
// noun. The loss could not be reproduced on any constructed path — fake world,
// real disk, or full world with the room prepared — so rather than trust that
// reasoning, buildRoomUpdate now asserts that fields it does not manage are
// unchanged between load and save, and this pins the behaviour.
func TestBuildRoomUpdate_PreservesUnmanagedContent(t *testing.T) {
	w := newFakeWorld()
	r := &rooms.Room{RoomId: 468, Title: "Temple", Description: "D"}
	r.Mutators = mutators.MutatorList{{MutatorId: "sanctuary"}}
	r.Containers = map[string]rooms.Container{"chest": {}}
	r.Nouns = map[string]string{"altar": "a", "pillars": "p", "candles": "c"}
	w.rooms[468] = r

	req := roomUpdateReq{
		RoomId: 468, Title: "Temple", Description: "D",
		Nouns: map[string]string{"altar": "a", "pillars": "p", "candles": "c"},
	}
	if res := buildRoomUpdate(w.deps(), req); !res.Ok {
		t.Fatalf("save should succeed: %+v", res)
	}
	saved := w.saved[len(w.saved)-1]

	if len(saved.Mutators) != 1 {
		t.Errorf("mutators must survive a room save: 1 -> %d", len(saved.Mutators))
	}
	if len(saved.Containers) != 1 {
		t.Errorf("containers must survive a room save: 1 -> %d", len(saved.Containers))
	}
	if len(saved.Nouns) != 3 {
		t.Errorf("all nouns must survive when the client sends them: 3 -> %d", len(saved.Nouns))
	}
}
