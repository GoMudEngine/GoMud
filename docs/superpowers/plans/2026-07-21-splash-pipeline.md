# Splash Pipeline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A reusable inline splash system — one `Splash` event renders a smooth SVG scene in the web output panel, refined Unicode/gradient ASCII on terminal, and a caption for screen readers — driving celestial events, season turns, and severe weather (mutation acquisition designed-for, deferred).

**Architecture:** A low-level `internal/splash` package owns the `Splash` event + recipient resolver. **Two listeners** partition delivery by client type: `internal/hooks/Splash_Deliver.go` renders ASCII/caption to non-web recipients; `modules/gmcp/gmcp.Splash.go` emits GMCP `Event.Splash` to web recipients. Web-detection is a new low-level `connections.ClientSettings.IsWebConnection` flag both read. SVG art is client-side, keyed by scene id.

**Tech Stack:** Go, testify, the `events`/`hooks`/`connections`/`users`/`rooms`/`templates`/`gmcp` packages; vanilla JS + inline SVG in `webclient-pure.html`.

**Spec:** `docs/superpowers/specs/2026-07-21-splash-pipeline-design.md`

---

## Grounded facts (verified — reference while implementing)

- **Event interface:** `events.Event` = `Type() string`. Events are plain structs
  in `internal/events/eventtypes.go` (see `Broadcast`, line 92).
- **Listener registration:** `events.RegisterListener(events.SomeEvent{}, HandlerFn)`
  in `internal/hooks/hooks.go` `RegisterListeners()`. Handler:
  `func(e events.Event) events.ListenerReturn` → `events.Continue`/`events.Cancel`.
- **Hook delivery pattern** (`internal/hooks/Broadcast_SendToAll.go`): iterate
  `users.GetAllActiveUsers()`; per user check `u.ScreenReader`; send via
  `connections.SendTo([]byte(text), u.ConnectionId())` (with
  `term.AnsiMoveCursorColumn.String()+term.AnsiEraseLine.String()` prefix unless
  SkipLineRefresh); queue `events.RedrawPrompt{UserId}` first.
- **Templates:** `templates.Process(path string, data any) (string, error)`;
  screen-reader variant: `templates.Process(path, data, templates.ForceScreenReaderUserId)`.
  `templates.AnsiParse(text)` converts ansi tags. Splash templates go under
  `_datafiles/world/dogmud/templates/generic/splash/`.
- **Existing consumers to migrate:** `internal/hooks/DayNightCycle_NotifySunriseSunset.go`
  (`events.DayNightCycle{IsSunrise}`, `gametime.GetDate()`), `MoonPhase_BroadcastEmote.go`
  (`events.MoonPhase{MoonName, PhaseName}`, `moonNameToSlug`).
- **GMCP emit (web):** `events.AddToQueue(gmcp.GMCPOut{UserId, Module, Payload})` —
  `GMCPOut` is defined in `modules/gmcp/gmcp.go:85`, so **only code in modules/ may
  emit it**. The gmcp dispatcher frames web vs telnet by `gmcpSettings.Client.Name == "WebClient"`.
- **Web client name is set** in `modules/gmcp/gmcp.go:146` (`onNetConnect`, websocket
  branch: `setting.Client.Name = "WebClient"`). `connections.ClientSettings`
  (`internal/connections/clientsettings.go:10`) currently has NO client-type field;
  read/write via `connections.GetClientSettings(id)` / `OverwriteClientSettings(id, cs)`.
- **Zone rooms:** `rooms.GetAllZoneRoomsIds(zoneName string) []int`
  (`internal/rooms/roommanager.go:252`). A loaded room exposes its players; resolve a
  room via `rooms.LoadRoom(id)` and read its player user-ids (see how
  `Room.SendText`/room player iteration works in `internal/rooms/rooms.go`).
- **Weather season event:** `weather.WeatherSeasonChanged{Zone, Track, From, To}`
  (`modules/weather/weather_events.go`), emitted in `resolveSeasons`
  (`modules/weather/weather_tick.go:106-116`) — currently unconsumed. Season/weather
  values are the `sim`/`seasons` enums.
