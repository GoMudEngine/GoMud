# Shops Package Context

## Overview

The `internal/shops` package implements the living-economy shopkeeper system:
persistent shop state (stock levels, merchant gold), dynamic pricing, restock
cadence, and buy/sell rule evaluation. Shop state survives server restarts and
reflects real NPC and player activity over time.

This package is the storage and pricing tier. Behavior that drives stock
change (forager deliveries, NPC sells, player purchases) lives in
`internal/behaviortree/`, `internal/forager/`, and `internal/actions/`.

## Key Components

### Core Files
- **shopinventory.go**: `ShopInventory` struct + `StockEntry`; `GetStock`,
  `AddStock`, `AddStockAtRound` stock-mutation helpers; `CraftSupport` tag
  constants; `StockEvent` depletion/refill event type.
- **persistence.go**: Disk I/O (`GetShopInventory`, `SaveShop`); in-memory
  shop cache; prewarm helpers.
- **pricing.go**: Dynamic pricing (`GetSellPrice`), `PricingConfig`,
  `PricingConfigFromBalance`; barter helpers.
- **buyrules.go**: `EvaluateBuyRules` — merchant acceptance and price
  negotiation when a seller brings an item to the shop.
- **restock_cadence.go**: Per-tier restock logic; `LastRestockByTier`.
- **overstock_decay.go**: `TickOverstockDecay` / `TickOverstockDecayWith` —
  time-based drain of unsold above-baseline stock (chunk 5.4).
- **effective_max_stock.go**: `EffectiveMaxStock` — caps MaxStock when the
  shop is below the gold reserve threshold.
- **craftdecision.go**: `ShouldCraftNow` — evaluates whether the shopkeeper's
  crafter NPC should fire this tick.
- **validation.go**: `ValidateShopMobTags` — startup panic if any shop-bearing
  mob is missing a `craft_support` tag.

## Key Functions

### Stock Management
- **`GetStock(itemId int) *StockEntry`**: Returns the live entry for an item,
  or nil if not stocked.
- **`AddStock(itemId int, qty int)`**: Increases current stock up to MaxStock;
  no depletion-event tracking.
- **`AddStockAtRound(itemId int, qty int, round uint64)`**: Round-aware variant.
  When the item was at 0 before this call, pushes a completed `StockEvent`
  into `StockEvents` history and clears `CurrentDepletion`. Pass `round = 0`
  to skip event tracking.

### Pricing
- **`GetSellPrice(item items.Item) int`**: Base buy-back price for legacy shop
  path (no `ShopInventory`).
- **`PricingConfigFromBalance() PricingConfig`**: Builds a `PricingConfig`
  snapshot from the current balance config; used by both pricing and buy-rules.
- **`EvaluateBuyRules(item, si, crafterSkill, buysGeneral, cfg, worn) BuyOffer`**:
  Determines whether the merchant will buy an item and at what price. Returns
  `offer.Price > 0` when the merchant accepts.
- **`ApplyBarterBuyBonus(price int, bonus float64) int`**: Applies a
  fractional barter bonus to a buy price (seller side).

### Overstock Decay (chunk 5.4)
- **`TickOverstockDecay(si *ShopInventory, round uint64)`**: Reads balance
  config and delegates to `TickOverstockDecayWith`.
- **`TickOverstockDecayWith(si, round, isComponent, decayRounds, decayQty)`**:
  Testable core. For each `StockEntry` whose `Current > RestockQty` and whose
  `LastGrewRound` is older than `decayRounds`, removes `decayQty` units (never
  below `RestockQty`), then re-stamps `LastGrewRound` to pace subsequent
  decays. Crafting materials (`is_component`) are always skipped.

  **Key semantics:**
  - `RestockQty` is the baseline. Items with `RestockQty 0` (NPC-dumped /
    backfilled entries) drain fully to 0 when unsold.
  - The grace period (`decayRounds`) is measured from `LastGrewRound`, so
    items that are actively receiving forager deliveries decay more slowly
    than items that have been sitting since the last restock.
  - Crafting materials are excluded so supply for crafting recipes is never
    silently eroded by the decay pass.

  Config knobs:
  - `ShopOverstockDecayRounds` (default 21600) — rounds between decay fires
    per entry.
  - `ShopOverstockDecayQty` (default 1) — units removed per fire.

