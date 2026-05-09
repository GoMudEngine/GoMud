# Knowledge System Context

## Overview

The `internal/knowledge` package maintains per-NPC × per-subject facts about what each NPC has learned over time. Each observer (an NPC mob template) holds a collection of records keyed by subject — either a player or another NPC template — storing whether they have met, the subject's learned name, a bounded log of recent observations (rooms and rounds), and a list of witnessed crime IDs. Knowledge is recorded through two primary auto-write triggers: routine observation via the forager/caravan room-change hook, and crime witnessing via the 1.3 substrate (assault/murder/theft recordings). The package serves as a query layer for future systems (town justice, bounties, dialogue branching, tactical AI) that need to ask "what does this NPC actually know about that target right now?"

## Key Components

### Core Files
- **types.go**: Enums and data structures (`SubjectType`, `Subject`, `Source`, `Confidence`, `Observation`, `Record`, `ObserverFile`)
- **knowledge.go**: Public API (`RecordMet`, `RecordObservation`, `RecordName`, `RecordCrimeWitnessed`, `Forget`, `ForgetFact`, `Get`, `HasMet`, `NameOf`, `LastSeen`, `WitnessedCrimes`, `FrequentedRooms`, `AllForObserver`, `AllObserversOfPlayer`, `RecordRoutineObservers`); lazy cache initialization; test seams
- **decay.go**: Frequency analysis (`FrequentedRooms` — top-K rooms from bounded observation log)
- **persistence.go**: Disk I/O (`loadOrLazyInit`, `loadObserverFileFromDisk`, `saveObserverFile`); cache management; file path generation
- **knowledge_test.go**: Unit tests for all public functions, observation bounding, crime joining, and persistence
- **persistence_test.go**: Tests for disk I/O, cache lifecycle, and concurrent saves
- **test_main_test.go**: TestMain setup with environment-variable override for temp-dir isolation

## Key Functions

### Subject Helpers
- **PlayerSubject(userId int) Subject**: Constructs a `Subject` with `Type: SubjectPlayer` and `Id: userId`. Idempotent constructor.
- **MobSubject(mobId int) Subject**: Constructs a `Subject` with `Type: SubjectMob` and `Id: mobId`. The id refers to the mob TEMPLATE, not a specific instance.

### Write API
- **RecordMet(observerMobId int, subject Subject, room int, source Source)**: Marks that the observer has met the subject. Creates a fresh record if missing, sets `HasMet: true`, stamps `Source` and `Confidence: ConfidenceHigh`, records `LearnedRound` on first call, updates `LastSeenRoom`, `LastSeenRound`, and `LastUpdatedRound`. Persists synchronously.

- **RecordObservation(observerMobId int, subject Subject, room int)**: Appends an observation (room, round) to the bounded log. Creates record if missing. Deduplicates same-round observations at the tail. Truncates log to the most recent `KnowledgeObservationLogMax` entries (default 32) to bound memory. Persists synchronously.

- **RecordName(observerMobId int, subject Subject, name string, source Source)**: Records the name the observer learned for the subject. Creates record if missing. Deduplicates — no-op if the name is already set to the same value. Stamps `LastUpdatedRound`. Persists synchronously.

- **RecordCrimeWitnessed(observerMobId int, subject Subject, crimeId int)**: Records that the observer witnessed a crime by the subject. Creates record if missing, appends crimeId to the crimes-witnessed list. Deduplicates — no-op if crimeId is already recorded. Stamps `LastUpdatedRound`. Persists synchronously.

### Read API
- **Get(observerMobId int, subject Subject) \*Record**: Returns the in-memory record for the given observer and subject, or nil if the record does not exist. Triggers lazy cache load on first access for this observer but does not create a record.

- **HasMet(observerMobId int, subject Subject) bool**: Returns true iff the observer has a record for the subject with `HasMet: true`.

- **NameOf(observerMobId int, subject Subject) (string, bool)**: Returns the learned name if a record exists and `NameLearned` is set, otherwise `("", false)`.

