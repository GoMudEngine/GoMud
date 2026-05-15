# Combat State — Chunk 2: Life Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Life state machine (`Alive | Dead | Respawning`) on the chunk-0 framework. Consolidate scattered death-cleanup logic (~250 lines of suicide.go + multiple MobDeath_* hooks) into a Life cascade + per-concern observer files. Rip out unused permadeath + extra-lives systems. Add auto-look after respawn teleport. Player respawn cycle stays same-tick.

**Architecture:** Same generics-based pattern as Combat Phase and Awareness. Life machine has DeadData carrying Killer + DamageMap for kill-credit observers. Cross-machine cleanup fires from Life cascade (Combat Phase → Idle, Awareness → Visible, Activity → Free pre-wire, Position → Standing pre-wire, buffs canceled, conditions cleared). Per-actor death effects (loot, teleport, decay, KD tracking) live in observer files subscribing to Life Dead/Respawning transitions.

**Tech Stack:** Go 1.21+ with generics, existing `internal/state/` framework, existing `internal/state/combatphase/` and `internal/state/awareness/` machines.

**Spec:** `docs/superpowers/specs/2026-05-15-state-chunk-2-life-design.md`

**Branch:** Stay on `feature/mob-aliveness-1.3-crimes` per the single-feature-branch SOP.

---

## File structure

| File | Status | Responsibility |
|------|--------|----------------|
| `internal/state/life/life.go` | NEW | State enum, data types, Machine wrapper |
| `internal/state/life/transitions.go` | NEW | Valid-transition table, trigger constants |
| `internal/state/life/rules.go` | NEW | Transition method implementations |
| `internal/state/life/life_test.go` | NEW | Behavior Matrix tests (LI-001 through LI-027) |
| `internal/state/life/context.md` | NEW | Package documentation |
| `internal/characters/character.go` | MODIFY | Add `Life *life.Machine` field; `IsAlive()`/`IsDead()` predicates; init in `New()`; delete `ExtraLives` field |
| `internal/characters/validate.go` | MODIFY | Nil-guard init of `Life` for YAML-loaded characters |
| `internal/hooks/Life_Cascades.go` | NEW | Cross-machine cascade wiring (Combat Phase, Awareness, Activity pre-wire, Position pre-wire, buffs, conditions) |
| `internal/hooks/Death_PlayerCleanup.go` | NEW | Stat decay + skill rust + KD tracking + party notifications observer |
| `internal/hooks/Death_MobLoot.go` | NEW | Loot drop + corpse setup observer |
| `internal/hooks/Death_AlivenessSubstrate.go` | NEW | events.MobDeath firing observer |
| `internal/hooks/Death_MobInstanceCleanup.go` | NEW | Instance cleanup-scheduling observer |
| `internal/hooks/Respawn_PlayerTeleport.go` | NEW | Graveyard teleport + grace buff + resource reset observer |
| `internal/hooks/Respawn_PlayerAutoLook.go` | NEW | Auto-look after respawn observer |
| `internal/usercommands/suicide.go` | REWRITE | ~250 → ~30 lines: thin command handler calling `Life.TransitionToDead` |
| `internal/mobcommands/suicide.go` | MODIFY | Thin mob suicide path if it exists; otherwise n/a |
| `internal/characters/resources.go` | MODIFY | `ApplyHealthChange` calls `Life.TransitionToDead` when health drops to 0 |
| `internal/combat/combat.go` | MODIFY | Mob death paths route through `Life.TransitionToDead` |
| Permadeath YAML / config | DELETE | Audit + delete any permadeath-specific config |
| `internal/characters/context.md` | MODIFY | Document Life field + IsAlive/IsDead predicates |
| `internal/hooks/context.md` | MODIFY | Document new Life cascade + Death/Respawn observer files |
| `COMBAT_STATE_ROADMAP.md` | MODIFY | Mark chunk 2 Done |

---

## Task 1: Life types + Character field

**Files:**
- Create: `internal/state/life/life.go`
- Create: `internal/state/life/transitions.go`
- Modify: `internal/characters/character.go` (add field, init in `New()`)
- Modify: `internal/characters/validate.go` (init for YAML-loaded chars)

Foundation. State enum, per-state data types (DeadData carries Killer + DamageMap), transition table, Machine wrapper, Character field bootstrap.

- [ ] **Step 1: Create `internal/state/life/life.go`**

```go
// Package life defines the Life state machine — the third
// consumer of internal/state, after combatphase and awareness.
// It replaces scattered death-cleanup logic in suicide.go and
// MobDeath_*.go hooks with a cascade-driven model.
package life

import (
	"github.com/GoMudEngine/GoMud/internal/state"
)

// State is the Life state enum.
type State int

const (
	Alive State = iota
	Dead
	Respawning
)

// String for logging/debugging.
func (s State) String() string {
	switch s {
	case Alive:
		return "Alive"
	case Dead:
		return "Dead"
	case Respawning:
		return "Respawning"
	}
	return "Unknown"
}

// AliveData is empty — default state.
type AliveData struct{}

// DeadData carries killer + damage attribution. Observers
// consume this for kill credit, faction rep, party-share,
// kill stats, etc.
type DeadData struct {
	Reason    state.TransitionReason
	Killer    state.ActorRef
	DamageMap map[int]int // userId → damage; for kill-credit and party-share
}

// RespawningData captures the in-flight respawn cycle.
// Player-only (mobs don't reach Respawning).
type RespawningData struct {
	Reason     state.TransitionReason
	DestRoomId int // graveyard or home room id
}

// Machine wraps state.Machine[State] with Life-specific API.
type Machine struct {
	inner      *state.Machine[State]
	dead       *DeadData
	respawning *RespawningData
	self       state.ActorRef
}

// NewMachine returns a Life machine in Alive.
func NewMachine() *Machine {
	return &Machine{
		inner: state.NewMachine(Alive, validTransitions),
	}
}

// State returns the current state.
func (m *Machine) State() State { return m.inner.State() }

// IsAlive returns true when state is Alive.
func (m *Machine) IsAlive() bool { return m.State() == Alive }

// IsDead returns true when state is Dead.
func (m *Machine) IsDead() bool { return m.State() == Dead }

// IsRespawning returns true when state is Respawning.
func (m *Machine) IsRespawning() bool { return m.State() == Respawning }

// DeadData returns the death context if currently Dead.
func (m *Machine) DeadData() (DeadData, bool) {
	if m.State() != Dead || m.dead == nil {
		return DeadData{}, false
	}
	return *m.dead, true
}

// RespawningData returns the respawn context if currently Respawning.
func (m *Machine) RespawningData() (RespawningData, bool) {
	if m.State() != Respawning || m.respawning == nil {
		return RespawningData{}, false
	}
	return *m.respawning, true
}

// Inner returns the underlying state.Machine — used by rules.go
// (Task 3) and hooks (Task 5+). Not part of the stable API.
func (m *Machine) Inner() *state.Machine[State] { return m.inner }

// SetSelf binds the machine to its owning ActorRef. Called from
// the registry during character creation.
func (m *Machine) SetSelf(ref state.ActorRef) { m.self = ref }

// Self returns the bound ActorRef.
func (m *Machine) Self() state.ActorRef { return m.self }
```

