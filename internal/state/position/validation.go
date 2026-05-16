package position

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/state"
)

// PairInvariantViolation describes which of the four invariants
// failed and provides context for logging.
type PairInvariantViolation struct {
	Invariant   string // "single-partner", "bidirectional", "matching-state", "role-exclusivity"
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
//  4. Role-exclusivity: for asymmetric positions (any except Clinch,
//     HalfGuard, Turtle), exactly one side is the controller
//     (ControlLevel ∈ {InControl, LosingControl}) and the other is
//     controlled (∈ {BecomingControlled, Controlled}). Both-Neutral
//     and one-Neutral-one-non-Neutral are violations for asymmetric
//     positions.
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

	// Invariant 4: role-exclusivity for asymmetric positions.
	if !isSymmetricGrapple(stateA) {
		aIsCtrl := IsControllerLevel(dA.ControlLevel)
		bIsCtrl := IsControllerLevel(dB.ControlLevel)
		aIsCd := IsControlledLevel(dA.ControlLevel)
		bIsCd := IsControlledLevel(dB.ControlLevel)

		// Exactly one controller + exactly one controlled required.
		if !(aIsCtrl && bIsCd) && !(aIsCd && bIsCtrl) {
			return PairInvariantViolation{
				Invariant: "role-exclusivity",
				Description: fmt.Sprintf(
					"asymmetric state %v requires one controller + one controlled; got a=%v, b=%v",
					stateA, dA.ControlLevel, dB.ControlLevel),
			}
		}
	}

	return nil
}

// isSymmetricGrapple returns true for grapple states where both
// sides can legitimately hold Neutral ControlLevel (Clinch,
// HalfGuard, Turtle).
func isSymmetricGrapple(s State) bool {
	return s == Clinch || s == HalfGuard || s == Turtle
}
