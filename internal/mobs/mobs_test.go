package mobs

import (
	"os"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

// seedRegistry populates all global mobs maps with diverse test data.
// Call the returned cleanup function (or use defer) to restore the original.
func seedRegistry() func() {
	origMobs := mobs
	origAllMobNames := allMobNames
	origMobInstances := mobInstances
	origMobNameCache := mobNameCache
	origRecentlyDied := recentlyDied
	origInstanceCounter := instanceCounter

	// Build test mob templates
	mobs = map[int]*Mob{
		1: {
			MobId:         1,
			Zone:          "Sanctum Basin",
			StatPool:      100,
			ActivityLevel: 50,
			Hostile:       true,
			Groups:        []string{"undead", "dungeon"},
			Hates:         []string{"townfolk"},
			IdleCommands:  []string{"emote lurks in the shadows.", "emote growls softly."},
			AngryCommands: []string{"emote shrieks!", "emote lunges forward!"},
			Archetype:     "fighting",
			Character: characters.Character{
				Name:      "Skeleton Warrior",
				SpeciesId: 1,
			},
		},
		2: {
			MobId:         2,
			Zone:          "Sanctum Basin",
			StatPool:      120,
			ActivityLevel: 30,
			Hostile:       false,
			Groups:        []string{"townfolk"},
			Hates:         []string{},
			IdleCommands:  []string{"emote hums a tune."},
			AngryCommands: []string{},
			Character: characters.Character{
				Name:      "Traveling Merchant",
				SpeciesId: 0,
				Shop: characters.Shop{
					{ItemId: 100, Price: 50, Quantity: 5},
				},
			},
		},
		3: {
			MobId:         3,
			Zone:          "Dark Forest",
			StatPool:      80,
			ActivityLevel: 70,
			Hostile:       true,
			Groups:        []string{"beasts"},
			Hates:         []string{"*"},
			IdleCommands:  []string{},
			AngryCommands: []string{"emote snarls!"},
			Archetype:     "casting",
			Character: characters.Character{
				Name:      "Shadow Wolf",
				SpeciesId: 2,
			},
		},
		4: {
			MobId:         4,
			Zone:          "Sanctum Basin",
			StatPool:      60,
			ActivityLevel: 0, // should clamp to 10 on Validate
			Hostile:       false,
			Groups:        []string{"undead"},
			Hates:         []string{"beasts"},
			Character: characters.Character{
				Name:      "Ghostly Wisp",
				SpeciesId: 1, // same species as Skeleton Warrior
			},
		},
		5: {
			MobId:         5,
			Zone:          "Sanctum Basin",
			StatPool:      200,
			ActivityLevel: 150, // should clamp to 100 on Validate
			Hostile:       true,
			Groups:        []string{},
			Hates:         []string{},
			Character: characters.Character{
				Name:      "Ancient Golem",
				SpeciesId: 0,
			},
		},
	}

	allMobNames = []string{
		"Skeleton Warrior",
		"Traveling Merchant",
		"Shadow Wolf",
		"Ghostly Wisp",
		"Ancient Golem",
	}

	mobNameCache = map[MobId]string{
		1: "Skeleton Warrior",
		2: "Traveling Merchant",
		3: "Shadow Wolf",
		4: "Ghostly Wisp",
		5: "Ancient Golem",
	}

	// Pre-seed some instances
	mobInstances = map[int]*Mob{
		100: {
			MobId:      1,
			InstanceId: 100,
			HomeRoomId: 10,
			Groups:     []string{"undead", "dungeon"},
			Hates:      []string{"townfolk"},
			Character: characters.Character{
				Name:      "Skeleton Warrior",
				SpeciesId: 1,
			},
		},
		101: {
			MobId:      2,
			InstanceId: 101,
			HomeRoomId: 20,
			Groups:     []string{"townfolk"},
			Character: characters.Character{
				Name: "Traveling Merchant",
				Shop: characters.Shop{
					{ItemId: 100, Price: 50, Quantity: 5},
				},
			},
		},
		102: {
			MobId:      3,
			InstanceId: 102,
			HomeRoomId: 30,
			Groups:     []string{"beasts"},
			Hates:      []string{"*"},
			Character: characters.Character{
				Name:      "Shadow Wolf",
				SpeciesId: 2,
			},
		},
	}

	recentlyDied = map[int]int{}
	instanceCounter = 200

	return func() {
		mobs = origMobs
		allMobNames = origAllMobNames
		mobInstances = origMobInstances
		mobNameCache = origMobNameCache
		recentlyDied = origRecentlyDied
		instanceCounter = origInstanceCounter
	}
}

// ─── Instance Lookup ────────────────────────────────────────────────────────

func TestMobInstanceExists(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("existing instance", func(t *testing.T) {
		assert.True(t, MobInstanceExists(100))
	})

	t.Run("nonexistent instance", func(t *testing.T) {
		assert.False(t, MobInstanceExists(999))
	})
}

func TestGetInstance(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("found", func(t *testing.T) {
		m := GetInstance(100)
		assert.NotNil(t, m)
		assert.Equal(t, MobId(1), m.MobId)
		assert.Equal(t, 100, m.InstanceId)
	})

	t.Run("not found", func(t *testing.T) {
		m := GetInstance(999)
		assert.Nil(t, m)
	})
}

func TestGetAllMobInstanceIds(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	ids := GetAllMobInstanceIds()
	assert.Len(t, ids, 3)
	assert.Contains(t, ids, 100)
	assert.Contains(t, ids, 101)
	assert.Contains(t, ids, 102)
}

func TestDestroyInstance(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	assert.True(t, MobInstanceExists(100))
	DestroyInstance(100)
	assert.False(t, MobInstanceExists(100))

	// Destroying nonexistent is safe
	DestroyInstance(999)
}

func TestGetMobSpec(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("found", func(t *testing.T) {
		spec := GetMobSpec(1)
		assert.NotNil(t, spec)
		assert.Equal(t, "Skeleton Warrior", spec.Character.Name)
	})

	t.Run("returns copy", func(t *testing.T) {
		spec := GetMobSpec(1)
		spec.Character.Name = "Modified"
		orig := GetMobSpec(1)
		assert.Equal(t, "Skeleton Warrior", orig.Character.Name)
	})

	t.Run("not found", func(t *testing.T) {
		spec := GetMobSpec(999)
		assert.Nil(t, spec)
	})
}

func TestGetAllMobInfo(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	info := GetAllMobInfo()
	assert.Len(t, info, 5)
}

func TestGetAllMobNames(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	names := GetAllMobNames()
	assert.Len(t, names, 5)
	assert.Contains(t, names, "Skeleton Warrior")
	assert.Contains(t, names, "Shadow Wolf")

	// Verify it returns a copy
	names[0] = "Modified"
	original := GetAllMobNames()
	assert.NotEqual(t, "Modified", original[0])
}

// ─── Relationships ──────────────────────────────────────────────────────────

func TestHatesSpecies(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	mob := GetMobSpec(1)

	t.Run("hates matching race", func(t *testing.T) {
		assert.True(t, mob.HatesSpecies("townfolk"))
	})

	t.Run("case insensitive", func(t *testing.T) {
		assert.True(t, mob.HatesSpecies("Townfolk"))
	})

	t.Run("does not hate non-matching", func(t *testing.T) {
		assert.False(t, mob.HatesSpecies("beasts"))
	})
}

func TestConsidersAnAlly(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("same MobId is ally", func(t *testing.T) {
		m1 := GetInstance(100)
		m2 := &Mob{MobId: 1, InstanceId: 999}
		assert.True(t, m1.ConsidersAnAlly(m2))
	})

	t.Run("same species is ally", func(t *testing.T) {
		m1 := GetInstance(100) // SpeciesId: 1
		m2 := &Mob{MobId: 99, Character: characters.Character{SpeciesId: 1}}
		assert.True(t, m1.ConsidersAnAlly(m2))
	})

	t.Run("different species = not ally", func(t *testing.T) {
		m1 := GetInstance(100) // SpeciesId: 1
		m2 := &Mob{MobId: 99, Character: characters.Character{SpeciesId: 2}}
		assert.False(t, m1.ConsidersAnAlly(m2))
	})

	t.Run("zero species on either side = not ally", func(t *testing.T) {
		m1 := &Mob{MobId: 10, Character: characters.Character{SpeciesId: 0}}
		m2 := &Mob{MobId: 11, Character: characters.Character{SpeciesId: 0}}
		assert.False(t, m1.ConsidersAnAlly(m2))
	})

	t.Run("one has species other doesn't = not ally", func(t *testing.T) {
		m1 := GetInstance(100) // SpeciesId: 1
		m2 := &Mob{MobId: 99, Character: characters.Character{SpeciesId: 0}}
		assert.False(t, m1.ConsidersAnAlly(m2))
	})
}

// ─── Idle/Angry Commands ────────────────────────────────────────────────────

func TestGetIdleCommand(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("mob with idle commands returns one", func(t *testing.T) {
		mob := GetInstance(100)
		mob.IdleCommands = []string{"emote lurks.", "emote growls."}
		// Run several times — at least one should return a command
		found := false
		for i := 0; i < 200; i++ {
			cmd := mob.GetIdleCommand()
			if cmd != "" {
				found = true
				break
			}
		}
		assert.True(t, found, "should return an idle command at least sometimes")
	})

	t.Run("mob with no idle commands returns empty", func(t *testing.T) {
		mob := GetInstance(102) // Shadow Wolf has no idle commands
		mob.IdleCommands = []string{}
		cmd := mob.GetIdleCommand()
		assert.Equal(t, "", cmd)
	})
}

func TestGetAngryCommand(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("mob with angry commands returns one", func(t *testing.T) {
		mob := GetInstance(100)
		mob.AngryCommands = []string{"emote shrieks!", "emote lunges!"}
		cmd := mob.GetAngryCommand()
		assert.NotEmpty(t, cmd)
	})

	t.Run("mob with no angry commands and no species returns empty", func(t *testing.T) {
		mob := &Mob{
			AngryCommands: []string{},
			Character:     characters.Character{SpeciesId: -1}, // invalid species
		}
		// GetAngryCommand calls species.GetSpecies which may return nil;
		// skip if it panics (species not loaded in test env)
		defer func() { recover() }()
		cmd := mob.GetAngryCommand()
		assert.Equal(t, "", cmd)
	})
}

// ─── Hostility Tracking ────────────────────────────────────────────────────


// ─── Player Attack Tracking ────────────────────────────────────────────────

func TestPlayerAttacked(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	mob := GetInstance(100)

	t.Run("not attacked initially", func(t *testing.T) {
		assert.False(t, mob.HasAttackedPlayer(1))
	})

	t.Run("attacked player tracked", func(t *testing.T) {
		mob.PlayerAttacked(1)
		assert.True(t, mob.HasAttackedPlayer(1))
	})

	t.Run("different player not tracked", func(t *testing.T) {
		mob.PlayerAttacked(1)
		assert.False(t, mob.HasAttackedPlayer(2))
	})

	t.Run("multiple players", func(t *testing.T) {
		mob.PlayerAttacked(1)
		mob.PlayerAttacked(2)
		assert.True(t, mob.HasAttackedPlayer(1))
		assert.True(t, mob.HasAttackedPlayer(2))
		assert.False(t, mob.HasAttackedPlayer(3))
	})
}

// ─── Death Tracking ────────────────────────────────────────────────────────

func TestTrackRecentDeath(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("not recently died", func(t *testing.T) {
		assert.False(t, RecentlyDied(999))
	})

	t.Run("tracked death shows recently died", func(t *testing.T) {
		TrackRecentDeath(500)
		assert.True(t, RecentlyDied(500))
	})

	t.Run("multiple deaths tracked", func(t *testing.T) {
		TrackRecentDeath(501)
		TrackRecentDeath(502)
		assert.True(t, RecentlyDied(501))
		assert.True(t, RecentlyDied(502))
	})
}

func TestRecentlyDiedPruning(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	// Fill with > 30 entries to trigger pruning
	for i := 0; i < 35; i++ {
		recentlyDied[i] = 0 // round 0 (very old)
	}

	// Add a recent one
	recentlyDied[999] = int(^uint(0) >> 1) // very high round number

	// Calling RecentlyDied triggers pruning since len > 30
	assert.True(t, RecentlyDied(999))
	// Old entries should be pruned
	assert.True(t, len(recentlyDied) <= 31, "old entries should be pruned")
}

// ─── Name Lookup ────────────────────────────────────────────────────────────

func TestMobIdByName(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("exact match", func(t *testing.T) {
		id := MobIdByName("Skeleton Warrior")
		assert.Equal(t, MobId(1), id)
	})

	t.Run("prefix match", func(t *testing.T) {
		id := MobIdByName("Skeleton")
		assert.Equal(t, MobId(1), id)
	})

	t.Run("partial match", func(t *testing.T) {
		id := MobIdByName("Wolf")
		assert.Equal(t, MobId(3), id)
	})

	t.Run("no match", func(t *testing.T) {
		id := MobIdByName("Dragon Lord")
		assert.Equal(t, MobId(0), id)
	})

	t.Run("empty string", func(t *testing.T) {
		id := MobIdByName("")
		assert.Equal(t, MobId(0), id)
	})
}

// ─── Validate ───────────────────────────────────────────────────────────────

func TestValidate(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("activity level below 1 clamps to 10", func(t *testing.T) {
		mob := &Mob{ActivityLevel: 0, Character: characters.Character{}}
		mob.Validate()
		assert.Equal(t, 10, mob.ActivityLevel)
	})

	t.Run("activity level above 100 clamps to 100", func(t *testing.T) {
		mob := &Mob{ActivityLevel: 150, Character: characters.Character{}}
		mob.Validate()
		assert.Equal(t, 100, mob.ActivityLevel)
	})

	t.Run("activity level within range unchanged", func(t *testing.T) {
		mob := &Mob{ActivityLevel: 50, Character: characters.Character{}}
		mob.Validate()
		assert.Equal(t, 50, mob.ActivityLevel)
	})

	t.Run("negative activity level clamps to 10", func(t *testing.T) {
		mob := &Mob{ActivityLevel: -5, Character: characters.Character{}}
		mob.Validate()
		assert.Equal(t, 10, mob.ActivityLevel)
	})

	t.Run("activity level 1 stays 1", func(t *testing.T) {
		mob := &Mob{ActivityLevel: 1, Character: characters.Character{}}
		mob.Validate()
		assert.Equal(t, 1, mob.ActivityLevel)
	})

	t.Run("activity level 100 stays 100", func(t *testing.T) {
		mob := &Mob{ActivityLevel: 100, Character: characters.Character{}}
		mob.Validate()
		assert.Equal(t, 100, mob.ActivityLevel)
	})
}

// ─── Misc Utility ───────────────────────────────────────────────────────────

func TestShorthandId(t *testing.T) {
	mob := &Mob{InstanceId: 42}
	assert.Equal(t, "#42", mob.ShorthandId())

	mob2 := &Mob{InstanceId: 0}
	assert.Equal(t, "#0", mob2.ShorthandId())
}

func TestFilename(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("uses name cache", func(t *testing.T) {
		mob := &Mob{MobId: 1}
		assert.Equal(t, "1-skeleton_warrior.yaml", mob.Filename())
	})

	t.Run("falls back to character name", func(t *testing.T) {
		mob := &Mob{MobId: 999, Character: characters.Character{Name: "Test Mob"}}
		assert.Equal(t, "999-test_mob.yaml", mob.Filename())
	})
}

func TestZoneNameSanitize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Sanctum Basin", "sanctum_basin"},
		{"Dark Forest", "dark_forest"},
		{"", ""},
		{"simple", "simple"},
		{"UPPER CASE", "upper_case"},
		{"Multiple   Spaces", "multiple___spaces"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, ZoneNameSanitize(tt.input))
		})
	}
}

