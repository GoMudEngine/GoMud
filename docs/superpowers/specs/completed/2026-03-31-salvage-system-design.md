# Salvage System Design Spec

## Overview

A new `salvage` command and standalone skill that lets players break down
crafted items (or specially tagged items) to recover a portion of their
materials. Uses existing crafting stations for free, or a portable tool
purchased from Fence Dealer Siv for field salvage.

## Skill: Salvage

- New standalone skill: `salvage`
- Primary stat: **Perception** (observation helps you spot reusable parts)
- Progression multiplier: **2.0** (utility skill, infrequent use)
- Registered in the `"scavenger"` profession alongside Search and Foraging
- Fires `OnSkillUse("salvage", userId)` on every salvage attempt
- All characters start at rank 1 (same as other skills via `initAllSkills`)
- Needs a help file: `help salvage`

## What's Salvageable

Two categories:

### 1. Crafted Items (recipe reverse-lookup)

Any item that is the `output.item_id` of a known recipe is salvageable.
The system reverse-lookups the recipe by item ID to determine what
ingredients could be returned.

No new data fields needed — the recipe registry already has this info.

### 2. Tagged Items (explicit salvage returns)

Non-crafted items (mob drops, quest rewards, vendor goods) can be marked
salvageable with a new `salvage_returns` field on ItemSpec:

```yaml
salvage_returns:
  - item_tag: iron-ingot
    quantity: 1
  - item_tag: leather-strip
    quantity: 1
```

**Tagged items always require the salvage tool** (since they don't belong
to a crafting discipline and have no associated station).

### 3. Everything Else

Not salvageable. Player sees:
"You can't find anything useful to salvage from that."

## Stations & Tool

### Station Salvage (free)

Salvage at the crafting station matching the item's recipe skill:
- Blacksmithing items → forge
- Alchemy items → alchemy bench
- Tailoring items → loom
- Cooking items → kitchen
- Jewelcrafting items → jeweler's bench
- Enchanting items → enchanting circle

The `salvage` command checks whether the player's current room has the
required station, same way `craft` does.

### Tool Salvage (anywhere, costs gold)

