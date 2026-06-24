package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/stats"
)

// seedSleepingBuff registers a minimal Sleeping buff spec in the global
// buffs registry for the duration of the test. Returns a cleanup func.
func seedSleepingBuff() func() {
	return buffs.SeedBuffsForTest(map[int]*buffs.BuffSpec{
		15: {
			BuffId:       15,
			Name:         "Sleeping",
			TriggerCount: 1000000000, // unlimited
			Flags:        []buffs.Flag{buffs.Sleeping},
		},
	})
}

// applySleepingBuff directly injects the Sleeping flag into a character's
// Buffs list without going through the full add-buff pipeline (which
// requires data-file-loaded buff specs). We build a minimal Buff and call
// Validate() so the flag index is rebuilt.
func applySleepingBuff(c *Character) {
	c.Buffs.List = append(c.Buffs.List, &buffs.Buff{
		BuffId:       15,
		TriggersLeft: 1000000000,
	})
	c.Buffs.Validate(true)
}

// TestHealthPerRound_SleepMultiplier verifies that HealthPerRound returns a
// value multiplied by SleepRegenMultiplier (default 5.0) when the character
// has the Sleeping buff flag.
func TestHealthPerRound_SleepMultiplier(t *testing.T) {
	cleanup := seedSleepingBuff()
	defer cleanup()

	c := &Character{
		HealthMax: stats.StatInfo{Value: 1000},
		Buffs:     buffs.New(),
	}

	base := c.HealthPerRound()
	if base <= 0 {
		t.Fatalf("base HealthPerRound should be positive, got %d", base)
	}

	applySleepingBuff(c)
	boosted := c.HealthPerRound()

	if boosted <= base {
		t.Errorf("expected boosted > base; got base=%d boosted=%d", base, boosted)
	}
	// Default SleepRegenMultiplier is 5.0, so boosted should be ~5×.
	// Accept 4x–6x to allow for int truncation.
	if boosted < base*4 || boosted > base*6 {
		t.Errorf("expected boosted ~5× base; got base=%d boosted=%d", base, boosted)
	}
}

// TestAddToxicity_ClampsToMaxAndReturnsTrue verifies AddToxicity clamps into
// [0, max] (rather than rejecting an over-max add) and always returns true.
func TestAddToxicity_ClampsToMaxAndReturnsTrue(t *testing.T) {
	c := &Character{} // zero Vitality → GetToxicityMax == ToxicityBaseMax (100)
	max := c.GetToxicityMax()
	if max <= 0 {
		t.Fatalf("expected positive toxicity max (config defaults), got %v", max)
	}

	c.Toxicity = max - 1
	if ok := c.AddToxicity(10); !ok {
		t.Fatalf("AddToxicity should always return true")
	}
	if c.Toxicity != max {
		t.Errorf("expected clamp to max %v, got %v", max, c.Toxicity)
	}

	// Negative amounts floor at 0.
	c.AddToxicity(-1000)
	if c.Toxicity != 0 {
		t.Errorf("expected floor at 0, got %v", c.Toxicity)
	}
}

// TestToxicitySicknessDamage verifies acute harm is 0 below the top band and scales
// with depth into it (ToxicitySicknessDamagePct default 0.02).
func TestToxicitySicknessDamage(t *testing.T) {
	c := &Character{HealthMax: stats.StatInfo{Value: 1000}}
	max := c.GetToxicityMax() // 100 (zero Vitality)

	c.Toxicity = max * 0.50 // below the 90% band
	if d := c.ToxicitySicknessDamage(); d != 0 {
		t.Errorf("expected 0 damage below 90%%, got %d", d)
	}

	c.Toxicity = max * 0.90 // exactly at the band: over=0 → 1000*0.02*1 = 20
	if d := c.ToxicitySicknessDamage(); d != 20 {
		t.Errorf("expected 20 damage at 90%%, got %d", d)
	}

	c.Toxicity = max // at max: over≈1 → 1000*0.02*~2 = ~40 (39 after float truncation)
	if d := c.ToxicitySicknessDamage(); d < 39 || d > 40 {
		t.Errorf("expected ~40 damage at max, got %d", d)
	}

	// Past max (e.g. an item pushed it over before clamp) scales higher still.
	c.Toxicity = max * 1.5 // over≈6 → 1000*0.02*~7 = ~140 (139 after float truncation)
	if d := c.ToxicitySicknessDamage(); d < 139 || d > 140 {
		t.Errorf("expected ~140 damage at 150%% max, got %d", d)
	}
}

// TestStaminaPerRound_SleepMultiplier verifies the multiplier applies to
// StaminaPerRound, composing on top of any mutation modifier.
func TestStaminaPerRound_SleepMultiplier(t *testing.T) {
	cleanup := seedSleepingBuff()
	defer cleanup()

	c := &Character{
		StaminaMax: stats.StatInfo{Value: 1000},
		Buffs:      buffs.New(),
	}

	base := c.StaminaPerRound()
	if base <= 0 {
		t.Fatalf("base StaminaPerRound should be positive, got %d", base)
	}

	applySleepingBuff(c)
	boosted := c.StaminaPerRound()

	if boosted <= base {
		t.Errorf("expected boosted > base; got base=%d boosted=%d", base, boosted)
	}
	if boosted < base*4 || boosted > base*6 {
		t.Errorf("expected boosted ~5× base; got base=%d boosted=%d", base, boosted)
	}
}

// TestConvictionPerRound_SleepMultiplier verifies the multiplier applies to
// ConvictionPerRound.
func TestConvictionPerRound_SleepMultiplier(t *testing.T) {
	cleanup := seedSleepingBuff()
	defer cleanup()

	c := &Character{
		ConvictionMax: stats.StatInfo{Value: 1000},
		Buffs:         buffs.New(),
	}

	base := c.ConvictionPerRound()
	if base <= 0 {
		t.Fatalf("base ConvictionPerRound should be positive, got %d", base)
	}

	applySleepingBuff(c)
	boosted := c.ConvictionPerRound()

	if boosted <= base {
		t.Errorf("expected boosted > base; got base=%d boosted=%d", base, boosted)
	}
	if boosted < base*4 || boosted > base*6 {
		t.Errorf("expected boosted ~5× base; got base=%d boosted=%d", base, boosted)
	}
}

// TestHealthPerRound_NoSleepNoBoost confirms the multiplier does NOT apply
// when the character is not sleeping.
func TestHealthPerRound_NoSleepNoBoost(t *testing.T) {
	cleanup := seedSleepingBuff()
	defer cleanup()

	c1 := &Character{
		HealthMax: stats.StatInfo{Value: 1000},
		Buffs:     buffs.New(),
	}
	c2 := &Character{
		HealthMax: stats.StatInfo{Value: 1000},
		Buffs:     buffs.New(),
	}
	// Only c2 gets the sleeping buff.
	applySleepingBuff(c2)

	plain := c1.HealthPerRound()
	boosted := c2.HealthPerRound()

	if plain >= boosted {
		t.Errorf("awake char should have lower regen than sleeping char; plain=%d boosted=%d", plain, boosted)
	}
}
