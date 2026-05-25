package mobs

import (
	"strings"
	"testing"
)

func TestValidatePatrol_OK(t *testing.T) {
	p := fourWaypointPatrol("strict")
	if err := validatePatrolStandalone(p); err != nil {
		t.Errorf("4-waypoint fixture should validate, got: %v", err)
	}
}

func TestValidatePatrol_EmptyWaypoints(t *testing.T) {
	p := &Patrol{Id: "broken", Waypoints: nil}
	err := validatePatrolStandalone(p)
	if err == nil || !strings.Contains(err.Error(), "waypoints") {
		t.Errorf("expected waypoints error, got: %v", err)
	}
}

func TestValidatePatrol_NegativeDwell(t *testing.T) {
	p := &Patrol{
		Id: "broken",
		Waypoints: []PatrolWaypoint{
			{Room: 100, DwellRounds: -1},
		},
	}
	err := validatePatrolStandalone(p)
	if err == nil || !strings.Contains(err.Error(), "dwell") {
		t.Errorf("expected dwell error, got: %v", err)
	}
}

func TestValidatePatrol_ZeroRoom(t *testing.T) {
	p := &Patrol{
		Id: "broken",
		Waypoints: []PatrolWaypoint{
			{Room: 0, DwellRounds: 5},
		},
	}
	err := validatePatrolStandalone(p)
	if err == nil || !strings.Contains(err.Error(), "room") {
		t.Errorf("expected room error, got: %v", err)
	}
}

func TestValidatePatrol_BadLoopShape(t *testing.T) {
	p := fourWaypointPatrol("not-a-real-shape")
	err := validatePatrolStandalone(p)
	if err == nil || !strings.Contains(err.Error(), "loop_shape") {
		t.Errorf("expected loop_shape error, got: %v", err)
	}
}

func TestValidatePatrol_SingleWaypoint_WarnsButValidates(t *testing.T) {
	p := &Patrol{
		Id: "lonely",
		Waypoints: []PatrolWaypoint{
			{Room: 100, DwellRounds: 5},
		},
	}
	if err := validatePatrolStandalone(p); err != nil {
		t.Errorf("single-waypoint patrol should validate (warn-only), got: %v", err)
	}
}

// World-aware tests require a rooms registry; integration via T13 boot smoke.
func TestValidatePatrolAgainstWorld_Stub(t *testing.T) {
	t.Skip("requires rooms + mapper fixtures — covered by boot smoke in T13")
}
