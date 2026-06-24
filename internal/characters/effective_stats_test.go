package characters

import (
	"testing"
)

// TestEffectiveStats_ToxicityPenalty verifies that GetEffectivePerception and
// GetEffectiveDexterity return the raw ValueAdj at zero toxicity and apply
// the configured penalty multipliers at high toxicity.
//
// Band values confirmed from GetToxicityPenalties (resources.go):
//
//	ratio >= 0.90 → (regenMult=0.60, perMult=0.80, dexMult=0.90)
//
// At max toxicity: ratio = 1.0 ≥ 0.90 → perMult=0.80, dexMult=0.90.
// With base stats 100: effective Per = int(100*0.80) = 80, Dex = int(100*0.90) = 90.
func TestEffectiveStats_ToxicityPenalty(t *testing.T) {
	c := &Character{}
	c.Stats.Perception.ValueAdj = 100
	c.Stats.Dexterity.ValueAdj = 100

	// At zero toxicity: no penalty; raw stats pass through unchanged.
	if got := c.GetEffectivePerception(); got != 100 {
		t.Errorf("GetEffectivePerception at tox=0: want 100, got %d", got)
	}
	if got := c.GetEffectiveDexterity(); got != 100 {
		t.Errorf("GetEffectiveDexterity at tox=0: want 100, got %d", got)
	}

	// At max toxicity (ratio = 1.0 ≥ 0.90): perMult=0.80, dexMult=0.90.
	c.Toxicity = c.GetToxicityMax()
	if got := c.GetEffectivePerception(); got != 80 {
		t.Errorf("GetEffectivePerception at max tox: want 80, got %d", got)
	}
	if got := c.GetEffectiveDexterity(); got != 90 {
		t.Errorf("GetEffectiveDexterity at max tox: want 90, got %d", got)
	}
}
