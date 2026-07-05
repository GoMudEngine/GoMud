package questengine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockPlayer struct {
	quests     map[string]bool
	items      map[int]bool
	flags      map[string]string
	roomId     int
	gold       int
	masterwork bool
}

func newMockPlayer(roomId int) *mockPlayer {
	return &mockPlayer{
		quests: make(map[string]bool),
		items:  make(map[int]bool),
		flags:  make(map[string]string),
		roomId: roomId,
	}
}

func (m *mockPlayer) HasQuest(token string) bool         { return m.quests[token] }
func (m *mockPlayer) HasItem(itemId int) bool            { return m.items[itemId] }
func (m *mockPlayer) GetRoomId() int                     { return m.roomId }
func (m *mockPlayer) GetQuestFlag(key string) string     { return m.flags[key] }
func (m *mockPlayer) GetGold() int                       { return m.gold }
func (m *mockPlayer) HasOwnMasterwork(skillMin int) bool { return m.masterwork }

func TestEvalConditions_Empty(t *testing.T) {
	p := newMockPlayer(100)
	assert.True(t, EvalConditions(Conditions{}, p))
}

func TestEvalConditions_Has(t *testing.T) {
	p := newMockPlayer(100)
	p.quests["3-start"] = true
	assert.True(t, EvalConditions(Conditions{Has: []string{"3-start"}}, p))
	assert.False(t, EvalConditions(Conditions{Has: []string{"3-start", "3-totem"}}, p))
}

func TestEvalConditions_Missing(t *testing.T) {
	p := newMockPlayer(100)
	p.quests["3-start"] = true
	assert.False(t, EvalConditions(Conditions{Missing: []string{"3-start"}}, p))
	assert.True(t, EvalConditions(Conditions{Missing: []string{"3-totem"}}, p))
}

func TestEvalConditions_InRoom(t *testing.T) {
	p := newMockPlayer(100)
	assert.True(t, EvalConditions(Conditions{InRoom: 100}, p))
	assert.False(t, EvalConditions(Conditions{InRoom: 200}, p))
}

func TestEvalConditions_HasItem(t *testing.T) {
	p := newMockPlayer(100)
	p.items[14] = true
	assert.True(t, EvalConditions(Conditions{HasItem: 14}, p))
	assert.False(t, EvalConditions(Conditions{HasItem: 15}, p))
}

func TestEvalConditions_MissingItem(t *testing.T) {
	p := newMockPlayer(100)
	p.items[14] = true
	assert.False(t, EvalConditions(Conditions{MissingItem: 14}, p))
	assert.True(t, EvalConditions(Conditions{MissingItem: 15}, p))
}

func TestEvalConditions_Combined(t *testing.T) {
	p := newMockPlayer(100)
	p.quests["3-start"] = true
	p.items[14] = true
	c := Conditions{Has: []string{"3-start"}, Missing: []string{"3-end"}, InRoom: 100, HasItem: 14}
	assert.True(t, EvalConditions(c, p))
	c.InRoom = 200
	assert.False(t, EvalConditions(c, p))
}

func TestEvalConditions_HasFlag(t *testing.T) {
	p := newMockPlayer(100)
	p.flags["11-branch"] = "rhett"
	assert.True(t, EvalConditions(Conditions{HasFlag: map[string]string{"11-branch": "rhett"}}, p))
	assert.False(t, EvalConditions(Conditions{HasFlag: map[string]string{"11-branch": "sylara"}}, p))
}

func TestEvalConditions_MissingFlag(t *testing.T) {
	p := newMockPlayer(100)
	p.flags["11-branch"] = "rhett"
	assert.True(t, EvalConditions(Conditions{MissingFlag: map[string]string{"11-branch": "sylara"}}, p))
	assert.False(t, EvalConditions(Conditions{MissingFlag: map[string]string{"11-branch": "rhett"}}, p))
}

func TestEvalConditions_HasFlag_Missing(t *testing.T) {
	p := newMockPlayer(100)
	assert.False(t, EvalConditions(Conditions{HasFlag: map[string]string{"11-branch": "rhett"}}, p))
}
