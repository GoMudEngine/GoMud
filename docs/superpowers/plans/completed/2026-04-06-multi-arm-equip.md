# Multi-Arm Equip Rework — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix weapon/shield equipping for characters with the extra-arms mutation using a pair-based hand model.

**Architecture:** Arms are grouped into pairs: (Weapon+Offhand), (ExtraArm1+ExtraArm2), (ExtraArm3+ExtraArm4). A 2H weapon fills a whole pair. 1H weapons and shields fill individual slots. Arm 1 is weapon-only; arms 2-6 accept weapons or shields. Defense scores (parry/block) use the best rating across all equipped weapons/shields.

**Tech Stack:** Go, YAML help templates

---

### Task 1: Create Pair-Based Hand Slot Helper

**Files:**
- Create: `internal/characters/hand_slots.go`

This new file contains the pair model logic used by `Wear()`, `Gearup`, and future code. Keeping it separate from the 3000+ line `character.go` makes it easier to reason about.

- [ ] **Step 1: Create hand_slots.go**

```go
package characters

import (
	"github.com/GoMudEngine/GoMud/internal/items"
)

// handSlot describes one arm slot with its label and a pointer to the
// equipment field.
type handSlot struct {
	Label   string
	ItemPtr *items.Item
}

// handPair groups two adjacent arm slots.
type handPair struct {
	First  handSlot
	Second handSlot
}

// getHandPairs returns the available hand pairs for this character.
// Pair A is always present. Pairs B and C require extra-arms mutation.
// getHandPairs returns the available hand pairs for this character.
// Pair A is always present. Pairs B and C require extra-arms mutation.
// Odd mutation levels produce a "half pair" where only the First slot
// exists and the Second is nil (no 2H allowed, only 1H/shield).
func (c *Character) getHandPairs() []handPair {
	pairs := []handPair{
		{
			First:  handSlot{"wielded", &c.Equipment.Weapon},
			Second: handSlot{"offhand", &c.Equipment.Offhand},
		},
	}
	if c.ExtraArms >= 1 {
		p := handPair{
			First: handSlot{"extra arm 1", &c.Equipment.ExtraArm1},
		}
		if c.ExtraArms >= 2 {
			p.Second = handSlot{"extra arm 2", &c.Equipment.ExtraArm2}
		}
		pairs = append(pairs, p)
	}
	if c.ExtraArms >= 3 {
		p := handPair{
			First: handSlot{"extra arm 3", &c.Equipment.ExtraArm3},
		}
		if c.ExtraArms >= 4 {
			p.Second = handSlot{"extra arm 4", &c.Equipment.ExtraArm4}
		}
		pairs = append(pairs, p)
	}
	return pairs
}

// is2H returns true if the item in this slot is a 2-handed weapon that
// occupies both slots of the pair.
func (s handSlot) is2H(c *Character) bool {
	if s.ItemPtr == nil || s.ItemPtr.ItemId < 1 {
		return false
	}
	return c.HandsRequired(*s.ItemPtr) >= 2
}

// isEmpty returns true if the slot has no item or is a nil half-pair slot.
func (s handSlot) isEmpty() bool {
	return s.ItemPtr == nil || s.ItemPtr.ItemId < 1 || s.ItemPtr.IsDisabled()
}

// isHalfPair returns true if the second slot of a pair doesn't exist
// (odd number of extra arms).
func (p handPair) isHalfPair() bool {
	return p.Second.ItemPtr == nil
}

// pairIsFree returns true if both slots in the pair are empty.
// A half-pair is "free" if the first slot is empty (2H can't go here though).
func (p handPair) pairIsFree() bool {
	if p.isHalfPair() {
		return false // Can't fit a 2H in a half-pair
	}
	return p.First.isEmpty() && p.Second.isEmpty()
}

// pairOccupantCount returns how many non-empty items are in the pair.
func (p handPair) pairOccupantCount() int {
	count := 0
	if !p.First.isEmpty() {
		count++
	}
	if !p.isHalfPair() && !p.Second.isEmpty() {
		count++
	}
	return count
}

// findFirstEmptySlot scans all pairs for the first empty slot.
// If weaponOnly is true, skips the Weapon slot (arm 1) for shields.
// Returns the slot pointer and label, or nil/"" if none found.
func (c *Character) findFirstEmptySlot(pairs []handPair, isShield bool) *handSlot {
	for pi := range pairs {
		p := &pairs[pi]
		// Skip first slot of first pair (Weapon) for shields
		if pi == 0 && isShield {
			if !p.isHalfPair() && p.Second.isEmpty() {
				return &p.Second
			}
			continue
		}
		// Don't place into a slot whose pair-partner holds a 2H
		// (the second slot of a 2H pair is implicitly consumed)
		if p.First.is2H(c) {
			// Both slots consumed by the 2H
			continue
		}
		if p.First.isEmpty() {
			return &p.First
		}
		if !p.isHalfPair() && p.Second.isEmpty() {
			return &p.Second
		}
	}
	return nil
}

// findFirstFreePair scans pairs for the first where both slots are empty.
func findFirstFreePair(pairs []handPair) *handPair {
	for i := range pairs {
		if pairs[i].pairIsFree() {
			return &pairs[i]
		}
	}
	return nil
}

// findCheapestPairToDisplace finds the pair with fewest occupants
// to make room for a 2H weapon. Returns nil if no pairs exist.
func findCheapestPairToDisplace(pairs []handPair) *handPair {
	var best *handPair
	bestCount := 3 // higher than max (2)
	for i := range pairs {
		count := pairs[i].pairOccupantCount()
		if count < bestCount {
			bestCount = count
			best = &pairs[i]
		}
	}
	return best
}

// bestParryRating returns the highest ParryRating across all equipped
// weapons (Weapon slot + all extra arm slots holding weapons).
func (c *Character) bestParryRating() int {
	best := 0
	slots := c.getWeaponAndArmSlots()
	for _, slot := range slots {
		if slot.ItemId < 1 {
			continue
		}
		spec := slot.GetSpec()
		if spec.Type == items.Weapon && spec.ParryRating > best {
			best = spec.ParryRating
		}
	}
	return best
}

// bestBlockRating returns the highest BlockRating across all equipped
// shields (Offhand + all extra arm slots holding offhand-type items).
func (c *Character) bestBlockRating() int {
	best := 0
	slots := c.getWeaponAndArmSlots()
	for _, slot := range slots {
		if slot.ItemId < 1 {
			continue
		}
		spec := slot.GetSpec()
		if spec.Type == items.Offhand && spec.BlockRating > best {
			best = spec.BlockRating
		}
	}
	return best
}

// hasAnyShield returns true if any arm slot holds a shield-type item.
func (c *Character) hasAnyShield() bool {
	slots := c.getWeaponAndArmSlots()
	for _, slot := range slots {
		if slot.ItemId < 1 {
			continue
		}
		spec := slot.GetSpec()
		if spec.Type == items.Offhand && (spec.DamageReduction > 0 || spec.Subtype == items.Wearable) {
			return true
		}
	}
	return false
}

// getWeaponAndArmSlots returns item copies from all weapon/arm slots.
func (c *Character) getWeaponAndArmSlots() []items.Item {
	slots := []items.Item{
		c.Equipment.Weapon,
		c.Equipment.Offhand,
	}
	if c.ExtraArms >= 1 {
		slots = append(slots, c.Equipment.ExtraArm1)
	}
	if c.ExtraArms >= 2 {
		slots = append(slots, c.Equipment.ExtraArm2)
	}
	if c.ExtraArms >= 3 {
		slots = append(slots, c.Equipment.ExtraArm3)
	}
	if c.ExtraArms >= 4 {
		slots = append(slots, c.Equipment.ExtraArm4)
	}
	return slots
}
```

