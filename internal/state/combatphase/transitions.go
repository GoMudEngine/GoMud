package combatphase

import "github.com/GoMudEngine/GoMud/internal/state"

// validTransitions enforces the Combat Phase invariant matrix.
// Vetoes layer additional rules on top.
var validTransitions = state.TransitionTable[State]{
	Idle:        {Engaging},
	Engaging:    {Engaged, Idle},     // Idle on cancel/target-died
	Engaged:     {Disengaging, Idle}, // Idle direct on death/despawn
	Disengaging: {Idle, Engaged},     // Engaged on flee failure
}

// Trigger reason constants. Use these in TransitionReason.Trigger
// to ensure stable strings across the codebase.
const (
	TriggerAttackCommand   = "attack_command"
	TriggerSurpriseAttack  = "surprise_attack"
	TriggerEngagementReady = "engagement_ready"
	TriggerFleeCommand     = "flee_command"
	TriggerFleeSuccess     = "flee_success"
	TriggerFleeFailure     = "flee_failure"
	TriggerTargetDied      = "target_died"
	TriggerSelfDied        = "self_died"
	TriggerTargetOutOfRoom = "target_out_of_room"
	TriggerCharm           = "charm_acquired"
	TriggerCombatantToggle = "combatant_toggle"
	TriggerForceIdle       = "force_idle"
)
