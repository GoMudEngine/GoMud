# Automation Panel — Phase 4 Implementation Plan (Cooldown-gated action queue)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans. Steps use checkbox (`- [ ]`) syntax.

**Goal:** A global FIFO action queue that drains one ability per shared `special-move` cooldown; triggers opt in (off / queue-at-back / queue-at-front, priority FIFO), with dedup, cap 10, manual Clear + death-clear, and highlight/float UI. Plus rally/warcry expiry echoes.

**Architecture:** Mostly client-side (queue + processor + UI), reusing the shipped trigger engine + the `special-move` cooldown already in `GMCPStructs["Commands"].State.cooldowns`. Small server bits: a `QueueMode` field on `UserTrigger` (storage + GMCP payload; inbound already unmarshals the whole trigger) and `expireMessage` on two buffs. Parked locally.

**Spec:** `docs/superpowers/specs/2026-06-08-automation-panel-phase4-queue-design.md`.

**Verified grounding:**
- `users.UserTrigger` + `GMCPAutomation_Trigger` + `buildAutomationPayload(...)` are the Phase-3 structures to extend. Inbound `Char.Automation.Set` unmarshals the full `UserTrigger`, so a new field round-trips with NO `gmcp.go` change.
- Shared cooldown: `GMCPStructs["Commands"].State.cooldowns["special-move"]` — **ready ⟺ absent**.
- **Death signal:** `GMCPStructs["Commands"].State.mode === "downed"` (`gmcp.Commands.go:173`, set when `Health <= 0`).
- **Buff expiry:** buffs have an `expireMessage:` field (e.g. `85-infraredvision.yaml`). **Warcry = `_datafiles/world/dogmud/buffs/79-warcry.yaml`, Rally = `80-rally.yaml`** — neither has `expireMessage` yet.
- Client trigger engine: `runTriggers(line, nowMs)`, `evalTriggerCondition`, `fireTriggerCommands(cmds, caps)`; `renderAutomation()` renders the triggers tab; the `Char.Automation` + `Char`/`Commands` GMCP update handlers.

**Conventions:** `git add` ONLY named files — never `-A`/`.`. Each task = 1 commit. Trailer EXACTLY: `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`. Prefer **codegraph MCP**. XSS: server strings via `textContent`/`dataset`.

---

### Task 1: `QueueMode` on UserTrigger + GMCP payload (server, TDD)

**Files:** `internal/users/userrecord.go`, `internal/users/userrecord_test.go`, `modules/gmcp/gmcp.Automation.go`, `modules/gmcp/gmcp.Automation_test.go`.

- [ ] **Step 1: Failing tests.** In `userrecord_test.go` add a case asserting `QueueMode` round-trips through `SetTrigger`:
```go
func TestUserTriggers_QueueMode(t *testing.T) {
	u := &UserRecord{}
	a := u.SetTrigger(UserTrigger{Name: "rally", Pattern: "*rally fades*", ThenCmds: "rally", QueueMode: "back", Enabled: true})
	if u.Triggers[0].QueueMode != "back" { t.Fatalf("queueMode not stored: %+v", u.Triggers[0]) }
	_ = a
}
```
In `gmcp.Automation_test.go` extend `TestBuildAutomationPayload_Triggers` (or add one) to assert the payload carries `QueueMode`.
Run both → FAIL.

- [ ] **Step 2: Add the field.** In `userrecord.go` add to `UserTrigger`:
```go
	QueueMode string            `yaml:"queuemode,omitempty" json:"queueMode,omitempty"` // ""=fire now, "back", "front"
```
In `gmcp.Automation.go` add to `GMCPAutomation_Trigger`:
```go
	QueueMode string `json:"queueMode,omitempty"`
```
and map it where `buildAutomationPayload` builds each trigger (`QueueMode: tr.QueueMode`).

