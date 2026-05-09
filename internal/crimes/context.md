# Crimes System Context

## Overview

The `internal/crimes` package maintains per-faction crime logs tracking harmful acts (murder, assault, theft) committed against faction-member mobs. Each crime record identifies the victim, location, time, perpetrator (player, mob, or unknown), and witness status. Crimes are recorded synchronously with the act and persisted to disk. Crimes drive faction-rep changes through a unified combat hookup; they are also consumed by admin commands and (in future chunks) by town justice, bounties, and rumor systems.

## Key Components

### Core Files
- **types.go**: Data structures (`Kind`, `PerpType`, `Perpetrator`, `Crime`, `FactionCrimes`)
- **crimes.go**: Public API (`Record`, `Resolve`, `FindRecentAssault`, `AllForFaction`, `AllForPlayer`, `WitnessesInRoom`, `IdentifiedPerp`, `UpgradeAssaultToMurder`, `PruneStale`); lazy cache initialization; test seams
- **persistence.go**: Disk I/O (`loadCrimesFromDisk`, `saveCrimesToDisk`, `SaveAllCrimes`, `ClearCache`); cache management; file path generation; test overrides
- **crimes_test.go**: Unit tests for all public functions, four-case murder upgrade logic, witness detection, and persistence
- **persistence_test.go**: Tests for disk I/O, cache lifecycle, and concurrent saves
- **test_main_test.go**: TestMain setup with environment-variable override for temp-dir isolation

## Key Functions

### Recording and Upgrade
- **Record(factionIds \[\]string, kind Kind, perp Perpetrator, victim \*mobs.Mob, instanceId int, roomId int, zone string, hadExternalWitness bool) \[\]int**: Creates a new crime row on each faction's log. Returns crime IDs (parallel to factionIds). `hadExternalWitness` indicates whether a non-victim faction-aligned mob was present at recording time; sticky field used by murder-upgrade logic to preserve perpetrator identity. Persists synchronously per-faction.

- **UpgradeAssaultToMurder(factionId string, crimeId int, perp Perpetrator, instanceId int, roomId int, zone string, preserveExistingPerp bool)**: Mutates an existing assault crime row to `kind: murder`, refreshing room and round to the death event. Idempotent — no-op if the crime is already not an assault. Persists synchronously. `preserveExistingPerp=true` keeps the assault-time perpetrator; `preserveExistingPerp=false` overwrites with the supplied perp. Used by the four-case murder-upgrade logic in `MobDeath_FactionRep.go`.

### Querying and Resolution
- **FindRecentAssault(factionId string, userId int, lookbackRounds uint64) \*Crime**: Returns the most recent unresolved assault crime committed by userId against factionId within lookbackRounds. Returns nil if no match. Murder rows and resolved rows are ignored. Used by combat-death hookup to detect assault-to-murder escalation.

- **AllForFaction(factionId string, includeResolved bool) \[\]\*Crime**: Returns all crimes against the given faction. Pass `includeResolved=false` to skip cleared records.

- **AllForPlayer(userId int, includeResolved bool) \[\]\*Crime**: Returns all crimes naming userId as the identified perpetrator across all factions. Walks the in-memory cache; does not load from disk for untouched factions (admin may add a disk-walking variant later).

- **Resolve(factionId string, crimeId int, resolvedBy string)**: Marks a specific crime as resolved. Idempotent — re-resolving is a no-op (preserves original resolved_round and resolved_by). Persists synchronously.

- **PruneStale(factionId string) int**: Resolves all unresolved crimes older than `Balance.CrimeStaleAfterRounds` with reason "stale". Returns count of rows resolved. Safety net for indefinite-storage growth; primary expiry is consumer-driven (town justice fines, redemption quests).

