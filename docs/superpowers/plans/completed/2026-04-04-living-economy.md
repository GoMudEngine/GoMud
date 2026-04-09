# Living Economy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace infinite static shops with a living economy: finite stock, dynamic pricing, autonomous NPC crafting with profit logic, rule-based buying, and non-combatant protection.

**Architecture:** New `ShopInventory` type with YAML persistence in `_datafiles/world/dogmud/shops/`. Dynamic pricing via scarcity multiplier curve (0.25x–5.0x) normalized by restock quantity. NPC craft decisions use profit calculations comparing material opportunity cost vs product value. Buy rules are composable by NPC role. Non-combatant flag blocks all combat/theft initiation.

**Tech Stack:** Go, YAML persistence, existing crafting/items/mobs packages

**Spec:** `docs/superpowers/specs/2026-04-04-living-economy-design.md`

---

## File Structure

### New Files
| File | Responsibility |
|------|---------------|
| `internal/shops/shopinventory.go` | ShopInventory type, stock operations, gold tracking |
| `internal/shops/pricing.go` | ScarcityMultiplier, CalcSellPrice, CalcBuyPrice |
| `internal/shops/persistence.go` | Save/Load shop state to YAML files |
| `internal/shops/shopinventory_test.go` | Tests for stock operations and pricing |
| `internal/shops/persistence_test.go` | Tests for save/load round-trip |
| `internal/shops/buyrules.go` | Rule-based buy evaluation (gear upgrade, materials, potions, general) |
| `internal/shops/buyrules_test.go` | Tests for buy rule evaluation |
| `internal/shops/craftdecision.go` | NPC craft/salvage profit decisions |
| `internal/shops/craftdecision_test.go` | Tests for craft decision logic |
| `internal/items/compare.go` | ComparePower utility for gear upgrades |
| `internal/items/compare_test.go` | Tests for item power comparison |

### Modified Files
| File | Changes |
|------|---------|
| `internal/mobs/mobs.go` | Add `NonCombatant` field to Mob struct |
| `internal/mobs/crafter.go` | Replace `TickMobCraft` with new profit-driven system |
| `internal/usercommands/buy.go` | Read from ShopInventory, dynamic pricing |
| `internal/usercommands/sell.go` | Rule-based buy evaluation, dynamic pricing |
| `internal/usercommands/attack.go` | Non-combatant check |
| `internal/usercommands/bash.go` | Non-combatant check |
| `internal/usercommands/kick.go` | Non-combatant check |
| `internal/usercommands/trip.go` | Non-combatant check |
| `internal/usercommands/grapple.go` | Non-combatant check |
| `internal/usercommands/taunt.go` | Non-combatant check |
| `internal/usercommands/shoot.go` | Non-combatant check |
| `internal/usercommands/skill.skullduggery.steal.go` | Non-combatant check |
| `internal/actions/cast.go` | Non-combatant check for harm spells |
| `internal/hooks/spell_resolution.go` | Skip non-combatants in AoE |
| `internal/hooks/MobIdle_HandleIdleMobs.go` | Restock timer + new craft decision call |
| `internal/configs/gameplay.go` (or balance) | New config knobs |
| `_datafiles/config.yaml` | Default values for new knobs |
| 17 merchant mob YAMLs | Add `non_combatant`, convert shop format |

---

## Phase 1: Infrastructure (Tasks 1–4)

### Task 1: ShopInventory Type and Stock Operations

**Files:**
- Create: `internal/shops/shopinventory.go`
- Create: `internal/shops/shopinventory_test.go`

- [ ] **Step 1: Create the shops package with core types**

Create `internal/shops/shopinventory.go`:

```go
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
	Gold          int          `yaml:"gold"`
	StartingGold  int          `yaml:"starting_gold"`  // Seed value; used for gold reserve calc
	LastRestock   uint64       `yaml:"last_restock"`
	Stock         []StockEntry `yaml:"inventory"`
	KnownRecipes  []string     `yaml:"known_recipes,omitempty"` // Recipes the NPC knows

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
```

- [ ] **Step 2: Write tests for stock operations**

Create `internal/shops/shopinventory_test.go`:

```go
package shops

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetStock_Found(t *testing.T) {
	si := &ShopInventory{
		Stock: []StockEntry{
			{ItemId: 100, RestockQty: 5, MaxStock: 10, Current: 3},
		},
	}
	entry := si.GetStock(100)
	assert.NotNil(t, entry)
	assert.Equal(t, 3, entry.Current)
}

func TestGetStock_NotFound(t *testing.T) {
	si := &ShopInventory{Stock: []StockEntry{}}
	assert.Nil(t, si.GetStock(999))
}

func TestAddStock_Existing(t *testing.T) {
	si := &ShopInventory{
		Stock: []StockEntry{
			{ItemId: 100, RestockQty: 5, MaxStock: 10, Current: 3},
		},
	}
	si.AddStock(100, 5)
	assert.Equal(t, 8, si.GetStock(100).Current)
}

func TestAddStock_CapsAtMax(t *testing.T) {
	si := &ShopInventory{
		Stock: []StockEntry{
			{ItemId: 100, RestockQty: 5, MaxStock: 10, Current: 8},
		},
	}
	si.AddStock(100, 5)
	assert.Equal(t, 10, si.GetStock(100).Current)
}

func TestAddStock_NewItem(t *testing.T) {
	si := &ShopInventory{Stock: []StockEntry{}}
	si.AddStock(200, 3)
	entry := si.GetStock(200)
	assert.NotNil(t, entry)
	assert.Equal(t, 3, entry.Current)
	assert.Equal(t, 0, entry.RestockQty)
	assert.Equal(t, 20, entry.MaxStock)
}

func TestRemoveStock(t *testing.T) {
	si := &ShopInventory{
		Stock: []StockEntry{
			{ItemId: 100, RestockQty: 5, MaxStock: 10, Current: 3},
		},
	}
	removed := si.RemoveStock(100, 2)
	assert.Equal(t, 2, removed)
	assert.Equal(t, 1, si.GetStock(100).Current)
}

func TestRemoveStock_CapsAtZero(t *testing.T) {
	si := &ShopInventory{
		Stock: []StockEntry{
			{ItemId: 100, RestockQty: 5, MaxStock: 10, Current: 1},
		},
	}
	removed := si.RemoveStock(100, 5)
	assert.Equal(t, 1, removed)
	assert.Equal(t, 0, si.GetStock(100).Current)
}

func TestRestock(t *testing.T) {
	si := &ShopInventory{
		Stock: []StockEntry{
			{ItemId: 100, RestockQty: 5, MaxStock: 10, Current: 3},
			{ItemId: 200, RestockQty: 0, MaxStock: 8, Current: 2}, // crafted, not restocked
			{ItemId: 300, RestockQty: 3, MaxStock: 5, Current: 5}, // already full
		},
	}
	restocked := si.Restock()
	assert.True(t, restocked)
	assert.Equal(t, 8, si.GetStock(100).Current)  // 3 + 5
	assert.Equal(t, 2, si.GetStock(200).Current)   // unchanged
	assert.Equal(t, 5, si.GetStock(300).Current)   // unchanged (at max)
}

func TestRestock_PartialFill(t *testing.T) {
	si := &ShopInventory{
		Stock: []StockEntry{
			{ItemId: 100, RestockQty: 8, MaxStock: 10, Current: 7},
		},
	}
	si.Restock()
	assert.Equal(t, 10, si.GetStock(100).Current) // only room for 3, not full 8
}

func TestGoldReserve(t *testing.T) {
	si := &ShopInventory{StartingGold: 500}
	assert.Equal(t, 250, si.GoldReserve(0.50))
}

func TestCanAfford(t *testing.T) {
	si := &ShopInventory{Gold: 300}
	assert.True(t, si.CanAfford(50, 250))
	assert.False(t, si.CanAfford(100, 250))
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/shops/... -v`
Expected: All tests pass