- [ ] **Step 3:** `go test ./internal/users/ ./modules/gmcp/ -run 'TestUserTriggers|TestBuildAutomationPayload'` PASS; `go build ./...` clean.
- [ ] **Step 4: Commit** (the 4 files). Subject: `feat(automation): QueueMode field on triggers (storage + GMCP)`

---

### Task 2: Rally & warcry expiry echoes (content)

**Files:** `_datafiles/world/dogmud/buffs/79-warcry.yaml`, `_datafiles/world/dogmud/buffs/80-rally.yaml`.

- [ ] **Step 1: Confirm the mechanism.** Read a buff that sets a non-empty `expireMessage` (grep buffs for `expireMessage:` with text) to confirm the exact field name + that it's emitted to the player on natural expiry. (If the emitting code lives in `internal/buffs`, skim it via codegraph to confirm `expireMessage` is the field used.)

- [ ] **Step 2: Add the echoes.** Add an `expireMessage:` line to each buff, first-person, no hard numbers, matching the house voice. Read each file first; then add e.g.:
  - `79-warcry.yaml`: `expireMessage: "Your warcry's fervor fades."`
  - `80-rally.yaml`: `expireMessage: "The strength of your rally drains away."`
  (Read `80-rally.yaml` to match its naming/tone; adjust wording to fit.)

- [ ] **Step 3: Verify** — grep both files for `expireMessage:` (non-empty). (The boot test in Task 8 confirms the buffs still load.)
- [ ] **Step 4: Commit** (the 2 buff files). Subject: `feat(buffs): rally + warcry emit an expiry echo (trigger-matchable)`

---

### Task 3: Trigger editor — "Queue this action" control

**Files:** `_datafiles/html/public/webclient-pure.html` (the trigger editor, `openTriggerEditor`).

- [ ] **Step 1:** Add a labeled `<select>` "Queue this action" with options: `Fire immediately` (value `""`), `Queue at back` (`back`), `Queue at front` (`front`). Default `""`. Place it near the Enabled toggle.
- [ ] **Step 2:** On **Save**, include `queueMode: queueSelect.value` in the trigger object sent via `SendGMCP("Char.Automation.Set", {kind:"trigger", ..., queueMode})`. On **pre-fill** (edit/duplicate), set the select to the trigger's `queueMode || ""`.
- [ ] **Step 3:** Verify — grep for `queueMode` in the editor build + save. Commit (ONLY webclient-pure.html). Subject: `feat(web): trigger editor — Queue this action (off/back/front)`

---

### Task 4: Action queue state + enqueue

**Files:** `_datafiles/html/public/webclient-pure.html` (script).

- [ ] **Step 1: Queue state + enqueue helper.** Add module-level `var actionQueue = [];` and:
```js
        var QUEUE_CAP = 10;
        // entry: { triggerId, name, commands }  (commands already $-substituted)
        function enqueueTriggerAction(trigger, resolvedCmds) {
            // dedup: one entry per trigger
            for (var i = 0; i < actionQueue.length; i++) {
                if (actionQueue[i].triggerId === trigger.id) return;
            }
            if (actionQueue.length >= QUEUE_CAP) return; // cap (quiet)
            var entry = { triggerId: trigger.id, name: trigger.name, commands: resolvedCmds };
            if (trigger.queueMode === "front") {
                // priority FIFO: after the last existing front, before the first back
                var insertAt = 0;
                while (insertAt < actionQueue.length && actionQueue[insertAt]._front) insertAt++;
                entry._front = true;
                actionQueue.splice(insertAt, 0, entry);
            } else {
                actionQueue.push(entry); // back
            }
            renderAutomation();
            drainQueue();
        }
```
- [ ] **Step 2: Hook the trigger match path.** In `runTriggers` (where a matched trigger currently calls `fireTriggerCommands(ok ? thenCmds : elseCmds, caps)`), branch: if `t.queueMode` is `"back"` or `"front"`, **resolve captures now** (substitute `$1..$9` from `caps` into the chosen command string) and call `enqueueTriggerAction(t, resolved)` INSTEAD of firing. Otherwise fire immediately as today. (Resolve at enqueue time because the captures are gone by drain time — reuse the same `$`-substitution `fireTriggerCommands` uses; factor it into a small `substituteCaptures(cmds, caps)` helper if needed.)
- [ ] **Step 3:** Verify — grep `enqueueTriggerAction` + the `queueMode` branch in `runTriggers`. Commit (ONLY webclient-pure.html). Subject: `feat(web): action queue state + enqueue (dedup, cap, priority FIFO)`

