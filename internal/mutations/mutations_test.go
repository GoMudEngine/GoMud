package mutations

import (
	"testing"
)

// buildOwned is a helper that builds a fake owned map from id→level pairs.
func buildOwned(pairs ...any) map[string]int {
	m := make(map[string]int)
	for i := 0; i+1 < len(pairs); i += 2 {
		m[pairs[i].(string)] = pairs[i+1].(int)
	}
	return m
}

// seedRegistry seeds the in-memory registry for unit tests so we don't need
// disk access.
func seedRegistry() {
	allMutations = map[string]*MutationSpec{
		"fast-reflexes": {
			MutationId: "fast-reflexes",
			Name:       "Fast Reflexes",
			Rarity:     3,
			Pro:        MutationEffect{Type: "stat_multiplier", Target: "dexterity", Value: 0.10},
			Con:        MutationEffect{Type: "stat_multiplier", Target: "strength", Value: -0.05},
		},
		"tough-skin": {
			MutationId: "tough-skin",
			Name:       "Tough Skin",
			Rarity:     3,
			Pro:        MutationEffect{Type: "natural_armor", Value: 25},
			Con:        MutationEffect{Type: "stat_multiplier", Target: "dexterity", Value: -0.05},
		},
		"iron-constitution": {
			MutationId: "iron-constitution",
			Name:       "Iron Constitution",
			Rarity:     5,
			Pro:        MutationEffect{Type: "health_multiplier", Value: 0.20},
			Con:        MutationEffect{Type: "stamina_regen_multiplier", Value: -0.15},
		},
		"adrenaline-surge": {
			MutationId: "adrenaline-surge",
			Name:       "Adrenaline Surge",
			Rarity:     7,
			Pro:        MutationEffect{Type: "conditional_damage_low_hp", Value: 0.20},
			Con:        MutationEffect{Type: "stamina_regen_multiplier", Value: -0.30},
		},
		"magical-resistance": {
			MutationId: "magical-resistance",
			Name:       "Magical Resistance",
			Rarity:     8,
			Pro:        MutationEffect{Type: "magical_damage_reduction", Value: 0.25},
			Con:        MutationEffect{Type: "conviction_cost_multiplier", Value: 0.25},
		},
		"pheromone-glands": {
			MutationId: "pheromone-glands",
			Name:       "Pheromone Glands",
			Rarity:     6,
			Pro:        MutationEffect{Type: "stat_flat", Target: "charisma", Value: 20},
			Con:        MutationEffect{Type: "aggro_magnet", Value: 2.0},
		},
	}
}

func TestGetStatMultiplier(t *testing.T) {
	seedRegistry()

	tests := []struct {
		name     string
		owned    map[string]int
		stat     string
		expected float64
	}{
		{
			name:     "no mutations",
			owned:    map[string]int{},
			stat:     "dexterity",
			expected: 0.0,
		},
		{
			name:     "fast-reflexes dex pro",
			owned:    buildOwned("fast-reflexes", 1),
			stat:     "dexterity",
			expected: 0.10,
		},
		{
			name:     "fast-reflexes strength con (negative)",
			owned:    buildOwned("fast-reflexes", 1),
			stat:     "strength",
			expected: -0.05,
		},
		{
			name:     "tough-skin dex con",
			owned:    buildOwned("tough-skin", 1),
			stat:     "dexterity",
			expected: -0.05,
		},
		{
			name:     "both mutations - dex stacks",
			owned:    buildOwned("fast-reflexes", 1, "tough-skin", 1),
			stat:     "dexterity",
			expected: 0.05, // +0.10 - 0.05
		},
		{
			name:     "stat not affected",
			owned:    buildOwned("fast-reflexes", 1),
			stat:     "willpower",
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetStatMultiplier(tt.owned, tt.stat)
			if abs(got-tt.expected) > 1e-9 {
				t.Errorf("GetStatMultiplier(%q) = %v, want %v", tt.stat, got, tt.expected)
			}
		})
	}
}

func TestGetNaturalArmor(t *testing.T) {
	seedRegistry()

	if got := GetNaturalArmor(map[string]int{}); got != 0 {
		t.Errorf("empty owned: want 0, got %d", got)
	}
	if got := GetNaturalArmor(buildOwned("tough-skin", 1)); got != 25 {
		t.Errorf("tough-skin: want 25, got %d", got)
	}
}

