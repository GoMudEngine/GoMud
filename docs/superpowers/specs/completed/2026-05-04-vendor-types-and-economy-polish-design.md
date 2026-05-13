# Vendor Types & Economy Polish — Design

**Date:** 2026-05-04
**Status:** approved, ready for plan
**Related:**
- `docs/superpowers/specs/completed/2026-04-04-living-economy-design.md`
- `docs/superpowers/specs/completed/2026-04-27-caravan-system-design.md`
- `docs/superpowers/specs/2026-04-29-stage-3-1-foragers-design.md`
- `docs/superpowers/specs/2026-04-30-stage-3-4-real-item-transfer-design.md`
- `docs/superpowers/plans/2026-05-01-economy-health-dashboard.md`

## Goals

Polish the forager + caravan economy so that:

1. Every vendor type has a clear, item-tag-driven inventory list, including
   raw mats and finished goods. Apothecary Ilsa (and her sibling specialists)
   buys all alchemy-tagged items, including ones not in her recipe list.
2. Specialist vendors no longer buy gear-upgrade items (they're non-combatants
   and never wear their purchases).
3. General stores fall back to "buy anything tagged that the vendor can still
   profitably resell."
4. Every shop is explicitly typed and audited; unused shopkeepers are stripped
   to flavor mobs / questgivers.
5. Tier-50 and Tier-40 mats refill at every shop on the existing crafter tick,
   in addition to forager + caravan deliveries — the "remote shop fills in
   too slowly" problem ends.
6. The `/admin/economy/` dashboard:
   - Shows every shopkeeper's name (template fallback when not spawned).
   - Replaces the gold-delta column with a stock-score-delta column.
   - Shows per-forager and per-caravan throughput (items delivered, by rarity
     tier) as a colored bar.
   - Distinguishes "forager despawned" from "forager idle / no state."
7. Foragers have a stuck-state watchdog so a wedged Halix self-recovers.

## Non-goals

- No changes to dynamic pricing math (`CalcBuyPrice` stays as-is).
- No changes to the forager state machine itself beyond watchdog + diagnostics.
- No changes to the caravan state machine beyond throughput counters.
- No changes to the buy/sell user commands beyond the rule swap inside
  `EvaluateBuyRules`.
- Stillwater bank — captured as a future-work memory; out of scope here.

## Architecture overview

```
┌────────────────┐    ┌──────────────────┐    ┌──────────────────┐
│  ItemSpec      │───►│ EvaluateBuyRules │◄───│  ShopInventory   │
│  vendor_       │    │  (rewritten)     │    │  +CraftSupport   │
│  categories[]  │    └──────────────────┘    └──────────────────┘
└────────────────┘             │
                               ▼
                       ┌──────────────────┐
                       │  Shop accepts /  │
                       │  rejects offer   │
                       └──────────────────┘

┌──────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│ TickMobCraft     │───►│ shopInv.Restock  │    │ Foragers /      │
│ (existing)       │    │ BaselineTiers()  │    │ Caravan layer   │
│                  │    │ (NEW: tier 50+40)│◄───│ rarer mats on   │
└──────────────────┘    └──────────────────┘    │ top             │
                                                 └─────────────────┘

┌────────────────┐    ┌──────────────────┐    ┌──────────────────┐
│ Forager NPC    │───►│ deliveries_by_   │───►│  YAML at         │
│ npcVisitVendors│    │ tier counter     │    │  foragers/<zone> │
└────────────────┘    └──────────────────┘    │  /<mobId>.yaml   │
                                              └──────────────────┘

┌────────────────┐    ┌──────────────────┐    ┌──────────────────┐
│ Caravan visit  │───►│ deliveries_by_   │───►│  YAML at         │
│ (dropoff side) │    │ tier counter     │    │  caravans/<zone> │
└────────────────┘    └──────────────────┘    │  /<mobId>.yaml   │
                                              └──────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│  /admin/economy/ dashboard                                       │
│  - Name from live mob OR mob template                            │
│  - StockScore % + delta-pp columns (replaces gold-delta)         │
│  - Tier-color bars (replaces bucket-color bars)                  │
│  - DeliveriesByTier tier-bar per forager / caravan               │
│  - StuckRounds + "(despawned)" / "(idle, no state)" diagnostics │
└──────────────────────────────────────────────────────────────────┘
```

## Component-by-component changes

### 1. Item-side: vendor_categories tag

