package behaviortree

import (
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func condPlayerHasQuest(params map[string]any, ctx *EvalContext) Result {
	quest := getStringParam(params, "quest")
	if quest == "" {
		return Failure
	}
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}
	if user.Character.HasQuest(quest) {
		return Success
	}
	return Failure
}

func condPlayerMissingQuest(params map[string]any, ctx *EvalContext) Result {
	quest := getStringParam(params, "quest")
	if quest == "" {
		return Failure
	}
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}
	if !user.Character.HasQuest(quest) {
		return Success
	}
	return Failure
}

func condPlayerHasItem(params map[string]any, ctx *EvalContext) Result {
	itemId := getIntParam(params, "item_id")
	if itemId == 0 {
		return Failure
	}
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}
	for _, item := range user.Character.Items {
		if item.ItemId == itemId {
			return Success
		}
	}
	return Failure
}

func condPlayerHasGold(params map[string]any, ctx *EvalContext) Result {
	amount := getIntParam(params, "amount")
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}
	if user.Character.Gold >= amount {
		return Success
	}
	return Failure
}

func condPlayerHasFlag(params map[string]any, ctx *EvalContext) Result {
	key := getStringParam(params, "flag_key")
	value := getStringParam(params, "flag_value")
	if key == "" {
		return Failure
	}
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}
	if user.Character.GetQuestFlag(key) == value {
		return Success
	}
	return Failure
}

func condPlayerHasSpell(params map[string]any, ctx *EvalContext) Result {
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}
	spell := getStringParam(params, "spell")
	if user.Character.HasSpell(spell) {
		return Success
	}
	return Failure
}

func condPlayerHasMiscData(params map[string]any, ctx *EvalContext) Result {
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}
	key := getStringParam(params, "key")
	value := getStringParam(params, "value")
	actual, _ := user.Character.GetMiscData(key).(string)
	if actual == value {
		return Success
	}
	return Failure
}

func condPlayersInRoom(params map[string]any, ctx *EvalContext) Result {
	room := rooms.LoadRoom(ctx.RoomId)
	if room == nil {
		return Failure
	}
	if len(room.GetPlayers()) > 0 {
		return Success
	}
	return Failure
}

func condMultipleEnemies(params map[string]any, ctx *EvalContext) Result {
	room := rooms.LoadRoom(ctx.RoomId)
	if room == nil {
		return Failure
	}
	// Count players plus charmed companion mobs
	count := len(room.GetPlayers()) + len(room.GetMobs(rooms.FindCharmed))
	if count > 1 {
		return Success
	}
	return Failure
}