- [ ] **Step 4: Commit**

```bash
git add internal/shops/shopinventory.go internal/shops/shopinventory_test.go
git commit -m "feat: add ShopInventory type with stock operations and tests"
```

---

### Task 2: Dynamic Pricing

**Files:**
- Create: `internal/shops/pricing.go`
- Modify: `internal/shops/shopinventory_test.go` (add pricing tests)

- [ ] **Step 1: Create pricing functions**

Create `internal/shops/pricing.go`:

```go
package shops

import (
	"math"
)

// PricingConfig holds the tunable knobs for dynamic pricing.
type PricingConfig struct {
	BuyRatio            float64 // Base buy/sell spread (default 0.50)
	PriceFloor          float64 // Min scarcity multiplier (default 0.25)
	PriceCeiling        float64 // Max scarcity multiplier (default 5.0)
	AbundanceThreshold  float64 // Stock/restock ratio for full abundance (default 3.0)
}

// DefaultPricingConfig returns sensible defaults.
func DefaultPricingConfig() PricingConfig {
	return PricingConfig{
		BuyRatio:           0.50,
		PriceFloor:         0.25,
		PriceCeiling:       5.0,
		AbundanceThreshold: 3.0,
	}
}

// ScarcityMultiplier computes the price multiplier based on current stock
// and the item's restock quantity (which normalizes the curve).
// Range: PriceFloor (overstocked) to PriceCeiling (out of stock).
func ScarcityMultiplier(current int, restockQty int, cfg PricingConfig) float64 {
	if restockQty <= 0 {
		// For NPC-crafted items with no restock, use MaxStock as normalizer.
		// Caller should pass maxStock as restockQty for these items.
		restockQty = 1 // Avoid division by zero; 1 makes the curve steep
	}

	ratio := float64(current) / float64(restockQty)

	if ratio <= 0 {
		return cfg.PriceCeiling
	}
	if ratio >= cfg.AbundanceThreshold {
		return cfg.PriceFloor
	}

	// Inverse quadratic: prices rise sharply as stock approaches zero
	t := ratio / cfg.AbundanceThreshold // 0.0 to 1.0
	mult := cfg.PriceFloor + (cfg.PriceCeiling-cfg.PriceFloor)*math.Pow(1.0-t, 2)
	return mult
}

// CalcSellPrice computes what the NPC charges a player to buy an item.
func CalcSellPrice(baseValue int, current int, restockQty int, cfg PricingConfig) int {
	mult := ScarcityMultiplier(current, restockQty, cfg)
	price := math.Ceil(float64(baseValue) * mult)
	if price < 1 {
		price = 1
	}
	return int(price)
}

// CalcBuyPrice computes what the NPC offers a player for an item.
func CalcBuyPrice(baseValue int, current int, restockQty int, cfg PricingConfig) int {
	mult := ScarcityMultiplier(current, restockQty, cfg)
	price := math.Ceil(float64(baseValue) * cfg.BuyRatio * mult)
	if price < 1 {
		price = 1
	}
	return int(price)
}

// ApplyBarterModifier adjusts a price based on the player's bartering skill.
// For sell prices (NPC selling to player): reduces price.
// For buy prices (NPC buying from player): increases price.
// discount/bonus are 0.0–1.0 representing max percentage adjustment.
func ApplyBarterSellDiscount(price int, discount float64) int {
	adjusted := float64(price) * (1.0 - discount)
	if adjusted < 1 {
		adjusted = 1
	}
	return int(math.Ceil(adjusted))
}

func ApplyBarterBuyBonus(price int, bonus float64) int {
	adjusted := float64(price) * (1.0 + bonus)
	return int(math.Ceil(adjusted))
}
```

- [ ] **Step 2: Write pricing tests**

Add to `internal/shops/shopinventory_test.go` (or create a separate `pricing_test.go`):

