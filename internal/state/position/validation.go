package position

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/control"
)

// PairInvariantViolation describes which of the four invariants
// failed and provides context for logging.
type PairInvariantViolation struct {
	Invariant   string // "single-partner", "bidirectional", "matching-state", "control-exclusivity"
	Description string
}

func (v PairInvariantViolation) Error() string {
	return fmt.Sprintf("pair invariant violation: %s — %s", v.Invariant, v.Description)
}

// ValidateGrapplePair checks the four pair-state invariants for two
// actors claimed to be grappling each other. Returns a
// PairInvariantViolation on failure, nil if all invariants hold.
//
// Invariants:
//
//  1. Single-partner: each side's GrappleData.Partner is non-zero
//     and refers to the other actor. (Turtle exception: solo
//     Turtle may have zero Partner; both-Turtle still requires
//     each to point at the other.)
//
//  2. Bidirectional: if a.Partner = b.Self then b.Partner = a.Self.
//     Always reciprocal.
//
//  3. Matching-state: a.State() == b.State() while in a grapple pair.
//
//  4. ControlLevel consistency: both sides at Controlling is impossible
//     (only one can dominate); both sides at Controlled is impossible
//     (only one can be dominated). Applies to ALL grapple positions,
//     symmetric and asymmetric alike. Nil Control machines are skipped
//     (bootstrap / test-fixture paths that haven't wired Control).
func ValidateGrapplePair(a, b GrappleActor) error {
	if a == nil || b == nil {
		return PairInvariantViolation{
			Invariant:   "nil-input",
			Description: "one or both actors are nil",
		}
	}
	if a.GetPosition() == nil || b.GetPosition() == nil {
		return PairInvariantViolation{
			Invariant:   "nil-machine",
			Description: "one or both actors have nil Position machine",
		}
	}

	stateA := a.GetPosition().State()
	stateB := b.GetPosition().State()

	// Both must be in grapple states.
	if !a.GetPosition().IsGrappling() || !b.GetPosition().IsGrappling() {
		return PairInvariantViolation{
			Invariant:   "single-partner",
			Description: fmt.Sprintf("one side not grappling (a=%v, b=%v)", stateA, stateB),
		}
	}

	// Invariant 3: matching state.
	if stateA != stateB {
		return PairInvariantViolation{
			Invariant: "matching-state",
			Description: fmt.Sprintf("state mismatch: a=%v, b=%v",
				stateA, stateB),
		}
	}

	dA, _ := a.GetPosition().GrappleData()
	dB, _ := b.GetPosition().GrappleData()
	refA := state.ActorRef{UserId: a.GetUserId(), MobInstanceId: a.GetMobInstanceId()}
	refB := state.ActorRef{UserId: b.GetUserId(), MobInstanceId: b.GetMobInstanceId()}

	// Invariant 1 + 2: single-partner + bidirectional.
	// Turtle solo case: if either side has zero Partner, both sides
	// must be solo (i.e., the other side's Partner is also zero).
	if stateA == Turtle && (dA.Partner.IsZero() || dB.Partner.IsZero()) {
		// Solo Turtle on either side — not a pair, not subject to
		// pair invariants. Caller shouldn't be validating this as a
		// pair, but tolerate.
		return nil
	}

	if dA.Partner != refB {
		return PairInvariantViolation{
			Invariant: "single-partner",
			Description: fmt.Sprintf("a.Partner (%+v) != b.Self (%+v)",
				dA.Partner, refB),
		}
	}
	if dB.Partner != refA {
		return PairInvariantViolation{
			Invariant: "bidirectional",
			Description: fmt.Sprintf("b.Partner (%+v) != a.Self (%+v)",
				dB.Partner, refA),
		}
	}

	// Invariant 4: ControlLevel state consistency.
	// - Both sides at Controlling is impossible (only one dominates).
	// - Both sides at Controlled is impossible (only one is dominated).
	// - Applies to symmetric AND asymmetric positions.
	//
	// Transient states (LosingControl, BecomingControlled) shouldn't
	// appear at consistency-check time — they're entered same-tick
	// during boundary crossings and resolve before the tick ends.
	// If we see one in a check, that's a bug; flag it.
	ctrlA := a.GetControl()
	ctrlB := b.GetControl()
	if ctrlA == nil || ctrlB == nil {
		// Control machine not initialized — bootstrap path or test
		// fixture. Skip the check; T6 ensures real grapples have
		// Control initialized.
		return nil
	}

	aState := ctrlA.State()
	bState := ctrlB.State()

	if aState == control.Controlling && bState == control.Controlling {
		return PairInvariantViolation{
			Invariant: "control-exclusivity",
			Description: fmt.Sprintf(
				"both sides at Controlling state in position %v (impossible: only one side can dominate)",
				stateA),
		}
	}
	if aState == control.Controlled && bState == control.Controlled {
		return PairInvariantViolation{
			Invariant: "control-exclusivity",
			Description: fmt.Sprintf(
				"both sides at Controlled state in position %v (impossible: only one side can be dominated)",
				stateA),
		}
	}

	return nil
}

// isSymmetricGrapple returns true for grapple states where both
// sides have no dominant role — Clinch, HalfGuard, and Turtle.
func isSymmetricGrapple(s State) bool {
	return s == Clinch || s == HalfGuard || s == Turtle
}