**Salvage Kit** — sold by Fence Dealer Siv in Thornwall for 1g.
- Works anywhere, no station needed
- Unlimited uses (not consumed — it's a set of picks and pry tools)
- Must be in inventory (backpack or equipped) to use
- New item definition with `component_tag: salvage-kit` or similar for
  easy detection
- When salvaging with the tool, the command just checks for the item
  in inventory instead of checking for a room station

### Priority

If the player has both the tool AND is at the right station, station
is used (no difference in outcome — just means the tool isn't required).

## Return Rate

Each ingredient from the recipe is rolled **independently**. The
per-ingredient recovery chance scales with salvage skill:

```
chance = minChance + (maxChance - minChance) * sqrt(skill / softCap)
```

Config values (in Balance section of config.yaml):
- `SalvageMinChance`: 0.15 (15% at skill 1)
- `SalvageMaxChance`: 0.85 (hard cap — never 100%)
- `SalvageSoftCap`: 50 (matches skill soft cap)

**Expected average yields** (over many attempts):
| Skill | Per-ingredient chance | 3-ingredient all-back odds |
|-------|---------------------|---------------------------|
| 1     | ~15%                | ~0.3%                     |
| 10    | ~46%                | ~10%                      |
| 25    | ~65%                | ~27%                      |
| 50    | ~85%                | ~61%                      |

For multi-quantity ingredients (e.g., 2x iron ingot), each unit is
rolled separately. So 2x iron ingot at 50% chance → expected 1 back,
but could be 0 or 2.

## Item Consumption

**The item is always destroyed**, regardless of what materials are
recovered. Even if every ingredient roll fails and the player gets
nothing back, the item is gone. There is no "try again" — salvage is
a one-way operation.

## Time (Rounds)

Salvage time scales with the total gold value of the recipe's
ingredients (proxy for item complexity):

```
rounds = clamp(floor(totalIngredientGoldValue / 10), 1, 5)
```

- Minimum: 1 round
- Maximum: 5 rounds
- Example: ingredients worth 25g total → 2 rounds

For tagged items (no recipe), use the total gold value of the
`salvage_returns` materials instead.

Uses the same `CraftingState` pattern as crafting — player enters a
multi-round activity, can be interrupted by combat or movement.

## Player Messages

**Starting salvage:**
"You begin carefully disassembling the iron short sword..."

**Success (got materials):**
"You salvage the iron short sword and recover: 1 iron ingot,
1 leather strip."

**Zero return:**
"You attempt to salvage the iron short sword but recover
nothing useful."

**No station/tool:**
"You need a forge to salvage that, or a salvage kit."

**Not salvageable:**
"You can't find anything useful to salvage from that."

**Interrupted:**
"Your salvage attempt is interrupted!" (Item NOT consumed if
interrupted before completion — same as crafting.)

All messages use descriptive language, no numbers shown to the player.

## Command Syntax

```
salvage <item-name>          Salvage an item from inventory
salvage 2.iron short sword   Disambiguate with diku-style numbering
```

The command searches backpack and equipped items (same as `craft`
targeting). Equipped items must be unequipped first — "You need to
remove that before you can salvage it."

## ItemSpec Changes

New field on ItemSpec:

```go
SalvageReturns []SalvageReturn `yaml:"salvage_returns,omitempty"`
```

Where `SalvageReturn` is:

```go
type SalvageReturn struct {
    ItemTag  string `yaml:"item_tag"`
    Quantity int    `yaml:"quantity"`
}
```

## Config Changes

New Balance fields in `_datafiles/config.yaml`:

```yaml
SalvageMinChance: 0.15
SalvageMaxChance: 0.85
SalvageSoftCap: 50
SalvageGoldPerRound: 10    # ingredient gold value per salvage round
SalvageMaxRounds: 5
```

## Recipe Reverse-Lookup

A new function in `internal/crafting/`:

```go
func GetRecipeByOutputItemId(itemId int) *RecipeSpec
```

Builds an index on first call (map[int]*RecipeSpec), returns nil if
no recipe produces that item ID. Used by the salvage command to
determine what ingredients to roll for.

## Skill Registration

- Add `Salvage SkillTag = "salvage"` to skills.go
- Add to `"scavenger"` profession: `{Search, Foraging, Salvage}`
- Add to `SkillPrimaryStats`: `"salvage": "perception"`
- Add to `SkillProgressionMultipliers` in config: `salvage: 2.0`
- Add to the explicit skill registration in `init()`

## Help File

`_datafiles/world/dogmud/templates/help/salvage.template` — explains:
- What salvage does
- Two modes: station (free) and tool (anywhere)
- Which stations correspond to which item types
- That items are always consumed
- That skill improves recovery rate
- Where to buy the salvage kit

## Test Coverage

### Unit Tests
- `TestSalvageChance`: Verify the curve produces correct values at
  skill 1, 25, 50, and above soft cap
- `TestGetRecipeByOutputItemId`: Verify reverse-lookup returns correct
  recipe and nil for unknown items

### Integration Tests
- `TestSalvageCommand_NoTarget`: Error message
- `TestSalvageCommand_NotSalvageable`: Non-craftable, non-tagged item
- `TestSalvageCommand_NoStation`: Correct error when missing station
- `TestSalvageCommand_WithTool`: Succeeds without station when tool
  is in inventory

### Data Integrity Tests (devtools)
- Every item with `salvage_returns` must reference valid `item_tag`
  values that exist as `component_tag` on at least one item
- Every `salvage_returns` entry must have `quantity >= 1`
- Add to `helpfile_completeness_test.go`: salvage skill has a help file

## Content Generation

When using `/new-item` to create items with `salvage_returns`, the
schema and CLAUDE.md docs must be updated to document the field and
validation requirements. The content generation guide should note
that tagged salvage items need valid item_tag references.

## Files Touched (estimated)

**New files:**
- `internal/usercommands/salvage.go` — command handler
- `_datafiles/world/dogmud/templates/help/salvage.template` — help
- `_datafiles/world/dogmud/items/other-0/salvage-kit.yaml` — tool item

**Modified files:**
- `internal/skills/skills.go` — register salvage skill
- `internal/items/itemspec.go` — add SalvageReturns field
- `internal/crafting/crafting.go` — add GetRecipeByOutputItemId
- `internal/configs/config.balance.go` — add salvage config fields
- `_datafiles/config.yaml` — add salvage balance values
- `internal/usercommands/usercommands.go` — register salvage command
- `internal/devtools/helpfile_completeness_test.go` — add data tests
- Fence Dealer Siv mob YAML — add salvage kit to shop inventory
- CLAUDE.md — document salvage system
