package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
	"github.com/stretchr/testify/assert"
)

// setCastingForTest puts a character into casting state with the supplied data.
func setCastingForTest(char *characters.Character, data activity.CastingData) {
	if char.Activity == nil {
		char.Activity = activity.NewMachine()
	}
	_ = char.Activity.TransitionToCasting(data,
		state.TransitionReason{Trigger: activity.TriggerCastBegin})
}

// TestInterruptTargetCast_NonCasting verifies that InterruptTargetCast returns
// false when the target is not currently casting.
func TestInterruptTargetCast_NonCasting(t *testing.T) {
	target := characters.New()
	// Activity is Free (default).
	by := state.ActorRef{MobInstanceId: 1}

	interrupted := InterruptTargetCast(target, by)

	assert.False(t, interrupted, "non-casting target should return false")
	assert.False(t, target.IsCasting(), "target should still not be casting")
}

// TestInterruptTargetCast_Casting verifies that InterruptTargetCast returns
// true, cancels the cast, and applies the 50% conviction refund.
func TestInterruptTargetCast_Casting(t *testing.T) {
	target := characters.New()
	target.ConvictionMax.Value = 100
	target.Conviction = 10 // starting conviction before refund

	setCastingForTest(target, activity.CastingData{
		SpellId:             "fireball",
		TotalConvictionCost: 20,
		ConvictionSpent:     4, // 16 unspent → refund 8
	})

	assert.True(t, target.IsCasting(), "pre-condition: target should be casting")

	by := state.ActorRef{MobInstanceId: 42}
	interrupted := InterruptTargetCast(target, by)

	assert.True(t, interrupted, "casting target should be interrupted")
	assert.False(t, target.IsCasting(), "target should no longer be casting after interrupt")

	// Conviction refund: 16 unspent → refund 8; starting 10 + 8 = 18
	assert.Equal(t, 18, target.Conviction, "50%% of unspent conviction should be refunded")
}

// TestInterruptTargetCast_ConvictionClamped verifies the refund does not
// exceed ConvictionMax.
func TestInterruptTargetCast_ConvictionClamped(t *testing.T) {
	target := characters.New()
	target.ConvictionMax.Value = 20
	target.Conviction = 18 // nearly full

	setCastingForTest(target, activity.CastingData{
		SpellId:             "icebolt",
		TotalConvictionCost: 30,
		ConvictionSpent:     0, // 30 unspent → refund 15; but max is 20
	})

	by := state.ActorRef{UserId: 5}
	InterruptTargetCast(target, by)

	assert.Equal(t, 20, target.Conviction, "conviction should be clamped to ConvictionMax")
}

// TestInterruptTargetCast_NilActivity verifies that a nil Activity field
// (zero-value character) is handled gracefully.
func TestInterruptTargetCast_NilActivity(t *testing.T) {
	var char characters.Character // zero value; Activity == nil
	by := state.ActorRef{UserId: 1}

	assert.NotPanics(t, func() {
		result := InterruptTargetCast(&char, by)
		assert.False(t, result)
	}, "nil Activity should not panic")
}