- **LastSeen(observerMobId int, subject Subject) (int, uint64, bool)**: Returns the room, round, and success boolean. Caller can check staleness by comparing the returned round to `util.GetRoundCount()`.

- **FrequentedRooms(observerMobId int, subject Subject, topK int) \[\]RoomCount**: Returns the top-K rooms from the observation log, sorted by frequency descending (ties broken by room ID ascending). Returns nil if the record does not exist or the log is empty. `RoomCount` struct contains `Room` and `Count` integers.

- **WitnessedCrimes(observerMobId int, subject Subject) \[\]WitnessedCrime**: Returns the joined view of the observer's witnessed crime IDs enriched with live data from the 1.3 crimes substrate. Each entry holds the `CrimeId`, crime `Kind` (assault/murder/theft), and `ResolvedRound` (0 if unresolved). Lazy filter on read: walks all loaded factions' crimes and returns only those whose IDs match the record's crimes-witnessed list. Callers decide whether to surface stale (resolved) entries. Returns nil if the record does not exist or has no witnessed crimes.

### Bulk/Helper API
- **AllForObserver(observerMobId int) \[\]\*Record**: Returns a snapshot copy of every record this observer holds. Returns nil if the observer has no records. Safe for concurrent reads; a shallow copy of the records slice is returned.

- **AllObserversOfPlayer(userId int) \[\]int**: Returns the list of observer mob template IDs that hold any record about this player. Best-effort: walks only loaded cache entries (does not scan disk). Useful for answering "who in the world knows this player?"

- **RecordRoutineObservers(movingInstanceId int, movingTemplateId int, room \*rooms.Room)**: Auto-write helper used by the forager/caravan room-change hook. Iterates all other NPC mobs currently in the room and writes observation + met records for each, treating the moving entity (identified by its template ID) as the subject. Skips the moving entity's own instance to avoid self-observation. Used to populate routine observation logs as NPCs traverse the world.

### Forget API
- **Forget(observerMobId int, subject Subject)**: Removes the entire record for the subject from the observer's file. Idempotent — no-op if the record does not exist. Intentionally does NOT cascade to opinion (1.1) or reputation (1.2/1.3) state; those systems track faction-level trust derived from witnessed behavior over time, not individual memory. Persists synchronously.

- **ForgetFact(observerMobId int, subject Subject, fact string)**: Clears a single fact from the record without removing the record itself. Valid fact keys are `"name"`, `"observations"`, and `"crimes"`. Unknown fact keys are a no-op. Stamps `LastUpdatedRound`. Same no-cascade policy as `Forget`. Persists synchronously.

## Global State

### Cache and Mutex
- **knowledgeCache**: `map[int]*ObserverFile` keyed by observer mob template id. Holds every loaded observer's knowledge table in memory.
- **knowledgeCacheMu**: `sync.RWMutex` protecting knowledgeCache and all in-cache mutations. Write operations (Record\*, Forget, ForgetFact) acquire the write lock; read operations acquire the read lock.
- **saveMu**: `sync.Mutex` serializing disk writes to prevent Windows `ERROR_SHARING_VIOLATION` on overlapping writes to the same observer file. Mirrors the `internal/crimes` and `internal/opinions` patterns.

### Test Seams
- **roundForTest**: Injection hook for `currentRound()` — lets tests freeze or advance the round count without reading `util.GetRoundCount()`. Production never sets this; tests assign a closure and must clear it after use.
- **observationLogMaxForTest**: Injection hook for `observationLogMax()` — lets tests override `configs.GetBalanceConfig().KnowledgeObservationLogMax` without standing up the full configs stack.
- **observerNameFor**: Injection hook for mob-name lookup — lets tests provide a fake implementation of `mobs.GetMobSpec()`. Production maps to the real template lookup.

## Data Structure Design

