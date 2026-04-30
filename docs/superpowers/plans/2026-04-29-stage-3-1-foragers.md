# Stage 3.1 Foragers Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add three forager NPCs (Marsh / Steppe / Fernway) that gather mats and feed the supply pipeline 3.0b wired up; extend the Stage 2 caravan with a Fernway pickup substate; standardize the hardcoded "high-regen room" mechanic into a `sanctuary` room mutator.

**Architecture:** New `internal/forager` package owns territory data, prey whitelists, vendor visit lists, and the per-forager state machine — same pattern as `internal/caravan`. New `internal/economy` package owns the audit-matrix-derived item-id → bucket map. Shop inventory gains a `RestockBuckets([]string)` variant of `Restock()`. Caravan transit legs gain a `fernway_pickup` substate that detects the Fernway forager at North Road 4038 and acquires a load flag. Auto-heal hook is refactored to read a new `RegenMultiplier` field on `MutatorSpec` instead of the hardcoded room-id switch.

**Tech Stack:** Go, YAML data files (mobs / behaviors / dialogue / rooms / mutators / config), Stage 1 parties + Stage 2 caravan + Stage 3.0d fold-recall + Stage 3.0e corpse salvage primitives.

---

## Decisions locked at plan time (from spec + scout)

**Forager mob IDs** (next free in each zone, scanned on disk 2026-04-29):

| Forager | Zone folder | Mob ID | Anchor (sanctuary) |
|---|---|---|---|
| Marsh Forager (Vella) | `stillwater_marsh/` | **371** | room 4123 |
| Steppe Forager (Halix) | `ironwind_steppe/` | **243** | room 468 |
| Fernway Forager (Kessa) | `the_fernway_south/` | **366** | new camp room (TBD) |

**Forager weapon item IDs** (next free in `items/weapons-10000/`, scanned on disk):

| Item ID | Filename | Subtype | Owner |
|---|---|---|---|
| **10033** | `10033-marsh_gaff_hook.yaml` | dagger (puncture) | Marsh Forager |
| **10034** | `10034-steppe_hunting_spear.yaml` | spear / staff (puncture) | Steppe Forager |
| **10035** | `10035-fernway_handaxe.yaml` | club (slashing) | Fernway Forager |

(Exact subtype confirmed by reading 1-2 similar weapon YAMLs in Task 12 before authoring.)

**Forager's Camp room id** — **4197** (next free above the Stillwater Marsh range that ends at 4196). Camp parent room: 4163 (Briar Hollow, central spine of Fernway South). Camp attaches via a new `north` exit from 4163 → camp, and a `south` exit from camp → 4163. (Confirm 4163 has no existing `north` exit during Task 14.)

**Stillwater vendor visit order** (Marsh forager — same 8 vendors the caravan visits, same locality-clustered order from the Stage 2 caravan plan):

| Order | Room | Mob | Name |
|---|---|---|---|
| 1 | 4102 | 336 | fishmonger Tov Brann |
| 2 | 4103 | 333 | innkeeper Sigrid |
| 3 | 4105 | 341 | storekeeper Wulf |
| 4 | 4106 | 337 | smith Brindle |
| 5 | 4125 | 338 | apothecary Ilsa |
| 6 | 4126 | 340 | pearl-carver Kess |
| 7 | 4135 | 348 | miller Bram |
| 8 | 4143 | 339 | weaver Edda |

Marsh forager territory entry: room 4177 (entry from 4138 in Stillwater).

**Thornwall vendor visit order** (Steppe forager — same 9 vendors the caravan visits):

| Order | Room | Mob | Name |
|---|---|---|---|
| 1 | 464 | 103 | food vendor |
| 2 | 470 | 97 | blacksmith Kerra |
| 3 | 471 | 98 | apothecary Voss |
| 4 | 475 | 104 | fence dealer Siv |
| 5 | 480 | 113 | weaver Maren |
| 6 | 481 | 248 | tavern cook Brynn |
| 7 | 482 | 108 | jeweler Tess |
| 8 | 483 | 109 | enchanter Vael |
| 9 | 507 | 273 | Whisper |

Steppe forager territory entry: room 3000 (Eastern Gate Approach, connects to Thornwall east gate).

**Fernway forager meeting point:** North Road **room 4038** (the road-mouth of Fernway's Western Trailhead 4153). Forager walks from camp (4197) → 4163 → 4156 → 4153 → 4038 and back. Fernway forager territory entry: room 4163 (own camp's parent).

**Steppe forager territory subset (safe northern half)** — selected by Y ≥ 0 in coord_map (no goblins, no apex predators):
3000, 3001, 3002, 3003, 3005, 3006, 3015, 3019, 3020, 3021, 3022, 3023, 3024, 3025, 3026, 3027, 3028, 3029. (Plan task 0 verifies exits to confirm reachability; Task 13 trims if any room turns out to spawn an apex predator.)

**Engine API anchors** (verified by scout):
- `mutators.MutatorSpec` struct — `internal/mutators/mutators.go:58`
- `Room.Mutators mutators.MutatorList` — `internal/rooms/rooms.go:100`
- `Room.ActiveMutators` iterator — `internal/rooms/rooms.go:2368`
- `roomRegenMultiplier(roomId int) float64` — `internal/hooks/NewRound_AutoHeal.go:375` (callers at lines 43, 266)
- `mobs.TickMobShopRestock` — `internal/mobs/crafter.go:88`
- `shops.ShopInventory.Restock() bool` — `internal/shops/shopinventory.go:71`
- Caravan state enum — `internal/caravan/state.go`
- `caravan_step` btree action — `internal/behaviortree/actions_caravan.go`
- Stage 3.0d fold-recall NPC YAML pattern: `fold_anchor_room: <id>` field + `tactics: { trigger: health_below:N, action: cast fold-recall, priority: 13 }` — see `_datafiles/world/dogmud/mobs/marches_spur_road/275-old_edrin.yaml:16,38` and `_datafiles/world/dogmud/mobs/thornwall_city/357-ketil.yaml:18,57`.
- `player_attack_immune: true` mob field — see Ketil 357.
- Existing player forage core — `internal/usercommands/skill.forage.go:54-119`.

**Cycle math** — `RoundsPerDay: 900`. New `CaravanDepotDwellRounds: 720` (was 360). Caravan cycle ≈ 1620 rounds = ~2 game days.

---

## File structure overview

| Layer | File | Purpose |
|---|---|---|
| Engine struct | `internal/mutators/mutators.go` | Add `RegenMultiplier float64` to MutatorSpec (Task 1) |
| Engine refactor | `internal/hooks/NewRound_AutoHeal.go` | Replace hardcoded switch with mutator-driven lookup (Task 2) |
| Engine config | `internal/configs/config.balance.go` | Add 6 forager knobs, bump caravan dwell (Task 3) |
| Engine config | `_datafiles/config.yaml` | Default values (Task 3) |
| Engine package (new) | `internal/economy/buckets.go` | Item-id → bucket map (Task 4) |
| Engine package (new) | `internal/economy/buckets_test.go` | Audit-matrix invariants (Task 4) |
| Engine method | `internal/shops/shopinventory.go` | `RestockBuckets([]string)` method (Task 5) |
| Engine refactor | `internal/usercommands/skill.forage.go` | Extract `ForageCore` (Task 6) |
| Engine package (new) | `internal/forager/state.go` | State enum + transitions (Task 7) |
| Engine package (new) | `internal/forager/territory.go` | Per-forager territory + prey whitelist + vendor list (Task 7) |
| Btree action (new) | `internal/behaviortree/actions_forager.go` | `forager_step` (Task 8) |
| Btree conditions (new) | `internal/behaviortree/conditions_forager.go` | 3 conditions (Task 8) |
| Caravan refactor | `internal/caravan/state.go` | Add `fernway_pickup` substate (Task 9) |
| Caravan refactor | `internal/caravan/visit.go` | Switch to `RestockBuckets(caravan_load)` (Task 9) |
| Caravan refactor | `internal/behaviortree/actions_caravan.go` | Wire fernway_pickup (Task 10) |
| Mutator content | `_datafiles/world/dogmud/mutators/sanctuary.yaml` | Sanctuary regen mutator (Task 11) |
| Room content | 23 existing rooms | Wire `sanctuary` mutator on 468, 4123, 101–120 (Task 11) |
| Room content (new) | `_datafiles/world/dogmud/rooms/the_fernway_south/4197.yaml` | Forager's Camp + sanctuary mutator (Task 14) |
| Room content | `_datafiles/world/dogmud/rooms/the_fernway_south/4163.yaml` | Add `north` exit to camp (Task 14) |
| Item content (new) | 3 weapon YAMLs | Forager weapons (Task 12) |
| Mob content (new) | 3 forager mob YAMLs + 3 behavior YAMLs + 3 dialogue YAMLs | Foragers (Tasks 13, 14, 15) |
| Hint content | `_datafiles/world/dogmud/hints.yaml` | Generalize temple-regen hint (Task 16) |
| Docs | `docs/economy/mat-audit-matrix.md` | Sync-marker comment (Task 17) |
| Docs | `docs/schemas/{mob,behavior,room,mutator}.md` | Document new fields (Task 17) |
| Docs | `PATCH_NOTES.md` | Stage 3.1 entry (Task 17) |

---

### Task 1: Add `RegenMultiplier` field to MutatorSpec

**Files:**
- Modify: `internal/mutators/mutators.go:58-74` — add field
- Modify: `internal/mutators/mutators_test.go` (or create if missing) — test field reads from YAML

- [ ] **Step 1: Read the existing field group around `LightMod` to confirm yaml-tag conventions.**

Run: read `internal/mutators/mutators.go:58-74`. Note the yaml tag style (`omitempty`, lowercase no-underscore key like `lightmod`).

- [ ] **Step 2: Write a failing test that asserts a YAML mutator with `regen_multiplier: 5.0` parses into MutatorSpec.RegenMultiplier == 5.0.**

In `internal/mutators/mutators_test.go` (create if missing), add:

```go
package mutators

import (
    "strings"
    "testing"

    "gopkg.in/yaml.v3"
)

func TestMutatorSpec_RegenMultiplierField(t *testing.T) {
    src := `mutatorid: test-sanctuary
regen_multiplier: 5.0
`
    var spec MutatorSpec
    if err := yaml.Unmarshal([]byte(src), &spec); err != nil {
        t.Fatalf("unmarshal: %v", err)
    }
    if spec.MutatorId != "test-sanctuary" {
        t.Errorf("MutatorId = %q, want test-sanctuary", spec.MutatorId)
    }
    if spec.RegenMultiplier != 5.0 {
        t.Errorf("RegenMultiplier = %v, want 5.0", spec.RegenMultiplier)
    }
}

func TestMutatorSpec_RegenMultiplierDefaultsZero(t *testing.T) {
    src := `mutatorid: nonregen-mutator`
    var spec MutatorSpec
    if err := yaml.Unmarshal([]byte(src), &spec); err != nil {
        t.Fatalf("unmarshal: %v", err)
    }
    if spec.RegenMultiplier != 0 {
        t.Errorf("RegenMultiplier = %v, want 0 (unset)", spec.RegenMultiplier)
    }
    _ = strings.TrimSpace
}
```

- [ ] **Step 3: Run test to verify it fails.**

Run: `cd "C:/Users/Calabe Davis/workspace/DOGMud" && go test ./internal/mutators/... -run RegenMultiplier -v`
Expected: FAIL — `unknown field RegenMultiplier on MutatorSpec` (compile error).

- [ ] **Step 4: Add the field.**

In `internal/mutators/mutators.go`, append to the MutatorSpec struct alongside `LightMod`:

```go
    RegenMultiplier float64                  `yaml:"regen_multiplier,omitempty"` // multiplies HP/SP/CP regen for any actor in the room (1.0 / 0 = no bonus)
```

- [ ] **Step 5: Run test to verify it passes.**

Run: `go test ./internal/mutators/... -run RegenMultiplier -v`
Expected: PASS.

- [ ] **Step 6: Run the full mutators package test suite to confirm no regressions.**

Run: `go test ./internal/mutators/...`
Expected: PASS.

- [ ] **Step 7: Commit.**

```bash
git add internal/mutators/mutators.go internal/mutators/mutators_test.go
git commit -m "$(cat <<'EOF'
feat(mutators): add RegenMultiplier field to MutatorSpec

Backs the Stage 3.1 sanctuary mutator that replaces the hardcoded
roomRegenMultiplier switch in the auto-heal hook.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: Refactor `roomRegenMultiplier` to mutator-driven

**Files:**
- Modify: `internal/hooks/NewRound_AutoHeal.go:375-387` — replace function body
- Modify: `internal/hooks/NewRound_AutoHeal.go:43,266` — update callers (pass *Room not roomId)
- Test: `internal/hooks/NewRound_AutoHeal_test.go` (create if missing)

- [ ] **Step 1: Write a failing test for the new signature.**

In `internal/hooks/NewRound_AutoHeal_test.go` (create if missing), add:

```go
package hooks

