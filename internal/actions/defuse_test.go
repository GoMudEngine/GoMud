package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/gamelock"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/statmods"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// SendTextVisual calls GetBiome which requires at least a default biome to
	// be registered. SeedBiomesForTest injects a minimal in-memory registry
	// without touching the filesystem. The returned cleanup is intentionally
	// not called — the seeded biomes should persist for the whole test run.
	rooms.SeedBiomesForTest(map[string]*rooms.BiomeInfo{
		`default`: {
			BiomeId:      `default`,
			Name:         `Default`,
			Symbol:       `•`,
			LitArea:      true,
			Description:  `A default biome used in tests.`,
			MovementCost: 1.0,
		},
	})
}

// ---------------------------------------------------------------------------
// Defuse test helpers
// ---------------------------------------------------------------------------

const defuseTestTrapBuffId = 42
const defuseTestContainerName = "chest"
const defuseTestExitName = "north"

// newDefuseActor creates a player-backed stubActorWithId with the given
// Perception and skullduggery skill rank.
func newDefuseActor(perception int, skillRank int) *stubActorWithId {
	char := characters.New()
	char.Stats.Perception.ValueAdj = perception
	if skillRank > 0 {
		char.Skills[string(skills.Skullduggery)] = skillRank
	}
	room := newDefuseRoom(5 /* difficulty */)
	return &stubActorWithId{
		stubActor: stubActor{char: char, room: room},
		isPlayer:  true,
		userId:    5001,
	}
}

// newDefuseRoom creates a room with a trapped chest (container) and a trapped
// north exit with the given lock difficulty. Difficulty 0 is effectively
// difficulty 5 (must be >0 for HasLock to return true).
func newDefuseRoom(difficulty uint8) *rooms.Room {
	if difficulty == 0 {
		difficulty = 5
	}
	r := &rooms.Room{RoomId: 99990}

	// Add a trapped container.
	r.Containers = map[string]rooms.Container{
		defuseTestContainerName: {
			Lock: gamelock.Lock{
				Difficulty:  difficulty,
				TrapBuffIds: []int{defuseTestTrapBuffId},
			},
		},
	}

	// Add a trapped exit.
	r.Exits = map[string]exit.RoomExit{
		defuseTestExitName: {
			RoomId: 2,
			Lock: gamelock.Lock{
				Difficulty:  difficulty,
				TrapBuffIds: []int{defuseTestTrapBuffId},
			},
		},
	}

	return r
}

// newDisarmKit creates a minimal disarm kit item for tests.
// The item uses an inline Spec so no global registry is needed.
// bonus controls the "defuse_bonus" StatMod value.
func newDisarmKit(itemId int, bonus int) items.Item {
	sm := statmods.StatMods{}
	if bonus > 0 {
		sm["defuse_bonus"] = bonus
	}
	return items.Item{
		ItemId: itemId,
		Spec: &items.ItemSpec{
			ItemId:   itemId,
			Name:     "disarm kit",
			StatMods: sm,
		},
	}
}

// seedDisarmKit places a disarm kit into the actor's backpack.
func seedDisarmKit(actor *stubActorWithId, itemId int, bonus int) {
	kit := newDisarmKit(itemId, bonus)
	actor.char.Items = append(actor.char.Items, kit)
}

// buffIds returns the AddBuff calls recorded on a stubActorWithId.
// The base stubActor.AddBuff discards calls; we override here to track them.
type trackingActor struct {
	*stubActorWithId
	buffIds []int
}

func (a *trackingActor) AddBuff(buffId int, source string) {
	a.buffIds = append(a.buffIds, buffId)
}

func newTrackingActor(perception int, skillRank int) *trackingActor {
	return &trackingActor{
		stubActorWithId: newDefuseActor(perception, skillRank),
	}
}

// ---------------------------------------------------------------------------
// TestDefuse_NoTrap — empty TargetNoun with no traps should return "no target"
// ---------------------------------------------------------------------------

func TestDefuse_NoTrap(t *testing.T) {
	actor := newDefuseActor(100, 3)

	result := Defuse(actor, DefuseOptions{TargetNoun: ""})

	assert.False(t, result.Succeeded)
	assert.Equal(t, "no target", result.Reason,
		"empty TargetNoun should return 'no target'")
}

// ---------------------------------------------------------------------------
// TestDefuse_NotAdvancedEnough — skill rank below 3 should be rejected
// ---------------------------------------------------------------------------

func TestDefuse_NotAdvancedEnough(t *testing.T) {
	actor := newDefuseActor(100, 2) // rank 2 — below the rank-3 gate

	// Seed kit so the skill gate fires before the kit gate.
	seedDisarmKit(actor, 9901, 0)

	result := Defuse(actor, DefuseOptions{TargetNoun: defuseTestContainerName})

	assert.False(t, result.Succeeded)
	assert.Equal(t, "not advanced enough", result.Reason,
		"rank 2 should fail the rank-3 gate")
}

// ---------------------------------------------------------------------------
// TestDefuse_InCombat — actor with Aggro set should be rejected
// ---------------------------------------------------------------------------

