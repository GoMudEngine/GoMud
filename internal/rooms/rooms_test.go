package rooms

import (
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/gamelock"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
)

// seedRegistry populates the global roomManager with test rooms and zones.
// Returns a cleanup function that restores the original manager.
func seedRegistry() func() {
	orig := roomManager
	roomManager = &RoomManager{
		rooms:             make(map[int]*Room),
		zones:             make(map[string]*ZoneConfig),
		roomsWithUsers:    make(map[int]int),
		roomsWithMobs:     make(map[int]int),
		roomIdToFileCache: make(map[int]string),
	}

	// Zone config
	zone := &ZoneConfig{
		Name:    "TestZone",
		RoomId:  1,
		RoomIds: map[int]struct{}{1: {}, 2: {}, 3: {}, 4: {}, 5: {}},
	}
	roomManager.zones["TestZone"] = zone

	// Room 1: hub with multiple exits
	roomManager.rooms[1] = &Room{
		RoomId:      1,
		Zone:        "TestZone",
		Title:       "Town Square",
		Description: "A bustling town square.",
		Exits: map[string]exit.RoomExit{
			"north": {RoomId: 2},
			"south": {RoomId: 3},
			"east":  {RoomId: 4, Lock: gamelock.Lock{Difficulty: 10}},
			"west":  {RoomId: 5, Secret: true},
		},
		Containers: map[string]Container{
			"chest": {
				Lock: gamelock.Lock{Difficulty: 5},
				Items: []items.Item{
					{ItemId: 100},
					{ItemId: 100},
					{ItemId: 200},
				},
			},
			"barrel": {},
		},
		Pvp:       true,
		visitors:  make(map[VisitorType]map[int]uint64),
		players:   []int{},
		Signs:     []Sign{},
		Biome:     "city",
	}

	// Room 2: simple room
	roomManager.rooms[2] = &Room{
		RoomId:      2,
		Zone:        "TestZone",
		Title:       "North Road",
		Description: "A road heading north.",
		Exits: map[string]exit.RoomExit{
			"south": {RoomId: 1},
		},
		visitors: make(map[VisitorType]map[int]uint64),
		players:  []int{},
	}

	// Room 3: no exits
	roomManager.rooms[3] = &Room{
		RoomId:      3,
		Zone:        "TestZone",
		Title:       "Dead End",
		Description: "A dead end.",
		Exits:       map[string]exit.RoomExit{},
		visitors:    make(map[VisitorType]map[int]uint64),
		players:     []int{},
	}

	// Room 4: locked destination
	roomManager.rooms[4] = &Room{
		RoomId:      4,
		Zone:        "TestZone",
		Title:       "Vault",
		Description: "A secure vault.",
		Exits: map[string]exit.RoomExit{
			"west": {RoomId: 1},
		},
		visitors: make(map[VisitorType]map[int]uint64),
		players:  []int{},
	}

	// Room 5: secret room, PvP off
	roomManager.rooms[5] = &Room{
		RoomId:      5,
		Zone:        "TestZone",
		Title:       "Hidden Garden",
		Description: "A peaceful hidden garden.",
		Exits: map[string]exit.RoomExit{
			"east": {RoomId: 1},
		},
		Pvp:      false,
		visitors: make(map[VisitorType]map[int]uint64),
		players:  []int{},
	}

	return func() { roomManager = orig }
}

// ─── SetExitLock ────────────────────────────────────────────────────────────

func TestRoom_SetExitLock(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	r := roomManager.rooms[1]

	t.Run("exit has lock", func(t *testing.T) {
		assert.True(t, r.Exits["east"].HasLock())
	})

	t.Run("lock then re-lock does not panic", func(t *testing.T) {
		r.SetExitLock("east", true)
		// After SetLocked(), UnlockedRound is 0 → IsLocked() true
		assert.True(t, r.Exits["east"].Lock.IsLocked())
	})

	t.Run("no-op on exit without lock", func(t *testing.T) {
		// North has no lock; SetExitLock should not panic
		r.SetExitLock("north", true)
		assert.False(t, r.Exits["north"].HasLock())
	})
}

// ─── Exit helpers (no keywords dependency) ──────────────────────────────────

func TestRoom_ExitExists(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	r := roomManager.rooms[1]

	t.Run("exit exists", func(t *testing.T) {
		_, ok := r.Exits["north"]
		assert.True(t, ok)
	})

	t.Run("exit does not exist", func(t *testing.T) {
		_, ok := r.Exits["nonexistent"]
		assert.False(t, ok)
	})
}

// ─── FindExitTo ─────────────────────────────────────────────────────────────

func TestRoom_FindExitTo(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	r := roomManager.rooms[1]

	t.Run("finds exit to room", func(t *testing.T) {
		exitName := r.FindExitTo(2)
		assert.Equal(t, "north", exitName)
	})

	t.Run("not found", func(t *testing.T) {
		exitName := r.FindExitTo(999)
		assert.Equal(t, "", exitName)
	})
}

// ─── GetRandomExit ──────────────────────────────────────────────────────────

func TestRoom_GetRandomExit(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("returns valid exit from room with exits", func(t *testing.T) {
		r := roomManager.rooms[2] // has "south" exit
		name, roomId := r.GetRandomExit()
		assert.Equal(t, "south", name)
		assert.Equal(t, 1, roomId)
	})

	t.Run("empty room returns empty", func(t *testing.T) {
		r := roomManager.rooms[3] // no exits
		name, roomId := r.GetRandomExit()
		assert.Equal(t, "", name)
		assert.Equal(t, 0, roomId)
	})
}

// ─── DataStore ──────────────────────────────────────────────────────────────

func TestRoom_DataStore(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	r := roomManager.rooms[1]

	t.Run("LongTermData round-trip", func(t *testing.T) {
		r.SetLongTermData("quest-flag", "complete")
		val := r.GetLongTermData("quest-flag")
		assert.Equal(t, "complete", val)
	})

	t.Run("LongTermData missing key", func(t *testing.T) {
		val := r.GetLongTermData("nonexistent")
		assert.Nil(t, val)
	})

	t.Run("LongTermData delete by nil", func(t *testing.T) {
		r.SetLongTermData("quest-flag", nil)
		val := r.GetLongTermData("quest-flag")
		assert.Nil(t, val)
	})

	t.Run("TempData round-trip", func(t *testing.T) {
		r.SetTempData("timer", 42)
		val := r.GetTempData("timer")
		assert.Equal(t, 42, val)
	})

	t.Run("TempData missing key", func(t *testing.T) {
		val := r.GetTempData("missing")
		assert.Nil(t, val)
	})

	t.Run("TempData delete by nil", func(t *testing.T) {
		r.SetTempData("timer", nil)
		val := r.GetTempData("timer")
		assert.Nil(t, val)
	})
}

// ─── IsPvp ──────────────────────────────────────────────────────────────────

func TestRoom_IsPvp(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("pvp room", func(t *testing.T) {
		assert.True(t, roomManager.rooms[1].IsPvp())
	})

	t.Run("non-pvp room", func(t *testing.T) {
		assert.False(t, roomManager.rooms[5].IsPvp())
	})
}

// ─── Signs ──────────────────────────────────────────────────────────────────

func TestRoom_Signs(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	r := roomManager.rooms[1]

	t.Run("add public sign", func(t *testing.T) {
		replaced := r.AddSign("Welcome!", 0, 7)
		assert.False(t, replaced, "first public sign should not replace")
	})

	t.Run("add private sign", func(t *testing.T) {
		replaced := r.AddSign("My rune", 42, 7)
		assert.False(t, replaced)
	})

	t.Run("replace existing public sign", func(t *testing.T) {
		replaced := r.AddSign("New welcome!", 0, 7)
		assert.True(t, replaced, "second public sign should replace")
	})

	t.Run("GetPublicSigns", func(t *testing.T) {
		public := r.GetPublicSigns()
		assert.Len(t, public, 1)
		assert.Equal(t, "New welcome!", public[0].DisplayText)
	})

	t.Run("GetPrivateSigns", func(t *testing.T) {
		private := r.GetPrivateSigns()
		assert.Len(t, private, 1)
		assert.Equal(t, 42, private[0].VisibleUserId)
	})
}

// ─── Container HasLock ──────────────────────────────────────────────────────

func TestContainer_HasLock(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	r := roomManager.rooms[1]

	t.Run("chest has lock", func(t *testing.T) {
		chest := r.Containers["chest"]
		assert.True(t, chest.HasLock())
	})

	t.Run("barrel has no lock", func(t *testing.T) {
		barrel := r.Containers["barrel"]
		assert.False(t, barrel.HasLock())
	})
}

// ─── Container Count ────────────────────────────────────────────────────────

func TestContainer_Count(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	r := roomManager.rooms[1]
	chest := r.Containers["chest"]

	t.Run("count existing item", func(t *testing.T) {
		assert.Equal(t, 2, chest.Count(100))
	})

	t.Run("count single item", func(t *testing.T) {
		assert.Equal(t, 1, chest.Count(200))
	})

	t.Run("count missing item", func(t *testing.T) {
		assert.Equal(t, 0, chest.Count(999))
	})
}

// ─── MarkVisited / HasVisited ───────────────────────────────────────────────

func TestRoom_MarkVisited_HasVisited(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	r := roomManager.rooms[2]

	t.Run("not visited initially", func(t *testing.T) {
		assert.False(t, r.HasVisited(42, VisitorUser))
	})

	t.Run("mark and check visited", func(t *testing.T) {
		r.MarkVisited(42, VisitorUser)
		assert.True(t, r.HasVisited(42, VisitorUser))
	})

	t.Run("different visitor type", func(t *testing.T) {
		assert.False(t, r.HasVisited(42, VisitorMob))
		r.MarkVisited(42, VisitorMob)
		assert.True(t, r.HasVisited(42, VisitorMob))
	})
}

// ─── GetRoomCount ───────────────────────────────────────────────────────────

