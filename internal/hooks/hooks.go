package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/caravan"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/forager"
)

// Register hooks here...
func RegisterListeners() {

	// Buffs
	events.RegisterListener(events.Buff{}, ApplyBuffs)

	// RoomChange Listeners
	events.RegisterListener(events.RoomChange{}, LocationMusicChange)
	events.RegisterListener(events.RoomChange{}, CleanupEphemeralRooms)
	events.RegisterListener(events.RoomChange{}, MobRoomChangeKnowledgeObservers)
	events.RegisterListener(events.RoomChange{}, MobRoomChangeFactsAutoWithdraw)
	events.RegisterListener(events.RoomChange{}, MobRoomChangeShadowFollow)
	events.RegisterListener(events.RoomChange{}, PresencePlayerEntry)

	// NewRound Listeners
	events.RegisterListener(events.NewRound{}, InactivePlayers)
	events.RegisterListener(events.NewRound{}, UpdateZoneMutators)
	events.RegisterListener(events.NewRound{}, CheckNewDay)
	events.RegisterListener(events.NewRound{}, CheckMoonPhase)
	events.RegisterListener(events.NewRound{}, CheckStorageFees)
	events.RegisterListener(events.NewRound{}, SpawnLootGoblin)
	events.RegisterListener(events.NewRound{}, UserRoundTick)
	events.RegisterListener(events.NewRound{}, MobRoundTick)
	events.RegisterListener(events.NewRound{}, HandleRespawns)
	//
	// Combat goes here
	//
	events.RegisterListener(events.NewRound{}, DoCombat)
	//
	// Done with combat
	//
	events.RegisterListener(events.NewRound{}, PresenceTick)
	events.RegisterListener(events.NewRound{}, AutoHeal)
	events.RegisterListener(events.NewRound{}, BloomTick) // Bloom drug: Crash, Withdrawal, decay
	events.RegisterListener(events.NewRound{}, BroadcastHints)
	events.RegisterListener(events.NewRound{}, IdleMobs)
	events.RegisterListener(events.MobIdle{}, HandleIdleMobs)
	events.RegisterListener(events.NewRound{}, FerryTick)     // Ferry vessels: schedule reconcile
	events.RegisterListener(events.NewRound{}, WarehouseTick) // Warehouse pools: accrual + dirty save

	// Turn Hooks
	events.RegisterListener(events.NewTurn{}, CleanupZombies)
	events.RegisterListener(events.NewTurn{}, AutoSave)
	events.RegisterListener(events.NewTurn{}, PruneBuffs)
	events.RegisterListener(events.NewTurn{}, ActionPoints)

	// ItemOwnership
	events.RegisterListener(events.ItemOwnership{}, CheckItemQuests)

	// MSP Sound
	events.RegisterListener(events.MSP{}, PlaySound)
	// Quest Events
	events.RegisterListener(events.Quest{}, HandleQuestUpdate)
	// Spawn events
	events.RegisterListener(events.PlayerSpawn{}, HandleJoin)
	// Player despawn: clear tracking/shadow state pointing at the leaving player
	events.RegisterListener(events.PlayerDespawn{}, PlayerDespawnTrackingCleanup)
	// Player despawn: tear down ephemeral jail cell + preserve sentence record
	events.RegisterListener(events.PlayerDespawn{}, PlayerDespawnJailCleanup)
	events.RegisterListener(events.PlayerDespawn{}, HandleLeave, events.Last) // This is a final listener, has to happen last

	// Day/Night cycle
	events.RegisterListener(events.DayNightCycle{}, NotifySunriseSunset)

	// Moon phase cycle (the three Witnesses)
	events.RegisterListener(events.MoonPhase{}, BroadcastMoonPhase)

	// Looking
	events.RegisterListener(events.Looking{}, HandleLookHints)

	// Messages
	events.RegisterListener(events.Message{}, Message_SendMessage)
	// Prompt
	events.RegisterListener(events.RedrawPrompt{}, RedrawPrompt_SendRedraw)

	// User Settings change
	events.RegisterListener(events.UserSettingChanged{}, ClearSettingCaches)

	events.RegisterListener(events.WebClientCommand{}, WebClientCommand_SendWebClientCommand)

	events.RegisterListener(events.CharacterCreated{}, BroadcastNewChar)
	events.RegisterListener(events.CharacterChanged{}, BroadcastNewChar)

	events.RegisterListener(events.Broadcast{}, Broadcast_SendToAll)

	events.RegisterListener(events.RebuildMap{}, HandleMapRebuild)

	// Mob death: pack flee behavior (Stage 42.7)
	events.RegisterListener(events.MobDeath{}, PackFlee)

	// Mob death: quest engine notifications
	events.RegisterListener(events.MobDeath{}, MobDeathQuestNotify)

	// Mob death: companion cleanup (remove from owner's list + notify)
	events.RegisterListener(events.MobDeath{}, CompanionCleanup)

	// Mob death: faction rep bump for damagers + same-room party members
	events.RegisterListener(events.MobDeath{}, MobDeathFactionRep)

	// Mob death: auto-claim open bounties targeting the dead mob
	events.RegisterListener(events.MobDeath{}, MobDeathBountyClaim)

	// Mob death: clear tracking/shadow state pointing at the dead mob
	events.RegisterListener(events.MobDeath{}, MobDeathTrackingCleanup)

	// Caravan: patrol-waypoint arrival drives vendor restocks + Fernway pickup
	events.RegisterListener(events.PatrolWaypointArrival{}, caravan.CaravanArrivalListener)
	// Caravan: runner-circuit completion returns residual cargo from Lars to wagon
	events.RegisterListener(events.PatrolCompleted{}, caravan.CaravanRunnerCompletionListener)

	// Forager: patrol-waypoint arrival drives per-vendor sell handoff
	events.RegisterListener(events.PatrolWaypointArrival{}, forager.ForagerArrivalListener)
	// Forager: oneshot delivery patrol completion advances forager_state
	events.RegisterListener(events.PatrolCompleted{}, forager.ForagerCompletionListener)

	// Skill use: quest engine notifications
	events.RegisterListener(events.SkillUsed{}, SkillUseQuestNotify)

	// Log tee to users
	events.RegisterListener(events.Log{}, FollowLogs)

	// Listener for debugging some stuff (catches all events)
	/*
		events.RegisterListener(nil, func(e events.Event) events.ListenerReturn {
			t := e.Type()
			if t != `NewTurn` && t != `Message` && t != `NewRound` && t != `Broadcast` {
				mudlog.Info("Event", "e.Type", e.Type(), "e", e)
			}
			return events.Continue
		})
	*/

}
