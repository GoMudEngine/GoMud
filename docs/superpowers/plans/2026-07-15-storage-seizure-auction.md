# Bank-storage Seizure Auction (econ #4) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** When a player can't pay bank-storage rent, seize their high-value stored items and
list them on the auction block (lien model) instead of deleting them; junk below a value
floor is still disposed.

**Architecture:** The storage-fee hook (`internal/hooks`) can't import the auctions plugin
(`modules/auctions`), so it emits a new `events.StorageItemSeized` event per seized slot.
The auctions module listens, enqueues each into a persisted `SeizedQueue`, and drains one lot
onto the single-slot block per free round. Seized lots reuse the existing escrow/NPC-buyer
bidding; at resolution a lien recoups the (tiny) unpaid rent and the surplus returns to the
ex-owner; an unsold seized lot is disposed (breaking the seize→unsold→re-seize loop).

**Tech Stack:** Go, GoMud event bus (`internal/events`), plugin config, testify, YAML persistence.

**Spec:** `docs/superpowers/specs/2026-07-15-storage-seizure-auction-design.md`

---

## File Structure

- **Create** `internal/configs/config.balance.shops_test.go` — validator default test.
- **Modify** `internal/configs/config.balance.go` — add `StorageSeizureMinValue` field.
- **Modify** `internal/configs/config.balance.shops.go` — default it in `validateShops()`.
- **Modify** `internal/events/eventtypes.go` — add `StorageItemSeized` event.
- **Create** `internal/events/storage_seized_test.go` — event `Type()` test.
- **Modify** `internal/hooks/StorageFee_MonthlyCharge.go` — floor-based dispose-vs-emit.
- **Create** `internal/hooks/StorageFee_seizure_test.go` — hook seizure behavior.
- **Modify** `modules/auctions/auctions.go` — `SeizedLot`, `SeizedQueue`, listener, drain,
  `AuctionItem.Seized/OwedLien/SeizedCount`, seized resolution (lien + stack delivery + dispose).
- **Modify** `modules/auctions/auctions_test.go` — enqueue / persist / drain / resolution tests.
- **Modify** `PATCH_NOTES.md`, `docs/PATH_TO_1.0.md` — at the end.

---

## Task 1: Config knob `StorageSeizureMinValue` (default 250)

**Files:**
- Modify: `internal/configs/config.balance.go:526` (struct field, next to `StorageFeePerItem`)
- Modify: `internal/configs/config.balance.shops.go` (`validateShops()`, STORAGE FEES block)
- Test: `internal/configs/config.balance.shops_test.go` (create)

- [ ] **Step 1: Write the failing test**

Create `internal/configs/config.balance.shops_test.go`:

```go
package configs

import "testing"

func TestValidateShops_StorageSeizureMinValueDefault(t *testing.T) {
	b := &Balance{}
	b.validateShops()
	if int(b.StorageSeizureMinValue) != 250 {
		t.Errorf("StorageSeizureMinValue default = %d, want 250", int(b.StorageSeizureMinValue))
	}
}

func TestValidateShops_StorageSeizureMinValuePreserved(t *testing.T) {
	b := &Balance{StorageSeizureMinValue: 1000}
	b.validateShops()
	if int(b.StorageSeizureMinValue) != 1000 {
		t.Errorf("StorageSeizureMinValue = %d, want 1000 (explicit value preserved)", int(b.StorageSeizureMinValue))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/configs/ -run TestValidateShops_StorageSeizure -v`
Expected: FAIL — `b.StorageSeizureMinValue` undefined (compile error).

- [ ] **Step 3: Add the struct field**

In `internal/configs/config.balance.go`, immediately after the `StorageFeePerItem` line (526):

```go
	StorageFeePerItem           ConfigInt   `yaml:"StorageFeePerItem"`                  // Gold charged per stored item per game month (default 1)
	StorageSeizureMinValue      ConfigInt   `yaml:"StorageSeizureMinValue"`             // Min aggregate stack value (spec.Value*Count) for a seized slot to be auctioned vs. disposed (default 250). Set very high to disable seizure-auction (dispose all).
```

- [ ] **Step 4: Add the validator default**

In `internal/configs/config.balance.shops.go`, in the `── STORAGE FEES ──` block right
after the `StorageFeePerItem` default:

```go
	// ── STORAGE FEES ─────────────────────────────────────────────────────────
	if b.StorageFeePerItem < 0 {
		b.StorageFeePerItem = 1
	}
	if b.StorageSeizureMinValue <= 0 {
		b.StorageSeizureMinValue = 250
	}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/configs/ -run TestValidateShops_StorageSeizure -v`
Expected: PASS (both tests).

- [ ] **Step 6: Commit**

```bash
git add internal/configs/config.balance.go internal/configs/config.balance.shops.go internal/configs/config.balance.shops_test.go
git commit -m "feat(configs): StorageSeizureMinValue knob for storage seizure auction (econ #4)"
```

---

## Task 2: `events.StorageItemSeized` event

**Files:**
- Modify: `internal/events/eventtypes.go` (add near `ItemOwnership`, ~line 238)
- Test: `internal/events/storage_seized_test.go` (create)

- [ ] **Step 1: Write the failing test**

Create `internal/events/storage_seized_test.go`:

```go
package events

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

func TestStorageItemSeized_Type(t *testing.T) {
	var e Event = StorageItemSeized{
		UserId: 7,
		Item:   items.Item{ItemId: 42},
		Count:  3,
		Owed:   1,
	}
	if e.Type() != "StorageItemSeized" {
		t.Errorf("Type() = %q, want StorageItemSeized", e.Type())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/events/ -run TestStorageItemSeized -v`
Expected: FAIL — `StorageItemSeized` undefined.

- [ ] **Step 3: Add the event type**

In `internal/events/eventtypes.go`, after the `ItemOwnership` type + its `Type()` method
(after line 239):

