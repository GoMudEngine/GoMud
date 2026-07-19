# Chrysalis Mutation Graph — Engine Core Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the flat random mutation-acquisition roll with a graph-aware engine: cluster-affinity drift, prerequisite gating, and a Body↔Belief opposition that degrades the opposing resource pool.

**Architecture:** Extend `MutationSpec` with `clusters` / `pole` / `prerequisites` data (boot-validated). Add a per-character `ClusterAffinity` accumulator fed by skill use and decayed over time; combine it with owned-mutation "gravity" to weight a new `GetGraphPool` acquisition function that gates candidates by affinity threshold, prerequisites, conflicts, and body-parts. Add pole-depth functions that scale `ConvictionMax` down (deep Body) and gear effectiveness down (deep Belief). All new logic lives in `internal/mutations` (pure, fixture-testable) with thin wiring in `internal/characters` and `internal/hooks`.

**Tech Stack:** Go, existing `internal/mutations` + `internal/characters` + `internal/hooks` packages, `ConfigFloat` balance knobs, `testify` assertions (matching existing tests).

**Scope note:** This plan is the **engine only**. Deferred to follow-on plans (see end): the ~9 phial recipes + strip/re-bloom, clean-break migration + skill-seeded re-bloom, mob archetype affinity-seeding + `archetype_pull`→cluster-tag re-curation, the damage-absorbed→Ironhide signal, the full per-cluster keystone content, and helpfile copy. The engine is unit-testable against synthetic fixtures via `mutations.SeedMutationsForTest`; it does not require the real content to exist.

**Spec:** `docs/superpowers/specs/2026-07-10-chrysalis-mutation-graph-design.md`

---

## File Structure

**Create:**
- `internal/mutations/graph.go` — `MutationPrereq` type, `MutationSpec` graph fields live in `mutations.go`; this file holds `KnownClusters`, `ValidateGraph()`, `ClustersForSkill`, `OwnedGravity`, `PrereqsMet`, `PoleDepth`.
- `internal/mutations/affinity.go` — `EffectiveAffinity`, `DecayAffinity`, `depthThreshold`, `affinityFor`, `GetGraphPool`.
- `internal/mutations/opposition.go` — `BodyConvictionScale`, `BeliefGearScale`.
- `internal/mutations/graph_test.go`, `affinity_test.go`, `opposition_test.go` — unit tests.

**Modify:**
- `internal/mutations/mutations.go` — add `Clusters`, `Pole`, `Prerequisites` fields to `MutationSpec`; extend `GearEffectivenessMultiplier`.
- `internal/characters/character.go` — add `ClusterAffinity` field + `AddClusterAffinity`.
- `internal/characters/progression.go` — feed affinity in `OnSkillUseScaled`.
- `internal/characters/validate.go` — apply `BodyConvictionScale` in `RecalculateStats`.
- `internal/hooks/NewRound_MobRoundTick.go` + `NewRound_UserRoundTick.go` — switch acquisition to `GetGraphPool`; decay affinity per tick.
- `internal/configs/config.balance.go` + `config.balance.progression.go` — new knobs.
- `main.go` (or the boot sequence that calls `mutations.LoadMutationFiles`) — call `mutations.ValidateGraph()`.

---

## Phase 1 — Data model & validation

### Task 1: Add graph fields to MutationSpec

**Files:**
- Modify: `internal/mutations/mutations.go`
- Create: `internal/mutations/graph.go`
- Test: `internal/mutations/graph_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/mutations/graph_test.go
package mutations

import "testing"

func TestMutationSpec_GraphFields(t *testing.T) {
	spec := &MutationSpec{
		MutationId:    "rending-claws",
		Name:          "Rending Claws",
		Rarity:        3,
		Clusters:      []string{"ravener"},
		Pole:          "body",
		Prerequisites: []MutationPrereq{{Id: "hollow-bones", MinLevel: 2}},
	}
	if got := spec.Prerequisites[0].MinLevel; got != 2 {
		t.Fatalf("MinLevel = %d, want 2", got)
	}
	if spec.Pole != "body" {
		t.Fatalf("Pole = %q, want body", spec.Pole)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/mutations/ -run TestMutationSpec_GraphFields -v`
Expected: FAIL — `spec.Clusters`, `spec.Pole`, `spec.Prerequisites`, and `MutationPrereq` undefined (compile error).

- [ ] **Step 3: Add the fields and the prereq type**

In `internal/mutations/mutations.go`, add to the `MutationSpec` struct (after the `ArchetypePull` field):

```go
	// Clusters lists the design-side playstyle clusters this mutation
	// belongs to (empty = universal/generalist). Steers acquisition drift.
	Clusters []string `yaml:"clusters,omitempty"`

	// Pole is "body", "belief", or "" (neutral). Drives the opposition:
	// deep Body shrinks the Conviction pool; deep Belief degrades gear.
	Pole string `yaml:"pole,omitempty"`

	// Prerequisites lists mutations (with min level) that must be owned
	// before this one can be acquired. Gates apex/spine mutations.
	Prerequisites []MutationPrereq `yaml:"prerequisites,omitempty"`
```

In `internal/mutations/graph.go`:

```go
package mutations

// MutationPrereq is a single prerequisite: an owned mutation id at a
// minimum level. A MinLevel of 0 is treated as 1.
type MutationPrereq struct {
	Id       string `yaml:"id"`
	MinLevel int    `yaml:"min_level"`
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/mutations/ -run TestMutationSpec_GraphFields -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/mutations/mutations.go internal/mutations/graph.go internal/mutations/graph_test.go
git commit -m "feat(mutations): add clusters/pole/prerequisites to MutationSpec"
```

