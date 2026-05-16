package mobcommands

import (
	"os"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/language"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/awareness"
	"github.com/GoMudEngine/GoMud/internal/state/life"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	lang "golang.org/x/text/language"
)

// ─── Test Infrastructure ──────────────────────────────────────────────────────

// setCombatPositionParallel writes BOTH the legacy CombatPosition
// field AND the new Position FSM in lockstep. F1 fixture helper
// for the chunk-4b transition window. Synthetic Partner ref for
// Clinched/Grounded (FSM requires non-zero).
func setCombatPositionParallel(c *characters.Character, pos characters.CombatPosition) {
	c.CombatPosition = pos
	if c.Position == nil {
		c.Position = position.NewMachine()
	}
	r := state.TransitionReason{Trigger: "test_setup"}
	switch pos {
	case characters.PositionStanding:
		c.Position.ForceStanding(r)
	case characters.PositionProne:
		c.Position.ForceStanding(r)
		_ = c.Position.TransitionToProne(position.ProneData{}, r)
	case characters.PositionClinched:
		c.Position.ForceStanding(r)
		_ = c.Position.TransitionToClinch(
			position.GrappleData{Partner: state.ActorRef{UserId: 1}},
			state.TransitionReason{Trigger: position.TriggerGrappleEntry},
		)
	case characters.PositionGrounded:
		c.Position.ForceStanding(r)
		_ = c.Position.TransitionToClinch(
			position.GrappleData{Partner: state.ActorRef{UserId: 1}},
			state.TransitionReason{Trigger: position.TriggerGrappleEntry},
		)
		_ = c.Position.TransitionToMount(
			position.GrappleData{Partner: state.ActorRef{UserId: 1}, ControlLevel: position.InControl},
			state.TransitionReason{Trigger: position.TriggerTakedownMount},
		)
	}
}

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "", "", false)
	language.InitTranslation(language.BundleCfg{
		DefaultLanguage: lang.English,
		Language:        lang.English,
	})
	os.Exit(m.Run())
}

func seedAllRegistries() func() {
	cleanupKeywords := keywords.SeedKeywordsForTest()

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
			AutoAggro: true,
			ActivityLevel: 50,
			Groups:        []string{"undead"},
			Character: characters.Character{
				Name: "Skeleton",
			},
		},
		2: {
			MobId:         2,
			Zone:          "TestZone",
			AutoAggro: false,
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
			AutoAggro: true,
			Groups:     []string{"undead"},
			Character: characters.Character{
				Name:          "Skeleton",
				RoomId:        1,
				Health:        50,
				Buffs:         buffs.New(),
				Cooldowns:     map[string]int{},
				Awareness:     awareness.NewMachine(),
				Life:          life.NewMachine(),
				MobInstanceId: 100,
				IsMob:         true,
			},
		},
		200: {
			MobId:      2,
			InstanceId: 200,
			HomeRoomId: 1,
			AutoAggro: false,
			Character: characters.Character{
				Name:          "Merchant",
				RoomId:        1,
				Health:        100,
				Buffs:         buffs.New(),
				Cooldowns:     map[string]int{},
				Awareness:     awareness.NewMachine(),
				Life:          life.NewMachine(),
				MobInstanceId: 200,
				IsMob:         true,
			},
		},
	}
	testMobInstances[100].Character.HealthMax.Value = 100
	testMobInstances[100].Character.StaminaMax.Value = 50
	testMobInstances[100].Character.Stamina = 50
	testMobInstances[100].Character.ConvictionMax.Value = 30
	testMobInstances[100].Character.Conviction = 30
	testMobInstances[100].Character.Stats.Strength.ValueAdj = 80
	testMobInstances[100].Character.Stats.Dexterity.ValueAdj = 80
	testMobInstances[100].Character.Stats.Willpower.ValueAdj = 60
	testMobInstances[100].Character.SpeciesId = 1
	testMobInstances[200].Character.HealthMax.Value = 100
	testMobInstances[200].Character.StaminaMax.Value = 50
	testMobInstances[200].Character.Stamina = 50
	testMobInstances[200].Character.Stats.Strength.ValueAdj = 60
	testMobInstances[200].Character.Stats.Dexterity.ValueAdj = 60
	testMobInstances[200].Character.SpeciesId = 1

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
	room1.AddPlayer(1)
	room1.AddPlayer(2)
	room1.AddMob(100)
	room1.AddMob(200)
	rooms.MarkRoomOccupancy(1, 2, 2)

	cleanupBiomes := rooms.SeedBiomesForTest(map[string]*rooms.BiomeInfo{
		"city": {
			BiomeId:      "city",
			Name:         "City",
			Symbol:       "#",
			LitArea:      true,
			MovementCost: 1.0,
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
			SpellId:    "sparks",
			Name:       "Sparks",
			Type:       spells.HarmSingle,
			Cost:       3,
			Difficulty: 10,
		},
	})

	cleanupSpecies := species.SeedSpeciesForTest(map[int]*species.Species{
		1: {
			SpeciesId: 1,
			Name:      "Human",
		},
	})

	cleanupItems := items.SeedItemsForTest(map[int]*items.ItemSpec{
		1: {
			ItemId: 1,
			Name:   "Short Sword",
			Type:   items.Weapon,
			Damage: items.Damage{
				DiceRoll: "1d6",
			},
		},
	})

	return func() {
		cleanupItems()
		cleanupSpecies()
		cleanupSpells()
		cleanupBiomes()
		cleanupRooms()
		cleanupUsers()
		cleanupMobs()
		cleanupBuffs()
		cleanupKeywords()
	}
}

