package characters

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// CarryCapacity returns weight capacity in pounds. When
// carryCapacityOverride is set (Stage 3.4 special mobs), returns it
// directly. Otherwise computes Strength × config multiplier.
func (c *Character) CarryCapacity() float64 {
	if c.carryCapacityOverride > 0 {
		return c.carryCapacityOverride
	}
	bal := configs.GetBalanceConfig()
	return float64(c.Stats.Strength.ValueAdj) * float64(bal.CarryCapacityMultiplier)
}

// GetCarriedWeight returns the total weight of all carried items in pounds
func (c *Character) GetCarriedWeight() float64 {
	// Backpack item weights
	backpackWeight := 0.0
	for _, item := range c.Items {
		backpackWeight += item.GetSpec().GetWeight()
	}
	// Apply Back slot weight reduction to backpack contents
	if c.Equipment.Back.ItemId > 0 {
		reduction := c.Equipment.Back.GetSpec().WeightReduction
		if reduction > 0 && reduction <= 1.0 {
			backpackWeight *= (1.0 - reduction)
		}
	}

	// Component bag item weights
	componentWeight := 0.0
	for _, item := range c.ComponentItems {
		componentWeight += item.GetSpec().GetWeight()
	}
	// Apply ComponentBag weight reduction
	if c.Equipment.ComponentBag.ItemId > 0 {
		reduction := c.Equipment.ComponentBag.GetSpec().WeightReduction
		if reduction > 0 && reduction <= 1.0 {
			componentWeight *= (1.0 - reduction)
		}
	}

	// Potion bandolier item weights
	potionWeight := 0.0
	for _, item := range c.PotionItems {
		potionWeight += item.GetSpec().GetWeight()
	}
	// Apply bandolier weight reduction
	if c.Equipment.Belt.ItemId > 0 {
		beltSpec := c.Equipment.Belt.GetSpec()
		if beltSpec.IsBandolier && beltSpec.WeightReduction > 0 && beltSpec.WeightReduction <= 1.0 {
			potionWeight *= (1.0 - beltSpec.WeightReduction)
		}
	}

	// Equipped item weights (all slots via GetAllItems)
	// Equipped items count at 50% weight — wearing something distributes
	// its load across your body rather than stuffing it in a bag.
	equippedWeight := 0.0
	for _, item := range c.Equipment.GetAllItems() {
		equippedWeight += item.GetSpec().GetWeight() * 0.5
	}

	return backpackWeight + componentWeight + potionWeight + equippedWeight
}

func (c *Character) FindKeyInBackpack(lockId string) (items.Item, bool) {

	lockId = strings.ToLower(lockId)

	for _, itm := range c.GetAllBackpackItems() {
		itmSpec := itm.GetSpec()
		if itmSpec.Type != items.Key {
			continue
		}

		if itmSpec.KeyLockId == lockId {
			return itm, true
		}
	}

	return items.Item{}, false
}

func (c *Character) HasKey(lockId string, difficulty int) (hasKey bool, hasSequence bool) {

	sequence := util.GetLockSequence(lockId, difficulty, string(configs.GetServerConfig().Seed))

	// Check whether they ahve a key for this lock
	return c.GetKey(`key-`+lockId) != ``, c.GetKey(lockId) == sequence
}

func (c *Character) KeyCount() int {
	if c.KeyRing == nil {
		c.KeyRing = make(map[string]string)
	}
	return len(c.KeyRing)
}

func (c *Character) GetKey(lockId string) string {
	if c.KeyRing == nil {
		c.KeyRing = make(map[string]string)
	}
	return c.KeyRing[strings.ToLower(lockId)]
}

func (c *Character) SetKey(lockId string, sequence string) {
	if c.KeyRing == nil {
		c.KeyRing = make(map[string]string)
	}
	if len(sequence) == 0 {
		delete(c.KeyRing, strings.ToLower(lockId))
	} else {
		c.KeyRing[strings.ToLower(lockId)] = strings.ToUpper(sequence)
	}
}

func (c *Character) GetRandomItem() (items.Item, bool) {
	if len(c.Items) == 0 {
		return items.Item{}, false
	}
	return c.Items[util.Rand(len(c.Items))], true
}

