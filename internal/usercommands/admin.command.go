package usercommands

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobcommands"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"

	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
* Role Permissions:
* command 				(All)
 */
func Command(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// args should look like one of the following:
	// target buffId - put buff on target if in the room
	// buffId - put buff on self
	// search searchTerm - search for buff by name, display results
	args := util.SplitButRespectQuotes(rest)

	if len(args) < 2 {
		// send some sort of help info?
		mobCommands := mobcommands.GetAllMobCommands()

		infoOutput, _ := templates.Process("admincommands/help/command.command", mobCommands, user.UserId)
		user.SendText(infoOutput)
		return true, nil
	}

	searchName := args[0]
	args = args[1:]
	cmd := strings.TrimPrefix(rest, searchName+` `)
	//cmd := strings.Join(args, ` `)

	target, err := actions.ResolveTargetActor(room, searchName)
	if err != nil {
		return true, nil
	}

	playerId := target.GetUserId()
	mobId := target.GetMobInstanceId()

	// Use the index for how many turns to defer the extra commands
	readyTurn := util.GetTurnCount()
	for _, oneCmd := range strings.Split(cmd, `;`) {
		if mobId > 0 {

			events.AddToQueue(events.Input{
				MobInstanceId: mobId,
				InputText:     oneCmd,
				ReadyTurn:     readyTurn,
			})

		} else if playerId > 0 {

			events.AddToQueue(events.Input{
				UserId:    playerId,
				InputText: oneCmd,
				ReadyTurn: readyTurn,
			})

		}
		readyTurn++
	}

	return true, nil
}
