# Automation Panel — Phase 3 Implementation Plan (Triggers)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Add the **Triggers** tab — Tier-1 client-side pattern automation: a wildcard pattern that, when it matches incoming game text, fires command(s), optionally gated by ONE if/else condition (four sources). This is the last automation phase and the complex core.

**Architecture:** Server-stores / client-executes, reusing the Phase-2 substrate. New `user.Triggers` storage (+ a `TriggerCondition` struct, open for Tier-2); triggers added to the `Char.Automation` payload; CRUD via the EXISTING inbound `Char.Automation.Set/Remove` (just add a `kind=="trigger"` branch — the binary `SendGMCP` path + the ws `TelnetIACHandler` are already wired from Phase 2). The **engine** runs in the web client: tap the incoming text stream, strip ANSI, wildcard-match, evaluate the condition against live GMCP state, fire commands. Pool conditions use **available-pool %** (excludes reserved), which needs a small `Char.Vitals` reserved extension.

**Tech Stack:** Go (UserRecord storage, GMCP payload, vitals extension, inbound handler branch), vanilla JS/CSS (the matching engine + builder editor).

**Spec:** `docs/superpowers/specs/2026-06-07-web-client-automation-panel-design.md` (Parts B/C/D/E/F/G — this is Phase 3). Phases 1 (macros/aliases) + 2 (ticks) are merged on master.

**Verified grounding (trust these):**
- `user.Ticks`/`UserTick` + `SetTick`/`RemoveTick` are the pattern to mirror for triggers (`internal/users/userrecord.go`).
- Inbound `Char.Automation.Set`/`Remove` already exist in `gmcp.go` `HandleIAC`, gated `kind=="tick"` — add a `kind=="trigger"` branch. Client `SendGMCP(pkg,obj)` binary helper exists; ws `TelnetIACHandler` is registered.
- `buildAutomationPayload(macros, aliases, ticks)` in `gmcp.Automation.go` — extend with triggers.
- `(c *Character) GetPoolReservation(pool string, poolMax int) int` (`validate.go:228`) returns reserved amount; available = `max - reservation`. Pools read in `gmcp.Char.go:443` from `user.Character.Health`/`HealthMax.Value` etc.
- Client text stream: `socket.onmessage` (`webclient-pure.html:1460`), `term.write(event.data)` (~1575) — tap `event.data` there. GMCP is out-of-band.
- Current target: `GMCPStructs["Char"].Enemies` (`GMCPCharModule_Enemy{id,name,hp,hp_max,engaged}`) — the `engaged:true` one is the target. Conditions list: `GMCPStructs["Char"].Conditions` (`{type, description, duration}`).

**Conventions:** `git add` ONLY named files — never `-A`/`.`. Each task = 1 commit. Trailer EXACTLY: `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`. Prefer **codegraph MCP**. XSS: server strings via `textContent`/`dataset`.

---

### Task 1: `UserTrigger` + `TriggerCondition` storage

**Files:** `internal/users/userrecord.go` + `internal/users/userrecord_test.go`.

- [ ] **Step 1: Failing test**
```go
func TestUserTriggers_SetAssignsIdAndUpdates(t *testing.T) {
	u := &UserRecord{}
	a := u.SetTrigger(UserTrigger{Name: "heal", Pattern: "*bleeding*", ThenCmds: "cast heal", Enabled: true})
	if a.Id == 0 || len(u.Triggers) != 1 { t.Fatalf("create failed: %+v", u.Triggers) }
	a.Name = "heal2"; u.SetTrigger(a)
	if len(u.Triggers) != 1 || u.Triggers[0].Name != "heal2" { t.Fatalf("update dup: %+v", u.Triggers) }
	u.RemoveTrigger(a.Id)
	for _, tr := range u.Triggers { if tr.Id == a.Id { t.Fatalf("remove failed") } }
}
```
Run `go test ./internal/users/ -run TestUserTriggers` → FAIL.

