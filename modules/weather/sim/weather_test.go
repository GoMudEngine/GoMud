package sim

import "testing"

func TestClamp01(t *testing.T) {
	cases := map[float64]float64{-0.5: 0, 0: 0, 0.5: 0.5, 1: 1, 1.5: 1}
	for in, want := range cases {
		if got := clamp01(in); got != want {
			t.Errorf("clamp01(%v) = %v, want %v", in, got, want)
		}
	}
}
