package rooms

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrapCoord(t *testing.T) {
	assert.Equal(t, 0, wrapCoord(4, 4))
	assert.Equal(t, 3, wrapCoord(-1, 4))
	assert.Equal(t, 2, wrapCoord(2, 4))
	assert.Equal(t, 0, wrapCoord(0, 4))
	assert.Equal(t, 3, wrapCoord(3, 4))
	assert.Equal(t, 2, wrapCoord(-2, 4))
	assert.Equal(t, 0, wrapCoord(8, 4))
}

func TestCubeIndex(t *testing.T) {
	assert.Equal(t, 0, cubeIndex(0, 0, 0))
	assert.Equal(t, 63, cubeIndex(3, 3, 3)) // max for 4x4x4
	// x*16 + y*4 + z
	assert.Equal(t, 16, cubeIndex(1, 0, 0))
	assert.Equal(t, 4, cubeIndex(0, 1, 0))
	assert.Equal(t, 1, cubeIndex(0, 0, 1))
}

func TestCubeTitle(t *testing.T) {
	assert.Equal(t, "Elemental Depths", cubeTitle(0))
	assert.Equal(t, "Lower Wastes", cubeTitle(1))
	assert.Equal(t, "Upper Wastes", cubeTitle(2))
	assert.Equal(t, "Elemental Heights", cubeTitle(3))
	assert.Equal(t, "Planar Wastes", cubeTitle(4))  // out of range
	assert.Equal(t, "Planar Wastes", cubeTitle(-1)) // out of range
}

func TestPickUniqueIndices(t *testing.T) {
	result := pickUniqueIndices(3, 10, nil)
	assert.Len(t, result, 3)

	seen := map[int]bool{}
	for _, v := range result {
		assert.False(t, seen[v], "duplicate index %d", v)
		seen[v] = true
		assert.True(t, v >= 0 && v < 10, "index %d out of range", v)
	}

	exclude := []int{0, 1, 2, 3, 4, 5, 6, 7}
	result = pickUniqueIndices(2, 10, exclude)
	assert.Len(t, result, 2)
	for _, v := range result {
		assert.True(t, v == 8 || v == 9, "expected 8 or 9, got %d", v)
	}
}

func TestCubeWrappingExits(t *testing.T) {
	// Room at (0,0,0): north → (0,1,0), south wraps → (0,3,0)
	northIdx := cubeIndex(0, wrapCoord(0+1, cubeSize), 0)
	assert.Equal(t, cubeIndex(0, 1, 0), northIdx)

	southIdx := cubeIndex(0, wrapCoord(0-1, cubeSize), 0)
	assert.Equal(t, cubeIndex(0, 3, 0), southIdx) // wraps

	// Room at (3,3,3): east wraps → (0,3,3), up wraps → (3,3,0)
	eastIdx := cubeIndex(wrapCoord(3+1, cubeSize), 3, 3)
	assert.Equal(t, cubeIndex(0, 3, 3), eastIdx)

	upIdx := cubeIndex(3, 3, wrapCoord(3+1, cubeSize))
	assert.Equal(t, cubeIndex(3, 3, 0), upIdx)
}

func TestCubeEntryRoomIndex(t *testing.T) {
	// Entry at (2,2,0)
	entryIdx := cubeIndex(2, 2, 0)
	assert.Equal(t, 2*cubeSize*cubeSize+2*cubeSize+0, entryIdx) // 2*16+2*4+0 = 40
}
