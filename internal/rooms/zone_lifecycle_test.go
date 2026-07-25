package rooms

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZoneFolderCollision(t *testing.T) {
	existing := []string{"Amber Valley", "Stillwater"}

	// Same folder (amber_valley), different display name -> collision.
	assert.Equal(t, "Amber Valley", ZoneFolderCollision("Amber_Valley", existing))
	assert.Equal(t, "Amber Valley", ZoneFolderCollision("amber valley", existing))

	// Genuinely new zone -> no collision.
	assert.Equal(t, "", ZoneFolderCollision("Thornwall", existing))

	// A name identical to an existing zone still reports it; CreateZone's
	// own duplicate check runs first, but this must not silently pass.
	assert.Equal(t, "Stillwater", ZoneFolderCollision("Stillwater", existing))
}
