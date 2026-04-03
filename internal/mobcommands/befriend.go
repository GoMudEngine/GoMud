package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Befriend(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	if rest == `revert` {

		if mob.Character.IsCharmed() {

			if charmedUserId := mob.Character.RemoveCharm(); charmedUserId > 0 {
				if charmedUser := users.GetByUserId(charmedUserId); charmedUser != nil {
					charmedUser.Character.TrackCharmed(mob.InstanceId, false)
				}
			}

		}

		return true, nil
	}

	playerId, _ := room.FindByName(rest)

	if playerId > 0 {

		// Anti-recursion: strip any companions the mob itself had before
		// charming it, so we never create companion chains.
		for _, subId := range mob.Character.GetCharmIds() {
			if subMob := mobs.GetInstance(subId); subMob != nil {
				subMob.Character.RemoveCharm()
				if subRoom := rooms.LoadRoom(subMob.Character.RoomId); subRoom != nil {
					subRoom.RemoveMob(subId)
				}
				mobs.DestroyInstance(subId)
			}
		}
		mob.Character.CharmedMobs = nil

		mob.Character.Charm(playerId, characters.CharmPermanent, characters.CharmExpiredRevert)

		if charmedUser := users.GetByUserId(playerId); charmedUser != nil {
			charmedUser.Character.TrackCharmed(mob.InstanceId, true)
		}

		sendRoomText(room,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> looks very friendly.`, mob.Character.Name))

	}

	return true, nil
}