- [ ] **Step 2: Build and verify**

Run: `go build ./internal/characters/`
Expected: Clean build.

- [ ] **Step 3: Commit**

```bash
git add internal/characters/hand_slots.go
git commit -m "feat: add pair-based hand slot helpers for multi-arm equipping"
```

---

### Task 2: Rewrite Wear() Weapon/Offhand Logic

**Files:**
- Modify: `internal/characters/character.go` — `Wear()` function (lines ~3371-3503)

Replace the weapon placement logic (lines 3396-3485) with pair-based placement. The armor slot switch cases (Head, Neck, Body, etc. from line 3504 onward) stay unchanged.

- [ ] **Step 1: Replace the weapon/shield placement in Wear()**

Find the section in `Wear()` starting at the `bothMartial` check (line ~3386) through the end of `case items.Offhand:` (line ~3503). Replace it with:

```go
	canDualWield := c.CanDualWield()

	// ── Pair-based weapon/shield placement ──────────────────────────
	if spec.Type == items.Weapon || spec.Type == items.Offhand {
		pairs := c.getHandPairs()
		iHandsRequired := c.HandsRequired(i)

		isShield := spec.Type == items.Offhand

		if iHandsRequired >= 2 {
			// 2H weapon: find a free pair, or displace the cheapest pair
			freePair := findFirstFreePair(pairs)
			if freePair == nil {
				// No free pair — displace the cheapest
				freePair = findCheapestPairToDisplace(pairs)
			}
			if freePair == nil {
				return returnItems, false, `You have no free pair of hands for a two-handed weapon.`
			}

			// Check for cursed items in the pair
			if !freePair.First.isEmpty() && freePair.First.ItemPtr.IsCursed() {
				return returnItems, false, `Your ` + freePair.First.ItemPtr.DisplayName() + ` is cursed and prevents you from removing it.`
			}
			if !freePair.Second.isEmpty() && freePair.Second.ItemPtr.IsCursed() {
				return returnItems, false, `Your ` + freePair.Second.ItemPtr.DisplayName() + ` is cursed and prevents you from removing it.`
			}

			// Displace existing items
			if !freePair.First.isEmpty() {
				returnItems = append(returnItems, *freePair.First.ItemPtr)
			}
			if !freePair.Second.isEmpty() {
				returnItems = append(returnItems, *freePair.Second.ItemPtr)
			}

			// Place 2H in first slot, clear second
			*freePair.First.ItemPtr = i
			*freePair.Second.ItemPtr = items.Item{}

			c.reapplyPermabuffs()
			return returnItems, true, ``
		}

		// 1H weapon or shield
		if isShield {
			// Shields can go in arms 2-6 (not arm 1 / Weapon slot)
			slot := c.findFirstEmptySlot(pairs, true)
			if slot != nil {
				*slot.ItemPtr = i
				c.reapplyPermabuffs()
				return returnItems, true, ``
			}
			// No empty slot — displace offhand (Pair A second slot)
			if pairs[0].Second.ItemPtr.IsCursed() {
				return returnItems, false, `Your ` + pairs[0].Second.ItemPtr.DisplayName() + ` is cursed and prevents you from removing it.`
			}
			// If Pair A first slot has a 2H, can't place shield in second
			if pairs[0].First.is2H(c) {
				return returnItems, false, `Your two-handed weapon leaves no room for a shield.`
			}
			returnItems = append(returnItems, *pairs[0].Second.ItemPtr)
			*pairs[0].Second.ItemPtr = i
			c.reapplyPermabuffs()
			return returnItems, true, ``
		}

		// 1H weapon
		// Check dual-wield / martial for Pair A placement
		bothMartial := spec.Subtype == items.Claws && c.Equipment.Weapon.GetSpec().Subtype == items.Claws

		// Try to find an empty slot across all pairs
		slot := c.findFirstEmptySlot(pairs, false)
		if slot != nil {
			// If placing in Pair A offhand, need dual-wield or martial
			if slot.Label == "offhand" && !canDualWield && !bothMartial {
				// Skip offhand, try extra arms instead
				slot = nil
				for pi := 1; pi < len(pairs); pi++ {
					p := &pairs[pi]
					if p.First.is2H(c) {
						continue
					}
					if p.First.isEmpty() {
						slot = &p.First
						break
					}
					if p.Second.isEmpty() {
						slot = &p.Second
						break
					}
				}
			}
			if slot != nil {
				*slot.ItemPtr = i
				c.reapplyPermabuffs()
				return returnItems, true, ``
			}
		}

		// No empty slots — displace Weapon slot (arm 1)
		if c.Equipment.Weapon.IsCursed() {
			return returnItems, false, `Your ` + c.Equipment.Weapon.DisplayName() + ` is cursed and prevents you from removing it.`
		}
		// If current weapon is 2H, also clear offhand
		if pairs[0].First.is2H(c) && !pairs[0].Second.isEmpty() {
			returnItems = append(returnItems, *pairs[0].Second.ItemPtr)
			*pairs[0].Second.ItemPtr = items.Item{}
		}
		returnItems = append(returnItems, c.Equipment.Weapon)
		c.Equipment.Weapon = i
		c.reapplyPermabuffs()
		return returnItems, true, ``
	}

	// ── Armor slots (unchanged) ─────────────────────────────────────
	switch spec.Type {
```

