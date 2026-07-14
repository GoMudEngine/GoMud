# NPC Auction Buyer — Shopkeeper (Econ #2.4, folding in #3 relist)

**Date:** 2026-07-14
**Status:** Approved (brainstorm), ready for implementation plan
**Goal:** Add the **shopkeeper** living-world auction buyer — a merchant that bids on items its shop
deals in, using **real per-shop gold**, and on a win **relists the item into that shop's stock**
(folding the nominally-separate #3 relisting into this substage so a shop never loses gold for
nothing).

**Scope:** econ substage #2.4 + #3. Out: #2.5 Official (needs a new `restricted` flag), #4
storage→auction. Numbers stay provisional (delegated to existing shop economics — almost nothing new
to tune).

---

## 1. Background

#2.1–#2.3 built the `NpcBuyer` framework (`Interested`/`MaxBid`/`Wallet`/`Flavor`), the sentinel
non-user bidder threaded through the escrow (`HighestBidIsNPC` + `npcBid` + `refundPreviousBidder`),
the per-tick `nextNpcBid` decision, wallet persistence (`WalletBalances`), and the per-archetype win
broadcast. Collector/craftsperson/adventurer all use a synthetic **regenerating `NpcWallet`**.

The shopkeeper differs in one hard way: its wallet is **real, per-shop gold**
(`shops.ShopInventory.Gold`), not a synthetic regen wallet. That "hybrid wallet" drives the design.

**The key realization:** the shopkeeper's entire valuation already exists as `shops.EvaluateBuyRules`.
Given `(item, *ShopInventory)` it returns a `BuyOffer{Price}` after matching the item's
`VendorCategories` against the shop's `CraftSupport`, applying dynamic buy pricing (`BuyRatio` +
scarcity), enforcing the overstock cap, and gating on the shop's gold reserve
(`GoldReserve(ShopGoldReserveRatio)`). `Price > 0` means "this shop wants it and can afford it."
The shopkeeper archetype is therefore a **thin adapter** over `EvaluateBuyRules` — reusing all of the
shop's real economics rather than inventing parallel premium/min-value knobs.

---

## 2. One dynamic meta-buyer over all live shops

A single `shopkeeper` persona in the `npcBuyers` registry with a **fixed** `Name()` =
`"The Merchants' Guild"` (fixed so `buyerByName(HighestBidderName)` refund-keying stays stable; the
specific backing shop is internal state). Per decision tick it scans `shops.AllShops()`, runs
`EvaluateBuyRules(item, shopInv, "", false, cfg, nil)` against each (the `crafterSkill`/`buysGeneral`/
`wornItems` args are documented no-ops in the current rules), and **selects the shop with the highest
positive `BuyOffer.Price`** it can afford.

```go
type shopSel struct {
    uuid  uuid.UUID          // item this selection was computed for
    shop  *shops.ShopInventory
    offer int                // best BuyOffer.Price (0 = no shop wants it)
}

func (s *shopkeeper) selectFor(item items.Item) shopSel {
    if s.sel.uuid == item.UUID { return s.sel }   // memoized within a decision
    cfg := shops.PricingConfigFromBalance()
    best := shopSel{uuid: item.UUID}
    for _, inv := range shops.AllShops() {
        off := shops.EvaluateBuyRules(item, inv, "", false, cfg, nil)
        if off.Price > best.offer {
            best.shop, best.offer = inv, off.Price
        }
    }
    s.sel = best
    return best
}

func (s *shopkeeper) Interested(item items.Item) bool { return s.selectFor(item).offer > 0 }
func (s *shopkeeper) MaxBid(item items.Item) int      { return s.selectFor(item).offer }
```

- `MaxBid` = the selected shop's `BuyOffer.Price` — already the buy-low price (via `BuyRatio`). The
  shop won't pay more at auction than over its own counter, so **players always outbid** (same
  guarantee as the other archetypes).
- Selection is memoized by item `UUID`; safe because `nextNpcBid` calls Interested→MaxBid→CanAfford
  for the same item on the single-threaded auction tick, and the high bidder is skipped by
  `nextNpcBid` (so the binding is frozen while the shopkeeper leads).

