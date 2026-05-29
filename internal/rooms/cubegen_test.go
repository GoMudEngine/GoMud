package rooms

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	assert.Equal(t, 124, cubeIndex(4, 4, 4)) // max for 5x5x5
	// x*25 + y*5 + z
	assert.Equal(t, 25, cubeIndex(1, 0, 0))
	assert.Equal(t, 5, cubeIndex(0, 1, 0))
	assert.Equal(t, 1, cubeIndex(0, 0, 1))
}

func TestCubeTitle(t *testing.T) {
	assert.Equal(t, "Elemental Depths", cubeTitle(0))
	assert.Equal(t, "Lower Wastes", cubeTitle(1))
	assert.Equal(t, "Upper Wastes", cubeTitle(2))
	assert.Equal(t, "Elemental Heights", cubeTitle(3))
	assert.Equal(t, "Elemental Apex", cubeTitle(4))
	assert.Equal(t, "Planar Wastes", cubeTitle(5))  // out of range
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
	// Room at (0,0,0): north -> (0,1,0), south wraps -> (0,4,0)
	northIdx := cubeIndex(0, wrapCoord(0+1, cubeSize), 0)
	assert.Equal(t, cubeIndex(0, 1, 0), northIdx)

	southIdx := cubeIndex(0, wrapCoord(0-1, cubeSize), 0)
	assert.Equal(t, cubeIndex(0, 4, 0), southIdx) // wraps

	// Room at (4,4,4): east wraps -> (0,4,4), up wraps -> (4,4,0)
	eastIdx := cubeIndex(wrapCoord(4+1, cubeSize), 4, 4)
	assert.Equal(t, cubeIndex(0, 4, 4), eastIdx)

	upIdx := cubeIndex(4, 4, wrapCoord(4+1, cubeSize))
	assert.Equal(t, cubeIndex(4, 4, 0), upIdx)
}

func TestCubeEntryRoomIndex(t *testing.T) {
	// Entry at (2,2,0)
	entryIdx := cubeIndex(2, 2, 0)
	assert.Equal(t, 2*cubeSize*cubeSize+2*cubeSize+0, entryIdx) // 2*25+2*5+0 = 60
}

func TestGenerateOasisCube_AllBossesDistinctRooms(t *testing.T) {
	// Seed a roomManager containing only the threshold room (5003).
	threshold := &Room{
		RoomId:    5003,
		Zone:      "Instance Planar Oasis",
		Exits:     make(map[string]exit.RoomExit),
		ExitsTemp: make(map[string]exit.TemporaryRoomExit),
	}
	cleanup := SeedRoomsForTest(
		map[int]*Room{5003: threshold},
		map[string]*ZoneConfig{},
	)
	defer cleanup()
	// Reset ephemeral state so the test is isolated and re-runnable.
	defer func() {
		for i := range ephemeralRoomChunks {
			ephemeralRoomChunks[i] = nil
		}
		originalRoomIdLookups = map[int]int{}
	}()

	roomIds, _, err := GenerateOasisCube(5003, "Instance Planar Oasis", 500, 1, true, "rejoin")
	require.NoError(t, err)
	assert.Len(t, roomIds, 125, "5x5x5 cube should have 125 rooms")

	bossIds := []int{320, 321, 322, 377}
	count := map[int]int{}
	roomOf := map[int]int{}
	for _, rid := range roomIds {
		r := LoadRoom(rid)
		require.NotNil(t, r)
		for _, si := range r.SpawnInfo {
			for _, bid := range bossIds {
				if si.MobId == bid {
					count[bid]++
					roomOf[bid] = rid
				}
			}
		}
	}

	for _, bid := range bossIds {
		assert.Equal(t, 1, count[bid], "boss %d should spawn exactly once", bid)
	}

	seen := map[int]bool{}
	for _, bid := range bossIds {
		assert.False(t, seen[roomOf[bid]], "bosses must be in distinct rooms")
		seen[roomOf[bid]] = true
	}
}
