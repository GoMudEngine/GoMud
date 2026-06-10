package engine

import "testing"

func TestTickPeriod(t *testing.T) {
	if got := TickPeriod(1); got != "1 hours" {
		t.Errorf("TickPeriod(1) = %q", got)
	}
	if got := TickPeriod(6); got != "6 hours" {
		t.Errorf("TickPeriod(6) = %q", got)
	}
	if got := TickPeriod(0); got != "1 hours" {
		t.Errorf("TickPeriod(0) must clamp to 1, got %q", got)
	}
	if got := TickPeriod(-3); got != "1 hours" {
		t.Errorf("TickPeriod(-3) must clamp to 1, got %q", got)
	}
}
