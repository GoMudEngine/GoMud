package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

// fireRangedWeapon builds a loaded-by-default shooting weapon with a known
// damage multiplier so damage-band assertions are deterministic.
func fireRangedWeapon(id int, mult float64, loaded bool) items.Item {
	return items.Item{
		ItemId: id,
		Loaded: loaded,
		Spec: &items.ItemSpec{
			ItemId:           id,
			Name:             "longbow",
			Type:             items.Weapon,
			Subtype:          items.Shooting,
			AmmoTag:          "arrows",
			DamageMultiplier: mult,
			Hands:            2,
		},
	}
}

// fireAttacker builds the shooter: Perception-dominant, Strength-starved, so
// any successful hit / damage proves Perception (not Strength) governs the
// ranged channel.
func fireAttacker() *characters.Character {
	c := characters.New()
	c.Stats.Perception.ValueAdj = 300
	c.Stats.Strength.ValueAdj = 1
	return c
}

// seedFireMobInRoom seeds rooms 1 (exit north→2) and 2, plus a single mob
// instance (500, "Skeleton") in defenderRoomId with the given Dexterity.
// Returns the instance id and a cleanup. A fresh call gives a fresh
// defender — important because ExecuteSkillMove applies real HP loss.
func seedFireMobInRoom(t *testing.T, defenderRoomId int, defenderDex int) (int, func()) {
	t.Helper()

	mobSpecs := map[int]*mobs.Mob{
		1: {MobId: 1, Zone: "TestZone", Character: characters.Character{Name: "Skeleton"}},
	}

	var defChar characters.Character
	defChar.Name = "Skeleton"
	defChar.RoomId = defenderRoomId
	defChar.Health = 100000
	defChar.Buffs = buffs.New()
	defChar.Cooldowns = map[string]int{}
	defChar.Stats.Dexterity.ValueAdj = defenderDex

	mobInstances := map[int]*mobs.Mob{
		500: {MobId: 1, InstanceId: 500, HomeRoomId: defenderRoomId, Character: defChar},
	}
	cleanupMobs := mobs.SeedMobsForTest(mobSpecs, mobInstances)

	room1 := &rooms.Room{
		RoomId: 1, Zone: "TestZone", Title: "Room One", Biome: "city",
		Exits: map[string]exit.RoomExit{"north": {RoomId: 2}},
	}
	room2 := &rooms.Room{RoomId: 2, Zone: "TestZone", Title: "Room Two", Biome: "city"}
	cleanupRooms := rooms.SeedRoomsForTest(
		map[int]*rooms.Room{1: room1, 2: room2},
		map[string]*rooms.ZoneConfig{
			"TestZone": {Name: "TestZone", RoomId: 1, RoomIds: map[int]struct{}{1: {}, 2: {}}},
		},
	)

	if defenderRoomId == 1 {
		room1.AddMob(500)
		rooms.MarkRoomOccupancy(1, 0, 1)
	} else {
		room2.AddMob(500)
		rooms.MarkRoomOccupancy(2, 0, 1)
	}

	return 500, func() {
		cleanupRooms()
		cleanupMobs()
	}
}

// ---------------------------------------------------------------------------
// 1. No ranged weapon → NoWeapon
// ---------------------------------------------------------------------------

func TestFire_NoRangedWeapon(t *testing.T) {
	char := fireAttacker()
	char.Equipment.Weapon = reloadMeleeWeapon(2)
	actor := newStubActor(char, newTestRoom())

	res := ExecuteFire(actor, "skeleton")

	assert.True(t, res.NoWeapon, "expected NoWeapon with only a melee weapon")
	assert.False(t, res.Executed)
}

// ---------------------------------------------------------------------------
// 2. Unloaded ranged weapon → NotLoaded (WeaponName set)
// ---------------------------------------------------------------------------

func TestFire_Unloaded(t *testing.T) {
	char := fireAttacker()
	char.Equipment.Weapon = fireRangedWeapon(1, 1.0, false)
	actor := newStubActor(char, newTestRoom())

	res := ExecuteFire(actor, "skeleton")

	assert.True(t, res.NotLoaded, "expected NotLoaded for an unloaded ranged weapon")
	assert.NotEmpty(t, res.WeaponName, "NotLoaded result should name the weapon")
	assert.False(t, res.Executed)
}

// ---------------------------------------------------------------------------
// 3. Loaded + same-room mob target → Executed, unloads, Perception governs
// ---------------------------------------------------------------------------

func TestFire_SameRoomMob_PerceptionGoverns(t *testing.T) {
	const runs = 8
	for i := 0; i < runs; i++ {
		// Fresh defender each run — ExecuteSkillMove mutates HP.
		_, cleanup := seedFireMobInRoom(t, 1, 1)

		char := fireAttacker()
		char.Equipment.Weapon = fireRangedWeapon(1, 1.0, true)
		actor := newStubActor(char, rooms.LoadRoom(1))

		res := ExecuteFire(actor, "skeleton")

		require.True(t, res.Executed, "run %d: expected Executed", i)
		assert.True(t, res.IsTargetMob, "run %d: target should be a mob", i)
		assert.Equal(t, 500, res.TargetMobInstanceId, "run %d", i)
		assert.False(t, res.CrossRoom, "run %d: same-room shot", i)
		// Writeback: the equipment slot itself is now unloaded.
		assert.False(t, char.Equipment.Weapon.Loaded,
			"run %d: weapon must be unloaded after firing", i)

		// Perception 300 attacker vs Dex 1 defender → always a hit.
		assert.True(t, res.MoveResult.Hit, "run %d: Perception attacker should always hit", i)
		// Damage drove off Perception (300), not the Strength floor (1).
		assert.Greater(t, res.MoveResult.Damage, 1,
			"run %d: damage should reflect Perception, not collapse to the Str floor", i)

		cleanup()
	}
}

