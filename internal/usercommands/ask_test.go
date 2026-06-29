package usercommands

import "testing"

// TestIsQuestInquiry locks the set of topics that count as a pointed
// "do you have work for me?" question, so a non-quest NPC answers clearly
// instead of with generic filler (B2). Matching is exact on the trimmed,
// lowercased topic — incidental substrings like "quester" must NOT trigger.
func TestIsQuestInquiry(t *testing.T) {
	yes := []string{"quest", "quests", "task", "tasks", "job", "jobs", "work", "Quests", " task ", "JOB"}
	for _, topic := range yes {
		if !isQuestInquiry(topic) {
			t.Errorf("isQuestInquiry(%q) = false, want true", topic)
		}
	}

	no := []string{"", "charm", "rite", "quester", "homework", "passage", "footing"}
	for _, topic := range no {
		if isQuestInquiry(topic) {
			t.Errorf("isQuestInquiry(%q) = true, want false", topic)
		}
	}
}
