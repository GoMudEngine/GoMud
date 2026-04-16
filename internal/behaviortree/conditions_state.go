package behaviortree

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/util"
)

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

func condStateGreaterThan(params map[string]any, ctx *EvalContext) Result {
	if ctx.MobState == nil {
		return Failure
	}
	key := getStringParam(params, "key")
	threshold := getIntParam(params, "value")
	if ctx.MobState.GetInt(key) > threshold {
		return Success
	}
	return Failure
}

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

func condItemMatches(params map[string]any, ctx *EvalContext) Result {
	itemId := getIntParam(params, "item_id")
	if ctx.Event.ItemId == itemId {
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
