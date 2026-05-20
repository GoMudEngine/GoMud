package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/util"

	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
* Role Permissions:
* modify 				(All)
* modify.role			(Change user roles)
 */
func Modify(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	args := util.SplitButRespectQuotes(rest)

	if len(args) < 3 {
		infoOutput, _ := templates.Process("admincommands/help/command.modify", nil, user.UserId)
		user.SendTextLegacy(infoOutput)
		return true, nil
	}

	// modify permission <username> <newperms>
	if args[0] == `role` {

		if !user.HasRolePermission(`modify.role`) {
			user.SendTextLegacy(`you do not have <ansi fg="command">modify.role</ansi> permission`)
			return true, nil
		}

		searchUser := args[1]
		newRole := args[2]

		allRoles := configs.GetRolesConfig()
		_, roleExists := allRoles[newRole]

		if newRole != users.RoleAdmin && newRole != users.RoleUser && !roleExists {
			user.SendTextLegacy(`<ansi fg="alert-4">Invalid permission type.</ansi>`)
			return true, nil
		}

		foundUsername := ``
		foundCharacterName := ``

		for _, u := range users.GetAllActiveUsers() {
			if strings.EqualFold(searchUser, u.Username) {

				if u.Role == users.RoleAdmin {
					user.SendTextLegacy(`<ansi fg="alert-4">Admin permissions cannot be removed this way.</ansi>`)
					return true, nil
				}

				if u.Role == newRole {
					user.SendTextLegacy(`<ansi fg="alert-4">That permission is already set for this user.</ansi>`)
					return true, nil
				}

				foundCharacterName = u.Character.Name
				foundUsername = u.Username

				u.Role = newRole

				users.SaveUser(*u)

				u.SendTextLegacy(`<ansi fg="alert-3">Your role has been set to: ` + newRole + `</ansi>`)
				break
			}
		}

		if len(foundUsername) == 0 {
			users.SearchOfflineUsers(func(u *users.UserRecord) bool {

				if strings.EqualFold(searchUser, u.Username) {

					if u.Role == users.RoleAdmin {
						user.SendTextLegacy(`<ansi fg="alert-4">Admin permissions cannot be removed this way.</ansi>`)
						return false
					}

					if u.Role == newRole {
						user.SendTextLegacy(`<ansi fg="alert-4">That permission is already set for this user.</ansi>`)
						return false
					}

					foundCharacterName = u.Character.Name
					foundUsername = u.Username

					u.Role = newRole

					users.SaveUser(*u)

					return false
				}

				return true
			})
		}

		if len(foundUsername) > 0 {
			user.SendTextLegacy(fmt.Sprintf(`Role changed for user <ansi fg="username">%s</ansi> (Character name: <ansi fg="username">%s</ansi>).`, foundUsername, foundCharacterName))
			return true, nil
		}

	}

	return true, nil
}
