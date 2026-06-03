# Mob Instance Goal-Progress Persistence Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Persist a mob's gold, equipment, and planner working state across the by-design performance despawn so strategic-goal progress survives respawn.

**Architecture:** Extend the existing `MobInstanceData` instance-save struct with three nil-able fields (`Gold *int`, `Equipment *characters.Worn`, `PlanState map[string]any`), broaden the save gate from `hasProgression` to `hasPersistableState` (template-diff), capture the new fields in `SaveMobInstance`, overlay them on restore in `NewMobById`, and add a `SaveMobInstance` call to `removeRoomFromMemory` so a purchase made since the last periodic save isn't lost on room unload. Death/admin-despawn paths are untouched (they still `DeleteMobInstance` = reset).

**Tech Stack:** Go, `gopkg.in/yaml.v2`, testify. Packages: `internal/mobs`, `internal/rooms`, `internal/characters`, `internal/items`.

**Spec:** `docs/superpowers/specs/2026-06-01-mob-instance-goal-progress-persistence-design.md`

---

## File Structure

- `internal/mobs/instance_save.go` — struct fields, capture logic, `collectPlanState`, `hasPersistableState`, `equipmentDiffers`, `planKeyPrefix` const, new imports. (Existing file; the home of all instance-save logic.)
- `internal/mobs/mobs.go` — restore overlay inside the existing `savedInstance != nil` branch (~line 437).
- `internal/rooms/roommanager.go` — save-at-despawn in `removeRoomFromMemory` (both destroy loops).
- `internal/mobs/instance_save_test.go` — new unit tests for the helpers + struct round-trip.
- `internal/mobs/mobs_test.go` — restore round-trip test (mirrors existing `TestNewMobById_StillLoadsInstanceFile`).
- `internal/rooms/roommanager_test.go` — save-at-despawn test (mirrors existing `TestRoomHasEssentialMob_NonEssentialUnloads`).

## Background the engineer needs

- **`MobInstanceData`** (`internal/mobs/instance_save.go:18-30`) currently holds only training/skills/mutations. `SaveMobInstance` (line 54) early-returns unless `MobProgressionEnabled` AND `hasProgression(mob)`. `LoadMobInstance` (line 115) unmarshals the file; the restore overlay that applies it lives in `mobs.go` (NOT in `LoadMobInstance`).
- **Restore overlay** is at `internal/mobs/mobs.go:417-437`, inside `NewMobById`, in the `if savedInstance != nil {` branch.
- **`GetMobSpec(mobId)`** (`mobs.go:643`) returns a *copy* of the template mob — use it for the save gate's template comparison.
- **`characters.Worn`** (`characters/worn.go:8`) is a flat struct of YAML-tagged `items.Item` slots. `items.Item` (`items/items.go:40`) carries enchant state (`enchanttier`, `enchanttype`, `enchantuses`). `UUID` is `yaml:"-"` (not serialized). A marshal round-trip preserves the loadout.
- **Planner working state** uses the `plan:` MiscData prefix (`planners.PlanKeyPrefix`). We CANNOT import `planners` from `mobs` (cycle), so a local `planKeyPrefix = "plan:"` const is defined in `mobs` with a comment that it must stay in sync.
- **Pointer presence semantics**: `Gold *int` / `Equipment *characters.Worn` distinguish "absent in save" (nil → leave template value) from a real zero (`Gold = 0` spent everything / empty `Worn` stripped gear). `yaml`'s `omitempty` on a pointer omits only when nil, so a non-nil `*int(0)` still emits `gold: 0` and round-trips. Old training-only save files have all three nil → restore overlays nothing → behavior identical to today (transparent migration).
- **Test helpers**: `mobs` package has `withMobProgressionEnabled(t)`, `seedRegistry()`, `NewMobById(mobId, homeRoomId)`, `instancePath(...)`. `rooms` package has `SeedRoomsForTest`, `mobs.SeedMobsForTest`, `mobs.SetInstanceForTest`. `configs.AddOverlayOverrides(map[string]any{...})` toggles config for a test.
- **Run all package tests** with: `go test ./internal/mobs/... ./internal/rooms/...` from the repo root.

