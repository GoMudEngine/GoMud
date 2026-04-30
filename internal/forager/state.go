package forager

// Package forager implements the Stage 3.1 forager NPC state machine.
// State transitions are pure functions; the forager_step btree action
// (internal/behaviortree/actions_forager.go in Task 8) decides WHEN
// to advance.

// ForagerState enumerates the six phases of one forager cycle.
//
// The cycle is:
//
//	Resting → TravelingToTerritory → Foraging → TravelingToDropoff →
//	Delivering → Recalling → (back to Resting)
//
// Most transitions are linear; the HP-emergency branch in
// forager_step can short-circuit any state to Recalling.
type ForagerState int

const (
	StateResting              ForagerState = iota // at sanctuary, recovering
	StateTravelingToTerritory                     // pathto territory entry
	StateForaging                                 // wandering + foraging
	StateTravelingToDropoff                       // pathto vendors / meeting point
	StateDelivering                               // visiting vendors / waiting for caravan
	StateRecalling                                // casting fold-recall home
)

var stateNames = map[ForagerState]string{
	StateResting:              "resting",
	StateTravelingToTerritory: "traveling_to_territory",
	StateForaging:             "foraging",
	StateTravelingToDropoff:   "traveling_to_dropoff",
	StateDelivering:           "delivering",
	StateRecalling:            "recalling",
}

var nameToState = func() map[string]ForagerState {
	m := make(map[string]ForagerState, len(stateNames))
	for s, n := range stateNames {
		m[n] = s
	}
	return m
}()

// Name returns the canonical string for a state, used as the value
// stored in MobState["forager_state"].
func (s ForagerState) Name() string {
	return stateNames[s]
}

// ParseState reverses Name(). Returns (StateResting, false) on
// unknown input — callers should treat !ok as "no state set" and
// default to StateResting.
func ParseState(name string) (ForagerState, bool) {
	s, ok := nameToState[name]
	return s, ok
}

// AdvanceState returns the next state in the cycle. After Recalling
// it wraps back to Resting.
func AdvanceState(cur ForagerState) ForagerState {
	return (cur + 1) % 6
}
