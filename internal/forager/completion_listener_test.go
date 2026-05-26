package forager

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
)

func TestForagerCompletionListener_IgnoresUnknownPatrolId(t *testing.T) {
	e := events.PatrolCompleted{
		MobInstanceId: 12345,
		PatrolId:      "thornwall_market_beat", // not a forager patrol
		RoomId:        100,
	}
	got := ForagerCompletionListener(e)
	if got != events.Continue {
		t.Errorf("expected events.Continue for unknown patrol, got %v", got)
	}
}

func TestForagerCompletionListener_IgnoresMissingMob(t *testing.T) {
	e := events.PatrolCompleted{
		MobInstanceId: 999999,
		PatrolId:      "marsh_forager_delivery",
		RoomId:        4123,
	}
	got := ForagerCompletionListener(e)
	if got != events.Continue {
		t.Errorf("expected events.Continue when mob missing, got %v", got)
	}
}

func TestForagerCompletionListener_IgnoresNonPatrolCompletedEvent(t *testing.T) {
	// Listener is registered on PatrolCompleted; if any other event
	// somehow gets routed here, it should no-op cleanly.
	got := ForagerCompletionListener(events.PatrolWaypointArrival{
		MobInstanceId: 12345,
		PatrolId:      "marsh_forager_delivery",
		ArrivalEvent:  "forager_vendor",
		RoomId:        4102,
	})
	if got != events.Continue {
		t.Errorf("expected events.Continue for non-PatrolCompleted event, got %v", got)
	}
}
