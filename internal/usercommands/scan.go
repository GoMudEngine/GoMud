package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Scan(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	user.SendTextLegacy(`You scan the surrounding area...`)
	user.SendTextLegacy(``)

	foundAnything := false

	for exitName, exitInfo := range room.Exits {

		if exitInfo.Secret {
			continue
		}

		adjRoom := rooms.LoadRoom(exitInfo.RoomId)
		if adjRoom == nil {
			continue
		}

		parts := []string{}

		for _, mobInstId := range adjRoom.GetMobs(rooms.FindAll) {
			mob := mobs.GetInstance(mobInstId)
			if mob == nil {
				continue
			}
			// Skip hidden mobs
			if mob.Character.IsHidden() {
				continue
			}
			parts = append(parts, fmt.Sprintf(`<ansi fg="mobname">%s</ansi>`, mob.Character.Name))
		}

		for _, playerId := range adjRoom.GetPlayers(rooms.FindAll) {
			if playerId == user.UserId {
				continue
			}
			u := users.GetByUserId(playerId)
			if u == nil {
				continue
			}
			parts = append(parts, fmt.Sprintf(`<ansi fg="username">%s</ansi>`, u.Character.Name))
		}

		dirLabel := fmt.Sprintf(`<ansi fg="exit">%s</ansi>`, exitName)
		titleLabel := fmt.Sprintf(`<ansi fg="room-title">%s</ansi>`, adjRoom.Title)

		if len(parts) > 0 {
			user.SendTextLegacy(fmt.Sprintf(`  %s (%s): %s`, dirLabel, titleLabel, strings.Join(parts, `, `)))
		} else {
			user.SendTextLegacy(fmt.Sprintf(`  %s (%s): nothing of interest`, dirLabel, titleLabel))
		}
		foundAnything = true
	}

	if !foundAnything {
		user.SendTextLegacy(`  There are no visible exits to scan.`)
	}

	user.SendTextLegacy(``)

	return true, nil
}
