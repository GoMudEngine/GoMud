# Guard Combat Capability — Design

**Date:** 2026-05-29 • **Size:** M • **Branch:** `feature/guard-combat-capability`
(off master, which has Town Justice 5.1a + 5.1b)
**Precursor to:** Town Justice 5.1c (arrest-resist path needs fightable guards)
**Depends on:** mob archetype system, behavior-tree archetypes, stat-distribution
(`internal/mobs/mobs.go` statpool roller), dialogue/quest engine.

---

## Overview

All Thornwall guards are `behavior_archetype: noncombat_questgiver` — an
archetype with **no combat behavior tree** (only `mob_idle`/`player_enter`/
`player_ask`/`player_give` nodes). So when Town Justice 5.1a issues
`attack @uid`, or a player attacks a guard, combat *state* is set but no rounds
ever fire: the guard never swings, never wins, never takes a turn. This is the
root cause of the 5.1b smoke's BUG-2 (passive guards) and BUG-3 (combat stall).
5.1a's lethal "attack on sight" has therefore **never actually functioned**, and
5.1c's arrest-resist path (player fights back → `SeverityAttack` → combat)
depends on guards that can fight.

This chunk makes the three active Thornwall enforcers combat-capable and a
credible deterrent, while preserving Captain Velk's quest role.

---

## Locked decisions

| Decision | Value |
|----------|-------|
| Scope | 3 mobs: 94 guard_captain_velk, 106 city_guard, 92 city_gate_guard. **Out:** 335 constable_drunn (tangled with deferred `stillwater_guards` work). |
| Rank-and-file combat btree | `tank_taunter` (existing archetype) on 106 + 92 |
| Captain combat btree | NEW hybrid `guard_captain` archetype on 94 (combat + questgiver) |
| Stat distribution archetype | `tank` on all 3 (matches the tank_taunter btree — Cha for taunt, Vit for HP) |
| Toughness | statpool: rank-and-file → **240**, captain → **300** ("serious deterrent") |
| Baseline | unchanged — human (speciesid 1) racial base is 100/stat; statpool adds Training on top |

---

## Components

### 1. Rank-and-file archetype swap (106 city_guard, 92 city_gate_guard)

Both are `behavior_archetype: noncombat_questgiver` today and carry **no quests
or dialogue**, so nothing to preserve. Straight field changes per mob YAML:

- `behavior_archetype: noncombat_questgiver` → **`tank_taunter`**
- `archetype: fighting` → **`tank`** (stat-distribution weighting)
- `statpool: 60` (106) / `75` (92) → **`240`**

`tank_taunter` (existing, `_datafiles/world/dogmud/behaviors/archetypes/tank_taunter.yaml`)
gives the full combat cascade (bash/trip/grapple/kick + taunt + self-buff),
panic-flee at low HP, and `packmate_hurt` → engage the attacker — so guards back
each other up, which reads exactly right for a city watch.

These mobs keep `schedule_id` (106 has `thornwall_city_guard_dayshift`) and
`idlecommands`. **Schedule execution is btree-independent** (consumed in
`internal/hooks/NewRound_IdleMobs_schedule.go` / `internal/mobs/schedule.go`, not
gated on `behavior_archetype`) — confirm at build that patrols/idle still run
under the combat archetype (the `mob_idle` → `try_goal_planner` node is present
in tank_taunter, same as questgiver).

### 2. New hybrid `guard_captain` archetype (94 Velk)

Velk is a **live quest NPC**: quests `10-the_drowning_posts_debt` and
`14-the_undertow` deliver items to `mob: 94` via `item_give` and instruct the
player to "Speak with Velk to close the case." He must fight **and** keep
serving those quests.

Author `_datafiles/world/dogmud/behaviors/archetypes/guard_captain.yaml` =
**tank_taunter's combat nodes** (the `mob_hurt`→flee, `packmate_hurt`→attack,
`mob_combat_round` cascade) **above** **noncombat_questgiver's non-combat nodes**
(`mob_idle`→`try_goal_planner`, `player_give`→return-item-or-quest-handling,
and the `player_ask` dialogue fallthrough). Selector order = combat reflexes
first (fire when engaged), questgiver/idle behavior when not in combat.

Velk YAML changes:
- `behavior_archetype: noncombat_questgiver` → **`guard_captain`**
- `archetype: fighting` → **`tank`**
- `statpool: 120` → **`300`**

**Build-time verification (flagged, not blocking):** confirm how the
dialogue/quest dispatcher fires relative to the btree —
- whether `player_ask` dialogue uses the documented "btree returns Failure for
  ask → dispatcher falls through to dialogue patterns" path (questgiver
  archetype comment), and the hybrid preserves that fallthrough;
