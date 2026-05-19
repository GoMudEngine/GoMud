// Round ticks for players
package hooks

import (
	"strconv"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state/presence"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

//
// Handle mobs that are bored
//

func IdleMobs(e events.Event) events.ListenerReturn {

	mc := configs.GetMemoryConfig()

	allMobInstances := mobs.GetAllMobInstanceIds()

	allowedUnloadCt := len(allMobInstances) - int(mc.MobUnloadThreshold)
	if allowedUnloadCt < 0 {
		allowedUnloadCt = 0
	}

	// Handle idle mob behavior
	tStart := time.Now()
	for _, mobId := range allMobInstances {

		mob := mobs.GetInstance(mobId)
		if mob == nil {
			allowedUnloadCt--
			continue
		}

		// Chunk 5 (Presence): despawn driven by Presence state, not BoredomCounter.
		// PresenceTick (T4) has already transitioned the mob to Despawning;
		// this hook fires the actual removal on the next tick.
		if mob.Character.Presence != nil && mob.Character.Presence.State() == presence.Despawning {
			if allowedUnloadCt > 0 {
				mob.Command(`despawn presence_despawning`)
				allowedUnloadCt--
			}
			continue
		}

		// If they are doing some sort of combat thing,
		// Don't do idle actions
		if mob.Character.IsInCombat() {
			if mob.Character.CurrentCombatTarget().UserId > 0 {
				user := users.GetByUserId(mob.Character.CurrentCombatTarget().UserId)
				if user == nil || user.Character.RoomId != mob.Character.RoomId {
					mob.Command(`emote mumbles about losing their quarry.`)
					mob.Character.EndAggro()
				}
			}
			continue
		}

		if mob.InConversation() {
			mob.Converse()
			continue
		}

		// Check whether they are currently in the middle of a path, or have one waiting to start.
		// This comes after checks for whether they are currently in a conersation, or in combat, etc.
		if currentStep := mob.Path.Current(); currentStep != nil || mob.Path.Len() > 0 {

			if currentStep != nil {

				// If their currentStep isn't actually the room they are in
				// They've somehow been moved. Reclaculate a new path.
				if currentStep.RoomId() != mob.Character.RoomId {

					reDoWaypoints := mob.Path.Waypoints()
					if len(reDoWaypoints) > 0 {
						newCommand := `pathto`
						for _, wpInt := range reDoWaypoints {
							newCommand += ` ` + strconv.Itoa(wpInt)
						}
						mob.Command(newCommand)
						continue
					}

					// if we were unable to come up with a new path, send them home.
					mob.Command(`pathto home`)

					continue
				}
			}

			if nextStep := mob.Path.Next(); nextStep != nil {

				if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
					if exitInfo, ok := room.Exits[nextStep.ExitName()]; ok {
						if exitInfo.RoomId == nextStep.RoomId() {
							mob.Command(nextStep.ExitName())
							// Stage 2 caravan: pace caravan crews a shade
							// slower than default mob walking. The noop
							// pushes lastCommandTurn forward so the next
							// path step waits ~1.5s. ~1 step per 5.5s real
							// instead of per 4s — visible in flavor without
							// being painful.
							for _, g := range mob.Groups {
								if g == "caravan" {
									mob.Command("noop", 1.5)
									break
								}
							}
							continue
						}
					}
				}

			}

			mob.Path.Clear()

			if mob.HomeRoomId == mob.Character.RoomId {
				mob.WanderCount = 0
			}

		}

		events.AddToQueue(events.MobIdle{MobInstanceId: mobId})

	}

	util.TrackTime(`IdleMobs()`, time.Since(tStart).Seconds())

	return events.Continue
}