import (
    "testing"

    "github.com/GoMudEngine/GoMud/internal/mutators"
    "github.com/GoMudEngine/GoMud/internal/rooms"
)

// fakeRoomWithMutators is a tiny test double that returns a fixed
// MutatorList from ActiveMutators. Real *rooms.Room construction is
// painful in tests; this lets us exercise the multiplier math in
// isolation.
//
// NOTE: the real implementation iterates room.ActiveMutators which
// yields mutators.Mutator values whose Spec() returns *MutatorSpec.

func TestRoomRegenMultiplier_NoMutators(t *testing.T) {
    r := &rooms.Room{}
    if got := roomRegenMultiplier(r); got != 1.0 {
        t.Errorf("got %v want 1.0", got)
    }
}

func TestRoomRegenMultiplier_SingleSanctuary(t *testing.T) {
    // Build a room with the sanctuary mutator manifested as live.
    // We rely on the real mutators package having "sanctuary"
    // registered with RegenMultiplier=5.0 by Task 11; this test
    // therefore depends on Task 11 being completed first OR on
    // injecting a synthetic spec into the mutator registry.
    r := buildRoomWithMutator(t, "sanctuary")
    if got := roomRegenMultiplier(r); got != 5.0 {
        t.Errorf("got %v want 5.0", got)
    }
}

func TestRoomRegenMultiplier_StacksMultiplicatively(t *testing.T) {
    r := buildRoomWithMutators(t, "sanctuary", "sanctuary")
    if got := roomRegenMultiplier(r); got != 25.0 {
        t.Errorf("got %v want 25.0 (5.0 * 5.0)", got)
    }
}
```

Add helper at the bottom:

```go
func buildRoomWithMutator(t *testing.T, ids ...string) *rooms.Room {
    t.Helper()
    r := &rooms.Room{}
    for _, id := range ids {
        r.Mutators = append(r.Mutators, mutators.Mutator{MutatorId: id})
    }
    return r
}

func buildRoomWithMutators(t *testing.T, ids ...string) *rooms.Room {
    return buildRoomWithMutator(t, ids...)
}
```

(Adjust `mutators.Mutator{...}` literal to match the actual struct shape — read `internal/mutators/mutators.go` for the `Mutator` type definition before writing the test.)

- [ ] **Step 2: Run test to verify it fails.**

Run: `go test ./internal/hooks/... -run RoomRegenMultiplier -v`
Expected: FAIL — `roomRegenMultiplier` takes `int`, not `*Room`.

- [ ] **Step 3: Refactor `roomRegenMultiplier` to take `*rooms.Room`.**

In `internal/hooks/NewRound_AutoHeal.go`, replace the existing function:

```go
// roomRegenMultiplier returns the regen multiplier applied to HP/SP/CP
// regen for any actor (player or mob) in the given room. 1.0 means no
// bonus. Sourced from the room's active mutators — any mutator with
// RegenMultiplier > 0 contributes; multiple mutators stack
// multiplicatively. Replaces the Stage 2-era hardcoded room-id switch.
func roomRegenMultiplier(room *rooms.Room) float64 {
    if room == nil {
        return 1.0
    }
    mult := 1.0
    for mut := range room.ActiveMutators {
        spec := mutators.GetMutatorSpec(mut.MutatorId)
        if spec == nil || spec.RegenMultiplier <= 0 {
            continue
        }
        mult *= spec.RegenMultiplier
    }
    return mult
}
```

Add `"github.com/GoMudEngine/GoMud/internal/mutators"` to the imports if not already present.

- [ ] **Step 4: Update the two callers.**

At `internal/hooks/NewRound_AutoHeal.go:43`:

```go
regenMultiplier := roomRegenMultiplier(rooms.LoadRoom(user.Character.RoomId))
```

At `internal/hooks/NewRound_AutoHeal.go:266`:

```go
mobRegenMult := roomRegenMultiplier(rooms.LoadRoom(mob.Character.RoomId))
```

- [ ] **Step 5: Run hooks tests.**

Run: `go test ./internal/hooks/... -run RoomRegenMultiplier -v`
Expected: TestNoMutators PASS. TestSingleSanctuary + TestStacksMultiplicatively will FAIL until Task 11 registers the `sanctuary` mutator. That's expected — we'll re-run them after Task 11.

- [ ] **Step 6: Run the full hooks test suite + a full build to confirm no regressions.**

Run: `go test ./internal/hooks/...`
Expected: TestSingleSanctuary + TestStacksMultiplicatively FAIL (expected — depend on Task 11). Other tests PASS.

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 7: Commit.**

```bash
git add internal/hooks/NewRound_AutoHeal.go internal/hooks/NewRound_AutoHeal_test.go
git commit -m "$(cat <<'EOF'
refactor(hooks): mutator-driven roomRegenMultiplier

Replaces the hardcoded room-id switch (testing arena 200 + Sanctum
tutorial 101-120 + Thornwall Temple 468) with a lookup driven by the
new MutatorSpec.RegenMultiplier field. Sanctuary mutator wiring lands
in a later task; tests for sanctuary wiring will pass once that task
ships.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 3: Add config knobs + bump caravan dwell

**Files:**
- Modify: `internal/configs/config.balance.go` — add 6 forager knobs + bump CaravanDepotDwellRounds default
- Modify: `_datafiles/config.yaml` — add knob defaults

- [ ] **Step 1: Read the existing balance config file to find where caravan knobs live.**

Run: `grep -n CaravanDepotDwellRounds internal/configs/config.balance.go`
Note line numbers; new knobs go alongside.

- [ ] **Step 2: Add the 6 new knobs + bump CaravanDepotDwell default.**

In `internal/configs/config.balance.go`, locate the existing `CaravanDepotDwellRounds` field and:

a. Bump its default value tag (or struct-literal default in the loader) from 360 to 720.

b. Add immediately after it:

```go
    FernwayPickupDwellRounds        int     `yaml:"fernway_pickup_dwell_rounds"`        // rounds caravan dwells at meeting point 4038
    ForagerForageDwellRounds        int     `yaml:"forager_forage_dwell_rounds"`        // rounds between forage attempts in territory
    ForagerCarryThresholdPct        float64 `yaml:"forager_carry_threshold_pct"`        // 0-1; trigger to head-home when satchel this % full
    ForagerHPRecallThresholdPct     float64 `yaml:"forager_hp_recall_threshold_pct"`    // 0-1; emergency recall trigger
    ForagerHealPotionThresholdPct   float64 `yaml:"forager_heal_potion_threshold_pct"`  // 0-1; auto-drink trigger
    ForagerWaitTimeoutRounds        int     `yaml:"forager_wait_timeout_rounds"`        // Fernway-only; max wait at 4038 before bailing
```

c. In the loader's default-fill block, set:

```go
    if cfg.CaravanDepotDwellRounds == 0 { cfg.CaravanDepotDwellRounds = 720 }
    if cfg.FernwayPickupDwellRounds == 0 { cfg.FernwayPickupDwellRounds = 6 }
    if cfg.ForagerForageDwellRounds == 0 { cfg.ForagerForageDwellRounds = 8 }
    if cfg.ForagerCarryThresholdPct == 0 { cfg.ForagerCarryThresholdPct = 0.75 }
    if cfg.ForagerHPRecallThresholdPct == 0 { cfg.ForagerHPRecallThresholdPct = 0.50 }
    if cfg.ForagerHealPotionThresholdPct == 0 { cfg.ForagerHealPotionThresholdPct = 0.75 }
    if cfg.ForagerWaitTimeoutRounds == 0 { cfg.ForagerWaitTimeoutRounds = 150 }
```

(Pattern-match to whatever defaulting style the existing code uses — read 5 lines above and below the existing default for `CaravanDepotDwellRounds`.)

- [ ] **Step 3: Add YAML defaults to `_datafiles/config.yaml`.**

In the `Balance:` block, alongside `CaravanDepotDwellRounds`:

```yaml
  CaravanDepotDwellRounds: 720
  FernwayPickupDwellRounds: 6
  ForagerForageDwellRounds: 8
  ForagerCarryThresholdPct: 0.75
  ForagerHPRecallThresholdPct: 0.50
  ForagerHealPotionThresholdPct: 0.75
  ForagerWaitTimeoutRounds: 150
```

- [ ] **Step 4: Build to confirm no compile errors.**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 5: Commit.**

```bash
git add internal/configs/config.balance.go _datafiles/config.yaml
git commit -m "$(cat <<'EOF'
feat(config): add 6 forager knobs + bump caravan dwell to 720

CaravanDepotDwellRounds 360 → 720 makes foragers the day-to-day supply
pipeline; caravan becomes a delivery-day event. New forager knobs gate
forage cadence, carry threshold, recall thresholds, and meeting-point
wait.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 4: Create `internal/economy` package with item-bucket map

**Files:**
- Create: `internal/economy/buckets.go`
- Create: `internal/economy/buckets_test.go`

- [ ] **Step 1: Write a failing test that asserts every classified item in the audit matrix has a bucket entry.**

In `internal/economy/buckets_test.go`:

```go
package economy

import (
    "testing"

    "github.com/GoMudEngine/GoMud/internal/items"
)

func TestBucketFor_KnownItems(t *testing.T) {
    cases := []struct {
        id     int
        bucket string
    }{
        {40001, "base"},        // iron ingot
        {40046, "fernway"},     // moonpetal
        {40051, "stillwater"},  // skitter-shrimp shell
        {40010, "thornwall"},   // chrysalis core
        {40004, "overlap"},     // healer's root
        {40062, "fernway"},     // oak bark (3.0b new)
        {99999, ""},            // unknown
    }
    for _, c := range cases {
        if got := BucketFor(c.id); got != c.bucket {
            t.Errorf("BucketFor(%d) = %q, want %q", c.id, got, c.bucket)
        }
    }
}

// TestBucketMap_AuditMatrixCoverage asserts every item with
// is_component: true OR component_tag set has a bucket entry.
// This catches drift between the audit matrix doc and the Go map.
//
// Quest/specialty and Defer-to-3.0e items are exceptions and listed
// explicitly.
func TestBucketMap_AuditMatrixCoverage(t *testing.T) {
    knownExceptions := map[int]bool{
        // Quest/specialty (15 items per audit matrix)
        40031: true, 40032: true, 40033: true, 40034: true, 40035: true,
        40036: true, 40037: true, 40038: true, 40039: true, 40040: true,
        40041: true, 40042: true, 40054: true, 40060: true, 40061: true,
        // Defer to 3.0e (cloth/leather adjacent — not yet bucketed)
        40052: true, 40055: true,
    }
    for id := 40001; id <= 40068; id++ {
        if knownExceptions[id] {
            continue
        }
        spec := items.GetItemSpec(id)
        if spec == nil {
            // Item id unused — skip (not all 40000s exist)
            continue
        }
        if spec.ComponentTag == "" && !spec.IsComponent {
            continue // not a crafting component, skip
        }
        if got := BucketFor(id); got == "" {
            t.Errorf("item %d (%s) is a component but has no bucket", id, spec.Name)
        }
    }
}
```

(If `items.GetItemSpec` doesn't exist by that exact name, `grep` for the right accessor — `items.GetSpec` or similar — and adjust.)

- [ ] **Step 2: Run test to verify it fails.**

Run: `go test ./internal/economy/... -v`
Expected: FAIL — `package economy` doesn't exist yet.

- [ ] **Step 3: Create the bucket map.**

In `internal/economy/buckets.go`:

```go
// Package economy holds shared classification data for the supply
// pipeline. The item-bucket map is the Go-side mirror of
// docs/economy/mat-audit-matrix.md (Stage 3.0b).
//
// Used by:
//   - internal/shops.RestockBuckets (gates which slots a forager or
//     caravan delivery refills)
//   - internal/forager (each forager's bucket list)
//   - internal/caravan (caravan_load tracking)
//
// Keep in sync with docs/economy/mat-audit-matrix.md. Drift is caught
// by TestBucketMap_AuditMatrixCoverage in buckets_test.go.
package economy

