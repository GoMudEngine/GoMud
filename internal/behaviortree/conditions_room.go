package behaviortree

import "strings"

// condCommandMatches checks if ctx.Event.Command matches any string in the
// "commands" list param.
func condCommandMatches(params map[string]any, ctx *EvalContext) Result {
	cmdList, ok := params["commands"]
	if !ok {
		return Failure
	}
	list, ok := cmdList.([]any)
	if !ok {
		return Failure
	}
	cmd := strings.ToLower(ctx.Event.Command)
	for _, v := range list {
		if s, ok := v.(string); ok && strings.ToLower(s) == cmd {
			return Success
		}
	}
	return Failure
}

// condCommandRestContains checks if ctx.Event.Rest contains any keyword from
// the "keywords" list param.
func condCommandRestContains(params map[string]any, ctx *EvalContext) Result {
	kwList, ok := params["keywords"]
	if !ok {
		return Failure
	}
	list, ok := kwList.([]any)
	if !ok {
		return Failure
	}
	rest := strings.ToLower(ctx.Event.Rest)
	for _, v := range list {
		if s, ok := v.(string); ok && strings.Contains(rest, strings.ToLower(s)) {
			return Success
		}
	}
	return Failure
}