**File:** `internal/items/items.go` (ItemSpec struct), every item YAML in
`_datafiles/world/dogmud/items/`.

Add `vendor_categories []string` to `ItemSpec` with YAML tag
`vendor_categories,omitempty`. Valid values mirror `shops.ValidCraftSupports`
EXCEPT `general` — items belong to a discipline; general stores accept
everything.

**Tag policy:**

- Raw mats — tag every discipline that uses the mat in any recipe.
  - `40057 lake mint` → `[alchemy]`
  - `40001 iron ingot` → `[blacksmithing, jewelcrafting]` (used in both)
  - `40002 leather strip` → `[blacksmithing, tailoring]` (sinew is more
    cross-cutting; check existing recipe data)
- Finished goods — tag with the producing discipline.
  - Iron sword → `[blacksmithing]`
  - Healing potion → `[alchemy]`
  - Cloak → `[tailoring]`
  - Ring → `[jewelcrafting]`
- Quest items, bound items, untradeable holdable props — leave empty.
  Validation skips items with `QuestToken != ""` (already a buy-rule reject).
- Items with `Value < 1` may also skip (no salable value).

**Migration:** sweep all item YAMLs as part of the implementation plan; tag
each in a single commit per discipline (alchemy, blacksmithing, etc.) so
diffs are reviewable. Validation in §6.1 panics at boot if any item is
missing a tag.

### 2. Buy rule rewrite (Approach 2 + gold-reserve gate)

**File:** `internal/shops/buyrules.go`.

Replace the entire 5-rule chain with a single tag-overlap rule. The
function signature stays the same so call sites (`internal/usercommands/sell.go`)
don't need to change.

```go
func EvaluateBuyRules(
    item items.Item,
    shopInv *ShopInventory,
    crafterSkill string,    // unused in new logic; kept for signature
    buysGeneral bool,       // unused in new logic; kept for signature
    cfg PricingConfig,
    wornItems []items.Item, // unused in new logic; kept for signature
) BuyOffer {
    spec := item.GetSpec()
    if spec.ItemId < 1 || spec.QuestToken != "" {
        return BuyOffer{}
    }

    // Spoiled / declining potions — refused regardless of tag match.
    if spec.Type == items.Potion && isPotionDeclining(item, spec) {
        return BuyOffer{}
    }

    // Tag-overlap gate.
    if len(spec.VendorCategories) == 0 {
        return BuyOffer{} // untagged items are not bought
    }
    if !vendorAcceptsAny(shopInv.CraftSupport, spec.VendorCategories) {
        return BuyOffer{}
    }

    // Overstock cap — the "48 iron ores" case.
    if entry := shopInv.GetStock(spec.ItemId); entry != nil &&
        entry.MaxStock > 0 && entry.Current >= entry.MaxStock {
        return BuyOffer{}
    }

    // Compute price.
    current, restock := 0, 1
    if entry := shopInv.GetStock(spec.ItemId); entry != nil {
        current = entry.Current
        if entry.RestockQty > 0 { restock = entry.RestockQty }
    }
    price := CalcBuyPrice(spec.Value, current, restock, cfg)

    // Gold-reserve gate.
    reserve := shopInv.GoldReserve(cfg.GoldReserveRatio)
    if !shopInv.CanAfford(price, reserve) {
        return BuyOffer{}
    }

    return BuyOffer{Price: price, Reason: pickReason(spec)}
}

// vendorAcceptsAny returns true if craftSupport is "general", or any
// of the item's tags matches craftSupport.
func vendorAcceptsAny(craftSupport string, itemTags []string) bool {
    if craftSupport == shops.CraftSupportGeneral {
        return true
    }
    for _, t := range itemTags {
        if t == craftSupport {
            return true
        }
    }
    return false
}

// pickReason returns the legacy Reason string for back-compat with
// any caller that inspects it (the sell command does, for flavor text).
func pickReason(spec *items.ItemSpec) string {
    if spec.Type == items.Potion { return "potion" }
    if spec.IsComponent          { return "craft_material" }
    return "general"
}
```

**Removed:**
- Rule 2 recipe-walk (`usesComponent`, `canCraftItem`).
- Rule 3 gear-upgrade (`isUpgrade`, `isEquipType`, all of `wornItems` flow
  in callers).
- Rule 5 `buysGeneral`.
- Helper functions `usesComponent`, `canCraftItem`, `isEquipType` deleted.