- [ ] **Step 2: Add types + field + helpers** (near `Ticks`):
```go
	Triggers       []UserTrigger         `yaml:"triggers,omitempty"`  // client-run text-pattern automation (web automation panel)
```
```go
// TriggerCondition is the optional single if/else gate on a UserTrigger.
// Open for Tier-2 (regex, multiple conditions, action types) without breaking v1.
// SourceKind: "pool"|"status"|"capture"|"target". SourceKey: pool->"hp"/"sp"/"cp",
// capture->"$1"/"$2", else "". Op: below|above|equals|contains|include|exclude|oneof|notoneof.
// Values: single for pool/capture/status; list (OR) for target oneof/notoneof.
type TriggerCondition struct {
	SourceKind string   `yaml:"sourcekind" json:"sourceKind"`
	SourceKey  string   `yaml:"sourcekey,omitempty" json:"sourceKey,omitempty"`
	Op         string   `yaml:"op" json:"op"`
	Values     []string `yaml:"values" json:"values"`
}

// UserTrigger is a wildcard text pattern that fires commands (optionally gated by
// one condition) when matched against incoming game text. Executed client-side.
type UserTrigger struct {
	Id        int               `yaml:"id" json:"id"`
	Name      string            `yaml:"name" json:"name"`
	Pattern   string            `yaml:"pattern" json:"pattern"`   // "*" wildcards -> $1..$n
	Condition *TriggerCondition `yaml:"condition,omitempty" json:"condition,omitempty"`
	ThenCmds  string            `yaml:"thencmds" json:"thenCmds"` // ";"-separated
	ElseCmds  string            `yaml:"elsecmds,omitempty" json:"elseCmds,omitempty"`
	Enabled   bool              `yaml:"enabled" json:"enabled"`
}

func (u *UserRecord) SetTrigger(t UserTrigger) UserTrigger {
	if t.Id == 0 {
		maxId := 0
		for _, ex := range u.Triggers { if ex.Id > maxId { maxId = ex.Id } }
		t.Id = maxId + 1
		u.Triggers = append(u.Triggers, t)
		return t
	}
	for i := range u.Triggers { if u.Triggers[i].Id == t.Id { u.Triggers[i] = t; return t } }
	u.Triggers = append(u.Triggers, t)
	return t
}

func (u *UserRecord) RemoveTrigger(id int) {
	for i := range u.Triggers {
		if u.Triggers[i].Id == id { u.Triggers = append(u.Triggers[:i], u.Triggers[i+1:]...); return }
	}
}
```
- [ ] **Step 3:** `go test ./internal/users/ -run TestUserTriggers` PASS; `go build ./internal/users/...` clean.
- [ ] **Step 4: Commit** (userrecord.go + userrecord_test.go). Subject: `feat(users): UserTrigger + TriggerCondition storage`

---

### Task 2: Triggers in the Char.Automation payload

**Files:** `modules/gmcp/gmcp.Automation.go` + `gmcp.Automation_test.go`.

- [ ] **Step 1: Test** (mirror the ticks test):
```go
func TestBuildAutomationPayload_Triggers(t *testing.T) {
	trigs := []users.UserTrigger{{Id: 1, Name: "heal", Pattern: "*bleeding*", ThenCmds: "cast heal", Enabled: true,
		Condition: &users.TriggerCondition{SourceKind: "pool", SourceKey: "hp", Op: "below", Values: []string{"30"}}}}
	p := buildAutomationPayload(nil, nil, nil, trigs)
	if len(p.Triggers) != 1 || p.Triggers[0].Pattern != "*bleeding*" || p.Triggers[0].Condition == nil ||
		p.Triggers[0].Condition.SourceKey != "hp" { t.Fatalf("trigger not mapped: %+v", p.Triggers) }
}
```
Update existing builder-test calls to the new 4-arg signature (pass `nil` for triggers).
Run → FAIL.

