# Splash Pipeline — Design

**Date:** 2026-07-21
**Status:** Approved design, pending implementation plan
**Roadmap:** `docs/PATH_TO_1.0.md` §5d (presentation/delight) + §1 weather polish. This
is sub-project **B** of the weather/seasons work; sub-project **A** (readability +
pacing) follows.

## Problem

DOGMud already prints full-screen ASCII "splash" art for a few celestial events
(sunrise, sunset, moon phases), but:

1. **The art looks pixelated** — the sun is a solid block of `#`/`@` with a
   background fill and the water is repeating `~` rows
   (`_datafiles/world/dogmud/templates/generic/sunrise.template`, `sunset.template`).
2. **Season changes and major weather have no splash at all.** A
   `weather.WeatherSeasonChanged` event already fires but has **zero consumers**
   (`modules/weather/weather_events.go`, `weather_tick.go`); storms/blizzards/dust
   only get a passive on-`look` alert line, never an onset announcement.
3. The web client renders none of this richly — it just shows the raw ASCII in
   the output feed like a terminal.

We want one reusable **splash pipeline** that renders an inline, panel-width
scene in the center output feed — a smooth **SVG scene on the web client** and
**refined Unicode/gradient ASCII on terminal** — driven by a single event, so the
same mechanism serves celestial events, season turns, severe weather, and (later)
mutation acquisition.

## Decisions locked in brainstorming

- **Inline, not overlay.** The splash renders in the center output panel, in the
  scroll, flowing with game text — *not* a modal/toast/takeover. (All overlay
  treatments were rejected.)
- **Per-client rendering.** Web → a panel-width SVG scene. Terminal/Mudlet →
  refined ASCII. Screen-reader → a one-line caption only.
- **Panel-width, short letterbox, static.** No animation.
- **Targeting:** severe **weather → only that zone**; **season turns → global**;
  celestial (sunrise/sunset/moon) → global. Mutation (later) → single user.
- **Severe-weather trigger:** storm, blizzard, dust storm only (not heatwave/fog).
- **Moon phases included** in the pipeline.
- **Full scene set this build** (all 17 scenes; see catalog).

## Goals

1. A reusable `Splash` event + a scene registry, so any subsystem can fire a
   splash by scene id + target without knowing about rendering or client types.
2. Per-client delivery: web SVG, terminal refined-ASCII, screen-reader caption —
   with **no double-render** (web must not also print the raw ASCII).
3. Migrate the existing celestial splashes (sunrise/sunset/moon) onto it with
   **refined, less-pixelated** art.
4. Wire the two missing consumers: **season change** (global) and **severe
   weather onset** (per-zone).
5. Leave a clean seam so **mutation acquisition** (§5d) can fire a user-targeted
   splash later with per-mutation art — designed for, not built now.

## Non-goals (explicitly deferred)

- **Mutation-acquisition consumer + per-mutation art** — §5d, a separate build.
  This spec only guarantees the `user` target mode works.
- **Weather/season readability + pacing tuning** — sub-project A (colors, contrast,
  slow-down, indoor prose). Separate spec.
- **Animation / transitions** — static only.
- **Per-scene sound, weather radar, forecast UI** — out of scope.

## Architecture

### 1. Scene registry

A **scene** is identified by a stable `SceneId` string and provides three
renderings:

| Rendering | Where it lives | Used by |
|---|---|---|
| Refined ASCII `.template` | `_datafiles/world/dogmud/templates/generic/splash/<id>.template` | terminal/Mudlet |
| Screen-reader caption `.screenreader.template` (or a caption string) | same dir, `<id>.screenreader.template` | screen-reader users; also the plain fallback |
| SVG scene builder | **client-side** in the web client JS, keyed by `<id>` | web client |

The Go side holds a small registry mapping `SceneId → {TemplatePath, defaultCaption}`.
The **SVG art is owned by the web client** (like the leather-mapper art), keyed by
the same scene id, with slots for dynamic data (zone name, date, phase). The Go
server never ships SVG markup — it ships the scene id + data, and the web client
renders it. This keeps art iteration client-side and the wire payload tiny.

### 2. The `Splash` event

```go
// internal/events (or a new internal/splash package for the helpers)
type SplashTarget uint8
const (
    SplashGlobal SplashTarget = iota // all active users
    SplashZone                       // all users in Zone
    SplashUser                       // just UserId
)

type Splash struct {
    SceneId string
    Caption string            // screen-reader / fallback line (may be built from Data)
    Target  SplashTarget
    Zone    string            // when Target == SplashZone
    UserId  int               // when Target == SplashUser
    Data    map[string]any    // dynamic slots (zone name, date, phase, mutation name…)
}
func (Splash) Type() string { return "Splash" }
```

Consumers just `events.AddToQueue(events.Splash{...})`.

### 3. Delivery hook (per-client branch)

