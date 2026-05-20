package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
	"github.com/GoMudEngine/GoMud/internal/state/position"
)

// wirePositionCrossMachineCascades registers Position-side observers
// for cross-machine transitions. 4a wires only the Life Dead →
// Standing cascade; 4b adds the per-round control-roll subscriber,
// and 4d may add Activity-side hooks if submissions touch Position.
//
// The Life Dead cascade COEXISTS with chunk-2 Life_Cascades.go
// pre-wire (which still resets c.CombatPosition = PositionStanding
// directly + clears GrappleControllerId). Both observers fire on
// death. Both reach Standing (the chunk-2 pre-wire on the legacy
// field; this observer on the new FSM). No drift is possible
// because the new FSM defaults to Standing and 4a has no writers.
// 4b removes the chunk-2 pre-wire once command sites cut over.
//
// Chunk-4b grappled-death fix (2026-05-16): if the deceased is in
// any grapple state with a resolvable partner, use TransitionPair
// to break both sides atomically. This avoids leaving the surviving
// partner stuck in Clinch/Mount/etc. with a stale Partner ref,
// which the Position consistency checker would otherwise force-
// break in the next tick (visible to players as a double "The
// grapple suddenly breaks apart." message). Falls back to the
// solo TransitionToStanding when there's no partner (e.g. solo
// Turtle) or the partner can't be resolved.
func wirePositionCrossMachineCascades(c *characters.Character) {
	c.Life.Inner().AfterTransition("position_life_dead",
		func(from, to life.State, r state.TransitionReason) {
			if from != life.Alive || to != life.Dead {
				return
			}
			if c.Position == nil || c.Position.IsStanding() {
				return
			}
			reason := state.TransitionReason{
				Trigger: position.TriggerDeath,
				Actor:   c.Position.Self(),
			}
			// Grappled death: break the pair atomically so the
			// surviving partner doesn't end up in a state-mismatched
			// pair waiting for the consistency checker.
			if c.Position.IsGrappling() {
				if partner := resolvePartner(c); partner != nil &&
					partner.Position != nil && partner.Position.IsGrappling() {
					if err := position.TransitionPair(
						c, partner, position.Standing, reason,
					); err == nil {
						return
					}
					// TransitionPair failed (rare — e.g. partner
					// already drifted out). Fall through to the
					// solo transition below; consistency checker
					// will catch any residual mismatch.
				}
			}
			_ = c.Position.TransitionToStanding(reason)
		})
}

func init() {
	characters.OnCharacterCreated(wirePositionCrossMachineCascades)
}
