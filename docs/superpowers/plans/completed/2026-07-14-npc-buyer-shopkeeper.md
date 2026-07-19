# Shopkeeper NPC Auction Buyer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `shopkeeper` living-world auction buyer that bids on items a real shop deals in, using
that shop's **real gold**, and on a win **relists the won item into the shop's stock**.

**Architecture:** The shopkeeper is a thin adapter over `shops.EvaluateBuyRules` (which already does
the VendorCategories↔CraftSupport match, dynamic buy price, overstock cap, and gold-reserve gate). Its
"wallet" is the selected shop's real `ShopInventory.Gold`, reached through a widened escrow seam
(`CanAfford`/`Spend`/`Refund`) that moves from `Wallet()` onto the `NpcBuyer` interface. A win routes
the item into the bound shop's `AffixedStock`.

**Tech Stack:** Go, `modules/auctions` (plugin module), `internal/shops`, `internal/items`,
`internal/configs`. Tests are table/unit style (`go test`), no network.

**Spec:** `docs/superpowers/specs/2026-07-14-npc-buyer-shopkeeper-design.md`

---

## File Structure

- **Modify** `modules/auctions/npc_buyers.go` — widen `NpcBuyer` interface (`CanAfford`/`Spend`/
  `Refund`); add delegating methods to the three regen archetypes; add the `shopkeeper` type,
  `shopSel`, the `saveShopFn` seam, the `shopkeeperEnabled` toggle, and register the persona.
- **Modify** `modules/auctions/auctions.go` — switch escrow call sites to the new seam; nil-guard the
  regen/persistence loops; add the `auctionWinReceiver` hook in the NPC-win block; read the
  `AuctionShopkeeperEnabled` config knob in `load()`.
- **Modify** `modules/auctions/npc_buyers_test.go` — new shopkeeper unit tests (seam + valuation +
  escrow + relist).
- **No new files.** The shopkeeper is ~60 lines added to the existing archetype file, consistent with
  how collector/craftsperson/adventurer live there.

Key verified facts the implementer can rely on:
- `shops.EvaluateBuyRules(item items.Item, shopInv *shops.ShopInventory, crafterSkill string, buysGeneral bool, cfg shops.PricingConfig, wornItems []items.Item) shops.BuyOffer` — `crafterSkill`/`buysGeneral`/`wornItems` are documented no-ops; pass `"", false, nil`. `BuyOffer{Price int; Reason string}`.
- `shops.AllShops() []*shops.ShopInventory`, `shops.PricingConfigFromBalance() shops.PricingConfig`, `shops.SaveShop(zone string, mobId, roomId int) error`.
- `(*shops.ShopInventory).CanAfford(amount, reserveFloor int) bool` = `Gold-amount >= reserveFloor`; `.GoldReserve(ratio float64) int` = `StartingGold*ratio`; `.AddAffixedStock(item items.Item, price, cap int)`; fields `Gold`, `StartingGold`, `CraftSupport`, `Zone`, `MobId`, `RoomId`, `BuysCount`.
- `configs.GetBalanceConfig().ShopGoldReserveRatio` (ConfigFloat, default 0.50), `.ShopAffixedStockCap` (ConfigInt, default 8). Cast with `float64(...)` / `int(...)`.
- `items.Item.UUID` is `internal/uuid.UUID` (comparable with `==`, has `.IsNil()`); `items.New(id)` assigns a fresh UUID. `item.GetSpec()` returns `ItemSpec` with `.Value int` and `.VendorCategories []string`.
- Test item seeding: `defer items.SeedItemsForTest(map[int]*items.ItemSpec{...})()`. Shop seeding: `shops.ClearCache(); shops.RegisterShop(zone, mobId, roomId, shops.ShopInventory{Gold, StartingGold, CraftSupport})` (no disk write when CraftSupport is non-empty and cache-missed).
- Valid disciplines: `shops.CraftSupportBlacksmithing` etc. (`shops.ValidCraftSupports`); items use the same minus `"general"`.

---

### Task 1: Widen the escrow seam onto `NpcBuyer` (pure refactor)

Move `CanAfford`/`Spend`/`Refund` from `Wallet()` onto the interface so a buyer can be backed by
something other than an `NpcWallet`. Behavior-preserving.

