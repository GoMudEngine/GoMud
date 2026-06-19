# Weather + Seasons Sync (DOGMud ← standalone weather-module) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bring DOGMud's vendored `modules/weather/` up to the standalone repo's
v0.2.0 (seasons S1–S3 + M4 per-room-refinement/admin/events), re-applying
DOGMud's adapter divergences, then tune it for DOGMud's world.

**Architecture:** The standalone (`C:/Users/Calabe Davis/workspace/weather-module`)
is the source of truth for new code. The pure packages (`sim`, `seasons`,
`crawler`, `content`) merge near-cleanly; the engine bridge, module root, and
emote-content model carry DOGMud-specific divergences that must be preserved.
Seasons sit on top of the proven `sim.Step` core as a pure climate transform —
`sim.Step` itself is untouched.

**Tech Stack:** Go, yaml.v2, DOGMud `internal/gametime` (fixed 12-month/365-day
calendar), `internal/rooms` biome registry, `internal/mutators`, `os.DirFS`
data loading (DOGMud does NOT embed).

---

## Source-of-truth paths

- **Standalone (new code):** `C:/Users/Calabe Davis/workspace/weather-module/`
  (referred to below as `$SA`). Verified at v0.2.0, all suites green.
- **DOGMud target:** `C:/Users/Calabe Davis/workspace/DOGMud/modules/weather/`
  (`$DM`) + data under `_datafiles/world/dogmud/`.
- **Specs:** `$SA/docs/superpowers/specs/2026-06-10-seasons-design.md` (the
  authoritative seasons design) and `…/2026-06-08-weather-module-design.md`.

## DOGMud divergences to PRESERVE (the "real merge, not a copy" list)

When a shared file is merged, start from the `$SA` version and re-apply these:

1. **Banded indoor emotes.** DOGMud's `content.Tables` uses
   `Indoor map[string]IndoorPool` (`IndoorPool{Mild,Strong []string}`) + a
   `StrongFeltThreshold = 0.5` const. All 8 shipped DOGMud emote YAMLs
   (`_datafiles/world/dogmud/weather/emotes/*.yaml`) use the `mild:`/`strong:`
   schema. The standalone replaced this with flat `Indoor map[string][]string`
   + a `Seasonal` variant layer. **Merge target keeps banding AND adds
   seasonal** — see Task 8 for the merged `Pick` signature.
2. **`SendText(category, …)` + `CategoryWeather`.** DOGMud's room/user SendText
   takes a leading message-category arg; weather/season emotes pass
   `CategoryWeather`. Upstream has no category param.
3. **`os.DirFS` data loading (no embed).** DOGMud loads data from
   `_datafiles/world/dogmud/…` at runtime via `os.DirFS`; the standalone embeds
   `files/datafiles/`. Every `fs.FS`/dir argument must point at the DOGMud disk
   layout, not an embed.
4. **`BiomeInfo.Indoor` + `MutatorSpec.OutdoorOnly`.** DOGMud-side engine flags
   (`internal/rooms/biomes.go`, `internal/mutators`) that gate outdoor-only
   mutators out of indoor rooms, filtered in `Room.ActiveMutators`. Keep DOGMud's
   versions of these.
5. **"Instance *" crawler skip.** DOGMud's `crawler/build.go` skips title-case
   `Instance *` zones (DOGMud's instance-zone naming slips past upstream's
   `instance_*` lowercase pattern).
6. **Presentation-only / `BuffsEnabled=false`.** DOGMud ships specs with no
   buffs (presentation-only weather). Seasonal specs likewise ship description-
   only (no default buffs, no default exits) — both are documented builder seams.

## Calendar adaptation (the key engine-bridge divergence)

DOGMud's `internal/gametime` has **no named-calendar system** (no
`GetCalendar`, no `CalendarConfig`). It hardcodes a **365-day year / 12 months**;
`GameDate` carries `Year/Month/Week/Day`, and month is
`1 + floor(day*24/730)` (730 = 365·24/12, gametime.go:220). The seasons
package's `monthOfDay(day, daysPerYear, monthsPerYear)` reduces to the *same*
730-hour month at `(12, 365)`. So the seasons math is already DOGMud-correct;
only the `engine/calendar.go` bridge must be rewritten (Task 4) to return a
fixed `(12, 365)` shape and read `gametime.GetDate().Day`.