---

### Task 2: Boot validation of graph data

**Files:**
- Modify: `internal/mutations/graph.go`
- Test: `internal/mutations/graph_test.go`
- Modify: the boot sequence that calls `mutations.LoadMutationFiles()` (grep for it: `grep -rn "LoadMutationFiles" --include=*.go .` — call site is the server boot, alongside the other `Load*Files` calls).

- [ ] **Step 1: Write the failing test**

```go
// internal/mutations/graph_test.go (append)
func TestValidateGraph_UnknownPrereqPanics(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"a": {MutationId: "a", Name: "A", Rarity: 1,
			Prerequisites: []MutationPrereq{{Id: "does-not-exist"}}},
	})
	defer cleanup()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on unknown prerequisite id")
		}
	}()
	ValidateGraph()
}

func TestValidateGraph_UnknownPolePanics(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"a": {MutationId: "a", Name: "A", Rarity: 1, Pole: "spooky"},
	})
	defer cleanup()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on unknown pole")
		}
	}()
	ValidateGraph()
}

func TestValidateGraph_ValidPasses(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"hollow-bones": {MutationId: "hollow-bones", Name: "Hollow Bones", Rarity: 2,
			Clusters: []string{"generalist"}, Pole: ""},
		"winged-flight": {MutationId: "winged-flight", Name: "Winged Flight", Rarity: 8,
			Clusters: []string{"generalist"},
			Prerequisites: []MutationPrereq{{Id: "hollow-bones", MinLevel: 1}}},
	})
	defer cleanup()
	ValidateGraph() // must not panic
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/mutations/ -run TestValidateGraph -v`
Expected: FAIL — `ValidateGraph` undefined.

- [ ] **Step 3: Implement `KnownClusters` + `ValidateGraph`**

In `internal/mutations/graph.go`:

```go
import "fmt"

// KnownClusters is the closed set of design-side cluster tags. "generalist"
// is the central hub. Adding a cluster here is a deliberate design act.
var KnownClusters = map[string]bool{
	"colossus": true, "ironhide": true, "zealot": true, "manifester": true,
	"ethereal": true, "weaver": true, "trickster": true, "stalker": true,
	"ravener": true, "generalist": true,
}

var knownPoles = map[string]bool{"": true, "body": true, "belief": true}

// ValidateGraph panics if any loaded mutation references an unknown cluster,
// an unknown pole, or a prerequisite id that does not exist. Called at boot
// after LoadMutationFiles (same convention as ValidateBodyPartTags).
func ValidateGraph() {
	for _, spec := range allMutations {
		if !knownPoles[spec.Pole] {
			panic(fmt.Sprintf("mutation %q: unknown pole %q (want body|belief|\"\")",
				spec.MutationId, spec.Pole))
		}
		for _, cl := range spec.Clusters {
			if !KnownClusters[cl] {
				panic(fmt.Sprintf("mutation %q: unknown cluster %q", spec.MutationId, cl))
			}
		}
		for _, p := range spec.Prerequisites {
			if _, ok := allMutations[p.Id]; !ok {
				panic(fmt.Sprintf("mutation %q: prerequisite %q does not exist",
					spec.MutationId, p.Id))
			}
		}
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/mutations/ -run TestValidateGraph -v`
Expected: PASS (all three)

- [ ] **Step 5: Wire into boot**

Find the boot call site: `grep -rn "mutations.ValidateBodyPartTags\|mutations.LoadMutationFiles" --include=*.go .`
Immediately after the existing `mutations.ValidateBodyPartTags()` call, add:

```go
	mutations.ValidateGraph()
```

- [ ] **Step 6: Verify build + boot**

Run: `go build ./...`
Expected: builds clean.
Run: `go run . 2>&1 | grep -m1 "mutations.LoadMutationFiles"` (Ctrl-C after the load line prints; no panic before it).
Expected: load line prints, no ValidateGraph panic (existing YAML has no graph fields yet, so all pass trivially).

- [ ] **Step 7: Commit**

```bash
git add internal/mutations/graph.go internal/mutations/graph_test.go <boot-file>
git commit -m "feat(mutations): boot validation for cluster/pole/prerequisite graph data"
```

---

## Phase 2 — Cluster affinity

### Task 3: Skill→cluster map and owned-gravity

**Files:**
- Modify: `internal/mutations/graph.go`
- Test: `internal/mutations/graph_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/mutations/graph_test.go (append)
func TestClustersForSkill(t *testing.T) {
	if got := ClustersForSkill("spellcasting"); len(got) != 1 || got[0] != "ethereal" {
		t.Fatalf("spellcasting -> %v, want [ethereal]", got)
	}
	if got := ClustersForSkill("cooking"); got != nil {
		t.Fatalf("cooking -> %v, want nil", got)
	}
}

func TestOwnedGravity(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"claws": {MutationId: "claws", Name: "Claws", Rarity: 3, Clusters: []string{"ravener"}},
		"fangs": {MutationId: "fangs", Name: "Fangs", Rarity: 4, Clusters: []string{"ravener", "stalker"}},
	})
	defer cleanup()
	g := OwnedGravity(map[string]int{"claws": 1, "fangs": 2})
	if g["ravener"] != 3 { // 1 + 2
		t.Fatalf("ravener gravity = %v, want 3", g["ravener"])
	}
	if g["stalker"] != 2 {
		t.Fatalf("stalker gravity = %v, want 2", g["stalker"])
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/mutations/ -run "TestClustersForSkill|TestOwnedGravity" -v`
Expected: FAIL — undefined `ClustersForSkill`, `OwnedGravity`.

