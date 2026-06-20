package configs

import "testing"

func TestBalanceConfig_DefaultPricingBaselineQty(t *testing.T) {
	cfg := &Balance{}
	cfg.Validate()
	if int(cfg.DefaultPricingBaselineQty) != 3 {
		t.Errorf("DefaultPricingBaselineQty default = %d, want 3", int(cfg.DefaultPricingBaselineQty))
	}
}

func TestBalanceConfig_CaravanDefaults(t *testing.T) {
	cfg := &Balance{}
	cfg.Validate()
	if cfg.CaravanDepotDwellRounds != 360 {
		t.Errorf("CaravanDepotDwellRounds default = %d, want 360", cfg.CaravanDepotDwellRounds)
	}
	if len(cfg.CaravanServedZones) == 0 {
		t.Error("CaravanServedZones default should not be empty")
	}
	expected := map[string]bool{"Stillwater": true, "Thornwall City": true}
	for _, z := range cfg.CaravanServedZones {
		if !expected[z] {
			t.Errorf("unexpected zone in default CaravanServedZones: %q", z)
		}
		delete(expected, z)
	}
	if len(expected) > 0 {
		t.Errorf("missing default zones: %v", expected)
	}
}

func TestBalance_BountyHunterDefaults(t *testing.T) {
	b := &Balance{}
	b.validateMisc()
	if b.BountyHunterGoldThreshold != 500 {
		t.Fatalf("BountyHunterGoldThreshold = %d, want 500", int(b.BountyHunterGoldThreshold))
	}
	if b.BountyHunterBaseStatpool != 250 {
		t.Fatalf("BountyHunterBaseStatpool = %d, want 250", int(b.BountyHunterBaseStatpool))
	}
	if b.BountyHunterStatpoolPerGold != 0.25 {
		t.Fatalf("BountyHunterStatpoolPerGold = %v, want 0.25", float64(b.BountyHunterStatpoolPerGold))
	}
	if b.BountyHunterMinStatpool != 300 || b.BountyHunterMaxStatpool != 500 {
		t.Fatalf("min/max statpool = %d/%d, want 300/500", int(b.BountyHunterMinStatpool), int(b.BountyHunterMaxStatpool))
	}
	if b.BountyHunterRedispatchCooldown != 500 {
		t.Fatalf("BountyHunterRedispatchCooldown = %d, want 500", int(b.BountyHunterRedispatchCooldown))
	}
	if b.BountyHunterGearGoldDivisor != 5 {
		t.Fatalf("BountyHunterGearGoldDivisor = %d, want 5", int(b.BountyHunterGearGoldDivisor))
	}
}

func TestBalance_MobUpgradeDefaults(t *testing.T) {
	b := &Balance{}
	b.validateMisc()
	if int(b.MobUpgradeGoldReserve) != 50 {
		t.Fatalf("MobUpgradeGoldReserve = %d, want 50", int(b.MobUpgradeGoldReserve))
	}
	if float64(b.MobUpgradeMinDelta) != 1.0 {
		t.Fatalf("MobUpgradeMinDelta = %v, want 1.0", float64(b.MobUpgradeMinDelta))
	}
}

func TestBalance_RangedDefaults(t *testing.T) {
	b := &Balance{}
	b.validateCombat()
	if b.RangedShotScale != 1.0 {
		t.Errorf("RangedShotScale default = %v, want 1.0", float64(b.RangedShotScale))
	}
	if b.RangedShieldDefenseBonus != 15 {
		t.Errorf("RangedShieldDefenseBonus default = %d, want 15", int(b.RangedShieldDefenseBonus))
	}
}

func TestBalanceConfig_ForagerDefaults(t *testing.T) {
	cfg := &Balance{}
	cfg.Validate()

	cases := []struct {
		name string
		got  any
		want any
	}{
		{"FernwayPickupDwellRounds", cfg.FernwayPickupDwellRounds, ConfigInt(6)},
		{"ForagerCarryThresholdPct", cfg.ForagerCarryThresholdPct, ConfigFloat(0.75)},
		{"ForagerHPRecallThresholdPct", cfg.ForagerHPRecallThresholdPct, ConfigFloat(0.50)},
		{"ForagerHealPotionThresholdPct", cfg.ForagerHealPotionThresholdPct, ConfigFloat(0.75)},
		{"ForagerWaitTimeoutRounds", cfg.ForagerWaitTimeoutRounds, ConfigInt(150)},
		{"ForagerRestCarryThreshold", cfg.ForagerRestCarryThreshold, ConfigFloat(0.5)},
		{"ForagerLockboxCapacity", cfg.ForagerLockboxCapacity, ConfigInt(500)},
		{"ChestBackpressureResumePct", cfg.ChestBackpressureResumePct, ConfigFloat(0.9)},
		{"ForagerStuckThresholdRounds", cfg.ForagerStuckThresholdRounds, ConfigInt(600)},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s default = %v, want %v", c.name, c.got, c.want)
		}
	}
}