func getTestMobAndRoom(t *testing.T) (*mobs.Mob, *rooms.Room) {
	t.Helper()
	mob := mobs.GetInstance(100)
	require.NotNil(t, mob, "test mob instance 100 must exist")
	room := rooms.LoadRoom(mob.Character.RoomId)
	require.NotNil(t, room, "test room must exist")
	return mob, room
}

// ─── Registry Tests ─────────────────────────────────────────────────────────

func TestGetAllMobCommands(t *testing.T) {
	cmds := GetAllMobCommands()
	assert.NotEmpty(t, cmds)
	// Verify some expected commands exist
	found := map[string]bool{}
	for _, c := range cmds {
		found[c] = true
	}
	assert.True(t, found["attack"], "attack command should be registered")
	assert.True(t, found["go"], "go command should be registered")
	assert.True(t, found["say"], "say command should be registered")
	assert.True(t, found["noop"], "noop command should be registered")
}

// ─── TryCommand Tests ───────────────────────────────────────────────────────

func TestTryCommandNoop(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	handled, err := TryCommand("noop", "", 100)
	assert.True(t, handled)
	assert.NoError(t, err)
}

func TestTryCommandUnknown(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	handled, err := TryCommand("nonexistentcommand", "", 100)
	assert.False(t, handled)
	assert.NoError(t, err)
}

func TestTryCommandMobNotFound(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	handled, err := TryCommand("noop", "", 99999)
	assert.False(t, handled)
	_ = err
}

func TestTryCommandSay(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	handled, err := TryCommand("say", "hello world", 100)
	assert.True(t, handled)
	assert.NoError(t, err)
}

func TestTryCommandLook(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	handled, err := TryCommand("look", "", 100)
	assert.True(t, handled)
	assert.NoError(t, err)
}

func TestTryCommandMovement(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	handled, err := TryCommand("go", "north", 100)
	_ = handled
	_ = err
	// Reset mob position
	mob := mobs.GetInstance(100)
	if mob != nil {
		mob.Character.RoomId = 1
	}
}

// ─── Direct Command Tests ───────────────────────────────────────────────────

func TestNoop(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)
	handled, err := Noop("", mob, room)
	assert.True(t, handled)
	assert.NoError(t, err)
}

func TestSay(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	t.Run("say_message", func(t *testing.T) {
		handled, err := Say("hello adventurers", mob, room)
		assert.True(t, handled)
		assert.NoError(t, err)
	})

	t.Run("say_empty", func(t *testing.T) {
		handled, err := Say("", mob, room)
		_ = handled
		_ = err
	})
}

