package dialogue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ─── Integration: Dialogue with Quest Gates ──────────────────────────────────
// Exercises: Match(), checkQuestGate(), applyQuestEffects(), PlayerState

func TestIntegration_DialogueWithQuestGates(t *testing.T) {
	// Build dialogue file with quest-gated patterns
	df := &DialogueFile{
		MobId:       1,
		Zone:        "test",
		DefaultMood: "neutral",
		Patterns: []Pattern{
			{
				Keywords:      []string{"quest", "task"},
				Responses:     []string{"I have a task for you!"},
				GrantsQuest:   "test-quest",
				QuestExcluded: []string{"test-quest"},
			},
			{
				Keywords:      []string{"quest", "task"},
				Responses:     []string{"You already have my quest."},
				QuestRequired: []string{"test-quest"},
			},
			{
				Keywords:  []string{"hello"},
				Responses: []string{"Greetings, traveler."},
			},
			{
				Keywords:  []string{""},
				Responses: []string{"I don't understand."},
			},
		},
	}

	// Track quests granted
	quests := map[string]bool{}
	ps := &PlayerState{
		HasQuest:   func(token string) bool { return quests[token] },
		HasItem:    func(itemId int) bool { return false },
		RemoveItem: func(itemId int) bool { return false },
		GiveQuest:  func(token string) { quests[token] = true },
	}

	// Before having the quest, asking about "quest" should offer it
	response, _, matched := Match(df, 1, "quest", ps)
	assert.True(t, matched, "Should match quest pattern")
	assert.Equal(t, "I have a task for you!", response)

	// Quest should have been granted
	assert.True(t, quests["test-quest"], "Quest should be granted after match")

	// Now that player has the quest, same topic should match different pattern
	response, _, matched = Match(df, 1, "task", ps)
	assert.True(t, matched, "Should still match quest pattern")
	assert.Equal(t, "You already have my quest.", response)

	// Non-quest topic should work normally
	response, _, matched = Match(df, 1, "hello", ps)
	assert.True(t, matched)
	assert.Equal(t, "Greetings, traveler.", response)

	// Unknown topic should hit fallback
	response, _, matched = Match(df, 1, "xyzzy", ps)
	assert.True(t, matched, "Fallback pattern should match")
	assert.Equal(t, "I don't understand.", response)

	// Nil PlayerState should skip quest checks (backward compat)
	response, _, matched = Match(df, 1, "quest", nil)
	assert.True(t, matched, "Should match with nil PlayerState")
}

func TestIntegration_DialogueMoodFiltering(t *testing.T) {
	df := &DialogueFile{
		MobId:       2,
		Zone:        "test",
		DefaultMood: "neutral",
		Patterns: []Pattern{
			{
				Keywords:  []string{"help"},
				Moods:     []string{"friendly"},
				Responses: []string{"Of course, I'll help you!"},
			},
			{
				Keywords:  []string{"help"},
				Moods:     []string{"hostile"},
				Responses: []string{"Get away from me!"},
			},
			{
				Keywords:  []string{"help"},
				Responses: []string{"Maybe..."},
			},
		},
	}

	// Default mood is neutral — mood-filtered patterns should be skipped
	response, _, matched := Match(df, 2, "help", nil)
	assert.True(t, matched)
	assert.Equal(t, "Maybe...", response, "Neutral mood should match unfiltered pattern")

	// Set mood to friendly
	SetMood(2, MoodFriendly)
	response, _, matched = Match(df, 2, "help", nil)
	assert.True(t, matched)
	assert.Equal(t, "Of course, I'll help you!", response,
		"Friendly mood should match friendly-filtered pattern")

	// Set mood to hostile
	SetMood(2, MoodHostile)
	response, _, matched = Match(df, 2, "help", nil)
	assert.True(t, matched)
	assert.Equal(t, "Get away from me!", response,
		"Hostile mood should match hostile-filtered pattern")

	// Clean up mood cache
	delete(moodCache, 2)
}

// ─── Integration: Dialogue Tree Progression ──────────────────────────────────
// Exercises: Greet(), TreeAdvance(), GetMemory(), UpdateMemory()

