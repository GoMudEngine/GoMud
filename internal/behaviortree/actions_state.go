package behaviortree

// actions_state.go — MobState read/write actions:
// actSetState, actIncrementState, actDecrementState

func actSetState(params map[string]any, ctx *EvalContext) Result {
	if ctx.MobState == nil {
		return Failure
	}
	key := getStringParam(params, "key")
	if key == "" {
		return Failure
	}
	value := params["value"]
	ctx.MobState.Set(key, value)
	return Success
}

// actIncrementState adds amount (default 1) to a numeric MobState key.
// params: key (string), amount (int, default 1)
func actIncrementState(params map[string]any, ctx *EvalContext) Result {
	if ctx.MobState == nil {
		return Failure
	}
	key := getStringParam(params, "key")
	if key == "" {
		return Failure
	}
	amount := getIntParam(params, "amount")
	if amount == 0 {
		amount = 1
	}
	ctx.MobState.Set(key, ctx.MobState.GetInt(key)+amount)
	return Success
}

// actDecrementState subtracts amount (default 1) from a numeric MobState key.
// params: key (string), amount (int, default 1)
func actDecrementState(params map[string]any, ctx *EvalContext) Result {
	if ctx.MobState == nil {
		return Failure
	}
	key := getStringParam(params, "key")
	if key == "" {
		return Failure
	}
	amount := getIntParam(params, "amount")
	if amount == 0 {
		amount = 1
	}
	ctx.MobState.Set(key, ctx.MobState.GetInt(key)-amount)
	return Success
}
