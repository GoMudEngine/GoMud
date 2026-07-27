package questengine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// fullMockPlayer is a player mock whose quest/item state can be mutated by
// the action context, so chained triggers see updated state immediately.
type fullMockPlayer struct {
	quests     map[string]bool
	items      map[int]bool
	flags      map[string]string
	roomId     int
	gold       int
	masterwork bool
}

func newFullMockPlayer(roomId int) *fullMockPlayer {
	return &fullMockPlayer{
		quests: make(map[string]bool),
		items:  make(map[int]bool),
		flags:  make(map[string]string),
		roomId: roomId,
	}
}

func (p *fullMockPlayer) HasQuest(token string) bool         { return p.quests[token] }
func (p *fullMockPlayer) HasItem(itemId int) bool            { return p.items[itemId] }
func (p *fullMockPlayer) GetRoomId() int                     { return p.roomId }
func (p *fullMockPlayer) GetQuestFlag(key string) string     { return p.flags[key] }
func (p *fullMockPlayer) GetGold() int                       { return p.gold }
func (p *fullMockPlayer) HasOwnMasterwork(skillMin int) bool { return p.masterwork }

// fullMockActionContext wraps mockActionContext but mutates the player's state
// on GrantQuest, ConsumeItem, and GiveItem so chained evaluation works.
type fullMockActionContext struct {
	*mockActionContext
	player *fullMockPlayer
}

func newFullMockActionContext(userId int, player *fullMockPlayer) *fullMockActionContext {
	return &fullMockActionContext{
		mockActionContext: newMockActionContext(userId),
		player:            player,
	}
}

func (c *fullMockActionContext) GrantQuest(token string) {
	c.mockActionContext.GrantQuest(token)
	c.player.quests[token] = true
}

func (c *fullMockActionContext) ConsumeItem(itemId int) {
	c.mockActionContext.ConsumeItem(itemId)
	delete(c.player.items, itemId)
}

func (c *fullMockActionContext) GiveItem(itemId int) {
	c.mockActionContext.GiveItem(itemId)
	c.player.items[itemId] = true
}

func (m *fullMockActionContext) SetQuestFlag(key, value string) {
	m.player.flags[key] = value
}

// seedTestQuests creates an Engine loaded with the Scholar's Collection quest (quest 3).
// Steps: start, totem, sac, end (Linear: false)
// Triggers:
//   - dialogue topic "collection" on mob 79 → grants 3-start
//   - item_give item 14 to mob 79 → grants 3-totem (requires 3-start, missing 3-totem)
//   - item_give item 40008 to mob 79 → grants 3-sac (requires 3-start, missing 3-sac)
//   - quest_granted token 3-totem (requires 3-sac) → grants 3-end
//   - quest_granted token 3-sac (requires 3-totem) → grants 3-end
func seedTestQuests() *Engine {
	q := &QuestDef{
		QuestId:     3,
		Name:        "The Scholar's Collection",
		Description: "Collect items for the scholar.",
		Steps: []QuestStep{
			{Id: "3-start", Description: "Speak to the scholar"},
			{Id: "3-totem", Description: "Deliver the totem"},
			{Id: "3-sac", Description: "Deliver the sacred text"},
			{Id: "3-end", Description: "Quest complete"},
		},
		Triggers: []TriggerDef{
			// Trigger 0: dialogue start
			{
				Event: "dialogue",
				Mob:   79,
				Topic: "collection",
				Conditions: Conditions{
					Missing: []string{"3-start"},
				},
				Actions: []ActionDef{
					{SendText: "The scholar nods eagerly."},
					{Grant: "3-start"},
				},
			},
			// Trigger 1: item_give totem (item 14) to scholar (mob 79)
			{
				Event: "item_give",
				Mob:   79,
				Item:  14,
				Conditions: Conditions{
					Has:     []string{"3-start"},
					Missing: []string{"3-totem"},
				},
				Actions: []ActionDef{
					{ConsumeItem: 14},
					{Grant: "3-totem"},
					{SendText: "The scholar accepts the totem."},
				},
			},
			// Trigger 2: item_give sacred text (item 40008) to scholar (mob 79)
			{
				Event: "item_give",
				Mob:   79,
				Item:  40008,
				Conditions: Conditions{
					Has:     []string{"3-start"},
					Missing: []string{"3-sac"},
				},
				Actions: []ActionDef{
					{ConsumeItem: 40008},
					{Grant: "3-sac"},
					{SendText: "The scholar carefully takes the text."},
				},
			},
			// Trigger 3: chain completion when totem delivered (needs sac already done)
			{
				Event:      "quest_granted",
				QuestToken: "3-totem",
				Conditions: Conditions{
					Has: []string{"3-sac"},
				},
				Actions: []ActionDef{
					{Grant: "3-end"},
					{GiveGold: 100},
					{SendText: "The scholar's collection is complete!"},
				},
			},
			// Trigger 4: chain completion when sac delivered (needs totem already done)
			{
				Event:      "quest_granted",
				QuestToken: "3-sac",
				Conditions: Conditions{
					Has: []string{"3-totem"},
				},
				Actions: []ActionDef{
					{Grant: "3-end"},
					{GiveGold: 100},
					{SendText: "The scholar's collection is complete!"},
				},
			},
		},
	}

	e := NewEngine()
	e.RegisterQuest(q)
	return e
}

