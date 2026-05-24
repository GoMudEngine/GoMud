# Mob Aliveness 2.1 — Mob `buy` Command

> **Phase 2 tactical (first chunk).** Adds a `buy` verb on the mob
> side, consolidated through `internal/actions/` so player `buy`
> and mob `buy` share one implementation. Verb-only — decision
> logic for *what* to buy lives in tactical/strategic layers and
> is out of scope here.

## Goal

Give NPCs a tactical primitive for purchasing from shops. The verb
must support disambiguation (multiple merchants in a room), gold
checks, and carry capacity. Strategic-layer goals such as "save up
for armor" become expressible only when this verb exists, but the
verb is permissive and dumb — it executes a single purchase, full
stop. Consumers (behavior trees, planners) decide whether and what
to buy.

The chunk's job is **verb plus consolidation** — extract the
existing player `buy` implementation into the shared `actions`
package, abstract it over the `Actor` interface, add a thin mob
wrapper, and use the refactor to close one pre-existing gap on the
player side (carry-capacity enforcement at purchase).

## Architectural musts

The chunk brief lists "mobcommand `buy`, integration with existing
shop pricing/stock, restocking interaction with NPC-buyer behavior"
as in-scope. Brainstorming locked in:

1. **Single consolidated implementation in `internal/actions/`.**
   Mirrors the established pattern for `combat_attack.go`,
   `cast.go`, `give.go`. Public entry point is
   `actions.Buy(buyer Actor, opts BuyOptions) BuyResult`. The
   existing internal seams of player `buy.go` (`tryPurchase`,
   `tryPurchaseFromInventory`, `validatePurchase`,
   `executePurchaseItem`, `executePurchaseBuff`,
   `effectiveRestock`, `sendMerchantMessage`) move with it,
   reshaped to take `Actor` instead of `*users.UserRecord`.

2. **Item + Buff only.** Merc and Pet sale paths are dropped from
   the consolidated core entirely. No current shop YAML sells
   either, so the cleanup is content-zero-impact. The
   `executePurchaseMerc` and `executePurchasePet` functions are
   deleted along with their dispatch branches in `tryPurchase`.
   Future merc hiring belongs in a separate chunk that designs
   proper paid-merc semantics (periodic upkeep, loyalty meter,
   walk-away-if-unpaid); the current "permanent charm on
   purchase" model is a pre-aliveness-era artifact we choose not
   to bless.

3. **Both shop backends supported symmetrically.** Mob buyers can
   purchase from either the legacy `Character.Shop` path or the
   newer `ShopInventory` path (dynamic pricing, persistence).
   Both backends already work for player buyers; consolidating on
   `Actor` makes them work for mob buyers with no per-backend
   special casing.

4. **Carry capacity gate, symmetric.** The pre-existing gap in
   player `buy` (no encumbrance check on purchase) is closed as a
   side-effect of the refactor. `validatePurchase` blocks the
   purchase **before** any side effects (destock, gold deduction,
   trade-item consume) when adding the new item's weight would
   push the buyer past `Character.CarryCapacity()`. Math:
   `carriedWeight + spec.Weight > capacity` → reject with
   `Reason="overburdened"`. Blocking pre-side-effect means no
   refund logic is needed. Applies only to item purchases —
   buffs do not occupy backpack space.

5. **Quest engine notification stays player-only.** The
   `command:buy` quest event fires inside `actions.Buy` gated by
   `buyer.IsPlayer()`. Mob buyers do not fire the event — mobs
   aren't quest subjects, and no current quest triggers on a
   nearby player witnessing a purchase.

6. **Verb is dumb.** No decision logic about what to buy lives in
   `actions.Buy`. No archetype filtering. No anti-recursion guards
   beyond what already exists (the in-place strip in the deleted
   `executePurchaseMerc` is going away with the merc path). A
   behavior tree that wires a bandit to `buy a kitten` will buy a
   kitten. Restrictions are content authoring concerns.

