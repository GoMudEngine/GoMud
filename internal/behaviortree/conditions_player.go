package behaviortree

import (
	"github.com/GoMudEngine/GoMud/internal/mobs"
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

// condPlayerInRoomMissingQuest succeeds when ANY player in the mob's
// room lacks the given quest token. Built for ambient/idle branches
// (mob_idle has no triggering player, so per-player conditions like
// player_missing_quest can't gate them) — e.g. the newbie-hub greeter
// only repeats his invitation while an un-Opened player is actually
// standing at the pool.
// params: quest (string token)
func condPlayerInRoomMissingQuest(params map[string]any, ctx *EvalContext) Result {
	quest := getStringParam(params, "quest")
	if quest == "" {
		return Failure
	}
	room := rooms.LoadRoom(ctx.RoomId)
	if room == nil {
		return Failure
	}
	for _, userId := range room.GetPlayers() {
		user := users.GetByUserId(userId)
		if user == nil {
			continue
		}
		if !user.Character.HasQuest(quest) {
			return Success
		}
	}
	return Failure
}

func condMultipleEnemies(params map[string]any, ctx *EvalContext) Result {
	room := rooms.LoadRoom(ctx.RoomId)
	if room == nil {
		return Failure
	}

	mob := mobs.GetInstance(ctx.InstanceId)
	charmedByUserId := 0
	if mob != nil {
		charmedByUserId = mob.Character.GetCharmedUserId()
	}

	count := 0

	// Players — skip the summoner if this is a charmed mob.
	for _, pId := range room.GetPlayers() {
		if charmedByUserId > 0 && pId == charmedByUserId {
			continue
		}
		count++
	}

	// Mobs — from a charmed mob's POV, fellow same-owner companions are
	// friends; count wild mobs + mobs charmed by someone else.
	// From a wild mob's POV, preserve original behavior (count charmed
	// companions; wild mobs don't count other wild mobs as enemies).
	for _, mId := range room.GetMobs() {
		if mob != nil && mId == mob.InstanceId {
			continue // don't count self
		}
		m := mobs.GetInstance(mId)
		if m == nil {
			continue
		}
		if charmedByUserId > 0 {
			// Charmed mob: skip fellow companions of same owner.
			if m.Character.IsCharmed(charmedByUserId) {
				continue
			}
			count++
		} else {
			// Wild mob: original behavior — only charmed companions count.
			if m.Character.IsCharmed() {
				count++
			}
		}
	}

	if count > 1 {
		return Success
	}
	return Failure
}