## Biome → track bindings (DOGMud's 17 biomes)

DOGMud world biomes: `cave dungeon fort house spiderweb` (indoor) + `city
cliffs desert farmland forest land mountains road shore snow swamp water`
(outdoor). DOGMud's CURRENT `DefaultClimate` only defines
`plains/forest/mountain/tundra/swamp/ocean/desert/default`, so most real biomes
fall through to `default` (a latent coverage bug). The standalone's
`sim/climate.go` adds profiles for `mountains/cliffs/snow/shore/water/farmland/
land/road/city/fort/slums/jungle` — which align with DOGMud's real ids — so
bringing it over fixes coverage. **DOGMud-specific binding decisions** (Task 12):
- Outdoor temperate (default): `city cliffs farmland forest land mountains road
  shore snow swamp water`.
- `desert` → unbound (no seasonal cycle), matching the standalone.
- `jungle`/`slums` profiles: harmless to keep (no DOGMud rooms use them); the
  `monsoon` track then has no bound biome and loads but stays dormant (fail-soft,
  validated). Tuning may bind `swamp → monsoon` for wet/dry flavor — left to
  Task 12.
- Indoor biomes (`cave dungeon fort house spiderweb`): leave unbound. Note
  DOGMud's `fort` is `indoor:true` whereas the standalone bound `fort →
  temperate`; **drop that binding for DOGMud** (an indoor biome shows no
  outdoor seasonal ambience anyway, but keeping it unbound avoids confusion).

---

## Phase 0 — Isolation & baseline

### Task 1: Worktree + green baseline

**Files:** none (setup)

- [ ] **Step 1: Create an isolated worktree** (via `superpowers:using-git-worktrees`).
  Branch: `feature/weather-seasons-sync`.

- [ ] **Step 2: Capture the current-green baseline**

Run: `go test ./modules/weather/...`
Expected: PASS (record the package list + counts; this is the regression anchor).

- [ ] **Step 3: Boot baseline (optional sanity)** — confirm the server boots and
  `mutators.LoadDataFiles() loadedCount=…` shows the current 8 weather + 8
  weather-indoor specs with no panic. Record the count.

- [ ] **Step 4: Commit** the worktree marker (no code yet) or skip to Task 2.

---

## Phase 1 — Pure packages (sim, seasons, calendar bridge)

### Task 2: `sim` — Track key, stock biome profiles, KnownWeatherTypes

**Files:**
- Modify: `modules/weather/sim/climate.go`
- Modify: `modules/weather/sim/weather.go`

- [ ] **Step 1: Add the `Track` field to the climate profile struct.**
In `sim/climate.go`, on the per-biome profile struct (the one with
`Weather/Influence/SpawnWeight`), add:

```go
	// Track names the season cycle this biome follows (seasons package);
	// "" = no seasons for this biome. Carried as data — Step ignores it.
	Track string `json:"track,omitempty"`
```

- [ ] **Step 2: Add stock-world biome profiles + bind temperate.**
Copy the new biome entries from `$SA/sim/climate.go`'s `DefaultClimate()` —
`mountains, cliffs, snow, shore, water, farmland, land, road, city, fort,
slums, jungle` — verbatim, and add `Track: "temperate"` to the existing
`plains/forest/mountain/tundra/swamp/ocean` profiles (jungle → `monsoon`,
desert/default → unbound). Use `diff -w --strip-trailing-cr` against `$SA` to
confirm the only delta is additive.

- [ ] **Step 3: Add `KnownWeatherTypes`.** Copy the `KnownWeatherTypes` var
from `$SA/sim/weather.go`:

```go
var KnownWeatherTypes = []WeatherType{
	"blizzard", Clear, "dust", "fog", "heatwave", "overcast", "rain", "snow", "storm",
}
```

- [ ] **Step 4: Build + test the pure sim package.**
Run: `go test ./modules/weather/sim/...`
Expected: PASS (golden-trace + determinism suite unchanged — that's the
architecture's regression guarantee).

- [ ] **Step 5: Commit** — `feat(weather/sim): add biome Track key + stock biome profiles`

### Task 3: `seasons` package (new, pure)

**Files:**
- Create: `modules/weather/seasons/track.go`
- Create: `modules/weather/seasons/resolve.go`
- Create: `modules/weather/seasons/apply.go`
- Create: `modules/weather/seasons/doc.go`
- Create: `modules/weather/seasons/{track,resolve,apply,arch,moduledata}_test.go`

- [ ] **Step 1: Copy all 9 files verbatim** from `$SA/seasons/`. The package
imports only `modules/weather/sim` + `gopkg.in/yaml.v2` + stdlib — no engine
deps, so it ports unchanged. The import path
`github.com/GoMudEngine/GoMud/modules/weather/sim` is identical in DOGMud.

- [ ] **Step 2: Test.**
Run: `go test ./modules/weather/seasons/...`
Expected: PASS (track validation, resolve incl. wraparound, blend math,
EffectiveClimate pass-through, ZoneSeasons, arch purity).

- [ ] **Step 3: Commit** — `feat(weather/seasons): add pure seasons package (port)`

### Task 4: `engine/calendar.go` (DOGMud rewrite)

**Files:**
- Create: `modules/weather/engine/calendar.go`
- Create: `modules/weather/engine/calendar_test.go`

- [ ] **Step 1: Write the DOGMud calendar bridge** (do NOT copy `$SA`'s —
it depends on the missing `gametime.GetCalendar`). DOGMud uses a fixed
12-month/365-day calendar:

```go
package engine

