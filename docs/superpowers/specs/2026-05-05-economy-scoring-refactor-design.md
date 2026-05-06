# Economy Health Scoring Refactor — Design

**Date:** 2026-05-05
**Status:** Approved, ready for plan
**Scope:** Replace the single weighted-fill-percentage shop score with a
five-axis scoring model (stock, throughput, input rate, logistics
health, shop gold). Add per-rarity-tier restock cadence. Tighten
caravan/forager failure-mode signals so stuck and despawned entities
visibly degrade the dashboard.

## Context

The economy dashboard's stated purpose is to support player crafting
grinding — telling the operator "can a player walk into shop X and
find materials Y to craft right now and over the next several
sessions?" The current scoring (`internal/economy/health/scoring.go`)
is a single per-shop weighted-fill-percentage that:

- Reads a freshly-restocked shop with zero economic activity as 100
  (perfect health).
- Reads a briskly-selling shop as low the moment customers buy.
- Has no throughput dimension, no shop-gold liquidity dimension, no
  input-rate dimension.
- Smooths over caravan/forager failure modes (stuck Kessa, despawned
  Halix) with a flat -30 stuck penalty out of 100, leaving real
  failures reading 70/100 instead of 0.
- Mixes a snapshot metric (shops) with rate metrics (caravans,
  foragers) inside a single overall score, so two very different
  states produce the same number.

The current restock cadence is a single global ticker (~2h),
indifferent to rarity — common iron-ore restocks at the same rate
as a tier-10 rare. This makes the stated "common materials are
always available for crafting grinding" guarantee impossible to
deliver.

## Goals

- Five independent score axes that each answer a different question
  about the economy.
- An overall score that blends the player-facing axes.
- Per-rarity-tier restock cadence so common mats are physically
  available on the cadence the player loop expects.
- Sharp logistics signals: a stuck or despawned caravan/forager
  reads near zero, not "moderate."
- All thresholds and weights configurable.
- Lazy migration; no flag day; existing shop persistence files
  load with zero-default counters and accumulate signal as snapshots
  roll forward.

## Non-goals

- Real-time event-log scoring (no append-only transaction journal).
- Per-player-resolution economy modeling.
- Admin UI for tuning thresholds — config knobs only.
- Score-formula changes for the existing `PerCaravanScore` /
  `PerForagerScore` (those become the new logistics health).

## The five score axes

### 1. Stock score (existing formula, renamed)

What the current `PerShopScore` computes today: weighted fill
percentage per shop. Sum over stock entries of
`RestockQty × min(Current/Max, 1)` divided by total `RestockQty`,
scaled to 0-100.

Answers: *"Is there material on the shelf right now?"*

Renamed in the dashboard from "Score" to "Stock Score." Aggregation
unchanged — per-shop, per-discipline mean.

### 2. Throughput score (new)

A Time-to-Refill (TtR) measurement, weighted heavily toward common
items because commons are what crafting grinders rely on.

Answers: *"How long does a player wait between depletion and
restoration of the materials they want?"*

**Per-item event tracking on `ShopInventory`:**

```go
type StockEvent struct {
    DepletedRound uint64  // round Current dropped to 0
    RefilledRound uint64  // round Current returned > 0 (0 if ongoing)
}

// New fields on ShopInventory:
StockEvents      map[int][]StockEvent  // itemId -> rolling 7-game-day window
CurrentDepletion map[int]uint64        // itemId -> DepletedRound (0 = not depleted)
```

Hook in existing `RemoveStock` and `AddStock`:
- `RemoveStock`: if Current hits 0 and `CurrentDepletion[itemId]` is
  unset, set it to `currentRound`.
- `AddStock`: if `CurrentDepletion[itemId]` is set, push a completed
  `{depleted, refilled}` event into `StockEvents[itemId]`, clear
  `CurrentDepletion[itemId]`.

**Window:** all completed events with `RefilledRound` within the
last 7 game-days. No event-count cap. Currently-depleted items
contribute their ongoing duration as a partial event regardless of
window.

**Per-item TtR target by rarity:**

| Rarity tier | Target TtR |
|-------------|------------|
| 50 (common) | 3 game-hours |
| 40          | 6 game-hours |
| 30          | 18 game-hours |
| 20          | 3 game-days |
| 10 (rare)   | 7 game-days |

Targets are ~3× the new restock ticker for that tier — a shop
score only drops when ~3 cycles in a row missed.

