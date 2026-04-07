package mobai

import "sort"

var presets = map[string][]TacticRule{
	"aggressive_melee": {
		{Trigger: "target_prone", Action: "kick", Priority: 10},
		{Trigger: "target_casting", Action: "bash", Priority: 9},
		{Trigger: "target_grappled", Action: "submit", Priority: 8},
	},
	"defensive_caster": {
		{Trigger: "missing_buff:2", Action: "cast chrysalis-cocoon", Priority: 10},
		{Trigger: "multiple_targets", Action: "cast conviction-barrage", Priority: 9},
		{Trigger: "health_below:30", Action: "flee", Priority: 8},
		{Trigger: "single_target", Action: "cast conviction-spike", Priority: 5},
	},
	"ambusher": {
		{Trigger: "after_action:surprise-strike", Action: "flee", Priority: 10},
		{Trigger: "no_aggro", Action: "track_memory", Priority: 9},
		{Trigger: "not_hidden", Action: "hide", Priority: 8},
		{Trigger: "target_casting", Action: "trip", Priority: 7},
	},
	"tank": {
		{Trigger: "target_casting", Action: "bash", Priority: 10},
		{Trigger: "target_prone", Action: "kick", Priority: 9},
		{Trigger: "health_below:20", Action: "call_for_help", Priority: 8},
	},
}

// GetPreset returns a copy of the named preset rule list, or nil if unknown.
func GetPreset(name string) []TacticRule {
	p, ok := presets[name]
	if !ok {
		return nil
	}
	result := make([]TacticRule, len(p))
	copy(result, p)
	return result
}

// MergeTactics combines preset rules with mob-specific custom rules.
func MergeTactics(preset []TacticRule, custom []TacticRule) []TacticRule {
	merged := make([]TacticRule, 0, len(preset)+len(custom))
	merged = append(merged, preset...)
	merged = append(merged, custom...)
	return merged
}

// EvaluateTactics sorts rules by priority (descending) and returns the action
// of the first rule whose trigger fires in the given context.
func EvaluateTactics(tactics []TacticRule, ctx *TriggerContext) string {
	if len(tactics) == 0 {
		return ""
	}
	sorted := make([]TacticRule, len(tactics))
	copy(sorted, tactics)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority > sorted[j].Priority
	})
	for _, rule := range sorted {
		if EvalTrigger(rule.Trigger, ctx) {
			return rule.Action
		}
	}
	return ""
}