func TestGetRoomCount(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("existing zone", func(t *testing.T) {
		count := GetRoomCount("TestZone")
		assert.Equal(t, 5, count)
	})

	t.Run("nonexistent zone", func(t *testing.T) {
		count := GetRoomCount("FakeZone")
		assert.Equal(t, 0, count)
	})
}

// ─── IsRoomLoaded ───────────────────────────────────────────────────────────

func TestIsRoomLoaded(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("loaded room", func(t *testing.T) {
		assert.True(t, IsRoomLoaded(1))
	})

	t.Run("unloaded room", func(t *testing.T) {
		assert.False(t, IsRoomLoaded(999))
	})
}

// ─── AddPlayer / RemovePlayer / GetPlayers ──────────────────────────────────

func TestRoom_AddRemovePlayer(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	r := roomManager.rooms[1]

	t.Run("add player", func(t *testing.T) {
		ct := r.AddPlayer(42)
		assert.Equal(t, 1, ct)
	})

	t.Run("add duplicate player is no-op", func(t *testing.T) {
		ct := r.AddPlayer(42)
		assert.Equal(t, 1, ct)
	})

	t.Run("add second player", func(t *testing.T) {
		ct := r.AddPlayer(99)
		assert.Equal(t, 2, ct)
	})

	t.Run("remove player", func(t *testing.T) {
		ct, found := r.RemovePlayer(42)
		assert.True(t, found)
		assert.Equal(t, 1, ct)
	})

	t.Run("remove nonexistent player", func(t *testing.T) {
		ct, found := r.RemovePlayer(42)
		assert.False(t, found)
		assert.Equal(t, 1, ct)
	})
}

// ─── Zone helpers ───────────────────────────────────────────────────────────

func TestGetAllZoneNames(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	names := GetAllZoneNames()
	assert.Len(t, names, 1)
	assert.Contains(t, names, "TestZone")
}

func TestGetAllZoneRoomsIds(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	ids := GetAllZoneRoomsIds("TestZone")
	assert.Len(t, ids, 5)

	ids = GetAllZoneRoomsIds("Nonexistent")
	assert.Empty(t, ids)
}

func TestFindZoneName(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("exact match", func(t *testing.T) {
		name := FindZoneName("TestZone")
		assert.Equal(t, "TestZone", name)
	})

	t.Run("partial match", func(t *testing.T) {
		name := FindZoneName("test")
		assert.Equal(t, "TestZone", name)
	})

	t.Run("no match", func(t *testing.T) {
		name := FindZoneName("zzzzz")
		assert.Equal(t, "", name)
	})
}

// ─── ZoneNameSanitize ───────────────────────────────────────────────────────

func TestZoneNameSanitize(t *testing.T) {
	assert.Equal(t, "test_zone", ZoneNameSanitize("Test Zone"))
	assert.Equal(t, "test_zone", ZoneNameSanitize("test_zone"))
	assert.Equal(t, "", ZoneNameSanitize(""))
}

// ─── ZoneToFolder ───────────────────────────────────────────────────────────

func TestZoneToFolder(t *testing.T) {
	assert.Equal(t, "test_zone/", ZoneToFolder("Test Zone"))
	assert.Equal(t, "arctic/", ZoneToFolder("Arctic"))
}

// ─── ValidateZoneName ───────────────────────────────────────────────────────

func TestValidateZoneName(t *testing.T) {
	t.Run("valid name", func(t *testing.T) {
		assert.NoError(t, ValidateZoneName("Test Zone 1"))
	})

	t.Run("empty name is valid", func(t *testing.T) {
		assert.NoError(t, ValidateZoneName(""))
	})

	t.Run("invalid characters", func(t *testing.T) {
		assert.Error(t, ValidateZoneName("Test!Zone"))
	})
}

// ─── GetDescription ─────────────────────────────────────────────────────────

func TestRoom_GetDescription(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	r := roomManager.rooms[1]
	assert.Equal(t, "A bustling town square.", r.GetDescription())
}

// ─── ZoneStats ──────────────────────────────────────────────────────────────

func TestZoneStats(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("existing zone", func(t *testing.T) {
		rootId, total, err := ZoneStats("TestZone")
		assert.NoError(t, err)
		assert.Equal(t, 1, rootId)
		assert.Equal(t, 5, total)
	})

	t.Run("nonexistent zone", func(t *testing.T) {
		_, _, err := ZoneStats("FakeZone")
		assert.Error(t, err)
	})
}

// ─── GetZoneRoot ────────────────────────────────────────────────────────────

func TestGetZoneRoot(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("existing zone", func(t *testing.T) {
		rootId, err := GetZoneRoot("TestZone")
		assert.NoError(t, err)
		assert.Equal(t, 1, rootId)
	})

	t.Run("nonexistent zone", func(t *testing.T) {
		_, err := GetZoneRoot("FakeZone")
		assert.Error(t, err)
	})
}

// ─── GetZoneBiome ───────────────────────────────────────────────────────────

func TestGetZoneBiome(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("existing zone (no biome set on zone config)", func(t *testing.T) {
		biome := GetZoneBiome("TestZone")
		assert.Equal(t, "", biome)
	})

	t.Run("nonexistent zone", func(t *testing.T) {
		biome := GetZoneBiome("FakeZone")
		assert.Equal(t, "", biome)
	})
}

// ─── GetZoneConfig ──────────────────────────────────────────────────────────

func TestGetZoneConfig(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("existing zone", func(t *testing.T) {
		cfg := GetZoneConfig("TestZone")
		assert.NotNil(t, cfg)
		assert.Equal(t, "TestZone", cfg.Name)
	})

	t.Run("nonexistent zone", func(t *testing.T) {
		cfg := GetZoneConfig("FakeZone")
		assert.Nil(t, cfg)
	})
}

// ─── MobCt / PlayerCt ──────────────────────────────────────────────────────

func TestRoom_MobCtPlayerCt(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	r := roomManager.rooms[1]
	assert.Equal(t, 0, r.MobCt())
	assert.Equal(t, 0, r.PlayerCt())

	r.AddPlayer(10)
	assert.Equal(t, 1, r.PlayerCt())
}

// ─── FindExitTo (deeper) ────────────────────────────────────────────────────

func TestRoom_FindExitTo_AllExits(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	r := roomManager.rooms[1]

	assert.Equal(t, "north", r.FindExitTo(2))
	assert.Equal(t, "south", r.FindExitTo(3))
	assert.Equal(t, "east", r.FindExitTo(4))
	assert.Equal(t, "west", r.FindExitTo(5))
	assert.Equal(t, "", r.FindExitTo(999))
}

// ─── HasRecentVisitors ──────────────────────────────────────────────────────

func TestRoom_HasRecentVisitors(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	r := roomManager.rooms[3]
	assert.False(t, r.HasRecentVisitors())

	r.MarkVisited(1, VisitorUser)
	assert.True(t, r.HasRecentVisitors())
}

// ─── GetExitInfo ────────────────────────────────────────────────────────────

func TestRoom_GetExitInfo(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	r := roomManager.rooms[1]

	t.Run("existing exit", func(t *testing.T) {
		info, ok := r.GetExitInfo("north")
		assert.True(t, ok)
		assert.Equal(t, 2, info.RoomId)
	})

	t.Run("nonexistent exit", func(t *testing.T) {
		_, ok := r.GetExitInfo("down")
		assert.False(t, ok)
	})
}

// ─── FindContainerByName ────────────────────────────────────────────────────

func TestRoom_FindContainerByName(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	r := roomManager.rooms[1]

	t.Run("exact match", func(t *testing.T) {
		name := r.FindContainerByName("chest")
		assert.Equal(t, "chest", name)
	})

	t.Run("partial match", func(t *testing.T) {
		name := r.FindContainerByName("bar")
		assert.Equal(t, "barrel", name)
	})

	t.Run("no match", func(t *testing.T) {
		name := r.FindContainerByName("zzz")
		assert.Equal(t, "", name)
	})
}

// ─── Container AddItem / RemoveItem / FindItemById ──────────────────────────

func TestContainer_AddRemoveFind(t *testing.T) {
	c := Container{
		Items: []items.Item{},
	}

	item1 := items.Item{ItemId: 42, UUID: [16]byte{1}}
	item2 := items.Item{ItemId: 99, UUID: [16]byte{2}}

	c.AddItem(item1)
	c.AddItem(item2)
	assert.Len(t, c.Items, 2)

	found, ok := c.FindItemById(42)
	assert.True(t, ok)
	assert.Equal(t, 42, found.ItemId)

	_, ok = c.FindItemById(999)
	assert.False(t, ok)

	c.RemoveItem(item1)
	assert.Len(t, c.Items, 1)
	assert.Equal(t, 99, c.Items[0].ItemId)
}

// ─── getRoomFromMemory ──────────────────────────────────────────────────────

func TestGetRoomFromMemory(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	r := getRoomFromMemory(1)
	assert.NotNil(t, r)
	assert.Equal(t, "Town Square", r.Title)

	r2 := getRoomFromMemory(999)
	assert.Nil(t, r2)
}

// ─── TemporaryExits ─────────────────────────────────────────────────────────

func TestRoom_TemporaryExits(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	r := roomManager.rooms[1]
	r.ExitsTemp = map[string]exit.TemporaryRoomExit{
		"portal": {RoomId: 100, Title: "portal", UserId: 42},
	}

	t.Run("find by user id", func(t *testing.T) {
		found, ok := r.FindTemporaryExitByUserId(42)
		assert.True(t, ok)
		assert.Equal(t, 100, found.RoomId)
	})

	t.Run("not found", func(t *testing.T) {
		_, ok := r.FindTemporaryExitByUserId(999)
		assert.False(t, ok)
	})

	t.Run("remove temporary exit", func(t *testing.T) {
		removed := r.RemoveTemporaryExit(exit.TemporaryRoomExit{RoomId: 100, Title: "portal", UserId: 42})
		assert.True(t, removed)
		assert.Len(t, r.ExitsTemp, 0)
	})

	t.Run("remove nonexistent", func(t *testing.T) {
		removed := r.RemoveTemporaryExit(exit.TemporaryRoomExit{RoomId: 999, Title: "void", UserId: 1})
		assert.False(t, removed)
	})
}