**Per-item score:**
```
mean_ttr      = mean over completed events in window
                + (currentRound - currentDepletion) if still depleted
item_score    = clamp(1 - mean_ttr / target_for_rarity, 0, 1) * 100
weight        = rarity_tier^2  // tier-50 weighted 25× tier-10
```

**Per-shop throughput score:**
```
throughput_score = sum(item_score × weight) / sum(weight)
```

Items with no events and not currently depleted contribute weight at
score 100 (always-available is the best case).

Aggregation: per-shop, per-discipline mean.

### 3. Input rate score (new)

Items entering the supply per game-day from foragers and restock
cycles, weighted toward commons (linear, not quadratic — commons
naturally dominate volume already).

Answers: *"Is new material flowing into the supply at the rate the
player loop expects?"*

**Counters needed:**
- `RestockCount int` on `ShopInventory` (cumulative, increments each
  fired restock cycle per item — sum across items so a shop with
  20 stock lines that all restocked = 20).
- Foragers already track `DeliveriesByTier` per snapshot.

**Score formula (per zone):**
```
input_per_day = (forager_deliveries_in_window + restocks_in_window)
                weighted by rarity_tier (linear)
                normalized to per-game-day rate

zone_target = sum over active shops in zone of:
              (expected_restocks_per_day_per_item × items_per_shop
               × rarity_tier)
            + sum over active foragers in zone of:
              (expected_deliveries_per_day × forager_capacity
               × tier_distribution_weight)

where:
  expected_restocks_per_day_per_item = 24 / cadence_hours_for_tier
  expected_deliveries_per_day        = 3   (8h forager cycle)
  tier_distribution_weight           = mean rarity_tier of items
                                       a forager typically gathers
                                       (defaults to 30 if unknown —
                                       refined as forager profiles
                                       record observed-tier mix)

input_score = clamp(input_per_day / zone_target, 0, 1) * 100
```

`zone_target` is auto-derived so adding/removing shops scales it
without manual tuning. `expected_restocks_per_day_per_item` comes
from the per-tier ticker (24h / cadence_hours).

Aggregation: per-zone score, plus a global mean across zones for
the overall card.

### 4. Logistics health (new, standalone — not in overall)

Composite of cycle rate and cargo flow, with hard multipliers for
stuck and despawned states. Applies symmetrically to caravans and
foragers.

Answers: *"Are the actual logistics entities (Halix, Ketil, Kessa)
functioning?"*

**Per-entity formula:**
```
cycle_score = clamp(100 × cycles_in_window / expected_cycles, 0, 100)
cargo_score = clamp(100 × cargo_delivered_in_window / target_cargo, 0, 100)
base        = 0.5 × cycle_score + 0.5 × cargo_score

if despawned:           score = 0
elif stuck > threshold: score = base × 0.4
else:                   score = base
```

**Targets:**
- Caravan: 1 cycle/game-day, target cargo = `cargo_capacity × cycles`
  (i.e., a caravan that completes a cycle should have moved roughly
  one full load).
- Forager: 3 cycles/game-day (8h cycle), target cargo computed
  similarly from forager `cargo_capacity`.