```go
// StorageItemSeized is emitted by the storage-fee hook when a stored slot is
// seized from a player who can't pay their bank-storage rent AND the slot's
// aggregate value (spec.Value * Count) clears the StorageSeizureMinValue floor.
// The auctions module listens and enqueues it onto the auction block. Sub-floor
// slots are disposed by the hook and never emit this event.
type StorageItemSeized struct {
	UserId int        // ex-owner; surplus after the lien returns here
	Item   items.Item // the seized item (a single representative of the stack)
	Count  int        // stack count seized (>=1); the winner receives all Count units
	Owed   int        // this lot's lien — unpaid rent to recoup from the sale before surplus
}

func (s StorageItemSeized) Type() string { return `StorageItemSeized` }
```

Verify `internal/events/eventtypes.go` already imports `github.com/GoMudEngine/GoMud/internal/items`
(it does — `ItemOwnership` uses `items.Item`). No import change needed.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/events/ -run TestStorageItemSeized -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/events/eventtypes.go internal/events/storage_seized_test.go
git commit -m "feat(events): StorageItemSeized event for storage seizure auction (econ #4)"
```

---

## Task 3: Hook — floor-based dispose vs. emit (replace deletion)

**Context:** The current forfeiture branch (`ChargeStorageFee`, `internal/hooks/StorageFee_MonthlyCharge.go:81-153`)
selects the cheapest slots and deletes them. We keep the identical *selection*, but each
selected slot is now either **disposed** (aggregate value < floor) or **emitted** as a
`StorageItemSeized` event (>= floor). The inbox notice distinguishes the two outcomes.

**Files:**
- Modify: `internal/hooks/StorageFee_MonthlyCharge.go:81-153`
- Test: `internal/hooks/StorageFee_seizure_test.go` (create)

- [ ] **Step 1: Write the failing test**

Create `internal/hooks/StorageFee_seizure_test.go`:

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureSeized registers a listener that records StorageItemSeized events, and
// returns the slice pointer plus a flush+unregister cleanup.
func captureSeized() (*[]events.StorageItemSeized, func()) {
	captured := []events.StorageItemSeized{}
	id := events.RegisterListener(events.StorageItemSeized{}, func(e events.Event) events.ListenerReturn {
		if evt, ok := e.(events.StorageItemSeized); ok {
			captured = append(captured, evt)
		}
		return events.Continue
	})
	return &captured, func() {
		events.ProcessEvents()
		events.UnregisterListener(events.StorageItemSeized{}, id)
	}
}

func TestChargeStorageFee_SeizesHighValue_DisposesLowValue(t *testing.T) {
	// Precondition: default config (fee 1g/slot, floor 250g).
	require.Equal(t, 1, int(configs.GetBalanceConfig().StorageFeePerItem))
	require.Equal(t, 250, int(configs.GetBalanceConfig().StorageSeizureMinValue))

	// High-value non-stackable (weapon) and low-value component.
	items.RegisterTestItemSpec(&items.ItemSpec{ItemId: 5001, Name: "gilded blade", Type: items.Weapon, Value: 1000})
	items.RegisterTestItemSpec(&items.ItemSpec{ItemId: 5002, Name: "twig", Type: items.Object, IsComponent: true, Value: 10})

	u := &users.UserRecord{UserId: 42, Character: &characters.Character{}}
	u.Character.Bank = 0
	u.Character.StorageFeeLastMonth = 0
	u.ItemStorage.AddItem(items.Item{ItemId: 5001}) // slot A: 1000 >= 250 -> seize
	u.ItemStorage.AddItem(items.Item{ItemId: 5002}) // slot B: 10   < 250 -> dispose
	require.Equal(t, 2, u.ItemStorage.SlotCount())

	seized, cleanup := captureSeized()

	// fee = 2 slots * 1g = 2; bank 0 -> shortfall 2 -> both slots selected.
	ChargeStorageFee(u, 999999)

	cleanup() // flush the event queue so the listener runs

	// Both slots left storage.
	assert.Equal(t, 0, u.ItemStorage.SlotCount(), "both selected slots removed from storage")
	// Exactly the high-value slot emitted a seizure event.
	require.Len(t, *seized, 1, "only the >=250 slot is auctioned")
	ev := (*seized)[0]
	assert.Equal(t, 42, ev.UserId)
	assert.Equal(t, 5001, ev.Item.ItemId)
	assert.Equal(t, 1, ev.Count)
	assert.Equal(t, 1, ev.Owed, "lien == feePerSlot")
}

func TestChargeStorageFee_SeizesAggregateComponentPile(t *testing.T) {
	require.Equal(t, 250, int(configs.GetBalanceConfig().StorageSeizureMinValue))

	// Each unit is 50g (below floor) but a stack of 6 aggregates to 300g (>= floor).
	items.RegisterTestItemSpec(&items.ItemSpec{ItemId: 5003, Name: "silk bolt", Type: items.Object, IsComponent: true, Value: 50})

	u := &users.UserRecord{UserId: 7, Character: &characters.Character{}}
	u.Character.Bank = 0
	for i := 0; i < 6; i++ {
		u.ItemStorage.AddItem(items.Item{ItemId: 5003})
	}
	require.Equal(t, 1, u.ItemStorage.SlotCount(), "identical components fold into one stacked slot")

	seized, cleanup := captureSeized()
	ChargeStorageFee(u, 999999) // fee = 1 slot * 1g = 1; bank 0 -> shortfall 1 -> select the slot
	cleanup()

	require.Len(t, *seized, 1, "the aggregate 300g pile is auctioned as one lot")
	assert.Equal(t, 5003, (*seized)[0].Item.ItemId)
	assert.Equal(t, 6, (*seized)[0].Count, "whole stack carried on the event")
	assert.Equal(t, 0, u.ItemStorage.SlotCount())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/hooks/ -run TestChargeStorageFee_Seizes -v`
