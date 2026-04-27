package caravan

// CaravanState enumerates the six phases of one caravan cycle.
//
// The cycle is:
//
//	ThornwallDwell → OutboundTransit → StillwaterRoute →
//	StillwaterDwell → InboundTransit → ThornwallRoute → (back to top)
//
// State transitions are driven by the caravan_step btree action reading
// environmental context (current room, dwell timer, route progress).
// AdvanceState is a pure function; the action decides WHEN to advance.
type CaravanState int

const (
	StateThornwallDwell  CaravanState = iota
	StateOutboundTransit
	StateStillwaterRoute
	StateStillwaterDwell
	StateInboundTransit
	StateThornwallRoute
)

var stateNames = map[CaravanState]string{
	StateThornwallDwell:  "thornwall_dwell",
	StateOutboundTransit: "outbound_transit",
	StateStillwaterRoute: "stillwater_route",
	StateStillwaterDwell: "stillwater_dwell",
	StateInboundTransit:  "inbound_transit",
	StateThornwallRoute:  "thornwall_route",
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
	return (cur + 1) % 6
}

// IsDwellState reports whether the caravan is at a depot waiting for
// the dwell timer to expire.
func IsDwellState(s CaravanState) bool {
	return s == StateThornwallDwell || s == StateStillwaterDwell
}

// IsTransitState reports whether the caravan is in long-haul travel
// between depots.
func IsTransitState(s CaravanState) bool {
	return s == StateOutboundTransit || s == StateInboundTransit
}

// IsRouteState reports whether the caravan is visiting vendor stops
// in the destination town.
func IsRouteState(s CaravanState) bool {
	return s == StateStillwaterRoute || s == StateThornwallRoute
}

// RouteForState returns a pointer to the Route that owns this state's
// transit + visit, or nil for dwell states.
func RouteForState(s CaravanState) *Route {
	switch s {
	case StateOutboundTransit, StateStillwaterRoute:
		return &OutboundRoute
	case StateInboundTransit, StateThornwallRoute:
		return &InboundRoute
	}
	return nil
}
