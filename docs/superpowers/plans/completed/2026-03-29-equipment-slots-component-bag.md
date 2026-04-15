# Equipment Slots + Component Bag Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add new equipment slots (wrist, back, shoulders, ring2, component bag), extend Extra Arms mutation to levels 3-4 with wrist slots, and implement the component bag crafting material system.

**Architecture:** Phase 1 adds the struct fields and item types (foundation). Phase 2 wires equip/remove routing. Phase 3 adds the component bag system. Phase 4 handles combat and mutation changes. Phase 5 is content and documentation. Each phase builds on the previous but produces a compilable, testable state.

**Tech Stack:** Go, YAML data files, Go templates

---

## Task 1: Item Type Constants + ItemSpec Fields

**Files:**
- Modify: `internal/items/itemspec.go`

- [ ] **Step 1: Add new ItemType constants**

In `internal/items/itemspec.go`, find the block of ItemType constants (around lines 101-128). After the `Ring` constant (line 112), add:

```go
	Wrist        ItemType = "wrist"        // Bracelets, bracers
	Back         ItemType = "back"         // Cloaks, backpacks
	Shoulders    ItemType = "shoulders"    // Pauldrons, mantles
	ComponentBag ItemType = "componentbag" // Crafting material bags
```

- [ ] **Step 2: Add new types to ItemTypes() list**

In `internal/items/itemspec.go`, find the `ItemTypes()` function (line 41). Add entries for the new types in the return slice. They share the 20000-29999 armor range (except ComponentBag at 30000-39999). Add after the Feet entry:

```go
		{string(Wrist), `Worn on the wrist.`, 20000, 10000, 29999},
		{string(Back), `Worn on the back.`, 20000, 10000, 29999},
		{string(Shoulders), `Worn on the shoulders.`, 20000, 10000, 29999},
		{string(ComponentBag), `A bag for crafting materials.`, 30000, 10000, 39999},
```

- [ ] **Step 3: Add IsComponent, WeightReduction, BagCapacity to ItemSpec**

In `internal/items/itemspec.go`, find the `ItemSpec` struct. Add these fields alongside the existing equipment fields:

```go
	IsComponent     bool    `yaml:"is_component,omitempty"`     // Auto-routes to component bag on pickup
	WeightReduction float64 `yaml:"weight_reduction,omitempty"` // 0.0-1.0, fraction of contents weight reduced
	BagCapacity     int     `yaml:"bag_capacity,omitempty"`     // Max items storable in component bag
```

- [ ] **Step 4: Build and verify**

Run: `go build ./...`

- [ ] **Step 5: Commit**

```bash
git add internal/items/itemspec.go
git commit -m "$(cat <<'EOF'
feat: add wrist, back, shoulders, componentbag item types

New ItemType constants and ItemSpec fields: IsComponent (auto-routes
to component bag), WeightReduction (bag weight reduction fraction),
BagCapacity (component bag size limit).

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Worn Struct Expansion

**Files:**
- Modify: `internal/characters/worn.go`

- [ ] **Step 1: Add new slot fields to Worn struct**

In `internal/characters/worn.go`, find the Worn struct (lines 5-18). Replace the full struct with:

```go
type Worn struct {
	Weapon       items.Item `yaml:"weapon,omitempty"`
	Offhand      items.Item `yaml:"offhand,omitempty"`
	ExtraArm1    items.Item `yaml:"extraarm1,omitempty"`
	ExtraArm2    items.Item `yaml:"extraarm2,omitempty"`
	ExtraArm3    items.Item `yaml:"extraarm3,omitempty"`
	ExtraArm4    items.Item `yaml:"extraarm4,omitempty"`
	Head         items.Item `yaml:"head,omitempty"`
	Neck         items.Item `yaml:"neck,omitempty"`
	Shoulders    items.Item `yaml:"shoulders,omitempty"`
	Body         items.Item `yaml:"body,omitempty"`
	Back         items.Item `yaml:"back,omitempty"`
	Belt         items.Item `yaml:"belt,omitempty"`
	Wrist1       items.Item `yaml:"wrist1,omitempty"`
	Wrist2       items.Item `yaml:"wrist2,omitempty"`
	ExtraWrist1  items.Item `yaml:"extrawrist1,omitempty"`
	ExtraWrist2  items.Item `yaml:"extrawrist2,omitempty"`
	ExtraWrist3  items.Item `yaml:"extrawrist3,omitempty"`
	ExtraWrist4  items.Item `yaml:"extrawrist4,omitempty"`
	Gloves       items.Item `yaml:"gloves,omitempty"`
	Ring         items.Item `yaml:"ring,omitempty"`
	Ring2        items.Item `yaml:"ring2,omitempty"`
	Legs         items.Item `yaml:"legs,omitempty"`
	Feet         items.Item `yaml:"feet,omitempty"`
	ComponentBag items.Item `yaml:"componentbag,omitempty"`
}
```

- [ ] **Step 2: Update StatMod() to include all new slots**

Replace the `StatMod()` method to include every slot. Add all new slots to the sum chain:

```go
func (w *Worn) StatMod(stat ...string) int {
	return w.Weapon.StatMod(stat...) +
		w.Offhand.StatMod(stat...) +
		w.ExtraArm1.StatMod(stat...) +
		w.ExtraArm2.StatMod(stat...) +
		w.ExtraArm3.StatMod(stat...) +
		w.ExtraArm4.StatMod(stat...) +
		w.Head.StatMod(stat...) +
		w.Neck.StatMod(stat...) +
		w.Shoulders.StatMod(stat...) +
		w.Body.StatMod(stat...) +
		w.Back.StatMod(stat...) +
		w.Belt.StatMod(stat...) +
		w.Wrist1.StatMod(stat...) +
		w.Wrist2.StatMod(stat...) +
		w.ExtraWrist1.StatMod(stat...) +
		w.ExtraWrist2.StatMod(stat...) +
		w.ExtraWrist3.StatMod(stat...) +
		w.ExtraWrist4.StatMod(stat...) +
		w.Gloves.StatMod(stat...) +
		w.Ring.StatMod(stat...) +
		w.Ring2.StatMod(stat...) +
		w.Legs.StatMod(stat...) +
		w.Feet.StatMod(stat...) +
		w.ComponentBag.StatMod(stat...)
}
```

- [ ] **Step 3: Update EnableAll(), GetAllItems(), GetAllItemPtrs()**

Update all three methods to include every new slot. Follow the same pattern as the existing code — for each new slot, add the `if slot.ItemId < 0` check in EnableAll, the `if slot.ItemId > 0` check in GetAllItems and GetAllItemPtrs. The implementer should read the current file and extend all three methods with the new slots.

- [ ] **Step 4: Update GetSlotPointer()**

Add new cases to the switch in `GetSlotPointer()`:

```go
	case "worn - shoulders":
		return &w.Shoulders
	case "worn - back":
		return &w.Back
	case "worn - wrist", "worn - wrist1":
		return &w.Wrist1
	case "worn - wrist2":
		return &w.Wrist2
	case "extra wrist 1":
		return &w.ExtraWrist1
	case "extra wrist 2":
		return &w.ExtraWrist2
	case "extra wrist 3":
		return &w.ExtraWrist3
	case "extra wrist 4":
		return &w.ExtraWrist4
	case "extra arm 3":
		return &w.ExtraArm3
	case "extra arm 4":
		return &w.ExtraArm4
	case "worn - ring2":
		return &w.Ring2
	case "worn - componentbag":
		return &w.ComponentBag
