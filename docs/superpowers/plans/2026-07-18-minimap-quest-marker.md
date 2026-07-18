# Minimap Quest Marker Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Mark the player's focused quest objective on the web leather minimap (destination marker + turn-by-turn next-step arrow) and add a web Quests panel that chooses the focus, reusing the existing `Character.LastQuestId` / `hint` plumbing.

**Architecture:** Two pure server primitives — `questengine.(*Engine).ResolveQuestTarget` (which room the focused quest points at) and `mapper.NextStep` (the next hop toward it, a thin exported wrapper over the cached `mapper.GetPath` the `pathto` command already uses). The existing `Char.Quests` GMCP payload is migrated to the authoritative `questengine` store and extended with `id/hint/focused/target_room/next_room/next_dir`. A new inbound `Char.Quests.Focus` GMCP message sets `LastQuestId` (so `hint` and the marker follow). The web client gains a Quests panel (left column, under Map) and marker rendering in `RoomGridSVG`.

**Tech Stack:** Go (GoMud engine), YAML quest data, vanilla-JS web client (SVG minimap), GMCP over telnet/websocket.

**Spec:** `docs/superpowers/specs/2026-07-18-minimap-quest-marker-design.md`

---

## Design notes & deliberate refinements over the spec

These are decisions made after verifying the code; they refine (not contradict) the approved spec. Implement as written here.

1. **Reuse `mapper.GetPath`, don't write a new BFS.** `mapper.GetPath(start, end...) ([]pathStep, error)` (`internal/mapper/mapper.path.go:196`) is the exact cached breadth-first pathfinder the `pathto` mob command uses (`internal/mobcommands/pathto.go:46`). The spec's "BFS next-step helper" collapses to a thin wrapper, `mapper.NextStep`, that returns the first hop's room id + exit name. `pathStep{exitName, roomId, waypoint}` has **unexported** fields, so the wrapper must live in package `mapper`.
2. **`map_target` lives on `questengine.QuestStep`, and the GMCP build loop migrates to `questengine`.** The authoritative live quest store read by `hint` is `questengine` (`internal/usercommands/hint.go:52,70` → `questengine.GetEngine().GetQuest(int)`). The current `Char.Quests` build loop uses the *other*, token-keyed `internal/quests` package (`modules/gmcp/gmcp.Char.go:608-643`). Migrating the loop to `questengine.GetEngine().GetQuest(questId)` (Task 5) makes GMCP consistent with `hint` and is the only place the new step field (`MapTarget`) and `Hint` are visible. The progress map (`GetQuestProgress() map[int]string`) is shared by both, so the swap is clean.
3. **Target resolution is deterministic.** Triggers are a flat quest-level list (`QuestDef.Triggers`), not per-step, each gated by `Conditions.Has` step tokens. Resolution: (1) the current step's `map_target` wins; (2) else a `room_enter` trigger whose `Conditions.Has` contains the current step token `"{questId}-{stepId}"` → its `Room`; (3) else `0` (no marker). Combat-in-place steps (e.g. quest 32 "First Blood") have no `room_enter` trigger, so they need an explicit `map_target` — Task 10 authors one as the SOP exercise.
4. **Next-step is single-zone.** `mapper.GetPath` routes within the room's zone. Cross-zone targets return `found=false` → the client draws no arrow (and no destination marker, since the target is off this zone's map). The spec's cross-zone "point at the boundary exit" behaviour is deferred to the follow-up list — the newbie hub (Pothole Coulee) is single-zone, so this covers the real use case. This divergence is intentional and called out here so it is not a silent gap.
5. **No JS test harness exists** in this repo (confirmed: no quest UI, no JS tests). Server tasks use Go TDD; web-client tasks (7-9) are verified by boot + browser + the adversarial-playtest SOP in Task 10. Nothing is pushed without user approval.

---

## File structure

**Server (Go):**
- `internal/questengine/types.go` — add `MapTarget` to `QuestStep` (modify).
- `internal/questengine/map_target.go` — **new**: `(*Engine).ResolveQuestTarget`. Pure quest→room logic; no mapper/GMCP dependency.
- `internal/questengine/map_target_test.go` — **new**: unit tests for resolution.
- `internal/mapper/mapper.path.go` — add exported `NextStep` + unexported `firstHop` (modify).
- `internal/mapper/mapper.path_test.go` — add `firstHop` / `NextStep` guard tests (modify).
- `modules/gmcp/gmcp.Char.go` — extend `GMCPCharModule_Payload_Quest`; migrate + extend the `Char.Quests` build loop (modify).
- `modules/gmcp/gmcp.go` — add inbound `Char.Quests.Focus` case (modify).