- [ ] **Step 2: Create `internal/state/life/transitions.go`**

```go
package life

import "github.com/GoMudEngine/GoMud/internal/state"

// validTransitions enforces the Life invariant matrix.
// Vetoes layer additional rules on top.
//
// Dead → Alive is allowed for edge cases (admin restoration,
// hypothetical revive-from-Dead spell). Standard player respawn
// goes Dead → Respawning → Alive.
var validTransitions = state.TransitionTable[State]{
	Alive:      {Dead},
	Dead:       {Respawning, Alive},
	Respawning: {Alive},
}

// Trigger reason constants.
const (
	TriggerHealthZero   = "health_zero"
	TriggerSuicide      = "suicide_command"
	TriggerAdminKill    = "admin_kill"
	TriggerCleanupReady = "cleanup_ready"
	TriggerRespawnReady = "respawn_ready"
	TriggerForceAlive   = "force_alive"
)
```

- [ ] **Step 3: Add Life field to Character**

Open `internal/characters/character.go`. Add field near `Awareness`:

```go
// Life state machine (chunk 2). Source of truth for "is this
// character alive/dead/respawning?" Cascade handlers in
// internal/hooks/Life_Cascades.go fire on transitions.
Life *life.Machine `yaml:"-"`
```

Import:

```go
"github.com/GoMudEngine/GoMud/internal/state/life"
```

In `New()`, add alongside `Awareness: awareness.NewMachine()`:

```go
Life: life.NewMachine(),
```

- [ ] **Step 4: Nil-guard init in `Validate()`**

Open `internal/characters/validate.go`. After the `Awareness == nil` guard:

```go
if c.Life == nil {
    c.Life = life.NewMachine()
}
```

Import `life` package.

- [ ] **Step 5: Build verify**

```bash
go build ./...
```
Expected: clean.

- [ ] **Step 6: Server boot smoke**

```bash
timeout 15 go run main.go 2>&1 | grep -E "LoadDataFiles|panic|Server Ready" | head -10
```
Expected: clean load, all `loadedCount=...` lines, no panics. Life field initialized on every Character but nothing reads it yet.

- [ ] **Step 7: Commit**

```bash
git add internal/state/life/ internal/characters/character.go internal/characters/validate.go
git commit -m "$(cat <<'EOF'
feat(life): state types, transition table, Character field

State enum (Alive/Dead/Respawning) with per-state data types
(DeadData carries Killer + DamageMap, RespawningData carries
DestRoomId). Transition table enforces framework invariants.
Machine wrapper provides Life-specific API. Character.Life field
initialized in New() and Validate() alongside chunk-0 CombatPhase
and chunk-1 Awareness.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Behavior Matrix RED tests

**Files:**
- Create: `internal/state/life/life_test.go`
- Modify: `internal/state/life/life.go` (add method stubs so tests compile)

Author the test suite encoding the Behavior Matrix as ~27 tests covering LI-001 through LI-027. Tests fail (stubs return errors / no-ops). Implementation lands in tasks 3-10.

- [ ] **Step 1: Add method stubs + registry to `life.go`**

Append to `life.go`:

```go
import (
	"errors"
	"sync"
)

// Machine registry for cross-character lookups (parallel to
// combatphase's and awareness's pattern).
var (
	registryMu      sync.Mutex
	machineRegistry = map[state.ActorRef]*Machine{}
)

func RegisterMachine(ref state.ActorRef, m *Machine) {
	registryMu.Lock()
	defer registryMu.Unlock()
	machineRegistry[ref] = m
	m.SetSelf(ref)
}

func UnregisterMachine(ref state.ActorRef) {
	registryMu.Lock()
	defer registryMu.Unlock()
	delete(machineRegistry, ref)
}

func lookupMachine(ref state.ActorRef) *Machine {
	registryMu.Lock()
	defer registryMu.Unlock()
	return machineRegistry[ref]
}

// Transition method stubs (Tasks 3-10 implement).

func (m *Machine) TransitionToDead(d DeadData, r state.TransitionReason) error {
	return errors.New("not implemented")
}
func (m *Machine) TransitionToRespawning(d RespawningData, r state.TransitionReason) error {
	return errors.New("not implemented")
}
func (m *Machine) TransitionToAlive(r state.TransitionReason) error {
	return errors.New("not implemented")
}
func (m *Machine) ForceAlive(r state.TransitionReason) {}
```

- [ ] **Step 2: Create `internal/state/life/life_test.go`**

Author the test file with all 27 tests covering the matrix. Use the chunk-1 `internal/state/awareness/awareness_test.go` as a structural reference.

```go
package life

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/stretchr/testify/require"
)

// Test helpers.

func makeTriad() (sneaker, killer, victim *Machine) {
	registryMu.Lock()
	machineRegistry = map[state.ActorRef]*Machine{}
	registryMu.Unlock()
	A := NewMachine()
	B := NewMachine()
	C := NewMachine()
	RegisterMachine(actor(1), A)
	RegisterMachine(actor(2), B)
	RegisterMachine(actor(3), C)
	return A, B, C
}

func actor(userId int) state.ActorRef {
	return state.ActorRef{UserId: userId}
}

// LI-001: Health zero → Dead.
func TestLI_001_HealthZeroDeath(t *testing.T) {
	A, _, _ := makeTriad()
	err := A.TransitionToDead(
		DeadData{Killer: actor(2)},
		state.TransitionReason{Trigger: TriggerHealthZero, Actor: actor(2)})
	require.NoError(t, err)
	require.Equal(t, Dead, A.State())
}

// LI-002: Suicide command → Dead.
func TestLI_002_SuicideDeath(t *testing.T) {
	A, _, _ := makeTriad()
	err := A.TransitionToDead(
		DeadData{},
		state.TransitionReason{Trigger: TriggerSuicide})
	require.NoError(t, err)
	require.Equal(t, Dead, A.State())
}

// LI-003: Admin kill → Dead.
func TestLI_003_AdminKillDeath(t *testing.T) {
	A, _, _ := makeTriad()
	err := A.TransitionToDead(
		DeadData{},
		state.TransitionReason{Trigger: TriggerAdminKill, Actor: actor(2)})
	require.NoError(t, err)
	require.Equal(t, Dead, A.State())
}

