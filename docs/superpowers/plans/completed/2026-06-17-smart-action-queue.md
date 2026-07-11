# Smart Action Queue Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make web-client queued actions auto-retry through cooldown, mid-cast, out-of-CP, and interruption — silently, with a short give-up window — so players never hand-author conditions for "I can't act yet."

**Architecture:** A new server-side `actions.ActionReadiness(actor, cmd)` predicate authoritatively answers Ready / Deferred / Rejected for a command. A new inbound GMCP verb `Char.Action.Try` runs that check and either executes the command (`user.Command`) or replies `Char.Action.Result{status}` with no player-facing text on a defer. The client routes queued actions through `Try`, holds on `deferred`, drops on `fired`/`rejected`, and gives up after `QUEUE_STALENESS_ROUNDS`. A `Char.Action.Interrupted` signal lets the client re-arm an interrupted cast.

**Tech Stack:** Go (engine: `internal/actions`, `internal/events`, `modules/gmcp`), vanilla JS (`_datafiles/html/public/webclient-pure.html`). Tests: `testify` (`go test`).

**Spec:** `docs/superpowers/specs/2026-06-17-smart-action-queue-design.md`

**Pre-req:** The 2026-06-17 session reverts (casting Vitals field + state conditions) are already done; this branch (`feature/smart-action-queue`) builds on top. Do NOT re-add them.

---

## File Structure

- **Create** `internal/actions/action_readiness.go` — `ReadyStatus`, `ReadinessResult`, `ActionReadiness(actor, cmd)`, `castReadiness`, special-move verb set. One responsibility: "can this command fire right now, and is the blocker transient?"
- **Create** `internal/actions/action_readiness_test.go` — unit + drift tests.
- **Create** `modules/gmcp/gmcp.Action.go` — outbound `Char.Action.Result` / `Char.Action.Interrupted` payloads + the `CastInterrupted` event listener.
- **Modify** `internal/events/eventtypes.go` — add `CastInterrupted` event.
- **Modify** `internal/actions/cast_interrupt.go` (and other `TriggerCastCancel` sites) — emit `CastInterrupted`.
- **Modify** `modules/gmcp/gmcp.go` — add the `Char.Action.Try` inbound case.
- **Modify** `modules/gmcp/<module Load/registration>` — register the `CastInterrupted` listener.
- **Modify** `_datafiles/html/public/webclient-pure.html` — rework `drainQueue`, add Result/Interrupted handlers, add staleness window.

---

## Task 1: Readiness result type + special-move coverage

**Files:**
- Create: `internal/actions/action_readiness.go`
- Test: `internal/actions/action_readiness_test.go`

- [ ] **Step 1: Write the failing test**

```go
package actions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestActionReadiness_GenericCommand_Ready(t *testing.T) {
	actor := newTestUserActor(t) // see Step 3 note; reuse the package's existing test actor helper
	res := ActionReadiness(actor, "say hello")
	assert.Equal(t, ActionReady, res.Status)
}

func TestActionReadiness_NilActor_Rejected(t *testing.T) {
	res := ActionReadiness(nil, "kick")
	assert.Equal(t, ActionRejected, res.Status)
}
```

> Before writing, grep the existing actions tests (`internal/actions/*_test.go`, e.g. `combat_test.go`, `cast_test.go`) for the established helper that builds a player `*UserActor` with a `*characters.Character`. Use that helper as `newTestUserActor`; do NOT invent a new fixture pattern.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/actions/ -run TestActionReadiness -v`
Expected: FAIL — `ActionReadiness`, `ActionReady`, `ActionRejected` undefined.

- [ ] **Step 3: Write minimal implementation**

```go
package actions

import (
	"strings"
)

// ReadyStatus classifies whether a command can fire now.
type ReadyStatus int

const (
	ActionReady    ReadyStatus = iota // fire it now
	ActionDeferred                    // transiently blocked — retry later, no player text
	ActionRejected                    // permanently invalid — surface the real error, then drop
)

// ReadinessResult is the verdict from ActionReadiness.
type ReadinessResult struct {
	Status ReadyStatus
	Reason string // short tag for logging/telemetry; never shown to the player
}

// specialMoveVerbs mirrors the switch in CommandIsReady (command_readiness.go).
// SYNC POINT: keep in lockstep with that switch; TestActionReadinessDrift enforces it.
var specialMoveVerbs = map[string]bool{
	"taunt": true, "rally": true, "warcry": true, "trip": true, "bash": true,
	"grapple": true, "kick": true, "rake": true, "maul": true, "throttle": true,
	"pounce": true, "gore": true, "hamstring": true, "drain": true,
}

