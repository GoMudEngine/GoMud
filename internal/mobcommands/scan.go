package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func Scan(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	actor := &actions.MobActor{Mob: mob, Room: room}
	_ = actions.Scan(actor, actions.ScanOptions{HostileOnly: false})
	return true, nil
}
