package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/stretchr/testify/assert"
)

// validateFoldRecall must reject when no anchor has been set.
func TestValidateFoldRecall_NoAnchor_Rejects(t *testing.T) {
	c := characters.New()
	c.RoomId = 4036

	a := &fakeActor{char: c, name: "TestPlayer", isPlayer: true, userId: 42}

	ok := validateFoldRecall(a)

	assert.False(t, ok, "validate must reject when no anchor exists")
	assert.NotEmpty(t, a.selfTexts, "actor should be told why it failed")
}

// validateFoldRecall must reject when the actor is already in the anchor room.
func TestValidateFoldRecall_AlreadyAtAnchor_Rejects(t *testing.T) {
	c := characters.New()
	c.RoomId = 4037
	c.SetMiscData("fold-anchor-room", 4037)

	a := &fakeActor{char: c, name: "TestPlayer", isPlayer: true, userId: 42}

	ok := validateFoldRecall(a)

	assert.False(t, ok, "validate must reject when actor is already on the anchor")
	assert.NotEmpty(t, a.selfTexts, "actor should be told why it failed")
}

// validateFoldRecall passes when an anchor is set and not the current room.
func TestValidateFoldRecall_HappyPath_Passes(t *testing.T) {
	c := characters.New()
	c.RoomId = 4036
	c.SetMiscData("fold-anchor-room", 4037)

	a := &fakeActor{char: c, name: "TestPlayer", isPlayer: true, userId: 42}

	ok := validateFoldRecall(a)

	assert.True(t, ok, "validate must pass when anchor exists and is not the current room")
}

// resolveFoldRecall: short-circuit when no anchor is set (defensive — validate
// should have caught it). Confirm the resolver doesn't blow up.
func TestResolveFoldRecall_NoAnchor_NoOp(t *testing.T) {
	c := characters.New()
	c.RoomId = 4036

	a := &fakeActor{char: c, name: "TestPlayer", isPlayer: true, userId: 42}

	// must not panic
	resolveFoldRecall(a)

	assert.NotEmpty(t, a.selfTexts, "actor should receive a 'fold collapses' message")
}