- **Weather tick:** `weatherModule.tick` (`weather_tick.go:143`) calls
  `sim.Step(...) (next, diff)` then `engine.Reconcile(m.state.Weather)`. Per-zone
  current weather lives in `m.state.Weather map[ZoneId]WeatherType`.
- **Web client GMCP receive:** `handleGMCP(namespace, obj)` in
  `_datafiles/html/public/webclient-pure.html` dispatches via the
  `GMCPUpdateHandlers` map (defined ~line 500). Add a `"Event.Splash"` key. Output
  is appended to the feed by the client's line-append path (grep `appendOutput`/
  `echo`/`termWrite` in the file to find the exact function).

---

## File structure

**Create:**
- `internal/splash/splash.go` — `Splash` event, `SplashTarget`, `Recipients()`.
- `internal/splash/splash_test.go` — resolver tests.
- `internal/hooks/Splash_Deliver.go` — terminal/SR delivery (non-web recipients).
- `internal/hooks/Splash_Deliver_test.go` — render-selection tests.
- `modules/gmcp/gmcp.Splash.go` — web delivery (GMCP emit for web recipients).
- `_datafiles/world/dogmud/templates/generic/splash/<id>.template` + `<id>.screenreader.template` (17 scenes ×2).
- web scene library + handler in `webclient-pure.html` (+ CSS).

**Modify:**
- `internal/connections/clientsettings.go` — add `IsWebConnection bool`.
- `modules/gmcp/gmcp.go` — set the flag on websocket connect.
- `internal/events/eventtypes.go` — (only if we decide to home the event here; plan homes it in `internal/splash`).
- `internal/hooks/hooks.go` — register the terminal delivery listener.
- `internal/hooks/DayNightCycle_NotifySunriseSunset.go`, `MoonPhase_BroadcastEmote.go` — emit `Splash`.
- `modules/weather/weather_tick.go` (+ a new `weather_splash.go`) — season + onset consumers.
- `internal/messaging` — `CategorySplash`.
- config — `SplashesEnabled`.
- `docs/PATH_TO_1.0.md`.

---

## Task 1: splash package — event + recipient resolver

**Files:**
- Create: `internal/splash/splash.go`
- Test: `internal/splash/splash_test.go`

- [ ] **Step 1: Write the failing test**

```go
package splash

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplashEventType(t *testing.T) {
	assert.Equal(t, "Splash", Splash{}.Type())
}

func TestTargetConstants(t *testing.T) {
	// Distinct values, User is not the zero value (so a zero-value Splash is Global).
	assert.Equal(t, SplashTarget(0), TargetGlobal)
	assert.NotEqual(t, TargetGlobal, TargetZone)
	assert.NotEqual(t, TargetZone, TargetUser)
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/splash/ -run TestSplash -v`
Expected: FAIL — package/types don't exist.

- [ ] **Step 3: Write `internal/splash/splash.go`**

```go
// Package splash defines the reusable in-world splash event: one event, rendered
// per-client downstream (SVG on web, refined ASCII on terminal, caption for
// screen readers). Delivery lives in listeners (internal/hooks + modules/gmcp);
// this package only defines the event and resolves recipients.
package splash

import (
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

type SplashTarget uint8

const (
	TargetGlobal SplashTarget = iota // all active users
	TargetZone                       // all users currently in Zone
	TargetUser                       // just UserId
)

// Splash is emitted by any subsystem that wants a splash scene shown. SceneId
// selects the art (terminal template + client-side SVG); Caption is the
// screen-reader / fallback line; Data fills dynamic slots (zone, date, name).
type Splash struct {
	SceneId string
	Caption string
	Target  SplashTarget
	Zone    string
	UserId  int
	// Data is passed straight to templates.Process and the GMCP payload. It is a
	// gametime.Date for celestial scenes, or a map[string]any (e.g. {"zone": …})
	// for season/weather scenes.
	Data any
}

func (Splash) Type() string { return "Splash" }

// Recipients resolves the users a splash should reach. Deterministic; callers
// (the terminal + gmcp listeners) partition by client type afterward.
func Recipients(s Splash) []*users.UserRecord {
	switch s.Target {
	case TargetUser:
		if u := users.GetByUserId(s.UserId); u != nil {
			return []*users.UserRecord{u}
		}
		return nil
	case TargetZone:
		return usersInZone(s.Zone)
	default: // TargetGlobal
		return users.GetAllActiveUsers()
	}
}

// usersInZone returns online users currently in any room of zone.
func usersInZone(zone string) []*users.UserRecord {
	out := []*users.UserRecord{}
	seen := map[int]bool{}
	for _, roomId := range rooms.GetAllZoneRoomsIds(zone) {
		room := rooms.LoadRoom(roomId)
		if room == nil {
			continue
		}
		for _, uid := range room.GetPlayers() { // returns []int of user-ids in the room
			if seen[uid] {
				continue
			}
			if u := users.GetByUserId(uid); u != nil {
				seen[uid] = true
				out = append(out, u)
			}
		}
	}
	return out
}
```

