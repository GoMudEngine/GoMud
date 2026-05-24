# Mob Aliveness 1.6 — NPC-to-NPC Relationships

> **Phase 1 substrate.** Kinship and friendship graph between mob
> templates: family, friend, rival, lover, employer/employee.
> Authored on each mob's YAML; engine builds an in-memory graph
> at startup with auto-mirror for symmetric and asymmetric edges.
> No player-target relationships v1 (romance is out of scope).

## Goal

Per-NPC structural relationship graph: "Voss is Lars's brother,"
"Marta is the smith's wife," "the priest employs the cook." Read
queries let consumers ask "who would care if X dies?" and "who
would chat with X?" — the two known v1 consumers (4.5 reactive
goal seeding, 3.6 NPC↔NPC idle conversation) live in later phases
but their substrate needs the graph.

The chunk's job is **substrate only** — the YAML schema, load-time
graph build with auto-mirror, read API, mutation API (in-memory
only v1), admin command, and a `context.md` to lock in package
conventions.

## Architectural musts

The brief lists "family, friend, rival, lover, employer/employee"
as types. v1 commits to:

1. **Source of truth: mob YAML files.** Each mob template gets an
   optional `relationships:` field. Authors declare relationships
   inline with the mob, where they're most discoverable. The
   relationships package reads all mob templates at startup and
   builds an in-memory graph.

2. **Templates, not instances.** Relationships key on mob TEMPLATE
   ids. "Voss is Lars's brother" is a permanent fact about the
   templates; instances inherit. Relationships survive deaths and
   respawns.

3. **Auto-mirror at load.** Authors declare an edge on one side;
   the engine creates the reverse:
   - Symmetric types (family, friend, rival, lover) → same-type
     reverse edge on the other mob.
   - Asymmetric pair (employer ↔ employee) → engine writes the
     inverse type. Author declares `type: employer, to: 99`; mob
     99's edges receive `type: employee, to: <author-id>`.
   - Subtype is per-side flavor and does NOT auto-mirror. Both
     sides can declare their own subtype if they want to (engine
     detects existing edge and merges rather than duplicating).

4. **Permissive validation.** Unknown `to:` ids, self-edges,
   unknown types, and pair-conflicts are logged as warnings, not
   panics. Bad authoring shouldn't crash the server; admins fix
   in-flight.

5. **No player-target relationships v1.** `relationships:` only
   references other mob templates. Romance and player-NPC bonds
   are explicitly out of scope per the brief.

6. **Mutation API exists, persistence deferred.** The brief lists
   "mutation API" in scope; v1 ships Add / Remove / ChangeType
   that mutate the in-memory graph. There are no v1 writers, so
   no persistence layer is shipped — runtime mutations are lost
   on server restart. When a real consumer lands (marriage
   system, hiring quest, etc.) an overlay file at
   `_datafiles/world/dogmud/relationships.overlay.yaml`
   (gitignored) layers runtime changes over the YAML-derived base
   at startup.

## Architecture & storage

`internal/relationships/` package, parallel to chunks 1.1–1.5.
Same patterns where applicable: lazy-load + cache + mutex. Unlike
chunks 1.4/1.5, there's no per-package YAML file v1 — the source
of truth is the existing mob YAML files. The package parses the
new `relationships:` field on each mob template at startup, builds
the in-memory graph, and serves queries from there.

## Schema

**Mob YAML addition** — new optional `relationships:` field at
the top level of any mob template:

```yaml
mobid: 95
name: temple priest Olen
# ... existing fields ...
relationships:
  - to: 96            # Tavern Keeper Marek
    type: friend
    subtype: drinking-companion
  - to: 117           # Barmaid Dal
    type: family
    subtype: niece
  - to: 113           # Weaver Maren
    type: rival
  - to: 248           # Tavern Cook Brynn
    type: employer    # Olen employs Brynn (asymmetric)
    subtype: priest-handyman
```

Existing mobs without a `relationships:` field are unchanged — the
field is optional and defaults to no edges.

**Go types (sketch — implementer may adjust naming):**

```go
type Type string

const (
    TypeFamily   Type = "family"
    TypeFriend   Type = "friend"
    TypeRival    Type = "rival"
    TypeLover    Type = "lover"
    TypeEmployer Type = "employer" // I am their employer
    TypeEmployee Type = "employee" // I am their employee (auto-mirror only)
)

type Relation struct {
    Other   int    // mob template id of the other party
    Type    Type
    Subtype string // optional flavor; "" if unset
}

// On mobs.Mob (the template struct), add:
//   Relationships []EdgeYAML `yaml:"relationships,omitempty"`
type EdgeYAML struct {
    To      int    `yaml:"to"`
    Type    Type   `yaml:"type"`
    Subtype string `yaml:"subtype,omitempty"`
}
```

The in-memory graph is keyed `map[int][]Relation` — for each mob
template id, a list of outgoing edges. Auto-mirror means every
edge is stored twice (once per side); the package owns this
duplication so callers always see a complete picture from
whichever side they query.

## Validation policy

