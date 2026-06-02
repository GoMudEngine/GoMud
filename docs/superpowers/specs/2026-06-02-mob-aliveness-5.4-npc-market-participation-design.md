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
(the prior approved spec). This version keeps that spec's verb-lift detail,
*drops* its proactive surplus-offload goal (deferred — see "Deferred"), and
*adds* the chest backfill, time-based overstock decay, and the shops-never-broke
gold model agreed in the 2026-06-02 re-brainstorm.

---

## Goal

NPCs put goods back into the economy through normal shop channels, so shop
stock reflects NPC activity, not just player drop-offs. Three concrete loops:

- **Gold-seeking mobs sell loot** — the existing `wealth-gold` (thief) and
  `upgrade-gear` (5.3) planners already try to `sell all`, but it is a no-op
  today (no `sell` verb). Lifting the verb makes those loops real: a thief
  fences stolen goods, a gear-seeker funds an upgrade.
- **Foragers supply, and their chest "overflow cache" drains back** into vendor
  stock instead of being a one-way sink.
- **Vendors slowly clear non-material junk** nobody buys, so shelves don't
  ossify.

## Why now

Three threads converge:

1. **The `sell` verb gap** ([[project_mob_sell_command_missing]]). No
   `actions.Sell`, no mob `sell` command. `wealth-gold` (4.4) and `upgrade-gear`
   (5.3) both issue `sell all`, which falls through mob dispatch to `emote looks
   a little confused (sell all)` — a visible no-op. Their sell-to-fund path is
   dead until the verb exists.
2. **The forager chest is a one-way drain**
   ([[project_vendor_backfill_from_forager_chests]]). Chunk 2.10-followups added
   a "deposit unsold goods" chest as an *overflow cache* whose goods were meant
   to flow back to vendors — but no backfill was ever built.
3. **5.4 itself** — the roadmap's last non-polish aliveness chunk — *is* "NPCs
   sell through shops." It cannot function without (1) and is the natural home
   for (2).

## Scope note (M → L)

The roadmap sizes 5.4 at M. This design is L: verb lift **+** chest backfill
(aggregate/demand-routed with back-pressure) **+** time-based overstock decay.
A proactive surplus-offload goal was considered and **dropped** (see Deferred).

## Deferred — proactive surplus-offload goal

The 06-01 spec proposed a `sell-surplus` goal driving combat looters to carry
displaced gear to town and sell it. The 06-02 re-brainstorm **dropped it**, for
a concrete reason: a combat mob's sellable surplus is too thin today to justify
a dedicated goal.

- Mobs **do** drop loot to the floor on death (`Death_MobLoot.go`), and idle
  mobs **do** pick up floor upgrades (`EquipBestFloorItem`), displacing the old
  piece into the backpack. So surplus exists in principle.
