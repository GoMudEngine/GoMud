package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
	"github.com/GoMudEngine/GoMud/internal/state/life"
)

// wireActivityCrossMachineCascades subscribes the Activity machine
// to the other machines that drive its interrupt rules:
//   - Life: Alive → Dead → Activity → Free (covers what the chunk-2
//     pre-wire in Life_Cascades.go used to do directly).
//
// Note: Crafting/Salvaging are interrupted by combat via the
// activity_self veto in CombatPhase_Vetoes.go — the veto prevents
// TransitionToEngaging from succeeding while any Activity is active.
// A separate AfterTransition cascade for combat-entry is therefore
// unreachable and has been removed.
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
}

func init() {
	characters.OnCharacterCreated(wireActivityCrossMachineCascades)
}
