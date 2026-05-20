package characters_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/stretchr/testify/assert"
)

func TestParseSubmissionPolicy(t *testing.T) {
	cases := map[string]characters.SubmissionPolicy{
		"mercy":   characters.PolicyMercy,
		"subdue":  characters.PolicySubdue,
		"cripple": characters.PolicyCripple,
		"lethal":  characters.PolicyLethal,
		"MERCY":   characters.PolicyMercy, // case-insensitive
		"":        characters.PolicySubdue, // default
	}
	for input, want := range cases {
		got, ok := characters.ParseSubmissionPolicy(input)
		assert.True(t, ok || input != "", "parse %q", input)
		assert.Equal(t, want, got)
	}
	_, ok := characters.ParseSubmissionPolicy("eviscerate")
	assert.False(t, ok, "unknown mode rejected")
}

func TestParseSurrenderPolicy(t *testing.T) {
	tests := []struct {
		input      string
		wantMode   characters.SurrenderMode
		wantThresh int
	}{
		{"never", characters.SurrenderNever, 0},
		{"always", characters.SurrenderAlways, 0},
		{"auto-tap-below 15", characters.SurrenderAutoTap, 15},
		{"auto-tap-below 50", characters.SurrenderAutoTap, 50},
	}
	for _, tc := range tests {
		p, ok := characters.ParseSurrenderPolicy(tc.input)
		assert.True(t, ok, "parse %q", tc.input)
		assert.Equal(t, tc.wantMode, p.Mode)
		assert.Equal(t, tc.wantThresh, p.HpPctThreshold)
	}
	_, ok := characters.ParseSurrenderPolicy("bogus")
	assert.False(t, ok)
	_, ok = characters.ParseSurrenderPolicy("auto-tap-below 150")
	assert.False(t, ok, "out-of-range HP pct rejected")
}

func TestDefaultSubmissionPolicyForArchetype(t *testing.T) {
	cases := map[string]characters.SubmissionPolicy{
		"predator":         characters.PolicySubdue,
		"bandit":           characters.PolicySubdue,
		"guard":            characters.PolicySubdue,
		"defensive_caster": characters.PolicyMercy,
		"tank_taunter":     characters.PolicySubdue,
		"leader":           characters.PolicyCripple,
		"generic_fighter":  characters.PolicySubdue,
		"civilian":         characters.PolicyMercy,
		"lookout":          characters.PolicySubdue,
		"":                 characters.PolicySubdue, // unknown → default
		"some_unrecognized": characters.PolicySubdue,
	}
	for arch, want := range cases {
		got := characters.DefaultSubmissionPolicyForArchetype(arch)
		assert.Equal(t, want, got, "archetype %q", arch)
	}
}

func TestDefaultSurrenderPolicyForArchetype(t *testing.T) {
	predator := characters.DefaultSurrenderPolicyForArchetype("predator")
	assert.Equal(t, characters.SurrenderNever, predator.Mode)

	civilian := characters.DefaultSurrenderPolicyForArchetype("civilian")
	assert.Equal(t, characters.SurrenderAlways, civilian.Mode)

	generic := characters.DefaultSurrenderPolicyForArchetype("generic_fighter")
	assert.Equal(t, characters.SurrenderAutoTap, generic.Mode)
	assert.Equal(t, 10, generic.HpPctThreshold)
}
