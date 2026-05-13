# Relationships System Context

## Overview

The `internal/relationships` package maintains a source-of-truth graph of structural relationships between NPC templates: family, friend, rival, lover, and asymmetric employer/employee bonds. Relationships are authored on each mob template's YAML file and read at startup into an in-memory graph with auto-mirror semantics — authors declare edges on one side, the engine creates reverse edges automatically (same type for symmetric relationships, inverse type for employer/employee). The package serves read queries (`RelationsOf`, `KinOf`, `AlliesOf`, `EmployerOf`) to downstream consumers (reactive goal seeding, NPC idle conversation) and mutation API (`Add`, `Remove`, `ChangeType`) for in-memory runtime changes. No persistence v1 — mutations are lost on server restart and will be layered via overlay file in future chunks.

## Key Components

### Core Files
- **types.go**: Type enum (`Type` constants: TypeFamily, TypeFriend, TypeRival, TypeLover, TypeEmployer, TypeEmployee); symmetry and inverse helpers (`IsSymmetric`, `InverseType`); data structures (`Relation`, `OwnedRelation`)
- **graph.go**: In-memory graph state (`graph map[int][]Relation`, `graphMu sync.RWMutex`); load-time graph construction (`LoadFromMobs`); helper functions (`hasEdge`)
- **relationships.go**: Public query API (`RelationsOf`, `RelationsOfType`, `KinOf`, `AlliesOf`, `RivalsOf`, `RelationsBetween`, `AreRelated`, `EmployerOf`, `EmployedBy`, `AllRelations`); mutation API (`Add`, `Remove`, `ChangeType`)
- **relationships_test.go**: Unit tests for query functions, mutation API, asymmetric pairs, and graph state verification
- **graph_test.go**: Unit tests for load-time auto-mirror (symmetric and asymmetric), validation (self-edge, unknown target, unknown type, deduplication)
- **test_main_test.go**: Test setup with mudlog initialization and graph reset helper

## Key Functions

### Type Helpers
- **IsSymmetric(t Type) bool**: Returns true for symmetric types (family, friend, rival, lover) and false for asymmetric pairs (employer/employee). Used to determine auto-mirror behavior at load and during mutations.
- **InverseType(t Type) Type**: Returns the auto-mirror reverse type. Symmetric types return themselves; employer returns employee and vice versa. Applied uniformly at load-time and by mutation API.

### Load-Time API
- **LoadFromMobs(input []MobEdges, validateMobId func(mobId int) bool)**: Populates the in-memory graph from authored mob edges. Walks the `input` slice (one entry per mob template with relationships); for each `EdgeInput`, declares the forward edge and auto-mirrors it with `InverseType` semantics. Validation: self-edges, unknown targets, unknown types, and duplicate edges are logged as warnings and skipped; first-wins wins for dedup, and mirrors are only added if not already present. Callback `validateMobId` allows the caller (mobs package) to confirm target mob ids without coupling. Called once at startup by `internal/mobs.LoadDataFiles`.