---

## 3. Hybrid wallet — real shop gold via a widened escrow seam

Move the escrow seam from `Wallet().Spend/Refund/CanAfford` onto the `NpcBuyer` interface itself, so a
buyer can be backed by something other than an `NpcWallet`:

```go
type NpcBuyer interface {
    Name() string
    Interested(item items.Item) bool
    MaxBid(item items.Item) int
    CanAfford(n int) bool         // ← escrow seam (was Wallet().CanAfford)
    Spend(n int)                  // ← escrow seam (was Wallet().Spend)
    Refund(n int)                 // ← escrow seam (was Wallet().Refund)
    Flavor() string
    Wallet() *NpcWallet           // regen wallet for persistence/regen; nil for shopkeeper
}
```

**Regen archetypes** (collector/craftsperson/adventurer): change the `wallet *NpcWallet` field to an
**embedded `*NpcWallet`** so `CanAfford/Spend/Refund/Regen` promote automatically — zero delegation
boilerplate. `Wallet()` returns the embedded pointer. Registry init changes from
`wallet: &NpcWallet{...}` to `NpcWallet: &NpcWallet{...}`. This is behavior-preserving.

**Shopkeeper**: `Wallet()` returns `nil`; escrow hits the selected/bound shop's real gold.

```go
type shopkeeper struct {
    name  string
    sel   shopSel               // memoized selection for the current decision
    bound *shops.ShopInventory  // shop escrowed against while high bidder
}

func (s *shopkeeper) Wallet() *NpcWallet { return nil }

func (s *shopkeeper) CanAfford(n int) bool {
    if s.sel.shop == nil { return false }
    return s.sel.shop.CanAfford(n, s.sel.shop.GoldReserve(reserveRatio()))
}
func (s *shopkeeper) Spend(n int) {
    s.bound = s.sel.shop            // freeze the binding for refund/win
    s.bound.Gold -= n
    saveShop(s.bound)
}
func (s *shopkeeper) Refund(n int) {
    if s.bound == nil { return }
    s.bound.Gold += n
    saveShop(s.bound)
}
```

`reserveRatio()` reads `configs.GetBalanceConfig().ShopGoldReserveRatio` (0.50 fallback, same as
`EvaluateBuyRules`). `saveShop` wraps `shops.SaveShop(inv.Zone, inv.MobId, inv.RoomId)` with error
logging. Because `EvaluateBuyRules` already refused any offer that would breach the reserve at the
shop's current gold, every `next ≤ MaxBid` bid is reserve-safe; refunds restore the shop's gold before
the next Spend, so the reserve invariant holds across the whole auction.

**Framework call-site changes** in `auctions.go`:
- `b.Wallet().CanAfford(next)` → `b.CanAfford(next)` (in `nextNpcBid`)
- `buyer.Wallet().Spend(bid)` → `buyer.Spend(bid)` (in `npcBid`)
- `b.Wallet().Refund(a.HighestBid)` → `b.Refund(a.HighestBid)` (in `refundPreviousBidder`)
- Regen loop: `for _, b := range npcBuyers { if w := b.Wallet(); w != nil { w.Regen(...) } }`
- Persistence save: `if w := b.Wallet(); w != nil { WalletBalances[b.Name()] = w.Balance }`
- Persistence load: unchanged (only sets balances on buyers present in the map).

---

## 4. Win → relist into the bound shop (folds #3 in)

A shopkeeper win must route the item into the bound shop's stock, not sink it. Use an **optional
type-asserted receiver** so the base interface stays clean:

```go
type auctionWinReceiver interface { Receive(item items.Item) }

func (s *shopkeeper) Receive(item items.Item) {
    if s.bound == nil { return }
    cap := int(configs.GetBalanceConfig().ShopAffixedStockCap)
    s.bound.AddAffixedStock(item, item.GetSpec().Value, cap)   // instance-safe, mirrors sell.go:327
    s.bound.BuysCount++
    saveShop(s.bound)
    s.bound = nil
}
```

In the NPC-win block of the resolution (`auctions.go` ~line 498), after resolving the buyer for the
flavor:

```go
if b := buyerByName(auctionNow.HighestBidderName); b != nil {
    flavor = b.Flavor()
    if r, ok := b.(auctionWinReceiver); ok {
        r.Receive(auctionNow.ItemData)   // shopkeeper relists; others no-op → sink
    }
}
```

All won items route through `AddAffixedStock` (holds the full `items.Item`, so exact affixes/enchants/
crafted state survive and the item becomes purchasable), mirroring the affixed-buyback path. Broadcast:
`"The Merchants' Guild has acquired the <item> for the shelves."` (`Flavor()` = `"for the shelves"`).

Non-shopkeeper archetypes don't implement `Receive`, so their win stays the existing flavored sink.

---

## 5. Registry & config

Registry gains one persona:
```go
&shopkeeper{name: "The Merchants' Guild"},
```
No synthetic wallet, so it contributes nothing to `WalletBalances` (guarded by `Wallet()==nil`).

Config: reuses `AuctionNpcBuyersEnabled` + shared `npcBidChancePct` + existing shop knobs
(`ShopGoldReserveRatio`, `ShopAffixedStockCap`, `BuyRatio`, scarcity). One optional new toggle
`AuctionShopkeeperEnabled` (bool, default true) read in `load()`; when false the shopkeeper is skipped
in `nextNpcBid`. **No new valuation numbers** — the payoff of delegating to `EvaluateBuyRules`.

---

## 6. Testing (unit, no network)

Shop stubs are cheap to construct (`&shops.ShopInventory{Gold, CraftSupport, Stock, ...}`); the tests
drive `EvaluateBuyRules` through the adapter. Since `AllShops()` reads a package cache, tests seed it
via the existing shop registration/`ClearCache` helpers (see `internal/shops/*_test.go`).

- **`Interested`/`MaxBid`:** a registered shop whose `CraftSupport` matches the item's
  `VendorCategories` with ample gold → interested at the `EvaluateBuyRules` price; no matching shop →
  not interested; a matching shop below its gold reserve → not interested (offer 0).
- **Selection:** with two matching shops, the higher-affordable-offer shop is chosen and becomes the
  bound shop on `Spend`.
- **Escrow:** `Spend(n)` debits exactly the bound shop's `Gold`; `Refund(n)` after an outbid restores
  it exactly; a second selection can bind a different shop.
- **Win/relist:** `Receive(item)` lands the exact instance in the **bound** shop's `AffixedStock` (and
  no other shop's), bumps `BuysCount`, and clears the binding.
- **Regen archetypes unchanged:** the embedding refactor is behavior-preserving (existing
  collector/craftsperson/adventurer tests stay green); `Wallet()==nil` shopkeeper is skipped by the
  regen + persistence loops (no nil panic).

The bid/resolution plumbing is already covered by the #2.1 framework tests — the shopkeeper supplies a
different valuation + a real-gold purse into the same engine.

---

## 7. Out of scope

- **#2.5 Official** — needs a new `restricted` item flag; deferred/optional.
- **#4 Bank-storage → auction** — separate substage.
- Commodity-merge of won items into regular `StockEntry` (AffixedStock is instance-safe and
  sufficient); fine-tuning of thresholds (delegated to existing shop config).
- **Followup (separate): a minimum-value floor on auction *listings*.** Gating what can be listed on
  the block (on the `auction` sell path, not the shopkeeper) keeps trivial items — a plain iron sword —
  off the block entirely, which also keeps them out of the shopkeeper's AffixedStock relist. Own
  spec/plan; not part of #2.4.

---

## 8. Success criteria

- The shopkeeper bids on items a real shop deals in (`VendorCategories` ↔ `CraftSupport`), up to that
  shop's own buy price, gold-gated by the shop's **real** gold above its reserve — players always
  outbid.
- On a shopkeeper win, the correct shop's `Gold` was spent, the exact item instance lands in that
  shop's resale stock (purchasable), and the win broadcasts "…for the shelves."
- No shop is ever left having spent gold for nothing (relist folded in); the reserve invariant holds.
- All existing NPC-buyer archetypes and #2.1–#2.3 tests are unchanged.
