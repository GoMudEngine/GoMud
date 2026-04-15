package mobai

import (
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
)

// TriggerContext holds the current state used to evaluate triggers.
type TriggerContext struct {
	MobChar           *characters.Character
	Target            *characters.Character
	EnemyCount        int
	HasAggro          bool
	HasMemory         bool
	CombatJustStarted bool
	LastAction        string
	PlayerEntered     bool
	IsHidden          bool
	ActiveBuffIds     []int
}

// EvalTrigger checks whether a trigger string matches the current context.
func EvalTrigger(trigger string, ctx *TriggerContext) bool {
	parts := strings.SplitN(trigger, ":", 2)
	triggerName := parts[0]
	triggerParam := ""
	if len(parts) > 1 {
		triggerParam = parts[1]
	}

	switch triggerName {
	case "combat_start":
		return ctx.CombatJustStarted
	case "target_casting":
		return ctx.Target != nil && ctx.Target.IsCasting()
	case "target_prone":
		return ctx.Target != nil && ctx.Target.CombatPosition == characters.PositionProne
	case "target_grappled":
		return ctx.Target != nil && ctx.Target.CombatPosition.IsGrapplePosition()
	case "health_below":
		if ctx.MobChar == nil || triggerParam == "" {
			return false
		}
		pct, err := strconv.Atoi(triggerParam)
		if err != nil {
			return false
		}
		if ctx.MobChar.HealthMax.Value <= 0 {
			return false
		}
		currentPct := (ctx.MobChar.Health * 100) / ctx.MobChar.HealthMax.Value
		return currentPct < pct
	case "multiple_targets":
		return ctx.EnemyCount > 1
	case "single_target":
		return ctx.EnemyCount == 1
	case "target_fled":
		return !ctx.HasAggro && ctx.HasMemory
	case "not_hidden":
		return !ctx.IsHidden
	case "no_aggro":
		return !ctx.HasAggro && ctx.HasMemory
	case "after_action":
		return ctx.LastAction == triggerParam
	case "player_entered":
		return ctx.PlayerEntered
	case "has_buff":
		if triggerParam == "" {
			return false
		}
		buffId, err := strconv.Atoi(triggerParam)
		if err != nil {
			return false
		}
		for _, id := range ctx.ActiveBuffIds {
			if id == buffId {
				return true
			}
		}
		return false
	case "missing_buff":
		if triggerParam == "" {
			return false
		}
		buffId, err := strconv.Atoi(triggerParam)
		if err != nil {
			return false
		}
		for _, id := range ctx.ActiveBuffIds {
			if id == buffId {
				return false
			}
		}
		return true
	default:
		return false
	}
}
