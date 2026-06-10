package crawler

import (
	"reflect"
	"testing"

	"github.com/GoMudEngine/GoMud/modules/weather/sim"
)

func TestBuild_BasicAdjacency(t *testing.T) {
	f := newFakeReader()
	// Zone A: room 1 (1->2 internal), room 2 (2->1 internal, 2->3 crosses to B)
	f.addRoom("A", "plains", 1, true, ExitView{ToRoom: 2})
	f.addRoom("A", "plains", 2, true, ExitView{ToRoom: 1}, ExitView{ToRoom: 3})
	// Zone B: room 3 (3->2 crosses back to A)
	f.addRoom("B", "forest", 3, true, ExitView{ToRoom: 2})

	g, err := Build(f, Options{IncludeSecretExits: true})
	if err != nil {
		t.Fatal(err)
	}

	if len(g.Nodes) != 2 {
		t.Fatalf("want 2 nodes, got %d", len(g.Nodes))
	}
	if len(g.Edges) != 1 {
		t.Fatalf("want 1 edge, got %d: %+v", len(g.Edges), g.Edges)
	}
	e := g.Edges[0]
	if e.A != "A" || e.B != "B" {
		t.Errorf("want canonical edge A-B, got %s-%s", e.A, e.B)
	}
	// 2->3 (A->B) and 3->2 (B->A) are two distinct crossing exits.
	if e.Weight != 2 {
		t.Errorf("want weight 2, got %d", e.Weight)
	}
}