func TestEmote(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	handled, err := Emote("growls menacingly", mob, room)
	assert.True(t, handled)
	assert.NoError(t, err)
}

func TestShout(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	handled, err := Shout("intruders!", mob, room)
	assert.True(t, handled)
	assert.NoError(t, err)
}

func TestBroadcast(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	handled, err := Broadcast("test broadcast", mob, room)
	assert.True(t, handled)
	assert.NoError(t, err)
}

func TestLook(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	t.Run("look_no_args", func(t *testing.T) {
		handled, err := Look("", mob, room)
		assert.True(t, handled)
		assert.NoError(t, err)
	})

	t.Run("look_secretly", func(t *testing.T) {
		handled, err := Look("secretly", mob, room)
		assert.True(t, handled)
		assert.NoError(t, err)
	})

	t.Run("look_at_player", func(t *testing.T) {
		handled, err := Look("alice", mob, room)
		assert.True(t, handled)
		_ = err
	})

	t.Run("look_at_exit", func(t *testing.T) {
		handled, err := Look("north", mob, room)
		assert.True(t, handled)
		_ = err
	})
}

func TestGoMob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	t.Run("go_north", func(t *testing.T) {
		handled, err := Go("north", mob, room)
		_ = handled
		_ = err
		mob.Character.RoomId = 1
		room.AddMob(100)
	})

	t.Run("go_invalid", func(t *testing.T) {
		handled, err := Go("west", mob, room)
		_ = handled
		_ = err
	})

	t.Run("go_empty", func(t *testing.T) {
		handled, err := Go("", mob, room)
		_ = handled
		_ = err
	})
}

func TestAttackMob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	t.Run("attack_player", func(t *testing.T) {
		handled, err := Attack("alice", mob, room)
		assert.True(t, handled)
		_ = err
		mob.Character.Aggro = nil
	})

	t.Run("attack_nobody", func(t *testing.T) {
		handled, err := Attack("nobody", mob, room)
		assert.True(t, handled)
		_ = err
	})

	t.Run("attack_empty", func(t *testing.T) {
		handled, err := Attack("", mob, room)
		_ = handled
		_ = err
		mob.Character.Aggro = nil
	})

	t.Run("attack_star", func(t *testing.T) {
		handled, err := Attack("*", mob, room)
		_ = handled
		_ = err
		mob.Character.Aggro = nil
	})
}

func TestWander(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	handled, err := Wander("", mob, room)
	_ = handled
	_ = err
	// Reset position
	mob.Character.RoomId = 1
	room.AddMob(100)
}

func TestSuicideMob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	handled, err := Suicide("", mob, room)
	assert.True(t, handled)
	_ = err
}

func TestGetMob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	t.Run("get_no_args", func(t *testing.T) {
		handled, err := Get("", mob, room)
		assert.True(t, handled)
		_ = err
	})

	t.Run("get_nonexistent", func(t *testing.T) {
		handled, err := Get("nothing_here", mob, room)
		assert.True(t, handled)
		_ = err
	})
}

func TestDropMob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	t.Run("drop_nonexistent", func(t *testing.T) {
		handled, err := Drop("nothing", mob, room)
		assert.True(t, handled)
		_ = err
	})

	t.Run("drop_all_empty_inventory", func(t *testing.T) {
		handled, err := Drop("all", mob, room)
		assert.True(t, handled)
		_ = err
	})
}

func TestEquipMob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	t.Run("equip_no_args", func(t *testing.T) {
		handled, err := Equip("", mob, room)
		assert.True(t, handled)
		_ = err
	})

	t.Run("equip_nonexistent", func(t *testing.T) {
		handled, err := Equip("nothing", mob, room)
		assert.True(t, handled)
		_ = err
	})
}

func TestRemoveMob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	t.Run("remove_no_args", func(t *testing.T) {
		handled, err := Remove("", mob, room)
		assert.True(t, handled)
		_ = err
	})

	t.Run("remove_nonexistent", func(t *testing.T) {
		handled, err := Remove("nothing", mob, room)
		assert.True(t, handled)
		_ = err
	})
}

