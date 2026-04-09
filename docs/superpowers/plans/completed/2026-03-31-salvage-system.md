# Salvage System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `salvage` command that lets players break down crafted or tagged items to recover materials, governed by a new perception-based salvage skill.

**Architecture:** New `salvage` skill registered alongside existing skills. Items are salvageable via recipe reverse-lookup (crafted items) or explicit `salvage_returns` field (tagged items). Salvage uses existing crafting stations for free, or a portable salvage kit from Fence Dealer Siv. Per-ingredient recovery chance scales with skill level via sqrt curve. Multi-round timing reuses the CraftingState pattern.

**Tech Stack:** Go, existing crafting/skills/items packages, YAML data files.

**Spec:** `docs/superpowers/specs/2026-03-31-salvage-system-design.md`

---

### Task 1: Register Salvage Skill + Config Values

**Files:**
- Modify: `internal/skills/skills.go` — add SkillTag, profession, primary stat, progression multiplier
- Modify: `internal/configs/config.balance.go` — add salvage config fields + defaults
- Modify: `_datafiles/config.yaml` — add salvage balance values

- [ ] **Step 1: Add Salvage SkillTag**

In `internal/skills/skills.go`, add the constant after `Enchanting` (around line 47):

```go
Salvage       SkillTag = `salvage`       // Breaking down items for materials
```

- [ ] **Step 2: Add scavenger profession and register skill**

In the `Professions` map (around line 53), add:

```go
"scavenger": {
    Search,
    Foraging,
    Salvage,
},
```

In the explicit registration slice in `init()` (around line 366), add `Salvage` to the list.

In `SkillPrimaryStats` map (around line 263), add:

```go
"salvage": "perception",
```

In `SkillProgressionMultipliers` map (around line 292), add:

```go
Salvage: 2.0,
```

- [ ] **Step 3: Add salvage config fields to Balance struct**

In `internal/configs/config.balance.go`, add after the recipe discovery fields (around line 163):

```go
// ── SALVAGE ──────────────────────────────────────────────────────────────
SalvageMinChance    ConfigFloat `yaml:"SalvageMinChance"`    // Per-ingredient recovery chance at skill 1 (default 0.15)
SalvageMaxChance    ConfigFloat `yaml:"SalvageMaxChance"`    // Hard cap on per-ingredient chance (default 0.85)
SalvageSoftCap      ConfigInt   `yaml:"SalvageSoftCap"`      // Skill level for max curve (default 50)
SalvageGoldPerRound ConfigInt   `yaml:"SalvageGoldPerRound"` // Ingredient gold value per salvage round (default 10)
SalvageMaxRounds    ConfigInt   `yaml:"SalvageMaxRounds"`    // Maximum salvage rounds (default 5)
```

In the `Validate()` method, add defaults:

```go
// ── SALVAGE ──────────────────────────────────────────────────────────────
if b.SalvageMinChance <= 0 {
    b.SalvageMinChance = 0.15
}
if b.SalvageMaxChance <= 0 {
    b.SalvageMaxChance = 0.85
}
if b.SalvageSoftCap < 1 {
    b.SalvageSoftCap = 50
}
if b.SalvageGoldPerRound < 1 {
    b.SalvageGoldPerRound = 10
}
if b.SalvageMaxRounds < 1 {
    b.SalvageMaxRounds = 5
}
```

- [ ] **Step 4: Add values to config.yaml**

In `_datafiles/config.yaml`, after the `SkillProgressionMultipliers` section, add:

```yaml
  # ── SALVAGE ────────────────────────────────────────────────────────────────
  SalvageMinChance: 0.15      # Per-ingredient recovery chance at skill 1
  SalvageMaxChance: 0.85      # Hard cap on per-ingredient chance
  SalvageSoftCap: 50          # Skill level where curve flattens
  SalvageGoldPerRound: 10     # Ingredient gold value per salvage round
  SalvageMaxRounds: 5         # Maximum salvage duration
```

Also add `salvage: 2.0` to the `SkillProgressionMultipliers` section.

- [ ] **Step 5: Build and test**

