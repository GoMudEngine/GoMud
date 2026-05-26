package mobs

import (
	"testing"
)

func TestApplyScheduleSpawnOverride_PlacesAtCurrentSegment(t *testing.T) {
	s := kerraTestSchedule()
	registerScheduleForTest(s)
	defer unregisterScheduleForTest(s.Id)

	got := applyScheduleSpawnOverride("thornwall_smith", 9999 /* homeRoomId */, 10 /* hour */)
	if got != 5678 {
		t.Errorf("hour 10: expected forge room 5678, got %d", got)
	}

	got = applyScheduleSpawnOverride("thornwall_smith", 9999, 19)
	if got != 9012 {
		t.Errorf("hour 19: expected tavern room 9012, got %d", got)
	}

	got = applyScheduleSpawnOverride("thornwall_smith", 9999, 23)
	if got != 1234 {
		t.Errorf("hour 23: expected loft room 1234, got %d", got)
	}
}

func TestApplyScheduleSpawnOverride_NoScheduleReturnsHome(t *testing.T) {
	got := applyScheduleSpawnOverride("", 9999, 10)
	if got != 9999 {
		t.Errorf("no schedule: expected home %d, got %d", 9999, got)
	}
}

func TestApplyScheduleSpawnOverride_UnknownScheduleReturnsHome(t *testing.T) {
	got := applyScheduleSpawnOverride("definitely_not_real", 9999, 10)
	if got != 9999 {
		t.Errorf("unknown schedule: expected home %d, got %d", 9999, got)
	}
}

func TestApplyScheduleSpawnOverride_PatrolSegmentFallsBackToFirstWaypoint(t *testing.T) {
	// Patrol segment with no target_room — spawn should land at the
	// patrol's first waypoint.
	RegisterScheduleForTest(&Schedule{
		Id: "guard_sched",
		Segments: []ScheduleSegment{
			{Start: 0, End: 24, Activity: "patrol", PatrolId: "guard_patrol",
				IdleCommands: []string{"watches."}},
		},
	})
	defer UnregisterScheduleForTest("guard_sched")

	registerPatrolForTest(&Patrol{
		Id:        "guard_patrol",
		Waypoints: []PatrolWaypoint{{Room: 4200, DwellRounds: 5}, {Room: 4201, DwellRounds: 5}},
	})
	defer unregisterPatrolForTest("guard_patrol")

	got := applyScheduleSpawnOverride("guard_sched", 9999 /* home */, 8 /* hour */)
	if got != 4200 {
		t.Errorf("expected first waypoint 4200, got %d", got)
	}
}

func TestApplyScheduleSpawnOverride_PatrolSegmentWithTargetRoomPrefersTarget(t *testing.T) {
	// If a patrol segment happens to set both target_room AND patrol_id,
	// the explicit target_room wins.
	RegisterScheduleForTest(&Schedule{
		Id: "guard_sched2",
		Segments: []ScheduleSegment{
			{Start: 0, End: 24, TargetRoom: 8888, Activity: "patrol", PatrolId: "guard_patrol2",
				IdleCommands: []string{"x"}},
		},
	})
	defer UnregisterScheduleForTest("guard_sched2")

	registerPatrolForTest(&Patrol{
		Id:        "guard_patrol2",
		Waypoints: []PatrolWaypoint{{Room: 4200, DwellRounds: 5}},
	})
	defer unregisterPatrolForTest("guard_patrol2")

	got := applyScheduleSpawnOverride("guard_sched2", 9999, 8)
	if got != 8888 {
		t.Errorf("expected explicit target_room 8888 to win, got %d", got)
	}
}

func TestApplyScheduleSpawnOverride_PatrolSegmentNoPatrolFallsBackToHome(t *testing.T) {
	// Defensive case: patrol segment with unresolved patrol_id → home.
	// Shouldn't happen post-load-time cross-check, but exercised here for safety.
	RegisterScheduleForTest(&Schedule{
		Id: "guard_sched3",
		Segments: []ScheduleSegment{
			{Start: 0, End: 24, Activity: "patrol", PatrolId: "does_not_exist",
				IdleCommands: []string{"x"}},
		},
	})
	defer UnregisterScheduleForTest("guard_sched3")

	got := applyScheduleSpawnOverride("guard_sched3", 9999, 8)
	if got != 9999 {
		t.Errorf("expected fallback to home 9999, got %d", got)
	}
}
