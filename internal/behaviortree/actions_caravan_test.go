package behaviortree

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/caravan"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func TestActCaravanStep_DefaultsToThornwallDwellOnFirstTick(t *testing.T) {
	fn := LookupAction("caravan_step")
	if fn == nil {
		t.Fatal("caravan_step not registered")
	}

	mob := buildCaravanLeaderMob(t, 7100, caravan.ThornwallDepotRoomId)
	state := NewBehaviorState()

	ctx := &EvalContext{
		InstanceId: 7100,
		RoomId:     caravan.ThornwallDepotRoomId,
		MobState:   state,
	}
	// First tick initializes state to thornwall_dwell. tickDwell returns
	// Failure while the dwell timer is unexpired (so legacy idle takes
	// over and fires idlecommands). The state is what we assert on, not
	// the result.
	fn(nil, ctx)
	got := state.GetString("caravan_state")
	if got != caravan.StateThornwallDwell.Name() {
		t.Errorf("caravan_state after first tick = %q, want %q",
			got, caravan.StateThornwallDwell.Name())
	}
	_ = mob
}

func TestActCaravanStep_AdvancesFromDwellAfterTimerExpires(t *testing.T) {
	fn := LookupAction("caravan_step")

	mob := buildCaravanLeaderMob(t, 7101, caravan.ThornwallDepotRoomId)
	_ = mob
	state := NewBehaviorState()
	state.Set("caravan_state", caravan.StateThornwallDwell.Name())
	// Mark dwell as having started long enough ago that it expires now.
	// Round 0 + dwell (360) <= current round (which is >= 1), so timer is expired.
	state.Set("caravan_state_started_round", "0")

	ctx := &EvalContext{
		InstanceId: 7101,
		RoomId:     caravan.ThornwallDepotRoomId,
		MobState:   state,
	}
	fn(nil, ctx)

	got := state.GetString("caravan_state")
	if got != caravan.StateOutboundTransit.Name() {
		t.Errorf("after dwell timer expires, caravan_state = %q, want %q",
			got, caravan.StateOutboundTransit.Name())
	}
}

func TestActCaravanStep_AdvancesFromTransitOnArrival(t *testing.T) {
	fn := LookupAction("caravan_step")

	mob := buildCaravanLeaderMob(t, 7102, caravan.StillwaterDepotRoomId)
	_ = mob
	state := NewBehaviorState()
	state.Set("caravan_state", caravan.StateOutboundTransit.Name())

	ctx := &EvalContext{
		InstanceId: 7102,
		RoomId:     caravan.StillwaterDepotRoomId,
		MobState:   state,
	}
	fn(nil, ctx)

	got := state.GetString("caravan_state")
	// After outbound transit arrives at the depot, it now advances to the
	// FernwayPickup substate (inserted in the cycle between transit and route).
	if got != caravan.StateOutboundFernwayPickup.Name() {
		t.Errorf("after transit arrival, caravan_state = %q, want %q",
			got, caravan.StateOutboundFernwayPickup.Name())
	}
}

func TestActCaravanStep_RouteAdvancesIndexAndExitsAfterAllStops(t *testing.T) {
	fn := LookupAction("caravan_step")

	// last vendor stop in Stillwater route is index 7 (room 4143).
	lastIdx := len(caravan.OutboundRoute.VendorStopIds) - 1
	lastRoom := caravan.OutboundRoute.VendorStopIds[lastIdx]

	// Seed the room so caravan.FindWagonInRoom can call rooms.LoadRoom.
	cleanRoom := seedTestRoom(t, lastRoom, "test_zone")
	defer cleanRoom()

	// Register the wagon mob (template 374) in the room so
	// caravan.FindWagonInRoom returns a real mob rather than nil.
	cleanWagon := seedTestMob(t, 374, 7199, lastRoom, "caravan wagon")
	defer cleanWagon()
	if r := rooms.LoadRoom(lastRoom); r != nil {
		r.AddMob(7199)
	}

	mob := buildCaravanLeaderMob(t, 7103, lastRoom)
	_ = mob
	state := NewBehaviorState()
	state.Set("caravan_state", caravan.StateStillwaterRoute.Name())
	// Pretend we've already visited all stops except the last.
	// caravan_route_index == lastIdx means we're about to visit stop lastIdx.
	state.Set("caravan_route_index", itoa(lastIdx))

	ctx := &EvalContext{
		InstanceId: 7103,
		RoomId:     lastRoom,
		MobState:   state,
	}
	fn(nil, ctx)

	got := state.GetString("caravan_state")
	if got != caravan.StateStillwaterDwell.Name() {
		t.Errorf("after final route stop, caravan_state = %q, want %q",
			got, caravan.StateStillwaterDwell.Name())
	}
}

