package configs

import "testing"

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

func TestBalanceConfig_ForagerDefaults(t *testing.T) {
	cfg := &Balance{}
	cfg.Validate()

	cases := []struct {
		name string
		got  any
		want any
	}{
		{"FernwayPickupDwellRounds", cfg.FernwayPickupDwellRounds, ConfigInt(6)},
		{"ForagerForageDwellRounds", cfg.ForagerForageDwellRounds, ConfigInt(8)},
		{"ForagerCarryThresholdPct", cfg.ForagerCarryThresholdPct, ConfigFloat(0.75)},
		{"ForagerHPRecallThresholdPct", cfg.ForagerHPRecallThresholdPct, ConfigFloat(0.50)},
		{"ForagerHealPotionThresholdPct", cfg.ForagerHealPotionThresholdPct, ConfigFloat(0.75)},
		{"ForagerWaitTimeoutRounds", cfg.ForagerWaitTimeoutRounds, ConfigInt(150)},
		{"ForagerRestCarryThreshold", cfg.ForagerRestCarryThreshold, ConfigFloat(0.5)},
		{"ForagerLockboxCapacity", cfg.ForagerLockboxCapacity, ConfigInt(500)},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s default = %v, want %v", c.name, c.got, c.want)
		}
	}
}
