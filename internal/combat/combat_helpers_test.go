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

// TestBuildWeaponSetup_ShootingWeaponMeleeClamp verifies that a ranged
// (Shooting-subtype) weapon equipped in the main hand clubs as a weak melee
// improvised weapon — its full damage multiplier is reserved for the deliberate
// SHOOT path, so the auto-attack melee swing clamps to UnloadedMeleeDamageCap.
// A normal melee weapon at a low multiplier is unaffected.
func TestBuildWeaponSetup_ShootingWeaponMeleeClamp(t *testing.T) {
	cleanup := species.SeedSpeciesForTest(map[int]*species.Species{
		9002: {SpeciesId: 9002, Name: "testhuman", UnarmedName: "fists", DamageMultiplier: 1.0},
	})
	defer cleanup()

	human := &characters.Character{SpeciesId: 9002}

	arbalest := items.Item{ItemId: 11, Spec: &items.ItemSpec{
		Type:             items.Weapon,
		Subtype:          items.Shooting,
		DamageMultiplier: 7.0,
	}}
	wsBow := buildWeaponSetup(human, human, arbalest, 0, 1)
	if wsBow.weaponDmgMult > unloadedMeleeDamageCap+1e-9 {
		t.Errorf("shooting weapon melee mult = %v, want clamped to <= %v", wsBow.weaponDmgMult, unloadedMeleeDamageCap)
	}

	sword := items.Item{ItemId: 12, Spec: &items.ItemSpec{
		Type:             items.Weapon,
		Subtype:          items.Slashing,
		DamageMultiplier: 1.2,
	}}
	wsSword := buildWeaponSetup(human, human, sword, 0, 1)
	if wsSword.weaponDmgMult < 1.2-1e-9 {
		t.Errorf("melee weapon melee mult = %v, want unaffected (1.2)", wsSword.weaponDmgMult)
	}
}