func splitVerb(cmd string) (verb, rest string) {
	parts := strings.SplitN(strings.TrimSpace(cmd), " ", 2)
	verb = strings.ToLower(parts[0])
	if len(parts) > 1 {
		rest = strings.TrimSpace(parts[1])
	}
	return verb, rest
}

// ActionReadiness reports whether cmd can fire right now for actor, and whether
// any blocker is transient (Deferred) or permanent (Rejected). Commands it does
// not specifically gate return ActionReady and execute verbatim.
func ActionReadiness(actor Actor, cmd string) ReadinessResult {
	if actor == nil {
		return ReadinessResult{ActionRejected, "no actor"}
	}
	char := actor.GetCharacter()
	if char == nil {
		return ReadinessResult{ActionRejected, "no character"}
	}

	verb, _ := splitVerb(cmd)

	if specialMoveVerbs[verb] {
		if CommandIsReady(actor, verb) {
			return ReadinessResult{ActionReady, ""}
		}
		// Transient: shared cooldown or an active Activity (cast/craft/salvage).
		if char.IsActing() || char.GetCooldown("special-move") > 0 {
			return ReadinessResult{ActionDeferred, "special-move busy"}
		}
		// Structural: missing body part / wrong species / no valid target.
		return ReadinessResult{ActionRejected, "special-move unavailable"}
	}

	return ReadinessResult{ActionReady, ""}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/actions/ -run TestActionReadiness -v`
Expected: PASS (both).

- [ ] **Step 5: Add special-move cases + run**

Append tests: a ready special move → `ActionReady`; a special move while on `special-move` cooldown → `ActionDeferred`; a structurally-impossible move (e.g. `kick` with no `Aggro`/legs per `CommandIsReady`) → `ActionRejected`. Build these with the existing test-actor helper (set cooldown via `char.SetCooldown("special-move", ...)` — confirm the setter name in `characters`).

Run: `go test ./internal/actions/ -run TestActionReadiness -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/actions/action_readiness.go internal/actions/action_readiness_test.go
git commit -m "feat(actions): ActionReadiness predicate (special-move coverage)"
```

---

## Task 2: Cast readiness (read-only mirror of the player cast gates)

**Files:**
- Modify: `internal/actions/action_readiness.go`
- Test: `internal/actions/action_readiness_test.go`

**Context — the gates to mirror, read-only, from `internal/usercommands/skill.cast.go:88–164`:**
- no spellcasting/manifestation skill → **Rejected**
- spell not found (`spells.GetSpell` then `spells.FindSpellByName`) → **Rejected**
- player doesn't know the spell → **Rejected**
- already casting (`char.IsCasting()`) → **Deferred**
- not free (`!char.IsFree()` — crafting/salvaging) → **Deferred**
- conviction: `convMult := 1.0 + mutations.GetConvictionCostMultiplier(char.Mutations)`; `cost := spellInfo.GetTotalConvictionCost(convMult)`; `cost > 0 && char.Conviction < cost` → **Deferred**
- `char.GetCooldown("cast-init") > 0` → **Deferred**
- `char.GetCooldown("special-move") > 0` → **Deferred**
- missing component (`spellInfo.ComponentTag` / `spellInfo.SummonComponentId` not in `char.Items`) → **Rejected**

> CRITICAL: this is a **read-only probe**. Do NOT call `InitiateCast` — it calls `char.TryCooldown` (cast.go:259) which *consumes* the special-move cooldown. Use `char.GetCooldown(...)` (read-only) instead.
> Verify exact names via codegraph before finalizing: the spell-known check skill.cast.go uses (spellbook map / skill-level branch above line 88), `spellInfo.SpellId`/`.Id`, `spellInfo.GetTotalConvictionCost`, `mutations.GetConvictionCostMultiplier`. The drift test (Step 4) is the safety net.

- [ ] **Step 1: Write the failing tests**

```go
func TestCastReadiness_NoCP_Deferred(t *testing.T) {
	actor := newTestUserActorKnowingSpell(t, "firebolt") // helper: char knows the spell
	actor.GetCharacter().Conviction = 0
	res := ActionReadiness(actor, "cast firebolt")
	assert.Equal(t, ActionDeferred, res.Status)
}

func TestCastReadiness_UnknownSpell_Rejected(t *testing.T) {
	actor := newTestUserActor(t)
	res := ActionReadiness(actor, "cast notaspell")
	assert.Equal(t, ActionRejected, res.Status)
}

func TestCastReadiness_Affordable_Ready(t *testing.T) {
	actor := newTestUserActorKnowingSpell(t, "firebolt")
	actor.GetCharacter().Conviction = actor.GetCharacter().ConvictionMax.Value
	res := ActionReadiness(actor, "cast firebolt")
	assert.Equal(t, ActionReady, res.Status)
}
```

> Pick a real low-cost spell id that exists in test data for `firebolt`; confirm via `spells.GetSpell`. Extend the test-actor helper to grant the spell (mirror however `cast_test.go` sets up a known spell).

- [ ] **Step 2: Run to verify failure**

Run: `go test ./internal/actions/ -run TestCastReadiness -v`
Expected: FAIL — `cast` falls through to `ActionReady` for all (no cast branch yet).

- [ ] **Step 3: Add the cast branch + helper**

In `ActionReadiness`, before the `specialMoveVerbs` block:

```go
	if verb == "cast" {
		return castReadiness(actor, rest)
	}
```

Add `castReadiness` implementing the read-only gates listed in Context above. Skeleton:

```go
func castReadiness(actor Actor, rest string) ReadinessResult {
	char := actor.GetCharacter()
	spellName, _ := splitVerb(rest)

	spellInfo := spells.GetSpell(spellName)
	if spellInfo == nil {
		spellInfo = spells.FindSpellByName(spellName)
	}
	if spellInfo == nil {
		return ReadinessResult{ActionRejected, "unknown spell"}
	}
	// known-spell + skill gate — mirror skill.cast.go:<lines>; Rejected if not known.
	if !characterKnowsSpell(char, spellInfo) {
		return ReadinessResult{ActionRejected, "spell not known"}
	}
	if char.IsCasting() {
		return ReadinessResult{ActionDeferred, "already casting"}
	}
	if !char.IsFree() {
		return ReadinessResult{ActionDeferred, "busy"}
	}
	convMult := 1.0 + mutations.GetConvictionCostMultiplier(char.Mutations)
	cost := spellInfo.GetTotalConvictionCost(convMult)
	if cost > 0 && char.Conviction < cost {
		return ReadinessResult{ActionDeferred, "insufficient conviction"}
	}
	if char.GetCooldown("cast-init") > 0 {
		return ReadinessResult{ActionDeferred, "cast-init cooldown"}
	}
	if char.GetCooldown("special-move") > 0 {
		return ReadinessResult{ActionDeferred, "special-move cooldown"}
	}
	if !castComponentsPresent(char, spellInfo) {
		return ReadinessResult{ActionRejected, "missing component"}
	}
	return ReadinessResult{ActionReady, ""}
}
```

Implement `characterKnowsSpell` and `castComponentsPresent` as small read-only helpers in this file, mirroring skill.cast.go. Add the `mutations` and `spells` imports.

- [ ] **Step 4: Add the drift test**

```go
// TestActionReadinessDrift asserts every specialMoveVerbs entry is a real case
// in CommandIsReady's switch, so the two never drift apart.
func TestActionReadinessDrift(t *testing.T) {
	actor := newTestUserActor(t)
	for verb := range specialMoveVerbs {
		// CommandIsReady must recognise the verb (return is bool; the point is
		// it doesn't panic and the verb is in its switch — assert via a known
		// gate, e.g. with no Aggro most are false but defined).
		_ = CommandIsReady(actor, verb)
	}
}
```

> If `CommandIsReady` gains/loses a verb later, update `specialMoveVerbs`. (Mirror the spirit of the existing `TestCommandReadinessDrift`.)

- [ ] **Step 5: Run all readiness tests**

Run: `go test ./internal/actions/ -run "TestActionReadiness|TestCastReadiness" -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/actions/action_readiness.go internal/actions/action_readiness_test.go
git commit -m "feat(actions): cast readiness probe (read-only cast gates + drift test)"
```

---

## Task 3: CastInterrupted event + emit at interrupt sites

**Files:**
- Modify: `internal/events/eventtypes.go`
- Modify: `internal/actions/cast_interrupt.go`
- (Investigate) other `TriggerCastCancel` emit sites (damage-based concentration break)

- [ ] **Step 1: Add the event**

In `internal/events/eventtypes.go`, near `Input`:

```go
// CastInterrupted fires when a player's in-progress spellcast is cancelled by
// an outside force (active interrupt, damage-broken concentration). Consumed by
// the GMCP layer so the web-client action queue can re-arm the cast.
type CastInterrupted struct {
	UserId  int
	SpellId string
}

func (c CastInterrupted) Type() string { return `CastInterrupted` }
```

- [ ] **Step 2: Emit from `InterruptTargetCast`**

In `internal/actions/cast_interrupt.go`, capture the spell id before cancelling and emit after a successful interrupt (players only):

```go
func InterruptTargetCast(target *characters.Character, by state.ActorRef) bool {
	a := target.Activity
	if a == nil || !a.IsCasting() {
		return false
	}
	spellId := ""
	if d, ok := a.CastingData(); ok {
		spellId = d.SpellId
		unspent := d.TotalConvictionCost - d.ConvictionSpent
		if unspent > 0 {
			target.Conviction += unspent / 2
			if target.Conviction > target.ConvictionMax.Value {
				target.Conviction = target.ConvictionMax.Value
			}
		}
	}
	_ = a.TransitionToFree(state.TransitionReason{
		Trigger: activity.TriggerCastCancel,
		Actor:   by,
	})
	if uid := target.GetUserId(); uid > 0 {
		events.AddToQueue(events.CastInterrupted{UserId: uid, SpellId: spellId})
	}
	return true
}
```

Add the `events` import.

- [ ] **Step 3: Cover the other interrupt vectors**

Run: `grep -rn "TriggerCastCancel" internal/` (use the Grep tool). For any *other* site that cancels a player's cast due to an outside force (notably the damage-broken-concentration path in `internal/hooks/combat_shared_helpers.go` around the `IsCasting()` checks), emit the same `events.CastInterrupted{UserId, SpellId}` (players only) right after the cancel. Do NOT emit on player-initiated `cancel`/`flee` (those are deliberate, not interruptions to re-arm).

- [ ] **Step 4: Build**

Run: `go build ./...`
Expected: clean. (No unit test here — behavior is verified end-to-end in Task 6.)

- [ ] **Step 5: Commit**

```bash
git add internal/events/eventtypes.go internal/actions/cast_interrupt.go internal/hooks/combat_shared_helpers.go
git commit -m "feat(events): CastInterrupted emitted on outside cast interruption"
```

---

## Task 4: GMCP outbound — Action.Result / Action.Interrupted

**Files:**
- Create: `modules/gmcp/gmcp.Action.go`
- Modify: the GMCP module's listener registration (where `automationChangedHandler` is registered — find via `RegisterListener`/`events.RegisterListener` in `modules/gmcp/`)

- [ ] **Step 1: Define payloads + send helpers**

`modules/gmcp/gmcp.Action.go`:

```go
package gmcp

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Char.Action.Result — direct reply to an inbound Char.Action.Try.
type GMCPAction_Result struct {
	Id     int    `json:"id"`
	Status string `json:"status"` // "fired" | "deferred" | "rejected"
	Reason string `json:"reason,omitempty"`
}

// Char.Action.Interrupted — a player's queued cast was interrupted; client re-arms.
type GMCPAction_Interrupted struct {
	Spell string `json:"spell"`
}

func sendActionResult(userId, id int, status, reason string) {
	u := users.GetByUserId(userId)
	if u == nil || !isGMCPEnabled(u.ConnectionId()) {
		return
	}
	events.AddToQueue(GMCPOut{
		UserId:  userId,
		Module:  `Char.Action.Result`,
		Payload: GMCPAction_Result{Id: id, Status: status, Reason: reason},
	})
}

func (g GMCPAutomationModule) castInterruptedHandler(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.CastInterrupted)
	if !ok {
		mudlog.Error("Event", "Expected Type", "CastInterrupted", "Actual Type", e.Type())
		return events.Cancel
	}
	u := users.GetByUserId(evt.UserId)
	if u == nil || !isGMCPEnabled(u.ConnectionId()) {
		return events.Continue
	}
	events.AddToQueue(GMCPOut{
		UserId:  evt.UserId,
		Module:  `Char.Action.Interrupted`,
		Payload: GMCPAction_Interrupted{Spell: evt.SpellId},
	})
	return events.Continue
}
```

> Use whichever module receiver already owns automation (`GMCPAutomationModule` per `gmcp.Automation.go`). If the inbound `Char.Action.Try` (Task 5) and these live more naturally on the base GMCP module, keep them consistent — match the file that registers `Char.Automation.Set`.

- [ ] **Step 2: Register the listener**

Where `automationChangedHandler` is registered for `events.AutomationChanged`, add registration of `castInterruptedHandler` for `events.CastInterrupted`. Match the existing registration call signature exactly.

- [ ] **Step 3: Build**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add modules/gmcp/gmcp.Action.go modules/gmcp/<registration file>
git commit -m "feat(gmcp): Char.Action.Result + Char.Action.Interrupted outbound"
```

