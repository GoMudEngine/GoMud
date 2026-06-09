package characters

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormattedName_MobUsesSmartTitleCase(t *testing.T) {
	f := FormattedName{Name: "captain of the guard", Type: "mobname"}
	out := f.String()
	if !strings.Contains(out, "Captain of the Guard") {
		t.Errorf("mob name not smart-title-cased; got %q", out)
	}
	if strings.Contains(out, "Of The") {
		t.Errorf("minor words should be lowercased; got %q", out)
	}
}

func TestGetFormattedAdjectives(t *testing.T) {
	// Setup: populate adjectiveSwaps with test data
	adjectiveSwaps = map[string]string{
		"charmed":       "♥friend",
		"charmed-short": "♥",
		"hidden":        "hidden",
		"hidden-short":  "?",
		"zombie":        "zOmBie",
	}

	tests := []struct {
		name         string
		excludeShort bool
		expected     []string
	}{
		{
			name:         "Include short adjectives",
			excludeShort: false,
			expected:     []string{"charmed", "charmed-short", "hidden", "hidden-short", "zombie"},
		},
		{
			name:         "Exclude short adjectives",
			excludeShort: true,
			expected:     []string{"charmed", "hidden", "zombie"},
		},
		{
			name:         "Empty adjectiveSwaps",
			excludeShort: false,
			expected:     []string{},
		},
	}

	for _, tt := range tests[:2] {
		t.Run(tt.name, func(t *testing.T) {
			got := GetFormattedAdjectives(tt.excludeShort)
			assert.ElementsMatch(t, tt.expected, got)
		})
	}

	// Test with empty adjectiveSwaps
	adjectiveSwaps = map[string]string{}
	t.Run(tests[2].name, func(t *testing.T) {
		got := GetFormattedAdjectives(tests[2].excludeShort)
		assert.Empty(t, got)
	})
}
func TestGetFormattedAdjective(t *testing.T) {
	// Setup: populate adjectiveSwaps with test data
	adjectiveSwaps = map[string]string{
		"charmed":       "♥friend",
		"charmed-short": "♥",
		"hidden":        "hidden",
		"hidden-short":  "?",
		"zombie":        "zOmBie",
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Existing adjective",
			input:    "charmed",
			expected: "♥friend",
		},
		{
			name:     "Existing short adjective",
			input:    "charmed-short",
			expected: "♥",
		},
		{
			name:     "Non-existing adjective returns input",
			input:    "unknown",
			expected: "unknown",
		},
		{
			name:     "Another existing adjective",
			input:    "zombie",
			expected: "zOmBie",
		},
		{
			name:     "Another short adjective",
			input:    "hidden-short",
			expected: "?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetFormattedAdjective(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}

	// Test with empty adjectiveSwaps
	adjectiveSwaps = map[string]string{}
	t.Run("Empty adjectiveSwaps returns input", func(t *testing.T) {
		got := GetFormattedAdjective("charmed")
		assert.Equal(t, "charmed", got)
	})
}