### Subject Polymorphism
```go
type SubjectType string
const (
    SubjectPlayer SubjectType = "player"
    SubjectMob    SubjectType = "mob"
)

type Subject struct {
    Type SubjectType `yaml:"type"`
    Id   int         `yaml:"id"`
}
```
The `Subject` is the polymorphic record key. For players, `Type: SubjectPlayer` and `Id` is the userId. For mobs, `Type: SubjectMob` and `Id` is the mob TEMPLATE id (not instance). This ensures knowledge persists across the subject mob's deaths and respawns.

### Source and Confidence Enums
```go
type Source string
const (
    SourceWitnessed Source = "witnessed"
    SourceTold      Source = "told"
    SourceDeduced   Source = "deduced"
    SourceUnknown   Source = "unknown"
)

type Confidence string
const (
    ConfidenceHigh Confidence = "high"
    ConfidenceMed  Confidence = "med"
    ConfidenceLow  Confidence = "low"
)
```
Every record carries source (how knowledge was acquired) and confidence (reliability tier) fields. v1 writes only `SourceWitnessed` and `ConfidenceHigh`; other enum values are placeholders for future gossip/inference write paths.

### Observation Log
```go
type Observation struct {
    Room  int    `yaml:"room"`
    Round uint64 `yaml:"round"`
}
```
Each observation is a (room, round) pair. The log is bounded FIFO — `RecordObservation` truncates to the most recent `KnowledgeObservationLogMax` entries when the log exceeds capacity. No time-based decay is applied; callers check staleness by comparing round to current round.

### Record Structure
```go
type Record struct {
    Subject          Subject       `yaml:"subject"`
    HasMet           bool          `yaml:"has_met"`
    NameLearned      string        `yaml:"name_learned,omitempty"`
    Source           Source        `yaml:"source"`
    Confidence       Confidence    `yaml:"confidence"`
    LastSeenRoom     int           `yaml:"last_seen_room"`
    LastSeenRound    uint64        `yaml:"last_seen_round"`
    Observations     []Observation `yaml:"observations,omitempty"`
    CrimesWitnessed  []int         `yaml:"crimes_witnessed,omitempty"`
    LearnedRound     uint64        `yaml:"learned_round"`
    LastUpdatedRound uint64        `yaml:"last_updated_round"`
}
```
The record structure captures all v1 fact types:
- **HasMet**: Boolean — the observer and subject have been in the same room.
- **NameLearned**: String — the learned name for the subject (empty if not yet learned).
- **Source/Confidence**: Applied uniformly to all facts in this record (all facts in v1 are witnessed/high).
- **LastSeenRoom/LastSeenRound**: The most recent observation (room and round).
- **Observations**: Bounded FIFO log of recent (room, round) pairs.
- **CrimesWitnessed**: List of crime IDs the observer witnessed.
- **LearnedRound**: Round the record was first created (learned_round).
- **LastUpdatedRound**: Round of the most recent mutation (any Record\* call).

### Observer File (On Disk)
```yaml
# _datafiles/world/dogmud/knowledge/{mobId}-{namesimple}.yaml
observer_mob_id: 99
observer_name: records_clerk_pell
records:
  - subject:
      type: player
      id: 17
    has_met: true
    name_learned: "Bob"
    source: witnessed
    confidence: high
    last_seen_room: 462
    last_seen_round: 2065719
    observations:
      - {room: 462, round: 2065600}
      - {room: 463, round: 2065650}
      - {room: 462, round: 2065719}
    crimes_witnessed: [3, 5]
    learned_round: 2065600
    last_updated_round: 2065719
  - subject:
      type: mob
      id: 101
    has_met: true
    name_learned: ""
    source: witnessed
    confidence: high
    last_seen_room: 500
    last_seen_round: 2065700
    observations:
      - {room: 500, round: 2065650}
      - {room: 500, round: 2065700}
    crimes_witnessed: []
    learned_round: 2065650
    last_updated_round: 2065700
```
Files are stored at `_datafiles/world/dogmud/knowledge/{observerMobId}-{namesimple}.yaml`, gitignored. The `ObserverFile` struct holds `ObserverMobId`, `ObserverName` (for filename reconstruction), and `Records` (slice of `\*Record`). One file per observer mob template; all instances of that template share the same knowledge.

