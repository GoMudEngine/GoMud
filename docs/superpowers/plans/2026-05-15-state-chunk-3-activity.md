# Combat State — Chunk 3: Activity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Activity state machine (`Free | Casting | Crafting | Salvaging`) on the chunk-0 framework. Consolidate the two scattered `Character.CastingState` + `Character.CraftingState` pointer fields, formalize per-activity interrupt rules, normalize three mob/player asymmetries, and clean up the salvage-hijacks-crafting-slot pattern. Ship the `cancel_activity` btree action primitive so mob behavior trees can tactically abort activities (authoring of those trees stays out of scope — content/aliveness work).

**Architecture:** Same generics-based pattern as Life (chunk 2) and Awareness (chunk 1). Star-topology transition table (every active state goes through Free; no direct active-to-active). Each state owns a typed data struct (CastingData/CraftingData/SalvagingData) replacing the legacy pointer fields. Cross-machine cascades wire Activity-side observers to Life Dead, Combat Phase Engaging (for craft/salvage only — casting is itself a combat action), damage application (concentration roll for cast, hard cancel for craft/salvage), and movement (hard cancel for craft/salvage, both actors).

**Tech Stack:** Go 1.21+ with generics, existing `internal/state/` framework, existing `internal/state/life/`, `internal/state/combatphase/`, `internal/state/awareness/` machines.

**Spec:** `docs/superpowers/specs/2026-05-15-state-chunk-3-activity-design.md`

**Branch:** Stay on `feature/mob-aliveness-1.3-crimes` per the single-feature-branch SOP.

---

## File structure

| File | Status | Responsibility |
|------|--------|----------------|
| `internal/state/activity/activity.go` | NEW | State enum, AliveData / CastingData / CraftingData / SalvagingData, Machine wrapper |
| `internal/state/activity/transitions.go` | NEW | Valid-transition table, trigger constants |
| `internal/state/activity/rules.go` | NEW | Transition method implementations |
| `internal/state/activity/activity_test.go` | NEW | Behavior Matrix tests (AC-001 through AC-038) |
| `internal/state/activity/context.md` | NEW | Package documentation, including intentional-asymmetry rationale |
| `internal/characters/character.go` | MODIFY | Add `Activity *activity.Machine` field; init in `New()`; delete `CastingState` + `CraftingState` pointer fields (Task 11); update `IsCasting()` / `IsCrafting()` bodies; add `IsFree()` / `IsSalvaging()` / `IsActing()` predicates |
| `internal/characters/casting.go` | DELETE (Task 11) | `CastingState` struct moves to `activity.CastingData` |
| `internal/characters/crafting.go` | DELETE (Task 11) | `CraftingState` struct moves to `activity.CraftingData` |
| `internal/characters/validate.go` | MODIFY | Nil-guard init of `Activity` for YAML-loaded characters |
| `internal/hooks/Activity_Cascades.go` | NEW | Cross-machine cascade (Life Dead → Free, Combat Phase Engaging → Free for craft/salvage, damage → cancel per policy, movement → cancel for craft/salvage) |
| `internal/hooks/Life_Cascades.go` | MODIFY | Remove the Activity pre-wire — Activity observer now subscribes to Life Dead directly |
| `internal/usercommands/skill.cast.go` | MODIFY | Use `Activity.TransitionToCasting` + `CastingData`; replace `c.IsCrafting()` early-exit with `c.IsActing()` |
| `internal/usercommands/craft.go` | MODIFY | Use `Activity.TransitionToCrafting` + `CraftingData` |
| `internal/usercommands/salvage.go` | MODIFY | Use `Activity.TransitionToSalvaging` + `SalvagingData`; remove `CraftingState.RecipeId = "salvage:<itemid>"` hijack and `MiscData["salvage_item_uuid"]` storage |
| `internal/usercommands/cancel.go` | MODIFY | Generalize: handle any non-Free activity, dispatch on current state for refund logic |
| `internal/usercommands/go.go` | MODIFY | Replace direct `c.CraftingState = nil` with `Activity.TransitionToFree(TriggerMovementInterrupt)` |
| `internal/mobcommands/cancel.go` | NEW | Mob parity for `cancel` (mirrors player) |
| `internal/mobcommands/cast.go` | MODIFY | Use `Activity.TransitionToCasting` |
| `internal/mobcommands/craft.go` | MODIFY | Use `Activity.TransitionToCrafting` |
| `internal/mobcommands/salvage.go` | MODIFY | Use `Activity.TransitionToSalvaging` with a 1-round SalvagingData for data-shape parity; keeps single-tick resolution |
| `internal/behaviortree/actions.go` | MODIFY | Add `cancel_activity` action primitive |
| `internal/hooks/combat_shared_helpers.go` | MODIFY | `checkConcentrationBreak` rewires to call `Activity.TransitionToFree(TriggerConcentrationBreak)` on roll failure |
| `internal/hooks/NewRound_DoCombat_unified.go` (or wherever damage applies) | MODIFY | Damage application path extended to fire `Activity.TransitionToFree(TriggerDamageInterrupt)` for Crafting + Salvaging (hard cancel, no roll) |
| `internal/hooks/NewRound_UserRoundTick.go` | MODIFY | `resolveSalvage` and craft round-tick consume `SalvagingData` / `CraftingData` directly via Activity machine |
| `internal/hooks/NewRound_MobRoundTick.go` | MODIFY | Same as user round-tick; remove the mob-only combat-cancel block (lines ~404) — cascade handles it |
| `internal/actions/command_readiness.go` | MODIFY | Special-moves gate switches from `IsCrafting()` to `IsActing()` |
| `internal/characters/context.md` | MODIFY | Document `Activity` field + predicates |
| `internal/hooks/context.md` | MODIFY | Document Activity cascade observers + interrupt policy wiring |
| `internal/forager/context.md` | MODIFY (if exists) | Document why forager FSM stays separate from Activity |
| `tools/testing/audits/2026-05-15-chunk-3-doc-helpfile-audit.md` | NEW | Doc audit deliverable (Task 12) |
| `COMBAT_STATE_ROADMAP.md` | MODIFY | Mark chunk 3 Done |

---

## Task 1: Activity types + Character field

**Files:**
- Create: `internal/state/activity/activity.go`
- Create: `internal/state/activity/transitions.go`
- Modify: `internal/characters/character.go` (add field, init in `New()`)
- Modify: `internal/characters/validate.go` (init for YAML-loaded chars)

Foundation. State enum, per-state data types (CastingData / CraftingData / SalvagingData mirroring the existing `CastingState` / `CraftingState` field shapes), transition table, Machine wrapper, Character field bootstrap.

- [ ] **Step 1: Create `internal/state/activity/activity.go`**

```go
// Package activity defines the Activity state machine — the fourth
// consumer of internal/state, after combatphase, awareness, life.
// It replaces Character.CastingState and Character.CraftingState
// pointer fields with a single state machine carrying per-state
// data, and cleans up the salvage-hijacks-crafting-slot pattern.
package activity

import (
	"sync"

	"github.com/GoMudEngine/GoMud/internal/state"
)

// State is the Activity state enum.
type State int

const (
	Free State = iota
	Casting
	Crafting
	Salvaging
)

// String for logging / debugging.
func (s State) String() string {
	switch s {
	case Free:
		return "Free"
	case Casting:
		return "Casting"
	case Crafting:
		return "Crafting"
	case Salvaging:
		return "Salvaging"
	}
	return "Unknown"
}

// FreeData is empty — the default state.
type FreeData struct{}

// CastingData replaces internal/characters/CastingState. Field
// shape preserved verbatim so existing consumers (per-tick spell
// resolution, prompt rendering) only need to swap accessor.
type CastingData struct {
	Reason               state.TransitionReason
	SpellId              string
	FoldsNeeded          int
	FoldsAccumulated     int
	FoldsPerRound        int
	TotalConvictionCost  int
	ConvictionSpent      int
	TargetUserIds        []int
	TargetMobInstanceIds []int
	SpellRest            string
}

// CraftingData replaces internal/characters/CraftingState.
type CraftingData struct {
	Reason         state.TransitionReason
	RecipeId       string
	RoundsTotal    int
	RoundsComplete int
	TargetSlot     string
}

// SalvagingData is new — gives salvage its own state with its
// own data instead of hijacking CraftingState + MiscData.
type SalvagingData struct {
	Reason         state.TransitionReason
	ItemUuid       string
	RoundsTotal    int
	RoundsComplete int
	SpoiledPotion  bool
}

// Machine wraps state.Machine[State] with Activity-specific API.
type Machine struct {
	inner     *state.Machine[State]
	casting   *CastingData
	crafting  *CraftingData
	salvaging *SalvagingData
	self      state.ActorRef
}

// NewMachine returns an Activity machine in Free.
func NewMachine() *Machine {
	return &Machine{
		inner: state.NewMachine(Free, validTransitions),
	}
}

// State returns the current state.
func (m *Machine) State() State { return m.inner.State() }

// IsFree returns true when state is Free.
func (m *Machine) IsFree() bool { return m.State() == Free }

// IsCasting returns true when state is Casting.
func (m *Machine) IsCasting() bool { return m.State() == Casting }

// IsCrafting returns true when state is Crafting.
func (m *Machine) IsCrafting() bool { return m.State() == Crafting }

// IsSalvaging returns true when state is Salvaging.
func (m *Machine) IsSalvaging() bool { return m.State() == Salvaging }

// IsActing returns true for any non-Free state.
func (m *Machine) IsActing() bool { return m.State() != Free }

// CastingData returns the casting context if currently Casting.
func (m *Machine) CastingData() (CastingData, bool) {
	if m.State() != Casting || m.casting == nil {
		return CastingData{}, false
	}
	return *m.casting, true
}

// CraftingData returns the crafting context if currently Crafting.
func (m *Machine) CraftingData() (CraftingData, bool) {
	if m.State() != Crafting || m.crafting == nil {
		return CraftingData{}, false
	}
	return *m.crafting, true
}

// SalvagingData returns the salvaging context if currently Salvaging.
func (m *Machine) SalvagingData() (SalvagingData, bool) {
	if m.State() != Salvaging || m.salvaging == nil {
		return SalvagingData{}, false
	}
	return *m.salvaging, true
}

// Inner returns the underlying state.Machine. Used by rules.go
// (Task 3) and hooks (Task 5+). Not part of the stable API.
func (m *Machine) Inner() *state.Machine[State] { return m.inner }

// SetSelf binds the machine to its owning ActorRef.
func (m *Machine) SetSelf(ref state.ActorRef) { m.self = ref }

// Self returns the bound ActorRef.
func (m *Machine) Self() state.ActorRef { return m.self }

// === Machine registry ===
// Cross-character lookups (parity with combatphase/awareness/life).

var (
	registryMu      sync.Mutex
	machineRegistry = map[state.ActorRef]*Machine{}
)

// RegisterMachine binds an ActorRef to its Machine.
func RegisterMachine(ref state.ActorRef, m *Machine) {
	registryMu.Lock()
	defer registryMu.Unlock()
	machineRegistry[ref] = m
	m.SetSelf(ref)
}

// UnregisterMachine removes a binding.
func UnregisterMachine(ref state.ActorRef) {
	registryMu.Lock()
	defer registryMu.Unlock()
	delete(machineRegistry, ref)
}
```

- [ ] **Step 2: Create `internal/state/activity/transitions.go`**

```go
package activity

import "github.com/GoMudEngine/GoMud/internal/state"

// validTransitions is the star-topology Activity transition table.
// Every active state can return to Free; Free can become any active
// state. No direct active-to-active transitions (cancel-then-start
// enforces serialization, and cross-activity-start veto is enforced
// by call-site IsFree() checks).
var validTransitions = state.TransitionTable[State]{
	Free:      {Casting, Crafting, Salvaging},
	Casting:   {Free},
	Crafting:  {Free},
	Salvaging: {Free},
}

// Trigger reason constants.
const (
	// Free → active
	TriggerCastBegin    = "cast_begin"
	TriggerCraftBegin   = "craft_begin"
	TriggerSalvageBegin = "salvage_begin"

	// active → Free, success
	TriggerCastComplete    = "cast_complete"
	TriggerCraftComplete   = "craft_complete"
	TriggerSalvageComplete = "salvage_complete"

	// active → Free, user-initiated
	TriggerCastCancel    = "cast_cancel"
	TriggerCraftCancel   = "craft_cancel"
	TriggerSalvageCancel = "salvage_cancel"

	// active → Free, externally induced
	TriggerConcentrationBreak = "concentration_break" // Casting only
	TriggerCombatInterrupt    = "combat_interrupt"    // Crafting / Salvaging
	TriggerMovementInterrupt  = "movement_interrupt"  // Crafting / Salvaging
	TriggerDamageInterrupt    = "damage_interrupt"    // Crafting / Salvaging (hard cancel, no roll)
	TriggerDeath              = "death"               // cascade from Life
)
```

