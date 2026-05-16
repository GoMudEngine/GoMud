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
func wirePositionCrossMachineCascades(c *characters.Character) {
	c.Life.Inner().AfterTransition("position_life_dead",
		func(from, to life.State, r state.TransitionReason) {
			if from != life.Alive || to != life.Dead {
				return
			}
			if c.Position == nil || c.Position.IsStanding() {
				return
			}
			_ = c.Position.TransitionToStanding(state.TransitionReason{
				Trigger: position.TriggerDeath,
				Actor:   c.Position.Self(),
			})
		})
}

func init() {
	characters.OnCharacterCreated(wirePositionCrossMachineCascades)
}
