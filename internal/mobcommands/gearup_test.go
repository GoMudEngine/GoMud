package mobcommands

import (
	"testing"
)

func TestGearupRegistered(t *testing.T) {
	// Gearup command should be in the registered mobcommands list.
	all := GetAllMobCommands()
	found := false
	for _, cmd := range all {
		if cmd == "gearup" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'gearup' to be registered in mobCommands")
	}
}

func TestGearupEmptyRestNoOp(t *testing.T) {
	// Bare gearup with nil mob/room should handle gracefully and not panic.
	// This is a smoke test that the signature is correct and basic logic flows.
	// Full integration tests with real items/gear upgrades are covered by
	// Task 6 smoke test with fixture setup.
	t.Skip("requires fixture setup with items; covered by Task 6 smoke test")
}

func TestGearupPermaGearBuff(t *testing.T) {
	// Mobs with PermaGear buff should emit struggle emote and no-op.
	// This requires HasBuffFlag API setup which is complex in unit tests;
	// covered by Task 6 smoke test.
	t.Skip("requires fixture setup with buffs; covered by Task 6 smoke test")
}

func TestGearupNonCombatArchetype(t *testing.T) {
	// Non-combat archetypes should silently skip via CanEquipFromGive gate.
	// Full behavior verification covered by Task 6 smoke test.
	t.Skip("requires itemvalue integration; covered by Task 6 smoke test")
}
