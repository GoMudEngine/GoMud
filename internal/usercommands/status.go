package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Status(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	tplTxt, _ := templates.Process("character/status", user, user.UserId)
	user.SendText(tplTxt)

	return true, nil
}
