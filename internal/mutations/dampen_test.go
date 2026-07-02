package mutations

import "testing"

func TestDampenBonus(t *testing.T) {
	cases := []struct {
		name   string
		raw    float64
		factor float64
		want   float64
	}{
		{"no bonus stays put", 1.0, 0.35, 1.0},
		{"below baseline untouched", 0.8, 0.35, 0.8},
		{"bonus collapses toward baseline", 2.0, 0.35, 1.35},
		{"full factor is a no-op", 1.5, 1.0, 1.5},
		{"zero factor removes the bonus", 1.5, 0.0, 1.0},
	}
	for _, c := range cases {
		if got := DampenBonus(c.raw, c.factor); got != c.want {
			t.Errorf("%s: DampenBonus(%v,%v)=%v want %v", c.name, c.raw, c.factor, got, c.want)
		}
	}
}
