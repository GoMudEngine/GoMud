package behaviortree

// actions_quest.go — quest, item, and gold actions:
// actGrantQuest, actSetQuestFlag, actGrantMutation, actGiveGold, actTakeGold,
// actGiveItem, actReturnItem, actTakeItem, actGiveItemMultiple, actSetMiscData

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/uuid"
)

func actGrantQuest(params map[string]any, ctx *EvalContext) Result {
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}
	quest := getStringParam(params, "quest")
	if quest == "" {
		return Failure
	}
	// Set the token synchronously (for any in-tree follow-up), then fire the
	// quest event so the player gets the progress/completion banner and any
	// rewards — the same pipeline the dialogue and questengine grant paths use.
	// Without this, behavior-tree grants (e.g. Hadwen granting 30-end at the
	// close of the Awakening Rite) advanced the quest silently, with no banner
	// and no rewards.
	user.Character.GiveQuestToken(quest)
	events.AddToQueue(events.Quest{
		UserId:     user.UserId,
		QuestToken: quest,
	})
	return Success
}

func actSetQuestFlag(params map[string]any, ctx *EvalContext) Result {
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}
	key := getStringParam(params, "flag_key")
	value := getStringParam(params, "flag_value")
	if key == "" {
		return Failure
	}
	user.Character.SetQuestFlag(key, value)
	return Success
}

// actGrantMutation rolls and grants a random mutation to the triggering player
// from the weighted acquisition pool. Returns Success even if no mutations are
// available (no eligible mutations is not an error).
func actGrantMutation(params map[string]any, ctx *EvalContext) Result {
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}
	if mutId := user.Character.GrantRandomMutation(); mutId != "" {
		events.AddToQueue(mutations.Gained{
			UserId:     user.UserId,
			MutationId: mutId,
			Rank:       1,
			IsNew:      true,
		})
	}
	return Success
}

func actGiveGold(params map[string]any, ctx *EvalContext) Result {
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}
	amount := getIntParam(params, "amount")
	if amount <= 0 {
		return Failure
	}
	user.Character.Gold += amount
	user.SendText(messaging.CategoryLoot, fmt.Sprintf("You receive %d gold.\n", amount))
	return Success
}

func actTakeGold(params map[string]any, ctx *EvalContext) Result {
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}
	amount := getIntParam(params, "amount")
	if amount <= 0 {
		return Failure
	}
	user.Character.Gold -= amount
	if user.Character.Gold < 0 {
		user.Character.Gold = 0
	}
	return Success
}

func actGiveItem(params map[string]any, ctx *EvalContext) Result {
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}
	itemId := getIntParam(params, "item_id")
	if itemId == 0 {
		return Failure
	}
	item := items.New(itemId)
	if !user.Character.StoreItem(item) {
		return Failure
	}
	user.SendText(messaging.CategoryLoot, fmt.Sprintf("You receive a %s.\n", item.Name()))
	return Success
}

// actReturnItem hands the triggering event's item back to the player.
// Used in player_give handlers to reject/return items. Unlike give_item,
// no item_id param is needed — the item comes from ctx.Event.
//
// This returns the ACTUAL item the mob was just handed, not a fresh copy.
// give.go transfers the real, stateful item into the mob's inventory before
// any handler fires, so minting a copy here both duplicated the item (the
// original stayed on the mob and later dropped as corpse loot) and reset the
// player's enchant tier / remaining uses / durability to template defaults.
//
// The exact instance is identified by ctx.Event.ItemUUID when the producer
// supplied one, falling back to the most recently stored item matching
// ctx.Event.ItemId. If the mob or the item cannot be resolved the action
// fails and logs — it never falls back to fabricating an item.
func actReturnItem(params map[string]any, ctx *EvalContext) Result {
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}
	itemId := ctx.Event.ItemId
	if itemId == 0 {
		return Failure
	}

	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		mudlog.Warn("actReturnItem", "msg", "mob instance not found",
			"instanceId", ctx.InstanceId, "itemId", itemId, "userId", user.UserId)
		return Failure
	}

	item, found := findHeldItem(&mob.Character, itemId, ctx.Event.ItemUUID)
	if !found {
		mudlog.Warn("actReturnItem", "msg", "item not held by mob",
			"instanceId", ctx.InstanceId, "itemId", itemId, "userId", user.UserId)
		return Failure
	}

	// TransferItemBetweenChars removes from the mob first and restores it to
	// the mob if the player cannot carry it, so no path can leave the item in
	// both inventories or in neither.
	if err := actions.TransferItemBetweenChars(item,
		&mob.Character, 0, mob.InstanceId,
		user.Character, user.UserId, 0,
	); err != nil {
		mudlog.Warn("actReturnItem", "msg", "transfer failed",
			"instanceId", ctx.InstanceId, "itemId", itemId, "userId", user.UserId, "error", err)
		return Failure
	}

	user.SendText(messaging.CategoryMobEmote, fmt.Sprintf("%s hands back the %s.\n", ctx.MobName, item.Name()))
	return Success
}

// findHeldItem locates a specific carried item on a character. It prefers an
// exact UUID match; when no usable UUID is supplied it falls back to the most
// recently stored item with a matching template id (StoreItem appends, so the
// last match is the one just handed over). Searches all three carry lists
// because StoreItem may auto-route into the component bag or bandolier.
func findHeldItem(c *characters.Character, itemId int, itemUUID uuid.UUID) (items.Item, bool) {
	lists := [][]items.Item{c.Items, c.ComponentItems, c.PotionItems}

	if !itemUUID.IsNil() {
		for _, list := range lists {
			for j := len(list) - 1; j >= 0; j-- {
				if list[j].UUID == itemUUID {
					return list[j], true
				}
			}
		}
		return items.Item{}, false
	}

	for _, list := range lists {
		for j := len(list) - 1; j >= 0; j-- {
			if list[j].ItemId == itemId {
				return list[j], true
			}
		}
	}
	return items.Item{}, false
}

func actTakeItem(params map[string]any, ctx *EvalContext) Result {
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}
	itemId := getIntParam(params, "item_id")
	if itemId == 0 {
		return Failure
	}
	for _, item := range user.Character.Items {
		if item.ItemId == itemId {
			user.Character.RemoveItem(item)
			return Success
		}
	}
	return Failure
}

// actGiveItemMultiple gives one or more copies of an item to the triggering user.
// params: item_id (int), count (int, default 1)
func actGiveItemMultiple(params map[string]any, ctx *EvalContext) Result {
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}
	itemId := getIntParam(params, "item_id")
	if itemId == 0 {
		return Failure
	}
	count := getIntParam(params, "count")
	if count <= 0 {
		count = 1
	}
	for i := 0; i < count; i++ {
		item := items.New(itemId)
		user.Character.StoreItem(item)
	}
	return Success
}

// actSetMiscData stores an arbitrary string value on the triggering user's
// character. Useful for quest-branch tracking via misc data keys.
// params: key (string), value (string)
func actSetMiscData(params map[string]any, ctx *EvalContext) Result {
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}
	key := getStringParam(params, "key")
	if key == "" {
		return Failure
	}
	value := getStringParam(params, "value")
	user.Character.SetMiscData(key, value)
	return Success
}
