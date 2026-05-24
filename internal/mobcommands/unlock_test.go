package mobcommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Unlock smoke tests ───────────────────────────────────────────────────────

func TestUnlock_CommandRegistered(t *testing.T) {
	_, ok := mobCommands["unlock"]
	assert.True(t, ok, "unlock must be registered in mobCommands")
}

// ─── RoutesProperly: empty args returns (true, nil) silently ─────────────────

func TestUnlock_EmptyArgs_RoutesProperly(t *testing.T) {
	cleanup := seedLockFixtures(t)
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	handled, err := Unlock("", mob, room)
	assert.True(t, handled)
	assert.NoError(t, err)
}

// ─── NoTarget: container not found ───────────────────────────────────────────

func TestUnlock_NoTarget_HandledTrue(t *testing.T) {
	cleanup := seedLockFixtures(t)
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	handled, err := Unlock("nonexistent", mob, room)
	assert.True(t, handled)
	assert.NoError(t, err)
}

// ─── AlreadyUnlocked: silently no-ops ────────────────────────────────────────

func TestUnlock_AlreadyUnlocked_Noop(t *testing.T) {
	cleanup := seedLockFixtures(t)
	defer cleanup()

	mob, room := getTestMobAndRoom(t)
	addLockboxToRoom(room, false) // start unlocked (UnlockedRound=1, long relockInterval)

	// The container is "unlocked" (UnlockedRound != 0). Unlock should no-op.
	before := room.Containers["lockbox"].Lock.UnlockedRound
	handled, err := Unlock("lockbox", mob, room)
	assert.True(t, handled)
	assert.NoError(t, err)
	// UnlockedRound must not change — Unlock was a no-op on an already-unlocked container.
	assert.Equal(t, before, room.Containers["lockbox"].Lock.UnlockedRound,
		"already-unlocked container must not have its UnlockedRound changed")
}

// ─── NoKey: mob lacks matching key, silently no-ops ─────────────────────────

func TestUnlock_NoKey_Noop(t *testing.T) {
	cleanup := seedLockFixtures(t)
	defer cleanup()

	mob, room := getTestMobAndRoom(t)
	addLockboxToRoom(room, true) // start locked (UnlockedRound == 0)

	// Mob has no key — ensure backpack is empty.
	require.Empty(t, mob.Character.Items)

	handled, err := Unlock("lockbox", mob, room)
	assert.True(t, handled)
	assert.NoError(t, err)
	// UnlockedRound must remain 0 — mob couldn't unlock (no key).
	assert.Equal(t, uint64(0), room.Containers["lockbox"].Lock.UnlockedRound,
		"mob without key must not change lock state")
}

// ─── Success: mob has key, locked → unlocked (UnlockedRound stamped) ─────────

func TestUnlock_Success_ContainerBecomesUnlocked(t *testing.T) {
	cleanup := seedLockFixtures(t)
	defer cleanup()

	// Set a non-zero round count so SetUnlocked() stamps a non-zero UnlockedRound.
	util.SetRoundCountForTest(42)
	defer util.SetRoundCountForTest(0)

	addKeyToMob(t)
	mob, room := getTestMobAndRoom(t)
	addLockboxToRoom(room, true) // start locked (UnlockedRound == 0)

	handled, err := Unlock("lockbox", mob, room)
	assert.True(t, handled)
	assert.NoError(t, err)
	// SetUnlocked() stamps UnlockedRound = GetRoundCount() = 42.
	assert.Equal(t, uint64(42), room.Containers["lockbox"].Lock.UnlockedRound,
		"UnlockedRound must be stamped with current round after successful Unlock")
}

// ─── Key retained: mob keeps the key after unlocking ─────────────────────────

func TestUnlock_KeyRetainedAfterUnlock(t *testing.T) {
	cleanup := seedLockFixtures(t)
	defer cleanup()

	addKeyToMob(t)
	mob, room := getTestMobAndRoom(t)
	addLockboxToRoom(room, true)

	before := len(mob.Character.Items)
	_, _ = Unlock("lockbox", mob, room)
	assert.Equal(t, before, len(mob.Character.Items),
		"mob must retain its key after unlocking")
}
