# content Package Context

## Overview
`content` is the pure data-loading layer for the weather module. It parses two
kinds of module data files from an `fs.FS` (typically the module's embedded
`files/` tree): climate profiles (YAML → `sim.Climate` merged over
`sim.DefaultClimate`) and ambient emote tables (YAML → `Tables` + `Pick`). No
engine imports — purity enforced by `arch_test.go`. The only non-stdlib
dependency is `gopkg.in/yaml.v2`, which the GoMud engine itself uses.

## Key Components
### Core Files
- **climate.go**: `ParseClimate` (one file → biome id + `sim.ClimateProfile`);
  `LoadClimate` (walks a directory, returns `DefaultClimate` overlaid with every
  `*.yaml` found; a missing directory is not an error — pure defaults are
  returned; the first malformed file aborts with an error).
- **emotes.go**: `Table` (per-weather-type ambient lines keyed by biome, split
  outdoor/indoor); `Tables` (weather type → `Table`); `ParseEmoteTable`; `LoadEmotes`
  (walks a directory, empty tables for a missing directory); `(Tables).Pick`
  (biome → "default" fallback; indoor NEVER falls back to outdoor — silence beats
  wrong prose; out-of-range roll result clamped to index 0).
- **arch_test.go**: purity guardrail — fails if any file imports a
  `GoMudEngine/GoMud/internal` path.
- **moduledata_test.go**: validates the SHIPPED YAML files under
  `files/datafiles/`. For emote tables: parseable, `filename == weather+".yaml"`,
  at least one outdoor-default and one indoor-default line. For mutator specs:
  parseable, `mutatorid` is `weather-` namespaced, filename matches
  `fileNameFor(mutatorid)` (mirrors `util.ConvertForFilename`), `respawnrate`
  forbidden (would fight the orchestrator), `decayintoid` forbidden (upstream
  `MutatorList.Remove` resets `SpawnedRound` and runs `Update` whose decay
  branch has no liveness guard — the decay target is instantly resurrected),
  `decayrate` required (self-heal safety net).
- **doc.go**: package-level comment.

### Key Types
```go
type Table struct {
    Weather string              // weather type this table covers
    Outdoor map[string][]string // biome -> lines (outdoor section)
    Indoor  map[string][]string // biome -> lines (indoor section)
}
type Tables map[sim.WeatherType]Table
```

## Core Functions
- `ParseClimate([]byte) (string, sim.ClimateProfile, error)` — parse one climate
  YAML into its biome id and profile.
- `LoadClimate(fs.FS, dir string) (sim.Climate, error)` — `sim.DefaultClimate()`
  overlaid with every `*.yaml` under `dir`.
- `ParseEmoteTable([]byte) (Table, error)` — parse one emote table YAML.
- `LoadEmotes(fs.FS, dir string) (Tables, error)` — all emote tables under `dir`.
- `(Tables).Pick(weather sim.WeatherType, biome string, indoor bool, roll func(int) int) string`
  — select one ambient line. `roll(n)` must return `[0,n)`; pass the engine's
  `util.Rand` (or a stub in tests) — NEVER the sim RNG, which must stay isolated
  from presentation randomness.

## Dependencies
- `github.com/GoMudEngine/GoMud/modules/weather/sim` (types only).
- `gopkg.in/yaml.v2` — the engine's own dependency; the standalone `go.mod`
  carries it for tests; `go.mod`/`go.sum` never travel to checkouts.
- Standard library (`io/fs`, `path`, `strings`, `fmt`). No engine imports.

## Consumers
- Module root (`weather_tick.go`): calls `LoadClimate` and `LoadEmotes` at
  startup; results are stored on `weatherModule` and re-used each tick.
- `engine.EmitAmbient`: receives a `content.Tables` and calls `Pick` with the
  engine's `util.Rand` as the roll function.

## Testing
- `climate_test.go`: `ParseClimate`, reject-missing-biome, `LoadClimate` merges
  override over defaults, missing dir returns pure defaults.
- `emotes_test.go`: `ParseEmoteTable`, `LoadEmotes` missing dir, `Pick` biome
  selection, indoor-never-falls-back-to-outdoor, roll forwarding, out-of-range
  clamp.
- `moduledata_test.go`: validates shipped YAML (see Key Components above).
- `arch_test.go`: engine-import purity guardrail.

All tests run standalone: `go test ./content/...` (no checkout required).