- whether quest `item_give` (the mechanism quests 10/14 use to hand Velk the
  notice/ledger) is handled by the **quest engine regardless of archetype**
  (very likely) — if so, the hybrid only needs the `player_ask` dialogue
  fallthrough, not special give handling.
Read the quest-engine `item_give` path + the dialogue dispatcher before
finalizing the hybrid tree; adjust the non-combat nodes to match what actually
must be preserved. If `item_give`/dialogue turn out to be btree-coupled in a way
the hybrid can't cleanly preserve, STOP and report.

### 3. Toughness (stat-bump)

Final stat = `Racial (species base) + Training + Mods` (`internal/stats/stats.go`
`Recalculate`). Human speciesid 1 has `base: 100` on every stat, so guards
already floor at 100. `statpool` adds Training, distributed by the `archetype`
field's weighting (`internal/mobs/mobs.go` ~line 447): `tank` favors
Cha 25% / Vit 20% / Str 15% / Dex 15% / Wil 15% / Per 10%.

Expected rough outcomes (random distribution):
- **Rank-and-file (pool 240):** Cha ~160, Vit ~148, Str/Dex/Wil ~136, Per ~124.
  Comfortably above a fresh player (100); a credible threat, beatable by a
  capable outlaw.
- **Captain Velk (pool 300):** proportionally higher (~Cha 175, Vit 160, others
  ~145), boss-tier.

Because the roll is random, the implementer must **boot-and-inspect**: spawn each
guard on a clean server, read rolled stats via an admin command (`mob info` /
`stat` inspection), and adjust the pool if the spread materially over/undershoots
the "serious deterrent" target. Tuning the pool number is in scope; the
archetype weighting is fixed.

### 4. Equipment (no change — verify only)

All three guards are already fully equipped with real combat gear (106/92: iron
spear 10019 + shield + armor; 94: steel longsword 10015 + shield + armor). No
equipment changes needed. Build-time: confirm the equipped weapons resolve and
the guards actually swing them (the boot-and-inspect combat check covers this).

---

## Data flow

### Player attacks a guard (BUG-3 fix)
Player `attack city guard` → guard enters combat → next `mob_combat_round` the
tank_taunter tree now fires (bash/trip/attack cascade) → guard swings back,
rounds advance, fight resolves. No more stall.

### Guard enforces the law (5.1a / 5.1c)
5.1a `RunGuardEnforcement` issues `attack @uid` → guard's combat tree drives the
fight to completion (previously a no-op print). For 5.1c, a `resist`-policy
player who fights back now faces a guard that actually fights — the resist path
becomes real.

---

## Testing

- **Boot smoke (SOP instance-wipe):** server boots clean; the new
  `guard_captain` archetype + the three modified guard YAMLs load with no panic;
  `Server Ready`.
- **Combat behavior (boot-and-inspect, manual):** spawn each guard; (a) read
  rolled stats and confirm they hit the "serious deterrent" band (tune pool if
  not); (b) attack each guard and confirm combat rounds actually fire and the
  guard swings back (BUG-3 gone); (c) confirm rank-and-file flee at low HP and
  engage on packmate-hurt.
- **Velk quest preservation (manual):** confirm `ask velk` dialogue still works
  and a quest `item_give` to Velk (quest 10 or 14 flow) still completes under
  the hybrid archetype — the keystone risk.
- **Schedule preservation:** confirm 106 still runs its dayshift schedule /
  patrol under tank_taunter (idle `try_goal_planner` node present).
- **Archetype unit coverage:** if the btree loader has archetype-load tests
  (`internal/behaviortree/archetype_*_test.go`), add a load+shape test for
  `guard_captain` mirroring the existing `noncombat`/`tank_taunter` tests.

Functional in-game smoke (commit a crime → guard fights → win/lose) deferred to
the user per the manual-smoke convention; the build does the boot + stat/combat
inspection.

---

## Out of scope

- **335 constable_drunn** — needs the deferred `stillwater_guards` faction; not
  enforcing yet (see `project_town_justice_5_1_followups`).
- **Town Justice 5.1c arrest** — this is its precursor; 5.1c resumes after.
- **Subdue/non-lethal-takedown weapons** (sap/net) — the richer combat-subdue
  arrest path stays out of both this chunk and 5.1c.
- **Guard reinforcement / call-for-help spawning, AI tactics tuning** beyond what
  tank_taunter already provides.
- **Mutation-triggered archetype shifts** and other archetype work
  (`project_mutation_triggered_archetype_shift`).
