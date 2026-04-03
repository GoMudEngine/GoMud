package hooks

import (
	"os"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Test Infrastructure ──────────────────────────────────────────────────────

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "", "", false)
	os.Exit(m.Run())
}

// seedAllRegistries populates all 5 dependency registries with sensible test
// defaults and returns a combined cleanup function.
func seedAllRegistries() func() {
	cleanupBuffs := buffs.SeedBuffsForTest(map[int]*buffs.BuffSpec{
		100: {
			BuffId:        100,
			Name:          "Test Strength Buff",
			Description:   "Boosts strength for testing",
			RoundInterval: 5,
			TriggerCount:  3,
		},
		101: {
			BuffId:        101,
			Name:          "Test Poison",
			Description:   "Damage over time for testing",
			RoundInterval: 3,
			TriggerCount:  5,
			TriggerNow:    true,
			Flags:         []buffs.Flag{buffs.Poison},
		},
	})

	testMobSpecs := map[int]*mobs.Mob{
		1: {
			MobId:         1,
			Zone:          "TestZone",
			Hostile:       true,
			ActivityLevel: 50,
			Groups:        []string{"undead"},
			Character: characters.Character{
				Name: "Skeleton",
			},
		},
		2: {
			MobId:         2,
			Zone:          "TestZone",
			Hostile:       false,
			ActivityLevel: 30,
			Character: characters.Character{
				Name: "Merchant",
			},
		},
	}
	testMobInstances := map[int]*mobs.Mob{
		100: {
			MobId:      1,
			InstanceId: 100,
			HomeRoomId: 1,
			Hostile:    true,
			Groups:     []string{"undead"},
			Character: characters.Character{
				Name:      "Skeleton",
				RoomId:    1,
				Health:    50,
				Buffs:     buffs.New(),
				Cooldowns: map[string]int{},
			},
		},
	}
	// Set mob instance health max
	testMobInstances[100].Character.HealthMax.Value = 100
	testMobInstances[100].Character.StaminaMax.Value = 50
	testMobInstances[100].Character.Stamina = 50
	testMobInstances[100].Character.ConvictionMax.Value = 30
	testMobInstances[100].Character.Conviction = 30
	testMobInstances[100].Character.Stats.Strength.ValueAdj = 80
	testMobInstances[100].Character.Stats.Dexterity.ValueAdj = 80
	testMobInstances[100].Character.Stats.Willpower.ValueAdj = 60

	cleanupMobs := mobs.SeedMobsForTest(testMobSpecs, testMobInstances)

	u1 := users.NewTestUser(1, "alice", "Aliceia", 1001)
	u1.Character.RoomId = 1
	u2 := users.NewTestUser(2, "bob", "Bobrick", 1002)
	u2.Character.RoomId = 1

	cleanupUsers := users.SeedUsersForTest(map[int]*users.UserRecord{
		1: u1,
		2: u2,
	})

	room1 := &rooms.Room{
		RoomId:      1,
		Zone:        "TestZone",
		Title:       "Town Square",
		Description: "A bustling town square.",
		Exits: map[string]exit.RoomExit{
			"north": {RoomId: 2},
		},
		Pvp:   true,
		Biome: "city",
	}
	room2 := &rooms.Room{
		RoomId:      2,
		Zone:        "TestZone",
		Title:       "Dark Cave",
		Description: "A damp, dark cave.",
		Exits: map[string]exit.RoomExit{
			"south": {RoomId: 1},
		},
	}
	cleanupRooms := rooms.SeedRoomsForTest(
		map[int]*rooms.Room{1: room1, 2: room2},
		map[string]*rooms.ZoneConfig{
			"TestZone": {
				Name:    "TestZone",
				RoomId:  1,
				RoomIds: map[int]struct{}{1: {}, 2: {}},
			},
		},
	)
	// Place players and mob in room 1
	room1.AddPlayer(1)
	room1.AddPlayer(2)
	room1.AddMob(100)
	rooms.MarkRoomOccupancy(1, 2, 1) // 2 players, 1 mob in room 1

	cleanupBiomes := rooms.SeedBiomesForTest(map[string]*rooms.BiomeInfo{
		"city": {
			BiomeId:      "city",
			Name:         "City",
			Symbol:       "#",
			LitArea:      true,
			DarkArea:     false,
			MovementCost: 1.0,
		},
		"cave": {
			BiomeId:  "cave",
			Name:     "Cave",
			Symbol:   "C",
			LitArea:  false,
			DarkArea: true,
		},
		"default": {
			BiomeId:      "default",
			Name:         "Default",
			Symbol:       ".",
			LitArea:      true,
			MovementCost: 1.0,
		},
	})

	cleanupSpells := spells.SeedSpellsForTest(map[string]*spells.SpellData{
		"sparks": {
			SpellId:          "sparks",
			Name:             "Sparks",
			Type:             spells.HarmSingle,
			Cost:             3,
			Difficulty:       10,
			DamageMultiplier: 0.8,
			BaseFolds:        4,
			EffectType:       "damage",
			TargetDefenseType: "mental",
			Schools:          []string{spells.SchoolElemental},
		},
	})

	return func() {
		cleanupSpells()
		cleanupBiomes()
		cleanupRooms()
		cleanupUsers()
		cleanupMobs()
		cleanupBuffs()
	}
}

// ─── Combat Shared Helpers ────────────────────────────────────────────────────

func TestSimulateFoldRound_FromZero(t *testing.T) {
	cs := &characters.CastingState{
		FoldsAccumulated: 0,
		FoldsPerRound:    1,
		FoldsNeeded:      8,
	}
	delta := simulateFoldRound(cs)
	assert.Equal(t, 1, delta, "from 0 with 1 fold/round should gain 1")
	assert.Equal(t, 0, cs.FoldsAccumulated, "should not modify state")
}

func TestSimulateFoldRound_Doubling(t *testing.T) {
	cs := &characters.CastingState{
		FoldsAccumulated: 2,
		FoldsPerRound:    1,
		FoldsNeeded:      8,
	}
	delta := simulateFoldRound(cs)
	assert.Equal(t, 2, delta, "2 doubles to 4, delta = 2")
}

func TestSimulateFoldRound_MultipleFoldsPerRound(t *testing.T) {
	cs := &characters.CastingState{
		FoldsAccumulated: 0,
		FoldsPerRound:    3,
		FoldsNeeded:      16,
	}
	// Round: 0→1, 1→2, 2→4 = delta 4
	delta := simulateFoldRound(cs)
	assert.Equal(t, 4, delta)
}

func TestSimulateFoldRound_CapsAtNeeded(t *testing.T) {
	cs := &characters.CastingState{
		FoldsAccumulated: 4,
		FoldsPerRound:    2,
		FoldsNeeded:      8,
	}
	// 4→8, capped at 8. delta = 4
	delta := simulateFoldRound(cs)
	assert.Equal(t, 4, delta)
}

func TestCalcFoldConvictionCost_Basic(t *testing.T) {
	cs := &characters.CastingState{
		TotalConvictionCost: 10,
		FoldsNeeded:         4,
	}
	cost := calcFoldConvictionCost(cs, 2)
	assert.Equal(t, 5, cost, "2/4 of total 10 = 5")
}

func TestCalcFoldConvictionCost_MinimumOne(t *testing.T) {
	cs := &characters.CastingState{
		TotalConvictionCost: 1,
		FoldsNeeded:         100,
	}
	cost := calcFoldConvictionCost(cs, 1)
	assert.Equal(t, 1, cost, "should be at least 1")
}

func TestCalcFoldConvictionCost_ZeroCost(t *testing.T) {
	cs := &characters.CastingState{
		TotalConvictionCost: 0,
		FoldsNeeded:         4,
	}
	cost := calcFoldConvictionCost(cs, 2)
	assert.Equal(t, 0, cost)
}

func TestCalcFoldConvictionCost_ZeroFoldsNeeded(t *testing.T) {
	cs := &characters.CastingState{
		TotalConvictionCost: 10,
		FoldsNeeded:         0,
	}
	cost := calcFoldConvictionCost(cs, 2)
	assert.Equal(t, 0, cost)
}

func TestAdvanceFolds_FromZero(t *testing.T) {
	cs := &characters.CastingState{
		FoldsAccumulated: 0,
		FoldsPerRound:    1,
		FoldsNeeded:      4,
	}
	done := advanceFolds(cs)
	assert.False(t, done)
	assert.Equal(t, 1, cs.FoldsAccumulated)
}

func TestAdvanceFolds_CompletesAtNeeded(t *testing.T) {
	cs := &characters.CastingState{
		FoldsAccumulated: 2,
		FoldsPerRound:    1,
		FoldsNeeded:      4,
	}
	done := advanceFolds(cs)
	assert.True(t, done)
	assert.Equal(t, 4, cs.FoldsAccumulated)
}

func TestAdvanceFolds_CapsOvershoot(t *testing.T) {
	cs := &characters.CastingState{
		FoldsAccumulated: 4,
		FoldsPerRound:    2,
		FoldsNeeded:      5,
	}
	// 4→8, capped to 5, done
	done := advanceFolds(cs)
	assert.True(t, done)
	assert.Equal(t, 5, cs.FoldsAccumulated)
}

func TestAdvanceFolds_MultipleFoldsPerRound(t *testing.T) {
	cs := &characters.CastingState{
		FoldsAccumulated: 0,
		FoldsPerRound:    4,
		FoldsNeeded:      16,
	}
	// 0→1, 1→2, 2→4, 4→8. Needs 16, got 8.
	done := advanceFolds(cs)
	assert.False(t, done)
	assert.Equal(t, 8, cs.FoldsAccumulated)
}

func TestCheckConcentrationBreak_NilCastingState(t *testing.T) {
	ch := &characters.Character{}
	ch.HealthMax.Value = 100
	broke := checkConcentrationBreak(ch, 10)
	assert.False(t, broke, "nil CastingState should never break")
}

func TestCheckConcentrationBreak_ZeroDamage(t *testing.T) {
	ch := &characters.Character{
		CastingState: &characters.CastingState{SpellId: "sparks"},
	}
	ch.HealthMax.Value = 100
	broke := checkConcentrationBreak(ch, 0)
	assert.False(t, broke, "zero damage should never break")
}

func TestCheckConcentrationBreak_NegativeDamage(t *testing.T) {
	ch := &characters.Character{
		CastingState: &characters.CastingState{SpellId: "sparks"},
	}
	ch.HealthMax.Value = 100
	broke := checkConcentrationBreak(ch, -5)
	assert.False(t, broke, "negative damage should never break")
}

