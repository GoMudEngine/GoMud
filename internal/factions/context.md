# Factions System Context

## Overview

The `internal/factions` package maintains institutional per-player reputation with game factions on a signed [-100, +100] scale. Each faction has a definition file with default starting reputation, display name, description, and ally/enemy relationship declarations. Per-player rep is cached in memory and persisted to disk. Combat (mob death), quests, admin commands, and NPC behavior checks all call into factions to read or mutate faction standing — the package is the persistent record of "how does the faction view this player right now."

## Key Components

### Core Files
- **types.go**: Data structures (`Definition`, `RepEntry`, `FactionRep`, `RepMin`/`RepMax` constants)
- **registry.go**: Eager definition loader (`LoadAllDefinitions`, `GetDefinition`, `AllDefinitions`); ally/enemy validation
- **factions.go**: Public API (`GetRep`, `SetRep`, `BumpRep`, `TierFor`, `FactionsForMob`, `IsPeacefulToward`); lazy cache initialization
- **persistence.go**: Disk I/O (`loadRepFromDisk`, `saveRepToDisk`, `SaveAllRep`, `ClearCache`); cache management; test seams
- **factions_test.go**: Unit tests for core API (Get/Set/Bump/TierFor, FactionsForMob, IsPeacefulToward)
- **registry_test.go**: Tests for eager definition loading and ally/enemy validation
- **persistence_test.go**: Tests for disk I/O, cache lifecycle, and concurrent saves
- **test_main_test.go**: TestMain setup with environment-variable overrides for temp-dir isolation

## Key Functions

### Reading Faction Definition
- **GetDefinition(factionId string) \*Definition**: Returns the immutable Definition for a factionId, or nil if unknown. Pure read; no side effects. Called by cache initialization and consumer validation.

- **AllDefinitions() [\]\*Definition**: Returns a snapshot of every loaded definition. Useful for listing all factions in admin commands. Read-only — callers must not mutate returned slice or its elements.

### Reading Rep State
- **GetRep(factionId string, userId int) int**: Returns the player's reputation with the given faction, or the faction's `default_rep` if no row exists. Pure read — does not write to disk. Lazy cache priming may add an empty `FactionRep` to the cache on first access but no file is created. Returns 0 if the faction is unknown.

- **TierFor(factionId string, userId int) opinions.Tier**: Sugar for `opinions.TierOf(GetRep(factionId, userId))` — returns one of five tiers (`TierHostile`, `TierCold`, `TierNeutral`, `TierWarm`, `TierFriendly`) instead of raw score. Reuses the opinions package's banding logic — no duplicate enum.

### Mutating Rep State
- **SetRep(factionId string, userId int, rep int)**: Assigns an absolute rep value (clamped to [-100, +100]), stamps `last_updated_round` to the current round, and persists synchronously. Used for admin overrides and player resets.

- **BumpRep(factionId string, userId int, delta int)**: Adds delta to the current rep (starting from `default_rep` if no row exists), clamps to [-100, +100], re-stamps the round, and persists. No-op with warning if the faction is unknown. The everyday mutator — combat, quests, and dialogue hooks all call this. No auto-propagation through the ally/enemy graph.

### Consumer Queries
- **FactionsForMob(mob \*mobs.Mob) \[\]string**: Returns the subset of `mob.Groups` that have definition files. Used by combat hookup (which factions get bumped on death) and by consumer code that needs to know "which factions does this mob belong to?" Returns nil if the mob is nil.

- **IsPeacefulToward(mob \*mobs.Mob, userId int) bool**: Returns true if the player has `TierWarm` or higher with at least one of the mob's defined factions. Used by `lookfortrouble.go` and `behaviortree/actions_party.go` to gate whether the mob initiates combat on sight. Returns false if the mob has no defined factions.

### Loading and Persistence
- **LoadAllDefinitions() error**: Reads every YAML file in `_datafiles/world/dogmud/factions/` and populates the in-memory registry. Validates that every `allies:` and `enemies:` reference resolves to a loaded faction; **panics on unknown reference** so authoring errors surface at boot, not at runtime. Idempotent — safe to call multiple times. Subsequent calls fully replace the registry contents. Called once at server startup.

