# Player Web-Client Mapper — Hybrid Style + Cartesian Consistency

**Date:** 2026-06-06
**Status:** Design approved (pending written-spec review)
**Origin:** Upstream cherry-pick re-review (see `MEMORY.md` →
`project_upstream_rereview_webclient`). First cluster of the
frontend/admin port: the player-facing mapper. Admin map *editor* and
the broader web-client layout are separate later chunks.

---

## 1. Goal

Rebuild the web-client minimap into a clean, readable, zone-scoped map in
a **hybrid visual style** (line-art skeleton + subtle biome tint +
optional faint glyph), driven by the GMCP data we already emit plus a new
"explored zone" snapshot. Along the way, add a **Cartesian-consistency
check engine** (phased: report → enforce) that protects the coordinate
sanity the map depends on, with explicit escape hatches for one-way
exits and intentionally non-Euclidean (toroidal / maze) zones.

We learn from upstream GoMud's mapper work but build our own
implementation, evolving our existing renderer rather than importing
theirs. End-state look should move toward `sample_mud_frontend.png`.

---

## 2. Scope

### In scope
- Hybrid restyle of the player web-client map (evolve `RoomGridSVG`).
- Full-explored-zone coverage via a new GMCP "visited rooms in this zone"
  snapshot + incremental per-move updates.
- Long-exit rendering (multi-cell `-x2`/`-x3` deltas) as proportional
  connectors.
- Toroidal / wrap-exit rendering as **edge stubs**.
- Up/down representation as **subtle per-room ▲/▼ ticks**.
- Cartesian-consistency check engine (report mode + config-gated
  enforcement), reusing the mapper crawl.
- New authoring primitives: `oneway` exit flag, `non_cartesian` zone
  flag, `MapConsistencyEnforce` config knob, `cartcheck` admin command.

### Out of scope (deferred to later chunks)
- Admin visual map-*editor* tool (drag rooms, draw exits, quick-build).
- Broader web-client three-column re-layout / panel-layouts system.
- 3D/isometric map rendering.
- Per-user map style toggle (A/B/C). Hybrid ships as the single default;
  the renderer is structured so a toggle is a later, cheap addition.
- World-wide (cross-zone) map view. Map is zone-scoped.

---

## 3. Current state (verified)

- **Client renderer:** `RoomGridSVG` class in
  `_datafiles/html/public/static/js/gmcp.js`, mounted into the WinBox
  "Map" panel by `_datafiles/html/public/webclient-pure.html` (~line
  406–443). SVG-based: rooms = rounded rects with a per-room fill
  `Color` + text label, joined by thick connector lines, current room
  red (`#c20000`), `+`/`−` zoom, viewBox pan. Coordinate-driven
  (positions = `x * spacing`).
- **GMCP room payload:** `modules/gmcp/gmcp.Room.go`
  (`GMCPRoomModule_Payload`) already ships `num`, `name`, `area`,
  `environment` (biome name), `coords` (x,y,z string), `exits`
  (name→roomId), `exitsv2` (per-exit `dx/dy/dz` deltas), `Contents`.
  Emitted incrementally on each move.
- **Server mapper:** `internal/mapper/` builds a per-zone coordinate map
  by **crawling exit deltas** from a start room (rooms store **no**
  absolute coords — confirmed: `internal/rooms/rooms.go` `Room` struct).
  Key facts:
  - `posDeltas` (mapper.go ~41) maps directions to deltas, including
    multi-cell variants (`north-x2` = 2 cells, `north-x3` = 3 cells) and
    `-gap` variants. This is how "long exits" are authored.
  - `getMapNode` (mapper.go ~902–922) determines a spatial edge: use
    `exit.MapDirection` if it resolves to a `posDelta`, else the exit's
    command name, else **skip the exit for mapping** (`continue`).
  - `crawledRooms` visited-set means a room is placed once; an exit to an
    already-placed room becomes a connector, never a re-placement.
  - The crawl does **not** currently guard against two distinct rooms
    resolving to the same cell (silent overlap) — this is the gap the
    consistency engine closes.
