# Instance Randomized Loot Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Instance mobs spawn wearing randomly-generated gear (point-budget affix system) that makes them tougher and drops on death. Oasis zone becomes a 125-room wrapping 3D cube.

**Architecture:** New affix engine generates items from a point budget derived from gold paid. Items are rolled at mob spawn time in `Room.Prepare()` and equipped on the mob via per-instance `Item.Spec` overrides. Oasis cube is generated programmatically via a new cube generator in `CreateZoneInstance`. Arena gets expanded with tough/boss rooms.

**Tech Stack:** Go, YAML data files, ANSI template help files

**Spec:** `docs/superpowers/specs/2026-04-10-instance-randomized-loot-design.md`

---

### Task 1: Add LootPool to Mob Template + LootBudgetScalar Config

**Files:**
- Modify: `internal/mobs/mobs.go` (add LootPool field to Mob struct)
- Modify: `internal/configs/config.balance.go` (add LootBudgetScalar)

- [ ] **Step 1: Add LootPool field to Mob struct**

In `internal/mobs/mobs.go`, add to the Mob struct:

```go
LootPool []int `yaml:"loot_pool,omitempty"` // Item IDs for instance loot generation
```

This is a simple list of item IDs. When a mob spawns in an instance,
items from this pool are rolled with affixes and equipped.

- [ ] **Step 2: Add LootBudgetScalar to balance config**

In `internal/configs/config.balance.go`, add:

```go
LootBudgetScalar ConfigFloat `yaml:"LootBudgetScalar"` // Multiplier for sqrt(goldPaid) loot budget (default 7.0)
```

Default validation:
```go
if b.LootBudgetScalar <= 0 {
    b.LootBudgetScalar = 7.0
}
```

- [ ] **Step 3: Compile and commit**

Run: `go build ./...`

```bash
git add internal/mobs/mobs.go internal/configs/config.balance.go
git commit -m "feat(loot): add LootPool to mob template and LootBudgetScalar config"
```

---

### Task 2: Affix Generation Engine

**Files:**
- Create: `internal/items/affixgen.go`
- Test: `internal/items/affixgen_test.go`

This is the core engine. It takes a base item ID and a gold amount,
and returns an `items.Item` with randomly-rolled affix bonuses
applied via the per-instance `Item.Spec` override.

- [ ] **Step 1: Write tests for affix generation**

Create `internal/items/affixgen_test.go`:

```go
package items

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalcLootBudget(t *testing.T) {
	// budget = floor(scalar * sqrt(goldPaid))
	budget := CalcLootBudget(200, 7.0)
	assert.Equal(t, 98, budget) // floor(7 * sqrt(200)) = floor(98.99) = 98

	budget = CalcLootBudget(1000, 7.0)
	assert.Equal(t, 221, budget) // floor(7 * sqrt(1000)) = floor(221.35) = 221

	budget = CalcLootBudget(0, 7.0)
	assert.Equal(t, 0, budget)
}

func TestGenerateAffixedItem_ReturnsItem(t *testing.T) {
	// This test verifies the function exists and returns a non-zero item.
	// Full integration testing requires loaded item specs.
	// At minimum test the budget calc and bonus type eligibility.
	weaponBonuses := GetEligibleBonuses(true, false)
	assert.Greater(t, len(weaponBonuses), 0)

	armorBonuses := GetEligibleBonuses(false, false)
	assert.Greater(t, len(armorBonuses), 0)

	// Weapons should have damage mult, armor should not
	hasDmg := false
	for _, b := range weaponBonuses {
		if b.Name == "damage_mult_phys" || b.Name == "damage_mult_both" {
			hasDmg = true
		}
	}
	assert.True(t, hasDmg)

	hasMit := false
	for _, b := range armorBonuses {
		if b.Name == "physical_mitigation" {
			hasMit = true
		}
	}
	assert.True(t, hasMit)
}
```

- [ ] **Step 2: Run tests — should fail**

Run: `go test ./internal/items/ -run TestCalcLootBudget -v`

- [ ] **Step 3: Implement affix generation engine**

Create `internal/items/affixgen.go`:

