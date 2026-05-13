# Mob Aliveness 1.6 — NPC Relationships Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship `internal/relationships/` — a per-mob-template kinship/friendship/rivalry/lover/employer-employee graph authored on each mob's YAML, built into an in-memory graph at startup with auto-mirror, plus a read+mutation API and an admin command. Also backfill missing `context.md` files for chunks 1.1–1.5.

**Architecture:** Mob YAMLs gain an optional `relationships:` field. After `mobs.LoadDataFiles()` completes, a new `relationships.LoadFromMobs()` call walks every loaded mob template, extracts edges, runs validation, and builds an in-memory graph (with auto-mirror for symmetric and asymmetric edge types). Lookup is O(1) by mob id; mutation is in-memory only v1.

**Tech Stack:** Go 1.21+, YAML via `gopkg.in/yaml.v3` (already used by mob loader), no new external dependencies.

**Spec:** `docs/superpowers/specs/2026-05-09-mob-aliveness-1.6-npc-relationships-design.md`

**Branch:** Stay on `feature/mob-aliveness-1.3-crimes` per the single-feature-branch SOP for the aliveness effort.

---

## File structure

| File | Responsibility |
|------|----------------|
| `internal/relationships/types.go` | `Type`, `Relation`, internal graph state |
| `internal/relationships/graph.go` | Graph builder, auto-mirror logic, validation |
| `internal/relationships/relationships.go` | Public API: read queries + mutations |
| `internal/relationships/test_main_test.go` | TestMain (no temp dir; pure in-memory) |
| `internal/relationships/types_test.go` | Type-level tests |
| `internal/relationships/graph_test.go` | Build / auto-mirror / validation tests |
| `internal/relationships/relationships_test.go` | Read / mutation API tests |
| `internal/relationships/context.md` | Package documentation (chunk 1.6) |
| `internal/mobs/mobs.go` | Add `Relationships []RelationshipYAMLEntry` field on Mob; new struct definition |
| `internal/mobs/mobs.go` (LoadDataFiles area, ~line 1124) | Call `relationships.LoadFromMobs(...)` after the mob load completes |
| `internal/usercommands/admin.relationship.go` | Admin command |
| `internal/usercommands/usercommands.go` | Register admin command |
| `_datafiles/world/dogmud/templates/admincommands/help/command.relationship.template` | Admin helpfile |
| `internal/opinions/context.md` | Chunk 1.1 backfill |
| `internal/factions/context.md` | Chunk 1.2 backfill |
| `internal/crimes/context.md` | Chunk 1.3 backfill |
| `internal/knowledge/context.md` | Chunk 1.4 backfill |
| `internal/bounties/context.md` | Chunk 1.5 backfill |
| `MOB_ALIVENESS_ROADMAP.md` | Mark 1.6 Done, roll-up to 6/40 |

---

## Task 1: Backfill `internal/opinions/context.md` (chunk 1.1)

**Files:**
- Create: `internal/opinions/context.md`

The opinions package was the chunk-1.1 ship. Its public API is captured in `internal/opinions/opinions.go`; persistence in `persistence.go`; decay in `decay.go`. The 1.1 spec is at `docs/superpowers/specs/2026-05-06-mob-aliveness-1.1-opinion-store-design.md`.

- [ ] **Step 1: Read the package**

```bash
ls internal/opinions/
wc -l internal/opinions/*.go
```

Read the .go files: `types.go`, `opinions.go`, `persistence.go`, `decay.go`. Also skim `internal/badinputtracker/context.md` and `internal/clans/context.md` for the established style.

- [ ] **Step 2: Write `internal/opinions/context.md`**

Required sections (per the roadmap rule):

- **Overview** (1 paragraph): What the package does. The chunk-1.1 spec abstract is the source.
- **Key Components**: file map listing each .go file with its responsibility.
- **Key Functions**: signatures + behavior for the public API. Cover Get/Set/Bump/Reset/AllRowsForUser/TierFor/Tier and the decay helper.
- **Global State**: cache map + mutex + decay seam.
- **Data Structure Design**: per-NPC YAML shape at `_datafiles/world/dogmud/opinions/{mobId}-{namesimple}.yaml`. Show a sample.
- **Integration Notes**: which packages call into opinions (combat first-aggression, dialogue mood, admin command). Which packages opinions imports (configs, mobs, util, mudlog).
- **Testing Notes**: brief pointer to `*_test.go` files and TestMain temp-dir pattern.

Write in the same prose style as `internal/badinputtracker/context.md` — informative but not chatty. ~150–250 lines.

- [ ] **Step 3: Verify Markdown renders**

Open the file and skim. No broken code fences, headings hierarchy makes sense, no truncation.

- [ ] **Step 4: Commit**

```bash
git add internal/opinions/context.md
git commit -m "docs(opinions): context.md backfill (chunk 1.1)"
```

---

## Task 2: Backfill `internal/factions/context.md` (chunk 1.2)

**Files:**
- Create: `internal/factions/context.md`

The factions package was the chunk-1.2 ship. Read `internal/factions/*.go` first. Spec is at `docs/superpowers/specs/2026-05-06-mob-aliveness-1.2-faction-system-design.md`.

- [ ] **Step 1: Read the package**

```bash
ls internal/factions/
```

Read the .go files. Note the dual-source pattern: faction definitions in committed YAML at `_datafiles/world/dogmud/factions/{slug}.yaml`; per-player rep state in gitignored YAML at `_datafiles/world/dogmud/factions.rep/{slug}.yaml`.

- [ ] **Step 2: Write the file**

Same section structure as Task 1. Cover:
- Definition vs rep state (two separate YAMLs per faction).
- Public API: `GetDefinition`, `AllDefinitions`, `GetRep`, `SetRep`, `BumpRep`, `TierFor`, `FactionsForMob`, `IsPeacefulToward`.
- The ally/enemy graph + validation at load.
- `bump_rep` quest engine action.
- Admin command `faction list/show/set/bump/reset`.
- Combat hookup `MobDeath_FactionRep` (chunk 1.2 + 1.3 four-case rewrite).

~200–300 lines.

- [ ] **Step 3: Verify + commit**

```bash
git add internal/factions/context.md
git commit -m "docs(factions): context.md backfill (chunk 1.2)"
```

---

## Task 3: Backfill `internal/crimes/context.md` (chunk 1.3)

**Files:**
- Create: `internal/crimes/context.md`

Cover the chunk-1.3 substrate (assault/murder/theft, witness model, 4-case upgrade with rep refund). Read `internal/crimes/*.go`. Spec at `docs/superpowers/specs/2026-05-06-mob-aliveness-1.3-crime-wanted-design.md`.

- [ ] **Step 1: Read the package**