Expected: FAIL — the current hook deletes both slots and emits nothing (`*seized` is empty).

- [ ] **Step 3: Rewrite the forfeiture branch**

In `internal/hooks/StorageFee_MonthlyCharge.go`, replace the forfeiture block (from
`// Cannot pay in full` down through the inbox notice, lines ~81-152) with the following.
Add `"github.com/GoMudEngine/GoMud/internal/events"` to the import block (the package
already imports `configs`, `users`, `fmt`, `sort`, `time`).

```go
	// Cannot pay in full — deduct what they have, then seize cheapest stacks to
	// cover the shortfall. High-value stacks are auctioned (emitted for the
	// auctions module); sub-floor stacks are disposed outright (as before).
	available := u.Character.Bank
	u.Character.Bank = 0
	shortfall := fee - available

	minValue := int(configs.GetBalanceConfig().StorageSeizureMinValue)

	type valuedSlot struct {
		idx        int
		stackValue int
		name       string
	}

	slots := u.ItemStorage.GetSlots()
	valued := make([]valuedSlot, len(slots))
	for i, slot := range slots {
		spec := slot.Item.GetSpec()
		stackValue := spec.Value * slot.Count
		var displayName string
		if slot.Count > 1 {
			displayName = fmt.Sprintf("a stack of %d %s", slot.Count, slot.Item.Name())
		} else {
			displayName = slot.Item.DisplayName()
		}
		valued[i] = valuedSlot{idx: i, stackValue: stackValue, name: displayName}
	}
	sort.Slice(valued, func(a, b int) bool {
		return valued[a].stackValue < valued[b].stackValue
	})

	// Select cheapest slots until the shortfall is covered (feePerSlot per slot).
	removeIdxs := map[int]bool{}
	goldCovered := 0
	for _, vs := range valued {
		if goldCovered >= shortfall {
			break
		}
		removeIdxs[vs.idx] = true
		goldCovered += feePerSlot
	}

	// Classify + act on each selected slot BEFORE removing (need the slot data).
	seizedNames := []string{}
	disposedNames := []string{}
	for _, vs := range valued {
		if !removeIdxs[vs.idx] {
			continue
		}
		slot := slots[vs.idx]
		if vs.stackValue < minValue {
			disposedNames = append(disposedNames, vs.name)
			continue
		}
		seizedNames = append(seizedNames, vs.name)
		events.AddToQueue(events.StorageItemSeized{
			UserId: u.UserId,
			Item:   slot.Item,
			Count:  slot.Count,
			Owed:   feePerSlot,
		})
	}

	// Remove selected slots in reverse index order to preserve indices.
	for i := len(slots) - 1; i >= 0; i-- {
		if removeIdxs[i] {
			u.ItemStorage.RemoveSlot(i)
		}
	}

	// Build the notice — show only the non-empty outcomes.
	notice := "Thornwall Bank Notice: Insufficient funds for storage fees."
	if len(seizedNames) > 0 {
		notice += " Seized and sent to auction to cover the debt (anything they fetch above what you owe returns to you): " + joinNames(seizedNames) + "."
	}
	if len(disposedNames) > 0 {
		notice += " Too little value to auction and disposed: " + joinNames(disposedNames) + "."
	}
	remaining := u.ItemStorage.SlotCount()
	notice += fmt.Sprintf(" Your remaining %d slot(s) are secure.", remaining)

	u.Inbox.Add(users.Message{
		FromName: "Thornwall Bank",
		Message:  notice,
		DateSent: time.Now(),
	})

	u.Character.StorageFeeLastMonth = currentMonth
}

// joinNames renders a comma-separated list of item names for a bank notice.
func joinNames(names []string) string {
	out := ""
	for i, n := range names {
		if i > 0 {
			out += ", "
		}
		out += n
	}
	return out
}
```

Delete the old `forfeited`/`removeIdxs`/`itemList` code and the old single inbox notice that
this block replaces (the entire span from `// Cannot pay in full` to the closing `}` of the
function). The pay-in-full branch (above the forfeiture block) is unchanged.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/hooks/ -run TestChargeStorageFee_Seizes -v`
Expected: PASS (both tests).

- [ ] **Step 5: Run the full hooks package to check nothing regressed**

Run: `go test ./internal/hooks/`
Expected: PASS (ok).

- [ ] **Step 6: Commit**

```bash
git add internal/hooks/StorageFee_MonthlyCharge.go internal/hooks/StorageFee_seizure_test.go
git commit -m "feat(hooks): seize high-value storage on unpaid rent, dispose junk (econ #4)"
```

---

## Task 4: `SeizedLot` type, `SeizedQueue` on manager, enqueue listener

**Context:** The auctions module registers listeners in `init()` (auctions.go:39-71) and
persists `AuctionManager` via `save()`/`load()` (auctions.go:127-190). We add the queue as a
persisted field and a listener that appends to it.

**Files:**
- Modify: `modules/auctions/auctions.go` (init listener; `AuctionManager` struct ~658;
  new `SeizedLot` type; new `storageSeizedHandler` method)
- Test: `modules/auctions/auctions_test.go`

- [ ] **Step 1: Write the failing test**

Add to `modules/auctions/auctions_test.go`:

```go
func TestStorageSeizedHandler_Enqueues(t *testing.T) {
	mod := &AuctionsModule{auctionMgr: AuctionManager{}}
	mod.storageSeizedHandler(events.StorageItemSeized{
		UserId: 11,
		Item:   items.Item{ItemId: 5001},
		Count:  3,
		Owed:   1,
	})
	if len(mod.auctionMgr.SeizedQueue) != 1 {
		t.Fatalf("SeizedQueue len = %d, want 1", len(mod.auctionMgr.SeizedQueue))
	}
	lot := mod.auctionMgr.SeizedQueue[0]
	if lot.ExOwnerUserId != 11 || lot.Item.ItemId != 5001 || lot.Count != 3 || lot.Owed != 1 {
		t.Errorf("enqueued lot = %+v, want {5001, count3, owner11, owed1}", lot)
	}
}

