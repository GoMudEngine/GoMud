# Newcomer Tutorial as a Guided Chain Quest — Design

**Date:** 2026-07-19
**Status:** Approved (design), pending spec review
**Relates to:** `project_newcomer_tutorial_redesign` (the 2026-07-17 guide-led Dewey
tutorial this reworks), `project_newbie_onboarding_feedback_backlog`

## Problem

The route-1 "complete newbie" tutorial (Newcomer Antechamber, rooms 6258–6262 +
6467, guided by Dewey / mob 9491) is a per-room behavior-tree state machine.
Malia's 2026-07-18 playtest surfaced:

1. **Compound instructions.** Several of Dewey's spoken lines bundle multiple
   actions ("say hello. Then ask dewey world"; every movement nudge lists all
   remaining steps). A brand-new player cannot parse a three-step instruction.
2. **No persistent, re-readable instruction.** Dewey speaks each prompt once via
   `mob_say`. If it scrolls past, there is nothing to type to get it back. Malia
   **got stuck after killing the straw effigy** because she never saw the
   `cast spike` prompt.
3. **The effigy is killable.** Straw effigy 9614 has finite HP; a confused player
   auto-swinging for many rounds drops it, stranding the `cast spike`/`trip`
   steps (no target).

## Goal

Rebuild the tutorial as a **long linear chain quest** the newcomer follows one
discrete, hand-held step at a time — teaching **quests** and the **`hint`**
command as the very first thing they do, so `hint` becomes their safety net for
the rest of the game. Every instruction is a single action with a persistent,
re-readable hint. Dewey keeps his warm in-room voice. The effigy stuck-point is
eliminated.

Crucially, do this by **adding the quest primitive the engine is missing**, not by
routing command detection through the behavior tree. The result is a general
capability every future quest can use.

## Non-Goals

- No change to route-2/route-3 onboarding, the Awakening quest (30), Two Roads
  (29), or the Pothole Coulee flow. The handoff is unchanged: the tutorial still
  ends by depositing the player at the Awakening Pool (5200) with quest 30
  granted.
- No change to the existing `command` quest event or the 27 command handlers that
  emit it (see below).

## Architecture

### The missing primitive: a central `command_issued` event

Today the quest engine can only observe a command if that command's handler
explicitly calls `questengine.Notify("command", …)`. Only ~27 "action" verbs do,
and each fires at an **action-success** point ("the cast successfully initiated,"
"a successful reload"). The tutorial's utility commands (`look`, `status`,
`inventory`, `help`, `conditions`, `cooldowns`, `say`, `warcry`, `flee`) have no
such hook.

There is a central chokepoint — `usercommands.TryCommand` — but its only central
quest hook is `room_interact`, which **intercepts** (a match swallows the
command, so `look` would not run). That is correct for hidden-noun discovery,
wrong for a tutorial.

**Fix — add a new, non-intercepting `command_issued` event** fired once from
`TryCommand` for every successfully-dispatched user command, carrying the
alias-resolved `Command` and the lowercased argument as `Noun`. Semantics:
*"the player typed this command."* This is distinct from `command`
(*"the action succeeded"*) — the two are kept separate on purpose, so **no
existing quest is touched and nothing regresses.** (Consolidating the 27 onto one
central hook was rejected: `TryCommand` only knows a command dispatched, not that
its action succeeded, so it would advance action-gated quests like First Shot on
a no-ammo `shoot`.)

Engine change (small, additive):
- `internal/usercommands/usercommands.go` — in `TryCommand`, after a command
  dispatches successfully and was not intercepted, fire
  `Notify("command_issued", EventDetails{UserId, RoomId, Command: alias,
  Noun: strings.ToLower(strings.TrimSpace(rest))}, bridge, bridge)`. Ignore the
  result (non-intercepting — the command has already run).
- `internal/questengine/loader.go` — add `"command_issued": true` to the
  `validEvents` whitelist.
- `EventDetails` already carries `Command`, `Noun`, `RoomId`, `UserId`;
  `matchTriggerFields` already filters on `Command` and `Noun`. No struct or
  matcher changes.

Movement (`north`) is an exit, not a dispatched command, and is detected via
`room_enter` (arrival in the next room), so it does not depend on
`command_issued`.

### The tutorial is then a pure quest — no behavior tree