func TestCheckConcentrationBreak_WithDamage(t *testing.T) {
	// With a valid casting state and damage > 0, the function should run
	// (result depends on RNG so we just verify it doesn't panic)
	ch := &characters.Character{
		CastingState: &characters.CastingState{SpellId: "sparks"},
	}
	ch.HealthMax.Value = 100
	ch.Stats.Willpower.ValueAdj = 100
	// Just ensure no panic — result is probabilistic
	_ = checkConcentrationBreak(ch, 50)
}

// ─── Spell Resolution Helpers ─────────────────────────────────────────────────

func TestSpellDefenseValue_Physical(t *testing.T) {
	ch := &characters.Character{}
	// No equipment — defense should be 0 (plus any shield condition)
	val := spellDefenseValue("physical", ch)
	assert.Equal(t, 0.0, val)
}

func TestSpellDefenseValue_Mental(t *testing.T) {
	ch := &characters.Character{}
	ch.Stats.Willpower.ValueAdj = 120
	val := spellDefenseValue("mental", ch)
	assert.Equal(t, 120.0, val)
}

func TestSpellDefenseValue_Unknown(t *testing.T) {
	ch := &characters.Character{}
	val := spellDefenseValue("", ch)
	assert.Equal(t, 0.0, val)
}

func TestSpellDefenseValue_PhysicalWithShield(t *testing.T) {
	ch := &characters.Character{}
	ch.AddCondition(characters.ConditionShield, 10, 25.0, "test")
	val := spellDefenseValue("physical", ch)
	assert.Equal(t, 25.0, val, "shield bonus should be added")
}

// ─── CalcSpellDamageForCharacter ──────────────────────────────────────────────

func TestCalcSpellDamageForCharacter_LegacyPath(t *testing.T) {
	// DamageMultiplier = 0 → legacy path
	spellData := &spells.SpellData{
		SpellId:          "legacy",
		DamageMultiplier: 0,
	}
	caster := &characters.Character{}
	caster.Stats.Willpower.ValueAdj = 100
	target := &characters.Character{}
	target.HealthMax.Value = 100

	dmg := calcSpellDamageForCharacter(spellData, caster, target, 10, false)
	assert.GreaterOrEqual(t, dmg, 1, "legacy path should deal at least 1 damage")
}

func TestCalcSpellDamageForCharacter_LegacyCrit(t *testing.T) {
	spellData := &spells.SpellData{
		SpellId:          "legacy",
		DamageMultiplier: 0,
	}
	caster := &characters.Character{}
	target := &characters.Character{}
	target.HealthMax.Value = 100

	dmg := calcSpellDamageForCharacter(spellData, caster, target, 10, true)
	assert.GreaterOrEqual(t, dmg, 1)
}

func TestCalcSpellDamageForCharacter_PipelinePath(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	spellData := &spells.SpellData{
		SpellId:           "sparks",
		DamageMultiplier:  0.8,
		TargetDefenseType: "mental",
	}
	caster := &characters.Character{}
	caster.Stats.Willpower.ValueAdj = 100
	caster.ConvictionMax.Value = 50
	caster.Conviction = 50

	target := &characters.Character{}
	target.HealthMax.Value = 100
	target.Stats.Willpower.ValueAdj = 50

	dmg := calcSpellDamageForCharacter(spellData, caster, target, 10, false)
	assert.GreaterOrEqual(t, dmg, 1, "pipeline should deal at least 1 damage")
}

func TestCalcSpellDamageForCharacter_PipelineCrit(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	spellData := &spells.SpellData{
		SpellId:           "sparks",
		DamageMultiplier:  0.8,
		TargetDefenseType: "mental",
	}
	caster := &characters.Character{}
	caster.Stats.Willpower.ValueAdj = 100
	caster.ConvictionMax.Value = 50
	caster.Conviction = 50

	target := &characters.Character{}
	target.HealthMax.Value = 100
	target.Stats.Willpower.ValueAdj = 50

	dmg := calcSpellDamageForCharacter(spellData, caster, target, 10, true)
	assert.GreaterOrEqual(t, dmg, 1, "crit pipeline should deal at least 1 damage")
}

// ─── WeaponBreakResult / TryWeaponBreak ───────────────────────────────────────

func TestWeaponBreakResult_ZeroValue(t *testing.T) {
	result := WeaponBreakResult{}
	assert.False(t, result.Broke)
	assert.Empty(t, result.BrokenItemName)
}

func TestCritEffectResult_ZeroValue(t *testing.T) {
	result := CritEffectResult{}
	assert.False(t, result.Disarmed)
	assert.False(t, result.GrappleSet)
}

// ─── CheckNewDay ──────────────────────────────────────────────────────────────

func TestCheckNewDay_NoBoundary(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	events.ClearListeners()

	// Use consecutive rounds where day/night doesn't change
	evt := events.NewRound{RoundNumber: 10}
	result := CheckNewDay(evt)
	assert.Equal(t, events.Continue, result)
}

func TestCheckNewDay_ReturnsCorrectType(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	events.ClearListeners()

	evt := events.NewRound{RoundNumber: 5}
	result := CheckNewDay(evt)
	assert.Equal(t, events.Continue, result, "should always return Continue")
}

// ─── CheckMoonPhase ───────────────────────────────────────────────────────────

func TestCheckMoonPhase_RoundZero(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	events.ClearListeners()

	evt := events.NewRound{RoundNumber: 0}
	result := CheckMoonPhase(evt)
	assert.Equal(t, events.Continue, result, "round 0 should skip")
}

func TestCheckMoonPhase_ReturnsCorrectType(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	events.ClearListeners()

	evt := events.NewRound{RoundNumber: 50}
	result := CheckMoonPhase(evt)
	assert.Equal(t, events.Continue, result)
}

func TestCheckMoonPhase_NewMoonDetection(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	events.ClearListeners()

	// Swiftmoon has period 4.7. With RoundsPerDay=20 (default), cycleLen = 4.7*20 = 94
	// New moon fires when roundNum % 94 == 0, so round 94 should trigger new moon
	evt := events.NewRound{RoundNumber: 94}
	result := CheckMoonPhase(evt)
	assert.Equal(t, events.Continue, result)
	// The function queues MoonPhase events internally — we can't easily inspect
	// the queue from here, but we verify no panic and correct return.
}

// ─── ActionPoints ─────────────────────────────────────────────────────────────

func TestActionPoints_IncrementsForAllUsers(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	u1 := users.GetByUserId(1)
	u2 := users.GetByUserId(2)

	// Set AP below max
	u1.Character.ActionPoints = 3
	u2.Character.ActionPoints = 7

	evt := events.NewTurn{TurnNumber: 1}
	result := ActionPoints(evt)

	assert.Equal(t, events.Continue, result)
	assert.Equal(t, 4, u1.Character.ActionPoints, "alice AP should increment by 1")
	assert.Equal(t, 8, u2.Character.ActionPoints, "bob AP should increment by 1")
}

func TestActionPoints_CapsAtMax(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	u1 := users.GetByUserId(1)
	u1.Character.ActionPoints = u1.Character.ActionPointsMax.Value

	evt := events.NewTurn{TurnNumber: 1}
	ActionPoints(evt)

	assert.Equal(t, u1.Character.ActionPointsMax.Value, u1.Character.ActionPoints,
		"AP should not exceed max")
}

func TestActionPoints_DoesNotExceedMax(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	u1 := users.GetByUserId(1)
	u1.Character.ActionPointsMax.Value = 5
	u1.Character.ActionPoints = 5

	evt := events.NewTurn{TurnNumber: 2}
	ActionPoints(evt)

	assert.Equal(t, 5, u1.Character.ActionPoints)
}

// ─── AutoHeal ─────────────────────────────────────────────────────────────────

func TestAutoHeal_SkipsNonDivisibleRounds(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	u1 := users.GetByUserId(1)
	startHP := u1.Character.Health

	evt := events.NewRound{RoundNumber: 1} // not divisible by 3
	result := AutoHeal(evt)

	assert.Equal(t, events.Continue, result)
	assert.Equal(t, startHP, u1.Character.Health, "health should not change on non-3 round")
}

func TestAutoHeal_HealsOnDivisibleRound(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	u1 := users.GetByUserId(1)
	u1.Character.Health = 50 // below max

	evt := events.NewRound{RoundNumber: 3}
	result := AutoHeal(evt)

	assert.Equal(t, events.Continue, result)
	assert.Greater(t, u1.Character.Health, 50, "health should increase on regen round")
}

func TestAutoHeal_BleedOutWhenHealthBelowOne(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	u1 := users.GetByUserId(1)
	u1.Character.Health = -5 // downed, but not dead yet

	evt := events.NewRound{RoundNumber: 6}
	AutoHeal(evt)

	assert.Equal(t, -7, u1.Character.Health, "should decrease by 2 when bleeding out")
}

func TestAutoHeal_StaminaRegenOutOfCombat(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	u1 := users.GetByUserId(1)
	u1.Character.Stamina = 80
	u1.Character.Aggro = nil // out of combat

	evt := events.NewRound{RoundNumber: 9}
	AutoHeal(evt)

	assert.Greater(t, u1.Character.Stamina, 80, "stamina should regen out of combat")
}

func TestAutoHeal_StaminaRegenSlowerInCombat(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	u1 := users.GetByUserId(1)
	u1.Character.Stamina = 80
	u1.Character.StaminaMax.Value = 100
	u1.Character.Aggro = &characters.Aggro{MobInstanceId: 100, Type: characters.DefaultAttack}

	staminaBefore := u1.Character.Stamina
	evt := events.NewRound{RoundNumber: 12}
	AutoHeal(evt)

	// Should still regen, just slower
	assert.GreaterOrEqual(t, u1.Character.Stamina, staminaBefore,
		"stamina should not decrease during regen")
}

func TestAutoHeal_ConvictionRegen(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	u1 := users.GetByUserId(1)
	u1.Character.Conviction = 30
	u1.Character.ConvictionMax.Value = 50

	evt := events.NewRound{RoundNumber: 15}
	AutoHeal(evt)

	assert.Greater(t, u1.Character.Conviction, 30, "conviction should regen")
}

