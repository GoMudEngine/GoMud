package mobs

import (
	"os"
	"path/filepath"
	"testing"
)

// TestInstanceSavePersistsShiftedArchetype verifies the 2026-07-10
// mutation-driven-archetype-shift persistence: a mob whose live
// BehaviorArchetype differs from its template round-trips the shift
// through a SaveMobInstance -> LoadMobInstance cycle, while a mob whose
// live archetype still matches the template writes no archetype field
// (empty on load).
func TestInstanceSavePersistsShiftedArchetype(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	withMobProgressionEnabled(t)

	// Mob 1 (Skeleton Warrior, seeded by seedRegistry) has no
	// BehaviorArchetype set on its template — i.e. template value "".

	// --- Shifted mob: live archetype differs from the template's "". ---
	shifted := NewMobById(1, 100)
	if shifted == nil {
		t.Fatal("NewMobById returned nil")
	}
	shiftedPath := instancePath(shifted.MobId, shifted.Zone, shifted.Character.Name, shifted.HomeRoomId)
	_ = os.Remove(shiftedPath)
	_ = os.Remove(filepath.Dir(shiftedPath))
	t.Cleanup(func() {
		_ = os.Remove(shiftedPath)
		_ = os.Remove(filepath.Dir(shiftedPath))
	})

	// Progression so the mob persists regardless of the new archetype
	// gate — isolates the archetype-field round-trip from the
	// worth-saving gate itself (mirrors the existing
	// TestSaveMobInstance_UncharmedMobWritesFile setup).
	shifted.Character.Stats.Strength.Training = 10
	shifted.BehaviorArchetype = "generic_fighter"

	if err := SaveMobInstance(shifted); err != nil {
		t.Fatalf("SaveMobInstance(shifted): %v", err)
	}

	loadedShifted := LoadMobInstance(shifted.MobId, shifted.Zone, shifted.Character.Name, shifted.HomeRoomId)
	if loadedShifted == nil {
		t.Fatal("LoadMobInstance(shifted) returned nil")
	}
	if loadedShifted.BehaviorArchetype != "generic_fighter" {
		t.Fatalf("round-trip BehaviorArchetype = %q, want generic_fighter", loadedShifted.BehaviorArchetype)
	}

	// --- Non-shifted mob: live archetype matches the template (""). ---
	same := NewMobById(1, 101)
	if same == nil {
		t.Fatal("NewMobById returned nil")
	}
	samePath := instancePath(same.MobId, same.Zone, same.Character.Name, same.HomeRoomId)
	_ = os.Remove(samePath)
	_ = os.Remove(filepath.Dir(samePath))
	t.Cleanup(func() {
		_ = os.Remove(samePath)
		_ = os.Remove(filepath.Dir(samePath))
	})

	same.Character.Stats.Strength.Training = 10
	// same.BehaviorArchetype left at its template default ("").

	if err := SaveMobInstance(same); err != nil {
		t.Fatalf("SaveMobInstance(same): %v", err)
	}

	loadedSame := LoadMobInstance(same.MobId, same.Zone, same.Character.Name, same.HomeRoomId)
	if loadedSame == nil {
		t.Fatal("LoadMobInstance(same) returned nil")
	}
	if loadedSame.BehaviorArchetype != "" {
		t.Fatalf("non-shifted mob persisted BehaviorArchetype %q, want empty", loadedSame.BehaviorArchetype)
	}
}
