package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/species"
)

func TestBuildWeaponSetup_UsesSpeciesNaturalAttack(t *testing.T) {
	cleanup := species.SeedSpeciesForTest(map[int]*species.Species{
		9001: {SpeciesId: 9001, Name: "testwolf", UnarmedName: "jaws", NaturalAttack: items.Bite, DamageMultiplier: 1.0},
		9002: {SpeciesId: 9002, Name: "testhuman", UnarmedName: "fists", DamageMultiplier: 1.0},
	})
	defer cleanup()

	wolf := &characters.Character{SpeciesId: 9001}
	human := &characters.Character{SpeciesId: 9002}
	noWeapon := items.Item{}

	wsWolf := buildWeaponSetup(wolf, wolf, noWeapon, 0, 1)
	if wsWolf.weaponSubType != items.Bite {
		t.Errorf("unarmed wolf subtype = %q, want %q", wsWolf.weaponSubType, items.Bite)
	}
	wsHuman := buildWeaponSetup(human, human, noWeapon, 0, 1)
	if wsHuman.weaponSubType != items.Unarmed {
		t.Errorf("unarmed human subtype = %q, want %q", wsHuman.weaponSubType, items.Unarmed)
	}
}

// TestBuildWeaponSetup_EquippedWeaponOverridesNaturalAttack verifies that
// a real equipped weapon wins over the species NaturalAttack field.
func TestBuildWeaponSetup_EquippedWeaponOverridesNaturalAttack(t *testing.T) {
	cleanup := species.SeedSpeciesForTest(map[int]*species.Species{
		9001: {SpeciesId: 9001, Name: "testwolf", UnarmedName: "jaws", NaturalAttack: items.Bite, DamageMultiplier: 1.0},
	})
	defer cleanup()

	wolf := &characters.Character{SpeciesId: 9001}
	sword := items.Item{ItemId: 10, Spec: &items.ItemSpec{
		Type:    items.Weapon,
		Subtype: items.Slashing,
	}}

	ws := buildWeaponSetup(wolf, wolf, sword, 0, 1)
	if ws.weaponSubType != items.Slashing {
		t.Errorf("wolf with sword subtype = %q, want %q", ws.weaponSubType, items.Slashing)
	}
}
