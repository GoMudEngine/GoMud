package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

const testMobInstId = 9901

// newStealTestMob creates a minimal *mobs.Mob suitable for steal testing.
// The returned mob has gold and low Perception so the attacker wins easily.
func newStealTestMob(instanceId int, gold int, perception int) *mobs.Mob {
	m := &mobs.Mob{
		InstanceId: instanceId,
	}
	m.Character.Name = "Bandit"
	m.Character.Buffs = buffs.New()
	m.Character.Stats.Perception.ValueAdj = perception
	m.Character.Gold = gold
	return m
}

// newStealTestRoom creates a minimal room. It is passed to stubActor so
// AreMobsAttacking returns false (no mobs registered to the room).
func newStealTestRoom() *rooms.Room {
	return &rooms.Room{RoomId: 99998}
}

// newStealPlayerActor creates a player-backed stubActor with the given
// Dexterity and skullduggery skill rank.
func newStealPlayerActor(dex int, skillRank int) *stubActorWithId {
	char := characters.New()
	char.Stats.Dexterity.ValueAdj = dex
	if skillRank > 0 {
		char.Skills[string(skills.Skullduggery)] = skillRank
	}
	room := newStealTestRoom()
	return &stubActorWithId{
		stubActor: stubActor{char: char, room: room},
		isPlayer:  true,
		userId:    1001,
	}
}

// newStealMobActor creates a mob-backed stubActor with the given
// Dexterity and skullduggery skill rank.
func newStealMobActor(dex int, skillRank int) *stubActorWithId {
	char := characters.New()
	char.Stats.Dexterity.ValueAdj = dex
	if skillRank > 0 {
		char.Skills[string(skills.Skullduggery)] = skillRank
	}
	room := newStealTestRoom()
	return &stubActorWithId{
		stubActor:     stubActor{char: char, room: room},
		isPlayer:      false,
		mobInstanceId: 9801,
	}
}

// stubActorWithId extends the existing stubActor to let tests control
// IsPlayer / GetUserId / GetMobInstanceId independently. It also tracks
// OnSkillUse calls so tests can assert that skill progression fires.
type stubActorWithId struct {
	stubActor
	isPlayer        bool
	userId          int
	mobInstanceId   int
	skillUsesByName map[string]int // counts per skill name
	skillUseTotal   int            // total calls across all skills
}

func (a *stubActorWithId) IsPlayer() bool       { return a.isPlayer }
func (a *stubActorWithId) GetUserId() int        { return a.userId }
func (a *stubActorWithId) GetMobInstanceId() int { return a.mobInstanceId }

// OnSkillUse records each call so tests can assert skill progression fires.
func (a *stubActorWithId) OnSkillUse(skillName string) bool {
	if a.skillUsesByName == nil {
		a.skillUsesByName = make(map[string]int)
	}
	a.skillUsesByName[skillName]++
	a.skillUseTotal++
	return false
}

// ---------------------------------------------------------------------------
// TestSteal_NoTarget
// ---------------------------------------------------------------------------

// TestSteal_NoTarget verifies that empty StealOptions returns a failure
// mentioning "no target" without triggering any cooldown.
func TestSteal_NoTarget(t *testing.T) {
	actor := newStealPlayerActor(100, 5)

	result := Steal(actor, StealOptions{})

	assert.False(t, result.Succeeded, "empty opts should not succeed")
	assert.False(t, result.OnCooldown, "empty opts should not set cooldown")
	assert.Equal(t, "no target", result.Reason)
}

// ---------------------------------------------------------------------------
// TestSteal_MobTarget_GoldSuccess
// ---------------------------------------------------------------------------

