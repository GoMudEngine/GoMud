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

		mob.Character.Charm(playerId, characters.CharmPermanent, characters.CharmExpiredRevert)

		if charmedUser := users.GetByUserId(playerId); charmedUser != nil {
			charmedUser.Character.TrackCharmed(mob.InstanceId, true)
		}

		sendRoomText(room,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> looks very friendly.`, mob.Character.Name))

	}

	return true, nil
}
