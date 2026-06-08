# In-Game Client Dashboard (Sub-project #1) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the in-game web client's bare xterm + floating WinBox panels with a docked, fluid-responsive, rearrangeable dashboard (arrangement C), delivering the framework + six core panels and reserved slots for the later Scene-Art (#2) and Triggers (#3) cards.

**Architecture:** A CSS-grid dashboard in `webclient-pure.html`, styled by a new `dashboard.css` (reusing `gomud.css` leather/brass tokens) and driven by a new `dashboard.js` (splitters, drag-to-swap, collapse, pop-out, responsive tab/drawer, localStorage persistence). Existing renderers are reused unchanged — xterm feed, `RoomGridSVG` map, vitals rows, comm — re-homed from WinBox windows into docked panel bodies. WinBox stays loaded **only** for pop-out.

**Tech Stack:** vanilla JS + CSS Grid, xterm.js (existing), WinBox (existing, pop-out only), GMCP handlers (existing). No build step; no new dependencies. No Go changes in this sub-project.

**Spec:** `docs/superpowers/specs/2026-06-06-web-client-dashboard-design.md`. Visual source of truth: `docs/superpowers/specs/2026-06-06-web-client-dashboard-mockups/`.

**Branch:** `feature/web-client-dashboard` (created; spec committed).

**Verification note (all tasks):** No automated UI tests. Verification per task = (a) the file changes are structurally sound (HTML/JS parse; `node --check` for JS files), and (b) a human visual smoke in `/webclient`. Subagents CANNOT use a browser; they verify by `node --check` on `dashboard.js`, structural self-review, and confirming no leftover references. A dev server may hold ports 80/55555 — do NOT boot another; the human reloads `/webclient` to smoke. `gomud.css`/`dashboard.css`/`dashboard.js`/`webclient-pure.html` are static assets served from disk live (hard-refresh picks them up; `webclient-pure.html` is re-read per request).

## File structure

- `_datafiles/html/public/webclient-pure.html` — restructured DOM (grid container + panel cards); links `dashboard.css` + `dashboard.js`; keeps xterm/GMCP/sound code.
- `_datafiles/html/public/static/css/dashboard.css` — NEW: grid layout (arrangement C), panel chrome, splitters, responsive reflow, tab/drawer. Uses `gomud.css` `:root` tokens.
- `_datafiles/html/public/static/js/dashboard.js` — NEW: `Dashboard` controller (init, splitters, swap, collapse, pop-out, responsive mode, persistence).
- Reused as-is: `static/js/gmcp.js` (`RoomGridSVG`), WinBox, xterm.

Panel ids (used across CSS/JS): `panel-map`, `panel-art`, `panel-trig`, `panel-feed`, `panel-cmd`, `panel-vitals`, `panel-status`, `panel-chat`, plus the `dash-session` strip. Each panel: `<section class="dash-panel" id="panel-X" data-panel="X" style="grid-area:X">` with a `.dash-panel-head` (title + `▾`/`⧉` buttons) and `.dash-panel-body`.

---

### Task 1: Grid shell + panel chrome + migrate renderers into docked panels

Delivers a static docked arrangement-C dashboard (no splitters/drag/responsive yet) replacing the floating WinBox panels. Biggest structural task.

**Files:**
- Create: `_datafiles/html/public/static/css/dashboard.css`
- Modify: `_datafiles/html/public/webclient-pure.html`

- [ ] **Step 1: Create `dashboard.css` with the grid + panel chrome**

Write to `_datafiles/html/public/static/css/dashboard.css`:

```css
/* In-game dashboard — arrangement C (visuals left). Reuses gomud.css tokens. */
#dashboard {
  display: grid;
  height: 100%;
  width: 100%;
  gap: var(--dash-gap, 6px);
  padding: 6px;
  box-sizing: border-box;
  background: var(--bg-base, #191310);
  /* columns: left visuals · center feed (dominant) · right social */
  grid-template-columns:
    minmax(200px, var(--col-l, 0.9fr))
    minmax(0, var(--col-c, 2fr))
    minmax(200px, var(--col-r, 0.9fr));
  grid-template-rows:
    auto                              /* session strip */
    minmax(0, var(--row-t, 1fr))
    minmax(0, var(--row-b, 1fr))
    auto;                             /* command bar */
  grid-template-areas:
    "sess sess sess"
    "map  feed vitals"
    "art  feed status"
    "trig cmd  chat";
}

/* Session strip */
#dash-session {
  grid-area: sess;
  display: flex; align-items: center; gap: 12px;
  padding: 4px 10px;
  background: linear-gradient(var(--bar-top, #241813), var(--bar-bottom, #1a110b));
  border: 1px solid var(--panel-border, #3a2a18);
  border-radius: 5px;
  font-family: var(--font-serif, Georgia, serif); font-size: 0.85rem;
  color: var(--text-primary, #d2c3a4);
}
#dash-session .sess-name { color: var(--title-gold, #e8d2a0); font-style: italic; font-weight: bold; }
#dash-session .sess-conn { color: var(--online-green, #7fae6a); }
#dash-session .sess-spacer { margin-left: auto; }
#dash-session button { font-size: 0.75rem; }

/* Panel shell */
.dash-panel {
  display: flex; flex-direction: column; min-height: 0; min-width: 0;
  background: var(--panel-bg, #201913);
  border: 1px solid var(--panel-border, #3a2a18);
  border-radius: 6px;
  box-shadow: inset 0 1px 0 rgba(201,168,106,0.10);
  overflow: hidden;
}
.dash-panel-head {
  display: flex; align-items: center; gap: 6px;
  padding: 4px 8px;
  background: linear-gradient(var(--bar-top, #241813), var(--bar-bottom, #1a110b));
  border-bottom: 1px solid var(--panel-border, #3a2a18);
  cursor: grab;            /* header is the drag handle for swap */
  user-select: none;
}
.dash-panel-head .ph-title {
  font-family: var(--font-serif, Georgia, serif); font-style: italic;
  font-size: 0.82rem; color: var(--title-gold, #e8d2a0);
}
.dash-panel-head .ph-btns { margin-left: auto; display: flex; gap: 4px; }
.dash-panel-head .ph-btn {
  width: 18px; height: 18px; line-height: 16px; text-align: center;
  font-size: 11px; cursor: pointer; border-radius: 3px;
  color: var(--brass-text, #3b2a10);
  background: var(--brass-grad, radial-gradient(circle at 34% 26%, #f4dd92, #cb9f42 46%, #8a6620));
  border: 1px solid var(--brass-border, #5e431a);
}
.dash-panel-body { flex: 1 1 auto; min-height: 0; overflow: auto; position: relative; }

/* Feed + command regions */
#panel-feed { grid-area: feed; }
#panel-feed .dash-panel-body { overflow: hidden; }   /* xterm manages its own scroll */
#panel-cmd  { grid-area: cmd; }
#panel-cmd .dash-panel-body { display: flex; padding: 4px; }
#panel-map { grid-area: map; } #panel-art { grid-area: art; } #panel-trig { grid-area: trig; }
#panel-vitals { grid-area: vitals; } #panel-status { grid-area: status; } #panel-chat { grid-area: chat; }

/* Reserved slots (sub-projects #2/#3) */
.dash-slot .dash-panel-body {
  display: flex; align-items: center; justify-content: center;
  color: var(--text-secondary, #9a8a6a); font-family: var(--font-serif, Georgia, serif);
  font-style: italic; font-size: 0.8rem; text-align: center; padding: 8px;
}

/* Collapsed panels show only the header */
.dash-panel.collapsed .dash-panel-body { display: none; }
```

- [ ] **Step 2: Restructure `webclient-pure.html` DOM into the grid**

Read `webclient-pure.html` first. Replace the `#main-container`/`#terminal`/`#input-area` body region with a `#dashboard` grid. Keep the sound icon (`#menu-icon`) + `#floating-menu` and the `#connect-button`. The xterm still mounts to a `#terminal` div — now inside `#panel-feed .dash-panel-body`; the command input (`#command-input`) moves into `#panel-cmd .dash-panel-body`.

New body structure (inside `<body>`, replacing the old `#main-container` + `#input-area`):

