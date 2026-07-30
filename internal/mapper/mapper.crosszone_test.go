package mapper

import "testing"

// stubZoneGraph installs a hand-built zone graph so these tests never touch the
// room files. Real graph construction is covered by the boot-time build.
func stubZoneGraph(t *testing.T, g map[string][]zoneLink) {
	t.Helper()
	zoneGraphMu.Lock()
	prevGraph, prevBuilt := zoneGraph, zoneGraphBuilt
	zoneGraph, zoneGraphBuilt = g, true
	zoneGraphMu.Unlock()
	t.Cleanup(func() {
		zoneGraphMu.Lock()
		zoneGraph, zoneGraphBuilt = prevGraph, prevBuilt
		zoneGraphMu.Unlock()
	})
}

// A small three-zone world:  Docks --west--> Old Quarter --north--> Noble
func npStub(t *testing.T) {
	stubZoneGraph(t, map[string][]zoneLink{
		"Docks": {
			{toZone: "Old Quarter", exitRoom: 5520, exitName: "west", destRoom: 6020},
		},
		"Old Quarter": {
			{toZone: "Docks", exitRoom: 6020, exitName: "east", destRoom: 5520},
			{toZone: "Noble", exitRoom: 6029, exitName: "north", destRoom: 6007},
		},
		"Noble": {
			{toZone: "Old Quarter", exitRoom: 6007, exitName: "south", destRoom: 6029},
		},
	})
}

func TestNextZoneHop_DirectNeighbour(t *testing.T) {
	npStub(t)
	l, ok := nextZoneHop("Docks", "Old Quarter")
	if !ok {
		t.Fatal("Docks -> Old Quarter should be reachable")
	}
	if l.exitRoom != 5520 || l.exitName != "west" || l.destRoom != 6020 {
		t.Fatalf("wrong border link: %+v", l)
	}
}

// The interesting case: two hops away. The FIRST leg must still be the border
// out of the player's current zone, not the final zone's border.
func TestNextZoneHop_TwoZonesAway(t *testing.T) {
	npStub(t)
	l, ok := nextZoneHop("Docks", "Noble")
	if !ok {
		t.Fatal("Docks -> Noble should be reachable via Old Quarter")
	}
	if l.exitRoom != 5520 || l.toZone != "Old Quarter" {
		t.Fatalf("first leg should exit Docks toward Old Quarter, got %+v", l)
	}
}

func TestNextZoneHop_SameZoneAndUnreachable(t *testing.T) {
	npStub(t)
	if _, ok := nextZoneHop("Docks", "Docks"); ok {
		t.Error("same zone should not produce a zone hop")
	}
	if _, ok := nextZoneHop("Docks", "Atlantis"); ok {
		t.Error("unknown zone must not be reachable")
	}
	if _, ok := nextZoneHop("", "Noble"); ok {
		t.Error("empty from-zone must not resolve")
	}
}

// Standing ON the border room, the next hop is the crossing exit itself — there
// is no in-zone leg left to walk.
func TestCrossZoneHopFromBorderRoom(t *testing.T) {
	npStub(t)
	next, dir, ok := crossZoneHop(5520, "Docks", "Noble")
	if !ok {
		t.Fatal("standing on the border should still yield a hop")
	}
	if next != 6020 || dir != "west" {
		t.Fatalf("want the crossing exit (6020 via west), got room %d dir %q", next, dir)
	}
}

// Guard the invariant the whole feature rests on: a zone graph edge must name a
// room in the FROM zone and a room in the TO zone. A link whose exitRoom and
// destRoom are the same room is malformed.
func TestZoneLinkShapeIsSane(t *testing.T) {
	npStub(t)
	zoneGraphMu.RLock()
	defer zoneGraphMu.RUnlock()
	for zone, links := range zoneGraph {
		for _, l := range links {
			if l.toZone == zone {
				t.Errorf("%s: link points back at its own zone", zone)
			}
			if l.exitRoom == l.destRoom {
				t.Errorf("%s -> %s: exitRoom == destRoom (%d)", zone, l.toZone, l.exitRoom)
			}
			if l.exitName == "" {
				t.Errorf("%s -> %s: link has no exit direction", zone, l.toZone)
			}
		}
	}
}
