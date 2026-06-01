# Mob Aliveness 5.3 — Equipment-Aware Shopping (Design)

**Status:** Approved — ready for implementation plan
**Date:** 2026-06-01
**Roadmap chunk:** 5.3 (Phase 5 — Cross-cutting features) • **Size:** L
**Depends on:** 2.1 (mob `buy`), 2.2 (`itemvalue` comparison), 2.3 (equip-if-better / `gearup`), 4.4 (strategic→tactical planners + `try_goal_planner`)

---

## Goal

NPCs that carry and use gold periodically improve their equipment by shopping:
they survey their own loadout, identify the upgrade that would help most,
travel to a vendor in their zone that stocks it, sell junk loot to fund the
purchase if needed, buy it, and equip it. A town thug or guard who picks up
the steel sword you dropped and shows up better-armed than the last time you
saw them is far more memorable than one frozen in starter gear.

This composes existing substrate — it is mostly **one new evaluator primitive,
one new goal type, one new planner, and archetype wiring** — not new
infrastructure.

## Why now

Phase 4 (4.4) shipped a `mastery-equip` planner, but it is an explicit
best-effort stub: it walks to *any* shop in the zone and issues a blind
`buy <slotname>` hint, using **rarity tier** as the upgrade proxy rather than
the 2.2 `itemvalue` comparison. Its own source comment says *"a slot-aware
stock filter is not available in 4.4."* 5.3 builds that missing filter and the
real decision loop on top of it.

## Background — what already exists (reused, not rebuilt)

- **`itemvalue` package (2.2):** `ItemValueDelta(char, profile, candidate)
  SwapDelta` already does smart slot placement (rings pick the weaker occupant,
  1H weapons compare Weapon vs Offhand, 2H displaces both) and returns a swap
  delta. `ProfileFor(stat, behavior)` derives an archetype-weighted
  `WeightProfile`. `IsUpgrade(char, profile, candidate)` is the boolean wrapper.
- **`shops` package:** `shops.AllShops()` exposes per-shop `Zone`, `RoomId`,
  `Gold`, and stock entries; `ShopInventory.GetStock(itemId)` returns the live
  entry (`Current`, `RestockQty`, `MaxStock`); `shops.CalcBuyPrice(value,
  current, restockQty, cfg)` computes the price a buyer pays under dynamic
  pricing; `PricingConfigFromBalance()` builds the config.
- **`wealth-gold` planner (4.4):** a working sell-loot-to-accumulate-gold loop
  (`pathto` a buying vendor → `sell all`), with `mobHasSellableItems` and
  `mobInVendorRoom` / `findShopInZoneBuying` helpers. 5.3 composes its logic for
  the "can't afford yet" branch.
- **`actions.Buy` (2.1)** with carry-capacity + gold gates; **`gearup`
  mobcommand (2.3)** which wears the best items from the backpack using
  `itemvalue`.
- **`try_goal_planner` btree action (4.4)** is already inserted into every
  non-boss archetype tree, so **no behavior-tree edits are required.**

## Design decisions (locked during brainstorming)

1. **Drive shape — survey-worst-slot.** The mob does not target a pre-authored
   slot. It scores every in-stock item against its current loadout; the highest
   positive `ItemValueDelta` naturally targets whichever slot benefits most.
2. **Funding — sell-then-buy in one drive.** If short on gold, the planner
   composes the existing `wealth-gold` sell-loot loop before buying. Self-
   contained "save up for armor."
3. **Shop reach — same zone only.** A mob shops only if a vendor exists in its
   current zone. Wilderness mobs simply never trigger the drive. (Matches the
   zone-scoped shop helpers.)
4. **Who shops — standing default on gold-using archetypes.** A low-weight
   `upgrade-gear` default goal on humanoid combat archetypes that already carry
   gold and operate where shops exist. Goal selection surfaces it when idle and
   gold allows.

---

## Architecture

### 1. Shop-stock evaluator (new planner-local helper)

Lives in `internal/planners/` (alongside the existing shop helpers). It is
deliberately **not** in `itemvalue` — that package stays free of a `shops`
import; the planner layer owns the shop-aware composition, mirroring the
existing `findShopInZoneSelling` pattern.

```go
// findBestUpgradeInZoneShops scans every available stock entry across all
// shops in the mob's zone, scores each as a potential swap for the mob via
// itemvalue, prices it under dynamic pricing, and returns the single best
// AFFORDABLE positive-delta upgrade across the zone.
//
// budget is the spendable gold (mob.Gold - reserve). minDelta gates out
// marginal upgrades. Returns ok=false when no qualifying upgrade exists.
func findBestUpgradeInZoneShops(
    mob *mobs.Mob, profile itemvalue.WeightProfile, budget int, minDelta float64,
) (res upgradeCandidate, ok bool)

type upgradeCandidate struct {
    ShopRoom int
    ItemId   int
    ItemName string  // canonical shop display name, for the `buy <name>` command
    Price    int
    Delta    float64
}
```

