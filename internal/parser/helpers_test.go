package parser

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitOnConnective(t *testing.T) {
	left, right, found := splitOnConnective("dagger to bank clerk", "to")
	require.True(t, found)
	assert.Equal(t, "dagger", left)
	assert.Equal(t, "bank clerk", right)

	_, _, found = splitOnConnective("dagger", "to")
	assert.False(t, found)
}

func TestResolveItem_FromCorpse(t *testing.T) {
	s, cleanup := seedParserTest(t)
	defer cleanup()
	s.Room.Corpses = []rooms.Corpse{{
		MobId:     1,
		Character: characters.Character{Name: "Skeleton"},
		Loot:      rooms.Container{Items: []items.Item{items.New(10001)}},
	}}

	// "sword corpse" — item from the trailing corpse container, no "from".
	m, ok := ResolveItem(s, "sword corpse")
	require.True(t, ok)
	assert.Equal(t, KindCorpse, m.Kind)   // resolved via the corpse container
	assert.Equal(t, 10001, m.Item.ItemId) // the looted item
	assert.Equal(t, 0, m.CorpseIdx)
}

func TestResolveItem_FromFloor(t *testing.T) {
	s, cleanup := seedParserTest(t)
	defer cleanup()
	s.Room.AddItem(items.New(40100), false)

	m, ok := ResolveItem(s, "lake iron nodule")
	require.True(t, ok)
	assert.Equal(t, KindFloorItem, m.Kind)
	assert.Equal(t, 40100, m.Item.ItemId)
}

func TestResolveActor_MultiWordMob(t *testing.T) {
	s, cleanup := seedParserTest(t)
	defer cleanup()
	m, ok := ResolveActor(s, "skeleton", KindMob, KindPlayer)
	require.True(t, ok)
	assert.Equal(t, 100, m.MobInstanceId)
}

func TestSplitTrailingContainer_Corpse(t *testing.T) {
	s, cleanup := seedParserTest(t)
	defer cleanup()
	s.Room.Corpses = []rooms.Corpse{{
		MobId:     1,
		Character: characters.Character{Name: "Skeleton"},
		Loot:      rooms.Container{Items: []items.Item{items.New(10001)}},
	}}

	// "sword corpse" (no "from") splits into item="sword", corpse container.
	itemPart, cm, ok := SplitTrailingContainer(s, "sword corpse")
	require.True(t, ok)
	assert.Equal(t, "sword", itemPart)
	assert.Equal(t, KindCorpse, cm.Kind)
	assert.Equal(t, 0, cm.CorpseIdx)
}

func TestSplitTrailingContainer_ExplicitFrom(t *testing.T) {
	s, cleanup := seedParserTest(t)
	defer cleanup()
	s.Room.Containers = map[string]rooms.Container{
		"wooden chest": {Items: []items.Item{items.New(10001)}},
	}
	itemPart, cm, ok := SplitTrailingContainer(s, "iron sword from wooden chest")
	require.True(t, ok)
	assert.Equal(t, "iron sword", itemPart)
	assert.Equal(t, KindRoomContainer, cm.Kind)
	assert.Equal(t, "wooden chest", cm.ContainerName)
}

func TestSplitTrailingContainer_NoContainer(t *testing.T) {
	s, cleanup := seedParserTest(t)
	defer cleanup()
	// Nothing trailing resolves to a container/corpse/pet.
	_, _, ok := SplitTrailingContainer(s, "lake iron nodule")
	assert.False(t, ok)
}

func TestSplitLeadingMatch(t *testing.T) {
	// Validator: "bank clerk" and "bank" are valid; nothing longer is.
	matches := func(c string) bool { return c == "bank clerk" || c == "bank" }

	// Longest valid leading span wins.
	head, tail, ok := SplitLeadingMatch("bank clerk smoketester", matches)
	require.True(t, ok)
	assert.Equal(t, "bank clerk", head)
	assert.Equal(t, "smoketester", tail)

	// Single-token head + trailing value.
	head, tail, ok = SplitLeadingMatch("bank smoketester extra", matches)
	require.True(t, ok)
	assert.Equal(t, "bank", head)
	assert.Equal(t, "smoketester extra", tail)

	// No leading span matches.
	_, _, ok = SplitLeadingMatch("nobody here", matches)
	assert.False(t, ok)
}
