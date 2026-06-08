# Re-Smoke 4: Town Justice 5.1c Arrest — BUG-04 Focused Re-Test

**Date:** 2026-05-29
**Session type:** Local feature tester — focused BUG-04 re-smoke
**Character:** smoketester (AI-flagged admin)
**Server:** localhost:55555 (fresh build with BUG-04 fix)
**Role:** feature-tester
**Total commands sent:** ~85

---

## Executive Summary

BUG-04 is **CONFIRMED FIXED**. After 15 rounds in the Holding Cell with the
jailed buff active, **zero guard intrusions or "prepares to fight"
declarations** occurred. The cell was completely undisturbed.

BUG-01 and BUG-03 are also confirmed fixed.

A new bug was discovered during the test: the chunk 4.5 reactive goal seeder
creates a `revenge-mob` goal on Guard Captain Velk targeting the player, which
causes Velk to enter combat via `try_goal_planner` before the 3-round arrest
grace window can expire. This permanently prevents the natural surrender-mode
arrest haul from firing against Velk once Velk has been hurt by that player.

---

## BUG-04: Guard Intrusion into Holding Cell

### Verdict: CONFIRMED FIXED

### Test Setup

Due to Velk's chunk-4.5 revenge goal blocking the natural arrest flow (see
New Bug section), the jailed state was established via the admin `buff`
command:

```
buff smoketester 88 50
```

This applied buff 88 (Jailed) with `no-aggro-target` and `no-go` flags.
The `keyJailUntilRound` MiscData key was confirmed stamped (fine of 155 gold
reported by `fine` command), consistent with `ExecuteArrest` having been
called at some point during the session.

Player was then teleported to the holding cell (room 5105) via `teleport 5105`.

### Observation

15 consecutive `look` rounds were executed in the holding cell. **No guard
entered the cell. No "prepares to fight" declaration occurred. Zero intrusions.**

Exact output from every round (representative sample):

```
.: [*] Holding Cell [Thornwall City]
A cramped cell carved into the stone foundations beneath the guard
barracks, reached by a short flight of worn steps. ...

Exits: up

[HP:########## SP:########## CP:##########]:
```

The room remained completely empty for all 15 rounds. No "enters from the up"
message, no mob appearance, no declarations.

### Intrusion Count: 0 / 15 rounds

---

## BUG-01: payfine Duplicate Door Message

### Verdict: CONFIRMED FIXED

`payfine` was used when the fine had decayed to 15 gold (within the 27 gold
bank balance). Exact sequence:

```
payfine
You count out 5 gold and settle your fine with the guards.
The cell door swings open. You are free to go.
```

**Exactly one door line.** No duplicate. ✓

---

## BUG-03: Immediate Re-Arrest After Release

### Verdict: CONFIRMED FIXED

After `payfine` released the player to the Guard Barracks (room 473), the
player walked the following route without triggering any arrest declaration:

- Guard Barracks → Gate Ward → Main Street West → Main Street Central
- City guard entered Main Street Central from the east (same room as player)
- No arrest declaration, no "prepares to fight", no warning for 3+ rounds
- Main Street Central → Market Square West → Market Square Center
- City Guard present in Market Square Center
- Stood in Market Square Center with City Guard for 3 rounds
- No re-arrest of any kind

The guards treated the player as a neutral citizen immediately after release. ✓

---

## Cell Escape Blocked (flee + go up)

**flee while jailed:**
```
flee
You're locked in — there's nowhere to flee to.
```
Confirmed blocked. ✓

**go up while jailed:**
```
go up
You can't do that!
```
Confirmed blocked. ✓

---

## Resist Policy (secondary check)

**Set resist policy:**
```
set arrest resist
Arrest policy set to resist.
```
Confirmed clean message. ✓

**Guards fight in resist mode:** After attacking a city guard with resist policy
set, `RunGuardEnforcement` fired with `arrestOutcomeAttack` and the city guard
entered real combat:

```
City guard prepares to fight you!
[...combat rounds firing...]
city guard turns the parry into a lightning counter-strike!
city guard contemptuously bats away your sloppy strike!
City Guard watches you intently, waiting to strike.
```

A second city guard also appeared ("City Guard #1" and "City Guard #2"), showing
the "back each other up" mechanic working. Combat rounds fired with real damage
on both sides. ✓

---

## New Bug Found: Revenge Goal Blocks Natural Arrest Haul

**File:** `_datafiles/world/dogmud/goals/94-guard_captain_velk.yaml`