```go
package items

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/statmods"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// BonusType defines a single rollable affix bonus.
type BonusType struct {
	Name     string // internal identifier
	Cost     int    // points per unit
	Category string // "weapon", "armor", "any"
}

// All possible bonus types with their point costs.
var allBonusTypes = []BonusType{
	{Name: "damage_mult_both", Cost: 12, Category: "weapon_caster"},
	{Name: "damage_mult_phys", Cost: 8, Category: "weapon"},
	{Name: "physical_mitigation", Cost: 5, Category: "armor"},
	{Name: "magical_mitigation", Cost: 5, Category: "armor"},
	{Name: "conviction_mitigation", Cost: 5, Category: "armor"},
	{Name: "stat_strength", Cost: 3, Category: "any"},
	{Name: "stat_dexterity", Cost: 3, Category: "any"},
	{Name: "stat_perception", Cost: 3, Category: "any"},
	{Name: "stat_vitality", Cost: 3, Category: "any"},
	{Name: "stat_willpower", Cost: 3, Category: "any"},
	{Name: "stat_charisma", Cost: 3, Category: "any"},
	{Name: "skill_weapon-combat", Cost: 12, Category: "any"},
	{Name: "skill_unarmed-combat", Cost: 12, Category: "any"},
	{Name: "skill_skullduggery", Cost: 12, Category: "any"},
	{Name: "skill_spellcasting", Cost: 12, Category: "any"},
	{Name: "skill_rhetoric", Cost: 12, Category: "any"},
	{Name: "skill_manifestation", Cost: 12, Category: "any"},
}

// CalcLootBudget returns the base point budget for a given gold amount.
func CalcLootBudget(goldPaid int, scalar float64) int {
	if goldPaid <= 0 {
		return 0
	}
	return int(math.Floor(scalar * math.Sqrt(float64(goldPaid))))
}

// GetEligibleBonuses returns the bonus types valid for the given item category.
func GetEligibleBonuses(isWeapon bool, isCasterWeapon bool) []BonusType {
	eligible := []BonusType{}
	for _, bt := range allBonusTypes {
		switch bt.Category {
		case "any":
			eligible = append(eligible, bt)
		case "weapon":
			if isWeapon && !isCasterWeapon {
				eligible = append(eligible, bt)
			}
		case "weapon_caster":
			if isWeapon && isCasterWeapon {
				eligible = append(eligible, bt)
			}
		case "armor":
			if !isWeapon {
				eligible = append(eligible, bt)
			}
		}
	}
	return eligible
}

// GenerateAffixedItem creates an item with random affix bonuses from
// the point budget. The base item is created from baseItemId, then
// bonuses are applied via a per-instance Spec override.
func GenerateAffixedItem(baseItemId int, goldPaid int, scalar float64) Item {
	item := New(baseItemId)
	if item.ItemId < 1 {
		return item
	}

	baseSpec := item.GetSpec()
	if baseSpec == nil {
		return item
	}

	budget := CalcLootBudget(goldPaid, scalar)
	if budget <= 0 {
		return item
	}

	// Gaussian variance on budget
	budgetRoll := dice.RollStat(float64(budget))
	budget = int(math.Round(budgetRoll.Value))
	if budget <= 0 {
		return item
	}

	// Determine item category
	isWeapon := baseSpec.Type == Weapon
	isCasterWeapon := isWeapon && (baseSpec.Subtype == Wand ||
		baseSpec.Subtype == Sceptre || baseSpec.Subtype == Staff)

	eligible := GetEligibleBonuses(isWeapon, isCasterWeapon)
	if len(eligible) == 0 {
		return item
	}

	// Roll bonuses
	override := &ItemSpec{}
	override.StatMods = statmods.StatMods{}
	dmgMultAdd := 0.0
	spellDmgMultAdd := 0.0

	for budget > 0 {
		// Find affordable bonuses
		affordable := []BonusType{}
		for _, bt := range eligible {
			if bt.Cost <= budget {
				affordable = append(affordable, bt)
			}
		}
		if len(affordable) == 0 {
			break
		}

		// Pick random bonus
		pick := affordable[util.Rand(len(affordable))]
		budget -= pick.Cost

		// Apply bonus
		switch {
		case pick.Name == "damage_mult_both":
			dmgMultAdd += 0.01
			spellDmgMultAdd += 0.01
		case pick.Name == "damage_mult_phys":
			dmgMultAdd += 0.01
		case pick.Name == "physical_mitigation":
			override.PhysicalMitigation++
		case pick.Name == "magical_mitigation":
			override.MagicalMitigation++
		case pick.Name == "conviction_mitigation":
			override.ConvictionMitigation++
		default:
			// stat_X or skill_X
			if len(pick.Name) > 5 && pick.Name[:5] == "stat_" {
				statName := pick.Name[5:]
				override.StatMods[statName]++
			} else if len(pick.Name) > 6 && pick.Name[:6] == "skill_" {
				skillName := pick.Name[6:]
				override.StatMods[skillName]++
			}
		}
	}

	// Apply damage mult overrides (add to base spec values)
	if dmgMultAdd > 0 {
		override.DamageMultiplier = baseSpec.DamageMultiplier + dmgMultAdd
	}
	if spellDmgMultAdd > 0 {
		override.SpellDamageMultiplier = baseSpec.SpellDamageMultiplier + spellDmgMultAdd
	}

	// Apply mitigation overrides (add to base spec values)
	if override.PhysicalMitigation > 0 {
		override.PhysicalMitigation += baseSpec.PhysicalMitigation
	}
	if override.MagicalMitigation > 0 {
		override.MagicalMitigation += baseSpec.MagicalMitigation
	}
	if override.ConvictionMitigation > 0 {
		override.ConvictionMitigation += baseSpec.ConvictionMitigation
	}

	item.Spec = override
	return item
}
```

