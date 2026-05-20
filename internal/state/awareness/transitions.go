package awareness

import "github.com/GoMudEngine/GoMud/internal/state"

// validTransitions enforces the Awareness invariant matrix.
// Vetoes layer additional rules on top.
var validTransitions = state.TransitionTable[State]{
	Visible:    {Concealing},
	Concealing: {Hidden, Visible}, // success vs failure
	Hidden:     {Revealing, Visible}, // Revealing for cascade; Visible for force (logout/death)
	Revealing:  {Visible},
}

// Trigger reason constants.
const (
	TriggerSneakCommand       = "sneak_command"
	TriggerSneakSuccess       = "sneak_success"
	TriggerSneakFailed        = "sneak_failed"
	TriggerCombatEntered      = "combat_entered"
	TriggerSurpriseRoundEnd   = "surprise_round_end"
	TriggerMovementDetected   = "movement_detected"
	TriggerObserverSearch     = "observer_search"
	TriggerLightChange        = "light_change"
	TriggerSkullduggeryFailed = "skullduggery_failed"
	TriggerNoisyAction        = "noisy_action"
	TriggerLogout             = "logout_safety_valve"
	TriggerDeath              = "death_cascade"
	TriggerForceVisible       = "force_visible"
)
