package mobs

import "sync"

// Schedule is a daily routine attached to NPCs via Mob.ScheduleId.
// Loaded from _datafiles/world/dogmud/schedules/<zone>/<id>.yaml at startup.
type Schedule struct {
	Id          string            `yaml:"id"`
	Description string            `yaml:"description,omitempty"`
	Segments    []ScheduleSegment `yaml:"segments"`
}

// ScheduleSegment covers a contiguous hour range [Start, End). When Start > End
// the segment wraps midnight (e.g. Start=22 End=6 covers 22-23 and 0-5).
type ScheduleSegment struct {
	Start        int      `yaml:"start"`              // 0-23 inclusive
	End          int      `yaml:"end"`                // 1-24 inclusive
	TargetRoom   int      `yaml:"target_room"`        // room the mob should occupy
	Activity     string   `yaml:"activity,omitempty"` // "" | "craft" | future maintenance verbs
	IdleCommands []string `yaml:"idlecommands,omitempty"`
}

// Package-level registry, populated by LoadSchedules at startup.
var (
	schedulesMu sync.RWMutex
	schedules   = map[string]*Schedule{}
)

// GetSchedule returns the schedule with the given id, or nil if no such id is
// loaded.
func GetSchedule(id string) *Schedule {
	if id == "" {
		return nil
	}
	schedulesMu.RLock()
	defer schedulesMu.RUnlock()
	return schedules[id]
}

// CurrentSegment returns the segment active at hour24 (0-23). Returns nil if
// no segment covers this hour. At runtime, loaded schedules are validated to
// cover all 24 hours exactly once, so nil should never be observed.
func (s *Schedule) CurrentSegment(hour24 int) *ScheduleSegment {
	if s == nil {
		return nil
	}
	for i := range s.Segments {
		seg := &s.Segments[i]
		if segContainsHour(seg, hour24) {
			return seg
		}
	}
	return nil
}

// segContainsHour reports whether [seg.Start, seg.End) contains hour24,
// handling the wrap-around case where Start > End.
//
// Inclusive at start, exclusive at end. End == 24 means "up to but not
// including midnight" (the day boundary).
func segContainsHour(seg *ScheduleSegment, hour24 int) bool {
	if seg.Start == seg.End {
		return false // empty segment — validation rejects these at load time
	}
	if seg.Start < seg.End {
		return hour24 >= seg.Start && hour24 < seg.End
	}
	// Wraps midnight: covers [Start, 24) and [0, End).
	return hour24 >= seg.Start || hour24 < seg.End
}