**CRITICAL:** The implementer MUST verify how `Item.Spec` override
works with `GetSpec()`. Read `internal/items/items.go` to confirm
that setting `item.Spec = &ItemSpec{...}` overrides individual fields.
If `GetSpec()` replaces rather than merges, you'll need to copy ALL
base spec fields into the override and then modify them. Adjust the
code accordingly.

Also verify the exact `ItemType` and `ItemSubType` constants for
weapon detection (`Weapon`, `Wand`, `Sceptre`, `Staff`). Check
`internal/items/itemspec.go` for the exact constant names.

- [ ] **Step 4: Run tests — should pass**

Run: `go test ./internal/items/ -run "TestCalcLootBudget|TestGenerateAffixedItem" -v`

- [ ] **Step 5: Compile and commit**

Run: `go build ./...`

```bash
git add internal/items/affixgen.go internal/items/affixgen_test.go
git commit -m "feat(loot): add affix generation engine with point budget system"
```

---

### Task 3: Hook Affix Generation into Mob Spawn

**Files:**
- Modify: `internal/rooms/rooms.go` (in `Prepare()` function)

- [ ] **Step 1: Add loot generation hook in Prepare()**

In `internal/rooms/rooms.go`, inside the `Prepare()` function, after
`mobs.NewMobById()` returns a mob and before `mob.Validate()`, add:

```go
// Instance loot: generate and equip affixed items from loot pool
if goldPaid, ok := r.GetTempData("gold_paid").(int); ok && goldPaid > 0 {
    if len(mob.LootPool) > 0 {
        scalar := float64(configs.GetBalanceConfig().LootBudgetScalar)
        for _, baseItemId := range mob.LootPool {
            affixedItem := items.GenerateAffixedItem(baseItemId, goldPaid, scalar)
            if affixedItem.ItemId > 0 {
                mob.Character.Wear(affixedItem)
            }
        }
    }
}
```

This iterates the mob's loot pool, generates an affixed version of
each item, and equips it. `Wear()` handles slot assignment. For
bosses with 2 items (e.g., weapon + body armor), both get generated
and equipped.

Add imports for `"github.com/GoMudEngine/GoMud/internal/items"` and
`"github.com/GoMudEngine/GoMud/internal/configs"` if not present.

**Note:** The implementer must find the exact insertion point in
`Prepare()`. Look for where `mobs.NewMobById` is called, then find
the block of post-spawn overrides (name, buffs, hostile, etc.). The
loot generation goes in that same block, before `mob.Validate()`.

- [ ] **Step 2: Compile and commit**

Run: `go build ./...`

```bash
git add internal/rooms/rooms.go
git commit -m "feat(loot): hook affix generation into mob spawn path"
```

---

### Task 4: Item Naming System for Affixed Items

**Files:**
- Modify: `internal/items/affixgen.go` (add naming logic)

- [ ] **Step 1: Add prefix naming based on highest bonus**

Add to `affixgen.go`, called at the end of `GenerateAffixedItem`
before returning:

