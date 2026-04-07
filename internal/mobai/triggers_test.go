package mobai

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/stretchr/testify/assert"
)

func TestEvalTrigger_CombatStart(t *testing.T) {
	ctx := &TriggerContext{CombatJustStarted: true}
	assert.True(t, EvalTrigger("combat_start", ctx))
	ctx.CombatJustStarted = false
	assert.False(t, EvalTrigger("combat_start", ctx))
}

func TestEvalTrigger_HealthBelow(t *testing.T) {
	mob := characters.New()
	mob.Health = 20
	mob.HealthMax.Value = 100
	ctx := &TriggerContext{MobChar: mob}
	assert.True(t, EvalTrigger("health_below:30", ctx))
	assert.False(t, EvalTrigger("health_below:10", ctx))
}

func TestEvalTrigger_MultipleTargets(t *testing.T) {
	ctx := &TriggerContext{EnemyCount: 3}
	assert.True(t, EvalTrigger("multiple_targets", ctx))
	ctx.EnemyCount = 1
	assert.False(t, EvalTrigger("multiple_targets", ctx))
}

func TestEvalTrigger_AfterAction(t *testing.T) {
	ctx := &TriggerContext{LastAction: "trip"}
	assert.True(t, EvalTrigger("after_action:trip", ctx))
	assert.False(t, EvalTrigger("after_action:bash", ctx))
}

func TestEvalTrigger_NoAggro(t *testing.T) {
	ctx := &TriggerContext{HasAggro: false, HasMemory: true}
	assert.True(t, EvalTrigger("no_aggro", ctx))
	ctx.HasAggro = true
	assert.False(t, EvalTrigger("no_aggro", ctx))
}

func TestEvalTrigger_HasBuff(t *testing.T) {
	ctx := &TriggerContext{ActiveBuffIds: []int{5, 10, 15}}
	assert.True(t, EvalTrigger("has_buff:10", ctx))
	assert.False(t, EvalTrigger("has_buff:99", ctx))
}

func TestEvalTrigger_MissingBuff(t *testing.T) {
	ctx := &TriggerContext{ActiveBuffIds: []int{5, 10}}
	assert.True(t, EvalTrigger("missing_buff:99", ctx))
	assert.False(t, EvalTrigger("missing_buff:10", ctx))
}

func TestEvalTrigger_TargetProne(t *testing.T) {
	target := characters.New()
	target.CombatPosition = characters.PositionProne
	ctx := &TriggerContext{Target: target}
	assert.True(t, EvalTrigger("target_prone", ctx))
	target.CombatPosition = characters.PositionStanding
	assert.False(t, EvalTrigger("target_prone", ctx))
}

func TestEvalTrigger_TargetGrappled(t *testing.T) {
	target := characters.New()
	target.CombatPosition = characters.PositionGrounded
	ctx := &TriggerContext{Target: target}
	assert.True(t, EvalTrigger("target_grappled", ctx))
	target.CombatPosition = characters.PositionClinched
	assert.True(t, EvalTrigger("target_grappled", ctx))
	target.CombatPosition = characters.PositionStanding
	assert.False(t, EvalTrigger("target_grappled", ctx))
}

func TestEvalTrigger_Unknown(t *testing.T) {
	ctx := &TriggerContext{}
	assert.False(t, EvalTrigger("nonexistent_trigger", ctx))
}
