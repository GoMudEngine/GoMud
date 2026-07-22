# Weather UX Polish Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Polish the weather system for 1.0 — fix ASCII-mode glyph mojibake, make ambient emotes intensity-scaled, slow the weather tempo, fill indoor prose, and add a calm-cold `frost` condition.

**Architecture:** Six independent threads. Thread 1 is a central display-layer fix in `util.ConvertToAscii`. Thread 2 adds an intensity-gated emit chance inside the existing `EmitAmbient` pass, driven by a pure `emitChance` helper. Thread 3 & 4 are content (emote YAML). Thread 5 is config-only. Thread 6 recolors low-contrast readable ANSI aliases against the dark web terminal. Each task is a self-contained, committable unit. Weather is atmospheric only (`BuffsEnabled: false`), so no mechanics change.

**Tech Stack:** Go (GoMud fork), YAML data files, `go test`, config in `_datafiles/config.yaml`.

**Spec:** `docs/superpowers/specs/2026-07-22-weather-ux-polish-design.md`

---

## File Structure

- `internal/util/util.go` — widen `unicodeToAscii` to `map[rune]string`; extend the fallback table. (Thread 1)
- `internal/util/util_test.go` — extend `TestConvertToAscii` table. (Thread 1)
- `internal/gametime/gametime.go` — drop the U+FE0F from the prompt sun glyph. (Thread 1)
- `modules/weather/weather_config.go` — two new chance knobs + clamps; bump emote cadence default. (Thread 2)
- `modules/weather/weather_config_test.go` — cover the new knobs. (Thread 2)
- `modules/weather/engine/emotes.go` — `emitChance` helper + intensity gate + signature. (Thread 2)
- `modules/weather/engine/emotes_test.go` — new; unit-test `emitChance`. (Thread 2)
- `modules/weather/weather.go` — pass the new config values at the one call site. (Thread 2)
- `_datafiles/config.yaml` — new/changed weather knobs. (Threads 2 & 5)
- `_datafiles/world/dogmud/weather/emotes/*.yaml` — fill indoor `strong`. (Thread 3)
- `modules/weather/sim/climate.go` — add `frost` to five cold climate maps. (Thread 4)
- `_datafiles/world/dogmud/weather/emotes/frost.yaml` — new emote table. (Thread 4)
- `modules/weather/content/shipped_emotes_test.go` — table count 8 → 9. (Thread 4)
- `_datafiles/world/dogmud/ansi-aliases.yaml` — recolor 8 low-contrast readable aliases. (Thread 6)

---

## Task 1: Charset-safe decorative glyphs (Thread 1)

**Files:**
- Modify: `internal/util/util.go:971-1020` (`ConvertToAscii` + `unicodeToAscii`)
- Test: `internal/util/util_test.go:1187` (extend `TestConvertToAscii`)
- Modify: `internal/gametime/gametime.go:74`

- [ ] **Step 1: Add failing test cases** to the `tests` table in `TestConvertToAscii` (`internal/util/util_test.go`), immediately after the `{"motd chars", ...}` entry (line 1206):

```go
		{"sun glyph with variation selector", "☀️", "*"},
		{"moon glyph", "☾", "("},
		{"weather glyphs", "⚡❄", "!*"},
		{"map glyphs", "▲▼≈⌂", "^v~#"},
		{"unmapped high rune passthrough", "café", "café"},
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/util/ -run TestConvertToAscii -v`
Expected: FAIL — the sun case yields `"*"` plus mojibake bytes from the undropped U+FE0F, and `☾ ⚡ ❄ ▲ ▼ ≈ ⌂` pass through unmapped.

- [ ] **Step 3: Widen the map type and the conversion loop.** In `internal/util/util.go`, change the loop body inside `ConvertToAscii` (currently around line 987-991):

```go
	for _, r := range s {
		if repl, ok := unicodeToAscii[r]; ok {
			b.WriteString(repl)
		} else {
			b.WriteRune(r)
		}
	}
```