// itemBucket maps item IDs to their supply bucket. Items not in the
// map (quest/specialty items, defer-to-3.0e cloth/leather) return ""
// from BucketFor.
var itemBucket = map[int]string{
    // Base bucket — universal feedstock (13 items)
    40001: "base", // iron ingot
    40003: "base", // wooden plank
    40006: "base", // glass vial
    40012: "base", // thread spool
    40013: "base", // bone needle
    40014: "base", // raw meat
    40015: "base", // wild vegetables
    40016: "base", // water flask
    40017: "base", // salt pouch
    40019: "base", // chain link
    40043: "base", // clay flask
    40044: "base", // sealed phial
    40045: "base", // crystalline decanter

    // Stillwater bucket (6 items)
    40051: "stillwater", // skitter-shrimp shell
    40053: "stillwater", // Stillwater black pearl
    40056: "stillwater", // marsh willow bark
    40057: "stillwater", // lake mint
    40058: "stillwater", // freshwater clam
    40059: "stillwater", // lake-iron nodule

    // Thornwall bucket (13 items) — in-shop crafted, not foraged
    40010: "thornwall", // Chrysalis Core
    40011: "thornwall", // Hive Fragment
    40018: "thornwall", // steel ingot
    40021: "thornwall", // copper wire
    40022: "thornwall", // silver wire
    40023: "thornwall", // gold wire
    40024: "thornwall", // polished stone
    40025: "thornwall", // raw gem
    40026: "thornwall", // gem dust
    40027: "thornwall", // chrysalis shard
    40028: "thornwall", // binding paste
    40029: "thornwall", // mutation catalyst
    40030: "thornwall", // chrysalis setting

    // Fernway bucket (8 items)
    40046: "fernway", // moonpetal
    40049: "fernway", // ironbark shaving
    40062: "fernway", // oak bark (3.0b)
    40063: "fernway", // shadowcap mushroom (3.0b)
    40064: "fernway", // wild hare meat (3.0b)
    40065: "fernway", // beeswax (3.0b)
    40066: "fernway", // blood-moss (3.0b)
    40067: "fernway", // pine pitch (3.0b)

    // Mid-tier overlap (11 items)
    40002: "overlap", // leather strip
    40004: "overlap", // healer's root
    40005: "overlap", // bitter thistle
    40007: "overlap", // cloth strip
    40008: "overlap", // spore sac
    40009: "overlap", // dustwalk herb
    40020: "overlap", // coal dust
    40047: "overlap", // veilbloom petal
    40048: "overlap", // serpent venom sac
    40050: "overlap", // putrid residue
    40068: "overlap", // sinew (3.0e)
}

// BucketFor returns the supply bucket for an item ID, or "" if the
// item is unbucketed (quest/specialty, deferred, or unknown).
func BucketFor(itemId int) string {
    return itemBucket[itemId]
}

// AllBuckets returns the canonical list of supply buckets used in
// the bucket-aware restock system. Stable order for callers that
// need determinism (tests, CLI output).
func AllBuckets() []string {
    return []string{"base", "stillwater", "thornwall", "fernway", "overlap"}
}
```

- [ ] **Step 4: Run test to verify it passes.**

Run: `go test ./internal/economy/... -v`
Expected: PASS.

- [ ] **Step 5: Commit.**

```bash
git add internal/economy/
git commit -m "$(cat <<'EOF'
feat(economy): item-id bucket map for supply-pipeline classification

Mirrors docs/economy/mat-audit-matrix.md (Stage 3.0b classification).
Backs RestockBuckets (Task 5), forager packages, and the caravan_load
flag system. TestBucketMap_AuditMatrixCoverage catches drift between
the doc and the Go map.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 5: Add `RestockBuckets([]string)` shop method

**Files:**
- Modify: `internal/shops/shopinventory.go` — add new method alongside existing `Restock`
- Modify: `internal/shops/shopinventory_test.go` — new tests

- [ ] **Step 1: Read existing `Restock()` to mirror its signature/behavior.**

Run: read `internal/shops/shopinventory.go:60-120` (find Restock).

- [ ] **Step 2: Write a failing test for RestockBuckets.**

In `internal/shops/shopinventory_test.go`, add:

```go
func TestRestockBuckets_OnlyFillsMatchingBucket(t *testing.T) {
    si := &ShopInventory{Entries: []ShopInventoryEntry{
        {ItemId: 40001 /*base*/, Qty: 0, MaxStock: 5, RestockQty: 5},
        {ItemId: 40051 /*stillwater*/, Qty: 0, MaxStock: 5, RestockQty: 5},
        {ItemId: 40010 /*thornwall*/, Qty: 0, MaxStock: 5, RestockQty: 5},
    }}
    refilled := si.RestockBuckets([]string{"stillwater"})
    if !refilled {
        t.Fatal("expected RestockBuckets to refill at least one slot")
    }
    if si.Entries[0].Qty != 0 {
        t.Errorf("base slot refilled but bucket not in list, qty=%d", si.Entries[0].Qty)
    }
    if si.Entries[1].Qty != 5 {
        t.Errorf("stillwater slot not refilled, qty=%d", si.Entries[1].Qty)
    }
    if si.Entries[2].Qty != 0 {
        t.Errorf("thornwall slot refilled but bucket not in list, qty=%d", si.Entries[2].Qty)
    }
}

func TestRestockBuckets_MultipleBucketsUnion(t *testing.T) {
    si := &ShopInventory{Entries: []ShopInventoryEntry{
        {ItemId: 40001 /*base*/, Qty: 0, MaxStock: 5, RestockQty: 5},
        {ItemId: 40051 /*stillwater*/, Qty: 0, MaxStock: 5, RestockQty: 5},
        {ItemId: 40046 /*fernway*/, Qty: 0, MaxStock: 5, RestockQty: 5},
    }}
    si.RestockBuckets([]string{"stillwater", "fernway"})
    if si.Entries[0].Qty != 0 {
        t.Errorf("base slot refilled, qty=%d", si.Entries[0].Qty)
    }
    if si.Entries[1].Qty != 5 || si.Entries[2].Qty != 5 {
        t.Errorf("expected stillwater + fernway refilled; got %d, %d",
            si.Entries[1].Qty, si.Entries[2].Qty)
    }
}

func TestRestockBuckets_EmptyListNoOp(t *testing.T) {
    si := &ShopInventory{Entries: []ShopInventoryEntry{
        {ItemId: 40001, Qty: 0, MaxStock: 5, RestockQty: 5},
    }}
    if si.RestockBuckets(nil) {
        t.Error("nil bucket list should be no-op")
    }
    if si.RestockBuckets([]string{}) {
        t.Error("empty bucket list should be no-op")
    }
    if si.Entries[0].Qty != 0 {
        t.Errorf("entry refilled despite empty bucket list, qty=%d", si.Entries[0].Qty)
    }
}
```

(Confirm `ShopInventoryEntry` field names — `Qty`, `MaxStock`, `RestockQty` — by reading the struct in `shopinventory.go`. If they differ, adjust test literals.)

- [ ] **Step 3: Run test to verify it fails.**

Run: `go test ./internal/shops/... -run RestockBuckets -v`
Expected: FAIL — undefined: `(*ShopInventory).RestockBuckets`.

- [ ] **Step 4: Implement RestockBuckets.**

In `internal/shops/shopinventory.go`, add alongside Restock:

```go
import (
    // ... existing imports
    "slices"

    "github.com/GoMudEngine/GoMud/internal/economy"
)

// RestockBuckets is like Restock(), but only refills slots whose
// item-id falls in one of the given supply buckets. Used by foragers
// (always one bucket per call — their region's) and the caravan
// (one or two buckets per call, based on caravan_load).
//
// nil or empty buckets returns false without modifying any entry.
func (si *ShopInventory) RestockBuckets(buckets []string) bool {
    if len(buckets) == 0 {
        return false
    }
    refilled := false
    for i := range si.Entries {
        entry := &si.Entries[i]
        bucket := economy.BucketFor(entry.ItemId)
        if bucket == "" || !slices.Contains(buckets, bucket) {
            continue
        }
        // Mirror Restock()'s top-up logic exactly.
        if entry.Qty >= entry.MaxStock {
            continue
        }
        entry.Qty += entry.RestockQty
        if entry.Qty > entry.MaxStock {
            entry.Qty = entry.MaxStock
        }
        refilled = true
    }
    return refilled
}
```

