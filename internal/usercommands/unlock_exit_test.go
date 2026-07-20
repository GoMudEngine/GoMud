package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/gamelock"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUnlockExit_MutatesCallerExitInfo pins the pointer-semantics contract that
// the three unlock branches in Go() depend on.
//
// GetExitInfo returns a COPY of the exit, and gamelock.Lock.SetUnlocked has a
// pointer receiver. The backpack-key branch re-reads exitInfo.Lock.IsLocked()
// after unlocking, to decide whether to show the "there's a lock" failure
// message. If unlockExit took exitInfo by value, that unlock would land on a
// throwaway copy, the re-read would still see "locked", and a player who just
// unlocked a door with a loose key would be wrongly told they can't pass.
//
// So this asserts the helper writes through to the caller's exitInfo, not that
// the door ends up readable-as-unlocked — the latter can't be checked in a unit
// test because gamelock relock timing consults gametime, which is not seeded
// here (AddPeriod returns the current round unchanged, so any unlock re-locks
// instantly).
func TestUnlockExit_MutatesCallerExitInfo(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	user, room := getTestUserAndRoom(t)

	// Advance the round counter so SetUnlocked stamps a non-zero UnlockedRound
	// (it records util.GetRoundCount()).
	util.SetRoundCountForTest(100)
	defer util.ResetRoundCountForTest()

	const exitName = "north"
	room.Exits = map[string]exit.RoomExit{
		exitName: {RoomId: 2, Lock: gamelock.Lock{Difficulty: 5}},
	}

	exitInfo, ok := room.GetExitInfo(exitName)
	require.True(t, ok, "precondition: the exit must resolve")
	require.Zero(t, exitInfo.Lock.UnlockedRound,
		"precondition: the local copy starts locked (UnlockedRound 0)")

	unlockExit(user, room, exitName, &exitInfo,
		"You pick the lock.",
		"Someone picks the lock.")

	assert.Equal(t, uint64(100), exitInfo.Lock.UnlockedRound,
		"unlockExit must SetUnlocked on the CALLER's exitInfo — the backpack branch "+
			"re-reads it to decide whether the unlock succeeded")

	// And it must persist to the room's real exit, not only the local copy.
	assert.Equal(t, uint64(100), room.Exits[exitName].Lock.UnlockedRound,
		"unlockExit must also clear the lock on the room's stored exit via SetExitLock")
}

// TestUnlockExit_NoOpOnNonRoomExit guards SetExitLock's contract: it silently
// does nothing for an exit the room doesn't have, so unlockExit must not panic
// on a mismatched name.
func TestUnlockExit_NoOpOnNonRoomExit(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	user, room := getTestUserAndRoom(t)

	room.Exits = map[string]exit.RoomExit{}

	local := exit.RoomExit{RoomId: 2, Lock: gamelock.Lock{Difficulty: 5}}

	assert.NotPanics(t, func() {
		unlockExit(user, room, "nonexistent", &local, "msg", "roommsg")
	}, "unlockExit must not panic when the room has no such exit")

}
