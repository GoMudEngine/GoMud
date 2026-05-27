package seeders

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
)

func TestFactionRepCounter_Registered(t *testing.T) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	found := false
	for _, reg := range registry["Communication"] {
		if reg.name == ruleNameFactionRepCounter {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("faction_rep_counter not registered for Communication")
	}
}

// TestFactionRepCounter_NoOp_OnCommunication verifies the stub
// returns without panic on any Communication event — including ones
// where SourceMobInstanceId is non-zero (mob speech) so the future
// TODO-ADAPT path has a non-trivial input to test against.
func TestFactionRepCounter_NoOp_OnCommunication(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()

	// Player speech — no SourceMobInstanceId.
	factionRepCounter(events.Communication{
		SourceUserId: 42,
		CommType:     "say",
		Message:      "Hello there",
	})

	// Mob speech — SourceMobInstanceId present (exercises the nil-guard
	// path that the future implementation will need to handle).
	factionRepCounter(events.Communication{
		SourceMobInstanceId: 99,
		CommType:            "say",
		Message:             "Greetings, traveller.",
	})
}

// TestFactionRepCounter_NonCommunicationEvent_NoOp confirms the rule
// early-returns gracefully when fed an unexpected event type (defensive
// type-assert path).
func TestFactionRepCounter_NonCommunicationEvent_NoOp(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()
	factionRepCounter(fakeEvent{typeName: "SomeOtherEvent"})
}

// Behavior tests (counter increment for verified positive-interaction
// event shape) are deferred to Task 15 smoke per spec §6.3 and the
// TODO-ADAPT stub notice in faction_rep_counter.go.
