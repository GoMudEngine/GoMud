package behaviortree

import (
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// roundForHour scans all rounds in one game day and returns the first
// roundOfDay whose Hour24 equals the target hour. If no round maps to
// that hour (can happen when RoundsPerDay < 24), returns -1.
func roundForHour(hour int) int {
	c := configs.GetTimingConfig()
	rpd := int(c.RoundsPerDay)
	for r := 0; r < rpd; r++ {
		hourFloat, _ := math.Modf(float64(r) / float64(rpd) * 24)
		if int(hourFloat) == hour {
			return r + rpd // offset by rpd so we never send round 0
		}
	}
	return -1
}

// setTestHour pins the global round count so subsequent gametime queries
// resolve to a known Hour24. Returns the actual Hour24 resolved plus a
// cleanup func.
func setTestHour(t *testing.T, hour int) (actualHour int, cleanup func()) {
	t.Helper()
	prev := util.GetRoundCount()
	r := roundForHour(hour)
	if r < 0 {
		return -1, func() { util.SetRoundCountForTest(prev) }
	}
	util.SetRoundCountForTest(uint64(r))
	gd := gametime.GetDate()
	return gd.Hour24, func() { util.SetRoundCountForTest(prev) }
}

// mustHour calls setTestHour and fatally fails the test if the resolved
// Hour24 doesn't match the expected value. Catches misconfigured test env
// before any assertion in the real test runs.
func mustHour(t *testing.T, hour int) func() {
	t.Helper()
	actual, cleanup := setTestHour(t, hour)
	if actual != hour {
		cleanup()
		t.Fatalf("mustHour(%d): gametime resolved Hour24=%d (RoundsPerDay=%d); "+
			"this exact hour is not representable in the test environment — adjust the test",
			hour, actual, configs.GetTimingConfig().RoundsPerDay)
	}
	return cleanup
}

// ---------------------------------------------------------------------------
// Binary period tests (pre-existing behaviour, backward compat)
// ---------------------------------------------------------------------------

func TestCondTimeOfDay_BinaryPeriod_Day(t *testing.T) {
	// Noon (12) is always representable and always day.
	defer mustHour(t, 12)()

	res := condTimeOfDay(map[string]any{"period": "day"}, nil)
	if res != Success {
		t.Errorf("expected Success at noon with period=day, got %v", res)
	}
}

func TestCondTimeOfDay_BinaryPeriod_Night_WhenNightEnabled(t *testing.T) {
	// With NightHours=0 (test-env default) IsNight() is always false.
	// Skip rather than assert a value that depends on server-config.
	c := configs.GetTimingConfig()
	if c.NightHours == 0 {
		t.Skip("NightHours=0 in test environment — IsNight() always false, nothing to assert")
	}
	halfNight := int(c.NightHours) / 2
	nightHour := 24 - halfNight
	defer mustHour(t, nightHour)()

	res := condTimeOfDay(map[string]any{"period": "night"}, nil)
	if res != Success {
		t.Errorf("expected Success at hour %d with period=night, got %v", nightHour, res)
	}
}

// ---------------------------------------------------------------------------
// Range tests — all hours chosen from the 20-round-per-day representable set:
//   0,1,2,3,4,6,7,8,9,10,12,13,14,15,16,18,19,20,21,22
// ---------------------------------------------------------------------------

func TestCondTimeOfDay_Range_BasicMatch(t *testing.T) {
	// 9 is representable; 9-16 window covers it.
	defer mustHour(t, 9)()

	res := condTimeOfDay(map[string]any{"range": "9-16"}, nil)
	if res != Success {
		t.Errorf("expected Success at 9am with range=9-16, got %v", res)
	}
}

func TestCondTimeOfDay_Range_InsideWindowMidpoint(t *testing.T) {
	defer mustHour(t, 13)()

	res := condTimeOfDay(map[string]any{"range": "9-16"}, nil)
	if res != Success {
		t.Errorf("expected Success at 13:00 with range=9-16, got %v", res)
	}
}

func TestCondTimeOfDay_Range_BeforeWindow(t *testing.T) {
	// 8 is representable and before window [9-16).
	defer mustHour(t, 8)()

	res := condTimeOfDay(map[string]any{"range": "9-16"}, nil)
	if res != Failure {
		t.Errorf("expected Failure at 8am with range=9-16, got %v", res)
	}
}

func TestCondTimeOfDay_Range_AtExclusiveEnd(t *testing.T) {
	// 16 is the exclusive end — must Fail.
	defer mustHour(t, 16)()

	res := condTimeOfDay(map[string]any{"range": "9-16"}, nil)
	if res != Failure {
		t.Errorf("expected Failure at 16:00 (exclusive end) with range=9-16, got %v", res)
	}
}

func TestCondTimeOfDay_Range_AfterWindow(t *testing.T) {
	defer mustHour(t, 18)()

	res := condTimeOfDay(map[string]any{"range": "9-16"}, nil)
	if res != Failure {
		t.Errorf("expected Failure at 18:00 with range=9-16, got %v", res)
	}
}

func TestCondTimeOfDay_Range_WrapAroundMidnight_Success(t *testing.T) {
	// range "21-6": covers 21,22,0,1,2,3,4 — all representable.
	for _, hour := range []int{21, 22, 0, 1, 4} {
		hour := hour
		t.Run("", func(t *testing.T) {
			defer mustHour(t, hour)()

			res := condTimeOfDay(map[string]any{"range": "21-6"}, nil)
			if res != Success {
				t.Errorf("expected Success at hour %d with range=21-6 (wrap), got %v", hour, res)
			}
		})
	}
}

func TestCondTimeOfDay_Range_WrapAroundMidnight_Failure(t *testing.T) {
	// range "21-6": hours 6,7,8,...,20 are outside — use 6,7,8,13,19.
	for _, hour := range []int{6, 7, 8, 13, 19} {
		hour := hour
		t.Run("", func(t *testing.T) {
			defer mustHour(t, hour)()

			res := condTimeOfDay(map[string]any{"range": "21-6"}, nil)
			if res != Failure {
				t.Errorf("expected Failure at hour %d with range=21-6 (wrap), got %v", hour, res)
			}
		})
	}
}

func TestCondTimeOfDay_Range_FullDay_AlwaysSuccess(t *testing.T) {
	// 0-24 always succeeds regardless of current hour.
	defer mustHour(t, 0)()

	res := condTimeOfDay(map[string]any{"range": "0-24"}, nil)
	if res != Success {
		t.Errorf("expected Success with range=0-24 (full day), got %v", res)
	}
}

func TestCondTimeOfDay_Range_EmptyRange_AlwaysFailure(t *testing.T) {
	// start == end is always Failure regardless of current hour.
	defer mustHour(t, 3)()

	res := condTimeOfDay(map[string]any{"range": "3-3"}, nil)
	if res != Failure {
		t.Errorf("expected Failure with range=3-3 (empty), got %v", res)
	}
}

func TestCondTimeOfDay_Range_Empty_FallsBackToPeriod(t *testing.T) {
	// Empty range string falls back to binary period evaluation.
	defer mustHour(t, 12)()

	res := condTimeOfDay(map[string]any{"range": "", "period": "day"}, nil)
	if res != Success {
		t.Errorf("expected Success: empty range should fall back to period=day, got %v", res)
	}
}

func TestCondTimeOfDay_Range_Malformed_AbcReturnsFailure(t *testing.T) {
	defer mustHour(t, 12)()

	res := condTimeOfDay(map[string]any{"range": "abc"}, nil)
	if res != Failure {
		t.Errorf("expected Failure with malformed range=abc, got %v", res)
	}
}

func TestCondTimeOfDay_Range_OutOfBounds_25(t *testing.T) {
	defer mustHour(t, 12)()

	res := condTimeOfDay(map[string]any{"range": "9-25"}, nil)
	if res != Failure {
		t.Errorf("expected Failure with out-of-bounds range=9-25, got %v", res)
	}
}

func TestCondTimeOfDay_Range_MissingEnd(t *testing.T) {
	defer mustHour(t, 12)()

	res := condTimeOfDay(map[string]any{"range": "9"}, nil)
	if res != Failure {
		t.Errorf("expected Failure with malformed range=9 (missing end), got %v", res)
	}
}

func TestCondTimeOfDay_Range_NegativeStart(t *testing.T) {
	defer mustHour(t, 12)()

	// "-1-17" -> SplitN gives ["", "1-17"], startStr="" -> invalid.
	res := condTimeOfDay(map[string]any{"range": "-1-17"}, nil)
	if res != Failure {
		t.Errorf("expected Failure with negative-start range=-1-17, got %v", res)
	}
}

func TestCondTimeOfDay_Range_TakesPrecedenceOverPeriod(t *testing.T) {
	// At midnight (hour=0), period=day would Fail, but range=21-6 (wrap) -> Success.
	cleanup := mustHour(t, 0)
	res := condTimeOfDay(map[string]any{
		"period": "day",
		"range":  "21-6",
	}, nil)
	cleanup()
	if res != Success {
		t.Errorf("expected Success at midnight: range=21-6 takes precedence over period=day, got %v", res)
	}

	// At 10am, range=21-6 Fails; range wins over period=day which would pass.
	cleanup = mustHour(t, 10)
	res = condTimeOfDay(map[string]any{
		"period": "day",
		"range":  "21-6",
	}, nil)
	cleanup()
	if res != Failure {
		t.Errorf("expected Failure at 10am: range=21-6 wins over period=day, got %v", res)
	}
}

// ---------------------------------------------------------------------------
// Pure unit tests for parseHourRange and inHourRange helpers
// ---------------------------------------------------------------------------

func TestParseHourRange_ValidCases(t *testing.T) {
	cases := []struct {
		input     string
		wantStart int
		wantEnd   int
		wantValid bool
	}{
		{"9-17", 9, 17, true},
		{"0-24", 0, 24, true},
		{"22-6", 22, 6, true}, // wrap-around
		{"0-0", 0, 0, true},   // empty range (valid parse, Failure at eval)
		{"23-24", 23, 24, true},
		{"0-1", 0, 1, true},
	}
	for _, tc := range cases {
		start, end, valid := parseHourRange(tc.input)
		if valid != tc.wantValid || start != tc.wantStart || end != tc.wantEnd {
			t.Errorf("parseHourRange(%q) = (%d, %d, %v), want (%d, %d, %v)",
				tc.input, start, end, valid, tc.wantStart, tc.wantEnd, tc.wantValid)
		}
	}
}

func TestParseHourRange_InvalidCases(t *testing.T) {
	invalid := []string{
		"abc",   // non-numeric
		"9",     // missing end
		"9-25",  // end out of range
		"-1-17", // negative start (SplitN["","1-17"], startStr="")
		"1--17", // negative end via double dash
		"",      // empty
		"9-",    // missing end value
		"-9",    // only end
	}
	for _, s := range invalid {
		_, _, valid := parseHourRange(s)
		if valid {
			t.Errorf("parseHourRange(%q): expected invalid, got valid", s)
		}
	}
}

func TestInHourRange_LinearAndWrapAround(t *testing.T) {
	// Linear [9, 17): start <= end
	for _, tc := range []struct {
		hour int
		want bool
	}{
		{8, false}, {9, true}, {10, true}, {16, true}, {17, false}, {18, false},
	} {
		got := inHourRange(tc.hour, 9, 17)
		if got != tc.want {
			t.Errorf("inHourRange(%d, 9, 17) = %v, want %v", tc.hour, got, tc.want)
		}
	}

	// Wrap-around [22, 6): covers 22, 23, 0, 1, 2, 3, 4, 5
	for _, tc := range []struct {
		hour int
		want bool
	}{
		{21, false}, {22, true}, {23, true}, {0, true}, {5, true}, {6, false}, {7, false},
	} {
		got := inHourRange(tc.hour, 22, 6)
		if got != tc.want {
			t.Errorf("inHourRange(%d, 22, 6) = %v, want %v", tc.hour, got, tc.want)
		}
	}

	// Edge: full day [0, 24)
	for _, h := range []int{0, 1, 12, 22, 23} {
		if !inHourRange(h, 0, 24) {
			t.Errorf("inHourRange(%d, 0, 24) should be true", h)
		}
	}

	// Edge: empty range [5, 5) — never true
	for _, h := range []int{0, 5, 12, 23} {
		if inHourRange(h, 5, 5) {
			t.Errorf("inHourRange(%d, 5, 5) should be false (empty range)", h)
		}
	}
}

// ---------------------------------------------------------------------------
// condMobAtTargetRoom tests
// ---------------------------------------------------------------------------

func TestCondMobAtTargetRoom_AtTarget(t *testing.T) {
	// Pin the clock to hour 10 so segment [9,18) is active.
	defer mustHour(t, 10)()

	mobs.RegisterScheduleForTest(&mobs.Schedule{
		Id: "test_schedule_at_target",
		Segments: []mobs.ScheduleSegment{
			{Start: 0, End: 9, TargetRoom: 1111},
			{Start: 9, End: 18, TargetRoom: 5678},
			{Start: 18, End: 24, TargetRoom: 9999},
		},
	})
	defer mobs.UnregisterScheduleForTest("test_schedule_at_target")

	cleanMob := seedTestMob(t, 7, 700, 5678, "TestSmith")
	defer cleanMob()

	// Set the schedule and place the mob at the forge (target_room 5678).
	mob := mobs.GetInstance(700)
	mob.ScheduleId = "test_schedule_at_target"
	mob.Character.RoomId = 5678

	ctx := &EvalContext{InstanceId: 700, RoomId: 5678}
	if res := condMobAtTargetRoom(nil, ctx); res != Success {
		t.Errorf("expected Success when mob is at target room, got %v", res)
	}
}

func TestCondMobAtTargetRoom_NotAtTarget(t *testing.T) {
	// Pin to hour 10 so segment [9,18) is active (target_room 5678).
	defer mustHour(t, 10)()

	mobs.RegisterScheduleForTest(&mobs.Schedule{
		Id: "test_schedule_not_at_target",
		Segments: []mobs.ScheduleSegment{
			{Start: 0, End: 9, TargetRoom: 1111},
			{Start: 9, End: 18, TargetRoom: 5678},
			{Start: 18, End: 24, TargetRoom: 9999},
		},
	})
	defer mobs.UnregisterScheduleForTest("test_schedule_not_at_target")

	cleanMob := seedTestMob(t, 8, 800, 1000, "TestSmithWalking")
	defer cleanMob()

	mob := mobs.GetInstance(800)
	mob.ScheduleId = "test_schedule_not_at_target"
	mob.Character.RoomId = 1000 // in transit, not at target

	ctx := &EvalContext{InstanceId: 800, RoomId: 1000}
	if res := condMobAtTargetRoom(nil, ctx); res != Failure {
		t.Errorf("expected Failure when mob is not at target room, got %v", res)
	}
}

func TestCondMobAtTargetRoom_NoSchedule(t *testing.T) {
	cleanMob := seedTestMob(t, 9, 900, 1000, "TestSmithNoSched")
	defer cleanMob()

	// Instance has no ScheduleId — should always Fail.
	mob := mobs.GetInstance(900)
	mob.ScheduleId = ""

	ctx := &EvalContext{InstanceId: 900, RoomId: 1000}
	if res := condMobAtTargetRoom(nil, ctx); res != Failure {
		t.Errorf("expected Failure with no schedule, got %v", res)
	}
}

func TestCondMobAtTargetRoom_NilCtx(t *testing.T) {
	if res := condMobAtTargetRoom(nil, nil); res != Failure {
		t.Errorf("expected Failure for nil context, got %v", res)
	}
}

func TestCondMobAtTargetRoom_UnknownScheduleId(t *testing.T) {
	cleanMob := seedTestMob(t, 10, 1000, 1234, "TestSmithBadSched")
	defer cleanMob()

	mob := mobs.GetInstance(1000)
	mob.ScheduleId = "nonexistent_schedule_xyz"

	ctx := &EvalContext{InstanceId: 1000, RoomId: 1234}
	if res := condMobAtTargetRoom(nil, ctx); res != Failure {
		t.Errorf("expected Failure for unknown schedule id, got %v", res)
	}
}
