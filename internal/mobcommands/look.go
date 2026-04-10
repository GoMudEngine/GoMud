package mobcommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Look(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	secretLook := false
	if strings.HasPrefix(rest, "secretly") {
		secretLook = true
		rest = strings.TrimSpace(strings.TrimPrefix(rest, "secretly"))
	}

	isSneaking := mob.Character.HasBuffFlag(buffs.Hidden)

	// trim off some fluff
	if len(rest) > 2 {
		if rest[0:3] == `at ` {
			rest = rest[3:]
		}
	}
	if len(rest) > 3 {
		if rest[0:4] == `the ` {
			rest = rest[4:]
		}
	}

	lookAt := rest

	if len(lookAt) == 0 {

		if !secretLook && !isSneaking {
			room.SendText(
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> is looking around.`, mob.Character.Name),
			)

			// Make it a "secret looks" now because we don't want another look message sent out by the lookRoom() func
			secretLook = true
		}
		lookRoom(mob, room.RoomId, secretLook || isSneaking)

		return true, nil

	}

	//
	// Check room exits
	//
	exitName, lookRoomId := room.FindExitByName(lookAt)
	if exitName != `` {

		exitInfo, _ := room.GetExitInfo(exitName)
		if exitInfo.Lock.IsLocked() {
			return true, nil
		}

		if !isSneaking {
			room.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> peers toward the %s.`, mob.Character.Name, exitName))
		}

		lookRoom(mob, lookRoomId, secretLook || isSneaking)

		return true, nil
	}

	//
	// Check for anything in their backpack they might want to look at
	//
	if lookItem, found := mob.Character.FindInBackpack(lookAt); found {

		if !isSneaking {
			room.SendText(
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> is admiring their <ansi fg="item">%s</ansi>.`, mob.Character.Name, lookItem.DisplayName()),
			)
		}

		return true, nil
	}

	//
	// look for any mobs, players, npcs
	//

	playerId, mobId := room.FindByName(lookAt)

	if playerId > 0 || mobId > 0 {

		if playerId > 0 {

			u := *users.GetByUserId(playerId)

			if !isSneaking {
				u.SendText(
					fmt.Sprintf(`<ansi fg="mobname">%s</ansi> is looking at you.`, mob.Character.Name),
				)

				room.SendText(
					fmt.Sprintf(`<ansi fg="mobname">%s</ansi> is looking at <ansi fg="username">%s</ansi>.`, mob.Character.Name, u.Character.Name),
					u.UserId)
			}

		} else if mobId > 0 {

			m := mobs.GetInstance(mobId)
			if m == nil {
				return true, nil
			}

			if !isSneaking {
				targetName := m.Character.GetMobName(0).String()
				room.SendText(
					fmt.Sprintf(`<ansi fg="mobname">%s</ansi> is looking at %s.`, mob.Character.Name, targetName),
				)
			}

		}

		return true, nil

	}

	//
	// Check for any equipment they are wearing they might want to look at
	//
	if lookItem, found := mob.Character.FindOnBody(lookAt); found {

		if !isSneaking {
			room.SendText(
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> is admiring their <ansi fg="item">%s</ansi>.`, mob.Character.Name, lookItem.DisplayName()),
			)
		}

		return true, nil
	}

	//
	// Look for any nouns in the room info
	//
	foundNoun, _ := room.FindNoun(lookAt)
	if len(foundNoun) > 0 {

		if !isSneaking {
			room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> is examining the <ansi fg="noun">%s</ansi>.`, mob.Character.Name, foundNoun))
		}

		return true, nil
	}

	//
	// Look for any pets in the room
	//
	petUserId := room.FindByPetName(rest)
	if petUserId > 0 {

		if petUser := users.GetByUserId(petUserId); petUser != nil {

			room.SendText(
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> is looking at %s.`, mob.Character.Name, petUser.Character.Pet.DisplayName()))

			return true, nil
		}
	}

	return true, nil
}

func lookRoom(mob *mobs.Mob, roomId int, secretLook bool) {

	room := rooms.LoadRoom(roomId)

	if mob == nil || room == nil {
		return
	}

	// Make sure to prepare the room before anyone looks in if this is the first time someone has dealt with it in a while.
	if room.PlayerCt() < 1 {
		room.Prepare(true)
	}

	if !secretLook {
		// Find the exit back
		lookFromName := room.FindExitTo(mob.Character.RoomId)
		if lookFromName == "" {
			room.SendText(
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> is looking into the room from somewhere...`, mob.Character.Name),
			)
		} else {
			room.SendText(
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> is looking into the room from the <ansi fg="exit">%s</ansi> exit`, mob.Character.Name, lookFromName),
			)
		}
	}

}