// ─── ZoneConfig ─────────────────────────────────────────────────────────────

func TestZoneConfig(t *testing.T) {
	z := NewZoneConfig("Test Zone")
	assert.Equal(t, "Test Zone", z.Name)
	assert.NotNil(t, z.RoomIds)

	assert.Equal(t, "Test Zone", z.Id())
	assert.Equal(t, "zone-config.yaml", z.Filename())

	err := z.Validate()
	assert.NoError(t, err)
}

// ─── ClearRoomCache ─────────────────────────────────────────────────────────

func TestClearRoomCache(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("clear existing room", func(t *testing.T) {
		err := ClearRoomCache(2)
		assert.NoError(t, err)
		assert.False(t, IsRoomLoaded(2))
	})

	t.Run("clear nonexistent room", func(t *testing.T) {
		err := ClearRoomCache(999)
		assert.Error(t, err)
	})
}

// ─── Room.Id ────────────────────────────────────────────────────────────────

func TestRoom_Id(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	r := roomManager.rooms[1]
	assert.Equal(t, 1, r.Id())
}

// ─── Room.Validate ──────────────────────────────────────────────────────────

func TestRoom_Validate(t *testing.T) {
	t.Run("valid room", func(t *testing.T) {
		r := &Room{
			RoomId:      99,
			Title:       "Test Room",
			Description: "A test room.",
			Exits:       map[string]exit.RoomExit{},
		}
		err := r.Validate()
		assert.NoError(t, err)
	})

	t.Run("empty title", func(t *testing.T) {
		r := &Room{
			RoomId:      99,
			Title:       "",
			Description: "A test room.",
		}
		err := r.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "title")
	})

	t.Run("empty description", func(t *testing.T) {
		r := &Room{
			RoomId:      99,
			Title:       "Test Room",
			Description: "",
		}
		err := r.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "description")
	})
}

// ─── Room.GetMapSymbol ──────────────────────────────────────────────────────

func TestRoom_GetMapSymbol(t *testing.T) {
	t.Run("override symbol", func(t *testing.T) {
		r := &Room{MapSymbol: "*"}
		assert.Equal(t, defaultMapSymbol, r.GetMapSymbol())
	})

	t.Run("normal symbol", func(t *testing.T) {
		r := &Room{MapSymbol: "X"}
		assert.Equal(t, "X", r.GetMapSymbol())
	})

	t.Run("empty symbol", func(t *testing.T) {
		r := &Room{MapSymbol: ""}
		assert.Equal(t, "", r.GetMapSymbol())
	})
}

// ─── GetRoomsWithMobs ───────────────────────────────────────────────────────

func TestGetRoomsWithMobs(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	// No mobs seeded
	mobs := GetRoomsWithMobs()
	assert.Empty(t, mobs)

	// Add a mob room
	roomManager.roomsWithMobs[1] = 2
	mobs = GetRoomsWithMobs()
	assert.Len(t, mobs, 1)
	assert.Contains(t, mobs, 1)
}

// ─── Container RecipeReady ──────────────────────────────────────────────────

func TestContainer_RecipeReady(t *testing.T) {
	t.Run("no recipes", func(t *testing.T) {
		c := Container{}
		assert.Equal(t, 0, c.RecipeReady())
	})

	t.Run("recipe ready", func(t *testing.T) {
		c := Container{
			Items: []items.Item{
				{ItemId: 10},
				{ItemId: 20},
			},
			Recipes: map[int][]int{
				99: {10, 20}, // makes item 99 from items 10 + 20
			},
		}
		assert.Equal(t, 99, c.RecipeReady())
	})

	t.Run("recipe not ready", func(t *testing.T) {
		c := Container{
			Items: []items.Item{
				{ItemId: 10},
			},
			Recipes: map[int][]int{
				99: {10, 20}, // needs item 20 too
			},
		}
		assert.Equal(t, 0, c.RecipeReady())
	})
}

// ─── Room.RemoveCorpse (deeper) ─────────────────────────────────────────────

func TestRoom_RemoveCorpse_NotFound(t *testing.T) {
	r := &Room{}
	r.AddCorpse(Corpse{
		MobId:        1,
		Character:    characters.Character{Name: "Goblin"},
		RoundCreated: 5,
	})

	removed := r.RemoveCorpse(Corpse{
		MobId:        2,
		Character:    characters.Character{Name: "Goblin"},
		RoundCreated: 5,
	})
	assert.False(t, removed)
	assert.Len(t, r.Corpses, 1)
}

// ─── Room.GetAllFloorItems ──────────────────────────────────────────────────

func TestRoom_GetAllFloorItems(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	r := roomManager.rooms[1]
	r.Items = []items.Item{
		{ItemId: 10},
		{ItemId: 20},
	}
	r.Stash = []items.Item{
		{ItemId: 30, StashedBy: 42},
	}

	floorItems := r.GetAllFloorItems(false)
	assert.Len(t, floorItems, 2)

	allItems := r.GetAllFloorItems(true)
	assert.Len(t, allItems, 3)
}

// ─── Room.AddItem / RemoveItem ──────────────────────────────────────────────

func TestRoom_AddRemoveItem(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	r := roomManager.rooms[3]
	r.Items = []items.Item{}

	itm := items.Item{ItemId: 42, UUID: [16]byte{1}}
	r.AddItem(itm, false)
	assert.Len(t, r.Items, 1)

	r.RemoveItem(itm, false)
	assert.Len(t, r.Items, 0)
}

// ─── Room.AddItem to stash ──────────────────────────────────────────────────

func TestRoom_AddRemoveStash(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	r := roomManager.rooms[3]
	r.Stash = []items.Item{}

	itm := items.Item{ItemId: 42, UUID: [16]byte{1}}
	r.AddItem(itm, true)
	assert.Len(t, r.Stash, 1)

	r.RemoveItem(itm, true)
	assert.Len(t, r.Stash, 0)
}

func TestRoom_AddCorpse(t *testing.T) {
	r := &Room{}
	assert.Empty(t, r.Corpses, "Expected no corpses initially")

	corpse := Corpse{
		MobId:        0,
		UserId:       123,
		Character:    characters.Character{Name: "TestPlayer"},
		RoundCreated: 10,
	}
	r.AddCorpse(corpse)
	assert.Len(t, r.Corpses, 1, "Expected exactly one corpse after adding")
	assert.Equal(t, corpse, r.Corpses[0], "Expected the added corpse to match")
}

func TestRoom_RemoveCorpse(t *testing.T) {
	r := &Room{}
	corpse1 := Corpse{
		MobId:        0,
		UserId:       123,
		Character:    characters.Character{Name: "PlayerOne"},
		RoundCreated: 10,
	}
	corpse2 := Corpse{
		MobId:        456,
		UserId:       0,
		Character:    characters.Character{Name: "MobOne"},
		RoundCreated: 11,
	}

	r.AddCorpse(corpse1)
	r.AddCorpse(corpse2)

	// Removing existing corpse
	removed := r.RemoveCorpse(corpse1)
	assert.True(t, removed, "Expected to remove an existing corpse successfully")
	assert.Len(t, r.Corpses, 1, "Expected exactly one corpse remaining")

	// Try removing a corpse that does not exist
	nonExistent := Corpse{
		MobId:        999,
		UserId:       999,
		Character:    characters.Character{Name: "Ghost"},
		RoundCreated: 99,
	}
	removed = r.RemoveCorpse(nonExistent)
	assert.False(t, removed, "Expected removal to fail for non-existent corpse")
	assert.Len(t, r.Corpses, 1, "Expected no change in corpses")
}

func TestRoom_FindCorpse(t *testing.T) {
	r := &Room{}

	playerCorpse := Corpse{
		UserId:       123,
		Character:    characters.Character{Name: "PlayerOne"},
		RoundCreated: 5,
		Prunable:     false,
	}
	mobCorpse := Corpse{
		MobId:        456,
		Character:    characters.Character{Name: "MobOne"},
		RoundCreated: 6,
		Prunable:     false,
	}
	r.AddCorpse(playerCorpse)
	r.AddCorpse(mobCorpse)

	// Exact search
	found, ok := r.FindCorpse("PlayerOne corpse")
	assert.True(t, ok, "Expected to find player corpse by exact name")
	assert.Equal(t, "PlayerOne", found.Character.Name, "Expected found corpse to match the correct character")

	// Searching for mob
	found, ok = r.FindCorpse("MobOne corpse")
	assert.True(t, ok, "Expected to find mob corpse by exact name")
	assert.Equal(t, "MobOne", found.Character.Name, "Expected found corpse to match the correct character")

	// Searching partial name (depends on your util.FindMatchIn logic)
	found, ok = r.FindCorpse("player")
	assert.True(t, ok, "Expected to find a close match for player corpse")
	assert.Equal(t, "PlayerOne", found.Character.Name, "Expected found corpse to be the player's")

	// Non-existent
	found, ok = r.FindCorpse("NonExistent")
	assert.False(t, ok, "Expected not to find a missing corpse")
}

// ─── BiomeInfo ──────────────────────────────────────────────────────────────

func seedBiomes() func() {
	orig := biomes
	biomes = map[string]*BiomeInfo{
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
		"forest": {
			BiomeId:      "forest",
			Name:         "Forest",
			Symbol:       "T",
			LitArea:      false,
			DarkArea:     false,
			MovementCost: 1.5,
		},
		"default": {
			BiomeId:      "default",
			Name:         "Default",
			Symbol:       "•",
			LitArea:      true,
			MovementCost: 1.0,
		},
	}
	return func() { biomes = orig }
}

