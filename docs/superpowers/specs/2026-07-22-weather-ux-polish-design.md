# Weather UX Polish (PATH_TO_1.0 §5d, sub-project A) — Design

**Date:** 2026-07-22
**Status:** approved design, pre-plan
**Scope:** presentation/pacing polish on the existing `modules/weather/` system.
No new mechanics (weather is atmospheric — `BuffsEnabled: false`).

## Motivation

Three playtest reports (2026-07-13, 2026-07-18, 2026-07-19) flagged the same
readability defect: the prompt day/night glyph (`☀️`/`☾`) and weather glyphs
(`⚡`/`❄`) survive `set charset ascii` as mojibake + trailing bytes. Beyond that
one bug, the weather system needs pacing and coverage polish before 1.0: ambient
emotes fire at a flat cadence regardless of conditions, weather *changes* too
fast to read as slow atmospheric systems, indoor prose is thin, and there is no
calm-cold condition (cold currently = snow/blizzard precipitation only).

Five independent threads. Each can ship on its own; they share one final
in-game playtest.

---

## Thread 1 — Charset-safe decorative glyphs

**Root cause.** All output routes through `conn.Write` →
`util.ConvertToAscii` (`internal/connections/connectiondetails.go:248`,
`internal/util/util.go:971`), which substitutes decorative runes via
`unicodeToAscii map[rune]byte`. The prompt/weather/map glyphs mojibake because
(a) they are simply absent from the table, and (b) `☀️` is **two** runes —
`☀` U+2600 plus an invisible variation-selector U+FE0F — and a `rune→byte` map
**cannot drop** the selector, so it leaks as raw UTF-8 (the "trailing bytes").

**Fix.**
1. Widen `unicodeToAscii` from `map[rune]byte` to `map[rune]string`. This allows
   a rune to map to `""` (drop, e.g. U+FE0F) or to multi-char output. Update the
   `ConvertToAscii` loop to `WriteString` the mapped value (falling back to the
   raw rune when unmapped). The fast-path high-byte check is unchanged.
2. Curate ASCII fallbacks for the full decorative set that reaches output.
   Enumerate the complete set during implementation (grep the codebase for
   emitted runes ≥ U+0080 in prompt/map/weather paths); known members:
   - Day/night: `☀`→`*`, `☾`→`(`, `☽`→`)`, U+FE0F→`` (drop)
   - Weather: `⚡`→`!`, `❄`→`*` (extend to any others found)
   - Mapper: `▲`→`^`, `▼`→`v`, `≈`→`~`, `⌂`→`#` (extend to the mapper glyph set)
3. Belt-and-suspenders at the source: `internal/gametime/gametime.go:74` emits
   `☀️` with the trailing U+FE0F — emit plain `☀` (no selector). The central
   table still covers any stragglers.

**Boundary.** This is a display-layer substitution only; it never changes what a
UTF-8-capable client receives. Map glyphs are fixed for free because they share
the same `conn.Write` path — no mapper code changes.

**Tests.** Table-driven `ConvertToAscii` tests: single decorative rune → ASCII;
multi-rune emoji (`☀️`) → single ASCII char with the selector dropped; unmapped
rune passes through; the no-high-byte fast path is unaffected.

---

## Thread 2 — Intensity-scaled emote cadence

**Current.** A single global timer (`weather.go:88`, `scheduleEmote`) fires
`engine.EmitAmbient` every `EmoteEveryRounds` (20) rounds ± 25% jitter — a flat
~80s cadence (4s rounds) whether the sky is a calm overcast or a raging storm.

**Approach.** Keep the global pass as the *ceiling*, and gate each per-room
weather emit inside `EmitAmbient` (`engine/emotes.go`) on a probability derived
from that zone's **already-computed felt intensity** `f`
(`covers[0].Effective`, emotes.go:47-54). Calm-but-non-clear weather (low `f`,
e.g. overcast) emits rarely; a storm/blizzard (high `f`) emits near every pass.

**Design.**
- Map `f` → emit chance with two config knobs, linearly interpolated:
  `EmoteMildChancePct` (chance as `f→0`, default **30**) and
  `EmoteStrongChancePct` (chance as `f→1`, default **100**). `chance =
  mild + (strong-mild)·clamp(f,0,1)`.
- Roll against `chance` with the presentation RNG already threaded into
  `EmitAmbient` (`roll`), immediately before `room.SendText`. On a miss, the
  room simply gets no line this pass (the `continue` still applies; no seasonal
  fallback — non-clear weather owns the room).
- Nudge the base cadence up modestly: `EmoteEveryRounds` 20 → **24** so even
  severe weather isn't a metronome.
- The seasonal-ambience branch (calm zones) is unchanged.

**Config-plumbing.** Add `EmoteMildChancePct` / `EmoteStrongChancePct` to
`weather_config.go` `Config` + `buildConfig` (with clamps 0..100) and to the
`config.yaml` weather block. The emit gate needs these two values passed into
`EmitAmbient` (extend its signature, or pass via a small params struct — decide
in the plan; signature extension is consistent with the existing style).

**Tests.** `buildConfig` covers the two new knobs + clamps. An `EmitAmbient`
test asserts that at `f≈0` most passes emit nothing and at `f≈1` nearly all
passes emit (seeded/deterministic `roll`).

---

## Thread 3 — Indoor prose coverage

**Respect the existing design.** `mild`-band indoor emptiness is **intentional
silence** — `StrongFeltThreshold = 0.5` (`content/emotes.go:13`) means light
weather is not felt through walls. We do **not** fill every `mild`.

