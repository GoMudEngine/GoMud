# General-Store Restock Fix (caravan-zone baseline supply)

**Status:** Design approved — pending spec review → implementation plan
**Type:** Living-economy followup (the open third of `project_store_restock_considered_fix` — cooking + enchanting halves already shipped 2026-06-02)
**Date:** 2026-06-05

## Problem

General stores in **caravan-served zones** never replenish their common basics, so
stock stays depleted (the canonical victim is `storekeeper_wulf` 341 in Stillwater;
several of his items already sit at quantity 0).

Root cause (same class as the already-fixed enchanter):
- `MobIdle_HandleIdleMobs.go` (~line 65) **skips the per-mob shop-restock tick
  entirely for any mob in a caravan-served zone** (`IsCaravanServedZone` →
  Stillwater, Thornwall City). Those vendors are meant to be supplied by the
  caravan visiting.
- The crafter tick (`TickMobCraft`) gives crafters in caravan zones a baseline
  self-refill of common tiers via `shopInv.RestockBaselineTiers()` (tops up
  rarity-tier 50/40 items with `RestockQty > 0`, layered under caravan delivery of
  rarer goods).
- A general store is a **non-crafter**, so it runs *neither* the per-mob cart tick
  *nor* the crafter baseline refill. The caravan only delivers rarer/bucketed
  goods, so the store's common basics (rope, lanterns, salt, flasks, planks) never
  come back.

The stock *data* is already fine: `mobs/crafter.go` seeds each legacy
`Character.Shop` item into the ShopInventory template with `RestockQty=5,
MaxStock=20`. The missing piece is the **tick** that applies that baseline refill
for non-crafter vendors in caravan zones.

## Design

Give non-crafter vendors in caravan-served zones the same baseline-tier self-refill
crafters already get there — common basics refill locally, rare goods stay
caravan-gated.

### New: `mobs.TickMobShopBaselineRestock(mob *Mob) bool`

In `internal/mobs/crafter.go`, alongside `TickMobShopRestock`. Cadence-gated baseline
refill for a non-crafter vendor:
- No-op if `mob.Crafter` (crafters already do this via `TickMobCraft`).
- Fetch `shops.GetShopInventory(mob.Zone, int(mob.MobId), mob.HomeRoomId)`; no-op if nil.
- Gate on the **tier-50 cadence** via the existing `LastRestockByTier` map +
  `shops.RestockCadenceHours(b, 50)` × `roundsPerHour` (mirror `TickMobShopRestock`'s
  cadence math; key the gate on tier 50 since `RestockBaselineTiers` covers 50+40
  together). Initialize `LastRestockByTier` if nil.
- When the cadence elapses, call `shopInv.RestockBaselineTiers()` and return whether
  it added stock; stamp `LastRestockByTier[50] = roundCount`.

This refills only tier-50/40 `RestockQty>0` entries — exactly the common basics —
and never touches tier 30/20/10 (those remain caravan-gated).

### Wire it in `MobIdle_HandleIdleMobs.go`

In the non-crafter restock branch (~lines 61-79), replace the bare caravan-zone skip
so that:
- **Non-caravan zone:** unchanged — `mobs.TickMobShopRestock(mob)` (full per-tier).
- **Caravan-served zone:** instead of skipping, call
  `mobs.TickMobShopBaselineRestock(mob)` for non-crafter shop mobs.
Both paths emit the existing "A supply cart pulls up outside…" room message on a
successful restock (reuse the current message block).

Sketch:
```go
restocked := false
if !configs.GetBalanceConfig().IsCaravanServedZone(mob.Zone) {
    restocked = mobs.TickMobShopRestock(mob)
} else {
    restocked = mobs.TickMobShopBaselineRestock(mob)
}
if restocked { /* existing supply-cart room message */ }
```
(`TickMobShopBaselineRestock` itself no-ops for crafters, so guarding on
non-crafter isn't strictly required, but the call only matters for mobs with a
shop — keep the existing has-shop pathing.)

### Scope

All **non-crafter shop mobs in caravan-served zones** (Stillwater, Thornwall City),
not just Wulf — it's the general fix and only ever tops up common basics they
already stock. Crafters and the enchanter (reserve-fed) are unaffected.

## Caveat / verification step

`RestockBaselineTiers` only catches items at rarity-tier **50 or 40**. Per the
rarity-tier-tagging audit, untagged items fall back to tier 50, so most general
basics qualify. **Verify Wulf's stock**: any general basic deliberately tagged
rarer (30/20/10) would stay caravan-gated; re-tag to 40/50 if it's meant to be a
common self-restocking good. (Audit-only; likely no data change needed.)

## Testing

Go unit tests (mobs/shops are unit-tested):
- `TickMobShopBaselineRestock`: with a non-crafter mob + a ShopInventory holding a
  tier-50 `RestockQty>0` depleted entry and a tier-30 entry — after the cadence
  elapses, the tier-50 entry's `Current` rises by `RestockQty` (capped at MaxStock)
  and the tier-30 entry is untouched. Before the cadence elapses (or on the first
  call that just stamps `LastRestockByTier`), nothing is added. Crafter mob → no-op.
- (Reuse existing shop-inventory fixtures / `RestockBaselineTiers` is already
  tested; this adds the cadence-gated non-crafter entry point + the MobIdle wiring.)
- Boot smoke (deferrable): boot, connect, deplete Wulf, idle past the tier-50
  cadence, confirm his common basics climb (supply-cart message fires) while any
  rare goods stay flat.

## Files touched (anticipated)

- `internal/mobs/crafter.go` — add `TickMobShopBaselineRestock`.
- `internal/hooks/MobIdle_HandleIdleMobs.go` — caravan-zone branch calls it.
- `internal/mobs/crafter_test.go` (or a new test file) — unit tests.
- `internal/mobs/context.md` / `internal/shops/context.md` — note the non-crafter
  caravan-zone baseline refill.
- Memory: update `project_store_restock_considered_fix` (general-store half done).

## Out of scope

- The crafter / caravan-delivery path (unchanged); the enchanting reserve (separate).
- Caravan *content* gaps (e.g. Fernway `deliveries_by_tier` empty — separate caravan-content followup).
- Sanctum general store (`merchant_adela` 63) — being replaced by the newbie-area rework.
- Re-tiering items wholesale (only re-tag a Wulf basic if the verification step finds one mis-tagged rare).
