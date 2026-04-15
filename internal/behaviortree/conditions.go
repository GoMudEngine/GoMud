package behaviortree

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
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

// --- helpers ---

func getIntParam(params map[string]any, key string) int {
	switch v := params[key].(type) {
	case int:
		return v
	case float64:
		return int(v)
	}
	return 0
}

func getStringParam(params map[string]any, key string) string {
	v, _ := params[key].(string)
	return v
}

// --- condition implementations ---

func condKeywordMatch(params map[string]any, ctx *EvalContext) Result {
	keywordsRaw, ok := params["keywords"].([]any)
	if !ok || len(keywordsRaw) == 0 {
		return Failure
	}

	words := strings.Fields(strings.ToLower(ctx.Event.Text))
	for _, kw := range keywordsRaw {
		kwStr, ok := kw.(string)
		if !ok {
			continue
		}
		kwLower := strings.ToLower(kwStr)
		for _, w := range words {
			if w == kwLower {
				return Success
			}
		}
	}
	return Failure
}

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

func condTimeOfDay(params map[string]any, ctx *EvalContext) Result {
	period := getStringParam(params, "period")
	isNight := gametime.IsNight()
	switch strings.ToLower(period) {
	case "night":
		if isNight {
			return Success
		}
	case "day":
		if !isNight {
			return Success
		}
	}
	return Failure
}

func condRoundMod(params map[string]any, ctx *EvalContext) Result {
	n := getIntParam(params, "n")
	if n <= 0 {
		return Failure
	}
	if util.GetRoundCount()%uint64(n) == 0 {
		return Success
	}
	return Failure
}

func condRandomChance(params map[string]any, ctx *EvalContext) Result {
	pct := getIntParam(params, "percent")
	if pct <= 0 {
		return Failure
	}
	if pct >= 100 {
		return Success
	}
	if util.Rand(100) < pct {
		return Success
	}
	return Failure
}

func condStateEquals(params map[string]any, ctx *EvalContext) Result {
	key := getStringParam(params, "key")
	value := getStringParam(params, "value")
	if ctx.MobState == nil {
		return Failure
	}
	if ctx.MobState.GetString(key) == value {
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

func condItemMatches(params map[string]any, ctx *EvalContext) Result {
	itemId := getIntParam(params, "item_id")
	if ctx.Event.ItemId == itemId {
		return Success
	}
	return Failure
}
