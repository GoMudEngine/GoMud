package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Test-local Actor stub (reuses stubActor from economy_test.go which is also
// in package actions, so it's already in scope for test binaries).
// ---------------------------------------------------------------------------

// newCastActor is a convenience wrapper that builds a stubActor with default
// character and a minimal empty room.
func newCastActor() (*stubActor, *characters.Character, *rooms.Room) {
	char := newTestChar()
	room := newTestRoom()
	actor := newStubActor(char, room)
	return actor, char, room
}

// seedTestSpell registers a minimal spell in the global spell registry and
// returns a cleanup function that restores the original registry.
// spellId must be unique within the test binary.
func seedTestSpell(spellId string, spellType spells.SpellType, baseFolds int) (*spells.SpellData, func()) {
	sd := &spells.SpellData{
		SpellId:   spellId,
		Name:      "Test " + spellId,
		Type:      spellType,
		BaseFolds: baseFolds,
		Cost:      5,
	}
	cleanup := spells.SeedSpellsForTest(map[string]*spells.SpellData{
		spellId: sd,
	})
	return sd, cleanup
}

// ---------------------------------------------------------------------------
// TestInitiateCast_InvalidSpell
// ---------------------------------------------------------------------------

// TestInitiateCast_InvalidSpell verifies that requesting an unknown spell name
// returns InvalidSpell=true and Initiated=false with no other flags set.
func TestInitiateCast_InvalidSpell(t *testing.T) {
	actor, _, _ := newCastActor()

	result := InitiateCast(actor, "no-such-spell-xyzzy", "")

	assert.True(t, result.InvalidSpell, "InvalidSpell should be set for unknown spell")
	assert.False(t, result.Initiated, "Initiated should be false")
	assert.False(t, result.AlreadyCasting)
	assert.False(t, result.OnCooldown)
	assert.False(t, result.SpellNotKnown)
	assert.False(t, result.NoTarget)
	assert.Nil(t, result.CastingState)
}

// ---------------------------------------------------------------------------
// TestInitiateCast_AlreadyCasting
// ---------------------------------------------------------------------------

// TestInitiateCast_AlreadyCasting verifies that if the character already has
// an active CastingState the function returns AlreadyCasting=true.
func TestInitiateCast_AlreadyCasting(t *testing.T) {
	sd, cleanup := seedTestSpell("test-already", spells.HelpSingle, 4)
	defer cleanup()

	actor, char, _ := newCastActor()
	// Pre-set an active casting state.
	char.CastingState = &characters.CastingState{
		SpellId:     "test-already",
		FoldsNeeded: 4,
	}

	result := InitiateCast(actor, sd.SpellId, "")

	assert.True(t, result.AlreadyCasting, "AlreadyCasting should be set")
	assert.False(t, result.Initiated, "Initiated should be false")
	assert.NotNil(t, result.SpellInfo, "SpellInfo should still be populated")
	assert.Nil(t, result.CastingState, "CastingState should not be built")
}

// ---------------------------------------------------------------------------
// TestInitiateCast_OnCooldown
// ---------------------------------------------------------------------------

// TestInitiateCast_OnCooldown verifies that if the special-move cooldown is
// already active the function returns OnCooldown=true.
func TestInitiateCast_OnCooldown(t *testing.T) {
	sd, cleanup := seedTestSpell("test-cooldown", spells.HelpSingle, 4)
	defer cleanup()

	actor, char, _ := newCastActor()
	// Consume the special-move cooldown slot by pre-setting a positive round count.
	// TryCooldown returns false when the key is already present with rounds > 0.
	char.Cooldowns["special-move"] = 5

	result := InitiateCast(actor, sd.SpellId, "")

	assert.True(t, result.OnCooldown, "OnCooldown should be set")
	assert.False(t, result.Initiated, "Initiated should be false")
	assert.NotNil(t, result.SpellInfo, "SpellInfo should still be populated")
}

// ---------------------------------------------------------------------------
// TestInitiateCast_FoldsCalculation
// ---------------------------------------------------------------------------

// TestInitiateCast_FoldsCalculation verifies that the computed FoldsNeeded
// equals NextPowerOfTwo(BaseFolds) and that FoldsPerRound matches the output
// of CalcFoldsPerRound for the character's stats.
func TestInitiateCast_FoldsCalculation(t *testing.T) {
	const baseFolds = 6
	sd, cleanup := seedTestSpell("test-folds", spells.Neutral, baseFolds)
	defer cleanup()

	actor, char, _ := newCastActor()

	result := InitiateCast(actor, sd.SpellId, "")

	require.True(t, result.Initiated, "expected successful initiation")

	expectedFoldsNeeded := characters.NextPowerOfTwo(baseFolds)
	assert.Equal(t, expectedFoldsNeeded, result.FoldsNeeded,
		"FoldsNeeded should equal NextPowerOfTwo(BaseFolds)")

	expectedFoldsPerRound := characters.CalcFoldsPerRound(
		char.Stats.Perception.ValueAdj,
		char.GetSkillLevel("spellcasting"),
	)
	assert.Equal(t, expectedFoldsPerRound, result.FoldsPerRound,
		"FoldsPerRound should match CalcFoldsPerRound output")
}

// TestInitiateCast_FoldsCalculation_DefaultBaseFolds verifies that a spell
// with BaseFolds=0 (the zero value) defaults to 4 before the NextPowerOfTwo
// call, so FoldsNeeded == 4.
func TestInitiateCast_FoldsCalculation_DefaultBaseFolds(t *testing.T) {
	sd, cleanup := seedTestSpell("test-folds-default", spells.Neutral, 0)
	defer cleanup()

	actor, _, _ := newCastActor()

	result := InitiateCast(actor, sd.SpellId, "")

	require.True(t, result.Initiated, "expected successful initiation")
	// BaseFolds 0 → treated as 4 → NextPowerOfTwo(4) == 4
	assert.Equal(t, 4, result.FoldsNeeded, "default BaseFolds=0 should yield FoldsNeeded=4")
}

// ---------------------------------------------------------------------------
// TestInitiateCast_CastingStateBuilt
// ---------------------------------------------------------------------------

// TestInitiateCast_CastingStateBuilt verifies that a fully successful call
// produces a non-nil CastingState whose fields mirror the result values, and
// that Initiated is true.
func TestInitiateCast_CastingStateBuilt(t *testing.T) {
	const baseFolds = 4
	sd, cleanup := seedTestSpell("test-built", spells.Neutral, baseFolds)
	defer cleanup()

	actor, _, _ := newCastActor()

	result := InitiateCast(actor, sd.SpellId, "some rest text")

	require.True(t, result.Initiated, "expected successful initiation")
	require.NotNil(t, result.CastingState, "CastingState must be non-nil on success")

	cs := result.CastingState
	assert.Equal(t, sd.SpellId, cs.SpellId,
		"CastingState.SpellId should match the spell")
	assert.Equal(t, result.FoldsNeeded, cs.FoldsNeeded,
		"CastingState.FoldsNeeded should match result")
	assert.Equal(t, result.FoldsPerRound, cs.FoldsPerRound,
		"CastingState.FoldsPerRound should match result")
	assert.Equal(t, result.TotalCost, cs.TotalConvictionCost,
		"CastingState.TotalConvictionCost should match result.TotalCost")
	assert.Equal(t, 0, cs.FoldsAccumulated,
		"FoldsAccumulated must start at zero")
	assert.Equal(t, 0, cs.ConvictionSpent,
		"ConvictionSpent must start at zero")
	assert.Equal(t, "some rest text", cs.SpellRest,
		"Neutral spell rest text should propagate to CastingState")
	// CastingState must NOT already be applied to the character — that is the
	// caller's responsibility (see InitiateCast doc comment).
	assert.Nil(t, actor.char.CastingState,
		"CastingState must NOT be applied to char by InitiateCast itself")
}

// ---------------------------------------------------------------------------
// TestInitiateCast_CostPropagation
// ---------------------------------------------------------------------------

// TestInitiateCast_CostPropagation verifies that TotalCost on the result
// equals the spell's Cost field (no multiplier is applied inside InitiateCast).
func TestInitiateCast_CostPropagation(t *testing.T) {
	sd, cleanup := seedTestSpell("test-cost", spells.Neutral, 4)
	defer cleanup()
	sd.Cost = 42

	actor, _, _ := newCastActor()
	result := InitiateCast(actor, sd.SpellId, "")

	require.True(t, result.Initiated)
	assert.Equal(t, 42, result.TotalCost,
		"TotalCost should equal the spell's raw Cost field")
	assert.Equal(t, 42, result.CastingState.TotalConvictionCost,
		"CastingState.TotalConvictionCost should equal the spell's raw Cost field")
}
