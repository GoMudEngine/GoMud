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
