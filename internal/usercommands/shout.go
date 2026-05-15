package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Shout(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if user.Muted {
		user.SendText(`You are <ansi fg="alert-5">MUTED</ansi>. You can only send <ansi fg="command">whisper</ansi>'s to Admins and Moderators.`)
		return true, nil
	}

	isSneaking := user.Character.IsHidden()
	isDrunk := user.Character.HasBuffFlag(buffs.Drunk)

	rest = strings.ToUpper(rest)

	if isDrunk {
		// modify the text to look like it's the speech of a drunk person
		rest = drunkify(rest)
	}

	if isSneaking {
		msg := fmt.Sprintf(`someone shouts, "<ansi fg="yellow">%s</ansi>"`, rest)
		room.SendTextCommunication(util.SplitStringNL(msg, 80), user.UserId)
	} else {
		msg := fmt.Sprintf(`<ansi fg="username">%s</ansi> shouts, "<ansi fg="yellow">%s</ansi>"`, user.Character.Name, rest)
		room.SendTextCommunication(util.SplitStringNL(msg, 80), user.UserId)
	}

	for _, roomInfo := range room.Exits {
		if otherRoom := rooms.LoadRoom(roomInfo.RoomId); otherRoom != nil {
			if sourceExit := otherRoom.FindExitTo(room.RoomId); sourceExit != `` {
				otherRoom.SendTextCommunication(fmt.Sprintf(`Someone shouts from the <ansi fg="exit">%s</ansi> direction, "<ansi fg="yellow">%s</ansi>"`, sourceExit, rest), user.UserId)
			}
		}
	}

	for _, roomInfo := range room.ExitsTemp {
		if otherRoom := rooms.LoadRoom(roomInfo.RoomId); otherRoom != nil {
			if sourceExit := otherRoom.FindExitTo(room.RoomId); sourceExit != `` {
				otherRoom.SendTextCommunication(fmt.Sprintf(`Someone shouts from the <ansi fg="exit">%s</ansi> direction, "<ansi fg="yellow">%s</ansi>"`, sourceExit, rest), user.UserId)
			}
		}
	}

	for mut := range room.ActiveMutators {
		spec := mut.GetSpec()
		if len(spec.Exits) == 0 {
			continue
		}
		for _, exitInfo := range spec.Exits {
			if otherRoom := rooms.LoadRoom(exitInfo.RoomId); otherRoom != nil {
				if sourceExit := otherRoom.FindExitTo(room.RoomId); sourceExit != `` {
					otherRoom.SendTextCommunication(fmt.Sprintf(`Someone shouts from the <ansi fg="exit">%s</ansi> direction, "<ansi fg="yellow">%s</ansi>"`, sourceExit, rest), user.UserId)
				}
			}
		}
	}

	selfMsg := fmt.Sprintf(`You shout, "<ansi fg="yellow">%s</ansi>"`, rest)
	user.SendText(util.SplitStringNL(selfMsg, 80))

	return true, nil
}
