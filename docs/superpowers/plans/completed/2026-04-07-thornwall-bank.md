# Thornwall Bank Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create a bank building in Thornwall City with unlimited item storage and monthly storage fees that auto-deduct from the player's bank balance.

**Architecture:** New room YAML + mob YAML for the bank, add `StorageCapacity` field to rooms, replace hardcoded 20-item cap in storage command, add a monthly fee hook that charges all players (online and offline) and forfeits cheapest items on non-payment.

**Tech Stack:** Go, YAML data files, existing storage/inbox/gametime infrastructure.

---

### Task 1: Add StorageCapacity Field to Rooms

**Files:**
- Modify: `internal/rooms/rooms.go:78` (add field after IsStorage)

- [ ] **Step 1: Add StorageCapacity field to Room struct**

In `internal/rooms/rooms.go`, add after the `IsStorage` line (line 78):

```go
StorageCapacity   int                               `yaml:"storagecapacity,omitempty" instance:"skip"` // Max items in storage (0 = default 20)
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: Clean build.

- [ ] **Step 3: Commit**

```bash
git add internal/rooms/rooms.go
git commit -m "feat: add StorageCapacity field to Room struct"
```

---

### Task 2: Use Room Capacity in Storage Command

**Files:**
- Modify: `internal/usercommands/storage.go:67-69`

- [ ] **Step 1: Replace hardcoded 20-item cap**

In `internal/usercommands/storage.go`, replace the capacity check (lines 67-71):

```go
		spaceLeft := 20 - len(itemsInStorage)
		if spaceLeft < 1 {
			user.SendText(`You can have 20 objects in storage`)
			return true, nil
		}
```

With:

```go
		storageCap := room.StorageCapacity
		if storageCap <= 0 {
			storageCap = 20
		}
		spaceLeft := storageCap - len(itemsInStorage)
		if spaceLeft < 1 {
			user.SendText(`Your storage is full.`)
			return true, nil
		}
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: Clean build.

- [ ] **Step 3: Commit**

```bash
git add internal/usercommands/storage.go
git commit -m "feat: use room StorageCapacity instead of hardcoded 20-item limit"
```

---

### Task 3: Add StorageFeePerItem Config Knob

**Files:**
- Modify: `internal/configs/config.balance.go` (add field + validation)
- Modify: `_datafiles/config.yaml` (add entry)

- [ ] **Step 1: Add field to BalanceConfig struct**

In `internal/configs/config.balance.go`, add near the other economy/shop config fields:

```go
StorageFeePerItem              ConfigInt   `yaml:"StorageFeePerItem"`              // Gold charged per stored item per game month (default 1)
```

- [ ] **Step 2: Add validation**

In the `Validate()` method:

```go
if b.StorageFeePerItem < 0 {
	b.StorageFeePerItem = 1
}
```

- [ ] **Step 3: Add config entry**

In `_datafiles/config.yaml`, add under the Balance section:

```yaml
  StorageFeePerItem: 1               # Gold charged per stored item per game month
```

- [ ] **Step 4: Verify build**

Run: `go build ./...`
Expected: Clean build.

- [ ] **Step 5: Commit**

```bash
git add internal/configs/config.balance.go _datafiles/config.yaml
git commit -m "feat: add StorageFeePerItem config knob (default 1g)"
```

---

### Task 4: Add StorageFeeLastMonth to Character

**Files:**
- Modify: `internal/characters/character.go` (add field)

- [ ] **Step 1: Add field to Character struct**

In `internal/characters/character.go`, add near the other economy-related fields (near `Bank`):

