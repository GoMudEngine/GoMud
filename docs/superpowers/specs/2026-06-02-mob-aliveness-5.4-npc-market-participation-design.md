# Mob Aliveness 5.4 — NPC Market Participation (Design)

**Status:** Draft — awaiting user spec review
**Date:** 2026-06-02
**Roadmap chunk:** 5.4 (Phase 5 — Cross-cutting features) • **Size:** L
(roadmap lists M; expanded to L during brainstorming — see "Scope note")
**Depends on:** 5.3 (equipment-aware shopping plumbing), 2.1 (`actions.Buy`
lift precedent), 4.4 (planners + `try_goal_planner`), 2.9 (forager state
machine), 3.8 (forager delivery patrol + `SellToVendor` handoff),
2.10-followups (forager locked-chest workflow)

**Supersedes:** `2026-06-01-mob-aliveness-5.4-npc-market-participation-design.md`
(the prior, leaner approved spec). This version keeps that spec's verb-lift
detail, anti-thrash design, and disposal stance, and *adds* the chest backfill,
time-based overstock decay, crafter targeting, and the shops-never-broke gold
model agreed in the 2026-06-02 re-brainstorm. See "Reconciliation with the
06-01 spec" for the exact deltas.

---

## Goal

NPCs sell looted and crafted goods through normal shop channels so that shop
stock reflects NPC activity, not just player drop-offs. The living economy
becomes a two-sided market:

- **Combat looters** offload displaced gear (the steel sword you dropped, now
  the bandit's old weapon) back into town shops for gold.
- **Crafters** carry surplus output to vendors (a near-no-op today, since
  crafters are also shopkeepers — future-facing for specialized bigger cities).
- **Foragers** keep supplying raw goods, and their **chest "overflow cache"
  finally drains back into circulation** instead of being a one-way sink.
- **Vendors** slowly clear non-material junk nobody buys, so shelves don't
  ossify.

## Why now

Three threads converge here:

1. **The `sell` verb gap** ([[project_mob_sell_command_missing]]). There is no
   `actions.Sell` and no mob `sell` command. The `wealth-gold` (4.4) and
   `upgrade-gear` (5.3) planners both issue `sell all`, which falls through mob
   dispatch to `emote looks a little confused (sell all)` — a visible no-op.
   Their entire sell-to-fund path is dead until the verb exists.
2. **The forager chest is a one-way drain**
   ([[project_vendor_backfill_from_forager_chests]]). Chunk 2.10-followups
   added a "deposit unsold goods" chest as an *overflow cache* whose goods were
   meant to flow back to vendors — but no backfill mechanism was ever built.
3. **5.4 itself** — the roadmap's last non-polish aliveness chunk — *is* "NPCs
   sell through shops." It cannot function without (1) and is the natural home
   for (2).

Bundling all three into one chunk was the explicit user decision during
brainstorming.

## Scope note (M → L)

The roadmap sizes 5.4 at M. This design is L: verb lift **+** a proactive
surplus-offload goal **+** the chest backfill (aggregate/demand-routed with
back-pressure) **+** time-based overstock decay. Recorded here so the size
delta isn't a surprise at plan time.

## Background — what already exists (reused, not rebuilt)

- **Shop-buys-from-seller is fully wired.** `shops.EvaluateBuyRules(item,
  shopInv, crafterSkill, buysGeneral, cfg, wornItems) BuyOffer` decides whether
  a shop buys an item and at what price, and already **rejects** quest items,
  items with no `VendorCategories`, categories the vendor doesn't accept,
  declining potions, and anything at the shop's `MaxStock` overstock cap
  (zero-price offer).
- **Seller payout already diminishes as stock fills.** The seller's price runs
  the same `ScarcityMultiplier` as buy pricing — offloading into a well-stocked
  shop pays progressively less, automatically. (No extra work; just noted so the
  economics are understood.)
- **`usercommands/sell.go`** — the ~400-line player sell command, `UserRecord`-
  bound: `trySellOne` (find item → `EvaluateBuyRules` → gold transfer →
  `AddStockAtRound` / `BuysCount++` / `SaveShop`, plus the `gear_upgrade`-wear
  branch), merchant resolution (`room.GetMobs(FindMerchant)` + buy-probe),
  quantity parsing (`sell N` / `sell all` / `all.name`), confirmation messaging.
  This is the lift surface.
- **`forager.SellToVendor`** (`internal/forager/vendor_sell.go`) — a **free
  stock handoff**, NOT a sale: bucket-matching satchel items move into matching
  vendor stock entries with **no gold transfer**, gated on `entry.Current <
  entry.MaxStock`. Fires from `ForagerArrivalListener` on delivery-patrol
  arrival. Stays as the forager/chest supply path; the chest backfill extends it.
- **Mobs accumulate sellable surplus.** `EquipBestFloorItem`
  (`internal/hooks/mob_equip_best_floor_item.go`) picks up the best floor item
  and `StoreItem`s it; 5.3 `gearup` displaces the old slot occupant into the
  backpack. Plus steal/give intake. A mob that upgrades is left holding the old
  piece — the surplus this drive offloads.
- **`actions.Buy` (2.1)** is the actor-pattern template for the `actions.Sell`
  lift. Planner helpers `findShopInZoneBuying`, `mobInVendorRoom`,
  `mobHasSellableItems` already exist (4.4 `wealth-gold`).
- **forager state machine** (2.9) — `StateForaging`/`StateDelivering`/
  `StateStoring`/`StateRecalling`; `StorageChestRoom`; chest-deposit primitive.
  The new `StateResting` back-pressure state slots in here.
- **`economy.BucketFor(itemId)`** — maps items to economic buckets; foragers
  carry `Buckets`. Decides which vendor wants which goods.

## Design decisions (locked during brainstorming)

1. **Two economic models, made explicit.**
   - **Sale (gold transfer):** `actions.Sell`. Seller is credited; item enters
     vendor stock. Used by planner-driven sellers (combat-looter offload,
     gear-upgrade funding, wealth-gold) and players (unchanged).
   - **Supply handoff (no gold):** `forager.SellToVendor` and the chest
     backfill. Supplier feeds vendor stock for free — a different relationship
     (the forager works *for* the vendor economy).
   - Documented in `internal/shops/context.md` and the forager context so "why
     two sell paths" is answered up front.

2. **Shops do not run out of gold from NPC selling.** When a **mob** sells via
   `actions.Sell`, the seller is credited but the shop's gold reserve is **not
   deducted**, and the merchant-gold check is skipped (mob sales never hit
   `MerchantBroke`). This keeps shop liquidity intact for *player* sells. A
   small, deliberate gold-creation leak, acceptable for now and trivially
   reversible (it gates on `seller.IsPlayer()`). Player sells are unchanged —
   they still draw down shop gold. *(Note: this is the opposite of the 06-01
   spec, which deducted shop gold; superseded per the re-brainstorm.)*

3. **`sell all` (no item name) is a mob inventory-sweep mode.** The player path
   keeps requiring an item name. `SellOptions` gains `SellAllSellable bool`; in
   that mode the seller offloads every actionable-surplus item to any willing
   merchant in the room, skipping rejected items rather than aborting.

4. **One proactive offload goal (`sell-surplus`), seeded on combat looters AND
   crafters.** Perpetual-with-floor goal (5.3 `upgrade-gear` model). Combat
   looters (`generic_fighter`, `leader`, `predator`, `thief`, `guard_captain`)
   accumulate displaced gear via `EquipBestFloorItem`/`gearup` and are the
   present-day economy contributors; `crafter` is added for future specialized
   cities (near-no-op while crafters are their own shopkeepers). Off pure-combat
   archetypes that don't loot. No btree edits (`try_goal_planner` present).

5. **Anti-thrash via rejection blacklist.** The planner calls `actions.Sell`
   **directly** (not via an emitted command string) so it can read the
   `SellResult`. A rejected item id is blacklisted in plan-state and excluded
   from future surplus, so the mob stops re-offering junk a vendor won't take.
   Once every remaining surplus item is blacklisted, `ContextScore` falls to the
   floor and the mob goes quiescent. Worst case: one wasted vendor trip, then
   quiet. The blacklist is `plan:`-prefixed, wiped by `ClearPlanState` on goal
   switch (occasional self-healing re-try).

6. **Overstock decay excludes crafting materials.** New time-based clearance
   shrinks unsold vendor stock held *above* a restock baseline so abandoned junk
   drains out. Items flagged `is_component: true` are **never decayed** —
   topped-up material stock is fine and stays. New knob
   `ShopOverstockDecayRounds` (conservative default). This *adds* active
   drainage on top of the existing `MaxStock` cap (the cap prevents *more*
   pileup; decay removes *existing* abandoned stock). *(The 06-01 spec treated
   caps as sufficient; superseded.)*

7. **Disposal of truly-unsellable junk: keep it.** A mob that can't sell an item
   keeps it (no ground-drop, no destroy — respects the parked loot-drop
   redesign). The donation-bin disposal sink is a future **chunk 5.5** that will
   retrofit the offload terminal action to donate instead of keep. *(Preserved
   from 06-01.)*

8. **Chest backfill = aggregate + demand-routed + back-pressure.**
   - All forager chests in a zone form an aggregate supply pool.
   - **Per-vendor restock tick** (chosen over a coordinated zone sweep): when a
     vendor restocks, it pulls bucket-eligible goods from the aggregate chest
     pool to fill its neediest stock entries (largest `MaxStock - Current` gaps
     first), capped at `MaxStock`. "Neediest-first" is per-vendor and ordered by
     which vendor ticks — an accepted approximation.
   - If every eligible vendor is topped off, goods **stay in the chest**.
   - **Back-pressure:** a forager whose chest reaches 100% enters `StateResting`
     and stops gathering until the backfill drains the chest to **≤ 90%**, then
     resumes. Self-regulating: a forager gathering a good nobody buys rests until
     demand returns. Intended behavior, not a stall bug.
   - Free supply-handoff model — **no gold**, chest → vendor stock.

## Architecture / components

### Component 1 — `actions.Sell` (the verb lift)

New `internal/actions/sell.go`:

```go
type SellOptions struct {
    ItemName        string // ignored when SellAllSellable
    Quantity        int    // 1, N, or unlimited sentinel (math.MaxInt)
    SellAllSellable bool   // mob inventory-sweep mode
    MerchantName    string // optional "sell X to <merchant>"
}

type SellStopReason int
const (
    SellStopSoldAll       SellStopReason = iota // ran out of matching items (normal)
    SellStopNoItem                              // seller never had the item
    SellStopNoMerchant                          // no willing merchant in room
    SellStopMerchantBroke                       // merchant ran out of gold (player path only)
    SellStopRejected                            // merchant declined the item type
)

type SellResult struct {
    Sold         int
    TotalGold    int
    Reason       SellStopReason
    LastItemName string // for messaging
}

func Sell(seller Actor, opts SellOptions) SellResult
```

- **Seller-side calls abstracted behind `Actor`:** inventory find
  (backpack → potions → components), `RemoveItem`, `Gold +=`,
  `OnSkillUse(bartering)`, feedback (`SendText`). The barter *bonus*
  (`shops.ApplyBarterBuyBonus`) reads `seller.GetCharacter().GetSkillLevel(
  skills.Bartering)` — symmetric for player and mob.
- **Player-only side-effects gate on `seller.IsPlayer()`:** `EventLog.Add`,
  `events.AddToQueue(ItemOwnership / EquipmentChange{UserId})`, player-facing
  confirmation/early-stop text, and `command:sell` quest notification.
  `MobActor.SendText` is already a no-op; merchant-refusal lines (spoken via
  `actions.Say`) still broadcast while the mob seller silently reads the result.
- **Gold model branch (decision 2):** credit the seller always; deduct shop gold
  and run the merchant-gold check **only when `seller.IsPlayer()`**.
- **`SellAllSellable` (decision 3):** iterate the seller's inventory, offer each
  actionable-surplus item to a willing merchant; skip rejected items.
- **Merchant side untouched:** `EvaluateBuyRules`, `AddStockAtRound`,
  `BuysCount++`, `SaveShop`, the `gear_upgrade`-wear branch, the legacy
  `mob.Character.Shop` path move verbatim.
- **Wrappers:** `usercommands/sell.go` collapses to parse + delegate + render
  from `SellResult`. New `mobcommands/sell.go`, registered in `mobCommands`.
- Cross-reference `forager.SellToVendor` from the `actions.Sell` doc comment so
  the sale-vs-supply split is discoverable.

This is the larger, riskier half. The plan must enumerate exactly which
seller-side calls move behind `Actor` and which stay `IsPlayer()`-gated.

### Component 2 — surplus-offload drive (`sell-surplus`)

Twin of 5.3's `upgrade-gear` (perpetual goal + planner).

**Goal type `sell-surplus`** (`internal/goals/catalog/sell_surplus.go`):
- `Predicate` always false (perpetual).
- `ContextScore`: floor `1.0` (4.6 dormancy never abandons it), rising to `2.5`
  when idle/out-of-combat **and** carrying actionable surplus. No shop scan in
  scoring (planner owns vendor decisions), per 5.3.
- Optional `min_value` int param overriding the config floor.

**"Actionable surplus"** = a non-equipped backpack item that: has vendor
`Value > 0` and `Value >= MobSurplusMinValue`; is not a quest item; is not a
component-bag material the mob may use; is not on the per-mob unsellable
blacklist.

**Planner `sell-surplus`** (`internal/planners/sell_surplus.go`):
- No actionable surplus → idle.
- Surplus, not at a buying vendor → sticky-resolve `findShopInZoneBuying`
  (`plan:sell-surplus:sell_shop_room`) + `pathto`.
- Surplus, at a buying vendor → sell the highest-value actionable item by
  **calling `actions.Sell` directly** (reads `SellResult`):
  - `SellStopRejected` / zero sold → blacklist that item id in
    `plan:sell-surplus:unsellable`.
  - sold → progress; re-evaluate next tick.
  - `SellStopNoMerchant` → clear sticky shop room, idle/re-resolve.
- The direct call (vs. emit-command) is the one intentional divergence from the
  5.3 pattern — documented in `planners/context.md`. Needed for the reject
  signal that anti-thrash depends on.

**Archetypes:** seed `sell-surplus` as a low-priority `default_goals` entry on
`generic_fighter`, `leader`, `predator`, `thief`, `guard_captain`, and
`crafter`.

### Component 3 — overstock decay

- New `internal/shops/overstock_decay.go`: on a vendor's restock tick, for each
  stock entry where `Current > restockBaseline` and the item is **not**
  `is_component`, decrement `Current` by a small amount once
  `ShopOverstockDecayRounds` have elapsed since the entry last grew. Decay never
  drops `Current` below the baseline.
- **`restockBaseline` definition pinned at implementation.** Intent: drain only
  stock pushed *above* what the shop naturally carries, without unwinding
  legitimate restocking. Candidates: the entry's `RestockQty`, or a fraction of
  `EffectiveMaxStock`. Provenance (NPC-dumped vs. natural) isn't tracked, so this
  is an approximation; the plan picks one and the smoke validates natural restock
  isn't fought.
- New balance knob `ShopOverstockDecayRounds` (conservative default, e.g.
  several in-game days). `is_component` exclusion is hard.

### Component 4 — chest backfill

- New `internal/forager/chest_backfill.go`: `BackfillVendorFromChests(vendor,
  room, shopInv)`, called from the vendor restock path. Builds the zone's
  aggregate chest pool (walk foragers' `StorageChestRoom` chests in the vendor's
  zone), ranks the vendor's own stock entries by gap, and for each gap pulls
  bucket-eligible items from the pool up to `MaxStock`. Free handoff:
  `entry.Current++`, remove from chest, `SaveShop` + chest persist on mutation.
  Mirrors `SellToVendor`'s persistence/throughput bookkeeping.
- **Back-pressure (`StateResting`):** new forager state. Entered when a deposit
  finds the chest at 100% (or a forager would deposit into a full chest). While
  `Resting`, the forager doesn't gather/deliver. A per-tick guard checks chest
  fill; once ≤ 90%, transition back to `StateForaging`. The backfill is the
  drain that makes resting terminate.

### Config knobs (`Balance`)

| Knob | Default | Purpose |
|------|---------|---------|
| `MobSurplusMinValue` | 10 | Minimum vendor `Value` for a surplus item to justify a vendor trip. |
| `ShopOverstockDecayRounds` | (conservative, ~several in-game days) | How slowly non-material overstock drains. |

## Data flow

**Planner sale (combat looter / gear-seeker):**
```
goal selection → sell-surplus|wealth-gold|upgrade-gear planner
  → actions.Sell(mobActor, ...)  [direct call for sell-surplus; reads result]
  → per item: EvaluateBuyRules → credit mob gold (shop gold untouched)
            → AddStockAtRound → SaveShop
  → rejected item → blacklist (anti-thrash)
```

**Forager / chest supply (free):**
```
forager delivery arrival → SellToVendor (satchel → vendor stock, no gold)
vendor restock tick → BackfillVendorFromChests (aggregate chests → neediest gaps)
chest hits 100% → forager StateResting → (backfill drains to ≤90%) → StateForaging
```

**Decay (clearance):**
```
restock tick → overstock decay: non-component entries above baseline shrink
               slowly after ShopOverstockDecayRounds
```

## Reconciliation with the 06-01 approved spec (review these)

Today's re-brainstorm chose to "adopt the expanded design." While rewriting I
folded in the 06-01 mechanisms the 06-02 brainstorm hadn't covered. Each is a
deliberate call you should confirm or veto:

1. **Kept combat-looter targeting** (06-01) in addition to today's crafter
   targeting — `sell-surplus` seeds on both. Reasoning: combat looters are the
   actual present-day surplus source; crafters are future-facing. If you want
   *only* crafters, drop the five combat archetypes from decision 4.
2. **Kept anti-thrash blacklist + direct `actions.Sell` call** (06-01). Today's
   brainstorm used an emitted `sell all` command with no reject signal; the
   direct call is strictly better and prevents junk-hauling loops.
3. **Kept keep-unsellable / no-drop disposal + chunk 5.5 donation bins** (06-01).
4. **Changed gold model** to shops-never-broke (06-02, decision 2) — opposite of
   06-01's shop-gold deduction.
5. **Added time-based decay** (06-02, decision 6) — 06-01 said caps suffice.

## Testing

- **Unit:** `actions.Sell` — player parity (name required, shop gold drawn),
  mob path (`SellAllSellable` sweep, shop gold untouched, never `MerchantBroke`),
  quest-item rejection, no-merchant, skip-rejected-continue, `SellResult` fields.
  `sell-surplus` goal — perpetual predicate, floor vs active tiers, blacklisted
  excluded, `min_value`. `sell-surplus` planner — nil→Failure, no-surplus→idle,
  surplus+not-at-vendor→pathto, blacklist-on-rejection, key prefixes. Overstock
  decay — component excluded, baseline floor, elapsed-rounds gate. Chest backfill
  — aggregate assembly, neediest-gap ordering, `MaxStock` cap, bucket
  eligibility, all-topped→stays-in-chest. Back-pressure — enter at 100%, exit
  at ≤90%.
- **Boot smoke:** `go build`, full package tests, server boots past data-file
  load without panic (archetype `default_goals` parse, goal-type + mob `sell`
  registration).
- **In-game smoke (deferred to user, per 2.8/2.9/2.10/5.3 precedent):** combat
  looter sells displaced gear and buys/funds with proceeds; mob holding only
  unsellable junk makes ≤1 trip then quiets (no thrash); `wealth-gold` mobs now
  actually sell; forager chest fills → forager rests → player buys from vendor →
  backfill drains chest → forager resumes; overstock junk drains while material
  stock holds.

## File touch list (anticipated; finalized in the plan)

- **New:** `internal/actions/sell.go` (+ test) — the lift
- **Modify:** `internal/usercommands/sell.go` — thin to a wrapper (+ keep test)
- **New:** `internal/mobcommands/sell.go` (+ register in `mobcommands.go`)
- **New:** `internal/goals/catalog/sell_surplus.go` (+ test)
- **New:** `internal/planners/sell_surplus.go` (+ test)
- **New:** `internal/shops/overstock_decay.go` (+ test)
- **New:** `internal/forager/chest_backfill.go` (+ test); `StateResting` in the
  forager state machine + transition guards
- **Modify:** vendor restock path — call `BackfillVendorFromChests` +
  overstock decay
- **Config:** `internal/configs/config.balance.go` (+ `.misc.go`) +
  `_datafiles/config.yaml` — `MobSurplusMinValue`, `ShopOverstockDecayRounds`
- **Archetype YAML:** `default_goals` on `generic_fighter`, `leader`,
  `predator`, `thief`, `guard_captain`, `crafter`
- **Context docs:** `internal/actions/context.md`, `internal/shops/context.md`,
  `internal/planners/context.md`, `internal/goals/catalog/context.md`,
  `internal/forager/context.md`
- **Roadmap:** flip 5.4 status on completion; add chunk 5.5 (town donation bins)

## Out of scope (explicit)

- Player↔NPC barter beyond existing shop UX.
- Coordinated zone-sweep distribution (per-vendor-tick chosen instead).
- Paying foragers gold for supply handoffs (free model retained).
- Deducting shop gold on mob sales (deferred; revisit if the gold leak matters).
- Cross-zone selling / backfill (same-zone only, matches 5.3).
- Donation bin / disposal sink — chunk 5.5; until then unsellable junk is kept.
- Ground-drop or destroy disposal — rejected (respects parked loot-drop redesign).
- Crafter → shop buyability investigation
  ([[project_crafted_items_buyability_investigation]]) — separate playtest.
- Instance-zone affix-scaled loot sellability
  ([[project_instance_zone_items_sellable_scaled]]) — separate chunk.