**Kept:**
- Quest-token reject (Rule 1).
- Aging-based potion reject (renamed/extracted as `isPotionDeclining`).

**Gold-reserve detail.** Existing helpers: `ShopInventory.GoldReserve(ratio)`
returns `int(StartingGold * ratio)`; `CanAfford(amount, reserveFloor)` returns
`Gold-amount >= reserveFloor`. We use the existing `ShopGoldReserveRatio`
balance config knob (no new field).

### 3. Per-vendor audit + gold defaults

**Files modified:**
- 6 mob YAMLs lose shopkeeper status (strip `crafter`, `crafterskill`,
  `crafterrecipeids`, `crafterrestockmaterials`, `craft_support`, `shop`,
  and where appropriate `behavior_archetype`).
- 16 mob YAMLs keep shopkeeper status; gold defaults are migrated.

**Cuts:**

| MobId | Name              | Zone           | Notes                                                    |
|-------|-------------------|----------------|----------------------------------------------------------|
| 52    | Korvath           | Sanctum Basin  | Keep `non_combatant: true` + `player_attack_immune: true`|
| 53    | Yenna             | Sanctum Basin  | Same as Korvath                                          |
| 333   | Sigrid            | Stillwater     | Innkeeper, dialogue/flavor only                          |
| 278   | Haral             | North Road     | Pure flavor                                              |
| 273   | Whisper           | Thornwall      | Pure flavor                                              |
| 348   | Bram              | Stillwater     | Reframe as flavor miller; drop `behavior_archetype`      |

**Save-file cleanup:** delete each cut shop's persisted state file
(`_datafiles/world/dogmud/shops/<zone>/<mobId>-room*.yaml`).

**Keepers and gold defaults** (all keep their `craft_support` tag):

| MobId | Name              | Zone               | Type       | New default starting_gold |
|-------|-------------------|--------------------|------------|-----------------|
| 63    | Adela             | Sanctum Basin      | general    | 5000            |
| 85    | Brecca            | Watcher's Crossing | general    | 5000            |
| 97    | Kerra             | Thornwall          | blacksmithing | 1000         |
| 98    | Voss              | Thornwall          | alchemy    | 1000            |
| 103   | Food vendor       | Thornwall          | cooking    | 1000            |
| 104   | Siv (fence)       | Thornwall          | general    | 5000            |
| 108   | Tess              | Thornwall          | jewelcrafting | 1000         |
| 109   | Vael              | Thornwall          | enchanting | 1000            |
| 113   | Maren             | Thornwall          | tailoring  | 1000            |
| 248   | Brynn             | Thornwall          | cooking    | 1000            |
| 336   | Tov Brann         | Stillwater         | cooking    | 1000            |
| 337   | Brindle           | Stillwater         | blacksmithing | 1000         |
| 338   | Ilsa              | Stillwater         | alchemy    | 1000            |
| 339   | Edda              | Stillwater         | tailoring  | 1000            |
| 340   | Kess              | Stillwater         | jewelcrafting | 1000         |
| 341   | Wulf              | Stillwater         | general    | 5000            |

**One-time hard reset of existing shop save files.** This change is
big enough — new buy rule, new tag system, new gold defaults, new stock
seeding — that piecemeal migration is risky and would leave shops in
half-old / half-new states. Instead:

- Delete the entire `_datafiles/world/dogmud/shops/` directory at the
  start of the implementation work. (Shops re-seed from mob templates on
  first boot via the existing `RegisterShop → loadFromDisk == nil → seed
  from template` path.)
- For the cut shopkeepers (Korvath, Yenna, Sigrid, Haral, Whisper, Bram),
  this also serves as their save-file cleanup since the entire dir is
  going away.
- Players lose any current shop economic state (NPC gold drift, current
  stock levels) — acceptable cost for a clean reset, since prod has been
  exercising these systems for less than two weeks and current state is
  unstable / heavily skewed by the bugs we're fixing.
- After the wipe, every shop boots fresh with: `starting_gold` per the
  table above (1000 specialist / 5000 general), `Stock` populated from
  the mob template's `crafterrestockmaterials` and `shop` lists,
  `LastRestock = 0`.

**No conditional migration logic** — the wipe-and-reseed is simpler,
more predictable, and avoids edge cases where a shop with manually set
`starting_gold: 200` (e.g., Ilsa today) ends up between worlds.

