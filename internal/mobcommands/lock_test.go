package mobcommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/gamelock"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Lock smoke tests ─────────────────────────────────────────────────────────

func TestLock_CommandRegistered(t *testing.T) {
	_, ok := mobCommands["lock"]
	assert.True(t, ok, "lock must be registered in mobCommands")
}

// seedLockFixtures returns a room with a lockable container and a mob that
// carries a matching key. Callers may mutate the container state before
// calling Lock/Unlock.
func seedLockFixtures(t *testing.T) (cleanup func()) {
	t.Helper()
	baseCleanup := seedAllRegistries()

	// Seed a key item spec whose KeyLockId matches the container lockId.
	// Container lockId format: "<roomId>-<containerName>" → "1-lockbox"
	keyCleanup := items.SeedItemsForTest(map[int]*items.ItemSpec{
		500: {
			ItemId:    500,
			Name:      "Small Brass Key",
			Type:      items.Key,
			KeyLockId: "1-lockbox",
		},
	})

	return func() {
		keyCleanup()
		baseCleanup()
	}
}

// addLockboxToRoom adds a lockable container named "lockbox" to room 1.
// Pass locked=true for an initially-locked container.
//
// Note on gamelock semantics: IsLocked() returns true when Difficulty > 0 AND
// UnlockedRound == 0. For an "unlocked" state, we set UnlockedRound = 1 and
// a long RelockInterval so that even with an uninitialized gametime config the
// lock stays unlocked. Callers must also set a non-zero round count via
// util.SetRoundCountForTest so that SetUnlocked() stamps a non-zero value.
func addLockboxToRoom(room *rooms.Room, locked bool) {
	lk := gamelock.Lock{Difficulty: 1} // Difficulty > 0 means it has a lock.
	if !locked {
		// Represent unlocked: UnlockedRound=1 and a huge relock window so
		// IsLocked() path-3 evaluates to false regardless of RoundsPerDay.
		lk.UnlockedRound = 1
		lk.RelockInterval = "9999999 rounds" // far-future relock: won't relock in tests
	}
	room.Containers = map[string]rooms.Container{
		"lockbox": {
			Lock: lk,
		},
	}
}

// addKeyToMob gives mob instance 100 a key item matching "1-lockbox".
func addKeyToMob(t *testing.T) {
	t.Helper()
	mob, _ := getTestMobAndRoom(t)
	key := items.Item{}
	key.ItemId = 500
	mob.Character.StoreItem(key)
}

// ─── RoutesProperly: empty args returns (true, nil) silently ─────────────────

func TestLock_EmptyArgs_RoutesProperly(t *testing.T) {
	cleanup := seedLockFixtures(t)
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	handled, err := Lock("", mob, room)
	assert.True(t, handled)
	assert.NoError(t, err)
}

// ─── NoTarget: container not found ───────────────────────────────────────────

func TestLock_NoTarget_HandledTrue(t *testing.T) {
	cleanup := seedLockFixtures(t)
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	handled, err := Lock("nonexistent", mob, room)
	assert.True(t, handled)
	assert.NoError(t, err)
}

// ─── AlreadyLocked: silently no-ops ─────────────────────────────────────────

func TestLock_AlreadyLocked_Noop(t *testing.T) {
	cleanup := seedLockFixtures(t)
	defer cleanup()

	mob, room := getTestMobAndRoom(t)
	addLockboxToRoom(room, true) // start locked (RotationSeed == 1)

	seedBefore := room.Containers["lockbox"].Lock.RotationSeed
	handled, err := Lock("lockbox", mob, room)
	assert.True(t, handled)
	assert.NoError(t, err)
	// RotationSeed must not change — no SetLocked called.
	assert.Equal(t, seedBefore, room.Containers["lockbox"].Lock.RotationSeed,
		"already-locked container must not have its RotationSeed bumped")
}

// ─── NoKey: mob lacks matching key, silently no-ops ─────────────────────────

func TestLock_NoKey_Noop(t *testing.T) {
	cleanup := seedLockFixtures(t)
	defer cleanup()

	mob, room := getTestMobAndRoom(t)
	// Unlocked state: UnlockedRound non-zero means SetUnlocked was called.
	addLockboxToRoom(room, false) // start unlocked

	// Mob has no key — ensure backpack is empty of keys.
	require.Empty(t, mob.Character.Items)

	before := room.Containers["lockbox"].Lock.RotationSeed
	handled, err := Lock("lockbox", mob, room)
	assert.True(t, handled)
	assert.NoError(t, err)
	// RotationSeed must not change — mob had no key, no SetLocked called.
	assert.Equal(t, before, room.Containers["lockbox"].Lock.RotationSeed,
		"mob without key must not change lock rotation seed")
}

// ─── Success: mob has key, unlocked → locked (RotationSeed incremented) ──────

func TestLock_Success_ContainerBecomesLocked(t *testing.T) {
	cleanup := seedLockFixtures(t)
	defer cleanup()

	addKeyToMob(t)
	mob, room := getTestMobAndRoom(t)
	addLockboxToRoom(room, false) // start unlocked (UnlockedRound != 0)

	before := room.Containers["lockbox"].Lock.RotationSeed
	handled, err := Lock("lockbox", mob, room)
	assert.True(t, handled)
	assert.NoError(t, err)
	// SetLocked() increments RotationSeed — this verifies the lock was applied.
	assert.Greater(t, room.Containers["lockbox"].Lock.RotationSeed, before,
		"RotationSeed must increment when mob successfully locks container")
}

// ─── Key retained: mob keeps the key after locking ──────────────────────────

func TestLock_KeyRetainedAfterLock(t *testing.T) {
	cleanup := seedLockFixtures(t)
	defer cleanup()

	addKeyToMob(t)
	mob, room := getTestMobAndRoom(t)
	addLockboxToRoom(room, false)

	before := len(mob.Character.Items)
	_, _ = Lock("lockbox", mob, room)
	assert.Equal(t, before, len(mob.Character.Items),
		"mob must retain its key after locking")
}
