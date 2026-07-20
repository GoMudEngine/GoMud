package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/messaging"
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

	target, err := actions.ResolveTargetActor(room, rest)
	if err != nil || !target.IsPlayer() {
		return true, nil
	}
	charmedUser := target.(*actions.UserActor).User
	playerId := charmedUser.UserId

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
	charmedUser.Character.TrackCharmed(mob.InstanceId, true)

	room.SendTextVisual(messaging.CategoryMobEmote,
		fmt.Sprintf(`<ansi fg="mobname">%s</ansi> looks very friendly.`, mob.Character.Name))

	return true, nil
}