- [ ] **Step 3: Implement**

In `internal/mutations/graph.go`:

```go
// skillClusters maps a skill tag to the cluster(s) its use drifts toward.
// Skills not listed produce no drift. (Ironhide's tank signal comes from a
// damage-absorbed hook wired in a follow-on plan.)
var skillClusters = map[string][]string{
	"weapon-combat":  {"colossus"},
	"unarmed-combat": {"ravener"},
	"ranged-combat":  {"stalker"},
	"skullduggery":   {"stalker"},
	"spellcasting":   {"ethereal"},
	"rhetoric":       {"zealot"},
	"manifestation":  {"manifester"},
}

// ClustersForSkill returns the clusters a skill's use drifts toward (nil if none).
func ClustersForSkill(skill string) []string { return skillClusters[skill] }

// OwnedGravity returns each cluster's pull from currently-owned mutations:
// sum of levels of owned mutations tagged with that cluster. Dual-cluster
// (bridge) mutations contribute to both.
func OwnedGravity(owned map[string]int) map[string]float64 {
	g := make(map[string]float64)
	for id, lvl := range owned {
		spec := GetMutation(id)
		if spec == nil {
			continue
		}
		for _, cl := range spec.Clusters {
			g[cl] += float64(lvl)
		}
	}
	return g
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/mutations/ -run "TestClustersForSkill|TestOwnedGravity" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/mutations/graph.go internal/mutations/graph_test.go
git commit -m "feat(mutations): skill->cluster map and owned-mutation gravity"
```

---

### Task 4: Character.ClusterAffinity field + accessor

**Files:**
- Modify: `internal/characters/character.go`
- Test: `internal/characters/character_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/characters/character_test.go (append)
func TestAddClusterAffinity_Accumulates(t *testing.T) {
	c := &Character{}
	c.AddClusterAffinity("ravener", 1.5)
	c.AddClusterAffinity("ravener", 0.5)
	if got := c.ClusterAffinity["ravener"]; got != 2.0 {
		t.Fatalf("ravener affinity = %v, want 2.0", got)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/characters/ -run TestAddClusterAffinity_Accumulates -v`
Expected: FAIL — `AddClusterAffinity` / `ClusterAffinity` undefined.

- [ ] **Step 3: Implement**

In `internal/characters/character.go`, add to the `Character` struct near `StatUseCount` (line ~273):

```go
	ClusterAffinity map[string]float64 `yaml:"clusteraffinity,omitempty"` // cluster -> drift affinity (mutation graph)
```

