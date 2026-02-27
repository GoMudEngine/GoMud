package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
Inspect — free command (no skill gate).
Reveals item details; depth scales with Perception stat.
*/
func Inspect(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if len(rest) == 0 {
		user.SendText("Type `help inspect` for more information on inspect.")
		return true, nil
	}

	// Inspect depth scales with Perception (1–4 equivalent tiers)
	perceptionAdj := user.Character.Stats.Perception.ValueAdj
	inspectLevel := 1
	if perceptionAdj >= 75 {
		inspectLevel = 4
	} else if perceptionAdj >= 50 {
		inspectLevel = 3
	} else if perceptionAdj >= 25 {
		inspectLevel = 2
	}

	matchItem, found := user.Character.FindInBackpack(rest)

	if !found {
		user.SendText(fmt.Sprintf("You don't have a %s to inspect. Is it still worn, perhaps?", rest))
	} else {

		user.SendText(
			fmt.Sprintf(`You inspect the <ansi fg="item">%s</ansi>.`, matchItem.DisplayName()),
		)
		room.SendText(
			fmt.Sprintf(`<ansi fg="username">%s</ansi> inspects their <ansi fg="item">%s</ansi>...`, user.Character.Name, matchItem.DisplayName()),
			user.UserId,
		)

		type inspectDetails struct {
			InspectLevel int
			Item         *items.Item
			ItemSpec     *items.ItemSpec
		}

		iSpec := matchItem.GetSpec()

		details := inspectDetails{
			InspectLevel: inspectLevel,
			Item:         &matchItem,
			ItemSpec:     &iSpec,
		}

		inspectTxt, _ := templates.Process("descriptions/inspect", details, user.UserId)
		user.SendText(inspectTxt)

		// Exercising perception via inspect
		events.AddToQueue(events.SkillUsed{UserId: user.UserId})
	}

	return true, nil
}
