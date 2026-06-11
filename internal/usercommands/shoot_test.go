package usercommands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/crimes"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/opinions"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// equipBow puts a (optionally loaded) longbow in the user's weapon slot. The
// inline Spec mirrors the actions package's fire test fixture.
func equipBow(c *characters.Character, loaded bool) {
	c.Equipment.Weapon = items.Item{
		ItemId: 70001,
		Loaded: loaded,
		Spec: &items.ItemSpec{
			ItemId:           70001,
			Name:             "longbow",
			Type:             items.Weapon,
			Subtype:          items.Shooting,
			AmmoTag:          "arrows",
			DamageMultiplier: 1.0,
			Hands:            2,
		},
	}
}

// isolateOpinions points the opinion store at a temp dir so test shots that
// bump disposition don't touch real data.
func isolateOpinions(t *testing.T) {
	t.Helper()
	t.Setenv("DOGMUD_OPINIONS_DIR_OVERRIDE", t.TempDir())
	opinions.ClearCache()
}

// TestShoot_UnloadedWeapon_NoDamage: an unloaded bow fires nothing — the mob's
// HP is untouched.
func TestShoot_UnloadedWeapon_NoDamage(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	isolateOpinions(t)

	user, room := getTestUserAndRoom(t)
	user.Character.Stats.Perception.ValueAdj = 300
	equipBow(user.Character, false) // unloaded

	mob := mobs.GetInstance(100)
	require.NotNil(t, mob)
	mob.Character.Health = 50

	handled, err := Shoot("skeleton", user, room, 0)
	assert.True(t, handled)
	assert.NoError(t, err)

	assert.Equal(t, 50, mob.Character.Health, "unloaded bow must deal no damage")
	assert.Nil(t, user.Character.Aggro, "an unfired shot must not set shooter aggro")
}

// TestShoot_SameRoomLoaded_DamageAndAggro: a loaded same-room shot drops the
// mob's HP, unloads the weapon, sets shooter aggro on the mob, and sets the
// mob's aggro back on the shooter.
func TestShoot_SameRoomLoaded_DamageAndAggro(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	isolateOpinions(t)

	user, room := getTestUserAndRoom(t)
	user.Character.Stats.Perception.ValueAdj = 300
	user.Character.Stats.Strength.ValueAdj = 1
	equipBow(user.Character, true)

	mob := mobs.GetInstance(100)
	require.NotNil(t, mob)
	mob.Character.Health = 100000
	mob.Character.HealthMax.Value = 100000
	mob.Character.Aggro = nil
	user.Character.Aggro = nil

	handled, err := Shoot("skeleton", user, room, 0)
	assert.True(t, handled)
	assert.NoError(t, err)

	assert.Less(t, mob.Character.Health, 100000, "a loaded same-room shot must damage the mob")
	assert.False(t, user.Character.Equipment.Weapon.Loaded, "firing must unload the weapon")

	require.NotNil(t, user.Character.Aggro, "same-room shot must set shooter aggro")
	assert.Equal(t, 100, user.Character.Aggro.MobInstanceId)

	require.NotNil(t, mob.Character.Aggro, "mob must retaliate with aggro on the shooter")
	assert.Equal(t, user.UserId, mob.Character.Aggro.UserId)
}

