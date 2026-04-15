package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/users"
)

//
// Checks for quests on the item
//

func CheckItemQuests(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.ItemOwnership)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "ItemOwnership", "Actual Type", e.Type())
		return events.Cancel
	}

	// Only care about users for this stuff
	if evt.UserId == 0 {
		return events.Continue
	}

	if evt.Gained {

		iSpec := evt.Item.GetSpec()
		if iSpec.QuestToken != `` {
			events.AddToQueue(events.Quest{
				UserId:     evt.UserId,
				QuestToken: iSpec.QuestToken,
			})
		}

		scripting.TryItemScriptEvent(`onFound`, evt.Item, evt.UserId)

		// Quest engine: item_gain notification
		if u := users.GetByUserId(evt.UserId); u != nil {
			bridge := questengine.NewGameBridge(u, u.Character.RoomId)
			questengine.GetEngine().Notify("item_gain", questengine.EventDetails{
				UserId: evt.UserId,
				RoomId: u.Character.RoomId,
				ItemId: evt.Item.ItemId,
			}, bridge, bridge)
		}

	} else {

		scripting.TryItemScriptEvent(`onLost`, evt.Item, evt.UserId)

	}

	return events.Continue
}