func TestBiomeInfo_GetSymbol(t *testing.T) {
	bi := &BiomeInfo{Symbol: "#"}
	assert.Equal(t, '#', bi.GetSymbol())

	bi2 := &BiomeInfo{Symbol: ""}
	assert.Equal(t, rune(0), bi2.GetSymbol())
}

func TestBiomeInfo_SymbolString(t *testing.T) {
	bi := &BiomeInfo{Symbol: "T"}
	assert.Equal(t, "T", bi.SymbolString())
}

// TestRoom_MapSymbolAndLegend_PerRoomOverridesBiome guards the city-biome
// mapsymbol bug: a room's per-room mapsymbol/maplegend must win over the
// biome's glyph/name; biome is only the fallback when the room sets neither.
func TestRoom_MapSymbolAndLegend_PerRoomOverridesBiome(t *testing.T) {
	cleanup := seedBiomes() // city Symbol "#" Name "City"; default "•"
	defer cleanup()

	// City-biome room WITH a per-room override — must keep T / Townsquare.
	r := &Room{Biome: "city", MapSymbol: "T", MapLegend: "Townsquare"}
	sym, legend := r.MapSymbolAndLegend()
	assert.Equal(t, "T", sym, "per-room mapsymbol must win over the city biome glyph")
	assert.Equal(t, "Townsquare", legend, "per-room maplegend must win over the biome name")

	// City-biome room WITHOUT an override — falls back to the biome.
	r2 := &Room{Biome: "city"}
	sym2, legend2 := r2.MapSymbolAndLegend()
	assert.Equal(t, "#", sym2, "no per-room mapsymbol falls back to the biome glyph")
	assert.Equal(t, "City", legend2, "no per-room maplegend falls back to the biome name")
}

func TestBiomeInfo_IsLit(t *testing.T) {
	t.Run("lit area", func(t *testing.T) {
		bi := &BiomeInfo{LitArea: true, DarkArea: false}
		assert.True(t, bi.IsLit())
	})
	t.Run("dark area is not lit", func(t *testing.T) {
		bi := &BiomeInfo{LitArea: false, DarkArea: true}
		assert.False(t, bi.IsLit())
	})
	t.Run("both dark and lit = not lit", func(t *testing.T) {
		bi := &BiomeInfo{LitArea: true, DarkArea: true}
		assert.False(t, bi.IsLit())
	})
	t.Run("neither = not lit", func(t *testing.T) {
		bi := &BiomeInfo{LitArea: false, DarkArea: false}
		assert.False(t, bi.IsLit())
	})
}

func TestBiomeInfo_IsDark(t *testing.T) {
	t.Run("dark area", func(t *testing.T) {
		bi := &BiomeInfo{LitArea: false, DarkArea: true}
		assert.True(t, bi.IsDark())
	})
	t.Run("lit area is not dark", func(t *testing.T) {
		bi := &BiomeInfo{LitArea: true, DarkArea: false}
		assert.False(t, bi.IsDark())
	})
	t.Run("both = not dark", func(t *testing.T) {
		bi := &BiomeInfo{LitArea: true, DarkArea: true}
		assert.False(t, bi.IsDark())
	})
}

func TestBiomeInfo_GetMovementCost(t *testing.T) {
	t.Run("default when zero", func(t *testing.T) {
		bi := &BiomeInfo{}
		assert.Equal(t, 1.0, bi.GetMovementCost())
	})
	t.Run("default when negative", func(t *testing.T) {
		bi := &BiomeInfo{MovementCost: -1.0}
		assert.Equal(t, 1.0, bi.GetMovementCost())
	})
	t.Run("custom cost", func(t *testing.T) {
		bi := &BiomeInfo{MovementCost: 2.5}
		assert.Equal(t, 2.5, bi.GetMovementCost())
	})
}

func TestBiomeInfo_Id(t *testing.T) {
	bi := &BiomeInfo{BiomeId: "Forest"}
	assert.Equal(t, "forest", bi.Id())
}

func TestBiomeInfo_Validate(t *testing.T) {
	t.Run("valid biome", func(t *testing.T) {
		bi := &BiomeInfo{BiomeId: "city", Name: "City", Symbol: "#"}
		assert.NoError(t, bi.Validate())
	})
	t.Run("empty id", func(t *testing.T) {
		bi := &BiomeInfo{Name: "City", Symbol: "#"}
		assert.Error(t, bi.Validate())
	})
	t.Run("empty name", func(t *testing.T) {
		bi := &BiomeInfo{BiomeId: "city", Symbol: "#"}
		assert.Error(t, bi.Validate())
	})
	t.Run("empty symbol", func(t *testing.T) {
		bi := &BiomeInfo{BiomeId: "city", Name: "City"}
		assert.Error(t, bi.Validate())
	})
	t.Run("question mark symbol", func(t *testing.T) {
		bi := &BiomeInfo{BiomeId: "city", Name: "City", Symbol: "?"}
		assert.Error(t, bi.Validate())
	})
	t.Run("both dark and lit", func(t *testing.T) {
		bi := &BiomeInfo{BiomeId: "city", Name: "City", Symbol: "#", DarkArea: true, LitArea: true}
		assert.Error(t, bi.Validate())
	})
}

func TestBiomeInfo_Filepath(t *testing.T) {
	bi := &BiomeInfo{BiomeId: "forest"}
	assert.Equal(t, "forest.yaml", bi.Filepath())
	// Second call should return cached
	assert.Equal(t, "forest.yaml", bi.Filepath())
}

func TestGetBiome(t *testing.T) {
	cleanupBiomes := seedBiomes()
	defer cleanupBiomes()

	t.Run("existing biome", func(t *testing.T) {
		b, ok := GetBiome("city")
		assert.True(t, ok)
		assert.Equal(t, "City", b.Name)
	})

	t.Run("empty name returns default", func(t *testing.T) {
		b, ok := GetBiome("")
		assert.True(t, ok)
		assert.Equal(t, "Default", b.Name)
	})

	t.Run("case insensitive", func(t *testing.T) {
		b, ok := GetBiome("CAVE")
		assert.True(t, ok)
		assert.Equal(t, "Cave", b.Name)
	})

	t.Run("nonexistent", func(t *testing.T) {
		_, ok := GetBiome("volcano")
		assert.False(t, ok)
	})
}

func TestGetAllBiomes(t *testing.T) {
	cleanupBiomes := seedBiomes()
	defer cleanupBiomes()

	all := GetAllBiomes()
	assert.Len(t, all, 4)
}

// ─── PruneSigns ─────────────────────────────────────────────────────────────

func TestRoom_PruneSigns(t *testing.T) {
	t.Run("no signs", func(t *testing.T) {
		r := &Room{}
		pruned := r.PruneSigns()
		assert.Empty(t, pruned)
	})

	t.Run("prunes expired signs", func(t *testing.T) {
		r := &Room{
			Signs: []Sign{
				{DisplayText: "Old sign", Expires: time.Now().Add(-1 * time.Hour)},
				{DisplayText: "Fresh sign", Expires: time.Now().Add(24 * time.Hour)},
				{DisplayText: "Also expired", Expires: time.Now().Add(-10 * time.Minute)},
			},
		}
		pruned := r.PruneSigns()
		assert.Len(t, pruned, 2)
		assert.Len(t, r.Signs, 1)
		assert.Equal(t, "Fresh sign", r.Signs[0].DisplayText)
	})

	t.Run("all fresh", func(t *testing.T) {
		r := &Room{
			Signs: []Sign{
				{DisplayText: "Sign 1", Expires: time.Now().Add(1 * time.Hour)},
				{DisplayText: "Sign 2", Expires: time.Now().Add(2 * time.Hour)},
			},
		}
		pruned := r.PruneSigns()
		assert.Empty(t, pruned)
		assert.Len(t, r.Signs, 2)
	})
}

// ─── Room.Filename / Filepath ───────────────────────────────────────────────

func TestRoom_Filename(t *testing.T) {
	r := &Room{RoomId: 42, Zone: "Test Zone"}
	assert.Equal(t, "42.yaml", r.Filename())
}

func TestRoom_Filepath(t *testing.T) {
	r := &Room{RoomId: 42, Zone: "Test Zone"}
	fp := r.Filepath()
	assert.Contains(t, fp, "test_zone")
	assert.Contains(t, fp, "42.yaml")
}

// ─── IsEphemeral ────────────────────────────────────────────────────────────

func TestRoom_IsEphemeral(t *testing.T) {
	t.Run("normal room", func(t *testing.T) {
		r := &Room{RoomId: 42}
		assert.False(t, r.IsEphemeral())
	})

	t.Run("ephemeral room", func(t *testing.T) {
		r := &Room{RoomId: ephemeralRoomIdMinimum + 1}
		assert.True(t, r.IsEphemeral())
	})
}

// ─── IsEphemeralRoomId ──────────────────────────────────────────────────────

func TestIsEphemeralRoomId(t *testing.T) {
	assert.False(t, IsEphemeralRoomId(42))
	assert.False(t, IsEphemeralRoomId(999999))
	assert.True(t, IsEphemeralRoomId(ephemeralRoomIdMinimum))
	assert.True(t, IsEphemeralRoomId(ephemeralRoomIdMinimum+100))
}

// ─── GetOriginalRoom ────────────────────────────────────────────────────────

func TestGetOriginalRoom(t *testing.T) {
	t.Run("non-ephemeral returns self", func(t *testing.T) {
		assert.Equal(t, 42, GetOriginalRoom(42))
	})

	t.Run("ephemeral returns lookup or self", func(t *testing.T) {
		origLookups := originalRoomIdLookups
		defer func() { originalRoomIdLookups = origLookups }()

		ephId := ephemeralRoomIdMinimum + 5
		originalRoomIdLookups[ephId] = 10
		assert.Equal(t, 10, GetOriginalRoom(ephId))
	})
}

// ─── GetChunkCount ──────────────────────────────────────────────────────────

