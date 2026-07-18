package characters

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetFocusQuestId_ValidLastQuest(t *testing.T) {
	c := &Character{LastQuestId: 71, QuestProgress: map[int]string{65: "discover", 71: "start"}}
	assert.Equal(t, 71, c.GetFocusQuestId())
}

func TestGetFocusQuestId_StaleLastQuest_FallsBackToLowestActive(t *testing.T) {
	// LastQuestId points at a quest no longer in progress (the smoketester case).
	c := &Character{LastQuestId: 87, QuestProgress: map[int]string{
		2: "end", 71: "start", 65: "discover", 30: "end",
	}}
	// Lowest-id non-"end" quest = 65.
	assert.Equal(t, 65, c.GetFocusQuestId())
}

func TestGetFocusQuestId_ZeroLastQuest_FallsBack(t *testing.T) {
	c := &Character{LastQuestId: 0, QuestProgress: map[int]string{67: "start", 71: "start"}}
	assert.Equal(t, 67, c.GetFocusQuestId())
}

func TestGetFocusQuestId_AllComplete_LowestId(t *testing.T) {
	c := &Character{LastQuestId: 0, QuestProgress: map[int]string{2: "end", 30: "end"}}
	assert.Equal(t, 2, c.GetFocusQuestId())
}

func TestGetFocusQuestId_NoQuests(t *testing.T) {
	c := &Character{LastQuestId: 87, QuestProgress: map[int]string{}}
	assert.Equal(t, 0, c.GetFocusQuestId())
}