- [ ] **Step 2: Write the file**

Sections:
- Overview: per-faction crime log with witness-required perp identification.
- Files: `types.go`, `crimes.go`, `persistence.go`.
- Public API: `Record`, `Resolve`, `FindRecentAssault`, `AllForFaction`, `AllForPlayer`, `PruneStale`, `WitnessesInRoom`, `IdentifiedPerp`, `UpgradeAssaultToMurder`, the four-case logic in `MobDeath_FactionRep`.
- Data: per-faction YAML at `_datafiles/world/dogmud/factions.crimes/{slug}.yaml`. Include the `had_external_witness` flag the post-shipping fix added.
- Integration: combat call sites (attack.go, MobDeath_FactionRep.go, skill.skullduggery.steal.go).
- Admin command `crime list/show/resolve/prune-stale`.

~250–350 lines.

- [ ] **Step 3: Verify + commit**

```bash
git add internal/crimes/context.md
git commit -m "docs(crimes): context.md backfill (chunk 1.3)"
```

---

## Task 4: Backfill `internal/knowledge/context.md` (chunk 1.4)

**Files:**
- Create: `internal/knowledge/context.md`

Cover the chunk-1.4 substrate. Read `internal/knowledge/*.go`. Spec at `docs/superpowers/specs/2026-05-09-mob-aliveness-1.4-knowledge-model-design.md`.

- [ ] **Step 1: Read the package**

- [ ] **Step 2: Write the file**

Sections:
- Overview: per-NPC × per-subject knowledge store; polymorphic Subject; auto-write triggers.
- Files: `types.go`, `persistence.go`, `knowledge.go`, `decay.go`.
- Public API: RecordMet, RecordObservation, RecordName, RecordCrimeWitnessed, Forget, ForgetFact, Get, HasMet, NameOf, LastSeen, FrequentedRooms, WitnessedCrimes, AllForObserver, AllObserversOfPlayer, RecordRoutineObservers, PlayerSubject, MobSubject.
- Data: per-observer-template YAML at `_datafiles/world/dogmud/knowledge/{mobId}-{namesimple}.yaml`.
- Decay table per fact type.
- Integration: forager/caravan room change hook (`MobRoomChange_KnowledgeObservers`), 1.3 crime-witnessing call sites.
- Admin command `knowledge show/forget/frequented`.
- Cross-substrate intersections (lazy filter on read for crime IDs; no cross-cascade).

~300–400 lines (largest of the substrate context.md files because the package is the densest).

- [ ] **Step 3: Verify + commit**

```bash
git add internal/knowledge/context.md
git commit -m "docs(knowledge): context.md backfill (chunk 1.4)"
```

---

## Task 5: Backfill `internal/bounties/context.md` (chunk 1.5)

**Files:**
- Create: `internal/bounties/context.md`

Cover the chunk-1.5 substrate (just shipped). Read `internal/bounties/*.go`. Spec at `docs/superpowers/specs/2026-05-09-mob-aliveness-1.5-bounty-state-design.md`.

- [ ] **Step 1: Read the package**

- [ ] **Step 2: Write the file**

Sections:
- Overview: single-registry bounty substrate with three issuer types and polymorphic target.
- Files: `types.go`, `persistence.go`, `bounties.go`.
- Public API: Declare, Withdraw, TryClaim, MarkExpired, PruneExpired, Get, AllOpen, OpenForTarget, OpenForIssuer, OpenAgainstPlayer, AllForTarget, AllRows, FactionIssuer, QuestIssuer, NPCIssuer.
- Data: single registry YAML at `_datafiles/world/dogmud/bounties.yaml`.
- Integration: `MobDeath_BountyClaim` auto-claim hook, `declare_bounty` quest engine action, single `bounty` command with role-gated subcommands, two physical bounty boards (Thornwall 473 + Stillwater 4110).
- Reward auto-compute: gold = floor(statpool × 0.5), rep = max(1, statpool/100).

~200–300 lines.

- [ ] **Step 3: Verify + commit**

```bash
git add internal/bounties/context.md
git commit -m "docs(bounties): context.md backfill (chunk 1.5)"
```

---

## Task 6: Package skeleton + types

**Files:**
- Create: `internal/relationships/types.go`
- Create: `internal/relationships/types_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/relationships/types_test.go
package relationships

import "testing"

func TestTypeConstants(t *testing.T) {
	cases := []struct {
		t    Type
		want string
	}{
		{TypeFamily, "family"},
		{TypeFriend, "friend"},
		{TypeRival, "rival"},
		{TypeLover, "lover"},
		{TypeEmployer, "employer"},
		{TypeEmployee, "employee"},
	}
	for _, c := range cases {
		if string(c.t) != c.want {
			t.Errorf("Type %s: got %s, want %s", c.want, c.t, c.want)
		}
	}
}

func TestIsSymmetric(t *testing.T) {
	cases := []struct {
		t    Type
		want bool
	}{
		{TypeFamily, true},
		{TypeFriend, true},
		{TypeRival, true},
		{TypeLover, true},
		{TypeEmployer, false},
		{TypeEmployee, false},
	}
	for _, c := range cases {
		if got := IsSymmetric(c.t); got != c.want {
			t.Errorf("IsSymmetric(%s): got %v, want %v", c.t, got, c.want)
		}
	}
}

func TestInverseType(t *testing.T) {
	if InverseType(TypeEmployer) != TypeEmployee {
		t.Errorf("InverseType(employer) should be employee")
	}
	if InverseType(TypeEmployee) != TypeEmployer {
		t.Errorf("InverseType(employee) should be employer")
	}
	// Symmetric types invert to themselves.
	if InverseType(TypeFamily) != TypeFamily {
		t.Errorf("InverseType(family) should be family")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/relationships/...`
Expected: build failure (package doesn't exist).

- [ ] **Step 3: Implement types.go**

```go
// internal/relationships/types.go
package relationships

type Type string

const (
	TypeFamily   Type = "family"
	TypeFriend   Type = "friend"
	TypeRival    Type = "rival"
	TypeLover    Type = "lover"
	TypeEmployer Type = "employer" // I am their employer
	TypeEmployee Type = "employee" // I am their employee (auto-mirror only)
)

// IsSymmetric returns true for relationship types where the reverse
// edge has the same type (family, friend, rival, lover) and false
// for asymmetric pairs (employer/employee).
func IsSymmetric(t Type) bool {
	switch t {
	case TypeFamily, TypeFriend, TypeRival, TypeLover:
		return true
	}
	return false
}

// InverseType returns the auto-mirror reverse type. Symmetric types
// return themselves; asymmetric pairs flip.
func InverseType(t Type) Type {
	switch t {
	case TypeEmployer:
		return TypeEmployee
	case TypeEmployee:
		return TypeEmployer
	}
	return t // symmetric: same type
}

// Relation is one outgoing edge from a mob to another mob.
type Relation struct {
	Other   int    // mob template id of the other party
	Type    Type
	Subtype string // optional flavor; "" if unset
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/relationships/...`
Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/relationships/types.go internal/relationships/types_test.go
git commit -m "feat(relationships): types + IsSymmetric/InverseType (chunk 1.6 T6)"
```

---

## Task 7: TestMain harness + graph state

**Files:**
- Create: `internal/relationships/test_main_test.go`
- Create: `internal/relationships/graph.go`

- [ ] **Step 1: Write test_main_test.go**

```go
// internal/relationships/test_main_test.go
package relationships

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// In-memory only; no temp dir needed (no v1 persistence).
	os.Exit(m.Run())
}