> Verify `room.GetPlayers()` is the correct accessor name for the room's player
> user-ids (grep `internal/rooms/rooms.go` for `GetPlayers`/`Players`); adjust the
> call if the codebase names it differently.

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/splash/ -run TestSplash -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/splash/
git commit -m "feat(splash): Splash event + recipient resolver"
```

---

## Task 2: connections web-client flag

**Files:**
- Modify: `internal/connections/clientsettings.go`
- Modify: `modules/gmcp/gmcp.go`

- [ ] **Step 1: Add the field**

In `internal/connections/clientsettings.go`, add to `ClientSettings`:

```go
	IsWebConnection   bool // Set true for websocket (web client) connections.
```

- [ ] **Step 2: Set it on websocket connect**

In `modules/gmcp/gmcp.go` `onNetConnect`, inside the `if n.IsWebSocket()` branch
(after `g.cache.Add(...)`), also set the low-level flag so non-gmcp code can read it:

```go
		cs := connections.GetClientSettings(n.ConnectionId())
		cs.IsWebConnection = true
		connections.OverwriteClientSettings(n.ConnectionId(), cs)
```

(`connections` is already imported by the gmcp module.)

- [ ] **Step 3: Build**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add internal/connections/clientsettings.go modules/gmcp/gmcp.go
git commit -m "feat(connections): IsWebConnection flag set on websocket connect"
```

---

## Task 3: messaging category + config toggle

**Files:**
- Modify: `internal/messaging` (category definitions)
- Modify: `internal/configs/config.gameplay.go` + `_datafiles/config.yaml`

- [ ] **Step 1: Add the category**

Find the message-category definitions (grep `CategoryWeather` in `internal/messaging`).
Add a sibling:

```go
	CategorySplash = "splash"
```

matching the exact form of the surrounding constants (string value + any registry
list the others are added to — replicate every place `CategoryWeather` appears).

- [ ] **Step 2: Add the config toggle**

In `internal/configs/config.gameplay.go` `GamePlay` struct add:

```go
	SplashesEnabled ConfigBool `yaml:"SplashesEnabled"` // Master toggle for scene splashes (celestial/season/weather)
```

Default it true in `Validate()` — `ConfigBool` defaults false, so seed it in
`_datafiles/config.yaml` under `GamePlay:`:

```yaml
  SplashesEnabled: true
```

- [ ] **Step 3: Build + commit**

Run: `go build ./...`

```bash
git add internal/messaging/ internal/configs/config.gameplay.go _datafiles/config.yaml
git commit -m "feat(splash): CategorySplash + SplashesEnabled toggle"
```

---

## Task 4: terminal/SR delivery hook

**Files:**
- Create: `internal/hooks/Splash_Deliver.go`
- Test: `internal/hooks/Splash_Deliver_test.go`
- Modify: `internal/hooks/hooks.go`

- [ ] **Step 1: Write the failing test**

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/splash"
	"github.com/stretchr/testify/assert"
)

func TestSplashTemplatePath(t *testing.T) {
	assert.Equal(t, "generic/splash/sunrise", splashTemplatePath("sunrise"))
	assert.Equal(t, "generic/splash/weather_storm", splashTemplatePath("weather_storm"))
}

func TestSplashDeliverWrongEventType(t *testing.T) {
	// A non-Splash event returns Continue without panic.
	assert.Equal(t, /* events.Continue */ 0, int(Splash_Deliver(dummyEvent{})))
}

