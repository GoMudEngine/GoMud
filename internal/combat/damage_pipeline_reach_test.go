package combat

// T3 (chunk 4c) — tests for CalcReachAdjustedItemMult.
// Verifies that the reach-utility factor is correctly composed with the
// weapon's base DamageMultiplier across (standing, clinch, mount) ×
// (dagger, sword, pike) position × weapon combinations.
//
// Helper makeTestWeapon builds a minimal items.Item with an inline
// ItemSpec so no data files need to be loaded.
//
// Helper forceMount transitions a fresh character's Position FSM
// through Standing → Clinch → Mount using a dummy partner UserId.

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/stretchr/testify/assert"
)

// makeTestWeapon returns an items.Item with an inline Spec.
// subtype drives the reach default; dmgMult is DamageMultiplier;
// reach > 0 sets an explicit per-item Reach (0 = use subtype default).
func makeTestWeapon(subtype items.ItemSubType, dmgMult float64, reach float64) items.Item {
	spec := &items.ItemSpec{
		Subtype:          subtype,
		Type:             items.Weapon,
		DamageMultiplier: dmgMult,
		Reach:            reach,
	}
	return items.Item{ItemId: 9999, Spec: spec}
}

// forceMount transitions c's Position FSM to Mount state.
// Uses a dummy partner (UserId 99) — the combat engine doesn't
// verify partner liveness in unit test contexts.
func forceMount(c *characters.Character) {
	partner := state.ActorRef{UserId: 99}
	reason := state.TransitionReason{Trigger: position.TriggerGrappleEntry}
	_ = c.Position.TransitionToClinch(
		position.GrappleData{Partner: partner},
		reason,
	)
	_ = c.Position.TransitionToMount(
		position.GrappleData{Partner: partner, ControlLevel: position.InControl},
		state.TransitionReason{Trigger: position.TriggerTakedownMount},
	)
}

// forceClinch transitions c's Position FSM to Clinch state.
func forceClinch(c *characters.Character) {
	partner := state.ActorRef{UserId: 99}
	reason := state.TransitionReason{Trigger: position.TriggerGrappleEntry}
	_ = c.Position.TransitionToClinch(
		position.GrappleData{Partner: partner},
		reason,
	)
}

// TestCalcReachAdjustedItemMult_StandingNoPenalty confirms that a sword
// at standing (no grapple) incurs no reach penalty.
func TestCalcReachAdjustedItemMult_StandingNoPenalty(t *testing.T) {
	attacker := characters.New()
	// Standing is the default position after New().
	weapon := makeTestWeapon(items.Slashing, 1.2, 1.0) // sword, 1.2× mult, 1.0m reach
	got := CalcReachAdjustedItemMult(weapon, attacker)
	// No grapple → utility = 1.0 → result = 1.2 × 1.0 = 1.2
	assert.InDelta(t, 1.2, got, 0.001, "standing: no reach penalty expected")
}

// TestCalcReachAdjustedItemMult_SwordInMount confirms a 1.0m sword in
// Mount (ground-grapple radius 0.3m) gets the expected 0.3 multiplier.
func TestCalcReachAdjustedItemMult_SwordInMount(t *testing.T) {
	cfg := configs.GetBalanceConfig()
	groundR := float64(cfg.ReachGroundGrappleRadius) // default 0.3

	attacker := characters.New()
	forceMount(attacker)
	weapon := makeTestWeapon(items.Slashing, 1.0, 1.0) // sword, base 1.0, reach 1.0
	got := CalcReachAdjustedItemMult(weapon, attacker)
	// utility = 0.3/1.0 = 0.3; base 1.0 × 0.3 = 0.3
	expected := 1.0 * (groundR / 1.0)
	assert.InDelta(t, expected, got, 0.001, "sword in mount: reach penalty")
}

