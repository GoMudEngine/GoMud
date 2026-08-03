package util

import "testing"

// Player phrasing never requires typing an apostrophe: "healers root" must
// match "Healer's Root" (prepush sweep 2026-08-03 — the shop matcher said
// "can't offer that right now" for in-stock goods). Both straight and
// typographic apostrophes are stripped on both sides of the comparison.
func TestFindMatchIn_ApostropheInsensitive(t *testing.T) {
	names := []string{"Healer's Root", "Bitter Thistle", "Elgar’s Hook-spear"}

	cases := []struct{ query, want string }{
		{"healers root", "Healer's Root"},
		{"healer's root", "Healer's Root"},
		{"elgars hook-spear", "Elgar’s Hook-spear"},
		{"bitter thistle", "Bitter Thistle"},
	}
	for _, tc := range cases {
		match, closeMatch := FindMatchIn(tc.query, names...)
		got := match
		if got == "" {
			got = closeMatch
		}
		if got != tc.want {
			t.Errorf("FindMatchIn(%q) = %q/%q, want %q", tc.query, match, closeMatch, tc.want)
		}
	}
}

func TestFindMatchIn_PrefixStillWorks(t *testing.T) {
	match, closeMatch := FindMatchIn("healer", "Healer's Root")
	if match != "" || closeMatch != "Healer's Root" {
		t.Errorf("prefix match broke: %q/%q", match, closeMatch)
	}
}