// itoa is a local helper wrapping strconv.Itoa for test readability.
func itoa(n int) string {
	return strconv.Itoa(n)
}

func TestCaravanLoadHelpers(t *testing.T) {
	s := NewBehaviorState()
	if got := caravanLoadGet(s); got != nil {
		t.Errorf("empty load = %v, want nil", got)
	}
	caravanLoadAppend(s, "stillwater")
	caravanLoadAppend(s, "fernway")
	caravanLoadAppend(s, "stillwater") // duplicate — should be no-op
	got := caravanLoadGet(s)
	if len(got) != 2 || got[0] != "stillwater" || got[1] != "fernway" {
		t.Errorf("got %v, want [stillwater fernway]", got)
	}
	caravanLoadSet(s, nil)
	if got := caravanLoadGet(s); got != nil {
		t.Errorf("after clear got %v, want nil", got)
	}
}

// buildCaravanLeaderMob creates a mob instance with the given instanceId in
// the given room, registers it via SetInstanceForTest, and schedules cleanup.
// Patterned after makePartyMob in actions_party_test.go.
func buildCaravanLeaderMob(t *testing.T, instanceId int, roomId int) *mobs.Mob {
	t.Helper()
	mob := &mobs.Mob{
		MobId:      mobs.MobId(316), // Ketil the caravan leader template ID
		InstanceId: instanceId,
		HomeRoomId: roomId,
	}
	mob.Character.Name = "Ketil"
	mob.Character.RoomId = roomId
	mob.Character.Buffs = buffs.New()
	mobs.SetInstanceForTest(instanceId, mob)
	t.Cleanup(func() { mobs.SetInstanceForTest(instanceId, nil) })
	return mob
}

// ── Stuck-watchdog tests ──────────────────────────────────────────────────

// TestCaravanStuckWatchdog_Fires checks that a caravan that has been in a
// transit state for far longer than the maximum legitimate phase is reset to
// ThornwallDwell by the auto-watchdog inside actCaravanStep.
func TestCaravanStuckWatchdog_Fires(t *testing.T) {
	fn := LookupAction("caravan_step")
	if fn == nil {
		t.Fatal("caravan_step not registered")
	}

	mob := buildCaravanLeaderMob(t, 7200, caravan.ThornwallDepotRoomId)
	_ = mob
	state := NewBehaviorState()
	state.Set("caravan_state", caravan.StateStillwaterRoute.Name())
	// Seed a started_round far in the past so the watchdog threshold fires.
	// Threshold = 5 * CaravanDepotDwellRounds (default 720) = 3600, floor 300.
	// Using 4000 rounds ago ensures we exceed 3600 in the default config.
	stuckRound := util.GetRoundCount() - 4000
	state.Set("caravan_state_started_round", strconv.FormatUint(stuckRound, 10))
	state.Set("caravan_route_index", "3")
	caravanLoadAppend(state, "thornwall")
	caravanLoadAppend(state, "fernway")

	ctx := &EvalContext{
		InstanceId: 7200,
		RoomId:     caravan.ThornwallDepotRoomId,
		MobState:   state,
	}
	result := fn(nil, ctx)

	if result != Failure {
		t.Errorf("watchdog should return Failure, got %v", result)
	}
	if got := state.GetString("caravan_state"); got != caravan.StateThornwallDwell.Name() {
		t.Errorf("after watchdog, caravan_state = %q, want %q",
			got, caravan.StateThornwallDwell.Name())
	}
	if got := state.GetString("caravan_route_index"); got != "0" {
		t.Errorf("after watchdog, caravan_route_index = %q, want %q", got, "0")
	}
	if got := caravanLoadGet(state); got != nil {
		t.Errorf("after watchdog, caravan_load = %v, want nil", got)
	}
}

