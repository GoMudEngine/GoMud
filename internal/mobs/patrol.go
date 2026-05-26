package mobs

import "sync"

// Patrol is a looped multi-room route attached to NPCs via Mob.PatrolId
// or via a schedule segment with activity: patrol + patrol_id.
// Loaded from _datafiles/world/dogmud/patrols/<zone>/<id>.yaml at startup.
type Patrol struct {
	Id             string           `yaml:"id"`
	Description    string           `yaml:"description,omitempty"`
	LoopShape      string           `yaml:"loop_shape,omitempty"`        // "strict" (default) | "yo-yo"
	MaxPathRetries int              `yaml:"max_path_retries,omitempty"` // chunk 3.7: override global ScheduleMaxPathRetries; 0 = use global default
	Waypoints      []PatrolWaypoint `yaml:"waypoints"`
}

// PatrolWaypoint is one stop on a patrol route.
type PatrolWaypoint struct {
	Room         int    `yaml:"room"`
	DwellRounds  int    `yaml:"dwell_rounds,omitempty"`  // 0 = move on immediately
	ArrivalEvent string `yaml:"arrival_event,omitempty"` // chunk 3.7: emitted via PatrolWaypointArrival on first-dwell-tick
}

// Package-level registry, populated by LoadPatrols at startup.
var (
	patrolsMu sync.RWMutex
	patrols   = map[string]*Patrol{}
)

// GetPatrol returns the patrol with the given id, or nil if no such id is
// loaded.
func GetPatrol(id string) *Patrol {
	if id == "" {
		return nil
	}
	patrolsMu.RLock()
	defer patrolsMu.RUnlock()
	return patrols[id]
}

// NextWaypoint returns the next (waypoint index, direction) given the
// current index and direction. Direction is meaningful only for yo-yo
// patrols; strict patrols always loop forward.
//
// For strict: increments idx, wraps to 0 at the end, always returns dir=+1.
// For yo-yo: increments by dir, flips dir at endpoints (0 or last).
//
// Caller is responsible for handling the "current dwell expired, time to
// advance" decision; this function is purely about computing the next
// position in the route.
func (p *Patrol) NextWaypoint(currentIdx, currentDir int) (nextIdx, nextDir int) {
	if p == nil || len(p.Waypoints) == 0 {
		return 0, +1
	}

	loop := p.LoopShape
	if loop == "" {
		loop = "strict"
	}

	if loop != "yo-yo" {
		// strict (default + any unrecognized value)
		next := currentIdx + 1
		if next >= len(p.Waypoints) {
			next = 0
		}
		return next, +1
	}

	// yo-yo
	if currentDir == 0 {
		currentDir = +1
	}
	next := currentIdx + currentDir
	if next >= len(p.Waypoints) {
		// At the last waypoint going forward — flip to reverse.
		return len(p.Waypoints) - 2, -1
	}
	if next < 0 {
		// At the first waypoint going reverse — flip to forward.
		return 1, +1
	}
	return next, currentDir
}

// registerPatrolForTest / unregisterPatrolForTest are test-only helpers
// for injecting patrols into the package-level registry.
func registerPatrolForTest(p *Patrol) {
	patrolsMu.Lock()
	defer patrolsMu.Unlock()
	patrols[p.Id] = p
}

func unregisterPatrolForTest(id string) {
	patrolsMu.Lock()
	defer patrolsMu.Unlock()
	delete(patrols, id)
}

// RegisterPatrolForTest / UnregisterPatrolForTest are exported wrappers
// for cross-package tests (e.g., hooks tests). Matches the chunk 3.2
// RegisterScheduleForTest pattern.
func RegisterPatrolForTest(p *Patrol)   { registerPatrolForTest(p) }
func UnregisterPatrolForTest(id string) { unregisterPatrolForTest(id) }
