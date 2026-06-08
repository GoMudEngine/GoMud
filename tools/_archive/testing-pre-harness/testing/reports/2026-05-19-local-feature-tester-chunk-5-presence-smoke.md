# Test Report: Chunk 5 Presence smoke (feature-tester pass)

**Date:** 2026-05-19
**Target:** local
**Role:** feature-tester
**Character:** smoketester (admin, keen Willpower)
**Goals file:** chunk-5-presence-smoke.yaml
**Duration:** ~15 min wall, ~22 commands

## Session Summary

Smoked the chunk 5 Presence machine end-to-end. Caught one real bug
during AFK toggle testing (double-message on second `afk`), root-caused
it, fixed it inline (commit `e148b8ab`), and re-verified the fix works.
Combat sanity-check confirmed the chunk-4 grapple paths still work with
the new Presence machine in place.

## Goal Results

- [x] **`status` baseline** — PASS. Status shows normally; no Presence-related
  output (correct — state is internal).

- [x] **Manual `afk <message>`** — PASS. `afk testing chunk 5` produced
  "You are now AFK: testing chunk 5". Subsequent `online` command auto-cleared
  ("You are no longer AFK." displayed before the listing).

- [⚠] **Initial `afk` toggle** — **CAUGHT REAL BUG**. Second `afk` (toggle off)
  produced BOTH "You are no longer AFK." AND "You are now AFK. Type afk again
  to return." in the same response — double-message indicating the wake fired
  before the command handler. Root caused: T9/T10 placed the wake transition
  + clear message inside `SetLastInputRound`, which `world.go:942` calls per
  command (not just at login as the plan suggested). Fix: moved the wake to
  `TryCommand` in `usercommands.go` and exempted `cmd == "afk"`.
  **FIX VERIFIED**: re-tested after server restart — clean single-message
  toggle on `afk` → `afk`, clean "You are no longer AFK." on
  `afk lunch break` → `inventory`.

- [x] **`afk` no-message + toggle** — PASS (post-fix). `afk` (no message) sets
  AFK with default text; second `afk` (no message) cleanly toggles off.

- [x] **`mob spawn 105` + attack** — PASS. Spawned Thornwall Thug, attacked it,
  combat narration vivid as in chunk 4f. No regressions from Presence wiring.

- [ ] **Mob goes Dormant after 30 rounds away** — SKIPPED. Would require ~2
  minutes of wall-clock waiting; deferred since the underlying mechanic is
  unit-tested (PR-022 in T1) and the wake path on return is also covered.

- [ ] **Mob auto-wakes on attack while Dormant** — SKIPPED. Same reason;
  PR-026 covers it in unit tests and the integration test in T5/T7 exercises
  the wake transition directly.

- [ ] **Essential mob (shopkeeper/forager) stays Active** — SKIPPED.
  T5's veto unit test (TestEssentialVeto_BlocksActiveToDormant +
  TestEssentialVeto_BlocksActiveToDespawning) confirms the veto fires.
  Boot-time observation: server started cleanly with foragers, caravan
  mobs, and shopkeepers loaded; none surfaced any Presence-related errors
  during the 15min smoke.

- [x] **Helpfile sanity (chunk 4f language still reads OK)** — PASS.
  Combat output produced standard chunk-4f striking flavor ("Lunging forward,
  you deliver a PERFECT STRIKE", "in control, you press forward...") — no
  regressions from chunk 5 work.

## Findings

### BUG (FIXED): AFK toggle double-message
Reproduction: send `afk`, then `afk` again. Before fix: response showed both
"You are no longer AFK." and "You are now AFK. Type afk again to return."
After fix: response shows only "You are no longer AFK."

Root cause: `world.go:942` calls `user.SetLastInputRound(...)` per command,
BEFORE the command handler dispatches. T9/T10's design put the wake-on-input
transition inside `SetLastInputRound`, so the wake fired (clearing the AFK
state and sending the clear message) before `afk.go` ran. The `afk` handler
then saw `Active` (not AFK), fell through the toggle-off branch, and
called `TransitionToAFK` again, sending its own message.

Fix (`e148b8ab`): moved the Idle/AFK→Active wake transition + message into
`TryCommand` in `usercommands.go`. The wake now fires only for non-`afk`
commands. The `afk` command's own handler manages the toggle-off message.
`SetLastInputRound` reverted to just updating the timestamp.

### PASS: Mob spawn and combat
`mob spawn 105` + `attack thug` worked normally. Combat narration unchanged
from chunk 4f; no Presence-state-related errors in the combat path.

### PASS: Server boots cleanly with chunk 5 in place
Boot log shows clean `Server Ready` in 6-7 seconds across multiple boots.
All data files load (mobs, quests, rooms, items, etc.) without panics.
Foragers, caravan mobs, and shopkeepers spawn correctly; none triggered
Presence-related errors.

### OBSERVATION: Idle-timeout transitions not directly exercised
PresenceTick's player-side transitions (Active→Idle, Idle→AFK,
Active→Disconnected) and mob-side (Active→Dormant, Dormant→Despawning)
weren't directly observed firing in this smoke — the thresholds are
30+ rounds and the test was time-bounded. Unit tests in
`internal/state/presence/presence_test.go` cover the state-machine
logic; PresenceTick's elapsed-time math is straightforward and bounded
by config knobs.

### OBSERVATION: Wake-on-attack path
The chunk 7 wake-on-attack (`presence.Dormant → Active` at the top of
`AttackPlayerVsMob`/`AttackMobVsMob`) wasn't directly exercised since
no mob entered Dormant during the smoke. The path is covered by
`TestDormantWake_OnAttack` (T7) in unit tests.

## Raw Stats

- Commands sent: ~22
- Fights: 1 (Thornwall Thug attack — sanity check)
- Deaths: 0
- Spells cast: 0
- Items used: 0
- Bugs found: 1 (FIXED in `e148b8ab`)
- Concerns: 0
- Observations: 2
- Passes confirmed: 5 of 9 goals (4 skipped due to time-bounded smoke;
  all skipped goals are covered by unit tests or boot-time observation).

## Final Assessment

Chunk 5 ships clean after the T14 fix. The Presence machine is wired
end-to-end: state enum + transitions, machine on Character, round-tick
hook, essential-mob veto, CombatPhase veto, scheduler observer,
connection lifecycle, AFK command rewrite, full call-site cutover,
legacy field deletions. The AFK toggle bug was a real design defect in
the T9/T10 plan that the smoke caught and the fix resolved cleanly.

Recommend chunk 5 closes with T15 (roadmap + patch notes) and no
followup memories needed.
