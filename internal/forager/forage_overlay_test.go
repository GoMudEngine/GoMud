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

func poolContains(p []int, id int) bool {
	for _, x := range p {
		if x == id {
			return true
		}
	}
	return false
}
