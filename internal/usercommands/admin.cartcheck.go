package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mapper"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
* Role Permissions:
* cartcheck 				(Admin)
 */
func CartCheck(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	var zoneNames []string
	rest = strings.TrimSpace(rest)
	if rest != "" {
		zoneNames = []string{rest}
	} else {
		zoneNames = rooms.GetAllZoneNames()
	}

	totalErr, totalWarn := 0, 0
	var lines []string

	for _, zoneName := range zoneNames {
		rootRoomId, err := rooms.GetZoneRoot(zoneName)
		if err != nil {
			continue
		}
		m := mapper.GetMapper(rootRoomId)
		if m == nil {
			continue
		}
		findings := m.CheckConsistency(zoneName, rooms.IsZoneNonCartesian(zoneName))
		if len(findings) == 0 {
			continue
		}
		lines = append(lines, fmt.Sprintf(`<ansi fg="yellow">%s</ansi>`, zoneName))
		for _, f := range findings {
			if f.Severity == "error" {
				totalErr++
			} else {
				totalWarn++
			}
			lines = append(lines, "  "+f.String())
		}
	}

	if len(lines) == 0 {
		user.SendText(messaging.CategorySystem, "Cartesian consistency: no findings. All checked zones are clean.")
		return true, nil
	}

	out := strings.Join(lines, "\n")
	out += fmt.Sprintf("\n\n%d error(s), %d warning(s).", totalErr, totalWarn)
	user.SendText(messaging.CategorySystem, out)
	return true, nil
}
