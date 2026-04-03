# Command Unification — Substage 2: Economy Commands + Go

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract shared core logic for drop, remove, equip, get, give, and go commands using atomic transfer primitives that prevent item duplication and loss.

**Architecture:** Atomic `TransferItem` / `TransferGold` functions in `internal/actions/transfer.go` become the single chokepoint for all item/gold movement. Every transfer test asserts conservation invariants. Per-command shared actions use these primitives. User/mob wrappers handle actor-specific concerns.

**Tech Stack:** Go, testify/assert, existing event system

---

## File Structure

| File | Responsibility |
|------|---------------|
| `internal/actions/transfer.go` | Atomic TransferItem, TransferGold, FloorPickupGold, FloorDropGold |
| `internal/actions/transfer_test.go` | Conservation invariant tests for all transfer primitives |
| `internal/actions/drop.go` | Shared DropItem, DropGold actions |
| `internal/actions/remove_equip.go` | Shared RemoveEquipment, EquipItem actions |
| `internal/actions/get.go` | Shared GetItemFromFloor, GetGoldFromFloor actions |
| `internal/actions/give.go` | Shared GiveItem, GiveGold actions |
| `internal/actions/go.go` | Shared FindExit, MoveActor actions |
| `internal/actions/economy_test.go` | Parity + conservation tests for all economy commands |
| `internal/usercommands/drop.go` | Thin wrapper (modify existing) |
| `internal/usercommands/remove.go` | Thin wrapper (modify existing) |
| `internal/usercommands/equip.go` | Thin wrapper (modify existing) |
| `internal/usercommands/get.go` | Thin wrapper (modify existing) |
| `internal/usercommands/give.go` | Thin wrapper (modify existing) |
| `internal/usercommands/go.go` | Thin wrapper (modify existing) |
| `internal/mobcommands/drop.go` | Thin wrapper (modify existing) |
| `internal/mobcommands/remove.go` | Thin wrapper (modify existing) |
| `internal/mobcommands/equip.go` | Thin wrapper (modify existing) |
| `internal/mobcommands/get.go` | Thin wrapper (modify existing) |
| `internal/mobcommands/give.go` | Thin wrapper (modify existing) |
| `internal/mobcommands/go.go` | Thin wrapper (modify existing) |

---

### Task 1: Atomic Transfer Primitives

**Files:**
- Create: `internal/actions/transfer.go`

- [ ] **Step 1: Create transfer.go with all primitives**

