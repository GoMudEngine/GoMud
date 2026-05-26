package health

// Internal tests for unexported functions in scoring.go.
// Uses package health (not health_test) to access unexported symbols.

import "testing"

func TestTerritoryMatchesZone_CaseInsensitiveDisplayPair(t *testing.T) {
	// Production pairs forager territory (snake_case) with shop zone
	// (display-case). Existing tests only covered snake-on-snake.
	cases := []struct {
		territory string
		zone      string
		want      bool
	}{
		// Real production pairs:
		{"stillwater_marsh", "Stillwater", true},
		{"thornwall_steppe", "Thornwall City", true},
		// Negative case: cross-zone shouldn't match.
		{"stillwater_marsh", "Thornwall City", false},
		// Edge: exact match still works.
		{"stillwater", "stillwater", true},
	}
	for _, tc := range cases {
		t.Run(tc.territory+"_vs_"+tc.zone, func(t *testing.T) {
			got := territoryMatchesZone(tc.territory, tc.zone)
			if got != tc.want {
				t.Errorf("territoryMatchesZone(%q, %q) = %v, want %v",
					tc.territory, tc.zone, got, tc.want)
			}
		})
	}
}