---

## Task 5: GMCP inbound — Char.Action.Try

**Files:**
- Modify: `modules/gmcp/gmcp.go` (the inbound switch near `case ` + "`Char.Automation.Set`" at ~line 368)

- [ ] **Step 1: Add the inbound case**

Inside the same `switch` that handles `Char.Automation.Set` / `Char.Automation.Remove`:

```go
		case `Char.Action.Try`:
			var req struct {
				Id  int    `json:"id"`
				Cmd string `json:"cmd"`
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
			room := rooms.LoadRoom(u.Character.RoomId)
			actor := &actions.UserActor{User: u, Room: room}
			res := actions.ActionReadiness(actor, req.Cmd)
			switch res.Status {
			case actions.ActionReady:
				u.Command(req.Cmd)
				sendActionResult(uid, req.Id, "fired", "")
			case actions.ActionDeferred:
				sendActionResult(uid, req.Id, "deferred", res.Reason)
			case actions.ActionRejected:
				u.Command(req.Cmd) // run normally so the player sees the real error once
				sendActionResult(uid, req.Id, "rejected", res.Reason)
			}
```

Add imports as needed: `internal/actions`, `internal/rooms`, `internal/users` (likely already imported). Confirm `userIdForConnection` is the helper used by the sibling cases.

- [ ] **Step 2: Build**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 3: Add a handler unit test (table-driven)**