**Files:**
- Modify: `modules/auctions/npc_buyers.go`
- Modify: `modules/auctions/auctions.go`

- [ ] **Step 1: Add the three methods to the interface + delegating impls**

In `modules/auctions/npc_buyers.go`, change the interface:

```go
// NpcBuyer is one living-world auction buyer archetype.
type NpcBuyer interface {
	Name() string
	Interested(item items.Item) bool
	MaxBid(item items.Item) int
	CanAfford(n int) bool // escrow seam: does the buyer's purse cover n?
	Spend(n int)          // escrow seam: debit the purse
	Refund(n int)         // escrow seam: credit the purse back (on outbid)
	Wallet() *NpcWallet   // synthetic regen wallet for persistence/regen; nil for real-gold buyers
	Flavor() string       // trailing phrase in the win broadcast, e.g. "for their collection"
}
```

Add three delegating one-liners to each regen archetype (place each next to its existing `Wallet()`):

```go
func (c *collector) CanAfford(n int) bool { return c.wallet.CanAfford(n) }
func (c *collector) Spend(n int)          { c.wallet.Spend(n) }
func (c *collector) Refund(n int)         { c.wallet.Refund(n) }
```
```go
func (c *craftsperson) CanAfford(n int) bool { return c.wallet.CanAfford(n) }
func (c *craftsperson) Spend(n int)          { c.wallet.Spend(n) }
func (c *craftsperson) Refund(n int)         { c.wallet.Refund(n) }
```
```go
func (a *adventurer) CanAfford(n int) bool { return a.wallet.CanAfford(n) }
func (a *adventurer) Spend(n int)          { a.wallet.Spend(n) }
func (a *adventurer) Refund(n int)         { a.wallet.Refund(n) }
```

- [ ] **Step 2: Switch the escrow call sites + nil-guard the wallet-only loops in `auctions.go`**

`nextNpcBid` (in `npc_buyers.go`, ~line 143): change the affordability check
```go
		if !b.Wallet().CanAfford(next) {
```
to
```go
		if !b.CanAfford(next) {
```

`npcBid` (in `auctions.go`, ~line 781): change
```go
	buyer.Wallet().Spend(bid)
```
to
```go
	buyer.Spend(bid)
```

`refundPreviousBidder` (in `auctions.go`, ~line 797): change
```go
			b.Wallet().Refund(a.HighestBid)
```
to
```go
			b.Refund(a.HighestBid)
```

Regen loop (in `auctions.go`, ~line 588-590): guard for nil wallet
```go
			for _, b := range npcBuyers {
				if w := b.Wallet(); w != nil {
					w.Regen(collectorRegenPerTick)
				}
			}
```

Persistence save (in `auctions.go`, ~line 164-166): guard for nil wallet
```go
	for _, b := range npcBuyers {
		if w := b.Wallet(); w != nil {
			mod.auctionMgr.WalletBalances[b.Name()] = w.Balance
		}
	}
```

Persistence load (in `auctions.go`, ~line 117-121): guard for nil wallet
```go
	for _, b := range npcBuyers {
		if w := b.Wallet(); w != nil {
			if bal, ok := mod.auctionMgr.WalletBalances[b.Name()]; ok {
				w.Balance = bal
			}
		}
	}
```

- [ ] **Step 3: Build + run the existing auctions suite (must stay green — pure refactor)**

Run: `go test ./modules/auctions/...`
Expected: PASS (all existing tests, unchanged — the three archetypes now satisfy the wider interface
and every escrow path routes through identical wallet calls).

- [ ] **Step 4: Commit**

```bash
git add modules/auctions/npc_buyers.go modules/auctions/auctions.go
git commit -m "refactor(auctions): move escrow seam (CanAfford/Spend/Refund) onto NpcBuyer"
```

---

### Task 2: The `shopkeeper` archetype (valuation + real-gold purse)

**Files:**
- Modify: `modules/auctions/npc_buyers.go`
- Test: `modules/auctions/npc_buyers_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `modules/auctions/npc_buyers_test.go`. Add these imports at the top of the file if missing:
`"github.com/GoMudEngine/GoMud/internal/shops"`.

```go
func newShopkeeperTestItem(t *testing.T) (items.Item, func()) {
	t.Helper()
	cleanup := items.SeedItemsForTest(map[int]*items.ItemSpec{
		901: {ItemId: 901, Name: "iron blade", Type: items.Weapon, Value: 1000,
			VendorCategories: []string{shops.CraftSupportBlacksmithing}},
		902: {ItemId: 902, Name: "no-vendor trinket", Type: items.Ring, Value: 1000},
	})
	return items.New(901), cleanup
}

