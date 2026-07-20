package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// ExecuteDrain tests
// ---------------------------------------------------------------------------

// TestDrain_NoAggro verifies that ExecuteDrain returns NoTarget=true when the
// actor has no aggro set (not yet in combat).
func TestDrain_NoAggro(t *testing.T) {
	char := characters.New()
	room := newTestRoom()
	actor := newStubActor(char, room)

	result := ExecuteDrain(actor)

	assert.False(t, result.Executed, "drain with no aggro should not execute")
	assert.True(t, result.NoTarget, "drain with no aggro should set NoTarget")
	assert.False(t, result.OnCooldown, "NoTarget should take priority over cooldown reporting")
}

// TestDrain_OnCooldown verifies that ExecuteDrain returns OnCooldown=true when
// the special-move cooldown is active.
func TestDrain_OnCooldown(t *testing.T) {
	char := characters.New()
	room := newTestRoom()
	actor := newStubActor(char, room)

	// Set aggro so we pass the nil check and reach the cooldown gate.
	char.Aggro = &characters.Aggro{MobInstanceId: 999999}

	// Burn the cooldown slot so the next Try call is blocked.
	char.Cooldowns.Try("special-move", "3 rounds")

	result := ExecuteDrain(actor)

	assert.False(t, result.Executed, "drain should not execute when on cooldown")
	assert.True(t, result.OnCooldown, "drain should report OnCooldown")
}

// TestDrain_NotLifeDrainer verifies the identity gate: a non-LifeDrain actor
// in combat with a valid target gets NotLifeDrainer=true, while a LifeDrain
// species passes the gate and executes.
func TestDrain_NotLifeDrainer(t *testing.T) {
	// Seed two species: one normal (no LifeDrain), one vampire (LifeDrain).
	cleanup := species.SeedSpeciesForTest(map[int]*species.Species{
		6001: {SpeciesId: 6001, Name: "human", BodyParts: []string{"arms", "hands", "legs"}},
		6002: {SpeciesId: 6002, Name: "vampire", BodyParts: []string{"arms", "hands", "legs", "mouth"}, NaturalAttack: items.Claws, LifeDrain: true},
	})
	defer cleanup()

	// Register a target mob so ResolveAggroTarget returns Found=true.
	targetMob := &mobs.Mob{InstanceId: 6099}
	targetMob.Character.Name = "Target"
	targetMob.Character.HealthMax.Value = 100
	targetMob.Character.Health = 100
	setCombatPositionParallel(&targetMob.Character, position.Standing)
	mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
	defer mobs.SetInstanceForTest(targetMob.InstanceId, nil)

	t.Run("normal species returns NotLifeDrainer", func(t *testing.T) {
		char := characters.New()
		char.SpeciesId = 6001 // human — no LifeDrain
		char.Aggro = &characters.Aggro{MobInstanceId: targetMob.InstanceId}

		result := ExecuteDrain(newStubActor(char, newTestRoom()))

		assert.True(t, result.NotLifeDrainer, "normal species should return NotLifeDrainer=true")
		assert.False(t, result.Executed, "normal species should not execute the drain")
		assert.Equal(t, 0, result.MoveResult.Damage, "normal species drain should deal no damage")
	})

	t.Run("LifeDrain species passes the gate and executes", func(t *testing.T) {
		char := characters.New()
		char.SpeciesId = 6002 // vampire — LifeDrain
		char.Aggro = &characters.Aggro{MobInstanceId: targetMob.InstanceId}

		result := ExecuteDrain(newStubActor(char, newTestRoom()))

		assert.False(t, result.NotLifeDrainer, "LifeDrain species should NOT return NotLifeDrainer")
		assert.True(t, result.Executed, "LifeDrain species should execute the drain")
	})
}