Algorithm:
1. For each shop in `shops.AllShops()` with `shop.Zone == mob.Character.Zone`:
   - For each available stock entry (`Available()` true, `Current > 0` or
     unlimited): build the candidate `items.Item` from `entry.ItemId`.
   - `delta := itemvalue.ItemValueDelta(mob.Character, profile, candidate)` —
     skip if delta ≤ `minDelta`.
   - `price := shops.CalcBuyPrice(...)` via the shop's stock entry + balance
     pricing config — skip if `price > budget`.
   - Track the best by delta (tie-break: lower price).
2. Return the best, with its shop room, or `ok=false`.

Also a cheaper companion used by `ContextScore` and the "save toward it"
branch:

```go
// bestUpgradePrice returns the price of the cheapest positive-delta in-stock
// upgrade in the zone IGNORING current gold (so the goal can decide whether
// it is worth saving toward). ok=false when no positive-delta upgrade exists
// in stock at any price.
func cheapestUpgradePrice(mob *mobs.Mob, profile itemvalue.WeightProfile, minDelta float64) (price int, ok bool)
```

### 2. Goal type — `upgrade-gear` (`internal/goals/catalog/upgrade_gear.go`)

A **perpetual drive** — it has no terminal "done" state; a mob can always want
better gear.

- **Params:** none required. Optional `reserve` (int) overrides the config
  default gold floor.
- **`IsSatisfied`:** always `false`. The drive never completes; activation is
  governed entirely by `ContextScore`.
- **`ContextScore`:** returns a small positive **floor** (e.g. `1.0`) in all
  cases so the 4.6 dormancy sweep never abandons a standing default goal, and
  a meaningfully higher score when an affordable in-stock upgrade exists and
  the mob is idle + out of combat. This makes goal-selection surface
  `upgrade-gear` only when it is actionable, while keeping it alive as a low-
  priority background want otherwise.
- **`AllowMultiple`: false.** One per mob.
- **Seeding:** archetype default via the existing `default_goals:` block. The
  `SeededFromArchetype` sentinel means it survives admin `goal clear` and
  reseeds on spawn.
- **`ParamSchema`:** declares the optional `reserve` int so the engine's
  `Add`-time validation accepts it (undeclared params cause a panic per the
  4.3 conventions).

### 3. Planner — `upgrade-gear` (`internal/planners/upgrade_gear.go`)

Stateless `PlanFn`, called per-tick when `upgrade-gear` is the current goal.
Intermediate state in `mob.Character.MiscData` under the `plan:upgrade-gear:`
prefix (sticky shop room), wiped on goal switch by the existing
`SetPlanStateClear` callback.

Branch order per tick (the mob is, by construction, idle and out of combat
when a planner runs):

1. **Misconfigured / nil mob** → `Failure`.
2. Build `profile := itemvalue.ProfileFor(...)` from the mob's stat tendency +
   behavior archetype.
3. `budget := mob.Gold - reserve`. Call `findBestUpgradeInZoneShops(mob,
   profile, budget, minDelta)`.
   - **Affordable upgrade found:**
     - At its shop room (`mob.RoomId == cand.ShopRoom`)? → emit
       `buy <cand.ItemName>`; on the **next tick** emit `gearup` to wear it,
       then clear the sticky room. (Buying lands the item in the backpack;
       `gearup` is the 2.3 itemvalue-aware wear step. We do **not** rely on the
       floor-loot auto-equip path, which only fires on room pickup.)
     - Not there? → sticky-cache `cand.ShopRoom` + emit
       `pathto <cand.ShopRoom>`. `Running`.
4. **No *affordable* upgrade, but a positive-delta upgrade exists in stock**
   (`cheapestUpgradePrice` returns `ok` with `price > budget`) → **save toward
   it** by composing the `wealth-gold` sell loop:
   - Has sellable junk + in a buying vendor room → `sell all`.
   - Has sellable junk + not at a vendor → sticky `findShopInZoneBuying` +
     `pathto`.
   - Nothing to sell → `wander` (hope for loot / a better drop). `Running`.
5. **No positive-delta upgrade in stock anywhere in zone** → `Running` with no
   command (idle); `ContextScore` will have already fallen to the floor so
   selection moves on to another goal or plain idle behavior.

The two `gearup`-after-`buy` ticks are sequenced with a one-shot MiscData flag
(`plan:upgrade-gear:pending_equip`) so the planner knows the previous tick
issued a buy and this tick should `gearup` rather than re-evaluate stock
(prevents a same-tick double-buy if stock/pricing shifted).