func TestGearupMob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	handled, err := Gearup("", mob, room)
	assert.True(t, handled)
	_ = err
}

func TestSayTo(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	t.Run("sayto_alice", func(t *testing.T) {
		handled, err := SayTo("alice hello there", mob, room)
		assert.True(t, handled)
		_ = err
	})

	t.Run("sayto_nobody", func(t *testing.T) {
		handled, err := SayTo("nobody hello", mob, room)
		assert.True(t, handled)
		_ = err
	})

	t.Run("sayto_empty", func(t *testing.T) {
		handled, err := SayTo("", mob, room)
		_ = handled
		_ = err
	})
}

func TestSayToOnly(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	handled, err := SayToOnly("alice secret message", mob, room)
	assert.True(t, handled)
	_ = err
}

func TestReplyTo(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	handled, err := ReplyTo("alice some reply", mob, room)
	_ = handled
	_ = err
}

func TestGiveMob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	t.Run("give_no_args", func(t *testing.T) {
		handled, err := Give("", mob, room)
		_ = handled
		_ = err
	})

	t.Run("give_gold_alice", func(t *testing.T) {
		mob.Character.Gold = 100
		handled, err := Give("50 gold alice", mob, room)
		_ = handled
		_ = err
	})
}

func TestShowMob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	t.Run("show_no_args", func(t *testing.T) {
		handled, err := Show("", mob, room)
		assert.True(t, handled)
		_ = err
	})
}

func TestSubmitMob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	t.Run("submit_not_in_combat", func(t *testing.T) {
		handled, err := Submit("", mob, room)
		assert.True(t, handled)
		_ = err
	})
}

func TestLookForTrouble(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	handled, err := LookForTrouble("", mob, room)
	_ = handled
	_ = err
}

// Regression for Fix A (shipped 2026-04-22): LookForTrouble must
// skip players with the NoAggroTarget buff flag (respawn grace).
// Without the early-continue, a hostile mob in the player's room
// would re-issue attack commands every idle tick; SetAggro bounces
// them, but the "prepares to fight" text fires anyway and combat
// resumes the moment grace expires.
func TestLookForTrouble_SkipsGraceProtectedPlayer(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	cleanupGraceBuff := buffs.SeedBuffsForTest(map[int]*buffs.BuffSpec{
		81: {
			BuffId:        81,
			Name:          "Respawn Grace",
			RoundInterval: 1,
			TriggerCount:  3,
			Flags:         []buffs.Flag{buffs.NoAggroTarget},
		},
	})
	defer cleanupGraceBuff()

	// Mob 100 (seeded in seedAllRegistries) is hostile:true and in
	// room 1 alongside user 1. A normal LookForTrouble pass would
	// pick user 1 as a target; the grace buff must prevent that.
	mob, room := getTestMobAndRoom(t)

	u1 := users.GetByUserId(1)
	require.NotNil(t, u1)
	u1.Character.Health = 100
	require.NoError(t, u1.Character.AddBuff(81, false))
	require.True(t, u1.Character.HasBuffFlag(buffs.NoAggroTarget),
		"grace buff must register NoAggroTarget flag")

	handled, err := LookForTrouble("", mob, room)
	require.NoError(t, err)
	require.True(t, handled)
	require.Nil(t, mob.Character.Aggro,
		"grace-protected player must not cause aggro acquisition")
}

func TestLookForAid(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	handled, err := LookForAid("", mob, room)
	_ = handled
	_ = err
}

func TestCallForHelp(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	handled, err := CallForHelp("", mob, room)
	_ = handled
	_ = err
}

func TestAidMob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	handled, err := Aid("", mob, room)
	_ = handled
	_ = err
}

func TestBefriendMob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	handled, err := Befriend("alice", mob, room)
	_ = handled
	_ = err
}

func TestConverseMob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	handled, err := Converse("", mob, room)
	_ = handled
	_ = err
}

func TestDespawnMob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	handled, err := Despawn("", mob, room)
	assert.True(t, handled)
	_ = err
}

func TestBreakMob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	handled, err := Break("", mob, room)
	assert.True(t, handled)
	_ = err
}