## Decay Rules

Unlike the 1.1 opinions package (exponential decay over rounds), the knowledge system applies **per-fact-type decay rules**. Identity facts are permanent; observations self-bound via FIFO; crimes-witnessed inherit decay from the 1.3 crimes substrate.

| Fact Type | Decay Rule | Rationale |
|-----------|-----------|-----------|
| **Has-Met / Name Learned** | Never decays | Permanent identity facts. Once an NPC learns a player's name or meets them, that memory sticks. |
| **Last-Seen (Room/Round)** | Implicit staleness | No time-based decay. Exposed as-is to callers; they compare `LastSeenRound` to current round to decide staleness. Future systems (gossip, rumor) can use the raw round data for temporal reasoning. |
| **Observation Log** | Bounded FIFO | Self-bounding at `KnowledgeObservationLogMax` (default 32). Newest entries always present; oldest entries drop off as new observations arrive. No separate stale-expiry needed. |
| **Crimes-Witnessed** | Lazy filter via 1.3 | Records list crime IDs. `WitnessedCrimes()` joins against the live crimes substrate at read time. Callers see crime resolve status and decide whether to surface stale entries. No knowledge layer storage of crime staleness. |

## Integration Notes

### Auto-Write Triggers

**Forager/Caravan Room-Change Hook** (`MobRoomChange_KnowledgeObservers`)
- When a forager or caravan NPC moves to a new room, the hook calls `RecordRoutineObservers(movingInstanceId, movingTemplateId, room)`.
- This writes observation and met records for all other NPCs currently in the room, using the moving NPC's template ID as the subject.
- Automates routine observation: as NPCs travel, they accumulate knowledge of one another's movement patterns and locations.

**Crime Witnessing** (1.3 → 1.4 Integration)
- Three call sites record crimes in the 1.3 crimes substrate and also write knowledge records for witnesses:
  1. **assault.go** (`internal/usercommands/attack.go`, `recordAssaultCrime`): Calls `knowledge.RecordMet` and `knowledge.RecordCrimeWitnessed` for each witness in the room when a player first attacks a faction-member mob.
  2. **MobDeath_FactionRep.go** (`internal/hooks/MobDeath_FactionRep.go`): Calls `knowledge.RecordCrimeWitnessed` for each witness when recording or upgrading a murder crime.
  3. **steal.go** (`internal/usercommands/skill.skullduggery.steal.go`): Calls `knowledge.RecordCrimeWitnessed` for each witness when a player fails a steal/pickpocket on a faction member.
- Knowledge integration is at the call site, not in the crimes package (crimes is agnostic to knowledge).

### Cross-Substrate Intersections

The knowledge layer reads from two other v1 substrates but does not cascade writes:

| Interaction | Direction | Purpose |
|---|---|---|
| **Opinions (1.1)** | Read-only in future consumers | Strategic systems may query both knowledge and opinion when deciding NPC behavior. No write coupling. |
| **Factions (1.2)** | Read-only in WitnessedCrimes | `WitnessedCrimes()` walks faction definitions to locate crime IDs in the crimes substrate. |
| **Crimes (1.3)** | Read-only via join in WitnessedCrimes | Crime IDs are stored in the knowledge record. `WitnessedCrimes()` enriches them with live crime status (kind, resolved round). |

**No cascade on Forget**: Calling `Forget` or `ForgetFact` on a knowledge record does NOT mutate opinion or crime state. Amnesia spells or debug commands may eventually implement their own cascade logic when needed, but v1 keeps the boundaries clean.

### Admin Command
- **admin.knowledge.go**: Subcommands `knowledge show/forget/frequented` expose the API for debugging and testing.
  - `knowledge show <observer> [subject]`: View records for an observer, optionally filtered by subject.
  - `knowledge forget <observer> <subject>`: Call `Forget()` to remove a record.
  - `knowledge frequented <observer> <subject> [topK]`: Call `FrequentedRooms()` to show the top-K rooms for a subject.

