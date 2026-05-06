package factions

import "testing"

func TestRepConstants(t *testing.T) {
	if RepMin != -100 {
		t.Errorf("RepMin = %d, want -100", RepMin)
	}
	if RepMax != 100 {
		t.Errorf("RepMax = %d, want 100", RepMax)
	}
}