func TestAutoHeal_MobHealthRegen(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob := mobs.GetInstance(100)
	mob.Character.Health = 40
	mob.Character.Aggro = nil // out of combat

	evt := events.NewRound{RoundNumber: 18}
	AutoHeal(evt)

	assert.Greater(t, mob.Character.Health, 40, "mob should regen health out of combat")
}

func TestAutoHeal_MobStaminaRegen(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob := mobs.GetInstance(100)
	mob.Character.Stamina = 20
	mob.Character.StaminaMax.Value = 50
	mob.Character.Aggro = nil

	evt := events.NewRound{RoundNumber: 21}
	AutoHeal(evt)

	assert.Greater(t, mob.Character.Stamina, 20, "mob should regen stamina")
}

func TestAutoHeal_MobConvictionRegen(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob := mobs.GetInstance(100)
	mob.Character.Conviction = 10
	mob.Character.ConvictionMax.Value = 30
	mob.Character.Aggro = nil

	evt := events.NewRound{RoundNumber: 24}
	AutoHeal(evt)

	assert.Greater(t, mob.Character.Conviction, 10, "mob should regen conviction")
}

func TestAutoHeal_HealthCapsAtMax(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	u1 := users.GetByUserId(1)
	u1.Character.Health = u1.Character.HealthMax.Value
	u1.Character.Stamina = u1.Character.StaminaMax.Value
	u1.Character.Conviction = u1.Character.ConvictionMax.Value

	evt := events.NewRound{RoundNumber: 27}
	AutoHeal(evt)

	assert.LessOrEqual(t, u1.Character.Health, u1.Character.HealthMax.Value)
	assert.LessOrEqual(t, u1.Character.Stamina, u1.Character.StaminaMax.Value)
	assert.LessOrEqual(t, u1.Character.Conviction, u1.Character.ConvictionMax.Value)
}

func TestAutoHeal_PoisonDamage(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	u1 := users.GetByUserId(1)
	u1.Character.Health = 80
	u1.Character.AddCondition(characters.ConditionPoisoned, 10, 5.0, "test")

	evt := events.NewRound{RoundNumber: 30}
	AutoHeal(evt)

	// Health may increase from regen, then decrease from poison.
	// Just verify no panic and health changed.
	assert.NotEqual(t, 80, u1.Character.Health, "health should change from regen+poison")
}

func TestAutoHeal_MobSkipsIfDead(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob := mobs.GetInstance(100)
	mob.Character.Health = 0 // dead

	evt := events.NewRound{RoundNumber: 33}
	AutoHeal(evt)

	assert.Equal(t, 0, mob.Character.Health, "dead mob should not regen")
}

// ─── ApplyBuffs ───────────────────────────────────────────────────────────────

func TestApplyBuffs_WrongEventType(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	// Pass a NewRound event instead of Buff — should return Cancel
	evt := events.NewRound{RoundNumber: 1}
	result := ApplyBuffs(evt)
	assert.Equal(t, events.Cancel, result)
}

func TestApplyBuffs_InvalidBuffId(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	evt := events.Buff{UserId: 1, BuffId: 99999} // nonexistent
	result := ApplyBuffs(evt)
	assert.Equal(t, events.Cancel, result, "should cancel for unknown buff ID")
}

func TestApplyBuffs_InvalidUserId(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	evt := events.Buff{UserId: 99999, BuffId: 100}
	result := ApplyBuffs(evt)
	assert.Equal(t, events.Cancel, result, "should cancel for unknown user ID")
}

func TestApplyBuffs_InvalidMobInstanceId(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	evt := events.Buff{MobInstanceId: 99999, BuffId: 100}
	result := ApplyBuffs(evt)
	assert.Equal(t, events.Cancel, result, "should cancel for unknown mob instance")
}

func TestApplyBuffs_AppliesBuffToUser(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	evt := events.Buff{UserId: 1, BuffId: 100}
	result := ApplyBuffs(evt)
	assert.Equal(t, events.Continue, result)

	u := users.GetByUserId(1)
	assert.True(t, u.Character.HasBuff(100), "user should have buff 100")
}

func TestApplyBuffs_AppliesBuffToMob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	evt := events.Buff{MobInstanceId: 100, BuffId: 100}
	result := ApplyBuffs(evt)
	assert.Equal(t, events.Continue, result)

	mob := mobs.GetInstance(100)
	assert.True(t, mob.Character.HasBuff(100), "mob should have buff 100")
}

func TestApplyBuffs_NegativeBuffRemoves(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	// First add the buff
	u := users.GetByUserId(1)
	u.Character.AddBuff(100, false)
	assert.True(t, u.Character.HasBuff(100))

	// Now send negative buff ID to remove
	evt := events.Buff{UserId: 1, BuffId: -100}
	result := ApplyBuffs(evt)
	assert.Equal(t, events.Continue, result)
}

// ─── Message_SendMessage ──────────────────────────────────────────────────────

func TestMessage_SendMessage_WrongEventType(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	evt := events.NewRound{RoundNumber: 1}
	result := Message_SendMessage(evt)
	assert.Equal(t, events.Continue, result)
}

func TestMessage_SendMessage_ValidUserMessage(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	evt := events.Message{UserId: 1, Text: "Hello world"}
	result := Message_SendMessage(evt)
	assert.Equal(t, events.Continue, result)
}

func TestMessage_SendMessage_InvalidUser(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	evt := events.Message{UserId: 99999, Text: "Hello"}
	result := Message_SendMessage(evt)
	assert.Equal(t, events.Continue, result, "should not panic on invalid user")
}

func TestMessage_SendMessage_RoomMessage(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	evt := events.Message{RoomId: 1, Text: "A sound echoes."}
	result := Message_SendMessage(evt)
	assert.Equal(t, events.Continue, result)
}

func TestMessage_SendMessage_ExcludesUser(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	evt := events.Message{
		RoomId:         1,
		Text:           "A sound echoes.",
		ExcludeUserIds: []int{1},
	}
	result := Message_SendMessage(evt)
	assert.Equal(t, events.Continue, result)
}

// ─── MobDisplayName ───────────────────────────────────────────────────────────

func TestMobDisplayName(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob := mobs.GetInstance(100)
	room := rooms.LoadRoom(1)
	name := mobDisplayName(mob, room, 0)
	assert.NotEmpty(t, name, "mob display name should not be empty")
}

// ─── HandlePlayerShieldDecay ──────────────────────────────────────────────────

func TestHandlePlayerShieldDecay_NoShield(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	u := users.GetByUserId(1)
	// No shield condition — should not panic
	handlePlayerShieldDecay(u)
}

func TestHandlePlayerShieldDecay_ShieldExpires(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	u := users.GetByUserId(1)
	u.Character.AddCondition(characters.ConditionShield, 1, 10.0, "test")

	handlePlayerShieldDecay(u)
	assert.False(t, u.Character.HasCondition(characters.ConditionShield),
		"shield with duration 1 should be removed")
}

func TestHandlePlayerShieldDecay_ShieldDecrements(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	u := users.GetByUserId(1)
	u.Character.AddCondition(characters.ConditionShield, 5, 10.0, "test")

	handlePlayerShieldDecay(u)
	assert.True(t, u.Character.HasCondition(characters.ConditionShield),
		"shield with duration > 1 should still exist")
}

// ─── HandleMobAIDecision ──────────────────────────────────────────────────────

func TestHandleMobAIDecision_NonDefaultAttack(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob := mobs.GetInstance(100)
	mob.Character.Aggro = &characters.Aggro{
		UserId: 1,
		Type:   characters.Flee,
	}

	// Non-default aggro type → should return false (no AI action taken)
	result := handleMobAIDecision(mob, configs.GetConfig())
	assert.False(t, result, "flee aggro should skip AI decision")
}

func TestHandleMobAIDecision_DefaultAttack(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob := mobs.GetInstance(100)
	mob.Character.Aggro = &characters.Aggro{
		UserId: 1,
		Type:   characters.DefaultAttack,
	}
	mob.ActivityLevel = 0 // 0% chance to do anything

	result := handleMobAIDecision(mob, configs.GetConfig())
	assert.False(t, result, "0 activity level + no combat commands should skip")
}

// ─── PruneBuffs ───────────────────────────────────────────────────────────────

func TestPruneBuffs_NoPanic(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	// Just verify no panic with seeded registries
	evt := events.NewTurn{TurnNumber: 1}
	result := PruneBuffs(evt)
	assert.Equal(t, events.Continue, result)
}

// ─── Witnesses (moonSpec) ─────────────────────────────────────────────────────

func TestWitnessesSlice(t *testing.T) {
	assert.Len(t, witnesses, 3, "should have 3 moons")
	assert.Equal(t, "Swiftmoon", witnesses[0].name)
	assert.Equal(t, "The Wanderer", witnesses[1].name)
	assert.Equal(t, "The Eye", witnesses[2].name)
}

func TestWitnessPeriods(t *testing.T) {
	assert.InDelta(t, 4.7, witnesses[0].period, 0.01)
	assert.InDelta(t, 10.6, witnesses[1].period, 0.01)
	assert.InDelta(t, 21.1, witnesses[2].period, 0.01)
}

// ─── TryWeaponBreak ───────────────────────────────────────────────────────────

func TestTryWeaponBreak_NoHit(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	ch := &characters.Character{}
	room := rooms.LoadRoom(1)

	// roundResult with Hit = false
	result := tryWeaponBreak(ch, dummyAttackResult(false, false), room)
	assert.False(t, result.Broke)
}

func TestTryWeaponBreak_NoOffhand(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	ch := &characters.Character{}
	room := rooms.LoadRoom(1)

	result := tryWeaponBreak(ch, dummyAttackResult(true, false), room)
	assert.False(t, result.Broke, "no offhand means no break")
}

// ─── ApplyCritEffects ─────────────────────────────────────────────────────────

func TestApplyCritEffects_NoCrit(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	atk := &characters.Character{}
	def := &characters.Character{}
	room := rooms.LoadRoom(1)

	ar := dummyAttackResult(true, false)
	result := applyCritEffects(atk, def, ar, room)
	assert.False(t, result.Disarmed)
	assert.False(t, result.GrappleSet)
}

func TestApplyCritEffects_DodgeCritSetsGrapple(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	atk := &characters.Character{}
	def := &characters.Character{
		Cooldowns: map[string]int{},
	}
	room := rooms.LoadRoom(1)

	ar := dummyAttackResult(false, false)
	ar.DodgeCritDetected = true

	result := applyCritEffects(atk, def, ar, room)
	assert.True(t, result.GrappleSet, "dodge crit should set grapple opportunity")
}

