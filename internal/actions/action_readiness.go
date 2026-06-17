package actions

import (
	"strings"
)

// ReadyStatus classifies whether a command can fire now.
type ReadyStatus int

const (
	ActionReady    ReadyStatus = iota // fire it now
	ActionDeferred                    // transiently blocked — retry later, no player text
	ActionRejected                    // permanently invalid — surface the real error, then drop
)

// ReadinessResult is the verdict from ActionReadiness.
type ReadinessResult struct {
	Status ReadyStatus
	Reason string // short tag for logging/telemetry; never shown to the player
}

// specialMoveVerbs mirrors the switch in CommandIsReady (command_readiness.go).
// SYNC POINT: keep in lockstep with that switch; TestActionReadinessDrift enforces it.
var specialMoveVerbs = map[string]bool{
	"taunt": true, "rally": true, "warcry": true, "trip": true, "bash": true,
	"grapple": true, "kick": true, "rake": true, "maul": true, "throttle": true,
	"pounce": true, "gore": true, "hamstring": true, "drain": true,
}

func splitVerb(cmd string) (verb, rest string) {
	parts := strings.SplitN(strings.TrimSpace(cmd), " ", 2)
	verb = strings.ToLower(parts[0])
	if len(parts) > 1 {
		rest = strings.TrimSpace(parts[1])
	}
	return verb, rest
}

// ActionReadiness reports whether cmd can fire right now for actor, and whether
// any blocker is transient (Deferred) or permanent (Rejected). Commands it does
// not specifically gate return ActionReady and execute verbatim.
func ActionReadiness(actor Actor, cmd string) ReadinessResult {
	if actor == nil {
		return ReadinessResult{ActionRejected, "no actor"}
	}
	char := actor.GetCharacter()
	if char == nil {
		return ReadinessResult{ActionRejected, "no character"}
	}

	verb, _ := splitVerb(cmd)

	if specialMoveVerbs[verb] {
		if CommandIsReady(actor, verb) {
			return ReadinessResult{ActionReady, ""}
		}
		// Transient: shared cooldown or an active Activity (cast/craft/salvage).
		if char.IsActing() || char.GetCooldown("special-move") > 0 {
			return ReadinessResult{ActionDeferred, "special-move busy"}
		}
		// Structural: missing body part / wrong species / no valid target.
		return ReadinessResult{ActionRejected, "special-move unavailable"}
	}

	return ReadinessResult{ActionReady, ""}
}
