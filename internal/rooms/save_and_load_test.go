package rooms

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/stretchr/testify/assert"
)

// ─── restoreSkipTaggedFields ────────────────────────────────────────────────

// TestRestoreSkipTaggedFields verifies the helper copies every field
// tagged `instance:"skip"` from src onto dst, leaves non-skip fields
// on dst alone.
func TestRestoreSkipTaggedFields(t *testing.T) {
	// src = "template" values
	src := &Room{
		RoomId:      100,
		Zone:        "template_zone",
		Title:       "Template Title",
		Description: "Template description.",
		Exits: map[string]exit.RoomExit{
			"north": {RoomId: 200},
			"south": {RoomId: 300},
		},
		Nouns: map[string]string{"altar": "A stone altar."},
		Gold:  0, // non-skip field; runtime state
	}

	// dst = "post-unmarshal" values with corrupt skip fields + legit
	// non-skip runtime state.
	dst := &Room{
		RoomId:      100,
		Zone:        "corrupt_zone",   // skip-tagged; should be restored
		Title:       "Corrupt Title",  // skip-tagged; should be restored
		Description: "Corrupt.",       // skip-tagged; should be restored
		Exits: map[string]exit.RoomExit{
			"east": {RoomId: 999}, // skip-tagged; should be restored
		},
		Nouns: map[string]string{"poison": "Bad data."}, // skip-tagged
		Gold:  42,                                       // NOT skip-tagged; should stay
	}

	restoreSkipTaggedFields(dst, src)

	// Skip-tagged fields now match src
	assert.Equal(t, "template_zone", dst.Zone)
	assert.Equal(t, "Template Title", dst.Title)
	assert.Equal(t, "Template description.", dst.Description)
	assert.Equal(t, 2, len(dst.Exits))
	assert.Equal(t, 200, dst.Exits["north"].RoomId)
	assert.Equal(t, 300, dst.Exits["south"].RoomId)
	_, hasEast := dst.Exits["east"]
	assert.False(t, hasEast, "east should have been replaced by template exits")
	assert.Equal(t, "A stone altar.", dst.Nouns["altar"])
	_, hasPoison := dst.Nouns["poison"]
	assert.False(t, hasPoison, "poison noun should have been replaced")

	// Non-skip field preserved
	assert.Equal(t, 42, dst.Gold)
}