(Read the existing `Restock()` body and copy the top-up logic verbatim — don't paraphrase. The test relies on identical semantics.)

- [ ] **Step 5: Run test to verify it passes.**

Run: `go test ./internal/shops/... -run RestockBuckets -v`
Expected: PASS.

- [ ] **Step 6: Run full shops test suite.**

Run: `go test ./internal/shops/...`
Expected: PASS.

- [ ] **Step 7: Commit.**

```bash
git add internal/shops/shopinventory.go internal/shops/shopinventory_test.go
git commit -m "$(cat <<'EOF'
feat(shops): RestockBuckets bucket-aware refill

Mirrors Restock() but only tops up slots whose item-id maps to one of
the given buckets. Backs Stage 3.1 forager + caravan supply gating.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 6: Extract forage core into reusable function

**Files:**
- Modify: `internal/usercommands/skill.forage.go` — extract core
- Test: `internal/usercommands/skill.forage_test.go` (extend or create)

- [ ] **Step 1: Read the existing Forage function (lines 54-119).**

Confirm: difficulty lookup, yield table, cooldown, search-skill scoring, dice roll, item creation, and quest-engine notification all happen in one function. The cooldown + quest-engine bits are player-specific; the rest is core.

- [ ] **Step 2: Write the extraction.**

In `internal/usercommands/skill.forage.go`, refactor:

```go
// ForageAttempt holds the inputs needed to run one forage roll. Used
// by both the player Forage command and NPC forager routines.
type ForageAttempt struct {
    Biome         string
    SearchScore   float64 // perception + skill multiplier
    AtNight       bool
}

// ForageResult is the outcome of a single attempt.
type ForageResult struct {
    Found  bool
    ItemId int     // 0 if not found
    // ItemSpec creation deferred to caller — caller handles "crumbles
    // in your hands" path if items.New rejects.
}

// ForageCore runs the dice roll for one forage attempt. Pure: no
// side effects, no character mutation, no event publication. Caller
// is responsible for cooldowns, item creation, inventory storage, and
// any quest-engine notifications.
func ForageCore(a ForageAttempt) ForageResult {
    yields, ok := forageYields[a.Biome]
    if !ok || len(yields) == 0 {
        return ForageResult{}
    }
    if a.AtNight {
        if night, hasNight := nightForageYields[a.Biome]; hasNight {
            yields = append(append([]int{}, yields...), night...)
        }
    }
    difficulty := forageDifficulty[a.Biome]
    if difficulty == 0 {
        difficulty = 130
    }
    roll := dice.RollStat(a.SearchScore)
    if roll.Value < difficulty {
        return ForageResult{}
    }
    return ForageResult{Found: true, ItemId: yields[util.Rand(len(yields))]}
}
```

Then refactor the existing `Forage` function to call `ForageCore`:

```go
func Forage(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

    biome := room.GetBiome()
    if _, ok := forageYields[biome.BiomeId]; !ok {
        user.SendText(`There is nothing here worth foraging. Try an outdoor area.`)
        return true, nil
    }

    if !user.Character.TryCooldown(`forage`, "6 rounds") {
        user.SendText(
            fmt.Sprintf("You need to wait %d more rounds before you can forage again.", user.Character.GetCooldown(`forage`)),
        )
        return true, fmt.Errorf("you're doing that too often")
    }

    searchRank := user.Character.GetSkillLevel(skills.Search)
    searchScore := float64(user.Character.Stats.Perception.ValueAdj) + combat.SkillMultiplier(searchRank)*25.0

    bridge := questengine.NewGameBridge(user, room.RoomId)
    questengine.GetEngine().Notify("command", questengine.EventDetails{
        UserId: user.UserId, RoomId: room.RoomId, Command: "forage",
    }, bridge, bridge)

    user.SendText(`You crouch low and begin searching the ground carefully...`)
    room.SendTextVisual(
        fmt.Sprintf(`<ansi fg="username">%s</ansi> is searching the ground for something.`, user.Character.Name),
        user.UserId,
    )

    result := ForageCore(ForageAttempt{
        Biome:       biome.BiomeId,
        SearchScore: searchScore,
        AtNight:     gametime.IsNight(),
    })

    if !result.Found {
        user.SendText(`You find nothing of use this time.`)
        return true, nil
    }

    newItem := items.New(result.ItemId)
    if !newItem.IsValid() {
        user.SendText(`You find something, but it crumbles in your hands.`)
        return true, nil
    }

    user.Character.StoreItem(newItem)
    events.AddToQueue(events.ItemOwnership{UserId: user.UserId, Item: newItem, Gained: true})
    user.Character.CheckSkillProgression(string(skills.Search), user.UserId, 1.0)

    user.SendText(fmt.Sprintf(`You find a <ansi fg="itemname">%s</ansi>.`, newItem.DisplayName()))
    return true, nil
}
```

- [ ] **Step 3: Add a test for ForageCore.**

In `internal/usercommands/skill.forage_test.go` (create if missing):

```go
package usercommands

import (
    "testing"
)

func TestForageCore_UnknownBiomeReturnsEmpty(t *testing.T) {
    r := ForageCore(ForageAttempt{Biome: "nonexistent", SearchScore: 1000})
    if r.Found {
        t.Error("expected unknown biome to return Found=false")
    }
}

func TestForageCore_HighScoreFinds(t *testing.T) {
    // Score 1000 vs forest difficulty 120 — should always find.
    found := false
    for i := 0; i < 50 && !found; i++ {
        r := ForageCore(ForageAttempt{Biome: "forest", SearchScore: 1000})
        if r.Found {
            found = true
        }
    }
    if !found {
        t.Error("expected at least one find in 50 attempts at SearchScore 1000")
    }
}
```

- [ ] **Step 4: Run tests.**

Run: `go test ./internal/usercommands/... -run Forage -v`
Expected: PASS.

- [ ] **Step 5: Run the full usercommands tests.**

Run: `go test ./internal/usercommands/...`
Expected: PASS (no regressions).

- [ ] **Step 6: Commit.**

```bash
git add internal/usercommands/skill.forage.go internal/usercommands/skill.forage_test.go
git commit -m "$(cat <<'EOF'
refactor(forage): extract ForageCore for NPC reuse

Splits the dice roll + yield lookup out of the player command so the
Stage 3.1 forager NPC routines can use the same yield table and
difficulty math without duplicating the data tables.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 7: Create `internal/forager` package

**Files:**
- Create: `internal/forager/state.go`
- Create: `internal/forager/territory.go`
- Create: `internal/forager/state_test.go`
- Create: `internal/forager/territory_test.go`

- [ ] **Step 1: Write `state.go` — ForagerState enum + transitions.**

```go
// Package forager implements the Stage 3.1 forager NPC state machine.
// State transitions are pure functions; the forager_step btree action
// (internal/behaviortree/actions_forager.go) decides WHEN to advance.
package forager

type ForagerState int

const (
    StateResting ForagerState = iota
    StateTravelingToTerritory
    StateForaging
    StateTravelingToDropoff
    StateDelivering
    StateRecalling
)

var stateNames = map[ForagerState]string{
    StateResting:              "resting",
    StateTravelingToTerritory: "traveling_to_territory",
    StateForaging:             "foraging",
    StateTravelingToDropoff:   "traveling_to_dropoff",
    StateDelivering:           "delivering",
    StateRecalling:            "recalling",
}

var nameToState = func() map[string]ForagerState {
    m := make(map[string]ForagerState, len(stateNames))
    for s, n := range stateNames {
        m[n] = s
    }
    return m
}()

func (s ForagerState) Name() string { return stateNames[s] }

func ParseState(name string) (ForagerState, bool) {
    s, ok := nameToState[name]
    return s, ok
}

// AdvanceState returns the next state in the normal cycle. Recalling
// always wraps back to Resting.
func AdvanceState(cur ForagerState) ForagerState {
    return (cur + 1) % 6
}
```

- [ ] **Step 2: Write `territory.go` — per-forager static data.**

```go
package forager

// ForagerKind identifies one of the three Stage 3.1 foragers.
type ForagerKind int

const (
    KindMarsh ForagerKind = iota
    KindSteppe
    KindFernway
)

// ForagerProfile holds the static data for one forager — territory,
// prey whitelist, vendor visit list, supply buckets, anchor room.
//
// Data is keyed by mob ID so the btree can resolve at runtime via
// the leader's MobId.
type ForagerProfile struct {
    Kind           ForagerKind
    MobId          int
    Name           string
    SanctuaryRoom  int       // fold-recall anchor
    TerritoryRooms []int     // wandering range
    PreyWhitelist  []int     // mobIds of legal engagement targets
    VendorRooms    []int     // delivery route (Marsh + Steppe). Empty for Fernway.
    MeetingRoom    int       // Fernway only: where caravan handoff happens. 0 for Marsh + Steppe.
    Buckets        []string  // supply buckets this forager fills
}

var profiles = map[int]*ForagerProfile{
    371: { // Marsh Forager (Vella)
        Kind:           KindMarsh,
        MobId:          371,
        Name:           "Vella",
        SanctuaryRoom:  4123,
        TerritoryRooms: []int{4177, 4178, 4179, 4180, 4181, 4182, 4183, 4184, 4185, 4186, 4187, 4188, 4189, 4190, 4191, 4192, 4193, 4194, 4195, 4196},
        PreyWhitelist:  []int{367 /*marsh rat*/, 368 /*dragonfly swarm*/},
        VendorRooms:    []int{4102, 4103, 4105, 4106, 4125, 4126, 4135, 4143},
        Buckets:        []string{"stillwater", "base", "overlap"},
    },
    243: { // Steppe Forager (Halix)
        Kind:           KindSteppe,
        MobId:          243,
        Name:           "Halix",
        SanctuaryRoom:  468,
        TerritoryRooms: []int{3000, 3001, 3002, 3003, 3005, 3006, 3015, 3019, 3020, 3021, 3022, 3023, 3024, 3025, 3026, 3027, 3028, 3029},
        PreyWhitelist:  []int{200 /*steppe rat*/, 201 /*dust crow*/, 213 /*dust hare*/, 234 /*ground squirrel*/, 214 /*sage grouse*/, 231 /*tumble beetle*/},
        VendorRooms:    []int{464, 470, 471, 475, 480, 481, 482, 483, 507},
        Buckets:        []string{"base", "overlap"},
    },
    366: { // Fernway Forager (Kessa)
        Kind:           KindFernway,
        MobId:          366,
        Name:           "Kessa",
        SanctuaryRoom:  4197, // Forager's Camp (created in Task 14)
        TerritoryRooms: []int{4157, 4158, 4159, 4160, 4161, 4162, 4163, 4164, 4165, 4166, 4167, 4168, 4169, 4170, 4171, 4172, 4173, 4174, 4175, 4176},
        PreyWhitelist:  []int{360 /*wild hare*/, 362 /*honey bees*/},
        VendorRooms:    nil,
        MeetingRoom:    4038,
        Buckets:        []string{"fernway"},
    },
}

// ProfileFor returns the static profile for a forager by mob ID, or
// nil if the mob ID is not a registered forager.
func ProfileFor(mobId int) *ForagerProfile {
    return profiles[mobId]
}

// AllProfiles returns every registered forager. Stable order by Kind.
func AllProfiles() []*ForagerProfile {
    out := make([]*ForagerProfile, 0, len(profiles))
    for _, k := range []int{371, 243, 366} {
        if p := profiles[k]; p != nil {
            out = append(out, p)
        }
    }
    return out
}
```

- [ ] **Step 3: Write tests.**

In `internal/forager/state_test.go`:

```go
package forager

import "testing"

func TestStateNameRoundTrip(t *testing.T) {
    for s := StateResting; s <= StateRecalling; s++ {
        name := s.Name()
        got, ok := ParseState(name)
        if !ok || got != s {
            t.Errorf("roundtrip %v -> %q -> %v,%v", s, name, got, ok)
        }
    }
}

func TestAdvanceStateWraps(t *testing.T) {
    if got := AdvanceState(StateRecalling); got != StateResting {
        t.Errorf("recall + 1 = %v, want resting", got)
    }
}

func TestParseStateUnknownReturnsZeroFalse(t *testing.T) {
    s, ok := ParseState("not_a_state")
    if ok || s != StateResting {
        t.Errorf("got %v,%v want StateResting,false", s, ok)
    }
}
```

In `internal/forager/territory_test.go`:

```go
package forager

import "testing"

func TestProfileFor_KnownIds(t *testing.T) {
    cases := []int{371, 243, 366}
    for _, id := range cases {
        if ProfileFor(id) == nil {
            t.Errorf("ProfileFor(%d) = nil, want profile", id)
        }
    }
}

func TestProfileFor_UnknownReturnsNil(t *testing.T) {
    if ProfileFor(99999) != nil {
        t.Error("expected nil for unknown mob id")
    }
}

func TestAllProfilesHasThree(t *testing.T) {
    if got := len(AllProfiles()); got != 3 {
        t.Errorf("AllProfiles() len = %d, want 3", got)
    }
}

func TestProfileBucketsNonEmpty(t *testing.T) {
    for _, p := range AllProfiles() {
        if len(p.Buckets) == 0 {
            t.Errorf("profile %s has empty Buckets", p.Name)
        }
    }
}

func TestFernwayHasMeetingRoomOthersDont(t *testing.T) {
    if ProfileFor(366).MeetingRoom != 4038 {
        t.Error("Fernway forager missing meeting room 4038")
    }
    if ProfileFor(371).MeetingRoom != 0 {
        t.Error("Marsh forager should not have meeting room")
    }
    if ProfileFor(243).MeetingRoom != 0 {
        t.Error("Steppe forager should not have meeting room")
    }
}

func TestVendorRoomsExclusiveToTownForagers(t *testing.T) {
    if len(ProfileFor(371).VendorRooms) == 0 {
        t.Error("Marsh forager should have vendor rooms")
    }
    if len(ProfileFor(243).VendorRooms) == 0 {
        t.Error("Steppe forager should have vendor rooms")
    }
    if len(ProfileFor(366).VendorRooms) != 0 {
        t.Error("Fernway forager should not have vendor rooms (caravan handoff)")
    }
}
```

- [ ] **Step 4: Run tests.**

Run: `go test ./internal/forager/... -v`
Expected: PASS.

- [ ] **Step 5: Commit.**

```bash
git add internal/forager/
git commit -m "$(cat <<'EOF'
feat(forager): state enum + per-forager profile registry

State enum mirrors the caravan state pattern. ForagerProfile holds
static data (territory, prey whitelist, vendor list, sanctuary,
meeting point, supply buckets) keyed by mob ID. Three foragers
registered: Marsh (371), Steppe (243), Fernway (366).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 8: Create `forager_step` btree action + supporting conditions

**Files:**
- Create: `internal/behaviortree/actions_forager.go`
- Create: `internal/behaviortree/conditions_forager.go`
- Create: `internal/behaviortree/actions_forager_test.go`

- [ ] **Step 1: Read `actions_caravan.go` to understand the `caravan_step` action's shape.**

This is the closest analog. Note: how it reads MobState, calls into the package's Advance/Should* helpers, issues pathto / runs commands, and writes MobState back.

- [ ] **Step 2: Implement `forager_step`.**

In `internal/behaviortree/actions_forager.go`:

```go
package behaviortree

import (
    "fmt"
    "strconv"

    "github.com/GoMudEngine/GoMud/internal/configs"
    "github.com/GoMudEngine/GoMud/internal/economy"
    "github.com/GoMudEngine/GoMud/internal/forager"
    "github.com/GoMudEngine/GoMud/internal/items"
    "github.com/GoMudEngine/GoMud/internal/mobs"
    "github.com/GoMudEngine/GoMud/internal/rooms"
    "github.com/GoMudEngine/GoMud/internal/shops"
    "github.com/GoMudEngine/GoMud/internal/skills"
    "github.com/GoMudEngine/GoMud/internal/usercommands"
    "github.com/GoMudEngine/GoMud/internal/util"
)

const (
    keyForagerState   = "forager_state"
    keyForageTimer    = "forager_forage_timer"
    keyFatigueTimer   = "forager_fatigue_timer"
    keyVisitIndex     = "forager_visit_index"
    keyWaitTimer      = "forager_wait_timer"
)

// foragerStepAction implements the `forager_step` btree action.
//
// It reads MobState["forager_state"] (defaulting to "resting" on
// first tick), dispatches to a per-state handler, then writes the
// (possibly-advanced) state back. Pure-glue code: state-machine math
// lives in internal/forager.
func foragerStepAction(ctx *Context) Status {
    mob := ctx.Mob
    if mob == nil {
        return StatusFailure
    }
    profile := forager.ProfileFor(mob.MobId)
    if profile == nil {
        return StatusFailure
    }

    // HP-emergency short-circuit. Drops directly into recalling
    // regardless of current state.
    cfg := configs.GetBalanceConfig()
    hpRatio := float64(mob.Character.Health) / float64(mob.Character.HealthMax.Value)
    if hpRatio <= cfg.ForagerHPRecallThresholdPct {
        if state, _ := readState(mob); state != forager.StateRecalling {
            writeState(mob, forager.StateRecalling)
            return castFoldRecall(mob)
        }
    }

    state, _ := readState(mob)
    switch state {
    case forager.StateResting:
        return tickResting(mob, profile)
    case forager.StateTravelingToTerritory:
        return tickTravelingToTerritory(mob, profile)
    case forager.StateForaging:
        return tickForaging(mob, profile, cfg)
    case forager.StateTravelingToDropoff:
        return tickTravelingToDropoff(mob, profile)
    case forager.StateDelivering:
        return tickDelivering(mob, profile, cfg)
    case forager.StateRecalling:
        return tickRecalling(mob, profile)
    }
    return StatusFailure
}

func readState(mob *mobs.Mob) (forager.ForagerState, bool) {
    raw := mob.GetState(keyForagerState)
    if raw == "" {
        return forager.StateResting, false
    }
    s, ok := forager.ParseState(raw)
    return s, ok
}

func writeState(mob *mobs.Mob, s forager.ForagerState) {
    mob.SetState(keyForagerState, s.Name())
}

func tickResting(mob *mobs.Mob, p *forager.ForagerProfile) Status {
    // If at sanctuary AND fully recovered, advance.
    if mob.Character.RoomId != p.SanctuaryRoom {
        // Got knocked off-sanctuary somehow — pathto back.
        runCmd(mob, fmt.Sprintf("pathto %d", p.SanctuaryRoom))
        return StatusRunning
    }
    if mob.Character.Health < mob.Character.HealthMax.Value {
        return StatusRunning
    }
    // Reset timers and advance.
    mob.SetState(keyForageTimer, "0")
    mob.SetState(keyFatigueTimer, "0")
    mob.SetState(keyVisitIndex, "0")
    mob.SetState(keyWaitTimer, "0")
    writeState(mob, forager.StateTravelingToTerritory)
    return StatusSuccess
}

func tickTravelingToTerritory(mob *mobs.Mob, p *forager.ForagerProfile) Status {
    if len(p.TerritoryRooms) == 0 {
        return StatusFailure
    }
    entry := p.TerritoryRooms[0]
    if mob.Character.RoomId == entry || contains(p.TerritoryRooms, mob.Character.RoomId) {
        writeState(mob, forager.StateForaging)
        return StatusSuccess
    }
    runCmd(mob, fmt.Sprintf("pathto %d", entry))
    return StatusRunning
}

func tickForaging(mob *mobs.Mob, p *forager.ForagerProfile, cfg *configs.BalanceConfig) Status {
    // Fatigue + carry checks.
    fatigue := getInt(mob, keyFatigueTimer) + 1
    mob.SetState(keyFatigueTimer, strconv.Itoa(fatigue))
    if fatigue >= 480 || carryRatio(mob) >= cfg.ForagerCarryThresholdPct {
        writeState(mob, forager.StateTravelingToDropoff)
        return StatusSuccess
    }

    // Forage tick.
    forageT := getInt(mob, keyForageTimer) + 1
    if forageT >= cfg.ForagerForageDwellRounds {
        mob.SetState(keyForageTimer, "0")
        attemptForage(mob, p)
    } else {
        mob.SetState(keyForageTimer, strconv.Itoa(forageT))
    }

    // Wander 1 random adjacent territory room.
    if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
        // Use existing helper (pattern from actions_room.go) to pick
        // a random territory-resident neighbor.
        wanderToTerritoryNeighbor(mob, p, room)
    }

    // Salvage any corpse in the room.
    runCmd(mob, "salvage corpse")

    return StatusRunning
}

