# Mob Aliveness 5.4 — NPC Market Participation (Design)

**Status:** Approved — ready for implementation plan
**Date:** 2026-06-01
**Roadmap chunk:** 5.4 (Phase 5 — Cross-cutting features) • **Size:** L
**Depends on:** 5.3 (equip-aware shopping plumbing), 2.1 (`actions.Buy` lift pattern), 4.4 (planners + `try_goal_planner`)

---

## Goal

Looting NPCs put goods back into the economy: a mob that picks up gear it won't
use carries the surplus to a town shop and sells it, so shop stock reflects NPC
activity, not just player drop-offs. A bandit who upgrades from the sword you
dropped and offloads its old weapon in town is the target story.

This is **two parts**: (A) lift `actions.Sell` (the missing mob `sell` verb —
the foundation, which also un-breaks the existing `wealth-gold` goal and 5.3's
save-up funding), and (B) a new **surplus-offload drive** (goal + planner) with
vendor-rejection anti-thrash.

## Scope decisions (locked during brainstorming)

1. **Both NPC→shop flows were considered; the crafter→shop flow is PARKED.** A
   code investigation found the crafter→shop pipeline appears to work end-to-end
   (no display filter, no reload reconciliation drop), but the user's in-game
   impression that crafted items often don't appear purchasable is **not
   refuted** — open suspects are ceiling-pricing of new crafted entries and
   craft-decision frequency (see `project_crafted_items_buyability_investigation`).
   Deferred to a user playtest. 5.4 does **not** rebuild and does **not** claim
   crafter→shop "done."
2. **`sell-surplus` is a DISTINCT drive from `wealth-gold`.** `wealth-gold`
   chases a gold *target* and stops; `sell-surplus` clears *inventory clutter*
   (heavy unused gear) regardless of gold. Different motivations, cleaner
   planners. Kept separate per user call.
3. **Perpetual-with-floor goal model**, consistent with 5.3's `upgrade-gear`.
4. **Disposal of truly-unsellable junk (interim):** the mob **keeps it and stops
   trying** (no ground-drop — respects the parked loot-drop-redesign). The
   donation-bin disposal sink is **chunk 5.5**, which will retrofit the
   surplus-offload terminal action to donate instead of keep.

## Background — what already exists (reused, not rebuilt)

- **Shop-buys-from-seller is fully wired.** `shops.EvaluateBuyRules(item,
  shopInv, crafterSkill, buysGeneral, cfg, wornItems) BuyOffer` already decides
  whether a shop buys an item and at what price, and already **rejects** the
  cases the user wants rejected: quest items, items with no `VendorCategories`,
  categories the vendor doesn't accept, declining potions, and anything already
  at the shop's `MaxStock` overstock cap (returns a zero-price offer).
- **Seller payout already scales down as stock fills.** The seller's price is
  `shops.CalcBuyPrice`, which runs the same `ScarcityMultiplier` as buy pricing
  — as the shop's `Current` for an item rises toward abundance, the multiplier
  falls toward `PriceFloor`. Offloading into a well-stocked shop pays
  progressively less, automatically.
- **The player `Sell`** (`internal/usercommands/sell.go`, ~400 lines,
  `*users.UserRecord`-bound) already drives all of the above: `trySellOne`
  (find item → `EvaluateBuyRules` → gold transfer → `AddStockAtRound` /
  `BuysCount++` / `SaveShop`, plus the `gear_upgrade`-wear branch where the
  shopkeeper wears a bought upgrade), merchant resolution
  (`room.GetMobs(FindMerchant)` + a buy-probe), quantity parsing
  (`sell N`/`sell all`/`all.name`), and confirmation messaging.
- **Mobs accumulate sellable surplus.** `internal/hooks/mob_equip_best_floor_item.go`
  (`EquipBestFloorItem`) picks up the best floor item and `StoreItem`s it; the
  5.3 `gearup` step displaces the old slot occupant into the backpack. Plus
  steal/give intake. So a mob that upgrades is left holding the old piece — the
  surplus this drive offloads.
- **`actions.Buy` (2.1)** is the actor-pattern template for the `actions.Sell`
  lift. Planner helpers `findShopInZoneBuying`, `mobInVendorRoom`,
  `mobHasSellableItems` already exist (chunk 4.4 `wealth-gold`).

---

## Architecture

### Part A — Lift `actions.Sell`

Move the seller-side logic out of `usercommands/sell.go` into the shared
`actions` package, abstracting the **seller** via the existing `actions.Actor`
interface. The **merchant** side (a separate shopkeeper mob) is unchanged.

