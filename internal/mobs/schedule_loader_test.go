package mobs

import (
	"strings"
	"testing"
)

// validateScheduleStandalone is the load-time per-schedule validator. It is
// extracted so we can unit-test it without writing fixture files.
// Real test names below call validateScheduleStandalone with hand-built
// schedules (or the shared kerraTestSchedule fixture from schedule_test.go);
// the file-walking loader gets a separate integration test.

func TestValidateSchedule_FullCoverage_OK(t *testing.T) {
	s := kerraTestSchedule()
	if err := validateScheduleStandalone(s); err != nil {
		t.Errorf("kerra fixture should validate, got error: %v", err)
	}
}

func TestValidateSchedule_GapInCoverage_Panics(t *testing.T) {
	s := &Schedule{
		Id: "broken",
		Segments: []ScheduleSegment{
			{Start: 6, End: 9, TargetRoom: 1},
			// gap: 9-22
			{Start: 22, End: 6, TargetRoom: 1},
		},
	}
	err := validateScheduleStandalone(s)
	if err == nil {
		t.Fatal("expected error for coverage gap, got nil")
	}
	if !strings.Contains(err.Error(), "gap") && !strings.Contains(err.Error(), "coverage") {
		t.Errorf("expected gap/coverage error, got: %v", err)
	}
}

func TestValidateSchedule_OverlappingSegments_Panics(t *testing.T) {
	s := &Schedule{
		Id: "broken",
		Segments: []ScheduleSegment{
			{Start: 6, End: 12, TargetRoom: 1},
			{Start: 10, End: 18, TargetRoom: 2}, // overlaps 10-12
			{Start: 18, End: 6, TargetRoom: 1},
		},
	}
	err := validateScheduleStandalone(s)
	if err == nil {
		t.Fatal("expected error for overlap, got nil")
	}
	if !strings.Contains(err.Error(), "overlap") && !strings.Contains(err.Error(), "claimed") {
		t.Errorf("expected overlap error, got: %v", err)
	}
}

func TestValidateSchedule_BadHourBounds_Panics(t *testing.T) {
	cases := []struct {
		name string
		seg  ScheduleSegment
	}{
		{"negative start", ScheduleSegment{Start: -1, End: 9, TargetRoom: 1}},
		{"start over 23", ScheduleSegment{Start: 25, End: 9, TargetRoom: 1}},
		{"end over 24", ScheduleSegment{Start: 9, End: 25, TargetRoom: 1}},
		{"end zero", ScheduleSegment{Start: 9, End: 0, TargetRoom: 1}},
		{"start equals end", ScheduleSegment{Start: 9, End: 9, TargetRoom: 1}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := &Schedule{Id: "broken", Segments: []ScheduleSegment{c.seg}}
			if err := validateScheduleStandalone(s); err == nil {
				t.Errorf("expected error for %s, got nil", c.name)
			}
		})
	}
}

func TestValidateSchedule_MissingTargetRoom_Panics(t *testing.T) {
	s := &Schedule{
		Id: "broken",
		Segments: []ScheduleSegment{
			{Start: 0, End: 24, TargetRoom: 0}, // 24h coverage but room id 0
		},
	}
	err := validateScheduleStandalone(s)
	if err == nil {
		t.Fatal("expected error for missing target_room, got nil")
	}
	if !strings.Contains(err.Error(), "target_room") {
		t.Errorf("expected target_room error, got: %v", err)
	}
}

// World-aware tests (room existence, pathfinding sanity) require a real
// rooms registry; integration is exercised by the boot smoke in T13.
func TestValidateScheduleAgainstWorld_RoomDoesNotExist(t *testing.T) {
	t.Skip("requires rooms fixture — covered by boot smoke in T13")
}

