package mobai

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetMemory(t *testing.T) {
	mem := SetMemory(5, 0, 100, 1000)
	assert.Equal(t, 5, mem.TargetUserId)
	assert.Equal(t, 100, mem.LastSeenRoomId)
	assert.Equal(t, uint64(1000), mem.LastSeenRound)
	assert.True(t, mem.Grudge)
}

func TestMemoryExpired(t *testing.T) {
	mem := SetMemory(5, 0, 100, 1000)
	assert.False(t, MemoryExpired(mem, 1200, 300))
	assert.True(t, MemoryExpired(mem, 1301, 300))
}

func TestMemoryExpired_Nil(t *testing.T) {
	assert.True(t, MemoryExpired(nil, 1000, 300))
}

func TestUpdateMemoryLocation(t *testing.T) {
	mem := SetMemory(5, 0, 100, 1000)
	UpdateMemoryLocation(mem, 200, 1500)
	assert.Equal(t, 200, mem.LastSeenRoomId)
	assert.Equal(t, uint64(1500), mem.LastSeenRound)
}
