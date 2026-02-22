package usercommands

import (
	"fmt"
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/language"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Online(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	headers := []string{
		language.T(`User.Name`),
		language.T(`Alignment`),
		language.T(`Profession`),
		language.T(`Online`),
		language.T(`Role`),
	}

	if user.Role != users.RoleUser {
		headers = append([]string{language.T(`UserId`)}, headers...)
		headers = append(headers, []string{language.T(`Zone`), language.T(`RoomId`)}...)
	}

	allFormatting := [][]string{}

	rows := [][]string{}

	userCt := 0
	for _, uid := range users.GetOnlineUserIds() {

		u := users.GetByUserId(uid)

		if u != nil {

			onlineInfo := u.GetOnlineInfo()

			userCt++

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

			row := []string{
				onlineInfo.CharacterName,
				onlineInfo.Alignment,
				onlineInfo.Profession,
				onlineTime,
				onlineInfo.Role,
			}

			formatting := []string{
				`<ansi fg="username">%s</ansi>`,
				`<ansi fg="` + onlineInfo.Alignment + `">%s</ansi>`,
				`<ansi fg="white-bold">%s</ansi>`,
				`<ansi fg="magenta">%s</ansi>`,
				`<ansi fg="role-` + permClass + `-bold">%s</ansi>`,
			}

			if user.Role != users.RoleUser {
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

	onlineResultsTable := templates.GetTable(tableTitle, headers, rows, allFormatting...)
	tplTxt, _ := templates.Process("tables/generic", onlineResultsTable, user.UserId)
	user.SendText(tplTxt)

	return true, nil
}