### Witness and Perpetrator Resolution
- **WitnessesInRoom(factionIds \[\]string, room \*rooms.Room, excludeInstanceId int) \[\]int**: Returns the list of mob instance IDs in the given room whose mob template's Groups overlap any of factionIds. Pass `excludeInstanceId = victim's instance` for murder (victim is dead); pass 0 for assault and theft (victim is alive and a self-witness).

- **IdentifiedPerp(userId int, witnesses \[\]int) Perpetrator**: Returns `PerpPlayer` if witnesses is non-empty, otherwise `PerpUnknown`. Helper that centralizes the perpetrator-identification logic.

## Global State

### Cache and Mutex
- **crimeCache**: `map[string]*FactionCrimes` keyed by factionId. Holds every loaded faction crime log in memory.
- **crimeCacheMu**: `sync.RWMutex` protecting crimeCache.
- **saveMu**: `sync.Mutex` serializing file I/O to prevent Windows `ERROR_SHARING_VIOLATION` on overlapping writes to the same faction's crime file. Mirrors the `internal/opinions` and `internal/factions` patterns.

### Test Seams
- **roundForTest**: Injection hook for `currentRound()` — lets tests freeze or advance the round count without reading `util.GetRoundCount()`.
- **staleAfterForTest**: Injection hook for `currentStaleAfter()` — lets tests override `Balance.CrimeStaleAfterRounds` without standing up the full configs stack.

## Data Structure Design

### Crime Kind Enum
```go
const (
    KindAssault Kind = "assault"  // first-aggression on a faction member
    KindMurder  Kind = "murder"   // death of a faction member
    KindTheft   Kind = "theft"    // failed steal/pickpocket on a faction member
)
```

### Perpetrator Type Enum
```go
const (
    PerpPlayer  PerpType = "player"   // identified player userId
    PerpMob     PerpType = "mob"      // mob mobId (future mob-on-mob crimes)
    PerpUnknown PerpType = "unknown"  // act happened, no witness to identify
)
```

### Perpetrator Structure
```go
type Perpetrator struct {
    Type PerpType `yaml:"type"`
    Id   int      `yaml:"id,omitempty"`  // userId or mobId; absent for unknown
}
```

### Crime Row
```go
type Crime struct {
    Id               int         `yaml:"id"`           // monotonic per-faction
    Kind             Kind        `yaml:"kind"`         // assault|murder|theft
    Zone             string      `yaml:"zone"`
    RoomId           int         `yaml:"room_id"`
    Round            uint64      `yaml:"round"`        // round recorded
    VictimMobId      int         `yaml:"victim_mob_id"`
    VictimInstanceId int         `yaml:"victim_instance_id"`
    Perpetrator      Perpetrator `yaml:"perpetrator"`
    ResolvedRound    uint64      `yaml:"resolved_round"`  // 0 = unresolved
    ResolvedBy       string      `yaml:"resolved_by"`     // "stale", "fine paid", etc.
    HadExternalWitness bool      `yaml:"had_external_witness,omitempty"`  // sticky
}
```

### Per-Faction Crime Log (gitignored, not committed)
```yaml
# _datafiles/world/dogmud/factions.crimes/{slug}.yaml
faction_id: warren
crimes:
  - id: 1
    kind: assault
    zone: labyrinth_low
    room_id: 2045
    round: 2000100
    victim_mob_id: 3102
    victim_instance_id: 51234
    perpetrator:
      type: player
      id: 17
    resolved_round: 0
    resolved_by: ""
    had_external_witness: true
  - id: 2
    kind: murder
    zone: labyrinth_low
    room_id: 2046
    round: 2000200
    victim_mob_id: 3102
    victim_instance_id: 51235
    perpetrator:
      type: unknown
    resolved_round: 0
    resolved_by: ""
    had_external_witness: false