Add the method (place near the other progression accessors, e.g. after the struct's methods in `character.go`):

```go
// AddClusterAffinity accumulates mutation-graph drift affinity toward a
// cluster. Lazily initializes the map.
func (c *Character) AddClusterAffinity(cluster string, amount float64) {
	if c.ClusterAffinity == nil {
		c.ClusterAffinity = make(map[string]float64)
	}
	c.ClusterAffinity[cluster] += amount
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/characters/ -run TestAddClusterAffinity_Accumulates -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/characters/character.go internal/characters/character_test.go
git commit -m "feat(characters): ClusterAffinity accumulator on Character"
```

---

### Task 5: Feed affinity on skill use

**Files:**
- Modify: `internal/characters/progression.go:235-258` (`OnSkillUseScaled`)
- Modify: `internal/configs/config.balance.go` + `config.balance.progression.go` (knob)
- Test: `internal/characters/progression_test.go`

- [ ] **Step 1: Add the config knob (no test — data default)**

In `internal/configs/config.balance.go`, add near `MutationProgressGainPerRound` (line ~266):

```go
	MutationAffinityPerSkillUse ConfigFloat `yaml:"MutationAffinityPerSkillUse"` // drift affinity added per cluster-relevant skill use (default 1.0)
```

In `internal/configs/config.balance.progression.go`, add near the other Mutation defaults (line ~75):

```go
	if b.MutationAffinityPerSkillUse <= 0 {
		b.MutationAffinityPerSkillUse = 1.0
	}
```

- [ ] **Step 2: Write the failing test**

```go
// internal/characters/progression_test.go (append)
func TestOnSkillUseScaled_FeedsClusterAffinity(t *testing.T) {
	c := &Character{Name: "T"}
	c.OnSkillUseScaled("spellcasting", 0, 1.0)
	if c.ClusterAffinity["ethereal"] <= 0 {
		t.Fatalf("expected ethereal affinity > 0, got %v", c.ClusterAffinity["ethereal"])
	}
	c.OnSkillUseScaled("cooking", 0, 1.0) // no mapping -> no new clusters
	if _, ok := c.ClusterAffinity["colossus"]; ok {
		t.Fatal("cooking should not add cluster affinity")
	}
}
```

- [ ] **Step 3: Run to verify it fails**

Run: `go test ./internal/characters/ -run TestOnSkillUseScaled_FeedsClusterAffinity -v`
Expected: FAIL — no affinity added.

- [ ] **Step 4: Implement the feed**

In `internal/characters/progression.go`, inside `OnSkillUseScaled`, after `c.TrackSkillUse(skillName)` (line 236):

```go
	// Mutation-graph drift: cluster-relevant skill use nudges affinity.
	if clusters := mutations.ClustersForSkill(skillName); clusters != nil {
		amt := float64(configs.GetBalanceConfig().MutationAffinityPerSkillUse)
		for _, cl := range clusters {
			c.AddClusterAffinity(cl, amt)
		}
	}
```

Ensure `internal/mutations` and `internal/configs` are imported in `progression.go` (they likely already are; add if the build complains).

- [ ] **Step 5: Run to verify it passes**

Run: `go test ./internal/characters/ -run TestOnSkillUseScaled_FeedsClusterAffinity -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/characters/progression.go internal/configs/config.balance.go internal/configs/config.balance.progression.go internal/characters/progression_test.go
git commit -m "feat(characters): skill use feeds mutation cluster affinity"
```

---

### Task 6: Affinity decay

**Files:**
- Create: `internal/mutations/affinity.go`
- Test: `internal/mutations/affinity_test.go`
- Modify: `config.balance.go` + `config.balance.progression.go` (decay knob)

- [ ] **Step 1: Add the decay knob**

In `config.balance.go`:

```go
	MutationAffinityDecay ConfigFloat `yaml:"MutationAffinityDecay"` // per-tick multiplicative decay of cluster affinity (default 0.98)
```

In `config.balance.progression.go`:

```go
	if b.MutationAffinityDecay <= 0 || b.MutationAffinityDecay > 1.0 {
		b.MutationAffinityDecay = 0.98
	}
```

- [ ] **Step 2: Write the failing test**

```go
// internal/mutations/affinity_test.go
package mutations

import "testing"

func TestDecayAffinity(t *testing.T) {
	aff := map[string]float64{"ravener": 10, "tiny": 0.005}
	DecayAffinity(aff, 0.5)
	if aff["ravener"] != 5 {
		t.Fatalf("ravener = %v, want 5", aff["ravener"])
	}
	if _, ok := aff["tiny"]; ok {
		t.Fatal("negligible affinity should be pruned")
	}
}
```

- [ ] **Step 3: Run to verify it fails**

Run: `go test ./internal/mutations/ -run TestDecayAffinity -v`
Expected: FAIL — `DecayAffinity` undefined.

- [ ] **Step 4: Implement**

In `internal/mutations/affinity.go`:

```go
package mutations

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/species"
)

// DecayAffinity multiplies every affinity by rate, pruning entries that
// fall below a negligible floor so the map does not grow unbounded.
func DecayAffinity(aff map[string]float64, rate float64) {
	for k, v := range aff {
		nv := v * rate
		if nv < 0.01 {
			delete(aff, k)
		} else {
			aff[k] = nv
		}
	}
}
```

- [ ] **Step 5: Run to verify it passes**

Run: `go test ./internal/mutations/ -run TestDecayAffinity -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/mutations/affinity.go internal/mutations/affinity_test.go internal/configs/config.balance.go internal/configs/config.balance.progression.go
git commit -m "feat(mutations): cluster affinity decay"
```

---

## Phase 3 — Graph acquisition

### Task 7: Prerequisite check + effective affinity

**Files:**
- Modify: `internal/mutations/graph.go` (PrereqsMet), `internal/mutations/affinity.go` (EffectiveAffinity)
- Test: `internal/mutations/affinity_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/mutations/affinity_test.go (append)
func TestPrereqsMet(t *testing.T) {
	spec := &MutationSpec{MutationId: "flight", Name: "Flight", Rarity: 8,
		Prerequisites: []MutationPrereq{{Id: "hollow-bones", MinLevel: 2}}}
	if PrereqsMet(map[string]int{"hollow-bones": 1}, spec) {
		t.Fatal("should be unmet at level 1")
	}
	if !PrereqsMet(map[string]int{"hollow-bones": 2}, spec) {
		t.Fatal("should be met at level 2")
	}
	if !PrereqsMet(map[string]int{}, &MutationSpec{MutationId: "x"}) {
		t.Fatal("no prerequisites -> always met")
	}
}

func TestEffectiveAffinity_AddsGravity(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"claws": {MutationId: "claws", Name: "Claws", Rarity: 3, Clusters: []string{"ravener"}},
	})
	defer cleanup()
	eff := EffectiveAffinity(map[string]int{"claws": 2}, map[string]float64{"ravener": 1})
	if eff["ravener"] != 3 { // action 1 + gravity 2
		t.Fatalf("ravener = %v, want 3", eff["ravener"])
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/mutations/ -run "TestPrereqsMet|TestEffectiveAffinity" -v`
Expected: FAIL — undefined `PrereqsMet`, `EffectiveAffinity`.

- [ ] **Step 3: Implement**

In `internal/mutations/graph.go`:

```go
// PrereqsMet reports whether owned satisfies every prerequisite of spec.
func PrereqsMet(owned map[string]int, spec *MutationSpec) bool {
	for _, p := range spec.Prerequisites {
		min := p.MinLevel
		if min < 1 {
			min = 1
		}
		if owned[p.Id] < min {
			return false
		}
	}
	return true
}
```

In `internal/mutations/affinity.go`:

```go
// EffectiveAffinity combines the character's action-driven affinity with the
// gravity of their currently-owned mutations. Returns a fresh map.
func EffectiveAffinity(owned map[string]int, actionAff map[string]float64) map[string]float64 {
	eff := make(map[string]float64, len(actionAff)+4)
	for k, v := range actionAff {
		eff[k] = v
	}
	for k, v := range OwnedGravity(owned) {
		eff[k] += v
	}
	return eff
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/mutations/ -run "TestPrereqsMet|TestEffectiveAffinity" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/mutations/graph.go internal/mutations/affinity.go internal/mutations/affinity_test.go
git commit -m "feat(mutations): prerequisite check and effective-affinity combiner"
```

---

### Task 8: GetGraphPool — gated, affinity-weighted acquisition

**Files:**
- Modify: `internal/mutations/affinity.go`
- Modify: `config.balance.go` + `config.balance.progression.go` (threshold knob)
- Test: `internal/mutations/affinity_test.go`

- [ ] **Step 1: Add the threshold knob**

In `config.balance.go`:

```go
	MutationAffinityPerRarity ConfigFloat `yaml:"MutationAffinityPerRarity"` // affinity required per point of rarity to unlock a mutation (default 1.0)
```

In `config.balance.progression.go`:

```go
	if b.MutationAffinityPerRarity <= 0 {
		b.MutationAffinityPerRarity = 1.0
	}
```

- [ ] **Step 2: Write the failing test**

```go
// internal/mutations/affinity_test.go (append)
func TestGetGraphPool_GatesByAffinityAndPrereqs(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		// universal enabler: no cluster -> always eligible
		"hollow-bones": {MutationId: "hollow-bones", Name: "HB", Rarity: 2},
		// ravener deep keystone: needs affinity >= rarity*1.0 = 5
		"apex": {MutationId: "apex", Name: "Apex", Rarity: 5, Clusters: []string{"ravener"}},
		// gated behind a prerequisite the owner lacks
		"flight": {MutationId: "flight", Name: "Flight", Rarity: 8, Clusters: []string{"generalist"},
			Prerequisites: []MutationPrereq{{Id: "hollow-bones", MinLevel: 1}}},
	})
	defer cleanup()

	// Low ravener affinity -> apex excluded, hollow-bones present, flight blocked (no prereq)
	pool := GetGraphPool(map[string]int{}, map[string]float64{"ravener": 1}, nil)
	if contains(pool, "apex") {
		t.Fatal("apex should be gated out at low affinity")
	}
	if !contains(pool, "hollow-bones") {
		t.Fatal("universal hollow-bones should always be eligible")
	}
	if contains(pool, "flight") {
		t.Fatal("flight should be blocked without its prerequisite")
	}

	// High ravener affinity -> apex now eligible
	pool2 := GetGraphPool(map[string]int{}, map[string]float64{"ravener": 10}, nil)
	if !contains(pool2, "apex") {
		t.Fatal("apex should be eligible once affinity clears threshold")
	}
}

func contains(s []string, want string) bool {
	for _, v := range s {
		if v == want {
			return true
		}
	}
	return false
}
```

- [ ] **Step 3: Run to verify it fails**

Run: `go test ./internal/mutations/ -run TestGetGraphPool_GatesByAffinityAndPrereqs -v`
Expected: FAIL — undefined `GetGraphPool`.

- [ ] **Step 4: Implement**

In `internal/mutations/affinity.go`:

```go
// depthThreshold is the affinity required to unlock a mutation of the given
// rarity. Rarer/deeper keystones demand more affinity.
func depthThreshold(rarity int) float64 {
	return float64(rarity) * float64(configs.GetBalanceConfig().MutationAffinityPerRarity)
}

// affinityFor returns the best affinity across a mutation's clusters. A
// mutation with no clusters (universal/generalist enabler) is always eligible.
func affinityFor(spec *MutationSpec, aff map[string]float64) float64 {
	if len(spec.Clusters) == 0 {
		return math.MaxFloat64
	}
	best := 0.0
	for _, cl := range spec.Clusters {
		if aff[cl] > best {
			best = aff[cl]
		}
	}
	return best
}

// GetGraphPool builds a weighted acquisition pool from the mutation graph.
// A candidate is included only if: not already owned, not conflicting, its
// body-part requirements fit the species, its prerequisites are owned, AND
// its best cluster affinity clears its rarity-based depth threshold. Weight
// scales with rarity (commoner = heavier) plus the surplus affinity, so a
// strongly-expressed cluster dominates the roll. aff must already fold in
// owned-gravity (see EffectiveAffinity). Pass nil sp to skip body filtering.
func GetGraphPool(owned map[string]int, aff map[string]float64, sp *species.Species) []string {
	pool := make([]string, 0, len(allMutations)*4)
	for id, spec := range allMutations {
		if _, has := owned[id]; has {
			continue
		}
		if HasConflict(owned, id) {
			continue
		}
		if !spec.CanApplyTo(sp) {
			continue
		}
		if !PrereqsMet(owned, spec) {
			continue
		}
		a := affinityFor(spec, aff)
		if a < depthThreshold(spec.Rarity) {
			continue
		}
		weight := 11 - spec.Rarity
		if weight < 1 {
			weight = 1
		}
		if a != math.MaxFloat64 {
			weight += int(a) // clustered mutations get louder as their cluster grows
		}
		for i := 0; i < weight; i++ {
			pool = append(pool, id)
		}
	}
	return pool
}
```

- [ ] **Step 5: Run to verify it passes**

Run: `go test ./internal/mutations/ -run TestGetGraphPool_GatesByAffinityAndPrereqs -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/mutations/affinity.go internal/mutations/affinity_test.go internal/configs/config.balance.go internal/configs/config.balance.progression.go
git commit -m "feat(mutations): GetGraphPool gated affinity-weighted acquisition"
```

---

### Task 9: Wire GetGraphPool into the acquisition ticks

**Files:**
- Modify: `internal/hooks/NewRound_MobRoundTick.go:284-290`
- Modify: `internal/hooks/NewRound_UserRoundTick.go:282-291`
- Test: manual smoke (the pure logic is covered by Task 8; this is call-site substitution).

- [ ] **Step 1: Swap the mob acquisition call site**

In `internal/hooks/NewRound_MobRoundTick.go`, replace the `if canAcquire {` block (lines ~284-290):

```go
	if canAcquire {
		sp := species.GetSpecies(mob.Character.SpeciesId)
		aff := mutations.EffectiveAffinity(mob.Character.Mutations, mob.Character.ClusterAffinity)
		pool := mutations.GetGraphPool(mob.Character.Mutations, aff, sp)
		if mutId := mutations.RollAcquisition(pool); mutId != "" {
			applyAcquiredMutation(mob, mutId)
		}
	}
	// Fade drift each acquisition tick so recent behavior dominates.
	mutations.DecayAffinity(mob.Character.ClusterAffinity, float64(mb.MutationAffinityDecay))
```

- [ ] **Step 2: Swap the user acquisition call site**

In `internal/hooks/NewRound_UserRoundTick.go`, replace the `} else if canAcquire {` pool lines (~283-285):

```go
							} else if canAcquire {
								sp := species.GetSpecies(user.Character.SpeciesId)
								aff := mutations.EffectiveAffinity(user.Character.Mutations, user.Character.ClusterAffinity)
								pool := mutations.GetGraphPool(user.Character.Mutations, aff, sp)
```

Then, after the acquisition/deepen branch completes for the user (locate the closing of the `if mob.Character.MutationProgress >= threshold` block for the user tick), add a decay call so user affinity also fades. If a clean single site is unclear, add it right after `user.Character.MutationProgress = 0` is set:

```go
	mutations.DecayAffinity(user.Character.ClusterAffinity, float64(configs.GetBalanceConfig().MutationAffinityDecay))
```

(Import `configs` in that file if not already present.)

- [ ] **Step 3: Build**

Run: `go build ./...`
Expected: builds clean.

- [ ] **Step 4: Run the full hooks + mutations test suites**

Run: `go test ./internal/hooks/... ./internal/mutations/... ./internal/characters/...`
Expected: PASS (no regressions; existing mob/user mutation tests still pass — old YAML has no cluster tags, so untagged mutations are universal and always eligible, preserving prior behavior for the seed content).

- [ ] **Step 5: Manual smoke**

Nuke instance saves (per SOP), boot, and confirm no panic and that a mob in combat still acquires:
```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . 2>&1 | grep -m1 "mutations.LoadMutationFiles"
```
Expected: loads clean, no panic. (Behavioral drift is not observable until content carries cluster tags — that's the follow-on content plan.)

- [ ] **Step 6: Commit**

```bash
git add internal/hooks/NewRound_MobRoundTick.go internal/hooks/NewRound_UserRoundTick.go
git commit -m "feat(hooks): acquisition ticks use graph pool + affinity decay"
```

---

## Phase 4 — Pole opposition

### Task 10: PoleDepth + Body Conviction scale

**Files:**
- Modify: `internal/mutations/graph.go` (PoleDepth)
- Create: `internal/mutations/opposition.go`
- Modify: `config.balance.go` + `config.balance.progression.go` (knobs)
- Test: `internal/mutations/opposition_test.go`

- [ ] **Step 1: Add knobs**

In `config.balance.go`:

```go
	MutationBodyConvictionDecayMax ConfigFloat `yaml:"MutationBodyConvictionDecayMax"` // max fraction of ConvictionMax lost to deep Body commitment (default 0.9)
	MutationBeliefGearDecayMax     ConfigFloat `yaml:"MutationBeliefGearDecayMax"`     // max fraction of gear effectiveness lost to deep Belief commitment (default 0.9)
	MutationPoleDecayRef           ConfigFloat `yaml:"MutationPoleDecayRef"`           // pole-depth at which decay reaches half its max (default 60.0)
```

In `config.balance.progression.go`:

```go
	if b.MutationBodyConvictionDecayMax <= 0 || b.MutationBodyConvictionDecayMax > 1.0 {
		b.MutationBodyConvictionDecayMax = 0.9
	}
	if b.MutationBeliefGearDecayMax <= 0 || b.MutationBeliefGearDecayMax > 1.0 {
		b.MutationBeliefGearDecayMax = 0.9
	}
	if b.MutationPoleDecayRef <= 0 {
		b.MutationPoleDecayRef = 60.0
	}
```

- [ ] **Step 2: Write the failing test**

```go
// internal/mutations/opposition_test.go
package mutations

import (
	"math"
	"testing"
)

func TestPoleDepth(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"claws": {MutationId: "claws", Name: "Claws", Rarity: 3, Pole: "body"},
		"ghost": {MutationId: "ghost", Name: "Ghost", Rarity: 5, Pole: "belief"},
	})
	defer cleanup()
	owned := map[string]int{"claws": 2, "ghost": 1}
	if d := PoleDepth(owned, "body"); d != 6 { // 3*2
		t.Fatalf("body depth = %v, want 6", d)
	}
	if d := PoleDepth(owned, "belief"); d != 5 {
		t.Fatalf("belief depth = %v, want 5", d)
	}
}

func TestBodyConvictionScale_MonotonicallyShrinks(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"claws": {MutationId: "claws", Name: "Claws", Rarity: 10, Pole: "body"},
	})
	defer cleanup()
	none := BodyConvictionScale(map[string]int{})
	shallow := BodyConvictionScale(map[string]int{"claws": 1})
	deep := BodyConvictionScale(map[string]int{"claws": 4})
	if none != 1.0 {
		t.Fatalf("no body mutations -> scale 1.0, got %v", none)
	}
	if !(shallow < 1.0 && deep < shallow) {
		t.Fatalf("scale must shrink with depth: none=%v shallow=%v deep=%v", none, shallow, deep)
	}
	if deep < 0 {
		t.Fatalf("scale must stay >= 0, got %v", deep)
	}
	_ = math.MaxFloat64
}
```

- [ ] **Step 3: Run to verify it fails**

Run: `go test ./internal/mutations/ -run "TestPoleDepth|TestBodyConvictionScale" -v`
Expected: FAIL — undefined `PoleDepth`, `BodyConvictionScale`.

- [ ] **Step 4: Implement**

In `internal/mutations/graph.go`:

```go
// PoleDepth is the summed rarity×level of owned mutations on the given pole
// ("body" or "belief"). Drives the opposition decay curves.
func PoleDepth(owned map[string]int, pole string) float64 {
	var d float64
	for id, lvl := range owned {
		spec := GetMutation(id)
		if spec == nil || spec.Pole != pole {
			continue
		}
		d += float64(spec.Rarity) * float64(lvl)
	}
	return d
}
```

In `internal/mutations/opposition.go`:

```go
package mutations

import "github.com/GoMudEngine/GoMud/internal/configs"

// poleScale returns a multiplier in [1-maxDecay, 1.0] that shrinks as pole
// depth grows: scale = 1 - maxDecay * depth/(depth+ref). Flat near zero,
// asymptotic toward the floor at extreme depth.
func poleScale(depth, maxDecay, ref float64) float64 {
	if depth <= 0 {
		return 1.0
	}
	frac := depth / (depth + ref)
	return 1.0 - maxDecay*frac
}

// BodyConvictionScale is the multiplier applied to ConvictionMax based on how
// deep the character has committed to the Body pole (chokes spells, taunt,
// and summons together — all Conviction-fuelled).
func BodyConvictionScale(owned map[string]int) float64 {
	b := configs.GetBalanceConfig()
	return poleScale(PoleDepth(owned, "body"),
		float64(b.MutationBodyConvictionDecayMax), float64(b.MutationPoleDecayRef))
}

// BeliefGearScale is the multiplier applied to gear effectiveness based on how
// deep the character has committed to the Belief pole (weapons/armor grow
// ornamental — extends incorporeal's gear_effectiveness_loss to the whole pole).
func BeliefGearScale(owned map[string]int) float64 {
	b := configs.GetBalanceConfig()
	return poleScale(PoleDepth(owned, "belief"),
		float64(b.MutationBeliefGearDecayMax), float64(b.MutationPoleDecayRef))
}
```

- [ ] **Step 5: Run to verify it passes**

Run: `go test ./internal/mutations/ -run "TestPoleDepth|TestBodyConvictionScale" -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/mutations/graph.go internal/mutations/opposition.go internal/mutations/opposition_test.go internal/configs/config.balance.go internal/configs/config.balance.progression.go
git commit -m "feat(mutations): pole-depth + Body/Belief opposition decay curves"
```

---

### Task 11: Apply Body Conviction decay in RecalculateStats

**Files:**
- Modify: `internal/characters/validate.go:107-125`
- Test: `internal/characters/godfunc_refactor_test.go` (new test alongside the pool-max tests)

- [ ] **Step 1: Write the failing test**

```go
// internal/characters/godfunc_refactor_test.go (append)
func TestRecalculateStats_BodyPoleShrinksConviction(t *testing.T) {
	cleanup := mutations.SeedMutationsForTest(map[string]*mutations.MutationSpec{
		"brute": {MutationId: "brute", Name: "Brute", Rarity: 10, Pole: "body"},
	})
	defer cleanup()

	base := newTestCharacterWithStats(t) // helper used by the other RecalculateStats tests
	base.RecalculateStats()
	full := base.ConvictionMax.Value

	base.Mutations = map[string]int{"brute": 4}
	base.RecalculateStats()
	shrunk := base.ConvictionMax.Value

	if !(shrunk < full) {
		t.Fatalf("deep Body should shrink ConvictionMax: full=%d shrunk=%d", full, shrunk)
	}
	if shrunk < 1 {
		t.Fatalf("ConvictionMax must stay floored at 1, got %d", shrunk)
	}
}
```

> Note: reuse whatever character-construction helper the sibling tests in this file use (e.g. the setup in `TestRecalculateStats_PoolMaxDerivation`). If none is factored out, inline the same construction here.

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/characters/ -run TestRecalculateStats_BodyPoleShrinksConviction -v`
Expected: FAIL — Conviction unchanged (decay not applied yet).

- [ ] **Step 3: Implement**

In `internal/characters/validate.go`, after the `health_multiplier` block (line ~112) and BEFORE the pool floors (line ~120):

```go
	// Mutation graph: deep Body-pole commitment shrinks the Conviction pool
	// (chokes spells, taunt, and summons — all Conviction-fuelled). Mirror of
	// the Belief pole's gear-effectiveness decay. Applied before the floor.
	if cScale := mutations.BodyConvictionScale(c.Mutations); cScale < 1.0 {
		c.ConvictionMax.Value = int(float64(c.ConvictionMax.Value) * cScale)
	}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/characters/ -run TestRecalculateStats_BodyPoleShrinksConviction -v`
Expected: PASS

- [ ] **Step 5: Run the full characters suite (no regressions)**

Run: `go test ./internal/characters/...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/characters/validate.go internal/characters/godfunc_refactor_test.go
git commit -m "feat(characters): deep Body-pole mutation commitment shrinks ConvictionMax"
```

---

### Task 12: Fold Belief gear decay into GearEffectivenessMultiplier

**Files:**
- Modify: `internal/mutations/mutations.go:649-654` (`GearEffectivenessMultiplier`)
- Test: `internal/mutations/opposition_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/mutations/opposition_test.go (append)
func TestGearEffectivenessMultiplier_FoldsBeliefDecay(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"ghost": {MutationId: "ghost", Name: "Ghost", Rarity: 10, Pole: "belief"},
	})
	defer cleanup()
	none := GearEffectivenessMultiplier(map[string]int{})
	deep := GearEffectivenessMultiplier(map[string]int{"ghost": 4})
	if none != 1.0 {
		t.Fatalf("no belief mutations -> full gear effectiveness, got %v", none)
	}
	if !(deep < none) {
		t.Fatalf("deep Belief should reduce gear effectiveness: none=%v deep=%v", none, deep)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/mutations/ -run TestGearEffectivenessMultiplier_FoldsBeliefDecay -v`
Expected: FAIL — deep equals none (belief decay not folded in).

- [ ] **Step 3: Implement**

In `internal/mutations/mutations.go`, change `GearEffectivenessMultiplier`:

```go
// GearEffectivenessMultiplier returns the multiplier consumers apply to
// gear-derived values (1.0 = full effectiveness, 0.0 = none). Combines the
// per-mutation gear_effectiveness_loss (incorporeal) with the Belief-pole
// opposition decay (deep casters render worn gear ornamental).
func GearEffectivenessMultiplier(owned map[string]int) float64 {
	return (1.0 - GetGearEffectivenessLoss(owned)) * BeliefGearScale(owned)
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/mutations/ -run TestGearEffectivenessMultiplier_FoldsBeliefDecay -v`
Expected: PASS

- [ ] **Step 5: Run the full mutations suite**

Run: `go test ./internal/mutations/...`
Expected: PASS (existing incorporeal gear tests still pass — with no belief-pole tag on the fixture, `BeliefGearScale` returns 1.0 and the product is unchanged).

- [ ] **Step 6: Commit**

```bash
git add internal/mutations/mutations.go internal/mutations/opposition_test.go
git commit -m "feat(mutations): Belief-pole depth folds into gear effectiveness decay"
```

---

## Phase 5 — Full build & docs

### Task 13: Full suite, boot smoke, and config documentation

**Files:**
- Modify: `_datafiles/config.yaml` (document the new knobs with defaults)
- Modify: `internal/mutations/README.md` if present (grep: `ls internal/mutations/*.md`) — else skip
- Test: full suite + boot

- [ ] **Step 1: Add the new knobs to the shipped config with defaults**

In `_datafiles/config.yaml`, under the `Balance:` section (alongside the existing `Mutation*` keys), add:

```yaml
    MutationAffinityPerSkillUse: 1.0
    MutationAffinityPerRarity: 1.0
    MutationAffinityDecay: 0.98
    MutationBodyConvictionDecayMax: 0.9
    MutationBeliefGearDecayMax: 0.9
    MutationPoleDecayRef: 60.0
```

- [ ] **Step 2: Run the complete affected suites**

Run: `go test ./internal/mutations/... ./internal/characters/... ./internal/hooks/... ./internal/configs/...`
Expected: PASS

- [ ] **Step 3: Full build**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 4: Boot smoke (SOP)**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . 2>&1 | grep -mE1 "mutations.LoadMutationFiles|panic"
```
Expected: `mutations.LoadMutationFiles() loadedCount=...` prints; no panic. Ctrl-C after.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/config.yaml
git commit -m "chore(config): document Chrysalis mutation-graph engine knobs"
```

---

## Self-Review (completed during authoring)

- **Spec coverage:** §3 graph data (Tasks 1–2), §4 keystone/prereq gating (Tasks 1, 7, 8), §5 drift/affinity engine (Tasks 3–9), §6 opposition/pool-decay (Tasks 10–12), config knobs (all phases). **Deliberately out of this plan** (spec §7–§11, follow-on): phials, migration/re-bloom, mob archetype seeding + `archetype_pull` re-curation, legibility/helpfiles, per-cluster content, and the damage-absorbed→Ironhide signal.
- **Placeholder scan:** none — every code step carries real Go.
- **Type consistency:** `GetGraphPool(owned, aff, sp)`, `EffectiveAffinity(owned, actionAff)`, `DecayAffinity(map, rate)`, `PoleDepth(owned, pole)`, `BodyConvictionScale(owned)`, `BeliefGearScale(owned)`, `ClustersForSkill(skill)`, `PrereqsMet(owned, spec)`, `AddClusterAffinity(cluster, amount)` — signatures used consistently across tasks and call sites.

---

## Follow-on plans (NOT in this plan)

1. **Content authoring** — the 9 clusters × ~3–4 keystones (effects, prereq spines, bridge nodes, apex transformations) as YAML carrying the new `clusters`/`pole`/`prerequisites` fields; Ravener + Generalist from the spec are the templates. This is what makes the engine's drift observable in play.
2. **Phials** — ~9 alchemy recipes with a flavoring step; drink = strip-others + re-bloom onto a cluster; unflavored → Generalist (grants Hollow Bones).
3. **Migration** — clean break: retire old 41, wipe ownership, one-time skill-seeded free re-bloom, and a reference sweep (items/mobs/spells that named old mutation ids) with boot validators.
4. **Mob integration** — spawn-time cluster affinity seeded from archetype; re-curate the provisional `archetype_pull` table into `clusters` tags feeding the existing `ReevaluateArchetypeShift`.
5. **Ironhide tank signal** — wire damage-absorbed → `AddClusterAffinity("ironhide", …)` at the damage-pipeline defender site.
6. **Helpfiles** — teach the shape + the worked prereq examples (opaque-but-hinted).
