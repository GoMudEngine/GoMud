package combat

// T4 (chunk 4c) — tests for the bludgeon narration subtype swap.
//
// When ShouldBludgeon fires (weapon reach > position grapple radius),
// bladed/ranged subtypes (Slashing, Cleaving, Stabbing, Shooting) must
// be remapped to Bludgeoning in attack-message selection. Natural-blunt
// (Fist, Claws, Bite, Sting, Slam, Gore, Whipping) and caster subtypes
// (Wand, Sceptre, Staff) are exempt — their own vocabulary already
// reads correctly.
//
// These tests exercise the pure predicate logic (ShouldBludgeon +
// the switch) in isolation, without going through buildAttackMessages,
// which requires a full round-trip with message-file data loaded.

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/stretchr/testify/assert"
)

// bludgeonDisplaySubtype is a thin position-aware wrapper around the real
// production function meleeDisplaySubtype (combat_helpers.go) so the tests
// can exercise the swap logic without the full combat message pipeline.
func bludgeonDisplaySubtype(
	weaponSubType items.ItemSubType,
	weaponReach float64,
	pos *position.Machine,
) items.ItemSubType {
	posRadius := 0.0
	if pos != nil {
		posRadius = PositionReachRadius(pos.State())
	}
	return meleeDisplaySubtype(weaponSubType, weaponReach, posRadius)
}

// TestBludgeonSwap_BladedInMount_SelectsBludgeoning — sword (Slashing, 1.0m)
// in Mount (radius 0.3m) → swap fires → displaySubtype = Bludgeoning.
func TestBludgeonSwap_BladedInMount_SelectsBludgeoning(t *testing.T) {
	attacker := characters.New()
	forceMount(attacker)
	got := bludgeonDisplaySubtype(items.Slashing, 1.0, attacker.Position)
	assert.Equal(t, items.Bludgeoning, got, "sword in mount must use Bludgeoning vocabulary")
}

// TestBludgeonSwap_CleavingInMount_SelectsBludgeoning — axe (Cleaving, 0.9m)
// in Mount (radius 0.3m) → swap fires.
func TestBludgeonSwap_CleavingInMount_SelectsBludgeoning(t *testing.T) {
	attacker := characters.New()
	forceMount(attacker)
	got := bludgeonDisplaySubtype(items.Cleaving, 0.9, attacker.Position)
	assert.Equal(t, items.Bludgeoning, got, "axe in mount must use Bludgeoning vocabulary")
}

// TestBludgeonSwap_StabbingInMount_SelectsBludgeoning — spear-class weapon
// with Stabbing subtype and explicit 2.0m reach in Mount → swap fires.
// (Dagger-class Stabbing at 0.3m does NOT trigger; see TestBludgeonSwap_DaggerFits_KeepsStabbing.)
func TestBludgeonSwap_StabbingInMount_SelectsBludgeoning(t *testing.T) {
	attacker := characters.New()
	forceMount(attacker)
	// Explicit reach 2.0m (a long spear, not a dagger).
	got := bludgeonDisplaySubtype(items.Stabbing, 2.0, attacker.Position)
	assert.Equal(t, items.Bludgeoning, got, "long stabbing weapon in mount must use Bludgeoning")
}

// TestBludgeonSwap_ShootingInMount_SelectsBludgeoning — ranged weapon used
// as an improvised club in Mount → swap fires.
func TestBludgeonSwap_ShootingInMount_SelectsBludgeoning(t *testing.T) {
	attacker := characters.New()
	forceMount(attacker)
	got := bludgeonDisplaySubtype(items.Shooting, 1.0, attacker.Position)
	assert.Equal(t, items.Bludgeoning, got, "shooting weapon used as club in mount must use Bludgeoning")
}

// TestBludgeonSwap_ShootingStanding_AlwaysBludgeons — a ranged weapon swung in
// a normal (non-grapple) Standing melee round must STILL narrate as Bludgeoning.
// Unlike bladed weapons (which only swap inside a grapple), Shooting subtypes
// swap unconditionally in the melee path: the auto-attack swing is always an
// improvised club, never "fires/snipes/arrow". The deliberate SHOOT command has
// its own message set and never routes through meleeDisplaySubtype.
func TestBludgeonSwap_ShootingStanding_AlwaysBludgeons(t *testing.T) {
	attacker := characters.New() // Standing → zero grapple radius
	got := bludgeonDisplaySubtype(items.Shooting, 1.0, attacker.Position)
	assert.Equal(t, items.Bludgeoning, got, "shooting weapon swung while standing must narrate as Bludgeoning")
}