```go
package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// TransferItemToBackpack atomically moves an item from a source
// (room floor, another character, equipment) to a character's
// backpack. Removes from source first; if StoreItem fails
// (capacity), rolls back by calling rollback.
//
// removeFrom: removes the item from its current location.
//   Returns true if found and removed.
// rollback: puts the item back if StoreItem fails.
//   Only called on add failure.
//
// Fires ItemOwnership gained event on success.
func TransferItemToBackpack(
	item items.Item,
	toChar *characters.Character,
	toActorUserId int,
	toActorMobInstanceId int,
	removeFrom func(items.Item),
	rollback func(items.Item),
) error {
	// Remove from source
	removeFrom(item)

	// Add to destination backpack
	if !toChar.StoreItem(item) {
		// Capacity exceeded — roll back
		rollback(item)
		return fmt.Errorf("cannot carry any more")
	}

	// Fire ownership event
	events.AddToQueue(events.ItemOwnership{
		UserId:        toActorUserId,
		MobInstanceId: toActorMobInstanceId,
		Item:          item,
		Gained:        true,
	})

	return nil
}

// TransferItemToFloor atomically moves an item from a character's
// inventory to the room floor.
//
// Fires ItemOwnership lost event on success.
func TransferItemToFloor(
	item items.Item,
	fromChar *characters.Character,
	fromActorUserId int,
	fromActorMobInstanceId int,
	room *rooms.Room,
) error {
	// Remove from character
	if !fromChar.RemoveItem(item) {
		return fmt.Errorf("item not found in inventory")
	}

	// Add to room floor (always succeeds — no capacity on rooms)
	room.AddItem(item, false)

	// Fire ownership event
	events.AddToQueue(events.ItemOwnership{
		UserId:        fromActorUserId,
		MobInstanceId: fromActorMobInstanceId,
		Item:          item,
		Gained:        false,
	})

	return nil
}

// TransferItemBetweenChars atomically moves an item from one
// character to another. Removes from source first; if destination
// StoreItem fails, rolls item back to source.
//
// Fires ItemOwnership events for both sides on success.
func TransferItemBetweenChars(
	item items.Item,
	fromChar *characters.Character,
	fromUserId int,
	fromMobInstanceId int,
	toChar *characters.Character,
	toUserId int,
	toMobInstanceId int,
) error {
	// Remove from source
	if !fromChar.RemoveItem(item) {
		return fmt.Errorf("item not found in inventory")
	}

	// Add to destination
	if !toChar.StoreItem(item) {
		// Roll back — return item to source
		fromChar.StoreItem(item)
		return fmt.Errorf("recipient cannot carry any more")
	}

	// Fire ownership events
	events.AddToQueue(events.ItemOwnership{
		UserId:        fromUserId,
		MobInstanceId: fromMobInstanceId,
		Item:          item,
		Gained:        false,
	})
	events.AddToQueue(events.ItemOwnership{
		UserId:        toUserId,
		MobInstanceId: toMobInstanceId,
		Item:          item,
		Gained:        true,
	})

	return nil
}

// TransferGold moves gold between two characters.
// Validates source has enough before transferring.
func TransferGold(amount int, from *characters.Character, to *characters.Character) error {
	if amount <= 0 {
		return fmt.Errorf("invalid amount")
	}
	if from.Gold < amount {
		return fmt.Errorf("not enough gold")
	}
	from.Gold -= amount
	to.Gold += amount
	return nil
}

// FloorPickupGold moves gold from room floor to character.
func FloorPickupGold(amount int, char *characters.Character, room *rooms.Room) error {
	if amount <= 0 {
		return fmt.Errorf("invalid amount")
	}
	if room.Gold < amount {
		return fmt.Errorf("not enough gold on floor")
	}
	room.Gold -= amount
	char.Gold += amount
	return nil
}

// FloorDropGold moves gold from character to room floor.
func FloorDropGold(amount int, char *characters.Character, room *rooms.Room) error {
	if amount <= 0 {
		return fmt.Errorf("invalid amount")
	}
	if char.Gold < amount {
		return fmt.Errorf("not enough gold")
	}
	char.Gold -= amount
	room.Gold += amount
	return nil
}
```

Note: `Room.RemoveItem` returns void, not bool — so for floor-to-backpack
transfers we use `removeFrom` as a plain `func(items.Item)` (no return
value to check). The item was already confirmed to exist via `FindOnFloor`
before calling the transfer.

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: clean build

- [ ] **Step 3: Commit**

```bash
git add internal/actions/transfer.go
git commit -m "feat: atomic transfer primitives with rollback for item safety"
```

---

### Task 2: Transfer Primitive Tests

**Files:**
- Create: `internal/actions/transfer_test.go`

- [ ] **Step 1: Create test helpers and conservation tests**

This file needs test helpers to create characters and rooms with items,
plus conservation invariant tests for every transfer primitive.

Read the existing test infrastructure in `internal/usercommands/usercommands_test.go`
to understand `seedAllRegistries()` and how test characters/rooms are
created. Replicate the pattern.

Test helpers needed:

```go
// countCharItems returns total items across all character inventory pools
func countCharItems(c *characters.Character) int {
    return len(c.Items) + len(c.ComponentItems) + len(c.PotionItems)
}

// countFloorItems returns total items on room floor (not stash)
func countFloorItems(r *rooms.Room) int {
    return len(r.Items)
}

// totalGold returns combined gold across character and room
func totalGold(c *characters.Character, r *rooms.Room) int {
    return c.Gold + r.Gold
}
```

Tests to write (each asserts conservation):

1. **TestTransferItemToFloor_Happy** — item in backpack → floor.
   Assert: char lost item, room gained item, total count unchanged.

2. **TestTransferItemToFloor_NotFound** — item not in backpack.
   Assert: error returned, counts unchanged.

3. **TestTransferItemToBackpack_Happy** — item on floor → backpack.
   Assert: room lost item, char gained item, total count unchanged.

4. **TestTransferItemToBackpack_CapacityFull** — backpack at capacity.
   Assert: error returned, rollback called, item back on floor,
   counts unchanged.

5. **TestTransferItemBetweenChars_Happy** — item from char A → char B.
   Assert: A lost item, B gained item, total count unchanged.

6. **TestTransferItemBetweenChars_NotFound** — item not on source.
   Assert: error, counts unchanged.