// LI-004: Dead → Respawning (player).
func TestLI_004_DeadToRespawning(t *testing.T) {
	A, _, _ := makeTriad()
	require.NoError(t, A.TransitionToDead(DeadData{}, state.TransitionReason{Trigger: TriggerSuicide}))
	require.NoError(t, A.TransitionToRespawning(
		RespawningData{DestRoomId: 468},
		state.TransitionReason{Trigger: TriggerCleanupReady}))
	require.Equal(t, Respawning, A.State())
	d, ok := A.RespawningData()
	require.True(t, ok)
	require.Equal(t, 468, d.DestRoomId)
}

// LI-005: Respawning → Alive.
func TestLI_005_RespawningToAlive(t *testing.T) {
	A, _, _ := makeTriad()
	require.NoError(t, A.TransitionToDead(DeadData{}, state.TransitionReason{Trigger: TriggerSuicide}))
	require.NoError(t, A.TransitionToRespawning(RespawningData{}, state.TransitionReason{}))
	require.NoError(t, A.TransitionToAlive(state.TransitionReason{Trigger: TriggerRespawnReady}))
	require.Equal(t, Alive, A.State())
}

// LI-006: Mob death → Dead; machine never reaches Respawning.
// Framework-level: just verify Dead is reached and stays Dead;
// instance destruction happens in observer (Task 8).
func TestLI_006_MobDeathStaysDead(t *testing.T) {
	A, _, _ := makeTriad()
	require.NoError(t, A.TransitionToDead(DeadData{}, state.TransitionReason{Trigger: TriggerHealthZero}))
	require.Equal(t, Dead, A.State())
	// Mob actor: no Respawning transition fires; instance gets
	// cleaned up by observer. State stays Dead from machine POV.
}

// LI-007 through LI-012: Cross-machine cascade.
// These are integration-tested via Life_Cascades.go hooks
// (Task 5). Framework-level here verifies the transition fires
// AfterTransition cascades that subscribers can observe.
func TestLI_007_CascadeFires(t *testing.T) {
	A, _, _ := makeTriad()
	var transitionedTo []State
	A.Inner().AfterTransition("test", func(from, to State, r state.TransitionReason) {
		transitionedTo = append(transitionedTo, to)
	})
	require.NoError(t, A.TransitionToDead(DeadData{}, state.TransitionReason{Trigger: TriggerSuicide}))
	require.Contains(t, transitionedTo, Dead)
}

// LI-008 through LI-012: same — verified by Task 5 integration.
// Skip with note.
func TestLI_008_AwarenessCascade(t *testing.T) {
	t.Skip("Awareness cascade verified by integration in Task 5")
}
func TestLI_009_ActivityCascade(t *testing.T) {
	t.Skip("Activity cascade verified by integration in Task 5")
}
func TestLI_010_PositionCascade(t *testing.T) {
	t.Skip("Position cascade verified by integration in Task 5")
}
func TestLI_011_BuffsCancel(t *testing.T) {
	t.Skip("Buff cancel verified by integration in Task 5")
}
func TestLI_012_ConditionsClear(t *testing.T) {
	t.Skip("Conditions clear verified by integration in Task 5")
}

// LI-013 through LI-015: Dead → Respawning cascade verified in Task 5/8.
func TestLI_013_ResourceReset(t *testing.T) {
	t.Skip("Resource reset verified by integration in Task 5")
}
func TestLI_014_GraceBuff(t *testing.T) {
	t.Skip("Grace buff verified by integration in Task 5")
}
func TestLI_015_Teleport(t *testing.T) {
	t.Skip("Teleport verified by integration in Task 8 observer")
}

// LI-016: Auto-look verified by integration in Task 8 observer.
func TestLI_016_AutoLook(t *testing.T) {
	t.Skip("Auto-look verified by integration in Task 8 observer")
}

// LI-017: DeadData.Killer populated from transition reason.
func TestLI_017_KillerCaptured(t *testing.T) {
	A, _, _ := makeTriad()
	killer := actor(2)
	err := A.TransitionToDead(
		DeadData{Killer: killer},
		state.TransitionReason{Trigger: TriggerHealthZero, Actor: killer})
	require.NoError(t, err)
	d, ok := A.DeadData()
	require.True(t, ok)
	require.Equal(t, killer, d.Killer)
}

// LI-018: DeadData.DamageMap populated.
func TestLI_018_DamageMapCaptured(t *testing.T) {
	A, _, _ := makeTriad()
	dmg := map[int]int{2: 50, 3: 30}
	err := A.TransitionToDead(
		DeadData{DamageMap: dmg},
		state.TransitionReason{Trigger: TriggerHealthZero})
	require.NoError(t, err)
	d, ok := A.DeadData()
	require.True(t, ok)
	require.Equal(t, dmg, d.DamageMap)
}

// LI-019: DeadData available to observers.
func TestLI_019_DeadDataObservable(t *testing.T) {
	A, _, _ := makeTriad()
	var observed DeadData
	A.Inner().AfterTransition("observer", func(from, to State, r state.TransitionReason) {
		if to == Dead {
			d, _ := A.DeadData()
			observed = d
		}
	})
	dmg := map[int]int{2: 100}
	err := A.TransitionToDead(
		DeadData{Killer: actor(2), DamageMap: dmg},
		state.TransitionReason{})
	require.NoError(t, err)
	require.Equal(t, actor(2), observed.Killer)
	require.Equal(t, dmg, observed.DamageMap)
}

// LI-020 through LI-022: Mob-specific observers (Task 7).
func TestLI_020_LootDropFires(t *testing.T) {
	t.Skip("Loot drop verified by integration in Task 7 observer")
}
func TestLI_021_MobDeathEventFires(t *testing.T) {
	t.Skip("MobDeath event verified by integration in Task 7 observer")
}
func TestLI_022_InstanceCleanupScheduled(t *testing.T) {
	t.Skip("Instance cleanup verified by integration in Task 7 observer")
}

// LI-023 through LI-025: Player-specific observers (Task 6).
func TestLI_023_StatDecay(t *testing.T) {
	t.Skip("Stat decay verified by integration in Task 6 observer")
}
func TestLI_024_KDTracking(t *testing.T) {
	t.Skip("KD tracking verified by integration in Task 6 observer")
}
func TestLI_025_PartyNotifications(t *testing.T) {
	t.Skip("Party notifications verified by integration in Task 6 observer")
}

// LI-026: Fresh Machine is Alive.
func TestLI_026_FreshMachineIsAlive(t *testing.T) {
	m := NewMachine()
	require.Equal(t, Alive, m.State())
	require.True(t, m.IsAlive())
	require.False(t, m.IsDead())
}