Every mechanism the tutorial needs already exists in the quest engine once
`command_issued` lands. The six Newcomer-Antechamber room behavior trees are
**deleted**; all logic moves into quest 28's triggers.

| Need | Quest mechanism |
|---|---|
| Detect a lesson command | `event: command_issued` trigger (`command:` + `noun:`) |
| Detect room arrival / greet | `event: room_enter` trigger (`room:`) |
| Dewey speaks in-room, with pacing | `npc_say` action (`mob: 9491`, `lines:` with per-line `delay`) — a real in-room `say`, multi-beat for the combat sequence |
| Advance the chain | `grant:` action (the step token) |
| Block a premature exit | `lock_exits` on the room's step start, `unlock_exits` on its completion |
| Progression teaser (spar) | `train_skill` action (spellcasting) |
| Handoff to the world | `teleport: 5200` + `grant: "30-start"` on the final step |
| Effigy is a live target | see Effigy fix |

`npc_say` (mob 9491) makes Dewey speak for real in the (solo-instance)
antechamber, so his voice is fully preserved without any behavior tree. Exit
gating uses `lock_exits`/`unlock_exits`; **fallback:** if the engine's locked-exit
message reads poorly for a tutorial (no key exists), retain one tiny per-room
behavior-tree branch that intercepts a premature direction command with a Dewey
nudge — the single legitimate behavior-tree job (interception), not a detection
punt. The plan resolves which after checking `lock_exits` UX.

## The Quest — id 28, "Waking to Gaius"

`_datafiles/world/dogmud/quests/28-waking_to_gaius.yaml`. `linear: true`,
`secret: false`. No mechanical reward (quest 30 gives first rewards); a short
completion `playermessage` is optional. Granted when the newcomer arrives in 6258
(a `room_enter room: 6258` trigger, gated so it never re-grants).

**Hint convention:** each step's `hint` names the **single next action** to take
after reaching that step (so the last lesson step in a room has "go north" as its
hint). This is what `hint` shows while the step is current.

| # | Token | Room | Advances on | `hint` (single next action) |
|---|---|---|---|---|
| 1 | `start` | 6258 | (granted on 6258 arrival) | Type `hint` for your next step — or `quests` to see the whole journey. |
| 2 | `hinted` | 6258 | `command_issued hint` | Take stock — type `look`. |
| 3 | `looked` | 6258 | `command_issued look` | Look closer — type `look guide`. |
| 4 | `examined` | 6258 | `command_issued look`/`guide` | The way north is open — type `north`. |
| 5 | `status_prompt` | 6259 | `room_enter 6259` | Read yourself — type `status`. |
| 6 | `statused` | 6259 | `command_issued status` | The way north is open — type `north`. |
| 7 | `carry_prompt` | 6260 | `room_enter 6260` | Pick that up — type `get token`. |
| 8 | `got` | 6260 | `command_issued get`/`token` | See what you carry — type `inventory`. |
| 9 | `invd` | 6260 | `command_issued inventory` | When you forget a command, type `help`. |
| 10 | `helped` | 6260 | `command_issued help` | The way north is open — type `north`. |
| 11 | `speaks_prompt` | 6261 | `room_enter 6261` | Speak up — type `say hello`. |
| 12 | `said` | 6261 | `command_issued say` | Ask me directly — type `ask dewey world`. |
| 13 | `asked` | 6261 | `command_issued ask` | The way north is open — type `north`. |
| 14 | `proving_prompt` | 6262 | `room_enter 6262` | Steel yourself — type `warcry`. |
| 15 | `warcried` | 6262 | `command_issued warcry` | See what's riding on you — type `conditions`. |
| 16 | `saw_conditions` | 6262 | `command_issued conditions` | See your resting moves — type `cooldowns`. |
| 17 | `checked_cooldowns` | 6262 | `command_issued cooldowns` | Now fight — type `attack effigy`. |
| 18 | `attacked` | 6262 | `command_issued attack`/`effigy` | Drive belief in — type `cast spike`. |
| 19 | `cast_spike` | 6262 | `command_issued cast`/`spike` | Take its legs — type `trip`. |
| 20 | `tripped` | 6262 | `command_issued trip` | Not every fight is worth finishing — type `flee`. |
| 21 | `fled` | 6467 | `command_issued flee` (carries player to 6467) | One step left — type `talk dewey`. |
| 22 | `end` | 6467 | `command_issued talk` | (terminal — handoff fires) |

