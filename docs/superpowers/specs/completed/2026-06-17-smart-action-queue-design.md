# Smart Action Queue — Design Spec

**Date:** 2026-06-17
**Status:** Approved design, implementation deferred until after the newbie-area prod push.
**Scope:** Web-client action queue + a server-side action-readiness endpoint. Player-only.

## Problem

The web-client trigger editor lets a player attach if/else **conditions** to a
trigger and choose a **queue mode** (fire now / queue at back / queue at front).
The queue then drains gated only on the shared `special-move` cooldown.

To make a queued action wait through other transient blockers — out of
conviction, mid-cast, a cast that just got interrupted — the player has to hand-
author a condition for each case. That is the wrong model: those blockers are
*never a choice*. A player always wants a queued action to fire as soon as it
*can*, and to give up if it stays blocked too long. Hand-encoding "is my
spellcasting active" or "is my CP depleted" as conditions is busywork, and the
condition UI for them is vague and easy to misuse.

Two conditions added on 2026-06-17 (`state:casting`, `state:outofcp`, plus the
`casting` field on `Char.Vitals`) embody exactly this wrong model and are
reverted as part of this work (see "Reverts" below).

## Locked Decisions

1. **Queue = auto-retry until it fires.** Queuing an action means: hold it and
   keep re-attempting through cooldown, mid-cast, out-of-CP, and interruption
   automatically. No conditions required for any of those cases.
2. **Conditions stay, but only for genuine discretionary gating** — HP/SP/CP
   thresholds, status includes/excludes, captures, target matching. The
   transient-blocker conditions are removed.
3. **Give up after a short staleness window.** A queued action retries each
   round and is dropped quietly once it has stayed blocked for
   `ActionQueueStalenessRounds` (default **6**). This fires fast actions
   (cooldown 2–3 rds, mid-cast 2–4 rds) but never fires a stale action many
   rounds late (a deep CP hole drops instead).
4. **Predictive and silent.** The player never sees failure text from a queued
   retry. The server authoritatively decides whether an action can fire now and
   reports back; the client holds the entry silently until it fires or expires.
5. **Architecture: server "try-or-defer" endpoint (Approach A).** The client
   stops firing queued actions as raw text and routes them through a new inbound
   GMCP verb. The server runs one authoritative readiness check and either
   executes the command or reports it deferred — no player-facing error on a
   defer.

## Architecture

### Flow

```
Trigger matches a line, queueMode != ""        (client)
  └─> enqueueTriggerAction(entry)              (client: FIFO, dedup by triggerId)
       └─> drainQueue() sends head entry as
           GMCP  Char.Action.Try {id, cmd}     (client → server)
                 │
                 ▼
           ActionReadiness(actor, cmd)          (server, authoritative)
              ├─ Ready    → user.Command(cmd); reply Result{id,"fired"}
              ├─ Deferred → (do nothing, no text); reply Result{id,"deferred",reason}
              └─ Rejected → user.Command(cmd)   ← runs normally so the player sees
                            the real error once; reply Result{id,"rejected"}
                 │
                 ▼
           Char.Action.Result {id,status,reason}  (server → client)
              ├─ fired    → drop entry, drain next
              ├─ deferred → keep entry, retry next round; drop after staleness window
              └─ rejected → drop entry (error already shown)

  Cast interrupted after a "fired" cast:
    server emits Char.Action.Interrupted {spell}  (server → client)
      └─> client re-enqueues the matching in-flight cast (front), within window
```

### Server components

**1. `actions.ActionReadiness(actor Actor, cmd string) ReadinessResult`**
*(new, `internal/actions/action_readiness.go`)*

Authoritative "can this command fire right now?" evaluator. Returns one of:

- `Ready` — fire it.
- `Deferred{Reason}` — transiently blocked; retry later, no player text.
- `Rejected{Reason}` — permanently invalid for this character/state; let the
  normal command path surface its error, then drop.

Coverage (the cases that motivated this; everything else falls through to
`Ready` and executes normally):

| Command shape        | Deferred when…                                   | Rejected when…                          |
|----------------------|--------------------------------------------------|-----------------------------------------|
| special-move verbs   | `!CommandIsReady` for a *transient* reason: special-move cooldown active, or `IsActing()` (casting/crafting/salvaging) | `CommandIsReady` false for a *structural* reason: missing body part / wrong species / no valid target |
| `cast <spell>`       | already casting, special-move cooldown active, or insufficient conviction for the spell's cost | spell unknown or not in spellbook |
| anything else        | —                                                | —  (returns `Ready`, executes verbatim) |

Reuses `CommandIsReady` for special moves. For casts, reuses `cast.go`'s
pre-resolution checks (`AlreadyCasting`, `OnCooldown`, conviction cost) **without
consuming the cast** — factor the affordability/cooldown probe out of
`actions.Cast` into a shared helper so the readiness probe and the real cast use
one source of truth.

> **Sync point.** Like `CommandIsReady`'s drift test, the cast/special-move
> gates here must stay in lockstep with the execute paths. Add a drift test
> mirroring `TestCommandReadinessDrift`.

The special-move distinction between Deferred (cooldown / IsActing) and Rejected
(missing body part / no target) matters: a cooldown clears on its own (retry),
but "you have no legs to kick with" never will (drop, show once).

**2. `Char.Action.Try` inbound handler** *(`modules/gmcp/gmcp.go`, beside the
existing `Char.Automation.Set` case)*

Payload `{ id int, cmd string }`. Resolves the user, calls `ActionReadiness`:
- `Ready` → `user.Command(cmd)`; reply `Result{id,"fired"}`.
- `Deferred` → reply `Result{id,"deferred",reason}` only. No command, no text.
- `Rejected` → `user.Command(cmd)` (normal error path), reply
  `Result{id,"rejected",reason}`.