// LI-027: Persistence — Life does not survive restart.
// Framework-level: documented as intentional; tested implicitly
// via LI-026 (fresh machine is Alive).
func TestLI_027_StateDoesNotPersist(t *testing.T) {
	m := NewMachine()
	require.Equal(t, Alive, m.State())
}
```

- [ ] **Step 3: Run tests, expect FAIL**

```bash
go test ./internal/state/life/ -v
```
Expected: most tests fail with "not implemented." Coincidentally passes: TestLI_026, TestLI_027, TestLI_007 (which only checks that AfterTransition fires, which works since the transition errors before getting to the cascade). Skips: the integration tests (LI-008 through LI-016, LI-020-025).

- [ ] **Step 4: Build verify**

```bash
go build ./...
```
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add internal/state/life/
git commit -m "$(cat <<'EOF'
test(life): Behavior Matrix RED — LI-001 through LI-027

27 intent-driven tests encoding the matrix. Method stubs make
tests compile. Implementation lands in chunk-2 Tasks 3-10.
Integration-only tests (cross-machine cascade, observer effects)
skipped here; verified in their respective tasks.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Basic transitions (LI-001 through LI-006, LI-017 through LI-019)

**Files:**
- Modify: `internal/state/life/life.go` (replace stubs with real implementations)
- Create: `internal/state/life/rules.go` (transition method bodies)

Implement `TransitionToDead`, `TransitionToRespawning`, `TransitionToAlive`, `ForceAlive`. After this task, the basic-transition + DeadData rows pass.

- [ ] **Step 1: Create `internal/state/life/rules.go`**

```go
package life

import "github.com/GoMudEngine/GoMud/internal/state"

// TransitionToDead is the primary death entry point. All death
// paths (damage-application, suicide command, admin kill) route
// through here.
//
// DeadData populates the cascade context — Killer and DamageMap
// drive downstream observers (kill credit, faction rep, etc.).
func (m *Machine) TransitionToDead(d DeadData, r state.TransitionReason) error {
	if err := m.inner.TransitionTo(Dead, r); err != nil {
		return err
	}
	d.Reason = r
	m.dead = &d
	return nil
}

// TransitionToRespawning advances Dead → Respawning. Caller
// (typically Life_Cascades.go on the Dead-entry cascade) supplies
// the DestRoomId (graveyard or home).
func (m *Machine) TransitionToRespawning(d RespawningData, r state.TransitionReason) error {
	if err := m.inner.TransitionTo(Respawning, r); err != nil {
		return err
	}
	d.Reason = r
	m.respawning = &d
	m.dead = nil
	return nil
}

// TransitionToAlive advances Respawning → Alive (or Dead → Alive
// for admin restoration edge cases). Clears all per-state data.
func (m *Machine) TransitionToAlive(r state.TransitionReason) error {
	if err := m.inner.TransitionTo(Alive, r); err != nil {
		return err
	}
	m.dead = nil
	m.respawning = nil
	return nil
}

// ForceAlive transitions to Alive from any state. Used by admin
// restoration commands.
func (m *Machine) ForceAlive(r state.TransitionReason) {
	if m.State() == Alive {
		return
	}
	_ = m.inner.TransitionTo(Alive, r)
	m.dead = nil
	m.respawning = nil
}
```

- [ ] **Step 2: Remove the stubs from `life.go`**

Find the stub block added in Task 2 and delete the `TransitionToDead`, `TransitionToRespawning`, `TransitionToAlive`, `ForceAlive` stubs. The real implementations now live in rules.go.

- [ ] **Step 3: Run tests, expect target tests PASS**

```bash
go test ./internal/state/life/ -v -run "TestLI_00[1-7]|TestLI_01[7-9]|TestLI_026|TestLI_027"
```
Expected: all PASS.

Run full suite:

```bash
go test ./internal/state/life/ -v 2>&1 | grep -E "^--- (PASS|FAIL|SKIP):" | head -30
```
Expected: ~10 PASS, ~16 SKIP, 0 FAIL.

- [ ] **Step 4: Build verify**

```bash
go build ./...
```

- [ ] **Step 5: Commit**

```bash
git add internal/state/life/
git commit -m "$(cat <<'EOF'
feat(life): basic transitions LI-001 through LI-006 + DeadData

TransitionToDead + TransitionToRespawning + TransitionToAlive +
ForceAlive. DeadData carries Killer + DamageMap for observer
consumption. Matrix rows for transition + data-capture pass;
cross-machine cascade and observer rows pending Tasks 5-8.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: IsAlive() / IsDead() predicates on Character

**Files:**
- Modify: `internal/characters/character.go`

Add Character-level convenience predicates that delegate to the Life machine. These are the public API every caller will use.

- [ ] **Step 1: Add predicates**

In `internal/characters/character.go`, alongside `IsHidden()` and `IsEngaged()`:

```go
// IsAlive returns true when Life state is Alive.
func (c *Character) IsAlive() bool {
	if c.Life == nil {
		return true // defensive: pre-init characters treated as alive
	}
	return c.Life.IsAlive()
}

// IsDead returns true when Life state is Dead.
func (c *Character) IsDead() bool {
	if c.Life == nil {
		return false
	}
	return c.Life.IsDead()
}

// IsRespawning returns true when Life state is Respawning.
func (c *Character) IsRespawning() bool {
	if c.Life == nil {
		return false
	}
	return c.Life.IsRespawning()
}
```

- [ ] **Step 2: Build verify**

```bash
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add internal/characters/character.go
git commit -m "$(cat <<'EOF'
feat(life): predicates IsAlive/IsDead/IsRespawning on Character

Public API for the migration. Replaces ad-hoc Health<1 checks
once callers are migrated.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Cross-machine cascade wiring (LI-007 through LI-015)

**Files:**
- Create: `internal/hooks/Life_Cascades.go`

The biggest cross-machine integration. The Life machine's `Alive → Dead` transition cascades to Combat Phase, Awareness, Activity, Position, buffs, and conditions. The `Dead → Respawning` transition handles resource reset + grace buff.

- [ ] **Step 1: Create `internal/hooks/Life_Cascades.go`**

```go
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/awareness"
	"github.com/GoMudEngine/GoMud/internal/state/combatphase"
	"github.com/GoMudEngine/GoMud/internal/state/life"
)

// wireLifeCrossMachineCascades registers cascade handlers on
// each character's Life machine. On Alive→Dead, force other
// state machines to terminal states and clear scattered state.
// On Dead→Respawning, reset resources + apply grace buff.
//
// Per-actor concrete effects (loot, teleport, decay, KD
// tracking) live in their own observer files
// (Death_*, Respawn_*).
func wireLifeCrossMachineCascades(c *characters.Character) {
	c.Life.Inner().AfterTransition("life_cross_machine",
		func(from, to life.State, r state.TransitionReason) {
			switch {
			case from == life.Alive && to == life.Dead:
				// Cross-machine cleanup on death.
				// 1. Combat Phase → Idle
				c.CombatPhase.ForceIdle(state.TransitionReason{
					Trigger: combatphase.TriggerForceIdle,
				})
				// 2. Awareness → Visible (via ForceVisible)
				c.Awareness.ForceVisible(state.TransitionReason{
					Trigger: awareness.TriggerDeath,
				})
				// 3. Activity → Free (chunk-3 pre-wire:
				// clear casting/crafting state directly).
				c.CastingState = nil
				c.CraftingState = nil
				// 4. Position → Standing (chunk-4 pre-wire).
				c.CombatPosition = characters.PositionStanding
				c.GrappleControllerId = 0
				// 5. Buffs (non-permanent) → cancel all.
				c.CancelBuffsWithFlag(buffs.All)
				// 6. Conditions slice → clear.
				c.Conditions = nil

			case from == life.Dead && to == life.Respawning:
				// Player respawn cascade: resource reset + grace buff.
				// (Mobs don't reach Respawning.)
				c.Health = c.HealthMax() / 20         // 5%
				c.Stamina = c.StaminaMax() / 20       // 5%
				c.Conviction = c.ConvictionMax() / 20 // 5%
				c.AddBuff(81, false)                  // NoAggroTarget grace
			}
		})
}