```html
  <div id="dashboard">
    <div id="dash-session">
      <span class="sess-name" id="sess-name">—</span>
      <span class="sess-conn" id="sess-conn">● Connected</span>
      <span class="sess-spacer"></span>
      <button class="brass" id="btn-reconnect" onclick="reconnect()">Reconnect</button>
      <button class="brass" id="btn-reset-layout">Reset layout</button>
    </div>

    <section class="dash-panel" id="panel-map" data-panel="map">
      <div class="dash-panel-head"><span class="ph-title">Map</span>
        <span class="ph-btns"><span class="ph-btn ph-collapse">▾</span><span class="ph-btn ph-popout">⧉</span></span></div>
      <div class="dash-panel-body"><div id="map-render" style="width:100%;height:100%"></div></div>
    </section>

    <section class="dash-panel dash-slot" id="panel-art" data-panel="art">
      <div class="dash-panel-head"><span class="ph-title">Scene</span>
        <span class="ph-btns"><span class="ph-btn ph-collapse">▾</span></span></div>
      <div class="dash-panel-body">Scene art — coming soon</div>
    </section>

    <section class="dash-panel dash-slot" id="panel-trig" data-panel="trig">
      <div class="dash-panel-head"><span class="ph-title">Triggers</span>
        <span class="ph-btns"><span class="ph-btn ph-collapse">▾</span></span></div>
      <div class="dash-panel-body">Triggers &amp; timers — coming soon</div>
    </section>

    <section class="dash-panel" id="panel-feed" data-panel="feed">
      <div class="dash-panel-head"><span class="ph-title">Game</span>
        <span class="ph-btns"><span class="ph-btn ph-popout">⧉</span></span></div>
      <div class="dash-panel-body"><div id="terminal"></div></div>
    </section>

    <section class="dash-panel" id="panel-cmd" data-panel="cmd">
      <div class="dash-panel-body">
        <input type="text" id="command-input" placeholder="Enter command..." style="flex:1">
      </div>
    </section>

    <section class="dash-panel" id="panel-vitals" data-panel="vitals">
      <div class="dash-panel-head"><span class="ph-title">Vitals</span>
        <span class="ph-btns"><span class="ph-btn ph-collapse">▾</span><span class="ph-btn ph-popout">⧉</span></span></div>
      <div class="dash-panel-body"><div id="vitals-bars"></div></div>
    </section>

    <section class="dash-panel" id="panel-status" data-panel="status">
      <div class="dash-panel-head"><span class="ph-title">Status &amp; Conditions</span>
        <span class="ph-btns"><span class="ph-btn ph-collapse">▾</span><span class="ph-btn ph-popout">⧉</span></span></div>
      <div class="dash-panel-body"><div id="status-conditions"></div></div>
    </section>

    <section class="dash-panel" id="panel-chat" data-panel="chat">
      <div class="dash-panel-head"><span class="ph-title">Chat</span>
        <span class="ph-btns"><span class="ph-btn ph-collapse">▾</span><span class="ph-btn ph-popout">⧉</span></span></div>
      <div class="dash-panel-body"><div id="chat-body"></div></div>
    </section>
  </div>
```

Add to `<head>`:
```html
    <link rel="stylesheet" href="{{ .CONFIG.FilePaths.WebCDNLocation }}/static/css/gomud.css" />
    <link rel="stylesheet" href="{{ .CONFIG.FilePaths.WebCDNLocation }}/static/css/dashboard.css" />
```
and before `</body>`:
```html
    <script src="{{ .CONFIG.FilePaths.WebCDNLocation }}/static/js/dashboard.js"></script>
```

- [ ] **Step 3: Re-home the renderers; remove the old WinBox layout JS**

