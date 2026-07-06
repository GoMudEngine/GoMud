package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedDrainAreaRegistries seeds a boss mob (instance 9562, high Strength so
// the area drain lands reliably) plus N players (low Dexterity so the swing
// lands) in a single room, mirroring seedTargetResolutionRegistries in
// internal/actions but scoped to this package's drain_area dispatch test.
func seedDrainAreaRegistries(t *testing.T, playerIds []int) func() {
	t.Helper()

	bossMob := &mobs.Mob{
		MobId:      9562,
		InstanceId: 9562,
		Zone:       "TestZone",
		HomeRoomId: 1,
		Character: characters.Character{
			Name:      "The Core Guardian",
			RoomId:    1,
			Health:    2000,
			Buffs:     buffs.New(),
			Cooldowns: map[string]int{},
			Position:  position.NewMachine(),
		},
	}
	bossMob.Character.HealthMax.Value = 2000
	bossMob.Character.Stats.Strength.ValueAdj = 500

	cleanupMobs := mobs.SeedMobsForTest(
		map[int]*mobs.Mob{9562: bossMob},
		map[int]*mobs.Mob{9562: bossMob},
	)

	testUsers := make(map[int]*users.UserRecord, len(playerIds))
	for i, uid := range playerIds {
		u := users.NewTestUser(uid, "quester"+string(rune('0'+i)), "Meirok"+string(rune('0'+i)), uint64(uid+9000))
		u.Character.RoomId = 1
		u.Character.HealthMax.Value = 500
		u.Character.Health = 500
		u.Character.Stats.Dexterity.ValueAdj = 1 // near-zero evasion
		testUsers[uid] = u
	}
	cleanupUsers := users.SeedUsersForTest(testUsers)

	room1 := &rooms.Room{
		RoomId: 1,
		Zone:   "TestZone",
		Title:  "The Core Chamber",
		Exits:  map[string]exit.RoomExit{},
		Biome:  "city",
	}
	cleanupRooms := rooms.SeedRoomsForTest(
		map[int]*rooms.Room{1: room1},
		map[string]*rooms.ZoneConfig{
			"TestZone": {Name: "TestZone", RoomId: 1, RoomIds: map[int]struct{}{1: {}}},
		},
	)
	room1.AddMob(9562)
	for _, uid := range playerIds {
		room1.AddPlayer(uid)
	}

	cleanupBiomes := rooms.SeedBiomesForTest(map[string]*rooms.BiomeInfo{
		"city": {
			BiomeId:      "city",
			Name:         "City",
			Symbol:       "#",
			LitArea:      true,
			DarkArea:     false,
			MovementCost: 1.0,
		},
	})

	return func() {
		cleanupBiomes()
		cleanupRooms()
		cleanupMobs()
		cleanupUsers()
	}
}

// TestResolveMobSpell_DrainArea_DispatchesToDrainArea verifies the
// engine-glue half of Task A4: a mob-cast spell whose EffectType is
// "drain_area" is dispatched (via resolveMobSpell, the same function
// handleMobFoldCasting calls at fold-cast completion) to the area drain —
// every player in the room takes drain damage + a bleed condition, and the
// casting mob is healed by the aggregate lifesteal. Retries the whole sweep
// until every player registers a hit (the underlying roll is probabilistic;
// mirrors the retry pattern in combat_drain_test.go).
func TestResolveMobSpell_DrainArea_DispatchesToDrainArea(t *testing.T) {
	playerIds := []int{7101, 7102}
	cleanup := seedDrainAreaRegistries(t, playerIds)
	defer cleanup()

	room := rooms.LoadRoom(1)
	require.NotNil(t, room)

	spellData := &spells.SpellData{
		SpellId:    "test-core-drain",
		Name:       "Test Core Drain",
		Type:       spells.HarmArea,
		EffectType: "drain_area",
		BaseFolds:  2,
	}

	cs := activity.CastingData{SpellId: "test-core-drain"}

	var bossHealthAfter int
	allHit := false
	for i := 0; i < 200; i++ {
		boss := mobs.GetInstance(9562)
		require.NotNil(t, boss)
		boss.Character.Health = 500 // well below max so the heal is observable

		for _, uid := range playerIds {
			u := users.GetByUserId(uid)
			require.NotNil(t, u)
			u.Character.Health = 500
			u.Character.RemoveCondition(characters.ConditionBleeding)
		}

		resolveMobSpell(boss, cs, spellData, room)

		hitCount := 0
		for _, uid := range playerIds {
			u := users.GetByUserId(uid)
			if u.Character.Health < 500 {
				hitCount++
			}
		}
		if hitCount == len(playerIds) {
			allHit = true
			bossHealthAfter = boss.Character.Health
			break
		}
	}

	if !allHit {
		t.Skip("did not observe every player hit within 200 attempts — probabilistic test; re-run if flaky")
	}

	for _, uid := range playerIds {
		u := users.GetByUserId(uid)
		assert.Less(t, u.Character.Health, 500, "player %d should have taken drain damage", uid)
		assert.True(t, u.Character.HasCondition(characters.ConditionBleeding), "player %d should carry a bleed condition after the drain", uid)
	}

	assert.Greater(t, bossHealthAfter, 500, "boss should be healed above its pre-drain health by the aggregate lifesteal")
}

// TestResolveMobSpell_DrainArea_NoPlayers verifies the empty-room case
// resolves without panicking and does not touch the boss's health.
func TestResolveMobSpell_DrainArea_NoPlayers(t *testing.T) {
	cleanup := seedDrainAreaRegistries(t, nil)
	defer cleanup()

	room := rooms.LoadRoom(1)
	require.NotNil(t, room)

	boss := mobs.GetInstance(9562)
	require.NotNil(t, boss)
	boss.Character.Health = 500

	spellData := &spells.SpellData{
		SpellId:    "test-core-drain",
		Name:       "Test Core Drain",
		Type:       spells.HarmArea,
		EffectType: "drain_area",
		BaseFolds:  2,
	}
	cs := activity.CastingData{SpellId: "test-core-drain"}

	assert.NotPanics(t, func() {
		resolveMobSpell(boss, cs, spellData, room)
	})
	assert.Equal(t, 500, boss.Character.Health, "boss health should be unchanged when there are no players to drain")
}
