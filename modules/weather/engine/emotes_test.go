package engine

import "testing"

func TestEmitChance(t *testing.T) {
	tests := []struct {
		name         string
		mild, strong int
		felt         float64
		want         int
	}{
		{"calm floor at felt 0", 30, 100, 0.0, 30},
		{"severe ceiling at felt 1", 30, 100, 1.0, 100},
		{"linear midpoint", 30, 100, 0.5, 65},
		{"negative felt clamps to floor", 30, 100, -0.3, 30},
		{"above-one felt clamps to ceiling", 30, 100, 1.7, 100},
		{"flat when mild==strong", 50, 50, 0.4, 50},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := emitChance(tt.mild, tt.strong, tt.felt); got != tt.want {
				t.Errorf("emitChance(%d,%d,%v) = %d, want %d",
					tt.mild, tt.strong, tt.felt, got, tt.want)
			}
		})
	}
}
