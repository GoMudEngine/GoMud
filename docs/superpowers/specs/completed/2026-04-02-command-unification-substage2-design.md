# Command Unification — Substage 2: Economy Commands + Go

**Date:** 2026-04-02
**Goal:** Extract shared core logic for get, drop, give, equip, remove,
and go commands. Unify behavior where possible, keep actor-specific
concerns in thin wrappers.

## Scope

Unify these 6 commands using the Actor interface + shared actions
pattern established in substage 1:

| Command | User LOC | Mob LOC | Shareability | Notes |
|---------|----------|---------|-------------|-------|
| drop | 155 | 87 | ~70% | Cleanest target |
| remove | 82 | 61 | ~75% | Cleanest target |
| equip | 204 | 105 | ~65% | Moderate |
| get | 531 | 104 | ~60% | Most complex user side |
| give | 299 | 146 | ~50% | Quest intercept on user side |
| go | ~450 | 178 | ~30% | Most divergent |

**Not in scope:** buy, sell (no mob equivalents — deferred to
substage 5 NPC behavior foundation).

## Approach

Same pattern as substage 1: shared action functions in
`internal/actions/` that handle the mechanical core, with thin
wrappers in each command package for actor-specific concerns.

The economy commands have more divergence than say/emote, so
the shared core will be narrower — focused on the item transfer
mechanics rather than the full command flow. The wrappers will
be thicker.

## Atomic Transfer Primitives

All item and gold movement flows through two chokepoint functions.
No command should call RemoveItem/StoreItem/AddItem directly for
transfers — always use these primitives.

### TransferItem

```go
// actions/transfer.go

// TransferItem atomically moves an item between two inventories.
// Removes from source first, then adds to destination. If the add
// fails (capacity, invalid state), the item is returned to source.
// Returns error only on true failure (item lost — should never happen).
func TransferItem(
    item items.Item,
    removeFrom func(items.Item) bool,
    addTo func(items.Item) bool,
    rollback func(items.Item),
) error
```

Usage examples:
- Floor to backpack: `removeFrom=room.RemoveItem, addTo=char.StoreItem, rollback=room.AddItem`
- Backpack to floor: `removeFrom=char.RemoveItem, addTo=room.AddItem, rollback=char.StoreItem`
- Player to mob: `removeFrom=player.RemoveItem, addTo=mob.StoreItem, rollback=player.StoreItem`

The function:
1. Calls `removeFrom(item)` — if false, return error (item not found)
2. Calls `addTo(item)` — if false, calls `rollback(item)` and returns error
3. On success, fires `ItemOwnership` event
4. Returns nil

This eliminates the ordering inconsistency (mob drop does add-then-remove,
user drop does remove-then-add). Both now use remove-then-add with rollback.

### TransferGold

```go
// TransferGold atomically moves gold between two characters.
// Validates source has enough gold before transferring.
func TransferGold(amount int, from *characters.Character, to *characters.Character) error
```

The function:
1. Validates `from.Gold >= amount`
2. Subtracts from source
3. Adds to destination
4. Returns nil (or error if insufficient)

No partial state is possible — gold is just an integer.

### FloorPickupGold / FloorDropGold

Similar helpers for room floor gold (room.Gold is a different
field than character inventory):

```go
func FloorPickupGold(amount int, char *characters.Character, room *rooms.Room) error
func FloorDropGold(amount int, char *characters.Character, room *rooms.Room) error
```

## Per-Command Design

### drop

**Shared core (`actions.DropItem`):**
- Find item in backpack by name
- Remove from character inventory
- Add to room floor
- Fire ItemOwnership event
- Return the item for wrapper messaging

**User wrapper keeps:**
- `drop all` / `drop all.item` loop (matchNum == -1)
- `drop N gold` with EquipmentChange event
- SendText to self + room with player ANSI colors
- Grenade type stub (disabled TODO)

**Mob wrapper keeps:**
- PermaGear buff guard (blocks all drops)
- `drop all` includes gold automatically (user excludes gold)
- SendRoomText with mob ANSI colors
- PlayerCt() < 1 optimization

