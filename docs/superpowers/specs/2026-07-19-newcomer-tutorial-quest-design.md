# Newcomer Tutorial as a Guided Chain Quest — Design

**Date:** 2026-07-19
**Status:** Approved (design), pending spec review
**Relates to:** `project_newcomer_tutorial_redesign` (the 2026-07-17 guide-led Dewey
tutorial this reworks), `project_newbie_onboarding_feedback_backlog`

## Problem

The route-1 "complete newbie" tutorial (Newcomer Antechamber, rooms 6258–6262 +
6467, guided by Dewey / mob 9491) works as a per-room behavior-tree state machine.
Two defects surfaced in Malia's 2026-07-18 playtest:

1. **Compound instructions.** Several of Dewey's spoken lines bundle multiple
   actions into one breath — e.g. room 6261's greeting ("say hello. Then ask
   dewey world") and every movement-block nudge ("say hello, then ask dewey
   world. Then north"). A brand-new player can't parse a three-step instruction.
2. **No persistent, re-readable instruction.** Dewey speaks each prompt once via
   `mob_say`. If it scrolls past — or the player kills the straw effigy and the
   "cast spike" prompt is buried — there is nothing to type to get it back. Malia
   **got stuck after killing the effigy** because she never saw the cast prompt.
3. **The effigy is killable.** Straw effigy 9614 has finite HP (vitality
   training:1000). A confused player auto-swinging for many rounds can drop it,
   which strands the `cast spike`/`trip` steps (no target).

## Goal

Rebuild the tutorial as a **long linear chain quest** the newcomer follows one
discrete, hand-held step at a time. This:
- Teaches **quests** and the **`hint`** command as the very first thing a new
  player does, so the always-available `hint` becomes their safety net for the
  rest of the game.
- Makes every instruction a single action with a persistent, re-readable hint.
- Keeps Dewey's warm in-room voice (the 2026-07-17 redesign's whole point).
- Eliminates the effigy stuck-point.

## Non-Goals

- No change to the route-2/route-3 onboarding, the Awakening quest (30), Two
  Roads (29), or the Pothole Coulee flow. The handoff target is unchanged: the
  tutorial still ends by depositing the player at the Awakening Pool (5200) with
  quest 30 granted.
- No global command instrumentation (see Architecture).
- No new engine code. Everything uses existing behavior-tree actions/checks and
  the existing quest engine.

## Architecture — quest spine + behavior-tree detector (hybrid, and why)

**Hard constraint:** the quest engine only detects a `command` event if that
command is individually instrumented to call `questengine.Notify("command", …)`.
Only "verb" commands are today (cast, trip, craft, consider, reload, shoot,
track, forage, drink, buy…). The tutorial's utility commands — `look`, `status`,
`inventory`, `help`, `conditions`, `cooldowns`, `say`, `warcry`, `flee`, `hint` —
are **not** instrumented. The **behavior tree** (`room_command`) is the only
mechanism that already sees every in-room command. Instrumenting a dozen
commands globally (firing quest events for every player on every `look`) was
considered and rejected as too much surface for marginal gain.

Therefore:

- **The quest (id 28, "Waking to Gaius") is the spine and single source of
  truth.** It is a pure step container: each step defines `id`, `description`,
  and `hint`. It carries **no triggers** — every step is advanced externally by
  a behavior-tree `grant_quest` action. The player follows it via `hint` /
  `quests` throughout.
- **The behavior trees remain the detector, Dewey's voice, and the movement
  gate.** Each room tree is reworked to: (a) gate branches on
  `player_has_quest` / `player_missing_quest` of the 28-tokens instead of its own
  `state_equals` flags (so the quest tokens are the only progression state);
  (b) speak Dewey's de-compounded line via `mob_say`; (c) `grant_quest` the next
  28-token on each step completion; (d) block premature exits until the room's
  last token is held.

This is proven feasible: the existing 6467 tree already calls `grant_quest:
30-start` on the tutorial character, so a pre-tutorial player can hold quest
progress; `grant_quest`, `player_has_quest`, and `player_missing_quest` are all
existing behavior-tree registry entries.

### Division of labor per step