---

### Task 5: Queue processor (cooldown-gated drain)

**Files:** `_datafiles/html/public/webclient-pure.html` (script).

- [ ] **Step 1: Drain logic.**
```js
        var QUEUE_MIN_FIRE_GAP_MS = 800; // ensure the cooldown registers between fires
        var lastQueueFireAt = 0;
        function cooldownReady() {
            var cds = (((GMCPStructs["Commands"] || {}).State) || {}).cooldowns || {};
            return !cds["special-move"];
        }
        function drainQueue() {
            if (!actionQueue.length) return;
            if (!cooldownReady()) return;
            if (Date.now() - lastQueueFireAt < QUEUE_MIN_FIRE_GAP_MS) return; // let prior fire register
            var entry = actionQueue.shift();
            lastQueueFireAt = Date.now();
            String(entry.commands || "").split(";").forEach(function(c){ c = c.trim(); if (c) SendData(c); });
            renderAutomation();
        }
```
- [ ] **Step 2: Run the processor on the right signals.** Call `drainQueue()` from: (a) the `Commands.State` GMCP update handler (the cooldown lives there — add `drainQueue()` where `Commands.State` is dispatched, mirroring how renderers hook GMCP updates; if there's no `Commands` handler yet, add one that calls `drainQueue()`), (b) the `Char` update path (defensive), and (c) a low-frequency safety `setInterval(drainQueue, 500)` so a stalled queue still drains when the cooldown frees even without a GMCP nudge.
- [ ] **Step 3:** Verify — grep `drainQueue` (defined + called from the Commands/GMCP path + the interval). Commit (ONLY webclient-pure.html). Subject: `feat(web): queue processor — one ability per special-move cooldown`

---

### Task 6: Clear + death-clear

**Files:** `_datafiles/html/public/webclient-pure.html` (script).

- [ ] **Step 1: Clear helper + button hook.** `function clearActionQueue(){ actionQueue = []; renderAutomation(); }`. Wire it to the status-bar Clear button (Task 7).
- [ ] **Step 2: Death-clear.** Track the last seen mode; when `GMCPStructs["Commands"].State.mode` transitions to `"downed"`, call `clearActionQueue()`. Add this check in the `Commands.State` update path (alongside `drainQueue`):
```js
        var _lastMode = "";
        function checkDeathClearsQueue() {
            var mode = (((GMCPStructs["Commands"] || {}).State) || {}).mode || "";
            if (mode === "downed" && _lastMode !== "downed") clearActionQueue();
            _lastMode = mode;
        }
```
called wherever `Commands.State` updates. (Reload/reconnect clears naturally — `actionQueue` resets.)
- [ ] **Step 3:** Verify — grep `clearActionQueue` + `checkDeathClearsQueue`. Commit (ONLY webclient-pure.html). Subject: `feat(web): clear queue on Clear button + on death (mode downed)`

---

### Task 7: Queue UI (highlight/float/badge + status bar)

**Files:** `_datafiles/html/public/webclient-pure.html` (renderAutomation triggers branch), `_datafiles/html/public/static/css/dashboard.css`.

- [ ] **Step 1: Render queued triggers first, highlighted + badged.** In `renderAutomation()`'s triggers branch: build a set of queued `triggerId`s (in queue order). Render the queued triggers FIRST as `.auto-row.pending` with a FIFO badge (`.badge`, position 1 = `.badge.next` brass, rest red) — in `actionQueue` order — then a divider, then the remaining (non-queued) triggers as normal `.auto-row`s. Use `textContent` for names.
- [ ] **Step 2: Status bar.** Above the trigger list (only when `actionQueue.length`), render a bar: `▸ N queued`, the cooldown state (`cooldownReady() ? "ready" : "cooling…"`), and a **Clear** button wired to `clearActionQueue()`.
- [ ] **Step 3: CSS.** Append `.auto-row.pending` (red glow), `.badge`/`.badge.next`, and the `.auto-qbar`/`.auto-qclear` styles to `dashboard.css`, matching the mock (`queue-panel.html`) + the leather tokens.
- [ ] **Step 4:** Verify — grep `auto-row pending`/`pending` + `auto-qbar` in html/css. Commit (the two files). Subject: `feat(web): queue UI — highlight/float queued triggers + status bar`

---

### Task 8: Boot + browser smoke

**Files:** none.

- [ ] **Step 1:** `go build ./...` clean; `go test ./internal/users/ ./modules/gmcp/` ok; boot (`timeout 90 go run . > /tmp/p4boot.log 2>&1`) → `Server Ready`, no panic (warcry/rally buffs still load with `expireMessage`).
- [ ] **Step 2: Browser smoke** (rebuild+restart; hard-refresh `/webclient`; log in):
  - A trigger set **Queue at back** enqueues on match (doesn't fire immediately); appears highlighted + floated to the top with badge.
  - Queue 3 abilities (e.g. via 3 triggers) → they drain **one per `special-move` cooldown** in order; badges renumber; entries drop back to normal as they fire. No all-at-once dump, no double-fire.
  - **Queue at front** jumps ahead of back entries (priority FIFO; two fronts keep fire-order).
  - Dedup: a re-firing queued trigger doesn't double-add. Cap: 11th rejected.
  - **Clear** empties it; **die** → queue clears (mode downed); refresh → clears.
  - Rally/warcry now print an expiry message (a trigger pattern can match it).
  - Macros/aliases/ticks/non-queued triggers unaffected; no console errors.
- [ ] **Step 3:** Record results; if clean → finishing-a-development-branch.

---

## Self-Review

**Spec coverage:** QueueMode storage+GMCP (Task 1) ✓; rally/warcry echoes (Task 2) ✓; editor control (Task 3) ✓; enqueue w/ dedup+cap+priority-FIFO (Task 4) ✓; cooldown-gated one-per-cooldown processor (Task 5) ✓; Clear + death-clear (Task 6) ✓; highlight/float/badge + status bar (Task 7) ✓; smoke (Task 8) ✓. Ephemeral queue (no persistence) ✓; no auto-retry (resolve-captures-at-enqueue, fire-and-forget) ✓.

**Open items from the spec — resolved:** death signal = `Commands.State.mode === "downed"` (Task 6); processor timing = cooldown-ready + `QUEUE_MIN_FIRE_GAP_MS` (Task 5); buff-expiry = the `expireMessage` field on buffs 79/80 (Task 2, with a confirm-the-mechanism step).

**Placeholder scan:** concrete code/paths throughout; the one "confirm the mechanism" step (Task 2 Step 1) and the buff wording reference real files to read first — not gaps.

**Identifier consistency:** `QueueMode`/`queueMode` consistent across userrecord.go (Task 1), gmcp payload (Task 1), the editor object (Task 3), and the engine read `t.queueMode` (Task 4). `actionQueue`/`enqueueTriggerAction`/`drainQueue`/`clearActionQueue`/`cooldownReady` defined once (Tasks 4–6) and referenced consistently (incl. renderAutomation Task 7). `special-move` cooldown key + `Commands.State.cooldowns`/`.mode` consistent with the verified GMCP source.
