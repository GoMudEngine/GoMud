package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
	"github.com/GoMudEngine/GoMud/internal/textutil"
	"github.com/GoMudEngine/GoMud/internal/users"
)

//
// Checks for quests on the item
//

func ApplyBuffs(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.Buff)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "Buff", "Actual Type", e.Type())
		return events.Cancel
	}

	//mudlog.Debug(`Event`, `type`, evt.Type(), `UserId`, evt.UserId, `MobInstanceId`, evt.MobInstanceId, `BuffId`, evt.BuffId)

	buffInfo := buffs.GetBuffSpec(evt.BuffId)
	if buffInfo == nil {
		return events.Cancel
	}

	var targetChar *characters.Character

	if evt.MobInstanceId > 0 {

		buffMob := mobs.GetInstance(evt.MobInstanceId)
		if buffMob == nil {
			return events.Cancel
		}

		targetChar = &buffMob.Character

	} else {

		buffUser := users.GetByUserId(evt.UserId)
		if buffUser == nil {
			return events.Cancel
		}

		targetChar = buffUser.Character
	}

	if evt.BuffId < 0 {
		targetChar.RemoveBuff(buffInfo.BuffId * -1)
		return events.Continue
	}

	// Snapshot whether the buff was already active BEFORE we add/refresh.
	// Used below to suppress start text on a pure refresh — refreshing an
	// already-active buff (e.g. ambusher's mob_idle → add_buff 9 tick)
	// shouldn't re-fire "{source} disappears into the shadows." every round.
	wasAlreadyActive := targetChar.HasBuff(evt.BuffId)

	// Apply the buff
	targetChar.AddBuff(evt.BuffId, false)

	//
	// Send YAML start text (if defined) — only on first application,
	// not on refresh of an already-active buff.
	//
	if !wasAlreadyActive && (buffInfo.StartUserText != "" || buffInfo.StartRoomText != "") {
		var charName, charPlainName string
		var sendFunc func(string)
		var roomId, excludeId int

		if evt.UserId != 0 {
			if u := users.GetByUserId(evt.UserId); u != nil {
				charName = u.Character.GetCharacterName(true)
				charPlainName = u.Character.GetCharacterName(false)
				roomId = u.Character.RoomId
				excludeId = u.UserId
				sendFunc = func(msg string) { u.SendText(msg) }
			}
		} else if evt.MobInstanceId != 0 {
			if m := mobs.GetInstance(evt.MobInstanceId); m != nil {
				charName = m.Character.GetCharacterName(true)
				charPlainName = m.Character.GetCharacterName(false)
				roomId = m.Character.RoomId
			}
		}

		if charName != "" {
			tCtx := textutil.TokenContext{
				SourceName:      charName,
				SourcePlainName: charPlainName,
			}
			cfg := textutil.SendTextConfig{
				UserSendFunc: sendFunc,
				RoomSendFunc: func(msg string, skip ...int) {
					if r := rooms.LoadRoom(roomId); r != nil {
						r.SendText(msg, skip...)
					}
				},
				ExcludeId: excludeId,
			}
			textutil.SendPhaseText(buffInfo.StartUserText, buffInfo.StartRoomText, tCtx, "cyan", cfg)
		}
	}

	// Remove buffs listed in start_remove_buffs (cure effects)
	if buffSpec := buffs.GetBuffSpec(evt.BuffId); buffSpec != nil && len(buffSpec.StartRemoveBuffs) > 0 {
		for _, removeId := range buffSpec.StartRemoveBuffs {
			targetChar.RemoveBuff(removeId)
		}
	}

	targetChar.TrackBuffStarted(evt.BuffId)

	//
	// If the buff calls for an immediate triggering
	//
	if buffInfo.TriggerNow {

		if evt.MobInstanceId > 0 && targetChar.Health <= 0 && targetChar.IsAlive() {
			// Buff-tick death (e.g. a DoT buff that fires TriggerNow):
			// route through the Life machine for same-tick observer firing
			// rather than queuing a suicide command for next tick.
			targetChar.Die(state.ActorRef{}, life.TriggerHealthZero)
		}

	}

	events.AddToQueue(events.BuffsTriggered{UserId: evt.UserId, MobInstanceId: evt.MobInstanceId, BuffIds: []int{evt.BuffId}})

	return events.Continue
}