**Web client:**
- `_datafiles/html/public/webclient-pure.html` — Quests panel `<section>`; `renderQuests()`; `Char.Quests` handler; focus click → `SendGMCP` (modify).
- `_datafiles/html/public/static/js/dashboard.js` — register `"quests"` panel id + icon (modify).
- `_datafiles/html/public/static/js/gmcp.js` — `RoomGridSVG.setQuestMarker` + destination marker + next-step arrow + legend rows + palette entry (modify).
- `_datafiles/html/public/static/css/dashboard.css` — optional `#panel-quests` sizing (modify).

**Content:**
- One newbie quest YAML (e.g. `_datafiles/world/dogmud/quests/32-first_blood.yaml`) — add a `map_target` (Task 10, SOP exercise).

---

## Task 1: Add `map_target` to the quest step schema

**Files:**
- Modify: `internal/questengine/types.go:22-26`

- [ ] **Step 1: Add the field**

In `internal/questengine/types.go`, change `QuestStep`:

```go
type QuestStep struct {
	Id          string `yaml:"id"`
	Description string `yaml:"description,omitempty"`
	Hint        string `yaml:"hint,omitempty"`
	MapTarget   int    `yaml:"map_target,omitempty"` // room the minimap marker points at during this step (0 = infer/none)
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/questengine/`
Expected: builds cleanly (no test yet — this is a struct-only change consumed in Task 2).

- [ ] **Step 3: Commit**

```bash
git add internal/questengine/types.go
git commit -m "feat(quests): add optional map_target to quest step schema"
```

---

## Task 2: Quest target resolution helper (`ResolveQuestTarget`)

**Files:**
- Create: `internal/questengine/map_target.go`
- Test: `internal/questengine/map_target_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/questengine/map_target_test.go`:

```go
package questengine

import "testing"

func newTargetTestEngine() *Engine {
	e := NewEngine()
	e.RegisterQuest(&QuestDef{
		QuestId: 32,
		Name:    "First Blood",
		Steps: []QuestStep{
			{Id: "start", MapTarget: 4201}, // explicit target
			{Id: "arrive"},                 // inferred via room_enter trigger
			{Id: "nowhere"},                // no target
			{Id: "end"},
		},
		Triggers: []TriggerDef{
			{
				Event:      "room_enter",
				Room:       4207,
				Conditions: Conditions{Has: []string{"32-arrive"}},
			},
		},
	})
	return e
}

func TestResolveQuestTarget_ExplicitMapTarget(t *testing.T) {
	e := newTargetTestEngine()
	if got := e.ResolveQuestTarget(32, "start"); got != 4201 {
		t.Fatalf("map_target step: want 4201, got %d", got)
	}
}

func TestResolveQuestTarget_RoomEnterInference(t *testing.T) {
	e := newTargetTestEngine()
	if got := e.ResolveQuestTarget(32, "arrive"); got != 4207 {
		t.Fatalf("room_enter inference: want 4207, got %d", got)
	}
}

func TestResolveQuestTarget_NoTarget(t *testing.T) {
	e := newTargetTestEngine()
	if got := e.ResolveQuestTarget(32, "nowhere"); got != 0 {
		t.Fatalf("no target: want 0, got %d", got)
	}
}

func TestResolveQuestTarget_TerminalAndUnknown(t *testing.T) {
	e := newTargetTestEngine()
	if got := e.ResolveQuestTarget(32, "end"); got != 0 {
		t.Fatalf("end step: want 0, got %d", got)
	}
	if got := e.ResolveQuestTarget(999, "start"); got != 0 {
		t.Fatalf("unknown quest: want 0, got %d", got)
	}
	if got := e.ResolveQuestTarget(32, ""); got != 0 {
		t.Fatalf("empty step: want 0, got %d", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/questengine/ -run TestResolveQuestTarget -v`
Expected: FAIL — `e.ResolveQuestTarget undefined`.

- [ ] **Step 3: Write the implementation**

Create `internal/questengine/map_target.go`:

```go
package questengine

import "fmt"

// ResolveQuestTarget returns the room id the given quest points at during
// currentStep, for the minimap destination marker. It returns 0 when there is
// no resolvable target (the client then draws no marker — no guessing).
//
// Resolution order:
//  1. The current step's explicit map_target.
//  2. Inference: a room_enter trigger gated on the current step token
//     (conditions.has contains "{questId}-{currentStep}") — its room.
//  3. 0.
func (e *Engine) ResolveQuestTarget(questId int, currentStep string) int {
	if currentStep == "" || currentStep == "end" {
		return 0
	}
	q := e.quests[questId]
	if q == nil {
		return 0
	}

	// 1. Explicit map_target on the current step.
	for _, step := range q.Steps {
		if step.Id == currentStep {
			if step.MapTarget != 0 {
				return step.MapTarget
			}
			break
		}
	}

	// 2. room_enter trigger gated on the current step.
	stepToken := fmt.Sprintf("%d-%s", questId, currentStep)
	for i := range q.Triggers {
		t := &q.Triggers[i]
		if t.Event != "room_enter" || t.Room == 0 {
			continue
		}
		for _, tok := range t.Conditions.Has {
			if tok == stepToken {
				return t.Room
			}
		}
	}

	return 0
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/questengine/ -run TestResolveQuestTarget -v`
Expected: PASS (all four tests).