func TestSeizedQueue_YAMLRoundTrip(t *testing.T) {
	mgr := AuctionManager{SeizedQueue: []SeizedLot{
		{Item: items.Item{ItemId: 5001}, Count: 2, ExOwnerUserId: 9, Owed: 1},
	}}
	out, err := yaml.Marshal(mgr)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back AuctionManager
	if err := yaml.Unmarshal(out, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(back.SeizedQueue) != 1 || back.SeizedQueue[0].Item.ItemId != 5001 || back.SeizedQueue[0].Count != 2 {
		t.Errorf("round-trip lost SeizedQueue: %+v", back.SeizedQueue)
	}
}
```

If `auctions_test.go` does not already import `yaml`, add `"gopkg.in/yaml.v2"` to its imports.
Confirm `events` and `items` are imported there (they are used by existing tests).

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./modules/auctions/ -run 'TestStorageSeizedHandler_Enqueues|TestSeizedQueue_YAMLRoundTrip' -v`
Expected: FAIL — `SeizedLot` / `SeizedQueue` / `storageSeizedHandler` undefined.

- [ ] **Step 3: Add the type, field, and handler**

In `modules/auctions/auctions.go`, add the `SeizedLot` type (place it just above
`type AuctionManager struct`):

```go
// SeizedLot is a stored item seized from a player who couldn't pay bank-storage
// rent, waiting for a free auction block. It lists anonymously; sale proceeds
// settle a lien (Owed) with the surplus returning to the ex-owner.
type SeizedLot struct {
	Item          items.Item `yaml:"Item"`
	Count         int        `yaml:"Count"`         // >=1; winner receives all Count units
	ExOwnerUserId int        `yaml:"ExOwnerUserId"` // surplus after the lien returns here
	Owed          int        `yaml:"Owed"`          // the lien — unpaid rent to recoup from the sale
}
```

Add the persisted field to `AuctionManager`:

```go
type AuctionManager struct {
	ActiveAuction   *AuctionItem `yaml:"ActiveAuction,omitempty"`
	maxHistoryItems int
	PastAuctions    []PastAuctionItem `yaml:"PastAuctions,omitempty"`
	WalletBalances  map[string]int    `yaml:"WalletBalances,omitempty"` // NPC persona name -> wallet balance
	SeizedQueue     []SeizedLot       `yaml:"SeizedQueue,omitempty"`    // storage seizures awaiting a free block
}
```

Add the handler method (near `newRoundHandler`):

```go
// storageSeizedHandler enqueues a seized storage item for the auction block. It
// runs on the event queue (same goroutine as newRoundHandler), so no locking.
func (mod *AuctionsModule) storageSeizedHandler(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.StorageItemSeized)
	if !ok {
		return events.Continue
	}
	count := evt.Count
	if count < 1 {
		count = 1
	}
	mod.auctionMgr.SeizedQueue = append(mod.auctionMgr.SeizedQueue, SeizedLot{
		Item:          evt.Item,
		Count:         count,
		ExOwnerUserId: evt.UserId,
		Owed:          evt.Owed,
	})
	return events.Continue
}
```

Register the listener in `init()`, right after the existing `NewRound` registration
(auctions.go:70):

```go
	events.RegisterListener(events.NewRound{}, a.newRoundHandler)
	events.RegisterListener(events.StorageItemSeized{}, a.storageSeizedHandler)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./modules/auctions/ -run 'TestStorageSeizedHandler_Enqueues|TestSeizedQueue_YAMLRoundTrip' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add modules/auctions/auctions.go modules/auctions/auctions_test.go
git commit -m "feat(auctions): persisted seizure queue + enqueue listener (econ #4)"
```

---

## Task 5: Seized-lot listing fields + drain onto a free block

**Context:** The block holds one lot (`StartAuction` fails if busy). We add `Seized`,
`OwedLien`, `SeizedCount` to `AuctionItem` and a `drainSeizedQueue` that lists the front lot
directly (NOT via `StartAuction`, which requires an online seller — the ex-owner may be
offline by the time the block frees). The drain is called from `newRoundHandler` when the
block is free.

**Files:**
- Modify: `modules/auctions/auctions.go` (`AuctionItem` struct ~665; new `drainSeizedQueue`;
  `newRoundHandler` top ~410)
- Test: `modules/auctions/auctions_test.go`

- [ ] **Step 1: Write the failing test**

Add to `modules/auctions/auctions_test.go`:

```go
func TestDrainSeizedQueue_ListsFrontLot(t *testing.T) {
	cleanup := items.SeedItemsForTest(map[int]*items.ItemSpec{
		5001: {ItemId: 5001, Name: "gilded blade", Type: items.Weapon, Value: 1000},
	})
	defer cleanup()

	// Ex-owner offline: getUser returns nil, drain must still list.
	origGetUser := getUser
	getUser = func(id int) *users.UserRecord { return nil }
	defer func() { getUser = origGetUser }()

	mod := &AuctionsModule{
		// drainSeizedQueue takes the duration as a param and never touches plug,
		// so a nil plug is fine here.
		auctionMgr: AuctionManager{SeizedQueue: []SeizedLot{{Item: items.New(5001), Count: 1, ExOwnerUserId: 77, Owed: 1}}},
	}
	mod.drainSeizedQueue(120)

	a := mod.auctionMgr.GetCurrentAuction()
	if a == nil {
		t.Fatal("drain did not list a lot")
	}
	if !a.Seized || a.OwedLien != 1 || a.SeizedCount != 1 {
		t.Errorf("lot fields = seized:%v owed:%d count:%d, want true/1/1", a.Seized, a.OwedLien, a.SeizedCount)
	}
	if a.SellerUserId != 77 || !a.Anonymous {
		t.Errorf("seller=%d anon=%v, want 77/true", a.SellerUserId, a.Anonymous)
	}
	if a.BuyoutPrice != 1000 {
		t.Errorf("buyout=%d, want 1000 (spec.Value*Count)", a.BuyoutPrice)
	}
	if a.MinimumBid != reserveFrom(1000, auctionReservePct) {
		t.Errorf("reserve=%d, want %d", a.MinimumBid, reserveFrom(1000, auctionReservePct))
	}
	if len(mod.auctionMgr.SeizedQueue) != 0 {
		t.Errorf("queue not drained: %d left", len(mod.auctionMgr.SeizedQueue))
	}
}

func TestDrainSeizedQueue_NoopWhenBusy(t *testing.T) {
	mod := &AuctionsModule{auctionMgr: AuctionManager{
		ActiveAuction: &AuctionItem{ItemData: items.Item{ItemId: 1}},
		SeizedQueue:   []SeizedLot{{Item: items.Item{ItemId: 5001}, Count: 1, ExOwnerUserId: 1, Owed: 1}},
	}}
	mod.drainSeizedQueue(120)
	if len(mod.auctionMgr.SeizedQueue) != 1 {
		t.Errorf("busy block should not drain; queue=%d", len(mod.auctionMgr.SeizedQueue))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./modules/auctions/ -run TestDrainSeizedQueue -v`
Expected: FAIL — `AuctionItem.Seized` / `drainSeizedQueue` undefined.

- [ ] **Step 3: Add the fields and the drain**

In `modules/auctions/auctions.go`, add fields to `AuctionItem` (after `LastUpdate`):

```go
	LastUpdate     time.Time
	// Seized-lot fields (storage-seizure auction, econ #4). Seized lots list
	// anonymously; the surplus after OwedLien settles to SellerUserId (ex-owner),
	// and the winner receives SeizedCount units.
	Seized      bool `yaml:"Seized,omitempty"`
	OwedLien    int  `yaml:"OwedLien,omitempty"`
	SeizedCount int  `yaml:"SeizedCount,omitempty"`
```

Add the drain method:

```go
// drainSeizedQueue lists the front seized lot if the block is free. Lists
// directly (not via StartAuction) because the ex-owner may be offline by now.
func (mod *AuctionsModule) drainSeizedQueue(durationSeconds int) {
	if mod.auctionMgr.ActiveAuction != nil || len(mod.auctionMgr.SeizedQueue) == 0 {
		return
	}
	lot := mod.auctionMgr.SeizedQueue[0]
	mod.auctionMgr.SeizedQueue = mod.auctionMgr.SeizedQueue[1:]

	count := lot.Count
	if count < 1 {
		count = 1
	}
	buyout := lot.Item.GetSpec().Value * count
	if buyout < 1 {
		buyout = 1
	}

	// Ex-owner name only for records; Anonymous hides it on the block anyway.
	sellerName := ``
	if u := getUser(lot.ExOwnerUserId); u != nil {
		sellerName = u.Character.Name
	}

	mod.auctionMgr.ActiveAuction = &AuctionItem{
		ItemData:          lot.Item,
		SellerUserId:      lot.ExOwnerUserId,
		SellerName:        sellerName,
		Anonymous:         true,
		EndTime:           time.Now().Add(time.Second * time.Duration(durationSeconds)),
		BuyoutPrice:       buyout,
		MinimumBid:        reserveFrom(buyout, auctionReservePct),
		HighestBid:        0,
		HighestBidUserId:  0,
		HighestBidderName: ``,
		Seized:            true,
		OwedLien:          lot.Owed,
		SeizedCount:       count,
	}
}
```

Wire it into `newRoundHandler` — replace the early-return guard at the top
(auctions.go:410-413):

```go
	auctionNow := mod.auctionMgr.GetCurrentAuction()
	if auctionNow == nil {
		mod.drainSeizedQueue(mod.seizedLotDuration())
		auctionNow = mod.auctionMgr.GetCurrentAuction()
		if auctionNow == nil {
			return events.Continue
		}
		// A freshly-listed seized lot; players see it via the normal reminder
		// broadcast on subsequent rounds. Nothing else to do this tick.
		return events.Continue
	}
```

Add the duration helper (tolerates a nil plug for tests):

```go
// seizedLotDuration is the block time for a seized lot — the same DurationSeconds
// player lots use (default 60).
func (mod *AuctionsModule) seizedLotDuration() int {
	if mod.plug != nil {
		if dur, ok := mod.plug.Config.Get(`DurationSeconds`).(int); ok && dur > 0 {
			return dur
		}
	}
	return 60
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./modules/auctions/ -run TestDrainSeizedQueue -v`
Expected: PASS (both cases).

- [ ] **Step 5: Commit**

```bash
git add modules/auctions/auctions.go modules/auctions/auctions_test.go
git commit -m "feat(auctions): list seized lots onto a free block (anon, offline-safe) (econ #4)"
```

---

## Task 6: Seized-lot resolution — lien settlement, stack delivery, unsold dispose

**Context:** At resolution, `newRoundHandler` currently delivers the item to the winner and
settles gold to the seller (auctions.go:433-516) then runs the NPC-win sink (518+). Seized
lots differ: the surplus (after a lien) goes to the ex-owner, the winner gets `SeizedCount`
units, and a no-bid seized lot is **disposed** (not returned to storage). We branch the
resolution into a dedicated `resolveSeizedLot`.

**Files:**
- Modify: `modules/auctions/auctions.go` (`newRoundHandler` resolution block; new
  `resolveSeizedLot` + small helpers)
- Test: `modules/auctions/auctions_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `modules/auctions/auctions_test.go`:

```go
func TestResolveSeizedLot_PlayerWin_LienAndSurplus(t *testing.T) {
	cleanup := items.SeedItemsForTest(map[int]*items.ItemSpec{
		5001: {ItemId: 5001, Name: "gilded blade", Type: items.Weapon, Value: 1000},
	})
	defer cleanup()

	winner := &users.UserRecord{UserId: 88, Character: &characters.Character{}}
	winner.Character.Name = "Winner"
	exOwner := &users.UserRecord{UserId: 77, Character: &characters.Character{}}
	exOwner.Character.Name = "ExOwner"

	origGetUser := getUser
	getUser = func(id int) *users.UserRecord {
		switch id {
		case 88:
			return winner
		case 77:
			return exOwner
		}
		return nil
	}
	defer func() { getUser = origGetUser }()

	mod := &AuctionsModule{auctionMgr: AuctionManager{}}
	// Winner already escrowed 400 at bid time; commission 5% -> afterCommission 380.
	// Lien 1 -> surplus 379 to ex-owner. Winner gets 1 unit.
	mod.auctionMgr.ActiveAuction = &AuctionItem{
		ItemData:          items.New(5001),
		SellerUserId:      77,
		Anonymous:         true,
		HighestBid:        400,
		HighestBidUserId:  88,
		HighestBidderName: "Winner",
		Seized:            true,
		OwedLien:          1,
		SeizedCount:       1,
	}

	mod.resolveSeizedLot(mod.auctionMgr.ActiveAuction)

	wantSurplus := 400 - commissionFor(400, auctionCommissionPct) - 1
	if exOwner.Character.Bank != wantSurplus {
		t.Errorf("ex-owner bank=%d, want %d (surplus after commission+lien)", exOwner.Character.Bank, wantSurplus)
	}
	// StoreItem adds to the winner's BACKPACK (c.Items), not the bank (ItemStorage).
	// Weightless seeded items pass the zero-stat carry-capacity gate.
	if len(winner.Character.Items) != 1 {
		t.Errorf("winner backpack items=%d, want 1", len(winner.Character.Items))
	}
}

func TestResolveSeizedLot_StackDeliversAllUnits(t *testing.T) {
	cleanup := items.SeedItemsForTest(map[int]*items.ItemSpec{
		5003: {ItemId: 5003, Name: "silk bolt", Type: items.Object, Value: 50},
	})
	defer cleanup()

	winner := &users.UserRecord{UserId: 88, Character: &characters.Character{}}
	origGetUser := getUser
	getUser = func(id int) *users.UserRecord {
		if id == 88 {
			return winner
		}
		return nil // ex-owner offline
	}
	defer func() { getUser = origGetUser }()

	mod := &AuctionsModule{auctionMgr: AuctionManager{}}
	mod.auctionMgr.ActiveAuction = &AuctionItem{
		ItemData:         items.New(5003),
		SellerUserId:     77,
		HighestBid:       300,
		HighestBidUserId: 88,
		Seized:           true,
		OwedLien:         1,
		SeizedCount:      6,
	}
	mod.resolveSeizedLot(mod.auctionMgr.ActiveAuction)

	// All 6 units delivered to the backpack (c.Items). Plain Object (no IsComponent)
	// so nothing auto-routes to a component bag; 6 StoreItem calls -> 6 backpack entries.
	if len(winner.Character.Items) != 6 {
		t.Errorf("winner received %d units, want 6", len(winner.Character.Items))
	}
}

func TestResolveSeizedLot_Unsold_Disposed(t *testing.T) {
	cleanup := items.SeedItemsForTest(map[int]*items.ItemSpec{
		5001: {ItemId: 5001, Name: "gilded blade", Type: items.Weapon, Value: 1000},
	})
	defer cleanup()

	exOwner := &users.UserRecord{UserId: 77, Character: &characters.Character{}}
	origGetUser := getUser
	getUser = func(id int) *users.UserRecord {
		if id == 77 {
			return exOwner
		}
		return nil
	}
	defer func() { getUser = origGetUser }()

	mod := &AuctionsModule{auctionMgr: AuctionManager{}}
	mod.auctionMgr.ActiveAuction = &AuctionItem{
		ItemData:         items.New(5001),
		SellerUserId:     77,
		HighestBid:       0,
		HighestBidUserId: 0,
		HighestBidIsNPC:  false, // no bids at all
		Seized:           true,
		OwedLien:         1,
		SeizedCount:      1,
	}
	mod.resolveSeizedLot(mod.auctionMgr.ActiveAuction)

	// Disposed: NOT returned to ex-owner storage.
	if exOwner.ItemStorage.SlotCount() != 0 {
		t.Errorf("unsold seized lot must NOT return to storage; slots=%d", exOwner.ItemStorage.SlotCount())
	}
	// Ex-owner got a disposal notice.
	if len(exOwner.Inbox) == 0 {
		t.Error("expected a disposal inbox notice")
	}
}

func TestResolveSeizedLot_SaleBelowLien_NoNegativeCredit(t *testing.T) {
	cleanup := items.SeedItemsForTest(map[int]*items.ItemSpec{
		5001: {ItemId: 5001, Name: "gilded blade", Type: items.Weapon, Value: 1000},
	})
	defer cleanup()

	winner := &users.UserRecord{UserId: 88, Character: &characters.Character{}}
	exOwner := &users.UserRecord{UserId: 77, Character: &characters.Character{}}
	origGetUser := getUser
	getUser = func(id int) *users.UserRecord {
		switch id {
		case 88:
			return winner
		case 77:
			return exOwner
		}
		return nil
	}
	defer func() { getUser = origGetUser }()

	mod := &AuctionsModule{auctionMgr: AuctionManager{}}
	// Sale of 5 with a huge lien (10) -> afterCommission <= 10 -> lien capped, surplus 0.
	mod.auctionMgr.ActiveAuction = &AuctionItem{
		ItemData:         items.New(5001),
		SellerUserId:     77,
		HighestBid:       5,
		HighestBidUserId: 88,
		Seized:           true,
		OwedLien:         10,
		SeizedCount:      1,
	}
	mod.resolveSeizedLot(mod.auctionMgr.ActiveAuction)

	if exOwner.Character.Bank != 0 {
		t.Errorf("ex-owner bank=%d, want 0 (sale below lien -> no surplus, no negative credit)", exOwner.Character.Bank)
	}
}

func TestResolveSeizedLot_NpcWin_LienSettlesToExOwner(t *testing.T) {
	cleanup := items.SeedItemsForTest(map[int]*items.ItemSpec{
		5001: {ItemId: 5001, Name: "gilded blade", Type: items.Weapon, Value: 1000},
	})
	defer cleanup()

	exOwner := &users.UserRecord{UserId: 77, Character: &characters.Character{}}
	origGetUser := getUser
	getUser = func(id int) *users.UserRecord {
		if id == 77 {
			return exOwner
		}
		return nil
	}
	defer func() { getUser = origGetUser }()

	mod := &AuctionsModule{auctionMgr: AuctionManager{}}
	// NPC winner (name not registered -> buyerByName nil -> no sink, but lien settles).
	mod.auctionMgr.ActiveAuction = &AuctionItem{
		ItemData:          items.New(5001),
		SellerUserId:      77,
		HighestBid:        400,
		HighestBidUserId:  0,
		HighestBidIsNPC:   true,
		HighestBidderName: "Nonexistent Collector",
		Seized:            true,
		OwedLien:          1,
		SeizedCount:       1,
	}
	mod.resolveSeizedLot(mod.auctionMgr.ActiveAuction)

	wantSurplus := 400 - commissionFor(400, auctionCommissionPct) - 1
	if exOwner.Character.Bank != wantSurplus {
		t.Errorf("ex-owner bank=%d, want %d (surplus settles even on an NPC win)", exOwner.Character.Bank, wantSurplus)
	}
}
```

Confirm `characters` is imported in `auctions_test.go` (add
`"github.com/GoMudEngine/GoMud/internal/characters"` if not).

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./modules/auctions/ -run TestResolveSeizedLot -v`
Expected: FAIL — `resolveSeizedLot` undefined.

- [ ] **Step 3: Implement `resolveSeizedLot` and branch it in**

In `modules/auctions/auctions.go`, add the helper:

```go
// resolveSeizedLot settles a seized lot at end-of-auction: deliver the stack to
// the winner (player or NPC), recoup the lien and return the surplus to the
// ex-owner, or dispose the item if it drew no bids. Mirrors the normal
// resolution but with lien math, Count-aware delivery, and dispose-on-unsold.
func (mod *AuctionsModule) resolveSeizedLot(a *AuctionItem) {
	sold := a.HighestBidUserId > 0 || a.HighestBidIsNPC

	if !sold {
		// No bids — dispose (do NOT return to storage; that would re-trigger the debt).
		if u := getUser(a.SellerUserId); u != nil {
			u.Inbox.Add(users.Message{
				FromName: `Auction System`,
				Message:  fmt.Sprintf(`Your seized <ansi fg="item">%s</ansi> found no buyer at auction and was disposed.`, a.ItemData.DisplayName()),
			})
			u.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="yellow">Your seized <ansi fg="item">%s</ansi> found no buyer and was disposed.</ansi>`, a.ItemData.DisplayName()))
		} else {
			users.SearchOfflineUsers(func(u *users.UserRecord) bool {
				if u.UserId == a.SellerUserId {
					u.Inbox.Add(users.Message{
						FromName: `Auction System`,
						Message:  fmt.Sprintf(`Your seized %s found no buyer at auction and was disposed.`, a.ItemData.DisplayName()),
					})
					users.SaveUser(*u)
					return false
				}
				return true
			})
		}
		return
	}

	count := a.SeizedCount
	if count < 1 {
		count = 1
	}

	// Deliver Count units to a player winner (NPC winner: item sinks/relists below).
	if a.HighestBidUserId > 0 {
		if winner := getUser(a.HighestBidUserId); winner != nil {
			for i := 0; i < count; i++ {
				winner.Character.StoreItem(a.ItemData)
				events.AddToQueue(events.ItemOwnership{UserId: winner.UserId, Item: a.ItemData, Gained: true})
			}
			winner.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="yellow">You have won the auction for <ansi fg="item">%s</ansi>! It has been added to your backpack.</ansi>`, a.ItemData.DisplayName()))
		} else {
			users.SearchOfflineUsers(func(u *users.UserRecord) bool {
				if u.UserId == a.HighestBidUserId {
					for i := 0; i < count; i++ {
						itemCopy := a.ItemData
						u.Inbox.Add(users.Message{
							FromName: `Auction System`,
							Message:  fmt.Sprintf(`You won the auction for <ansi fg="item">%s</ansi> while you were offline.`, a.ItemData.DisplayName()),
							Item:     &itemCopy,
						})
					}
					users.SaveUser(*u)
					return false
				}
				return true
			})
		}
	}

	// Lien settlement: surplus after commission + lien returns to the ex-owner.
	afterCommission := a.HighestBid - commissionFor(a.HighestBid, auctionCommissionPct)
	lien := a.OwedLien
	if lien > afterCommission {
		lien = afterCommission
	}
	surplus := afterCommission - lien
	if surplus > 0 {
		if seller := getUser(a.SellerUserId); seller != nil {
			seller.Character.Bank += surplus
			events.AddToQueue(events.EquipmentChange{UserId: seller.UserId, BankChange: surplus})
			seller.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="yellow">Your seized <ansi fg="item">%s</ansi> sold at auction. After the debt and the house's cut, <ansi fg="gold">%d gold</ansi> was returned to your account.</ansi>`, a.ItemData.DisplayName(), surplus))
		} else {
			users.SearchOfflineUsers(func(u *users.UserRecord) bool {
				if u.UserId == a.SellerUserId {
					u.Inbox.Add(users.Message{
						FromName: `Auction System`,
						Message:  fmt.Sprintf(`Your seized %s sold at auction. After the debt and the house's cut, %d gold was returned to your account.`, a.ItemData.DisplayName(), surplus),
						Gold:     surplus,
					})
					users.SaveUser(*u)
					return false
				}
				return true
			})
		}
	}

	// NPC winner takes the item out of circulation (or a shopkeeper relists it).
	if a.HighestBidIsNPC {
		if b := buyerByName(a.HighestBidderName); b != nil {
			if r, ok := b.(auctionWinReceiver); ok {
				r.Receive(a.ItemData)
			}
		}
	}
}
```

Now branch it into `newRoundHandler`. In the `IsEnded()` block, after
`mod.auctionMgr.EndAuction()` and the auction-end broadcast `for` loop, the current code is:

```go
		// Give the item to the winner and let them know. An NPC win counts as
		// sold too (HighestBidUserId is 0 for an NPC — the item sinks below).
		if auctionNow.HighestBidUserId > 0 || auctionNow.HighestBidIsNPC {
			// ...existing winner-delivery + seller-settlement + NPC-sink body (auctions.go:435-527)...
		}
