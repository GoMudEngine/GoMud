# Mutation-Triggered Archetype Shift Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Mobs that acquire a mutation with an `archetype_pull` shift their behavior archetype mid-fight (rarest pull wins), with baseline spellbook parity so caster shifts always work.

**Architecture:** A new optional `archetype_pull` field on mutation YAML, boot-validated in `behaviortree`. `behaviortree.ReevaluateArchetypeShift(mob)` is called from the mob mutation-acquisition hook; it gates on eligibility (FROM set + no per-mob btree), picks the rarest owned pull, and performs the swap (archetype field, btree state reset, policy re-derive, additive goals merge, room flavor). Instance saves persist a shifted archetype. All mobs get the player starter spellbook merged under authored spells at `Validate()`.

**Tech Stack:** Go, yaml.v2, existing test helpers (`mutations.SeedMutationsForTest`, `mobs.SetInstanceForTest`, `rooms.SeedRoomsForTest`, behaviortree's `buildMutationMob`/`seedMutationTestRoom`).

**Spec:** `docs/superpowers/specs/2026-07-10-mutation-archetype-shift-design.md`

## File map

| File | Change |
|---|---|
| `internal/mutations/mutations.go` | `ArchetypePull` field on `MutationSpec`; `AllSpecs()` accessor |
| `internal/characters/character.go` | Exported `StarterSpells`; `New()` builds spellbook from it |
| `internal/mobs/mobs.go` | Spellbook parity seeding in `Mob.Validate()`; instance-restore of shifted archetype |
| `internal/goals/store.go` | Exported `MergeArchetypeDefaults` wrapper |
| `internal/behaviortree/archetype_shift.go` (new) | FROM/TO sets, pull validation, `ReevaluateArchetypeShift` |
| `internal/hooks/NewRound_MobRoundTick.go` | Extract `applyAcquiredMutation`; call the shift |
| `internal/mobs/instance_save.go` | Persist `behavior_archetype` when shifted |
| `main.go` | Wire `behaviortree.ValidateArchetypePulls()` |
| `_datafiles/world/dogmud/mutations/*.yaml` | 10 `archetype_pull` entries |

---

### Task 1: `ArchetypePull` field + `AllSpecs()` accessor (mutations)

**Files:**
- Modify: `internal/mutations/mutations.go` (struct ~line 29-54; accessor near `GetMutation` ~line 126)
- Test: `internal/mutations/archetype_pull_test.go` (new)

- [ ] **Step 1: Write the failing test**

```go
package mutations

import (
	"testing"

	"gopkg.in/yaml.v2"
)

func TestArchetypePullYAMLRoundtrip(t *testing.T) {
	src := "mutationid: test-pull\nname: Test Pull\nrarity: 5\narchetype_pull: generic_fighter\n"
	var m MutationSpec
	if err := yaml.Unmarshal([]byte(src), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m.ArchetypePull != "generic_fighter" {
		t.Fatalf("ArchetypePull = %q, want %q", m.ArchetypePull, "generic_fighter")
	}
}

func TestAllSpecsReturnsSeeded(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"a": {MutationId: "a", Rarity: 3},
		"b": {MutationId: "b", Rarity: 7},
	})
	defer cleanup()

	specs := AllSpecs()
	if len(specs) != 2 {
		t.Fatalf("AllSpecs len = %d, want 2", len(specs))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/mutations/ -run "TestArchetypePull|TestAllSpecs" -v`
Expected: compile FAIL — `m.ArchetypePull undefined` and `undefined: AllSpecs`

- [ ] **Step 3: Implement**

In `MutationSpec`, after the `RequiresBodyParts` field:

```go
	// ArchetypePull optionally names a behavior archetype this mutation
	// pulls its owner toward. When a MOB acquires a pull-mutation, the
	// archetype-shift path (behaviortree.ReevaluateArchetypeShift) may
	// re-archetype it. Validated at boot by
	// behaviortree.ValidateArchetypePulls (whitelist + file existence);
	// this package only carries the string. Players are unaffected.
	//
	// PROVISIONAL CONTENT: the pull table is expected to be re-curated
	// when the mutation-graph redesign lands (see the 2026-07-10
	// mutation-archetype-shift design doc).
	ArchetypePull string `yaml:"archetype_pull,omitempty"`
```

After `GetMutation` (~line 126):

```go
// AllSpecs returns every loaded mutation spec. Read-only convenience
// for boot-time validation passes (e.g. archetype_pull validation);
// callers must not mutate the returned specs.
func AllSpecs() []*MutationSpec {
	out := make([]*MutationSpec, 0, len(allMutations))
	for _, spec := range allMutations {
		out = append(out, spec)
	}
	return out
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/mutations/ -run "TestArchetypePull|TestAllSpecs" -v`
Expected: PASS (both)

- [ ] **Step 5: Commit**

```bash
git add internal/mutations/mutations.go internal/mutations/archetype_pull_test.go
git commit -m "feat(mutations): archetype_pull field + AllSpecs accessor"
```

---

### Task 2: Shared `StarterSpells` baseline (characters)

**Files:**
- Modify: `internal/characters/character.go` (~line 316-322, the `SpellBook:` literal in `New()`)
- Test: `internal/characters/starter_spells_test.go` (new)

- [ ] **Step 1: Write the failing test**

```go
package characters

import "testing"

// TestStarterSpellsSeededOnNew guards the shared baseline: New()
// (player creation) and mobs.Mob.Validate() (actor parity) must both
// derive from StarterSpells, so a change to the list propagates to both.
func TestStarterSpellsSeededOnNew(t *testing.T) {
	want := []string{"conviction-spike", "chrysalis-glow", "identify"}
	if len(StarterSpells) != len(want) {
		t.Fatalf("StarterSpells = %v, want %v", StarterSpells, want)
	}
	for i, id := range want {
		if StarterSpells[i] != id {
			t.Fatalf("StarterSpells[%d] = %q, want %q", i, StarterSpells[i], id)
		}
	}

	c := New()
	for _, id := range StarterSpells {
		if c.SpellBook[id] != 1 {
			t.Errorf("New().SpellBook[%q] = %d, want 1", id, c.SpellBook[id])
		}
	}
}
```

Note: check `New()`'s actual signature at `internal/characters/character.go` (~line 300) before writing — if it takes arguments, mirror an existing call site from `characters` package tests.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/characters/ -run TestStarterSpellsSeededOnNew -v`
Expected: compile FAIL — `undefined: StarterSpells`

- [ ] **Step 3: Implement**

Above `New()` in `character.go`:

```go
// StarterSpells is the baseline spellbook every new player receives at
// creation AND every mob receives at Validate() (actor parity — added
// 2026-07-10 so mutation-driven archetype shifts into caster roles
// always have something to cast). Values seed at 1 (fresh-player
// proficiency) and never overwrite authored entries.
var StarterSpells = []string{
	"conviction-spike", // starting attack spell
	"chrysalis-glow",   // light source for caves
	"identify",         // inspect item properties
}

// starterSpellbook builds a fresh SpellBook map from StarterSpells.
func starterSpellbook() map[string]int {
	sb := make(map[string]int, len(StarterSpells))
	for _, id := range StarterSpells {
		sb[id] = 1
	}
	return sb
}
```

Replace the inline literal in `New()`:

```go
		// Starting spells — attack, utility light for dark zones, and
		// basic item inspection so new players can evaluate drops.
		SpellBook: starterSpellbook(),
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/characters/ -run TestStarterSpellsSeededOnNew -v && go test ./internal/characters/ 2>&1 | tail -2`
Expected: PASS; full package still green

- [ ] **Step 5: Commit**

```bash
git add internal/characters/character.go internal/characters/starter_spells_test.go
git commit -m "refactor(characters): extract StarterSpells shared baseline"
```

---

### Task 3: Baseline spellbook parity for mobs

**Files:**
- Modify: `internal/mobs/mobs.go` — inside `func (r *Mob) Validate() error` (~line 1066), just before `r.Character.Validate()` (~line 1088)
- Test: `internal/mobs/spellbook_parity_test.go` (new)

- [ ] **Step 1: Write the failing test**

```go
package mobs

import (
	"testing"

	"gopkg.in/yaml.v2"

	"github.com/GoMudEngine/GoMud/internal/characters"
)

// TestSpellbookParitySeeding: every mob gets the player StarterSpells
// merged UNDER any authored spellbook at Validate() — authored entries
// are never modified, baseline entries seed at 1.
func TestSpellbookParitySeeding(t *testing.T) {
	// Case 1: no authored spellbook → all baseline spells at 1.
	var plain Mob
	if err := yaml.Unmarshal([]byte("mobid: 991\ncharacter:\n  name: plainprobe\n"), &plain); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	plain.Validate()
	for _, id := range characters.StarterSpells {
		if plain.Character.SpellBook[id] != 1 {
			t.Errorf("plain mob SpellBook[%q] = %d, want 1", id, plain.Character.SpellBook[id])
		}
	}

	// Case 2: authored spellbook preserved, baseline merged under it.
	src := "mobid: 992\ncharacter:\n  name: casterprobe\n  spellbook:\n    nerve-disruption: 50\n    conviction-spike: 25\n"
	var caster Mob
	if err := yaml.Unmarshal([]byte(src), &caster); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	caster.Validate()
	if caster.Character.SpellBook["nerve-disruption"] != 50 {
		t.Errorf("authored nerve-disruption = %d, want 50 (untouched)", caster.Character.SpellBook["nerve-disruption"])
	}
	if caster.Character.SpellBook["conviction-spike"] != 25 {
		t.Errorf("authored conviction-spike = %d, want 25 (never overwritten by baseline)", caster.Character.SpellBook["conviction-spike"])
	}
	if caster.Character.SpellBook["identify"] != 1 {
		t.Errorf("baseline identify = %d, want 1 (merged)", caster.Character.SpellBook["identify"])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/mobs/ -run TestSpellbookParitySeeding -v`
Expected: FAIL — `plain mob SpellBook["conviction-spike"] = 0, want 1`

- [ ] **Step 3: Implement**

In `(r *Mob) Validate()`, directly before the `r.Character.Validate()` call (~line 1088):

```go
	// Actor parity (2026-07-10): every mob carries the player baseline
	// spellbook so mutation-driven shifts into caster archetypes always
	// have something to cast. Baseline merges UNDER authored spellbooks —
	// an authored entry is never modified; missing entries seed at 1
	// (fresh-player proficiency). Inert for non-caster btrees.
	if r.Character.SpellBook == nil {
		r.Character.SpellBook = make(map[string]int, len(characters.StarterSpells))
	}
	for _, spellId := range characters.StarterSpells {
		if _, ok := r.Character.SpellBook[spellId]; !ok {
			r.Character.SpellBook[spellId] = 1
		}
	}
```

(`internal/mobs/mobs.go` already imports `characters`.)

- [ ] **Step 4: Run tests**

Run: `go test ./internal/mobs/ -run TestSpellbookParitySeeding -v && go test ./internal/mobs/ 2>&1 | tail -2`
Expected: PASS; full package green

- [ ] **Step 5: Commit**

```bash
git add internal/mobs/mobs.go internal/mobs/spellbook_parity_test.go
git commit -m "feat(mobs): merge player baseline spellbook into every mob at Validate"
```

---

### Task 4: Exported `goals.MergeArchetypeDefaults`

**Files:**
- Modify: `internal/goals/store.go` (after `loadOrLazyInit`, ~line 68)
- Test: `internal/goals/merge_defaults_test.go` (new)

- [ ] **Step 1: Write the failing test**

Note: `internal/goals/test_main_test.go` already redirects persistence for the whole package — no per-test dir setup needed. Use unique template ids to avoid cache collisions.

```go
package goals

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// TestMergeArchetypeDefaults: the exported wrapper additively merges the
// mob's CURRENT archetype's defaults — used by the archetype-shift path
// after swapping BehaviorArchetype. Existing goals survive.
func TestMergeArchetypeDefaults(t *testing.T) {
	const mobId = 99801
	const name = "mergeprobe"

	SetArchetypeDefaultsLookup(func(m *mobs.Mob) []GoalDefault {
		if m == nil {
			return nil
		}
		switch m.BehaviorArchetype {
		case "tank_taunter":
			return []GoalDefault{{Type: "survival", Priority: 90}}
		}
		return nil
	})
	t.Cleanup(func() { SetArchetypeDefaultsLookup(nil) })

	// Pre-existing goal of a different type.
	if _, err := Add(mobId, name, &Goal{Type: "wealth", Priority: 50}); err != nil {
		t.Fatalf("Add(wealth): %v", err)
	}

	m := &mobs.Mob{MobId: mobs.MobId(mobId), BehaviorArchetype: "tank_taunter"}
	m.Character.Name = name
	MergeArchetypeDefaults(mobId, name, m)

	types := map[string]bool{}
	for _, g := range GoalsOf(mobId, name) {
		types[g.Type] = true
	}
	if !types["wealth"] {
		t.Error("pre-existing wealth goal was lost by the merge")
	}
	if !types["survival"] {
		t.Error("archetype default survival goal was not merged in")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/goals/ -run TestMergeArchetypeDefaults -v`
Expected: compile FAIL — `undefined: MergeArchetypeDefaults`

- [ ] **Step 3: Implement**

In `store.go`, after `loadOrLazyInit`:

```go
// MergeArchetypeDefaults additively merges the mob's CURRENT archetype's
// default goals into its goal list. Used by behaviortree's mutation-
// driven archetype shift (2026-07-10) after swapping BehaviorArchetype,
// so a re-archetyped mob picks up its new role's defaults without losing
// learned/reactive goals. Thin exported wrapper over the 5.3 merge-seed
// path: idempotent, additive-only, admin-set goals at >= priority block
// individual merges via the normal Add conflict logic.
func MergeArchetypeDefaults(templateId int, namesimple string, mob *mobs.Mob) {
	mg := loadOrLazyInit(templateId, namesimple)
	mergeSeedFromArchetype(templateId, namesimple, mg, mob)
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/goals/ -run TestMergeArchetypeDefaults -v && go test ./internal/goals/ 2>&1 | tail -2`
Expected: PASS; full package green

- [ ] **Step 5: Commit**

```bash
git add internal/goals/store.go internal/goals/merge_defaults_test.go
git commit -m "feat(goals): exported MergeArchetypeDefaults for archetype shifts"
```

---

### Task 5: Shift sets + `ValidateArchetypePulls` (behaviortree)

**Files:**
- Create: `internal/behaviortree/archetype_shift.go`
- Test: `internal/behaviortree/archetype_shift_test.go` (new)

- [ ] **Step 1: Write the failing test**

```go
package behaviortree

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mutations"
)

func TestValidateArchetypePullsCore(t *testing.T) {
	exists := func(path string) bool {
		return strings.Contains(path, "generic_fighter") || strings.Contains(path, "predator")
	}

	// Valid pulls: whitelisted + file exists.
	good := []*mutations.MutationSpec{
		{MutationId: "no-pull", Rarity: 3},
		{MutationId: "brawn", Rarity: 9, ArchetypePull: "generic_fighter"},
		{MutationId: "fangs", Rarity: 5, ArchetypePull: "predator"},
	}
	if err := validateArchetypePulls(good, exists); err != nil {
		t.Fatalf("valid pulls: unexpected error %v", err)
	}

	// Non-whitelisted target (boss archetype).
	bad := []*mutations.MutationSpec{
		{MutationId: "hubris", Rarity: 9, ArchetypePull: "boss_soren"},
	}
	if err := validateArchetypePulls(bad, exists); err == nil {
		t.Fatal("non-whitelisted pull: expected error, got nil")
	}

	// Whitelisted but no archetype file on disk.
	missing := []*mutations.MutationSpec{
		{MutationId: "ghost", Rarity: 9, ArchetypePull: "pure_caster"},
	}
	if err := validateArchetypePulls(missing, exists); err == nil {
		t.Fatal("missing archetype file: expected error, got nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/behaviortree/ -run TestValidateArchetypePullsCore -v`
Expected: compile FAIL — `undefined: validateArchetypePulls`

- [ ] **Step 3: Implement**

Create `internal/behaviortree/archetype_shift.go`:

```go
package behaviortree

// archetype_shift.go — mutation-driven archetype shift (2026-07-10).
//
// Design: docs/superpowers/specs/2026-07-10-mutation-archetype-shift-design.md
// Mobs that acquire a mutation carrying an archetype_pull may re-archetype
// mid-fight. FROM set protects authored behavior; TO whitelist is what any
// mob can credibly play. Pull table is PROVISIONAL pending the mutation-
// graph redesign.

import (
	"fmt"
	"os"

	"github.com/GoMudEngine/GoMud/internal/mutations"
)

// shiftEligibleFrom is the FROM set: only these archetypes (and
// archetype-less mobs, for which a shift grants an archetype) ever
// shift. Authored specialists — bosses, leader, casters, thief/
// lookout/scout, noncombat_* — never shift.
var shiftEligibleFrom = map[string]bool{
	"":                true,
	"generic_fighter": true,
	"predator":        true,
	"prey":            true,
	"combat_passive":  true,
}

// shiftTargetWhitelist is the TO set: archetypes any mob can credibly
// play. archer is excluded (needs a ranged weapon + ammo we can't
// conjure); ambusher is TO-only (any mob can take the Hidden buff,
// but authored ambushers keep their tuning).
var shiftTargetWhitelist = map[string]bool{
	"generic_fighter":  true,
	"predator":         true,
	"prey":             true,
	"combat_passive":   true,
	"tank_taunter":     true,
	"defensive_caster": true,
	"pure_caster":      true,
	"ambusher":         true,
}

// validateArchetypePulls is the testable core of ValidateArchetypePulls.
// fileExists is injected so tests don't depend on the config data path.
func validateArchetypePulls(specs []*mutations.MutationSpec, fileExists func(string) bool) error {
	for _, spec := range specs {
		if spec.ArchetypePull == "" {
			continue
		}
		if !shiftTargetWhitelist[spec.ArchetypePull] {
			return fmt.Errorf("mutation %q: archetype_pull %q is not in the shift target whitelist",
				spec.MutationId, spec.ArchetypePull)
		}
		if !fileExists(GetArchetypePath(spec.ArchetypePull)) {
			return fmt.Errorf("mutation %q: archetype_pull %q has no archetype file at %s",
				spec.MutationId, spec.ArchetypePull, GetArchetypePath(spec.ArchetypePull))
		}
	}
	return nil
}

// ValidateArchetypePulls panics at boot when any mutation's
// archetype_pull names a nonexistent archetype or one outside the
// target whitelist — same convention as the schedule validators;
// caught by the pre-push boot test. Call after mutations and behavior
// data files are loaded.
func ValidateArchetypePulls() {
	err := validateArchetypePulls(mutations.AllSpecs(), func(path string) bool {
		_, statErr := os.Stat(path)
		return statErr == nil
	})
	if err != nil {
		panic(err.Error())
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/behaviortree/ -run TestValidateArchetypePullsCore -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/behaviortree/archetype_shift.go internal/behaviortree/archetype_shift_test.go
git commit -m "feat(behaviortree): archetype shift sets + archetype_pull boot validation"
```

---

### Task 6: `ReevaluateArchetypeShift` core

**Files:**
- Modify: `internal/behaviortree/archetype_shift.go`
- Modify: `internal/behaviortree/archetype_shift_test.go`

- [ ] **Step 1: Write the failing tests** (append to `archetype_shift_test.go`)

```go
// --- ReevaluateArchetypeShift ---
// Reuses buildMutationMob (actions_mutation_test.go) and
// seedMutationTestRoom/mutTestRoomId (actions_mutation_at_target_test.go).

func seedShiftSpecs(t *testing.T) {
	t.Helper()
	cleanup := mutations.SeedMutationsForTest(map[string]*mutations.MutationSpec{
		"brawn":    {MutationId: "brawn", Rarity: 9, ArchetypePull: "generic_fighter"},
		"fangs":    {MutationId: "fangs", Rarity: 5, ArchetypePull: "predator"},
		"aura":     {MutationId: "aura", Rarity: 5, ArchetypePull: "combat_passive"},
		"plain":    {MutationId: "plain", Rarity: 10}, // no pull — rarity must NOT matter
	})
	t.Cleanup(cleanup)
}

func TestReevaluateArchetypeShift_IneligibleArchetypeNeverShifts(t *testing.T) {
	seedShiftSpecs(t)
	seedMutationTestRoom(t)
	mob := buildMutationMob(t, 9401, 99901, mutTestRoomId)
	mob.BehaviorArchetype = "boss_soren"
	mob.Character.Mutations = map[string]int{"brawn": 1}

	ReevaluateArchetypeShift(mob)
	if mob.BehaviorArchetype != "boss_soren" {
		t.Fatalf("boss shifted to %q; specialists must never shift", mob.BehaviorArchetype)
	}
}

func TestReevaluateArchetypeShift_PerMobTreeBlocks(t *testing.T) {
	seedShiftSpecs(t)
	seedMutationTestRoom(t)
	mob := buildMutationMob(t, 9402, 99902, mutTestRoomId)
	mob.BehaviorArchetype = "generic_fighter"
	mob.Character.Mutations = map[string]int{"fangs": 1}

	// Install a per-mob tree — it shadows archetypes, so no shift.
	node, err := LoadTreeFromBytes([]byte("tree:\n  type: selector\n  children:\n    - type: action\n      event: mob_idle\n      do: attack\n"))
	if err != nil {
		t.Fatalf("LoadTreeFromBytes: %v", err)
	}
	t.Cleanup(GetEngine().SetMobTreeForTest(99902, node))

	ReevaluateArchetypeShift(mob)
	if mob.BehaviorArchetype != "generic_fighter" {
		t.Fatalf("per-mob-tree mob shifted to %q; shadowed mobs must not shift", mob.BehaviorArchetype)
	}
}

func TestReevaluateArchetypeShift_RarestPullWins(t *testing.T) {
	seedShiftSpecs(t)
	seedMutationTestRoom(t)
	mob := buildMutationMob(t, 9403, 99903, mutTestRoomId)
	mob.BehaviorArchetype = "prey"
	// fangs (r5, predator) + brawn (r9, generic_fighter) + plain (r10, NO pull)
	mob.Character.Mutations = map[string]int{"fangs": 1, "brawn": 1, "plain": 1}

	ReevaluateArchetypeShift(mob)
	if mob.BehaviorArchetype != "generic_fighter" {
		t.Fatalf("got %q, want generic_fighter (rarest PULL wins; pull-less rarity ignored)", mob.BehaviorArchetype)
	}
}

func TestReevaluateArchetypeShift_RarityTieAlphabeticalKey(t *testing.T) {
	seedShiftSpecs(t)
	seedMutationTestRoom(t)
	mob := buildMutationMob(t, 9404, 99904, mutTestRoomId)
	mob.BehaviorArchetype = "generic_fighter"
	// aura + fangs both r5 → "aura" < "fangs" alphabetically → combat_passive.
	mob.Character.Mutations = map[string]int{"fangs": 1, "aura": 1}

	ReevaluateArchetypeShift(mob)
	if mob.BehaviorArchetype != "combat_passive" {
		t.Fatalf("got %q, want combat_passive (alphabetical tiebreak)", mob.BehaviorArchetype)
	}
}

func TestReevaluateArchetypeShift_SameTargetNoOp(t *testing.T) {
	seedShiftSpecs(t)
	seedMutationTestRoom(t)
	mob := buildMutationMob(t, 9405, 99905, mutTestRoomId)
	mob.BehaviorArchetype = "predator"
	mob.Character.Mutations = map[string]int{"fangs": 1}
	sentinel := NewBehaviorState()
	mob.BTreeState = sentinel

	ReevaluateArchetypeShift(mob)
	if mob.BTreeState != sentinel {
		t.Fatal("same-target shift must be a silent no-op; BTreeState was reset")
	}
}

func TestReevaluateArchetypeShift_SwapResetsStateAndPolicies(t *testing.T) {
	seedShiftSpecs(t)
	seedMutationTestRoom(t)
	mob := buildMutationMob(t, 9406, 99906, mutTestRoomId)
	mob.BehaviorArchetype = "prey"
	mob.Character.Mutations = map[string]int{"brawn": 1}
	mob.BTreeState = NewBehaviorState()

	ReevaluateArchetypeShift(mob)
	if mob.BehaviorArchetype != "generic_fighter" {
		t.Fatalf("got %q, want generic_fighter", mob.BehaviorArchetype)
	}
	if mob.BTreeState != nil {
		t.Fatal("BTreeState must be reset to nil on shift")
	}
	wantSub := characters.DefaultSubmissionPolicyForArchetype("generic_fighter")
	if mob.Character.SubmissionPolicy != wantSub {
		t.Errorf("SubmissionPolicy = %v, want re-derived %v", mob.Character.SubmissionPolicy, wantSub)
	}
	wantSur := characters.DefaultSurrenderPolicyForArchetype("generic_fighter")
	if mob.Character.SurrenderPolicy != wantSur {
		t.Errorf("SurrenderPolicy = %v, want re-derived %v", mob.Character.SurrenderPolicy, wantSur)
	}
}

func TestReevaluateArchetypeShift_AuthoredPolicyPreserved(t *testing.T) {
	seedShiftSpecs(t)
	seedMutationTestRoom(t)
	mob := buildMutationMob(t, 9407, 99907, mutTestRoomId)
	mob.BehaviorArchetype = "prey"
	mob.Character.Mutations = map[string]int{"brawn": 1}
	// Authored YAML override (any non-empty value engages the guard).
	mob.SubmissionPolicy = "authored-value"
	prior := mob.Character.SubmissionPolicy

	ReevaluateArchetypeShift(mob)
	if mob.Character.SubmissionPolicy != prior {
		t.Error("authored submission_policy must not be re-derived on shift")
	}
}
```

Add `"github.com/GoMudEngine/GoMud/internal/characters"` to the test file imports.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/behaviortree/ -run TestReevaluateArchetypeShift -v`
Expected: compile FAIL — `undefined: ReevaluateArchetypeShift`

- [ ] **Step 3: Implement** (append to `archetype_shift.go`)

```go
// archetypeShiftFlavor gives the room-visible line per shift target.
// Keys are TO-whitelist archetypes; anything absent falls back to the
// generic line. No hard numbers, no mechanics leakage (player-message
// SOP).
var archetypeShiftFlavor = map[string]string{
	"generic_fighter":  "squares up, settling into a fighter's stance",
	"predator":         "drops low, its movements turning predatory",
	"prey":             "grows skittish, eyes darting for escape routes",
	"combat_passive":   "stills, its aggression draining away",
	"tank_taunter":     "plants itself and swells with challenge",
	"defensive_caster": "draws inward, the air around it beginning to hum",
	"pure_caster":      "goes strangely calm, eyes lighting with gathered will",
	"ambusher":         "melts toward the shadows, patient and watching",
}

// mobHasPerMobTree reports whether the mob has a per-mob behavior tree —
// per-mob trees shadow archetypes entirely (TryMobBehavior resolution
// order), so such mobs never shift. Checks the engine caches before
// falling back to an os.Stat, mirroring TryMobBehavior.
func mobHasPerMobTree(mobId int, zone string, name string) bool {
	e := GetEngine()
	if e.GetTree(mobId) != nil {
		return true
	}
	if e.HasNoTree(mobId) {
		return false
	}
	_, err := os.Stat(GetBehaviorPath(mobId, zone, name))
	return err == nil
}

// strongestArchetypePull returns the pull of the rarest owned mutation
// that has one (alphabetical key tiebreak, matching the codebase's
// standard rarity sort). Mutations without a pull are ignored entirely —
// their rarity does not compete. Empty string = no pull owned.
func strongestArchetypePull(owned map[string]int) string {
	bestKey, bestRarity, bestPull := "", -1, ""
	for key := range owned {
		spec := mutations.GetMutation(key)
		if spec == nil || spec.ArchetypePull == "" {
			continue
		}
		if spec.Rarity > bestRarity || (spec.Rarity == bestRarity && key < bestKey) {
			bestKey, bestRarity, bestPull = key, spec.Rarity, spec.ArchetypePull
		}
	}
	return bestPull
}

// ReevaluateArchetypeShift re-archetypes a mob toward its strongest
// mutation pull. Called from the mob mutation-ACQUISITION path (never
// deepening) right after the new mutation lands. Silent no-op unless
// all gates pass. Mid-combat is the normal case — the next btree event
// simply evaluates the new tree.
func ReevaluateArchetypeShift(mob *mobs.Mob) {
	if mob == nil {
		return
	}
	if !shiftEligibleFrom[mob.BehaviorArchetype] {
		return
	}
	if mobHasPerMobTree(int(mob.MobId), mob.Zone, mob.Character.Name) {
		return
	}
	target := strongestArchetypePull(mob.Character.Mutations)
	if target == "" || target == mob.BehaviorArchetype {
		return
	}

	prior := mob.BehaviorArchetype
	mob.BehaviorArchetype = target
	mob.BTreeState = nil // EnsureBTreeState lazily re-inits on the next event

	// Re-derive policies with the same author-override guard the spawn
	// path uses (mobs.go): explicit YAML policies stay; archetype-derived
	// ones follow the new role.
	if mob.SubmissionPolicy == "" {
		mob.Character.SubmissionPolicy = characters.DefaultSubmissionPolicyForArchetype(target)
	}
	if mob.SurrenderPolicy == "" {
		mob.Character.SurrenderPolicy = characters.DefaultSurrenderPolicyForArchetype(target)
	}

	// Additively merge the new archetype's default goals (learned goals
	// survive). Goal WEIGHTS need no work — goals/lookup.go keys off
	// mob.BehaviorArchetype dynamically.
	goals.MergeArchetypeDefaults(int(mob.MobId), util.ConvertForFilename(mob.Character.Name), mob)

	if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
		flavor, ok := archetypeShiftFlavor[target]
		if !ok {
			flavor = "carries itself differently, something fundamental shifted"
		}
		room.SendTextVisual(messaging.CategoryMutation, fmt.Sprintf(
			`<ansi fg="magenta"><ansi fg="mobname">%s</ansi> %s.</ansi>`,
			mob.Character.Name, flavor))
	}

	mudlog.Info("ArchetypeShift",
		"mobId", int(mob.MobId), "instanceId", mob.InstanceId,
		"name", mob.Character.Name, "from", prior, "to", target)
}
```

Extend the file's imports to:

```go
import (
	"fmt"
	"os"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
)
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/behaviortree/ -run "TestReevaluateArchetypeShift|TestValidateArchetypePulls" -v && go test ./internal/behaviortree/ 2>&1 | tail -2`
Expected: all PASS; full package green. (The goals merge is a no-op in these tests — no defaults lookup registered — so no goal files are written into the fixtures dir.)

- [ ] **Step 5: Commit**

```bash
git add internal/behaviortree/archetype_shift.go internal/behaviortree/archetype_shift_test.go
git commit -m "feat(behaviortree): ReevaluateArchetypeShift — mutation-driven archetype swap"
```

---

### Task 7: Hook wiring — shift on acquisition

**Files:**
- Modify: `internal/hooks/NewRound_MobRoundTick.go` (`tickMobMutationAcquisition`, the `if canAcquire {` branch ~line 284)
- Test: `internal/hooks/mutation_acquire_shift_test.go` (new)

- [ ] **Step 1: Extract the acquisition side-effect block**

In `tickMobMutationAcquisition`, the acquisition branch currently reads:

```go
	if canAcquire {
		sp := species.GetSpecies(mob.Character.SpeciesId)
		pool := mutations.GetWeightedPool(mob.Character.Mutations, sp)
		if mutId := mutations.RollAcquisition(pool); mutId != "" {
			// ... ~30 lines: set map entry, room text, world event ...
		}
	}
```

Refactor: move everything inside `if mutId := ...; mutId != ""` into a new function in the same file, and call it:

```go
	if canAcquire {
		sp := species.GetSpecies(mob.Character.SpeciesId)
		pool := mutations.GetWeightedPool(mob.Character.Mutations, sp)
		if mutId := mutations.RollAcquisition(pool); mutId != "" {
			applyAcquiredMutation(mob, mutId)
		}
	}
```

```go
// applyAcquiredMutation applies a newly rolled mutation to a mob:
// records it, announces it (room text + world event), and re-evaluates
// the mob's archetype against its mutation pulls (2026-07-10 shift
// feature — acquisition only; deepening never re-archetypes).
// Extracted from tickMobMutationAcquisition so the deterministic
// side-effect path is testable without the RNG/threshold plumbing.
func applyAcquiredMutation(mob *mobs.Mob, mutId string) {
	if mob.Character.Mutations == nil {
		mob.Character.Mutations = make(map[string]int)
	}
	mob.Character.Mutations[mutId] = 1
	if spec := mutations.GetMutation(mutId); spec != nil {
		// [MOVE the existing room-text + worldevents block here VERBATIM —
		// the `if room := rooms.LoadRoom(...)` visual send and the
		// worldevents.EmitWorldEvent(MobMutationGained) call, unchanged.]
	}

	behaviortree.ReevaluateArchetypeShift(mob)
}
```

The bracketed line is an instruction to relocate the existing code unchanged, not a placeholder to write new code. Add `"github.com/GoMudEngine/GoMud/internal/behaviortree"` to the file's imports if not already present.

- [ ] **Step 2: Write the test**

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mutations"
)

// TestApplyAcquiredMutationTriggersShift: acquiring a pull-mutation
// re-archetypes an eligible mob. End-to-end over the extracted
// deterministic path (no RNG).
func TestApplyAcquiredMutationTriggersShift(t *testing.T) {
	cleanup := mutations.SeedMutationsForTest(map[string]*mutations.MutationSpec{
		"brawn": {MutationId: "brawn", Rarity: 9, ArchetypePull: "generic_fighter", Name: "Brawn", Visual: "muscles ripple"},
	})
	defer cleanup()

	mob := &mobs.Mob{MobId: 99950, InstanceId: 88801, BehaviorArchetype: "prey"}
	mob.Character.Name = "shiftprobe"
	mob.Character.RoomId = 0 // no room — the visual sends no-op safely
	cleanupMob := mobs.SeedMobsForTest(
		map[int]*mobs.Mob{99950: mob},
		map[int]*mobs.Mob{88801: mob},
	)
	defer cleanupMob()

	applyAcquiredMutation(mob, "brawn")

	if mob.Character.Mutations["brawn"] != 1 {
		t.Fatal("mutation was not recorded")
	}
	if mob.BehaviorArchetype != "generic_fighter" {
		t.Fatalf("BehaviorArchetype = %q, want generic_fighter", mob.BehaviorArchetype)
	}
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/hooks/ -run TestApplyAcquiredMutationTriggersShift -v && go test ./internal/hooks/ 2>&1 | tail -2`
Expected: PASS; full package green

- [ ] **Step 4: Build**

Run: `go build .`
Expected: clean

- [ ] **Step 5: Commit**

```bash
git add internal/hooks/NewRound_MobRoundTick.go internal/hooks/mutation_acquire_shift_test.go
git commit -m "feat(hooks): archetype shift on mob mutation acquisition"
```

---

### Task 8: Instance-save persistence

**Files:**
- Modify: `internal/mobs/instance_save.go` (struct ~line 21; save-build ~line 95-105; the worth-saving check ~line 305-340)
- Modify: `internal/mobs/mobs.go` (restore block ~line 430-465)
- Test: `internal/mobs/instance_save_archetype_test.go` (new)

- [ ] **Step 1: Write the failing test**

The save builder is inline in `SaveMobInstance(mob *Mob) error` (`data := MobInstanceData{...}` at ~line 94); the worth-saving predicate is `hasPersistableState(mob)` (~line 305); loading is `LoadMobInstance(mobId, zone, name, homeRoomId) *MobInstanceData`. The package's `test_main_test.go` already redirects the data path, and `instance_save_test.go`'s `TestSaveMobInstance_UncharmedMobWritesFile` (~line 67) is the write-harness to mirror — copy its mob-construction and cleanup shape.

```go
package mobs

import "testing"

// TestInstanceSavePersistsShiftedArchetype: a mob whose live
// BehaviorArchetype differs from its template persists the shift
// through a SaveMobInstance → LoadMobInstance round trip. A
// non-shifted mob (live == template) writes no archetype field.
func TestInstanceSavePersistsShiftedArchetype(t *testing.T) {
	tmpl := &Mob{MobId: 99960, BehaviorArchetype: "prey"}
	tmpl.Character.Name = "saveprobe"
	cleanup := SeedMobsForTest(map[int]*Mob{99960: tmpl}, map[int]*Mob{})
	defer cleanup()

	// Shifted mob — give it a skill so hasPersistableState is true
	// independent of the new archetype gate (isolates the field test),
	// mirroring TestSaveMobInstance_UncharmedMobWritesFile's setup.
	shifted := &Mob{MobId: 99960, BehaviorArchetype: "generic_fighter", HomeRoomId: 4111}
	shifted.Character.Name = "saveprobe"
	shifted.Character.Zone = "testzone"
	shifted.Character.Skills = map[string]int{"weapon-combat": 3}
	if err := SaveMobInstance(shifted); err != nil {
		t.Fatalf("SaveMobInstance: %v", err)
	}
	t.Cleanup(func() { DeleteMobInstance(99960, "testzone", "saveprobe", 4111) })

	loaded := LoadMobInstance(99960, "testzone", "saveprobe", 4111)
	if loaded == nil {
		t.Fatal("LoadMobInstance returned nil")
	}
	if loaded.BehaviorArchetype != "generic_fighter" {
		t.Fatalf("round-trip BehaviorArchetype = %q, want generic_fighter", loaded.BehaviorArchetype)
	}

	// Non-shifted mob: live archetype == template → field stays empty.
	same := &Mob{MobId: 99960, BehaviorArchetype: "prey", HomeRoomId: 4112}
	same.Character.Name = "saveprobe"
	same.Character.Zone = "testzone"
	same.Character.Skills = map[string]int{"weapon-combat": 3}
	if err := SaveMobInstance(same); err != nil {
		t.Fatalf("SaveMobInstance(same): %v", err)
	}
	t.Cleanup(func() { DeleteMobInstance(99960, "testzone", "saveprobe", 4112) })

	loadedSame := LoadMobInstance(99960, "testzone", "saveprobe", 4112)
	if loadedSame == nil {
		t.Fatal("LoadMobInstance(same) returned nil")
	}
	if loadedSame.BehaviorArchetype != "" {
		t.Fatalf("non-shifted mob persisted BehaviorArchetype %q, want empty", loadedSame.BehaviorArchetype)
	}
}
```

If `SaveMobInstance` reads the zone from a different field than `Character.Zone` (check its body — it may use `mob.Zone`), set that field instead; mirror whatever `TestSaveMobInstance_UncharmedMobWritesFile` sets.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/mobs/ -run TestInstanceSavePersistsShiftedArchetype -v`
Expected: compile FAIL — `data.BehaviorArchetype undefined`

- [ ] **Step 3: Implement**

In `MobInstanceData` (after `MutationProgress`):

```go
	// BehaviorArchetype persists a mutation-driven archetype shift
	// (2026-07-10). Written only when the live value differs from the
	// template's; empty = never shifted.
	BehaviorArchetype string `yaml:"behavior_archetype,omitempty"`
```

In `SaveMobInstance`, immediately after the `data := MobInstanceData{...}` literal (~line 102):

```go
	// Persist a mutation-driven archetype shift (2026-07-10): only when
	// the live value differs from the template's.
	if tmpl := GetMobSpec(mob.MobId); tmpl != nil && mob.BehaviorArchetype != tmpl.BehaviorArchetype {
		data.BehaviorArchetype = mob.BehaviorArchetype
	}
```

In `hasPersistableState`, before the final `return false`:

```go
	if tmpl := GetMobSpec(mob.MobId); tmpl != nil && mob.BehaviorArchetype != tmpl.BehaviorArchetype {
		return true
	}
```

In `mobs.go`, inside the `if savedInstance != nil {` restore block, after `mob.Character.MutationProgress = savedInstance.MutationProgress`:

```go
			// Restore a mutation-driven archetype shift (2026-07-10).
			// Policies were derived from the TEMPLATE archetype earlier in
			// this spawn path, so re-derive them for the restored one —
			// same author-override guard as the original derivation.
			if savedInstance.BehaviorArchetype != "" {
				mob.BehaviorArchetype = savedInstance.BehaviorArchetype
				if mob.SubmissionPolicy == "" {
					mob.Character.SubmissionPolicy = characters.DefaultSubmissionPolicyForArchetype(mob.BehaviorArchetype)
				}
				if mob.SurrenderPolicy == "" {
					mob.Character.SurrenderPolicy = characters.DefaultSurrenderPolicyForArchetype(mob.BehaviorArchetype)
				}
			}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/mobs/ -run TestInstanceSavePersistsShiftedArchetype -v && go test ./internal/mobs/ 2>&1 | tail -2`
Expected: PASS; full package green

- [ ] **Step 5: Commit**

```bash
git add internal/mobs/instance_save.go internal/mobs/mobs.go internal/mobs/instance_save_archetype_test.go
git commit -m "feat(mobs): persist mutation-shifted archetype in instance saves"
```

---

### Task 9: Boot wiring

**Files:**
- Modify: `main.go` (~line 1305, next to `behaviortree.ValidateAutoAggroBehaviorGates()`)

- [ ] **Step 1: Wire the validator**

After the existing `behaviortree.ValidateAutoAggroBehaviorGates()` call:

```go
	// Mutations carrying an archetype_pull must reference a real,
	// whitelisted shift-target archetype — panic at boot, not mid-fight.
	behaviortree.ValidateArchetypePulls()
```

- [ ] **Step 2: Build + full test sweep**

Run: `go build . && go test ./internal/mutations/ ./internal/characters/ ./internal/mobs/ ./internal/goals/ ./internal/behaviortree/ ./internal/hooks/ 2>&1 | tail -8`
Expected: clean build, all packages `ok`

- [ ] **Step 3: Commit**

```bash
git add main.go
git commit -m "feat(boot): validate mutation archetype_pull targets at startup"
```

---

### Task 10: Content — the 10 provisional pulls

**Files:**
- Modify: 10 files under `_datafiles/world/dogmud/mutations/`

- [ ] **Step 1: Add `archetype_pull` to each** (one line per file, placed after `rarity:`)

| File | Line to add |
|---|---|
| `incorporeal.yaml` | `archetype_pull: defensive_caster` |
| `extra-arms.yaml` | `archetype_pull: generic_fighter` |
| `clawed-hands.yaml` | `archetype_pull: predator` |
| `toxic-bite.yaml` | `archetype_pull: predator` |
| `blinding-spit.yaml` | `archetype_pull: predator` |
| `sonic-shout.yaml` | `archetype_pull: tank_taunter` |
| `dense-muscles.yaml` | `archetype_pull: tank_taunter` |
| `large.yaml` | `archetype_pull: generic_fighter` |
| `small.yaml` | `archetype_pull: ambusher` |
| `pacifism-aura.yaml` | `archetype_pull: combat_passive` |

Each with the shared comment on the line above:

```yaml
# PROVISIONAL pull — re-curate with the mutation-graph redesign.
archetype_pull: <value>
```

- [ ] **Step 2: Boot test (validates the pulls end-to-end per SOP)**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . 
```

Watch for: `mutations.LoadMutationFi loadedCount=41`, `mobs.LoadDataFiles() loadedCount=...`, NO panic from `ValidateArchetypePulls`, clean load past quests. Kill the server after (`taskkill //F //IM GoMud.exe` on Windows).

Negative check (proves the validator bites): temporarily change one pull to `archetype_pull: boss_soren`, boot, expect a panic naming the mutation; revert.

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/mutations/
git commit -m "content(mutations): provisional archetype_pull table (10 mutations)"
```

---

### Task 11: Manual smoke + wrap-up

- [ ] **Step 1: Manual smoke** (local server; no admin mutation-grant command exists for mobs, so drive it through the real acquisition tick)

1. Temporarily raise `MobMutationRate` (and, if acquisition still takes too long, lower `MutationBaseProgress`) in `_datafiles/config.yaml`'s `Balance` block so an in-combat mob acquires a mutation within a few rounds. Note the original values.
2. Wipe instance saves per SOP (`rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*`), boot, and pick a fight with a steppe wolf (`ironwind_steppe`, archetype `predator` — eligible). Stay in combat without killing it.
3. Watch for: the mutation-gain room text, then (if the rolled mutation carries a pull differing from `predator`) the shift flavor line, and changed combat behavior (e.g. a `generic_fighter` cascade: bash/trip attempts instead of predation). The `ArchetypeShift` Info log line (from/to) is the ground truth — acquisition rolls are random, so repeat with another mob if the first rolled a pull-less mutation.
4. Persistence: with a shifted mob alive, restart WITHOUT wiping instance saves; the `ArchetypeShift` log's mob should come back with the shifted archetype (verify via its behavior, or the instance-save YAML under `_datafiles/world/dogmud/mobs.instances/` containing `behavior_archetype:`).
5. Restore the original config values.

- [ ] **Step 2: Full suite + verify**

Run: `go test ./... 2>&1 | grep -E "FAIL|ok" | head -40`
Expected: no FAIL lines

- [ ] **Step 3: Update PATCH_NOTES.md** (rides the next push)

Player-facing entry, no mechanics leakage:

```markdown
- **Mutations change how creatures fight.** A creature that mutates
  mid-battle may fight differently afterward — a wolf that sprouts
  extra arms stops circling and starts brawling; something that slips
  partway out of the physical world stops trading blows and starts
  channeling. Watch for the change in its bearing.
```

- [ ] **Step 4: Final commit**

```bash
git add PATCH_NOTES.md
git commit -m "docs(patch-notes): mutation-driven behavior shifts"
```