Also remove the old `case items.Weapon:` and `case items.Offhand:` from the switch statement below, since weapons and offhand are now handled above. The switch should start at `case items.Head:`.

Delete these lines from the original switch:
- The `case items.Weapon:` block (lines ~3447-3485)
- The `case items.Offhand:` block (lines ~3486-3503)

And change the switch opening from `switch spec.Type {` to just continuing with the armor cases starting at `case items.Head:`.

- [ ] **Step 2: Build and verify**

Run: `go build ./internal/characters/`
Expected: Clean build.

- [ ] **Step 3: Commit**

```bash
git add internal/characters/character.go
git commit -m "feat: rewrite Wear() with pair-based weapon/shield placement"
```

---

### Task 3: Update equip.go Arm Slot Targeting

**Files:**
- Modify: `internal/usercommands/equip.go`

Replace the `arm1`/`arm2`/`arm3`/`arm4` suffix parsing with `arm#N` / `N.arm` parsing that supports arms 1-6 and allows 2H weapons + shields in extra arms.

- [ ] **Step 1: Replace the arm targeting block**

Replace lines 28-122 in `equip.go` (from the "Check for arm1/arm2..." comment through the `return true, nil` of the arm handler) with:

```go
	// Check for arm#N / N.arm targeting for explicit arm slot placement
	targetArmSlot := 0
	restForArm := strings.TrimSpace(rest)

	// Try to extract arm specifier from the end of input
	// Supports: "sword arm#3", "sword 3.arm", "sword arm3" (legacy)
	words := strings.Fields(restForArm)
	if len(words) >= 2 {
		lastWord := words[len(words)-1]
		armName, armIdx := util.GetMatchNumber(lastWord)
		if strings.EqualFold(armName, "arm") && armIdx >= 1 && armIdx <= 6 {
			targetArmSlot = armIdx
			rest = strings.TrimSpace(strings.Join(words[:len(words)-1], " "))
		}
	}
```