func TestGetChunkCount(t *testing.T) {
	// Save and restore
	origChunks := ephemeralRoomChunks
	defer func() { ephemeralRoomChunks = origChunks }()

	ephemeralRoomChunks = [ephemeralChunksLimit][]int{}
	assert.Equal(t, 0, GetChunkCount())

	ephemeralRoomChunks[0] = []int{1, 2, 3}
	assert.Equal(t, 1, GetChunkCount())

	ephemeralRoomChunks[5] = []int{10}
	assert.Equal(t, 2, GetChunkCount())
}

// ─── FindEphemeralRoomIds ───────────────────────────────────────────────────

func TestFindEphemeralRoomIds(t *testing.T) {
	origLookups := originalRoomIdLookups
	defer func() { originalRoomIdLookups = origLookups }()

	originalRoomIdLookups = map[int]int{
		ephemeralRoomIdMinimum + 1: 42,
		ephemeralRoomIdMinimum + 2: 42,
		ephemeralRoomIdMinimum + 3: 99,
	}

	ids := FindEphemeralRoomIds(42)
	assert.Len(t, ids, 2)

	ids = FindEphemeralRoomIds(99)
	assert.Len(t, ids, 1)

	ids = FindEphemeralRoomIds(777)
	assert.Empty(t, ids)
}

// ─── AddTemporaryExit ───────────────────────────────────────────────────────

func TestRoom_AddTemporaryExit(t *testing.T) {
	r := &Room{}

	t.Run("add first exit", func(t *testing.T) {
		added := r.AddTemporaryExit("portal", exit.TemporaryRoomExit{RoomId: 100, UserId: 1})
		assert.True(t, added)
		assert.Len(t, r.ExitsTemp, 1)
	})

	t.Run("duplicate name rejected", func(t *testing.T) {
		added := r.AddTemporaryExit("portal", exit.TemporaryRoomExit{RoomId: 200, UserId: 2})
		assert.False(t, added)
		assert.Len(t, r.ExitsTemp, 1)
	})

	t.Run("ephemeral_overwrite_allowed", func(t *testing.T) {
		r2 := &Room{}
		r2.AddTemporaryExit("eph_portal", exit.TemporaryRoomExit{RoomId: ephemeralRoomIdMinimum + 10, UserId: 1})
		added := r2.AddTemporaryExit("eph_portal", exit.TemporaryRoomExit{RoomId: ephemeralRoomIdMinimum + 20, UserId: 2})
		assert.True(t, added)
		assert.Equal(t, ephemeralRoomIdMinimum+20, r2.ExitsTemp["eph_portal"].RoomId)
	})

	t.Run("mixed_existing_ephemeral_new_normal_rejected", func(t *testing.T) {
		r3 := &Room{}
		r3.AddTemporaryExit("eph_portal", exit.TemporaryRoomExit{RoomId: ephemeralRoomIdMinimum + 30, UserId: 1})
		added := r3.AddTemporaryExit("eph_portal", exit.TemporaryRoomExit{RoomId: 500, UserId: 2})
		assert.False(t, added)
		assert.Equal(t, ephemeralRoomIdMinimum+30, r3.ExitsTemp["eph_portal"].RoomId)
	})

	t.Run("mixed_existing_normal_new_ephemeral_rejected", func(t *testing.T) {
		r4 := &Room{}
		r4.AddTemporaryExit("portal", exit.TemporaryRoomExit{RoomId: 600, UserId: 1})
		added := r4.AddTemporaryExit("portal", exit.TemporaryRoomExit{RoomId: ephemeralRoomIdMinimum + 40, UserId: 2})
		assert.False(t, added)
		assert.Equal(t, 600, r4.ExitsTemp["portal"].RoomId)
	})

	t.Run("empty title defaults to exit name", func(t *testing.T) {
		added := r.AddTemporaryExit("rift", exit.TemporaryRoomExit{RoomId: 300, UserId: 3})
		assert.True(t, added)
		assert.Equal(t, "rift", r.ExitsTemp["rift"].Title)
	})
}

// ─── GetDescriptionFormatted ────────────────────────────────────────────────

func TestRoom_GetDescriptionFormatted(t *testing.T) {
	t.Run("no nouns", func(t *testing.T) {
		r := &Room{Description: "A simple room."}
		desc := r.GetDescriptionFormatted(80, false)
		assert.Contains(t, desc, "A simple room.")
	})

	t.Run("with noun highlighting", func(t *testing.T) {
		r := &Room{
			Description: "There is a torch on the wall.",
			Nouns:       map[string]string{"torch": "a fiery torch"},
		}
		desc := r.GetDescriptionFormatted(80, true)
		assert.Contains(t, desc, `<ansi fg="noun">torch</ansi>`)
	})
}

// ─── GetAllRoomIds ──────────────────────────────────────────────────────────

func TestGetAllRoomIds(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	// Seed the file cache too
	roomManager.roomIdToFileCache[1] = "1.yaml"
	roomManager.roomIdToFileCache[2] = "2.yaml"
	roomManager.roomIdToFileCache[3] = "3.yaml"

	ids := GetAllRoomIds()
	assert.Len(t, ids, 3)
}

// ─── GetRoomsWithPlayers ────────────────────────────────────────────────────

func TestGetRoomsWithPlayers(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("no players anywhere", func(t *testing.T) {
		rooms := GetRoomsWithPlayers()
		assert.Empty(t, rooms)
	})

	t.Run("with players", func(t *testing.T) {
		roomManager.rooms[1].AddPlayer(10)
		roomManager.roomsWithUsers[1] = 1
		rooms := GetRoomsWithPlayers()
		assert.Len(t, rooms, 1)
		assert.Contains(t, rooms, 1)
	})
}

// ─── Room.GetBiome ──────────────────────────────────────────────────────────

func TestRoom_GetBiome(t *testing.T) {
	cleanupBiomes := seedBiomes()
	defer cleanupBiomes()

	t.Run("room with biome set", func(t *testing.T) {
		r := &Room{Biome: "city"}
		bi := r.GetBiome()
		assert.NotNil(t, bi)
		assert.Equal(t, "City", bi.Name)
	})

	t.Run("room with nonexistent biome falls back to default", func(t *testing.T) {
		r := &Room{Biome: "volcano"}
		bi := r.GetBiome()
		assert.NotNil(t, bi)
		assert.Equal(t, "Default", bi.Name)
	})
}

// ─── ZoneConfig.Filepath ────────────────────────────────────────────────────

func TestZoneConfig_Filepath(t *testing.T) {
	z := NewZoneConfig("Test Zone")
	fp := z.Filepath()
	assert.Contains(t, fp, "test_zone")
	assert.Contains(t, fp, "zone-config.yaml")
}

// ─── Room.Gold ──────────────────────────────────────────────────────────────

func TestRoom_Gold(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	r := roomManager.rooms[1]
	assert.Equal(t, 0, r.Gold)
	r.Gold = 100
	assert.Equal(t, 100, r.Gold)
}

// ─── Container.Gold ─────────────────────────────────────────────────────────

func TestContainer_Gold(t *testing.T) {
	c := Container{Gold: 50}
	assert.Equal(t, 50, c.Gold)

	c.Gold += 25
	assert.Equal(t, 75, c.Gold)
}

// ─── Container.DespawnRound ─────────────────────────────────────────────────

func TestContainer_DespawnRound(t *testing.T) {
	c := Container{DespawnRound: 0}
	assert.Equal(t, uint64(0), c.DespawnRound)

	c.DespawnRound = 100
	assert.Equal(t, uint64(100), c.DespawnRound)
}

// ─── Room IsBank/IsStorage/IsCharacterRoom ──────────────────────────────────

func TestRoom_SpecialFlags(t *testing.T) {
	r := &Room{}
	assert.False(t, r.IsBank)
	assert.False(t, r.IsStorage)
	assert.False(t, r.IsCharacterRoom)

	r.IsBank = true
	r.IsStorage = true
	r.IsCharacterRoom = true
	assert.True(t, r.IsBank)
	assert.True(t, r.IsStorage)
	assert.True(t, r.IsCharacterRoom)
}

// ─── Room.IdleMessages ──────────────────────────────────────────────────────

func TestRoom_IdleMessages(t *testing.T) {
	r := &Room{
		IdleMessages: []string{"The wind howls.", "A distant rumble."},
	}
	assert.Len(t, r.IdleMessages, 2)
	assert.Equal(t, uint8(0), r.LastIdleMessage)
}

// ─── Room.SkillTraining ─────────────────────────────────────────────────────

func TestRoom_SkillTraining(t *testing.T) {
	r := &Room{
		SkillTraining: map[string]TrainingRange{
			"weapon-combat": {Min: 0, Max: 10},
			"spellcasting":  {Min: 5, Max: 25},
		},
	}
	assert.Len(t, r.SkillTraining, 2)
	tr := r.SkillTraining["weapon-combat"]
	assert.Equal(t, 0, tr.Min)
	assert.Equal(t, 10, tr.Max)
}

// ─── Room.Station ───────────────────────────────────────────────────────────

func TestRoom_Station(t *testing.T) {
	r := &Room{Station: "forge"}
	assert.Equal(t, "forge", r.Station)
}

// ─── GetZonesWithMutators ───────────────────────────────────────────────────

func TestGetZonesWithMutators(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	// No mutators on our test zone
	names, ids := GetZonesWithMutators()
	assert.Empty(t, names)
	assert.Empty(t, ids)
}

// ─── Room.Validate with SpawnInfo ───────────────────────────────────────────

