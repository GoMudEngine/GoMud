package dialogue

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/util"
)

// Match checks patterns against topic text for the given mob instance.
// It respects the mob's current mood when patterns declare mood filters.
// Returns (responseText, moodChange, matched).
// An empty-keyword pattern acts as the fallback when no specific keyword fires.
func Match(df *DialogueFile, mobInstanceId int, topic string) (string, string, bool) {
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

	response := matched.Responses[util.Rand(len(matched.Responses))]
	return response, matched.MoodChange, true
}

// TreeAdvance attempts to advance a player's position in the mob's conversation tree.
// It checks triggers against the topic, enforces node prerequisites, and updates memory.
// Returns (nodeText, hints, moodChange, advanced).
// Returns (_, _, _, false) if no tree node matches — caller should fall through to Match().
func TreeAdvance(df *DialogueFile, mobInstanceId, userId int, topic string) (string, string, string, bool) {
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

		// Node matched — update memory
		UpdateMemory(mobInstanceId, userId, node.Id, node.Unlocks, topic)

		return node.Text, node.Hints, node.MoodChange, true
	}

	return "", "", "", false
}

// Greet returns the tree root greeting for the 'talk' command.
// Returns ("", "", false) if no tree is defined.
// The root greeting is delivered each visit regardless of prior state.
func Greet(df *DialogueFile, mobInstanceId, userId int) (string, string, bool) {
	if df.Tree == nil {
		return "", "", false
	}

	if df.Tree.Root.Text == "" {
		return "", "", false
	}

	mem := GetMemory(mobInstanceId, userId)

	if IsExpired(mem, df.Memory.ExpiryPeriod) {
		ResetMemory(mobInstanceId, userId)
		mem = GetMemory(mobInstanceId, userId)
	}

	mem.CurrentRootSeen = true
	mem.LastVisitRound = util.GetRoundCount()

	return df.Tree.Root.Text, df.Tree.Root.Hints, true
}
