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
