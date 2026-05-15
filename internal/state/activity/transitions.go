package activity

import "github.com/GoMudEngine/GoMud/internal/state"

// validTransitions is the star-topology Activity transition table.
// Every active state can return to Free; Free can become any active
// state. No direct active-to-active transitions (cancel-then-start
// enforces serialization, and cross-activity-start veto is enforced
// by call-site IsFree() checks).
var validTransitions = state.TransitionTable[State]{
	Free:      {Casting, Crafting, Salvaging},
	Casting:   {Free},
	Crafting:  {Free},
	Salvaging: {Free},
}

// Trigger reason constants.
const (
	// Free → active
	TriggerCastBegin    = "cast_begin"
	TriggerCraftBegin   = "craft_begin"
	TriggerSalvageBegin = "salvage_begin"

	// active → Free, success
	TriggerCastComplete    = "cast_complete"
	TriggerCraftComplete   = "craft_complete"
	TriggerSalvageComplete = "salvage_complete"

	// active → Free, user-initiated
	TriggerCastCancel    = "cast_cancel"
	TriggerCraftCancel   = "craft_cancel"
	TriggerSalvageCancel = "salvage_cancel"

	// active → Free, externally induced
	TriggerConcentrationBreak = "concentration_break" // Casting only
	TriggerCombatInterrupt    = "combat_interrupt"    // Crafting / Salvaging
	TriggerMovementInterrupt  = "movement_interrupt"  // Crafting / Salvaging
	TriggerDamageInterrupt    = "damage_interrupt"    // Crafting / Salvaging (hard cancel, no roll)
	TriggerDeath              = "death"               // cascade from Life
)
