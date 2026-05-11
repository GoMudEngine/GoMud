package mutations

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
// disk access.  It calls Validate() on each spec to migrate legacy Pro/Con
// fields into the Pros/Cons slices that sumEffects iterates.
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
		// Multi-effect mutation for testing new getters
		"bio-adaptation": {
			MutationId: "bio-adaptation",
			Name:       "Bio-Adaptation",
			Rarity:     4,
			Pros: []MutationEffect{
				{Type: "stamina_regen_multiplier", Value: 0.15},
				{Type: "health_regen_multiplier", Value: 0.10},
				{Type: "dodge_modifier", Value: 5.0},
				{Type: "damage_multiplier", Value: 0.08},
				{Type: "movement_speed", Value: 0.05},
				{Type: "health_regen", Value: 2.0},
				{Type: "skill_progression_multiplier", Value: 0.10},
				{Type: "stat_progression_multiplier", Value: 0.05},
			},
			Cons: []MutationEffect{
				{Type: "conviction_damage_reduction", Value: -0.05},
			},
		},
		"natural-claws": {
			MutationId: "natural-claws",
			Name:       "Natural Claws",
			Rarity:     5,
			Pro:        MutationEffect{Type: "natural_weapon", Value: 8.0},
			Con:        MutationEffect{Type: "stat_multiplier", Target: "charisma", Value: -0.05},
		},
		"sunblessed": {
			MutationId: "sunblessed",
			Name:       "Sunblessed",
			Rarity:     6,
			Pros: []MutationEffect{
				{Type: "health_regen_if_lit", Value: 3.0},
				{Type: "flag", Target: "sunblessed"},
			},
		},
		"thick-hide": {
			MutationId:  "thick-hide",
			Name:        "Thick Hide",
			Rarity:      4,
			Pro:         MutationEffect{Type: "natural_armor", Value: 15},
			Con:         MutationEffect{Type: "movement_speed", Value: -0.10},
			Conflicts:   []string{"fast-reflexes"},
		},
	}

	// Run Validate on each spec to migrate Pro/Con → Pros/Cons slices
	for _, spec := range allMutations {
		_ = spec.Validate()
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

// ─── Stage 12.2: level scaling & deepening tests ─────────────────────────────

func TestLevelMultiplier(t *testing.T) {
	cases := []struct {
		level int
		want  float64
	}{
		{1, 1.0},
		{2, 1.5},
		{3, 2.0},
		{0, 1.0}, // default
		{4, 2.5}, // level 4 multiplier
		{5, 1.0}, // beyond max → default
	}
	for _, c := range cases {
		if got := LevelMultiplier(c.level); abs(got-c.want) > 1e-9 {
			t.Errorf("LevelMultiplier(%d) = %v, want %v", c.level, got, c.want)
		}
	}
}

func TestTotalMutationEvents(t *testing.T) {
	if got := TotalMutationEvents(map[string]int{}); got != 0 {
		t.Errorf("empty map: want 0, got %d", got)
	}
	owned := buildOwned("fast-reflexes", 1, "tough-skin", 2)
	if got := TotalMutationEvents(owned); got != 3 {
		t.Errorf("want 3, got %d", got)
	}
}

func TestCanDeepen(t *testing.T) {
	if CanDeepen(map[string]int{}) {
		t.Error("empty map: want false")
	}
	allMax := buildOwned("fast-reflexes", 3, "tough-skin", 3)
	if CanDeepen(allMax) {
		t.Error("all at max: want false")
	}
	oneBelow := buildOwned("fast-reflexes", 3, "tough-skin", 1)
	if !CanDeepen(oneBelow) {
		t.Error("one below max: want true")
	}
}

func TestRollDeepening(t *testing.T) {
	seedRegistry()

	// Returns "" when map is empty
	if got := RollDeepening(map[string]int{}); got != "" {
		t.Errorf("empty map: want \"\", got %q", got)
	}

	// Returns "" when all mutations are at max level
	allMax := buildOwned("fast-reflexes", 3, "tough-skin", 3)
	if got := RollDeepening(allMax); got != "" {
		t.Errorf("all at max: want \"\", got %q", got)
	}

	// Returns an id < 3 when one exists
	oneBelow := buildOwned("fast-reflexes", 3, "tough-skin", 1)
	got := RollDeepening(oneBelow)
	if got != "tough-skin" {
		t.Errorf("only tough-skin is below max: want \"tough-skin\", got %q", got)
	}
}

func TestGetNaturalArmorScaled(t *testing.T) {
	seedRegistry()

	// L2: 25 × 1.5 = 37.5 → int(37)
	if got := GetNaturalArmor(buildOwned("tough-skin", 2)); got != 37 {
		t.Errorf("tough-skin L2: want 37, got %d", got)
	}
	// L3: 25 × 2.0 = 50
	if got := GetNaturalArmor(buildOwned("tough-skin", 3)); got != 50 {
		t.Errorf("tough-skin L3: want 50, got %d", got)
	}
}

func TestGetStatMultiplierScaled(t *testing.T) {
	seedRegistry()

	// fast-reflexes at L3: dex pro 0.10 × 2.0 = 0.20
	if got := GetStatMultiplier(buildOwned("fast-reflexes", 3), "dexterity"); abs(got-0.20) > 1e-9 {
		t.Errorf("fast-reflexes L3 dex: want 0.20, got %v", got)
	}
	// fast-reflexes at L3: str con -0.05 × 2.0 = -0.10
	if got := GetStatMultiplier(buildOwned("fast-reflexes", 3), "strength"); abs(got-(-0.10)) > 1e-9 {
		t.Errorf("fast-reflexes L3 str: want -0.10, got %v", got)
	}
}

func TestGetAdrenalSurgeBonus(t *testing.T) {
	seedRegistry()

	// Not owned → 0
	if got := GetAdrenalSurgeBonus(map[string]int{}); got != 0.0 {
		t.Errorf("not owned: want 0, got %v", got)
	}
	// L1 → 0.20
	if got := GetAdrenalSurgeBonus(buildOwned("adrenaline-surge", 1)); abs(got-0.20) > 1e-9 {
		t.Errorf("L1: want 0.20, got %v", got)
	}
	// L2 → 0.30
	if got := GetAdrenalSurgeBonus(buildOwned("adrenaline-surge", 2)); abs(got-0.30) > 1e-9 {
		t.Errorf("L2: want 0.30, got %v", got)
	}
	// L3 → 0.40
	if got := GetAdrenalSurgeBonus(buildOwned("adrenaline-surge", 3)); abs(got-0.40) > 1e-9 {
		t.Errorf("L3: want 0.40, got %v", got)
	}
}

// ─── Stage 40.2: Additional getter tests ─────────────────────────────────────

func TestGetMutationLoad(t *testing.T) {
	seedRegistry()

	tests := []struct {
		name  string
		owned map[string]int
		want  float64
	}{
		{"empty", map[string]int{}, 0.0},
		{"single L1", buildOwned("fast-reflexes", 1), 3.0},        // rarity 3 × level 1
		{"single L2", buildOwned("fast-reflexes", 2), 6.0},        // rarity 3 × level 2
		{"two mutations", buildOwned("fast-reflexes", 1, "iron-constitution", 1), 8.0}, // 3+5
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetMutationLoad(tt.owned)
			if abs(got-tt.want) > 1e-9 {
				t.Errorf("GetMutationLoad = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasConflict(t *testing.T) {
	seedRegistry()

	tests := []struct {
		name        string
		owned       map[string]int
		candidateId string
		want        bool
	}{
		{"no conflict", buildOwned("tough-skin", 1), "iron-constitution", false},
		{"direct conflict (thick-hide conflicts with fast-reflexes)", buildOwned("fast-reflexes", 1), "thick-hide", true},
		{"reverse conflict", buildOwned("thick-hide", 1), "fast-reflexes", true},
		{"candidate not found", map[string]int{}, "nonexistent", false},
		{"empty owned", map[string]int{}, "fast-reflexes", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasConflict(tt.owned, tt.candidateId)
			if got != tt.want {
				t.Errorf("HasConflict = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasMutation(t *testing.T) {
	owned := buildOwned("fast-reflexes", 1, "tough-skin", 2)

	if !HasMutation(owned, "fast-reflexes") {
		t.Error("expected HasMutation to return true for fast-reflexes")
	}
	if HasMutation(owned, "iron-constitution") {
		t.Error("expected HasMutation to return false for iron-constitution")
	}
}

func TestGetMutationLevel(t *testing.T) {
	owned := buildOwned("fast-reflexes", 2, "tough-skin", 3)

	if got := GetMutationLevel(owned, "fast-reflexes"); got != 2 {
		t.Errorf("GetMutationLevel(fast-reflexes) = %d, want 2", got)
	}
	if got := GetMutationLevel(owned, "nonexistent"); got != 0 {
		t.Errorf("GetMutationLevel(nonexistent) = %d, want 0", got)
	}
}

func TestGetStaminaRegenMultiplier(t *testing.T) {
	seedRegistry()

	// iron-constitution has stamina_regen_multiplier con = -0.15
	got := GetStaminaRegenMultiplier(buildOwned("iron-constitution", 1))
	if abs(got-(-0.15)) > 1e-9 {
		t.Errorf("iron-constitution: want -0.15, got %v", got)
	}

	// bio-adaptation has stamina_regen_multiplier pro = 0.15
	got = GetStaminaRegenMultiplier(buildOwned("bio-adaptation", 1))
	if abs(got-0.15) > 1e-9 {
		t.Errorf("bio-adaptation: want 0.15, got %v", got)
	}

	// combined: -0.15 + 0.15 = 0
	got = GetStaminaRegenMultiplier(buildOwned("iron-constitution", 1, "bio-adaptation", 1))
	if abs(got) > 1e-9 {
		t.Errorf("combined: want 0, got %v", got)
	}
}

func TestGetNaturalWeaponBonus(t *testing.T) {
	seedRegistry()

	if got := GetNaturalWeaponBonus(map[string]int{}); got != 0 {
		t.Errorf("empty: want 0, got %v", got)
	}
	if got := GetNaturalWeaponBonus(buildOwned("natural-claws", 1)); abs(got-8.0) > 1e-9 {
		t.Errorf("natural-claws L1: want 8, got %v", got)
	}
}

func TestGetConvictionResistance(t *testing.T) {
	seedRegistry()

	if got := GetConvictionResistance(map[string]int{}); got != 0 {
		t.Errorf("empty: want 0, got %v", got)
	}
	// bio-adaptation has conviction_damage_reduction con = -0.05
	got := GetConvictionResistance(buildOwned("bio-adaptation", 1))
	if abs(got-(-0.05)) > 1e-9 {
		t.Errorf("bio-adaptation: want -0.05, got %v", got)
	}
}

func TestHasMutationFlag(t *testing.T) {
	seedRegistry()

	if HasMutationFlag(map[string]int{}, "sunblessed") {
		t.Error("empty owned should not have flag")
	}
	if !HasMutationFlag(buildOwned("sunblessed", 1), "sunblessed") {
		t.Error("sunblessed should grant sunblessed flag")
	}
	if HasMutationFlag(buildOwned("fast-reflexes", 1), "sunblessed") {
		t.Error("fast-reflexes should not grant sunblessed flag")
	}
}

func TestGetConditionalHealthRegen(t *testing.T) {
	seedRegistry()

	// sunblessed has health_regen_if_lit = 3.0
	if got := GetConditionalHealthRegen(buildOwned("sunblessed", 1), true); got != 3 {
		t.Errorf("sunblessed + lit: want 3, got %d", got)
	}
	if got := GetConditionalHealthRegen(buildOwned("sunblessed", 1), false); got != 0 {
		t.Errorf("sunblessed + dark: want 0, got %d", got)
	}
	if got := GetConditionalHealthRegen(map[string]int{}, true); got != 0 {
		t.Errorf("no mutations + lit: want 0, got %d", got)
	}
}

func TestGetHealthRegenMultiplier(t *testing.T) {
	seedRegistry()

	if got := GetHealthRegenMultiplier(map[string]int{}); got != 0 {
		t.Errorf("empty: want 0, got %v", got)
	}
	// bio-adaptation has health_regen_multiplier = 0.10
	got := GetHealthRegenMultiplier(buildOwned("bio-adaptation", 1))
	if abs(got-0.10) > 1e-9 {
		t.Errorf("bio-adaptation: want 0.10, got %v", got)
	}
}

func TestGetDodgeModifier(t *testing.T) {
	seedRegistry()

	if got := GetDodgeModifier(map[string]int{}); got != 0 {
		t.Errorf("empty: want 0, got %v", got)
	}
	// bio-adaptation has dodge_modifier = 5.0
	got := GetDodgeModifier(buildOwned("bio-adaptation", 1))
	if abs(got-5.0) > 1e-9 {
		t.Errorf("bio-adaptation: want 5, got %v", got)
	}
}

func TestGetDamageMultiplier(t *testing.T) {
	seedRegistry()

	if got := GetDamageMultiplier(map[string]int{}); got != 0 {
		t.Errorf("empty: want 0, got %v", got)
	}
	// bio-adaptation has damage_multiplier = 0.08
	got := GetDamageMultiplier(buildOwned("bio-adaptation", 1))
	if abs(got-0.08) > 1e-9 {
		t.Errorf("bio-adaptation: want 0.08, got %v", got)
	}
}

func TestGetMovementSpeedModifier(t *testing.T) {
	seedRegistry()

	if got := GetMovementSpeedModifier(map[string]int{}); got != 0 {
		t.Errorf("empty: want 0, got %v", got)
	}
	// bio-adaptation has movement_speed pro = 0.05
	got := GetMovementSpeedModifier(buildOwned("bio-adaptation", 1))
	if abs(got-0.05) > 1e-9 {
		t.Errorf("bio-adaptation: want 0.05, got %v", got)
	}
	// thick-hide has movement_speed con = -0.10
	got = GetMovementSpeedModifier(buildOwned("thick-hide", 1))
	if abs(got-(-0.10)) > 1e-9 {
		t.Errorf("thick-hide: want -0.10, got %v", got)
	}
	// combined
	got = GetMovementSpeedModifier(buildOwned("bio-adaptation", 1, "thick-hide", 1))
	if abs(got-(-0.05)) > 1e-9 {
		t.Errorf("combined: want -0.05, got %v", got)
	}
}

func TestGetHealthRegen(t *testing.T) {
	seedRegistry()

	if got := GetHealthRegen(map[string]int{}); got != 0 {
		t.Errorf("empty: want 0, got %d", got)
	}
	// bio-adaptation has health_regen = 2.0
	if got := GetHealthRegen(buildOwned("bio-adaptation", 1)); got != 2 {
		t.Errorf("bio-adaptation: want 2, got %d", got)
	}
}

func TestGetSkillProgressionMultiplier(t *testing.T) {
	seedRegistry()

	if got := GetSkillProgressionMultiplier(map[string]int{}); got != 0 {
		t.Errorf("empty: want 0, got %v", got)
	}
	// bio-adaptation has skill_progression_multiplier = 0.10
	got := GetSkillProgressionMultiplier(buildOwned("bio-adaptation", 1))
	if abs(got-0.10) > 1e-9 {
		t.Errorf("bio-adaptation: want 0.10, got %v", got)
	}
}

func TestGetStatProgressionMultiplier(t *testing.T) {
	seedRegistry()

	if got := GetStatProgressionMultiplier(map[string]int{}); got != 0 {
		t.Errorf("empty: want 0, got %v", got)
	}
	// bio-adaptation has stat_progression_multiplier = 0.05
	got := GetStatProgressionMultiplier(buildOwned("bio-adaptation", 1))
	if abs(got-0.05) > 1e-9 {
		t.Errorf("bio-adaptation: want 0.05, got %v", got)
	}
}

// ─── Stage rarity uplift tests ───────────────────────────────────────────────

func TestGetWeightedPool_RarityUplift_NoMutations(t *testing.T) {
	seedRegistry()
	owned := map[string]int{}
	pool := GetWeightedPool(owned)
	assert.Greater(t, len(pool), 0, "pool should not be empty")
}

func TestGetWeightedPool_RarityUplift_HighAvgLevel(t *testing.T) {
	seedRegistry()
	basePool := GetWeightedPool(map[string]int{})

	owned := map[string]int{
		"thick-hide": 4,
		"iron-gut":   4,
	}
	upliftPool := GetWeightedPool(owned)

	assert.Greater(t, len(basePool), len(upliftPool),
		"high avg level should produce a smaller pool (reduced common weights)")
}

func TestRarityBonus_Calculation(t *testing.T) {
	tests := []struct {
		name     string
		owned    map[string]int
		expected int
	}{
		{"no mutations", map[string]int{}, 0},
		{"all level 1", map[string]int{"a": 1, "b": 1}, 0},
		{"avg level 2", map[string]int{"a": 2, "b": 2}, 1},
		{"avg level 3", map[string]int{"a": 3, "b": 3}, 2},
		{"avg level 4", map[string]int{"a": 4, "b": 4}, 3},
		{"mixed levels", map[string]int{"a": 1, "b": 3}, 1},
		{"single level 4", map[string]int{"a": 4}, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calcRarityBonus(tt.owned)
			assert.Equal(t, tt.expected, got)
		})
	}
}

// ─── Stage 2.2a: Incorporeal mutation tests ──────────────────────────────────

func TestGetGearEffectivenessLoss_PerRank(t *testing.T) {
	prev := allMutations
	defer func() { allMutations = prev }()
	allMutations = map[string]*MutationSpec{
		"test-incorporeal": {
			MutationId: "test-incorporeal",
			Name:       "Test Incorporeal",
			Rarity:     10,
			Cons: []MutationEffect{
				{Type: "gear_effectiveness_loss", Target: "", Value: 0.25},
			},
		},
	}

	cases := []struct {
		level int
		want  float64
	}{
		{1, 0.25},
		{2, 0.50},
		{3, 0.75},
		{4, 1.00},
	}
	for _, c := range cases {
		owned := map[string]int{"test-incorporeal": c.level}
		got := GetGearEffectivenessLoss(owned)
		if got != c.want {
			t.Errorf("level %d: got %.2f, want %.2f", c.level, got, c.want)
		}
	}
}

func TestGetGearEffectivenessLoss_Clamping(t *testing.T) {
	prev := allMutations
	defer func() { allMutations = prev }()
	allMutations = map[string]*MutationSpec{
		"a": {
			MutationId: "a", Name: "A", Rarity: 1,
			Cons: []MutationEffect{
				{Type: "gear_effectiveness_loss", Value: 0.50},
			},
		},
		"b": {
			MutationId: "b", Name: "B", Rarity: 1,
			Cons: []MutationEffect{
				{Type: "gear_effectiveness_loss", Value: 0.50},
			},
		},
	}
	// Two sources each at level 2 → 0.50×2 + 0.50×2 = 2.0 → clamped to 1.0
	owned := map[string]int{"a": 2, "b": 2}
	got := GetGearEffectivenessLoss(owned)
	if got != 1.0 {
		t.Errorf("expected clamp to 1.0, got %.2f", got)
	}
}

func TestGetGearEffectivenessLoss_NoEffect(t *testing.T) {
	got := GetGearEffectivenessLoss(map[string]int{})
	if got != 0.0 {
		t.Errorf("expected 0.0 for empty owned, got %.2f", got)
	}
}

func TestGearEffectivenessMultiplier_Inverts(t *testing.T) {
	prev := allMutations
	defer func() { allMutations = prev }()
	allMutations = map[string]*MutationSpec{
		"x": {
			MutationId: "x", Name: "X", Rarity: 1,
			Cons: []MutationEffect{
				{Type: "gear_effectiveness_loss", Value: 0.25},
			},
		},
	}
	for level := 0; level <= 4; level++ {
		owned := map[string]int{}
		if level > 0 {
			owned["x"] = level
		}
		mul := GearEffectivenessMultiplier(owned)
		loss := GetGearEffectivenessLoss(owned)
		if mul+loss != 1.0 {
			t.Errorf("level %d: mul %.2f + loss %.2f != 1.0", level, mul, loss)
		}
	}
}

func TestGetPhysicalDefenseBonus_PerRank(t *testing.T) {
	prev := allMutations
	defer func() { allMutations = prev }()
	allMutations = map[string]*MutationSpec{
		"y": {
			MutationId: "y", Name: "Y", Rarity: 1,
			Pros: []MutationEffect{
				{Type: "physical_defense_bonus", Value: 15},
			},
		},
	}
	// LevelMultiplier curve: 1.0/1.5/2.0/2.5
	cases := []struct {
		level int
		want  float64
	}{
		{1, 15.0},
		{2, 22.5},
		{3, 30.0},
		{4, 37.5},
	}
	for _, c := range cases {
		owned := map[string]int{"y": c.level}
		got := GetPhysicalDefenseBonus(owned)
		if got != c.want {
			t.Errorf("level %d: got %.2f, want %.2f", c.level, got, c.want)
		}
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