func tickTravelingToDropoff(mob *mobs.Mob, p *forager.ForagerProfile) Status {
    var dest int
    switch p.Kind {
    case forager.KindFernway:
        dest = p.MeetingRoom
    default:
        if len(p.VendorRooms) == 0 {
            return StatusFailure
        }
        dest = p.VendorRooms[0]
        mob.SetState(keyVisitIndex, "0")
    }
    if mob.Character.RoomId == dest {
        writeState(mob, forager.StateDelivering)
        return StatusSuccess
    }
    runCmd(mob, fmt.Sprintf("pathto %d", dest))
    return StatusRunning
}

func tickDelivering(mob *mobs.Mob, p *forager.ForagerProfile, cfg *configs.BalanceConfig) Status {
    if p.Kind == forager.KindFernway {
        return tickDeliveringFernway(mob, p, cfg)
    }
    return tickDeliveringTown(mob, p)
}

func tickDeliveringTown(mob *mobs.Mob, p *forager.ForagerProfile) Status {
    idx := getInt(mob, keyVisitIndex)
    if idx >= len(p.VendorRooms) {
        writeState(mob, forager.StateRecalling)
        return StatusSuccess
    }
    target := p.VendorRooms[idx]
    if mob.Character.RoomId != target {
        runCmd(mob, fmt.Sprintf("pathto %d", target))
        return StatusRunning
    }
    // Fire RestockBuckets at every shop-bearing mob in this room.
    visitVendorsInRoom(target, p.Buckets)
    mob.SetState(keyVisitIndex, strconv.Itoa(idx+1))
    return StatusRunning
}

func tickDeliveringFernway(mob *mobs.Mob, p *forager.ForagerProfile, cfg *configs.BalanceConfig) Status {
    if mob.Character.RoomId != p.MeetingRoom {
        runCmd(mob, fmt.Sprintf("pathto %d", p.MeetingRoom))
        return StatusRunning
    }
    waitT := getInt(mob, keyWaitTimer) + 1
    mob.SetState(keyWaitTimer, strconv.Itoa(waitT))
    if waitT >= cfg.ForagerWaitTimeoutRounds {
        // Caravan never came — bail home with the satchel.
        writeState(mob, forager.StateRecalling)
        return StatusSuccess
    }
    // Caravan-arrival detection happens on the caravan side
    // (Task 10's fernway_pickup substate fires the handoff).
    return StatusRunning
}

func tickRecalling(mob *mobs.Mob, p *forager.ForagerProfile) Status {
    if mob.Character.RoomId == p.SanctuaryRoom {
        writeState(mob, forager.StateResting)
        return StatusSuccess
    }
    return castFoldRecall(mob)
}

func castFoldRecall(mob *mobs.Mob) Status {
    runCmd(mob, "cast fold-recall")
    return StatusRunning
}

func visitVendorsInRoom(roomId int, buckets []string) {
    room := rooms.LoadRoom(roomId)
    if room == nil {
        return
    }
    for _, instId := range room.GetMobs(rooms.FindAll) {
        m := mobs.GetInstance(instId)
        if m == nil || !m.HasShop() {
            continue
        }
        si := shops.GetShopInventory(m.Zone, m.MobId, roomId)
        if si == nil {
            continue
        }
        if si.RestockBuckets(buckets) {
            room.SendText(fmt.Sprintf(
                `<ansi fg="mobname">%s</ansi> lays a satchel of mats on the counter.`,
                "the forager"))
        }
    }
}

func attemptForage(mob *mobs.Mob, p *forager.ForagerProfile) {
    room := rooms.LoadRoom(mob.Character.RoomId)
    if room == nil {
        return
    }
    biome := room.GetBiome()
    searchRank := mob.Character.GetSkillLevel(skills.Search)
    score := float64(mob.Character.Stats.Perception.ValueAdj) +
        float64(searchRank) // simpler than combat.SkillMultiplier for NPC path

    result := usercommands.ForageCore(usercommands.ForageAttempt{
        Biome:       biome.BiomeId,
        SearchScore: score,
        // Foragers don't operate at night — they're at sanctuary
        // resting between cycles. Skip the night-yield branch.
        AtNight: false,
    })
    if !result.Found {
        return
    }
    item := items.New(result.ItemId)
    if !item.IsValid() {
        return
    }
    mob.Character.StoreItem(item)
    room.SendText(fmt.Sprintf(
        `<ansi fg="mobname">%s</ansi> stoops over a patch of growth and tucks something into a satchel.`,
        p.Name))
}

func wanderToTerritoryNeighbor(mob *mobs.Mob, p *forager.ForagerProfile, room *rooms.Room) {
    // Iterate the room's exits; pick one that lands on a territory room.
    candidates := []string{}
    for dir, exit := range room.Exits {
        if contains(p.TerritoryRooms, exit.RoomId) {
            candidates = append(candidates, dir)
        }
    }
    if len(candidates) == 0 {
        return
    }
    runCmd(mob, candidates[util.Rand(len(candidates))])
}

func getInt(mob *mobs.Mob, key string) int {
    n, _ := strconv.Atoi(mob.GetState(key))
    return n
}

func contains(haystack []int, needle int) bool {
    for _, h := range haystack {
        if h == needle {
            return true
        }
    }
    return false
}

func carryRatio(mob *mobs.Mob) float64 {
    cap := mob.Character.CarryCapacity()
    if cap <= 0 {
        return 0
    }
    return float64(mob.Character.CarryWeight()) / float64(cap)
}

func runCmd(mob *mobs.Mob, cmd string) {
    mob.Command(cmd) // adjust to whatever the mob-command-issue API is in this codebase
}
```

(Read `actions_caravan.go` for the actual Mob/Context API names — `Mob.Command` may be `mob.Command()` or `events.AddToQueue(events.Input{...})` — match the existing pattern. Same for `Status` enum / `Context` type / how MobState read/write actually works.)

- [ ] **Step 3: Write supporting conditions.**

In `internal/behaviortree/conditions_forager.go`:

```go
package behaviortree

import (
    "github.com/GoMudEngine/GoMud/internal/configs"
    "github.com/GoMudEngine/GoMud/internal/forager"
    "github.com/GoMudEngine/GoMud/internal/mobs"
)

// mobCanSafelyEngage reports whether the forager mob may engage the
// given target. Used in the `foraging` state's combat sub-routine.
func mobCanSafelyEngage(self *mobs.Mob, target *mobs.Mob) bool {
    if self == nil || target == nil {
        return false
    }
    profile := forager.ProfileFor(self.MobId)
    if profile == nil {
        return false
    }
    // Prey whitelist
    found := false
    for _, prey := range profile.PreyWhitelist {
        if target.MobId == prey {
            found = true
            break
        }
    }
    if !found {
        return false
    }
    // 60% stat-pool gate
    if effectiveStatSum(target) > effectiveStatSum(self)*0.6 {
        return false
    }
    // HP gate
    return float64(self.Character.Health)/float64(self.Character.HealthMax.Value) >= 0.75
}

func mobInventoryAtThreshold(mob *mobs.Mob) bool {
    if mob == nil {
        return false
    }
    cfg := configs.GetBalanceConfig()
    cap := mob.Character.CarryCapacity()
    if cap <= 0 {
        return false
    }
    return float64(mob.Character.CarryWeight())/float64(cap) >= cfg.ForagerCarryThresholdPct
}

func mobHPBelowRecallThreshold(mob *mobs.Mob) bool {
    if mob == nil {
        return false
    }
    cfg := configs.GetBalanceConfig()
    return float64(mob.Character.Health)/float64(mob.Character.HealthMax.Value) <= cfg.ForagerHPRecallThresholdPct
}

func effectiveStatSum(m *mobs.Mob) int {
    s := &m.Character.Stats
    return s.Strength.ValueAdj + s.Dexterity.ValueAdj + s.Vitality.ValueAdj +
        s.Perception.ValueAdj + s.Willpower.ValueAdj + s.Charisma.ValueAdj
}
```

- [ ] **Step 4: Write tests.**

In `internal/behaviortree/actions_forager_test.go`:

```go
package behaviortree

import (
    "testing"

    "github.com/GoMudEngine/GoMud/internal/forager"
)

func TestForagerStep_ReadsAndWritesState(t *testing.T) {
    // Build a fake forager mob anchored at its sanctuary.
    mob := newTestMob(371) // Marsh forager
    mob.Character.RoomId = 4123
    mob.Character.Health = mob.Character.HealthMax.Value

    if got, _ := readState(mob); got != forager.StateResting {
        t.Errorf("default state = %v, want resting", got)
    }
    // After tick, fully-rested forager at sanctuary should advance.
    foragerStepAction(&Context{Mob: mob})
    if got, _ := readState(mob); got != forager.StateTravelingToTerritory {
        t.Errorf("after tick state = %v, want traveling_to_territory", got)
    }
}

func TestForagerStep_HPEmergencyForcesRecall(t *testing.T) {
    mob := newTestMob(371)
    mob.Character.RoomId = 4180
    mob.SetState(keyForagerState, "foraging")
    mob.Character.Health = 10 // out of 100
    foragerStepAction(&Context{Mob: mob})
    if got, _ := readState(mob); got != forager.StateRecalling {
        t.Errorf("expected recalling after HP < 50%%, got %v", got)
    }
}
```

(`newTestMob(...)` is a test helper to instantiate a mob with the registered profile's defaults. Look at how existing btree tests build test mobs — `actions_caravan_test.go` is the closest reference; use its pattern.)

- [ ] **Step 5: Run tests.**

Run: `go test ./internal/behaviortree/... -run Forager -v`
Expected: PASS.

- [ ] **Step 6: Run full behaviortree tests.**

Run: `go test ./internal/behaviortree/...`
Expected: PASS, no regressions.

- [ ] **Step 7: Register the action and conditions in the btree action map.**

In `internal/behaviortree/actions.go` (or wherever `caravan_step` is registered — `grep -nF caravan_step internal/behaviortree/`), add an entry mapping `"forager_step"` to `foragerStepAction`. Same for the new conditions in conditions registration.

- [ ] **Step 8: Run the full test suite.**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 9: Commit.**

```bash
git add internal/behaviortree/actions_forager.go internal/behaviortree/conditions_forager.go internal/behaviortree/actions_forager_test.go internal/behaviortree/actions.go internal/behaviortree/conditions.go
git commit -m "$(cat <<'EOF'
feat(btree): forager_step action + 3 forager conditions