```

- [ ] **Step 5: Update GetAllSlotTypes()**

Add new types to the return slice:

```go
func GetAllSlotTypes() []string {
	return []string{
		string(items.Weapon),
		string(items.Offhand),
		string(items.Head),
		string(items.Neck),
		string(items.Shoulders),
		string(items.Body),
		string(items.Back),
		string(items.Belt),
		string(items.Wrist),
		string(items.Gloves),
		string(items.Ring),
		string(items.Legs),
		string(items.Feet),
		string(items.ComponentBag),
	}
}
```

- [ ] **Step 6: Build and verify**

Run: `go build ./...`

- [ ] **Step 7: Commit**

```bash
git add internal/characters/worn.go
git commit -m "$(cat <<'EOF'
feat: expand Worn struct with wrist, back, shoulders, ring2,
component bag, extra arm 3-4, extra wrist 1-4 slots

All methods updated: StatMod, EnableAll, GetAllItems,
GetAllItemPtrs, GetSlotPointer, GetAllSlotTypes.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Weight Calculation Fix + Reduction Logic

**Files:**
- Modify: `internal/characters/character.go`

- [ ] **Step 1: Add ComponentItems field to Character**

In `internal/characters/character.go`, find the Character struct. Add near the `Items` field:

```go
	ComponentItems []items.Item `yaml:"componentitems,omitempty"` // Contents of equipped component bag
```

- [ ] **Step 2: Overhaul GetCarriedWeight()**

Replace `GetCarriedWeight()` (lines 344-386). The new version:
1. Sums backpack items with Back slot weight reduction
2. Sums component items with ComponentBag weight reduction
3. Sums ALL equipped slots (fixing the ExtraArm1/2 bug, adding all new slots)

```go
func (c *Character) GetCarriedWeight() float64 {
	// Backpack item weights
	backpackWeight := 0.0
	for _, item := range c.Items {
		backpackWeight += item.GetSpec().GetWeight()
	}
	// Apply Back slot weight reduction to backpack contents
	if c.Equipment.Back.ItemId > 0 {
		reduction := c.Equipment.Back.GetSpec().WeightReduction
		if reduction > 0 && reduction <= 1.0 {
			backpackWeight *= (1.0 - reduction)
		}
	}

	// Component bag item weights
	componentWeight := 0.0
	for _, item := range c.ComponentItems {
		componentWeight += item.GetSpec().GetWeight()
	}
	// Apply ComponentBag weight reduction
	if c.Equipment.ComponentBag.ItemId > 0 {
		reduction := c.Equipment.ComponentBag.GetSpec().WeightReduction
		if reduction > 0 && reduction <= 1.0 {
			componentWeight *= (1.0 - reduction)
		}
	}

	// Equipped item weights (all slots)
	equippedWeight := 0.0
	for _, item := range c.Equipment.GetAllItems() {
		equippedWeight += item.GetSpec().GetWeight()
	}

	return backpackWeight + componentWeight + equippedWeight
}
```

Note: `GetAllItems()` was updated in Task 2 to include all slots, so this naturally covers every equipped item.

- [ ] **Step 3: Build and verify**

Run: `go build ./...`

- [ ] **Step 4: Update carry capacity tests**

The carry capacity tests may need adjustment if they create characters and check weight. Run `go test ./internal/characters/ -v` and fix any failures related to the new weight formula.

- [ ] **Step 5: Commit**

```bash
git add internal/characters/character.go
git commit -m "$(cat <<'EOF'
feat: overhaul weight calculation with bag reductions

GetCarriedWeight now applies Back slot weight reduction to backpack
contents and ComponentBag reduction to component items. Fixes bug
where ExtraArm1/2 weights weren't counted. Uses GetAllItems() for
all equipped slot weights.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Equip Routing — Wear() + Disabled Slots

**Files:**
- Modify: `internal/characters/character.go`

- [ ] **Step 1: Raise ExtraArms cap from 2 to 4**

In `internal/characters/character.go`, find lines 2560-2568:

```go
	if lvl, ok := c.Mutations["extra-arms"]; ok && lvl > 0 {
		c.ExtraArms = lvl
		if c.ExtraArms > 2 {
			c.ExtraArms = 2
		}
	}
