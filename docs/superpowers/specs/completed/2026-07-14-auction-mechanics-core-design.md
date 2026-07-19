# Auction Mechanics Core — Design (Econ Loop, sub-project #1 of 4)

**Date:** 2026-07-14
**Status:** Approved (brainstorm), ready for implementation plan
**Goal:** Rebuild the auction house's economic core — real escrowed gold flow, a buyout price
with a derived reserve, a house commission (gold sink), and reliable return of unsold items —
so the marketplace is economically sound and ready for the living-world NPC buyers (#2), shopkeeper
relisting (#3), and the bank-storage→auction pipeline (#4).

**Scope note:** This is sub-project #1 of the econ-loop arc. NPC buyers, shopkeeper relisting, and
the storage pipeline are OUT of scope here — this piece is the skeleton they hang on.

---

## 1. Current state (what's broken)

`modules/auctions/auctions.go` is a working *player* auction house (one active lot, `auction`
command, per-player notify toggle, offline delivery via inbox). But its economics are broken:

- **Gold faucet:** the only gold movement is `sellerUser.Character.Bank += HighestBid` (line ~407).
  The **winner is never debited**, there is **no escrow**, and **no affordability check** at bid time
  (you can bid gold you don't have). Every sale mints gold.
- **No reserve / no commission.**
- **Unsold + offline seller can lose the item:** the no-winner path only returns the item to an
  *online* seller (`users.GetByUserId` → backpack); an offline seller's item appears to be dropped.

---

## 2. The model

- **Listing:** seller lists `item + buyout price + duration`. The **reserve = minimum bid =
  `buyout × AuctionReservePct`** (default 0.25), derived automatically. No other seller input.
- **Bidding + escrow:** a bid must be `≥ reserve` and `> current high bid`, and `≤ the bidder's bank`.
  On a valid bid the gold is **escrowed** (debited from the bidder's bank into the lot); the previous
  high bidder is **refunded** immediately (bank credit; saved if offline). Bidding `≥ buyout` caps at
  buyout and **ends the lot immediately** (buy-it-now).
- **Resolution — sold** (≥1 bid): commission = `HighestBid × AuctionCommissionPct` (default 0.05) is a
  **gold sink** (it's already out of the winner's bank; it is simply not passed on). Seller receives
  `HighestBid − commission` (bank, or inbox if offline). Winner gets the item (backpack, inbox if
  offline) — unchanged.
- **Resolution — unsold** (no bids): the item is **reliably returned** to the seller — to **bank item
  storage** (`user.ItemStorage`) if there's room, else the **inbox/mudmail** — so it's never lost,
  online or off. No escrow to unwind (there was no bid).

**Net gold flow on a sale:** winner `−HighestBid`, seller `+(HighestBid − commission)`, commission
sunk. Gold-neutral minus the sink. No minting.

---

## 3. Data model changes (`AuctionItem`)

Add to `AuctionItem`:
```go
BuyoutPrice int // seller-set buy-it-now; reserve/min-bid derive from this
```
The escrow is represented implicitly by the existing `HighestBid` + `HighestBidUserId` (the current
high bid is the gold currently held from the current high bidder). `MinimumBid` becomes the derived
reserve (`BuyoutPrice × AuctionReservePct`), not a separate seller input.

---

## 4. Config knobs (plugin config)

The module already reads `mod.plug.Config.Get("DurationSeconds")` / `"Anonymous"`. Add, with defaults
set in the module `init` alongside `maxHistoryItems`:
- `AuctionReservePct` (float, default **0.25**) — reserve/min-bid as a fraction of buyout.
- `AuctionCommissionPct` (float, default **0.05**) — house cut of the sale price; sunk.

---

## 5. Component changes

### 5.1 Listing command (`auctionCommand`, ~line 300-325)
Change the amount prompt from "minimum bid" to **"buyout price"**. On submit, call
`StartAuction(item, userId, buyout, duration, anon)`. Confirmation text mentions the buyout and the
derived reserve.

### 5.2 `StartAuction(item, userId, buyout, durationSeconds, anon)`
Set `BuyoutPrice = buyout`, `MinimumBid = int(float64(buyout) * AuctionReservePct)` (min 1). Rest
unchanged (removes the item from the seller — already done by the command).

### 5.3 `Bid(userId, bid) error`
Add, in order:
1. Reserve/increment check: `bid ≥ MinimumBid` and `bid > HighestBid` (existing, retargeted to reserve).
2. **Affordability:** bidder's `Character.Bank ≥ bid`, else error.
3. **Buy-it-now:** if `bid ≥ BuyoutPrice`, set `bid = BuyoutPrice` and mark the lot to end now
   (e.g. set `EndTime = now`) so the resolution tick finalizes it.
4. **Escrow swap:** debit the new bidder's bank by `bid`; if there was a previous `HighestBidUserId`,
   credit that user's bank by the old `HighestBid` (refund) and `SaveUser` if offline. Emit
   `EquipmentChange{BankChange}` events for prompt/GMCP refresh on both.
5. Set `HighestBid/HighestBidUserId/HighestBidderName`.

### 5.4 Resolution (the end-of-auction tick, ~line 340-486)
- **Sold branch** (`HighestBidUserId > 0`): replace `seller.Bank += HighestBid` with
  `seller.Bank += HighestBid − commission` (commission = `int(float64(HighestBid) * AuctionCommissionPct)`).
  The winner's gold was already taken at bid time (escrow), so **do not** debit again here. Winner
  item delivery unchanged.
- **Unsold branch** (no bids): return the item to the seller reliably — try
  `seller.ItemStorage` (respecting the storage cap) then fall back to `seller.Inbox.Add({Item})`;
  works for online AND offline sellers. Notify.

---

## 6. Edge cases

- **Offline high bidder outbid:** refund to their bank + `SaveUser`. (They see it next login.)
- **Server restart / copyover mid-auction:** the debit is already persisted in the bidder's saved
  bank and the held amount in the saved `ActiveAuction`; resolution credits the seller on the tick.
  Consistent. (A crash in the tiny window between bank-debit and auction-save could lose escrow —
  accepted, same risk profile as any single-write.)
- **Storage full on unsold return:** fall back to inbox/mudmail.
- **Buyout bid:** capped at buyout, ends immediately; the same escrow/commission path applies.
- **Seller = only bidder:** disallowed (existing "already highest bidder" guard covers self; the
  affordability check + reserve stand).
- **Anonymous lots:** unchanged (names masked in broadcasts).

---

## 7. Testing

Unit tests on `AuctionManager` (no network) with seeded users/items:
- `StartAuction` derives `MinimumBid = 25% of buyout` (rounding, min 1).
- `Bid` rejects below reserve / below bank / not-higher-than-current; accepts a valid bid and
  **escrows** (bidder bank down, held on the lot).
- **Outbid refund:** second bidder wins the high slot; first bidder's bank restored exactly.
- **Buy-it-now:** a `≥ buyout` bid caps at buyout and flags the lot ended.
- **Resolution sold:** seller receives `HighestBid − commission`; commission not credited anywhere
  (sunk); winner's bank already reduced.
- **Resolution unsold:** item returns to storage (or inbox when full); seller bank unchanged.

Manual: list a lot, bid from a second connection, get outbid (see refund), win, confirm the seller's
bank nets bid−commission and the item transfers; let a lot expire with no bids and confirm it returns
to storage.

---

## 8. Out of scope (later econ-loop sub-projects)

- **#2 Living-world NPC buyers** — the five archetypes with interest filters + valuation + gold-gated
  wallets (itself ~5 substages). The buyout/reserve here is the value basis they'll reason about.
- **#3 Shopkeeper relisting** — a shopkeeper-archetype win routes the item into shop stock.
- **#4 Bank-storage → auction** — storage items whose owner can't pay go to the block instead of
  being deleted.

---

## 9. Success criteria

- Auctions no longer mint gold: a sale is winner `−bid`, seller `+(bid − commission)`, commission sunk.
- You cannot bid more than your bank; outbid bidders are refunded.
- A seller-set buyout drives a 25% reserve; a buyout bid wins instantly.
- Unsold lots always come back to the seller (storage or inbox) — never lost, online or offline.
- Config knobs (`AuctionReservePct`, `AuctionCommissionPct`) tune reserve + commission.
