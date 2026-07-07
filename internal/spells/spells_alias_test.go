package spells

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSpellAliasResolution verifies ResolveSpell/ResolveSpellId resolve a token
// by canonical id, then alias, then full display name.
func TestSpellAliasResolution(t *testing.T) {
	allSpells = map[string]*SpellData{
		"conviction-ward": {
			SpellId: "conviction-ward",
			Name:    "Conviction Ward",
			Aliases: []string{"ward"},
		},
	}
	buildSpellAliasIndex()

	tests := []struct {
		name  string
		token string
		want  string
	}{
		{"canonical id", "conviction-ward", "conviction-ward"},
		{"alias", "ward", "conviction-ward"},
		{"full display name", "conviction ward", "conviction-ward"},
		{"no match", "nonesuch", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ResolveSpellId(tt.token))
		})
	}
}