```go
func TestScarcityMultiplier_OutOfStock(t *testing.T) {
	cfg := DefaultPricingConfig()
	mult := ScarcityMultiplier(0, 5, cfg)
	assert.Equal(t, 5.0, mult)
}

func TestScarcityMultiplier_FullyAbundant(t *testing.T) {
	cfg := DefaultPricingConfig()
	mult := ScarcityMultiplier(15, 5, cfg) // ratio = 3.0 = threshold
	assert.Equal(t, 0.25, mult)
}

func TestScarcityMultiplier_AtRestock(t *testing.T) {
	cfg := DefaultPricingConfig()
	mult := ScarcityMultiplier(5, 5, cfg) // ratio = 1.0
	// Should be between floor and ceiling
	assert.Greater(t, mult, cfg.PriceFloor)
	assert.Less(t, mult, cfg.PriceCeiling)
}

func TestScarcityMultiplier_Monotonic(t *testing.T) {
	cfg := DefaultPricingConfig()
	// Prices should decrease as stock increases
	prev := ScarcityMultiplier(0, 5, cfg)
	for stock := 1; stock <= 15; stock++ {
		curr := ScarcityMultiplier(stock, 5, cfg)
		assert.LessOrEqual(t, curr, prev, "stock=%d should have lower mult than stock=%d", stock, stock-1)
		prev = curr
	}
}

func TestCalcSellPrice(t *testing.T) {
	cfg := DefaultPricingConfig()
	// Out of stock: 10g base × 5.0 = 50g
	assert.Equal(t, 50, CalcSellPrice(10, 0, 5, cfg))
}

func TestCalcBuyPrice(t *testing.T) {
	cfg := DefaultPricingConfig()
	// Out of stock: 10g base × 0.5 × 5.0 = 25g
	assert.Equal(t, 25, CalcBuyPrice(10, 0, 5, cfg))
}

func TestCalcBuyPrice_Abundant(t *testing.T) {
	cfg := DefaultPricingConfig()
	// Abundant: 10g base × 0.5 × 0.25 = 1.25 → 2 (ceil)
	price := CalcBuyPrice(10, 15, 5, cfg)
	assert.LessOrEqual(t, price, 2)
}

func TestBarterDiscount(t *testing.T) {
	assert.Equal(t, 85, ApplyBarterSellDiscount(100, 0.15))
}

func TestBarterBonus(t *testing.T) {
	assert.Equal(t, 115, ApplyBarterBuyBonus(100, 0.15))
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/shops/... -v`
Expected: All tests pass

- [ ] **Step 4: Commit**

```bash
git add internal/shops/pricing.go internal/shops/shopinventory_test.go
git commit -m "feat: add dynamic pricing with scarcity multiplier curve"
```

---

### Task 3: Shop Persistence

**Files:**
- Create: `internal/shops/persistence.go`
- Create: `internal/shops/persistence_test.go`

- [ ] **Step 1: Create persistence functions**

Create `internal/shops/persistence.go`:

```go
package shops

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v2"
)

var (
	shopCache   = make(map[string]*ShopInventory)
	shopCacheMu sync.RWMutex
)

// shopKey returns a unique key for a shop based on zone + mobId + roomId.
func shopKey(zone string, mobId int, roomId int) string {
	return fmt.Sprintf("%s/%d/room%d", util.ConvertForFilename(zone), mobId, roomId)
}

// shopPath returns the filesystem path for a shop's persistence file.
func shopPath(zone string, mobId int, roomId int) string {
	zonePath := util.ConvertForFilename(zone)
	filename := fmt.Sprintf("%d-room%d.yaml", mobId, roomId)
	return util.FilePath(
		configs.GetFilePathsConfig().DataFiles.String(), `/`, `shops`, `/`, zonePath, `/`, filename,
	)
}

// GetShopInventory returns the ShopInventory for a given mob, loading from
// disk or cache. Returns nil if no shop exists for this mob.
func GetShopInventory(zone string, mobId int, roomId int) *ShopInventory {
	key := shopKey(zone, mobId, roomId)

	shopCacheMu.RLock()
	if si, ok := shopCache[key]; ok {
		shopCacheMu.RUnlock()
		return si
	}
	shopCacheMu.RUnlock()

	// Try loading from disk
	si := loadFromDisk(zone, mobId, roomId)
	if si == nil {
		return nil
	}

	shopCacheMu.Lock()
	shopCache[key] = si
	shopCacheMu.Unlock()

	return si
}

// RegisterShop creates or loads a ShopInventory for a mob. If no persisted
// state exists, seeds from the provided template. Called at mob spawn.
func RegisterShop(zone string, mobId int, roomId int, template ShopInventory) *ShopInventory {
	key := shopKey(zone, mobId, roomId)

	shopCacheMu.Lock()
	defer shopCacheMu.Unlock()

	// Already cached?
	if si, ok := shopCache[key]; ok {
		return si
	}

	// Try loading persisted state
	si := loadFromDisk(zone, mobId, roomId)
	if si == nil {
		// First boot: seed from template
		si = &ShopInventory{
			Gold:         template.StartingGold,
			StartingGold: template.StartingGold,
			Stock:        make([]StockEntry, len(template.Stock)),
			KnownRecipes: template.KnownRecipes,
		}
		copy(si.Stock, template.Stock)
		// Seed current stock: materials get restockQty, crafted goods start at 0
		for i := range si.Stock {
			if si.Stock[i].RestockQty > 0 {
				si.Stock[i].Current = si.Stock[i].RestockQty
			}
		}
	}

	shopCache[key] = si
	return si
}

// SaveShop writes a shop's current state to disk.
func SaveShop(zone string, mobId int, roomId int) error {
	key := shopKey(zone, mobId, roomId)

	shopCacheMu.RLock()
	si, ok := shopCache[key]
	shopCacheMu.RUnlock()

	if !ok || si == nil {
		return nil
	}

	savePath := shopPath(zone, mobId, roomId)
	dir := filepath.Dir(savePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("shops.SaveShop: mkdir %s: %w", dir, err)
	}

	bytes, err := yaml.Marshal(si)
	if err != nil {
		return fmt.Errorf("shops.SaveShop: marshal: %w", err)
	}

	if err := os.WriteFile(savePath, bytes, 0644); err != nil {
		return fmt.Errorf("shops.SaveShop: write %s: %w", savePath, err)
	}

	return nil
}

// SaveAllShops persists all cached shop inventories to disk.
// Called at server shutdown.
func SaveAllShops() {
	shopCacheMu.RLock()
	keys := make([]string, 0, len(shopCache))
	for k := range shopCache {
		keys = append(keys, k)
	}
	shopCacheMu.RUnlock()

	// Each ShopInventory stores its own location for save path construction.
	saved := 0
	shopCacheMu.RLock()
	for _, si := range shopCache {
		if err := saveShopDirect(si); err != nil {
			mudlog.Error("shops.SaveAllShops", "error", err)
		} else {
			saved++
		}
	}
	shopCacheMu.RUnlock()
	mudlog.Info("shops.SaveAllShops", "saved", saved)
}

// loadFromDisk reads a shop file. Returns nil if not found.
func loadFromDisk(zone string, mobId int, roomId int) *ShopInventory {
	savePath := shopPath(zone, mobId, roomId)
	bytes, err := os.ReadFile(savePath)
	if err != nil {
		return nil
	}

	var si ShopInventory
	if err := yaml.Unmarshal(bytes, &si); err != nil {
		mudlog.Error("shops.loadFromDisk", "path", savePath, "error", err)
		return nil
	}
	return &si
}

// ClearCache removes all cached shop inventories (used in tests).
func ClearCache() {
	shopCacheMu.Lock()
	shopCache = make(map[string]*ShopInventory)
	shopCacheMu.Unlock()
}
```