Run: `go build ./... && go test ./internal/skills/ ./internal/configs/ -v -timeout 30s`
Expected: Clean build, all tests pass.

- [ ] **Step 6: Commit**

```bash
git commit -m "feat: register salvage skill and config values"
```

---

### Task 2: Add SalvageReturns to ItemSpec

**Files:**
- Modify: `internal/items/itemspec.go` — add SalvageReturn type and field

- [ ] **Step 1: Add SalvageReturn struct and field**

In `internal/items/itemspec.go`, add the struct before or after ItemSpec:

```go
// SalvageReturn defines a material recovered when salvaging a tagged item.
type SalvageReturn struct {
    ItemTag  string `yaml:"item_tag"`
    Quantity int    `yaml:"quantity"`
}
```

Add the field to ItemSpec after `BandolierCapacity`:

```go
SalvageReturns []SalvageReturn `yaml:"salvage_returns,omitempty"` // Custom salvage returns for non-crafted items
```

- [ ] **Step 2: Build**

Run: `go build ./...`
Expected: Clean build.

- [ ] **Step 3: Commit**

```bash
git commit -m "feat: add SalvageReturns field to ItemSpec"
```

---

### Task 3: Recipe Reverse-Lookup

**Files:**
- Modify: `internal/crafting/crafting.go` — add GetRecipeByOutputItemId function

- [ ] **Step 1: Write failing test**

Add to `internal/crafting/crafting_test.go`:

```go
func TestGetRecipeByOutputItemId(t *testing.T) {
    // Seed the registry with test data
    cleanup := seedAllRegistries()
    defer cleanup()

    // Should find a recipe that exists
    recipe := GetRecipeByOutputItemId(10014) // iron short sword
    if recipe == nil {
        // Try any known recipe output — adjust ID based on what seedAllRegistries loads
        t.Log("Skipping specific ID test — checking general functionality")
    }

    // Should return nil for unknown item ID
    recipe = GetRecipeByOutputItemId(999999)
    assert.Nil(t, recipe, "Unknown item ID should return nil")
}
```

Note: The test may need adjustment based on what test helpers exist in the crafting package. Check for `seedAllRegistries` or similar helpers. If none exist, check how other tests in that file set up recipe data.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/crafting/ -run TestGetRecipeByOutputItemId -v`
Expected: FAIL — `GetRecipeByOutputItemId` undefined.

- [ ] **Step 3: Implement the function**

In `internal/crafting/crafting.go`, add:

```go
var recipeByOutputId map[int]*RecipeSpec

// GetRecipeByOutputItemId returns the recipe that produces the given item ID,
// or nil if no recipe outputs that item. Builds an index on first call.
func GetRecipeByOutputItemId(itemId int) *RecipeSpec {
    if recipeByOutputId == nil {
        recipeByOutputId = make(map[int]*RecipeSpec)
        for _, r := range allRecipes {
            recipeByOutputId[r.Output.ItemId] = r
        }
    }
    return recipeByOutputId[itemId]
}
```

Also add a reset call in any existing `ClearRecipes()` or test cleanup function so the index rebuilds after registry changes. If no such function exists, the lazy init handles it.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/crafting/ -run TestGetRecipeByOutputItemId -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git commit -m "feat: add recipe reverse-lookup by output item ID"
```

---

### Task 4: Salvage Chance Calculation

**Files:**
- Create: `internal/crafting/salvage.go` — salvage chance math + helper functions
- Create: `internal/crafting/salvage_test.go` — tests

- [ ] **Step 1: Write failing tests**

Create `internal/crafting/salvage_test.go`:

