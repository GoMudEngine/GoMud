package characters

import (
	"github.com/GoMudEngine/GoMud/internal/enchantments"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// MigrateEnchantments re-applies all enchantment effects using the current
// enchantment definitions. This handles:
//   - Reserve pool changes (e.g. health → stamina)
//   - Rebalanced tier effects
//   - New tier counts (clamping if old tier exceeds new max)
//   - Stripping enchantments whose definitions no longer exist
//
// Called once per character load via LoadUser().
func (c *Character) MigrateEnchantments() {
	updated := 0

	// Migrate backpack items
	for i := range c.Items {
		if migrateEnchantedItem(&c.Items[i]) {
			updated++
		}
	}

	// Migrate component bag items
	for i := range c.ComponentItems {
		if migrateEnchantedItem(&c.ComponentItems[i]) {
			updated++
		}
	}

	// Migrate potion bandolier items
	for i := range c.PotionItems {
		if migrateEnchantedItem(&c.PotionItems[i]) {
			updated++
		}
	}

	// Migrate all equipped items
	for _, ptr := range c.Equipment.GetAllItemPtrs() {
		if migrateEnchantedItem(ptr) {
			updated++
		}
	}

	if updated > 0 {
		mudlog.Info("MigrateEnchantments", "character", c.Name, "items_updated", updated)
	}
}

// migrateEnchantedItem re-applies the enchantment definition to a single item.
// Returns true if the item was modified.
func migrateEnchantedItem(item *items.Item) bool {
	if item.EnchantType == "" {
		return false
	}

	def := enchantments.GetEnchantment(item.EnchantType)
	if def == nil {
		// Enchantment definition no longer exists — strip it
		enchantments.StripEnchantment(item)
		return true
	}

	// Clamp tier to new definition's max
	maxTier := len(def.Tiers) - 1
	if item.EnchantTier > maxTier {
		item.EnchantTier = maxTier
	}

	// Update reserve pool in case it changed
	item.ReservePool = def.ReservePool

	// Re-apply effects from the current definition
	enchantments.ApplyTier(item, def, item.EnchantTier)

	return true
}