```

Change ONLY the guard line: wrap that entire `if auctionNow.HighestBidUserId > 0 ||
auctionNow.HighestBidIsNPC { ... }` statement so it becomes the `else` arm of a seized check.
Concretely, insert `if auctionNow.Seized { mod.resolveSeizedLot(auctionNow) } else {` before
that `if`, and a closing `}` after its closing brace:

```go
		if auctionNow.Seized {
			mod.resolveSeizedLot(auctionNow)
		} else if auctionNow.HighestBidUserId > 0 || auctionNow.HighestBidIsNPC {
			// ...existing winner-delivery + seller-settlement + NPC-sink body, UNCHANGED...
		}
```

Do not edit anything inside that body — only the guard changes (the existing
`if HighestBidUserId>0 || HighestBidIsNPC` becomes `else if ...`). All seized-lot behavior
lives in `resolveSeizedLot`; nothing is duplicated.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./modules/auctions/ -run TestResolveSeizedLot -v`
Expected: PASS (all three).

- [ ] **Step 5: Run the whole auctions package**

Run: `go test ./modules/auctions/`
Expected: PASS (ok) — existing player/NPC auction tests still green.

- [ ] **Step 6: Commit**

```bash
git add modules/auctions/auctions.go modules/auctions/auctions_test.go
git commit -m "feat(auctions): seized-lot resolution — lien, stack delivery, dispose-if-unsold (econ #4)"
```

