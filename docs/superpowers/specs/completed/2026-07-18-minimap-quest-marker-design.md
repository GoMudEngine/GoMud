# Minimap Quest Marker — Design

**Date:** 2026-07-18
**Status:** Design approved (visual brainstorm with the team, incl. Malia on visual design); ready for implementation plan.
**Scope:** A web-client feature that marks the player's current quest objective on the leather minimap and steers them toward it turn-by-turn, plus a Quests panel to choose which quest is focused. Web-only. Terminal play is unaffected.

---

## 1. Problem

The newbie world (Pothole Coulee especially) is a hub-and-spoke layout that reads as maze-like to first-timers — confirmed firsthand while playtesting: it's easy to get turned around between the hub, shop, inn, and the seven trail-heads. The `hint` command gives text directions, but there is no *visual* "go here" cue. After the footing hand-off a player also juggles ~7 active trail quests at once, so "what am I heading toward right now" is unclear.

## 2. Goals & Non-Goals

**Goals**
- Show a visual destination + a turn-by-turn "next step" on the web minimap for the player's **focused** quest.
- Let the player pick which quest is focused, from a web Quests panel that doubles as a quest log.
- Reuse the existing focus/hint plumbing rather than inventing new state.

**Non-Goals**
- No markers for *all* active quests at once (clutter — decided against).
- No ASCII/terminal map marker. The feature is **web-only** (the minimap itself is web-only). Terminal players keep `hint` exactly as today.
- No full route line — only the *next hop*.
- No cross-zone routing beyond pointing at the zone-boundary exit.

## 3. Key decisions (settled in brainstorm)

1. **One marker, the focused quest** — not one-per-active-quest.
2. **"Focused quest" is the existing `Character.LastQuestId`.** It is already persisted (`yaml:"lastquestid"`), already set on quest progress (`internal/characters/quests.go`), and already what `hint` (no arg) resolves against (`internal/usercommands/hint.go`). We **reuse it as-is** — no new focus field, no change to `hint`.
3. **The web Quests panel sets the focus.** Clicking a quest sends a GMCP message that sets `LastQuestId`. Because `hint` and the marker both read `LastQuestId`, the panel, `hint`, and the marker are three views of one focus.
4. **Indication = destination marker + next-step arrow, shown together.**
   - **Destination marker** on the target room when it's on the visible map.
   - **Next-step arrow** points at the next room on the shortest path (server BFS), updating each move: to the adjacent node when it's visited, or out the correct *exit direction* when the next room is still fog.
