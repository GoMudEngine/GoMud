package seeders

import "testing"

func TestRegisteredFactionIds(t *testing.T) {
	// isFaction stub: only "thornwall_citizens" and "bandits" are real factions.
	isFaction := func(g string) bool {
		return g == "thornwall_citizens" || g == "bandits"
	}

	tests := []struct {
		name   string
		groups []string
		want   []string
	}{
		{"factionless (only non-faction groups)", []string{"construct", "humanoid"}, []string{}},
		{"empty groups", nil, []string{}},
		{"one faction among non-factions", []string{"humanoid", "thornwall_citizens"}, []string{"thornwall_citizens"}},
		{"multiple factions", []string{"bandits", "coulee_folk", "thornwall_citizens"}, []string{"bandits", "thornwall_citizens"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := registeredFactionIds(tc.groups, isFaction)
			if len(got) != len(tc.want) {
				t.Fatalf("registeredFactionIds(%v) = %v, want %v", tc.groups, got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Fatalf("registeredFactionIds(%v) = %v, want %v", tc.groups, got, tc.want)
				}
			}
		})
	}
}