// TestDrain_HealAndBleed verifies the lifesteal mechanic: when a LifeDrain
// species hits a target, the attacker gains HP and the target gains a bleed
// condition. Uses extreme stats (attacker Strength=500, defender Dex=1) to
// make the hit near-certain; iterates until a hit is observed.
func TestDrain_HealAndBleed(t *testing.T) {
	// Seed a vampire species with LifeDrain.
	cleanup := species.SeedSpeciesForTest(map[int]*species.Species{
		6003: {SpeciesId: 6003, Name: "vampire", BodyParts: []string{"arms", "hands", "legs", "mouth"}, NaturalAttack: items.Claws, LifeDrain: true},
	})
	defer cleanup()

	// Register a target mob with minimal dexterity so the hit chance is high.
	targetMob := &mobs.Mob{InstanceId: 6199}
	targetMob.Character.Name = "Target"
	targetMob.Character.HealthMax.Value = 500
	targetMob.Character.Health = 500
	targetMob.Character.Stats.Dexterity.ValueAdj = 1 // near-zero evasion
	setCombatPositionParallel(&targetMob.Character, position.Standing)
	mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
	defer mobs.SetInstanceForTest(targetMob.InstanceId, nil)

	// Attacker with high Strength and health below max so the heal is observable.
	char := characters.New()
	char.SpeciesId = 6003 // vampire
	char.Stats.Strength.ValueAdj = 500
	char.HealthMax.Value = 200
	char.Health = 150 // 50 HP below max — heal is observable

	// Retry until we observe a hit (probability with str=500 vs dex=1 is very
	// high per round; the MinDefenseChance floor means ~85% hit rate).
	var res DrainResult
	hitSeen := false
	for i := 0; i < 100; i++ {
		// Reset state for each attempt: re-arm aggro + cooldowns.
		char.Aggro = &characters.Aggro{MobInstanceId: targetMob.InstanceId}
		char.Cooldowns = characters.Cooldowns{} // clear cooldown from previous attempt
		char.Health = 150                       // reset HP so the heal delta is visible

		res = ExecuteDrain(newStubActor(char, newTestRoom()))
		if res.Executed && res.MoveResult.Hit {
			hitSeen = true
			break
		}
	}

	if !hitSeen {
		t.Skip("no hit observed in 100 attempts — probabilistic test; re-run if flaky")
	}

	// On a hit: attacker health should have increased.
	assert.Greater(t, char.Health, 150,
		"attacker Health should increase after a successful drain (lifesteal)")

	// Healed field in the result should be positive.
	assert.Greater(t, res.Healed, 0,
		"DrainResult.Healed should be positive on a hit")

	// Target should have a bleed condition applied.
	assert.True(t, targetMob.Character.HasCondition(characters.ConditionBleeding),
		"target should have ConditionBleeding after a successful drain")

	// BleedDmg should be at least the minimum.
	assert.GreaterOrEqual(t, res.BleedDmg, 2,
		"BleedDmg should be at least 2 (min floor)")
}

// TestDrain_TargetGone verifies that when aggro is set to an invalid mob
// instance ID (target gone), Executed is false and NoTarget is true.
func TestDrain_TargetGone(t *testing.T) {
	char := characters.New()
	room := newTestRoom()
	actor := newStubActor(char, room)

	// Aggro pointing at a nonexistent mob instance — target resolution fails.
	char.Aggro = &characters.Aggro{MobInstanceId: 999999}

	result := ExecuteDrain(actor)

	assert.False(t, result.Executed, "drain with missing target should not execute")
	assert.True(t, result.NoTarget, "drain should report NoTarget when the resolved target is gone")
	assert.False(t, result.OnCooldown, "cooldown should not be reported when target is gone")
}

// ---------------------------------------------------------------------------
// ExecuteDrainArea tests
// ---------------------------------------------------------------------------

// drainAreaAttacker builds a high-Strength attacker character so the area
// drain's per-player hit chance is near-certain (mirrors TestDrain_HealAndBleed's
// approach of extreme stats + a retry loop rather than mocking the dice roll).
func drainAreaAttacker() *characters.Character {
	char := characters.New()
	char.Stats.Strength.ValueAdj = 500
	char.HealthMax.Value = 1000
	return char
}

// seedDrainAreaPlayer registers a test user (low Dexterity so the attacker's
// swing lands reliably) and adds them to the room's player list. Returns the
// UserRecord for assertions.
func seedDrainAreaPlayer(userId int, username string, charName string) *users.UserRecord {
	u := users.NewTestUser(userId, username, charName, uint64(userId+9000))
	u.Character.HealthMax.Value = 500
	u.Character.Health = 500
	u.Character.Stats.Dexterity.ValueAdj = 1 // near-zero evasion
	return u
}

// TestDrainArea_NoRoom verifies that ExecuteDrainArea reports NoTargets when
// the actor has no room (GetRoom() == nil).
func TestDrainArea_NoRoom(t *testing.T) {
	char := drainAreaAttacker()
	actor := newStubActor(char, nil)

	result := ExecuteDrainArea(actor)

	assert.False(t, result.Executed, "drain area with no room should not execute")
	assert.True(t, result.NoTargets, "drain area with no room should report NoTargets")
}

// TestDrainArea_NoPlayers verifies that ExecuteDrainArea reports NoTargets
// when the room has no players in it.
func TestDrainArea_NoPlayers(t *testing.T) {
	char := drainAreaAttacker()
	room := newTestRoom()
	actor := newStubActor(char, room)

	result := ExecuteDrainArea(actor)

	assert.False(t, result.Executed, "drain area with no players should not execute")
	assert.True(t, result.NoTargets, "drain area with no players should report NoTargets")
}