func TestDefuse_InCombat(t *testing.T) {
	actor := newDefuseActor(100, 5)
	actor.char.Aggro = &characters.Aggro{
		Type:          characters.DefaultAttack,
		MobInstanceId: 1,
	}
	seedDisarmKit(actor, 9902, 0)

	result := Defuse(actor, DefuseOptions{TargetNoun: defuseTestContainerName})

	assert.False(t, result.Succeeded)
	assert.Equal(t, "in combat", result.Reason)
}

// ---------------------------------------------------------------------------
// TestDefuse_NoDisarmKit — missing disarm kit should block the attempt
// ---------------------------------------------------------------------------

func TestDefuse_NoDisarmKit(t *testing.T) {
	actor := newDefuseActor(100, 5)
	// No kit in backpack.

	result := Defuse(actor, DefuseOptions{TargetNoun: defuseTestContainerName})

	assert.False(t, result.Succeeded)
	assert.Equal(t, "no disarm kit", result.Reason,
		"missing kit should return 'no disarm kit'")
}

// ---------------------------------------------------------------------------
// TestDefuse_Success — high-Per actor vs low-difficulty trap (container)
// ---------------------------------------------------------------------------

func TestDefuse_Success(t *testing.T) {
	// difficulty=1, Per=500, rank=5 → defuseScore >> trapDifficulty (10)
	actor := newDefuseActor(500, 5)

	succeeded := 0
	const trials = 20
	for i := 0; i < trials; i++ {
		// Re-seed kit and reset room trap each trial.
		actor.char.Items = nil
		seedDisarmKit(actor, 9910+i, 0)
		// Reset trap on container.
		actor.room.Containers = map[string]rooms.Container{
			defuseTestContainerName: {
				Lock: gamelock.Lock{
					Difficulty:  1,
					TrapBuffIds: []int{defuseTestTrapBuffId},
				},
			},
		}

		result := Defuse(actor, DefuseOptions{TargetNoun: defuseTestContainerName})
		if result.Succeeded {
			succeeded++
			assert.Equal(t, defuseTestContainerName, result.TrapName,
				"TrapName should be the container name")
			assert.True(t, result.KitConsumed,
				"kit should be consumed on success")
			// Verify trap was cleared from the container.
			container := actor.room.Containers[defuseTestContainerName]
			assert.Nil(t, container.Lock.TrapBuffIds,
				"trap buff IDs should be nil after successful defuse")
			// Kit should be gone from backpack.
			_, stillHas := actor.char.FindInBackpack("disarm kit")
			assert.False(t, stillHas,
				"disarm kit should be consumed after defuse attempt")
		}
	}

	assert.Greater(t, succeeded, trials/2,
		"high-Per actor vs difficulty-1 trap should succeed most of the time")

	// Skill progression must fire on every attempt.
	require.Greater(t, actor.skillUsesByName[string(skills.Skullduggery)], 0,
		"Defuse should call OnSkillUse('skullduggery') on every kit-present attempt")
	assert.GreaterOrEqual(t, actor.skillUsesByName[string(skills.Skullduggery)],
		trials,
		"OnSkillUse should have been called once per trial")
}

// ---------------------------------------------------------------------------
// TestDefuse_FailureTriggers — low-skill vs high-difficulty trap
// ---------------------------------------------------------------------------

func TestDefuse_FailureTriggers(t *testing.T) {
	// difficulty=100 → trapDifficulty=1000; Per=10, rank=3 →
	// defuseScore ≈ 10 + 75 = 85. Actor loses the roll reliably.
	tracker := newTrackingActor(10, 3)

	failedTrials := 0
	const trials = 20
	for i := 0; i < trials; i++ {
		// Re-seed kit and reset room trap each trial.
		tracker.char.Items = nil
		tracker.char.Items = append(tracker.char.Items, newDisarmKit(9920+i, 0))
		tracker.room.Containers = map[string]rooms.Container{
			defuseTestContainerName: {
				Lock: gamelock.Lock{
					Difficulty:  100,
					TrapBuffIds: []int{defuseTestTrapBuffId},
				},
			},
		}
		tracker.buffIds = nil

		result := Defuse(tracker, DefuseOptions{TargetNoun: defuseTestContainerName})
		if !result.Succeeded {
			failedTrials++
			assert.NotEmpty(t, result.TriggeredTraps,
				"TriggeredTraps should list the trap buff IDs on failure")
			assert.Equal(t, []int{defuseTestTrapBuffId}, result.TriggeredTraps)
			assert.True(t, result.KitConsumed,
				"kit is consumed even on failure")
			// The trackingActor overrides AddBuff; verify it was called.
			assert.Contains(t, tracker.buffIds, defuseTestTrapBuffId,
				"trap buff should have been applied to the actor on failure")
		}
	}

	assert.Greater(t, failedTrials, trials/2,
		"low-skill actor vs difficulty-100 trap should fail most of the time")
}

// ---------------------------------------------------------------------------
// TestDefuse_KitBonus — kit with defuse_bonus > 0 is reported in result
// ---------------------------------------------------------------------------