```go
package crafting

import (
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestCalcSalvageChance(t *testing.T) {
    tests := []struct {
        name     string
        skill    int
        minPct   float64
        maxPct   float64
    }{
        {"skill 1 — near minimum", 1, 0.14, 0.20},
        {"skill 10 — low-mid", 10, 0.40, 0.52},
        {"skill 25 — mid", 25, 0.55, 0.70},
        {"skill 50 — at soft cap", 50, 0.84, 0.86},
        {"skill 100 — above cap still capped", 100, 0.84, 0.86},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            chance := CalcSalvageChance(tt.skill, 0.15, 0.85, 50)
            assert.GreaterOrEqual(t, chance, tt.minPct)
            assert.LessOrEqual(t, chance, tt.maxPct)
        })
    }
}

func TestCalcSalvageRounds(t *testing.T) {
    assert.Equal(t, 1, CalcSalvageRounds(5, 10, 5))   // 5g = 1 round (minimum)
    assert.Equal(t, 2, CalcSalvageRounds(20, 10, 5))  // 20g = 2 rounds
    assert.Equal(t, 5, CalcSalvageRounds(200, 10, 5)) // 200g = capped at 5
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/crafting/ -run "TestCalcSalvage" -v`
Expected: FAIL — functions undefined.

- [ ] **Step 3: Implement salvage.go**

Create `internal/crafting/salvage.go`:

```go
package crafting

import (
    "math"

    "github.com/GoMudEngine/GoMud/internal/items"
    "github.com/GoMudEngine/GoMud/internal/util"
)

// CalcSalvageChance returns the per-ingredient recovery probability for
// a given salvage skill level. Uses a sqrt curve:
//   chance = min + (max - min) * sqrt(clamp(skill, 0, softCap) / softCap)
func CalcSalvageChance(skill int, minChance, maxChance float64, softCap int) float64 {
    if skill < 1 {
        skill = 1
    }
    ratio := float64(skill) / float64(softCap)
    if ratio > 1.0 {
        ratio = 1.0
    }
    return minChance + (maxChance-minChance)*math.Sqrt(ratio)
}

// CalcSalvageRounds determines how many rounds a salvage attempt takes
// based on the total gold value of ingredients.
func CalcSalvageRounds(totalGoldValue int, goldPerRound int, maxRounds int) int {
    if goldPerRound < 1 {
        goldPerRound = 10
    }
    rounds := totalGoldValue / goldPerRound
    if rounds < 1 {
        rounds = 1
    }
    if rounds > maxRounds {
        rounds = maxRounds
    }
    return rounds
}

// RollSalvageReturns rolls each ingredient independently and returns
// the items recovered. Each unit of each ingredient is rolled separately.
func RollSalvageReturns(ingredients []RecipeIngredient, chance float64) []RecipeIngredient {
    var recovered []RecipeIngredient
    for _, ing := range ingredients {
        qty := 0
        for i := 0; i < ing.Quantity; i++ {
            if util.Rand(10000) < int(chance*10000) {
                qty++
            }
        }
        if qty > 0 {
            recovered = append(recovered, RecipeIngredient{
                ItemTag:  ing.ItemTag,
                Quantity: qty,
            })
        }
    }
    return recovered
}

// RollSalvageReturnsFromSpec rolls salvage returns for tagged items
// (items with SalvageReturns on their ItemSpec).
func RollSalvageReturnsFromSpec(returns []items.SalvageReturn, chance float64) []RecipeIngredient {
    var recovered []RecipeIngredient
    for _, ret := range returns {
        qty := 0
        for i := 0; i < ret.Quantity; i++ {
            if util.Rand(10000) < int(chance*10000) {
                qty++
            }
        }
        if qty > 0 {
            recovered = append(recovered, RecipeIngredient{
                ItemTag:  ret.ItemTag,
                Quantity: qty,
            })
        }
    }
    return recovered
}

// CalcIngredientGoldValue sums the gold value of all ingredients in a recipe
// by looking up each component tag's item value.
func CalcIngredientGoldValue(ingredients []RecipeIngredient) int {
    total := 0
    for _, ing := range ingredients {
        if spec := items.FindSpecByComponentTag(ing.ItemTag); spec != nil {
            total += spec.Value * ing.Quantity
        }
    }
    return total
}
```