func TestEngine_BasicTrigger(t *testing.T) {
	e := seedTestQuests()
	player := newFullMockPlayer(100)
	ctx := newFullMockActionContext(1, player)

	result := e.Notify("dialogue", EventDetails{
		UserId: 1,
		MobId:  79,
		Topic:  "collection",
	}, player, ctx)

	assert.True(t, result.Handled)
	assert.True(t, player.HasQuest("3-start"))
	assert.Contains(t, ctx.granted, "3-start")
	assert.Contains(t, ctx.sentTexts, "The scholar nods eagerly.")
}

func TestEngine_NoMatch(t *testing.T) {
	e := seedTestQuests()
	player := newFullMockPlayer(100)
	ctx := newFullMockActionContext(1, player)

	// Wrong topic
	result := e.Notify("dialogue", EventDetails{
		UserId: 1,
		MobId:  79,
		Topic:  "weather",
	}, player, ctx)
	assert.False(t, result.Handled)

	// Wrong mob
	result = e.Notify("dialogue", EventDetails{
		UserId: 1,
		MobId:  50,
		Topic:  "collection",
	}, player, ctx)
	assert.False(t, result.Handled)

	// No triggers at all for this event type
	result = e.Notify("room_enter", EventDetails{
		UserId: 1,
		RoomId: 100,
	}, player, ctx)
	assert.False(t, result.Handled)
}

func TestEngine_ConditionGating(t *testing.T) {
	e := seedTestQuests()
	player := newFullMockPlayer(100)
	player.items[14] = true
	ctx := newFullMockActionContext(1, player)

	// Try to give totem without having 3-start — should not match
	result := e.Notify("item_give", EventDetails{
		UserId: 1,
		MobId:  79,
		ItemId: 14,
	}, player, ctx)
	assert.False(t, result.Handled)
	assert.False(t, player.HasQuest("3-totem"))

	// Now grant 3-start and try again
	player.quests["3-start"] = true
	ctx2 := newFullMockActionContext(1, player)

	result = e.Notify("item_give", EventDetails{
		UserId: 1,
		MobId:  79,
		ItemId: 14,
	}, player, ctx2)
	assert.True(t, result.Handled)
	assert.True(t, player.HasQuest("3-totem"))
}

func TestEngine_ChainedGrantCompletion(t *testing.T) {
	e := seedTestQuests()
	player := newFullMockPlayer(100)
	player.quests["3-start"] = true
	player.quests["3-totem"] = true // totem already delivered
	player.items[40008] = true
	ctx := newFullMockActionContext(1, player)

	// Give sacred text — should grant 3-sac, which chains to grant 3-end
	result := e.Notify("item_give", EventDetails{
		UserId: 1,
		MobId:  79,
		ItemId: 40008,
	}, player, ctx)

	assert.True(t, result.Handled)
	assert.True(t, player.HasQuest("3-sac"), "3-sac should be granted")
	assert.True(t, player.HasQuest("3-end"), "3-end should be granted via chain")
	assert.Contains(t, ctx.granted, "3-sac")
	assert.Contains(t, ctx.granted, "3-end")
	assert.Equal(t, 100, ctx.givenGold)
}

func TestEngine_DepthGuard(t *testing.T) {
	// Create a circular chain: quest A grants token that triggers quest B,
	// which grants token that triggers quest A again.
	e := NewEngine()
	qA := &QuestDef{
		QuestId: 100,
		Name:    "Circular A",
		Steps:   []QuestStep{{Id: "100-a"}},
		Triggers: []TriggerDef{
			{
				Event: "dialogue",
				Topic: "start-loop",
				Actions: []ActionDef{
					{Grant: "100-a"},
				},
			},
			{
				Event:      "quest_granted",
				QuestToken: "101-b",
				Actions: []ActionDef{
					{Grant: "100-a"},
				},
			},
		},
	}
	qB := &QuestDef{
		QuestId: 101,
		Name:    "Circular B",
		Steps:   []QuestStep{{Id: "101-b"}},
		Triggers: []TriggerDef{
			{
				Event:      "quest_granted",
				QuestToken: "100-a",
				Actions: []ActionDef{
					{Grant: "101-b"},
				},
			},
		},
	}

	e.RegisterQuest(qA)
	e.RegisterQuest(qB)

	player := newFullMockPlayer(100)
	ctx := newFullMockActionContext(1, player)

	// Should not crash or infinite loop — guard stops it
	result := e.Notify("dialogue", EventDetails{
		UserId: 1,
		Topic:  "start-loop",
	}, player, ctx)

	assert.True(t, result.Handled)
	// Both tokens get granted once each (the circular re-grant is blocked by MarkGranted)
	assert.True(t, player.HasQuest("100-a"))
	assert.True(t, player.HasQuest("101-b"))
}