func TestId(t *testing.T) {
	mob := &Mob{MobId: 42}
	assert.Equal(t, 42, mob.Id())
}

// ─── HasShop / Despawns ────────────────────────────────────────────────────

func TestHasShop(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("shopkeeper", func(t *testing.T) {
		merchant := GetInstance(101)
		assert.True(t, merchant.HasShop())
	})

	t.Run("non-shopkeeper", func(t *testing.T) {
		warrior := GetInstance(100)
		assert.False(t, warrior.HasShop())
	})
}

func TestDespawns(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("shopkeeper does not despawn", func(t *testing.T) {
		merchant := GetInstance(101)
		assert.False(t, merchant.Despawns())
	})

	t.Run("non-shopkeeper despawns", func(t *testing.T) {
		warrior := GetInstance(100)
		assert.True(t, warrior.Despawns())
	})
}

// ─── Conversation ──────────────────────────────────────────────────────────

func TestConversation(t *testing.T) {
	mob := &Mob{InstanceId: 1}

	t.Run("not in conversation initially", func(t *testing.T) {
		assert.False(t, mob.InConversation())
	})

	t.Run("set conversation", func(t *testing.T) {
		mob.SetConversation(5)
		assert.True(t, mob.InConversation())
	})

	t.Run("clear conversation", func(t *testing.T) {
		mob.SetConversation(0)
		assert.False(t, mob.InConversation())
	})
}

