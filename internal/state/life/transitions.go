package life

import "github.com/GoMudEngine/GoMud/internal/state"

// validTransitions enforces the Life invariant matrix.
// Vetoes layer additional rules on top.
//
// Dead → Alive is allowed for edge cases (admin restoration,
// hypothetical revive-from-Dead spell). Standard player respawn
// goes Dead → Respawning → Alive.
var validTransitions = state.TransitionTable[State]{
	Alive:      {Dead},
	Dead:       {Respawning, Alive},
	Respawning: {Alive},
}

// Trigger reason constants.
const (
	TriggerHealthZero      = "health_zero"
	TriggerSuicide         = "suicide_command"
	TriggerAdminKill       = "admin_kill"
	TriggerCleanupReady    = "cleanup_ready"
	TriggerRespawnReady    = "respawn_ready"
	TriggerRespawnComplete = "respawn_complete"
	TriggerForceAlive      = "force_alive"
)
