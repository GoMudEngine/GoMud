package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Biome(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	biome, ok := rooms.GetBiome(room.Biome)

	if !ok {
		user.SendText(messaging.CategorySystem, `No biome information found about this area.`)
		return false, fmt.Errorf(`biome %s not found`, room.Biome)
	}

	biomeTxt, _ := templates.Process("descriptions/biome", biome, user.UserId)
	user.SendText(messaging.CategorySystem, biomeTxt)

	return true, nil
}