```

Stored lazily on first mutation for each faction. The `id` is monotonic per-faction. The `perpetrator` is only meaningful if `type: player` (id holds userId) or `type: mob` (id holds mobId); `type: unknown` has no id. The `resolved_round` and `resolved_by` are only filled after `Resolve()` is called. The `had_external_witness` flag is true when a non-victim faction-aligned mob was present at crime recording time; it persists through `UpgradeAssaultToMurder` to inform the four-case murder decision logic.

## Integration Notes

### Crime Recording Call Sites
- **assault.go** (`internal/usercommands/attack.go`, `recordAssaultCrime`): Records assault crime at first-aggression on a faction-member mob. Victim is alive (self-witness). Passes `hadExternalWitness = true` if other faction-members are in the room.
- **kill.go** / **MobDeath_FactionRep.go** (`internal/hooks/MobDeath_FactionRep.go`): Records or upgrades murder crime when a player kills a faction-member mob. Implements the four-case upgrade logic (see "Four-Case Murder Upgrade" section below). Victim is dead (not a self-witness); `hadExternalWitness = false` for fresh-murder path.
- **steal.go** (`internal/usercommands/skill.skullduggery.steal.go`): Records theft crime at FAILED steal/pickpocket on a faction member. Successful theft is silent; only the failed attempt produces a record.

### Knowledge Integration (Chunk 1.4)
Each crime call site also writes knowledge records for witnesses via `knowledge.RecordCrimeWitnessed` and `knowledge.RecordMet`, connecting witnesses to the perpetrator and marking that they met a player. The crimes package is agnostic to knowledge; the integration point is at the caller (attack.go, MobDeath_FactionRep.go, steal.go).

### Rep Change Consumers
- **MobDeath_FactionRep.go**: Bumps killer's rep via `factions.BumpRep` when a murder crime is recorded. The delta depends on whether the assault was prior-recorded (incremental delta) or fresh (full murder delta).

### Admin Command
- **admin.crime.go**: Subcommands `crime list/show/resolve/prune-stale` expose the API.

### Dependencies
- **factions**: Calls `FactionsForMob` to determine which factions a mob belongs to, and `GetDefinition` to validate faction existence.
- **mobs**: Calls `GetMobSpec` to validate the victim and `GetInstance` to check witnesses in a room.
- **rooms**: Calls `LoadRoom` to fetch the room object for witness queries.
- **configs**: Reads `Balance.CrimeStaleAfterRounds` (default ~168 hours in rounds) and rep deltas (`CrimeRepDeltaAssault`, `CrimeRepDeltaMurder`) for rep calculations.
- **util**: Uses `GetRoundCount()` for timestamps and `ConvertForFilename` for path generation.
- **mudlog**: Logs warnings on disk I/O errors.

### Imported By
- `internal/hooks/MobDeath_FactionRep.go` (four-case upgrade + fresh murder)
- `internal/usercommands/attack.go` (assault recording)
- `internal/usercommands/skill.skullduggery.steal.go` (theft recording)
- `internal/usercommands/admin.crime.go` (admin API)
- `internal/knowledge/*` (witnesses, chunk 1.4)

## Four-Case Murder Upgrade Logic

When a player kills a faction-member mob in `MobDeath_FactionRep.go`, the engine checks for an unresolved assault crime committed by the same player against the same faction within 100 rounds. If an assault row exists, it is upgraded in-place to murder; if not, a fresh murder row is recorded. The upgrade uses two flags — `currentExternal` (whether witnesses are present now) and `assault.HadExternalWitness` (whether the prior assault was witnessed) — to implement four distinct cases:

### Case A: External Witness Now
- **Condition**: `currentExternal = true` (witnesses in the room at death time)
- **Action**: Upgrade the assault to murder. Set perpetrator to identified (the killer is observed). Pay the incremental rep delta (`CrimeRepDeltaMurder - CrimeRepDeltaAssault`). Write knowledge for witnesses linking them to the crime and the player.
- **Rationale**: The witnesses saw the killing blow; perpetrator identity is confirmed. Rep change is incremental because the assault already paid a penalty.

### Case B: No External Witness Now, Assault Was Externally Witnessed
- **Condition**: `currentExternal = false` AND `assault.HadExternalWitness = true`
- **Action**: Upgrade the assault to murder. Keep the existing perpetrator (assault was seen; identity is known). Pay no additional rep delta.
- **Rationale**: No one saw the killing blow, but the assault (and therefore the perpetrator) was publicly known. Rep doesn't change because the unresolved assault row already penalized the player.

### Case C: Lone Assault, Lone Murder
- **Condition**: `currentExternal = false` AND `assault.HadExternalWitness = false`
- **Action**: Upgrade the assault to murder. Set perpetrator to unknown (no one witnessed either blow). Refund the assault rep delta.
- **Rationale**: The fight was private from start to finish. Record the murder (for future forensic/rumor systems) but clear the rep penalty since no witness was present to establish guilt.

### Fresh Murder (No Prior Assault)
- **Condition**: No unresolved assault row found
- **Action**: Record a fresh murder row with `hadExternalWitness = false`. If witnesses are present, set perpetrator to identified; if not, set to unknown. Pay the full murder rep delta only if perpetrator is identified.
- **Rationale**: Murder as a standalone act (not an escalation from assault). Rep applies only if identity is confirmed.

## Testing Notes

### Test Files
- **crimes_test.go**: ~250 lines covering Record (faction batching, hadExternalWitness recording), FindRecentAssault (lookback window, unresolved filtering), UpgradeAssaultToMurder (in-place mutation, preserveExistingPerp toggle), AllForFaction/AllForPlayer (filtering), WitnessesInRoom (faction membership overlap), IdentifiedPerp (witness-based identification), PruneStale (age cutoff), and the four-case murder upgrade scenarios.
- **persistence_test.go**: ~80 lines covering disk I/O, cache lifecycle, lazy initialization, and concurrent save serialization via saveMu.
- **test_main_test.go**: TestMain that sets `DOGMUD_FACTIONS_CRIMES_DIR_OVERRIDE` to a temp dir so tests don't touch the real factions.crimes directory.

### Test Seam Pattern
Tests inject fake implementations via global function pointers (`roundForTest`, `staleAfterForTest`) without building the full configs, mobs, or rooms stacks. After each test, seams are cleared so production code sees nil and uses the real implementations.

### Key Test Cases
- `Record` batches across multiple factions and stamps `hadExternalWitness`.
- `FindRecentAssault` searches in reverse order and respects lookback window.
- `UpgradeAssaultToMurder` mutates kind and optionally preserves perpetrator.
- `AllForFaction` filters by resolved status.
- `AllForPlayer` walks cache for all matching crimes.
- `WitnessesInRoom` checks faction-membership overlap and excludes the specified instance.
- `IdentifiedPerp` returns `PerpPlayer` iff witnesses is non-empty.
- `PruneStale` respects the age threshold and returns count.
- Four-case murder upgrade: Case A (external now → identified + incremental delta), Case B (assault external, murder not → keep perp, no delta), Case C (lone → unknown, refund), fresh (no assault → full delta if identified).