func TestRoom_Validate_SpawnInfo(t *testing.T) {
	t.Run("spawn info with default respawn rate", func(t *testing.T) {
		r := &Room{
			RoomId:      99,
			Title:       "Test",
			Description: "Test room.",
			SpawnInfo: []SpawnInfo{
				{MobId: 1},
			},
		}
		err := r.Validate()
		assert.NoError(t, err)
		assert.Equal(t, "15 real minutes", r.SpawnInfo[0].RespawnRate)
	})

	t.Run("spawn info with mob and item = error", func(t *testing.T) {
		r := &Room{
			RoomId:      99,
			Title:       "Test",
			Description: "Test room.",
			SpawnInfo: []SpawnInfo{
				{MobId: 1, ItemId: 5},
			},
		}
		err := r.Validate()
		assert.Error(t, err)
	})

	t.Run("spawn info with mob and gold = error", func(t *testing.T) {
		r := &Room{
			RoomId:      99,
			Title:       "Test",
			Description: "Test room.",
			SpawnInfo: []SpawnInfo{
				{MobId: 1, Gold: 10},
			},
		}
		err := r.Validate()
		assert.Error(t, err)
	})
}

// ─── GetRoomWithMostItems ───────────────────────────────────────────────────

func TestGetRoomWithMostItems(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	// Add items to rooms
	roomManager.rooms[2].Items = []items.Item{
		{ItemId: 10}, {ItemId: 20}, {ItemId: 30},
	}
	roomManager.rooms[4].Items = []items.Item{
		{ItemId: 10},
	}

	roomId, itemCt := GetRoomWithMostItems(false, 1, 0)
	assert.Equal(t, 2, roomId)
	assert.Equal(t, 3, itemCt)
}

// ─── Room.MapLegend ─────────────────────────────────────────────────────────

func TestRoom_MapLegend(t *testing.T) {
	r := &Room{MapLegend: "Town"}
	assert.Equal(t, "Town", r.MapLegend)
}

// ─── Room.MusicFile ─────────────────────────────────────────────────────────

func TestRoom_MusicFile(t *testing.T) {
	r := &Room{MusicFile: "tavern.mp3"}
	assert.Equal(t, "tavern.mp3", r.MusicFile)
}

// ─── ZoneConfig fields ──────────────────────────────────────────────────────

func TestZoneConfig_Fields(t *testing.T) {
	z := &ZoneConfig{
		Name:         "Test Zone",
		RoomId:       1,
		DefaultBiome: "city",
		MusicFile:    "zone.mp3",
		Region:       "Thornwall Region",
		IdleMessages: []string{"msg1", "msg2"},
	}
	assert.Equal(t, "Test Zone", z.Id())
	assert.Equal(t, "city", z.DefaultBiome)
	assert.Equal(t, "zone.mp3", z.MusicFile)
	assert.Equal(t, "Thornwall Region", z.Region)
	assert.Len(t, z.IdleMessages, 2)
}

// ─── GetMemoryUsage ─────────────────────────────────────────────────────────

func TestGetMemoryUsage(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	result := GetMemoryUsage()
	assert.Contains(t, result, "rooms")
	assert.Contains(t, result, "zones")
	assert.Contains(t, result, "roomsWithUsers")
	assert.Contains(t, result, "roomIdToFileCache")
	assert.Equal(t, 5, result["rooms"].Count)
	assert.Equal(t, 1, result["zones"].Count)
}

// ─── FindExitTo with temporary exits ────────────────────────────────────────

func TestRoom_FindExitTo_TempExits(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	r := roomManager.rooms[3] // dead end room with no exits
	r.ExitsTemp = map[string]exit.TemporaryRoomExit{
		"portal": {RoomId: 999, Title: "portal", UserId: 42},
	}

	// Should find through temp exit
	exitName := r.FindExitTo(999)
	assert.Equal(t, "portal", exitName)

	// Normal exit not found still returns empty
	exitName = r.FindExitTo(888)
	assert.Equal(t, "", exitName)
}

// ─── FindOnFloor ────────────────────────────────────────────────────────────

func TestRoom_FindOnFloor(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	r := roomManager.rooms[3]
	r.Items = []items.Item{
		{ItemId: 42, UUID: [16]byte{1}},
		{ItemId: 99, UUID: [16]byte{2}},
	}
	r.Stash = []items.Item{
		{ItemId: 77, UUID: [16]byte{3}},
	}

	t.Run("find on floor by !itemId", func(t *testing.T) {
		found, ok := r.FindOnFloor("!42", false)
		assert.True(t, ok)
		assert.Equal(t, 42, found.ItemId)
	})

	t.Run("find in stash by !itemId", func(t *testing.T) {
		found, ok := r.FindOnFloor("!77", true)
		assert.True(t, ok)
		assert.Equal(t, 77, found.ItemId)
	})

	t.Run("not found on floor", func(t *testing.T) {
		_, ok := r.FindOnFloor("!999", false)
		assert.False(t, ok)
	})

	t.Run("not found in stash", func(t *testing.T) {
		_, ok := r.FindOnFloor("!999", true)
		assert.False(t, ok)
	})
}

// ─── Container.FindItem ─────────────────────────────────────────────────────

func TestContainer_FindItem(t *testing.T) {
	c := &Container{
		Items: []items.Item{
			{ItemId: 10, UUID: [16]byte{1}},
			{ItemId: 20, UUID: [16]byte{2}},
		},
	}

	t.Run("find by !itemId", func(t *testing.T) {
		found, ok := c.FindItem("!10")
		assert.True(t, ok)
		assert.Equal(t, 10, found.ItemId)
	})

	t.Run("not found", func(t *testing.T) {
		_, ok := c.FindItem("!999")
		assert.False(t, ok)
	})
}

// ─── Room.FindContainerByName deeper ────────────────────────────────────────

func TestRoom_FindContainerByName_Empty(t *testing.T) {
	r := &Room{Containers: map[string]Container{}}
	name := r.FindContainerByName("chest")
	assert.Equal(t, "", name)
}

// ─── Room.MarkVisited with nil visitors ─────────────────────────────────────

func TestRoom_MarkVisited_NilVisitors(t *testing.T) {
	r := &Room{visitors: nil}
	r.MarkVisited(42, VisitorUser)
	assert.True(t, r.HasVisited(42, VisitorUser))
}

// ─── Room.PruneVisitors ─────────────────────────────────────────────────────

func TestRoom_PruneVisitors_NilMap(t *testing.T) {
	r := &Room{visitors: nil}
	ct := r.PruneVisitors()
	assert.Equal(t, 0, ct)
	assert.Nil(t, r.visitors) // Fast path: no allocation for nil map
}

// ─── Room.Nouns empty ───────────────────────────────────────────────────────

func TestRoom_FindNoun_Empty(t *testing.T) {
	r := &Room{Nouns: map[string]string{}}
	found, desc := r.FindNoun("torch")
	assert.Equal(t, "", found)
	assert.Equal(t, "", desc)
}

// ─── Room.Validate container items ──────────────────────────────────────────

func TestRoom_Validate_ContainerItems(t *testing.T) {
	r := &Room{
		RoomId:      99,
		Title:       "Test",
		Description: "Test room.",
		Items: []items.Item{
			{ItemId: 10},
		},
		Stash: []items.Item{
			{ItemId: 20},
		},
		Containers: map[string]Container{
			"chest": {
				Items: []items.Item{
					{ItemId: 30},
				},
			},
		},
	}
	err := r.Validate()
	assert.NoError(t, err)
}

// ─── Room Corpse search ─────────────────────────────────────────────────────

func TestRoom_FindCorpse_Empty(t *testing.T) {
	r := &Room{}
	_, ok := r.FindCorpse("anything")
	assert.False(t, ok)
}

// ─── Room.SpawnInfo struct ──────────────────────────────────────────────────

func TestRoom_SpawnInfo(t *testing.T) {
	r := &Room{
		RoomId:      1,
		Title:       "Test",
		Description: "Test room.",
		SpawnInfo: []SpawnInfo{
			{ItemId: 10, RespawnRate: "5 real minutes"},
		},
	}
	err := r.Validate()
	assert.NoError(t, err)
	assert.Equal(t, "5 real minutes", r.SpawnInfo[0].RespawnRate)
}

// ─── Room.GetRoomWithMostItems with gold ────────────────────────────────────

func TestGetRoomWithMostItems_Gold(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	roomManager.rooms[3].Gold = 500

	roomId, ct := GetRoomWithMostItems(false, 0, 100)
	assert.Equal(t, 3, roomId)
	assert.Equal(t, 500, ct)
}

// ─── Room.GetMobs / GetPlayers (empty cases) ───────────────────────────────

func TestRoom_GetMobs_Empty(t *testing.T) {
	r := &Room{mobs: []int{}}
	result := r.GetMobs()
	assert.Empty(t, result)
}

func TestRoom_GetPlayers_Empty(t *testing.T) {
	r := &Room{players: []int{}}
	result := r.GetPlayers()
	assert.Empty(t, result)
}

// ─── Room.GetRandomExit with secret/locked exits ────────────────────────────

func TestRoom_GetRandomExit_SkipsSecret(t *testing.T) {
	r := &Room{
		Exits: map[string]exit.RoomExit{
			"west": {RoomId: 5, Secret: true}, // Should be skipped
		},
		visitors: make(map[VisitorType]map[int]uint64),
	}
	name, roomId := r.GetRandomExit()
	assert.Equal(t, "", name)
	assert.Equal(t, 0, roomId)
}

// ─── Room.RemoveItem stash ──────────────────────────────────────────────────

func TestRoom_RemoveItem_Stash(t *testing.T) {
	r := &Room{
		Stash: []items.Item{
			{ItemId: 10, UUID: [16]byte{1}},
			{ItemId: 20, UUID: [16]byte{2}},
		},
	}
	r.RemoveItem(items.Item{ItemId: 10, UUID: [16]byte{1}}, true)
	assert.Len(t, r.Stash, 1)
	assert.Equal(t, 20, r.Stash[0].ItemId)
}

// ─── Room.SetExitLock deeper ────────────────────────────────────────────────

func TestRoom_SetExitLock_Deeper(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	r := roomManager.rooms[1]

	t.Run("lock exit with lock", func(t *testing.T) {
		r.SetExitLock("east", true)
		assert.True(t, r.Exits["east"].Lock.IsLocked())
	})

	t.Run("nonexistent exit is no-op", func(t *testing.T) {
		r.SetExitLock("nonexistent", true)
	})
}