// TestBludgeonSwap_DaggerFits_KeepsStabbing — dagger reach 0.3m exactly
// matches Mount radius 0.3m → no swap (<=, not strictly greater).
func TestBludgeonSwap_DaggerFits_KeepsStabbing(t *testing.T) {
	attacker := characters.New()
	forceMount(attacker)
	// Default Stabbing reach = 0.3m (same as mount radius default).
	got := bludgeonDisplaySubtype(items.Stabbing, 0.3, attacker.Position)
	assert.Equal(t, items.Stabbing, got, "dagger (reach=radius) must keep Stabbing vocabulary")
}

// TestBludgeonSwap_FistKeepsFist — Fist reach 0.1m is always less than any
// grapple radius → no swap, regardless of position.
func TestBludgeonSwap_FistKeepsFist(t *testing.T) {
	attacker := characters.New()
	forceMount(attacker)
	got := bludgeonDisplaySubtype(items.Fist, 0.1, attacker.Position)
	assert.Equal(t, items.Fist, got, "Fist must never swap — always fits in grapple")
}

// TestBludgeonSwap_NaturalBluntKeepsSubtype — Slam/Gore/Whipping keep their
// own vocabulary even if their reach somehow exceeded the radius.
func TestBludgeonSwap_NaturalBluntKeepsSubtype(t *testing.T) {
	attacker := characters.New()
	forceMount(attacker)
	// Whipping reach 0.5m > mount radius 0.3m → ShouldBludgeon fires, but
	// Whipping is not in the switch case so it keeps its subtype.
	assert.Equal(t, items.Whipping, bludgeonDisplaySubtype(items.Whipping, 0.5, attacker.Position),
		"Whipping must keep its subtype despite exceeding radius")
	assert.Equal(t, items.Slam, bludgeonDisplaySubtype(items.Slam, 0.5, attacker.Position),
		"Slam must keep its subtype")
	assert.Equal(t, items.Gore, bludgeonDisplaySubtype(items.Gore, 0.5, attacker.Position),
		"Gore must keep its subtype")
}

// TestBludgeonSwap_CasterStaffKeepsStaff — Staff in Mount may exceed the
// grapple radius (1.5m > 0.3m), but caster subtypes are exempt from the
// swap — their vocabulary already reads as bludgeoning-flavored.
func TestBludgeonSwap_CasterStaffKeepsStaff(t *testing.T) {
	attacker := characters.New()
	forceMount(attacker)
	// Staff reach 1.5m > mount radius 0.3m — ShouldBludgeon fires, but
	// Staff is not in the bladed switch case; exempt.
	got := bludgeonDisplaySubtype(items.Staff, 1.5, attacker.Position)
	assert.Equal(t, items.Staff, got, "Staff in mount must keep Staff vocabulary (caster exempt)")
}

// TestBludgeonSwap_CasterWandInClinch_KeepsWand — Wand reach 0.4m exactly
// equals Clinch radius 0.5m... no wait, 0.4 < 0.5 so no swap anyway.
// Test the wand in mount where reach (0.4m) > radius (0.3m) — still exempt.
func TestBludgeonSwap_CasterWandInMount_KeepsWand(t *testing.T) {
	attacker := characters.New()
	forceMount(attacker)
	// Wand reach 0.4m > mount radius 0.3m: ShouldBludgeon fires, but Wand is exempt.
	got := bludgeonDisplaySubtype(items.Wand, 0.4, attacker.Position)
	assert.Equal(t, items.Wand, got, "Wand in mount must keep Wand vocabulary (caster exempt)")
}

// TestBludgeonSwap_StandingNoSwap — a sword at Standing position has zero
// grapple radius → ShouldBludgeon returns false → no swap.
func TestBludgeonSwap_StandingNoSwap(t *testing.T) {
	attacker := characters.New()
	// Position defaults to Standing after New().
	got := bludgeonDisplaySubtype(items.Slashing, 1.0, attacker.Position)
	assert.Equal(t, items.Slashing, got, "sword at Standing must keep Slashing (no grapple)")
}

// TestBludgeonSwap_NilPosition_NoSwap — nil Position machine (uninitialized
// character path) must not panic and must return the original subtype.
func TestBludgeonSwap_NilPosition_NoSwap(t *testing.T) {
	got := bludgeonDisplaySubtype(items.Slashing, 1.0, nil)
	assert.Equal(t, items.Slashing, got, "nil Position must not swap — defensive fallback")
}

// TestBludgeonSwap_SwordInClinch_SelectsBludgeoning — sword (1.0m) in Clinch
// (radius 0.5m) → swap fires.
func TestBludgeonSwap_SwordInClinch_SelectsBludgeoning(t *testing.T) {
	attacker := characters.New()
	forceClinch(attacker)
	got := bludgeonDisplaySubtype(items.Slashing, 1.0, attacker.Position)
	assert.Equal(t, items.Bludgeoning, got, "sword in clinch must use Bludgeoning vocabulary")
}
