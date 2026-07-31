package dialogue

import (
	"slices"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/util"
)

// checkQuestGate returns true if the player satisfies quest/item/flag conditions.
//
// When ps is nil every check passes (backward compat for mob-to-mob and other
// non-user contexts). Production callers always pass a fully-populated state
// built by buildPlayerState.
//
// Every callback field is optional and must be nil-checked before use. Four of
// them (HasQuest, HasItem, RemoveItem, GiveQuest) were previously invoked
// unguarded, so a partially-populated state panicked on those and only those.
//
// A missing *interrogative* callback fails the gate closed: with no way to ask
// whether the player holds a token or an item, we must not conclude that they
// do. The exception is questExcluded, where the same reasoning runs the other
// way — we cannot confirm the player holds the excluding token, so the node
// stays available rather than silently vanishing.
func checkQuestGate(questRequired, questExcluded []string, requiresItem int, flagRequired, flagExcluded map[string]string, masterworkRequired int, ps *PlayerState) bool {
	if ps == nil {
		return true
	}

	if len(questRequired) > 0 && ps.HasQuest == nil {
		return false
	}
	for _, token := range questRequired {
		if !ps.HasQuest(token) {
			return false
		}
	}

	if ps.HasQuest != nil && slices.ContainsFunc(questExcluded, ps.HasQuest) {
		return false
	}

	if requiresItem > 0 {
		if ps.HasItem == nil || !ps.HasItem(requiresItem) {
			return false
		}
	}

	if masterworkRequired > 0 && ps.HasOwnMasterwork != nil && !ps.HasOwnMasterwork(masterworkRequired) {
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
func applyQuestEffects(grantsQuest string, requiresItem int, givesItem int, flagSet *QuestFlagSet, bumpsRep []RepBump, givesGold int, ps *PlayerState) {
	if ps == nil {
		return
	}
	if requiresItem > 0 && ps.RemoveItem != nil {
		ps.RemoveItem(requiresItem)
	}
	if grantsQuest != "" && ps.GiveQuest != nil {
		ps.GiveQuest(grantsQuest)
	}
	if givesItem > 0 && ps.GiveItem != nil {
		ps.GiveItem(givesItem)
	}
	if flagSet != nil && ps.SetQuestFlag != nil {
		ps.SetQuestFlag(flagSet.Key, flagSet.Value)
	}
	if ps.BumpRep != nil {
		for _, rb := range bumpsRep {
			if rb.Faction != "" && rb.Delta != 0 {
				ps.BumpRep(rb.Faction, rb.Delta)
			}
		}
	}
	if givesGold > 0 && ps.GiveGold != nil {
		ps.GiveGold(givesGold)
	}
}

// Match checks patterns against topic text for the given mob instance.
// It respects the mob's current mood when patterns declare mood filters.
// Returns (responseText, moodChange, matched).
// An empty-keyword pattern acts as the fallback when no specific keyword fires.
// When ps is nil, quest/item checks are skipped.
// Match returns (response, moodChange, ok) for a topic against the mob's
// patterns. Thin wrapper over MatchWithFallbackInfo for callers that don't
// care whether the catch-all pattern was used.
func Match(df *DialogueFile, mobInstanceId int, topic string, ps *PlayerState) (string, string, bool) {
	resp, mood, ok, _ := MatchWithFallbackInfo(df, mobInstanceId, topic, ps)
	return resp, mood, ok
}

// MatchWithFallbackInfo is Match plus a usedFallback flag: true when the
// response came from the empty-keyword catch-all pattern rather than a specific
// keyword match. Callers can use it to give a clearer answer to a pointed
// question (e.g. "ask <npc> quest") instead of generic filler.
func MatchWithFallbackInfo(df *DialogueFile, mobInstanceId int, topic string, ps *PlayerState) (string, string, bool, bool) {
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
		if !checkQuestGate(p.QuestRequired, p.QuestExcluded, p.RequiresItem, p.QuestFlagRequired, p.QuestFlagExcluded, p.MasterworkRequired, ps) {
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
		return "", "", false, false
	}

	usedFallback := matched == defaultPattern

	applyQuestEffects(matched.GrantsQuest, matched.RequiresItem, matched.GivesItem, matched.SetsQuestFlag, matched.BumpsRep, matched.GivesGold, ps)

	response := matched.Responses[util.Rand(len(matched.Responses))]
	return response, matched.MoodChange, true, usedFallback
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
		if !checkQuestGate(node.QuestRequired, node.QuestExcluded, node.RequiresItem, node.QuestFlagRequired, node.QuestFlagExcluded, node.MasterworkRequired, ps) {
			continue
		}

		// Node matched — fire quest effects and update memory
		applyQuestEffects(node.GrantsQuest, node.RequiresItem, node.GivesItem, node.SetsQuestFlag, node.BumpsRep, node.GivesGold, ps)
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
			if checkQuestGate(v.QuestRequired, v.QuestExcluded, 0, v.QuestFlagRequired, v.QuestFlagExcluded, v.MasterworkRequired, ps) {
				applyQuestEffects(v.GrantsQuest, v.RequiresItem, v.GivesItem, v.SetsQuestFlag, v.BumpsRep, v.GivesGold, ps)
				return v.Text, v.Hints, true
			}
		}
	}

	return df.Tree.Root.Text, df.Tree.Root.Hints, true
}
