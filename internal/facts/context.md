# Facts System Context

## Overview

The `internal/facts` package maintains a central registry of world facts and per-NPC awareness records. Each fact is a stand-alone assertion about the world (e.g., "The king is dead," "Thornhall was razed") with optional spatial scope (zone, region), significance rating, and scheduled expiry or manual withdrawal. NPCs record awareness of facts through a separate per-observer file-backed store, tracking how each fact was learned (witnessed, told, deduced) and when. The package replaces the in-memory `recentGossipEvents` TempData with persistent awareness, integrates with the `worldevents.Significance` enum for unified event classification, and auto-withdraws facts when their bound mob respawns. It serves gossip lines, decision-making, and future systems (town memory, dialogue branching, tactical knowledge) with a queryable substrate: "does NPC X actually know fact Y?"

## Key Components

### Core Files
- **types.go**: Enums and data structures (`Status` constants: active/withdrawn/expired; `Source` constants: witnessed/told/deduced/unknown; `Fact`, `FactKnowledge`, `Awareness`, `Registry` structs; `MobAwarenessSeed` for load-time seeding; `DeclareOpts` declaration options)
- **facts.go**: Public API for registry writes (`Declare`, `Withdraw`, `Expire`, `WithdrawAllBoundTo`, `PruneExpired`); registry reads (`GetFact`, `AllActiveFacts`, `AllFactsByTag`, `AllRows`); awareness writes (`RecordHeardEvent`, `RecordKnowsFact`, `ForgetFact`, `ForgetAll`); awareness reads (`HeardEvent`, `KnowsFact`, `KnownFactsOf`, `AllForObserver`); load-time hook (`LoadFromMobs`); test seams for round injection and bounded FIFO cap
- **persistence.go**: Disk I/O for registry and per-observer awareness files (tmp-rename atomicity), dual-mutex serialization pattern (`registrySaveMu`, `awarenessSaveMu`), lazy-load double-check-lock pattern for in-memory caches
- **facts_test.go**: Unit tests for registry operations (declare/collision/withdraw/expire/prune/withdraw-on-respawn), registry reads (AllActiveFacts/AllFactsByTag), awareness lifecycle (heard events with bounded FIFO, known facts, idempotence)
- **persistence_test.go**: Tests for disk I/O, cache lifecycle, concurrent access patterns
- **types_test.go**: Type enum and data structure tests
- **test_main_test.go**: TestMain setup with temp-dir isolation, `resetCaches()` helper

## Key Functions

### Type Helpers
- **Status enum**: `StatusActive` (writable, visible), `StatusWithdrawn` (manually retired), `StatusExpired` (time-expired)
- **Source enum**: `SourceWitnessed` (directly observed), `SourceTold` (learned via gossip), `SourceDeduced` (inferred), `SourceUnknown` (unspecified)

### Registry Write API
- **Declare(factId string, opts DeclareOpts) error**: Creates a new active fact. Returns error if factId already exists. Stamps `DeclaredRound` to current game round. Persists immediately.
- **Withdraw(factId string)**: Transitions an active fact to withdrawn status. Idempotent. No-op if fact is already withdrawn or expired. Persists.
- **Expire(factId string)**: Transitions an active fact to expired status. Idempotent. No-op if fact is not active. Persists.
- **WithdrawAllBoundTo(mobTemplateId int) int**: Flips all active facts whose `WithdrawOnRespawnOf` field matches the given mob template id to withdrawn status. Returns count of facts affected. Atomic — either all are flipped or none. Used by auto-withdraw hook when a mob respawns. Persists.
- **PruneExpired() int**: Walks all active facts; flips any past their `ExpiryRound` to expired status. Returns count of facts expired. Called on-demand (currently via admin command; could be periodic). Persists.

### Registry Read API
- **GetFact(factId string) \*Fact**: Returns the fact with the given id, regardless of status, or nil if not found. Fast lookup. No filtering.
- **AllActiveFacts() \[\]\*Fact**: Returns a snapshot copy of every fact currently in active status. Used by gossip selection, dialogue gating, and admin displays. Lazy filter — withdrawn/expired facts are not included.
- **AllFactsByTag(tag string) \[\]\*Fact**: Returns active facts that include the given tag. Useful for scoped queries (e.g., "all politics facts" or "all death facts").
- **AllRows() \[\]\*Fact**: Returns every fact regardless of status. Admin/debug only.