Then replace the entire arm-slot equip block (the old `if targetArmSlot > 0` through its `return true, nil`) with:

```go
	if targetArmSlot > 0 {
		// Check whether the user has an item in their inventory that matches
		matchItem, found := user.Character.FindInBackpack(rest)
		if !found {
			user.SendText(fmt.Sprintf(`You don't have a "%s" to wear.`, rest))
			return true, nil
		}

		iSpec := matchItem.GetSpec()
		if iSpec.Type != items.Weapon && iSpec.Type != items.Offhand {
			user.SendText(`You can only wield weapons or shields in arm slots.`)
			return true, nil
		}

		// Arm 1 (Weapon) is weapon-only
		if targetArmSlot == 1 && iSpec.Type == items.Offhand {
			user.SendText(`You can only wield weapons in your primary hand (arm 1).`)
			return true, nil
		}

		handsReq := user.Character.HandsRequired(matchItem)

		// 2H weapons must go in odd-numbered arms (first slot of a pair)
		if handsReq >= 2 && targetArmSlot%2 == 0 {
			user.SendText(`A two-handed weapon needs a pair of arms. Try arm 1, 3, or 5.`)
			return true, nil
		}

		// Map arm number to equipment slot pointers
		pairs := user.Character.GetHandPairs()
		pairIdx := (targetArmSlot - 1) / 2
		slotInPair := (targetArmSlot - 1) % 2 // 0 = first, 1 = second

		if pairIdx >= len(pairs) {
			user.SendText(fmt.Sprintf(`You don't have enough arms for arm slot %d.`, targetArmSlot))
			return true, nil
		}

		pair := &pairs[pairIdx]
		var targetSlot *handSlot
		if slotInPair == 0 {
			targetSlot = &pair.First
		} else {
			targetSlot = &pair.Second
		}

		if targetSlot.ItemPtr.IsDisabled() {
			user.SendText(fmt.Sprintf(`Arm %d is not available.`, targetArmSlot))
			return true, nil
		}

		// For 2H: displace both slots of the pair
		var displaced []items.Item
		if handsReq >= 2 {
			if !pair.First.isEmpty() {
				if pair.First.ItemPtr.IsCursed() {
					user.SendText(`Your ` + pair.First.ItemPtr.DisplayName() + ` is cursed and prevents you from removing it.`)
					return true, nil
				}
				displaced = append(displaced, *pair.First.ItemPtr)
			}
			if !pair.Second.isEmpty() {
				if pair.Second.ItemPtr.IsCursed() {
					user.SendText(`Your ` + pair.Second.ItemPtr.DisplayName() + ` is cursed and prevents you from removing it.`)
					return true, nil
				}
				displaced = append(displaced, *pair.Second.ItemPtr)
			}

			user.Character.CancelBuffsWithFlag(buffs.Hidden)
			user.Character.RemoveItem(matchItem)

			*pair.First.ItemPtr = matchItem
			*pair.Second.ItemPtr = items.Item{}

		} else {
			// 1H: displace just the target slot
			if !targetSlot.isEmpty() {
				if targetSlot.ItemPtr.IsCursed() {
					user.SendText(`Your ` + targetSlot.ItemPtr.DisplayName() + ` is cursed and prevents you from removing it.`)
					return true, nil
				}
				displaced = append(displaced, *targetSlot.ItemPtr)
			}

			// Check if this slot is consumed by a 2H in the pair's first slot
			if slotInPair == 1 && pair.First.is2H(&user.Character) {
				user.SendText(`That arm is holding a two-handed weapon with its pair. Remove the two-handed weapon first.`)
				return true, nil
			}

			user.Character.CancelBuffsWithFlag(buffs.Hidden)
			user.Character.RemoveItem(matchItem)
			*targetSlot.ItemPtr = matchItem
		}

		for _, di := range displaced {
			if di.ItemId > 0 {
				user.SendText(fmt.Sprintf(`You remove your <ansi fg="item">%s</ansi> from arm %d and return it to your backpack.`, di.DisplayName(), targetArmSlot))
				user.Character.StoreItem(di)
			}
		}

		user.SendText(fmt.Sprintf(`You wield your <ansi fg="item">%s</ansi> in arm %d.`, matchItem.DisplayName(), targetArmSlot))
		room.SendText(
			fmt.Sprintf(`<ansi fg="username">%s</ansi> wields their <ansi fg="item">%s</ansi>.`, user.Character.Name, matchItem.DisplayName()),
			user.UserId,
		)

		user.Character.Validate(true)
		events.AddToQueue(events.EquipmentChange{
			UserId:       user.UserId,
			ItemsWorn:    []items.Item{matchItem},
			ItemsRemoved: displaced,
		})

		// Quest engine: command notification
		bridge := questengine.NewGameBridge(user, room.RoomId)
		questengine.GetEngine().Notify("command", questengine.EventDetails{
			UserId:  user.UserId,
			RoomId:  room.RoomId,
			Command: "equip",
		}, bridge, bridge)

		return true, nil
	}