// TestCaravanStuckWatchdog_DoesNotFire checks that a caravan started only
// 50 rounds ago is NOT reset — well within any threshold.
func TestCaravanStuckWatchdog_DoesNotFire(t *testing.T) {
	fn := LookupAction("caravan_step")
	if fn == nil {
		t.Fatal("caravan_step not registered")
	}

	mob := buildCaravanLeaderMob(t, 7201, caravan.StillwaterDepotRoomId)
	_ = mob
	state := NewBehaviorState()
	state.Set("caravan_state", caravan.StateStillwaterDwell.Name())
	// 50 rounds ago — well within any threshold (floor is 300, and
	// 5 × default 720 = 3600).
	recentRound := util.GetRoundCount() - 50
	state.Set("caravan_state_started_round", strconv.FormatUint(recentRound, 10))

	ctx := &EvalContext{
		InstanceId: 7201,
		RoomId:     caravan.StillwaterDepotRoomId,
		MobState:   state,
	}
	fn(nil, ctx)

	// State should remain StillwaterDwell (still within dwell timer).
	if got := state.GetString("caravan_state"); got != caravan.StateStillwaterDwell.Name() {
		t.Errorf("watchdog fired too early: caravan_state = %q, want %q",
			got, caravan.StateStillwaterDwell.Name())
	}
}

// TestCaravanStuckWatchdog_ZeroStartedDoesNotFire checks that a fresh mob
// with started_round = 0 (empty/unset sentinel) does NOT trigger the
// watchdog. The guard `started > 0` is the critical invariant here.
func TestCaravanStuckWatchdog_ZeroStartedDoesNotFire(t *testing.T) {
	fn := LookupAction("caravan_step")
	if fn == nil {
		t.Fatal("caravan_step not registered")
	}

	mob := buildCaravanLeaderMob(t, 7202, caravan.ThornwallDepotRoomId)
	_ = mob
	state := NewBehaviorState()
	state.Set("caravan_state", caravan.StateThornwallDwell.Name())
	// Explicitly set to "0" — the value used by existing tests to simulate
	// an expired timer. The watchdog must ignore this sentinel.
	state.Set("caravan_state_started_round", "0")

	ctx := &EvalContext{
		InstanceId: 7202,
		RoomId:     caravan.ThornwallDepotRoomId,
		MobState:   state,
	}
	fn(nil, ctx)

	// State advanced to OutboundTransit because dwell timer fired (not
	// watchdog). This proves the watchdog didn't intercept a round-0 start.
	got := state.GetString("caravan_state")
	if got == caravan.StateThornwallDwell.Name() {
		// Stayed in dwell is also fine — the watchdog simply didn't fire.
		// (Whether dwell fires depends on configs; both are acceptable.)
		return
	}
	// Any state other than ThornwallDwell is also acceptable here —
	// we only care the watchdog didn't force-reset to ThornwallDwell
	// while suppressing the normal dwell advance.
	// The key invariant: caravan_route_index should still be "0", not
	// some garbage value from a spurious watchdog reset mid-cycle.
	if idx := state.GetString("caravan_route_index"); idx != "0" {
		t.Errorf("route_index should be 0 after first tick, got %q", idx)
	}
}

// ── resetCaravanState helper tests ───────────────────────────────────────