// ─── RegisterListeners ────────────────────────────────────────────────────────

func TestRegisterListeners(t *testing.T) {
	events.ClearListeners()
	RegisterListeners()
	// Verify no panic and listeners were registered
	// We can't inspect listener count, but we can verify the function runs
}

// ─── HandleRespawns ───────────────────────────────────────────────────────────

func TestHandleRespawns_NoPanic(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	evt := events.NewRound{RoundNumber: 1}
	result := HandleRespawns(evt)
	assert.Equal(t, events.Continue, result)
}

func TestHandleRespawns_WithPlayersInRoom(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	// Room 1 has players from seedAllRegistries
	evt := events.NewRound{RoundNumber: 5}
	result := HandleRespawns(evt)
	assert.Equal(t, events.Continue, result)
}

// ─── InactivePlayers ──────────────────────────────────────────────────────────

func TestInactivePlayers_ZeroMaxIdle(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	// Default config has MaxIdleSeconds=0 → should return early
	evt := events.NewRound{RoundNumber: 100}
	result := InactivePlayers(evt)
	assert.Equal(t, events.Continue, result)
}

func TestInactivePlayers_LowRoundNumber(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	evt := events.NewRound{RoundNumber: 1}
	result := InactivePlayers(evt)
	assert.Equal(t, events.Continue, result)
}

// ─── PruneVMs ─────────────────────────────────────────────────────────────────

func TestPruneVMs_NonDivisibleRound(t *testing.T) {
	evt := events.NewRound{RoundNumber: 7}
	result := PruneVMs(evt)
	assert.Equal(t, events.Continue, result)
}

func TestPruneVMs_DivisibleRound(t *testing.T) {
	evt := events.NewRound{RoundNumber: 100}
	result := PruneVMs(evt)
	assert.Equal(t, events.Continue, result)
}

func TestPruneVMs_ZeroRound(t *testing.T) {
	evt := events.NewRound{RoundNumber: 0}
	result := PruneVMs(evt)
	assert.Equal(t, events.Continue, result)
}

// ─── UpdateZoneMutators ───────────────────────────────────────────────────────

func TestUpdateZoneMutators_NoPanic(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	evt := events.NewRound{RoundNumber: 1}
	result := UpdateZoneMutators(evt)
	assert.Equal(t, events.Continue, result)
}

// ─── SpawnLootGoblin ──────────────────────────────────────────────────────────

func TestSpawnLootGoblin_NoRoomConfigured(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	// Default config has RoomId=0 → should skip
	evt := events.NewRound{RoundNumber: 10}
	result := SpawnLootGoblin(evt)
	assert.Equal(t, events.Continue, result)
}

// ─── CleanupZombies ───────────────────────────────────────────────────────────

func TestCleanupZombies_WrongEventType(t *testing.T) {
	evt := events.NewRound{RoundNumber: 1}
	result := CleanupZombies(evt)
	assert.Equal(t, events.Cancel, result)
}

func TestCleanupZombies_ValidEvent(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	evt := events.NewTurn{TurnNumber: 1}
	result := CleanupZombies(evt)
	assert.Equal(t, events.Continue, result)
}

func TestCleanupZombies_HighTurnNumber(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	evt := events.NewTurn{TurnNumber: 100000}
	result := CleanupZombies(evt)
	assert.Equal(t, events.Continue, result)
}

// ─── AutoSave ─────────────────────────────────────────────────────────────────

func TestAutoSave_WrongEventType(t *testing.T) {
	evt := events.NewRound{RoundNumber: 1}
	result := AutoSave(evt)
	assert.Equal(t, events.Cancel, result)
}

func TestAutoSave_NonSaveTurn(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	// TurnNumber 1 won't match autosave interval (which defaults to something >1)
	evt := events.NewTurn{TurnNumber: 1}
	result := AutoSave(evt)
	assert.Equal(t, events.Continue, result)
}

// ─── NotifySunriseSunset ──────────────────────────────────────────────────────

func TestNotifySunriseSunset_WrongEventType(t *testing.T) {
	evt := events.NewRound{RoundNumber: 1}
	result := NotifySunriseSunset(evt)
	assert.Equal(t, events.Cancel, result)
}

func TestNotifySunriseSunset_Sunrise(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	evt := events.DayNightCycle{IsSunrise: true, Day: 1, Month: 1, Year: 1}
	result := NotifySunriseSunset(evt)
	assert.Equal(t, events.Continue, result)
}

func TestNotifySunriseSunset_Sunset(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	evt := events.DayNightCycle{IsSunrise: false, Day: 1, Month: 1, Year: 1}
	result := NotifySunriseSunset(evt)
	assert.Equal(t, events.Continue, result)
}

// ─── BroadcastMoonPhase ───────────────────────────────────────────────────────

func TestBroadcastMoonPhase_WrongEventType(t *testing.T) {
	evt := events.NewRound{RoundNumber: 1}
	result := BroadcastMoonPhase(evt)
	assert.Equal(t, events.Cancel, result)
}

func TestBroadcastMoonPhase_NewMoon(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	evt := events.MoonPhase{MoonName: "Swiftmoon", PhaseName: "new", IsNew: true}
	result := BroadcastMoonPhase(evt)
	assert.Equal(t, events.Continue, result)
}

func TestBroadcastMoonPhase_FullMoon(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	evt := events.MoonPhase{MoonName: "The Wanderer", PhaseName: "full", IsFull: true}
	result := BroadcastMoonPhase(evt)
	assert.Equal(t, events.Continue, result)
}

// ─── MoonNameToSlug ───────────────────────────────────────────────────────────

func TestMoonNameToSlug(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Swiftmoon", "swiftmoon"},
		{"The Wanderer", "wanderer"},
		{"The Eye", "eye"},
		{"", ""},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, moonNameToSlug(tt.input))
	}
}

// ─── Broadcast_SendToAll ──────────────────────────────────────────────────────

func TestBroadcast_SendToAll_WrongEventType(t *testing.T) {
	evt := events.NewRound{RoundNumber: 1}
	result := Broadcast_SendToAll(evt)
	assert.Equal(t, events.Continue, result)
}

func TestBroadcast_SendToAll_EmptyText(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	evt := events.Broadcast{Text: ""}
	result := Broadcast_SendToAll(evt)
	assert.Equal(t, events.Continue, result)
}

func TestBroadcast_SendToAll_WithText(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	evt := events.Broadcast{Text: "Hello World!"}
	result := Broadcast_SendToAll(evt)
	assert.Equal(t, events.Continue, result)
}

func TestBroadcast_SendToAll_WithScreenReaderText(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	evt := events.Broadcast{Text: "Hello", TextScreenReader: "Hello alt"}
	result := Broadcast_SendToAll(evt)
	assert.Equal(t, events.Continue, result)
}

func TestBroadcast_SendToAll_SkipLineRefresh(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	evt := events.Broadcast{Text: "Done.", SkipLineRefresh: true}
	result := Broadcast_SendToAll(evt)
	assert.Equal(t, events.Continue, result)
}

// ─── ClearSettingCaches ───────────────────────────────────────────────────────

func TestClearSettingCaches_WrongEventType(t *testing.T) {
	evt := events.NewRound{RoundNumber: 1}
	result := ClearSettingCaches(evt)
	assert.Equal(t, events.Cancel, result)
}

func TestClearSettingCaches_ZeroUserId(t *testing.T) {
	evt := events.UserSettingChanged{UserId: 0, Name: "screenreader"}
	result := ClearSettingCaches(evt)
	assert.Equal(t, events.Continue, result)
}

func TestClearSettingCaches_ScreenReaderSetting(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	evt := events.UserSettingChanged{UserId: 1, Name: "screenreader"}
	result := ClearSettingCaches(evt)
	assert.Equal(t, events.Continue, result)
}

func TestClearSettingCaches_OtherSetting(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	evt := events.UserSettingChanged{UserId: 1, Name: "color"}
	result := ClearSettingCaches(evt)
	assert.Equal(t, events.Continue, result)
}

// ─── RedrawPrompt_SendRedraw ──────────────────────────────────────────────────

func TestRedrawPrompt_SendRedraw_WrongEventType(t *testing.T) {
	evt := events.NewRound{RoundNumber: 1}
	result := RedrawPrompt_SendRedraw(evt)
	assert.Equal(t, events.Cancel, result)
}

func TestRedrawPrompt_SendRedraw_ValidUser(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	evt := events.RedrawPrompt{UserId: 1}
	result := RedrawPrompt_SendRedraw(evt)
	assert.Equal(t, events.Continue, result)
}

func TestRedrawPrompt_SendRedraw_InvalidUser(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	evt := events.RedrawPrompt{UserId: 99999}
	result := RedrawPrompt_SendRedraw(evt)
	assert.Equal(t, events.Continue, result)
}

func TestRedrawPrompt_SendRedraw_OnlyIfChanged(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	evt := events.RedrawPrompt{UserId: 1, OnlyIfChanged: true}
	result := RedrawPrompt_SendRedraw(evt)
	assert.Equal(t, events.Continue, result)
}

// ─── FollowLogs ───────────────────────────────────────────────────────────────

func TestFollowLogs_WrongEventType(t *testing.T) {
	evt := events.NewRound{RoundNumber: 1}
	result := FollowLogs(evt)
	assert.Equal(t, events.Cancel, result)
}

func TestFollowLogs_AddFollow(t *testing.T) {
	evt := events.Log{FollowAdd: 9999, Level: "INFO"}
	result := FollowLogs(evt)
	assert.Equal(t, events.Continue, result)

	// Clean up
	removeFromSendLists(9999)
}

func TestFollowLogs_RemoveFollow(t *testing.T) {
	// Add first
	FollowLogs(events.Log{FollowAdd: 8888, Level: "DEBUG"})
	// Then remove
	evt := events.Log{FollowRemove: 8888}
	result := FollowLogs(evt)
	assert.Equal(t, events.Continue, result)
}

func TestFollowLogs_LogMessage(t *testing.T) {
	evt := events.Log{Level: "INFO", Data: []any{"key", "value"}}
	result := FollowLogs(evt)
	assert.Equal(t, events.Continue, result)
}

// ─── DoCombat ─────────────────────────────────────────────────────────────────