---

## Task 7: Full build, boot smoke test, docs

**Files:**
- Modify: `PATCH_NOTES.md` (new dated entry at top)
- Modify: `docs/PATH_TO_1.0.md` (mark #4 done)

- [ ] **Step 1: Build the whole project**

Run: `go build ./...`
Expected: no output (success).

- [ ] **Step 2: Run the full test suite for touched packages**

Run: `go test ./internal/configs/ ./internal/events/ ./internal/hooks/ ./modules/auctions/`
Expected: all `ok`.

- [ ] **Step 3: Boot smoke test (per pre-push SOP)**

Nuke instance saves (do NOT touch `shops/`), then boot and watch for a clean load past
data-file loading with no panic:

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
```

Run the server (`go run .`), confirm it reaches the ready state with
`mobs.LoadDataFiles() loadedCount=...` / `quests.LoadDataFiles() loadedCount=...` and no
panic, then stop it. (Config change is additive with a validator default, so no config-file
edit is required for boot.)

- [ ] **Step 4: Add the patch note**

Prepend to `PATCH_NOTES.md` (under `# DOGMud Patch Notes`):

```markdown
## 2026-07-15 — Auction house: the bank sells what you can't keep

If your bank-storage rent goes unpaid and you don't have the coin to cover it, the bank no
longer simply throws out your goods. Anything genuinely valuable is seized and put up on the
auction block instead — anonymously — where other players and the town's NPC buyers can bid
on it. Whatever it fetches goes to settle what you owed, and any surplus is returned to your
account. Only low-value oddments are disposed of outright, and an item nobody bids on is let
go after its turn on the block. A seized pile of crafting materials worth enough all together
will go up as a single lot.
```

- [ ] **Step 5: Mark #4 done in the roadmap**

In `docs/PATH_TO_1.0.md`, change the econ arc **#4** line from `⬜` to `✅` with a one-line
completion note dated 2026-07-15 referencing this plan and spec, in the same style as the
#2.x entries above it.

- [ ] **Step 6: Commit**

```bash
git add PATCH_NOTES.md docs/PATH_TO_1.0.md
git commit -m "docs(auctions): patch notes + roadmap for storage seizure auction (econ #4)"
```

---

## Notes for the implementer

- **Event-queue flush in tests:** `events.AddToQueue` is asynchronous; a listener only sees
  the event after `events.ProcessEvents()`. The `captureSeized` helper in Task 3 flushes in
  its cleanup — call cleanup before asserting.
- **`getUser` override:** the auctions package resolves users through the package var
  `getUser` (defaults to `users.GetByUserId`). Tests override it; always restore it with a
  deferred closure.
- **Offline paths reuse `users.SearchOfflineUsers` + `users.SaveUser`** — mirror the existing
  patterns already in `newRoundHandler` / `returnUnsoldItem`; don't invent a new offline path.
- **No hard numbers to players, except gold:** bank/auction notices already state gold
  amounts (matching the existing storage-fee notice); that's the sanctioned exception. Don't
  surface bid counts, round counts, or internal values.
- **Concurrency:** `storageSeizedHandler`, `drainSeizedQueue`, and `newRoundHandler` all run
  on the event-processing goroutine, so `auctionMgr` needs no locking (consistent with the
  existing lock-free module).
```
