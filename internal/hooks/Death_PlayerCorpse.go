package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// wirePlayerCorpse subscribes to player Life Alive→Dead transitions
// and creates a player corpse in the death room when
// Death.CorpsesEnabled is set in config.
//
// The corpse is created BEFORE the Respawn_PlayerTeleport observer
// fires (which moves the player out of the room), so it is left
// behind in the correct pre-respawn location.
//
// Migrated from internal/usercommands/suicide.go lines 278-284
// as part of chunk-2 Task 9.
func wirePlayerCorpse(c *characters.Character) {
	c.Life.Inner().AfterTransition("player_corpse",
		func(from, to life.State, r state.TransitionReason) {
			if from != life.Alive || to != life.Dead {
				return
			}
			// Only fire for player characters.
			if c.GetUserId() == 0 {
				return
			}

			config := configs.GetGamePlayConfig()
			if !config.Death.CorpsesEnabled {
				return
			}

			u := users.GetByUserId(c.GetUserId())
			if u == nil {
				return
			}
			room := rooms.LoadRoom(c.RoomId)
			if room == nil {
				return
			}
			room.AddCorpse(rooms.Corpse{
				UserId:       u.UserId,
				Character:    *u.Character,
				RoundCreated: util.GetRoundCount(),
			})
		})
}

func init() {
	characters.OnCharacterCreated(wirePlayerCorpse)
}