func TestGetHealthMultiplier(t *testing.T) {
	seedRegistry()

	if got := GetHealthMultiplier(map[string]int{}); got != 0.0 {
		t.Errorf("empty owned: want 0, got %v", got)
	}
	if got := GetHealthMultiplier(buildOwned("iron-constitution", 1)); abs(got-0.20) > 1e-9 {
		t.Errorf("iron-constitution: want 0.20, got %v", got)
	}
}

func TestGetMagicalResistance(t *testing.T) {
	seedRegistry()

	if got := GetMagicalResistance(map[string]int{}); got != 0.0 {
		t.Errorf("empty owned: want 0, got %v", got)
	}
	if got := GetMagicalResistance(buildOwned("magical-resistance", 1)); abs(got-0.25) > 1e-9 {
		t.Errorf("magical-resistance: want 0.25, got %v", got)
	}
}

func TestGetConvictionCostMultiplier(t *testing.T) {
	seedRegistry()

	if got := GetConvictionCostMultiplier(map[string]int{}); got != 0.0 {
		t.Errorf("empty owned: want 0, got %v", got)
	}
	if got := GetConvictionCostMultiplier(buildOwned("magical-resistance", 1)); abs(got-0.25) > 1e-9 {
		t.Errorf("magical-resistance: want 0.25, got %v", got)
	}
}

func TestGetAggroMagnet(t *testing.T) {
	seedRegistry()

	if got := GetAggroMagnet(map[string]int{}); got != 0.0 {
		t.Errorf("empty owned: want 0, got %v", got)
	}
	if got := GetAggroMagnet(buildOwned("pheromone-glands", 1)); abs(got-2.0) > 1e-9 {
		t.Errorf("pheromone-glands: want 2.0, got %v", got)
	}
}

func TestGetWeightedPool(t *testing.T) {
	seedRegistry()

	// Empty owned — all mutations in pool
	pool := GetWeightedPool(map[string]int{})
	if len(pool) == 0 {
		t.Fatal("expected non-empty pool for empty owned")
	}

	// Count occurrences of fast-reflexes (rarity 3 → 11-3 = 8 entries)
	frCount := countOccurrences(pool, "fast-reflexes")
	if frCount != 8 {
		t.Errorf("fast-reflexes (rarity 3) should appear 8 times, got %d", frCount)
	}

	// Count magical-resistance (rarity 8 → 11-8 = 3 entries)
	mrCount := countOccurrences(pool, "magical-resistance")
	if mrCount != 3 {
		t.Errorf("magical-resistance (rarity 8) should appear 3 times, got %d", mrCount)
	}

	// Owning a mutation excludes it from the pool
	pool2 := GetWeightedPool(buildOwned("fast-reflexes", 1))
	if countOccurrences(pool2, "fast-reflexes") > 0 {
		t.Error("owned mutation should be excluded from pool")
	}
}

func TestIsAdrenalSurgeActive(t *testing.T) {
	seedRegistry()

	owned := buildOwned("adrenaline-surge", 1)
	noOwned := map[string]int{}

	tests := []struct {
		name      string
		owned     map[string]int
		currentHP int
		maxHP     int
		expected  bool
	}{
		{"no mutation", noOwned, 10, 100, false},
		{"at exactly 25%", owned, 25, 100, false}, // 25*4 = 100, NOT < 100
		{"just below 25%", owned, 24, 100, true},   // 24*4 = 96 < 100
		{"full HP", owned, 100, 100, false},
		{"zero maxHP guard", owned, 0, 0, false},
		{"1 HP out of 100", owned, 1, 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAdrenalSurgeActive(tt.owned, tt.currentHP, tt.maxHP)
			if got != tt.expected {
				t.Errorf("IsAdrenalSurgeActive(%d/%d) = %v, want %v", tt.currentHP, tt.maxHP, got, tt.expected)
			}
		})
	}
}

func TestRollAcquisition(t *testing.T) {
	seedRegistry()

	pool := GetWeightedPool(map[string]int{})
	result := RollAcquisition(pool)
	if result == "" {
		t.Error("RollAcquisition on non-empty pool should return a mutation id")
	}
	if GetMutation(result) == nil {
		t.Errorf("RollAcquisition returned %q which is not a known mutation", result)
	}

	// Empty pool
	if got := RollAcquisition([]string{}); got != "" {
		t.Errorf("RollAcquisition on empty pool should return \"\", got %q", got)
	}
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func countOccurrences(pool []string, id string) int {
	n := 0
	for _, v := range pool {
		if v == id {
			n++
		}
	}
	return n
}
