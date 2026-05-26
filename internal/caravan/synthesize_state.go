package caravan

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// CaravanPatrolId is the canonical id of the patrol that drives the
// Thornwall ↔ Stillwater caravan. Used by SynthesizeStateForLeader to
// recognize the caravan leader.
const CaravanPatrolId = "caravan_thornwall_stillwater"

// SynthesizeStateForLeader returns the canonical caravan state name
// derived from the leader's patrol state. Used by the economy-health
// dashboard.
//
// Chunk 3.8: with the truncated 4-waypoint main route, the mapping is:
//
//	wp0 (Thornwall depot) → ThornwallRoute if Lars has an active
//	                         oneshot patrol, else ThornwallDwell
//	wp1 (Outbound Fernway) → OutboundFernwayPickup
//	wp2 (Stillwater depot) → StillwaterRoute if Lars active, else
//	                         StillwaterDwell
//	wp3 (Inbound Fernway) → InboundFernwayPickup
//	in-transit             → Outbound/InboundTransit based on next idx
func SynthesizeStateForLeader(leader *mobs.Mob) (CaravanState, bool) {
	if leader == nil || leader.PatrolId != CaravanPatrolId {
		return 0, false
	}
	p := mobs.GetPatrol(CaravanPatrolId)
	if p == nil || len(p.Waypoints) == 0 {
		return 0, false
	}
	idx := miscDataInt(&leader.Character, "patrol_waypoint_idx")
	if idx < 0 || idx >= len(p.Waypoints) {
		idx = 0
	}

	wp := p.Waypoints[idx]
	atWaypoint := leader.Character.RoomId == wp.Room

	if !atWaypoint {
		// In-transit. With 4 waypoints:
		//   heading toward wp1 or wp2 → outbound
		//   heading toward wp3 or wp0 → inbound
		if idx == 1 || idx == 2 {
			return StateOutboundTransit, true
		}
		return StateInboundTransit, true
	}

	// At a waypoint — dispatch on arrival_event.
	switch wp.ArrivalEvent {
	case "caravan_fernway_pickup":
		if idx == 1 {
			return StateOutboundFernwayPickup, true
		}
		return StateInboundFernwayPickup, true
	case "caravan_depot":
		larsActive := isRunnerCircuitActive()
		if idx == 0 {
			if larsActive {
				return StateThornwallRoute, true
			}
			return StateThornwallDwell, true
		}
		// idx == 2 (Stillwater)
		if larsActive {
			return StateStillwaterRoute, true
		}
		return StateStillwaterDwell, true
	}

	// Unrecognized arrival_event — defensive fallback to Thornwall dwell.
	return StateThornwallDwell, true
}

// isRunnerCircuitActive reports whether any instanced Lars (mob 359)
// is currently running one of the runner-circuit oneshot patrols.
// Returns false if Lars isn't currently instanced.
func isRunnerCircuitActive() bool {
	for _, instId := range mobs.GetAllMobInstanceIds() {
		m := mobs.GetInstance(instId)
		if m == nil || int(m.MobId) != RunnerMobId {
			continue
		}
		if _, isRunner := runnerCircuitPatrols[m.PatrolId]; isRunner {
			return true
		}
	}
	return false
}

// miscDataInt reads an integer value from a Character's MiscData map,
// handling both int and float64 (the latter can occur after YAML round-trips).
// Returns 0 if the key is unset or the value is not numeric.
//
// This is a package-local copy; do not import the equivalent from
// internal/hooks to avoid an import cycle.
func miscDataInt(c *characters.Character, key string) int {
	val := c.GetMiscData(key)
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case int:
		return v
	case float64:
		return int(v)
	}
	return 0
}