At load time, the package walks every mob template. For each
`relationships:` entry:

- **`to:` is an unknown mob template id** → log warning, skip.
- **`to:` equals the declaring mob's own id** (self-edge) → log
  warning, skip.
- **`type:` is not one of the known enum values** → log warning,
  skip.
- **Pair conflicts** (mob A says "friend with B," mob B says
  "rival with A") → log warning. Resolution rule: keep both edges
  as-declared (asymmetric in opinion). The graph reflects what
  authors wrote; it doesn't try to reconcile contradictions.
  Authors fix in YAML.
- **Duplicate edges** (mob A declares two edges to mob B with the
  same type) → log warning, keep first, drop later. Subtype from
  later wins if first had no subtype.

Logging uses `mudlog.Warn` with enough context (mob id, target id,
type) for an admin to find the YAML and fix.

## Public API

**Reads** (the consumer-facing surface):

```go
// All edges this NPC has, regardless of type.
func RelationsOf(mobId int) []Relation

// Filtered by type.
func RelationsOfType(mobId int, types ...Type) []Relation

// Convenience for the most-asked queries.
func KinOf(mobId int) []Relation       // family + lover
func AlliesOf(mobId int) []Relation    // family + friend + lover
func RivalsOf(mobId int) []Relation    // rival

// Pair introspection.
func RelationsBetween(a, b int) []Relation  // edges with Other==b on a's side
func AreRelated(a, b int) bool              // any edge between the pair?

// Asymmetric specifics.
func EmployerOf(mobId int) []int  // mob ids this NPC employs
func EmployedBy(mobId int) []int  // mob ids this NPC is employed by
```

**Writes** (mutation API; in-memory only v1):

```go
// Add an edge with auto-mirror semantics. No-op if an identical
// edge already exists.
func Add(a, b int, t Type, subtype string)

// Remove an edge (and its mirror).
func Remove(a, b int, t Type)

// Atomic "remove old, add new." Same auto-mirror semantics.
func ChangeType(a, b int, oldType, newType Type, newSubtype string)
```

**Bulk listing** (admin/debug):

```go
func AllRelations() []Relation  // every edge in the graph (snapshot copy)
```

## Decay

None. Relationships are authored facts; they don't decay. Runtime
mutations are explicit (Add / Remove / ChangeType). Death of an
NPC instance does NOT remove relationships from the template —
the template's edges stay, because the slain instance was just one
manifestation of that template.

If a future consumer wants "this template's relationships are now
historical because the named character is canonically dead," that
ships in a later chunk via a `Status` field on the edge or a
canonical-death marker. v1 doesn't model that.

## Substrate intersections

