package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Buy is the mob-side entry into actions.Buy. The mob has no
// list-fall-through (mobs don't read text), so empty rest is a
// silent no-op.
func Buy(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	if rest == "" {
		return true, nil
	}
	actor := &actions.MobActor{Mob: mob, Room: room}
	actions.Buy(actor, actions.BuyOptions{Request: rest})
	return true, nil
}
