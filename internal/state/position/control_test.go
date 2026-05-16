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