```

- [ ] **Step 2: Add missing imports**

Add `"github.com/GoMudEngine/GoMud/internal/util"` and `"github.com/GoMudEngine/GoMud/internal/characters"` to the import block if not already present. The `characters` import is needed for `handSlot` / `handPair` types.

**Important:** The `handSlot` and `handPair` types are unexported (lowercase). The `equip.go` file is in the `usercommands` package, so it can't access them directly. Instead, add a public method to `Character`:

In `internal/characters/hand_slots.go`, add this exported method:

```go
// GetHandPairs returns the available hand pairs (exported for equip command).
func (c *Character) GetHandPairs() []handPair {
	return c.getHandPairs()
}
```

Wait — `handPair` is unexported too. The simplest fix: **export the types**. Rename `handSlot` → `HandSlot`, `handPair` → `HandPair`, and their fields. Update all references in `hand_slots.go`.

Actually, since `equip.go` calls `pair.First.is2H(&user.Character)` and similar, the types and methods need to be exported. Go back to Task 1's `hand_slots.go` and capitalize all type names, field names, and method names:
- `handSlot` → `HandSlot`
- `handPair` → `HandPair`
- `Label`, `ItemPtr` (already capitalized)
- `is2H` → `Is2H`
- `isEmpty` → `IsEmpty`
- `pairIsFree` → `PairIsFree`
- `pairOccupantCount` → `PairOccupantCount`
- `getHandPairs` → `GetHandPairs` (remove the private wrapper)
- `findFirstEmptySlot` → `FindFirstEmptySlot`
- `findFirstFreePair` → `FindFirstFreePair`
- `findCheapestPairToDisplace` → `FindCheapestPairToDisplace`

Update all references in the `Wear()` code from Task 2 accordingly.

- [ ] **Step 3: Build and verify**

Run: `go build ./internal/characters/ ./internal/usercommands/`
Expected: Clean build.

- [ ] **Step 4: Commit**

```bash
git add internal/usercommands/equip.go internal/characters/hand_slots.go
git commit -m "feat: equip command supports arm#N targeting with 2H pair logic"
```

---

### Task 4: Fix Defense Scores — Parry and Block From Extra Arms

**Files:**
- Modify: `internal/characters/character.go` — `GetDefenseScore()` and `HasShield()`

- [ ] **Step 1: Update DefenseParry in GetDefenseScore()**

Replace the `case DefenseParry:` block (lines 2156-2163) with:

```go
	case DefenseParry:
		// Parry: Dexterity + WeaponCombat skill + best weapon ParryRating
		weaponSkill := float64(c.GetSkillLevel(skills.WeaponCombat)) * skillWeight
		parryRating := c.bestParryRating()
		return dex + weaponSkill + float64(parryRating)