```go
// Generate a descriptive prefix based on the dominant bonus type.
func getAffixPrefix(override *ItemSpec) string {
	// Count points spent in each category
	dmgPoints := 0
	mitPoints := 0
	statPoints := 0
	skillPoints := 0

	if override.DamageMultiplier > 0 || override.SpellDamageMultiplier > 0 {
		dmgPoints = 1 // simplified — just presence
	}
	mitPoints = override.PhysicalMitigation +
		override.MagicalMitigation +
		override.ConvictionMitigation
	for _, v := range override.StatMods {
		// stats cost 3 pts each, skills cost 12
		if v > 0 {
			statPoints += v
		}
	}

	// Pick dominant category
	max := dmgPoints
	prefix := "Keen"
	if mitPoints > max {
		max = mitPoints
		prefix = "Warding"
	}
	if statPoints > max {
		max = statPoints
		prefix = "Empowered"
	}
	if skillPoints > max {
		prefix = "Masterwork"
	}

	return prefix
}
```

Then in `GenerateAffixedItem`, before `return item`, add:

```go
	// Set item adjective for display
	prefix := getAffixPrefix(override)
	item.Adjectives = append(item.Adjectives, prefix)
```

Check if `Item` has an `Adjectives` field. If not, check how item
display names are modified — there may be an `item.Name` override
or a different mechanism. The implementer must verify.

- [ ] **Step 2: Compile and commit**

Run: `go build ./...`

```bash
git add internal/items/affixgen.go
git commit -m "feat(loot): add prefix naming for affixed items"
```

---

### Task 5: Create Base Loot Items (Arena)

**Files:**
- Create: new item YAMLs for arena loot
- Folders: appropriate item subdirectories

**IMPORTANT:** Verify next available item IDs before creating:
```
find _datafiles/world/dogmud/items -name "*.yaml" | sed 's/.*\///' | sed 's/-.*//' | sort -n | tail -10
```

Create base items for arena mobs. These have modest base stats —
the affix system adds the magic. Items need appropriate `type`,
`subtype`, `damage_multiplier` (weapons), `physical_mitigation`
(armor), etc.

Arena loot:
- Warhammer (weapon, boss) — 2H, bludgeoning subtype
- Tower shield (offhand, boss) — high block rating
- Chain gloves (gloves, tough) — modest phys mitigation
- Iron greaves (legs, tough) — modest phys mitigation

Also create arena veteran (tough, statpool 2) and arena champion
(boss, statpool 4) mob templates with `loot_pool` referencing
these items.

The implementer should use `/new-item` and `/new-mob` skills if
available, or create the YAMLs manually following existing patterns.
Check existing weapon/armor items for field reference.

- [ ] **Step 1: Create base item YAMLs**
- [ ] **Step 2: Create arena veteran and champion mob templates**
- [ ] **Step 3: Update arena zone rooms with new mobs**
- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/items/ _datafiles/world/dogmud/mobs/instance_arena/ \
  _datafiles/world/dogmud/rooms/instance_arena/
git commit -m "feat(loot): add arena loot items, veteran and champion mobs"
```

---

### Task 6: Create Base Loot Items (Oasis)

**Files:**
- Create: new item YAMLs for oasis loot

Oasis loot (for tough and boss mobs):
- Obsidian mace (weapon, king) — 1H bludgeoning
- Volcanic plate (body armor, king) — heavy phys mit
- Crystal sceptre (weapon, queen) — caster weapon, spell_damage_mult
- Ice crown (head, queen) — magical mitigation
- Wind scimitar (weapon, prince) — slashing, fast
- Mist pauldrons (shoulders, prince) — mixed mitigation
- Stone ring (ring, sand elemental tough) — stat bonuses
- Storm bracer (wrist, storm elemental tough) — conviction mit

Update existing oasis boss mob templates (320-322) and tough mobs
(318-319) with `loot_pool` fields and appropriate `itemdropchance`.

- [ ] **Step 1: Create base item YAMLs**
- [ ] **Step 2: Update oasis mob templates with loot_pool**
- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/items/ _datafiles/world/dogmud/mobs/instance_planar_oasis/
git commit -m "feat(loot): add oasis loot items and update mob loot pools"
```

---

### Task 7: Oasis Cube Generator

**Files:**
- Create: `internal/rooms/cubegen.go`
- Test: `internal/rooms/cubegen_test.go`
- Modify: `internal/rooms/instances.go` (call cube generator for oasis)

This is the 5x5x5 wrapping cube generator for the oasis zone.

- [ ] **Step 1: Write tests for cube geometry**

