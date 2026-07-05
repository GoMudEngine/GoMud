package forager

import "testing"

func TestZoneAndStormForageOverlays(t *testing.T) {
	// Seed overlay tables for the test (these are package vars).
	ZoneForageYields["Test Zone"] = []int{999001}
	StormForageYields["swamp"] = []int{999002}
	defer func() {
		delete(ZoneForageYields, "Test Zone")
		delete(StormForageYields, "swamp")
	}()

	// A zone-forage item is reachable only when the zone matches.
	if !poolContains(buildForagePool("swamp", "Test Zone", "clear", false), 999001) {
		t.Fatal("zone overlay item missing when zone matches")
	}
	if poolContains(buildForagePool("swamp", "Other Zone", "clear", false), 999001) {
		t.Fatal("zone overlay item leaked into a non-matching zone")
	}
	// A storm-forage item is reachable only during a storm in that biome.
	if !poolContains(buildForagePool("swamp", "Other Zone", "storm", false), 999002) {
		t.Fatal("storm overlay item missing during storm")
	}
	if poolContains(buildForagePool("swamp", "Other Zone", "clear", false), 999002) {
		t.Fatal("storm overlay item leaked in clear weather")
	}
	// base biome commons still present
	if len(buildForagePool("swamp", "", "clear", false)) == 0 {
		t.Fatal("base swamp pool should be non-empty")
	}
}

func TestPinnacleForageReagentsPlaced(t *testing.T) {
	cases := []struct {
		zone string
		id   int
	}{
		{"Stillwater Marsh", 40198},
		{"Stillwater Marsh", 40202},
		{"Ironwind Steppe", 40199},
		{"The Fernway South", 40200},
		{"Labyrinth of Low Tunnels", 40201},
		{"The Confluence", 40203},
	}
	for _, c := range cases {
		// reachable in its own zone (biome irrelevant to zone overlay)
		if !poolContains(buildForagePool("cave", c.zone, "clear", false), c.id) {
			t.Fatalf("reagent %d not reachable in zone %q", c.id, c.zone)
		}
		// NOT reachable in a different zone
		if poolContains(buildForagePool("cave", "Nowhere Zone", "clear", false), c.id) {
			t.Fatalf("reagent %d leaked outside zone %q", c.id, c.zone)
		}
	}
	// Stormfront Residue: mountains + storm only
	if !poolContains(buildForagePool("mountains", "", "storm", false), 40204) {
		t.Fatal("Stormfront Residue not reachable in mountains during storm")
	}
	if poolContains(buildForagePool("mountains", "", "clear", false), 40204) {
		t.Fatal("Stormfront Residue leaked in clear weather")
	}
	if poolContains(buildForagePool("swamp", "", "storm", false), 40204) {
		t.Fatal("Stormfront Residue leaked into the wrong biome during storm")
	}
}

func poolContains(p []int, id int) bool {
	for _, x := range p {
		if x == id {
			return true
		}
	}
	return false
}
