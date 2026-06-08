# Automation Panel — Phase 1 Implementation Plan (Panel shell + Macros & Aliases)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stand up the automation panel in the reserved `#panel-trig` slot with all four tabs, fully wiring the two systems that already exist server-side — **Macros** and **Aliases** — end-to-end (GMCP read → list → left-click fire → modal add/edit/remove via existing commands). Ticks & Triggers tabs render a "coming soon" placeholder until their own phases.

**Architecture:** Server-stores / client-executes. A new `Char.Automation` GMCP module reads `user.Macros` + `user.Aliases` and pushes them on login and on change; the client renders tabs/lists/modals and persists edits by firing the existing `set =N` / `alias name=value` commands (one canonical write path, telnet-compatible). No new storage in Phase 1.

**Tech Stack:** Go (GMCP module + events), vanilla JS/CSS web client (xterm dashboard), GMCP event bus.

**Spec:** `docs/superpowers/specs/2026-06-07-web-client-automation-panel-design.md` (this implements Phase 1 of its phasing section).

---

## File Structure

- **Create:** `modules/gmcp/gmcp.Automation.go` — new GMCP module emitting `Char.Automation` (macros + aliases now; ticks/triggers added in Phases 2–3).
- **Modify:** `internal/events/eventtypes.go` — add an `AutomationChanged` event.
- **Modify:** `internal/usercommands/set.go` — emit `AutomationChanged` after a macro write/delete (`cmdSetMacro`).
- **Modify:** `internal/usercommands/alias.go` — emit `AutomationChanged` after an alias add/remove.
- **Modify:** `_datafiles/html/public/webclient-pure.html` — replace the `#panel-trig` placeholder with the panel (tabs + lists + modal); render from `GMCPStructs["Char"].Automation`; wire fire/edit/remove.
- **Modify:** `_datafiles/html/public/static/css/dashboard.css` — panel/tab/list/modal styles (leather theme).

**Verification note:** Go changes have a test suite + boot test; front-end changes verify via structural greps + a browser smoke (no JS test runner). `git add` ONLY named files — the working tree has unrelated world-state files; never `git add -A`. Co-author trailer on every commit: `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`.

---

### Task 1: Add the `AutomationChanged` event

**Files:**
- Modify: `internal/events/eventtypes.go`

- [ ] **Step 1: Add the event type**

Find an existing simple event in `eventtypes.go` (e.g. `BuffsTriggered`) to mirror the shape, and add:
```go
// AutomationChanged fires when a user's macros/aliases/ticks/triggers change,
// so the Char.Automation GMCP payload can be re-pushed.
type AutomationChanged struct {
	UserId int
}

func (a AutomationChanged) Type() string { return `AutomationChanged` }
```
(Match the exact `Type()`/interface convention used by the neighboring event structs — confirm via codegraph `codegraph_node BuffsTriggered`.)

- [ ] **Step 2: Build to verify the event compiles**

Run: `go build ./internal/events/...`
Expected: clean build.

- [ ] **Step 3: Commit**
```
git add internal/events/eventtypes.go
git commit -m "feat(events): AutomationChanged event for the automation GMCP

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: `Char.Automation` GMCP module (read path)

**Files:**
- Create: `modules/gmcp/gmcp.Automation.go`
- Test: `modules/gmcp/gmcp.Automation_test.go`

Mirror the skeleton of an existing small module — **read `modules/gmcp/gmcp.Comm.go` first** for the exact module-registration boilerplate (struct, `init`/register, `RegisterListener`, payload send helper). Verify the payload-send helper name via codegraph (`codegraph_node` on the Comm module's send function).

- [ ] **Step 1: Write the payload builder test (TDD)**

Create `gmcp.Automation_test.go`:
```go
package gmcp

import "testing"

