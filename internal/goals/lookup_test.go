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
