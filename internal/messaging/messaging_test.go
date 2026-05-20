package messaging

import "testing"

func TestCategoryDefaultIsZero(t *testing.T) {
	if CategoryDefault != 0 {
		t.Fatalf("CategoryDefault must be zero-value (got %d)", CategoryDefault)
	}
}

func TestCategoryStringRoundTrip(t *testing.T) {
	seen := map[string]Category{}
	for c := CategoryDefault; c < categoryMax; c++ {
		s := c.String()
		if s == "" || s == "Unknown" {
			t.Errorf("category %d returned %q (every enum value must have a real String())", c, s)
		}
		if prev, ok := seen[s]; ok && prev != c {
			t.Errorf("category %q is ambiguous (%d and %d)", s, prev, c)
		}
		seen[s] = c
	}
}

func TestUnknownCategoryString(t *testing.T) {
	if Category(-1).String() != "Unknown" {
		t.Fatalf("negative Category should stringify to Unknown")
	}
}
