package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/events"
)

// Register hooks here...
func RegisterListeners() {

	// Buffs
	events.RegisterListener(events.Buff{}, ApplyBuffs)

	// RoomChange Listeners
	events.RegisterListener(events.RoomChange{}, LocationMusicChange)
	events.RegisterListener(events.RoomChange{}, CleanupEphemeralRooms)
	events.RegisterListener(events.RoomChange{}, SpawnGuide)

	// NewRound Listeners
	events.RegisterListener(events.NewRound{}, PruneVMs)
	events.RegisterListener(events.NewRound{}, InactivePlayers)
	events.RegisterListener(events.NewRound{}, UpdateZoneMutators)
	events.RegisterListener(events.NewRound{}, CheckNewDay)
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
	events.RegisterListener(events.NewRound{}, AutoHeal)
	events.RegisterListener(events.NewRound{}, IdleMobs)
	events.RegisterListener(events.MobIdle{}, HandleIdleMobs)

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
	events.RegisterListener(events.PlayerDespawn{}, HandleLeave, events.Last) // This is a final listener, has to happen last

	// Levelup Notifications — disabled in Stage 3.5 (no level-ups occur)
	// Guide dismissal now handled by CheckGuideBySkillRanks() called from combat hooks

	// Day/Night cycle
	events.RegisterListener(events.DayNightCycle{}, NotifySunriseSunset)

	// Looking
	events.RegisterListener(events.Looking{}, HandleLookHints)

	// Messages
	events.RegisterListener(events.Message{}, Message_SendMessage)
	// Prompt
	events.RegisterListener(events.RedrawPrompt{}, RedrawPrompt_SendRedraw)

	// User Settings change
	events.RegisterListener(events.UserSettingChanged{}, ClearSettingCaches)

	events.RegisterListener(events.PlayerDrop{}, HandlePlayerDrop)
	events.RegisterListener(events.WebClientCommand{}, WebClientCommand_SendWebClientCommand)

	events.RegisterListener(events.CharacterCreated{}, BroadcastNewChar)
	events.RegisterListener(events.CharacterChanged{}, BroadcastNewChar)

	events.RegisterListener(events.Broadcast{}, Broadcast_SendToAll)

	events.RegisterListener(events.RebuildMap{}, HandleMapRebuild)

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
