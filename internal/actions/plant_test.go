package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Plant test helpers
// ---------------------------------------------------------------------------

const testPlantMobInstId = 9902

// newPlantTestMob creates a minimal *mobs.Mob suitable for plant testing.
// Low Perception so the attacker wins easily.
func newPlantTestMob(instanceId int, perception int) *mobs.Mob {
	m := &mobs.Mob{
		InstanceId: instanceId,
	}
	m.Character.Name = "Guard"
	m.Character.Buffs = buffs.New()
	m.Character.Stats.Perception.ValueAdj = perception
	return m
}

// newPlantPlayerActor creates a player-backed stubActor with the given
// Dexterity and skullduggery skill rank.
func newPlantPlayerActor(dex int, skillRank int) *stubActorWithId {
	char := characters.New()
	char.Stats.Dexterity.ValueAdj = dex
	if skillRank > 0 {
		char.Skills[string(skills.Skullduggery)] = skillRank
	}
	room := newStealTestRoom()
	return &stubActorWithId{
		stubActor: stubActor{char: char, room: room},
		isPlayer:  true,
		userId:    1002,
	}
}

// newPlantMobActor creates a mob-backed stubActor with the given
// Dexterity and skullduggery skill rank.
func newPlantMobActor(dex int, skillRank int) *stubActorWithId {
	char := characters.New()
	char.Stats.Dexterity.ValueAdj = dex
	if skillRank > 0 {
		char.Skills[string(skills.Skullduggery)] = skillRank
	}
	room := newStealTestRoom()
	return &stubActorWithId{
		stubActor:     stubActor{char: char, room: room},
		isPlayer:      false,
		mobInstanceId: 9802,
	}
}

// plantTestItem returns a minimal items.Item for use in tests.
// ItemId 1 is used; name lookup via !1 bypasses the global registry.
// The item is stored directly into the actor's backpack via StoreItem.
const plantTestItemId = 1

func plantTestItem() items.Item {
	return items.Item{ItemId: plantTestItemId}
}

// seedPlantItem puts a test item into the actor's backpack.
func seedPlantItem(actor *stubActorWithId) {
	actor.char.StoreItem(plantTestItem())
}

// ---------------------------------------------------------------------------
// TestPlant_RequiresItem
// ---------------------------------------------------------------------------

// TestPlant_RequiresItem verifies that an empty ItemNoun returns "no item"
// without touching any cooldown.
func TestPlant_RequiresItem(t *testing.T) {
	actor := newPlantPlayerActor(100, 5)

	result := Plant(actor, PlantOptions{TargetMobInstanceId: testPlantMobInstId})

	assert.False(t, result.Succeeded)
	assert.False(t, result.OnCooldown, "empty item should not set cooldown")
	assert.Equal(t, "no item", result.Reason)
}

// ---------------------------------------------------------------------------
// TestPlant_NoTarget
// ---------------------------------------------------------------------------

// TestPlant_NoTarget verifies that empty PlantOptions (no target) returns
// "no target" without setting cooldown or rolling.
func TestPlant_NoTarget(t *testing.T) {
	actor := newPlantPlayerActor(100, 5)
	seedPlantItem(actor)

	// ItemNoun set but no target id or container — triggers before item lookup.
	result := Plant(actor, PlantOptions{ItemNoun: "!1"})

	assert.False(t, result.Succeeded)
	assert.False(t, result.OnCooldown)
	assert.Equal(t, "no target", result.Reason)
}

// ---------------------------------------------------------------------------
// TestPlant_InCombat
// ---------------------------------------------------------------------------

// TestPlant_InCombat verifies that a character with active Aggro cannot plant.
func TestPlant_InCombat(t *testing.T) {
	actor := newPlantPlayerActor(100, 5)
	actor.char.Aggro = &characters.Aggro{
		Type:          characters.DefaultAttack,
		MobInstanceId: 1,
	}
	seedPlantItem(actor)

	result := Plant(actor, PlantOptions{
		TargetMobInstanceId: testPlantMobInstId,
		ItemNoun:            "!1",
	})

	assert.False(t, result.Succeeded)
	assert.Equal(t, "in combat", result.Reason)
}

// ---------------------------------------------------------------------------
// TestPlant_MobTarget_Success
// ---------------------------------------------------------------------------

// TestPlant_MobTarget_Success verifies that a strong attacker (Dex 200,
// skullduggery rank 8) reliably plants an item on a mob with near-zero
// Perception. The test confirms the item moved from actor backpack to mob.
func TestPlant_MobTarget_Success(t *testing.T) {
	target := newPlantTestMob(testPlantMobInstId, 1) // Perception 1 = nearly blind
	mobs.SetInstanceForTest(testPlantMobInstId, target)
	defer mobs.SetInstanceForTest(testPlantMobInstId, nil)

	// Dex 200, rank 8 — attacker score will be far above Perception 1.
	actor := newPlantPlayerActor(200, 8)

	succeeded := 0
	const trials = 20
	for i := 0; i < trials; i++ {
		// Re-seed the item and reset cooldown each trial.
		actor.char.Items = nil
		seedPlantItem(actor)
		target.Character.Items = nil
		delete(actor.char.Cooldowns, skills.Skullduggery.String("steal"))

		mobs.SetInstanceForTest(testPlantMobInstId, target)

		result := Plant(actor, PlantOptions{
			TargetMobInstanceId: testPlantMobInstId,
			ItemNoun:            "!1",
		})
		if result.Succeeded {
			succeeded++
			assert.Equal(t, plantTestItemId, result.PlantedItemId,
				"PlantedItemId should match the planted item")
			// Item should now be in the mob's inventory.
			_, mobHasItem := target.Character.FindInBackpack("!1")
			assert.True(t, mobHasItem,
				"planted item should be in mob's inventory after success")
			// Item should no longer be in actor's backpack.
			_, actorStillHas := actor.char.FindInBackpack("!1")
			assert.False(t, actorStillHas,
				"actor should no longer have the item after successful plant")
		}
	}

	// With Dex 200 vs Perception 1, expect near-100% success.
	assert.Greater(t, succeeded, trials/2,
		"high-stat attacker should succeed majority of plant attempts")

	// Skill progression must fire on every attempt.
	require.Greater(t, actor.skillUsesByName[string(skills.Skullduggery)], 0,
		"Plant should call OnSkillUse('skullduggery') on every attempt")
	assert.GreaterOrEqual(t, actor.skillUsesByName[string(skills.Skullduggery)],
		trials,
		"OnSkillUse should have been called once per trial")
}