The same hard-reset principle applies to the **new** persistence
directories — there's nothing to migrate there yet:
- `_datafiles/world/dogmud/foragers/` — created fresh, populated as
  foragers tick.
- `_datafiles/world/dogmud/caravans/` — created fresh, populated as
  caravans tick.

**Settlement-tier pattern (memory for content design):**
- Small village / roadside inn: 1 general store, optionally 1 specialist.
- Town (Thornwall, Stillwater): all specialists + general + bank.
- Large city: sectional, all specialists + general + bank in multiple
  sections, elite vendors (later), auction house (later).

### 4. Tier-50/40 baseline restock

**File:** `internal/shops/shopinventory.go` (new method),
`internal/mobs/crafter.go` (call-site swap).

**New method:**

```go
// RestockBaselineTiers tops up StockEntries whose item carries
// rarity_tier 50 or 40, by RestockQty per call (capped at MaxStock).
// Skips entries with RestockQty <= 0 (NPC-crafted, untouched).
// Returns true if any stock was added.
func (si *ShopInventory) RestockBaselineTiers() bool {
    restocked := false
    for i := range si.Stock {
        e := &si.Stock[i]
        if e.RestockQty <= 0 { continue }
        spec := items.GetItemSpec(e.ItemId)
        if spec == nil { continue }
        if spec.RarityTier != 50 && spec.RarityTier != 40 { continue }
        room := e.MaxStock - e.Current
        if room <= 0 { continue }
        add := e.RestockQty
        if add > room { add = room }
        e.Current += add
        restocked = true
    }
    return restocked
}
```

**Call-site swap** in `internal/mobs/crafter.go::TickMobCraft`:

```go
// REPLACE:
//   if !b.IsCaravanServedZone(mob.Zone) {
//       restocked = shopInv.Restock()
//   }
// WITH:
if b.IsCaravanServedZone(mob.Zone) {
    restocked = shopInv.RestockBaselineTiers()
} else {
    restocked = shopInv.Restock()
}
```

Cadence stays at the existing `CrafterMaterialRestockRate` knob. No new
config.

**Edge case:** if a shop's stock list contains items with no `rarity_tier`
in their ItemSpec (deferred items per `mat-audit-matrix.md`), they're
skipped — they don't restock via this path. They were never restocking
via this path before either; this preserves that behavior.

### 5. Forager liveness diagnostics + watchdog

**Files:**
- `internal/economy/health/snapshot.go` — extend ForagerSnapshot.
- `internal/economy/health/capture.go` — distinguish despawned vs idle.
- `internal/behaviortree/actions_forager.go` — watchdog.
- `internal/configs/config.balance.go` — new `ForagerStuckThresholdRounds`
  knob.

**ForagerSnapshot additions:**
- `StuckRounds uint64` — `currentRound - state_entered_round`.
- State strings extended:
  - `"(despawned)"` — no live mob instance for the profile's MobId.
  - `"(idle, no state)"` — live mob exists, but `forager_state` BTreeState
    key is empty.

**captureForagers logic:**

```go
for each forager profile p in forager.AllProfiles():
    liveMob := find mob instance with MobId == p.MobId
    if liveMob == nil:
        emit ForagerSnapshot{State: "(despawned)", RoomId: p.SanctuaryRoom, ...}
        continue
    btreeState := liveMob.BTreeState
    stateName := btreeState.GetString("forager_state")
    if stateName == "":
        mudlog.Warn("forager state missing on live mob",
            "mobId", p.MobId, "name", p.Name, "roomId", liveMob.RoomId)
        emit ForagerSnapshot{State: "(idle, no state)", RoomId: liveMob.RoomId, ...}
        continue
    // Normal path — emit live snapshot with StuckRounds populated.
```

**Watchdog in actForagerStep:**

```go
// At top of actForagerStep, before reading current state.
startedStr := ctx.MobState.GetString(keyStateStartedRound)
started, _ := strconv.ParseUint(startedStr, 10, 64)
threshold := uint64(cfg.ForagerStuckThresholdRounds)
now := util.GetRoundCount()
// Guard against unsigned underflow when started has somehow ended up
// in the future (clock-skew, replay, fresh-init race).
if started > 0 && threshold > 0 && now > started &&
    now - started > threshold {
    mudlog.Warn("forager watchdog: stuck state, force-resetting to recalling",
        "mobId", mob.MobId, "name", profile.Name,
        "state", ctx.MobState.GetString(keyForagerState),
        "stuckRounds", now - started)
    transitionForager(ctx.MobState, forager.StateRecalling)
    return Success
}
```

