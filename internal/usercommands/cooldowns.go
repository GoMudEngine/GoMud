package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Cooldowns(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	cdTxt, _ := templates.Process("character/cooldowns", user.Character.GetAllCooldowns(), user.UserId)
	user.SendText(messaging.CategorySystem, cdTxt)

	return true, nil
}