| Intersection                             | v1 policy                                                |
|------------------------------------------|----------------------------------------------------------|
| Relationships ↔ 1.4 knowledge            | Independent. The structural fact "Voss is Lars's brother" is in 1.6; the awareness "Lars knows Voss is his brother" is 1.4. v1 assumes kin always know their kin (1.4 doesn't auto-seed knowledge from 1.6 yet). |
| Relationships ↔ 4.5 reactive goals       | 4.5 will read 1.6 (`AlliesOf(victimId)`) to seed revenge goals. Not wired v1.                          |
| Relationships ↔ 3.6 idle conversation    | 3.6 will read 1.6 to find chat-candidate pairs. Not wired v1.                                        |
| Relationships ↔ death events             | No-op v1. Templates persist regardless of instance death.                                            |

The general rule continues from 1.4 / 1.5: **no retroactive
feedback loops between substrates.**

## Admin command

`internal/usercommands/admin.relationship.go`, registered with
admin role.

```
relationship show <mobId>                            — all edges for an NPC
relationship between <mobIdA> <mobIdB>               — edges connecting the pair
relationship add <mobIdA> <mobIdB> <type> [subtype]  — runtime add (in-memory v1)
relationship remove <mobIdA> <mobIdB> <type>         — runtime remove
relationship list                                    — every edge (debug)
```

`<mobId>` accepts numeric template id only — the unified-parser-
helper memory entry covers the multi-word lookup case for a future
sweep.

`<type>` ∈ `family | friend | rival | lover | employer | employee`.

**Output formatting:**

`relationship show 95`:
```
Relationships for mob 95 (temple priest Olen):
  family       → mob 117 (barmaid Dal)         [niece]
  friend       → mob 96  (tavern keeper Marek) [drinking-companion]
  rival        → mob 113 (weaver Maren)
  employer-of  → mob 248 (tavern cook Brynn)   [priest-handyman]
```

`relationship between 95 96`:
```
95 (Olen) ↔ 96 (Marek):
  friend       (95→96 subtype: drinking-companion)
  friend       (96→95 subtype: poker-night-host)
```

`relationship list` is a paginated dump sorted by mobId, used for
debug.

**Help template:**
`_datafiles/world/dogmud/templates/admincommands/help/command.relationship.template`,
mirrors the format used by command.knowledge / command.bounty.

## Documentation: context.md

Per the new aliveness-roadmap maintenance rule, every chunk that
creates a new package SHIPS a `context.md` in the established
DOGMud style. Chunk 1.6 ships `internal/relationships/context.md`
AND backfills the missing context.md files for the previous five
substrate chunks:

- `internal/opinions/context.md` (chunk 1.1)
- `internal/factions/context.md` (chunk 1.2)
- `internal/crimes/context.md` (chunk 1.3)
- `internal/knowledge/context.md` (chunk 1.4)
- `internal/bounties/context.md` (chunk 1.5)

Style references in the codebase:

- `internal/badinputtracker/context.md` (~170 lines) — small,
  single-responsibility package
- `internal/clans/context.md` (~190 lines) — medium, multi-file
  package
- `internal/buffs/context.md` (~700 lines) — large, deeply-
  integrated package

Required sections per file: Overview, Key Components (file map),
Key Functions (signatures + behavior), Global State (if any), Data
Structure Design (schemas + YAML shapes), Integration Notes (which
packages consume / are consumed by), Testing Notes.

## Testing

**Unit tests** in `internal/relationships/`:

- Auto-mirror at load (symmetric): declare `type: friend, to: 96`
  on mob 95; assert mob 96's edges include
  `type: friend, to: 95`.
- Auto-mirror with asymmetric: declare `type: employer, to: 248`
  on mob 95; assert mob 248's edges include
  `type: employee, to: 95`.
- Subtype is per-side: both sides declare with different
  subtypes; both sides keep their own.
- Validation: unknown `to:` id, self-edge, unknown type, pair
  conflict, duplicate edge — each logged as warning, edge
  handled per the policy table.
- Read API: each query function (RelationsOf, RelationsOfType,
  KinOf, AlliesOf, RivalsOf, RelationsBetween, AreRelated,
  EmployerOf, EmployedBy) tested against a seeded graph.
- Mutation API: Add creates edge + mirror; Add of existing edge
  is no-op; Remove drops edge + mirror; ChangeType atomic.

**Integration test:**

- Inject a synthetic mob with `relationships:` declared into the
  mobs registry via the test seam used by chunk 1.4 hook tests
  (or stand up minimal fixtures). Verify the relationships
  package picks up the edges at startup and serves them.

**Smoke test goal file**
(`tools/testing/goals/relationships-thornwall-smoke.yaml`):

- Admin runs `relationship show <mobId>` for an authored NPC
  with relationships and confirms output shape matches.
- Admin runs `relationship between <mobA> <mobB>` and confirms
  bidirectional rendering.
- Admin tests `relationship add 95 100 friend` and re-runs `show
  95` — confirms in-memory mutation. Note in report: persistence
  is NOT expected v1 (will be lost on server restart).

## Performance

- Read-heavy substrate: the graph is built once at startup and
  serves O(1) lookups by mob id (with O(edges-per-mob) traversal
  for queries).
- Map-keyed by mob id; expected size is mob roster (low hundreds)
  × relationships-per-mob (single digits) — easily fits in memory.
- Mutations are rare and serialized via mutex.
- Auto-mirror at load is one-time cost.

## Out of scope (v1)

| Item                                          | Why deferred                                              |
|-----------------------------------------------|-----------------------------------------------------------|
| Player-target relationships                   | Brief explicitly excludes player-NPC bonds and romance.   |
| Persistence of runtime mutations              | No v1 consumer writes; ships when first writer lands.     |
| Authored-content seeding (Stillwater/Thornwall) | Chunk 1.6 substrate only; content authoring comes with 6.1 town-flavor pass + later content chunks. |
| Subtype-aware queries                         | v1 consumers don't filter on subtype; data on disk for v2.|
| Asymmetric subtypes that auto-flip            | Engine doesn't try to auto-flip "brother" → "brother" (already symmetric) or "mentor" → "student" (would need a closed enum). v1: subtype is per-side flavor. |
| Death-driven graph mutation                   | Templates persist; instances die. v1 doesn't track canonical character death. |
| Group-relationships ("the smiths' guild are rivals") | v1 is mob-to-mob only.                              |
| Relationship strength/intensity (close vs distant) | Type + subtype enough for v1.                         |

## Open questions / deferred decisions

- **Knowledge integration.** v1 doesn't auto-seed knowledge
  records from relationships ("Lars knows Voss is his brother"
  isn't auto-written into Lars's knowledge file). Defer until
  4.5 / 3.6 consumers want it.
- **Subtype inverse map.** If a future content authoring pass
  wants "brother" → "sister" / "father" → "child" auto-flipping,
  add a closed-enum subtype with a known inverse map. v1 free-
  form is good enough.
- **Faction-driven relationships.** Could "all members of faction
  A are rivals of all members of faction B" be expressed
  succinctly? v1 says no — express via per-mob edges. Group
  relationships ship if/when authoring volume justifies them.

## Migration

None at the substrate level. New package, new optional YAML
field on mob templates. Existing mobs without a `relationships:`
field load with no edges, behavior unchanged. The `context.md`
backfills for 1.1–1.5 are pure documentation additions.