7. **TestTransferItemBetweenChars_DestFull** — destination at capacity.
   Assert: error, item rolled back to source, counts unchanged.

8. **TestTransferGold_Happy** — 50 gold from A to B.
   Assert: A.Gold decreased, B.Gold increased, total unchanged.

9. **TestTransferGold_Insufficient** — more gold than source has.
   Assert: error, balances unchanged.

10. **TestFloorPickupGold_Happy** — gold from floor to char.
    Assert: room.Gold decreased, char.Gold increased, total unchanged.

11. **TestFloorDropGold_Happy** — gold from char to floor.
    Assert: char.Gold decreased, room.Gold increased, total unchanged.

12. **TestFloorPickupGold_Insufficient** — more gold than floor has.
    Assert: error, balances unchanged.

For creating test items, use `items.New(itemId)` or construct an
`items.Item{ItemId: N}` directly. Read the items package to find
the right approach. You may need to seed item specs with
`items.SeedItemsForTest()` if it exists, or create items inline.

For capacity testing, set the character's Strength low and fill the
backpack to trigger `StoreItem` returning false.

- [ ] **Step 2: Run tests**

Run: `go test ./internal/actions/ -v -run Transfer`
Expected: all 12 tests pass

- [ ] **Step 3: Commit**

```bash
git add internal/actions/transfer_test.go
git commit -m "test: conservation invariant tests for all transfer primitives"
```

---

### Task 3: Shared Drop + Remove Actions

**Files:**
- Create: `internal/actions/drop.go`
- Create: `internal/actions/remove_equip.go` (remove portion only)
- Modify: `internal/usercommands/drop.go`
- Modify: `internal/mobcommands/drop.go`
- Modify: `internal/usercommands/remove.go`
- Modify: `internal/mobcommands/remove.go`

- [ ] **Step 1: Create shared drop action**

`internal/actions/drop.go`:

```go
package actions

import (
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// DropItemResult contains the result of a drop action.
type DropItemResult struct {
	Item  items.Item
	Found bool
	Err   error
}

// DropItem finds an item by name in the actor's backpack and moves
// it to the room floor using the atomic transfer primitive.
func DropItem(actor Actor, itemName string) DropItemResult {
	char := actor.GetCharacter()
	room := actor.GetRoom()

	matchItem, found := char.FindInBackpack(itemName)
	if !found {
		return DropItemResult{Found: false}
	}

	err := TransferItemToFloor(
		matchItem, char,
		actor.GetUserId(), actor.GetMobInstanceId(),
		room,
	)

	return DropItemResult{Item: matchItem, Found: true, Err: err}
}
```

- [ ] **Step 2: Create shared remove action**

`internal/actions/remove_equip.go`:

```go
package actions

import (
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// RemoveEquipResult contains the result of an equipment removal.
type RemoveEquipResult struct {
	Item  items.Item
	Found bool
	Err   error
}

// RemoveEquipment finds equipped item by name, unequips it,
// and stores it in the actor's backpack.
func RemoveEquipment(actor Actor, itemName string) RemoveEquipResult {
	char := actor.GetCharacter()

	matchItem, found := char.FindOnBody(itemName)
	if !found {
		return RemoveEquipResult{Found: false}
	}

	// Cancel hidden buff when removing equipment
	char.CancelBuffsWithFlag(buffs.Hidden)

	if !char.RemoveFromBody(matchItem) {
		return RemoveEquipResult{Item: matchItem, Found: true,
			Err: fmt.Errorf("failed to remove from body")}
	}

	if !char.StoreItem(matchItem) {
		// Item removed from body but can't fit in backpack —
		// drop to floor as fallback to prevent loss
		actor.GetRoom().AddItem(matchItem, false)
	}

	// Fire equipment change event
	events.AddToQueue(events.EquipmentChange{
		UserId:        actor.GetUserId(),
		MobInstanceId: actor.GetMobInstanceId(),
		ItemsRemoved:  []items.Item{matchItem},
	})

	char.Validate()

	return RemoveEquipResult{Item: matchItem, Found: true}
}
```

Add `"fmt"` to the imports.

- [ ] **Step 3: Update user drop wrapper**

Read the existing `internal/usercommands/drop.go` fully first. Then
refactor the single-item drop path to use `actions.DropItem()`. Keep
the `drop all`, `drop all.item`, and `drop N gold` paths in the wrapper,
but have the single-item case delegate to the shared action.

