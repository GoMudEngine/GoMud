package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDropMobLoot_GoesIntoCorpse locks in the corpse-loot redesign: when
// corpses are enabled, a dead mob's carried items + gold go into the corpse's
// Loot container instead of onto the room floor. Nothing should leak to the
// floor (room.Items / room.Gold), and exactly one corpse should be created
// whose Loot mirrors what the mob carried.
func TestDropMobLoot_GoesIntoCorpse(t *testing.T) {
	cleanup := items.SeedItemsForTest(map[int]*items.ItemSpec{
		3: {ItemId: 3, Name: "Reward Trinket"},
	})
	defer cleanup()

	biomesCleanup := rooms.SeedBiomesForTest(map[string]*rooms.BiomeInfo{
		"default": {BiomeId: "default", Name: "Default"},
	})
	defer biomesCleanup()

	// Corpses default off in tests (no config file loaded → ConfigBool zero
	// value is false). Enable via the in-memory overlay mechanism the other
	// hooks tests use, and restore the default afterward.
	require.NoError(t, configs.AddOverlayOverrides(map[string]any{
		"GamePlay.Death.CorpsesEnabled": true,
	}))
	defer configs.AddOverlayOverrides(map[string]any{
		"GamePlay.Death.CorpsesEnabled": false,
	})

	m := &mobs.Mob{}
	m.Character = characters.Character{}
	m.Character.Items = []items.Item{{ItemId: 3}}
	m.Character.Gold = 42
	m.ItemDropChance = 100

	room := &rooms.Room{RoomId: 12345, Biome: "default"}

	dropMobLootAndSetCorpse(m, room)

	// (a) Nothing leaked to the floor.
	assert.Equal(t, 0, len(room.Items),
		"no items should drop to the floor when corpses are enabled")
	assert.Equal(t, 0, room.Gold,
		"no gold should drop to the floor when corpses are enabled")

	// (b) Exactly one corpse whose Loot mirrors the mob's carried loot.
	require.Equal(t, 1, len(room.Corpses), "exactly one corpse should be created")
	corpse := room.Corpses[0]
	assert.Equal(t, 42, corpse.Loot.Gold, "corpse Loot.Gold should match the mob's gold")
	assert.Equal(t, 1, len(corpse.Loot.Items), "corpse Loot.Items should match the mob's carried items")
}

// TestDropMobLootAndSetCorpse_NeverDropsSkipsEquippedItem locks in the Core
// Guardian boss-gear design: an equipped item flagged NeverDrops must never
// land on the floor, while a normal equipped item and the mob's carried
// (guaranteed-loot) Items still drop as usual. This is the surgical
// alternative to PermaGear — PermaGear would also suppress carried Items +
// Gold, which the Guardian's intended loot depends on.
func TestDropMobLootAndSetCorpse_NeverDropsSkipsEquippedItem(t *testing.T) {
	cleanup := items.SeedItemsForTest(map[int]*items.ItemSpec{
		1: {ItemId: 1, Name: "Boss Plating", Type: items.Body, NeverDrops: true},
		2: {ItemId: 2, Name: "Ordinary Helm", Type: items.Head},
		3: {ItemId: 3, Name: "Reward Trinket"},
	})
	defer cleanup()

	biomesCleanup := rooms.SeedBiomesForTest(map[string]*rooms.BiomeInfo{
		"default": {BiomeId: "default", Name: "Default"},
	})
	defer biomesCleanup()

	m := &mobs.Mob{}
	m.Character = characters.Character{}
	m.Character.Equipment.Body = items.Item{ItemId: 1}
	m.Character.Equipment.Head = items.Item{ItemId: 2}
	m.Character.Items = []items.Item{{ItemId: 3}}
	m.ItemDropChance = 100

	room := &rooms.Room{RoomId: 12345, Biome: "default"}

	dropMobLootAndSetCorpse(m, room)

	droppedIds := map[int]bool{}
	for _, itm := range room.Items {
		droppedIds[itm.ItemId] = true
	}

	assert.False(t, droppedIds[1], "NeverDrops equipped item must never drop to the floor")
	assert.True(t, droppedIds[2], "a normal equipped item should still drop")
	assert.True(t, droppedIds[3], "carried (guaranteed loot) Items must still drop even when other equipped gear is NeverDrops")
}
