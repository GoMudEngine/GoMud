package engine

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func TestIsOutdoorBiome(t *testing.T) {
	// Indoor detection now reads BiomeInfo.Indoor from the engine's biome
	// registry rather than a hardcoded name set. Seed a registry with one
	// outdoor and one indoor biome to exercise both paths plus the
	// unknown-biome default.
	restore := rooms.SeedBiomesForTest(map[string]*rooms.BiomeInfo{
		"forest": {BiomeId: "forest", Indoor: false},
		"cave":   {BiomeId: "cave", Indoor: true},
	})
	defer restore()

	if !isOutdoorBiome("forest") {
		t.Error("forest should be outdoor")
	}
	if isOutdoorBiome("cave") {
		t.Error("cave (Indoor: true) should be indoor")
	}
	if !isOutdoorBiome("unknownbiome") {
		t.Error("unknown biome should default to outdoor")
	}
	if !isOutdoorBiome("") {
		t.Error("empty biome should default to outdoor")
	}
}
