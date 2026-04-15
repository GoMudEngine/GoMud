package spells

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// seedRegistry populates the in-memory spell registry for tests without disk access.
func seedRegistry() {
	allSpells = map[string]*SpellData{
		"sparks": {
			SpellId:          "sparks",
			Name:             "Sparks",
			Type:             HarmSingle,
			Cost:             3,
			HealthCost:       0,
			Difficulty:       10,
			DamageMultiplier: 0.8,
			BaseFolds:        4,
			Schools:          []string{SchoolElemental},
		},
		"heal": {
			SpellId:         "heal",
			Name:            "Heal",
			Type:            HelpSingle,
			Cost:            5,
			HealthCost:      0,
			Difficulty:      15,
			EffectMagnitude: 3,
			BaseFolds:       6,
			Schools:         []string{SchoolVital},
		},
		"pyretic-surge": {
			SpellId:          "pyretic-surge",
			Name:             "Pyretic Surge",
			Type:             HarmSingle,
			Cost:             8,
			HealthCost:       2,
			Difficulty:       25,
			DamageMultiplier: 1.5,
			BaseFolds:        12,
			Schools:          []string{SchoolElemental, SchoolVital},
		},
		"conviction-ward": {
			SpellId:    "conviction-ward",
			Name:       "Conviction Ward",
			Type:       HelpSingle,
			Cost:       4,
			HealthCost: 0,
			Difficulty: 20,
			BaseFolds:  8,
			Schools:    []string{SchoolEnhancement},
		},
	}
}

// ─── FindSpell ──────────────────────────────────────────────────────────────

func TestFindSpell(t *testing.T) {
	seedRegistry()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"exact ID match", "sparks", "sparks"},
		{"exact ID match 2", "heal", "heal"},
		{"name match (lowercase)", "sparks", "sparks"},
		{"not found", "nonexistent", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindSpell(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ─── GetSpell ───────────────────────────────────────────────────────────────

func TestGetSpell(t *testing.T) {
	seedRegistry()

	tests := []struct {
		name    string
		spellId string
		wantNil bool
	}{
		{"found", "sparks", false},
		{"not found", "nonexistent", true},
		{"found pyretic-surge", "pyretic-surge", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetSpell(tt.spellId)
			if tt.wantNil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
				assert.Equal(t, tt.spellId, got.SpellId)
			}
		})
	}
}

// ─── FindSpellByName ────────────────────────────────────────────────────────

func TestFindSpellByName(t *testing.T) {
	seedRegistry()

	tests := []struct {
		name    string
		input   string
		wantId  string
		wantNil bool
	}{
		{"exact name", "Sparks", "sparks", false},
		{"case-insensitive", "sparks", "sparks", false},
		{"prefix match", "Pyre", "pyretic-surge", false},
		{"not found", "zzz-no-match", "", true},
		{"exact name Heal", "Heal", "heal", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindSpellByName(tt.input)
			if tt.wantNil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
				assert.Equal(t, tt.wantId, got.SpellId)
			}
		})
	}
}

// ─── GetAllSpells ───────────────────────────────────────────────────────────

func TestGetAllSpells(t *testing.T) {
	seedRegistry()

	all := GetAllSpells()
	assert.Equal(t, 4, len(all))

	// Should be a copy — modifying it should not affect the registry
	all["test-extra"] = &SpellData{SpellId: "test-extra"}
	assert.Equal(t, 4, len(allSpells), "GetAllSpells should return a copy")
}

// ─── MaxFoldsForSkill ───────────────────────────────────────────────────────

