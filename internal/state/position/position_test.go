package position_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/position"
)

// --- PO-001 through PO-004: Default + nil-safety ---

func TestPO_001_NewMachineStartsStanding(t *testing.T) {
	m := position.NewMachine()
	if m.State() != position.Standing {
		t.Errorf("expected Standing, got %v", m.State())
	}
	if !m.IsStanding() {
		t.Errorf("IsStanding() = false, want true")
	}
	if m.IsGrappling() {
		t.Errorf("IsGrappling() = true, want false")
	}
}

func TestPO_002_StandingHasNoData(t *testing.T) {
	m := position.NewMachine()
	if _, ok := m.ProneData(); ok {
		t.Errorf("ProneData() should return ok=false in Standing")
	}
	if _, ok := m.SupineData(); ok {
		t.Errorf("SupineData() should return ok=false in Standing")
	}
	if _, ok := m.GrappleData(); ok {
		t.Errorf("GrappleData() should return ok=false in Standing")
	}
}

func TestPO_003_StateStringFormatted(t *testing.T) {
	// Sanity: each enum value has a non-empty, non-"Unknown" String().
	for s := position.Standing; s <= position.Turtle; s++ {
		got := s.String()
		if got == "" || got == "Unknown" {
			t.Errorf("State(%d).String() = %q, want non-empty + non-Unknown", s, got)
		}
	}
}

func TestPO_004_ControlLevelStringFormatted(t *testing.T) {
	for c := position.InControl; c <= position.Controlled; c++ {
		got := c.String()
		if got == "" || got == "Unknown" {
			t.Errorf("ControlLevel(%d).String() = %q, want non-empty + non-Unknown", c, got)
		}
	}
}

// --- PO-005 through PO-018: Basic transitions (14 representative samples) ---

func TestPO_005_StandingToProne(t *testing.T) {
	m := position.NewMachine()
	err := m.TransitionToProne(
		position.ProneData{MinRecoveryRounds: 2},
		state.TransitionReason{Trigger: position.TriggerKnockdownFaceForward},
	)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if m.State() != position.Prone {
		t.Errorf("expected Prone, got %v", m.State())
	}
}

func TestPO_006_StandingToSupine(t *testing.T) {
	m := position.NewMachine()
	err := m.TransitionToSupine(
		position.SupineData{MinRecoveryRounds: 2},
		state.TransitionReason{Trigger: position.TriggerKnockdownFaceBackward},
	)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if m.State() != position.Supine {
		t.Errorf("expected Supine, got %v", m.State())
	}
}

func TestPO_007_StandingToClinch(t *testing.T) {
	m := position.NewMachine()
	err := m.TransitionToClinch(
		position.GrappleData{Partner: state.ActorRef{UserId: 1}},
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if m.State() != position.Clinch {
		t.Errorf("expected Clinch, got %v", m.State())
	}
}

func TestPO_008_ProneToStanding(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToProne(position.ProneData{}, state.TransitionReason{Trigger: position.TriggerKnockdownFaceForward})
	err := m.TransitionToStanding(state.TransitionReason{Trigger: position.TriggerStandCommand})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if m.State() != position.Standing {
		t.Errorf("expected Standing, got %v", m.State())
	}
}

func TestPO_009_SupineToGuard(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToSupine(position.SupineData{}, state.TransitionReason{Trigger: position.TriggerKnockdownFaceBackward})
	err := m.TransitionToGuard(
		position.GrappleData{Partner: state.ActorRef{UserId: 2}},
		state.TransitionReason{Trigger: position.TriggerGuardPull},
	)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if m.State() != position.Guard {
		t.Errorf("expected Guard, got %v", m.State())
	}
}

func TestPO_010_ClinchToMount(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 3}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	err := m.TransitionToMount(
		position.GrappleData{Partner: state.ActorRef{UserId: 3}},
		state.TransitionReason{Trigger: position.TriggerTakedownMount},
	)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if m.State() != position.Mount {
		t.Errorf("expected Mount, got %v", m.State())
	}
}