- [ ] **Step 3: Verify the new package builds**

Run:
```bash
go build ./internal/state/activity/
```
Expected: clean (no rules.go yet — transitions referenced by validTransitions only).

- [ ] **Step 4: Modify `internal/characters/character.go` — add the Activity field**

Find the existing state-machine fields (look for `Life *life.Machine` from chunk 2, alongside `CombatPhase *combatphase.Machine` and `Awareness *awareness.Machine`). Add `Activity *activity.Machine` next to them. Update imports.

```go
import (
    // ... existing imports
    "github.com/GoMudEngine/GoMud/internal/state/activity"
)

// Inside Character struct, with the other machines:
Activity *activity.Machine `yaml:"-"`
```

Then find `func New()` (the Character constructor) and initialize the Activity machine alongside the others:

```go
func New() *Character {
    c := &Character{
        // ... existing initializations
        CombatPhase: combatphase.NewMachine(),
        Awareness:   awareness.NewMachine(),
        Life:        life.NewMachine(),
        Activity:    activity.NewMachine(),
        // ... rest
    }
    // ... existing post-init
    return c
}
```

Do NOT delete `CastingState` or `CraftingState` here — that's Task 11. Both fields coexist with the new machine through Tasks 6-10 (migration window).

- [ ] **Step 5: Modify `internal/characters/validate.go` — nil-guard for YAML-loaded characters**

Find the `Validate()` method (look for where chunk 2 added `if c.Life == nil { c.Life = life.NewMachine() }`). Add the same pattern for Activity:

```go
if c.Activity == nil {
    c.Activity = activity.NewMachine()
}
```

Update imports.

- [ ] **Step 6: Build verify**

Run:
```bash
go build ./...
```
Expected: clean. Any compile errors here are about the new field/import — fix them.

- [ ] **Step 7: Run the existing test suite to confirm no regressions**

Run:
```bash
go test ./internal/state/... ./internal/characters/ -count=1 2>&1 | grep -E "^(ok|FAIL)"
```
Expected: all PASS (no behavior change yet, just new field initialized to Free).

- [ ] **Step 8: Commit**

```bash
git add internal/state/activity/activity.go \
        internal/state/activity/transitions.go \
        internal/characters/character.go \
        internal/characters/validate.go
git commit -m "$(cat <<'EOF'
feat(activity): state types, transition table, Character field

Chunk 3 foundation. State enum (Free/Casting/Crafting/Salvaging),
per-state data structs (CastingData/CraftingData/SalvagingData)
mirroring legacy CastingState/CraftingState field shapes, star-
topology valid-transitions table, machine wrapper with predicates
and data accessors, trigger constants for every transition kind.

Character.Activity field added (initialized in New + nil-guarded
in Validate). Coexists with legacy CastingState/CraftingState
fields through the migration window; legacy fields deleted in
Task 11.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Behavior Matrix RED tests (AC-001 through AC-038)

**Files:**
- Create: `internal/state/activity/activity_test.go`

All ~38 Behavior Matrix tests authored as failing skeletons. Each test name maps to a row from the spec's matrix preview. The TransitionTo* methods don't exist yet (Task 3 adds them), so the tests will all fail with `m.TransitionToCasting undefined` until Task 3 lands.

- [ ] **Step 1: Create `internal/state/activity/activity_test.go` with all rows**

Use this template; each row gets a `t.Run` block. Tests fail because the Transition methods don't exist yet — that's the RED phase.

```go
package activity_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
)

// --- AC-001 through AC-012: Basic transitions ---

func TestAC_001_FreeToCasting(t *testing.T) {
	m := activity.NewMachine()
	err := m.TransitionToCasting(
		activity.CastingData{SpellId: "fireball", FoldsNeeded: 3},
		state.TransitionReason{Trigger: activity.TriggerCastBegin},
	)
	if err != nil {
		t.Fatalf("expected transition to succeed, got %v", err)
	}
	if m.State() != activity.Casting {
		t.Errorf("expected Casting, got %v", m.State())
	}
}

func TestAC_002_FreeToCrafting(t *testing.T) {
	m := activity.NewMachine()
	err := m.TransitionToCrafting(
		activity.CraftingData{RecipeId: "iron_dagger", RoundsTotal: 4},
		state.TransitionReason{Trigger: activity.TriggerCraftBegin},
	)
	if err != nil {
		t.Fatalf("expected transition to succeed, got %v", err)
	}
	if m.State() != activity.Crafting {
		t.Errorf("expected Crafting, got %v", m.State())
	}
}

func TestAC_003_FreeToSalvaging(t *testing.T) {
	m := activity.NewMachine()
	err := m.TransitionToSalvaging(
		activity.SalvagingData{ItemUuid: "abc-123", RoundsTotal: 3},
		state.TransitionReason{Trigger: activity.TriggerSalvageBegin},
	)
	if err != nil {
		t.Fatalf("expected transition to succeed, got %v", err)
	}
	if m.State() != activity.Salvaging {
		t.Errorf("expected Salvaging, got %v", m.State())
	}
}

func TestAC_004_CastingToFreeOnComplete(t *testing.T) {
	m := activity.NewMachine()
	_ = m.TransitionToCasting(activity.CastingData{SpellId: "fireball"},
		state.TransitionReason{Trigger: activity.TriggerCastBegin})
	err := m.TransitionToFree(state.TransitionReason{Trigger: activity.TriggerCastComplete})
	if err != nil {
		t.Fatalf("expected transition to succeed, got %v", err)
	}
	if m.State() != activity.Free {
		t.Errorf("expected Free, got %v", m.State())
	}
}

func TestAC_005_CraftingToFreeOnComplete(t *testing.T) {
	m := activity.NewMachine()
	_ = m.TransitionToCrafting(activity.CraftingData{RecipeId: "iron_dagger"},
		state.TransitionReason{Trigger: activity.TriggerCraftBegin})
	err := m.TransitionToFree(state.TransitionReason{Trigger: activity.TriggerCraftComplete})
	if err != nil {
		t.Fatalf("expected transition to succeed, got %v", err)
	}
	if m.State() != activity.Free {
		t.Errorf("expected Free, got %v", m.State())
	}
}

func TestAC_006_SalvagingToFreeOnComplete(t *testing.T) {
	m := activity.NewMachine()
	_ = m.TransitionToSalvaging(activity.SalvagingData{ItemUuid: "abc-123"},
		state.TransitionReason{Trigger: activity.TriggerSalvageBegin})
	err := m.TransitionToFree(state.TransitionReason{Trigger: activity.TriggerSalvageComplete})
	if err != nil {
		t.Fatalf("expected transition to succeed, got %v", err)
	}
	if m.State() != activity.Free {
		t.Errorf("expected Free, got %v", m.State())
	}
}

func TestAC_007_CastingToFreeOnCancel(t *testing.T) {
	m := activity.NewMachine()
	_ = m.TransitionToCasting(activity.CastingData{SpellId: "fireball"},
		state.TransitionReason{Trigger: activity.TriggerCastBegin})
	err := m.TransitionToFree(state.TransitionReason{Trigger: activity.TriggerCastCancel})
	if err != nil {
		t.Fatalf("cancel should succeed, got %v", err)
	}
	if m.State() != activity.Free {
		t.Errorf("expected Free after cancel, got %v", m.State())
	}
}

func TestAC_008_CraftingToFreeOnCancel(t *testing.T) {
	m := activity.NewMachine()
	_ = m.TransitionToCrafting(activity.CraftingData{RecipeId: "iron_dagger"},
		state.TransitionReason{Trigger: activity.TriggerCraftBegin})
	err := m.TransitionToFree(state.TransitionReason{Trigger: activity.TriggerCraftCancel})
	if err != nil {
		t.Fatalf("cancel should succeed, got %v", err)
	}
	if m.State() != activity.Free {
		t.Errorf("expected Free after cancel, got %v", m.State())
	}
}

func TestAC_009_SalvagingToFreeOnCancel(t *testing.T) {
	m := activity.NewMachine()
	_ = m.TransitionToSalvaging(activity.SalvagingData{ItemUuid: "abc-123"},
		state.TransitionReason{Trigger: activity.TriggerSalvageBegin})
	err := m.TransitionToFree(state.TransitionReason{Trigger: activity.TriggerSalvageCancel})
	if err != nil {
		t.Fatalf("cancel should succeed, got %v", err)
	}
	if m.State() != activity.Free {
		t.Errorf("expected Free after cancel, got %v", m.State())
	}
}

func TestAC_010_StartsInFree(t *testing.T) {
	m := activity.NewMachine()
	if m.State() != activity.Free {
		t.Errorf("expected Free, got %v", m.State())
	}
	if !m.IsFree() {
		t.Errorf("IsFree() = false, want true")
	}
	if m.IsActing() {
		t.Errorf("IsActing() = true, want false")
	}
}

func TestAC_011_CastingDataAccessible(t *testing.T) {
	m := activity.NewMachine()
	_ = m.TransitionToCasting(
		activity.CastingData{SpellId: "fireball", FoldsNeeded: 5, ConvictionSpent: 12},
		state.TransitionReason{Trigger: activity.TriggerCastBegin},
	)
	d, ok := m.CastingData()
	if !ok {
		t.Fatal("expected casting data to be available")
	}
	if d.SpellId != "fireball" || d.FoldsNeeded != 5 || d.ConvictionSpent != 12 {
		t.Errorf("data lost in transition: %+v", d)
	}
}

func TestAC_012_DataClearedOnReturnToFree(t *testing.T) {
	m := activity.NewMachine()
	_ = m.TransitionToCasting(activity.CastingData{SpellId: "fireball"},
		state.TransitionReason{Trigger: activity.TriggerCastBegin})
	_ = m.TransitionToFree(state.TransitionReason{Trigger: activity.TriggerCastComplete})
	if _, ok := m.CastingData(); ok {
		t.Errorf("CastingData() should return ok=false when state is Free")
	}
}

// --- AC-013 through AC-024: Per-activity interrupt policy ---

func TestAC_013_CastingConcentrationBreakOnDamage(t *testing.T) {
	t.Skip("integration test — verified in Task 5 cascade tests")
}

func TestAC_014_CraftingHardCancelOnDamage(t *testing.T) {
	t.Skip("integration test — verified in Task 5 cascade tests")
}

func TestAC_015_SalvagingHardCancelOnDamage(t *testing.T) {
	t.Skip("integration test — verified in Task 5 cascade tests")
}

func TestAC_016_CraftingCancelOnCombatEntry(t *testing.T) {
	t.Skip("integration test — verified in Task 5 cascade tests")
}

func TestAC_017_SalvagingCancelOnCombatEntry(t *testing.T) {
	t.Skip("integration test — verified in Task 5 cascade tests")
}

func TestAC_018_CastingNotInterruptedByCombatEntry(t *testing.T) {
	t.Skip("integration test — verified in Task 5 cascade tests")
}

func TestAC_019_CraftingCancelOnMovement(t *testing.T) {
	t.Skip("integration test — verified in Task 5 cascade tests")
}

func TestAC_020_SalvagingCancelOnMovement(t *testing.T) {
	t.Skip("integration test — verified in Task 5 cascade tests")
}

func TestAC_021_CastingNotCanceledByMovement(t *testing.T) {
	t.Skip("integration test — verified in Task 5 cascade tests")
}

func TestAC_022_ConvictionRefundOnCastCancel(t *testing.T) {
	t.Skip("integration test — verified in Task 9 cancel command tests")
}

func TestAC_023_NoRefundOnCraftCancel(t *testing.T) {
	t.Skip("integration test — verified in Task 9 cancel command tests")
}

func TestAC_024_NoRefundOnSalvageCancel(t *testing.T) {
	t.Skip("integration test — verified in Task 9 cancel command tests")
}

// --- AC-025 through AC-028: Cross-activity start veto ---

func TestAC_025_CastingToCastingFails(t *testing.T) {
	m := activity.NewMachine()
	_ = m.TransitionToCasting(activity.CastingData{SpellId: "fireball"},
		state.TransitionReason{Trigger: activity.TriggerCastBegin})
	err := m.TransitionToCasting(activity.CastingData{SpellId: "icebolt"},
		state.TransitionReason{Trigger: activity.TriggerCastBegin})
	if err == nil {
		t.Fatal("expected error transitioning Casting→Casting")
	}
	if m.State() != activity.Casting {
		t.Errorf("state should remain Casting on failed transition")
	}
}

func TestAC_026_CraftingToCastingFails(t *testing.T) {
	m := activity.NewMachine()
	_ = m.TransitionToCrafting(activity.CraftingData{RecipeId: "iron_dagger"},
		state.TransitionReason{Trigger: activity.TriggerCraftBegin})
	err := m.TransitionToCasting(activity.CastingData{SpellId: "fireball"},
		state.TransitionReason{Trigger: activity.TriggerCastBegin})
	if err == nil {
		t.Fatal("expected error transitioning Crafting→Casting directly")
	}
}