```

Replace with:

```go
	if lvl, ok := c.Mutations["extra-arms"]; ok && lvl > 0 {
		c.ExtraArms = lvl
		if c.ExtraArms > 4 {
			c.ExtraArms = 4
		}
	}
```

- [ ] **Step 2: Add new slots to Wear() method**

In the `Wear()` method, add routing for new slot types. In the slot-routing switch (around line 2912), add cases after the existing `items.Feet` case:

```go
	case items.Wrist:
		if c.Equipment.Wrist1.IsDisabled() && c.Equipment.Wrist2.IsDisabled() {
			return returnItems, false, `You can't wear things on your wrists.`
		}
		// Fill first empty wrist slot
		if !c.Equipment.Wrist1.IsDisabled() && c.Equipment.Wrist1.ItemId == 0 {
			c.Equipment.Wrist1 = i
		} else if !c.Equipment.Wrist2.IsDisabled() && c.Equipment.Wrist2.ItemId == 0 {
			c.Equipment.Wrist2 = i
		} else if c.ExtraArms >= 1 && !c.Equipment.ExtraWrist1.IsDisabled() && c.Equipment.ExtraWrist1.ItemId == 0 {
			c.Equipment.ExtraWrist1 = i
		} else if c.ExtraArms >= 2 && !c.Equipment.ExtraWrist2.IsDisabled() && c.Equipment.ExtraWrist2.ItemId == 0 {
			c.Equipment.ExtraWrist2 = i
		} else if c.ExtraArms >= 3 && !c.Equipment.ExtraWrist3.IsDisabled() && c.Equipment.ExtraWrist3.ItemId == 0 {
			c.Equipment.ExtraWrist3 = i
		} else if c.ExtraArms >= 4 && !c.Equipment.ExtraWrist4.IsDisabled() && c.Equipment.ExtraWrist4.ItemId == 0 {
			c.Equipment.ExtraWrist4 = i
		} else {
			// All full — displace Wrist1
			returnItems = append(returnItems, c.Equipment.Wrist1)
			c.Equipment.Wrist1 = i
		}
	case items.Back:
		if c.Equipment.Back.IsDisabled() {
			return returnItems, false, `You can't wear things on your back.`
		}
		returnItems = append(returnItems, c.Equipment.Back)
		c.Equipment.Back = i
	case items.Shoulders:
		if c.Equipment.Shoulders.IsDisabled() {
			return returnItems, false, `You can't wear things on your shoulders.`
		}
		returnItems = append(returnItems, c.Equipment.Shoulders)
		c.Equipment.Shoulders = i
	case items.ComponentBag:
		if c.Equipment.ComponentBag.IsDisabled() {
			return returnItems, false, `You can't equip a component bag.`
		}
		returnItems = append(returnItems, c.Equipment.ComponentBag)
		c.Equipment.ComponentBag = i
		// Auto-sort component items from backpack into bag
		c.SortComponentItems()
```

Also update the Ring case to overflow to Ring2:

Find the existing Ring case:

```go
	case items.Ring:
		if c.Equipment.Ring.IsDisabled() {
			return returnItems, false, `You can't wear rings.`
		}
		returnItems = append(returnItems, c.Equipment.Ring)
		c.Equipment.Ring = i
```

Replace with:

```go
	case items.Ring:
		if c.Equipment.Ring.IsDisabled() && c.Equipment.Ring2.IsDisabled() {
			return returnItems, false, `You can't wear rings.`
		}
		if !c.Equipment.Ring.IsDisabled() && c.Equipment.Ring.ItemId == 0 {
			c.Equipment.Ring = i
		} else if !c.Equipment.Ring2.IsDisabled() && c.Equipment.Ring2.ItemId == 0 {
			c.Equipment.Ring2 = i
		} else {
			returnItems = append(returnItems, c.Equipment.Ring)
			c.Equipment.Ring = i
		}
```

- [ ] **Step 3: Extend extra arm overflow to arms 3-4**

In Wear(), find the extra arms block (lines 2895-2909). Replace with:

```go
	if spec.Type == items.Weapon && iHandsRequired < 2 && c.ExtraArms > 0 {
		if c.Equipment.Weapon.ItemId > 0 && c.Equipment.Offhand.ItemId > 0 {
			if c.ExtraArms >= 1 && !c.Equipment.ExtraArm1.IsDisabled() && c.Equipment.ExtraArm1.ItemId == 0 {
				c.Equipment.ExtraArm1 = i
				c.reapplyPermabuffs()
				return returnItems, true, ``
			}
			if c.ExtraArms >= 2 && !c.Equipment.ExtraArm2.IsDisabled() && c.Equipment.ExtraArm2.ItemId == 0 {
				c.Equipment.ExtraArm2 = i
				c.reapplyPermabuffs()
				return returnItems, true, ``
			}
			if c.ExtraArms >= 3 && !c.Equipment.ExtraArm3.IsDisabled() && c.Equipment.ExtraArm3.ItemId == 0 {
				c.Equipment.ExtraArm3 = i
				c.reapplyPermabuffs()
				return returnItems, true, ``
			}
			if c.ExtraArms >= 4 && !c.Equipment.ExtraArm4.IsDisabled() && c.Equipment.ExtraArm4.ItemId == 0 {
				c.Equipment.ExtraArm4 = i
				c.reapplyPermabuffs()
				return returnItems, true, ``
			}
		}
	}
