# New Plymouth Supply — Pricing Fix + Wiring Recipe (Design Spec)

*Date: 2026-06-20 · Re-scoped from "chunk 3.5" engine prerequisite.*

> **What this is.** The New Plymouth master plan
> (`docs/superpowers/specs/2026-06-20-new-plymouth-design.md` §6) called for
> building a "scoped chunk 3.5" supply/restock engine before the city. A
> read-only code investigation found the supply loop **already exists and runs
> in production** (the Thornwall↔Stillwater caravan runner circuit), so the
> prerequisite collapses to **one standalone engine bug fix** plus a
> **documented wiring recipe** the Docks district build will execute.
>
> **This does NOT implement roadmap chunk 3.5 ("Maintenance routines" — a
> reusable activity library).** That chunk stays deferred; the investigation
> confirmed it isn't needed for the city's supply model. The roadmap's
> per-segment `activity: craft` gate + the existing caravan/forager supply
> machinery already cover it.

---

## 0. Background — what already exists (verified in code)

The "warehouse → mini-caravan → district vendor" model is the **live
Thornwall/Stillwater caravan runner circuit**:

- **Delivery into vendor stock works.** A runner loads cargo at a depot, runs a
  `oneshot` patrol (chunk 3.8), and at each waypoint tagged
  `arrival_event: "caravan_vendor"`, `CaravanArrivalListener`
  (`internal/caravan/arrival_listener.go:30`) → `VisitVendorsInRoom`
  (`internal/caravan/visit.go:42`) physically transfers items from the runner's
  inventory into the destination vendor's `ShopInventory` (`entry.Current++`).
- **Rare-tier supply is already delivery-gated.** `CaravanServedZones`
  (`Balance.CaravanServedZones []string`, `_datafiles/config.yaml:951`) makes
  `TickMobCraft` (`internal/mobs/crafter.go:294`) skip ticker auto-refill for
  tiers 30/20/10 — those flow only from the runner.
- **Global vendor backfill exists** (chunk 5.4): foragers deposit surplus in
  lockbox chests; `forager.BackfillVendorFromChests` pulls from a global pool
  via `shops.SelectStockTransfers` on restock ticks. The pool
  (`forager.RegisterChestRoom(zone, roomId)`) is generic — a Docks warehouse
  room could feed it.

**Conclusion:** no supply engine needs building. Making NP work this way is
config + ~6 lines of Go wiring + content authoring — and that wiring references
NP-specific patrol IDs / items that don't exist until the Docks build, so it
belongs **with** the Docks build (§2 is its recipe), not before it.

The **one** genuinely standalone, build-now engine defect is the pricing bug
(§1) — it affects crafted goods game-wide, not just NP.

---

## 1. The fix — decouple pricing baseline from ticker restock quantity

### The bug

`StockEntry.RestockQty` is overloaded. It means two different things:
1. **Ticker semantics:** "how many the auto-restock ticker adds per cadence."
   `RestockQty <= 0` = "don't auto-refill" (intentional for caravan-only and
   crafter-produced items).
2. **Pricing semantics:** the "normal stock level" denominator in the scarcity
   curve.

`EffectiveRestock` (`internal/actions/buy.go:499`) supplies the pricing
baseline, falling back to `MaxStock/2` when `RestockQty == 0`:

```go
func EffectiveRestock(entry *shops.StockEntry) int {
    if entry.RestockQty > 0 {
        return entry.RestockQty
    }
    norm := entry.MaxStock / 2   // ← too high for low-volume goods
    if norm < 1 {
        norm = 1
    }
    return norm
}
```

A brand-new crafted entry gets `RestockQty:0, MaxStock:20`
(`shops.AddStockAtRound`, `shopinventory.go:151`), so the pricing baseline is
**10**. Fed into `ScarcityMultiplier(current, baseline, cfg)`
(`internal/shops/pricing.go:52`; ceiling 5× / floor 0.25× / abundance at
ratio ≥ 3), a slowly-restocking crafter is priced **2.4–5×** until stock is
near full — and with `MaxStock:20` and baseline 10 you can never reach the
abundance floor (it needs 30 units). Result: crafted goods appear in `list`
but read as "not for sale." This is the user's longstanding complaint
(`project_crafted_items_buyability_investigation`).

### The fix (approach B)

Keep `RestockQty == 0`'s **ticker** meaning ("no auto-refill") untouched. Fix
only the **pricing baseline**: when `RestockQty == 0`, use a small, tunable
default instead of `MaxStock/2`.

**New config knob** — `Balance.DefaultPricingBaselineQty int`, **default 3**.
Add to `internal/configs/config.balance.shops.go` alongside the existing shop
pricing knobs (`ShopPriceFloor`, `ShopPriceCeiling`, `ShopAbundanceThreshold`,
`CrafterIngredientReservePct`), following that file's field + default pattern.

**Signature change (threaded, not a hidden config read)** — per the testability
preference (`project_evaluatebuyrules_testability`), pass the baseline in rather
than reading global config inside `EffectiveRestock`:

```go
// internal/actions/buy.go
func EffectiveRestock(entry *shops.StockEntry, defaultBaseline int) int {
    if entry.RestockQty > 0 {
        return entry.RestockQty
    }
    if defaultBaseline < 1 {
        defaultBaseline = 1
    }
    return defaultBaseline
}
```

**Both callers** pass `configs.GetBalanceConfig().DefaultPricingBaselineQty`:
- `tryPurchaseFromInventory` (`internal/actions/buy.go:513`)
- `buildShopStockFromInventory` (`internal/usercommands/list.go:99`)

(These are the *only* two callers — confirmed via codegraph. No restock-mechanic
code calls `EffectiveRestock`, so ticker behavior is provably untouched.)

