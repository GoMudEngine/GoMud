package state

import "errors"

// ActorRef is a discriminated reference to either a user or mob.
// Zero value means "no actor."
type ActorRef struct {
	UserId        int
	MobInstanceId int
}

// IsZero returns true if the ActorRef points to nobody.
func (a ActorRef) IsZero() bool {
	return a.UserId == 0 && a.MobInstanceId == 0
}

// IsPlayer returns true if the ActorRef is a player target.
func (a ActorRef) IsPlayer() bool {
	return a.UserId > 0
}

// IsMob returns true if the ActorRef is a mob target.
func (a ActorRef) IsMob() bool {
	return a.MobInstanceId > 0
}

// TransitionReason captures why a state transition occurred.
// Used by veto, cascade, and observer handlers to decide
// whether to react and how. Propagated through the entire
// transition chain.
type TransitionReason struct {
	// Trigger is a stable string identifier ("attack_command",
	// "flee_success", "target_died", "surprise_attack", etc.)
	Trigger string

	// Actor is who initiated the transition (may be the same
	// character whose state is changing, or another character).
	Actor ActorRef

	// Target is an optional companion ActorRef (the combat
	// target for Engaging transitions, the killer for death
	// transitions, etc.).
	Target ActorRef

	// Metadata is open key-value storage for transition-specific
	// data (e.g., damage amount on a hurt transition).
	Metadata map[string]any
}

// TransitionTable maps "from" states to the set of valid
// "to" states. Used by Machine to reject framework-invariant
// violations before any veto handler runs.
type TransitionTable[S comparable] map[S][]S

// IsAllowed returns true if from→to is in the table.
func (t TransitionTable[S]) IsAllowed(from, to S) bool {
	for _, allowed := range t[from] {
		if allowed == to {
			return true
		}
	}
	return false
}

// Framework error types.
var (
	ErrInvalidTransition = errors.New("state: transition not allowed by table")
	ErrVetoed            = errors.New("state: transition vetoed")
)

// VetoError wraps a veto reason with the vetoing handler's
// identifier for debugging.
type VetoError struct {
	HandlerName string
	Reason      string
}

func (e *VetoError) Error() string {
	return "state: vetoed by " + e.HandlerName + ": " + e.Reason
}

func (e *VetoError) Unwrap() error { return ErrVetoed }