func (c *Character) StoreItem(i items.Item) bool {
	if i.ItemId < 1 {
		return false
	}

	i.Validate()

	// Check if adding this item would exceed carry capacity
	newWeight := c.GetCarriedWeight() + i.GetSpec().GetWeight()
	capacity := c.CarryCapacity()

	// Allow up to 2x capacity (overloaded, but possible)
	if newWeight > capacity*2.0 {
		return false
	}

	// Auto-route component items to the component bag
	iSpec := i.GetSpec()
	if iSpec.IsComponent && c.Equipment.ComponentBag.ItemId > 0 {
		bagSpec := c.Equipment.ComponentBag.GetSpec()
		if bagSpec.BagCapacity > 0 && len(c.ComponentItems) < bagSpec.BagCapacity {
			c.ComponentItems = append(c.ComponentItems, i)
			return true
		}
	}

	// Auto-route potions and throwables to the bandolier
	if (iSpec.Type == items.Potion || (iSpec.Subtype == items.Drinkable && len(iSpec.BuffIds) > 0) || iSpec.Subtype == items.Throwable) && c.Equipment.Belt.ItemId > 0 {
		beltSpec := c.Equipment.Belt.GetSpec()
		if beltSpec.IsBandolier && beltSpec.BandolierCapacity > 0 && len(c.PotionItems) < beltSpec.BandolierCapacity {
			c.PotionItems = append(c.PotionItems, i)
			return true
		}
	}

	c.Items = append(c.Items, i)

	return true
}

func (c *Character) RemoveItem(i items.Item) bool {
	for j := len(c.Items) - 1; j >= 0; j-- {
		if c.Items[j].Equals(i) {
			c.Items = append(c.Items[:j], c.Items[j+1:]...)
			return true
		}
	}
	for j := len(c.ComponentItems) - 1; j >= 0; j-- {
		if c.ComponentItems[j].Equals(i) {
			c.ComponentItems = append(c.ComponentItems[:j], c.ComponentItems[j+1:]...)
			return true
		}
	}
	for j := len(c.PotionItems) - 1; j >= 0; j-- {
		if c.PotionItems[j].Equals(i) {
			c.PotionItems = append(c.PotionItems[:j], c.PotionItems[j+1:]...)
			return true
		}
	}
	return false
}

// Copies over an existing item with a new item
// Returns true if successfully replaces an item
func (c *Character) UpdateItem(originalItm items.Item, replacement items.Item) bool {
	for j := len(c.Items) - 1; j >= 0; j-- {
		if c.Items[j].Equals(originalItm) {
			// If the number of uses remaining has decremented from the original item
			// The item gets destroyed from existence
			if originalItm.Uses >= 1 && replacement.Uses < 1 {
				c.Items = append(c.Items[:j], c.Items[j+1:]...)
			} else {
				c.Items[j] = replacement
			}
			return true
		}
	}
	return false
}

func (c *Character) UseItem(i items.Item) int {
	for j := len(c.Items) - 1; j >= 0; j-- {
		if c.Items[j].Equals(i) {
			usesLeft := c.Items[j].Uses
			if usesLeft > 0 {
				usesLeft--
			}
			if usesLeft <= 0 {
				c.Items = append(c.Items[:j], c.Items[j+1:]...)
			} else {
				c.Items[j].Uses = usesLeft
				c.Items[j].LastUsedRound = util.GetRoundCount()
			}

			return usesLeft
		}
	}

	return 0
}

// FindInPotions searches the bandolier for a matching potion, oldest first.
func (c *Character) FindInPotions(itemName string) (items.Item, bool) {
	if itemName == `` || len(c.PotionItems) == 0 {
		return items.Item{}, false
	}

	// Sort by CraftedRound ascending (oldest first) for consumption priority
	oldestIdx := -1
	var oldestRound uint64 = ^uint64(0)

	for idx := range c.PotionItems {
		closeMatch, fullMatch := items.FindMatchIn(itemName, c.PotionItems[idx])
		matched := fullMatch.ItemId != 0 || closeMatch.ItemId != 0
		if matched && c.PotionItems[idx].CraftedRound < oldestRound {
			oldestIdx = idx
			oldestRound = c.PotionItems[idx].CraftedRound
		}
	}

	if oldestIdx >= 0 {
		return c.PotionItems[oldestIdx], true
	}

	return items.Item{}, false
}

