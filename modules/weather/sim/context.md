# sim Package Context

## Overview
`sim` is the pure, engine-independent core of the weather module: the geography
`Graph` (produced by `crawler`, consumed everywhere) and the deterministic
weather simulation that runs on it. Nothing here imports the GoMud engine
(`internal/*`) — enforced by `arch_test.go` — and the package is stdlib-only,
so everything is unit-testable without a server and portable across GoMud and
DOGMud. All randomness flows through a serializable PRNG carried in `State`,
making every run exactly reproducible from a seed.

## Key Components
### Core Files
- **graph.go**: `Graph`, `ZoneNode`, `Edge`; `GraphVersion`; JSON cache codec
  (`ToJSON`/`FromJSON`); `Zones()` (sorted, for deterministic iteration);
  `Neighbors` (lazy adjacency index); `FindZone` (case-insensitive lookup).
- **rng.go**: `RNG` — serializable splitmix64 PRNG; its entire state is one
  `uint64` cursor (`State.RNGState`).
- **weather.go**: core value types — `WeatherType` (open, data-driven; `Clear`
  is the calm baseline), `ZoneId` (= `string`), `FrontId`, `Front`, `State`,
  `Clock`, `ZoneChange`, `StateDiff`; `clamp01`.
- **climate.go**: `ClimateProfile`/`WeatherInfluence`/`Climate` (biome →
  weather weights, terrain influence, spawn weight); `Climate.For` (falls back
  to the `"default"` profile); `DefaultClimate()` (built-in profiles for the
  standard biomes); `Config`/`DefaultConfig()` (front budget, spawn chance,
  history length, hard age cap, coverage falloff/threshold/radius).
- **tick.go**: `Step` — the simulation tick — and its helpers (`ageAndFeedback`,
  `moveFronts`, `evolveTypes`, `removeDead`, `spawnFronts`, `resolveWeather`,
  `zonesWithin`, `diffWeather`, weighted-pick helpers).
- **state.go**: `NewState`/`DeriveSeed` plus the `State` persistence codec
  (`ToJSON`/`StateFromJSON`).
- **query.go**: `Coverage`/`Covering` — read-only front-projection query that
  mirrors `resolveWeather`'s coverage rule exactly (a consistency test pins
  them together).
- **mutate.go**: `ForceSpawn`/`ClearZones` — pure admin-action mutations.
- **arch_test.go**: purity guardrail — fails if any file imports a
  `GoMudEngine/GoMud/internal` path.

### Key Structures
```go
type ZoneNode struct {                 // one zone = one graph node
    Zone, Biome string
    Rooms       int
    HasOutdoor  bool
}
type Edge struct {                     // undirected, canonical (A <= B)
    A, B   string
    Weight int                         // # of room-exits crossing the border
}
type Graph struct {
    Version      int
    BuiltAtRound uint64
    Nodes        map[string]ZoneNode
    Edges        []Edge
    Components   int                   // connected-component count
    // + unexported adj: lazy adjacency index; nil after FromJSON,
    //   rebuilt on the first Neighbors call (not serialized).
}
type Front struct {                    // one traveling weather system
    Id        FrontId
    Type      WeatherType
    Zone      ZoneId    // center
    Intensity float64   // 0..1; <= 0 means death
    Moisture  float64   // 0..1
    Age, MaxAge int     // soft cap; past MaxAge decay accelerates
    History   []ZoneId  // bounded recent path (no immediate backtrack)
}
type State struct {                    // full sim state — serializable
    Round    uint64
    RNGState uint64                    // PRNG cursor
    NextID   FrontId
    Fronts   []Front
    Weather  map[ZoneId]WeatherType    // resolved per-zone weather
}
type Coverage struct {
    Front     Front
    Effective float64  // projected intensity at the queried zone
    Hops      int      // graph distance from the front's center
}
```

## Core Functions
### Graph
- `(*Graph) ToJSON / FromJSON` — on-disk cache format (indented JSON);
  `GraphVersion` lets a loader detect a stale cache and rebuild.
- `(*Graph) Neighbors(zone) []Edge` — adjacent zones, each Edge oriented from
  the queried zone (`Edge.A == zone`). Returns a **shared slice** from the
  lazily-built index: callers MUST NOT mutate it (copy before sorting). Nil
  for unknown or isolated zones.
- `(*Graph) FindZone(name) (string, bool)` — case-insensitive resolution to
  the canonical zone key; exact match wins.

