package mobai

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvaluateTactics_HighestPriorityWins(t *testing.T) {
	tactics := []TacticRule{
		{Trigger: "combat_start", Action: "cast shield", Priority: 5},
		{Trigger: "combat_start", Action: "trip", Priority: 10},
	}
	ctx := &TriggerContext{CombatJustStarted: true}
	action := EvaluateTactics(tactics, ctx)
	assert.Equal(t, "trip", action)
}

func TestEvaluateTactics_NoMatch(t *testing.T) {
	tactics := []TacticRule{
		{Trigger: "target_casting", Action: "trip", Priority: 10},
	}
	ctx := &TriggerContext{CombatJustStarted: true}
	action := EvaluateTactics(tactics, ctx)
	assert.Equal(t, "", action)
}

func TestEvaluateTactics_Empty(t *testing.T) {
	action := EvaluateTactics(nil, &TriggerContext{})
	assert.Equal(t, "", action)
}

func TestGetPreset_Known(t *testing.T) {
	preset := GetPreset("aggressive_melee")
	assert.Greater(t, len(preset), 0)
}

func TestGetPreset_Unknown(t *testing.T) {
	preset := GetPreset("nonexistent")
	assert.Nil(t, preset)
}

func TestMergeTactics(t *testing.T) {
	preset := []TacticRule{
		{Trigger: "target_prone", Action: "kick", Priority: 10},
	}
	custom := []TacticRule{
		{Trigger: "health_below:20", Action: "flee", Priority: 15},
	}
	merged := MergeTactics(preset, custom)
	assert.Equal(t, 2, len(merged))
}
