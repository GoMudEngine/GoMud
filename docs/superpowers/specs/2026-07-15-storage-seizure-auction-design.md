# Econ #4 — Bank-storage → Auction (lien seizure)

**Date:** 2026-07-15
**Status:** Design (approved decisions; pending spec review)
**Roadmap:** `docs/PATH_TO_1.0.md` §1 econ arc, item **#4** ("Bank-storage → auction —
storage items whose owner can't pay go to the block instead of being lost").
**Depends on:** #1 auction mechanics core (escrow, reserve, commission, unsold-return)
and #2 NPC buyers (collector/craftsperson/adventurer/shopkeeper), both merged on local
master.

---

## 1. Problem

Bank storage charges monthly rent (`slots × StorageFeePerItem`). When a player's bank
can't cover it, the current forfeiture path (`internal/hooks/StorageFee_MonthlyCharge.go`)
drains the bank to 0 and **deletes** the cheapest slots outright to cover the shortfall.
The item is simply lost.

#4 replaces that deletion: a genuinely valuable seized item is **listed on the auction
block** instead. Players (and the NPC buyers from #2) can bid on it; the bank recovers the
unpaid rent from the sale (a lien), and any surplus returns to the ex-owner. The player
gets a real chance to reclaim value rather than losing the item to a silent delete.

Low-value junk is still disposed immediately — the block is reserved for items worth a
buyer's attention.

---

## 2. Approved design decisions

1. **Proceeds = lien model.** The bank recoups the unpaid rent off the top; the surplus
   returns to the ex-owner. (Economically identical to a pure sink today since no gold
   flows out of the bank/house yet, but gives the better player story for free.)
2. **Persisted seizure queue** in the auctions module holds seized lots until the
   one-at-a-time block frees; survives restarts.
3. **Unsold seized lot → dispose** (delete) after a fair auction, rather than returning it
   to storage (which would re-trigger the same debt endlessly).
4. **Value floor (per unit).** A seized slot whose **per-unit** value (`spec.Value`) is
   below `StorageSeizureMinValue` (default **250g**) is disposed immediately and never
   reaches the block. Keying on per-unit (not stack) value deliberately avoids qualifying a
   large pile of individually-cheap items, which would otherwise list as an un-sellable
   mega-lot. In practice, qualifying slots are almost always non-stackable gear
   (`Count == 1`).

---

## 3. Architecture & layering

`internal/hooks` does **not** import `modules/` (verified). So the fee hook communicates
with the auctions module via the existing event bus:

```
StorageFee hook  --events.StorageItemSeized-->  auctions module listener
  (internal/hooks)                                 (modules/auctions)
```

Both packages import `internal/events` and `internal/items`; `internal/events` already
carries `items.Item` (precedent: `events.ItemOwnership`). No import cycle.

### New event — `internal/events/eventtypes.go`

```go
// StorageItemSeized is emitted when the storage-fee hook seizes a stored
// item (value >= StorageSeizureMinValue) from a player who cannot pay their
// rent. The auctions module listens and enqueues it for the block.
type StorageItemSeized struct {
    UserId int         // ex-owner; surplus after the lien returns here
    Item   items.Item  // the seized stack (Count folded per item semantics)
    Count  int         // stack count seized
    Owed   int         // this lot's apportioned share of the unpaid rent (the lien)
}
```

(`Type()` / `Data()` methods per the existing event interface.)

---

## 4. Seizure trigger & selection — `internal/hooks/StorageFee_MonthlyCharge.go`

Only the **forfeiture branch** (bank can't pay in full) changes. Everything up to the
"cannot pay in full" point is unchanged: drain the bank to 0, compute
`shortfall = fee − available`, and select cheapest slots by `spec.Value × Count` ascending,
enough slots to cover the shortfall at `feePerSlot` per slot (existing selection loop).

The cheapest-first **selection order** is unchanged (still by `spec.Value × slot.Count`
ascending — it decides *which* slots to grab to minimise player pain). The auction **floor**
is evaluated separately, on **per-unit** value. For each **selected** slot, replace the
unconditional delete with:

```
unitValue   = spec.Value           // per unit, NOT stack value
owedPortion = feePerSlot           // this slot's lien; disposed slots owe nothing

remove the slot from storage (as today)

if !StorageSeizureAuctionEnabled OR unitValue < StorageSeizureMinValue:
    dispose the whole slot  (all Count units gone; owedPortion written off — as today)
    add slot name to the "disposed" list
else:
    emit events.StorageItemSeized{UserId, Item: slot.Item, Count: slot.Count, Owed: owedPortion}
    add slot name to the "seized & listed" list
```

A qualifying slot is seized as one whole stack (its `Count` carried on the event); the lot
lists once and the winner receives all `Count` units (§6). Because only high-per-unit-value
items qualify, `Count` is ~always 1 and a `unitValue × Count` buyout stays a fair price.

**Debt apportionment:** each auctioned slot carries a flat `feePerSlot` lien; disposed
sub-floor slots carry none (nothing sells, so their portion is written off — identical to
today). Per-lot settlement stays independent — no shared running counter across lots that
resolve at different times. `feePerSlot` defaults to **1g**, so the lien is deliberately
tiny — the surplus returned to the ex-owner is essentially the full sale price. The lien
machinery is correct and future-proofs a higher `StorageFeePerItem`, but is economically
minor at the default (this is why the lien vs. player-keeps-all models are indistinguishable
today). Settlement caps recovery at `min(owed, afterCommission)`, so a flat per-slot lien can
never over-charge a given lot.

**Inbox notice** is reworded to distinguish the two outcomes, e.g.:

> Thornwall Bank Notice: Insufficient funds for storage fees.
> These items were seized and sent to auction to cover the debt — anything they fetch above
> what you owe (and our fee) returns to you: `<seized list>`.
> These items had too little value to auction and were disposed: `<disposed list>`.
> Your remaining N slot(s) are secure.

(Only the non-empty lists are shown. Follows the no-hard-numbers convention for *effects*;
gold amounts in a bank notice are acceptable, matching the existing fee notice.)

---

## 5. Seizure queue — `modules/auctions/`

`AuctionManager` gains persisted queue state (saved/loaded with existing module state in
`save()` / `load()`):

```go
type SeizedLot struct {
    Item          items.Item
    Count         int
    ExOwnerUserId int
    Owed          int   // the lien
}

// on AuctionManager:
SeizedQueue []SeizedLot
```

- **Listener** (registered at module load alongside `newRoundHandler`) on
  `events.StorageItemSeized` → append one `SeizedLot` carrying `Count`. The slot is always
  listed as a **single stacked lot** (one block slot, matches how storage bills per-stack,
  and the lien is per-stack). The winner receives all `Count` units at resolution (§6).
- **Drain** in `newRoundHandler`: after the current lot resolves (or when there is no
  active auction), if `ActiveAuction == nil` and `len(SeizedQueue) > 0`, pop the front and
  list it (see §6). One seized lot promoted per free block; any backlog drains over
  subsequent rounds and survives restarts.

Player-listed lots continue to fail with the existing "an auction is already active"
message when the block is busy — seized lots do not preempt an active lot, they wait.

---

## 6. Listing a seized lot & lien settlement

### Listing

New fields on `AuctionItem`:

```go
Seized   bool   // this lot came from a storage seizure
OwedLien int    // rent to recoup from the sale before surplus returns to the seller
```

The drain lists via a seized-aware path (either an overload of `StartAuction` or a small
helper that sets the same fields plus `Seized`/`OwedLien`). Parameters:

| Field         | Value                                                          |
|---------------|---------------------------------------------------------------|
| `SellerUserId`| ex-owner (so surplus routes back to them)                     |
| `Anonymous`   | **true** — the block shows an anonymous/bank seller, not the dispossessed player |
| `buyout`      | `spec.Value × Count`                                          |
| reserve       | existing `reserveFrom(buyout, 25%)`                          |
| duration      | module plugin config `DurationSeconds` (default 60), same as player lots |
| `Seized`      | true                                                          |
| `OwedLien`    | `SeizedLot.Owed`                                             |

**Stack win-delivery.** Normal lots are single-item, so the existing win path grants one
unit. A seized lot may carry `Count > 1`; the win-delivery gains a seized-only branch that
grants all `Count` units (loop `StoreItem` / restore the stack; offline path attaches the
stack to the inbox). `Count` is ~always 1, but this keeps stacks correct.

**NPC buyers participate for free.** A seized lot is a normal `ActiveAuction`, so the
collector / craftsperson / adventurer / shopkeeper bid logic in `newRoundHandler` already
applies with no extra wiring.

### Settlement (sold — player or NPC won)

In `newRoundHandler`'s resolution block, when `ActiveAuction.Seized` is true the seller
settlement is replaced with lien math (the winner already paid at bid time via escrow; the
held gold settles here):

```
afterCommission = HighestBid - commissionFor(HighestBid, auctionCommissionPct)
lienRecouped    = min(OwedLien, afterCommission)     // recovered by the house (gold sink)
surplus         = afterCommission - lienRecouped     // returns to the ex-owner

credit surplus to ExOwner bank (online) or inbox gold (offline)  // reuse existing offline path
```

Ex-owner notice: what sold, that the rent was recovered, and the surplus returned (descriptive,
no raw internal balance leakage beyond gold amounts already shown in bank notices).

The NPC-win item-sink and shopkeeper-relist behavior (`auctionWinReceiver.Receive`) is
unchanged — a seized lot won by an NPC sinks / relists exactly as any other lot.

### Settlement (unsold — no player and no NPC bid)

When `ActiveAuction.Seized` is true **and** the lot ended with no bidder, do **not** call
`returnUnsoldItem` (that path would push it back to the ex-owner's storage and re-trigger the
debt next month). Instead **dispose** the item and send an inbox notice:

> Your seized `<item>` found no buyer at auction and was disposed.

This is the loop-breaker: the player got a fair shot; a lot that clears the 250g floor but
still attracts zero bids is gone.

---

## 7. Config

`internal/configs/config.balance.shops.go` (next to `StorageFeePerItem`):

| Knob                            | Type / default | Meaning                                                        |
|---------------------------------|----------------|----------------------------------------------------------------|
| `StorageSeizureAuctionEnabled`  | bool, **true** | Master toggle. Off ⇒ old behavior (seized slots deleted).      |
| `StorageSeizureMinValue`        | int, **250**   | Minimum stack value (`spec.Value × Count`) to auction rather than dispose. |

Both defaulted in the shops-config validator. `StorageFeePerItem` and the module
`DurationSeconds` / reserve / commission knobs are reused unchanged.

---

## 8. Testing

Unit tests (`modules/auctions/auctions_test.go`, plus a hook-level test for selection):

- **Selection/floor:** per-unit value ≥ 250 → `StorageItemSeized` emitted; per-unit value
  < 250 → disposed, no event. A many-unit stack of individually-cheap items (high *stack*
  value, low *unit* value) → disposed, not listed. Toggle off → all seized slots disposed
  (regression to old behavior).
- **Stack win-delivery:** a seized lot with `Count > 1` grants all `Count` units to the
  winner (online loop + offline inbox).
- **Debt apportionment:** each auctioned slot's `Owed == feePerSlot`; a disposed sub-floor
  slot mixed into the same seizure carries no lien.
- **Queue drain:** enqueue while block busy → nothing lists; free block → front pops and
  lists; multiple seized lots drain one-per-round; queue persists across `save()`/`load()`.
- **Lien settlement (sold):** `surplus = bid − commission − min(owed, bid−commission)`;
  house keeps the lien; ex-owner credited surplus; offline ex-owner gets inbox gold.
- **Lien settlement, small sale:** sale ≤ owed ⇒ surplus 0, no negative credit.
- **NPC win:** seized lot won by an NPC sinks/relists as normal; lien still settles.
- **Unsold seized:** no bids ⇒ disposed, ex-owner notified, item **not** returned to storage
  (distinguish from the normal unsold-return path, which is unchanged for player lots).

Full suite green + boot clean (pre-push SOP) before merge.

---

## 9. Out of scope / deferred

- Retuning `StorageFeePerItem` or the 250g floor — provisional, defer to post-build
  playtest (project convention).
- Any change to the player-listing `auction` command UX or the one-at-a-time block model
  for player lots.
- #2.5 Official NPC buyer (separate optional substage).

---

## 10. Files touched (anticipated)

- `internal/events/eventtypes.go` — new `StorageItemSeized` event.
- `internal/hooks/StorageFee_MonthlyCharge.go` — forfeiture branch: floor + dispose/emit.
- `modules/auctions/auctions.go` — `SeizedLot` + `SeizedQueue`, listener registration,
  drain in `newRoundHandler`, `Seized`/`OwedLien` on `AuctionItem`, seized-aware listing +
  lien settlement + unsold-dispose.
- `internal/configs/config.balance.go` + `config.balance.shops.go` — two new knobs +
  defaults.
- `modules/auctions/auctions_test.go` (+ a hook selection test) — coverage above.
- `PATCH_NOTES.md`, `docs/PATH_TO_1.0.md` (mark #4 done) — at merge.