### Awareness Write API
- **RecordHeardEvent(observerMobId int, eventId uint64)**: Appends an event id to the observer's heard_events FIFO. Bounded to `FactsHeardEventsMax` (default 32); oldest entries fall off when over cap. Deduplicates: no-op if eventId is already in the list. Lazy-loads or creates observer awareness. Persists synchronously.
- **RecordKnowsFact(observerMobId int, factId string, source Source)**: Records that the observer knows a fact and how they learned it. Creates observer record if missing. Idempotent — if the factId is already in the known list, no-op (existing Source and LearnedRound are preserved). Stamps `LastUpdatedRound` on change. Persists synchronously.
- **ForgetFact(observerMobId int, factId string)**: Removes a single fact from the observer's known list. Idempotent — no-op if the fact was not known. Stamps `LastUpdatedRound`. Persists synchronously.
- **ForgetAll(observerMobId int)**: Clears all awareness (heard events + known facts) for an observer. Stamps `LastUpdatedRound`. Persists synchronously.

### Awareness Read API
- **HeardEvent(observerMobId int, eventId uint64) bool**: Returns true iff the observer's heard_events FIFO includes the given event id. Lazy-loads cache on first call.
- **KnowsFact(observerMobId int, factId string) bool**: Returns true iff the observer's known_facts includes the given fact id, regardless of fact status. Lazy-loads cache on first call.
- **KnownFactsOf(observerMobId int) \[\]KnownFact**: Returns the joined view — walks the observer's known fact ids against the active fact registry. Returns only facts currently in StatusActive (lazy filter on read). Each entry holds the Fact pointer, Source, and LearnedRound. Used for dialogue gating and NPC decision-making.
- **AllForObserver(observerMobId int) \*Awareness**: Returns the raw awareness record for an observer (heard_events + known_facts). Used by admin commands and bulk operations.

### Load-Time API
- **LoadFromMobs(seeds \[\]MobAwarenessSeed, nameLookup func(mobId int) string)**: Seeds per-NPC awareness records from authored `knows_facts:` declarations on mob YAMLs. Called by `mobs.LoadDataFiles` after the mob registry is built. Wires the production `observerNameFor` callback to the real mob-name lookup function. For each seed, iterates the `KnowsFacts` slice and calls `RecordKnowsFact(mobId, factId, SourceWitnessed)` for each. No-op if a mob has no `knows_facts:` field.

## Global State

### Registry Cache and Synchronization
```go
var (
    registry      *Registry        // singleton; loaded once
    registryMu    sync.RWMutex     // protects registry during all reads/writes
    registrySaveMu sync.Mutex      // serializes disk writes to facts.yaml
)
```

The `registry` is a single in-memory instance populated at first access via `loadOrLazyInitRegistry()`. The `registryMu` is held:
- Read lock for queries (GetFact, AllActiveFacts, AllFactsByTag, AllRows) and lazy-load check.
- Write lock for mutations (Declare, Withdraw, Expire, WithdrawAllBoundTo, PruneExpired) and initialization.

The `registrySaveMu` serializes disk writes to prevent Windows `ERROR_SHARING_VIOLATION`. Mirrors the pattern in `internal/crimes` and `internal/knowledge`.

### Awareness Cache and Synchronization
```go
var (
    awarenessCache    = make(map[int]*Awareness)  // mob template id → observer record
    awarenessCacheMu  sync.RWMutex                // protects awarenessCache
    awarenessSaveMu   sync.Mutex                  // serializes awareness disk writes
)
```

Per-observer awareness records are lazy-loaded into `awarenessCache` (keyed by mob template id) via the double-check-lock pattern in `loadOrLazyInitAwareness()`. The `awarenessCacheMu` is held:
- Read lock for queries (HeardEvent, KnowsFact, KnownFactsOf, AllForObserver) and lazy-load check.
- Write lock for mutations (RecordHeardEvent, RecordKnowsFact, ForgetFact, ForgetAll) and initialization.

The `awarenessSaveMu` serializes disk writes. Each observer's awareness is stored in a separate file (`facts.awareness/{mobId}-{namesimple}.yaml`), so concurrent saves to different observers do not block; the mutex is a safety valve for concurrent writes to the same observer.