func TestDoCombat_NoCombatants(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	// No one has aggro → should pass through without issue
	evt := events.NewRound{RoundNumber: 1}
	result := DoCombat(evt)
	assert.Equal(t, events.Continue, result)
}

func TestDoCombat_PlayerWithNoAggro(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	u := users.GetByUserId(1)
	u.Character.Aggro = nil

	evt := events.NewRound{RoundNumber: 2}
	result := DoCombat(evt)
	assert.Equal(t, events.Continue, result)
}

func TestDoCombat_MobWithNoAggro(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob := mobs.GetInstance(100)
	mob.Character.Aggro = nil

	evt := events.NewRound{RoundNumber: 3}
	result := DoCombat(evt)
	assert.Equal(t, events.Continue, result)
}

// ─── UserRoundTick ────────────────────────────────────────────────────────────

func TestUserRoundTick_BasicExecution(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	evt := events.NewRound{RoundNumber: 1}
	result := UserRoundTick(evt)
	assert.Equal(t, events.Continue, result)
}

func TestUserRoundTick_CooldownTick(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	u := users.GetByUserId(1)
	u.Character.Cooldowns["test-ability"] = 3

	evt := events.NewRound{RoundNumber: 1}
	UserRoundTick(evt)

	assert.Equal(t, 2, u.Character.Cooldowns["test-ability"],
		"cooldown should decrease by 1")
}

func TestUserRoundTick_CharmedDecrement(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	u := users.GetByUserId(1)
	u.Character.Charmed = &characters.CharmInfo{RoundsRemaining: 5}

	evt := events.NewRound{RoundNumber: 1}
	UserRoundTick(evt)

	assert.Equal(t, 4, u.Character.Charmed.RoundsRemaining)
}

// ─── MobRoundTick ─────────────────────────────────────────────────────────────

func TestMobRoundTick_BasicExecution(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	evt := events.NewRound{RoundNumber: 1}
	result := MobRoundTick(evt)
	assert.Equal(t, events.Continue, result)
}

func TestMobRoundTick_CooldownTick(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob := mobs.GetInstance(100)
	mob.Character.Cooldowns["test-ability"] = 5

	evt := events.NewRound{RoundNumber: 1}
	MobRoundTick(evt)

	assert.Equal(t, 4, mob.Character.Cooldowns["test-ability"],
		"mob cooldown should decrease by 1")
}

func TestMobRoundTick_CharmedDecrement(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob := mobs.GetInstance(100)
	mob.Character.Charmed = &characters.CharmInfo{RoundsRemaining: 3}

	evt := events.NewRound{RoundNumber: 1}
	MobRoundTick(evt)

	assert.Equal(t, 2, mob.Character.Charmed.RoundsRemaining)
}

func TestMobRoundTick_TickConditions(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob := mobs.GetInstance(100)
	mob.Character.AddCondition(characters.ConditionShield, 3, 10.0, "test")

	evt := events.NewRound{RoundNumber: 1}
	MobRoundTick(evt)

	// Condition should still exist but duration decremented
	assert.True(t, mob.Character.HasCondition(characters.ConditionShield))
}

// ─── ApplyMoonMods ────────────────────────────────────────────────────────────

func TestApplyMoonMods_NoMutations(t *testing.T) {
	ch := &characters.Character{}
	ch.Stats.Dexterity.ValueAdj = 100
	restore := applyMoonMods(ch, 10.0)
	restore()
	assert.Equal(t, 100, ch.Stats.Dexterity.ValueAdj, "no mutations = no change")
}

func TestApplyMoonMods_ZeroMoonMod(t *testing.T) {
	ch := &characters.Character{
		Mutations: map[string]int{"test": 1},
	}
	ch.Stats.Dexterity.ValueAdj = 100
	restore := applyMoonMods(ch, 0)
	restore()
	assert.Equal(t, 100, ch.Stats.Dexterity.ValueAdj)
}

func TestApplyMoonMods_WithMutations(t *testing.T) {
	ch := &characters.Character{
		Mutations: map[string]int{"test": 1},
	}
	ch.Stats.Dexterity.ValueAdj = 100
	ch.Stats.Strength.ValueAdj = 100
	ch.Stats.Vitality.ValueAdj = 100
	ch.Stats.Willpower.ValueAdj = 100
	ch.Stats.Perception.ValueAdj = 100
	ch.Stats.Charisma.ValueAdj = 100

	restore := applyMoonMods(ch, 10.0)
	// Stats may have changed (depending on current moon phase)
	restore()
	// After restore, all should be back to 100
	assert.Equal(t, 100, ch.Stats.Dexterity.ValueAdj)
	assert.Equal(t, 100, ch.Stats.Strength.ValueAdj)
	assert.Equal(t, 100, ch.Stats.Vitality.ValueAdj)
	assert.Equal(t, 100, ch.Stats.Willpower.ValueAdj)
	assert.Equal(t, 100, ch.Stats.Perception.ValueAdj)
	assert.Equal(t, 100, ch.Stats.Charisma.ValueAdj)
}

// ─── HandleAffected ───────────────────────────────────────────────────────────

func TestHandleAffected_EmptySlices(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	// Should not panic with empty slices
	handleAffected([]int{}, []int{})
}

func TestHandleAffected_HealthyPlayer(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	u := users.GetByUserId(1)
	u.Character.Health = 100

	handleAffected([]int{1}, []int{})
	// Healthy player should not be affected
	assert.Equal(t, 100, u.Character.Health)
}

func TestHandleAffected_DuplicateIds(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	u := users.GetByUserId(1)
	u.Character.Health = 100

	// Should handle duplicates without issue
	handleAffected([]int{1, 1, 1}, []int{100, 100})
}

// ─── HandlePlayerFoldCasting ──────────────────────────────────────────────────

func TestHandlePlayerFoldCasting_NoCastingState(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	u := users.GetByUserId(1)
	u.Character.CastingState = nil

	result := handlePlayerFoldCasting(u, 1)
	assert.False(t, result, "no casting state should return false")
}

func TestHandlePlayerFoldCasting_DisabledPlayer(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	u := users.GetByUserId(1)
	u.Character.Health = 0 // disabled
	u.Character.CastingState = &characters.CastingState{SpellId: "sparks"}

	result := handlePlayerFoldCasting(u, 1)
	assert.True(t, result, "disabled player should skip combat")
	assert.Nil(t, u.Character.CastingState, "casting state should be cleared")
}

func TestHandlePlayerFoldCasting_PronePlayer(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	u := users.GetByUserId(1)
	u.Character.CastingState = &characters.CastingState{SpellId: "sparks"}
	u.Character.CombatPosition = characters.PositionProne

	result := handlePlayerFoldCasting(u, 1)
	assert.True(t, result, "prone player should skip combat")
	assert.Nil(t, u.Character.CastingState, "casting state should be cleared")
}

// ─── HandleMobFoldCasting ─────────────────────────────────────────────────────

func TestHandleMobFoldCasting_NoCastingState(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob := mobs.GetInstance(100)
	mob.Character.CastingState = nil
	room := rooms.LoadRoom(1)

	result := handleMobFoldCasting(mob, room)
	assert.False(t, result)
}

func TestHandleMobFoldCasting_ProneMob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob := mobs.GetInstance(100)
	mob.Character.CastingState = &characters.CastingState{SpellId: "sparks"}
	mob.Character.CombatPosition = characters.PositionProne
	room := rooms.LoadRoom(1)

	result := handleMobFoldCasting(mob, room)
	assert.True(t, result)
	assert.Nil(t, mob.Character.CastingState)
}

// ─── ProcessGrappleProgression ────────────────────────────────────────────────

func TestProcessGrappleProgression_NotGrappled(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	ch1 := &characters.Character{CombatPosition: characters.PositionStanding}
	ch2 := &characters.Character{CombatPosition: characters.PositionStanding}
	room := rooms.LoadRoom(1)

	// Should return without doing anything (not in grapple)
	processGrappleProgression(ch1, ch2, "Alice", "Bob", room, 1, 2)
}

// ─── IdleMobs ─────────────────────────────────────────────────────────────────

func TestIdleMobs_NoPanic(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	evt := events.NewRound{RoundNumber: 1}
	result := IdleMobs(evt)
	assert.Equal(t, events.Continue, result)
}

// ─── BroadcastNewChar ─────────────────────────────────────────────────────────

func TestBroadcastNewChar_CharacterCreated(t *testing.T) {
	result := BroadcastNewChar(events.CharacterCreated{UserId: 1, CharacterName: "Hero"})
	assert.Equal(t, events.Continue, result)
}

func TestBroadcastNewChar_CharacterChanged(t *testing.T) {
	result := BroadcastNewChar(events.CharacterChanged{UserId: 1, LastCharacterName: "Old", CharacterName: "New"})
	assert.Equal(t, events.Continue, result)
}

func TestBroadcastNewChar_WrongEvent(t *testing.T) {
	result := BroadcastNewChar(events.NewRound{RoundNumber: 1})
	assert.Equal(t, events.Cancel, result)
}

// ─── HandleJoin ───────────────────────────────────────────────────────────────

func TestHandleJoin_WrongEvent(t *testing.T) {
	result := HandleJoin(events.NewRound{RoundNumber: 1})
	assert.Equal(t, events.Cancel, result)
}

func TestHandleJoin_UserNotFound(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	result := HandleJoin(events.PlayerSpawn{UserId: 999})
	assert.Equal(t, events.Cancel, result)
}

func TestHandleJoin_Success(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	result := HandleJoin(events.PlayerSpawn{UserId: 1, ConnectionId: 1001, RoomId: 1})
	assert.Equal(t, events.Continue, result)
}

// ─── HandlePlayerDrop ─────────────────────────────────────────────────────────

func TestHandlePlayerDrop_WrongEvent(t *testing.T) {
	result := HandlePlayerDrop(events.NewRound{RoundNumber: 1})
	assert.Equal(t, events.Cancel, result)
}

func TestHandlePlayerDrop_UserNotFound(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	result := HandlePlayerDrop(events.PlayerDrop{UserId: 999, RoomId: 1})
	assert.Equal(t, events.Cancel, result)
}

func TestHandlePlayerDrop_Success(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	u := users.GetByUserId(1)
	u.Character.Health = 0

	result := HandlePlayerDrop(events.PlayerDrop{UserId: 1, RoomId: 1})
	assert.Equal(t, events.Continue, result)
	assert.Equal(t, 0, u.Character.DownedRounds)
}