Note: The `SaveAllShops` function needs the zone/mobId/roomId to construct the path. The implementer should either store these in the cache value struct or use the key parsing approach. Choose whichever is cleaner — the key format is `zone/mobId/roomRoomId` so it can be parsed back.

- [ ] **Step 2: Write persistence tests**

Create `internal/shops/persistence_test.go` with round-trip save/load tests using a temp directory. Test that:
- `RegisterShop` with no file on disk seeds from template
- `SaveShop` + `loadFromDisk` round-trips correctly
- `RegisterShop` with existing file on disk loads persisted state
- `GetShopInventory` returns cached data on second call
- `ClearCache` resets the cache

Use `t.TempDir()` for the test directory and override the data files path if possible, or test `loadFromDisk` / save at the unit level with direct file operations.

- [ ] **Step 3: Run tests**

Run: `go test ./internal/shops/... -v`
Expected: All tests pass

- [ ] **Step 4: Commit**

```bash
git add internal/shops/persistence.go internal/shops/persistence_test.go
git commit -m "feat: add shop inventory persistence (save/load YAML)"
```

---

### Task 4: Non-Combatant Flag

**Files:**
- Modify: `internal/mobs/mobs.go:58-104` (add field)
- Modify: `internal/usercommands/attack.go`
- Modify: `internal/usercommands/bash.go`
- Modify: `internal/usercommands/kick.go`
- Modify: `internal/usercommands/trip.go`
- Modify: `internal/usercommands/grapple.go`
- Modify: `internal/usercommands/taunt.go`
- Modify: `internal/usercommands/shoot.go`
- Modify: `internal/usercommands/skill.skullduggery.steal.go`
- Modify: `internal/actions/cast.go`
- Modify: `internal/hooks/spell_resolution.go`

- [ ] **Step 1: Add NonCombatant field to Mob struct**

In `internal/mobs/mobs.go`, add to the Mob struct (after `CharmImmune` field, around line 89):

```go
	NonCombatant            bool     `yaml:"non_combatant,omitempty"`           // If true, cannot be attacked, stolen from, or aggroed
```

- [ ] **Step 2: Add helper function**

In `internal/mobs/mobs.go`, add a public helper:

```go
// IsNonCombatant returns true if the mob is flagged as a non-combatant
// (shopkeepers, quest NPCs, etc.) that cannot be attacked or stolen from.
func (m *Mob) IsNonCombatant() bool {
	return m.NonCombatant
}
```

- [ ] **Step 3: Add non-combatant check to attack.go**

In `internal/usercommands/attack.go`, find where mob targets are resolved (the `mobs.GetInstance(attackMobInstanceId)` block). Before `SetAggro`, add:

```go
	if m.IsNonCombatant() {
		user.SendText(fmt.Sprintf(`You can't attack <ansi fg="mobname">%s</ansi>.`, m.Character.Name))
		return true, nil
	}
```

- [ ] **Step 4: Add non-combatant check to bash, kick, trip, grapple, taunt, shoot**

For each of bash.go, kick.go, trip.go, grapple.go, taunt.go — in the out-of-combat targeting block where `targetMId > 0`, add before `SetAggro`:

```go
			if m := mobs.GetInstance(targetMId); m != nil && m.IsNonCombatant() {
				user.SendText(fmt.Sprintf(`You can't attack <ansi fg="mobname">%s</ansi>.`, m.Character.Name))
				return true, nil
			}
```

For shoot.go, add the check after `res.IsCharmed` check and before the party check.

- [ ] **Step 5: Add non-combatant check to steal**

In `internal/usercommands/skill.skullduggery.steal.go`, find where the target mob is resolved. Add:

```go
	if mob.IsNonCombatant() {
		user.SendText(fmt.Sprintf(`You can't steal from <ansi fg="mobname">%s</ansi>.`, mob.Character.Name))
		return true, nil
	}
```

- [ ] **Step 6: Add non-combatant check to harm spells**

In `internal/actions/cast.go`, in the HarmSingle mob target block (after the companion check), add:

```go
				// Non-combatant check
				if actor.IsPlayer() {
					if m := mobs.GetInstance(mId); m != nil && m.IsNonCombatant() {
						actor.SendText(fmt.Sprintf("You can't target %s with a harmful spell.", m.Character.Name))
						return CastResult{SpellInfo: spellInfo, NoTarget: true}
					}
				}
```

In `internal/hooks/spell_resolution.go`, in the HarmArea mob target filtering (around line 73), add non-combatant skip alongside the companion check:

```go
			// Don't damage non-combatant NPCs
			if m := mobs.GetInstance(mId); m != nil && (m.Character.IsCharmed() || m.IsNonCombatant()) {
				continue
			}
```

- [ ] **Step 7: Verify build and tests**

Run: `go build ./... && go test ./internal/...`
Expected: All pass

- [ ] **Step 8: Commit**

```bash
git add internal/mobs/mobs.go internal/usercommands/attack.go internal/usercommands/bash.go internal/usercommands/kick.go internal/usercommands/trip.go internal/usercommands/grapple.go internal/usercommands/taunt.go internal/usercommands/shoot.go internal/usercommands/skill.skullduggery.steal.go internal/actions/cast.go internal/hooks/spell_resolution.go
git commit -m "feat: add non_combatant flag to protect shop NPCs from combat and theft"
```

---

## Phase 2: Buy/Sell Rework (Tasks 5–6)

### Task 5: Rewrite buy.go to Use ShopInventory

**Files:**
- Modify: `internal/usercommands/buy.go`

This is a significant rewrite. The current buy.go reads from `Character.Shop` (the old static list). The new version reads from `ShopInventory` with dynamic pricing.

- [ ] **Step 1: Read current buy.go fully**

Read the entire file to understand all paths: item purchase, buff purchase, mercenary purchase, pet purchase, multi-buy. The rewrite only changes item purchases (the most common path). Buff/mercenary/pet purchases can continue using the old Shop system for now — they don't have economic dynamics.

- [ ] **Step 2: Modify item price resolution**

In the `tryPurchase` function (or wherever item price is resolved), replace the static price lookup with:

```go
	// Dynamic pricing from ShopInventory
	shopInv := shops.GetShopInventory(shopMob.Zone, int(shopMob.MobId), shopMob.HomeRoomId)
	if shopInv == nil {
		// Fallback to legacy shop behavior if no ShopInventory registered
		// (non-migrated merchants)
		return legacyTryPurchase(...)
	}

	entry := shopInv.GetStock(itemSpec.ItemId)
	if entry == nil || entry.Current <= 0 {
		// Item not in stock
		return false
	}

	priceCfg := shops.PricingConfigFromBalance() // reads from config.yaml
	price := shops.CalcSellPrice(itemSpec.Value, entry.Current, effectiveRestock(entry), priceCfg)

	// Apply bartering discount
	barterSkill := user.Character.GetSkillLevel(skills.Bartering)
	discount := calcBarterDiscount(barterSkill)
	price = shops.ApplyBarterSellDiscount(price, discount)
