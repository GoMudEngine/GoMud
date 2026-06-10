# engine Package Context

## Overview
`engine` concentrates all direct engine-world calls (room/zone/biome reads,
mutator application, game clock, ambient emotes, cache codec) and implements
`crawler.WorldReader`. Together with the root `weather` package — which imports
`internal/*` for plugin infrastructure — these are the only packages that touch
the engine; `sim`, `crawler`, and `content` stay pure. That split keeps the
module portable across GoMud and DOGMud.

## Key Components
### Core Files
- **worldreader.go**: `WorldReader` implements `crawler.WorldReader` over
  `internal/rooms` (`GetAllZoneNames`, `GetZoneBiome`, `GetAllZoneRoomsIds`,
  `LoadRoom`). `NewWorldReader()` returns it as the interface. `isOutdoorBiome`
  derives a room's outdoor flag from its biome id (GoMud has no explicit
  indoor/outdoor flag), using the `indoorBiomes` heuristic set. Also used by
  `EmitAmbient` for the indoor-detection heuristic.
- **cache.go**: `CacheIdentifier` (the plugin-storage key) and `DecodeCache`,
  a pure, version-checked decoder that returns ok=false for absent/empty/
  unparseable/stale data so the caller knows to rebuild.
- **state.go**: `StateIdentifier` (plugin-storage key "simstate"),
  `StateVersion` (versioned envelope), `EncodeState`, `DecodeState`. Version
  mismatch → ok=false → caller re-seeds (discard, don't migrate).
- **apply.go**: `mutatorSet` interface (seam for test fakes — the real
  `MutatorList.Add` consults the global spec registry, so unit tests inject a
  fake at this interface). `MutatorIdFor` maps a weather type to its
  `weather-*` mutator id ("" for calm). `applyChange` (Has-guard because
  `MutatorList.Add` appends duplicates when the mutator is already live).
  `reconcileZone` (forces exact match between live weather-* mutators and the
  target type). `Apply(diff)` walks a `StateDiff` and calls `applyChange` for
  each zone — an exported low-level primitive with **no production caller**
  today (M4's per-room refinement is its likely consumer).
  **`Reconcile(weather map)`** forces every zone in the map to match
  the resolved weather — used at boot, after state restore, and after a graph
  rebuild. `Reconcile` is the single path by which module state reaches engine
  mutators (tick, commands, exports, post-rebuild): because specs carry
  `decayrate`, a bare diff-apply would let engine-side decay drift persist.
  `StripBuffs()` clears buff id lists on all loaded `weather-*` specs — boot-time
  only, no restore path. `warnedMutators` warn-once map (safe on single goroutine).
- **clock.go**: `TickPeriod(hours int) string` — renders game-hour count as a
  `gametime.AddPeriod` period string; values < 1 clamp to 1. `NextTickRound`
  returns the round number one period from now. `CurrentRound` exposes
  `util.GetRoundCount`.
- **emotes.go**: `EmitAmbient(weather, tables, roll)` — sends one ambient line
  into each occupied room whose zone has non-calm weather. Room biome drives
  table variant; `isOutdoorBiome` determines the indoor/outdoor section.
  `roll` is the presentation RNG (pass `util.Rand`) — NEVER the sim RNG.
  Returns lines sent.

## Dependencies
- `internal/rooms`, `internal/mutators`, `internal/gametime`, `internal/util`,
  `internal/mudlog` (engine).
- `github.com/GoMudEngine/GoMud/modules/weather/{sim,crawler,content}` — pure types.

## Consumers
- The module root (`weather.go`) uses `NewWorldReader()`, `DecodeCache`/`CacheIdentifier`.
- The module root (`weather_tick.go`) uses `EncodeState`/`DecodeState`,
  `TickPeriod`/`NextTickRound`/`CurrentRound`, `EmitAmbient`, `StripBuffs`.
- The module root (`weather_commands.go`, `weather_api.go`) calls
  `Reconcile`/`CurrentRound` after any state mutation.

## Testing
- `cache_test.go` covers `DecodeCache` (pure).
- `worldreader_test.go` covers `isOutdoorBiome`.
- `state_test.go` covers `EncodeState`/`DecodeState` (pure, in-checkout).
- `apply_test.go` covers `MutatorIdFor`, `applyChange`, `reconcileZone` via
  a fake `mutatorSet` (in-checkout unit test).
- `clock_test.go` covers `TickPeriod` (pure, in-checkout).
- `WorldReader`, `Reconcile`, and `EmitAmbient` are thin engine-glue
  verified by the in-checkout build and boot smoke test. `Apply` itself is
  exercised only through `applyChange`'s unit tests (no production caller).
- These tests compile only inside a GoMud checkout (engine imports).

## Build note
This package compiles only inside a checkout (it imports `internal/*`). In the
standalone repo, test the pure core with `go test ./sim/... ./crawler/... ./content/...`.