### Test Seams
- **roundForTest**: Injection hook for `currentRound()` — lets tests freeze or advance the round count without calling `util.GetRoundCount()`. Production never sets this; tests assign a closure.
- **heardEventsMaxForTest**: Injection hook for `effectiveHeardEventsMax()` — lets tests override the bounded FIFO cap without wiring the full balance-config stack.
- **observerNameFor**: Callback for mob-name lookup (by template id). Stub returns empty string by default. Called during awareness cache initialization and set to a real lookup function by `LoadFromMobs(seeds, nameLookup)`.

## Data Structure Design

### Fact Registry Shape (facts.yaml)
```yaml
facts:
  - id: king-dead
    description: The king is dead.
    significance: global
    declared_round: 1250
    expiry_round: 3000        # optional; 0 never expires
    zone: sanctum_basin       # optional
    region: temple_district   # optional
    tags:
      - politics
      - death
    withdraw_on_respawn_of: 5001  # optional; auto-withdraw when mob 5001 respawns
    status: active            # active, withdrawn, or expired
  - id: thornhall-razed
    description: The town of Thornhall was razed to the ground.
    significance: regional
    declared_round: 900
    status: active
```

Each fact is a stable, authored assertion. The `id` is a unique string key. The `Significance` field (from `worldevents.Significance` enum) classifies global/regional/local events. Optional fields: `Zone`, `Region` (spatial scope); `ExpiryRound` (automatic expiry); `Tags` (for scoped queries); `WithdrawOnRespawnOf` (mob template id to trigger withdrawal).

### Per-Observer Awareness Shape (facts.awareness/{mobId}-{namesimple}.yaml)
```yaml
observer_mob_id: 1141
observer_name: townguard
heard_events:
  - 102    # event id from worldevents log
  - 105
  - 103
known_facts:
  - fact_id: king-dead
    source: told
    learned_round: 1260
  - fact_id: thornhall-razed
    source: witnessed
    learned_round: 1050
last_updated_round: 1275
```

Per-observer files store heard events (a bounded FIFO of event ids) and known facts (fact id + source + round learned). The `LastUpdatedRound` tracks when the file was last modified. Gitignored at the directory level; runtime state only.

## Lifecycle (Callout)

Facts have three withdrawal signals:

1. **Manual withdrawal**: `Withdraw(factId)` flips status to withdrawn. Admin-driven or explicit game logic.
2. **Time-based expiry**: `PruneExpired()` checks all active facts against `ExpiryRound` and flips expired ones. On-demand; could be called from a periodic sweep.
3. **Auto-withdrawal on mob respawn**: Facts with `WithdrawOnRespawnOf` field set are withdrawn when that mob template respawns. Triggered by the `MobRoomChange_FactsAutoWithdraw` listener hook on mob room-change events. When mob 5001 (a questgiver) respawns, any facts declared with `withdraw_on_respawn_of: 5001` are automatically withdrawn.

Awareness reads filter withdrawn and expired facts lazily: `KnownFactsOf()` joins the observer's known_facts list against the registry and skips any facts not in `StatusActive`. This means withdrawn facts silently drop from view without explicit forget operations.

## Integration Notes

### Worldevents Integration
The `Fact` struct imports `worldevents.Significance` enum. At chunk 1.7 T1, worldevents received a stable `Id uint64` field (additive change), allowing awareness records to reference event ids reliably. The gossip system (`buildGossipLine` in chunk 1.7 T13) reads fact ids and event ids from both the awareness substrate (via `HeardEvent` / `RecordHeardEvent`) and the facts registry (via `AllActiveFacts`).

### Knowledge Package Parity (1.4)
The `internal/knowledge` package (1.4) maintains per-NPC × per-subject awareness of specific players and mobs. The `internal/facts` package (1.7) maintains per-NPC awareness of world facts. Both are independently authored, independently persisted, and independently queried. No cross-package mutations. Knowledge uses a `Subject` polymorphism (player or mob); facts use flat fact ids.

### Gossip System Integration (chunk 1.7 T13)
The `buildGossipLine` function in `internal/hooks/MobIdle_HandleIdleMobs.go` was migrated from in-memory `recentGossipEvents` TempData to persistent facts. It queries `facts.AllActiveFacts()` and `facts.HeardEvent()` to populate gossip candidates and filter what each NPC plausibly knows. It also reads `knowledge.KnownFactsOf()` to surface known facts in gossip output. The integration creates a dual-substrate gossip pool: facts (world truths) + knowledge (personal awareness).