- **Exit struct:** `internal/exit/exit.go` has `Secret`, `Lock`,
  `MapDirection`; **no** one-way/spatial flag today (a one-way exit is
  currently indistinguishable from a missing reciprocal).
- **Zone config:** `internal/rooms/zoneconfig.go` `ZoneConfig` with a
  `Validate()` hook and `DefaultBiome`. New `non_cartesian` flag lands
  here.
- **Instance exemption hook:** `rooms.IsEphemeralRoomId(...)` exists.

---

## 4. Design

### 4.A Visual style — Hybrid (default)
- Rooms: small rounded rects (~15px) with a **subtle, desaturated biome
  tint** fill and a thin amber stroke (`#b8893f`).
- **Optional faint glyph** per room (mapsymbol or biome default symbol),
  low-opacity amber.
- Connectors: thin amber lines between room centers.
- Current room: red fill (`~#d6362a`) with a light stroke.
- Dark ground (`~#15100b`), amber palette — matches
  `sample_mud_frontend.png`.
- Controls: fit / center / zoom (existing zoom retained; add fit +
  center).

### 4.B Renderer — evolve `RoomGridSVG`
Keep the SVG, coordinate-driven approach (lower risk than swapping to
GoMud's canvas; preserves our GMCP pipeline). Changes:
1. Restyle room nodes (size, tint, glyph, stroke) per 4.A.
2. Add a **biome → tint** lookup table (client-side), keyed on the
   `environment` biome name already in the payload.
3. Add glyph rendering from a new per-room `symbol` field (see 4.C).
4. Add **edge-stub** rendering for wrap exits (see 4.E).
5. Add **▲/▼ vertical-exit ticks** (see 4.F).
6. Add **fit** and **center-on-current** controls.
7. Consume a bulk **zone snapshot** message in addition to incremental
   room updates (see 4.C).

### 4.C Data pipeline — GMCP additions
Two additions to `modules/gmcp/gmcp.Room.go` (+ the server crawl):
1. **Per-room map symbol.** Add a `symbol` field to the room payload,
   sourced from `room.MapSymbol` (falling back to the biome default
   symbol, mirroring `getMapNode` mapper.go ~885–900). Lets the client
   draw the faint glyph and pick the right tint without a second lookup.
2. **Explored-zone snapshot.** New GMCP message (working name
   `Room.Zone` / `World.Map`-equivalent) carrying every room the player
   has **visited** in the current zone, each with: roomId, crawled
   `x,y,z`, biome, symbol, and its spatial exits as
   `{toRoomId, dx, dy, dz, kind}` where `kind ∈ {normal, long, wrap,
   vertical}` (server classifies — see 4.E). Built from the existing
   per-zone mapper crawl (`mapper.GetMapper(roomId)` +
   `CrawledRoomIds()`), filtered to the player's **visited** set.
   - Sent once on connect/zone-entry (bulk), then incremental per-move
     `Room` updates extend the visited set.
   - "Visited" tracking: reuse existing player map-memory if present;
     otherwise track visited roomIds per zone on the character. (Confirm
     existing visited-room storage at plan time.)

> Coverage is **full explored zone**, zone-scoped. The client renders the
> union of visited rooms for the current zone.

### 4.D Long exits
No new work — falls out of coordinate-driven rendering. A `-x2`/`-x3`
exit places its destination 2–3 cells away; the connector line spans the
empty intervening cells. Connector simply spans the gap (no special gap
styling). The payload `dx/dy/dz` already carry the true magnitude.

### 4.E Wrap / toroidal exits — edge stubs
A toroidal zone (e.g. Instance Planar Oasis) authors edge wraps as
**real reciprocal compass exits** to the opposite edge. The crawl places
those two rooms on opposite sides of the grid, so a naive connector is a
long line plowing across the whole zone (×10 for a 5×5 face).

**Detection (generic):** an exit is a **wrap** when its *nominal* compass
delta (from name/`MapDirection`, unit or `-xN`) **disagrees** with the
*actual* coordinate delta between the two placed rooms. The server
classifies each exit's `kind` and sends it in the snapshot:
- `normal` — actual delta == nominal delta, magnitude 1.
- `long` — actual == nominal, magnitude > 1 (`-x2`/`-x3`).
- `wrap` — actual != nominal (toroidal edge / torn).
- `vertical` — up/down (dz != 0).

**Rendering:** `wrap` exits draw as a short **teal stub + chevron**
pointing off the room's edge in the exit's nominal direction ("continues,
wraps to far side"), instead of a full connector. Row/column symmetry
communicates the pairing.

**Consistency interaction:** the same `actual != nominal` signal is what
the checker flags. Gated by the zone flag:
- Cartesian zone → `actual != nominal` is a **bug** → warn/panic.
- `non_cartesian: true` zone → expected **wrap** → render as stub, no
  warning.

So the single `non_cartesian` flag does double duty: silences the
validator **and** switches the renderer from long-line to edge-stub. No
extra authoring concept.

### 4.F Up/down — subtle per-room ticks
Every room with a vertical exit gets faint, low-opacity **▲ (up)** and/or
**▼ (down)** ticks in its top-right / bottom-right corner. Decision:
keep them on **all** rooms (accepted trade-off: on a solid-cube middle
level all rooms show both — acceptable as subtle marks). The map shows
one z-level (the player's current floor); up/down commands move between
levels. A small "Level N of M" affordance is a nice-to-have, not
required for v1.

### 4.G Cartesian-consistency engine
One check engine, reusing the mapper crawl and `getMapNode`'s
spatial-edge rule. Two checks:
1. **Coordinate collision** — two *distinct* rooms resolve to the same
   `(x,y,z)` within a zone's crawl. (New; the crawl doesn't guard this.)
2. **Reciprocity / delta agreement** — each spatial exit has a reciprocal
   on the destination, and their deltas are opposite. (Name-level
   reciprocity for a torus passes; the *magnitude* mismatch is the wrap
   signal, exempted by `non_cartesian`.)

Plus one **soft warning**:
3. **Long exit crosses an occupied cell** — a `long` connector's
   straight-line span passes over another room's cell.

**Exemptions (mostly free, by reusing the spatial-edge rule):**
- Non-spatial exits (name + `MapDirection` both non-compass) → not edges.
- Ephemeral/instance rooms (`IsEphemeralRoomId`) → skipped.
- Cross-zone edges (per-zone crawl) → not in any zone's coordinate space.
- Exits flagged `oneway: true` → reciprocity check skipped (collision
  check still applies).
- Zones flagged `non_cartesian: true` → collision + reciprocity hard
  checks skipped (wrap rendering instead).

**Rollout (phased):**
- **Phase 1 — report mode.** Ship the engine, the `cartcheck [zone]`
  admin command, and a **non-fatal startup summary** log. Ship the
  exemption primitives. Boot the real world, read the report, fix
  flagged issues. Mapper renders defensively meanwhile.
- **Phase 2 — enforce.** Flip `MapConsistencyEnforce: off|warn|panic`
  (config) to `panic`, gating server boot on consistency, once the
  report is clean.

---

## 5. New primitives

| Primitive | Location | Purpose |
|---|---|---|
| `oneway: true` (exit) | `internal/exit/exit.go` (+ loader) | Mark an intentional one-way spatial exit; skips reciprocity check. |
| `non_cartesian: true` (zone) | `internal/rooms/zoneconfig.go` | Exempt a zone from hard collision/reciprocity checks; switch wrap rendering on. |
| `MapConsistencyEnforce` | `internal/configs` (Balance/GamePlay) | `off` \| `warn` \| `panic`. Drives Phase 1 → Phase 2. |
| `cartcheck [zone]` | `internal/usercommands` (admin) | Run the consistency report on demand; print collisions, non-reciprocal exits, occupied-cell crossings per zone. |
| `symbol` (GMCP room payload) | `modules/gmcp/gmcp.Room.go` | Per-room glyph for client tint/glyph. |
| Zone snapshot msg (`Room.Zone`) | `modules/gmcp/gmcp.Room.go` | Bulk visited-rooms-in-zone payload. |

> Per `MEMORY.md` admin-command-wiring checklist: `cartcheck` must wire
> handler + registration + helpfile.

---

## 6. Components & interfaces

**Server**
- `internal/mapper`: add a `Consistency(zone)` pass that walks the crawl
  and returns structured findings (collisions, reciprocity failures,
  occupied-cell crossings), honoring exemptions. Add exit-`kind`
  classification used by both the snapshot builder and the checker.
- `modules/gmcp/gmcp.Room.go`: add `symbol` to room payload; add the
  zone-snapshot message + builder (crawl ∩ visited).
- `internal/exit`: `OneWay` field + loader plumbing.
- `internal/rooms/zoneconfig.go`: `NonCartesian` field; call consistency
  in startup validation per `MapConsistencyEnforce`.
- `internal/usercommands/cartcheck.go`: admin command → report.
- `internal/configs`: `MapConsistencyEnforce` knob.

**Client** (`gmcp.js` `RoomGridSVG` + `webclient-pure.html` wiring)
- Biome→tint table; glyph rendering; restyled nodes/connectors.
- Edge-stub rendering for `kind:wrap`; proportional connectors for
  `kind:long`; ▲/▼ ticks for `kind:vertical`.
- Consume zone snapshot (bulk set) + incremental room updates (extend).
- Fit / center-on-current controls.

Each unit is independently testable: the consistency pass is pure over a
crawl; the snapshot builder is pure over (crawl, visited-set); the
renderer is a pure function of (snapshot, current room).

---

## 7. Edge cases
- **Solid 3D cube:** every middle-level room shows ▲▼ — accepted.
- **Toroidal face:** wrap edges → stubs; zone is `non_cartesian`.
- **One-way cardinal exit:** `oneway: true` → reciprocity skipped,
  collision still enforced.
- **One-way portal (non-compass):** auto-exempt (not a spatial edge).
- **Cross-zone exit:** not in either zone's coordinate space.
- **Unvisited rooms:** excluded from the snapshot (fog-of-war by
  exploration).
- **Instance zones:** absent at boot; `IsEphemeralRoomId` skip if
  validated at runtime.
- **Stale instance saves:** per project SOP, wipe `mobs.instances` /
  `rooms.instances` before smoke-testing map changes.

---

## 8. Testing
- **Server unit:** consistency pass over synthetic crawls — clean grid
  (pass), injected collision (fail), missing reciprocal (fail), `oneway`
  (pass), torus under `non_cartesian` (pass) vs Cartesian (fail),
  occupied-cell crossing (soft warn). Exit-`kind` classification table.
- **Server unit:** snapshot builder returns only visited rooms with
  correct kinds/deltas.
- **Client:** render snapshots (grid, long exit, torus face, vertical
  cube level) and assert node/stub/tick counts. (JS lint already in CI.)
- **Boot test (pre-push SOP):** server starts clean past data load;
  `cartcheck` report on real world reviewed.
- **Smoke:** in-game map panel for a normal zone, a long-exit zone, and
  Instance Planar Oasis (wraps render as stubs).

---

## 9. Phasing (suggested plan order)
1. Server: exit-`kind` classification + consistency pass (pure, tested).
2. `cartcheck` command + non-fatal startup report (`MapConsistencyEnforce:
   warn`). Run on real world; fix findings.
3. New primitives: `oneway`, `non_cartesian` + loaders/validation.
4. GMCP: `symbol` field + zone snapshot message/builder.
5. Client: hybrid restyle + snapshot consumption + long/wrap/vertical
   rendering + fit/center.
6. Smoke (normal + long + toroidal zones), then flip
   `MapConsistencyEnforce: panic` once clean.

---

## 10. Open items to confirm at plan time
- Existing player **visited-room** storage (reuse vs add per-zone set).
- Exact GMCP message name/namespace for the zone snapshot
  (`Room.Zone` vs `World.Map`) and client subscription.
- Whether `cartcheck` should also lint `-gap` exits.
