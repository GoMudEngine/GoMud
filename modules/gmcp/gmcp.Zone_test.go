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
		cfgs:     map[string]*rooms.ZoneConfig{"Testzone": {Name: "Testzone", RoomId: 100}},
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
