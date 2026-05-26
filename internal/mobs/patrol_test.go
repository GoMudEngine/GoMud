package mobs

import (
	"testing"
)

// Fixture: 4-waypoint patrol covering all combinations.
//
// rooms: 100 → 101 → 102 → 103
// dwell: 5 / 10 / 5 / 0
func fourWaypointPatrol(loopShape string) *Patrol {
	return &Patrol{
		Id:          "test_patrol",
		Description: "4-waypoint fixture",
		LoopShape:   loopShape,
		Waypoints: []PatrolWaypoint{
			{Room: 100, DwellRounds: 5},
			{Room: 101, DwellRounds: 10},
			{Room: 102, DwellRounds: 5},
			{Room: 103, DwellRounds: 0},
		},
	}
}

func TestPatrol_NextWaypoint_StrictLoop(t *testing.T) {
	p := fourWaypointPatrol("strict")
	cases := []struct {
		currentIdx int
		direction  int
		wantIdx    int
		wantDir    int
	}{
		{0, +1, 1, +1},
		{1, +1, 2, +1},
		{2, +1, 3, +1},
		{3, +1, 0, +1}, // wrap to start
	}
	for _, c := range cases {
		gotIdx, gotDir := p.NextWaypoint(c.currentIdx, c.direction)
		if gotIdx != c.wantIdx || gotDir != c.wantDir {
			t.Errorf("strict from idx=%d dir=%+d: want (%d, %+d), got (%d, %+d)",
				c.currentIdx, c.direction, c.wantIdx, c.wantDir, gotIdx, gotDir)
		}
	}
}

func TestPatrol_NextWaypoint_YoYoForwardAndReverse(t *testing.T) {
	p := fourWaypointPatrol("yo-yo")
	// Forward up to the end, then flip and reverse back.
	cases := []struct {
		currentIdx int
		direction  int
		wantIdx    int
		wantDir    int
	}{
		{0, +1, 1, +1},
		{1, +1, 2, +1},
		{2, +1, 3, +1},
		{3, +1, 2, -1}, // at last, flip to reverse
		{2, -1, 1, -1},
		{1, -1, 0, -1},
		{0, -1, 1, +1}, // at first going reverse, flip to forward
	}
	for _, c := range cases {
		gotIdx, gotDir := p.NextWaypoint(c.currentIdx, c.direction)
		if gotIdx != c.wantIdx || gotDir != c.wantDir {
			t.Errorf("yo-yo from idx=%d dir=%+d: want (%d, %+d), got (%d, %+d)",
				c.currentIdx, c.direction, c.wantIdx, c.wantDir, gotIdx, gotDir)
		}
	}
}

func TestPatrol_NextWaypoint_StrictEmptyDirectionIgnored(t *testing.T) {
	// Strict loops ignore direction — always +1, always wraps end → start.
	p := fourWaypointPatrol("strict")
	gotIdx, gotDir := p.NextWaypoint(3, -1)
	if gotIdx != 0 || gotDir != +1 {
		t.Errorf("strict with dir=-1: want (0, +1), got (%d, %+d)", gotIdx, gotDir)
	}
}

func TestPatrol_NextWaypoint_DefaultsToStrictWhenLoopShapeEmpty(t *testing.T) {
	p := fourWaypointPatrol("") // empty loop_shape → defaults to strict
	gotIdx, gotDir := p.NextWaypoint(3, +1)
	if gotIdx != 0 || gotDir != +1 {
		t.Errorf("empty loop_shape with idx=3 dir=+1: want (0, +1), got (%d, %+d)",
			gotIdx, gotDir)
	}
}

func TestGetPatrol_EmptyReturnsNil(t *testing.T) {
	if got := GetPatrol(""); got != nil {
		t.Errorf("GetPatrol(\"\"): want nil, got %+v", got)
	}
}

func TestGetPatrol_UnknownReturnsNil(t *testing.T) {
	if got := GetPatrol("definitely_not_real"); got != nil {
		t.Errorf("unknown patrol id: want nil, got %+v", got)
	}
}

