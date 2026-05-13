# Opinions System Context

## Overview

The `internal/opinions` package maintains NPC dispositions toward individual players on a signed [-100, +100] scale. Each mob template has a default disposition (neutral starting point), and this package caches per-player opinion deltas, applies exponential decay over time, and persists them to disk. Combat, dialogue, quests, and admin commands all call into opinions to read or mutate NPC mood — the package is the persistent record of "how does this NPC feel about this player right now."

## Key Components

### Core Files
- **types.go**: Data structures (`Opinion`, `MobOpinions`, `Tier` constants and helpers)
- **opinions.go**: Public API (`Get`, `Set`, `Bump`, `TierFor`, `AllRowsForUser`); lazy cache initialization
- **persistence.go**: Disk I/O (`loadFromDisk`, `saveToDisk`, `SaveAllOpinions`); cache management; test seams
- **decay.go**: Exponential decay math (`decayedScore`, `pull`)
- **opinions_test.go**: Comprehensive unit tests for all public functions and cache behavior
- **persistence_test.go**: Tests for disk I/O and cache management
- **test_main_test.go**: TestMain setup with temp-dir override for tests

## Key Functions

### Reading Opinion State
- **Get(mobId int, userId int) int**: Returns the decay-adjusted score for this user toward this mob. Reads the default disposition if no row exists. Does not mutate anything; lazy cache priming may occur on first access but no file is created.

- **TierFor(mobId int, userId int) Tier**: Sugar for `TierOf(Get(mobId, userId))` — returns one of five tiers instead of a raw score.

- **TierOf(score int) Tier**: Maps a raw score to a tier (`TierHostile` ≤ -50, `TierCold` -49 to -15, `TierNeutral` -14 to +14, `TierWarm` +15 to +49, `TierFriendly` ≥ +50).

### Mutating Opinion State
- **Set(mobId int, userId int, score int)**: Assigns an absolute score (clamped to [-100, +100]), stamps the current round, and persists synchronously.

- **Bump(mobId int, userId int, delta int)**: Adds delta to the decay-adjusted score (starting from default if no row exists), clamps, re-stamps the round, and persists. The everyday mutator — combat, dialogue, and quest hooks all call this.

### Admin/Reporting
- **AllRowsForUser(userId int) []Row**: Returns every opinion row mentioning the given userId across all cached mobs. Decay is applied to show the same value NPC behavior would observe. Walks only the in-memory cache, no disk I/O.

### Decay Helper
- **decayedScore(score, def int, anchorRound, nowRound, halfLifeRounds uint64) int**: Applies integer-step exponential decay from the anchor round to now. Each half-life elapses, the score takes one step toward the default disposition. Returns the score unchanged if halfLife is 0 (decay disabled) or if now ≤ anchor.

## Global State

### Cache and Mutex
- **opinionCache**: `map[int]*MobOpinions` keyed by mobId. Holds every loaded mob opinion table in memory.
- **opinionCacheMu**: `sync.RWMutex` protecting opinionCache and nameByMobId.
- **nameByMobId**: `map[int]string` — remembers the namesimple used at file write time so persistence can reconstruct the path without re-reading the mob template.
- **saveMu**: `sync.Mutex` serializing file I/O to prevent Windows ERROR_SHARING_VIOLATION on overlapping writes.

### Test Seams
- **defaultProviderForTest**: Injection hook for resolveTemplate — lets tests provide a fake mob-template lookup.
- **roundForTest**: Injection hook for currentRound — lets tests freeze or advance the round count without reading util.GetRoundCount().
- **halfLifeForTest**: Injection hook for currentHalfLife — lets tests override DecayHalfLifeRounds without standing up the full configs stack.

## Data Structure Design

### Per-NPC Opinion File
```yaml
mob_id: 101
default_disposition: 0
opinions:
  17:                          # userId 17
    score: -30
    last_updated_round: 2075029
  42:
    score: 65
    last_updated_round: 2000000
```

Stored at `_datafiles/world/dogmud/opinions/{mobId}-{namesimple}.yaml`. The `default_disposition` is snapshotted from the mob template at first write and serves as the asymptote for decay; if a mob's template default changes, existing files must be manually updated or deleted to pick up the new default.

### Score Semantics
- **[-100, +100]**: Signed scalar, where -100 is maximum hostility and +100 is maximum friendliness.
- **Clamping**: All Set and Bump operations clamp to this range; overflow wraps to the boundary.
- **Default**: Mobs without custom rows default to their mob template's `default_disposition` (typically 0 = neutral).

### Opinion Type
```go
type Opinion struct {
	Score            int    `yaml:"score"`
	LastUpdatedRound uint64 `yaml:"last_updated_round"`
}
```

### Tier Constants
```go
TierHostile  Tier = iota  // <= -50
TierCold                  // -49 .. -15
TierNeutral              // -14 .. +14
TierWarm                 // +15 .. +49
TierFriendly             // >= +50
```

## Integration Notes

### Callers
- **attack.go**: Checks first-aggression via `TierFor` and applies `Bump` on successful hit (combat honor/dishonor tracking).
- **dialogue**: NPC dialogue mood gates use `Get` or `TierFor` to show different responses based on disposition.
- **admin.opinion.go**: Admin command to view all opinions for a user via `AllRowsForUser`.
- **target.go**: Target-switch logic applies `Bump` on attack context changes.
- **factions.go**: Companion chunk that reads opinions for NPC allegiance decisions.

### Dependencies
- **configs**: Reads `DispositionDecayHalfLifeRounds` from the balance config.
- **mobs**: Calls `GetMobSpec` to resolve mob templates and extract `default_disposition`.
- **util**: Uses `GetRoundCount()` for time-based decay and `ConvertForFilename` for path generation.
- **mudlog**: Logs warnings on disk I/O errors.

## Testing Notes

### Test Files
- **opinions_test.go**: ~230 lines covering Get, Set, Bump, TierFor, AllRowsForUser, decay edge cases, and clamping behavior.
- **persistence_test.go**: ~150 lines covering disk I/O, cache lifecycle, concurrent saves, and file marshaling.
- **test_main_test.go**: TestMain that sets `DOGMUD_OPINIONS_DIR_OVERRIDE` to a temp dir so tests don't touch the real opinions directory.

### Test Seam Pattern
Tests inject fake implementations via global function pointers (`defaultProviderForTest`, `roundForTest`, `halfLifeForTest`) without building mobs, configs, or util stacks. After each test, seams are cleared so production code sees nil and uses the real implementations.

### Isolated Decay Testing
Decay tests freeze the round count and half-life via seams, then verify `decayedScore` pulls the score toward default in integer steps without overshooting.
