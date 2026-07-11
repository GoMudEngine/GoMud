# Weather Module Backport Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Backport the standalone GoMud weather module into DOGMud as a full DOGMud-native, presentation-only weather system per `docs/superpowers/specs/2026-06-10-weather-module-backport-design.md`.

**Architecture:** The five Go packages from `C:/Users/Calabe Davis/workspace/weather-module` vendor into `modules/weather/` (pure packages byte-identical; adapter layer gets DOGMud edits). All data files live in `_datafiles/world/dogmud/` (mutators in the engine's existing flat dir, climate/emotes under a new `weather/` subtree). Weather applies as zone-level mutators; indoor rooms are filtered at render time via two new generic engine flags (`BiomeInfo.Indoor`, `MutatorSpec.OutdoorOnly`). Indoor ambient emotes are intensity-banded (mild = silence, strong = muted lines).

**Tech Stack:** Go 1.25, yaml.v2, GoMud plugin API (`internal/plugins`), `internal/mutators`, `internal/rooms`, `internal/messaging`, `internal/gametime`.

**Branch:** `feature/weather-module-backport` off `master`.

**Verified API facts (do not re-derive):**
- `users.UserRecord.SendText(cat messaging.Category, txt string)` — userrecord.go:424
- `rooms.Room.SendText(cat messaging.Category, txt string, excludeUserIds ...int)` — rooms.go:263
- `messaging.CategoryWeather` and `messaging.CategorySystem` exist — messaging.go:89,78
- `rooms.GetZoneConfig(zone) *ZoneConfig` (has `.Mutators mutators.MutatorList`), `rooms.GetRoomsWithPlayers()`, `rooms.GetAllZoneNames()`, `rooms.GetZoneBiome(zone)` (returns `DefaultBiome`), `rooms.GetAllZoneRoomsIds(zone)`, `rooms.LoadRoom(id)` — all exist with upstream-compatible signatures (roommanager.go)
- `Room.GetBiome() *BiomeInfo` never returns nil in practice (falls back to default biome) — rooms.go:2567; `BiomeInfo.BiomeId string` field exists — biomes.go:15
- `Room.ActiveMutators(yield func(mutators.Mutator) bool)` — rooms.go:2584 — is the single merge point of room + zone mutators used by render paths
- `mutators.MutatorList` has `Add/Remove/Has/GetActive`; `MutatorSpec` has `PlayerBuffIds/MobBuffIds/NativeBuffIds/DecayRate/LightMod/...` — mutators.go
- Mutator specs load from `<DataFiles>/mutators` flat dir; `DataFiles: _datafiles/world/dogmud` (config.yaml:200) → `_datafiles/world/dogmud/mutators/`
- `plugins.Plugin`: `AddUserCommand(cmd, handler, allowWhenDowned, isAdminOnly)`, `ExportFunction`, `WriteBytes/ReadBytes` (persists to `<DataFiles>/plugin-data/`), `Config.Get(name)` reads flattened `Modules.<plugin>.<name>` from config.yaml — plugins.go, pluginconfig.go
- `events.NewRound` has `.RoundNumber`; `events.RegisterListener(events.NewRound{}, fn)` — pattern used by modules/follow
- `gametime.GetDate().AddPeriod("N hours") uint64` — gametime.go:277
- `user.HasRolePermission(permissionId string, simpleMatch ...bool) bool` — userrecord.go:484, same as upstream
- DOGMud color patterns: `gray`, `blue`, `mute-dblue`, `mute-lblue`, `brown`, `orange`, `swamp`, `gold` exist; **`frost` and `embers` do NOT** — use `mute-lblue` for snow/blizzard, `orange` for heatwave, `brown` for dust
- DOGMud module import path is `github.com/GoMudEngine/GoMud/...` (fork keeps upstream module name)
- Source repo path: `C:/Users/Calabe Davis/workspace/weather-module`

---

### Task 0: Branch + vendor the module source

**Files:**
- Create: `modules/weather/` (sim/, crawler/, content/, engine/, root *.go from source repo)

- [ ] **Step 1: Create branch**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
git checkout -b feature/weather-module-backport master
```

- [ ] **Step 2: Copy module source (Go packages only — no go.mod/go.sum/docs/scripts/files)**

```bash
SRC="C:/Users/Calabe Davis/workspace/weather-module"
mkdir -p modules/weather
cp "$SRC"/*.go modules/weather/
cp -r "$SRC"/sim "$SRC"/crawler "$SRC"/content "$SRC"/engine modules/weather/
cp "$SRC"/context.md modules/weather/ 2>/dev/null || true
```

Do NOT copy: `go.mod`, `go.sum`, `LICENSE`, `README.md`, `docs/`, `scripts/`, `files/`, `.gitattributes`, `.gitignore`, `CONTRIBUTING.md`. The `files/` data is replaced by `_datafiles` content in Tasks 6–8.

- [ ] **Step 3: Regenerate all-modules.go**

```bash
go generate ./...
git diff modules/all-modules.go
```

Expected: `modules/all-modules.go` gains a `weather` import line. If `go generate` produces nothing, check `modules/all-modules.go` for the generator comment and add the import by hand following the existing pattern.

- [ ] **Step 4: Build to surface compile errors (EXPECTED to fail)**

```bash
go build ./... 2>&1 | head -30
```

Expected failures (fix in Tasks 1–5): `user.SendText` arity in weather.go/weather_commands.go, `//go:embed files/*` with no files dir, `room.SendText` arity in engine/emotes.go. Record any OTHER errors — they are fork drift the later tasks must address.

- [ ] **Step 5: Commit the raw vendor (does not build yet — fine on a feature branch)**

```bash
git add modules/weather modules/all-modules.go
git commit -m "chore(weather): vendor weather module source from standalone repo (pre-port)"
```

---

### Task 1: Engine flag — `BiomeInfo.Indoor`

**Files:**
- Modify: `internal/rooms/biomes.go` (struct at line 14)
- Modify: `_datafiles/world/dogmud/biomes/cave.yaml`, `dungeon.yaml`, `house.yaml`, `fort.yaml`, `spiderweb.yaml`
- Test: `internal/rooms/biomes_test.go` (create or extend existing)

- [ ] **Step 1: Write the failing test**

Add to `internal/rooms/biomes_test.go` (create the file with package `rooms` if absent):

```go
func TestBiomeInfo_IndoorFlag(t *testing.T) {
	yamlSrc := []byte("biomeid: testcave\nname: Test Cave\nsymbol: \"^\"\nindoor: true\n")
	var bi BiomeInfo
	if err := yaml.Unmarshal(yamlSrc, &bi); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !bi.Indoor {
		t.Error("expected indoor: true to set BiomeInfo.Indoor")
	}

	var bi2 BiomeInfo
	if err := yaml.Unmarshal([]byte("biomeid: plains\nname: Plains\nsymbol: \".\"\n"), &bi2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if bi2.Indoor {
		t.Error("expected Indoor to default false")
	}
}
```

Use the same yaml import the package already uses (`gopkg.in/yaml.v2`).

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/rooms/ -run TestBiomeInfo_IndoorFlag -v`
Expected: FAIL (compile error: `bi.Indoor` undefined)

- [ ] **Step 3: Add the field**

In `internal/rooms/biomes.go`, after the `MovementCost` field:

```go
	Indoor         bool    `yaml:"indoor,omitempty"`       // Sheltered from weather; outdoor-only mutators don't render here
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/rooms/ -run TestBiomeInfo_IndoorFlag -v`
Expected: PASS

- [ ] **Step 5: Flag the five indoor biomes**

Append `indoor: true` to each of `_datafiles/world/dogmud/biomes/cave.yaml`, `dungeon.yaml`, `house.yaml`, `fort.yaml`, `spiderweb.yaml` (read each file first; add the key at top level alongside `biomeid:`).

- [ ] **Step 6: Commit**

```bash
git add internal/rooms/biomes.go internal/rooms/biomes_test.go _datafiles/world/dogmud/biomes/
git commit -m "feat(rooms): indoor flag on biomes; mark cave/dungeon/house/fort/spiderweb indoor"
```

---

### Task 2: Engine flag — `MutatorSpec.OutdoorOnly` + render-time filter in `ActiveMutators`

**Files:**
- Modify: `internal/mutators/mutators.go` (MutatorSpec struct at line 58)
- Modify: `internal/rooms/rooms.go` (`ActiveMutators` at line 2584)
- Test: `internal/rooms/rooms_mutator_filter_test.go` (create)

- [ ] **Step 1: Write the failing test**

Create `internal/rooms/rooms_mutator_filter_test.go`. Use the package's existing test seam (`seedRegistry()` in rooms_test.go seeds zones/rooms; check how it builds rooms and reuse it — if it doesn't cover mutator specs, register a spec via the mutators package test helpers or set up the minimal registry the same way `internal/mutators` tests do). The behavior under test:

```go
package rooms

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mutators"
)

// An outdoor-only zone mutator must be yielded for outdoor-biome rooms and
// skipped for indoor-biome rooms; a normal mutator is yielded for both.
func TestActiveMutators_OutdoorOnlySkippedIndoors(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	// Seed two biomes: one indoor, one outdoor.
	SeedBiomesForTest(t, BiomeInfo{BiomeId: "testfield", Name: "Field", Symbol: "."},
		BiomeInfo{BiomeId: "testhouse", Name: "House", Symbol: "H", Indoor: true})

	// Register one outdoor-only spec and one normal spec.
	mutators.SeedSpecsForTest(t,
		mutators.MutatorSpec{MutatorId: "weather-test-rain", OutdoorOnly: true},
		mutators.MutatorSpec{MutatorId: "test-sanctuary"},
	)

	zc := GetZoneConfig("TestZone")
	zc.Mutators.Add("weather-test-rain")
	zc.Mutators.Add("test-sanctuary")

	outdoorRoom := roomManager.rooms[1]
	outdoorRoom.Biome = "testfield"
	indoorRoom := roomManager.rooms[2]
	indoorRoom.Biome = "testhouse"

	got := map[string]bool{}
	for mut := range outdoorRoom.ActiveMutators {
		got[mut.MutatorId] = true
	}
	if !got["weather-test-rain"] || !got["test-sanctuary"] {
		t.Errorf("outdoor room should see both mutators, got %v", got)
	}

	got = map[string]bool{}
	for mut := range indoorRoom.ActiveMutators {
		got[mut.MutatorId] = true
	}
	if got["weather-test-rain"] {
		t.Error("indoor room must NOT see outdoor-only mutator")
	}
	if !got["test-sanctuary"] {
		t.Error("indoor room should still see normal mutator")
	}
}
```

Note: `SeedBiomesForTest` exists (internal/rooms/test_helpers.go:39 — verify its exact signature and adapt the call). `mutators.SeedSpecsForTest` may NOT exist — check `internal/mutators` for an existing test seam (a test that registers specs); if none, add a small exported test helper `SeedSpecsForTest(t *testing.T, specs ...MutatorSpec)` in `internal/mutators/test_helpers.go` that inserts into `allMutators` and restores via `t.Cleanup`.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/rooms/ -run TestActiveMutators_OutdoorOnlySkippedIndoors -v`
Expected: FAIL (compile: `OutdoorOnly` undefined)

- [ ] **Step 3: Add the spec field and the filter**

In `internal/mutators/mutators.go`, add to `MutatorSpec` after `Pvp`:

```go
	OutdoorOnly bool `yaml:"outdooronly,omitempty"` // skip this mutator's effects in indoor-biome rooms (weather etc.)
```

In `internal/rooms/rooms.go`, replace `ActiveMutators`:

```go
func (r *Room) ActiveMutators(yield func(mutators.Mutator) bool) {

	var activeMutators mutators.MutatorList
	if zoneConfig := GetZoneConfig(r.Zone); zoneConfig != nil {
		activeMutators = append(r.Mutators.GetActive(), zoneConfig.Mutators.GetActive()...)
	}

	indoor := false
	if b := r.GetBiome(); b != nil {
		indoor = b.Indoor
	}

	for _, mut := range activeMutators {
		if indoor {
			if spec := mut.GetSpec(); spec != nil && spec.OutdoorOnly {
				continue
			}
		}
		if !yield(mut) {
			return
		}
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/rooms/ -run TestActiveMutators_OutdoorOnlySkippedIndoors -v`
Expected: PASS

- [ ] **Step 5: Run the full rooms + mutators packages**

Run: `go test ./internal/rooms/ ./internal/mutators/`
Expected: PASS (no regressions — the filter only activates for indoor biomes + outdooronly specs, neither of which exist in current data)

- [ ] **Step 6: Commit**

```bash
git add internal/mutators/ internal/rooms/
git commit -m "feat(mutators): outdooronly spec flag, filtered in ActiveMutators for indoor biomes"
```

---

### Task 3: Port the module root (compile fixes, config, content paths, no embed)

**Files:**
- Modify: `modules/weather/weather.go`
- Modify: `modules/weather/weather_tick.go`
- Modify: `modules/weather/weather_commands.go`
- Modify: `_datafiles/config.yaml` (Modules block at line ~1341)

- [ ] **Step 1: Remove the embedded FS and fix sendLine**

In `modules/weather/weather.go`:

Remove these lines (the embed):

```go
import (
	"embed"
	...
)

//go:embed files/*
var files embed.FS
```

and in `init()` remove:

```go
	if err := module.plug.AttachFileSystem(files); err != nil {
		panic(err)
	}
```

Replace `sendLine` (bottom of file):

```go
// sendLine writes one line to a user. It is the ONLY place this module calls
// the engine's SendText. DOGMud's fork signature is SendText(category, text).
func sendLine(user *users.UserRecord, text string) {
	user.SendText(messaging.CategorySystem, text)
}
```

Add `"github.com/GoMudEngine/GoMud/internal/messaging"` to imports; remove `"embed"`.

- [ ] **Step 2: Point loadContent at _datafiles**

In `modules/weather/weather_tick.go`, replace `loadContent`:

```go
// loadContent loads climate profiles and emote tables from the world's
// datafiles tree (_datafiles/world/dogmud/weather/...). Both fail soft:
// defaults / silence plus a warning.
func (m *weatherModule) loadContent() {
	dataFS := os.DirFS(configs.GetFilePathsConfig().DataFiles.String())

	climate, err := content.LoadClimate(dataFS, "weather/climate")
	if err != nil {
		mudlog.Warn("Weather: climate profiles failed to load; using defaults", "error", err)
	}
	m.climate = climate

	tables, err := content.LoadEmotes(dataFS, "weather/emotes")
	if err != nil {
		mudlog.Warn("Weather: emote tables failed to load", "error", err)
	}
	m.tables = tables
}
```

Add imports `"os"` and `"github.com/GoMudEngine/GoMud/internal/configs"`.

- [ ] **Step 3: Add the config block**

In `_datafiles/config.yaml`, under the existing `Modules:` key (line ~1341), add after the `playtest:` block (match its indentation exactly — two spaces for the module name, four for keys):

```yaml
  weather:
    Enabled: true
    Seed: 0
    TickEveryGameHours: 1
    MaxActiveFronts: 8
    SpawnRateScale: 1.0
    EmoteMode: module
    EmoteEveryRounds: 20
    BuffsEnabled: false
    Persist: true
    IncludeSecretExits: true
    RebuildGraphOnBoot: false
```

- [ ] **Step 4: Build (commands file may still fail — that's Step 5)**

Run: `go build ./... 2>&1 | head -20`

- [ ] **Step 5: Fix any remaining compile errors in module root files**

`weather_commands.go` and `weather_api.go` route all user output through `sendLine`, so they should compile once Step 1 lands. `cmdWeather`'s signature `func(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error)` must match DOGMud's `usercommands.UserCommand` type — if the build complains, open `internal/usercommands` for the type definition and adapt the signature only (body unchanged). Apply the same minimal-diff principle to any other drift the build surfaces; record what changed in the commit message.

- [ ] **Step 6: Build clean**

Run: `go build ./...`
Expected: success (engine/ package fixed in Task 4 — if engine/ errors remain, they are limited to `emotes.go`/`worldreader.go`, fixed next; you may temporarily proceed if the ONLY errors are in Task 4's files)

- [ ] **Step 7: Commit**

```bash
git add modules/weather _datafiles/config.yaml
git commit -m "feat(weather): port module root to DOGMud — config block, datafiles content paths, SendText category"
```

---

### Task 4: Adapter — indoor flag from BiomeInfo + CategoryWeather emotes

**Files:**
- Modify: `modules/weather/engine/worldreader.go`
- Modify: `modules/weather/engine/emotes.go`
- Test: `modules/weather/engine/emotes_test.go` (extend — see what exists), `modules/weather/engine/worldreader_test.go`

- [ ] **Step 1: Replace the indoor-biome heuristic with the engine flag**

In `modules/weather/engine/worldreader.go`, delete the `indoorBiomes` map and `isOutdoorBiome` func. Replace with:

```go
// isOutdoorBiome reports whether a biome id is outdoors, per the engine's
// biome registry (BiomeInfo.Indoor, set in biome YAML). Unknown or empty
// biomes default to outdoors.
func isOutdoorBiome(biomeID string) bool {
	if b, ok := rooms.GetBiome(biomeID); ok && b != nil {
		return !b.Indoor
	}
	return true
}
```

Check `rooms.GetBiome`'s exact signature first (biomes.go:134) — it returns `(*BiomeInfo, bool)` per the codegraph trail; adapt if different.

- [ ] **Step 2: Run existing engine tests; fix fixtures**

Run: `go test ./modules/weather/engine/...`

Existing worldreader tests asserted the old hardcoded map (`cave`, `underground`, ...). Update them: tests that exercise indoor detection should seed the biome registry (use `rooms.SeedBiomesForTest`) with an `Indoor: true` biome instead of relying on hardcoded names.

- [ ] **Step 3: Write failing test for intensity-gated indoor emotes**

The emote scheduler change (this step) and the content schema change (Task 5) interlock; write this test against the NEW `EmitAmbient` signature:

In `modules/weather/engine/emotes_test.go` add (adapting to the package's existing fake/seam style — read the file first):

```go
// Indoor rooms under mild weather get silence; under strong weather get the
// strong indoor pool. Outdoor rooms get outdoor lines regardless.
func TestEmitAmbient_IndoorIntensityGate(t *testing.T) {
	// Graph: one zone "Z" with a front of intensity 0.9 centered there (felt=0.9),
	// and a second zone "W" with a weak front (felt=0.2).
	// tables: rain with outdoor default, indoor mild=[], indoor strong=["roof line"].
	// Expect: indoor room in Z hears "roof line"; indoor room in W hears nothing;
	// outdoor room in either hears an outdoor line.
	...
}
```

This test needs live rooms with players, which the engine package can't easily fake — IF the existing `emotes_test.go`/`apply_test.go` fake at the `mutatorSet`/reader seam only and there is no room seam, then instead put the gating logic in `content.Tables.Pick` (pure, fully testable — Task 5) and keep `EmitAmbient` a thin loop, testing only `Pick` exhaustively. Decide by reading the existing engine tests; prefer the pure-function home for the logic either way.

- [ ] **Step 4: Rewrite EmitAmbient with felt intensity + CategoryWeather**

In `modules/weather/engine/emotes.go`:

```go
package engine

import (
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/modules/weather/content"
	"github.com/GoMudEngine/GoMud/modules/weather/sim"
)

// EmitAmbient sends one ambient weather line into each occupied room whose
// zone currently has non-calm weather. The room's biome picks the table
// variant; indoor biomes get the intensity-banded indoor section (mild
// weather is silent indoors). roll is the presentation RNG (pass util.Rand)
// — NEVER the sim RNG. Returns lines sent.
func EmitAmbient(g *sim.Graph, fronts []sim.Front, simCfg sim.Config,
	weather map[sim.ZoneId]sim.WeatherType, tables content.Tables, roll func(int) int) int {

	sent := 0
	felt := map[sim.ZoneId]float64{}

	for _, roomId := range rooms.GetRoomsWithPlayers() {
		room := rooms.LoadRoom(roomId)
		if room == nil {
			continue
		}
		w := weather[room.Zone]
		if w == "" || w == sim.Clear {
			continue
		}

		f, ok := felt[room.Zone]
		if !ok {
			if covers := sim.Covering(g, fronts, simCfg, room.Zone); len(covers) > 0 {
				f = covers[0].Effective
			}
			felt[room.Zone] = f
		}

		biomeId, indoor := "", false
		if b := room.GetBiome(); b != nil {
			biomeId, indoor = b.BiomeId, b.Indoor
		}
		line := tables.Pick(w, biomeId, indoor, f, roll)
		if line == "" {
			continue
		}
		room.SendText(messaging.CategoryWeather, line)
		sent++
	}
	return sent
}
```

Update the caller in `modules/weather/weather.go` `onNewRound`:

```go
	if m.cfg.EmoteMode == EmoteModeModule && evt.RoundNumber >= m.nextEmote {
		engine.EmitAmbient(m.graph, m.state.Fronts, m.simCfg, m.state.Weather, m.tables, util.Rand)
		m.scheduleEmote(evt.RoundNumber)
	}
```

(Compiles only after Task 5 changes `Pick`'s signature — do Tasks 4+5 as one build unit, committing after both test suites pass.)

- [ ] **Step 5: Commit (joint with Task 5 if needed for a green build)**

```bash
git add modules/weather
git commit -m "feat(weather): engine adapter — biome-flag indoor detection, felt-intensity emote gating, CategoryWeather"
```

---

### Task 5: Content — intensity-banded indoor emote schema

**Files:**
- Modify: `modules/weather/content/emotes.go`
- Test: `modules/weather/content/emotes_test.go` (extend)

- [ ] **Step 1: Write the failing tests**

Add to `modules/weather/content/emotes_test.go`:

```go
func TestPick_IndoorIntensityBands(t *testing.T) {
	tables := Tables{
		"rain": {
			Weather: "rain",
			Outdoor: map[string][]string{"default": {"out"}},
			Indoor: map[string]IndoorPool{
				"default": {Mild: nil, Strong: []string{"roof"}},
			},
		},
	}
	first := func(n int) int { return 0 }

	// Outdoor ignores intensity banding entirely.
	if got := tables.Pick("rain", "city", false, 0.1, first); got != "out" {
		t.Errorf("outdoor mild: got %q want %q", got, "out")
	}
	// Indoor below threshold: silence (mild pool empty).
	if got := tables.Pick("rain", "house", true, 0.2, first); got != "" {
		t.Errorf("indoor mild: got %q want silence", got)
	}
	// Indoor at/above threshold: strong pool.
	if got := tables.Pick("rain", "house", true, 0.7, first); got != "roof" {
		t.Errorf("indoor strong: got %q want %q", got, "roof")
	}
}

func TestPick_IndoorBiomeFallback(t *testing.T) {
	tables := Tables{
		"storm": {
			Weather: "storm",
			Indoor: map[string]IndoorPool{
				"default": {Strong: []string{"generic"}},
				"fort":    {Strong: []string{"stone walls"}},
			},
		},
	}
	first := func(n int) int { return 0 }
	if got := tables.Pick("storm", "fort", true, 0.9, first); got != "stone walls" {
		t.Errorf("biome-specific: got %q", got)
	}
	if got := tables.Pick("storm", "house", true, 0.9, first); got != "generic" {
		t.Errorf("default fallback: got %q", got)
	}
	// Indoor NEVER falls back to outdoor: no outdoor section here, mild empty → silence.
	if got := tables.Pick("storm", "fort", true, 0.1, first); got != "" {
		t.Errorf("mild with empty mild pool: got %q want silence", got)
	}
}

func TestParseEmoteTable_IndoorBands(t *testing.T) {
	src := []byte(`weather: rain
outdoor:
  default:
    - "rain falls"
indoor:
  default:
    mild: []
    strong:
      - "rain drums on the roof"
`)
	tbl, err := ParseEmoteTable(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(tbl.Indoor["default"].Strong) != 1 {
		t.Errorf("expected 1 strong indoor line, got %+v", tbl.Indoor["default"])
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./modules/weather/content/ -run 'TestPick_Indoor|TestParseEmoteTable_IndoorBands' -v`
Expected: FAIL (compile: `IndoorPool` undefined, `Pick` arity)

- [ ] **Step 3: Implement the schema + gating**

In `modules/weather/content/emotes.go`:

```go
// StrongFeltThreshold is the felt-intensity at which weather becomes
// perceptible indoors (drumming on roofs, wind in the eaves). Below it,
// indoor rooms get the mild pool — usually empty, i.e. silence.
const StrongFeltThreshold = 0.5

// IndoorPool holds intensity-banded indoor lines for one biome key.
// Mild plays below StrongFeltThreshold (usually empty: light weather
// doesn't register through walls); Strong plays at/above it.
type IndoorPool struct {
	Mild   []string `yaml:"mild"`
	Strong []string `yaml:"strong"`
}

// Table holds the ambient lines for one weather type, keyed by biome with a
// "default" fallback. Outdoor lines are a single pool (the weather TYPE
// already encodes outdoor severity); indoor lines are intensity-banded.
type Table struct {
	Weather string                `yaml:"weather"`
	Outdoor map[string][]string   `yaml:"outdoor"`
	Indoor  map[string]IndoorPool `yaml:"indoor"`
}
```

Replace `Pick`:

```go
// Pick selects one ambient line for (weather, biome, indoor, felt), or ""
// when nothing matches. Fallbacks: exact biome -> "default" biome. Indoor
// never falls back to outdoor — silence beats wrong prose. Indoor pools are
// intensity-banded: felt < StrongFeltThreshold picks Mild (usually empty),
// otherwise Strong. roll(n) must return a value in [0,n); pass util.Rand —
// NEVER the sim RNG. Out-of-range roll results clamp to the first line.
func (ts Tables) Pick(weather sim.WeatherType, biome string, indoor bool, felt float64, roll func(int) int) string {
	t, ok := ts[weather]
	if !ok {
		return ""
	}

	var lines []string
	if indoor {
		pool, ok := t.Indoor[biome]
		if !ok || (len(pool.Mild) == 0 && len(pool.Strong) == 0) {
			pool = t.Indoor["default"]
		}
		if felt >= StrongFeltThreshold {
			lines = pool.Strong
		} else {
			lines = pool.Mild
		}
	} else {
		lines = t.Outdoor[biome]
		if len(lines) == 0 {
			lines = t.Outdoor["default"]
		}
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
```

- [ ] **Step 4: Run the full content + engine + module test suites**

Run: `go test ./modules/weather/...`
Expected: PASS after fixing any old-schema fixtures in `emotes_test.go` / `moduledata_test.go` (the moduledata test validated the shipped `files/` data which no longer exists in the module — repoint it at `_datafiles/world/dogmud/weather/` using `os.DirFS`, or delete it if Task 8's data isn't authored yet and re-enable there).

- [ ] **Step 5: Build everything and commit Tasks 4+5 together**

```bash
go build ./...
go test ./modules/weather/...
git add modules/weather
git commit -m "feat(weather): intensity-banded indoor emote pools (mild=silence, strong=muted lines)"
```

---

### Task 6: Author the 8 weather mutator specs

**Files:**
- Create: `_datafiles/world/dogmud/mutators/weather_overcast.yaml`, `weather_rain.yaml`, `weather_storm.yaml`, `weather_fog.yaml`, `weather_snow.yaml`, `weather_blizzard.yaml`, `weather_dust.yaml`, `weather_heatwave.yaml`

- [ ] **Step 1: Write all 8 spec files**

Rules (enforced by the module's design): every spec carries `outdooronly: true`; NO `playerbuffids`/`mobbuffids`/`nativebuffids`; NO `respawnrate`; NO `decayintoid`; `decayrate` present as self-heal net; `lightmod` only on storm/blizzard/dust; alert only on severe types. Filenames are the mutator id lowercased with non-alphanumerics as `_`.

`weather_overcast.yaml`:
```yaml
mutatorid: weather-overcast
outdooronly: true
namemodifier:
  behavior: append
  text: (overcast)
  colorpattern: gray
descriptionmodifier:
  behavior: append
  text: A flat gray ceiling of cloud hangs low overhead, dulling every color.
  colorpattern: gray
decayrate: 6 hours
```

`weather_rain.yaml`:
```yaml
mutatorid: weather-rain
outdooronly: true
namemodifier:
  behavior: append
  text: (raining)
  colorpattern: blue
descriptionmodifier:
  behavior: append
  text: A steady rain falls, beading and dripping from every surface.
  colorpattern: blue
decayrate: 4 hours
```

`weather_storm.yaml`:
```yaml
mutatorid: weather-storm
outdooronly: true
namemodifier:
  behavior: append
  text: (storm-wracked)
  colorpattern: mute-dblue
descriptionmodifier:
  behavior: append
  text: Rain lashes down in wind-driven sheets while thunder rolls overhead.
  colorpattern: mute-dblue
alertmodifier:
  behavior: append
  text: A storm rages overhead.
  colorpattern: mute-dblue
lightmod: -1
decayrate: 3 hours
```

`weather_fog.yaml`:
```yaml
mutatorid: weather-fog
outdooronly: true
namemodifier:
  behavior: append
  text: (fogbound)
  colorpattern: gray
descriptionmodifier:
  behavior: append
  text: A dense fog presses close, swallowing shapes more than a few paces out.
  colorpattern: gray
decayrate: 4 hours
```

`weather_snow.yaml`:
```yaml
mutatorid: weather-snow
outdooronly: true
namemodifier:
  behavior: append
  text: (snowing)
  colorpattern: mute-lblue
descriptionmodifier:
  behavior: append
  text: Snow drifts down in fat, silent flakes, softening every edge.
  colorpattern: mute-lblue
decayrate: 5 hours
```

`weather_blizzard.yaml`:
```yaml
mutatorid: weather-blizzard
outdooronly: true
namemodifier:
  behavior: append
  text: (blizzard)
  colorpattern: mute-lblue
descriptionmodifier:
  behavior: append
  text: Howling wind drives the snow sideways, scouring everything it touches.
  colorpattern: mute-lblue
alertmodifier:
  behavior: append
  text: A blizzard howls through here.
  colorpattern: mute-lblue
lightmod: -1
decayrate: 3 hours
```

`weather_dust.yaml`:
```yaml
mutatorid: weather-dust
outdooronly: true
namemodifier:
  behavior: append
  text: (dust-choked)
  colorpattern: brown
descriptionmodifier:
  behavior: append
  text: Wind-borne grit scours the air, stinging eyes and gritting teeth.
  colorpattern: brown
alertmodifier:
  behavior: append
  text: A dust storm scours the area.
  colorpattern: brown
lightmod: -1
decayrate: 3 hours
```

`weather_heatwave.yaml`:
```yaml
mutatorid: weather-heatwave
outdooronly: true
namemodifier:
  behavior: append
  text: (sweltering)
  colorpattern: orange
descriptionmodifier:
  behavior: append
  text: The air shimmers with heat; every breath feels drawn through a forge.
  colorpattern: orange
decayrate: 6 hours
```

- [ ] **Step 2: Boot-load check**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . 2>&1 | grep -E "mutators.LoadDataFiles|panic" | head -5
```

Expected: `mutators.LoadDataFiles() loadedCount=14` (6 existing + 8 new), no panic. Kill the server after the line appears.

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/mutators/
git commit -m "content(weather): 8 weather mutator specs — outdoor-only, presentation-only, DOGMud palette"
```

---

### Task 7: Author the 17 climate profiles

**Files:**
- Create: `_datafiles/world/dogmud/weather/climate/<biome>.yaml` × 17

- [ ] **Step 1: Write all 17 files**

Schema (content/climate.go): `biome`, `weather` (type→weight), `influence.intensityDelta/moistureDelta/movementResistance`, `spawnWeight`. **A file replaces the biome's profile wholesale — omitted keys are zero**, so every outdoor file must set all keys explicitly. Weights are relative within the file.

`water.yaml`:
```yaml
biome: water
weather: { clear: 2, overcast: 2, fog: 2, rain: 2, storm: 1 }
influence: { intensityDelta: 0.06, moistureDelta: 0.10, movementResistance: 0.0 }
spawnWeight: 1.4
```

`shore.yaml`:
```yaml
biome: shore
weather: { clear: 3, overcast: 2, fog: 2, rain: 2, storm: 1 }
influence: { intensityDelta: 0.03, moistureDelta: 0.05, movementResistance: 0.0 }
spawnWeight: 1.1
```

`cliffs.yaml`:
```yaml
biome: cliffs
weather: { clear: 3, overcast: 2, fog: 1, rain: 1, storm: 1 }
influence: { intensityDelta: 0.0, moistureDelta: -0.02, movementResistance: 0.1 }
spawnWeight: 0.8
```

`desert.yaml`:
```yaml
biome: desert
weather: { clear: 4, heatwave: 2, dust: 2, overcast: 1 }
influence: { intensityDelta: -0.02, moistureDelta: -0.10, movementResistance: 0.0 }
spawnWeight: 1.0
```

`snow.yaml`:
```yaml
biome: snow
weather: { clear: 2, overcast: 2, snow: 3, blizzard: 1 }
influence: { intensityDelta: 0.02, moistureDelta: 0.0, movementResistance: 0.05 }
spawnWeight: 1.1
```

`mountains.yaml`:
```yaml
biome: mountains
weather: { clear: 3, overcast: 2, snow: 2, storm: 1 }
influence: { intensityDelta: -0.08, moistureDelta: -0.08, movementResistance: 0.4 }
spawnWeight: 0.7
```

`swamp.yaml`:
```yaml
biome: swamp
weather: { fog: 3, overcast: 2, rain: 2, clear: 1 }
influence: { intensityDelta: 0.02, moistureDelta: 0.06, movementResistance: 0.15 }
spawnWeight: 1.0
```

`forest.yaml`:
```yaml
biome: forest
weather: { clear: 3, overcast: 2, rain: 2, fog: 1, storm: 1 }
influence: { intensityDelta: 0.0, moistureDelta: 0.02, movementResistance: 0.1 }
spawnWeight: 0.9
```

`farmland.yaml`:
```yaml
biome: farmland
weather: { clear: 4, overcast: 2, rain: 2, storm: 1 }
influence: { intensityDelta: 0.0, moistureDelta: 0.0, movementResistance: 0.0 }
spawnWeight: 0.9
```

`land.yaml`:
```yaml
biome: land
weather: { clear: 4, overcast: 2, rain: 1, fog: 1 }
influence: { intensityDelta: 0.0, moistureDelta: 0.0, movementResistance: 0.0 }
spawnWeight: 0.8
```

`road.yaml`:
```yaml
biome: road
weather: { clear: 4, overcast: 2, rain: 1 }
influence: { intensityDelta: 0.0, moistureDelta: 0.0, movementResistance: 0.0 }
spawnWeight: 0.6
```

`city.yaml`:
```yaml
biome: city
weather: { clear: 4, overcast: 2, rain: 2, fog: 1 }
influence: { intensityDelta: -0.01, moistureDelta: 0.0, movementResistance: 0.05 }
spawnWeight: 0.7
```

Indoor biomes (fronts never form; crossing terrain influence is mildly
sapping; zero spawn) — `cave.yaml`, `dungeon.yaml`, `house.yaml`,
`fort.yaml`, `spiderweb.yaml`, all five identical except the `biome:` line:

```yaml
biome: cave
weather: { clear: 1 }
influence: { intensityDelta: -0.04, moistureDelta: -0.04, movementResistance: 0.3 }
spawnWeight: 0.0
```

- [ ] **Step 2: Sanity-check weights parse**

Run: `go test ./modules/weather/content/ -v -run TestParseClimate`
(Existing parse tests cover the schema; a quick `go run .` boot in Task 9 validates the real files end-to-end via the module's load-warning log line.)

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/weather/climate/
git commit -m "content(weather): climate profiles for all 17 DOGMud biomes"
```

---

### Task 8: Author the emote tables (DOGMud voice, banded indoor)

**Files:**
- Create: `_datafiles/world/dogmud/weather/emotes/<type>.yaml` × 8 (overcast, rain, storm, fog, snow, blizzard, dust, heatwave)

- [ ] **Step 1: Write all 8 files**

Rules: 80-char lines, no hard numbers, narrator voice. Severe types (storm, blizzard, dust) MUST have non-empty `strong` indoor pools (always audible); mild types may leave `mild` empty (silence). Add biome-specific outdoor flavor where it earns its keep.

`rain.yaml`:
```yaml
weather: rain
outdoor:
  default:
    - "Rain patters down steadily, beading on every surface."
    - "A cold runnel of rainwater finds its way down the back of your neck."
    - "Puddles widen and merge underfoot."
    - "The rain eases for a breath, then settles back into its steady rhythm."
  forest:
    - "Rain drips from leaf to leaf, a thousand small drumbeats overhead."
    - "The canopy sheds fat, gathered drops long after each gust."
  city:
    - "Rainwater chuckles along the gutters and pools between the cobbles."
    - "Awnings sag and drip; somewhere a shutter bangs against the wet."
  swamp:
    - "Rain stipples the standing water, setting the reeds nodding."
indoor:
  default:
    mild: []
    strong:
      - "The sound of rain on the roof settles into a rhythmic, soothing patter."
      - "Water trickles past outside; somewhere a slow drip finds its rhythm."
```

`storm.yaml`:
```yaml
weather: storm
outdoor:
  default:
    - "Thunder cracks close enough to feel in your teeth."
    - "Wind-driven rain stings like flung gravel."
    - "Lightning whitewashes the world for a heartbeat, then the dark slams back."
    - "The wind leans on you like a shoulder, gusting and shoving."
  forest:
    - "The trees groan and thrash, shedding twigs and torn leaves."
  city:
    - "A loose shutter hammers somewhere down the street, slave to the wind."
    - "Rain sheets off the rooflines in ragged silver curtains."
  water:
    - "The water heaves into whitecaps, spray mixing with the rain."
indoor:
  default:
    mild:
      - "Wind moans low around the eaves."
    strong:
      - "Thunder rolls through the walls, rattling anything not nailed down."
      - "The storm hammers at the roof, wind worrying every joint and seam."
      - "A flash from outside paints the room white for an instant."
```

`overcast.yaml`:
```yaml
weather: overcast
outdoor:
  default:
    - "The cloud ceiling hangs low and unbroken, gray from edge to edge."
    - "A dull, even light flattens every shadow."
    - "The air sits still and heavy, waiting on weather that hasn't decided."
indoor:
  default:
    mild: []
    strong:
      - "The light through every crack and window is gray and listless."
```

`fog.yaml`:
```yaml
weather: fog
outdoor:
  default:
    - "The fog drifts in slow banks, swallowing shapes a few paces out."
    - "Sound arrives strangely through the murk — close, then suddenly far."
    - "Beads of mist settle cold on your skin and hair."
  swamp:
    - "The fog lies thick on the water, moving only when something moves it."
  city:
    - "Lamplight and doorways become dim halos in the drifting murk."
indoor:
  default:
    mild: []
    strong:
      - "Mist curls in at the threshold and thins to nothing in the warmth."
```

`snow.yaml`:
```yaml
weather: snow
outdoor:
  default:
    - "Snow sifts down in fat, unhurried flakes."
    - "The world's sounds arrive muffled, wrapped in falling snow."
    - "Snow gathers in the creases of your clothes and melts to cold water."
  forest:
    - "Loaded boughs shed sudden sleeves of snow with a soft thump."
indoor:
  default:
    mild: []
    strong:
      - "Snow whispers against the windows, piling soft in the corners outside."
```

`blizzard.yaml`:
```yaml
weather: blizzard
outdoor:
  default:
    - "The wind drives the snow level, scouring every exposed inch of skin."
    - "The world whites out; up and down are a rumor."
    - "Cold knifes through every seam in your clothing."
indoor:
  default:
    mild:
      - "Wind keens around the corners of the building."
    strong:
      - "The blizzard howls against the walls, packing snow into every crack."
      - "Something on the roof strains and thrums with each long gust."
```

`dust.yaml`:
```yaml
weather: dust
outdoor:
  default:
    - "Wind-borne grit rasps across everything, hissing like dry rain."
    - "You squint against the stinging haze; the horizon is a brown smear."
    - "Dust finds your teeth, your eyes, the inside of your collar."
indoor:
  default:
    mild:
      - "Fine dust sifts down from the rafters with each gust outside."
    strong:
      - "Grit hisses against the walls; the air inside tastes of dry earth."
```

`heatwave.yaml`:
```yaml
weather: heatwave
outdoor:
  default:
    - "Heat shimmers up off the ground in wavering curtains."
    - "Sweat dries to salt almost before it forms."
    - "The air is an oven's breath; even the wind has given up."
  city:
    - "The stones radiate stored heat like a banked forge."
indoor:
  default:
    mild: []
    strong:
      - "The trapped air hangs hot and motionless, thick as wool."
```

- [ ] **Step 2: Validate against the parser**

Add (or extend) a shipped-data test in `modules/weather/content/moduledata_test.go` pointed at the real tree:

```go
// Validates the shipped DOGMud weather data parses and obeys the
// severe-types-always-audible-indoors rule.
func TestShippedDogmudEmoteTables(t *testing.T) {
	fsys := os.DirFS("../../../_datafiles/world/dogmud")
	tables, err := LoadEmotes(fsys, "weather/emotes")
	if err != nil {
		t.Fatalf("LoadEmotes: %v", err)
	}
	if len(tables) != 8 {
		t.Fatalf("expected 8 emote tables, got %d", len(tables))
	}
	for _, severe := range []sim.WeatherType{"storm", "blizzard", "dust"} {
		pool := tables[severe].Indoor["default"]
		if len(pool.Strong) == 0 {
			t.Errorf("%s must have non-empty strong indoor pool", severe)
		}
	}
}
```

Run: `go test ./modules/weather/content/ -run TestShippedDogmud -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/weather/emotes/ modules/weather/content/
git commit -m "content(weather): DOGMud-voiced emote tables with intensity-banded indoor pools"
```

---

### Task 9: Helpfile + crawler skip-list check + boot smoke

**Files:**
- Create: `_datafiles/world/dogmud/templates/help/weather.template`
- Verify: crawler instance-zone skip patterns vs DOGMud zone names
- Verify: `MutatorList.Remove` decayintoid parity (Task 6 specs carry no decayintoid, so the gotcha is moot — confirm specs only)

- [ ] **Step 1: Write the helpfile**

Create `_datafiles/world/dogmud/templates/help/weather.template` (study `time.template` or another short command helpfile in the same dir first and mirror its ansi header conventions exactly):

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">weather</ansi>

The <ansi fg="command">weather</ansi> command reports the current weather where you stand,
including any weather system moving through the area.

<ansi fg="yellow">Usage: </ansi>

  <ansi fg="command">weather</ansi> - Report the local weather conditions.

Weather moves across the world as fronts — a storm born over open water
may roll inland, batter the coast, and spend itself against the
mountains. Watch room descriptions, listen for it on the roof, and plan
your travels accordingly.
```

(Admin subcommands are intentionally undocumented in the player helpfile; `weather` with no permission shows local weather only.)

- [ ] **Step 2: Check the crawler skip patterns against real zone names**

```bash
ls _datafiles/world/dogmud/rooms/
grep -rn "instance_\|ephemeral_" modules/weather/crawler/build.go
```

Compare the crawler's skip patterns to DOGMud's instanced-zone naming (DOGMud has instance zones from the buy-in instance system — find how their zone names are generated, e.g. grep `instance` in `internal/rooms/roommanager.go`). If DOGMud instance zones don't match `instance_*`/`ephemeral_*`, extend the crawler's skip list (in the vendored copy) with the DOGMud pattern and note it in the commit.

- [ ] **Step 3: Full test suite + boot smoke**

```bash
go build ./...
go test ./modules/weather/... ./internal/rooms/ ./internal/mutators/
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . 2>&1 | tee /tmp/weather-boot.log | grep -E "Weather:|mutators.LoadDataFiles|panic"
```

Expected boot lines: `mutators.LoadDataFiles() loadedCount=14`, `Weather: built geography graph zones=N edges=M components=K` (N = DOGMud zone count), `Weather: fresh simulation state seed=...`, `Weather: buffs disabled by config`. No panics. Second boot: `Weather: loaded geography cache` + `Weather: restored simulation state`.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/templates/help/weather.template modules/weather
git commit -m "feat(weather): player helpfile; crawler skip-list verified against DOGMud zones"
```

---

### Task 10: Live smoke test (manual, with the playtest harness optional)

**Files:** none (verification)

- [ ] **Step 1: Boot and force weather**

Boot locally (instance saves already wiped in Task 9). Log in with the admin account. Run:

```
weather status
weather zones
weather spawn storm <current zone> 0.9
```

- [ ] **Step 2: Verify outdoor rendering**

In an outdoor room of that zone: `look` shows `(storm-wracked)` in the room name, the storm description line, the alert banner. Verify `weather` reports the front with felt intensity.

- [ ] **Step 3: Verify indoor muting**

Walk into a `house`/`fort`/`cave` biome room in the same zone: `look` shows NO weather tag/description/alert. Wait up to ~25 rounds: a `strong` indoor line ("Thunder rolls through the walls...") arrives. Then `weather clear`, confirm tags drop, and spawn a weak front (`weather spawn rain <zone> 0.2`): the indoor room must stay silent through 2+ emote cycles while outdoor rooms still see `(raining)` + outdoor lines.

- [ ] **Step 4: Verify persistence**

`weather spawn snow <zone> 0.7`, shut the server down cleanly, reboot, run `weather fronts` — the snow front survives; outdoor rooms re-render `(snowing)` within one tick (boot reconcile).

- [ ] **Step 5: Record results**

Append findings (pass/fail per step, any oddities) to the PR/commit notes. Fix anything broken before Task 11.

---

### Task 11: PATCH_NOTES + merge

**Files:**
- Modify: `PATCH_NOTES.md`

- [ ] **Step 1: Add the patch notes entry**

Match the file's existing format (read the top entry first). Content: dated 2026-06-10 — "Weather: living weather systems now travel the world. Fronts form over the terrain that feeds them, roll zone to zone, and die against the terrain that starves them. Rooms show it (name tags, descriptions, dimmed light in severe weather), and you'll hear it indoors when it's heavy enough. Try the `weather` command."

- [ ] **Step 2: Final verification**

```bash
go build ./... && go test ./modules/weather/... ./internal/rooms/ ./internal/mutators/
```

Expected: clean build, all green.

- [ ] **Step 3: Merge to master (no prod push — that's end-of-day, bundled per SOP)**

```bash
git checkout master
git merge --no-ff feature/weather-module-backport -m "Merge feature/weather-module-backport: living weather systems (DOGMud-native backport)"
```

Then re-run the Task 9 boot smoke once on master.

---

## Self-review notes

- Spec coverage: code layout (T0), engine adaptation/SendText/config (T3), indoor model both halves (T1+T2 render filter, T4+T5 banded emotes), 17 climates (T7), 8 specs no-buffs (T6), emotes DOGMud voice (T8), commands+helpfile (T3+T9), verification items 1–5 (T9 step 2 = item 1; T0 step 4 surfaces item 2 plugin-storage drift as compile/runtime errors with `WriteBytes`/`ReadBytes` confirmed present; item 3 moot — no decayintoid in shipped specs; item 4 resolved — colors chosen from the verified palette; item 5 = T10 admin `weather graph` spot-check), testing/rollout (T9–T11).
- Types consistent: `IndoorPool{Mild,Strong}`, `Pick(weather, biome, indoor, felt, roll)`, `EmitAmbient(g, fronts, simCfg, weather, tables, roll)` used identically in T4/T5/T8.
- Known judgment points left to the implementer, by design: exact test-seam adaptation in T2/T4 (existing helpers vary), `usercommands.UserCommand` signature match in T3 step 5.