func TestPO_011_MountToSideControl(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 4}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	_ = m.TransitionToMount(position.GrappleData{Partner: state.ActorRef{UserId: 4}}, state.TransitionReason{Trigger: position.TriggerTakedownMount})
	err := m.TransitionToSideControl(
		position.GrappleData{Partner: state.ActorRef{UserId: 4}},
		state.TransitionReason{Trigger: position.TriggerPositionAdvance},
	)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if m.State() != position.SideControl {
		t.Errorf("expected SideControl, got %v", m.State())
	}
}

func TestPO_012_SideControlToMount(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 5}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	_ = m.TransitionToSideControl(position.GrappleData{Partner: state.ActorRef{UserId: 5}}, state.TransitionReason{Trigger: position.TriggerTakedownSide})
	err := m.TransitionToMount(
		position.GrappleData{Partner: state.ActorRef{UserId: 5}},
		state.TransitionReason{Trigger: position.TriggerPositionAdvance},
	)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if m.State() != position.Mount {
		t.Errorf("expected Mount, got %v", m.State())
	}
}

func TestPO_013_MountToBackGround(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 6}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	_ = m.TransitionToMount(position.GrappleData{Partner: state.ActorRef{UserId: 6}}, state.TransitionReason{Trigger: position.TriggerTakedownMount})
	err := m.TransitionToBackGround(
		position.GrappleData{Partner: state.ActorRef{UserId: 6}},
		state.TransitionReason{Trigger: position.TriggerBackTakeGround},
	)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if m.State() != position.BackGround {
		t.Errorf("expected BackGround, got %v", m.State())
	}
}

func TestPO_014_BackGroundToStanding(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 7}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	_ = m.TransitionToBackGround(position.GrappleData{Partner: state.ActorRef{UserId: 7}}, state.TransitionReason{Trigger: position.TriggerTakedownBackGround})
	err := m.TransitionToStanding(state.TransitionReason{Trigger: position.TriggerPositionEscape})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if m.State() != position.Standing {
		t.Errorf("expected Standing, got %v", m.State())
	}
}

func TestPO_015_GuardToStanding(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToSupine(position.SupineData{}, state.TransitionReason{Trigger: position.TriggerKnockdownFaceBackward})
	_ = m.TransitionToGuard(position.GrappleData{Partner: state.ActorRef{UserId: 8}}, state.TransitionReason{Trigger: position.TriggerGuardPull})
	err := m.TransitionToStanding(state.TransitionReason{Trigger: position.TriggerGrappleBreak})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestPO_016_TurtleAllowsZeroPartner(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 9}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	_ = m.TransitionToSideControl(position.GrappleData{Partner: state.ActorRef{UserId: 9}}, state.TransitionReason{Trigger: position.TriggerTakedownSide})
	err := m.TransitionToTurtle(
		position.GrappleData{Partner: state.ActorRef{}}, // zero — solo defensive curl
		state.TransitionReason{Trigger: position.TriggerTurtleDefend},
	)
	if err != nil {
		t.Fatalf("Turtle should allow zero Partner; got %v", err)
	}
}

func TestPO_017_CrucifixViaSideControl(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 10}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	_ = m.TransitionToSideControl(position.GrappleData{Partner: state.ActorRef{UserId: 10}}, state.TransitionReason{Trigger: position.TriggerTakedownSide})
	err := m.TransitionToCrucifix(
		position.GrappleData{Partner: state.ActorRef{UserId: 10}},
		state.TransitionReason{Trigger: position.TriggerArmIsolation},
	)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if m.State() != position.Crucifix {
		t.Errorf("expected Crucifix, got %v", m.State())
	}
}

func TestPO_018_BackStandingViaClinch(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 11}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	err := m.TransitionToBackStanding(
		position.GrappleData{Partner: state.ActorRef{UserId: 11}},
		state.TransitionReason{Trigger: position.TriggerBackTakeStanding},
	)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if m.State() != position.BackStanding {
		t.Errorf("expected BackStanding, got %v", m.State())
	}
}

// --- PO-019 through PO-024: Invalid-transition rejection ---

