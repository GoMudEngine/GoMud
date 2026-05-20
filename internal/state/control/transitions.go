package control

import "github.com/GoMudEngine/GoMud/internal/state"

// validTransitions enforces the ControlLevel invariant matrix.
//
// Stable states (Controlling, Neutral, Controlled) can transition to
// any of the other stables; transient states (LosingControl,
// BecomingControlled) are entered same-tick during boundary crossings
// and immediately transition to their resolution state.
//
// The machine itself enforces only that transitions follow the
// gradient — the same-tick chaining is implemented in the
// TransitionTo* methods (see control.go).
var validTransitions = state.TransitionTable[State]{
	Controlling:        {LosingControl},
	LosingControl:      {Neutral, Controlling}, // Same-tick resolution to whichever stable state is the target
	Neutral:            {LosingControl, BecomingControlled},
	BecomingControlled: {Neutral, Controlled}, // Same-tick resolution
	Controlled:         {BecomingControlled},
}

// Trigger reason constants.
const (
	// TriggerGrappleEnter fires when a grapple pair enters its
	// initial position. Sets per-side ControlLevel based on the
	// position's symmetry class (see internal/state/position/pair.go).
	TriggerGrappleEnter = "grapple_enter"

	// TriggerDriftWin fires when a side won the per-round drift
	// roll by enough margin to shift state toward Controlling.
	TriggerDriftWin = "drift_win"

	// TriggerDriftLoss fires when a side lost the per-round drift
	// roll by enough margin to shift state toward Controlled.
	TriggerDriftLoss = "drift_loss"

	// TriggerGrappleExit fires when a grapple breaks (escape, death,
	// etc.). Resets ControlLevel to Neutral.
	TriggerGrappleExit = "grapple_exit"
)
