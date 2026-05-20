package usercommands

import (
	"fmt"
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/language"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Online(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	isAdmin := user.Role != users.RoleUser

	headers := []string{
		language.T(`User.Name`),
		language.T(`Title`),
		language.T(`Online`),
		language.T(`Role`),
	}

	if isAdmin {
		headers = append([]string{language.T(`UserId`)}, headers...)
		headers = append(headers, []string{language.T(`Zone`), language.T(`RoomId`)}...)
	}

	allFormatting := [][]string{}

	rows := [][]string{}

	userCt := 0
	aiCt := 0
	humanCt := 0
	for _, uid := range users.GetOnlineUserIds() {

		u := users.GetByUserId(uid)

		if u != nil {

			onlineInfo := u.GetOnlineInfo()

			userCt++
			if onlineInfo.IsAI {
				aiCt++
			} else {
				humanCt++
			}

			onlineTime := onlineInfo.OnlineTimeStr
			if onlineInfo.IsAFK {
				onlineTime += ` <ansi fg="8">(afk)</ansi>`
			}

			permClass := `user`
			if onlineInfo.Role != users.RoleUser {
				if onlineInfo.Role == users.RoleAdmin {
					permClass = `admin`
				} else {
					permClass = `mod`
				}
			}

			// For admins, show [AI] tag next to AI-connected player names
			charName := onlineInfo.CharacterName
			if isAdmin && onlineInfo.IsAI {
				charName += ` <ansi fg="8">[AI]</ansi>`
			}

			row := []string{
				charName,
				onlineInfo.Title,
				onlineTime,
				onlineInfo.Role,
			}

			formatting := []string{
				`<ansi fg="username">%s</ansi>`,
				`<ansi fg="white-bold">%s</ansi>`,
				`<ansi fg="magenta">%s</ansi>`,
				`<ansi fg="role-` + permClass + `-bold">%s</ansi>`,
			}

			if isAdmin {
				row = append([]string{strconv.Itoa(u.UserId)}, row...)
				row = append(row, []string{u.Character.Zone, strconv.Itoa(u.Character.RoomId)}...)

				formatting = append([]string{`<ansi fg="userid">%s</ansi>`}, formatting...)
				formatting = append(formatting, []string{`<ansi fg="zone">%s</ansi>`, `<ansi fg="1">%s</ansi>`}...)
			}

			allFormatting = append(allFormatting, formatting)

			rows = append(rows, row)
		}
	}

	tableTitle := fmt.Sprintf(language.T(`%d users online`), userCt)
	if userCt == 1 {
		tableTitle = fmt.Sprintf(language.T(`%d user online`), userCt)
	}
	// Add AI/human breakdown for admins
	if isAdmin && aiCt > 0 {
		tableTitle += fmt.Sprintf(` (%d human, %d AI)`, humanCt, aiCt)
	}

	onlineResultsTable := templates.GetTable(tableTitle, headers, rows, allFormatting...)
	tplTxt, _ := templates.Process("tables/generic", onlineResultsTable, user.UserId)
	user.SendText(messaging.CategorySystem, tplTxt)

	return true, nil
}
