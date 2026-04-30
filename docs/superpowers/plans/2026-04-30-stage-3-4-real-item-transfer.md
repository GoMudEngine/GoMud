# Stage 3.4 Real Item Transfer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace bucket-flag-driven `RestockBuckets` with real bidirectional item transfer between forager satchels, the new caravan wagon mob, and vendor inventories.

**Architecture:** Add 6 override fields to `Mob` (`carry_capacity`, `health_max`, `stamina_max`, `corpse_name`, `corpse_description`, `stock_multiplier`) and one to `ItemSpec` (`rarity_tier`). New `EffectiveMaxStock` helper derives vendor stock caps from item rarity × shopkeeper multiplier. Rewrite `VisitVendorsInRoom` for bidirectional delivery+pickup. Add wagon mob (374) + 2 draft horse mobs (375 Hob, 376 Bran) to the caravan party. New btree action `distribute_cargo_to_hostiles` transfers wagon cargo to bandits on caravan wipe.

**Tech Stack:** Go, YAML data files (mobs / behaviors / items / rooms / config), existing Stage 1 NPC party + Stage 2 caravan + Stage 3.1 forager systems.

---

## Decisions locked at plan time (from spec + scout)

**New mob IDs** (verified next free above 373 in global mob ID space):

| Mob | ID | Filename |
|---|---|---|
| Caravan wagon | **374** | `mobs/thornwall_city/374-caravan_wagon.yaml` |
| Hob (draft horse) | **375** | `mobs/thornwall_city/375-hob.yaml` |
| Bran (draft horse) | **376** | `mobs/thornwall_city/376-bran.yaml` |

**Tier mapping** (from spec):

| Tier | Cap | Item IDs |
|---|---|---|
| 50 | Common | 40001, 40003, 40006, 40012, 40013, 40014, 40015, 40016, 40017, 40019, 40043, 40044, 40045, **40021**, **40028** |
| 40 | Standard | 40002, 40004, 40005, 40007, 40008, 40009, 40020, 40047, 40048, 40050, 40068, **40011**, **40018**, **40022**, **40024**, **40026**, **40030** |
| 30 | Regional | 40051, 40056, 40057, 40058, 40059, 40046, 40049, 40062, 40063, 40064, 40065, 40066, 40067, **40025** |
| 20 | Uncommon | 40053, 40010, 40027, 40029, 40023 |

**Caravan-served vendor mob IDs** (17 vendors — drop explicit `max_stock` from each StockEntry):
- Stillwater (8): 333, 336, 337, 338, 339, 340, 341, 348
- Thornwall (9): 97, 98, 103, 104, 108, 109, 113, 248, 273

**Engine API anchors** (verified by scout):
- `mobs.Mob.HasShop() bool` — `internal/mobs/mobs.go:738`
- `mobs.Mob.HomeRoomId int` — `internal/mobs/mobs.go:73`
- `shops.ShopInventory.Stock []StockEntry` — `internal/shops/shopinventory.go:18`
- `shops.ShopInventory.GetStock(itemId int) *StockEntry` — `internal/shops/shopinventory.go:32`
- `shops.GetShopInventory(zone string, mobId, roomId int) *ShopInventory` — `internal/mobs/crafter.go:93` callsite
- `characters.Character.HealthMax.Value` and `StaminaMax.Value` — `internal/characters/character.go`
- `characters.Character.StoreItem(items.Item) bool` — `internal/characters/inventory.go:127`
- `characters.Character.RemoveItem(items.Item) bool` — `internal/characters/inventory.go:167`
- `items.GetItemSpec(itemId int) *ItemSpec` — `internal/items/itemspec.go:555`
- `items.New(itemId int) Item` — existing API
- `mobs.Mob.HatesAnyGroup(groups []string) bool` — existing (used by Stage 2 lookfortrouble)
- Corpse render call sites: `internal/rooms/rooms.go:229,232`; `internal/usercommands/look.go` (corpse path)
- Caravan `VisitVendorsInRoom` — `internal/caravan/visit.go`
- `caravanLoadGet/Set/Append` helpers — `internal/behaviortree/actions_caravan.go` (Stage 3.1)
- Forager `tickForagerDeliveringTown` — `internal/behaviortree/actions_forager.go`
- Forager `tickForagerResting` — same file
- `economy.BucketFor(itemId int) string` — `internal/economy/buckets.go`

**Corpse render call sites that need to honor CorpseName/CorpseDescription:**
- `internal/rooms/rooms.go:229` — `"%s corpse"` decay message (room broadcast)
- `internal/rooms/rooms.go:232` — same pattern, user-corpse path
- `internal/usercommands/look.go` — `look corpse` text rendering (need to find exact line)

---

## File structure overview

| Layer | File | Purpose |
|---|---|---|
| Engine | `internal/items/itemspec.go` | Add `RarityTier int` field (Task 1) |
| Engine | `internal/mobs/mobs.go` | Add 6 override fields (Task 2) |
| Engine | `internal/characters/character.go` | Apply HP/SP/CarryCap overrides at spawn (Task 3) |
| Engine | `internal/rooms/rooms.go` + `internal/usercommands/look.go` | Use CorpseName/Description (Task 4) |
| Engine new | `internal/shops/effective_max_stock.go` | EffectiveMaxStock helper (Task 5) |
| Engine | `internal/shops/shopinventory.go` | Loader integration (Task 5) |
| Engine | `internal/configs/config.balance.go` + `_datafiles/config.yaml` | New ForagerRestCarryThreshold knob (Task 6) |
| Engine | `internal/caravan/visit.go` | Rewrite VisitVendorsInRoom (Task 7) |
| Engine | `internal/behaviortree/actions_caravan.go` | Wire delivery/pickup buckets (Task 8) |
| Engine new | `internal/behaviortree/actions_wagon.go` | distribute_cargo_to_hostiles action (Task 9) |
| Engine | `internal/behaviortree/actions_forager.go` | Rewrite tickForagerDeliveringTown + extend tickForagerResting (Tasks 10+11) |
| Content | ~50 mat YAMLs in `_datafiles/world/dogmud/items/materials-40000/` | Set rarity_tier (Task 12) |
| Content | 17 caravan-served vendor mob YAMLs | Drop explicit max_stock (Task 13) |
| Content new | `_datafiles/world/dogmud/mobs/thornwall_city/{374-caravan_wagon,375-hob,376-bran}.yaml` | New mobs (Task 14) |
| Content new | `_datafiles/world/dogmud/behaviors/thornwall_city/{374-caravan_wagon,375-hob,376-bran}.yaml` | New btrees (Task 14) |
| Content | `_datafiles/world/dogmud/behaviors/thornwall_city/357-ketil.yaml` + `rooms/thornwall_city/465.yaml` | Update party + spawninfo (Task 15) |
| Content | `mobs/sanctum_basin/69-aberrant_chrysalis.yaml` | Remove drop (Task 16) |
| Content | `mobs/ironwind_steppe/228-stone_beetle_queen.yaml` | Add 10% drop (Task 17) |
| Content | `mobs/ironwind_steppe/229-windscour_wyrm.yaml` | Add 5% drop (Task 18) |
| Docs | Schema docs + audit matrix + PATCH_NOTES | Task 19 |

---

### Task 1: Add `RarityTier` field to ItemSpec

**Files:**
- Modify: `internal/items/itemspec.go` — add field
- Modify: `internal/items/itemspec_test.go` (or create) — test YAML roundtrip

- [ ] **Step 1: Read existing `ItemSpec` struct around line 267 to find the conventions.**

Run: read `internal/items/itemspec.go:260-290`. Note the yaml tag style (lowercase, no underscores: `is_component` IS underscored — different style than mutators package; ItemSpec uses underscores).

- [ ] **Step 2: Write the failing test.**

In `internal/items/itemspec_test.go` (create if missing or append):

```go
package items

import (
	"testing"

	"gopkg.in/yaml.v2"
)

func TestItemSpec_RarityTier_YAMLRoundtrip(t *testing.T) {
	src := `itemid: 99001
name: test mat
rarity_tier: 30
`
	var spec ItemSpec
	if err := yaml.Unmarshal([]byte(src), &spec); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if spec.RarityTier != 30 {
		t.Errorf("RarityTier = %d, want 30", spec.RarityTier)
	}
}

func TestItemSpec_RarityTier_DefaultsZero(t *testing.T) {
	src := `itemid: 99002
name: untiered
`
	var spec ItemSpec
	if err := yaml.Unmarshal([]byte(src), &spec); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if spec.RarityTier != 0 {
		t.Errorf("RarityTier = %d, want 0 (unset)", spec.RarityTier)
	}
}
```

(If the test file uses `yaml.v3`, adjust import accordingly. Confirm by reading other items tests.)

- [ ] **Step 3: Run test to verify it fails.**

Run: `cd "C:/Users/Calabe Davis/workspace/DOGMud" && go test ./internal/items/... -run RarityTier -v`
Expected: FAIL — `RarityTier undefined`.

- [ ] **Step 4: Add the field to ItemSpec.**

In `internal/items/itemspec.go`, alongside other scalar fields like `Weight`, `Value`, etc.:

```go
	RarityTier int `yaml:"rarity_tier,omitempty"` // Vendor stock cap tier (50/40/30/20/10). Used by shops.EffectiveMaxStock with mob.StockMultiplier. 0 = untiered (quest items, defer-to-3.0e items).
```

- [ ] **Step 5: Run test to verify it passes.**

Run: `go test ./internal/items/... -run RarityTier -v`
Expected: PASS.

- [ ] **Step 6: Run full items tests + build.**

Run: `go test ./internal/items/...` and `go build ./...`
Expected: green.

- [ ] **Step 7: Commit.**