**Shared helper (`actions.DropGold`):**
- Transfer gold from character to room floor
- Fire EquipmentChange event

### remove

**Shared core (`actions.RemoveEquipment`):**
- Find item on body by name
- RemoveFromBody
- StoreItem back to backpack
- CancelBuffsWithFlag(Hidden) — both sides do this
- Fire EquipmentChange event
- Return the item for wrapper messaging

**User wrapper keeps:**
- `remove all` loop
- Cursed item check (spellcasting rank >= 4 to bypass)
- SendText to self + room

**Mob wrapper keeps:**
- PermaGear buff guard
- `remove all` loop
- Room-only SendText

### equip

**Shared core (`actions.EquipItem`):**
- Find item in backpack by name
- Validate type (Weapon or Wearable)
- Call character.Wear(item)
- StoreItem any displaced items back to backpack
- character.Validate()
- Fire EquipmentChange event
- Return equipped item + any displaced items for messaging

**User wrapper keeps:**
- `equip all` → delegates to Gearup
- Extra arm slot targeting (arm1-arm4 suffix parsing)
- WornBuffIds onStart script trigger
- Quest engine "command"/"equip" notification
- SendText to self + room

**Mob wrapper keeps:**
- PermaGear buff guard
- `equip all` loop
- `equip random` keyword
- Same-item no-op guard
- Room-only SendText

### get

This is the most complex. The user version has component bag routing,
bandolier routing, pet inventory, container access, hidden container
discovery, encumbrance warnings, corpse fallback, noun fallback.

**Shared core (`actions.GetItemFromFloor`):**
- FindOnFloor by name
- RemoveItem from room
- StoreItem to character backpack
- Fire ItemOwnership event
- Return the item for wrapper messaging

**Shared helper (`actions.GetGoldFromFloor`):**
- Pick up gold from room floor
- Add to character gold

**User wrapper keeps:**
- `get all` with sub-routing (component bag, bandolier, container, floor)
- `get all.item` wildcard loop
- `from bag`/`from bandolier` suffix parsing + sub-inventory access
- Container item retrieval
- Pet inventory access
- Exploding item guard
- Encumbrance warning
- Stash recovery (auto-detect owner)
- Hidden container discovery check
- Corpse fallback, noun fallback
- Visibility check (dark room guard)
- SendText to self + room

**Mob wrapper keeps:**
- `get all` (gold then items, no containers)
- Stash/ground suffix stripping
- Room-only SendText
- No carry capacity enforcement

### give

**Shared core (`actions.GiveItem`):**
- Transfer item from source actor's backpack to target
- Fire ItemOwnership events for both sides
- Return the item for wrapper messaging

**Shared helper (`actions.GiveGold`):**
- Transfer gold between characters

**User wrapper keeps:**
- Preposition stripping, target parsing
- Give to player: dual SendText + room message
- Give to mob: quest engine `item_give` notification with ConsumeItem
  handling, `scripting.TryMobScriptEvent("onGive")`, mob default
  gearup behavior
- Give to pet: capacity check, floor spill
- Self-give gold easter egg
- EquipmentChange events for gold

**Mob wrapper keeps:**
- Same parse logic
- Give to player: player gets SendText
- Give to mob: room-only SendText
- No quest intercept, no onGive scripting

### go

The most divergent command. The shared core is thin.

**Shared core (`actions.FindExit`):**
- Look up exit by name/direction in room
- Return the exit info or nil

**Shared core (`actions.MoveActor`):**
- Remove actor from current room
- Add actor to destination room
- Fire RoomChange event (uses GetUserId/GetMobInstanceId)
- Return old room + new room for wrapper messaging

**User wrapper keeps:**
- Combat lock (Aggro != nil) with death-room exception
- Quest sequence lock check
- Crafting state cancellation
- NoMovement buff guard
- Encumbrance stamina cost scaling
- Sneak detection (buff flag + misc-data)
- Sneak stamina multiplier
- Hidden mob detection on room entry
- Room script interception
- SendText departure/arrival messages

