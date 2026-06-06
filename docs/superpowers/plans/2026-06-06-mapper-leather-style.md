# Tooled-Leather Mapper Restyle + Connection Types — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restyle the web mapper into the approved antique tooled-leather look and render every connection with biome/exit-aware styling, plus party-member position markers.

**Architecture:** Two parts. **Part A (server)** enriches the existing `Zone.Map` GMCP snapshot with per-exit flags (`locked`/`secret`/`oneway`/`gate`/`toZone`/`stub`) and party-member room IDs. **Part B (client)** rewrites the *drawing* inside `RoomGridSVG` (`gmcp.js`) from the dark-amber tiles into the leather style (emboss relief, frayed hide, craquelure, ornate border/compass/legend, raised current room, biome+connection-type paths, party figures), keeping the existing data plumbing (`setZoneSnapshot`, floor filter `cz`, fog, fit/center/zoom).

**Tech Stack:** Go (server: `internal/mapper`, `internal/exit`, `modules/gmcp`), vanilla JS + SVG (client: `_datafiles/html/public/static/js/gmcp.js`), Go `testing`.

**Spec:** `docs/superpowers/specs/2026-06-06-mapper-leather-style-design.md`
**Visual source of truth (committed):** `docs/superpowers/specs/2026-06-06-mapper-leather-mockups/` — `01-surface-and-rooms.html`, `02-connection-types.html`, `03-emboss-craquelure.html`. These are runnable mockups; the client tasks **port their drawing code** (adapting the mock's flat `px(x)`/`py(y)` to the renderer's `room.x * this.spacing`).

---

## Conventions for the implementer

- Branch is already `feature/mapper-leather-style`. Do NOT switch branches.
- `git add` ONLY the files named in each task (the working tree has unrelated uncommitted world-state files — never `git add -A`).
- Go: `go build ./...` and `go test ./internal/mapper/...` after server changes.
- JS: there is **no JS unit-test harness** in this repo. Validate client JS with `node --check "_datafiles/html/public/static/js/gmcp.js"` (syntax) and the final manual smoke. Don't invent a JS test harness.
- The mockups use a seeded RNG and SVG `feTurbulence`/`clipPath`/emboss; port them faithfully.
- Co-author trailer on commits: `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`.
- Smoke SOP: before booting, wipe instance saves: `rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*`.

---

## File Structure

**Part A — server**

| File | Change |
|---|---|
| `internal/mapper/mapper.node.go` | Add `Gate bool` to `nodeExit`. |
| `internal/mapper/mapper.go` | Populate `Gate` in `getMapNode` from `exitInfo.ExitMessage`. |
| `internal/mapper/mapper.snapshot.go` | Extend `SnapshotExit` (locked/secret/oneway/gate/toZone/stub); emit stub + cross-zone exits. |
| `internal/mapper/mapper.snapshot_test.go` | Tests for the new exit fields + stub logic. |
| `modules/gmcp/gmcp.Zone.go` | Add `Party []int` to payload; populate from the party system. |

**Part B — client (all in one file + the html)**

| File | Change |
|---|---|
| `_datafiles/html/public/static/js/gmcp.js` | New leather render path inside `RoomGridSVG` (helpers, surface, tokens, connections, glue). |
| `_datafiles/html/public/webclient-pure.html` | Pass `obj.Map.party` into `setZoneSnapshot`. |
| `internal/mapper/context.md`, `CLAUDE.md`, `PATCH_NOTES.md` | Docs. |

---

# PART A — Server data

### Task 1: Add `Gate` to the mapper exit node

**Files:**
- Modify: `internal/mapper/mapper.node.go`
- Modify: `internal/mapper/mapper.go` (`getMapNode`)

- [ ] **Step 1: Add the field.** In `internal/mapper/mapper.node.go`, add to `nodeExit` (after `OneWay`):

```go
	OneWay         bool   // intentional one-way spatial exit (skips reciprocity check)
	Gate           bool   // exit is a threshold/gate (has an ExitMessage)
	Direction      positionDelta
```

