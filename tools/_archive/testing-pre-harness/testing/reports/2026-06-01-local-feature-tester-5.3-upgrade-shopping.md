# Feature Test Report: 5.3 Upgrade Shopping Observation

**Date:** 2026-06-01
**Target:** local (localhost:55555)
**Role:** feature-tester
**Character:** smoketester
**Goals file:** 5.3-upgrade-shopping-observation.yaml
**Duration:** ~40 minutes
**Command count:** ~80

---

## Session Summary

Tested the 5.3 upgrade-gear goal system. The smoketester character is
admin-flagged (confirmed via `goal` command returning full usage text).
Used admin path throughout.

The session was complicated by a bounty hunter combat encounter immediately
after connecting, which required extended recovery at the Thornwall temple
before testing could begin.

Teleported to room 4110 (Constabulary, Stillwater) to observe Constable Drunn
(mob 335, guard_captain archetype). Confirmed via `goal list 335` that Drunn
had **no goals** despite having `seeded_from_archetype: true` in his goals
file. Manually added `upgrade-gear` and `survival` goals via admin commands.
The `survival` goal did not persist (likely a type-registration issue or
conflict during save). Only `upgrade-gear` (g3) remained.

Gave Drunn 22 gold (from bank withdrawal) to bring his total to 42 gold.
Re-added the upgrade-gear goal with `reserve=0` override to make budget =
42 > 0. Observed Drunn for 30+ rounds with `goal current 335` confirming
`upgrade-gear` selected, but Drunn never walked toward the blacksmith (room
4106, 3 hops away).

---

## Goal Results

- [x] **PASS** — Admin-flagged confirmed. Used `goal` command; received full
  usage text. Mode: **ADMIN**.

- [~] **BLOCKED** — ADMIN PATH: `goal list 335` showed `Constable Drunn
  (335) has no goals` even though `seeded_from_archetype: true` in the on-
  disk goals file. The archetype-seed only runs for truly fresh mobs (no
  file at all); an existing file with `goals: []` is treated as intentional
  empty. Goals must be seeded manually for existing mobs. Added `upgrade-gear
  priority=30 reserve=0` via `goal add 335 upgrade-gear 30 reserve=0`.
  Confirmed `goal current 335` shows `g3 upgrade-gear priority=30`.

- [~] **BLOCKED** — ORGANIC / SUSTAINED OBSERVATION: Over 30+ idle rounds,
  Drunn did NOT move toward the blacksmith or any shop. `locate *Drunn*`
  confirmed he remained at room 4110 throughout. No shopping-related movement,
  buy message, or equip/gearup flavor observed.

- [~] **BLOCKED** — No verbatim shopping signal lines captured. Representative
  lines from observation:
  ```
  goal current 335
  Current goal for constable Drunn (mob 335): g3 upgrade-gear priority=30
  .
  ```
  ```
  constable Drunn stands at the door of the constabulary, arms folded,
  watching the square
  ```
  ```
  constable Drunn drums two fingers on the head of his club, eyes on the
  street
  ```
  ```
  constable Drunn scans the waterfront road with a long, unhurried sweep
  of his gaze
  ```
  These are all idle emotes, no pathto or shopping behavior.

- [x] **PASS** — No gold-drain re-buy loop observed. Drunn's gold did not
  drain; no buy/re-buy cycling detected. Safety properties hold (by absence:
  the drive never activated at all, so no safety violations occurred).

---

## Findings

### OBSERVATION — Empty goals file for existing mob (seeding gap)

`_datafiles/world/dogmud/goals/335-constable_drunn.yaml` has
`seeded_from_archetype: true` but `goals: []`. The `seedFromArchetype`
function in `internal/goals/store.go:422-442` checks `mg.SeededFromArchetype`
as an idempotency sentinel and skips if already true. Since Drunn's file was
created with the flag set (likely during a prior boot or admin operation), the
default goals from the `guard_captain` archetype (`upgrade-gear` priority 30,
`survival` priority 80) were never populated.

This is a deployment gap: if a mob template's archetype is changed AFTER its
goals file is first written (or if the file is pre-created with the sentinel
flag), the mob never gets its archetype-default goals without manual seeding
or `goal clear`.

