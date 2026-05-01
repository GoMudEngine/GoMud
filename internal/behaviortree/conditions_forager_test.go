package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
)

// buildForagerCondMob creates a minimal mob for condition tests.
func buildForagerCondMob(
	t *testing.T,
	instanceId int,
	mobId mobs.MobId,
	hp, hpMax int,
) *mobs.Mob {
	t.Helper()
	mob := &mobs.Mob{
		MobId:      mobId,
		InstanceId: instanceId,
		HomeRoomId: 1,
	}
	mob.Character.RoomId = 1
	mob.Character.Health = hp
	mob.Character.HealthMax.Value = hpMax
	mob.Character.Buffs = buffs.New()
	mob.Character.Stats.Strength.ValueAdj = 100
	mobs.SetInstanceForTest(instanceId, mob)
	t.Cleanup(func() { mobs.SetInstanceForTest(instanceId, nil) })
	return mob
}

// ── condMobInventoryAtThreshold ──────────────────────────────────────

// TestCondMobInventoryAtThreshold_BelowFalse verifies that an empty
// inventory returns Failure (carry ratio 0 < threshold).
func TestCondMobInventoryAtThreshold_BelowFalse(t *testing.T) {
	mob := buildForagerCondMob(t, 8500, 371, 100, 100)
	_ = mob

	ctx := &EvalContext{InstanceId: 8500}
	assert.Equal(t, Failure,
		condMobInventoryAtThreshold(nil, ctx),
		"empty inventory should return Failure")
}

// TestCondMobInventoryAtThreshold_UnknownMobFails verifies Failure when
// the mob instance does not exist.
func TestCondMobInventoryAtThreshold_UnknownMobFails(t *testing.T) {
	ctx := &EvalContext{InstanceId: 99998}
	assert.Equal(t, Failure, condMobInventoryAtThreshold(nil, ctx))
}

// ── condMobHPBelowRecallThreshold ────────────────────────────────────

// TestCondMobHPBelowRecallThreshold_AboveFalse verifies that a mob at
// full HP returns Failure (above the 50% recall threshold).
func TestCondMobHPBelowRecallThreshold_AboveFalse(t *testing.T) {
	mob := buildForagerCondMob(t, 8501, 371, 100, 100)
	_ = mob

	ctx := &EvalContext{InstanceId: 8501}
	assert.Equal(t, Failure,
		condMobHPBelowRecallThreshold(nil, ctx),
		"full HP should return Failure")
}

// TestCondMobHPBelowRecallThreshold_BelowTrue verifies that a mob at
// 30% HP returns Success (below the 50% recall threshold).
func TestCondMobHPBelowRecallThreshold_BelowTrue(t *testing.T) {
	mob := buildForagerCondMob(t, 8502, 371, 30, 100)
	_ = mob

	ctx := &EvalContext{InstanceId: 8502}
	assert.Equal(t, Success,
		condMobHPBelowRecallThreshold(nil, ctx),
		"30% HP should return Success")
}

// TestCondMobHPBelowRecallThreshold_AtThresholdTrue verifies the
// boundary: at exactly 50% HP the condition fires (≤ threshold).
func TestCondMobHPBelowRecallThreshold_AtThresholdTrue(t *testing.T) {
	mob := buildForagerCondMob(t, 8503, 371, 50, 100)
	_ = mob

	ctx := &EvalContext{InstanceId: 8503}
	assert.Equal(t, Success,
		condMobHPBelowRecallThreshold(nil, ctx),
		"50% HP (exactly at threshold) should return Success")
}

// TestCondMobHPBelowRecallThreshold_UnknownMobFails verifies Failure
// when no mob instance exists.
func TestCondMobHPBelowRecallThreshold_UnknownMobFails(t *testing.T) {
	ctx := &EvalContext{InstanceId: 99997}
	assert.Equal(t, Failure, condMobHPBelowRecallThreshold(nil, ctx))
}

// ── condMobCanSafelyEngage ───────────────────────────────────────────

// TestCondMobCanSafelyEngage_NoAggroFails verifies that a mob without
// aggro returns Failure.
func TestCondMobCanSafelyEngage_NoAggroFails(t *testing.T) {
	mob := buildForagerCondMob(t, 8504, 371, 100, 100)
	_ = mob // no Aggro set

	ctx := &EvalContext{InstanceId: 8504}
	assert.Equal(t, Failure, condMobCanSafelyEngage(nil, ctx))
}

// TestCondMobCanSafelyEngage_NonPreyTargetFails verifies that targeting
// a mob not on the prey whitelist returns Failure.
func TestCondMobCanSafelyEngage_NonPreyTargetFails(t *testing.T) {
	// Create the forager (mob 371) with aggro on a non-prey mob.
	foragerMob := buildForagerCondMob(t, 8505, 371, 100, 100)
	// Target mob ID 999 is not on Tova's prey whitelist.
	targetMob := buildForagerCondMob(t, 8506, 999, 100, 100)
	_ = targetMob

	foragerMob.Character.Aggro = &characters.Aggro{
		MobInstanceId: 8506,
	}

	ctx := &EvalContext{InstanceId: 8505}
	assert.Equal(t, Failure, condMobCanSafelyEngage(nil, ctx))
}
