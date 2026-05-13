package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/combat"
)

// ConsiderResult is the structured output of an actor's consider
// action. Ratio is self power divided by target power; values > 1
// mean the actor outclasses the target.
type ConsiderResult struct {
	Ratio          float64 // 0 if target power is 0 (degenerate)
	SelfPower      float64
	TargetPower    float64
	TargetName     string
	TargetIsPlayer bool
}

// Consider computes a power-ratio assessment of target from
// actor's POV. For UserActor: also formats a colored prediction
// string and calls actor.SendText(...). For MobActor: SendText
// is a no-op (existing actor abstraction), so the math runs
// silently. Triggers OnStatUse("perception") on the actor.
func Consider(actor Actor, target Actor) ConsiderResult {
	actor.OnStatUse("perception")

	selfChar := actor.GetCharacter()
	targetChar := target.GetCharacter()

	selfPower := combat.PowerScore(*selfChar)
	targetPower := combat.PowerScore(*targetChar)

	result := ConsiderResult{
		SelfPower:      selfPower,
		TargetPower:    targetPower,
		TargetName:     target.GetName(),
		TargetIsPlayer: target.IsPlayer(),
	}
	if targetPower > 0 {
		result.Ratio = selfPower / targetPower
	}

	// Format and emit prediction text. UserActor delivers to the
	// player connection; MobActor.SendText is a no-op.
	considerType := "mob"
	if result.TargetIsPlayer {
		considerType = "user"
	}
	actor.SendText(fmt.Sprintf(
		`You consider <ansi fg="%sname">%s</ansi>...`,
		considerType, result.TargetName))
	actor.SendText(fmt.Sprintf(
		`Your instincts tell you: %s`, predictionFor(result.Ratio)))

	return result
}

// predictionFor maps a power ratio to the canonical prediction
// text + color. Ratio = 0 (degenerate target) is treated as
// "will not survive" — preserved verbatim from the pre-refactor
// usercommands.Consider behavior.
func predictionFor(ratio float64) string {
	switch {
	case ratio > 4:
		return `<ansi fg="blue-bold">They pose no threat to you</ansi>`
	case ratio > 3:
		return `<ansi fg="green">You hold a clear advantage</ansi>`
	case ratio > 2:
		return `<ansi fg="green">The odds favor you</ansi>`
	case ratio > 1:
		return `<ansi fg="yellow">An even contest — tread carefully</ansi>`
	case ratio > 0.5:
		return `<ansi fg="red-bold">They have the upper hand</ansi>`
	case ratio > 0:
		return `<ansi fg="red-bold">You are severely outmatched</ansi>`
	default:
		return `<ansi fg="red-bold">You will not survive this fight</ansi>`
	}
}