import (
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/modules/weather/seasons"
)

// DOGMud's gametime hardcodes a 365-day, 12-month year (gametime.go: month =
// 1 + floor(day*24/730), 730 = 365*24/12). There is no named-calendar config,
// so the shape is constant.
const (
	dogmudMonthsPerYear = 12
	dogmudDaysPerYear   = 365
)

// CalendarShape reports the active calendar's (monthsPerYear, daysPerYear) —
// the shape seasons.Load validates track files against.
func CalendarShape() (monthsPerYear, daysPerYear int) {
	return dogmudMonthsPerYear, dogmudDaysPerYear
}

// CalendarNow is the current calendar position for season resolution.
// GameDate.Day is the 1-based day-of-year (gametime subtracts whole years).
func CalendarNow() seasons.CalendarPos {
	return seasons.CalendarPos{DayOfYear: gametime.GetDate().Day, DaysPerYear: dogmudDaysPerYear}
}
```

- [ ] **Step 2: Write a test** asserting `CalendarShape() == (12, 365)` and that
`CalendarNow().DaysPerYear == 365` with a non-negative `DayOfYear`.

```go
func TestCalendarShape_DOGMud(t *testing.T) {
	m, d := CalendarShape()
	if m != 12 || d != 365 {
		t.Fatalf("got (%d,%d), want (12,365)", m, d)
	}
}
```

- [ ] **Step 3: Test.** Run: `go test ./modules/weather/engine/ -run Calendar -v` → PASS
- [ ] **Step 4: Commit** — `feat(weather/engine): DOGMud fixed-calendar season bridge`

---

## Phase 2 — Engine layer

### Task 5: `engine/refine.go` (per-room refinement, M4)

**Files:**
- Create: `modules/weather/engine/refine.go`
- Create: `modules/weather/engine/refine_test.go`

- [ ] **Step 1: Copy `$SA/engine/refine.go` + test.** Re-apply divergence #4
(DOGMud `BiomeInfo.Indoor` / `MutatorSpec.OutdoorOnly`) — confirm the
`rooms`/`mutators` symbols it references exist in DOGMud with the same shape
(verify via codegraph: `RefineRoom`, `RefineOccupiedRooms`, `roomWantId`,
`MutatorIdFor`). Adjust any SendText/category call to DOGMud's signature (#2).
- [ ] **Step 2: Test.** `go test ./modules/weather/engine/ -run Refine` → PASS
- [ ] **Step 3: Commit** — `feat(weather/engine): per-room weather refinement (M4 port)`

### Task 6: `engine/apply.go` merge (season reconcile layer)

**Files:**
- Modify: `modules/weather/engine/apply.go`
- Modify/verify: `modules/weather/engine/apply_test.go`

- [ ] **Step 1: Merge.** Diff `$DM/engine/apply.go` vs `$SA/engine/apply.go`
(`diff -w --strip-trailing-cr`). Bring over the `ReconcileSeasons` /
`reconcileZone`-with-`season-` prefix additions and any refine hooks. Preserve
DOGMud's `OutdoorOnly`/`Indoor` filtering and SendText category usage. The two
namespaces (`weather-*`, `season-*`) must never strip each other — verify the
prefix-scoped reconcile.
- [ ] **Step 2: Test.** `go test ./modules/weather/engine/...` → PASS (namespace
isolation test: weather reconcile never strips `season-*` and vice versa).
- [ ] **Step 3: Commit** — `feat(weather/engine): season reconcile layer`

### Task 7: `engine/emotes.go` + `engine/worldreader.go` merge

**Files:**
- Modify: `modules/weather/engine/emotes.go`
- Modify: `modules/weather/engine/worldreader.go`

- [ ] **Step 1: Merge `engine/emotes.go`.** This is the engine-side caller of
`content.Tables.Pick`. After Task 8 changes the `Pick` signature to take BOTH
`felt` and `season`, this caller must pass both: DOGMud's existing `felt`
(intensity) computation stays, and the new `season` arg comes from the zone's
resolved `ZoneSeason.Season` (`""` when unbound/seasons-off). Add the
seasonal-ambience pass (`SeasonalTables.Pick`) that fires in CALM zones at a
lower cadence and yields to weather emotes (spec §6 / S-R1). Preserve
`CategoryWeather` SendText (#2).
- [ ] **Step 2: Merge `engine/worldreader.go`** (+23 lines): bring over any
season/biome-track reads; preserve DOGMud `os.DirFS` usage (#3).
- [ ] **Step 3: Build (do not test yet — content pkg changes land in Task 8).**
Run: `go build ./modules/weather/...` (expected to fail only if Task 8 not yet
done; sequence Task 8 before re-testing engine).
- [ ] **Step 4: Commit** — `feat(weather/engine): wire seasonal emotes + felt/season Pick`

---

## Phase 3 — Content layer (the emote-model merge)

### Task 8: `content/emotes.go` merged model (banded indoor + seasonal)

**Files:**
- Modify: `modules/weather/content/emotes.go`
- Modify: `modules/weather/content/emotes_test.go`
- Modify: `modules/weather/content/climate.go` (Track key passthrough)

This is the one genuine design merge. **Keep DOGMud's `IndoorPool{Mild,Strong}`
+ `StrongFeltThreshold` (base section stays banded)** and **add the standalone's
seasonal layers**, with seasonal indoor sections ALSO banded (`IndoorPool`) for
schema consistency with DOGMud's existing data.

- [ ] **Step 1: Define the merged types.**

```go
// IndoorPool: banded indoor lines (DOGMud). Mild plays below
// StrongFeltThreshold (usually empty = silence through walls); Strong at/above.
type IndoorPool struct {
	Mild   []string `yaml:"mild"`
	Strong []string `yaml:"strong"`
}

