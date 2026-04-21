package actions

import (
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// CommandIsReady returns true iff the named mob command would actually
// execute its effect right now. Checks cover: shared special-move
// cooldown, aggro target (where required), and target state (where
// relevant — e.g., trip needs standing target, grapple needs non-
// clinched target). Resource costs (stamina/conviction) are NOT checked
// — if a mob is resource-starved the command will no-op at execution
// time. For v1 this is acceptable because the shared cooldown is the
// dominant gate.
//
// Unknown command names return false. This lets a behavior tree safely
// include commands that don't exist yet without firing a spurious
// Success.
func CommandIsReady(mob *mobs.Mob, cmd string) bool {
	if mob == nil {
		return false
	}
	char := &mob.Character

	// All supported commands share the special-move cooldown.
	if char.GetCooldown("special-move") > 0 {
		return false
	}

	switch cmd {
	case "taunt":
		return char.Aggro != nil

	case "rally", "warcry":
		return true

	case "trip":
		if char.Aggro == nil {
			return false
		}
		target := ResolveAggroTarget(char.Aggro)
		if !target.Found {
			return false
		}
		return !target.Char.CombatPosition.IsGroundPosition()

	case "bash":
		if char.Aggro == nil {
			return false
		}
		return char.HasShield()

	case "grapple":
		if char.Aggro == nil {
			return false
		}
		target := ResolveAggroTarget(char.Aggro)
		if !target.Found {
			return false
		}
		return !target.Char.CombatPosition.IsGrapplePosition()

	case "kick":
		return char.Aggro != nil
	}

	return false
}
