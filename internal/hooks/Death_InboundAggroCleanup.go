package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
)

// wireInboundAggroCleanup subscribes to Life Alive→Dead transitions
// for both player and mob actors. For each dying actor, it walks the
// room and calls EndAggro on any mob that is targeting the dying
// character. For player deaths, it also clears aggro on companions.
//
// This prevents combat round processing from trying to hit or cast
// at an actor that is no longer in a valid combat state.
//
// Migrated from internal/usercommands/suicide.go lines 197-224
// as part of chunk-2 Task 9.
func wireInboundAggroCleanup(c *characters.Character) {
	c.Life.Inner().AfterTransition("inbound_aggro_cleanup",
		func(from, to life.State, r state.TransitionReason) {
			if from != life.Alive || to != life.Dead {
				return
			}

			room := rooms.LoadRoom(c.RoomId)
			if room == nil {
				return
			}

			isPlayer := c.GetUserId() != 0

			// Walk every fighting mob in the room and clear aggro that
			// targets the dying actor.
			for _, mobInstId := range room.GetMobs(rooms.FindFighting) {
				mob := mobs.GetInstance(mobInstId)
				if mob == nil || !mob.Character.IsInCombat() {
					continue
				}

				if isPlayer {
					// Standard aggro targeting this player (UserId shape).
					if mob.Character.EngagedTarget().UserId == c.GetUserId() {
						mob.Character.EndAggro()
						continue
					}
					// Spell-cast-shape aggro: abort in-flight cast targeting
					// this player.
					if mob.Character.Aggro != nil && mob.Character.Aggro.Type == characters.SpellCast {
						for _, tid := range mob.Character.Aggro.SpellInfo.TargetUserIds {
							if tid == c.GetUserId() {
								mob.Character.EndAggro()
								break
							}
						}
					}
				} else {
					// Mob death: clear any mob targeting the dying mob
					// instance by MobInstanceId.
					if mob.Character.EngagedTarget().MobInstanceId == c.MobInstanceId {
						mob.Character.EndAggro()
					}
				}
			}

			// For player deaths, also clear aggro on the player's
			// companions so they arrive at the respawn room clean.
			if isPlayer {
				for _, compInstId := range c.GetCharmIds() {
					comp := mobs.GetInstance(compInstId)
					if comp == nil {
						continue
					}
					comp.Character.EndAggro()
				}
			}
		})
}

func init() {
	characters.OnCharacterCreated(wireInboundAggroCleanup)
}