**Why reset to Recalling, not Resting:** Recalling triggers
`dumpSatchelToLockbox` on arrival — orphaned items get put away. Resting
skips that step.

**New balance config:** `ForagerStuckThresholdRounds` (default 600 — about
10 game-hours; longer than any normal cycle phase).

### 6. Dashboard rework

**Files:**
- `internal/economy/health/snapshot.go` — new fields.
- `internal/economy/health/capture.go` — new metric capture.
- `internal/economy/health/delta.go` — new delta types.
- `internal/web/admin.economyhealth.go` — JSON wiring.
- `_datafiles/html/admin/economy/index.html` — dashboard JS / HTML.
- `internal/forager/throughput.go` — NEW file in the existing forager package.
- `internal/caravan/throughput.go` — NEW file in the existing caravan package.

**6.1 Name fallback.** In `lookupShopMobName`, after the live-instance
walk:

```go
// Fallback to template (always loaded at boot).
if t := mobs.GetMobTemplate(mobId); t != nil {
    return t.Character.Name
}
return ""
```

**6.2 StockScore + delta.**

`ShopSnapshot` adds:
```go
StockScore float64 `json:"stock_score"` // 0.0 to 1.0
```

Computed at capture time:
```go
total := 0; cap := 0
for _, e := range inv.Stock {
    total += e.Current
    cap += e.MaxStock
}
if cap > 0 { ss.StockScore = float64(total) / float64(cap) }
```

`ShopDelta` adds:
```go
StockScoreDelta int `json:"stock_score_delta"` // percentage points
```

**Dashboard table** — column changes:
- "Gold" column stays (current value, no delta).
- The five gold-delta columns (1h/6h/1d/3d/1w) become **stock-score-delta
  columns** showing `+Npp` / `-Npp` / `—`.
- Discipline rollup row: `StockScore` aggregated as
  `sum(currents) / sum(maxes)` across the discipline.

**6.3 Throughput counters: new files in existing packages.**

New `internal/forager/throughput.go` (added to the existing `forager`
package alongside `state.go`, `territory.go`, `forage_core.go`):

```go
type Throughput struct {
    MobId             int            `yaml:"mob_id"`
    Zone              string         `yaml:"zone"`
    DeliveriesByTier  map[int]int    `yaml:"deliveries_by_tier"`
    LastUpdatedRound  uint64         `yaml:"last_updated_round"`
}

// API mirrors internal/shops/persistence.go shape:
//   GetThroughput(zone, mobId) *Throughput
//   IncrementDelivery(zone, mobId, rarityTier int)
//   SaveThroughput(zone, mobId) error
//   SaveAllThroughputs()
//   PrewarmFromPersistedFiles(...) (int, error)
```

Identical structure in `internal/caravan/throughput.go`. Putting them in
the existing packages avoids package proliferation — both packages already
own the relevant domain logic.

**File paths:**
- `_datafiles/world/dogmud/foragers/<zone>/<mobId>.yaml`
- `_datafiles/world/dogmud/caravans/<zone>/<mobId>.yaml`

**Increment points:**

- **Forager:** in `actions_forager.go::npcVisitVendorsInRoom`, after each
  successful item handoff, call:
  ```go
  spec := items.GetItemSpec(item.ItemId)
  if spec != nil && spec.RarityTier > 0 {
      forager.IncrementDelivery(mob.Zone, int(mob.MobId), spec.RarityTier)
  }
  ```
  Persist via the existing `mutated` save trigger (extended to also save
  the throughput file).

- **Caravan:** in `internal/caravan/visit.go` (the dropoff side, where
  items are handed to destination shops; do NOT count pickups), same
  increment using `caravan.IncrementDelivery`.

**6.4 Snapshot wiring.**

`ForagerSnapshot` and `CaravanSnapshot` add:
```go
DeliveriesByTier map[int]int `json:"deliveries_by_tier"`
```

`captureForagers` reads via `foragers.GetThroughput(zone, mobId)` and
copies the map into the snapshot. Same for caravans.

**6.5 Throughput delta.**

