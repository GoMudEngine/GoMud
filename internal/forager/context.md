# Forager Package Context

## Overview

The `internal/forager` package implements the forager NPC state machine,
its territory/profile registry, yield-table data, throughput tracking, and
the two supply-chain entry points: `SellToVendor` (live delivery) and
`BackfillVendorFromChests` (overnight drain of lockbox overflow into vendor
gaps). The behavior-tree action `forager_step`
(`internal/behaviortree/actions_forager.go`) calls into this package each
idle tick to advance the state machine; the forager package itself contains
only pure state logic and I/O.

## Key Components

### Core Files
- **state.go**: `ForagerState` enum (7 states), `AdvanceState`, `ParseState`,
  `Name`.
- **territory.go**: `ForagerProfile` struct; `profiles` map keyed by mob ID;
  `ForagerKind` enum (Marsh/Steppe/Fernway).
- **forage_core.go**: `ForageDifficulty` + `ForageYields` + `NightForageYields`
  tables; `ForageAttempt`; pure roll logic shared by player and NPC forage
  commands.
- **vendor_sell.go**: `SellToVendor` — live delivery path.
- **chest_backfill.go**: `BackfillVendorFromChests` + `selectBackfillTransfers`
  (pure) + `chestPoolFromRooms` / `chestPoolAll` / `chestPoolForZone` —
  lockbox-to-vendor supply drain (chunk 5.4). Backfill aggregates across ALL
  forager chests globally (not per-zone).
- **chest_index.go**: `RegisterChestRoom` / `ChestRoomsForZone` /
  `ChestRoomsAll` — the zone-to-lockbox-room index (chunk 5.4).
- **throughput.go**: Per-forager delivery metrics (rarity tier counts, lbs).
- **arrival_listener.go**, **completion_listener.go**: World-event hooks that
  advance state on arrival / patrol completion.

## Key Functions

### State Machine
- **`AdvanceState(cur ForagerState) ForagerState`**: Returns the next linear
  state, wrapping `Recalling → Resting`.
- **`ParseState(name string) (ForagerState, bool)`**: Reverses `Name()`.
  Returns `(StateResting, false)` on unknown input; callers treat `!ok` as
  "no state set" and default to `StateResting`.

### Supply Chain — Live Delivery
- **`SellToVendor(roomId int, p *ForagerProfile, mob *mobs.Mob)`**: Walks all
  shopkeeper mobs in `roomId`. For each vendor with a `ShopInventory`, scans
  the forager's satchel (mob.Character.Items) and transfers items whose
  `economy.BucketFor(itemId)` is in `p.Buckets` and whose stock entry is
  below MaxStock. The transfer is **free — no gold changes hands** (the
  forager is a supply-side NPC, not a buyer). Saves the shop and throughput
  counters on any mutation.

### Supply Chain — Chest Backfill (chunk 5.4)
- **`BackfillVendorFromChests(vendorMob *mobs.Mob, shopInv *shops.ShopInventory)`**:
  Top-off path called during shop restock ticks. Aggregates item counts from
  ALL forager lockboxes across all zones (global aggregation — not per-zone),
  selects transfers targeting the neediest stock gaps (widest
  `MaxStock - Current` first), removes items from lockbox containers, and adds
  them to `shopInv.Current`. Also **free — no gold**. Saves the shop on any
  mutation.
- **`selectBackfillTransfers(si, pool) map[int]int`**: Pure helper. Given a
  vendor's `ShopInventory` and an item-count pool, returns a per-item transfer
  plan that fills gaps largest-first without exceeding pool availability. Pure
  function; injected into tests via `loadRoomFn` seam.
- **`chestPoolFromRooms(chestRooms []int) (pool map[int]int, nonEmpty []int)`**:
  Shared helper that aggregates item counts from lockbox containers across the
  given room IDs. Uses `loadRoomFn` seam (default `rooms.LoadRoom`).
- **`chestPoolAll() (map[int]int, []int)`**: Global pool — aggregates across
  every registered chest room in all zones via `ChestRoomsAll()`. Used by
  `BackfillVendorFromChests`.
- **`chestPoolForZone(zone string) (pool map[int]int, chestRooms []int)`**:
  Thin wrapper around `chestPoolFromRooms(ChestRoomsForZone(zone))`. Kept for
  backward compatibility with existing tests.

### Chest Index (chunk 5.4)
- **`RegisterChestRoom(zone string, chestRoom int)`**: Records that `zone` has
  a forager lockbox in `chestRoom`. Called by `tickForagerStoring`
  (`actions_forager.go`) the first time a forager deposits into its lockbox.
  Idempotent. No-op for zero values.
- **`ChestRoomsForZone(zone string) []int`**: Returns registered chest-room
  IDs for the zone in stable (sorted) order.
