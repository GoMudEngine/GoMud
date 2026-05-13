package worldevents

import "testing"

func TestEmitAssignsMonotonicIds(t *testing.T) {
	// Reset state for test isolation
	ready = false
	eventBuffer = make([]WorldEvent, 0, 200)
	maxEvents = 200
	ready = true

	// Reset the counter to ensure deterministic test
	nextEventId.Store(0)

	EmitWorldEvent(WorldEvent{Description: "first"})
	EmitWorldEvent(WorldEvent{Description: "second"})

	evts := GetRecentWorldEvents(2, nil)
	if len(evts) != 2 {
		t.Fatalf("expected 2 events, got %d", len(evts))
	}
	if evts[0].Id == 0 {
		t.Errorf("first event got id 0; should start >= 1")
	}
	if evts[1].Id != evts[0].Id+1 && evts[0].Id != evts[1].Id+1 {
		t.Errorf("ids not monotonic: %d, %d", evts[0].Id, evts[1].Id)
	}
}