### Simulation
- `Step(prev State, g *Graph, climate Climate, cfg Config, now Clock) (State, StateDiff)`
  — one coarse tick, a pure function of its inputs. In order: age fronts and
  apply the occupied biome's influence (the weather ← biome feedback half);
  move fronts along edges (chance damped by `MovementResistance`, destination
  weighted by edge weight, no immediate backtrack via `History`); re-roll a
  *moved* front's type from the destination climate (biased to keep its
  current type, so changes read naturally); drop dead fronts (intensity ≤ 0 or
  past `FrontHardAge`); maybe spawn one front under `MaxActiveFronts` (origin
  weighted by climate `SpawnWeight`, type from the origin's weights); resolve
  per-zone weather with **intensity-scaled area coverage** — each front
  projects onto zones within `MaxFrontRadius` hops at
  `Intensity × CoverageFalloff^hops`, covering while `>= MinProjected`; per
  zone the highest effective intensity wins (ties → lowest front id);
  frontless zones are `Clear` — then emit the `StateDiff` of changed zones
  (sorted; a `From` of `""` means "previously unset", not "was clear").
- `NewState(seed) State` — fresh run (NextID 1, empty non-nil Weather map).
- `DeriveSeed(g) uint64` — stable default seed: FNV-1a over sorted zone names,
  so each world keeps its seed across boots but two worlds differ. Used when
  the configured seed is 0.
- `(State) ToJSON / StateFromJSON` — persistence codec (the engine layer wraps
  it in a versioned envelope; see `engine/state.go`).

### Queries & admin mutations
- `Covering(g, fronts, cfg, zone) []Coverage` — every front whose projection
  reaches `zone`, strongest first. Powers the player `weather` view and the
  exported `GetWeather` intensity.
- `ForceSpawn(prev, g, cfg, wtype, zone, intensity, now) (State, StateDiff, bool)`
  — inject a front, bypassing budget and spawn chance but flowing through the
  same resolve+diff path as `Step`. `intensity <= 0` defaults to 0.6; fixed
  `MaxAge` 24; ok=false for an unknown zone. Pure — input state unmodified.
- `ClearZones(prev, g, cfg, zones, now) (State, StateDiff)` — nil/empty zones
  removes every front; named zones remove every front whose *coverage* reaches
  one of them (so the zone actually clears even when the front is centered
  elsewhere). Pure. Kept fronts share `History` backing arrays with the input;
  any later mutation must go through `cloneFronts` (`Step` does).
- **Neither admin mutation consumes RNG** — admin actions must not perturb the
  deterministic trace.

## Determinism
All randomness comes from the `RNG` built from `State.RNGState`; the advanced
cursor is written back into the next `State`. Same seed + graph + tick count ⇒
byte-identical `State` (asserted by `TestStep_Deterministic` via
`reflect.DeepEqual` and by a golden-trace test). Anything that would consume
randomness outside `Step` (emote selection, admin spawns) deliberately does
not touch this RNG.

## Dependencies
Standard library only (`encoding/json`, `sort`, `strings`). No engine imports
(enforced), no third-party imports.

## Consumers
- `crawler.Build` produces the `*Graph`; `engine.DecodeCache` round-trips it.
- The module root drives `Step` from the weather tick and feeds
  `State.Weather` to `engine.Reconcile`; commands/exports call `Covering`,
  `ForceSpawn`, `ClearZones`, `NewState`, `DeriveSeed`, `FindZone`.
- `content.LoadClimate` returns a `Climate`; `engine.EmitAmbient` reads
  `State.Weather`.

## Testing
- `graph_test.go`: JSON round-trip, `Neighbors` (stability, orientation,
  unknown/isolated zones, post-decode rebuild), `FindZone`.
- `rng_test.go`: determinism, cursor round-trip, ranges.
- `weather_test.go`: `clamp01`.
- `climate_test.go`: profile fallback, defaults.
- `tick_test.go`: per-behavior tests (feedback, movement resistance, type
  evolution, death, budget, coverage), golden trace, full-state determinism,
  storm-dies-crossing-mountains feedback scenario.
- `state_test.go`: `NewState`, `DeriveSeed`, state JSON round-trip.
- `query_test.go`: `Covering` projection/ordering + consistency with
  `resolveWeather`.
- `mutate_test.go`: `ForceSpawn`/`ClearZones` incl. purity assertions.
- `arch_test.go`: purity guardrail.

## Deferred (later milestones)
- **Prevailing-wind direction** (biasing where fronts originate/travel, e.g.
  west→east) needs directional edge metadata, derivable by a future crawler
  pass from each exit's `MapDirection`.
- Calm-zone variety (occasional light fog/overcast in frontless zones) and a
  windward "orographic" precipitation spike are noted enrichments, not built.