Trigger authoring per step:
- Each advancing trigger also runs Dewey's `npc_say` reaction and `grant`s the
  next token; where a room is entered it also `lock_exits` (block north until that
  room's lessons are done); the room's last lesson step `unlock_exits`.
- Steps 5/7/11/14 (`room_enter`) grant the room's first token, speak Dewey's
  greeting, and lock the north exit.
- Step 20 (`tripped`) speaks Dewey's paced three-beat close via a multi-line
  `npc_say` and fires `train_skill` (spellcasting) for the "you grew by doing"
  teaser.
- Step 22 (`end`, on `talk`) runs the paced farewell `npc_say`, then
  `teleport: 5200` and `grant: "30-start"` — the existing handoff, now a quest
  action. Quest 28 completes as the real world opens.

Loose noun-matching note: an empty `noun:` on a trigger is a wildcard (the matcher
only filters when `noun` is set), so the plain-`look` step (3) also matches
`look guide`. Harmless — the hint directs a plain `look` first, and a player who
jumps ahead simply repeats `look guide` for step 4.

## Effigy fix (mob 9614)

The player is meant to *practice on and flee* the effigy, never kill it.
**Make it effectively unkillable:** raise its vitality far past any plausible
tutorial session (it deals no damage, so no downside, no time pressure),
guaranteeing a live target for `cast spike`/`trip`. The **persistent quest hint**
independently fixes the buried-prompt half of the bug — `hint` re-shows the
current combat step at any time.

## Files

- **Modify (engine):** `internal/usercommands/usercommands.go` (fire
  `command_issued` in `TryCommand`); `internal/questengine/loader.go`
  (whitelist `command_issued`).
- **Create:** `_datafiles/world/dogmud/quests/28-waking_to_gaius.yaml`.
- **Modify:** `_datafiles/world/dogmud/mobs/newcomer_antechamber/9614-straw_effigy.yaml`
  (unkillable vitality).
- **Delete:** `_datafiles/world/dogmud/behaviors/rooms/newcomer_antechamber/`
  `{6258,6259,6260,6261,6262,6467}.yaml` (all logic moves into quest 28). The
  per-mob `behaviors/newcomer_antechamber/9491-dewey.yaml` (archetype-greeting
  suppression) stays. **Fallback exception:** if `lock_exits` UX is poor, keep a
  minimal exit-intercept branch in each of these six files instead of deleting.

## Authoring constraints (carried from the 2026-07-17 build)

- No semicolons and no literal asterisks in any spoken line (`npc_say`) — a
  semicolon truncates the underlying `say`; asterisks render verbatim.
- 80-column wrap. Hints are narrator/2nd-person imperative; Dewey's `npc_say`
  lines stay 1st-person. No hard numbers.

## Verification

1. **ID check:** `python tools/id_inventory.py --type quests` — 28 is free
   (verified: gap 22–28).
2. **Existing-quest regression = none by construction:** the `command` event and
   its 27 handlers are untouched; `command_issued` is a new event no existing
   quest references. Still, boot-test confirms all quests load and a quick harness
   re-check of one command-gated quest (e.g. First Blood `kick`/`trip`) confirms
   `command` still fires.
3. **Boot test** (nuke instance saves first): clean boot,
   `quests.LoadDataFiles() loadedCount` +1, no panic, the six deleted room trees
   no longer referenced.
4. **`command_issued` unit coverage:** a Go test that a dispatched utility command
   (`look`) fires `command_issued` with the right `Command`/`Noun`, and that an
   intercepted command does not double-fire.
5. **Adversarial content playtest (mandatory SOP gate).** Drive a **fresh
   complete newbie** (route 1) through the whole tutorial in a client, reading
   every line. Confirm: `28-start` granted on arrival and `hint`/`quests`
   immediately show the quest and first step (a pre-tutorial character CAN hold
   the quest — the existing 30-start grant proves it, but verify live); every
   Dewey prompt is a single action; `hint` mid-combat re-shows the current step;
   the effigy cannot be killed and `cast spike`/`trip` always have a target;
   premature `north` is blocked with a Dewey nudge; the handoff completes quest
   28, grants 30, and lands the player at 5200 with Cleric Hadwen. Fix and re-run
   until clean, then hand to the user.