- [ ] **Step 2: Implement.** Add payload structs mirroring the storage (json tags `sourceKind/sourceKey/op/values` and `id/name/pattern/condition/thenCmds/elseCmds/enabled`):
```go
type GMCPAutomation_Condition struct {
	SourceKind string   `json:"sourceKind"`
	SourceKey  string   `json:"sourceKey,omitempty"`
	Op         string   `json:"op"`
	Values     []string `json:"values"`
}
type GMCPAutomation_Trigger struct {
	Id        int                       `json:"id"`
	Name      string                    `json:"name"`
	Pattern   string                    `json:"pattern"`
	Condition *GMCPAutomation_Condition `json:"condition,omitempty"`
	ThenCmds  string                    `json:"thenCmds"`
	ElseCmds  string                    `json:"elseCmds,omitempty"`
	Enabled   bool                      `json:"enabled"`
}
```
Add `Triggers []GMCPAutomation_Trigger \`json:"triggers"\`` to the payload. Change the builder to `buildAutomationPayload(macros, aliases map[string]string, ticks []users.UserTick, triggers []users.UserTrigger)`, map each trigger (copy the Condition pointer into a `*GMCPAutomation_Condition` when non-nil), preserve order. Update `sendAutomation` to pass `user.Triggers`.
- [ ] **Step 3:** `go test ./modules/gmcp/ -run TestBuildAutomationPayload` (all cases) PASS; `go build ./modules/gmcp/...` clean.
- [ ] **Step 4: Commit** (the two gmcp files). Subject: `feat(gmcp): include triggers in the Char.Automation payload`

---

### Task 3: Inbound trigger CRUD + vitals reserved extension

**Files:** `modules/gmcp/gmcp.go`, `modules/gmcp/gmcp.Char.go`.

- [ ] **Step 1: Add `kind=="trigger"` to the inbound handlers.** In `gmcp.go` `HandleIAC`, extend the existing `Char.Automation.Set`/`Remove` cases. The Set case's decode struct currently embeds `users.UserTick`; restructure so it can decode EITHER kind — decode into a struct with `Kind`, the tick fields, AND the trigger fields, then branch:
```go
		case `Char.Automation.Set`:
			var decoded struct {
				Kind    string             `json:"kind"`
				users.UserTick                              // tick fields (promoted)
				Trigger users.UserTrigger  `json:"-"`       // filled below for kind==trigger
			}
			if uid := userIdForConnection(connectionId); uid > 0 {
				if u := users.GetByUserId(uid); u != nil {
					switch {
					case json.Unmarshal(payload, &decoded) != nil:
						// ignore malformed
					case decoded.Kind == `tick`:
						u.SetTick(decoded.UserTick)
						events.AddToQueue(events.AutomationChanged{UserId: uid})
					case decoded.Kind == `trigger`:
						var tr users.UserTrigger
						if json.Unmarshal(payload, &tr) == nil {
							u.SetTrigger(tr)
							events.AddToQueue(events.AutomationChanged{UserId: uid})
						}
					}
				}
			}
```
(Simpler alternative if the embedded-tick + separate-trigger unmarshal is awkward: keep the tick case as-is and add a SECOND `json.Unmarshal(payload, &trig)` only when `kind=="trigger"`. The implementer may choose whichever is cleaner — the REQUIREMENT is: `kind=="tick"` → SetTick, `kind=="trigger"` → SetTrigger, both emit AutomationChanged.) The `Remove` case: add `decoded.Kind == "trigger"` → `u.RemoveTrigger(decoded.Id)`.

- [ ] **Step 2: Vitals reserved extension.** In `gmcp.Char.go`, add to `GMCPCharModule_Payload_Vitals`:
```go
	HpReserved         int `json:"hp_reserved,omitempty"`
	StaminaReserved    int `json:"stamina_reserved,omitempty"`
	ConvictionReserved int `json:"conviction_reserved,omitempty"`
```
and populate them in the vitals builder (~line 443) via `GetPoolReservation`:
```go
			HpReserved:         user.Character.GetPoolReservation("health", user.Character.HealthMax.Value),
			StaminaReserved:    user.Character.GetPoolReservation("stamina", user.Character.StaminaMax.Value),
			ConvictionReserved: user.Character.GetPoolReservation("conviction", user.Character.ConvictionMax.Value),
```
(Confirm the pool-name strings `"health"/"stamina"/"conviction"` match what `GetPoolReservation`/`ReservePool` expect — check `enchantments`/`validate.go`. The client computes available% = `current / max(1, max - reserved)`.)