### Mob YAML Seeding
Mob templates have an optional `knows_facts: [factId, ...]` field. At startup, `mobs.LoadDataFiles` calls `facts.LoadFromMobs(seeds, nameLookup)` to seed awareness records. Each mob's authored fact list is recorded as `SourceWitnessed` in that NPC's awareness. This establishes canonical NPC knowledge at startup.

### Auto-Withdraw Hook
The `internal/hooks/MobRoomChange_FactsAutoWithdraw.go` hook listens for `RoomChange` events on mobs. When a mob respawns (room change from nil to a real room), it calls `facts.WithdrawAllBoundTo(mobTemplateId)` to withdraw any facts bound to that mob via the `withdraw_on_respawn_of` field. Protects quest-specific facts from persisting across mob respawns.

### Admin Command
`internal/usercommands/admin.fact.go` registers a `fact` command with 10 subcommands:
```
fact list                                — all active facts
fact show <factId>                       — detail on one fact
fact declare <id> <description> [...]    — create a fact
fact withdraw <factId>                   — manual withdrawal
fact expire <factId>                     — manual expiry
fact prune-expired                       — run PruneExpired sweep
fact awareness <mobId>                   — show observer's heard_events + known_facts
fact teach <mobId> <factId>              — call RecordKnowsFact(witnessed)
fact forget <mobId> <factId>             — call ForgetFact
fact forget-all <mobId>                  — call ForgetAll
```

### Dependencies
- **configs**: Reads `FactsHeardEventsMax` (bounded FIFO cap).
- **util**: Calls `GetRoundCount()` for current round; `ConvertForFilename()` for awareness file paths.
- **mudlog**: Logs warnings on persistence errors.
- **worldevents**: Imports `Significance` enum (Global/Regional/Local).
- Intentionally does NOT import mobs (uses callback for name lookup) to avoid circular dependencies.

### Imported By
- **internal/mobs**: Calls `LoadFromMobs` during data-file startup.
- **internal/hooks**: MobRoomChange listener wires `WithdrawAllBoundTo`. MobIdle gossip wires `HeardEvent`, `AllActiveFacts`, `RecordHeardEvent`.
- **internal/usercommands**: Admin `fact` command.
- **Future: dialogue, tactics, town memory**: Will query `KnownFactsOf`, `AllActiveFacts`, `HeardEvent`.

## Testing Notes

### Test Files
- **facts_test.go**: ~150 lines covering registry operations (declare/collision/withdraw/expire/prune/WithdrawAllBoundTo), registry reads (AllActiveFacts/AllFactsByTag), awareness writes (RecordHeardEvent with bounded FIFO + dedup, RecordKnowsFact with idempotence, ForgetFact, ForgetAll), awareness reads (HeardEvent, KnowsFact, KnownFactsOf), lifecycle transitions.
- **persistence_test.go**: Tests for disk I/O (load/save round-trip), cache lifecycle, lazy-load double-check-lock correctness, concurrent saves.
- **types_test.go**: Data structure marshaling/unmarshaling tests.
- **test_main_test.go**: TestMain setup with temp-dir isolation via config override; `resetCaches()` helper clears both registry and awareness caches and removes facts.yaml from disk.

### Test Helpers and Patterns
- **resetCaches()**: Clears `registry`, wipes `awarenessCache`, removes facts.yaml from disk. Called between tests to ensure clean state.
- **roundForTest / heardEventsMaxForTest**: Injection hooks for testing; tests assign closures and must clear (`defer`) them after use.
- **In-memory pattern**: Tests do not rely on disk I/O for core logic tests. Disk I/O is tested separately in persistence_test.go. The `resetCaches()` helper keeps disk state synchronized with test expectations.

### Key Test Scenarios
- **Registry lifecycle**: Declare → GetFact (active) → Withdraw → GetFact (withdrawn) → AllActiveFacts (filtered).
- **Expiry**: Declare with ExpiryRound, advance round, PruneExpired, verify status flipped.
- **WithdrawAllBoundTo**: Declare multiple facts with WithdrawOnRespawnOf, call WithdrawAllBoundTo(mobId), verify count and status.
- **Bounded FIFO**: RecordHeardEvent 6 times with max=4; verify oldest entries fall off, dedup works.
- **Known facts idempotence**: RecordKnowsFact same id twice; second call is no-op.
- **Lazy filter on read**: Record a fact id in awareness, withdraw the fact, call KnownFactsOf; fact filtered out.
- **Lazy load**: First call to awareness functions for new mobId triggers disk load (if exists) or creates fresh record.