func TestBuild_UnknownExitTargetIgnored(t *testing.T) {
	f := newFakeReader()
	// Room 1 in A exits to room 99, which belongs to no known zone.
	f.addRoom("A", "plains", 1, true, ExitView{ToRoom: 99})

	g, err := Build(f, Options{IncludeSecretExits: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Edges) != 0 {
		t.Errorf("dangling exit target should produce no edge, got %+v", g.Edges)
	}
}

func TestBuild_SecretExitsOption(t *testing.T) {
	makeWorld := func() *fakeReader {
		f := newFakeReader()
		// One normal crossing exit A(1)->B(2) and one secret crossing exit
		// A(1)->B(3).
		f.addRoom("A", "plains", 1, true,
			ExitView{ToRoom: 2},
			ExitView{ToRoom: 3, Secret: true},
		)
		f.addRoom("B", "forest", 2, true)
		f.addRoom("B", "forest", 3, true)
		return f
	}

	// Included: both exits count -> weight 2.
	g, _ := Build(makeWorld(), Options{IncludeSecretExits: true})
	if len(g.Edges) != 1 || g.Edges[0].Weight != 2 {
		t.Fatalf("with secrets included want one edge weight 2, got %+v", g.Edges)
	}

	// Excluded: only the normal exit counts -> weight 1.
	g, _ = Build(makeWorld(), Options{IncludeSecretExits: false})
	if len(g.Edges) != 1 || g.Edges[0].Weight != 1 {
		t.Fatalf("with secrets excluded want one edge weight 1, got %+v", g.Edges)
	}
}

func TestBuild_ExcludeZonePatterns(t *testing.T) {
	f := newFakeReader()
	f.addRoom("Town", "city", 1, true, ExitView{ToRoom: 2})
	f.addRoom("instance_jail", "city", 2, false, ExitView{ToRoom: 1})

	g, err := Build(f, Options{
		IncludeSecretExits:  true,
		ExcludeZonePatterns: []string{"instance_*"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := g.Nodes["instance_jail"]; ok {
		t.Errorf("excluded zone should not appear as a node")
	}
	if len(g.Nodes) != 1 {
		t.Errorf("want 1 node after exclusion, got %d", len(g.Nodes))
	}
	if len(g.Edges) != 0 {
		t.Errorf("edge to an excluded zone should be dropped, got %+v", g.Edges)
	}
}

func TestBuild_NodeMetadata(t *testing.T) {
	f := newFakeReader()
	// Zone "Cavern": 2 rooms, both indoor -> hasOutdoor false.
	f.addRoom("Cavern", "cave", 1, false)
	f.addRoom("Cavern", "cave", 2, false)
	// Zone "Glade": 1 indoor + 1 outdoor -> hasOutdoor true.
	f.addRoom("Glade", "forest", 3, false)
	f.addRoom("Glade", "forest", 4, true)

	g, err := Build(f, Options{IncludeSecretExits: true})
	if err != nil {
		t.Fatal(err)
	}

	cav := g.Nodes["Cavern"]
	if cav.Biome != "cave" || cav.Rooms != 2 || cav.HasOutdoor {
		t.Errorf("Cavern node wrong: %+v", cav)
	}
	gld := g.Nodes["Glade"]
	if gld.Biome != "forest" || gld.Rooms != 2 || !gld.HasOutdoor {
		t.Errorf("Glade node wrong: %+v", gld)
	}
}

func TestBuild_Components(t *testing.T) {
	f := newFakeReader()
	// Component 1: A <-> B connected.
	f.addRoom("A", "plains", 1, true, ExitView{ToRoom: 2})
	f.addRoom("B", "forest", 2, true, ExitView{ToRoom: 1})
	// Component 2: C connected to D.
	f.addRoom("C", "swamp", 3, true, ExitView{ToRoom: 4})
	f.addRoom("D", "swamp", 4, true, ExitView{ToRoom: 3})
	// Component 3: Island, isolated (no exits).
	f.addRoom("Island", "ocean", 5, true)

	g, err := Build(f, Options{IncludeSecretExits: true})
	if err != nil {
		t.Fatal(err)
	}
	if g.Components != 3 {
		t.Errorf("want 3 components, got %d", g.Components)
	}
}

func TestDefaultOptions(t *testing.T) {
	o := DefaultOptions()
	if !o.IncludeSecretExits {
		t.Error("DefaultOptions should include secret exits")
	}
	want := map[string]bool{"instance_*": true, "ephemeral_*": true}
	if len(o.ExcludeZonePatterns) != len(want) {
		t.Fatalf("want %d default exclude patterns, got %v", len(want), o.ExcludeZonePatterns)
	}
	for _, p := range o.ExcludeZonePatterns {
		if !want[p] {
			t.Errorf("unexpected default exclude pattern %q", p)
		}
	}
}

func TestBuild_RoomLoadFailureSkipped(t *testing.T) {
	f := newFakeReader()
	f.addRoom("A", "plains", 1, true, ExitView{ToRoom: 2})
	f.addRoom("B", "forest", 2, true, ExitView{ToRoom: 1})
	// Register a room id in zone A that cannot be loaded (present in the zone's
	// RoomIDs, absent from rooms) — buildEdges must skip it without panicking.
	f.zoneRooms["A"] = append(f.zoneRooms["A"], 99)

	g, err := Build(f, Options{IncludeSecretExits: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Edges) != 1 || g.Edges[0].Weight != 2 {
		t.Fatalf("want one A-B edge weight 2, got %+v", g.Edges)
	}
}

func TestBuild_BadExcludePattern(t *testing.T) {
	f := newFakeReader()
	f.addRoom("Town", "city", 1, true)

	_, err := Build(f, Options{ExcludeZonePatterns: []string{"[bad"}})
	if err == nil {
		t.Fatal("want error for malformed exclude pattern, got nil")
	}
}

func TestBuild_EndToEnd(t *testing.T) {
	f := newFakeReader()
	f.addRoom("Frostfang", "tundra", 1, true, ExitView{ToRoom: 2})
	f.addRoom("Frostfang", "tundra", 2, true, ExitView{ToRoom: 1}, ExitView{ToRoom: 3})
	f.addRoom("Saltmarsh", "swamp", 3, true, ExitView{ToRoom: 2})

	opts := DefaultOptions()
	opts.BuiltAtRound = 777
	g, err := Build(f, opts)
	if err != nil {
		t.Fatal(err)
	}

	if g.Version != 1 {
		t.Errorf("want version 1, got %d", g.Version)
	}
	if g.BuiltAtRound != 777 {
		t.Errorf("want BuiltAtRound 777, got %d", g.BuiltAtRound)
	}
	if g.Components != 1 {
		t.Errorf("want 1 component, got %d", g.Components)
	}

	// The whole graph must survive the cache round-trip unchanged.
	b, err := g.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	got, err := sim.FromJSON(b)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(g, got) {
		t.Errorf("graph changed across JSON round trip")
	}
}