Keep all user-specific concerns: SendText to self + room, EquipmentChange
event for gold, grenade stub.

For gold drops, use `actions.FloorDropGold()`.

- [ ] **Step 4: Update mob drop wrapper**

Read the existing `internal/mobcommands/drop.go` fully first. Refactor
single-item drop to use `actions.DropItem()`. Keep PermaGear guard and
`drop all` in wrapper.

For gold drops, use `actions.FloorDropGold()`.

- [ ] **Step 5: Update user remove wrapper**

Read `internal/usercommands/remove.go`. Refactor single-item path to
use `actions.RemoveEquipment()`. Keep cursed item check (user-specific),
`remove all` loop, and user SendText.

- [ ] **Step 6: Update mob remove wrapper**

Read `internal/mobcommands/remove.go`. Refactor single-item path to
use `actions.RemoveEquipment()`. Keep PermaGear guard and `remove all` loop.

- [ ] **Step 7: Verify build**

Run: `go build ./...`
Expected: clean build

- [ ] **Step 8: Commit**

```bash
git add internal/actions/drop.go internal/actions/remove_equip.go \
  internal/usercommands/drop.go internal/mobcommands/drop.go \
  internal/usercommands/remove.go internal/mobcommands/remove.go
git commit -m "refactor: shared Drop and Remove actions with atomic transfers"
```

---

### Task 4: Shared Equip Action

**Files:**
- Modify: `internal/actions/remove_equip.go` (add equip)
- Modify: `internal/usercommands/equip.go`
- Modify: `internal/mobcommands/equip.go`

- [ ] **Step 1: Add shared equip action**

Add to `internal/actions/remove_equip.go`:

```go
// EquipItemResult contains the result of an equip action.
type EquipItemResult struct {
	Item          items.Item
	DisplacedItems []items.Item
	Found         bool
	Equipped      bool
	FailureReason string
}

// EquipItem finds an item by name in backpack, validates it's
// wearable, and equips it. Displaced items go back to backpack.
func EquipItem(actor Actor, itemName string) EquipItemResult {
	char := actor.GetCharacter()

	matchItem, found := char.FindInBackpack(itemName)
	if !found {
		return EquipItemResult{Found: false}
	}

	// Type check
	itemSpec := matchItem.GetSpec()
	if itemSpec == nil {
		return EquipItemResult{Found: true, FailureReason: "invalid item"}
	}
	if itemSpec.Type != items.Weapon && itemSpec.Subtype != items.Wearable {
		// Check if it's a wearable type
		if !itemSpec.IsWearable() && !itemSpec.IsWeapon() {
			return EquipItemResult{Found: true,
				FailureReason: "that can't be worn or wielded"}
		}
	}

	// Remove from backpack first
	char.RemoveItem(matchItem)

	// Attempt to wear
	displaced, worn, reason := char.Wear(matchItem)
	if !worn {
		// Failed — put it back
		char.StoreItem(matchItem)
		return EquipItemResult{Item: matchItem, Found: true,
			Equipped: false, FailureReason: reason}
	}

	// Store displaced items in backpack
	for _, di := range displaced {
		if !char.StoreItem(di) {
			// Backpack full — drop to floor
			actor.GetRoom().AddItem(di, false)
		}
	}

	char.Validate()

	// Fire equipment change event
	events.AddToQueue(events.EquipmentChange{
		UserId:        actor.GetUserId(),
		MobInstanceId: actor.GetMobInstanceId(),
		ItemsWorn:     []items.Item{matchItem},
		ItemsRemoved:  displaced,
	})

	return EquipItemResult{
		Item: matchItem, DisplacedItems: displaced,
		Found: true, Equipped: true,
	}
}
```

Note: The type check logic above is approximate — read `items.ItemSpec`
to find the correct way to check if an item is a weapon or wearable.
The existing user equip.go has this check — copy the exact pattern.

- [ ] **Step 2: Update user equip wrapper**

Read `internal/usercommands/equip.go` fully. Refactor single-item path
to use `actions.EquipItem()`. Keep user-specific: `equip all` → Gearup,
extra arm slot targeting, WornBuffIds script trigger, quest notification,
user SendText.

- [ ] **Step 3: Update mob equip wrapper**

Read `internal/mobcommands/equip.go` fully. Refactor single-item path
to use `actions.EquipItem()`. Keep mob-specific: PermaGear guard,
`equip all` loop, `equip random`, same-item no-op guard.

