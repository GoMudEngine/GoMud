# Enchanting Supply Chain — Tiered Potion Salvage → Enchanting Mats (Design)

**Status:** Draft — awaiting user spec review
**Date:** 2026-06-02
**Related:** the enchanting half of [[project_store_restock_considered_fix]] (cooking
half shipped 2026-06-02). Parallel to the cooking supply chain (waste stream →
under-served crafter): there, forager corpse-salvage → meat → cooks; here,
spoiled potions → enchanting mats → enchanter. Builds on 5.4 overstock decay.

---

## Goal

Give enchanting a renewable, difficulty-scaled material supply by salvaging
potions into enchanting reagents — the better/harder the potion, the better the
mats. Two consumers share one mapping:

- **Player salvage** (on-demand, primary): a player salvages a spoiled/declining
  potion and gets enchanting mats scaled to that potion's craft difficulty.
- **NPC living-economy loop** (passive, slow, GLOBAL): when *any* alchemy vendor's
  overstock potion decays (the existing 5.4 sweep), it's converted via the same
  mapping into a global enchanting-mat reserve, and every enchanter draws from
  that reserve (neediest mat gap first) on its restock tick — smoothing supply
  toward whichever enchanter player demand is draining hardest.

This fixes the real bottlenecks the survey found: **binding-paste** (used by all
19 enchants), **chrysalis-setting** (10+ enchants), and **mutation-catalyst**
(9 enchants) were salvage/vendor/caravan-starved; they now have a renewable
source tied to the abundant potion economy.

## Why now / what's broken

- **Enchanter Vael (109, Thornwall) is supply-starved.** Thornwall is a
  caravan-served zone, so Vael skips the per-mob cart restock; the caravan
  doesn't deliver to him; he's not a crafter. His chrysalis mats sit at 0. The
  only current source of chrysalis core/shard is Old Edrin's 75% boss loot.
- **Potions are an abundant, spoilable, under-used resource.** All 25 potions
  have `aging:` blocks and spoil to `PhaseSpoiled` (potency 0) — dead weight
  today.
- **The spoiled-potion → binding-paste salvage already exists** but is flat:
  `actions.Salvage`'s `spoiledPotion` branch (`internal/actions/salvage.go:201-207`)
  always yields `binding-paste` ×(1 or 2, scaled only by salvage chance),
  regardless of which potion. We make it tiered.

## Background — what already exists (reused, not rebuilt)

- **`actions.Salvage` (actor-pattern, player + mob)** with a `SpoiledPotion`
  mode. `salvageItem(actor, uuid, spoiledPotion, chance)` (salvage.go:164)
  currently hardcodes `{ItemTag: "binding-paste", Quantity: 1|2}`. The player
  command (`usercommands/salvage.go:57-73`) detects a spoiled/declining potion
  (Type potion + `Aging.HasAging()` + `CraftedRound>0` → `GetAgingPhase` is
  `PhaseSpoiled`/`PhaseDeclining`) and sets `SpoiledPotion: true`.
- **`crafting.GetRecipeByOutputItemId(itemId)`** (used at salvage.go:77) — maps a
  potion item id to the alchemy recipe that makes it → gives its `skill_minimum`
  (our difficulty key, since no potion has a `rarity_tier`).
- **5.4 overstock decay** (`internal/shops/overstock_decay.go`): drains
  non-component overstock above the RestockQty baseline, `is_component`-exempt,
  ~24-in-game-day grace (`ShopOverstockDecayRounds=21600`, `RoundsPerDay=900`),
  1 unit/fire. Crafted potions are added at RestockQty 0 (baseline 0) so they're
  drainable; the alchemy crafter re-stamps `LastGrewRound` on every craft, so
  only genuinely-abandoned surplus decays (modeled — cannot starve the shop).
- **Enchanter Vael's shop** already lists the mats (40028 binding-paste, 40027
  chrysalis-shard, 40029 mutation-catalyst, 40030 chrysalis-setting, 40010
  chrysalis-core, 40011 hive-fragment) — at quantity 0. Depositing via
  `AddStockAtRound` restocks those existing entries.
- **Alchemy vendors:** apothecary Voss (98, Thornwall) and apothecary Ilsa (338,
  Stillwater) — both `craft_support: alchemy` crafters that stock/produce potions.
  Enchanter: Vael (109, Thornwall) — the only enchanter today. Under the global
  reserve (decision 6), **both** Ilsa's and Voss's decayed potions feed the
  reserve and can supply Vael — no same-zone restriction, no new transport.
