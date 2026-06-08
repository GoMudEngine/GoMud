# In-Game Client Dashboard — Framework + Core Panels (Sub-project #1) — Design

**Date:** 2026-06-06
**Status:** Approved (brainstorm) — ready for implementation plan
**Author:** brainstormed with the visual companion

## Goal

Replace the in-game web client's bare xterm terminal + floating WinBox panels
with a **docked, fluid-responsive, rearrangeable dashboard** (the "hybrid C"
model: docked by default, still rearrangeable and pop-out-able). This is
**sub-project #1 of 3** in the client overhaul; it delivers the layout
framework plus the six core panels, with reserved slots for the two panels
that come later.

North-star reference: `sample_mud_frontend.png`. Parent sequence + shared
context: see the `project_web_overhaul_sequence` memory note and the shipped
chrome theme (`docs/superpowers/specs/2026-06-06-web-chrome-theme-design.md`).

## Context — current state

`_datafiles/html/public/webclient-pure.html` (served inside the `/webclient`
iframe, wrapped by the now-leather chrome) currently:
- Renders the game feed as an **xterm.js terminal** (`#terminal`), with a
  command `<input>` (`#command-input`) below it.
- Creates three **floating WinBox windows** on the right — `Map` (mounts
  `RoomGridSVG` from `gmcp.js`), `Comm` (Communications), `Char.Vitals`
  (HP/SP/CP rows for self/companions/party) — positioned by bespoke JS
  (`PANEL_WIDTH`, `resizeAllPanels`, `isTooSmallForPanels`, `updateVitalsWindow`).
- Has a brass sound toggle (`#menu-icon`) + volume panel.
- The `/webclient` iframe is now `display:block` full-bleed (fixed earlier).

GMCP update handlers already exist for `Char.Vitals`, `Comm`/`Char.Vitals`/
`Party.Vitals`, and the `Zone.Map` snapshot. The leather mapper
(`RoomGridSVG`) is complete and reused as-is.

## Locked decisions (from the brainstorm)