Note: `items.FindSpecByComponentTag` may not exist — if not, you'll need to add a small helper in the items package that iterates item specs to find one by component_tag. Check if such a function exists first; if not, add one.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/crafting/ -run "TestCalcSalvage" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git commit -m "feat: salvage chance calculation and ingredient rolling"
```

---

### Task 5: Salvage Command Handler

**Files:**
- Create: `internal/usercommands/salvage.go` — the `salvage` command
- Modify: `internal/usercommands/usercommands.go` — register the command

This is the largest task. The command must:
1. Parse item name from rest string
2. Find item in player's backpack (not equipped — require unequip first)
3. Determine if salvageable (recipe reverse-lookup OR salvage_returns)
4. Check station or tool
5. Calculate rounds and start CraftingState-style multi-round activity
6. On completion: roll returns, destroy item, give materials, fire OnSkillUse

- [ ] **Step 1: Create salvage.go with the command handler**

Create `internal/usercommands/salvage.go`. The structure should follow `craft.go` patterns closely. Key sections:

```go
package usercommands

import (
    "fmt"
    "strings"

    "github.com/GoMudEngine/GoMud/internal/characters"
    "github.com/GoMudEngine/GoMud/internal/configs"
    "github.com/GoMudEngine/GoMud/internal/crafting"
    "github.com/GoMudEngine/GoMud/internal/events"
    "github.com/GoMudEngine/GoMud/internal/items"
    "github.com/GoMudEngine/GoMud/internal/rooms"
    "github.com/GoMudEngine/GoMud/internal/skills"
    "github.com/GoMudEngine/GoMud/internal/users"
    "github.com/GoMudEngine/GoMud/internal/util"
)

func Salvage(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
    if rest == "" {
        user.SendText(`Salvage what? Usage: <ansi fg="command">salvage <item></ansi>`)
        return true, nil
    }

    // Already salvaging?
    if user.Character.CraftingState != nil {
        user.SendText(`You're already busy.`)
        return true, nil
    }

    // Find item in backpack
    itemName, matchNum := util.GetMatchNumber(rest)
    itm, found := user.Character.FindInBackpack(itemName, matchNum)
    if !found {
        user.SendText(fmt.Sprintf(`You don't have "%s" in your backpack.`, rest))
        return true, nil
    }

    spec := itm.GetSpec()

    // Determine salvage source: recipe or tagged
    recipe := crafting.GetRecipeByOutputItemId(spec.ItemId)
    hasSalvageReturns := len(spec.SalvageReturns) > 0

    if recipe == nil && !hasSalvageReturns {
        user.SendText(`You can't find anything useful to salvage from that.`)
        return true, nil
    }

    // Station or tool check
    hasTool := userHasSalvageKit(user)
    requiredStation := ""
    if recipe != nil {
        requiredStation = recipe.Station
    }

    if hasSalvageReturns && recipe == nil {
        // Tagged items always require tool
        if !hasTool {
            user.SendText(`You need a salvage kit to salvage that.`)
            return true, nil
        }
    } else if requiredStation != "" && room.Station != requiredStation {
        if !hasTool {
            user.SendText(fmt.Sprintf(
                `You need a %s to salvage that, or a salvage kit.`,
                strings.ReplaceAll(requiredStation, "_", " ")))
            return true, nil
        }
    }

    // Calculate rounds
    bal := configs.GetBalanceConfig()
    var totalGold int
    if recipe != nil {
        totalGold = crafting.CalcIngredientGoldValue(recipe.Ingredients)
    } else {
        totalGold = calcSalvageReturnGoldValue(spec.SalvageReturns)
    }
    rounds := crafting.CalcSalvageRounds(totalGold,
        int(bal.SalvageGoldPerRound), int(bal.SalvageMaxRounds))

    // Start salvage (reuse CraftingState with a salvage marker)
    // Store the item index and salvage info in MiscData for resolution
    user.Character.SetTempData("salvage_item_id", itm.ItemId)
    user.Character.SetTempData("salvage_instance", itm) // the actual item to remove
    user.Character.CraftingState = &characters.CraftingState{
        RecipeId:       "salvage:" + fmt.Sprintf("%d", spec.ItemId),
        RoundsTotal:    rounds,
        RoundsComplete: 0,
    }

    user.SendText(fmt.Sprintf(
        `You begin carefully disassembling the %s...`,
        spec.NameOrDefault()))

    return true, nil
}

