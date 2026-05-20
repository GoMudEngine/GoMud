package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mapper"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

/*
* Role Permissions:
* build 				(All)
 */
func Build(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// args should look like one of the following:
	// info <optional room id>
	// <move to room id>
	args := util.SplitButRespectQuotes(rest)

	if len(args) < 2 {
		// send some sort of help info?
		infoOutput, _ := templates.Process("admincommands/help/command.build", nil, user.UserId, user.UserId)
		user.SendTextLegacy(infoOutput)
	} else {

		// #build zone "The Arctic"
		if args[0] == "zone" {

			zoneName := strings.Join(args[1:], ` `)

			if roomId, err := rooms.CreateZone(zoneName); err != nil {
				user.SendTextLegacy(err.Error())
			} else {
				user.SendTextLegacy(fmt.Sprintf("Zone %s created.", zoneName))

				if err := rooms.MoveToRoom(user.UserId, roomId); err != nil {
					user.SendTextLegacy(err.Error())
				} else {
					user.SendTextLegacy(fmt.Sprintf("Moved to room %d.", roomId))
					events.AddToQueue(events.Input{
						UserId:    user.UserId,
						InputText: `look`,
					}, -1)
				}
			}
		}

		// build room north <south>
		// build room north-x2 <south-x2>
		// build room attic:up <down>
		if args[0] == "room" {

			var err error

			// Prep exit parts of command
			// There are special exit names (defined in internal/mapper/mapper.go ) that
			// translate to compass directions such as east-x3 (east 3 "room spaces" away)
			// exit names can be freeform, but if they are to show up on a map they must have
			// a compass direction. This can be achieved by appending ":direction" to an exit name
			// such as: "attic:up" or "zipline:east-x3"
			exitName := args[1]
			exitDirection := exitName // A special exit name that corresponds to special distances
			returnExitName := ``
			returnExitDirection := ``

			if exitName, exitDirection, err = mapper.AdjustExitName(exitName); err != nil {
				user.SendTextLegacy(err.Error())
				return true, err
			}

			// Prep optional "return exit" parts of command
			// If not supplied, defaults to a possible reciprocal exit
			if len(args) > 2 {
				returnExitName = args[2]
			} else {
				returnExitName = mapper.GetReciprocalExit(exitDirection)
			}

			if returnExitName != `` {
				if returnExitName, returnExitDirection, err = mapper.AdjustExitName(returnExitName); err != nil {
					user.SendTextLegacy(err.Error())
					return true, err
				}
			}

			// #build (room north) - room+north are two args
			var destinationRoom *rooms.Room = nil
			// If it's a compass direction, reject it if a room already exists in that direction

			rMapper := mapper.GetMapper(room.RoomId)
			if rMapper == nil {
				err := fmt.Errorf("Could not find mapper for roomId: %d", room.RoomId)
				mudlog.Error("Map", "error", err)
				user.SendTextLegacy(`No map found (or an error occured)"`)
				return true, err
			}

			// Is there a room in that direction already, even if blocked by a wall?
			gotoRoomId, _ := rMapper.FindAdjacentRoom(user.Character.RoomId, exitName, 1)

			if gotoRoomId == 0 {

				newRoom, err := rooms.BuildRoom(user.Character.RoomId, exitName, exitDirection)

				// If there was a problem building the room, send the error to the user before returning
				if err != nil {
					user.SendTextLegacy(err.Error())
					user.SendTextLegacy(fmt.Sprintf("Error building room %s.", exitName))
					return false, nil
				}

				destinationRoom = newRoom

			} else {
				destinationRoom = rooms.LoadRoom(gotoRoomId)
				if _, ok := destinationRoom.Exits[exitName]; !ok {
					rooms.ConnectRoom(user.Character.RoomId, destinationRoom.RoomId, exitName, exitDirection)
				}
			}

			// Connect the exit back
			if len(returnExitName) > 0 {
				rooms.ConnectRoom(destinationRoom.RoomId, user.Character.RoomId, returnExitName, returnExitDirection)
			}

			if err := rooms.MoveToRoom(user.UserId, destinationRoom.RoomId); err != nil {
				user.SendTextLegacy(err.Error())
			} else {
				user.SendTextLegacy(fmt.Sprintf("Moved to room %d.", destinationRoom.RoomId))

				events.AddToQueue(events.Input{
					UserId:    user.UserId,
					InputText: `look`,
				}, -1)
			}

		}

	}

	return true, nil
}