- **Globalized chest backfill (chunk 5.4 precedent):** `chestPoolAll` aggregates
  all forager chests; `selectBackfillTransfers(shopInv, pool)` is the generic
  neediest-gap allocator. This chunk reuses that allocator for the enchanter draw
  (lifting it to `internal/shops`).

## Design decisions (locked during brainstorming)

1. **Difficulty key = the potion's alchemy-recipe `skill_minimum`** (0–40 across
   the 25 potions). No potion has `rarity_tier`; `skill_minimum` is the natural,
   already-present difficulty proxy. Looked up via `GetRecipeByOutputItemId`.

2. **One shared mapping**, used by BOTH player salvage and the NPC decay loop, so
   they stay consistent. The mapping is `potion item id → []SalvageReturn`
   (enchanting mats + quantities), with per-band drop chances.

3. **Tiered band → mat mapping** (starting values; chances are config knobs):

   | Band | Potion skill_min | Example potions | Yield |
   |------|------------------|-----------------|-------|
   | 1 | 0–9 | healing salve, stamina tonic, ironhide, stone stomach | `binding-paste` ×1–2 (current behavior) |
   | 2 | 10–17 | mindshield, cat's eye, warrior's, windrunner, swiftfoot, veilguard | `binding-paste` ×1 + **~25%** `chrysalis-setting` ×1 |
   | 3 | 18–27 | berserker, lake-tonic, silver tongue, renewal, battle trance, essence of growth | `binding-paste` ×1 + **~35%** `chrysalis-setting` + **~12%** `mutation-catalyst` |
   | 4 | 28+ | savant's, purging, **mutagen brew, chrysalis catalyst** | **~40%** `mutation-catalyst` + **~30%** `chrysalis-setting` + **~8%** `chrysalis-core` (chrysalis-themed high-tier potions) |

   Rationale: the cheapest potions keep the current binding-paste behavior; the
   chrysalis/mutation-themed top-tier potions (mutagen brew skill 35, chrysalis
   catalyst skill 40) are the thematically-correct source for the rare chrysalis
   mats. binding-paste stays the reliable common floor across bands 1–3.