func userHasSalvageKit(user *users.UserRecord) bool {
    for _, itm := range user.Character.Items {
        if itm.GetSpec().ComponentTag == "salvage-kit" {
            return true
        }
    }
    return false
}
```

Note: The salvage resolution (rolling returns, destroying item, giving materials) happens in the combat round tick where CraftingState is advanced — the same place crafting resolves. You'll need to add a check there for `RecipeId` starting with `"salvage:"` and handle it differently from normal crafting.

- [ ] **Step 2: Add salvage resolution to the round tick**

Find where `CraftingState` is advanced in `internal/hooks/NewRound_UserRoundTick.go` (the crafting completion section). Add a branch: if `RecipeId` starts with `"salvage:"`, resolve as salvage instead of craft.

The salvage resolution code:
1. Parse the item ID from the RecipeId (`"salvage:10014"` → 10014)
2. Look up recipe by output ID (or get SalvageReturns from item spec)
3. Calculate salvage chance from user's salvage skill level
4. Roll returns using `crafting.RollSalvageReturns()`
5. Remove the item from inventory
6. Give recovered materials via `StoreItem()`
7. Send appropriate message (got something / got nothing)
8. Fire `OnSkillUse("salvage", userId)`
9. Clear CraftingState

- [ ] **Step 3: Register the command**

In `internal/usercommands/usercommands.go`, add to the command map:

```go
`salvage`: {Salvage, false, false, false},
```

- [ ] **Step 4: Build and test manually**

Run: `go build ./...`
Expected: Clean build. There may be missing helper functions (like `FindInBackpack`, `NameOrDefault`, `FindSpecByComponentTag`) — check what exists and adapt. The item search should follow the same pattern as `craft.go` for finding items.

- [ ] **Step 5: Commit**

```bash
git commit -m "feat: implement salvage command handler and round resolution"
```

---

### Task 6: Salvage Kit Item + Fence Dealer Siv

**Files:**
- Create: `_datafiles/world/dogmud/items/other-0/32-salvage_kit.yaml` — the tool item
- Modify: Fence Dealer Siv's mob definition — add to shop inventory

- [ ] **Step 1: Create salvage kit item**

Create `_datafiles/world/dogmud/items/other-0/32-salvage_kit.yaml`:

```yaml
itemid: 32
name: Salvage Kit
displayname: salvage kit
description: >-
  A compact roll of sturdy picks, pry bars, and cutting tools
  designed for disassembling crafted goods. The handles are worn
  smooth from frequent use.
type: object
value: 1
weight: 1.5
component_tag: salvage-kit
```

Verify the item ID 32 doesn't conflict — check that no other file in `other-0/` uses ID 32.

- [ ] **Step 2: Add to Fence Dealer Siv's shop**

Read Siv's mob file at `_datafiles/world/dogmud/mobs/thornwall_city/104-fence_dealer_siv.yaml` and add the salvage kit to the `shopstock` or equivalent field. Follow the same pattern as other merchant mobs — check how Apothecary Voss or Blacksmith Kerra list their shop items.

- [ ] **Step 3: Build and verify**

Run: `go build ./...`
Expected: Clean build.

- [ ] **Step 4: Commit**

```bash
git commit -m "feat: add salvage kit item and stock at Fence Dealer Siv"
```

---

### Task 7: Help File + Data Integrity Tests

**Files:**
- Create: `_datafiles/world/dogmud/templates/help/salvage.template`
- Modify: `internal/devtools/helpfile_completeness_test.go` — add salvage data tests

- [ ] **Step 1: Create help file**

Create `_datafiles/world/dogmud/templates/help/salvage.template`:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">salvage</ansi>

The <ansi fg="command">salvage</ansi> command lets you break down items to
recover crafting materials. Your skill determines how much you
recover -- higher skill means better returns.

<ansi fg="yellow">Usage:</ansi>
  <ansi fg="command">salvage <item></ansi>     Break down an item for materials

<ansi fg="yellow">What can be salvaged:</ansi>
  Any item that was produced by a crafting recipe, plus certain
  special items marked as salvageable.

<ansi fg="yellow">Where to salvage:</ansi>
  Crafted items can be salvaged for free at the station where
  they were made -- swords at the forge, potions at the alchemy
  bench, cloth at the loom, and so on.

  Alternatively, buy a <ansi fg="itemname">salvage kit</ansi> from Fence
  Dealer Siv in Thornwall's back alleys. With the kit in your
  pack, you can salvage anywhere.

<ansi fg="yellow">How it works:</ansi>
  Salvage always destroys the item. Each ingredient has a chance
  to be recovered based on your salvage skill. At low skill you
  may get nothing back; at high skill you recover most materials.
  You will never recover everything -- crafting always costs
  something.

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help craft</ansi>, <ansi fg="command">help skills</ansi>
```

