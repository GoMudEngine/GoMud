package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Consider is the mob-side entry into actions.Consider. The mob
// has no list-fall-through, so empty rest is a silent no-op.
// MobActor.SendText is a no-op, so the math runs but no text
// is emitted.
func Consider(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	if rest == "" {
		return true, nil
	}
	target, err := actions.ResolveTargetActor(room, rest,
		actions.ResolveTargetOptions{ExcludeMobInstanceId: mob.InstanceId})
	if err != nil {
		return true, nil
	}
	actor := &actions.MobActor{Mob: mob, Room: room}
	actions.Consider(actor, target)
	return true, nil
}
