package mapper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildTestMapper creates a simple 3-room chain: 1 -east-> 2 -south-> 3
func buildTestMapper() *mapper {
	m := NewMapper(1)
	m.crawledRooms = map[int]*mapNode{
		1: {RoomId: 1, Pos: positionDelta{x: 0, y: 0, z: 0}, Exits: map[string]nodeExit{"east": {RoomId: 2}}},
		2: {RoomId: 2, Pos: positionDelta{x: 1, y: 0, z: 0}, Exits: map[string]nodeExit{"south": {RoomId: 3}}},
		3: {RoomId: 3, Pos: positionDelta{x: 1, y: 1, z: 0}, Exits: map[string]nodeExit{}},
	}
	return m
}

// buildCycleMapper creates a graph with a cycle: 1 <-> 2 -> 3
func buildCycleMapper() *mapper {
	m := NewMapper(1)
	m.crawledRooms = map[int]*mapNode{
		1: {RoomId: 1, Pos: positionDelta{x: 0, y: 0, z: 0}, Exits: map[string]nodeExit{"to2": {RoomId: 2}}},
		2: {RoomId: 2, Pos: positionDelta{x: 1, y: 0, z: 0}, Exits: map[string]nodeExit{"to1": {RoomId: 1}, "to3": {RoomId: 3}}},
		3: {RoomId: 3, Pos: positionDelta{x: 2, y: 0, z: 0}, Exits: map[string]nodeExit{}},
	}
	return m
}

func Test_findPath_DestMatch(t *testing.T) {
	m := buildTestMapper()
	_, err := m.findPath(1, 1)
	assert.ErrorIs(t, err, ErrPathDestMatch)
}

func Test_findPath_RoomNotFound(t *testing.T) {
	m := NewMapper(1)
	m.crawledRooms = map[int]*mapNode{
		1: {RoomId: 1, Pos: positionDelta{}, Exits: map[string]nodeExit{}},
	}

	_, err := m.findPath(1, 2)
	assert.ErrorIs(t, err, ErrRoomNotFound, "expected ErrRoomNotFound for missing dest")

	_, err = m.findPath(2, 1)
	assert.ErrorIs(t, err, ErrRoomNotFound, "expected ErrRoomNotFound for missing source")
}

func Test_findPath_Simple(t *testing.T) {
	m := buildTestMapper()
	path, err := m.findPath(1, 3)
	require.NoError(t, err)
	require.Len(t, path, 2)

	assert.Equal(t, "east", path[0].exitName)
	assert.Equal(t, 2, path[0].roomId)
	assert.False(t, path[0].waypoint)

	assert.Equal(t, "south", path[1].exitName)
	assert.Equal(t, 3, path[1].roomId)
	assert.True(t, path[1].waypoint)
}

func Test_findPath_Unreachable(t *testing.T) {
	m := NewMapper(1)
	m.crawledRooms = map[int]*mapNode{
		1: {RoomId: 1, Pos: positionDelta{}, Exits: map[string]nodeExit{}},
		2: {RoomId: 2, Pos: positionDelta{x: 1}, Exits: map[string]nodeExit{}},
	}

	_, err := m.findPath(1, 2)
	assert.ErrorIs(t, err, ErrPathNotFound)
}

func Test_findPath_Cycle(t *testing.T) {
	m := buildCycleMapper()
	path, err := m.findPath(1, 3)
	require.NoError(t, err)
	require.Len(t, path, 2)

	assert.Equal(t, "to2", path[0].exitName)
	assert.Equal(t, 2, path[0].roomId)

	assert.Equal(t, "to3", path[1].exitName)
	assert.Equal(t, 3, path[1].roomId)
}

func Test_GetPath_StartNotFound(t *testing.T) {
	_, err := GetPath(999999)
	assert.ErrorIs(t, err, ErrPathNotFound, "expected ErrPathNotFound for missing start")
}

func Test_firstHop(t *testing.T) {
	n, dir, ok := firstHop([]pathStep{
		{exitName: "east", roomId: 2},
		{exitName: "south", roomId: 3},
	})
	require.True(t, ok)
	assert.Equal(t, 2, n)
	assert.Equal(t, "east", dir)

	_, _, ok = firstHop(nil)
	assert.False(t, ok)

	_, _, ok = firstHop([]pathStep{})
	assert.False(t, ok)
}

func Test_NextStep_SameRoom(t *testing.T) {
	_, _, ok := NextStep(5, 5)
	assert.False(t, ok, "same-room target yields no next step")
}

func Test_NextStep_NoPath(t *testing.T) {
	// Unknown start room => GetPath errors => no next step.
	_, _, ok := NextStep(999999, 4)
	assert.False(t, ok)
}