forager_step drives the per-forager state machine via MobState. HP
emergency short-circuits any state to recalling. Foraging tick fires
ForageCore (3.1 reuse), salvages corpses on encounter, wanders within
territory. Delivering tick fires RestockBuckets at each town vendor
(Marsh + Steppe) or idles at the meeting point (Fernway).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 9: Add `fernway_pickup` substate to caravan state machine

**Files:**
- Modify: `internal/caravan/state.go` — add 2 substates
- Modify: `internal/caravan/state_test.go` — extend tests
- Modify: `internal/caravan/visit.go` — switch to RestockBuckets

- [ ] **Step 1: Read the existing state enum to confirm the cycle.**

Already known from the spec. The new states slot in mid-transit:

```
ThornwallDwell → OutboundTransit → OutboundFernwayPickup → StillwaterRoute →
StillwaterDwell → InboundTransit → InboundFernwayPickup → ThornwallRoute → ...
```

- [ ] **Step 2: Add the 2 new states to `state.go`.**

In `internal/caravan/state.go`, extend the enum:

```go
const (
    StateThornwallDwell  CaravanState = iota
    StateOutboundTransit
    StateOutboundFernwayPickup
    StateStillwaterRoute
    StateStillwaterDwell
    StateInboundTransit
    StateInboundFernwayPickup
    StateThornwallRoute
)

var stateNames = map[CaravanState]string{
    StateThornwallDwell:        "thornwall_dwell",
    StateOutboundTransit:       "outbound_transit",
    StateOutboundFernwayPickup: "outbound_fernway_pickup",
    StateStillwaterRoute:       "stillwater_route",
    StateStillwaterDwell:       "stillwater_dwell",
    StateInboundTransit:        "inbound_transit",
    StateInboundFernwayPickup:  "inbound_fernway_pickup",
    StateThornwallRoute:        "thornwall_route",
}
```

Update `AdvanceState` (the `% 6` becomes `% 8`):

```go
func AdvanceState(cur CaravanState) CaravanState {
    return (cur + 1) % 8
}
```

Update predicate helpers:

```go
func IsTransitState(s CaravanState) bool {
    return s == StateOutboundTransit || s == StateInboundTransit ||
        s == StateOutboundFernwayPickup || s == StateInboundFernwayPickup
}

func IsFernwayPickupState(s CaravanState) bool {
    return s == StateOutboundFernwayPickup || s == StateInboundFernwayPickup
}

func RouteForState(s CaravanState) *Route {
    switch s {
    case StateOutboundTransit, StateOutboundFernwayPickup, StateStillwaterRoute:
        return &OutboundRoute
    case StateInboundTransit, StateInboundFernwayPickup, StateThornwallRoute:
        return &InboundRoute
    }
    return nil
}
```

- [ ] **Step 3: Extend tests.**

In `internal/caravan/state_test.go`, add:

```go
func TestAdvanceState_IncludesFernwayPickup(t *testing.T) {
    cases := []struct {
        from CaravanState
        to   CaravanState
    }{
        {StateOutboundTransit, StateOutboundFernwayPickup},
        {StateOutboundFernwayPickup, StateStillwaterRoute},
        {StateInboundTransit, StateInboundFernwayPickup},
        {StateInboundFernwayPickup, StateThornwallRoute},
        {StateThornwallRoute, StateThornwallDwell}, // wraps
    }
    for _, c := range cases {
        if got := AdvanceState(c.from); got != c.to {
            t.Errorf("AdvanceState(%v) = %v, want %v", c.from, got, c.to)
        }
    }
}

func TestIsFernwayPickupState(t *testing.T) {
    if !IsFernwayPickupState(StateOutboundFernwayPickup) ||
        !IsFernwayPickupState(StateInboundFernwayPickup) {
        t.Error("expected pickup states to return true")
    }
    if IsFernwayPickupState(StateOutboundTransit) {
        t.Error("transit state should not be pickup")
    }
}
```

- [ ] **Step 4: Switch `visit.go` to use RestockBuckets + caravan_load.**

In `internal/caravan/visit.go`, replace `Restock()` call with `RestockBuckets(buckets)`, where `buckets` comes from a parameter:

```go
// VisitVendorsInRoom calls RestockBuckets() on every shop-bearing mob
// in the given room, gated by the caravan's current load buckets.
// Returns the list of mob names that received a delivery.
func VisitVendorsInRoom(roomId int, buckets []string) []string {
    room := rooms.LoadRoom(roomId)
    if room == nil {
        return nil
    }
    var visited []string
    for _, instId := range room.GetMobs(rooms.FindAll) {
        mob := mobs.GetInstance(instId)
        if mob == nil || !mob.HasShop() {
            continue
        }
        si := shops.GetShopInventory(mob.Zone, mob.MobId, roomId)
        if si == nil {
            continue
        }
        if si.RestockBuckets(buckets) {
            visited = append(visited, mob.Character.Name)
        }
    }
    return visited
}
```

(Caller — `actions_caravan.go` — will be updated to pass buckets in Task 10.)

- [ ] **Step 5: Run state + visit tests.**

Run: `go test ./internal/caravan/... -v`
Expected: PASS (the visit-test caller signature change may need a tiny test-side update — fix inline).

- [ ] **Step 6: Commit.**

```bash
git add internal/caravan/state.go internal/caravan/state_test.go internal/caravan/visit.go internal/caravan/visit_test.go
git commit -m "$(cat <<'EOF'
feat(caravan): fernway_pickup substate + bucket-aware vendor visit

Adds 2 new substates inside outbound + inbound transit legs for the
Fernway-forager handoff. VisitVendorsInRoom now takes a bucket list
and delegates to RestockBuckets, gating which slots each visit fills.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 10: Wire `fernway_pickup` in actions_caravan.go

**Files:**
- Modify: `internal/behaviortree/actions_caravan.go` — handle 2 new states + caravan_load tracking

- [ ] **Step 1: Read the existing caravan_step dispatcher.**

Note how it currently switches on state and decides advance.

- [ ] **Step 2: Add caravan_load tracking + fernway_pickup handlers.**

In `internal/behaviortree/actions_caravan.go`, extend the dispatcher:

```go
const keyCaravanLoad = "caravan_load"

// caravanLoadGet returns the current load bucket list.
func caravanLoadGet(mob *mobs.Mob) []string {
    raw := mob.GetState(keyCaravanLoad)
    if raw == "" {
        return nil
    }
    return strings.Split(raw, ",")
}

func caravanLoadSet(mob *mobs.Mob, buckets []string) {
    mob.SetState(keyCaravanLoad, strings.Join(buckets, ","))
}

func caravanLoadAppend(mob *mobs.Mob, b string) {
    cur := caravanLoadGet(mob)
    for _, x := range cur {
        if x == b { return }
    }
    cur = append(cur, b)
    caravanLoadSet(mob, cur)
}
```

In the state dispatcher, add cases:

```go
case caravan.StateOutboundTransit:
    // Existing transit behavior — pathto Fernway meeting point first.
    return tickTransitToFernway(mob, /*outbound*/ true)
case caravan.StateOutboundFernwayPickup:
    return tickFernwayPickup(mob, /*nextState*/ caravan.StateStillwaterRoute)
case caravan.StateStillwaterRoute:
    // Existing — but pre-set caravan_load to ["stillwater"] at start
    if !mob.GetState("caravan_load_set_stillwater") {
        caravanLoadAppend(mob, "stillwater")
        mob.SetState("caravan_load_set_stillwater", "1")
    }
    return tickStillwaterRoute(mob)
// ... mirror for inbound
case caravan.StateThornwallRoute:
    if !mob.GetState("caravan_load_set_thornwall") {
        caravanLoadAppend(mob, "thornwall")
        mob.SetState("caravan_load_set_thornwall", "1")
    }
    return tickThornwallRoute(mob)
```

(The `caravan_load_set_*` flags reset on the next dwell-state entry.)

`tickFernwayPickup`:

```go
func tickFernwayPickup(mob *mobs.Mob, nextState caravan.CaravanState) Status {
    cfg := configs.GetBalanceConfig()
    pickupRoom := 4038
    if mob.Character.RoomId != pickupRoom {
        runCmd(mob, fmt.Sprintf("pathto %d", pickupRoom))
        return StatusRunning
    }
    // Dwell N rounds; on first tick at room, scan for forager.
    dwellT := getInt(mob, "fernway_pickup_dwell")
    mob.SetState("fernway_pickup_dwell", strconv.Itoa(dwellT+1))

    if dwellT == 0 {
        if foragerInRoom(pickupRoom) {
            caravanLoadAppend(mob, "fernway")
            room := rooms.LoadRoom(pickupRoom)
            room.SendText(fmt.Sprintf(
                `<ansi fg="mobname">Kessa</ansi> hands a satchel to the caravan; the wagon rolls on.`))
        }
    }
    if dwellT >= cfg.FernwayPickupDwellRounds {
        mob.SetState("fernway_pickup_dwell", "0")
        // Advance.
        mob.SetState("caravan_state", nextState.Name())
    }
    return StatusRunning
}

func foragerInRoom(roomId int) bool {
    room := rooms.LoadRoom(roomId)
    if room == nil { return false }
    for _, instId := range room.GetMobs(rooms.FindAll) {
        m := mobs.GetInstance(instId)
        if m == nil { continue }
        if m.MobId == 366 { // Fernway forager
            return true
        }
    }
    return false
}
```

In `tickStillwaterRoute` (existing handler), update the `VisitVendorsInRoom` call:

```go
visited := caravan.VisitVendorsInRoom(target, caravanLoadGet(mob))
```

Same in `tickThornwallRoute`. Ensure the `caravan_load_set_*` and `fernway_pickup_dwell` MobState fields are cleared when the caravan enters the next dwell state (resetting for the next cycle):

```go
case caravan.StateThornwallDwell, caravan.StateStillwaterDwell:
    if dwellT == 0 {
        // First tick of dwell — clear cycle state
        caravanLoadSet(mob, nil)
        mob.SetState("caravan_load_set_stillwater", "")
        mob.SetState("caravan_load_set_thornwall", "")
        mob.SetState("fernway_pickup_dwell", "0")
    }
```

- [ ] **Step 3: Add a test for the fernway_pickup substate.**

In `internal/behaviortree/actions_caravan_test.go`, add:

```go
func TestCaravan_FernwayPickup_AppendsLoad(t *testing.T) {
    caravan := newTestCaravanLeader()
    caravan.Character.RoomId = 4038
    putForagerInRoom(t, 4038, 366) // helper similar to existing test fixtures
    caravan.SetState("caravan_state", "outbound_fernway_pickup")
    caravanStepAction(&Context{Mob: caravan})

    load := caravan.GetState("caravan_load")
    if !strings.Contains(load, "fernway") {
        t.Errorf("caravan_load = %q, want to contain fernway", load)
    }
}

func TestCaravan_FernwayPickup_NoForagerNoLoad(t *testing.T) {
    caravan := newTestCaravanLeader()
    caravan.Character.RoomId = 4038
    caravan.SetState("caravan_state", "outbound_fernway_pickup")
    caravanStepAction(&Context{Mob: caravan})

    load := caravan.GetState("caravan_load")
    if strings.Contains(load, "fernway") {
        t.Errorf("caravan_load = %q, want no fernway (forager absent)", load)
    }
}
```

- [ ] **Step 4: Run tests.**

Run: `go test ./internal/behaviortree/... -run Caravan -v`
Expected: PASS.

- [ ] **Step 5: Commit.**

```bash
git add internal/behaviortree/actions_caravan.go internal/behaviortree/actions_caravan_test.go
git commit -m "$(cat <<'EOF'
feat(btree): caravan fernway_pickup handler + caravan_load tracking

Adds tickFernwayPickup which dwells at North Road 4038 long enough to
detect the Fernway forager and acquire the fernway bucket flag.
caravan_load is appended/cleared across the cycle and consumed by
VisitVendorsInRoom for bucket-aware vendor restock.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 11: Create `sanctuary` mutator + wire on existing rooms

**Files:**
- Create: `_datafiles/world/dogmud/mutators/sanctuary.yaml`
- Modify: `_datafiles/world/dogmud/rooms/thornwall_city/468.yaml`
- Modify: `_datafiles/world/dogmud/rooms/stillwater/4123.yaml`
- Modify: `_datafiles/world/dogmud/rooms/sanctum_basin/{101..120}.yaml` (20 rooms)

- [ ] **Step 1: Create the sanctuary mutator YAML.**

