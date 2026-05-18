package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// TestGrappleTick_ControlLevelShiftsOverTime verifies that per-round
// ticks produce ControlLevel drift. Needs test infrastructure to fire
// processGrappleTick directly without an event + a fresh Mount pair;
// covered by T28 smoke for now.
func TestGrappleTick_ControlLevelShiftsOverTime(t *testing.T) {
	t.Skip("integration test — depends on test infrastructure for direct tick call")
}

// TestGrappleTick_ThresholdTriggersEscape verifies that hitting
// Controlled on either side triggers a position transition to the
// default escape target. Needs dice RNG override + direct tick call.
// Covered by T28 smoke for now.
func TestGrappleTick_ThresholdTriggersEscape(t *testing.T) {
	t.Skip("integration test — needs dice RNG override; deferred to T28 smoke")
}

// TestProcessGrapplePair_StashesDriftSnapshot verifies that after a
// processGrapplePair call, both the controller and the controlled
// character have a DriftRollSnapshot populated with the current round
// number and non-zero z-scores.
func TestProcessGrapplePair_StashesDriftSnapshot(t *testing.T) {
	a := characters.New()
	a.SetUserId(1)
	b := characters.New()
	b.SetUserId(2)

	// Give both characters meaningful stats so the roll produces
	// a non-trivial result.
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

	// Walk Standing → Clinch → Mount (same path as ConsistencyCheck tests).
	if err := position.TransitionPair(a, b, position.Clinch,
		state.TransitionReason{Trigger: position.TriggerGrappleEntry}); err != nil {
		t.Fatalf("TransitionPair → Clinch failed: %v", err)
	}
	if err := position.TransitionPair(a, b, position.Mount,
		state.TransitionReason{Trigger: position.TriggerTakedownMount}); err != nil {
		t.Fatalf("TransitionPair → Mount failed: %v", err)
	}

	roundBefore := util.GetRoundCount()
	processGrapplePair(a, b)
	roundAfter := util.GetRoundCount()

	// Both snapshots should be stamped with a round in [roundBefore, roundAfter].
	if a.LastDriftRoll.Round < roundBefore || a.LastDriftRoll.Round > roundAfter {
		t.Errorf("a.LastDriftRoll.Round = %d, expected in [%d, %d]",
			a.LastDriftRoll.Round, roundBefore, roundAfter)
	}
	if b.LastDriftRoll.Round < roundBefore || b.LastDriftRoll.Round > roundAfter {
		t.Errorf("b.LastDriftRoll.Round = %d, expected in [%d, %d]",
			b.LastDriftRoll.Round, roundBefore, roundAfter)
	}

	// Both sides should see the same round.
	if a.LastDriftRoll.Round != b.LastDriftRoll.Round {
		t.Errorf("round mismatch: a=%d b=%d", a.LastDriftRoll.Round, b.LastDriftRoll.Round)
	}

	// Z-scores should be non-zero — dice.OpposedRollStat with stat=100
	// produces rolls far enough from zero that both z-scores will be set.
	if a.LastDriftRoll.AttackerZScore == 0 && a.LastDriftRoll.DefenderZScore == 0 {
		t.Error("both z-scores are zero — snapshot was not populated")
	}

	// Both snapshots should store the same margin (same roll, stored on both sides).
	if a.LastDriftRoll.MarginAttacker != b.LastDriftRoll.MarginAttacker {
		t.Errorf("margin mismatch: a=%v b=%v",
			a.LastDriftRoll.MarginAttacker, b.LastDriftRoll.MarginAttacker)
	}
}
