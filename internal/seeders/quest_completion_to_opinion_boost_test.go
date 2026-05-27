package seeders

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
)

func TestQuestCompletionToOpinion_Registered(t *testing.T) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	found := false
	for _, reg := range registry["Quest"] {
		if reg.name == ruleNameQuestCompletionToOpinion {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("quest_completion_to_opinion_boost not registered for Quest")
	}
}

// TestQuestCompletionToOpinion_ZeroFields_NoPanic confirms the rule
// does not panic when called with an empty event.
func TestQuestCompletionToOpinion_ZeroFields_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on zero-value event: %v", r)
		}
	}()
	questCompletionToOpinion(events.Quest{})
}

// TestQuestCompletionToOpinion_NonCompletionToken_NoPanic confirms
// the rule returns cleanly (no panic, no side-effects) for progress
// tokens that do not end in "-end".
func TestQuestCompletionToOpinion_NonCompletionToken_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on progress token: %v", r)
		}
	}()
	questCompletionToOpinion(events.Quest{UserId: 1, QuestToken: "5-start"})
	questCompletionToOpinion(events.Quest{UserId: 1, QuestToken: "5-2-of-3"})
}

// TestQuestCompletionToOpinion_CompletionToken_NoPanic confirms the
// rule reaches the giver-lookup stub path without panicking. The stub
// returns 0 (no quests.Quest.GiverMobTemplateId field exists yet), so
// no opinion bump fires, but the routing path is exercised.
func TestQuestCompletionToOpinion_CompletionToken_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on completion token: %v", r)
		}
	}()
	questCompletionToOpinion(events.Quest{UserId: 1, QuestToken: "5-end"})
}

// TestResolveQuestGiverMobId_Stub confirms the stub always returns 0.
// Remove / replace this test when GiverMobTemplateId is wired.
func TestResolveQuestGiverMobId_Stub(t *testing.T) {
	got := resolveQuestGiverMobId("5-end")
	if got != 0 {
		t.Errorf("expected stub to return 0, got %d", got)
	}
}

// TestResolveQuestOpinionBump_Default confirms the stub returns the
// supplied default.
func TestResolveQuestOpinionBump_Default(t *testing.T) {
	got := resolveQuestOpinionBump("5-end", questCompletionDefaultBump)
	if got != questCompletionDefaultBump {
		t.Errorf("expected %d, got %d", questCompletionDefaultBump, got)
	}
}

// TestQuestCompletionToOpinion_WrongEventType_NoPanic confirms the
// rule returns cleanly when dispatched for the wrong event type.
func TestQuestCompletionToOpinion_WrongEventType_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on wrong event type: %v", r)
		}
	}()
	questCompletionToOpinion(events.Communication{})
}

// End-to-end (opinion bump + live quests registry) requires GiverMobTemplateId
// on quests.Quest; deferred to the quests-package extension (see TODO-ADAPT).