### `StockEntry.LastGrewRound`
Set by `AddStockAtRound` whenever stock increases (forager delivery,
backfill, restock tick, or player sale). Read by `TickOverstockDecayWith` to
enforce the grace period. Zero means "never grown"; a zero `LastGrewRound`
satisfies the decay condition immediately.

## Global State

### Shop Cache
- **`shopCache`**: `map[string]*ShopInventory` keyed by
  `"{zone}/{mobId}-room{roomId}"`. Populated lazily by `GetShopInventory`,
  persisted by `SaveShop`.
- **`shopCacheMu`**: `sync.RWMutex` protecting the cache.

### Persistence
Shop state lives at
`_datafiles/world/dogmud/shops/{zone}/{mobId}-room{roomId}.yaml`. This
directory is **not** wiped by the instance-save cleanup SOP — it is
persistent living-economy state. Deleting a file resets that merchant to
template defaults (500g starting gold, base stock levels).

## Data Structure Design

### StockEntry
```go
type StockEntry struct {
    ItemId        int    `yaml:"item_id"`
    RestockQty    int    `yaml:"restock_qty"`               // supply-cart quantity per restock (0 = NPC/forager only)
    MaxStock      int    `yaml:"max_stock"`                 // hard cap on accumulation
    Current       int    `yaml:"current"`                   // persisted live count
    LastGrewRound uint64 `yaml:"last_grew_round,omitempty"` // drives overstock decay grace period
}
```

### ShopInventory (abbreviated)
```go
type ShopInventory struct {
    Gold                   int
    StartingGold           int
    LastRestock            uint64
    Stock                  []StockEntry
    KnownRecipes           []string
    CraftSupport           string
    LastRestockByTier      map[int]uint64
    SalesCount             int
    BuysCount              int
    RestockCount           int
    ConsumedByCrafterCount int
    StockEvents            map[int][]StockEvent
    CurrentDepletion       map[int]uint64
    // Location fields (not persisted)
    Zone   string
    MobId  int
    RoomId int
}
```

## Integration Notes

### Stock Contributors
- **`internal/behaviortree/`** — `TickOverstockDecay` is called from the
  shop restock hook each server tick.
- **`internal/forager/vendor_sell.go`** — `SellToVendor` transfers items
  directly into `StockEntry.Current` (free supply handoff, no gold).
- **`internal/forager/chest_backfill.go`** — `BackfillVendorFromChests`
  tops off vendors from forager lockboxes (also free, no gold).
- **`internal/actions/sell.go`** — player and mob sales use
  `EvaluateBuyRules` + `AddStockAtRound`; player sells draw down
  `ShopInventory.Gold`, mob sells do not (see `actions` package context.md).

### Consumers
- **`internal/usercommands/buy.go`**, **`internal/mobcommands/buy.go`**,
  **`internal/actions/buy.go`** — purchase flow reads pricing + stock.
- **`internal/economy/`** — economy dashboard reads `ShopInventory` for
  throughput and health scoring.
- **`internal/hooks/`** — restock cadence hook fires `TickOverstockDecay`
  each tick.

## Testing Notes

- **overstock_decay_test.go**: Covers baseline-clamping, crafting-material
  exclusion, grace-period gating, pacing (re-stamp after decay), nil/zero-qty
  guards.
- **pricing_test.go**, **buyrules_test.go**, **restock_cadence_test.go**,
  **effective_max_stock_test.go**: Unit coverage for each sub-system.
- `TickOverstockDecayWith` injects `isComponent`, `decayRounds`, and
  `decayQty` so tests don't need loaded item specs or a live config stack.