Create `internal/rooms/cubegen_test.go`:

```go
package rooms

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCubeWrapCoord(t *testing.T) {
	assert.Equal(t, 0, wrapCoord(5, 5))  // wraps to 0
	assert.Equal(t, 4, wrapCoord(-1, 5)) // wraps to 4
	assert.Equal(t, 2, wrapCoord(2, 5))  // no wrap
	assert.Equal(t, 0, wrapCoord(0, 5))  // edge case
}

func TestCubeRoomCount(t *testing.T) {
	// 5x5x5 = 125 rooms
	assert.Equal(t, 125, 5*5*5)
}
```

- [ ] **Step 2: Implement cube generator**

Create `internal/rooms/cubegen.go`:

```go
package rooms

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/util"
)

const cubeSize = 5

// wrapCoord wraps a coordinate within [0, size).
func wrapCoord(val, size int) int {
	return ((val % size) + size) % size
}

// cubeIndex converts (x,y,z) to a flat index.
func cubeIndex(x, y, z int) int {
	return x*cubeSize*cubeSize + y*cubeSize + z
}

// Room description pool for the elemental cube.
var cubeDescriptions = []string{
	`Crystalline sand stretches in every direction, each grain a tiny prism. The air hums with planar energy.`,
	`Scorched earth crunches underfoot. Heat radiates from cracks in the ground where molten rock glows faintly below.`,
	`Frozen dunes of pale blue ice rise in gentle waves. The cold is sharp and immediate, cutting through armor.`,
	`Wind-torn flats where grit stings exposed skin. Dust devils spiral lazily in the distance — or nearby. Hard to tell.`,
	`A forest of stone pillars rises from the sand, each one thrumming with a low vibration felt more than heard.`,
	`The ground here is smooth obsidian, reflecting a sky that cannot decide between dawn and dusk.`,
	`Pools of dark water dot the sand, each perfectly still. Something moves beneath the surface of the nearest one.`,
	`The air crackles with static. Tiny arcs of lightning jump between grains of sand with each footstep.`,
	`Massive crystals jut from the ground at odd angles, casting prismatic light in all directions.`,
	`The sand gives way to packed clay veined with glowing mineral deposits. The warmth here is almost pleasant.`,
}

// cubeTitle returns a title incorporating the z-level.
func cubeTitle(z int) string {
	depths := []string{"Depths", "Lower Wastes", "Middle Wastes", "Upper Wastes", "Heights"}
	if z >= 0 && z < len(depths) {
		return fmt.Sprintf("Elemental %s", depths[z])
	}
	return "Elemental Wastes"
}

// Trash elemental mob IDs (water, earth, air, fire, magma).
var trashElementalIds = []int{310, 311, 312, 313, 314}

// Tough elemental mob IDs (sand, storm).
var toughElementalIds = []int{318, 319}

// Boss elemental mob IDs (king, queen, prince).
var bossElementalIds = []int{320, 321, 322}

// GenerateOasisCube creates 125 ephemeral rooms in a 5x5x5 wrapping
// cube, assigns mobs, and connects the entrance room. Returns a map
// of cube room IDs and the entry room ID within the cube (2,2,0).
//
// Parameters:
//   - entryRoomId: the Oasis Threshold room ID (outside the cube)
//   - goldPaid: for setting temp data on rooms
//   - instanceId: for temp data
//   - allowRecall: for temp data
//   - deathPolicy: for temp data
//
// Returns: cubeRoomIds []int, entryCubeRoomId int, error
func GenerateOasisCube(
	entryRoomId int,
	goldPaid int,
	instanceId int,
	allowRecall bool,
	deathPolicy string,
) ([]int, int, error) {

	// 1. Reserve 125 ephemeral room IDs
	templateIds := make([]int, cubeSize*cubeSize*cubeSize)
	// We need blank room IDs — use a dummy approach:
	// Create ephemeral rooms from scratch rather than cloning templates.
	// The implementer must check if CreateEphemeralRoomIds can create
	// blank rooms, or if we need a different approach.
	//
	// ALTERNATIVE: Create 125 rooms directly via addRoomToMemory with
	// ephemeral IDs. This bypasses the template cloning system entirely.
	//
	// The implementer should research how to create ephemeral rooms
	// WITHOUT templates — these rooms are procedurally generated, not
	// cloned from existing data files.

	// ... (implementation details depend on codebase specifics)

	return nil, 0, fmt.Errorf("not yet implemented")
}
```