// UseItemFromPotions consumes a potion from the bandolier.
func (c *Character) UseItemFromPotions(i items.Item) int {
	for j := len(c.PotionItems) - 1; j >= 0; j-- {
		if c.PotionItems[j].Equals(i) {
			usesLeft := c.PotionItems[j].Uses
			if usesLeft > 0 {
				usesLeft--
			}
			if usesLeft <= 0 {
				c.PotionItems = append(c.PotionItems[:j], c.PotionItems[j+1:]...)
			} else {
				c.PotionItems[j].Uses = usesLeft
				c.PotionItems[j].LastUsedRound = util.GetRoundCount()
			}
			return usesLeft
		}
	}
	return 0
}

func (c *Character) FindInComponents(itemName string) (items.Item, bool) {
	if itemName == `` || len(c.ComponentItems) == 0 {
		return items.Item{}, false
	}

	closeMatchItem, matchItem := items.FindMatchIn(itemName, c.ComponentItems...)

	if matchItem.ItemId != 0 {
		return matchItem, true
	}

	if closeMatchItem.ItemId != 0 {
		return closeMatchItem, true
	}

	return items.Item{}, false
}

func (c *Character) FindInBackpack(itemName string) (items.Item, bool) {

	if itemName == `` {
		return items.Item{}, false
	}

	closeMatchItem, matchItem := items.FindMatchIn(itemName, c.Items...)

	if matchItem.ItemId != 0 {
		return matchItem, true
	}

	if closeMatchItem.ItemId != 0 {
		return closeMatchItem, true
	}

	return items.Item{}, false
}

func (c *Character) FindOnBody(itemName string) (items.Item, bool) {

	if itemName == `` {
		return items.Item{}, false
	}

	partialMatch, fullMatch := items.FindMatchIn(itemName,
		c.Equipment.Weapon,
		c.Equipment.Offhand,
		c.Equipment.ExtraArm1,
		c.Equipment.ExtraArm2,
		c.Equipment.ExtraArm3,
		c.Equipment.ExtraArm4,
		c.Equipment.Head,
		c.Equipment.Neck,
		c.Equipment.Shoulders,
		c.Equipment.Body,
		c.Equipment.Back,
		c.Equipment.Belt,
		c.Equipment.Wrist1,
		c.Equipment.Wrist2,
		c.Equipment.ExtraWrist1,
		c.Equipment.ExtraWrist2,
		c.Equipment.ExtraWrist3,
		c.Equipment.ExtraWrist4,
		c.Equipment.Gloves,
		c.Equipment.Ring,
		c.Equipment.Ring2,
		c.Equipment.Legs,
		c.Equipment.Feet,
		c.Equipment.Tail,
		c.Equipment.ComponentBag)

	if fullMatch.ItemId != 0 {
		return fullMatch, true
	}

	if partialMatch.ItemId != 0 {
		return partialMatch, true
	}

	return items.Item{}, false
}