```

- [ ] **Step 2: Update DefenseBlock in GetDefenseScore()**

Replace the `case DefenseBlock:` block (lines 2165-2173) with:

```go
	case DefenseBlock:
		// Block: (Strength + Dexterity)/2 + WeaponCombat skill + best shield BlockRating
		str := float64(c.Stats.Strength.ValueAdj)
		weaponSkill := float64(c.GetSkillLevel(skills.WeaponCombat)) * skillWeight
		blockRating := c.bestBlockRating()
		return (str+dex)/2 + weaponSkill + float64(blockRating)
```

- [ ] **Step 3: Update HasShield()**

Replace the `HasShield()` function (lines 2072-2083) with:

```go
func (c *Character) HasShield() bool {
	// Species-based natural bash (elementals, golems, etc.)
	if sp := species.GetSpecies(c.SpeciesId); sp != nil && sp.NaturalBash {
		return true
	}
	return c.hasAnyShield()
}
```

- [ ] **Step 4: Build and test**

Run: `go build ./... && go test ./internal/characters/ -run TestGetDefenseScore -v`

The existing test only checks Weapon and Offhand slots so it should still pass. The new behavior (extra arms contributing) is an extension, not a breaking change.

- [ ] **Step 5: Commit**

```bash
git add internal/characters/character.go
git commit -m "feat: parry/block defense scores use best rating across all arm slots"
```

---

### Task 5: Fix Gearup ("wear all") for Extra Arms

**Files:**
- Modify: `internal/usercommands/gearup.go`
- Modify: `internal/mobcommands/gearup.go`

- [ ] **Step 1: Update user gearup**

Replace the entire `Gearup` function in `internal/usercommands/gearup.go` with:

```go
func Gearup(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if rest == "all" {
		rest = ""
	}

	allBackpackItems := user.Character.GetAllBackpackItems()
	wearableCount := 0
	wornSomething := false

	// Sort by value descending so best items get equipped first
	sort.Slice(allBackpackItems, func(i, j int) bool {
		return allBackpackItems[i].GetSpec().Value > allBackpackItems[j].GetSpec().Value
	})

	for _, itm := range allBackpackItems {
		itmSpec := itm.GetSpec()

		if itmSpec.Type != items.Weapon && itmSpec.Subtype != items.Wearable {
			continue
		}

		wearableCount++

		// Try to equip via the normal wear command — it handles pair
		// logic, slot finding, and displacement automatically.
		// Using Command() so the player sees the normal equip messages.
		user.Command(fmt.Sprintf(`wear !%d`, itm.ItemId), -1)

		// Check if the item left the backpack (was actually equipped)
		if _, stillInBp := user.Character.FindInBackpack(fmt.Sprintf("!%d", itm.ItemId)); !stillInBp {
			wornSomething = true
		}
	}

	if !wornSomething {
		if wearableCount == 0 {
			user.SendText("You have nothing to wear.")
		} else {
			user.SendText("You're already wearing everything you can!")
		}
	}

	return true, nil
}
```

Add `"sort"` to the import block.

Note: This approach delegates to the existing `wear` command which now uses pair-based logic. Each item is tried in value order — best items first. The wear command will auto-place into the first available slot and skip if all slots are full.

- [ ] **Step 2: Update mob gearup**

Apply the same pattern to `internal/mobcommands/gearup.go`. Replace the `else` block (lines 37-86, the no-argument case) with:

```go
	} else {
		allBackpackItems := mob.Character.GetAllBackpackItems()

		// Sort by value descending
		sort.Slice(allBackpackItems, func(i, j int) bool {
			return allBackpackItems[i].GetSpec().Value > allBackpackItems[j].GetSpec().Value
		})

		isCharmed := mob.Character.IsCharmed()

		for _, itm := range allBackpackItems {
			itmSpec := itm.GetSpec()

			if itmSpec.Type != items.Weapon && itmSpec.Subtype != items.Wearable {
				continue
			}

			// Track what was worn before
			oldEquipped := mob.Character.Equipment.GetAllItems()

			mob.Command(fmt.Sprintf(`wear !%d`, itm.ItemId))

			// If charmed, drop any displaced items
			if isCharmed {
				newEquipped := mob.Character.Equipment.GetAllItems()
				for _, oldItm := range oldEquipped {
					if oldItm.ItemId < 1 {
						continue
					}
					found := false
					for _, newItm := range newEquipped {
						if oldItm.ItemId == newItm.ItemId {
							found = true
							break
						}
					}
					if !found {
						mob.Command(fmt.Sprintf(`drop !%d`, oldItm.ItemId))
					}
				}
			}
		}
	}
