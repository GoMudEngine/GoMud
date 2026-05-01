package caravan

// CaravanState enumerates the eight phases of one caravan cycle.
//
// The cycle is:
//
//	ThornwallDwell → OutboundTransit → OutboundFernwayPickup → StillwaterRoute →
//	StillwaterDwell → InboundTransit → InboundFernwayPickup → ThornwallRoute → (back to top)
//
// State transitions are driven by the caravan_step btree action reading
// environmental context (current room, dwell timer, route progress).
// AdvanceState is a pure function; the action decides WHEN to advance.
type CaravanState int

const (
	StateThornwallDwell        CaravanState = iota
	StateOutboundTransit                    // path to Fernway meeting point
	StateOutboundFernwayPickup              // dwell at 4038, handoff if forager present
	StateStillwaterRoute
	StateStillwaterDwell
	StateInboundTransit                     // path to Fernway meeting point
	StateInboundFernwayPickup               // dwell at 4038, handoff if forager present
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

var nameToState = func() map[string]CaravanState {
	m := make(map[string]CaravanState, len(stateNames))
	for s, n := range stateNames {
		m[n] = s
	}
	return m
}()

// Name returns the canonical string for a state, used as the value
// stored in MobState["caravan_state"].
func (s CaravanState) Name() string {
	return stateNames[s]
}

// ParseState reverses Name(). Returns (StateThornwallDwell, false) on
// unknown input — callers should treat !ok as "no state set" and
// default to StateThornwallDwell.
func ParseState(name string) (CaravanState, bool) {
	s, ok := nameToState[name]
	return s, ok
}

// AdvanceState returns the next state in the cycle. After
// StateThornwallRoute it wraps back to StateThornwallDwell.
func AdvanceState(cur CaravanState) CaravanState {
	return (cur + 1) % 8
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

// RouteForState returns a pointer to the Route that owns this state's
// transit + visit, or nil for dwell states.
func RouteForState(s CaravanState) *Route {
	switch s {
	case StateOutboundTransit, StateOutboundFernwayPickup, StateStillwaterRoute:
		return &OutboundRoute
	case StateInboundTransit, StateInboundFernwayPickup, StateThornwallRoute:
		return &InboundRoute
	}
	return nil
}
