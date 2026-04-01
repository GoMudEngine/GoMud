package questengine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockActionContext tracks all actions executed for test assertions.
type mockActionContext struct {
	granted       []string
	consumedItems []int
	givenItems    []int
	givenGold     int
	sentTexts     []string
	roomTexts     []string
	spawnedMobs   []SpawnDef
	spawnedItems  []SpawnDef
	taughtSpells  []string
	appliedBuffs  []BuffDef
	teleported    int
	lockedExits   []ExitLock
	unlockedExits []ExitLock
	npcSays       []NpcSayDef
	sequences     []SequenceDef
	userId        int
}

func newMockActionContext(userId int) *mockActionContext {
	return &mockActionContext{userId: userId}
}

func (m *mockActionContext) GrantQuest(token string)            { m.granted = append(m.granted, token) }
func (m *mockActionContext) ConsumeItem(itemId int)             { m.consumedItems = append(m.consumedItems, itemId) }
func (m *mockActionContext) GiveItem(itemId int)                { m.givenItems = append(m.givenItems, itemId) }
func (m *mockActionContext) GiveGold(amount int)                { m.givenGold += amount }
func (m *mockActionContext) SendText(text string)               { m.sentTexts = append(m.sentTexts, text) }
func (m *mockActionContext) RoomText(text string)               { m.roomTexts = append(m.roomTexts, text) }
func (m *mockActionContext) SpawnMob(s SpawnDef)                { m.spawnedMobs = append(m.spawnedMobs, s) }
func (m *mockActionContext) SpawnItem(s SpawnDef)               { m.spawnedItems = append(m.spawnedItems, s) }
func (m *mockActionContext) TeachSpell(spellId string)          { m.taughtSpells = append(m.taughtSpells, spellId) }
func (m *mockActionContext) TrainSkill(skill string, level int) {}
func (m *mockActionContext) ApplyBuff(b BuffDef)                { m.appliedBuffs = append(m.appliedBuffs, b) }
func (m *mockActionContext) Teleport(roomId int)                { m.teleported = roomId }
func (m *mockActionContext) LockExits(e ExitLock)               { m.lockedExits = append(m.lockedExits, e) }
func (m *mockActionContext) UnlockExits(e ExitLock)             { m.unlockedExits = append(m.unlockedExits, e) }
func (m *mockActionContext) QueueNpcSay(n NpcSayDef)            { m.npcSays = append(m.npcSays, n) }
func (m *mockActionContext) QueueSequence(s SequenceDef)        { m.sequences = append(m.sequences, s) }
func (m *mockActionContext) GiveMutation()                      {}
func (m *mockActionContext) GetUserId() int                     { return m.userId }

func TestExecuteAction_Grant(t *testing.T) {
	ctx := newMockActionContext(1)
	err := ExecuteAction(ActionDef{Grant: "3-start"}, ctx)
	assert.NoError(t, err)
	assert.Equal(t, []string{"3-start"}, ctx.granted)
}

func TestExecuteAction_ConsumeItem(t *testing.T) {
	ctx := newMockActionContext(1)
	err := ExecuteAction(ActionDef{ConsumeItem: 14}, ctx)
	assert.NoError(t, err)
	assert.Equal(t, []int{14}, ctx.consumedItems)
}

func TestExecuteAction_GiveItem(t *testing.T) {
	ctx := newMockActionContext(1)
	err := ExecuteAction(ActionDef{GiveItem: 10005}, ctx)
	assert.NoError(t, err)
	assert.Equal(t, []int{10005}, ctx.givenItems)
}

func TestExecuteAction_GiveGold(t *testing.T) {
	ctx := newMockActionContext(1)
	err := ExecuteAction(ActionDef{GiveGold: 50}, ctx)
	assert.NoError(t, err)
	assert.Equal(t, 50, ctx.givenGold)
}

func TestExecuteAction_NpcSay(t *testing.T) {
	ctx := newMockActionContext(1)
	npc := &NpcSayDef{Mob: 79, Lines: []SayLineDef{{Text: "Hello"}}}
	err := ExecuteAction(ActionDef{NpcSay: npc}, ctx)
	assert.NoError(t, err)
	assert.Len(t, ctx.npcSays, 1)
	assert.Equal(t, 79, ctx.npcSays[0].Mob)
}

func TestExecuteAction_LockExits(t *testing.T) {
	ctx := newMockActionContext(1)
	lock := &ExitLock{Room: 113, PlayerScoped: true}
	err := ExecuteAction(ActionDef{LockExits: lock}, ctx)
	assert.NoError(t, err)
	assert.Len(t, ctx.lockedExits, 1)
	assert.True(t, ctx.lockedExits[0].PlayerScoped)
}

func TestExecuteAction_Teleport(t *testing.T) {
	ctx := newMockActionContext(1)
	err := ExecuteAction(ActionDef{Teleport: 200}, ctx)
	assert.NoError(t, err)
	assert.Equal(t, 200, ctx.teleported)
}

func TestExecuteAction_Sequence(t *testing.T) {
	ctx := newMockActionContext(1)
	seq := &SequenceDef{
		DelayBetween: 3,
		Lines:        []SayLineDef{{Text: "Line 1"}, {Text: "Line 2"}},
	}
	err := ExecuteAction(ActionDef{Sequence: seq}, ctx)
	assert.NoError(t, err)
	assert.Len(t, ctx.sequences, 1)
	assert.Equal(t, 2, len(ctx.sequences[0].Lines))
}

func TestExecuteAction_EmptyAction(t *testing.T) {
	ctx := newMockActionContext(1)
	err := ExecuteAction(ActionDef{}, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no recognized field")
}

func TestExecuteAction_MultipleActions(t *testing.T) {
	ctx := newMockActionContext(1)
	actions := []ActionDef{
		{Grant: "3-start"},
		{GiveGold: 25},
		{SendText: "Quest started!"},
	}
	for _, a := range actions {
		err := ExecuteAction(a, ctx)
		assert.NoError(t, err)
	}
	assert.Equal(t, []string{"3-start"}, ctx.granted)
	assert.Equal(t, 25, ctx.givenGold)
	assert.Equal(t, []string{"Quest started!"}, ctx.sentTexts)
}