```

Add `"sort"` to the import block.

- [ ] **Step 3: Build and verify**

Run: `go build ./internal/usercommands/ ./internal/mobcommands/`
Expected: Clean build.

- [ ] **Step 4: Commit**

```bash
git add internal/usercommands/gearup.go internal/mobcommands/gearup.go
git commit -m "feat: gearup fills extra arm slots, best items first"
```

---

### Task 6: Update Help Files

**Files:**
- Modify: `_datafiles/world/dogmud/templates/help/equip.template`
- Modify: `_datafiles/world/dogmud/templates/help/equipment.template`

- [ ] **Step 1: Update equip.template**

Replace `_datafiles/world/dogmud/templates/help/equip.template`:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">equip</ansi>

The <ansi fg="command">equip</ansi> command wields or wears an item from
your backpack.

<ansi fg="yellow">Usage: </ansi>

  <ansi fg="command">equip stick</ansi>          Wield a stick.
  <ansi fg="command">equip cloak</ansi>          Wear a cloak (back slot).
  <ansi fg="command">equip all</ansi>            Equip best gear from backpack.
  <ansi fg="command">equip sword arm#3</ansi>    Wield in arm 3 (extra arms).
  <ansi fg="command">equip sword 3.arm</ansi>    Same thing, alternate syntax.
  <ansi fg="command">equip shield arm#2</ansi>   Shield in offhand (arm 2).

Aliases: <ansi fg="command">wield</ansi>, <ansi fg="command">wear</ansi>

<ansi fg="yellow">━━━ Arm Pairs (Extra Arms Mutation) ━━━</ansi>

With extra arms, your hands are grouped into pairs:

  <ansi fg="yellow">Pair A:</ansi> Arm 1 (weapon) + Arm 2 (offhand)
  <ansi fg="yellow">Pair B:</ansi> Arm 3 + Arm 4  (extra arms level 1-2)
  <ansi fg="yellow">Pair C:</ansi> Arm 5 + Arm 6  (extra arms level 3-4)

Two-handed weapons occupy a full pair. They must go in an
odd-numbered arm (1, 3, or 5).

Arm 1 only holds weapons. Arms 2-6 hold weapons or shields.

Without a specific arm target, weapons and shields fill the
first available slot automatically.

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help equipment</ansi>,
  <ansi fg="command">help item-names</ansi>, <ansi fg="command">help remove</ansi>
```

