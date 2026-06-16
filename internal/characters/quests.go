package characters

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/quests"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func (c *Character) GetMemoryCapacity() int {
	// Map is now a free command; memory capacity based on Perception
	memCap := (c.Stats.Perception.ValueAdj >> 1)
	if memCap < 0 {
		memCap = 0
	}
	return memCap + 5
}

func (c *Character) GetMapSprawlCapacity() int {
	// Map is now a free command; sprawl capacity based on Perception
	sprawlCap := (c.Stats.Perception.ValueAdj >> 2)
	if sprawlCap < 0 {
		sprawlCap = 0
	}
	return sprawlCap
}

// Remember visiting a room. This may cause to forget an older room if the memory is full.
func (c *Character) RememberRoom(roomId int) {
	mapHistory := c.GetMemoryCapacity()
	if len(c.roomHistory) >= mapHistory*2 {
		// Prune out everything except {mapHistory}-1 items at the end
		c.roomHistory = c.roomHistory[len(c.roomHistory)-(mapHistory-1):]
	}
	c.roomHistory = append(c.roomHistory, roomId)
}

func (c *Character) IsQuestDone(questToken string) bool {
	testQuestId, _ := quests.TokenToParts(questToken)
	if c.QuestProgress == nil {
		c.QuestProgress = make(map[int]string)
	}

	stage := c.QuestProgress[testQuestId]

	return stage == `end`
}

func (c *Character) HasQuest(questToken string) bool {

	if c.QuestProgress == nil {
		c.QuestProgress = make(map[int]string)
	}

	testQuestId, testQuestStep := quests.TokenToParts(questToken)

	currentStep, ok := c.QuestProgress[testQuestId]
	if !ok {
		return false
	}

	// If on that step currently, then true
	if currentStep == testQuestStep {
		return true
	}

	currentToken := quests.PartsToToken(testQuestId, currentStep)

	// If the current token comes after the test token then they've already done that quest
	return quests.IsTokenAfter(questToken, currentToken)
}

func (c *Character) GetQuestProgress() map[int]string {

	if c.QuestProgress == nil {
		c.QuestProgress = make(map[int]string)
	}

	retMap := make(map[int]string)
	for questId, stepName := range c.QuestProgress {
		retMap[questId] = stepName
	}
	return retMap
}

func (c *Character) GiveQuestToken(questToken string) bool {

	if c.QuestProgress == nil {
		c.QuestProgress = make(map[int]string)
	}

	questId, newStep := quests.TokenToParts(questToken)
	currentProgress := c.QuestProgress[questId]

	currentToken := quests.PartsToToken(questId, currentProgress)

	if quests.IsTokenAfter(currentToken, questToken) {
		c.QuestProgress[questId] = newStep
		c.LastQuestId = questId
		return true
	}

	return false
}

func (c *Character) ClearQuestToken(questToken string) {

	if c.QuestProgress == nil {
		c.QuestProgress = make(map[int]string)
	}

	questId, _ := quests.TokenToParts(questToken)

	delete(c.QuestProgress, questId)
}

func (c *Character) SetQuestFlag(key, value string) {
	if c.QuestFlags == nil {
		c.QuestFlags = make(map[string]string)
	}
	// Runtime defense-in-depth: log if flag isn't in the registry.
	// Skip if registry is empty (test environment / data not loaded yet).
	if len(quests.GetFlagRegistry()) > 0 {
		if err := quests.ValidateFlag(key, value); err != nil {
			mudlog.Error("SetQuestFlag", "warning", err.Error())
		}
	}
	c.QuestFlags[key] = value
}

func (c *Character) GetQuestFlag(key string) string {
	if c.QuestFlags == nil {
		return ""
	}
	return c.QuestFlags[key]
}

func (c *Character) HasQuestFlag(key string) bool {
	if c.QuestFlags == nil {
		return false
	}
	_, ok := c.QuestFlags[key]
	return ok
}

func (c *Character) ClearQuestFlag(key string) {
	if c.QuestFlags == nil {
		return
	}
	delete(c.QuestFlags, key)
}

// questCooldownKey is the MiscData key under which a repeatable quest's
// "available again at round N" timestamp is stored.
func questCooldownKey(questId int) string {
	return fmt.Sprintf("questcd-%d", questId)
}

// SetQuestCooldown records that quest questId may not be re-taken until
// cooldownRounds rounds from now. Stored in persisted MiscData so the
// cooldown survives logout.
func (c *Character) SetQuestCooldown(questId int, cooldownRounds uint64) {
	c.SetMiscData(questCooldownKey(questId), util.GetRoundCount()+cooldownRounds)
}

// QuestCooldownActive reports whether quest questId is still inside its
// post-completion cooldown window (current round < stored "available at").
func (c *Character) QuestCooldownActive(questId int) bool {
	v := c.GetMiscData(questCooldownKey(questId))
	if v == nil {
		return false
	}
	return util.GetRoundCount() < miscDataToUint64(v)
}

// miscDataToUint64 coerces a MiscData value to uint64. MiscData stores any,
// and an integer written as uint64 may return from YAML reload as int,
// int64, or float64 — handle all of them; anything else yields 0.
func miscDataToUint64(v any) uint64 {
	switch n := v.(type) {
	case uint64:
		return n
	case int:
		if n < 0 {
			return 0
		}
		return uint64(n)
	case int64:
		if n < 0 {
			return 0
		}
		return uint64(n)
	case float64:
		if n < 0 {
			return 0
		}
		return uint64(n)
	default:
		return 0
	}
}
