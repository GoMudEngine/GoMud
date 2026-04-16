package behaviortree

import (
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// ConditionFunc is the signature for all registered condition checks.
type ConditionFunc func(params map[string]any, ctx *EvalContext) Result

// conditionRegistry maps condition names to their implementations.
var conditionRegistry = map[string]ConditionFunc{}

func init() {
	conditionRegistry["keyword_match"] = condKeywordMatch
	conditionRegistry["player_has_quest"] = condPlayerHasQuest
	conditionRegistry["player_missing_quest"] = condPlayerMissingQuest
	conditionRegistry["player_has_item"] = condPlayerHasItem
	conditionRegistry["player_has_gold"] = condPlayerHasGold
	conditionRegistry["player_has_flag"] = condPlayerHasFlag
	conditionRegistry["mob_in_combat"] = condMobInCombat
	conditionRegistry["mob_health_below"] = condMobHealthBelow
	conditionRegistry["mob_at_home"] = condMobAtHome
	conditionRegistry["time_of_day"] = condTimeOfDay
	conditionRegistry["round_mod"] = condRoundMod
	conditionRegistry["random_chance"] = condRandomChance
	conditionRegistry["state_equals"] = condStateEquals
	conditionRegistry["players_in_room"] = condPlayersInRoom
	conditionRegistry["item_matches"] = condItemMatches
	conditionRegistry["mob_has_buff"] = condMobHasBuff
	conditionRegistry["player_has_spell"] = condPlayerHasSpell
	conditionRegistry["player_has_misc_data"] = condPlayerHasMiscData
	conditionRegistry["state_greater_than"] = condStateGreaterThan
	conditionRegistry["multiple_enemies"] = condMultipleEnemies
	conditionRegistry["command_matches"] = condCommandMatches
	conditionRegistry["command_rest_contains"] = condCommandRestContains
	conditionRegistry["mob_in_room"] = condMobInRoom
}

// LookupCondition returns the condition function for the given name,
// or nil if not found.
func LookupCondition(name string) ConditionFunc {
	return conditionRegistry[name]
}

// ConditionNode wraps a registered condition function with its params.
type ConditionNode struct {
	Name   string
	Params map[string]any
	Fn     ConditionFunc
}

func (n *ConditionNode) Evaluate(ctx *EvalContext) Result {
	return n.Fn(n.Params, ctx)
}

// --- condition implementations ---

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

func condMobInCombat(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	if mob.Character.Aggro != nil {
		return Success
	}
	return Failure
}

func condMobHealthBelow(params map[string]any, ctx *EvalContext) Result {
	pct := getIntParam(params, "percent")
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	maxHP := mob.Character.HealthMax.Value
	if maxHP <= 0 {
		return Failure
	}
	currentPct := mob.Character.Health * 100 / maxHP
	if currentPct < pct {
		return Success
	}
	return Failure
}

func condMobAtHome(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	if mob.Character.RoomId == mob.HomeRoomId {
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

func condMobHasBuff(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	buffId := getIntParam(params, "buff_id")
	if mob.Character.HasBuff(buffId) {
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

// condMobInRoom checks if a mob with the given template mob_id is present in
// the room.
func condMobInRoom(params map[string]any, ctx *EvalContext) Result {
	mobId := getIntParam(params, "mob_id")
	room := rooms.LoadRoom(ctx.RoomId)
	if room == nil {
		return Failure
	}
	for _, instId := range room.GetMobs(rooms.FindAll) {
		m := mobs.GetInstance(instId)
		if m != nil && int(m.MobId) == mobId {
			return Success
		}
	}
	return Failure
}