func TestIntegration_DialogueTreeProgression(t *testing.T) {
	df := &DialogueFile{
		MobId:       3,
		Zone:        "test",
		DefaultMood: "neutral",
		Tree: &Tree{
			Root: TreeRoot{
				Text:  "Welcome, stranger.",
				Hints: "You could ask about the village or the forest.",
			},
			Nodes: []TreeNode{
				{
					Id:       "village",
					Triggers: []string{"village", "town"},
					Text:     "The village is peaceful.",
					Hints:    "Perhaps ask about the elder.",
					Unlocks:  []string{"elder"},
				},
				{
					Id:       "elder",
					Triggers: []string{"elder"},
					Requires: []string{"village"},
					Text:     "The elder knows many secrets.",
					Hints:    "Ask about the forest.",
					Unlocks:  []string{"forest"},
				},
				{
					Id:       "forest",
					Triggers: []string{"forest"},
					Requires: []string{"elder"},
					Text:     "The forest is dark and dangerous.",
				},
			},
		},
	}

	userId := 100

	// Step 1: Greet should return root text
	greetText, hints, ok := Greet(df, 3, userId, nil)
	assert.True(t, ok, "Greet should succeed with tree defined")
	assert.Equal(t, "Welcome, stranger.", greetText)
	assert.Equal(t, "You could ask about the village or the forest.", hints)

	// Step 2: Ask about "forest" — should fail (requires "elder" which requires "village")
	_, _, _, advanced := TreeAdvance(df, 3, userId, "forest", nil)
	assert.False(t, advanced, "Forest should not advance without prerequisites")

	// Step 3: Ask about "village" — should succeed and unlock "elder"
	text, hints, _, advanced := TreeAdvance(df, 3, userId, "village", nil)
	assert.True(t, advanced, "Village should advance")
	assert.Equal(t, "The village is peaceful.", text)
	assert.Equal(t, "Perhaps ask about the elder.", hints)

	// Verify memory tracks the unlock
	mem := GetMemory(3, userId)
	assert.True(t, mem.UnlockedNodes["village"], "Village node should be marked visited")
	assert.True(t, mem.UnlockedNodes["elder"], "Elder node should be unlocked by village")

	// Step 4: Ask about "elder" — should now succeed (village is unlocked)
	text, _, _, advanced = TreeAdvance(df, 3, userId, "elder", nil)
	assert.True(t, advanced, "Elder should advance now that village is unlocked")
	assert.Equal(t, "The elder knows many secrets.", text)

	// Step 5: Ask about "forest" — should now succeed (elder is unlocked)
	text, _, _, advanced = TreeAdvance(df, 3, userId, "forest", nil)
	assert.True(t, advanced, "Forest should advance now that elder is unlocked")
	assert.Equal(t, "The forest is dark and dangerous.", text)

	// Clean up
	ResetMemory(3, userId)
	delete(moodCache, 3)
}

func TestIntegration_DialogueTreeWithQuestGates(t *testing.T) {
	df := &DialogueFile{
		MobId:       4,
		Zone:        "test",
		DefaultMood: "neutral",
		Tree: &Tree{
			Root: TreeRoot{
				Text: "Hello there.",
				Variants: []QuestGreeting{
					{
						QuestRequired: []string{"saved-village"},
						Text:          "Thank you for saving us!",
						Hints:         "The elder wishes to reward you.",
					},
				},
			},
			Nodes: []TreeNode{
				{
					Id:            "fetch-quest",
					Triggers:      []string{"quest", "task"},
					Text:          "Bring me 5 herbs.",
					GrantsQuest:   "herb-quest",
					QuestExcluded: []string{"herb-quest"},
				},
				{
					Id:            "fetch-complete",
					Triggers:      []string{"quest", "task"},
					Text:          "You already accepted my task!",
					QuestRequired: []string{"herb-quest"},
				},
			},
		},
	}

	userId := 200
	quests := map[string]bool{}
	ps := &PlayerState{
		HasQuest:   func(token string) bool { return quests[token] },
		HasItem:    func(itemId int) bool { return false },
		RemoveItem: func(itemId int) bool { return false },
		GiveQuest:  func(token string) { quests[token] = true },
	}

	// Greet without quest — should get default root
	greetText, _, ok := Greet(df, 4, userId, ps)
	assert.True(t, ok)
	assert.Equal(t, "Hello there.", greetText)

	// Ask about quest — should grant herb-quest
	text, _, _, advanced := TreeAdvance(df, 4, userId, "quest", ps)
	assert.True(t, advanced)
	assert.Equal(t, "Bring me 5 herbs.", text)
	assert.True(t, quests["herb-quest"], "herb-quest should be granted")

	// Ask about quest again — should get different node
	text, _, _, advanced = TreeAdvance(df, 4, userId, "task", ps)
	assert.True(t, advanced)
	assert.Equal(t, "You already accepted my task!", text)

	// Greet with saved-village quest — should get variant greeting
	quests["saved-village"] = true
	greetText, hints, ok := Greet(df, 4, userId, ps)
	assert.True(t, ok)
	assert.Equal(t, "Thank you for saving us!", greetText)
	assert.Equal(t, "The elder wishes to reward you.", hints)

	// Clean up
	ResetMemory(4, userId)
	delete(moodCache, 4)
}