- [ ] **Step 4: Verify build**

Run: `go build ./...`
Expected: clean build

- [ ] **Step 5: Commit**

```bash
git add internal/actions/remove_equip.go \
  internal/usercommands/equip.go internal/mobcommands/equip.go
git commit -m "refactor: shared Equip action with atomic transfers"
```

---

### Task 5: Shared Get Action

**Files:**
- Create: `internal/actions/get.go`
- Modify: `internal/usercommands/get.go`
- Modify: `internal/mobcommands/get.go`

- [ ] **Step 1: Create shared get actions**

`internal/actions/get.go`:

```go
package actions

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// GetItemResult contains the result of a floor pickup.
type GetItemResult struct {
	Item  items.Item
	Found bool
	Err   error
}

// GetItemFromFloor finds an item by name on the room floor and
// moves it to the actor's backpack using atomic transfer.
func GetItemFromFloor(actor Actor, itemName string, stash bool) GetItemResult {
	room := actor.GetRoom()

	matchItem, found := room.FindOnFloor(itemName, stash)
	if !found {
		return GetItemResult{Found: false}
	}

	char := actor.GetCharacter()
	err := TransferItemToBackpack(
		matchItem, char,
		actor.GetUserId(), actor.GetMobInstanceId(),
		func(i items.Item) { room.RemoveItem(i, stash) },
		func(i items.Item) { room.AddItem(i, stash) },
	)

	return GetItemResult{Item: matchItem, Found: true, Err: err}
}

// GetGoldFromFloor picks up gold from the room floor.
func GetGoldFromFloor(actor Actor, amount int) error {
	return FloorPickupGold(amount, actor.GetCharacter(), actor.GetRoom())
}
```

- [ ] **Step 2: Update user get wrapper**

Read `internal/usercommands/get.go` fully (531 lines). This is the most
complex wrapper. Refactor ONLY the basic floor pickup path to use
`actions.GetItemFromFloor()`. All other paths stay in the wrapper:
- `get all` sub-routing
- `get all.item` wildcard loop
- `from bag`/`from bandolier` sub-inventory access
- Container retrieval
- Pet inventory
- Exploding item guard
- Encumbrance warning
- Stash recovery
- Hidden container discovery
- Corpse/noun fallbacks
- Visibility check

The basic floor pickup is the path that calls `room.FindOnFloor` →
`room.RemoveItem` → `char.StoreItem` → `ItemOwnership` event.
Replace that sequence with `actions.GetItemFromFloor()`.

- [ ] **Step 3: Update mob get wrapper**

Read `internal/mobcommands/get.go` (104 lines). Simpler — refactor both
the `get all` loop and single-item path to use `actions.GetItemFromFloor()`.
For gold, use `actions.GetGoldFromFloor()`.

- [ ] **Step 4: Verify build**

Run: `go build ./...`
Expected: clean build

- [ ] **Step 5: Commit**

```bash
git add internal/actions/get.go \
  internal/usercommands/get.go internal/mobcommands/get.go
git commit -m "refactor: shared Get action with atomic transfers"
```

---

### Task 6: Shared Give Action

**Files:**
- Create: `internal/actions/give.go`
- Modify: `internal/usercommands/give.go`
- Modify: `internal/mobcommands/give.go`

- [ ] **Step 1: Create shared give actions**

`internal/actions/give.go`:

```go
package actions

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// GiveItemResult contains the result of an item give.
type GiveItemResult struct {
	Item  items.Item
	Found bool
	Err   error
}

// GiveItemToChar finds an item in the source actor's backpack and
// transfers it to a target character using atomic transfer.
func GiveItemToChar(
	actor Actor,
	itemName string,
	toChar *characters.Character,
	toUserId int,
	toMobInstanceId int,
) GiveItemResult {
	char := actor.GetCharacter()

	matchItem, found := char.FindInBackpack(itemName)
	if !found {
		return GiveItemResult{Found: false}
	}

	err := TransferItemBetweenChars(
		matchItem,
		char, actor.GetUserId(), actor.GetMobInstanceId(),
		toChar, toUserId, toMobInstanceId,
	)

	return GiveItemResult{Item: matchItem, Found: true, Err: err}
}

// GiveGoldToChar transfers gold from actor to target character.
func GiveGoldToChar(actor Actor, amount int, to *characters.Character) error {
	return TransferGold(amount, actor.GetCharacter(), to)
}
```