func TestAC_027_CastingToCraftingFails(t *testing.T) {
	m := activity.NewMachine()
	_ = m.TransitionToCasting(activity.CastingData{SpellId: "fireball"},
		state.TransitionReason{Trigger: activity.TriggerCastBegin})
	err := m.TransitionToCrafting(activity.CraftingData{RecipeId: "iron_dagger"},
		state.TransitionReason{Trigger: activity.TriggerCraftBegin})
	if err == nil {
		t.Fatal("expected error transitioning Casting→Crafting directly")
	}
}

func TestAC_028_SalvagingToCraftingFails(t *testing.T) {
	m := activity.NewMachine()
	_ = m.TransitionToSalvaging(activity.SalvagingData{ItemUuid: "abc-123"},
		state.TransitionReason{Trigger: activity.TriggerSalvageBegin})
	err := m.TransitionToCrafting(activity.CraftingData{RecipeId: "iron_dagger"},
		state.TransitionReason{Trigger: activity.TriggerCraftBegin})
	if err == nil {
		t.Fatal("expected error transitioning Salvaging→Crafting directly")
	}
}

// --- AC-029 through AC-034: Mob/player parity ---

func TestAC_029_MobCraftCancelsOnCombatEntry(t *testing.T) {
	t.Skip("integration test — verified in Task 5/7 cascade + migration tests")
}

func TestAC_030_PlayerCraftCancelsOnDamage(t *testing.T) {
	t.Skip("integration test — verified in Task 5/7 cascade + migration tests")
}

func TestAC_031_MobCraftCancelsOnDamage(t *testing.T) {
	t.Skip("integration test — verified in Task 5/7 cascade + migration tests")
}

func TestAC_032_PlayerCastConcentrationBreakSurvivable(t *testing.T) {
	t.Skip("integration test — verified in Task 5/6 cascade + migration tests")
}

func TestAC_033_MobCastConcentrationBreakSurvivable(t *testing.T) {
	t.Skip("integration test — verified in Task 5/6 cascade + migration tests")
}

func TestAC_034_BtreeCancelActivityTriggersFree(t *testing.T) {
	t.Skip("integration test — verified in Task 9 btree primitive tests")
}

// --- AC-035 through AC-038: Cascade verification ---

func TestAC_035_LifeDeadCascadesActivityFree(t *testing.T) {
	t.Skip("integration test — verified in Task 5 cascade tests")
}

func TestAC_036_MovementCascadesCraftingFree(t *testing.T) {
	t.Skip("integration test — verified in Task 5 cascade tests")
}

func TestAC_037_MovementCascadesSalvagingFree(t *testing.T) {
	t.Skip("integration test — verified in Task 5 cascade tests")
}

func TestAC_038_CombatPhaseEngagingCascadesCraftingFree(t *testing.T) {
	t.Skip("integration test — verified in Task 5 cascade tests")
}
```

The integration-test rows are skipped at the unit-test layer because they require multiple machines wired together via hook observers. They become live in Task 5+ when cascades land.

- [ ] **Step 2: Run tests — expect AC-001 through AC-012, AC-025-028 to FAIL (transition methods don't exist), skip-rows to SKIP**

```bash
go test ./internal/state/activity/ -v -count=1 2>&1 | grep -E "^--- (PASS|FAIL|SKIP)" | head -40
```

Expected: 16 FAIL (the basic-transitions + veto rows), 22 SKIP (integration rows). 0 PASS.

If you get `undefined: m.TransitionToCasting` style errors at compile, that's also fine — rules.go doesn't exist yet. Task 3 fixes both.

- [ ] **Step 3: Commit RED phase**

```bash
git add internal/state/activity/activity_test.go
git commit -m "$(cat <<'EOF'
test(activity): Behavior Matrix RED — AC-001 through AC-038

