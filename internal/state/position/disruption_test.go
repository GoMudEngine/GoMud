package position

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/state/control"
)

func TestPositionDisruptionDmgEquiv_StandingReturnsZero(t *testing.T) {
	if got := PositionDisruptionDmgEquiv(Standing, control.Neutral); got != 0 {
		t.Errorf("Standing → %d, want 0", got)
	}
	if got := PositionDisruptionDmgEquiv(Standing, control.Controlling); got != 0 {
		t.Errorf("Standing controller → %d, want 0", got)
	}
	if got := PositionDisruptionDmgEquiv(Standing, control.Controlled); got != 0 {
		t.Errorf("Standing controlled → %d, want 0", got)
	}
}

func TestPositionDisruptionDmgEquiv_ProneSupine(t *testing.T) {
	if got := PositionDisruptionDmgEquiv(Prone, control.Neutral); got != 30 {
		t.Errorf("Prone → %d, want 30", got)
	}
	if got := PositionDisruptionDmgEquiv(Supine, control.Neutral); got != 25 {
		t.Errorf("Supine → %d, want 25", got)
	}
}

func TestPositionDisruptionDmgEquiv_Clinch(t *testing.T) {
	if got := PositionDisruptionDmgEquiv(Clinch, control.Controlling); got != 40 {
		t.Errorf("Clinch controller → %d, want 40", got)
	}
	if got := PositionDisruptionDmgEquiv(Clinch, control.Controlled); got != 40 {
		t.Errorf("Clinch controlled → %d, want 40", got)
	}
}

func TestPositionDisruptionDmgEquiv_BackStanding(t *testing.T) {
	if got := PositionDisruptionDmgEquiv(BackStanding, control.Controlling); got != 35 {
		t.Errorf("BackStanding controller → %d, want 35", got)
	}
	if got := PositionDisruptionDmgEquiv(BackStanding, control.Controlled); got != 50 {
		t.Errorf("BackStanding controlled → %d, want 50", got)
	}
}

func TestPositionDisruptionDmgEquiv_Mount(t *testing.T) {
	if got := PositionDisruptionDmgEquiv(Mount, control.Controlling); got != 35 {
		t.Errorf("Mount controller → %d, want 35", got)
	}
	if got := PositionDisruptionDmgEquiv(Mount, control.Controlled); got != 60 {
		t.Errorf("Mount controlled → %d, want 60", got)
	}
}

func TestPositionDisruptionDmgEquiv_SideControl(t *testing.T) {
	if got := PositionDisruptionDmgEquiv(SideControl, control.Controlling); got != 30 {
		t.Errorf("SideControl controller → %d, want 30", got)
	}
	if got := PositionDisruptionDmgEquiv(SideControl, control.Controlled); got != 55 {
		t.Errorf("SideControl controlled → %d, want 55", got)
	}
}

func TestPositionDisruptionDmgEquiv_KneeOnBelly(t *testing.T) {
	if got := PositionDisruptionDmgEquiv(KneeOnBelly, control.Controlling); got != 30 {
		t.Errorf("KneeOnBelly controller → %d, want 30", got)
	}
	if got := PositionDisruptionDmgEquiv(KneeOnBelly, control.Controlled); got != 50 {
		t.Errorf("KneeOnBelly controlled → %d, want 50", got)
	}
}

func TestPositionDisruptionDmgEquiv_NorthSouth(t *testing.T) {
	if got := PositionDisruptionDmgEquiv(NorthSouth, control.Controlling); got != 30 {
		t.Errorf("NorthSouth controller → %d, want 30", got)
	}
	if got := PositionDisruptionDmgEquiv(NorthSouth, control.Controlled); got != 45 {
		t.Errorf("NorthSouth controlled → %d, want 45", got)
	}
}

func TestPositionDisruptionDmgEquiv_Crucifix(t *testing.T) {
	if got := PositionDisruptionDmgEquiv(Crucifix, control.Controlling); got != 35 {
		t.Errorf("Crucifix controller → %d, want 35", got)
	}
	if got := PositionDisruptionDmgEquiv(Crucifix, control.Controlled); got != 70 {
		t.Errorf("Crucifix controlled → %d, want 70", got)
	}
}

func TestPositionDisruptionDmgEquiv_BackGround(t *testing.T) {
	if got := PositionDisruptionDmgEquiv(BackGround, control.Controlling); got != 35 {
		t.Errorf("BackGround controller → %d, want 35", got)
	}
	if got := PositionDisruptionDmgEquiv(BackGround, control.Controlled); got != 65 {
		t.Errorf("BackGround controlled → %d, want 65", got)
	}
}

func TestPositionDisruptionDmgEquiv_HalfGuard(t *testing.T) {
	if got := PositionDisruptionDmgEquiv(HalfGuard, control.Controlling); got != 30 {
		t.Errorf("HalfGuard controller → %d, want 30", got)
	}
	if got := PositionDisruptionDmgEquiv(HalfGuard, control.Controlled); got != 40 {
		t.Errorf("HalfGuard controlled → %d, want 40", got)
	}
}

// Guard is inverted: bottom (Controlling) has free hands, lower disruption
// than top (Controlled) who is stuck on top of someone's hips.
func TestPositionDisruptionDmgEquiv_GuardInverted(t *testing.T) {
	if got := PositionDisruptionDmgEquiv(Guard, control.Controlling); got != 25 {
		t.Errorf("Guard bottom (controller) → %d, want 25", got)
	}
	if got := PositionDisruptionDmgEquiv(Guard, control.Controlled); got != 40 {
		t.Errorf("Guard top (controlled) → %d, want 40", got)
	}
}

func TestPositionDisruptionDmgEquiv_Turtle(t *testing.T) {
	if got := PositionDisruptionDmgEquiv(Turtle, control.Controlling); got != 35 {
		t.Errorf("Turtle controller → %d, want 35", got)
	}
	if got := PositionDisruptionDmgEquiv(Turtle, control.Controlled); got != 45 {
		t.Errorf("Turtle controlled → %d, want 45", got)
	}
}

// Sanity invariant: for every NON-Guard ground grapple, controlled-role
// disruption is >= controller-role disruption (the bottom guy has it worse).
func TestPositionDisruptionDmgEquiv_ControlledWorseExceptGuard(t *testing.T) {
	positions := []State{Mount, SideControl, KneeOnBelly, NorthSouth,
		Crucifix, BackGround, HalfGuard, BackStanding, Turtle}
	for _, pos := range positions {
		ctrl := PositionDisruptionDmgEquiv(pos, control.Controlling)
		cd := PositionDisruptionDmgEquiv(pos, control.Controlled)
		if cd < ctrl {
			t.Errorf("position %v: controlled (%d) < controller (%d); expected controlled >= controller", pos, cd, ctrl)
		}
	}
}

// Guard inversion sanity: controller-role (bottom) < controlled-role (top).
func TestPositionDisruptionDmgEquiv_GuardControllerLowerThanControlled(t *testing.T) {
	ctrl := PositionDisruptionDmgEquiv(Guard, control.Controlling)
	cd := PositionDisruptionDmgEquiv(Guard, control.Controlled)
	if ctrl >= cd {
		t.Errorf("Guard inversion broken: controller %d >= controlled %d; expected controller < controlled", ctrl, cd)
	}
}
