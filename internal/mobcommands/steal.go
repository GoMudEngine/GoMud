package mobcommands

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Steal lets a mob attempt skullduggery theft. The mob command CLI
// accepts a target name (mob, player, or container noun). For
// btree-driven theft the action is called via actions.Steal directly
// (Task 13). This wrapper exists for admin testing and parity with
// the player command.
func Steal(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	opts := parseMobStealArgs(strings.ToLower(strings.TrimSpace(rest)), mob, room)
	actor := &actions.MobActor{Mob: mob, Room: room}
	actions.Steal(actor, opts)
	return true, nil
}

func parseMobStealArgs(rest string, mob *mobs.Mob, room *rooms.Room) actions.StealOptions {
	if rest == "" {
		return actions.StealOptions{}
	}

	args := util.SplitButRespectQuotes(rest)

	// Support optional "from" preposition: `steal from <target>`
	targetNoun := args[0]
	if targetNoun == "from" && len(args) > 1 {
		targetNoun = args[1]
	}

	// Try mob/player resolution first.
	target, err := actions.ResolveTargetActor(room, targetNoun,
		actions.ResolveTargetOptions{ExcludeMobInstanceId: mob.InstanceId})
	if err == nil {
		if target.IsPlayer() {
			return actions.StealOptions{TargetUserId: target.GetUserId()}
		}
		return actions.StealOptions{TargetMobInstanceId: target.GetMobInstanceId()}
	}

	// Try room container.
	if containerName := room.FindContainerByName(targetNoun); containerName != "" {
		return actions.StealOptions{ContainerNoun: containerName}
	}

	return actions.StealOptions{}
}