- [ ] **Step 2: Update user give wrapper**

Read `internal/usercommands/give.go` fully. Refactor the item transfer
paths to use `actions.GiveItemToChar()` and `actions.GiveGoldToChar()`.

Keep in user wrapper:
- Preposition stripping and target parsing
- Give to mob: quest engine `item_give` notification (with ConsumeItem
  flag handling — if quest consumed the item, skip the normal transfer)
- `scripting.TryMobScriptEvent("onGive")` after transfer
- Mob default gearup behavior
- Give to pet path
- Self-give gold easter egg
- Dual EquipmentChange events for gold
- All SendText formatting

IMPORTANT: The quest engine intercept in give.go is critical. The
current flow is: transfer item to mob FIRST, then fire onGive script.
With the shared action, the flow should stay the same but use
`TransferItemBetweenChars`. If the quest engine handles the item_give
event and sets ConsumeItem, the item should be removed from the player
but NOT added to the mob — read the existing code carefully to preserve
this behavior.

- [ ] **Step 3: Update mob give wrapper**

Read `internal/mobcommands/give.go`. Refactor item/gold transfer
paths to use `actions.GiveItemToChar()` and `actions.GiveGoldToChar()`.
Keep mob-specific target resolution and room messaging.

- [ ] **Step 4: Verify build**

Run: `go build ./...`
Expected: clean build

- [ ] **Step 5: Commit**

```bash
git add internal/actions/give.go \
  internal/usercommands/give.go internal/mobcommands/give.go
git commit -m "refactor: shared Give action with atomic transfers"
```

---

### Task 7: Shared Go Action

**Files:**
- Create: `internal/actions/go.go`
- Modify: `internal/usercommands/go.go`
- Modify: `internal/mobcommands/go.go`

The go command has the most divergence. The shared core is intentionally
thin — just exit lookup and room transition mechanics.

- [ ] **Step 1: Create shared go actions**

`internal/actions/go.go`:

```go
package actions

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// FindExitResult contains the result of an exit lookup.
type FindExitResult struct {
	ExitName   string
	RoomId     int
	Found      bool
	UsesKey    bool
	IsLocked   bool
}

// FindExit looks up an exit by name/direction in the given room.
// Returns exit info or Found=false.
func FindExit(room *rooms.Room, exitName string) FindExitResult {
	exitInfo, exitName := room.FindExitByName(exitName)
	if exitInfo == nil {
		return FindExitResult{Found: false}
	}
	return FindExitResult{
		ExitName: exitName,
		RoomId:   exitInfo.RoomId,
		Found:    true,
		UsesKey:  exitInfo.HasLock(),
		IsLocked: exitInfo.IsLocked(),
	}
}

// MoveActorResult contains the result of a room transition.
type MoveActorResult struct {
	FromRoom *rooms.Room
	ToRoom   *rooms.Room
	Success  bool
}

// MoveActor handles the room transition for any actor.
// The caller is responsible for all pre-checks (combat lock,
// stamina, buffs, etc.) and post-move effects (messages,
// scripts, discovery).
//
// This function:
// 1. Loads the destination room
// 2. Removes actor from current room
// 3. Adds actor to destination room
// 4. Updates character RoomId
// 5. Fires RoomChange event
func MoveActor(actor Actor, toRoomId int) MoveActorResult {
	toRoom := rooms.LoadRoom(toRoomId)
	if toRoom == nil {
		return MoveActorResult{Success: false}
	}

	fromRoom := actor.GetRoom()
	char := actor.GetCharacter()

	// The actual room transition — caller handles user/mob
	// specific add/remove (users use different room methods
	// than mobs). Return the rooms so the wrapper can do
	// the actor-specific transition.
	char.RoomId = toRoomId

	events.AddToQueue(events.RoomChange{
		UserId:        actor.GetUserId(),
		MobInstanceId: actor.GetMobInstanceId(),
		FromRoomId:    fromRoom.RoomId,
		ToRoomId:      toRoomId,
	})

	return MoveActorResult{
		FromRoom: fromRoom,
		ToRoom:   toRoom,
		Success:  true,
	}
}
```