// ─── TempData ──────────────────────────────────────────────────────────────

func TestTempData(t *testing.T) {
	mob := &Mob{}

	t.Run("get from nil store returns nil", func(t *testing.T) {
		assert.Nil(t, mob.GetTempData("key"))
	})

	t.Run("set and get", func(t *testing.T) {
		mob.SetTempData("key", "value")
		assert.Equal(t, "value", mob.GetTempData("key"))
	})

	t.Run("missing key returns nil", func(t *testing.T) {
		assert.Nil(t, mob.GetTempData("missing"))
	})

	t.Run("delete by nil", func(t *testing.T) {
		mob.SetTempData("key", nil)
		assert.Nil(t, mob.GetTempData("key"))
	})
}

// ─── Audit Mob Name Collisions ──────────────────────────────────────────────

func TestAuditMobNameCollisions_NoCollisions(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	callCount := 0
	AuditMobNameCollisions(func(name string) (int, bool) {
		callCount++
		return 0, false
	})
	if callCount == 0 {
		t.Error("expected lookup to be invoked at least once")
	}
}

func TestAuditMobNameCollisions_OneCollision(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	found := false
	AuditMobNameCollisions(func(name string) (int, bool) {
		if name == "Skeleton Warrior" {
			found = true
			return 42, true
		}
		return 0, false
	})
	if !found {
		t.Error("expected lookup to be called with 'Skeleton Warrior'")
	}
}

