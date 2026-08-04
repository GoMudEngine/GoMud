# Admin Builder Page — Design (admin web-building, sub-project 1b)

**Date:** 2026-07-22
**Status:** approved design, pre-plan
**Epic:** Admin web-building. **Sub-project 1b** — the room-builder UI, built on the
**1a authored coordinate model** (done, merged `09b2be086`).

## Motivation

Admins can build rooms in-game today only through synchronous text-prompt commands
(`build room`, `room set`, `room edit exits`), and the existing `/admin/*` web app is
read-only (dead Submit buttons). 1b delivers the first real **graphical room builder**:
an admin opens a web page, sees the world as an editable map, clicks a room to edit it,
clicks a ghost cell to create a neighbor, and wires exits — all persisting to the
authorable room YAML templates and re-rendering live. It raises content velocity and
lets non-coders contribute rooms.

Scope is **rooms only**. Mob/item/dialogue authoring and room spawn-lists/containers are
later sub-projects (2/3/4/5).

## Architecture

### The page + session
- A new standalone page at **`/build`**, served by `internal/web` alongside `/admin/*`
  (`web.go:292`), gated by the **same admin-account HTTP Basic-Auth** (`auth.go`,
  `doBasicAuth` → `uRecord.Role == RoleAdmin`). No new credential system.
- The page opens its **own WebSocket → GMCP session** as that admin user (the same
  `/ws` + GMCP transport the play client uses). It is NOT a panel inside the play
  client; it stands alone.
- All build mutations flow over **new client→server `Build.*` GMCP packages**, added to
  the existing `HandleIAC` switch (`modules/gmcp/gmcp.go:274`), each **gated on
  `RoleAdmin`** (`userIdForConnection` → role check). Server responses + map refreshes
  come back via `dispatchGMCP` (`gmcp.go:527`).

### The canvas (editor-grade, graphical SVG)
- A **new editor canvas component** (JS/SVG), sharing the leather/SVG rendering DNA of
  `RoomGridSVG` (`gmcp.js`) but purpose-built for editing — not the fog-of-war,
  small-circle play mapper.
- Renders **one plane at a time** (a plane selector in the toolbar switches it). Rooms
  are labelled biome-tinted nodes on an explicit grid; exits are styled lines
  (spatial road/secret-dashed/one-way/portal-stub); the selected room gets a gold
  selection glow; **dashed `+` ghost cells** sit adjacent to rooms in each open
  direction. Pan (drag) + zoom (scroll).
- Position data comes from the **`Zone.Map` snapshot**, which 1a extended to carry
  authored `x/y/z/plane`. **This requires 1a's deferred Task 6** (mapper reads authored
  coords) so a room the builder moves/creates shows at its authored position
  immediately — Task 6 lands as 1b foundation (below).

### Layout (locked: "A — map + right inspector")
- Left ~70%: the editor canvas. Right ~250px: the **inspector** (selected room's
  fields). Top: a slim **toolbar** — plane selector, current zone label, create-mode
  affordance, undo, and the room **Save** button.

## Interaction model (locked)
- **Select-on-map:** click a room node → the inspector loads its fields.
- **Instant ghost-cell create:** click a dashed `+` → the room is created *immediately*
  at that cell (blank "Untitled", valid + saved), with the reciprocal exit auto-wired,
  and is selected with the cursor in Title. Sketch the map by clicking, fill prose
  after. (`Build.Room.Create`.)
- **Exits** are wired from the inspector's Exits section (`+ add exit`):
  - **Spatial** — pick a direction; the target is the room already in that cell; the
    reciprocal is auto-added. Rejected if it would cross to another **Euclidean** plane
    (per 1a; a spatial exit into a **non-Euclidean** plane — a maze boundary — is
    allowed).
  - **Portal / door** — a named exit (e.g. "enter inn"); pick **any** room as target,
    **including on another plane** (overworld → interior → instance). Non-spatial.
  - Per-exit flags: secret, one-way, lock, exit-message.
  - Existing-to-existing wiring is inspector-only (no shift-click quick-link in v1).
- **Save model:** **explicit Save per room.** Create is instant; text/biome/flag/noun
  edits batch in the inspector until Save → one `Build.Room.Update`. Exit add/remove
  apply immediately (they change the map).