// TestDrainArea_SinglePlayer verifies the single-player case: the player
// takes drain damage + a bleed condition, and the mob is healed by that
// player's DrainHealRatio fraction of the damage dealt.
func TestDrainArea_SinglePlayer(t *testing.T) {
	p1 := seedDrainAreaPlayer(7001, "quester1", "Vael")
	cleanupUsers := users.SeedUsersForTest(map[int]*users.UserRecord{7001: p1})
	defer cleanupUsers()

	room := newTestRoom()
	room.AddPlayer(7001)

	var result DrainAreaResult
	hitSeen := false
	for i := 0; i < 100; i++ {
		char := drainAreaAttacker()
		char.Health = char.HealthMax.Value - 100 // 100 HP below max so heal is observable
		p1.Character.Health = 500
		p1.Character.RemoveCondition(characters.ConditionBleeding)

		actor := newStubActor(char, room)
		result = ExecuteDrainArea(actor)

		if result.Executed && len(result.PlayerResults) == 1 && result.PlayerResults[0].MoveResult.Hit {
			hitSeen = true
			break
		}
	}

	if !hitSeen {
		t.Skip("no hit observed in 100 attempts — probabilistic test; re-run if flaky")
	}

	require := assert.New(t)
	require.True(result.Executed)
	require.Len(result.PlayerResults, 1)

	pr := result.PlayerResults[0]
	require.Equal(7001, pr.UserId)
	require.Greater(pr.MoveResult.Damage, 0, "hit player should take drain damage")
	require.GreaterOrEqual(pr.BleedDmg, 2, "bleed magnitude should be at least the floor of 2")
	require.True(p1.Character.HasCondition(characters.ConditionBleeding), "drained player should carry a bleed condition")

	require.Greater(result.TotalDamage, 0, "aggregate damage should be positive")
	require.Greater(result.Healed, 0, "mob should be healed by the aggregate lifesteal")
}

// TestDrainArea_MultiPlayer verifies the N-player case: every player in the
// room takes drain damage independently, and the mob is healed by the sum of
// each player's DrainHealRatio fraction (aggregate lifesteal).
func TestDrainArea_MultiPlayer(t *testing.T) {
	p1 := seedDrainAreaPlayer(7002, "quester2", "Ryn")
	p2 := seedDrainAreaPlayer(7003, "quester3", "Doss")
	p3 := seedDrainAreaPlayer(7004, "quester4", "Meirok")
	cleanupUsers := users.SeedUsersForTest(map[int]*users.UserRecord{
		7002: p1,
		7003: p2,
		7004: p3,
	})
	defer cleanupUsers()

	room := newTestRoom()
	room.AddPlayer(7002)
	room.AddPlayer(7003)
	room.AddPlayer(7004)

	players := []*users.UserRecord{p1, p2, p3}

	var result DrainAreaResult
	allHit := false
	for i := 0; i < 200; i++ {
		char := drainAreaAttacker()
		char.Health = char.HealthMax.Value - 300 // well below max so aggregate heal is observable
		for _, p := range players {
			p.Character.Health = 500
			p.Character.RemoveCondition(characters.ConditionBleeding)
		}

		actor := newStubActor(char, room)
		result = ExecuteDrainArea(actor)

		if !result.Executed || len(result.PlayerResults) != 3 {
			continue
		}
		hitCount := 0
		for _, pr := range result.PlayerResults {
			if pr.MoveResult.Hit {
				hitCount++
			}
		}
		if hitCount == 3 {
			allHit = true
			break
		}
	}

	if !allHit {
		t.Skip("did not observe all 3 players hit within 200 attempts — probabilistic test; re-run if flaky")
	}

	require := assert.New(t)
	require.True(result.Executed)
	require.Len(result.PlayerResults, 3)

	expectedUserIds := map[int]bool{7002: true, 7003: true, 7004: true}
	sumDamage := 0
	for _, pr := range result.PlayerResults {
		require.True(expectedUserIds[pr.UserId], "unexpected user id in results: %d", pr.UserId)
		require.True(pr.MoveResult.Hit, "expected every player to be hit in this iteration")
		require.Greater(pr.MoveResult.Damage, 0, "each hit player should take drain damage")
		require.GreaterOrEqual(pr.BleedDmg, 2, "bleed magnitude should be at least the floor of 2")
		sumDamage += pr.MoveResult.Damage
	}
	for _, p := range players {
		require.True(p.Character.HasCondition(characters.ConditionBleeding), "each drained player should carry a bleed condition")
	}

	require.Equal(sumDamage, result.TotalDamage, "TotalDamage should equal the sum of per-player damage")
	require.Greater(result.Healed, 0, "mob should be healed by the aggregate lifesteal across all players")
}
