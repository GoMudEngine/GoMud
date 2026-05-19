// Vision-related helpers on Character. Currently houses HasAnyBlindSource
// for chunk 6's Perception machine. Future messaging framework will add
// CanSee / CanSeeClearly / CanSeeShapes here.
package characters

import (
	"github.com/GoMudEngine/GoMud/internal/state/perception"
)

// HasAnyBlindSource returns true if any active blind source is currently
// affecting this character. Used by Perception expire-paths in AddBuff,
// RemoveBuff, AddCondition, RemoveCondition to decide whether to fire
// the Blinded→Sighted transition when one of multiple overlapping
// sources clears.
//
// Sources checked:
//   - Buff 3 (Blinded) — _datafiles/world/dogmud/buffs/3-blinded.yaml
//   - Buff 77 (Flashbang Blindness) — _datafiles/world/dogmud/buffs/77-flashbang_blindness.yaml
//   - ConditionBlinded — currently applied by blinding-flash and
//     blinding-spit mutations (see usercommands/mutation_blinding_*.go).
//
// Note: uses TriggersLeft > 0 rather than HasBuff to correctly detect the
// "just removed" state. RemoveBuff marks a buff expired (TriggersLeft=0) but
// defers the actual map-entry prune to the next game-tick Advance call.
// HasBuff checks map membership only and returns true for expired buffs;
// TriggersLeft > 0 returns false immediately after RemoveBuff fires.
func (c *Character) HasAnyBlindSource() bool {
	if c == nil {
		return false
	}
	if c.Buffs.TriggersLeft(perception.BuffIdBlinded) > 0 {
		return true
	}
	if c.Buffs.TriggersLeft(perception.BuffIdFlashbangBlindness) > 0 {
		return true
	}
	if c.HasCondition(ConditionBlinded) {
		return true
	}
	return false
}