func TestStartOneshotPatrol_AssignsAndResetsMiscData(t *testing.T) {
	RegisterPatrolForTest(&Patrol{
		Id:        "test_oneshot_assign",
		LoopShape: "oneshot",
		Waypoints: []PatrolWaypoint{{Room: 100, DwellRounds: 2}},
	})
	t.Cleanup(func() { UnregisterPatrolForTest("test_oneshot_assign") })

	mob := &Mob{InstanceId: 90100}
	// Pre-seed dirty MiscData to verify reset.
	mob.Character.SetMiscData("patrol_waypoint_idx", 7)
	mob.Character.SetMiscData("patrol_direction", -1)
	mob.Character.SetMiscData("patrol_dwell_remaining", 99)
	mob.Character.SetMiscData("patrol_path_fail_count", 5)

	ok := StartOneshotPatrol(mob, "test_oneshot_assign")
	if !ok {
		t.Fatal("StartOneshotPatrol returned false for valid oneshot patrol")
	}
	if mob.PatrolId != "test_oneshot_assign" {
		t.Errorf("PatrolId = %q, want %q", mob.PatrolId, "test_oneshot_assign")
	}
	if v, _ := mob.Character.GetMiscData("patrol_waypoint_idx").(int); v != 0 {
		t.Errorf("patrol_waypoint_idx = %d, want 0", v)
	}
	if v, _ := mob.Character.GetMiscData("patrol_direction").(int); v != 1 {
		t.Errorf("patrol_direction = %d, want 1", v)
	}
	if v, _ := mob.Character.GetMiscData("patrol_dwell_remaining").(int); v != 0 {
		t.Errorf("patrol_dwell_remaining = %d, want 0", v)
	}
	if v, _ := mob.Character.GetMiscData("patrol_path_fail_count").(int); v != 0 {
		t.Errorf("patrol_path_fail_count = %d, want 0", v)
	}
}

func TestStartOneshotPatrol_RejectsNonOneshot(t *testing.T) {
	RegisterPatrolForTest(&Patrol{
		Id:        "test_strict_patrol",
		LoopShape: "strict",
		Waypoints: []PatrolWaypoint{{Room: 100, DwellRounds: 2}},
	})
	t.Cleanup(func() { UnregisterPatrolForTest("test_strict_patrol") })

	mob := &Mob{InstanceId: 90101}
	ok := StartOneshotPatrol(mob, "test_strict_patrol")
	if ok {
		t.Errorf("StartOneshotPatrol should return false for strict patrol, got true")
	}
	if mob.PatrolId != "" {
		t.Errorf("PatrolId should remain empty after rejection, got %q", mob.PatrolId)
	}
}

func TestStartOneshotPatrol_RejectsUnknownPatrol(t *testing.T) {
	mob := &Mob{InstanceId: 90102}
	ok := StartOneshotPatrol(mob, "does_not_exist")
	if ok {
		t.Errorf("StartOneshotPatrol should return false for unknown patrol, got true")
	}
}

func TestStartOneshotPatrol_NilMobReturnsFalse(t *testing.T) {
	ok := StartOneshotPatrol(nil, "anything")
	if ok {
		t.Errorf("StartOneshotPatrol should return false for nil mob, got true")
	}
}

func TestClearOneshotPatrol_ClearsAllPatrolMiscData(t *testing.T) {
	mob := &Mob{InstanceId: 90103, PatrolId: "test_patrol"}
	mob.Character.SetMiscData("patrol_waypoint_idx", 5)
	mob.Character.SetMiscData("patrol_direction", -1)
	mob.Character.SetMiscData("patrol_dwell_remaining", 3)
	mob.Character.SetMiscData("patrol_path_fail_count", 2)

	ClearOneshotPatrol(mob)

	if mob.PatrolId != "" {
		t.Errorf("PatrolId = %q, want empty", mob.PatrolId)
	}
	if v, _ := mob.Character.GetMiscData("patrol_waypoint_idx").(int); v != 0 {
		t.Errorf("patrol_waypoint_idx = %d, want 0", v)
	}
	if v, _ := mob.Character.GetMiscData("patrol_direction").(int); v != 0 {
		t.Errorf("patrol_direction = %d, want 0", v)
	}
	if v, _ := mob.Character.GetMiscData("patrol_dwell_remaining").(int); v != 0 {
		t.Errorf("patrol_dwell_remaining = %d, want 0", v)
	}
	if v, _ := mob.Character.GetMiscData("patrol_path_fail_count").(int); v != 0 {
		t.Errorf("patrol_path_fail_count = %d, want 0", v)
	}
}

func TestClearOneshotPatrol_NilMobNoOps(t *testing.T) {
	// Just verify no panic.
	ClearOneshotPatrol(nil)
}

func TestPatrol_OneshotLoopShape_RoundTrips(t *testing.T) {
	p := &Patrol{
		Id:        "test_oneshot",
		LoopShape: "oneshot",
		Waypoints: []PatrolWaypoint{
			{Room: 100, DwellRounds: 3},
			{Room: 101, DwellRounds: 1},
		},
	}
	RegisterPatrolForTest(p)
	t.Cleanup(func() { UnregisterPatrolForTest("test_oneshot") })

	got := GetPatrol("test_oneshot")
	if got == nil {
		t.Fatal("patrol not registered")
	}
	if got.LoopShape != "oneshot" {
		t.Errorf("LoopShape = %q, want %q", got.LoopShape, "oneshot")
	}
}
