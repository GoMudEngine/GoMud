# Weather Module Backport — Design Spec

**Date:** 2026-06-10
**Status:** Approved pending user review
**Source module:** `C:/Users/Calabe Davis/workspace/weather-module` (standalone repo, M3 complete)
**Source spec:** `weather-module/docs/superpowers/specs/2026-06-08-weather-module-design.md`

## Goal

Backport the GoMud weather module to DOGMud as a **full DOGMud-native,
presentation-only** weather system: named fronts travel zone-to-zone across
the world's exit graph, biomes shape what weather forms and how it moves, and
players experience it through room name tags, description lines, alert
banners, light changes, and ambient emotes — with a properly muted,
intensity-aware indoor experience.

Ships via normal SOP: build clean, tests pass, merge to master locally,
smoke-test, push to prod at end of day.

## Decisions (settled during brainstorm)

| Question | Decision |
|---|---|
| Adaptation depth | Full DOGMud-native: climate profiles for all 17 DOGMud biomes, DOGMud-voiced emotes, indoor handling fixed |
| Gameplay mechanics | **Presentation only.** No buffs this pass. Severe-weather `lightmod` is the only mechanical side effect (inherent to light) |
| Data loading | Module Go code vendors into `modules/weather/`; all data files live in DOGMud's existing `_datafiles` tree. Engine changes limited to two generic data-driven flags (see indoor model) |
| Indoor model | Indoor rooms never show weather mutators (render-time filter via `BiomeInfo.Indoor` + `MutatorSpec.OutdoorOnly`); indoor emotes are intensity-gated (mild weather = silence, strong weather = muted sensory lines) |
| Rollout | Normal pre-push SOP, prod at end of day |

## Architecture

### Code layout

The five Go packages vendor into `modules/weather/`, mirroring how the
playtest module was adopted:

```
modules/weather/
  sim/        pure simulation core (byte-identical from source repo, with tests)
  crawler/    pure geography crawler (byte-identical, with tests)
  content/    pure data-file layer (byte-identical, with tests)
  engine/     engine adapter — the ONLY package touching internal/* (DOGMud edits here)
  weather*.go module root: plugin lifecycle, tick loop, commands, exports (DOGMud edits here)
```

Registered via `go generate ./...` → `modules/all-modules.go`. The source
repo's architecture tests (pure packages must not import `internal/*`) come
along unchanged.

Drop `go.mod`/`go.sum`/`docs/`/`scripts/` — in-checkout modules build as part
of the engine.

### Data layout

All player-facing data moves into DOGMud's `_datafiles` tree, loaded by the
engine's existing loaders:

| Source repo location | DOGMud location |
|---|---|
| `files/datafiles/mutators/weather_*.yaml` (8 specs) | `<DataFiles>/mutators/` (the flat dir `mutators.LoadDataFiles()` already reads) |
| `files/datafiles/emotes/*.yaml` | `_datafiles/world/dogmud/weather/emotes/` (read by the module's `content` package) |
| `files/datafiles/climate/*.yaml` (new, per DOGMud biome) | `_datafiles/world/dogmud/weather/climate/` |
| `files/data-overlays/config.yaml` (`Modules.weather.*`) | `Modules.weather.*` block in `_datafiles/config.yaml`, like `Modules.playtest.*` |

The `content` package's reader gets pointed at the on-disk paths instead of
the embedded module FS. Geography cache + persisted simulation state go
wherever the fork's plugin-storage API puts them; if that API drifted from
upstream, fall back to flat YAML under `_datafiles/world/dogmud/weather/`
(same pattern as economy snapshots).

## Engine adaptation (compile-time fixes)

- **`SendText(category, text)`** — DOGMud added a category parameter. All
  player-facing output flows through one helper (`sendLine`); pick the
  appropriate category there.
- **Config access** — read `Modules.weather.*` via the fork's module-config
  API (as the playtest module does), not upstream's module-overlay mechanism.
- **Plugin storage / exports / events / gametime** — verify at compile time;
  the module's only engine-coupled code is `engine/` + root, so drift
  surfaces as compile errors there. Fix in place.
- **Mutator lifecycle invariants still apply:** weather specs must never set
  `respawnrate` or `decayintoid`; `decayrate` stays as the self-heal net if
  the module is ever disabled mid-storm. Verify DOGMud's `MutatorList.Remove`
  behaves like upstream's (the decayintoid-resurrection gotcha).

## The indoor model (refinement over the standalone module)

Two changes, both in module code (no engine changes):

### 1. Render-time indoor filtering (REVISED 2026-06-10 during planning)

The standalone module applies weather as a *zone* mutator, so indoor rooms
render `(storm-wracked)` tags and "rain lashes the cobblestones" description
lines. The originally-approved fix (apply mutators per room, outdoor rooms
only) turned out to have two implementation problems discovered during
planning: room-level mutators persist into `rooms.instances/` saves (the
stale-instance-shadowing trap; prod doesn't deploy those saves), and per-room
reconciliation means `LoadRoom`-ing every room in covered zones each tick,
fighting the room-unload/perf system.

**Revised mechanism, same player-visible behavior:** weather stays a
zone-level mutator (exactly like the source module — cheap, no room loading,
nothing persisted per room), and indoor rooms are filtered at render time via
two small, generic, data-driven engine additions:

- `BiomeInfo.Indoor` (`indoor: true` in biome YAML) — set on `cave`,
  `dungeon`, `house`, `fort`, `spiderweb`.
- `MutatorSpec.OutdoorOnly` (`outdooronly: true` in mutator YAML) — set on
  all 8 weather specs; reusable by existing mutators (`desert_sun`,
  `forest_mist`) later.
- `Room.ActiveMutators` skips outdoor-only mutators when the room's biome is
  indoor — the single merge point all render paths (name/description/alert/
  lightmod via GetDetails/GetVisibility) already flow through.

Indoor rooms get no name tag, no weather description line, no alert, no
lightmod. Their entire weather experience is the emote channel below. The
"zero engine changes" decision is amended to "two generic data-driven engine
flags + one filter in ActiveMutators."

### 2. Intensity-gated indoor emotes

Indoor emote pools are keyed by **felt-intensity band**, not just weather
type:

```yaml
# emotes/rain.yaml (shape sketch)
default:
  outdoor:
    - "Rain patters across the stones, beading on every surface."
  indoor:
    mild: []          # light rain doesn't register through walls
    strong:
      - "The sound of rain on the roof settles into a rhythmic, soothing patter."
```

The emote scheduler gates on the zone's felt intensity: below the threshold,
indoor rooms are silent; above it, muted sensory lines play. The heaviest
types (storm, blizzard) should always have `strong` indoor lines — they are
audible/feelable anywhere. Band threshold is a module constant tuned during
smoke testing (start: `strong` ≥ 0.5 felt intensity); the YAML schema keeps
it data-driven per line pool after that.

Outdoor pools keep the existing single-pool shape (intensity already shapes
*which weather type* a zone has; outdoor lines don't need banding in v1).
Indoor never falls back to outdoor (inherited rule — silence beats "rain
falls around you" inside a tavern).

## Content plan — all 17 DOGMud biomes

Climate profiles (`weather/climate/<biome>.yaml`) authored for every DOGMud
biome. Directional sketch (weights tuned during implementation):

| Biome | Births | Terrain influence |
|---|---|---|
| `water` | rain, storm, fog | strong moisture feed — open water powers storms |
| `shore` | rain, fog, overcast | moderate feed |
| `cliffs` | storm, fog, overcast | moderate; exposed |
| `desert` | dust, heatwave, clear | strong moisture sap |
| `snow` | snow, blizzard | cold; sustains winter systems |
| `mountains` | snow, storm | strong drag — ranges bleed fronts dry |
| `swamp` | fog, rain, overcast | humid; sustains fog |
| `forest` | rain, overcast, fog | mild |
| `farmland` | rain, overcast, clear | mild |
| `land` | clear, overcast, rain | neutral baseline |
| `road` | clear, overcast | neutral |
| `city` | overcast, rain | neutral — streets get full weather |
| `cave`, `dungeon`, `house`, `fort`, `spiderweb` | — (spawnWeight 0) | indoor: excluded from mutator application; intensity-gated indoor emotes only |

Note: indoor biomes having `spawnWeight: 0` means fronts don't *form* there,
but a zone whose outdoor rooms sit under a front still passes weather to its
indoor rooms via the emote channel.

- **Emote tables** rewritten in DOGMud's voice for each weather type ×
  biome where it matters (forest storm lines ≠ city storm lines), 80-char
  wrapped, no hard numbers, first-person-free narrator tone.
- **Mutator specs** (8): name tag, description line, alert + `lightmod` for
  severe types only (storm, blizzard, dust). **No buff ids** in any spec, and
  `BuffsEnabled: false` in config as belt-and-suspenders.
- **Colors** remapped to pattern names that exist in DOGMud's
  `color-patterns.yaml` (verify `gray`, `blue`, `mute-dblue`, `frost`,
  `brown`, `embers`; substitute DOGMud equivalents where missing).

## Commands

- Player: `weather` — current zone's weather, dominant front + felt intensity.
- Admin (permission key `weather`): `weather status / zones / fronts /
  spawn <type> <zone> [intensity] / clear [zone] / graph [zone] / rebuild`.
- Wired per the admin-command checklist: handler, registration, **helpfile**.

## Config (`Modules.weather.*` in `_datafiles/config.yaml`)

Same knobs as the standalone module: `Enabled: true`, `Seed: 0` (stable
derivation from zone names), `TickEveryGameHours: 1`, `MaxActiveFronts: 8`,
`SpawnRateScale: 1.0`, `EmoteMode: module`, `EmoteEveryRounds: 20`,
`BuffsEnabled: false`, `Persist: true`, `IncludeSecretExits: true`,
`RebuildGraphOnBoot: false`.

## Verification items (resolve during implementation)

1. DOGMud instance-zone naming vs the crawler's `instance_*`/`ephemeral_*`
   skip patterns — confirm instanced/ephemeral zones are actually excluded,
   extend the skip list if DOGMud names differ.
2. Plugin-storage API parity (geography cache + sim state persistence).
3. `MutatorList.Remove` decayintoid behavior parity.
4. Color pattern availability (see Content plan).
5. Crawler behavior on DOGMud's cross-zone oneshot/portal exits — non-compass
   exits are non-spatial for the mapper but the weather crawler counts *all*
   exits as adjacency by design; sanity-check the resulting graph with
   `weather graph` on a few zones.

## Testing

- Pure-package tests (`sim`, `crawler`, `content`) run in-tree:
  `go test ./modules/weather/...`.
- New unit tests for the two indoor-model changes: outdoor-only room
  reconciliation, intensity-banded indoor emote selection (including the
  empty-`mild` silence case).
- Boot smoke per SOP: wipe instance saves, boot, watch
  `mutators.LoadDataFiles loadedCount` rise by 8, `Weather: built geography
  graph zones=... edges=...`, no panics.
- Live smoke: `weather spawn storm <zone> 0.9` → outdoor room shows tag +
  description + alert; adjacent indoor room shows none of those but hears a
  `strong` indoor line within an emote cycle; `weather clear` cleans up;
  reboot restores fronts (`Persist`).

## Out of scope (deferred)

- Gameplay hooks (weather buffs, ranged/perception/foraging modifiers) — a
  natural follow-up once the ranged weapons system lands.
- GMCP/web-client weather display (would suit the dashboard).
- Seasons (v2 in the source repo's roadmap; data formats leave the seams).
- Per-room *outdoor* refinement (e.g. sheltered alley vs open square).