// ---------------------------------------------------------------------------
// TestPlant_SharesStealCooldown
// ---------------------------------------------------------------------------

// TestPlant_SharesStealCooldown verifies that after a successful Steal call
// the skullduggery cooldown also blocks a subsequent Plant call on the same
// actor. This confirms the shared cooldown key is honoured.
func TestPlant_SharesStealCooldown(t *testing.T) {
	const mobGold = 50

	// Register a mob for the Steal call.
	stealMob := newStealTestMob(testMobInstId+10, mobGold, 1)
	mobs.SetInstanceForTest(testMobInstId+10, stealMob)
	defer mobs.SetInstanceForTest(testMobInstId+10, nil)

	// Register a separate mob for the Plant call.
	plantMob := newPlantTestMob(testPlantMobInstId+10, 1)
	mobs.SetInstanceForTest(testPlantMobInstId+10, plantMob)
	defer mobs.SetInstanceForTest(testPlantMobInstId+10, nil)

	actor := newPlantPlayerActor(200, 8)

	// First: Steal (sets the shared cooldown).
	_ = Steal(actor, StealOptions{TargetMobInstanceId: testMobInstId + 10})

	// Second: Plant on a different mob — must be blocked by the cooldown.
	seedPlantItem(actor)
	result := Plant(actor, PlantOptions{
		TargetMobInstanceId: testPlantMobInstId + 10,
		ItemNoun:            "!1",
	})

	assert.True(t, result.OnCooldown,
		"Plant after Steal must be blocked by shared skullduggery cooldown")
}

// ---------------------------------------------------------------------------
// TestPlant_MobOnPlayer_DetectionWin
// ---------------------------------------------------------------------------

// TestPlant_MobOnPlayer_DetectionWin verifies that the detection-win branch
// in plantOnPlayer (result.Detected=true after successful plant) is reachable
// under realistic stats.
//
// Design rationale (mirrors steal detection test):
//
//	Plant uses: attackerScore (Dex + SkillMult(rank)*25) * StealSkillMultiplier
//	            vs defenderScore = raw Perception
//	Detection uses: CalcSearchScore (Perception + SkillMult(search)*25)
//	                vs CalcSneakScore (Dex + SkillMult(skullduggery)*25)
//
// By giving the defender a high Search skill rank we boost CalcSearchScore
// far above their raw Perception, while their raw Perception stays low enough
// that the plant roll succeeds reliably.
//
//	Actor:    Dex=110, skullduggery rank=2
//	          attackerScore  ≈ 145   (beats Per=40 → plant nearly certain)
//	          CalcSneakScore ≈ 145
//	Defender: Perception=40, Search rank=50
//	          CalcSearchScore ≈ 40 + 3.0*25 = 115
//
// Detection is 115 vs 145, a ~24% per-trial chance. Over 50 trials the
// probability of zero detections is <0.0001%.
func TestPlant_MobOnPlayer_DetectionWin(t *testing.T) {
	const trials = 50

	// Register target player with high Search skill.
	targetUser := users.NewTestUser(4001, "watchful2", "Watchful2", 0)
	targetUser.Character.Stats.Perception.ValueAdj = 40
	if targetUser.Character.Skills == nil {
		targetUser.Character.Skills = make(map[string]int)
	}
	targetUser.Character.Skills[string(skills.Search)] = 50
	cleanup := users.SeedUsersForTest(map[int]*users.UserRecord{
		4001: targetUser,
	})
	defer cleanup()

	// Mob actor: Dex=110, rank=2.
	actor := newPlantMobActor(110, 2)

	detectionCount := 0
	successCount := 0

	for i := 0; i < trials; i++ {
		// Re-seed item and reset cooldown each trial.
		actor.char.Items = nil
		seedPlantItem(actor)
		delete(actor.char.Cooldowns, skills.Skullduggery.String("steal"))

		result := Plant(actor, PlantOptions{
			TargetUserId: 4001,
			ItemNoun:     "!1",
		})
		if result.Succeeded {
			successCount++
			// Return the item to actor for the next trial.
			if itm, found := targetUser.Character.FindInBackpack("!1"); found {
				targetUser.Character.RemoveItem(itm)
			}
		}
		if result.Detected && result.Succeeded {
			detectionCount++
		}
	}

	// Plant should succeed frequently.
	require.Greater(t, successCount, trials/2,
		"plant must succeed in majority of trials to exercise the detection path")

	// Detection must fire at least once.
	require.Greater(t, detectionCount, 0,
		"detection-win branch (result.Detected=true after successful plant) "+
			"must be reachable; check CalcSearchScore vs CalcSneakScore math")
}