- Stuck threshold: `LogisticsStuckRounds` config knob, default
  3000 rounds (down from today's 5000 — too lenient).

**Counters needed:**
- `LbsDelivered uint64` on `Caravan` and `Forager` (cumulative
  pounds delivered to a shop).

**Despawn detection:** a forager is "despawned" when its profile
exists but the snapshot has no live row for it (current snapshot
already has placeholder rows for inactive foragers via
`(not active)` state — extend to mark this as a despawn signal).

**Display:** standalone panels for caravans and foragers showing
the score, current state, time-in-state, cargo, and the multiplier
that's applying (e.g., "stuck — 0.4× multiplier active").

**Not folded into overall** — logistics failures already propagate
downstream (stuck Kessa → low input rate → low overall). Folding it
in directly would double-count.

### 5. Shop gold score (new)

Per-shop liquidity. A merchant with 0 gold cannot buy from players,
which breaks half the shop loop and is invisible in today's
dashboard.

Answers: *"Can the merchant actually buy what players are selling?"*

**Per-shop formula:**
```
ratio       = clamp(gold / starting_gold, 0, 1.5)
gold_score  = (ratio / 1.5) * 100
```

A shop at 100% of starting gold scores ~67; at 150%+ scores 100;
at 50% scores 33; at 0 scores 0. The `1.5` ceiling rewards merchants
that have accumulated player sales without giving infinite headroom.

Aggregation: per-shop, per-discipline mean. Surfaces in the existing
per-shop table as a new column.

### 6. Overall score

```
overall = 0.40 × stock_mean
        + 0.30 × input_mean
        + 0.20 × throughput_mean
        + 0.10 × gold_mean
```

Renormalized over components with data (preserves the bootstrap-
window behavior the existing overall already has). Logistics is
not in this blend.

Card layout (replaces today's 5 cards):
| Overall | Stock | Input | Throughput | Shop Gold |

Logistics gets its own panel below caravans/foragers, not a card.

## Restock ticker — per-rarity-tier cadence

Replace the single global restock ticker with a per-tier cadence.

| Tier | Common→Rare | Cadence |
|------|-------------|---------|
| 50   | common      | 1 game-hour |
| 40   |             | 2 game-hours |
| 30   |             | 6 game-hours |
| 20   |             | 24 game-hours |
| 10   | rare        | 5 game-days |

Tier 10 rares aren't currently restocked via ticker — primary input
is foragers + player sales. The 5-day backstop ensures rares can
never be permanently zero on a shop.

**Implementation:** the existing restock loop fires per item, not
per shop. Replace the single cadence check with a per-item check
keyed by the item's `rarity_tier`. Config knobs:
`RestockCadenceTier{50,40,30,20,10}Hours` (with the tier-10 value
expressed in game-days internally for readability).

**Scope note.** This is a player-facing behavior change (commons
will refill faster than today). It's bundled in this spec because
TtR scoring is meaningless if the engine can't physically meet the
targets — the two are tightly coupled.

## Persistence schema changes

### ShopInventory (new fields)

```go
SalesCount       int                       // cumulative items sold to players
BuysCount        int                       // cumulative items bought from players
RestockCount     int                       // cumulative items added by restock cycles
StockEvents      map[int][]StockEvent      // itemId -> rolling 7-day events
CurrentDepletion map[int]uint64            // itemId -> DepletedRound (0 = not depleted)
```

All zero/empty defaults. Existing shop YAMLs load cleanly with no
migration step beyond reading the new fields.

### Caravan / Forager (new field)

```go
LbsDelivered uint64  // cumulative pounds delivered to shops over lifetime
```

Zero default. Snapshot captures it; logistics score uses
window deltas.

## Snapshot integration

Snapshots already record `Shops`, `Caravans`, `Foragers`. Extend
those records with the new counter values. Delta computation
(`internal/economy/health/delta.go`) gains:

- `ShopDelta.SalesDelta`, `BuysDelta`, `RestocksDelta` for the new
  counters.
- New `StockEventsClosedInWindow []StockEvent` — events whose
  `RefilledRound` falls within the delta window. Drives the per-
  window TtR cells in the throughput table.
- `CaravanDelta.LbsDeliveredDelta`, `ForagerDelta.LbsDeliveredDelta`.

## Dashboard structure

### Top row (5 cards, replaces today's 5)
- Overall — score
- Stock — mean shop stock score
- Input — global input rate
- Throughput — mean shop throughput score
- Shop Gold — mean shop gold score

### Stock table (existing, relabeled)
- Header rename: "Score" → "Stock Score"
- Per-shop row gains a new column: "Shop Gold" (the gold score).
- Drop the per-window `Δ stock pp` cells — they're noise.

### Throughput table (new)
| Shop | Throughput Score | TtR (commons) | TtR (rares) | Currently depleted | 1h | 6h | 1d | 3d | 1w |

Per-window cells = median TtR for events that completed in that
window. Currently-depleted = count of items at 0 right now (red
when > 0 for a tier-50 item).

### Input rate table (new)
Per-zone: total items/day in, breakdown by source (forager / restock),
breakdown by tier. Score column = input rate score. Compares to
zone target.

### Logistics panel (new)
Per-caravan + per-forager: name, score, state, time-in-state, cargo,
multiplier active (none / stuck / despawned).

## Configuration knobs

New entries in `Balance` config:

```yaml
# Restock cadence per rarity tier (decreasing rarity → longer cadence)
RestockCadenceTier50Hours: 1
RestockCadenceTier40Hours: 2
RestockCadenceTier30Hours: 6
RestockCadenceTier20Hours: 24
RestockCadenceTier10Days:  5

# TtR targets per rarity tier (game-time)
TtRTargetTier50Hours:      3
TtRTargetTier40Hours:      6
TtRTargetTier30Hours:      18
TtRTargetTier20Days:       3
TtRTargetTier10Days:       7

# Throughput rolling window
TtRWindowGameDays:         7

# Logistics
LogisticsStuckRounds:      3000  # down from current 5000
LogisticsStuckMultiplier:  0.4

# Score blending
ScoreWeightStock:          0.40
ScoreWeightInput:          0.30
ScoreWeightThroughput:     0.20
ScoreWeightShopGold:       0.10
```

Existing knobs (`ShopAbundanceThreshold`, etc.) remain unchanged.

## Phased delivery

The implementation plan should phase delivery:

1. **Restock ticker change.** Per-tier cadence. Config + engine
   change. Independent ship; observable in-game without dashboard
   work.
2. **Persistent counters.** Add `SalesCount`/`BuysCount`/
   `RestockCount`/`StockEvents`/`CurrentDepletion` to ShopInventory;
   wire `RemoveStock`/`AddStock` hooks. Add `LbsDelivered` to
   caravan/forager. No score consumers yet — counters just start
   accumulating.
3. **Scoring functions.** Implement throughput, input rate,
   logistics, shop-gold. Update overall blend. Old `PerShopScore`
   stays as `StockScore` for backwards compatibility.
4. **Dashboard updates.** New cards, new tables, relabel existing.
5. **Logistics tightening sanity.** Confirm Halix/Kessa/Tova types
   of failures now register as near-zero scores; verify
   `LogisticsStuckRounds` default doesn't false-positive on healthy
   slow routes.

## Migration

- ShopInventory new fields: zero/empty defaults; old YAMLs load
  with empty maps.
- Caravan/forager new field: zero default.
- Snapshot YAMLs are append-only; old snapshots load with the new
  fields absent (zero on read).
- The score functions handle empty-window cases as "no recent
  stress" → score 100 for that axis (matches the existing bootstrap
  behavior of `PerCaravanScore` returning `(0, false)` when history
  < 24 samples; new code returns 100 when window has zero events
  *and* nothing currently depleted).

No flag day. After 7 game-days of running on the new code, the
throughput window is fully populated.

## Test plan

### Unit
- `TtRBucket`: events that completed in window, currently-depleted
  partials, weight aggregation.
- Throughput score: items-with-no-history → 100; mostly-depleted
  shop → low; per-tier weight curve.
- Input rate score: zone target derivation; per-tier weighting;
  bootstrap (no history) → 100.
- Logistics: cycle/cargo/composite math; stuck multiplier; despawn
  detection.
- Shop gold: linear ramp from 0 → 1.5× starting gold.
- Overall: weighted blend; renormalization.

### Integration
- Buy/sell increment `SalesCount`/`BuysCount`.
- Restock cycle increments `RestockCount` and pushes
  `StockEvent`s when items go 0 → positive.
- Per-tier ticker fires at the right cadence (use a mock clock).
- Snapshot delta carries the new fields end-to-end.
- Dashboard renders all five cards and the three new tables with
  fixture data.

### Manual smoke
- Stand up a fresh server, watch a tier-50 item go 0 → 1 → 0; verify
  a `StockEvent` is recorded.
- Force a forager despawn; verify logistics score drops to 0 within
  a snapshot.
- Force a caravan stuck > 3000 rounds; verify multiplier kicks in.

## Files touched

### Engine
- `internal/economy/health/scoring.go` — main scoring rewrite
- `internal/economy/health/scoring_test.go` — extensive new tests
- `internal/economy/health/snapshot.go` — capture new counters
- `internal/economy/health/delta.go` — new delta fields
- `internal/economy/health/delta_test.go`
- `internal/shops/inventory.go` (or sibling) — new fields on
  ShopInventory; wire RemoveStock/AddStock hooks
- `internal/shops/restock.go` (or sibling) — per-tier cadence
- `internal/caravan/*.go` — `LbsDelivered` counter on delivery
- `internal/forager/*.go` — `LbsDelivered` counter on delivery
- `internal/configs/balance.go` — new config fields

### Dashboard
- `_datafiles/html/admin/economy/index.html` — major layout work:
  new cards, new tables, relabel existing, drop noisy delta cells

### Documentation
- `CLAUDE.md` — update "Shop Persistence (Living Economy)" section
  to mention per-tier ticker
- `_datafiles/world/dogmud/templates/help/bank.template` — no
  change needed (this spec doesn't touch banking)

## Out of scope

- Real-time transaction event log.
- Per-player or per-character usage histograms.
- Admin UI knob editor.
- Caravan route customization, forager territory tuning.
- Score-formula changes for non-economy systems.
- Renaming `PerShopScore` symbol in Go (`StockScore` is the
  dashboard label; the Go function name can stay or be updated as
  the implementer prefers).
