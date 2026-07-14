# Auction Mechanics Core Implementation Plan (Econ Loop #1)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give the auction house a sound economic core — escrowed gold flow, seller buyout with a derived reserve, a commission sink, and reliable unsold-item return.

**Architecture:** All changes live in the single-file module `modules/auctions/auctions.go`. Pure math helpers (`reserveFrom`, `commissionFor`) are unit-tested; the `AuctionManager.StartAuction`/`Bid` methods are unit-tested with seeded users; the `newRoundHandler` resolution (I/O-heavy) is covered by the helpers + build + manual.

**Tech Stack:** Go; `modules/auctions` (plugin), `Character.Bank int`, `user.ItemStorage.AddItem`, `user.Inbox.Add(users.Message{...})`, `events.EquipmentChange{BankChange}`, `mod.plug.Config.Get`.

**Spec:** `docs/superpowers/specs/2026-07-14-auction-mechanics-core-design.md`

> **Test harness note:** the manager methods use `users.GetByUserId`. Confirm `users.NewTestUser(id, ...)` registers the user so `GetByUserId` finds it (the hooks tests rely on this). Set a test user's gold via `u.Character.Bank = N`.

---

## File map

- **Modify** `modules/auctions/auctions.go` — `AuctionItem.BuyoutPrice`, `reserveFrom`/`commissionFor` helpers, `StartAuction`, `Bid`, `newRoundHandler` resolution, config readers, the listing prompt.
- **Create** `modules/auctions/auctions_test.go` — unit tests for the helpers + StartAuction + Bid.
- **Modify** `PATCH_NOTES.md`.

---

## Task 1: Buyout field + reserve math

**Files:** Modify `modules/auctions/auctions.go`; Create `modules/auctions/auctions_test.go`

- [ ] **Step 1: Write the failing test**

Create `modules/auctions/auctions_test.go`:

```go
package auctions

import "testing"

func TestReserveFrom(t *testing.T) {
	// reserve = 25% of buyout, min 1
	if got := reserveFrom(1000, 0.25); got != 250 {
		t.Errorf("reserveFrom(1000,0.25)=%d want 250", got)
	}
	if got := reserveFrom(2, 0.25); got != 1 { // rounds down to 0 -> min 1
		t.Errorf("reserveFrom(2,0.25)=%d want 1 (floor min)", got)
	}
	if got := reserveFrom(0, 0.25); got != 1 {
		t.Errorf("reserveFrom(0,..)=%d want 1", got)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./modules/auctions/ -run TestReserveFrom -v`
Expected: FAIL — `undefined: reserveFrom`.

- [ ] **Step 3: Add the field + helper + StartAuction change**

In `modules/auctions/auctions.go`, add `BuyoutPrice int` to the `AuctionItem` struct (near `MinimumBid`).

Add the pure helper (top-level):
```go
// reserveFrom derives the reserve / minimum bid from a buyout price. Floors at 1.
func reserveFrom(buyout int, pct float64) int {
	r := int(float64(buyout) * pct)
	if r < 1 {
		r = 1
	}
	return r
}
```

Change `StartAuction` to take a buyout and derive the reserve. Update the signature and body:
```go
func (am *AuctionManager) StartAuction(item items.Item, userId int, buyout int, durationSeconds int, anon bool) bool {
	if am.ActiveAuction != nil {
		return false
	}
	if u := users.GetByUserId(userId); u != nil {
		am.ActiveAuction = &AuctionItem{
			ItemData:          item,
			SellerUserId:      userId,
			SellerName:        u.Character.Name,
			Anonymous:         anon,
			EndTime:           time.Now().Add(time.Second * time.Duration(durationSeconds)),
			BuyoutPrice:       buyout,
			MinimumBid:        reserveFrom(buyout, auctionReservePct),
			HighestBid:        0,
			HighestBidUserId:  0,
			HighestBidderName: ``,
		}
		return true
	}
	return false
}
```
Add a package-level default `var auctionReservePct = 0.25` for now (Task 4 wires it to config).

- [ ] **Step 4: Add a StartAuction test**

Append to `auctions_test.go`:
```go
func TestStartAuction_DerivesReserve(t *testing.T) {
	u := users.NewTestUser(9001, "seller", "Seller", 9901) // adjust to the real NewTestUser signature
	_ = u
	am := &AuctionManager{}
	if !am.StartAuction(items.Item{ItemId: 1}, 9001, 1000, 60, false) {
		t.Fatal("StartAuction should succeed for a valid user")
	}
	if am.ActiveAuction.BuyoutPrice != 1000 {
		t.Errorf("BuyoutPrice=%d want 1000", am.ActiveAuction.BuyoutPrice)
	}
	if am.ActiveAuction.MinimumBid != 250 {
		t.Errorf("MinimumBid(reserve)=%d want 250", am.ActiveAuction.MinimumBid)
	}
}
```
Add the imports (`users`, `items`). Verify the real `users.NewTestUser` signature and `items.Item` literal; adjust as needed so the user is findable by `GetByUserId(9001)`.