### Read API (Query)
- **RelationsOf(mobId int) []Relation**: Returns all outgoing edges for the given mob template, regardless of type. Snapshot copy, safe from concurrent mutations.
- **RelationsOfType(mobId int, types ...Type) []Relation**: Filters `RelationsOf` by type(s). Pass zero types to get all edges. Used to implement the convenience helpers.
- **KinOf(mobId int) []Relation**: Convenience for family + lover edges (immediate kin).
- **AlliesOf(mobId int) []Relation**: Convenience for family + friend + lover edges (allies).
- **RivalsOf(mobId int) []Relation**: Convenience for rival edges.
- **RelationsBetween(a, b int) []Relation**: Returns edges from `a` to `b` (a's outgoing view of the pair). Empty if not connected or if only `b → a` exists.
- **AreRelated(a, b int) bool**: Returns true if any edge from `a` to `b` exists (fast mutual-reachability check).
- **EmployerOf(mobId int) []int**: Returns mob template IDs this NPC employs (filter edges on TypeEmployer, extract Other field).
- **EmployedBy(mobId int) []int**: Returns mob template IDs that employ this NPC (filter edges on TypeEmployee, extract Other field).
- **AllRelations() []OwnedRelation**: Flattens every edge in the graph into a sorted list of (Owner, Relation) pairs for debug/admin display. Each forward and mirror edge appears separately. Sorted by owner mobId, then by Other mobId.

### Mutation API (Write; in-memory v1)
- **Add(a, b int, t Type, subtype string)**: Appends an edge from `a` to `b` with the given type and optional subtype. Auto-mirrors: calculates `InverseType(t)` and adds the reverse edge on side `b` (no subtype on mirror). No-op if identical edge already exists. Acquires write lock.
- **Remove(a, b int, t Type)**: Drops both the forward edge (`a → b`) and its mirror (`b → a` with `InverseType(t)`) atomically. No-op if edge doesn't exist.
- **ChangeType(a, b int, oldType, newType Type, newSubtype string)**: Atomically removes the old edge and adds the new one, with auto-mirror semantics. Both sides update in one operation. Equivalent to `Remove` followed by `Add` but serialized under one write lock.

## Global State

### Graph Cache and Synchronization
```go
var (
    graph   = make(map[int][]Relation)  // mob template id → outgoing edges
    graphMu sync.RWMutex                 // protects graph
)
```

The `graph` is keyed by mob template id and holds the adjacency list — each mob's outgoing edges. Auto-mirror means every edge is stored twice (once per side) so `RelationsOf(a)` always includes all edges, regardless of authoring direction. The `graphMu` is held:
- Read lock for query operations (RelationsOf, all Read API functions).
- Write lock for load-time population (LoadFromMobs) and mutations (Add, Remove, ChangeType).

### No Persistence v1
The in-memory graph is built once at startup from mob YAML and never written to disk. Runtime mutations (Add, Remove, ChangeType) affect only `graph` — they do not persist and are lost on server restart. When a real consumer lands (marriage system, hiring quest), an overlay file `_datafiles/world/dogmud/relationships.overlay.yaml` (gitignored) will layer runtime changes over the YAML-derived base.

## Data Structure Design

### Type Enum
```go
type Type string

const (
    TypeFamily   Type = "family"
    TypeFriend   Type = "friend"
    TypeRival    Type = "rival"
    TypeLover    Type = "lover"
    TypeEmployer Type = "employer"  // I am their employer
    TypeEmployee Type = "employee"  // I am their employee (auto-mirror only)
)
```

The six types split into two categories: symmetric (family, friend, rival, lover) where both sides see the same type, and asymmetric (employer ↔ employee) where sides see opposite types. The enum is closed — unknown types are logged as warnings at load time.

### Edge Representation
```go
type Relation struct {
    Other   int    // mob template id of the other party
    Type    Type
    Subtype string // optional flavor; "" if unset
}
```

A `Relation` is one outgoing edge from a mob to another. The `Other` field identifies the target mob by template id (not instance id). The `Type` identifies the relationship class. The `Subtype` is free-form flavor (e.g., "drinking-companion", "niece", "priest-handyman") and is per-side — both mobs can declare their own subtype for the same edge without duplicating the edge itself.

### OwnedRelation (for bulk queries)
```go
type OwnedRelation struct {
    Owner int
    Relation
}
```

Pairs an edge with its owner mob id for `AllRelations()` snapshots. Used by admin commands for debug output.

### Mob YAML Schema
On each mob template (`internal/mobs.Mob`), add an optional `relationships:` field:

```yaml
mobid: 95
name: temple priest Olen
# ... existing fields ...
relationships:
  - to: 96
    type: friend
    subtype: drinking-companion
  - to: 117
    type: family
    subtype: niece
  - to: 113
    type: rival
  - to: 248
    type: employer
    subtype: priest-handyman
```

Each entry in the `relationships:` list declares one outgoing edge. The `to:` field must be a valid mob template id. The `type:` field is one of the six enum values. The `subtype:` field is optional and defaulted to an empty string. The field structure mirrors `EdgeYAML` in the mob loader:

```go
type EdgeYAML struct {
    To      int    `yaml:"to"`
    Type    Type   `yaml:"type"`
    Subtype string `yaml:"subtype,omitempty"`
}
```

### Graph Storage (In-Memory)
```
map[int][]Relation
  95 → [Relation{Other: 96, Type: friend, Subtype: "drinking-companion"},
        Relation{Other: 117, Type: family, Subtype: "niece"},
        Relation{Other: 113, Type: rival, Subtype: ""},
        Relation{Other: 248, Type: employer, Subtype: "priest-handyman"}]
  96 → [Relation{Other: 95, Type: friend, Subtype: ""}]  // auto-mirrored, no subtype
  117 → [Relation{Other: 95, Type: family, Subtype: ""}] // auto-mirrored, no subtype
  ...
```

Each mob template id maps to a slice of outgoing edges. Auto-mirror means symmetric and asymmetric relationships both create reverse edges: mob 96 automatically gains the mirror friend-edge to mob 95 (with no subtype, per the per-side rule), and mob 248 automatically gains the mirror employee-edge to mob 95 (the inverse of employer).

## Validation Policy (Callout)

At load time, each authored edge is validated:

- **Self-edge** (`to:` equals the declaring mob's own id): Logged as warning, skipped. Prevents circular dependencies at the source.
- **Unknown target** (`to:` is not a known mob template id): Logged as warning, skipped. Allows authors to fix typos without crashing the server.
- **Unknown type** (`type:` is not one of the six enum values): Logged as warning, skipped. Prevents misspellings like `"fren"` instead of `"friend"`.
- **Duplicate edge** (mob A declares two edges to mob B with the same type): Logged as warning, first wins, later skipped. Subtype from later declaration wins if the first had no subtype (rare edge case for missing flavor).
- **Pair conflicts** (mob A says "friend with B," mob B says "rival with A"): Not an error. Both edges are stored as-declared; the graph reflects authored opinions, not reconciled truth. Authors fix in YAML if they want consensus.

All violations are logged via `mudlog.Warn` with enough context (from mobId, to mobId, type) for an admin to locate the YAML file and fix. Warnings do not halt startup.

## Integration Notes

### Load-Time Hook
The relationships package is called by `internal/mobs.LoadDataFiles()` after all mob templates are loaded into memory. The call:

```go
relationshipInputs := ... // slice of MobEdges built from each mob's YAML
relationships.LoadFromMobs(relationshipInputs, mobs.IsMobTemplate)
```

The callback `mobs.IsMobTemplate` allows relationships to validate target ids against the mob registry without circular imports.

### Future Consumers (Not Wired v1)
- **4.5 Reactive Goal Seeding**: Will read `AlliesOf(victimId)` to seed revenge goals when an NPC dies.
- **3.6 NPC Idle Conversation**: Will read relationship pairs to find natural conversation candidates (friends, family, rivals) during idle turns.

Both consumers work against the populated `graph` and require no changes to the relationships package itself.

### Independent of Knowledge (1.4)
Structural fact ("Voss is Lars's brother") lives here in relationships. Awareness ("Lars knows Voss is his brother") lives in `internal/knowledge`. v1 assumes kin always know their kin, but the two substrates are independent — changes to one do not retroactively seed or update the other.

### No Persistence v1
Runtime mutations are in-memory only. On server restart, `graph` is repopulated from mob YAML; runtime changes are lost. When a first real consumer needs persistence (e.g., marriage quest grants a lover edge), an overlay file will be added:

```yaml
# _datafiles/world/dogmud/relationships.overlay.yaml (future, gitignored)
# Layered over YAML at startup; persisted after each mutation
relationships:
  - from: 95
    to: 200
    type: lover
    subtype: spouse
```

### Admin Command
`internal/usercommands/admin.relationship.go` registers subcommands:

```
relationship show <mobId>                           — all edges for an NPC
relationship between <mobIdA> <mobIdB>              — pair introspection
relationship add <mobIdA> <mobIdB> <type> [subtype] — runtime add (in-memory v1)
relationship remove <mobIdA> <mobIdB> <type>        — runtime remove
relationship list                                   — full graph (debug)
```

Output uses display names (mob name lookup) and formats edges with types and subtypes for readability.

### Dependencies
- **mobs**: Imports `mobs.IsMobTemplate` callback for validation; the relationships package does not directly import mobs to avoid coupling.
- **mudlog**: Logs warnings on validation failures.
- No other package dependencies. The relationships package is intentionally lightweight and independent.

### Imported By
- **internal/mobs**: Calls `LoadFromMobs` during data-file startup.
- **internal/usercommands/admin.relationship**: Admin command implementation.
- **Future: internal/hooks (4.5 reactive goals, 3.6 idle conversation)**: Will query via `AlliesOf`, `RivalsOf`, `RelationsBetween`.

## Testing Notes

### Test Files
- **relationships_test.go**: ~100 lines covering query functions (RelationsOf, RelationsOfType, KinOf, AlliesOf, RivalsOf, RelationsBetween, AreRelated, EmployerOf, EmployedBy, AllRelations), mutation API (Add with auto-mirror, Add idempotence, Remove with mirror cleanup, ChangeType atomic), and asymmetric-pair mirroring.
- **graph_test.go**: ~170 lines covering load-time behavior (auto-mirror for symmetric, auto-mirror for asymmetric with inverse type, per-side subtypes, first-wins dedup) and validation cases (self-edge, unknown target, unknown type, duplicate edges all logged and handled correctly).
- **test_main_test.go**: TestMain setup; initializes mudlog to suppress nil-pointer errors during tests. Provides `resetGraph()` helper to wipe the in-memory graph between tests.

### Test Helpers
- **alwaysValid(int) bool**: Approves every mob id; used by load tests to focus on graph logic rather than validation.
- **buildFromSpecs([]edgeSpec)**: Resets graph and manually populates edges in test format (`From`, `To`, `Type`, `Subtype`). Applies auto-mirror during construction to simulate load-time behavior.
- **edgesOf(mobId int) []Relation**: Acquires read lock and returns a snapshot of edges for the given mob. Used to inspect graph state in assertions.

### Key Test Scenarios
- **Symmetric auto-mirror**: Declare `type: friend, to: 96` on mob 95; assert mob 96 has `type: friend, to: 95`.
- **Asymmetric auto-mirror**: Declare `type: employer, to: 248` on mob 95; assert mob 248 has `type: employee, to: 95`.
- **Per-side subtypes**: Both mobs declare different subtypes; both sides keep their own (no subtype on mirror, per the design rule).
- **Validation**: Unknown `to:`, self-edge, unknown type, duplicate edge all produce warnings and are handled per the validation policy.
- **Query coverage**: Each read-API function tested against a seeded graph with multiple relationship types.
- **Mutation coverage**: Add (no-op on duplicate), Remove (mirror also removed), ChangeType (both sides update atomically).
- **AllRelations snapshot**: Returns all forward + mirror edges, sorted by owner and Other.

### In-Memory Test Pattern
Tests do not rely on disk I/O or external fixtures. The `resetGraph()` helper clears the in-memory map between tests, and `buildFromSpecs()` populates it programmatically. This follows the pattern used by `internal/crimes` and `internal/opinions` test suites.
