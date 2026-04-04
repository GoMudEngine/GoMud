package shops

// StockEntry represents one item type in a shop's inventory.
type StockEntry struct {
	ItemId     int `yaml:"item_id"`
	RestockQty int `yaml:"restock_qty"` // How many the supply cart brings (0 = NPC-crafted only)
	MaxStock   int `yaml:"max_stock"`   // Hard cap on accumulation
	Current    int `yaml:"current"`     // Actual current count (persisted)
}

// ShopInventory is the persistent economic state for one shop NPC.
type ShopInventory struct {
	Gold         int          `yaml:"gold"`
	StartingGold int          `yaml:"starting_gold"` // Seed value; used for gold reserve calc
	LastRestock  uint64       `yaml:"last_restock"`
	Stock        []StockEntry `yaml:"inventory"`
	KnownRecipes []string     `yaml:"known_recipes,omitempty"` // Recipes the NPC knows

	// Location fields (not persisted — set at registration time for save path)
	Zone   string `yaml:"-"`
	MobId  int    `yaml:"-"`
	RoomId int    `yaml:"-"`
}

// GetStock returns the StockEntry for an item, or nil if not stocked.
func (si *ShopInventory) GetStock(itemId int) *StockEntry {
	for i := range si.Stock {
		if si.Stock[i].ItemId == itemId {
			return &si.Stock[i]
		}
	}
	return nil
}

// AddStock increases current stock for an item, capped at MaxStock.
// If the item isn't in the stock list, it's added as a temporary entry
// (RestockQty=0, MaxStock=20).
func (si *ShopInventory) AddStock(itemId int, qty int) {
	entry := si.GetStock(itemId)
	if entry == nil {
		si.Stock = append(si.Stock, StockEntry{
			ItemId:     itemId,
			RestockQty: 0,
			MaxStock:   20,
			Current:    0,
		})
		entry = &si.Stock[len(si.Stock)-1]
	}
	entry.Current += qty
	if entry.Current > entry.MaxStock {
		entry.Current = entry.MaxStock
	}
}

// RemoveStock decreases current stock by qty. Returns actual amount removed.
func (si *ShopInventory) RemoveStock(itemId int, qty int) int {
	entry := si.GetStock(itemId)
	if entry == nil || entry.Current <= 0 {
		return 0
	}
	removed := qty
	if removed > entry.Current {
		removed = entry.Current
	}
	entry.Current -= removed
	return removed
}

// Restock applies the supply cart delivery for all items with RestockQty > 0.
// Returns true if any stock was added.
func (si *ShopInventory) Restock() bool {
	restocked := false
	for i := range si.Stock {
		e := &si.Stock[i]
		if e.RestockQty <= 0 {
			continue
		}
		room := e.MaxStock - e.Current
		if room <= 0 {
			continue
		}
		add := e.RestockQty
		if add > room {
			add = room
		}
		e.Current += add
		restocked = true
	}
	return restocked
}

// GoldReserve returns the minimum gold the NPC should hold back
// from discretionary purchases (gear upgrades).
func (si *ShopInventory) GoldReserve(ratio float64) int {
	return int(float64(si.StartingGold) * ratio)
}

// CanAfford returns true if spending amount would not drop below
// the given reserve floor.
func (si *ShopInventory) CanAfford(amount int, reserveFloor int) bool {
	return si.Gold-amount >= reserveFloor
}
