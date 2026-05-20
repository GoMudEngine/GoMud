package position

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/state/control"
)

func TestTargetSideHitModifier_Standing(t *testing.T) {
	got := TargetSideHitModifier(Standing, control.Neutral)
	if got != 1.0 {
		t.Errorf("Standing → %v, want 1.0", got)
	}
}

func TestTargetSideHitModifier_Mount(t *testing.T) {
	if got := TargetSideHitModifier(Mount, control.Controlling); got != 1.12 {
		t.Errorf("Mount controller → %v, want 1.12", got)
	}
	if got := TargetSideHitModifier(Mount, control.Controlled); got != 1.20 {
		t.Errorf("Mount controlled → %v, want 1.20", got)
	}
}

func TestTargetSideHitModifier_Crucifix(t *testing.T) {
	if got := TargetSideHitModifier(Crucifix, control.Controlled); got != 1.25 {
		t.Errorf("Crucifix controlled → %v, want 1.25", got)
	}
}

func TestTargetSideHitModifier_GuardInverted(t *testing.T) {
	if got := TargetSideHitModifier(Guard, control.Controlling); got != 1.08 {
		t.Errorf("Guard controller (bottom) → %v, want 1.08", got)
	}
	if got := TargetSideHitModifier(Guard, control.Controlled); got != 1.10 {
		t.Errorf("Guard controlled (top) → %v, want 1.10", got)
	}
}

func TestAttackerSelfHitModifier_Standing(t *testing.T) {
	got := AttackerSelfHitModifier(Standing, control.Neutral)
	if got != 1.0 {
		t.Errorf("Standing → %v, want 1.0", got)
	}
}

func TestAttackerSelfHitModifier_MountControlling(t *testing.T) {
	got := AttackerSelfHitModifier(Mount, control.Controlling)
	if got != 1.10 {
		t.Errorf("Mount controller (apex) → %v, want 1.10", got)
	}
}

func TestAttackerSelfHitModifier_MountControlled(t *testing.T) {
	got := AttackerSelfHitModifier(Mount, control.Controlled)
	if got != 0.70 {
		t.Errorf("Mount controlled (under) → %v, want 0.70", got)
	}
}

func TestAttackerSelfHitModifier_Crucifix(t *testing.T) {
	if got := AttackerSelfHitModifier(Crucifix, control.Controlled); got != 0.50 {
		t.Errorf("Crucifix controlled (arm trapped) → %v, want 0.50", got)
	}
}

func TestComposition_MountControllerSwingingAtControlled(t *testing.T) {
	atkSelf := AttackerSelfHitModifier(Mount, control.Controlling)
	tgtSide := TargetSideHitModifier(Mount, control.Controlled)
	net := atkSelf * tgtSide
	want := 1.32
	const epsilon = 1e-9
	if diff := net - want; diff < -epsilon || diff > epsilon {
		t.Errorf("Mount controller vs controlled net = %v, want %v", net, want)
	}
}

func TestComposition_ThirdPartyAttackingMountedDefender(t *testing.T) {
	atkSelf := AttackerSelfHitModifier(Standing, control.Neutral)
	tgtSide := TargetSideHitModifier(Mount, control.Controlled)
	net := atkSelf * tgtSide
	want := 1.20
	const epsilon = 1e-9
	if diff := net - want; diff < -epsilon || diff > epsilon {
		t.Errorf("Third party vs mounted defender net = %v, want %v", net, want)
	}
}

func TestComposition_MountControlledSwingingBack(t *testing.T) {
	atkSelf := AttackerSelfHitModifier(Mount, control.Controlled)
	tgtSide := TargetSideHitModifier(Mount, control.Controlling)
	net := atkSelf * tgtSide
	want := 0.70 * 1.12
	const epsilon = 1e-9
	if diff := net - want; diff < -epsilon || diff > epsilon {
		t.Errorf("Mount-controlled swinging back net = %v, want %v", net, want)
	}
}
