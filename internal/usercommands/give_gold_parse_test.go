package usercommands

import "testing"

// Finding 12: resolution sliced the amount at len-4 while execution sliced at
// len-5, so the compact form "50gold" resolved as 50 and transferred 5. Both
// paths now call parseGoldPhrase. The critical property is that ONE parse
// serves both, so these cases pin the parser itself.

func TestParseGoldPhrase(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantAmount int
		wantIsGold bool
		wantOK     bool
	}{
		// The actual regression. Pre-fix this yielded 5.
		{"compact form", "50gold", 50, true, true},
		{"spaced form", "50 gold", 50, true, true},
		{"single digit compact", "5gold", 5, true, true},
		{"single digit spaced", "5 gold", 5, true, true},
		{"large compact", "123456gold", 123456, true, true},
		{"extra whitespace", "50   gold", 50, true, true},
		{"zero", "0gold", 0, true, true},

		// Gold-shaped but not givable: isGold true so the caller emits a
		// gold-specific error rather than falling through to item lookup.
		{"negative", "-5gold", 0, true, false},
		{"non numeric", "somegold", 0, true, false},
		{"empty amount", " gold", 0, true, false},

		// Not gold at all.
		{"bare gold word", "gold", 0, false, false},
		{"item name", "rusty dagger", 0, false, false},
		{"golden item is not gold", "golden crown", 0, false, false},
		{"empty", "", 0, false, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			amount, isGold, ok := parseGoldPhrase(tc.input)
			if amount != tc.wantAmount || isGold != tc.wantIsGold || ok != tc.wantOK {
				t.Errorf("parseGoldPhrase(%q) = (%d, %v, %v), want (%d, %v, %v)",
					tc.input, amount, isGold, ok, tc.wantAmount, tc.wantIsGold, tc.wantOK)
			}
		})
	}
}

// TestParseGoldPhrase_CompactAndSpacedAgree is the specific invariant the bug
// violated: the two accepted syntaxes for the same amount must produce the
// same number.
func TestParseGoldPhrase_CompactAndSpacedAgree(t *testing.T) {
	for _, amt := range []string{"1", "5", "50", "100", "9999"} {
		compact, _, okC := parseGoldPhrase(amt + "gold")
		spaced, _, okS := parseGoldPhrase(amt + " gold")
		if !okC || !okS {
			t.Fatalf("both forms of %sgold should parse; got okC=%v okS=%v", amt, okC, okS)
		}
		if compact != spaced {
			t.Errorf("compact %sgold = %d but spaced %s gold = %d; the two syntaxes must agree",
				amt, compact, amt, spaced)
		}
	}
}