- **SaveAllRep()**: Flushes every cached `FactionRep` to disk. Defined for parity with `opinions.SaveAllOpinions`; not currently wired into shutdown (synchronous-on-mutation save covers persistence).

## Global State

### Caches and Mutexes
- **definitions**: `map[string]*Definition` keyed by `factionId`. Holds every loaded faction definition in memory. Immutable after `LoadAllDefinitions()` completes.
- **definitionsMu**: `sync.RWMutex` protecting the definitions map.
- **repCache**: `map[string]*FactionRep` keyed by `factionId`. Holds every faction's per-player rep table in memory. Lazy-initialized on first `GetRep`/`SetRep`/`BumpRep` for each faction.
- **repCacheMu**: `sync.RWMutex` protecting the repCache map.
- **saveMu**: `sync.Mutex` serializing file I/O to prevent Windows `ERROR_SHARING_VIOLATION` on overlapping writes to the same faction's rep file. Mirrors the `internal/opinions` pattern.

### Test Seams
- **roundForTest**: Injection hook for `currentRound()` — lets tests freeze or advance the round count without reading `util.GetRoundCount()`.

## Data Structure Design

### Faction Definition (committed)
```yaml
# _datafiles/world/dogmud/factions/{slug}.yaml
faction_id: warren
display_name: "The Warren"
description: |
  A coalition of warren scouts, warriors, and the tunnel shaman that
  have carved out territory in the Labyrinth of Low Tunnels.
  Surface-dwellers are mistrusted on sight.
default_rep: -25
allies: []
enemies: [thornwall_guards]
# Optional — only guard factions set this:
# holding_cell_room: 5106
```

Stored eagerly at startup. The `faction_id` is the stable key (used internally). The `display_name` and `description` are for human-readable output. The `default_rep` is the starting point for new players (what `GetRep` returns when no row exists). The `allies` and `enemies` arrays list related factionIds — declarative only (rep changes do NOT auto-propagate through them).

**`holding_cell_room` (optional, int):** Room ID of the faction's holding
cell for arrested players. Omit (or set to 0) for non-guard factions —
`ExecuteArrest` treats 0 as "no jail" and returns false. Only guard
factions that actually jail players set this field. Current values:
`thornwall_guards` → 5105, `stillwater_guards` → 5106.

**`release_room` (optional, int):** Room ID where a freed prisoner of
this faction is released after serving a sentence or paying a fine.
Omit (or set to 0) for factions without a specific release destination;
`ResolveDetention` falls back to the hardcoded `barracksRoomId` (473)
when 0. Current values: `thornwall_guards` → 473,
`stillwater_guards` → 4110. Read seam: `justice.releaseRoomFn` in
`internal/justice/justice.go`.

Boot-time validation (`factions.ValidateHoldingCells(roomExistsFn)`) is
called from main.go after `rooms.LoadDataFiles()`. It iterates every
loaded definition and panics if `holding_cell_room != 0` and the room
doesn't exist, OR if `release_room != 0` and the room doesn't exist.
The DI callback (`func(int) bool`) breaks the factions ← rooms import
cycle. Read seam: `justice.cellRoomFn` in `internal/justice/justice.go`.

### Runtime Rep State (gitignored, not committed)
```yaml
# _datafiles/world/dogmud/factions.rep/{slug}.yaml
faction_id: warren
players:
  17:
    rep: 30
    last_updated_round: 1843201
  92:
    rep: -75
    last_updated_round: 1846020
```

Stored lazily on first mutation. The per-player `rep` scalar is clamped to [-100, +100]. The `last_updated_round` is updated on every Set/Bump for audit and decay calculations if added later.

### Go Types

**Definition** (immutable after load):
```go
type Definition struct {
    FactionId       string   `yaml:"faction_id"`
    DisplayName     string   `yaml:"display_name"`
    Description     string   `yaml:"description"`
    DefaultRep      int      `yaml:"default_rep"`
    Allies          []string `yaml:"allies"`
    Enemies         []string `yaml:"enemies"`
    HoldingCellRoom int      `yaml:"holding_cell_room"` // 0 = no jail
}
```