1. **Sequencing:** this overhaul is 3 sub-projects — (#1) framework + core
   panels [this doc], (#2) Scene-Art panel + offline image pipeline,
   (#3) tabbed Triggers/Tick-Timers/**Macros** card. Build #1 first; #2 and #3
   plug into reserved slots later.
2. **Default layout: arrangement C** — *visuals left*: left column **Map /
   Scene-Art slot / Triggers slot**, center **Game Feed (dominant) + Command
   Bar**, right column **Vitals / Status & Conditions / Chat**.
3. **Dock engine: custom CSS-grid + splitters** (no heavy dependency; fully
   leather-themed; best fluid responsiveness). WinBox stays loaded **only** for
   the pop-out feature.
4. **Responsive reflow:** 3-col (≥~1100px) → 2-col with a single tabbed side
   rail (~700–1100px) → 1-col with a **bottom tab-bar + drawer** (<~700px).
5. **Persistence:** `localStorage` for splitter sizes, panel arrangement,
   collapsed/open state, pop-out state; a "Reset layout" control.
6. **Macros** fold into sub-project #3's tabbed card (not a standalone panel).
   **Room Contents** occupant list is out (feed shows occupants in text).

Visual source of truth:
`docs/superpowers/specs/2026-06-06-web-client-dashboard-mockups/`
(`default-layout.html`, `responsive-reflow.html`).

## Layout & grid

- A CSS grid container fills the iframe viewport (exact-fit, no body scroll —
  the iframe is already `display:block` full-bleed).
- Arrangement C via `grid-template-areas`. Desktop columns approximately
  `minmax(220px, 0.9fr) minmax(0, 2fr) minmax(220px, 0.9fr)`; the center feed
  is the dominant, flexible track. Rows use `fr` for the panel stacks with the
  command bar as an `auto` row under the feed.
- All tracks use `clamp()`/`minmax()` so panels shrink gracefully to a floor,
  then the layout reflows (see Responsive) rather than crushing content.
- Region map (areas):
  ```
  "map   feed  vitals"
  "art   feed  status"
  "trig  cmd   chat"
  ```
  with a thin top **session/connection strip** spanning all columns.

## Panel chrome & interactions

- **Panel shell:** each docked panel is a leather card (gomud.css tokens) with
  a header bar — serif-gold title, a brass `▾` collapse toggle, and a brass
  `⧉` pop-out button. Body scrolls internally if needed.
- **Splitters:** draggable handles between columns and between rows resize the
  adjacent regions (adjusting the grid track sizes). Splitter drags are
  pointer-event based and persist their resulting sizes.
- **Rearrange:** drag a panel's header onto another panel to **swap** their
  grid slots. (Swap, not free placement — keeps the grid valid and simple.)
- **Collapse:** `▾` collapses a panel to just its header (frees its track space
  to siblings); state persists.
- **Pop-out:** `⧉` detaches the panel's DOM into a floating **WinBox** window
  (reusing the already-loaded lib); the grid slot shows a "popped out — re-dock"
  stub; closing/re-docking returns the DOM to its slot. Only this one feature
  uses WinBox.
- **Reset layout:** a control (in the session strip or a small gear) clears the
  persisted layout and restores arrangement C defaults.

## Responsive reflow

Driven by container width (the iframe), via media/container queries on the grid
container — no JS layout math:
- **≥~1100px:** full 3-column C.
- **~700–1100px:** 2 columns — `feed`+`cmd` on the left; **all six side panels
  collapse into one right rail rendered as a tab strip** (one panel visible at a
  time, tabs to switch). Splitters reduce to the single feed↔rail divider.
- **<~700px:** 1 column — `feed`+`cmd` fill; side panels are reached via a
  **fixed bottom tab-bar** of icons (map / vitals / status / chat / art /
  triggers) that slides the chosen panel up as a **drawer** over the feed;
  tapping the active icon or a close affordance dismisses it.

The drawer/tab behaviors are CSS-state + minimal JS toggles; the same panel DOM
is reused across all three modes (no duplicate markup).

## Persistence

- A single `localStorage` key (e.g. `dogmud.dashboard.layout.v1`) stores:
  splitter/track sizes, the panel→slot arrangement, per-panel collapsed state,
  and which panels are popped out.
- Restored on load/reconnect; invalid/legacy payloads are ignored (fall back to
  defaults). "Reset layout" deletes the key and re-renders defaults.
- Versioned key so future schema changes don't brick a saved layout.

## Panels in scope + data sources

| Panel | Source | Notes |
|-------|--------|-------|
| **Game Feed** | existing xterm `#terminal` | re-homed into the `feed` region; unchanged behavior |
| **Command Bar** | existing `#command-input` | re-homed into the `cmd` region |
| **Map** | existing `RoomGridSVG` (`gmcp.js`) | mounts into the `map` slot; mapper code untouched |
| **Vitals** | existing vitals-row rendering + `Char/Party.Vitals` GMCP | mounts into `vitals` slot; self/companions/party rows |
| **Chat (tabbed)** | today's `Comm` panel + Comm GMCP | upgraded with channel tabs (Say/Tell/OOC/Trade) + its own input line |
| **Character Status & Conditions** | existing GMCP char data | NEW panel; surfaces stance, light, encumbrance, and buffs/conditions **if** GMCP provides them — see Risks |
| **Scene-Art slot** | — | reserved placeholder tile (sub-project #2) |
| **Triggers slot** | — | reserved placeholder tile (sub-project #3) |
| **Session strip** | existing char/connection state | thin top strip: character name + connection status + Reconnect + Reset-layout |

## Migration & theme

- Remove the bespoke floating-WinBox creation + sizing JS for Map/Comm/Vitals
  (`resizeAllPanels`, `isTooSmallForPanels`, WinBox `new` calls, the
  PANEL_WIDTH layout math) and replace with the grid framework. Keep the
  GMCP update handlers and the content renderers (`RoomGridSVG`,
  `createVitalsRow`/`updateVitalsWindow`, comm rendering) — they now target the
  docked panel bodies instead of WinBox mounts.
- Keep WinBox **loaded** for the pop-out feature only.
- Style all panels/headers/splitters with the shared **`gomud.css`** leather/
  brass tokens (the dashboard CSS may live in `gomud.css` or a dedicated
  `dashboard.css` linked from `webclient-pure.html` — implementer's call, but
  reuse the tokens).

## Acceptance / verification

- Default load shows arrangement C with all six panels + two reserved slots; no
  body scroll inside the iframe at desktop width.
- Splitters resize regions; dragging a panel header onto another swaps them;
  collapse/pop-out/re-dock work; "Reset layout" restores defaults.
- Reload preserves the user's sizes/arrangement/collapsed/pop-out state.
- Resizing the window fluidly reflows 3-col → 2-col tabbed rail → 1-col bottom
  tab-bar+drawer with no crushed/overflowing panels and the feed always usable.
- Map, Vitals, and Chat update live from GMCP exactly as before; the feed and
  command input behave exactly as before.
- Boot test: server starts clean, `/webclient` loads without console errors.

## Risks / open items

- **Status & Conditions data:** the panel depends on GMCP exposing stance/
  conditions/affects. If the current GMCP payloads don't include them, the
  implementation should surface what *is* available (light, encumbrance, basic
  state) and the plan flags a small, optional Go-side GMCP addition as a
  follow-up rather than expanding sub-project #1.
- **Pop-out ↔ grid DOM hand-off:** moving a live panel's DOM (esp. the xterm
  feed or the SVG map) into/out of a WinBox must preserve event handlers and
  GMCP bindings — pop-out should be restricted to safe panels first if the feed
  proves fragile (feed pop-out can be deferred).
- **Chat tabs vs the feed:** confirm during implementation that comm routing
  cleanly separates channel chatter from the main feed without losing messages.