- [ ] **Step 2: Add data integrity test for salvage_returns**

In `internal/devtools/helpfile_completeness_test.go`, add:

```go
// TestSalvageReturns_ValidTags ensures every item with salvage_returns
// references valid component_tag values.
func TestSalvageReturns_ValidTags(t *testing.T) {
    // This test loads all item specs and verifies that any
    // salvage_returns entries reference item_tags that exist
    // as component_tag on at least one item spec.
    // Implementation depends on how item specs are loaded in tests.
    // Use the same pattern as other data integrity tests in this file.
}
```

Note: The exact implementation depends on how item specs are loaded in the devtools test package. Follow the pattern of `listYAMLBasenames` and `helpFileExists` — you may need to parse item YAML files directly.

- [ ] **Step 3: Run tests**

Run: `go test ./internal/devtools/ -v -timeout 30s`
Expected: All tests pass including the new salvage ones.

- [ ] **Step 4: Commit**

```bash
git commit -m "docs: add salvage help file and data integrity tests"
```

---

### Task 8: Update CLAUDE.md + Schema Docs

**Files:**
- Modify: `CLAUDE.md` — document salvage system
- Modify: `docs/schemas/` — update item schema if it exists

- [ ] **Step 1: Add salvage section to CLAUDE.md**

Add after the Alchemy & Potions section:

```markdown
## Salvage System
Players can break down crafted items (or items with `salvage_returns` on
their ItemSpec) to recover materials. New standalone skill: `salvage`,
primary stat: Perception, progression multiplier 2.0.

### How It Works
- `salvage <item>` starts a multi-round activity (1-5 rounds based on
  ingredient gold value).
- Each ingredient is rolled independently. Chance scales with skill:
  `chance = min + (max - min) * sqrt(skill / softCap)`.
- Config: `SalvageMinChance` (0.15), `SalvageMaxChance` (0.85),
  `SalvageSoftCap` (50).
- Item is always consumed, even if no materials recovered.

### Stations & Tool
- Free at the recipe's crafting station (forge, alchemy bench, etc.).
- Salvage Kit (sold by Fence Dealer Siv, 1g) allows salvage anywhere.
- Tagged items (non-crafted with `salvage_returns`) require the tool.

### ItemSpec Fields
- `salvage_returns`: list of `{item_tag, quantity}` for non-crafted items.
  Every `item_tag` must match a valid `component_tag` on an existing item.
```

- [ ] **Step 2: Commit**

```bash
git commit -m "docs: document salvage system in CLAUDE.md"
```

---

## Manual Testing Checklist

After all tasks complete, verify on a running server:

1. `skills` — shows salvage skill at novice
2. `help salvage` — displays help text
3. Buy salvage kit from Siv — `buy salvage kit`
4. Craft an item (e.g., iron short sword at forge)
5. `salvage iron short sword` at forge — works, shows materials or nothing
6. `salvage iron short sword` away from forge without tool — error about needing forge or kit
7. `salvage iron short sword` away from forge WITH kit — works
8. `salvage gold ring` (non-craftable, no salvage_returns) — "can't find anything useful"
9. Skill progression — see "sharpening" messages over multiple salvages
10. Multi-round timing — longer for expensive items, shorter for cheap
