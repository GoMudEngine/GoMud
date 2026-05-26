package caravan

// CaravanState enumerates the eight phases of one caravan cycle.
//
// The cycle is:
//
//	ThornwallDwell → OutboundTransit → OutboundFernwayPickup → StillwaterRoute →
//	StillwaterDwell → InboundTransit → InboundFernwayPickup → ThornwallRoute → (back to top)
//
// Chunk 3.7: state is synthesized from patrol position by
// SynthesizeStateForLeader; the btree caravan_step driver is removed.
type CaravanState int

const (
	StateThornwallDwell        CaravanState = iota
	StateOutboundTransit                    // path to Fernway meeting point
	StateOutboundFernwayPickup              // dwell at 4038, handoff if forager present
	StateStillwaterRoute
	StateStillwaterDwell
	StateInboundTransit  // path to Fernway meeting point
	StateInboundFernwayPickup              // dwell at 4038, handoff if forager present
	StateThornwallRoute
)

var stateNames = map[CaravanState]string{
	StateThornwallDwell:        "thornwall_dwell",
	StateOutboundTransit:       "outbound_transit",
	StateOutboundFernwayPickup: "outbound_fernway_pickup",
	StateStillwaterRoute:       "stillwater_route",
	StateStillwaterDwell:       "stillwater_dwell",
	StateInboundTransit:        "inbound_transit",
	StateInboundFernwayPickup:  "inbound_fernway_pickup",
	StateThornwallRoute:        "thornwall_route",
}

// Name returns the canonical string for a state, used in the economy
// dashboard JSON payload.
func (s CaravanState) Name() string {
	return stateNames[s]
}

// IsDwellState reports whether the caravan is at a depot waiting for
// the dwell timer to expire.
func IsDwellState(s CaravanState) bool {
	return s == StateThornwallDwell || s == StateStillwaterDwell
}

// IsTransitState reports whether the caravan is in long-haul travel
// between depots — including the brief Fernway-pickup substates that
// sit inside each transit leg.
func IsTransitState(s CaravanState) bool {
	return s == StateOutboundTransit ||
		s == StateInboundTransit ||
		s == StateOutboundFernwayPickup ||
		s == StateInboundFernwayPickup
}

// IsRouteState reports whether the caravan is visiting vendor stops
// in the destination town.
func IsRouteState(s CaravanState) bool {
	return s == StateStillwaterRoute || s == StateThornwallRoute
}

// IsFernwayPickupState reports whether the caravan is at the Fernway
// meeting point waiting for the forager handoff.
func IsFernwayPickupState(s CaravanState) bool {
	return s == StateOutboundFernwayPickup ||
		s == StateInboundFernwayPickup
}