func TestPO_019_StandingToMountFails(t *testing.T) {
	m := position.NewMachine()
	err := m.TransitionToMount(
		position.GrappleData{Partner: state.ActorRef{UserId: 12}},
		state.TransitionReason{Trigger: position.TriggerTakedownMount},
	)
	if err == nil {
		t.Fatal("Standing → Mount should fail (must go via Clinch or Prone/Supine)")
	}
	if m.State() != position.Standing {
		t.Errorf("state should remain Standing on failed transition, got %v", m.State())
	}
}

func TestPO_020_StandingToBackStandingFails(t *testing.T) {
	m := position.NewMachine()
	err := m.TransitionToBackStanding(
		position.GrappleData{Partner: state.ActorRef{UserId: 13}},
		state.TransitionReason{Trigger: position.TriggerBackTakeStanding},
	)
	if err == nil {
		t.Fatal("Standing → BackStanding should fail (must go via Clinch)")
	}
}

func TestPO_021_ClinchToKneeOnBellyFails(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 14}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	err := m.TransitionToKneeOnBelly(
		position.GrappleData{Partner: state.ActorRef{UserId: 14}},
		state.TransitionReason{Trigger: position.TriggerPositionAdvance},
	)
	if err == nil {
		t.Fatal("Clinch → KneeOnBelly should fail (KOB requires ground first; go via SC)")
	}
}

func TestPO_022_SupineToBackGroundFails(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToSupine(position.SupineData{}, state.TransitionReason{Trigger: position.TriggerKnockdownFaceBackward})
	err := m.TransitionToBackGround(
		position.GrappleData{Partner: state.ActorRef{UserId: 15}},
		state.TransitionReason{Trigger: position.TriggerBackTakeGround},
	)
	if err == nil {
		t.Fatal("Supine → BackGround should fail (attacker would need to flip target first)")
	}
}

func TestPO_023_MountToProneFails(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 16}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	_ = m.TransitionToMount(position.GrappleData{Partner: state.ActorRef{UserId: 16}}, state.TransitionReason{Trigger: position.TriggerTakedownMount})
	err := m.TransitionToProne(
		position.ProneData{},
		state.TransitionReason{Trigger: position.TriggerKnockdownFaceForward},
	)
	if err == nil {
		t.Fatal("Mount → Prone should fail (controller can't drop into a non-grapple knockdown directly)")
	}
}

func TestPO_024_GrappleRequiresNonZeroPartner(t *testing.T) {
	m := position.NewMachine()
	err := m.TransitionToClinch(
		position.GrappleData{Partner: state.ActorRef{}}, // zero Partner
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	)
	if err == nil {
		t.Fatal("Clinch should reject zero Partner (only Turtle allows it)")
	}
}

// --- PO-025 through PO-028: GrappleData carries data ---

func TestPO_025_ClinchDataPreserved(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToClinch(
		position.GrappleData{Partner: state.ActorRef{UserId: 17}, ControlLevel: position.LosingControl},
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	)
	d, ok := m.GrappleData()
	if !ok {
		t.Fatal("expected GrappleData to be available")
	}
	if d.Partner.UserId != 17 {
		t.Errorf("Partner.UserId = %d, want 17", d.Partner.UserId)
	}
	if d.ControlLevel != position.LosingControl {
		t.Errorf("ControlLevel = %v, want LosingControl", d.ControlLevel)
	}
}

func TestPO_026_GrappleDataDefaultsNeutral(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToClinch(
		position.GrappleData{Partner: state.ActorRef{UserId: 18}}, // ControlLevel omitted
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	)
	d, _ := m.GrappleData()
	if d.ControlLevel != position.Neutral {
		t.Errorf("default ControlLevel = %v, want Neutral", d.ControlLevel)
	}
}

