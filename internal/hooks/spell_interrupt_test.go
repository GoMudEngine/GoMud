package hooks

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

// setMobCastingForSpellTest puts a mob's Character into the Casting activity
// state. Mirrors internal/actions/cast_interrupt_test.go's setCastingForTest
// helper (unexported there, so replicated here for this package's tests).
func setMobCastingForSpellTest(mob *mobs.Mob, spellId string) {
	mob.Character.Activity = activity.NewMachine()
	_ = mob.Character.Activity.TransitionToCasting(
		activity.CastingData{SpellId: spellId},
		state.TransitionReason{Trigger: activity.TriggerCastBegin},
	)
}

// newSpellTestMob builds a minimal standalone mob instance (not registered in
// the mob registry) suitable for exercising maybeInterruptSpellOnMob directly.
func newSpellTestMob(instanceId int) *mobs.Mob {
	m := &mobs.Mob{
		MobId:      1,
		InstanceId: instanceId,
		HomeRoomId: 1,
		Character: characters.Character{
			Name:      "Core Guardian",
			RoomId:    1,
			Health:    500,
			Buffs:     buffs.New(),
			Cooldowns: map[string]int{},
		},
	}
	m.Character.HealthMax.Value = 500
	return m
}

// TestMaybeInterruptSpellOnMob_DisruptionSpellInterruptsCastingMob: a player
// casting a configured disruption spell (e.g. "neural-stun") at a mid-cast
// mob cancels the cast via the shared InterruptTargetCast primitive.
func TestMaybeInterruptSpellOnMob_DisruptionSpellInterruptsCastingMob(t *testing.T) {
	mob := newSpellTestMob(600)
	setMobCastingForSpellTest(mob, "core-discharge")
	require.True(t, mob.Character.IsCasting(), "pre-condition: mob should be casting")

	by := state.ActorRef{UserId: 1}
	interrupted := maybeInterruptSpellOnMob(mob, "neural-stun", by)

	assert.True(t, interrupted, "a configured disruption spell must interrupt a casting mob")
	assert.False(t, mob.Character.IsCasting(), "mob's cast should be cancelled after the interrupt")
}

// TestMaybeInterruptSpellOnMob_NonDisruptionSpellDoesNotInterrupt: a spell NOT
// in BossInterruptSpellIds (e.g. a generic damage spell) must never
// interrupt a cast — only the allowlisted disruption spells do.
func TestMaybeInterruptSpellOnMob_NonDisruptionSpellDoesNotInterrupt(t *testing.T) {
	mob := newSpellTestMob(601)
	setMobCastingForSpellTest(mob, "core-discharge")
	require.True(t, mob.Character.IsCasting(), "pre-condition: mob should be casting")

	by := state.ActorRef{UserId: 1}
	interrupted := maybeInterruptSpellOnMob(mob, "fireball", by)

	assert.False(t, interrupted, "a non-allowlisted spell must not interrupt a cast")
	assert.True(t, mob.Character.IsCasting(), "mob should still be casting")
}

// TestMaybeInterruptSpellOnMob_DisruptionSpellOnNonCastingMob_NoOp: casting a
// disruption spell at a mob that isn't casting anything is a no-op.
func TestMaybeInterruptSpellOnMob_DisruptionSpellOnNonCastingMob_NoOp(t *testing.T) {
	mob := newSpellTestMob(602)
	require.False(t, mob.Character.IsCasting(), "pre-condition: mob should not be casting")

	by := state.ActorRef{UserId: 1}
	interrupted := maybeInterruptSpellOnMob(mob, "neural-stun", by)

	assert.False(t, interrupted, "a disruption spell cast at a non-casting mob is a no-op")
}

// TestMaybeInterruptSpellOnMob_NilMob_NoPanic: defensive nil guard.
func TestMaybeInterruptSpellOnMob_NilMob_NoPanic(t *testing.T) {
	by := state.ActorRef{UserId: 1}
	assert.NotPanics(t, func() {
		interrupted := maybeInterruptSpellOnMob(nil, "neural-stun", by)
		assert.False(t, interrupted)
	})
}

// TestMaybeInterruptSpellOnMob_AllThreeDisruptionSpellsInterrupt confirms all
// three configured disruption spell ids (neural-stun, sensory-overload,
// kinetic-shove) interrupt — not just the first one checked.
func TestMaybeInterruptSpellOnMob_AllThreeDisruptionSpellsInterrupt(t *testing.T) {
	for i, spellId := range []string{"neural-stun", "sensory-overload", "kinetic-shove"} {
		mob := newSpellTestMob(610 + i)
		setMobCastingForSpellTest(mob, "core-discharge")

		by := state.ActorRef{UserId: 1}
		interrupted := maybeInterruptSpellOnMob(mob, spellId, by)

		assert.True(t, interrupted, "%q should interrupt a casting mob", spellId)
		assert.False(t, mob.Character.IsCasting(), "%q should have cancelled the cast", spellId)
	}
}