7. **Mob bartering progression falls out naturally.** Because
   `actions.Buy` calls `buyer.OnSkillUse("bartering")` symmetrically
   and `MobActor.OnSkillUse` routes through `Character.OnSkillUse`,
   mobs that run `buy` regularly will probabilistically rank up
   bartering and hit the discount tier on the same curve as
   players. No new mechanism, no gating. Balance tuning (whether
   the rate is right for mobs given how rarely they'll buy) is
   deferred to playtesting.

## Architecture & module layout

Three files in play.

**New: `internal/actions/buy.go`.** Holds the consolidated
implementation. Contents are lifted from
`internal/usercommands/buy.go` and reshaped to operate on `Actor`
instead of `*users.UserRecord`. Internal helpers
(`tryPurchase`, `tryPurchaseFromInventory`, `validatePurchase`,
`executePurchaseItem`, `executePurchaseBuff`, `effectiveRestock`,
`sendMerchantMessage`, `purchaseContext`) move here as unexported.
The merc/pet paths and their helpers (`executePurchaseMerc`,
`executePurchasePet`, the merc/pet branches in fuzzy-matching,
charm cap checks, pet replacement logic) are deleted.

**Modified: `internal/usercommands/buy.go`.** Collapses to a thin
wrapper of ~30 lines: parse `rest`, build a `UserActor`, call
`actions.Buy`. Internal helpers move out. Behavior preserved
end-to-end except for the new encumbrance gate and the loss of
merc/pet sale routes (which were unreachable in current content).

**New: `internal/mobcommands/buy.go`.** Thin wrapper of ~30 lines:
parse `rest`, build a `MobActor`, call `actions.Buy`. Registered
in `mobcommands.go` alongside the other verbs with
`AllowedWhenDowned: false`.

## Public API

```go
// BuyOptions controls how a purchase is attempted.
type BuyOptions struct {
    // Request is the raw "rest" string, e.g. "5 iron ingot from marko".
    // Quantity prefix and "from <merchant>" suffix are parsed inside
    // actions.Buy, so wrappers do raw passthrough.
    Request string

    // TargetMerchantUserId, when > 0, restricts merchant selection
    // to a specific user merchant. Wrappers may set this directly
    // for cases where the caller has already resolved the target.
    TargetMerchantUserId int

    // TargetMerchantMobInstanceId, when > 0, restricts merchant
    // selection to a specific mob merchant.
    TargetMerchantMobInstanceId int
}

// BuyResult is the outcome of an attempted purchase.
type BuyResult struct {
    Success   bool   // at least one unit purchased
    Purchased int    // actual units purchased (may be < Requested)
    Requested int    // requested quantity (1 if unspecified)
    SaleType  string // "item" | "buff" | "" on failure
    Reason    string // populated on failure
}

// Buy executes a purchase on behalf of buyer. Returns a BuyResult
// describing the outcome.
func Buy(buyer Actor, opts BuyOptions) BuyResult
```

`Reason` vocabulary on failure:

| Reason | Meaning |
|---|---|
| `no_request` | empty `Request` and no fallback path |
| `no_merchant` | no merchant in room (or `from <name>` resolved to a non-merchant) |
| `no_match` | merchant has stock but request didn't fuzzy-match any item |
| `out_of_stock` | matched item but stock depleted between resolution and validation |
| `insufficient_gold` | buyer can't afford the price |
| `missing_trade_item` | item requires a trade-in item the buyer doesn't have |
| `overburdened` | buyer is overburdened (item purchases only) |
| `self_target` | buyer tried to `buy ... from <themselves>` (player-side guard) |

`SaleType` is `"item"`, `"buff"`, or `""` (failure or no sale routed).

## End-to-end flow

`actions.Buy(buyer, opts)` runs the following sequence:

1. **Parse quantity & merchant target.** Strip a leading integer
   from `opts.Request` if present (e.g. `"5 iron ingot"` →
   `quantity=5, item="iron ingot"`). Strip a trailing
   `from <merchant>` clause via `util.SplitButRespectQuotes`. If
   the merchant name resolves through `actions.ResolveTargetActor`,
   set `TargetMerchantUserId` / `TargetMerchantMobInstanceId`
   accordingly. Self-targeting guard fires for player buyers
   only.

2. **Resolve merchant list.** If a specific target was set, the
   list contains only that merchant. Otherwise iterate
   `room.GetPlayers(rooms.FindMerchant)` then
   `room.GetMobs(rooms.FindMerchant)`. Empty list → return
   `Reason="no_merchant"` and short-circuit (with the
   buyer-facing "Visit a merchant..." message if
   `buyer.IsPlayer()`).

3. **Per merchant, attempt purchase.** For mob merchants, dispatch
   to `tryPurchaseFromInventory` if `shops.GetShopInventory(...)`
   returns non-nil; otherwise call `Restock()` on the legacy
   `Character.Shop` and dispatch to `tryPurchase`. Player
   merchants always use `tryPurchase`.

4. **Inside `tryPurchase` / `tryPurchaseFromInventory`: fuzzy
   match.** Build a name→ShopItem (or name→StockEntry) map of
   instock items. Apply `util.FindMatchIn` against the request.
   No match → merchant `Command("say ...")` rejection with a
   random alternative suggestion (mob merchants only; player
   merchants don't emote rejections). Return false → outer loop
   tries the next merchant.

5. **`validatePurchase` (new encumbrance gate first).** Order:
   - **Encumbrance check** for item sales: compute the new
     item's weight via `items.New(itemId).GetSpec().Weight`, and
     reject if
     `buyer.GetCharacter().GetCarriedWeight() + weight >
     buyer.GetCharacter().CarryCapacity()`. Return false with
     `Reason="overburdened"` and a buyer-facing message ("You
     can't carry any more." for players; silent for mobs). **No
     side effects yet.**
   - **Stock check:** `matchedShopItem.Available()` (legacy) or
     `entry.Current > 0` (ShopInventory).
   - **Gold check:** `buyer.GetCharacter().Gold >= price`.
   - **Trade item check** if `TradeItemId > 0`: buyer has the
     item in backpack.
   - All checks pass → **side effects**: destock from
     `Character.Shop.Destock` (legacy) or
     `shopInv.RemoveStockAtRound` (ShopInventory); deduct buyer
     gold; emit `events.EquipmentChange{GoldChange: -price}` for
     player buyers only; credit merchant gold (mob legacy
     merchant: `+1` per sale cheat preserved; player merchant or
     ShopInventory: full price); consume trade item.

6. **Execute by sale type.**
   - **Item:** `items.New(itemId)` →
     `events.ItemOwnership{Gained: true}` →
     `buyer.GetCharacter().StoreItem(newItm)` → buyer-facing
     "You purchase the X" text (via `buyer.SendText`) → room
     broadcast via `buyer.SendRoomText(..., excludeSelf=true)`.
   - **Buff:** `buyer.AddBuff(buffId, "shop")` → buyer-facing
     "You pay X" text → merchant "say" follow-up emote.

7. **Post-success bookkeeping.**
   - `buyer.OnSkillUse("bartering")` — works symmetrically.
   - Merchant mob: `shopMob.Character.OnStatUse("charisma", 0)`.
   - Quest engine `command:buy` event — **gated by
     `buyer.IsPlayer()`**, fires only for player buyers.

8. **Quantity loop.** If `quantity > 1` and the first unit
   succeeded, retry steps 3–7 against the same merchant until
   either `quantity` units are bought or a unit fails. On partial
   success, send the buyer a yellow "Purchased N of M before
   running short." line.

## Actor touchpoints

Every player-record reference in current `buy.go` collapses to one
of these Actor calls:

| Current code | New code |
|---|---|
| `user.Character` | `buyer.GetCharacter()` |
| `user.Character.Gold` | `buyer.GetCharacter().Gold` |
| `user.Character.StoreItem(...)` | `buyer.GetCharacter().StoreItem(...)` |
| `user.SendText(...)` | `buyer.SendText(...)` (no-op for mobs) |
| `room.SendTextVisual(..., user.UserId)` | `buyer.SendRoomText(..., excludeSelf=true)` |
| `user.AddBuff(buffId, "shop")` | `buyer.AddBuff(buffId, "shop")` |
| `user.Character.OnSkillUse(string(skills.Bartering), user.UserId)` | `buyer.OnSkillUse("bartering")` |
| `user.PlaySound("purchase", "other")` | gated by `if buyer.IsPlayer() { ... }` — mobs have no sound channel |
| `user.EventLog.Add(...)` | gated by `if buyer.IsPlayer() { ... }` — mobs have no event log |
| `events.EquipmentChange{UserId: user.UserId, ...}` | gated by `if buyer.IsPlayer() { ... }` — mobs have no prompt to update |
| quest engine `Notify(...)` | gated by `if buyer.IsPlayer() { ... }` |

Merchant-side branching (player merchant vs mob merchant) stays
as-is — `shopMob *mobs.Mob` and `shopUser *users.UserRecord` remain
the parameters of the lower-level execute helpers, since shop
state lives on those concrete types and there's no win in
abstracting the merchant the same way as the buyer.

## Wrappers

**`internal/usercommands/buy.go`** after consolidation:

```go
func Buy(rest string, user *users.UserRecord, room *rooms.Room,
    flags events.EventFlag) (bool, error) {

    if rest == "" {
        return List(rest, user, room, flags)
    }

    actor := &actions.UserActor{User: user, Room: room}
    actions.Buy(actor, actions.BuyOptions{Request: rest})
    return true, nil
}
```

All buyer-facing messages live inside `actions.Buy`; the wrapper
discards the `BuyResult` because no user-command caller currently
consumes it. (The result is still returned for future programmatic
callers and tests.)

**`internal/mobcommands/buy.go`** (new):

```go
func Buy(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
    if rest == "" {
        return true, nil // no-op for mobs; mobs don't read 'list'
    }
    actor := &actions.MobActor{Mob: mob, Room: room}
    actions.Buy(actor, actions.BuyOptions{Request: rest})
    return true, nil
}
```

Registered in `mobcommands.go`:

```go
"buy": {Buy, false}, // AllowedWhenDowned: false
```

## Testing

The test plan covers the existing player surface plus the new mob
surface plus the new gate.

**Existing player tests (must stay green).** `internal/usercommands/usercommands_test.go`
contains `TestBuy`, `TestBuyBranches`, `TestBuyEmptyArgs`,
`TestBuyNoMerchant`. After consolidation these drive the same
public command surface and should pass without modification. Any
tests that assert merc or pet purchase behavior are deleted along
with the dropped code paths. Any tests that drive a `buy` while
the character is already overburdened need a starting-gold or
capacity setup tweak — almost certainly none do today; audit
during implementation.

**New `internal/actions/buy_test.go`.** Drives the consolidated
core directly via both `UserActor` and `MobActor`:

| # | Case | Buyer | Merchant | Backend | Asserts |
|---|---|---|---|---|---|
| 1 | Item happy path | user | mob | legacy | gold debited, item in backpack, stock down, merchant +1g |
| 2 | Item happy path | user | mob | ShopInventory | gold debited, item in backpack, `shopInv.Gold` += full price |
| 3 | Item mob buyer | mob | mob | legacy | mob's gold debited, item in mob's backpack |
| 4 | Item mob buyer | mob | mob | ShopInventory | mob's gold debited, `shopInv.Gold` += full price |
| 5 | Item player merchant | user | user | legacy | merchant gets full price (not the +1 cheat) |
| 6 | Buff player buyer | user | mob | — | buff applied to user, gold debited |
| 7 | Buff mob buyer | mob | mob | — | buff applied to mob, mob's gold debited |
| 8 | Insufficient gold | both | mob | both | no side effects, `Reason=insufficient_gold` |
| 9 | Overburdened buyer | both | mob | legacy | no side effects (validates pre-side-effect gate), `Reason=overburdened` |
| 10 | No match | both | mob | legacy | merchant "say" rejection (mob merchant), no side effects |
| 11 | Quantity `buy 5 X` partial | user | mob | legacy | `Purchased=3, Requested=5`, partial-success message |
| 12 | `from <merchant>` disambig | user | two mob merchants in room | legacy | correct merchant resolved |
| 13 | No merchant in room | both | — | — | `Reason=no_merchant` |
| 14 | Quest engine fires for player only | user vs mob | mob | legacy | quest engine receives event for user, not for mob |

**New `internal/mobcommands/buy_test.go`.** One or two
TryCommand-level smoke tests confirming the registered `buy`
verb wires into `actions.Buy` and basic plumbing works end-to-end
through the mob command pipeline. Thorough cases live in actions
tests; this just exercises the wrapper.

**Parity registry update.**
`internal/mobcommands/command_parity_test.go` tracks which player
commands have mob equivalents — `buy` joins the list.

## Smoke test

After unit tests pass, a manual smoke test in a running server:

1. Spawn a test mob in a shop room (e.g., Thornwall blacksmith).
2. Give the mob some gold via admin command.
3. From the admin console, issue `mob <instanceId> buy iron ingot`
   (or equivalent admin syntax for forcing a mob command).
4. Verify: mob's gold decreased, ingot in mob's inventory,
   merchant restock state updated, room broadcast text visible
   to nearby players, `mob.Character.Skills[bartering]` use count
   incremented.
5. Repeat with `buy heal from <merchant>` for the buff path.
6. Repeat from a starting overburdened state — verify the
   purchase fails with the encumbrance message and no gold is
   spent.

## Out of scope / deferred

- **Mob decision logic.** "What to buy" lives in chunks 2.2
  (item-comparison primitive), 2.3 (equip-if-better behavior),
  4.4 (strategic→tactical translation), and 5.3 (equipment-aware
  shopping). The verb is dumb.
- **Paid-merc hiring.** Today's "buy a merc → permanent charm"
  model is dropped from `actions.Buy` entirely. A future chunk
  may design proper paid-merc semantics (hire fee + periodic
  upkeep + loyalty meter + walk-away-if-unpaid). May not live
  under `buy` at all.
- **Pet purchases.** Companions replaced pets. `executePurchasePet`
  is deleted. `Pet` and `PetType` types stay where they live
  (other systems reference them); only the `buy` path stops
  routing to them.
- **Auto-equip after purchase.** A mob who buys a sword does not
  automatically wield it — chunk 2.3 handles that.
- **Bartering training mechanics for mobs.** Mob bartering
  progresses naturally via the symmetric `OnSkillUse` call; no
  new training mechanism in this chunk. Balance tuning (whether
  the progression rate is right for mobs) deferred to
  playtesting.
- **Restock policy changes.** Legacy `Character.Shop.Restock()`
  on every access and ShopInventory persistence behavior both
  preserved unchanged.
- **Faction / opinion / crime hooks.** Buying from a
  faction-aligned merchant does *not* shift rep or opinion in
  this chunk.
- **`buy` with no args** on the mob side. Mob wrapper treats
  empty `rest` as no-op; mobs don't read `list` output. Player
  wrapper preserves the fall-through to `List(...)`.
- **Player-visible documentation.** No new player-facing help —
  players already know `buy`. Behavior-tree author docs for the
  new mob verb fit into the eventual `internal/behaviortree/`
  context update, not here.

## Roadmap touchpoints

This chunk:

- Closes chunk **2.1** on `MOB_ALIVENESS_ROADMAP.md`.
- Unblocks **2.2** (item-comparison primitive), **2.3**
  (equip-if-better behavior), and **5.3** (equipment-aware
  shopping) by providing the `buy` verb they will dispatch.
- Surfaces a follow-on memory entry for the deferred paid-merc
  hiring system (capture in MEMORY.md when the chunk ships).