| Step kind | Detected by | Advances via |
|---|---|---|
| Command lessons (`look`, `status`, `warcry`, `cast spike`, …) | btree `room_command` (`command_matches` + `command_rest_contains` for args) | btree `grant_quest` |
| Room-arrival prompts (entering the next room) | btree `room_enter` (already fires Dewey's greeting) | btree `grant_quest` |
| Quest grant (tutorial start) | btree `room_enter` on 6258, gated `player_missing_quest 28-start` + `28-end` | btree `grant_quest 28-start` |
| Final handoff | btree `room_command talk` in 6467 | btree `grant_quest 28-end` (+ existing 30-start grant + move to 5200) |

The quest file itself has no `triggers:` block — a deliberate inversion of the
usual pattern, valid because all detection lives in the behavior trees.

## The Quest — id 28, "Waking to Gaius"

`_datafiles/world/dogmud/quests/28-waking_to_gaius.yaml`. `linear: true`,
`secret: false`. No mechanical reward (the Awakening quest 30 gives first
rewards); an optional short `playermessage` on completion is fine.

**Hint convention:** each step's `hint` names the **single next action to take
after reaching that step** (so the last lesson step in a room has "go north" as
its hint). This is what `hint` shows while that step is current.

| # | Token id | Room | Reached by | `hint` (next single action) |
|---|---|---|---|---|
| 1 | `start` | 6258 | 6258 room_enter (grant) | Type `hint` to see your next step — or `quests` to see the whole journey. |
| 2 | `hinted` | 6258 | `hint` | Take stock of where you stand — type `look`. |
| 3 | `looked` | 6258 | `look` | Look closer at one thing — type `look guide`. |
| 4 | `examined` | 6258 | `look guide` | The way north is open — type `north`. |
| 5 | `status_prompt` | 6259 | 6259 room_enter | Read yourself — type `status`. |
| 6 | `statused` | 6259 | `status` | The way north is open — type `north`. |
| 7 | `carry_prompt` | 6260 | 6260 room_enter | Pick that up — type `get token`. |
| 8 | `got` | 6260 | `get token` | See what you carry — type `inventory`. |
| 9 | `invd` | 6260 | `inventory` | When you forget a command, type `help`. |
| 10 | `helped` | 6260 | `help` | The way north is open — type `north`. |
| 11 | `speaks_prompt` | 6261 | 6261 room_enter | Speak up — type `say hello`. |
| 12 | `said` | 6261 | `say hello` | Ask me directly — type `ask dewey world`. |
| 13 | `asked` | 6261 | `ask dewey …` | The way north is open — type `north`. |
| 14 | `proving_prompt` | 6262 | 6262 room_enter | Steel yourself — type `warcry`. |
| 15 | `warcried` | 6262 | `warcry` | See what's riding on you — type `conditions`. |
| 16 | `saw_conditions` | 6262 | `conditions` | See your resting moves — type `cooldowns`. |
| 17 | `checked_cooldowns` | 6262 | `cooldowns` | Now fight — type `attack effigy`. |
| 18 | `attacked` | 6262 | `attack effigy` | Drive belief into it — type `cast spike`. |
| 19 | `cast_spike` | 6262 | `cast spike` | Take its legs — type `trip`. |
| 20 | `tripped` | 6262 | `trip` | Not every fight is worth finishing — type `flee`. |
| 21 | `fled` | 6467 | `flee` (carries player to 6467) | One step left — type `talk dewey`. |
| 22 | `end` | 6467 | `talk dewey` | (terminal — quest complete; handoff fires) |

Notes:
- Steps 5/7/11/14 are the room-arrival prompts; the btree grants them in the
  same `room_enter` branch that speaks Dewey's greeting for that room.
- Step ids for room 6262 reuse the current behavior-tree flag names
  (`warcried`, `saw_conditions`, `checked_cooldowns`, `attacked`, `cast_spike`,
  `tripped`) to minimise churn — they simply become quest tokens.
- `end` completing the quest coincides with the existing 6467 handoff (move to
  5200 + grant 30-start), so the tutorial quest reads "complete" exactly as the
  real world opens up.

## Behavior-tree rework (6258, 6259, 6260, 6261, 6262, 6467)

For each room tree, mechanical transform:

1. **Grant the tutorial quest.** 6258's `room_enter` branch adds
   `grant_quest 28-start`, gated `player_missing_quest 28-start` AND
   `player_missing_quest 28-end` (idempotent; never re-grants).
2. **Replace flag state with quest tokens.** Every `check: state_equals key:X
   value:"1"` becomes `check: player_has_quest quest:"28-X"`; every
   `state_equals key:X value:""` becomes `check: player_missing_quest
   quest:"28-X"`. Every `do: set_state key:X value:"1"` becomes
   `do: grant_quest quest:"28-X"`.
3. **De-compound every `mob_say`.** Each spoken line names exactly one next
   action. Greetings (6260, 6261) are trimmed to the first step only.
   Movement-block nudges stop listing all remaining steps; they point at the
   current step and reinforce the tool: e.g. *"Not through yet — type `hint` if
   you've lost the thread."*
4. **Movement gate unchanged in spirit:** the block branch intercepts a
   direction command while the room's last token is missing
   (`player_missing_quest`), releasing once it's held.
5. **Room-arrival grants:** 6259/6260/6261/6262 `room_enter` branches
   `grant_quest` their room's first token alongside the greeting.
6. **6467 `talk dewey`:** add `grant_quest 28-end` to the existing handoff
   sequence (before/with the 30-start grant).

Authoring constraints carried over (from the 2026-07-17 build notes): **no
semicolons and no literal asterisks in any `mob_say`** (semicolon truncates the
`say`; asterisks render verbatim). 80-column wrap. Hints are narrator/2nd-person
imperative; Dewey's `mob_say` lines stay 1st-person. No hard numbers.

## Effigy fix (mob 9614)

Root cause: the effigy is killable, and killing it strands `cast spike`/`trip`.
The player is meant to *practice on and then flee* the effigy — never to kill it.

- **Make it effectively unkillable:** raise its vitality far past any plausible
  tutorial session (it already deals no damage, so there is no downside and no
  time pressure). This guarantees a live target for the `cast spike` and `trip`
  steps.
- The **persistent quest hint** ("Drive belief into it — type `cast spike`")
  removes the "buried prompt" half of the bug independently: `hint` re-shows the
  current combat step at any time.

Together these eliminate the stuck-point without new engine work. (An instant
respawn-on-death was considered as a backstop but is unnecessary once the effigy
can't die during the lesson.)

## Verification

1. **ID check** before authoring: `python tools/id_inventory.py --type quests`
   confirms 28 is free (verified: gap 22–28).
2. **Boot test** (nuke instance saves first): server boots clean,
   `quests.LoadDataFiles() loadedCount` increases by 1, no panic, all six room
   trees load.
3. **Adversarial content playtest (mandatory SOP gate).** Drive a **fresh
   "complete newbie"** (route 1) through the entire tutorial in a client with a
   critical mandate. Confirm, reading every line:
   - `28-start` is granted on arrival and `hint`/`quests` immediately show the
     tutorial quest and its first step (the pre-tutorial character CAN hold the
     quest — verify live, not just in theory).
   - Every Dewey prompt is a **single** action; no compound instructions remain.
   - `hint` at any point re-shows the current single step (test mid-combat).
   - The effigy **cannot be killed** across the lesson; `cast spike`/`trip`
     always have a target; the player flees successfully to 6467.
   - The handoff completes quest 28, grants quest 30, and lands the player at
     the Awakening Pool (5200) with Cleric Hadwen.
   Fix anything found; re-run until clean. Only then hand to the user.

## Files

- **Create:** `_datafiles/world/dogmud/quests/28-waking_to_gaius.yaml`
- **Modify:** `_datafiles/world/dogmud/behaviors/rooms/newcomer_antechamber/`
  `{6258,6259,6260,6261,6262,6467}.yaml`
- **Modify:** `_datafiles/world/dogmud/mobs/newcomer_antechamber/9614-straw_effigy.yaml`
  (raise vitality to effectively-unkillable)
- **No engine/Go changes.**
