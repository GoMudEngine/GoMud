package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// wireRespawnTeleport subscribes to player Life Dead→Respawning
// transitions and teleports the player to their configured home /
// graveyard room. Mob actors are skipped (player-only observer).
//
// Chunk-2 Task 9 will thin suicide.go so this observer becomes
// the authoritative teleport path. For now it is wired but
// dormant until TransitionToRespawning is called.
func wireRespawnTeleport(c *characters.Character) {
	c.Life.Inner().AfterTransition("respawn_teleport",
		func(from, to life.State, r state.TransitionReason) {
			if from != life.Dead || to != life.Respawning {
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

			rooms.MoveToRoom(u.UserId, c.ResolveRespawnRoom())
		})
}

func init() {
	characters.OnCharacterCreated(wireRespawnTeleport)
}
