package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
)

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