// resetGraph wipes the in-memory graph between tests.
func resetGraph() {
	graphMu.Lock()
	graph = make(map[int][]Relation)
	graphMu.Unlock()
}
```

- [ ] **Step 2: Implement graph.go skeleton**

```go
// internal/relationships/graph.go
package relationships

import "sync"

// graph is the in-memory adjacency map: mob template id → outgoing
// edges. Edges are stored on both sides (auto-mirrored at load time
// or via Add) so callers always see a complete picture from
// whichever side they query.
var (
	graph   = make(map[int][]Relation)
	graphMu sync.RWMutex
)
```

- [ ] **Step 3: Verify build**

Run: `go test ./internal/relationships/...`
Expected: PASS (T6 tests still pass; new code compiles).

- [ ] **Step 4: Commit**

```bash
git add internal/relationships/test_main_test.go internal/relationships/graph.go
git commit -m "feat(relationships): test harness + graph state (T7)"
```

---

## Task 8: Auto-mirror at build time

**Files:**
- Modify: `internal/relationships/graph.go`
- Create: `internal/relationships/graph_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/relationships/graph_test.go
package relationships

import (
	"testing"
)

// edgeSpec is a test-only authoring shape: "mob X declares this edge."
type edgeSpec struct {
	From    int
	To      int
	Type    Type
	Subtype string
}

func buildFromSpecs(specs []edgeSpec) {
	resetGraph()
	graphMu.Lock()
	defer graphMu.Unlock()
	for _, s := range specs {
		// Forward edge.
		graph[s.From] = append(graph[s.From], Relation{Other: s.To, Type: s.Type, Subtype: s.Subtype})
		// Mirror edge.
		mirror := Relation{Other: s.From, Type: InverseType(s.Type)}
		// Skip if mirror already exists (de-dup).
		exists := false
		for _, r := range graph[s.To] {
			if r.Other == s.From && r.Type == mirror.Type {
				exists = true
				break
			}
		}
		if !exists {
			graph[s.To] = append(graph[s.To], mirror)
		}
	}
}

func TestAutoMirror_Symmetric(t *testing.T) {
	buildFromSpecs([]edgeSpec{
		{From: 95, To: 96, Type: TypeFriend, Subtype: "drinking-companion"},
	})

	// 95 has friend → 96.
	if got := edgesOf(95); len(got) != 1 || got[0].Other != 96 || got[0].Type != TypeFriend {
		t.Errorf("95: got %v", got)
	}
	if got := edgesOf(95)[0].Subtype; got != "drinking-companion" {
		t.Errorf("95 subtype: got %q", got)
	}

	// 96 has friend → 95 (auto-mirrored, no subtype).
	if got := edgesOf(96); len(got) != 1 || got[0].Other != 95 || got[0].Type != TypeFriend {
		t.Errorf("96: got %v", got)
	}
	if got := edgesOf(96)[0].Subtype; got != "" {
		t.Errorf("96 subtype should be empty (per-side), got %q", got)
	}
}

func TestAutoMirror_Asymmetric(t *testing.T) {
	buildFromSpecs([]edgeSpec{
		{From: 95, To: 248, Type: TypeEmployer, Subtype: "priest-handyman"},
	})

	// 95: employer → 248.
	if got := edgesOf(95); got[0].Type != TypeEmployer {
		t.Errorf("95 should be employer: got %s", got[0].Type)
	}
	// 248: employee → 95 (auto-mirrored inverse).
	if got := edgesOf(248); got[0].Type != TypeEmployee {
		t.Errorf("248 should be employee: got %s", got[0].Type)
	}
}