- **`ChestRoomsAll() []int`**: Returns every registered chest-room ID across
  all zones, stable-sorted and deduped. Used by `chestPoolAll()` for global
  backfill aggregation.

  **Chest index invariant:** The `zone → chest-rooms` index
  (`chest_index.go`) is the single source of truth for backfill lookup. It
  is self-populated from `mob.StorageChestRoom` at `tickForagerStoring` —
  it is NOT duplicated into the static `profiles` registry or any other
  structure. Chest rooms are fixed for a server's lifetime (the index only
  grows). `TestForagerStoring_RegistersChestRoom` guards the write side;
  `TestChestPoolForZone_AggregatesViaIndex` and `TestBackfill_GlobalCrossZone`
  guard the read side. Anyone moving or removing the `RegisterChestRoom` call
  in `tickForagerStoring` or the `ChestRoomsAll` lookup in `chestPoolAll`
  must keep all three tests green so the map cannot silently drift from the
  YAML-authored chest rooms.

## Global State

### Chest Index
- **`chestIndex`**: `map[string]map[int]bool` — zone → set of chest room IDs.
  Protected by `chestIndexMu` (`sync.RWMutex`). Populated at runtime from
  `tickForagerStoring`; never pre-seeded from YAML so there is no divergence
  risk between the registry and the actual placement.

### Profiles Registry
- **`profiles`**: `map[int]*ForagerProfile` — static, authored at compile
  time. Keyed by mob ID. Does not contain chest room IDs; those live in the
  chest index.

## Data Structure Design

### ForagerState Cycle
```
Resting → TravelingToTerritory → Foraging → TravelingToDropoff
    → Delivering → Storing → Recalling → (back to Resting)
```
`Storing` is skipped for foragers without a `storage_chest_room`; those go
directly `Delivering → Recalling`.

### ForagerProfile
```go
type ForagerProfile struct {
    Kind             ForagerKind
    MobId            int
    Name             string
    SanctuaryRoom    int
    TerritoryRooms   []int
    PreyWhitelist    []int
    VendorRooms      []int
    MeetingRoom      int
    Buckets          []string
    DeliveryPatrolId string
}
```

## Integration Notes

### Back-Pressure (chunk 5.4)
When a forager's lockbox is at or above `ChestBackpressureResumePct × ForagerLockboxCapacity`
items, `tickForagerResting` returns `Failure` — the forager stays resting and
does not start a new gather cycle. The backfill pass drains the lockbox over
time; once fill drops to ≤ `ChestBackpressureResumePct`, the forager resumes.

- `chestFillRatio(mob)` measures fill as
  `len(lockbox.Items) / ForagerLockboxCapacity`.
- `ForagerLockboxCapacity` (default 500) is the **same cap** that
  `dumpSatchelToLockbox` uses. The carry-ratio backstop (`ForagerRestCarryThreshold`)
  is a separate, secondary gate that fires only when the lockbox itself was full
  and surplus remains in the satchel.
- Config knobs: `ForagerLockboxCapacity` (default 500),
  `ChestBackpressureResumePct` (default 0.9).

### Callers
- **`internal/behaviortree/actions_forager.go`**: `forager_step` btree
  action drives all state transitions; calls `SellToVendor` during
  `StateDelivering`, calls `RegisterChestRoom` + lockbox deposit during
  `StateStoring`.
- **`internal/hooks/`**: Shop restock hook calls `BackfillVendorFromChests`
  during each restock tick.
- **`internal/actions/forage.go`**: `Forage` action uses `forage_core.go`
  yield/difficulty tables.

### Dependencies
- **`internal/shops`**: `GetShopInventory`, `SaveShop`, `StockEntry`.
- **`internal/mobs`**, **`internal/rooms`**: Instance and room resolution.
- **`internal/economy`**: `BucketFor` for supply-bucket filtering.
- **`internal/configs`**: `ForagerLockboxCapacity`, `ChestBackpressureResumePct`.

## Testing Notes

- **chest_index_test.go**: Covers `RegisterChestRoom` idempotency and
  `ChestRoomsForZone` ordering. `TestForagerStoring_RegistersChestRoom`
  (write-side invariant guard) and `TestChestPoolForZone_AggregatesViaIndex`
  (read-side invariant guard) are the two tests that must remain green to
  protect the chest index invariant.
- **chest_backfill_test.go**: Covers `selectBackfillTransfers` gap-sort and
  pool-depletion logic; `chestPoolForZone` via injected `loadRoomFn`.
- **forage_core_test.go**, **state_test.go**, **territory_test.go**,
  **throughput_test.go**: Unit coverage for the other sub-systems.
- `loadRoomFn` is a package-level seam (`var loadRoomFn = rooms.LoadRoom`)
  that tests override to inject fake rooms without touching the real room
  loader.
