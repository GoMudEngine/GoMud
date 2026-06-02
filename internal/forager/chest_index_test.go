package forager

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChestIndex_RegisterAndLookup(t *testing.T) {
	RegisterChestRoom("stillwater", 4198)
	RegisterChestRoom("stillwater", 4198) // idempotent
	RegisterChestRoom("stillwater", 4199)
	RegisterChestRoom("", 5)         // ignored
	RegisterChestRoom("ironwind", 0) // ignored
	assert.Equal(t, []int{4198, 4199}, ChestRoomsForZone("stillwater"))
	assert.Nil(t, ChestRoomsForZone("nowhere"))
}