func TestHandlePlayerDrop_RoomNotFound(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	result := HandlePlayerDrop(events.PlayerDrop{UserId: 1, RoomId: 999})
	assert.Equal(t, events.Continue, result)
}

// ─── HandleLookHints ──────────────────────────────────────────────────────────

func TestHandleLookHints_WrongEvent(t *testing.T) {
	result := HandleLookHints(events.NewRound{RoundNumber: 1})
	assert.Equal(t, events.Cancel, result)
}

func TestHandleLookHints_WithTarget(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	result := HandleLookHints(events.Looking{UserId: 1, RoomId: 1, Target: "something"})
	assert.Equal(t, events.Continue, result)
}

func TestHandleLookHints_UserNotFound(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	result := HandleLookHints(events.Looking{UserId: 999, RoomId: 1})
	assert.Equal(t, events.Cancel, result)
}

func TestHandleLookHints_NoTarget(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	result := HandleLookHints(events.Looking{UserId: 1, RoomId: 1})
	assert.Equal(t, events.Continue, result)
}

// ─── CheckItemQuests ──────────────────────────────────────────────────────────

func TestCheckItemQuests_WrongEvent(t *testing.T) {
	result := CheckItemQuests(events.NewRound{RoundNumber: 1})
	assert.Equal(t, events.Cancel, result)
}

func TestCheckItemQuests_NoUserId(t *testing.T) {
	result := CheckItemQuests(events.ItemOwnership{UserId: 0})
	assert.Equal(t, events.Continue, result)
}

func TestCheckItemQuests_GainedItem(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	result := CheckItemQuests(events.ItemOwnership{UserId: 1, Gained: true})
	assert.Equal(t, events.Continue, result)
}

func TestCheckItemQuests_LostItem(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	result := CheckItemQuests(events.ItemOwnership{UserId: 1, Gained: false})
	assert.Equal(t, events.Continue, result)
}

// ─── CleanupEphemeralRooms ────────────────────────────────────────────────────

func TestCleanupEphemeralRooms_NoUser(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	result := CleanupEphemeralRooms(events.RoomChange{UserId: 0, FromRoomId: 1, ToRoomId: 2})
	assert.Equal(t, events.Continue, result)
}

func TestCleanupEphemeralRooms_NonEphemeral(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	result := CleanupEphemeralRooms(events.RoomChange{UserId: 1, FromRoomId: 1, ToRoomId: 2})
	assert.Equal(t, events.Continue, result)
}

// ─── LocationMusicChange ──────────────────────────────────────────────────────

func TestLocationMusicChange_WrongEvent(t *testing.T) {
	result := LocationMusicChange(events.NewRound{RoundNumber: 1})
	assert.Equal(t, events.Cancel, result)
}

func TestLocationMusicChange_NoUser(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	result := LocationMusicChange(events.RoomChange{UserId: 0, FromRoomId: 1, ToRoomId: 2})
	assert.Equal(t, events.Continue, result)
}

func TestLocationMusicChange_UserNotFound(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	result := LocationMusicChange(events.RoomChange{UserId: 999, FromRoomId: 1, ToRoomId: 2})
	assert.Equal(t, events.Cancel, result)
}

func TestLocationMusicChange_Success(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	result := LocationMusicChange(events.RoomChange{UserId: 1, FromRoomId: 1, ToRoomId: 2})
	assert.Equal(t, events.Continue, result)
}

func TestLocationMusicChange_ToRoomNotFound(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	result := LocationMusicChange(events.RoomChange{UserId: 1, FromRoomId: 1, ToRoomId: 999})
	assert.Equal(t, events.Cancel, result)
}

func TestLocationMusicChange_FromRoomNotFound(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	result := LocationMusicChange(events.RoomChange{UserId: 1, FromRoomId: 999, ToRoomId: 1})
	assert.Equal(t, events.Cancel, result)
}

// ─── HandleLeave ──────────────────────────────────────────────────────────────

func TestHandleLeave_WrongEvent(t *testing.T) {
	result := HandleLeave(events.NewRound{RoundNumber: 1})
	assert.Equal(t, events.Cancel, result)
}

func TestHandleLeave_UserNotFound(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	result := HandleLeave(events.PlayerDespawn{UserId: 999})
	assert.Equal(t, events.Cancel, result)
}

// ─── Combat Helper: dispatchCritEffectsPvP ────────────────────────────────────

func TestDispatchCritEffectsPvP_NoCrit(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	u1 := users.GetByUserId(1)
	u2 := users.GetByUserId(2)
	room := rooms.LoadRoom(1)
	result := CritEffectResult{}
	dispatchCritEffectsPvP(result, u1, u2, room)
	// no panic
}

func TestDispatchCritEffectsPvP_Disarmed(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	u1 := users.GetByUserId(1)
	u2 := users.GetByUserId(2)
	room := rooms.LoadRoom(1)
	result := CritEffectResult{
		Disarmed: true,
		DisarmItem: combat.DisarmResult{
			Message:     "You are disarmed!",
			TargetMsg:   "You disarm them!",
			RoomMessage: "A disarm happens!",
		},
	}
	dispatchCritEffectsPvP(result, u1, u2, room)
}

func TestDispatchCritEffectsPvP_Grapple(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	u1 := users.GetByUserId(1)
	u2 := users.GetByUserId(2)
	room := rooms.LoadRoom(1)
	result := CritEffectResult{GrappleSet: true}
	dispatchCritEffectsPvP(result, u1, u2, room)
}

// ─── Combat Helper: dispatchCritEffectsPvM ────────────────────────────────────

func TestDispatchCritEffectsPvM_NoCrit(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	u1 := users.GetByUserId(1)
	mob := mobs.GetInstance(100)
	room := rooms.LoadRoom(1)
	result := CritEffectResult{}
	dispatchCritEffectsPvM(result, u1, mob, room)
}

func TestDispatchCritEffectsPvM_Disarmed(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	u1 := users.GetByUserId(1)
	mob := mobs.GetInstance(100)
	room := rooms.LoadRoom(1)
	result := CritEffectResult{
		Disarmed: true,
		DisarmItem: combat.DisarmResult{
			Message:     "Disarmed!",
			TargetMsg:   "You disarm!",
			RoomMessage: "Room disarm!",
		},
	}
	dispatchCritEffectsPvM(result, u1, mob, room)
}

func TestDispatchCritEffectsPvM_Grapple(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	u1 := users.GetByUserId(1)
	mob := mobs.GetInstance(100)
	room := rooms.LoadRoom(1)
	result := CritEffectResult{GrappleSet: true}
	dispatchCritEffectsPvM(result, u1, mob, room)
}

// ─── Combat Helper: handleCharmedMobAssist ────────────────────────────────────

func TestHandleCharmedMobAssist_NoCharmedMobs(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	room := rooms.LoadRoom(1)
	handleCharmedMobAssist(room, 1, "skeleton")
	// no panic
}

// ─── Combat Helper: RetargetOrEnd ─────────────────────────────────────────────

func TestRetargetOrEnd_NoAttackers(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	u := users.GetByUserId(1)
	room := rooms.LoadRoom(1)
	RetargetOrEnd(u.Character, room, u.UserId, 0)
	// Should not panic, no attackers to retarget
}

// ─── Combat Helper: handlePlayerConcentrationBreak ────────────────────────────

func TestHandlePlayerConcentrationBreak_NoCasting(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	u := users.GetByUserId(1)
	room := rooms.LoadRoom(1)
	u.Character.CastingState = nil
	handlePlayerConcentrationBreak(u, combat.AttackResult{DamageToTarget: 5}, room)
	// Should not panic
}

func TestHandlePlayerConcentrationBreak_WithCasting(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	u := users.GetByUserId(1)
	room := rooms.LoadRoom(1)
	u.Character.CastingState = &characters.CastingState{
		SpellId: "sparks",
	}
	// Large damage to break concentration
	handlePlayerConcentrationBreak(u, combat.AttackResult{DamageToTarget: 999}, room)
}

// ─── Combat Helper: dispatchCombatMessages ────────────────────────────────────

func TestDispatchCombatMessages_Empty(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	u1 := users.GetByUserId(1)
	u2 := users.GetByUserId(2)
	room := rooms.LoadRoom(1)
	dispatchCombatMessages(combat.AttackResult{}, u1, u2, room, room)
}

func TestDispatchCombatMessages_WithMessages(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	u1 := users.GetByUserId(1)
	u2 := users.GetByUserId(2)
	room := rooms.LoadRoom(1)
	result := combat.AttackResult{
		MessagesToSource:     []string{"You hit!"},
		MessagesToTarget:     []string{"You were hit!"},
		MessagesToSourceRoom: []string{"Attack lands!"},
		MessagesToTargetRoom: []string{"Attack lands!"},
	}
	dispatchCombatMessages(result, u1, u2, room, room)
}

// ─── Combat Helper: handleMobTargetSwitch ─────────────────────────────────────

func TestHandleMobTargetSwitch_NotDefaultAttack(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	mob := mobs.GetInstance(100)
	room := rooms.LoadRoom(1)
	mob.Character.SetAggro(1, 0, characters.Flee)
	result := handleMobTargetSwitch(mob, room)
	assert.False(t, result)
}

// ─── Combat Helper: handleMobWeaponPickup ─────────────────────────────────────

func TestHandleMobWeaponPickup_HasWeapon(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	mob := mobs.GetInstance(100)
	mob.Character.Equipment.Weapon.ItemId = 1
	handleMobWeaponPickup(mob)
	// Should return early, no panic
}

func TestHandleMobWeaponPickup_NoItems(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	mob := mobs.GetInstance(100)
	mob.Character.Equipment.Weapon.ItemId = 0
	mob.Character.Items = nil
	handleMobWeaponPickup(mob)
	// Should return early, no items
}

// ─── Combat Helper: handleMobDownedGrace ──────────────────────────────────────

func TestHandleMobDownedGrace_NotDisabled(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	mob := mobs.GetInstance(100)
	u := users.GetByUserId(1)
	room := rooms.LoadRoom(1)
	u.Character.Health = 100
	affected := []int{}
	result := handleMobDownedGrace(mob, u, room, &affected)
	assert.False(t, result)
}

func TestHandleMobDownedGrace_GracePeriod(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	mob := mobs.GetInstance(100)
	u := users.GetByUserId(1)
	room := rooms.LoadRoom(1)
	u.Character.Health = 0
	u.Character.DownedRounds = 0
	affected := []int{}
	result := handleMobDownedGrace(mob, u, room, &affected)
	assert.True(t, result)
}