A single listener (`internal/hooks/Splash_Deliver.go`, registered in `hooks.go`)
consumes `events.Splash` and resolves the target user set, then delivers **per
connection**:

```
resolve recipients:
  SplashGlobal → users.GetAllActiveUsers()
  SplashZone   → zone fan-out helper (see §4)
  SplashUser   → [users.GetByUserId(UserId)]

for each recipient user u:
  if u.ScreenReader:
        u.SendText(CategorySplash, caption)                     // caption only
  else if connection is WebClient (gmcp client name == "WebClient"):
        events.AddToQueue(GMCPOut{u.UserId, "Event.Splash",
            SplashPayload{Scene: SceneId, Caption: caption, Data: Data}})
  else: // telnet / Mudlet
        text := templates.Process("generic/splash/"+SceneId, Data, u.UserId)
        u.SendText(CategorySplash, text)                         // refined ASCII inline
```

Client type is read the same way the GMCP dispatcher does
(`modules/gmcp` client settings; web connections auto-register as `WebClient`).
This guarantees a web user gets the SVG **instead of**, not in addition to, the
ASCII. A new `messaging.CategorySplash` keeps splashes on their own toggleable
category.

> **Decoupling note:** the delivery hook needs "is this connection a web client?"
> Expose a tiny predicate from the gmcp module (e.g. `gmcp.IsWebClient(userId) bool`)
> rather than importing gmcp internals into hooks, mirroring the existing
> conversationadapter-style seam.

### 4. Zone fan-out helper

New helper (weather splashes need it; today only all-users `Broadcast` and
single-`Room.SendText` exist):

```go
// resolve all online users currently in a zone
func UsersInZone(zone string) []*users.UserRecord
```

Implementation: iterate the zone's rooms (`rooms` package exposes zone→roomIds),
collect each room's player user-ids, map to `*UserRecord`. Lives wherever the
rooms/users seam is cleanest (likely a small helper in `internal/rooms` or the
delivery hook). The delivery hook uses it for `SplashZone`.

### 5. Web client rendering

- New GMCP handler key `GMCPUpdateHandlers["Event.Splash"]` in
  `_datafiles/html/public/webclient-pure.html`.
- A client-side **scene library** — an object `SplashScenes = { "<id>": function(data){ return "<svg…>"; }, … }`
  (kept in a dedicated JS file or a `<script>` block) returning an SVG string
  sized to the output panel (`viewBox` letterbox, `width:100%`), with dynamic
  text filled from `data`.
- The handler appends the SVG as a **new inline block element in the output feed**
  (same append path as a normal output line), so it scrolls with play. Full panel
  width, static.
- Unknown scene id → fall back to printing the `caption` as a normal line (never
  break the feed).

### 6. Terminal rendering — refined ASCII art style

The refined `.template` art replaces block-fill sprites with **shaded/half-block
glyphs and real color gradients**:

- **Suns / moons:** a soft disc using `░▒▓█` shading rings + a radial-ish color
  gradient (bright core → warm/ cool halo) instead of a solid `#`/`@` block with
  `bg` fill. A faint outer halo of `░`.
- **Horizons/ground:** half-blocks (`▀▄`) for crisp edges.
- **Water/atmosphere:** graded block ramps (`▂▃▄▅▆▇`) and varied ripple glyphs
  (`≈≋`) rather than uniform `~` rows; a colored reflection column under the sun.
- **Palette:** 256-color `<ansi fg="…">` gradients with adequate luminance floors
  (this also pre-empts the contrast problems sub-project A fixes globally).
- Every scene has an `<id>.screenreader.template` that is caption-only (no art).
- **≤ ~14 lines tall, ≤ 78 cols** (MUD line-width rule).

### 7. Scene catalog (17 scenes)

| Group | Scene ids | Target |
|---|---|---|
| Celestial | `sunrise`, `sunset` | global |
| Moons (3 moons × 2 phases) | `moon_eye_full`, `moon_eye_new`, `moon_swiftmoon_full`, `moon_swiftmoon_new`, `moon_wanderer_full`, `moon_wanderer_new` | global |
| Seasons — temperate | `season_temperate_winter`, `season_temperate_spring`, `season_temperate_summer`, `season_temperate_autumn` | global |
| Seasons — monsoon | `season_monsoon_wet`, `season_monsoon_dry` | global |
| Severe weather | `weather_storm`, `weather_blizzard`, `weather_dust` | zone |

Each = refined ASCII template + screen-reader template + client-side SVG builder +
caption. (Season/weather captions interpolate the zone name; moon/sun captions
interpolate the date where the existing templates already do.)

## Consumers (this build)

1. **Sunrise / sunset** — migrate `internal/hooks/DayNightCycle_NotifySunriseSunset.go`
   from its `Broadcast` of the raw template to `events.Splash{SceneId:"sunrise"|"sunset",
   Target:SplashGlobal, Data: date}`. Existing templates get refined + an SVG scene.