```

On successful purchase:
```go
	shopInv.RemoveStock(itemSpec.ItemId, 1)
	shopInv.Gold += price
	shops.SaveShop(shopMob.Zone, int(shopMob.MobId), shopMob.HomeRoomId)
```

- [ ] **Step 3: Add helper for effective restock normalization**

For items with `RestockQty == 0` (crafted goods), use `MaxStock / 2` as the normalizer so the pricing curve works sensibly:

```go
func effectiveRestock(entry *shops.StockEntry) int {
	if entry.RestockQty > 0 {
		return entry.RestockQty
	}
	// Crafted goods: use half of max stock as the "normal" level
	norm := entry.MaxStock / 2
	if norm < 1 {
		norm = 1
	}
	return norm
}
```

- [ ] **Step 4: Verify build and existing tests**

Run: `go build ./... && go test ./internal/usercommands/...`
Expected: All pass

- [ ] **Step 5: Commit**

```bash
git add internal/usercommands/buy.go
git commit -m "feat: rewrite buy command to use ShopInventory with dynamic pricing"
```

---

### Task 6: Rewrite sell.go with Rule-Based Buy Behavior

**Files:**
- Create: `internal/shops/buyrules.go`
- Create: `internal/shops/buyrules_test.go`
- Modify: `internal/usercommands/sell.go`

- [ ] **Step 1: Create buy rule evaluation**

Create `internal/shops/buyrules.go`:

```go
package shops

import (
	"github.com/GoMudEngine/GoMud/internal/items"
)

// BuyOffer represents what an NPC is willing to pay for an item.
type BuyOffer struct {
	Price  int
	Reason string // "gear_upgrade", "craft_material", "potion", "general", ""
}

// EvaluateBuyRules checks all buy rules in priority order and returns
// the best offer, or a zero-price offer if the NPC won't buy.
//
// Parameters:
//   - item: the item being offered
//   - shopInv: the NPC's current shop inventory
//   - craftSkill: the NPC's craft skill (empty if not a crafter)
//   - buysGeneral: whether this NPC buys miscellaneous goods
//   - cfg: pricing config
//
// The caller is responsible for checking gold availability.
func EvaluateBuyRules(item items.Item, shopInv *ShopInventory, craftSkill string, buysGeneral bool, cfg PricingConfig) BuyOffer {
	spec := item.GetSpec()
	if spec.ItemId < 1 {
		return BuyOffer{}
	}

	// Quest items: never buy
	if spec.QuestToken != "" {
		return BuyOffer{}
	}

	// Rule 1: Gear upgrade — handled by caller (needs equipment comparison)
	// The caller checks this before calling EvaluateBuyRules.

	// Rule 2: Craft materials
	if craftSkill != "" && spec.ComponentTag != "" {
		entry := shopInv.GetStock(spec.ItemId)
		current := 0
		restock := 1
		if entry != nil {
			current = entry.Current
			restock = entry.RestockQty
			if entry.Current >= entry.MaxStock {
				// At capacity, don't buy more
				goto nextRule
			}
		}
		if restock <= 0 {
			restock = 1
		}
		price := CalcBuyPrice(spec.Value, current, restock, cfg)
		if price > 0 {
			return BuyOffer{Price: price, Reason: "craft_material"}
		}
	}
nextRule:

	// Rule 3: Potions (non-expired)
	if spec.Type == items.Potion {
		// Reject declining or spoiled potions
		phase := items.GetAgingPhase(item)
		if phase == items.AgingDeclining || phase == items.AgingSpoiled {
			return BuyOffer{}
		}
		entry := shopInv.GetStock(spec.ItemId)
		current := 0
		restock := 1
		if entry != nil {
			current = entry.Current
			restock = entry.RestockQty
			if entry.Current >= entry.MaxStock {
				return BuyOffer{}
			}
		}
		if restock <= 0 {
			restock = 1
		}
		price := CalcBuyPrice(spec.Value, current, restock, cfg)
		if price > 0 {
			return BuyOffer{Price: price, Reason: "potion"}
		}
	}

	// Rule 4: General goods
	if buysGeneral {
		// Steep discount — junk dealer
		price := int(float64(spec.Value) * 0.25)
		if price < 1 {
			price = 1
		}
		return BuyOffer{Price: price, Reason: "general"}
	}

	return BuyOffer{}
}
```

Note: The implementer must check exact type names for `items.Potion` and `items.GetAgingPhase`. Read `internal/items/` to find the correct constants and function signatures. The aging system uses `GetAgingPhase(item)` which returns phase constants — verify the exact names.

- [ ] **Step 2: Write buy rule tests**

Create `internal/shops/buyrules_test.go` with tests for:
- Craft material offered to a crafter NPC → gets a price
- Craft material offered to a non-crafter → no offer
- Quest item → always rejected
- Item at max_stock → rejected
- General goods at a general merchant → 25% price
- Potion in fresh phase → accepted
- Potion in spoiled phase → rejected

- [ ] **Step 3: Rewrite sell.go**

Replace `mob.GetSellPrice(item)` with the new rule-based system:

```go
	shopInv := shops.GetShopInventory(mob.Zone, int(mob.MobId), mob.HomeRoomId)
	if shopInv == nil {
		// Legacy fallback
		sellValue = mob.GetSellPrice(item)
	} else {
		cfg := shops.PricingConfigFromBalance()
		offer := shops.EvaluateBuyRules(item, shopInv, mob.CrafterSkill, mob.BuysGeneral, cfg)
		sellValue = offer.Price

		// Apply bartering bonus
		barterSkill := user.Character.GetSkillLevel(skills.Bartering)
		bonus := calcBarterBonus(barterSkill)
		sellValue = shops.ApplyBarterBuyBonus(sellValue, bonus)
	}
