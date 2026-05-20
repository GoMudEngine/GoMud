package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Exits(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	details := rooms.GetDetails(room, user)

	exitTxt, _ := templates.Process("descriptions/exits", details, user.UserId)
	user.SendText(messaging.CategoryRoomDescription, exitTxt)

	return true, nil
}
