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
func (c *Character) HasAnyBlindSource() bool {
	if c == nil {
		return false
	}
	if c.Buffs.HasBuff(perception.BuffIdBlinded) {
		return true
	}
	if c.Buffs.HasBuff(perception.BuffIdFlashbangBlindness) {
		return true
	}
	if c.HasCondition(ConditionBlinded) {
		return true
	}
	return false
}