- [ ] **Step 5: Commit**

```bash
git add internal/questengine/map_target.go internal/questengine/map_target_test.go
git commit -m "feat(quests): ResolveQuestTarget (map_target + room_enter inference)"
```

---

## Task 3: Next-step pathfinding wrapper (`mapper.NextStep`)

**Files:**
- Modify: `internal/mapper/mapper.path.go` (add exported `NextStep` + unexported `firstHop`)
- Test: `internal/mapper/mapper.path_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/mapper/mapper.path_test.go`:

```go
func Test_firstHop(t *testing.T) {
	n, dir, ok := firstHop([]pathStep{
		{exitName: "east", roomId: 2},
		{exitName: "south", roomId: 3},
	})
	require.True(t, ok)
	assert.Equal(t, 2, n)
	assert.Equal(t, "east", dir)

	_, _, ok = firstHop(nil)
	assert.False(t, ok)

	_, _, ok = firstHop([]pathStep{})
	assert.False(t, ok)
}

func Test_NextStep_SameRoom(t *testing.T) {
	_, _, ok := NextStep(5, 5)
	assert.False(t, ok, "same-room target yields no next step")
}

func Test_NextStep_NoPath(t *testing.T) {
	// Unknown start room => GetPath errors => no next step.
	_, _, ok := NextStep(999999, 4)
	assert.False(t, ok)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/mapper/ -run 'Test_firstHop|Test_NextStep' -v`
Expected: FAIL — `firstHop`/`NextStep` undefined.

- [ ] **Step 3: Write the implementation**

Append to `internal/mapper/mapper.path.go`:

```go
// firstHop extracts the next room + exit direction from a computed path.
func firstHop(path []pathStep) (nextRoomId int, exitName string, found bool) {
	if len(path) == 0 {
		return 0, "", false
	}
	return path[0].roomId, path[0].exitName, true
}

// NextStep returns the next room to head toward on the shortest in-zone path
// from fromRoomId to toRoomId, and the compass exit name to take. found is
// false when from == to, when the target is unreachable, or when the target is
// in another zone (the per-zone mapper cannot route across zones — callers then
// draw no arrow). Thin wrapper over the cached GetPath used by `pathto`.
func NextStep(fromRoomId, toRoomId int) (nextRoomId int, exitName string, found bool) {
	if fromRoomId == toRoomId {
		return 0, "", false
	}
	path, err := GetPath(fromRoomId, toRoomId)
	if err != nil {
		return 0, "", false
	}
	return firstHop(path)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/mapper/ -run 'Test_firstHop|Test_NextStep' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/mapper/mapper.path.go internal/mapper/mapper.path_test.go
git commit -m "feat(mapper): NextStep wrapper exposing next hop + exit direction"
```

---

## Task 4: Extend the `Char.Quests` GMCP payload struct

**Files:**
- Modify: `modules/gmcp/gmcp.Char.go:995-999`

- [ ] **Step 1: Extend the struct**

In `modules/gmcp/gmcp.Char.go`, replace `GMCPCharModule_Payload_Quest`:

```go
type GMCPCharModule_Payload_Quest struct {
	Id          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Completion  int    `json:"completion"`
	Hint        string `json:"hint,omitempty"`
	Focused     bool   `json:"focused,omitempty"`
	// Focused quest only; omitted/zero when there is no resolvable target.
	TargetRoom int    `json:"target_room,omitempty"`
	NextRoom   int    `json:"next_room,omitempty"`
	NextDir    string `json:"next_dir,omitempty"`
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./modules/gmcp/`
Expected: builds cleanly (fields unused until Task 5).

- [ ] **Step 3: Commit**

```bash
git add modules/gmcp/gmcp.Char.go
git commit -m "feat(gmcp): extend Char.Quests payload with id/hint/focus/target fields"
```

---

## Task 5: Migrate + populate the `Char.Quests` build loop