Then change the map declaration from `var unicodeToAscii = map[rune]byte{` to `var unicodeToAscii = map[rune]string{` and convert **every** existing value from a byte literal to a string literal (`'-'` → `"-"`, etc.). The full replacement map:

```go
// unicodeToAscii maps decorative Unicode runes to ASCII string equivalents.
// String values (not bytes) so a rune can drop to "" (e.g. variation selectors)
// or expand to multiple chars. Applied per-recipient in AsciiMode (see
// ConvertToAscii); never alters output for UTF-8-capable clients.
var unicodeToAscii = map[rune]string{
	// Box-drawing: light
	'─': "-", '│': "|",
	'┌': "+", '┐': "+", '└': "+", '┘': "+",
	'├': "+", '┤': "+", '┬': "+", '┴': "+", '┼': "+",
	// Box-drawing: double
	'═': "=", '║': "|",
	'╔': "+", '╗': "+", '╚': "+", '╝': "+",
	'╠': "+", '╣': "+", '╦': "+", '╩': "+", '╬': "+",
	// Box-drawing: mixed single/double
	'╒': "+", '╕': "+", '╘': "+", '╛': "+",
	'╞': "+", '╡': "+", '╤': "+", '╧': "+", '╪': "+",
	'╓': "+", '╖': "+", '╙': "+", '╜': "+",
	'╟': "+", '╢': "+", '╥': "+", '╨': "+", '╫': "+",
	// Block elements
	'█': "#", '▓': "#", '▒': ":", '░': ".",
	'▄': "-", '▀': "_", '▌': "|", '▐': "|",
	// Bullet / misc
	'•': "*",
	// Diagonal lines
	'╲': "\\", '╱': "/",
	// Sun / moon / weather (prompt + splash glyphs)
	'☀': "*", '☾': "(", '☽': ")",
	'\uFE0F': "", // emoji variation selector — drop (was the "trailing bytes" leak)
	'\uFE0E': "", // text-presentation selector — drop too, for safety
	'⚡': "!", '❄': "*", '✦': "*", '✧': "*", '❆': "*", '❅': "*",
	// Map / directional
	'▲': "^", '▼': "v", '△': "^", '▽': "v",
	'≈': "~", '⌂': "#", '◆': "*", '●': "o", '○': "o",
}
```

> **Implementer note:** the two "variation selector" map keys shown above are
> INVISIBLE characters and must be written as explicit Go rune escapes, NOT the
> raw glyphs. Use exactly these two entries in the map:
>
> ```go
> 	'\uFE0F': "", // emoji variation selector — the rune trailing off "☀️"
> 	'\uFE0E': "", // text-presentation selector — drop too, for safety
> ```
>
> The U+FE0F trailing the sun glyph is what produced the "trailing bytes"
> mojibake in ASCII mode. Delete the invisible-literal lines from the map block
> above and use these escaped forms instead.

- [ ] **Step 4: Fix the prompt glyph at its source.** In `internal/gametime/gametime.go:74`, remove the trailing U+FE0F variation selector so the emitted glyph is a single rune:

```go
		return fmt.Sprintf(`<ansi fg="%s">☀</ansi>`, dayNight)
```

(The night branch at line 72 already emits a bare `☾` — leave it; it is now in the table.)

- [ ] **Step 5: Run the test to verify it passes**

Run: `go test ./internal/util/ -run TestConvertToAscii -v`
Expected: PASS (all cases, including the pre-existing ones).

- [ ] **Step 6: Discovery sweep for stragglers.** Find any other decorative runes that reach output through the same path and add fallbacks if missing:

Run: `grep -rhoP '[^\x00-\x7F]' internal/splash/*.go internal/usercommands/map*.go modules/gmcp/*.go 2>/dev/null | sort -u`
For each rune returned that is a decorative/symbol glyph (NOT letters of prose), add a `'X': "y",` entry to the map. If the splash scenes are already pure-ASCII (the de-pixelation redesign), this returns nothing new — that is fine. Do not add mappings for accented letters or CJK; only box/symbol/emoji glyphs.