- [ ] **Step 3:** `go build ./...` clean.
- [ ] **Step 4: Commit** (gmcp.go + gmcp.Char.go). Subject: `feat(gmcp): inbound trigger CRUD + Char.Vitals reserved fields`

---

### Task 4: Client trigger engine (stream tap + match + condition + fire)

**Files:** `webclient-pure.html` (script).

- [ ] **Step 1: Helpers.** Add:
```js
        // Strip ANSI/escape sequences for trigger matching.
        function stripAnsi(s) { return String(s).replace(/\x1b\[[0-9;?]*[A-Za-z]/g, "").replace(/\x1b\][^\x07]*\x07/g, ""); }
        // Compile a "*"-wildcard pattern to a regex with capture groups.
        function compileTriggerPattern(pat) {
            var esc = String(pat).replace(/[.*+?^${}()|[\]\\]/g, function(ch){ return ch === "*" ? " " : "\\" + ch; });
            esc = esc.replace(/ /g, "(.*?)");
            try { return new RegExp(esc); } catch (e) { return null; }
        }
```
- [ ] **Step 2: Condition evaluation** against live GMCP:
```js
        function availablePct(cur, mx, reserved) {
            var avail = Math.max(1, (mx || 0) - (reserved || 0));
            return (Number(cur) || 0) / avail * 100;
        }
        function evalTriggerCondition(cond, caps) {
            if (!cond) return true;
            var a = (GMCPStructs["Char"] || {});
            var v = (cond.values && cond.values[0]) || "";
            if (cond.sourceKind === "pool") {
                var vit = a.Vitals || {};
                var pct = cond.sourceKey === "hp" ? availablePct(vit.hp, vit.hp_max, vit.hp_reserved)
                        : cond.sourceKey === "sp" ? availablePct(vit.stamina, vit.stamina_max, vit.stamina_reserved)
                        : availablePct(vit.conviction, vit.conviction_max, vit.conviction_reserved);
                var n = parseFloat(v) || 0;
                return cond.op === "below" ? pct < n : cond.op === "above" ? pct > n : Math.round(pct) === Math.round(n);
            }
            if (cond.sourceKind === "status") {
                var has = (a.Conditions || []).some(function(c){ return c && String(c.type).toLowerCase() === v.toLowerCase(); });
                return cond.op === "exclude" ? !has : has;
            }
            if (cond.sourceKind === "capture") {
                var idx = parseInt(String(cond.sourceKey).replace("$",""), 10) || 0;
                var capv = caps[idx] || "";
                return cond.op === "contains" ? capv.toLowerCase().indexOf(v.toLowerCase()) >= 0 : capv.toLowerCase() === v.toLowerCase();
            }
            if (cond.sourceKind === "target") {
                var eng = (a.Enemies || []).filter(function(e){ return e && e.engaged; })[0];
                var name = eng ? String(eng.name).toLowerCase() : "";
                var list = (cond.values || []).map(function(s){ return String(s).toLowerCase().trim(); });
                var inList = !!name && list.indexOf(name) >= 0;
                return cond.op === "notoneof" ? !inList : inList;
            }
            return true;
        }
```
- [ ] **Step 3: The matcher + firing** with a runaway cooldown:
```js
        var triggerLastFire = {};   // id -> last fire ms (cooldown)
        var TRIGGER_MIN_INTERVAL_MS = 1000; // per-trigger floor (runaway guard)
        function fireTriggerCommands(cmds, caps) {
            String(cmds || "").split(";").forEach(function(c){
                c = c.trim();
                if (!c) return;
                c = c.replace(/\$([0-9])/g, function(_, d){ return caps[parseInt(d,10)] || ""; });
                SendData(c);
            });
        }
        function runTriggers(line, nowMs) {
            var a = GMCPStructs["Char"] && GMCPStructs["Char"].Automation;
            if (!a || !a.triggers) return;
            a.triggers.forEach(function(t){
                if (!t.enabled || !t.pattern) return;
                var re = compileTriggerPattern(t.pattern);
                if (!re) return;
                var m = re.exec(line);
                if (!m) return;
                if (triggerLastFire[t.id] && (nowMs - triggerLastFire[t.id]) < TRIGGER_MIN_INTERVAL_MS) return; // cooldown
                triggerLastFire[t.id] = nowMs;
                var ok = evalTriggerCondition(t.condition, m); // m[1..n] are captures
                fireTriggerCommands(ok ? t.thenCmds : t.elseCmds, m);
            });
        }
```
(Note `Date.now()` IS available in the browser client — only workflow scripts lack it.)
- [ ] **Step 4: Tap the stream.** In `socket.onmessage`, where text is written (`term.write(event.data)` ~line 1575) — only for the regular game-text path (not GMCP/control frames already handled earlier in onmessage) — add, before/after the write:
```js
                stripAnsi(event.data).split(/\r?\n/).forEach(function(line){ if (line) runTriggers(line, Date.now()); });
```
- [ ] **Step 5: Verify** — grep `runTriggers`, `evalTriggerCondition`, `compileTriggerPattern`. Commit (ONLY webclient-pure.html). Subject: `feat(web): client trigger engine (stream tap + wildcard match + condition)`