**Files:**
- Modify: `modules/gmcp/gmcp.Char.go:604-648`
- Check imports at top of `modules/gmcp/gmcp.Char.go`: ensure `questengine`, `mapper` are imported; the `quests` import may become unused **in this loop** but is likely used elsewhere in the file — only remove it if `go build` reports it unused.

- [ ] **Step 1: Replace the build loop**

Replace lines 604-648 (the `if all || g.wantsGMCPPayload(\`Char.Quests\`, ...)` block) with:

```go
	if all || g.wantsGMCPPayload(`Char.Quests`, gmcpModule) {

		payload.Quests = []GMCPCharModule_Payload_Quest{}

		engine := questengine.GetEngine()
		focusId := user.Character.LastQuestId

		for questId, questStep := range user.Character.GetQuestProgress() {

			qDef := engine.GetQuest(questId)
			if qDef == nil {
				continue
			}

			// Secret quests are not sent.
			if qDef.Secret {
				continue
			}

			completedSteps := 0
			totalSteps := len(qDef.Steps)

			questPayload := GMCPCharModule_Payload_Quest{
				Id:          questId,
				Name:        qDef.Name,
				Description: qDef.Description,
				Completion:  0,
				Focused:     questId == focusId,
			}

			for _, step := range qDef.Steps {
				completedSteps++
				if step.Id == questStep {
					questPayload.Description = step.Description
					questPayload.Hint = step.Hint
					break
				}
			}

			if totalSteps > 0 {
				questPayload.Completion = int(math.Floor(float64(completedSteps)/float64(totalSteps)) * 100)
			}

			// Marker data is computed for the focused quest only.
			if questPayload.Focused {
				if target := engine.ResolveQuestTarget(questId, questStep); target != 0 {
					questPayload.TargetRoom = target
					if next, dir, ok := mapper.NextStep(user.Character.RoomId, target); ok {
						questPayload.NextRoom = next
						questPayload.NextDir = dir
					}
				}
			}

			payload.Quests = append(payload.Quests, questPayload)
		}

		if !all {
			return payload.Quests, `Char.Quests`
		}
	}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./modules/gmcp/`
Expected: builds cleanly. If `quests` is now unused in the file, remove it from the import block; if `mapper`/`questengine` were missing, add them. Re-run until clean.

- [ ] **Step 3: Boot-test the server (data-load + emit path)**

Wipe instance saves per the SOP, then boot:

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . 2>&1 | head -60
```

Expected: server reaches `quests.LoadDataFiles()` / `questengine` load and the world-boot lines without panic. Ctrl-C after it's up. (This confirms the migrated loop doesn't break quest loading.)

- [ ] **Step 4: Commit**

```bash
git add modules/gmcp/gmcp.Char.go
git commit -m "feat(gmcp): Char.Quests uses questengine + carries focus/target/next-step"
```

---

## Task 6: Inbound `Char.Quests.Focus` handler

**Files:**
- Modify: `modules/gmcp/gmcp.go` — add a `case` in the inbound switch alongside `Char.Action.Try` (after line 447)

- [ ] **Step 1: Add the inbound case**

In `modules/gmcp/gmcp.go`, immediately after the `Char.Action.Try` case block (ends ~line 447), add:

```go
		case `Char.Quests.Focus`:
			var req struct {
				Id int `json:"id"`
			}
			if err := json.Unmarshal(payload, &req); err != nil {
				break
			}
			uid := userIdForConnection(connectionId)
			if uid <= 0 {
				break
			}
			u := users.GetByUserId(uid)
			if u == nil {
				break
			}
			// Only allow focusing an active quest.
			if _, ok := u.Character.GetQuestProgress()[req.Id]; !ok {
				break
			}
			u.Character.LastQuestId = req.Id
			// Re-emit Char.Quests so the panel, marker, and `hint` all follow.
			events.AddToQueue(GMCPCharUpdate{
				UserId:     uid,
				Identifier: `Char.Quests`,
			})
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./modules/gmcp/`
Expected: builds cleanly (`events`, `users` already imported in this file — the neighbouring cases use them).

- [ ] **Step 3: Boot-test**

Run: `go run . 2>&1 | head -40`
Expected: server boots without panic. Ctrl-C.

- [ ] **Step 4: Commit**

```bash
git add modules/gmcp/gmcp.go
git commit -m "feat(gmcp): inbound Char.Quests.Focus sets LastQuestId + re-emits Char.Quests"
```

---

## Task 7: Add the Quests panel shell (HTML + panel registry + CSS)

**Files:**
- Modify: `_datafiles/html/public/webclient-pure.html` (left column, after `panel-map` ~line 266)
- Modify: `_datafiles/html/public/static/js/dashboard.js:11` (`SIDE_PANELS`) and `:14-21` (`PANEL_ICONS`)
- Modify: `_datafiles/html/public/static/css/dashboard.css` (optional sizing)

- [ ] **Step 1: Add the panel markup**

In `webclient-pure.html`, insert directly after the closing `</section>` of `panel-map` (~line 266), before `panel-vitals`:

```html
        <section class="dash-panel" id="panel-quests" data-panel="quests">
          <div class="dash-panel-head"><span class="ph-title">Quests</span>
            <span class="ph-btns"><span class="ph-btn ph-collapse">▾</span><span class="ph-btn ph-popout">⧉</span></span></div>
          <div class="dash-panel-body"><div id="quests-list"><div class="quests-empty">No active quests.</div></div></div>
        </section>