## Inspector fields (v1)
Title, Description, Biome (dropdown), Map symbol + **map legend**, **room flags**
(bank / storage / pvp / character-creation-room), **Nouns** (examinable `look <noun>`
details — key + text list), **Idle messages** (list), **Music** file, and the **Exits**
editor. Skill-training ranges are **excluded** — vestigial in DOGMud (no `train`
command, zero rooms use it; use-based progression replaced level/XP training). The dead
`Room.SkillTraining` field + its never-fired `mapper.snapshot.go:82` tag-check are noted
as minor cleanup (not part of 1b's critical path).

## `Build.*` GMCP protocol (client→server, RoleAdmin-gated)
Added as cases in `HandleIAC` (`gmcp.go:274`), each resolving the admin via
`userIdForConnection` and rejecting non-admins:

| Package | Payload | Server action |
|---------|---------|---------------|
| `Build.Room.Create` | `{plane,x,y,z, fromRoomId, dir}` | `ValidatePlacement` gate → `NewRoom` (id from `GetNextRoomId`) → `SaveRoomTemplate` → reciprocal `ConnectRoom(from,dir,new)` → both saved |
| `Build.Room.Update` | `{roomId, title,description,biome,symbol,legend,flags,nouns,idlemessages,music}` | mutate the loaded template → `SaveRoomTemplate` (fixes the biome/other non-persisting gaps by always routing through save) |
| `Build.Room.Delete` | `{roomId}` | remove room + clean up exits pointing to it |
| `Build.Exit.Add` | `{roomId, dir|portalName, toRoomId, secret,oneway,lock,message}` | add exit (+ reciprocal for spatial) → `SaveRoomTemplate` both; spatial cross-Euclidean-plane rejected |
| `Build.Exit.Remove` | `{roomId, exitName}` | remove exit (+ reciprocal) → save both |

**Server → client** after every mutation: `Build.Result {ok, error}` (surfaced as a
toast/inline error in the builder), plus a refreshed **`Zone.Map`** snapshot for the
affected plane so the canvas re-renders. `SaveRoomTemplate` already mutates the live
game in-memory (no restart), so edits are immediately live for players too.

**Concurrency:** each `Build.*` handler runs on the single MainWorker goroutine (same as
all GMCP + the world tick), so mutations are serialized against the tick without the
coarse global `RunWithMUDLocked` the `/admin/*` app uses per request.

## Foundation / build order (for the plan)
1. **Task 6 from 1a — mapper reads authored coords** (deferred into here). The runtime
   mapper + `Zone.Map` snapshot read stored `x/y/z/plane` instead of crawling, so the
   builder canvas shows authored positions. **Fix precondition:** the ironwind_steppe
   overlap is already resolved (`b98d0c1b6`), and the world is collision-free, so this
   can flip on cleanly (validate `cartcheck` stays clean world-wide after).
2. `/build` page scaffold + admin Basic-Auth + WebSocket/GMCP session.
3. `Build.*` GMCP handlers (server) with the RoleAdmin gate + `SaveRoomTemplate`/
   `ConnectRoom`/`NewRoom`/`ValidatePlacement` wiring + unit tests on the pure seams.
4. The editor-grade SVG canvas (render `Zone.Map` per plane; select; ghost cells;
   pan/zoom).
5. The inspector (fields + exits editor) + Save.
6. Create/edit/connect end-to-end + `Build.Result`/`Zone.Map` refresh loop.

## Out of scope (explicit)
- Mob/item/dialogue authoring; room spawn-lists; containers (sub-projects 2/3/4/5).
- Deleting/creating **zones** and setting `zone-config` (a small follow-up; 1b edits
  rooms within existing zones + creates rooms via ghost cells).
- Bulk/offline data editing (that's the existing `/admin/*` app's niche).
- The `Room.SkillTraining` dead-code cleanup (noted, not required for 1b).

## Testing & verification
- **Unit:** `Build.*` handler seams against fixtures — create honors `ValidatePlacement`
  (rejects an occupied cell), update round-trips fields through `SaveRoomTemplate`, exit
  add/remove maintains reciprocity, cross-Euclidean-plane spatial exit is rejected,
  non-admin GMCP is refused.
- **Boot-clean** after any struct/loader change; `cartcheck` clean world-wide after
  Task 6 flips the mapper to authored coords.
- **REQUIRED content/UX gate** (CLAUDE.md): drive the real `/build` page in a browser as
  an admin — create a small room cluster by clicking ghost cells, edit fields + Save,
  wire a spatial exit and a cross-plane portal, confirm the map re-renders live and the
  new rooms are walkable in a play client. Read every interaction as a confused human
  would; fix what it surfaces before handing over.
