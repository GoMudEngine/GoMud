# Pack Tactics Revamp Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the `mobs.MakeHostile` / `IsHostile` broad-group hostility-propagation hack with a routine-scoped, btree-driven pack reaction system that fires a `packmate_hurt` event on same-room packmates when a mob is attacked.

**Architecture:** New `Routine` and `RoutineLinks` string fields on `mobs.Mob`. A new `FindPackmatesInRoom` helper identifies packmates by routine match / link, scoped to the victim's room. A new `dispatchPackmateHurt` fires the `packmate_hurt` btree event on each packmate from inside `handleAggroAndAssist` (replacing the `MakeHostile` call). Archetypes (existing `generic_fighter`, `tank_taunter`, `melee_self_buff`, `pure_caster`; new `lookout`, `support_caster`, `leader`) gain `packmate_hurt` handlers. `callforhelp` is extended to fire `heard_callforhelp` on routine-matching mobs in adjacent rooms. After the infrastructure and archetypes are in place, priority-zone mobs get routines assigned, then the old system is deleted.

**Tech Stack:** Go (engine + tests). YAML (mob data, archetype behavior trees). `stretchr/testify` for assertions. Existing btree engine via `behaviortree.TryMobBehavior`.

**Spec:** `docs/superpowers/specs/2026-04-22-pack-tactics-revamp-design.md`

**Branch:** `feature/pack-tactics-revamp`

---

### Task 1: Create the feature branch and confirm baseline

**Files:**
- None (branch creation only)

- [ ] **Step 1: Branch from development**

```bash
git checkout development
git pull --ff-only origin development
git checkout -b feature/pack-tactics-revamp
```

- [ ] **Step 2: Confirm full test suite is green on the branch**

Run: `go build ./... && go test ./...`
Expected: `ok` for each package, no `FAIL`. Captures the pre-change baseline.

---

### Task 2: Add `Routine` and `RoutineLinks` fields to `mobs.Mob`

**Files:**
- Modify: `internal/mobs/mobs.go` (struct definition near line 120)
- Modify: `internal/mobs/mobs_test.go` (append new test)

- [ ] **Step 1: Write failing test — YAML with routine + routine_links populates the fields**

Append to `internal/mobs/mobs_test.go`:

```go
func TestMobYAMLLoadsRoutineAndRoutineLinks(t *testing.T) {
	yamlSrc := []byte(`
mobid: 999
zone: TestZone
routine: bandit_camp_guard
routine_links:
  - watch_north_road
  - bandit_back_camp
character:
  name: test bandit
`)
	var m Mob
	err := yaml.Unmarshal(yamlSrc, &m)
	require.NoError(t, err)
	assert.Equal(t, "bandit_camp_guard", m.Routine)
	assert.Equal(t, []string{"watch_north_road", "bandit_back_camp"}, m.RoutineLinks)
}
```

Ensure the imports at the top of the test file include `"gopkg.in/yaml.v2"` (match whatever version other mob tests use).

- [ ] **Step 2: Run test — it fails because the struct fields don't exist yet**

Run: `go test ./internal/mobs/ -run TestMobYAMLLoadsRoutineAndRoutineLinks -v`
Expected: FAIL (compile error — `m.Routine undefined`).

- [ ] **Step 3: Add the fields to `mobs.Mob`**

In `internal/mobs/mobs.go`, find the `Groups []string` field (around line 82) and add the new fields nearby (keep the existing comment style and alignment):

```go
	// Groups — existing
	Groups                  []string `yaml:"groups,omitempty"`  // (keep existing comment)

	// Pack-combat routine (v2-ready — see specs/2026-04-22-pack-tactics-revamp-design.md).
	// Freeform string compared with equality to other mobs' Routine for pack
	// identification. Mobs without a routine don't participate in packs.
	Routine                 string   `yaml:"routine,omitempty"`

	// Other routine strings this mob also reacts to. Example: a bandit
	// lookout with routine "watch_north_road" might list "bandit_camp_guard"
	// here so it receives the camp's call-for-help.
	RoutineLinks            []string `yaml:"routine_links,omitempty"`
```

(If the existing `Groups` field doesn't already have a `yaml` tag, don't touch it — keep the new fields' tag format consistent with whatever pattern mob YAML uses today. Read a few mob yamls first to confirm.)

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/mobs/ -run TestMobYAMLLoadsRoutineAndRoutineLinks -v`
Expected: PASS.

- [ ] **Step 5: Run full mobs package tests (regression)**

Run: `go test ./internal/mobs/`
Expected: `ok  github.com/GoMudEngine/GoMud/internal/mobs`.

- [ ] **Step 6: Commit**

```bash
git add internal/mobs/mobs.go internal/mobs/mobs_test.go
git commit -m "feat(mobs): add Routine and RoutineLinks fields

Pack-tactics revamp infrastructure (Task 1/N). Freeform-string fields
for the routine-scoped pack identification introduced by the pack-
tactics revamp spec. No consumers yet; just the data surface.

Spec: docs/superpowers/specs/2026-04-22-pack-tactics-revamp-design.md"
```

---

### Task 3: Implement `FindPackmatesInRoom` helper

**Files:**
- Create: `internal/mobs/packmates.go`
- Create: `internal/mobs/packmates_test.go`

- [ ] **Step 1: Write the failing tests first**

Create `internal/mobs/packmates_test.go`:

```go
package mobs

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/stretchr/testify/assert"
)

// seedPackmateScenario creates a victim + N candidate mobs in the same
// starting state. Returns (victim, candidates slice). Caller sets
// per-candidate routines/state before calling FindPackmatesInRoom.
func seedPackmateScenario(t *testing.T, victimRoutine string, candidateCount int) (*Mob, []*Mob) {
	t.Helper()
	victim := &Mob{
		InstanceId: 1,
		Routine:    victimRoutine,
		Character: characters.Character{
			RoomId: 100,
			Health: 50,
		},
	}
	candidates := make([]*Mob, candidateCount)
	instances := map[int]*Mob{1: victim}
	for i := 0; i < candidateCount; i++ {
		id := 100 + i
		m := &Mob{
			InstanceId: id,
			Character: characters.Character{
				RoomId: 100,
				Health: 50,
			},
		}
		candidates[i] = m
		instances[id] = m
	}
	cleanup := SeedMobsForTest(nil, instances)
	t.Cleanup(cleanup)
	return victim, candidates
}

func TestFindPackmatesInRoom_SameRoutine(t *testing.T) {
	victim, candidates := seedPackmateScenario(t, "bandit_camp_guard", 2)
	candidates[0].Routine = "bandit_camp_guard"
	candidates[1].Routine = "different_routine"

	got := FindPackmatesInRoom(victim)

	assert.Len(t, got, 1)
	assert.Equal(t, candidates[0].InstanceId, got[0].InstanceId)
}

func TestFindPackmatesInRoom_RoutineLinks_VictimLinksToCandidate(t *testing.T) {
	victim, candidates := seedPackmateScenario(t, "watch_north_road", 1)
	victim.RoutineLinks = []string{"bandit_camp_guard"}
	candidates[0].Routine = "bandit_camp_guard"

	got := FindPackmatesInRoom(victim)
	assert.Len(t, got, 1)
}

func TestFindPackmatesInRoom_RoutineLinks_CandidateLinksToVictim(t *testing.T) {
	victim, candidates := seedPackmateScenario(t, "bandit_camp_guard", 1)
	candidates[0].Routine = "watch_north_road"
	candidates[0].RoutineLinks = []string{"bandit_camp_guard"}

	got := FindPackmatesInRoom(victim)
	assert.Len(t, got, 1)
}

func TestFindPackmatesInRoom_DifferentRoom_Excluded(t *testing.T) {
	victim, candidates := seedPackmateScenario(t, "wolf_pack", 1)
	candidates[0].Routine = "wolf_pack"
	candidates[0].Character.RoomId = 999 // different room

	got := FindPackmatesInRoom(victim)
	assert.Empty(t, got)
}

func TestFindPackmatesInRoom_CharmedExcluded(t *testing.T) {
	victim, candidates := seedPackmateScenario(t, "wolf_pack", 1)
	candidates[0].Routine = "wolf_pack"
	candidates[0].Character.Charm(42, 100, "")

	got := FindPackmatesInRoom(victim)
	assert.Empty(t, got, "charmed mob should not be a packmate")
}

func TestFindPackmatesInRoom_DeadExcluded(t *testing.T) {
	victim, candidates := seedPackmateScenario(t, "wolf_pack", 1)
	candidates[0].Routine = "wolf_pack"
	candidates[0].Character.Health = 0

	got := FindPackmatesInRoom(victim)
	assert.Empty(t, got, "dead mob should not be a packmate")
}

func TestFindPackmatesInRoom_NoRoutineVictim_EmptyResult(t *testing.T) {
	victim, candidates := seedPackmateScenario(t, "", 2)
	candidates[0].Routine = "wolf_pack"
	candidates[1].Routine = "bandit_camp_guard"

	got := FindPackmatesInRoom(victim)
	assert.Empty(t, got, "victim with no routine has no packmates")
}

func TestFindPackmatesInRoom_ExcludesVictimItself(t *testing.T) {
	victim, _ := seedPackmateScenario(t, "wolf_pack", 0)

	got := FindPackmatesInRoom(victim)
	assert.Empty(t, got, "victim must not match itself")
}
```

- [ ] **Step 2: Run tests to verify they all fail**

Run: `go test ./internal/mobs/ -run TestFindPackmatesInRoom -v`
Expected: FAIL (compile error — `FindPackmatesInRoom undefined`).

- [ ] **Step 3: Implement `FindPackmatesInRoom` in a new file**

Create `internal/mobs/packmates.go`:

```go
package mobs

// FindPackmatesInRoom returns all mobs in the victim's room that should
// receive a packmate_hurt event when the victim is attacked.
//
// Inclusion rules (all must hold):
//   - Not the victim itself.
//   - Same RoomId as the victim.
//   - Health > 0 (dead mobs don't react).
//   - Not charmed (charmed mobs follow their owner, not the wild pack).
//   - Routine matches: either both mobs share a non-empty Routine, OR
//     the victim's Routine appears in the candidate's RoutineLinks, OR
//     the candidate's Routine appears in the victim's RoutineLinks.
//
// See docs/superpowers/specs/2026-04-22-pack-tactics-revamp-design.md
// (§ Packmate identification).
func FindPackmatesInRoom(victim *Mob) []*Mob {
	if victim == nil {
		return nil
	}
	if victim.Routine == "" && len(victim.RoutineLinks) == 0 {
		return nil
	}

	allIds := GetAllMobInstanceIds()
	packmates := make([]*Mob, 0, 4)

	for _, id := range allIds {
		if id == victim.InstanceId {
			continue
		}
		m := GetInstance(id)
		if m == nil {
			continue
		}
		if m.Character.RoomId != victim.Character.RoomId {
			continue
		}
		if m.Character.Health <= 0 {
			continue
		}
		if m.Character.IsCharmed() {
			continue
		}
		if !routinesMatch(victim, m) {
			continue
		}
		packmates = append(packmates, m)
	}
	return packmates
}