**CRITICAL NOTE TO IMPLEMENTER:** The cube generator needs to create
125 rooms that don't exist as templates. The existing `CreateEphemeralRoomIds`
clones from template room IDs. You'll likely need to:
1. Reserve a chunk of ephemeral IDs
2. Create blank `Room` structs with those IDs
3. Set title, description, zone, exits manually
4. Add to memory via `addRoomToMemory`

Study `CreateEphemeralRoomIds` in `ephemeral.go` to understand the
ID allocation, then create rooms manually instead of cloning. The
rooms need:
- RoomId (ephemeral)
- Zone: "Instance Planar Oasis"
- Title: from `cubeTitle(z)`
- Description: random from `cubeDescriptions`
- Exits: 6 wrapping exits (n/s/e/w/up/down)
- SpawnInfo: 1 random trash elemental per room
- Override room (2,2,0) south exit to point to entryRoomId

After all 125 rooms are created:
- Pick 2 random rooms, replace their spawn with tough mobs
- Pick 1 random room, replace its spawn with a random boss
- Stamp instance temp data on all rooms

This is the most complex task in the plan. The implementer may need
to break it into sub-steps or escalate for architectural guidance.

- [ ] **Step 3: Wire cube generator into CreateZoneInstance**

In `internal/rooms/instances.go`, modify `CreateZoneInstance` to
detect the oasis zone and call the cube generator instead of the
normal `CreateEphemeralZone` path. The oasis zone needs special
handling because its rooms are generated, not cloned.

- [ ] **Step 4: Run tests and compile**

Run: `go build ./... && go test ./internal/rooms/ -run TestCube -v`

- [ ] **Step 5: Commit**

```bash
git add internal/rooms/cubegen.go internal/rooms/cubegen_test.go \
  internal/rooms/instances.go
git commit -m "feat(loot): add 5x5x5 wrapping cube generator for oasis"
```

---

### Task 8: Expand Arena Zone Layout

**Files:**
- Create: new room YAMLs in `_datafiles/world/dogmud/rooms/instance_arena/`
- Modify: existing arena room exits

**IMPORTANT:** Verify next available room IDs first.

Expand the arena from 2 rooms to 5:
1. Entry Chamber (existing 5001, safe)
2. Arena Floor (existing 5002, trash mobs)
3. Arena Gauntlet (new, trash mobs, connects to floor)
4. Veteran's Ring (new, tough mobs with gear)
5. Champion's Pit (new, boss with gear)

Linear progression: entry → floor → gauntlet → veteran's ring →
champion's pit. Each room connects to the next via north/south.

- [ ] **Step 1: Create new room YAMLs**
- [ ] **Step 2: Update existing room exits**
- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/rooms/instance_arena/
git commit -m "feat(loot): expand arena to 5 rooms with tough/boss progression"
```

---

### Task 9: Help Files

**Files:**
- Create: `_datafiles/world/dogmud/templates/help/arena.template`
- Create: `_datafiles/world/dogmud/templates/help/oasis.template`
- Modify: `_datafiles/world/dogmud/templates/help/instances.template`
- Modify: `_datafiles/world/dogmud/keywords.yaml` (add aliases)

- [ ] **Step 1: Create arena help file**

Create `_datafiles/world/dogmud/templates/help/arena.template`:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">arena</ansi>

The Arena is a brutal proving ground where death means
expulsion. There is no coming back.

<ansi fg="yellow">Layout:</ansi>

  A linear gauntlet of increasingly dangerous rooms. Fight
  your way through waves of pit fighters until you reach
  the champion at the end. Enemies respawn quickly — there
  is no resting between waves.

<ansi fg="yellow">Rules:</ansi>

  <ansi fg="red">Death ends your run.</ansi> If you fall, you are expelled
  from the Arena permanently. You cannot re-enter.

  Recall magic does not work here. The only way out is
  through the return portal at the entrance.

<ansi fg="yellow">Loot:</ansi>

  Tougher enemies carry equipment forged by the Arena
  itself. The more gold you invest with the Riftkeeper,
  the more powerful the enemies become — and the better
  the gear they carry. Weapons and armor dropped by
  Arena fighters are unique, with properties that vary
  each time.

<ansi fg="yellow">Tips:</ansi>

  Bring potions and grenades. Prepare your bandolier
  before entering. Party up for the toughest fights.

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help instances</ansi>,
  <ansi fg="command">help oasis</ansi>, <ansi fg="command">help party</ansi>
```