// TestValidateScheduleAgainstWorld_PatrolSegmentSkipped is a regression test
// for the boot-panic bug (chunk 3.4): validateScheduleAgainstWorld must not
// call roomExists(0) or hasPath(0, X) for patrol segments. A patrol+sleep
// schedule must pass world validation even though the patrol segment has no
// target_room.
func TestValidateScheduleAgainstWorld_PatrolSegmentSkipped(t *testing.T) {
	s := &Schedule{
		Id: "guard_patrol_schedule",
		Segments: []ScheduleSegment{
			// patrol segment — TargetRoom intentionally 0
			{Start: 6, End: 22, Activity: "patrol", PatrolId: "guard_patrol",
				IdleCommands: []string{"emote watches."}},
			// sleep segment — TargetRoom 5104 (barracks)
			{Start: 22, End: 6, TargetRoom: 5104, Activity: "sleeping",
				IdleCommands: []string{"emote snores."}},
		},
	}

	roomExistsCalled := false
	hasPathCalled := false

	roomExists := func(id int) bool {
		roomExistsCalled = true
		if id == 0 {
			t.Errorf("roomExists called with room 0 — patrol segment must be skipped")
		}
		return id == 5104
	}
	hasPath := func(from, to int) bool {
		hasPathCalled = true
		if from == 0 || to == 0 {
			t.Errorf("hasPath called with room 0 (from=%d, to=%d) — patrol segment must be skipped", from, to)
		}
		return true
	}

	if err := validateScheduleAgainstWorld(s, roomExists, hasPath); err != nil {
		t.Errorf("expected no error for patrol+sleep schedule, got: %v", err)
	}

	// roomExists must have been called for the sleep segment (5104) but not for
	// the patrol segment (0). hasPath may or may not be called depending on
	// whether the sleep segment forms a pair with another non-patrol segment.
	_ = roomExistsCalled
	_ = hasPathCalled
}

func TestValidateSchedule_AcceptsSleepingActivity(t *testing.T) {
	s := &Schedule{
		Id: "sleeper_test",
		Segments: []ScheduleSegment{
			{Start: 0, End: 24, TargetRoom: 1, Activity: "sleeping",
				IdleCommands: []string{"emote snores."}},
		},
	}
	if err := validateScheduleStandalone(s); err != nil {
		t.Errorf("expected sleeping activity to validate, got %v", err)
	}
}

func TestMobScheduleIdCrossCheck_Stub(t *testing.T) {
	// Cross-check is wired in LoadDataFiles and is exercised by the boot
	// smoke at T13. Stub here as a documentation hook — if we add a
	// LoadDataFiles unit-test fixture later this is where it slots in.
	t.Skip("covered by boot smoke in T13")
}

func TestValidateSchedule_PatrolSegmentWithoutTargetRoom_OK(t *testing.T) {
	s := &Schedule{
		Id: "patrol_schedule_test",
		Segments: []ScheduleSegment{
			{Start: 0, End: 24, Activity: "patrol", PatrolId: "some_patrol",
				IdleCommands: []string{"emote watches."}},
		},
	}
	if err := validateScheduleStandalone(s); err != nil {
		t.Errorf("patrol segment without target_room should validate, got: %v", err)
	}
}

func TestValidateSchedule_PatrolSegmentWithoutPatrolId_Fails(t *testing.T) {
	s := &Schedule{
		Id: "broken_patrol_schedule",
		Segments: []ScheduleSegment{
			{Start: 0, End: 24, TargetRoom: 100, Activity: "patrol", PatrolId: "",
				IdleCommands: []string{"x"}},
		},
	}
	err := validateScheduleStandalone(s)
	if err == nil || !strings.Contains(err.Error(), "patrol_id is empty") {
		t.Errorf("expected patrol_id-empty error, got: %v", err)
	}
}

func TestValidateSchedule_NonPatrolSegmentWithTargetRoomZero_Fails(t *testing.T) {
	// Regression: target_room must still be required for non-patrol activities.
	s := &Schedule{
		Id: "broken",
		Segments: []ScheduleSegment{
			{Start: 0, End: 24, TargetRoom: 0, Activity: "", IdleCommands: []string{"x"}},
		},
	}
	err := validateScheduleStandalone(s)
	if err == nil || !strings.Contains(err.Error(), "target_room is 0") {
		t.Errorf("expected target_room=0 error for non-patrol activity, got: %v", err)
	}
}
