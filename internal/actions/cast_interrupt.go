package actions

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
)

// InterruptTargetCast cancels target's in-progress spellcast using the
// engine's standard cast-cancel path (50% unspent-conviction refund +
// the TriggerCastCancel activity transition). Returns true if the target
// was casting and the cast was interrupted; false (no-op) otherwise.
func InterruptTargetCast(target *characters.Character, by state.ActorRef) bool {
	a := target.Activity
	if a == nil || !a.IsCasting() {
		return false
	}
	if d, ok := a.CastingData(); ok {
		unspent := d.TotalConvictionCost - d.ConvictionSpent
		if unspent > 0 {
			target.Conviction += unspent / 2
			if target.Conviction > target.ConvictionMax.Value {
				target.Conviction = target.ConvictionMax.Value
			}
		}
	}
	_ = a.TransitionToFree(state.TransitionReason{
		Trigger: activity.TriggerCastCancel,
		Actor:   by,
	})
	return true
}
