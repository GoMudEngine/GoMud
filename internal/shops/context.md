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
  time-based drain of unsold above-baseline stock; now returns
  `[]DecayedUnit` so callers can act on what was removed (chunk 5.4).
- **enchant_reserve.go**: `AddToReserve` / `ReservePool` / `DrainReserve`
  — global in-memory pool of enchanting mats produced by alchemy-vendor
  potion decay; drawn by enchanters neediest-gap-first (chunk 5.4).
- **stock_transfers.go**: `SelectStockTransfers` — shared neediest-gap-first
  allocator; used by both the forager chest backfill and the enchanter
  reserve draw (chunk 5.4).
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
- **`TickOverstockDecay(si *ShopInventory, round uint64) []DecayedUnit`**:
  Reads balance config and delegates to `TickOverstockDecayWith`. Returns
  the set of items and quantities removed this sweep so callers can act on
  them (e.g. convert decayed potions to enchanting mats).
- **`TickOverstockDecayWith(si, round, isComponent, decayRounds, decayQty) []DecayedUnit`**:
  Testable core. For each `StockEntry` whose `Current > RestockQty` and whose
  `LastGrewRound` is older than `decayRounds`, removes `decayQty` units (never
  below `RestockQty`), then re-stamps `LastGrewRound` to pace subsequent
  decays. Crafting materials (`is_component`) are always skipped.
  Returns a `[]DecayedUnit` — one entry per affected `StockEntry`.

  **Key semantics:**
  - `RestockQty` is the baseline. Items with `RestockQty 0` (NPC-dumped /
    backfilled entries) drain fully to 0 when unsold.
  - The grace period (`decayRounds`) is measured from `LastGrewRound`, so
    items that are actively receiving forager deliveries decay more slowly
    than items that have been sitting since the last restock.
  - Crafting materials are excluded so supply for crafting recipes is never
    silently eroded by the decay pass.
  - The caller in `hooks/MobIdle_HandleIdleMobs.go` inspects each
    `DecayedUnit`: if `ItemSpec.Type == items.Potion`, it calls
    `crafting.EnchantSalvageYield` and feeds the resulting mats into
    `AddToReserve`.

  Config knobs:
  - `ShopOverstockDecayRounds` (default 21600) — rounds between decay fires
    per entry.
  - `ShopOverstockDecayQty` (default 1) — units removed per fire.

### Global Enchant-Mat Reserve (chunk 5.4)

The enchant reserve is a virtual analog of the forager chests: alchemy
vendors (Ilsa's, Voss's) accumulate enchanting mats here as their
unsold potions decay; enchanting-supply vendors (Vael, future
enchanters with `craft_support: "enchanting"`) draw from it to fill
their own stock gaps.

- **`AddToReserve(matItemId, qty int)`**: Adds `qty` units of
  `matItemId` to the global reserve. No-op for zero ID or qty. Called
  by the shop-restock hook after each decayed-potion conversion.
- **`ReservePool() map[int]int`**: Returns a snapshot copy of the
  current reserve (safe to iterate and mutate without holding the
  lock). Called by the enchanter draw path to build the input pool for
  `SelectStockTransfers`.
- **`DrainReserve(matItemId, qty int)`**: Removes up to `qty` units of
  `matItemId`; safe when `qty` exceeds the available balance (clamps
  to 0). Called after `SelectStockTransfers` confirms how many units
  to transfer.

  **Persistence:** the reserve is in-memory only (v1). A server restart
  re-accumulates from ongoing potion decay; no bootstrapping is needed
  because alchemy shops decay slowly and continuously.

  **Thread safety:** all three functions acquire `enchantReserveMu`.

### Stock Transfer Allocator (chunk 5.4)

- **`SelectStockTransfers(shopInv *ShopInventory, pool map[int]int) map[int]int`**:
  Pure function. Given a shop's current inventory and a pool of
  available item IDs → quantities, returns a transfer plan (item ID →
  qty to move) that fills the shop's biggest stock gaps first.

  Algorithm: compute `gap = MaxStock - Current` for every stocked item
  that appears in the pool; sort gaps descending (ties broken by item
  ID); greedily assign from pool until each item's gap or pool supply
  is exhausted. Only items the shop already carries (existing
  `StockEntry`) are eligible — the allocator never introduces new
  item IDs.

  **Shared by two call sites:**
  1. `internal/forager/chest_backfill.go` — `BackfillVendorFromChests`
     passes a pool built from all forager chest contents.
  2. `internal/hooks/MobIdle_HandleIdleMobs.go` — the enchanter draw
     path passes `ReservePool()` as the pool and calls `DrainReserve`
     for each entry in the result.

  Lifted from the forager-internal `selectBackfillTransfers` so both
  paths share one tested, deterministic allocator.

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

### Enchant Reserve
- **`enchantReserve map[int]int`** — in-memory map of item ID → qty
  for enchanting mats waiting to be drawn by enchanters.
- **`enchantReserveMu sync.Mutex`** — guards `enchantReserve`.
  Both fields are package-private; callers use the `AddToReserve` /
  `ReservePool` / `DrainReserve` API.
- **`ResetEnchantReserveForTest()`** — test helper; clears the reserve
  between test cases so state doesn't leak across sub-tests.

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
  guards, and the `[]DecayedUnit` return slice (content + length).
- **enchant_reserve_test.go**: Covers `AddToReserve` accumulation,
  `DrainReserve` partial and over-drain, and zero-ID/zero-qty no-op guards.
  Each test calls `ResetEnchantReserveForTest()` in setup to isolate state.
- **stock_transfers_test.go**: Covers the neediest-gap-first ordering,
  pool-cap clamping, items-not-in-shop exclusion, and empty-pool / empty-shop
  edge cases.
- **pricing_test.go**, **buyrules_test.go**, **restock_cadence_test.go**,
  **effective_max_stock_test.go**: Unit coverage for each sub-system.
- `TickOverstockDecayWith` injects `isComponent`, `decayRounds`, and
  `decayQty` so tests don't need loaded item specs or a live config stack.
