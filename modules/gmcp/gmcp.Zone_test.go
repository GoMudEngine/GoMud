package gmcp

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/rooms"
)

type fakeZoneWorld struct {
	cfgs     map[string]*rooms.ZoneConfig
	blockers map[string][]rooms.ZoneBlocker
	deleted  []string
	saved    []rooms.ZoneConfig
}

func newFakeZoneWorld() *fakeZoneWorld {
	return &fakeZoneWorld{
		// RoomIds mirrors the live engine, which populates this per-zone lookup
		// as rooms load. buildZoneList reads RoomCount from it rather than
		// rescanning every room in the world.
		cfgs: map[string]*rooms.ZoneConfig{
			"Testzone": {Name: "Testzone", RoomId: 100, RoomIds: map[int]struct{}{100: {}}},
		},
		blockers: map[string][]rooms.ZoneBlocker{},
	}
}

func (w *fakeZoneWorld) deps() zoneDeps {
	return zoneDeps{
		load: func(z string) *rooms.ZoneConfig { return w.cfgs[z] },
		save: func(c rooms.ZoneConfig) error {
			w.saved = append(w.saved, c)
			cp := c
			w.cfgs[c.Name] = &cp
			return nil
		},
		del: func(z string) error {
			w.deleted = append(w.deleted, z)
			delete(w.cfgs, z)
			return nil
		},
		blockers:  func(z string) []rooms.ZoneBlocker { return w.blockers[z] },
		zoneNames: func() []string { return []string{"Testzone"} },
		roomIds:   func(z string) []int { return []int{100} },
	}
}

func TestBuildZoneDelete_BlocksWhenNotEmpty(t *testing.T) {
	w := newFakeZoneWorld()
	w.blockers["Testzone"] = []rooms.ZoneBlocker{
		{Kind: "room", Id: "room 101"},
		{Kind: "inbound-exit", Id: "room 900 (Other) east"},
	}

	res := buildZoneDelete(w.deps(), "Testzone")

	if res.Ok {
		t.Fatal("delete must be refused when the zone is not empty")
	}
	if len(res.ZoneRefs) != 2 {
		t.Errorf("expected 2 blockers surfaced, got %d", len(res.ZoneRefs))
	}
	if len(w.deleted) != 0 {
		t.Error("d.del must not be called when blocked")
	}
}

func TestBuildZoneDelete_DeletesCleanZone(t *testing.T) {
	w := newFakeZoneWorld()
	res := buildZoneDelete(w.deps(), "Testzone")
	if !res.Ok {
		t.Fatalf("clean zone should delete, got %+v", res)
	}
	if len(w.deleted) != 1 || w.deleted[0] != "Testzone" {
		t.Errorf("expected Testzone deleted, got %v", w.deleted)
	}
}

func TestBuildZoneDelete_UnknownZone(t *testing.T) {
	w := newFakeZoneWorld()
	if res := buildZoneDelete(w.deps(), "Nowhere"); res.Ok {
		t.Error("unknown zone must not report success")
	}
}

func TestBuildZoneList_ReportsRoomCounts(t *testing.T) {
	w := newFakeZoneWorld()
	rowsAny := buildZoneList(w.deps())
	if len(rowsAny.Zones) != 1 {
		t.Fatalf("expected 1 zone, got %d", len(rowsAny.Zones))
	}
	row := rowsAny.Zones[0]
	if row.Zone != "Testzone" || row.RoomCount != 1 {
		t.Errorf("unexpected row: %+v", row)
	}
}

func TestBuildZoneGet_MapsFieldsAndEnums(t *testing.T) {
	w := newFakeZoneWorld()
	w.cfgs["Testzone"] = &rooms.ZoneConfig{
		Name: "Testzone", RoomId: 100, DefaultBiome: "land",
		Region: "Windward Marches", Instanced: true, DeathPolicy: "ejected",
		NonCartesian: true, DefaultPlane: 7,
	}
	d, ok := buildZoneGet(w.deps(), "Testzone")
	if !ok {
		t.Fatal("expected zone detail")
	}
	if d.Name != "Testzone" || d.DefaultBiome != "land" || !d.Instanced ||
		d.DeathPolicy != "ejected" || !d.NonCartesian || d.DefaultPlane != 7 {
		t.Errorf("fields not mapped: %+v", d)
	}
	if len(d.Enums.DeathPolicies) == 0 {
		t.Error("death policy enum must be server-supplied")
	}
}

