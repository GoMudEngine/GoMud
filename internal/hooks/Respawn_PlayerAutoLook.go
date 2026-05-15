package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// wireRespawnAutoLook subscribes to player Life Respawning→Alive
// transitions and queues a `look` command so the new room description
// renders without the player typing it manually.
//
// Note: a parallel auto-look fix for fold-recall is logged as
// followup memory `auto-look-after-room-change`.
func wireRespawnAutoLook(c *characters.Character) {
	c.Life.Inner().AfterTransition("respawn_auto_look",
		func(from, to life.State, r state.TransitionReason) {
			if from != life.Respawning || to != life.Alive {
				return
			}
			// Only fire for player characters.
			if c.GetUserId() == 0 {
				return
			}
			u := users.GetByUserId(c.GetUserId())
			if u == nil {
				return
			}
			u.Command("look")
		})
}

func init() {
	characters.OnCharacterCreated(wireRespawnAutoLook)
}
