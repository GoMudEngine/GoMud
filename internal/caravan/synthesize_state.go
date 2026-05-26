package caravan

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// CaravanPatrolId is the canonical id of the patrol that drives the
// Thornwall ↔ Stillwater caravan. Used by SynthesizeStateForLeader to
// recognize the caravan leader.
const CaravanPatrolId = "caravan_thornwall_stillwater"

// SynthesizeStateForLeader derives the dashboard-facing CaravanState
// from the leader mob's patrol state (waypoint index + arrival_event
// + in-transit flag). Returns (_, false) if the mob is not running
// the caravan patrol.
//
// This replaces the pre-3.7 read of BTreeState["caravan_state"]. The
// patrol layer is now the source of truth for caravan movement state;
// the synthesizer adapts patrol state back into the dashboard's
// canonical state-name vocabulary so the economy-health JSON payload
// is byte-identical post-migration.
//
// Waypoint layout (22 waypoints, strict loop):
//
//	wp0:     room 465   caravan_depot           → ThornwallDwell  (departure, 360-round dwell)
//	wp1:     room 4038  caravan_fernway_pickup   → OutboundFernwayPickup
//	wp2:     room 4109  caravan_depot            → StillwaterDwell (arrival)
//	wp3-10:  rooms 4102..4143  caravan_vendor    → StillwaterRoute
//	wp11:    room 4109  caravan_depot            → StillwaterDwell (departure)
//	wp12:    room 4038  caravan_fernway_pickup   → InboundFernwayPickup
//	wp13:    room 465   caravan_depot            → ThornwallDwell  (arrival)
//	wp14-21: rooms 464..483   caravan_vendor     → ThornwallRoute
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
		// In transit toward waypoint idx. Classify outbound vs inbound by
		// whether the next waypoint is in the outbound half (wp 1–10) or
		// inbound half (wp 11–21) of the cycle.
		if idx <= 10 {
			return StateOutboundTransit, true
		}
		return StateInboundTransit, true
	}

	// At a waypoint — map arrival_event + waypoint role to a state.
	switch wp.ArrivalEvent {
	case "caravan_fernway_pickup":
		if idx == 1 {
			return StateOutboundFernwayPickup, true
		}
		return StateInboundFernwayPickup, true

	case "caravan_depot":
		// Thornwall depot: wp0 (departure, long dwell) and wp13 (arrival).
		// Stillwater depot: wp2 (arrival) and wp11 (departure).
		// Distinguish by room id — both Thornwall stops share room 465.
		if wp.Room == 465 {
			return StateThornwallDwell, true
		}
		return StateStillwaterDwell, true

	case "caravan_vendor":
		// Stillwater vendor circuit: wp3–wp10. Thornwall vendor circuit: wp14–wp21.
		if idx <= 10 {
			return StateStillwaterRoute, true
		}
		return StateThornwallRoute, true
	}

	// Unrecognized arrival_event on a caravan patrol — fall back to
	// ThornwallDwell as the safest default (matches initial-spawn state).
	return StateThornwallDwell, true
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
