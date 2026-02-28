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

	isSneaking := user.Character.HasBuffFlag(buffs.Hidden)
	isDrunk := user.Character.HasBuffFlag(buffs.Drunk)

	rest = strings.ToUpper(rest)

	if isDrunk {
		// modify the text to look like it's the speech of a drunk person
		rest = drunkify(rest)
	}

	// Wrap long shout text to 80 chars
	rest = util.NormalizeAndWrap(rest, 80)

	if isSneaking {
		room.SendTextCommunication(fmt.Sprintf(`someone shouts, "<ansi fg="yellow">%s</ansi>"`, rest), user.UserId)
	} else {
		room.SendTextCommunication(fmt.Sprintf(`<ansi fg="username">%s</ansi> shouts, "<ansi fg="yellow">%s</ansi>"`, user.Character.Name, rest), user.UserId)
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

	user.SendText(fmt.Sprintf(`You shout, "<ansi fg="yellow">%s</ansi>"`, rest))

	return true, nil
}
