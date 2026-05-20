package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Who(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	details := rooms.GetDetails(room, user)

	whoTxt, _ := templates.Process("descriptions/who", details, user.UserId)
	user.SendTextLegacy(whoTxt)

	return true, nil
}