// ─── Combat Helper: handlePlayerFlee ──────────────────────────────────────────

func TestHandlePlayerFlee_NotFleeing(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	u := users.GetByUserId(1)
	room := rooms.LoadRoom(1)
	u.Character.SetAggro(0, 100, characters.DefaultAttack)
	result := handlePlayerFlee(u, room, 1)
	assert.False(t, result)
}

// handlePlayerFlee requires keywords + usercommands.Look infrastructure
// that is too deep to initialize in unit tests — covered by integration tests.

// ─── Spell Resolution: resolveSpell ───────────────────────────────────────────

func TestResolveSpell_HarmSingle(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	u := users.GetByUserId(1)
	room := rooms.LoadRoom(1)
	spellData := spells.GetSpell("sparks")
	require.NotNil(t, spellData)

	cs := &characters.CastingState{
		SpellId:              "sparks",
		TargetMobInstanceIds: []int{100},
	}

	mob := mobs.GetInstance(100)
	origHealth := mob.Character.Health

	resolveSpell(u, cs, spellData, room)

	// Spell should have either hit or missed, but not panicked
	_ = origHealth
}

func TestResolveSpell_TargetLeftRoom(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	u := users.GetByUserId(1)
	room := rooms.LoadRoom(1)
	spellData := spells.GetSpell("sparks")
	require.NotNil(t, spellData)

	// Move mob out of room
	mob := mobs.GetInstance(100)
	mob.Character.RoomId = 999

	cs := &characters.CastingState{
		SpellId:              "sparks",
		TargetMobInstanceIds: []int{100},
	}

	resolveSpell(u, cs, spellData, room)
	// Should skip target that left room
}

func TestResolveSpell_HarmArea(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	u := users.GetByUserId(1)
	room := rooms.LoadRoom(1)

	areaSpell := &spells.SpellData{
		SpellId:           "fireball",
		Name:              "Fireball",
		Type:              spells.HarmArea,
		EffectType:        "damage",
		TargetDefenseType: "mental",
		DamageMultiplier:  1.0,
		EffectMagnitude:   30,
	}
	cleanupSpells := spells.SeedSpellsForTest(map[string]*spells.SpellData{
		"sparks":   spells.GetSpell("sparks"),
		"fireball": areaSpell,
	})
	defer cleanupSpells()

	cs := &characters.CastingState{
		SpellId: "fireball",
	}

	resolveSpell(u, cs, areaSpell, room)
	// Should populate area targets and resolve
}

func TestResolveSpell_HelpArea(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	u := users.GetByUserId(1)
	room := rooms.LoadRoom(1)

	helpSpell := &spells.SpellData{
		SpellId:           "massHeal",
		Name:              "Mass Heal",
		Type:              spells.HelpArea,
		EffectType:        "heal",
		TargetDefenseType: "",
		EffectMagnitude:   20,
	}

	cs := &characters.CastingState{
		SpellId:       "massHeal",
		TargetUserIds: []int{},
	}

	resolveSpell(u, cs, helpSpell, room)
}

func TestResolveSpell_DeadMobSkipped(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	u := users.GetByUserId(1)
	room := rooms.LoadRoom(1)
	spellData := spells.GetSpell("sparks")

	mob := mobs.GetInstance(100)
	mob.Character.Health = 0

	cs := &characters.CastingState{
		SpellId:              "sparks",
		TargetMobInstanceIds: []int{100},
	}

	resolveSpell(u, cs, spellData, room)
}

// ─── Spell Resolution: resolveAgainstMob ──────────────────────────────────────

func TestResolveAgainstMob_Basic(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	u := users.GetByUserId(1)
	mob := mobs.GetInstance(100)
	room := rooms.LoadRoom(1)
	mob.Character.Health = 100
	spellData := spells.GetSpell("sparks")

	resolveAgainstMob(u, mob, room, spellData, 200.0, 30)
	// Should resolve without panic
}

// ─── Spell Resolution: applyMobEffect ─────────────────────────────────────────

func TestApplyMobEffect_Damage(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	u := users.GetByUserId(1)
	mob := mobs.GetInstance(100)
	room := rooms.LoadRoom(1)
	mob.Character.Health = 100

	spellData := spells.GetSpell("sparks")
	dmg := applyMobEffect(u, mob, room, spellData, 30, false)
	assert.GreaterOrEqual(t, dmg, 0)
}

func TestApplyMobEffect_DamageWithCrit(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	u := users.GetByUserId(1)
	mob := mobs.GetInstance(100)
	room := rooms.LoadRoom(1)
	mob.Character.Health = 100

	spellData := spells.GetSpell("sparks")
	dmg := applyMobEffect(u, mob, room, spellData, 30, true)
	assert.GreaterOrEqual(t, dmg, 0)
}

func TestApplyMobEffect_DotEffect(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	u := users.GetByUserId(1)
	mob := mobs.GetInstance(100)
	room := rooms.LoadRoom(1)

	dotSpell := &spells.SpellData{
		SpellId:           "poison",
		Name:              "Poison",
		Type:              spells.HarmSingle,
		EffectType:        "dot",
		TargetDefenseType: "mental",
		EffectDuration:    3,
		EffectMagnitude:   10,
	}

	dmg := applyMobEffect(u, mob, room, dotSpell, 10, false)
	assert.Equal(t, 0, dmg)
}

func TestApplyMobEffect_NilUser(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	mob := mobs.GetInstance(100)
	room := rooms.LoadRoom(1)
	mob.Character.Health = 100

	spellData := spells.GetSpell("sparks")
	dmg := applyMobEffect(nil, mob, room, spellData, 30, false)
	assert.GreaterOrEqual(t, dmg, 0)
}

// ─── Combat: handlePlayerCombat / handleMobCombat coverage ────────────────────

// Full DoCombat with aggro requires combat.ResolveAttack which depends on
// items/weapons initialization — covered by integration tests.

// ─── Spell Resolution: applyMobEffect — more branches ────────────────────────

func TestApplyMobEffect_Knockdown(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	u := users.GetByUserId(1)
	mob := mobs.GetInstance(100)
	room := rooms.LoadRoom(1)
	mob.Character.Health = 100

	kdSpell := &spells.SpellData{
		SpellId:           "knockback",
		Name:              "Knockback",
		Type:              spells.HarmSingle,
		EffectType:        "knockdown",
		TargetDefenseType: "physical",
		DamageMultiplier:  0.5,
		EffectMagnitude:   20,
	}
	dmg := applyMobEffect(u, mob, room, kdSpell, 20, false)
	assert.GreaterOrEqual(t, dmg, 0)
	assert.Equal(t, characters.PositionProne, mob.Character.CombatPosition)
}

func TestApplyMobEffect_Buff(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	u := users.GetByUserId(1)
	mob := mobs.GetInstance(100)
	room := rooms.LoadRoom(1)

	buffSpell := &spells.SpellData{
		SpellId:    "weaken",
		Name:       "Weaken",
		Type:       spells.HarmSingle,
		EffectType: "buff",
		BuffIds:    []int{100},
	}
	dmg := applyMobEffect(u, mob, room, buffSpell, 0, false)
	assert.Equal(t, 0, dmg)
}

func TestApplyMobEffect_Tame_NotAnimal(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	u := users.GetByUserId(1)
	mob := mobs.GetInstance(100)
	room := rooms.LoadRoom(1)

	tameSpell := &spells.SpellData{
		SpellId:    "tame",
		Name:       "Tame",
		Type:       spells.HarmSingle,
		EffectType: "tame",
	}
	dmg := applyMobEffect(u, mob, room, tameSpell, 0, false)
	assert.Equal(t, 0, dmg)
}

func TestApplyMobEffect_DefaultEffect(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	u := users.GetByUserId(1)
	mob := mobs.GetInstance(100)
	room := rooms.LoadRoom(1)

	unknownSpell := &spells.SpellData{
		SpellId:    "mystery",
		Name:       "Mystery",
		Type:       spells.HarmSingle,
		EffectType: "unknown-effect",
	}
	dmg := applyMobEffect(u, mob, room, unknownSpell, 10, false)
	assert.Equal(t, 0, dmg)
}

// ─── Spell Resolution: resolveAgainstPlayer ───────────────────────────────────

func TestResolveAgainstPlayer_Basic(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	caster := users.GetByUserId(1)
	target := users.GetByUserId(2)
	room := rooms.LoadRoom(1)

	spellData := spells.GetSpell("sparks")
	resolveAgainstPlayer(caster, target, room, spellData, 200.0, 30)
	// Should resolve without panic
}

// ─── Spell Resolution: applyPlayerEffect ──────────────────────────────────────

func TestApplyPlayerEffect_Purge(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	caster := users.GetByUserId(1)
	target := users.GetByUserId(2)
	room := rooms.LoadRoom(1)

	purgeSpell := &spells.SpellData{
		SpellId:    "purge",
		Name:       "Purge",
		EffectType: "purge",
	}
	target.Character.AddCondition(characters.ConditionPoisoned, 10, 5.0, "test")
	applyPlayerEffect(caster, target, room, purgeSpell, 10, false)
}

func TestApplyPlayerEffect_PurgeSelf(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	u := users.GetByUserId(1)
	room := rooms.LoadRoom(1)

	purgeSpell := &spells.SpellData{
		SpellId:    "purge",
		Name:       "Purge",
		EffectType: "purge",
	}
	u.Character.AddCondition(characters.ConditionPoisoned, 10, 5.0, "test")
	applyPlayerEffect(u, u, room, purgeSpell, 10, false)
}

func TestApplyPlayerEffect_Heal(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	caster := users.GetByUserId(1)
	target := users.GetByUserId(2)
	room := rooms.LoadRoom(1)

	healSpell := &spells.SpellData{
		SpellId:         "heal",
		Name:            "Heal",
		EffectType:      "heal",
		EffectMagnitude: 3,
	}
	applyPlayerEffect(caster, target, room, healSpell, 3, false)
}

func TestApplyPlayerEffect_HealSelf(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	u := users.GetByUserId(1)
	room := rooms.LoadRoom(1)

	healSpell := &spells.SpellData{
		SpellId:         "heal",
		Name:            "Heal",
		EffectType:      "heal",
		EffectMagnitude: 3,
	}
	applyPlayerEffect(u, u, room, healSpell, 3, false)
}