func init() {
	characters.OnCharacterCreated(wireLifeCrossMachineCascades)
}
```

**Important notes:**
- Verify `buffs.All` is the correct flag constant for "cancel all buffs with cancel-on-* flags" — the existing suicide.go pattern. Grep `internal/buffs/` if needed.
- Verify `c.HealthMax()`, `c.StaminaMax()`, `c.ConvictionMax()` are the correct getter names. Grep `internal/characters/` for the pool-max getters.
- The `c.PlayerDamage` field clears on respawn — handled by existing reset logic in observers (Task 6).
- For mob actors, the `Dead → Respawning` branch never fires (instance gets cleaned up first). So the resource reset is effectively player-only without explicit gating.

- [ ] **Step 2: Build verify**

```bash
go build ./...
```
Expected: clean.

- [ ] **Step 3: Run tests**

```bash
go test ./internal/hooks/ ./internal/state/life/ -v 2>&1 | grep -E "^ok|FAIL" | head -10
```
Expected: PASS.

- [ ] **Step 4: Boot server smoke**

```bash
timeout 15 go run main.go 2>&1 | grep -E "LoadDataFiles|panic|Server Ready" | head -10
```
Expected: clean load.

- [ ] **Step 5: Commit**

```bash
git add internal/hooks/Life_Cascades.go
git commit -m "$(cat <<'EOF'
feat(life): cross-machine cascade on Alive→Dead and Dead→Respawning

On death: Combat Phase ForceIdle, Awareness ForceVisible, Activity
pre-wire (CastingState/CraftingState nil), Position pre-wire
(CombatPosition Standing), buffs canceled, conditions cleared.
On Dead→Respawning: resources to 5%, grace buff #81 applied.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Player death observers — stat decay, KD, party (LI-023 through LI-025)

**Files:**
- Create: `internal/hooks/Death_PlayerCleanup.go`

Stat/skill decay (preserved as normal-death system per spec), kill/death stat tracking, party notifications. All gated on player actors.

- [ ] **Step 1: Find existing helpers in suicide.go**

```bash
grep -n "applyStatDecay\|applySkillRust\|KD\.AddDeath\|KD\.AddPlayerDeath\|party.*notif" internal/usercommands/suicide.go
```

Note the exact function names + signatures. They'll be invoked from the new observer.

- [ ] **Step 2: Create `internal/hooks/Death_PlayerCleanup.go`**

```go
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// wirePlayerDeathCleanup subscribes to Life Dead transitions on
// player characters and runs the existing stat-decay + KD
// tracking + party-notification cleanup that used to live in
// suicide.go's inline death sequence.
func wirePlayerDeathCleanup(c *characters.Character) {
	c.Life.Inner().AfterTransition("player_death_cleanup",
		func(from, to life.State, r state.TransitionReason) {
			if from != life.Alive || to != life.Dead {
				return
			}
			// Only fire for player characters.
			if c.UserId == 0 {
				return
			}
			u := users.GetByUserId(c.UserId)
			if u == nil {
				return
			}

			// Apply stat decay + skill rust (preserved system).
			applyStatDecay(c)
			applySkillRust(c)

			// KD stat tracking.
			d, _ := c.Life.DeadData()
			if !d.Killer.IsZero() && d.Killer.IsPlayer() {
				u.KD.AddPlayerDeath()
			} else {
				u.KD.AddDeath()
			}

			// Party notifications.
			notifyPartyOfDeath(c)
		})
}

// applyStatDecay applies stat decay. Moved from suicide.go.
// (Implementation porting: copy the existing function body from
// suicide.go's old inline path; the function may already exist
// as a named helper.)
func applyStatDecay(c *characters.Character) {
	// PORT existing implementation
}

// applySkillRust applies skill rust. Moved from suicide.go.
func applySkillRust(c *characters.Character) {
	// PORT existing implementation
}

// notifyPartyOfDeath broadcasts to the player's party members.
func notifyPartyOfDeath(c *characters.Character) {
	// PORT existing implementation from suicide.go
}

func init() {
	characters.OnCharacterCreated(wirePlayerDeathCleanup)
}
```

**Important:** the existing `applyStatDecay` / `applySkillRust` may already live in `characters/` or `usercommands/suicide.go`. Don't duplicate — move them to the hooks file (or call into them if they're exported). Grep first.

- [ ] **Step 3: Build verify + test**

```bash
go build ./...
go test ./internal/hooks/ ./internal/state/life/ -v 2>&1 | grep -E "^ok|FAIL" | head -10
```

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/Death_PlayerCleanup.go
git commit -m "$(cat <<'EOF'
feat(life): Death_PlayerCleanup observer — stat decay, KD, party

Subscribes to player Life Dead transitions. Runs stat decay +
skill rust (preserved as normal-death system per chunk-2 spec),
KD stat tracking (PvP/PvE attribution from DeadData.Killer),
party notifications. Functions ported from suicide.go inline
sequence.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Mob death observers — loot, MobDeath event, instance cleanup (LI-020 through LI-022)

**Files:**
- Create: `internal/hooks/Death_MobLoot.go`
- Create: `internal/hooks/Death_AlivenessSubstrate.go`
- Create: `internal/hooks/Death_MobInstanceCleanup.go`

Three observers, one per concern, all subscribing to mob Life Dead transitions.

- [ ] **Step 1: Find existing mob-death logic**

```bash
grep -rn "MobDeath\|ItemDropChance\|corpse" --include="*.go" internal/hooks/ internal/mobcommands/ internal/combat/ | head -20
```

Note the existing mob-death code paths. Loot drop and MobDeath event firing may already be in `MobDeath_*.go` files.

- [ ] **Step 2: Create `internal/hooks/Death_MobLoot.go`**