All 38 Behavior Matrix rows authored as failing skeletons. 16
unit-level transitions + veto rows fail (TransitionTo methods
don't exist yet — Task 3 implements). 22 integration rows are
SKIP at the unit layer — they require cross-machine wiring
(Tasks 5+) and live in cascade / migration tests.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Basic transitions (AC-001 through AC-012, AC-025-028)

**Files:**
- Create: `internal/state/activity/rules.go`

The four `TransitionTo*` methods. After this, the unit-level Behavior Matrix rows (16 of them) go GREEN. Pattern mirrors chunk-2's `internal/state/life/rules.go` — set the per-state data BEFORE calling `m.inner.TransitionTo` (because AfterTransition observers fire during that call), with rollback on error.

- [ ] **Step 1: Create `internal/state/activity/rules.go`**

```go
package activity

import "github.com/GoMudEngine/GoMud/internal/state"

// TransitionToCasting moves Free → Casting and stores the casting
// context. Caller is responsible for any pre-checks (e.g., "is the
// character free", "does the character have enough conviction").
func (m *Machine) TransitionToCasting(d CastingData, r state.TransitionReason) error {
	d.Reason = r
	prev := m.casting
	m.casting = &d
	if err := m.inner.TransitionTo(Casting, r); err != nil {
		m.casting = prev
		return err
	}
	return nil
}

// TransitionToCrafting moves Free → Crafting and stores the
// crafting context.
func (m *Machine) TransitionToCrafting(d CraftingData, r state.TransitionReason) error {
	d.Reason = r
	prev := m.crafting
	m.crafting = &d
	if err := m.inner.TransitionTo(Crafting, r); err != nil {
		m.crafting = prev
		return err
	}
	return nil
}

// TransitionToSalvaging moves Free → Salvaging and stores the
// salvaging context.
func (m *Machine) TransitionToSalvaging(d SalvagingData, r state.TransitionReason) error {
	d.Reason = r
	prev := m.salvaging
	m.salvaging = &d
	if err := m.inner.TransitionTo(Salvaging, r); err != nil {
		m.salvaging = prev
		return err
	}
	return nil
}

// TransitionToFree returns the machine to Free, clearing all
// per-state data. All cancel / complete / interrupt paths route
// through here.
func (m *Machine) TransitionToFree(r state.TransitionReason) error {
	if err := m.inner.TransitionTo(Free, r); err != nil {
		return err
	}
	m.casting = nil
	m.crafting = nil
	m.salvaging = nil
	return nil
}

// ForceFree transitions to Free from any state. Used by admin
// commands and emergency cleanup. Idempotent if already Free.
func (m *Machine) ForceFree(r state.TransitionReason) {
	if m.State() == Free {
		return
	}
	_ = m.inner.TransitionTo(Free, r)
	m.casting = nil
	m.crafting = nil
	m.salvaging = nil
}
```

- [ ] **Step 2: Run the unit-level tests to verify they pass**

```bash
go test ./internal/state/activity/ -v -count=1 -run 'TestAC_00[1-9]|TestAC_01[0-2]|TestAC_02[5-8]' 2>&1 | grep -E "^--- (PASS|FAIL)" | head -20
```

Expected: 16 PASS (AC-001 through AC-012, AC-025-028).

- [ ] **Step 3: Full test run for the package**

```bash
go test ./internal/state/activity/ -count=1 2>&1 | grep -E "^(ok|FAIL)"
```

Expected: `ok` for the package.

- [ ] **Step 4: Commit**

```bash
git add internal/state/activity/rules.go
git commit -m "$(cat <<'EOF'
feat(activity): basic transitions AC-001 through AC-012, AC-025-028

TransitionToCasting/Crafting/Salvaging/Free + ForceFree. Per-state
data stored BEFORE inner.TransitionTo (so AfterTransition observers
see populated data), with rollback on error — same pattern as
Life machine's TransitionToDead.

TransitionToFree clears all per-state data slots (since star
topology means it's the only exit). 16 unit-level Behavior Matrix
rows now GREEN; 22 integration rows still SKIP pending cascades.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Predicates on Character

**Files:**
- Modify: `internal/characters/character.go` (predicates)

Wire the Activity machine's predicates onto Character. `IsCasting()` and `IsCrafting()` already exist (currently `c.CastingState != nil`); their bodies get rewritten to query the machine. `IsSalvaging()`, `IsFree()`, `IsActing()` are new.

- [ ] **Step 1: Find and rewrite `IsCasting()` and `IsCrafting()`**

Find the existing predicates in `internal/characters/spells.go` (per the survey, lines 83 and 86 — verify with grep before editing):

```bash
grep -n "func (c \*Character) IsCasting\|func (c \*Character) IsCrafting" internal/characters/
```

Replace bodies:

```go
// IsCasting returns true when the character is mid-spell-cast.
// Queries the Activity state machine (was c.CastingState != nil).
func (c *Character) IsCasting() bool {
	if c.Activity == nil {
		// Defensive: a Character constructed outside New() won't
		// have the machine. Validate() handles YAML-loaded chars.
		return false
	}
	return c.Activity.IsCasting()
}

// IsCrafting returns true when the character is mid-craft.
// Queries the Activity state machine (was c.CraftingState != nil).
func (c *Character) IsCrafting() bool {
	if c.Activity == nil {
		return false
	}
	return c.Activity.IsCrafting()
}
```

The legacy fields `CastingState` / `CraftingState` still exist on Character at this point (Task 11 deletes them) — but they're no longer read by the predicates. Migration of writers happens in Tasks 6-8.

- [ ] **Step 2: Add the new predicates**

Append after `IsCrafting` (or in the same file — match codebase grouping):

```go
// IsSalvaging returns true when the character is mid-salvage.
// New in chunk 3 — replaces the implicit
// CraftingState.RecipeId starts-with "salvage:" check.
func (c *Character) IsSalvaging() bool {
	if c.Activity == nil {
		return false
	}
	return c.Activity.IsSalvaging()
}

// IsFree returns true when no Activity is locked in.
func (c *Character) IsFree() bool {
	if c.Activity == nil {
		return true
	}
	return c.Activity.IsFree()
}

// IsActing returns true for any non-Free Activity. Canonical
// "busy with a locked-in mechanic" gate — replaces the previous
// IsCrafting()-only check at the special-moves gate in
// actions/command_readiness.go (rewired in Task 10).
func (c *Character) IsActing() bool {
	if c.Activity == nil {
		return false
	}
	return c.Activity.IsActing()
}
```

- [ ] **Step 3: Build verify**

```bash
go build ./...
```
Expected: clean.

- [ ] **Step 4: Run tests**

```bash
go test ./internal/characters/ ./internal/state/activity/ -count=1 2>&1 | grep -E "^(ok|FAIL)"
```
Expected: `ok` for both. `IsCasting()` / `IsCrafting()` still return false in tests that didn't transition through the machine (since the machine starts Free) — that's correct; nothing else uses these predicates yet.

- [ ] **Step 5: Commit**

```bash
git add internal/characters/spells.go
git commit -m "$(cat <<'EOF'
feat(activity): IsFree/IsSalvaging/IsActing + rewire IsCasting/IsCrafting

IsCasting() and IsCrafting() now query Character.Activity instead
of reading the legacy CastingState/CraftingState pointer fields.
IsSalvaging(), IsFree(), IsActing() are new — IsActing() becomes
the canonical busy gate (rewired in Task 10).

Legacy fields still exist on Character through the migration
window; deleted in Task 11.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Cross-machine cascade — Life Dead, Combat Phase, damage, movement

**Files:**
- Create: `internal/hooks/Activity_Cascades.go`
- Modify: `internal/hooks/Life_Cascades.go` (remove Activity pre-wire)

Activity-side observers subscribe to other machines' transitions. After this, Activity reacts to Life Dead (cancels), Combat Phase Engaging on self (cancels craft/salvage only — casting is exempt per policy), movement (cancels craft/salvage), and damage (cancels craft/salvage hard; casting is handled separately by the rewired concentration-break path in Task 6).

- [ ] **Step 1: Create `internal/hooks/Activity_Cascades.go`**

```go
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
	"github.com/GoMudEngine/GoMud/internal/state/combatphase"
	"github.com/GoMudEngine/GoMud/internal/state/life"
)

// wireActivityCrossMachineCascades subscribes the Activity machine
// to the other machines that drive its interrupt rules:
//   - Life: Alive → Dead → Activity → Free (covers what the chunk-2
//     pre-wire in Life_Cascades.go used to do directly).
//   - Combat Phase: Idle → Engaging (self-initiated) → cancel only
//     if current Activity is Crafting or Salvaging. Casting is
//     exempt — casting IS a combat action.
//
// Movement cancel and damage cancel for Crafting/Salvaging are not
// machine-to-machine transitions and don't fit AfterTransition;
// they're wired at the call sites (go.go for movement in Task 7,
// damage application path in Task 10).
func wireActivityCrossMachineCascades(c *characters.Character) {
	// Life: Alive → Dead → cancel any active Activity.
	c.Life.Inner().AfterTransition("activity_life_dead",
		func(from, to life.State, r state.TransitionReason) {
			if from != life.Alive || to != life.Dead {
				return
			}
			if c.Activity == nil || c.Activity.IsFree() {
				return
			}
			_ = c.Activity.TransitionToFree(state.TransitionReason{
				Trigger: activity.TriggerDeath,
				Actor:   c.Activity.Self(),
			})
		})

	// Combat Phase: Idle → Engaging (self-initiated) → cancel
	// Crafting/Salvaging. Casting is exempt.
	c.CombatPhase.Inner().AfterTransition("activity_combat_entry",
		func(from, to combatphase.State, r state.TransitionReason) {
			if from != combatphase.Idle || to != combatphase.Engaging {
				return
			}
			if c.Activity == nil {
				return
			}
			switch c.Activity.State() {
			case activity.Crafting, activity.Salvaging:
				_ = c.Activity.TransitionToFree(state.TransitionReason{
					Trigger: activity.TriggerCombatInterrupt,
					Actor:   c.Activity.Self(),
				})
			}
		})
}

func init() {
	characters.OnCharacterCreated(wireActivityCrossMachineCascades)
}
```

- [ ] **Step 2: Modify `internal/hooks/Life_Cascades.go` — remove the Activity pre-wire**

Find the chunk-2 Life cascade body. Locate the lines:

```go
// 3. Activity → Free (chunk-3 pre-wire:
//    clear casting/crafting state directly).
c.CastingState = nil
c.CraftingState = nil
```

(Per the survey, these are around lines 43-45 of `Life_Cascades.go`.)

Replace with a comment noting that the Activity cascade now handles this:

```go
// 3. Activity → Free is now handled by Activity_Cascades.go
//    (chunk 3). The legacy CastingState/CraftingState pointer
//    nil-outs that lived here are gone — Activity.TransitionToFree
//    (fired by the activity_life_dead AfterTransition observer)
//    clears the per-state data slots inside the machine.
```

DO NOT touch the other lines in the Dead-cascade block (Combat Phase → Idle, Awareness → Visible, Position → Standing pre-wire, buff cancel, conditions clear). Those stay.

- [ ] **Step 3: Build verify**

```bash
go build ./...
```
Expected: clean. Compile errors here would be from missing imports or syntax in Activity_Cascades.go — fix them.

- [ ] **Step 4: Run state-machine + hooks tests**

```bash
go test ./internal/state/... ./internal/hooks/ ./internal/characters/ -count=1 2>&1 | grep -E "^(ok|FAIL)"
```
Expected: all PASS. Activity_Cascades observers fire on Life Dead and Combat Phase Engaging transitions — but neither is invoked from production code yet for Activity-meaningful scenarios, so behavior is unchanged.

- [ ] **Step 5: Write a quick integration test for the Life-cascade path (AC-035)**

Add a test in `internal/hooks/Activity_Cascades_test.go` (NEW):

```go
package hooks_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
	"github.com/GoMudEngine/GoMud/internal/state/life"
	_ "github.com/GoMudEngine/GoMud/internal/hooks" // wire init() observers
)

func TestActivityCascadeOnLifeDead(t *testing.T) {
	c := characters.New()
	_ = c.Activity.TransitionToCasting(
		activity.CastingData{SpellId: "fireball", FoldsAccumulated: 2},
		state.TransitionReason{Trigger: activity.TriggerCastBegin},
	)
	if !c.IsCasting() {
		t.Fatal("setup: expected character to be casting")
	}

	_ = c.Life.TransitionToDead(
		life.DeadData{},
		state.TransitionReason{Trigger: life.TriggerSuicide},
	)

	if c.IsCasting() {
		t.Errorf("Activity should have cascaded to Free on Life Dead")
	}
	if !c.IsFree() {
		t.Errorf("Activity state = %v, expected Free", c.Activity.State())
	}
}

func TestActivityCascadeOnCombatEntryForCrafting(t *testing.T) {
	c := characters.New()
	_ = c.Activity.TransitionToCrafting(
		activity.CraftingData{RecipeId: "iron_dagger", RoundsTotal: 4},
		state.TransitionReason{Trigger: activity.TriggerCraftBegin},
	)

	// Simulate Combat Phase Idle → Engaging.
	// The exact API depends on chunk-0 — TransitionToEngaging or similar.
	// If the framework requires a target, supply a dummy.
	_ = c.CombatPhase.TransitionToEngaging(state.ActorRef{UserId: 999},
		state.TransitionReason{Trigger: "test_attack"})

	if !c.IsFree() {
		t.Errorf("Activity should cascade to Free on Combat Phase Engaging; got %v", c.Activity.State())
	}
}

func TestActivityCastingNotCanceledByCombatEntry(t *testing.T) {
	c := characters.New()
	_ = c.Activity.TransitionToCasting(
		activity.CastingData{SpellId: "fireball"},
		state.TransitionReason{Trigger: activity.TriggerCastBegin},
	)

	_ = c.CombatPhase.TransitionToEngaging(state.ActorRef{UserId: 999},
		state.TransitionReason{Trigger: "test_attack"})

	if !c.IsCasting() {
		t.Errorf("Casting should NOT cascade Free on Combat Phase Engaging; got %v", c.Activity.State())
	}
}
```

Note: the exact `c.CombatPhase.TransitionToEngaging(...)` signature is whatever chunk 0 defined. If it differs, adjust the call to match. The Combat Phase machine's signature lives in `internal/state/combatphase/rules.go` — read it first if uncertain.

- [ ] **Step 6: Run the new tests**

```bash
go test ./internal/hooks/ -v -count=1 -run 'TestActivityCascade' 2>&1 | grep -E "^--- (PASS|FAIL)"
```
Expected: 3 PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/hooks/Activity_Cascades.go \
        internal/hooks/Life_Cascades.go \
        internal/hooks/Activity_Cascades_test.go
git commit -m "$(cat <<'EOF'
feat(activity): cross-machine cascade — Life Dead + Combat Phase Engaging

Activity_Cascades.go subscribes to:
  - Life Alive→Dead → Activity → Free (TriggerDeath)
  - Combat Phase Idle→Engaging (self) → Activity → Free if current
    state is Crafting or Salvaging (TriggerCombatInterrupt).
    Casting is exempt per the per-activity policy.

Life_Cascades.go: legacy chunk-2 Activity pre-wire (direct
CastingState=nil / CraftingState=nil) removed. The Activity-side
observer now owns the death cleanup.

Movement-cancel and damage-cancel for Crafting/Salvaging are wired
at call sites in Tasks 7/10 (they're not machine transitions, so
they don't fit AfterTransition).

Three integration tests added covering AC-035 (Life cascade) and
the Combat Phase exemption for Casting (AC-018) vs cancel for
Crafting (AC-038).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Casting migration

**Files:**
- Modify: `internal/usercommands/skill.cast.go`
- Modify: `internal/mobcommands/cast.go`
- Modify: `internal/hooks/combat_shared_helpers.go` (concentration break)
- Modify: `internal/hooks/NewRound_DoCombat_unified.go` (or wherever spell resolution reads CastingState)

Move all casting writers from `c.CastingState = ...` to `c.Activity.TransitionToCasting(CastingData{...}, ...)`. Move all readers from `c.CastingState` to `c.Activity.CastingData()`. Rewire `checkConcentrationBreak` to call the machine's transition on roll failure. Until this lands, casting still uses the legacy path (CastingState pointer field still exists, set in parallel — see below).

- [ ] **Step 1: Audit casting writer + reader sites**

```bash
git --no-pager grep -n "CastingState" -- internal/ | grep -v "_test.go"
```

Categorize hits:
- WRITERS: places that set `c.CastingState = &CastingState{...}` or `c.CastingState = nil`
- READERS: places that read `c.CastingState.SpellId`, `.FoldsAccumulated`, etc.

Note each path. The user-side starter is `skill.cast.go`; mob-side is `mobcommands/cast.go`. Concentration break is in `combat_shared_helpers.go`. Per-tick spell resolution is somewhere in `NewRound_DoCombat_*` files — find it.

- [ ] **Step 2: Modify `internal/usercommands/skill.cast.go`**

Find the section where `CastingState` is set on cast initiation (typically near the bottom of the `Cast` function, after the can-cast checks pass). Replace `c.CastingState = &CastingState{...}` with a parallel write to both legacy and new (migration safety):

```go
// Legacy + new in parallel. Legacy field deleted in Task 11.
castData := activity.CastingData{
    SpellId:              spellId,
    FoldsNeeded:          spellInfo.Folds,
    FoldsAccumulated:     0,
    FoldsPerRound:        foldsPerRound,
    TotalConvictionCost:  totalCost,
    ConvictionSpent:      perRoundCost, // first round paid up front
    TargetUserIds:        targetUserIds,
    TargetMobInstanceIds: targetMobIds,
    SpellRest:            spellRest,
}
if err := user.Character.Activity.TransitionToCasting(
    castData,
    state.TransitionReason{
        Trigger: activity.TriggerCastBegin,
        Actor:   state.ActorRef{UserId: user.UserId},
    },
); err != nil {
    // Activity machine refused (likely not Free). Send the same
    // "you can't cast while X" error the early IsCrafting check
    // used to produce.
    user.SendText("You're already busy with something else.")
    return true, nil
}

// Mirror to legacy field through Task 11.
user.Character.CastingState = &characters.CastingState{
    SpellId:              spellId,
    FoldsNeeded:          spellInfo.Folds,
    FoldsAccumulated:     0,
    FoldsPerRound:        foldsPerRound,
    TotalConvictionCost:  totalCost,
    ConvictionSpent:      perRoundCost,
    TargetUserIds:        targetUserIds,
    TargetMobInstanceIds: targetMobIds,
    SpellRest:            spellRest,
}
```

Also replace the early `if user.Character.IsCrafting() { ... }` check with `if !user.Character.IsFree() { ... }` so it covers all activity types.

Add the new imports:
```go
"github.com/GoMudEngine/GoMud/internal/state"
"github.com/GoMudEngine/GoMud/internal/state/activity"
```

- [ ] **Step 3: Modify `internal/mobcommands/cast.go`**

Same parallel-write pattern, mob actor ref. Find the cast initiation section and replace:

```go
castData := activity.CastingData{
    SpellId:              spellId,
    FoldsNeeded:          spellInfo.Folds,
    FoldsAccumulated:     0,
    FoldsPerRound:        foldsPerRound,
    TotalConvictionCost:  totalCost,
    ConvictionSpent:      perRoundCost,
    TargetUserIds:        targetUserIds,
    TargetMobInstanceIds: targetMobIds,
    SpellRest:            spellRest,
}
if err := mob.Character.Activity.TransitionToCasting(
    castData,
    state.TransitionReason{
        Trigger: activity.TriggerCastBegin,
        Actor:   state.ActorRef{MobInstanceId: mob.InstanceId},
    },
); err != nil {
    // Mob can't start cast — likely busy. Silent failure (no
    // SendText to a mob); btree will pick another action next tick.
    return true, nil
}

// Mirror to legacy field through Task 11.
mob.Character.CastingState = &characters.CastingState{
    SpellId:              spellId,
    FoldsNeeded:          spellInfo.Folds,
    FoldsAccumulated:     0,
    FoldsPerRound:        foldsPerRound,
    TotalConvictionCost:  totalCost,
    ConvictionSpent:      perRoundCost,
    TargetUserIds:        targetUserIds,
    TargetMobInstanceIds: targetMobIds,
    SpellRest:            spellRest,
}
```

Add imports as in Step 2.

- [ ] **Step 4: Modify `internal/hooks/combat_shared_helpers.go` — rewire `checkConcentrationBreak`**

Find the function. Currently it does the willpower roll and on failure does `c.CastingState = nil` + sends a message. Add the parallel machine transition:

```go
// checkConcentrationBreak rolls willpower vs damage and, on
// failure, cancels the cast. Both the legacy CastingState and
// the new Activity machine are updated through Task 11.
func checkConcentrationBreak(c *characters.Character, dmg int) bool {
    // ... existing roll logic returning `broke bool` ...

    if broke {
        // New: route through Activity machine.
        if c.Activity != nil && c.Activity.IsCasting() {
            _ = c.Activity.TransitionToFree(state.TransitionReason{
                Trigger: activity.TriggerConcentrationBreak,
                Actor:   c.Activity.Self(),
            })
        }
        // Legacy: nil out the pointer (still used by readers
        // until Task 11 sunsets it).
        c.CastingState = nil
    }
    return broke
}
```

- [ ] **Step 5: Modify the spell-resolution reader (per-tick consumer)**

Find the per-round consumer that reads `CastingState.FoldsAccumulated`, increments it, and on completion fires the spell. Per the survey it's somewhere in `NewRound_DoCombat_unified.go`. Pattern: keep reading from legacy `CastingState` for now (parallel-write keeps them in sync), but ALSO update the machine when the cast completes:

```go
// On cast completion (FoldsAccumulated >= FoldsNeeded):
//   1. Resolve the spell as today.
//   2. Clear legacy pointer as today: c.CastingState = nil
//   3. NEW: transition Activity machine.
if c.Activity != nil && c.Activity.IsCasting() {
    _ = c.Activity.TransitionToFree(state.TransitionReason{
        Trigger: activity.TriggerCastComplete,
        Actor:   c.Activity.Self(),
    })
}
c.CastingState = nil
```

When advancing folds on a per-tick increment (not yet complete), also keep the Activity machine's CastingData in sync. Since CastingData is value-typed and lives inside the machine, you have to either:
- (A) accept the slight drift between the two during the round (legacy is updated; new is read-only between transitions) — works as long as both completion paths fire the Free transition.
- (B) re-transition with updated data — wasteful (machine refuses Casting→Casting per AC-025).

Pick (A) for the migration window. After Task 11 sunsets the legacy field, the round-tick consumer is rewritten to update CastingData via the machine API. Plan: add a helper method `UpdateCastingProgress(folds int)` to the Machine if needed — but per the survey, the round tick just increments by `FoldsPerRound`, so it can be inferred.

Cleaner approach: keep the per-tick fold accumulation on the legacy field through the migration window. At Task 11, the round-tick consumer is rewritten to track the per-tick increment in CastingData directly via a new `Machine.AdvanceCastingFolds(n int)` helper. Don't preemptively add the helper — it's part of the Task 11 sunset rewrite.

- [ ] **Step 6: Build + test**

```bash
go build ./...
go test ./internal/state/... ./internal/characters/ ./internal/hooks/ ./internal/usercommands/ ./internal/mobcommands/ -count=1 2>&1 | grep -E "^(ok|FAIL)"
```
Expected: all PASS. Behavior is unchanged (legacy CastingState still in play); the machine just runs in parallel.

- [ ] **Step 7: Smoke check — boot the server, verify cast still works**

```bash
go build -o /tmp/dogmud-t6.exe . && /tmp/dogmud-t6.exe > /tmp/dogmud-t6.log 2>&1 &
SERVER_PID=$!
until grep -qE "Server Ready|panic" /tmp/dogmud-t6.log; do sleep 3; done
grep -E "Server Ready|panic" /tmp/dogmud-t6.log | head -5
kill -9 $SERVER_PID 2>/dev/null
```
Expected: `Server Ready`, no panic.

- [ ] **Step 8: Commit**

```bash
git add internal/usercommands/skill.cast.go \
        internal/mobcommands/cast.go \
        internal/hooks/combat_shared_helpers.go \
        internal/hooks/NewRound_DoCombat_unified.go
git commit -m "$(cat <<'EOF'
refactor(activity): casting migration — parallel writes through Task 11

Cast initiation (player + mob) now writes BOTH legacy CastingState
and new Activity.TransitionToCasting in parallel. The cast-time
pre-check switches from IsCrafting() to !IsFree() so it covers
all activity types.

checkConcentrationBreak rewired to fire Activity.TransitionToFree
(TriggerConcentrationBreak) on roll failure, in addition to the
legacy CastingState = nil clear.

Cast-complete path (per-tick consumer) also fires
Activity.TransitionToFree (TriggerCastComplete). Per-tick fold
accumulation continues to use the legacy field through the
migration window; Task 11 rewrites the consumer to use the
Machine API directly.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Crafting migration

**Files:**
- Modify: `internal/usercommands/craft.go`
- Modify: `internal/mobcommands/craft.go`
- Modify: `internal/usercommands/go.go` (movement cancel)
- Modify: `internal/hooks/NewRound_UserRoundTick.go` (round-tick consumer)
- Modify: `internal/hooks/NewRound_MobRoundTick.go` (round-tick consumer + remove mob-only combat-cancel block)

Mirror of Task 6 for crafting. Parallel-write CraftingData + legacy CraftingState. Replace `go.go`'s direct `c.CraftingState = nil` with the machine transition. Remove the mob-only combat-cancel block in `tickMobCrafting` (now covered by the cascade observer from Task 5).

- [ ] **Step 1: Audit crafting writer + reader sites**

```bash
git --no-pager grep -n "CraftingState" -- internal/ | grep -v "_test.go" | grep -v "salvage"
```

(Filter out salvage hits — those go in Task 8.)

- [ ] **Step 2: Modify `internal/usercommands/craft.go`**

Find craft initiation. Parallel-write:

```go
craftData := activity.CraftingData{
    RecipeId:       recipe.Id,
    RoundsTotal:    recipe.Rounds,
    RoundsComplete: 0,
    TargetSlot:     targetSlot,
}
if err := user.Character.Activity.TransitionToCrafting(
    craftData,
    state.TransitionReason{
        Trigger: activity.TriggerCraftBegin,
        Actor:   state.ActorRef{UserId: user.UserId},
    },
); err != nil {
    user.SendText("You're already busy with something else.")
    return true, nil
}

// Mirror to legacy field through Task 11.
user.Character.CraftingState = &characters.CraftingState{
    RecipeId:       recipe.Id,
    RoundsTotal:    recipe.Rounds,
    RoundsComplete: 0,
    TargetSlot:     targetSlot,
}
```

Replace the early `if user.Character.IsCrafting() { ... }` re-entrancy check with the same — it now reads from the machine via the rewritten predicate.

- [ ] **Step 3: Modify `internal/mobcommands/craft.go`** — same parallel-write pattern, mob actor ref:

```go
craftData := activity.CraftingData{
    RecipeId:       recipe.Id,
    RoundsTotal:    recipe.Rounds,
    RoundsComplete: 0,
    TargetSlot:     targetSlot,
}
if err := mob.Character.Activity.TransitionToCrafting(
    craftData,
    state.TransitionReason{
        Trigger: activity.TriggerCraftBegin,
        Actor:   state.ActorRef{MobInstanceId: mob.InstanceId},
    },
); err != nil {
    return true, nil
}

// Mirror to legacy field through Task 11.
mob.Character.CraftingState = &characters.CraftingState{
    RecipeId:       recipe.Id,
    RoundsTotal:    recipe.Rounds,
    RoundsComplete: 0,
    TargetSlot:     targetSlot,
}
```

Add imports as in Step 2.

- [ ] **Step 4: Modify `internal/usercommands/go.go` — movement cancel routes through machine**

Find the section (per the survey, lines ~64-67) that does `c.CraftingState = nil` + interrupt message on successful movement. Replace with:

```go
// New: route through Activity machine. Same behavior — cancels
// crafting AND salvaging (which also lives on Activity now).
if user.Character.Activity != nil && !user.Character.Activity.IsFree() {
    switch user.Character.Activity.State() {
    case activity.Crafting:
        _ = user.Character.Activity.TransitionToFree(state.TransitionReason{
            Trigger: activity.TriggerMovementInterrupt,
            Actor:   state.ActorRef{UserId: user.UserId},
        })
        user.SendText("Your movement interrupts your crafting.")
    case activity.Salvaging:
        _ = user.Character.Activity.TransitionToFree(state.TransitionReason{
            Trigger: activity.TriggerMovementInterrupt,
            Actor:   state.ActorRef{UserId: user.UserId},
        })
        user.SendText("Your movement interrupts your salvaging.")
    }
}

// Legacy: also nil-out the pointer (Task 11 deletes this).
user.Character.CraftingState = nil
```

Casting is NOT canceled by movement per the per-activity policy table.

- [ ] **Step 5: Modify `internal/hooks/NewRound_UserRoundTick.go` — craft round-tick**

The existing round-tick reads `c.CraftingState`, increments `RoundsComplete`, and on `>= RoundsTotal` fires recipe yield + clears. Add the parallel machine transition on completion:

```go
// On craft completion (RoundsComplete >= RoundsTotal):
//   1. Yield recipe outputs as today.
//   2. Clear legacy pointer: c.CraftingState = nil
//   3. NEW: transition Activity.
if c.Activity != nil && c.Activity.IsCrafting() {
    _ = c.Activity.TransitionToFree(state.TransitionReason{
        Trigger: activity.TriggerCraftComplete,
        Actor:   c.Activity.Self(),
    })
}
c.CraftingState = nil
```

Per-round increment continues to use the legacy field through the migration window (same rationale as Task 6 step 5).

- [ ] **Step 6: Modify `internal/hooks/NewRound_MobRoundTick.go` — same for mob, REMOVE mob-only combat-cancel block**

Mirror the craft-complete cascade. Then find the `tickMobCrafting` function and **delete** the block that auto-cancels mob crafting on `IsInCombat()` (per the survey, around line 404):

```go
// DELETE this block — Activity_Cascades.go observer now handles
// combat-entry cancel for both player and mob (parity per AC-038):
//
// if mob.Character.IsInCombat() {
//     mob.Character.CraftingState = nil
//     return
// }
```

Mob/player parity is now machine-driven.

- [ ] **Step 7: Build + test**

```bash
go build ./...
go test ./internal/state/... ./internal/characters/ ./internal/hooks/ ./internal/usercommands/ ./internal/mobcommands/ -count=1 2>&1 | grep -E "^(ok|FAIL)"
```
Expected: all PASS.

- [ ] **Step 8: Smoke check**

Boot the server, confirm `Server Ready`, no panic.

- [ ] **Step 9: Commit**

```bash
git add internal/usercommands/craft.go \
        internal/mobcommands/craft.go \
        internal/usercommands/go.go \
        internal/hooks/NewRound_UserRoundTick.go \
        internal/hooks/NewRound_MobRoundTick.go
git commit -m "$(cat <<'EOF'
refactor(activity): crafting migration — parallel writes + cascade-driven mob combat cancel

Craft initiation (player + mob) writes BOTH legacy CraftingState
and new Activity.TransitionToCrafting in parallel through Task 11.

Movement cancel in go.go routes through Activity.TransitionToFree
(TriggerMovementInterrupt) for Crafting AND Salvaging (matches
per-activity policy). Legacy pointer nil-out preserved through
migration window.

Per-round consumer fires Activity.TransitionToFree
(TriggerCraftComplete) on completion alongside the legacy clear.

DELETED: mob-only auto-cancel-on-combat block in
NewRound_MobRoundTick.tickMobCrafting. The Activity_Cascades
observer (Task 5) now handles combat-entry cancel for both
actors — restoring the parity asymmetry called out in the spec.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: Salvaging migration + hijack cleanup

**Files:**
- Modify: `internal/usercommands/salvage.go`
- Modify: `internal/mobcommands/salvage.go`
- Modify: `internal/hooks/NewRound_UserRoundTick.go` (`resolveSalvage`)
- Modify: `internal/hooks/NewRound_MobRoundTick.go` (mob salvage resolver, if multi-round; otherwise single-tick path)

Salvage gets its own state. The current hijack (`CraftingState.RecipeId = "salvage:<itemid>"` + `MiscData["salvage_item_uuid"]` + spoiled-potion flag in MiscData) is replaced by `SalvagingData`. Mob salvage stays single-tick at the resolution layer but uses `TransitionToSalvaging` for data-shape parity.

- [ ] **Step 1: Audit salvage sites**

```bash
git --no-pager grep -n "salvage_item_uuid\|salvage:\|salvage_spoiled" -- internal/ | head -20
```

Note the storage sites and the resolver consumers.

- [ ] **Step 2: Modify `internal/usercommands/salvage.go`**

Find the section that sets `CraftingState.RecipeId = "salvage:<itemid>"` + `MiscData["salvage_item_uuid"] = uuid`. Replace with:

```go
salvageData := activity.SalvagingData{
    ItemUuid:      item.Uuid,
    RoundsTotal:   rounds,
    SpoiledPotion: isSpoiledPotion,
}
if err := user.Character.Activity.TransitionToSalvaging(
    salvageData,
    state.TransitionReason{
        Trigger: activity.TriggerSalvageBegin,
        Actor:   state.ActorRef{UserId: user.UserId},
    },
); err != nil {
    user.SendText("You're already busy with something else.")
    return true, nil
}

// Mirror to legacy CraftingState slot through Task 11 for any
// readers still on the old path (mainly the round-tick resolver
// until Step 4 below rewires it).
user.Character.CraftingState = &characters.CraftingState{
    RecipeId:    "salvage:" + item.ItemId,
    RoundsTotal: rounds,
}
if user.Character.MiscData == nil {
    user.Character.MiscData = map[string]any{}
}
user.Character.MiscData["salvage_item_uuid"] = item.Uuid
if isSpoiledPotion {
    user.Character.MiscData["salvage_spoiled_potion"] = true
}
```

- [ ] **Step 3: Modify `internal/mobcommands/salvage.go`**

Per the spec: mob salvage stays single-tick at the RESOLUTION layer, but uses `TransitionToSalvaging` with a 1-round `SalvagingData` for data-shape parity:

```go
salvageData := activity.SalvagingData{
    ItemUuid:    item.Uuid,
    RoundsTotal: 1,
}
_ = mob.Character.Activity.TransitionToSalvaging(
    salvageData,
    state.TransitionReason{
        Trigger: activity.TriggerSalvageBegin,
        Actor:   state.ActorRef{MobInstanceId: mob.InstanceId},
    },
)

// Resolve immediately (existing behavior — single-tick).
// ... existing salvage roll + yield logic ...

_ = mob.Character.Activity.TransitionToFree(
    state.TransitionReason{Trigger: activity.TriggerSalvageComplete},
)
```

No legacy CraftingState mirror for mobs since the existing mob salvage path doesn't use CraftingState (per survey).

- [ ] **Step 4: Modify the salvage resolver in `internal/hooks/NewRound_UserRoundTick.go`**

Find `resolveSalvage`. It currently looks up `c.CraftingState.RecipeId` (starts with `"salvage:"`) + `c.MiscData["salvage_item_uuid"]`. Rewire to read SalvagingData:

```go
func resolveSalvage(c *characters.Character) {
    // New: read from Activity machine.
    d, ok := c.Activity.SalvagingData()
    if !ok {
        // Fallback for migration window: also check legacy
        // CraftingState if Activity isn't holding salvage data.
        // Remove this fallback in Task 11.
        if c.CraftingState == nil ||
            !strings.HasPrefix(c.CraftingState.RecipeId, "salvage:") {
            return
        }
        uuid, _ := c.MiscData["salvage_item_uuid"].(string)
        spoiled, _ := c.MiscData["salvage_spoiled_potion"].(bool)
        d = activity.SalvagingData{
            ItemUuid:      uuid,
            RoundsTotal:   c.CraftingState.RoundsTotal,
            RoundsComplete: c.CraftingState.RoundsComplete,
            SpoiledPotion: spoiled,
        }
    }

    // ... existing increment + completion roll using d fields ...

    // On completion:
    if d.RoundsComplete >= d.RoundsTotal {
        // ... existing yield + materials roll ...

        _ = c.Activity.TransitionToFree(state.TransitionReason{
            Trigger: activity.TriggerSalvageComplete,
            Actor:   c.Activity.Self(),
        })
        // Legacy clear through Task 11.
        c.CraftingState = nil
        delete(c.MiscData, "salvage_item_uuid")
        delete(c.MiscData, "salvage_spoiled_potion")
    }
}
```

- [ ] **Step 5: Build + test**

```bash
go build ./...
go test ./... -count=1 2>&1 | grep -E "^(ok|FAIL)" | head -20
```
Expected: all PASS.

- [ ] **Step 6: Smoke check**

Server boots cleanly.

- [ ] **Step 7: Commit**

```bash
git add internal/usercommands/salvage.go \
        internal/mobcommands/salvage.go \
        internal/hooks/NewRound_UserRoundTick.go \
        internal/hooks/NewRound_MobRoundTick.go
git commit -m "$(cat <<'EOF'
refactor(activity): salvaging migration — own state, hijack cleanup

Salvage now uses Activity.TransitionToSalvaging + SalvagingData
instead of hijacking CraftingState.RecipeId="salvage:<itemid>" and
MiscData["salvage_item_uuid"]/["salvage_spoiled_potion"].

Player salvage: parallel-writes legacy CraftingState + MiscData
keys through the migration window so any unmigrated reader keeps
working until Task 11.

Mob salvage: uses TransitionToSalvaging with a 1-round
SalvagingData for data-shape parity; resolution remains
single-tick (no per-round messaging for mobs by spec design).

resolveSalvage reads SalvagingData first; legacy CraftingState +
MiscData fallback for any in-flight salvages started before this
commit. Fallback removed in Task 11.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 9: Cancel command unification + mob cancel + btree primitive

**Files:**
- Modify: `internal/usercommands/cancel.go` (dispatch on Activity state)
- Create: `internal/mobcommands/cancel.go` (mob parity)
- Modify: `internal/behaviortree/actions.go` (`cancel_activity` action)
- Create: `internal/behaviortree/actions_test.go` or extend existing test file — smoke test for `cancel_activity`

- [ ] **Step 1: Modify `internal/usercommands/cancel.go`**

Replace the body with dispatch-on-state. Pattern (preserve the existing conviction-refund logic — extract from current cast cancel path):

```go
package usercommands

import (
    "github.com/GoMudEngine/GoMud/internal/events"
    "github.com/GoMudEngine/GoMud/internal/rooms"
    "github.com/GoMudEngine/GoMud/internal/state"
    "github.com/GoMudEngine/GoMud/internal/state/activity"
    "github.com/GoMudEngine/GoMud/internal/users"
)

func Cancel(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
    a := user.Character.Activity
    if a == nil || a.IsFree() {
        user.SendText("You aren't doing anything to cancel.")
        return true, nil
    }

    switch a.State() {
    case activity.Casting:
        d, _ := a.CastingData()
        // Refund 50% of unspent conviction (existing behavior).
        unspent := d.TotalConvictionCost - d.ConvictionSpent
        if unspent > 0 {
            refund := unspent / 2
            user.Character.Conviction += refund
            if user.Character.Conviction > user.Character.ConvictionMax.Value {
                user.Character.Conviction = user.Character.ConvictionMax.Value
            }
        }
        _ = a.TransitionToFree(state.TransitionReason{
            Trigger: activity.TriggerCastCancel,
            Actor:   state.ActorRef{UserId: user.UserId},
        })
        // Legacy clear through Task 11.
        user.Character.CastingState = nil
        user.SendText("You stop casting.")

    case activity.Crafting:
        _ = a.TransitionToFree(state.TransitionReason{
            Trigger: activity.TriggerCraftCancel,
            Actor:   state.ActorRef{UserId: user.UserId},
        })
        user.Character.CraftingState = nil
        user.SendText("You stop crafting.")

    case activity.Salvaging:
        _ = a.TransitionToFree(state.TransitionReason{
            Trigger: activity.TriggerSalvageCancel,
            Actor:   state.ActorRef{UserId: user.UserId},
        })
        user.Character.CraftingState = nil
        delete(user.Character.MiscData, "salvage_item_uuid")
        delete(user.Character.MiscData, "salvage_spoiled_potion")
        user.SendText("You stop salvaging.")
    }
    return true, nil
}
```

- [ ] **Step 2: Create `internal/mobcommands/cancel.go`**

Mirror, mob-flavored:

```go
package mobcommands

import (
    "github.com/GoMudEngine/GoMud/internal/mobs"
    "github.com/GoMudEngine/GoMud/internal/rooms"
    "github.com/GoMudEngine/GoMud/internal/state"
    "github.com/GoMudEngine/GoMud/internal/state/activity"
)

func Cancel(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
    a := mob.Character.Activity
    if a == nil || a.IsFree() {
        return true, nil
    }

    switch a.State() {
    case activity.Casting:
        d, _ := a.CastingData()
        unspent := d.TotalConvictionCost - d.ConvictionSpent
        if unspent > 0 {
            refund := unspent / 2
            mob.Character.Conviction += refund
            if mob.Character.Conviction > mob.Character.ConvictionMax.Value {
                mob.Character.Conviction = mob.Character.ConvictionMax.Value
            }
        }
        _ = a.TransitionToFree(state.TransitionReason{
            Trigger: activity.TriggerCastCancel,
            Actor:   state.ActorRef{MobInstanceId: mob.InstanceId},
        })
        mob.Character.CastingState = nil

    case activity.Crafting:
        _ = a.TransitionToFree(state.TransitionReason{
            Trigger: activity.TriggerCraftCancel,
            Actor:   state.ActorRef{MobInstanceId: mob.InstanceId},
        })
        mob.Character.CraftingState = nil

    case activity.Salvaging:
        _ = a.TransitionToFree(state.TransitionReason{
            Trigger: activity.TriggerSalvageCancel,
            Actor:   state.ActorRef{MobInstanceId: mob.InstanceId},
        })
        mob.Character.CraftingState = nil
        delete(mob.Character.MiscData, "salvage_item_uuid")
        delete(mob.Character.MiscData, "salvage_spoiled_potion")
    }
    return true, nil
}
```

Register in `internal/mobcommands/mobcommands.go`'s command table (look for the existing registry — `commands["suicide"] = Suicide` etc.).

- [ ] **Step 3: Modify `internal/behaviortree/actions.go` — add `cancel_activity`**

Find the action registry (the map of action names → functions). Add:

```go
// In the actions registry block:
"cancel_activity": actionCancelActivity,
```

Implementation (place with the other action funcs):

```go
// actionCancelActivity aborts the mob's current Activity, if any.
// Returns Success if anything was canceled, Failure if already Free.
// Used by behavior trees for tactical aborts:
//   - panic-flee on low HP (cancel offensive cast → flee)
//   - swap to heal mid-cast when ally is dying
//   - drop craft to defend when ambushed
func actionCancelActivity(mob *mobs.Mob, ctx EvalContext) Status {
    if mob.Character.Activity == nil ||
        mob.Character.Activity.IsFree() {
        return Failure
    }
    // Delegate to the mob cancel command for the refund + cleanup
    // logic so we don't duplicate it here.
    _, _ = mobcommands.Cancel("", mob, rooms.LoadRoom(mob.Character.RoomId))
    return Success
}
```

Add imports: `mobcommands`, `rooms`.

If a circular import results (mobcommands → btree → mobcommands), invert by calling `mob.Character.Activity.TransitionToFree(...)` directly from `actionCancelActivity` (dispatch on state same as cancel.go does — duplicates logic but breaks the cycle). Prefer the delegation path if no cycle; verify by attempting a build.

- [ ] **Step 4: Smoke test for the btree primitive**

Extend the existing `internal/behaviortree/actions_test.go` (or create it):

```go
func TestCancelActivityBtreeAction(t *testing.T) {
    // Setup a mob with an active cast.
    mob := mobs.NewMobByIdFresh(1, "test_zone")
    if mob == nil {
        t.Fatal("test fixture: mob 1 must exist")
    }
    _ = mob.Character.Activity.TransitionToCasting(
        activity.CastingData{SpellId: "fireball"},
        state.TransitionReason{Trigger: activity.TriggerCastBegin},
    )

    status := actionCancelActivity(mob, EvalContext{})
    if status != Success {
        t.Fatalf("expected Success, got %v", status)
    }
    if !mob.Character.Activity.IsFree() {
        t.Errorf("Activity should be Free after cancel_activity; got %v",
            mob.Character.Activity.State())
    }

    // Idempotency: calling again on Free returns Failure.
    status = actionCancelActivity(mob, EvalContext{})
    if status != Failure {
        t.Errorf("expected Failure when Activity already Free, got %v", status)
    }
}
```

If `mobs.NewMobByIdFresh` isn't the right constructor for tests, use whatever the existing btree tests do.

- [ ] **Step 5: Build + test**

```bash
go build ./...
go test ./internal/usercommands/ ./internal/mobcommands/ ./internal/behaviortree/ ./internal/state/activity/ -count=1 2>&1 | grep -E "^(ok|FAIL)"
```
Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/usercommands/cancel.go \
        internal/mobcommands/cancel.go \
        internal/mobcommands/mobcommands.go \
        internal/behaviortree/actions.go \
        internal/behaviortree/actions_test.go
git commit -m "$(cat <<'EOF'
feat(activity): cancel command unification + mob parity + btree primitive

usercommands/cancel.go: dispatches on Activity.State() — handles
Casting (50% conviction refund), Crafting, Salvaging. Replaces
the previous cast-only cancel.

mobcommands/cancel.go: NEW — mirrors player cancel, registered in
the mob command table for parity.

behaviortree action cancel_activity: aborts current mob Activity.
Returns Success if anything was canceled, Failure if already Free.
Enables tactical-abort patterns (panic-flee on low HP, swap to
heal mid-cast, drop craft to defend) — authoring of those
behavior trees is deferred to content / aliveness work.

Smoke test verifies the btree → cancel → IsFree path.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 10: Damage interrupt for craft/salvage + IsActing gate audit

**Files:**
- Modify: damage-application paths (likely `internal/hooks/NewRound_DoCombat_unified.go` and/or `combat_shared_helpers.go`)
- Modify: `internal/actions/command_readiness.go` (IsCrafting → IsActing)
- Audit + migrate: 30+ call sites of `IsCrafting()`

The two missing pieces from Task 5's cascade work: (a) damage cancels craft/salvage as a hard cancel (no roll), (b) the special-moves gate switches from `IsCrafting()` to `IsActing()`.

- [ ] **Step 1: Find the damage application path**

```bash
git --no-pager grep -n "checkConcentrationBreak\|ApplyHealthChange" -- internal/hooks/ | head -10
```

`checkConcentrationBreak` (Task 6) handles Casting. We need to add a sibling that handles Crafting + Salvaging unconditionally on damage:

- [ ] **Step 2: Add `cancelCraftOrSalvageOnDamage` helper in `combat_shared_helpers.go`**

```go
// cancelCraftOrSalvageOnDamage cancels the character's Activity if
// it's Crafting or Salvaging. No roll — damage is a hard cancel
// per the per-activity policy. Casting is handled separately by
// checkConcentrationBreak.
func cancelCraftOrSalvageOnDamage(c *characters.Character) {
    if c.Activity == nil {
        return
    }
    switch c.Activity.State() {
    case activity.Crafting:
        _ = c.Activity.TransitionToFree(state.TransitionReason{
            Trigger: activity.TriggerDamageInterrupt,
            Actor:   c.Activity.Self(),
        })
        c.CraftingState = nil
    case activity.Salvaging:
        _ = c.Activity.TransitionToFree(state.TransitionReason{
            Trigger: activity.TriggerDamageInterrupt,
            Actor:   c.Activity.Self(),
        })
        c.CraftingState = nil
        delete(c.MiscData, "salvage_item_uuid")
        delete(c.MiscData, "salvage_spoiled_potion")
    }
}
```

- [ ] **Step 3: Call the helper from every damage-application site**

Find every site where `checkConcentrationBreak` is called (it's invoked whenever a caster takes damage). Each of those sites is also a place where a crafter or salvager taking damage should have their activity cancelled. Add a call:

```go
// Existing:
broke := checkConcentrationBreak(target, dmg)

// NEW: also cancel craft/salvage on any damage (hard cancel).
cancelCraftOrSalvageOnDamage(target)
```

The two helpers are independent — the target might be Casting (concentration check) OR Crafting/Salvaging (hard cancel) OR Free (neither fires).

- [ ] **Step 4: Modify `internal/actions/command_readiness.go` — IsCrafting → IsActing**

Find the special-moves gate (per the survey, around line 29). It currently checks `c.IsCrafting()`:

```go
// Old (line ~29):
if c.IsCrafting() {
    return false, "You can't do that while crafting."
}

// New:
if c.IsActing() {
    return false, "You can't do that while you're busy."
}
```

The error message should generalize — "You can't do that while you're busy" or similar. (Pick a phrasing consistent with the codebase's tone — check other "you can't do X" messages.)

- [ ] **Step 5: Audit the 30+ `IsCrafting()` call sites**

```bash
git --no-pager grep -n "IsCrafting()" -- internal/ | grep -v "_test.go" | grep -v "internal/state/activity/"
```

For each hit, decide:
- KEEP `IsCrafting()` if the caller genuinely wants "is crafting specifically" (the craft command's own re-entrancy check is the canonical example).
- MIGRATE to `IsActing()` if the caller wants "is busy with anything" (most special-moves gates, the cast pre-check, the rally pre-check, etc.).

Walk every hit. Commit the audit decisions one site at a time if any are non-obvious. Don't batch-replace blindly.

- [ ] **Step 6: Build + test + smoke**

```bash
go build ./...
go test ./... -count=1 2>&1 | grep -E "^(ok|FAIL)" | head -20
```

Boot server, verify clean.

- [ ] **Step 7: Commit**

```bash
git add internal/hooks/combat_shared_helpers.go \
        internal/hooks/NewRound_DoCombat_unified.go \
        internal/actions/command_readiness.go \
        $(git --no-pager grep -l "IsCrafting" -- internal/ | grep -v _test)
git commit -m "$(cat <<'EOF'
feat(activity): damage-interrupt for craft/salvage + IsActing gate

cancelCraftOrSalvageOnDamage helper added to combat_shared_helpers
and called at every damage-application site alongside the existing
checkConcentrationBreak. Hard cancel (no roll) per the spec's
per-activity policy.

actions/command_readiness.go: special-moves gate switches from
IsCrafting() to IsActing() so casting + salvaging also block bash
/ kick / taunt / rally / warcry / trip. Error message generalized.

IsCrafting() call sites audited — most migrated to IsActing(),
a few (craft command's own re-entrancy check) deliberately
preserved.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 11: Sunset — delete CastingState/CraftingState fields + structs

**Files:**
- Modify: `internal/characters/character.go` (delete `CastingState` + `CraftingState` fields)
- Delete: `internal/characters/casting.go`
- Delete: `internal/characters/crafting.go`
- Remove the parallel-write blocks from Tasks 6-9
- Remove the salvage MiscData fallback in `resolveSalvage` (Task 8)

After this, the legacy pointer fields are gone, structs are gone, and the Activity machine is the sole source of truth.

- [ ] **Step 1: Audit any remaining direct `c.CastingState` / `c.CraftingState` reads**

```bash
git --no-pager grep -n "CastingState\|CraftingState" -- internal/ | grep -v "_test.go"
```

Any remaining hit needs migration. After Tasks 6-9, the remaining hits should be:
- The parallel-write mirror blocks in `skill.cast.go`, `craft.go`, `salvage.go`, `cancel.go` — DELETE these.
- The per-tick fold accumulation in `NewRound_DoCombat_unified.go` (Task 6 step 5 deferred this) — REWRITE to update CastingData via a new Machine helper.
- The per-tick round increment in `NewRound_UserRoundTick.go` / `NewRound_MobRoundTick.go` (Task 7 step 5 deferred this) — REWRITE similarly.

- [ ] **Step 2: Add per-tick helpers to the Machine**

In `internal/state/activity/rules.go`:

```go
// AdvanceCastingFolds increments folds accumulated and conviction
// spent on the current cast. Returns the updated data and true if
// the cast just completed (FoldsAccumulated >= FoldsNeeded).
//
// Caller is responsible for transitioning to Free + resolving the
// spell when complete = true.
func (m *Machine) AdvanceCastingFolds(folds int, convictionCost int) (CastingData, bool) {
    if m.State() != Casting || m.casting == nil {
        return CastingData{}, false
    }
    m.casting.FoldsAccumulated += folds
    m.casting.ConvictionSpent += convictionCost
    complete := m.casting.FoldsAccumulated >= m.casting.FoldsNeeded
    return *m.casting, complete
}

// AdvanceCraftingRound increments rounds complete on the current
// craft. Returns the updated data and true if the craft just
// completed.
func (m *Machine) AdvanceCraftingRound() (CraftingData, bool) {
    if m.State() != Crafting || m.crafting == nil {
        return CraftingData{}, false
    }
    m.crafting.RoundsComplete++
    complete := m.crafting.RoundsComplete >= m.crafting.RoundsTotal
    return *m.crafting, complete
}

// AdvanceSalvagingRound is the Salvaging equivalent.
func (m *Machine) AdvanceSalvagingRound() (SalvagingData, bool) {
    if m.State() != Salvaging || m.salvaging == nil {
        return SalvagingData{}, false
    }
    m.salvaging.RoundsComplete++
    complete := m.salvaging.RoundsComplete >= m.salvaging.RoundsTotal
    return *m.salvaging, complete
}
```

- [ ] **Step 3: Rewire the per-tick consumers**

In `NewRound_DoCombat_unified.go` (caster per-tick), replace:

```go
// OLD:
c.CastingState.FoldsAccumulated += foldsThisTick
c.CastingState.ConvictionSpent += costThisTick
if c.CastingState.FoldsAccumulated >= c.CastingState.FoldsNeeded {
    // resolve spell
    c.CastingState = nil
}

// NEW:
d, complete := c.Activity.AdvanceCastingFolds(foldsThisTick, costThisTick)
if complete {
    resolveSpell(c, d) // uses d.SpellId, d.TargetUserIds, etc.
    _ = c.Activity.TransitionToFree(state.TransitionReason{
        Trigger: activity.TriggerCastComplete,
        Actor:   c.Activity.Self(),
    })
}
```

(Where `resolveSpell` is the existing per-spell-id dispatch — adjust to take CastingData instead of *CastingState.)

Similarly for `NewRound_UserRoundTick.go` craft + salvage consumers, and the mob equivalents.

- [ ] **Step 4: Delete parallel-write mirror blocks**

For each command modified in Tasks 6-9, delete the `user.Character.CastingState = &characters.CastingState{...}` (and Craft/Salvage equivalents) blocks. The Activity machine is now sole truth.

- [ ] **Step 5: Delete the salvage MiscData fallback**

In `resolveSalvage` (Task 8 step 4), remove the `// Fallback for migration window` block. SalvagingData is the only path.

- [ ] **Step 6: Delete `CastingState` and `CraftingState` from `internal/characters/character.go`**

```go
// DELETE these fields:
CastingState  *CastingState  `yaml:"-"`
CraftingState *CraftingState `yaml:"-"`
```

- [ ] **Step 7: Delete `internal/characters/casting.go` and `internal/characters/crafting.go`**

```bash
git rm internal/characters/casting.go internal/characters/crafting.go
```

- [ ] **Step 8: Build — iterate until clean**

```bash
go build ./... 2>&1 | head -30
```

Any remaining reference to `CastingState` / `CraftingState` / the deleted struct types will surface as compile errors. Fix each — migrate to Activity machine API.

- [ ] **Step 9: Run full test suite**

```bash
go test ./... -count=1 2>&1 | grep -E "FAIL|^ok" | head -20
```
Expected: all PASS. Test fixtures that constructed Character with `.CastingState = &CastingState{...}` need updating to use `.Activity.TransitionToCasting(...)`.

- [ ] **Step 10: Final grep — zero remaining references**

```bash
git --no-pager grep -n "CastingState\|CraftingState" -- internal/ | head
```

Expected: zero hits except possibly historical comments. Any hit = sunset incomplete.

- [ ] **Step 11: Smoke check**

Boot server, verify clean, exercise cast / craft / salvage paths if possible.

- [ ] **Step 12: Commit**

```bash
git add -A
git commit -m "$(cat <<'EOF'
chore(activity): sunset CastingState / CraftingState — Activity is sole truth

DELETED:
  - Character.CastingState *CastingState field
  - Character.CraftingState *CraftingState field
  - internal/characters/casting.go (CastingState struct)
  - internal/characters/crafting.go (CraftingState struct)
  - Parallel-write mirror blocks in skill.cast.go, craft.go,
    salvage.go, cancel.go (Tasks 6-9 migration safety)
  - Salvage MiscData fallback in resolveSalvage
  - MiscData["salvage_item_uuid"] / ["salvage_spoiled_potion"]
    writers

ADDED to Machine API (rules.go):
  - AdvanceCastingFolds(folds, convictionCost)
  - AdvanceCraftingRound()
  - AdvanceSalvagingRound()
Each returns the updated per-state data + a "just completed" bool
so per-tick consumers can detect completion without re-querying.

Test fixtures that constructed Character with .CastingState =
&CastingState{...} migrated to .Activity.TransitionToCasting(...).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 12: Survey docs + helpfiles affected by chunk 3

**Files:** (survey — no production code changes)
- Create: `tools/testing/audits/2026-05-15-chunk-3-doc-helpfile-audit.md`

Mirrors chunk-2's T12. Produces a punch list that Task 13 consumes.

- [ ] **Step 1: Grep context.md files**

```bash
git --no-pager grep -ln "CastingState\|CraftingState\|IsCasting\|IsCrafting\|IsActing\|Activity machine\|salvage hijack" -- '*/context.md'
```

For each hit: DELETE / UPDATE / KEEP-AS-IS.

- [ ] **Step 2: Grep helpfiles (if any exist)**

Per chunk-2's T12 findings, `_datafiles/helpfiles/` doesn't exist in DOGMud — user help is embedded in command handlers. Verify the same for chunk-3 work:

```bash
ls _datafiles/helpfiles/ 2>&1 | head
git --no-pager grep -ln "cast\|craft\|salvage\|cancel" -- _datafiles/helpfiles/ 2>&1 | head
```

If no helpfiles, document the lack and move on.

- [ ] **Step 3: Grep top-level docs**

```bash
git --no-pager grep -ln "CastingState\|CraftingState\|Activity machine" -- '*.md' '*.txt' | head
```

Check PATCH_NOTES.md, COMBAT_STATE_ROADMAP.md, MOB_ALIVENESS_ROADMAP.md, CLAUDE.md.

- [ ] **Step 4: Grep scripting docs**

Per chunk-2 finding, scripting docs (`_datafiles/guides/building/scripting/`) sometimes drift from code. Grep:

```bash
git --no-pager grep -ln "CastingState\|CraftingState\|IsCasting\|IsCrafting" -- _datafiles/guides/
```

- [ ] **Step 5: Grep YAML lore for "mid-cast" / "mid-craft" mentions**

```bash
git --no-pager grep -ln "mid-cast\|mid-craft\|interrupt your craft\|interrupt your cast" -- _datafiles/
```

- [ ] **Step 6: Produce the audit document**

Write `tools/testing/audits/2026-05-15-chunk-3-doc-helpfile-audit.md` using the chunk-2 template (review chunk-2's audit for structure). Sections: context.md needing updates, deletions, keep-as-is, helpfiles (or note lack), top-level docs, YAML lore, scripting docs, test goals/roles, net new context.md files needed (`internal/state/activity/context.md`).

Be concise — one line per file.

- [ ] **Step 7: Commit**

```bash
git add tools/testing/audits/2026-05-15-chunk-3-doc-helpfile-audit.md
git commit -m "$(cat <<'EOF'
docs(audits): chunk-3 doc + helpfile audit

Survey of context.md files, helpfiles (or lack thereof), top-level
docs, YAML lore, scripting docs, and test fixtures mentioning
concepts changed (CastingState/CraftingState → Activity machine,
salvage hijack cleanup, IsActing gate, per-activity interrupt
policy). Feeds Task 13 documentation updates.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 13: Documentation + helpfile updates

**Files:**
- Read: `tools/testing/audits/2026-05-15-chunk-3-doc-helpfile-audit.md` (Task 12 output)
- Create: `internal/state/activity/context.md`
- Modify: `internal/characters/context.md` (Activity Machine Integration section)
- Modify: `internal/hooks/context.md` (Activity cascade observers section)
- Modify: `internal/forager/context.md` (if exists) — document why forager FSM stays separate
- Modify: every file flagged by the Task 12 audit
- Verify scripting docs — `IsCasting` / `IsCrafting` user-visible API names preserved; doc still accurate

- [ ] **Step 1: Re-read the Task 12 audit**

```bash
cat tools/testing/audits/2026-05-15-chunk-3-doc-helpfile-audit.md
```

Punch list for the rest of this task.

- [ ] **Step 2: Create `internal/state/activity/context.md`**

Use `internal/state/life/context.md` as the structural template. Required sections:
- **Overview** — package purpose, replaces what
- **Key Components** (file map): activity.go / transitions.go / rules.go / activity_test.go
- **State diagram** (ASCII): star topology
- **Per-state data**: FreeData, CastingData, CraftingData, SalvagingData — fields + meaning
- **Trigger constants** — purpose of each
- **Key functions** — TransitionTo* + Advance* + data accessors
- **Cascade integration** — list the observer files (Activity_Cascades.go) + which transitions they consume
- **Character API** — IsFree / IsCasting / IsCrafting / IsSalvaging / IsActing
- **Per-activity interrupt policy** (the table from the spec)
- **Notes on intentional asymmetries** — REQUIRED per user feedback during brainstorm:
  - No Foraging or Tracking state (one-shot today; ceremony without payoff)
  - Mob forager `forager.ForagerState` left in btree (different abstraction layer; AI orchestration vs character mechanic state — see `internal/forager/context.md`)
  - No IsForaging() / IsTracking() predicates on Character
  - Salvage gets its own state despite structural similarity to crafting (cleans up hijack, future divergence likely)
- **Persistence** — fully ephemeral; resets to Free on login
- **Testing notes** — Behavior Matrix AC-001 through AC-038, unit-level vs integration split

Target ~200-250 lines, match `internal/state/life/context.md` style.

- [ ] **Step 3: Update `internal/characters/context.md`**

Append "Activity Machine Integration (chunk 3)" section after the chunk-2 Life section. Covers: Activity field, IsFree/IsCasting/IsCrafting/IsSalvaging/IsActing predicates, sunset note (CastingState/CraftingState deleted, struct files gone, salvage MiscData keys deleted).

- [ ] **Step 4: Update `internal/hooks/context.md`**

Append "Activity Machine Cascade + Observers (chunk 3)" section. Covers: Activity_Cascades.go (Life Dead + Combat Phase Engaging), the call-site wirings (movement in go.go, damage via cancelCraftOrSalvageOnDamage), the per-tick consumer pattern (AdvanceCastingFolds / AdvanceCraftingRound / AdvanceSalvagingRound).

- [ ] **Step 5: Update `internal/forager/context.md` (if exists)**

If file exists, add a "Relationship to Character.Activity (chunk-3 boundary)" section explaining the asymmetry — `forager.ForagerState` is AI orchestration, `Character.Activity` is character mechanic state, mob foragers stay `Activity = Free` throughout the forage loop. Cross-reference `internal/state/activity/context.md` for the rationale.

If the file doesn't exist, skip this step — note it in the audit's post-T13 additions section.

- [ ] **Step 6: Update each audit-flagged file**

For each file the audit flagged: apply the recommended action (delete obsolete prose, update for new behavior, etc.). Stay surgical — don't rewrite unrelated sections.

- [ ] **Step 7: Verify scripting docs accuracy**

The IsCasting() / IsCrafting() function signatures are preserved (they query the machine internally). Verify scripting docs still describe them accurately. If new functions like IsActing() / IsSalvaging() are exposed to scripts (check `internal/scripting/`), document them.

- [ ] **Step 8: Build + smoke (defensive — docs shouldn't break anything but verify)**

```bash
go build ./...
```

- [ ] **Step 9: Update audit doc with post-T13 additions**

If you found anything during T13 work that the audit missed, append a "Post-T13 additions" section to the audit doc recording it.

- [ ] **Step 10: Commit**

```bash
git add internal/state/activity/context.md \
        internal/characters/context.md \
        internal/hooks/context.md \
        $(git diff --name-only -- internal/forager/ _datafiles/ tools/testing/audits/) \
        $(git diff --name-only -- '*.md')
git commit -m "$(cat <<'EOF'
docs(activity): chunk-3 documentation + helpfile updates

NEW: internal/state/activity/context.md — package docs for the
Activity state machine. Star topology + per-state data + trigger
constants + cascade integration. Notes on intentional asymmetries
section explains why Foraging/Tracking aren't states and why mob
forager FSM stays in btree (different abstraction layer).

UPDATED:
- internal/characters/context.md — Activity Machine Integration
  section with field, predicates, sunset note.
- internal/hooks/context.md — Activity cascade observers,
  movement + damage call-site wiring.
- internal/forager/context.md (where applicable) — relationship
  to Character.Activity.
- Audit-flagged files updated per the T12 punch list.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 14: Build / test / smoke validation

**Files:** (verification only)

- [ ] **Step 1: Full build**

```bash
go build ./...
```
Expected: clean.

- [ ] **Step 2: Full test suite**

```bash
go test ./... -count=1 2>&1 | grep -E "^(ok|FAIL)" | head -40
```
Expected: every package PASS.

- [ ] **Step 3: Activity Behavior Matrix status**

```bash
go test ./internal/state/activity/ -v -count=1 2>&1 | grep -E "^--- (PASS|FAIL|SKIP)" | awk '{print $2}' | sort | uniq -c
```
Expected: ~16 PASS (unit-level), ~22 SKIP (integration — verified by Tasks 5/9 hook + btree tests), 0 FAIL.

- [ ] **Step 4: Chunk 0 / 1 / 2 regression check**

```bash
go test ./internal/state/combatphase/ ./internal/state/awareness/ ./internal/state/life/ -count=1 2>&1 | grep -E "^(ok|FAIL)"
```
Expected: all PASS — no regressions.

- [ ] **Step 5: Server boot**

```bash
go build -o /tmp/dogmud-chunk3-validate.exe . && /tmp/dogmud-chunk3-validate.exe > /tmp/dogmud-chunk3-validate.log 2>&1 &
SERVER_PID=$!
until grep -qE "Server Ready|panic|FATAL" /tmp/dogmud-chunk3-validate.log; do sleep 3; done
grep -E "Server Ready|panic|FATAL|loadedCount" /tmp/dogmud-chunk3-validate.log | head -25
kill -9 $SERVER_PID 2>/dev/null
```
Expected: `Server Ready`, no panic, all data files load.

- [ ] **Step 6: Note in-game smoke scenarios deferred to user**

Per chunk-2 pattern, the in-game smoke scenarios are user-driven. After this task, the user runs:
- Player cast initiation + completion + cancel (manual `cancel` command)
- Player craft initiation + movement-cancel (was already working — regression test)
- Player craft initiation + damage-cancel (NEW behavior; verify it actually cancels now)
- Player craft + attack-self-initiates → combat (NEW: combat entry cancels craft for player)
- Player salvage start + manual cancel
- Mob cast + concentration break (was already working — regression)
- Mob craft + combat entry cancel (was already working — regression after the mob-only block deletion)
- Mob craft + damage cancel (NEW for mob)
- Btree `cancel_activity` action invoked from a test mob's btree
- Chunk 0 / 1 / 2 regression — no Awareness, Life, Combat Phase changes broke

DO NOT commit; just verify and report.

---

## Task 15: Roadmap closeout

**Files:**
- Modify: `COMBAT_STATE_ROADMAP.md`

- [ ] **Step 1: Mark chunk 3 Done**

Update the progress table row for chunk 3 from `Not started` to `Done (2026-05-15)` (or actual completion date).

Add a "Chunk 3 — Shipped" section parallel to the chunk 0/1/2 shipped paragraphs. Cover:
- New package `internal/state/activity/` with Free/Casting/Crafting/Salvaging
- Per-state data shapes preserved from legacy CastingState/CraftingState
- Salvage hijack cleaned up (own state)
- Per-activity interrupt policy formalized (table)
- Three mob/player parity asymmetries resolved
- `cancel_activity` btree primitive ships; tactical authoring deferred
- Sunset: CastingState/CraftingState fields + structs gone
- Behavior Matrix: ~38 tests, ~16 PASS + ~22 SKIP (integration-deferred)
- Intentional-asymmetry rationale documented in `internal/state/activity/context.md`

Update "Next" pointer to chunk 4 (Position).

- [ ] **Step 2: Commit**

```bash
git add COMBAT_STATE_ROADMAP.md
git commit -m "$(cat <<'EOF'
docs(roadmap): chunk 3 (Activity machine) Done

Activity machine consolidates Character.CastingState +
Character.CraftingState pointer fields, formalizes per-activity
interrupt rules, normalizes three mob/player asymmetries, and
cleans up the salvage-hijacks-crafting-slot pattern. Adds
cancel_activity btree primitive for tactical activity-abort
(authoring deferred to content/aliveness work after chunk 6).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Spec coverage check

| Spec section | Task(s) |
|---|---|
| State enumeration (Free/Casting/Crafting/Salvaging) | T1 |
| Per-state data (CastingData/CraftingData/SalvagingData) | T1 |
| Transition table (star topology) | T1 (transitions.go) |
| Trigger constants | T1 (transitions.go) |
| Machine API (TransitionTo*, *Data accessors, Inner/SetSelf) | T1, T3, T11 (Advance* helpers) |
| Character predicates (IsFree/IsCasting/IsCrafting/IsSalvaging/IsActing) | T4 |
| Per-activity interrupt policy: Cast + damage concentration break | T6 |
| Per-activity interrupt policy: Craft + damage hard cancel | T10 |
| Per-activity interrupt policy: Salvage + damage hard cancel | T10 |
| Per-activity interrupt policy: Craft + combat entry cancel | T5 (cascade) |
| Per-activity interrupt policy: Salvage + combat entry cancel | T5 (cascade) |
| Per-activity interrupt policy: Cast + combat entry no-op | T5 (cascade exempts Casting) |
| Per-activity interrupt policy: Craft + movement cancel | T7 (go.go) |
| Per-activity interrupt policy: Salvage + movement cancel | T7 (go.go) |
| Per-activity interrupt policy: Death cascade | T5 (Life observer) |
| Cross-activity start veto (transition table refusal) | T1 (table) + T6/T7/T8 (call-site error) |
| Mob/player parity normalization | T5 (cascade for both) + T7 (delete mob-only block) + T10 (damage for both) |
| Cancel command unification | T9 |
| Mob cancel command parity | T9 |
| Btree `cancel_activity` primitive | T9 |
| Sunset CastingState + CraftingState fields/structs | T11 |
| Sunset salvage MiscData hijack | T11 |
| Sunset chunk-2 Life cascade Activity pre-wire | T5 |
| Sunset mob-only combat-cancel in tickMobCrafting | T7 |
| IsActing replaces IsCrafting at special-moves gate | T10 |
| Behavior Matrix authored (~38 rows) | T2 |
| Notes on intentional asymmetries | T13 (context.md) |
| Doc + helpfile audit | T12 |
| Doc + helpfile updates | T13 |
| Build / test / smoke validation | T14 |
| Roadmap closeout | T15 |

All spec sections covered.

## Known followups (out of chunk 3)

- **Position machine (chunk 4)** may forbid Crafting while Clinched / Grounded — a Position-side veto added when chunk 4 lands.
- **Forager FSM unification** if cross-cutting consumers ever pull for it — currently `forager.ForagerState` stays in btree per the intentional-asymmetry decision.
- **Multi-round mob salvage with per-round messaging** — deferred per the spec; chunk 3 keeps mobs single-tick at the resolution layer.
- **Tactical activity-cancel behavior trees** — primitive ships in chunk 3; authoring (panic-flee on low HP, swap to heal mid-cast, etc.) is content/aliveness work after chunk 6.
- **Shared ability cooldown** — master spec logs this as a Phase-7 candidate (helper, not machine).