// FindItem searches backpack and equipped items as a single pool for
// disambiguation. Returns the item, a source description, and whether found.
func (c *Character) FindItem(itemName string) (items.Item, string, bool) {
	if itemName == "" {
		return items.Item{}, "", false
	}

	// Build combined pool of all items with source labels
	type candidate struct {
		item   items.Item
		source string
	}
	var pool []candidate

	for _, item := range c.Items {
		if item.ItemId > 0 {
			pool = append(pool, candidate{item, "in your backpack"})
		}
	}

	for _, item := range c.PotionItems {
		if item.ItemId > 0 {
			pool = append(pool, candidate{item, "in your bandolier"})
		}
	}

	slotItems := []struct {
		item   items.Item
		source string
	}{
		{c.Equipment.Weapon, "wielded"},
		{c.Equipment.Offhand, "offhand"},
		{c.Equipment.ExtraArm1, "extra arm"},
		{c.Equipment.ExtraArm2, "extra arm"},
		{c.Equipment.ExtraArm3, "extra arm 3"},
		{c.Equipment.ExtraArm4, "extra arm 4"},
		{c.Equipment.Head, "worn - head"},
		{c.Equipment.Neck, "worn - neck"},
		{c.Equipment.Shoulders, "worn - shoulders"},
		{c.Equipment.Body, "worn - body"},
		{c.Equipment.Back, "worn - back"},
		{c.Equipment.Belt, "worn - belt"},
		{c.Equipment.Wrist1, "worn - wrist"},
		{c.Equipment.Wrist2, "worn - wrist"},
		{c.Equipment.ExtraWrist1, "extra wrist 1"},
		{c.Equipment.ExtraWrist2, "extra wrist 2"},
		{c.Equipment.ExtraWrist3, "extra wrist 3"},
		{c.Equipment.ExtraWrist4, "extra wrist 4"},
		{c.Equipment.Gloves, "worn - gloves"},
		{c.Equipment.Ring, "worn - ring"},
		{c.Equipment.Ring2, "worn - ring"},
		{c.Equipment.Legs, "worn - legs"},
		{c.Equipment.Feet, "worn - feet"},
		{c.Equipment.Tail, "worn - tail"},
		{c.Equipment.ComponentBag, "worn - componentbag"},
	}
	for _, slot := range slotItems {
		if slot.item.ItemId > 0 {
			pool = append(pool, candidate{slot.item, slot.source})
		}
	}

	// Extract items for FindMatchIn
	poolItems := make([]items.Item, len(pool))
	for i, c := range pool {
		poolItems[i] = c.item
	}

	closeMatch, fullMatch := items.FindMatchIn(itemName, poolItems...)

	// Find source for the match
	findSource := func(match items.Item) string {
		for _, c := range pool {
			if c.item.ItemId == match.ItemId && c.item.UUID == match.UUID {
				return c.source
			}
		}
		return "in your backpack"
	}

	if fullMatch.ItemId != 0 {
		return fullMatch, findSource(fullMatch), true
	}
	if closeMatch.ItemId != 0 {
		return closeMatch, findSource(closeMatch), true
	}

	return items.Item{}, "", false
}

func (c *Character) GetAllBackpackItems() []items.Item {
	return append([]items.Item{}, c.Items...)
}

// GetAllCarriedItems returns items from all pools (backpack, component bag, bandolier).
// Used by scripting APIs that need to find items regardless of which pool they're in.
func (c *Character) GetAllCarriedItems() []items.Item {
	all := append([]items.Item{}, c.Items...)
	all = append(all, c.ComponentItems...)
	all = append(all, c.PotionItems...)
	return all
}

// SortComponentItems moves is_component items from backpack into ComponentItems
// up to the equipped bag's capacity. Returns the count of items moved.
func (c *Character) SortComponentItems() int {
	if c.Equipment.ComponentBag.ItemId < 1 {
		return 0
	}
	bagSpec := c.Equipment.ComponentBag.GetSpec()
	capacity := bagSpec.BagCapacity
	if capacity <= 0 {
		return 0
	}

	moved := 0
	remaining := make([]items.Item, 0, len(c.Items))
	for _, item := range c.Items {
		if item.GetSpec().IsComponent && len(c.ComponentItems) < capacity {
			c.ComponentItems = append(c.ComponentItems, item)
			moved++
		} else {
			remaining = append(remaining, item)
		}
	}
	c.Items = remaining
	return moved
}

// SortPotionItems moves drinkable items from backpack into PotionItems
// up to the equipped bandolier's capacity. Returns the count of items moved.
func (c *Character) SortPotionItems() int {
	if c.Equipment.Belt.ItemId < 1 {
		return 0
	}
	beltSpec := c.Equipment.Belt.GetSpec()
	if !beltSpec.IsBandolier {
		return 0
	}
	capacity := beltSpec.BandolierCapacity
	if capacity <= 0 {
		return 0
	}

	moved := 0
	remaining := make([]items.Item, 0, len(c.Items))
	for _, item := range c.Items {
		sub := item.GetSpec().Subtype
		if (sub == items.Drinkable || sub == items.Throwable) && len(c.PotionItems) < capacity {
			c.PotionItems = append(c.PotionItems, item)
			moved++
		} else {
			remaining = append(remaining, item)
		}
	}
	c.Items = remaining
	return moved
}