---

### Task 1: Add goal-progress fields to `MobInstanceData`

**Files:**
- Modify: `internal/mobs/instance_save.go` (struct at lines 18-30, imports at lines 3-13)
- Test: `internal/mobs/instance_save_test.go` (append)

- [ ] **Step 1: Write the failing test**

Append to `internal/mobs/instance_save_test.go`:

```go
func TestMobInstanceData_GoalProgressFields_RoundTrip(t *testing.T) {
	gold := 999
	in := MobInstanceData{
		Gold: &gold,
		Equipment: &characters.Worn{
			Body: items.Item{ItemId: 1, EnchantTier: 3, EnchantType: "frost"},
		},
		PlanState: map[string]any{"plan:wealth-gold:target_shop_room": 4101},
	}

	bytes, err := yaml.Marshal(&in)
	assert.NoError(t, err)

	var out MobInstanceData
	assert.NoError(t, yaml.Unmarshal(bytes, &out))

	assert.NotNil(t, out.Gold)
	assert.Equal(t, 999, *out.Gold)
	assert.NotNil(t, out.Equipment)
	assert.Equal(t, 1, out.Equipment.Body.ItemId)
	assert.Equal(t, 3, out.Equipment.Body.EnchantTier)
	assert.Equal(t, "frost", out.Equipment.Body.EnchantType)
	assert.Equal(t, 4101, out.PlanState["plan:wealth-gold:target_shop_room"])
}

func TestMobInstanceData_GoldZero_RoundTrips(t *testing.T) {
	zero := 0
	in := MobInstanceData{Gold: &zero}
	b, err := yaml.Marshal(&in)
	assert.NoError(t, err)

	var out MobInstanceData
	assert.NoError(t, yaml.Unmarshal(b, &out))
	assert.NotNil(t, out.Gold, "non-nil *int(0) must survive marshal (presence semantics)")
	assert.Equal(t, 0, *out.Gold)
}
```

This requires the `instance_save_test.go` imports to include `characters`, `items`, `yaml`, and testify `assert`. Add any missing ones to that file's import block:

```go
import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/mobs/ -run TestMobInstanceData_GoalProgress -v`
Expected: FAIL — compile error, `Gold`/`Equipment`/`PlanState` are not fields of `MobInstanceData`.

- [ ] **Step 3: Add the fields and imports**

In `internal/mobs/instance_save.go`, change the import block (lines 3-13) to add `bytes`, `strings`, and `characters`:

```go
import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v2"
)
```

Add the three fields to the `MobInstanceData` struct (after `MutationProgress` at line 29):

```go
	MutationProgress   float64        `yaml:"mutation_progress,omitempty"`

	// Goal-progress persistence (2026-06-01). Pointers / nil-able so that
	// "absent in the save" (old file or non-goal mob) is distinguishable
	// from a real zero value (mob spent all gold / stripped all gear).
	Gold      *int             `yaml:"gold,omitempty"`
	Equipment *characters.Worn `yaml:"equipment,omitempty"`
	PlanState map[string]any   `yaml:"plan_state,omitempty"`
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/mobs/ -run TestMobInstanceData_GoalProgress -v`
Expected: PASS (both round-trip tests).

- [ ] **Step 5: Commit**

```bash
git add internal/mobs/instance_save.go internal/mobs/instance_save_test.go
git commit -m "feat(mobs): add gold/equipment/plan-state fields to MobInstanceData"
```

---

### Task 2: `collectPlanState` helper

**Files:**
- Modify: `internal/mobs/instance_save.go`
- Test: `internal/mobs/instance_save_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/mobs/instance_save_test.go`:

```go
func TestCollectPlanState_OnlyPlanPrefixedKeys(t *testing.T) {
	mob := &Mob{}
	mob.Character.MiscData = map[string]any{
		"plan:wealth-gold:target_shop_room": 4101,
		"plan:upgrade-gear:worst_slot":      "body",
		"conversation_line_idx":             2,
		"faction_kills:bandits":             3,
	}

	got := collectPlanState(mob)
	assert.Len(t, got, 2)
	assert.Equal(t, 4101, got["plan:wealth-gold:target_shop_room"])
	assert.Equal(t, "body", got["plan:upgrade-gear:worst_slot"])
	_, hasNonPlan := got["conversation_line_idx"]
	assert.False(t, hasNonPlan)
}

func TestCollectPlanState_NilMiscData(t *testing.T) {
	mob := &Mob{}
	assert.Nil(t, collectPlanState(mob))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/mobs/ -run TestCollectPlanState -v`
Expected: FAIL — `collectPlanState` undefined.

- [ ] **Step 3: Implement the helper**

Add to `internal/mobs/instance_save.go` (near `hasProgression`, at the end of the file):

```go
// planKeyPrefix MUST match planners.PlanKeyPrefix. It is duplicated here
// rather than imported because internal/planners imports internal/mobs;
// referencing it the other way would form an import cycle.
const planKeyPrefix = "plan:"

// collectPlanState returns a copy of every "plan:"-prefixed MiscData entry
// on the mob — the planners' working state (target shop room, worst slot,
// etc.). Returns nil if the mob has no MiscData or no plan keys.
func collectPlanState(mob *Mob) map[string]any {
	if mob.Character.MiscData == nil {
		return nil
	}
	out := map[string]any{}
	for k, v := range mob.Character.MiscData {
		if strings.HasPrefix(k, planKeyPrefix) {
			out[k] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/mobs/ -run TestCollectPlanState -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/mobs/instance_save.go internal/mobs/instance_save_test.go
git commit -m "feat(mobs): collectPlanState helper for plan:-prefixed MiscData"
```

---

### Task 3: `equipmentDiffers` helper

**Files:**
- Modify: `internal/mobs/instance_save.go`
- Test: `internal/mobs/instance_save_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/mobs/instance_save_test.go`:

```go
func TestEquipmentDiffers(t *testing.T) {
	a := characters.Worn{Body: items.Item{ItemId: 1}}
	b := characters.Worn{Body: items.Item{ItemId: 1}}
	assert.False(t, equipmentDiffers(a, b), "identical loadouts must not differ")

	c := characters.Worn{Body: items.Item{ItemId: 2}}
	assert.True(t, equipmentDiffers(a, c), "different itemId in a slot must differ")

	// Enchant tier change on the same item must register as a difference.
	d := characters.Worn{Body: items.Item{ItemId: 1, EnchantTier: 1}}
	assert.True(t, equipmentDiffers(a, d), "different enchant tier must differ")

	// UUID must NOT count as a difference (it is yaml:"-").
	e := characters.Worn{Body: items.New(1)}
	f := characters.Worn{Body: items.New(1)}
	assert.False(t, equipmentDiffers(e, f), "differing UUIDs alone must not register as a difference")
}
```