In `webclient-pure.html`'s script:
- The xterm `term.open(...)` target stays `#terminal` (now inside the feed panel) — call `fitAddon.fit()` on load and on resize as before.
- Mount `RoomGridSVG` on `#map-render` (already the mapper's mount id) — keep the existing `new RoomGridSVG('#map-render', mapOptions)` call but run it unconditionally at init instead of inside the WinBox `oncreate`.
- Vitals: keep `createVitalsRow`/`updateVitalsRow`/`updateVitalsWindow` but have them target `#vitals-bars` directly (drop the WinBox mount/visibility logic).
- Comm: render comm lines into `#chat-body` (tabs added in Task 7).
- **Remove** the WinBox window creation for `Map`/`Comm`/`Char.Vitals`, the `GMCPWindows` window objects, `PANEL_WIDTH`/`resizeAllPanels`/`isTooSmallForPanels`/`computeLayout`, and the related sizing math. Keep the GMCP **update handlers** (`GMCPUpdateHandlers['Char.Vitals']` etc.) but point them at the new mount nodes. Keep WinBox `<script>` loaded (Task 5 uses it).
- Populate `#sess-name` from the character name when known (GMCP/connect), and wire `#btn-reset-layout` to call `Dashboard.resetLayout()` (defined in Task 6; safe no-op stub until then).

- [ ] **Step 4: Verify**

`node --check` is not applicable to `.html`; instead confirm: the file has no remaining `new WinBox(` for Map/Comm/Vitals, no references to removed helpers (`grep -n "resizeAllPanels\|isTooSmallForPanels\|GMCPWindows\['Map'\]"`), and the new panel ids exist. Human smoke: hard-refresh `/webclient` → docked arrangement C renders, the feed/map/vitals show and update live, command input works.

- [ ] **Step 5: Commit**
```bash
git add _datafiles/html/public/static/css/dashboard.css _datafiles/html/public/webclient-pure.html
git commit -m "feat(webclient): docked dashboard grid + migrate panels off WinBox"
```

---

### Task 2: `dashboard.js` controller + responsive reflow

Introduces the controller and the 3-col → 2-col tabbed-rail → 1-col bottom-tab+drawer reflow.

**Files:**
- Create: `_datafiles/html/public/static/js/dashboard.js`
- Modify: `_datafiles/html/public/static/css/dashboard.css`

- [ ] **Step 1: Controller skeleton (`dashboard.js`)**

```js
"use strict";
(function (global) {
  const PANELS = ["map","art","trig","feed","cmd","vitals","status","chat"];
  const SIDE   = ["map","vitals","status","chat","art","trig"]; // rail/drawer order

  const Dashboard = {
    el: null,
    mode: "wide",        // "wide" | "rail" | "phone"
    init() {
      this.el = document.getElementById("dashboard");
      if (!this.el) return;
      this._applyMode();
      window.addEventListener("resize", () => this._applyMode());
      // Tasks 3-6 init hooks:
      if (this.initSplitters) this.initSplitters();
      if (this.initRearrange) this.initRearrange();
      if (this.initPopout) this.initPopout();
      if (this.restoreLayout) this.restoreLayout();
    },
    _applyMode() {
      const w = this.el.clientWidth;
      const mode = w >= 1100 ? "wide" : (w >= 700 ? "rail" : "phone");
      if (mode === this.mode) return;
      this.mode = mode;
      this.el.dataset.mode = mode;     // CSS keys off [data-mode]
      this._buildRailOrDrawer();
    },
    _buildRailOrDrawer() { /* Task 2 Step 3 */ },
    resetLayout() { /* Task 6 */ },
  };
  global.Dashboard = Dashboard;
  document.addEventListener("DOMContentLoaded", () => Dashboard.init());
})(window);
```

- [ ] **Step 2: Responsive CSS (`dashboard.css`)**

Append mode-specific rules. `#dashboard[data-mode="wide"]` uses the base grid (Task 1). Add:

```css
/* 2-col tabbed rail */
#dashboard[data-mode="rail"] {
  grid-template-columns: minmax(0, 2.2fr) minmax(240px, 1fr);
  grid-template-rows: auto minmax(0,1fr) auto;
  grid-template-areas:
    "sess sess"
    "feed rail"
    "cmd  rail";
}
#dashboard[data-mode="rail"] #dash-rail { grid-area: rail; display: flex; flex-direction: column; min-height: 0; }
#dashboard[data-mode="rail"] .dash-panel[data-panel]:not(#panel-feed):not(#panel-cmd) { display: none; }
#dashboard[data-mode="rail"] #dash-rail .dash-panel.rail-active { display: flex; }
#dash-rail-tabs { display: flex; flex-wrap: wrap; gap: 3px; padding: 3px; }
#dash-rail-tabs .rail-tab { font-size: 0.72rem; padding: 3px 8px; }   /* uses .brass */

/* 1-col phone: feed full + bottom tab bar + drawer */
#dashboard[data-mode="phone"] {
  grid-template-columns: 1fr;
  grid-template-rows: auto minmax(0,1fr) auto;
  grid-template-areas: "sess" "feed" "cmd";
}
#dashboard[data-mode="phone"] .dash-panel:not(#panel-feed):not(#panel-cmd) { display: none; }
#dash-tabbar {
  position: fixed; left: 0; right: 0; bottom: 0; z-index: 50;
  display: none; justify-content: space-around;
  background: linear-gradient(var(--bar-top), var(--bar-bottom));
  border-top: 2px solid var(--gold-rule);
}
#dashboard[data-mode="phone"] ~ #dash-tabbar { display: flex; }
#dash-tabbar .tabbar-btn { flex:1; text-align:center; padding:8px 0; font-size:18px; color: var(--ink-gold); cursor:pointer; }
#dash-drawer {
  position: fixed; left:0; right:0; bottom:0; z-index: 49;
  height: 60vh; transform: translateY(100%); transition: transform .2s ease;
  background: var(--panel-bg); border-top: 2px solid var(--gold-rule);
  display: flex; flex-direction: column;
}
#dash-drawer.open { transform: translateY(0); }
```

- [ ] **Step 3: `_buildRailOrDrawer()` — move side panels into the rail/drawer container**

Implement so the SAME panel DOM is relocated (not duplicated). In `wide`, side panels sit in their grid slots. In `rail`, append them into `#dash-rail` (created once) with a `#dash-rail-tabs` brass tab per panel; clicking a tab adds `.rail-active` to one panel. In `phone`, build `#dash-tabbar` (icons per `SIDE` panel) + a `#dash-drawer`; tapping an icon moves that panel into the drawer body and adds `.open`; tapping again closes. On returning to `wide`, move panels back to their original grid slots (track original parent via `data-home-slot`). Provide a `_homePanel(name)` helper that re-appends a panel to `#dashboard` with its `grid-area` restored. Icons map: map 🗺 · vitals ❤ · status ⚔ · chat 💬 · art 🖼 · trig ⌨.

- [ ] **Step 4: Link + verify**

`node --check static/js/dashboard.js` passes. Human smoke: resize the window across 1100px and 700px — layout reflows to tabbed rail then bottom-tab+drawer; tapping tabs/icons reveals each panel; widening restores arrangement C.

- [ ] **Step 5: Commit**
```bash
git add _datafiles/html/public/static/js/dashboard.js _datafiles/html/public/static/css/dashboard.css
git commit -m "feat(webclient): dashboard controller + responsive rail/drawer reflow"
```

---

### Task 3: Splitters (resize regions in wide mode)

**Files:** Modify `dashboard.js`, `dashboard.css`.

- [ ] **Step 1: Splitter elements + CSS**

Add two column splitters (between L|C and C|R) and one row splitter (between the top and bottom panel rows) as `.dash-splitter` divs placed in the grid. CSS:
```css
.dash-splitter { background: transparent; z-index: 5; }
.dash-splitter.col { cursor: col-resize; width: 6px; }
.dash-splitter.row { cursor: row-resize; height: 6px; }
.dash-splitter:hover { background: rgba(201,168,106,0.25); }
#dashboard[data-mode="rail"] .dash-splitter.row, 
#dashboard[data-mode="phone"] .dash-splitter { display: none; } /* only feed↔rail col splitter survives in rail */
```

- [ ] **Step 2: `initSplitters()` — pointer-drag to adjust CSS custom props**

Implement drag handlers that adjust the `--col-l`/`--col-c`/`--col-r` (and `--row-t`/`--row-b`) custom properties on `#dashboard` based on pointer delta vs container size (convert to `fr` ratios). Clamp to sensible mins. Persist the resulting values via `Dashboard.saveLayout()` (Task 6) on pointer-up. Use `setPointerCapture` for robust dragging.

```js
Dashboard.initSplitters = function () {
  const el = this.el;
  const drag = (splitter, axis) => {
    splitter.addEventListener("pointerdown", (e) => {
      splitter.setPointerCapture(e.pointerId);
      const rect = el.getBoundingClientRect();
      const onMove = (ev) => {
        if (axis === "x") {
          const ratio = Math.min(0.45, Math.max(0.12, (ev.clientX - rect.left) / rect.width));
          // left/right columns scale around the dominant center
          el.style.setProperty(splitter.dataset.edge === "l" ? "--col-l" : "--col-r", (ratio / (1 - ratio)).toFixed(3) + "fr");
        } else {
          const r = Math.min(0.8, Math.max(0.2, (ev.clientY - rect.top) / rect.height));
          el.style.setProperty("--row-t", r.toFixed(3) + "fr");
          el.style.setProperty("--row-b", (1 - r).toFixed(3) + "fr");
        }
      };
      const onUp = () => { el.removeEventListener("pointermove", onMove); if (this.saveLayout) this.saveLayout(); };
      el.addEventListener("pointermove", onMove);
      el.addEventListener("pointerup", onUp, { once: true });
    });
  };
  el.querySelectorAll(".dash-splitter.col").forEach((s) => drag(s, "x"));
  el.querySelectorAll(".dash-splitter.row").forEach((s) => drag(s, "y"));
};
```
(Position the splitter divs in the grid via thin tracks or absolute overlays at the column/row boundaries — implementer chooses; absolute overlays aligned to the boundaries are simplest and avoid disturbing `grid-template-areas`.)

- [ ] **Step 3: Verify + Commit**

Human smoke: dragging the dividers resizes regions; the center feed stays dominant; sizes survive a reload (after Task 6). `node --check` passes.
```bash
git add _datafiles/html/public/static/js/dashboard.js _datafiles/html/public/static/css/dashboard.css
git commit -m "feat(webclient): draggable region splitters"
```

---

### Task 4: Rearrange (drag-to-swap) + collapse

**Files:** Modify `dashboard.js`.

- [ ] **Step 1: `initRearrange()` — header drag swaps grid-area**

Implement HTML5 drag (or pointer-based) on `.dash-panel-head`: on drop onto another panel, **swap** the two panels' `style.grid-area` (and their `data-home-slot`). Persist via `saveLayout()`. Add a drag-over highlight class. Disable in `rail`/`phone` modes.

```js
Dashboard.initRearrange = function () {
  let dragName = null;
  this.el.querySelectorAll(".dash-panel-head").forEach((head) => {
    const panel = head.closest(".dash-panel");
    head.setAttribute("draggable", "true");
    head.addEventListener("dragstart", () => { if (this.mode === "wide") dragName = panel.dataset.panel; });
    panel.addEventListener("dragover", (e) => { if (dragName && this.mode === "wide") e.preventDefault(); });
    panel.addEventListener("drop", (e) => {
      e.preventDefault();
      if (!dragName || this.mode !== "wide" || dragName === panel.dataset.panel) return;
      this._swap(dragName, panel.dataset.panel); dragName = null; if (this.saveLayout) this.saveLayout();
    });
  });
};
Dashboard._swap = function (a, b) {
  const pa = document.getElementById("panel-" + a), pb = document.getElementById("panel-" + b);
  const ga = pa.style.gridArea || a, gb = pb.style.gridArea || b;
  pa.style.gridArea = gb; pb.style.gridArea = ga;
  pa.dataset.homeSlot = gb; pb.dataset.homeSlot = ga;
};
```

- [ ] **Step 2: Collapse toggle**

Wire `.ph-collapse` clicks to toggle `.collapsed` on the panel; persist. (CSS from Task 1 hides the body.)
```js
Dashboard.el.querySelectorAll(".ph-collapse").forEach((b) =>
  b.addEventListener("click", () => { b.closest(".dash-panel").classList.toggle("collapsed"); if (Dashboard.saveLayout) Dashboard.saveLayout(); }));
```
(Place inside an init function called from `Dashboard.init`.)

- [ ] **Step 3: Verify + Commit**

Human smoke: drag a panel header onto another → they swap; `▾` collapses to header only; both survive reload (after Task 6).
```bash
git add _datafiles/html/public/static/js/dashboard.js
git commit -m "feat(webclient): drag-to-swap panel rearrange + collapse"
```

---

### Task 5: Pop-out via WinBox

**Files:** Modify `dashboard.js`.

- [ ] **Step 1: `initPopout()` — detach panel body into a floating WinBox**

Wire `.ph-popout` clicks: create a `new WinBox({ title, mount: panelBody, ... })`, leave a "popped out — re-dock" stub in the slot, and on WinBox close, move the body DOM back and remove the stub. Track popped panels for persistence. **Restrict pop-out to side panels first** (`vitals`,`status`,`chat`,`map`); guard the feed pop-out behind a flag (per spec risk) — the feed/command `⧉` may be omitted in Task 1 markup if deferred (feed already has `⧉`; gate it to no-op with a console note if fragile).

```js
Dashboard.initPopout = function () {
  this.el.querySelectorAll(".ph-popout").forEach((btn) => {
    const panel = btn.closest(".dash-panel"); const name = panel.dataset.panel;
    btn.addEventListener("click", () => this.popout(name));
  });
};
Dashboard.popout = function (name) {
  const panel = document.getElementById("panel-" + name);
  const body = panel.querySelector(".dash-panel-body");
  const title = panel.querySelector(".ph-title").textContent;
  const stub = document.createElement("div");
  stub.className = "dash-panel-body popout-stub";
  stub.innerHTML = '<div style="padding:10px;text-align:center;color:var(--text-secondary)">Popped out — <a href="#" style="color:var(--ink-gold)">re-dock</a></div>';
  panel.appendChild(stub);
  const wb = new WinBox({ title, mount: body, width: 360, height: 280, onclose: () => { panel.replaceChild(body, stub); this._popped.delete(name); if (this.saveLayout) this.saveLayout(); return false; } });
  stub.querySelector("a").addEventListener("click", (e) => { e.preventDefault(); wb.close(); });
  this._popped = this._popped || new Set(); this._popped.add(name);
  if (this.saveLayout) this.saveLayout();
};
```

- [ ] **Step 2: Verify + Commit**

Human smoke: `⧉` on Vitals/Status/Chat/Map floats it; re-dock returns it intact and live. (Feed/command pop-out deferred if fragile.)
```bash
git add _datafiles/html/public/static/js/dashboard.js
git commit -m "feat(webclient): panel pop-out via WinBox (side panels)"
```

---

### Task 6: Persistence (localStorage) + reset

**Files:** Modify `dashboard.js`.

- [ ] **Step 1: `saveLayout()` / `restoreLayout()` / `resetLayout()`**

Schema (key `dogmud.dashboard.layout.v1`):
```js
// { cols:{l,c,r}, rows:{t,b}, arrange:{panelName:gridArea,...}, collapsed:[names], popped:[names] }
Dashboard.LS_KEY = "dogmud.dashboard.layout.v1";
Dashboard.saveLayout = function () {
  const cs = getComputedStyle(this.el);
  const state = {
    cols: { l: this.el.style.getPropertyValue("--col-l"), c: this.el.style.getPropertyValue("--col-c"), r: this.el.style.getPropertyValue("--col-r") },
    rows: { t: this.el.style.getPropertyValue("--row-t"), b: this.el.style.getPropertyValue("--row-b") },
    arrange: {}, collapsed: [], popped: Array.from(this._popped || []),
  };
  PANELS.forEach((n) => {
    const p = document.getElementById("panel-" + n); if (!p) return;
    if (p.style.gridArea) state.arrange[n] = p.style.gridArea;
    if (p.classList.contains("collapsed")) state.collapsed.push(n);
  });
  try { localStorage.setItem(this.LS_KEY, JSON.stringify(state)); } catch (e) {}
};
Dashboard.restoreLayout = function () {
  let s; try { s = JSON.parse(localStorage.getItem(this.LS_KEY) || "null"); } catch (e) { s = null; }
  if (!s) return;
  if (s.cols) { for (const k of ["l","c","r"]) if (s.cols[k]) this.el.style.setProperty("--col-"+k, s.cols[k]); }
  if (s.rows) { if (s.rows.t) this.el.style.setProperty("--row-t", s.rows.t); if (s.rows.b) this.el.style.setProperty("--row-b", s.rows.b); }
  if (s.arrange) Object.entries(s.arrange).forEach(([n, ga]) => { const p = document.getElementById("panel-"+n); if (p) { p.style.gridArea = ga; p.dataset.homeSlot = ga; } });
  (s.collapsed||[]).forEach((n) => document.getElementById("panel-"+n)?.classList.add("collapsed"));
  (s.popped||[]).forEach((n) => this.popout && this.popout(n));
};
Dashboard.resetLayout = function () {
  try { localStorage.removeItem(this.LS_KEY); } catch (e) {}
  location.reload();
};
```
Call `saveLayout()` from splitter/swap/collapse/popout (already wired in Tasks 3-5). Call `restoreLayout()` in `init()` (already hooked). Wire `#btn-reset-layout` → `Dashboard.resetLayout()`.

- [ ] **Step 2: Verify + Commit**

Human smoke: rearrange/resize/collapse/pop-out, reload `/webclient` → state restored; "Reset layout" → defaults return. `node --check` passes.
```bash
git add _datafiles/html/public/static/js/dashboard.js
git commit -m "feat(webclient): persist + reset dashboard layout (localStorage)"
```

---

### Task 7: Chat panel — channel tabs + own input

**Files:** Modify `webclient-pure.html`, `dashboard.css`.

- [ ] **Step 1: Chat tab strip + input in the chat panel body**

Replace `#chat-body` content with a tab strip (`Say / Tell / OOC / Trade / All`) + a `#chat-log` scroll area + a `#chat-input`. Style tabs with `.brass`. Route incoming comm GMCP/text to per-channel buffers; the active tab shows its buffer (All = merged). `#chat-input` sends to the server with the active channel's command prefix (e.g. Tell → `tell `, OOC → `ooc `, Say → `say `). Reuse the existing comm-message handling — split it to append to the correct channel buffer instead of one panel.

```css
#chat-body { display:flex; flex-direction:column; height:100%; }
#chat-tabs { display:flex; gap:3px; padding:3px; flex-wrap:wrap; }
#chat-tabs .chat-tab { font-size:0.72rem; padding:2px 8px; } /* .brass; .active = pressed */
#chat-log { flex:1 1 auto; overflow:auto; padding:4px 6px; font-family:var(--font-mono); font-size:0.8rem; }
#chat-input { margin:4px; }
```

- [ ] **Step 2: Verify + Commit**

Human smoke: chat lines land under the right tab; switching tabs filters; typing in the chat input with a channel selected sends with the right prefix; the main feed no longer duplicates chatter (or duplicates acceptably — confirm routing).
```bash
git add _datafiles/html/public/webclient-pure.html _datafiles/html/public/static/css/dashboard.css
git commit -m "feat(webclient): tabbed chat panel with per-channel input"
```

---

### Task 8: Character Status & Conditions panel

**Files:** Modify `webclient-pure.html` (+ `dashboard.css` if needed). Client-only — no Go changes.

- [ ] **Step 1: Render available status into `#status-conditions`**

From the GMCP data the client already receives (inspect existing handlers — e.g. `Char.Vitals`, `Char.Info`/`Char.Status` if present, buffs/affects if present), render a compact status block: character state/stance, light/dark, encumbrance tier, and a row of condition/buff chips when available. If a given datum is not in GMCP, omit it gracefully. Use the leather chip style.

```css
#status-conditions { padding:6px; display:flex; flex-direction:column; gap:6px; }
.status-line { font-family:var(--font-serif); font-size:0.85rem; color:var(--text-primary); }
.cond-chips { display:flex; flex-wrap:wrap; gap:4px; }
.cond-chip { font-size:0.72rem; padding:2px 8px; border-radius:12px; border:1px solid var(--panel-border);
  background:#1d1610; color:var(--ink-gold); }
```

- [ ] **Step 2: Note the GMCP gap (do NOT expand scope)**

If stance/conditions/affects are not in current GMCP payloads, render what *is* available and leave a code comment + add a one-line note to the spec's Risks that a small Go-side GMCP `Char.Conditions` (or similar) addition is a follow-up. Do not add Go code in this sub-project.

- [ ] **Step 3: Verify + Commit**

Human smoke: the Status & Conditions panel shows real character state; no console errors when a datum is missing.
```bash
git add _datafiles/html/public/webclient-pure.html _datafiles/html/public/static/css/dashboard.css
git commit -m "feat(webclient): character status & conditions panel"
```

---

### Task 9: Full smoke + leftover-reference sweep

**Files:** None (verification only; small fixes if found).

- [ ] **Step 1: Cross-mode + cross-browser smoke**

Human smoke in `/webclient` (and fullscreen `/webclient-pure`): default arrangement C; splitters; drag-swap; collapse; pop-out/re-dock; reload persistence; reset; reflow at 1100px and 700px incl. drawer; live GMCP updates for feed/map/vitals/chat/status; command input; sound toggle still works.

- [ ] **Step 2: Leftover-reference sweep**
```bash
grep -rn "GMCPWindows\|resizeAllPanels\|isTooSmallForPanels\|PANEL_WIDTH" _datafiles/html/public/webclient-pure.html
```
Expected: no matches (all replaced by the dashboard). Fix any stragglers.

- [ ] **Step 3: Final commit (if fixes)**
```bash
git add -- <specific files>
git commit -m "chore(webclient): dashboard smoke fixes"
```

---

## Self-review

- **Spec coverage:** layout C grid (T1), panel chrome + interactions (T1,T3,T4,T5), responsive reflow 3→2→1 (T2), persistence + reset (T6), panels + data sources (T1 map/vitals/feed/cmd, T7 chat, T8 status, reserved slots T1), migration off WinBox + theme tokens (T1), scope boundaries (slots reserved; no Go changes) — all covered.
- **Placeholders:** none in the sense of "TBD" — JS is provided as working implementations or precise algorithms; the only deliberately-deferred items (feed/map pop-out fragility, GMCP conditions gap) are the spec's flagged risks with explicit fallbacks.
- **Type/name consistency:** panel ids (`panel-map`…`panel-chat`), `data-panel` names, CSS custom props (`--col-l/c/r`, `--row-t/b`), `[data-mode]`, the `LS_KEY` schema, and the `Dashboard.*` method names are used consistently across tasks.
- **Verification realism:** every task notes subagents verify by `node --check` (JS) + structural/grep checks; the authoritative visual smoke is the human's (server holds the ports).
```
