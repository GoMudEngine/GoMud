package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Forage runs a single forage attempt for the mob. Thin wrapper
// over actions.Forage. Mob actor is silent — the structured result
// is consumed by the btree primitive try_forage.
func Forage(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	actor := actions.NewMobActorInRoom(mob, room)
	_ = actions.Forage(actor, actions.ForageOptions{})
	return true, nil
}