```go
StorageFeeLastMonth int `yaml:"storagefee_lastmonth,omitempty"` // Game month when storage fees were last charged
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: Clean build.

- [ ] **Step 3: Commit**

```bash
git add internal/characters/character.go
git commit -m "feat: add StorageFeeLastMonth field to Character"
```

---

### Task 5: Implement Monthly Storage Fee Hook

**Files:**
- Create: `internal/hooks/StorageFee_MonthlyCharge.go`

- [ ] **Step 1: Create the fee processing hook**

Create `internal/hooks/StorageFee_MonthlyCharge.go`:

```go
package hooks

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// CheckStorageFees runs on every NewRound and checks whether the game
// month has changed. When it has, it charges all online users their
// storage fee. Offline users are charged on next login via
// ChargeStorageFee (called from user validation or login path).
func CheckStorageFees(e events.Event) events.ListenerReturn {

	gdNow := gametime.GetDate()
	currentMonth := gdNow.Year*12 + gdNow.Month

	for _, u := range users.GetAllActiveUsers() {
		ChargeStorageFee(u, currentMonth)
	}

	return events.Continue
}

// ChargeStorageFee processes the monthly storage fee for a single user.
// Safe to call multiple times — uses StorageFeeLastMonth to prevent
// double-charging. Called from the round tick for online users and
// from the login path for returning offline users.
func ChargeStorageFee(u *users.UserRecord, currentMonth int) {
	if u.ItemStorage.Items == nil || len(u.ItemStorage.Items) == 0 {
		u.Character.StorageFeeLastMonth = currentMonth
		return
	}

	if u.Character.StorageFeeLastMonth >= currentMonth {
		return // Already charged this month
	}

	feePerItem := int(configs.GetBalanceConfig().StorageFeePerItem)
	if feePerItem <= 0 {
		u.Character.StorageFeeLastMonth = currentMonth
		return
	}

	itemCount := len(u.ItemStorage.Items)
	fee := itemCount * feePerItem

	if u.Character.Bank >= fee {
		// Can pay in full
		u.Character.Bank -= fee
		u.Character.StorageFeeLastMonth = currentMonth

		// Warn if they won't be able to pay next month
		if u.Character.Bank < fee {
			msg := fmt.Sprintf(
				"Thornwall Bank Notice: Your monthly storage fee of "+
					"%dg has been collected. You have %dg remaining "+
					"in your account. Next month's fee will be %dg "+
					"-- please deposit additional gold or retrieve "+
					"items to avoid forfeiture.",
				fee, u.Character.Bank, fee)
			u.Inbox.Add(users.Message{
				FromName: "Thornwall Bank",
				Message:  msg,
				DateSent: time.Now(),
			})
		}
		return
	}

	// Cannot pay in full — deduct what they have, forfeit cheapest items
	available := u.Character.Bank
	u.Character.Bank = 0
	shortfall := fee - available

	// How many items must be forfeited to cover the shortfall
	itemsToRemove := int(math.Ceil(float64(shortfall) / float64(feePerItem)))
	if itemsToRemove > len(u.ItemStorage.Items) {
		itemsToRemove = len(u.ItemStorage.Items)
	}

	// Sort by gold value ascending (cheapest first)
	type valuedItem struct {
		idx   int
		value int
		item  items.Item
	}
	valued := make([]valuedItem, len(u.ItemStorage.Items))
	for i, itm := range u.ItemStorage.Items {
		spec := itm.GetSpec()
		valued[i] = valuedItem{idx: i, value: spec.Value, item: itm}
	}
	sort.Slice(valued, func(a, b int) bool {
		return valued[a].value < valued[b].value
	})

	// Forfeit the cheapest items
	forfeited := make([]string, 0, itemsToRemove)
	removeSet := make(map[int]bool, itemsToRemove)
	for i := 0; i < itemsToRemove && i < len(valued); i++ {
		forfeited = append(forfeited, valued[i].item.DisplayName())
		removeSet[valued[i].idx] = true
	}

	// Rebuild storage without forfeited items
	kept := make([]items.Item, 0, len(u.ItemStorage.Items)-len(removeSet))
	for i, itm := range u.ItemStorage.Items {
		if !removeSet[i] {
			kept = append(kept, itm)
		}
	}
	u.ItemStorage.Items = kept

	// Send forfeiture notice
	itemList := ""
	for i, name := range forfeited {
		if i > 0 {
			itemList += ", "
		}
		itemList += name
	}
	remaining := len(u.ItemStorage.Items)
	msg := fmt.Sprintf(
		"Thornwall Bank Notice: Insufficient funds for storage "+
			"fees. The following items were forfeited: %s. Your "+
			"remaining %d items are secure.",
		itemList, remaining)
	u.Inbox.Add(users.Message{
		FromName: "Thornwall Bank",
		Message:  msg,
		DateSent: time.Now(),
	})

	u.Character.StorageFeeLastMonth = currentMonth
}
```

- [ ] **Step 2: Register the hook**

In `internal/hooks/hooks.go`, add with the other NewRound listeners:

```go
events.RegisterListener(events.NewRound{}, CheckStorageFees)
```

- [ ] **Step 3: Add offline user fee charging on login**

Find where users are loaded/validated on login. In the user's `Validate()` or the login handler, add:

```go
// Charge any missed storage fees on login
currentMonth := gametime.GetDate().Year*12 + gametime.GetDate().Month
hooks.ChargeStorageFee(u, currentMonth)
```

Note: This may need to be called from the user loading path rather than directly from hooks to avoid import cycles. If so, expose the fee logic as a function in a shared package, or call it from the login handler in `internal/usercommands/` or the user validation path. Read the login flow to find the right insertion point.

- [ ] **Step 4: Verify build**

Run: `go build ./...`
Expected: Clean build.

- [ ] **Step 5: Commit**

```bash
git add internal/hooks/StorageFee_MonthlyCharge.go internal/hooks/hooks.go
git commit -m "feat: implement monthly storage fee with forfeiture and inbox warnings"
```

---

### Task 6: Create Bank Room and NPC

**Files:**
- Create: `_datafiles/world/dogmud/rooms/thornwall_city/510.yaml`
- Create: `_datafiles/world/dogmud/mobs/thornwall_city/120-bank_clerk.yaml`
- Modify: `_datafiles/world/dogmud/rooms/thornwall_city/465.yaml` (add south exit)

- [ ] **Step 1: Create bank room YAML**

Create `_datafiles/world/dogmud/rooms/thornwall_city/510.yaml`:

```yaml
roomid: 510
zone: Thornwall City
title: Thornwall Bank
description: A solid stone building with iron-banded doors and narrow
  windows. Inside, a long counter of dark wood separates the public
  lobby from the vault area behind. Heavy ledgers line the shelves,
  and the faint clink of counted coins drifts from a back room. The
  air smells of ink, old paper, and the metallic tang of gold. A
  large iron strongbox sits against the far wall, its lock the size
  of a fist.
