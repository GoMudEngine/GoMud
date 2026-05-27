package catalog

import (
	"testing"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestSurvival_Registered(t *testing.T) {
	if _, ok := goals.LookupGoalType("survival"); !ok {
		t.Fatalf("survival not registered")
	}
}

func TestSurvival_Predicate_FullHPNotInCombat_True(t *testing.T) {
	meta, _ := goals.LookupGoalType("survival")
	mob := &mobs.Mob{}
	mob.Character.Health = 100
	mob.Character.HealthMax.Value = 100
	g := &goals.Goal{Type: "survival", Priority: 80}
	if !meta.Predicate(g, mob) {
		t.Errorf("predicate at full HP: got false, want true (recovered = satisfied)")
	}
}

func TestSurvival_Predicate_LowHP_False(t *testing.T) {
	meta, _ := goals.LookupGoalType("survival")
	mob := &mobs.Mob{}
	mob.Character.Health = 20
	mob.Character.HealthMax.Value = 100
	g := &goals.Goal{Type: "survival", Priority: 80}
	if meta.Predicate(g, mob) {
		t.Errorf("predicate at 20%% HP: got true, want false")
	}
}

func TestSurvival_ContextScore_FullHP_Zero(t *testing.T) {
	meta, _ := goals.LookupGoalType("survival")
	mob := &mobs.Mob{}
	mob.Character.Health = 100
	mob.Character.HealthMax.Value = 100
	got := meta.ContextScore(&goals.Goal{Type: "survival"}, mob)
	if got != 0 {
		t.Errorf("context score at full HP: got %f, want 0 (filtered)", got)
	}
}

func TestSurvival_ContextScore_MidWound_Linear(t *testing.T) {
	meta, _ := goals.LookupGoalType("survival")
	mob := &mobs.Mob{}
	mob.Character.Health = 40
	mob.Character.HealthMax.Value = 100
	got := meta.ContextScore(&goals.Goal{Type: "survival"}, mob)
	// Between flee (25) and safe (60) thresholds → linear from 1.0 to 3.0.
	// At 40%: roughly (60-40)/(60-25) = 0.57 fraction → 1.0 + 0.57*(3.0-1.0) ≈ 2.14
	if got < 1.5 || got > 2.5 {
		t.Errorf("context score at 40%% HP: got %f, want ~2.14 (1.5-2.5)", got)
	}
}

func TestSurvival_ContextScore_Critical_FivePointZero(t *testing.T) {
	meta, _ := goals.LookupGoalType("survival")
	mob := &mobs.Mob{}
	mob.Character.Health = 5
	mob.Character.HealthMax.Value = 100
	got := meta.ContextScore(&goals.Goal{Type: "survival"}, mob)
	if got != 5.0 {
		t.Errorf("context score at 5%% HP: got %f, want 5.0", got)
	}
}
