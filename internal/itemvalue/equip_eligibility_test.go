package itemvalue

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
)

func TestCanEquipFromGive_NilCharacter(t *testing.T) {
	if CanEquipFromGive(nil, "generic_fighter") {
		t.Errorf("expected false for nil character")
	}
}

func TestCanEquipFromGive_RegularBruiser(t *testing.T) {
	// Default character with no special archetype, no species → true.
	ch := &characters.Character{}
	if !CanEquipFromGive(ch, "generic_fighter") {
		t.Errorf("expected true for generic_fighter")
	}
}

func TestCanEquipFromGive_NonCombatArchetypes(t *testing.T) {
	cases := []string{
		"noncombat_passive",
		"noncombat_questgiver",
		"noncombat_shopkeeper",
		"prey",
		"combat_passive",
	}
	for _, arch := range cases {
		ch := &characters.Character{}
		if CanEquipFromGive(ch, arch) {
			t.Errorf("expected false for archetype %q", arch)
		}
	}
}

func TestCanScanFloorLoot_CharmedSkipsScan(t *testing.T) {
	// Charmed (companion) bruiser — should skip floor-loot scan
	// even though they CAN equip from give.
	ch := &characters.Character{}
	ch.Charm(1, -1, "")
	if !CanEquipFromGive(ch, "generic_fighter") {
		t.Errorf("charmed bruiser should still pass CanEquipFromGive (push)")
	}
	if CanScanFloorLoot(ch, "generic_fighter") {
		t.Errorf("charmed bruiser should NOT pass CanScanFloorLoot (pull)")
	}
}

func TestCanScanFloorLoot_NonCharmedBruiser(t *testing.T) {
	ch := &characters.Character{}
	if !CanScanFloorLoot(ch, "generic_fighter") {
		t.Errorf("non-charmed bruiser should pass CanScanFloorLoot")
	}
}
