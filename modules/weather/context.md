# weather Package Context (module root)

## Overview
The root `weather` package is the GoMud plugin entry point. It wires the full
module lifecycle: plugin registration, config, geography-graph management,
weather-simulation tick, ambient emotes, admin/player commands, and the exported
API. It imports `internal/*` for plugin infrastructure
(plugins/events/users/mudlog/util/rooms); engine-world calls live in `engine/`;
pure algorithms live in `sim`/`crawler`; data-file parsing lives in `content/`.
All fields of `weatherModule` are touched only from the single game-loop
goroutine — no synchronization needed.

## Key Components
- **weather.go**: the `files` embed.FS (`//go:embed files/*` — the config
  overlay plus `datafiles/` mutator specs and emote tables; the engine loads
  `mutators/*` from it via the plugin registry, `content` loaders read the
  rest). `weatherModule` struct (plug, cfg, graph, started, simReady,
  simCfg, climate, tables, state, nextTick, nextEmote). `init()` → `plugins.New`
  + `AttachFileSystem` + `SetOnLoad`, then registers the `weather` command as a
  **player** command (not admin-only; admin subcommands are gated in-handler) and
  the exports. Command/export registration MUST happen in `init()`, not `onLoad`:
  `plugins.Load()` harvests the plugin's `userCommands` map into the engine
  registry BEFORE invoking `onLoad`, so anything registered in `onLoad` is lost.
  Behavior is gated on `cfg.Enabled`/`simReady` in-handler instead. `onLoad`:
  loads config, then (when enabled) registers `SetOnSave` and a `NewRound`
  listener. `onNewRound`: one-time startup (loadOrBuildGraph + startSim), the
  jittered ambient-emote pass, and the coarse weather tick. `loadOrBuildGraph`/
  `rebuildGraph`: cache-or-crawl; `rebuildGraph` also calls `startSim` and
  `engine.Reconcile`. `sendLine` is the SOLE `user.SendText` call site.
- **weather_tick.go**: `startSim` (idempotent; graceful degradation — logs once
  and stays idle when no graph exists). `loadContent` (climate overrides + emote
  tables from the embedded FS, both fail-soft). `loadOrInitState` (restore from
  `engine.DecodeState`, or `sim.NewState`/`sim.DeriveSeed` on a fresh start).
  `tick` (Step → `engine.Reconcile`; Reconcile rather than bare Apply so
  engine-side `decayrate` drift self-corrects within one tick). `persistState`
  (cheap; called per-tick, from onSave, and from every command/export mutation path). `onSave` (plugins.Save hook).
  `scheduleEmote` (±25% jitter so ambience doesn't metronome).
- **weather_commands.go**: bare `weather` shows local conditions (player view;
  includes the dominant front via `sim.Covering`). Subcommands `zones`, `fronts`,
  `spawn <type> <zone> [intensity]`, `clear [zone]`, `graph [zone]`, `rebuild`,
  `status` are admin/mod-gated via `HasRolePermission("weather", true)`. `spawn`
  and `clear` call `sim.ForceSpawn`/`sim.ClearZones` then `engine.Reconcile` +
  `persistState`.
- **weather_api.go**: `registerExports` exposes `GetWeather`, `GetFronts`,
  `SpawnFront` via `plugin.ExportFunction`. All three guard `simReady` so callers
  during boot get empty-but-valid answers. The MainWorker-goroutine guarantee
  applies to mutating exports (same as commands). `SpawnFront` calls
  `engine.Reconcile` + `persistState`.
- **weather_config.go**: `Config` struct (Enabled, IncludeSecretExits,
  RebuildGraphOnBoot, Seed, TickEveryGameHours, MaxActiveFronts, SpawnRateScale,
  EmoteMode, EmoteEveryRounds, BuffsEnabled, Persist). Keys are flat because
  plugin config lookup reads flattened scalar leaves. `buildConfig(getter)`
  (testable, applies defaults and sanity clamps). `simConfig()` maps module
  config onto `sim.Config`. `loadConfig(*plugins.Plugin)`.

## Dependencies
- `internal/plugins, events, users, mudlog, util, rooms` (engine, plugin infra).
- `modules/weather/{sim,crawler,engine,content}`.

## Threading
GoMud runs a single game-loop goroutine (MainWorker) for both NewRound listeners
and command handlers, so `weatherModule` fields need no synchronization.
Exported functions are invoked on the same goroutine.

## Build/Testing
Compiles only inside a GoMud checkout (imports `internal/*`). `weather_config_test.go`
covers `buildConfig`; the registration/command/tick/export paths are verified by
the in-checkout build and a boot smoke test (first-round build → state persist →
reload → tick).

## DOGMud backport
Only `user.SendText` differs (DOGMud takes a message category). It is isolated in
`sendLine` — a one-line change to backport. See CONTRIBUTING.md.
