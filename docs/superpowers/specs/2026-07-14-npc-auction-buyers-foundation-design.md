# NPC Auction Buyers — Foundation + Collector (Econ Loop #2, substage #2.1)

**Date:** 2026-07-14
**Status:** Approved (brainstorm), ready for implementation plan
**Goal:** Build the NPC-bidder engine — a non-user sentinel bidder threaded through the escrowed
auction flow, a per-tick bid-decision loop, and an `NpcBuyer` interest/valuation/wallet framework —
proven end-to-end with the first archetype, the **Collector**. This is the skeleton the remaining
four archetypes (craftsperson/adventurer/shopkeeper/official) plug into.

**Scope:** substage #2.1 of the living-world NPC buyers sub-arc (#2 of the econ loop). Only the
Collector archetype ships here. The other archetypes (#2.2–#2.5) and shopkeeper relisting (#3) are OUT.

---

## 1. Current state

After econ #1, `modules/auctions/auctions.go` has a sound escrowed auction: `Bid` debits the bidder's
bank (`getUser`), refunds the previous high bidder, and resolution pays the seller `bid − commission`.
Everything keys off `HighestBidUserId` (a real user). There are **no NPC bidders**, and the escrow
path assumes a user bank — so an NPC bidder needs a parallel gold path (a wallet) and a sentinel so it
can hold the high bid without a user id.

Item facts the Collector uses: `item.GetSpec().Value` (int, valuation basis) and `item.GetSpec().Type`
(explicit equipment types: `Weapon/Offhand/Head/Neck/Body/Belt/Gloves/Ring/Wrist/Legs/Feet/Back/Shoulders/…`).

---

## 2. The `NpcBuyer` framework

```go
type NpcBuyer interface {
	Name() string                    // persona display name, e.g. "Collector Veyd"
	Interested(item items.Item) bool // does this archetype want it?
	MaxBid(item items.Item) int      // valuation ceiling — never bids above this
	Wallet() *NpcWallet              // gold-gated funds
}
```
```go
// NpcWallet is a persistent, slowly-regenerating gold balance for a persona.
// (The shopkeeper archetype in #2.4 will instead back Wallet() with real shop
// gold; the interface is the seam.)
type NpcWallet struct {
	Balance int `json:"balance"`
	Cap     int `json:"cap"`
}
func (w *NpcWallet) CanAfford(n int) bool { return w.Balance >= n }
func (w *NpcWallet) Spend(n int)          { w.Balance -= n }
func (w *NpcWallet) Refund(n int)         { w.Balance += n; if w.Balance > w.Cap { w.Balance = w.Cap } }
func (w *NpcWallet) Regen(amount int)     { w.Balance += amount; if w.Balance > w.Cap { w.Balance = w.Cap } }
```

A package-level registry of the active buyers (for #2.1: 1–2 collectors), and a `buyerByName(name)`
lookup used by refunds.

---

## 3. Sentinel non-user bidder

`AuctionItem` gains:
```go
HighestBidIsNPC bool // when true, the high bid is an NPC; HighestBidUserId is 0 and HighestBidderName is the persona
```
The persona's identity is carried by the existing `HighestBidderName` (a string — persists cleanly;
no interface serialization). `HighestBidUserId == 0 && HighestBidIsNPC` is the sentinel.

**`npcBid(buyer NpcBuyer, bid int)`** (internal, mirrors user `Bid` on the wallet side):
1. Refund the previous high bidder: if `HighestBidUserId > 0` → `refundUser(...)`; else if
   `HighestBidIsNPC` → `buyerByName(HighestBidderName).Wallet().Refund(HighestBid)`.
2. `buyer.Wallet().Spend(bid)`.
3. Set `HighestBid = bid`, `HighestBidUserId = 0`, `HighestBidIsNPC = true`,
   `HighestBidderName = buyer.Name()`.

**User `Bid` change:** when a user outbids an NPC (current `HighestBidIsNPC`), refund the NPC's
wallet (`buyerByName(HighestBidderName).Wallet().Refund(HighestBid)`) instead of `refundUser`. Set
`HighestBidIsNPC = false` on a user bid.

---

## 4. Bid-decision tick

In `newRoundHandler`, in the periodic **auction-update** branch (~line 523, gated by `UpdateSeconds`),
after the human notifications, give the NPC buyers a look:
- For each enabled buyer that is **not** already the high bidder:
  - skip unless `buyer.Interested(item)`,
  - compute `next := HighestBid+1` (or `MinimumBid` if no bid yet),
  - skip unless `next ≤ buyer.MaxBid(item)` **and** `buyer.Wallet().CanAfford(next)`,
  - with probability `AuctionNpcBidChance` (default ~0.35), place `npcBid(buyer, next)` (a small
    single-increment nudge — organic, not a jump to max), and **break** (one NPC bid per tick).

Because a buyer never bids past `MaxBid`, a player who values the item more always wins — NPCs add
liveliness and a soft floor, never a guaranteed purchase. Lots nobody wants still go unsold.

Also add a **wallet regen** step (each buyer `Wallet().Regen(perTick)`) on the same cadence.

---

## 5. Resolution

In `newRoundHandler`'s end branch:
- **NPC won** (`HighestBidUserId == 0 && HighestBidIsNPC`): the seller is paid from the already-held
  NPC gold **minus commission** — reuse the same `payout = HighestBid − commissionFor(...)` path
  (seller bank / inbox as today). The item goes into the collector's collection (a **sink** — removed
  from the world) with a flavor broadcast ("Collector Veyd has acquired the …"). No item delivered to
  a winning-user path.
