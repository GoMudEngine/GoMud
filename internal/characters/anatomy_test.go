package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/species"
)

func TestCharacter_HasBodyPart(t *testing.T) {
	cleanup := species.SeedSpeciesForTest(map[int]*species.Species{
		7001: {SpeciesId: 7001, Name: "armed", BodyParts: []string{"arms", "legs", "mouth"}},
		7002: {SpeciesId: 7002, Name: "noarms", BodyParts: []string{"legs", "mouth"}},
	})
	defer cleanup()

	armed := &Character{SpeciesId: 7001}
	beast := &Character{SpeciesId: 7002}

	if !armed.HasBodyPart("arms") {
		t.Error("armed should have arms")
	}
	if beast.HasBodyPart("arms") {
		t.Error("beast should NOT have arms")
	}
	if !beast.HasBodyPart("legs") {
		t.Error("beast should have legs")
	}
}
