package caravan

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
)

func TestCaravanArrivalListener_IgnoresNonCaravanEvents(t *testing.T) {
	e := events.PatrolWaypointArrival{
		MobInstanceId: 12345,
		PatrolId:      "thornwall_market_beat",
		ArrivalEvent:  "",
	}
	got := CaravanArrivalListener(e)
	if got != events.Continue {
		t.Errorf("expected events.Continue for non-caravan event, got %v", got)
	}
}

func TestCaravanArrivalListener_NoOpOnUnknownArrivalEvent(t *testing.T) {
	e := events.PatrolWaypointArrival{
		MobInstanceId: 12345,
		PatrolId:      CaravanPatrolId,
		ArrivalEvent:  "made_up_event_name",
	}
	got := CaravanArrivalListener(e)
	if got != events.Continue {
		t.Errorf("expected events.Continue for unknown arrival_event, got %v", got)
	}
}

func TestCaravanArrivalListener_NoOpOnMissingLeaderInstance(t *testing.T) {
	// Caravan patrol id + valid arrival_event, but the instance id doesn't exist.
	// Listener should no-op cleanly without panicking.
	e := events.PatrolWaypointArrival{
		MobInstanceId: 999999, // not a live instance
		PatrolId:      CaravanPatrolId,
		ArrivalEvent:  "caravan_vendor",
	}
	got := CaravanArrivalListener(e)
	if got != events.Continue {
		t.Errorf("expected events.Continue when leader instance missing, got %v", got)
	}
}