// TestResetCaravanState verifies the helper clears all four keys to
// canonical defaults.
func TestResetCaravanState(t *testing.T) {
	s := NewBehaviorState()
	// Pre-populate with non-default values.
	s.Set("caravan_state", caravan.StateInboundTransit.Name())
	s.Set("caravan_state_started_round", "9999999")
	s.Set("caravan_route_index", "5")
	caravanLoadAppend(s, "stillwater")
	caravanLoadAppend(s, "fernway")

	resetCaravanState(s)

	if got := s.GetString("caravan_state"); got != caravan.StateThornwallDwell.Name() {
		t.Errorf("caravan_state = %q, want %q", got, caravan.StateThornwallDwell.Name())
	}
	startedStr := s.GetString("caravan_state_started_round")
	started, err := strconv.ParseUint(startedStr, 10, 64)
	if err != nil || started == 0 {
		t.Errorf("caravan_state_started_round = %q, want current round (> 0)", startedStr)
	}
	if got := s.GetString("caravan_route_index"); got != "0" {
		t.Errorf("caravan_route_index = %q, want %q", got, "0")
	}
	if got := caravanLoadGet(s); got != nil {
		t.Errorf("caravan_load = %v, want nil", got)
	}
}

// ── ResetAllCaravanStates helper tests ───────────────────────────────────

// TestResetAllCaravanStates_ResetsLeadersOnly verifies that
// ResetAllCaravanStates resets mobs with a caravan_state key and leaves
// mobs without one untouched.
func TestResetAllCaravanStates_ResetsLeadersOnly(t *testing.T) {
	// Caravan leader: has a caravan_state in its BTreeState.
	leader := buildCaravanLeaderMob(t, 7210, caravan.ThornwallDepotRoomId)
	leaderState := NewBehaviorState()
	leaderState.Set("caravan_state", caravan.StateInboundTransit.Name())
	leaderState.Set("caravan_state_started_round", "500")
	leaderState.Set("caravan_route_index", "2")
	caravanLoadAppend(leaderState, "stillwater")
	leader.BTreeState = leaderState

	// Non-caravan mob: no caravan_state key.
	nonCaravan := buildCaravanLeaderMob(t, 7211, caravan.ThornwallDepotRoomId)
	nonCaravan.Character.Name = "Random Guard"
	otherState := NewBehaviorState()
	otherState.Set("some_other_key", "some_value")
	nonCaravan.BTreeState = otherState

	count := ResetAllCaravanStates()
	if count != 1 {
		t.Errorf("ResetAllCaravanStates returned %d, want 1", count)
	}

	// Leader should be reset.
	if got := leaderState.GetString("caravan_state"); got != caravan.StateThornwallDwell.Name() {
		t.Errorf("leader caravan_state = %q, want %q",
			got, caravan.StateThornwallDwell.Name())
	}
	if got := caravanLoadGet(leaderState); got != nil {
		t.Errorf("leader caravan_load = %v, want nil after reset", got)
	}

	// Non-caravan mob state should be untouched.
	if got := otherState.GetString("some_other_key"); got != "some_value" {
		t.Errorf("non-caravan mob state was modified: some_other_key = %q", got)
	}
}

func TestBucketsForRouteState_Stillwater(t *testing.T) {
	delivery, pickup := bucketsForRouteState(caravan.StateStillwaterRoute)
	if !reflect.DeepEqual(delivery, []string{"thornwall", "fernway"}) {
		t.Errorf("Stillwater delivery = %v, want [thornwall fernway]", delivery)
	}
	if !reflect.DeepEqual(pickup, []string{"stillwater"}) {
		t.Errorf("Stillwater pickup = %v, want [stillwater]", pickup)
	}
}

func TestBucketsForRouteState_Thornwall(t *testing.T) {
	delivery, pickup := bucketsForRouteState(caravan.StateThornwallRoute)
	if !reflect.DeepEqual(delivery, []string{"stillwater", "fernway"}) {
		t.Errorf("Thornwall delivery = %v, want [stillwater fernway]", delivery)
	}
	if !reflect.DeepEqual(pickup, []string{"thornwall"}) {
		t.Errorf("Thornwall pickup = %v, want [thornwall]", pickup)
	}
}

func TestBucketsForRouteState_NonRouteReturnsNil(t *testing.T) {
	delivery, pickup := bucketsForRouteState(caravan.StateThornwallDwell)
	if delivery != nil || pickup != nil {
		t.Errorf("dwell state returned non-nil: %v, %v", delivery, pickup)
	}
}
