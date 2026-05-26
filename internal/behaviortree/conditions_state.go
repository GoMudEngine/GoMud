package behaviortree

import (
	"strconv"
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
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

// loggedTimeOfDayMisconfigs tracks already-logged misconfigured range
// strings so warnings don't spam every btree tick. sync.Map gives
// lock-free reads for this low-cardinality set.
var loggedTimeOfDayMisconfigs sync.Map

func condTimeOfDay(params map[string]any, ctx *EvalContext) Result {
	rangeStr := getStringParam(params, "range")

	// range parameter takes precedence over period when both are set.
	if rangeStr != "" {
		start, end, valid := parseHourRange(rangeStr)
		if !valid {
			if _, already := loggedTimeOfDayMisconfigs.LoadOrStore("err:"+rangeStr, true); !already {
				mudlog.Error("time_of_day",
					"error", "invalid `range` parameter",
					"value", rangeStr)
			}
			return Failure
		}

		// Empty range (start == end) always Failure.
		if start == end {
			if _, already := loggedTimeOfDayMisconfigs.LoadOrStore("empty:"+rangeStr, true); !already {
				mudlog.Warn("time_of_day",
					"warn", "empty `range` (start == end) always returns Failure",
					"value", rangeStr)
			}
			return Failure
		}

		// Full-day range (0-24) always Success.
		if start == 0 && end == 24 {
			if _, already := loggedTimeOfDayMisconfigs.LoadOrStore("full:"+rangeStr, true); !already {
				mudlog.Warn("time_of_day",
					"warn", "full-day `range` (0-24) always returns Success — consider removing the gate",
					"value", rangeStr)
			}
			return Success
		}

		hour := gametime.GetDate().Hour24
		if inHourRange(hour, start, end) {
			return Success
		}
		return Failure
	}

	// Existing binary form (period: day / period: night) — unchanged.
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

// parseHourRange parses a "start-end" hour string. Hours are in 0-23
// 24h format; end=24 is allowed to mean "end of day". Returns
// (start, end, true) on success; (0, 0, false) on any malformed input.
//
// Wrap-around (start > end, e.g., "22-6") is valid and handled by
// inHourRange.
func parseHourRange(s string) (int, int, bool) {
	parts := strings.SplitN(s, "-", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	startStr := strings.TrimSpace(parts[0])
	endStr := strings.TrimSpace(parts[1])
	if startStr == "" || endStr == "" {
		return 0, 0, false
	}
	start, err := strconv.Atoi(startStr)
	if err != nil {
		return 0, 0, false
	}
	end, err := strconv.Atoi(endStr)
	if err != nil {
		return 0, 0, false
	}
	if start < 0 || start > 23 {
		return 0, 0, false
	}
	if end < 0 || end > 24 {
		return 0, 0, false
	}
	return start, end, true
}

// inHourRange reports whether hour falls within [start, end). Handles
// wrap-around when start > end (e.g., 22-6 covers 22, 23, 0, 1, 2, 3, 4, 5).
func inHourRange(hour, start, end int) bool {
	if start <= end {
		return hour >= start && hour < end
	}
	// Wrap-around: in range if at or after start, OR strictly before end.
	return hour >= start || hour < end
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

// condMobAtTargetRoom returns Success when the mob is at the room its current
// schedule segment names as target_room. Returns Failure when the mob has no
// schedule, no current segment, or is in transit. See chunk 3.2 spec.
func condMobAtTargetRoom(_ map[string]any, ctx *EvalContext) Result {
	if ctx == nil {
		return Failure
	}
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil || mob.ScheduleId == "" {
		return Failure
	}
	s := mobs.GetSchedule(mob.ScheduleId)
	if s == nil {
		return Failure
	}
	seg := s.CurrentSegment(gametime.GetDate().Hour24)
	if seg == nil {
		return Failure
	}
	if mob.Character.RoomId == seg.TargetRoom {
		return Success
	}
	return Failure
}
