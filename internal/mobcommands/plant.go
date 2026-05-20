package mobcommands

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Plant lets a mob attempt to slip an item into another character's
// or container's inventory. The mob command CLI accepts:
//
//	plant <item> <target>
//	plant <item> on <target>
//	plant <item> in <target>
//
// For btree-driven planting the action is called via actions.Plant
// directly (Task 13). This wrapper exists for admin testing and
// parity with the player command.
func Plant(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	rest = strings.ToLower(strings.TrimSpace(rest))
	if rest == "" {
		return true, nil
	}

	args := util.SplitButRespectQuotes(rest)
	if len(args) < 2 {
		return true, nil
	}

	itemNoun := args[0]

	// Skip optional "on" / "in" preposition.
	targetIdx := 1
	if len(args) >= 3 && (args[1] == "on" || args[1] == "in") {
		targetIdx = 2
	}
	targetNoun := args[targetIdx]

	opts := actions.PlantOptions{ItemNoun: itemNoun}

	target, err := actions.ResolveTargetActor(room, targetNoun,
		actions.ResolveTargetOptions{ExcludeMobInstanceId: mob.InstanceId})
	if err == nil {
		if target.IsPlayer() {
			opts.TargetUserId = target.GetUserId()
		} else {
			opts.TargetMobInstanceId = target.GetMobInstanceId()
		}
	} else if containerName := room.FindContainerByName(targetNoun); containerName != "" {
		opts.ContainerNoun = containerName
	}

	actor := &actions.MobActor{Mob: mob, Room: room}
	actions.Plant(actor, opts)
	return true, nil
}
