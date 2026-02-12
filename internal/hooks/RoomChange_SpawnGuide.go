package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

//
// RoomChangeHandler waits for RoomChange events
// Also sends music changes out
//

const guideMobId = 38

func SpawnGuide(e events.Event) events.ListenerReturn {

	// DOGMud: Tutorial bypassed — new characters spawn directly in Town Square
	return events.Continue

	evt := e.(events.RoomChange)

	// If this isn't a user changing rooms, just pass it along.
	if evt.UserId == 0 {
		return events.Continue
	}

	if evt.ToRoomId < 1 {
		return events.Continue
	}

	user := users.GetByUserId(evt.UserId)
	if user.Character.Level > 5 {
		return events.Continue
	}

	fromRoomOriginal := rooms.GetOriginalRoom(evt.FromRoomId)
	if fromRoomOriginal >= 900 && fromRoomOriginal <= 999 {
		return events.Continue
	}

	toRoomOriginal := rooms.GetOriginalRoom(evt.ToRoomId)
	if toRoomOriginal >= 900 && toRoomOriginal <= 999 {
		return events.Continue
	}

	roundNow := util.GetRoundCount()

	var lastGuideRound uint64 = 0
	tmpLGR := user.GetTempData(`lastGuideRound`)
	if tmpLGRUint, ok := tmpLGR.(uint64); ok {
		lastGuideRound = tmpLGRUint
	}

	if (roundNow - lastGuideRound) < uint64(configs.GetTimingConfig().SecondsToRounds(300)) {
		return events.Continue
	}

	for _, miid := range user.Character.GetCharmIds() {
		if testMob := mobs.GetInstance(miid); testMob != nil && testMob.MobId == guideMobId {
			return events.Continue // already have the mob, we can skip this.
		}
	}

	// Get the new room
	room := rooms.LoadRoom(evt.ToRoomId)

	// Create the mob
	guideMob := mobs.NewMobById(guideMobId, 1)

	// Give them a clearly identifying (however long) name
	guideMob.Character.Name = fmt.Sprintf(`%s's Guide`, user.Character.Name)

	// Add the guide to the room
	room.AddMob(guideMob.InstanceId)

	// Charm the mob
	guideMob.Character.Charm(evt.UserId, characters.CharmPermanent, characters.CharmExpiredDespawn)

	// Track it
	user.Character.TrackCharmed(guideMob.InstanceId, true)

	room.SendText(`<ansi fg="mobname">` + guideMob.Character.Name + `</ansi> appears in a shower of sparks!`)

	guideMob.Command(`sayto ` + user.ShorthandId() + ` I'll be here to help protect you while you learn the ropes.`)
	guideMob.Command(`sayto ` + user.ShorthandId() + ` I can create a portal to take us back to Town Square any time. Just <ansi fg="command">ask</ansi> me about it.`)

	user.SendText(`<ansi fg="alert-3">Your guide will try and stick around until you reach level 5.</ansi>`)

	user.SetTempData(`lastGuideRound`, roundNow)

	return events.Continue
}