### Effect (current `ScarcityMultiplier` curve, `MaxStock:20`)

| `Current` | baseline `MaxStock/2` = 10 (today) | baseline 3 (fixed) |
|---|---|---|
| 1 | ~4.7× | ~4.0× (last-unit scarcity premium — intended) |
| 5 | ~3.2× | **~1.2×** |
| 9 | ~2.5× | **0.25× (floor)** |
| 20 (full) | 0.78× | 0.25× (floor) |

A crafter accumulating a modest 5–9 units now sells near or below base; only
the last unit or two carry a scarcity premium. `DefaultPricingBaselineQty` is a
tunable knob if 3 proves too aggressive in playtest.

---

## 2. NP supply wiring recipe (reference — built during the Docks district build, NOT now)

This section is documentation the Docks build follows. It uses the existing
caravan model end-to-end; the only Go change is registering the NP circuit.

1. **Config** — add the NP zones to `CaravanServedZones` (`_datafiles/config.yaml`)
   so tiers 30/20/10 flow only from the runner, not the ticker. (Tier 50/40
   basics still trickle via the baseline restock.)
2. **Register the NP runner circuit** (~6 lines, the only Go change):
   - `internal/caravan/runner_completion_listener.go` — add
     `"np_docks_runner_circuit": {}` to `runnerCircuitPatrols`.
   - `internal/caravan/arrival_listener.go` `bucketsForRunnerPatrol` — add a
     case returning the NP delivery buckets (e.g. `[]string{"np_imported", "base"}`),
     pickup `[]string{}` (delivery-only from the Docks).
   - `internal/economy/buckets.go` — add NP imported items to the `itemBucket`
     map (sea salt, exotic cloth, spice, etc.), or route via existing
     `"base"`/`"overlap"` buckets.
3. **The warehouse = a Dock Master merchant (Option A — zero new abstraction).**
   Author a Dock Master shop NPC in the Docks zone, **not** in
   `CaravanServedZones`, so it self-restocks imported goods via the tier ticker.
   The runner circuit's pickup pass drains its overstock into the runner; the
   runner then delivers to district vendors. This is exactly the
   Thornwall→Stillwater pattern already operating.
4. **Runner circuit patrol YAML** — `_datafiles/world/dogmud/patrols/new_plymouth/`
   with `loop_shape: oneshot`, waypoints tagged `arrival_event: "caravan_vendor"`
   at each district vendor room, originating at the Docks depot.
5. **District vendor StockEntry declarations** — each district vendor must
   **pre-declare** every deliverable item as a `StockEntry` (`VisitVendorsInRoom`
   silently skips items with no existing entry). Set `Current: 0`,
   `MaxStock: <buffer>`, and `RestockQty: 0` — pricing is now sane at
   `RestockQty:0` thanks to §1, so authors no longer need to set a non-zero
   `RestockQty` (which would wrongly re-enable the ticker in non-caravan zones).

**Cross-reference:** record this recipe in the NP master plan §6 / the Docks
district spec when that build starts.

---

## 3. Testing (TDD)

Unit tests, no server boot required.

**`internal/actions/buy_test.go`** — `EffectiveRestock`:
- `RestockQty > 0` → returns `RestockQty` (unchanged behavior).
- `RestockQty == 0`, `defaultBaseline = 3` → returns 3 (NOT `MaxStock/2`).
- `RestockQty == 0`, `defaultBaseline = 0` → clamps to 1.

**`internal/shops/pricing_test.go`** — extend existing `ScarcityMultiplier`
coverage with the bug-repro table (assert the multiplier drops into a sane band
with baseline 3):
- `(current=5, restockQty=3)` → mult ≈ 1.2× (assert `< 1.5`).
- `(current=9, restockQty=3)` → floor (assert `== PriceFloor`).
- `(current=1, restockQty=3)` → high but bounded (assert `<= PriceCeiling`).
- Existing `TestScarcityMultiplier_*` tests must still pass unchanged.

**Config** — assert `DefaultPricingBaselineQty` defaults to 3 when absent
(mirror an existing `config.balance.misc.go` default test if one exists).

---

## 4. Scope / non-goals

**In:** the §1 pricing fix (config knob + `EffectiveRestock` signature + 2
callers + tests). The §2 recipe as **documentation only**.

**Out:**
- Roadmap chunk 3.5 ("Maintenance routines" activity library) — stays deferred.
- Any NP content (Dock Master, runner patrol, district vendors, buckets) — built
  during the Docks district build using §2.
- The `np_docks_runner_circuit` Go registration — built with the Docks content
  it references (inert until then).
- Pricing-curve redesign — we adjust the baseline, not the `ScarcityMultiplier`
  shape.

---

## 5. File map

| File | Change |
|---|---|
| `internal/configs/config.balance.shops.go` | Add `DefaultPricingBaselineQty int` field + default 3 (follow the file's existing shop-knob pattern) |
| `internal/actions/buy.go:499` | `EffectiveRestock` gains `defaultBaseline int` param; `RestockQty==0` returns it |
| `internal/actions/buy.go:513` | `tryPurchaseFromInventory` passes the config value |
| `internal/usercommands/list.go:99` | `buildShopStockFromInventory` passes the config value |
| `internal/actions/buy_test.go` | New `EffectiveRestock` tests |
| `internal/shops/pricing_test.go` | Extended scarcity-band tests |

*(Balance config lives under `internal/configs/` — `config.balance.shops.go`
for shop/pricing knobs; confirm the default-init pattern in that file before
editing.)*

---

## 6. Next step

writing-plans → a small TDD plan for §1 only. The §2 recipe is carried into the
Docks district build.
