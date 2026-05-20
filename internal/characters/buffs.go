package characters

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/perception"
)

func (c *Character) IsDisabled() bool {
	return c.Health <= 0
}

func (c *Character) HasBuffFlag(buffFlag buffs.Flag) bool {
	return c.Buffs.HasFlag(buffFlag, false)
}

// HasFlagFromAnySource returns true if the character has the given flag from
// either active buffs OR permanent mutation effects. Use this instead of
// HasBuffFlag when the check should also honor mutation-granted flags.
func (c *Character) HasFlagFromAnySource(buffFlag buffs.Flag) bool {
	if c.Buffs.HasFlag(buffFlag, false) {
		return true
	}
	return mutations.HasMutationFlag(c.Mutations, string(buffFlag))
}

func (c *Character) CancelBuffsWithFlag(buffFlag buffs.Flag) bool {
	if c.Buffs.HasFlag(buffFlag, true) {
		c.Validate(true)
		return true
	}
	return false
}

// CancelCombatBuffs cancels all active buffs with the CancelIfCombat flag
// AND strips matching buffs from the permaBuffIds list so they don't
// re-apply during Validate(). Call this when a character enters combat
// (as attacker or defender) or dies.
//
// Without the permabuff strip, mobs seeded with CancelIfCombat-tagged
// buffs via `buffids:` (e.g. Hidden on ambushers) would see the buff
// re-applied every Validate() call — the combat system would strip the
// active instance but the next Validate would put it right back. This
// surfaced as "(hidden)" tags persisting on ambushers mid-combat.
func (c *Character) CancelCombatBuffs() {
	filtered := make([]int, 0, len(c.permaBuffIds))
	for _, id := range c.permaBuffIds {
		spec := buffs.GetBuffSpec(id)
		if spec == nil {
			filtered = append(filtered, id)
			continue
		}
		keep := true
		for _, f := range spec.Flags {
			if f == buffs.CancelIfCombat {
				keep = false
				break
			}
		}
		if keep {
			filtered = append(filtered, id)
		}
	}
	c.permaBuffIds = filtered

	c.CancelBuffsWithFlag(buffs.CancelIfCombat)
}

func (c *Character) HasBuff(buffId int) bool {
	return c.Buffs.HasBuff(buffId)
}

func (c *Character) AddBuff(buffId int, isPermanent bool) error {
	buffId = int(math.Abs(float64(buffId)))
	if !c.Buffs.AddBuff(buffId, isPermanent) {
		return fmt.Errorf(`failed to add buff. target: "%s" buffId: %d`, c.Name, buffId)
	}
	// Chunk 6 (Perception): blind-source buffs trigger Sighted → Blinded.
	// Guard against re-entry: only fire if state is currently Sighted.
	if (buffId == perception.BuffIdBlinded || buffId == perception.BuffIdFlashbangBlindness) &&
		c.Perception != nil && c.Perception.State() == perception.Sighted {
		_ = c.Perception.TransitionTo(perception.Blinded,
			state.TransitionReason{Trigger: perception.TriggerBuffApplied, Metadata: map[string]any{"buffId": buffId}})
	}
	c.Validate()
	return nil
}

// AddBuffScaled adds a buff with its duration scaled by durationMult.
func (c *Character) AddBuffScaled(buffId int, durationMult float64) error {
	buffId = int(math.Abs(float64(buffId)))
	if !c.Buffs.AddBuffScaled(buffId, durationMult) {
		return fmt.Errorf(`failed to add buff. target: "%s" buffId: %d`, c.Name, buffId)
	}
	// Chunk 6 (Perception): see AddBuff above.
	if (buffId == perception.BuffIdBlinded || buffId == perception.BuffIdFlashbangBlindness) &&
		c.Perception != nil && c.Perception.State() == perception.Sighted {
		_ = c.Perception.TransitionTo(perception.Blinded,
			state.TransitionReason{Trigger: perception.TriggerBuffApplied, Metadata: map[string]any{"buffId": buffId}})
	}
	c.Validate()
	return nil
}

func (c *Character) TrackBuffStarted(buffId int) {
	c.Buffs.Started(buffId)
}

func (c *Character) GetBuffs(buffId ...int) []*buffs.Buff {
	return c.Buffs.GetBuffs(buffId...)
}

func (c *Character) RemoveBuff(buffId int) {
	buffId = int(math.Abs(float64(buffId)))
	c.Buffs.RemoveBuff(buffId)
	// Chunk 6 (Perception): clearing a blind-source buff may flip
	// Blinded → Sighted, but only if no other blind source remains.
	if (buffId == perception.BuffIdBlinded || buffId == perception.BuffIdFlashbangBlindness) &&
		c.Perception != nil && c.Perception.State() == perception.Blinded && !c.HasAnyBlindSource() {
		_ = c.Perception.TransitionTo(perception.Sighted,
			state.TransitionReason{Trigger: perception.TriggerBuffExpired, Metadata: map[string]any{"buffId": buffId}})
	}
	c.Validate()
}

// Used with SpawnInfo to gift spawning mobs with permabuffs
func (c *Character) SetPermaBuffs(buffIds []int) {
	c.permaBuffIds = buffIds
}

// RemovePermaBuff removes a buff ID from the permanent buff list so
// it won't be re-applied during Validate(). Use this when a permabuff
// should be permanently lost (e.g., revealing a hidden mob).
func (c *Character) RemovePermaBuff(buffId int) {
	for i, id := range c.permaBuffIds {
		if id == buffId {
			c.permaBuffIds = append(c.permaBuffIds[:i], c.permaBuffIds[i+1:]...)
			return
		}
	}
}

func (c *Character) reapplyPermabuffs(removedItems ...items.Item) {

	buffIdCount := map[int]int{}

	for _, buffId := range c.permaBuffIds {
		buffIdCount[buffId] = 100 // Special case permabuffs associated with certain mobs
	}

	// Apply any buffs that come from a species
	if rInfo := species.GetSpecies(c.SpeciesId); rInfo != nil {
		for _, buffId := range rInfo.BuffIds {
			buffIdCount[buffId] = 100 // Don't allow species buffs to be removed, keep this number high
		}
	}

	// Apply any buffs from pet
	if c.Pet.Exists() {
		for _, buffId := range c.Pet.GetBuffs() {
			buffIdCount[buffId] = 100 // Don't allow pet buffs to be removed, keep this number high
		}
	}

	// Track any buffs that come from an item
	// If these don't show up as still being required by an item (such as a yaml file was changed)
	// This will cause them to be removed.
	for _, b := range c.Buffs.List {
		if b.PermaBuff {
			if _, ok := buffIdCount[b.BuffId]; !ok {
				buffIdCount[b.BuffId] = 0
			}
		}
	}

	// Make a list of all item buffs provided by existing worn items
	for _, itm := range c.GetAllWornItems() {
		spec := itm.GetSpec()
		for _, buffId := range spec.WornBuffIds {
			buffIdCount[buffId] = buffIdCount[buffId] + 1
		}

	}
	// Remove any buffs that come specifically from item
	for _, removedItem := range removedItems {
		iSpec := removedItem.GetSpec()
		if len(iSpec.WornBuffIds) > 0 {
			for _, buffId := range iSpec.WornBuffIds {
				buffIdCount[buffId] = buffIdCount[buffId] - 1
			}
		}
	}

	for buffId, ct := range buffIdCount {
		if ct < 1 {
			c.RemoveBuff(buffId)
		} else {
			c.AddBuff(buffId, true)
		}
	}
}