Write `_datafiles/world/dogmud/mutators/sanctuary.yaml`:

```yaml
mutatorid: sanctuary
regenmultiplier: 5.0
descriptionmodifier:
  behavior: append
  text: A peace older than the stones themselves settles over you here. Wounds close more easily and breath comes more deeply.
  colorpattern: pearl
```

- [ ] **Step 2: Add mutator to room 468 (Thornwall Temple Interior).**

Append to `_datafiles/world/dogmud/rooms/thornwall_city/468.yaml`:

```yaml
mutators:
- mutatorid: sanctuary
```

(If the file already has top-level keys, place `mutators:` alongside them — match indentation of existing top-level keys.)

- [ ] **Step 3: Add mutator to room 4123 (Stillwater Temple Interior).**

Append the same `mutators:` block to `_datafiles/world/dogmud/rooms/stillwater/4123.yaml`.

- [ ] **Step 4: Add mutator to all 20 Sanctum Basin tutorial rooms.**

For each of rooms 101-120 (`_datafiles/world/dogmud/rooms/sanctum_basin/{101..120}.yaml`), append:

```yaml
mutators:
- mutatorid: sanctuary
```

(20 file edits — script-friendly. If a room already has a `mutators:` block, append the entry rather than overwriting.)

- [ ] **Step 5: Boot the server and verify load.**

Run: `go run . -test_boot` (or whatever the boot-only smoke command is in this repo; if none, `go run .` and Ctrl-C after `mobs.LoadDataFiles()` finishes without panic).

Expected: clean load, no panics. Look for `mutators.LoadDataFiles() loadedCount=N+1` (one more than before).

- [ ] **Step 6: Re-run the auto-heal hook tests from Task 2 — sanctuary tests should now pass.**

Run: `go test ./internal/hooks/... -run RoomRegenMultiplier -v`
Expected: ALL tests now PASS, including TestSingleSanctuary and TestStacksMultiplicatively.

- [ ] **Step 7: Commit.**

```bash
git add _datafiles/world/dogmud/mutators/sanctuary.yaml _datafiles/world/dogmud/rooms/thornwall_city/468.yaml _datafiles/world/dogmud/rooms/stillwater/4123.yaml _datafiles/world/dogmud/rooms/sanctum_basin/10{1..9}.yaml _datafiles/world/dogmud/rooms/sanctum_basin/11{0..9}.yaml _datafiles/world/dogmud/rooms/sanctum_basin/120.yaml
git commit -m "$(cat <<'EOF'
feat(content): sanctuary mutator + wire on temples + tutorial zone

Replaces the hardcoded auto-heal-hook room-id switch with a YAML
mutator. Sanctuary applied to Thornwall Temple Interior (468),
Stillwater Temple Interior (4123), and all 20 Sanctum Basin tutorial
rooms (101-120). Stillwater Temple becomes a known sanctuary for the
Marsh forager's recall destination; Sanctum tutorial regen behavior
preserved.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 12: Create 3 forager weapons

**Files:**
- Create: `_datafiles/world/dogmud/items/weapons-10000/10033-marsh_gaff_hook.yaml`
- Create: `_datafiles/world/dogmud/items/weapons-10000/10034-steppe_hunting_spear.yaml`
- Create: `_datafiles/world/dogmud/items/weapons-10000/10035-fernway_handaxe.yaml`

- [ ] **Step 1: Read 2 existing low-tier weapon YAMLs to confirm the schema.**

Run: read `_datafiles/world/dogmud/items/weapons-10000/10031-lake_iron_hook_spear.yaml` and `10025-edrins_gnarled_staff.yaml`.

Note: required fields, weapon-subtype values, damage_multiplier scale.

- [ ] **Step 2: Author the gaff hook (puncture 1H).**

Mirror the lake-iron hook spear's structure. Modest damage_multiplier (~0.45-0.55 range — match low-tier puncture).

- [ ] **Step 3: Author the hunting spear (puncture 1H).**

Pattern after a 1H spear in the existing range. damage_multiplier ~0.50.

- [ ] **Step 4: Author the hand axe (slashing 1H).**

Pattern after existing 1H slashing weapons. damage_multiplier ~0.55.

- [ ] **Step 5: Boot the server.**

Run: `go run .` to first-line stop. Watch for `items.LoadDataFiles() loadedCount=...` to grow by 3.

Expected: clean boot. Three new items load.

- [ ] **Step 6: Commit.**

```bash
git add _datafiles/world/dogmud/items/weapons-10000/1003{3,4,5}-*.yaml
git commit -m "$(cat <<'EOF'
feat(items): forager weapons (gaff hook, hunting spear, hand axe)

Three low-tier 1H weapons for the Stage 3.1 foragers. Gaff hook for
the Marsh forager (Vella), hunting spear for Steppe (Halix), hand
axe for Fernway (Kessa). Statted to match existing low-tier weapons
in the same combat skill.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 13: Create Marsh Forager (Vella) — mob + behavior + dialogue

**Files:**
- Create: `_datafiles/world/dogmud/mobs/stillwater_marsh/371-vella.yaml`
- Create: `_datafiles/world/dogmud/behaviors/stillwater_marsh/371-vella.yaml`
- Create: `_datafiles/world/dogmud/dialogue/stillwater_marsh/371-vella.yaml`
- Modify: `_datafiles/world/dogmud/rooms/stillwater/4123.yaml` — add Vella to spawninfo

- [ ] **Step 1: Read 2 existing similar mobs as references.**

Run: read `_datafiles/world/dogmud/mobs/thornwall_city/357-ketil.yaml` (caravan crew, has player_attack_immune + fold_anchor_room) and `_datafiles/world/dogmud/mobs/marches_spur_road/275-old_edrin.yaml` (NPC fold-recall pattern).

- [ ] **Step 2: Write `mobs/stillwater_marsh/371-vella.yaml`.**

```yaml
mobid: 371
zone: Stillwater Marsh
archetype: fighting
behavior_archetype: ""              # custom btree below
statpool: 150                       # slightly above human baseline
itemdropchance: 0
groups:
  - forager
hostile: false
non_combatant: false                # engages prey wildlife
player_attack_immune: true          # rebuff direct player attacks
maxwander: -1                       # unlimited; cross-zone walking on delivery routes
activitylevel: 20
charm_immune: true
fold_anchor_room: 4123              # Stillwater Temple Interior

idlecommands:
  - 'emote checks the satchel at her hip, eyes scanning the reedline'
  - ''
  - 'emote tucks a strand of hair behind one ear and squints at the water'
  - ''

tactics:
  - trigger: health_below:50
    action: cast fold-recall
    priority: 13
  - trigger: health_below:75
    action: drink potion
    priority: 11

character:
  name: Vella
  description: |
    A weathered marshfolk woman in her middle years, slight but
    wiry, dressed in oiled canvas and waxed linen. Mud is set
    deep in the seams of her boots and the cuffs of her trousers.
    A leather satchel hangs at her hip, marked with the small
    smudges of plant resins and fish scales. She moves through
    the reeds with the careful patience of someone who has done
    this work since girlhood.
  speciesid: 1
  level: 1
  gold: 30

  stats:
    strength:    {training: 25}
    dexterity:   {training: 25}
    vitality:    {training: 25}
    perception:  {training: 30}
    willpower:   {training: 25}
    charisma:    {training: 20}

  spellbook:
    fold-recall: 25

  skills:
    weapon-combat: 15
    search:        25
    salvage:       20
    spellcasting:  15

  equipment:
    weapon:
      itemid: 10033       # marsh gaff hook
    belt:
      itemid: 30043       # potion bandolier (existing item — confirm ID at impl time)
```

(Confirm the bandolier item ID by `grep -n bandolier _datafiles/world/dogmud/items/`. If different, adjust.)

- [ ] **Step 3: Write `behaviors/stillwater_marsh/371-vella.yaml`.**

```yaml
# Marsh Forager Vella (371) — Stage 3.1 forager.
# Drives the state machine via forager_step.

tree:
  type: sequence
  children:

    - type: selector
      children:

        # On hit / in combat: HP-emergency check + counter-attack.
        - type: sequence
          event: mob_hurt
          children:
            - type: condition
              check: mob_hp_below_recall_threshold
            - type: action
              do: cast
              spell: fold-recall
        - type: sequence
          event: mob_hurt
          children:
            - type: action
              do: attack

        # Drive the forager state machine.
        - type: sequence
          event: mob_idle
          children:
            - type: action
              do: forager_step
```

- [ ] **Step 4: Write `dialogue/stillwater_marsh/371-vella.yaml`.**

Light dialogue per spec (6-10 patterns + 2-3 root nodes). Use existing forager-flavor patterns from elsewhere as reference. Greeting + weather + a couple of quest-hookable open-ended responses. Voice: practical marsh-woman; first-person.

(Schema reference: `docs/schemas/dialogue.md`. Use the same shape as a recently-created NPC like `_datafiles/world/dogmud/dialogue/marches_spur_road/275-old_edrin.yaml`.)

- [ ] **Step 5: Add Vella to `4123.yaml` spawninfo.**

In `_datafiles/world/dogmud/rooms/stillwater/4123.yaml`, append to spawninfo:

```yaml
- mobid: 371        # Vella, Marsh Forager (Stage 3.1)
```

- [ ] **Step 6: Boot test.**

Run: `go run .` and watch for clean load. Watch `mobs.LoadDataFiles() loadedCount` increase by 1.

- [ ] **Step 7: Commit.**

```bash
git add _datafiles/world/dogmud/mobs/stillwater_marsh/371-vella.yaml _datafiles/world/dogmud/behaviors/stillwater_marsh/371-vella.yaml _datafiles/world/dogmud/dialogue/stillwater_marsh/371-vella.yaml _datafiles/world/dogmud/rooms/stillwater/4123.yaml
git commit -m "$(cat <<'EOF'
feat(content): Marsh Forager Vella (mob 371)

Stillwater Marsh forager. Statpool 150, player-attack-immune, anchored
to Stillwater Temple Interior (4123). Wields gaff hook (10033). Light
dialogue + forager_step btree.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 14: Create Fernway Forager (Kessa) + camp room

**Files:**
- Create: `_datafiles/world/dogmud/rooms/the_fernway_south/4197.yaml` — Forager's Camp room
- Modify: `_datafiles/world/dogmud/rooms/the_fernway_south/4163.yaml` — add `north` exit to camp
- Create: `_datafiles/world/dogmud/mobs/the_fernway_south/366-kessa.yaml`
- Create: `_datafiles/world/dogmud/behaviors/the_fernway_south/366-kessa.yaml`
- Create: `_datafiles/world/dogmud/dialogue/the_fernway_south/366-kessa.yaml`

- [ ] **Step 1: Verify room 4163 has no existing `north` exit.**

Run: read `_datafiles/world/dogmud/rooms/the_fernway_south/4163.yaml`. If `north:` already exists in the `exits:` block, abort and pick a different parent room (4172 or 4172 are alternatives — check both). Otherwise proceed.

- [ ] **Step 2: Create the camp room.**

Write `_datafiles/world/dogmud/rooms/the_fernway_south/4197.yaml`:

```yaml
roomid: 4197
zone: The Fernway South
title: Forager's Camp
description: >
  A small clearing tucked off the main forest path, sheltered
  from the wind by a half-circle of old pines. A lean-to of
  bark and lashed branches stands at the back of the clearing,
  its roof shingled with strips of cedar. Inside, a banked
  firepit sends up a thin curl of woodsmoke. Drying racks of
  hazel rods hold stretched hides and bundles of herbs in
  neat rows. The clearing has the well-tended quietness of a
  place lived in by one careful hand. The air smells of pine
  pitch and curing leather.
biome: forest
coord:
  x: -14
  y: -19
  z: 0
exits:
  south:
    roomid: 4163
nouns:
  lean-to: |
    A practical shelter of debarked pine and woven cedar,
    open to the south, with a rolled bedding pallet inside
    and a row of pegged tools along the back wall — a
    skinning knife, a smaller scaling knife, a coil of
    leather cord, a spare pair of mocassins.
  firepit: |
    A round of fieldstones banked against the wind, its
    coals carefully managed. A tin kettle sits on a rotating
    iron arm, ready for a cup of tea between rounds.
  drying-racks: |
    Three drying racks of hazel rods stand in a tidy line,
    each carrying its own work — strips of rabbit hide on
    one, bundles of moonpetal and pine pitch on another, a
    half-rendered length of sinew on the third.
mutators:
- mutatorid: sanctuary
idlemessages:
- a wood pigeon coos from a high branch
- ''
- the kettle hisses softly on the firepit
- ''
- a strip of hide rocks gently in the breeze
spawninfo:
- mobid: 366        # Kessa, Fernway Forager (Stage 3.1)
```

- [ ] **Step 3: Add the north exit on room 4163.**

In `_datafiles/world/dogmud/rooms/the_fernway_south/4163.yaml`, in the `exits:` block, add:

```yaml
  north:
    roomid: 4197
