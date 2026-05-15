package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
	"github.com/GoMudEngine/GoMud/internal/state/combatphase"
	"github.com/GoMudEngine/GoMud/internal/state/life"
)

// wireActivityCrossMachineCascades subscribes the Activity machine
// to the other machines that drive its interrupt rules:
//   - Life: Alive → Dead → Activity → Free (covers what the chunk-2
//     pre-wire in Life_Cascades.go used to do directly).
//   - Combat Phase: Idle → Engaging (self-initiated) → cancel only
//     if current Activity is Crafting or Salvaging. Casting is
//     exempt — casting IS a combat action.
//
// Movement cancel and damage cancel for Crafting/Salvaging are not
// machine-to-machine transitions and don't fit AfterTransition;
// they're wired at the call sites (go.go for movement in Task 7,
// damage application path in Task 10).
func wireActivityCrossMachineCascades(c *characters.Character) {
	// Life: Alive → Dead → cancel any active Activity.
	c.Life.Inner().AfterTransition("activity_life_dead",
		func(from, to life.State, r state.TransitionReason) {
			if from != life.Alive || to != life.Dead {
				return
			}
			if c.Activity == nil || c.Activity.IsFree() {
				return
			}
			_ = c.Activity.TransitionToFree(state.TransitionReason{
				Trigger: activity.TriggerDeath,
				Actor:   c.Activity.Self(),
			})
		})

	// Combat Phase: Idle → Engaging (self-initiated) → cancel
	// Crafting/Salvaging. Casting is exempt.
	c.CombatPhase.Inner().AfterTransition("activity_combat_entry",
		func(from, to combatphase.State, r state.TransitionReason) {
			if from != combatphase.Idle || to != combatphase.Engaging {
				return
			}
			if c.Activity == nil {
				return
			}
			switch c.Activity.State() {
			case activity.Crafting, activity.Salvaging:
				_ = c.Activity.TransitionToFree(state.TransitionReason{
					Trigger: activity.TriggerCombatInterrupt,
					Actor:   c.Activity.Self(),
				})
			}
		})
}

func init() {
	characters.OnCharacterCreated(wireActivityCrossMachineCascades)
}
