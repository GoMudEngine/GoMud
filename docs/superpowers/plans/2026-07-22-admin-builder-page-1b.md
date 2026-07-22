# Admin Builder Page (1b) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A standalone `/build` web page where an admin edits the world as a graphical, plane-aware SVG map — click a room to edit, click a ghost cell to create, wire exits — persisting to room YAML templates and re-rendering live over `Build.*` GMCP.

**Architecture:** Server-authoritative. The page authenticates via the existing admin Basic-Auth and opens its own GMCP session; new RoleAdmin-gated `Build.*` GMCP packages mutate rooms via `SaveRoomTemplate`/`ConnectRoom`/`NewRoom`/`ValidatePlacement` and reply with `Build.Result` + a refreshed `Zone.Map`. Foundation first (mapper reads authored coords), then transport, then UI.

**Tech Stack:** Go (GoMud fork), GMCP over WebSocket, vanilla JS + inline SVG (matching `gmcp.js`), `go test`.

**Spec:** `docs/superpowers/specs/2026-07-22-admin-builder-page-1b-design.md`

**Verified seams:** `HandleIAC` switch `modules/gmcp/gmcp.go:216` (client→server cases; `userIdForConnection` `:517`); `dispatchGMCP` `gmcp.go:527`; `ConnectRoom(from,to,exitName,mapDirection...) error` `roommanager.go:833` (one-way — call twice for reciprocal); `NewRoom(zone) *Room` `rooms.go:126`; `GetNextRoomId() int` `roommanager.go:129`; `SaveRoomTemplate(Room)` `save_and_load.go:177`; `ValidatePlacement(plane,x,y,z,exclude) error` `internal/rooms/placement.go`; `/admin/*` route + `doBasicAuth` `internal/web/web.go:292`,`auth.go`; mapper crawl `Start` `internal/mapper/mapper.go:249`; `Zone.Map` snapshot `mapper.snapshot.go`,`gmcp.Zone.go`.

---

## File Structure

- `internal/mapper/mapper.go` — `Start` reads authored coords (1a Task 6). (Task 1)
- `internal/web/web.go`, `internal/web/build.go` *(new)* — `/build` page route + handler. (Task 2)
- `_datafiles/html/public/build.html` *(new)* — the builder page (canvas + inspector + toolbar + GMCP client). (Tasks 2,4,5)
- `modules/gmcp/gmcp.Build.go` *(new)* — `Build.*` handlers + result/refresh senders. (Task 3)
- `modules/gmcp/gmcp.go` — route `Build.*` in `HandleIAC`. (Task 3)
- `_datafiles/html/public/static/js/builder.js` *(new)* — editor SVG canvas + inspector logic. (Tasks 4,5)

---

## Task 1: Mapper reads authored coords (folds in 1a Task 6)

**Files:** Modify `internal/mapper/mapper.go` (`Start`, `mapper.go:249-339`), `getMapNode`; Test `internal/mapper/mapper_authored_test.go` *(new)*.

- [ ] **Step 1: Failing parity test.** Create `mapper_authored_test.go`: build hand-placed `mapNode`s whose authored coords match a valid crawl; call the new authored path; assert every node's `Pos` equals its authored coord (proving `Start` used stored coords, not a re-crawl). Use the `mkMapper`/`node` helpers from `mapper.consistency_test.go`.

```go
package mapper

import "testing"

func TestStart_UsesAuthoredCoords(t *testing.T) {
	// rooms 1 (0,0,0) and 2 (1,0,0), east/west reciprocal — authored coords set.
	// (Back with fixture rooms or the in-memory test rooms the consistency tests use.)
	// After Start(), node 2's Pos must be (1,0,0) taken from Room.X/Y/Z, not re-derived.
	// Assert node positions == authored.
}
```

- [ ] **Step 2: Run — FAIL** (Start still crawls). `go test ./internal/mapper/ -run TestStart_UsesAuthoredCoords`

- [ ] **Step 3: Implement.** Extract the current crawl body into `startByCrawl()` (the fallback + migration/validation tool). Rewrite `Start` to: keep the BFS traversal to discover the reachable room set + exits, but set `node.Pos = positionDelta{x: room.X, y: room.Y, z: room.Z}` and `node.Plane = room.Plane` from stored coords (via `rooms.LoadRoom`), sizing `RoomGrid` from authored min/max. If a reached room has no authored coords (all-zero AND plane 0 while others differ — an un-migrated straggler), fall back to `startByCrawl()` for the whole zone and log a warning. Show the full rewritten `Start` in the commit.