### OBSERVATION — goal scores CtxMod column is a placeholder (always 1.00)

`goal scores 335` always shows `CtxMod: 1.00` regardless of actual mob state.
Confirmed by reading `internal/usercommands/admin.goal.go:348`: the code
hardcodes `ctxMod := 1.0` and includes a comment "weights and CtxMod are
displayed as defaults at 4.2 ship — the goal-type registry is empty until
4.3." This comment is stale; the registry IS populated at 4.3, but the
display was never updated to call the live ContextScore function. As a result,
the admin `goal scores` command cannot be used to verify whether a mob is
actively scoring its upgrade-gear goal as "active" vs "floor."

### CONCERN — Upgrade-gear planner not visibly firing despite goal being CURRENT

With `upgrade-gear` selected as CURRENT and `reserve=0 / gold=42 / budget=42`,
Drunn should:
1. Call `scanZoneUpgrades` → find iron short sword (price 15, delta ~20.2)
2. Issue `pathto 4106` (blacksmith, 3 hops)
3. Walk to the shop over 3 rounds

After 30+ rounds this did NOT happen. Code analysis reveals a likely root
cause:

**`TryMobBehavior` returns `result == Success` (line 203 of helpers.go).**
The guard_captain archetype's `mob_idle` → `try_goal_planner` node returns
`Running` (StatusRunning maps to btree Running via `translatePlannerStatus`).
The top-level SelectorNode propagates `Running`. `TryMobBehavior` returns
`Running == Success` → `false`. This causes the legacy idle code in
`HandleIdleMobs` to also run (it fires when `TryMobBehavior` returns false).

Consequence: both `mob.Command("pathto 4106")` (from planner) and
`mob.Command("emote ...")` (from legacy idle) are queued each tick. While
these should not cancel each other, the path-walking code in
`NewRound_IdleMobs.go` skips `MobIdle` emission while a path is active (line
163 `continue`). This means `try_goal_planner` should NOT be re-issuing
`pathto` while the mob walks, and the mob SHOULD advance. The mechanism is
correct in theory.

Alternative cause: if the `reserve=0` param is lost during YAML round-trip
(stored as `int` but read back as a type not matched by `goalParamIntOr`'s
`case int` / `case int64` switch), the reserve would fall back to the config
default of 50, making `budget = 42 - 50 = -8 <= 0`. In this case
`scanZoneUpgrades` is not called with `onlyAffordable=true` and the planner
checks path 3 (save-up) instead. This is a plausible explanation for why
no `pathto` is issued.

To distinguish: check `goal show 335 g3` after a server restart (disk
round-trip). If `reserve: 0` persists as integer, the YAML type is fine.

### OBSERVATION — Survival goal not persisting

When `goal add 335 survival 80` was issued, it printed `Added goal g2
(type=survival, priority=80)`. After reading the file:
`_datafiles/world/dogmud/goals/335-constable_drunn.yaml` shows
`next_goal_id: 3` (confirming g2 was assigned) but only g3 (upgrade-gear)
is present in the goals list. g2 (survival) is absent. The file was saved
correctly for g3 but g2 was either removed by a conflict resolution or the
save for g2 failed. Requires investigation.

### BUG (Minor) — Sleep command fails outdoors and in non-standard locations

`sleep` command returns:
```
Something prevented you from sleeping: failed to add buff. target:
"smoketester" buffId: 15.
```
This is the pre-existing BUG-4 from the 5.1b smoke notes. Confirmed still
present. Not a 5.3 regression.

---

## Raw Stats

- Rounds observed (Drunn idle): ~35
- Shopping moves observed: 0
- Buy events observed: 0
- Gearup/equip flavor observed: 0
- Goal current reads: 4 (all showed `upgrade-gear`)
- Gold given to Drunn: 22 (player withdrew from Counting House)
- Drunn final room: 4110 (Constabulary) — never moved
- Admin seeding required: yes (goals file existed but empty)
- Reserve override used: `reserve=0` goal param
- Survival goal added: yes (goal add); did not persist in file