func TestIntegration_DialogueItemRequirement(t *testing.T) {
	df := &DialogueFile{
		MobId:       5,
		Zone:        "test",
		DefaultMood: "neutral",
		Patterns: []Pattern{
			{
				Keywords:     []string{"trade"},
				Responses:    []string{"Thank you for the gem!"},
				RequiresItem: 42,
				MoodChange:   "grateful",
			},
			{
				Keywords:  []string{"trade"},
				Responses: []string{"You don't have what I need."},
			},
		},
	}

	hasGem := true
	removedItem := false
	ps := &PlayerState{
		HasQuest:   func(token string) bool { return false },
		HasItem:    func(itemId int) bool { return hasGem && itemId == 42 },
		RemoveItem: func(itemId int) bool { removedItem = true; hasGem = false; return true },
		GiveQuest:  func(token string) {},
	}

	// With item — should match first pattern and consume
	response, moodChange, matched := Match(df, 5, "trade", ps)
	assert.True(t, matched)
	assert.Equal(t, "Thank you for the gem!", response)
	assert.Equal(t, "grateful", moodChange)
	assert.True(t, removedItem, "Item should be consumed")

	// Without item — should match second pattern
	response, _, matched = Match(df, 5, "trade", ps)
	assert.True(t, matched)
	assert.Equal(t, "You don't have what I need.", response)

	delete(moodCache, 5)
}

func TestIntegration_MoodManagement(t *testing.T) {
	mobId := 99

	// Default mood
	mood := GetMood(mobId, "neutral")
	assert.Equal(t, MoodNeutral, mood)

	// Set mood directly
	SetMood(mobId, MoodFriendly)
	mood = GetMood(mobId, "neutral")
	assert.Equal(t, MoodFriendly, mood)

	// Shift mood
	ShiftMood(mobId, "hostile", "neutral")
	mood = GetMood(mobId, "neutral")
	assert.Equal(t, MoodHostile, mood)

	// Shift with empty target should not change
	ShiftMood(mobId, "", "neutral")
	mood = GetMood(mobId, "neutral")
	assert.Equal(t, MoodHostile, mood, "Empty shift should not change mood")

	// Clean up
	delete(moodCache, mobId)
}

// TestTreeAdvance_SubstringShadowAndReorderFix reproduces the Bloom Trail
// trigger-collision bug and proves the reorder fix. TreeAdvance matches nodes
// in file order using SUBSTRING matching (strings.Contains(topic, trigger)), so
// an EARLIER lore node with a short trigger that is an incidental substring of
// the topic shadows a later gated grant node. Real instance: a "work" node with
// trigger "live" shadowed "bloom_delivery" because "deLIVEry" contains "live".
// The fix is to place the gated grant node FIRST.
func TestTreeAdvance_SubstringShadowAndReorderFix(t *testing.T) {
	loreNode := TreeNode{
		Id:       "lore",
		Triggers: []string{"live"}, // substring of "delivery"
		Text:     "lore text",
	}
	grantNode := TreeNode{
		Id:            "grant",
		Triggers:      []string{"delivery"},
		QuestRequired: []string{"prereq"},
		GrantsQuest:   "next-token",
		Text:          "grant text",
	}

	makePS := func(grantedOut *string) *PlayerState {
		return &PlayerState{
			HasQuest:  func(tok string) bool { return tok == "prereq" },
			GiveQuest: func(tok string) { *grantedOut = tok },
		}
	}

	// BUGGY ORDER: lore before grant. "delivery" hits lore via the "live"
	// substring; the grant node is shadowed and never fires.
	buggy := &DialogueFile{MobId: 7777, Zone: "test", DefaultMood: "neutral",
		Tree: &Tree{Nodes: []TreeNode{loreNode, grantNode}}}
	var grantedBuggy string
	text, _, _, advanced := TreeAdvance(buggy, 7777, 8888, "delivery", makePS(&grantedBuggy))
	assert.True(t, advanced, "some node advances")
	assert.Equal(t, "lore text", text, "BUG: lore node shadows the grant via 'live' substring")
	assert.Equal(t, "", grantedBuggy, "BUG: grant never fires, no token granted")

	// FIXED ORDER: grant before lore. The gated grant node matches "delivery"
	// first and fires; non-quest players still fall through (gate skips it).
	fixed := &DialogueFile{MobId: 7779, Zone: "test", DefaultMood: "neutral",
		Tree: &Tree{Nodes: []TreeNode{grantNode, loreNode}}}
	var grantedFixed string
	text, _, _, advanced = TreeAdvance(fixed, 7779, 8889, "delivery", makePS(&grantedFixed))
	assert.True(t, advanced, "grant node advances")
	assert.Equal(t, "grant text", text, "FIX: grant-first node wins over the lore substring")
	assert.Equal(t, "next-token", grantedFixed, "FIX: the quest token is granted")

	// Non-quest player on the fixed order: grant gate fails, falls through to lore.
	noQuest := &PlayerState{HasQuest: func(string) bool { return false }, GiveQuest: func(string) {}}
	text, _, _, advanced = TreeAdvance(fixed, 7781, 8890, "delivery", noQuest)
	assert.True(t, advanced, "non-quest player still gets a node")
	assert.Equal(t, "lore text", text, "non-quest player falls through to lore (no regression)")
}
