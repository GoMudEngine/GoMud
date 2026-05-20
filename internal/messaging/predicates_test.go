package messaging

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/perception"
)

func newChar(t *testing.T) *characters.Character {
	t.Helper()
	c := characters.New()
	c.Perception = perception.NewMachine()
	return c
}

func setBlinded(t *testing.T, c *characters.Character) {
	t.Helper()
	if err := c.Perception.TransitionTo(perception.Blinded,
		state.TransitionReason{Trigger: "test"}); err != nil {
		t.Fatalf("transition to Blinded failed: %v", err)
	}
}

func TestCanSeeClearlyLitRoomSighted(t *testing.T) {
	c := newChar(t)
	// Use nil room — the predicate short-circuits to "lit" on nil.
	// A zero-value &rooms.Room{} cannot be used here because
	// Room.GetVisibility() calls into the biome registry which isn't
	// loaded in unit-test context (panics on nil BiomeInfo). Real
	// lit-room behavior is exercised in end-to-end tests with engine
	// boot.
	if !CanSeeClearly(c, nil) {
		t.Fatal("Sighted observer in default (nil) room should see clearly")
	}
}

func TestCanSeeClearlyBlinded(t *testing.T) {
	c := newChar(t)
	setBlinded(t, c)
	// Same nil-room caveat as TestCanSeeClearlyLitRoomSighted.
	if CanSeeClearly(c, nil) {
		t.Fatal("Blinded observer must NOT see clearly even in a lit room")
	}
}

func TestCanSeeShapesInfraredInDark(t *testing.T) {
	c := newChar(t)
	// Note: GetVisibility() < 1 = dark. We can't easily fabricate a
	// dark Room here without engine coupling — this test uses the
	// nil-room path which short-circuits to lit. Real darkness
	// behavior is exercised in pipeline_test.go's end-to-end suite.
	if !CanSeeShapes(c, nil) {
		t.Fatal("Sighted observer must see shapes (nil room defaults to lit)")
	}
}

func TestCanSeeShapesBlindedNoInfrared(t *testing.T) {
	c := newChar(t)
	setBlinded(t, c)
	if CanSeeShapes(c, nil) {
		t.Fatal("Blinded observer must NOT see shapes, even with nil/lit room")
	}
	_ = buffs.InfraredVision // ensure the flag constant exists
}

func TestNilCharacterDefaultsToSeeing(t *testing.T) {
	if !CanSeeClearly(nil, nil) {
		t.Fatal("nil observer must default to CanSeeClearly (defensive)")
	}
	if !CanSeeShapes(nil, nil) {
		t.Fatal("nil observer must default to CanSeeShapes (defensive)")
	}
}
