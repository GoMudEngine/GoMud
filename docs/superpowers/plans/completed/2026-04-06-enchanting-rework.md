# Enchanting System Rework — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix broken enchanting targeting UX, rebalance all enchantments, add coverage for all equipment slots, add 2H weapon scaling and lifesteal mechanic.

**Architecture:** Replace item-name-based targeting with equip-slot-based targeting using `#N`/`N.` disambiguation. Store the resolved slot label in `CraftingState` so completion can use `GetSlotPointer()` directly. Add lifesteal as a new stat mod key read during combat resolution. Thornguard uses the existing `return_damage` stat mod. All 18 enchantments get 5 tiers with Chrysalis theming.

**Tech Stack:** Go, YAML data files

---

### Task 1: Add TargetSlot to CraftingState and Write Slot Resolution Helper

**Files:**
- Modify: `internal/characters/crafting.go`
- Create: `internal/usercommands/enchant_slot.go`

- [ ] **Step 1: Add TargetSlot field to CraftingState**

In `internal/characters/crafting.go`, add the new field:

```go
package characters

// CraftingState tracks an active crafting operation for a character.
// Not persisted — the field on Character uses yaml:"-".
// Logging out mid-craft silently discards progress (same behaviour as CastingState).
type CraftingState struct {
	RecipeId       string
	RoundsTotal    int
	RoundsComplete int
	TargetSlot     string // equipment slot label for enchanting (e.g. "wielded", "worn - ring2")
}
```

- [ ] **Step 2: Create the slot resolution helper**

Create `internal/usercommands/enchant_slot.go`:

```go
package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// enchantSlotCandidate pairs a slot label with the item occupying it.
type enchantSlotCandidate struct {
	SlotLabel string
	Item      items.Item
}

// resolveEnchantSlot finds the equipment slot to enchant based on the recipe's
// target_type and the player's optional slot specifier (e.g. "weapon#2", "2.ring").
//
// Returns the slot label (for GetSlotPointer), the item pointer, and an error
// message string. On success errMsg is empty.
func resolveEnchantSlot(equipment *characters.Worn, targetType string, specifier string) (slotLabel string, item *items.Item, errMsg string) {

	// Build ordered candidate list for this target type
	candidates := buildSlotCandidates(equipment, targetType)

	if len(candidates) == 0 {
		return "", nil, fmt.Sprintf(
			"You don't have a %s equipped to enchant.",
			strings.ReplaceAll(targetType, "_", " "))
	}

	// No specifier — auto-select if unambiguous
	if specifier == "" {
		if len(candidates) == 1 {
			c := candidates[0]
			ptr := equipment.GetSlotPointer(c.SlotLabel)
			return c.SlotLabel, ptr, ""
		}
		// Multiple candidates — tell the player to specify
		lines := "Which slot? You have multiple options:\n"
		for i, c := range candidates {
			lines += fmt.Sprintf("  %s#%d — %s\n",
				slotSpecName(targetType), i+1, c.Item.DisplayName())
		}
		return "", nil, lines
	}

	// Parse the specifier for #N suffix or N. prefix
	baseName, idx := util.GetMatchNumber(specifier)

	// Validate the base name matches the target type's player-facing name
	expectedName := slotSpecName(targetType)
	if baseName != "" && !strings.EqualFold(baseName, expectedName) {
		return "", nil, fmt.Sprintf(
			"That recipe targets %s slots, not %s.",
			expectedName, baseName)
	}

	// idx is 1-based
	if idx < 1 || idx > len(candidates) {
		return "", nil, fmt.Sprintf(
			"There is no %s#%d equipped.", expectedName, idx)
	}

	c := candidates[idx-1]
	ptr := equipment.GetSlotPointer(c.SlotLabel)
	if ptr == nil || ptr.ItemId < 1 {
		return "", nil, "That equipment slot is empty."
	}
	return c.SlotLabel, ptr, ""
}

// slotSpecName returns the player-facing name for a target type.
// This is what the player types as the slot specifier.
func slotSpecName(targetType string) string {
	switch targetType {
	case "offhand":
		return "shield"
	default:
		return targetType
	}
}

// buildSlotCandidates returns an ordered list of equipped items matching the
// given target type. Only slots with an item whose ItemSpec.Type matches are
// included.
func buildSlotCandidates(eq *characters.Worn, targetType string) []enchantSlotCandidate {
	var candidates []enchantSlotCandidate

	// Define the slot scan order per target type.
	// Each entry is (slot label, item pointer).
	type slotEntry struct {
		label string
		item  items.Item
	}

	var scanOrder []slotEntry

	switch targetType {
	case "weapon":
		scanOrder = []slotEntry{
			{"wielded", eq.Weapon},
			{"offhand", eq.Offhand},
			{"extra arm 1", eq.ExtraArm1},
			{"extra arm 2", eq.ExtraArm2},
			{"extra arm 3", eq.ExtraArm3},
			{"extra arm 4", eq.ExtraArm4},
		}
	case "offhand": // shields
		scanOrder = []slotEntry{
			{"offhand", eq.Offhand},
			{"extra arm 1", eq.ExtraArm1},
			{"extra arm 2", eq.ExtraArm2},
			{"extra arm 3", eq.ExtraArm3},
			{"extra arm 4", eq.ExtraArm4},
		}
	case "head":
		scanOrder = []slotEntry{{"worn - head", eq.Head}}
	case "neck":
		scanOrder = []slotEntry{{"worn - neck", eq.Neck}}
	case "shoulders":
		scanOrder = []slotEntry{{"worn - shoulders", eq.Shoulders}}
	case "body":
		scanOrder = []slotEntry{{"worn - body", eq.Body}}
	case "back":
		scanOrder = []slotEntry{{"worn - back", eq.Back}}
	case "belt":
		scanOrder = []slotEntry{{"worn - belt", eq.Belt}}
	case "wrist":
		scanOrder = []slotEntry{
			{"worn - wrist", eq.Wrist1},
			{"worn - wrist2", eq.Wrist2},
			{"extra wrist 1", eq.ExtraWrist1},
			{"extra wrist 2", eq.ExtraWrist2},
			{"extra wrist 3", eq.ExtraWrist3},
			{"extra wrist 4", eq.ExtraWrist4},
		}
	case "gloves":
		scanOrder = []slotEntry{{"worn - gloves", eq.Gloves}}
	case "ring":
		scanOrder = []slotEntry{
			{"worn - ring", eq.Ring},
			{"worn - ring2", eq.Ring2},
		}
	case "legs":
		scanOrder = []slotEntry{{"worn - legs", eq.Legs}}
	case "feet":
		scanOrder = []slotEntry{{"worn - feet", eq.Feet}}
	case "tail":
		scanOrder = []slotEntry{{"worn - tail", eq.Tail}}
	}

	for _, s := range scanOrder {
		if s.item.ItemId < 1 {
			continue
		}
		spec := s.item.GetSpec()
		if spec == nil {
			continue
		}
		if string(spec.Type) == targetType {
			candidates = append(candidates, enchantSlotCandidate{
				SlotLabel: s.label,
				Item:      s.item,
			})
		}
	}

	return candidates
}
```

- [ ] **Step 3: Build and verify**

Run: `go build ./internal/characters/ ./internal/usercommands/`
Expected: Clean build.

- [ ] **Step 4: Commit**

```bash
git add internal/characters/crafting.go internal/usercommands/enchant_slot.go
git commit -m "feat: add TargetSlot to CraftingState and slot resolution helper"
```

---

### Task 2: Rewrite craftEnchanting to Use Slot Targeting

**Files:**
- Modify: `internal/usercommands/craft.go`

- [ ] **Step 1: Replace the craftEnchanting function**

Replace the entire `craftEnchanting` function in `internal/usercommands/craft.go` (lines 106-230) with:

```go
// craftEnchanting handles the enchanting sub-path of craft, which requires
// player-specific target disambiguation not available to mob actors.
func craftEnchanting(rest string, recipe *crafting.RecipeSpec, user *users.UserRecord, room *rooms.Room) (bool, error) {
	// Known-recipe gate
	if !user.Character.HasRecipe(recipe.RecipeId) {
		user.SendText(`<ansi fg="red">You don't know that recipe yet. Keep crafting to discover new ones!</ansi>`)
		return true, nil
	}

	// Already crafting?
	if user.Character.IsCrafting() {
		user.SendText(`<ansi fg="red">You are already working on something. Finish or be interrupted first.</ansi>`)
		return true, nil
	}

	// Skill gate
	skillLevel := user.Character.GetSkillLevel(skills.SkillTag(recipe.Skill))
	if skillLevel < recipe.SkillMinimum {
		user.SendText(fmt.Sprintf(
			`<ansi fg="red">Your %s skill is too low (requires %d, you have %d).</ansi>`,
			recipe.Skill, recipe.SkillMinimum, skillLevel))
		return true, nil
	}

	// Station check
	if recipe.Station != "" && room.Station != recipe.Station {
		user.SendText(fmt.Sprintf(
			`<ansi fg="red">You need to be at a %s to craft that.</ansi>`,
			strings.ReplaceAll(recipe.Station, "_", " ")))
		return true, nil
	}

	// Ingredient check
	ok, missing := crafting.HasIngredients(user.Character.Items, user.Character.ComponentItems, recipe)
	if !ok {
		user.SendText(fmt.Sprintf(`<ansi fg="red">You are missing: %s.</ansi>`, missing))
		return true, nil
	}

	// Slot-based target resolution
	// Strip the recipe name from the input to get the optional slot specifier.
	specifier := ""
	recipeName := strings.ToLower(recipe.Name)
	restLower := strings.ToLower(strings.TrimSpace(rest))
	if strings.HasPrefix(restLower, recipeName) {
		specifier = strings.TrimSpace(rest[len(recipeName):])
	} else if strings.HasPrefix(restLower, strings.ToLower(recipe.RecipeId)) {
		specifier = strings.TrimSpace(rest[len(recipe.RecipeId):])
	}

	slotLabel, targetItem, errMsg := resolveEnchantSlot(&user.Character.Equipment, recipe.TargetType, specifier)
	if errMsg != "" {
		user.SendText(fmt.Sprintf(`<ansi fg="red">%s</ansi>`, errMsg))
		return true, nil
	}

	if targetItem == nil {
		user.SendText(`<ansi fg="red">Could not find a valid item in that slot.</ansi>`)
		return true, nil
	}

	// Safety: complete immediately if time_rounds <= 0
	if recipe.TimeRounds <= 0 {
		completeCraft(user, recipe)
		return true, nil
	}

	// Start multi-round enchanting with the resolved slot
	user.Character.CraftingState = &characters.CraftingState{
		RecipeId:    recipe.RecipeId,
		RoundsTotal: recipe.TimeRounds,
		TargetSlot:  slotLabel,
	}
	user.SendText(fmt.Sprintf(
		`<ansi fg="yellow">You begin enchanting <ansi fg="itemname">%s</ansi>... (%s)</ansi>`,
		targetItem.DisplayName(), craftTimeDesc(recipe.TimeRounds)))

	// Quest engine: command notification
	bridge := questengine.NewGameBridge(user, room.RoomId)
	questengine.GetEngine().Notify("command", questengine.EventDetails{
		UserId:  user.UserId,
		RoomId:  room.RoomId,
		Command: "craft",
	}, bridge, bridge)

	return true, nil
}
```

- [ ] **Step 2: Remove unused imports**

Remove the `"github.com/GoMudEngine/GoMud/internal/crafting"` import reference to `crafting.EquipmentSlot` if it was only used in the old enchanting code. Keep the `crafting` import itself since it's still used for `HasIngredients`, `FindRecipeByName`, etc. Also remove the `"github.com/GoMudEngine/GoMud/internal/items"` import if no longer referenced.

Check the import block and remove any that are now unused.

- [ ] **Step 3: Build and verify**

Run: `go build ./internal/usercommands/`
Expected: Clean build.

- [ ] **Step 4: Commit**

```bash
git add internal/usercommands/craft.go
git commit -m "feat: rewrite craftEnchanting to use equip-slot targeting"
```

---

### Task 3: Rewrite Enchanting Completion Path

**Files:**
- Modify: `internal/hooks/NewRound_UserRoundTick.go`

- [ ] **Step 1: Replace the enchanting completion block**

In `internal/hooks/NewRound_UserRoundTick.go`, find the block starting at line 297 (`if crafting.IsEnchantingRecipe(recipe)`) and replace the entire enchanting branch (lines 297-344) with:

```go
								if crafting.IsEnchantingRecipe(recipe) {
									// Enchanting: use the stored slot label to find the target
									targetSlot := user.Character.CraftingState.TargetSlot
									targetItem := user.Character.Equipment.GetSlotPointer(targetSlot)
									if targetItem == nil || targetItem.ItemId < 1 {
										user.SendText(`<ansi fg="red">The item is no longer equipped. The enchanting fails, but your materials are returned.</ansi>`)
										// Don't consume ingredients on slot-empty failure
									} else {
										eDef := enchantments.GetEnchantment(recipe.EnchantType)
										if eDef != nil {
											targetItem.EnchantType = recipe.EnchantType
											targetItem.EnchantTier = 0
											targetItem.EnchantUses = 0
											targetItem.ReservePool = eDef.ReservePool
											enchantments.ApplyTier(targetItem, eDef, 0)
										}
									}
								} else {
```

Note: The `else` connects to the existing normal crafting branch that follows.

Also update the success/failure flow so that when the target slot is empty, ingredients are NOT consumed. This requires restructuring the success block slightly — move the `ConsumeIngredients` call to AFTER the enchanting target validation:

Find the `ConsumeIngredients` call (line 295) and move it inside both branches:
- For enchanting: consume AFTER validating the slot is occupied
- For normal crafting: consume where it currently is (before output)

The restructured logic should be:

```go
								if crafting.IsEnchantingRecipe(recipe) {
									// Enchanting: use the stored slot label to find the target
									targetSlot := user.Character.CraftingState.TargetSlot
									targetItem := user.Character.Equipment.GetSlotPointer(targetSlot)
									if targetItem == nil || targetItem.ItemId < 1 {
										user.SendText(`<ansi fg="red">The item is no longer equipped. The enchanting fails, but your materials are returned.</ansi>`)
									} else {
										user.Character.Items, user.Character.ComponentItems = crafting.ConsumeIngredients(user.Character.Items, user.Character.ComponentItems, recipe)
										eDef := enchantments.GetEnchantment(recipe.EnchantType)
										if eDef != nil {
											targetItem.EnchantType = recipe.EnchantType
											targetItem.EnchantTier = 0
											targetItem.EnchantUses = 0
											targetItem.ReservePool = eDef.ReservePool
											enchantments.ApplyTier(targetItem, eDef, 0)
										}
									}
								} else {
									user.Character.Items, user.Character.ComponentItems = crafting.ConsumeIngredients(user.Character.Items, user.Character.ComponentItems, recipe)
```

Make sure the original `ConsumeIngredients` call on line 295 is removed (it's now inside the branches).

- [ ] **Step 2: Remove the EquipmentSlot slice construction**

Delete the entire `eq := user.Character.Equipment` block and `equipSlots` slice (lines 299-324 in the original) and the `candidates := crafting.FindTargetItems(...)` call. These are no longer needed.

- [ ] **Step 3: Build and verify**

Run: `go build ./internal/hooks/`
Expected: Clean build.

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/NewRound_UserRoundTick.go
git commit -m "feat: enchanting completion uses stored slot label instead of item search"
```

---

### Task 4: Two-Handed Weapon Scaling in ApplyTier

**Files:**
- Modify: `internal/enchantments/enchantments.go`

- [ ] **Step 1: Add 2H multiplier to ApplyTier**

In `internal/enchantments/enchantments.go`, modify the `ApplyTier` function. After the line `tierDef := def.Tiers[tier]` (line 115), add 2H detection. Then in the effect application loop, multiply values by the 2H multiplier.

Replace the `ApplyTier` function body (lines 110-185) with:

```go
func ApplyTier(item *items.Item, def *EnchantmentDef, tier int) {
	if tier < 0 || tier >= len(def.Tiers) {
		return
	}

	tierDef := def.Tiers[tier]

	// Detect 2-handed weapons — double effects and reserve
	twoHandMult := 1
	baseSpec := items.GetItemSpec(item.ItemId)
	if baseSpec != nil && baseSpec.Hands >= 2 {
		twoHandMult = 2
	}

	// Ensure we have an override spec to work with
	var newSpec items.ItemSpec
	if item.Spec != nil {
		newSpec = *item.Spec
	} else {
		if baseSpec == nil {
			return
		}
		newSpec = *baseSpec
	}

	// Reset to base spec first to avoid stacking from previous tiers
	if baseSpec != nil {
		newSpec.Damage = baseSpec.Damage
		newSpec.DamageReduction = baseSpec.DamageReduction
		newSpec.DamageMultiplier = baseSpec.DamageMultiplier
		newSpec.PhysicalMitigation = baseSpec.PhysicalMitigation
		newSpec.MagicalMitigation = baseSpec.MagicalMitigation
		newSpec.ConvictionMitigation = baseSpec.ConvictionMitigation
		newSpec.StatMods = copyStatMods(baseSpec.StatMods)
	}

	// Apply tier effects (doubled for 2H weapons)
	for effectKey, effectVal := range tierDef.Effects {
		scaledVal := effectVal * twoHandMult
		switch effectKey {
		case "damage_bonus":
			if newSpec.Damage.BaseDamage > 0 {
				newSpec.Damage.BaseDamage += scaledVal
			} else {
				newSpec.Damage.BonusDamage += scaledVal
			}
		case "damage_multiplier_bonus":
			// Int value interpreted as hundredths: 10 = +0.10
			newSpec.DamageMultiplier += float64(scaledVal) / 100.0
		case "dr_bonus":
			newSpec.DamageReduction += scaledVal
		case "physical_mitigation_bonus":
			newSpec.PhysicalMitigation += scaledVal
		case "magical_mitigation_bonus":
			newSpec.MagicalMitigation += scaledVal
		case "conviction_mitigation_bonus":
			newSpec.ConvictionMitigation += scaledVal
		default:
			// Treat as a stat mod (e.g. "willpower_statmod", "return_damage", "lifesteal_pct")
			newSpec.StatMods.Add(effectKey, scaledVal)
		}
	}

	if newSpec.Damage.BaseDamage == 0 && newSpec.Damage.DiceRoll != "" {
		newSpec.Damage.FormatDiceRoll()
	}
	newSpec.AutoCalculateValue()

	item.Spec = &newSpec

	// Update adjective: remove any previous enchant adjective, add current
	cleanAdjectives := make([]string, 0, len(item.Adjectives))
	for _, adj := range item.Adjectives {
		if !isEnchantAdjective(adj, def) {
			cleanAdjectives = append(cleanAdjectives, adj)
		}
	}
	if tierDef.Adjective != "" {
		cleanAdjectives = append(cleanAdjectives, tierDef.Adjective)
	}
	item.Adjectives = cleanAdjectives
}
```

- [ ] **Step 2: Add 2H multiplier to GetTierReservePct**

Modify `GetTierReservePct` to accept an optional hands parameter:

```go
// GetTierReservePct returns the reserve_pct for the given enchantment at the
// given tier. If hands >= 2 (two-handed weapon), the reserve is doubled.
// Returns 0 if the enchantment or tier is not found.
func GetTierReservePct(enchantType string, tier int, hands ...int) float64 {
	def := GetEnchantment(enchantType)
	if def == nil {
		return 0
	}
	if tier < 0 || tier >= len(def.Tiers) {
		return 0
	}
	pct := def.Tiers[tier].ReservePct
	if len(hands) > 0 && hands[0] >= 2 {
		pct *= 2.0
	}
	return pct
}
```

- [ ] **Step 3: Update callers of GetTierReservePct to pass hands**

Search for all callers of `GetTierReservePct` and pass the item's `Hands` value:

Run: `grep -rn "GetTierReservePct" internal/`

For each caller, add the item's `Hands` field. The typical pattern is:

```go
// Before:
reservePct := enchantments.GetTierReservePct(item.EnchantType, item.EnchantTier)

// After:
reservePct := enchantments.GetTierReservePct(item.EnchantType, item.EnchantTier, item.GetSpec().Hands)
```

- [ ] **Step 4: Build and verify**

Run: `go build ./...`
Expected: Clean build.

- [ ] **Step 5: Commit**

```bash
git add internal/enchantments/enchantments.go
git commit -m "feat: 2H weapons get doubled enchantment effects and reserve costs"
```

---

### Task 5: Lifesteal Combat Integration

**Files:**
- Modify: `internal/hooks/NewRound_DoCombat_helpers.go`

- [ ] **Step 1: Add lifesteal check after damage dealt (Player vs Mob)**

In `internal/hooks/NewRound_DoCombat_helpers.go`, find the return damage block for Player vs Mob (around line 1120, the block starting with `// Return damage — fire elementals`). AFTER that block (after line 1141), add:

```go
	// Lifesteal — Hungering Touch enchantment heals attacker on hit
	if roundResult.Hit && roundResult.DamageToTarget > 0 {
		lifestealPct := user.Character.StatMod("lifesteal_pct")
		if lifestealPct > 0 {
			healAmt := int(float64(roundResult.DamageToTarget) * float64(lifestealPct) / 100.0)
			if healAmt > 0 {
				user.Character.Health += healAmt
				if user.Character.Health > user.Character.HealthMax.Value {
					user.Character.Health = user.Character.HealthMax.Value
				}
				healDesc := combat.GetHealDescription(healAmt, user.Character.HealthMax.Value)
				user.SendText(fmt.Sprintf(
					`<ansi fg="green">Your weapon feeds on the blow! (%s)</ansi>`,
					healDesc))
			}
		}
	}
```

- [ ] **Step 2: Add lifesteal check (Mob vs Player — mob attacking player)**

Find the return damage block for Mob vs Player (around line 1349). AFTER that block, add the same lifesteal logic but for the attacking mob:

```go
	// Lifesteal — mob enchantment heals attacker on hit
	if roundResult.Hit && roundResult.DamageToTarget > 0 {
		lifestealPct := mob.Character.StatMod("lifesteal_pct")
		if lifestealPct > 0 {
			healAmt := int(float64(roundResult.DamageToTarget) * float64(lifestealPct) / 100.0)
			if healAmt > 0 {
				mob.Character.Health += healAmt
				if mob.Character.Health > mob.Character.HealthMax.Value {
					mob.Character.Health = mob.Character.HealthMax.Value
				}
			}
		}
	}
```

- [ ] **Step 3: Add lifesteal check (Mob vs Mob)**

Find the return damage block for Mob vs Mob (around line 1534). AFTER that block, add:

```go
	// Lifesteal — attacking mob heals on hit
	if roundResult.Hit && roundResult.DamageToTarget > 0 {
		lifestealPct := atkMob.Character.StatMod("lifesteal_pct")
		if lifestealPct > 0 {
			healAmt := int(float64(roundResult.DamageToTarget) * float64(lifestealPct) / 100.0)
			if healAmt > 0 {
				atkMob.Character.Health += healAmt
				if atkMob.Character.Health > atkMob.Character.HealthMax.Value {
					atkMob.Character.Health = atkMob.Character.HealthMax.Value
				}
			}
		}
	}
```

- [ ] **Step 4: Build and verify**

Run: `go build ./internal/hooks/`
Expected: Clean build.

- [ ] **Step 5: Commit**

```bash
git add internal/hooks/NewRound_DoCombat_helpers.go
git commit -m "feat: add lifesteal_pct combat integration for Hungering Touch enchantment"
```

---

### Task 6: Update Existing Enchantment YAML Files

**Files:**
- Modify: all 10 files in `_datafiles/world/dogmud/enchantments/`

All existing enchantments are rebalanced to 5 tiers with the standard reserve
curve: 0.01 → 0.02 → 0.04 → 0.06 → 0.08.

- [ ] **Step 1: Update honed-edge.yaml**

Replace `_datafiles/world/dogmud/enchantments/honed-edge.yaml`:

```yaml
enchantid: honed-edge
name: "Honed Edge"
reserve_pool: health
target_type: weapon
tiers:
  - tier: 0
    reserve_pct: 0.01
    effects:
      damage_multiplier_bonus: 2
    adjective: keen
    description_suffix: "The edge holds an unusually sharp line."
    tier_up_message: ""
  - tier: 1
    reserve_pct: 0.02
    effects:
      damage_multiplier_bonus: 4
    adjective: razor-edged
    description_suffix: "The blade gleams with unnatural sharpness."
    tier_up_message: "The weapon hums faintly as the edge bites deeper."
  - tier: 2
    reserve_pct: 0.04
    effects:
      damage_multiplier_bonus: 6
    adjective: chitin-honed
    description_suffix: "Chitinous striations reinforce the blade's edge."
    tier_up_message: "The Chrysalis hardens the edge beyond mortal craft."
  - tier: 3
    reserve_pct: 0.06
    effects:
      damage_multiplier_bonus: 8
    adjective: living-edged
    description_suffix: "The edge resharpens itself, alive with Chrysalis energy."
    tier_up_message: "The blade grows eager — it knows where to cut."
  - tier: 4
    reserve_pct: 0.08
    effects:
      damage_multiplier_bonus: 10
    adjective: chrysalis-forged
    description_suffix: "The weapon's edge is Chrysalis-forged — perfect and eternal."
    tier_up_message: "The Chrysalis has perfected the blade. It will never dull."
```

- [ ] **Step 2: Update serpents-edge.yaml**

Replace `_datafiles/world/dogmud/enchantments/serpents-edge.yaml`:

```yaml
enchantid: serpents-edge
name: "Serpent's Edge"
reserve_pool: stamina
target_type: weapon
tiers:
  - tier: 0
    reserve_pct: 0.01
    effects:
      damage_multiplier_bonus: 5
    adjective: shimmering
    description_suffix: "A faint iridescent sheen plays across its surface."
    tier_up_message: ""
  - tier: 1
    reserve_pct: 0.02
    effects:
      damage_multiplier_bonus: 10
    adjective: scaled
    description_suffix: "Scales form along the edge, hard and iridescent."
    tier_up_message: "The weapon writhes subtly, as if alive."
  - tier: 2
    reserve_pct: 0.04
    effects:
      damage_multiplier_bonus: 15
    adjective: writhing
    description_suffix: "The surface undulates with serpent-like motion."
    tier_up_message: "The blade throbs, bonding with your flesh."
  - tier: 3
    reserve_pct: 0.06
    effects:
      damage_multiplier_bonus: 20
    adjective: serpent-fang
    description_suffix: "Fanged protrusions grow from the blade, glistening."
    tier_up_message: "A hunger pulses through the weapon — it calls for blood."
  - tier: 4
    reserve_pct: 0.08
    effects:
      damage_multiplier_bonus: 25
    adjective: living-chitin
    description_suffix: "The weapon hardens into living chitin, pulsing with malice."
    tier_up_message: "The Chrysalis consumes the blade — you are one now."
```

- [ ] **Step 3: Update hungering-touch.yaml**

Replace `_datafiles/world/dogmud/enchantments/hungering-touch.yaml`:

```yaml
enchantid: hungering-touch
name: "Hungering Touch"
reserve_pool: health
target_type: weapon
tiers:
  - tier: 0
    reserve_pct: 0.01
    effects:
      lifesteal_pct: 3
    adjective: veined
    description_suffix: "Dark veins pulse beneath the surface, hungry for blood."
    tier_up_message: ""
  - tier: 1
    reserve_pct: 0.02
    effects:
      lifesteal_pct: 5
    adjective: tendril-wrapped
    description_suffix: "Writhing tendrils extend from the weapon, seeking warmth."
    tier_up_message: "The weapon's hunger deepens — you taste metal and rot."
  - tier: 2
    reserve_pct: 0.04
    effects:
      lifesteal_pct: 8
    adjective: hungering
    description_suffix: "A pulsing mass coats the blade, breathing with you."
    tier_up_message: "The tendrils sink into your arm. You become the hunger."
  - tier: 3
    reserve_pct: 0.06
    effects:
      lifesteal_pct: 11
    adjective: parasitic
    description_suffix: "Parasitic growths fuse with your blood and the blade."
    tier_up_message: "Your essence feeds the Chrysalis. Resistance is futile."
  - tier: 4
    reserve_pct: 0.08
    effects:
      lifesteal_pct: 15
    adjective: devouring
    description_suffix: "The weapon is a devouring maw of chitin and hunger."
    tier_up_message: "The Chrysalis has devoured you entirely. You are eternal hunger."
```

- [ ] **Step 4: Update carapace-ward.yaml**

Replace `_datafiles/world/dogmud/enchantments/carapace-ward.yaml`:

```yaml
enchantid: carapace-ward
name: "Carapace Ward"
reserve_pool: health
target_type: body
tiers:
  - tier: 0
    reserve_pct: 0.01
    effects:
      physical_mitigation_bonus: 1
    adjective: shell-traced
    description_suffix: "Faint shell-like patterns trace across the surface."
    tier_up_message: ""
  - tier: 1
    reserve_pct: 0.02
    effects:
      physical_mitigation_bonus: 2
    adjective: plated
    description_suffix: "Chitinous plates emerge, layering natural armor."
    tier_up_message: "The armor thickens — you feel the Chrysalis hardening."
  - tier: 2
    reserve_pct: 0.04
    effects:
      physical_mitigation_bonus: 3
    adjective: carapaced
    description_suffix: "A full carapace forms, gleaming with insectile strength."
    tier_up_message: "The Chrysalis encases you in living armor."
  - tier: 3
    reserve_pct: 0.06
    effects:
      physical_mitigation_bonus: 4
    adjective: exoskeletal
    description_suffix: "The armor is now exoskeleton — grown, not forged."
    tier_up_message: "Your skin and armor become one. The Chrysalis protects."
  - tier: 4
    reserve_pct: 0.08
    effects:
      physical_mitigation_bonus: 5
    adjective: chrysalis-shelled
    description_suffix: "A perfect Chrysalis shell — ancient and unyielding."
    tier_up_message: "The Chrysalis has remade your armor eternal."
```

- [ ] **Step 5: Update sporeweave.yaml**

Replace `_datafiles/world/dogmud/enchantments/sporeweave.yaml`:

```yaml
enchantid: sporeweave
name: "Sporeweave"
reserve_pool: stamina
target_type: body
tiers:
  - tier: 0
    reserve_pct: 0.01
    effects:
      dexterity_statmod: 1
    adjective: filament-laced
    description_suffix: "Fine filaments weave through the fabric."
    tier_up_message: ""
  - tier: 1
    reserve_pct: 0.02
    effects:
      dexterity_statmod: 2
    adjective: spore-dusted
    description_suffix: "Spores settle across the fabric, shimmering faintly."
    tier_up_message: "The spores drift into your lungs — you breathe the Chrysalis."
  - tier: 2
    reserve_pct: 0.04
    effects:
      dexterity_statmod: 3
    adjective: mycelial
    description_suffix: "Mycelial networks spread across the armor like roots."
    tier_up_message: "The fungal threads bind to your muscles — you move as one."
  - tier: 3
    reserve_pct: 0.06
    effects:
      dexterity_statmod: 4
    adjective: fungal-woven
    description_suffix: "The armor is woven entirely of fungal threads, growing."
    tier_up_message: "The Chrysalis grows through you — agility beyond mortal limits."
  - tier: 4
    reserve_pct: 0.08
    effects:
      dexterity_statmod: 5
    adjective: spore-blooming
    description_suffix: "Blooming spore clusters erupt, releasing living clouds."
    tier_up_message: "The Chrysalis has made you swift as spores on the wind."
```

- [ ] **Step 6: Update chrysalis-sight.yaml**

Replace `_datafiles/world/dogmud/enchantments/chrysalis-sight.yaml`:

```yaml
enchantid: chrysalis-sight
name: "Chrysalis Sight"
reserve_pool: conviction
target_type: head
tiers:
  - tier: 0
    reserve_pct: 0.01
    effects:
      perception_statmod: 1
    adjective: clouded
    description_suffix: "The visor clouds with prismatic light."
    tier_up_message: ""
  - tier: 1
    reserve_pct: 0.02
    effects:
      perception_statmod: 2
    adjective: multi-faceted
    description_suffix: "The surface becomes faceted like an insect's eye."
    tier_up_message: "You see through a thousand lenses — you see everything."
  - tier: 2
    reserve_pct: 0.04
    effects:
      perception_statmod: 3
    adjective: crystalline-eyed
    description_suffix: "Crystalline structures form around the eyes, gleaming."
    tier_up_message: "The Chrysalis grants vision beyond mortal comprehension."
  - tier: 3
    reserve_pct: 0.06
    effects:
      perception_statmod: 4
    adjective: all-seeing
    description_suffix: "Multiple eyes watch from the helm, unblinking."
    tier_up_message: "You see the world's hidden geometry. You are its eye."
  - tier: 4
    reserve_pct: 0.08
    effects:
      perception_statmod: 5
    adjective: eye-clustered
    description_suffix: "Eyes cluster across the helm — ancient, alien, all-knowing."
    tier_up_message: "The Chrysalis sees through you forever. You are its vision."
```

- [ ] **Step 7: Update mindweave.yaml**

Replace `_datafiles/world/dogmud/enchantments/mindweave.yaml`:

```yaml
enchantid: mindweave
name: "Mindweave"
reserve_pool: conviction
target_type: head
tiers:
  - tier: 0
    reserve_pct: 0.01
    effects:
      willpower_statmod: 1
    adjective: pulsing
    description_suffix: "The helm pulses in rhythm with your thoughts."
    tier_up_message: ""
  - tier: 1
    reserve_pct: 0.02
    effects:
      willpower_statmod: 2
    adjective: neural-threaded
    description_suffix: "Neural threads extend through your mind, pulsing gently."
    tier_up_message: "Your thoughts become one with the Chrysalis's whisper."
  - tier: 2
    reserve_pct: 0.04
    effects:
      willpower_statmod: 3
    adjective: mind-linked
    description_suffix: "Your mind is linked to something vast and ancient."
    tier_up_message: "The Chrysalis dwells in your consciousness now, eternal."
  - tier: 3
    reserve_pct: 0.06
    effects:
      willpower_statmod: 4
    adjective: crown-tendriled
    description_suffix: "Tendrils of force crown your thoughts — invasive, transforming."
    tier_up_message: "You are no longer alone in your mind. The Chrysalis is you."
  - tier: 4
    reserve_pct: 0.08
    effects:
      willpower_statmod: 5
    adjective: neural-web
    description_suffix: "A neural web of pure will encompasses your consciousness."
    tier_up_message: "The Chrysalis has consumed your mind. You are transcendent."
```

- [ ] **Step 8: Update ironblood.yaml**

Replace `_datafiles/world/dogmud/enchantments/ironblood.yaml`:

```yaml
enchantid: ironblood
name: "Ironblood"
reserve_pool: conviction
target_type: neck
tiers:
  - tier: 0
    reserve_pct: 0.01
    effects:
      conviction_mitigation_bonus: 1
      willpower_statmod: 1
    adjective: warm-veined
    description_suffix: "Warm veins pulse beneath the surface."
    tier_up_message: ""
  - tier: 1
    reserve_pct: 0.02
    effects:
      conviction_mitigation_bonus: 2
      willpower_statmod: 1
    adjective: ore-blooded
    description_suffix: "Your blood becomes molten ore — veins glow with heat."
    tier_up_message: "The Chrysalis infuses your essence with metal and fire."
  - tier: 2
    reserve_pct: 0.04
    effects:
      conviction_mitigation_bonus: 3
      willpower_statmod: 2
    adjective: iron-pulsed
    description_suffix: "Iron courses through your veins, hardening your resolve."
    tier_up_message: "You are no longer flesh — you are living iron."
  - tier: 3
    reserve_pct: 0.06
    effects:
      conviction_mitigation_bonus: 4
      willpower_statmod: 2
    adjective: living-ore
    description_suffix: "Your essence becomes living ore, pulsing with power."
    tier_up_message: "The Chrysalis has remade you — eternal and unyielding."
  - tier: 4
    reserve_pct: 0.08
    effects:
      conviction_mitigation_bonus: 5
      willpower_statmod: 3
    adjective: ironblood
    description_suffix: "You are Ironblood — living veins of metal and will."
    tier_up_message: "The Chrysalis flows through your core. You are iron eternal."
```

- [ ] **Step 9: Update predators-instinct.yaml**

Replace `_datafiles/world/dogmud/enchantments/predators-instinct.yaml`:

```yaml
enchantid: predators-instinct
name: "Predator's Instinct"
reserve_pool: stamina
target_type: ring
tiers:
  - tier: 0
    reserve_pct: 0.01
    effects:
      perception_statmod: 1
    adjective: warm
    description_suffix: "The ring pulses with warmth, alive in your palm."
    tier_up_message: ""
  - tier: 1
    reserve_pct: 0.02
    effects:
      perception_statmod: 2
    adjective: pulsing
    description_suffix: "The ring's pulse quickens — heartbeat of a hunter."
    tier_up_message: "The ring feeds on your instincts — you smell everything."
  - tier: 2
    reserve_pct: 0.04
    effects:
      perception_statmod: 3
    adjective: ember-eyed
    description_suffix: "Ember-like eyes form on the ring's surface, watching."
    tier_up_message: "Your eyes see prey. The Chrysalis has awakened the predator."
  - tier: 3
    reserve_pct: 0.06
    effects:
      perception_statmod: 4
    adjective: predatory
    description_suffix: "The ring becomes a predatory thing, thirsting for the hunt."
    tier_up_message: "You are one with the Chrysalis — hunger and instinct made flesh."
  - tier: 4
    reserve_pct: 0.08
    effects:
      perception_statmod: 5
    adjective: hunter-souled
    description_suffix: "A hunter's soul dwells in the ring — ancient and eternal."
    tier_up_message: "The Chrysalis has made you the perfect predator forevermore."
```

- [ ] **Step 10: Update chrysalis-stride.yaml**

Replace `_datafiles/world/dogmud/enchantments/chrysalis-stride.yaml`:

```yaml
enchantid: chrysalis-stride
name: "Chrysalis Stride"
reserve_pool: stamina
target_type: legs
tiers:
  - tier: 0
    reserve_pct: 0.01
    effects:
      dexterity_statmod: 1
    adjective: grooved
    description_suffix: "Grooves form along the leg armor like exoskeletons."
    tier_up_message: ""
  - tier: 1
    reserve_pct: 0.02
    effects:
      dexterity_statmod: 2
    adjective: jointed
    description_suffix: "Articulated joints emerge, bending in new directions."
    tier_up_message: "Your legs twist and reform — the Chrysalis teaches you to move."
  - tier: 2
    reserve_pct: 0.04
    effects:
      dexterity_statmod: 3
    adjective: chitin-legged
    description_suffix: "Chitin plates replace your shins, hard and gleaming."
    tier_up_message: "You leap with insectile grace — you are not fully human."
  - tier: 3
    reserve_pct: 0.06
    effects:
      dexterity_statmod: 4
    adjective: insectile
    description_suffix: "Your legs are segmented chitin, pulsing with life."
    tier_up_message: "The Chrysalis remolds your limbs into something faster."
  - tier: 4
    reserve_pct: 0.08
    effects:
      dexterity_statmod: 5
    adjective: chrysalis-limbed
    description_suffix: "Your legs are living chrysalis — capable of impossible speed."
    tier_up_message: "You run on wings of chitin. The metamorphosis is complete."
```

- [ ] **Step 11: Commit all enchantment updates**

```bash
git add _datafiles/world/dogmud/enchantments/
git commit -m "feat: rebalance all enchantments to 5 tiers with standard reserve curve"
```

---

### Task 7: Create New Enchantment YAML Files

**Files:**
- Create: 8 new files in `_datafiles/world/dogmud/enchantments/`

- [ ] **Step 1: Create chitin-brace.yaml**

Create `_datafiles/world/dogmud/enchantments/chitin-brace.yaml`:

```yaml
enchantid: chitin-brace
name: "Chitin Brace"
reserve_pool: health
target_type: wrist
tiers:
  - tier: 0
    reserve_pct: 0.01
    effects:
      physical_mitigation_bonus: 1
    adjective: ridged
    description_suffix: "Hard ridges form along the bracer's surface."
    tier_up_message: ""
  - tier: 1
    reserve_pct: 0.02
    effects:
      physical_mitigation_bonus: 2
    adjective: chitin-ridged
    description_suffix: "Chitin ridges harden into natural armor plates."
    tier_up_message: "The bracer fuses with your wrist — chitin and bone."
  - tier: 2
    reserve_pct: 0.04
    effects:
      physical_mitigation_bonus: 3
    adjective: shell-braced
    description_suffix: "A shell of living chitin encases the bracer entirely."
    tier_up_message: "The Chrysalis hardens your guard beyond mortal craft."
  - tier: 3
    reserve_pct: 0.06
    effects:
      physical_mitigation_bonus: 4
    adjective: exo-braced
    description_suffix: "The bracer becomes exoskeleton — grown, not forged."
    tier_up_message: "Your wrist is Chrysalis-armored. Blades glance away."
  - tier: 4
    reserve_pct: 0.08
    effects:
      physical_mitigation_bonus: 5
    adjective: chrysalis-braced
    description_suffix: "Living chitin flows from wrist to forearm, impenetrable."
    tier_up_message: "The Chrysalis has perfected your defense. Nothing gets through."
```

- [ ] **Step 2: Create rootbind.yaml**

Create `_datafiles/world/dogmud/enchantments/rootbind.yaml`:

```yaml
enchantid: rootbind
name: "Rootbind"
reserve_pool: health
target_type: belt
tiers:
  - tier: 0
    reserve_pct: 0.01
    effects:
      vitality_statmod: 1
    adjective: root-threaded
    description_suffix: "Fine root-like threads weave through the leather."
    tier_up_message: ""
  - tier: 1
    reserve_pct: 0.02
    effects:
      vitality_statmod: 2
    adjective: bark-bound
    description_suffix: "Bark-like growths reinforce the belt with living wood."
    tier_up_message: "The roots dig deeper — you feel the earth's endurance."
  - tier: 2
    reserve_pct: 0.04
    effects:
      vitality_statmod: 3
    adjective: rootbound
    description_suffix: "A tangle of living roots anchors you to the ground."
    tier_up_message: "The Chrysalis channels the earth's vitality through you."
  - tier: 3
    reserve_pct: 0.06
    effects:
      vitality_statmod: 4
    adjective: deep-rooted
    description_suffix: "Ancient roots pulse through the belt, drawing on deep earth."
    tier_up_message: "You are rooted to something primal. Your endurance is endless."
  - tier: 4
    reserve_pct: 0.08
    effects:
      vitality_statmod: 5
    adjective: earthbound
    description_suffix: "The belt is a living root system — you are the earth itself."
    tier_up_message: "The Chrysalis has made you one with the deep roots. Unbreakable."
```

- [ ] **Step 3: Create rootwalker.yaml**

Create `_datafiles/world/dogmud/enchantments/rootwalker.yaml`:

```yaml
enchantid: rootwalker
name: "Rootwalker"
reserve_pool: health
target_type: feet
tiers:
  - tier: 0
    reserve_pct: 0.01
    effects:
      physical_mitigation_bonus: 1
    adjective: root-soled
    description_suffix: "Root-like soles grip the earth with each step."
    tier_up_message: ""
  - tier: 1
    reserve_pct: 0.02
    effects:
      physical_mitigation_bonus: 2
    adjective: bark-footed
    description_suffix: "Bark-like growths form across the boots, grounding you."
    tier_up_message: "The earth steadies your footing — impacts soften."
  - tier: 2
    reserve_pct: 0.04
    effects:
      physical_mitigation_bonus: 3
    adjective: earthwalker
    description_suffix: "Your boots have become part of the earth, absorbing force."
    tier_up_message: "The Chrysalis roots you. Blows that should stagger do not."
  - tier: 3
    reserve_pct: 0.06
    effects:
      physical_mitigation_bonus: 4
    adjective: stone-treading
    description_suffix: "Your steps leave root-prints — the ground itself protects you."
    tier_up_message: "You walk on living stone. The earth rises to shield you."
  - tier: 4
    reserve_pct: 0.08
    effects:
      physical_mitigation_bonus: 5
    adjective: rootwalker
    description_suffix: "Your feet are the roots of the world — unshakable, eternal."
    tier_up_message: "The Chrysalis has made you the rootwalker. Immovable."
```

- [ ] **Step 4: Create chrysalis-bond.yaml**

Create `_datafiles/world/dogmud/enchantments/chrysalis-bond.yaml`:

```yaml
enchantid: chrysalis-bond
name: "Chrysalis Bond"
reserve_pool: conviction
target_type: ring
tiers:
  - tier: 0
    reserve_pct: 0.01
    effects:
      conviction_mitigation_bonus: 1
    adjective: bonded
    description_suffix: "The ring hums with a faint psychic resonance."
    tier_up_message: ""
  - tier: 1
    reserve_pct: 0.02
    effects:
      conviction_mitigation_bonus: 2
    adjective: mind-bonded
    description_suffix: "A psychic tether binds the ring to your consciousness."
    tier_up_message: "The ring shields your thoughts — mental attacks glance away."
  - tier: 2
    reserve_pct: 0.04
    effects:
      conviction_mitigation_bonus: 3
    adjective: soul-linked
    description_suffix: "The ring pulses in time with your convictions, warding doubt."
    tier_up_message: "The Chrysalis bonds your will. Your resolve becomes armor."
  - tier: 3
    reserve_pct: 0.06
    effects:
      conviction_mitigation_bonus: 4
    adjective: will-forged
    description_suffix: "Pure will radiates from the ring, deflecting psychic assault."
    tier_up_message: "Your mind is a fortress. The Chrysalis guards the gates."
  - tier: 4
    reserve_pct: 0.08
    effects:
      conviction_mitigation_bonus: 5
    adjective: chrysalis-bonded
    description_suffix: "The ring is a shard of the Chrysalis itself — your mind is unbreakable."
    tier_up_message: "The Chrysalis has sealed your conviction. Nothing can break you."
```

- [ ] **Step 5: Create spore-mantle.yaml**

Create `_datafiles/world/dogmud/enchantments/spore-mantle.yaml`:

```yaml
enchantid: spore-mantle
name: "Spore Mantle"
reserve_pool: stamina
target_type: shoulders
tiers:
  - tier: 0
    reserve_pct: 0.01
    effects:
      magical_mitigation_bonus: 1
    adjective: spore-dusted
    description_suffix: "A fine dusting of luminous spores clings to the pauldrons."
    tier_up_message: ""
  - tier: 1
    reserve_pct: 0.02
    effects:
      magical_mitigation_bonus: 2
    adjective: spore-veiled
    description_suffix: "A veil of drifting spores surrounds the shoulders."
    tier_up_message: "The spores absorb incoming energy — magic dissipates on contact."
  - tier: 2
    reserve_pct: 0.04
    effects:
      magical_mitigation_bonus: 3
    adjective: fungal-mantled
    description_suffix: "A living fungal mantle grows across the shoulders."
    tier_up_message: "The Chrysalis weaves a ward of spores. Magic unravels against it."
  - tier: 3
    reserve_pct: 0.06
    effects:
      magical_mitigation_bonus: 4
    adjective: mycelial-cloaked
    description_suffix: "Mycelial networks form a shimmering anti-magic barrier."
    tier_up_message: "Spells break apart before they reach you. The mantle devours them."
  - tier: 4
    reserve_pct: 0.08
    effects:
      magical_mitigation_bonus: 5
    adjective: spore-crowned
    description_suffix: "A crown of living spores blooms across your shoulders, consuming magic."
    tier_up_message: "The Chrysalis has made you anathema to magic. Spells die on your skin."
```

- [ ] **Step 6: Create thornguard.yaml**

Create `_datafiles/world/dogmud/enchantments/thornguard.yaml`:

```yaml
enchantid: thornguard
name: "Thornguard"
reserve_pool: health
target_type: offhand
tiers:
  - tier: 0
    reserve_pct: 0.01
    effects:
      return_damage: 5
    adjective: barbed
    description_suffix: "Sharp barbs protrude from the shield's face."
    tier_up_message: ""
  - tier: 1
    reserve_pct: 0.02
    effects:
      return_damage: 10
    adjective: thorn-studded
    description_suffix: "Thorny growths stud the shield, punishing attackers."
    tier_up_message: "The thorns grow hungry — they bite back at every blow."
  - tier: 2
    reserve_pct: 0.04
    effects:
      return_damage: 15
    adjective: spine-ridged
    description_suffix: "Bony spines ridge the shield, lacerating on impact."
    tier_up_message: "The Chrysalis arms your defense. Every block draws blood."
  - tier: 3
    reserve_pct: 0.06
    effects:
      return_damage: 20
    adjective: chitin-thorned
    description_suffix: "Chitin thorns erupt across the shield, dripping with malice."
    tier_up_message: "Attackers recoil in pain. The shield fights back."
  - tier: 4
    reserve_pct: 0.08
    effects:
      return_damage: 25
    adjective: thornguard
    description_suffix: "The shield is a wall of living thorns — to strike it is agony."
    tier_up_message: "The Chrysalis has made your shield a weapon. Defense is attack."
```

- [ ] **Step 7: Create venomgrip.yaml**

Create `_datafiles/world/dogmud/enchantments/venomgrip.yaml`:

```yaml
enchantid: venomgrip
name: "Venomgrip"
reserve_pool: stamina
target_type: gloves
tiers:
  - tier: 0
    reserve_pct: 0.01
    effects:
      strength_statmod: 1
    adjective: slick
    description_suffix: "A slick, oily sheen coats the gloves."
    tier_up_message: ""
  - tier: 1
    reserve_pct: 0.02
    effects:
      strength_statmod: 2
    adjective: venom-slick
    description_suffix: "Venom seeps from the gloves, strengthening your grip."
    tier_up_message: "The venom hardens your hands — your grip becomes crushing."
  - tier: 2
    reserve_pct: 0.04
    effects:
      strength_statmod: 3
    adjective: toxin-laced
    description_suffix: "Toxin-laced chitin forms across the knuckles."
    tier_up_message: "The Chrysalis poisons your strength. Every blow bites deeper."
  - tier: 3
    reserve_pct: 0.06
    effects:
      strength_statmod: 4
    adjective: chitin-fisted
    description_suffix: "Your hands are encased in venomous chitin, powerful and deadly."
    tier_up_message: "Your fists are weapons of the Chrysalis. Bone and venom."
  - tier: 4
    reserve_pct: 0.08
    effects:
      strength_statmod: 5
    adjective: venomgrip
    description_suffix: "Living venom flows through your grip — nothing escapes your hands."
    tier_up_message: "The Chrysalis has perfected your strength. You crush without effort."
```

- [ ] **Step 8: Create shadowweave.yaml**

Create `_datafiles/world/dogmud/enchantments/shadowweave.yaml`:

```yaml
enchantid: shadowweave
name: "Shadowweave"
reserve_pool: conviction
target_type: back
tiers:
  - tier: 0
    reserve_pct: 0.01
    effects:
      magical_mitigation_bonus: 1
    adjective: shadow-touched
    description_suffix: "Shadows cling to the cloak, absorbing stray light."
    tier_up_message: ""
  - tier: 1
    reserve_pct: 0.02
    effects:
      magical_mitigation_bonus: 2
    adjective: shadow-woven
    description_suffix: "The cloak is woven from living shadow, drinking in magic."
    tier_up_message: "The shadows deepen — magic slides off you like water."
  - tier: 2
    reserve_pct: 0.04
    effects:
      magical_mitigation_bonus: 3
    adjective: void-touched
    description_suffix: "The cloak touches the void — magic unravels in its presence."
    tier_up_message: "The Chrysalis wraps you in darkness. Spells cannot find you."
  - tier: 3
    reserve_pct: 0.06
    effects:
      magical_mitigation_bonus: 4
    adjective: null-cloaked
    description_suffix: "A null zone surrounds the cloak, consuming magical energy."
    tier_up_message: "You exist in a pocket of anti-magic. The Chrysalis devours spells."
  - tier: 4
    reserve_pct: 0.08
    effects:
      magical_mitigation_bonus: 5
    adjective: shadowweave
    description_suffix: "The cloak IS shadow — magic dies in its embrace."
    tier_up_message: "The Chrysalis has woven you into shadow. Magic cannot touch you."
```

- [ ] **Step 9: Commit all new enchantments**

```bash
git add _datafiles/world/dogmud/enchantments/
git commit -m "feat: add 8 new enchantments for uncovered equipment slots"
```

---

### Task 8: Update Existing Recipe YAML Files

**Files:**
- Modify: all 10 files in `_datafiles/world/dogmud/recipes/enchanting/`

Recipe names use hyphens to match recipe IDs. The `skill_minimum` values use
the real skill scale (0-50). Recipe names drop the apostrophes for cleaner
parsing (e.g. "Serpents-Edge" not "Serpent's-Edge").

- [ ] **Step 1: Update honed-edge.yaml recipe**

Replace `_datafiles/world/dogmud/recipes/enchanting/honed-edge.yaml`:

```yaml
id: honed-edge
name: "Honed-Edge"
skill: enchanting
skill_minimum: 0
station: enchanting_circle
time_rounds: 3
target_type: weapon
enchant_type: honed-edge
ingredients:
  - item_tag: binding-paste
    quantity: 1
  - item_tag: healers-root
    quantity: 1
output:
  item_id: 0
  quantity: 0
success_message: "The paste bonds to the blade and hardens. When you test the edge, it cuts cleaner than before."
failure_message: "The paste dries and flakes away without taking hold. The materials are wasted, but the weapon is unharmed."
```

(Unchanged from current — this recipe is fine as-is.)

- [ ] **Step 2: Update serpents-edge.yaml recipe**

Replace `_datafiles/world/dogmud/recipes/enchanting/serpents-edge.yaml`:

```yaml
id: serpents-edge
name: "Serpents-Edge"
skill: enchanting
skill_minimum: 8
station: enchanting_circle
time_rounds: 4
target_type: weapon
enchant_type: serpents-edge
ingredients:
  - item_tag: chrysalis-setting
    quantity: 1
  - item_tag: mutation-catalyst
    quantity: 1
  - item_tag: binding-paste
    quantity: 1
output:
  item_id: 0
  quantity: 0
success_message: "Chrysalis energy flows through the setting and into the blade. The metal shimmers with an iridescent sheen."
failure_message: "The catalyst rejects the binding. The materials crumble to dust, but the weapon is unharmed."
```

(Name changed: removed apostrophe from "Serpent's-Edge" → "Serpents-Edge" for cleaner parsing.)

- [ ] **Step 3: Update hungering-touch.yaml recipe**

Replace `_datafiles/world/dogmud/recipes/enchanting/hungering-touch.yaml`:

```yaml
id: hungering-touch
name: "Hungering-Touch"
skill: enchanting
skill_minimum: 30
station: enchanting_circle
time_rounds: 6
target_type: weapon
enchant_type: hungering-touch
ingredients:
  - item_tag: chrysalis-setting
    quantity: 2
  - item_tag: mutation-catalyst
    quantity: 2
  - item_tag: binding-paste
    quantity: 1
output:
  item_id: 0
  quantity: 0
success_message: "Twin catalysts writhe within the settings, fusing into the blade's edge. The weapon trembles with an insatiable hunger."
failure_message: "The catalyst rejects the binding. The materials crumble to dust, but the weapon is unharmed."
```

(Unchanged from current.)

- [ ] **Step 4: Update carapace-ward.yaml recipe**

Replace `_datafiles/world/dogmud/recipes/enchanting/carapace-ward.yaml`:

```yaml
id: carapace-ward
name: "Carapace-Ward"
skill: enchanting
skill_minimum: 3
station: enchanting_circle
time_rounds: 4
target_type: body
enchant_type: carapace-ward
ingredients:
  - item_tag: chrysalis-setting
    quantity: 1
  - item_tag: mutation-catalyst
    quantity: 1
  - item_tag: binding-paste
    quantity: 1
output:
  item_id: 0
  quantity: 0
success_message: "Chitinous patterns spiral across the fabric, hardening it with Chrysalis resilience. The armor gleams with natural protection."
failure_message: "The catalyst rejects the binding. The materials crumble to dust, but the armor is unharmed."
```

(Unchanged from current.)

- [ ] **Step 5: Update sporeweave.yaml recipe**

Replace `_datafiles/world/dogmud/recipes/enchanting/sporeweave.yaml`:

```yaml
id: sporeweave
name: "Sporeweave"
skill: enchanting
skill_minimum: 10
station: enchanting_circle
time_rounds: 5
target_type: body
enchant_type: sporeweave
ingredients:
  - item_tag: chrysalis-setting
    quantity: 1
  - item_tag: mutation-catalyst
    quantity: 1
  - item_tag: binding-paste
    quantity: 2
output:
  item_id: 0
  quantity: 0
success_message: "Spores bloom and weave through the fibers, creating a web of mutation that pulses with dormant potential. The armor grows organic and supple."
failure_message: "The catalyst rejects the binding. The materials crumble to dust, but the armor is unharmed."
```

(Unchanged from current.)

- [ ] **Step 6: Update chrysalis-sight.yaml recipe**

Replace `_datafiles/world/dogmud/recipes/enchanting/chrysalis-sight.yaml`:

```yaml
id: chrysalis-sight
name: "Chrysalis-Sight"
skill: enchanting
skill_minimum: 6
station: enchanting_circle
time_rounds: 4
target_type: head
enchant_type: chrysalis-sight
ingredients:
  - item_tag: chrysalis-setting
    quantity: 1
  - item_tag: mutation-catalyst
    quantity: 1
  - item_tag: binding-paste
    quantity: 1
output:
  item_id: 0
  quantity: 0
success_message: "The setting fuses with the helm, its surface rippling with multifaceted awareness. Your perception sharpens in all directions."
failure_message: "The catalyst rejects the binding. The materials crumble to dust, but the helm is unharmed."
```

(Unchanged from current.)

- [ ] **Step 7: Update mindweave.yaml recipe**

Replace `_datafiles/world/dogmud/recipes/enchanting/mindweave.yaml`:

```yaml
id: mindweave
name: "Mindweave"
skill: enchanting
skill_minimum: 35
station: enchanting_circle
time_rounds: 7
target_type: head
enchant_type: mindweave
ingredients:
  - item_tag: chrysalis-setting
    quantity: 2
  - item_tag: mutation-catalyst
    quantity: 2
  - item_tag: binding-paste
    quantity: 2
output:
  item_id: 0
  quantity: 0
success_message: "All elements converge in a cascade of Chrysalis insight, weaving through the helm's crown. Your thoughts sharpen to crystalline clarity."
failure_message: "The catalyst rejects the binding. The materials crumble to dust, but the helm is unharmed."
```

(Unchanged from current.)

- [ ] **Step 8: Update ironblood.yaml recipe**

Replace `_datafiles/world/dogmud/recipes/enchanting/ironblood.yaml`:

```yaml
id: ironblood
name: "Ironblood"
skill: enchanting
skill_minimum: 20
station: enchanting_circle
time_rounds: 5
target_type: neck
enchant_type: ironblood
ingredients:
  - item_tag: chrysalis-setting
    quantity: 2
  - item_tag: mutation-catalyst
    quantity: 1
  - item_tag: binding-paste
    quantity: 1
output:
  item_id: 0
  quantity: 0
success_message: "Twin settings spiral with molten resolve, infusing your essence with Chrysalis fortitude. Your blood runs thick with mutation's promise."
failure_message: "The catalyst rejects the binding. The materials crumble to dust, but the amulet is unharmed."
```

(Unchanged from current.)

- [ ] **Step 9: Update predators-instinct.yaml recipe**

Replace `_datafiles/world/dogmud/recipes/enchanting/predators-instinct.yaml`:

```yaml
id: predators-instinct
name: "Predators-Instinct"
skill: enchanting
skill_minimum: 25
station: enchanting_circle
time_rounds: 6
target_type: ring
enchant_type: predators-instinct
ingredients:
  - item_tag: chrysalis-setting
    quantity: 1
  - item_tag: mutation-catalyst
    quantity: 2
  - item_tag: binding-paste
    quantity: 1
output:
  item_id: 0
  quantity: 0
success_message: "The catalysts merge within the setting, birthing a hunger that sings through your nerves. Your reflexes sharpen to predatory perfection."
failure_message: "The catalyst rejects the binding. The materials crumble to dust, but the ring is unharmed."
```

(Name changed: removed apostrophe from "Predator's-Instinct" → "Predators-Instinct".)

- [ ] **Step 10: Update chrysalis-stride.yaml recipe**

Replace `_datafiles/world/dogmud/recipes/enchanting/chrysalis-stride.yaml`:

```yaml
id: chrysalis-stride
name: "Chrysalis-Stride"
skill: enchanting
skill_minimum: 15
station: enchanting_circle
time_rounds: 5
target_type: legs
enchant_type: chrysalis-stride
ingredients:
  - item_tag: chrysalis-setting
    quantity: 1
  - item_tag: mutation-catalyst
    quantity: 1
  - item_tag: binding-paste
    quantity: 2
output:
  item_id: 0
  quantity: 0
success_message: "Metamorphic energy kindles in the leg coverings, granting them the swift grace of Chrysalis emergence. Your stride quickens with predatory intent."
failure_message: "The catalyst rejects the binding. The materials crumble to dust, but the legs are unharmed."
```

(Unchanged from current.)

- [ ] **Step 11: Commit all recipe updates**

```bash
git add _datafiles/world/dogmud/recipes/enchanting/
git commit -m "feat: update existing enchanting recipes (name cleanup, consistency)"
```

---

### Task 9: Create New Recipe YAML Files

**Files:**
- Create: 8 new files in `_datafiles/world/dogmud/recipes/enchanting/`

- [ ] **Step 1: Create chitin-brace.yaml recipe**

Create `_datafiles/world/dogmud/recipes/enchanting/chitin-brace.yaml`:

```yaml
id: chitin-brace
name: "Chitin-Brace"
skill: enchanting
skill_minimum: 2
station: enchanting_circle
time_rounds: 3
target_type: wrist
enchant_type: chitin-brace
ingredients:
  - item_tag: binding-paste
    quantity: 1
  - item_tag: healers-root
    quantity: 2
output:
  item_id: 0
  quantity: 0
success_message: "The paste hardens into chitinous ridges along the bracer. You flex your wrist — it feels armored yet supple."
failure_message: "The paste crumbles away without bonding. The materials are wasted, but the bracer is unharmed."
```

- [ ] **Step 2: Create rootbind.yaml recipe**

Create `_datafiles/world/dogmud/recipes/enchanting/rootbind.yaml`:

```yaml
id: rootbind
name: "Rootbind"
skill: enchanting
skill_minimum: 5
station: enchanting_circle
time_rounds: 3
target_type: belt
enchant_type: rootbind
ingredients:
  - item_tag: binding-paste
    quantity: 2
  - item_tag: healers-root
    quantity: 1
output:
  item_id: 0
  quantity: 0
success_message: "Root-like tendrils burrow into the leather, anchoring it with living wood. You feel the earth's endurance flow through you."
failure_message: "The roots wither before bonding. The materials are wasted, but the belt is unharmed."
```

- [ ] **Step 3: Create rootwalker.yaml recipe**

Create `_datafiles/world/dogmud/recipes/enchanting/rootwalker.yaml`:

```yaml
id: rootwalker
name: "Rootwalker"
skill: enchanting
skill_minimum: 7
station: enchanting_circle
time_rounds: 3
target_type: feet
enchant_type: rootwalker
ingredients:
  - item_tag: chrysalis-setting
    quantity: 1
  - item_tag: binding-paste
    quantity: 1
output:
  item_id: 0
  quantity: 0
success_message: "Root-like soles form beneath the boots, gripping the ground with living purpose. Each step feels steadier, more grounded."
failure_message: "The setting rejects the boot leather. The materials crumble to dust, but the boots are unharmed."
```

- [ ] **Step 4: Create chrysalis-bond.yaml recipe**

Create `_datafiles/world/dogmud/recipes/enchanting/chrysalis-bond.yaml`:

```yaml
id: chrysalis-bond
name: "Chrysalis-Bond"
skill: enchanting
skill_minimum: 12
station: enchanting_circle
time_rounds: 4
target_type: ring
enchant_type: chrysalis-bond
ingredients:
  - item_tag: chrysalis-setting
    quantity: 1
  - item_tag: mutation-catalyst
    quantity: 1
  - item_tag: binding-paste
    quantity: 1
output:
  item_id: 0
  quantity: 0
success_message: "The setting merges with the ring in a flash of psychic resonance. You feel your convictions harden into armor."
failure_message: "The catalyst rejects the binding. The materials crumble to dust, but the ring is unharmed."
```

- [ ] **Step 5: Create spore-mantle.yaml recipe**

Create `_datafiles/world/dogmud/recipes/enchanting/spore-mantle.yaml`:

```yaml
id: spore-mantle
name: "Spore-Mantle"
skill: enchanting
skill_minimum: 14
station: enchanting_circle
time_rounds: 4
target_type: shoulders
enchant_type: spore-mantle
ingredients:
  - item_tag: chrysalis-setting
    quantity: 1
  - item_tag: mutation-catalyst
    quantity: 1
  - item_tag: binding-paste
    quantity: 1
output:
  item_id: 0
  quantity: 0
success_message: "Luminous spores bloom across the pauldrons, forming a living ward. Magic crackles and dies against the fungal barrier."
failure_message: "The spores fail to take root. The materials crumble to dust, but the armor is unharmed."
```

- [ ] **Step 6: Create thornguard.yaml recipe**

Create `_datafiles/world/dogmud/recipes/enchanting/thornguard.yaml`:

```yaml
id: thornguard
name: "Thornguard"
skill: enchanting
skill_minimum: 18
station: enchanting_circle
time_rounds: 5
target_type: offhand
enchant_type: thornguard
ingredients:
  - item_tag: chrysalis-setting
    quantity: 1
  - item_tag: mutation-catalyst
    quantity: 1
  - item_tag: binding-paste
    quantity: 2
output:
  item_id: 0
  quantity: 0
success_message: "Sharp barbs erupt from the shield's face, hardening into chitin thorns. Attackers will think twice before striking."
failure_message: "The catalyst rejects the binding. The materials crumble to dust, but the shield is unharmed."
```

- [ ] **Step 7: Create venomgrip.yaml recipe**

Create `_datafiles/world/dogmud/recipes/enchanting/venomgrip.yaml`:

```yaml
id: venomgrip
name: "Venomgrip"
skill: enchanting
skill_minimum: 22
station: enchanting_circle
time_rounds: 5
target_type: gloves
enchant_type: venomgrip
ingredients:
  - item_tag: chrysalis-setting
    quantity: 2
  - item_tag: mutation-catalyst
    quantity: 1
  - item_tag: binding-paste
    quantity: 1
output:
  item_id: 0
  quantity: 0
success_message: "Venom seeps from the settings into the glove leather, hardening into chitin across the knuckles. Your grip tightens with newfound strength."
failure_message: "The catalyst rejects the binding. The materials crumble to dust, but the gloves are unharmed."
```

- [ ] **Step 8: Create shadowweave.yaml recipe**

Create `_datafiles/world/dogmud/recipes/enchanting/shadowweave.yaml`:

```yaml
id: shadowweave
name: "Shadowweave"
skill: enchanting
skill_minimum: 28
station: enchanting_circle
time_rounds: 6
target_type: back
enchant_type: shadowweave
ingredients:
  - item_tag: chrysalis-setting
    quantity: 2
  - item_tag: mutation-catalyst
    quantity: 2
  - item_tag: binding-paste
    quantity: 1
output:
  item_id: 0
  quantity: 0
success_message: "Shadows pour from the settings into the cloak's fabric, weaving a darkness that drinks in light and magic alike."
failure_message: "The catalyst rejects the binding. The materials crumble to dust, but the cloak is unharmed."
```

- [ ] **Step 9: Commit all new recipes**

```bash
git add _datafiles/world/dogmud/recipes/enchanting/
git commit -m "feat: add 8 new enchanting recipes for uncovered equipment slots"
```

---

### Task 10: Production Migration — Re-apply Enchantments

**Files:**
- Create: `internal/characters/migrate_enchantments.go`
- Modify: `internal/users/users.go` (add migration call)

- [ ] **Step 1: Create the migration function**

Create `internal/characters/migrate_enchantments.go`:

```go
package characters

import (
	"github.com/GoMudEngine/GoMud/internal/enchantments"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// MigrateEnchantments re-applies all enchantment effects using the current
// enchantment definitions. This handles:
//   - Reserve pool changes (e.g. health → stamina)
//   - Rebalanced tier effects
//   - New tier counts (clamping if old tier exceeds new max)
//   - Stripping enchantments whose definitions no longer exist
//
// Called once per character load via LoadUser().
func (c *Character) MigrateEnchantments() {
	updated := 0

	// Migrate backpack items
	for i := range c.Items {
		if migrateEnchantedItem(&c.Items[i]) {
			updated++
		}
	}

	// Migrate component bag items (unlikely but safe)
	for i := range c.ComponentItems {
		if migrateEnchantedItem(&c.ComponentItems[i]) {
			updated++
		}
	}

	// Migrate potion bandolier items (unlikely but safe)
	for i := range c.PotionItems {
		if migrateEnchantedItem(&c.PotionItems[i]) {
			updated++
		}
	}

	// Migrate all equipped items
	for _, ptr := range c.Equipment.GetAllItemPtrs() {
		if migrateEnchantedItem(ptr) {
			updated++
		}
	}

	if updated > 0 {
		mudlog.Info("MigrateEnchantments", "character", c.Name, "items_updated", updated)
	}
}

// migrateEnchantedItem re-applies the enchantment definition to a single item.
// Returns true if the item was modified.
func migrateEnchantedItem(item *items.Item) bool {
	if item.EnchantType == "" {
		return false
	}

	def := enchantments.GetEnchantment(item.EnchantType)
	if def == nil {
		// Enchantment definition no longer exists — strip it
		enchantments.StripEnchantment(item)
		return true
	}

	// Clamp tier to new definition's max
	maxTier := len(def.Tiers) - 1
	if item.EnchantTier > maxTier {
		item.EnchantTier = maxTier
	}

	// Update reserve pool in case it changed
	item.ReservePool = def.ReservePool

	// Re-apply effects from the current definition
	enchantments.ApplyTier(item, def, item.EnchantTier)

	return true
}
```

- [ ] **Step 2: Add migration call to LoadUser**

In `internal/users/users.go`, find the migration block (around line 355-363) and add the new migration call after the existing ones:

```go
	loadedUser.Character.MigrateLegacyPotions()
	loadedUser.Character.MigrateEnchantments()
```

- [ ] **Step 3: Build and verify**

Run: `go build ./...`
Expected: Clean build.

- [ ] **Step 4: Commit**

```bash
git add internal/characters/migrate_enchantments.go internal/users/users.go
git commit -m "feat: add enchantment migration to re-apply effects on character load"
```

---

### Task 11: Cleanup and Final Verification

**Files:**
- Modify: `internal/crafting/crafting.go` (if cleanup needed)
- Verify: full build

- [ ] **Step 1: Check if EquipmentSlot and FindTargetItems are still used**

Run: `grep -rn "EquipmentSlot\|FindTargetItems\|FindTargetItem" internal/ --include="*.go"`

If `EquipmentSlot`, `FindTargetItems`, and `FindTargetItem` are only
referenced within `crafting.go` itself (and possibly the old craft.go code
that was replaced), they can be removed. If other code references them, leave
them.

- [ ] **Step 2: Remove unused code if safe**

If the grep from Step 1 shows no external references, remove:
- The `TargetCandidate` struct (lines 255-260)
- The `EquipmentSlot` struct (lines 264-267)
- The `FindTargetItems` function (lines 272-314)
- The `FindTargetItem` function (lines 317-323)

- [ ] **Step 3: Full build and verification**

Run: `go build ./...`
Expected: Clean build with no errors.

- [ ] **Step 4: Verify data file loading**

Run the server briefly and check logs for:
- `enchantments.LoadEnchantmentFiles()` shows `loadedCount: 18`
- `crafting.LoadRecipeFiles()` shows the enchanting recipes loading
- No panics from filepath mismatches

- [ ] **Step 5: Commit cleanup**

```bash
git add internal/crafting/crafting.go
git commit -m "chore: remove unused enchanting targeting code"
```
