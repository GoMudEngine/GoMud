package rooms

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/gamelock"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
)

// ─── Integration: Room Player Tracking ───────────────────────────────────────
// Exercises: Room.AddPlayer(), Room.RemovePlayer(), Room.GetPlayers()

func TestIntegration_RoomPlayerTracking(t *testing.T) {
	r := &Room{
		players: []int{},
	}

	// Initially empty
	assert.Empty(t, r.GetPlayers(), "Room should start with no players")

	// Add a player
	count := r.AddPlayer(1)
	assert.Equal(t, 1, count, "Count should be 1 after adding player")
	assert.Equal(t, []int{1}, r.GetPlayers())

	// Add second player
	count = r.AddPlayer(2)
	assert.Equal(t, 2, count)
	assert.ElementsMatch(t, []int{1, 2}, r.GetPlayers())

	// Add third player
	r.AddPlayer(3)
	assert.Len(t, r.GetPlayers(), 3)

	// Duplicate add should not increase count
	count = r.AddPlayer(1)
	assert.Equal(t, 3, count, "Duplicate add should not change count")
	assert.Len(t, r.GetPlayers(), 3)

	// Remove a player
	remaining, found := r.RemovePlayer(2)
	assert.True(t, found, "Should find player to remove")
	assert.Equal(t, 2, remaining)
	assert.ElementsMatch(t, []int{1, 3}, r.GetPlayers())

	// Remove non-existent player
	_, found = r.RemovePlayer(999)
	assert.False(t, found, "Should not find non-existent player")

	// Remove remaining players
	r.RemovePlayer(1)
	r.RemovePlayer(3)
	assert.Empty(t, r.GetPlayers(), "Room should be empty after removing all")
}

// ─── Integration: Room Exit Structure ────────────────────────────────────────
// Exercises: Room.Exits map, exit.RoomExit, secret exits

func TestIntegration_RoomExitStructure(t *testing.T) {
	// Create two rooms with cross-linked exits
	room1 := &Room{
		RoomId: 1,
		Zone:   "test-zone",
		Title:  "Town Square",
		Exits: map[string]exit.RoomExit{
			"north": {RoomId: 2},
			"east":  {RoomId: 3, Secret: true},
		},
		players: []int{},
	}

	room2 := &Room{
		RoomId: 2,
		Zone:   "test-zone",
		Title:  "Market Street",
		Exits: map[string]exit.RoomExit{
			"south": {RoomId: 1},
		},
		players: []int{},
	}

	// Verify exit connections
	assert.Len(t, room1.Exits, 2, "Room 1 should have 2 exits")
	assert.Len(t, room2.Exits, 1, "Room 2 should have 1 exit")

	// Verify cross-linking
	northExit := room1.Exits["north"]
	assert.Equal(t, 2, northExit.RoomId, "North from room 1 should lead to room 2")

	southExit := room2.Exits["south"]
	assert.Equal(t, 1, southExit.RoomId, "South from room 2 should lead to room 1")

	// Verify secret exit
	eastExit := room1.Exits["east"]
	assert.True(t, eastExit.Secret, "East exit should be secret")
	assert.False(t, northExit.Secret, "North exit should not be secret")

	// Verify locked exit
	room3 := &Room{
		RoomId: 3,
		Zone:   "test-zone",
		Title:  "Hidden Chamber",
		Exits: map[string]exit.RoomExit{
			"west": {
				RoomId: 1,
				Lock:   gamelock.Lock{Difficulty: 50},
			},
		},
		players: []int{},
	}

	westExit := room3.Exits["west"]
	assert.True(t, westExit.HasLock(), "Locked exit should report HasLock()")
	assert.Equal(t, uint8(50), westExit.Lock.Difficulty)

	// Unlocked exit
	assert.False(t, northExit.HasLock(), "Normal exit should not have lock")
}