// TestShoot_CrossRoomLoaded_NoShooterAggro_MobPursues: a loaded cross-room shot
// damages the adjacent-room mob, leaves the shooter WITHOUT aggro (one-shot
// model), and writes the CombatMemory breadcrumb that drives revenge pursuit.
func TestShoot_CrossRoomLoaded_NoShooterAggro_MobPursues(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	isolateOpinions(t)

	user, room := getTestUserAndRoom(t)
	user.Character.Stats.Perception.ValueAdj = 300
	user.Character.Stats.Strength.ValueAdj = 1
	user.Character.Aggro = nil
	equipBow(user.Character, true)

	// A fresh mob in the adjacent room (room 2).
	target := &mobs.Mob{
		MobId:      1,
		InstanceId: 400,
		HomeRoomId: 2,
		Character: characters.Character{
			Name:      "Skeleton",
			RoomId:    2,
			Health:    100000,
			Buffs:     buffs.New(),
			Cooldowns: map[string]int{},
		},
	}
	target.Character.HealthMax.Value = 100000
	target.Character.Stats.Dexterity.ValueAdj = 80
	mobs.SetInstanceForTest(400, target)
	defer mobs.SetInstanceForTest(400, nil)
	room2 := rooms.LoadRoom(2)
	require.NotNil(t, room2)
	room2.AddMob(400)
	defer room2.RemoveMob(400)

	handled, err := Shoot("skeleton north", user, room, 0)
	assert.True(t, handled)
	assert.NoError(t, err)

	assert.Less(t, target.Character.Health, 100000, "cross-room shot must damage the adjacent-room mob")
	assert.Nil(t, user.Character.Aggro, "cross-room shot must NOT set shooter aggro (one-shot model)")

	require.NotNil(t, target.CombatMemory, "cross-room shot must seed CombatMemory for revenge pursuit")
	assert.Equal(t, user.UserId, target.CombatMemory.TargetUserId, "memory must target the shooter")
	assert.Equal(t, user.Character.RoomId, target.CombatMemory.LastSeenRoomId, "memory must point at the shooter's room")
	assert.True(t, target.CombatMemory.Grudge, "pursuit memory must carry a grudge")
}

// TestShoot_RecordsAssaultCrimeOnFactionMob: shooting a faction-aligned mob
// records an assault crime attributing the shooter as the perpetrator —
// mirroring melee's recordAssaultCrime semantics.
func TestShoot_RecordsAssaultCrimeOnFactionMob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	dir := t.TempDir()
	t.Setenv("DOGMUD_FACTIONS_DIR_OVERRIDE", dir)
	t.Setenv("DOGMUD_FACTIONS_REP_DIR_OVERRIDE", t.TempDir())
	t.Setenv("DOGMUD_FACTIONS_CRIMES_DIR_OVERRIDE", t.TempDir())
	t.Setenv("DOGMUD_OPINIONS_DIR_OVERRIDE", t.TempDir())
	if err := os.WriteFile(filepath.Join(dir, "thornwall_citizens.yaml"),
		[]byte(`faction_id: thornwall_citizens
display_name: "Citizens"
description: "x"
default_rep: 0
allies: []
enemies: []
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := factions.LoadAllDefinitions(); err != nil {
		t.Fatal(err)
	}
	factions.ClearCache()
	crimes.ClearCache()
	opinions.ClearCache()

	user, room := getTestUserAndRoom(t)
	user.Character.Stats.Perception.ValueAdj = 300
	equipBow(user.Character, true)

	target := &mobs.Mob{
		MobId:      100,
		InstanceId: 410,
		HomeRoomId: 1,
		Groups:     []string{"thornwall_citizens"},
		Character: characters.Character{
			Name:      "city beggar",
			RoomId:    1,
			Health:    100000,
			Buffs:     buffs.New(),
			Cooldowns: map[string]int{},
		},
	}
	target.Character.HealthMax.Value = 100000
	mobs.SetInstanceForTest(410, target)
	defer mobs.SetInstanceForTest(410, nil)
	room.AddMob(410)
	defer room.RemoveMob(410)

	if _, err := Shoot("beggar", user, room, 0); err != nil {
		t.Fatalf("Shoot: %v", err)
	}

	got := crimes.AllForFaction("thornwall_citizens", false)
	require.Len(t, got, 1, "expected exactly one assault crime")
	assert.Equal(t, crimes.KindAssault, got[0].Kind)
	assert.Equal(t, crimes.PerpPlayer, got[0].Perpetrator.Type)
	assert.Equal(t, user.UserId, got[0].Perpetrator.Id)
}

// TestShoot_NoWeapon_Message: with no ranged weapon equipped, the command
// reports the missing weapon and fires nothing.
func TestShoot_NoWeapon(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	user, room := getTestUserAndRoom(t)
	user.Character.Equipment.Weapon = items.Item{} // no weapon

	handled, err := Shoot("skeleton", user, room, 0)
	assert.True(t, handled)
	assert.NoError(t, err)
	assert.Nil(t, user.Character.Aggro)
}
