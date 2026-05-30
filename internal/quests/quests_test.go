package quests

import (
	"testing"

	"gopkg.in/yaml.v2"
)

func TestQuestReward_RepFieldsUnmarshal(t *testing.T) {
	raw := []byte("rep_faction: thornwall_guards\nrep_amount: 15\n")
	var r QuestReward
	if err := yaml.Unmarshal(raw, &r); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if r.RepFaction != "thornwall_guards" {
		t.Fatalf("RepFaction = %q, want thornwall_guards", r.RepFaction)
	}
	if r.RepAmount != 15 {
		t.Fatalf("RepAmount = %d, want 15", r.RepAmount)
	}
}
