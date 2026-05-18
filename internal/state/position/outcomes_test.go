package position

import "testing"

func TestOutcomeTierBucketing(t *testing.T) {
	cases := []struct {
		name string
		absZ float64
		want OutcomeTier
	}{
		{"deep_hold", 0.0, TierHold},
		{"hold_just_under", 0.499, TierHold},
		{"one_step_low", 0.500, TierOneStep},
		{"one_step_mid", 0.75, TierOneStep},
		{"one_step_high", 0.999, TierOneStep},
		{"two_step_low", 1.000, TierTwoStep},
		{"two_step_mid", 1.5, TierTwoStep},
		{"two_step_high", 1.999, TierTwoStep},
		{"three_step_low", 2.000, TierThreeStep},
		{"three_step_high", 5.0, TierThreeStep},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := OutcomeTierFromAbsZ(c.absZ)
			if got != c.want {
				t.Errorf("OutcomeTierFromAbsZ(%v) = %v, want %v", c.absZ, got, c.want)
			}
		})
	}
}

func TestSubWindowGate(t *testing.T) {
	if SubWindowOpens(1.499) {
		t.Error("z=1.499 should not open sub window")
	}
	if !SubWindowOpens(1.500) {
		t.Error("z=1.500 should open sub window")
	}
	if !SubWindowOpens(3.0) {
		t.Error("z=3.0 should open sub window")
	}
}