```

Also update the 2H weapon clearing to include arms 3-4:

```go
		if iHandsRequired == 2 {
			if c.Equipment.ExtraArm1.ItemId > 0 {
				returnItems = append(returnItems, c.Equipment.ExtraArm1)
				c.Equipment.ExtraArm1 = items.Item{}
			}
			if c.Equipment.ExtraArm2.ItemId > 0 {
				returnItems = append(returnItems, c.Equipment.ExtraArm2)
				c.Equipment.ExtraArm2 = items.Item{}
			}
			if c.Equipment.ExtraArm3.ItemId > 0 {
				returnItems = append(returnItems, c.Equipment.ExtraArm3)
				c.Equipment.ExtraArm3 = items.Item{}
			}
			if c.Equipment.ExtraArm4.ItemId > 0 {
				returnItems = append(returnItems, c.Equipment.ExtraArm4)
				c.Equipment.ExtraArm4 = items.Item{}
			}
		}
```

- [ ] **Step 4: Add new slots to disabled slot handling**

In the disabled slot switch (lines 2667-2718), add cases for new types:

```go
			case items.Wrist:
				if c.Equipment.Wrist1.ItemId > 0 {
					itemFoundInDisabledSlot = c.Equipment.Wrist1
				}
				c.Equipment.Wrist1 = items.ItemDisabledSlot
				if c.Equipment.Wrist2.ItemId > 0 {
					c.StoreItem(c.Equipment.Wrist2)
				}
				c.Equipment.Wrist2 = items.ItemDisabledSlot
			case items.Back:
				if c.Equipment.Back.ItemId > 0 {
					itemFoundInDisabledSlot = c.Equipment.Back
				}
				c.Equipment.Back = items.ItemDisabledSlot
			case items.Shoulders:
				if c.Equipment.Shoulders.ItemId > 0 {
					itemFoundInDisabledSlot = c.Equipment.Shoulders
				}
				c.Equipment.Shoulders = items.ItemDisabledSlot
			case items.ComponentBag:
				if c.Equipment.ComponentBag.ItemId > 0 {
					itemFoundInDisabledSlot = c.Equipment.ComponentBag
				}
				c.Equipment.ComponentBag = items.ItemDisabledSlot
