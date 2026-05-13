package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

/*
Skullduggery Skill
Plant: slip an item from backpack onto a mob or into a container.
Shares the skullduggery cooldown with steal.
*/
func Plant(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	skillLevel := user.Character.GetSkillLevel(skills.Skullduggery)

	if skillLevel < 1 {
		return false, nil
	}

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) == 0 {
		user.SendText("Plant what?")
		return true, nil
	}

	if len(args) == 1 {
		user.SendText("Plant on whom?")
		return true, nil
	}

	opts, ok := parsePlantArgs(args, room, user)
	if !ok {
		// parsePlantArgs sent the appropriate message already.
		return true, nil
	}

	actor := &actions.UserActor{User: user, Room: room}
	result := actions.Plant(actor, opts)

	// actions.Plant emits player-facing text for every outcome except
	// the cooldown case (which returns without calling SendText).
	if result.OnCooldown {
		user.SendText(fmt.Sprintf(
			"You need to wait %s before you can do that again.",
			result.Reason))
	}

	return true, nil
}

// parsePlantArgs resolves CLI arguments to PlantOptions. Accepts:
//
//	plant <item> <target>
//	plant <item> on <target>
//	plant <item> in <target>
//
// Returns (opts, true) on success; (zero, false) when the wrapper
// already sent a rebuff and should not call Plant.
func parsePlantArgs(args []string, room *rooms.Room, user *users.UserRecord) (actions.PlantOptions, bool) {
	// Locate optional preposition ("on" or "in") to split item and target.
	prepIdx := -1
	for i, a := range args {
		if a == "on" || a == "in" {
			prepIdx = i
			break
		}
	}

	var itemNoun, targetNoun string
	if prepIdx > 0 && prepIdx < len(args)-1 {
		// plant <item...> on/in <target...>
		itemNoun = strings.Join(args[:prepIdx], " ")
		targetNoun = strings.Join(args[prepIdx+1:], " ")
	} else {
		// plant <item> <target>  — last token is target
		itemNoun = strings.Join(args[:len(args)-1], " ")
		targetNoun = args[len(args)-1]
	}

	if targetNoun == "" {
		user.SendText("Plant on whom?")
		return actions.PlantOptions{}, false
	}

	// Try mob/player resolution first.
	target, err := actions.ResolveTargetActor(room, targetNoun)
	if err == nil {
		if target.IsPlayer() {
			user.SendText("You can't plant items on other players.")
			return actions.PlantOptions{}, false
		}
		return actions.PlantOptions{
			ItemNoun:            itemNoun,
			TargetMobInstanceId: target.(*actions.MobActor).Mob.InstanceId,
		}, true
	}

	// Try room container.
	containerName := room.FindContainerByName(targetNoun)
	if containerName != "" {
		return actions.PlantOptions{
			ItemNoun:      itemNoun,
			ContainerNoun: containerName,
		}, true
	}

	user.SendText("Plant on whom?")
	return actions.PlantOptions{}, false
}
