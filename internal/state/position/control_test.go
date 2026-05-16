package position_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/state/position"
)

func TestShiftControl_TowardInControl(t *testing.T) {
	got := position.ShiftControl(position.Controlled, -2)
	want := position.Neutral
	if got != want {
		t.Errorf("ShiftControl(Controlled, -2) = %v, want %v", got, want)
	}
}

func TestShiftControl_TowardControlled(t *testing.T) {
	got := position.ShiftControl(position.InControl, 3)
	want := position.BecomingControlled
	if got != want {
		t.Errorf("ShiftControl(InControl, 3) = %v, want %v", got, want)
	}
}

func TestShiftControl_ClampsHigh(t *testing.T) {
	got := position.ShiftControl(position.Neutral, 100)
	if got != position.Controlled {
		t.Errorf("ShiftControl should clamp at Controlled; got %v", got)
	}
}

func TestShiftControl_ClampsLow(t *testing.T) {
	got := position.ShiftControl(position.Neutral, -100)
	if got != position.InControl {
		t.Errorf("ShiftControl should clamp at InControl; got %v", got)
	}
}

func TestMarginToDelta_NoShift(t *testing.T) {
	if got := position.MarginToDelta(0.3); got != 0 {
		t.Errorf("MarginToDelta(0.3) = %d, want 0", got)
	}
}

func TestMarginToDelta_OneLevel(t *testing.T) {
	if got := position.MarginToDelta(0.7); got != 1 {
		t.Errorf("MarginToDelta(0.7) = %d, want 1", got)
	}
}

func TestMarginToDelta_TwoLevels(t *testing.T) {
	if got := position.MarginToDelta(1.5); got != 2 {
		t.Errorf("MarginToDelta(1.5) = %d, want 2", got)
	}
}

func TestMarginToDelta_Crit(t *testing.T) {
	if got := position.MarginToDelta(2.5); got != 3 {
		t.Errorf("MarginToDelta(2.5) = %d, want 3", got)
	}
}

func TestInitialControlForPair_Mount(t *testing.T) {
	if got := position.InitialControlForPair(position.Mount, position.RoleController); got != position.InControl {
		t.Errorf("InitialControlForPair(Mount, Controller) = %v, want InControl", got)
	}
	if got := position.InitialControlForPair(position.Mount, position.RoleControlled); got != position.Controlled {
		t.Errorf("InitialControlForPair(Mount, Controlled) = %v, want Controlled", got)
	}
}

func TestInitialControlForPair_GuardInversion(t *testing.T) {
	// Guard's "controller" (per our naming) is the BOTTOM person
	// trapping top with legs. Bottom starts InControl.
	if got := position.InitialControlForPair(position.Guard, position.RoleController); got != position.InControl {
		t.Errorf("InitialControlForPair(Guard, Controller) = %v, want InControl (bottom is controller)", got)
	}
}

func TestInitialControlForPair_SymmetricClinch(t *testing.T) {
	if got := position.InitialControlForPair(position.Clinch, position.RoleController); got != position.Neutral {
		t.Errorf("InitialControlForPair(Clinch, Controller) = %v, want Neutral (symmetric)", got)
	}
	if got := position.InitialControlForPair(position.Clinch, position.RoleControlled); got != position.Neutral {
		t.Errorf("InitialControlForPair(Clinch, Controlled) = %v, want Neutral", got)
	}
}

func TestDefaultEscapeTarget_Mount(t *testing.T) {
	if got := position.DefaultEscapeTarget(position.Mount); got != position.HalfGuard {
		t.Errorf("DefaultEscapeTarget(Mount) = %v, want HalfGuard", got)
	}
}

func TestDefaultEscapeTarget_Guard(t *testing.T) {
	if got := position.DefaultEscapeTarget(position.Guard); got != position.Standing {
		t.Errorf("DefaultEscapeTarget(Guard) = %v, want Standing", got)
	}
}

// TestShiftControl_OneStepFromNeutralDoesNotReachControlled locks in
// the cap invariant relied on by Position_GrappleTick.processGrapplePair:
// after capping per-round delta at 1, the controlled side moves from
// Neutral to BecomingControlled in one round — NOT Controlled. This
// keeps the threshold-escape transition from firing on round 1 of a
// fresh Clinch grapple. See bug log 2026-05-16 (highwayman grapple
// breaking on round 1).
func TestShiftControl_OneStepFromNeutralDoesNotReachControlled(t *testing.T) {
	got := position.ShiftControl(position.Neutral, 1)
	if got == position.Controlled {
		t.Fatalf("ShiftControl(Neutral, 1) reached Controlled — gradient broken")
	}
	if got != position.BecomingControlled {
		t.Errorf("ShiftControl(Neutral, 1) = %v, want BecomingControlled", got)
	}
}

// TestShiftControl_OneStepFromNeutralTowardInControl is the symmetric
// invariant for the controller side: Neutral - 1 = LosingControl (one
// step toward InControl), not InControl directly.
func TestShiftControl_OneStepFromNeutralTowardInControl(t *testing.T) {
	got := position.ShiftControl(position.Neutral, -1)
	if got != position.LosingControl {
		t.Errorf("ShiftControl(Neutral, -1) = %v, want LosingControl", got)
	}
}
