package perception

import "github.com/GoMudEngine/GoMud/internal/state"

// transitions enforces the Perception invariant matrix. Two states,
// two edges. Re-entry (Sighted→Sighted or Blinded→Blinded) is not in
// the table — callers should check current state before firing the
// transition to avoid ErrInvalidTransition.
var transitions = state.TransitionTable[State]{
	Sighted: {Blinded},
	Blinded: {Sighted},
}

// Trigger reason constants. Used in state.TransitionReason.Trigger.
const (
	TriggerBuffApplied      = "buff_applied"
	TriggerBuffExpired      = "buff_expired"
	TriggerConditionAdded   = "condition_added"
	TriggerConditionRemoved = "condition_removed"
)

// Blind-source buff IDs. Detected by ID rather than by flag because
// the existing buff YAMLs don't carry a "blinded" flag — they only
// have stat mods. Adding flags to the YAML would touch data; detecting
// by ID keeps chunk 6 dormant on the data side.
const (
	BuffIdBlinded            = 3  // _datafiles/world/dogmud/buffs/3-blinded.yaml
	BuffIdFlashbangBlindness = 77 // _datafiles/world/dogmud/buffs/77-flashbang_blindness.yaml
)