biome: city
isbank: true
isstorage: true
storagecapacity: 1000
coord:
  x: 5
  y: -1
  z: 0
exits:
  north:
    roomid: 465
nouns:
  counter: A long counter of dark polished wood, worn smooth by years
    of transactions. Behind it, cubbyholes stuffed with rolled
    parchment and wax-sealed documents line the wall.
  strongbox: A massive iron strongbox bolted to the stone floor. The
    lock mechanism is dwarven-made -- intricate and unforgiving.
  ledgers: Rows of leather-bound ledgers, their spines cracked with
    use. The most recent one lies open, its pages dense with neat
    columns of figures.
spawninfo:
- mobid: 120
  message: A bank clerk looks up from behind the counter.
  respawnrate: 2 real minutes
idlemessages:
- The clink of coins being counted drifts from the back room.
- A clerk runs a finger down a column of figures, lips moving.
- Someone in the vault area shifts a heavy strongbox with a grunt.
```

- [ ] **Step 2: Create bank clerk mob YAML**

Create `_datafiles/world/dogmud/mobs/thornwall_city/120-bank_clerk.yaml`:

```yaml
mobid: 120
zone: Thornwall City
statpool: 80
itemdropchance: 0
hostile: false
charm_immune: true
non_combatant: true
maxwander: 0
groups:
  - humanoid