// TestCalcReachAdjustedItemMult_DaggerInMount confirms a dagger whose
// subtype default reach (0.3m) fits inside the mount radius (0.3m),
// so no penalty applies.
func TestCalcReachAdjustedItemMult_DaggerInMount(t *testing.T) {
	attacker := characters.New()
	forceMount(attacker)
	// Stabbing subtype default reach = 0.3; explicit reach = 0 (use default).
	weapon := makeTestWeapon(items.Stabbing, 0.8, 0)
	got := CalcReachAdjustedItemMult(weapon, attacker)
	// reach 0.3 == radius 0.3 → utility = 1.0; base 0.8 × 1.0 = 0.8
	assert.InDelta(t, 0.8, got, 0.001, "dagger in mount: no penalty (reach fits)")
}

// TestCalcReachAdjustedItemMult_PikeInMountFloors confirms that a weapon
// whose reach far exceeds the mount radius is floored to
// ReachUtilityFloor (default 0.15).
func TestCalcReachAdjustedItemMult_PikeInMountFloors(t *testing.T) {
	cfg := configs.GetBalanceConfig()
	floor := float64(cfg.ReachUtilityFloor) // default 0.15

	attacker := characters.New()
	forceMount(attacker)
	// Explicit reach 3.0m: raw utility = 0.3/3.0 = 0.1 → floored to 0.15
	weapon := makeTestWeapon(items.Stabbing, 1.0, 3.0)
	got := CalcReachAdjustedItemMult(weapon, attacker)
	expected := 1.0 * floor
	assert.InDelta(t, expected, got, 0.001, "pike in mount: floored at ReachUtilityFloor")
}

// TestCalcReachAdjustedItemMult_SwordInClinch confirms a 1.0m sword in
// Clinch (standing-grapple radius 0.5m) gets a 0.5 multiplier.
func TestCalcReachAdjustedItemMult_SwordInClinch(t *testing.T) {
	cfg := configs.GetBalanceConfig()
	standR := float64(cfg.ReachStandingGrappleRadius) // default 0.5

	attacker := characters.New()
	forceClinch(attacker)
	weapon := makeTestWeapon(items.Slashing, 1.0, 1.0) // sword, base 1.0, reach 1.0
	got := CalcReachAdjustedItemMult(weapon, attacker)
	expected := 1.0 * (standR / 1.0)
	assert.InDelta(t, expected, got, 0.001, "sword in clinch: partial reach penalty")
}

// TestCalcReachAdjustedItemMult_ZeroDmgMultDefaultsToOne confirms that a
// weapon with DamageMultiplier == 0 (unset in YAML) defaults the base to
// 1.0 rather than propagating a zero.
func TestCalcReachAdjustedItemMult_ZeroDmgMultDefaultsToOne(t *testing.T) {
	attacker := characters.New()
	// DamageMultiplier left at zero (omitempty in YAML); reach 0.3 = mount radius.
	weapon := makeTestWeapon(items.Stabbing, 0, 0.3)
	forceMount(attacker)
	got := CalcReachAdjustedItemMult(weapon, attacker)
	// baseMult defaults to 1.0; reach 0.3 = radius 0.3 → utility = 1.0
	assert.InDelta(t, 1.0, got, 0.001, "zero DamageMultiplier defaults to 1.0")
}

// TestCalcReachAdjustedItemMult_NilPosition confirms that a nil Position
// machine falls back to returning baseMult unchanged.
func TestCalcReachAdjustedItemMult_NilPosition(t *testing.T) {
	attacker := characters.New()
	attacker.Position = nil // simulate a character that hasn't been initialized
	weapon := makeTestWeapon(items.Slashing, 1.5, 1.0)
	got := CalcReachAdjustedItemMult(weapon, attacker)
	// No position → no reach penalty; returns baseMult = 1.5
	assert.InDelta(t, 1.5, got, 0.001, "nil Position: baseMult unchanged")
}

// TestCalcReachAdjustedItemMult_UnarmedItem confirms that a zero-ItemId
// (no weapon / fist) returns 1.0.
func TestCalcReachAdjustedItemMult_UnarmedItem(t *testing.T) {
	attacker := characters.New()
	forceMount(attacker)
	unarmed := items.Item{ItemId: 0}
	got := CalcReachAdjustedItemMult(unarmed, attacker)
	assert.InDelta(t, 1.0, got, 0.001, "unarmed (ItemId 0): returns 1.0")
}