func TestDrinkMob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	handled, err := Drink("", mob, room)
	assert.True(t, handled)
	_ = err
}

func TestEatMob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	handled, err := Eat("", mob, room)
	assert.True(t, handled)
	_ = err
}

func TestSneakMob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	handled, err := Sneak("", mob, room)
	_ = handled
	_ = err
}

func TestPathto(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	t.Run("pathto_room", func(t *testing.T) {
		handled, err := Pathto("2", mob, room)
		_ = handled
		_ = err
	})

	t.Run("pathto_empty", func(t *testing.T) {
		handled, err := Pathto("", mob, room)
		_ = handled
		_ = err
	})
}

func TestPortal(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	handled, err := Portal("2", mob, room)
	_ = handled
	_ = err
}

func TestPutMob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	handled, err := Put("nothing in nothing", mob, room)
	assert.True(t, handled)
	_ = err
}

func TestBashMob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	// Not in combat
	handled, err := Bash("", mob, room)
	assert.True(t, handled)
	_ = err
}

func TestGrappleMob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	handled, err := Grapple("", mob, room)
	assert.True(t, handled)
	_ = err
}

func TestKickMob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	handled, err := Kick("", mob, room)
	assert.True(t, handled)
	_ = err
}

func TestTripMob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	handled, err := Trip("", mob, room)
	assert.True(t, handled)
	_ = err
}

func TestCastMob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	t.Run("cast_no_args", func(t *testing.T) {
		handled, err := Cast("", mob, room)
		_ = handled
		_ = err
	})

	t.Run("cast_spell_name", func(t *testing.T) {
		mob.Character.SpellBook = map[string]int{"sparks": 1}
		handled, err := Cast("sparks", mob, room)
		_ = handled
		_ = err
	})
}

func TestShootMob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	handled, err := Shoot("alice", mob, room)
	_ = handled
	_ = err
}

func TestGiveQuest(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	t.Run("givequest_no_args", func(t *testing.T) {
		handled, err := GiveQuest("", mob, room)
		_ = handled
		_ = err
	})

	t.Run("givequest_alice", func(t *testing.T) {
		handled, err := GiveQuest("alice 1", mob, room)
		_ = handled
		_ = err
	})
}

func TestSelljunk(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	handled, err := Selljunk("health potion", mob, room)
	_ = handled
	_ = err
}

// ─── TryCommand routing tests ───────────────────────────────────────────────

func TestTryCommandRouting(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	cmds := []struct {
		cmd  string
		rest string
	}{
		{"noop", ""},
		{"say", "test"},
		{"emote", "waves"},
		{"shout", "help!"},
		{"look", ""},
		{"attack", ""},
		{"drop", "all"},
		{"get", ""},
		{"equip", ""},
		{"remove", ""},
		{"gearup", ""},
		{"broadcast", "test"},
		{"submit", ""},
		{"break", ""},
		{"drink", ""},
		{"eat", ""},
		{"wander", ""},
		{"despawn", ""},
	}

	for _, c := range cmds {
		t.Run("trycommand_"+c.cmd, func(t *testing.T) {
			handled, err := TryCommand(c.cmd, c.rest, 100)
			_ = handled
			_ = err
		})
	}
}

// ─── RegisterCommand ────────────────────────────────────────────────────────

func TestRegisterCommand(t *testing.T) {
	RegisterCommand("testcmd", func(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
		return true, nil
	}, false)

	cmds := GetAllMobCommands()
	found := false
	for _, c := range cmds {
		if c == "testcmd" {
			found = true
			break
		}
	}
	assert.True(t, found, "testcmd should be registered")
}

// ─── Combat state tests ────────────────────────────────────────────────────