func TestIntegration_RoomTemporaryExits(t *testing.T) {
	r := &Room{
		RoomId:    1,
		Exits:     map[string]exit.RoomExit{"north": {RoomId: 2}},
		ExitsTemp: map[string]exit.TemporaryRoomExit{},
		players:   []int{},
	}

	// Add a temporary exit (e.g., a portal)
	tempExit := exit.TemporaryRoomExit{
		RoomId: 99,
		Title:  "Magical Portal",
		UserId: 1,
	}
	added := r.AddTemporaryExit("portal", tempExit)
	assert.True(t, added, "Should successfully add temporary exit")

	// Find it by user ID
	found, ok := r.FindTemporaryExitByUserId(1)
	assert.True(t, ok, "Should find temporary exit by user ID")
	assert.Equal(t, 99, found.RoomId)

	// Remove it
	removed := r.RemoveTemporaryExit(tempExit)
	assert.True(t, removed, "Should successfully remove temporary exit")

	// Should no longer be findable
	_, ok = r.FindTemporaryExitByUserId(1)
	assert.False(t, ok, "Should not find removed temporary exit")
}

// ─── Integration: Room Containers ────────────────────────────────────────────
// Exercises: Container.AddItem(), RemoveItem(), FindItemById(), HasLock()

func TestIntegration_RoomContainers(t *testing.T) {
	r := &Room{
		RoomId: 1,
		Zone:   "test-zone",
		Containers: map[string]Container{
			"chest": {
				Lock: gamelock.Lock{Difficulty: 30},
			},
			"barrel": {},
		},
		players: []int{},
	}

	// Verify container locks
	chest := r.Containers["chest"]
	assert.True(t, chest.HasLock(), "Chest should have a lock")

	barrel := r.Containers["barrel"]
	assert.False(t, barrel.HasLock(), "Barrel should not have a lock")

	// Add items to container
	item1 := items.Item{ItemId: 100, Spec: &items.ItemSpec{Name: "Gold Coin"}}
	item2 := items.Item{ItemId: 101, Spec: &items.ItemSpec{Name: "Silver Ring"}}

	chest.AddItem(item1)
	chest.AddItem(item2)
	assert.Len(t, chest.Items, 2, "Chest should have 2 items")

	// Find item by ID
	found, ok := chest.FindItemById(100)
	assert.True(t, ok, "Should find item by ID")
	assert.Equal(t, 100, found.ItemId)

	// Find non-existent item
	_, ok = chest.FindItemById(999)
	assert.False(t, ok, "Should not find non-existent item")

	// Remove item
	chest.RemoveItem(item1)
	assert.Len(t, chest.Items, 1, "Chest should have 1 item after removal")

	_, ok = chest.FindItemById(100)
	assert.False(t, ok, "Removed item should no longer be findable")

	// Remaining item should still be there
	found, ok = chest.FindItemById(101)
	assert.True(t, ok)
	assert.Equal(t, 101, found.ItemId)
}

func TestIntegration_ContainerCount(t *testing.T) {
	c := &Container{}

	c.AddItem(items.Item{ItemId: 10})
	c.AddItem(items.Item{ItemId: 10})
	c.AddItem(items.Item{ItemId: 20})

	assert.Equal(t, 2, c.Count(10), "Should count 2 items with ID 10")
	assert.Equal(t, 1, c.Count(20), "Should count 1 item with ID 20")
	assert.Equal(t, 0, c.Count(99), "Should count 0 for missing ID")
}

func TestIntegration_RoomCorpseTracking(t *testing.T) {
	r := &Room{
		players: []int{},
	}

	// Add a corpse and verify
	corpse := Corpse{
		MobId:        1,
		UserId:       0,
		RoundCreated: 100,
	}
	r.AddCorpse(corpse)
	assert.Len(t, r.Corpses, 1)

	// Remove it
	removed := r.RemoveCorpse(corpse)
	assert.True(t, removed)
	assert.Empty(t, r.Corpses)

	// Remove non-existent
	removed = r.RemoveCorpse(Corpse{MobId: 999})
	assert.False(t, removed)
}