func TestDefuse_KitBonus(t *testing.T) {
	// Per=500, rank=5, bonus=50. We only check KitBonusUsed is correct.
	// Use difficulty=1 so the actor wins reliably.
	actor := newDefuseActor(500, 5)

	const kitBonus = 50
	actor.char.Items = append(actor.char.Items, newDisarmKit(9930, kitBonus))

	result := Defuse(actor, DefuseOptions{TargetNoun: defuseTestContainerName})

	// With Per=500 vs difficulty 5, success is nearly certain. If it happens
	// to fail on the first trial (extremely unlikely), the kit is still
	// consumed and the bonus should still be reported.
	assert.True(t, result.KitConsumed, "kit should always be consumed")
	assert.Equal(t, kitBonus, result.KitBonusUsed,
		"KitBonusUsed should equal the kit's defuse_bonus StatMod")

	// Kit should be gone from backpack.
	_, stillHas := actor.char.FindInBackpack("disarm kit")
	assert.False(t, stillHas, "disarm kit should be consumed after attempt")
}

// ---------------------------------------------------------------------------
// TestDefuse_ExitTarget — defuse via exit noun instead of container
// ---------------------------------------------------------------------------

func TestDefuse_ExitTarget(t *testing.T) {
	// Per=500, rank=5, difficulty=1 → success reliably.
	actor := newDefuseActor(500, 5)
	seedDisarmKit(actor, 9940, 0)

	result := Defuse(actor, DefuseOptions{TargetNoun: defuseTestExitName})

	// Success is highly probable. Verify exit trap is cleared on success.
	if result.Succeeded {
		assert.Equal(t, defuseTestExitName, result.TrapName)
		assert.True(t, result.KitConsumed)
		exitInfo, ok := actor.room.Exits[defuseTestExitName]
		require.True(t, ok, "exit should still exist")
		assert.Nil(t, exitInfo.Lock.TrapBuffIds,
			"exit trap buff IDs should be nil after successful defuse")
	} else {
		// On failure, TriggeredTraps must be populated.
		assert.NotEmpty(t, result.TriggeredTraps)
	}
}

// ---------------------------------------------------------------------------
// TestDefuse_SkillProgressionFires — OnSkillUse is called on every attempt
// where a kit is present (regardless of roll outcome)
// ---------------------------------------------------------------------------

func TestDefuse_SkillProgressionFires(t *testing.T) {
	// Use moderate stats so some attempts succeed and some fail.
	actor := newDefuseActor(50, 3)

	const trials = 30
	for i := 0; i < trials; i++ {
		actor.char.Items = nil
		seedDisarmKit(actor, 9950+i, 0)
		actor.room.Containers = map[string]rooms.Container{
			defuseTestContainerName: {
				Lock: gamelock.Lock{
					Difficulty:  5,
					TrapBuffIds: []int{defuseTestTrapBuffId},
				},
			},
		}
		Defuse(actor, DefuseOptions{TargetNoun: defuseTestContainerName})
	}

	assert.GreaterOrEqual(t, actor.skillUsesByName[string(skills.Skullduggery)],
		trials,
		"OnSkillUse('skullduggery') should be called once per kit-present attempt")
}

// ---------------------------------------------------------------------------
// TestDefuse_NoLockOnTarget — targeting a container without a lock
// ---------------------------------------------------------------------------

func TestDefuse_NoLockOnTarget(t *testing.T) {
	actor := newDefuseActor(100, 5)
	seedDisarmKit(actor, 9960, 0)

	// Override room with an unlocked container (Difficulty=0 → HasLock=false).
	actor.room.Containers = map[string]rooms.Container{
		"barrel": {
			Lock: gamelock.Lock{Difficulty: 0},
		},
	}

	result := Defuse(actor, DefuseOptions{TargetNoun: "barrel"})

	assert.False(t, result.Succeeded)
	// Kit should NOT be consumed when the target has no lock.
	// (The kit gate fires AFTER target resolution; no lock → returns early
	// before the kit is consumed.)
	_, stillHas := actor.char.FindInBackpack("disarm kit")
	assert.True(t, stillHas,
		"disarm kit should not be consumed when there is no lock to defuse")
}

// ---------------------------------------------------------------------------
// TestDefuse_NoTrapsOnLock — targeting a locked container with no traps
// ---------------------------------------------------------------------------

func TestDefuse_NoTrapsOnLock(t *testing.T) {
	actor := newDefuseActor(100, 5)
	seedDisarmKit(actor, 9970, 0)

	// Container has a lock but TrapBuffIds is empty.
	actor.room.Containers = map[string]rooms.Container{
		defuseTestContainerName: {
			Lock: gamelock.Lock{
				Difficulty:  5,
				TrapBuffIds: []int{}, // no traps
			},
		},
	}

	result := Defuse(actor, DefuseOptions{TargetNoun: defuseTestContainerName})

	assert.False(t, result.Succeeded)
	// Kit should NOT be consumed when there are no traps on the lock.
	_, stillHas := actor.char.FindInBackpack("disarm kit")
	assert.True(t, stillHas,
		"disarm kit should not be consumed when the lock has no traps")
}

// Ensure buffs package is used (for Buffs.New in room/character init).
var _ = buffs.New
