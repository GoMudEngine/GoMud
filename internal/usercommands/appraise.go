package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Appraise(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	merchantMobs := room.GetMobs(rooms.FindMerchant)
	if len(merchantMobs) == 0 {
		user.SendText(`You need to be at a merchant to appraise items.`)
		return true, nil
	}

	for _, mobId := range merchantMobs {

		mob := mobs.GetInstance(mobId)
		if mob == nil {
			continue
		}

		if rest == "" {

			mob.Command(`say I will appraise items for 20 gold.`)

			return true, nil
		}

		item, found := user.Character.FindInBackpack(rest)
		if !found {
			user.SendText("You don't have that item.")
			return true, nil
		}

		itemSpec := item.GetSpec()
		if itemSpec.ItemId < 1 {
			return true, nil
		}

		type inspectDetails struct {
			InspectLevel int
			Item         *items.Item
			ItemSpec     *items.ItemSpec
		}

		details := inspectDetails{
			InspectLevel: 3,
			Item:         &item,
			ItemSpec:     &itemSpec,
		}

		appraisePrice := 20

		if appraisePrice > user.Character.Gold {

			mob.Command(fmt.Sprintf("say That costs %d gold to appraise, which you don't seem to have.", appraisePrice))

			return true, nil
		}

		user.Character.Gold -= appraisePrice
		mob.Character.Gold += appraisePrice

		events.AddToQueue(events.EquipmentChange{
			UserId:     user.UserId,
			GoldChange: appraisePrice,
		})

		user.SendText(fmt.Sprintf(`You give <ansi fg="mobname">%s</ansi> %d gold to appraise <ansi fg="itemname">%s</ansi>.`, mob.Character.Name, appraisePrice, itemSpec.Name))
		room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> appraises <ansi fg="itemname">%s</ansi>.`, user.Character.Name, itemSpec.Name), user.UserId)

		inspectTxt, _ := templates.Process("descriptions/inspect", details, user.UserId)
		user.SendText(inspectTxt)

		break
	}

	return true, nil
}