func TestPO_027_ProneDataPreserved(t *testing.T) {
	m := position.NewMachine()
	src := state.ActorRef{UserId: 19}
	_ = m.TransitionToProne(
		position.ProneData{MinRecoveryRounds: 3, KnockdownSource: src},
		state.TransitionReason{Trigger: position.TriggerKnockdownFaceForward},
	)
	d, ok := m.ProneData()
	if !ok {
		t.Fatal("expected ProneData to be available")
	}
	if d.MinRecoveryRounds != 3 {
		t.Errorf("MinRecoveryRounds = %d, want 3", d.MinRecoveryRounds)
	}
	if d.KnockdownSource.UserId != 19 {
		t.Errorf("KnockdownSource.UserId = %d, want 19", d.KnockdownSource.UserId)
	}
}

func TestPO_028_DataClearedOnReturnToStanding(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 20}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	_ = m.TransitionToStanding(state.TransitionReason{Trigger: position.TriggerGrappleBreak})
	if _, ok := m.GrappleData(); ok {
		t.Errorf("GrappleData() should return ok=false after returning to Standing")
	}
}

// --- PO-029 through PO-036: Predicate correctness (Machine-level) ---

func TestPO_029_IsGrapplingMatchesGrappleStates(t *testing.T) {
	cases := []struct {
		name  string
		setup func(*position.Machine)
		want  bool
	}{
		{"Standing", func(m *position.Machine) {}, false},
		{"Prone", func(m *position.Machine) {
			_ = m.TransitionToProne(position.ProneData{}, state.TransitionReason{Trigger: position.TriggerKnockdownFaceForward})
		}, false},
		{"Supine", func(m *position.Machine) {
			_ = m.TransitionToSupine(position.SupineData{}, state.TransitionReason{Trigger: position.TriggerKnockdownFaceBackward})
		}, false},
		{"Clinch", func(m *position.Machine) {
			_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
		}, true},
		{"Mount", func(m *position.Machine) {
			_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
			_ = m.TransitionToMount(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerTakedownMount})
		}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := position.NewMachine()
			tc.setup(m)
			if got := m.IsGrappling(); got != tc.want {
				t.Errorf("IsGrappling() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPO_030_IsTopDominantMatchesTopStates(t *testing.T) {
	tops := []struct {
		name  string
		setup func(*position.Machine)
	}{
		{"Mount", func(m *position.Machine) {
			_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
			_ = m.TransitionToMount(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerTakedownMount})
		}},
		{"SideControl", func(m *position.Machine) {
			_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
			_ = m.TransitionToSideControl(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerTakedownSide})
		}},
		{"BackGround", func(m *position.Machine) {
			_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
			_ = m.TransitionToBackGround(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerTakedownBackGround})
		}},
	}
	for _, tc := range tops {
		t.Run(tc.name, func(t *testing.T) {
			m := position.NewMachine()
			tc.setup(m)
			if !m.IsTopDominant() {
				t.Errorf("IsTopDominant() = false for %s, want true", tc.name)
			}
		})
	}
}

func TestPO_031_IsGuardNotTopDominant(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToSupine(position.SupineData{}, state.TransitionReason{Trigger: position.TriggerKnockdownFaceBackward})
	_ = m.TransitionToGuard(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerGuardPull})
	if m.IsTopDominant() {
		t.Errorf("Guard should NOT be IsTopDominant (it's bottom-active)")
	}
	if !m.IsGrappling() || !m.IsGroundGrapple() {
		t.Errorf("Guard should be IsGrappling AND IsGroundGrapple")
	}
}