**Mob wrapper keeps:**
- NoMovement buff guard
- Numeric room ID teleport (mob-only)
- `home` keyword → pathto delegation
- Lock check (mobs refuse locked exits)
- Darkness-aware movement messages (sendMovementMessage helper)
- Waypoint/onPath script hooks

## Testing Strategy

### Parity Tests

Each shared action gets parity tests:
- Create test items, test rooms, test characters
- Run the shared action with both UserActor and MobActor
- Assert same mechanical outcome (item moved, gold transferred,
  room state changed)
- Actor-specific side effects (SendText, quest hooks) are NOT
  tested for parity — those are intentional divergences

### Conservation Invariant Tests

Every transfer test MUST assert item/gold conservation — the total
count across all inventories is unchanged after the operation.

Pattern for every transfer test:

```go
func TestDropItem_Conservation(t *testing.T) {
    // Setup: item in backpack, nothing on floor
    countBefore := countItems(char) + countFloorItems(room)
    goldBefore := char.Gold + room.Gold

    // Act
    actions.DropItem(actor, "sword")

    // Assert conservation
    countAfter := countItems(char) + countFloorItems(room)
    goldAfter := char.Gold + room.Gold
    assert.Equal(t, countBefore, countAfter,
        "item count changed — duplication or loss")
    assert.Equal(t, goldBefore, goldAfter,
        "gold total changed — duplication or loss")

    // Assert transfer happened correctly
    assert.False(t, charHasItem(char, "sword"))
    assert.True(t, floorHasItem(room, "sword"))
}
```

Test cases that MUST exist for each transfer primitive:
1. **Happy path** — item moves, counts conserved
2. **Source missing** — item not in source, counts unchanged,
   error returned
3. **Destination full** — add fails, item rolled back to source,
   counts conserved
4. **Gold insufficient** — transfer rejected, balances unchanged

### Edge Case Tests

- Drop last item in backpack (empty inventory after)
- Get item when backpack is at capacity (should fail gracefully)
- Give item to mob that has PermaGear (should fail, item returns)
- Equip item that displaces another (both items accounted for)
- Transfer during concurrent access (if applicable — may not be
  testable without integration tests)

## Order of Implementation

1. **drop + remove** — cleanest, establish the pattern
2. **equip** — moderate complexity
3. **get** — most complex user side
4. **give** — quest intercept complexity
5. **go** — most divergent, thin shared core

## What This Does NOT Change

- Container system (backpack, component bag, bandolier) stays user-side
- Pet inventory system stays user-side
- Quest engine intercepts stay user-side
- Mob AI commands (gearup, PermaGear) stay mob-side
- Mob teleport-by-roomId stays mob-side
- Waypoint/pathing system stays mob-side

## Risks

1. **Item duplication / loss:** The primary risk in this substage.
   Mitigated by: atomic TransferItem with rollback, conservation
   invariant tests on every transfer, and funneling ALL item movement
   through the transfer primitives. No command should call
   RemoveItem/StoreItem/AddItem directly for transfers.

2. **Item transfer ordering:** Mob drop currently does AddItem then
   RemoveItem (duplication window); user does RemoveItem then AddItem
   (loss window). TransferItem standardizes on remove-first with
   rollback, eliminating both failure modes.

3. **PermaGear guard placement:** Three mob commands (drop, equip,
   remove) check PermaGear. Keeping it in mob wrappers since it's a
   mob-only concept — simpler than a shared helper.

4. **Get complexity:** The user get command is 531 lines with many
   sub-paths. The shared core will be narrow (just floor pickup).
   Most of the user-side complexity stays in the wrapper. This is
   correct — don't force-share code that's truly user-specific.

5. **Go divergence:** With only ~30% shareability, the shared core
   for go is very thin (exit lookup + room transition). This is fine —
   the value is in having a consistent Actor-based interface for room
   transitions, not in eliminating code duplication.