// routinesMatch returns true if a and b should be considered packmates
// for combat-reaction purposes.
func routinesMatch(a, b *Mob) bool {
	if a.Routine != "" && a.Routine == b.Routine {
		return true
	}
	for _, linked := range b.RoutineLinks {
		if linked != "" && linked == a.Routine {
			return true
		}
	}
	for _, linked := range a.RoutineLinks {
		if linked != "" && linked == b.Routine {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/mobs/ -run TestFindPackmatesInRoom -v`
Expected: all 8 tests PASS.

- [ ] **Step 5: Run full mobs package (regression)**

Run: `go test ./internal/mobs/`
Expected: `ok`.

- [ ] **Step 6: Commit**

```bash
git add internal/mobs/packmates.go internal/mobs/packmates_test.go
git commit -m "feat(mobs): FindPackmatesInRoom — routine-scoped pack identification

Pack-tactics revamp (Task 2/N). Helper used by the combat dispatcher
to identify which mobs receive packmate_hurt when the victim is
attacked. Same-room + matching-routine (or matching routine_link) +
alive + not charmed.

Spec: docs/superpowers/specs/2026-04-22-pack-tactics-revamp-design.md"
```

---

### Task 4: Implement `dispatchPackmateHurt`

**Files:**
- Create: `internal/hooks/packmate_hurt.go`
- Create: `internal/hooks/packmate_hurt_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/hooks/packmate_hurt_test.go`:

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
)

// TestDispatchPackmateHurt_FiresOnEachPackmate verifies the dispatcher
// calls behaviortree.TryMobBehavior with a packmate_hurt event on every
// mob that FindPackmatesInRoom returns, and no others.
func TestDispatchPackmateHurt_FiresOnEachPackmate(t *testing.T) {
	// Two packmate candidates (same routine as victim) and one unrelated.
	victim := &mobs.Mob{
		InstanceId: 1,
		Routine:    "bandit_camp_guard",
		Character:  characters.Character{RoomId: 100, Health: 50},
	}
	packmateA := &mobs.Mob{
		InstanceId: 2,
		Routine:    "bandit_camp_guard",
		Character:  characters.Character{RoomId: 100, Health: 50},
	}
	packmateB := &mobs.Mob{
		InstanceId: 3,
		Routine:    "bandit_camp_guard",
		Character:  characters.Character{RoomId: 100, Health: 50},
	}
	unrelated := &mobs.Mob{
		InstanceId: 4,
		Routine:    "temple_service",
		Character:  characters.Character{RoomId: 100, Health: 50},
	}
	cleanup := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{
		1: victim, 2: packmateA, 3: packmateB, 4: unrelated,
	})
	t.Cleanup(cleanup)

	// Install a spy that records which mob instance ids received which event.
	type firing struct {
		instId int
		evt    string
	}
	recorded := []firing{}
	orig := dispatchEventFn
	t.Cleanup(func() { dispatchEventFn = orig })
	dispatchEventFn = func(instId int, evt behaviortree.EventContext) bool {
		recorded = append(recorded, firing{instId: instId, evt: evt.EventType})
		return true
	}

	dispatchPackmateHurt(victim, 42 /*attackerUserId*/, 0 /*attackerMobInstanceId*/)

	assert.Len(t, recorded, 2)
	ids := []int{recorded[0].instId, recorded[1].instId}
	assert.Contains(t, ids, 2)
	assert.Contains(t, ids, 3)
	assert.NotContains(t, ids, 4, "unrelated mob must not receive packmate_hurt")
	for _, r := range recorded {
		assert.Equal(t, "packmate_hurt", r.evt)
	}
}
```

- [ ] **Step 2: Run test — expect FAIL (symbol doesn't exist)**

Run: `go test ./internal/hooks/ -run TestDispatchPackmateHurt_FiresOnEachPackmate -v`
Expected: FAIL (compile error: `dispatchPackmateHurt undefined`, `dispatchEventFn undefined`).

- [ ] **Step 3: Implement the dispatcher with a swappable event hook**

Create `internal/hooks/packmate_hurt.go`:

```go
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// dispatchEventFn is a package-level indirection so tests can spy on the
// behaviortree call without firing a real tree. Production assigns it to
// behaviortree.TryMobBehavior via init().
var dispatchEventFn = behaviortree.TryMobBehavior

// dispatchPackmateHurt fires a `packmate_hurt` btree event on every mob
// that shares a pack with the victim (same room, matching routine/link,
// alive, not charmed).
//
// Called from handleAggroAndAssist after the defender-side aggro has
// been established. Replaces the former mobs.MakeHostile call.
//
// Spec: docs/superpowers/specs/2026-04-22-pack-tactics-revamp-design.md
// (§ Event dispatch).
func dispatchPackmateHurt(victim *mobs.Mob, attackerUserId int, attackerMobInstanceId int) {
	if victim == nil {
		return
	}
	packmates := mobs.FindPackmatesInRoom(victim)
	for _, pm := range packmates {
		ctx := behaviortree.EventContext{
			EventType: "packmate_hurt",
			UserId:    attackerUserId,
			// MobInstanceId overloaded to mean "attacker mob" here so the
			// handler can identify PvM vs MvM context. Victim identity is
			// available via the receiving mob's room neighbor scan.
			MobInstanceId: attackerMobInstanceId,
		}
		dispatchEventFn(pm.InstanceId, ctx)
	}
}
```

If the `behaviortree.EventContext` struct does not have `UserId` / `MobInstanceId` fields (verify by grepping `internal/behaviortree/types.go`), use whatever fields the struct provides (match the existing `mob_hurt` event's shape — read how `mob_hurt` is fired for reference).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/hooks/ -run TestDispatchPackmateHurt_FiresOnEachPackmate -v`
Expected: PASS.

- [ ] **Step 5: Run full hooks package (regression)**

Run: `go test ./internal/hooks/`
Expected: `ok`.

- [ ] **Step 6: Commit**

```bash
git add internal/hooks/packmate_hurt.go internal/hooks/packmate_hurt_test.go
git commit -m "feat(hooks): dispatchPackmateHurt fires packmate_hurt btree event

Pack-tactics revamp (Task 3/N). Dispatcher called from combat to fan
out a packmate_hurt event to all of the victim's same-room packmates.
Event type: packmate_hurt. Handlers land in archetypes in subsequent
tasks.

Indirection via dispatchEventFn lets tests spy on the behaviortree
call without firing a real tree; prod value is
behaviortree.TryMobBehavior.

Spec: docs/superpowers/specs/2026-04-22-pack-tactics-revamp-design.md"
```

---

### Task 5: Wire `dispatchPackmateHurt` into `handleAggroAndAssist`

**Files:**
- Modify: `internal/hooks/NewRound_DoCombat_unified.go` (around line 666, PvM branch)

- [ ] **Step 1: Read the existing MakeHostile block to confirm line range**

Run: `grep -n "MakeHostile\|defMob" internal/hooks/NewRound_DoCombat_unified.go | head -10`
Expected output includes the `for _, groupName := range defMob.Groups { mobs.MakeHostile(...) }` block that Fix A earlier referenced at lines 666–669. Confirm the surrounding context (`case atk.IsPlayer() && !def.IsPlayer()` — the PvM branch).

- [ ] **Step 2: Replace the MakeHostile loop with a single dispatchPackmateHurt call**

In `internal/hooks/NewRound_DoCombat_unified.go`, inside the `case atk.IsPlayer() && !def.IsPlayer():` block, replace:

```go
			defMob := asMob(def)
			if cfg != nil {
				for _, groupName := range defMob.Groups {
					mobs.MakeHostile(groupName, atk.GetUserId(),
						cfg.Timing.MinutesToRounds(2)-atkChar.Stats.Charisma.ValueAdj)
				}
			}
```

with:

```go
			defMob := asMob(def)
			// Pack-tactics revamp: same-room routine-matching packmates
			// receive packmate_hurt. Replaces the former mobs.MakeHostile
			// group-flag propagation (which flagged every taxonomic group
			// the defender belonged to, causing priests to aggro on
			// respawning players who had attacked bandits).
			dispatchPackmateHurt(defMob, atk.GetUserId(), 0)
```

If the `mobs` import is no longer used anywhere else in the file after this change, goimports will clean it up on save; otherwise leave the import list alone.

- [ ] **Step 3: Verify build still compiles**

Run: `go build ./...`
Expected: `ok` (no errors). The old `MakeHostile` / `IsHostile` symbols are still defined in mobs.go at this point — deletion happens in a later task.

- [ ] **Step 4: Run combat-related tests (hooks + mobs)**

Run: `go test ./internal/hooks/ ./internal/mobs/`
Expected: `ok` for both. The pre-existing `TestMakeHostileAndIsHostile` still passes because the function still exists; it just has one fewer caller.

- [ ] **Step 5: Commit**

```bash
git add internal/hooks/NewRound_DoCombat_unified.go
git commit -m "refactor(hooks): handleAggroAndAssist fires packmate_hurt on PvM

Pack-tactics revamp (Task 4/N). The PvM branch used to call
mobs.MakeHostile for every group the defender belonged to. That
flagged the entire taxonomic group (e.g., 'humanoid') as hostile
to the attacker — any humanoid NPC in any zone would then aggro
the respawning player. Replaced with dispatchPackmateHurt which
fires packmate_hurt only on same-room routine-matching mobs.

mobs.MakeHostile / IsHostile / ReduceHostility remain defined for
now; they're deleted in a later task once the remaining consumer
(LookForTrouble group-hostility branch) is also removed.

Spec: docs/superpowers/specs/2026-04-22-pack-tactics-revamp-design.md"
```

---

### Task 6: Add `packmate_hurt` handler to `generic_fighter` archetype

**Files:**
- Modify: `_datafiles/world/dogmud/behaviors/archetypes/generic_fighter.yaml`
- Create: `internal/behaviortree/generic_fighter_packmate_hurt_test.go`

- [ ] **Step 1: Read the existing generic_fighter archetype to understand its shape**

Run: `cat _datafiles/world/dogmud/behaviors/archetypes/generic_fighter.yaml`
Expected: a YAML root with `tree:` that has `type: selector` and children keyed on `event: mob_combat_round`. The new `packmate_hurt` handler is added as a sibling selector-child gated on that event.

- [ ] **Step 2: Write the failing integration test**

Create `internal/behaviortree/generic_fighter_packmate_hurt_test.go`. Model it on the existing `melee_self_buff_archetype_integration_test.go` (read that file first for the test-harness pattern):

```go
package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
)

const genericFighterYAML = "../../_datafiles/world/dogmud/behaviors/archetypes/generic_fighter.yaml"

func TestGenericFighter_PackmateHurt_IssuesAttackCommand(t *testing.T) {
	cleanup := LoadArchetypeForTest("generic_fighter", genericFighterYAML)
	defer cleanup()

	mob := &mobs.Mob{
		InstanceId:        500,
		BehaviorArchetype: "generic_fighter",
		Character: characters.Character{
			RoomId: 1,
			Health: 50,
		},
	}
	seedCleanup := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{500: mob})
	defer seedCleanup()

	attackerUserId := 42
	ok := TryMobBehavior(mob.InstanceId, EventContext{
		EventType: "packmate_hurt",
		UserId:    attackerUserId,
	})

	assert.True(t, ok, "packmate_hurt handler should match and return ok")
	// Assert on command queue: mob should have issued `attack @42`.
	// Use whatever test helper exposes pending mob commands. If none
	// exists yet, use the archetype test pattern from
	// melee_self_buff_archetype_integration_test.go — that file's
	// assertions provide the model.
	assert.Contains(t, MobCommandsForTest(mob.InstanceId), "attack @42")
}
```

If `LoadArchetypeForTest` and `MobCommandsForTest` don't exist yet, check `test_export.go` and the melee_self_buff test file for the actual helper names — reuse them. Do NOT invent new test infrastructure.

- [ ] **Step 3: Run test — expect FAIL**

Run: `go test ./internal/behaviortree/ -run TestGenericFighter_PackmateHurt -v`
Expected: FAIL (archetype has no `packmate_hurt` handler yet, so the event doesn't match).

- [ ] **Step 4: Add the handler to the archetype YAML**

Edit `_datafiles/world/dogmud/behaviors/archetypes/generic_fighter.yaml`. Inside the root selector's children list, add a new child (sibling to the existing `mob_combat_round` child) BEFORE existing children so the handler can route first when relevant:

```yaml
    # packmate_hurt: a nearby packmate was attacked — engage the attacker.
    - type: sequence
      event: packmate_hurt
      children:
        - type: action
          do: command
          cmd: "attack @{event.UserId}"
```

The `{event.UserId}` token must match how the existing engine interpolates event fields in command strings — verify by grepping for similar patterns in already-working archetypes. If the engine does not interpolate, fall back to a native btree action like `attack_event_attacker` and add the corresponding Go action (check `actions_combat.go` and add a mirror of the existing `attack` action that reads `ctx.Event.UserId`).

- [ ] **Step 5: Run test — expect PASS**

Run: `go test ./internal/behaviortree/ -run TestGenericFighter_PackmateHurt -v`
Expected: PASS.

- [ ] **Step 6: Run full behaviortree suite (regression)**

Run: `go test ./internal/behaviortree/`
Expected: `ok`.

- [ ] **Step 7: Commit**

```bash
git add _datafiles/world/dogmud/behaviors/archetypes/generic_fighter.yaml \
        internal/behaviortree/generic_fighter_packmate_hurt_test.go
git commit -m "feat(archetypes): generic_fighter reacts to packmate_hurt

Pack-tactics revamp (Task 5/N). On packmate_hurt, a generic_fighter
issues 'attack @<attacker>' to engage the attacker. Same behavior
wolves (243) and zombies (301) already have during own-mob_hurt, now
cascaded to their packmates.

Spec: docs/superpowers/specs/2026-04-22-pack-tactics-revamp-design.md"
```

---

### Task 7: Add `packmate_hurt` handler to `tank_taunter` archetype

**Files:**
- Modify: `_datafiles/world/dogmud/behaviors/archetypes/tank_taunter.yaml`
- Create: `internal/behaviortree/tank_taunter_packmate_hurt_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/behaviortree/tank_taunter_packmate_hurt_test.go`. Model it exactly on Task 6's test, but assert on the `taunt` command and then the `attack` command in sequence:

```go
package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
)

const tankTaunterYAML = "../../_datafiles/world/dogmud/behaviors/archetypes/tank_taunter.yaml"

func TestTankTaunter_PackmateHurt_TauntsThenEngages(t *testing.T) {
	cleanup := LoadArchetypeForTest("tank_taunter", tankTaunterYAML)
	defer cleanup()

	mob := &mobs.Mob{
		InstanceId:        501,
		BehaviorArchetype: "tank_taunter",
		Character: characters.Character{
			RoomId: 1,
			Health: 50,
		},
	}
	seedCleanup := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{501: mob})
	defer seedCleanup()

	ok := TryMobBehavior(mob.InstanceId, EventContext{
		EventType: "packmate_hurt",
		UserId:    42,
	})
	assert.True(t, ok)

	cmds := MobCommandsForTest(mob.InstanceId)
	assert.Contains(t, cmds, "taunt @42")
	assert.Contains(t, cmds, "attack @42")
}
```

- [ ] **Step 2: Run test — expect FAIL**

Run: `go test ./internal/behaviortree/ -run TestTankTaunter_PackmateHurt -v`
Expected: FAIL.

- [ ] **Step 3: Add the handler to `tank_taunter.yaml`**

Add a new selector child:

```yaml
    - type: sequence
      event: packmate_hurt
      children:
        - type: action
          do: command
          cmd: "taunt @{event.UserId}"
        - type: action
          do: command
          cmd: "attack @{event.UserId}"
```

- [ ] **Step 4: Run test — expect PASS**

Run: `go test ./internal/behaviortree/ -run TestTankTaunter_PackmateHurt -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/behaviors/archetypes/tank_taunter.yaml \
        internal/behaviortree/tank_taunter_packmate_hurt_test.go
git commit -m "feat(archetypes): tank_taunter reacts to packmate_hurt

Pack-tactics revamp (Task 6/N). On packmate_hurt, a tank_taunter
taunts the attacker then engages. Applies to flesh_golem (305),
earth_elemental (311), magma_elemental (314).

Spec: docs/superpowers/specs/2026-04-22-pack-tactics-revamp-design.md"
```

---

### Task 8: Add `packmate_hurt` handler to `melee_self_buff` archetype

**Files:**
- Modify: `_datafiles/world/dogmud/behaviors/archetypes/melee_self_buff.yaml`
- Create: `internal/behaviortree/melee_self_buff_packmate_hurt_test.go`

- [ ] **Step 1: Write the failing test**

Create the test file following Task 6's pattern. Assert that on `packmate_hurt`:
1. The mob fires `cast_best_in_category` with `category: self_defense, target: self` first (reusing the existing buff catalog).
2. Then issues `attack @<attacker>`.

```go
package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
)

const meleeSelfBuffYAML2 = "../../_datafiles/world/dogmud/behaviors/archetypes/melee_self_buff.yaml"

func TestMeleeSelfBuff_PackmateHurt_SurgesThenEngages(t *testing.T) {
	cleanup := LoadArchetypeForTest("melee_self_buff", meleeSelfBuffYAML2)
	defer cleanup()

	mob := &mobs.Mob{
		InstanceId:        502,
		BehaviorArchetype: "melee_self_buff",
		Character: characters.Character{
			RoomId: 1,
			Health: 50,
		},
	}
	seedCleanup := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{502: mob})
	defer seedCleanup()

	ok := TryMobBehavior(mob.InstanceId, EventContext{
		EventType: "packmate_hurt",
		UserId:    42,
	})
	assert.True(t, ok)

	cmds := MobCommandsForTest(mob.InstanceId)
	assert.Contains(t, cmds, "attack @42")
	// The self_defense buff category cast is fire-and-forget via
	// cast_best_in_category; verify via the action's effect (spell
	// queued) using whatever spy the existing melee_self_buff test
	// uses. Read that test and mirror its assertions.
}
```

- [ ] **Step 2: Run test — expect FAIL**

Run: `go test ./internal/behaviortree/ -run TestMeleeSelfBuff_PackmateHurt -v`
Expected: FAIL.

- [ ] **Step 3: Add the handler to `melee_self_buff.yaml`**

Add a new selector child:

```yaml
    - type: sequence
      event: packmate_hurt
      children:
        - type: action
          do: cast_best_in_category
          category: self_defense
          target: self
        - type: action
          do: command
          cmd: "attack @{event.UserId}"
```

- [ ] **Step 4: Run test — expect PASS**

Run: `go test ./internal/behaviortree/ -run TestMeleeSelfBuff_PackmateHurt -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/behaviors/archetypes/melee_self_buff.yaml \
        internal/behaviortree/melee_self_buff_packmate_hurt_test.go
git commit -m "feat(archetypes): melee_self_buff reacts to packmate_hurt

Pack-tactics revamp (Task 7/N). On packmate_hurt, cast best
self_defense buff then engage. Applies to vampire (304), air (312),
fire (313).

Spec: docs/superpowers/specs/2026-04-22-pack-tactics-revamp-design.md"
```

---

### Task 9: Add `packmate_hurt` handler to `pure_caster` archetype

**Files:**
- Modify: `_datafiles/world/dogmud/behaviors/archetypes/pure_caster.yaml`
- Create: `internal/behaviortree/pure_caster_packmate_hurt_test.go`

- [ ] **Step 1: Write the failing test**

The pure_caster's packmate_hurt handler is opportunistic — if a same-room packmate has HP ratio below 40%, cast heal on them; otherwise fall through to the existing solo decision tree. Test both cases.

Create `internal/behaviortree/pure_caster_packmate_hurt_test.go`:

```go
package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
)

const pureCasterYAML2 = "../../_datafiles/world/dogmud/behaviors/archetypes/pure_caster.yaml"

// When a nearby packmate is below 40% HP, the pure_caster's packmate_hurt
// handler casts a heal spell at them (from the heal_friendly category).
func TestPureCaster_PackmateHurt_HealsWoundedPackmate(t *testing.T) {
	cleanup := LoadArchetypeForTest("pure_caster", pureCasterYAML2)
	defer cleanup()

	caster := &mobs.Mob{
		InstanceId:        503,
		BehaviorArchetype: "pure_caster",
		Routine:           "wraith_haunt",
		Character: characters.Character{
			RoomId: 1,
			Health: 50,
		},
	}
	caster.Character.HealthMax.Value = 100

	wounded := &mobs.Mob{
		InstanceId: 504,
		Routine:    "wraith_haunt",
		Character: characters.Character{
			RoomId: 1,
			Health: 30, // 30/100 = below 40%
		},
	}
	wounded.Character.HealthMax.Value = 100

	seedCleanup := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{
		503: caster, 504: wounded,
	})
	defer seedCleanup()

	ok := TryMobBehavior(caster.InstanceId, EventContext{
		EventType: "packmate_hurt",
		UserId:    42,
	})
	assert.True(t, ok)

	// Expect a heal_friendly cast targeting the wounded packmate.
	cmds := MobCommandsForTest(caster.InstanceId)
	assertHealCastOnPackmate(t, cmds, wounded.InstanceId)
}

// When no same-room packmate is wounded below 40%, the handler falls
// through to the existing solo decision tree (the caster's normal
// combat logic kicks in on the next mob_combat_round).
func TestPureCaster_PackmateHurt_NoWoundedPackmate_FallsThrough(t *testing.T) {
	cleanup := LoadArchetypeForTest("pure_caster", pureCasterYAML2)
	defer cleanup()

	caster := &mobs.Mob{
		InstanceId:        505,
		BehaviorArchetype: "pure_caster",
		Routine:           "wraith_haunt",
		Character: characters.Character{
			RoomId: 1,
			Health: 50,
		},
	}
	caster.Character.HealthMax.Value = 100
	healthyAlly := &mobs.Mob{
		InstanceId: 506,
		Routine:    "wraith_haunt",
		Character: characters.Character{
			RoomId: 1,
			Health: 95, // above 40% — not wounded
		},
	}
	healthyAlly.Character.HealthMax.Value = 100
	seedCleanup := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{
		505: caster, 506: healthyAlly,
	})
	defer seedCleanup()

	ok := TryMobBehavior(caster.InstanceId, EventContext{
		EventType: "packmate_hurt",
		UserId:    42,
	})
	// The handler matched (packmate_hurt fired) but its conditional
	// heal branch did not fire. Caller can now fall through to the
	// normal mob_combat_round tick. Assert no heal command queued.
	_ = ok
	cmds := MobCommandsForTest(caster.InstanceId)
	assertNoHealCast(t, cmds)
}

// Helpers — implement these at the bottom of the file once you know
// the exact command/spell string format the engine uses.
func assertHealCastOnPackmate(t *testing.T, cmds []string, targetId int) {
	t.Helper()
	for _, c := range cmds {
		// The cast_best_in_category action issues commands like
		// "cast <spell> #<targetMobInstanceId>" when targeting a mob.
		// Match on "heal" prefix + "#<targetId>" substring.
		if len(c) > 0 && containsAll(c, []string{"cast", "heal", formatMobTarget(targetId)}) {
			return
		}
	}
	t.Errorf("expected a heal cast targeting packmate #%d, got: %v", targetId, cmds)
}

func assertNoHealCast(t *testing.T, cmds []string) {
	t.Helper()
	for _, c := range cmds {
		if containsAll(c, []string{"cast", "heal"}) {
			t.Errorf("did not expect a heal cast, got: %q", c)
		}
	}
}

func containsAll(s string, needles []string) bool {
	for _, n := range needles {
		if !strContains(s, n) {
			return false
		}
	}
	return true
}

// strContains wraps strings.Contains to keep imports tidy; if the test
// file already imports "strings", inline the call instead.
func strContains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func formatMobTarget(id int) string {
	return "#" + formatInt(id)
}

func formatInt(n int) string {
	// strconv.Itoa without the import bikeshed
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := []byte{}
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}
```

(If `strings` is already imported elsewhere in the package, use `strings.Contains` + `strconv.Itoa` directly instead of the local helpers — simpler.)

- [ ] **Step 2: Run tests — expect FAIL**

Run: `go test ./internal/behaviortree/ -run TestPureCaster_PackmateHurt -v`
Expected: FAIL on both tests.

- [ ] **Step 3: Add the handler to `pure_caster.yaml`**

Insert a new selector child at the **top** of the `tree.children` list (so it evaluates before the existing solo decision priorities, but only matches on `packmate_hurt` events — other events fall through to the existing children unchanged):

```yaml
    # packmate_hurt: opportunistic heal on most-wounded same-room packmate.
    # Falls through (handler fails) if no packmate is below the HP threshold,
    # so the caster can resume solo behavior on its next mob_combat_round.
    - type: sequence
      event: packmate_hurt
      children:
        - type: condition
          check: packmate_below_hp_ratio
          threshold: 0.40
        - type: action
          do: cast_best_in_category
          category: heal_friendly
          target: most_wounded_packmate
```

This references two engine features that may not exist yet:
- `check: packmate_below_hp_ratio` — a btree condition. Add it in Task 10 below.
- `target: most_wounded_packmate` — a target resolver for `cast_best_in_category`. Add it in Task 11 below.

Keep this task focused on the YAML; the engine additions are the next two tasks. After those land, Task 9's tests will pass.

- [ ] **Step 4: Commit the archetype change (tests still failing pending Tasks 10 + 11)**

```bash
git add _datafiles/world/dogmud/behaviors/archetypes/pure_caster.yaml \
        internal/behaviortree/pure_caster_packmate_hurt_test.go
git commit -m "feat(archetypes): pure_caster opportunistic packmate heal (WIP)

Pack-tactics revamp (Task 8/N). Adds the packmate_hurt handler to
pure_caster that heals a same-room packmate below 40% HP. Handler
depends on two engine additions (Tasks 9 + 10): the
packmate_below_hp_ratio btree condition and the
most_wounded_packmate target resolver. Tests are committed failing
here; they turn green after those tasks land.

Spec: docs/superpowers/specs/2026-04-22-pack-tactics-revamp-design.md"
```

---

### Task 10: Implement `packmate_below_hp_ratio` btree condition

**Files:**
- Modify: `internal/behaviortree/conditions_mob.go` (add new condition handler)
- Create: `internal/behaviortree/conditions_mob_packmate_test.go`

- [ ] **Step 1: Read existing condition handlers to understand the shape**

Run: `grep -n "mob_health_below\|func cond\|checkFn" internal/behaviortree/conditions_mob.go | head -15`
Expected: pattern where `conditionsMob` registers string keys (like `"mob_health_below"`) mapping to `func(params map[string]any, ctx *EvalContext) bool`. Model the new condition after `mob_health_below`.

- [ ] **Step 2: Write the failing test**

Create `internal/behaviortree/conditions_mob_packmate_test.go`:

```go
package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
)

func TestCondition_PackmateBelowHpRatio_MatchesWoundedPackmate(t *testing.T) {
	self := &mobs.Mob{
		InstanceId: 700,
		Routine:    "bandit_camp_guard",
		Character:  characters.Character{RoomId: 1, Health: 50},
	}
	wounded := &mobs.Mob{
		InstanceId: 701,
		Routine:    "bandit_camp_guard",
		Character:  characters.Character{RoomId: 1, Health: 20},
	}
	wounded.Character.HealthMax.Value = 100
	cleanup := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{700: self, 701: wounded})
	defer cleanup()

	ctx := &EvalContext{InstanceId: 700}
	ok := condPackmateBelowHpRatio(map[string]any{"threshold": 0.40}, ctx)
	assert.True(t, ok)
}

func TestCondition_PackmateBelowHpRatio_NoMatchWhenHealthy(t *testing.T) {
	self := &mobs.Mob{
		InstanceId: 702,
		Routine:    "bandit_camp_guard",
		Character:  characters.Character{RoomId: 1, Health: 50},
	}
	healthy := &mobs.Mob{
		InstanceId: 703,
		Routine:    "bandit_camp_guard",
		Character:  characters.Character{RoomId: 1, Health: 95},
	}
	healthy.Character.HealthMax.Value = 100
	cleanup := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{702: self, 703: healthy})
	defer cleanup()

	ctx := &EvalContext{InstanceId: 702}
	ok := condPackmateBelowHpRatio(map[string]any{"threshold": 0.40}, ctx)
	assert.False(t, ok)
}
```

- [ ] **Step 3: Run tests — expect FAIL (symbol undefined)**

Run: `go test ./internal/behaviortree/ -run TestCondition_PackmateBelowHpRatio -v`
Expected: FAIL.

- [ ] **Step 4: Implement the condition**

Add to `internal/behaviortree/conditions_mob.go`:

```go
// condPackmateBelowHpRatio returns true iff at least one same-room
// packmate (per mobs.FindPackmatesInRoom) has HP ratio strictly below
// the `threshold` param (default 0.40). Used by the pure_caster
// archetype's packmate_hurt handler to skip the heal branch when no
// packmate actually needs healing.
func condPackmateBelowHpRatio(params map[string]any, ctx *EvalContext) bool {
	self := mobs.GetInstance(ctx.InstanceId)
	if self == nil {
		return false
	}
	threshold := getFloatParam(params, "threshold", 0.40)
	for _, pm := range mobs.FindPackmatesInRoom(self) {
		maxHp := pm.Character.HealthMax.Value
		if maxHp <= 0 {
			continue
		}
		ratio := float64(pm.Character.Health) / float64(maxHp)
		if ratio < threshold {
			return true
		}
	}
	return false
}
```

Register the handler in the conditions map (find the existing registrations in `conditions_mob.go`'s `init()` or registration function and add `"packmate_below_hp_ratio": condPackmateBelowHpRatio` alongside `"mob_health_below": condMobHealthBelow`).

If `getFloatParam` doesn't exist, use whatever param-reader the other conditions use (likely something like `asFloat64(params["threshold"])` with a fallback).

- [ ] **Step 5: Run tests — expect PASS**

Run: `go test ./internal/behaviortree/ -run TestCondition_PackmateBelowHpRatio -v`
Expected: PASS (2/2).

- [ ] **Step 6: Commit**

```bash
git add internal/behaviortree/conditions_mob.go \
        internal/behaviortree/conditions_mob_packmate_test.go
git commit -m "feat(btree): packmate_below_hp_ratio condition

Pack-tactics revamp (Task 9/N). Returns true if any same-room packmate
has HP ratio strictly below the given threshold. Used by the
pure_caster archetype to gate opportunistic packmate heals.

Spec: docs/superpowers/specs/2026-04-22-pack-tactics-revamp-design.md"
```

---

### Task 11: Implement `most_wounded_packmate` target resolver for `cast_best_in_category`

**Files:**
- Modify: `internal/behaviortree/action_cast_best_in_category.go` (extend target-resolution switch)
- Create: `internal/behaviortree/action_cast_best_in_category_packmate_test.go`

- [ ] **Step 1: Read the existing target resolution to understand the shape**

Run: `grep -n "target.*self\|target.*aggro\|resolveTarget\|case \"self\"" internal/behaviortree/action_cast_best_in_category.go | head -15`
Expected: a function that takes a `target` string ("self", "aggro", etc.) and returns a resolved target identity. Add a new case for `"most_wounded_packmate"`.

- [ ] **Step 2: Write the failing test**

Create `internal/behaviortree/action_cast_best_in_category_packmate_test.go`. The test seeds two packmates at different HP ratios and asserts the resolver picks the lower one.

```go
package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
)

func TestResolveTarget_MostWoundedPackmate(t *testing.T) {
	self := &mobs.Mob{
		InstanceId: 800,
		Routine:    "wraith_haunt",
		Character:  characters.Character{RoomId: 1, Health: 50},
	}
	self.Character.HealthMax.Value = 100

	lightlyWounded := &mobs.Mob{
		InstanceId: 801,
		Routine:    "wraith_haunt",
		Character:  characters.Character{RoomId: 1, Health: 60},
	}
	lightlyWounded.Character.HealthMax.Value = 100

	mostWounded := &mobs.Mob{
		InstanceId: 802,
		Routine:    "wraith_haunt",
		Character:  characters.Character{RoomId: 1, Health: 10},
	}
	mostWounded.Character.HealthMax.Value = 100

	cleanup := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{
		800: self, 801: lightlyWounded, 802: mostWounded,
	})
	defer cleanup()

	ctx := &EvalContext{InstanceId: 800}
	resolved := resolveCastTarget("most_wounded_packmate", ctx)

	assert.Equal(t, TargetShape{MobInstanceId: 802}, resolved,
		"resolver must pick the lower-ratio packmate")
}
```

(`TargetShape` and `resolveCastTarget` are placeholder names — use whatever the existing code uses. If the resolver returns `(int /*userId*/, int /*mobInstId*/)` tuple, adapt the assertion accordingly.)

- [ ] **Step 3: Run test — expect FAIL**

Run: `go test ./internal/behaviortree/ -run TestResolveTarget_MostWoundedPackmate -v`
Expected: FAIL (resolver doesn't recognize `most_wounded_packmate` yet).

- [ ] **Step 4: Implement the resolver case**

Find the existing target-resolution switch in `action_cast_best_in_category.go` and add:

```go
case "most_wounded_packmate":
    self := mobs.GetInstance(ctx.InstanceId)
    if self == nil {
        return /* unresolved target */
    }
    var best *mobs.Mob
    bestRatio := 1.0
    for _, pm := range mobs.FindPackmatesInRoom(self) {
        maxHp := pm.Character.HealthMax.Value
        if maxHp <= 0 {
            continue
        }
        ratio := float64(pm.Character.Health) / float64(maxHp)
        if ratio < bestRatio {
            bestRatio = ratio
            best = pm
        }
    }
    if best != nil {
        return /* target shape for mobInstId = best.InstanceId */
    }
    return /* unresolved target */
```

The exact return shape depends on the existing function signature — match it.

- [ ] **Step 5: Run test — expect PASS**

Run: `go test ./internal/behaviortree/ -run TestResolveTarget_MostWoundedPackmate -v`
Expected: PASS.

- [ ] **Step 6: Re-run the Task 9 pure_caster tests — they should now also pass**

Run: `go test ./internal/behaviortree/ -run TestPureCaster_PackmateHurt -v`
Expected: PASS (both cases).

- [ ] **Step 7: Commit**

```bash
git add internal/behaviortree/action_cast_best_in_category.go \
        internal/behaviortree/action_cast_best_in_category_packmate_test.go
git commit -m "feat(btree): most_wounded_packmate target resolver

Pack-tactics revamp (Task 10/N). Extends cast_best_in_category's target
resolution with a 'most_wounded_packmate' option that picks the
same-room packmate with the lowest HP ratio. Closes the pure_caster
packmate_hurt handler; its tests now pass end-to-end.

Spec: docs/superpowers/specs/2026-04-22-pack-tactics-revamp-design.md"
```

---

### Task 12: Create `lookout` archetype

**Files:**
- Create: `_datafiles/world/dogmud/behaviors/archetypes/lookout.yaml`
- Create: `internal/behaviortree/lookout_archetype_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/behaviortree/lookout_archetype_test.go`. On `packmate_hurt`, assert the mob first issues `callforhelp`, then on a second tick issues `attack @<attacker>`. Model on Task 6's test.

```go
package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
)

const lookoutYAML = "../../_datafiles/world/dogmud/behaviors/archetypes/lookout.yaml"

func TestLookout_PackmateHurt_FirstTickCallforhelp(t *testing.T) {
	cleanup := LoadArchetypeForTest("lookout", lookoutYAML)
	defer cleanup()

	mob := &mobs.Mob{
		InstanceId:        900,
		BehaviorArchetype: "lookout",
		Character:         characters.Character{RoomId: 1, Health: 50},
	}
	seedCleanup := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{900: mob})
	defer seedCleanup()

	ok := TryMobBehavior(mob.InstanceId, EventContext{
		EventType: "packmate_hurt",
		UserId:    42,
	})
	assert.True(t, ok)
	cmds := MobCommandsForTest(mob.InstanceId)
	assert.Contains(t, cmds, "callforhelp")
	assert.Contains(t, cmds, "attack @42")
}
```

- [ ] **Step 2: Run test — expect FAIL**

Run: `go test ./internal/behaviortree/ -run TestLookout -v`
Expected: FAIL (archetype file doesn't exist).

- [ ] **Step 3: Create the archetype YAML**

Create `_datafiles/world/dogmud/behaviors/archetypes/lookout.yaml`:

```yaml
# lookout archetype
#
# Stationed watcher. When a packmate is hurt (or the mob itself is
# hurt), it first calls for help (bringing in packmates from adjacent
# rooms), then engages the attacker.
#
# Example users: bandit scouts watching the road, city watch sentries,
# posted guards. Distinguished from `generic_fighter` by the
# callforhelp-first behavior.
#
# Spec: docs/superpowers/specs/2026-04-22-pack-tactics-revamp-design.md

tree:
  type: selector
  children:
    - type: sequence
      event: packmate_hurt
      children:
        - type: action
          do: command
          cmd: callforhelp
        - type: action
          do: command
          cmd: "attack @{event.UserId}"

    - type: sequence
      event: mob_hurt
      children:
        - type: action
          do: command
          cmd: callforhelp
        - type: action
          do: command
          cmd: "attack @{event.UserId}"
```

- [ ] **Step 4: Run test — expect PASS**

Run: `go test ./internal/behaviortree/ -run TestLookout -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/behaviors/archetypes/lookout.yaml \
        internal/behaviortree/lookout_archetype_test.go
git commit -m "feat(archetypes): lookout — callforhelp then engage

Pack-tactics revamp (Task 11/N). New archetype for sentries and
road-watchers. On packmate_hurt or mob_hurt, fires callforhelp to
pull in adjacent-room packmates, then engages the attacker.

Spec: docs/superpowers/specs/2026-04-22-pack-tactics-revamp-design.md"
```

---

### Task 13: Create `support_caster` archetype

**Files:**
- Create: `_datafiles/world/dogmud/behaviors/archetypes/support_caster.yaml`
- Create: `internal/behaviortree/support_caster_archetype_test.go`

- [ ] **Step 1: Write the failing test**

Model on Task 9's pure_caster test. Two assertions:
- If a same-room packmate is below 70% HP → cast heal on them.
- If no packmate below 70% → cast beneficial buff on a packmate; if no unbuffed packmates, fall through to engage.

```go
package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
)

const supportCasterYAML = "../../_datafiles/world/dogmud/behaviors/archetypes/support_caster.yaml"

func TestSupportCaster_PackmateHurt_HealsIfWounded(t *testing.T) {
	cleanup := LoadArchetypeForTest("support_caster", supportCasterYAML)
	defer cleanup()

	caster := &mobs.Mob{
		InstanceId:        1000,
		BehaviorArchetype: "support_caster",
		Routine:           "bandit_camp_guard",
		Character:         characters.Character{RoomId: 1, Health: 50},
	}
	caster.Character.HealthMax.Value = 100
	wounded := &mobs.Mob{
		InstanceId: 1001,
		Routine:    "bandit_camp_guard",
		Character:  characters.Character{RoomId: 1, Health: 60},
	}
	wounded.Character.HealthMax.Value = 100 // 60% — under 70% threshold
	cleanup2 := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{1000: caster, 1001: wounded})
	defer cleanup2()

	ok := TryMobBehavior(caster.InstanceId, EventContext{
		EventType: "packmate_hurt",
		UserId:    42,
	})
	assert.True(t, ok)
	cmds := MobCommandsForTest(caster.InstanceId)
	assertHealCastOnPackmate(t, cmds, wounded.InstanceId)
}

func TestSupportCaster_PackmateHurt_NoWounded_EngagesAttacker(t *testing.T) {
	cleanup := LoadArchetypeForTest("support_caster", supportCasterYAML)
	defer cleanup()

	caster := &mobs.Mob{
		InstanceId:        1002,
		BehaviorArchetype: "support_caster",
		Routine:           "bandit_camp_guard",
		Character:         characters.Character{RoomId: 1, Health: 50},
	}
	caster.Character.HealthMax.Value = 100
	healthy := &mobs.Mob{
		InstanceId: 1003,
		Routine:    "bandit_camp_guard",
		Character:  characters.Character{RoomId: 1, Health: 100},
	}
	healthy.Character.HealthMax.Value = 100
	cleanup2 := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{1002: caster, 1003: healthy})
	defer cleanup2()

	ok := TryMobBehavior(caster.InstanceId, EventContext{
		EventType: "packmate_hurt",
		UserId:    42,
	})
	assert.True(t, ok)
	cmds := MobCommandsForTest(caster.InstanceId)
	assert.Contains(t, cmds, "attack @42")
}
```

(`assertHealCastOnPackmate` reuses the helper from Task 9. If that test file isn't in the same package or the helper isn't exported, duplicate the helper here — keep it tightly inlined.)

- [ ] **Step 2: Run tests — expect FAIL**

Run: `go test ./internal/behaviortree/ -run TestSupportCaster -v`
Expected: FAIL.

- [ ] **Step 3: Create the archetype YAML**

Create `_datafiles/world/dogmud/behaviors/archetypes/support_caster.yaml`:

```yaml
# support_caster archetype
#
# Packmate-focused caster. On packmate_hurt (or mob_hurt):
#   1. If a same-room packmate is wounded (<70% HP), heal them.
#   2. Else if a same-room packmate is unbuffed, cast a beneficial
#      buff on them. (Uses buff_friendly category if present.)
#   3. Else engage the attacker.
#
# Pair name with pure_caster: pure = self-focused, support =
# packmate-focused. Applies to healers, supportive priests, and
# any caster narratively tied to keeping its pack alive.
#
# Spec: docs/superpowers/specs/2026-04-22-pack-tactics-revamp-design.md

tree:
  type: selector
  children:
    - type: sequence
      event: packmate_hurt
      children:
        - type: selector
          children:
            # 1. Heal most-wounded packmate if one is below 70% HP
            - type: sequence
              children:
                - type: condition
                  check: packmate_below_hp_ratio
                  threshold: 0.70
                - type: action
                  do: cast_best_in_category
                  category: heal_friendly
                  target: most_wounded_packmate
            # 2. Buff fallback (only fires if no heal fired)
            - type: action
              do: cast_best_in_category
              category: buff_friendly
              target: most_wounded_packmate
            # 3. Engage the attacker if no support action fired
            - type: action
              do: command
              cmd: "attack @{event.UserId}"

    - type: sequence
      event: mob_hurt
      children:
        - type: action
          do: cast_best_in_category
          category: self_defense
          target: self
        - type: action
          do: command
          cmd: "attack @{event.UserId}"
```

Note: this archetype references a `buff_friendly` category. If that category doesn't exist in the spell catalog yet, step 4 will fall back to either the attack command (which is fine for v1) or fail the buff-path subtests. For this revamp, tests only assert heal + attack fallback; buff_friendly emerging empty-categoried is acceptable behavior.

- [ ] **Step 4: Run tests — expect PASS**

Run: `go test ./internal/behaviortree/ -run TestSupportCaster -v`
Expected: PASS (both).

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/behaviors/archetypes/support_caster.yaml \
        internal/behaviortree/support_caster_archetype_test.go
git commit -m "feat(archetypes): support_caster — packmate-focused caster

Pack-tactics revamp (Task 12/N). New archetype paired with pure_caster.
Priority order: heal wounded packmate -> buff packmate -> engage
attacker. 70% HP threshold distinguishes from pure_caster's
opportunistic-only 40% heal.

Spec: docs/superpowers/specs/2026-04-22-pack-tactics-revamp-design.md"
```

---

### Task 14: Create `leader` archetype

**Files:**
- Create: `_datafiles/world/dogmud/behaviors/archetypes/leader.yaml`
- Create: `internal/behaviortree/leader_archetype_test.go`

- [ ] **Step 1: Write the failing test**

On `packmate_hurt`, assert the mob first issues `rally` or `warcry`, then `attack @<attacker>`.

```go
package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
)

const leaderYAML = "../../_datafiles/world/dogmud/behaviors/archetypes/leader.yaml"

func TestLeader_PackmateHurt_RallyThenEngage(t *testing.T) {
	cleanup := LoadArchetypeForTest("leader", leaderYAML)
	defer cleanup()

	mob := &mobs.Mob{
		InstanceId:        1100,
		BehaviorArchetype: "leader",
		Character:         characters.Character{RoomId: 1, Health: 50},
	}
	seedCleanup := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{1100: mob})
	defer seedCleanup()

	ok := TryMobBehavior(mob.InstanceId, EventContext{
		EventType: "packmate_hurt",
		UserId:    42,
	})
	assert.True(t, ok)
	cmds := MobCommandsForTest(mob.InstanceId)
	// rally or warcry — either is acceptable, so match on both
	foundBuff := false
	for _, c := range cmds {
		if c == "rally" || c == "warcry" {
			foundBuff = true
			break
		}
	}
	assert.True(t, foundBuff, "leader should rally or warcry first; got: %v", cmds)
	assert.Contains(t, cmds, "attack @42")
}
```

- [ ] **Step 2: Run test — expect FAIL**

Run: `go test ./internal/behaviortree/ -run TestLeader -v`
Expected: FAIL.

- [ ] **Step 3: Create the archetype YAML**

Create `_datafiles/world/dogmud/behaviors/archetypes/leader.yaml`:

```yaml
# leader archetype
#
# Pack leader. On packmate_hurt: rally or warcry to buff packmates,
# then engage the attacker. Shares most of tank_taunter's offensive
# kit minus the mandatory taunt — the leader commands support rather
# than drawing focus itself.
#
# Applies to bandit chiefs, wolf alphas, sergeant-style mobs.
#
# Spec: docs/superpowers/specs/2026-04-22-pack-tactics-revamp-design.md

tree:
  type: selector
  children:
    - type: sequence
      event: packmate_hurt
      children:
        - type: action
          do: command_best_of
          cmds: [rally, warcry]
        - type: action
          do: command
          cmd: "attack @{event.UserId}"

    - type: sequence
      event: mob_hurt
      children:
        - type: action
          do: command_best_of
          cmds: [rally, warcry]
        - type: action
          do: command
          cmd: "attack @{event.UserId}"
```

`command_best_of` is the existing btree action from the tank-and-generic-archetypes spec (2026-04-21) that self-gates on command readiness and fires the first ready one.

- [ ] **Step 4: Run test — expect PASS**

Run: `go test ./internal/behaviortree/ -run TestLeader -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/behaviors/archetypes/leader.yaml \
        internal/behaviortree/leader_archetype_test.go
git commit -m "feat(archetypes): leader — rally/warcry then engage

Pack-tactics revamp (Task 13/N). New archetype for bandit chiefs,
alphas, sergeants. Rallies or warcries on packmate_hurt (whichever
is ready) to buff packmates, then engages the attacker. No taunt
(distinction from tank_taunter).

Spec: docs/superpowers/specs/2026-04-22-pack-tactics-revamp-design.md"
```

---

### Task 15: Extend `callforhelp` to fire `heard_callforhelp` on adjacent-room routine-matched mobs

**Files:**
- Modify: `internal/mobcommands/callforhelp.go`
- Create: `internal/mobcommands/callforhelp_packmate_test.go`

- [ ] **Step 1: Write the failing test**

The test seeds:
- Caller mob in room 1 with routine "bandit_camp_guard"
- One matching-routine mob in room 2 (adjacent via exit)
- One different-routine mob in room 2

Then asserts that after `CallForHelp` runs, only the matching mob receives a `heard_callforhelp` event.

```go
package mobcommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCallForHelp_FiresHeardCallforhelpOnRoutineMatch(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	// Caller in room 1 with a routine.
	caller := mobs.GetInstance(100) // Skeleton from seed
	require.NotNil(t, caller)
	caller.Routine = "bandit_camp_guard"

	// Two mobs in room 2: one routine-matched, one not.
	matcher := &mobs.Mob{
		InstanceId: 2000,
		Routine:    "bandit_camp_guard",
		Character:  characters.Character{RoomId: 2, Health: 50},
	}
	nonMatcher := &mobs.Mob{
		InstanceId: 2001,
		Routine:    "temple_service",
		Character:  characters.Character{RoomId: 2, Health: 50},
	}
	seed2 := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{
		2000: matcher,
		2001: nonMatcher,
	})
	defer seed2()

	room2 := &rooms.Room{RoomId: 2, Exits: map[string]exit.RoomExit{"south": {RoomId: 1}}}
	rooms.SeedRoomsForTest(map[int]*rooms.Room{2: room2}, nil)
	room2.AddMob(2000)
	room2.AddMob(2001)

	room1 := rooms.LoadRoom(1)
	require.NotNil(t, room1)
	// Ensure room1 has an exit to room2
	if room1.Exits == nil {
		room1.Exits = map[string]exit.RoomExit{}
	}
	room1.Exits["north"] = exit.RoomExit{RoomId: 2}

	// Spy on behaviortree.TryMobBehavior.
	recorded := map[int]string{}
	orig := dispatchEventFn
	t.Cleanup(func() { dispatchEventFn = orig })
	dispatchEventFn = func(instId int, ctx behaviortree.EventContext) bool {
		recorded[instId] = ctx.EventType
		return true
	}

	_, _ = CallForHelp("", caller, room1)

	assert.Equal(t, "heard_callforhelp", recorded[2000], "matching-routine mob should receive heard_callforhelp")
	_, fired := recorded[2001]
	assert.False(t, fired, "non-matching-routine mob should not receive heard_callforhelp")
}
```

(If `dispatchEventFn` doesn't exist in the `mobcommands` package, introduce it at the same time as the production change — a package-level `var dispatchEventFn = behaviortree.TryMobBehavior` that tests can swap.)

- [ ] **Step 2: Run test — expect FAIL**

Run: `go test ./internal/mobcommands/ -run TestCallForHelp_FiresHeardCallforhelp -v`
Expected: FAIL (event not fired, no routine-match logic yet).

- [ ] **Step 3: Modify `callforhelp.go`**

Inside the `for _, nearbyMobInstanceId := range testRoom.GetMobs(...)` loop, after the existing `ConsidersAnAlly` / `mobNameSearch` filters (or replacing them when the match is routine-based), add:

```go
		// Pack-tactics revamp: fire heard_callforhelp on routine-matching
		// mobs in adjacent rooms. Their archetype decides whether to
		// respond (default lookout/generic_fighter: go back + engage).
		if routineMatch(mob, mobInfo) {
			dispatchEventFn(mobInfo.InstanceId, behaviortree.EventContext{
				EventType: "heard_callforhelp",
				// Propagate the caller identity so the receiving archetype
				// can route back via the `go <exitToCaller>` path.
				MobInstanceId: mob.InstanceId,
			})
		}
```

Add the package-level indirection at the top of the file:

```go
var dispatchEventFn = behaviortree.TryMobBehavior
```

And the helper (below the function):

```go
// routineMatch uses the same inclusion rule as mobs.FindPackmatesInRoom
// but applied across rooms (callforhelp already scoped the search to
// adjacent rooms). Returns true if the two mobs share a routine or one
// references the other's routine via RoutineLinks.
func routineMatch(a, b *mobs.Mob) bool {
	if a.Routine != "" && a.Routine == b.Routine {
		return true
	}
	for _, linked := range b.RoutineLinks {
		if linked != "" && linked == a.Routine {
			return true
		}
	}
	for _, linked := range a.RoutineLinks {
		if linked != "" && linked == b.Routine {
			return true
		}
	}
	return false
}
```

(If `mobs.RoutinesMatch` ends up being exported from the mobs package — consider doing it in a refactor step — delete this local copy and call through. For v1, local is fine.)

The existing `ConsidersAnAlly` path + `go <room>` + `attack @<player>` commands may still be preferable for mobs WITHOUT an archetype handler for `heard_callforhelp`. Keep that path for now; the event approach runs *in addition* for routine-matched mobs. A later refactor after content migration can decide whether to remove the legacy branch.

- [ ] **Step 4: Run test — expect PASS**

Run: `go test ./internal/mobcommands/ -run TestCallForHelp_FiresHeardCallforhelp -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/mobcommands/callforhelp.go \
        internal/mobcommands/callforhelp_packmate_test.go
git commit -m "feat(mobcommands): callforhelp fires heard_callforhelp on routine match

Pack-tactics revamp (Task 14/N). Adjacent-room propagation: when a mob
calls for help, routine-matching mobs in adjacent rooms receive a
heard_callforhelp btree event so their archetype can decide the
response (default lookout: go back + engage). Runs in addition to the
existing ConsidersAnAlly legacy path for archetype-less mobs.

Spec: docs/superpowers/specs/2026-04-22-pack-tactics-revamp-design.md"
```

---

### Task 16: Add default `heard_callforhelp` handler to `generic_fighter` and `lookout` archetypes

**Files:**
- Modify: `_datafiles/world/dogmud/behaviors/archetypes/generic_fighter.yaml`
- Modify: `_datafiles/world/dogmud/behaviors/archetypes/lookout.yaml`
- Create: `internal/behaviortree/heard_callforhelp_test.go`

- [ ] **Step 1: Write the failing test**

Assert that a generic_fighter with a `heard_callforhelp` event issues `go` toward the caller's room via the connecting exit, then engages (engages happens naturally on mob_combat_round once it arrives in a fight — the handler itself just needs to issue the `go`).

```go
package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
)

func TestGenericFighter_HeardCallforhelp_IssuesGoCommand(t *testing.T) {
	cleanup := LoadArchetypeForTest("generic_fighter", genericFighterYAML)
	defer cleanup()

	mob := &mobs.Mob{
		InstanceId:        1200,
		BehaviorArchetype: "generic_fighter",
		Character:         characters.Character{RoomId: 2, Health: 50},
	}
	seedCleanup := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{1200: mob})
	defer seedCleanup()

	ok := TryMobBehavior(mob.InstanceId, EventContext{
		EventType:     "heard_callforhelp",
		MobInstanceId: 100, // caller
	})
	assert.True(t, ok)
	cmds := MobCommandsForTest(mob.InstanceId)
	// Engine should infer the connecting exit from the caller's room.
	// Match a "go " prefix loosely.
	foundGo := false
	for _, c := range cmds {
		if len(c) > 3 && c[:3] == "go " {
			foundGo = true
			break
		}
	}
	assert.True(t, foundGo, "heard_callforhelp should issue a go command, got: %v", cmds)
}
```

- [ ] **Step 2: Run test — expect FAIL**

Run: `go test ./internal/behaviortree/ -run TestGenericFighter_HeardCallforhelp -v`
Expected: FAIL.

- [ ] **Step 3: Add the handler to `generic_fighter.yaml`**

Add a new selector child:

```yaml
    # heard_callforhelp: a routine-matched packmate in an adjacent room
    # called for help. Navigate toward the caller; engagement happens
    # automatically once we're in their room (combat loop picks up the
    # active fight).
    - type: sequence
      event: heard_callforhelp
      children:
        - type: action
          do: go_to_caller_room
```

This references a new btree action `go_to_caller_room` that resolves the caller mob's room from the event context and issues a `go <exitName>` command. Add it to `actions_mob.go`:

```go
// actGoToCallerRoom issues a `go <exit>` command toward the room
// containing the event's caller mob (Event.MobInstanceId). If no
// direct exit leads there, fails (falls through to next selector
// child or returns Failure).
func actGoToCallerRoom(params map[string]any, ctx *EvalContext) Result {
	self := mobs.GetInstance(ctx.InstanceId)
	if self == nil {
		return Failure
	}
	caller := mobs.GetInstance(ctx.Event.MobInstanceId)
	if caller == nil {
		return Failure
	}
	myRoom := rooms.LoadRoom(self.Character.RoomId)
	if myRoom == nil {
		return Failure
	}
	for exitName, info := range myRoom.Exits {
		if info.RoomId == caller.Character.RoomId {
			self.Command(fmt.Sprintf("go %s", exitName))
			return Success
		}
	}
	return Failure
}
```

Register in the actions map alongside existing entries.

- [ ] **Step 4: Run test — expect PASS**

Run: `go test ./internal/behaviortree/ -run TestGenericFighter_HeardCallforhelp -v`
Expected: PASS.

- [ ] **Step 5: Mirror the handler in `lookout.yaml`**

Add the same sequence to `lookout.yaml`'s root selector children (copy-paste is fine — keep archetype files self-contained).

- [ ] **Step 6: Run full behaviortree suite**

Run: `go test ./internal/behaviortree/`
Expected: `ok`.

- [ ] **Step 7: Commit**

```bash
git add _datafiles/world/dogmud/behaviors/archetypes/generic_fighter.yaml \
        _datafiles/world/dogmud/behaviors/archetypes/lookout.yaml \
        internal/behaviortree/actions_mob.go \
        internal/behaviortree/heard_callforhelp_test.go
git commit -m "feat(archetypes): heard_callforhelp handler + go_to_caller_room action

Pack-tactics revamp (Task 15/N). generic_fighter and lookout gain
default heard_callforhelp handlers that navigate to the caller's
room; new go_to_caller_room btree action resolves the connecting
exit from the event context.

Spec: docs/superpowers/specs/2026-04-22-pack-tactics-revamp-design.md"
```

---

### Task 17: Content migration — North Road bandits

**Files:**
- Modify: `_datafiles/world/dogmud/mobs/north_road/284-bandit_fighter.yaml`
- Modify: `_datafiles/world/dogmud/mobs/north_road/285-bandit_caster.yaml`
- Modify: `_datafiles/world/dogmud/mobs/north_road/286-soren.yaml`
- Modify: `_datafiles/world/dogmud/mobs/north_road/283-bandit_lookout.yaml` (if this file exists — verify with ls)
- Modify: `_datafiles/world/dogmud/mobs/north_road/287-bloodline_agent.yaml` (stays unarchetyped + no routine — solo mob)

- [ ] **Step 1: Inspect each mob's current fields**

Run: `ls _datafiles/world/dogmud/mobs/north_road/`
Then `cat` each one to confirm they don't already have routine/archetype fields.

- [ ] **Step 2: Assign routines + archetypes**

Add the following fields (near the top of each YAML, after `zone:`):

**284-bandit_fighter.yaml:**
```yaml
behavior_archetype: generic_fighter
routine: bandit_camp_guard
```

**285-bandit_caster.yaml:**
```yaml
behavior_archetype: support_caster
routine: bandit_camp_guard
```

**286-soren.yaml** (the bandit leader):
```yaml
behavior_archetype: leader
routine: bandit_camp_guard
```

**283-bandit_lookout.yaml** (if present — the mob watching from the road):
```yaml
behavior_archetype: lookout
routine: watch_north_road
routine_links:
  - bandit_camp_guard
```

**287-bloodline_agent.yaml** — no changes. Solo mob, no pack, hostile:false already.

- [ ] **Step 3: Rebuild to confirm YAML parses**

Run: `go build ./...`
Expected: `ok`.

- [ ] **Step 4: Run mob-loading tests (regression)**

Run: `go test ./internal/mobs/`
Expected: `ok`.

- [ ] **Step 5: Delete any stale instance saves for these mobs**

Run: `rm -f _datafiles/world/dogmud/mobs.instances/summons/28{3,4,5,6,7}-*.yaml`
(gitignored, but clears any dev-local instance state; harmless if none exist.)

- [ ] **Step 6: Commit**

```bash
git add _datafiles/world/dogmud/mobs/north_road/
git commit -m "content(north_road): routines + archetypes for bandit pack

Pack-tactics revamp (Task 16/N). Bandit fighter/caster/leader at the
camp all route to bandit_camp_guard. Bandit lookout on the road uses
watch_north_road with a link back to bandit_camp_guard so
callforhelp flows in both directions. Bloodline agent stays solo
(no routine, no archetype).

Spec: docs/superpowers/specs/2026-04-22-pack-tactics-revamp-design.md"
```

---

### Task 18: Content migration — Ironwind Steppe

**Files:**
- Modify: every `_datafiles/world/dogmud/mobs/ironwind_steppe/*.yaml` that represents a mob that should form packs (goblins, wolves, etc.). Skip solo/boss mobs.

- [ ] **Step 1: Audit the zone's mobs**

Run: `ls _datafiles/world/dogmud/mobs/ironwind_steppe/` and `grep -H "^name:\|^groups:\|^hostile:" _datafiles/world/dogmud/mobs/ironwind_steppe/*.yaml` to classify each.

- [ ] **Step 2: Assign routines**

For each goblin mob in a shared spawn context, add:
```yaml
behavior_archetype: generic_fighter
routine: goblin_warband
```

For wolves:
```yaml
behavior_archetype: generic_fighter
routine: wolf_pack_ironwind
```

For deep_gnawer / scrapper / warband-leader type mobs whose narrative role is "leader", use `behavior_archetype: leader` and the same routine.

For scouts that should `callforhelp` first, use `behavior_archetype: lookout` + a `watch_*` routine with `routine_links` pointing at the main warband.

- [ ] **Step 3: Rebuild + test + commit**

```bash
go build ./...
go test ./internal/mobs/
git add _datafiles/world/dogmud/mobs/ironwind_steppe/
git commit -m "content(ironwind_steppe): routines + archetypes for goblin + wolf packs

Pack-tactics revamp (Task 17/N).

Spec: docs/superpowers/specs/2026-04-22-pack-tactics-revamp-design.md"
```

---

### Task 19: Content migration — instanced zones (arena, planar oasis)

**Files:**
- Modify: `_datafiles/world/dogmud/mobs/instance_arena/*.yaml`
- Modify: `_datafiles/world/dogmud/mobs/instance_planar_oasis/*.yaml`

- [ ] **Step 1: Identify existing archetypes in these zones**

The Phase 4 work already assigned archetypes to most instanced mobs (vampire, elementals, golems). The only change needed here is adding **routine** (and optional links) to the mobs that should form packs.

Run: `grep -H "behavior_archetype:" _datafiles/world/dogmud/mobs/instance_arena/*.yaml _datafiles/world/dogmud/mobs/instance_planar_oasis/*.yaml`

- [ ] **Step 2: Assign routines**

For arena champion + arena loot mobs that appear together, use `routine: arena_gauntlet_<wave>` (different wave = different routine so waves don't cross-aggro).

For planar elementals of the same element, use `routine: plane_<element>` (e.g. `plane_air`, `plane_earth`).

For solo boss-tier mobs (elemental_king, phantom, arena_champion if solo), skip routine — they're meant to fight alone.

- [ ] **Step 3: Rebuild + test + commit**

```bash
go build ./...
go test ./internal/mobs/
git add _datafiles/world/dogmud/mobs/instance_arena/ \
        _datafiles/world/dogmud/mobs/instance_planar_oasis/
git commit -m "content(instances): routines for arena + planar oasis packs

Pack-tactics revamp (Task 18/N). Wave-scoped routines for arena mobs,
element-scoped routines for planar_oasis elementals. Boss-tier solo
mobs stay routine-less.

Spec: docs/superpowers/specs/2026-04-22-pack-tactics-revamp-design.md"
```

---

### Task 20: Content migration — Dustwalk Road bandits

**Files:**
- Modify: `_datafiles/world/dogmud/mobs/dustwalk_road/*.yaml` — bandit mobs only. Skip tessara (road warden, neutral) + any merchants.

- [ ] **Step 1: Audit the zone**

Run: `grep -H "^name:\|^groups:" _datafiles/world/dogmud/mobs/dustwalk_road/*.yaml` and identify bandit mobs vs civilians/NPCs.

- [ ] **Step 2: Assign routines**

For dustwalk bandits:
```yaml
behavior_archetype: generic_fighter
routine: dustwalk_bandit_camp
```

For any bandit leader in the zone:
```yaml
behavior_archetype: leader
routine: dustwalk_bandit_camp
```

Road warden Tessara (83): no routine, no archetype. She's a quest NPC.

- [ ] **Step 3: Rebuild + test + commit**

```bash
go build ./...
go test ./internal/mobs/
git add _datafiles/world/dogmud/mobs/dustwalk_road/
git commit -m "content(dustwalk_road): routines + archetypes for bandit camp

Pack-tactics revamp (Task 19/N).

Spec: docs/superpowers/specs/2026-04-22-pack-tactics-revamp-design.md"
```

---

### Task 21: Delete `MakeHostile`, `IsHostile`, `ReduceHostility`, and their tests

**Files:**
- Modify: `internal/mobs/mobs.go` — delete `MakeHostile`, `IsHostile`, `ReduceHostility`, `mobsHatePlayers`, `mobsHatePlayersMu`
- Modify: `internal/mobs/memory.go` — delete the `mobsHatePlayers` reporting block (lines 20–22)
- Modify: `internal/hooks/NewRound_MobRoundTick.go` — delete the `mobs.ReduceHostility()` call at line 46
- Modify: `internal/mobs/mobs_test.go` — delete `TestMakeHostileAndIsHostile` + `TestReduceHostility`
- Modify: `internal/mobcommands/lookfortrouble.go` — delete the group-hostility branch (lines ~117–130)

- [ ] **Step 1: Check for any remaining callers of the soon-to-be-deleted symbols**

Run: `git grep -n "MakeHostile\|IsHostile\|ReduceHostility\|mobsHatePlayers" -- internal/`
Expected: only definitions (mobs.go, memory.go, NewRound_MobRoundTick.go), tests, and the branch in lookfortrouble.go. **No other production consumers.** If any others surface, fold those deletions into this task.

- [ ] **Step 2: Delete the group-hostility branch in `lookfortrouble.go`**

In `internal/mobcommands/lookfortrouble.go`, find the block (originally lines 117–130):

```go
			// Hostility default to 5 minutes
			for _, groupName := range mob.Groups {
				// Does this group hate this player?
				if mobs.IsHostile(groupName, playerId) {

					allPotentialTargets = append(allPotentialTargets, playerId)

					if !ignoreUser {
						for i := 0; i < entries; i++ {
							nonDownedUserTargets = append(nonDownedUserTargets, playerId)
						}
					}
					break
				}
			}
```

Delete it entirely. Keep the `hostile:true` branch, `HatesSpecies` branch, and `HatesMob` branch that precede/follow. Also keep Fix A's `NoAggroTarget` early-continue intact.

- [ ] **Step 3: Delete `mobs.ReduceHostility()` call**

In `internal/hooks/NewRound_MobRoundTick.go`, delete line 46 (`mobs.ReduceHostility()`).

- [ ] **Step 4: Delete the `mobsHatePlayers` memory reporting**

In `internal/mobs/memory.go`, delete lines 20–22:
```go
	mobsHatePlayersMu.RLock()
	ret["mobsHatePlayers"] = util.MemoryResult{Memory: util.MemoryUsage(mobsHatePlayers), Count: len(mobsHatePlayers)}
	mobsHatePlayersMu.RUnlock()
```

- [ ] **Step 5: Delete `MakeHostile`, `IsHostile`, `ReduceHostility` + state in `mobs.go`**

Delete:
- The `mobsHatePlayers` package-level `var` (around line 38) and its mutex.
- The `ReduceHostility` function (lines 999–1016).
- The `IsHostile` function (lines 1018–1031).
- The `MakeHostile` function (lines 1033–1045).

- [ ] **Step 6: Delete the tests that exercise these**

In `internal/mobs/mobs_test.go`, delete:
- `TestMakeHostileAndIsHostile`
- `TestReduceHostility`
- Any other tests that reference the removed symbols (a `go test` compile error will flag them).

- [ ] **Step 7: Build to confirm nothing else references the deleted symbols**

Run: `go build ./...`
Expected: `ok` — no undefined-symbol errors.

- [ ] **Step 8: Run the full test suite**

Run: `go test ./...`
Expected: `ok` across all packages.

- [ ] **Step 9: Commit**

```bash
git add -A
git commit -m "refactor: delete mobs.MakeHostile/IsHostile/ReduceHostility + state

Pack-tactics revamp (Task 20/N). The replacement is in place:
dispatchPackmateHurt routes packmate_hurt to same-room routine-matched
mobs, and archetypes handle the reaction. The old broad-group hostility
propagation is no longer needed.

Removes:
- mobs.MakeHostile, IsHostile, ReduceHostility
- mobsHatePlayers state map + mutex
- memory.go mobsHatePlayers reporting
- NewRound_MobRoundTick.go ReduceHostility() tick call
- Group-hostility branch in LookForTrouble (Fix A's NoAggroTarget
  early-continue stays)
- TestMakeHostileAndIsHostile, TestReduceHostility

Spec: docs/superpowers/specs/2026-04-22-pack-tactics-revamp-design.md"
```

---

### Task 22: Full regression check + in-game smoke

**Files:**
- None (verification + merge)

- [ ] **Step 1: Run full test suite**

Run: `go test ./...`
Expected: every package `ok`, no `FAIL`. Specifically confirm:
- `internal/mobs/` passes without the hostility tests
- `internal/hooks/TestLookForTrouble_SkipsGraceProtectedPlayer` (Fix A) still passes
- `internal/hooks/TestDispatchPackmateHurt_FiresOnEachPackmate` passes
- All archetype tests pass
- All pack_roaming tests pass unchanged

- [ ] **Step 2: Full build**

Run: `go build ./...`
Expected: `ok`.

- [ ] **Step 3: In-game smoke test (human player)**

Using quester9 (15.yaml) as the test character, verify these scenarios hands-on:

1. **Primary bug fix (Olen regression):**
   - Go to North Road. Attack a bandit. Die. Respawn in Thornwall.
   - Expected: temple priest Olen does **not** engage. Quester9 can walk away.

2. **Pack reaction at North Road:**
   - Approach the bandit camp. Attack one bandit.
   - Expected: remaining bandits at the camp engage on the same or next round.
   - Expected: bandit lookout (on the road, if present) walks in via `callforhelp` propagation.

3. **No false positives on civilians:**
   - Walk into the Temple District alone (no combat). Olen idles normally (conviction-ward, idle emotes).
   - Attack Olen directly. He retaliates (defender-aggro still works, that's unchanged).
   - Other priests in the temple (if any) do NOT engage from Olen's fight — they have no matching routine.

4. **Pack roaming unchanged:**
   - Observe goblins or wolves moving through the steppe. Pack alpha-follow still works (`pack_roaming.go` logic untouched).

5. **Respawn grace still holds:**
   - Die to a pack mob. Respawn. No mob engages for ~3 rounds (Fix A's grace buff still active; nobody should re-target).

- [ ] **Step 4: Merge to development**

```bash
git checkout development
git merge --no-ff feature/pack-tactics-revamp -m "Merge feature/pack-tactics-revamp into development

Replaces mobs.MakeHostile / IsHostile hostility-propagation hack with
a routine-scoped pack reaction system driven by btree archetypes.

- New Routine / RoutineLinks fields on mobs.Mob
- FindPackmatesInRoom + dispatchPackmateHurt infrastructure
- packmate_hurt event + handlers in generic_fighter, tank_taunter,
  melee_self_buff, pure_caster
- New archetypes: lookout, support_caster, leader
- Adjacent-room heard_callforhelp event; go_to_caller_room btree
  action for default response
- Content migration for north_road, ironwind_steppe, instanced
  zones, dustwalk_road
- Deletion of MakeHostile / IsHostile / ReduceHostility / state

Spec: docs/superpowers/specs/2026-04-22-pack-tactics-revamp-design.md"
```

- [ ] **Step 5: Update memory + patch notes**

- Delete `project_pack_tactics_revamp.md` (resolved).
- Update `MEMORY.md` — add a `Completed (2026-04-22)` bullet for the revamp; remove the Future Work entry.
- Add a PATCH_NOTES.md entry at the top describing the user-visible changes (no more "Olen aggros respawning player"; pack mobs now react tightly; no broad-group aggro bleed-through).
- Commit docs changes.

```bash
git add PATCH_NOTES.md \
        "C:/Users/Calabe Davis/.claude/projects/C--Users-Calabe-Davis-workspace-DOGMud/memory/MEMORY.md"
rm "C:/Users/Calabe Davis/.claude/projects/C--Users-Calabe-Davis-workspace-DOGMud/memory/project_pack_tactics_revamp.md"
git commit -m "docs: patch notes + memory update for pack-tactics revamp"
```

---

## Self-Review

**1. Spec coverage:**
- ✅ `Routine` / `RoutineLinks` fields on Mob — Task 2
- ✅ `FindPackmatesInRoom` — Task 3
- ✅ `dispatchPackmateHurt` — Task 4
- ✅ Wiring into `handleAggroAndAssist` — Task 5
- ✅ Archetype handlers for existing archetypes (generic_fighter, tank_taunter, melee_self_buff, pure_caster) — Tasks 6–9 + 10 + 11
- ✅ New archetypes (lookout, support_caster, leader) — Tasks 12–14
- ✅ Adjacent-room `heard_callforhelp` propagation — Task 15
- ✅ `heard_callforhelp` handler + `go_to_caller_room` action — Task 16
- ✅ Content migration (north_road, ironwind_steppe, instanced zones, dustwalk_road) — Tasks 17–20
- ✅ Deletion of `MakeHostile` / `IsHostile` / `ReduceHostility` / state / branch in `LookForTrouble` — Task 21
- ✅ Full regression + smoke + merge — Task 22

**Spec decision 7 (archetype name load-time validation):** spec says add it "if not present from Companion Phase 4". Existing behavior is lazy warn-on-first-use (`helpers.go:169`), not boot-time hard-fail. I did NOT add a separate task for tightening this to hard-fail validation. Rationale: existing warn-on-use is adequate for typo-catching (warning is logged on first resolution attempt), and adding hard-fail validation is a separate stylistic choice that doesn't affect the revamp's correctness. If the user wants hard-fail later, it's a 10-line task added to `main.go`'s boot sequence.

**2. Placeholder scan:** The plan does reference `LoadArchetypeForTest` and `MobCommandsForTest` test helpers that I haven't verified exist. Tasks 6, 7, 8, 9, 12, 13, 14, 16 note "reuse whatever helper the existing melee_self_buff test uses — do not invent new infrastructure." That's not a TBD, it's a directive to the implementer to find and reuse existing helpers. Acceptable.

Task 9's test file uses hand-rolled `strContains` / `formatInt` helpers as a fallback for imports; the task note explicitly says "if the package already imports strings, use strings.Contains + strconv.Itoa directly — simpler." Acceptable (implementer picks the cleaner path).

**3. Type consistency:** `dispatchPackmateHurt(victim, attackerUserId, attackerMobInstanceId)` is used consistently across Tasks 4 + 5. `routineMatch` / `routinesMatch` naming is the same in both mobs/packmates.go (Task 3) and mobcommands/callforhelp.go (Task 15 — local copy). `packmate_hurt` event name is consistent everywhere.

**4. Ambiguity check:** Task 15's note that the legacy `ConsidersAnAlly` + `go` + `attack` path runs "in addition to" the new event path is explicit. Task 21 doesn't re-delete the legacy path — that's a follow-up cleanup post-migration once we're confident all pack-forming mobs have archetypes. Flagged in the Task 15 commit message. No other ambiguity.

Plan complete and saved to `docs/superpowers/plans/2026-04-22-pack-tactics-revamp.md`.
