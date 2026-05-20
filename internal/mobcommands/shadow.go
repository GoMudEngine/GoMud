package mobcommands

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Shadow lets a mob begin shadowing a target while hidden. The mob
// must already carry the Hidden buff (buff 9). The mob command CLI
// accepts the target name as the argument. For btree-driven shadowing
// the action is called via actions.Shadow directly (Task 13).
func Shadow(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	target := strings.TrimSpace(strings.ToLower(rest))
	if target == "" {
		return true, nil
	}

	resolved, err := actions.ResolveTargetActor(room, target,
		actions.ResolveTargetOptions{ExcludeMobInstanceId: mob.InstanceId})
	if err != nil {
		return true, nil
	}

	opts := actions.ShadowOptions{}
	if resolved.IsPlayer() {
		opts.TargetUserId = resolved.GetUserId()
	} else {
		opts.TargetMobInstanceId = resolved.GetMobInstanceId()
	}

	actor := &actions.MobActor{Mob: mob, Room: room}
	actions.Shadow(actor, opts)
	return true, nil
}
