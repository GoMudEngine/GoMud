# Inventory Rework — Design Spec (2026-03-29)

Seven interconnected inventory improvements addressing player feedback
from the prod scrape. Built bottom-up: parser first, then features on top.

---

## 1. Parser Enhancement — N.item and all.item

### Problem

Players expect diku-style `3.dagger` disambiguation. The engine supports
`dagger#3` via `util.GetMatchNumber()` but not the dot prefix format.
Players also want `drop all.dagger` to drop all daggers.

### Current Implementation

`util.GetMatchNumber(input string) (string, int)` in `internal/util/util.go:288-304`
splits on `#` and returns `(itemName, matchNumber)`. All item-finding
functions (`FindMatchIn`, `FindInBackpack`, `FindOnBody`, `FindOnFloor`)
already use this, so extending the parser propagates everywhere.

### Fix

Extend `GetMatchNumber()` to parse three formats:

1. **`N.item`** — if input starts with digits followed by `.`, split there.
   Edge case: `st.elmo` won't trigger because `"st"` isn't a number.
2. **`all.item`** — if input starts with `all.`, return `(itemName, -1)`
   where -1 is a sentinel meaning "all matching items".
3. **`item#N`** — existing logic, unchanged.
4. **Default** — `(input, 1)`.

Commands that support "all" semantics (`get`, `drop`) check for -1 and
loop over all matches. Commands that don't support it treat -1 as 1.

### Files Changed

- `internal/util/util.go` — extend `GetMatchNumber()`

---

## 2. Inventory Stacking (Display Only)

### Problem

Five iron ingots show as five separate lines in inventory. Players want
duplicate items grouped with a count.

### Current Implementation

`inventory.go:115-128` iterates all backpack items individually, building
parallel `itemNames` and `itemNamesFormatted` slices. The template renders
them comma-separated.

### Fix

Before passing items to the template, group items by a **stack key**:
`ItemId` + `EnchantType` + `EnchantTier` + `Uses` (for consumables).

Items with identical stack keys collapse into one entry with a count.
Storage is unchanged — items remain individual in the `Items` slice.
Stacking is purely a display concern.

Display format:
```
Carrying: iron ingot (x5), healing potion (x3), keen dagger,
          shimmering dagger, rusty sword
```

Rules:
- Stacked items show `(xN)` after the name when N > 1
- Items with different enchantments are NOT stacked (different stack key)
- Partially-used consumables are NOT stacked with full ones (different
  `Uses` value means different stack key)
- Single items display normally with no count

### Files Changed

- `internal/usercommands/inventory.go` — grouping logic before template

---

## 3. Carry Capacity Rework

### Problem

Current capacity is `Strength x 3.0` — a baseline Str 100 character
can carry 300 lbs, which is far too generous. The display shows raw
numbers `(5.2/45.0 lbs)` which leaks internal values.

### Fix — Three Parts

**A. Reduce capacity via config multiplier:**

Add `CarryCapacityMultiplier` to Balance config (default `0.65`).
Baseline Str 100 = ~65 lbs capacity (~78% reduction from 300).

`CarryCapacity()` changes from:
```go
return float64(c.Stats.Strength.ValueAdj) * 3.0
```
to:
```go
return float64(c.Stats.Strength.ValueAdj) * bal.CarryCapacityMultiplier
```

**B. Replace numeric display with colored tier labels:**

| Ratio (weight/capacity) | Label | ANSI Color |
|---|---|---|
| 0-25% | light | green |
| 25-50% | moderate | yellow |
| 50-75% | heavy | red |
| 75-100% | overburdened | red-bold |
| 100%+ | crushed | magenta-bold |

The inventory command passes `EncumbranceLabel` and `EncumbranceColor`
to the template instead of the raw `Count` string.

Template display:
```
Carrying: iron ingot, dagger    Encumbrance: [light]
```
Where `[light]` is colored green.

**C. Prompt token `{enc}`:**

Add `{enc}` to `ProcessPromptString()` in `userrecord.prompt.go`. Renders
the colored tier label inline. Off by default — players opt in via
`set prompt ... {enc} ...`.

Document in `set-prompt.template` help file.

**Existing encumbrance penalties (verified, no changes needed):**
- Movement stamina: 1-5x multiplier when over capacity (`go.go:424-435`)
- Combat swings: up to 50% reduction when over capacity
  (`combat_helpers.go:93-100`)

### Files Changed

- `internal/characters/character.go` — use config multiplier
- `internal/usercommands/inventory.go` — compute tier label
- `_datafiles/world/dogmud/templates/character/inventory.template` — display tier
- `_datafiles/config.yaml` (Balance section) — add `CarryCapacityMultiplier`
- `internal/users/userrecord.prompt.go` — add `{enc}` case
- `_datafiles/world/dogmud/templates/help/set-prompt.template` — document token

---

## 4. Enchanting Disambiguation

### Problem

`crafting.FindTargetItem()` returns the first backpack item matching the
target type. Players can't choose which item to enchant and must drop
everything else. Equipped items can't be enchanted at all.

### Current Implementation

`FindTargetItem(inv []items.Item, targetType string) (int, bool)` in
`internal/crafting/crafting.go:206-219` iterates inventory and returns
the first match by item type. Called from `craft.go:70-78` (validation)
and `NewRound_UserRoundTick.go:264-277` (application).

### Fix — Three Parts

**A. Inline targeting:**

