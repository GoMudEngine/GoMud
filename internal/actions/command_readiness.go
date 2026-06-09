package actions

import "github.com/GoMudEngine/GoMud/internal/species"

// CommandIsReady returns true iff the named mob command would actually
// execute its effect right now. Mirrors the early-return gates in each
// Execute* function so the behavior tree's command_best_of action can
// self-gate and fall through cleanly.
//
// Takes an Actor (not a *mobs.Mob) so player-side callers can self-gate
// too if needed in the future. Actor also gives us GetCharacter() for
// the universal state checks.
//
// Unknown command names return false. This lets a behavior tree safely
// include commands that don't exist yet without firing a spurious
// Success.
//
// SYNC POINT: if this function gains a new gate, update the
// corresponding Execute* function. Drift is caught by
// TestCommandReadinessDrift in command_readiness_drift_test.go.
func CommandIsReady(actor Actor, cmd string) bool {
	if actor == nil {
		return false
	}
	char := actor.GetCharacter()
	if char == nil {
		return false
	}

	// Universal gates (apply to every command).
	// IsActing() blocks bash/kick/taunt/rally/warcry/trip/grapple while
	// Casting, Crafting, or Salvaging — any active Activity state.
	if char.IsActing() {
		return false
	}
	if char.GetCooldown("special-move") > 0 {
		return false
	}

	switch cmd {
	case "taunt":
		return char.Aggro != nil

	case "rally":
		return !char.HasBuff(80)

	case "warcry":
		return !char.HasBuff(79)

	case "trip":
		if char.Aggro == nil {
			return false
		}
		target := ResolveAggroTarget(char.Aggro)
		if !target.Found {
			return false
		}
		return !target.Char.IsOnFloor()

	case "bash":
		if char.Aggro == nil {
			return false
		}
		naturalBash := false
		if sp := species.GetSpecies(char.SpeciesId); sp != nil {
			naturalBash = sp.NaturalBash
		}
		if !char.HasShield() && !naturalBash {
			return false
		}
		if !char.HasBodyPart("arms") && !naturalBash {
			return false
		}
		return true

	case "grapple":
		if char.Aggro == nil {
			return false
		}
		target := ResolveAggroTarget(char.Aggro)
		if !target.Found {
			return false
		}
		if !char.HasBodyPart("arms") {
			return false
		}
		return !target.Char.IsGrappling()

	case "kick":
		return char.Aggro != nil
	}

	return false
}