type dummyEvent struct{}

func (dummyEvent) Type() string { return "dummy" }

var _ = splash.Splash{} // keep import
```

*(Adjust the `events.Continue` integer/const comparison to the codebase's
`ListenerReturn` type; the point is the wrong-type guard returns Continue.)*

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/hooks/ -run TestSplash -v`
Expected: FAIL — `splashTemplatePath`/`Splash_Deliver` undefined.

- [ ] **Step 3: Write `internal/hooks/Splash_Deliver.go`**

```go
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/splash"
	"github.com/GoMudEngine/GoMud/internal/templates"
)

func splashTemplatePath(sceneId string) string { return "generic/splash/" + sceneId }

// Splash_Deliver renders a splash for every NON-web recipient (refined ASCII, or
// the caption for screen-reader users). Web recipients are handled by the gmcp
// module's listener, which emits an SVG scene instead.
func Splash_Deliver(e events.Event) events.ListenerReturn {
	evt, ok := e.(splash.Splash)
	if !ok {
		return events.Continue
	}
	if !bool(configs.GetGamePlayConfig().SplashesEnabled) {
		return events.Continue
	}

	tmpl := splashTemplatePath(evt.SceneId)

	for _, u := range splash.Recipients(evt) {
		if connections.GetClientSettings(u.ConnectionId()).IsWebConnection {
			continue // gmcp listener handles web recipients
		}
		if u.ScreenReader {
			if evt.Caption != "" {
				u.SendText(messaging.CategorySplash, evt.Caption)
			}
			continue
		}
		text, err := templates.Process(tmpl, evt.Data)
		if err != nil || text == "" {
			if evt.Caption != "" {
				u.SendText(messaging.CategorySplash, evt.Caption) // fallback
			}
			continue
		}
		u.SendText(messaging.CategorySplash, text)
	}
	return events.Continue
}
```

- [ ] **Step 4: Register the listener**

In `internal/hooks/hooks.go` `RegisterListeners()`, add (with a `splash` import):

```go
	events.RegisterListener(splash.Splash{}, Splash_Deliver)
```

- [ ] **Step 5: Run tests + build**

Run: `go test ./internal/hooks/ -run TestSplash -v && go build ./...`
Expected: PASS + clean.

- [ ] **Step 6: Commit**

```bash
git add internal/hooks/Splash_Deliver.go internal/hooks/Splash_Deliver_test.go internal/hooks/hooks.go
git commit -m "feat(splash): terminal + screen-reader delivery hook"
```

---

## Task 5: gmcp web delivery listener

**Files:**
- Create: `modules/gmcp/gmcp.Splash.go`

- [ ] **Step 1: Write `modules/gmcp/gmcp.Splash.go`**

```go
package gmcp

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/splash"
)

// SplashPayload is the GMCP Event.Splash body the web client renders as an inline
// SVG scene. Art is client-side, keyed by Scene.
type SplashPayload struct {
	Scene   string         `json:"scene"`
	Caption string         `json:"caption"`
	Data    map[string]any `json:"data,omitempty"`
}

// deliverSplashToWeb emits Event.Splash to every WEB recipient of a splash.
func (g *GMCPModule) deliverSplashToWeb(e events.Event) events.ListenerReturn {
	evt, ok := e.(splash.Splash)
	if !ok {
		return events.Continue
	}
	if !bool(configs.GetGamePlayConfig().SplashesEnabled) {
		return events.Continue
	}
	payload := SplashPayload{Scene: evt.SceneId, Caption: evt.Caption, Data: evt.Data}
	for _, u := range splash.Recipients(evt) {
		if !connections.GetClientSettings(u.ConnectionId()).IsWebConnection {
			continue // non-web handled by the internal/hooks listener
		}
		events.AddToQueue(GMCPOut{UserId: u.UserId, Module: "Event.Splash", Payload: payload})
	}
	return events.Continue
}
```

- [ ] **Step 2: Register it in the gmcp module init**

Find where the gmcp module registers its other event listeners (grep
`RegisterListener` in `modules/gmcp/gmcp.go`) and add:

```go
	events.RegisterListener(splash.Splash{}, g.deliverSplashToWeb)
```

