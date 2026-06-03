package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/justice"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
)

// wirePlayerDeathJailCleanup subscribes to player Life Alive→Dead
// transitions and tears down any active instanced jail cell, ending
// the detention so the player respawns free with no orphaned instance.
// Per the cell zone's death_policy (ejected), dying voids the remaining
// sentence. Crimes are NOT cleared — death is not absolution.
// Mirrors wirePlayerDeathBountyResolve in PlayerDeath_BountyResolve.go.
func wirePlayerDeathJailCleanup(c *characters.Character) {
	c.Life.Inner().AfterTransition("player_death_jail_cleanup",
		func(from, to life.State, r state.TransitionReason) {
			if from != life.Alive || to != life.Dead {
				return
			}
			// Only fire for player characters.
			if c.GetUserId() == 0 {
				return
			}
			justice.HandleJailedDeath(c)
		})
}

func init() {
	characters.OnCharacterCreated(wirePlayerDeathJailCleanup)
}
