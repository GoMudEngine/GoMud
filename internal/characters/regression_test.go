package characters

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ─── Regression Tests ────────────────────────────────────────────────────────
// These tests guard against specific past bug fixes. Each test documents the
// original bug and verifies the fix remains in place.

// TestRegression_IsDisabledOnlyChecksHealth guards against Stage 39.2 bug
// where IsDisabled() returned true when stamina or conviction was 0,
// incorrectly disabling the character. Only health <= 0 should disable.
func TestRegression_IsDisabledOnlyChecksHealth(t *testing.T) {
	testCases := []struct {
		name       string
		health     int
		stamina    int
		conviction int
		wantDisabled bool
	}{
		{"all_resources_full", 100, 100, 100, false},
		{"zero_stamina", 100, 0, 100, false},
		{"zero_conviction", 100, 100, 0, false},
		{"zero_stamina_and_conviction", 100, 0, 0, false},
		{"zero_health", 0, 100, 100, true},
		{"negative_health", -5, 100, 100, true},
		{"all_zero", 0, 0, 0, true},
		{"health_one", 1, 0, 0, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := New()
			c.Health = tc.health
			c.Stamina = tc.stamina
			c.Conviction = tc.conviction

			got := c.IsDisabled()
			assert.Equal(t, tc.wantDisabled, got,
				"IsDisabled() with health=%d, stamina=%d, conviction=%d",
				tc.health, tc.stamina, tc.conviction)
		})
	}
}

// TestRegression_AlignmentFullyRemoved guards against Stage 29.1 regression
// where alignment dead code could be reintroduced. The Character struct
// must not contain an Alignment field.
func TestRegression_AlignmentFullyRemoved(t *testing.T) {
	charType := reflect.TypeOf(Character{})

	// Verify no field named "Alignment" exists on the Character struct
	for i := 0; i < charType.NumField(); i++ {
		field := charType.Field(i)
		assert.NotEqual(t, "Alignment", field.Name,
			"Character struct must not have an Alignment field (removed in Stage 29.1)")
	}

	// Also verify no alignment-related YAML tags
	for i := 0; i < charType.NumField(); i++ {
		field := charType.Field(i)
		yamlTag := field.Tag.Get("yaml")
		assert.NotContains(t, yamlTag, "alignment",
			"No YAML tags should reference alignment (field: %s)", field.Name)
	}
}

// TestRegression_StatProgressionTriggersOnUse guards against Stage 4.5 bug
// where stats other than Vitality didn't increase via use-based progression.
// All six stats must be incrementable via IncreaseStat() and trackable via
// TrackStatUse(). This ensures the progression pipeline handles every stat.
func TestRegression_StatProgressionTriggersOnUse(t *testing.T) {
	statNames := []string{
		"strength", "dexterity", "perception",
		"vitality", "willpower", "charisma",
	}

	for _, statName := range statNames {
		t.Run(statName+"_increase", func(t *testing.T) {
			c := New()
			c.Stats.Strength.Base = 100
			c.Stats.Dexterity.Base = 100
			c.Stats.Perception.Base = 100
			c.Stats.Vitality.Base = 100
			c.Stats.Willpower.Base = 100
			c.Stats.Charisma.Base = 100
			c.Validate()

			initialValue := c.GetStatValue(statName)

			// IncreaseStat must work for all six stats (the Stage 4.5 fix)
			ok := c.IncreaseStat(statName, 1)
			assert.True(t, ok,
				"IncreaseStat(%q) should succeed", statName)

			newValue := c.GetStatValue(statName)
			assert.Equal(t, initialValue+1, newValue,
				"%s should increase by 1 after IncreaseStat", statName)
		})

		t.Run(statName+"_tracking", func(t *testing.T) {
			c := New()
			assert.Equal(t, 0, c.GetStatUseCount(statName),
				"%s use count should start at 0", statName)

			c.TrackStatUse(statName)
			assert.Equal(t, 1, c.GetStatUseCount(statName),
				"%s use count should increment to 1", statName)

			// Multiple uses accumulate
			for i := 0; i < 10; i++ {
				c.TrackStatUse(statName)
			}
			assert.Equal(t, 11, c.GetStatUseCount(statName),
				"%s use count should accumulate", statName)
		})

		t.Run(statName+"_progression_chance", func(t *testing.T) {
			// Verify CalculateProgressionChance returns non-zero for
			// rank 0 — this is what enables stat progression
			chance := CalculateProgressionChance(0, StatSoftCap)
			assert.Greater(t, chance, 0.0,
				"Progression chance at rank 0 must be positive for %s", statName)
		})
	}
}