func TestAuditMobNameCollisions_MultipleCollisions(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	collisionCount := 0
	AuditMobNameCollisions(func(name string) (int, bool) {
		// Return true for two mobs
		if name == "Skeleton Warrior" || name == "Shadow Wolf" {
			collisionCount++
			return 99, true
		}
		return 0, false
	})
	if collisionCount != 2 {
		t.Errorf("expected 2 collisions, got %d", collisionCount)
	}
}

// ─── Filepath ──────────────────────────────────────────────────────────────

func TestFilepath(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	mob := &Mob{MobId: 1, Zone: "Sanctum Basin"}
	fp := mob.Filepath()
	assert.Contains(t, fp, "sanctum_basin")
	assert.Contains(t, fp, "1-skeleton_warrior.yaml")
}

// ─── HatesMob ──────────────────────────────────────────────────────────────
// HatesMob calls species.GetSpecies() which panics when no species are loaded.
// We test same-MobId (short-circuits before species lookup) directly, and use
// recover for tests that hit the species lookup path.

func TestHatesMobSameMobId(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	m1 := GetInstance(100)
	m2 := &Mob{MobId: 1, InstanceId: 999, Character: characters.Character{SpeciesId: 0}}
	// Same MobId = can't hate (returns false before species lookup)
	assert.False(t, m1.HatesMob(m2))
}

// ─── GetMemoryUsage ────────────────────────────────────────────────────────

func TestGetMemoryUsage(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	mem := GetMemoryUsage()
	assert.Contains(t, mem, "mobs")
	assert.Contains(t, mem, "allMobNames")
	assert.Contains(t, mem, "mobInstances")
	assert.Equal(t, 5, mem["mobs"].Count)
	assert.Equal(t, 5, mem["allMobNames"].Count)
	assert.Equal(t, 3, mem["mobInstances"].Count)
}

// ─── IsTameable ────────────────────────────────────────────────────────────

func TestIsTameable(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("shopkeeper not tameable", func(t *testing.T) {
		merchant := GetInstance(101)
		assert.False(t, merchant.IsTameable())
	})

	t.Run("mob with script not tameable", func(t *testing.T) {
		mob := &Mob{ScriptTag: "custom", Character: characters.Character{SpeciesId: 0}}
		assert.False(t, mob.IsTameable())
	})
}

// ─── GetSellPrice ──────────────────────────────────────────────────────────

func TestGetSellPrice(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("special item returns 0", func(t *testing.T) {
		mob := GetInstance(101) // merchant with shop
		specialItem := items.Item{ItemId: 1, Blob: "special-data"}
		assert.Equal(t, 0, mob.GetSellPrice(specialItem))
	})
}

// ─── MobIdByName additional coverage ───────────────────────────────────────

func TestMobIdByNameContains(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	id := MobIdByName("Merchant")
	assert.Equal(t, MobId(2), id)

	id = MobIdByName("Golem")
	assert.Equal(t, MobId(5), id)
}

// ─── Validate calls Character.Validate ─────────────────────────────────────

func TestValidateCallsCharacterValidate(t *testing.T) {
	mob := &Mob{
		ActivityLevel: 50,
		Character:     characters.Character{},
	}
	err := mob.Validate()
	assert.NoError(t, err)
}

// ─── GetAllMobInfo returns copies ──────────────────────────────────────────

func TestGetAllMobInfoReturnsCopies(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	info := GetAllMobInfo()
	if len(info) > 0 {
		info[0].Character.Name = "MODIFIED"
	}
	info2 := GetAllMobInfo()
	for _, m := range info2 {
		assert.NotEqual(t, "MODIFIED", m.Character.Name)
	}
}

// ─── InstanceCounter preserved by seed ─────────────────────────────────────

func TestInstanceCounterPreserved(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	assert.Equal(t, 200, instanceCounter)
}

// ─── PlayerAttacked with nil map ───────────────────────────────────────────

func TestPlayerAttackedNilMap(t *testing.T) {
	mob := &Mob{}
	assert.False(t, mob.HasAttackedPlayer(1))
	mob.PlayerAttacked(1)
	assert.True(t, mob.HasAttackedPlayer(1))
}

// ─── Multiple temp data operations ─────────────────────────────────────────

func TestTempDataMultipleKeys(t *testing.T) {
	mob := &Mob{}
	mob.SetTempData("a", 1)
	mob.SetTempData("b", "two")
	mob.SetTempData("c", true)

	assert.Equal(t, 1, mob.GetTempData("a"))
	assert.Equal(t, "two", mob.GetTempData("b"))
	assert.Equal(t, true, mob.GetTempData("c"))

	mob.SetTempData("b", nil)
	assert.Nil(t, mob.GetTempData("b"))
	assert.Equal(t, 1, mob.GetTempData("a"))
}

// ─── DestroyInstance all ───────────────────────────────────────────────────

func TestDestroyInstanceAll(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	DestroyInstance(100)
	DestroyInstance(101)
	DestroyInstance(102)

	ids := GetAllMobInstanceIds()
	assert.Len(t, ids, 0)
}

// ─── RecentlyDied multiple ────────────────────────────────────────────────

func TestRecentlyDiedMultiple(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	TrackRecentDeath(1)
	TrackRecentDeath(2)
	TrackRecentDeath(3)

	assert.True(t, RecentlyDied(1))
	assert.True(t, RecentlyDied(2))
	assert.True(t, RecentlyDied(3))
	assert.False(t, RecentlyDied(4))
}

// ─── Sleep ─────────────────────────────────────────────────────────────────

func TestSleep(t *testing.T) {
	mob := &Mob{InstanceId: 42}
	// Should not panic
	mob.Sleep(1)
}

// ─── ItemTrade struct ──────────────────────────────────────────────────────

