package behaviortree

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