func TestShopkeeper_InterestedAndMaxBid(t *testing.T) {
	item, cleanup := newShopkeeperTestItem(t)
	defer cleanup()

	shops.ClearCache()
	defer shops.ClearCache()
	// A blacksmith with ample gold matches; a tailor with the same gold does not.
	smith := shops.RegisterShop("testzone", 1, 100, shops.ShopInventory{
		Gold: 5000, StartingGold: 5000, CraftSupport: shops.CraftSupportBlacksmithing})
	shops.RegisterShop("testzone", 2, 100, shops.ShopInventory{
		Gold: 5000, StartingGold: 5000, CraftSupport: shops.CraftSupportTailoring})

	sk := &shopkeeper{name: "The Merchants' Guild"}
	if !sk.Interested(item) {
		t.Fatal("shopkeeper should want a blacksmith item when a blacksmith shop has gold")
	}
	// MaxBid must equal what that shop would pay over its own counter.
	want := shops.EvaluateBuyRules(item, smith, "", false, shops.PricingConfigFromBalance(), nil).Price
	if want <= 0 {
		t.Fatalf("test setup: expected a positive counter offer, got %d", want)
	}
	if got := sk.MaxBid(item); got != want {
		t.Errorf("MaxBid=%d want %d (EvaluateBuyRules price)", got, want)
	}
}

func TestShopkeeper_NoMatchingShop(t *testing.T) {
	item, cleanup := newShopkeeperTestItem(t)
	defer cleanup()
	shops.ClearCache()
	defer shops.ClearCache()
	shops.RegisterShop("testzone", 2, 100, shops.ShopInventory{
		Gold: 5000, StartingGold: 5000, CraftSupport: shops.CraftSupportTailoring})

	sk := &shopkeeper{name: "The Merchants' Guild"}
	if sk.Interested(item) {
		t.Error("shopkeeper should NOT want an item no live shop's discipline accepts")
	}
	if got := sk.MaxBid(item); got != 0 {
		t.Errorf("MaxBid=%d want 0 when uninterested", got)
	}
}

func TestShopkeeper_EscrowUsesRealShopGold(t *testing.T) {
	item, cleanup := newShopkeeperTestItem(t)
	defer cleanup()
	shops.ClearCache()
	defer shops.ClearCache()
	smith := shops.RegisterShop("testzone", 1, 100, shops.ShopInventory{
		Gold: 5000, StartingGold: 5000, CraftSupport: shops.CraftSupportBlacksmithing})

	// No disk in unit tests.
	orig := saveShopFn
	saveShopFn = func(zone string, mobId, roomId int) error { return nil }
	defer func() { saveShopFn = orig }()

	sk := &shopkeeper{name: "The Merchants' Guild"}
	if !sk.Interested(item) { // computes + memoizes the selection
		t.Fatal("precondition: shopkeeper should be interested")
	}
	if !sk.CanAfford(400) {
		t.Fatal("shop with 5000 gold / 2500 reserve should afford 400")
	}
	sk.Spend(400)
	if smith.Gold != 4600 {
		t.Errorf("after Spend(400) shop gold=%d want 4600", smith.Gold)
	}
	sk.Refund(400) // outbid -> restore
	if smith.Gold != 5000 {
		t.Errorf("after Refund(400) shop gold=%d want 5000", smith.Gold)
	}
}
```

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./modules/auctions/ -run TestShopkeeper -v`
Expected: FAIL — `undefined: shopkeeper`, `undefined: saveShopFn`.

- [ ] **Step 3: Implement the shopkeeper type**

Add to `modules/auctions/npc_buyers.go`. First the imports (add to the import block):

```go
import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/shops"
	"github.com/GoMudEngine/GoMud/internal/uuid"
)
```

Add the toggle + save seam near the other provisional knobs:

```go
	shopkeeperEnabled = true // gated by AuctionShopkeeperEnabled config

	// saveShopFn persists a shop after the shopkeeper mutates its gold/stock.
	// Overridable in tests to avoid disk I/O.
	saveShopFn = shops.SaveShop
)

// reserveRatio returns the shop gold-reserve fraction from config (0.50 fallback).
func reserveRatio() float64 {
	r := float64(configs.GetBalanceConfig().ShopGoldReserveRatio)
	if r <= 0 {
		r = 0.50
	}
	return r
}

func persistShop(inv *shops.ShopInventory) {
	if err := saveShopFn(inv.Zone, inv.MobId, inv.RoomId); err != nil {
		mudlog.Error("auctions.shopkeeper", "msg", "SaveShop failed", "error", err)
	}
}
```
> Note: the `var (...)` block above already opens earlier in the file at the existing knobs; append
> `shopkeeperEnabled`/`saveShopFn` inside it and close it before the `func reserveRatio`.

Then the type (place after the adventurer archetype):

```go
// ── Shopkeeper archetype: bids from a REAL shop's gold on items that shop
// deals in (VendorCategories ↔ CraftSupport), then relists the win into that
// shop's stock. Thin adapter over shops.EvaluateBuyRules. ──

// shopSel is the shopkeeper's memoized per-item shop selection.
type shopSel struct {
	uuid  uuid.UUID
	shop  *shops.ShopInventory
	offer int // best affordable BuyOffer.Price across live shops (0 = none)
}

type shopkeeper struct {
	name  string
	sel   shopSel              // memoized selection for the current decision
	bound *shops.ShopInventory // shop escrowed against while high bidder
}

func (s *shopkeeper) Name() string   { return s.name }
func (s *shopkeeper) Flavor() string { return "for the shelves" }
func (s *shopkeeper) Wallet() *NpcWallet { return nil } // real-gold buyer

// selectFor picks the shop with the highest affordable counter-offer for item.
// Memoized by item UUID within a single decision (Interested→MaxBid→CanAfford
// run for the same item on the single-threaded auction tick).
func (s *shopkeeper) selectFor(item items.Item) shopSel {
	if !item.UUID.IsNil() && s.sel.uuid == item.UUID {
		return s.sel
	}
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

func (s *shopkeeper) Interested(item items.Item) bool {
	if !shopkeeperEnabled {
		return false
	}
	return s.selectFor(item).offer > 0
}

func (s *shopkeeper) MaxBid(item items.Item) int { return s.selectFor(item).offer }

func (s *shopkeeper) CanAfford(n int) bool {
	if s.sel.shop == nil {
		return false
	}
	return s.sel.shop.CanAfford(n, s.sel.shop.GoldReserve(reserveRatio()))
}

func (s *shopkeeper) Spend(n int) {
	s.bound = s.sel.shop // freeze the binding for refund/win
	if s.bound == nil {
		return
	}
	s.bound.Gold -= n
	persistShop(s.bound)
}

func (s *shopkeeper) Refund(n int) {
	if s.bound == nil {
		return
	}
	s.bound.Gold += n
	persistShop(s.bound)
}
```

- [ ] **Step 4: Run the shopkeeper tests**

Run: `go test ./modules/auctions/ -run TestShopkeeper -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add modules/auctions/npc_buyers.go modules/auctions/npc_buyers_test.go
git commit -m "feat(auctions): shopkeeper NPC buyer valuation + real-shop-gold purse"
```

---

### Task 3: Win → relist into the bound shop

**Files:**
- Modify: `modules/auctions/npc_buyers.go` (add `Receive`)
- Modify: `modules/auctions/auctions.go` (wire the receiver into the NPC-win block)
- Test: `modules/auctions/npc_buyers_test.go`

- [ ] **Step 1: Write the failing test**

Append to `modules/auctions/npc_buyers_test.go`:

```go
func TestShopkeeper_WinRelistsIntoBoundShop(t *testing.T) {
	item, cleanup := newShopkeeperTestItem(t)
	defer cleanup()
	shops.ClearCache()
	defer shops.ClearCache()
	smith := shops.RegisterShop("testzone", 1, 100, shops.ShopInventory{
		Gold: 5000, StartingGold: 5000, CraftSupport: shops.CraftSupportBlacksmithing})
	other := shops.RegisterShop("testzone", 9, 100, shops.ShopInventory{
		Gold: 5000, StartingGold: 5000, CraftSupport: shops.CraftSupportBlacksmithing})

	orig := saveShopFn
	saveShopFn = func(zone string, mobId, roomId int) error { return nil }
	defer func() { saveShopFn = orig }()

	sk := &shopkeeper{name: "The Merchants' Guild"}
	sk.Interested(item) // select the best shop
	sk.Spend(300)       // bind to the chosen shop

	// Capture the binding before Receive clears it (test is in-package).
	bound := sk.bound
	if bound == nil {
		t.Fatal("expected a bound shop after Spend")
	}

	var r auctionWinReceiver = sk // the receiver contract used by the resolution
	r.Receive(item)

	if len(bound.AffixedStock) != 1 || bound.AffixedStock[0].Item.ItemId != 901 {
		t.Errorf("won item not relisted into bound shop: %+v", bound.AffixedStock)
	}
	if bound.BuysCount != 1 {
		t.Errorf("BuysCount=%d want 1", bound.BuysCount)
	}
	// The non-bound shop must be untouched.
	notBound := smith
	if bound == smith {
		notBound = other
	}
	if len(notBound.AffixedStock) != 0 {
		t.Errorf("non-bound shop should not receive the item")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./modules/auctions/ -run TestShopkeeper_WinRelists -v`
Expected: FAIL — `undefined: auctionWinReceiver`, `sk.Receive undefined`.

- [ ] **Step 3: Implement `Receive` + the receiver interface**

Add to `modules/auctions/npc_buyers.go`:

```go
// auctionWinReceiver lets a buyer take custody of the item it won (instead of
// the default sink). Only the shopkeeper implements it.
type auctionWinReceiver interface {
	Receive(item items.Item)
}

// Receive routes a won lot into the bound shop's resale stock. AddAffixedStock
// holds the full item instance, so exact affixes/enchants survive and the item
// becomes purchasable — mirroring the counter-buyback path in actions/sell.go.
func (s *shopkeeper) Receive(item items.Item) {
	if s.bound == nil {
		return
	}
	cap := int(configs.GetBalanceConfig().ShopAffixedStockCap)
	s.bound.AddAffixedStock(item, item.GetSpec().Value, cap)
	s.bound.BuysCount++
	persistShop(s.bound)
	s.bound = nil
}
```

- [ ] **Step 4: Wire the receiver into the NPC-win block in `auctions.go`**

In the NPC-winner block (~line 498-502), change:
```go
			if auctionNow.HighestBidIsNPC {
				flavor := "for their collection"
				if b := buyerByName(auctionNow.HighestBidderName); b != nil {
					flavor = b.Flavor()
				}
```
to:
```go
			if auctionNow.HighestBidIsNPC {
				flavor := "for their collection"
				if b := buyerByName(auctionNow.HighestBidderName); b != nil {
					flavor = b.Flavor()
					if r, ok := b.(auctionWinReceiver); ok {
						r.Receive(auctionNow.ItemData) // shopkeeper relists; others no-op sink
					}
				}
```

- [ ] **Step 5: Run the relist test + full module suite**

Run: `go test ./modules/auctions/...`
Expected: PASS (new relist test + all prior).

- [ ] **Step 6: Commit**

```bash
git add modules/auctions/npc_buyers.go modules/auctions/auctions.go modules/auctions/npc_buyers_test.go
git commit -m "feat(auctions): shopkeeper win relists won item into bound shop stock"
```

---

### Task 4: Register the persona + config toggle

**Files:**
- Modify: `modules/auctions/npc_buyers.go` (registry)
- Modify: `modules/auctions/auctions.go` (`load()` reads `AuctionShopkeeperEnabled`)
- Test: `modules/auctions/npc_buyers_test.go`

- [ ] **Step 1: Write the failing test**

Append to `modules/auctions/npc_buyers_test.go`:

```go
func TestShopkeeper_RegisteredAndResolvable(t *testing.T) {
	if buyerByName("The Merchants' Guild") == nil {
		t.Fatal("shopkeeper persona must be registered so refunds/flavor/receive resolve by name")
	}
}

func TestShopkeeper_DisabledIsUninterested(t *testing.T) {
	item, cleanup := newShopkeeperTestItem(t)
	defer cleanup()
	shops.ClearCache()
	defer shops.ClearCache()
	shops.RegisterShop("testzone", 1, 100, shops.ShopInventory{
		Gold: 5000, StartingGold: 5000, CraftSupport: shops.CraftSupportBlacksmithing})

	shopkeeperEnabled = false
	defer func() { shopkeeperEnabled = true }()
	sk := &shopkeeper{name: "The Merchants' Guild"}
	if sk.Interested(item) {
		t.Error("disabled shopkeeper must not be interested")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./modules/auctions/ -run TestShopkeeper_Registered -v`