- [ ] **Step 7: Build + commit**

Run: `go build ./... && go test ./internal/util/ ./internal/gametime/`
Expected: build clean, tests pass.

```bash
git add internal/util/util.go internal/util/util_test.go internal/gametime/gametime.go
git commit -m "fix(charset): ASCII-mode fallbacks for prompt/weather/map glyphs

Widen unicodeToAscii to map[rune]string so multi-rune emoji convert
cleanly and variation selectors drop. Fixes the prompt sun/moon glyph
mojibake reported in ASCII mode; drops U+FE0F at the gametime source.

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: Intensity-scaled emote cadence (Thread 2)

**Files:**
- Modify: `modules/weather/weather_config.go:21-35` (Config), `:96-127` (buildConfig)
- Test: `modules/weather/weather_config_test.go`
- Create: `modules/weather/engine/emotes_test.go`
- Modify: `modules/weather/engine/emotes.go:26-64`
- Modify: `modules/weather/weather.go:89`
- Modify: `_datafiles/config.yaml` (weather block)

- [ ] **Step 1: Write the failing `emitChance` unit test.** Create `modules/weather/engine/emotes_test.go`:

```go
package engine

import "testing"

func TestEmitChance(t *testing.T) {
	tests := []struct {
		name             string
		mild, strong     int
		felt             float64
		want             int
	}{
		{"calm floor at felt 0", 30, 100, 0.0, 30},
		{"severe ceiling at felt 1", 30, 100, 1.0, 100},
		{"linear midpoint", 30, 100, 0.5, 65},
		{"negative felt clamps to floor", 30, 100, -0.3, 30},
		{"above-one felt clamps to ceiling", 30, 100, 1.7, 100},
		{"flat when mild==strong", 50, 50, 0.4, 50},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := emitChance(tt.mild, tt.strong, tt.felt); got != tt.want {
				t.Errorf("emitChance(%d,%d,%v) = %d, want %d",
					tt.mild, tt.strong, tt.felt, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./modules/weather/engine/ -run TestEmitChance -v`
Expected: FAIL — `emitChance` undefined.

- [ ] **Step 3: Add the `emitChance` helper** at the top of `modules/weather/engine/emotes.go` (after the imports, before `EmitAmbient`):

```go
// emitChance maps a zone's felt weather intensity (0..1) to the percent chance
// that an ambient weather line fires this pass: mildPct at felt<=0 rising
// linearly to strongPct at felt>=1. Calm weather whispers; severe weather
// speaks steadily. The felt scale is clamp01 (see sim.Coverage.Effective).
func emitChance(mildPct, strongPct int, felt float64) int {
	if felt < 0 {
		felt = 0
	} else if felt > 1 {
		felt = 1
	}
	return mildPct + int(float64(strongPct-mildPct)*felt)
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./modules/weather/engine/ -run TestEmitChance -v`
Expected: PASS.

- [ ] **Step 5: Wire the gate into `EmitAmbient`.** Change its signature (add `mildChancePct, strongChancePct int` before `roll`):

```go
func EmitAmbient(g *sim.Graph, fronts []sim.Front, simCfg sim.Config,
	weather map[sim.ZoneId]sim.WeatherType, zoneSeasons map[sim.ZoneId]seasons.ZoneSeason,
	tables content.Tables, seasonal content.SeasonalTables,
	mildChancePct, strongChancePct int, roll func(int) int) int {
```

Inside the "Weather wins" branch, after `felt[room.Zone] = f` is resolved and before the `season`/`Pick` block, insert the gate (a skipped pass is silence — non-clear weather still owns the room, so `continue`):

```go
			if roll(100) >= emitChance(mildChancePct, strongChancePct, f) {
				continue
			}
```

- [ ] **Step 6: Update the call site.** In `modules/weather/weather.go:89`:

```go
		engine.EmitAmbient(m.graph, m.state.Fronts, m.simCfg, m.state.Weather, m.zoneSeasons, m.tables, m.seasonalTables, m.cfg.EmoteMildChancePct, m.cfg.EmoteStrongChancePct, util.Rand)
```

- [ ] **Step 7: Write the failing config test.** Add to `modules/weather/weather_config_test.go`:

```go
func TestBuildConfig_EmoteChanceKnobs(t *testing.T) {
	// Defaults when unset.
	c := buildConfig(func(string) any { return nil })
	if c.EmoteMildChancePct != 30 || c.EmoteStrongChancePct != 100 {
		t.Errorf("defaults: got mild=%d strong=%d, want 30/100",
			c.EmoteMildChancePct, c.EmoteStrongChancePct)
	}
	if c.EmoteEveryRounds != 24 {
		t.Errorf("EmoteEveryRounds default: got %d, want 24", c.EmoteEveryRounds)
	}
	// Out-of-range clamps to 0..100.
	over := map[string]any{"EmoteMildChancePct": 250, "EmoteStrongChancePct": -40}
	c = buildConfig(func(k string) any { return over[k] })
	if c.EmoteMildChancePct != 100 || c.EmoteStrongChancePct != 0 {
		t.Errorf("clamp: got mild=%d strong=%d, want 100/0",
			c.EmoteMildChancePct, c.EmoteStrongChancePct)
	}
}
```

- [ ] **Step 8: Run to verify it fails**

Run: `go test ./modules/weather/ -run TestBuildConfig_EmoteChanceKnobs -v`
Expected: FAIL — fields undefined / wrong defaults.

- [ ] **Step 9: Add the config fields, defaults, and clamps.** In `modules/weather/weather_config.go`, add to the `Config` struct (after `EmoteEveryRounds`, line 31):

```go
	EmoteMildChancePct   int // ambient emit chance (%) at felt intensity 0
	EmoteStrongChancePct int // ambient emit chance (%) at felt intensity 1
```

In `buildConfig`, change the `EmoteEveryRounds` default from `20` to `24` (line 106) and add the two new resolves in the struct literal:

```go
		EmoteEveryRounds:     intOr(get("EmoteEveryRounds"), 24),
		EmoteMildChancePct:   intOr(get("EmoteMildChancePct"), 30),
		EmoteStrongChancePct: intOr(get("EmoteStrongChancePct"), 100),
```

After the existing clamps (after the `SpawnRateScale < 0` clamp, ~line 122), add:

```go
	if c.EmoteMildChancePct < 0 {
		c.EmoteMildChancePct = 0
	}
	if c.EmoteMildChancePct > 100 {
		c.EmoteMildChancePct = 100
	}
	if c.EmoteStrongChancePct < 0 {
		c.EmoteStrongChancePct = 0
	}
	if c.EmoteStrongChancePct > 100 {
		c.EmoteStrongChancePct = 100
	}
```

- [ ] **Step 10: Run to verify config test + gate build**

Run: `go build ./... && go test ./modules/weather/... -run 'TestBuildConfig_EmoteChanceKnobs|TestEmitChance' -v`
Expected: build clean, both pass.

- [ ] **Step 11: Update `_datafiles/config.yaml`** weather block — set `EmoteEveryRounds: 24` and add the two knobs (leave other keys; `TickEveryGameHours`/`SpawnRateScale` handled in Task 3):

```yaml
    EmoteMode: module
    EmoteEveryRounds: 24
    EmoteMildChancePct: 30
    EmoteStrongChancePct: 100
```

- [ ] **Step 12: Commit**

```bash
git add modules/weather/weather_config.go modules/weather/weather_config_test.go modules/weather/engine/emotes.go modules/weather/engine/emotes_test.go modules/weather/weather.go _datafiles/config.yaml
git commit -m "feat(weather): intensity-scaled ambient emote cadence

Gate each per-room weather emote on a felt-intensity-derived chance
(mild 30% -> strong 100%) via a pure emitChance helper; calm skies
whisper, storms speak steadily. Base cadence 20->24 rounds.

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: Weather tempo (Thread 5, config-only)

**Files:**
- Modify: `_datafiles/config.yaml` (weather block)

- [ ] **Step 1: Slow the tempo.** In the `_datafiles/config.yaml` weather block, change:

```yaml
    TickEveryGameHours: 8
    SpawnRateScale: 0.7
```

(from `1` and `1.0`). Add a brief inline comment noting the rationale: weather steps every ~20 min real-time and spawns are thinned so weather reads as a rare, stable atmospheric event; raise `SpawnRateScale` toward 1.0 first if playtest finds weather too sparse.

- [ ] **Step 2: Verify the sim picks it up (no code path broken)**

Run: `go build ./...`
Expected: clean (config-only change; the boot smoke in Task 6 confirms runtime).

- [ ] **Step 3: Commit**

```bash
git add _datafiles/config.yaml
git commit -m "tune(weather): slow tempo (tick 1->8) and thin spawns (1.0->0.7)

Weather now reads as a rare, stable atmospheric event rather than
minute-to-minute flicker. Both are playtest-tunable; spawn rate is the
first dial-back if weather feels too sparse.

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 4: Indoor prose coverage (Thread 3, content)

**Files:**
- Modify: `_datafiles/world/dogmud/weather/emotes/{overcast,fog,rain,dust,heatwave,snow,blizzard}.yaml`

Goal: every non-calm type reaches **≥3** indoor `default.strong` lines (storm already has 3). Keep `mild` as authored (empty = intentional silence). All lines ≤80 chars; describe what the weather does to the *shelter*, never restate the outdoor line.

- [ ] **Step 1: overcast.yaml** — replace the `strong:` list under `indoor.default` with:

```yaml
    strong:
      - "The light through every crack and window is gray and listless."
      - "Even indoors the day feels dimmed, colors gone muddy and tired."
      - "No shadow moves across the floor; the flat gray light never shifts."
```

- [ ] **Step 2: fog.yaml** — replace the `strong:` list under `indoor.default` with:

```yaml
    strong:
      - "Mist curls in at the threshold and thins to nothing in the warmth."
      - "Damp creeps under the door and beads on the cold windowpanes."
      - "The world beyond the glass has simply dissolved into pale nothing."
```

- [ ] **Step 3: rain.yaml** — append one line to the existing `strong:` list under `indoor.default` (bringing it to 3):

```yaml
      - "A gust flings a handful of rain against the panes, then relents."
```

- [ ] **Step 4: dust.yaml** — replace the `strong:` list under `indoor.default` with (keep the existing `mild:` line):

```yaml
    strong:
      - "Grit hisses against the walls; the air inside tastes of dry earth."
      - "A gritty film settles over everything no matter how tight the shutters."
      - "Each breath carries the faint, mineral taste of the storm outside."
```

- [ ] **Step 5: heatwave.yaml** — replace the `strong:` list under `indoor.default` with:

```yaml
    strong:
      - "The trapped air hangs hot and motionless, thick as wool."
      - "The walls themselves give off warmth, offering no relief at all."
      - "Time seems to thicken; every small motion costs more than it should."
```

- [ ] **Step 6: snow.yaml** — replace the `strong:` list under `indoor.default` with:

```yaml
    strong:
      - "Snow whispers against the windows, piling soft in the corners outside."
      - "The hush outside presses against the walls, deep and close and complete."
      - "Cold seeps up through the floorboards; the fire seems to give less heat."
```

- [ ] **Step 7: blizzard.yaml** — append one line to the existing `strong:` list under `indoor.default` (bringing it to 3):

```yaml
      - "Frost creeps across the inside of the glass in feathered white ferns."
```

- [ ] **Step 8: Verify all lines parse and obey the ≤80-char rule**

Run: `go test ./modules/weather/content/ -run TestShippedDogmudEmoteTables -v`
Expected: PASS (parse OK, no line > 80 chars). NOTE: this still expects **8** tables here — frost is added in Task 5, which updates the count. Run Task 4 before Task 5, or the count assertion is unaffected either way at this step.

- [ ] **Step 9: Commit**

```bash
git add _datafiles/world/dogmud/weather/emotes/
git commit -m "content(weather): fill indoor strong prose to >=3 lines per type

Bring overcast/fog/rain/dust/heatwave/snow/blizzard indoor strong pools
to three lines each (storm already had three). Mild pools left silent by
design (weather below the felt threshold is not heard through walls).

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 5: New `frost` weather type (Thread 4)

**Files:**
- Create: `_datafiles/world/dogmud/weather/emotes/frost.yaml`
- Modify: `modules/weather/sim/climate.go` (5 cold entries)
- Modify: `modules/weather/content/shipped_emotes_test.go:19`

- [ ] **Step 1: Author `frost.yaml`.** Create `_datafiles/world/dogmud/weather/emotes/frost.yaml`:

```yaml
weather: frost
outdoor:
  default:
    - "A hard frost has seized the world; every surface wears a fur of rime."
    - "The cold is absolute and windless, and your breath hangs in white clouds."
    - "Ice-ferns spread across stone and puddle, crackling faintly as they grow."
    - "A low freezing mist clings to the ground, riming all that it touches."
  forest:
    - "Every twig and needle is sheathed in glittering white, still as glass."
    - "The frozen wood is silent but for the tick of ice tightening its grip."
  city:
    - "Rime whitens the eaves and doorframes; the streets ring hard underfoot."
    - "Freezing mist pools between the buildings, haloing each cold lantern."
  water:
    - "A skin of ice creeps out from the banks, thin and clear as held breath."
    - "Frozen mist drifts across the still water, leaving rime on the reeds."
indoor:
  default:
    mild: []
    strong:
      - "Cold seeps through the walls; frost-ferns bloom on the inner glass."
      - "Your breath fogs indoors, and the very nails in the wood feel like ice."
      - "The quiet outside is glassy and total, the cold pressing at every seam."
```

- [ ] **Step 2: Add `frost` to the cold climate maps.** In `modules/weather/sim/climate.go`, add a `"frost"` weight to each of these five `Weather:` maps (leave all other fields unchanged):

  - `mountain` (line ~88): `map[WeatherType]float64{"overcast": 4, "snow": 4, "storm": 2, "fog": 3, "frost": 2}`
  - `tundra` (line ~100): `map[WeatherType]float64{"clear": 5, "overcast": 4, "snow": 6, "blizzard": 2, "fog": 2, "frost": 3}`
  - `mountains` (line ~123): `map[WeatherType]float64{"overcast": 4, "snow": 4, "storm": 2, "fog": 3, "frost": 2}`
  - `cliffs` (line ~129): `map[WeatherType]float64{"clear": 3, "overcast": 4, "storm": 3, "fog": 3, "frost": 2}`
  - `snow` (line ~135): `map[WeatherType]float64{"clear": 5, "overcast": 4, "snow": 6, "blizzard": 2, "fog": 2, "frost": 3}`

  Do NOT add frost to temperate/warm biomes (plains, forest, land, farmland, city, desert, swamp, ocean, shore, water, road, jungle) — the sim cannot yet season-gate weather, so a temperate frost would appear in summer.

- [ ] **Step 3: Update the shipped-table count.** In `modules/weather/content/shipped_emotes_test.go:19-20`, change the expected count:

```go
	if len(tables) != 9 {
		t.Fatalf("expected 9 emote tables, got %d", len(tables))
	}
```

- [ ] **Step 4: Run the content + sim tests**

Run: `go test ./modules/weather/content/ ./modules/weather/sim/ -v`
Expected: PASS — 9 tables, frost lines ≤80 chars, frost outdoor default non-empty; climate maps still valid.

- [ ] **Step 5: Build**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 6: Commit**

```bash
git add _datafiles/world/dogmud/weather/emotes/frost.yaml modules/weather/sim/climate.go modules/weather/content/shipped_emotes_test.go
git commit -m "feat(weather): add calm-cold 'frost' condition

A still, bitterly cold condition blending hoarfrost rime and low
freezing mist. Added to cold climates only (tundra/snow/mountain/cliffs)
where a hard freeze is plausible year-round, sidestepping the missing
seasonal weather bias. No splash (frost is not a severe onset).

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 6: Category color contrast recolor (Thread 6)

**Files:**
- Modify: `_datafiles/world/dogmud/ansi-aliases.yaml`

Recolor only the low-contrast **readable-text** aliases (verified ≥4.5:1 against
the web terminal bg `#0c0c0d`). Leave intentional de-emphasis (the `8`/237/240/
`*-downed` families, `suggested-text`, `item-nothing`) and the already-passing
`weather` (75) / `time-of-day` (179) untouched.

- [ ] **Step 1: Apply the recolors.** In `_datafiles/world/dogmud/ansi-aliases.yaml`, change exactly these values (do NOT touch other aliases that share the old numeric value — e.g. `mobname-downed: 124` stays 124, downed mobs are meant to be faint):

  - `night: 19` → `night: 153`   (the `☾` prompt moon glyph; 1.5→13.0:1)
  - `holy: 21` → `holy: 111`   (2.3→8.9:1)
  - `zone: 124` → `zone: 167`   (2.6→5.3:1)
  - `room-zone: 124` → `room-zone: 167`   (2.6→5.3:1)
  - `username-aggro: 124` → `username-aggro: 196`   (2.6→4.9:1; aggro should pop)
  - `spell-harmful: 124` → `spell-harmful: 203`   (2.6→6.6:1)
  - `room-title: 128` → `room-title: 170`   (3.5→6.1:1)
  - `item-cursed: 54` → `item-cursed: 133`   (1.7→4.7:1)

  Note there are two `room-zone` entries in the file (lines ~36 and ~114, both `124`) — change both, and the `zone: 124` at line ~36. Search the file for `: 124` and `: 128`/`: 54`/`: 21`/`: 19` to confirm you catch exactly the eight aliases above and nothing else.

- [ ] **Step 2: Verify no readable alias remains below 4.5:1.** Re-run the audit script (from the plan header of this task set — reproduced here) and confirm the eight aliases now clear 4.5 and nothing new regressed:

```bash
python3 - <<'PY'
import re
def xr(i):
    if i<=231:
        i-=16; s=[0,95,135,175,215,255]; return (s[i//36%6],s[i//6%6],s[i%6])
    g=8+(i-232)*10; return (g,g,g)
def lum(c):
    def f(x):
        x/=255; return x/12.92 if x<=0.03928 else ((x+0.055)/1.055)**2.4
    r,g,b=c; return .2126*f(r)+.7152*f(g)+.0722*f(b)
BG=lum((12,12,13))
def con(i):
    L=lum(xr(i)); a,b=max(L,BG),min(L,BG); return (a+.05)/(b+.05)
for a,i in [("night",153),("holy",111),("zone",167),("room-zone",167),
            ("username-aggro",196),("spell-harmful",203),("room-title",170),
            ("item-cursed",133)]:
    print(f"{a:16s} {i:3d} {con(i):4.1f}:1", "OK" if con(i)>=4.5 else "LOW")
PY
```
Expected: every line prints `OK`.

- [ ] **Step 3: Build sanity (YAML loads)**

Run: `go build ./...`
Expected: clean. (Runtime load of the alias file is confirmed by the boot-smoke in Task 7.)

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/ansi-aliases.yaml
git commit -m "fix(color): raise contrast on low-legibility readable aliases

Recolor night(moon)/holy/zone/room-zone/username-aggro/spell-harmful/
room-title/item-cursed from sub-4.5:1 to >=4.5:1 against the web dark
terminal (also helps telnet). Intentional de-emphasis colors and the
already-legible weather/time-of-day left unchanged.

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 7: Boot smoke + adversarial playtest (content gate — REQUIRED)

**Files:** none (verification only)

- [ ] **Step 1: Full test suite + build**

Run: `go build ./... && go test ./internal/util/ ./internal/gametime/ ./modules/weather/...`
Expected: all green.

- [ ] **Step 2: Wipe instance saves (SOP) and boot-smoke**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* \
       _datafiles/world/dogmud/rooms.instances/*
go run .
```
Expected: server boots past data-file loading with NO panic; watch for the weather module loading its climate + emote tables (frost.yaml among them) cleanly. Kill the server after confirming a clean start.

- [ ] **Step 3: Adversarial in-game playtest (CLAUDE.md content gate).** Run `/playtest local bug-finder` (or a weather-focused goals file) with an explicitly critical mandate. Spawn a fresh character, `set charset ascii`, and drive the weather flow:
  1. Confirm the prompt day/night glyph and any map glyphs render as clean ASCII (`*`/`(`/`^`/`~`/`#`) — NO mojibake or trailing bytes.
  2. `weather spawn storm <zone> 1.0` and a calm one (`weather spawn overcast <zone> 0.2`); over several minutes confirm the storm emotes steadily while the overcast only whispers occasionally (intensity scaling).
  3. Confirm weather *changes* feel slow/systemic (Thread 5) — and specifically that weather still appears often enough to notice at all; if it reads dead, note it (first dial-back = raise `SpawnRateScale`).
  4. Enter an indoor room during weather; confirm indoor prose lands and reads distinctly from outdoor.
  5. `weather spawn frost <cold-zone> 0.8`; confirm frost reads distinctly from snow and fog (still, crystalline, freezing mist), outdoor + indoor.
  6. On the WEB client (dark terminal), spot-check the Thread 6 recolors: a zone header, the night `☾` prompt moon glyph, an aggro mob name, a harmful spell, a room title, a cursed item — all now clearly legible; and confirm de-emphasis text (system messages, dead/downed mobs, secret exits) still reads as intentionally subdued, not washed out.
  Read every line as a confused human would. Fix anything it surfaces, re-run if needed, and only then hand to the user.

- [ ] **Step 4: Report.** Summarize the playtest findings + any fixes. Do NOT claim the work done on a clean boot alone.

---

## Self-Review notes

- **Spec coverage:** Thread 1 → Task 1; Thread 2 → Task 2; Thread 3 → Task 4; Thread 4 → Task 5; Thread 5 → Task 3; Thread 6 → Task 6; content gate → Task 7. All six threads + testing covered.
- **Config table (spec) vs plan:** `EmoteMildChancePct` 30, `EmoteStrongChancePct` 100, `EmoteEveryRounds` 24, `TickEveryGameHours` 8, `SpawnRateScale` 0.7 — all present (Tasks 2 & 3).
- **Thread 6 recolors:** 8 aliases (night/holy/zone/room-zone/username-aggro/spell-harmful/room-title/item-cursed), each verified ≥4.5:1 by the Task 6 Step 2 script; de-emphasis and weather/time-of-day deliberately excluded.
- **Type consistency:** `emitChance(mildPct, strongPct int, felt float64) int` defined in Task 2 Step 3, called with the same arg order in Step 5 and tested in Step 1; `EmitAmbient` signature extended once (Step 5) and the sole caller updated (Step 6). `unicodeToAscii` type change (Task 1) has exactly one consumer (`ConvertToAscii`), updated in the same task.
- **Ordering:** Task 4 (count still 8) before or after Task 5 (count → 9) both fine — the count assertion is only touched in Task 5. If executed out of order, run Task 5 Step 3 before re-running the content test.
```