func TestBuildAutomationPayload_MacrosAndAliases(t *testing.T) {
	macros := map[string]string{"=1": "get all;wield sword"}
	aliases := map[string]string{"kk": "kick goblin"}
	p := buildAutomationPayload(macros, aliases)
	if len(p.Macros) != 1 || p.Macros[0].Key != "=1" || p.Macros[0].Commands != "get all;wield sword" {
		t.Fatalf("macro not mapped: %+v", p.Macros)
	}
	if len(p.Aliases) != 1 || p.Aliases[0].Name != "kk" || p.Aliases[0].Command != "kick goblin" {
		t.Fatalf("alias not mapped: %+v", p.Aliases)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./modules/gmcp/ -run TestBuildAutomationPayload`
Expected: FAIL (undefined `buildAutomationPayload`).

- [ ] **Step 3: Implement the module + payload builder**

Create `gmcp.Automation.go` with:
- Payload structs:
```go
type GMCPAutomation_Macro struct {
	Key      string `json:"key"`      // "=1"
	Commands string `json:"commands"` // ";"-joined
}
type GMCPAutomation_Alias struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}
type GMCPAutomation_Payload struct {
	Macros  []GMCPAutomation_Macro `json:"macros"`
	Aliases []GMCPAutomation_Alias `json:"aliases"`
	// Ticks/Triggers added in Phases 2-3.
}
```
- A pure builder (sorted for stable output):
```go
func buildAutomationPayload(macros, aliases map[string]string) GMCPAutomation_Payload {
	p := GMCPAutomation_Payload{}
	mkeys := make([]string, 0, len(macros))
	for k := range macros { mkeys = append(mkeys, k) }
	sort.Strings(mkeys)
	for _, k := range mkeys { p.Macros = append(p.Macros, GMCPAutomation_Macro{Key: k, Commands: macros[k]}) }
	akeys := make([]string, 0, len(aliases))
	for k := range aliases { akeys = append(akeys, k) }
	sort.Strings(akeys)
	for _, k := range akeys { p.Aliases = append(p.Aliases, GMCPAutomation_Alias{Name: k, Command: aliases[k]}) }
	return p
}
```
- Module registration mirroring `gmcp.Comm.go`: register listeners for `events.PlayerSpawn{}` (login push) and `events.AutomationChanged{}` (change push); in each handler, load the user, call `buildAutomationPayload(user.Macros, user.Aliases)`, and send it under the identifier **`Char.Automation`** using the same send helper the other modules use. (Look up the user by `UserId` the way `gmcp.Comm.go` / `gmcp.Char.go` handlers do.)

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./modules/gmcp/ -run TestBuildAutomationPayload`
Expected: PASS.

- [ ] **Step 5: Build the whole module**

Run: `go build ./modules/gmcp/...`
Expected: clean.

- [ ] **Step 6: Commit**
```
git add modules/gmcp/gmcp.Automation.go modules/gmcp/gmcp.Automation_test.go
git commit -m "feat(gmcp): Char.Automation module exposing macros + aliases

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: Emit `AutomationChanged` on macro/alias writes

**Files:**
- Modify: `internal/usercommands/set.go` (`cmdSetMacro`, ~line 501)
- Modify: `internal/usercommands/alias.go` (`Alias`, after `AddCommandAlias`, ~line 27)

- [ ] **Step 1: Emit after a macro change**

In `cmdSetMacro`, after the macro map is mutated (both the delete branch ~line 521 and the set branch), queue the event:
```go
events.AddToQueue(events.AutomationChanged{UserId: user.UserId})
```
(`set.go` already imports `events`; confirm. Place it once before `return true, nil` so both branches cover it.)

- [ ] **Step 2: Emit after an alias change**

In `alias.go`, after `addedAlias, deletedAlias := user.AddCommandAlias(...)` and the messaging, before `return true, nil` (~line 35), add the same:
```go
events.AddToQueue(events.AutomationChanged{UserId: user.UserId})
```

- [ ] **Step 3: Build to verify**

Run: `go build ./internal/usercommands/...`
Expected: clean. (If `events` isn't imported in `alias.go`, add it.)

- [ ] **Step 4: Commit**
```
git add internal/usercommands/set.go internal/usercommands/alias.go
git commit -m "feat(commands): emit AutomationChanged on macro/alias edits

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: Client panel shell (tabs + list containers)

**Files:**
- Modify: `_datafiles/html/public/webclient-pure.html` (the `#panel-trig` section, ~line 304)
- Modify: `_datafiles/html/public/static/css/dashboard.css` (append panel styles)

- [ ] **Step 1: Replace the placeholder body with the tab strip + list host**

Replace the `#panel-trig` `.dash-panel-body` placeholder content (`Triggers &amp; timers — coming soon`) with:
```html
<div class="auto-tabs">
  <span class="auto-tab" data-tab="ticks">Ticks</span>
  <span class="auto-tab" data-tab="triggers">Triggers</span>
  <span class="auto-tab active" data-tab="macros">Macros</span>
  <span class="auto-tab" data-tab="aliases">Aliases</span>
</div>
<div class="auto-addbar"><span class="auto-add" title="Add">+ New</span></div>
<div id="auto-list" class="auto-list"></div>
```
(Default to the Macros tab so Phase 1 shows working content immediately.)

- [ ] **Step 2: Add CSS (append to dashboard.css)**

Append leather-themed styles reusing existing tokens: `.auto-tabs` (flex strip), `.auto-tab`/`.auto-tab.active` (madder underline on active, matching the right column), `.auto-addbar`/`.auto-add` (brass button), `.auto-list` (scrolling column), `.auto-row` (brass-bordered tile, gold hover), `.auto-row .name` (IBM Plex Mono), `.auto-row .sum` (muted ellipsis), `.auto-tog`/`.auto-tog.on` (the enable pill, used in later phases). Match `.inv-*` patterns already in the file.

- [ ] **Step 3: Verify markup + styles present**

Grep `webclient-pure.html` for `auto-tabs` and `data-tab="macros"`; grep `dashboard.css` for `.auto-tab` and `.auto-row`.
Expected: present.

- [ ] **Step 4: Commit**
```
git add _datafiles/html/public/webclient-pure.html _datafiles/html/public/static/css/dashboard.css
git commit -m "feat(web): automation panel shell (tabs + list) in #panel-trig

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: Render Macros & Aliases lists + left-click fire

**Files:**
- Modify: `_datafiles/html/public/webclient-pure.html` (script section)

- [ ] **Step 1: Add a `renderAutomation()` JS function + GMCP hook**

Add a function that reads `GMCPStructs["Char"] && GMCPStructs["Char"].Automation`, reads the active tab from a module-level `autoActiveTab` (default `'macros'`), and rebuilds `#auto-list`:
- **Macros tab:** one `.auto-row` per `automation.macros[]` — name = the key (e.g. `=1`), summary = the commands string. Use `textContent`/`dataset` for all server strings (XSS-safe, per the inventory panel convention).
- **Aliases tab:** one row per `automation.aliases[]` — name = alias name, summary = the command.
- **Ticks/Triggers tabs:** render a single muted "Coming soon" line (placeholder until Phases 2–3).
- **Left-click a row → fire now:** macros → `SendData("=" + key.replace('=',''))` (i.e. send `=1`); aliases → `SendData(aliasName)` (server expands it).
- Wire tab clicks to set `autoActiveTab` + re-render; wire it into the existing `Char` GMCP update handler so the panel refreshes on every `Char.Automation` push (mirror how `renderInventory()` is invoked).

- [ ] **Step 2: Verify the function + wiring**

Grep `webclient-pure.html` for `renderAutomation` and `GMCPStructs["Char"].Automation`.
Expected: present, and invoked from the Char GMCP update path.

- [ ] **Step 3: Commit**
```
git add _datafiles/html/public/webclient-pure.html
git commit -m "feat(web): render macros/aliases lists + left-click fire

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 6: Modal editor framework + right-click context menu

**Files:**
- Modify: `_datafiles/html/public/webclient-pure.html` (script + a modal root element)
- Modify: `_datafiles/html/public/static/css/dashboard.css` (modal + context-menu styles)

- [ ] **Step 1: Add a reusable modal overlay**

Add a hidden modal root (`#auto-modal` with a `.scrim` backdrop + `.auto-modal` card: header, body slot, footer with Cancel/Save) and JS helpers `openAutoModal(title, bodyEl, onSave)` / `closeAutoModal()`. Backdrop click + Cancel close; Save calls `onSave()` then closes. Style in dashboard.css reusing leather tokens (mirror the spec mock `trig-panel-v2.html`).

- [ ] **Step 2: Add the right-click context menu**

On `.auto-row` `contextmenu`, show a small menu (Edit / Duplicate / Remove) positioned at the cursor — reuse the existing `.inv-ctx` menu pattern from the inventory panel if present, else add `.auto-ctx`. Each item dispatches to handlers added in Task 7.

- [ ] **Step 3: Verify**

Grep for `openAutoModal` and `auto-ctx` (or reused `inv-ctx`) in the source.
Expected: present.

- [ ] **Step 4: Commit**
```
git add _datafiles/html/public/webclient-pure.html _datafiles/html/public/static/css/dashboard.css
git commit -m "feat(web): modal editor framework + row context menu for automation

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 7: Macro & Alias editor forms (add / edit / remove / duplicate)

**Files:**
- Modify: `_datafiles/html/public/webclient-pure.html` (script)

- [ ] **Step 1: Macro editor**

Build a form (Number 1–9 + Commands textarea) injected into the modal. **Save** → `SendData('set =' + num + ' ' + commands)`. (Empty commands + Save on an existing macro = delete, since `set =N` with no value deletes.) **Remove** (context menu) → `SendData('set =' + num + ' ')`. **Duplicate** → open the editor pre-filled with the same commands and the next free number.

- [ ] **Step 2: Alias editor**

Build a form (Alias name + Expands-to command). **Save** → `SendData('alias ' + name + '=' + command)`. **Remove** → `SendData('alias ' + name + '=')`. **Duplicate** → editor pre-filled, name suffixed (e.g. `kk2`).
- Wire the "+ New" button to open the correct empty editor for the active tab (macros/aliases in Phase 1; ticks/triggers show a "coming soon" toast).
- After Save, no manual refresh needed — the server emits `AutomationChanged` → `Char.Automation` re-push → `renderAutomation()`.

- [ ] **Step 3: Verify**

Grep for `set =` and `'alias '` builders in the source.
Expected: present.

- [ ] **Step 4: Commit**
```
git add _datafiles/html/public/webclient-pure.html
git commit -m "feat(web): macro + alias add/edit/remove/duplicate editors

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 8: Context docs (`context.md`)

**Files:**
- Modify: `internal/events/context.md`
- Modify: `internal/usercommands/context.md`
- Create: `modules/gmcp/context.md`

- [ ] **Step 1: Document the new event**

In `internal/events/context.md`, add an entry for `AutomationChanged` matching the format of the neighbouring event entries already in that file (name, fields, when it fires, who listens). One short entry: fires when a user's macros/aliases/ticks/triggers change; the `Char.Automation` GMCP module listens and re-pushes.

- [ ] **Step 2: Note the command emissions**

In `internal/usercommands/context.md`, add a line noting that `cmdSetMacro` (`set.go`) and `Alias` (`alias.go`) now emit `events.AutomationChanged{UserId}` so the web automation panel refreshes.

- [ ] **Step 3: Create the gmcp context doc**

Create `modules/gmcp/context.md` documenting the **`Char.Automation`** module (this package had no context.md; scope this file to the automation module + a one-line pointer that other `gmcp.*` modules follow the same listener/payload pattern). Cover: the outbound read payload (`{macros, aliases}` now; `ticks, triggers` added in Phases 2–3), the push triggers (`PlayerSpawn` + `AutomationChanged`), and a forward note that **inbound** `Char.Automation.Set`/`Remove` (ticks/triggers CRUD) lands in Phases 2–3 via `HandleIAC` (cross-ref the spec).

- [ ] **Step 4: Commit**
```
git add internal/events/context.md internal/usercommands/context.md modules/gmcp/context.md
git commit -m "docs(context): Char.Automation GMCP module + AutomationChanged event

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 9: Help files (macros/alias panel cross-reference)

**Files:**
- Modify: `_datafiles/world/dogmud/templates/help/macros.template`
- Modify: `_datafiles/world/dogmud/templates/help/alias.template`

- [ ] **Step 1: Cross-reference the panel in `macros.template`**

Before the final `See also:` line, add (80-col, matching the file's `<ansi>` style):
```
  <ansi fg="black-bold">Web client:</ansi> your macros also appear in the
  <ansi fg="command">Triggers &amp; Timers</ansi> panel, where you can add, edit,
  and run them with the mouse.

```

- [ ] **Step 2: Cross-reference the panel in `alias.template`**

Before its final `See also:` line, add the same-style note:
```
  <ansi fg="black-bold">Web client:</ansi> your custom aliases also appear in the
  <ansi fg="command">Triggers &amp; Timers</ansi> panel for point-and-click
  editing.

```

- [ ] **Step 3: Verify**

Grep both templates for `Triggers &amp; Timers`.
Expected: present in each. (No new helpfiles in Phase 1 — `help ticks`/`help triggers` topics are Phases 2–3.)

- [ ] **Step 4: Commit**
```
git add _datafiles/world/dogmud/templates/help/macros.template _datafiles/world/dogmud/templates/help/alias.template
git commit -m "docs(help): cross-reference the web automation panel from macros/alias help

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 10: Boot + browser smoke

**Files:** none (verification only)

- [ ] **Step 1: Build + boot the server**

Run: `go build ./...` (expect clean), then boot locally and confirm it starts past data-file load + GMCP module registration with no panic (per the pre-push SOP — watch for clean `LoadDataFiles` lines).

- [ ] **Step 2: Browser smoke**

Hard-refresh `/webclient`, log in, open the Triggers & Timers panel:
- Macros & Aliases tabs list the account's real macros/aliases (cross-check with the `macros` / `alias` commands in the feed).
- Left-click a macro fires `=N`; left-click an alias runs its expansion.
- "+ New" + the modal: add a macro and an alias; they appear in the list immediately (GMCP refresh) and persist after relog. Edit, Duplicate, Remove all work.
- Ticks/Triggers tabs show the "coming soon" placeholder. No console errors.

- [ ] **Step 3: Record result**

Note any issues; if clean, Phase 1 is ready for finishing-a-development-branch (or proceed to the Phase 2 plan).

---

## Appendix — Ticks/Triggers write path (inbound GMCP, for Phases 2–3)

**SUPERSEDES the earlier command-string idea.** Inbound client→server GMCP is verified available (`gmcp.go` `HandleIAC` switch already handles `Core.Hello`, `Core.Supports.Set/Remove`, `Char.Login`; inbound ws → `HandleInput` → the registered IAC handler). Since ticks/triggers are *client-executed* (web-only), there is **no telnet command** for them — CRUD goes over GMCP:

- **Server (Phase 2 adds, Phase 3 reuses):** new cases in `HandleIAC` — `Char.Automation.Set` (payload = one full tick/trigger object; absent `id` = create, present = update) and `Char.Automation.Remove` (`{kind, id}`). Each unmarshals JSON, finds the user by `connectionId` (existing `users.GetAllActiveUsers()` loop), mutates `user.Ticks`/`user.Triggers`, and queues `events.AutomationChanged{UserId}` so the read payload re-pushes.
- **Client (Phase 2 adds):** a `SendGMCP(pkg, obj)` helper sending a **binary** ws frame (so the `0xFF` IAC byte isn't UTF-8-mangled): `socket.send(new Uint8Array([255,250,201, ...utf8(pkg+" "+JSON.stringify(obj)), 255,240]))`. Save → `SendGMCP("Char.Automation.Set", {kind,…})`; Remove → `Char.Automation.Remove`; Duplicate → Set a copy without an `id`.
- No quoting scheme, no command parser. Macros/aliases keep their existing `set`/`alias` commands (Phase 1, unchanged).

## Later phases (separate plans, written when each predecessor lands)

- **Phase 2 — Ticks:** `user.Ticks` storage (+ user-save), extend the outbound `Char.Automation` payload with ticks, add the **inbound `Char.Automation.Set/Remove` handler + the `SendGMCP` client helper** (first phase to need them), the client `setInterval` runtime + tick row (enable toggle, left-click fire-now) + tick editor + `help ticks` topic.
- **Phase 3 — Triggers:** `user.Triggers` storage (+ `Condition` struct), extend the payload, **reuse** the Phase-2 inbound GMCP handler/helper for trigger CRUD, the vitals-GMCP reserved/available addition, the client stream-tap match/condition engine (with the runaway cooldown) + builder editor + `help triggers` topic (with the explicit **available-pool %** wording).

---

## Self-Review

**Spec coverage (Phase 1 scope):** panel shell (Task 4) ✓; Macros/Aliases tabs render + fire (Task 5) ✓; modal + context menu (Task 6) ✓; add/edit/remove/duplicate via existing commands (Task 7) ✓; `Char.Automation` GMCP read + change-push (Tasks 1–3) ✓; context docs (Task 8) ✓; macros/alias help panel cross-ref (Task 9) ✓; boot + smoke (Task 10) ✓. Ticks/Triggers correctly deferred to their phases with placeholders (incl. their `help ticks`/`help triggers` topics).

**Placeholder scan:** no "TBD"/vague steps; commands and code shown. GMCP module boilerplate intentionally references `gmcp.Comm.go` as the concrete template + codegraph verification (the one area not fully quoted, because the module-registration skeleton must match the repo's exact pattern).

**Identifier consistency:** `AutomationChanged` (Task 1) used in Tasks 2–3; `buildAutomationPayload` signature identical in Task 2 test + impl; GMCP identifier `Char.Automation` consistent server (Task 2) ↔ client `GMCPStructs["Char"].Automation` (Task 5); `set =N` / `alias name=value` write forms consistent with the verified set.go/alias.go behavior.
