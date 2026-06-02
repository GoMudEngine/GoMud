package shops

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnchantReserve_AddDrainPool(t *testing.T) {
	ResetEnchantReserveForTest()
	AddToReserve(40028, 3)
	AddToReserve(40028, 2)
	AddToReserve(40030, 1)
	assert.Equal(t, map[int]int{40028: 5, 40030: 1}, ReservePool())

	DrainReserve(40028, 4)
	assert.Equal(t, 1, ReservePool()[40028])

	// Pool() is a copy — mutating it doesn't affect the reserve.
	p := ReservePool()
	p[40028] = 999
	assert.Equal(t, 1, ReservePool()[40028])

	AddToReserve(40028, 0) // no-op
	AddToReserve(0, 5)     // ignored (zero id)
	assert.Equal(t, 1, ReservePool()[40028])

	// Drain more than present removes the entry entirely.
	DrainReserve(40030, 99)
	_, ok := ReservePool()[40030]
	assert.False(t, ok)
}