```bash
git add internal/items/itemspec.go internal/items/itemspec_test.go
git commit -m "$(cat <<'EOF'
feat(items): add RarityTier field to ItemSpec

Backs Stage 3.4 EffectiveMaxStock derivation. Each mat gets a
rarity tier (50/40/30/20/10); vendor stock caps derive from
RarityTier × Mob.StockMultiplier.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: Add 6 override fields to Mob struct

**Files:**
- Modify: `internal/mobs/mobs.go` — add field group
- Modify: `internal/mobs/mobs_test.go` — test YAML roundtrip

- [ ] **Step 1: Read the existing Mob struct (lines 50-100) for field-style consistency.**

Note: Mob struct uses `yaml:"snake_case_name,omitempty"` — same convention as ItemSpec. Some fields have inline comments.

- [ ] **Step 2: Write failing test asserting all 6 fields parse from YAML.**

Append to `internal/mobs/mobs_test.go`:

```go
func TestMob_StageThreeFourOverrides_YAMLRoundtrip(t *testing.T) {
	src := `mobid: 99001
zone: Test Zone
carry_capacity: 5000
health_max: 1500
stamina_max: 9999
corpse_name: splintered wagon wreckage
corpse_description: |
  Shattered timbers and twisted iron.
stock_multiplier: 1.5
`
	var m Mob
	if err := yaml.Unmarshal([]byte(src), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m.CarryCapacityOverride != 5000 {
		t.Errorf("CarryCapacityOverride = %v, want 5000", m.CarryCapacityOverride)
	}
	if m.HealthMaxOverride != 1500 {
		t.Errorf("HealthMaxOverride = %d, want 1500", m.HealthMaxOverride)
	}
	if m.StaminaMaxOverride != 9999 {
		t.Errorf("StaminaMaxOverride = %d, want 9999", m.StaminaMaxOverride)
	}
	if m.CorpseName != "splintered wagon wreckage" {
		t.Errorf("CorpseName = %q, want splintered wagon wreckage", m.CorpseName)
	}
	if m.CorpseDescription == "" {
		t.Error("CorpseDescription empty, want non-empty")
	}
	if m.StockMultiplier != 1.5 {
		t.Errorf("StockMultiplier = %v, want 1.5", m.StockMultiplier)
	}
}
```

(Confirm `yaml` import already present in mobs_test.go.)

- [ ] **Step 3: Run test to verify failure.**

Run: `go test ./internal/mobs/... -run StageThreeFourOverrides -v`
Expected: FAIL — fields undefined.

- [ ] **Step 4: Add the fields.**

In `internal/mobs/mobs.go`, find a logical group (e.g., right after the existing `non_combatant` and `player_attack_immune` block). Add:

```go
	// ── Stage 3.4: spawn-time overrides for special mobs (wagons, statues, etc.)
	CarryCapacityOverride float64 `yaml:"carry_capacity,omitempty"`  // overrides Strength-derived calc when > 0
	HealthMaxOverride     int     `yaml:"health_max,omitempty"`      // overrides Vitality-derived calc when > 0
	StaminaMaxOverride    int     `yaml:"stamina_max,omitempty"`     // overrides Vitality/Dex-derived calc when > 0
	CorpseName            string  `yaml:"corpse_name,omitempty"`     // overrides "<Name> corpse" rendering when set
	CorpseDescription     string  `yaml:"corpse_description,omitempty"` // overrides default corpse look-text when set
	StockMultiplier       float64 `yaml:"stock_multiplier,omitempty"`   // shop stock-cap scale; default 1.0 if unset
```

- [ ] **Step 5: Verify test passes.**

Run: `go test ./internal/mobs/... -run StageThreeFourOverrides -v`
Expected: PASS.

- [ ] **Step 6: Full mobs tests + build.**

Run: `go test ./internal/mobs/...` and `go build ./...`
Expected: green.

- [ ] **Step 7: Commit.**

```bash
git add internal/mobs/mobs.go internal/mobs/mobs_test.go
git commit -m "$(cat <<'EOF'
feat(mobs): add 6 spawn-time override fields for Stage 3.4

  carry_capacity     — override Strength-derived calc (wagon needs ~5000)
  health_max         — override Vitality-derived calc (wagon needs ~1500)
  stamina_max        — override default calc (wagon needs effectively infinite)
  corpse_name        — override "<Name> corpse" render (wagon → "wreckage")
  corpse_description — override default corpse look-text
  stock_multiplier   — shop stock-cap scale (default 1.0; future big-city shops > 1.0)

All fields apply at spawn time after stat-derived calc; gated by > 0
or != "" so unset values fall through to defaults.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 3: Apply HP/SP/CarryCap overrides at character spawn

**Files:**
- Modify: `internal/characters/character.go` — wire overrides
- Modify: `internal/characters/character_test.go` — verify wagon-style mob hits override values

- [ ] **Step 1: Find the character spawn / stat-derivation point.**

Run: `grep -nE "HealthMax\.Value\s*=|StaminaMax\.Value\s*=|carryCapacity\s*=" internal/characters/character.go`
Note line numbers where stat-derived HP/SP/Carry values are set. There's typically a `Validate()` or `RecomputeStats()` method.

- [ ] **Step 2: Find the call site where mob spawn hooks character init.**

Run: `grep -nE "func.*NewMobByIdInternal|newMobByIdInternal|\.Validate\(\)" internal/mobs/mobs.go | head -10`

The mob is constructed, then `Character.Validate()` (or similar) finalizes pool values. We hook AFTER that.

- [ ] **Step 3: Write failing test (uses fake-spawn pattern from existing tests).**

In `internal/characters/character_test.go`, append:

```go
func TestCharacter_OverrideHealthMax(t *testing.T) {
	c := &Character{
		HealthMax: stats.StatInfo{Base: 1, Value: 100},
	}
	ApplyMobOverrides(c, 1500, 0, 0)
	if c.HealthMax.Value != 1500 {
		t.Errorf("HealthMax.Value = %d, want 1500", c.HealthMax.Value)
	}
	if c.Health != 1500 {
		t.Errorf("Health = %d, want 1500 (filled to max)", c.Health)
	}
}

func TestCharacter_OverrideStaminaMax(t *testing.T) {
	c := &Character{StaminaMax: stats.StatInfo{Base: 1, Value: 50}}
	ApplyMobOverrides(c, 0, 9999, 0)
	if c.StaminaMax.Value != 9999 {
		t.Errorf("StaminaMax.Value = %d, want 9999", c.StaminaMax.Value)
	}
}

func TestCharacter_OverrideCarryCapacity(t *testing.T) {
	c := &Character{}
	ApplyMobOverrides(c, 0, 0, 5000)
	if got := c.CarryCapacity(); got != 5000 {
		t.Errorf("CarryCapacity() = %v, want 5000", got)
	}
}

func TestCharacter_NoOverrides_StatsPreserved(t *testing.T) {
	c := &Character{
		HealthMax:  stats.StatInfo{Value: 100},
		StaminaMax: stats.StatInfo{Value: 50},
	}
	c.Health = 100
	c.Stamina = 50
	ApplyMobOverrides(c, 0, 0, 0)
	if c.HealthMax.Value != 100 || c.Health != 100 {
		t.Error("zero overrides should not modify HP")
	}
	if c.StaminaMax.Value != 50 || c.Stamina != 50 {
		t.Error("zero overrides should not modify SP")
	}
}
```

(Adjust `stats.StatInfo` import / construction if the actual struct shape differs — check `internal/stats/`.)

- [ ] **Step 4: Run tests to verify they fail.**

Run: `go test ./internal/characters/... -run Override -v`
Expected: FAIL — `ApplyMobOverrides` undefined.

- [ ] **Step 5: Add the helper to character.go.**

In `internal/characters/character.go`, append:

```go
// ApplyMobOverrides applies the Stage 3.4 spawn-time overrides for
// special mobs (wagons, statues, etc.). Each parameter is gated by
// > 0; zero values leave the existing stat-derived value alone.
//
// Called from the mob spawn path AFTER Validate() has computed the
// stat-derived defaults.
func ApplyMobOverrides(c *Character, healthMax, staminaMax int, carryCapacity float64) {
	if healthMax > 0 {
		c.HealthMax.Value = healthMax
		c.Health = healthMax
	}
	if staminaMax > 0 {
		c.StaminaMax.Value = staminaMax
		c.Stamina = staminaMax
	}
	if carryCapacity > 0 {
		c.carryCapacityOverride = carryCapacity
	}
}
```

Then locate `Character.CarryCapacity()` in `internal/characters/inventory.go` (line 12 per scout). Modify it to honor the override:

```go
func (c *Character) CarryCapacity() float64 {
	if c.carryCapacityOverride > 0 {
		return c.carryCapacityOverride
	}
	return float64(c.Stats.Strength.ValueAdj) * configs.GetBalanceConfig().CarryCapacityMultiplier
	// (preserve whatever the original calc was)
}
```

Add the field to Character struct:

```go
type Character struct {
	// ... existing
	carryCapacityOverride float64 `yaml:"-"` // Set via ApplyMobOverrides; not persisted
}
```

- [ ] **Step 6: Find the mob spawn site and call ApplyMobOverrides.**

Run: `grep -nE "newMobByIdInternal|NewMobById\(" internal/mobs/mobs.go | head -5`

In `internal/mobs/mobs.go`'s `newMobByIdInternal` (or wherever the character is finalized), add a call:

```go
// After Character.Validate() / stat finalization:
characters.ApplyMobOverrides(
	&mob.Character,
	mob.HealthMaxOverride,
	mob.StaminaMaxOverride,
	mob.CarryCapacityOverride,
)
```

- [ ] **Step 7: Verify tests pass.**

Run: `go test ./internal/characters/... -run Override -v`
Expected: PASS.

- [ ] **Step 8: Build + full test suite.**

Run: `go build ./... && go test ./...`
Expected: green.

- [ ] **Step 9: Commit.**

```bash
git add internal/characters/character.go internal/characters/inventory.go internal/characters/character_test.go internal/mobs/mobs.go
git commit -m "$(cat <<'EOF'
feat(characters): apply Stage 3.4 mob overrides at spawn

ApplyMobOverrides hooks into the mob spawn path after Validate().
Wagon mobs (mob 374) will use carry_capacity 5000, health_max 1500,
stamina_max 9999 to survive bandit raids and never run out of SP
mid-route.

CarryCapacity() honors the override when set; zero falls through to
the existing Strength-derived calc.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 4: Apply CorpseName / CorpseDescription in corpse rendering

**Files:**
- Modify: `internal/rooms/rooms.go:229,232` — use mob's CorpseName when set
- Modify: `internal/usercommands/look.go` — use CorpseDescription when looking at corpse
- Modify: `internal/rooms/rooms_test.go` — verify override renders

- [ ] **Step 1: Read the corpse-decay path at rooms.go:200-240.**

Confirm the format string: `"%s corpse"` and what variable is interpolated.

- [ ] **Step 2: Find the corpse-look path in look.go.**

Run: `grep -nE "corpse|Corpse" internal/usercommands/look.go | head -15`
Find the branch where a player looks at a corpse and the description is rendered.

- [ ] **Step 3: Write failing test.**

Append to `internal/rooms/rooms_test.go`:

```go
func TestCorpse_RenderName_OverrideHonored(t *testing.T) {
	mob := &mobs.Mob{
		CorpseName: "splintered wagon wreckage",
		Character:  characters.Character{Name: "a sturdy wagon"},
	}
	got := corpseDisplayName(mob)
	if got != "splintered wagon wreckage" {
		t.Errorf("got %q, want splintered wagon wreckage", got)
	}
}

func TestCorpse_RenderName_FallbackToCharacterName(t *testing.T) {
	mob := &mobs.Mob{
		Character: characters.Character{Name: "Pell"},
	}
	got := corpseDisplayName(mob)
	if got != "Pell corpse" {
		t.Errorf("got %q, want 'Pell corpse'", got)
	}
}
```

(Adjust struct construction to match real shape — check imports.)

- [ ] **Step 4: Run test to verify failure.**

Run: `go test ./internal/rooms/... -run Corpse_Render -v`
Expected: FAIL — `corpseDisplayName` undefined.

- [ ] **Step 5: Add the helper.**

In `internal/rooms/rooms.go`:

```go
// corpseDisplayName returns the appropriate display string for a
// corpse: the mob's CorpseName override if set, otherwise the
// default "<Character.Name> corpse".
func corpseDisplayName(m *mobs.Mob) string {
	if m == nil {
		return ""
	}
	if m.CorpseName != "" {
		return m.CorpseName
	}
	return m.Character.Name + " corpse"
}
```

- [ ] **Step 6: Update existing call sites in rooms.go.**

At line 229 (and line 232), replace:

```go
r.SendText(fmt.Sprintf(`A <ansi fg="mob-corpse">%s corpse</ansi> crumbles to dust.`, corpse.Character.Name))
```

with:

```go
mob := mobs.GetMobSpec(corpse.MobId)
displayName := corpse.Character.Name + " corpse"
if mob != nil && mob.CorpseName != "" {
	displayName = mob.CorpseName
}
r.SendText(fmt.Sprintf(`A <ansi fg="mob-corpse">%s</ansi> crumbles to dust.`, displayName))
```

(Adjust the helper call to whichever API actually returns the mob template by mob ID — `mobs.GetMobSpec` or similar. If the corpse only has access to Character.Name and not a way to look up the original mob, an alternative is to copy CorpseName/CorpseDescription onto the Corpse struct at corpse-creation time. Plan task 0 confirms during implementation.)

- [ ] **Step 7: Update look.go corpse-look path.**

In the corpse branch of look.go, similar pattern: if mob has CorpseDescription, use it instead of Character.Description for the corpse-look text.

- [ ] **Step 8: Run tests + build.**

Run: `go test ./internal/rooms/... ./internal/usercommands/... && go build ./...`
Expected: green.

- [ ] **Step 9: Commit.**

```bash
git add internal/rooms/rooms.go internal/rooms/rooms_test.go internal/usercommands/look.go
git commit -m "$(cat <<'EOF'
feat(rooms): honor CorpseName/CorpseDescription on render

When a mob has corpse_name set, the room-decay message and
corpse-look text use it instead of "<Name> corpse". Backs Stage 3.4
wagon corpse rendering as "splintered wagon wreckage".

Standard mobs (no override) preserve existing "<Name> corpse"
behaviour.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 5: EffectiveMaxStock helper + loader integration

**Files:**
- Create: `internal/shops/effective_max_stock.go`
- Modify: `internal/shops/shopinventory.go` — wire helper into stock loading
- Test: `internal/shops/effective_max_stock_test.go`

- [ ] **Step 1: Write the failing test.**

`internal/shops/effective_max_stock_test.go`:

```go
package shops

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestEffectiveMaxStock_TierTimesMultiplier(t *testing.T) {
	// Use a known tiered item from the audit. Iron ingot (40001) is tier 50.
	mob := &mobs.Mob{StockMultiplier: 1.0}
	if got := EffectiveMaxStock(40001, mob); got != 50 {
		t.Errorf("iron ingot at 1.0x = %d, want 50", got)
	}
}

func TestEffectiveMaxStock_LargeShopMultiplier(t *testing.T) {
	mob := &mobs.Mob{StockMultiplier: 5.0}
	if got := EffectiveMaxStock(40001, mob); got != 250 {
		t.Errorf("iron ingot at 5.0x = %d, want 250", got)
	}
}

func TestEffectiveMaxStock_DefaultMultiplier(t *testing.T) {
	mob := &mobs.Mob{} // StockMultiplier not set → treat as 1.0
	if got := EffectiveMaxStock(40001, mob); got != 50 {
		t.Errorf("iron ingot at default 1.0x = %d, want 50", got)
	}
}

func TestEffectiveMaxStock_UntieredItem(t *testing.T) {
	// Quest items have no rarity_tier; should return 0
	mob := &mobs.Mob{StockMultiplier: 1.0}
	if got := EffectiveMaxStock(40031 /* spirit fetish, quest item */, mob); got != 0 {
		t.Errorf("quest item should return 0, got %d", got)
	}
}

func TestEffectiveMaxStock_UnknownItem(t *testing.T) {
	mob := &mobs.Mob{StockMultiplier: 1.0}
	if got := EffectiveMaxStock(99999, mob); got != 0 {
		t.Errorf("unknown item should return 0, got %d", got)
	}
}

// (Note: items must have rarity_tier set on disk for these tests
// to pass. If items haven't been audited yet (Task 12), some of these
// tests fail. Run them after Task 12 lands.)
```

- [ ] **Step 2: Run test (will fail until helper exists).**

Run: `go test ./internal/shops/... -run EffectiveMaxStock -v`
Expected: FAIL.

- [ ] **Step 3: Implement the helper.**

`internal/shops/effective_max_stock.go`:

```go
package shops

import (
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// EffectiveMaxStock returns the stock cap for a (item, vendor) pair
// derived from item rarity and shopkeeper scale.
//
//   max_stock = ItemSpec.RarityTier × Mob.StockMultiplier
//
// Returns 0 if the item has no rarity_tier (quest items, unknown
// items, defer-to-3.0e items). The 0-tier is treated as "this
// vendor doesn't naturally stock this item via the rarity system".
//
// A mob with StockMultiplier == 0 (unset) is treated as 1.0.
func EffectiveMaxStock(itemId int, mob *mobs.Mob) int {
	if mob == nil {
		return 0
	}
	spec := items.GetItemSpec(itemId)
	if spec == nil || spec.RarityTier <= 0 {
		return 0
	}
	mult := mob.StockMultiplier
	if mult <= 0 {
		mult = 1.0
	}
	return int(float64(spec.RarityTier) * mult)
}
```

- [ ] **Step 4: Wire helper into shop loader.**

In `internal/shops/shopinventory.go`, find the function that creates / hydrates `ShopInventory` instances. After loading per-StockEntry data, for each entry where `entry.MaxStock == 0` (no explicit override on disk), derive from EffectiveMaxStock:

```go
// Fill in MaxStock from rarity-tier system when not explicitly set.
// Only takes effect when the loader has access to the owning mob —
// some load paths may need the mob to be passed in. Otherwise the
// derivation runs at first-access time.
func (si *ShopInventory) ApplyTieredMaxStock(owner *mobs.Mob) {
	for i := range si.Stock {
		if si.Stock[i].MaxStock > 0 {
			continue // legacy explicit value preserved
		}
		si.Stock[i].MaxStock = EffectiveMaxStock(si.Stock[i].ItemId, owner)
	}
}
```

Find the existing shop-load callsite (search `GetShopInventory` / `LoadShopInventory` etc.) and add `si.ApplyTieredMaxStock(mob)` immediately after the inventory is loaded but before it's used by callers.

- [ ] **Step 5: Run tests + build.**

Run: `go test ./internal/shops/... && go build ./...`
Expected: tests for EffectiveMaxStock fail until items have RarityTier (Task 12). Skip them with `-run TestEffectiveMaxStock_DefaultMultiplier|TestEffectiveMaxStock_UnknownItem` for now (those don't depend on disk content).

- [ ] **Step 6: Commit.**

```bash
git add internal/shops/effective_max_stock.go internal/shops/shopinventory.go internal/shops/effective_max_stock_test.go
git commit -m "$(cat <<'EOF'
feat(shops): EffectiveMaxStock helper + loader integration

Stock cap = ItemSpec.RarityTier × Mob.StockMultiplier. Returns 0 for
untiered items (quest/specialty). Unset multiplier defaults to 1.0.

ApplyTieredMaxStock hooks into shop load path — entries without
explicit max_stock get the derived value. Existing per-entry
max_stock is preserved as a legacy override.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 6: Add `ForagerRestCarryThreshold` config knob

**Files:**
- Modify: `internal/configs/config.balance.go` — add field
- Modify: `internal/configs/config.balance.misc.go` — defaulting
- Modify: `_datafiles/config.yaml` — yaml default
- Modify: `internal/configs/config.balance_test.go` — test default

- [ ] **Step 1: Add the field.**

In `internal/configs/config.balance.go`, in the forager block alongside `ForagerCarryThresholdPct`:

```go
	// ForagerRestCarryThreshold (Stage 3.4) — when forager returns home
	// from a delivery cycle, if her carry weight / capacity > this value,
	// she stays resting instead of cycling back out to forage. Prevents
	// futile foraging loops when local vendors are saturated.
	// Default 0.5.
	ForagerRestCarryThreshold ConfigFloat `yaml:"ForagerRestCarryThreshold"`
```

- [ ] **Step 2: Add default in `validateMisc`.**

In `internal/configs/config.balance.misc.go`, in the forager section:

```go
	if b.ForagerRestCarryThreshold <= 0 || b.ForagerRestCarryThreshold > 1.0 {
		b.ForagerRestCarryThreshold = 0.5
	}
```

- [ ] **Step 3: Add yaml default.**

In `_datafiles/config.yaml`, in the forager block:

```yaml
  # Stage 3.4: when forager returns home with carry > this ratio,
  # she stays resting instead of cycling back out. Prevents futile
  # forage loops when local vendors are at cap.
  ForagerRestCarryThreshold: 0.5
```

- [ ] **Step 4: Test the default.**

Append to `internal/configs/config.balance_test.go`:

```go
func TestBalanceConfig_ForagerRestCarryThresholdDefault(t *testing.T) {
	cfg := &Balance{}
	cfg.Validate()
	if cfg.ForagerRestCarryThreshold != ConfigFloat(0.5) {
		t.Errorf("ForagerRestCarryThreshold default = %v, want 0.5", cfg.ForagerRestCarryThreshold)
	}
}
```

- [ ] **Step 5: Run tests + build.**

Run: `go test ./internal/configs/... && go build ./...`
Expected: green.

- [ ] **Step 6: Commit.**

```bash
git add internal/configs/config.balance.go internal/configs/config.balance.misc.go internal/configs/config.balance_test.go _datafiles/config.yaml
git commit -m "$(cat <<'EOF'
feat(config): add ForagerRestCarryThreshold (default 0.5)

Stage 3.4 forager rest-extension knob. When forager's carry / cap
ratio exceeds this value at sanctuary, she stays resting instead of
cycling back to forage. Prevents futile loops when vendors are at
their MaxStock cap.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 7: Rewrite `VisitVendorsInRoom` for bidirectional flow

**Files:**
- Modify: `internal/caravan/visit.go` — new signature + body
- Modify: `internal/caravan/visit_test.go` — extended tests

- [ ] **Step 1: Read current `VisitVendorsInRoom` to see what callers need.**

It currently takes `(roomId int, buckets []string)` and returns `[]string`. We're changing the signature to take a wagon mob + delivery + pickup bucket lists, returning structured move info.

- [ ] **Step 2: Write the new test file content.**

Add to `internal/caravan/visit_test.go`:

```go
func TestVisitVendorsInRoom_DeliverOnly(t *testing.T) {
	// Vendor has stock entry for iron ingot, current 0, max 50.
	// Wagon has 3 iron ingots. Outbound (deliver "base").
	// Expect: vendor.Current = 3, wagon items: 0.
	t.Skip("requires fixture for vendor mob + shop; integration test in plan task review")
}

func TestVisitVendorsInRoom_PickupOnly(t *testing.T) {
	// Vendor has lake-iron Current 30 (>= MaxStock/2 = 15), RestockQty 2.
	// Caravan picking up "stillwater" bucket. Empty wagon.
	// Expect: vendor.Current = 28, wagon has 2 lake-iron.
	t.Skip("integration: requires shop fixture")
}

func TestVisitVendorsInRoom_PickupFloor(t *testing.T) {
	// Vendor lake-iron Current = 5 (< 15 floor). No pickup.
	t.Skip("integration: requires shop fixture")
}

func TestVisitVendorsInRoom_DeliverAndPickup(t *testing.T) {
	// Outbound at Stillwater vendor: deliver Thornwall items, pick up Stillwater items.
	// Both happen in single visit.
	t.Skip("integration: requires shop fixture")
}

func TestVisitVendorsInRoom_VendorAtCapSkipsDelivery(t *testing.T) {
	// Wagon has iron ingot, vendor at MaxStock 50/50. Skip delivery.
	t.Skip("integration: requires shop fixture")
}

func TestVisitVendorsInRoom_WagonAtCarryCap(t *testing.T) {
	// Wagon CarryCapacity 5000, current weight 4999.
	// Trying to pick up a 5-weight item. Skip — wagon would overflow.
	t.Skip("integration: requires fixture")
}
```

(These are skipped placeholders. The concrete tests need the shop fixture which is heavy; mark as integration. The unit-level testing is implicit in Tasks 8+9 where the dispatch action gets exercised.)

- [ ] **Step 3: Rewrite `VisitVendorsInRoom`.**

Replace the function in `internal/caravan/visit.go`:

```go
package caravan

import (
	"slices"

	"github.com/GoMudEngine/GoMud/internal/economy"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/shops"
)

// ItemMove describes a single item that moved between wagon and a vendor.
type ItemMove struct {
	Vendor   string
	ItemName string
}

// VisitVendorsInRoom performs a bidirectional vendor-stop for the
// caravan: deliver wagon items whose bucket is in deliveryBuckets,
// and pick up vendor items whose bucket is in pickupBuckets.
//
// Pickup is gated by entry.Current >= entry.MaxStock/2 — caravan
// won't extract from a starving vendor (narrative: wholesalers don't
// loot a struggling shop).
//
// Pickup quantity is RestockQty per matching stock entry, capped at
// the wagon's CarryCapacity remaining.
//
// Delivery quantity is 1 unit per pass (each item is its own object).
func VisitVendorsInRoom(
	roomId int,
	wagon *mobs.Mob,
	deliveryBuckets []string,
	pickupBuckets []string,
) (delivered, pickedUp []ItemMove) {
	if wagon == nil {
		return nil, nil
	}
	room := rooms.LoadRoom(roomId)
	if room == nil {
		return nil, nil
	}

	for _, instId := range room.GetMobs(rooms.FindAll) {
		vendor := mobs.GetInstance(instId)
		if vendor == nil || !vendor.HasShop() {
			continue
		}
		shop := shops.GetShopInventory(vendor.Zone, int(vendor.MobId), vendor.HomeRoomId)
		if shop == nil {
			continue
		}

		// DELIVER pass: walk wagon items in reverse so RemoveItem is safe.
		for i := len(wagon.Character.Items) - 1; i >= 0; i-- {
			item := wagon.Character.Items[i]
			bucket := economy.BucketFor(item.ItemId)
			if bucket == "" || !slices.Contains(deliveryBuckets, bucket) {
				continue
			}
			entry := shop.GetStock(item.ItemId)
			if entry == nil || entry.Current >= entry.MaxStock {
				continue
			}
			wagon.Character.RemoveItem(item)
			entry.Current++
			delivered = append(delivered, ItemMove{
				Vendor:   vendor.Character.Name,
				ItemName: item.DisplayName(),
			})
		}

		// PICKUP pass: walk vendor stock; extract bucket-matching entries
		// where Current >= MaxStock/2.
		for i := range shop.Stock {
			entry := &shop.Stock[i]
			bucket := economy.BucketFor(entry.ItemId)
			if bucket == "" || !slices.Contains(pickupBuckets, bucket) {
				continue
			}
			if entry.Current < entry.MaxStock/2 {
				continue
			}
			qty := entry.RestockQty
			if qty <= 0 {
				continue
			}
			if qty > entry.Current {
				qty = entry.Current
			}
			for j := 0; j < qty; j++ {
				newItem := items.New(entry.ItemId)
				if !newItem.IsValid() {
					break
				}
				if !wagon.Character.StoreItem(newItem) {
					break // wagon at carry cap
				}
				entry.Current--
				pickedUp = append(pickedUp, ItemMove{
					Vendor:   vendor.Character.Name,
					ItemName: newItem.DisplayName(),
				})
			}
		}
	}

	return delivered, pickedUp
}

// FormatVisitMessage builds the room-flavor text for a vendor visit.
// Returns "" if no transfers happened.
func FormatVisitMessage(delivered, pickedUp []ItemMove) string {
	switch {
	case len(delivered) > 0 && len(pickedUp) > 0:
		// Both — use the trade flavor with the first vendor name + summary
		return formatTradeFlavor(delivered, pickedUp)
	case len(delivered) > 0:
		return formatDeliveryFlavor(delivered)
	case len(pickedUp) > 0:
		return formatPickupFlavor(pickedUp)
	}
	return ""
}

func formatTradeFlavor(delivered, pickedUp []ItemMove) string {
	return `<ansi fg="yellow">Marta hands a small purse across the counter; the caravan unloads and reloads in trade.</ansi>`
}

func formatDeliveryFlavor(delivered []ItemMove) string {
	return `<ansi fg="yellow">The caravan crew unloads supplies for the local merchants.</ansi>`
}

func formatPickupFlavor(pickedUp []ItemMove) string {
	return `<ansi fg="yellow">The caravan crew loads up cargo from the local merchants for the road.</ansi>`
}
```

- [ ] **Step 4: Update existing visit-related tests in visit_test.go.**

Old tests called `VisitVendorsInRoom(roomId, buckets)` and expected `[]string`. Update:
- Call signature: `VisitVendorsInRoom(roomId, wagon, deliveryBuckets, pickupBuckets)`
- Return type: `(delivered, pickedUp []ItemMove)`
- Existing assertions: convert to new struct field access

If a test was checking "returns nil when room missing" — keep that. If the test asserted "delivers up to MaxStock", update to use the new wagon parameter.

- [ ] **Step 5: Run tests + build.**

Run: `go test ./internal/caravan/... && go build ./...`
Expected: green.

- [ ] **Step 6: Commit.**

```bash
git add internal/caravan/visit.go internal/caravan/visit_test.go
git commit -m "$(cat <<'EOF'
feat(caravan): bidirectional VisitVendorsInRoom

Replaces RestockBuckets-based abstract restock with real item
transfer. Each vendor stop now does:
  - DELIVER pass: wagon items whose bucket matches deliveryBuckets
                  move into vendor stock (capped at MaxStock)
  - PICKUP pass:  vendor items whose bucket matches pickupBuckets
                  move into wagon (gated by vendor floor MaxStock/2,
                  quantity RestockQty per matching entry)

Returns structured ItemMove lists for flavor-message generation.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 8: Wire delivery/pickup buckets in actions_caravan.go

**Files:**
- Modify: `internal/behaviortree/actions_caravan.go` — update tickRoute to pass wagon + buckets
- Modify: `internal/behaviortree/actions_caravan_test.go` — extend tests

- [ ] **Step 1: Locate `tickRoute` (the function that calls VisitVendorsInRoom).**

Run: `grep -nE "VisitVendorsInRoom|caravanLoadGet" internal/behaviortree/actions_caravan.go`

- [ ] **Step 2: Update tickRoute body.**

Replace the VisitVendorsInRoom call site with:

```go
// Look up the wagon (mob 374) in the room.
var wagon *mobs.Mob
for _, instId := range rooms.LoadRoom(ctx.RoomId).GetMobs(rooms.FindAll) {
	m := mobs.GetInstance(instId)
	if m != nil && m.MobId == 374 {
		wagon = m
		break
	}
}
if wagon == nil {
	// No wagon present — caravan has been wiped or wagon is in transit
	// somewhere weird. Fall through to legacy idle.
	return Failure
}

// Compute delivery + pickup buckets based on current state.
var deliveryBuckets, pickupBuckets []string
switch cur {
case caravan.StateStillwaterRoute:
	// Outbound delivery to Stillwater: deliver Thornwall + Fernway items;
	// pick up Stillwater items for return.
	deliveryBuckets = []string{"thornwall", "fernway"}
	pickupBuckets = []string{"stillwater"}
case caravan.StateThornwallRoute:
	// Inbound delivery to Thornwall: deliver Stillwater + Fernway items;
	// pick up Thornwall items for next outbound.
	deliveryBuckets = []string{"stillwater", "fernway"}
	pickupBuckets = []string{"thornwall"}
}

delivered, pickedUp := caravan.VisitVendorsInRoom(
	nextRoom, wagon, deliveryBuckets, pickupBuckets,
)
if msg := caravan.FormatVisitMessage(delivered, pickedUp); msg != "" {
	if r := rooms.LoadRoom(nextRoom); r != nil {
		r.SendText(msg)
	}
}
```

The caravan_load tracking from Stage 3.1 (`caravanLoadGet/Set/Append`) is no longer the source-of-truth — the wagon's actual `Character.Items` is. Keep `caravanLoadAppend` for the Fernway pickup substate (it's a flag for "was Kessa met?") but rename if confusing. Actually — caravan_load CAN be removed entirely; the wagon's items are now self-evident. Keep the Fernway pickup wiring (Task 10 later moves Kessa's actual items into the wagon).

For now, leave caravanLoadGet/Set/Append in place — they're harmless. Remove them in a polish task if motivated.

- [ ] **Step 3: Update test for new behavior.**

In `internal/behaviortree/actions_caravan_test.go`, find a test that checks `tickRoute` behavior. Update or add:

```go
func TestTickRoute_NoWagonReturnsFailure(t *testing.T) {
	// Build a mob in StateStillwaterRoute with no wagon mob in the room.
	// Expect Failure (so legacy idle takes over).
	// Implementation hint: existing test infrastructure pattern.
	t.Skip("update once test fixture supports adding/removing wagon mob")
}
```

(Skip-test is fine for now; integration smoke test in Task 21 verifies end-to-end.)

- [ ] **Step 4: Run tests + build.**

Run: `go test ./internal/behaviortree/... && go build ./...`
Expected: green.

- [ ] **Step 5: Commit.**

```bash
git add internal/behaviortree/actions_caravan.go internal/behaviortree/actions_caravan_test.go
git commit -m "$(cat <<'EOF'
feat(btree): wire bidirectional VisitVendorsInRoom in tickRoute

tickRoute looks up wagon (mob 374) in current room and computes
delivery + pickup bucket lists from current state. Stillwater route
delivers thornwall+fernway, picks up stillwater. Thornwall route
delivers stillwater+fernway, picks up thornwall. Wagon items move
in real-time; flavor messages reflect actual transfers.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 9: New btree action `distribute_cargo_to_hostiles`

**Files:**
- Create: `internal/behaviortree/actions_wagon.go`
- Test: `internal/behaviortree/actions_wagon_test.go`

- [ ] **Step 1: Write test.**

`internal/behaviortree/actions_wagon_test.go`:

```go
package behaviortree

import "testing"

func TestActDistributeCargoToHostiles_Registered(t *testing.T) {
	if _, ok := actionRegistry["distribute_cargo_to_hostiles"]; !ok {
		t.Error("distribute_cargo_to_hostiles not registered")
	}
}

// Functional tests skipped — require room + mob fixture + items in
// wagon inventory + hostile mob with HatesAnyGroup. Covered in
// integration smoke test (Task 21).
func TestActDistributeCargoToHostiles_RoundRobin(t *testing.T) {
	t.Skip("integration: requires fixture")
}
func TestActDistributeCargoToHostiles_NoHostilesReturnsFailure(t *testing.T) {
	t.Skip("integration: requires fixture")
}
```

- [ ] **Step 2: Run test.**

Run: `go test ./internal/behaviortree/... -run DistributeCargoToHostiles -v`
Expected: PASS for `Registered` (after Step 3); skips for the others.

- [ ] **Step 3: Implement the action.**

`internal/behaviortree/actions_wagon.go`:

```go
package behaviortree

// actions_wagon.go — Stage 3.4 wagon-specific btree actions.
//
// distribute_cargo_to_hostiles: on wagon mob_death, walks all mobs in
// the room, identifies hostiles (those whose Hates intersects the
// wagon's Groups), and distributes wagon items round-robin into their
// inventories until either the wagon is empty or all hostiles are at
// CarryCapacity. Items that don't fit drop as standard wagon-corpse
// loot via the engine's normal mob-death corpse-creation path.

import (
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func init() {
	actionRegistry["distribute_cargo_to_hostiles"] = actDistributeCargoToHostiles
}

func actDistributeCargoToHostiles(params map[string]any, ctx *EvalContext) Result {
	wagon := mobs.GetInstance(ctx.InstanceId)
	if wagon == nil {
		return Failure
	}
	room := rooms.LoadRoom(ctx.RoomId)
	if room == nil {
		return Failure
	}
	if len(wagon.Character.Items) == 0 {
		return Success // nothing to distribute, but not a failure
	}

	// Find hostile mobs in the room — those whose Hates intersects
	// the wagon's Groups (e.g., bandits with hates: ["caravan"]).
	var hostiles []*mobs.Mob
	for _, instId := range room.GetMobs(rooms.FindAll) {
		if instId == wagon.InstanceId {
			continue
		}
		m := mobs.GetInstance(instId)
		if m == nil {
			continue
		}
		if m.HatesAnyGroup(wagon.Groups) {
			hostiles = append(hostiles, m)
		}
	}
	if len(hostiles) == 0 {
		// No hostiles — items will drop as standard corpse loot via
		// the engine's mob-death path. We didn't transfer; return
		// Failure so any subsequent btree branches can run.
		return Failure
	}

	// Round-robin distribution.
	h := 0
	for i := len(wagon.Character.Items) - 1; i >= 0; i-- {
		item := wagon.Character.Items[i]
		placed := false
		// Try every hostile before giving up on this item.
		for tries := 0; tries < len(hostiles); tries++ {
			target := hostiles[h]
			h = (h + 1) % len(hostiles)
			if target.Character.StoreItem(item) {
				wagon.Character.RemoveItem(item)
				placed = true
				break
			}
		}
		if !placed {
			// All hostiles full; remaining items stay in wagon.Items
			// and drop as standard corpse loot.
			break
		}
	}

	return Success
}
```

- [ ] **Step 4: Run tests + build.**

Run: `go test ./internal/behaviortree/... && go build ./...`
Expected: green (Registered passes, others skip).

- [ ] **Step 5: Commit.**

```bash
git add internal/behaviortree/actions_wagon.go internal/behaviortree/actions_wagon_test.go
git commit -m "$(cat <<'EOF'
feat(btree): distribute_cargo_to_hostiles wagon-death action

Fires on wagon mob_death. Walks room mobs, finds hostiles (those
whose Hates intersects wagon.Groups), distributes wagon items
round-robin into their inventories until wagon is empty or all
hostiles are at CarryCapacity. Leftovers drop as standard corpse
loot via the engine's mob-death path.

Pays off Stage 3.4 design intent: bandits who win the brawl actually
loot the wagon's cargo, making bandit kills mechanically meaningful
for players who arrive after the fight.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 10: Rewrite `tickForagerDeliveringTown` for real items

**Files:**
- Modify: `internal/behaviortree/actions_forager.go` — replace `tickForagerDeliveringTown` body

- [ ] **Step 1: Locate the function.**

Run: `grep -nE "tickForagerDeliveringTown" internal/behaviortree/actions_forager.go`

- [ ] **Step 2: Replace body.**

```go
func tickForagerDeliveringTown(
	p *forager.ForagerProfile,
	mob *mobs.Mob,
	ctx *EvalContext,
) Result {
	idx := getIntFromState(ctx.MobState, keyVisitIndex)
	if idx >= len(p.VendorRooms) {
		transitionForager(ctx.MobState, forager.StateRecalling)
		return Success
	}
	target := p.VendorRooms[idx]
	if ctx.RoomId != target {
		mob.Command(fmt.Sprintf("pathto %d", target))
		return Success
	}

	// Real-item delivery: walk forager satchel; for each item whose
	// bucket is in profile.Buckets, find a matching vendor stock
	// entry and transfer 1 unit (capped at MaxStock).
	deliverForagerSatchel(p, mob, ctx, target)

	ctx.MobState.Set(keyVisitIndex, strconv.Itoa(idx+1))
	return Success
}

func deliverForagerSatchel(
	p *forager.ForagerProfile,
	mob *mobs.Mob,
	ctx *EvalContext,
	roomId int,
) {
	room := rooms.LoadRoom(roomId)
	if room == nil {
		return
	}
	for _, instId := range room.GetMobs(rooms.FindAll) {
		vendor := mobs.GetInstance(instId)
		if vendor == nil || !vendor.HasShop() {
			continue
		}
		shop := shops.GetShopInventory(vendor.Zone, int(vendor.MobId), vendor.HomeRoomId)
		if shop == nil {
			continue
		}
		for i := len(mob.Character.Items) - 1; i >= 0; i-- {
			item := mob.Character.Items[i]
			bucket := economy.BucketFor(item.ItemId)
			if bucket == "" || !slices.Contains(p.Buckets, bucket) {
				continue
			}
			entry := shop.GetStock(item.ItemId)
			if entry == nil || entry.Current >= entry.MaxStock {
				continue
			}
			mob.Character.RemoveItem(item)
			entry.Current++
			room.SendText(fmt.Sprintf(
				`<ansi fg="mobname">%s</ansi> hands a %s to <ansi fg="mobname">%s</ansi>.`,
				p.Name, item.DisplayName(), vendor.Character.Name,
			))
		}
	}
}
```

Add the `slices` and `economy` imports to the import block if not already present.

- [ ] **Step 3: Run forager tests + build.**

Run: `go test ./internal/behaviortree/... && go build ./...`
Expected: green.

- [ ] **Step 4: Commit.**

```bash
git add internal/behaviortree/actions_forager.go
git commit -m "$(cat <<'EOF'
feat(forager): real-item delivery in tickForagerDeliveringTown

Forager (Marsh: Vella, Steppe: Halix) now physically transfers
items from her satchel to vendor stock at each vendor stop. Items
that don't fit (vendor at MaxStock) stay in the satchel for next
vendor / next cycle.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 11: Extend `tickForagerResting` with carry-cap rule

**Files:**
- Modify: `internal/behaviortree/actions_forager.go` — add carry-cap check

- [ ] **Step 1: Locate `tickForagerResting`.**

- [ ] **Step 2: Add carry-cap gate.**

After the existing HP-full check, before the state advance:

```go
// Stage 3.4: stay home if satchel still over rest threshold.
// Vendors didn't absorb much last cycle; foraging more would just
// overflow back to satchel. Narratively: "Vella sits at the temple
// looking content; the merchants don't need more right now."
restThreshold := float64(configs.GetBalanceConfig().ForagerRestCarryThreshold)
if carryRatio(mob) > restThreshold {
	return Failure // continue resting
}
```

- [ ] **Step 3: Test passes for build.**

Run: `go test ./internal/behaviortree/... && go build ./...`
Expected: green.

- [ ] **Step 4: Commit.**

```bash
git add internal/behaviortree/actions_forager.go
git commit -m "$(cat <<'EOF'
feat(forager): rest extension — stay home if satchel still over threshold

When vendors are at MaxStock (saturated economy steady state), the
forager would otherwise burn forever in a "go to vendors, deliver
nothing, come home" loop. This extension keeps her at sanctuary
until vendor demand re-opens space (players buy from vendors).

Threshold is config-driven via ForagerRestCarryThreshold (default 0.5).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 12: Set `rarity_tier` on all mat YAMLs

**Files:**
- Modify: ~50 YAML files in `_datafiles/world/dogmud/items/materials-40000/`

- [ ] **Step 1: Set tier 50 (15 items).**

Add `rarity_tier: 50` to:
- `40001-iron_ingot.yaml`, `40003-wooden_plank.yaml`, `40006-glass_vial.yaml`
- `40012-thread_spool.yaml`, `40013-bone_needle.yaml`, `40014-raw_meat.yaml`
- `40015-wild_vegetables.yaml`, `40016-water_flask.yaml`, `40017-salt_pouch.yaml`
- `40019-chain_link.yaml`, `40043-clay_flask.yaml`, `40044-sealed_phial.yaml`, `40045-crystalline_decanter.yaml`
- `40021-copper_wire.yaml`, `40028-binding_paste.yaml`

Insert as a top-level field in each. Example:

```yaml
itemid: 40001
name: iron ingot
# ... existing fields
rarity_tier: 50
```

- [ ] **Step 2: Set tier 40 (17 items).**

Add `rarity_tier: 40`:
- `40002-leather_strip.yaml`, `40004-healers_root.yaml`, `40005-bitter_thistle.yaml`
- `40007-cloth_strip.yaml`, `40008-spore_sac.yaml`, `40009-dustwalk_herb.yaml`
- `40020-coal_dust.yaml`, `40047-veilbloom_petal.yaml`, `40048-serpent_venom_sac.yaml`
- `40050-putrid_residue.yaml`, `40068-sinew.yaml`
- `40011-hive_fragment.yaml`, `40018-steel_ingot.yaml`, `40022-silver_wire.yaml`
- `40024-polished_stone.yaml`, `40026-gem_dust.yaml`, `40030-chrysalis_setting.yaml`

(Confirm exact filenames with `ls` — names may differ slightly.)

- [ ] **Step 3: Set tier 30 (14 items).**

Add `rarity_tier: 30`:
- `40051-skitter_shrimp_shell.yaml`, `40056-marsh_willow_bark.yaml`, `40057-lake_mint.yaml`
- `40058-freshwater_clam.yaml`, `40059-lake_iron_nodule.yaml`
- `40046-moonpetal.yaml`, `40049-ironbark_shaving.yaml`
- `40062-oak_bark.yaml`, `40063-shadowcap_mushroom.yaml`, `40064-wild_hare_meat.yaml`
- `40065-beeswax.yaml`, `40066-blood_moss.yaml`, `40067-pine_pitch.yaml`
- `40025-raw_gem.yaml`

- [ ] **Step 4: Set tier 20 (5 items).**

Add `rarity_tier: 20`:
- `40053-stillwater_black_pearl.yaml`
- `40010-chrysalis_core.yaml`
- `40027-chrysalis_shard.yaml`
- `40029-mutation_catalyst.yaml`
- `40023-gold_wire.yaml`

- [ ] **Step 5: Verify by spot-checking a few files.**

Run: `head -5 _datafiles/world/dogmud/items/materials-40000/40001-iron_ingot.yaml _datafiles/world/dogmud/items/materials-40000/40010-chrysalis_core.yaml`
Expected: `rarity_tier: 50` and `rarity_tier: 20` respectively.

- [ ] **Step 6: Build + run tests.**

Run: `go build ./... && go test ./internal/items/... ./internal/shops/...`
Expected: green. The previously-skipped EffectiveMaxStock tests for tier=50 should now pass.

- [ ] **Step 7: Commit.**

```bash
git add _datafiles/world/dogmud/items/materials-40000/
git commit -m "$(cat <<'EOF'
feat(items): set rarity_tier on 51 mat YAMLs

Per the audit-matrix tier mapping (Stage 3.4):
  Tier 50 (Common): 15 — Base bucket + copper wire + binding paste
  Tier 40 (Standard): 17 — Mid-tier overlap + most Thornwall refined
  Tier 30 (Regional): 14 — Stillwater (5) + Fernway (8) + raw gem
  Tier 20 (Uncommon): 5 — pearl + 3 chrysalis goods + gold wire

Tier 10 (Ultra-rare) reserved for future content. Quest items
(40031-40042, 40054, 40060, 40061) and defer-to-3.0e items
(40052, 40055) intentionally have no rarity_tier.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 13: Drop explicit `max_stock` from caravan-served vendor YAMLs

**Files:**
- Modify: 17 mob YAMLs in `_datafiles/world/dogmud/mobs/{stillwater,thornwall_city}/`

- [ ] **Step 1: List the targets.**

Stillwater (8): `333-innkeeper_sigrid.yaml`, `336-fishmonger_tov_brann.yaml`, `337-smith_brindle.yaml`, `338-apothecary_ilsa.yaml`, `339-weaver_edda.yaml`, `340-pearl_carver_kess.yaml`, `341-storekeeper_wulf.yaml`, `348-miller_bram.yaml`

Thornwall (9): `97-blacksmith_kerra.yaml`, `98-apothecary_voss.yaml`, `103-food_vendor.yaml`, `104-fence_dealer_siv.yaml`, `108-jeweler_tess.yaml`, `109-enchanter_vael.yaml`, `113-weaver_maren.yaml`, `248-tavern_cook_brynn.yaml`, `273-whisper.yaml`

(Confirm exact filenames in their respective zone folders.)

- [ ] **Step 2: For each YAML, find the `inventory:` block and remove `max_stock:` lines.**

Each StockEntry currently looks like:

```yaml
- itemid: 40001
  quantity: 0
  quantitymax: 0
  restock_qty: 2
  max_stock: 10  # ← REMOVE this line
```

After:

```yaml
- itemid: 40001
  quantity: 0
  quantitymax: 0
  restock_qty: 2
```

The loader's EffectiveMaxStock will now derive MaxStock from `RarityTier × StockMultiplier (1.0)`.

(If the YAML key is `max_stock` vs `maxstock`, confirm by reading 1-2 example files.)

- [ ] **Step 3: Build + boot test.**

Run: `go build ./... && go test ./...`
Expected: green.

- [ ] **Step 4: Commit.**

```bash
git add _datafiles/world/dogmud/mobs/stillwater/ _datafiles/world/dogmud/mobs/thornwall_city/
git commit -m "$(cat <<'EOF'
feat(content): drop explicit max_stock from 17 caravan-served vendors

After Stage 3.4 EffectiveMaxStock wiring, vendor stock caps derive
from item rarity_tier × shopkeeper stock_multiplier (default 1.0).
Removing the per-entry override lets the new system take effect.

Vendors retain their RestockQty and Current values — those are
flow-rate concerns, not cap concerns.

Affected vendors: 8 Stillwater + 9 Thornwall = 17 caravan-served
shops. Non-caravan-served vendors (Sanctum Basin, Dustwalk, etc.)
retain their explicit max_stock as legacy overrides.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 14: Create wagon mob + 2 horses (mobs + behaviors)

**Files:**
- Create: 6 YAML files (3 mob + 3 behavior)

- [ ] **Step 1: Write `mobs/thornwall_city/374-caravan_wagon.yaml`.**

```yaml
mobid: 374
zone: Thornwall City
archetype: ""
behavior_archetype: ""
statpool: 100
itemdropchance: 100
groups:
  - caravan
  - merchant_train
hostile: false
non_combatant: false
player_attack_immune: true
maxwander: -1
activitylevel: 0
charm_immune: true

# Stage 3.4 spawn-time overrides
carry_capacity: 5000
health_max: 1500
stamina_max: 9999
corpse_name: splintered wagon wreckage
corpse_description: |
  Shattered timbers and twisted iron bands lie heaped where
  the supply wagon once stood, the canvas roof torn loose
  and trampled into the dirt. The driver's bench has split
  clean in two; the iron lantern is bent beyond use.
  Scattered among the wreckage, broken crates and split
  sacks lie half-emptied — though the bandits did their
  best to take what they could carry.

character:
  name: a sturdy oak-and-iron supply wagon
  description: |
    A hardwood freight wagon, broad-bedded and shoulder-high,
    its frame banded with cold-forged iron. The bed is roofed
    in tarred canvas stretched tight over hoop-frames, the
    canvas weatherbeaten and patched in three places along
    the seams. A pair of leather-padded yokes at the front
    rig the wagon to its draft team. The wagon's right rear
    wheel has been reset twice — the new spokes are paler
    than the rest. A small iron lantern hangs at the
    driver's bench, unlit by day.
  speciesid: 1
  level: 1
  gold: 0
  stats:
    vitality: {training: 60}
    strength: {training: 10}
    dexterity: {training: 10}
    perception: {training: 5}
    willpower: {training: 5}
    charisma: {training: 10}
```

- [ ] **Step 2: Write `behaviors/thornwall_city/374-caravan_wagon.yaml`.**

```yaml
# Caravan wagon (374) — Stage 3.4.
# Pure follower with a death-handler that distributes cargo to bandits.

tree:
  type: sequence
  children:

    # Always: ensure caravan party (idempotent)
    - type: action
      do: party_ensure_npc_party
      leader_mob_id: 357
      home_room_id: 465

    - type: selector
      children:

        # On death: distribute cargo to hostile mobs in the room
        - type: sequence
          event: mob_death
          children:
            - type: action
              do: distribute_cargo_to_hostiles

        # Default: follow the leader
        - type: sequence
          event: mob_idle
          children:
            - type: action
              do: party_follow_leader
```

- [ ] **Step 3: Write `mobs/thornwall_city/375-hob.yaml`.**

```yaml
mobid: 375
zone: Thornwall City
archetype: fighting
behavior_archetype: ""
statpool: 110
itemdropchance: 0
groups:
  - caravan
  - merchant_train
  - animal
hostile: false
non_combatant: false
player_attack_immune: true
maxwander: -1
activitylevel: 5
charm_immune: true

idlecommands:
  - 'emote stamps a hoof and shakes her mane'
  - ''
  - 'emote nuzzles at the wagon-tongue, looking for handouts'
  - ''

character:
  name: Hob, a dappled-grey draft horse
  description: |
    A solid, broad-shouldered draft horse with a patient
    eye, her coat dappled grey going to white at the muzzle.
    Her hooves are heavy with iron shoes, recently reset.
    The harness across her chest is good leather, oiled and
    supple. A small brass bell hangs from her bridle —
    silent at rest, soft at a walk.
  speciesid: 1
  level: 1
  gold: 0
  stats:
    strength: {training: 35}
    dexterity: {training: 15}
    vitality: {training: 30}
    perception: {training: 15}
    willpower: {training: 10}
    charisma: {training: 5}
  skills:
    weapon-combat: 5
```

- [ ] **Step 4: Write `behaviors/thornwall_city/375-hob.yaml`.**

```yaml
# Hob, draft horse (375) — Stage 3.4.

tree:
  type: sequence
  children:
    - type: action
      do: party_ensure_npc_party
      leader_mob_id: 357
      home_room_id: 465
    - type: selector
      children:
        - type: sequence
          event: mob_hurt
          children:
            - type: action
              do: attack
        - type: sequence
          event: mob_idle
          children:
            - type: action
              do: party_follow_leader
```

- [ ] **Step 5: Write `mobs/thornwall_city/376-bran.yaml`.**

(Same template as Hob, but description differs: bay-coated, slightly younger.)

```yaml
mobid: 376
zone: Thornwall City
archetype: fighting
behavior_archetype: ""
statpool: 110
itemdropchance: 0
groups:
  - caravan
  - merchant_train
  - animal
hostile: false
non_combatant: false
player_attack_immune: true
maxwander: -1
activitylevel: 5
charm_immune: true

idlecommands:
  - 'emote tosses his head, ears flicking forward'
  - ''
  - 'emote paws at the dirt, restless'
  - ''

character:
  name: Bran, a bay draft horse
  description: |
    A heavy-shouldered draft horse, his coat rich bay going
    to black at the legs and mane. Younger than Hob and a
    little restless, his ears flick at every sound on the
    road. The harness across his chest matches Hob's — good
    leather, oiled. His hooves are blacked and shod.
    A small brass bell hangs from his bridle.
  speciesid: 1
  level: 1
  gold: 0
  stats:
    strength: {training: 35}
    dexterity: {training: 18}
    vitality: {training: 28}
    perception: {training: 16}
    willpower: {training: 8}
    charisma: {training: 5}
  skills:
    weapon-combat: 5
```

- [ ] **Step 6: Write `behaviors/thornwall_city/376-bran.yaml`.**

(Identical to Hob's btree.)

```yaml
# Bran, draft horse (376) — Stage 3.4.

tree:
  type: sequence
  children:
    - type: action
      do: party_ensure_npc_party
      leader_mob_id: 357
      home_room_id: 465
    - type: selector
      children:
        - type: sequence
          event: mob_hurt
          children:
            - type: action
              do: attack
        - type: sequence
          event: mob_idle
          children:
            - type: action
              do: party_follow_leader
```

- [ ] **Step 7: Build + boot test.**

Run: `go build ./...`
Expected: clean. (Server boot test happens in Task 15 after spawninfo wiring.)

- [ ] **Step 8: Commit.**

```bash
git add _datafiles/world/dogmud/mobs/thornwall_city/{374-caravan_wagon,375-hob,376-bran}.yaml _datafiles/world/dogmud/behaviors/thornwall_city/{374-caravan_wagon,375-hob,376-bran}.yaml
git commit -m "$(cat <<'EOF'
feat(content): caravan wagon + 2 draft horses (mobs 374-376)

  374 caravan wagon — passive follower; carry_capacity 5000,
                       health_max 1500, stamina_max 9999;
                       corpse_name "splintered wagon wreckage";
                       death-handler distributes cargo to bandits.
  375 Hob (dappled-grey) — pulls the wagon; token combat-back if hit
  376 Bran (bay)         — pulls the wagon; token combat-back if hit

All three are player_attack_immune (rebuff like shopkeepers + caravan
crew). Horses are group: animal so corpses are salvageable per Stage
3.0e.

Wired into the caravan party in the next task.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 15: Update Ketil's btree + room 465 spawninfo

**Files:**
- Modify: `_datafiles/world/dogmud/behaviors/thornwall_city/357-ketil.yaml`
- Modify: `_datafiles/world/dogmud/rooms/thornwall_city/465.yaml`

- [ ] **Step 1: Update Ketil's `party_ensure_npc_party` member list.**

Find the action in `357-ketil.yaml`:

```yaml
- type: action
  do: party_ensure_npc_party
  leader_mob_id: 357
  home_room_id: 465
  member_mob_ids: [358, 359]
```

(Confirm the actual field name — might be `members` or omit if it auto-detects from spawninfo. If the latter, no edit to ketil.yaml is needed.)

If explicit member list, change to:

```yaml
  member_mob_ids: [358, 359, 374, 375, 376]
```

- [ ] **Step 2: Update room 465 spawninfo.**

Find `_datafiles/world/dogmud/rooms/thornwall_city/465.yaml` `spawninfo:` block. After the existing 357/358/359 entries, append:

```yaml
- mobid: 374        # caravan wagon (Stage 3.4)
- mobid: 375        # Hob, draft horse (Stage 3.4)
- mobid: 376        # Bran, draft horse (Stage 3.4)
```

- [ ] **Step 3: Boot the server locally.**

Run the server (per Pre-Push SOP). Watch for:
- `mobs.LoadDataFiles() loadedCount=...` increment by 3
- `rooms.LoadDataFiles()` clean
- `behavior_trees.LoadDataFiles()` clean
- No panics

- [ ] **Step 4: Commit.**

```bash
git add _datafiles/world/dogmud/behaviors/thornwall_city/357-ketil.yaml _datafiles/world/dogmud/rooms/thornwall_city/465.yaml
git commit -m "$(cat <<'EOF'
feat(content): expand caravan party with wagon + horses

Ketil's party_ensure_npc_party member list grows from [358, 359] to
[358, 359, 374, 375, 376]. Room 465 (Market Square Center) spawninfo
gains the 3 new mobs.

Caravan party is now 6 mobs: leader + 2 guards + 2 horses + wagon.
The wagon's btree handles party_follow_leader for cargo movement;
horses follow same.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 16: Remove Chrysalis Core drop from Aberrant Chrysalis

**Files:**
- Modify: `_datafiles/world/dogmud/mobs/sanctum_basin/69-aberrant_chrysalis.yaml`

- [ ] **Step 1: Find and remove the drop entry.**

Read `69-aberrant_chrysalis.yaml`. Find the items / drops block containing `itemid: 40010`. Remove that entry.

If the YAML uses an `items:` block:

```yaml
items:
  - itemid: 40010   # ← REMOVE this entry
    chance: ...
  - itemid: <other>  # ← KEEP others
```

- [ ] **Step 2: Build + test.**

Run: `go build ./... && go test ./...`
Expected: green.

- [ ] **Step 3: Commit.**

```bash
git add _datafiles/world/dogmud/mobs/sanctum_basin/69-aberrant_chrysalis.yaml
git commit -m "$(cat <<'EOF'
fix(content): remove Chrysalis Core drop from Aberrant Chrysalis

Tutorial-tier mob shouldn't drop a tier-20 specialty mat. Chrysalis
Core moves to stone beetle queen (10%) and windscour wyrm (5%) in
the Ironwind Steppe — see next 2 tasks.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 17: Add Chrysalis Core 10% drop to stone beetle queen

**Files:**
- Modify: `_datafiles/world/dogmud/mobs/ironwind_steppe/228-stone_beetle_queen.yaml`

- [ ] **Step 1: Read existing drops block to confirm convention.**

- [ ] **Step 2: Add drop entry.**

In the `items:` (or whatever the drops field is named) block:

```yaml
- itemid: 40010
  chance: 10        # 10% drop rate
```

(Confirm the exact field name + value scale — e.g., `chance: 10` could be a percentage or a roll-on-100 number. Read 1-2 sibling YAMLs for the pattern.)

- [ ] **Step 3: Build + commit.**

```bash
git add _datafiles/world/dogmud/mobs/ironwind_steppe/228-stone_beetle_queen.yaml
git commit -m "$(cat <<'EOF'
feat(content): stone beetle queen drops Chrysalis Core (10%)

Thematic re-source for Stage 3.4 — chrysalis = transformation/
cocoon imagery; brood-mothers yield chrysalis residue. Replaces
the unbalanced tutorial-mob drop.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 18: Add Chrysalis Core 5% drop to windscour wyrm

**Files:**
- Modify: `_datafiles/world/dogmud/mobs/ironwind_steppe/229-windscour_wyrm.yaml`

- [ ] **Step 1: Add drop entry.**

```yaml
- itemid: 40010
  chance: 5        # 5% drop rate
```

- [ ] **Step 2: Commit.**

```bash
git add _datafiles/world/dogmud/mobs/ironwind_steppe/229-windscour_wyrm.yaml
git commit -m "$(cat <<'EOF'
feat(content): windscour wyrm drops Chrysalis Core (5%)

Apex-tier rare source for Stage 3.4. Players who tackle the wyrm
get a small chance at the chrysalis goods. Combined with the
stone beetle queen (10%), Chrysalis Core has two real Ironwind
Steppe sources gating Vael's chrysalis production.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 19: Schema docs + audit matrix + PATCH_NOTES

**Files:**
- Modify: `docs/schemas/mob.md` — document 6 new override fields
- Modify: `docs/schemas/item.md` — document `rarity_tier`
- Modify: `docs/schemas/behavior.md` — document `distribute_cargo_to_hostiles`
- Modify: `docs/economy/mat-audit-matrix.md` — add tier column
- Modify: `PATCH_NOTES.md` — Stage 3.4 entry

- [ ] **Step 1: Update mob.md.**

Add 6 new field rows to the field reference table:

```markdown
| `carry_capacity`     | float | no | (Stage 3.4) Override Strength-derived carry capacity calc. Used by special mobs (wagons) where the default doesn't fit. |
| `health_max`         | int   | no | (Stage 3.4) Override Vitality-derived max HP. |
| `stamina_max`        | int   | no | (Stage 3.4) Override default max SP. |
| `corpse_name`        | string| no | (Stage 3.4) Override "<Name> corpse" rendering. |
| `corpse_description` | string| no | (Stage 3.4) Override default corpse look-text. |
| `stock_multiplier`   | float | no | (Stage 3.4) Shop-stock-cap scale; default 1.0. EffectiveMaxStock = item.RarityTier × stock_multiplier. |
```

- [ ] **Step 2: Update item.md.**

```markdown
| `rarity_tier` | int | no | (Stage 3.4) Vendor-stock cap tier (50/40/30/20/10). Higher = more common. EffectiveMaxStock = rarity_tier × shopkeeper.stock_multiplier. 0 = untiered (quest items). |
```

- [ ] **Step 3: Update behavior.md.**

Add to the wagon-specific actions section:

```markdown
### Action: distribute_cargo_to_hostiles

Stage 3.4. Fires on `mob_death` for the wagon (mob 374). Walks all
mobs in the room, finds those whose `Hates` intersects the wagon's
`Groups`, and distributes wagon items round-robin into their
inventories until either the wagon is empty or all hostiles are at
their `CarryCapacity`. Items that don't fit drop as standard wagon-
corpse loot.
```

- [ ] **Step 4: Update mat-audit-matrix.md with tier column.**

Add `Tier` column to the audit table; populate per the tier mapping in this plan.

- [ ] **Step 5: Append Stage 3.4 entry to PATCH_NOTES.md.**

```markdown
## 2026-04-30 — Stage 3.4: Real Item Transfer (dev only)

**Note:** Final stage of the caravan/economy effort. Once this lands
on `development`, the entire economy stack (Stages 3.0b through 3.4)
promotes to `master` as a coherent update.

- The caravan now physically hauls items: a new wagon mob (374) with
  ~5000 carry capacity rides with the caravan party. `look wagon`
  shows the actual cargo. Two draft horses (Hob, Bran) pull it.
- Wagon dies if the caravan is wiped at the bandit camp; cargo is
  distributed to bandit inventories (round-robin), with leftovers as
  wreckage corpse loot. Players who kill the bandits afterward get
  the cargo.
- Wagon corpse renders as "splintered wagon wreckage" with custom
  description.
- Vendor stock caps now derive from item `rarity_tier` (50/40/30/20/10)
  × shopkeeper `stock_multiplier` (default 1.0). Per-vendor max_stock
  overrides removed for the 17 caravan-served vendors. Future big-city
  shops can set stock_multiplier > 1.0 for proportionally larger stock.
- Foragers now physically deliver items from their satchels to vendor
  inventories (no more abstract RestockBuckets). Items that don't fit
  stay in the satchel for next vendor / next cycle.
- New forager rest extension: when carry > 50% on return home,
  forager stays at sanctuary instead of cycling back out. Prevents
  futile loops in saturated economies.
- Caravan vendor stops are now BIDIRECTIONAL — caravan delivers
  items it brought AND picks up items the local vendors produce in
  abundance, hauling them across town. Pays off the "wholesalers
  seeking arbitrage" worldbuilding from the Stage 2 caravan.
- Chrysalis Core (40010) re-sourced: removed from Aberrant Chrysalis
  in Sanctum Basin tutorial. Now drops 10% from stone beetle queen
  and 5% from windscour wyrm in Ironwind Steppe.
- 6 new mob override fields: carry_capacity, health_max, stamina_max,
  corpse_name, corpse_description, stock_multiplier.
- New btree action `distribute_cargo_to_hostiles` for the wagon's
  death handler.
- New config knob `ForagerRestCarryThreshold` (default 0.5) for the
  rest extension.
- ItemSpec gains `rarity_tier` field; mat YAMLs (51 of them) now
  carry the tier per the Stage 3.0b audit matrix.
```

- [ ] **Step 6: Commit.**

```bash
git add docs/schemas/mob.md docs/schemas/item.md docs/schemas/behavior.md docs/economy/mat-audit-matrix.md PATCH_NOTES.md
git commit -m "$(cat <<'EOF'
docs(3.4): schema docs + audit matrix tier column + PATCH_NOTES

  - mob.md: 6 new override fields documented
  - item.md: rarity_tier documented
  - behavior.md: distribute_cargo_to_hostiles action documented
  - mat-audit-matrix: tier column added per the agreed mapping
  - PATCH_NOTES: full Stage 3.4 dev-only entry covering wagon, draft
    horses, real item transfer, rarity tiers, forager rest extension,
    bidirectional vendor visits, and Chrysalis Core re-source

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Self-review checklist (run at end)

- [ ] **Spec coverage check.** Walk each section of the spec:
  - Architecture overview — Tasks 1-11
  - Wagon mob — Task 14
  - Draft horses — Task 14
  - Caravan party expansion — Task 15
  - Engine override fields — Tasks 1-4
  - RarityTier system — Tasks 1, 5, 12-13
  - Bidirectional vendor visit — Tasks 7, 8
  - Forager → vendor real-item delivery — Task 10
  - Forager rest extension — Tasks 6, 11
  - Wagon death distribution — Task 9
  - Caravan death recovery — handled by existing systems (Stage 2 + Task 9 wagon-death wired)
  - Chrysalis Core re-source — Tasks 16-18
  - Edge cases — covered in spec; no separate task; smoke test catches
  - Testing strategy — embedded in each task + Task 21 end-to-end smoke
  - Out of scope — preserved as is

- [ ] **No placeholders** (no "TBD"/"TODO"/"implement later" in tasks).

- [ ] **Type/method consistency:**
  - `EffectiveMaxStock(itemId int, mob *mobs.Mob) int` — defined Task 5, used Tasks 5/13
  - `ApplyMobOverrides(c *Character, healthMax, staminaMax int, carryCapacity float64)` — defined Task 3, used Task 3
  - `corpseDisplayName(*mobs.Mob) string` — defined Task 4, used Task 4
  - `VisitVendorsInRoom(roomId int, wagon *mobs.Mob, deliveryBuckets, pickupBuckets []string) (delivered, pickedUp []ItemMove)` — defined Task 7, used Task 8
  - `actDistributeCargoToHostiles` action key — defined Task 9, used in Task 14 wagon btree

- [ ] **Run after every task:** `go test ./... && go build ./...`

---

## Final verification (after all tasks)

1. `go test ./...` full suite green
2. Server boot clean (per Pre-Push SOP):
   - `mobs.LoadDataFiles loadedCount=N+3` (gained wagon + 2 horses)
   - `items.LoadDataFiles` clean (51 mat YAMLs gained `rarity_tier`)
   - `rooms.LoadDataFiles` clean
   - `behavior_trees.LoadDataFiles` clean (3 new btree files)
3. **In-game smoke test (13-step sequence from spec Section 5):**
   - Caravan party 6-mob at Thornwall depot
   - `look wagon` shows description + cargo
   - Halix delivers to Thornwall vendors directly
   - Caravan transit to Stillwater; wagon empty until Fernway pickup
   - Fernway forager handoff (wagon items grow)
   - At each Stillwater vendor: deliver fernway, pick up stillwater (flavor messages, `look wagon` shifts)
   - Stillwater dwell + Vella delivery
   - Caravan inbound; Thornwall vendor visits (deliver stillwater + fernway, pick up thornwall)
   - Brindle has fewer lake-iron than start of cycle, Kerra has more
   - Caravan wipe at 4052: wagon dies, cargo distributes to bandits, kill bandits → loot
   - Wagon corpse renders as "splintered wagon wreckage"
   - Capacity smoke: 5+ cycles without players, wagon accumulates but well below cap, Vella's rest extension fires
   - Chrysalis Core: Aberrant Chrysalis no drop; stone beetle queen ~10%; wyrm ~5%

---

## Execution handoff

**Plan complete.**

Two execution options:

1. **Subagent-Driven (recommended)** — fresh subagent per task, two-stage review (spec + quality), tight iteration.
2. **Inline Execution** — execute tasks here using the executing-plans skill, batched with checkpoints.

Which approach?
