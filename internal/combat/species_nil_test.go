package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/species"
)

// Finding 28: species.GetSpecies is a bare map lookup (species.go:73-75) and
// returns nil for an unknown SpeciesId. buildWeaponSetup dereferenced
// raceInfo.UnarmedName before its own raceInfo != nil check, and combat.go
// dereferenced the result inline with no check at all. A malformed or
// unmigrated SpeciesId therefore panicked mid-combat.
//
// A third site (combat_helpers.go, the defense-message path) already used the
// correct "fists" default; these lock the other two to the same contract.

// unknownSpeciesId returns an id that is guaranteed absent from the species
// table, so GetSpecies returns nil.
func unknownSpeciesId(t *testing.T) int {
	t.Helper()
	const probe = 999999
	if species.GetSpecies(probe) != nil {
		t.Skipf("species id %d unexpectedly exists; cannot exercise the nil path", probe)
	}
	return probe
}

func TestBuildWeaponSetup_UnknownSpeciesDoesNotPanic(t *testing.T) {
	id := unknownSpeciesId(t)

	src := characters.New()
	src.SpeciesId = id
	tgt := characters.New()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("buildWeaponSetup panicked on unknown SpeciesId %d: %v", id, r)
		}
	}()

	ws := buildWeaponSetup(src, tgt, items.Item{}, 0, 1)

	if ws.weaponName == "" {
		t.Error("weaponName is empty for an unknown species; expected the 'fists' fallback")
	}
}

// The fallback must be the documented "fists" default, not an empty string
// and not something that would render as a blank weapon name in combat text.
func TestBuildWeaponSetup_UnknownSpeciesFallsBackToFists(t *testing.T) {
	id := unknownSpeciesId(t)

	src := characters.New()
	src.SpeciesId = id
	tgt := characters.New()

	ws := buildWeaponSetup(src, tgt, items.Item{}, 0, 1)
	if ws.weaponName != "fists" {
		t.Errorf("weaponName = %q, want %q", ws.weaponName, "fists")
	}
	if ws.attacks < 1 {
		t.Errorf("attacks = %d, want at least 1; a zero-value species must still swing", ws.attacks)
	}
}

// GetDefaultDistributionDamage and GetDefaultDiceRoll are the deeper sites the
// original finding did not name. Exercise them directly.
func TestCharacterDefaults_UnknownSpeciesDoNotPanic(t *testing.T) {
	id := unknownSpeciesId(t)

	c := characters.New()
	c.SpeciesId = id

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("character damage defaults panicked on unknown SpeciesId %d: %v", id, r)
		}
	}()

	if attacks, _, _, _ := c.GetDefaultDistributionDamage(); attacks < 1 {
		t.Errorf("GetDefaultDistributionDamage attacks = %d, want at least 1", attacks)
	}
	c.GetDefaultDiceRoll()
}