- [ ] **Step 2: Update equipment.template Extra Arms section**

In `_datafiles/world/dogmud/templates/help/equipment.template`, replace the `━━━ Mutations ━━━` section (lines 81-92) with:

```
<ansi fg="yellow">━━━ Mutations ━━━</ansi>

Some mutations alter your available slots:

  <ansi fg="yellow">Extra Arms</ansi>   Unlocks additional arm and wrist slots,
               grouped in pairs:
               Level 1-2: Pair B (arms 3-4, wrists 3-4)
               Level 3-4: Pair C (arms 5-6, wrists 5-6)

               Two-handed weapons fill a whole pair. Shields
               and one-handed weapons fill a single slot.
               Arm 1 holds weapons only; arms 2-6 hold
               weapons or shields.

  <ansi fg="yellow">Tail</ansi>         Replaces your legs slot with a tail slot.
               Changes how certain combat moves work.
```

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/templates/help/equip.template _datafiles/world/dogmud/templates/help/equipment.template
git commit -m "docs: update equip and equipment help files for pair-based arm system"
```

---

### Task 7: Build, Test, and Verify

**Files:**
- None (verification only)

- [ ] **Step 1: Full build**

Run: `go build ./...`
Expected: Clean build.

- [ ] **Step 2: Run existing tests**

Run: `go test ./internal/characters/ ./internal/combat/ ./internal/crafting/`
Expected: All pass.

- [ ] **Step 3: Verify no stale references**

Run: `grep -rn "arm1\|arm2\|arm3\|arm4" internal/usercommands/equip.go`
Expected: No matches (old `arm1`/`arm2` suffix parsing is gone).

Run: `grep -rn "iHandsRequired == 2" internal/characters/character.go | grep -i "clear\|extra"`
Expected: No matches (the old "clear all extra arms" block is gone).

- [ ] **Step 4: Commit if any cleanup needed**

```bash
git add -A && git commit -m "chore: cleanup after multi-arm equip rework"
```