const StrongFeltThreshold = 0.5

// TableSection is one outdoor/indoor pair for a seasonal variant. Indoor is
// banded to match the base section.
type TableSection struct {
	Outdoor map[string][]string   `yaml:"outdoor"`
	Indoor  map[string]IndoorPool `yaml:"indoor"`
}

type Tables map[sim.WeatherType]Table

type Table struct {
	Weather sim.WeatherType       `yaml:"weather"`
	Outdoor map[string][]string   `yaml:"outdoor"`
	Indoor  map[string]IndoorPool `yaml:"indoor"`
	// Seasonal holds optional per-season variants keyed by season NAME.
	Seasonal map[string]TableSection `yaml:"seasonal"`
}
```

- [ ] **Step 2: Merged `Pick` signature — felt AND season.**

```go
// Pick selects one ambient line for (weather, biome, indoor, felt, season).
// Lookup order: season variant section (when season!="" and present) ->
// base section; within a section, biome -> "default". Indoor uses the felt-
// banded pool (Mild below StrongFeltThreshold, else Strong) and never falls
// back to outdoor. roll(n) returns [0,n); pass util.Rand — NEVER the sim RNG.
func (ts Tables) Pick(weather sim.WeatherType, biome string, indoor bool, felt float64, season string, roll func(int) int) string {
	t, ok := ts[weather]
	if !ok {
		return ""
	}
	var lines []string
	if season != "" {
		if v, ok := t.Seasonal[season]; ok {
			lines = bandedSectionLines(v.Outdoor, v.Indoor, biome, indoor, felt)
		}
	}
	if len(lines) == 0 {
		lines = bandedSectionLines(t.Outdoor, t.Indoor, biome, indoor, felt)
	}
	if len(lines) == 0 {
		return ""
	}
	i := roll(len(lines))
	if i < 0 || i >= len(lines) {
		i = 0
	}
	return lines[i]
}

