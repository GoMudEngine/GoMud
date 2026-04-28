package mobs

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/stretchr/testify/assert"
)

// stampFoldAnchor must write the room ID into MiscData when the field is
// set. Unit-tested in isolation from the rest of the spawn pipeline so
// the test doesn't depend on world boot / Validate / RegisterMobShop.
func TestStampFoldAnchor_Stamps(t *testing.T) {
	c := characters.New()
	stampFoldAnchor(c, 4037)

	got := c.GetMiscData("fold-anchor-room")
	assert.Equal(t, 4037, got, "MiscData should hold the anchor room ID")
}

// stampFoldAnchor must be a no-op when the field is zero (the YAML default)
// — otherwise every mob would get a spurious anchor at room 0.
func TestStampFoldAnchor_NoOpWhenZero(t *testing.T) {
	c := characters.New()
	stampFoldAnchor(c, 0)

	got := c.GetMiscData("fold-anchor-room")
	assert.Nil(t, got, "MiscData must NOT be set when FoldAnchorRoom is zero")
}

// stampFoldAnchor must also no-op for negative values (defensive — YAML
// authors might typo a negative).
func TestStampFoldAnchor_NoOpWhenNegative(t *testing.T) {
	c := characters.New()
	stampFoldAnchor(c, -1)

	got := c.GetMiscData("fold-anchor-room")
	assert.Nil(t, got, "MiscData must NOT be set for negative anchor IDs")
}
