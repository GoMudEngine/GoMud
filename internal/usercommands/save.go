package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Save(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	user.SendText(messaging.CategorySystem, "Saving...")
	users.SaveUser(*user)
	user.SendText(messaging.CategorySystem, "done.")

	return true, nil
}
