package shops

import (
	"slices"

	"github.com/GoMudEngine/GoMud/internal/economy"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// CraftSupport tags a shop with the crafting discipline its stock
// supports. Single-discipline shops use the matching skill tag;
// multi-discipline / mixed merchants use "general". Empty is INVALID
// — a startup validator panics if any shop-bearing mob is missing
// the tag (see ValidateShopMobTags).
const (
	CraftSupportBlacksmithing = "blacksmithing"
	CraftSupportAlchemy       = "alchemy"
	CraftSupportTailoring     = "tailoring"
	CraftSupportCooking       = "cooking"
	CraftSupportJewelcrafting = "jewelcrafting"
	CraftSupportEnchanting    = "enchanting"
	CraftSupportGeneral       = "general"
)

// ValidCraftSupports is the canonical set. Mirrors the player crafting
// skills in internal/skills/skills.go plus "general" for mixed shops.
var ValidCraftSupports = []string{
	CraftSupportBlacksmithing,
	CraftSupportAlchemy,
	CraftSupportTailoring,
	CraftSupportCooking,
	CraftSupportJewelcrafting,
	CraftSupportEnchanting,
	CraftSupportGeneral,
}

// IsValidCraftSupport reports whether v is one of ValidCraftSupports.
func IsValidCraftSupport(v string) bool {
	return slices.Contains(ValidCraftSupports, v)
}

// ValidVendorCategories is the canonical set of values that may appear in
// ItemSpec.VendorCategories. Mirrors ValidCraftSupports MINUS "general"
// — items belong to discipline(s); general stores accept everything.
var ValidVendorCategories = []string{
	CraftSupportBlacksmithing,
	CraftSupportAlchemy,
	CraftSupportTailoring,
	CraftSupportCooking,
	CraftSupportJewelcrafting,
	CraftSupportEnchanting,
}

// IsValidVendorCategory reports whether v is one of ValidVendorCategories.
func IsValidVendorCategory(v string) bool {
	return slices.Contains(ValidVendorCategories, v)
}

// StockEvent records one depletion→refill cycle for a single item.
// DepletedRound is the round Current dropped to 0; RefilledRound is
// the round Current returned > 0. RefilledRound = 0 means the
// event is still ongoing (item currently depleted).
type StockEvent struct {
	DepletedRound uint64 `yaml:"depleted_round"`
	RefilledRound uint64 `yaml:"refilled_round"`
}

// StockEntry represents one item type in a shop's inventory.
type StockEntry struct {
	ItemId        int    `yaml:"item_id"`
	RestockQty    int    `yaml:"restock_qty"`               // How many the supply cart brings (0 = NPC-crafted only)
	MaxStock      int    `yaml:"max_stock"`                 // Hard cap on accumulation
	Current       int    `yaml:"current"`                   // Actual current count (persisted)
	LastGrewRound uint64 `yaml:"last_grew_round,omitempty"` // Round Current last increased; drives overstock decay grace period.
}

// ShopInventory is the persistent economic state for one shop NPC.
type ShopInventory struct {
	Gold         int          `yaml:"gold"`
	StartingGold int          `yaml:"starting_gold"` // Seed value; used for gold reserve calc
	LastRestock  uint64       `yaml:"last_restock"`
	Stock        []StockEntry `yaml:"inventory"`
	KnownRecipes []string     `yaml:"known_recipes,omitempty"` // Recipes the NPC knows
	CraftSupport string       `yaml:"craft_support,omitempty"` // Discipline this shop's stock supports — see ValidCraftSupports

	// LastRestockByTier records the round of the most recent restock
	// per rarity tier. Replaces the single LastRestock for the
	// per-tier cadence model. Persisted; zero value for a tier means
	// "never restocked at this tier yet" — callers initialize to
	// currentRound on first encounter.
	LastRestockByTier map[int]uint64 `yaml:"last_restock_by_tier,omitempty"`

	// SalesCount is the cumulative number of items the shop has sold
	// to players. Drives the throughput score's "items moved" axis.
	SalesCount int `yaml:"sales_count,omitempty"`

	// BuysCount is the cumulative number of items the shop has bought
	// FROM players.
	BuysCount int `yaml:"buys_count,omitempty"`

	// RestockCount is the cumulative number of items added to stock
	// via fired restock cycles (not foragers/caravans/player sales).
	// Drives input rate scoring.
	RestockCount int `yaml:"restock_count,omitempty"`

	// ConsumedByCrafterCount is the cumulative number of items the
	// shop's crafter NPC has consumed as ingredients to produce other
	// items. Distinct from BuysCount (player-driven inflow) and from
	// "items destroyed in failed crafts" — this counts only material
	// consumption by the crafter mob's own work (success or failure).
	ConsumedByCrafterCount int `yaml:"consumed_by_crafter_count,omitempty"`

	// StockEvents holds completed depletion→refill events per item,
	// rolling 7-game-day window. Drives Time-to-Refill (TtR) scoring.
	StockEvents map[int][]StockEvent `yaml:"stock_events,omitempty"`

	// CurrentDepletion records the round each item went to 0 and is
	// still depleted. Cleared when the item refills (event pushed to
	// StockEvents). Items not present here are not currently depleted.
	CurrentDepletion map[int]uint64 `yaml:"current_depletion,omitempty"`

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
// Use AddStockAtRound when you want depletion-event tracking
// (Phase-2 throughput scoring).
func (si *ShopInventory) AddStock(itemId int, qty int) {
	si.AddStockAtRound(itemId, qty, 0)
}

// AddStockAtRound is the round-aware variant: when round > 0 and the
// item is currently depleted (Current was 0 before this call), push
// a completed StockEvent into history and clear CurrentDepletion.
// round = 0 means "don't track" (used by Phase-2-naive callers
// that don't have a round handy).
func (si *ShopInventory) AddStockAtRound(itemId int, qty int, round uint64) {
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
	wasDepleted := entry.Current == 0
	entry.Current += qty
	if entry.Current > entry.MaxStock {
		entry.Current = entry.MaxStock
	}
	if qty > 0 {
		growRound := round
		if growRound == 0 {
			growRound = util.GetRoundCount()
		}
		entry.LastGrewRound = growRound
	}
	if !wasDepleted || qty <= 0 || round == 0 {
		return
	}
	depRound, ok := si.CurrentDepletion[itemId]
	if !ok {
		return
	}
	if si.StockEvents == nil {
		si.StockEvents = map[int][]StockEvent{}
	}
	si.StockEvents[itemId] = append(si.StockEvents[itemId], StockEvent{
		DepletedRound: depRound,
		RefilledRound: round,
	})
	delete(si.CurrentDepletion, itemId)
}

// RemoveStock decreases current stock by qty. Returns actual amount
// removed.
func (si *ShopInventory) RemoveStock(itemId int, qty int) int {
	return si.RemoveStockAtRound(itemId, qty, 0)
}

// RemoveStockAtRound is the round-aware variant: when round > 0 and
// the removal causes Current to hit 0, mark CurrentDepletion[itemId]
// = round so a later AddStockAtRound can produce a completed event.
func (si *ShopInventory) RemoveStockAtRound(itemId int, qty int, round uint64) int {
	entry := si.GetStock(itemId)
	if entry == nil || entry.Current <= 0 {
		return 0
	}
	removed := qty
	if removed > entry.Current {
		removed = entry.Current
	}
	entry.Current -= removed
	if entry.Current == 0 && round > 0 {
		if si.CurrentDepletion == nil {
			si.CurrentDepletion = map[int]uint64{}
		}
		if _, exists := si.CurrentDepletion[itemId]; !exists {
			si.CurrentDepletion[itemId] = round
		}
	}
	return removed
}

// Restock applies the supply cart delivery for all items with RestockQty > 0.
// Returns true if any stock was added.
func (si *ShopInventory) Restock() bool {
	restocked := false
	now := util.GetRoundCount()
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
		e.LastGrewRound = now
		si.RestockCount += add
		restocked = true
	}
	return restocked
}

// RestockBuckets is like Restock(), but only tops up entries whose
// item-id falls in one of the given supply buckets. Used by foragers
// (always one bucket per call — their region's) and the caravan (one
// or two buckets per call, based on caravan_load).
//
// As with Restock(), entries with RestockQty <= 0 (NPC-crafted items)
// are skipped — those don't come from the supply cart.
//
// Returns true if any stock was added. nil/empty buckets is a no-op.
func (si *ShopInventory) RestockBuckets(buckets []string) bool {
	if len(buckets) == 0 {
		return false
	}
	restocked := false
	now := util.GetRoundCount()
	for i := range si.Stock {
		e := &si.Stock[i]
		if e.RestockQty <= 0 {
			continue
		}
		bucket := economy.BucketFor(e.ItemId)
		if bucket == "" || !slices.Contains(buckets, bucket) {
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
		e.LastGrewRound = now
		si.RestockCount += add
		restocked = true
	}
	return restocked
}

// GoldReserve returns the minimum gold the NPC should hold back
// from discretionary purchases (gear upgrades).
func (si *ShopInventory) GoldReserve(ratio float64) int {
	return int(float64(si.StartingGold) * ratio)
}

// RestockBaselineTiers tops up StockEntries whose item carries
// rarity_tier 50 or 40, by RestockQty per call (capped at MaxStock).
// Skips entries with RestockQty <= 0 (NPC-crafted, untouched).
// Returns true if any stock was added.
//
// Used in caravan-served zones to layer baseline supply on top of
// caravan/forager deliveries — common mats (tiers 50/40) refill via
// this method on the existing crafter tick, while rarer mats (30/20/10)
// remain dependent on caravan/forager flow.
func (si *ShopInventory) RestockBaselineTiers() bool {
	restocked := false
	now := util.GetRoundCount()
	for i := range si.Stock {
		e := &si.Stock[i]
		if e.RestockQty <= 0 {
			continue
		}
		spec := items.GetItemSpec(e.ItemId)
		if spec == nil {
			continue
		}
		if spec.RarityTier != 50 && spec.RarityTier != 40 {
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
		e.LastGrewRound = now
		si.RestockCount += add
		restocked = true
	}
	return restocked
}

// RestockTier tops up StockEntries whose item carries the given
// rarity_tier, by RestockQty per call (capped at MaxStock). Skips
// entries with RestockQty <= 0 (NPC-crafted items don't restock).
// Returns true if any stock was added.
//
// Replaces the all-or-nothing Restock() / RestockBaselineTiers()
// approach with per-tier granularity, enabling per-rarity-tier
// cadences (commons restock often, rares rarely).
func (si *ShopInventory) RestockTier(rarityTier int) bool {
	restocked := false
	now := util.GetRoundCount()
	for i := range si.Stock {
		e := &si.Stock[i]
		if e.RestockQty <= 0 {
			continue
		}
		spec := items.GetItemSpec(e.ItemId)
		if spec == nil {
			continue
		}
		// Items without an explicit rarity_tier (typically pre-tier-system
		// content like tutorial gear) default to tier 50 for restock
		// purposes — otherwise they'd never be picked up by any per-tier
		// ticker (the Phase-1 ticker only iterates 50/40/30/20/10).
		itemTier := spec.RarityTier
		if itemTier == 0 {
			itemTier = 50
		}
		if itemTier != rarityTier {
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
		e.LastGrewRound = now
		si.RestockCount += add
		restocked = true
	}
	return restocked
}

// CanAfford returns true if spending amount would not drop below
// the given reserve floor.
func (si *ShopInventory) CanAfford(amount int, reserveFloor int) bool {
	return si.Gold-amount >= reserveFloor
}
