package combat

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// GrappleMoveResult wraps GrappleResult with disarm and failure penalty info.
// Callers use these fields to dispatch messaging and analytics.
type GrappleMoveResult struct {
	GrappleResult
	PositionDesc   string             // "clinched" or "grounded"
	DisarmResult   *DisarmResult      // nil if no disarm triggered
	CritFailure    *CritFailureResult // nil if no crit failure
	DefensePenalty bool               // true if z < 0.5 failure penalty applied
}

// ExecuteGrappleMove performs the full grapple command resolution:
// AttemptGrapple → ApplyGrappleResult → crit disarm check → failure penalties.
// Callers handle only messaging and analytics.
func ExecuteGrappleMove(attacker, defender *characters.Character,
	attackerId int, room *rooms.Room) GrappleMoveResult {

	result := GrappleMoveResult{}

	// Attempt the grapple
	result.GrappleResult = AttemptGrapple(attacker, defender)

	if result.Success {
		// Apply grapple positions
		ApplyGrappleResult(attacker, defender, result.GrappleResult, attackerId)

		// Determine position description
		result.PositionDesc = "clinched"
		if result.IsGroundGrapple {
			result.PositionDesc = "grounded"
		}

		// Check for grapple crit disarm (z > 2.0 in any grapple position)
		if result.AttackZScore > 2.0 {

			disarm := AttemptCritDisarm(attacker, defender, 15.0)
			if disarm.Success {
				result.DisarmResult = &disarm
			}
		}
	} else {
		// Simple failure (z < 0.5): Defense penalty
		if result.AttackZScore < 0.5 {
			attacker.AddCondition(characters.ConditionDefensePenalty, 1, 0.85, "failed grapple")
			result.DefensePenalty = true
		}

		// Critical failure (z < -2.0): Fall prone + reversal opportunity
		if result.AttackZScore < -2.0 {
			critResult := HandleGrappleCritFailure(attacker, defender)
			result.CritFailure = &critResult
		}
	}

	return result
}

// GrappleMoveDisarmWeapon returns the disarmed weapon item, if any.
// Convenience accessor for callers that need the item.
func (r *GrappleMoveResult) GrappleMoveDisarmWeapon() *items.Item {
	if r.DisarmResult == nil {
		return nil
	}
	return &r.DisarmResult.Weapon
}
