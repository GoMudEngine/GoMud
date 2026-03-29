# Equipment Slots Expansion + Component Bag — Design Spec (2026-03-29)

Adds new equipment slots (wrist, back, shoulders, ring2, component bag),
extends the Extra Arms mutation to levels 3-4 with extra wrist slots,
and introduces the component bag system for crafting material storage.

---

## 1. New Worn Struct Fields + Item Types

### New Default Slots

```go
Wrist1       items.Item `yaml:"wrist1,omitempty"`
Wrist2       items.Item `yaml:"wrist2,omitempty"`
Ring2        items.Item `yaml:"ring2,omitempty"`
Back         items.Item `yaml:"back,omitempty"`
Shoulders    items.Item `yaml:"shoulders,omitempty"`
ComponentBag items.Item `yaml:"componentbag,omitempty"`
```

### New Mutation-Gated Slots

Unlocked by Extra Arms mutation level. Each arm comes with a wrist.

```go
ExtraArm3    items.Item `yaml:"extraarm3,omitempty"` // Extra Arms level 3
ExtraArm4    items.Item `yaml:"extraarm4,omitempty"` // Extra Arms level 4
ExtraWrist1  items.Item `yaml:"extrawrist1,omitempty"` // Extra Arms level 1
ExtraWrist2  items.Item `yaml:"extrawrist2,omitempty"` // Extra Arms level 2
ExtraWrist3  items.Item `yaml:"extrawrist3,omitempty"` // Extra Arms level 3
ExtraWrist4  items.Item `yaml:"extrawrist4,omitempty"` // Extra Arms level 4
```

### New ItemType Constants

```go
Wrist        ItemType = "wrist"        // Bracelets, bracers
Back         ItemType = "back"         // Cloaks (stats) or backpacks (weight reduction)
Shoulders    ItemType = "shoulders"    // Pauldrons, mantles
ComponentBag ItemType = "componentbag" // Crafting material bags
```

### New ItemSpec Fields

```go
IsComponent     bool    `yaml:"is_component,omitempty"`     // Auto-routes to component bag on pickup
WeightReduction float64 `yaml:"weight_reduction,omitempty"` // 0.0-1.0, fraction of contents weight reduced
BagCapacity     int     `yaml:"bag_capacity,omitempty"`     // Max items storable in this bag
```

`WeightReduction` is used by Back and ComponentBag items. A cloak has 0
(stats only). A leather backpack might have 0.5. A component pouch
might have 0.3. The reduction applies to the *contents* carried by the
player, not the bag's own weight.

`BagCapacity` controls how many items the component bag can hold. If
full, new component items fall through to the regular backpack.

### Backward Compatibility

All new Worn fields use `omitempty`. Existing character YAML saves load
cleanly — new slots default to empty `Item{}` (ItemId 0).

### Files Changed

- `internal/characters/worn.go` — new fields
- `internal/items/itemspec.go` — new types, `IsComponent`, `WeightReduction`,
  `BagCapacity` fields, `ItemTypes()` list update

---

## 2. Weight Calculation Overhaul

### Current Bug

`GetCarriedWeight()` does not include ExtraArm1/2 in the weight total.

### New Weight Formula

```
1. backpackWeight = sum of all Items[] weights
2. If Back slot has WeightReduction > 0:
      backpackWeight *= (1.0 - Back.GetSpec().WeightReduction)
3. componentWeight = sum of all ComponentItems[] weights
4. If ComponentBag slot has WeightReduction > 0:
      componentWeight *= (1.0 - ComponentBag.GetSpec().WeightReduction)
5. equippedWeight = sum of ALL equipped slot weights
   (includes all new slots + ExtraArm1-4 + ExtraWrist1-4)
6. Total = backpackWeight + componentWeight + equippedWeight
```

