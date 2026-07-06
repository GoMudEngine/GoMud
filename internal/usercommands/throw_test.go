package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setMobCastingForTest puts a mob's Character into the Casting activity
// state. Mirrors internal/actions/cast_interrupt_test.go's setCastingForTest
// helper (unexported there, so replicated here for this package's tests).
func setMobCastingForTest(mob *mobs.Mob, spellId string) {
	mob.Character.Activity = activity.NewMachine()
	_ = mob.Character.Activity.TransitionToCasting(
		activity.CastingData{SpellId: spellId},
		state.TransitionReason{Trigger: activity.TriggerCastBegin},
	)
}

// newTestMob builds a minimal standalone mob instance (not registered in the
// mob registry) suitable for exercising maybeInterruptOnThrow directly.
func newTestMob(instanceId int) *mobs.Mob {
	m := &mobs.Mob{
		MobId:      1,
		InstanceId: instanceId,
		HomeRoomId: 1,
		Character: characters.Character{
			Name:      "Warden-Prime",
			RoomId:    1,
			Health:    500,
			Buffs:     buffs.New(),
			Cooldowns: map[string]int{},
		},
	}
	m.Character.HealthMax.Value = 500
	return m
}

// TestMaybeInterruptOnThrow_DisruptorInterruptsCastingMob: a configured
// disruptor item (the flashbang, 30057) thrown at a mid-cast mob cancels the
// cast via the shared InterruptTargetCast primitive.
func TestMaybeInterruptOnThrow_DisruptorInterruptsCastingMob(t *testing.T) {
	mob := newTestMob(500)
	setMobCastingForTest(mob, "core-discharge")
	require.True(t, mob.Character.IsCasting(), "pre-condition: mob should be casting")

	by := state.ActorRef{UserId: 1}
	interrupted := maybeInterruptOnThrow(mob, 30057, by)

	assert.True(t, interrupted, "a configured disruptor must interrupt a casting mob")
	assert.False(t, mob.Character.IsCasting(), "mob's cast should be cancelled after the interrupt")
}

// TestMaybeInterruptOnThrow_NonDisruptorDoesNotInterrupt: a thrown item NOT
// in BossInterruptItemIds (generic grenade / rock / etc.) must never
// interrupt a cast — only the allowlisted disruptor does.
func TestMaybeInterruptOnThrow_NonDisruptorDoesNotInterrupt(t *testing.T) {
	mob := newTestMob(501)
	setMobCastingForTest(mob, "core-discharge")
	require.True(t, mob.Character.IsCasting(), "pre-condition: mob should be casting")

	by := state.ActorRef{UserId: 1}
	interrupted := maybeInterruptOnThrow(mob, 40001, by) // arbitrary non-disruptor item id

	assert.False(t, interrupted, "a non-disruptor item must not interrupt a cast")
	assert.True(t, mob.Character.IsCasting(), "mob should still be casting")
}

// TestMaybeInterruptOnThrow_DisruptorOnNonCastingMob_NoOp: throwing a
// disruptor at a mob that isn't casting anything is a no-op — nothing to
// interrupt.
func TestMaybeInterruptOnThrow_DisruptorOnNonCastingMob_NoOp(t *testing.T) {
	mob := newTestMob(502)
	require.False(t, mob.Character.IsCasting(), "pre-condition: mob should not be casting")

	by := state.ActorRef{UserId: 1}
	interrupted := maybeInterruptOnThrow(mob, 30057, by)

	assert.False(t, interrupted, "a disruptor thrown at a non-casting mob is a no-op")
}

// TestMaybeInterruptOnThrow_NilMob_NoPanic: defensive nil guard.
func TestMaybeInterruptOnThrow_NilMob_NoPanic(t *testing.T) {
	by := state.ActorRef{UserId: 1}
	assert.NotPanics(t, func() {
		interrupted := maybeInterruptOnThrow(nil, 30057, by)
		assert.False(t, interrupted)
	})
}
