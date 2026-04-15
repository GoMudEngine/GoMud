package usercommands

import (
	"errors"
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Later this can be used for zombie specific decision making, such as when players intentionally zombify and let "ai" take over
func ZombieAct(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if !user.Character.HasAdjective(`zombie`) {
		return false, errors.New(`not a zombie!`)
	}

	if util.Rand(5) == 0 {
		room.SendTextVisual(fmt.Sprintf(`<ansi fg="username">%s</ansi> moans, groans and sways a bit...`, user.Character.Name), user.UserId)
	}

	return true, nil
}