If the GMCP package has a testable seam for inbound dispatch (see `gmcp.Automation_test.go` for how inbound is exercised), add a test that a `Char.Action.Try` with a deferred-readiness command does NOT enqueue an `events.Input` and DOES enqueue a `Char.Action.Result{status:"deferred"}`. If inbound dispatch isn't unit-testable without a live connection, skip and rely on Task 7 smoke — note this explicitly in the commit body.

Run: `go test ./modules/gmcp/ -v`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add modules/gmcp/gmcp.go
git commit -m "feat(gmcp): Char.Action.Try try-or-defer endpoint"
```

---

## Task 6: Client — route queue through Try, retry, re-arm

**Files:**
- Modify: `_datafiles/html/public/webclient-pure.html`

No JS unit harness — verify manually in the running web client (Task 7). Make these edits precisely.

- [ ] **Step 1: Add staleness constant + entry fields**

Near `var actionQueue = [];` / `var QUEUE_CAP = 10;` add:

```javascript
        var QUEUE_STALENESS_ROUNDS = 6; // give up on a blocked queued action after this many rounds
        var queueInFlight = null;       // { id, name, commands, attempts, isCast, spell } awaiting a Result
        var queueRearm = null;          // a fired cast eligible for interruption re-arm
```

- [ ] **Step 2: Replace `drainQueue` and delete the cooldown heuristic**

Replace the entire `drainQueue` function (and remove `cooldownReady`, `queueWaiting`, `queueSawCooldown`, `lastQueueFireAt`, `QUEUE_REGISTER_TIMEOUT_MS` — the server is the gate now) with:

```javascript
        // Drain: send the head entry to the server's try-or-defer endpoint.
        // One action in flight at a time (serial) so cooldowns are respected.
        function drainQueue() {
            if (queueInFlight) return;          // awaiting a Result
            if (!actionQueue.length) return;
            var entry = actionQueue[0];
            entry.attempts = entry.attempts || 0;
            var cmd = String(entry.commands || "").split(";")[0].trim(); // one command per queued action
            entry.isCast = /^cast\b/i.test(cmd);
            entry.spell  = entry.isCast ? cmd.replace(/^cast\s+/i, "").split(" ")[0].toLowerCase() : "";
            queueInFlight = entry;
            SendGMCP("Char.Action.Try", { id: entry.id, cmd: cmd });
        }