```

- [ ] **Step 4: Write `mobs/the_fernway_south/366-kessa.yaml`.**

Mirror Vella's structure but with:
- name: Kessa
- statpool: 150
- fold_anchor_room: 4197 (camp)
- equipment.weapon.itemid: 10035 (hand axe)
- description: forest-themed (light frame, leather, hand axe + small bag)
- skills: same as Vella (slight emphasis on search 25, salvage 20)

- [ ] **Step 5: Write `behaviors/the_fernway_south/366-kessa.yaml`.**

Identical to Vella's btree structure (forager_step on idle).

- [ ] **Step 6: Write `dialogue/the_fernway_south/366-kessa.yaml`.**

Light dialogue: forest-themed flavor, hint at the caravan rendezvous, wary but friendly.

- [ ] **Step 7: Boot test.**

Run: `go run .`. Verify rooms.LoadDataFiles loadedCount +1, mobs.LoadDataFiles loadedCount +1.

Look for any "loadedCount=" line ending unexpectedly — that means a parse error.

- [ ] **Step 8: Commit.**

```bash
git add _datafiles/world/dogmud/rooms/the_fernway_south/{4163,4197}.yaml _datafiles/world/dogmud/mobs/the_fernway_south/366-kessa.yaml _datafiles/world/dogmud/behaviors/the_fernway_south/366-kessa.yaml _datafiles/world/dogmud/dialogue/the_fernway_south/366-kessa.yaml
git commit -m "$(cat <<'EOF'
feat(content): Fernway Forager Kessa + Forager's Camp room

New room 4197 (Forager's Camp) attached north of 4163, with sanctuary
mutator. Kessa (mob 366) anchored to the camp. Hands a satchel to the
caravan at North Road 4038 each cycle.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 15: Create Steppe Forager (Halix)

**Files:**
- Create: `_datafiles/world/dogmud/mobs/ironwind_steppe/243-halix.yaml`
- Create: `_datafiles/world/dogmud/behaviors/ironwind_steppe/243-halix.yaml`
- Create: `_datafiles/world/dogmud/dialogue/ironwind_steppe/243-halix.yaml`
- Modify: `_datafiles/world/dogmud/rooms/thornwall_city/468.yaml` — add Halix to spawninfo

- [ ] **Step 1: Write `mobs/ironwind_steppe/243-halix.yaml`.**

Mirror Vella but with:
- statpool: **225** (per spec — Ironwind is more dangerous)
- name: Halix
- description: weathered steppe-walker, dust on the boots, a coat of layered hides over linen
- fold_anchor_room: 468 (Thornwall Temple Interior)
- equipment.weapon.itemid: 10034 (hunting spear)

- [ ] **Step 2: Behavior tree — same forager_step pattern.**

- [ ] **Step 3: Dialogue — steppe-themed, dry humor, mentions ridge predators ("never go near the southern ridge after sundown").**

- [ ] **Step 4: Spawninfo at room 468.**

In `_datafiles/world/dogmud/rooms/thornwall_city/468.yaml`, append:

```yaml
- mobid: 243        # Halix, Steppe Forager (Stage 3.1)
```

- [ ] **Step 5: Boot test, commit.**

Run boot, then commit:

```bash
git add _datafiles/world/dogmud/mobs/ironwind_steppe/243-halix.yaml _datafiles/world/dogmud/behaviors/ironwind_steppe/243-halix.yaml _datafiles/world/dogmud/dialogue/ironwind_steppe/243-halix.yaml _datafiles/world/dogmud/rooms/thornwall_city/468.yaml
git commit -m "$(cat <<'EOF'
feat(content): Steppe Forager Halix (mob 243)

Ironwind Steppe forager. Statpool 225 (zone is more dangerous). Wields
hunting spear (10034). Anchored to Thornwall Temple Interior (468).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 16: Update temple-regen hint

**Files:**
- Modify: `_datafiles/world/dogmud/hints.yaml:107-110`

- [ ] **Step 1: Read the existing hint.**

The current line reads:

```yaml
  - >-
    Healing up slow? Try hanging out in the Sanctum Basin or
    the Temple in Thornwall. Some rooms are great places to
    rest.
```

- [ ] **Step 2: Replace with the generalized hint.**

Edit `_datafiles/world/dogmud/hints.yaml`, replace the above 4 lines with:

```yaml
  - >-
    Healing up slow? Some rooms are sanctuaries — temples, certain
    camps, the Sanctum Basin tutorial — and regenerate health,
    stamina, and conviction much faster than ordinary rooms. Look
    for a peaceful description.
```

- [ ] **Step 3: Boot test.**

Run: `go run .`. Hints load without parse error.

- [ ] **Step 4: Commit.**

```bash
git add _datafiles/world/dogmud/hints.yaml
git commit -m "$(cat <<'EOF'
docs(hints): generalize temple-regen hint for sanctuary mutator

Old hint named only Thornwall Temple + Sanctum Basin. New hint
references the sanctuary class so the Stillwater Temple, Fernway
Forager's Camp, and any future sanctuary room are all covered.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 17: Documentation + PATCH_NOTES

**Files:**
- Modify: `docs/economy/mat-audit-matrix.md` — add sync marker
- Modify: `docs/schemas/mob.md` — document fold_anchor_room is now used by foragers
- Modify: `docs/schemas/room.md` (or create `docs/schemas/mutator.md`) — document `regen_multiplier` field
- Modify: `docs/schemas/behavior.md` — document `forager_step` action + 3 new conditions
- Modify: `PATCH_NOTES.md` — Stage 3.1 dev-only entry

- [ ] **Step 1: Update mat-audit-matrix.md sync marker.**

At the top of `docs/economy/mat-audit-matrix.md`, add a line under the existing Purpose block:

> **Implementation note:** This matrix is mirrored in
> `internal/economy/buckets.go`. Drift between the two is caught at
> test time by `TestBucketMap_AuditMatrixCoverage`.

- [ ] **Step 2: Update docs/schemas/mutator.md (or room.md).**

If `mutator.md` exists, add a `regen_multiplier` field row. If not, append to `room.md` under a "Mutator fields" section:

```
- `regen_multiplier` (float, default 1.0) — multiplies HP/SP/CP regen
  for any actor in the room. Multiple mutators stack multiplicatively.
  Example: the `sanctuary` mutator sets this to 5.0.
```

- [ ] **Step 3: Update docs/schemas/mob.md.**

Locate the section describing fold-related fields. Add a note that `fold_anchor_room` is also used by Stage 3.1 foragers (alongside Stage 3.0d hermits and Stage 2 caravan crew).

- [ ] **Step 4: Update docs/schemas/behavior.md.**

Add entries for `forager_step` action and the 3 new conditions:

```
### Action: forager_step

Drives the per-forager state machine for Stage 3.1 forager NPCs.
Reads/writes MobState["forager_state"]. Dispatches per-state
sub-routines (resting, foraging, delivering, recalling, etc.).

### Condition: mob_can_safely_engage

True if the target is in the forager's prey whitelist AND the target's
effective stat-sum ≤ self stat-sum × 0.6 AND self HP ≥ 75%.

### Condition: mob_inventory_at_threshold

True if the mob's carried weight / carry-capacity ≥
ForagerCarryThresholdPct (default 75%).

### Condition: mob_hp_below_recall_threshold

True if mob HP / max-HP ≤ ForagerHPRecallThresholdPct (default 50%).
```

- [ ] **Step 5: Update PATCH_NOTES.md.**

Append (most recent at bottom, matching existing format):

```markdown
### 2026-04-29 — Stage 3.1: Forager NPCs (development branch)

- Added three forager NPCs: Vella (Stillwater Marsh, mob 371), Halix (Ironwind Steppe, mob 243), and Kessa (Fernway South, mob 366). Each gathers raw materials in their home territory, salvages corpses they encounter, and feeds the supply pipeline that 3.0b wired up.
- Vella delivers directly to Stillwater town vendors. Halix delivers directly to Thornwall town vendors. Kessa hands off to the caravan at North Road 4038 — caravan distributes Fernway mats to both towns symmetrically.
- Foragers are player-attack-immune (rebuff messaging like shopkeepers / caravan crew). They engage prey wildlife only and fold-recall when health drops to 50%, when carry capacity hits 75%, or when their fatigue timer expires.
- New room: Forager's Camp (4197) in The Fernway South — Kessa's anchor and a public sanctuary.
- Caravan cycle slowed: depot dwells 360 → 720 rounds. Cycle ≈ 2 game days. Caravan transit now stops briefly at North Road 4038 in both directions to meet Kessa.
- New room mutator `sanctuary` standardizes the previously-hardcoded high-regen rooms (Thornwall Temple Interior, Sanctum Basin tutorial). Stillwater Temple Interior + Forager's Camp now share the same 5x regen behavior.
- Bucket-aware shop restock (`RestockBuckets`) gates which slots a delivery fills. New `internal/economy` package mirrors the audit matrix.
- Three new low-tier 1H weapons: marsh gaff hook (10033), steppe hunting spear (10034), fernway hand axe (10035).
- Hint update: temple-regen hint generalized to reference the sanctuary class.
```

- [ ] **Step 6: Commit.**

```bash
git add docs/economy/mat-audit-matrix.md docs/schemas/ PATCH_NOTES.md
git commit -m "$(cat <<'EOF'
docs(3.1): forager / sanctuary / bucket schema docs + PATCH_NOTES

Documents:
- regen_multiplier mutator field
- fold_anchor_room expanded usage
- forager_step btree action + 3 new conditions
- mat-audit-matrix sync marker
- Stage 3.1 PATCH_NOTES entry (dev-only)

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Self-review checklist (run at end)

- [ ] **Spec coverage check.** Walk each section of the spec and confirm a task implements it:
  - Architecture overview (Section 3 of spec) — Tasks 4-10 + content tasks
  - Forager state machine (Section 4) — Tasks 7, 8
  - Combat behavior + targeting (Section 5) — Tasks 8, 13-15
  - Fold-recall integration — Tasks 13-15 (mob YAMLs)
  - Player-attack rebuff — Tasks 13-15 (`player_attack_immune: true`)
  - RestockBuckets — Task 5
  - Caravan route extension — Tasks 9, 10
  - Cadence retune — Task 3
  - Sanctuary mutator standardization — Tasks 1, 2, 11
  - Hint update — Task 16
  - Forager's Camp room — Task 14
- [ ] **No placeholders** (no "TBD", "TODO", "implement later"). All decisions in the "Decisions locked at plan time" section.
- [ ] **Type/method consistency:**
  - `RestockBuckets` signature: `func (si *ShopInventory) RestockBuckets(buckets []string) bool` — used identically in Task 5 + Task 9 + Task 8.
  - `ForagerProfile.Buckets []string` — used in Task 7 (definition) + Task 8 (consumer).
  - `MutatorSpec.RegenMultiplier float64` — defined Task 1, read Task 2.
  - Mob YAML field names: `player_attack_immune`, `non_combatant`, `fold_anchor_room`, `statpool` — match across Tasks 13, 14, 15.
- [ ] **Run after every task:** `go test ./...` and `go build ./...`

---

## Final verification (after all tasks)

1. `go test ./...` — full suite green
2. Boot the server locally and confirm clean load (Pre-Push SOP from CLAUDE.md): watch for `mobs.LoadDataFiles() loadedCount=`, `rooms.LoadDataFiles() loadedCount=`, `mutators.LoadDataFiles() loadedCount=` etc. without panic
3. In-game smoke test (12 steps from spec):
   1. Three foragers each at their sanctuary in `resting`
   2. Marsh forager wakes, walks to Stillwater Marsh, forages (visible flavor)
   3. Forager engages prey, salvages corpse
   4. Satchel fills; forager exits territory toward Stillwater town
   5. Forager visits Smith Brindle; verify Stillwater-bucket slots refilled
   6. After last vendor, fold-recall to Stillwater Temple Interior; verify sanctuary description + 5× regen
   7. Repeat for Steppe (Thornwall vendors) and Fernway (idle at 4038)
   8. Caravan-Fernway handoff: caravan stops at 4038 with forager present, acquires fernway flag, distributes at next town
   9. Caravan-Fernway miss: caravan passes through 4038 with no forager, no flag, Fernway slots stay empty
  10. `attack <forager>` → rebuff message for all three
  11. Drag a predator into a forager's room; HP < 50% triggers fold-recall to sanctuary
  12. Stillwater Temple Interior shows 5× regen for a player

---

## Execution handoff

**Plan complete.**

Two execution options:

1. **Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration.
2. **Inline Execution** — Execute tasks in this session using executing-plans, batch execution with checkpoints.

Which approach?