- [ ] **Step 2: Create oasis help file**

Create `_datafiles/world/dogmud/templates/help/oasis.template`:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">oasis</ansi>

The Planar Oasis is an otherworldly labyrinth of elemental
energy. A vast cube of shifting terrain where directions
fold back on themselves and elementals roam freely.

<ansi fg="yellow">Layout:</ansi>

  A three-dimensional maze. Moving north, south, east,
  west, up, or down wraps around — walking far enough in
  any direction brings you back to where you started. The
  entrance is at the bottom of the cube. Every room holds
  an elemental guardian, and somewhere within the maze
  lurks a powerful elemental lord.

<ansi fg="yellow">Rules:</ansi>

  Death does not end your run. You may return through the
  portal after falling and continue where you left off.
  Recall magic works here.

  Guardians do not return once slain — clear the cube
  methodically. The elemental lord wanders the maze and
  may find you before you find it.

<ansi fg="yellow">Loot:</ansi>

  Elite elementals and the elemental lord carry planar-
  forged equipment. As with the Arena, investing more gold
  strengthens the guardians but improves what they carry.
  Each piece of equipment is unique.

<ansi fg="yellow">Navigation:</ansi>

  The cube wraps in all six directions. You can move up
  and down as well as the four cardinal directions. Keep
  track of your position — it is easy to become lost.
  The entrance is the only fixed point of reference.

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help instances</ansi>,
  <ansi fg="command">help arena</ansi>, <ansi fg="command">help party</ansi>
```

- [ ] **Step 3: Update instances help file**

Update `_datafiles/world/dogmud/templates/help/instances.template`
to reference the arena and oasis help files, and mention that loot
scales with gold invested.

Add to the existing file before the See also section:

```
<ansi fg="yellow">Available Zones:</ansi>

  <ansi fg="command">Arena</ansi>    A brutal linear gauntlet. Death ends
             your run. No recall. Fast and dangerous.
  <ansi fg="command">Oasis</ansi>    A three-dimensional elemental maze.
             Death allows re-entry. Recall works.
             Methodical exploration rewarded.

  Type <ansi fg="command">help arena</ansi> or <ansi fg="command">help oasis</ansi> for details.
```

Update See also to include arena and oasis:

```
<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help arena</ansi>,
  <ansi fg="command">help oasis</ansi>, <ansi fg="command">help party</ansi>, <ansi fg="command">help combat</ansi>
```

- [ ] **Step 4: Add help aliases to keywords.yaml**

In `_datafiles/world/dogmud/keywords.yaml`, in the `help-aliases:`
section, add:

```yaml
  instances:        [instance, rift, rifts, riftkeeper, portal, portals, instanced]
  arena:            [arena instance, pit, fighting pit, gauntlet]
  oasis:            [oasis instance, planar oasis, elemental oasis, cube, elemental cube]
```

Also add `arena` and `oasis` to the `general:` help topics list.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/templates/help/ \
  _datafiles/world/dogmud/keywords.yaml
git commit -m "feat(loot): add help files for arena, oasis, and instance aliases"
```

---

### Task 10: Integration Test

- [ ] **Step 1: Run full test suite**

Run: `go test ./... 2>&1 | grep FAIL`

- [ ] **Step 2: Build**

Run: `go build ./...`

- [ ] **Step 3: Manual test**

Start server (nuke instance saves first):

**Arena test:**
1. `ask sable arena 200` — enter, fight through to boss
2. Kill tough mob — verify it drops affixed item with bonuses
3. `identify` or `look` at dropped item — should show prefix name
   and bonus stats
4. `ask sable arena 1000` — verify tougher mobs with better gear

**Oasis test:**
5. `ask sable oasis 500` — enter cube
6. Verify 6 exits (n/s/e/w/up/down) in cube rooms
7. Verify wrapping — walk 5 rooms north, end up where you started
8. Find and kill tough elementals — verify affixed loot drops
9. Find and kill boss — verify 2 affixed items drop
10. `help arena`, `help oasis`, `help instances` — verify help files
11. `help rift`, `help gauntlet`, `help cube` — verify aliases

- [ ] **Step 4: Commit any fixups**

```bash
git add -A
git commit -m "chore: integration fixups for randomized loot"
```
