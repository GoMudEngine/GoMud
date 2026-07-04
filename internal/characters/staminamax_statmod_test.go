package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

// Note on HealthMax/StaminaMax/ConvictionMax in this package's test
// environment: no config file is loaded, so every Balance multiplier that
// feeds RecalculateStats' pool-max derivation (StaminaBase,
// StaminaPerStrength, etc.) is 0 -- see pool_reservation_pinnacle_test.go
// for the established pattern. RecalculateStats only ever writes the .Mods
// field on these pools; .Base is left untouched, so setting .Base directly
// gives a deterministic post-Validate() pool max in this environment
// without needing real config data. This also avoids the pool-max floor
// clamp (Value < 1 -> 1) breaking additivity of the +60 delta: with no
// .Base set, the unequipped baseline would compute Mods=0 and get floored
// to 1, so "base+60" would not equal the equipped result's true 0+60=60.
func TestStaminaMaxStatmod(t *testing.T) {
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		999950: {ItemId: 999950, Name: "wind boots", Type: items.Feet,
			StatMods: map[string]int{"staminamax": 60}},
	})()

	c := New()
	c.StaminaMax.Base = 200
	c.Validate()
	base := c.StaminaMax.Value

	c.Equipment.Feet = items.New(999950)
	c.Validate()
	if c.StaminaMax.Value != base+60 {
		t.Fatalf("expected StaminaMax %d (+60), got %d", base+60, c.StaminaMax.Value)
	}
}
