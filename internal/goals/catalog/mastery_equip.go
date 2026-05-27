// Package-level notes:
//
// # Shop-zone scan deferred to chunk 4.4
//
// shopForSlotProximity always returns 2 ("no shop nearby") in this
// implementation. The real shop-in-zone scan that will return 0 (in
// current room) or 1 (in zone) ships in chunk 4.4's planner. Until
// then, ContextScore always falls through to the "no shop nearby"
// branch and returns either 1.5 (slot empty) or 0.3 (slot occupied
// but below tier target).
//
// # Rarity-tier tuning is patchy until project_rarity_tier_audit lands
//
// 145/213 weapons and armor items lack a rarity_tier tag per
// project_rarity_tier_audit. Untagged items get the engine fallback
// tier (50, per 474afd98). Goals that use min_rarity_tier ≤ 50 will
// therefore treat most untagged items as satisfying the predicate —
// which is the intended "good-enough default" behavior until the audit
// tags mid/high-tier content explicitly. Authors in untagged-heavy
// zones should set min_rarity_tier ≤ 50 to get predictable behavior.

package catalog

import (
	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// rarityFallbackTier is the tier-50 engine fallback applied to any item
// whose ItemSpec.RarityTier == 0 (i.e. untagged). Matches the fallback
// introduced in commit 474afd98.
const rarityFallbackTier = 50

func init() {
	goals.RegisterGoalType("mastery-equip", goals.GoalTypeMeta{
		Predicate:     masteryEquipPredicate,
		ContextScore:  masteryEquipContextScore,
		AllowMultiple: true,
		DedupKey:      masteryEquipDedupKey,
		Params: []goals.ParamSchema{
			{Key: "slot", Required: true, GoType: "string"},
			{Key: "min_rarity_tier", Required: true, GoType: "int"},
		},
	})
}

func masteryEquipDedupKey(g *goals.Goal) string {
	if s, ok := g.Params["slot"].(string); ok {
		return s
	}
	return ""
}

// masteryEquipPredicate: satisfied when equipped item in slot has
// rarity_tier >= target. Untagged items (RarityTier == 0) use the
// engine fallback (rarityFallbackTier = 50). Empty slots return tier 0
// and never satisfy the predicate.
func masteryEquipPredicate(g *goals.Goal, mob *mobs.Mob) bool {
	if mob == nil {
		return false
	}
	slot, _ := g.Params["slot"].(string)
	target := paramIntOr(g, "min_rarity_tier", 0)
	if slot == "" || target == 0 {
		return false
	}
	tier := mobSlotRarityTier(mob, slot)
	return tier >= target
}

// masteryEquipContextScore tiers:
//   - 0 if predicate already satisfied
//   - 1.5 if slot is empty (very motivated to acquire anything)
//   - 1.5 if a shop selling slot items is in the current room (4.4+)
//   - 1.0 if a shop is elsewhere in the zone (4.4+)
//   - 0.3 if no shop nearby (4.3 fallback — always this branch for now)
//
// The shop-proximity tiers (0, 1) are dead code until chunk 4.4 wires
// shopForSlotProximity to a real zone-shop scan. See file-level note.
func masteryEquipContextScore(g *goals.Goal, mob *mobs.Mob) float64 {
	if mob == nil {
		return 0
	}
	slot, _ := g.Params["slot"].(string)
	target := paramIntOr(g, "min_rarity_tier", 0)
	if slot == "" || target == 0 {
		return 0
	}
	tier := mobSlotRarityTier(mob, slot)
	if tier >= target {
		return 0 // predicate satisfied — no urgency
	}
	if mobSlotIsEmpty(mob, slot) {
		return 1.5 // nothing equipped — highly motivated
	}
	switch shopForSlotProximity(mob, slot) {
	case 0: // shop in current room (4.4+)
		return 1.5
	case 1: // shop elsewhere in zone (4.4+)
		return 1.0
	}
	return 0.3 // no shop nearby — 4.3 always takes this branch
}

// ─── helpers ────────────────────────────────────────────────────────────────

// mobSlotItem returns the item currently equipped in the named slot on mob,
// or items.Item{} (zero value, ItemId == 0) for unknown or empty slots.
//
// Slot strings are the author-facing names used in goal params (e.g.
// "weapon", "head"). Internally we delegate to Worn.GetSlotPointer which
// uses the engine's longer label strings. Unknown slot strings produce an
// empty item rather than a panic.
func mobSlotItem(mob *mobs.Mob, slot string) items.Item {
	ptr := mob.Character.Equipment.GetSlotPointer(slotLabel(slot))
	if ptr == nil {
		return items.Item{}
	}
	return *ptr
}

// slotLabel converts the short author-facing slot name used in goal params
// to the internal label string expected by Worn.GetSlotPointer. Unrecognised
// slots map to "" and GetSlotPointer returns nil for those.
func slotLabel(slot string) string {
	switch slot {
	case "weapon":
		return "wielded"
	case "offhand":
		return "offhand"
	case "head":
		return "worn - head"
	case "neck":
		return "worn - neck"
	case "shoulders":
		return "worn - shoulders"
	case "body":
		return "worn - body"
	case "back":
		return "worn - back"
	case "belt":
		return "worn - belt"
	case "wrist", "wrist1":
		return "worn - wrist"
	case "wrist2":
		return "worn - wrist2"
	case "gloves":
		return "worn - gloves"
	case "ring":
		return "worn - ring"
	case "ring2":
		return "worn - ring2"
	case "legs":
		return "worn - legs"
	case "feet":
		return "worn - feet"
	case "tail":
		return "worn - tail"
	case "componentbag":
		return "worn - componentbag"
	}
	return "" // unknown → GetSlotPointer returns nil
}

// mobSlotIsEmpty reports whether the named slot is unoccupied
// (ItemId == 0 means no item equipped).
func mobSlotIsEmpty(mob *mobs.Mob, slot string) bool {
	return mobSlotItem(mob, slot).ItemId == 0
}

// mobSlotRarityTier returns the rarity tier of the item equipped in slot,
// applying the engine fallback (rarityFallbackTier = 50) for real items
// whose ItemSpec.RarityTier is 0 (untagged). Returns 0 only for empty
// slots (ItemId == 0), so empty slots never satisfy a min_rarity_tier
// predicate.
func mobSlotRarityTier(mob *mobs.Mob, slot string) int {
	item := mobSlotItem(mob, slot)
	if item.ItemId == 0 {
		return 0 // empty slot
	}
	tier := item.GetSpec().RarityTier
	if tier == 0 {
		return rarityFallbackTier // untagged item → tier-50 fallback
	}
	return tier
}

// shopForSlotProximity is a 4.3 heuristic stub that always returns 2
// ("no shop nearby"). The real shop-in-zone scan ships in chunk 4.4's
// planner. See file-level note above.
func shopForSlotProximity(_ *mobs.Mob, _ string) int {
	return 2
}
