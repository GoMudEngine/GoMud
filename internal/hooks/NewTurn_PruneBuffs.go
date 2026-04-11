package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/textutil"
	"github.com/GoMudEngine/GoMud/internal/users"
)

//
// Prune all buffs that have expired.
//

func PruneBuffs(e events.Event) events.ListenerReturn {

	/*
		evt, typeOk := e.(events.NewTurn)
		if !typeOk {
			mudlog.Error("Event", "Expected Type", "NewTurn", "Actual Type", e.Type())
			return events.Cancel
		}
	*/

	roomsWithPlayers := rooms.GetRoomsWithPlayers()
	for _, roomId := range roomsWithPlayers {
		// Get rooom
		if room := rooms.LoadRoom(roomId); room != nil {

			// Handle outstanding player buffs
			logOff := false
			for _, uId := range room.GetPlayers(rooms.FindBuffed) {

				user := users.GetByUserId(uId)

				logOff = false
				if buffsToPrune := user.Character.Buffs.Prune(); len(buffsToPrune) > 0 {
					for _, buffInfo := range buffsToPrune {
						// Send YAML end text (if defined).
						endBuffSpec := buffs.GetBuffSpec(buffInfo.BuffId)
						if endBuffSpec != nil && (endBuffSpec.EndUserText != "" || endBuffSpec.EndRoomText != "") {
							tCtx := textutil.TokenContext{
								SourceName:      user.Character.GetCharacterName(true),
								SourcePlainName: user.Character.GetCharacterName(false),
							}
							cfg := textutil.SendTextConfig{
								UserSendFunc: func(msg string) { user.SendText(msg) },
								RoomSendFunc: func(msg string, skip ...int) {
									if r := rooms.LoadRoom(user.Character.RoomId); r != nil {
										r.SendText(msg, skip...)
									}
								},
								ExcludeId: user.UserId,
							}
							textutil.SendPhaseText(endBuffSpec.EndUserText, endBuffSpec.EndRoomText, tCtx, "cyan", cfg)
						}

						scripting.TryBuffScriptEvent(`onEnd`, uId, 0, buffInfo.BuffId)

						if buffInfo.BuffId == 0 { // Log them out // logoff // logout
							if !user.Character.HasAdjective(`zombie`) { // if they are currently a zombie, we don't log them out from this buff being removed
								logOff = true
							}
						}
					}

					user.Character.Validate()

					if logOff {
						mudlog.Info("MEDITATION LOGOFF")
						events.AddToQueue(events.System{Command: "logoff", Data: uId})
					}
				}

			}
		}
	}

	// Handle outstanding mob buffs
	for _, mobInstanceId := range mobs.GetAllMobInstanceIds() {

		mob := mobs.GetInstance(mobInstanceId)

		if buffsToPrune := mob.Character.Buffs.Prune(); len(buffsToPrune) > 0 {
			for _, buffInfo := range buffsToPrune {
				// Send YAML end text (if defined).
				endBuffSpec := buffs.GetBuffSpec(buffInfo.BuffId)
				if endBuffSpec != nil && endBuffSpec.EndRoomText != "" {
					tCtx := textutil.TokenContext{
						SourceName:      mob.Character.GetCharacterName(true),
						SourcePlainName: mob.Character.GetCharacterName(false),
					}
					cfg := textutil.SendTextConfig{
						RoomSendFunc: func(msg string, skip ...int) {
							if r := rooms.LoadRoom(mob.Character.RoomId); r != nil {
								r.SendText(msg, skip...)
							}
						},
					}
					textutil.SendPhaseText("", endBuffSpec.EndRoomText, tCtx, "cyan", cfg)
				}

				scripting.TryBuffScriptEvent(`onEnd`, 0, mobInstanceId, buffInfo.BuffId)
			}

			mob.Character.Validate()
		}

	}

	return events.Continue

}