---

### Task 5: Triggers tab render + test-fire

**Files:** `webclient-pure.html`.

- [ ] **Step 1:** In `renderAutomation()`, special-case `autoActiveTab === 'triggers'` (remove its "Coming soon"): for each `a.triggers[]` build an `.auto-row` with a `.auto-tog`(`.on` when enabled), `.name`=name, `.sum`=pattern (+ " (off)" when disabled), `dataset.autoType="tick"`→ use `"trigger"`, `dataset.autoId`, `dataset.autoKey=String(id)`. Empty → "No triggers configured."
- [ ] **Step 2:** Extend the `#auto-list` click handler: toggle (`.auto-tog`) on a trigger row → `SendGMCP("Char.Automation.Set", {kind:"trigger", ...current, enabled:!current.enabled})` (read current trigger from `GMCPStructs.Char.Automation.triggers` by id), stopPropagation. Row-body click on a trigger → **test-fire**: evaluate its condition against current live state with empty captures and fire then/else (call a small helper that mirrors the engine's fire path with `caps=[line]`/empty). 
- [ ] **Step 3:** Verify — grep `autoType==="trigger"`/`"trigger"` in render + click. Commit (ONLY webclient-pure.html). Subject: `feat(web): triggers tab — rows, toggle, test-fire`

---

### Task 6: Trigger editor (progressive builder)

**Files:** `webclient-pure.html`.

- [ ] **Step 1: Build the editor** into the modal (mirror the tick editor, richer): Name; "When I see" pattern input (hint: `*` wildcards → `$1 $2`); a **"Do"** section that starts as a single ThenCmds textarea, with a **"+ Add ▾"** control that inserts an **if/else clause** — a source `<select>` (my HP % / my SP % / my CP % / my conditions / capture $1 / capture $2 / my target), an operator `<select>` whose options swap by source kind (pool: below/above/equals; status: include/exclude; capture: equals/contains; target: is one of/is not one of), a value input (number for pool; text for capture; status name for status; comma-list for target), and the Then + Else command textareas. The clause has a ✕ to remove it (back to plain ThenCmds). Enabled toggle.
- [ ] **Step 2: Build the object + Save.** On Save assemble `{kind:"trigger", id: editingId||0, name, pattern, thenCmds, elseCmds, enabled, condition: <null or {sourceKind, sourceKey, op, values}>}` and `SendGMCP("Char.Automation.Set", obj)`. Map the UI: pool source → sourceKind "pool", sourceKey "hp"/"sp"/"cp"; conditions → "status" (values=[name]); capture → "capture", sourceKey "$1"/"$2"; target → "target", values=split-comma list. **Remove** (context menu) → `SendGMCP("Char.Automation.Remove",{kind:"trigger",id})`; **Duplicate** → editor pre-filled, id=0. Wire **"+ New"** for the triggers tab + the context-menu Edit/Duplicate/Remove for `dataset.autoType==="trigger"` (read current fields from the GMCP triggers by id).
- [ ] **Step 3:** Verify — grep `"Char.Automation.Set"` with a trigger object + the source/op selects. Commit (ONLY webclient-pure.html). Subject: `feat(web): trigger editor — progressive if/else builder`

---

### Task 7: Help topic + context docs

**Files:** Create `_datafiles/world/dogmud/templates/help/triggers.template`; modify `modules/gmcp/context.md`.

- [ ] **Step 1: `help triggers` topic** (mirror `macros.template` house style). Cover: what a trigger is (when matching text appears, run command(s)); `*` wildcards → `$1 $2`; the optional if/else with the four sources; managed in the **Triggers & Timers** panel (no command); runs only in the web client; the per-trigger cooldown. **MUST explicitly state the available-pool-% rule** (user requirement): pool conditions are a percentage of your *usable* (unreserved) pool, not the total — with the worked example: "If half your health is reserved, 'HP below 30%' fires at 30% of the remaining half — i.e. when your usable health bar is below 30%, not the whole bar."
- [ ] **Step 2: context docs** — update `modules/gmcp/context.md`: `Char.Automation` payload now includes `triggers` (+ the condition shape); inbound `Char.Automation.Set/Remove` now handle `kind=="trigger"`; `Char.Vitals` gained `hp_reserved`/`stamina_reserved`/`conviction_reserved` (for available-pool-% triggers).
- [ ] **Step 3:** Verify — `ls` the template; grep `context.md` for `triggers`. Commit (the named files). Subject: `docs: help triggers topic + context for triggers & vitals reserved`

---

### Task 8: Boot + browser smoke

**Files:** none.

- [ ] **Step 1:** `go build ./...` clean; `go test ./internal/users/ ./modules/gmcp/` ok; boot (`timeout 90 go run . > /tmp/p3boot.log 2>&1`) → `Server Ready`, no panic (new help template + vitals fields + inbound branch load).
- [ ] **Step 2: Browser smoke** (rebuild+restart server; hard-refresh `/webclient`; log in):
- Triggers tab: + New → make `*you are hungry*` → `eat ration` (no condition). Trigger something that prints "you are hungry" → `eat ration` fires once (cooldown prevents spam).
- Conditional: pattern `*HP*` (or any line you can produce) with **If my HP % below 30 then cast heal else bandage** — verify it branches by your actual usable HP (test with a reserved-pool item if available; otherwise verify the threshold fires relative to usable, not total).
- Wildcard capture: `* tells you '*'` → `reply got it` (and a capture condition `$1 equals <name>`).
- Toggle disables it; left-click test-fires; Edit/Duplicate/Remove; survives relog + refresh. Macros/Aliases/Ticks still work. No console errors; no trigger-flood.
- [ ] **Step 3:** Record results; if clean, ready for finishing-a-development-branch.

---

## Self-Review

**Spec coverage (Phase 3):** `UserTrigger`+`TriggerCondition` storage (Task 1) ✓; payload (Task 2) ✓; inbound trigger CRUD + vitals reserved (Task 3) ✓; client engine — stream tap, wildcard match, 4-source condition eval w/ available-pool-%, cooldown (Task 4) ✓; triggers tab + test-fire (Task 5) ✓; progressive builder editor (Task 6) ✓; help (with explicit available-% wording) + context (Task 7) ✓; smoke (Task 8) ✓. Tier-1 (one pattern, one if/else); `TriggerCondition` struct open for Tier-2.

**Placeholder scan:** concrete code in every code step; the one "implementer's choice" (the inbound decode structure in Task 3 Step 1) states the hard requirement explicitly and offers a concrete simpler alternative — not a gap.

**Identifier consistency:** `UserTrigger`/`TriggerCondition` field names + json tags identical across userrecord.go (Task 1), the gmcp payload (Task 2), the inbound decode (Task 3), and the client editor object + engine reads (Tasks 4/5/6). `buildAutomationPayload(macros, aliases, ticks, triggers)` 4-arg signature consistent test↔impl↔`sendAutomation`. `Char.Automation.Set/Remove` + `kind:"trigger"` consistent server↔client. Condition `sourceKind`/`op` vocab consistent between the editor (Task 6 builds), the engine (Task 4 reads), and the storage doc (Task 1). Vitals `hp_reserved` etc. consistent server (Task 3) ↔ engine `availablePct` (Task 4).