Note: The actual room add/remove calls differ between users and mobs
(users use `rooms.MoveToRoom` or similar, mobs use `room.RemoveMob` /
`room.AddMob`). Read both go.go files to understand the exact room
transition calls. The shared MoveActor may need to be adjusted to
either include the add/remove or let the wrapper handle it. If the
room transition methods are too different, MoveActor should just do
the RoomId update + event, and let wrappers handle add/remove.

- [ ] **Step 2: Update user go wrapper**

Read `internal/usercommands/go.go` fully. Replace the exit lookup with
`actions.FindExit()`. Use `actions.MoveActor()` for the room transition
where appropriate.

Keep in user wrapper:
- Combat lock (Aggro != nil) with death-room exception
- Quest sequence lock
- Crafting state cancellation
- NoMovement buff guard
- Encumbrance stamina cost
- Sneak detection and stamina multiplier
- Hidden mob detection on room entry
- Room script interception
- All departure/arrival SendText messages

- [ ] **Step 3: Update mob go wrapper**

Read `internal/mobcommands/go.go` fully. Replace exit lookup with
`actions.FindExit()`. Use `actions.MoveActor()` for standard exit
movement.

Keep in mob wrapper:
- NoMovement buff guard
- Numeric room ID teleport (mob-only)
- `home` keyword → pathto
- Lock check (mobs refuse locked exits)
- Darkness-aware movement messages
- Waypoint/onPath script hooks

- [ ] **Step 4: Verify build**

Run: `go build ./...`
Expected: clean build

- [ ] **Step 5: Commit**

```bash
git add internal/actions/go.go \
  internal/usercommands/go.go internal/mobcommands/go.go
git commit -m "refactor: shared Go action with FindExit and MoveActor"
```

---

### Task 8: Economy Parity + Conservation Tests

**Files:**
- Create: `internal/actions/economy_test.go`

- [ ] **Step 1: Write parity and conservation tests for all economy actions**

Tests needed (each tests with both UserActor and MobActor where
applicable, and asserts conservation invariants):

**Drop tests:**
1. DropItem happy path — item moves from backpack to floor, counts conserved
2. DropItem not found — returns Found=false, counts unchanged
3. FloorDropGold — gold moves, total conserved
4. FloorDropGold insufficient — error, balances unchanged

**Remove tests:**
5. RemoveEquipment happy path — item moves from body to backpack
6. RemoveEquipment not found — Found=false
7. RemoveEquipment cancels hidden buff

**Equip tests:**
8. EquipItem happy path — item moves from backpack to body
9. EquipItem displaces existing — old item in backpack, new equipped
10. EquipItem not found — Found=false
11. EquipItem non-wearable — Equipped=false with reason

**Get tests:**
12. GetItemFromFloor happy path — item moves from floor to backpack, conserved
13. GetItemFromFloor not found — Found=false
14. GetItemFromFloor capacity full — error, item stays on floor, conserved
15. GetGoldFromFloor — gold moves, conserved

**Give tests:**
16. GiveItemToChar happy path — item transfers between chars, conserved
17. GiveItemToChar not found — Found=false
18. GiveItemToChar dest full — error, item stays with source, conserved
19. GiveGoldToChar — gold transfers, conserved
20. GiveGoldToChar insufficient — error, unchanged

**Go tests:**
21. FindExit found — returns correct room ID
22. FindExit not found — Found=false

- [ ] **Step 2: Run tests**

Run: `go test ./internal/actions/ -v -run "Drop|Remove|Equip|Get|Give|Go"`
Expected: all tests pass

- [ ] **Step 3: Commit**

```bash
git add internal/actions/economy_test.go
git commit -m "test: parity + conservation tests for all economy actions"
```

---

### Task 9: Final Verification

- [ ] **Step 1: Full build**

Run: `go build ./...`
Expected: clean build

- [ ] **Step 2: Run all tests**

Run: `go test ./internal/actions/ ./internal/usercommands/ ./internal/mobcommands/ ./internal/combat/ -count=1 -timeout 120s`
Expected: all tests pass

- [ ] **Step 3: Manual smoke test**

Start server locally and verify:
- `drop sword` works (player drops item to floor)
- `get sword` works (player picks up from floor)
- `give sword merchant` works (item transfers)
- `equip sword` / `remove sword` work
- `go north` works for player movement
- Mob NPCs still function (combat, idle behavior, item handling)
- No item duplication or loss during normal play

- [ ] **Step 4: Final commit if fixups needed**

```bash
git add -A
git commit -m "fix: substage 2 smoke test fixups"
```
