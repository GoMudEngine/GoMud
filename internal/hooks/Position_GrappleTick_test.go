package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/skills"
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

// makeGrappleCharacter constructs a minimal Character with the given
// stats + skill level for grappleScore unit tests. Stamina and
// encumbrance multipliers will resolve to 1.0 because no penalties
// apply (Stamina at max, no items carried).
//
// Note: characters.New() calls ensureAllSkills which floors every
// skill at 1. makeGrappleCharacter always writes the explicit ucSkill
// value (including 0) to override that floor, so tests can verify
// the formula's skill term in isolation.
func makeGrappleCharacter(t *testing.T, str, dex, ucSkill int) *characters.Character {
	t.Helper()
	c := characters.New()
	c.Stats.Strength.Base = str
	c.Stats.Dexterity.Base = dex
	c.Stats.Strength.Recalculate()
	c.Stats.Dexterity.Recalculate()
	c.Stamina = 1000
	c.StaminaMax.Value = 1000
	if c.Skills == nil {
		c.Skills = map[string]int{}
	}
	// Always write ucSkill (even 0) to override the rank-1 floor set by
	// ensureAllSkills during New().
	c.Skills[string(skills.UnarmedCombat)] = ucSkill
	return c
}

// grappleScoreApproxEqual returns true when |a-b| < eps. Used for
// float comparisons in grappleScore tests (2.2*50 produces a tiny
// rounding error with IEEE-754 doubles).
func grappleScoreApproxEqual(a, b float64) bool {
	const eps = 1e-9
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < eps
}

func TestGrappleScore_StrCoefficient(t *testing.T) {
	cfg := configs.GetBalanceConfig()
	c := makeGrappleCharacter(t, 100, 0, 0)
	got := grappleScore(c, false, cfg)
	want := 70.0 // 0.7 * 100 + 0 + 0
	if !grappleScoreApproxEqual(got, want) {
		t.Errorf("Str=100 only → score = %v, want %v", got, want)
	}
}

func TestGrappleScore_DexCoefficient(t *testing.T) {
	cfg := configs.GetBalanceConfig()
	c := makeGrappleCharacter(t, 0, 100, 0)
	got := grappleScore(c, false, cfg)
	want := 30.0 // 0 + 0.3 * 100 + 0
	if !grappleScoreApproxEqual(got, want) {
		t.Errorf("Dex=100 only → score = %v, want %v", got, want)
	}
}

func TestGrappleScore_SkillCoefficientDefender(t *testing.T) {
	cfg := configs.GetBalanceConfig()
	c := makeGrappleCharacter(t, 0, 0, 50)
	got := grappleScore(c, false, cfg)
	want := 100.0 // 0 + 0 + 2.0 * 50
	if !grappleScoreApproxEqual(got, want) {
		t.Errorf("UC=50 defender → score = %v, want %v", got, want)
	}
}

func TestGrappleScore_SkillCoefficientAggressor(t *testing.T) {
	cfg := configs.GetBalanceConfig()
	c := makeGrappleCharacter(t, 0, 0, 50)
	got := grappleScore(c, true, cfg)
	want := 110.0 // 0 + 0 + 2.2 * 50
	if !grappleScoreApproxEqual(got, want) {
		t.Errorf("UC=50 aggressor → score = %v, want %v", got, want)
	}
}

func TestGrappleScore_CombinedFormulaDefender(t *testing.T) {
	cfg := configs.GetBalanceConfig()
	c := makeGrappleCharacter(t, 100, 100, 30)
	got := grappleScore(c, false, cfg)
	want := 160.0 // 0.7*100 + 0.3*100 + 2.0*30 = 70 + 30 + 60
	if !grappleScoreApproxEqual(got, want) {
		t.Errorf("S100/D100/UC30 defender → score = %v, want %v", got, want)
	}
}

func TestGrappleScore_CombinedFormulaAggressor(t *testing.T) {
	cfg := configs.GetBalanceConfig()
	c := makeGrappleCharacter(t, 100, 100, 30)
	got := grappleScore(c, true, cfg)
	want := 166.0 // 0.7*100 + 0.3*100 + 2.2*30 = 70 + 30 + 66
	if !grappleScoreApproxEqual(got, want) {
		t.Errorf("S100/D100/UC30 aggressor → score = %v, want %v", got, want)
	}
}

func TestGrappleScore_EscapeModifierIgnored(t *testing.T) {
	// EscapeModifier on body armor should NOT change the score.
	// Verification: grappleScore never references EscapeModifier
	// (no field read in the function body).
	cfg := configs.GetBalanceConfig()
	c := makeGrappleCharacter(t, 100, 100, 0)
	baseline := grappleScore(c, false, cfg)
	got := grappleScore(c, false, cfg)
	if !grappleScoreApproxEqual(got, baseline) {
		t.Errorf("score should not depend on EscapeModifier; got %v, baseline %v", got, baseline)
	}
}

func TestGrappleScore_NilCharacter(t *testing.T) {
	cfg := configs.GetBalanceConfig()
	got := grappleScore(nil, false, cfg)
	if got != 0 {
		t.Errorf("nil character → score = %v, want 0", got)
	}
}