```go
// internal/actions/sell.go
type SellOptions struct {
    ItemName string
    Quantity int // 1, N, or unlimited sentinel (math.MaxInt) for "sell all"
}

type SellStopReason int
const (
    SellStopSoldAll     SellStopReason = iota // ran out of matching items (normal)
    SellStopNoItem                            // seller never had the item
    SellStopNoMerchant                        // no willing merchant in room
    SellStopMerchantBroke                     // merchant ran out of gold
    SellStopRejected                          // merchant declined the item type
)

type SellResult struct {
    Sold      int            // count actually sold
    TotalGold int            // gold received
    Reason    SellStopReason // why the loop stopped
    LastItemName string      // display name (for messaging)
}

func Sell(seller Actor, opts SellOptions) SellResult
```

- **Seller-side calls abstracted behind `Actor`:** inventory find
  (backpack → potions → components), `RemoveItem`, `Gold +=`,
  `OnSkillUse(bartering)`, and feedback (`SendText`). The bartering *bonus*
  (`shops.ApplyBarterBuyBonus`) uses the seller's bartering skill via
  `seller.GetCharacter().GetSkillLevel(skills.Bartering)` — works symmetrically
  for player and mob.
- **Player-only side-effects gate on `seller.IsPlayer()`:** `EventLog.Add`,
  `events.AddToQueue(ItemOwnership / EquipmentChange{UserId})`, and the
  player-facing confirmation/early-stop text. `MobActor.SendText` is already a
  no-op, so merchant-refusal lines (spoken by the merchant via `actions.Say`)
  still broadcast to the room while the mob seller silently reads the result.
- **Merchant side untouched:** `EvaluateBuyRules`, shop gold deduction,
  `AddStockAtRound`, `BuysCount++`, `SaveShop`, the `gear_upgrade`-wear branch,
  and the legacy `mob.Character.Shop` path all move verbatim.
- **Wrappers:** `usercommands/sell.go` collapses to a thin wrapper calling
  `actions.Sell` + rendering player messages from `SellResult`. New
  `mobcommands/sell.go` wrapper registered in the `mobCommands` map.
- **Quest notification** (`command:sell`, if any) gates on `seller.IsPlayer()`.

This is the larger, riskier half (a big player-coupled function). The
implementation plan must enumerate exactly which seller-side calls move behind
the `Actor` interface and which stay player-gated.

### Part B — Surplus-offload drive

Structurally a twin of 5.3's `upgrade-gear` (perpetual goal + planner).

**Goal type `sell-surplus`** (`internal/goals/catalog/sell_surplus.go`):
- `Predicate` always false (perpetual drive).
- `ContextScore`: floor `1.0` always (so 4.6 dormancy never abandons this
  standing default), rising to `2.5` when the mob is idle/out-of-combat **and**
  carries **actionable surplus** (see definition). Cheap and self-contained —
  no shop scan in scoring (the planner owns vendor decisions), per the 5.3
  precedent.
- Optional `min_value` int param overriding the config floor.

**"Actionable surplus"** = a non-equipped backpack item that:
- has vendor `Value > 0` and `Value >= MobSurplusMinValue` (don't trek to town
  for trivial junk),
- is not a quest item (`QuestToken == ""`),
- is not a component-bag material the mob may use,
- is not on the planner's per-mob **unsellable blacklist** (see anti-thrash).

Equip-best / `upgrade-gear` run first and claim upgrades (displacing the old
piece into the backpack); what remains is what `sell-surplus` offloads.

**Planner `sell-surplus`** (`internal/planners/sell_surplus.go`):
- No actionable surplus → idle (`Running`, empty command).
- Has actionable surplus, **not** at a buying vendor → sticky-resolve
  `findShopInZoneBuying` (`plan:sell-surplus:sell_shop_room`) + `pathto`.
- Has actionable surplus, **at** a buying vendor → sell the highest-value
  actionable surplus item by **calling `actions.Sell` directly** (not emitting a
  `sell` command string) so the planner can read the `SellResult`:
  - `SellStopRejected` (or zero sold) → **blacklist** that item id in
    `plan:sell-surplus:unsellable` so it's excluded from future surplus and the
    mob stops re-offering it.
  - sold → progress; next tick re-evaluates.
  - `SellStopNoMerchant` → clear the sticky shop room and idle/re-resolve.

  *Rationale for the direct call:* the fire-and-forget command model (used by
  5.3) gives the planner no rejection signal, which is exactly what anti-thrash
  needs. `actions.Sell` is the actor-pattern entry point and is safe to call
  from the planner. (This is the one intentional divergence from the 5.3
  emit-command pattern; documented here and in `planners/context.md`.)

