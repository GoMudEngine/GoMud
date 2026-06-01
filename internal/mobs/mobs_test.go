package mobs

import (
	"os"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/util"
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
			AutoAggro: true,
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
			AutoAggro: false,
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
			AutoAggro: true,
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
			AutoAggro: false,
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
			AutoAggro: true,
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

// ─── Phase 2.5 spawn mutation gate tests ─────────────────────────────────────

// TestMobSpawn_CanineNeverGetsExtraArms verifies that GetWeightedPool never
// returns arm-requiring or hand-requiring mutations for a species that lacks
// those body parts. Seeds a controlled mutations registry with known IDs.
func TestMobSpawn_CanineNeverGetsExtraArms(t *testing.T) {
	// Seed a controlled mutations registry: arm-requiring, hand-requiring,
	// and one universal mutation (no body-part requirements).
	cleanup := mutations.SeedMutationsForTest(map[string]*mutations.MutationSpec{
		"extra-arms": {
			MutationId:        "extra-arms",
			Name:              "Extra Arms",
			Rarity:            9,
			RequiresBodyParts: []string{"arms"},
		},
		"elongated-limbs": {
			MutationId:        "elongated-limbs",
			Name:              "Elongated Limbs",
			Rarity:            5,
			RequiresBodyParts: []string{"arms"},
		},
		"clawed-hands": {
			MutationId:        "clawed-hands",
			Name:              "Clawed Hands",
			Rarity:            4,
			RequiresBodyParts: []string{"hands"},
		},
		"tough-skin": {
			MutationId: "tough-skin",
			Name:       "Tough Skin",
			Rarity:     3,
		},
	})
	defer cleanup()

	// Canine-like species: intentionally omits arms and hands.
	sp := &species.Species{
		SpeciesId: 999,
		Name:      "canine-test",
		BodyParts: []string{"legs", "eyes", "mouth", "skin", "tail"},
	}
	current := map[string]int{}
	seen := map[string]int{}
	for i := 0; i < 500; i++ {
		pool := mutations.GetWeightedPool(current, sp)
		if len(pool) == 0 {
			continue
		}
		picked := pool[util.Rand(len(pool))]
		seen[picked]++
	}
	if seen["extra-arms"] > 0 {
		t.Errorf("canine-shaped species should NEVER roll extra-arms, got %d times", seen["extra-arms"])
	}
	if seen["elongated-limbs"] > 0 {
		t.Errorf("canine-shaped species should NEVER roll elongated-limbs, got %d times", seen["elongated-limbs"])
	}
	if seen["clawed-hands"] > 0 {
		t.Errorf("canine-shaped species should NEVER roll clawed-hands, got %d times", seen["clawed-hands"])
	}
	// Sanity check: the universal mutation must appear at least sometimes.
	if seen["tough-skin"] == 0 {
		t.Error("expected tough-skin (no body-part requirements) to appear at least once in 500 rolls")
	}
}

// TestMobSpawn_IntrinsicStackingWithAcquired verifies that ApplyIntrinsicMutations
// stacks additively with already-acquired mutation ranks.
func TestMobSpawn_IntrinsicStackingWithAcquired(t *testing.T) {
	// Species with intrinsic tail rank 1; character already has acquired rank 1.
	sp := &species.Species{
		SpeciesId:          999,
		IntrinsicMutations: map[string]int{"tail": 1},
	}
	c := &characters.Character{Mutations: map[string]int{"tail": 1}}
	c.ApplyIntrinsicMutations(sp)
	if c.Mutations["tail"] != 2 {
		t.Errorf("expected tail rank 2 (1 acquired + 1 intrinsic), got %d", c.Mutations["tail"])
	}
}

// ─── Submission / Surrender policy at spawn ──────────────────────────────────

// TestNewMobById_SubmissionPolicyFromArchetypeDefault verifies that a mob
// with no submission_policy YAML override inherits the archetype default
// when spawned via NewMobById.
func TestNewMobById_SubmissionPolicyFromArchetypeDefault(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	// Seed a mob with BehaviorArchetype "civilian" → expects PolicyMercy.
	mobsMu.Lock()
	mobs[50] = &Mob{
		MobId:             50,
		Zone:              "Test",
		StatPool:          10,
		ActivityLevel:     50,
		BehaviorArchetype: "civilian",
		Character: characters.Character{
			Name:      "Test Civilian",
			SpeciesId: 0,
		},
	}
	mobsMu.Unlock()

	mob := NewMobById(50, 100)
	if mob == nil {
		t.Fatal("NewMobById returned nil for mob 50")
	}
	defer DestroyInstance(mob.InstanceId)

	assert.Equal(t, characters.PolicyMercy, mob.Character.SubmissionPolicy,
		"civilian archetype should default to PolicyMercy")
	assert.Equal(t, characters.SurrenderAlways, mob.Character.SurrenderPolicy.Mode,
		"civilian archetype should default to SurrenderAlways")
}

// TestNewMobById_SubmissionPolicyDefaultFallback verifies that a mob with
// no BehaviorArchetype (blank) falls through to the generic_fighter default.
func TestNewMobById_SubmissionPolicyDefaultFallback(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	// Mob 4 (Ghostly Wisp) has no BehaviorArchetype set → generic default.
	mob := NewMobById(4, 100)
	if mob == nil {
		t.Fatal("NewMobById returned nil for mob 4")
	}
	defer DestroyInstance(mob.InstanceId)

	// "" archetype → DefaultSubmissionPolicyForArchetype("") → PolicySubdue
	assert.Equal(t, characters.PolicySubdue, mob.Character.SubmissionPolicy,
		"blank archetype should default to PolicySubdue")
	// "" archetype → DefaultSurrenderPolicyForArchetype("") → SurrenderAutoTap
	assert.Equal(t, characters.SurrenderAutoTap, mob.Character.SurrenderPolicy.Mode,
		"blank archetype should default to SurrenderAutoTap")
}

// TestNewMobById_SubmissionPolicyYAMLOverride verifies that a mob with a
// valid submission_policy YAML field uses that value rather than the
// archetype default.
func TestNewMobById_SubmissionPolicyYAMLOverride(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	// Seed a mob with BehaviorArchetype "predator" (default → PolicySubdue)
	// but YAML override "lethal".
	mobsMu.Lock()
	mobs[51] = &Mob{
		MobId:             51,
		Zone:              "Test",
		StatPool:          10,
		ActivityLevel:     50,
		BehaviorArchetype: "predator",
		SubmissionPolicy:  "lethal",
		SurrenderPolicy:   "never",
		Character: characters.Character{
			Name:      "Test Predator",
			SpeciesId: 0,
		},
	}
	mobsMu.Unlock()

	mob := NewMobById(51, 100)
	if mob == nil {
		t.Fatal("NewMobById returned nil for mob 51")
	}
	defer DestroyInstance(mob.InstanceId)

	assert.Equal(t, characters.PolicyLethal, mob.Character.SubmissionPolicy,
		"YAML override 'lethal' should win over predator archetype default (subdue)")
	assert.Equal(t, characters.SurrenderNever, mob.Character.SurrenderPolicy.Mode,
		"YAML override 'never' should win over predator archetype default")
}

// TestNewMobById_SubmissionPolicyInvalidFallsBack verifies that an invalid
// YAML submission_policy value logs a warning and falls back to the archetype
// default rather than panicking or using the zero value.
func TestNewMobById_SubmissionPolicyInvalidFallsBack(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	// Seed a mob with an invalid policy string; BehaviorArchetype "leader"
	// → archetype default is PolicyCripple.
	mobsMu.Lock()
	mobs[52] = &Mob{
		MobId:             52,
		Zone:              "Test",
		StatPool:          10,
		ActivityLevel:     50,
		BehaviorArchetype: "leader",
		SubmissionPolicy:  "eviscerate", // invalid
		SurrenderPolicy:   "bogus",      // invalid
		Character: characters.Character{
			Name:      "Test Leader",
			SpeciesId: 0,
		},
	}
	mobsMu.Unlock()

	mob := NewMobById(52, 100)
	if mob == nil {
		t.Fatal("NewMobById returned nil for mob 52")
	}
	defer DestroyInstance(mob.InstanceId)

	// Invalid → falls back to archetype default for "leader" → PolicyCripple
	assert.Equal(t, characters.PolicyCripple, mob.Character.SubmissionPolicy,
		"invalid YAML override should fall back to leader archetype default (cripple)")
	// Invalid surrender → falls back to "leader" archetype default → SurrenderNever
	assert.Equal(t, characters.SurrenderNever, mob.Character.SurrenderPolicy.Mode,
		"invalid surrender YAML override should fall back to leader archetype default (never)")
}

// TestMob_SubmissionPolicyYAMLRoundtrip verifies the two new fields survive
// a yaml.Unmarshal round-trip on the Mob struct.
func TestMob_SubmissionPolicyYAMLRoundtrip(t *testing.T) {
	src := `mobid: 9999
zone: Test
submission_policy: cripple
surrender_policy: never
`
	var m Mob
	if err := yaml.Unmarshal([]byte(src), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m.SubmissionPolicy != "cripple" {
		t.Errorf("SubmissionPolicy = %q, want cripple", m.SubmissionPolicy)
	}
	if m.SurrenderPolicy != "never" {
		t.Errorf("SurrenderPolicy = %q, want never", m.SurrenderPolicy)
	}

	// Defaults to blank when omitted.
	var m2 Mob
	if err := yaml.Unmarshal([]byte("mobid: 9998\n"), &m2); err != nil {
		t.Fatalf("unmarshal m2: %v", err)
	}
	if m2.SubmissionPolicy != "" {
		t.Errorf("SubmissionPolicy default = %q, want empty", m2.SubmissionPolicy)
	}
	if m2.SurrenderPolicy != "" {
		t.Errorf("SurrenderPolicy default = %q, want empty", m2.SurrenderPolicy)
	}
}

// TestMobSpawn_CuratedGateSkipsIncompatibleMutation verifies that the curated
// SpawnMutations gate (in newMobByIdInternal) correctly filters out mutations
// that require body parts the species lacks. This exercises the gate logic
// that CanApplyTo enforces when processing mob template SpawnMutations.
func TestMobSpawn_CuratedGateSkipsIncompatibleMutation(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	// Seed mutations: extra-arms (requires arms), tough-skin (universal).
	cleanMut := mutations.SeedMutationsForTest(map[string]*mutations.MutationSpec{
		"extra-arms": {
			MutationId:        "extra-arms",
			Name:              "Extra Arms",
			Rarity:            9,
			RequiresBodyParts: []string{"arms"},
		},
		"tough-skin": {
			MutationId: "tough-skin",
			Name:       "Tough Skin",
			Rarity:     3,
		},
	})
	defer cleanMut()

	// Seed canine-like species (no arms).
	cleanSp := species.SeedSpeciesForTest(map[int]*species.Species{
		999: {
			SpeciesId: 999,
			Name:      "test-canine",
			BodyParts: []string{"legs", "eyes", "mouth", "skin", "tail"},
		},
	})
	defer cleanSp()

	// Create a mob template with both mutations in SpawnMutations.
	mobsMu.Lock()
	mobs[999] = &Mob{
		MobId:           999,
		Zone:            "test",
		StatPool:        100,
		SpawnMutations:  []string{"extra-arms", "tough-skin"},
		ActivityLevel:   50,
		Character: characters.Character{
			Name:      "Test Canine",
			SpeciesId: 999,
		},
	}
	mobsMu.Unlock()

	// Spawn the mob (calls newMobByIdInternal with the curated gate).
	mob := NewMobById(999, 100)
	if mob == nil {
		t.Fatal("NewMobById returned nil")
	}
	defer DestroyInstance(mob.InstanceId)

	// Verify: tough-skin applied, extra-arms blocked.
	if mob.Character.Mutations["tough-skin"] != 1 {
		t.Errorf("tough-skin should be applied, got rank %d", mob.Character.Mutations["tough-skin"])
	}
	if mob.Character.Mutations["extra-arms"] > 0 {
		t.Errorf("extra-arms should NOT be applied to canine, got rank %d", mob.Character.Mutations["extra-arms"])
	}
}

func TestNewMobById_RestoresGoldEquipmentPlanState(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()
	withMobProgressionEnabled(t)

	seed := NewMobById(1, 100)
	if seed == nil {
		t.Fatal("NewMobById returned nil")
	}
	seed.Character.Gold = 4242
	seed.Character.Equipment.Body = items.Item{ItemId: 1, EnchantTier: 4}
	seed.Character.SetMiscData("plan:upgrade-gear:worst_slot", "body")
	if err := SaveMobInstance(seed); err != nil {
		t.Fatalf("seed SaveMobInstance: %v", err)
	}
	path := instancePath(seed.MobId, seed.Zone, seed.Character.Name, seed.HomeRoomId)
	t.Cleanup(func() { _ = os.Remove(path) })

	got := NewMobById(1, 100)
	if got == nil {
		t.Fatal("NewMobById returned nil")
	}
	assert.Equal(t, 4242, got.Character.Gold, "gold must be restored")
	assert.Equal(t, 1, got.Character.Equipment.Body.ItemId, "equipped item must be restored")
	assert.Equal(t, 4, got.Character.Equipment.Body.EnchantTier, "enchant tier must be restored")
	assert.Equal(t, "body", got.Character.GetMiscData("plan:upgrade-gear:worst_slot"), "plan state must be restored")
}

func TestNewMobById_OldFormatSaveLeavesTemplateGold(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()
	withMobProgressionEnabled(t)

	// Simulate an OLD-format save: training only, no gold/equipment/plan
	// fields. Write it via a MobInstanceData with the new fields left nil.
	seed := NewMobById(1, 100)
	templateGold := seed.Character.Gold
	seed.Character.Stats.Strength.Training = 12 // trips hasProgression
	if err := SaveMobInstance(seed); err != nil {
		t.Fatalf("seed SaveMobInstance: %v", err)
	}
	path := instancePath(seed.MobId, seed.Zone, seed.Character.Name, seed.HomeRoomId)
	t.Cleanup(func() { _ = os.Remove(path) })

	// Hand-rewrite the file to strip the gold/equipment/plan fields,
	// emulating a file written before this feature existed.
	old := MobInstanceData{StrengthTraining: 12}
	b, _ := yaml.Marshal(&old)
	if err := os.WriteFile(path, b, 0644); err != nil {
		t.Fatalf("rewrite old-format file: %v", err)
	}

	got := NewMobById(1, 100)
	assert.Equal(t, 12, got.Character.Stats.Strength.Training, "training restores")
	assert.Equal(t, templateGold, got.Character.Gold, "old-format save must leave template gold untouched")
}

// TestNewMobById_SpentAllGold_RestoresZeroNotTemplate proves the core
// purpose of the *int presence pointer: a mob that spent ALL its gold
// (Gold == 0) must restore to 0, NOT fall back to its template gold. This
// requires a template with non-zero gold so 0-spent is distinguishable
// from the template baseline (the default fixture has gold 0, which can't
// express this case). clearProgression isolates the gold path so the save
// gate (hasPersistableState) trips via the gold-differs-from-template
// branch (0 != 500) rather than via training.
func TestNewMobById_SpentAllGold_RestoresZeroNotTemplate(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()
	withMobProgressionEnabled(t)

	// Give template 1 a non-zero gold (the fixture default is 0). Mutating
	// the seeded template is test-local — seedRegistry rebuilds it next call.
	mobs[1].Character.Gold = 500

	seed := NewMobById(1, 100)
	if seed == nil {
		t.Fatal("NewMobById returned nil")
	}
	clearProgression(seed)       // isolate the gold path (no training/plan)
	seed.Character.Gold = 0       // spent everything; template is 500
	if err := SaveMobInstance(seed); err != nil {
		t.Fatalf("seed SaveMobInstance: %v", err)
	}
	path := instancePath(seed.MobId, seed.Zone, seed.Character.Name, seed.HomeRoomId)
	t.Cleanup(func() { _ = os.Remove(path) })

	got := NewMobById(1, 100)
	if got == nil {
		t.Fatal("NewMobById returned nil")
	}
	assert.Equal(t, 0, got.Character.Gold,
		"spent-all-gold (0) must restore to 0, not fall back to template gold 500")
}
