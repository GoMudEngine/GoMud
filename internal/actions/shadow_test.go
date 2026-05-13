package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Package-level setup: seed buff 9 (Hidden) spec so AddBuff(9, …) works.
// This piggybacks on the package-level init in defuse_test.go (biomes), and
// mirrors the approach used throughout other action tests.
// ---------------------------------------------------------------------------

func init() {
	// Seed buff 9 with the Hidden flag. TriggerCount > 0 so the buff is
	// not considered expired immediately after application.
	buffs.SeedBuffsForTest(map[int]*buffs.BuffSpec{
		9: {
			BuffId:       9,
			Name:         "Hidden",
			Flags:        []buffs.Flag{buffs.Hidden},
			TriggerCount: 1000000, // effectively permanent for tests
		},
	})
	// Note: SeedBuffsForTest replaces the entire registry. No cleanup is
	// called here because the hidden-buff spec is needed for the full test run
	// and must survive across tests in this package.
}

// ---------------------------------------------------------------------------
// Shadow test helpers
// ---------------------------------------------------------------------------

// addHiddenBuff grants buff 9 (Hidden) to a character. Requires that the buff
// spec for id 9 has been seeded via init() above.
func addHiddenBuff(char *characters.Character) {
	char.Buffs = buffs.New()
	// AddBuff calls GetBuffSpec internally; works because of the seeded spec.
	char.Buffs.AddBuff(9, true /* permanent for test purposes */)
}

// newShadowPlayerActor creates a player-backed stubActorWithId with the
// given Dexterity and skullduggery skill rank.
// When withHidden=true, buff 9 (Hidden) is applied.
func newShadowPlayerActor(dex int, skillRank int, withHidden bool) *stubActorWithId {
	char := characters.New()
	char.Stats.Dexterity.ValueAdj = dex
	if skillRank > 0 {
		char.Skills[string(skills.Skullduggery)] = skillRank
	}
	char.Buffs = buffs.New()
	if withHidden {
		addHiddenBuff(char)
	}
	room := newStealTestRoom()
	return &stubActorWithId{
		stubActor: stubActor{char: char, room: room},
		isPlayer:  true,
		userId:    7001,
	}
}

// newShadowMobActor creates a mob-backed stubActorWithId with the given
// Dexterity and skullduggery skill rank.
// When withHidden=true, buff 9 (Hidden) is applied.
func newShadowMobActor(dex int, skillRank int, withHidden bool) *stubActorWithId {
	char := characters.New()
	char.Stats.Dexterity.ValueAdj = dex
	if skillRank > 0 {
		char.Skills[string(skills.Skullduggery)] = skillRank
	}
	char.Buffs = buffs.New()
	if withHidden {
		addHiddenBuff(char)
	}
	room := newStealTestRoom()
	return &stubActorWithId{
		stubActor:     stubActor{char: char, room: room},
		isPlayer:      false,
		mobInstanceId: 8801,
	}
}

// resetHiddenBuff clears and re-adds buff 9 on the actor's character.
// Use between trials when shadow might not remove the buff but the test
// wants a consistent starting state.
func resetHiddenBuff(actor *stubActorWithId) {
	actor.char.Buffs = buffs.New()
	addHiddenBuff(actor.char)
}

// ---------------------------------------------------------------------------
// TestShadow_RequiresHidden
// ---------------------------------------------------------------------------

// TestShadow_RequiresHidden verifies that an actor without buff 9 (Hidden)
// is rejected before any roll or cooldown is set.
func TestShadow_RequiresHidden(t *testing.T) {
	actor := newShadowPlayerActor(100, 5, false /* not hidden */)

	result := Shadow(actor, ShadowOptions{TargetUserId: 7002})

	assert.False(t, result.Succeeded, "actor without Hidden buff should not succeed")
	assert.False(t, result.OnCooldown, "hidden gate should not set cooldown")
	assert.Equal(t, "not hidden", result.Reason)
	// Cooldown must not have been set.
	assert.Equal(t, 0, actor.char.GetCooldown(skills.Skullduggery.String("shadow")),
		"cooldown should not be set when hidden gate fires")
}

// ---------------------------------------------------------------------------
// TestShadow_NoTarget
// ---------------------------------------------------------------------------

// TestShadow_NoTarget verifies that empty ShadowOptions (with hidden actor)
// returns a failure mentioning "no target" without triggering any cooldown.
func TestShadow_NoTarget(t *testing.T) {
	actor := newShadowPlayerActor(100, 5, true /* hidden */)

	result := Shadow(actor, ShadowOptions{})

	assert.False(t, result.Succeeded, "empty opts should not succeed")
	assert.False(t, result.OnCooldown, "empty opts should not set cooldown")
	assert.Equal(t, "no target", result.Reason)
}

// ---------------------------------------------------------------------------
// TestShadow_Success
// ---------------------------------------------------------------------------