```

- [ ] **Step 2: Register the panel id + icon**

In `dashboard.js` line 11, add `"quests"` right after `"map"`:

```js
var SIDE_PANELS = ["map", "quests", "vitals", "art", "chat", "status", "trig"];
```

In `PANEL_ICONS` (lines 14-21), add an entry (match the existing key/emoji style used there):

```js
  quests: "❖",
```

- [ ] **Step 3: Add panel CSS**

In `dashboard.css`, near the `#panel-map`/`#panel-vitals` flex rules (~line 148-149), add:

```css
#panel-quests { flex: 0 0 auto; }
#panel-quests .dash-panel-body { max-height: 220px; overflow: auto; }
.quests-empty { color: #5a4a32; font: italic 12px Georgia, serif; padding: 4px 2px; }
```

- [ ] **Step 4: Verify the panel renders empty**

Boot the server (`go run .`), open the web client, log in. Expected: a "Quests" panel appears in the left column under Map showing "No active quests." It drags/collapses/pops-out like its siblings (auto-wired via `data-panel`). No console errors.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/html/public/webclient-pure.html _datafiles/html/public/static/js/dashboard.js _datafiles/html/public/static/css/dashboard.css
git commit -m "feat(web): Quests panel shell in the left column under Map"
```

---

## Task 8: Render the quest list + focus click round-trip

**Files:**
- Modify: `_datafiles/html/public/webclient-pure.html` — add `renderQuests()` and a `"Char.Quests"` handler entry in `GMCPUpdateHandlers` (~line 690)

- [ ] **Step 1: Add the render function**

In `webclient-pure.html`, near the other panel renderers (`updateVitalsWindow`, `renderInventory`), add:

```js
function renderQuests() {
    var host = document.getElementById("quests-list");
    if (!host) { return; }
    var data = (GMCPStructs.Char && GMCPStructs.Char.Quests) ? GMCPStructs.Char.Quests : null;
    var quests = Array.isArray(data) ? data : [];
    if (!quests.length) {
        host.innerHTML = '<div class="quests-empty">No active quests.</div>';
        return;
    }
    // Focused quest first, then by name.
    quests.sort(function (a, b) {
        if (!!b.focused - !!a.focused) { return (!!b.focused) - (!!a.focused); }
        return (a.name || "").localeCompare(b.name || "");
    });
    var html = "";
    quests.forEach(function (q) {
        var focused = !!q.focused;
        var pct = Math.max(0, Math.min(100, q.completion || 0));
        html += '<div class="qrow' + (focused ? " focus" : "") + '" data-quest-id="' + q.id + '">'
              + '<span class="qdot">' + (focused ? "◉" : "◎") + '</span>'
              + '<span class="qname">' + escapeHtml(q.name || "") + '</span>'
              + '<span class="qbar"><i style="width:' + pct + '%"></i></span>'
              + '</div>';
        if (focused && q.hint) {
            var mark = q.target_room ? ' <span class="qonmap">◆ marked on map</span>' : '';
            html += '<div class="qhint">' + escapeHtml(q.hint) + mark + '</div>';
        }
    });
    host.innerHTML = html;
    Array.prototype.forEach.call(host.querySelectorAll(".qrow"), function (row) {
        row.addEventListener("click", function () {
            var id = parseInt(row.getAttribute("data-quest-id"), 10);
            if (!isNaN(id)) { SendGMCP("Char.Quests.Focus", { id: id }); }
        });
    });
}
```

Note: reuse the existing `escapeHtml` helper in this file. If none exists, add a minimal one next to `renderQuests`:

```js
function escapeHtml(s) {
    return String(s).replace(/[&<>"']/g, function (c) {
        return { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c];
    });
}
```

- [ ] **Step 2: Register the GMCP handler**

In the `GMCPUpdateHandlers` object (~line 690, next to the `"Char.Vitals"` / `"Char.Inventory"` entries), add:

```js
    "Char.Quests": function() { renderQuests(); },
