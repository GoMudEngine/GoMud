package questengine

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/stretchr/testify/assert"
)

// BumpRepCall records a single BumpRep call for test assertions.
type BumpRepCall struct {
	Faction string
	Delta   int
}

// StatIncreaseCall records a single IncreaseStat call for test assertions.
type StatIncreaseCall struct {
	Stat   string
	Amount int
}

// mockActionContext tracks all actions executed for test assertions.
type mockActionContext struct {
	granted        []string
	consumedItems  []int
	givenItems     []int
	givenGold      int
	gold           int
	sentTexts      []string
	roomTexts      []string
	spawnedMobs    []SpawnDef
	spawnedItems   []SpawnDef
	taughtSpells   []string
	appliedBuffs   []BuffDef
	teleported     int
	lockedExits    []ExitLock
	unlockedExits  []ExitLock
	npcSays        []NpcSayDef
	sequences      []SequenceDef
	bumpedRep      []BumpRepCall
	increasedStats []StatIncreaseCall
	learnedRecipes []string
	userId         int
}

func newMockActionContext(userId int) *mockActionContext {
	return &mockActionContext{userId: userId}
}

func (m *mockActionContext) GrantQuest(token string) { m.granted = append(m.granted, token) }
func (m *mockActionContext) ConsumeItem(itemId int) {
	m.consumedItems = append(m.consumedItems, itemId)
}
func (m *mockActionContext) GiveItem(itemId int) { m.givenItems = append(m.givenItems, itemId) }
func (m *mockActionContext) GiveGold(amount int) { m.givenGold += amount }
func (m *mockActionContext) ChargeGold(amount int) {
	if amount > m.gold {
		amount = m.gold
	}
	m.gold -= amount
}
func (m *mockActionContext) SendText(_ messaging.Category, text string) {
	m.sentTexts = append(m.sentTexts, text)
}
func (m *mockActionContext) RoomText(text string) { m.roomTexts = append(m.roomTexts, text) }
func (m *mockActionContext) SpawnMob(s SpawnDef)  { m.spawnedMobs = append(m.spawnedMobs, s) }
func (m *mockActionContext) SpawnItem(s SpawnDef) { m.spawnedItems = append(m.spawnedItems, s) }
func (m *mockActionContext) TeachSpell(spellId string) {
	m.taughtSpells = append(m.taughtSpells, spellId)
}
func (m *mockActionContext) TrainSkill(skill string, level int) {}
func (m *mockActionContext) IncreaseStat(stat string, amount int) {
	m.increasedStats = append(m.increasedStats, StatIncreaseCall{Stat: stat, Amount: amount})
}
func (m *mockActionContext) LearnRecipe(recipe string) {
	m.learnedRecipes = append(m.learnedRecipes, recipe)
}
func (m *mockActionContext) ApplyBuff(b BuffDef)            { m.appliedBuffs = append(m.appliedBuffs, b) }
func (m *mockActionContext) Teleport(roomId int)            { m.teleported = roomId }
func (m *mockActionContext) LockExits(e ExitLock)           { m.lockedExits = append(m.lockedExits, e) }
func (m *mockActionContext) UnlockExits(e ExitLock)         { m.unlockedExits = append(m.unlockedExits, e) }
func (m *mockActionContext) QueueNpcSay(n NpcSayDef)        { m.npcSays = append(m.npcSays, n) }
func (m *mockActionContext) QueueSequence(s SequenceDef)    { m.sequences = append(m.sequences, s) }
func (m *mockActionContext) GiveMutation()                  {}
func (m *mockActionContext) SetQuestFlag(key, value string) {}
func (m *mockActionContext) BumpRep(factionId string, delta int) {
	m.bumpedRep = append(m.bumpedRep, BumpRepCall{Faction: factionId, Delta: delta})
}
func (m *mockActionContext) GetUserId() int { return m.userId }

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

func TestExecuteAction_BumpRep(t *testing.T) {
	ctx := newMockActionContext(17)
	a := ActionDef{BumpRep: &BumpRepDef{Faction: "warren", Delta: 30}}
	err := ExecuteAction(a, ctx)
	assert.NoError(t, err)
	assert.Equal(t, []BumpRepCall{{Faction: "warren", Delta: 30}}, ctx.bumpedRep)
}

func TestExecuteAction_TrainStat(t *testing.T) {
	ctx := newMockActionContext(5)
	a := ActionDef{TrainStat: &StatDef{Stat: "strength", Amount: 5}}
	err := ExecuteAction(a, ctx)
	assert.NoError(t, err)
	assert.Equal(t, []StatIncreaseCall{{Stat: "strength", Amount: 5}}, ctx.increasedStats)
}

func TestExecuteAction_TrainStat_Dexterity(t *testing.T) {
	ctx := newMockActionContext(5)
	a := ActionDef{TrainStat: &StatDef{Stat: "dexterity", Amount: 3}}
	err := ExecuteAction(a, ctx)
	assert.NoError(t, err)
	assert.Len(t, ctx.increasedStats, 1)
	assert.Equal(t, "dexterity", ctx.increasedStats[0].Stat)
	assert.Equal(t, 3, ctx.increasedStats[0].Amount)
}

func TestExecuteAction_LearnRecipe(t *testing.T) {
	ctx := newMockActionContext(7)
	a := ActionDef{LearnRecipe: &RecipeDef{Recipe: "iron-dagger"}}
	err := ExecuteAction(a, ctx)
	assert.NoError(t, err)
	assert.Equal(t, []string{"iron-dagger"}, ctx.learnedRecipes)
}

func TestExecuteAction_LearnRecipe_MultipleRecipes(t *testing.T) {
	ctx := newMockActionContext(7)
	recipes := []string{"iron-dagger", "iron-buckler"}
	for _, r := range recipes {
		err := ExecuteAction(ActionDef{LearnRecipe: &RecipeDef{Recipe: r}}, ctx)
		assert.NoError(t, err)
	}
	assert.Equal(t, recipes, ctx.learnedRecipes)
}