- **User won:** unchanged (#1 behavior).
- **No bids:** unchanged (`returnUnsoldItem`).

> Sinks are intentional and selective: only Collector (and later Official) sink items; shopkeeper
> relists (#3), adventurer/craftsperson consume into their own use — all later.

---

## 6. The Collector archetype (provisional numbers — tuned in live playtest)

```go
type collector struct {
	name   string
	wallet *NpcWallet
}
func (c *collector) Name() string { return c.name }
func (c *collector) Interested(item items.Item) bool {
	spec := item.GetSpec()
	return isEquipment(spec.Type) && spec.Value >= collectorMinValue
}
func (c *collector) MaxBid(item items.Item) int {
	return int(float64(item.GetSpec().Value) * collectorPremium)
}
func (c *collector) Wallet() *NpcWallet { return c.wallet }
```
- `isEquipment(t items.ItemType) bool` — set membership over the equipment ItemTypes.
- Provisional knobs: `collectorMinValue` **500**, `collectorPremium` **1.0** (bids up to ~market value,
  so any player valuing it above market wins), wallet `Cap` **10000**, regen ~a few % of cap per game
  hour, `AuctionNpcBidChance` **0.35**. 1–2 named collector personas.

---

## 7. Persistence

Persona **wallet balances** must survive restarts. Persist them via the module's existing plugin save
(`save()`/`load()` already round-trip `auctionMgr`) — add a `map[string]int` (persona name → balance)
to the persisted state, applied back to the live buyers on `load`. The buyer definitions themselves are
code (personas), only balances persist. Regen makes balances drift up while offline is fine (clamped to cap).

---

## 8. Config knobs (plugin config, defaults inline like #1)

`AuctionNpcBuyersEnabled` (bool, default true), `AuctionNpcBidChance` (0.35), `CollectorMinValue`
(500), `CollectorPremium` (1.0), `CollectorWalletCap` (10000), `CollectorWalletRegenPerHour`.

---

## 9. Edge cases

- **NPC outbid by NPC:** the refund path handles NPC→NPC (`buyerByName` refund) as well as user↔NPC.
- **Wallet drained:** `CanAfford` gates every bid; a broke collector simply stops bidding.
- **Buy-it-now:** NPCs do **not** trigger buy-it-now — they only nudge incrementally, so `HighestBid`
  never jumps to `BuyoutPrice` via an NPC. (A user can still buy-it-now over an NPC → NPC refunded.)
- **Restart mid-auction:** the NPC-held gold is reflected in the persisted wallet balance and the
  persisted `HighestBid`/`HighestBidIsNPC`/`HighestBidderName`; resolution settles from there.
- **Persona renamed/removed in code:** `buyerByName` returns nil → the refund is skipped (gold not
  returned to a now-nonexistent persona); acceptable and logged. Keep persona names stable.

---

## 10. Testing

Unit (no network):
- Collector `Interested` (equipment + `Value ≥ min` yes; a cheap consumable/component no) and `MaxBid`.
- `NpcWallet` `CanAfford`/`Spend`/`Refund`/`Regen` (clamp at cap).
- **Sentinel flow:** seed an auction; `npcBid` sets the NPC as high bidder + debits the wallet; a
  higher **user** `Bid` refunds the NPC wallet and clears `HighestBidIsNPC`; a higher `npcBid` from a
  second persona refunds the first.
- **Resolution — NPC win:** seller receives `bid − commission`; the winning NPC's wallet is not
  re-credited (it already paid); no user item-delivery.

Manual: list a prestige item; watch a collector place competing bids up to its ceiling over a few
ticks; outbid it as a player (confirm it stops / is refunded); let a collector win a lot nobody else
wanted (confirm seller paid net-of-commission, item sinks with the flavor line).

---

## 11. Out of scope

- **#2.2 Craftsperson / #2.3 Adventurer / #2.4 Shopkeeper / #2.5 Official** — each an `NpcBuyer`
  implementation (interest + valuation + wallet source) plugging into this framework.
- **#3 Shopkeeper relisting** — a shopkeeper win routing the item into shop stock.
- Fine valuation/wallet tuning — provisional here, tuned in live playtest.

---

## 12. Success criteria

- An NPC collector can hold the high bid (sentinel), gold-gated by a persistent regenerating wallet.
- It bids incrementally on prestige items up to its valuation, never above — players can always outbid.
- On an NPC win, the seller is paid net-of-commission and the item sinks (flavored); on a user outbid,
  the NPC wallet is refunded. No gold is minted beyond the (rate-capped, config'd) wallet regen.
- Wallet balances persist across restarts.