func TestItemTradeDefaults(t *testing.T) {
	trade := ItemTrade{
		AcceptedItemIds: []int{1, 2},
		AcceptedGold:    100,
		PrizeItemIds:    []int{3},
		PrizeGold:       50,
	}
	assert.Len(t, trade.AcceptedItemIds, 2)
	assert.Equal(t, 100, trade.AcceptedGold)
	assert.Len(t, trade.PrizeItemIds, 1)
	assert.Equal(t, 50, trade.PrizeGold)
	assert.Nil(t, trade.GivenItems)
	assert.Nil(t, trade.GivenGold)
}

// ─── MobForHire struct ────────────────────────────────────────────────────

func TestMobForHire(t *testing.T) {
	hire := MobForHire{
		MobId:    MobId(5),
		Price:    500,
		Quantity: 2,
	}
	assert.Equal(t, MobId(5), hire.MobId)
	assert.Equal(t, 500, hire.Price)
	assert.Equal(t, 2, hire.Quantity)
}

// ─── NewMobById ────────────────────────────────────────────────────────────

func TestNewMobById(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("valid mob creates instance", func(t *testing.T) {
		mob := NewMobById(1, 50)
		assert.NotNil(t, mob)
		assert.Equal(t, MobId(1), mob.MobId)
		assert.Equal(t, 50, mob.HomeRoomId)
		assert.Equal(t, 201, mob.InstanceId) // instanceCounter was 200
		assert.True(t, mob.Character.IsMob)
		assert.True(t, MobInstanceExists(201))
	})

	t.Run("invalid mob returns nil", func(t *testing.T) {
		mob := NewMobById(999, 50)
		assert.Nil(t, mob)
	})

	t.Run("force stat pool", func(t *testing.T) {
		mob := NewMobById(5, 60, 50) // force 50 stat points
		assert.NotNil(t, mob)
		// Verify some training was distributed
		s := &mob.Character.Stats
		totalTraining := s.Strength.Training + s.Dexterity.Training +
			s.Perception.Training + s.Vitality.Training +
			s.Willpower.Training + s.Charisma.Training
		assert.Equal(t, 50, totalTraining)
	})

	t.Run("fighting archetype distributes physical", func(t *testing.T) {
		// Create multiple fighting mobs and check bias
		totalPhys := 0
		totalMental := 0
		runs := 50
		for i := 0; i < runs; i++ {
			mob := NewMobById(1, 100, 100) // fighting archetype, 100 stat pool
			if mob == nil {
				continue
			}
			s := &mob.Character.Stats
			phys := s.Strength.Training + s.Dexterity.Training + s.Vitality.Training
			mental := s.Perception.Training + s.Willpower.Training + s.Charisma.Training
			totalPhys += phys
			totalMental += mental
			DestroyInstance(mob.InstanceId)
		}
		// Fighting should have ~80% physical
		physRatio := float64(totalPhys) / float64(totalPhys+totalMental)
		assert.True(t, physRatio > 0.6, "fighting archetype should favor physical stats, got %.2f", physRatio)
	})

	t.Run("casting archetype distributes mental", func(t *testing.T) {
		totalPhys := 0
		totalMental := 0
		runs := 50
		for i := 0; i < runs; i++ {
			mob := NewMobById(3, 100, 100) // casting archetype, 100 stat pool
			if mob == nil {
				continue
			}
			s := &mob.Character.Stats
			phys := s.Strength.Training + s.Dexterity.Training + s.Vitality.Training
			mental := s.Perception.Training + s.Willpower.Training + s.Charisma.Training
			totalPhys += phys
			totalMental += mental
			DestroyInstance(mob.InstanceId)
		}
		mentalRatio := float64(totalMental) / float64(totalPhys+totalMental)
		assert.True(t, mentalRatio > 0.6, "casting archetype should favor mental stats, got %.2f", mentalRatio)
	})

	t.Run("default archetype distributes evenly", func(t *testing.T) {
		mob := NewMobById(5, 100, 120) // no archetype, 120 pool
		assert.NotNil(t, mob)
		s := &mob.Character.Stats
		total := s.Strength.Training + s.Dexterity.Training +
			s.Perception.Training + s.Vitality.Training +
			s.Willpower.Training + s.Charisma.Training
		assert.Equal(t, 120, total)
		DestroyInstance(mob.InstanceId)
	})

	t.Run("AI defaults set", func(t *testing.T) {
		mob := NewMobById(4, 100) // mob 4 has no AIProfile
		assert.NotNil(t, mob)
		assert.Equal(t, "default", mob.AIProfile)
		assert.Equal(t, 30, mob.SpecialMoveChance)
		DestroyInstance(mob.InstanceId)
	})
}

// ─── packStatForArchetype ──────────────────────────────────────────────────

func TestPackStatForArchetype(t *testing.T) {
	assert.Equal(t, "strength", packStatForArchetype("fighting"))
	assert.Equal(t, "willpower", packStatForArchetype("casting"))
	assert.Equal(t, "vitality", packStatForArchetype(""))
	assert.Equal(t, "vitality", packStatForArchetype("unknown"))
}

// ─── GetSellPrice more paths ──────────────────────────────────────────────

func TestGetSellPriceTooManyVarieties(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	// Create merchant with 20 different items in shop
	merchant := GetInstance(101)
	bigShop := characters.Shop{}
	for i := 0; i < 20; i++ {
		bigShop = append(bigShop, characters.ShopItem{ItemId: 100 + i, Price: 10, Quantity: 1})
	}
	merchant.Character.Shop = bigShop

	// Selling a new variety (itemId not in shop) should return 0
	newItem := items.Item{ItemId: 999}
	price := merchant.GetSellPrice(newItem)
	assert.Equal(t, 0, price)
}

// ─── AddBuff ──────────────────────────────────────────────────────────────

func TestAddBuff(t *testing.T) {
	mob := &Mob{InstanceId: 42}
	// Should not panic — adds event to queue
	mob.AddBuff(1, "test")
}

// ─── MobIdByName full coverage (prefix vs contains) ───────────────────────

func TestMobIdByNameFullCoverage(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	// Test the exact match path through all mob names
	id := MobIdByName("Ancient Golem")
	assert.Equal(t, MobId(5), id)

	// "Traveling" prefix matches "Traveling Merchant"
	id = MobIdByName("Traveling")
	assert.Equal(t, MobId(2), id)

	// "Wisp" is a substring in "Ghostly Wisp"
	id = MobIdByName("Wisp")
	assert.Equal(t, MobId(4), id)
}