func TestPO_032_IsStandingGrappleMatchesClinchAndBackStanding(t *testing.T) {
	cases := []struct {
		name  string
		setup func(*position.Machine)
		want  bool
	}{
		{"Clinch", func(m *position.Machine) {
			_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
		}, true},
		{"BackStanding", func(m *position.Machine) {
			_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
			_ = m.TransitionToBackStanding(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerBackTakeStanding})
		}, true},
		{"Mount", func(m *position.Machine) {
			_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
			_ = m.TransitionToMount(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerTakedownMount})
		}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := position.NewMachine()
			tc.setup(m)
			if got := m.IsStandingGrapple(); got != tc.want {
				t.Errorf("IsStandingGrapple() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPO_033_IsOnFloorMatchesProneSupineAndGroundGrapples(t *testing.T) {
	cases := []struct {
		name  string
		setup func(*position.Machine)
		want  bool
	}{
		{"Standing", func(m *position.Machine) {}, false},
		{"Prone", func(m *position.Machine) {
			_ = m.TransitionToProne(position.ProneData{}, state.TransitionReason{Trigger: position.TriggerKnockdownFaceForward})
		}, true},
		{"Supine", func(m *position.Machine) {
			_ = m.TransitionToSupine(position.SupineData{}, state.TransitionReason{Trigger: position.TriggerKnockdownFaceBackward})
		}, true},
		{"Clinch", func(m *position.Machine) {
			_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
		}, false},
		{"Mount", func(m *position.Machine) {
			_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
			_ = m.TransitionToMount(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerTakedownMount})
		}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := position.NewMachine()
			tc.setup(m)
			if got := m.IsOnFloor(); got != tc.want {
				t.Errorf("IsOnFloor() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPO_034_ProneIsNotSupine(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToProne(position.ProneData{}, state.TransitionReason{Trigger: position.TriggerKnockdownFaceForward})
	if !m.IsProne() {
		t.Errorf("IsProne() should be true")
	}
	if m.IsSupine() {
		t.Errorf("IsSupine() should be false when in Prone")
	}
}

func TestPO_035_SupineIsNotProne(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToSupine(position.SupineData{}, state.TransitionReason{Trigger: position.TriggerKnockdownFaceBackward})
	if !m.IsSupine() {
		t.Errorf("IsSupine() should be true")
	}
	if m.IsProne() {
		t.Errorf("IsProne() should be false when in Supine")
	}
}

func TestPO_036_RegistryRoundTrip(t *testing.T) {
	m := position.NewMachine()
	ref := state.ActorRef{UserId: 42}
	position.RegisterMachine(ref, m)
	defer position.UnregisterMachine(ref)
	if m.Self() != ref {
		t.Errorf("Self() = %v, want %v", m.Self(), ref)
	}
}

// --- PO-037 through PO-040: Cascade verification (integration — Task 5) ---

func TestPO_037_LifeDeadCascadesPositionStanding(t *testing.T) {
	t.Skip("integration test — verified in Task 5 cascade tests (internal/hooks/Position_Cascades_test.go)")
}

func TestPO_038_CascadeFiresFromMount(t *testing.T) {
	t.Skip("integration test — verified in Task 5 cascade tests")
}

func TestPO_039_CascadeFiresFromGuard(t *testing.T) {
	t.Skip("integration test — verified in Task 5 cascade tests")
}

func TestPO_040_CascadeFiresFromBackGround(t *testing.T) {
	t.Skip("integration test — verified in Task 5 cascade tests")
}

// --- PO-041 through PO-043: Btree primitive smoke (integration — Task 6) ---

func TestPO_041_BtreeMobInMountFires(t *testing.T) {
	t.Skip("integration test — verified in Task 6 btree primitive tests (internal/behaviortree/conditions_position_test.go)")
}

func TestPO_042_BtreeTargetIsGrappledFires(t *testing.T) {
	t.Skip("integration test — verified in Task 6 btree primitive tests")
}

func TestPO_043_BtreeMobIsProneFires(t *testing.T) {
	t.Skip("integration test — verified in Task 6 btree primitive tests")
}

// --- PO-044, PO-045: Turtle Partner edge case ---

func TestPO_044_TurtleZeroPartnerAccepted(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	_ = m.TransitionToSideControl(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerTakedownSide})
	err := m.TransitionToTurtle(
		position.GrappleData{Partner: state.ActorRef{}},
		state.TransitionReason{Trigger: position.TriggerTurtleDefend},
	)
	if err != nil {
		t.Errorf("TransitionToTurtle should accept zero Partner; got %v", err)
	}
}

func TestPO_045_MountZeroPartnerRejected(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	err := m.TransitionToMount(
		position.GrappleData{Partner: state.ActorRef{}}, // zero
		state.TransitionReason{Trigger: position.TriggerTakedownMount},
	)
	if err == nil {
		t.Errorf("TransitionToMount should reject zero Partner")
	}
}
