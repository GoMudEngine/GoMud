package engine

import "testing"

func TestIsOutdoorBiome(t *testing.T) {
	if !isOutdoorBiome("forest") {
		t.Error("forest should be outdoor")
	}
	if isOutdoorBiome("cave") {
		t.Error("cave should be indoor")
	}
	if !isOutdoorBiome("") {
		t.Error("unknown/empty biome should default to outdoor")
	}
}
