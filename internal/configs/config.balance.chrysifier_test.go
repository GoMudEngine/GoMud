package configs

import "testing"

func TestChrysifierDefaults(t *testing.T) {
	b := GetBalanceConfig()
	if float64(b.HomunculusCraftScale) != 4.0 {
		t.Fatalf("HomunculusCraftScale = %v, want 4.0", float64(b.HomunculusCraftScale))
	}
	if int(b.HomunculusConvictionReserve) != 1000 {
		t.Fatalf("HomunculusConvictionReserve = %v, want 1000", int(b.HomunculusConvictionReserve))
	}
}
