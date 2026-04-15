package mobai

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEffectiveReactionDelay(t *testing.T) {
	assert.Equal(t, 1.5, GetEffectiveReactionDelay(0, 0.25, 4.0))
	assert.Equal(t, 0.5, GetEffectiveReactionDelay(0.5, 0.25, 4.0))
	assert.Equal(t, 0.25, GetEffectiveReactionDelay(0.1, 0.25, 4.0))
	assert.Equal(t, 4.0, GetEffectiveReactionDelay(5.0, 0.25, 4.0))
}

func TestShouldFollowTactic_Always(t *testing.T) {
	for i := 0; i < 100; i++ {
		assert.True(t, ShouldFollowTactic(1.0))
	}
}

func TestShouldFollowTactic_Never(t *testing.T) {
	for i := 0; i < 100; i++ {
		assert.False(t, ShouldFollowTactic(0.0))
	}
}

func TestResolveTactics_PresetOnly(t *testing.T) {
	tactics := ResolveTactics("aggressive_melee", nil)
	assert.Greater(t, len(tactics), 0)
}

func TestResolveTactics_CustomOnly(t *testing.T) {
	custom := []TacticRule{{Trigger: "combat_start", Action: "flee", Priority: 1}}
	tactics := ResolveTactics("", custom)
	assert.Equal(t, 1, len(tactics))
}

func TestResolveTactics_Merged(t *testing.T) {
	custom := []TacticRule{{Trigger: "health_below:10", Action: "flee", Priority: 99}}
	tactics := ResolveTactics("aggressive_melee", custom)
	assert.Greater(t, len(tactics), 3) // preset + custom
}

func TestResolveTactics_None(t *testing.T) {
	tactics := ResolveTactics("", nil)
	assert.Nil(t, tactics)
}

func TestProcessSignal_QueuesReaction(t *testing.T) {
	ClearAllPending()

	tactics := []TacticRule{
		{Trigger: "combat_start", Action: "trip", Priority: 10},
	}
	ctx := &TriggerContext{}
	signal := SignalData{MobInstanceId: 1, SignalType: "combat_start"}
	cfg := ReactorConfig{Enabled: true, ReactionDelayMin: 0.25, ReactionDelayMax: 4.0, TurnsPerSecond: 10}

	action := ProcessSignal(signal, ctx, tactics, 1.0, 0.5, 0, cfg)
	assert.Equal(t, "trip", action)
	assert.Equal(t, 1, PendingCount())

	ClearAllPending()
}

func TestProcessSignal_Disabled(t *testing.T) {
	ClearAllPending()

	tactics := []TacticRule{
		{Trigger: "combat_start", Action: "trip", Priority: 10},
	}
	ctx := &TriggerContext{}
	signal := SignalData{MobInstanceId: 1, SignalType: "combat_start"}
	cfg := ReactorConfig{Enabled: false}

	action := ProcessSignal(signal, ctx, tactics, 1.0, 0.5, 0, cfg)
	assert.Equal(t, "", action)
	assert.Equal(t, 0, PendingCount())
}

func TestProcessSignal_NoMatchingTrigger(t *testing.T) {
	ClearAllPending()

	tactics := []TacticRule{
		{Trigger: "target_casting", Action: "trip", Priority: 10},
	}
	ctx := &TriggerContext{}
	signal := SignalData{MobInstanceId: 1, SignalType: "combat_start"}
	cfg := ReactorConfig{Enabled: true, ReactionDelayMin: 0.25, ReactionDelayMax: 4.0, TurnsPerSecond: 10}

	action := ProcessSignal(signal, ctx, tactics, 1.0, 0.5, 0, cfg)
	assert.Equal(t, "", action)
	assert.Equal(t, 0, PendingCount())
}

func TestClearPendingForMob(t *testing.T) {
	ClearAllPending()
	pendingReactions = append(pendingReactions,
		PendingReaction{MobInstanceId: 1, Action: "trip", FireTurn: 999},
		PendingReaction{MobInstanceId: 2, Action: "bash", FireTurn: 999},
		PendingReaction{MobInstanceId: 1, Action: "kick", FireTurn: 1000},
	)
	ClearPendingForMob(1)
	assert.Equal(t, 1, PendingCount())
}