// ─── Room fields ────────────────────────────────────────────────────────────

func TestRoom_Biome(t *testing.T) {
	r := &Room{Biome: "forest"}
	assert.Equal(t, "forest", r.Biome)
}

func TestRoom_Zone(t *testing.T) {
	r := &Room{Zone: "TestZone"}
	assert.Equal(t, "TestZone", r.Zone)
}

func TestRoom_Title(t *testing.T) {
	r := &Room{Title: "Test Room"}
	assert.Equal(t, "Test Room", r.Title)
}

// ─── MapSymbolOverrides ─────────────────────────────────────────────────────

func TestMapSymbolOverrides(t *testing.T) {
	assert.Contains(t, MapSymbolOverrides, "*")
	assert.Equal(t, defaultMapSymbol, MapSymbolOverrides["*"])
}

// ─── FindFlag constants ─────────────────────────────────────────────────────

func TestFindFlagConstants(t *testing.T) {
	assert.Equal(t, FindFighting, FindFightingPlayer|FindFightingMob)
	assert.Equal(t, FindIdle, FindCharmed|FindNeutral)
}

// ─── VisitorType constants ──────────────────────────────────────────────────

func TestVisitorTypeConstants(t *testing.T) {
	assert.Equal(t, "user", string(VisitorUser))
	assert.Equal(t, "mob", string(VisitorMob))
}

// ─── Corpse struct ──────────────────────────────────────────────────────────

func TestCorpseStruct(t *testing.T) {
	c := Corpse{
		MobId:        1,
		UserId:       0,
		Character:    characters.Character{Name: "Goblin"},
		RoundCreated: 10,
		Prunable:     false,
	}
	assert.Equal(t, 1, c.MobId)
	assert.Equal(t, "Goblin", c.Character.Name)
	assert.Equal(t, uint64(10), c.RoundCreated)
	assert.False(t, c.Prunable)
}

// ─── Room.AddItem validates ─────────────────────────────────────────────────

func TestRoom_AddItem_Validates(t *testing.T) {
	r := &Room{Items: []items.Item{}, Stash: []items.Item{}}

	// Items with no UUID get one after Validate
	itm := items.Item{ItemId: 42}
	r.AddItem(itm, false)
	assert.Len(t, r.Items, 1)

	itm2 := items.Item{ItemId: 99}
	r.AddItem(itm2, true)
	assert.Len(t, r.Stash, 1)
}

// ─── Room.GetExitInfo with temp and normal ──────────────────────────────────

func TestRoom_GetExitInfo_Deeper(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	r := roomManager.rooms[1]

	t.Run("secret exit returns secret flag", func(t *testing.T) {
		info, ok := r.GetExitInfo("west")
		assert.True(t, ok)
		assert.True(t, info.Secret)
	})

	t.Run("locked exit returns lock info", func(t *testing.T) {
		info, ok := r.GetExitInfo("east")
		assert.True(t, ok)
		assert.True(t, info.HasLock())
	})
}

// ─── AffectsNone/Player/Room constants ──────────────────────────────────────

func TestAffectsConstants(t *testing.T) {
	assert.Equal(t, "", AffectsNone)
	assert.Equal(t, "player", AffectsPlayer)
	assert.Equal(t, "room", AffectsRoom)
}

// ─── Room.HasRecentVisitors edge cases ──────────────────────────────────────

func TestRoom_HasRecentVisitors_NilMap(t *testing.T) {
	r := &Room{visitors: nil}
	assert.False(t, r.HasRecentVisitors())
}

// ─── Room.HasVisited edge cases ─────────────────────────────────────────────

func TestRoom_HasVisited_NilVisitorType(t *testing.T) {
	r := &Room{visitors: make(map[VisitorType]map[int]uint64)}
	assert.False(t, r.HasVisited(42, VisitorUser))
}

// ─── Room.GetPublicSigns/GetPrivateSigns with mixed ─────────────────────────

func TestRoom_Signs_Mixed(t *testing.T) {
	r := &Room{
		Signs: []Sign{
			{DisplayText: "Public 1", VisibleUserId: 0, Expires: time.Now().Add(1 * time.Hour)},
			{DisplayText: "Private 1", VisibleUserId: 10, Expires: time.Now().Add(1 * time.Hour)},
			{DisplayText: "Public 2", VisibleUserId: 0, Expires: time.Now().Add(1 * time.Hour)},
			{DisplayText: "Private 2", VisibleUserId: 20, Expires: time.Now().Add(1 * time.Hour)},
		},
	}
	pub := r.GetPublicSigns()
	assert.Len(t, pub, 2)
	priv := r.GetPrivateSigns()
	assert.Len(t, priv, 2)
}

// ─── Room.RemoveCorpse match criteria ───────────────────────────────────────

func TestRoom_RemoveCorpse_MatchCriteria(t *testing.T) {
	r := &Room{}
	c := Corpse{
		MobId:        1,
		UserId:       0,
		Character:    characters.Character{Name: "Goblin"},
		RoundCreated: 10,
	}
	r.AddCorpse(c)

	// Wrong MobId
	assert.False(t, r.RemoveCorpse(Corpse{MobId: 2, UserId: 0, Character: characters.Character{Name: "Goblin"}, RoundCreated: 10}))
	// Wrong UserId
	assert.False(t, r.RemoveCorpse(Corpse{MobId: 1, UserId: 1, Character: characters.Character{Name: "Goblin"}, RoundCreated: 10}))
	// Wrong Name
	assert.False(t, r.RemoveCorpse(Corpse{MobId: 1, UserId: 0, Character: characters.Character{Name: "Orc"}, RoundCreated: 10}))
	// Wrong RoundCreated
	assert.False(t, r.RemoveCorpse(Corpse{MobId: 1, UserId: 0, Character: characters.Character{Name: "Goblin"}, RoundCreated: 99}))
	// Correct - still there
	assert.True(t, r.RemoveCorpse(c))
	assert.Len(t, r.Corpses, 0)
}

// ─── ClearRoomCache deeper ──────────────────────────────────────────────────

func TestClearRoomCache_CleansZone(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	// Add file cache entry
	roomManager.roomIdToFileCache[1] = "1.yaml"
	roomManager.roomsWithUsers[1] = 1
	roomManager.roomsWithMobs[1] = 1

	err := ClearRoomCache(1)
	assert.NoError(t, err)

	// Verify room removed from zone
	zone := roomManager.zones["TestZone"]
	_, exists := zone.RoomIds[1]
	assert.False(t, exists)

	// Verify removed from all maps
	assert.False(t, IsRoomLoaded(1))
	_, inUsers := roomManager.roomsWithUsers[1]
	assert.False(t, inUsers)
	_, inMobs := roomManager.roomsWithMobs[1]
	assert.False(t, inMobs)
	_, inCache := roomManager.roomIdToFileCache[1]
	assert.False(t, inCache)
}

// ─── Room.GetExitInfo with temp exits ───────────────────────────────────────

func TestRoom_GetExitInfo_Normal(t *testing.T) {
	r := &Room{
		Exits: map[string]exit.RoomExit{
			"north": {RoomId: 2},
			"south": {RoomId: 3, Secret: true},
		},
	}

	info, ok := r.GetExitInfo("north")
	assert.True(t, ok)
	assert.Equal(t, 2, info.RoomId)

	info, ok = r.GetExitInfo("south")
	assert.True(t, ok)
	assert.True(t, info.Secret)

	_, ok = r.GetExitInfo("west")
	assert.False(t, ok)
}

// ─── Room.AddSign with multiple private signs ───────────────────────────────

func TestRoom_AddSign_MultiplePrivate(t *testing.T) {
	r := &Room{Signs: []Sign{}}

	// Two different users can each have a private sign
	r.AddSign("Rune A", 10, 7)
	r.AddSign("Rune B", 20, 7)
	assert.Len(t, r.Signs, 2)

	// Same user replaces their rune
	replaced := r.AddSign("Rune A2", 10, 7)
	assert.True(t, replaced)
	assert.Len(t, r.Signs, 2)
}

// ─── Room.GetDescriptionFormatted with long text ────────────────────────────

func TestRoom_GetDescriptionFormatted_NoNouns(t *testing.T) {
	r := &Room{
		Description: "Short room.",
		Nouns:       map[string]string{"something": "desc"},
	}
	// With highlight but noun not in text
	desc := r.GetDescriptionFormatted(80, true)
	assert.Contains(t, desc, "Short room.")
	assert.NotContains(t, desc, "<ansi")
}

// ─── Room.FindOnFloor with multiple items ───────────────────────────────────

func TestRoom_FindOnFloor_Empty(t *testing.T) {
	r := &Room{Items: []items.Item{}, Stash: []items.Item{}}

	_, ok := r.FindOnFloor("anything", false)
	assert.False(t, ok)

	_, ok = r.FindOnFloor("anything", true)
	assert.False(t, ok)
}

// ─── Room.GetAllFloorItems edge cases ───────────────────────────────────────

func TestRoom_GetAllFloorItems_Empty(t *testing.T) {
	r := &Room{Items: []items.Item{}, Stash: []items.Item{}}
	assert.Empty(t, r.GetAllFloorItems(false))
	assert.Empty(t, r.GetAllFloorItems(true))
}

// ─── Room.FindCorpse with corpses ───────────────────────────────────────────

func TestRoom_FindCorpse_Multiple(t *testing.T) {
	r := &Room{}
	r.AddCorpse(Corpse{MobId: 1, Character: characters.Character{Name: "Goblin"}, RoundCreated: 5})
	r.AddCorpse(Corpse{MobId: 2, Character: characters.Character{Name: "Orc"}, RoundCreated: 6})

	found, ok := r.FindCorpse("Goblin corpse")
	assert.True(t, ok)
	assert.Equal(t, "Goblin", found.Character.Name)

	found, ok = r.FindCorpse("Orc corpse")
	assert.True(t, ok)
	assert.Equal(t, "Orc", found.Character.Name)

	_, ok = r.FindCorpse("Dragon corpse")
	assert.False(t, ok)
}