- [ ] **Step 3: Build + commit**

Run: `go build ./...`

```bash
git add modules/gmcp/gmcp.Splash.go modules/gmcp/gmcp.go
git commit -m "feat(splash): gmcp web delivery (Event.Splash) for web clients"
```

---

## Task 6: refined ASCII art style + sunrise/sunset scenes + migrate day-night

**Files:**
- Create: `_datafiles/world/dogmud/templates/generic/splash/sunrise.template` + `.screenreader.template`
- Create: `.../splash/sunset.template` + `.screenreader.template`
- Modify: `internal/hooks/DayNightCycle_NotifySunriseSunset.go`

**Refined ASCII style (apply to every scene):** replace solid block sprites +
`bg` fills with shaded discs (`░▒▓█`) + a real `<ansi fg="256code">` gradient
core→halo; half-blocks (`▀▄`) for horizons; graded ramps (`▂▃▄▅▆▇`) + varied
ripples (`≈≋`) for water/atmosphere instead of uniform `~`; ≤14 lines, ≤78 cols.

- [ ] **Step 1: Author `sunrise.template`** (refined; keep the date footer from the
  current file). Example skeleton:

```
 <ansi fg="239">┌───────────────────────────────────────────────────────────────────────┐</ansi>
        <ansi fg="94">·        ˙   ✦          ·        ˙          ·   </ansi>
              <ansi fg="220">░▒▓▓▒░</ansi>
            <ansi fg="221">░▒▓█████▓▒░</ansi>
           <ansi fg="229">▒▓█████████▓▒</ansi>
            <ansi fg="220">░▒▓█████▓▒░</ansi>
     <ansi fg="74">▂▃▄▅▆▇</ansi><ansi fg="221">▆▅▄▅▆</ansi><ansi fg="74">▇▆▅▄▃▂▃▄▅▆▇</ansi>
     <ansi fg="31">≈≋≈ ≋</ansi><ansi fg="179">≈≋≈ ≋≈</ansi><ansi fg="31">≋ ≈≋≈ ≋ ≈≋ ≈≋</ansi>
                     <ansi fg="230">The sun rises.</ansi>

           It is <ansi fg="230">day {{ .Day }}</ansi> of <ansi fg="230">year {{ .Year }}</ansi>, the month of <ansi fg="230">{{ month .Month }}</ansi>.
 <ansi fg="239">└───────────────────────────────────────────────────────────────────────┘</ansi>
```

(Center to ≤78 cols; use higher-luminance 256 codes so it reads on dark terminals.)

- [ ] **Step 2: Author `sunrise.screenreader.template`:**

```
The sun rises. It is day {{ .Day }} of year {{ .Year }}, the month of {{ month .Month }}.
```

- [ ] **Step 3: Author `sunset.template` + `.screenreader.template`** in the same
  refined style (warm→dusky palette; "The sun sets." caption).

- [ ] **Step 4: Migrate the day-night hook** — replace the `Broadcast` in
  `internal/hooks/DayNightCycle_NotifySunriseSunset.go` with:

```go
	if evt.IsSunrise {
		events.AddToQueue(splash.Splash{
			SceneId: "sunrise",
			Caption: "The sun rises.",
			Target:  splash.TargetGlobal,
			Data:    gametime.GetDate(),
		})
		return events.Continue
	}
	events.AddToQueue(splash.Splash{
		SceneId: "sunset",
		Caption: "The sun sets.",
		Target:  splash.TargetGlobal,
		Data:    gametime.GetDate(),
	})
	return events.Continue
```

`Splash.Data` is `any` (Task 1), so the snippet passes `gametime.GetDate()`
straight through as `Data` — the delivery hook forwards it unchanged to
`templates.Process`, exactly like the current code. No wrapper needed.

- [ ] **Step 5: Build + boot-smoke the templates load, then commit**

Run: `go build ./...`

```bash
git add _datafiles/world/dogmud/templates/generic/splash/sunrise* _datafiles/world/dogmud/templates/generic/splash/sunset* internal/hooks/DayNightCycle_NotifySunriseSunset.go internal/splash/splash.go
git commit -m "feat(splash): refined sunrise/sunset scenes; migrate day-night to Splash"
```