- [ ] **Step 2: Populate it.** In `internal/mapper/mapper.go`, in `getMapNode`, extend the `nodeExit` literal:

```go
		exitNode := nodeExit{
			RoomId:         exitInfo.RoomId,
			Secret:         exitInfo.Secret,
			LockDifficulty: int(exitInfo.Lock.Difficulty),
			OneWay:         exitInfo.OneWay,
			Gate:           exitInfo.ExitMessage != ``,
		}
```

- [ ] **Step 3: Build.** Run `go build ./...` — Expected: success.

- [ ] **Step 4: Commit.**

```bash
git add internal/mapper/mapper.node.go internal/mapper/mapper.go
git commit -m "feat(mapper): nodeExit.Gate from exit ExitMessage (threshold detection)"
```

---

### Task 2: Enrich `SnapshotExit` + emit stub/cross-zone exits

**Files:**
- Modify: `internal/mapper/mapper.snapshot.go`
- Modify: `internal/mapper/mapper.snapshot_test.go`

Current `SnapshotExit` has `ToRoomId/DX/DY/DZ/Kind`. Current `Snapshot` drops any exit whose destination isn't a *visited same-zone* room. We add per-exit flags and also emit dropped exits as **stubs** (so the client can draw "more this way" and cross-zone labels).

- [ ] **Step 1: Write the failing test.** Append to `internal/mapper/mapper.snapshot_test.go`:

```go
func TestSnapshotExitFlagsAndStubs(t *testing.T) {
	nodes := map[int]*mapNode{
		1: node(1, 0, 0, 0, map[string]nodeExit{
			"north":  {RoomId: 2, Direction: d(0, -1, 0)},                      // visited -> full edge
			"east":   {RoomId: 3, Direction: d(1, 0, 0), LockDifficulty: 25},   // visited, locked
			"south":  {RoomId: 9, Direction: d(0, 1, 0)},                       // UNVISITED -> stub
			"up":     {RoomId: 4, Direction: d(0, 0, 1), Secret: true, OneWay: true, Gate: true},
		}),
		2: node(2, 0, -1, 0, map[string]nodeExit{}),
		3: node(3, 1, 0, 0, map[string]nodeExit{}),
		4: node(4, 0, 0, 1, map[string]nodeExit{}),
		// room 9 deliberately NOT in crawledRooms either (uncrawled)
	}
	m := mkMapper(nodes)
	visited := map[int]struct{}{1: {}, 2: {}, 3: {}, 4: {}}

	snap := m.Snapshot(visited)
	var r1 *SnapshotRoom
	for i := range snap {
		if snap[i].RoomId == 1 {
			r1 = &snap[i]
		}
	}
	if r1 == nil {
		t.Fatal("room 1 missing")
	}
	byTo := map[int]SnapshotExit{}
	for _, e := range r1.Exits {
		byTo[e.ToRoomId] = e
	}
	if !byTo[3].Locked {
		t.Error("east exit to 3 should be Locked")
	}
	if e := byTo[4]; !e.Secret || !e.OneWay || !e.Gate {
		t.Errorf("up exit flags wrong: %+v", e)
	}
	if !byTo[9].Stub {
		t.Error("south exit to unvisited/uncrawled room 9 should be a Stub")
	}
	if byTo[2].Stub {
		t.Error("north exit to visited room 2 should NOT be a stub")
	}
}
```

- [ ] **Step 2: Run it — verify it fails.** Run `go test ./internal/mapper/ -run TestSnapshotExitFlagsAndStubs -v` — Expected: FAIL (unknown fields `Locked`/`Secret`/`OneWay`/`Gate`/`Stub`).

- [ ] **Step 3: Extend the struct.** In `internal/mapper/mapper.snapshot.go`, replace the `SnapshotExit` struct with:

```go
// SnapshotExit is one classified spatial edge for the web map renderer.
type SnapshotExit struct {
	ToRoomId int      `json:"to"`
	DX       int      `json:"dx"`
	DY       int      `json:"dy"`
	DZ       int      `json:"dz"`
	Kind     ExitKind `json:"kind"`
	Locked   bool     `json:"locked,omitempty"`
	Secret   bool     `json:"secret,omitempty"`
	OneWay   bool     `json:"oneway,omitempty"`
	Gate     bool     `json:"gate,omitempty"`
	Stub     bool     `json:"stub,omitempty"`   // destination not a visited same-zone room
	ToZone   string   `json:"tozone,omitempty"` // set when the destination is in a different zone
}
```

- [ ] **Step 4: Rewrite the exit loop.** In `Snapshot`, replace the existing `for _, e := range n.Exits { ... }` block with:

```go
		// Capture the source room's zone once (for cross-zone detection).
		srcZone := ""
		if room := rooms.LoadRoom(id); room != nil {
			srcZone = room.Zone
		}

		for _, e := range n.Exits {
			se := SnapshotExit{
				ToRoomId: e.RoomId,
				DX:       e.Direction.x,
				DY:       e.Direction.y,
				DZ:       e.Direction.z,
				Locked:   e.LockDifficulty > 0,
				Secret:   e.Secret,
				OneWay:   e.OneWay,
				Gate:     e.Gate,
			}
			dst, crawled := r.crawledRooms[e.RoomId]
			_, vis := visited[e.RoomId]
			if crawled && vis {
				actual := positionDelta{x: dst.Pos.x - n.Pos.x, y: dst.Pos.y - n.Pos.y, z: dst.Pos.z - n.Pos.z}
				se.Kind = classifyKind(e.Direction, actual)
			} else {
				se.Stub = true
				se.Kind = classifyKind(e.Direction, e.Direction) // nominal placement
			}
			if dr := rooms.LoadRoom(e.RoomId); dr != nil && dr.Zone != "" && dr.Zone != srcZone {
				se.ToZone = dr.Zone
			}
			sr.Exits = append(sr.Exits, se)
		}
```

(The existing `sr.Name`/`sr.Biome`/`sr.Tags` block above this loop is unchanged. Note `srcZone` reuses a `rooms.LoadRoom(id)` — you may merge it into the existing room-load block that already runs for biome/name/tags rather than loading twice; if you do, capture `srcZone = room.Zone` there.)

- [ ] **Step 5: Run the test — verify pass.** Run `go test ./internal/mapper/ -run TestSnapshotExitFlagsAndStubs -v` — Expected: PASS. Then `go test ./internal/mapper/...` — Expected: PASS (existing snapshot tests still green; note the older `TestSnapshotFiltersVisitedAndClassifies` asserted `len(r1.Exits)==1` for a one-visited-exit room — if that room also had an unvisited exit it will now also get a stub; check that test's fixture: room 1 there has `north`→2 (visited) and `east-x3`→3 (unvisited) → east-x3 now becomes a **stub** rather than being dropped, so that test's `len(r1.Exits) != 1` assertion will break. **Update that test**: room 1 now has 2 exits — the normal north (kind normal, not stub) and the east-x3 stub. Assert the non-stub exit is to room 2 with `ExitNormal`, and that the other is `Stub`).

  Concretely, update `TestSnapshotFiltersVisitedAndClassifies` so the exits assertion reads:

```go
	// north->2 is a full edge; east-x3->3 (unvisited) is now a stub.
	var normalExits, stubExits int
	for _, e := range r1.Exits {
		if e.Stub {
			stubExits++
		} else {
			normalExits++
			if e.ToRoomId != 2 || e.Kind != ExitNormal {
				t.Fatalf("expected non-stub exit to room 2 normal, got %+v", e)
			}
		}
	}
	if normalExits != 1 || stubExits != 1 {
		t.Fatalf("expected 1 normal + 1 stub exit, got %d/%d", normalExits, stubExits)
	}
```

- [ ] **Step 6: Commit.**