```

- [ ] **Step 3: Add row CSS**

In `dashboard.css`, near the Task 7 rules, add:

```css
.qrow { display:flex; align-items:center; gap:6px; padding:4px 4px; border-radius:4px; cursor:pointer; font:12px Georgia,serif; color:#d2c3a4; }
.qrow:hover { background:rgba(201,168,106,.07); }
.qrow.focus { background:rgba(203,159,66,.15); box-shadow:inset 2px 0 0 #cb9f42; color:#e8d2a0; }
.qrow .qdot { width:14px; text-align:center; color:#5a4a32; }
.qrow.focus .qdot { color:#cb9f42; }
.qrow .qname { flex:1; }
.qbar { width:46px; height:5px; background:#160e08; border:1px solid #3a2a18; border-radius:3px; overflow:hidden; }
.qbar > i { display:block; height:100%; background:linear-gradient(#f4dd92,#cb9f42); }
.qhint { padding:2px 6px 5px 26px; font:italic 11px Georgia,serif; color:#9a8a6a; }
.qonmap { color:#6bb0a0; font-style:normal; }
```

- [ ] **Step 4: Verify in-game**

Boot, log in a character that has active quests (a fresh newbie who finished "find your footing" has the seven trail quests). Expected: the panel lists them, focused quest first with a filled ◉ dot + its hint + "◆ marked on map" when a target resolves. Clicking another row moves the focus (dot/highlight jump; `hint` with no arg — typed in the game input — now returns that quest). No console errors.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/html/public/webclient-pure.html _datafiles/html/public/static/css/dashboard.css
git commit -m "feat(web): render Quests panel from Char.Quests + click-to-focus"
```

---

## Task 9: Destination marker + next-step arrow on the minimap

**Files:**
- Modify: `_datafiles/html/public/static/js/gmcp.js` — `RoomGridSVG`: `setQuestMarker`, marker draw in `_buildRoomTokenInGroup` (~after line 689), next-step arrow, legend rows, `LEATHER` palette (~line 1054)
- Modify: `_datafiles/html/public/webclient-pure.html` — call `gr.setQuestMarker(...)` from the `Char.Quests` handler

- [ ] **Step 1: Add a quest colour to the palette**

In `gmcp.js`, in the `RoomGridSVG.LEATHER` object (~line 1054), add:

```js
    questGold: "#e8b84a",
    questPatina: "#6bb0a0",
```

- [ ] **Step 2: Add `setQuestMarker` + arrowhead marker def**

In `RoomGridSVG`, add a method that stores the focused quest's marker data and re-renders. Store it beside the existing `_party` state; re-use the existing full re-render path (the same one `setZoneSnapshot` triggers) so target/arrow redraw on every update:

```js
setQuestMarker(marker) {
    // marker: {targetRoom, nextRoom, nextDir, name} or null
    this._questMarker = marker || null;
    this._renderAll();  // use the existing method setZoneSnapshot calls to redraw the grid + edges
}
```

Replace `this._renderAll()` with whatever the file's existing "redraw every room group + edges" entry point is (the one `setZoneSnapshot` pass 1/2 performs — verify the method name while editing; if it is inline in `setZoneSnapshot`, extract those two passes into a `_renderAll()` method and call it from both `setZoneSnapshot` and `setQuestMarker`).

Ensure the SVG `<defs>` (where the map's other markers/gradients live) includes an arrowhead + brass gradient once:

```js
// in the defs-building code, add once:
// <radialGradient id="questBrass" cx="34%" cy="26%">
//   <stop offset="0" stop-color="#f4dd92"/><stop offset="46%" stop-color="#cb9f42"/><stop offset="100%" stop-color="#8a6620"/>
// </radialGradient>
// <marker id="questHead" markerWidth="7" markerHeight="7" refX="5.2" refY="3" orient="auto" markerUnits="strokeWidth">
//   <path d="M0,0.4 L6,3 L0,5.6 L1.7,3 Z" fill="#e8b84a"/>
// </marker>
```

Add these with the same `lEl("radialGradient"...)`/`lEl("marker"...)` helper the file already uses for its gradient/marker defs.

- [ ] **Step 3: Draw the destination marker per room**

In `_buildRoomTokenInGroup(g, room, isCurrent, svc)`, right after the party-marker block (~line 689), add:

```js
// Quest destination marker (brass ring + copper-patina glow), distinct from party figure
const qm = this._questMarker;
if (qm && qm.targetRoom && (qm.targetRoom === id || qm.targetRoom === String(id))) {
    g.appendChild(lEl("circle", { cx: cx, cy: cy, r: 12, fill: L.questPatina, opacity: "0.30" }));
    g.appendChild(lEl("circle", { cx: cx, cy: cy, r: 8.5, fill: L.roomFill, stroke: "url(#questBrass)", "stroke-width": 2.2 }));
    g.appendChild(lEl("circle", { cx: cx, cy: cy, r: 3, fill: L.questPatina }));
    g.appendChild(lEl("path", { d: "M" + cx + "," + (cy - 14) + " L" + cx + "," + (cy - 24), stroke: "#cb9f42", "stroke-width": 1.6 }));
    g.appendChild(lEl("path", { d: "M" + cx + "," + (cy - 24) + " L" + (cx + 8) + "," + (cy - 21) + " L" + cx + "," + (cy - 18) + " Z", fill: "url(#questBrass)" }));
    if (qm.name) {
        g.appendChild(lEl("text", { x: cx, y: cy + 21, "text-anchor": "middle", "font-family": "Georgia,serif", "font-size": "7.5", "font-style": "italic", fill: L.questPatina }, qm.name));
    }
}
```

(`id`, `cx`, `cy`, `L`, `lEl` are already in scope in this method — confirm the local names while editing; the party block just above uses exactly these.)

- [ ] **Step 4: Draw the next-step arrow (once, after all room groups)**

In `_renderAll()` (the edges/second pass), after edges are drawn, add an arrow from the current room toward `nextRoom`. If the next room is on the map, point at its node; else point a short stub out the `nextDir` compass direction from the current node:

```js
_drawQuestArrow() {
    const qm = this._questMarker;
    if (!qm || !this.currentCenterId || !qm.nextRoom) { return; }
    const from = this._roomPos(this.currentCenterId);   // {x,y} of a room node — use the file's existing node-position lookup
    if (!from) { return; }
    let to = this._roomPos(qm.nextRoom);
    if (!to && qm.nextDir) {
        to = this._dirOffset(from, qm.nextDir, 34);     // short stub in the compass direction
    }
    if (!to) { return; }
    // shorten both ends so the arrow sits between node edges
    const dx = to.x - from.x, dy = to.y - from.y, len = Math.hypot(dx, dy) || 1;
    const ux = dx / len, uy = dy / len;
    const x1 = from.x + ux * 9, y1 = from.y + uy * 9;
    const x2 = to.x - ux * 6, y2 = to.y - uy * 6;
    this.worldSvg.appendChild(lEl("line", {
        x1: x1, y1: y1, x2: x2, y2: y2,
        stroke: this.LEATHER ? this.LEATHER.questGold : "#e8b84a",
        "stroke-width": 2.6, "marker-end": "url(#questHead)",
    }));
}
```

`_roomPos(id)` and `_dirOffset(pos, dir, dist)` — reuse the file's existing node-coordinate math (the same math `centerOnRoom` and the edge-drawing pass use to place nodes/connectors). If a compass-delta helper doesn't exist, add a small one mapping `north/south/east/west/up/down` to `{dx,dy}` scaled by `dist` (up/down → no planar offset → return null so no stub is drawn). Call `this._drawQuestArrow()` at the end of `_renderAll()`.

- [ ] **Step 5: Add the two legend rows**

In the legend-drawing code (search for the existing "Party member" / "Road / trail" legend text), add two rows mirroring the mock (`docs/superpowers/specs/.../full-map-v2.html` lines 80-83):

```js
// Quest destination: small brass ring + patina core, label "Quest destination"
// Next step: short gold line with url(#questHead), label "Next step"
```

Draw them with the same `lEl` calls the surrounding legend rows use, extending the legend card height to fit.

- [ ] **Step 6: Wire the client handler to feed the marker**

In `webclient-pure.html`, extend the `Char.Quests` handler so it also updates the map marker (find the focused quest and hand its target/next-step to the renderer):

```js
    "Char.Quests": function() {
        renderQuests();
        if (gr && gr.setQuestMarker) {
            var data = (GMCPStructs.Char && GMCPStructs.Char.Quests) ? GMCPStructs.Char.Quests : [];
            var f = Array.isArray(data) ? data.filter(function (q) { return q.focused; })[0] : null;
            if (f && f.target_room) {
                gr.setQuestMarker({ targetRoom: f.target_room, nextRoom: f.next_room || 0, nextDir: f.next_dir || "", name: f.name || "" });
            } else {
                gr.setQuestMarker(null);
            }
        }
    },
```

- [ ] **Step 7: Verify in-game (browser)**

Boot, log in a newbie with a focused trail quest whose current step resolves a target (Task 10 authors one). Expected: the target room shows the brass-ring + patina destination marker with the quest name; a gold arrow points from the current room toward the next hop; when the next room is still fog, the arrow points out the correct compass exit. Walk one room and confirm the arrow advances. Legend shows the two new rows. Colours read against the leather theme. No console errors.

- [ ] **Step 8: Commit**

```bash
git add _datafiles/html/public/static/js/gmcp.js _datafiles/html/public/webclient-pure.html
git commit -m "feat(web): quest destination marker + next-step arrow on the minimap"
```

---

## Task 10: Author a `map_target` + adversarial in-game verification (content SOP)

**Files:**
- Modify: one newbie quest YAML — e.g. `_datafiles/world/dogmud/quests/32-first_blood.yaml` (add `map_target` to the `start` step pointing at the Drill Yard room)

This task exercises the feature end-to-end and satisfies the standing content SOP: **content changes get an adversarial playtest-harness review before user handoff.**

- [ ] **Step 1: Find the Drill Yard room id**

Determine the room id of the Drill Yard (quest 32's arena — the room containing training dummy mob 9109). Grep the room YAML:

```bash
grep -rl "9109" _datafiles/world/dogmud/rooms/ | head
```

Note the room id from the matching file's `roomid:` field.

- [ ] **Step 2: Add `map_target` to the start step**

In `32-first_blood.yaml`, add to the `start` step (the combat step has no `room_enter` trigger, so it needs an explicit target):

```yaml
steps:
  - id: start
    map_target: <drill-yard-room-id>
    description: "Take your cudgel to the training dummy in the Drill
      Yard and learn by doing. Drillmaster Vorn keeps the yard."
    hint: "Type wield cudgel, then attack dummy to start your first
      fight."
```

- [ ] **Step 3: Boot-test (data load)**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . 2>&1 | head -60
```

Expected: `questengine` loads without panic (the new `map_target` key parses). Leave the server running for Step 4.

- [ ] **Step 4: Adversarial playtest-harness run**

With the server up, run the content SOP review via the playtest harness (`/playtest local feature-tester`) driving the newbie flow: reach the trail quests, focus quest 32 in the web Quests panel, and verify against these acceptance checks. Because the harness strips ANSI/omits the SVG, use it to drive commands + read GMCP, and confirm the visual marker in the **browser** yourself:
  - `Char.Quests` GMCP for the focused quest carries `focused:true`, `hint`, `target_room` = the Drill Yard id, and `next_room`/`next_dir` toward it.
  - Clicking a different quest row sends `Char.Quests.Focus` and the server re-emits `Char.Quests` with the new `focused` quest; `hint` (no arg) switches to it.
  - Browser: destination marker on the Drill Yard node, arrow advances as you walk toward it, fog case points out the right exit.
  - Focusing a quest, then walking, keeps the arrow following the shortest path.

Write the run report to `tools/playtest/reports/`.

- [ ] **Step 5: Fix anything the adversarial run surfaces, then commit**

```bash
git add _datafiles/world/dogmud/quests/32-first_blood.yaml
git commit -m "content(quests): map_target on First Blood start step (drill yard)"
```

- [ ] **Step 6: Hand off to the user**

Report the adversarial-run findings + the commit list. **Do not push** — the batch waits for user playtest + explicit push approval (pre-push SOP: PATCH_NOTES.md, `Logging.LogToFile:false`, boot test).

---

## Self-review notes (author)

- **Spec coverage:** §5.1 server → Tasks 1-6; §5.2 web → Tasks 7-9; §3.5 target resolution → Task 2; §4 visuals → Task 9 (mock-faithful); §3.2/3.3 focus reuse → Tasks 5-6 (LastQuestId, no `hint` change); §7 edge cases → covered (no target → 0/no marker; target==current → marker no arrow via `firstHop` guard; fog → `next_dir` stub; cross-zone → deferred, see below); §8 testing → Go tests in Tasks 2-3, in-game in Task 10.
- **Deliberate spec divergences (documented above, not silent):** (1) next-step reuses `mapper.GetPath` instead of a new BFS; (2) `map_target` on `questengine.QuestStep` + GMCP loop migrated to `questengine`; (3) next-hop computed in `gmcp.Char.go` (not `questengine`) to avoid a mapper import in the pure quest package; (4) cross-zone boundary-exit arrow deferred to follow-ups (single-zone newbie hub covers the real need).
- **Type consistency:** payload JSON keys (`id/hint/focused/target_room/next_room/next_dir`) are used identically in the struct (Task 4), the build loop (Task 5), the client render (Task 8), and the marker feed (Task 9). `ResolveQuestTarget(int, string) int` and `NextStep(int,int) (int,string,bool)` signatures match every call site.
- **Follow-ups (out of scope):** per-step `map_target` authoring pass across all newbie quests once proven; cross-zone boundary-exit arrow; whole-route overlay.
