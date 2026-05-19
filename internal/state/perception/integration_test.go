package perception_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/state/perception"
)

// seedBlindBuffs registers minimal BuffSpec entries for the two blind-source
// buff IDs (3 = Blinded, 77 = Flashbang Blindness) into the global buff
// registry and returns a cleanup function that restores the original state.
//
// Without seeded specs, buffs.AddBuff returns false (spec not found) and
// characters.AddBuff returns an error, causing every PE-INT test to fail
// at setup rather than at the assertion under test.
//
// TriggerCount=5, RoundInterval=1 gives a non-zero lifespan so the buff
// is not immediately expired on add.
func seedBlindBuffs(t *testing.T) func() {
	t.Helper()
	cleanup := buffs.SeedBuffsForTest(map[int]*buffs.BuffSpec{
		perception.BuffIdBlinded: {
			BuffId:        perception.BuffIdBlinded,
			Name:          "Blinded",
			Description:   "Your eyes are blinded.",
			TriggerCount:  5,
			RoundInterval: 1,
		},
		perception.BuffIdFlashbangBlindness: {
			BuffId:        perception.BuffIdFlashbangBlindness,
			Name:          "Flashbang Blindness",
			Description:   "A flash of light sears your vision.",
			TriggerCount:  3,
			RoundInterval: 1,
		},
	})
	return cleanup
}

// TestMain initialises the mudlog logger once for all integration tests in
// this file. buffs.Validate() calls mudlog.Warn when it encounters an unknown
// buffId; without initialisation the logger panics on a nil receiver.
func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "", "", false)
	m.Run()
}

// PE-INT-001: AddBuff(3) → Blinded.
func TestIntegration_BuffBlindedAppliesBlinded(t *testing.T) {
	defer seedBlindBuffs(t)()

	c := characters.New()
	if c.Perception.State() != perception.Sighted {
		t.Fatalf("initial state = %v, want Sighted", c.Perception.State())
	}
	if err := c.AddBuff(perception.BuffIdBlinded, false); err != nil {
		t.Fatalf("AddBuff(3): %v", err)
	}
	if c.Perception.State() != perception.Blinded {
		t.Errorf("after AddBuff(3), state = %v, want Blinded", c.Perception.State())
	}
}

// PE-INT-002: AddBuff(3) + AddCondition(Blinded) + RemoveBuff(3) → still Blinded.
func TestIntegration_OverlapKeepsBlinded(t *testing.T) {
	defer seedBlindBuffs(t)()

	c := characters.New()
	if err := c.AddBuff(perception.BuffIdBlinded, false); err != nil {
		t.Fatalf("AddBuff(3): %v", err)
	}
	c.AddCondition(characters.ConditionBlinded, 5, 0.7, "test-overlap")
	c.RemoveBuff(perception.BuffIdBlinded)
	if c.Perception.State() != perception.Blinded {
		t.Errorf("after removing buff but condition still active, state = %v, want Blinded", c.Perception.State())
	}
}

// PE-INT-003: Add both, remove both → Sighted.
func TestIntegration_AllSourcesClearedReturnsSighted(t *testing.T) {
	defer seedBlindBuffs(t)()

	c := characters.New()
	_ = c.AddBuff(perception.BuffIdBlinded, false)
	c.AddCondition(characters.ConditionBlinded, 5, 0.7, "test-clear")
	c.RemoveBuff(perception.BuffIdBlinded)
	c.RemoveCondition(characters.ConditionBlinded)
	if c.Perception.State() != perception.Sighted {
		t.Errorf("after clearing all sources, state = %v, want Sighted", c.Perception.State())
	}
}

// PE-INT-004 (PE-010 from spec matrix): re-applying buff while already
// Blinded is a no-op (no ErrInvalidTransition propagated, no log spam).
func TestIntegration_ReapplyBuffNoOp(t *testing.T) {
	defer seedBlindBuffs(t)()

	c := characters.New()
	if err := c.AddBuff(perception.BuffIdBlinded, false); err != nil {
		t.Fatalf("first AddBuff: %v", err)
	}
	// Re-add the same buff (the buff system stacks duration, but the
	// blind-source state is the same). The current-state guard in
	// AddBuff prevents the transition from firing twice.
	if err := c.AddBuff(perception.BuffIdBlinded, false); err != nil {
		t.Fatalf("second AddBuff: %v", err)
	}
	if c.Perception.State() != perception.Blinded {
		t.Errorf("after duplicate AddBuff, state = %v, want Blinded", c.Perception.State())
	}
}

// PE-INT-005: Flashbang (buff 77) drives the same Perception transitions.
func TestIntegration_FlashbangBlindness(t *testing.T) {
	defer seedBlindBuffs(t)()

	c := characters.New()
	if err := c.AddBuff(perception.BuffIdFlashbangBlindness, false); err != nil {
		t.Fatalf("AddBuff(77): %v", err)
	}
	if c.Perception.State() != perception.Blinded {
		t.Errorf("after AddBuff(77), state = %v, want Blinded", c.Perception.State())
	}
	c.RemoveBuff(perception.BuffIdFlashbangBlindness)
	if c.Perception.State() != perception.Sighted {
		t.Errorf("after RemoveBuff(77), state = %v, want Sighted", c.Perception.State())
	}
}

// PE-INT-006: ConditionBlinded alone drives the transitions.
func TestIntegration_ConditionBlindedSolo(t *testing.T) {
	c := characters.New()
	c.AddCondition(characters.ConditionBlinded, 3, 0.5, "test-solo-cond")
	if c.Perception.State() != perception.Blinded {
		t.Errorf("after AddCondition(Blinded), state = %v, want Blinded", c.Perception.State())
	}
	c.RemoveCondition(characters.ConditionBlinded)
	if c.Perception.State() != perception.Sighted {
		t.Errorf("after RemoveCondition(Blinded), state = %v, want Sighted", c.Perception.State())
	}
}

// PE-INT-007: Mixed source order — condition added first, then buff,
// then condition removed → still Blinded (buff still active).
func TestIntegration_MixedSourceOrder(t *testing.T) {
	defer seedBlindBuffs(t)()

	c := characters.New()
	c.AddCondition(characters.ConditionBlinded, 5, 0.7, "test-order")
	if c.Perception.State() != perception.Blinded {
		t.Fatalf("after AddCondition, state = %v, want Blinded", c.Perception.State())
	}
	if err := c.AddBuff(perception.BuffIdBlinded, false); err != nil {
		t.Fatalf("AddBuff(3): %v", err)
	}
	// Re-adding-while-already-Blinded path; state must remain Blinded.
	if c.Perception.State() != perception.Blinded {
		t.Errorf("after AddBuff while condition active, state = %v, want Blinded", c.Perception.State())
	}
	c.RemoveCondition(characters.ConditionBlinded)
	// Buff still active → still Blinded.
	if c.Perception.State() != perception.Blinded {
		t.Errorf("after RemoveCondition (buff still active), state = %v, want Blinded", c.Perception.State())
	}
	c.RemoveBuff(perception.BuffIdBlinded)
	if c.Perception.State() != perception.Sighted {
		t.Errorf("after RemoveBuff (no sources left), state = %v, want Sighted", c.Perception.State())
	}
}
