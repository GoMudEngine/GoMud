package state

// VetoHandler may return a non-nil error to block a pending
// transition. The framework runs vetoes in registration order;
// the first non-nil return halts the chain and is returned to
// the TransitionTo caller. No state change occurs.
type VetoHandler[S comparable] func(from, to S, reason TransitionReason) error

// CascadeHandler runs after a successful state change in the
// same stack frame. May call TransitionTo on this or other
// machines. Fires in registration order; all cascade handlers
// run (none short-circuit).
type CascadeHandler[S comparable] func(from, to S, reason TransitionReason)

// ObserverHandler is a pure observer. Cannot veto, does not
// participate in cascade ordering. Used by quest engine,
// aliveness substrate, telemetry. Fires after all cascade
// handlers complete.
type ObserverHandler[S comparable] func(from, to S, reason TransitionReason)
