package usercommands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/devtools"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

const devtoolUsage = `Usage:
  devtool check <zone>                             - Check zone for consistency issues
  devtool makezone <name> <width> <height>         - Generate a WxH grid of rooms in zone
  devtool linkzones <zoneA>/<roomIdA> <dir> <zoneB>/<roomIdB> - Link two rooms with bidirectional exit
  devtool json <json_string>                        - Execute a JSON API request`

/*
 * Role Permissions:
 * devtool    (Admin only)
 */
func Devtool(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	args := util.SplitButRespectQuotes(rest)

	if len(args) == 0 {
		user.SendText(devtoolUsage)
		return true, nil
	}

	subCmd := strings.ToLower(args[0])

	switch subCmd {

	case "check":
		if len(args) < 2 {
			user.SendText("Usage: devtool check <zone>")
			return true, nil
		}
		zoneName := strings.Join(args[1:], " ")
		report, issueCount, err := devtools.CheckZoneConsistency(zoneName)
		if err != nil {
			user.SendText(fmt.Sprintf("Error: %s", err.Error()))
			return true, nil
		}
		user.SendText(report)
		user.SendText(fmt.Sprintf("Total issues: %d", issueCount))

	case "makezone":
		if len(args) < 4 {
			user.SendText("Usage: devtool makezone <name> <width> <height>")
			return true, nil
		}
		zoneName := args[1]
		width, wErr := strconv.Atoi(args[2])
		height, hErr := strconv.Atoi(args[3])
		if wErr != nil || hErr != nil {
			user.SendText("Width and height must be integers.")
			return true, nil
		}
		firstId, lastId, err := devtools.GenerateGrid(zoneName, width, height)
		if err != nil {
			user.SendText(fmt.Sprintf("Error: %s", err.Error()))
			return true, nil
		}
		user.SendText(fmt.Sprintf(
			"Created %d rooms in zone %q (room IDs %d–%d).",
			width*height, zoneName, firstId, lastId,
		))

	case "linkzones":
		// linkzones <zoneA>/<roomIdA> <dir> <zoneB>/<roomIdB>
		if len(args) < 4 {
			user.SendText("Usage: devtool linkzones <zoneA>/<roomIdA> <dir> <zoneB>/<roomIdB>")
			return true, nil
		}
		zoneA, roomIdA, err := parseZoneRoom(args[1])
		if err != nil {
			user.SendText(fmt.Sprintf("Error parsing first arg: %s", err.Error()))
			return true, nil
		}
		direction := strings.ToLower(args[2])
		zoneB, roomIdB, err := parseZoneRoom(args[3])
		if err != nil {
			user.SendText(fmt.Sprintf("Error parsing third arg: %s", err.Error()))
			return true, nil
		}
		if err := devtools.LinkRooms(zoneA, roomIdA, direction, zoneB, roomIdB); err != nil {
			user.SendText(fmt.Sprintf("Error: %s", err.Error()))
			return true, nil
		}
		user.SendText(fmt.Sprintf(
			"Linked room %d (%s) %s ↔ room %d (%s).",
			roomIdA, zoneA, direction, roomIdB, zoneB,
		))

	case "json":
		if len(args) < 2 {
			user.SendText("Usage: devtool json <json_string>")
			return true, nil
		}
		jsonInput := strings.Join(args[1:], " ")
		result := devtools.HandleJSON(jsonInput)
		user.SendText(result)

	default:
		user.SendText(fmt.Sprintf("Unknown subcommand %q.\n%s", subCmd, devtoolUsage))
	}

	return true, nil
}

// parseZoneRoom splits "zoneName/roomId" into its components.
func parseZoneRoom(s string) (zone string, roomId int, err error) {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("%q must be in the format zone/roomId", s)
	}
	zone = parts[0]
	roomId, err = strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, fmt.Errorf("room ID %q is not an integer", parts[1])
	}
	return zone, roomId, nil
}