```bash
git add internal/mapper/mapper.snapshot.go internal/mapper/mapper.snapshot_test.go
git commit -m "feat(mapper): snapshot exit flags (locked/secret/oneway/gate/toZone) + stub exits"
```

---

### Task 3: Party-member positions in the Zone payload

**Files:**
- Modify: `modules/gmcp/gmcp.Zone.go`

The party API (verified): `parties.Get(userId int) *parties.Party` (nil if none) and `party.GetMembers() []int`; each member's room is `users.GetByUserId(uId).Character.RoomId` (mirrors `modules/gmcp/gmcp.Party.go`).

- [ ] **Step 1: Add the payload field.** In `modules/gmcp/gmcp.Zone.go`, extend `GMCPZoneModule_Payload`:

```go
type GMCPZoneModule_Payload struct {
	Zone     string                `json:"zone"`
	CurrentZ int                   `json:"cz"`
	Party    []int                 `json:"party,omitempty"` // room IDs currently holding party members
	Rooms    []mapper.SnapshotRoom `json:"rooms"`
}
```

- [ ] **Step 2: Populate it.** In `buildAndSend`, after the `visited` set is built and before constructing the payload, add (and add `"github.com/GoMudEngine/GoMud/internal/parties"` to the imports):

```go
	// Party-member room positions (excluding self), de-duplicated.
	party := []int{}
	if p := parties.Get(evt.UserId); p != nil {
		seen := map[int]bool{}
		for _, uId := range p.GetMembers() {
			if uId == evt.UserId {
				continue
			}
			if mu := users.GetByUserId(uId); mu != nil {
				rid := mu.Character.RoomId
				if !seen[rid] {
					seen[rid] = true
					party = append(party, rid)
				}
			}
		}
	}
```

And set it in the payload literal:

```go
	payload := GMCPZoneModule_Payload{
		Zone:     room.Zone,
		CurrentZ: cz,
		Party:    party,
		Rooms:    m.Snapshot(visited),
	}
```

- [ ] **Step 3: Build.** Run `go build ./...` — Expected: success (no import cycle: gmcp already imports users/rooms/mapper; parties is a leaf).

- [ ] **Step 4: Commit.**

```bash
git add modules/gmcp/gmcp.Zone.go
git commit -m "feat(gmcp): include party-member room positions in Zone.Map payload"
```

> **Part A is complete and shippable** — the snapshot now carries everything the client render needs. Boot test (optional now, required before merge): `go run .` and confirm clean load + `Zone.Map` frames carry the new fields.

---

# PART B — Client leather render

> All Part B steps modify `_datafiles/html/public/static/js/gmcp.js` (except Task 9 also touches `webclient-pure.html`). Validate each with `node --check "_datafiles/html/public/static/js/gmcp.js"`. The new render is not visually complete until Task 8 wires it together; keep `node --check` green at every step.

### Task 4: Port the leather drawing toolkit

**Files:** Modify `_datafiles/html/public/static/js/gmcp.js`

Add a self-contained toolkit (module-level functions + a few `RoomGridSVG` statics) ported from the mockups. These do not yet change rendering.

- [ ] **Step 1: Add the seeded RNG + emboss + texture helpers.** Near the top of `gmcp.js` (above the `RoomGridSVG` class), add the verbatim helpers from `docs/superpowers/specs/2026-06-06-mapper-leather-mockups/03-emboss-craquelure.html`: the `rng(seed)` function, the constants `HI="#efce8c"`, `SH="#140b04"`, and the functions `embLine(g,x1,y1,x2,y2,col,w,op,d)`, `embCirc(g,cx,cy,r,faceStroke,faceFill,w,d)`, `embText(g,x,y,attrs,s,face,d)`, `hidePath(W,H,m,fray,nickP,rnd)`, and `craquelure(g,W,H,step,jit,rnd)`. Convert them to module-scope functions that take an explicit `NS`/`el`/`txt` (define small module-level `el(tag,attrs)` and `txt(attrs,s)` helpers identical to the mockups). These are pure drawing helpers — no `this`.