> **Note:** apply the `Splash.Data any` change from Step 4 back into Task 1's file
> and its test before this commit builds.

---

## Task 7: web client — Event.Splash handler + scene library + inline SVG

**Files:**
- Modify: `_datafiles/html/public/webclient-pure.html`

- [ ] **Step 1: Add the scene library** (a `<script>` block or inline object). One
  builder per scene id, returning an SVG string sized to the panel. Start with
  sunrise (port the SVG from the approved mockup) as the worked example:

```js
const SplashScenes = {
  sunrise: function(d){ return `
    <svg class="splash-scene" viewBox="0 0 560 150" preserveAspectRatio="xMidYMid meet">
      <defs>
        <linearGradient id="sk" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#16233c"/><stop offset=".55" stop-color="#5c4a5b"/><stop offset="1" stop-color="#e6a04a"/></linearGradient>
        <radialGradient id="su" cx="50%" cy="50%" r="50%"><stop offset="0" stop-color="#fff7dc"/><stop offset=".5" stop-color="#ffd766"/><stop offset="1" stop-color="#ef972c"/></radialGradient>
        <linearGradient id="wt" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#3f8fb0"/><stop offset="1" stop-color="#102f3f"/></linearGradient>
      </defs>
      <rect width="560" height="96" fill="url(#sk)"/>
      <circle cx="280" cy="90" r="28" fill="url(#su)"/>
      <rect y="96" width="560" height="54" fill="url(#wt)"/>
      <text x="280" y="145" text-anchor="middle" font-family="Georgia,serif" font-size="12" fill="#ffe9a8">The sun rises.</text>
    </svg>`; }
  // + sunset, moons, seasons, weather — Task 12
};
```

- [ ] **Step 2: Add the GMCP handler** — a `"Event.Splash"` key in the
  `GMCPUpdateHandlers` map:

```js
  "Event.Splash": function(){
    const p = (GMCPStructs["Event"] && GMCPStructs["Event"].Splash) || {};
    const build = SplashScenes[p.scene];
    const html = build ? build(p.data || {}) : null;
    if (html) { appendSceneToOutput(html); }
    else if (p.caption) { appendOutputLine(p.caption); } // fallback
  }
```

- [ ] **Step 3: Add `appendSceneToOutput`** using the client's existing output-append
  mechanism. Grep the file for the function that appends a normal output line
  (e.g. `appendOutput`, `writeToTerminal`, the element the feed appends into) and
  wrap the SVG in a full-width block appended at the same place:

```js
  function appendSceneToOutput(svgHtml){
    const el = document.createElement('div');
    el.className = 'splash-scene-wrap';
    el.innerHTML = svgHtml;
    /* append `el` to the SAME output container normal lines use, then autoscroll */
    OUTPUT_CONTAINER.appendChild(el);      // replace OUTPUT_CONTAINER with the real ref
    scrollOutputToBottom();                // reuse existing autoscroll
  }
```

- [ ] **Step 4: Add CSS** (inline `<style>` or `dashboard.css`):

```css
  .splash-scene-wrap { margin:8px 0; }
  .splash-scene { display:block; width:100%; border-radius:6px; }
```

- [ ] **Step 5: Manual verify** — deferred to Task 13 (needs a running server + web
  client). Confirm the file parses (no JS syntax error) by loading the page.

- [ ] **Step 6: Commit**

```bash
git add _datafiles/html/public/webclient-pure.html _datafiles/html/public/static/css/dashboard.css
git commit -m "feat(splash): web client Event.Splash handler + inline SVG scene (sunrise)"
```

---

## Task 8: moon phases → Splash

**Files:**
- Modify: `internal/hooks/MoonPhase_BroadcastEmote.go`
- Create: 6 moon templates (Task 12 authors the art; here wire the emit)

- [ ] **Step 1: Emit Splash from the moon hook**

Replace the `Broadcast` in `BroadcastMoonPhase` with:

```go
	sceneId := "moon_" + moonNameToSlug(evt.MoonName) + "_" + evt.PhaseName
	events.AddToQueue(splash.Splash{
		SceneId: sceneId,
		Caption: evt.MoonName + " is " + evt.PhaseName + ".",
		Target:  splash.TargetGlobal,
	})
```

