package mapper

import "testing"

// GetReciprocalExit used to scan posDeltas for the inverse delta and return
// whichever name Go's randomized map iteration reached first. Because every
// horizontal direction shares its delta with a "-gap" twin ("south" and
// "south-gap" are both {0,1,0}), `build room north` produced a return exit
// with mapdirection "south-gap" about half the time — which renders as a
// BLANK connector on the ASCII map, so the two rooms look unconnected.
//
// The reciprocal must be deterministic and must preserve the variant suffix:
// the opposite of a 2-space exit is a 2-space exit, the opposite of a gap is
// a gap.
func TestGetReciprocalExit_IsDeterministic(t *testing.T) {
	for _, tc := range []struct{ dir, want string }{
		{"north", "south"}, {"south", "north"},
		{"east", "west"}, {"west", "east"},
		{"northeast", "southwest"}, {"southwest", "northeast"},
		{"northwest", "southeast"}, {"southeast", "northwest"},
		{"up", "down"}, {"down", "up"},
	} {
		for i := 0; i < 200; i++ {
			if got := GetReciprocalExit(tc.dir); got != tc.want {
				t.Fatalf("GetReciprocalExit(%q) = %q, want %q (iteration %d)", tc.dir, got, tc.want, i)
			}
		}
	}
}

func TestGetReciprocalExit_PreservesVariantSuffix(t *testing.T) {
	for _, tc := range []struct{ dir, want string }{
		{"north-x2", "south-x2"}, {"southeast-x3", "northwest-x3"},
		{"north-gap", "south-gap"}, {"east-gap2", "west-gap2"},
		{"southwest-gap3", "northeast-gap3"},
	} {
		for i := 0; i < 50; i++ {
			if got := GetReciprocalExit(tc.dir); got != tc.want {
				t.Fatalf("GetReciprocalExit(%q) = %q, want %q", tc.dir, got, tc.want)
			}
		}
	}
}

func TestGetReciprocalExit_UnknownDirectionReturnsEmpty(t *testing.T) {
	for _, dir := range []string{"", "portal", "attic", "north-x9", "inward"} {
		if got := GetReciprocalExit(dir); got != "" {
			t.Errorf("GetReciprocalExit(%q) = %q, want empty", dir, got)
		}
	}
}

// Every name in the delta table must round-trip: reciprocal of the reciprocal
// is the original. This catches a flip table that drifts out of sync with
// posDeltas when a new direction is added.
func TestGetReciprocalExit_RoundTripsEveryKnownDirection(t *testing.T) {
	for name := range posDeltas {
		rev := GetReciprocalExit(name)
		if rev == "" {
			t.Errorf("GetReciprocalExit(%q) returned empty; every known direction needs an opposite", name)
			continue
		}
		if back := GetReciprocalExit(rev); back != name {
			t.Errorf("%q -> %q -> %q, want round-trip back to %q", name, rev, back, name)
		}
	}
}