```go
type ForagerDelta struct {
    DeliveriesByTierDelta map[int]int
    StuckRoundsDelta      int64 // optional, useful for "is the forager
                                 // making progress?"
}
type CaravanDelta struct {
    DeliveriesByTierDelta map[int]int
}
```

`ComputeForagerDelta(now ForagerSnapshot, old *ForagerSnapshot)` and
`ComputeCaravanDelta` mirror `ComputeShopDelta`.

**6.6 Tier-color bars.**

All colored bars (shop stock, caravan throughput, forager throughput)
switch from supply bucket → rarity tier. Five tier colors:

| Tier | Color   | Hex     |
|------|---------|---------|
| 50   | grey    | `#9aa0a6`|
| 40   | green   | `#5fb878`|
| 30   | blue    | `#5b9dd9`|
| 20   | purple  | `#a979e8`|
| 10   | gold    | `#d4a73a`|

**Three bar locations:**

1. **Shop stock bar (rightmost column on shop rows):**
   Segments are per-tier `Current / total_max`. Replaces today's per-bucket
   segments.

2. **Caravan throughput bar (replaces "cycles: see scoring"):**
   Segments are per-tier `DeliveriesByTierDelta` for the selected window,
   widths normalized within the bar (relative breakdown). Number to the
   right of the bar shows total items moved in window.

3. **Forager throughput bar (replaces "cycles: see scoring"):**
   Same as caravan — per-tier delta over selected window.

**Cargo-fill columns stay** ("47/200 lbs") on caravan and forager rows —
they answer "are they about to overflow."

**6.7 Implementation order**

1. Name template fallback (smallest, can ship first as a standalone PR).
2. StockScore + delta computation + dashboard column swap (gold-delta out,
   stock-score-delta in).
3. New `throughput.go` files in `internal/forager` and `internal/caravan`,
   following the `internal/shops/persistence.go` shape.
4. Snapshot capture wiring for new metrics (DeliveriesByTier on
   ForagerSnapshot / CaravanSnapshot).
5. Tier-color frontend updates (shop stock bar, caravan + forager
   throughput bars).

(`RarityTier` is already a public field on `ItemSpec` — no accessor work
needed.)

## Tests & validation

### Item tagging integrity (boot)

`internal/items/validation.go` — new file.

```go
func ValidateVendorCategories(specs map[int]*ItemSpec) error
```

Panics on any item that:
- has no `vendor_categories` AND `Value > 0` AND `QuestToken == ""`, OR
- carries an unknown vendor_category value (not in
  `shops.ValidCraftSupports` minus `general`).

Wired into `main.go::loadAllDataFiles` after items load. Cold boot
panics; reload logs Error.

### Recipe ↔ item tag coverage

`internal/crafting/validation.go` — new file.

```go
func ValidateRecipeIngredientTags(
    recipes map[string]*Recipe,
    specs map[int]*ItemSpec,
) error
```

For each recipe, look up the canonical item for each ingredient (by
ItemTag → ComponentTag match), verify the recipe's `Skill` is in the
item's `vendor_categories`. Cold boot panic on mismatches.

### Buy rule unit tests

`internal/shops/buyrules_test.go` — replace existing scenarios:

- Quest item → reject.
- Spoiled / declining potion → reject.
- Untagged item → reject.
- Specialist (alchemy) + alchemy-tagged mat → accept.
- Specialist (alchemy) + smithing-tagged mat → reject.
- Specialist (alchemy) + multi-tagged mat (alchemy+jewelcrafting) → accept.
- General + any tagged mat → accept.
- Vendor at MaxStock for that item → reject ("48 iron ores" case).
- Vendor with insufficient gold (would drop below GoldReserve) → reject.
- Old gear-upgrade scenarios all assert reject (regression: rule is gone).

### Tier-50/40 baseline restock

`internal/shops/shopinventory_test.go`:
- `TestRestockBaselineTiers_TopsUpTier50And40`
- `TestRestockBaselineTiers_SkipsRarerTiers`
- `TestRestockBaselineTiers_SkipsCrafterEntries` (RestockQty == 0)
- `TestRestockBaselineTiers_SkipsAtCap`

### Forager watchdog

`internal/behaviortree/actions_forager_test.go`:
- `TestForagerWatchdog_ResetsStuckMobToRecalling` — stuck > threshold,
  tick, asserts `forager_state == "recalling"`.
- `TestForagerWatchdog_DoesNotResetActiveForager` — recent state_started,
  tick advances normally.