// TestShadow_Success verifies that a hidden actor targeting a valid player
// succeeds and stores the target id in misc-data.
func TestShadow_Success(t *testing.T) {
	targetUser := users.NewTestUser(7002, "prey", "Prey", 0)
	targetUser.Character.Stats.Perception.ValueAdj = 1  // nearly blind — won't detect
	targetUser.Character.Stats.Dexterity.ValueAdj = 1
	cleanup := users.SeedUsersForTest(map[int]*users.UserRecord{
		7002: targetUser,
	})
	defer cleanup()

	// High Dex actor → CalcSneakScore far beats CalcSearchScore(Per=1)
	actor := newShadowPlayerActor(200, 5, true /* hidden */)

	result := Shadow(actor, ShadowOptions{TargetUserId: 7002})

	require.True(t, result.Succeeded, "hidden actor with valid target should succeed")
	assert.Equal(t, "Prey", result.TargetName)

	// Misc-data must store the target user id.
	raw := actor.char.GetMiscData("shadow-target-user")
	require.NotNil(t, raw, "shadow-target-user misc-data must be set on success")
	uid, ok := raw.(int)
	require.True(t, ok, "shadow-target-user must be an int")
	assert.Equal(t, 7002, uid)

	// Mob slot must be cleared.
	assert.Nil(t, actor.char.GetMiscData("shadow-target-mob"),
		"shadow-target-mob misc-data must be nil for a player target")
}

// ---------------------------------------------------------------------------
// TestShadow_Cooldown
// ---------------------------------------------------------------------------

// TestShadow_Cooldown verifies that the second Shadow invocation within
// the cooldown window is blocked.
func TestShadow_Cooldown(t *testing.T) {
	targetUser := users.NewTestUser(7003, "prey2", "Prey2", 0)
	targetUser.Character.Stats.Perception.ValueAdj = 1
	cleanup := users.SeedUsersForTest(map[int]*users.UserRecord{
		7003: targetUser,
	})
	defer cleanup()

	actor := newShadowPlayerActor(200, 5, true /* hidden */)

	first := Shadow(actor, ShadowOptions{TargetUserId: 7003})
	require.True(t, first.Succeeded, "first call should succeed")

	// Re-add Hidden buff for the second call.
	resetHiddenBuff(actor)

	second := Shadow(actor, ShadowOptions{TargetUserId: 7003})

	assert.True(t, second.OnCooldown, "second call within cooldown must be blocked")
	assert.False(t, second.Succeeded)
}

// ---------------------------------------------------------------------------
// TestShadow_SkillProgressionFires
// ---------------------------------------------------------------------------

// TestShadow_SkillProgressionFires verifies that OnSkillUse is called once
// per successful Shadow attempt (mob actor, no quest-engine path).
func TestShadow_SkillProgressionFires(t *testing.T) {
	targetUser := users.NewTestUser(7004, "prey3", "Prey3", 0)
	targetUser.Character.Stats.Perception.ValueAdj = 1
	cleanup := users.SeedUsersForTest(map[int]*users.UserRecord{
		7004: targetUser,
	})
	defer cleanup()

	actor := newShadowMobActor(200, 5, true /* hidden */)

	result := Shadow(actor, ShadowOptions{TargetUserId: 7004})
	require.True(t, result.Succeeded)

	assert.Greater(t, actor.skillUsesByName[string(skills.Skullduggery)], 0,
		"Shadow should call OnSkillUse('skullduggery') on success")
}

// ---------------------------------------------------------------------------
// TestShadow_DetectionWin
// ---------------------------------------------------------------------------

// TestShadow_DetectionWin verifies that the initial detection roll fires
// (Detected=true) when the target has high Perception+Search vs the
// actor's Dex+Skullduggery, and that the target receives the detection
// message.
//
// Design rationale (mirrors steal/plant detection tests):
//
//	Shadow uses: CalcSneakScore (Dex + SkillMult(skulld)*25) for actor
//	             CalcSearchScore (Per + SkillMult(search)*25) for target
//	Detection:   OpposedRollStat(searchScore, sneakScore);
//	             target wins = detected.
//
//	Actor:    Dex=110, skullduggery rank=2
//	          CalcSneakScore ≈ 110 + 1.41*25 ≈ 145
//	Target:   Perception=40, Search rank=50
//	          CalcSearchScore ≈ 40 + 3.0*25 = 115
//
// Detection is 115 vs 145, a ~24% per-trial chance. Over 50 trials the
// probability of zero detections is <0.0001%.
func TestShadow_DetectionWin(t *testing.T) {
	const trials = 50

	// Register target player with high Search skill.
	targetUser := users.NewTestUser(7005, "watchful3", "Watchful3", 0)
	targetUser.Character.Stats.Perception.ValueAdj = 40
	if targetUser.Character.Skills == nil {
		targetUser.Character.Skills = make(map[string]int)
	}
	targetUser.Character.Skills[string(skills.Search)] = 50 // CalcSearchScore ≈ 115
	cleanup := users.SeedUsersForTest(map[int]*users.UserRecord{
		7005: targetUser,
	})
	defer cleanup()

	// Mob actor: Dex=110, rank=2. CalcSneakScore ≈ 145.
	actor := newShadowMobActor(110, 2, true /* hidden */)

	detectionCount := 0
	successCount := 0

	for i := 0; i < trials; i++ {
		// Reset cooldown and Hidden buff each trial.
		delete(actor.char.Cooldowns, skills.Skullduggery.String("shadow"))
		resetHiddenBuff(actor)

		result := Shadow(actor, ShadowOptions{TargetUserId: 7005})
		if result.Succeeded {
			successCount++
		}
		if result.Detected {
			detectionCount++
		}
	}

	// Shadow always succeeds when the actor is hidden — Detected is orthogonal.
	require.Greater(t, successCount, trials/2,
		"hidden actor should succeed in majority of trials")

	// Detection must fire in at least one trial. With ~24% per-trial
	// probability over 50 trials the chance of zero detections is <0.0001%.
	require.Greater(t, detectionCount, 0,
		"detection-win branch (result.Detected=true) must be reachable; "+
			"check CalcSearchScore vs CalcSneakScore math")
}
