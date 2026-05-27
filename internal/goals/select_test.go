package goals

import (
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// Test plumbing: a no-op mob pointer is fine for cases that don't
// register a ContextScore. Select must tolerate nil-ish mob input
// when no registered type needs it.
var testMob = &mobs.Mob{}

// helper: build a goal with sensible defaults
func gFixture(id, gtype string, prio int) *Goal {
	return &Goal{Id: id, Type: gtype, Priority: prio, CreatedAt: time.Now().UTC()}
}

func TestSelect_EmptyGoalList_NoPrev_ReturnsNoGoals(t *testing.T) {
	got, switched, reason := Select(nil, nil, testMob, nil, 0, 0, 1000)
	if got != nil {
		t.Errorf("got=%v, want nil", got)
	}
	if switched {
		t.Errorf("switched=true, want false")
	}
	if reason.Kind != "no_goals" {
		t.Errorf("reason.Kind=%q, want %q", reason.Kind, "no_goals")
	}
}

func TestSelect_SingleGoal_NoPrev_Switches(t *testing.T) {
	g := gFixture("g1", "wealth", 30)
	got, switched, reason := Select([]*Goal{g}, nil, testMob, nil, 0, 0, 1000)
	if got != g {
		t.Errorf("got=%v, want g1", got)
	}
	if !switched {
		t.Errorf("switched=false, want true")
	}
	if reason.Kind != "switched" {
		t.Errorf("reason.Kind=%q, want %q", reason.Kind, "switched")
	}
}

func TestSelect_SingleGoal_SameAsPrev_KeptTopUnchanged(t *testing.T) {
	g := gFixture("g1", "wealth", 30)
	got, switched, reason := Select([]*Goal{g}, nil, testMob, g, 500, 500, 1000)
	if got != g {
		t.Errorf("got=%v, want g1", got)
	}
	if switched {
		t.Errorf("switched=true, want false")
	}
	if reason.Kind != "kept_top_unchanged" {
		t.Errorf("reason.Kind=%q, want %q", reason.Kind, "kept_top_unchanged")
	}
}

func TestSelect_HysteresisMargin_Blocks(t *testing.T) {
	// g2 outscores g1 by only 2 — below default margin 5.0 — should keep g1.
	prev := gFixture("g1", "wealth", 30)
	chal := gFixture("g2", "debt", 32)
	got, switched, reason := Select([]*Goal{prev, chal}, nil, testMob, prev, 500, 500, 1000)
	if got != prev {
		t.Errorf("got=%v, want g1 (prev)", got)
	}
	if switched {
		t.Errorf("switched=true, want false")
	}
	if reason.Kind != "kept_hysteresis_margin" {
		t.Errorf("reason.Kind=%q, want %q", reason.Kind, "kept_hysteresis_margin")
	}
}

func TestSelect_HysteresisMinHold_Blocks(t *testing.T) {
	// g2 outscores g1 by 10 (above margin) but g1 held only 10 rounds.
	prev := gFixture("g1", "wealth", 30)
	chal := gFixture("g2", "revenge", 40)
	got, switched, reason := Select([]*Goal{prev, chal}, nil, testMob, prev, 990, 990, 1000)
	if got != prev {
		t.Errorf("got=%v, want g1 (prev)", got)
	}
	if switched {
		t.Errorf("switched=true, want false")
	}
	if reason.Kind != "kept_hysteresis_min_hold" {
		t.Errorf("reason.Kind=%q, want %q", reason.Kind, "kept_hysteresis_min_hold")
	}
}

func TestSelect_BothGatesPass_Switches(t *testing.T) {
	prev := gFixture("g1", "wealth", 30)
	chal := gFixture("g2", "revenge", 60)
	// held 500 rounds (>= 100 min-hold default), gap 30 (>= 5.0 margin default)
	got, switched, reason := Select([]*Goal{prev, chal}, nil, testMob, prev, 500, 500, 1000)
	if got != chal {
		t.Errorf("got=%v, want g2 (challenger)", got)
	}
	if !switched {
		t.Errorf("switched=false, want true")
	}
	if reason.Kind != "switched" {
		t.Errorf("reason.Kind=%q, want %q", reason.Kind, "switched")
	}
}

func TestSelect_PrevRemoved_SwitchesPrevInvalid(t *testing.T) {
	prev := gFixture("g1", "wealth", 30)
	current := gFixture("g2", "revenge", 60)
	// prev not in the goal list (was removed)
	got, switched, reason := Select([]*Goal{current}, nil, testMob, prev, 500, 500, 1000)
	if got != current {
		t.Errorf("got=%v, want g2", got)
	}
	if !switched {
		t.Errorf("switched=false, want true")
	}
	if reason.Kind != "switched_prev_invalid" {
		t.Errorf("reason.Kind=%q, want %q", reason.Kind, "switched_prev_invalid")
	}
}

func TestSelect_PrevExpired_SwitchesPrevInvalid(t *testing.T) {
	prev := gFixture("g1", "wealth", 30)
	prev.ExpiresAt = time.Now().Add(-time.Hour).UTC()
	chal := gFixture("g2", "revenge", 60)
	got, switched, reason := Select([]*Goal{prev, chal}, nil, testMob, prev, 500, 500, 1000)
	if got != chal {
		t.Errorf("got=%v, want g2", got)
	}
	if !switched {
		t.Errorf("switched=false, want true")
	}
	if reason.Kind != "switched_prev_invalid" {
		t.Errorf("reason.Kind=%q, want %q", reason.Kind, "switched_prev_invalid")
	}
}

func TestSelect_AllFilteredPrevValid_KeptNoCandidates(t *testing.T) {
	// register a type whose ContextScore always returns 0
	resetRegistry()
	RegisterGoalType("contextzero", GoalTypeMeta{
		ContextScore: func(g *Goal, m *mobs.Mob) float64 { return 0 },
	})
	defer resetRegistry()
	// prev is "contextzero" (filtered) AND in the list (validity check).
	prev := gFixture("g1", "contextzero", 30)
	other := gFixture("g2", "contextzero", 90)
	got, switched, reason := Select([]*Goal{prev, other}, nil, testMob, prev, 500, 500, 1000)
	if got != prev {
		t.Errorf("got=%v, want prev (filtered candidates, prev valid)", got)
	}
	if switched {
		t.Errorf("switched=true, want false")
	}
	if reason.Kind != "kept_no_candidates" {
		t.Errorf("reason.Kind=%q, want %q", reason.Kind, "kept_no_candidates")
	}
}

func TestSelect_ArchetypeWeight_ElevatesLowerPriority(t *testing.T) {
	prev := gFixture("g1", "wealth", 50)
	chal := gFixture("g2", "revenge", 30)
	// archetype weights: revenge × 2.0 → effective 60; wealth × 1.0 → 50
	weights := map[string]float64{"revenge": 2.0}
	got, switched, _ := Select([]*Goal{prev, chal}, weights, testMob, prev, 500, 500, 1000)
	if got != chal {
		t.Errorf("got=%v, want g2 (revenge elevated by weight)", got)
	}
	if !switched {
		t.Errorf("switched=false, want true (gap=10 ≥ margin=5)")
	}
}

func TestSelect_ContextScoreZero_FiltersGoal(t *testing.T) {
	resetRegistry()
	RegisterGoalType("zerocontext", GoalTypeMeta{
		ContextScore: func(g *Goal, m *mobs.Mob) float64 { return 0 },
	})
	defer resetRegistry()
	candidate := gFixture("g1", "wealth", 30)
	filtered := gFixture("g2", "zerocontext", 90) // would normally win on priority
	got, switched, _ := Select([]*Goal{candidate, filtered}, nil, testMob, nil, 0, 0, 1000)
	if got != candidate {
		t.Errorf("got=%v, want g1 (zerocontext filtered)", got)
	}
	if !switched {
		t.Errorf("switched=false, want true")
	}
}

func TestSelect_ContextScoreElevates(t *testing.T) {
	resetRegistry()
	RegisterGoalType("emergency", GoalTypeMeta{
		ContextScore: func(g *Goal, m *mobs.Mob) float64 { return 3.0 },
	})
	defer resetRegistry()
	prev := gFixture("g1", "wealth", 50)
	chal := gFixture("g2", "emergency", 30) // 30 * 3.0 = 90 effective
	got, switched, _ := Select([]*Goal{prev, chal}, nil, testMob, prev, 500, 500, 1000)
	if got != chal {
		t.Errorf("got=%v, want g2 (emergency elevated by contextscore)", got)
	}
	if !switched {
		t.Errorf("switched=false, want true (gap=40 ≥ margin=5)")
	}
}

func TestSelect_TieBreak_PriorityDescThenIdAsc(t *testing.T) {
	// two goals with identical effective score (same priority, no weights, no contextscore)
	g1 := gFixture("g1", "alpha", 50)
	g2 := gFixture("g2", "beta", 50)
	got, _, _ := Select([]*Goal{g1, g2}, nil, testMob, nil, 0, 0, 1000)
	if got.Id != "g1" {
		t.Errorf("got=%s, want g1 (id-asc tie-break)", got.Id)
	}
}

func TestSelect_StaleCurrentSinceRound_HeldClampsToZero(t *testing.T) {
	// currentSinceRound > nowRound. heldRounds must NOT underflow; min-hold gate must block.
	prev := gFixture("g1", "wealth", 30)
	chal := gFixture("g2", "revenge", 90)
	got, switched, reason := Select([]*Goal{prev, chal}, nil, testMob, prev,
		2_000_000, 2_000_000, 1_000_000) // since > now
	if got != prev {
		t.Errorf("got=%v, want prev (held clamped, min-hold blocks)", got)
	}
	if switched {
		t.Errorf("switched=true, want false")
	}
	if reason.Kind != "kept_hysteresis_min_hold" {
		t.Errorf("reason.Kind=%q, want %q", reason.Kind, "kept_hysteresis_min_hold")
	}
}