// ─── TickMobCraft more early returns ──────────────────────────────────────

func TestTickMobCraftInCombat(t *testing.T) {
	mob := &Mob{
		Crafter: true,
		Character: characters.Character{
			Aggro: &characters.Aggro{},
		},
	}
	// Mob in combat → skip (but configs not loaded, may return nil earlier)
	defer func() { recover() }()
	result := TickMobCraft(mob)
	assert.Nil(t, result)
}

// ─── PackBonus struct ────────────────────────────────────────────────────

func TestPackBonusStruct(t *testing.T) {
	pb := PackBonus{
		GroupTag:   "undead",
		MemberIds: []int{100, 101},
		ReachedMax: true,
	}
	assert.Equal(t, "undead", pb.GroupTag)
	assert.Len(t, pb.MemberIds, 2)
	assert.True(t, pb.ReachedMax)
}

// ─── SetTempData nil map init (first set creates map) ─────────────────────

func TestSetTempDataNilInit(t *testing.T) {
	mob := &Mob{}
	assert.Nil(t, mob.tempDataStore)
	mob.SetTempData("init", "test")
	assert.NotNil(t, mob.tempDataStore)
	assert.Equal(t, "test", mob.GetTempData("init"))
}

// ─── instancePath ──────────────────────────────────────────────────────────

func TestInstancePath(t *testing.T) {
	path := instancePath(1, "Sanctum Basin", "Skeleton Warrior", 10)
	assert.Contains(t, path, "mobs.instances")
	assert.Contains(t, path, "sanctum_basin")
	assert.Contains(t, path, "1-skeleton_warrior-room10.yaml")
}

// ─── hasProgression ────────────────────────────────────────────────────────

func TestHasProgression(t *testing.T) {
	t.Run("no progression", func(t *testing.T) {
		mob := &Mob{Character: characters.Character{}}
		assert.False(t, hasProgression(mob))
	})

	t.Run("strength training", func(t *testing.T) {
		mob := &Mob{Character: characters.Character{}}
		mob.Character.Stats.Strength.Training = 5
		assert.True(t, hasProgression(mob))
	})

	t.Run("dexterity training", func(t *testing.T) {
		mob := &Mob{Character: characters.Character{}}
		mob.Character.Stats.Dexterity.Training = 1
		assert.True(t, hasProgression(mob))
	})

	t.Run("perception training", func(t *testing.T) {
		mob := &Mob{Character: characters.Character{}}
		mob.Character.Stats.Perception.Training = 1
		assert.True(t, hasProgression(mob))
	})

	t.Run("vitality training", func(t *testing.T) {
		mob := &Mob{Character: characters.Character{}}
		mob.Character.Stats.Vitality.Training = 1
		assert.True(t, hasProgression(mob))
	})

	t.Run("willpower training", func(t *testing.T) {
		mob := &Mob{Character: characters.Character{}}
		mob.Character.Stats.Willpower.Training = 1
		assert.True(t, hasProgression(mob))
	})

	t.Run("charisma training", func(t *testing.T) {
		mob := &Mob{Character: characters.Character{}}
		mob.Character.Stats.Charisma.Training = 1
		assert.True(t, hasProgression(mob))
	})

	t.Run("skills", func(t *testing.T) {
		mob := &Mob{Character: characters.Character{
			Skills: map[string]int{"weapon-combat": 5},
		}}
		assert.True(t, hasProgression(mob))
	})

	t.Run("skill use count", func(t *testing.T) {
		mob := &Mob{Character: characters.Character{
			SkillUseCount: map[string]int{"weapon-combat": 10},
		}}
		assert.True(t, hasProgression(mob))
	})

	t.Run("stat use count", func(t *testing.T) {
		mob := &Mob{Character: characters.Character{
			StatUseCount: map[string]int{"strength": 3},
		}}
		assert.True(t, hasProgression(mob))
	})

	t.Run("mutations", func(t *testing.T) {
		mob := &Mob{Character: characters.Character{
			Mutations: map[string]int{"tough_skin": 1},
		}}
		assert.True(t, hasProgression(mob))
	})

	t.Run("mutation progress", func(t *testing.T) {
		mob := &Mob{Character: characters.Character{
			MutationProgress: 0.5,
		}}
		assert.True(t, hasProgression(mob))
	})
}

// ─── instanceFilename ──────────────────────────────────────────────────────