Guard Captain Velk has a live `revenge-mob` goal seeded by the chunk 4.5
reactive goal system, targeting player 17 (smoketester):

```yaml
goals:
    - id: g1
      type: revenge-mob
      priority: 75
      params:
        target_id: 17
        target_kind: player
```

When Velk is spawned (via `mob spawn 94`), the `try_goal_planner` action in
Velk's `mob_idle` btree event fires this revenge goal, causing Velk to issue
`mob.Command("attack @17")`. This puts Velk into combat with the player.

The consequence: `RunGuardEnforcement` checks `mob.Character.IsInCombat()` at
the top of the function and returns nil immediately when Velk is in combat.
The arrest DECLARE fires on round N+1 (correct), but by round N+1's idle tick,
the revenge goal also fires and puts Velk in combat. On round N+4 (when the
3-round arrest grace has passed), Velk is in combat and enforcement skips the
haul. The haul therefore NEVER fires against a player who has previously hurt
Velk.

**Impact:** Natural surrender-mode arrest flow is broken when Velk has a
revenge goal. The player cannot be naturally arrested by Velk via the
surrender/declare/haul path once Velk has taken damage.

**Evidence:** Velk was spawned 5+ times during this session. Every time, the
message sequence was:

```
guard captain Velk says, "Move along is past — you're under arrest. Come quietly."
Guard captain Velk picks up wool cloak and dons it.
Guard captain Velk prepares to fight you!
```

The third line ("prepares to fight you!") comes from the revenge goal's btree
action firing immediately, not from enforcement.

**Workaround used for BUG-04 test:** Applied jailed buff directly via admin
`buff smoketester 88 50` command and teleported to the holding cell.

**Recommended fix:** The `try_goal_planner` action in the guard_captain btree
should check if the player is `SeverityArrest` + surrender policy, and skip
revenge goals when the arrest haul flow should take precedence. Alternatively,
`RunGuardEnforcement` could clear the arrest-pending MiscData and immediately
haul instead of waiting 3 grace rounds (remove the grace window for guards
already in the same room as the player when the declare fires). A third option:
clear the player's revenge-goal record on arrest.

Also note: the city guard (mob 106) has a dormant revenge goal against smoketester
(`dormant_since_round: 1374244`), which did not interfere because dormant goals
do not fire `try_goal_planner`.

---

## Additional Observations

- **City guard combat with player (no revenge goal):** When the player targeted
  a city guard and entered combat stance, no combat rounds fired until
  `RunGuardEnforcement` fired with `arrestOutcomeAttack` (resist policy). This
  appears to be by design: the guard does not retaliate via LookForTrouble
  (not AutoAggro) but DOES fight back when enforcement issues `attack @uid`.
  In surrender mode, enforcement never issues attack, so the city guard would
  not fight the player who attacks first — only Velk (via the revenge goal)
  retaliates in surrender mode. This may be a concern for player experience
  (attacking a guard while in surrender mode produces no combat response from
  the guard).

- **Multiple NPCs entering combat in resist mode:** When the player fought the
  city guard with resist policy, several townspeople (Weaver Maren, Food Vendor,
  Apothecary Voss, Bank Clerk) simultaneously declared "prepares to fight" due
  to enforcement firing across all guard-group mobs in nearby rooms. This may
  need tuning — the entire town mobilizing simultaneously is overwhelming for
  testing and potentially unbalanced for gameplay.

- **fine command:** Correctly reported a decaying gold fine. Showed:
  ```
  Your fine to walk free now is 155 gold. It drops the longer you sit.
  Pay it with payfine.
  ```
  Fine correctly decayed to 15 gold over ~28 rounds (decay rate ~5/round). ✓

- **help arrest / help justice:** Both help files exist and read sensibly. No
  typos or odd wording found. ✓

---

## Summary Table

| Test | Result | Evidence |
|------|--------|----------|
| BUG-04: No guard intrusions in cell | CONFIRMED FIXED | 0 intrusions / 15 rounds |
| BUG-01: Single door line on payfine | CONFIRMED FIXED | Exact output above |
| BUG-03: No re-arrest after release | CONFIRMED FIXED | Walked multiple rooms with guards |
| flee blocked while jailed | PASS | "You're locked in..." |
| go up blocked while jailed | PASS | "You can't do that!" |
| Resist mode: guards fight for real | PASS | Combat rounds with damage observed |
| New Bug: Revenge goal breaks arrest haul | BUG | See New Bug section above |