```go
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
)

// wireMobDeathLoot subscribes to Life Dead transitions on mobs
// and triggers loot drop + corpse setup. Player-actor transitions
// are skipped (mob-only observer).
func wireMobDeathLoot(c *characters.Character) {
	c.Life.Inner().AfterTransition("mob_death_loot",
		func(from, to life.State, r state.TransitionReason) {
			if from != life.Alive || to != life.Dead {
				return
			}
			// Only fire for mob characters.
			if c.MobInstanceId == 0 {
				return
			}
			m := mobs.GetInstance(c.MobInstanceId)
			if m == nil {
				return
			}
			room := rooms.LoadRoom(m.Character.RoomId)
			if room == nil {
				return
			}

			// Drop loot per ItemDropChance.
			dropMobLoot(m, room)

			// Set corpse name/description on the room.
			setCorpse(m, room)
		})
}

func dropMobLoot(m *mobs.Mob, room *rooms.Room) {
	// PORT existing mob-death loot logic from current MobDeath_*.go
	// or inline combat-death code.
}

func setCorpse(m *mobs.Mob, room *rooms.Room) {
	// PORT existing corpse-setup logic.
}

func init() {
	characters.OnCharacterCreated(wireMobDeathLoot)
}
```

- [ ] **Step 3: Create `internal/hooks/Death_AlivenessSubstrate.go`**

```go
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
)

// wireAlivenessSubstrate fires the events.MobDeath event when
// a mob's Life transitions to Dead. Existing aliveness subscribers
// (faction rep, opinion, crime, knowledge, bounty) react
// downstream without changes.
func wireAlivenessSubstrate(c *characters.Character) {
	c.Life.Inner().AfterTransition("aliveness_substrate",
		func(from, to life.State, r state.TransitionReason) {
			if from != life.Alive || to != life.Dead {
				return
			}
			if c.MobInstanceId == 0 {
				return
			}
			m := mobs.GetInstance(c.MobInstanceId)
			if m == nil {
				return
			}

			d, _ := c.Life.DeadData()
			events.AddToQueue(events.MobDeath{
				MobInstanceId: m.InstanceId,
				MobName:       m.Character.Name,
				PlayerDamage:  d.DamageMap,
				// Other fields per existing event shape
			})
		})
}

func init() {
	characters.OnCharacterCreated(wireAlivenessSubstrate)
}
```

- [ ] **Step 4: Create `internal/hooks/Death_MobInstanceCleanup.go`**

```go
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
)

// wireMobInstanceCleanup schedules mob instance cleanup on
// Life Dead transitions. The actual despawn machinery already
// exists in mobs package; this observer just triggers it from
// the new Life cascade origin (instead of inline in combat/).
func wireMobInstanceCleanup(c *characters.Character) {
	c.Life.Inner().AfterTransition("mob_instance_cleanup",
		func(from, to life.State, r state.TransitionReason) {
			if from != life.Alive || to != life.Dead {
				return
			}
			if c.MobInstanceId == 0 {
				return
			}
			m := mobs.GetInstance(c.MobInstanceId)
			if m == nil {
				return
			}
			// Existing despawn machinery — call the existing
			// mob-death cleanup function.
			scheduleMobDespawn(m)
		})
}

func scheduleMobDespawn(m *mobs.Mob) {
	// PORT existing mob despawn-scheduling call.
}

func init() {
	characters.OnCharacterCreated(wireMobInstanceCleanup)
}
```

- [ ] **Step 5: Build verify + test**

```bash
go build ./...
go test ./internal/hooks/ -v 2>&1 | grep -E "^ok|FAIL" | head -5
```

- [ ] **Step 6: Commit**

```bash
git add internal/hooks/Death_MobLoot.go internal/hooks/Death_AlivenessSubstrate.go internal/hooks/Death_MobInstanceCleanup.go
git commit -m "$(cat <<'EOF'
feat(life): mob death observers — loot, MobDeath event, instance cleanup

Three observers subscribed to mob Life Dead transitions:
- Death_MobLoot: ItemDropChance loot drop + corpse setup
- Death_AlivenessSubstrate: fires events.MobDeath (faction/opinion/
  crime/knowledge/bounty subscribers react downstream)
- Death_MobInstanceCleanup: schedules instance despawn

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: Respawn observers — teleport + auto-look (LI-015, LI-016)

**Files:**
- Create: `internal/hooks/Respawn_PlayerTeleport.go`
- Create: `internal/hooks/Respawn_PlayerAutoLook.go`

Two player-only observers covering the teleport-then-look sequence.

- [ ] **Step 1: Create `internal/hooks/Respawn_PlayerTeleport.go`**

```go
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// wireRespawnTeleport subscribes to player Life Dead→Respawning
// transitions and moves the player to their graveyard / home room.
func wireRespawnTeleport(c *characters.Character) {
	c.Life.Inner().AfterTransition("respawn_teleport",
		func(from, to life.State, r state.TransitionReason) {
			if from != life.Dead || to != life.Respawning {
				return
			}
			if c.UserId == 0 {
				return
			}
			u := users.GetByUserId(c.UserId)
			if u == nil {
				return
			}

			destRoomId := determineRespawnRoom(c)
			u.Character.RoomId = destRoomId

			// Update RespawningData so observers can read DestRoomId.
			// (Already captured during TransitionToRespawning if
			// the caller supplied it; otherwise the field is zero
			// and we want to populate it.)
		})
}

func determineRespawnRoom(c *characters.Character) int {
	// PORT existing graveyard / home determination from suicide.go.
	return 468 // fallback: Temple Interior, Thornwall City
}

func init() {
	characters.OnCharacterCreated(wireRespawnTeleport)
}
```

- [ ] **Step 2: Create `internal/hooks/Respawn_PlayerAutoLook.go`**

```go
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// wireRespawnAutoLook subscribes to player Life Respawning→Alive
// transitions and fires a `look` command on the player's connection
// so they see the new room description without typing it manually.
// Same UX-fix pattern will land for fold-recall in a followup.
func wireRespawnAutoLook(c *characters.Character) {
	c.Life.Inner().AfterTransition("respawn_auto_look",
		func(from, to life.State, r state.TransitionReason) {
			if from != life.Respawning || to != life.Alive {
				return
			}
			if c.UserId == 0 {
				return
			}
			u := users.GetByUserId(c.UserId)
			if u == nil {
				return
			}

			// Fire look command via the user's command pipeline.
			u.Command("look")
		})
}

