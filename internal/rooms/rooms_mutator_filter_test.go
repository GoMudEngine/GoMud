package rooms

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mutators"
)

func TestActiveMutators_OutdoorOnlySkippedIndoors(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	biomesCleanup := SeedBiomesForTest(map[string]*BiomeInfo{
		"testfield": {BiomeId: "testfield", Name: "Field", Symbol: "."},
		"testhouse": {BiomeId: "testhouse", Name: "House", Symbol: "H", Indoor: true},
	})
	defer biomesCleanup()

	mutators.SeedSpecsForTest(t,
		mutators.MutatorSpec{MutatorId: "weather-test-rain", OutdoorOnly: true},
		mutators.MutatorSpec{MutatorId: "test-sanctuary"},
	)

	zc := GetZoneConfig("TestZone")
	zc.Mutators.Add("weather-test-rain")
	zc.Mutators.Add("test-sanctuary")

	outdoorRoom := roomManager.rooms[1]
	outdoorRoom.Biome = "testfield"
	indoorRoom := roomManager.rooms[2]
	indoorRoom.Biome = "testhouse"

	got := map[string]bool{}
	for mut := range outdoorRoom.ActiveMutators {
		got[mut.MutatorId] = true
	}
	if !got["weather-test-rain"] || !got["test-sanctuary"] {
		t.Errorf("outdoor room should see both mutators, got %v", got)
	}

	got = map[string]bool{}
	for mut := range indoorRoom.ActiveMutators {
		got[mut.MutatorId] = true
	}
	if got["weather-test-rain"] {
		t.Error("indoor room must NOT see outdoor-only mutator")
	}
	if !got["test-sanctuary"] {
		t.Error("indoor room should still see normal mutator")
	}
}
