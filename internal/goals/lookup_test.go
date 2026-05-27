package goals

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestSetWeightsLookup_Registered_ResolvesViaCallback(t *testing.T) {
	called := false
	SetWeightsLookup(func(mob *mobs.Mob) map[string]float64 {
		called = true
		return map[string]float64{"revenge": 2.0}
	})
	defer SetWeightsLookup(nil)

	mob := &mobs.Mob{}
	got := resolveWeights(mob)
	if !called {
		t.Errorf("lookup callback not invoked")
	}
	if got["revenge"] != 2.0 {
		t.Errorf("revenge weight: got %f, want 2.0", got["revenge"])
	}
}

func TestResolveWeights_NoLookupRegistered_ReturnsNil(t *testing.T) {
	SetWeightsLookup(nil)
	got := resolveWeights(&mobs.Mob{})
	if got != nil {
		t.Errorf("got=%v, want nil (no lookup registered)", got)
	}
}

func TestSetArchetypeDefaultsLookup_Registered_Resolves(t *testing.T) {
	called := false
	SetArchetypeDefaultsLookup(func(mob *mobs.Mob) []GoalDefault {
		called = true
		return []GoalDefault{
			{Type: "survival", Priority: 80},
			{Type: "wealth-gold", Priority: 40, Params: map[string]any{"target": 500}},
		}
	})
	defer SetArchetypeDefaultsLookup(nil)

	got := resolveArchetypeDefaults(&mobs.Mob{})
	if !called {
		t.Errorf("defaults lookup callback not invoked")
	}
	if len(got) != 2 {
		t.Fatalf("len=%d, want 2", len(got))
	}
	if got[0].Type != "survival" || got[1].Type != "wealth-gold" {
		t.Errorf("types: got [%s, %s], want [survival, wealth-gold]", got[0].Type, got[1].Type)
	}
}

func TestResolveArchetypeDefaults_NoLookup_ReturnsNil(t *testing.T) {
	SetArchetypeDefaultsLookup(nil)
	got := resolveArchetypeDefaults(&mobs.Mob{})
	if got != nil {
		t.Errorf("got=%v, want nil", got)
	}
}