(Moon templates need no dynamic data today; `Data` nil is fine.)

- [ ] **Step 2: Build + commit**

Run: `go build ./...`

```bash
git add internal/hooks/MoonPhase_BroadcastEmote.go
git commit -m "feat(splash): migrate moon-phase announcements to Splash"
```

---

## Task 9: season-change consumer (global)

**Files:**
- Create: `modules/weather/weather_splash.go`
- Modify: `modules/weather/weather.go` (register the listener in the module's init/setup)

- [ ] **Step 1: Write the consumer**

```go
package weather

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/splash"
)

// onSeasonChanged turns a per-zone season flip into a GLOBAL season splash.
func onSeasonChanged(e events.Event) events.ListenerReturn {
	evt, ok := e.(WeatherSeasonChanged)
	if !ok {
		return events.Continue
	}
	sceneId := seasonSceneId(evt.Track, evt.To) // e.g. "season_temperate_winter"
	events.AddToQueue(splash.Splash{
		SceneId: sceneId,
		Caption: seasonCaption(evt.To, evt.Zone), // "Winter descends on <zone>."
		Target:  splash.TargetGlobal,
		Data:    map[string]any{"zone": evt.Zone},
	})
	return events.Continue
}
```

Implement `seasonSceneId(track, season)` and `seasonCaption(season, zone)` mapping
the `seasons`/`Track` enums to the catalog ids (`season_temperate_winter`,
`season_monsoon_wet`, …) and human captions. (Read the `seasons` package for the
enum values/String() to map exactly.)

- [ ] **Step 2: Register it** where the weather module wires its listeners (grep
  `RegisterListener` in `modules/weather/`):

```go
	events.RegisterListener(WeatherSeasonChanged{}, onSeasonChanged)
```

- [ ] **Step 3: Build + commit**

```bash
git add modules/weather/weather_splash.go modules/weather/weather.go
git commit -m "feat(splash): global splash on season change"
```

---

## Task 10: severe-weather onset detection + consumer (per-zone)

**Files:**
- Modify: `modules/weather/weather_tick.go`
- Modify: `modules/weather/weather_splash.go`

- [ ] **Step 1: Snapshot previous weather + diff for onset**

In `weatherModule.tick`, capture the pre-step per-zone weather, and after
`engine.Reconcile(m.state.Weather)` call a new `m.emitSevereOnsets(prev)`:

```go
	prev := map[ZoneId]WeatherType{}       // use the sim's ZoneId/WeatherType types
	for z, w := range m.state.Weather { prev[z] = w }
	next, diff := sim.Step(...)
	m.state = next
	_ = diff
	engine.Reconcile(m.state.Weather)
	m.emitSevereOnsets(prev)               // NEW
```

- [ ] **Step 2: Implement `emitSevereOnsets`** in `weather_splash.go`:

```go
// severeScene maps the severe weather types to their splash scene ids. Only
// these three trigger a splash (per design).
func severeScene(w WeatherType) (string, bool) {
	switch w {
	case WeatherStorm:    return "weather_storm", true
	case WeatherBlizzard: return "weather_blizzard", true
	case WeatherDust:     return "weather_dust", true
	}
	return "", false
}

// emitSevereOnsets fires one zone-scoped splash when a zone TRANSITIONS INTO a
// severe weather type (not while it persists, not when it clears).
func (m *weatherModule) emitSevereOnsets(prev map[ZoneId]WeatherType) {
	for zoneId, w := range m.state.Weather {
		scene, severe := severeScene(w)
		if !severe {
			continue
		}
		if prev[zoneId] == w {
			continue // already severe last tick — no re-announce
		}
		zoneName := m.zoneName(zoneId) // resolve display zone name from ZoneId
		events.AddToQueue(splash.Splash{
			SceneId: scene,
			Caption: severeCaption(w, zoneName),
			Target:  splash.TargetZone,
			Zone:    zoneName,
			Data:    map[string]any{"zone": zoneName},
		})
	}
}
```

Match `WeatherStorm/Blizzard/Dust` to the sim's actual enum identifiers (read
`modules/weather/sim/weather.go`); implement `m.zoneName(ZoneId)` from the graph
(the reverse of however `ZoneId` is derived) and `severeCaption`.