5. **Target-room resolution** (per the focused quest's current step): explicit `map_target: <roomid>` on the step wins → else infer from a `room_enter` trigger on that step → else **no marker** (panel still shows the hint; the map just gets no arrow — no guessing).
6. **Panel home:** left column, directly under the Map (the panel you pick from sits above the map it steers). It's a standard drag-rearrangeable `dash-panel`; its body scrolls (`overflow:auto`) so a long quest list never blows out the column.

## 4. Visual design (approved)

Faithful to the existing leather mapper (`RoomGridSVG` in `gmcp.js`, `RoomGridSVG.LEATHER` palette).

- **Destination marker** (distinct from the verdigris party *figure*): a **brass ring** (radial gradient `#f4dd92 → #cb9f42 → #8a6620`) around the dark room node, a soft **oxidized-copper-patina glow** (`--copper #4a6b5d` at ~0.30 opacity) and a **patina core** (`#6bb0a0`), a small brass pin/flag above, and the quest name in italic patina beneath. Yellow+green so it never blurs with the small green party figure (different shape *and* emphasis).
- **Next-step arrow:** gold (`#e8b84a`) segment with an auto-aligned `marker-end` arrowhead, from the current room toward the next hop. When the next room is visited it also gets a thin subtle gold ring ("go here next").
- **Legend:** two new rows — "Quest destination" and "Next step."
- **"You are here"** treatment is unchanged (the real raised embossed token + warm glow + emphasis ring + italic name label).
- **Quests panel:** standard `dash-panel` (gold border, gradient head with `ph-title` "Quests" + ▾/⧉ buttons, copper left-column border-accent). Body lists active quests, each a row with a focus dot (**filled brass ◉** = focused, hollow **◎** = not) + name + a small progress bar. The focused row is highlighted (brass tint + brass left-edge) and shows its **current hint text** and a "◆ marked on map" note. Clicking a row moves the focus.

## 5. Architecture

Almost all of the focus/hint/quest-list plumbing already exists. The new work is: extend one GMCP payload, add two small server helpers (target resolution + BFS), add an inbound focus message, and build the web panel + the marker rendering.

### 5.1 Server

- **Extend `Char.Quests` GMCP** (`modules/gmcp/gmcp.Char.go`). Today `GMCPCharModule_Payload_Quest` is `{Name, Description, Completion}`, rebuilt on quest progress and on request. Add:
  - `Id int` — the quest id (so a panel click can name it, and the client can match the focused quest).
  - `Hint string` — the current step's `Hint` (the panel shows the hint, the same string `hint` prints).
  - `Focused bool` (or a top-level `focused` id) — true for the `LastQuestId` quest.
  - On the focused quest only: `TargetRoom int`, `NextRoom int`, `NextDir string` (empty when the next room is visited and thus drawable; set to a compass dir when the next room is still fog). Omitted/zero when there is no resolvable target.
- **Target resolution helper** (questengine): given a quest + current step, return the target room id via `step.MapTarget` (new field) → else the `room_enter` trigger's `room` for that step → else 0 (no target).
- **Next-step helper:** breadth-first search over the room exit graph from the player's current room to the target room; return the first hop's room id + the compass direction of that exit. Cross-zone: stop the BFS at the zone boundary and return the boundary exit's direction. Unreachable: return none. One BFS for the focused quest per emit; negligible cost.
- **Quest step schema** (`internal/questengine/types.go`): add optional `map_target: <roomid>` to a step (`yaml:"map_target,omitempty"`). Undeclared → resolution falls through to the `room_enter` inference. No existing quests need editing; author it where a non-arrival step has a meaningful location (e.g. "buy a shirt *at the shop*").
- **Inbound focus message:** a GMCP client→server message (following the module's existing inbound-GMCP dispatch) that sets `user.Character.LastQuestId = questId` (validated against the player's active quests), then re-emits `Char.Quests`. `hint` and the marker then follow automatically. Persists with the character (`LastQuestId` is already saved).

### 5.2 Web client

- **Quests panel** — a new `dash-panel` in the left column under Map (`webclient-pure.html` + a small renderer + `dashboard.css`). Renders from `Char.Quests`: one row per quest (focus dot, name, progress bar), focused row highlighted with its hint + "marked on map." Click → send the focus GMCP message. Body uses the standard `overflow:auto` so it scrolls; it's drag-rearrangeable like every other panel.
- **Marker rendering in `RoomGridSVG`** (`gmcp.js`) — read `TargetRoom`/`NextRoom`/`NextDir` from `Char.Quests` (client keeps the latest). Draw the destination marker on the target room's node when that room is currently rendered; draw the next-step arrow from the current room toward `NextRoom` (to its node if drawn, else out the `NextDir` exit). Add the two legend rows. Re-render on `Char.Quests` update and on room change (the same triggers that already refresh the map/party).

### 5.3 Component boundaries

- `questengine`: "what room does this quest point at, and what's the next hop from here?" — pure functions over quest defs + the room graph. Testable without GMCP or the client.
- `gmcp.Char.go`: serialize that into the existing `Char.Quests` payload; handle the inbound focus message. No pathfinding logic of its own beyond calling the helpers.
- `gmcp.js` `RoomGridSVG`: given target/next-step in the payload, draw. No quest knowledge beyond the payload.
- Quests panel JS: render the list, own the click→focus round-trip. No map knowledge.

## 6. Data flow

- **On quest progress or move:** server emits `Char.Quests` (focused quest carries target/next-step) and `Zone.Map` (as today). Client updates the panel and the marker.
- **On panel click:** client sends the focus GMCP → server sets `LastQuestId`, re-emits `Char.Quests` → client updates the focused highlight + hint + marker; `hint` (no arg) now returns the newly-focused quest.

## 7. Edge cases

- **No active quests:** empty panel, no marker.
- **No resolvable target for the current step:** no marker/arrow; panel still shows the hint.
- **Target is the current room:** draw the destination marker on the current node; no arrow.
- **Next room in fog:** arrow points out the `NextDir` exit from the current node (no node to point at yet).
- **Cross-zone target:** arrow points at the zone-boundary exit; destination marker isn't drawn (it's off this zone's map).
- **Unreachable / instanced (ephemeral tutorial) rooms:** no marker. The feature targets the persistent world; the guided antechamber doesn't need it.

## 8. Testing

- **Server unit tests:** target resolution (map_target override, room_enter inference, none); BFS next-step (adjacent, multi-hop, cross-zone→boundary exit, unreachable→none); `Char.Quests` payload includes the new fields for the focused quest; inbound focus message sets `LastQuestId` and rejects a non-active quest id.
- **In-game verification (content SOP):** drive the newbie flow on the web client — focus a trail quest in the panel, confirm the destination marker + next-step arrow appear and that `hint` (no arg) switches to that quest; walk one step and confirm the arrow advances; confirm the fog case points out the right exit. Colors verified against the leather theme (and, if needed, raw bytes are only for terminal colors — this is a web render, so verify in the browser).

## 9. Files touched (anticipated)

- `modules/gmcp/gmcp.Char.go` — extend `Char.Quests` payload + build; inbound focus handler.
- `internal/questengine/types.go` — add `map_target` to the step struct.
- `internal/questengine/` (new helper file) — target resolution + BFS next-step.
- `_datafiles/html/public/static/js/gmcp.js` — `RoomGridSVG` destination marker + next-step arrow + legend rows.
- `_datafiles/html/public/webclient-pure.html` + `static/css/dashboard.css` (+ a small panel JS) — the Quests panel.

## 10. Follow-ups (out of scope here)

- Optional per-step `map_target` authoring pass across the newbie quests once the marker exists (so command/item steps get precise targets).
- A future full-route overlay (whole path, not just next hop) if desired.