// ---------------------------------------------------------------------------
// 4. rangedDefenseScore: shield adds exactly RangedShieldDefenseBonus
// ---------------------------------------------------------------------------

func TestRangedDefenseScore_ShieldBonus(t *testing.T) {
	base := characters.New()
	base.Stats.Dexterity.ValueAdj = 50

	withShield := characters.New()
	withShield.Stats.Dexterity.ValueAdj = 50
	withShield.Equipment.Offhand = items.Item{
		ItemId: 99,
		Spec: &items.ItemSpec{
			ItemId:      99,
			Name:        "buckler",
			Type:        items.Offhand,
			BlockRating: 10,
		},
	}

	baseScore := rangedDefenseScore(base)
	shieldScore := rangedDefenseScore(withShield)

	bonus := float64(configs.GetBalanceConfig().RangedShieldDefenseBonus)
	assert.Equal(t, baseScore+bonus, shieldScore,
		"a shield should add exactly RangedShieldDefenseBonus to the ranged defense score")
}

// ---------------------------------------------------------------------------
// 5. Cross-room: loaded + valid exit + adjacent-room target → CrossRoom
// ---------------------------------------------------------------------------

func TestFire_CrossRoomMob(t *testing.T) {
	_, cleanup := seedFireMobInRoom(t, 2, 1) // mob lives in room 2
	defer cleanup()

	char := fireAttacker()
	char.Equipment.Weapon = fireRangedWeapon(1, 1.0, true)
	actor := newStubActor(char, rooms.LoadRoom(1))

	res := ExecuteFire(actor, "skeleton north")

	require.True(t, res.Executed, "expected a cross-room shot to execute")
	assert.True(t, res.CrossRoom, "expected CrossRoom true")
	assert.Equal(t, "north", res.ExitName, "expected the exit name carried through")
	assert.Equal(t, 2, res.TargetRoomId, "target should be resolved in room 2")
	assert.Equal(t, 500, res.TargetMobInstanceId)
	assert.False(t, char.Equipment.Weapon.Loaded, "weapon must be unloaded after firing")
}

// ---------------------------------------------------------------------------
// 6. Fire does NOT consume the special-move cooldown
// ---------------------------------------------------------------------------

func TestFire_DoesNotConsumeSpecialMoveCooldown(t *testing.T) {
	_, cleanup := seedFireMobInRoom(t, 1, 1)
	defer cleanup()

	char := fireAttacker()
	char.Equipment.Weapon = fireRangedWeapon(1, 1.0, true)
	actor := newStubActor(char, rooms.LoadRoom(1))

	res := ExecuteFire(actor, "skeleton")
	require.True(t, res.Executed)

	// Reload owns the cooldown; firing must leave it free.
	assert.True(t, char.Cooldowns.Try("special-move", "5 rounds"),
		"firing must NOT consume the special-move cooldown")
}

// ---------------------------------------------------------------------------
// 7. Charmed mob target → IsCharmed, not executed, weapon stays loaded
// ---------------------------------------------------------------------------

func TestFire_CharmedMob_NotExecuted(t *testing.T) {
	_, cleanup := seedFireMobInRoom(t, 1, 1)
	defer cleanup()

	// stubActor charms by key 0 (GetUserId()==0, GetMobInstanceId()==0).
	mobs.GetInstance(500).Character.Charm(0, 10, "")

	char := fireAttacker()
	char.Equipment.Weapon = fireRangedWeapon(1, 1.0, true)
	actor := newStubActor(char, rooms.LoadRoom(1))

	res := ExecuteFire(actor, "skeleton")

	assert.True(t, res.IsCharmed, "expected IsCharmed for a charmed mob")
	assert.False(t, res.Executed, "must not fire on a charmed mob")
	assert.True(t, char.Equipment.Weapon.Loaded, "weapon must stay loaded — no shot wasted")
}

// ---------------------------------------------------------------------------
// 8. Non-combatant mob target → IsNonCombatant, weapon stays loaded
// ---------------------------------------------------------------------------

func TestFire_NonCombatantMob_NotExecuted(t *testing.T) {
	_, cleanup := seedFireMobInRoom(t, 1, 1)
	defer cleanup()

	mobs.GetInstance(500).Character.NonCombatant = true

	char := fireAttacker()
	char.Equipment.Weapon = fireRangedWeapon(1, 1.0, true)
	actor := newStubActor(char, rooms.LoadRoom(1))

	res := ExecuteFire(actor, "skeleton")

	assert.True(t, res.IsNonCombatant, "expected IsNonCombatant for a non_combatant mob")
	assert.False(t, res.Executed, "must not fire on a non-combatant mob")
	assert.True(t, char.Equipment.Weapon.Loaded, "weapon must stay loaded — no shot wasted")
}

// ---------------------------------------------------------------------------
// Balance pin: arbalest baseline raw damage band
// ---------------------------------------------------------------------------

// Spec balance target: arbalest (mult 7.0) at stat 100 / rank 0 must produce
// raw damage in the 180-220 band BEFORE mitigation:
// 100 × SkillMultiplier(0)=1.0 × 7.0 × ChannelScale(0.30) = 210.
func TestRangedShotRawDamage_BalanceBand(t *testing.T) {
	raw := combat.CalcRawDamage(100, 0, 7.0, combat.ChannelPhysical)
	if raw < 180 || raw > 220 {
		t.Errorf("arbalest baseline raw %v outside 180-220 spec band", raw)
	}
}
