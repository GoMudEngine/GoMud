package actions

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
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
	spellId := ""
	if d, ok := a.CastingData(); ok {
		spellId = d.SpellId
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
	if uid := target.GetUserId(); uid > 0 {
		events.AddToQueue(events.CastInterrupted{UserId: uid, SpellId: spellId})
	}
	return true
}
