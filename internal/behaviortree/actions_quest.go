package behaviortree

// actions_quest.go — quest, item, and gold actions:
// actGrantQuest, actSetQuestFlag, actGrantMutation, actGiveGold, actTakeGold,
// actGiveItem, actReturnItem, actTakeItem, actGiveItemMultiple, actSetMiscData

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/users"
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
	user.Character.GiveQuestToken(quest)
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
	sp := species.GetSpecies(user.Character.SpeciesId)
	pool := mutations.GetWeightedPool(user.Character.Mutations, sp)
	if len(pool) == 0 {
		return Success // no mutations available, but not an error
	}
	mutId := mutations.RollAcquisition(pool)
	if mutId == "" {
		return Success
	}
	if user.Character.Mutations == nil {
		user.Character.Mutations = make(map[string]int)
	}
	user.Character.Mutations[mutId] = 1
	user.Character.Validate()
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
	user.SendTextLegacy(fmt.Sprintf("You receive %d gold.\n", amount))
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
	user.SendTextLegacy(fmt.Sprintf("You receive a %s.\n", item.Name()))
	return Success
}

// actReturnItem gives a fresh copy of the triggering event's item back
// to the player. Used in player_give handlers to reject/return items.
// Unlike give_item, no item_id param is needed — the item comes from
// ctx.Event.ItemId.
func actReturnItem(params map[string]any, ctx *EvalContext) Result {
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}
	itemId := ctx.Event.ItemId
	if itemId == 0 {
		return Failure
	}
	item := items.New(itemId)
	if !user.Character.StoreItem(item) {
		return Failure
	}
	user.SendTextLegacy(fmt.Sprintf("%s hands back the %s.\n", ctx.MobName, item.Name()))
	return Success
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
