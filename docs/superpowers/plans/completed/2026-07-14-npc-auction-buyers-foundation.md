# NPC Auction Buyers — Foundation + Collector Implementation Plan (Econ #2.1)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the NPC-bidder engine (non-user sentinel bidder + bid-decision tick + `NpcBuyer` framework) and prove it with the Collector archetype.

**Architecture:** A new `modules/auctions/npc_buyers.go` holds the `NpcBuyer` interface, `NpcWallet`, the collector, and the buyer registry. `modules/auctions/auctions.go` gets the sentinel field, `npcBid`, a user-`Bid` refund change, the tick decision + regen, the NPC-win resolution branch, config, and wallet persistence. Builds on the econ-#1 helpers (`getUser`, `refundUser`, `commissionFor`, `auctionCommissionPct`).

**Tech Stack:** Go; `modules/auctions`; `items.ItemType`/`GetSpec().Value`; `util.Rand`; the module plugin save/load.

**Spec:** `docs/superpowers/specs/2026-07-14-npc-auction-buyers-foundation-design.md`

> **Test seam:** reuse the `fakeUsers(...)` helper from `auctions_test.go` (econ #1) for user bidders. NPC buyers are constructed directly in tests (no global lookup needed).

---

## File map

- **Create** `modules/auctions/npc_buyers.go` — `NpcBuyer`, `NpcWallet`, `collector`, `isEquipment`, registry, `buyerByName`, `nextNpcBid`, provisional knobs.
- **Create** `modules/auctions/npc_buyers_test.go` — wallet + collector + `nextNpcBid` unit tests.
- **Modify** `modules/auctions/auctions.go` — `HighestBidIsNPC` field, `npcBid`, user `Bid` NPC-refund, tick decision + regen, NPC-win resolution, config reads, `WalletBalances` persistence.
- **Modify** `modules/auctions/auctions_test.go` — sentinel-flow tests.
- **Modify** `PATCH_NOTES.md`.

---

## Task 1: NpcBuyer framework + Collector

**Files:** Create `modules/auctions/npc_buyers.go`, `modules/auctions/npc_buyers_test.go`

- [ ] **Step 1: Write the failing tests**

Create `modules/auctions/npc_buyers_test.go`:
```go
package auctions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

func TestNpcWallet(t *testing.T) {
	w := &NpcWallet{Balance: 100, Cap: 200}
	if !w.CanAfford(100) || w.CanAfford(101) {
		t.Fatal("CanAfford wrong")
	}
	w.Spend(60)
	if w.Balance != 40 {
		t.Fatalf("Spend: balance=%d want 40", w.Balance)
	}
	w.Refund(1000) // clamps to cap
	if w.Balance != 200 {
		t.Fatalf("Refund clamp: balance=%d want 200", w.Balance)
	}
	w.Regen(1000) // clamps to cap
	if w.Balance != 200 {
		t.Fatalf("Regen clamp: balance=%d want 200", w.Balance)
	}
}

func TestCollector_InterestAndMaxBid(t *testing.T) {
	collectorMinValue = 500
	collectorPremium = 1.0
	c := &collector{name: "Veyd", wallet: &NpcWallet{Balance: 10000, Cap: 10000}}

	sword := items.Item{ItemId: 1} // needs a seeded spec — see note
	_ = sword

	// Use SeedItemsForTest to give known Type/Value.
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		101: {ItemId: 101, Name: "fine blade", Type: items.Weapon, Value: 800},
		102: {ItemId: 102, Name: "cheap dagger", Type: items.Weapon, Value: 100},
		103: {ItemId: 103, Name: "herb", Type: items.Potion, Value: 900},
	})()
	if !c.Interested(items.New(101)) {
		t.Error("collector should want a valuable weapon")
	}
	if c.Interested(items.New(102)) {
		t.Error("collector should NOT want a cheap weapon (below min value)")
	}
	if c.Interested(items.New(103)) {
		t.Error("collector should NOT want a non-equipment potion")
	}
	if got := c.MaxBid(items.New(101)); got != 800 {
		t.Errorf("MaxBid=%d want 800 (Value*1.0)", got)
	}
}
```
(Verify `items.SeedItemsForTest` + `items.New` signatures against econ-#1 usage / the pinnacle tests.)

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./modules/auctions/ -run 'TestNpcWallet|TestCollector' -v`
Expected: FAIL — undefined symbols.

- [ ] **Step 3: Write `npc_buyers.go`**

```go
package auctions

import "github.com/GoMudEngine/GoMud/internal/items"

// NpcBuyer is one living-world auction buyer archetype.
type NpcBuyer interface {
	Name() string
	Interested(item items.Item) bool
	MaxBid(item items.Item) int
	Wallet() *NpcWallet
}

// NpcWallet is a persistent, gold-gated balance (regenerates toward Cap).
type NpcWallet struct {
	Balance int `json:"balance"`
	Cap     int `json:"cap"`
}

func (w *NpcWallet) CanAfford(n int) bool { return w.Balance >= n }
func (w *NpcWallet) Spend(n int)          { w.Balance -= n }
func (w *NpcWallet) Refund(n int)         { w.add(n) }
func (w *NpcWallet) Regen(amount int)     { w.add(amount) }
func (w *NpcWallet) add(n int) {
	w.Balance += n
	if w.Balance > w.Cap {
		w.Balance = w.Cap
	}
}

// ── Provisional knobs (tuned in playtest; overridden from config in load) ──
var (
	npcBuyersEnabled = true
	npcBidChancePct  = 35 // percent chance per update tick that an interested NPC nudges the bid
	collectorMinValue = 500
	collectorPremium  = 1.0
	collectorWalletCap = 10000
	collectorRegenPerTick = 5 // gold per update tick (placeholder; config sets the real rate)
)

// isEquipment reports whether an item type is wearable/wieldable gear.
func isEquipment(t items.ItemType) bool {
	switch t {
	case items.Weapon, items.Offhand, items.Head, items.Neck, items.Body,
		items.Belt, items.Gloves, items.Ring, items.Wrist, items.Legs,
		items.Feet, items.Back, items.Shoulders:
		return true
	}
	return false
}

// ── Collector archetype ──
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

// ── Registry ──
var npcBuyers = []NpcBuyer{
	&collector{name: "Collector Veyd", wallet: &NpcWallet{Balance: 10000, Cap: 10000}},
	&collector{name: "Lady Ashcombe", wallet: &NpcWallet{Balance: 10000, Cap: 10000}},
}

func buyerByName(name string) NpcBuyer {
	for _, b := range npcBuyers {
		if b.Name() == name {
			return b
		}
	}
	return nil
}
```

> Verify the equipment `ItemType` constant names in `internal/items/itemspec.go` (Weapon/Head/…) and
> add any missing ones (e.g. `Legs`, `Feet`, `Back`, `Shoulders`, `ComponentBag`) if the exact set differs.

- [ ] **Step 4: Run tests**

Run: `go test ./modules/auctions/ -run 'TestNpcWallet|TestCollector' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add modules/auctions/npc_buyers.go modules/auctions/npc_buyers_test.go
git commit -m "feat(auctions): NpcBuyer framework + regenerating wallet + collector"
```

---

## Task 2: Sentinel bidder + npcBid + user-Bid refund

**Files:** Modify `modules/auctions/auctions.go`, `modules/auctions/auctions_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `auctions_test.go`:
```go
func TestNpcBid_SentinelAndRefunds(t *testing.T) {
	seller := users.NewTestUser(9030, "sell4", "Sell4", 9930)
	user := users.NewTestUser(9031, "usr", "Usr", 9931)
	user.Character.Bank = 10000
	defer fakeUsers(seller, user)()

	col := &collector{name: "TestCol", wallet: &NpcWallet{Balance: 5000, Cap: 5000}}
	col2 := &collector{name: "TestCol2", wallet: &NpcWallet{Balance: 5000, Cap: 5000}}
	prev := npcBuyers
	npcBuyers = []NpcBuyer{col, col2}
	defer func() { npcBuyers = prev }()

	am := &AuctionManager{}
	am.StartAuction(items.Item{ItemId: 1}, 9030, 1000, 60, false) // reserve 250

	// NPC bids -> sentinel high bidder, wallet debited.
	am.npcBid(col, 300)
	if !am.ActiveAuction.HighestBidIsNPC || am.ActiveAuction.HighestBidUserId != 0 || am.ActiveAuction.HighestBidderName != "TestCol" {
		t.Fatalf("npcBid should make TestCol the sentinel high bidder: %+v", am.ActiveAuction)
	}
	if col.wallet.Balance != 4700 {
		t.Errorf("col wallet=%d want 4700", col.wallet.Balance)
	}

	// A second NPC outbids -> first refunded.
	am.npcBid(col2, 400)
	if col.wallet.Balance != 5000 {
		t.Errorf("col refunded to %d want 5000", col.wallet.Balance)
	}

	// A user outbids the NPC -> NPC refunded, sentinel cleared.
	if err := am.Bid(9031, 500); err != nil {
		t.Fatalf("user bid rejected: %v", err)
	}
	if col2.wallet.Balance != 5000 {
		t.Errorf("col2 refunded to %d want 5000", col2.wallet.Balance)
	}
	if am.ActiveAuction.HighestBidIsNPC || am.ActiveAuction.HighestBidUserId != 9031 {
		t.Errorf("user bid should clear the NPC sentinel: %+v", am.ActiveAuction)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./modules/auctions/ -run TestNpcBid -v`
Expected: FAIL — `HighestBidIsNPC` / `npcBid` undefined.

- [ ] **Step 3: Add the field + `npcBid` + user-`Bid` change**

Add to the `AuctionItem` struct (near `HighestBidderName`):
```go
	HighestBidIsNPC bool // sentinel: high bid is an NPC (HighestBidUserId==0, name in HighestBidderName)
```

Add `npcBid`:
```go
// npcBid places a bid on behalf of an NPC buyer, escrowing from its wallet and
// refunding whoever held the high bid (user or NPC).
func (am *AuctionManager) npcBid(buyer NpcBuyer, bid int) {
	if am.ActiveAuction == nil {
		return
	}
	am.refundPreviousBidder()
	buyer.Wallet().Spend(bid)
	am.ActiveAuction.HighestBid = bid
	am.ActiveAuction.HighestBidUserId = 0
	am.ActiveAuction.HighestBidIsNPC = true
	am.ActiveAuction.HighestBidderName = buyer.Name()
}

// refundPreviousBidder returns the currently-held escrow to whoever holds the
// high bid — a user (bank) or an NPC (wallet). Safe when there is no bid.
func (am *AuctionManager) refundPreviousBidder() {
	a := am.ActiveAuction
	if a == nil || a.HighestBid <= 0 {
		return
	}
	if a.HighestBidIsNPC {
		if b := buyerByName(a.HighestBidderName); b != nil {
			b.Wallet().Refund(a.HighestBid)
		}
		return
	}
	if a.HighestBidUserId > 0 {
		refundUser(a.HighestBidUserId, a.HighestBid)
	}
}
```

In the existing user `Bid`, replace the escrow-refund block
```go
	if am.ActiveAuction.HighestBidUserId > 0 {
		refundUser(am.ActiveAuction.HighestBidUserId, am.ActiveAuction.HighestBid)
	}
```
with
```go
	am.refundPreviousBidder()
```
and, right after setting `HighestBidderName = u.Character.Name`, add:
```go
	am.ActiveAuction.HighestBidIsNPC = false
```

- [ ] **Step 4: Run tests**

Run: `go test ./modules/auctions/ -run 'TestBid|TestNpcBid' -v`
Expected: PASS (existing Bid tests still green + the new sentinel test).

- [ ] **Step 5: Commit**

```bash
git add modules/auctions/auctions.go modules/auctions/auctions_test.go
git commit -m "feat(auctions): sentinel NPC bidder + npcBid + unified refund path"
```

---

## Task 3: Bid-decision tick + wallet regen

**Files:** Modify `modules/auctions/auctions.go`, `modules/auctions/npc_buyers_test.go`

- [ ] **Step 1: Write the failing test (deterministic selection)**

Append to `npc_buyers_test.go`:
```go
func TestNextNpcBid(t *testing.T) {
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		201: {ItemId: 201, Name: "prize", Type: items.Weapon, Value: 800},
	})()
	collectorMinValue = 500
	collectorPremium = 1.0
	rich := &collector{name: "Rich", wallet: &NpcWallet{Balance: 5000, Cap: 5000}}
	broke := &collector{name: "Broke", wallet: &NpcWallet{Balance: 10, Cap: 5000}}

	item := items.New(201)

	// A fresh lot (no bid): the affordable interested buyer bids the reserve.
	b, bid, ok := nextNpcBid([]NpcBuyer{broke, rich}, item, 0, 250, "", false)
	if !ok || b.Name() != "Rich" || bid != 250 {
		t.Fatalf("expected Rich to bid 250, got %v %d %v", b, bid, ok)
	}
	// Current bid already at/above the buyer's MaxBid (800): nobody bids.
	if _, _, ok := nextNpcBid([]NpcBuyer{rich}, item, 800, 801, "Rich", true); ok {
		t.Fatal("nobody should bid past MaxBid / when already the high NPC")
	}
	// The current high NPC is skipped.
	if _, _, ok := nextNpcBid([]NpcBuyer{rich}, item, 300, 301, "Rich", true); ok {
		t.Fatal("the current high NPC must not bid against itself")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./modules/auctions/ -run TestNextNpcBid -v`
Expected: FAIL — `nextNpcBid` undefined.

- [ ] **Step 3: Add `nextNpcBid` + wire the tick + regen**

Add to `npc_buyers.go`:
```go
// nextNpcBid picks the first enabled buyer that should place a competing bid on
// the current lot, and the amount (one increment). Pure over the buyer set +
// current bid state; the caller applies the probability gate. Returns ok=false
// if no buyer should bid.
func nextNpcBid(buyers []NpcBuyer, item items.Item, highBid, minBid int, highName string, highIsNPC bool) (NpcBuyer, int, bool) {
	next := minBid
	if highBid > 0 {
		next = highBid + 1
	}
	for _, b := range buyers {
		if highIsNPC && b.Name() == highName {
			continue // already the high bidder
		}
		if !b.Interested(item) {
			continue
		}
		if next > b.MaxBid(item) {
			continue
		}
		if !b.Wallet().CanAfford(next) {
			continue
		}
		return b, next, true
	}
	return nil, 0, false
}
```

In `auctions.go` `newRoundHandler`, in the periodic **auction-update** branch (~line 523, after the
human notification loop), add the NPC pass:
```go
			// Living-world NPC buyers: regen wallets and maybe place a competing bid.
			if npcBuyersEnabled && auctionNow.BuyoutPrice > 0 {
				for _, b := range npcBuyers {
					b.Wallet().Regen(collectorRegenPerTick)
				}
				if util.Rand(100) < npcBidChancePct {
					if b, bid, ok := nextNpcBid(npcBuyers, auctionNow.ItemData, auctionNow.HighestBid, auctionNow.MinimumBid, auctionNow.HighestBidderName, auctionNow.HighestBidIsNPC); ok {
						mod.auctionMgr.npcBid(b, bid)
					}
				}
			}
```
Add the `internal/util` import if not present.

- [ ] **Step 4: Run tests + build**

Run: `go test ./modules/auctions/ -run TestNextNpcBid -v && go build ./modules/auctions/`
Expected: PASS + build clean.

- [ ] **Step 5: Commit**

```bash
git add modules/auctions/npc_buyers.go modules/auctions/npc_buyers_test.go modules/auctions/auctions.go
git commit -m "feat(auctions): NPC bid-decision tick + wallet regen"
```

---

## Task 4: NPC-win resolution

**Files:** Modify `modules/auctions/auctions.go`

- [ ] **Step 1: Handle the NPC-won case in resolution**

In `newRoundHandler`'s end branch, the sold path currently delivers the item to `HighestBidUserId`
(the winner) and pays the seller. When the winner is an NPC (`HighestBidIsNPC`), there is no user to
deliver to — the item **sinks** into the collection. Guard the winner-item-delivery block so it only
runs for a real user, and pay the seller either way:

- Wrap the existing "Give the item to the winner" block (`if auctionNow.HighestBidUserId > 0 { … }`)
  so it still runs for user winners (it already checks `> 0`, which is false for an NPC — good, no
  change needed there; the item is simply not delivered to anyone).
- The seller-payout block (`if auctionNow.SellerUserId > 0 { … payout … }`) already runs for any
  winner with `HighestBid > 0`. Confirm it triggers when `HighestBidIsNPC` (it keys off
  `HighestBidUserId != 0` in the outer branch — **this is the bug to fix**).

Fix the outer sold/unsold split: it currently branches on `if auctionNow.HighestBidUserId != 0`.
Change it to treat an NPC bid as "sold":
```go
	sold := auctionNow.HighestBidUserId != 0 || auctionNow.HighestBidIsNPC
	if sold {
		// ... winner delivery (user only, guarded by HighestBidUserId > 0) ...
		// ... seller payout (payout = HighestBid - commission) ...
		if auctionNow.HighestBidIsNPC {
			// Item sinks into the collector's collection — flavor broadcast.
			for _, uid := range users.GetOnlineUserIds() {
				if u := users.GetByUserId(uid); u != nil {
					if on := u.GetConfigOption(`auction`); on == nil || on.(bool) {
						u.SendText(messaging.CategoryBroadcast, fmt.Sprintf(`<ansi fg="yellow"><ansi fg="username">%s</ansi> has acquired the <ansi fg="item">%s</ansi> for their collection.</ansi>`, auctionNow.HighestBidderName, auctionNow.ItemData.DisplayName()))
					}
				}
			}
		}
	} else if auctionNow.SellerUserId > 0 {
		// ... returnUnsoldItem (unchanged) ...
	}
```
Locate the current `if auctionNow.HighestBidUserId != 0 {` that opens the sold branch (~line 363) and
replace its condition with the `sold` variable; add the NPC flavor broadcast at the end of the sold
branch. The seller payout already uses `payout` from Task 3 of econ #1.

- [ ] **Step 2: Build + full module test**

Run: `go build ./modules/auctions/ && go test ./modules/auctions/`
Expected: build clean, all tests pass.

- [ ] **Step 3: Commit**

```bash
git add modules/auctions/auctions.go
git commit -m "feat(auctions): NPC win resolution — seller paid, item sinks with flavor"
```

---

## Task 5: Config knobs + wallet persistence

**Files:** Modify `modules/auctions/auctions.go`

- [ ] **Step 1: Persist wallet balances**

Add to the `AuctionManager` struct:
```go
	WalletBalances map[string]int `yaml:"WalletBalances,omitempty"` // persona name -> wallet balance
```

In `save()` (before `WriteStruct`), snapshot live wallets:
```go
	mod.auctionMgr.WalletBalances = map[string]int{}
	for _, b := range npcBuyers {
		mod.auctionMgr.WalletBalances[b.Name()] = b.Wallet().Balance
	}
```
In `load()` (after `ReadIntoStruct`), apply saved balances back to the live buyers:
```go
	for _, b := range npcBuyers {
		if bal, ok := mod.auctionMgr.WalletBalances[b.Name()]; ok {
			b.Wallet().Balance = bal
		}
	}
```

- [ ] **Step 2: Config knobs (with defaults) in `load()`**

Alongside the econ-#1 reserve/commission reads, add:
```go
	if v, ok := mod.plug.Config.Get(`AuctionNpcBuyersEnabled`).(bool); ok {
		npcBuyersEnabled = v
	}
	if v, ok := mod.plug.Config.Get(`AuctionNpcBidChance`).(float64); ok && v >= 0 {
		npcBidChancePct = int(v * 100)
	}
	if v, ok := mod.plug.Config.Get(`CollectorMinValue`).(int); ok && v > 0 {
		collectorMinValue = v
	}
	if v, ok := mod.plug.Config.Get(`CollectorPremium`).(float64); ok && v > 0 {
		collectorPremium = v
	}
	if v, ok := mod.plug.Config.Get(`CollectorWalletRegenPerHour`).(int); ok && v >= 0 {
		// convert per-hour to per-update-tick if you prefer; a flat per-tick is fine for v1.
		collectorRegenPerTick = v
	}
```
(Adjust types to what the plugin config exposes; keep the inline defaults.)

- [ ] **Step 3: Build + full suite**

Run: `go build ./... && go test ./modules/auctions/ ./... 2>&1 | grep -E "^(FAIL|---)" | head`
Expected: build clean, no failures.

- [ ] **Step 4: Commit**

```bash
git add modules/auctions/auctions.go
git commit -m "feat(auctions): NPC buyer config knobs + wallet persistence"
```

---

## Task 6: Verify + patch notes

**Files:** Modify `PATCH_NOTES.md`

- [ ] **Step 1: Full build + suite + boot-smoke**

```bash
go build ./... && go test ./... 2>&1 | grep -E "^(FAIL|---)" | head   # expect none
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run .   # wait for "Server Ready", no panic
```

- [ ] **Step 2: Manual**

List a prestige weapon (Value ≥ 500) with a buyout. Over a few update ticks, watch a collector place
competing bids up to ~its ceiling. Outbid it as a player and confirm it stops / the wallet is
refunded. Separately, list a prestige item and let a collector win it — confirm the seller is paid
net-of-commission and the "…acquired … for their collection" flavor line fires (item sinks).

- [ ] **Step 3: Patch notes**

Prepend to `PATCH_NOTES.md`:
```markdown
## 2026-07-14 — The auction house comes alive: collectors

The auction house now has interested buyers of its own. Wealthy collectors watch
the block and will bid on fine equipment they covet — competing with you up to
what they think it's worth, but never beyond, so anything you truly want you can
still win. When a collector does win, the piece disappears into their collection.
They spend real, limited coin, so they can't buy everything. (More kinds of NPC
buyers — merchants, adventurers, crafters — are coming.)
```

- [ ] **Step 4: Commit**

```bash
git add PATCH_NOTES.md
git commit -m "docs(auctions): patch notes for NPC collector buyers"
```

---

## Final verification

- [ ] `go build ./...` clean.
- [ ] `go test ./modules/auctions/` passes; full suite `go test ./...` all ok.
- [ ] Boot-smoke clean (`Server Ready`, no panic).
- [ ] Manual: collector competes up to ceiling, refunds on user outbid, wins + sinks with flavor,
  wallet gold-gated + persists.
