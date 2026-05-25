package mobs

import (
	"testing"
)

// Fixture: Kerra-shaped schedule covering all 24 hours.
//
// 6-9   loft (waking)
// 9-18  forge (craft)
// 18-22 tavern
// 22-6  loft (sleep, wraps midnight)
func kerraTestSchedule() *Schedule {
	return &Schedule{
		Id:          "thornwall_smith",
		Description: "Kerra test fixture",
		Segments: []ScheduleSegment{
			{Start: 6, End: 9, TargetRoom: 1234, Activity: "", IdleCommands: []string{"emote wakes."}},
			{Start: 9, End: 18, TargetRoom: 5678, Activity: "craft", IdleCommands: []string{"emote hammers."}},
			{Start: 18, End: 22, TargetRoom: 9012, Activity: "", IdleCommands: []string{"emote sips."}},
			{Start: 22, End: 6, TargetRoom: 1234, Activity: "", IdleCommands: []string{"emote snores."}},
		},
	}
}

func TestSchedule_CurrentSegment_BasicHours(t *testing.T) {
	s := kerraTestSchedule()
	cases := []struct {
		hour int
		want int // expected segment Start
	}{
		{7, 6},   // waking segment
		{10, 9},  // forge segment
		{17, 9},  // still forge (exclusive end)
		{18, 18}, // tavern starts at 18
		{21, 18}, // still tavern
		{22, 22}, // sleep starts at 22
	}
	for _, c := range cases {
		seg := s.CurrentSegment(c.hour)
		if seg == nil {
			t.Errorf("hour %d: expected segment with Start=%d, got nil", c.hour, c.want)
			continue
		}
		if seg.Start != c.want {
			t.Errorf("hour %d: expected segment Start=%d, got Start=%d", c.hour, c.want, seg.Start)
		}
	}
}

func TestSchedule_CurrentSegment_WrapsMidnight(t *testing.T) {
	s := kerraTestSchedule()
	// 22-6 wraps midnight; 23, 0, 5 should all return the wrap segment.
	for _, hour := range []int{22, 23, 0, 1, 5} {
		seg := s.CurrentSegment(hour)
		if seg == nil || seg.Start != 22 {
			t.Errorf("hour %d: expected wrap segment Start=22, got %+v", hour, seg)
		}
	}
}

func TestSchedule_CurrentSegment_ExclusiveEnd(t *testing.T) {
	s := kerraTestSchedule()
	// Hour 6 is the exclusive end of the sleep wrap segment AND the inclusive
	// start of the waking segment. The waking segment should win.
	seg := s.CurrentSegment(6)
	if seg == nil || seg.Start != 6 {
		t.Errorf("hour 6: expected waking segment Start=6 to win, got %+v", seg)
	}
}

func TestSchedule_CurrentSegment_SameRoomTwiceNonAdjacent(t *testing.T) {
	// Olen pattern: chamber appears in two non-adjacent segments with different
	// idlecommands. The resolver must return the right segment for the right
	// hour, not just the first segment that matches the room.
	s := &Schedule{
		Id: "thornwall_temple_priest",
		Segments: []ScheduleSegment{
			{Start: 4, End: 6, TargetRoom: 100, IdleCommands: []string{"rise"}},
			{Start: 6, End: 10, TargetRoom: 200, IdleCommands: []string{"prayers-morning"}},
			{Start: 10, End: 12, TargetRoom: 100, IdleCommands: []string{"rest"}},
			{Start: 12, End: 18, TargetRoom: 200, IdleCommands: []string{"prayers-afternoon"}},
			{Start: 18, End: 22, TargetRoom: 300, IdleCommands: []string{"tavern"}},
			{Start: 22, End: 4, TargetRoom: 100, IdleCommands: []string{"sleep"}},
		},
	}
	if got := s.CurrentSegment(5).IdleCommands[0]; got != "rise" {
		t.Errorf("hour 5: want rise, got %s", got)
	}
	if got := s.CurrentSegment(11).IdleCommands[0]; got != "rest" {
		t.Errorf("hour 11: want rest, got %s", got)
	}
	if got := s.CurrentSegment(23).IdleCommands[0]; got != "sleep" {
		t.Errorf("hour 23: want sleep, got %s", got)
	}
}

func TestSchedule_CurrentSegment_NoCoverageReturnsNil(t *testing.T) {
	// Defensive: at runtime a validated schedule always covers 24h, but the
	// resolver should not panic if given a gap-having fixture.
	s := &Schedule{
		Id:       "broken",
		Segments: []ScheduleSegment{{Start: 9, End: 17, TargetRoom: 1}},
	}
	if seg := s.CurrentSegment(3); seg != nil {
		t.Errorf("hour 3: expected nil for uncovered hour, got %+v", seg)
	}
}