**3. `Char.Action.Result` / `Char.Action.Interrupted` outbound** *(new GMCP
sub-namespace, `modules/gmcp/gmcp.Action.go`)*

- `Result {id, status: "fired"|"deferred"|"rejected", reason}` — direct reply to
  a `Try`.
- `Interrupted {spell}` — emitted from the existing cast-interruption path
  (`internal/hooks/combat_shared_helpers.go` / `cast_interrupt.go`) to the
  affected user. Always emitted; the client decides relevance. (The server does
  not track which casts came from the queue.)

### Client components *(`_datafiles/html/public/webclient-pure.html`)*

**Queue state** — extend each `actionQueue` entry with `attempts` (rounds tried)
and, for casts, an `inFlight` marker once a `fired` result arrives for a multi-
round cast.

**`drainQueue()`** — replace the `cooldownReady()` heuristic entirely:
- If head entry is awaiting a `Result`, do nothing (serial — one in flight).
- Otherwise send `SendGMCP("Char.Action.Try", {id, cmd})` for the head and mark
  it awaiting.

**`Char.Action.Result` handler:**
- `fired` → if the command is a cast, mark the entry `inFlight` and keep it set
  aside for interruption re-arm until the cast resolves (cleared on the next
  successful round / a short timeout); otherwise drop it. Drain next.
- `deferred` → increment `attempts`; if `attempts >= ActionQueueStalenessRounds`
  drop quietly (console log only); else leave queued and re-attempt on the next
  round tick (driven by the per-round `Commands.State` / `Playtest.Round` push
  the client already receives).
- `rejected` → drop the entry (the server already showed the real error).

**`Char.Action.Interrupted` handler** — if an `inFlight` cast entry matches the
interrupted spell and is still within its staleness window, re-enqueue it at the
front and clear `inFlight` so it retries.

**Retry clock** — retries advance on the per-round GMCP push the client already
consumes (no new timers, no `setTimeout` race). The old
`QUEUE_REGISTER_TIMEOUT_MS` heuristic and `cooldownReady()` gate are deleted; the
server is now the gate.

**Unchanged client behavior:** `QUEUE_CAP` (10), dedup by `triggerId`, priority-
FIFO front insertion, queue cleared on death, "Clear" button. Immediate-fire
trigger commands (`queueMode === ""`) still go out via `SendData` raw — no Try,
no retry. Only **queued** actions use the new path.

## Config

- `ActionQueueStalenessRounds` (new, `Balance`, default **6**) — rounds a queued
  action retries before it is dropped. The retry loop runs client-side, so the
  bound must reach the client: default to a client constant matching `6`, and
  during planning confirm whether a client-facing config push already exists
  (other client-tunables) — if so, source it from the Balance knob; if not, the
  constant stands and promotion is a later nicety. Do **not** add a new config-
  push channel just for this.

## Removals (Reverts of 2026-06-17 session work)

These ship as part of cleaning up before the prod push, independent of building
the queue:

- `modules/gmcp/gmcp.Char.go` — remove the `Casting` field from
  `GMCPCharModule_Payload_Vitals` and the `Casting: user.Character.IsCasting()`
  populate line.
- `_datafiles/html/public/webclient-pure.html` — remove the `state:casting` /
  `state:outofcp` source options, the `opsByKind["state"]` entry, the
  `rebuildOps` value-input hide for `state`, the pre-fill `state` mapping, the
  save no-value `state` branch, and the `state` block in `evalTriggerCondition`.
- `internal/users/userrecord.go` — restore the `TriggerCondition` doc comment.

The equipment-panel combo Drop fix and the sleeping-mob gating from the same
session are unrelated and stay.

## Edge Cases

- **Generic commands** (`say`, movement, etc.) return `Ready` and execute at
  once — queuing them is harmless and behaves as today.
- **Serial drain** — one action in flight at a time, so the queue respects the
  shared cooldown naturally (the next `Try` won't be sent until the prior result
  lands).
- **Interruption with no matching in-flight cast** — client ignores the signal.
- **Disconnect / reconnect** — the queue is client-side and is lost, as today.
- **TOCTOU** — eliminated: the server *executes* on Ready in the same handler
  call, so there is no readiness-check-then-send gap.
- **Deferred that never clears** — dropped silently after the staleness window;
  no user-facing text either way.

## Testing

**Server (unit):**
- `ActionReadiness`: cast with insufficient CP → `Deferred`; unknown/unlearned
  spell → `Rejected`; special move on cooldown / while `IsActing` → `Deferred`;
  special move missing required body part → `Rejected`; ready special move and
  affordable cast → `Ready`; generic command → `Ready`.
- Drift test mirroring `TestCommandReadinessDrift` for the cast gates.
- `Char.Action.Try` handler: fired runs `user.Command`; deferred runs nothing
  and emits no message; rejected runs the command (error path) once.
- Interruption emits `Char.Action.Interrupted` for the casting user.

**Client (manual / playtest, no JS unit harness):**
- Queue a special move while on cooldown → fires silently when ready.
- Queue `cast <spell>` with no CP → fires after CP regen if within window; drops
  quietly past the window; no failure text in either case.
- Interrupt a queued cast → it re-fires once concentration is regained.
- Queue past the staleness window while permanently blocked → drops with no text.

## Out of Scope

- The existing `cooldown` condition stays as-is. It is arguably subsumed by
  auto-retry but removing it changes established behavior; revisit separately.
- No server-side persistent queue; the queue remains client-side.
- Mob/AI action selection is unaffected — this is the player web client only.
- No new discretionary condition types.