Weight reduction applies only to contents, not to the bag item itself
(the bag's own weight always counts at full).

### Files Changed

- `internal/characters/character.go` — `GetCarriedWeight()` overhaul

---

## 3. Equip Routing Updates

### Wear() Slot Routing

| ItemType | Slot(s) | Routing Logic |
|----------|---------|---------------|
| `wrist` | Wrist1 → Wrist2 → ExtraWrist1-4 | Fill first empty. Extra wrists gated by ExtraArms level. |
| `ring` | Ring → Ring2 | Fill first empty. Existing Ring routing unchanged, overflow to Ring2. |
| `back` | Back | Single slot, displaces existing. |
| `shoulders` | Shoulders | Single slot, displaces existing. |
| `componentbag` | ComponentBag | Single slot, displaces existing. On equip: auto-migrate `is_component` items from backpack into bag. |

### Extra Arm Routing

Raise cap from 2 to 4. When ExtraArm1/2 are occupied and
`ExtraArms >= 3`, overflow to ExtraArm3, then ExtraArm4. Each extra arm
level also unlocks the corresponding ExtraWrist slot.

| ExtraArms Level | Arm Slots Unlocked | Wrist Slots Unlocked |
|-----------------|-------------------|---------------------|
| 1 | ExtraArm1 | ExtraWrist1 |
| 2 | ExtraArm2 | ExtraWrist2 |
| 3 | ExtraArm3 | ExtraWrist3 |
| 4 | ExtraArm4 | ExtraWrist4 |

### Equip Command Suffix

Extend `equip <item> arm3` / `equip <item> arm4` syntax in `equip.go`.

### Remove/Unequip

Add all new slots to `FindOnBody()` and `RemoveFromBody()` search
lists. When removing a ComponentBag, all `ComponentItems` spill back
into the main backpack `Items`.

### Disabled Slot System

Species `DisabledSlots` gets new valid entries: `"wrist1"`, `"wrist2"`,
`"ring2"`, `"back"`, `"shoulders"`, `"componentbag"`.

### Files Changed

- `internal/characters/character.go` — `Wear()`, `FindOnBody()`,
  `RemoveFromBody()`, disabled slot handling, ExtraArms cap raise
- `internal/usercommands/equip.go` — arm3/arm4 suffix parsing

---

## 4. Component Bag System

### Storage

New field on Character:

```go
ComponentItems []items.Item `yaml:"componentitems,omitempty"`
```

Separate slice from `Items` (backpack). The ComponentBag equipment slot
holds the bag *item*. `ComponentItems` holds the bag's *contents*.

### Auto-Routing on Pickup/Buy

When `StoreItem()` is called:

1. Is the item flagged `is_component: true`?
2. Is a ComponentBag equipped?
3. Is the bag not full (`len(ComponentItems) < BagCapacity`)?
4. If all yes → append to `ComponentItems` instead of `Items`.
5. Otherwise → regular backpack.

### Crafting Integration

`crafting.HasIngredients()` and `crafting.ConsumeIngredients()` search
both `Items` and `ComponentItems`. Consume from `ComponentItems` first
(use materials from the bag before the backpack).

### Sort Command

New `sort` command moves all `is_component` items from backpack into
the component bag (up to capacity). Reports what was moved:

```
You sort your materials into your component pouch.
Moved: iron ingot (x3), binding paste (x2), raw gemstone.
```

### Equipping a New Bag

When a ComponentBag is equipped via `Wear()`, auto-run the sort logic
(move `is_component` items from backpack into bag).

### Removing the Bag

When the ComponentBag is removed, all `ComponentItems` spill back into
the regular backpack `Items`.

### Inventory Display

Component bag contents show in a separate section:

```
 Carrying: keen dagger, healing potion          Encumbrance: [light]
 Components: iron ingot (x3), binding paste (x2)
```

Uses the same stacking display logic as the main inventory.

### Files Changed

- `internal/characters/character.go` — `ComponentItems` field,
  `StoreItem()` auto-routing
- `internal/items/itemspec.go` — `IsComponent`, `BagCapacity` fields
- `internal/crafting/crafting.go` — search both item pools
- `internal/usercommands/sort.go` — new command
- `internal/usercommands/inventory.go` — component section display
- `_datafiles/world/dogmud/templates/character/inventory.template`

---

## 5. Extra Arms Mutation — Levels 3-4

### Cap Raise

In `character.go` where `ExtraArms` is derived from mutation level,
raise the cap from 2 to 4.

### Escalating Penalties

| Level | Slots Unlocked | Charisma | Aggro Magnet | Dex Bonus |
|-------|---------------|----------|--------------|-----------|
| 1 | Arm1 + Wrist1 | -30 | 1.5x | +5% |
| 2 | Arm2 + Wrist2 | -30 | 1.5x | +5% |
| 3 | Arm3 + Wrist3 | -50 | 2.0x | +3% |
| 4 | Arm4 + Wrist4 | -70 | 2.5x | +1% |

Levels 3-4 are dramatically more monstrous. The charisma hit makes
social interactions painful and the aggro magnet makes you a priority
target. Dexterity bonus diminishes because coordinating 6-8 arms is
harder.

Configured in mutation YAML data. Check whether the mutation system
supports per-level scaling or if engine changes are needed.

### Files Changed

- `internal/characters/character.go` — ExtraArms cap 2 → 4
- `_datafiles/world/dogmud/mutations/extra-arms.yaml` — level 3-4 data

---

## 6. Combat Integration for Arms 3-4

### collectAttackWeapons()

Add ExtraArm3/4 to the weapon list (gated by `ExtraArms >= 3/4`).

### calcDualWieldPenalty()

Replace hardcoded if-else with formula. Each arm beyond offhand gets
+20 additional penalty:

```go
if weapIdx >= 2 {
    penalty += (weapIdx - 1) * 20
}
```

| weapIdx | Slot | Extra Penalty |
|---------|------|--------------|
| 0 | Main hand | +0 |
| 1 | Offhand | +base (skill-reduced) |
| 2 | Extra arm 1 | +base + 20 |
| 3 | Extra arm 2 | +base + 40 |
| 4 | Extra arm 3 | +base + 60 |
| 5 | Extra arm 4 | +base + 80 |

### PowerScore()

Add ExtraArm3/4 to the DPS estimate in `calculations.go`.

### Files Changed

- `internal/combat/combat_helpers.go` — `collectAttackWeapons()`,
  `calcDualWieldPenalty()`
- `internal/combat/calculations.go` — `PowerScore()`

---

## 7. Template, Content, and Documentation

### Inventory Template

Add new slots to equipment display. Slot ordering:

```
 Weapon:      ...
 Offhand:     ...
 Arm 3-6:     ...    (mutation-gated, only shown when unlocked)
 Head:        ...
 Neck:        ...
 Shoulders:   ...    (NEW)
 Body:        ...
 Back:        ...    (NEW)
 Belt:        ...
 Wrist:       ...    (NEW, two slots, same label)
 Wrist:       ...    (NEW)
 Wrist 3-6:   ...    (mutation-gated, only shown when unlocked)
 Gloves:      ...
 Ring:        ...
 Ring:        ...    (NEW, second ring)
 Legs:        ...
 Feet:        ...
 Components:  ...    (NEW)
```

### Item Content

- Create starter component pouch item (sold at tailor in Thornwall):
  small capacity (~10 items), 30% weight reduction
- Update existing bracelet item YAMLs: change type from `ring` to `wrist`
- Update tailor vendor stock to include the component pouch

### Documentation

- **MOTD update** — explain new equipment slots, where to buy component
  bags, the `sort` command
- **CLAUDE.md** — add Equipment Slots section documenting all slots,
  mutation gating, component bag system, weight reduction mechanics
- **PATCH_NOTES.md** — dated entry for the equipment expansion
- **Help files** — update `help equip`, `help inventory`; create
  `help sort`, `help component-bag`

### Files Changed

- `_datafiles/world/dogmud/templates/character/inventory.template`
- `internal/items/itemspec.go` — `ItemTypes()` with ID ranges
- Bracelet item YAMLs — type `ring` → `wrist`
- New component pouch item YAML
- Tailor vendor stock YAML
- `_datafiles/world/dogmud/templates/motd.template`
- `CLAUDE.md`
- `PATCH_NOTES.md`
- Help templates: equip, inventory, sort (new), component-bag (new)

---

## Summary of All Files Changed

| File | Sections | Change |
|------|----------|--------|
| `internal/characters/worn.go` | 1 | 12 new slot fields, update all methods |
| `internal/items/itemspec.go` | 1, 4 | New types, IsComponent, WeightReduction, BagCapacity |
| `internal/characters/character.go` | 2, 3, 4, 5 | Weight calc, Wear(), FindOnBody(), RemoveFromBody(), ExtraArms cap, ComponentItems, StoreItem() |
| `internal/usercommands/equip.go` | 3 | arm3/arm4 suffix parsing |
| `internal/crafting/crafting.go` | 4 | Search both item pools |
| `internal/usercommands/sort.go` | 4 | New command |
| `internal/usercommands/inventory.go` | 4 | Component section display |
| `_datafiles/.../inventory.template` | 7 | New slot rendering |
| `internal/combat/combat_helpers.go` | 6 | Arms 3-4 weapons + penalty formula |
| `internal/combat/calculations.go` | 6 | Arms 3-4 in PowerScore |
| `_datafiles/.../mutations/extra-arms.yaml` | 5 | Levels 3-4 penalties |
| `_datafiles/.../items/` | 7 | Component pouch, bracelet type fix |
| Tailor vendor YAML | 7 | Stock update |
| MOTD, CLAUDE.md, PATCH_NOTES.md | 7 | Documentation |
| Help templates | 7 | equip, inventory, sort, component-bag |
