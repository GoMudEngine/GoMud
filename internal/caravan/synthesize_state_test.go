package caravan

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestSynthesizeStateForLeader_NonCaravanMobReturnsFalse(t *testing.T) {
	mob := &mobs.Mob{PatrolId: "thornwall_market_beat"}
	state, ok := SynthesizeStateForLeader(mob)
	if ok {
		t.Errorf("expected (_, false) for non-caravan patrol, got (%v, true)", state)
	}
}

func TestSynthesizeStateForLeader_NoPatrolIdReturnsFalse(t *testing.T) {
	mob := &mobs.Mob{}
	state, ok := SynthesizeStateForLeader(mob)
	if ok {
		t.Errorf("expected (_, false) for mob with no patrol_id, got (%v, true)", state)
	}
}

func TestSynthesizeStateForLeader_NilMobReturnsFalse(t *testing.T) {
	state, ok := SynthesizeStateForLeader(nil)
	if ok {
		t.Errorf("expected (_, false) for nil mob, got (%v, true)", state)
	}
}

func TestSynthesizeStateForLeader_WaypointMapping(t *testing.T) {
	registerTestCaravanPatrol(t)

	cases := []struct {
		name        string
		waypointIdx int
		want        CaravanState
	}{
		{"wp0 (Thornwall depot, departure)", 0, StateThornwallDwell},
		{"wp1 (Outbound Fernway pickup)", 1, StateOutboundFernwayPickup},
		{"wp2 (Stillwater depot, arrival)", 2, StateStillwaterDwell},
		{"wp3-10 (Stillwater vendor circuit)", 5, StateStillwaterRoute},
		{"wp10 (Stillwater vendor circuit end)", 10, StateStillwaterRoute},
		{"wp11 (Stillwater depot, departure)", 11, StateStillwaterDwell},
		{"wp12 (Inbound Fernway pickup)", 12, StateInboundFernwayPickup},
		{"wp13 (Thornwall depot, arrival)", 13, StateThornwallDwell},
		{"wp14-21 (Thornwall vendor circuit)", 17, StateThornwallRoute},
		{"wp21 (Thornwall vendor circuit end)", 21, StateThornwallRoute},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mob := &mobs.Mob{PatrolId: CaravanPatrolId}
			// At-waypoint: mob's room matches the waypoint's room
			p := mobs.GetPatrol(CaravanPatrolId)
			mob.Character.RoomId = p.Waypoints[tc.waypointIdx].Room
			mob.Character.SetMiscData("patrol_waypoint_idx", tc.waypointIdx)
			got, ok := SynthesizeStateForLeader(mob)
			if !ok {
				t.Fatalf("expected ok=true for caravan patrol, got false")
			}
			if got != tc.want {
				t.Errorf("waypoint %d: got state %s, want %s",
					tc.waypointIdx, got.Name(), tc.want.Name())
			}
		})
	}
}

func TestSynthesizeStateForLeader_InTransitOutbound(t *testing.T) {
	registerTestCaravanPatrol(t)

	// Mob is heading toward wp1 from wp0 — in some intermediate room (not 4038).
	mob := &mobs.Mob{PatrolId: CaravanPatrolId}
	mob.Character.SetMiscData("patrol_waypoint_idx", 1)
	mob.Character.RoomId = 9999 // somewhere between waypoints

	got, ok := SynthesizeStateForLeader(mob)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != StateOutboundTransit {
		t.Errorf("got %s, want OutboundTransit when mob is in transit toward wp1 (idx <= 10)",
			got.Name())
	}
}

func TestSynthesizeStateForLeader_InTransitInbound(t *testing.T) {
	registerTestCaravanPatrol(t)

	mob := &mobs.Mob{PatrolId: CaravanPatrolId}
	mob.Character.SetMiscData("patrol_waypoint_idx", 14)
	mob.Character.RoomId = 9999 // somewhere between waypoints

	got, ok := SynthesizeStateForLeader(mob)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != StateInboundTransit {
		t.Errorf("got %s, want InboundTransit when mob is in transit toward wp14 (idx > 10)",
			got.Name())
	}
}

// registerTestCaravanPatrol registers the canonical caravan patrol shape
// so the synthesizer can resolve it. Mirrors the waypoint structure that
// will be authored in T7.
func registerTestCaravanPatrol(t *testing.T) {
	t.Helper()
	mobs.RegisterPatrolForTest(&mobs.Patrol{
		Id:        CaravanPatrolId,
		LoopShape: "strict",
		Waypoints: []mobs.PatrolWaypoint{
			{Room: 465, DwellRounds: 360, ArrivalEvent: "caravan_depot"},         // wp0
			{Room: 4038, DwellRounds: 8, ArrivalEvent: "caravan_fernway_pickup"}, // wp1
			{Room: 4109, DwellRounds: 20, ArrivalEvent: "caravan_depot"},         // wp2
			{Room: 4102, DwellRounds: 5, ArrivalEvent: "caravan_vendor"},         // wp3
			{Room: 4103, DwellRounds: 5, ArrivalEvent: "caravan_vendor"},
			{Room: 4105, DwellRounds: 5, ArrivalEvent: "caravan_vendor"},
			{Room: 4106, DwellRounds: 5, ArrivalEvent: "caravan_vendor"},
			{Room: 4125, DwellRounds: 5, ArrivalEvent: "caravan_vendor"},
			{Room: 4126, DwellRounds: 5, ArrivalEvent: "caravan_vendor"},
			{Room: 4135, DwellRounds: 5, ArrivalEvent: "caravan_vendor"},
			{Room: 4143, DwellRounds: 5, ArrivalEvent: "caravan_vendor"},         // wp10
			{Room: 4109, DwellRounds: 20, ArrivalEvent: "caravan_depot"},         // wp11
			{Room: 4038, DwellRounds: 8, ArrivalEvent: "caravan_fernway_pickup"}, // wp12
			{Room: 465, DwellRounds: 20, ArrivalEvent: "caravan_depot"},          // wp13
			{Room: 464, DwellRounds: 5, ArrivalEvent: "caravan_vendor"},          // wp14
			{Room: 470, DwellRounds: 5, ArrivalEvent: "caravan_vendor"},
			{Room: 471, DwellRounds: 5, ArrivalEvent: "caravan_vendor"},
			{Room: 475, DwellRounds: 5, ArrivalEvent: "caravan_vendor"},
			{Room: 480, DwellRounds: 5, ArrivalEvent: "caravan_vendor"},
			{Room: 481, DwellRounds: 5, ArrivalEvent: "caravan_vendor"},
			{Room: 482, DwellRounds: 5, ArrivalEvent: "caravan_vendor"},
			{Room: 483, DwellRounds: 5, ArrivalEvent: "caravan_vendor"}, // wp21
		},
	})
	t.Cleanup(func() { mobs.UnregisterPatrolForTest(CaravanPatrolId) })
}
