package catalog

import (
	"testing"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestWealthGold_Registered(t *testing.T) {
	if _, ok := goals.LookupGoalType("wealth-gold"); !ok {
		t.Fatalf("wealth-gold not registered")
	}
}

func TestWealthGold_Predicate_AtTarget_True(t *testing.T) {
	meta, _ := goals.LookupGoalType("wealth-gold")
	mob := &mobs.Mob{}
	mob.Character.Gold = 500
	g := &goals.Goal{Type: "wealth-gold", Params: map[string]any{"target": 500}}
	if !meta.Predicate(g, mob) {
		t.Errorf("predicate at target: got false, want true")
	}
}

func TestWealthGold_Predicate_BelowTarget_False(t *testing.T) {
	meta, _ := goals.LookupGoalType("wealth-gold")
	mob := &mobs.Mob{}
	mob.Character.Gold = 200
	g := &goals.Goal{Type: "wealth-gold", Params: map[string]any{"target": 500}}
	if meta.Predicate(g, mob) {
		t.Errorf("predicate below target: got true, want false")
	}
}

func TestWealthGold_ContextScore_Satisfied_Zero(t *testing.T) {
	meta, _ := goals.LookupGoalType("wealth-gold")
	mob := &mobs.Mob{}
	mob.Character.Gold = 500
	g := &goals.Goal{Type: "wealth-gold", Params: map[string]any{"target": 500}}
	if got := meta.ContextScore(g, mob); got != 0 {
		t.Errorf("score satisfied: got %f, want 0", got)
	}
}

func TestWealthGold_ContextScore_HalfWay_Scaled(t *testing.T) {
	meta, _ := goals.LookupGoalType("wealth-gold")
	mob := &mobs.Mob{}
	mob.Character.Gold = 250
	g := &goals.Goal{Type: "wealth-gold", Params: map[string]any{"target": 500}}
	got := meta.ContextScore(g, mob)
	// Baseline 1.0 + (target-gold)/target = 0.5 added → 1.5
	if got < 1.4 || got > 1.6 {
		t.Errorf("score at 50%% target: got %f, want ~1.5", got)
	}
}

func TestWealthGold_ContextScore_Empty_MaxTwo(t *testing.T) {
	meta, _ := goals.LookupGoalType("wealth-gold")
	mob := &mobs.Mob{}
	mob.Character.Gold = 0
	g := &goals.Goal{Type: "wealth-gold", Params: map[string]any{"target": 500}}
	got := meta.ContextScore(g, mob)
	// Baseline 1.0 + (target-gold)/target = 1.0 added, capped at 2.0
	if got != 2.0 {
		t.Errorf("score empty: got %f, want 2.0", got)
	}
}