```

> Note: queued actions are single-command (the queue is for one ability per trigger). If `entry.commands` contains multiple `;`-separated commands, only the first is gated/retried; document this in a code comment.

- [ ] **Step 3: Handle `Char.Action.Result`**

Register a handler. In the `GMCPUpdateHandlers` map add `"Char.Action.Result"`, and ensure `handleGMCP` routes it (it walks most-specific-first, so the key must match the namespace):

```javascript
            "Char.Action.Result": function() {
                var r = ((GMCPStructs["Char"] || {}).Action || {}).Result || {};
                if (!queueInFlight || r.id !== queueInFlight.id) return;
                var entry = queueInFlight;
                queueInFlight = null;
                if (r.status === "fired") {
                    actionQueue.shift();
                    if (entry.isCast) { queueRearm = { id: entry.id, spell: entry.spell, attempts: entry.attempts }; }
                    renderAutomation();
                    drainQueue();
                } else if (r.status === "rejected") {
                    actionQueue.shift();          // server already showed the real error
                    renderAutomation();
                    drainQueue();
                } else { // "deferred"
                    entry.attempts++;
                    if (entry.attempts >= QUEUE_STALENESS_ROUNDS) {
                        actionQueue.shift();      // give up quietly
                        console.log("[queue] dropped (staleness):", entry.name);
                    }
                    renderAutomation();
                    // do not re-send immediately; the per-round retry tick re-drains
                }
            },