Note: `items.New(1)` assigns a fresh random `UUID` each call; the test proves UUID is excluded from the comparison. `items.New` requires item spec 1 to be registered — `seedRegistry()` is not needed here because `items.New` falls back to a bare `Item{ItemId:1}` shape whose marshaled bytes match regardless of spec. If `items.New(1)` returns a zero Item because spec 1 is absent in the test binary, both `e` and `f` are still equal, so the assertion holds.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/mobs/ -run TestEquipmentDiffers -v`
Expected: FAIL — `equipmentDiffers` undefined.

- [ ] **Step 3: Implement the helper**

Add to `internal/mobs/instance_save.go` (after `collectPlanState`):

```go
// equipmentDiffers reports whether two worn loadouts differ in any
// persistent field. It compares marshaled YAML bytes: items.Item.UUID is
// yaml:"-" (excluded) and the unexported tempDataStore is not marshaled,
// so this is a value comparison that ignores per-instance identity and
// correctly detects a changed itemId or enchant tier in any slot.
func equipmentDiffers(a, b characters.Worn) bool {
	ab, _ := yaml.Marshal(&a)
	bb, _ := yaml.Marshal(&b)
	return !bytes.Equal(ab, bb)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/mobs/ -run TestEquipmentDiffers -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/mobs/instance_save.go internal/mobs/instance_save_test.go
git commit -m "feat(mobs): equipmentDiffers YAML-byte comparison helper"
```

---

### Task 4: `hasPersistableState` gate

**Files:**
- Modify: `internal/mobs/instance_save.go`
- Test: `internal/mobs/instance_save_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/mobs/instance_save_test.go`:

```go
// clearProgression zeroes every field hasProgression looks at, plus
// MiscData. NewMobById with no saved instance RANDOMIZES the stat pool
// into training (mobs.go else-branch), so each gate sub-case must start
// from a clean slate to isolate the specific path under test.
func clearProgression(m *Mob) {
	m.Character.Stats.Strength.Training = 0
	m.Character.Stats.Dexterity.Training = 0
	m.Character.Stats.Perception.Training = 0
	m.Character.Stats.Vitality.Training = 0
	m.Character.Stats.Willpower.Training = 0
	m.Character.Stats.Charisma.Training = 0
	m.Character.Skills = nil
	m.Character.SkillUseCount = nil
	m.Character.StatUseCount = nil
	m.Character.Mutations = nil
	m.Character.MutationProgress = 0
	m.Character.MiscData = nil
}

func TestHasPersistableState(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	// Clean mob: no progression, gold == template, equipment == template,
	// no plan state → not persistable. Gold/equipment are copied verbatim
	// from the template at construction, so they match GetMobSpec(1).
	base := NewMobById(1, 100)
	if base == nil {
		t.Fatal("NewMobById returned nil")
	}
	clearProgression(base)
	assert.False(t, hasPersistableState(base), "clean template-equal mob must not be persistable")

	// Gold change alone trips the gate.
	goldMob := NewMobById(1, 100)
	clearProgression(goldMob)
	goldMob.Character.Gold = goldMob.Character.Gold + 500
	assert.True(t, hasPersistableState(goldMob), "gold change must be persistable")

	// Plan state alone trips the gate.
	planMob := NewMobById(1, 100)
	clearProgression(planMob)
	planMob.Character.SetMiscData("plan:wealth-gold:target_shop_room", 4101)
	assert.True(t, hasPersistableState(planMob), "plan state must be persistable")

	// Equipment change alone trips the gate.
	eqMob := NewMobById(1, 100)
	clearProgression(eqMob)
	eqMob.Character.Equipment.Body = items.Item{ItemId: 99999, EnchantTier: 5}
	assert.True(t, hasPersistableState(eqMob), "equipment change must be persistable")

	// Training alone still trips the gate (existing hasProgression path).
	trainMob := NewMobById(1, 100)
	clearProgression(trainMob)
	trainMob.Character.Stats.Strength.Training = 5
	assert.True(t, hasPersistableState(trainMob), "training must remain persistable")
}
```

Note: the `base` false-case assumes `NewMobById` copies gold + equipment verbatim from template 1 (it does — only the stat pool is randomized, and `clearProgression` removes that). `GetMobSpec(1)` returns non-nil because `seedRegistry` registers template 1.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/mobs/ -run TestHasPersistableState -v`
Expected: FAIL — `hasPersistableState` undefined.

- [ ] **Step 3: Implement the gate**

Add to `internal/mobs/instance_save.go` (after `equipmentDiffers`):

```go
// hasPersistableState reports whether a mob has any state worth saving to
// its instance file: stat/skill/mutation progression (hasProgression),
// planner working state, gold that differs from its template, or an
// equipment loadout that differs from its template. The template
// comparison keeps the gate meaningful — without it every mob in the
// world (all of which carry template gold/equipment) would write a file
// every save interval.
func hasPersistableState(mob *Mob) bool {
	if hasProgression(mob) {
		return true
	}
	if collectPlanState(mob) != nil {
		return true
	}
	tmpl := GetMobSpec(mob.MobId)
	if tmpl == nil {
		return false
	}
	if mob.Character.Gold != tmpl.Character.Gold {
		return true
	}
	return equipmentDiffers(mob.Character.Equipment, tmpl.Character.Equipment)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/mobs/ -run TestHasPersistableState -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/mobs/instance_save.go internal/mobs/instance_save_test.go
git commit -m "feat(mobs): hasPersistableState gate (template-diff on gold/equipment)"
```

---

### Task 5: Capture new fields in `SaveMobInstance` + swap the gate

**Files:**
- Modify: `internal/mobs/instance_save.go` (`SaveMobInstance`, lines 54-111)
- Test: `internal/mobs/instance_save_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/mobs/instance_save_test.go`:

```go
func TestSaveMobInstance_CapturesGoldEquipmentPlanState(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()
	withMobProgressionEnabled(t)

	mob := NewMobById(1, 100)
	if mob == nil {
		t.Fatal("NewMobById returned nil")
	}
	path := instancePath(mob.MobId, mob.Zone, mob.Character.Name, mob.HomeRoomId)
	_ = os.Remove(path)
	t.Cleanup(func() { _ = os.Remove(path) })

	mob.Character.Gold = 1234
	mob.Character.Equipment.Body = items.Item{ItemId: 1, EnchantTier: 2}
	mob.Character.SetMiscData("plan:wealth-gold:target_shop_room", 4101)

	assert.NoError(t, SaveMobInstance(mob))

	loaded := LoadMobInstance(mob.MobId, mob.Zone, mob.Character.Name, mob.HomeRoomId)
	assert.NotNil(t, loaded)
	assert.NotNil(t, loaded.Gold)
	assert.Equal(t, 1234, *loaded.Gold)
	assert.NotNil(t, loaded.Equipment)
	assert.Equal(t, 1, loaded.Equipment.Body.ItemId)
	assert.Equal(t, 2, loaded.Equipment.Body.EnchantTier)
	assert.Equal(t, 4101, loaded.PlanState["plan:wealth-gold:target_shop_room"])
}

func TestSaveMobInstance_GoldChangeAloneWritesFile(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()
	withMobProgressionEnabled(t)

	mob := NewMobById(1, 100)
	path := instancePath(mob.MobId, mob.Zone, mob.Character.Name, mob.HomeRoomId)
	_ = os.Remove(path)
	t.Cleanup(func() { _ = os.Remove(path) })

	// No training — only a gold change. Old gate (hasProgression) would skip.
	mob.Character.Gold = mob.Character.Gold + 777
	assert.NoError(t, SaveMobInstance(mob))

	_, statErr := os.Stat(path)
	assert.NoError(t, statErr, "gold-change-only mob must now write an instance file")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/mobs/ -run TestSaveMobInstance_CapturesGold -v`
Expected: FAIL — saved file has nil `Gold`/`Equipment`/`PlanState` (not yet captured); and the gold-change-only test fails because `hasProgression` skips the write.

- [ ] **Step 3: Wire capture + swap the gate**

In `internal/mobs/instance_save.go`, change the gate in `SaveMobInstance` (lines 65-68) from:

```go
	// Only save if the mob has gained any training beyond zero
	if !hasProgression(mob) {
		return nil
	}
```

to:

```go
	// Only save if the mob has progression, planner working state, or
	// gold/equipment that differs from its template.
	if !hasPersistableState(mob) {
		return nil
	}
```

Then, after the existing `if len(mob.Character.Mutations) > 0 { ... }` block (line 91) and before `savePath := ...` (line 93), add the capture:

```go
	// Goal-progress capture (2026-06-01). Captured unconditionally for a
	// persistable mob — the live value IS the truth, so capturing even when
	// it equals the template is harmless (restore becomes a no-op). Pointers
	// preserve the spent-all-gold (0) and stripped-gear (empty) cases.
	gold := mob.Character.Gold
	data.Gold = &gold

	eq := mob.Character.Equipment
	data.Equipment = &eq

	if planState := collectPlanState(mob); planState != nil {
		data.PlanState = planState
	}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/mobs/ -run TestSaveMobInstance -v`
Expected: PASS (new tests + existing `TestSaveMobInstance_CharmedMobSkipsWrite` still passes — the charmed guard is above the gate).

- [ ] **Step 5: Run the full mobs package to check for regressions**

Run: `go test ./internal/mobs/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/mobs/instance_save.go internal/mobs/instance_save_test.go
git commit -m "feat(mobs): capture gold/equipment/plan-state in SaveMobInstance"
```

---

### Task 6: Restore overlay in `NewMobById`

**Files:**
- Modify: `internal/mobs/mobs.go` (the `savedInstance != nil` branch, ~lines 417-437)
- Test: `internal/mobs/mobs_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/mobs/mobs_test.go` (mirrors `TestNewMobById_StillLoadsInstanceFile` at line 1428):

```go
func TestNewMobById_RestoresGoldEquipmentPlanState(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()
	withMobProgressionEnabled(t)

	seed := NewMobById(1, 100)
	if seed == nil {
		t.Fatal("NewMobById returned nil")
	}
	seed.Character.Gold = 4242
	seed.Character.Equipment.Body = items.Item{ItemId: 1, EnchantTier: 4}
	seed.Character.SetMiscData("plan:upgrade-gear:worst_slot", "body")
	if err := SaveMobInstance(seed); err != nil {
		t.Fatalf("seed SaveMobInstance: %v", err)
	}
	path := instancePath(seed.MobId, seed.Zone, seed.Character.Name, seed.HomeRoomId)
	t.Cleanup(func() { _ = os.Remove(path) })

	got := NewMobById(1, 100)
	if got == nil {
		t.Fatal("NewMobById returned nil")
	}
	assert.Equal(t, 4242, got.Character.Gold, "gold must be restored")
	assert.Equal(t, 1, got.Character.Equipment.Body.ItemId, "equipped item must be restored")
	assert.Equal(t, 4, got.Character.Equipment.Body.EnchantTier, "enchant tier must be restored")
	assert.Equal(t, "body", got.Character.GetMiscData("plan:upgrade-gear:worst_slot"), "plan state must be restored")
}

func TestNewMobById_OldFormatSaveLeavesTemplateGold(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()
	withMobProgressionEnabled(t)

	// Simulate an OLD-format save: training only, no gold/equipment/plan
	// fields. Write it via a MobInstanceData with the new fields left nil.
	seed := NewMobById(1, 100)
	templateGold := seed.Character.Gold
	seed.Character.Stats.Strength.Training = 12 // trips hasProgression
	if err := SaveMobInstance(seed); err != nil {
		t.Fatalf("seed SaveMobInstance: %v", err)
	}
	path := instancePath(seed.MobId, seed.Zone, seed.Character.Name, seed.HomeRoomId)
	t.Cleanup(func() { _ = os.Remove(path) })

	// Hand-rewrite the file to strip the gold/equipment/plan fields,
	// emulating a file written before this feature existed.
	old := MobInstanceData{StrengthTraining: 12}
	b, _ := yaml.Marshal(&old)
	if err := os.WriteFile(path, b, 0644); err != nil {
		t.Fatalf("rewrite old-format file: %v", err)
	}

	got := NewMobById(1, 100)
	assert.Equal(t, 12, got.Character.Stats.Strength.Training, "training restores")
	assert.Equal(t, templateGold, got.Character.Gold, "old-format save must leave template gold untouched")
}
```

Ensure `mobs_test.go` imports `items` and `yaml` (it already uses both elsewhere — `items.Item` appears at line 1269, `yaml` at line 1298 — so no import change needed).

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/mobs/ -run TestNewMobById_RestoresGold -v`
Expected: FAIL — gold/equipment/plan are not overlaid on restore (assert mismatches).

- [ ] **Step 3: Add the overlay**

In `internal/mobs/mobs.go`, inside the `if savedInstance != nil {` branch, after `mob.Character.MutationProgress = savedInstance.MutationProgress` (line 437) and before the closing `}` of that branch (line 438), add:

```go
			mob.Character.MutationProgress = savedInstance.MutationProgress

			// Goal-progress restore (2026-06-01). Each guarded by presence:
			// nil means the field was absent in the save (old-format file or
			// a non-goal mob) — leave the template value untouched.
			if savedInstance.Gold != nil {
				mob.Character.Gold = *savedInstance.Gold
			}
			if savedInstance.Equipment != nil {
				mob.Character.Equipment = *savedInstance.Equipment
			}
			if savedInstance.PlanState != nil {
				if mob.Character.MiscData == nil {
					mob.Character.MiscData = map[string]any{}
				}
				for k, v := range savedInstance.PlanState {
					mob.Character.MiscData[k] = v
				}
			}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/mobs/ -run TestNewMobById -v`
Expected: PASS (new restore tests + existing `TestNewMobById_StillLoadsInstanceFile` / `TestNewMobByIdFresh_IgnoresInstanceFile`).

- [ ] **Step 5: Run the full mobs package**

Run: `go test ./internal/mobs/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/mobs/mobs.go internal/mobs/mobs_test.go
git commit -m "feat(mobs): restore gold/equipment/plan-state in NewMobById overlay"
```

---

### Task 7: Save-at-despawn in `removeRoomFromMemory`

**Files:**
- Modify: `internal/rooms/roommanager.go` (`removeRoomFromMemory`, lines 496-510)
- Test: `internal/rooms/roommanager_test.go`

- [ ] **Step 1: Write the failing test**

Add `"github.com/GoMudEngine/GoMud/internal/configs"` to the import block of `internal/rooms/roommanager_test.go` (it is not currently imported there; the other identifiers — `mobs`, `characters`, `util`, `assert` — already are).

Append to `internal/rooms/roommanager_test.go` (mirrors `TestRoomHasEssentialMob_NonEssentialUnloads`):

```go
func TestRemoveRoomFromMemory_SavesChangedMob(t *testing.T) {
	cleanupMobs := mobs.SeedMobsForTest(map[int]*mobs.Mob{}, map[int]*mobs.Mob{})
	defer cleanupMobs()

	// Enable mob progression so SaveMobInstance writes.
	configs.AddOverlayOverrides(map[string]any{"Balance.MobProgressionEnabled": true})
	t.Cleanup(func() {
		configs.AddOverlayOverrides(map[string]any{"Balance.MobProgressionEnabled": false})
	})

	const instId = 2002
	mob := &mobs.Mob{
		MobId:      50,
		Zone:       "TestZone",
		HomeRoomId: 9100,
		Groups:     []string{"bandit"}, // non-essential → room unloads
	}
	mob.Character.Name = "Testmob"
	mob.Character.Stats.Strength.Training = 1 // trips hasPersistableState
	mob.Character.Gold = 999
	mobs.SetInstanceForTest(instId, mob)

	cleanupRooms := SeedRoomsForTest(
		map[int]*Room{
			9100: {
				RoomId:      9100,
				Zone:        "TestZone",
				Title:       "Dark Alley",
				Description: "A shadowy alley.",
				mobs:        []int{instId},
				lastVisited: 0,
			},
		},
		map[string]*ZoneConfig{
			"TestZone": {Name: "TestZone", RoomId: 9100, RoomIds: map[int]struct{}{9100: {}}},
		},
	)
	defer cleanupRooms()

	// Clean any stale save, and ensure cleanup after.
	mobs.DeleteMobInstance(50, "TestZone", "Testmob", 9100)
	t.Cleanup(func() { mobs.DeleteMobInstance(50, "TestZone", "Testmob", 9100) })

	r := roomManager.rooms[9100]
	removeRoomFromMemory(r)

	// Room unloaded AND the mob's current gold was saved on the way out.
	_, stillLoaded := roomManager.rooms[9100]
	assert.False(t, stillLoaded, "non-essential room must unload")

	saved := mobs.LoadMobInstance(50, "TestZone", "Testmob", 9100)
	assert.NotNil(t, saved, "removeRoomFromMemory must save the mob before destroying it")
	assert.NotNil(t, saved.Gold)
	assert.Equal(t, 999, *saved.Gold, "saved gold must reflect the mob's current gold")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/rooms/ -run TestRemoveRoomFromMemory_SavesChangedMob -v`
Expected: FAIL — `saved` is nil (no save happens before destroy today).

- [ ] **Step 3: Add the save calls**

In `internal/rooms/roommanager.go`, change the first destroy loop (lines 496-498) from:

```go
	for _, mobInstanceId := range room.mobs {
		mobs.DestroyInstance(mobInstanceId)
	}
```

to:

```go
	for _, mobInstanceId := range room.mobs {
		// Save goal progress (gold/equipment/plan-state) before destroying
		// the instance, so a purchase made since the last periodic save
		// survives the perf despawn. Gated internally (no-op for unchanged
		// mobs). A save failure must not block room unload.
		if m := mobs.GetInstance(mobInstanceId); m != nil {
			if err := mobs.SaveMobInstance(m); err != nil {
				mudlog.Error("removeRoomFromMemory", "save_instance", mobInstanceId, "error", err)
			}
		}
		mobs.DestroyInstance(mobInstanceId)
	}
```

And change the `SpawnInfo` loop (lines 500-510) to save before its destroy:

```go
	for _, spawnDetails := range room.SpawnInfo {
		if spawnDetails.InstanceId > 0 {

			if m := mobs.GetInstance(spawnDetails.InstanceId); m != nil {
				if m.Character.RoomId == room.RoomId {
					if err := mobs.SaveMobInstance(m); err != nil {
						mudlog.Error("removeRoomFromMemory", "save_spawn_instance", spawnDetails.InstanceId, "error", err)
					}
					mobs.DestroyInstance(spawnDetails.InstanceId)
				}
			}

		}
	}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/rooms/ -run TestRemoveRoomFromMemory_SavesChangedMob -v`
Expected: PASS.

- [ ] **Step 5: Run the full rooms package**

Run: `go test ./internal/rooms/...`
Expected: PASS (existing `TestRoomHasEssentialMob_*` tests unaffected).

- [ ] **Step 6: Commit**

```bash
git add internal/rooms/roommanager.go internal/rooms/roommanager_test.go
git commit -m "feat(rooms): save mob goal progress before perf-despawn"
```

---

### Task 8: Full build + boot smoke

**Files:** none (verification only)

- [ ] **Step 1: Build the whole project**

Run: `go build ./...`
Expected: no errors.

- [ ] **Step 2: Run the touched packages' tests together**

Run: `go test ./internal/mobs/... ./internal/rooms/...`
Expected: PASS.

- [ ] **Step 3: Boot smoke (per CLAUDE.md pre-push SOP)**

Wipe instance saves first (so template→instance overlay is exercised cleanly), then boot:

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . 2>&1 | head -80
```

Expected: server boots past data-file loading (`mobs.LoadDataFiles() loadedCount=...`, etc.) with no panic. Stop the server after confirming a clean boot.

- [ ] **Step 4: Commit (if any boot-smoke fixups were needed)**

```bash
git add -A
git commit -m "chore: boot-smoke fixups for mob goal-progress persistence"
```

(If no fixups were needed, skip this commit.)

---

## Notes for the implementer

- **Do NOT** touch the death / admin-despawn / suicide paths — they must keep calling `DeleteMobInstance` (reset on death is intended).
- **Import cycle**: never import `internal/planners` from `internal/mobs`. The `planKeyPrefix` const is intentionally duplicated.
- **Charmed-mob guard** stays at the very top of `SaveMobInstance` (above the gate) — companions persist via `CompanionInfo`, not instance files. Don't move it.
- The in-game end-to-end watch (a live combat mob saving up → buying → respawning geared) depends on the named-combat-mob despawn fix (`fix/ghost-guards-spawn-schedule-mismatch`); it is out of scope for this plan. The unit + integration tests above stand on their own.
