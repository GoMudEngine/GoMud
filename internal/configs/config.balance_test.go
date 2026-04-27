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
