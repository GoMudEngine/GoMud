package itemvalue

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
)

func TestItemValue_EmptySpec(t *testing.T) {
	got := ItemValue(items.ItemSpec{}, PhysicalBruiser)
	if got != 0.0 {
		t.Errorf("ItemValue(empty, Bruiser) = %f, want 0", got)
	}
}

func TestItemValue_PositiveStatMod(t *testing.T) {
	spec := items.ItemSpec{
		StatMods: map[string]int{"strength": 10},
	}
	// Bruiser strength weight = 1.5 → 10 × 1.5 = 15
	got := ItemValue(spec, PhysicalBruiser)
	want := 15.0
	if got != want {
		t.Errorf("ItemValue strength+10 on Bruiser = %f, want %f", got, want)
	}
}

func TestItemValue_NegativeStatModPenalizes(t *testing.T) {
	spec := items.ItemSpec{
		StatMods: map[string]int{"strength": -5},
	}
	// Bruiser strength weight = 1.5 → -5 × 1.5 = -7.5
	got := ItemValue(spec, PhysicalBruiser)
	want := -7.5
	if got != want {
		t.Errorf("ItemValue strength-5 on Bruiser = %f, want %f", got, want)
	}
}

func TestItemValue_StatNotInProfileDefaultsToWeight1(t *testing.T) {
	// Bruiser has no entry for "foobar" — should default to 1.0
	spec := items.ItemSpec{
		StatMods: map[string]int{"foobar": 7},
	}
	got := ItemValue(spec, PhysicalBruiser)
	want := 7.0
	if got != want {
		t.Errorf("ItemValue unknown-stat+7 on Bruiser = %f, want %f", got, want)
	}
}

func TestItemValue_DamageMultiplier(t *testing.T) {
	spec := items.ItemSpec{DamageMultiplier: 1.2}
	// Bruiser PhysicalDamageWeight = 1.0 → 1.2 × 100 × 1.0 = 120
	got := ItemValue(spec, PhysicalBruiser)
	want := 120.0
	if got != want {
		t.Errorf("ItemValue dmg×1.2 on Bruiser = %f, want %f", got, want)
	}
}

func TestItemValue_SpellDamageMultiplier(t *testing.T) {
	spec := items.ItemSpec{SpellDamageMultiplier: 1.3}
	// MagicalPure SpellDamageWeight = 1.5 → 1.3 × 100 × 1.5 = 195
	got := ItemValue(spec, MagicalPure)
	want := 195.0
	if got != want {
		t.Errorf("ItemValue spelldmg×1.3 on MagicalPure = %f, want %f", got, want)
	}
}

func TestItemValue_MitigationChannels(t *testing.T) {
	spec := items.ItemSpec{
		PhysicalMitigation:   10,
		MagicalMitigation:    20,
		ConvictionMitigation: 5,
	}
	// PhysicalTank weights: phys 1.2, mag 1.0, conv 1.2
	// 10×1.2 + 20×1.0 + 5×1.2 = 12 + 20 + 6 = 38
	got := ItemValue(spec, PhysicalTank)
	want := 38.0
	if got != want {
		t.Errorf("ItemValue mit on Tank = %f, want %f", got, want)
	}
}

func TestItemValue_WeightCost(t *testing.T) {
	spec := items.ItemSpec{Weight: 4}
	// Stealth WeightPenaltyPerLb = 1.8 → -4 × 1.8 = -7.2
	got := ItemValue(spec, Stealth)
	want := -7.2
	if got != want {
		t.Errorf("ItemValue weight=4 on Stealth = %f, want %f", got, want)
	}
}

func TestItemValue_ProfilesDifferentiate(t *testing.T) {
	// Same item — a +10 strength ring — scores higher on
	// Bruiser (strength weight 1.5) than on MagicalPure
	// (strength weight 0.3).
	spec := items.ItemSpec{
		StatMods: map[string]int{"strength": 10},
	}
	bruiserScore := ItemValue(spec, PhysicalBruiser)
	pureScore := ItemValue(spec, MagicalPure)
	if bruiserScore <= pureScore {
		t.Errorf("expected Bruiser score (%f) > MagicalPure score (%f) for strength ring",
			bruiserScore, pureScore)
	}
}

func TestItemValue_WorkedExample_BruiserSword(t *testing.T) {
	// Plain 1.5× sword, 4 lb, no stat mods, on PhysicalBruiser.
	// Expected: 1.5 × 100 × 1.0 + 0 - 4 × 0.5 = 148
	spec := items.ItemSpec{
		DamageMultiplier: 1.5,
		Weight:           4,
	}
	got := ItemValue(spec, PhysicalBruiser)
	want := 148.0
	if got != want {
		t.Errorf("worked example bruiser sword: got %f want %f", got, want)
	}
}

func TestItemValue_WorkedExample_CursedSword(t *testing.T) {
	// 1.2× sword, 4 lb, strength: -5, on PhysicalBruiser.
	// Expected: -5×1.5 + 1.2×100 - 4×0.5 = -7.5 + 120 - 2 = 110.5
	spec := items.ItemSpec{
		DamageMultiplier: 1.2,
		Weight:           4,
		StatMods:         map[string]int{"strength": -5},
	}
	got := ItemValue(spec, PhysicalBruiser)
	want := 110.5
	if got != want {
		t.Errorf("worked example cursed sword: got %f want %f", got, want)
	}
}

func TestIsUpgrade_NonEquippableIsNotUpgrade(t *testing.T) {
	char := &characters.Character{}
	// items.Item{} has ItemId=0 → not equippable → SwapDelta
	// returns Score=0 → IsUpgrade returns false.
	candidate := items.Item{}
	if IsUpgrade(char, PhysicalBruiser, candidate) {
		t.Errorf("expected IsUpgrade(empty item) = false")
	}
}