Allow `craft <recipe> <item-specifier>`:
- `craft enchant keen dagger` — match by name
- `craft enchant keen dagger#2` or `craft enchant keen 2.dagger` — uses
  Layer 1 disambiguation

**B. Ambiguity prompt with cancel:**

If the player types just `craft enchant keen` and multiple valid targets
exist, show a numbered list:

```
Which item do you want to enchant?
  [1] iron dagger
  [2] iron dagger | keen
  [3] steel sword (wielded)
  [0] Cancel
Choose a number, or type the item name:
```

Player responds with number or name. Typing `0`, `cancel`, or any
non-matching command aborts. Pending choice expires if the player issues
any other command (no stale prompts).

**C. Search equipped items:**

Expand search to include all equipment slots, not just backpack.
Equipped items show their slot in the list: `(wielded)`, `(worn - body)`,
etc.

**Updated function signature:**

```go
func FindTargetItem(backpack []items.Item, equipment *characters.Worn,
    targetType string, specifier string) (*items.Item, string, bool)
```

Returns: item pointer, source (`"backpack"` or slot name), found bool.

### Files Changed

- `internal/crafting/crafting.go` — update `FindTargetItem` signature
- `internal/usercommands/craft.go` — parse item specifier, ambiguity prompt
- `internal/hooks/NewRound_UserRoundTick.go` — pass equipment to FindTargetItem

---

## 5. Multi-Buy

### Problem

`buy <item>` purchases one item. No quantity support.

### Current Implementation

`buy.go:20-109` parses `buy <item> [from <merchant>]`. Calls
`tryPurchase()` once, which handles a single transaction.

### Fix

Parse a leading integer from the buy input:
- `buy 5 iron ingot` — buy 5 copies
- `buy iron ingot` — buy 1 (unchanged)

Logic:
1. Parse leading integer from `rest` (if present)
2. If remaining string after stripping number is empty, treat the whole
   input as the item name (handles items literally named "5")
3. Loop `tryPurchase()` N times
4. Check funds before each iteration — stop early if insufficient gold
5. Check carry capacity before each iteration — stop early if overburdened
6. Report: `You purchase 5 iron ingots for 50 gold.` or partial:
   `You purchased 3 of 5 iron ingots before running out of gold.`

### Files Changed

- `internal/usercommands/buy.go` — parse quantity, loop purchases

---

## 6. Look Direction Priority Fix

### Problem

When `l n` is typed and no north exit exists, the input falls through
to inventory matching where "n" fuzzy-matches items like "necklace".

### Root Cause

`look.go` checks exits via `FindExitByName()`, which returns empty when
the exit doesn't exist. The input then falls through to backpack/body
matching where short strings like "n" fuzzy-match item names.

### Fix

After the exit check block (~line 252), add an early-out: if the input
is a recognized direction alias (via `TryDirectionAlias`), never fall
through to item/mob matching. Show "There is no exit in that direction."

```go
if alias := keywords.TryDirectionAlias(lookAt); alias != lookAt {
    user.SendText("There is no exit in that direction.")
    return true, nil
}
```

### Files Changed

- `internal/usercommands/look.go` — early-out for direction aliases

---

## 7. Worn Item Targeting

### Problem

`look` and `identify` check `FindInBackpack` first. If a matching item
exists in backpack, the equipped version is unreachable. The `#N`
disambiguation doesn't help because backpack and equipment are searched
in separate calls with separate numbering.

### Fix

Create a unified search method that merges both pools:

```go
func (c *Character) FindItem(itemName string) (items.Item, string, bool)
```

Returns: item, source description (`"backpack"`, `"wielded"`,
`"worn - body"`, etc.), found bool. Searches backpack first, then all
equipment slots, as a **single pool** — so `dagger#2` can reach the
wielded dagger if the first match was in backpack.

**Commands updated** to use `FindItem`:
- `look` (item inspection section)
- `identify` / `id`

**Commands unchanged** (intentionally single-pool):
- `equip` — only searches backpack
- `remove` — only searches equipped
- `drop` — only searches backpack

### Files Changed

- `internal/characters/character.go` — new `FindItem()` method
- `internal/usercommands/look.go` — use `FindItem`
- `internal/usercommands/identify.go` — use `FindItem`

---

## Summary of All Files Changed

| File | Sections | Change |
|------|----------|--------|
| `internal/util/util.go` | 1 | N.item + all.item parsing |
| `internal/usercommands/inventory.go` | 2, 3 | Stacking display, tier labels |
| `_datafiles/.../inventory.template` | 2, 3 | Stacked format, encumbrance label |
| `internal/characters/character.go` | 3, 7 | Config multiplier, FindItem() |
| `_datafiles/config.yaml` | 3 | CarryCapacityMultiplier |
| `internal/users/userrecord.prompt.go` | 3 | {enc} prompt token |
| `_datafiles/.../set-prompt.template` | 3 | Document {enc} token |
| `internal/crafting/crafting.go` | 4 | Updated FindTargetItem |
| `internal/usercommands/craft.go` | 4 | Item specifier, ambiguity prompt |
| `internal/hooks/NewRound_UserRoundTick.go` | 4 | Pass equipment to FindTargetItem |
| `internal/usercommands/buy.go` | 5 | Quantity parsing, loop purchases |
| `internal/usercommands/look.go` | 6, 7 | Direction early-out, FindItem |
| `internal/usercommands/identify.go` | 7 | FindItem for worn items |