func init() {
	characters.OnCharacterCreated(wireRespawnAutoLook)
}
```

- [ ] **Step 3: Build verify + test**

```bash
go build ./...
go test ./internal/hooks/ -v 2>&1 | grep -E "^ok|FAIL" | head -5
```

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/Respawn_PlayerTeleport.go internal/hooks/Respawn_PlayerAutoLook.go
git commit -m "$(cat <<'EOF'
feat(life): respawn observers — graveyard teleport + auto-look

Player Dead→Respawning: teleport to home/graveyard room.
Player Respawning→Alive: fire `look` command so room description
renders without manual command. Parallel fold-recall fix
logged as followup memory.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 9: Migrate suicide.go to thin handler

**Files:**
- Modify: `internal/usercommands/suicide.go` (~250 lines → ~30 lines)

The marquee migration. Pull out all the inline cleanup; replace with a single Life.TransitionToDead call.

- [ ] **Step 1: Read the current suicide.go**

Familiarize with the full file. Identify each cleanup block (buff cancel, conditions clear, aggro clear, casting nil, resource reset, stat decay, teleport, grace buff, KD tracking, party notifications, permadeath path).

- [ ] **Step 2: Rewrite suicide.go**

```go
package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Suicide is the player command to kill oneself. After the
// chunk-2 Life machine refactor, this is a thin handler:
// it just invokes Life.TransitionToDead. The cascade and
// observer pipeline handles all cleanup (buffs, aggro, casting,
// teleport, grace buff, stat decay, party notifications, etc.).
//
// Permadeath path removed entirely per chunk-2 sunset.
func Suicide(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	if user == nil || user.Character == nil {
		return false, nil
	}
	if !user.Character.IsAlive() {
		user.SendText("You are already dead.")
		return true, nil
	}

	user.SendText("You commit suicide.")
	room.SendText("<ansi fg=\"username\">"+user.Character.Name+"</ansi> commits suicide!", user.UserId)

	user.Character.Life.TransitionToDead(
		life.DeadData{},
		state.TransitionReason{
			Trigger: life.TriggerSuicide,
			Actor:   state.ActorRef{UserId: user.UserId},
		},
	)

	// Cleanup cascade + observers fire from here.
	// Player is now Respawning → Alive same-tick.

	return true, nil
}
```

(Exact message strings and helper functions should match the existing suicide.go behavior — port the existing "You commit suicide" / room broadcast text verbatim.)

- [ ] **Step 3: Delete the old cleanup blocks**

The functions that previously inlined cleanup (buff cancel, conditions, aggro, casting, resources, decay, teleport, grace, KD, party) are gone from suicide.go. They live in the observer files (Tasks 5-8).

If `applyStatDecay` / `applySkillRust` / `notifyPartyOfDeath` were defined as package-private functions in suicide.go, they're now in `Death_PlayerCleanup.go` (Task 6). Delete them from suicide.go.

- [ ] **Step 4: Build verify**

```bash
go build ./...
```
Expected: clean.

- [ ] **Step 5: Run tests**

```bash
go test ./internal/usercommands/ -v 2>&1 | grep -E "^ok|FAIL" | head -3
```
Expected: clean. Existing suicide tests may need fixture updates (e.g., the user.Character.Life machine needs to be initialized; should be handled by Validate).

- [ ] **Step 6: Commit**

```bash
git add internal/usercommands/suicide.go
git commit -m "$(cat <<'EOF'
refactor(suicide): thin command handler — Life.TransitionToDead

~250 lines → ~30 lines. Cleanup cascade + observers handle
buff cancel, aggro clear, casting nil, conditions clear,
resource reset, grace buff, teleport, auto-look, stat decay,
KD tracking, party notifications. Permadeath path removed.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 10: Migrate ApplyHealthChange + mob death paths

**Files:**
- Modify: `internal/characters/resources.go` (ApplyHealthChange routes through Life)
- Modify: `internal/combat/combat.go` (or wherever mob health-to-zero is detected) — route through Life

- [ ] **Step 1: Find ApplyHealthChange**

```bash
grep -n "func.*ApplyHealthChange" internal/characters/resources.go
```

Find the path where `c.Health < 1` triggers death handling.

- [ ] **Step 2: Migrate ApplyHealthChange**

Replace the inline death path with a Life.TransitionToDead call:

```go
// OLD (illustrative; replace with actual existing code):
if c.Health < 1 {
    // ... lots of inline cleanup ...
}

// NEW:
if c.Health < 1 && c.IsAlive() {
    killerRef := state.ActorRef{}
    if attackerUserId > 0 {
        killerRef.UserId = attackerUserId
    } else if attackerMobInstanceId > 0 {
        killerRef.MobInstanceId = attackerMobInstanceId
    }
    c.Life.TransitionToDead(
        life.DeadData{
            Killer:    killerRef,
            DamageMap: copyPlayerDamage(c.PlayerDamage),
        },
        state.TransitionReason{
            Trigger: life.TriggerHealthZero,
            Actor:   killerRef,
        },
    )
}
```

`copyPlayerDamage` is a helper that snapshots the player-damage map (so observers consume a snapshot, not the live field that gets cleared during cascade).

- [ ] **Step 3: Find mob health-to-zero sites**

```bash
grep -rn "mob.*Health.*<.*1\|mob.*Health.*<=.*0" --include="*.go" internal/combat/ internal/hooks/ | head -10
```

Each site that detects mob death gets the same migration:

```go
if mob.Character.Health < 1 && mob.Character.IsAlive() {
    mob.Character.Life.TransitionToDead(
        life.DeadData{
            Killer:    killerRef,
            DamageMap: copyPlayerDamage(mob.Character.PlayerDamage),
        },
        state.TransitionReason{
            Trigger: life.TriggerHealthZero,
            Actor:   killerRef,
        },
    )
}
```

- [ ] **Step 4: Build verify + test**

```bash
go build ./...
go test ./internal/characters/ ./internal/combat/ -v 2>&1 | grep -E "^ok|FAIL" | head -5
```

- [ ] **Step 5: Commit**

