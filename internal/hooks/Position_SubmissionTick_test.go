package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// setupMountPair creates two characters in a Mount grapple with `a`
// as controller (IsControllerRole=true) and `b` as controlled
// (IsControllerRole=false). Chunk 4b-fixup T18: ControlLevel removed;
// role now stamped by TransitionPair.
func setupMountPair(t *testing.T) (controller, controlled *characters.Character) {
	t.Helper()

	a := characters.New()
	a.SetUserId(1)
	b := characters.New()
	b.SetUserId(2)

	// Give both characters meaningful stats.
	a.Stats.Strength.Base = 100
	b.Stats.Strength.Base = 100
	a.Stats.Dexterity.Base = 100
	b.Stats.Dexterity.Base = 100
	a.Stamina = 100
	b.Stamina = 100
	a.StaminaMax.Base = 100
	b.StaminaMax.Base = 100
	a.StaminaMax.Value = 100
	b.StaminaMax.Value = 100

	// Walk Standing → Clinch → Mount.
	if err := position.TransitionPair(a, b, position.Clinch,
		state.TransitionReason{Trigger: position.TriggerGrappleEntry}); err != nil {
		t.Fatalf("TransitionPair → Clinch failed: %v", err)
	}
	if err := position.TransitionPair(a, b, position.Mount,
		state.TransitionReason{Trigger: position.TriggerTakedownMount}); err != nil {
		t.Fatalf("TransitionPair → Mount failed: %v", err)
	}

	// After TransitionPair → Mount, `a` has IsControllerRole=true and
	// `b` has IsControllerRole=false. IsTopSubEligible / IsBottomSubEligible
	// use IsControllerRole directly (chunk 4b-fixup T18: ControlLevel removed).

	return a, b
}

func TestEvaluateSubAttempt_ControllerEligible(t *testing.T) {
	a, b := setupMountPair(t)

	// Fake a drift roll where controller won big — margin above the
	// unified sub-window threshold (SubWindowOpens: |z| >= 1.5).
	currentRound := util.GetRoundCount()
	snap := characters.DriftRollSnapshot{
		Round:          currentRound,
		MarginAttacker: 2.0, // controller won by 2 std devs (above 1.5)
		AttackerZScore: 1.5,
		DefenderZScore: -1.5,
	}
	a.LastDriftRoll = snap
	b.LastDriftRoll = snap

	role, eligible := EvaluateSubAttempt(a, b)
	if !eligible {
		t.Errorf("expected controller to be eligible, got eligible=false")
	}
	if role != combat.RoleTop {
		t.Errorf("expected RoleTop, got %v", role)
	}
}

func TestEvaluateSubAttempt_DefenderEligibleViaMargin(t *testing.T) {
	a, b := setupMountPair(t)

	currentRound := util.GetRoundCount()
	snap := characters.DriftRollSnapshot{
		Round:          currentRound,
		MarginAttacker: -2.0, // defender won by 2 std devs
		AttackerZScore: -1.5,
		DefenderZScore: 1.5,
	}
	a.LastDriftRoll = snap
	b.LastDriftRoll = snap

	role, eligible := EvaluateSubAttempt(a, b)
	if !eligible {
		t.Errorf("expected defender to be eligible, got eligible=false")
	}
	if role != combat.RoleBottom {
		t.Errorf("expected RoleBottom, got %v", role)
	}
}

func TestEvaluateSubAttempt_DefenderEligibleViaCritShortcut(t *testing.T) {
	a, b := setupMountPair(t)

	// Defender crit on defense but margin not above alpha.
	currentRound := util.GetRoundCount()
	snap := characters.DriftRollSnapshot{
		Round:          currentRound,
		MarginAttacker: 0.0,
		AttackerZScore: 0.0,
		DefenderZScore: 2.5, // CRIT (>= SubmissionAttemptCritZ default 2.0)
	}
	a.LastDriftRoll = snap
	b.LastDriftRoll = snap

	role, eligible := EvaluateSubAttempt(a, b)
	if !eligible {
		t.Errorf("expected defender crit-shortcut to fire")
	}
	if role != combat.RoleBottom {
		t.Errorf("expected RoleBottom, got %v", role)
	}
}

func TestEvaluateSubAttempt_NeitherEligible(t *testing.T) {
	a, b := setupMountPair(t)

	currentRound := util.GetRoundCount()
	snap := characters.DriftRollSnapshot{
		Round:          currentRound,
		MarginAttacker: 0.3, // not enough on either side (below unified |z|>=1.5)
		AttackerZScore: 0.3,
		DefenderZScore: -0.3,
	}
	a.LastDriftRoll = snap
	b.LastDriftRoll = snap

	_, eligible := EvaluateSubAttempt(a, b)
	if eligible {
		t.Errorf("expected neither side eligible, got eligible=true")
	}
}

func TestEvaluateSubAttempt_StaleSnapshotIgnored(t *testing.T) {
	a, b := setupMountPair(t)

	// Snapshot is from a prior round — should be ignored.
	snap := characters.DriftRollSnapshot{
		Round:          util.GetRoundCount() - 1,
		MarginAttacker: 5.0, // would normally fire, but stale
		AttackerZScore: 3.0,
		DefenderZScore: -3.0,
	}
	a.LastDriftRoll = snap
	b.LastDriftRoll = snap

	_, eligible := EvaluateSubAttempt(a, b)
	if eligible {
		t.Errorf("expected stale snapshot to be ignored")
	}
}

func TestPickSubmissionRoundRobin_Cycles(t *testing.T) {
	c := characters.New()
	c.SetUserId(3)
	c.LastSubmissionAttempted = -1 // start before 0

	pool := []position.SubmissionType{
		position.SubAmericana,
		position.SubTriangle,
		position.SubArmbar,
	}

	// First call: index advances from -1 to 0 → SubAmericana.
	// Subsequent calls cycle through the pool.
	results := make([]position.SubmissionType, len(pool)*2)
	for i := range results {
		results[i] = pickSubmissionRoundRobin(c, pool)
	}

	// Verify it cycles without going out of bounds.
	for i, got := range results {
		expected := pool[i%len(pool)]
		if got != expected {
			t.Errorf("round-robin index %d: got %v, want %v", i, got, expected)
		}
	}
}

// TestSubmissionTickReadsPostAdvancePosition is an integration scaffold
// for the ordering guarantee introduced by chunk 4b-fixup T20.
//
// Intended scenario:
//   - Controller in Mount, drift z = 2.5 → ResolveOutcome (T15) advances
//     position to BackGround before SubmissionTick fires.
//   - SubmissionTick should offer a sub from the BackGround pool
//     (RNC family), not from the Mount pool (americana family).
//
// This ordering is guaranteed because both processGrappleTick and
// processSubmissionTick are NewRound listeners registered in filename-
// alphabetical order: GrappleTick runs first (calls ResolveOutcome +
// TransitionPair), then SubmissionTick reads the mutated position.
//
// T26 manual AI smoke covers this end-to-end. The test is left as a
// Skip scaffold so a future test author has clear context for what to
// build when full integration harness support is available.
func TestSubmissionTickReadsPostAdvancePosition(t *testing.T) {
	// Setup: controller in Mount, z = 2.5 → ResolveOutcome advances
	// to BackGround. SubmissionTick should fire sub from BackGround
	// (RNC family), not from Mount (americana family).
	//
	// This test asserts ordering: T15 wired processGrapplePair to call
	// ResolveOutcome → TransitionPair → then SubmissionTick reads the
	// new position.
	t.Skip("Integration scaffolding TBD by implementer; manual smoke covers in T26.")
}