- **But mobs never inherit player gear** (players don't drop on death). The only
  live sources are NPC-vs-NPC kills (sporadic — most combat is mob-vs-player),
  bought-upgrade displacement (5.3), theft (already handled by `wealth-gold` on
  thieves), and the occasional player `give`.

The behavior earns its keep once there is a **richer, mob-reachable surplus
stream** — specifically **mobs salvaging corpses with an expanded salvage-yield
table** (mob salvage core exists from 2.9; the *yield* is what's thin). Revisit
the proactive offload goal then. Until then, near-term mob selling is fully
served by the existing `wealth-gold` / `upgrade-gear` planners that the verb
lift un-breaks.

(The 06-01 spec's donation-bin disposal sink, "chunk 5.5," was the disposal end
of that dropped goal. It is deferred alongside it — no unsellable-surplus
accumulation to dispose of without the proactive goal.)

## Background — what already exists (reused, not rebuilt)

- **Shop-buys-from-seller is fully wired.** `shops.EvaluateBuyRules(item,
  shopInv, crafterSkill, buysGeneral, cfg, wornItems) BuyOffer` decides whether
  a shop buys an item and at what price, and already **rejects** quest items,
  items with no `VendorCategories`, unaccepted categories, declining potions,
  and anything at the `MaxStock` overstock cap (zero-price offer).
- **Seller payout already diminishes as stock fills** — the same
  `ScarcityMultiplier` as buy pricing. Offloading into a well-stocked shop pays
  progressively less, automatically. (No work; just noted.)
- **`usercommands/sell.go`** — the ~400-line player sell command, `UserRecord`-
  bound: `trySellOne` (find → `EvaluateBuyRules` → gold transfer →
  `AddStockAtRound` / `BuysCount++` / `SaveShop`, plus the `gear_upgrade`-wear
  branch), merchant resolution (`room.GetMobs(FindMerchant)` + buy-probe),
  quantity parsing (`sell N` / `sell all` / `all.name`), confirmation messaging.
  This is the lift surface.
- **`forager.SellToVendor`** (`internal/forager/vendor_sell.go`) — a **free
  stock handoff**, NOT a sale: bucket-matching satchel items move into matching
  vendor stock entries with **no gold transfer**, gated on `entry.Current <
  entry.MaxStock`. Fires from `ForagerArrivalListener` on delivery-patrol
  arrival. Stays as the forager/chest supply path; the chest backfill extends it.
- **`actions.Buy` (2.1)** is the actor-pattern template for the lift. Planner
  helpers `findShopInZoneBuying`, `mobInVendorRoom`, `mobHasSellableItems` exist
  (4.4 `wealth-gold`).
- **forager state machine** (2.9) — `StateForaging` / `StateDelivering` /
  `StateStoring` / `StateRecalling`; `StorageChestRoom`; chest-deposit
  primitive. The new `StateResting` back-pressure state slots in here.
- **`economy.BucketFor(itemId)`** — maps items to economic buckets; foragers
  carry `Buckets`. Decides which vendor wants which goods.

## Design decisions (locked during brainstorming)

1. **Two economic models, made explicit.**
   - **Sale (gold transfer):** `actions.Sell`. Seller credited; item enters
     vendor stock. Used by planner-driven sellers (`wealth-gold`, `upgrade-gear`
     funding) and players (unchanged).
   - **Supply handoff (no gold):** `forager.SellToVendor` and the chest backfill.
     Supplier feeds vendor stock for free — the forager works *for* the vendor
     economy.
   - Documented in `internal/shops/context.md` and the forager context.

2. **Shops do not run out of gold from NPC selling.** When a **mob** sells via
   `actions.Sell`, the seller is credited but the shop's gold reserve is **not
   deducted**, and the merchant-gold check is skipped (mob sales never hit
   `MerchantBroke`). Keeps shop liquidity intact for *player* sells. A small,
   deliberate gold-creation leak, acceptable for now and trivially reversible
   (gates on `seller.IsPlayer()`). Player sells unchanged — they still draw down
   shop gold. *(Opposite of the 06-01 spec, which deducted shop gold;
   superseded.)*

3. **`sell all` (no item name) is a mob inventory-sweep mode.** The player path
   keeps requiring an item name. `SellOptions` gains `SellAllSellable bool`; in
   that mode the seller offloads every sellable (non-quest, `Value > 0`,
   non-component-the-mob-uses) inventory item to any willing merchant in the
   room, skipping rejected items rather than aborting. Bare `sell all` from a
   mob maps to this; bare `sell all` from a player keeps its current rejection.

4. **Overstock decay excludes crafting materials.** New time-based clearance
   shrinks unsold vendor stock held *above* a restock baseline so abandoned junk
   drains out. Items flagged `is_component: true` are **never decayed** —
   topped-up material stock is fine and stays. The `MaxStock` cap prevents
   *more* pileup; this decay removes *existing* abandoned stock. New knob
   `ShopOverstockDecayRounds` (conservative default). *(The 06-01 spec treated
   caps as sufficient; superseded.)*

5. **Chest backfill = aggregate + demand-routed + back-pressure.**
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
     demand returns. Intended, not a stall bug.
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
  `OnSkillUse(bartering)`, feedback (`SendText`). The barter *bonus* reads
  `seller.GetCharacter().GetSkillLevel(skills.Bartering)` — symmetric.
- **Player-only side-effects gate on `seller.IsPlayer()`:** `EventLog.Add`,
  `events.AddToQueue(ItemOwnership / EquipmentChange{UserId})`, player-facing
  confirmation/early-stop text, `command:sell` quest notification.
  `MobActor.SendText` is already a no-op; merchant-refusal lines (spoken via
  `actions.Say`) still broadcast while the mob seller reads the result silently.
- **Gold model branch (decision 2):** credit the seller always; deduct shop gold
  and run the merchant-gold check **only when `seller.IsPlayer()`**.
- **`SellAllSellable` (decision 3):** iterate inventory, offer each sellable item
  to a willing merchant; skip rejected items.
- **Merchant side untouched:** `EvaluateBuyRules`, `AddStockAtRound`,
  `BuysCount++`, `SaveShop`, the `gear_upgrade`-wear branch, the legacy
  `mob.Character.Shop` path move verbatim.
- **Wrappers:** `usercommands/sell.go` collapses to parse + delegate + render
  from `SellResult`. New `mobcommands/sell.go`, registered in `mobCommands`.
  The existing `wealth-gold` / `upgrade-gear` planners emit `sell all`, which now
  resolves through `mobcommands.sell → actions.Sell(SellAllSellable)`.
- Cross-reference `forager.SellToVendor` from the `actions.Sell` doc comment so
  the sale-vs-supply split is discoverable.

This is the larger, riskier half. The plan must enumerate exactly which
seller-side calls move behind `Actor` and which stay `IsPlayer()`-gated.

> **Pre-existing thrash note (out of scope):** `wealth-gold` checks
> `mobHasSellableItems` (`Value > 0`, non-quest) but a vendor may still reject an
> item (wrong category / overstock). Without the dropped `sell-surplus` planner's
> blacklist, a `wealth-gold` mob can re-path to a vendor that won't buy. This is
> pre-existing `wealth-gold` behavior, not introduced here; `SellAllSellable`'s
> skip-rejected-continue keeps a single trip from looping forever, but the mob may
> retry across ticks. Left as a known limitation; the blacklist returns with the
> deferred proactive goal.

### Component 2 — overstock decay

- New `internal/shops/overstock_decay.go`: on a vendor's restock tick, for each
  stock entry where `Current > restockBaseline` and the item is **not**
  `is_component`, decrement `Current` by a small amount once
  `ShopOverstockDecayRounds` have elapsed since the entry last grew. Decay never
  drops `Current` below the baseline.
- **`restockBaseline` definition pinned at implementation.** Intent: drain only
  stock pushed *above* what the shop naturally carries, without unwinding
  legitimate restocking. Candidates: the entry's `RestockQty`, or a fraction of
  `EffectiveMaxStock`. Provenance isn't tracked, so this is an approximation; the
  plan picks one and the smoke validates natural restock isn't fought.
- New balance knob `ShopOverstockDecayRounds` (conservative default, e.g.
  several in-game days). `is_component` exclusion is hard.

### Component 3 — chest backfill

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
| `ShopOverstockDecayRounds` | (conservative, ~several in-game days) | How slowly non-material overstock drains. |

## Data flow

**Planner sale (thief / gear-seeker — existing planners, now un-broken):**
```
goal selection → wealth-gold | upgrade-gear planner → "sell all"
  → mobcommands.sell → actions.Sell(mobActor, SellAllSellable)
  → per item: EvaluateBuyRules → credit mob gold (shop gold untouched)
            → AddStockAtRound → SaveShop
  → mob now has gold → buy / gearup proceeds
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

## Reconciliation with the 06-01 approved spec

Today's re-brainstorm chose the expanded design. The deltas from 06-01:

1. **Dropped the proactive `sell-surplus` goal** (and its combat-looter
   targeting, anti-thrash blacklist, and chunk-5.5 donation bins) — deferred
   until mob corpse-salvage yields a richer surplus stream. See "Deferred."
2. **Changed the gold model** to shops-never-broke (decision 2) — opposite of
   06-01's shop-gold deduction.
3. **Added time-based decay** (decision 4) — 06-01 said caps suffice.
4. **Added the chest backfill** (component 3) — out of scope in 06-01.
5. **Kept** 06-01's `actions.Sell` lift detail (actor abstraction,
   `IsPlayer()`-gated side-effects) and its sale-vs-supply observation.

## Testing

- **Unit:** `actions.Sell` — player parity (name required, shop gold drawn),
  mob path (`SellAllSellable` sweep, shop gold untouched, never `MerchantBroke`),
  quest-item rejection, no-merchant, skip-rejected-continue, `SellResult` fields.
  Overstock decay — component excluded, baseline floor, elapsed-rounds gate.
  Chest backfill — aggregate assembly, neediest-gap ordering, `MaxStock` cap,
  bucket eligibility, all-topped→stays-in-chest. Back-pressure — enter at 100%,
  exit at ≤90%.
- **Boot smoke:** `go build`, full package tests, server boots past data-file
  load without panic (mob `sell` registration).
- **In-game smoke (deferred to user, per 2.8/2.9/2.10/5.3 precedent):**
  `wealth-gold` thief now actually sells stolen goods (no more confused emote)
  and an `upgrade-gear` mob funds a purchase by selling; forager chest fills →
  forager rests → player buys from vendor → backfill drains chest → forager
  resumes; overstock junk drains while material stock holds.

## File touch list (anticipated; finalized in the plan)

- **New:** `internal/actions/sell.go` (+ test) — the lift
- **Modify:** `internal/usercommands/sell.go` — thin to a wrapper (+ keep test)
- **New:** `internal/mobcommands/sell.go` (+ register in `mobcommands.go`)
- **New:** `internal/shops/overstock_decay.go` (+ test)
- **New:** `internal/forager/chest_backfill.go` (+ test); `StateResting` in the
  forager state machine + transition guards
- **Modify:** vendor restock path — call `BackfillVendorFromChests` +
  overstock decay
- **Config:** `internal/configs/config.balance.go` (+ `.misc.go`) +
  `_datafiles/config.yaml` — `ShopOverstockDecayRounds`
- **Context docs:** `internal/actions/context.md`, `internal/shops/context.md`,
  `internal/forager/context.md`
- **Roadmap:** flip 5.4 status on completion; log the deferred proactive-offload
  goal (gated on mob corpse-salvage yield expansion)

## Out of scope (explicit)

- Proactive surplus-offload goal — deferred (see "Deferred").
- Donation bin / disposal sink (old chunk 5.5) — deferred with the proactive
  goal.
- Player↔NPC barter beyond existing shop UX.
- Coordinated zone-sweep distribution (per-vendor-tick chosen instead).
- Paying foragers gold for supply handoffs (free model retained).
- Deducting shop gold on mob sales (deferred; revisit if the gold leak matters).
- Cross-zone selling / backfill (same-zone only, matches 5.3).
- Crafter → shop buyability investigation
  ([[project_crafted_items_buyability_investigation]]) — separate playtest.
- Instance-zone affix-scaled loot sellability
  ([[project_instance_zone_items_sellable_scaled]]) — separate chunk.