func TestMaxFoldsForSkill(t *testing.T) {
	tests := []struct {
		name       string
		skillLevel int
		want       int
	}{
		{"level 0", 0, 4},
		{"level 4", 4, 4},
		{"level 5", 5, 6},
		{"level 9", 9, 6},
		{"level 10", 10, 8},
		{"level 19", 19, 8},
		{"level 20", 20, 10},
		{"level 30", 30, 12},
		{"level 40", 40, 16},
		{"level 50", 50, 20},
		{"level 60", 60, 24},
		{"level 70", 70, 28},
		{"level 80", 80, 32},
		{"level 100", 100, 32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaxFoldsForSkill(tt.skillLevel)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ─── GetTotalConvictionCost ─────────────────────────────────────────────────

func TestGetTotalConvictionCost(t *testing.T) {
	seedRegistry()

	spell := GetSpell("sparks") // cost = 3

	tests := []struct {
		name       string
		multiplier float64
		want       int
	}{
		{"1x multiplier", 1.0, 3},
		{"2x multiplier", 2.0, 6},
		{"0.5x multiplier", 0.5, 1},
		{"zero multiplier defaults to 1x", 0.0, 3},
		{"negative multiplier defaults to 1x", -1.0, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := spell.GetTotalConvictionCost(tt.multiplier)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ─── GetTotalHealthCost ─────────────────────────────────────────────────────

func TestGetTotalHealthCost(t *testing.T) {
	seedRegistry()

	spell := GetSpell("pyretic-surge") // healthcost = 2

	tests := []struct {
		name       string
		multiplier float64
		want       int
	}{
		{"1x multiplier", 1.0, 2},
		{"3x multiplier", 3.0, 6},
		{"zero multiplier defaults to 1x", 0.0, 2},
		{"negative multiplier defaults to 1x", -1.0, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := spell.GetTotalHealthCost(tt.multiplier)
			assert.Equal(t, tt.want, got)
		})
	}

	// Spell with no health cost
	sparks := GetSpell("sparks") // healthcost = 0
	assert.Equal(t, 0, sparks.GetTotalHealthCost(1.0))
}

// ─── GetSchoolsString ───────────────────────────────────────────────────────

func TestGetSchoolsString(t *testing.T) {
	seedRegistry()

	tests := []struct {
		name    string
		spellId string
		want    string
	}{
		{"single school", "sparks", "elemental"},
		{"multiple schools", "pyretic-surge", "elemental, vital"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spell := GetSpell(tt.spellId)
			assert.NotNil(t, spell)
			assert.Equal(t, tt.want, spell.GetSchoolsString())
		})
	}

	// Empty schools
	noSchool := &SpellData{Schools: nil}
	assert.Equal(t, "Unknown", noSchool.GetSchoolsString())
}

// ─── GetEligibleSpells ──────────────────────────────────────────────────────

func TestGetEligibleSpells(t *testing.T) {
	seedRegistry()

	// Player knows sparks, skill level 20 → maxFolds = 10
	known := map[string]int{"sparks": 1}
	eligible := GetEligibleSpells(known, 20)

	// Should include heal (folds=6), conviction-ward (folds=8), NOT pyretic-surge (folds=12)
	found := map[string]bool{}
	for _, id := range eligible {
		found[id] = true
	}
	assert.True(t, found["heal"], "heal (folds=6) should be eligible at skill 20")
	assert.True(t, found["conviction-ward"], "conviction-ward (folds=8) should be eligible at skill 20")
	assert.False(t, found["pyretic-surge"], "pyretic-surge (folds=12) should NOT be eligible at skill 20")
	assert.False(t, found["sparks"], "known spell should not appear")
}

// ─── GetEligibleSpells quest-gated exclusion ─────────────────────────────────

func TestGetEligibleSpells_QuestGatedExcluded(t *testing.T) {
	// Build a registry that contains one regular spell and one quest-gated spell,
	// both well within the skill threshold so the only reason to exclude the
	// quest-gated one is the QuestRequired field.
	allSpells = map[string]*SpellData{
		"sparks": {
			SpellId:   "sparks",
			Name:      "Sparks",
			Type:      HarmSingle,
			BaseFolds: 4,
			Schools:   []string{SchoolElemental},
			// No QuestRequired → discoverable
		},
		"summon-steppe-spirit": {
			SpellId:      "summon-steppe-spirit",
			Name:         "Summon Steppe Spirit",
			Type:         HelpSingle,
			BaseFolds:    6,
			Schools:      []string{SchoolManifestation},
			QuestRequired: "12-end",
			// Quest-gated → must NEVER appear in eligible list
		},
	}

	// High skill → maxFolds = 32, so both spells are within range by folds.
	// Empty spellbook so neither is "known".
	known := map[string]int{}
	eligible := GetEligibleSpells(known, 100, SchoolElemental, SchoolManifestation)

	found := map[string]bool{}
	for _, id := range eligible {
		found[id] = true
	}

	assert.True(t, found["sparks"], "sparks (no quest required) should be discoverable")
	assert.False(t, found["summon-steppe-spirit"],
		"summon-steppe-spirit has QuestRequired set and must never appear in eligible list")
}
