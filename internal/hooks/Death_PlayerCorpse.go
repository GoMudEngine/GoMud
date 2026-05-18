package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
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

			d, _ := c.Life.DeadData()

			// chunk 4d: partial gold transfer — skip full corpse creation
			// when GoldLossFraction > 0 (subdue/cripple submission outcomes).
			// The victim wakes at the temple poorer but without a lootable corpse.
			if d.GoldLossFraction > 0 && !d.Killer.IsZero() {
				transferPartialGold(c, d.Killer, d.GoldLossFraction)
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

// transferPartialGold moves a fraction of the dying character's gold
// to the killer (player or mob). Used by chunk-4d subdue and cripple
// outcomes that intentionally leave the victim alive but poorer.
// No corpse is created in the calling branch.
func transferPartialGold(victim *characters.Character, killerRef state.ActorRef, fraction float64) {
	if fraction <= 0 || fraction > 1.0 || victim.Gold <= 0 {
		return
	}
	loss := int(float64(victim.Gold) * fraction)
	if loss < 1 {
		loss = 1
	}
	victim.Gold -= loss
	switch {
	case killerRef.UserId > 0:
		if u := users.GetByUserId(killerRef.UserId); u != nil {
			u.Character.Gold += loss
		}
	case killerRef.MobInstanceId > 0:
		if m := mobs.GetInstance(killerRef.MobInstanceId); m != nil {
			m.Character.Gold += loss
		}
	}
}

func init() {
	characters.OnCharacterCreated(wirePlayerCorpse)
}
