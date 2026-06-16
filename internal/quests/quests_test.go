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

func TestQuest_RepeatableFieldsLoad(t *testing.T) {
	src := []byte("questid: 9999\nname: Repeat Me\nrepeatable: true\ncooldown_rounds: 200\n")
	var q Quest
	if err := yaml.Unmarshal(src, &q); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !q.Repeatable {
		t.Errorf("Repeatable: want true, got false")
	}
	if q.CooldownRounds != 200 {
		t.Errorf("CooldownRounds: want 200, got %d", q.CooldownRounds)
	}
}

func TestQuest_RepeatableDefaultsFalse(t *testing.T) {
	src := []byte("questid: 9998\nname: One Shot\n")
	var q Quest
	if err := yaml.Unmarshal(src, &q); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if q.Repeatable {
		t.Errorf("Repeatable: want false (default), got true")
	}
	if q.CooldownRounds != 0 {
		t.Errorf("CooldownRounds: want 0 (default), got %d", q.CooldownRounds)
	}
}
