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

	// Don't leave the focus quest dangling at a quest we just removed. This
	// fires on the common repeatable-quest-completion reset (and explicit quest
	// removal): without it, LastQuestId keeps pointing at a quest no longer in
	// progress, which broke hint (no arg) and leaves the web Quests panel /
	// minimap marker with no focus. Clearing to 0 lets GetFocusQuestId / hint
	// fall back deterministically to an active quest.
	if c.LastQuestId == questId {
		c.LastQuestId = 0
	}
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

// GetFocusQuestId returns the player's "focused" quest — the one hint (no arg),
// the web Quests panel, and the minimap marker all resolve against. It is
// LastQuestId when that quest is still in progress; otherwise (LastQuestId is
// zero, or points at a quest that has since been removed from progress) it
// falls back deterministically to the lowest-id quest still in progress that
// is not yet complete, then to the lowest-id quest of any kind, then 0. The
// determinism matters: the marker re-resolves on every emit, so a random
// fallback would make the focus jitter between quests. Pure — never mutates.
func (c *Character) GetFocusQuestId() int {
	prog := c.GetQuestProgress()
	if len(prog) == 0 {
		return 0
	}
	// A still-valid, still-ACTIVE explicit focus wins. We skip a LastQuestId
	// that sits at "end": completing a quest sets LastQuestId to it, so without
	// this a just-finished quest would stay the focus (hint/panel/marker) even
	// when the player has moved on to another active quest — e.g. after the
	// Two Roads bridge completes into Find Your Footing.
	if c.LastQuestId != 0 {
		if step, ok := prog[c.LastQuestId]; ok && step != "end" {
			return c.LastQuestId
		}
	}
	// Fallback: lowest-id quest still in progress and not at "end".
	best := 0
	for questId, step := range prog {
		if step == "end" {
			continue
		}
		if best == 0 || questId < best {
			best = questId
		}
	}
	if best != 0 {
		return best
	}
	// Everything is complete: lowest-id quest, so hint can still report it.
	for questId := range prog {
		if best == 0 || questId < best {
			best = questId
		}
	}
	return best
}
