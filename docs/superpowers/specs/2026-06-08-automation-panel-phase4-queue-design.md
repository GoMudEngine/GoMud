# Automation Panel — Phase 4 Design (Cooldown-gated action queue)

**Date:** 2026-06-08
**Status:** Approved (brainstorm) — ready for implementation plan
**Author:** brainstormed with the visual companion

## Goal

Let a player set up a **rotation that auto-executes as the shared ability
cooldown allows**. A trigger can *queue* its action instead of firing it
immediately; a single global **FIFO queue** drains **one ability per
`special-move` cooldown cycle**. The driving use case: several buffs drop at
once → their re-cast triggers all fire → the abilities go out in order as the
cooldown frees, instead of failing because they all share one cooldown.

This is **Phase 4** of the automation panel (Phases 1–3 — Macros/Aliases/Ticks/
Triggers — are shipped, [[project_web_client_automation_panel]]). Parked
locally like the rest of the web overhaul. Two bundled workstreams: the **queue**
(client) and the **rally/warcry expiry echoes** (server content) that enable the
"react to a buff dropping" half of the use case.

## Context

- **Triggers** (Phase 3) match incoming game text (`*` wildcards → `$1…$n`) and
  fire command(s), optionally gated by one if/else condition. Stored as
  `users.UserTrigger`; the client engine taps `socket.onmessage`, matches, and
  `SendData`s the commands. (`webclient-pure.html`.)
