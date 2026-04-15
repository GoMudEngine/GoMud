package behaviortree

import (
	"github.com/GoMudEngine/GoMud/internal/util"
)

// CooldownDecorator skips its child if it was last run within N rounds.
type CooldownDecorator struct {
	Rounds   int
	StateKey string // unique key for tracking last-run round in BehaviorState
	Child    Node
}

func (d *CooldownDecorator) Evaluate(ctx *EvalContext) Result {
	lastRun := ctx.MobState.GetInt(d.StateKey)
	currentRound := int(util.GetRoundCount())
	if currentRound-lastRun < d.Rounds {
		return Failure
	}
	result := d.Child.Evaluate(ctx)
	if result == Success {
		ctx.MobState.Set(d.StateKey, currentRound)
	}
	return result
}

// RepeatDecorator runs its child N times.
type RepeatDecorator struct {
	Times int
	Child Node
}

func (d *RepeatDecorator) Evaluate(ctx *EvalContext) Result {
	for i := 0; i < d.Times; i++ {
		result := d.Child.Evaluate(ctx)
		if result == Failure {
			return Failure
		}
	}
	return Success
}

// InvertDecorator flips Success↔Failure.
type InvertDecorator struct {
	Child Node
}

func (d *InvertDecorator) Evaluate(ctx *EvalContext) Result {
	result := d.Child.Evaluate(ctx)
	if result == Success {
		return Failure
	}
	if result == Failure {
		return Success
	}
	return Running
}

// RandomDecorator runs its child with N% probability.
type RandomDecorator struct {
	Percent int
	Child   Node
}

func (d *RandomDecorator) Evaluate(ctx *EvalContext) Result {
	if util.Rand(100) >= d.Percent {
		return Failure
	}
	return d.Child.Evaluate(ctx)
}

// DelayDecorator waits N rounds using BehaviorState to track start.
type DelayDecorator struct {
	Rounds   int
	StateKey string
	Child    Node
}

func (d *DelayDecorator) Evaluate(ctx *EvalContext) Result {
	startRound := ctx.MobState.GetInt(d.StateKey)
	currentRound := int(util.GetRoundCount())
	if startRound == 0 {
		ctx.MobState.Set(d.StateKey, currentRound)
		return Running
	}
	if currentRound-startRound < d.Rounds {
		return Running
	}
	ctx.MobState.Delete(d.StateKey)
	return d.Child.Evaluate(ctx)
}