// bandedSectionLines resolves biome -> "default" within an outdoor/indoor pair.
// Outdoor is a flat list; indoor is felt-banded. Indoor never falls back to
// outdoor.
func bandedSectionLines(outdoor map[string][]string, indoor map[string]IndoorPool, biome string, useIndoor bool, felt float64) []string {
	if useIndoor {
		pool, ok := indoor[biome]
		if !ok || (len(pool.Mild) == 0 && len(pool.Strong) == 0) {
			pool = indoor["default"]
		}
		if felt >= StrongFeltThreshold {
			return pool.Strong
		}
		return pool.Mild
	}
	lines := outdoor[biome]
	if len(lines) == 0 {
		lines = outdoor["default"]
	}
	return lines
}
```

- [ ] **Step 3: Seasonal-ambience tables (banded indoor too).** Port the
standalone's `SeasonalKey`, `SeasonalTables`, `seasonalEmoteFile`,
`LoadSeasonalEmotes`, and `SeasonalTables.Pick` — but change their indoor maps
from `map[string][]string` to `map[string]IndoorPool` and their `Pick` to take
`felt`, reusing `bandedSectionLines`.

- [ ] **Step 4: `content/climate.go`** — add the `Track string \`yaml:"track"\``
field to the climate-file struct and pass `Track: cf.Track` through to the
`sim` profile (2-line additive change; copy from `$SA`).

- [ ] **Step 5: Update unit tests** for the merged `Pick` (felt banding paths +
season variant lookup order + fallthrough). Port the standalone's seasonal
lookup-order tests, adapting the indoor expectations to banded pools.

- [ ] **Step 6: Test.**
Run: `go test ./modules/weather/content/... ./modules/weather/engine/...`
Expected: PASS

- [ ] **Step 7: Commit** — `feat(weather/content): merge banded-indoor + seasonal emote model`

---

## Phase 4 — Module root (config, tick, api, commands, events, admin)

### Task 9: `weather_events.go` + `weather_config.go`

**Files:**
- Create: `modules/weather/weather_events.go`
- Modify: `modules/weather/weather_config.go`
- Modify: `modules/weather/weather_config_test.go`

- [ ] **Step 1: Copy `$SA/weather_events.go`** verbatim (`WeatherSeasonChanged`,
`WeatherAdminAction`, `WeatherConfigChanged` event types). Verify DOGMud's
`events` bus interface matches (`Type() string`).
- [ ] **Step 2: Merge `weather_config.go`** (the +442 file). Bring over:
`SeasonsEnabled` (default `true`), `PerRoomRefinement` (RefineOff/Occupied/All)
+ its parse, refine/admin config, the seasonal `seasonalEmoteOneIn` knob.
PRESERVE: DOGMud's `BuffsEnabled=false` default, `os.DirFS` data paths (#3),
`CategoryWeather`, and the DOGMud config-block key names under
`Modules.weather.*`. Diff carefully — much of the 442 is config divergence, not
new feature.
- [ ] **Step 3: Test.** `go test ./modules/weather/ -run Config` → PASS
- [ ] **Step 4: Commit** — `feat(weather): seasons + refine config, events`

### Task 10: `weather_tick.go` + `weather.go` + `weather_api.go`

**Files:**
- Modify: `modules/weather/weather_tick.go`
- Modify: `modules/weather/weather.go`
- Modify: `modules/weather/weather_api.go`

