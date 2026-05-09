# Bank Storage Stacking — Design

**Date:** 2026-05-05
**Status:** Approved, ready for plan
**Scope:** Refactor `users.Storage` (the per-player bank) to stack
identical items into a single slot, instead of recording every item
as its own entry. Update commands, display, capacity accounting,
storage-fee billing, and migrate existing player data.

## Context

Today every item in a player's bank takes its own slot in
`users.Storage.Items []items.Item`. Capacity is `len(Items)` against
the room's `storagecapacity` (1000 in Thornwall). A player with 30
iron ore consumes 30 slots and pays 30g/month in storage fees. With
crafting mats accumulating in normal play and ranged ammo coming
soon (arrows, bolts), this scales poorly and feels punitive for the
exact items players want to hoard.

Inventory already does *display-only* stacking on the same key
(ItemId + Uses + EnchantType + EnchantTier) — see CLAUDE.md
"Inventory & Item Disambiguation". Banks should go further and
stack at the data layer, so capacity and billing reflect stacks
rather than units.

## Goals

- Stackable items in the bank consume one slot per stack regardless
  of count.
- Capacity (`storagecapacity`) becomes a slot/stack ceiling.
- Storage fee (`StorageFeePerItem`) bills per slot, so a stack of N
  costs the same as a single item.
- Multi-quantity `storage add`/`storage remove` for ergonomic bulk
  moves (mirror of `buy 5 iron ingot`).
- Existing player banks migrate transparently on first
  load/validate. No flag day, no manual cleanup.

## Non-goals

- Per-stack count cap (no ceiling — slot count is the only limit).
- Cross-bank transfer or inter-player banking.
- UI to inspect stack composition over time.
- Score formula changes — separate spec.

## Stack key

Two items stack iff they match on every per-instance field that
differentiates an item visually or mechanically. Centralized as a
single helper in `internal/items`:

```go
// SameStack returns true if a and b are interchangeable enough to
// stack into one bank slot or one inventory display row.
func SameStack(a, b Item) bool
```

Equality compared:

- `ItemId`
- `Uses`
- `EnchantType`
- `EnchantTier`
- `EnchantBonus`
- `BottleMultiplier` (potions — different bottles age at different
  rates so they're mechanically distinct)
- `StatMods` (deep equal of the map)
- `StatModsType`

Inventory display stacking (currently inline in inventory render
code) and bank storage stacking both call `SameStack` so the rule
stays consistent.

## Stackability predicate

Whether an item is *eligible* to stack depends on its spec, not its
state. New helper on `items.ItemSpec`:

```go
// IsStackable reports whether two instances of this spec can ever
// occupy the same bank slot. Items with no per-instance state of
// their own (ammo, food, components, potions, grenades) return
// true; gear and quest items return false.
func (s *ItemSpec) IsStackable() bool
```

True iff `Type` is one of: `Component`, `Food`, `Drink`, `Potion`,
`Grenade`, `Ammo` (forward-declared for arrows/bolts; harmless to
list before the gear ships).

`IsStackable` is type-eligibility only. Per-instance variation
(StatMods, enchantments, Uses, BottleMultiplier) is still handled
by `SameStack` — two different stat-modifying potions go to two
slots, two identical ones go to one. A non-stackable item always
lives in its own slot with `Count: 1`, even if it shares an
ItemId with another non-stackable.

## Data shape

```go
type Storage struct {
    Slots []StorageSlot `yaml:"slots"`

    // Legacy field. Read on load for migration, then cleared.
    // Kept in struct (omitempty) for one release as a safety net.
    Items []items.Item `yaml:"items,omitempty"`
}

type StorageSlot struct {
    Item  items.Item `yaml:"item"`
    Count int        `yaml:"count"` // ≥ 1
}
```

Capacity = `len(Slots)`. A slot's `Count` has no engine-imposed
ceiling.

## Commands

`storage add` and `storage remove` gain optional leading quantity,
mirroring `buy`'s parser:

| Form                           | Behavior                                     |
|--------------------------------|----------------------------------------------|
| `storage add iron-ore`         | Move 1 from inventory → bank stack           |
| `storage add 5 iron-ore`       | Move 5; cap at how many the player has       |
| `storage add all iron-ore`     | Move every matching unit from inventory      |
| `storage add all`              | Existing behavior (deposit everything)       |
| `storage remove iron-ore`      | Move 1 from bank stack → inventory           |
| `storage remove 5 iron-ore`    | Move 5; cap at stack size                    |
| `storage remove all.iron-ore`  | Move the entire stack                        |
| `storage remove 3`             | Existing numbered slot remove                |

`storage` (no args) renders the slot list; each line shows
`<itemname> (xN)` when `N > 1`, plain `<itemname>` when `N == 1`.

`storage add all` and `storage add all <name>` walk both the
backpack *and* the on-character component bag, fixing today's gap
where players have to `unsort` to bank components.

Carry-capacity check on `remove` is per-unit: removing 5 iron ore
into a near-full backpack stops at the unit count that fits, with
a "You can't carry the rest." message.

## Display

`storage` output groups by slot, format `iron-ore (x12)`. Same
grammar inventory already uses.

## Storage fee compatibility

`internal/hooks/StorageFee_MonthlyCharge.go` becomes:

```go
itemCount := len(u.ItemStorage.Slots)
fee := itemCount * feePerItem
```

`StorageFeePerItem` config knob keeps its name and current default
(1g). Forfeiture sorts slots by per-stack value
(`spec.Value × Count`) ascending and drops the cheapest *whole*
slots until the shortfall is covered. No partial-stack peeling —
billing is per-slot, forfeiture is per-slot.

Forfeiture notice still names the items lost ("a stack of 12 iron
ore" reads cleaner than the unit list when stacks are involved —
include the count when `Count > 1`).

## Persistence migration

On `users.UserRecord` load (the existing validate hook that
`migrate_enchantments.go` already piggybacks on):

1. If `Storage.Items` is non-empty:
   - For each legacy item, check `IsStackable()` and look for an
     existing matching slot via `SameStack`.
   - If found: `slot.Count++`.
   - Else: append a new `StorageSlot{Item: itm, Count: 1}`.
2. Set `Storage.Items = nil`.
3. Mark the user dirty so the next persist writes the new shape.

Online players: migrated on the next periodic save tick.
Offline players: migrated on next login. No mass cleanup needed.

The legacy `Items` field stays in the struct (`omitempty`) for one
release. Following release: remove the field and the migration
branch.

## Stack mutation invariants

- `AddItem(itm)` looks for a stackable match first; falls back to a
  new slot.
- `RemoveItem(itm)` (existing API, called by storage fee forfeiture
  among others) removes one matched unit. If a stack hits Count 0,
  drop the slot.
- Save/load round-trips through the new field unchanged. (Test:
  marshal → unmarshal → SameStack check.)

## Test plan

Unit tests in `internal/users/storage_test.go`:

- `AddItem` of identical components folds into one slot with Count 2.
- `AddItem` of two distinct enchantments of the same ItemId stays
  in two slots.
- `AddItem` of a non-stackable type (sword) never folds.
- `RemoveItem` decrements Count, drops slot at 0.
- Migration: a Storage with 3 iron ore + 1 sword in legacy `Items`
  loads as `[{iron-ore, 3}, {sword, 1}]` slots.
- Storage fee: 1 slot of Count 50 charges 1g, not 50g.
- Forfeiture: cheapest stack-value slot drops first.

Integration test in `internal/usercommands/storage_test.go` (new
file or extension):

- `storage add 5 iron-ore` consumes 5 from inventory, banks 5.
- `storage add 5 iron-ore` when player has only 3 banks 3 with a
  warning.
- `storage remove all.iron-ore` empties the stack.
- `storage` display shows `(x12)` suffix for Count > 1.

## Out of scope

- Per-stack count caps.
- Stack splitting commands (`storage split`).
- Cross-bank stack merging UI.
- Inventory data-layer stacking — display-only stacking there
  stays as-is.
- Score formula audit — separate spec.

## Files touched

- `internal/users/storage.go` — new `StorageSlot`, new methods.
- `internal/users/storage_test.go` — unit tests.
- `internal/items/item.go` (or sibling) — new `SameStack` and
  `ItemSpec.IsStackable`.
- `internal/usercommands/storage.go` — quantity parsing, new
  `add all` extension to walk component bag.
- `internal/usercommands/storage_test.go` — integration tests
  (new).
- `internal/hooks/StorageFee_MonthlyCharge.go` — bill per slot,
  forfeit per slot, name stacks in notices.
- `internal/characters/migrations.go` (or `validate.go`) — wire
  the legacy-Items migration into the existing validate flow.
- `_datafiles/world/dogmud/templates/help/bank.template` — update
  fee description if it currently says "per item."