- [ ] **Step 4: Carry `plane` in the snapshot** (if 1a didn't already): `SnapshotRoom.Plane` in `mapper.snapshot.go` + `Zone.Map` payload in `gmcp.Zone.go`, populated from `node.Plane`.

- [ ] **Step 5: Run mapper tests — PASS.** `go test ./internal/mapper/`

- [ ] **Step 6: Verify cartcheck stays clean world-wide.** Boot with `MapConsistencyEnforce=panic`, wipe instances first; confirm `ValidateZoneConsistency errors=0` (the ironwind fix already cleared the one blocker). Kill after "Server Ready".

- [ ] **Step 7: Commit.**
```bash
git add internal/mapper/
git commit -m "feat(mapper): read authored coords (1a Task 6, builder foundation)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: `/build` page route + admin auth + shell

**Files:** Create `internal/web/build.go`; Modify `internal/web/web.go:292` area; Create `_datafiles/html/public/build.html` (shell only this task).

- [ ] **Step 1: Register the route.** In `web.go`, next to the `/admin/*` registrations, add `/build` served by `doBasicAuth` (admin-gated) returning `build.html`. Confirm from `auth.go` that `doBasicAuth` enforces `RoleAdmin` (it does — `auth.go:44-49`). Do NOT wrap in `RunWithMUDLocked` (the page is static; only the later GMCP mutations touch game state, and those run on MainWorker). Serve the page's static JS from the existing `/static/` file server.

```go
// in web.go, alongside /admin/* :
mux.HandleFunc("/build", doBasicAuth(func(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, filepath.Join(publicHtmlPath, "build.html"))
}))
```
(Match the exact router + path helpers used for `/admin/*` — mirror that registration.)

- [ ] **Step 2: Page shell.** `build.html`: a full-doc page (own `<head>`) that opens a WebSocket to `/ws`, performs the GMCP `Core.Hello` + `Char.Login` handshake **as the authenticated admin** (reuse the handshake from `webclient-pure.html` — the Basic-Auth identity must map to the GMCP login; if the WS login needs credentials, pass the admin session through — verify how `webclient-pure.html` logs in and mirror it). Lay out the toolbar + left canvas `<div id="canvas">` + right `<aside id="inspector">` per Layout A. No editing logic yet — just: connect, receive a `Zone.Map`, log it.

> **Implementer note:** the auth bridge (Basic-Auth page identity → GMCP `Char.Login` session) is the one integration risk. Read `webclient-pure.html`'s login/handshake (`SendGMCP('Char.Login', …)`) and `internal/inputhandlers/login.go`; confirm whether `/build` can auto-login the Basic-Auth'd admin or must present the same login step. Resolve this before Task 3.

- [ ] **Step 3: Manual smoke.** Boot; browse `http://localhost/build` (or the dev port); Basic-Auth as the admin account; confirm the shell loads, the WS connects, and a `Zone.Map` arrives in the console.

- [ ] **Step 4: Commit.**
```bash
git add internal/web/ _datafiles/html/public/build.html
git commit -m "feat(build): /build admin page route + shell + GMCP session

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: `Build.*` GMCP handlers (server)

**Files:** Create `modules/gmcp/gmcp.Build.go`; Modify `modules/gmcp/gmcp.go` (`HandleIAC` switch); Test `modules/gmcp/gmcp.Build_test.go` *(new)*.

- [ ] **Step 1: Failing unit tests** for the pure decision seams. Create `gmcp.Build_test.go` covering: a non-admin user is refused; `Build.Room.Create` calls `ValidatePlacement` and refuses an occupied cell; an update round-trips fields; a spatial exit crossing Euclidean planes is refused. Structure the handlers so the core logic is a testable function taking `(uid int, payload …)` and returning `(BuildResult, error)`, with the GMCP/connection plumbing thin around it.

- [ ] **Step 2: Run — FAIL.**

- [ ] **Step 3: Implement handlers.** `gmcp.Build.go`:

```go
package gmcp

// BuildResult is echoed to the client after every mutation.
type BuildResult struct {
	Ok    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	RoomId int   `json:"roomId,omitempty"` // e.g. the newly created room
}

// requireAdmin resolves the connection's user and returns its id iff admin.
func requireAdmin(connectionId uint64) (int, bool) {
	uid := userIdForConnection(connectionId)
	if uid <= 0 {
		return 0, false
	}
	u := users.GetByUserId(uid)
	if u == nil || u.Role != users.RoleAdmin { // confirm exact field/method: users.RoleAdmin
		return 0, false
	}
	return uid, true
}

// buildRoomCreate: create a room at (plane,x,y,z) as a neighbour of fromRoomId in dir.
func buildRoomCreate(uid, fromRoomId int, dir string, plane, x, y, z int) BuildResult {
	if err := rooms.ValidatePlacement(plane, x, y, z, 0); err != nil {
		return BuildResult{Error: err.Error()}
	}
	from := rooms.LoadRoomTemplate(fromRoomId)
	if from == nil {
		return BuildResult{Error: "source room not found"}
	}
	nr := rooms.NewRoom(from.Zone) // inherit the source zone
	nr.X, nr.Y, nr.Z, nr.Plane = x, y, z, plane
	nr.Title, nr.Description = "Untitled", ""
	rooms.SaveRoomTemplate(*nr) // assigns id via GetNextRoomId when 0
	// reciprocal exits (ConnectRoom is one-way — call both directions):
	rooms.ConnectRoom(fromRoomId, nr.RoomId, dir)
	rooms.ConnectRoom(nr.RoomId, fromRoomId, mapper.GetReciprocalExit(dir))
	return BuildResult{Ok: true, RoomId: nr.RoomId}
}
```

Add `buildRoomUpdate(uid, roomId int, fields …)` (load template, set title/description/biome/mapsymbol/maplegend/flags/nouns/idlemessages/music, `SaveRoomTemplate`), `buildRoomDelete`, `buildExitAdd` (spatial: same-plane delta check via authored coords + reciprocal `ConnectRoom`; portal: named exit, any target incl. cross-plane, no reciprocal auto), `buildExitRemove`. Each returns `BuildResult`. **Reuse existing gaps-fix:** always route through `SaveRoomTemplate` so biome/exit-delete persist (the in-game bugs).

- [ ] **Step 4: Route in `HandleIAC`.** In `gmcp.go`'s switch (`gmcp.go:274`), add cases mirroring the `Char.Action.Try` pattern (`gmcp.go:419`): parse the JSON payload, `uid, ok := requireAdmin(connectionId)`, dispatch to the `buildX` function, then send the result + a refreshed map:

```go
case `Build.Room.Create`:
	uid, ok := requireAdmin(connectionId)
	if !ok { sendBuildResult(connectionId, BuildResult{Error: "admin only"}); break }
	// parse payload -> fromRoomId, dir, plane,x,y,z
	res := buildRoomCreate(uid, fromRoomId, dir, plane, x, y, z)
	sendBuildResult(connectionId, res)
	sendZoneMapRefresh(uid) // re-push Zone.Map for the current plane
```

- [ ] **Step 5: Result + refresh senders.** `sendBuildResult` and `sendZoneMapRefresh` via `dispatchGMCP` (`gmcp.go:527`) — the former emits `Build.Result`, the latter re-emits the `Zone.Map` for the admin (reuse the existing `Zone.Map` build path). Clear the affected zone's mapper cache first (`mapper.ClearCache()` / `GetMapper(…, true)`) so the re-render reflects the new room/exit.

- [ ] **Step 6: Run tests — PASS; build.** `go test ./modules/gmcp/ && go build ./...`

- [ ] **Step 7: Commit.**
```bash
git add modules/gmcp/
git commit -m "feat(gmcp): Build.* room-authoring packages (admin-gated)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 4: Editor SVG canvas

**Files:** Create `_datafiles/html/public/static/js/builder.js`; Modify `build.html` to load it.

The canvas is a new component sharing `RoomGridSVG`'s SVG/leather DNA (`gmcp.js`) but editor-grade. Build it incrementally; each step is verifiable in the browser.

- [ ] **Step 1: Render a plane.** `BuilderCanvas` class: consume the `Zone.Map` snapshot (`{zone, rooms:[{num,x,y,z,plane,biome,name,exits:[…]}]}`), filter to the **selected plane**, and draw biome-tinted rounded-rect room nodes at `x*CELL, y*CELL` with the room id label, plus styled exit lines (spatial solid, secret dashed, one-way arrow, portal stub). Reuse `gmcp.js` biome colors + connection styling. Pan (pointer-drag on the SVG world group) + zoom (wheel). Verify: the current zone renders.

- [ ] **Step 2: Plane selector + zone context.** Toolbar dropdown lists planes present in the snapshot (+ labels); switching re-filters the canvas. Show the current zone name.

- [ ] **Step 3: Selection.** `onRoomClick(room)` → highlight (gold ring) + fire a `roomSelected` event the inspector listens to. (This is the hook `RoomGridSVG` already exposes but the play client leaves unwired — here it's central.)

- [ ] **Step 4: Ghost cells.** For each room, for each compass direction with no exit and a free adjacent cell (checked against the snapshot's occupied cells on that plane), draw a dashed `+` node. Clicking it computes `(plane, x+dx, y+dy, z)` and the `dir`, and sends `Build.Room.Create {fromRoomId, dir, plane,x,y,z}`. On the `Zone.Map` refresh, the new room appears; auto-select it (match by returned `roomId` from `Build.Result`).

- [ ] **Step 5: Verify in browser.** Load `/build`, pan/zoom, switch planes, click a room (selection fires), click a ghost cell (a new room appears and is selected). Server log shows the `Build.Room.Create`.

- [ ] **Step 6: Commit.**
```bash
git add _datafiles/html/public/static/js/builder.js _datafiles/html/public/build.html
git commit -m "feat(build): editor SVG canvas — render, select, plane switch, ghost-create

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 5: Inspector (fields + exits) + Save

**Files:** Modify `builder.js`, `build.html`.

- [ ] **Step 1: Field form.** On `roomSelected`, populate the inspector: Title, Description (textarea), Biome (dropdown of valid biomes), Map symbol + legend, room-flag checkboxes (bank/storage/pvp/character-room), Nouns (key→text repeatable rows), Idle messages (list), Music. Track dirty state.

- [ ] **Step 2: Save.** The Save button sends one `Build.Room.Update {roomId, …all fields}`; on `Build.Result{ok}` clear dirty + toast; on error show it inline. (Explicit save; create + exit ops are immediate.)

- [ ] **Step 3: Exits editor.** List current exits (dir/target + flags, each with remove `✕`). `+ add exit`: choose **spatial** (direction dropdown → resolves the adjacent room from the snapshot; sends `Build.Exit.Add {roomId, dir, toRoomId}`) or **portal** (name field + a room picker across planes; sends `Build.Exit.Add {roomId, portalName, toRoomId}`), plus secret/one-way/lock/message flags. Remove sends `Build.Exit.Remove`. Each applies immediately and refreshes the map.

- [ ] **Step 4: Verify in browser.** Select a room, edit title/desc/biome + Save (persists — check the YAML + a play client), add a spatial exit and a cross-plane portal, remove an exit — all reflect live on the canvas.

- [ ] **Step 5: Commit.**
```bash
git add _datafiles/html/public/static/js/builder.js _datafiles/html/public/build.html
git commit -m "feat(build): inspector fields + exits editor + explicit save

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 6: End-to-end + content/UX gate (REQUIRED)

**Files:** none (verification).

- [ ] **Step 1: Full build + tests.** `go build ./... && go test ./internal/mapper/ ./internal/rooms/ ./modules/gmcp/`

- [ ] **Step 2: Boot-clean** (wipe instances; `MapConsistencyEnforce=panic`) — errors=0.

- [ ] **Step 3: REQUIRED browser playtest** (CLAUDE.md gate). As an admin on `/build`: sketch a small room cluster by clicking ghost cells; edit titles/descriptions/biomes + Save; wire a spatial exit and a cross-plane **portal** into an interior; delete an exit and a room. Confirm: the map re-renders live after each op; `ValidatePlacement` blocks an overlap; a non-admin (or logged-out) cannot reach `/build` or send `Build.*`; and the created rooms are **walkable in a real play client** (open the play webclient, walk into them, read the prose). Read every interaction as a confused human would; fix what it surfaces, re-run.

- [ ] **Step 4: Report.** Summarize findings + fixes. Do not claim done on boot-clean alone.

---

## Self-Review notes
- **Spec coverage:** Task 6-mapper → T1; page+auth → T2; Build.* protocol → T3; canvas (select/plane/ghost-create) → T4; inspector fields + exits + save → T5; e2e + gate → T6.
- **Ordering:** T1 (foundation) → T2 (page) → T3 (server protocol) before T4/T5 (frontend that calls it). T4 before T5 (inspector needs selection).
- **Type consistency:** `BuildResult`, `requireAdmin`, `buildRoomCreate/Update/Delete/ExitAdd/ExitRemove` defined in T3 and routed in T3 Step 4; the client `Build.*` senders (T4/T5) match the payloads in T3's handlers and the spec's protocol table.
- **Risk notes:** (1) the Basic-Auth→GMCP-login bridge (T2 Step 2 — resolve before T3). (2) `ConnectRoom` is one-way — reciprocal needs two calls (handled in `buildRoomCreate`/`buildExitAdd`). (3) map-cache invalidation before the `Zone.Map` refresh (T3 Step 5). (4) the SVG canvas (T4) is the largest frontend piece — build incrementally, verify each step in the browser.