**Anti-thrash:** rejected items are blacklisted → excluded from the surplus
count → once every remaining surplus item is blacklisted, `ContextScore` falls
to the floor and the mob goes quiescent. Worst case is one wasted trip to a
vendor that won't buy, then quiet. The blacklist is `plan:`-prefixed plan-state,
wiped by `ClearPlanState` on goal switch (so the mob occasionally re-tries — an
acceptable, self-healing reset).

**Archetypes:** seed `sell-surplus` as a low-priority `default_goals` entry on
the looting combat archetypes that run `EquipBestFloorItem` and thus accumulate
displaced gear: `generic_fighter`, `leader`, `predator`, `thief`,
`guard_captain`. No btree edits (`try_goal_planner` already present).

### Config knobs (`Balance`)

| Knob | Default | Purpose |
|------|---------|---------|
| `MobSurplusMinValue` | 10 | Minimum vendor `Value` for a surplus item to be worth a trip to a vendor (skip trivial junk). |

(Reserve/min-delta style knobs from 5.3 don't apply; selling has no reserve.)

### Economic integration (mostly already covered)

- **No bloat:** `EvaluateBuyRules` rejects intake at the shop's `MaxStock` cap;
  the shop gold reserve bounds spend. "Decay/clearance" from the roadmap is
  therefore **already satisfied** by overstock caps — documented, not rebuilt.
  (Active time-based clearance, if ever wanted, belongs with the 5.5 bin's
  expiry sweep, not here.)
- **5.3 ↔ 5.4 loop:** mobs buy from shops (5.3) and sell to shops (5.4),
  bounded by gold/stock caps. A mob won't re-buy gear it just displaced (its
  `itemvalue` delta is ≤ 0). Emergent and safe — noted, not guarded.

---

## Out of scope (explicit)

- **Crafter → shop flow** — parked for a user playtest; not rebuilt, not claimed
  done (`project_crafted_items_buyability_investigation`).
- **Donation bin / disposal sink** — chunk 5.5. Until then, truly-unsellable
  junk is kept by the mob (no ground-drop, no destroy).
- **Ground-drop or destroy disposal** — explicitly rejected (respects the parked
  loot-drop-redesign).
- **Player↔NPC barter beyond what shop UX already supports.**
- **Crafter-output pricing/normalizer fixes** — tracked separately
  (`project_store_restock_considered_fix`, the crafted-items investigation).

---

## Testing

**Unit tests:**
- `actions.Sell`: seller-abstraction branches — player vs mob feedback gating;
  no-item, no-merchant, merchant-broke, rejected, and successful single + multi
  (`sell N` / `sell all`) outcomes; quest-item refusal; the `SellResult` fields
  populate correctly. (Live shop/item data isn't loaded under `go test`, so
  deep price/stock paths are covered by the in-game smoke — same constraint as
  2.1/5.3.)
- `sell-surplus` goal: `Predicate` always false; `ContextScore` floor vs active
  tiers (idle + actionable surplus vs broke/in-combat); blacklisted items
  excluded from surplus; `min_value` honored.
- `sell-surplus` planner: nil → Failure; no surplus → idle; surplus + not at
  vendor → pathto; the blacklist-on-rejection anti-thrash path; key prefixes.

**Boot smoke:** clean server start (archetype `default_goals` parse, goal-type
registration, mob `sell` command registration).

**In-game smoke (deferred to user, per precedent):** give a `thief`/`generic_fighter`
mob a sellable surplus item + place it in a zone with a buying vendor; observe
path → sell → stock/gold change. Verify a mob holding only unsellable junk makes
at most one trip then goes quiet (no thrash). Confirm `wealth-gold` mobs now
actually sell (the verb no longer no-ops).

---

## File touch list (anticipated; finalized in the plan)

- **New:** `internal/actions/sell.go` (+ test) — the lift
- **Modify:** `internal/usercommands/sell.go` — thin to a wrapper (+ keep its test)
- **New:** `internal/mobcommands/sell.go` (+ register in `mobcommands.go`)
- **New:** `internal/goals/catalog/sell_surplus.go` (+ test)
- **New:** `internal/planners/sell_surplus.go` (+ test)
- **Config:** `internal/configs/config.balance.go` + `config.balance.misc.go` +
  `_datafiles/config.yaml` — `MobSurplusMinValue`
- **Archetype YAML:** `default_goals` additions on `generic_fighter`, `leader`,
  `predator`, `thief`, `guard_captain`
- **Context docs:** `internal/actions/context.md` (or divergences),
  `internal/planners/context.md`, `internal/goals/catalog/context.md`
- **Roadmap:** flip 5.4 status; add 5.5 (town donation bins) chunk entry