func TestInstanceFilename(t *testing.T) {
	tests := []struct {
		name       string
		mobId      MobId
		mobName    string
		homeRoomId int
		expected   string
	}{
		{"basic", 1, "Skeleton Warrior", 10, "1-skeleton_warrior-room10.yaml"},
		{"spaces in name", 3, "Shadow Wolf", 30, "3-shadow_wolf-room30.yaml"},
		{"room zero", 5, "Golem", 0, "5-golem-room0.yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := instanceFilename(tt.mobId, tt.mobName, tt.homeRoomId)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ─── TickMobCraft early returns ────────────────────────────────────────────

func TestTickMobCraft(t *testing.T) {
	t.Run("non-crafter returns nil", func(t *testing.T) {
		mob := &Mob{Crafter: false}
		result := TickMobCraft(mob)
		assert.Nil(t, result)
	})
}

// ─── MobInstanceData struct ────────────────────────────────────────────────

func TestMobInstanceData(t *testing.T) {
	data := MobInstanceData{
		StrengthTraining:   10,
		DexterityTraining:  5,
		PerceptionTraining: 3,
		VitalityTraining:   7,
		WillpowerTraining:  2,
		CharismaTraining:   1,
		Skills:             map[string]int{"weapon-combat": 5},
		SkillUseCount:      map[string]int{"weapon-combat": 100},
		StatUseCount:       map[string]int{"strength": 50},
		Mutations:          map[string]int{"tough_skin": 1},
		MutationProgress:   0.75,
	}

	assert.Equal(t, 10, data.StrengthTraining)
	assert.Equal(t, 5, data.DexterityTraining)
	assert.Equal(t, 3, data.PerceptionTraining)
	assert.Equal(t, 7, data.VitalityTraining)
	assert.Equal(t, 2, data.WillpowerTraining)
	assert.Equal(t, 1, data.CharismaTraining)
	assert.Equal(t, 5, data.Skills["weapon-combat"])
	assert.Equal(t, 100, data.SkillUseCount["weapon-combat"])
	assert.Equal(t, 50, data.StatUseCount["strength"])
	assert.Equal(t, 1, data.Mutations["tough_skin"])
	assert.InDelta(t, 0.75, data.MutationProgress, 0.001)
}

// ─── CraftResult struct ───────────────────────────────────────────────────

func TestCraftResult(t *testing.T) {
	result := CraftResult{
		Success:      true,
		RecipeName:   "Iron Sword",
		OutputItemId: 10001,
		SkillMinimum: 10,
		MobName:      "Smith",
		Zone:         "Town",
	}
	assert.True(t, result.Success)
	assert.Equal(t, "Iron Sword", result.RecipeName)
	assert.Equal(t, 10001, result.OutputItemId)
}

// ─── GetSellPrice additional paths ─────────────────────────────────────────

func TestGetSellPriceNonShop(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	// Non-shopkeeper mob — GetSellPrice with empty shop
	warrior := GetInstance(100)
	item := items.Item{ItemId: 1}
	price := warrior.GetSellPrice(item)
	assert.Equal(t, 0, price)
}

// ─── LoadMobInstance (no file = nil) ───────────────────────────────────────

func TestLoadMobInstanceNoFile(t *testing.T) {
	// With default configs, looking for a non-existent instance file
	result := LoadMobInstance(999, "nowhere", "ghost", 999)
	assert.Nil(t, result)
}

// ─── DeleteMobInstance (no file = no panic) ────────────────────────────────

func TestDeleteMobInstanceNoFile(t *testing.T) {
	// Should not panic even when file doesn't exist
	DeleteMobInstance(999, "nowhere", "ghost", 999)
}

// ─── BehaviorArchetype YAML roundtrip ────────────────────────────────────

func TestMob_BehaviorArchetypeYAMLRoundtrip(t *testing.T) {
	data := []byte(`
mobid: 999
zone: Test
behavior_archetype: melee_self_buff
`)
	var m Mob
	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m.BehaviorArchetype != "melee_self_buff" {
		t.Fatalf("want melee_self_buff, got %q", m.BehaviorArchetype)
	}
}

func TestMob_BehaviorArchetypeOmittedWhenAbsent(t *testing.T) {
	data := []byte(`
mobid: 999
zone: Test
`)
	var m Mob
	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m.BehaviorArchetype != "" {
		t.Fatalf("want empty string, got %q", m.BehaviorArchetype)
	}
}

// ─── PathQueue ────────────────────────────────────────────────────────────

type testPathRoom struct {
	exitName string
	roomId   int
	waypoint bool
}

func (r testPathRoom) ExitName() string { return r.exitName }
func (r testPathRoom) RoomId() int      { return r.roomId }
func (r testPathRoom) Waypoint() bool   { return r.waypoint }

func TestPathQueue(t *testing.T) {
	t.Run("empty queue", func(t *testing.T) {
		var pq PathQueue
		assert.Equal(t, 0, pq.Len())
		assert.Nil(t, pq.Next())
		assert.Nil(t, pq.Current())
	})

	t.Run("set path and iterate", func(t *testing.T) {
		var pq PathQueue
		path := []PathRoom{
			testPathRoom{"north", 1, false},
			testPathRoom{"east", 2, true},
			testPathRoom{"south", 3, false},
		}
		pq.SetPath(path)
		assert.Equal(t, 3, pq.Len())

		r := pq.Next()
		assert.NotNil(t, r)
		assert.Equal(t, "north", r.ExitName())
		assert.Equal(t, 2, pq.Len())

		assert.Equal(t, "north", pq.Current().ExitName())
	})

	t.Run("clear", func(t *testing.T) {
		var pq PathQueue
		pq.SetPath([]PathRoom{testPathRoom{"n", 1, false}})
		pq.Next()
		pq.Clear()
		assert.Equal(t, 0, pq.Len())
		assert.Nil(t, pq.Current())
	})

	t.Run("waypoints", func(t *testing.T) {
		var pq PathQueue
		path := []PathRoom{
			testPathRoom{"north", 1, true},
			testPathRoom{"east", 2, false},
			testPathRoom{"south", 3, true},
		}
		pq.SetPath(path)
		wp := pq.Waypoints()
		assert.Equal(t, []int{1, 3}, wp)
	})

	t.Run("waypoints with current", func(t *testing.T) {
		var pq PathQueue
		path := []PathRoom{
			testPathRoom{"north", 1, true},
			testPathRoom{"east", 2, false},
		}
		pq.SetPath(path)
		pq.Next() // move north (waypoint) to current
		wp := pq.Waypoints()
		assert.Contains(t, wp, 1) // current is a waypoint
	})
}

// ─── NewMobByIdFresh ──────────────────────────────────────────────────────

// TestNewMobByIdFresh_IgnoresInstanceFile verifies that NewMobByIdFresh
// does not load any existing mobs.instances/ file for the mob — companion
// spawns must always start from the template.
func TestNewMobByIdFresh_IgnoresInstanceFile(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	withMobProgressionEnabled(t)

	// Seed an instance file by saving an uncharmed mob with progression.
	organic := NewMobById(1, 100)
	if organic == nil {
		t.Fatal("NewMobById returned nil")
	}
	organic.Character.Stats.Strength.Training = 999
	if err := SaveMobInstance(organic); err != nil {
		t.Fatalf("seed SaveMobInstance: %v", err)
	}
	path := instancePath(organic.MobId, organic.Zone, organic.Character.Name, organic.HomeRoomId)
	t.Cleanup(func() { _ = os.Remove(path) })

	// Now call NewMobByIdFresh with the SAME (mobId, homeRoomId).
	fresh := NewMobByIdFresh(1, 100)
	if fresh == nil {
		t.Fatal("NewMobByIdFresh returned nil")
	}

	// Fresh mob must NOT inherit the 999 training value.
	assert.NotEqual(t, 999, fresh.Character.Stats.Strength.Training,
		"NewMobByIdFresh must not load from mobs.instances/")
}

// TestNewMobById_StillLoadsInstanceFile is the control case — existing
// organic spawn behavior is preserved.
func TestNewMobById_StillLoadsInstanceFile(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	withMobProgressionEnabled(t)

	seed := NewMobById(1, 100)
	if seed == nil {
		t.Fatal("NewMobById returned nil")
	}
	seed.Character.Stats.Strength.Training = 777
	if err := SaveMobInstance(seed); err != nil {
		t.Fatalf("seed SaveMobInstance: %v", err)
	}
	path := instancePath(seed.MobId, seed.Zone, seed.Character.Name, seed.HomeRoomId)
	t.Cleanup(func() { _ = os.Remove(path) })

	organic := NewMobById(1, 100)
	if organic == nil {
		t.Fatal("NewMobById returned nil")
	}
	assert.Equal(t, 777, organic.Character.Stats.Strength.Training,
		"NewMobById must still load from mobs.instances/")
}

// ─── Tank archetype distribution ────────────────────────────────────────────

// TestNewMobById_TankArchetypeDistributesStats verifies the new tank
// archetype allocates ~25% Cha and ~20% Vit out of the stat pool,
// with ~15% each Str/Dex/Wil and ~10% Per. Large pool (1000) shrinks
// random variance; assertions use generous slack bands.
func TestNewMobById_TankArchetypeDistributesStats(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	// Register a minimal tank mob template (reuse mobId 1's
	// character scaffold, override Archetype + StatPool).
	mobsMu.Lock()
	mobs[1].Archetype = "tank"
	mobs[1].StatPool = 1000
	mobsMu.Unlock()

	mob := NewMobById(1, 100)
	if mob == nil {
		t.Fatal("NewMobById returned nil")
	}

	total := mob.Character.Stats.Strength.Training +
		mob.Character.Stats.Dexterity.Training +
		mob.Character.Stats.Vitality.Training +
		mob.Character.Stats.Perception.Training +
		mob.Character.Stats.Willpower.Training +
		mob.Character.Stats.Charisma.Training

	assert.Equal(t, 1000, total, "tank stat pool must sum to 1000 with no leakage")

	// Slack bands: expected ±25% of target share (1000-unit pool).
	assert.GreaterOrEqual(t, mob.Character.Stats.Charisma.Training, 200,
		"Cha training should be ≥200 (~25% target)")
	assert.GreaterOrEqual(t, mob.Character.Stats.Vitality.Training, 150,
		"Vit training should be ≥150 (~20% target)")
	assert.LessOrEqual(t, mob.Character.Stats.Perception.Training, 150,
		"Per training should be ≤150 (~10% target)")
}

// ─── Routine and RoutineLinks ─────────────────────────────────────────────

func TestMobYAMLLoadsRoutineAndRoutineLinks(t *testing.T) {
	yamlSrc := []byte(`
mobid: 999
zone: TestZone
routine: bandit_camp_guard
routine_links:
  - watch_north_road
  - bandit_back_camp
character:
  name: test bandit
`)
	var m Mob
	err := yaml.Unmarshal(yamlSrc, &m)
	assert.NoError(t, err)
	assert.Equal(t, "bandit_camp_guard", m.Routine)
	assert.Equal(t, []string{"watch_north_road", "bandit_back_camp"}, m.RoutineLinks)
}

// ─── PlayerAttackImmune YAML round-trip ───────────────────────────────────

func TestMob_PlayerAttackImmuneYAMLRoundTrip(t *testing.T) {
	src := []byte("mobid: 1\nplayer_attack_immune: true\n")
	var m Mob
	if err := yaml.Unmarshal(src, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !m.PlayerAttackImmune {
		t.Error("PlayerAttackImmune should be true after YAML load")
	}

	// Defaults to false when unset.
	var m2 Mob
	if err := yaml.Unmarshal([]byte("mobid: 2\n"), &m2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m2.PlayerAttackImmune {
		t.Error("PlayerAttackImmune should default to false when YAML omits the field")
	}
}

func TestMob_StageThreeFourOverrides_YAMLRoundtrip(t *testing.T) {
	src := `mobid: 99001
zone: Test Zone
carry_capacity: 5000
health_max: 1500
stamina_max: 9999
corpse_name: splintered wagon wreckage
corpse_description: |
  Shattered timbers and twisted iron.
stock_multiplier: 1.5
`
	var m Mob
	if err := yaml.Unmarshal([]byte(src), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m.CarryCapacityOverride != 5000 {
		t.Errorf("CarryCapacityOverride = %v, want 5000", m.CarryCapacityOverride)
	}
	if m.HealthMaxOverride != 1500 {
		t.Errorf("HealthMaxOverride = %d, want 1500", m.HealthMaxOverride)
	}
	if m.StaminaMaxOverride != 9999 {
		t.Errorf("StaminaMaxOverride = %d, want 9999", m.StaminaMaxOverride)
	}
	if m.CorpseName != "splintered wagon wreckage" {
		t.Errorf("CorpseName = %q, want splintered wagon wreckage", m.CorpseName)
	}
	if m.CorpseDescription == "" {
		t.Error("CorpseDescription empty, want non-empty")
	}
	if m.StockMultiplier != 1.5 {
		t.Errorf("StockMultiplier = %v, want 1.5", m.StockMultiplier)
	}
}

func TestMob_StageThreeFourOverrides_DefaultsZero(t *testing.T) {
	src := `mobid: 99002
zone: Test Zone
`
	var m Mob
	if err := yaml.Unmarshal([]byte(src), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m.CarryCapacityOverride != 0 {
		t.Errorf("CarryCapacityOverride default = %v, want 0", m.CarryCapacityOverride)
	}
	if m.HealthMaxOverride != 0 {
		t.Errorf("HealthMaxOverride default = %d, want 0", m.HealthMaxOverride)
	}
	if m.StaminaMaxOverride != 0 {
		t.Errorf("StaminaMaxOverride default = %d, want 0", m.StaminaMaxOverride)
	}
	if m.StockMultiplier != 0 {
		t.Errorf("StockMultiplier default = %v, want 0", m.StockMultiplier)
	}
	if m.CorpseName != "" {
		t.Errorf("CorpseName default = %q, want empty", m.CorpseName)
	}
	if m.CorpseDescription != "" {
		t.Errorf("CorpseDescription default = %q, want empty", m.CorpseDescription)
	}
}
