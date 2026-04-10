package rooms

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrapCoord(t *testing.T) {
	assert.Equal(t, 0, wrapCoord(5, 5))
	assert.Equal(t, 4, wrapCoord(-1, 5))
	assert.Equal(t, 2, wrapCoord(2, 5))
	assert.Equal(t, 0, wrapCoord(0, 5))
	assert.Equal(t, 4, wrapCoord(4, 5))
	assert.Equal(t, 3, wrapCoord(-2, 5))
	assert.Equal(t, 0, wrapCoord(10, 5))
}

func TestCubeIndex(t *testing.T) {
	assert.Equal(t, 0, cubeIndex(0, 0, 0))
	assert.Equal(t, 124, cubeIndex(4, 4, 4))
	assert.Equal(t, 62, cubeIndex(2, 2, 2)) // middle of cube

	// Verify x*25 + y*5 + z
	assert.Equal(t, 25, cubeIndex(1, 0, 0))
	assert.Equal(t, 5, cubeIndex(0, 1, 0))
	assert.Equal(t, 1, cubeIndex(0, 0, 1))
}

func TestCubeTitle(t *testing.T) {
	assert.Equal(t, "Elemental Depths", cubeTitle(0))
	assert.Equal(t, "Lower Wastes", cubeTitle(1))
	assert.Equal(t, "Middle Wastes", cubeTitle(2))
	assert.Equal(t, "Upper Wastes", cubeTitle(3))
	assert.Equal(t, "Elemental Heights", cubeTitle(4))
	assert.Equal(t, "Planar Wastes", cubeTitle(5)) // out of range
	assert.Equal(t, "Planar Wastes", cubeTitle(-1)) // out of range
}

func TestPickUniqueIndices(t *testing.T) {
	// Basic: pick 3 from 10
	result := pickUniqueIndices(3, 10, nil)
	assert.Len(t, result, 3)

	// All unique
	seen := map[int]bool{}
	for _, v := range result {
		assert.False(t, seen[v], "duplicate index %d", v)
		seen[v] = true
		assert.True(t, v >= 0 && v < 10, "index %d out of range", v)
	}

	// With exclusions
	exclude := []int{0, 1, 2, 3, 4, 5, 6, 7}
	result = pickUniqueIndices(2, 10, exclude)
	assert.Len(t, result, 2)
	for _, v := range result {
		assert.True(t, v == 8 || v == 9, "expected 8 or 9, got %d", v)
	}
}

func TestCubeWrappingExits(t *testing.T) {
	// Verify that wrapping arithmetic produces correct neighbor indices.

	// Room at (0,0,0): north should go to (0,1,0), south to (0,4,0)
	northIdx := cubeIndex(0, wrapCoord(0+1, 5), 0)
	assert.Equal(t, cubeIndex(0, 1, 0), northIdx)

	southIdx := cubeIndex(0, wrapCoord(0-1, 5), 0)
	assert.Equal(t, cubeIndex(0, 4, 0), southIdx) // wraps

	// Room at (4,4,4): east wraps to (0,4,4), up wraps to (4,4,0)
	eastIdx := cubeIndex(wrapCoord(4+1, 5), 4, 4)
	assert.Equal(t, cubeIndex(0, 4, 4), eastIdx) // wraps

	upIdx := cubeIndex(4, 4, wrapCoord(4+1, 5))
	assert.Equal(t, cubeIndex(4, 4, 0), upIdx) // wraps
}

func TestCubeEntryRoomIndex(t *testing.T) {
	// The entry room is at (2,2,0) — verify its index.
	entryIdx := cubeIndex(2, 2, 0)
	assert.Equal(t, 2*25+2*5+0, entryIdx) // 60
}