- [ ] **Step 1: Merge `weather.go`** (+56): `loadSeasons()` (load tracks via
`seasons.Load` against `CalendarShape()`, load seasonal emote tables via
`content.LoadSeasonalEmotes`), store `m.tracks`/`m.seasonalEmotes`, the
`seasons active tracks=… seasonalZones=…` boot log. Preserve `os.DirFS` paths
(#3) — tracks at `_datafiles/world/dogmud/weather/seasons/`, seasonal emotes at
`…/weather/emotes/seasons/`.
- [ ] **Step 2: Merge `weather_tick.go`** (+179): the per-tick pipeline —
`pos := engine.CalendarNow()`; `effClimate := seasons.EffectiveClimate(...)`;
`sim.Step(effClimate)`; `engine.Reconcile` (weather); `seasons.ZoneSeasons` +
`engine.ReconcileSeasons` (season layer); `WeatherSeasonChanged` events on flips
(in-memory prev-season map, no persistence); `PerRoomRefinement` dispatch
(`RefineOff`/`RefineOccupied`/`RefineAll`). Preserve DOGMud's tick cadence
config + SendText category.
- [ ] **Step 3: Merge `weather_api.go`** (+28): add `GetSeason(zone) map[string]any`
(`{track,season,blend}`, empty when unbound/off). `GetWeather` unchanged.
- [ ] **Step 4: Build.** `go build ./modules/weather/...` → success
- [ ] **Step 5: Test.** `go test ./modules/weather/...` → PASS
- [ ] **Step 6: Commit** — `feat(weather): season tick pipeline + GetSeason API`

### Task 11: `weather_commands.go` + `weather_admin.go` + `weather_admin_api.go`

**Files:**
- Modify: `modules/weather/weather_commands.go`
- Create: `modules/weather/weather_admin.go`
- Create: `modules/weather/weather_admin_api.go`

- [ ] **Step 1: Merge `weather_commands.go`** (+58): add the local season line to
the player `weather` view + `weather status`/`weather zones`, and the
`weather seasons` subcommand (lists tracks + current positions). Preserve
DOGMud's existing command surface + help wiring.
- [ ] **Step 2: Copy `weather_admin.go` + `weather_admin_api.go`** (M4 admin
surface). Re-apply SendText category (#2) and verify admin-authz matches
DOGMud's pattern (`internal` admin gate). Confirm any new admin command is
fully wired (handler + registration + helpfile) per the admin-command-wiring
checklist.
- [ ] **Step 3: Build + vet + test.**
Run: `go build ./... && go vet ./modules/weather/... && go test ./modules/weather/...`
Expected: PASS
- [ ] **Step 4: Commit** — `feat(weather): season command surface + admin (M4 port)`

---

## Phase 5 — Data files + wiring

### Task 12: Season data + biome bindings + config block

**Files:**
- Create: `_datafiles/world/dogmud/weather/seasons/temperate.yaml`
- Create: `_datafiles/world/dogmud/weather/seasons/monsoon.yaml`
- Create: `_datafiles/world/dogmud/mutators/season_temperate_{winter,spring,summer,autumn}.yaml`
- Create: `_datafiles/world/dogmud/mutators/season_monsoon_{wet,dry}.yaml`
- Create: `_datafiles/world/dogmud/weather/emotes/seasons/temperate_{winter,spring,summer,autumn}.yaml`
- Create: `_datafiles/world/dogmud/weather/emotes/seasons/monsoon_{wet,dry}.yaml`
- Modify: `_datafiles/config.yaml` (`Modules.weather.*`)
- (Climate biome bindings already in `sim/climate.go` from Task 2.)

- [ ] **Step 1: Copy track YAMLs** from `$SA/files/datafiles/seasons/` →
DOGMud `weather/seasons/`. These are calendar-agnostic (month-keyed) and valid
for DOGMud's 12-month year as-is.
- [ ] **Step 2: Copy season mutator specs** from `$SA/files/datafiles/mutators/
season_*.yaml` → DOGMud `mutators/`. Verify each has `decayrate` (safety net),
NO `respawnrate`, NO `decayintoid`, NO buffs, NO exits (presentation-only, #6).
Filename must match `mutators` loader `Filepath()` (id-derived).
- [ ] **Step 3: Port seasonal emote tables** from `$SA/files/datafiles/emotes/
seasons/` → DOGMud `weather/emotes/seasons/`, CONVERTING their flat `indoor:`
maps to the banded `mild:/strong:` schema (Task 8's `IndoorPool`). Outdoor
sections copy unchanged. Where the standalone had a single indoor line, place it
under `strong:` (it's a felt-perceptible indoor line) and leave `mild:` empty.
- [ ] **Step 4: Confirm biome→track bindings** in `sim/climate.go` match the
DOGMud decision (temperate for the 11 outdoor biomes; desert + indoor biomes
unbound; drop the standalone `fort → temperate` binding since DOGMud `fort` is
indoor). Optionally bind `swamp → monsoon` (tuning — leave temperate for now,
revisit in Task 14).
- [ ] **Step 5: Add the config block** to `_datafiles/config.yaml`:
`Modules.weather.SeasonsEnabled: true`, `PerRoomRefinement: occupied` (see
Task 14 for the default decision), plus any existing weather knobs.
- [ ] **Step 6: Commit** — `feat(weather): season tracks, mutators, emotes, bindings`

### Task 13: Boot test (pre-push SOP)

**Files:** none

- [ ] **Step 1: Nuke instance saves** (SOP):
`rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*`
(do NOT touch `shops/`).
- [ ] **Step 2: Boot the server locally.** Confirm clean startup past data
loading: `mutators.LoadDataFiles() loadedCount=…` includes the 8 weather + 8
weather-indoor + 6 season specs with NO `duplicate mutator id` / `filepath
mismatch` / season-validation panic, and a `Weather: seasons active tracks=2
seasonalZones=N` line with no WARN/ERROR.
- [ ] **Step 3: Record** `seasonalZones=N` (expected: the count of zones whose
biome is temperate-bound). Investigate if N is unexpectedly 0.
- [ ] **Step 4: Commit** any boot-fix tweaks.

---

## Phase 6 — Tuning (the user explicitly asked for this)

### Task 14: Tune & smoke

**Files:** `_datafiles/config.yaml`, `sim/climate.go` / sim front-lifetime
knobs, track/mutator YAMLs as needed.

- [ ] **Step 1: Front lifetimes.** Address the standing tuning note: "a 0.9
storm decays to overcast in ~1 sim tick (~150s) on the 19-zone graph." Inspect
the sim's front-decay / lifetime knobs (`sim/tick.go`, `sim/state.go`) and
lengthen front lifetimes so a strong front persists for several ticks. Re-run
`go test ./modules/weather/sim/...` after any constant change (golden traces may
need regen — only if intentional).
- [ ] **Step 2: Per-room refinement default.** Decide `RefineOff` vs
`RefineOccupied` vs `RefineAll` for DOGMud's world size. `RefineOccupied` is the
safe default (only rooms with players get per-room mutators). Confirm no
per-tick perf regression (watch idle CPU vs the
[[reference_prod_perf_baseline]] ~7-8% idle).
- [ ] **Step 3: Biome→track tuning.** Optionally bind `swamp → monsoon` (or add
a DOGMud-bespoke track) if wet/dry reads better than temperate for the marsh
zones. Adjust per-season `weatherWeightMultipliers` to DOGMud's biome mix.
- [ ] **Step 4: Smoke near a season boundary.** Use admin gametime tools to pin
the calendar near a transition; observe odds blend (Task: `weather` shows
season + blend), then the boundary flip (season mutator + `WeatherSeasonChanged`
event), `GetSeason`, and a reboot mid-season (reconcile re-asserts, no persisted
state, no event flood). Verify banded indoor emotes still fire correctly in an
indoor seasonal-adjacent room and seasonal variants render in a bound outdoor
zone.
- [ ] **Step 5: Verbosity/spam check (S-R1).** Confirm seasonal ambience reads as
occasional atmosphere (≈2-3 lines / 10 min in an occupied bound zone), weather
emotes pre-empt seasonal ones, and one ambient line max per pass.
- [ ] **Step 6: Update PATCH_NOTES.md** with the weather+seasons sync entry.
- [ ] **Step 7: Commit** — `tune(weather): front lifetimes, refinement default, season bindings`

---

## Self-review checklist (run before handoff)

- Every shared-file merge task names the DOGMud divergences to preserve (the
  numbered list) — ✓ Tasks 2,5,6,7,8,9,10,11.
- The emote-model merge (the one real design fork) is fully specified with the
  merged `Pick` signature + `bandedSectionLines` — ✓ Task 8.
- The calendar incompatibility is resolved with a concrete DOGMud bridge — ✓ Task 4.
- Biome→track bindings enumerated for DOGMud's actual 17 biomes — ✓ Task 12 + plan header.
- Boot test (pre-push SOP) + instance-save wipe included — ✓ Task 13.
- Tuning phase covers the known front-lifetime issue + perf watch — ✓ Task 14.

## Verification gates (every task)

- `go build ./...` after any signature change.
- `go test ./modules/weather/...` is the regression anchor (golden/determinism
  suites must stay green — they prove `sim.Step` is untouched).
- Final: boot the server (Task 13) — YAML load-time panics are invisible to
  `go build` (pre-push SOP).