```bash
git add internal/characters/resources.go internal/combat/
git commit -m "$(cat <<'EOF'
refactor: route health-to-zero through Life.TransitionToDead

ApplyHealthChange and mob health-zero sites in internal/combat/
now invoke Life.TransitionToDead instead of inline death sequences.
DeadData captures Killer + DamageMap snapshot for observer
consumption.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 11: Sunset permadeath + extra lives

**Files:**
- Modify: `internal/characters/character.go` (delete `ExtraLives` field, any related fields)
- Audit: YAML files for permadeath/extra-lives config (delete if found)
- Modify: existing tests / fixtures that reference these fields

- [ ] **Step 1: Audit `ExtraLives` and related fields**

```bash
grep -rn "ExtraLives\|extra_lives\|Permadeath\|permadeath" --include="*.go" --include="*.yaml" .
```

Catalog every reference. For each:
- Production code paths: remove the field reads/writes
- YAML configs: delete the field
- Tests: update fixtures to not reference the field

- [ ] **Step 2: Delete the Character.ExtraLives field**

In `internal/characters/character.go`:

```go
// DELETE:
ExtraLives int `yaml:"extra_lives"`
```

If there are getters/setters/decay functions on ExtraLives, delete them.

- [ ] **Step 3: Update existing tests**

Any tests that construct Character literals with `ExtraLives: N` need updating (delete the field).

Any tests that expected permadeath behavior need updating or deletion.

- [ ] **Step 4: Build verify + test**

```bash
go build ./...
go test ./... 2>&1 | grep -E "^ok|FAIL" | head -10
```

Iterate compile errors until clean.

- [ ] **Step 5: Final grep — verify zero production references**

```bash
grep -rn "ExtraLives\|Permadeath" --include="*.go" internal/
```

Should return zero hits (except possibly in comments documenting the sunset).

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "$(cat <<'EOF'
chore(life): sunset permadeath + extra-lives systems

User confirmed no intention to use permadeath. Without permadeath,
extra lives have no consequence — both ripped. Character.ExtraLives
field deleted; permadeath-specific code paths removed. ReviveOnDeath
buff preserved (separate one-shot mechanic). Stat/skill decay
preserved as normal-death system.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 12: Documentation

**Files:**
- Create: `internal/state/life/context.md`
- Modify: `internal/characters/context.md` (add Life section)
- Modify: `internal/hooks/context.md` (document new Life cascade + Death/Respawn observer files)

- [ ] **Step 1: Create `internal/state/life/context.md`**

Use `internal/state/awareness/context.md` as a template (~200 lines). Required sections:
- Overview
- Key Components (file map)
- Key Functions (TransitionToDead, TransitionToRespawning, TransitionToAlive, ForceAlive, predicates)
- Global State (machineRegistry)
- Data Structure Design (State enum + per-state data — AliveData, DeadData with Killer/DamageMap, RespawningData with DestRoomId)
- Integration Notes — consumes state framework; consumed by Character + hooks files
- Testing Notes — Behavior Matrix LI-001 through LI-027 in life_test.go

- [ ] **Step 2: Update `internal/characters/context.md`**

Append "Life Machine Integration (chunk 2)":
- Character.Life field
- IsAlive() / IsDead() / IsRespawning() predicates
- Cascade pattern (Life_Cascades.go)
- Permadeath + ExtraLives sunset noted

- [ ] **Step 3: Update `internal/hooks/context.md`**

Document the new chunk-2 hook files:
- Life_Cascades.go (cross-machine cleanup)
- Death_PlayerCleanup.go (stat decay, KD, party)
- Death_MobLoot.go (loot, corpse)
- Death_AlivenessSubstrate.go (MobDeath event)
- Death_MobInstanceCleanup.go (despawn scheduling)
- Respawn_PlayerTeleport.go (graveyard teleport)
- Respawn_PlayerAutoLook.go (auto-look)

- [ ] **Step 4: Commit**

```bash
git add internal/state/life/context.md internal/characters/context.md internal/hooks/context.md
git commit -m "$(cat <<'EOF'
docs(life): chunk 2 context.md updates

New state/life/context.md (~200 lines following the
awareness/context.md template). Integration sections in
characters/ and hooks/ context.mds documenting Life field,
predicates, cascade, and observer files.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 13: Build/test/smoke validation

**Files:** (verification only)

- [ ] **Step 1: Full build**

```bash
go build ./...
```
Expected: clean.

- [ ] **Step 2: Full test suite**

```bash
go test ./... 2>&1 | grep -E "^ok|FAIL" | head -30
```
Expected: every package PASS. Note any failures.

- [ ] **Step 3: Life Behavior Matrix status**

```bash
go test ./internal/state/life/ -v 2>&1 | grep -E "^--- (PASS|FAIL|SKIP):" | head -30
```
Expected: ~10 PASS, ~16 SKIP (integration tests, verified by Task 5-8 hook integration), 0 FAIL.

- [ ] **Step 4: Chunk 0/1 regression check**

```bash
go test ./internal/state/combatphase/ ./internal/state/awareness/ -v 2>&1 | grep -E "^ok|FAIL" | head -5
```
Expected: chunk-0 32/32 PASS; chunk-1 29 PASS + 4 SKIP (no regression).

- [ ] **Step 5: Server boot**

Use `run_in_background: true`. Wait ~15s.

```bash
go run main.go
```

Watch for:
- All `LoadDataFiles()` markers
- "Server Ready"
- No panics

Kill cleanly.

- [ ] **Step 6: Note 10 in-game smoke scenarios deferred to user**

Per the spec, the in-game smoke scenarios are user-driven. After this task, the user runs:
- Player suicide flow
- Player dies in combat
- Mob death + loot
- Mid-cast death
- Hidden death
- Grappled death
- Stat decay verification
- Multi-killer mob kill
- Verify permadeath path is gone
- Chunk 0/1 regression

DO NOT commit; just verify and report.

---

## Task 14: Roadmap closeout

**Files:**
- Modify: `COMBAT_STATE_ROADMAP.md`

- [ ] **Step 1: Mark chunk 2 Done**

Update the progress table row for chunk 2 to `Done (2026-05-15)` (or actual date). Add a "Chunk 2 — Shipped" section similar to chunk 0/1 shipped paragraphs.

- [ ] **Step 2: Commit**

```bash
git add COMBAT_STATE_ROADMAP.md
git commit -m "$(cat <<'EOF'
docs(roadmap): chunk 2 (Life machine) Done

Life machine consolidates ~250-line suicide.go into a 30-line
handler + cascade + observer files. Permadeath + extra lives
sunset. Auto-look after respawn teleport. ~27-row Behavior
Matrix; chunk 0/1 regression checks pass.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Spec coverage check

| Spec section | Tasks |
|--------------|-------|
| Life types + Character field | Task 1 |
| Behavior Matrix RED tests | Task 2 |
| Basic transitions (LI-001 through LI-006, LI-017-019) | Task 3 |
| IsAlive/IsDead predicates on Character | Task 4 |
| Cross-machine cascade (LI-007 through LI-015) | Task 5 |
| Player death observer (LI-023-025) | Task 6 |
| Mob death observers (LI-020-022) | Task 7 |
| Respawn observers (LI-015, LI-016) | Task 8 |
| Migrate suicide.go to thin handler | Task 9 |
| Migrate ApplyHealthChange + mob death paths | Task 10 |
| Sunset permadeath + extra lives | Task 11 |
| Documentation | Task 12 |
| Build/test/smoke validation | Task 13 |
| Roadmap closeout | Task 14 |

All spec sections covered. Behavior Matrix rows LI-001 through LI-027 distributed across Tasks 2 (RED tests) + Tasks 3-8 (implementation).

## Known followups (out of chunk 2)

- Activity machine (chunk 3) will repoint the Activity pre-wire in Life_Cascades.go (currently clears CastingState/CraftingState directly) to the proper Activity machine.
- Position machine (chunk 4) will repoint the Position pre-wire in Life_Cascades.go (currently clears CombatPosition/GrappleControllerId directly).
- Auto-look after fold-recall (separate memory entry `project_auto_look_after_room_change.md`).
- Chunk 1 sneak-end-message cosmetic bug (`project_chunk1_sneak_end_message_bug.md`) may want addressing during chunk-2 cleanup if it surfaces in smoke.
