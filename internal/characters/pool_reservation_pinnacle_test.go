package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/enchantments"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// Note on HealthMax/StaminaMax/ConvictionMax in this package's test
// environment: no config file is loaded, so every Balance multiplier that
// feeds RecalculateStats' pool-max derivation (HealthBase, HealthPerStrength,
// HealthPerVitality, etc.) is 0 -- see TestRecalculateStats_PoolMaxDerivation
// in godfunc_refactor_test.go, which characterizes this exact behavior.
// RecalculateStats only ever writes the .Mods field on these pools; .Base is
// left untouched, and StatInfo.Recalculate() computes Value = Base + Training
// + Mods. Setting .Base directly therefore gives a deterministic post-
// Validate() pool max in this environment, without needing real config data.

// TestItemSpecPoolReservation_Health proves that an ItemSpec's
// ReserveHealthPct clamps the CURRENT health pool (not HealthMax) the same
// way Chrysalis enchantment reservations do, once Validate() runs.
func TestItemSpecPoolReservation_Health(t *testing.T) {
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		999910: {ItemId: 999910, Name: "hungry blade", Type: items.Weapon, Hands: 2, ReserveHealthPct: 0.25},
	})()

	c := New()
	c.HealthMax.Base = 400
	c.Equipment.Weapon = items.New(999910)
	// Start current health absurdly high so the only thing that can bring it
	// down is the reservation clamp inside Validate().
	c.Health = 999999

	c.Validate()

	if c.HealthMax.Value != 400 {
		t.Fatalf("test setup invariant broken: expected HealthMax.Value 400, got %d", c.HealthMax.Value)
	}
	if res := c.GetPoolReservation("health", c.HealthMax.Value); res != 100 {
		t.Fatalf("expected 100 reserved (25%% of 400), got %d", res)
	}
	if c.Health != 300 {
		t.Fatalf("expected current health clamped to 300, got %d", c.Health)
	}
}

// TestItemSpecPoolReservation_StaminaAndConviction proves the stamina and
// conviction switch branches of GetPoolReservation both work, using a single
// worn item (Neck slot) carrying both percentages at once.
func TestItemSpecPoolReservation_StaminaAndConviction(t *testing.T) {
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		999911: {
			ItemId:               999911,
			Name:                 "seething prism",
			Type:                 items.Neck,
			ReserveStaminaPct:    0.15,
			ReserveConvictionPct: 0.15,
		},
	})()

	c := New()
	c.StaminaMax.Base = 200
	c.ConvictionMax.Base = 200
	c.Equipment.Neck = items.New(999911)
	c.Stamina = 999999
	c.Conviction = 999999

	c.Validate()

	if c.StaminaMax.Value != 200 || c.ConvictionMax.Value != 200 {
		t.Fatalf("test setup invariant broken: StaminaMax=%d ConvictionMax=%d", c.StaminaMax.Value, c.ConvictionMax.Value)
	}

	if res := c.GetPoolReservation("stamina", c.StaminaMax.Value); res != 30 {
		t.Fatalf("expected 30 stamina reserved (15%% of 200), got %d", res)
	}
	if c.Stamina != 170 {
		t.Fatalf("expected current stamina clamped to 170, got %d", c.Stamina)
	}

	if res := c.GetPoolReservation("conviction", c.ConvictionMax.Value); res != 30 {
		t.Fatalf("expected 30 conviction reserved (15%% of 200), got %d", res)
	}
	if c.Conviction != 170 {
		t.Fatalf("expected current conviction clamped to 170, got %d", c.Conviction)
	}
}

// TestItemSpecPoolReservation_StacksWithChrysalisEnchant proves that a single
// equipped item carrying BOTH a Chrysalis reserve enchantment AND an ItemSpec
// reserve_*_pct contributes through both mechanisms, and the contributions
// ADD. The pre-pinnacle early-continue structure made dual contribution
// impossible; this test locks in the intentional stacking behavior.
func TestItemSpecPoolReservation_StacksWithChrysalisEnchant(t *testing.T) {
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		999913: {ItemId: 999913, Name: "hungry razor", Type: items.Weapon, Hands: 1, ReserveHealthPct: 0.25},
	})()
	defer enchantments.SeedEnchantmentsForTest(map[string]*enchantments.EnchantmentDef{
		"test-hunger": {
			EnchantId:   "test-hunger",
			Name:        "Test Hunger",
			ReservePool: "health",
			Tiers:       []enchantments.TierDef{{Tier: 0, ReservePct: 0.10}},
		},
	})()

	c := New()
	c.HealthMax.Base = 400
	blade := items.New(999913)
	blade.EnchantType = "test-hunger"
	blade.EnchantTier = 0
	blade.ReservePool = "health"
	c.Equipment.Weapon = blade
	c.Health = 999999

	c.Validate()

	if c.HealthMax.Value != 400 {
		t.Fatalf("test setup invariant broken: expected HealthMax.Value 400, got %d", c.HealthMax.Value)
	}
	// Spec: floor(400 * 0.25) = 100. Enchant tier 0: floor(400 * 0.10) = 40
	// (1-handed, so no two-hand doubling). Total = 140.
	if res := c.GetPoolReservation("health", c.HealthMax.Value); res != 140 {
		t.Fatalf("expected 140 reserved (100 spec + 40 enchant), got %d", res)
	}
	if c.Health != 260 {
		t.Fatalf("expected current health clamped to 260 (400 - 140), got %d", c.Health)
	}
}

// TestItemSpecPoolReservation_NoReservationWhenUnequipped proves that
// GetPoolReservation only counts equipped items -- an item sitting in the
// backpack (not worn) must not reserve any pool.
func TestItemSpecPoolReservation_NoReservationWhenUnequipped(t *testing.T) {
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		999912: {ItemId: 999912, Name: "unworn blade", Type: items.Weapon, Hands: 2, ReserveHealthPct: 0.25},
	})()

	c := New()
	c.HealthMax.Base = 400
	c.Items = append(c.Items, items.New(999912))

	if res := c.GetPoolReservation("health", c.HealthMax.Value); res != 0 {
		t.Fatalf("expected 0 reserved for an unequipped item, got %d", res)
	}
}