- [ ] **Step 5: Run tests**

Run: `go test ./modules/auctions/ -run 'TestReserveFrom|TestStartAuction' -v`
Expected: PASS. Then fix the two existing `StartAuction(...)` call sites (the listing command ~line 321) to pass `buyout` — for now pass the existing `amt` (renamed in Task 4). Build: `go build ./modules/auctions/`.

- [ ] **Step 6: Commit**

```bash
git add modules/auctions/auctions.go modules/auctions/auctions_test.go
git commit -m "feat(auctions): buyout price + derived reserve"
```

---

## Task 2: Escrow + affordability + buy-it-now in Bid

**Files:** Modify `modules/auctions/auctions.go`; `modules/auctions/auctions_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `auctions_test.go`:
```go
func TestBid_EscrowsAndRefunds(t *testing.T) {
	a := users.NewTestUser(9010, "abid", "Abid", 9910)
	b := users.NewTestUser(9011, "bbid", "Bbid", 9911)
	a.Character.Bank = 1000
	b.Character.Bank = 1000
	seller := users.NewTestUser(9012, "sell2", "Sell2", 9912)
	_ = seller

	am := &AuctionManager{}
	am.StartAuction(items.Item{ItemId: 1}, 9012, 1000, 60, false) // buyout 1000, reserve 250

	// Below reserve -> rejected.
	if err := am.Bid(9010, 100); err == nil {
		t.Fatal("bid below reserve should be rejected")
	}
	// Can't afford -> rejected.
	a.Character.Bank = 10
	if err := am.Bid(9010, 300); err == nil {
		t.Fatal("bid above bank should be rejected")
	}
	a.Character.Bank = 1000

	// Valid bid escrows (bank down by bid).
	if err := am.Bid(9010, 300); err != nil {
		t.Fatalf("valid bid rejected: %v", err)
	}
	if a.Character.Bank != 700 {
		t.Errorf("A bank=%d want 700 (300 escrowed)", a.Character.Bank)
	}

	// B outbids -> A refunded, B escrowed.
	if err := am.Bid(9011, 400); err != nil {
		t.Fatalf("outbid rejected: %v", err)
	}
	if a.Character.Bank != 1000 {
		t.Errorf("A bank=%d want 1000 (refunded)", a.Character.Bank)
	}
	if b.Character.Bank != 600 {
		t.Errorf("B bank=%d want 600 (400 escrowed)", b.Character.Bank)
	}
}

