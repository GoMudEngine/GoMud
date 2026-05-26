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

	// Depot waypoints (wp0, wp2) return *Dwell when Lars has no active
	// runner-circuit patrol. *Route tests require Lars instancing (too
	// involved for unit tests — rely on smoke testing).
	cases := []struct {
		name        string
		waypointIdx int
		want        CaravanState
	}{
		{"wp0 (Thornwall depot)", 0, StateThornwallDwell},
		{"wp1 (Outbound Fernway pickup)", 1, StateOutboundFernwayPickup},
		{"wp2 (Stillwater depot)", 2, StateStillwaterDwell},
		{"wp3 (Inbound Fernway pickup)", 3, StateInboundFernwayPickup},
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

	// Mob is heading toward wp1 (Fernway) — outbound leg.
	mob := &mobs.Mob{PatrolId: CaravanPatrolId}
	mob.Character.SetMiscData("patrol_waypoint_idx", 1)
	mob.Character.RoomId = 9999 // somewhere between waypoints

	got, ok := SynthesizeStateForLeader(mob)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != StateOutboundTransit {
		t.Errorf("got %s, want OutboundTransit when mob is in transit toward wp1",
			got.Name())
	}
}

func TestSynthesizeStateForLeader_InTransitOutboundToDepot(t *testing.T) {
	registerTestCaravanPatrol(t)

	// Mob is heading toward wp2 (Stillwater depot) — still outbound.
	mob := &mobs.Mob{PatrolId: CaravanPatrolId}
	mob.Character.SetMiscData("patrol_waypoint_idx", 2)
	mob.Character.RoomId = 9999 // somewhere between waypoints

	got, ok := SynthesizeStateForLeader(mob)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != StateOutboundTransit {
		t.Errorf("got %s, want OutboundTransit when mob is in transit toward wp2",
			got.Name())
	}
}

func TestSynthesizeStateForLeader_InTransitInbound(t *testing.T) {
	registerTestCaravanPatrol(t)

	// Mob is heading toward wp3 (inbound Fernway) — inbound leg.
	mob := &mobs.Mob{PatrolId: CaravanPatrolId}
	mob.Character.SetMiscData("patrol_waypoint_idx", 3)
	mob.Character.RoomId = 9999 // somewhere between waypoints

	got, ok := SynthesizeStateForLeader(mob)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != StateInboundTransit {
		t.Errorf("got %s, want InboundTransit when mob is in transit toward wp3",
			got.Name())
	}
}

// registerTestCaravanPatrol registers the 4-waypoint caravan patrol shape
// so the synthesizer can resolve it. Mirrors the T15 YAML truncation.
func registerTestCaravanPatrol(t *testing.T) {
	t.Helper()
	mobs.RegisterPatrolForTest(&mobs.Patrol{
		Id:        CaravanPatrolId,
		LoopShape: "strict",
		Waypoints: []mobs.PatrolWaypoint{
			{Room: 465, DwellRounds: 360, ArrivalEvent: "caravan_depot"},         // wp0 Thornwall
			{Room: 4038, DwellRounds: 8, ArrivalEvent: "caravan_fernway_pickup"}, // wp1 outbound Fernway
			{Room: 4109, DwellRounds: 180, ArrivalEvent: "caravan_depot"},        // wp2 Stillwater
			{Room: 4038, DwellRounds: 8, ArrivalEvent: "caravan_fernway_pickup"}, // wp3 inbound Fernway
		},
	})
	t.Cleanup(func() { mobs.UnregisterPatrolForTest(CaravanPatrolId) })
}
