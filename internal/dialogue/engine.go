package dialogue

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/util"
)

// checkQuestGate returns true if the player satisfies quest/item/flag conditions.
// When ps is nil all checks pass (backward compat for mob-to-mob or non-user contexts).
func checkQuestGate(questRequired, questExcluded []string, requiresItem int, flagRequired, flagExcluded map[string]string, ps *PlayerState) bool {
	if ps == nil {
		return true
	}

	for _, token := range questRequired {
		if !ps.HasQuest(token) {
			return false
		}
	}

	for _, token := range questExcluded {
		if ps.HasQuest(token) {
			return false
		}
	}

	if requiresItem > 0 && !ps.HasItem(requiresItem) {
		return false
	}

	// Quest flag checks
	if ps.GetQuestFlag != nil {
		for key, val := range flagRequired {
			if ps.GetQuestFlag(key) != val {
				return false
			}
		}
		for key, val := range flagExcluded {
			if ps.GetQuestFlag(key) == val {
				return false
			}
		}
	}

	return true
}

// applyQuestEffects fires quest grants, item consumption, item giving, and flag sets after a node/pattern matches.
func applyQuestEffects(grantsQuest string, requiresItem int, givesItem int, flagSet *QuestFlagSet, ps *PlayerState) {
	if ps == nil {
		return
	}
	if requiresItem > 0 {
		ps.RemoveItem(requiresItem)
	}
	if grantsQuest != "" {
		ps.GiveQuest(grantsQuest)
	}
	if givesItem > 0 && ps.GiveItem != nil {
		ps.GiveItem(givesItem)
	}
	if flagSet != nil && ps.SetQuestFlag != nil {
		ps.SetQuestFlag(flagSet.Key, flagSet.Value)
	}
}

// Match checks patterns against topic text for the given mob instance.
// It respects the mob's current mood when patterns declare mood filters.
// Returns (responseText, moodChange, matched).
// An empty-keyword pattern acts as the fallback when no specific keyword fires.
// When ps is nil, quest/item checks are skipped.
func Match(df *DialogueFile, mobInstanceId int, topic string, ps *PlayerState) (string, string, bool) {
	topic = strings.ToLower(topic)
	currentMood := GetMood(mobInstanceId, df.DefaultMood)

	var defaultPattern *Pattern
	var matched *Pattern

	for i := range df.Patterns {
		p := &df.Patterns[i]

		// Apply mood filter when specified
		if len(p.Moods) > 0 {
			moodMatch := false
			for _, m := range p.Moods {
				if Mood(m) == currentMood {
					moodMatch = true
					break
				}
			}
			if !moodMatch {
				continue
			}
		}

		// Apply quest/item gate
		if !checkQuestGate(p.QuestRequired, p.QuestExcluded, p.RequiresItem, p.QuestFlagRequired, p.QuestFlagExcluded, ps) {
			continue
		}

		// Single empty-string keyword marks this as the fallback pattern
		if len(p.Keywords) == 1 && p.Keywords[0] == "" {
			if defaultPattern == nil {
				defaultPattern = p
			}
			continue
		}

		// Check for keyword substring match
		for _, kw := range p.Keywords {
			if kw != "" && strings.Contains(topic, strings.ToLower(kw)) {
				matched = p
				break
			}
		}
		if matched != nil {
			break
		}
	}

	if matched == nil {
		matched = defaultPattern
	}

	if matched == nil || len(matched.Responses) == 0 {
		return "", "", false
	}

	applyQuestEffects(matched.GrantsQuest, matched.RequiresItem, matched.GivesItem, matched.SetsQuestFlag, ps)

	response := matched.Responses[util.Rand(len(matched.Responses))]
	return response, matched.MoodChange, true
}

// TreeAdvance attempts to advance a player's position in the mob's conversation tree.
// It checks triggers against the topic, enforces node prerequisites, and updates memory.
// Returns (nodeText, hints, moodChange, advanced).
// Returns (_, _, _, false) if no tree node matches — caller should fall through to Match().
// When ps is nil, quest/item checks are skipped.
func TreeAdvance(df *DialogueFile, mobInstanceId, userId int, topic string, ps *PlayerState) (string, string, string, bool) {
	if df.Tree == nil {
		return "", "", "", false
	}

	topic = strings.ToLower(topic)
	mem := GetMemory(mobInstanceId, userId)

	if IsExpired(mem, df.Memory.ExpiryPeriod) {
		ResetMemory(mobInstanceId, userId)
		mem = GetMemory(mobInstanceId, userId)
	}

	for i := range df.Tree.Nodes {
		node := &df.Tree.Nodes[i]

		// Check triggers
		triggered := false
		for _, t := range node.Triggers {
			if strings.Contains(topic, strings.ToLower(t)) {
				triggered = true
				break
			}
		}
		if !triggered {
			continue
		}

		// Enforce requires
		allUnlocked := true
		for _, req := range node.Requires {
			if !mem.UnlockedNodes[req] {
				allUnlocked = false
				break
			}
		}
		if !allUnlocked {
			continue
		}

		// Enforce quest/item gate
		if !checkQuestGate(node.QuestRequired, node.QuestExcluded, node.RequiresItem, node.QuestFlagRequired, node.QuestFlagExcluded, ps) {
			continue
		}

		// Node matched — fire quest effects and update memory
		applyQuestEffects(node.GrantsQuest, node.RequiresItem, node.GivesItem, node.SetsQuestFlag, ps)
		UpdateMemory(mobInstanceId, userId, node.Id, node.Unlocks, topic)

		return node.Text, node.Hints, node.MoodChange, true
	}

	return "", "", "", false
}

// Greet returns the tree root greeting for the 'talk' command.
// Returns ("", "", false) if no tree is defined.
// When ps is non-nil and the root has Variants, the first matching variant
// greeting is used instead of the default root text.
func Greet(df *DialogueFile, mobInstanceId, userId int, ps *PlayerState) (string, string, bool) {
	if df.Tree == nil {
		return "", "", false
	}

	if df.Tree.Root.Text == "" && len(df.Tree.Root.Variants) == 0 {
		return "", "", false
	}

	mem := GetMemory(mobInstanceId, userId)

	if IsExpired(mem, df.Memory.ExpiryPeriod) {
		ResetMemory(mobInstanceId, userId)
		mem = GetMemory(mobInstanceId, userId)
	}

	mem.CurrentRootSeen = true
	mem.LastVisitRound = util.GetRoundCount()

	// Check quest-variant greetings first
	if ps != nil {
		for _, v := range df.Tree.Root.Variants {
			if checkQuestGate(v.QuestRequired, v.QuestExcluded, 0, v.QuestFlagRequired, v.QuestFlagExcluded, ps) {
				return v.Text, v.Hints, true
			}
		}
	}

	return df.Tree.Root.Text, df.Tree.Root.Hints, true
}