2. **Moon phases** — migrate `internal/hooks/MoonPhase_BroadcastEmote.go` to
   `events.Splash{SceneId:"moon_<moon>_<phase>", Target:SplashGlobal, Data: date}`.
3. **Season change** — new listener on `weather.WeatherSeasonChanged{Zone,Track,From,To}`
   (currently unconsumed). Maps `To` season → scene id, emits
   `events.Splash{SceneId, Target:SplashGlobal, Data:{zone}}`. (Global per the
   decision, even though the event is per-zone — a season turn is world-scale
   flavor.)
4. **Severe weather onset** — the weather tick already resolves per-zone weather
   each step (`modules/weather/weather_tick.go` / `sim/tick.go resolveWeather`).
   Add onset detection: when a zone's resolved weather **transitions into**
   storm/blizzard/dust (was not that type last tick, now is), emit
   `events.Splash{SceneId:"weather_<type>", Target:SplashZone, Zone, Data:{zone}}`.
   Requires tracking the previous per-zone weather type across ticks (the sim
   already holds current weather; capture the pre-step snapshot to diff). Fires
   once per onset, not every tick while it persists. No splash when it *clears*.

## Design-for-later (not built now)

**Mutation acquisition** (§5d): the grant seam at
`internal/hooks/NewRound_UserRoundTick.go` (the `canAcquire` branch, with `user` +
mutation `spec` in scope) will emit `events.Splash{SceneId:"mutation_<id or cluster>",
Target:SplashUser, UserId, Data:{name, description}}`. This spec only guarantees the
`SplashUser` path works end-to-end; the mutation scenes/art + consumer are a §5d
task.

## Screen-reader & accessibility

- Screen-reader users always get the caption line only (no art), via the existing
  `ForceScreenReaderUserId` / `ScreenReader` flag path already used by sunrise/moon.
- Captions are authored to stand alone ("The sun rises." / "Winter descends on
  Stillwater." / "A storm breaks over the Reed Flats.").

## Config

- `messaging.CategorySplash` — players can toggle splashes like other categories.
- A module/gameplay toggle `SplashesEnabled` (default true) as a global kill-switch.
- (Weather-onset detection lives behind the existing `Modules.weather` enablement.)

## Error handling

- Unknown scene id: terminal falls back to the caption line; web falls back to the
  caption line. Never blocks the feed or panics.
- Missing template file: `templates.Process` already returns empty + logs; deliver
  the caption instead.
- Zone fan-out with no online users in the zone: no-op.

## Testing

- **Unit:** the delivery hook's recipient resolution per target mode (global/zone/
  user); `UsersInZone`; scene-id→template-path mapping; the severe-weather onset
  diff (transition-in fires once, persistence/clear does not).
- **Per-client branch:** a table test that a web-flagged user yields a GMCP
  `Event.Splash` and a telnet user yields ASCII text (mock the client predicate).
- **Boot-clean smoke:** all 17 templates + screenreader variants load; no panic.
- **Live harness walk:** trigger a season change and a storm (admin `weather spawn`
  / time skip), confirm terminal shows refined ASCII and (via the web client) the
  inline SVG scene renders in the output panel; confirm a desert-zone player does
  NOT get another zone's storm splash but DOES get the global season splash.
- **Content playtest gate:** the scene art + captions are player-facing content —
  run the adversarial playtest review before handoff (per SOP), reading every
  scene in a real client.

## Files touched / created

**Create:**
- `internal/events` (or `internal/splash`) — `Splash` event + `SplashTarget`.
- `internal/hooks/Splash_Deliver.go` — the delivery hook.
- `internal/hooks/WeatherSeasonChanged_Splash.go` — season-change consumer.
- zone fan-out helper (`UsersInZone`).
- `_datafiles/world/dogmud/templates/generic/splash/<id>.template` +
  `<id>.screenreader.template` — 17 scenes × 2.
- web client scene library (SVG builders) + `GMCPUpdateHandlers["Event.Splash"]`
  handler in `webclient-pure.html` (+ CSS for the inline scene block).
- `messaging.CategorySplash`.

**Modify:**
- `internal/hooks/DayNightCycle_NotifySunriseSunset.go`, `MoonPhase_BroadcastEmote.go`
  — emit `Splash` instead of raw `Broadcast`.
- `modules/weather/weather_tick.go` (+ sim) — severe-weather onset detection +
  emit `Splash`.
- `modules/gmcp` — export `IsWebClient(userId)` predicate.
- `internal/hooks/hooks.go` — register the new listeners.
- config — `SplashesEnabled`.
- `docs/PATH_TO_1.0.md` — §5d / §1 status.

## Future enhancements

- Mutation-acquisition consumer + per-mutation art (§5d).
- Optional per-scene fade-in (deferred — static chosen now).
- Heatwave/heavy-fog scenes; weather-*clearing* splashes.
- Moon phases beyond full/new (quarters) if the calendar model adds them.
- A generic "significant world event" splash for quests/bosses.