```

- [ ] **Step 5: Update RemoveFromBody() with new slots**

Add new else-if branches for every new slot:

```go
	} else if i.Equals(c.Equipment.ExtraArm3) {
		c.Equipment.ExtraArm3 = items.Item{}
	} else if i.Equals(c.Equipment.ExtraArm4) {
		c.Equipment.ExtraArm4 = items.Item{}
	} else if i.Equals(c.Equipment.Shoulders) {
		c.Equipment.Shoulders = items.Item{}
	} else if i.Equals(c.Equipment.Back) {
		c.Equipment.Back = items.Item{}
	} else if i.Equals(c.Equipment.Wrist1) {
		c.Equipment.Wrist1 = items.Item{}
	} else if i.Equals(c.Equipment.Wrist2) {
		c.Equipment.Wrist2 = items.Item{}
	} else if i.Equals(c.Equipment.ExtraWrist1) {
		c.Equipment.ExtraWrist1 = items.Item{}
	} else if i.Equals(c.Equipment.ExtraWrist2) {
		c.Equipment.ExtraWrist2 = items.Item{}
	} else if i.Equals(c.Equipment.ExtraWrist3) {
		c.Equipment.ExtraWrist3 = items.Item{}
	} else if i.Equals(c.Equipment.ExtraWrist4) {
		c.Equipment.ExtraWrist4 = items.Item{}
	} else if i.Equals(c.Equipment.Ring2) {
		c.Equipment.Ring2 = items.Item{}
	} else if i.Equals(c.Equipment.ComponentBag) {
		// Spill component bag contents back to backpack
		for _, ci := range c.ComponentItems {
			c.Items = append(c.Items, ci)
		}
		c.ComponentItems = nil
		c.Equipment.ComponentBag = items.Item{}
```

- [ ] **Step 6: Update FindOnBody() with new slots**

Add the new slots to the `items.FindMatchIn` call in `FindOnBody()`. Read the current code and add all new slot items to the argument list.

- [ ] **Step 7: Update FindItem() with new slots**

The `FindItem()` method (added in the inventory rework) builds a pool of candidates from slot items. Add all new slots to the `slotItems` list with appropriate source labels.

- [ ] **Step 8: Build and verify**

Run: `go build ./...`

- [ ] **Step 9: Commit**

```bash
git add internal/characters/character.go
git commit -m "$(cat <<'EOF'
feat: equip routing for all new slots + extra arms 3-4

Wear() routes wrist (overflow to extra wrists), ring (overflow to
ring2), back, shoulders, componentbag. Extra arms cap raised to 4.
2H weapons clear all extra arms. RemoveFromBody spills component
bag contents. Disabled slot handling covers all new types.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Equip Command — Arm 3-4 Suffix Parsing

**Files:**
- Modify: `internal/usercommands/equip.go`

- [ ] **Step 1: Extend arm suffix parsing**

In `internal/usercommands/equip.go`, find the arm suffix parsing (lines 26-35):

```go
	if strings.HasSuffix(restLower, " arm1") {
		targetArmSlot = 1
		rest = strings.TrimSpace(rest[:len(rest)-5])
	} else if strings.HasSuffix(restLower, " arm2") {
		targetArmSlot = 2
		rest = strings.TrimSpace(rest[:len(rest)-5])
	}
```

Replace with:

```go
	if strings.HasSuffix(restLower, " arm1") {
		targetArmSlot = 1
		rest = strings.TrimSpace(rest[:len(rest)-5])
	} else if strings.HasSuffix(restLower, " arm2") {
		targetArmSlot = 2
		rest = strings.TrimSpace(rest[:len(rest)-5])
	} else if strings.HasSuffix(restLower, " arm3") {
		targetArmSlot = 3
		rest = strings.TrimSpace(rest[:len(rest)-5])
	} else if strings.HasSuffix(restLower, " arm4") {
		targetArmSlot = 4
		rest = strings.TrimSpace(rest[:len(rest)-5])
	}
```

- [ ] **Step 2: Update the direct arm equip block**

Find the direct arm equip block (lines 68-75) where it assigns to ExtraArm1/ExtraArm2. Replace the if/else with a switch:

```go
		var oldItem items.Item
		switch targetArmSlot {
		case 1:
			oldItem = user.Character.Equipment.ExtraArm1
			user.Character.Equipment.ExtraArm1 = matchItem
		case 2:
			oldItem = user.Character.Equipment.ExtraArm2
			user.Character.Equipment.ExtraArm2 = matchItem
		case 3:
			oldItem = user.Character.Equipment.ExtraArm3
			user.Character.Equipment.ExtraArm3 = matchItem
		case 4:
			oldItem = user.Character.Equipment.ExtraArm4
			user.Character.Equipment.ExtraArm4 = matchItem
		}
```

- [ ] **Step 3: Build and verify**

Run: `go build ./...`

- [ ] **Step 4: Commit**

```bash
git add internal/usercommands/equip.go
git commit -m "$(cat <<'EOF'
feat: support equip arm3/arm4 suffix for extra arm slots

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Component Bag — StoreItem + Sort + Crafting

**Files:**
- Modify: `internal/characters/character.go`
- Create: `internal/usercommands/sort.go`
- Modify: `internal/crafting/crafting.go`

- [ ] **Step 1: Add SortComponentItems() method**

In `internal/characters/character.go`, add after `StoreItem()`:

```go
// SortComponentItems moves is_component items from backpack into ComponentItems
// up to the equipped bag's capacity. Returns the count of items moved.
func (c *Character) SortComponentItems() int {
	if c.Equipment.ComponentBag.ItemId < 1 {
		return 0
	}
	bagSpec := c.Equipment.ComponentBag.GetSpec()
	capacity := bagSpec.BagCapacity
	if capacity <= 0 {
		return 0
	}

	moved := 0
	remaining := make([]items.Item, 0, len(c.Items))
	for _, item := range c.Items {
		if item.GetSpec().IsComponent && len(c.ComponentItems) < capacity {
			c.ComponentItems = append(c.ComponentItems, item)
			moved++
		} else {
			remaining = append(remaining, item)
		}
	}
	c.Items = remaining
	return moved
}
```

- [ ] **Step 2: Update StoreItem() with auto-routing**

Replace `StoreItem()` (lines 1320-1339):

```go
func (c *Character) StoreItem(i items.Item) bool {
	if i.ItemId < 1 {
		return false
	}

	i.Validate()

	newWeight := c.GetCarriedWeight() + i.GetSpec().GetWeight()
	capacity := c.CarryCapacity()

	if newWeight > capacity*2.0 {
		return false
	}

	// Auto-route component items to the component bag
	iSpec := i.GetSpec()
	if iSpec.IsComponent && c.Equipment.ComponentBag.ItemId > 0 {
		bagSpec := c.Equipment.ComponentBag.GetSpec()
		if bagSpec.BagCapacity > 0 && len(c.ComponentItems) < bagSpec.BagCapacity {
			c.ComponentItems = append(c.ComponentItems, i)
			return true
		}
	}

	c.Items = append(c.Items, i)
	return true
}
```

- [ ] **Step 3: Create sort command**

Create `internal/usercommands/sort.go`:

```go
package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Sort(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if user.Character.Equipment.ComponentBag.ItemId < 1 {
		user.SendText(`You don't have a component bag equipped.`)
		return true, nil
	}

	moved := user.Character.SortComponentItems()

	if moved == 0 {
		user.SendText(`No crafting materials found to sort.`)
		return true, nil
	}

	user.SendText(fmt.Sprintf(
		`<ansi fg="green">You sort your materials into your %s. (%d items moved)</ansi>`,
		user.Character.Equipment.ComponentBag.DisplayName(), moved))

	return true, nil
}
```

Register the command: find where commands are registered (likely `internal/usercommands/usercommands.go`) and add the `sort` command following the existing pattern.

- [ ] **Step 4: Update crafting ingredient functions**

In `internal/crafting/crafting.go`, update `HasIngredients` to search both pools:

```go
func HasIngredients(inv []items.Item, componentInv []items.Item, recipe *RecipeSpec) (bool, string) {
	counts := make(map[string]int)
	for _, item := range componentInv {
		spec := item.GetSpec()
		if spec.ComponentTag != "" {
			counts[spec.ComponentTag]++
		}
	}
	for _, item := range inv {
		spec := item.GetSpec()
		if spec.ComponentTag != "" {
			counts[spec.ComponentTag]++
		}
	}
	for _, ing := range recipe.Ingredients {
		if counts[ing.ItemTag] < ing.Quantity {
			return false, ing.ItemTag
		}
	}
	return true, ""
}
```

Update `ConsumeIngredients` to consume from ComponentItems first:

```go
func ConsumeIngredients(inv []items.Item, componentInv []items.Item, recipe *RecipeSpec) ([]items.Item, []items.Item) {
	needed := make(map[string]int)
	for _, ing := range recipe.Ingredients {
		needed[ing.ItemTag] = ing.Quantity
	}

	// Consume from component bag first
	newComponent := make([]items.Item, 0, len(componentInv))
	for _, item := range componentInv {
		spec := item.GetSpec()
		if spec.ComponentTag != "" {
			if remaining := needed[spec.ComponentTag]; remaining > 0 {
				needed[spec.ComponentTag]--
				continue
			}
		}
		newComponent = append(newComponent, item)
	}

	// Then consume from backpack
	newInv := make([]items.Item, 0, len(inv))
	for _, item := range inv {
		spec := item.GetSpec()
		if spec.ComponentTag != "" {
			if remaining := needed[spec.ComponentTag]; remaining > 0 {
				needed[spec.ComponentTag]--
				continue
			}
		}
		newInv = append(newInv, item)
	}

	return newInv, newComponent
}
```

**IMPORTANT:** Update all callers of `HasIngredients` and `ConsumeIngredients` to pass both `user.Character.Items` and `user.Character.ComponentItems`. Search for these function calls across the codebase (they're in `craft.go` and `NewRound_UserRoundTick.go`) and update the signatures.

- [ ] **Step 5: Build and verify**

Run: `go build ./...`

- [ ] **Step 6: Commit**

```bash
git add internal/characters/character.go \
        internal/usercommands/sort.go \
        internal/crafting/crafting.go \
        internal/usercommands/usercommands.go \
        internal/usercommands/craft.go \
        internal/hooks/NewRound_UserRoundTick.go
git commit -m "$(cat <<'EOF'
feat: component bag system with auto-routing and sort command

StoreItem auto-routes is_component items to ComponentItems when a
bag is equipped. Sort command migrates existing materials. Crafting
searches both pools, consuming from component bag first.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Combat — Arms 3-4 + Penalty Formula

**Files:**
- Modify: `internal/combat/combat_helpers.go`
- Modify: `internal/combat/calculations.go`

- [ ] **Step 1: Add arms 3-4 to collectAttackWeapons**

In `internal/combat/combat_helpers.go`, find `collectAttackWeapons` (line 127). After the ExtraArm2 check (line 142-144), add:

```go
	if sourceChar.ExtraArms >= 3 && sourceChar.Equipment.ExtraArm3.ItemId > 0 && sourceChar.Equipment.ExtraArm3.GetSpec().Type == items.Weapon {
		attackWeapons = append(attackWeapons, sourceChar.Equipment.ExtraArm3)
	}
	if sourceChar.ExtraArms >= 4 && sourceChar.Equipment.ExtraArm4.ItemId > 0 && sourceChar.Equipment.ExtraArm4.GetSpec().Type == items.Weapon {
		attackWeapons = append(attackWeapons, sourceChar.Equipment.ExtraArm4)
	}
```

- [ ] **Step 2: Replace hardcoded penalty with formula**

In `calcDualWieldPenalty` (line 156), replace lines 175-180:

```go
	// Extra arm weapons get escalating additional penalties
	if weapIdx == 2 {
		penalty += 20
	} else if weapIdx >= 3 {
		penalty += 40
	}
```

With:

```go
	// Extra arm weapons get escalating penalties: +20 per arm beyond offhand
	if weapIdx >= 2 {
		penalty += (weapIdx - 1) * 20
	}
```

- [ ] **Step 3: Add arms 3-4 to PowerScore**

In `internal/combat/calculations.go`, find lines 74-80. After ExtraArm2, add:

```go
	if char.ExtraArms >= 3 && char.Equipment.ExtraArm3.ItemId > 0 {
		addWeaponDPS(char.Equipment.ExtraArm3, true)
	}
	if char.ExtraArms >= 4 && char.Equipment.ExtraArm4.ItemId > 0 {
		addWeaponDPS(char.Equipment.ExtraArm4, true)
	}
```

- [ ] **Step 4: Build and run tests**

Run: `go build ./... && go test ./internal/combat/ -v`

- [ ] **Step 5: Commit**

```bash
git add internal/combat/combat_helpers.go internal/combat/calculations.go
git commit -m "$(cat <<'EOF'
feat: combat support for extra arms 3-4 with escalating penalties

collectAttackWeapons includes arms 3-4. calcDualWieldPenalty uses
formula: +20 per arm beyond offhand (arm1 +20, arm2 +40, arm3 +60,
arm4 +80). PowerScore includes arms 3-4 DPS.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: Extra Arms Mutation Data

**Files:**
- Modify: `_datafiles/world/dogmud/mutations/extra-arms.yaml`

- [ ] **Step 1: Update mutation for levels 3-4**

Read the current `extra-arms.yaml` to understand the format. The mutation system may use per-level scaling or flat values. Update the YAML to include escalating penalties for levels 3 and 4. If the system doesn't support per-level scaling natively, the implementer should check how mutation levels map to effect magnitudes and adjust accordingly.

The target penalty curve:
- Level 1-2: Charisma -30, Aggro 1.5x, Dex +5%
- Level 3: Charisma -50, Aggro 2.0x, Dex +3%
- Level 4: Charisma -70, Aggro 2.5x, Dex +1%

If the mutation system applies effects per-level (multiplicatively), the YAML values may need to be set as per-level increments. Check `internal/mutations/` for how effects are applied and set values accordingly.

- [ ] **Step 2: Commit**

```bash
git add _datafiles/world/dogmud/mutations/extra-arms.yaml
git commit -m "$(cat <<'EOF'
feat: extra arms mutation levels 3-4 with escalating penalties

Level 3: -50 charisma, 2.0x aggro, +3% dex.
Level 4: -70 charisma, 2.5x aggro, +1% dex.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 9: Inventory Template + Display

**Files:**
- Modify: `_datafiles/world/dogmud/templates/character/inventory.template`
- Modify: `internal/usercommands/inventory.go`

- [ ] **Step 1: Update inventory template with new slots**

Replace the equipment section of `_datafiles/world/dogmud/templates/character/inventory.template`. Read the current file first (it was modified in the inventory rework). Update the equipment block to include all new slots in order:

```
{{ if not .Equipment.Weapon.IsDisabled }}   <ansi fg="yellow">Weapon:     </ansi><ansi fg="itemname">{{ .Equipment.Weapon.NameComplex }}</ansi>
{{ end -}}
{{- if not .Equipment.Offhand.IsDisabled }}   <ansi fg="yellow">Offhand:    </ansi><ansi fg="itemname">{{ .Equipment.Offhand.NameComplex }}</ansi>
{{ end -}}
{{- if not .Equipment.ExtraArm1.IsDisabled }}   <ansi fg="yellow">Arm 3:      </ansi><ansi fg="itemname">{{ .Equipment.ExtraArm1.NameComplex }}</ansi>
{{ end -}}
{{- if not .Equipment.ExtraArm2.IsDisabled }}   <ansi fg="yellow">Arm 4:      </ansi><ansi fg="itemname">{{ .Equipment.ExtraArm2.NameComplex }}</ansi>
{{ end -}}
{{- if not .Equipment.ExtraArm3.IsDisabled }}   <ansi fg="yellow">Arm 5:      </ansi><ansi fg="itemname">{{ .Equipment.ExtraArm3.NameComplex }}</ansi>
{{ end -}}
{{- if not .Equipment.ExtraArm4.IsDisabled }}   <ansi fg="yellow">Arm 6:      </ansi><ansi fg="itemname">{{ .Equipment.ExtraArm4.NameComplex }}</ansi>
{{ end -}}
{{- if not .Equipment.Head.IsDisabled }}   <ansi fg="yellow">Head:       </ansi><ansi fg="itemname">{{ .Equipment.Head.NameComplex }}</ansi>
{{ end -}}
{{- if not .Equipment.Neck.IsDisabled }}   <ansi fg="yellow">Neck:       </ansi><ansi fg="itemname">{{ .Equipment.Neck.NameComplex }}</ansi>
{{ end -}}
{{- if not .Equipment.Shoulders.IsDisabled }}   <ansi fg="yellow">Shoulders:  </ansi><ansi fg="itemname">{{ .Equipment.Shoulders.NameComplex }}</ansi>
{{ end -}}
{{- if not .Equipment.Body.IsDisabled }}   <ansi fg="yellow">Body:       </ansi><ansi fg="itemname">{{ .Equipment.Body.NameComplex }}</ansi>
{{ end -}}
{{- if not .Equipment.Back.IsDisabled }}   <ansi fg="yellow">Back:       </ansi><ansi fg="itemname">{{ .Equipment.Back.NameComplex }}</ansi>
{{ end -}}
{{- if not .Equipment.Belt.IsDisabled }}   <ansi fg="yellow">Belt:       </ansi><ansi fg="itemname">{{ .Equipment.Belt.NameComplex }}</ansi>
{{ end -}}
{{- if not .Equipment.Wrist1.IsDisabled }}   <ansi fg="yellow">Wrist:      </ansi><ansi fg="itemname">{{ .Equipment.Wrist1.NameComplex }}</ansi>
{{ end -}}
{{- if not .Equipment.Wrist2.IsDisabled }}   <ansi fg="yellow">Wrist:      </ansi><ansi fg="itemname">{{ .Equipment.Wrist2.NameComplex }}</ansi>
{{ end -}}
{{- if not .Equipment.ExtraWrist1.IsDisabled }}   <ansi fg="yellow">Wrist 3:    </ansi><ansi fg="itemname">{{ .Equipment.ExtraWrist1.NameComplex }}</ansi>
{{ end -}}
{{- if not .Equipment.ExtraWrist2.IsDisabled }}   <ansi fg="yellow">Wrist 4:    </ansi><ansi fg="itemname">{{ .Equipment.ExtraWrist2.NameComplex }}</ansi>
{{ end -}}
{{- if not .Equipment.ExtraWrist3.IsDisabled }}   <ansi fg="yellow">Wrist 5:    </ansi><ansi fg="itemname">{{ .Equipment.ExtraWrist3.NameComplex }}</ansi>
{{ end -}}
{{- if not .Equipment.ExtraWrist4.IsDisabled }}   <ansi fg="yellow">Wrist 6:    </ansi><ansi fg="itemname">{{ .Equipment.ExtraWrist4.NameComplex }}</ansi>
{{ end -}}
{{- if not .Equipment.Gloves.IsDisabled }}   <ansi fg="yellow">Gloves:     </ansi><ansi fg="itemname">{{ .Equipment.Gloves.NameComplex }}</ansi>
{{ end -}}
{{- if not .Equipment.Ring.IsDisabled }}   <ansi fg="yellow">Ring:       </ansi><ansi fg="itemname">{{ .Equipment.Ring.NameComplex }}</ansi>
{{ end -}}
{{- if not .Equipment.Ring2.IsDisabled }}   <ansi fg="yellow">Ring:       </ansi><ansi fg="itemname">{{ .Equipment.Ring2.NameComplex }}</ansi>
{{ end -}}
{{- if not .Equipment.Legs.IsDisabled }}   <ansi fg="yellow">Legs:       </ansi><ansi fg="itemname">{{ .Equipment.Legs.NameComplex }}</ansi>
{{ end -}}
{{- if not .Equipment.Feet.IsDisabled }}   <ansi fg="yellow">Feet:       </ansi><ansi fg="itemname">{{ .Equipment.Feet.NameComplex }}</ansi>
{{ end -}}
{{- if not .Equipment.ComponentBag.IsDisabled }}   <ansi fg="yellow">Components: </ansi><ansi fg="itemname">{{ .Equipment.ComponentBag.NameComplex }}</ansi>
{{ end }}
```

- [ ] **Step 2: Add component items section to inventory display**

In `internal/usercommands/inventory.go`, after the backpack stacking logic and before the `invData` map, add component item stacking using the same pattern. Pass the stacked component list to the template as `ComponentNames` and `ComponentNamesFormatted`. Update the template to render:

```
 Components: iron ingot (x3), binding paste (x2)
```

Only shown when the player has component items.

- [ ] **Step 3: Build and verify**

Run: `go build ./...`

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/templates/character/inventory.template \
        internal/usercommands/inventory.go
git commit -m "$(cat <<'EOF'
feat: inventory template with all new equipment slots + component display

Equipment section shows wrist, shoulders, back, ring2, component bag,
extra arms 3-4, extra wrists 1-4. Component bag contents rendered
as stacked list below backpack items.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 10: Content — Bracelet Fix + Component Pouch + Vendor

**Files:**
- Modify: `_datafiles/world/dogmud/items/armor-20000/ring/20035-engraved_bracelet.yaml`
- Create: component pouch item YAML
- Modify: tailor vendor YAML (weaver_maren)

- [ ] **Step 1: Fix bracelet item type**

In `_datafiles/world/dogmud/items/armor-20000/ring/20035-engraved_bracelet.yaml`, change:

```yaml
type: ring
```

to:

```yaml
type: wrist
```

Search for any other bracelet/bangle items and fix them too.

Note: The file may need to move to a `wrist/` subdirectory depending on how the engine routes by type. Check `Filepath()` behavior — if it routes by type string, the file may need to be at `armor-20000/wrist/20035-engraved_bracelet.yaml`. Verify before committing.

- [ ] **Step 2: Create starter component pouch**

Create a new item YAML for the component pouch. Find the next available item ID in the appropriate range. The pouch should have:
- `type: componentbag`
- `subtype: wearable`
- `bag_capacity: 10`
- `weight_reduction: 0.3`
- Reasonable weight (~1 lb) and value (~25 gold)
- Flavor description about a sturdy leather pouch for crafting materials

- [ ] **Step 3: Add pouch to tailor vendor stock**

In the weaver_maren mob YAML (`_datafiles/world/dogmud/mobs/thornwall_city/113-weaver_maren.yaml`), add the component pouch item ID to the shop inventory.

- [ ] **Step 4: Flag existing crafting materials as is_component**

Search `_datafiles/world/dogmud/items/materials-40000/` for crafting material items. Add `is_component: true` to each one. Common materials: iron ingot, binding paste, raw gemstone, cloth, leather, herbs, etc.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/items/ \
        _datafiles/world/dogmud/mobs/thornwall_city/113-weaver_maren.yaml
git commit -m "$(cat <<'EOF'
feat: fix bracelet type, add component pouch, flag materials

Bracelet items now type 'wrist'. Starter component pouch sold at
tailor in Thornwall (10 capacity, 30% weight reduction). Crafting
materials flagged is_component for auto-routing.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 11: Documentation + MOTD

**Files:**
- Modify: `CLAUDE.md`
- Modify: `PATCH_NOTES.md`
- Modify: `_datafiles/world/dogmud/templates/motd.template`
- Create: `_datafiles/world/dogmud/templates/help/sort.template`
- Modify: `_datafiles/world/dogmud/templates/help/equip.template`
- Modify: `_datafiles/world/dogmud/templates/help/inventory.template`

- [ ] **Step 1: Update CLAUDE.md**

Add equipment slot documentation covering all slots, mutation gating, component bag system, weight reduction mechanics.

- [ ] **Step 2: Update PATCH_NOTES.md**

Add dated entry for the equipment expansion covering new slots, component bag, extra arms 3-4, bracelet fix.

- [ ] **Step 3: Update MOTD**

Add a section explaining the new equipment slots, where to buy component bags, and the `sort` command.

- [ ] **Step 4: Create sort help file**

Create `_datafiles/world/dogmud/templates/help/sort.template` documenting the sort command.

- [ ] **Step 5: Update equip and inventory help files**

Update to reference new slots (wrist, back, shoulders, component bag) and the arm3/arm4 syntax.

- [ ] **Step 6: Commit**

```bash
git add CLAUDE.md PATCH_NOTES.md \
        _datafiles/world/dogmud/templates/motd.template \
        _datafiles/world/dogmud/templates/help/sort.template \
        _datafiles/world/dogmud/templates/help/equip.template \
        _datafiles/world/dogmud/templates/help/inventory.template
git commit -m "$(cat <<'EOF'
docs: update all documentation for equipment slot expansion

CLAUDE.md, PATCH_NOTES, MOTD, and help files for equip, inventory,
sort command.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 12: Smoke Test

- [ ] **Step 1: Full build**

Run: `go build ./...`

- [ ] **Step 2: Run tests**

Run: `go test ./...`

- [ ] **Step 3: Manual verification checklist**

1. **Wrist slot**: Equip a bracelet → goes to Wrist1. Equip another → goes to Wrist2.
2. **Ring2**: Equip two rings → first to Ring, second to Ring2.
3. **Back slot**: Equip a cloak → shows in Back slot.
4. **Shoulders**: Equip pauldrons → shows in Shoulders slot.
5. **Component bag**: Equip pouch → `is_component` items auto-sort from backpack.
6. **Sort command**: `sort` moves remaining materials into bag.
7. **Auto-route on pickup**: Pick up crafting material → goes to component bag.
8. **Crafting from bag**: Craft a recipe → consumes materials from component bag.
9. **Remove bag**: Remove component bag → contents spill to backpack.
10. **Weight reduction**: With backpack in Back slot, verify encumbrance tier changes.
11. **Extra arms 3-4**: With mutation level 3+, `equip sword arm3` works.
12. **Combat arms 3-4**: Attack with 4+ weapons, verify escalating hit penalties.
13. **Bracelet fix**: Engraved bracelet equips to wrist, not ring.
14. **Buy pouch**: Buy component pouch from tailor in Thornwall.
15. **Inventory display**: All new slots render correctly in equipment list.
