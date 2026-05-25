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

// applyScheduleSpawnOverride returns the room a scheduled mob should spawn
// at given the current hour. Returns homeRoomId unchanged when the mob has
// no schedule, when the schedule is unknown, or when the schedule has no
// segment covering the current hour. Called from newMobByIdInternal during
// instance creation.
func applyScheduleSpawnOverride(scheduleId string, homeRoomId int, hour24 int) int {
	if scheduleId == "" {
		return homeRoomId
	}
	s := GetSchedule(scheduleId)
	if s == nil {
		return homeRoomId
	}
	seg := s.CurrentSegment(hour24)
	if seg == nil {
		return homeRoomId
	}
	return seg.TargetRoom
}

// registerScheduleForTest / unregisterScheduleForTest are unexported helpers
// used by tests within this package. RegisterScheduleForTest / Unregister are
// the exported variants for cross-package test injection (hooks tests, etc.).
func registerScheduleForTest(s *Schedule) {
	schedulesMu.Lock()
	defer schedulesMu.Unlock()
	schedules[s.Id] = s
}

func unregisterScheduleForTest(id string) {
	schedulesMu.Lock()
	defer schedulesMu.Unlock()
	delete(schedules, id)
}

// RegisterScheduleForTest injects a Schedule into the registry so that
// tests in other packages can exercise schedule-dependent code paths
// without writing fixture YAML files. Call t.Cleanup(UnregisterScheduleForTest)
// after each call to avoid cross-test pollution.
func RegisterScheduleForTest(s *Schedule) {
	registerScheduleForTest(s)
}

// UnregisterScheduleForTest removes a previously injected schedule.
func UnregisterScheduleForTest(id string) {
	unregisterScheduleForTest(id)
}