// TestSteal_MobTarget_GoldSuccess verifies that a strong attacker (Dex 80,
// skullduggery rank 8) reliably steals gold from a mob with near-zero
// Perception. The test registers a live mob instance and confirms the gold
// transfer happened.
func TestSteal_MobTarget_GoldSuccess(t *testing.T) {
	const mobGold = 50

	target := newStealTestMob(testMobInstId, mobGold, 1) // Perception 1 = nearly blind
	mobs.SetInstanceForTest(testMobInstId, target)
	defer mobs.SetInstanceForTest(testMobInstId, nil)

	// Dex 200, rank 8 — attacker score will be far above Perception 1.
	actor := newStealPlayerActor(200, 8)

	// Run many trials to confirm success is not just luck.
	succeeded := 0
	const trials = 20
	for i := 0; i < trials; i++ {
		// Reset mob gold and actor gold each trial.
		target.Character.Gold = mobGold
		goldBefore := actor.char.Gold

		// Re-register after possible mutation.
		mobs.SetInstanceForTest(testMobInstId, target)

		// Reset cooldown each trial so only the first result matters for
		// the gold-transfer assertion.
		delete(actor.char.Cooldowns, skills.Skullduggery.String("steal"))

		result := Steal(actor, StealOptions{TargetMobInstanceId: testMobInstId})
		if result.Succeeded {
			succeeded++
			assert.Greater(t, result.StoleGold, 0,
				"should have stolen positive gold")
			assert.Equal(t, mobGold-result.StoleGold, target.Character.Gold,
				"mob gold should decrease by stolen amount")
			assert.Equal(t, goldBefore+result.StoleGold, actor.char.Gold,
				"actor gold should increase by stolen amount")
		}
	}

	// With Dex 200 vs Perception 1, we expect near-100% success.
	// Allow a small tolerance for dice variance.
	assert.Greater(t, succeeded, trials/2,
		"high-stat attacker should succeed majority of attempts")

	// Skill progression must fire on every steal attempt (win or lose).
	// Over 20 trials the counter must be at least 20.
	require.Greater(t, actor.skillUsesByName[string(skills.Skullduggery)], 0,
		"Steal should call OnSkillUse('skullduggery') on every attempt")
	assert.GreaterOrEqual(t, actor.skillUsesByName[string(skills.Skullduggery)], trials,
		"OnSkillUse should have been called once per trial")
}

// ---------------------------------------------------------------------------
// TestSteal_OnCooldown
// ---------------------------------------------------------------------------

// TestSteal_OnCooldown verifies that after one steal attempt the action is
// gated by a cooldown on the second call, and that the mob's gold is
// unchanged on the blocked attempt.
func TestSteal_OnCooldown(t *testing.T) {
	const mobGold = 50

	target := newStealTestMob(testMobInstId+1, mobGold, 1)
	mobs.SetInstanceForTest(testMobInstId+1, target)
	defer mobs.SetInstanceForTest(testMobInstId+1, nil)

	// Very strong attacker to make the first steal succeed reliably.
	actor := newStealPlayerActor(200, 8)

	first := Steal(actor, StealOptions{TargetMobInstanceId: testMobInstId + 1})
	// First call should either succeed or fail (depends on dice), but the
	// cooldown is set either way.
	_ = first

	// Second call must be blocked — cooldown was set on first call.
	goldAfterFirst := target.Character.Gold
	second := Steal(actor, StealOptions{TargetMobInstanceId: testMobInstId + 1})

	assert.True(t, second.OnCooldown, "second steal must be on cooldown")
	assert.Equal(t, goldAfterFirst, target.Character.Gold,
		"mob gold must not change on blocked attempt")
}

// ---------------------------------------------------------------------------
// TestSteal_MobOnPlayer
// ---------------------------------------------------------------------------

// TestSteal_MobOnPlayer verifies the mob-actor steal-from-player path. A mob
// with very high Dex+skill steals gold from a player with low gold. The test
// does not assert on Detected because the detection roll is probabilistic.
func TestSteal_MobOnPlayer(t *testing.T) {
	const playerGold = 100

	// Register target player.
	targetUser := users.NewTestUser(2001, "victim", "Victim", 0)
	targetUser.Character.Gold = playerGold
	targetUser.Character.Stats.Perception.ValueAdj = 5 // nearly blind
	cleanup := users.SeedUsersForTest(map[int]*users.UserRecord{
		2001: targetUser,
	})
	defer cleanup()

	// Mob actor with very high Dex and skullduggery to win the roll.
	actor := newStealMobActor(200, 8)
	actorGoldBefore := actor.char.Gold

	result := Steal(actor, StealOptions{TargetUserId: 2001})

	// With Dex 200 vs Perception 5, success is highly probable.
	// We check the transfer happened correctly when it succeeds.
	if result.Succeeded {
		assert.Greater(t, result.StoleGold, 0, "should steal positive gold")
		assert.Equal(t, playerGold-result.StoleGold, targetUser.Character.Gold,
			"target player gold should decrease by stolen amount")
		assert.Equal(t, actorGoldBefore+result.StoleGold, actor.char.Gold,
			"actor gold should increase by stolen amount")
	} else {
		// If detected, actor should have been flagged.
		assert.True(t, result.Detected,
			"failure must be due to detection, not other error")
	}
}