### 4. Config knobs (`Balance`)

| Knob | Default | Purpose |
|------|---------|---------|
| `MobUpgradeGoldReserve` | 50 | Gold floor a shopping mob will not spend below. |
| `MobUpgradeMinDelta` | (small, e.g. 1.0) | Minimum `itemvalue` swap delta worth buying — prevents churn on marginal upgrades and on the `gearup` round-trip. |

Per-archetype priority is expressed in the archetype YAML `goal_weights:` /
`default_goals:` blocks — no new knob.

### 5. Archetype wiring

Add `upgrade-gear` as a **low-weight** `default_goals:` entry on gold-using
humanoid combat archetypes that operate in zones with vendors. Candidate set
(finalized at plan time against current archetype YAML): `thief`,
`guard` / `guard_captain`, and any mercenary-style fighter archetype.
**Excluded:** wilderness predators, animals, casters-without-gold, boss
archetypes.

For the smoke test, seed the goal concretely on one or two town mobs in a zone
that has a shop (e.g. a Thornwall or Stillwater humanoid) — exact mobs chosen
during plan-writing.

No behavior-tree edits: `try_goal_planner` is already present in every
non-boss tree.

---

## Out of scope (explicit)

- **Cross-respawn persistence — the "shows up in better gear *next time*"
  line.** Mob instance saves are wiped in prod and per the local smoke-test
  SOP, so purchased gear persists only for the **instance's lifetime**, not
  across death/respawn. A remembered-loadout system is a separate persistence
  design; 5.3 delivers within-lifetime upgrade behavior. (Within a long-lived
  instance the mob genuinely walks around in the better gear it bought.)
- **Adjacent-zone shopping.** Same-zone only by decision. The `zoneAdjacentTo`
  helper exists if we ever want wilderness mobs to "go to town," but that is a
  later extension.
- **Touching the existing `mastery-equip` planner.** It serves author-directed
  "upgrade slot X to tier Y" goals. It is now partially redundant with
  `upgrade-gear` but is left alone to keep scope tight; deprecation is a
  possible later cleanup, not part of 5.3.
- **5.4 — NPCs selling looted/crafted goods into shop stock** (the economy
  feedback direction). Separate chunk; depends on this one's plumbing.
- **NPCs commissioning custom crafts from player-crafters** (already Out in the
  roadmap).

---

## Testing

**Unit tests:**
- *Evaluator* (`findBestUpgradeInZoneShops`): highest-positive-delta candidate
  wins; ties broken by lower price; items priced above budget are excluded;
  the gold reserve is respected; `minDelta` filters marginal upgrades;
  `ok=false` when no qualifying upgrade exists; multi-shop zone picks the best
  across all shops and returns the correct room.
- *`cheapestUpgradePrice`*: ignores current gold; `ok=false` when no positive-
  delta upgrade in stock.
- *Planner* state transitions: buy-then-equip when at shop + affordable;
  pathto when remote; sell-loop when an upgrade exists but unaffordable;
  idle/Running when nothing in stock; misconfigured/nil → Failure;
  pending-equip one-shot sequences `buy` then `gearup`.
- *Goal*: `IsSatisfied` always false; `ContextScore` returns the positive
  floor when nothing affordable and a higher score when an affordable upgrade
  exists + mob idle/out-of-combat; `ParamSchema` accepts optional `reserve`.

**Boot smoke:** server starts cleanly past data-file load (archetype
`default_goals` parse, goal-type registration symmetry check).

**In-game smoke (deferred to user, per the 2.8/2.9/2.10 precedent):** seed a
town mob with some starting gold, a low-value junk item, and place it in a zone
whose shop stocks a clear upgrade for one of its slots. Observe: path to shop →
(sell junk if short) → buy the upgrade → `gearup` → mob now wears it. Verify it
does not loop-buy, does not spend below the reserve, and ignores marginal
upgrades below `minDelta`.

---

## File touch list (anticipated; finalized in the plan)

- **New:** `internal/goals/catalog/upgrade_gear.go` (+ test)
- **New:** `internal/planners/upgrade_gear.go` (+ test)
- **New evaluator helpers:** added to `internal/planners/helpers.go` (or a new
  `shop_upgrade.go` in the package) (+ test)
- **Config:** `internal/configs/config.balance.go` — two new knobs +
  `config.yaml` defaults
- **Archetype YAML:** `default_goals:` additions on the chosen archetypes
- **Context docs:** `internal/planners/context.md`,
  `internal/goals/catalog/context.md` updates
- **Roadmap:** flip 5.3 status; note 5.1d as "done as far as taken" while in
  the doc
