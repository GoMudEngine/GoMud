package casing

import "testing"

func TestAssertCanonical(t *testing.T) {
	// Canonical input: no panic.
	AssertCanonical("Temple Priest Olen", "mob", "371-tova.yaml")
	AssertCanonical("Captain of the Guard", "mob", "x.yaml")

	// Non-canonical: panics.
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on non-canonical name")
		}
	}()
	AssertCanonical("temple priest olen", "mob", "371-tova.yaml")
}
