package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/pets"
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

// TestApplyPetDamage_RespectsPhysicalMitigation pins that pet damage is
// mitigated like every other physical source.
//
// Before this, applyPetDamage rolled the pet's authored damage and added it to
// result.DamageToTarget without ever consulting GetPhysicalMitigation() — a
// heavily armored boss took exactly the same pet damage as an unarmored newbie.
//
// The pet joins on a 20% per-round roll, so the comparison is made over many
// invocations of the same pet against two targets that differ ONLY in
// mitigation. The join rate is identical in expectation for both, so the ratio
// of accumulated damage isolates the mitigation factor.
func TestApplyPetDamage_RespectsPhysicalMitigation(t *testing.T) {
	const (
		iterations  = 20000
		petBase     = 20
		petVariance = 3
	)

	newOwner := func() *characters.Character {
		c := &characters.Character{RoomId: 1}
		c.Pet = pets.Pet{
			Name: "Rex",
			Type: "testdog",
			Damage: items.Damage{
				Attacks:    1,
				BaseDamage: petBase,
				Variance:   petVariance,
			},
		}
		return c
	}

	// ConditionShield feeds GetPhysicalMitigation()'s non-gear term, so it sets
	// mitigation without needing the item data files loaded.
	newTarget := func(mitigationPct int) *characters.Character {
		c := &characters.Character{RoomId: 1}
		c.HealthMax.Value = 500
		c.Health = 500
		if mitigationPct > 0 {
			c.AddCondition(characters.ConditionShield, 100, float64(mitigationPct), "test")
		}
		return c
	}

	totalDamage := func(mitigationPct int) int {
		owner := newOwner()
		target := newTarget(mitigationPct)
		total := 0
		for i := 0; i < iterations; i++ {
			res := &AttackResult{}
			applyPetDamage(res, owner, target, Mob)
			total += res.DamageToTarget
		}
		return total
	}

	unarmored := totalDamage(0)
	armored := totalDamage(50)

	if unarmored <= 0 {
		t.Fatalf("expected the unarmored target to take pet damage, got %d", unarmored)
	}
	if armored >= unarmored {
		t.Fatalf("a target with 50%% physical mitigation took %d pet damage vs %d unarmored; "+
			"higher mitigation must yield lower expected pet damage", armored, unarmored)
	}

	// 50% mitigation should halve expected damage. Tolerance is wide enough to
	// absorb the binomial join-rate noise (~1% relative at n=20000) many times
	// over, so this cannot flake, while still failing loudly on no mitigation
	// (ratio 1.0) or a double-applied one (ratio 0.25).
	ratio := float64(armored) / float64(unarmored)
	if ratio < 0.40 || ratio > 0.60 {
		t.Errorf("armored/unarmored pet damage ratio = %.3f, want ~0.50 (50%% mitigation)", ratio)
	}
}
