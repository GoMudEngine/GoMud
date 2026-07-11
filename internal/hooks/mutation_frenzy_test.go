package hooks

import "testing"

func TestShouldFrenzy(t *testing.T) {
	if !shouldFrenzy(true, 40, 100) {
		t.Fatal("flag + <50% HP should frenzy")
	}
	if shouldFrenzy(true, 80, 100) {
		t.Fatal(">50% HP should not frenzy")
	}
	if shouldFrenzy(true, 50, 100) {
		t.Fatal("exactly 50% HP should not frenzy (strictly below half)")
	}
	if shouldFrenzy(false, 10, 100) {
		t.Fatal("without the flag, never frenzy")
	}
	if shouldFrenzy(true, 0, 0) {
		t.Fatal("zero max HP must not frenzy")
	}
}