- **The shared cooldown:** kick/trip/bash/grapple/taunt/rally/warcry/**any
  spell** all share ONE cooldown, tracking tag **`special-move`** (deliberate
  tradeoff design). **Potions do NOT** use it (grenades do — ignored; ranged
  redesign is future). Exposed client-side via
  `GMCPStructs["Commands"].State.cooldowns["special-move"]` — **ready ⟺ that key
  is absent** from the map.
- **Inbound GMCP** path (`Char.Automation.Set/Remove` + the `SendGMCP` binary
  helper + ws `TelnetIACHandler`) is wired (Phases 2–3) and reused for the new
  per-trigger field.
- Most buffs emit an **expiry message** when they wear off (so a trigger can
  match it); **rally and warcry do not** — that gap is fixed here. The live
  field is **`end_user_text`** on the buff (sent via `CategoryBuffExpire` in
  `NewTurn_PruneBuffs.go` on natural expiry); a stray `expireMessage` key in one
  buff file is dead YAML — do NOT use it.

## Locked decisions (from the brainstorm)

1. **One global FIFO queue**, client-side, **ephemeral** (not persisted; gone on
   reload/reconnect; actively cleared on **death**).
2. **Per-trigger queue placement** — a trigger's action can be: **off** (fire
   immediately, today's behavior) · **queue at back** (normal FIFO) · **queue at
   front** (priority). Persisted on the trigger.
3. **Priority FIFO ordering:** the queue drains all **front** entries before any
   **back** entries; within each lane it is fire-order. A new front inserts
   after existing fronts but ahead of all backs; a new back appends to the end.
4. **Dedup:** at most **one entry per trigger** — a trigger already queued won't
   add a second entry while it's pending.
5. **Cap = 10.** A push beyond 10 is rejected (the queue holds; notify quietly).
6. **One ability per cooldown, no success-tracking.** The processor fires the
   next entry when `special-move` is ready, then waits a full cooldown cycle. It
   does NOT detect whether the action succeeded — if an attempt fails (e.g. lost
   concentration), it is the **player's responsibility** to have a trigger that
   matches the failure message and re-queues the action (at back). The queue
   never auto-retries.
7. **Manual Clear only** — a Clear button empties the queue. **No** auto-clear on
   leaving combat. **Death clears the queue** (alongside buffs/debuffs).
8. **UI:** queued triggers are **highlighted and floated to the top** of the
   Triggers tab with a **FIFO badge** (`1` = next to fire, brass; others red),
   plus a queue **status bar** (count + cooldown state + Clear). As #1 fires it
   drops back to its normal position and #2 becomes "next". (Interleaved into the
   trigger list, not a separate strip — confirmed in the mock.)
9. **Rally/warcry expiry echoes** (server content) — add expiry messages to those
   two buffs so a trigger can match "rally fades" → enqueue the recast.

Visual source of truth: `queue-panel.html` (companion session; reproduced here).

## Part A — Per-trigger queue placement (storage + GMCP + editor)

- **Storage:** add `QueueMode string` to `users.UserTrigger`
  (`yaml:"queuemode,omitempty" json:"queueMode,omitempty"`), values `""` (off),
  `"back"`, `"front"`. (The `TriggerCondition`/rest of `UserTrigger` unchanged.)
- **GMCP:** add `QueueMode string `json:"queueMode,omitempty"`` to
  `GMCPAutomation_Trigger`; map it in `buildAutomationPayload`. (The inbound
  `Char.Automation.Set` already unmarshals the full `UserTrigger`, so the field
  round-trips with no handler change.)
- **Editor:** in the trigger editor, add a **"Queue this action"** control —
  e.g. a select: *Fire immediately* · *Queue at back* · *Queue at front*. Save
  writes `queueMode`; pre-fill restores it.

## Part B — The client queue + processor

In `webclient-pure.html`:

- **State:** `var actionQueue = [];` — entries `{ triggerId, name, commands,
  placement }` (placement `"back"|"front"`). Module-level.
- **Enqueue (on trigger match):** when a matched trigger has `queueMode` set,
  instead of firing now:
  - **Dedup:** if an entry with this `triggerId` already exists, do nothing.
  - **Cap:** if `actionQueue.length >= 10`, reject (quiet notice; don't add).
  - **Insert (priority FIFO):** `back` → push to end; `front` → insert *after the
    last existing front entry* (and before the first back) so fronts stay
    fire-ordered ahead of backs.
  - Re-render the Triggers tab (highlight/float) + the status bar.
- **Processor — `drainQueue()`:** runs on each `Commands.State` / `Char` GMCP
  update (and a low-frequency safety interval). If `actionQueue` is non-empty AND
  `special-move` is **ready** (absent from `Commands.State.cooldowns`) AND we are
  not in the post-fire wait:
  - Shift the head entry, `SendData` its commands (split `;`, like fire-now).
  - Set a `firedAt` guard so we don't fire again until the cooldown has visibly
    registered — i.e. wait until a subsequent GMCP update shows `special-move`
    present (cooling) OR a short fallback timeout, THEN resume. This enforces
    **one ability per cooldown** and avoids dumping the whole queue in one tick
    before the server registers the cooldown.
  - Re-render.
- The processor is **append-only-safe** with the trigger engine (both run in the
  client; enqueue happens in the match path, drain in the GMCP-update path).

## Part C — Clear + death-clear

- **Clear button** (status bar): empties `actionQueue`, re-renders.
- **Death clears the queue.** Detect the player's death client-side and empty the
  queue. *Open item (resolve in plan):* prefer a GMCP death signal if one exists
  (check for a death/respawn event or a Char indicator); otherwise match the
  death message in the text stream (the same tap the trigger engine uses). Reload
  / reconnect clears naturally (ephemeral state).

## Part D — UI (Triggers tab)

- **Queued triggers:** rendered at the **top** of the list, in queue order, with
  a `.pending` highlight (red glow) and a circular **FIFO badge** — position `1`
  brass ("next"), `2…N` red. Below a divider, the normal (non-queued / not-
  currently-queued) triggers in their usual order.
- **Status bar** above the list: `▸ N queued`, the shared-cooldown state
  (ready / cooling), and a **Clear** button. Hidden when the queue is empty.
- When an entry fires, its trigger drops back into the normal list and the badges
  renumber. Reuse the leather styling (`.auto-row`, brass, madder/red accents).

## Part E — Rally/warcry expiry echoes (server content)

- Most buffs emit a message when they expire; **rally** and **warcry** don't.
  Add an **`end_user_text`** line (the live buff wear-off field, emitted by
  `NewTurn_PruneBuffs.go` on natural expiry) to the rally (`80-rally.yaml`) and
  warcry (`79-warcry.yaml`) buff definitions so a trigger can match the echo.
  Player-facing, first-person, no hard numbers.

## Scope / boundaries

- **In:** the `QueueMode` trigger field (storage + GMCP + editor), the client
  queue + cooldown-paced processor (priority FIFO, dedup, cap 10, one-per-
  cooldown, no retry), Clear + death-clear, the highlight/float/badge + status-
  bar UI, and the rally/warcry expiry echoes.
- **Out (deferred / not now):** persisting the queue across sessions; auto-retry
  of failed actions (player's own trigger handles it); queueing potions or
  non-`special-move` actions behind this gate; grenades/ranged (future redesign);
  multi-cooldown queues (only `special-move` is gated).
- **Server work:** small — one new `UserTrigger` field + its GMCP mapping, plus
  the buff expiry echoes (content). The processor/UI are client-side.

## Acceptance / verification

- A trigger set to **Queue at back** enqueues (doesn't fire immediately) on
  match; one set to **Queue at front** jumps ahead of back entries (priority
  FIFO; multiple fronts keep fire-order). **Off** still fires immediately.
- Multiple triggers firing together (e.g. 3 buff-drop echoes) enqueue and drain
  **one ability per `special-move` cooldown** in the correct order — no
  all-at-once dump, no double-fire before the cooldown registers.
- Dedup holds (a re-firing queued trigger doesn't add a second entry); cap 10
  rejects the 11th.
- **Clear** empties the queue; **death** clears it; reload/reconnect clears it.
- The Triggers tab highlights + floats queued triggers with correct FIFO badges
  and a working status bar; entries drop back to normal as they fire.
- Rally and warcry now print an expiry message a trigger can match.
- `go build ./...` clean; server boots clean; `/webclient` loads with no console
  errors; macros/aliases/ticks/(non-queued) triggers unaffected.

## Risks / open items

- **Death signal** (Part C): confirm the cleanest client-side death detection
  (GMCP vs text-pattern) in the plan.
- **Processor timing** (Part B): the post-fire wait must reliably gate on the
  cooldown GMCP update so exactly one ability fires per cooldown; tune the
  fallback timeout.
- **Buff-expiry mechanism** (Part E): confirm the field/hook before editing the
  rally/warcry buffs.
- **Cooldown source of truth:** the queue and the Phase-3 cooldown *condition*
  both read `Commands.State.cooldowns["special-move"]`; keep them consistent.