- [ ] **Step 3: Build + commit**

```bash
git add modules/weather/weather_tick.go modules/weather/weather_splash.go
git commit -m "feat(splash): per-zone splash on severe-weather onset"
```

---

## Task 11: author the remaining scenes (catalog)

**Files:** create templates + SVG builders for the remaining 15 scenes.

For EACH scene below, create three renderings following the Task 6 style guide and
the Task 7 SVG pattern: `<id>.template`, `<id>.screenreader.template`, and a
`SplashScenes["<id>"]` SVG builder. Captions interpolate `{{ .zone }}` where noted.

- [ ] **Moons (6):** `moon_eye_full`, `moon_eye_new`, `moon_swiftmoon_full`,
  `moon_swiftmoon_new`, `moon_wanderer_full`, `moon_wanderer_new` — a shaded disc
  (bright full / dark-rimmed new), per-moon tint. (Port tone from the existing
  `generic/moon_*` templates but refined.)
- [ ] **Seasons — temperate (4):** `season_temperate_winter/spring/summer/autumn`
  — frost/bud/haze/rust palettes; caption e.g. "Winter descends on {{ .zone }}."
- [ ] **Seasons — monsoon (2):** `season_monsoon_wet`, `season_monsoon_dry`.
- [ ] **Severe weather (3):** `weather_storm`, `weather_blizzard`, `weather_dust`
  — dramatic sky + the caption "A storm breaks over {{ .zone }}." etc.

- [ ] **Commit per group** (4 commits) so review is chunked:

```bash
git add _datafiles/world/dogmud/templates/generic/splash/moon_* _datafiles/html/public/webclient-pure.html
git commit -m "content(splash): moon-phase scenes (ascii + svg)"
# repeat for seasons-temperate, seasons-monsoon, severe-weather
```

---

## Task 12: docs

- [ ] Update `docs/PATH_TO_1.0.md` §5d / §1: splash pipeline shipped (celestial +
  season + severe weather); note mutation-acquisition splash is the remaining §5d
  consumer.
- [ ] Commit: `docs(splash): PATH_TO_1.0 status`.

---

## Task 13: integration verification (boot + harness walk + content playtest)

**Files:** none (verification only)

- [ ] **Step 1: Boot smoke** — nuke instances, `go run .`, confirm all
  `generic/splash/*` templates load, no panic, `moderation`/weather load clean.
- [ ] **Step 2: Terminal walk** (telnet/harness): force events (`weather spawn`,
  time skip to sunrise, a season boundary) and read the refined ASCII scenes inline
  — confirm they're centered ≤80 cols and readable on a dark background.
- [ ] **Step 3: Web walk** — open `webclient-pure.html` against the server; trigger
  a sunrise + a storm + a season change; confirm the SVG scene renders inline in the
  output feed, panel-width, flowing with text; confirm a player in a different zone
  does NOT receive another zone's storm but DOES receive the global season splash.
- [ ] **Step 4: Screen-reader path** — a `ScreenReader` user gets the caption line
  only (no art) on each event.
- [ ] **Step 5: Content playtest gate** — per SOP, run the adversarial playtest
  review over the scene prose/captions before handoff (player-facing content).
- [ ] **Step 6:** `go test ./...` green.

---

## Self-review checklist (run after implementing)

- Spec sections → tasks: event+resolver (T1), web flag (T2), category+config (T3),
  terminal delivery (T4), web delivery (T5), sunrise/sunset+style+migration (T6),
  web handler+scene lib (T7), moon migration (T8), season consumer (T9), weather
  onset (T10), remaining art (T11), docs (T12), verify (T13). ✅
- Type names consistent: `splash.Splash`, `splash.TargetGlobal/Zone/User`,
  `splash.Recipients`, `Splash_Deliver`, `SplashPayload`, `splashTemplatePath`,
  scene ids match the catalog. ✅
- `Splash.Data` is `any` (Task 6 decision) — ensure Task 1 struct + delivery calls
  agree. ✅
- No placeholders — plumbing steps show real code; art steps reference the style
  guide + worked example (art is authored in execution). ✅
```