func TestAttackInCombat(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	// Set mob in combat
	mob.Character.Aggro = &characters.Aggro{UserId: 1}

	t.Run("attack_while_in_combat", func(t *testing.T) {
		handled, err := Attack("bobrick", mob, room)
		_ = handled
		_ = err
	})

	t.Run("bash_in_combat", func(t *testing.T) {
		handled, err := Bash("", mob, room)
		_ = handled
		_ = err
	})

	t.Run("grapple_in_combat", func(t *testing.T) {
		handled, err := Grapple("", mob, room)
		_ = handled
		_ = err
	})

	t.Run("kick_in_combat", func(t *testing.T) {
		handled, err := Kick("", mob, room)
		_ = handled
		_ = err
	})

	t.Run("trip_in_combat", func(t *testing.T) {
		handled, err := Trip("", mob, room)
		_ = handled
		_ = err
	})

	mob.Character.Aggro = nil
}

// ─── Deeper branch tests ────────────────────────────────────────────────────

func TestSubmitInCombat(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	// In combat but not grounded
	mob.Character.Aggro = &characters.Aggro{UserId: 1}
	setCombatPositionParallel(&mob.Character, characters.PositionStanding)
	handled, err := Submit("", mob, room)
	assert.True(t, handled)
	_ = err

	// In combat and grounded but not grapple controller
	setCombatPositionParallel(&mob.Character, characters.PositionGrounded)
	handled, err = Submit("", mob, room)
	assert.True(t, handled)
	_ = err

	mob.Character.Aggro = nil
	setCombatPositionParallel(&mob.Character, characters.PositionStanding)
}

func TestWanderBranches(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	t.Run("charmed_mob", func(t *testing.T) {
		mob.Character.Charmed = &characters.CharmInfo{UserId: 1, RoundsRemaining: 10}
		handled, err := Wander("", mob, room)
		assert.True(t, handled)
		assert.Error(t, err)
		mob.Character.Charmed = nil
	})

	t.Run("max_wander_zero", func(t *testing.T) {
		mob.MaxWander = 0
		handled, err := Wander("", mob, room)
		assert.True(t, handled)
		_ = err
		mob.MaxWander = 5
	})

	t.Run("wander_loot", func(t *testing.T) {
		mob.MaxWander = 5
		handled, err := Wander("loot", mob, room)
		_ = handled
		_ = err
	})

	t.Run("away_from_home_exceeded", func(t *testing.T) {
		mob.Character.RoomId = 2
		mob.HomeRoomId = 1
		mob.MaxWander = 1
		mob.WanderCount = 5
		handled, err := Wander("", mob, room)
		_ = handled
		_ = err
		mob.Character.RoomId = 1
		mob.WanderCount = 0
	})
}

func TestGetMobDeeper(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	t.Run("get_all_nothing", func(t *testing.T) {
		handled, err := Get("all", mob, room)
		assert.True(t, handled)
		_ = err
	})

	t.Run("get_gold_from_room", func(t *testing.T) {
		room.Gold = 50
		handled, err := Get("gold", mob, room)
		assert.True(t, handled)
		_ = err
		room.Gold = 0
	})
}

func TestEquipRemoveDeeper(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	t.Run("equip_all", func(t *testing.T) {
		handled, err := Equip("all", mob, room)
		assert.True(t, handled)
		_ = err
	})

	t.Run("remove_all", func(t *testing.T) {
		handled, err := Remove("all", mob, room)
		assert.True(t, handled)
		_ = err
	})
}

func TestShootInCombat(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	mob.Character.Aggro = &characters.Aggro{UserId: 1}
	handled, err := Shoot("", mob, room)
	_ = handled
	_ = err
	mob.Character.Aggro = nil
}

func TestConverseBranches(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	t.Run("no_converse_file", func(t *testing.T) {
		handled, err := Converse("", mob, room)
		assert.True(t, handled)
		_ = err
	})
}

func TestCallForHelpInCombat(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	mob.Character.Aggro = &characters.Aggro{UserId: 1}
	handled, err := CallForHelp("", mob, room)
	_ = handled
	_ = err
	mob.Character.Aggro = nil
}

func TestSayEmpty(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	// Say with empty string
	handled, err := Say("", mob, room)
	assert.True(t, handled)
	_ = err
}

func TestDropAllWithGold(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	mob.Character.Gold = 100
	handled, err := Drop("all", mob, room)
	assert.True(t, handled)
	_ = err
	mob.Character.Gold = 0
}