idlecommands:
  - 'emote adjusts a stack of coins on the counter'
  - 'emote makes a notation in a heavy ledger'
  - ''
  - 'emote examines a wax seal with a practiced eye'
  - ''
activitylevel: 15
character:
  name: bank clerk
  description: |
    A precise, bespectacled figure in a dark coat with ink-stained
    cuffs. The bank clerk moves with the careful efficiency of
    someone who handles other people's gold for a living. Every
    gesture is measured, every word considered. A heavy ring of
    keys hangs from their belt.
  speciesid: 1
  level: 1
```

- [ ] **Step 3: Add south exit to Market Square Center (room 465)**

In `_datafiles/world/dogmud/rooms/thornwall_city/465.yaml`, add to the exits section:

```yaml
  south:
    roomid: 510
```

- [ ] **Step 4: Verify build**

Run: `go build ./...`
Expected: Clean build.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/rooms/thornwall_city/510.yaml \
       _datafiles/world/dogmud/mobs/thornwall_city/120-bank_clerk.yaml \
       _datafiles/world/dogmud/rooms/thornwall_city/465.yaml
git commit -m "feat: create Thornwall Bank room and bank clerk NPC"
```

---

### Task 7: Update Help Files

**Files:**
- Modify: `_datafiles/world/dogmud/templates/help/bank.template`
- Modify: `_datafiles/world/dogmud/templates/help/storage.template`

- [ ] **Step 1: Update bank help file**

Replace the contents of `_datafiles/world/dogmud/templates/help/bank.template`:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">bank</ansi>

The <ansi fg="command">bank</ansi> command lets you deposit and
withdraw gold at a bank. The Thornwall Bank is located south of
Market Square.

<ansi fg="yellow">Gold Commands: </ansi>

  <ansi fg="command">bank</ansi> - See your gold on hand and
  in the bank

  <ansi fg="command">bank deposit [amount/all]</ansi> - Deposit
  gold into the bank

  <ansi fg="command">bank withdraw [amount/all]</ansi> - Withdraw
  gold from the bank

<ansi fg="yellow">Item Storage: </ansi>

The bank also offers item storage. Use the
<ansi fg="command">storage</ansi> command while at the bank to
store and retrieve items. Unlike other storage locations,
the bank has no item limit.

<ansi fg="yellow">Storage Fees: </ansi>

The bank charges a monthly fee of 1 gold per stored item,
deducted automatically from your bank balance. If you
cannot cover the fee, the bank will forfeit your least
valuable items first and notify you via
<ansi fg="command">inbox</ansi>. Keep gold deposited to
cover your storage costs.

```

- [ ] **Step 2: Update storage help file**

Replace the contents of `_datafiles/world/dogmud/templates/help/storage.template`:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">storage</ansi>

The <ansi fg="command">storage</ansi> command lets you store and
retrieve items at storage locations around the world.

<ansi fg="yellow">Usage: </ansi>

  <ansi fg="command">storage</ansi> - See what you have in
  storage

  <ansi fg="command">storage add [item/all]</ansi> - Add an item
  to storage

  <ansi fg="command">storage remove [item/all]</ansi> - Remove an
  item from storage

<ansi fg="yellow">Locations: </ansi>

Most storage locations hold up to 20 items. The Thornwall
Bank (south of Market Square) offers unlimited storage
but charges a monthly fee of 1 gold per item, deducted
from your bank balance. Type <ansi fg="command">help
bank</ansi> for details.

```

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/templates/help/bank.template \
       _datafiles/world/dogmud/templates/help/storage.template
git commit -m "docs: update bank and storage help files with fees and location"
```

---

### Task 8: Final Integration Test

- [ ] **Step 1: Run full test suite**

Run: `go test ./... -count=1 2>&1 | tail -30`
Expected: All tests pass.

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: Clean build.

- [ ] **Step 3: Commit any cleanup**

If needed: `chore: final cleanup for Thornwall Bank`
