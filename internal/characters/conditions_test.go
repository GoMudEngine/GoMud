package characters

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ─── ConditionBleeding ──────────────────────────────────────────────────────

func TestConditionBleeding_DisplayName(t *testing.T) {
	assert.Equal(t, "Bleeding", ConditionBleeding.DisplayName())
}

func TestConditionBleeding_Description(t *testing.T) {
	desc := ConditionBleeding.Description()
	assert.NotEmpty(t, desc)
	assert.Contains(t, desc, "blood")
}

func TestConditionBleeding_AddAndCheck(t *testing.T) {
	ch := &Character{}
	assert.False(t, ch.HasCondition(ConditionBleeding))

	ch.AddCondition(ConditionBleeding, 5, 3.0, "hamstring")
	assert.True(t, ch.HasCondition(ConditionBleeding))
	assert.Equal(t, 3.0, ch.GetConditionMagnitude(ConditionBleeding))
	assert.Equal(t, 5, ch.GetConditionDuration(ConditionBleeding))
}

func TestConditionBleeding_TickExpires(t *testing.T) {
	ch := &Character{}
	ch.AddCondition(ConditionBleeding, 2, 5.0, "test")

	ch.TickConditions()
	assert.True(t, ch.HasCondition(ConditionBleeding))
	assert.Equal(t, 1, ch.GetConditionDuration(ConditionBleeding))

	ch.TickConditions()
	assert.False(t, ch.HasCondition(ConditionBleeding))
}

func TestConditionBleeding_OverwritesExisting(t *testing.T) {
	ch := &Character{}
	ch.AddCondition(ConditionBleeding, 3, 2.0, "first")
	ch.AddCondition(ConditionBleeding, 5, 8.0, "second")

	// Should overwrite, not stack
	assert.Equal(t, 8.0, ch.GetConditionMagnitude(ConditionBleeding))
	assert.Equal(t, 5, ch.GetConditionDuration(ConditionBleeding))

	// Only one condition entry
	count := 0
	for _, c := range ch.Conditions {
		if c.Type == ConditionBleeding {
			count++
		}
	}
	assert.Equal(t, 1, count)
}

func TestConditionBleeding_RemoveExplicitly(t *testing.T) {
	ch := &Character{}
	ch.AddCondition(ConditionBleeding, 10, 5.0, "test")
	assert.True(t, ch.HasCondition(ConditionBleeding))

	ch.RemoveCondition(ConditionBleeding)
	assert.False(t, ch.HasCondition(ConditionBleeding))
}

// ─── All Conditions Display ─────────────────────────────────────────────────

func TestAllConditions_HaveDisplayNames(t *testing.T) {
	conditions := []ConditionType{
		ConditionRecoveryPenalty,
		ConditionDefensePenalty,
		ConditionGrappleController,
		ConditionShield,
		ConditionRegen,
		ConditionBlinded,
		ConditionPoisoned,
		ConditionEnchantWithdrawal,
		ConditionBleeding,
	}

	for _, c := range conditions {
		t.Run(c.DisplayName(), func(t *testing.T) {
			assert.NotEqual(t, "Unknown Condition", c.DisplayName(),
				"condition %d should have a display name", c)
			assert.NotEmpty(t, c.Description(),
				"condition %d should have a description", c)
		})
	}
}