func TestBid_BuyItNow(t *testing.T) {
	b := users.NewTestUser(9020, "buyer", "Buyer", 9920)
	b.Character.Bank = 5000
	seller := users.NewTestUser(9021, "sell3", "Sell3", 9921)
	_ = seller
	am := &AuctionManager{}
	am.StartAuction(items.Item{ItemId: 1}, 9021, 1000, 600, false)

	if err := am.Bid(9020, 5000); err != nil { // >= buyout
		t.Fatalf("buyout bid rejected: %v", err)
	}
	if am.ActiveAuction.HighestBid != 1000 {
		t.Errorf("buyout should cap bid at 1000, got %d", am.ActiveAuction.HighestBid)
	}
	if !am.ActiveAuction.IsEnded() {
		t.Error("a buyout bid should end the lot immediately")
	}
	if b.Character.Bank != 4000 {
		t.Errorf("buyer bank=%d want 4000 (1000 escrowed)", b.Character.Bank)
	}
}
```

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./modules/auctions/ -run TestBid -v`
Expected: FAIL (current `Bid` doesn't escrow / affordability / buyout).

- [ ] **Step 3: Rewrite `Bid`**

Replace the `Bid` method:
```go
func (am *AuctionManager) Bid(userId int, bid int) error {
	if am.ActiveAuction == nil {
		return errors.New("There is not an auction to bid on.")
	}
	if am.ActiveAuction.HighestBidUserId == userId {
		return errors.New("You are already the highest bidder.")
	}

	// Buy-it-now: a bid at/above buyout caps at buyout and ends the lot.
	buyNow := false
	if am.ActiveAuction.BuyoutPrice > 0 && bid >= am.ActiveAuction.BuyoutPrice {
		bid = am.ActiveAuction.BuyoutPrice
		buyNow = true
	}

	minBid := am.ActiveAuction.MinimumBid
	if am.ActiveAuction.HighestBid > 0 {
		minBid = am.ActiveAuction.HighestBid + 1
	}
	if bid < minBid {
		return fmt.Errorf(`The minimum bid is <ansi fg="gold">%d gold</ansi>`, minBid)
	}

	u := users.GetByUserId(userId)
	if u == nil {
		return errors.New("User not found.")
	}
	if u.Character.Bank < bid {
		return fmt.Errorf(`You only have <ansi fg="gold">%d gold</ansi> in the bank.`, u.Character.Bank)
	}

	// Escrow: debit the new bidder, refund the previous high bidder.
	if am.ActiveAuction.HighestBidUserId > 0 {
		refundUser(am.ActiveAuction.HighestBidUserId, am.ActiveAuction.HighestBid)
	}
	u.Character.Bank -= bid
	events.AddToQueue(events.EquipmentChange{UserId: u.UserId, BankChange: -bid})

	am.ActiveAuction.HighestBid = bid
	am.ActiveAuction.HighestBidUserId = userId
	am.ActiveAuction.HighestBidderName = u.Character.Name

	if buyNow {
		am.ActiveAuction.EndTime = time.Now()
	}
	return nil
}

// refundUser returns escrowed gold to a bidder (online or offline).
func refundUser(userId int, amount int) {
	if amount <= 0 {
		return
	}
	if u := users.GetByUserId(userId); u != nil {
		u.Character.Bank += amount
		events.AddToQueue(events.EquipmentChange{UserId: u.UserId, BankChange: amount})
		return
	}
	users.SearchOfflineUsers(func(u *users.UserRecord) bool {
		if u.UserId == userId {
			u.Character.Bank += amount
			users.SaveUser(*u)
			return false
		}
		return true
	})
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./modules/auctions/ -run TestBid -v`
Expected: PASS. Build: `go build ./modules/auctions/`.

- [ ] **Step 5: Commit**

```bash
git add modules/auctions/auctions.go modules/auctions/auctions_test.go
git commit -m "feat(auctions): escrow bids, affordability check, buy-it-now"
```

---

## Task 3: Commission on sale + reliable unsold-return (resolution)

**Files:** Modify `modules/auctions/auctions.go`; `modules/auctions/auctions_test.go`

- [ ] **Step 1: Write the failing test (pure helper)**

Append to `auctions_test.go`:
```go
func TestCommissionFor(t *testing.T) {
	if got := commissionFor(1000, 0.05); got != 50 {
		t.Errorf("commissionFor(1000,0.05)=%d want 50", got)
	}
	if got := commissionFor(10, 0.05); got != 0 { // rounds down
		t.Errorf("commissionFor(10,0.05)=%d want 0", got)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./modules/auctions/ -run TestCommissionFor -v`
Expected: FAIL — `undefined: commissionFor`.

- [ ] **Step 3: Add the helper + wire the resolution**

Add the pure helper:
```go
// commissionFor is the house cut of a sale (a gold sink). Rounds down.
func commissionFor(bid int, pct float64) int {
	return int(float64(bid) * pct)
}
```
Add a package-level default `var auctionCommissionPct = 0.05` (Task 4 wires config).

In `newRoundHandler`'s **sold** branch, the winner already paid via escrow (Task 2), so replace the
seller credit (the `sellerUser.Character.Bank += auctionNow.HighestBid` at ~line 407, and the offline
`Gold: auctionNow.HighestBid` in the inbox message at ~line 432) with the net-of-commission amount:
```go
	commission := commissionFor(auctionNow.HighestBid, auctionCommissionPct)
	payout := auctionNow.HighestBid - commission
	// ... sellerUser.Character.Bank += payout
	// ... offline inbox Gold: payout
	// (the EquipmentChange BankChange also uses payout)
```
Add a short note in the seller message that a commission was withheld (no raw numbers required, but
"(after the auction house's cut)" is fine).

In the **unsold** branch (`else if auctionNow.SellerUserId > 0`, ~line 442), make the return reliable
for online AND offline sellers, preferring bank storage:
```go
	returnUnsoldItem(auctionNow.SellerUserId, auctionNow.ItemData)
```
and add the helper:
```go
// auctionStorageCap bounds how many items an unsold-return may push into storage
// before falling back to the inbox (bank storage cap is room-based; use a flat
// cap here since the seller may be offline / not at a bank).
const auctionStorageCap = 20

// returnUnsoldItem returns an unsold lot to its seller — bank storage if there's
// room, else the inbox. Works whether the seller is online or offline.
func returnUnsoldItem(sellerUserId int, item items.Item) {
	deliver := func(u *users.UserRecord, offline bool) {
		if u.ItemStorage.SlotCount() < auctionStorageCap {
			u.ItemStorage.AddItem(item)
		} else {
			u.Inbox.Add(users.Message{
				FromName: `Auction System`,
				Message:  `Your auction ended with no bids. The item is returned.`,
				Item:     &item,
			})
		}
		if offline {
			users.SaveUser(*u)
		}
	}
	if u := users.GetByUserId(sellerUserId); u != nil {
		deliver(u, false)
		u.SendText(messaging.CategorySystem, `<ansi fg="yellow">Your auction ended with no bids; the item was returned to your bank storage.</ansi>`)
		return
	}
	users.SearchOfflineUsers(func(u *users.UserRecord) bool {
		if u.UserId == sellerUserId {
			deliver(u, true)
			return false
		}
		return true
	})
}
```
Replace the old online-only backpack return (`user.Character.StoreItem(...)`) with the call above.

- [ ] **Step 4: Run tests + build**

Run: `go test ./modules/auctions/ -run 'TestCommissionFor' -v && go build ./modules/auctions/`
Expected: PASS + build clean.

- [ ] **Step 5: Commit**

```bash
git add modules/auctions/auctions.go modules/auctions/auctions_test.go
git commit -m "feat(auctions): commission sink on sale + reliable unsold-return to storage/inbox"
```

---

## Task 4: Config knobs + listing prompt (buyout)

**Files:** Modify `modules/auctions/auctions.go`

- [ ] **Step 1: Read the config knobs (with defaults)**

At the top of `newRoundHandler` and `auctionCommand` (or once in `load`), set the package-level
`auctionReservePct`/`auctionCommissionPct` from plugin config, mirroring the existing
`DurationSeconds`/`Anonymous` reads:
```go
	if v, ok := mod.plug.Config.Get(`AuctionReservePct`).(float64); ok && v > 0 {
		auctionReservePct = v
	}
	if v, ok := mod.plug.Config.Get(`AuctionCommissionPct`).(float64); ok && v >= 0 {
		auctionCommissionPct = v
	}
```
(Do this in the module `load` callback so it's set once at startup.)

- [ ] **Step 2: Retarget the listing prompt to buyout**

At `modules/auctions/auctions.go` ~line 295, change the prompt text and variable meaning:
```go
	questionAmount := cmdPrompt.Ask(`Set a buyout (buy-it-now) price?`, []string{})
	// ... amt is now the buyout ...
	// confirmation text: "Auctioning your X — buyout N gold, reserve M gold."
```
and change the `StartAuction(matchItem, user.UserId, amt, duration, anonymous)` call so `amt` is the
buyout (already the case once renamed). Show the derived reserve (`reserveFrom(amt, auctionReservePct)`)
in the confirmation.

- [ ] **Step 3: Build + full suite**

Run: `go build ./... && go test ./modules/auctions/ ./... 2>&1 | grep -E "^(FAIL|---)" | head`
Expected: build clean, no failures.

- [ ] **Step 4: Commit**

```bash
git add modules/auctions/auctions.go
git commit -m "feat(auctions): config knobs (reserve/commission) + buyout listing prompt"
```

---

## Task 5: Verify + patch notes

**Files:** Modify `PATCH_NOTES.md`

- [ ] **Step 1: Full build + suite + boot-smoke**

```bash
go build ./... && go test ./... 2>&1 | grep -E "^(FAIL|---)" | head   # expect none
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run .   # wait for "Server Ready", no panic
```

- [ ] **Step 2: Manual (two connections)**

List a lot (set a buyout), bid from a second character, get outbid (confirm the first bidder's bank
is refunded), win, and confirm the seller's bank rose by `bid − commission` and the item transferred.
Separately, let a lot expire with no bids and confirm the item returns to the seller's bank storage
(or inbox if full). Try a buy-it-now bid and confirm instant win.

- [ ] **Step 3: Patch notes**

Prepend to `PATCH_NOTES.md`:
```markdown
## 2026-07-14 — Auction house: real bidding economy

The auction house now handles gold properly. When you bid, the gold is set aside
from your bank; if you're outbid it comes straight back, and the winner actually
pays (no more free gold). Sellers set a buy-it-now price — pay it to win instantly
— and a lot only sells if bids clear a reserve, with a small house commission on
the sale. A lot that ends with no bids is returned to the seller's bank storage
(or mailbox), never lost.
```

- [ ] **Step 4: Commit**

```bash
git add PATCH_NOTES.md
git commit -m "docs(auctions): patch notes for auction economy overhaul"
```

---

## Final verification

- [ ] `go build ./...` clean.
- [ ] `go test ./modules/auctions/` passes; full suite `go test ./...` all ok.
- [ ] Boot-smoke clean (`Server Ready`, no panic).
- [ ] Manual: escrow/refund, commission split, buy-it-now, unsold-return all behave.