- `TestCaptureForagers_DistinguishesDespawnedFromIdle` — fixture with one
  despawned profile + one live mob with empty BTreeState; asserts two
  distinct State strings.

### Throughput counters

`internal/foragers/throughput_test.go`:
- Counter increments on delivery.
- Save/load round-trip preserves counter.
- Snapshot capture reads from disk and reflects current value.
- Delta computation handles missing-tier keys correctly.

Identical mirrored tests in `internal/caravans/throughput_test.go`.

### Fresh-seed integrity (replaces old migration tests)

`internal/shops/persistence_test.go`:
- `TestRegisterShop_SeedsStartingGoldFromTemplate` — fresh shop with no
  save file picks up `starting_gold` from the mob template.
- `TestRegisterShop_SeedsStockFromCrafterRestockMaterials` — stock
  entries are created with `RestockQty` and `Current` matching the
  template seed convention.
- `TestRegisterShop_LoadsExistingSaveFile` — once a save file exists,
  it's loaded verbatim (the wipe is a one-time content op, not a
  per-boot behavior).

### New-character tutorial smoke (no regression in Sanctum Basin)

A `feel-tester` AI run from a **fresh character account** must complete
the Sanctum Basin tutorial chain end-to-end after this change ships.
Ensures vendor-rule + gold-default + cut-shopkeeper changes haven't
broken the new-player path. Specifically:

- Character creation completes.
- Chrysalis Priest mutation step works (Korvath / Yenna are still
  reachable, dialogue works, attack-immune holds).
- Adela still sells starter gear at expected prices.
- Combat trainer step completes.
- Player can leave Sanctum Basin (find Stillwater or Thornwall) with
  reasonable gold remaining.

A new goal file lives at
`tools/testing/goals/vendor-economy-polish-tutorial-regression.yaml`
that an admin runs against local before promoting to prod. This is a
content / UX safety check on top of the unit suite — the unit suite
proves the rule is correct, the smoke test proves the rule doesn't
strand new players.

### Boot smoke (per CLAUDE.md pre-push SOP)

Server must boot cleanly past datafile loading. Watch for:
- `mobs.LoadDataFiles()` — completes
- `items.LoadDataFiles()` — completes
- `ValidateVendorCategories()` — completes (no panic)
- `ValidateRecipeIngredientTags()` — completes (no panic)
- `ValidateShopMobTags()` — completes (no panic)
- `shops.PrewarmFromPersistedFiles()` — completes
- `forager.PrewarmFromPersistedFiles()` — NEW, completes
- `caravan.PrewarmFromPersistedFiles()` — NEW, completes

## Risks & mitigations

- **Item tag sweep is large.** Mitigated by per-discipline commits with
  validation enabled — boot fails fast on any miss.
- **One-time wipe loses live shop state.** Acceptable cost — the bugs
  we're fixing have skewed current state heavily, and the systems are <2
  weeks old in prod. The wipe is documented in PATCH_NOTES; no players
  permanently lose anything (their personal gold and inventories are
  untouched).
- **Tutorial regression risk.** Mitigated by the fresh-character feel-tester
  AI run before any prod push (see test §6.5). If the run can't get out
  of Sanctum Basin, ship is blocked.
- **Removed gear-upgrade rule could regress an existing test player flow.**
  Mitigated by the regression test in §6.3 (gear-upgrade scenarios assert
  reject), and shopkeepers were never wearing what they bought anyway.
- **Watchdog could mask a real bug by silently resetting.** The watchdog
  always logs Warn with the stuck state and round count. Diagnostics in
  the dashboard surface the same. Halix's first reset will tell us
  exactly what state he was stuck in.
- **Throughput files inside existing forager/caravan packages.** The btree
  action files already import both `forager` and `caravan` — adding
  throughput types to those same packages keeps the import graph
  unchanged. No new cycles introduced.

## Out-of-scope / future work

- Stillwater bank (memory: `project_stillwater_bank.md`).
- Auction house (city tier, far future).
- Elite vendors (city tier, far future).
- Multi-tag `craft_support` for cross-discipline shops (no need today).
- General store "won't buy if I can already sell at a loss" — current
  design covers via overstock cap; finer dynamic-pricing-aware refusal
  can come later if overstock cap proves insufficient.
- Caravan/forager throughput counter cleanup / decay (if files grow
  unbounded, add a cap or rolling window — not needed at current scale).