4. **Player salvage quantity still scales with salvage skill** (the existing
   `chance` arg): the band gives the mat *types* + base chances; salvage skill
   can bump quantities / improve the rare rolls (e.g. high salvage skill upgrades
   a band's chance). Exact skill-scaling is a tuning detail in the plan.

5. **NPC loop reuses the existing (glacial) decay rate — no new floor/slowdown**
   (Option A). Modeling showed the decay cannot starve the alchemy shop
   (self-balancing: in-demand potions are continuously re-crafted and never
   decay; components are decay-exempt). The loop is a slow living-economy feed;
   player salvage is the on-demand primary supply.

6. **NPC loop is GLOBAL, demand-routed — the analog of the globalized forager
   chests.** Decayed-potion mats from *every* alchemy vendor feed one **global
   enchanting-mat reserve** (a `map[matItemId]int` pool), and *every* enchanter
   draws from it on its restock tick to fill its **neediest** mat gaps
   (neediest-first, MaxStock-capped). So Ilsa's Stillwater decay backfills Vael
   in Thornwall, and with more alchemists/enchanters the reserve smooths supply
   toward whichever enchanter player demand is draining hardest — exactly how the
   chunk-5.4 chest backfill (`chestPoolAll` + `selectBackfillTransfers`) treats
   forager chests as one group. The neediest-gap allocator
   `selectBackfillTransfers(shopInv, pool)` is generic and is **reused** here with
   the reserve as the pool.

## Architecture / components

### Component 1 — Shared mapping (`internal/crafting/enchant_salvage_map.go`, new)

```go
// EnchantSalvageYield maps a potion (by item id → its alchemy recipe
// skill_minimum band) to enchanting-mat returns, rolling per-band chances.
// roll is an injected RNG [0,1) source so callers (player vs NPC) and tests are
// deterministic-friendly. qtyBonus (0+) lets high salvage skill bump yields;
// pass 0 for the NPC loop. Returns []crafting.RecipeIngredient (ItemTag +
// Quantity) to match the type the existing salvage `recovered` path uses.
func EnchantSalvageYield(potionItemId int, roll func() float64, qtyBonus int) []RecipeIngredient
```

(Returns `[]crafting.RecipeIngredient` — same type the salvage `recovered` block
already uses — so Component 2 is a drop-in assignment. The NPC loop resolves each
`ItemTag` → item id via `items.FindSpecByComponentTag(tag)` (the same helper the
crafter's salvage path uses, `crafter.go:308`) before `AddStockAtRound`.)

- Resolves the band: `GetRecipeByOutputItemId(potionItemId)` → `skill_minimum`
  (fallback band 1 if no recipe). Band thresholds + per-mat chances read from
  config knobs (decision 3).
- Always returns at least `binding-paste` for bands 1–3 (the common floor);
  band 4 may return only rare mats. Higher bands roll the rarer mats per the
  table.
- Pure + unit-testable via the injected `roll`.

### Component 2 — Player salvage uses the mapping (`internal/actions/salvage.go`)

Replace the hardcoded binding-paste block (salvage.go:201-207):

```go
if spoiledPotion {
    qtyBonus := 0
    if chance > 0.5 { qtyBonus = 1 } // preserve the existing skill bump
    recovered = crafting.EnchantSalvageYield(spec.ItemId, util.Rand01, qtyBonus)
}
```

(`util.Rand01` = a [0,1) helper; if none exists, wrap `util.Rand`.) Behavior for
band-1 potions is unchanged (binding-paste ×1–2); higher-band potions now yield
better mats. The player still sees the recovered items via the existing salvage
resolve path.

### Component 3 — NPC global loop: alchemy decay → reserve → enchanters

Mirrors the globalized chest backfill (one global pool → neediest gaps). Two
halves: feed and draw.

**3a. Global enchanting-mat reserve** (new, `internal/shops/enchant_reserve.go`
or a small dedicated store): a mutex-guarded `map[int]int` (mat item id → count)
with `AddToReserve(itemId, qty)`, `ReservePool() map[int]int` (copy), and
`DrainReserve(itemId, qty)`. In-memory v1 (resets on restart — it refills from
ongoing decay; persistence is a deferred decision, noted in the plan). This is
the analog of the aggregated forager-chest pool, but virtual (no physical
container).

**3b. Feed — decay reports removals; potions convert into the reserve.**
- Change `TickOverstockDecay(si, round)` to return `[]DecayedUnit{ItemId, Qty}`
  (currently returns nothing; existing callers ignore it). Pure shops change.
- At the hook layer (`internal/hooks/MobIdle_HandleIdleMobs.go`, the existing
  restock-tick market block where decay is already called): for each decayed unit
  whose item is a **potion** (`items.GetItemSpec(id).Type == items.Potion`), call
  `EnchantSalvageYield(id, util.Rand01, 0)` once per unit, resolve each result's
  `ItemTag → item id` (`items.FindSpecByComponentTag`), and `AddToReserve(matId,
  qty)`. (This is zone-agnostic — every alchemy vendor's decayed potions feed the
  one reserve.)

**3c. Draw — each enchanter pulls from the reserve, neediest-gap first.**
- On an **enchanter's** restock tick (it's a shopkeeper mob with
  `craft_support: enchanting`; reuse the same restock-tick hook point), build the
  reserve pool and reuse the generic allocator:
  `transfers := selectBackfillTransfers(enchanterShop, ReservePool())`
  (neediest gap first, only mats the enchanter stocks, MaxStock-capped). For each
  transfer: `DrainReserve(matId, qty)` + `AddStockAtRound(matId, qty)` into the
  enchanter's shop, then `SaveShop` the enchanter on mutation.
- Because the reserve is global and the allocator ranks by gap, the most
  demand-drained enchanter (largest gaps) is filled first when the reserve is
  scarce — the global smoothing the user wants.
- `selectBackfillTransfers` is currently unexported in `internal/forager`
  (chunk 5.4). It's pure and generic; **lift it to `internal/shops`** (e.g.
  `shops.SelectStockTransfers(si, pool)`) and have the forager chest backfill
  call the lifted version — so both the chest backfill and this enchanter draw
  share one allocator. (Small, DRY refactor; keeps the two "global pool → neediest
  gaps" consumers consistent.)

This lives at the hook layer (crosses shops + mobs + crafting); the pure decay +
reserve + allocator stay in `internal/shops`.

### Component 4 — Config knobs (`Balance`)

Per-band mat chances (so balance is tunable without code):
`EnchantSalvageBand2SettingPct` (25), `EnchantSalvageBand3SettingPct` (35),
`EnchantSalvageBand3CatalystPct` (12), `EnchantSalvageBand4CatalystPct` (40),
`EnchantSalvageBand4SettingPct` (30), `EnchantSalvageBand4CorePct` (8), plus
band thresholds `EnchantSalvageBand2Min` (10), `Band3Min` (18), `Band4Min` (28).
(Or a compact table struct — finalized in the plan.) No new decay knobs
(Option A reuses the existing rate).

## Data flow

```
# Player (on-demand, primary):
player salvage <spoiled potion>
  → usercommands/salvage.go detects spoiled/declining → SpoiledPotion=true
  → actions.Salvage → EnchantSalvageYield(potionId, rand, skillBonus)
  → player receives binding-paste / chrysalis-setting / mutation-catalyst / core

# NPC (passive, slow, GLOBAL):
any alchemy vendor restock tick → TickOverstockDecay returns decayed units
  → hook: for each decayed POTION unit → EnchantSalvageYield(potionId, rand, 0)
  → resolve tags → AddToReserve(matId, qty)        # global reserve, zone-agnostic
any enchanter restock tick
  → SelectStockTransfers(enchanterShop, ReservePool())   # neediest gap first
  → DrainReserve + AddStockAtRound into enchanter shop → SaveShop(enchanter)
# Ilsa (Stillwater) decay + Voss (Thornwall) decay both feed the one reserve;
# Vael (Thornwall) — and any future enchanter — draw from it, neediest first.
```

## Testing

- **Unit (`enchant_salvage_map_test.go`):** band resolution by skill_min
  (band 1/2/3/4 boundaries); a band-1 potion yields binding-paste; with `roll`
  forced low/high, band-3/4 potions yield the rare mats vs just the floor; an
  unknown/no-recipe potion falls back to band 1; `qtyBonus` increases quantities.
- **Unit (decay):** `TickOverstockDecay` returns the decayed `[]DecayedUnit`
  matching what it removed; non-potion decays still work; component-exempt holds.
- **Unit (reserve + draw):** `AddToReserve`/`DrainReserve`/`ReservePool` round-trip
  (concurrency-safe, copy-out); `SelectStockTransfers(enchanterShop, reserve)`
  fills the neediest mat gap first and respects MaxStock + stocked-only (the lift
  keeps the chest-backfill tests green); a scarce reserve goes to the largest-gap
  enchanter first (global smoothing).
- **Boot smoke:** clean boot; no panic (new knobs parse; mapping loads).
- **In-game smoke (deferred to user):** salvage a low-tier potion → binding paste;
  salvage a high-tier chrysalis-themed potion (mutagen brew / chrysalis catalyst)
  → see chrysalis-setting/mutation-catalyst/core rolls; over time, confirm Vael's
  binding-paste/chrysalis-setting stock ticks up from Voss's decayed potions;
  confirm the alchemy shop is NOT starved of potions (in-demand potions stay
  stocked).

## File touch list (anticipated; finalized in the plan)

- **New:** `internal/crafting/enchant_salvage_map.go` (+ test) — the mapping
- **Modify:** `internal/actions/salvage.go` — spoiled branch uses the mapping
- **Modify:** `internal/shops/overstock_decay.go` (+ test) — return decayed units
- **New:** `internal/shops/enchant_reserve.go` (+ test) — global mat reserve
  (`AddToReserve` / `ReservePool` / `DrainReserve`)
- **Refactor:** lift `selectBackfillTransfers` from `internal/forager` to
  `internal/shops` as `SelectStockTransfers` (the chest backfill calls the lifted
  version) — one shared neediest-gap allocator
- **Modify:** `internal/hooks/MobIdle_HandleIdleMobs.go` — feed decayed potions
  into the reserve; on enchanter restock tick, draw from the reserve
- **Config:** `internal/configs/config.balance.go` (+ validation + config.yaml) —
  band thresholds + chance knobs
- **Context docs:** `internal/crafting/context.md`, `internal/shops/context.md`
- **Roadmap/memory:** mark the enchanting half of
  [[project_store_restock_considered_fix]] addressed (general-store still open)

## Out of scope (explicit)

- **Slowing/flooring the alchemy decay** — Option A keeps the existing rate
  (modeled safe).
- **Enchanter-as-crafter / new material-producing recipes / a distill mechanic**
  — unnecessary; salvage is the mechanism.
- **Fixing the chrysalis-core/shard rare supply beyond the band-4 salvage chance**
  — boss loot (Old Edrin) remains; the caravan-routing gap to Vael is a separate
  optional fix (the band-4 salvage now gives a renewable trickle anyway).
- **General-store restock** — the third store type from the original report; not
  this chunk.
- **Reserve persistence across restarts** — v1 reserve is in-memory (refills from
  ongoing decay); persisting it is a deferred decision (plan-time).
- **Generalizing the reserve beyond enchanting** — a single global reserve any
  vendor type could draw from is a future idea; this chunk scopes it to the
  alchemy-decay → enchanter flow (YAGNI).
