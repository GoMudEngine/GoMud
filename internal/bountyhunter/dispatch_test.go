package bountyhunter

import "testing"

func TestShouldDispatch(t *testing.T) {
	// (maxBountyGold, threshold, hasActive, lastKilledRound, nowRound, cooldown)
	if !shouldDispatch(600, 500, false, 0, 1000, 500) {
		t.Fatalf("expected dispatch when over threshold, idle, off cooldown")
	}
	if shouldDispatch(400, 500, false, 0, 1000, 500) {
		t.Fatalf("below threshold must not dispatch")
	}
	if shouldDispatch(600, 500, true, 0, 1000, 500) {
		t.Fatalf("active hunt must not double-dispatch")
	}
	if shouldDispatch(600, 500, false, 800, 1000, 500) {
		t.Fatalf("within redispatch cooldown (200<500) must not dispatch")
	}
	if !shouldDispatch(600, 500, false, 400, 1000, 500) {
		t.Fatalf("past cooldown (600>=500) should dispatch")
	}
}
