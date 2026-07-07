package rooms

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/gametime"
)

type Corpse struct {
	UserId       int
	MobId        int
	Character    characters.Character
	RoundCreated uint64
	Prunable     bool // Whether it can be removed
	WasCharmed   bool // True if the mob was a charmed companion when it died

	// Stage 3.4: optional overrides for special-mob corpses (wagons,
	// statues, etc.). Stamped from the dying mob's YAML overrides at
	// corpse creation time.
	CorpseName        string
	CorpseDescription string

	// Corpse-loot redesign (2026-07-07): mob loot lives here, not on the floor.
	Loot            Container // items + gold looted from the dead mob
	OwnerUserIds    []int     // who may loot before RoundOwnedUntil (empty = anyone)
	LootMode        string    // "ffa" | "roundrobin" | "leaderhold" ("" = solo/ffa)
	RoundOwnedUntil uint64    // round at which ownership opens to free-for-all

	// RRAssignee gates individual loot items by loot mode. Keyed by the item's
	// stable per-instance UID (items.Item.UUID.String()) -> the userId that item
	// is reserved for. Empty/nil for ffa (any owner may take any item). For
	// round-robin the items are dealt across members; for leader-hold every item
	// maps to the leader. Layered on top of ownership (LootAllowed) via
	// CanTakeItem.
	RRAssignee map[string]int
}

// CanTakeItem reports whether userId may take this item now, layering loot-mode
// assignment on top of ownership. Ownership must already be checked by the
// caller (LootAllowed); this only adds the mode gate.
func (c *Corpse) CanTakeItem(itemUID string, userId int, now uint64) bool {
	if now >= c.RoundOwnedUntil || len(c.RRAssignee) == 0 {
		return true
	}
	owner, ok := c.RRAssignee[itemUID]
	return !ok || owner == userId
}

// LootAllowed reports whether userId may loot this corpse at round `now`.
// Free-for-all once now >= RoundOwnedUntil, or when there is no owner set
// (mob/environment kill). Otherwise only listed owners may loot.
func (c *Corpse) LootAllowed(userId int, now uint64) bool {
	if now >= c.RoundOwnedUntil || len(c.OwnerUserIds) == 0 {
		return true
	}
	for _, id := range c.OwnerUserIds {
		if id == userId {
			return true
		}
	}
	return false
}

// HasLoot reports whether the corpse still holds any items or gold.
func (c *Corpse) HasLoot() bool {
	return len(c.Loot.Items) > 0 || c.Loot.Gold > 0
}

// DisplayName returns the rendered corpse name. If CorpseName is set
// (Stage 3.4 special mobs), returns it directly. Otherwise returns
// the standard "<Name> corpse" form.
func (c Corpse) DisplayName() string {
	if c.CorpseName != "" {
		return c.CorpseName
	}
	return c.Character.Name + " corpse"
}

func (c *Corpse) Update(roundNow uint64, decayRate string) {

	if c.Prunable {
		return
	}

	if decayRate == `` {
		decayRate = `1 week`
	}

	gd := gametime.GetDate(c.RoundCreated)
	decayRound := gd.AddPeriod(decayRate)

	// Has enough time passed to do the respawn?
	if roundNow >= decayRound {
		c.Prunable = true
	}

}