**Work.** Audit all `_datafiles/world/dogmud/weather/emotes/*.yaml`; the real gap
is weather types with thin or missing indoor **`strong`** coverage (e.g.
`snow.yaml` has a single strong line, empty mild).
- Bring every non-calm type to **≥3** indoor `default.strong` lines.
- Add `city.strong` / `forest.strong` indoor variants where that condition is
  common in those biomes.
- Voice rule: describe what the weather *does to the shelter* (rain drumming,
  draughts, light through windows) — never restate the outdoor line.
- Line-width ≤ 80 chars; no hard numbers; no semicolons/colons in list scalars
  (YAML/command-separator gotchas).

**No engine change** — content only.

---

## Thread 4 — New `frost` weather type

A single calm, bitterly-cold condition blending hard-freeze rime **and** low
freezing mist (design decision: user chose both the hoarfrost and freezing-fog
characters as one type). Distinct mood from snow (precipitation) and plain fog
(not cold): still, crystalline, breath clouding, rime on every surface.

**Wiring.**
- Add `"frost"` to the climate weight maps in `sim/climate.go` for
  **genuinely cold climates only** — the tundra/`snow` archetype, `mountain`,
  and `cliffs` — where a hard freeze is climatically plausible in any season.
  Modest weights (e.g. `snow`/tundra: 3; `mountain`: 2; `cliffs`: 2).
- **Do NOT** add frost to temperate `land` or warm climates. The sim does not
  yet bias weather selection by season (`climate.go:20`: Track is "carried as
  data — Step ignores it"), so a temperate frost would appear year-round,
  including summer. Cold-climate-only sidesteps this entirely.
- Author `_datafiles/world/dogmud/weather/emotes/frost.yaml`: outdoor
  (`default` + `forest`/`city`/`water` variants) and indoor `strong` (cold
  seeping through walls, frost-ferns blooming on the inside of the windows).
  Flavor range spans crystalline-clear rime and low freezing mist. Same authoring
  rules as Thread 3.

**No splash.** `severeScene` (`weather_splash.go:58`) hard-whitelists only
storm/blizzard/dust — frost is excluded by construction. Consistent with the
policy that only *severe onsets* splash; frost is calm.

**Prompt glyph (optional, minor).** Frost may reuse a cold glyph (`❄`) in the
prompt with the Thread-1 ASCII fallback; decide during the plan. Not required.

**Future enhancement (out of scope).** Frost in temperate zones during winter,
gated on building sim season-biasing of weather selection.

---

## Thread 5 — Weather tempo (config-only)

**Goal.** Make weather read as slow atmospheric systems, not flicker. Purely
atmospheric, so slower is safe; the only risk is *too* still (playtest catches).

All levers already exist as config — **zero code**:
- `TickEveryGameHours` **1 → 8**. The sim steps once per this many game-hours;
  each step is when fronts move one hop, spawn, and decay. At 8 (≈20 min real
  per tick, given `RoundsPerDay=900` and 4s rounds) a zone's condition holds at
  least ~20 min (was ~2.5), fronts drift one zone-hop per ~20 min, and systems
  span hours in real-time. `FrontHardAge` is in ticks (48), so the hard-age
  backstop stretches to ~16 real hours automatically — but fronts normally die
  well before that via intensity decay, so no separate change is needed.
- `SpawnRateScale` **1.0 → 0.7**. Fewer fronts are born → more clear-sky
  stretches *between* systems. Net "quiet most of the time, weather is an event".

**Set in** `_datafiles/config.yaml` weather block. Document the rationale inline.

**Sparseness risk + first dial-back.** Tick 8 and the thinned spawn rate
compound: weather becomes genuinely rare across the world. This is the intended
"weather is an event" feel, but the playtest must confirm weather appears
**often enough to be noticed at all** in a normal session. If it reads as dead,
the first lever is to raise `SpawnRateScale` back toward 1.0 (more events) while
keeping the slow tick — so each event stays slow and stable, there are just more
of them. Lowering the tick is the second lever.

---

## Testing & the content gate

- **Unit:** `ConvertToAscii` widening (incl. FE0F drop + multi-rune emoji);
  `buildConfig` for `EmoteMildChancePct`/`EmoteStrongChancePct`; an
  `EmitAmbient` intensity-gate test.
- **Boot-clean smoke:** frost is a new authored weather type + emote file → must
  load without panic. Wipe instance saves first (SOP), then boot and watch for a
  clean weather-module load.
- **REQUIRED final task — adversarial in-game playtest** (CLAUDE.md content
  gate). Spawn a fresh character **in ASCII mode**; force conditions with
  `weather spawn <type> <zone> [intensity]`; verify:
  1. glyphs read cleanly in ASCII mode (prompt, weather, map) — no mojibake;
  2. emote cadence feels atmospheric, not spammy — calm overcast whispers,
     storms speak steadily (Thread 2);
  3. weather changes feel slow/systemic (Thread 5);
  4. indoor prose lands when sheltering (Thread 3);
  5. frost reads distinctly from snow and fog (Thread 4).
  Read every line as a confused human would; fix what it finds; re-run if needed
  before handing to the user.

## Config summary (new + changed knobs)

| Knob | From | To | Thread |
|------|------|----|--------|
| `EmoteMildChancePct` (new) | — | 30 | 2 |
| `EmoteStrongChancePct` (new) | — | 100 | 2 |
| `EmoteEveryRounds` | 20 | 24 | 2 |
| `TickEveryGameHours` | 1 | 8 | 5 |
| `SpawnRateScale` | 1.0 | 0.7 | 5 |

All defaults are playtest-tunable; none is load-bearing for correctness.