func TestBuildZoneGet_UnknownZone(t *testing.T) {
	w := newFakeZoneWorld()
	if _, ok := buildZoneGet(w.deps(), "Nowhere"); ok {
		t.Error("unknown zone must not return detail")
	}
}

func baseZoneReq() zoneUpdateReq {
	return zoneUpdateReq{
		Name: "Testzone", RoomId: 100, DefaultBiome: "land",
		DeathPolicy: "rejoin", DefaultPlane: 7,
	}
}

// A non_cartesian zone on plane 0 marks plane 0 (the overworld) non-Euclidean
// via the sticky OR in PlaneRegistry.Mark, disabling collision and reciprocity
// enforcement for the ENTIRE world. Refuse to save that combination.
func TestBuildZoneUpdate_RejectsNonCartesianOnOverworldPlane(t *testing.T) {
	w := newFakeZoneWorld()
	req := baseZoneReq()
	req.NonCartesian = true
	req.DefaultPlane = 0

	if res := buildZoneUpdate(w.deps(), req); res.Ok {
		t.Error("non_cartesian with plane 0 must be rejected")
	}
	if len(w.saved) != 0 {
		t.Errorf("nothing may be saved on validation failure, got %d", len(w.saved))
	}
}

func TestBuildZoneUpdate_AllowsNonCartesianOnOwnPlane(t *testing.T) {
	w := newFakeZoneWorld()
	req := baseZoneReq()
	req.NonCartesian = true
	req.DefaultPlane = 7

	if res := buildZoneUpdate(w.deps(), req); !res.Ok {
		t.Errorf("non_cartesian on its own plane should save: %+v", res)
	}
}

func TestBuildZoneUpdate_RejectsUnknownDeathPolicy(t *testing.T) {
	w := newFakeZoneWorld()
	req := baseZoneReq()
	req.DeathPolicy = "vaporize"
	if res := buildZoneUpdate(w.deps(), req); res.Ok {
		t.Error("unknown death policy must be rejected")
	}
}

func TestBuildZoneUpdate_RoundTripsFields(t *testing.T) {
	w := newFakeZoneWorld()
	req := baseZoneReq()
	req.Region = "Windward Marches"
	req.MusicFile = "theme.mp3"
	req.Instanced = true
	req.EntryRoom = 100

	if res := buildZoneUpdate(w.deps(), req); !res.Ok {
		t.Fatalf("update should succeed: %+v", res)
	}
	got := w.saved[0]
	if got.Region != "Windward Marches" || got.MusicFile != "theme.mp3" ||
		!got.Instanced || got.EntryRoom != 100 {
		t.Errorf("fields not round-tripped: %+v", got)
	}
}

func TestBuildZoneUpdate_RejectsEntryRoomOutsideZone(t *testing.T) {
	w := newFakeZoneWorld() // roomIds returns only [100]
	req := baseZoneReq()
	req.Instanced = true
	req.EntryRoom = 999
	if res := buildZoneUpdate(w.deps(), req); res.Ok {
		t.Error("entry room outside the zone must be rejected")
	}
}

// buildZoneList used to derive RoomCount from the roomIds dep, whose real
// wiring rescans EVERY room in the world with a disk read per room — once per
// zone. Measured on the live world: 49 zones x 1384 rooms = 67,816
// LoadRoomTemplate calls, ~10s, for a list the author expects instantly.
//
// ZoneConfig.RoomIds is the engine's own per-zone room lookup, kept current as
// rooms load and only pruned by a genuine delete (idle-room unloading does not
// touch it). The count must come from there, and the expensive scan must not
// run at all.
func TestBuildZoneList_DoesNotRescanEveryRoom(t *testing.T) {
	w := newFakeZoneWorld()
	w.cfgs["Testzone"] = &rooms.ZoneConfig{
		Name:   "Testzone",
		RoomId: 100,
		RoomIds: map[int]struct{}{
			100: {}, 101: {}, 102: {}, 103: {},
		},
	}

	d := w.deps()
	roomIdsCalls := 0
	d.roomIds = func(z string) []int {
		roomIdsCalls++
		return []int{100} // deliberately wrong count; must not be used
	}

	got := buildZoneList(d)

	if len(got.Zones) != 1 {
		t.Fatalf("expected 1 zone, got %d", len(got.Zones))
	}
	if got.Zones[0].RoomCount != 4 {
		t.Errorf("RoomCount = %d, want 4 (from ZoneConfig.RoomIds)", got.Zones[0].RoomCount)
	}
	if roomIdsCalls != 0 {
		t.Errorf("buildZoneList called the full-room-scan dep %d times; it must not call it at all", roomIdsCalls)
	}
}
