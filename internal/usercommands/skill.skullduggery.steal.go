package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

/*
Skullduggery Skill
Level 2 - Steal: attempt to steal from a mob or room container using
an opposed Dex+skill vs Perception roll.
*/
func Steal(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	skillLevel := user.Character.GetSkillLevel(skills.Skullduggery)

	// Requires skullduggery rank 1 to even see the command. Rank 2
	// is required for the actual steal — that gate lives inside
	// actions.Steal AFTER target validation, so immune-mob targets get
	// the canonical rebuff regardless of the player's skill level
	// (otherwise a rank-1 thief targeting a forager would see
	// "not advanced enough" — a misleading hint that the command would
	// work with more skill).
	if skillLevel < 1 {
		return false, nil
	}

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) == 0 {
		user.SendText(messaging.CategorySystem, "Steal from whom?")
		return true, nil
	}

	opts := parseStealArgs(args, room, user)
	if opts == nil {
		// parseStealArgs sent the appropriate message already.
		return true, nil
	}

	actor := &actions.UserActor{User: user, Room: room}
	result := actions.Steal(actor, *opts)

	// actions.Steal emits player-facing text for every outcome except
	// the cooldown case (which returns without calling SendText).
	if result.OnCooldown {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(
			"You need to wait %s before you can do that again.",
			result.Reason))
	}

	return true, nil
}

// parseStealArgs resolves CLI arguments to StealOptions.
// Args are already lowercased. Accepts:
//
//	steal <mob|container>
//	steal from <mob|container>
//
// Returns nil when the target cannot be resolved (message already sent).
// The "can't steal from players" guard lives here since the player
// command explicitly forbids PvP; actions.Steal supports it for mobs.
func parseStealArgs(args []string, room *rooms.Room, user *users.UserRecord) *actions.StealOptions {
	// Support optional "from" preposition: `steal from <target>`
	targetNoun := args[0]
	if targetNoun == "from" && len(args) > 1 {
		targetNoun = args[1]
	}

	// Try mob/player resolution first.
	target, err := actions.ResolveTargetActor(room, targetNoun)
	if err == nil {
		if target.IsPlayer() {
			user.SendText(messaging.CategorySystem, "You can't steal from other players.")
			return nil
		}
		return &actions.StealOptions{
			TargetMobInstanceId: target.(*actions.MobActor).Mob.InstanceId,
		}
	}

	// Try room container.
	containerName := room.FindContainerByName(targetNoun)
	if containerName != "" {
		return &actions.StealOptions{
			ContainerNoun: containerName,
		}
	}

	user.SendText(messaging.CategorySystem, "Steal from whom?")
	return nil
}