**RepEntry** (per-player row):
```go
type RepEntry struct {
    Rep              int    `yaml:"rep"`
    LastUpdatedRound uint64 `yaml:"last_updated_round"`
}
```

**FactionRep** (full per-player table for one faction):
```go
type FactionRep struct {
    FactionId string            `yaml:"faction_id"`
    Players   map[int]*RepEntry `yaml:"players"`
}
```

**Score semantics:**
- **[-100, +100]**: Signed scalar, where -100 is maximum hostility and +100 is maximum friendliness.
- **Clamping**: All Set and Bump operations clamp to this range.
- **Default**: Players without custom rows default to the faction definition's `default_rep`.
- **No decay**: Faction rep is institutional memory; unlike individual NPC opinions, it does not drift back to neutral over time.

## Integration Notes

### Callers
- **Combat hookup** (`internal/hooks/MobDeath_FactionRep.go`): On mob death, bumps the killer's rep with each of the dead mob's defined factions by `Balance.FactionMemberKillRep` (default -10). Party propagation applies the bump to all party members in the same room.
- **Quest engine** (`internal/questengine/actions.go`): `bump_rep` action calls the bridge's `BumpRep` to mutate quest-granted rep gains.
- **Admin command** (`internal/usercommands/admin.faction.go`): `faction list/show/set/bump/reset` subcommands expose the API to administrators.
- **NPC peace check** (`internal/mobcommands/lookfortrouble.go`, `internal/behaviortree/actions_party.go`): Calls `IsPeacefulToward` to gate whether a mob initiates combat on sight.
- **Dialogue and quest gating**: Future chunks will add `questFlagRequired` / `requires` gates that call `TierFor` to conditionally show or grant options.

### Dependencies
- **opinions**: Calls `opinions.TierOf(rep)` to map raw rep to tier enums. Single source of truth for tier banding.
- **configs**: Reads `FactionMemberKillRep` from `Balance` config for combat magnitude.
- **mobs**: Calls `GetMobSpec` to validate faction membership and reads `mob.Groups` to filter defined factions.
- **util**: Uses `GetRoundCount()` for `last_updated_round` stamping and `ConvertForFilename` for path generation.
- **mudlog**: Logs warnings on disk I/O errors and unknown faction references.

## Testing Notes

### Test Files
- **registry_test.go**: ~100 lines covering definition loading, validation of ally/enemy references, and idempotent re-load.
- **factions_test.go**: ~200 lines covering GetRep (default fallback), SetRep (clamp), BumpRep (read-add-clamp), TierFor (tier banding), FactionsForMob (subset filtering), IsPeacefulToward (tier threshold).
- **persistence_test.go**: ~150 lines covering disk I/O, cache lifecycle, lazy initialization, and concurrent save serialization.
- **test_main_test.go**: TestMain that sets `DOGMUD_FACTIONS_DIR_OVERRIDE` and `DOGMUD_FACTIONS_REP_DIR_OVERRIDE` to temp dirs so tests don't touch the real factions or factions.rep directories.

### Test Seam Pattern
Tests inject `roundForTest` to freeze the round count, avoiding a hard dependency on `util.GetRoundCount()`. After each test, the seam is cleared so production code sees nil and uses the real implementation. Definitions are loaded fresh for each test via `LoadAllDefinitions()` into an isolated temp dir.

### Key Test Cases
- `GetRep` returns `default_rep` when no row exists; doesn't write a row to disk.
- `SetRep` clamps to [-100, +100], updates the timestamp, and persists.
- `BumpRep` reads current (or default), adds delta, clamps, and persists.
- `BumpRep` on unknown faction is a no-op (warned via mudlog, not panicked).
- `TierFor` correctly maps scores across tier boundaries.
- `FactionsForMob` returns only groups that have loaded definitions.
- `IsPeacefulToward` returns true only when at least one faction is `TierWarm` or higher.
- Definition load validates ally/enemy references and panics on unknown faction.
- Concurrent `BumpRep` calls to the same faction converge via `saveMu` serialization.
