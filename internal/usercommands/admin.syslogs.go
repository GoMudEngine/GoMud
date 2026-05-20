package usercommands

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

/*
* Role Permissions:
* syslogs 				(All)
 */
func SysLogs(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if rest == "" {
		infoOutput, _ := templates.Process("admincommands/help/command.syslogs", nil, user.UserId)
		user.SendTextLegacy(infoOutput)
		return true, nil
	}

	args := util.SplitButRespectQuotes(rest)

	if args[0] == "off" {

		events.AddToQueue(events.Log{FollowRemove: user.ConnectionId()})
		user.SendTextLegacy(`Log follow disabled.`)

		return true, nil
	}

	events.AddToQueue(events.Log{FollowAdd: user.ConnectionId(), Level: strings.ToUpper(args[0])})
	user.SendTextLegacy(`Log follow enabled. Use <ansi fg="command">syslogs off</ansi> to turn it off.`)

	return true, nil
}