// ─── GetRoomWithMostItems skip recently visited ─────────────────────────────

func TestGetRoomWithMostItems_SkipVisited(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	roomManager.rooms[2].Items = []items.Item{{ItemId: 10}, {ItemId: 20}}
	roomManager.rooms[2].MarkVisited(1, VisitorUser)

	roomManager.rooms[4].Items = []items.Item{{ItemId: 30}}

	roomId, _ := GetRoomWithMostItems(true, 1, 0)
	// Room 2 has more items but was recently visited, should get room 4
	assert.Equal(t, 4, roomId)
}

// ─── Container.FindItem deeper ──────────────────────────────────────────────

func TestContainer_FindItem_Empty(t *testing.T) {
	c := &Container{Items: []items.Item{}}
	_, ok := c.FindItem("anything")
	assert.False(t, ok)
}

// ─── addRoomToMemory ────────────────────────────────────────────────────────

func TestAddRoomToMemory(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("add new room", func(t *testing.T) {
		r := &Room{
			RoomId:      99,
			Zone:        "TestZone",
			Title:       "New Room",
			Description: "A new room.",
			Exits:       map[string]exit.RoomExit{},
		}
		err := addRoomToMemory(r)
		assert.NoError(t, err)
		assert.True(t, IsRoomLoaded(99))
		// Verify it was added to zone
		zone := roomManager.zones["TestZone"]
		_, exists := zone.RoomIds[99]
		assert.True(t, exists)
	})

	t.Run("duplicate room returns error", func(t *testing.T) {
		r := &Room{
			RoomId:      99,
			Zone:        "TestZone",
			Title:       "Duplicate",
			Description: "Should fail.",
		}
		err := addRoomToMemory(r)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already stored")
	})

	t.Run("new zone auto-created", func(t *testing.T) {
		r := &Room{
			RoomId:      200,
			Zone:        "NewZone",
			Title:       "New Zone Room",
			Description: "In a new zone.",
			Exits:       map[string]exit.RoomExit{},
		}
		err := addRoomToMemory(r)
		assert.NoError(t, err)
		cfg := GetZoneConfig("NewZone")
		assert.NotNil(t, cfg)
		assert.Equal(t, "NewZone", cfg.Name)
		_, exists := cfg.RoomIds[200]
		assert.True(t, exists)
	})
}

// ─── GetFilePath ────────────────────────────────────────────────────────────

func TestGetFilePath_Cached(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	roomManager.roomIdToFileCache[1] = "testzone/1.yaml"
	path := roomManager.GetFilePath(1)
	assert.Equal(t, "testzone/1.yaml", path)
}

func TestGetFilePath_NotCached(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	// Room 999 not in cache, searchForRoomFile will return empty
	path := roomManager.GetFilePath(999)
	assert.Equal(t, "", path)
}

// ─── EphemeralRoomMaintenance ───────────────────────────────────────────────

func TestEphemeralRoomMaintenance_Empty(t *testing.T) {
	origLookups := originalRoomIdLookups
	origChunks := ephemeralRoomChunks
	defer func() {
		originalRoomIdLookups = origLookups
		ephemeralRoomChunks = origChunks
	}()

	originalRoomIdLookups = map[int]int{}
	ephemeralRoomChunks = [ephemeralChunksLimit][]int{}

	result := EphemeralRoomMaintenance()
	assert.Empty(t, result)
}

// ─── removeRoomFromMemory ───────────────────────────────────────────────────

func TestRemoveRoomFromMemory_NotFound(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	r := &Room{RoomId: 999}
	removeRoomFromMemory(r) // should not panic
	assert.Equal(t, 5, len(roomManager.rooms))
}

func TestRemoveRoomFromMemory_HasPlayers(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	roomManager.rooms[1].AddPlayer(42)
	r := roomManager.rooms[1]
	removeRoomFromMemory(r) // should not remove because it has players
	assert.True(t, IsRoomLoaded(1))
}

func TestFindNoun(t *testing.T) {
	// Create a room with various noun mappings (including aliases).
	r := &Room{
		Nouns: map[string]string{
			"torch":            "a fiery torch",
			"torchAlias":       ":torch", // alias -> :torch means "torch"
			"lamp":             "an illuminating lamp",
			"lampAlias":        ":lamp", // alias -> :lamp means "lamp"
			"candles":          "some wax candles",
			"pony":             "a small horse",
			"mystery":          "just a riddle",
			"secret":           ":mystery",                   // chain alias -> :mystery
			"projector screen": "a wide, matte-white screen", // multi-word matching
			"island":           "a tropical paradise",
			"rocks":            ":island",
			"rocky island":     ":island",
			"feet":             ":hands",
			"hands":            ":feet",
		},
	}

	// Table-driven tests.
	tests := []struct {
		name          string
		inputNoun     string
		wantFoundNoun string
		wantDesc      string
	}{
		{
			name:          "Direct match (torch)",
			inputNoun:     "torch",
			wantFoundNoun: "torch",
			wantDesc:      "a fiery torch",
		},
		{
			name:          "Direct match (candles)",
			inputNoun:     "candles",
			wantFoundNoun: "candles",
			wantDesc:      "some wax candles",
		},
		{
			name:          "Direct match (pony)",
			inputNoun:     "pony",
			wantFoundNoun: "pony",
			wantDesc:      "a small horse",
		},
		{
			name:          "Alias match (torchAlias -> :torch)",
			inputNoun:     "torchAlias",
			wantFoundNoun: "torch",
			wantDesc:      "a fiery torch",
		},
		{
			name:          "Alias match (lampAlias -> :lamp)",
			inputNoun:     "lampAlias",
			wantFoundNoun: "lamp",
			wantDesc:      "an illuminating lamp",
		},
		{
			name:          "Chain alias (secret -> :mystery)",
			inputNoun:     "secret",
			wantFoundNoun: "mystery",
			wantDesc:      "just a riddle",
		},
		{
			name:          "Plural form by adding s (pony -> ponies)",
			inputNoun:     "ponies",
			wantFoundNoun: "pony",
			wantDesc:      "a small horse",
		},
		{
			name:          "Plural form by adding es (torches)",
			inputNoun:     "torches",
			wantFoundNoun: "torch",
			wantDesc:      "a fiery torch",
		},

		{
			name:          "Multi-word input, second word valid (red torches)",
			inputNoun:     "red torches",
			wantFoundNoun: "torch",
			wantDesc:      "a fiery torch",
		},
		{
			name:          "Multi-word input, multi-word match",
			inputNoun:     "projector screen",
			wantFoundNoun: "projector screen",
			wantDesc:      "a wide, matte-white screen",
		},
		{
			name:          "Single-word input, multi-word match 1",
			inputNoun:     "projector",
			wantFoundNoun: "projector screen",
			wantDesc:      "a wide, matte-white screen",
		},
		{
			name:          "Single-word input, multi-word match 2",
			inputNoun:     "screen",
			wantFoundNoun: "projector screen",
			wantDesc:      "a wide, matte-white screen",
		},
		{
			name:          "Multi-word input, first word valid (torch something)",
			inputNoun:     "torch something",
			wantFoundNoun: "torch",
			wantDesc:      "a fiery torch",
		},
		{
			name:          "No match",
			inputNoun:     "gibberish",
			wantFoundNoun: "",
			wantDesc:      "",
		},
		{
			name:          "Multi-word no match (foo bar)",
			inputNoun:     "foo bar",
			wantFoundNoun: "",
			wantDesc:      "",
		},
		{
			name:          "Multi-word input, first word valid (island)",
			inputNoun:     "island",
			wantFoundNoun: "island",
			wantDesc:      "a tropical paradise",
		},
		// Issue #356: Fix Noun aliases
		{
			name:          "Multi-word input, first word valid (rocks)",
			inputNoun:     "rocks",
			wantFoundNoun: "island",
			wantDesc:      "a tropical paradise",
		},
		{
			name:          "Multi-word input, first word valid (rocky island)",
			inputNoun:     "rocky island",
			wantFoundNoun: "island",
			wantDesc:      "a tropical paradise",
		},
		{
			name:          "Multi-word input, first word valid (rocky)",
			inputNoun:     "rocky",
			wantFoundNoun: "island",
			wantDesc:      "a tropical paradise",
		},
		{
			name:          "circular references (hands)",
			inputNoun:     "hands",
			wantFoundNoun: "",
			wantDesc:      "",
		},
		{
			name:          "circular references (feet)",
			inputNoun:     "feet",
			wantFoundNoun: "",
			wantDesc:      "",
		},
	}

	// Run each sub-test.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFound, gotDesc := r.FindNoun(tt.inputNoun)
			if gotFound != tt.wantFoundNoun || gotDesc != tt.wantDesc {
				t.Errorf("FindNoun(%q) = (%q, %q), want (%q, %q)",
					tt.inputNoun, gotFound, gotDesc, tt.wantFoundNoun, tt.wantDesc)
			}
		})
	}
}

func TestPruneVisitors_EmptyMapFastPath(t *testing.T) {
	r := &Room{
		RoomId:   9991,
		Zone:     "test-zone",
		visitors: nil, // nil map is valid and should short-circuit
	}

	// Should not panic on nil map.
	r.PruneVisitors()

	// Map should still be nil (no allocation performed).
	if r.visitors != nil {
		t.Fatalf("expected visitors to stay nil, got %v", r.visitors)
	}

	// Also works on explicit empty map.
	r.visitors = map[VisitorType]map[int]uint64{}
	r.PruneVisitors()
	if len(r.visitors) != 0 {
		t.Fatalf("expected empty visitors, got %d entries", len(r.visitors))
	}
}