```

On successful sale, update shop inventory:
```go
	shopInv.AddStock(item.ItemId, 1)
	shopInv.Gold -= sellValue
	shops.SaveShop(mob.Zone, int(mob.MobId), mob.HomeRoomId)
```

- [ ] **Step 4: Add BuysGeneral field to Mob struct**

In `internal/mobs/mobs.go`, add:
```go
	BuysGeneral             bool     `yaml:"buys_general,omitempty"`            // Whether this merchant buys misc goods
```

- [ ] **Step 5: Verify build and tests**

Run: `go build ./... && go test ./internal/shops/... ./internal/usercommands/...`
Expected: All pass

- [ ] **Step 6: Commit**

```bash
git add internal/shops/buyrules.go internal/shops/buyrules_test.go internal/usercommands/sell.go internal/mobs/mobs.go
git commit -m "feat: rule-based NPC buy behavior with dynamic pricing"
```

---

## Phase 3: Craft Decision Rework (Tasks 7–8)

### Task 7: Item Power Comparison Utility

**Files:**
- Create: `internal/items/compare.go`
- Create: `internal/items/compare_test.go`

- [ ] **Step 1: Create ComparePower function**

Create `internal/items/compare.go`:

```go
package items

// ItemPower computes a rough numeric power score for an item based on
// its stat modifiers, damage multiplier, and mitigation values.
// Used by NPC craft/buy logic to decide if an item is an upgrade.
func ItemPower(spec ItemSpec) float64 {
	power := 0.0

	// Stat modifiers
	for _, mod := range spec.StatMods {
		power += float64(mod.Value)
	}

	// Damage (weapon)
	power += spec.DamageMultiplier * 100

	// Spell damage (caster weapon)
	power += spec.SpellDamageMultiplier * 50

	// Mitigation (armor)
	power += float64(spec.PhysicalMitigation)
	power += float64(spec.MagicalMitigation)
	power += float64(spec.ConvictionMitigation)

	return power
}

// IsUpgrade returns true if candidate is strictly better than current
// for the same equipment slot.
func IsUpgrade(current ItemSpec, candidate ItemSpec) bool {
	if candidate.ItemId == 0 {
		return false
	}
	// Must be same slot type
	if current.Type != candidate.Type {
		return false
	}
	return ItemPower(candidate) > ItemPower(current)
}
```

Note: The implementer must verify the exact field names on `ItemSpec` — `StatMods`, `DamageMultiplier`, `SpellDamageMultiplier`, `PhysicalMitigation`, `MagicalMitigation`, `ConvictionMitigation`. Read `internal/items/itemspec.go` to confirm.

- [ ] **Step 2: Write tests**

Create `internal/items/compare_test.go` with tests for:
- Empty item has zero power
- Weapon with damage multiplier scores higher than one without
- Armor with mitigation scores higher
- `IsUpgrade` returns true when candidate has higher power
- `IsUpgrade` returns false for different slot types

- [ ] **Step 3: Verify**

Run: `go test ./internal/items/... -v`
Expected: All pass

- [ ] **Step 4: Commit**

```bash
git add internal/items/compare.go internal/items/compare_test.go
git commit -m "feat: add ItemPower and IsUpgrade utility for NPC gear decisions"
```

---

### Task 8: Profit-Driven Craft and Salvage Decisions

**Files:**
- Create: `internal/shops/craftdecision.go`
- Create: `internal/shops/craftdecision_test.go`
- Modify: `internal/mobs/crafter.go`
- Modify: `internal/hooks/MobIdle_HandleIdleMobs.go`

- [ ] **Step 1: Create craft decision logic**

Create `internal/shops/craftdecision.go`:

```go
package shops

