package usercommands

import "testing"

// assess discloses how much Conviction a raised companion will reserve —
// as a descriptive band, never a raw number (owner request 2026-08-02:
// companions' CP reservation was judged fair but undisclosed).

func TestConvictionReserveBand(t *testing.T) {
	const maxPool = 1000
	cases := []struct {
		reserve int
		want    string
	}{
		{50, "a small part"},           // 5%
		{149, "a small part"},          // just under 15%
		{150, "a modest share"},        // 15%
		{299, "a modest share"},        // just under 30%
		{300, "a significant portion"}, // 30%
		{499, "a significant portion"},
		{500, "a heavy share"}, // 50%
		{749, "a heavy share"},
		{750, "nearly all"}, // 75%
		{999, "nearly all"},
		{1000, "more than your spirit could hold"}, // >= max
		{1500, "more than your spirit could hold"},
	}
	for _, tc := range cases {
		if got := convictionReserveBand(tc.reserve, maxPool); got != tc.want {
			t.Errorf("convictionReserveBand(%d, %d) = %q, want %q", tc.reserve, maxPool, got, tc.want)
		}
	}
	// Degenerate pool never divides by zero.
	if got := convictionReserveBand(10, 0); got != "more than your spirit could hold" {
		t.Errorf("zero max pool: got %q", got)
	}
}

func TestJoinRaiseForms(t *testing.T) {
	cases := []struct {
		forms []string
		want  string
	}{
		{[]string{"skeleton"}, "a skeleton"},
		{[]string{"skeleton", "zombie"}, "a skeleton or zombie"},
		{[]string{"skeleton", "zombie", "wraith"}, "a skeleton, zombie, or wraith"},
	}
	for _, tc := range cases {
		if got := joinRaiseForms(tc.forms); got != tc.want {
			t.Errorf("joinRaiseForms(%v) = %q, want %q", tc.forms, got, tc.want)
		}
	}
}
