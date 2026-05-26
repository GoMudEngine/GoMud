// Round ticks for players
package hooks

import (
	"strconv"
	"time"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/conversationadapter"
	"github.com/GoMudEngine/GoMud/internal/conversations"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gametime"
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

		// Chunk 3.2: schedule executor. Runs before the path-walker so it can
		// clear stale paths on segment transitions and queue new pathtos before
		// the walker consumes them.
		if mob.ScheduleId != "" {
			plan := scheduleTickPlan(mob, gametime.GetDate().Hour24)
			applySchedulePlan(mob, plan)
		}

		// Chunk 3.4: patrol executor. Reads active_patrol_id stamped by the
		// schedule branch above (if the current schedule segment has
		// activity: patrol), otherwise falls back to mob.PatrolId for
		// standalone patrols. Reads-and-clears the stamp so it doesn't
		// linger across ticks.
		{
			var activePatrolId string
			if id := getMiscDataString(&mob.Character, "active_patrol_id"); id != "" {
				activePatrolId = id
				mob.Character.SetMiscData("active_patrol_id", "")
			}
			if activePatrolId == "" && mob.PatrolId != "" {
				activePatrolId = mob.PatrolId
			}
			if activePatrolId != "" {
				plan := patrolTickPlan(mob, activePatrolId)
				applyPatrolPlan(mob, plan, activePatrolId)
			}
		}

		// Chunk 3.6: NPC↔NPC idle conversations.
		// Phase 1: if this mob is already in a conversation, advance the
		// state machine one tick (fires the next line or finalises/aborts).
		if partnerId, ok := mob.Character.GetMiscData(conversations.MiscDataPartnerId).(int); ok && partnerId > 0 {
			conversations.TickConversation(conversationadapter.AdaptMob(mob), partnerId)
		}

		// Phase 2: if fully idle and not on cooldown, roll for a new
		// conversation. ConversationBaseChancePct is a float percent
		// (1.0 = 1 %), so multiply by 100 and roll out of 10 000 to
		// preserve fractional precision.
		if conversationsTriggerEligible(mob) {
			cfg := configs.GetBalanceConfig()
			if util.Rand(10000) < int(float64(cfg.ConversationBaseChancePct)*100) {
				if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
					conversations.TryStart(conversationadapter.AdaptMob(mob), room.GetMobs())
				}
			}
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

// conversationsTriggerEligible gates the per-tick conversation trigger for a
// mob. It duplicates the conversations package's isFullyIdle logic so the hot
// path doesn't allocate an adapter just to peek at eligibility.
func conversationsTriggerEligible(mob *mobs.Mob) bool {
	if mob == nil {
		return false
	}
	// Already in a conversation.
	if partnerId, ok := mob.Character.GetMiscData(conversations.MiscDataPartnerId).(int); ok && partnerId > 0 {
		return false
	}
	// On cooldown from a recently completed conversation.
	if until, ok := mob.Character.GetMiscData(conversations.MiscDataCooldownUntilRound).(uint64); ok && uint64(util.GetRoundCount()) < until {
		return false
	}
	// Combat or pending aggro.
	if mob.Character.Aggro != nil || mob.Character.IsInCombat() {
		return false
	}
	// Sleeping mob shouldn't start chatting.
	if mob.Character.HasBuffFlag(buffs.Sleeping) {
		return false
	}
	// Mid-walk on a path — let the mob arrive before striking up a chat.
	if mob.Path.Len() > 0 || mob.Path.Current() != nil {
		return false
	}
	return true
}
