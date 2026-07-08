package parser

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNounAdapter_MultiWord(t *testing.T) {
	s, cleanup := seedParserTest(t)
	defer cleanup()
	s.Room.Nouns = map[string]string{"hare paths": "Faint trails worn by hares."}

	m, ok := nounAdapter(s, "hare paths")
	require.True(t, ok)
	assert.Equal(t, KindNoun, m.Kind)
	assert.Equal(t, "hare paths", m.Name)

	_, ok = nounAdapter(s, "dragon")
	assert.False(t, ok)
}

func TestExitAdapter(t *testing.T) {
	s, cleanup := seedParserTest(t)
	defer cleanup()
	m, ok := exitAdapter(s, "north")
	require.True(t, ok)
	assert.Equal(t, KindExit, m.Kind)
	assert.Equal(t, "north", m.Name)
}

func TestContainerAdapter(t *testing.T) {
	s, cleanup := seedParserTest(t)
	defer cleanup()
	s.Room.Containers = map[string]rooms.Container{
		"wooden chest": {Items: []items.Item{items.New(10001)}},
	}
	m, ok := containerAdapter(s, "wooden chest")
	require.True(t, ok)
	assert.Equal(t, KindRoomContainer, m.Kind)
	assert.Equal(t, "wooden chest", m.ContainerName)
}

func TestCorpseAdapter(t *testing.T) {
	s, cleanup := seedParserTest(t)
	defer cleanup()
	s.Room.Corpses = []rooms.Corpse{{
		MobId:     1,
		Character: characters.Character{Name: "Skeleton"},
		Loot:      rooms.Container{Items: []items.Item{items.New(10001)}},
	}}
	m, ok := corpseAdapter(s, "corpse")
	require.True(t, ok)
	assert.Equal(t, KindCorpse, m.Kind)
	assert.Equal(t, 0, m.CorpseIdx)
}

func TestFloorItemAdapter(t *testing.T) {
	s, cleanup := seedParserTest(t)
	defer cleanup()
	s.Room.AddItem(items.New(40100), false) // "lake iron nodule"

	m, ok := floorItemAdapter(s, "lake iron nodule")
	require.True(t, ok)
	assert.Equal(t, KindFloorItem, m.Kind)
	assert.Equal(t, 40100, m.Item.ItemId)
}

func TestInventoryItemAdapter(t *testing.T) {
	s, cleanup := seedParserTest(t)
	defer cleanup()
	s.User.Character.StoreItem(items.New(10001)) // Iron Sword to backpack

	m, ok := inventoryItemAdapter(s, "iron sword")
	require.True(t, ok)
	assert.Equal(t, KindInventoryItem, m.Kind)
	assert.Equal(t, 10001, m.Item.ItemId)
}

func TestPotionItemAdapter(t *testing.T) {
	s, cleanup := seedParserTest(t)
	defer cleanup()
	s.User.Character.PotionItems = append(s.User.Character.PotionItems, items.New(30001))

	m, ok := potionItemAdapter(s, "healing potion")
	require.True(t, ok)
	assert.Equal(t, KindPotionItem, m.Kind)
	assert.Equal(t, 30001, m.Item.ItemId)
}