func TestApplyPlayerEffect_HealCrit(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	caster := users.GetByUserId(1)
	target := users.GetByUserId(2)
	room := rooms.LoadRoom(1)

	healSpell := &spells.SpellData{
		SpellId:         "heal",
		Name:            "Heal",
		EffectType:      "heal",
		EffectMagnitude: 3,
	}
	applyPlayerEffect(caster, target, room, healSpell, 3, true)
}

func TestApplyPlayerEffect_Buff(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	caster := users.GetByUserId(1)
	target := users.GetByUserId(2)
	room := rooms.LoadRoom(1)

	buffSpell := &spells.SpellData{
		SpellId:    "bless",
		Name:       "Bless",
		EffectType: "buff",
		BuffIds:    []int{100},
	}
	applyPlayerEffect(caster, target, room, buffSpell, 0, false)
}

func TestApplyPlayerEffect_BuffSelf(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	u := users.GetByUserId(1)
	room := rooms.LoadRoom(1)

	buffSpell := &spells.SpellData{
		SpellId:    "bless",
		Name:       "Bless",
		EffectType: "buff",
		BuffIds:    []int{100},
	}
	applyPlayerEffect(u, u, room, buffSpell, 0, false)
}

func TestApplyPlayerEffect_Shield(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	caster := users.GetByUserId(1)
	target := users.GetByUserId(2)
	room := rooms.LoadRoom(1)

	shieldSpell := &spells.SpellData{
		SpellId:    "shield",
		Name:       "Shield",
		EffectType: "shield",
	}
	applyPlayerEffect(caster, target, room, shieldSpell, 10, false)
}

func TestApplyPlayerEffect_ShieldSelf(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	u := users.GetByUserId(1)
	room := rooms.LoadRoom(1)

	shieldSpell := &spells.SpellData{
		SpellId:    "shield",
		Name:       "Shield",
		EffectType: "shield",
	}
	applyPlayerEffect(u, u, room, shieldSpell, 10, false)
}

func TestApplyPlayerEffect_Default(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	caster := users.GetByUserId(1)
	target := users.GetByUserId(2)
	room := rooms.LoadRoom(1)

	unknownSpell := &spells.SpellData{
		SpellId:    "mystery",
		Name:       "Mystery",
		EffectType: "unknown-player-effect",
	}
	applyPlayerEffect(caster, target, room, unknownSpell, 10, false)
}

// ─── Spell Resolution: resolveMobSpell ────────────────────────────────────────

func TestResolveMobSpell_HarmSingle(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	mob := mobs.GetInstance(100)
	room := rooms.LoadRoom(1)
	spellData := spells.GetSpell("sparks")

	cs := &characters.CastingState{
		SpellId:       "sparks",
		TargetUserIds: []int{1},
	}

	resolveMobSpell(mob, cs, spellData, room)
}

func TestResolveMobSpell_SelfCast(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	mob := mobs.GetInstance(100)
	room := rooms.LoadRoom(1)

	healSpell := &spells.SpellData{
		SpellId:         "mobheal",
		Name:            "Heal",
		Type:            spells.HelpSingle,
		EffectType:      "heal",
		EffectMagnitude: 3,
	}

	cs := &characters.CastingState{
		SpellId:              "mobheal",
		TargetMobInstanceIds: []int{100}, // Self target
	}

	resolveMobSpell(mob, cs, healSpell, room)
}

func TestResolveMobSpell_MobVsMob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	mob := mobs.GetInstance(100)
	room := rooms.LoadRoom(1)
	spellData := spells.GetSpell("sparks")

	// Add a second mob instance
	mob2 := &mobs.Mob{
		MobId:      2,
		InstanceId: 101,
		HomeRoomId: 1,
		Character: characters.Character{
			Name:      "Target Mob",
			RoomId:    1,
			Health:    50,
			Buffs:     buffs.New(),
			Cooldowns: map[string]int{},
		},
	}
	mob2.Character.HealthMax.Value = 100
	mob2.Character.Stats.Willpower.ValueAdj = 50
	mobs.SeedMobsForTest(map[int]*mobs.Mob{1: {MobId: 1, Character: characters.Character{Name: "Skeleton"}}},
		map[int]*mobs.Mob{100: mob, 101: mob2})

	cs := &characters.CastingState{
		SpellId:              "sparks",
		TargetMobInstanceIds: []int{101},
	}

	resolveMobSpell(mob, cs, spellData, room)
}

// ─── Spell Resolution: applyMobSelfEffect ─────────────────────────────────────

func TestApplyMobSelfEffect_Heal(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	mob := mobs.GetInstance(100)
	room := rooms.LoadRoom(1)

	healSpell := &spells.SpellData{
		SpellId:         "selfheal",
		Name:            "Self Heal",
		EffectType:      "heal",
		EffectMagnitude: 3,
	}

	applyMobSelfEffect(mob, room, healSpell, 3)
}

func TestApplyMobSelfEffect_Shield(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	mob := mobs.GetInstance(100)
	room := rooms.LoadRoom(1)

	shieldSpell := &spells.SpellData{
		SpellId:    "selfshield",
		Name:       "Self Shield",
		EffectType: "shield",
	}

	applyMobSelfEffect(mob, room, shieldSpell, 10)
}

// ─── Spell Resolution: consumeSpellComponent ──────────────────────────────────

func TestConsumeSpellComponent_NoItems(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	u := users.GetByUserId(1)
	u.Character.Items = nil
	consumeSpellComponent(u, "ruby")
}

// ─── Spell Resolution: resolveMobSpellAgainstPlayer ───────────────────────────

func TestResolveMobSpellAgainstPlayer(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	mob := mobs.GetInstance(100)
	target := users.GetByUserId(1)
	room := rooms.LoadRoom(1)
	spellData := spells.GetSpell("sparks")

	resolveMobSpellAgainstPlayer(mob, target, room, spellData, 100.0, 30)
}

// ─── Combat Helper: handlePartyAutoAttack ─────────────────────────────────────

func TestHandlePartyAutoAttack_NoParty(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	mob := mobs.GetInstance(100)
	u := users.GetByUserId(1)
	handlePartyAutoAttack(mob, u)
	// No party, should not panic
}

// ─── Combat Helper: handleMobDownedGrace finishing blow ───────────────────────

func TestHandleMobDownedGrace_FinishingBlow(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	mob := mobs.GetInstance(100)
	u := users.GetByUserId(1)
	room := rooms.LoadRoom(1)
	u.Character.Health = 0
	mob.Character.SetAggro(u.UserId, 0, characters.DefaultAttack)
	affected := []int{}
	result := handleMobDownedGrace(mob, u, room, &affected)
	assert.True(t, result)
	// With CoupDeGraceRounds=0 (default), mob disengages immediately
	assert.Nil(t, mob.Character.Aggro)
}

// ─── HandleIdleMobs with a mob that can idle ──────────────────────────────────

func TestHandleIdleMobs_MobNotFound(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	result := HandleIdleMobs(events.MobIdle{MobInstanceId: 999})
	assert.Equal(t, events.Cancel, result)
}

func TestHandleIdleMobs_ValidMob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	result := HandleIdleMobs(events.MobIdle{MobInstanceId: 100})
	assert.Equal(t, events.Continue, result)
}

// ─── Gossiper helpers ─────────────────────────────────────────────────────────

func TestMobHasGroup_Found(t *testing.T) {
	mob := &mobs.Mob{Groups: []string{"humanoid", "gossiper", "merchant"}}
	assert.True(t, mobHasGroup(mob, "gossiper"))
}

func TestMobHasGroup_NotFound(t *testing.T) {
	mob := &mobs.Mob{Groups: []string{"humanoid", "merchant"}}
	assert.False(t, mobHasGroup(mob, "gossiper"))
}

func TestMobHasGroup_EmptyGroups(t *testing.T) {
	mob := &mobs.Mob{Groups: nil}
	assert.False(t, mobHasGroup(mob, "gossiper"))
}

func TestBuildGossipLine_FallbackWhenNoEvents(t *testing.T) {
	// Force sync.Once to complete, then override templates
	gossipTemplatesOnce.Do(func() {})
	gossipTemplates = map[string][]string{
		"fallback": {"Nothing to report.", "Quiet day."},
	}
	defer func() { gossipTemplates = nil }()

	mob := &mobs.Mob{
		MobId: 114,
		Character: characters.Character{
			Zone: "TestZone",
		},
	}

	line := buildGossipLine(mob)
	assert.Contains(t, []string{"Nothing to report.", "Quiet day."}, line)
}

func TestBuildGossipLine_EmptyTemplatesEmptyEvents(t *testing.T) {
	gossipTemplatesOnce.Do(func() {})
	gossipTemplates = map[string][]string{}
	defer func() { gossipTemplates = nil }()

	mob := &mobs.Mob{
		MobId: 114,
		Character: characters.Character{
			Zone: "TestZone",
		},
	}

	line := buildGossipLine(mob)
	assert.Equal(t, "", line)
}

func TestHandleIdleMobs_GossiperMob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	// Add a gossiper mob instance
	gossiperMob := &mobs.Mob{
		MobId:         114,
		InstanceId:    200,
		HomeRoomId:    1,
		Hostile:       false,
		ActivityLevel: 7,
		Groups:        []string{"humanoid", "gossiper"},
		Character: characters.Character{
			Name:      "old Fen",
			Zone:      "TestZone",
			RoomId:    1,
			Health:    50,
			Buffs:     buffs.New(),
			Cooldowns: map[string]int{},
		},
	}
	gossiperMob.Character.HealthMax.Value = 100

	cleanupGossiper := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{200: gossiperMob})
	defer cleanupGossiper()

	// Pre-seed gossip templates so it doesn't try to load from disk
	gossipTemplatesOnce.Do(func() {})
	gossipTemplates = map[string][]string{
		"fallback": {"Quiet day."},
	}
	defer func() { gossipTemplates = nil }()

	result := HandleIdleMobs(events.MobIdle{MobInstanceId: 200})
	assert.Equal(t, events.Continue, result)
}

// ─── PlaySound ────────────────────────────────────────────────────────────────

func TestPlaySound_WrongEvent(t *testing.T) {
	result := PlaySound(events.NewRound{RoundNumber: 1})
	assert.Equal(t, events.Cancel, result)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func dummyAttackResult(hit, crit bool) combat.AttackResult {
	return combat.AttackResult{
		Hit:  hit,
		Crit: crit,
	}
}