### Forager/Caravan Discriminators
- The knowledge package assumes two helper functions exist in the forager and caravan packages:
  - **IsForagerMob(templateId int) bool**: Returns true iff this template ID is a forager NPC.
  - **IsCaravanMob(templateId int) bool**: Returns true iff this template ID is a caravan NPC.
- The hook uses these to decide whether to call `RecordRoutineObservers` for a given mob.

### Dependencies
- **configs**: Reads `Balance.KnowledgeObservationLogMax` (default 32) for observation log bounding.
- **mobs**: Calls `GetMobSpec()` to fetch mob display names and `GetInstance()` to resolve mob instances in rooms when writing routine observations.
- **rooms**: Calls `GetMobs()` on the room object to iterate occupants for `RecordRoutineObservers`.
- **factions**: Calls `AllDefinitions()` to iterate faction crime logs in `WitnessedCrimes()`.
- **crimes**: Calls `AllForFaction()` to enrich witnessed crime IDs with live status in `WitnessedCrimes()`.
- **util**: Uses `GetRoundCount()` for timestamps and `ConvertForFilename()` for path generation.
- **mudlog**: Logs warnings on disk I/O errors.

### Imported By
- `internal/usercommands/attack.go` (assault crime witnessing)
- `internal/usercommands/skill.skullduggery.steal.go` (theft crime witnessing)
- `internal/hooks/MobDeath_FactionRep.go` (murder crime witnessing)
- `internal/usercommands/admin.knowledge.go` (admin API)
- `internal/modules/mob.room_change.go` or hook registration (forager/caravan observation auto-write)

## Testing Notes

### Test Files
- **knowledge_test.go**: ~400 lines covering RecordMet (fresh vs. idempotent, LearnedRound vs. LastSeenRound), RecordObservation (bounded FIFO, same-round dedup, tail truncation), RecordName (idempotent on same name), RecordCrimeWitnessed (dedup, lazy join with crimes substrate), Forget/ForgetFact (removal, fact-level clearing), HasMet, NameOf, LastSeen, WitnessedCrimes (joined view with live resolve status), FrequentedRooms (frequency tally, top-K sorting, stable tie-breaking by room ID), AllForObserver (snapshot copy), AllObserversOfPlayer (reverse lookup), and RecordRoutineObservers (mob-in-room iteration, moving entity exclusion).
- **persistence_test.go**: ~90 lines covering disk I/O (write, read, filename generation), cache lifecycle (lazy init, double-check-lock), and concurrent save serialization via saveMu.
- **test_main_test.go**: TestMain that sets `DOGMUD_KNOWLEDGE_DIR_OVERRIDE` to a temp dir so tests don't touch the real knowledge directory.

### Test Seam Pattern
Tests inject fake implementations via global function pointers (`roundForTest`, `observationLogMaxForTest`, `observerNameFor`) without building the full mobs, rooms, or factions stacks. After each test, seams are cleared so production code sees nil and uses the real implementations. The pattern is identical to the 1.3 crimes package.

### Key Test Scenarios
- `RecordMet` creates on first call, updates on subsequent calls, preserves `LearnedRound`.
- `RecordObservation` appends to log, deduplicates same-round tail entries, truncates log when it exceeds max.
- `RecordName` is idempotent if the name is unchanged; sets `LastUpdatedRound` on change.
- `RecordCrimeWitnessed` appends crime ID, deduplicates, creates record if missing.
- `Forget` removes the entire record; `ForgetFact` clears individual fields.
- `FrequentedRooms` tallies room frequencies, returns top-K sorted by count (ties by room ID), returns nil for missing records.
- `WitnessedCrimes` joins crime IDs against live substrate, returns enriched view with kind and resolve status.
- `AllObserversOfPlayer` performs reverse lookup across in-memory cache.
- `RecordRoutineObservers` iterates room occupants, skips moving entity, writes observations for each.
- Persistence: files are created on first write, loaded lazily on first access, saved synchronously on every Record\* call.