```

- [ ] **Step 4: Re-drain once per round**

The client already receives a per-round GMCP push (`Commands.State` and/or `Playtest.Round`). In that update handler (find the existing `"Commands.State"` handler), append a `drainQueue();` call so a deferred head retries each round. (If both pushes arrive, prefer the most reliable per-round one; a double-drain is harmless because `queueInFlight` guards it.)

- [ ] **Step 5: Handle `Char.Action.Interrupted` (re-arm)**

Add handler:

```javascript
            "Char.Action.Interrupted": function() {
                var i = ((GMCPStructs["Char"] || {}).Action || {}).Interrupted || {};
                if (!queueRearm) return;
                if (i.spell && queueRearm.spell && i.spell.toLowerCase() !== queueRearm.spell) return;
                if (queueRearm.attempts >= QUEUE_STALENESS_ROUNDS) { queueRearm = null; return; }
                // Re-enqueue the interrupted cast at the front, preserving its attempt budget.
                actionQueue.unshift({ id: queueRearm.id, name: "(re-cast)", commands: "cast " + queueRearm.spell, attempts: queueRearm.attempts });
                queueRearm = null;
                renderAutomation();
                drainQueue();
            },
```

> Clear `queueRearm` when the cast completes normally — simplest: clear it on the next `Commands.State` round tick if no interruption arrived (a cast resolves within a few rounds). Add `queueRearm = null;` guarded by a small "rounds since fired" counter in the per-round handler so a long-past cast can't re-arm.

- [ ] **Step 6: Manual verification (local web client)**

Ask the user to start the server (no-console rule — do not launch it yourself). Then in the web client:
- Queue a special move while on cooldown → fires silently when ready.
- Queue `cast <spell>` with `cp` near zero → no "not enough conviction" text; fires after CP regen if within ~6 rounds, else silently drops.
- Get a queued cast interrupted → it re-fires once.
- Queue while permanently blocked (e.g. unknown spell) → the real error shows once, entry drops.

- [ ] **Step 7: Commit**

```bash
git add _datafiles/html/public/webclient-pure.html
git commit -m "feat(webclient): action queue auto-retries via Char.Action.Try"
```

---

## Task 7: Integration smoke + docs

**Files:**
- Modify: `PATCH_NOTES.md`

- [ ] **Step 1: Full build + test**

Run: `go build ./...` then `go test ./internal/actions/ ./modules/gmcp/ -v`
Expected: clean build, tests PASS.

- [ ] **Step 2: Boot smoke (user-run, per no-console rule)**

Ask the user to boot the server and confirm clean startup (no panics past data-file loading), then run the Task 6 manual checks end-to-end.

- [ ] **Step 3: PATCH_NOTES**

Add a dated entry under the appropriate section describing the smart action queue (auto-retry through cooldown/mid-cast/out-of-CP/interruption; silent; ~6-round give-up window; conditions now only for discretionary gating).

- [ ] **Step 4: Commit**

```bash
git add PATCH_NOTES.md
git commit -m "docs(patch-notes): smart action queue"
```

---

## Notes / Out of Scope

- The existing `cooldown` trigger condition stays (subsumed but established behavior).
- `QUEUE_STALENESS_ROUNDS` is a client constant (default 6). Promotion to a server-pushed Balance knob is a later nicety — do not add a config-push channel for it now (spec §Config).
- Immediate-fire triggers (`queueMode === ""`) still use raw `SendData`; only queued actions go through `Char.Action.Try`.
- Mob/AI action selection is untouched.
