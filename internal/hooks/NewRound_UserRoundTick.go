// Round ticks for players
package hooks

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

//
// Player Round Tick
//

func UserRoundTick(e events.Event) events.ListenerReturn {

	evt := e.(events.NewRound)

	roomsWithPlayers := rooms.GetRoomsWithPlayers()
	for _, roomId := range roomsWithPlayers {
		// Get rooom
		if room := rooms.LoadRoom(roomId); room != nil {
			room.RoundTick()

			allowIdleMessages := true
			if handled, err := scripting.TryRoomIdleEvent(roomId); err == nil {
				if handled { // For this event, handled represents whether to reject the move.
					allowIdleMessages = false
				}
			}

			if allowIdleMessages {

				chanceIn100 := 5
				if room.RoomId == -1 {
					chanceIn100 = 20
				}

				var idleMsgs []string

				if len(room.IdleMessages) > 0 {
					idleMsgs = room.IdleMessages
				} else {
					if zCfg := rooms.GetZoneConfig(room.Zone); zCfg != nil {
						if len(zCfg.IdleMessages) > 0 {
							idleMsgs = zCfg.IdleMessages
						}
					}
				}

				idleMsgCt := len(idleMsgs)
				if idleMsgCt > 0 && util.Rand(100) < chanceIn100 {

					if targetRoomId, err := strconv.Atoi(idleMsgs[0]); err == nil {
						idleMsgCt = 0
						if tgtRoom := rooms.LoadRoom(targetRoomId); tgtRoom != nil {
							idleMsgs = tgtRoom.IdleMessages
							idleMsgCt = len(idleMsgs)
						}
					}

					if idleMsgCt > 0 {
						// pick a random message
						idleMsgIndex := uint8(util.Rand(idleMsgCt))

						// If it's a repeating message, treat it as a non-message
						// (Unless it's the only one)
						if idleMsgIndex != room.LastIdleMessage || idleMsgCt == 1 {

							room.LastIdleMessage = idleMsgIndex

							msg := idleMsgs[idleMsgIndex]
							if msg != `` {
								room.SendText(msg)
							}

						}
					}

				}
			}

			for _, uId := range room.GetPlayers() {

				user := users.GetByUserId(uId)
				if user == nil {
					continue
				}

				if user.Character.HasAdjective(`zombie`) {
					user.Command(`zombieact`)
				}

				// Roundtick any cooldowns
				user.Character.Cooldowns.RoundTick()

				// Stage 7.5: Attempt automatic recovery from prone (uses DEX)
				if attemptMade, success := user.Character.AttemptRecovery(user.Character.Stats.Dexterity.ValueAdj); attemptMade {
					if success {
						user.SendText("You scramble to your feet!")
						if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
							room.SendText("<ansi fg=\"username\">"+user.Character.Name+"</ansi> clambers to their feet in a rushed panic.", user.UserId)
						}
					} else {
						user.SendText("You attempt to stand, but slip back down in the chaos of battle!")
						if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
							room.SendText("<ansi fg=\"username\">"+user.Character.Name+"</ansi> attempts to stand, but slips and falls in the chaos of battle.", user.UserId)
						}
					}
				}

				if user.Character.Charmed != nil && user.Character.Charmed.RoundsRemaining > 0 {
					user.Character.Charmed.RoundsRemaining--
				}

				if triggeredBuffs := user.Character.Buffs.Trigger(); len(triggeredBuffs) > 0 {

					//
					// Fire onTrigger for buff script
					//
					triggeredBuffIds := []int{}
					for _, buff := range triggeredBuffs {

						if buff.Expired() {
							triggeredBuffIds = append(triggeredBuffIds, buff.BuffId)
							continue
						}

						_, err := scripting.TryBuffScriptEvent(`onTrigger`, uId, 0, buff.BuffId)

						if buff.TriggersLeft != buffs.TriggersLeftUnlimited || err != scripting.ErrEventNotFound {
							triggeredBuffIds = append(triggeredBuffIds, buff.BuffId)
						}

					}

					events.AddToQueue(events.BuffsTriggered{UserId: user.UserId, BuffIds: triggeredBuffIds})
				}

				// Stage 7.5: Clear recovery penalty flag at end of round
				user.Character.RecoveryPenaltyThisRound = false

				// Stage 8.6: Clear defense penalty flag at end of round
				user.Character.DefensePenaltyNextRound = false

				// Recalculate all stats at the end of the round tick
				user.Character.Validate()

				// Only do this every 15 rounds to keep spam down.
				if evt.RoundNumber%15 == 0 {

					if !user.DidTip(`status train`) && user.Character.StatPoints > 0 {
						user.SendText(`<ansi fg="alert-5">TIP:</ansi> <ansi fg="tip-text">Type <ansi fg="command">status train</ansi> to use the status points you've earned through leveling.</ansi>`)
						user.SendText(``)
					}

				}

			}

		}

	}

	return events.Continue
}
