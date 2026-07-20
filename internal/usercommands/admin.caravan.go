package usercommands

// admin.caravan.go — admin override for the caravan state machine.
//
// Usage:
//   caravan reset            — reset all caravan leaders in the world
//   caravan reset <instId>   — reset a single leader by instance ID

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

/*
* Role Permissions:
* caravan	(Admin only)
 */
func Caravan(rest string, user *users.UserRecord, room *rooms.Room,
	flags events.EventFlag) (bool, error) {

	args := util.SplitButRespectQuotes(rest)

	if len(args) == 0 {
		user.SendText(messaging.CategorySystem, `Usage: <ansi fg="command">caravan reset [instanceId]</ansi>`)
		return true, nil
	}

	subCmd := strings.ToLower(args[0])

	switch subCmd {
	case "reset":
		if len(args) >= 2 {
			// Targeted reset by instance ID.
			instId, err := strconv.Atoi(args[1])
			if err != nil {
				user.SendText(messaging.CategorySystem, fmt.Sprintf(
					`<ansi fg="red">Invalid instance ID %q — must be an integer.</ansi>`,
					args[1],
				))
				return true, nil
			}
			if behaviortree.ResetCaravanStateByInstanceId(instId) {
				user.SendText(messaging.CategorySystem, fmt.Sprintf(
					`<ansi fg="yellow">Caravan leader #%d reset to ThornwallDwell.</ansi>`,
					instId,
				))
			} else {
				user.SendText(messaging.CategorySystem, fmt.Sprintf(
					`<ansi fg="red">Instance #%d not found or is not a caravan leader.</ansi>`,
					instId,
				))
			}
			return true, nil
		}

		// Global reset — all caravan leaders.
		count := behaviortree.ResetAllCaravanStates()
		if count == 0 {
			user.SendText(messaging.CategorySystem,
				`<ansi fg="yellow">No active caravan leaders found.</ansi>`,
			)
		} else {
			user.SendText(messaging.CategorySystem, fmt.Sprintf(
				`<ansi fg="yellow">Reset %d caravan leader(s) to ThornwallDwell.</ansi>`,
				count,
			))
		}
		return true, nil

	default:
		user.SendText(messaging.CategorySystem, `Usage: <ansi fg="command">caravan reset [instanceId]</ansi>`)
	}

	return true, nil
}
