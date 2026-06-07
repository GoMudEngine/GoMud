# Automation Panel — Phase 2 Implementation Plan (Ticks)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the **Ticks** tab to the automation panel — user-defined real-time-second timers that fire command(s) from the web client. This phase also introduces the two pieces Phase 3 reuses: **inbound GMCP** (`Char.Automation.Set/Remove`) for tick/trigger CRUD, and a client **`SendGMCP`** binary-frame helper.

**Architecture:** Server-stores / client-executes. New `user.Ticks` storage persists with the user save and is exposed by extending the existing outbound `Char.Automation` payload. CRUD comes IN over GMCP (`Char.Automation.Set`/`Remove` added to `gmcp.go`'s `HandleIAC` switch). The runtime is client-side: a JS `setInterval` per enabled tick `SendData`s its commands; left-click fires now; an enable toggle pauses it.

**Tech Stack:** Go (UserRecord storage, GMCP module + inbound IAC handler), vanilla JS/CSS web client, GMCP.

**Spec:** `docs/superpowers/specs/2026-06-07-web-client-automation-panel-design.md` (Parts B/C/D/E + phasing — this is Phase 2). Phase 1 (Macros & Aliases) is already merged on master.

---

## File Structure

- **Modify:** `internal/users/userrecord.go` — add `UserTick` type, `Ticks []UserTick` field, and `SetTick`/`RemoveTick` helpers.
- **Modify:** `modules/gmcp/gmcp.Automation.go` + `gmcp.Automation_test.go` — add ticks to the payload + builder.
- **Modify:** `modules/gmcp/gmcp.go` — add inbound `Char.Automation.Set` / `Char.Automation.Remove` cases to `HandleIAC`.
- **Modify:** `_datafiles/html/public/webclient-pure.html` — `SendGMCP` helper, Ticks tab render (rows + toggle), `setInterval` runtime, tick editor.
- **Modify:** `_datafiles/html/public/static/css/dashboard.css` — tick-row styles (reuse `.auto-row`/`.auto-tog` from Phase 1).
- **Create:** `_datafiles/world/dogmud/templates/help/ticks.template` — `help ticks` topic.
- **Modify:** `modules/gmcp/context.md`, `internal/users/context.md` (if present) — doc the inbound handler + Ticks storage.

**Conventions:** `git add` ONLY named files (unrelated world-state files in the tree — never `-A`/`.`). Each task its own commit. Trailer EXACTLY: `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`. Prefer **codegraph MCP** for Go symbol verification. XSS: server strings via `textContent`/`dataset`, never `innerHTML`.

---

### Task 1: `UserTick` storage on the user record

**Files:**
- Modify: `internal/users/userrecord.go`
- Test: `internal/users/userrecord_test.go` (create or extend)

- [ ] **Step 1: Write a failing test for SetTick/RemoveTick**

In a `userrecord_test.go`:
```go
func TestUserTicks_SetAssignsIdAndUpdates(t *testing.T) {
	u := &UserRecord{}
	a := u.SetTick(UserTick{Name: "sip", Commands: "drink health", IntervalSec: 30, Enabled: true})
	if a.Id == 0 || len(u.Ticks) != 1 { t.Fatalf("create failed: %+v", u.Ticks) }
	a.Name = "sip2"
	u.SetTick(a)
	if len(u.Ticks) != 1 || u.Ticks[0].Name != "sip2" { t.Fatalf("update should not duplicate: %+v", u.Ticks) }
	// interval floor
	b := u.SetTick(UserTick{Name: "fast", IntervalSec: 0})
	if b.IntervalSec != 1 { t.Fatalf("interval floor not applied: %d", b.IntervalSec) }
	u.RemoveTick(a.Id)
	for _, tk := range u.Ticks { if tk.Id == a.Id { t.Fatalf("remove failed") } }
}
```
Run: `go test ./internal/users/ -run TestUserTicks` → FAIL (undefined).

- [ ] **Step 2: Add the type, field, and helpers**

Add near the `Macros`/`Aliases` fields (userrecord.go ~line 39):
```go
	Ticks          []UserTick            `yaml:"ticks,omitempty"`   // client-run interval timers (web automation panel)
```
And a `UserTick` type + helpers (new block in the same file):
```go
// UserTick is a user-defined real-time-second timer that fires commands from
// a capable client (the web automation panel). Stored per account; executed
// client-side.
type UserTick struct {
	Id          int    `yaml:"id" json:"id"`
	Name        string `yaml:"name" json:"name"`
	Commands    string `yaml:"commands" json:"commands"` // ";"-separated
	IntervalSec int    `yaml:"intervalsec" json:"intervalSec"`
	Enabled     bool   `yaml:"enabled" json:"enabled"`
}

// SetTick creates (Id==0 → assigns next id, appends) or updates (matching Id) a
// tick, enforcing a 1-second interval floor. Returns the stored tick.
func (u *UserRecord) SetTick(t UserTick) UserTick {
	if t.IntervalSec < 1 {
		t.IntervalSec = 1
	}
	if t.Id == 0 {
		maxId := 0
		for _, ex := range u.Ticks {
			if ex.Id > maxId {
				maxId = ex.Id
			}
		}
		t.Id = maxId + 1
		u.Ticks = append(u.Ticks, t)
		return t
	}
	for i := range u.Ticks {
		if u.Ticks[i].Id == t.Id {
			u.Ticks[i] = t
			return t
		}
	}
	u.Ticks = append(u.Ticks, t)
	return t
}

// RemoveTick deletes the tick with the given id, if present.
func (u *UserRecord) RemoveTick(id int) {
	for i := range u.Ticks {
		if u.Ticks[i].Id == id {
			u.Ticks = append(u.Ticks[:i], u.Ticks[i+1:]...)
			return
		}
	}
}
```

- [ ] **Step 3:** `go test ./internal/users/ -run TestUserTicks` → PASS; `go build ./internal/users/...` clean.

- [ ] **Step 4: Commit** (add ONLY `internal/users/userrecord.go internal/users/userrecord_test.go`)
Subject: `feat(users): UserTick storage (per-account interval timers)`

---

### Task 2: Expose ticks in the `Char.Automation` payload

**Files:**
- Modify: `modules/gmcp/gmcp.Automation.go`
- Modify: `modules/gmcp/gmcp.Automation_test.go`

- [ ] **Step 1: Extend the test**

Add to `gmcp.Automation_test.go` a case asserting ticks map through. Update the builder call to the new signature (Step 3):
```go
func TestBuildAutomationPayload_Ticks(t *testing.T) {
	ticks := []users.UserTick{{Id: 1, Name: "sip", Commands: "drink health", IntervalSec: 30, Enabled: true}}
	p := buildAutomationPayload(nil, nil, ticks)
	if len(p.Ticks) != 1 || p.Ticks[0].Id != 1 || p.Ticks[0].IntervalSec != 30 || !p.Ticks[0].Enabled {
		t.Fatalf("tick not mapped: %+v", p.Ticks)
	}
}
```
(Import `internal/users` in the test if not already.)

- [ ] **Step 2: Run → FAIL** (`buildAutomationPayload` arity / `Ticks` field undefined).

- [ ] **Step 3: Add the tick payload type + field + builder param**

In `gmcp.Automation.go`:
```go
type GMCPAutomation_Tick struct {
	Id          int    `json:"id"`
	Name        string `json:"name"`
	Commands    string `json:"commands"`
	IntervalSec int    `json:"intervalSec"`
	Enabled     bool   `json:"enabled"`
}
```
Add `Ticks []GMCPAutomation_Tick `json:"ticks"`` to `GMCPAutomation_Payload`. Change the builder signature to `buildAutomationPayload(macros, aliases map[string]string, ticks []users.UserTick) GMCPAutomation_Payload`, init `Ticks: make([]GMCPAutomation_Tick, 0, len(ticks))`, and append each tick (preserve order — ticks are already an ordered slice). Update `sendAutomation` to pass `user.Ticks`:
```go
Payload: buildAutomationPayload(user.Macros, user.Aliases, user.Ticks),
```

- [ ] **Step 4:** `go test ./modules/gmcp/ -run TestBuildAutomationPayload` (both cases) → PASS; `go build ./modules/gmcp/...` clean.

- [ ] **Step 5: Commit** (add ONLY the two gmcp files)
Subject: `feat(gmcp): include ticks in the Char.Automation payload`

---

### Task 3: Inbound `Char.Automation.Set` / `Remove` (HandleIAC)

**Files:**
- Modify: `modules/gmcp/gmcp.go` (the `switch command` in `HandleIAC`, ~line 272-396)

- [ ] **Step 1: Add the two cases**

Before the `default:` case, add (mirroring the existing user-lookup loop used by `External.Discord`):
```go
		case `Char.Automation.Set`:
			var decoded struct {
				Kind string `json:"kind"`
				users.UserTick
			}
			if err := json.Unmarshal(payload, &decoded); err == nil && decoded.Kind == `tick` {
				if uid := userIdForConnection(connectionId); uid > 0 {
					if u := users.GetByUserId(uid); u != nil {
						u.SetTick(decoded.UserTick)
						events.AddToQueue(events.AutomationChanged{UserId: uid})
					}
				}
			}
		case `Char.Automation.Remove`:
			var decoded struct {
				Kind string `json:"kind"`
				Id   int    `json:"id"`
			}
			if err := json.Unmarshal(payload, &decoded); err == nil && decoded.Kind == `tick` {
				if uid := userIdForConnection(connectionId); uid > 0 {
					if u := users.GetByUserId(uid); u != nil {
						u.RemoveTick(decoded.Id)
						events.AddToQueue(events.AutomationChanged{UserId: uid})
					}
				}
			}
```
(The `kind` gate means Phase 3 just adds a `decoded.Kind == "trigger"` branch — no new package.) The embedded `users.UserTick` unmarshals the flat `{kind,id,name,commands,intervalSec,enabled}` JSON via Go field promotion.

- [ ] **Step 2: Add the small `userIdForConnection` helper** (DRY — the same loop already appears inline for Discord/Hello). In `gmcp.go`:
```go
func userIdForConnection(connectionId uint64) int {
	for _, user := range users.GetAllActiveUsers() {
		if user.ConnectionId() == connectionId {
			return user.UserId
		}
	}
	return 0
}
```
(If `users`/`events` aren't imported in `gmcp.go`, add them — `users` almost certainly is.)

- [ ] **Step 3:** `go build ./modules/gmcp/...` clean.

- [ ] **Step 4: Commit** (add ONLY `modules/gmcp/gmcp.go`)
Subject: `feat(gmcp): inbound Char.Automation.Set/Remove for ticks`

---

### Task 4: Client `SendGMCP` binary-frame helper

**Files:**
- Modify: `_datafiles/html/public/webclient-pure.html` (script, near `SendData`)

- [ ] **Step 1: Add the helper**

Next to `SendData` (~line 1443):
```js
        // Send a client->server GMCP frame as a BINARY ws message so the 0xFF
        // IAC byte survives (a text frame would UTF-8-mangle it). pkg is the
        // package string, obj is JSON-serializable.
        function SendGMCP(pkg, obj) {
            if (!socket || socket.readyState !== WebSocket.OPEN) return false;
            var body = pkg + " " + JSON.stringify(obj);
            var bodyBytes = new TextEncoder().encode(body); // UTF-8
            var frame = new Uint8Array(bodyBytes.length + 5);
            frame[0] = 255; frame[1] = 250; frame[2] = 201;            // IAC SB GMCP
            frame.set(bodyBytes, 3);
            frame[frame.length - 2] = 255; frame[frame.length - 1] = 240; // IAC SE
            socket.send(frame);
            return true;
        }
```

- [ ] **Step 2: Verify** — grep `webclient-pure.html` for `function SendGMCP`. (Manual runtime verification happens in the smoke; you can sanity-check the frame bytes by eye.)

- [ ] **Step 3: Commit** (ONLY `webclient-pure.html`)
Subject: `feat(web): SendGMCP binary-frame helper for client->server GMCP`

---

### Task 5: Ticks tab — render rows with enable toggle + fire-now

**Files:**
- Modify: `_datafiles/html/public/webclient-pure.html` (`renderAutomation`)
- Modify: `_datafiles/html/public/static/css/dashboard.css` (tick-row styles if needed)

- [ ] **Step 1: Render ticks in `renderAutomation()`**

In the `else` branch that currently shows "Coming soon" for ticks/triggers, special-case `'ticks'`: for each `a.ticks[]`, build an `.auto-row` (the Phase-1 row style, distinct from the macro/alias chips because ticks carry name+interval+enabled):
- a `.auto-tog` enable pill (add `.on` class when `tick.enabled`); clicking it toggles enabled (Step: `SendGMCP("Char.Automation.Set", {kind:"tick", ...tick, enabled: !tick.enabled})`).
- `.name` = `tick.name` (textContent); `.sum` = `"every " + tick.intervalSec + "s"` + (disabled ? " (off)" : "").
- `dataset.autoType="tick"`, `dataset.autoId=tick.id`.
- Empty state: "No ticks configured."
Keep the `triggers` tab on "Coming soon."

- [ ] **Step 2: Left-click a tick row → fire now**

Extend the `#auto-list` click handler: if the clicked target is the toggle (`.auto-tog`), toggle enabled (Step 1) and stop. Otherwise if `closest(".auto-row")` with `dataset.autoType==="tick"`, fire its commands now: split `dataset` commands on `;` and `SendData` each part. (Store the commands in `dataset.autoCmd` on the row so the click handler has them.)

- [ ] **Step 3: Verify** — grep for `autoType="tick"`/`autoType = "tick"` and the toggle handler.

- [ ] **Step 4: Commit** (the two files)
Subject: `feat(web): ticks tab — rows with enable toggle + fire-now`

---

### Task 6: Tick `setInterval` runtime

**Files:**
- Modify: `_datafiles/html/public/webclient-pure.html` (script)

- [ ] **Step 1: Add timer management**

Add a module-level `var tickTimers = {};` and a `rebuildTickTimers()` function:
- clear all existing timers (`for id in tickTimers: clearInterval(tickTimers[id])`; reset `tickTimers={}`).
- read `GMCPStructs["Char"].Automation.ticks`; for each **enabled** tick with `intervalSec >= 1`, `tickTimers[id] = setInterval(function(){ commands.split(';').forEach(c => { c = c.trim(); if (c) SendData(c); }); }, intervalSec*1000)` (capture commands per-tick in a closure/IIFE).

- [ ] **Step 2: Rebuild on every Char.Automation push + on load**

Call `rebuildTickTimers()` from the same place `renderAutomation()` is invoked on the `Char.Automation` GMCP update (so add/edit/remove/enable changes restart timers), and once on load. Disabling a tick (or removing it) clears its timer because the rebuilt set omits it.

- [ ] **Step 3: Verify** — grep for `rebuildTickTimers` and that it's called alongside `renderAutomation` on the Char.Automation handler.

- [ ] **Step 4: Commit** (ONLY `webclient-pure.html`)
Subject: `feat(web): client tick timer runtime (setInterval per enabled tick)`

---

### Task 7: Tick editor (add / edit / remove via GMCP)

**Files:**
- Modify: `_datafiles/html/public/webclient-pure.html` (editors)

- [ ] **Step 1: Build the tick editor form** (into the existing modal): Name, Commands (textarea, `;`-separated), Every N seconds (number, min 1), Enabled (toggle). **Save** → `SendGMCP("Char.Automation.Set", {kind:"tick", id: <id or 0 for new>, name, commands, intervalSec, enabled})`. **Remove** (context menu) → `SendGMCP("Char.Automation.Remove", {kind:"tick", id})`. **Duplicate** → open editor pre-filled with the same fields and `id: 0` (new).

- [ ] **Step 2: Wire "+ New" + context menu for the ticks tab**

When the active tab is `ticks`, "+ New" opens the empty tick editor (replace the Phase-1 "coming soon" toast for ticks). The right-click context menu's Edit/Duplicate/Remove dispatch to the tick editor / GMCP remove when `dataset.autoType==="tick"` (read the tick's current fields from `GMCPStructs["Char"].Automation.ticks` by `dataset.autoId`).

- [ ] **Step 3:** After Save/Remove no manual refresh — the server emits `AutomationChanged` → `Char.Automation` re-push → `renderAutomation()` + `rebuildTickTimers()`.

- [ ] **Step 4: Verify** — grep for `Char.Automation.Set` and `Char.Automation.Remove` in the client; confirm the tick editor builds the `{kind:"tick",...}` object.

- [ ] **Step 5: Commit** (ONLY `webclient-pure.html`)
Subject: `feat(web): tick editor (add/edit/remove/duplicate via inbound GMCP)`

---

### Task 8: Help topic + context docs

**Files:**
- Create: `_datafiles/world/dogmud/templates/help/ticks.template`
- Modify: `modules/gmcp/context.md` (+ `internal/users/context.md` if it exists)

- [ ] **Step 1: `help ticks` topic**

Create `ticks.template` in the ANSI-templated style of the other help files (READ `macros.template` for the house style). Cover: what a tick is (a timer that runs command(s) every N seconds while you're connected to a capable client / the web client); that they're managed in the **Triggers & Timers** panel; the enable/disable toggle and minimum 1-second interval; that definitions save to your account and sync, but only run in the web client.

- [ ] **Step 2: context docs**

Update `modules/gmcp/context.md`: note the outbound `Char.Automation` payload now includes `ticks`, and document the **inbound** `Char.Automation.Set`/`Remove` handler (tick kind; trigger kind in Phase 3) in `HandleIAC`. If `internal/users/context.md` exists, note the new `UserTick`/`Ticks` storage + `SetTick`/`RemoveTick`.

- [ ] **Step 3: Verify** — `ls` the new template; grep `context.md` for `Char.Automation.Set`.

- [ ] **Step 4: Commit** (add ONLY the named files)
Subject: `docs: help ticks topic + context for inbound Char.Automation + UserTick`

---

### Task 9: Boot + browser smoke

**Files:** none (verification only)

- [ ] **Step 1:** `go build ./...` clean; boot (`timeout 90 go run . > /tmp/p2boot.log 2>&1`) and confirm `Server Ready` with no panic (the new help template loads; the inbound handler + storage compile in).

- [ ] **Step 2: Browser smoke** (hard-refresh `/webclient`, log in):
- Ticks tab: empty state initially; **+ New** opens the editor; create a tick ("sip" = `drink health`, every 5s, enabled) → it appears as a row with the toggle on; **the command fires every 5s** in the feed; **left-click** the row fires it immediately; the **toggle** pauses/resumes it (feed stops/starts); **Edit/Duplicate/Remove** work; all of it survives **relog** (server-persisted) and a **hard-refresh** (GMCP re-push repopulates + timers rebuild).
- Confirm the GMCP round-trip: a Set from the panel reaches the server (the tick persists) — i.e. the binary `SendGMCP` frame is parsed by `HandleIAC`.
- Macros/Aliases tabs still work (no regression). No console errors.

- [ ] **Step 3:** Record results; if clean, ready for finishing-a-development-branch.

---

## Self-Review

**Spec coverage (Phase 2):** `user.Ticks` storage (Task 1) ✓; outbound payload (Task 2) ✓; inbound GMCP CRUD (Task 3) ✓; `SendGMCP` helper (Task 4) ✓; ticks render + toggle + fire-now (Task 5) ✓; `setInterval` runtime (Task 6) ✓; tick editor (Task 7) ✓; `help ticks` + context docs (Task 8) ✓; smoke (Task 9) ✓. Real-time-second interval with a 1s floor; client-executed; server-persisted — all per spec.

**Placeholder scan:** all steps carry concrete code/commands; the one area told to mirror an existing template (`ticks.template` house style, context.md tone) references a concrete file to copy from.

**Identifier consistency:** `UserTick`{Id,Name,Commands,IntervalSec,Enabled} identical across userrecord.go (Task 1), the gmcp payload tick (Task 2 — same json tags), the inbound decode (Task 3, embedded `users.UserTick`), and the client editor object (Task 7, `{kind:"tick",...}`). `buildAutomationPayload(macros, aliases, ticks)` signature consistent between the test (Task 2 Step 1) and impl (Step 3) and caller `sendAutomation`. `Char.Automation.Set/Remove` strings identical server (Task 3) ↔ client (Tasks 5/7). `rebuildTickTimers` defined (Task 6) and called with `renderAutomation` (Tasks 6) — same Char.Automation push path established in Phase 1.