// edgesOf is a test helper that takes the cache RLock.
func edgesOf(mobId int) []Relation {
	graphMu.RLock()
	defer graphMu.RUnlock()
	out := make([]Relation, len(graph[mobId]))
	copy(out, graph[mobId])
	return out
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/relationships/...`
Expected: FAIL (`InverseType` is defined but the test references logic that doesn't exist standalone — actually this test should already pass via `buildFromSpecs`. Run it and see).

If the test passes already, that means the inline build logic exercises the auto-mirror correctly via the helpers. If it fails, debug.

- [ ] **Step 3: Run tests to verify they pass**

Run: `go test ./internal/relationships/...`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/relationships/graph_test.go
git commit -m "feat(relationships): auto-mirror at build (symmetric + asymmetric) (T8)"
```

---

## Task 9: LoadFromMobs + validation

**Files:**
- Modify: `internal/relationships/graph.go`
- Modify: `internal/relationships/graph_test.go`

This task wires up the public load entry point. It takes a slice of (mobId, edges) tuples — the relationships package doesn't import mobs directly; the caller (T12 will do this in `mobs.LoadDataFiles`) hands over the flattened data.

- [ ] **Step 1: Add an authoring-input type to graph.go**

```go
// MobEdges is the flattened input for LoadFromMobs: one entry per
// mob template that has authored relationships.
type MobEdges struct {
	MobId int
	Edges []EdgeInput
}

// EdgeInput is the per-edge authoring shape. Mirrors the YAML on
// the mob template (To, Type, Subtype) but pre-typed.
type EdgeInput struct {
	To      int
	Type    Type
	Subtype string
}
```

- [ ] **Step 2: Write failing tests for validation policy**

Append to `graph_test.go`:

```go
import (
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

func TestLoadFromMobs_HappyPath(t *testing.T) {
	resetGraph()
	LoadFromMobs([]MobEdges{
		{MobId: 95, Edges: []EdgeInput{
			{To: 96, Type: TypeFriend, Subtype: "drinking-companion"},
			{To: 117, Type: TypeFamily, Subtype: "niece"},
			{To: 248, Type: TypeEmployer, Subtype: "priest-handyman"},
		}},
		{MobId: 96, Edges: []EdgeInput{}}, // mob with no relationships
	}, alwaysValid)

	if got := len(edgesOf(95)); got != 3 {
		t.Errorf("95 should have 3 edges, got %d", got)
	}
	// 96 mirror of 95's friend edge.
	if got := edgesOf(96); len(got) != 1 || got[0].Other != 95 || got[0].Type != TypeFriend {
		t.Errorf("96 mirror: %v", got)
	}
	// 248 mirror as employee.
	if got := edgesOf(248); len(got) != 1 || got[0].Other != 95 || got[0].Type != TypeEmployee {
		t.Errorf("248 mirror: %v", got)
	}
}

func TestLoadFromMobs_UnknownTargetSkipped(t *testing.T) {
	resetGraph()
	// Mob 99 doesn't exist (validate returns false).
	validate := func(id int) bool { return id == 95 }
	LoadFromMobs([]MobEdges{
		{MobId: 95, Edges: []EdgeInput{
			{To: 99, Type: TypeFriend},
		}},
	}, validate)

	if got := len(edgesOf(95)); got != 0 {
		t.Errorf("edge to unknown mob should be skipped, got %d edges", got)
	}
}

func TestLoadFromMobs_SelfEdgeSkipped(t *testing.T) {
	resetGraph()
	LoadFromMobs([]MobEdges{
		{MobId: 95, Edges: []EdgeInput{
			{To: 95, Type: TypeFriend},
		}},
	}, alwaysValid)

	if got := len(edgesOf(95)); got != 0 {
		t.Errorf("self-edge should be skipped, got %d edges", got)
	}
}

func TestLoadFromMobs_UnknownTypeSkipped(t *testing.T) {
	resetGraph()
	LoadFromMobs([]MobEdges{
		{MobId: 95, Edges: []EdgeInput{
			{To: 96, Type: Type("nonsense")},
		}},
	}, alwaysValid)

	if got := len(edgesOf(95)); got != 0 {
		t.Errorf("unknown-type edge should be skipped, got %d", got)
	}
}

func TestLoadFromMobs_DuplicateEdgesDeduped(t *testing.T) {
	resetGraph()
	LoadFromMobs([]MobEdges{
		{MobId: 95, Edges: []EdgeInput{
			{To: 96, Type: TypeFriend, Subtype: "first"},
			{To: 96, Type: TypeFriend, Subtype: "second"},
		}},
	}, alwaysValid)

	got := edgesOf(95)
	if len(got) != 1 {
		t.Errorf("duplicate edges should dedupe, got %d", len(got))
	}
	if got[0].Subtype != "first" {
		t.Errorf("first declaration wins for subtype, got %q", got[0].Subtype)
	}
}

// alwaysValid is a test helper that approves every mob id.
func alwaysValid(int) bool { return true }

func init() {
	// Quiet warnings during validation tests.
	_ = mudlog.SetLogLevel("error")
}
```

If `mudlog.SetLogLevel` doesn't exist in this codebase, drop the init() block — warnings during tests are noise but not blocking.

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/relationships/...`
Expected: FAIL (LoadFromMobs undefined).

- [ ] **Step 4: Implement LoadFromMobs in graph.go**

```go
import (
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// LoadFromMobs populates the in-memory graph from authored mob
// edges. `validateMobId` is a callback the caller provides to
// confirm a mob template id exists (the relationships package
// doesn't import mobs directly to avoid coupling). Validation:
//   - to: == declaring mobId  → warn + skip (self-edge)
//   - to: not validateMobId   → warn + skip (unknown target)
//   - type: not in known enum → warn + skip
//   - duplicate edges (same from/to/type) → warn + first wins
//   - pair conflicts (A says friend, B says rival) → both kept
//     as-declared; warn
func LoadFromMobs(input []MobEdges, validateMobId func(mobId int) bool) {
	graphMu.Lock()
	defer graphMu.Unlock()

	graph = make(map[int][]Relation)

	knownTypes := map[Type]bool{
		TypeFamily: true, TypeFriend: true, TypeRival: true,
		TypeLover: true, TypeEmployer: true, TypeEmployee: true,
	}

	for _, me := range input {
		for _, ei := range me.Edges {
			if ei.To == me.MobId {
				mudlog.Warn("relationships: self-edge skipped",
					"mobId", me.MobId)
				continue
			}
			if !validateMobId(ei.To) {
				mudlog.Warn("relationships: edge to unknown mob skipped",
					"from", me.MobId, "to", ei.To)
				continue
			}
			if !knownTypes[ei.Type] {
				mudlog.Warn("relationships: unknown type skipped",
					"from", me.MobId, "to", ei.To, "type", ei.Type)
				continue
			}
			// Forward edge with dedup.
			if hasEdge(me.MobId, ei.To, ei.Type) {
				mudlog.Warn("relationships: duplicate edge skipped",
					"from", me.MobId, "to", ei.To, "type", ei.Type)
				continue
			}
			graph[me.MobId] = append(graph[me.MobId],
				Relation{Other: ei.To, Type: ei.Type, Subtype: ei.Subtype})
			// Mirror edge (no subtype on the mirror; per-side).
			mirrorType := InverseType(ei.Type)
			if !hasEdge(ei.To, me.MobId, mirrorType) {
				graph[ei.To] = append(graph[ei.To],
					Relation{Other: me.MobId, Type: mirrorType})
			}
		}
	}
}

// hasEdge checks if from→to with the given type already exists.
// Caller must hold graphMu (write lock).
func hasEdge(from, to int, t Type) bool {
	for _, r := range graph[from] {
		if r.Other == to && r.Type == t {
			return true
		}
	}
	return false
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/relationships/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/relationships/graph.go internal/relationships/graph_test.go
git commit -m "feat(relationships): LoadFromMobs + validation policy (T9)"
```

---

## Task 10: Read API

**Files:**
- Create: `internal/relationships/relationships.go`
- Create: `internal/relationships/relationships_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/relationships/relationships_test.go
package relationships

import (
	"testing"
)

func seedSimpleGraph(t *testing.T) {
	t.Helper()
	resetGraph()
	LoadFromMobs([]MobEdges{
		{MobId: 95, Edges: []EdgeInput{
			{To: 96, Type: TypeFriend},
			{To: 117, Type: TypeFamily, Subtype: "niece"},
			{To: 113, Type: TypeRival},
			{To: 248, Type: TypeEmployer},
		}},
	}, alwaysValid)
}

func TestRelationsOf(t *testing.T) {
	seedSimpleGraph(t)
	if got := len(RelationsOf(95)); got != 4 {
		t.Errorf("RelationsOf(95): got %d, want 4", got)
	}
	if got := len(RelationsOf(404)); got != 0 {
		t.Errorf("RelationsOf(404 unknown): got %d, want 0", got)
	}
}

func TestRelationsOfType(t *testing.T) {
	seedSimpleGraph(t)
	if got := len(RelationsOfType(95, TypeFamily)); got != 1 {
		t.Errorf("family count: %d", got)
	}
	if got := len(RelationsOfType(95, TypeFamily, TypeFriend)); got != 2 {
		t.Errorf("family+friend count: %d", got)
	}
	// No types passed = all edges.
	if got := len(RelationsOfType(95)); got != 4 {
		t.Errorf("no-filter count: %d", got)
	}
}

func TestKinAlliesRivals(t *testing.T) {
	seedSimpleGraph(t)
	if got := len(KinOf(95)); got != 1 {
		t.Errorf("KinOf: %d", got)
	}
	if got := len(AlliesOf(95)); got != 2 {
		t.Errorf("AlliesOf (family+friend+lover): %d", got)
	}
	if got := len(RivalsOf(95)); got != 1 {
		t.Errorf("RivalsOf: %d", got)
	}
}

func TestRelationsBetween(t *testing.T) {
	seedSimpleGraph(t)
	got := RelationsBetween(95, 96)
	if len(got) != 1 || got[0].Other != 96 {
		t.Errorf("RelationsBetween(95, 96): %v", got)
	}
	if got := RelationsBetween(95, 999); len(got) != 0 {
		t.Errorf("RelationsBetween(95, 999): %v", got)
	}
}

func TestAreRelated(t *testing.T) {
	seedSimpleGraph(t)
	if !AreRelated(95, 96) {
		t.Errorf("95 and 96 should be related")
	}
	if AreRelated(95, 999) {
		t.Errorf("95 and 999 should not be related")
	}
}

func TestEmployerEmployedBy(t *testing.T) {
	seedSimpleGraph(t)
	emp := EmployerOf(95)
	if len(emp) != 1 || emp[0] != 248 {
		t.Errorf("EmployerOf(95): %v, want [248]", emp)
	}
	by := EmployedBy(248)
	if len(by) != 1 || by[0] != 95 {
		t.Errorf("EmployedBy(248): %v, want [95]", by)
	}
}

func TestAllRelations(t *testing.T) {
	seedSimpleGraph(t)
	all := AllRelations()
	// 4 forward edges from 95 + 4 auto-mirrored = 8 total entries.
	if len(all) != 8 {
		t.Errorf("AllRelations: got %d, want 8", len(all))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/relationships/...`
Expected: FAIL (functions undefined).

- [ ] **Step 3: Implement read API**

```go
// internal/relationships/relationships.go
package relationships

import "sort"

// RelationsOf returns all outgoing edges for a mob template.
func RelationsOf(mobId int) []Relation {
	graphMu.RLock()
	defer graphMu.RUnlock()
	out := make([]Relation, len(graph[mobId]))
	copy(out, graph[mobId])
	return out
}

// RelationsOfType filters by type. Empty types slice → all types.
func RelationsOfType(mobId int, types ...Type) []Relation {
	graphMu.RLock()
	defer graphMu.RUnlock()
	if len(types) == 0 {
		out := make([]Relation, len(graph[mobId]))
		copy(out, graph[mobId])
		return out
	}
	want := make(map[Type]bool, len(types))
	for _, t := range types {
		want[t] = true
	}
	out := make([]Relation, 0)
	for _, r := range graph[mobId] {
		if want[r.Type] {
			out = append(out, r)
		}
	}
	return out
}

// KinOf returns family + lover edges.
func KinOf(mobId int) []Relation {
	return RelationsOfType(mobId, TypeFamily, TypeLover)
}

// AlliesOf returns family + friend + lover edges.
func AlliesOf(mobId int) []Relation {
	return RelationsOfType(mobId, TypeFamily, TypeFriend, TypeLover)
}

// RivalsOf returns rival edges.
func RivalsOf(mobId int) []Relation {
	return RelationsOfType(mobId, TypeRival)
}

// RelationsBetween returns edges from a to b (a's view of the
// relationship). Empty if not connected.
func RelationsBetween(a, b int) []Relation {
	graphMu.RLock()
	defer graphMu.RUnlock()
	out := make([]Relation, 0)
	for _, r := range graph[a] {
		if r.Other == b {
			out = append(out, r)
		}
	}
	return out
}

// AreRelated returns true if any edge connects a to b.
func AreRelated(a, b int) bool {
	graphMu.RLock()
	defer graphMu.RUnlock()
	for _, r := range graph[a] {
		if r.Other == b {
			return true
		}
	}
	return false
}

// EmployerOf returns mob ids this NPC employs.
func EmployerOf(mobId int) []int {
	rels := RelationsOfType(mobId, TypeEmployer)
	out := make([]int, len(rels))
	for i, r := range rels {
		out[i] = r.Other
	}
	return out
}

// EmployedBy returns mob ids that employ this NPC.
func EmployedBy(mobId int) []int {
	rels := RelationsOfType(mobId, TypeEmployee)
	out := make([]int, len(rels))
	for i, r := range rels {
		out[i] = r.Other
	}
	return out
}

// AllRelations returns a flat snapshot of every edge in the graph.
// Each forward + mirror edge is represented separately. Sorted by
// owner mob id, then by Other.
type OwnedRelation struct {
	Owner int // the mob whose edge this is
	Relation
}

func AllRelations() []OwnedRelation {
	graphMu.RLock()
	defer graphMu.RUnlock()
	out := make([]OwnedRelation, 0)
	for owner, edges := range graph {
		for _, r := range edges {
			out = append(out, OwnedRelation{Owner: owner, Relation: r})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Owner != out[j].Owner {
			return out[i].Owner < out[j].Owner
		}
		return out[i].Other < out[j].Other
	})
	return out
}
```

NOTE on `AllRelations`: the test expects `len(AllRelations())` equal to 8 (4 forward + 4 mirror). Verify the return type matches. If the test was written assuming `[]Relation` (not `[]OwnedRelation`), adjust the return type or the test. Either is fine as long as they line up.

If the test fails because of the return-type mismatch, update the test to use `OwnedRelation`. That's the right shape because callers debugging the graph want to know which mob owns each edge.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/relationships/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/relationships/
git commit -m "feat(relationships): read API (T10)"
```

---

## Task 11: Mutation API

**Files:**
- Modify: `internal/relationships/relationships.go`
- Modify: `internal/relationships/relationships_test.go`

- [ ] **Step 1: Write failing tests**

```go
func TestAdd(t *testing.T) {
	resetGraph()
	Add(95, 96, TypeFriend, "new-friend")

	if !AreRelated(95, 96) {
		t.Errorf("after Add, 95 and 96 should be related")
	}
	if got := edgesOf(96); len(got) != 1 || got[0].Type != TypeFriend {
		t.Errorf("auto-mirror missing on 96: %v", got)
	}

	// Add of identical edge is no-op.
	Add(95, 96, TypeFriend, "different-subtype")
	if got := len(edgesOf(95)); got != 1 {
		t.Errorf("duplicate Add should be no-op: %d edges", got)
	}
}

func TestAdd_Asymmetric(t *testing.T) {
	resetGraph()
	Add(95, 248, TypeEmployer, "")
	if got := edgesOf(95); got[0].Type != TypeEmployer {
		t.Errorf("95 should have employer edge")
	}
	if got := edgesOf(248); got[0].Type != TypeEmployee {
		t.Errorf("248 should have employee mirror")
	}
}

func TestRemove(t *testing.T) {
	resetGraph()
	Add(95, 96, TypeFriend, "")
	Remove(95, 96, TypeFriend)
	if AreRelated(95, 96) {
		t.Errorf("after Remove, 95 and 96 should not be related")
	}
	// Mirror removed too.
	if len(edgesOf(96)) != 0 {
		t.Errorf("mirror should be removed too")
	}
}

func TestChangeType(t *testing.T) {
	resetGraph()
	Add(95, 96, TypeFriend, "")
	ChangeType(95, 96, TypeFriend, TypeRival, "fell-out")

	rels := RelationsBetween(95, 96)
	if len(rels) != 1 || rels[0].Type != TypeRival {
		t.Errorf("after ChangeType: %v", rels)
	}
	if rels[0].Subtype != "fell-out" {
		t.Errorf("subtype: %q", rels[0].Subtype)
	}
	// Mirror also changed.
	if mrels := RelationsBetween(96, 95); mrels[0].Type != TypeRival {
		t.Errorf("mirror: %s", mrels[0].Type)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/relationships/...`
Expected: FAIL.

- [ ] **Step 3: Implement**

```go
// Add appends an edge with auto-mirror. No-op if identical edge
// already exists. In-memory only — no persistence v1.
func Add(a, b int, t Type, subtype string) {
	graphMu.Lock()
	defer graphMu.Unlock()
	if hasEdge(a, b, t) {
		return
	}
	graph[a] = append(graph[a], Relation{Other: b, Type: t, Subtype: subtype})
	mirrorType := InverseType(t)
	if !hasEdge(b, a, mirrorType) {
		graph[b] = append(graph[b], Relation{Other: a, Type: mirrorType})
	}
}

// Remove drops an edge and its mirror.
func Remove(a, b int, t Type) {
	graphMu.Lock()
	defer graphMu.Unlock()
	graph[a] = removeEdge(graph[a], b, t)
	graph[b] = removeEdge(graph[b], a, InverseType(t))
}

// removeEdge filters out matching (Other, Type) entries. Caller
// must hold graphMu (write lock).
func removeEdge(edges []Relation, other int, t Type) []Relation {
	out := edges[:0]
	for _, r := range edges {
		if r.Other == other && r.Type == t {
			continue
		}
		out = append(out, r)
	}
	return out
}

// ChangeType atomically removes (a→b oldType) and adds (a→b newType
// + subtype). Mirror also flips.
func ChangeType(a, b int, oldType, newType Type, newSubtype string) {
	graphMu.Lock()
	defer graphMu.Unlock()
	graph[a] = removeEdge(graph[a], b, oldType)
	graph[b] = removeEdge(graph[b], a, InverseType(oldType))
	graph[a] = append(graph[a], Relation{Other: b, Type: newType, Subtype: newSubtype})
	mirrorType := InverseType(newType)
	if !hasEdge(b, a, mirrorType) {
		graph[b] = append(graph[b], Relation{Other: a, Type: mirrorType})
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/relationships/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/relationships/
git commit -m "feat(relationships): mutation API (Add/Remove/ChangeType) (T11)"
```

---

## Task 12: Mob YAML field + LoadDataFiles hook

**Files:**
- Modify: `internal/mobs/mobs.go`

- [ ] **Step 1: Add the YAML entry type + struct field**

Open `internal/mobs/mobs.go`. Locate the `type Mob struct {` block at line ~67. Add a new field. Also add a typed entry struct earlier in the file (or wherever similar small types live).

```go
// RelationshipYAMLEntry is the per-edge authoring shape on a mob
// template's `relationships:` field. Consumed by the relationships
// package at startup.
type RelationshipYAMLEntry struct {
	To      int    `yaml:"to"`
	Type    string `yaml:"type"`
	Subtype string `yaml:"subtype,omitempty"`
}
```

In the `Mob` struct, add:

```go
Relationships []RelationshipYAMLEntry `yaml:"relationships,omitempty"`
```

(Place near related fields — `Groups`, etc.)

- [ ] **Step 2: Wire LoadFromMobs into LoadDataFiles**

Locate the end of `LoadDataFiles` at ~line 1124 (after `mudlog.Info("mobs.LoadDataFiles()", ...)`). Add a call that flattens the loaded mobs into `[]relationships.MobEdges` and invokes `relationships.LoadFromMobs`:

```go
import (
	"github.com/GoMudEngine/GoMud/internal/relationships"
)

// At the end of LoadDataFiles, after mobs are committed to the cache:
mobsMu.RLock()
edges := make([]relationships.MobEdges, 0, len(mobs))
for id, spec := range mobs {
	if len(spec.Relationships) == 0 {
		continue
	}
	conv := make([]relationships.EdgeInput, 0, len(spec.Relationships))
	for _, e := range spec.Relationships {
		conv = append(conv, relationships.EdgeInput{
			To:      e.To,
			Type:    relationships.Type(e.Type),
			Subtype: e.Subtype,
		})
	}
	edges = append(edges, relationships.MobEdges{
		MobId: int(id),
		Edges: conv,
	})
}
mobsMu.RUnlock()

relationships.LoadFromMobs(edges, func(mobId int) bool {
	mobsMu.RLock()
	defer mobsMu.RUnlock()
	_, ok := mobs[MobId(mobId)]
	return ok
})
```

(Adjust the `mobs[id]` lookup to use the actual map key type — `MobId` likely. Read the surrounding code to mirror.)

- [ ] **Step 3: Verify build + existing mob tests**

Run: `go build ./... && go test ./internal/mobs/... ./internal/relationships/...`
Expected: clean / PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/mobs/mobs.go
git commit -m "feat(relationships): mob YAML field + LoadDataFiles hook (T12)"
```

---

## Task 13: Admin command + helpfile

**Files:**
- Create: `internal/usercommands/admin.relationship.go`
- Modify: `internal/usercommands/usercommands.go`
- Create: `_datafiles/world/dogmud/templates/admincommands/help/command.relationship.template`

- [ ] **Step 1: Implement the admin command**

```go
// internal/usercommands/admin.relationship.go
package usercommands

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/relationships"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
 * Role Permissions:
 * relationship  (Admin)
 */

func Relationship(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	args := strings.Fields(rest)
	if len(args) == 0 {
		relationshipUsage(user)
		return true, nil
	}
	switch strings.ToLower(args[0]) {
	case "show":
		return relationshipShow(args[1:], user)
	case "between":
		return relationshipBetween(args[1:], user)
	case "add":
		return relationshipAdd(args[1:], user)
	case "remove":
		return relationshipRemove(args[1:], user)
	case "list":
		return relationshipList(user)
	default:
		relationshipUsage(user)
	}
	return true, nil
}

func relationshipUsage(user *users.UserRecord) {
	if out, err := templates.Process("admincommands/help/command.relationship", nil, user.UserId); err == nil && strings.TrimSpace(out) != "" {
		user.SendText(out)
		return
	}
	user.SendText(
		"Usage:\r\n" +
			"  relationship show <mobId>\r\n" +
			"  relationship between <mobIdA> <mobIdB>\r\n" +
			"  relationship add <mobIdA> <mobIdB> <type> [subtype]\r\n" +
			"  relationship remove <mobIdA> <mobIdB> <type>\r\n" +
			"  relationship list\r\n" +
			"\r\n" +
			"<type> ∈ family | friend | rival | lover | employer | employee\r\n",
	)
}

func relationshipShow(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 1 {
		relationshipUsage(user)
		return true, nil
	}
	mobId, err := strconv.Atoi(args[0])
	if err != nil {
		user.SendText(fmt.Sprintf("Bad mob id %q\r\n", args[0]))
		return true, nil
	}
	rels := relationships.RelationsOf(mobId)
	if len(rels) == 0 {
		user.SendText(fmt.Sprintf("No relationships for mob %d.\r\n", mobId))
		return true, nil
	}
	var b strings.Builder
	mobName := mobNameFor(mobId)
	fmt.Fprintf(&b, "Relationships for mob %d (%s):\r\n", mobId, mobName)
	for _, r := range rels {
		typeLabel := string(r.Type)
		if r.Type == relationships.TypeEmployer {
			typeLabel = "employer-of"
		} else if r.Type == relationships.TypeEmployee {
			typeLabel = "employed-by"
		}
		other := mobNameFor(r.Other)
		subLabel := ""
		if r.Subtype != "" {
			subLabel = fmt.Sprintf(" [%s]", r.Subtype)
		}
		fmt.Fprintf(&b, "  %-12s → mob %d (%s)%s\r\n", typeLabel, r.Other, other, subLabel)
	}
	user.SendText(b.String())
	return true, nil
}

func relationshipBetween(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 2 {
		relationshipUsage(user)
		return true, nil
	}
	a, err := strconv.Atoi(args[0])
	if err != nil {
		user.SendText(fmt.Sprintf("Bad mob id %q\r\n", args[0]))
		return true, nil
	}
	b, err := strconv.Atoi(args[1])
	if err != nil {
		user.SendText(fmt.Sprintf("Bad mob id %q\r\n", args[1]))
		return true, nil
	}
	aSide := relationships.RelationsBetween(a, b)
	bSide := relationships.RelationsBetween(b, a)
	if len(aSide) == 0 && len(bSide) == 0 {
		user.SendText(fmt.Sprintf("No relationship between mob %d and mob %d.\r\n", a, b))
		return true, nil
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "%d (%s) ↔ %d (%s):\r\n", a, mobNameFor(a), b, mobNameFor(b))
	for _, r := range aSide {
		sub := ""
		if r.Subtype != "" {
			sub = fmt.Sprintf(" subtype: %s", r.Subtype)
		}
		fmt.Fprintf(&sb, "  %-12s (%d→%d%s)\r\n", r.Type, a, b, sub)
	}
	for _, r := range bSide {
		sub := ""
		if r.Subtype != "" {
			sub = fmt.Sprintf(" subtype: %s", r.Subtype)
		}
		fmt.Fprintf(&sb, "  %-12s (%d→%d%s)\r\n", r.Type, b, a, sub)
	}
	user.SendText(sb.String())
	return true, nil
}

func relationshipAdd(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 3 {
		relationshipUsage(user)
		return true, nil
	}
	a, err := strconv.Atoi(args[0])
	if err != nil {
		user.SendText(fmt.Sprintf("Bad mob id %q\r\n", args[0]))
		return true, nil
	}
	b, err := strconv.Atoi(args[1])
	if err != nil {
		user.SendText(fmt.Sprintf("Bad mob id %q\r\n", args[1]))
		return true, nil
	}
	t := relationships.Type(strings.ToLower(args[2]))
	subtype := ""
	if len(args) >= 4 {
		subtype = strings.Join(args[3:], " ")
	}
	relationships.Add(a, b, t, subtype)
	user.SendText(fmt.Sprintf("Added: %d → %d (%s).\r\n", a, b, t))
	return true, nil
}

func relationshipRemove(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 3 {
		relationshipUsage(user)
		return true, nil
	}
	a, err := strconv.Atoi(args[0])
	if err != nil {
		user.SendText(fmt.Sprintf("Bad mob id %q\r\n", args[0]))
		return true, nil
	}
	b, err := strconv.Atoi(args[1])
	if err != nil {
		user.SendText(fmt.Sprintf("Bad mob id %q\r\n", args[1]))
		return true, nil
	}
	t := relationships.Type(strings.ToLower(args[2]))
	relationships.Remove(a, b, t)
	user.SendText(fmt.Sprintf("Removed: %d → %d (%s).\r\n", a, b, t))
	return true, nil
}

func relationshipList(user *users.UserRecord) (bool, error) {
	all := relationships.AllRelations()
	if len(all) == 0 {
		user.SendText("Graph is empty.\r\n")
		return true, nil
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Owner != all[j].Owner {
			return all[i].Owner < all[j].Owner
		}
		return all[i].Other < all[j].Other
	})
	var b strings.Builder
	fmt.Fprintf(&b, "Relationship graph (%d edges):\r\n", len(all))
	for _, e := range all {
		sub := ""
		if e.Subtype != "" {
			sub = fmt.Sprintf(" [%s]", e.Subtype)
		}
		fmt.Fprintf(&b, "  %4d → %4d  %-10s%s\r\n", e.Owner, e.Other, e.Type, sub)
	}
	user.SendText(b.String())
	return true, nil
}

// mobNameFor returns the mob template's name for display, or "?" if
// not loaded.
func mobNameFor(mobId int) string {
	spec := mobs.GetMobSpec(mobs.MobId(mobId))
	if spec == nil {
		return "?"
	}
	return spec.Character.Name
}
```

If `mobNameFor` collides with another helper in the package, rename to `relationshipMobNameFor`. The chunk-1.4 admin.knowledge.go also has a mobNameFor candidate; check whether to share or rename.

- [ ] **Step 2: Register the command**

In `internal/usercommands/usercommands.go`, find where `Bounty` was registered (chunk 1.5) and add an entry for `relationship`:

```go
"relationship": {Relationship, true, true, true},  // mirror the admin pattern (admin-only)
```

Read the surrounding entries to verify the right shape. If the command needs to be admin-only, ensure the "admin" flag is set; the command has no player surface v1.

- [ ] **Step 3: Write the help template**

Create `_datafiles/world/dogmud/templates/admincommands/help/command.relationship.template`:

```
<ansi fg="white-bold">relationship</ansi> — inspect and manipulate the mob-to-mob graph

  <ansi fg="cyan">relationship show</ansi> <ansi fg="yellow"><mobId></ansi>
    List all edges for an NPC (numeric mob template id).

  <ansi fg="cyan">relationship between</ansi> <ansi fg="yellow"><mobIdA> <mobIdB></ansi>
    Show edges connecting the two mobs (both directions).

  <ansi fg="cyan">relationship add</ansi> <ansi fg="yellow"><mobIdA> <mobIdB> <type> [subtype]</ansi>
    Runtime add (in-memory only — lost on server restart).
    <type> ∈ family | friend | rival | lover | employer | employee
    Symmetric types auto-mirror; employer auto-creates employee
    on the other side.

  <ansi fg="cyan">relationship remove</ansi> <ansi fg="yellow"><mobIdA> <mobIdB> <type></ansi>
    Drop the edge and its mirror.

  <ansi fg="cyan">relationship list</ansi>
    Dump every edge in the graph (debug).
```

- [ ] **Step 4: Verify build + tests**

Run: `go build ./... && go test ./internal/usercommands/... ./internal/relationships/...`
Expected: clean / PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/usercommands/admin.relationship.go internal/usercommands/usercommands.go _datafiles/world/dogmud/templates/admincommands/help/command.relationship.template
git commit -m "feat(relationships): admin relationship show/between/add/remove/list (T13)"
```

---

## Task 14: `context.md` for `internal/relationships/`

**Files:**
- Create: `internal/relationships/context.md`

Author the chunk-1.6 context.md following the same pattern as the backfilled ones in T1–T5.

- [ ] **Step 1: Write the file**

Sections (per the roadmap rule):
- **Overview**: source-of-truth on mob YAMLs; in-memory graph; auto-mirror; six relationship types.
- **Key Components**: file map (`types.go`, `graph.go`, `relationships.go`).
- **Key Functions**: signatures + behavior for `LoadFromMobs`, `Add`, `Remove`, `ChangeType`, `RelationsOf`, `RelationsOfType`, `KinOf`, `AlliesOf`, `RivalsOf`, `RelationsBetween`, `AreRelated`, `EmployerOf`, `EmployedBy`, `AllRelations`, `IsSymmetric`, `InverseType`.
- **Global State**: `graph map[int][]Relation` + `graphMu sync.RWMutex`.
- **Data Structure Design**: `relationships:` field on mob YAML, sample. Auto-mirror semantics. Per-side subtype.
- **Integration Notes**: consumed by 4.5 reactive goals + 3.6 idle conversation (future). Loaded by `mobs.LoadDataFiles` post-mob-load. No persistence v1.
- **Testing Notes**: pointer to `*_test.go` files; in-memory test harness; alwaysValid helper.

~200–300 lines.

- [ ] **Step 2: Commit**

```bash
git add internal/relationships/context.md
git commit -m "docs(relationships): context.md for chunk 1.6"
```

---

## Task 15: Roadmap update

**Files:**
- Modify: `MOB_ALIVENESS_ROADMAP.md`

- [ ] **Step 1: Mark chunk 1.6 as Done in the progress tracker**

Locate the `| 1.6 | Substrate | NPC-to-NPC relationships | M | — | Not started |` row and change `Not started` to `Done`. Update the roll-up: `5 / 40 done` → `6 / 40 done`.

- [ ] **Step 2: Update the chunk's mini-brief**

Locate the `### 1.6 NPC-to-NPC relationships` section. Change `**Status:** Not started` to `**Status:** Done (2026-05-09)`. Append a `Shipped:` paragraph mirroring the format of 1.1–1.5.

Suggested template:

```markdown
- **Shipped:** `internal/relationships/` package storing the in-memory mob-to-mob relationship graph. Source of truth: each mob template's YAML gains an optional `relationships:` field with `to`, `type`, `subtype`. Six types (family, friend, rival, lover, employer, employee); engine auto-mirrors symmetric (same-type reverse) and asymmetric (employer ↔ employee) at load time. Subtype is per-side flavor. Permissive validation — unknown ids, self-edges, unknown types, conflicts all warn-not-panic. Public API: RelationsOf, RelationsOfType, KinOf, AlliesOf, RivalsOf, RelationsBetween, AreRelated, EmployerOf, EmployedBy, AllRelations, plus mutation Add/Remove/ChangeType (in-memory only v1; persistence overlay deferred). Loader hook in `mobs.LoadDataFiles` flattens mob templates into `LoadFromMobs(edges, validateMobId)` post-load. Admin command `relationship show/between/add/remove/list` + helpfile. **Backfilled `context.md` for chunks 1.1–1.5** plus authored fresh one for 1.6, per the new aliveness roadmap maintenance rule. Spec at `docs/superpowers/specs/2026-05-09-mob-aliveness-1.6-npc-relationships-design.md`, plan at `docs/superpowers/plans/2026-05-09-mob-aliveness-1.6-npc-relationships.md`.
```

- [ ] **Step 3: Commit**

```bash
git add MOB_ALIVENESS_ROADMAP.md
git commit -m "docs(roadmap): mark chunk 1.6 (npc relationships) as Done"
```

---

## Final review

After all tasks complete, dispatch the `superpowers:code-reviewer` agent for a holistic pass before smoke testing.

Suggested smoke goal file: `tools/testing/goals/relationships-thornwall-smoke.yaml` covering: admin shows authored relationships (note: pre-test setup needs the controller to seed at least one NPC's relationships in YAML before the smoke run; or the smoke focuses on runtime mutation), runtime add via admin command, between query, removal.

Branch state after completion: `feature/mob-aliveness-1.3-crimes` carries chunks 1.1, 1.2, 1.3, 1.4, 1.5, AND 1.6. When confidence is high, merge to development with `--no-ff`.