func TestEngine_DuplicateGrantNoop(t *testing.T) {
	e := seedTestQuests()
	player := newFullMockPlayer(100)
	player.quests["3-start"] = true // Already has start

	ctx := newFullMockActionContext(1, player)

	// Try dialogue again — condition missing:3-start blocks it
	result := e.Notify("dialogue", EventDetails{
		UserId: 1,
		MobId:  79,
		Topic:  "collection",
	}, player, ctx)

	assert.False(t, result.Handled)
	assert.Empty(t, ctx.granted)
}

func TestEngine_ItemGiveConsumeItem(t *testing.T) {
	e := seedTestQuests()
	player := newFullMockPlayer(100)
	player.quests["3-start"] = true
	player.items[14] = true
	ctx := newFullMockActionContext(1, player)

	result := e.Notify("item_give", EventDetails{
		UserId: 1,
		MobId:  79,
		ItemId: 14,
	}, player, ctx)

	assert.True(t, result.Handled)
	assert.True(t, result.ConsumeItem, "ConsumeItem should be true for item_give with consume_item action")
	assert.Contains(t, ctx.consumedItems, 14)
}

func TestEngine_MultipleTriggersCanMatch(t *testing.T) {
	e := seedTestQuests()
	player := newFullMockPlayer(100)
	player.quests["3-start"] = true

	// Deliver totem first
	player.items[14] = true
	ctx1 := newFullMockActionContext(1, player)
	result := e.Notify("item_give", EventDetails{
		UserId: 1,
		MobId:  79,
		ItemId: 14,
	}, player, ctx1)
	assert.True(t, result.Handled)
	assert.True(t, player.HasQuest("3-totem"))

	// Deliver sacred text second
	player.items[40008] = true
	ctx2 := newFullMockActionContext(1, player)
	result = e.Notify("item_give", EventDetails{
		UserId: 1,
		MobId:  79,
		ItemId: 40008,
	}, player, ctx2)
	assert.True(t, result.Handled)
	assert.True(t, player.HasQuest("3-sac"))
	// Should also chain to 3-end since totem was already done
	assert.True(t, player.HasQuest("3-end"))
}

func TestEngine_RegisterAndGetQuest(t *testing.T) {
	e := NewEngine()
	assert.Nil(t, e.GetQuest(3))

	q := &QuestDef{QuestId: 3, Name: "Test"}
	e.RegisterQuest(q)
	assert.NotNil(t, e.GetQuest(3))
	assert.Equal(t, "Test", e.GetQuest(3).Name)
}

// seedCommandIssuedQuest creates an Engine loaded with a two-step quest that
// advances on the non-intercepting "command_issued" event — proving quests
// can gate a step on "the player typed command X [noun]" without any
// per-handler instrumentation.
// Steps: 200-start, 200-look (Linear: false)
// Triggers:
//   - command_issued command "look" noun "guide" (requires 200-start,
//     missing 200-look) → grants 200-look
func seedCommandIssuedQuest() *Engine {
	q := &QuestDef{
		QuestId: 200,
		Name:    "Command Issued Test Quest",
		Steps: []QuestStep{
			{Id: "200-start", Description: "Begin"},
			{Id: "200-look", Description: "Look at the guide"},
		},
		Triggers: []TriggerDef{
			{
				Event:   "command_issued",
				Command: "look",
				Noun:    "guide",
				Conditions: Conditions{
					Has:     []string{"200-start"},
					Missing: []string{"200-look"},
				},
				Actions: []ActionDef{
					{Grant: "200-look"},
				},
			},
		},
	}

	e := NewEngine()
	e.RegisterQuest(q)
	return e
}

func TestCommandIssuedTriggerMatchesNoun(t *testing.T) {
	e := seedCommandIssuedQuest()
	player := newFullMockPlayer(100)
	player.quests["200-start"] = true // grant prior step first
	ctx := newFullMockActionContext(1, player)

	// Wrong noun should not advance the quest
	result := e.Notify("command_issued", EventDetails{
		UserId:  1,
		Command: "look",
		Noun:    "grey",
	}, player, ctx)
	assert.False(t, result.Handled)
	assert.False(t, player.HasQuest("200-look"))

	// Matching command + noun should advance the quest
	ctx2 := newFullMockActionContext(1, player)
	result = e.Notify("command_issued", EventDetails{
		UserId:  1,
		Command: "look",
		Noun:    "guide",
	}, player, ctx2)
	assert.True(t, result.Handled)
	assert.True(t, player.HasQuest("200-look"))
}