Expected: FAIL — `buyerByName` returns nil (persona not yet registered).

- [ ] **Step 3: Register the persona**

In `modules/auctions/npc_buyers.go`, add to the `npcBuyers` slice:
```go
var npcBuyers = []NpcBuyer{
	&collector{name: "Collector Veyd", wallet: &NpcWallet{Balance: 10000, Cap: 10000}},
	&collector{name: "Lady Ashcombe", wallet: &NpcWallet{Balance: 10000, Cap: 10000}},
	&craftsperson{name: "Master Ordwin", wallet: &NpcWallet{Balance: 6000, Cap: 6000}},
	&adventurer{name: "Sellsword Kest", wallet: &NpcWallet{Balance: 6000, Cap: 6000}},
	&shopkeeper{name: "The Merchants' Guild"},
}
```

- [ ] **Step 4: Read the config toggle in `load()`**

In `modules/auctions/auctions.go` `load()`, alongside the other NPC-buyer knobs (~line 132), add:
```go
	if v, ok := mod.plug.Config.Get(`AuctionShopkeeperEnabled`).(bool); ok {
		shopkeeperEnabled = v
	}
```

- [ ] **Step 5: Run the tests**

Run: `go test ./modules/auctions/ -run TestShopkeeper -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add modules/auctions/npc_buyers.go modules/auctions/auctions.go modules/auctions/npc_buyers_test.go
git commit -m "feat(auctions): register shopkeeper persona + AuctionShopkeeperEnabled toggle"
```

---

### Task 5: Full verification + patch notes

**Files:**
- Modify: `PATCH_NOTES.md`
- Modify: `docs/PATH_TO_1.0.md` (mark #2.4 + #3)

- [ ] **Step 1: Full test suite**

Run: `go test ./...`
Expected: PASS across the repo (record the count; prior baseline was 91/91 in `modules/auctions` plus
the repo suite). Investigate any failure before proceeding — do not paper over.

- [ ] **Step 2: Boot smoke test (per CLAUDE.md pre-push SOP)**

First wipe instance saves (do NOT touch `shops/`):
```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
```
Then build + boot and watch for a clean load past data files (no panics):
```bash
go build ./... && ./<server-binary>   # or the project's usual run command
```
Expected: server reaches `mobs.LoadDataFiles()`, `quests.LoadDataFiles()`, etc. without panic;
auctions module loads. Ctrl-C after a clean boot.

- [ ] **Step 3: Add patch notes**

Append a dated entry to `PATCH_NOTES.md` describing the shopkeeper buyer (real shop gold, relists the
win into shop stock, `AuctionShopkeeperEnabled` toggle).

- [ ] **Step 4: Mark the roadmap**

In `docs/PATH_TO_1.0.md`, flip **#2.4 Shopkeeper** to ✅ and note **#3 relisting** was folded in
(done), with the min-auction-value floor left as a followup.

- [ ] **Step 5: Commit**

```bash
git add PATCH_NOTES.md docs/PATH_TO_1.0.md
git commit -m "docs(auctions): patch notes + roadmap for shopkeeper buyer (econ #2.4 + #3)"
```

---

## Notes for the implementer

- **Single-threaded assumption:** the auction update tick runs on one goroutine, so the shopkeeper's
  per-decision memoization (`sel`) and frozen binding (`bound`) are race-free. Don't add locking.
- **Reserve invariant:** `EvaluateBuyRules` already rejects any offer that would breach the shop's gold
  reserve at its current gold, and refunds restore gold before the next `Spend`, so every bid the
  shopkeeper places is reserve-safe. Do not add a second reserve check in `Spend`.
- **`Wallet()==nil` is load-bearing:** it is what keeps the shopkeeper out of the regen + persistence
  loops. Keep those loops guarded.
- **Don't wipe `_datafiles/world/dogmud/shops/`** during smoke tests — that's persistent economy state
  (CLAUDE.md).
