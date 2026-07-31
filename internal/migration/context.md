# Migration Context

## Purpose

`internal/migration` upgrades on-disk data files when the server binary is
newer than the version recorded in config. It runs once at start-up, **before**
the main data-load block, and it rewrites YAML in place — player saves, room
files, and config.

Everything is guarded by a whole-datafiles backup with automatic restore on
failure, because these migrations edit files the game cannot regenerate.

## Files

- **migration.go** — `Run` (entry point) and `doAllMigrations` (the ordered
  version ladder).
- **backup.go** — `datafilesBackup`, `copyDir`, `copyFile`.
- **classify.go** — `PlayerSignals` and `ClassifyPlayer`.
- **grant.go** — `SeedForCluster`.
- **0.9.1.go … 0.15.0.go** — one file per version step, named for the version
  it upgrades *to*.

## Control flow

```go
func Run(lastConfigVersion, serverVersion version.Version) error
```

1. If `lastConfigVersion == serverVersion`, return immediately — no backup, no
   work.
2. Back up the entire datafiles tree; `defer os.RemoveAll(backupFolder)`.
3. Run `doAllMigrations(lastConfigVersion)`.
4. **On error, copy the backup back over datafiles** and return the error.
5. On success, write `Server.CurrentVersion = serverVersion` to config.

`doAllMigrations` is a flat list of `if lastConfigVersion.IsOlderThan(...)`
blocks in ascending order. A server three versions behind runs all three steps
in sequence.

## The version ladder

| To | Migration | What it does |
|----|-----------|--------------|
| 0.9.1  | `migrate_RoomZoneConfig` | introduces per-zone config files |
| 0.10.0 | `migrate_UserStatsRename` | renames stat keys in user saves |
| 0.11.0 | `migrate_RollCharacterStats` | rerolls stats onto the 100-baseline model |
| 0.12.0 | `migrate_RaceToSpecies` | `race:` → `species:` |
| 0.13.0 | `migrate_SeedWarrenRepFromQuestToken` | seeds faction rep from a legacy quest token |
| 0.14.0 | `migrate_ReclassifyPlayerMutations` | wipes retired mutation 41 and reclassifies every save onto the cluster graph |
| 0.15.0 | `migrate_BackfillCoords` | crawls exit deltas to backfill authored x/y/z/plane on every non-instance room |

The two newest take a `dryRun bool` so they can be exercised without writing.

## Mutation reclassification (0.14.0)

```go
type PlayerSignals struct { /* play-pattern counters read from the save */ }
func ClassifyPlayer(s PlayerSignals) string        // → cluster name
func SeedForCluster(cluster string) map[string]int // → mutation grants
```

`extractSignals` reads raw play signals out of the YAML map, `ClassifyPlayer`
picks the cluster that best fits, and `SeedForCluster` returns the mutations to
grant.

**`SeedForCluster("admin")` grants 11 keystone mutations and freezes drift.**
That is why an admin character is useless for evaluating mutation pacing — it
was seeded, not grown. Any playtest of drift or apex pacing must use a
non-admin character.

## Coordinate backfill (0.15.0)

Walks each rooms directory, groups rooms into connected components by spatial
exits, and assigns coordinates by crawling deltas from an arbitrary origin per
component (`crawlComponent`). Zones marked `non_cartesian` are read from
`loadNonCartesianZones` and skipped for collision purposes.
`writeCoordsPreservingOrder` re-emits the YAML with the new fields inserted
without reordering the rest of the file — important, because these are
hand-authored files people read in diffs.

`countCollisions` reports how many rooms landed on an occupied cell; a non-zero
count means the world was not Cartesian-consistent at migration time.

## Gotchas

- **Migrations run before data loading.** Nothing is in memory yet — no rooms,
  no mobs, no factions. The 0.13.0 step has to call
  `factions.LoadAllDefinitions()` itself for exactly this reason, and any new
  migration that needs loaded data must do the same.
- **The backup is deleted by `defer` even on the error path** — but only after
  the restore has already run, so ordering is correct. Do not add an early
  return between the restore and the deferred cleanup.
- **A partially-applied migration is restored wholesale, not rolled back
  step-by-step.** Any migration that writes outside the datafiles tree escapes
  the safety net.
- **Version comparison is `IsOlderThan`, not equality.** Steps are cumulative;
  never write a migration that assumes the previous one just ran in this
  process.
- **`Run` only writes `Server.CurrentVersion` on full success.** A failed
  migration leaves the version untouched, so the next boot retries from the
  same point.

## Dependencies

`configs`, `version`, `factions` (0.13.0 only), plus direct YAML and filesystem
access. Deliberately minimal — this code must work before the engine is up.

## Consumers

`main.go` only, at start-up.