// ---------------------------------------------------------------------------
// TestSteal_MobOnPlayer_DetectionWin
// ---------------------------------------------------------------------------

// TestSteal_MobOnPlayer_DetectionWin verifies that the detection-win branch in
// stealFromPlayer (result.Detected = true + targetUser.SendText) is reachable
// under realistic stats.
//
// Design rationale:
//
//	Theft uses: attackerScore (Dex + SkillMult(rank)*25) * StealSkillMultiplier
//	            vs defenderScore = raw Perception
//	Detection uses: CalcSearchScore (Perception + SkillMult(search)*25)
//	                vs CalcSneakScore (Dex + SkillMult(skullduggery)*25)
//
// By giving the defender a high Search skill rank we boost CalcSearchScore
// far above their raw Perception, while their raw Perception stays low enough
// that the theft roll succeeds reliably.
//
//	Actor:    Dex=110, skullduggery rank=2
//	          attackerScore  ≈ 145   (beats Per=40 → theft nearly certain)
//	          CalcSneakScore ≈ 145
//	Defender: Perception=40, Search rank=50
//	          CalcSearchScore ≈ 40 + 3.0*25 = 115
//
// Detection is 115 vs 145, a ~24% per-trial chance. Over 50 trials the
// probability of seeing at least one detection event is >99.999%.
func TestSteal_MobOnPlayer_DetectionWin(t *testing.T) {
	const playerGold = 200
	const trials = 50

	// Register target player with high Search skill so CalcSearchScore
	// outpaces the actor's CalcSneakScore significantly.
	targetUser := users.NewTestUser(3001, "watchful", "Watchful", 0)
	targetUser.Character.Gold = playerGold
	targetUser.Character.Stats.Perception.ValueAdj = 40 // low raw Per → theft usually succeeds
	if targetUser.Character.Skills == nil {
		targetUser.Character.Skills = make(map[string]int)
	}
	targetUser.Character.Skills[string(skills.Search)] = 50 // CalcSearchScore ≈ 115
	cleanup := users.SeedUsersForTest(map[int]*users.UserRecord{
		3001: targetUser,
	})
	defer cleanup()

	// Mob actor: Dex=110, rank=2 (just meets the gate). attackerScore ≈ 145.
	actor := newStealMobActor(110, 2)

	detectionCount := 0
	successCount := 0

	for i := 0; i < trials; i++ {
		// Reset gold and cooldown each trial.
		targetUser.Character.Gold = playerGold
		delete(actor.char.Cooldowns, skills.Skullduggery.String("steal"))

		result := Steal(actor, StealOptions{TargetUserId: 3001})
		if result.Succeeded {
			successCount++
		}
		if result.Detected && result.Succeeded {
			detectionCount++
		}
	}

	// Theft should succeed frequently (high attackerScore vs low raw Perception).
	require.Greater(t, successCount, trials/2,
		"theft must succeed in majority of trials to exercise the detection path")

	// Detection must fire in at least one trial. With ~24% per-successful-theft
	// probability over 50 trials the chance of zero detections is <0.0001%.
	require.Greater(t, detectionCount, 0,
		"detection-win branch (result.Detected=true after successful theft) "+
			"must be reachable; check CalcSearchScore vs CalcSneakScore math")
}

// ---------------------------------------------------------------------------
// TestSteal_InCombat
// ---------------------------------------------------------------------------

// TestSteal_InCombat verifies that a character with active Aggro cannot steal.
func TestSteal_InCombat(t *testing.T) {
	actor := newStealPlayerActor(100, 5)
	actor.char.Aggro = &characters.Aggro{
		Type:          characters.DefaultAttack,
		MobInstanceId: 1,
	}

	result := Steal(actor, StealOptions{TargetMobInstanceId: testMobInstId})

	assert.False(t, result.Succeeded)
	assert.Equal(t, "in combat", result.Reason)
}
