package caravan

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
)

func TestCaravanRunnerCompletionListener_IgnoresUnknownPatrol(t *testing.T) {
	e := events.PatrolCompleted{
		MobInstanceId: 12345,
		PatrolId:      "thornwall_market_beat", // not a runner-circuit patrol
		RoomId:        100,
	}
	got := CaravanRunnerCompletionListener(e)
	if got != events.Continue {
		t.Errorf("expected events.Continue for unknown patrol, got %v", got)
	}
}

func TestCaravanRunnerCompletionListener_IgnoresMissingRunner(t *testing.T) {
	e := events.PatrolCompleted{
		MobInstanceId: 999999, // not a live instance
		PatrolId:      "thornwall_runner_circuit",
		RoomId:        465,
	}
	got := CaravanRunnerCompletionListener(e)
	if got != events.Continue {
		t.Errorf("expected events.Continue when runner missing, got %v", got)
	}
}

func TestCaravanRunnerCompletionListener_IgnoresNonPatrolCompletedEvent(t *testing.T) {
	got := CaravanRunnerCompletionListener(events.PatrolWaypointArrival{
		MobInstanceId: 12345,
		PatrolId:      "thornwall_runner_circuit",
		ArrivalEvent:  "caravan_vendor",
		RoomId:        464,
	})
	if got != events.Continue {
		t.Errorf("expected events.Continue for non-PatrolCompleted event, got %v", got)
	}
}
