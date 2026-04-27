package behaviortree

import (
	"strconv"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/caravan"
	"github.com/GoMudEngine/GoMud/internal/mobs"
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
	result := fn(nil, ctx)
	if result == Failure {
		t.Errorf("caravan_step returned Failure on first tick; want Success")
	}
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
	if got != caravan.StateStillwaterRoute.Name() {
		t.Errorf("after transit arrival, caravan_state = %q, want %q",
			got, caravan.StateStillwaterRoute.Name())
	}
}

func TestActCaravanStep_RouteAdvancesIndexAndExitsAfterAllStops(t *testing.T) {
	fn := LookupAction("caravan_step")

	// last vendor stop in Stillwater route is index 7 (room 4143).
	lastIdx := len(caravan.OutboundRoute.VendorStopIds) - 1
	lastRoom := caravan.OutboundRoute.VendorStopIds[lastIdx]

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