- [ ] **Step 2: Add the icon drawers.** Also port (module-scope) the connection icon drawers from `docs/superpowers/specs/2026-06-06-mapper-leather-mockups/02-connection-types.html`: `arrowhead(g,x,y,ang,col)`, and the body of `drawConn`'s sub-icons — extract them into `drawLockedDoor(g,a,b)`, `drawArchGate(g,a,b)`, and keep the per-type line styling logic available (you will call these from Task 7). Each takes endpoint pixel coords `a=[x1,y1]`, `b=[x2,y2]`.

- [ ] **Step 3: Add palette + render statics to `RoomGridSVG`.** After the class, add the leather palette as statics (so they're tunable in one place):

```js
RoomGridSVG.LEATHER = {
  ink: "#c9a86a", ink2: "#9c8048", title: "#e8d2a0", roomFill: "#2a1d12",
  label: "#e8d8b8", locked: "#d0633f", water: "#6f99c0", trail: "#a98a55",
  road: "#c9a86a", ridge: "#a98a55", plain: "#9c8048",
  legendBg: "#241810", party: "#6bb0a0", partyDk: "#243f3a",
  emboss: 0.6, fray: 3.4, nickP: 0.08, crackStep: 24, crackJit: 9, vig: 0.5
};
RoomGridSVG.SERVICE_GLYPHS = { bank: "$", shop: "S", trainer: "T", storage: "▢" };
RoomGridSVG.SERVICE_ORDER = ["bank", "shop", "trainer", "storage"];
```

- [ ] **Step 4: Validate.** Run `node --check "_datafiles/html/public/static/js/gmcp.js"` — Expected: no errors.

- [ ] **Step 5: Commit.**

```bash
git add _datafiles/html/public/static/js/gmcp.js
git commit -m "feat(webclient): leather drawing toolkit (emboss/hide/craquelure/icons)"
```

---

### Task 5: Surface renderer (`_renderSurface`)

**Files:** Modify `gmcp.js`

Build the static leather surface (defs, hide, texture, border, title, compass, legend) as a method that draws into a dedicated `<g>` so it can be rebuilt only when the zone/floor changes.

- [ ] **Step 1: Add the method.** Port the surface-drawing portion of `01-surface-and-rooms.html`'s `render()` (everything up to the per-room loop: defs/gradients/grain/vignette/sheen/hide+rim/clipped texture+craquelure/border+corners/title/compass/legend) into a `RoomGridSVG.prototype._renderSurface = function(zone, level)`. Use `this.svg` as the root, a fixed internal coordinate canvas (the mock uses `W=320,H=300` — keep an internal `this._W/this._H` design canvas and map room coords into it via the existing `spacing`, OR keep the SVG viewBox as the world and draw the surface sized to the current view; **recommended:** draw the surface into a separate full-viewBox `<g class="surface">` sized to the panel, independent of room world-coords, and the rooms/edges into a `<g class="world">` that uses world coords — see Task 8 for how these compose). Title uses `zone` + `"~ Level " + level + " ~"`. Pull all colors from `RoomGridSVG.LEATHER`. Seed the rng from a stable value (e.g. a hash of `zone+level`) so the fray/craquelure are stable per zone-floor.

- [ ] **Step 2: Validate.** `node --check ...` — Expected: clean.

- [ ] **Step 3: Commit.**

```bash
git add _datafiles/html/public/static/js/gmcp.js
git commit -m "feat(webclient): _renderSurface — leather hide, border, compass, legend"
```

---

### Task 6: Room tokens (normal / service / current / party / stairs / fog)

**Files:** Modify `gmcp.js`

Replace the room-drawing in `addRoom` (both new and update branches) with leather tokens.

- [ ] **Step 1: Port token drawing.** Using `embCirc`/`embText` and `RoomGridSVG.LEATHER`, draw per `01-surface-and-rooms.html`'s room loop:
  - normal: embossed circle r6.5 fill roomFill gold stroke.
  - service (from `room.tags` via `_serviceFor`): embossed circle r7 + outer ring r9.5 + embossed gold glyph.
  - current (`room.cur`/the current room id): whisper glow + outer emphasis ring (emboss `d*1.3`) + raised token (r7.5, face `#3a2a18`, emboss `d*2.4`) + visible italic name label.
  - `<title>` tooltip = `room.name` on every token.
  - stairs ▲/▼ ticks from vertical exits (reuse the existing `_drawVerticalTick` data path / `ExitsMeta`).
  - party figure (verdigris head+shoulders) at the room corner when the room id is in `this._party` (set in Task 8).
  - fog: per-room opacity by Chebyshev distance from the current room (helper `_fog(room)`), as in the mock.
  Keep the existing centered geometry (`room.x * this.spacing`, `room.y * this.spacing`).

- [ ] **Step 2: Validate.** `node --check ...` — clean.

- [ ] **Step 3: Commit.**

```bash
git add _datafiles/html/public/static/js/gmcp.js
git commit -m "feat(webclient): leather room tokens — raised current room, party figures, stairs, fog"
```

---

### Task 7: Connection-type rendering

**Files:** Modify `gmcp.js`

Replace `_drawEdge` / the connector half of `_drawEdgesForRoom` with connection-type routing driven by the new `SnapshotExit` fields.

- [ ] **Step 1: Add `_drawConnection(srcRoom, exitMeta)`.** Given the source room and one `ExitsMeta` entry `{to,dx,dy,dz,kind,locked,secret,oneway,gate,stub,tozone}`, route:
  - `kind==='vertical'` → `_drawVerticalTick` (stairs); return.
  - `stub` (and not vertical) → dim dashed stub in the (dx,dy) direction; if `tozone` set, draw the `"→ <tozone>"` label at the stub end (cross-zone road-out). Return.
  - else it connects to a placed room `to`:
    - `locked` → near-half normal + far-half red + `drawLockedDoor` at midpoint (port from `02`).
    - `secret` → faint dotted + `?` at midpoint.
    - `oneway` → embossed line + `arrowhead` at ~0.72 toward `to`.
    - `gate` → embossed line + `drawArchGate` at midpoint.
    - otherwise biome path: pick kind via `_edgeKind(srcRoom.biome, R(to).biome)` (water→trail→road→ridge→plain precedence) and draw road (embossed tan + dotted centre; if `kind==='long'` add two league ticks), trail (dashed), water (dashed blue), ridge (embossed), plain (embossed) — all per `02`/`01`.
  Use `this.rooms.get(to)` for the destination coords (only for non-stub). Reuse `this.drawnEdges`/`drawnWrapStubs`/`drawnVerticalTicks` dedup sets so resends don't duplicate.

- [ ] **Step 2: Route from `_drawEdgesForRoom`.** Replace its body so that for a room with `ExitsMeta`, it calls `this._drawConnection(meRoom, e)` for each entry (drop the old `_drawEdge` kind-routing); keep the non-`ExitsMeta` fallback (Room.Info path) as a plain thin connector.

- [ ] **Step 3: Validate.** `node --check ...` — clean.

- [ ] **Step 4: Commit.**

```bash
git add _datafiles/html/public/static/js/gmcp.js
git commit -m "feat(webclient): connection-type rendering (biome paths, locked/secret/oneway/gate, stubs, cross-zone)"
```

---

### Task 8: Wire it together in `setZoneSnapshot` + html

**Files:** Modify `gmcp.js`, `_datafiles/html/public/webclient-pure.html`

- [ ] **Step 1: Compose layers.** Ensure the SVG has two groups: a `surface` `<g>` (full-panel, leather) and a `world` `<g>` (rooms/connections in world coords, pannable/zoomable via the existing viewBox math). In the constructor, create `this.surfaceGroup` and reuse `connectionsGroup`/`roomsGroup` inside the world group. NOTE: the leather surface (border/compass/legend) should stay fixed to the panel, while rooms pan/zoom — so apply the existing viewBox/zoom only to the world group's transform, and keep the surface drawn in panel coordinates. (Simplest robust approach: surface in its own `<svg>` or a non-transformed `<g>` sized to `viewBox 0 0 W H` matching the panel; world group transformed by the zoom/center. Pick one and keep geometry consistent.)

- [ ] **Step 2: Rewrite `setZoneSnapshot(zone, snapshotRooms, currentZ, party)`.** Add the `party` param; store `this._party = {}` keyed by roomId from the `party` array. On zone OR floor change: clear the world group, redraw the surface via `_renderSurface(zone, currentZ)`. Keep the existing floor filter + two-pass (place all nodes, then draw connections). After placing, call `_drawEdgesForRoom` for each floor room (now routing through `_drawConnection`). Keep `_applyZoom()` for the world group.

- [ ] **Step 3: Pass party from the html.** In `webclient-pure.html`, update the `Zone.Map` handler call:

```js
                    gr.setZoneSnapshot(obj.Map.zone, obj.Map.rooms || [], obj.Map.cz || 0, obj.Map.party || []);
```

- [ ] **Step 4: Validate.** `node --check "_datafiles/html/public/static/js/gmcp.js"` — clean. Confirm `webclient-pure.html` branch matches the surrounding handler style.

- [ ] **Step 5: Commit.**

```bash
git add _datafiles/html/public/static/js/gmcp.js _datafiles/html/public/webclient-pure.html
git commit -m "feat(webclient): compose leather surface + world layers; consume party in snapshot"
```

---

### Task 9: Smoke test + tuning

**Files:** none (config/runtime) — plus any dial tweaks to `RoomGridSVG.LEATHER`.

- [ ] **Step 1: Boot + connect.** Wipe instance saves, `go build ./...`, `go run .`. Connect with the **web client** (not raw telnet), hard-refresh (Ctrl+F5).

- [ ] **Step 2: Verify against the mockups.** Walk a city zone (Thornwall) and check: leather surface + frayed edge + craquelure; embossed border/compass/title (with floor); raised current room follows you; service `$`/`S`/`▢` tokens; biome paths (road vs trail vs water); a locked door (e.g. Chrysalis Den's locked west exit) shows the door + red far-half; stubs point at unexplored exits; stairs ▲▼ where vertical exits exist; party figure appears when a second character is in a nearby room (test with a second login or an AI tester); fog dims distant rooms; hover shows room names.

- [ ] **Step 3: Tune dials if needed.** Adjust `RoomGridSVG.LEATHER` (emboss, fray, crackStep/crackJit, vig) and re-refresh until it matches the locked look. Commit any tweaks:

```bash
git add _datafiles/html/public/static/js/gmcp.js
git commit -m "tune(webclient): leather mapper dials per smoke"
```

---

### Task 10: Docs

**Files:** Modify `internal/mapper/context.md`, `CLAUDE.md`, `PATCH_NOTES.md`

- [ ] **Step 1: `internal/mapper/context.md`** — note the new `SnapshotExit` fields (locked/secret/oneway/gate/toZone/stub) and `nodeExit.Gate`. ALSO document that the mapper has **two consumers**: the in-game ASCII `map` command (`internal/usercommands/skill.map.go` → `GetLimitedMap`/`GetLegend`, Perception-scaled detail, symbols `@`=You / `☺`=Player·Party·NPC / `☠`=Mob / `☹`=Friend + biome/mapsymbol glyphs) and the web leather map (`Zone.Map` snapshot → `RoomGridSVG`). Briefly describe each and that connection-type/party data is web-only.
- [ ] **Step 2: `CLAUDE.md`** — extend the map section: the web mapper renders an antique tooled-leather style; connection types are inferred per-exit; party positions come via `Zone.Map.party`; the visual source of truth is `docs/superpowers/specs/2026-06-06-mapper-leather-mockups/`.
- [ ] **Step 3: `PATCH_NOTES.md`** — dated entry: leather mapper restyle, connection-type styling, party markers.
- [ ] **Step 4: Commit.**

```bash
git add internal/mapper/context.md CLAUDE.md PATCH_NOTES.md
git commit -m "docs: leather mapper restyle, connection types, party markers"
```

---

### Task 11: `help map` helpfile — both maps documented

**Files:** Modify `_datafiles/world/dogmud/templates/help/map.template` (exists)

The helpfile must clearly cover BOTH the in-game ASCII `map` command and the web-client leather map, so players understand every element of each.

- [ ] **Step 1: Read the current file.** Read `_datafiles/world/dogmud/templates/help/map.template` and keep its `<ansi ...>` styling + the existing usage/related lines.
- [ ] **Step 2: Document the in-game map.** Keep/expand the existing section: `map` / `map wide`; Perception-scaled detail; symbols — `@` = You, `☺` = Player / Party Member / NPC, `☠` = Mob, `☹` = Friend; biome glyphs + per-room map symbols + the legend; secret/hidden areas may not show. (Verify symbols against `internal/usercommands/skill.map.go` `OverrideSymbol` calls.)
- [ ] **Step 3: Add a web-map section.** A clearly-headed section explaining the web client's leather map and EVERY element: the raised tile = your current room; gold `$`/`S`/`▢` = bank/shop/storage; the verdigris figure = a party member in that room; `▲`/`▼` = stairs up/down; path styles — solid road, dashed trail, dashed-blue waterway, ridge path, a long "highway"; a small door icon (red beyond it) = a **locked** exit; faint dotted + `?` = a **secret** passage; an arrowhead = a **one-way** passage; an archway = a **gated/threshold** exit; a dashed stub = an **unexplored** exit ("more this way"), labelled `→ Zone` when it leads to another area; dimmer rooms are farther from you; hover a room for its name; the `fit`/`ctr`/`−`/`+` controls. Note the map shows one floor at a time (use stairs to change levels).
- [ ] **Step 4: Wrap at 80 cols; keep `<ansi>` styling consistent.** Confirm the template has no unclosed tags and reads cleanly.
- [ ] **Step 5: Commit.**

```bash
git add _datafiles/world/dogmud/templates/help/map.template
git commit -m "docs(help): document both the in-game ASCII map and the web leather map"
```

---

## Self-Review (completed by plan author)

**Spec coverage:** §2 surface/frame → Tasks 4,5. §3 rooms (raised current, service, stairs, fog, tooltip) → Task 6. §4 party → Tasks 3,6,8. §5 connection types (all rows) → Tasks 2 (data) + 7 (render). §6 server data → Tasks 1,2,3. §7 client rewrite → Tasks 4–8. §8 out-of-scope respected. Docs → Tasks 10, 11 (Task 11 = `help map` covering BOTH the in-game ASCII map and the web leather map; context.md covers both consumers).

**Placeholder scan:** Server tasks have complete code + tests. Client tasks reference the **committed mockup files** for the bulky drawing routines (faithful ports, exact file + function named) rather than re-pasting ~400 lines — these are precise pointers to committed source, not "TBD". The one genuine judgment call (surface-fixed vs world-pannable layering, Task 8 Step 1) is called out explicitly with a recommended approach.

**Type consistency:** `SnapshotExit` fields (locked/secret/oneway/gate/stub/tozone) are defined in Task 2 and consumed by the client in Task 7 with matching lowercase JSON keys. `party` is `[]int` in Task 3 and read as `obj.Map.party` in Task 8. `nodeExit.Gate` (Task 1) → `SnapshotExit.Gate` (Task 2). `RoomGridSVG.LEATHER`/`SERVICE_GLYPHS` defined in Task 4, used in 5–7.

**Note on JS testing:** no harness exists; client validation is `node --check` + the Task 9 smoke, per repo convention — not a gap.