import (
	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// CraftDecision represents the NPC's chosen action for this tick.
type CraftDecision struct {
	Action    string            // "craft", "salvage", or "" (nothing to do)
	RecipeId  string            // Recipe to craft (if Action == "craft")
	ItemId    int               // Item to salvage (if Action == "salvage")
	Profit    float64           // Expected profit margin
	ForSelf   bool              // Crafting for self-gear upgrade
}

// MaterialCost calculates the opportunity cost of consuming ingredients
// from the shop's inventory at current scarcity prices.
func MaterialCost(recipe *crafting.RecipeSpec, shopInv *ShopInventory, cfg PricingConfig) float64 {
	cost := 0.0
	for _, ing := range recipe.Ingredients {
		entry := shopInv.GetStock(ing.ItemId)
		current := 0
		restock := 1
		if entry != nil {
			current = entry.Current
			restock = entry.RestockQty
			if restock <= 0 {
				restock = 1
			}
		}
		spec := items.New(ing.ItemId).GetSpec()
		unitCost := float64(CalcSellPrice(spec.Value, current, restock, cfg))
		cost += unitCost * float64(ing.Quantity)
	}
	return cost
}

// ProductValue calculates what the crafted output would be worth at
// current stock levels.
func ProductValue(recipe *crafting.RecipeSpec, shopInv *ShopInventory, cfg PricingConfig) float64 {
	entry := shopInv.GetStock(recipe.Output.ItemId)
	current := 0
	maxStock := 8
	if entry != nil {
		current = entry.Current
		maxStock = entry.MaxStock
	}
	norm := maxStock / 2
	if norm < 1 {
		norm = 1
	}
	spec := items.New(recipe.Output.ItemId).GetSpec()
	return float64(CalcSellPrice(spec.Value, current, norm, cfg)) * float64(recipe.Output.Quantity)
}

// HasMaterialsWithReserve checks if the shop has all ingredients for a
// recipe while respecting the material reserve threshold.
func HasMaterialsWithReserve(recipe *crafting.RecipeSpec, shopInv *ShopInventory, reserve int) bool {
	for _, ing := range recipe.Ingredients {
		entry := shopInv.GetStock(ing.ItemId)
		if entry == nil {
			return false
		}
		// Must have enough to craft AND still keep reserve
		if entry.Current-ing.Quantity < reserve {
			return false
		}
	}
	return true
}

// EvaluateCraftOptions scores all profitable craft options and returns
// the best one. Returns nil CraftDecision if nothing is profitable.
func EvaluateCraftOptions(recipes []*crafting.RecipeSpec, shopInv *ShopInventory, cfg PricingConfig, reserve int) *CraftDecision {
	var best *CraftDecision

	for _, recipe := range recipes {
		// Check max stock
		entry := shopInv.GetStock(recipe.Output.ItemId)
		if entry != nil && entry.Current >= entry.MaxStock {
			continue
		}

		if !HasMaterialsWithReserve(recipe, shopInv, reserve) {
			continue
		}

		matCost := MaterialCost(recipe, shopInv, cfg)
		prodVal := ProductValue(recipe, shopInv, cfg)

		if prodVal <= matCost {
			continue // not profitable
		}

		profit := prodVal - matCost
		if best == nil || profit > best.Profit {
			best = &CraftDecision{
				Action:   "craft",
				RecipeId: recipe.RecipeId,
				Profit:   profit,
			}
		}
	}

	return best
}

// EvaluateSalvageOptions finds the most profitable item to break down.
// Returns nil if no salvage is profitable.
func EvaluateSalvageOptions(shopInv *ShopInventory, cfg PricingConfig) *CraftDecision {
	var best *CraftDecision

	for _, entry := range shopInv.Stock {
		if entry.Current <= 1 {
			continue // keep at least 1
		}
		if entry.RestockQty > 0 {
			continue // don't salvage raw materials
		}

		spec := items.New(entry.ItemId).GetSpec()
		if len(spec.SalvageReturns) == 0 {
			continue // can't be salvaged
		}

		norm := entry.MaxStock / 2
		if norm < 1 {
			norm = 1
		}
		itemValue := float64(CalcSellPrice(spec.Value, entry.Current, norm, cfg))

		// Calculate expected salvage return value
		salvageValue := 0.0
		for _, ret := range spec.SalvageReturns {
			retSpec := items.New(ret.ItemId).GetSpec()
			retEntry := shopInv.GetStock(ret.ItemId)
			retCurrent := 0
			retRestock := 1
			if retEntry != nil {
				retCurrent = retEntry.Current
				retRestock = retEntry.RestockQty
				if retRestock <= 0 {
					retRestock = 1
				}
			}
			unitVal := float64(CalcSellPrice(retSpec.Value, retCurrent, retRestock, cfg))
			salvageValue += unitVal * float64(ret.Quantity)
		}

		if salvageValue <= itemValue {
			continue
		}

		profit := salvageValue - itemValue
		if best == nil || profit > best.Profit {
			best = &CraftDecision{
				Action: "salvage",
				ItemId: entry.ItemId,
				Profit: profit,
			}
		}
	}

	return best
}
```

Note: The implementer must verify exact field names for `recipe.Ingredients` (may be `Ingredients` or `Components`), `recipe.Output`, and `spec.SalvageReturns`. Read `internal/crafting/crafting.go` and `internal/items/itemspec.go` to confirm the struct fields.

- [ ] **Step 2: Write craft decision tests**

Create `internal/shops/craftdecision_test.go` with tests for:
- Profitable craft: material cost < product value → returns craft decision
- Unprofitable craft: material cost > product value → returns nil
- At max stock: skips recipe
- Reserve respected: won't craft if it would drop below reserve
- Salvage: overstock item with salvage returns → returns salvage decision
- Salvage: item worth more than salvage returns → no salvage

- [ ] **Step 3: Rewrite TickMobCraft in crafter.go**

Replace the existing `TickMobCraft` and `pickEligibleRecipe` functions in `internal/mobs/crafter.go` with the new profit-driven system:

1. Check restock timer (existing logic)
2. Call `shopInv.Restock()` for material delivery
3. Fire restock emote to room
4. Priority 1: Check self-gear upgrade (loop equipment slots, check if any known recipe produces an upgrade via `items.IsUpgrade`)
5. Priority 2: Call `shops.EvaluateCraftOptions(recipes, shopInv, cfg, reserve)`
6. Priority 3: Call `shops.EvaluateSalvageOptions(shopInv, cfg)`
7. Execute the chosen action (craft or salvage)

The existing `CraftResult` struct can be extended with a `Salvaged bool` field.

- [ ] **Step 4: Update MobIdle hook**

In `internal/hooks/MobIdle_HandleIdleMobs.go`, update the crafter tick block (around line 54) to pass the shop inventory context and handle restock emotes.

- [ ] **Step 5: Verify build and tests**

Run: `go build ./... && go test ./internal/shops/... ./internal/mobs/...`
Expected: All pass

- [ ] **Step 6: Commit**

```bash
git add internal/shops/craftdecision.go internal/shops/craftdecision_test.go internal/mobs/crafter.go internal/hooks/MobIdle_HandleIdleMobs.go
git commit -m "feat: profit-driven NPC craft/salvage decisions with self-gear priority"
```

---

## Phase 4: Data Migration and Config (Tasks 9–11)

### Task 9: Add Config Knobs

**Files:**
- Modify: `_datafiles/config.yaml`
- Modify: config structs (likely `internal/configs/` — find the Balance struct)

- [ ] **Step 1: Add config fields**

Add to the Balance config struct:

```go
	ShopBuyRatio            float64 `yaml:"ShopBuyRatio,omitempty"`            // default 0.50
	ShopPriceFloor          float64 `yaml:"ShopPriceFloor,omitempty"`          // default 0.25
	ShopPriceCeiling        float64 `yaml:"ShopPriceCeiling,omitempty"`        // default 5.0
	ShopAbundanceThreshold  float64 `yaml:"ShopAbundanceThreshold,omitempty"`  // default 3.0
	ShopMaterialReserve     int     `yaml:"ShopMaterialReserve,omitempty"`     // default 1
	ShopGoldReserveRatio    float64 `yaml:"ShopGoldReserveRatio,omitempty"`    // default 0.50
	BarterMaxDiscount       float64 `yaml:"BarterMaxDiscount,omitempty"`       // default 0.15
	BarterMaxBonus          float64 `yaml:"BarterMaxBonus,omitempty"`          // default 0.15
```

- [ ] **Step 2: Add defaults to config.yaml**

```yaml
  # --- Shop Economy ---
  ShopBuyRatio: 0.50
  ShopPriceFloor: 0.25
  ShopPriceCeiling: 5.0
  ShopAbundanceThreshold: 3.0
  ShopMaterialReserve: 1
  ShopGoldReserveRatio: 0.50
  BarterMaxDiscount: 0.15
  BarterMaxBonus: 0.15
```

- [ ] **Step 3: Create PricingConfigFromBalance helper**

In `internal/shops/pricing.go`, add:

```go
// PricingConfigFromBalance creates a PricingConfig from the game's balance settings.
func PricingConfigFromBalance() PricingConfig {
	b := configs.GetBalanceConfig()
	cfg := DefaultPricingConfig()
	if b.ShopBuyRatio > 0 {
		cfg.BuyRatio = float64(b.ShopBuyRatio)
	}
	if b.ShopPriceFloor > 0 {
		cfg.PriceFloor = float64(b.ShopPriceFloor)
	}
	if b.ShopPriceCeiling > 0 {
		cfg.PriceCeiling = float64(b.ShopPriceCeiling)
	}
	if b.ShopAbundanceThreshold > 0 {
		cfg.AbundanceThreshold = float64(b.ShopAbundanceThreshold)
	}
	return cfg
}
```

This requires importing `configs` in the shops package. Check for import cycles — if `configs` imports `shops`, this won't work and the helper should live in the calling code instead.

- [ ] **Step 4: Verify build**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/configs/ _datafiles/config.yaml internal/shops/pricing.go
git commit -m "feat: add shop economy config knobs with defaults"
```

---

### Task 10: Migrate Merchant Mob YAMLs

**Files:**
- Modify: 17 merchant mob YAML files
- Modify: `internal/mobs/mobs.go` (shop template parsing)

- [ ] **Step 1: Add ShopTemplate field to Mob struct**

Add a new field for the persistent shop configuration:

```go
	ShopTemplate            *shops.ShopInventory `yaml:"shoptemplate,omitempty"`   // Persistent shop config (replaces old Shop for economy)
```

Or alternatively, add the new shop fields inline in the mob YAML under a `shopconfig:` key. The implementer should choose the approach that fits the existing YAML loading pattern.

- [ ] **Step 2: Update mob spawn to register shops**

In the mob spawn path (wherever mobs are instantiated from templates), add:

```go
	if mob has shopconfig {
		shops.RegisterShop(zone, mobId, roomId, shopTemplate)
	}
```

- [ ] **Step 3: Convert Blacksmith Kerra as reference**

Update `_datafiles/world/dogmud/mobs/thornwall_city/97-blacksmith_kerra.yaml`:

```yaml
non_combatant: true
shopconfig:
  starting_gold: 500
  buys_general: false
  craft_skill: blacksmithing
  starting_recipes:
    - iron-dagger
    - iron-buckler
    - iron-short-sword
  restock_interval: 1440
  restock_emote: >-
    A supply cart rumbles to a stop outside the smithy.
    The driver unloads crates of raw materials before moving on.
  stock:
    - item_id: 40001
      restock_qty: 8
      max_stock: 15
    - item_id: 40002
      restock_qty: 6
      max_stock: 12
    - item_id: 40003
      restock_qty: 4
      max_stock: 10
    - item_id: 40020
      restock_qty: 4
      max_stock: 10
    - item_id: 10005
      restock_qty: 0
      max_stock: 8
```

- [ ] **Step 4: Convert remaining 16 merchants**

For each merchant, determine:
- `starting_gold` based on their importance (100–500)
- `restock_qty` and `max_stock` per item based on rarity
- `non_combatant: true` for all town merchants
- `buys_general: true` for general merchants and fence dealers
- `craft_skill` and `starting_recipes` for crafter merchants
- `restock_emote` appropriate to their location

Keep the old `shop:` field intact as a legacy fallback until all code paths are migrated.

- [ ] **Step 5: Verify server starts with migrated data**

Run: `go build ./... && go run .` (or however the server starts)
Expected: No panics, merchants accessible

- [ ] **Step 6: Commit**

```bash
git add _datafiles/world/dogmud/mobs/ internal/mobs/mobs.go
git commit -m "feat: migrate merchant mobs to living economy shop format"
```

---

### Task 11: Documentation Updates

**Files:**
- Modify: `internal/mobs/context.md`
- Modify: `internal/items/context.md`
- Modify: `internal/characters/context.md`
- Modify: `CLAUDE.md`

- [ ] **Step 1: Update mobs/context.md**

Add sections covering:
- `ShopInventory` persistence model and `shops/` directory
- Non-combatant flag behavior
- NPC craft decision priority (self-gear → profitable stock → salvage)
- Restock timer mechanics

- [ ] **Step 2: Update items/context.md**

Add section covering:
- `ComparePower` / `IsUpgrade` utility functions
- Salvage integration with shop NPCs

- [ ] **Step 3: Update characters/context.md**

Add note that shop inventory is decoupled from Character.Items for merchant NPCs.

- [ ] **Step 4: Update CLAUDE.md**

Add to the "Room Instance Saves" section or create a new "Shop Persistence" section:

```markdown
## Shop Persistence (Living Economy)
Shop economic state (stock levels, gold, restock timers) persists in
`_datafiles/world/dogmud/shops/{zone}/{mobid}-room{roomid}.yaml`.
This is separate from `rooms.instances/` and `mobs.instances/` and is
NOT cleaned by the instance save cleanup SOP. Deleting a shop file
resets that merchant to template defaults.
```

- [ ] **Step 5: Commit**

```bash
git add internal/mobs/context.md internal/items/context.md internal/characters/context.md CLAUDE.md
git commit -m "docs: update context files and CLAUDE.md for living economy system"
```

---

### Task 12: Full Build and Test Verification

**Files:** None (verification only)

- [ ] **Step 1: Full build**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 2: Run all tests**

Run: `go test ./... 2>&1 | tail -30`
Expected: All tests pass

- [ ] **Step 3: Smoke test**

Start the server and verify:
- Buy from blacksmith Kerra → price reflects stock level
- Buy multiple of same item → price increases with each purchase
- Sell an iron ore to Kerra → she buys it, price based on her stock
- Wait for restock timer → supply cart message appears, stock refills
- Kerra crafts on idle tick → new item appears in shop
- Try to attack Kerra → "You can't attack blacksmith Kerra."
- Try to steal from Kerra → "You can't steal from blacksmith Kerra."
- Cast harm spell on Kerra → "You can't target blacksmith Kerra..."

- [ ] **Step 4: Commit any fixups**

```bash
git add -A
git commit -m "fix: post-integration fixups for living economy"
```
