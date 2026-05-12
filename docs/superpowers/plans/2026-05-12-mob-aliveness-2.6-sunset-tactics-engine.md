# Mob Aliveness 2.6 — Sunset Legacy Tactics Engine Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Delete the legacy `internal/mobai/` tactics engine and migrate all 44 tactic-using mobs to the behavior tree (btree) system. Eliminates the dual-system smell + the Edrin priority-race bug structurally (btree selectors are inherently priority-ordered, no async reaction queue to race against `InitiateCast`).

**Architecture:** Zero new btree primitives (the existing `mob_has_buff` + invert decorator covers `missing_buff`). Three existing preset bundles delete outright (`aggressive_melee`, `tank`, `ambusher`) — their rules are already in btree archetypes' combat cascades, or get folded into the archetype via small branch additions. One new `defensive_caster` archetype absorbs 4 mobs (3 from the preset of the same name + the lone `caster_backline` user). Five per-boss archetypes for named encounter mobs preserve their unique spell rotations. Five existing archetypes gain a shared panic-flee branch.

**Tech Stack:** Go 1.21+; existing `internal/behaviortree` package; existing btree YAML schema in `_datafiles/world/dogmud/behaviors/archetypes/`.

**Spec:** `docs/superpowers/specs/2026-05-12-mob-aliveness-2.6-sunset-tactics-engine-design.md`

**Branch:** Stay on `feature/mob-aliveness-1.3-crimes` per the single-feature-branch SOP.

---

## File structure

| File | Status | Responsibility |
|------|--------|----------------|
| `_datafiles/world/dogmud/behaviors/archetypes/generic_fighter.yaml` | MODIFY | Add `mob_hurt + health_below:25 → flee` panic-flee branch |
| `_datafiles/world/dogmud/behaviors/archetypes/predator.yaml` | MODIFY | Same panic-flee branch |
| `_datafiles/world/dogmud/behaviors/archetypes/leader.yaml` | MODIFY | Same panic-flee branch |
| `_datafiles/world/dogmud/behaviors/archetypes/lookout.yaml` | MODIFY | Same panic-flee branch |
| `_datafiles/world/dogmud/behaviors/archetypes/tank_taunter.yaml` | MODIFY | Same panic-flee branch + new `mob_hurt + health_below:20 → callforhelp` branch (absorbs the `tank` preset) |
| `_datafiles/world/dogmud/behaviors/archetypes/ambusher.yaml` | MODIFY | Add `mob_combat_round + target_is_casting → trip` branch (absorbs the `ambusher` preset's third rule) |
| `_datafiles/world/dogmud/behaviors/archetypes/defensive_caster.yaml` | NEW | Caster pattern absorbed from `defensive_caster` + `caster_backline` presets |
| `_datafiles/world/dogmud/behaviors/archetypes/boss_edrin.yaml` | NEW | Edrin's per-boss tree |
| `_datafiles/world/dogmud/behaviors/archetypes/boss_sylara.yaml` | NEW | Windwarden Sylara's per-boss tree |
| `_datafiles/world/dogmud/behaviors/archetypes/boss_rhett.yaml` | NEW | Geomancer Rhett's per-boss tree |
| `_datafiles/world/dogmud/behaviors/archetypes/boss_soren.yaml` | NEW | Soren's per-boss tree |
| `_datafiles/world/dogmud/behaviors/archetypes/boss_chrysalis_phantom.yaml` | NEW | Chrysalis Phantom's per-boss tree |
| 44 mob YAMLs across many zone folders | MODIFY | Strip `tactic_preset:`, `tactics:`, `reaction_delay:`, `tactical_discipline:`; reassign `behavior_archetype:` for the boss + defensive_caster cases |
| `internal/mobai/tactics.go` | DELETE | Presets + EvaluateTactics |
| `internal/mobai/reactor.go` | DELETE | pendingReactions queue, Reaction(), ProcessPendingReactions, signal dispatch |
| `internal/mobai/actions.go` | DELETE | ExecuteAction + helpers |
| `internal/mobai/types.go` | DELETE | Engine-internal types (TacticRule, Signal, TriggerContext, MobActor, etc.) |
| `internal/mobai/*_test.go` | DELETE | Tests for the deleted files |
| `internal/mobs/mobs.go` | MODIFY | Remove `Tactics`, `TacticPreset`, `ReactionDelay`, `TacticalDiscipline` Mob fields (lines 121-124); remove any `mobai.*` callers |
| `internal/hooks/*.go` | MODIFY | Remove any `mobai.Reaction(...)` / `mobai.ProcessPendingReactions(...)` callers |
| `internal/mobs/context.md` | MODIFY | Remove legacy tactics-engine references; note the chunk-2.6 migration |
| `internal/behaviortree/context.md` | MODIFY | Add panic-flee pattern documentation; list new archetypes (defensive_caster + 5 boss) |
| `MOB_ALIVENESS_ROADMAP.md` | MODIFY | Reframe chunk 2.6 title; mark Done; roll-up 13/41 → 14/41 |
| `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\project_tactics_cast_preemption.md` | DELETE | Bug resolved; memory note obsolete |

---

## Task 1: Augment existing archetypes (panic-flee, call_for_help, ambusher trip)

**Files:**
- Modify: `_datafiles/world/dogmud/behaviors/archetypes/generic_fighter.yaml`
- Modify: `_datafiles/world/dogmud/behaviors/archetypes/predator.yaml`
- Modify: `_datafiles/world/dogmud/behaviors/archetypes/leader.yaml`
- Modify: `_datafiles/world/dogmud/behaviors/archetypes/lookout.yaml`
- Modify: `_datafiles/world/dogmud/behaviors/archetypes/tank_taunter.yaml`
- Modify: `_datafiles/world/dogmud/behaviors/archetypes/ambusher.yaml`

The panic-flee branch is shared across 5 archetypes — same YAML, copied verbatim. Inserted as the FIRST child of each archetype's top-level selector (so emergency flee outranks any combat action).

### Step 1: Add panic-flee branch to `generic_fighter.yaml`

Open the file. Find the `tree:` block. The first child of the top-level selector is currently the `packmate_hurt → attack` branch. Insert ABOVE it:

```yaml
    # NEW: panic-flee at critical HP (absorbs the inline tactic
    # `health_below:25 → flee` from chunk 2.6 migration).
    - type: sequence
      event: mob_hurt
      children:
        - type: condition
          check: mob_health_below
          percent: 25
        - type: action
          do: flee
```

(The condition check is `mob_health_below`, with `percent` as the param — verify by reading `internal/behaviortree/conditions_mob.go` if the exact param name is different.)

### Step 2: Add the same panic-flee branch to `predator.yaml`

Open the file. The top-level selector's first child is currently the `mob_idle → target_weakest_mob_in_room` branch (chunk 2.4 work). Insert ABOVE it the same panic-flee branch.

### Step 3: Add the same panic-flee branch to `leader.yaml`

Open the file. The top-level selector's first child is currently the `packmate_hurt → rally/warcry + attack` branch. Insert ABOVE it the same panic-flee branch.

### Step 4: Add the same panic-flee branch to `lookout.yaml`

Open the file. The top-level selector's first child is currently the `player_enter → ambush` branch (chunk 2.4 work). Insert ABOVE it the same panic-flee branch.

### Step 5: Add panic-flee + new `call_for_help` branch to `tank_taunter.yaml`

Open the file. Top-level selector starts with `packmate_hurt → attack`. Insert TWO branches at the top:

```yaml
    # NEW: panic-flee at critical HP (absorbs the inline tactic
    # `health_below:25 → flee` from chunk 2.6 migration).
    - type: sequence
      event: mob_hurt
      children:
        - type: condition
          check: mob_health_below
          percent: 25
        - type: action
          do: flee

    # NEW: call for help at low HP (absorbs the `tank` tactic preset).
    - type: sequence
      event: mob_hurt
      children:
        - type: condition
          check: mob_health_below
          percent: 20
        - type: action
          do: command
          cmd: callforhelp
```

Order matters: panic-flee at 25% comes first (ranks higher), then call_for_help at 20% (fires only if mob is below 20% AND for some reason didn't flee).

### Step 6: Add `target_is_casting → trip` branch to `ambusher.yaml`

Open the file. The existing tree has these branches (in order): `player_enter + mob_has_buff:9 → attack`, `mob_idle + buff:9 + players_in_room → attack`, `mob_hurt → flee`, `mob_idle + NOT buff:9 → add_buff`.

Add a NEW branch for interrupt-trip during combat. Place it AFTER the `mob_hurt → flee` (a fleeing mob shouldn't be tripping; flee outranks). The new branch:

```yaml
    # NEW: interrupt casting targets mid-combat (absorbs the
    # `ambusher` tactic preset's third rule).
    - type: sequence
      event: mob_combat_round
      children:
        - type: condition
          check: target_is_casting
        - type: action
          do: command_best_of
          cmds: [trip]
```

(The action is `command_best_of` to share the special-move cooldown with bash/kick/grapple, matching the pattern used in `generic_fighter`'s combat cascade.)

### Step 7: Boot the server to verify all archetypes parse

Run: `go build ./...` — clean.

```bash
go run . > /tmp/boot_task1.log 2>&1 &
```

Wait ~15s. Check:
```bash
grep -E "behaviors.LoadDataFiles|panic|Server Ready" /tmp/boot_task1.log
```

Expected:
- `behaviors.LoadDataFiles() loadedCount=<existing baseline>` (no change — we're modifying existing files, not adding new ones)
- `Server Ready`
- No panics

Kill the server:
```bash
ps -ef 2>/dev/null | grep -E '(go run|dogmud)' | grep -v grep | awk '{print $2}' | xargs -r kill -9 2>/dev/null
```

### Step 8: Commit

```bash
git add _datafiles/world/dogmud/behaviors/archetypes/
git commit -m "feat(behaviors): augment archetypes with panic-flee + call_for_help + trip branches"
```

---

## Task 2: Create defensive_caster archetype + 5 per-boss archetypes

**Files:**
- Create: `_datafiles/world/dogmud/behaviors/archetypes/defensive_caster.yaml`
- Create: `_datafiles/world/dogmud/behaviors/archetypes/boss_edrin.yaml`
- Create: `_datafiles/world/dogmud/behaviors/archetypes/boss_sylara.yaml`
- Create: `_datafiles/world/dogmud/behaviors/archetypes/boss_rhett.yaml`
- Create: `_datafiles/world/dogmud/behaviors/archetypes/boss_soren.yaml`
- Create: `_datafiles/world/dogmud/behaviors/archetypes/boss_chrysalis_phantom.yaml`

Each per-boss archetype is authored from the boss's existing inline `tactics:` block — the spec listed the exact rules. The defensive_caster archetype absorbs the 4 mobs using the old `defensive_caster` and `caster_backline` presets.

### Step 1: Create `defensive_caster.yaml`

```yaml
# defensive_caster archetype
#
# Caster pattern with self-preservation: panic-buff (chrysalis-
# cocoon) when buff 2 is missing, AoE on multiple targets,
# single-target spike otherwise, flee at low HP.
#
# Used by: goblin_shaman (219), tunnel_shaman (74),
# bandit_caster (285), elemental_queen (321). Absorbs the
# legacy `defensive_caster` and `caster_backline` tactic presets
# (chunk 2.6 migration).
#
# Spec: docs/superpowers/specs/2026-05-12-mob-aliveness-2.6-sunset-tactics-engine-design.md

tree:
  type: selector
  children:
    # Panic-flee at low HP — wins over any cast
    - type: sequence
      event: mob_hurt
      children:
        - type: condition
          check: mob_health_below
          percent: 30
        - type: action
          do: flee

    # Per-round combat cascade
    - type: selector
      event: mob_combat_round
      children:
        # Panic-buff: cast cocoon when buff 2 is missing
        - type: sequence
          children:
            - type: decorator
              mod: invert
              child:
                type: condition
                check: mob_has_buff
                buff_id: 2
            - type: action
              do: cast
              spell: chrysalis-cocoon

        # Multi-target AoE
        - type: sequence
          children:
            - type: condition
              check: multiple_enemies
            - type: action
              do: cast
              spell: conviction-barrage

        # Single-target fallback
        - type: action
          do: cast
          spell: conviction-spike
```

### Step 2: Create `boss_edrin.yaml`

```yaml
# boss_edrin archetype
#
# Old Edrin (mob 275) — a fragile mind-spike caster whose
# defining trick is fold-recall: when below 30% HP, he opens
# a fold and recalls home. Below that he uses a heal at 50%
# HP. Combat rotation: opening conviction-ward, interrupt
# with mind-spike, hemorrhagic-burst on multi-target,
# pyretic-surge single-target fallback.
#
# Migrated from inline tactics in chunk 2.6.
#
# Spec: docs/superpowers/specs/2026-05-12-mob-aliveness-2.6-sunset-tactics-engine-design.md

tree:
  type: selector
  children:
    # Emergency recall at HP<30
    - type: sequence
      event: mob_hurt
      children:
        - type: condition
          check: mob_health_below
          percent: 30
        - type: action
          do: cast
          spell: fold-recall

    # Panic-flee at HP<25 (only fires if fold-recall failed)
    - type: sequence
      event: mob_hurt
      children:
        - type: condition
          check: mob_health_below
          percent: 25
        - type: action
          do: flee

    # Heal at HP<50
    - type: sequence
      event: mob_hurt
      children:
        - type: condition
          check: mob_health_below
          percent: 50
        - type: action
          do: cast
          spell: heal

    # Combat cascade
    - type: selector
      event: mob_combat_round
      children:
        # Opening buff: conviction-ward when buff is missing.
        # (Replaces the legacy `combat_start` trigger.)
        - type: sequence
          children:
            - type: decorator
              mod: invert
              child:
                type: condition
                check: mob_has_buff
                buff_id: 4
            - type: action
              do: cast
              spell: conviction-ward

        # Interrupt: mind-spike on a casting target
        - type: sequence
          children:
            - type: condition
              check: target_is_casting
            - type: action
              do: cast
              spell: mind-spike

        # Multi-target AoE
        - type: sequence
          children:
            - type: condition
              check: multiple_enemies
            - type: action
              do: cast
              spell: hemorrhagic-burst

        # Single-target fallback
        - type: action
          do: cast
          spell: pyretic-surge
```

**Note on `buff_id: 4`**: The conviction-ward buff has a specific ID. Verify by reading `_datafiles/world/dogmud/buffs/4-*.yaml` (or the actual buff matching conviction-ward). If the buff id differs, update the value before commit. Same pattern applies to `buff_id: 2` in defensive_caster (chrysalis-cocoon's buff).

### Step 3: Create `boss_sylara.yaml`

```yaml
# boss_sylara archetype
#
# Windwarden Sylara (mob 241) — quest-giving boss caster.
# Defensive caster with heal on HP<30, panic-buff chrysalis-
# cocoon when missing, opening conviction-ward, and bash to
# interrupt casters.
#
# Migrated from inline tactics in chunk 2.6.

tree:
  type: selector
  children:
    # Heal at HP<30
    - type: sequence
      event: mob_hurt
      children:
        - type: condition
          check: mob_health_below
          percent: 30
        - type: action
          do: cast
          spell: heal

    # Combat cascade
    - type: selector
      event: mob_combat_round
      children:
        # Panic-buff: chrysalis-cocoon when buff 2 is missing
        - type: sequence
          children:
            - type: decorator
              mod: invert
              child:
                type: condition
                check: mob_has_buff
                buff_id: 2
            - type: action
              do: cast
              spell: chrysalis-cocoon

        # Opening: conviction-ward when buff is missing
        - type: sequence
          children:
            - type: decorator
              mod: invert
              child:
                type: condition
                check: mob_has_buff
                buff_id: 4
            - type: action
              do: cast
              spell: conviction-ward

        # Interrupt: bash on a casting target
        - type: sequence
          children:
            - type: condition
              check: target_is_casting
            - type: action
              do: command_best_of
              cmds: [bash]
```

### Step 4: Create `boss_rhett.yaml`

```yaml
# boss_rhett archetype
#
# Geomancer Rhett (mob 242) — Thornwall scholar / questgiver
# who can defend himself. His only inline tactic was a
# defensive opening (conviction-armor). The aggressive_melee
# preset he also had was an authoring artifact (he's not really
# aggressive_melee). Migrated as a defense-only caster.
#
# Quest-giving behavior rides on his dialogue YAML, not this
# archetype — switching from `noncombat_questgiver` to
# `boss_rhett` keeps the dialogue flow intact.
#
# Migrated from inline tactics in chunk 2.6.

tree:
  type: selector
  children:
    # Panic-flee at HP<25 (Rhett's not a fighter)
    - type: sequence
      event: mob_hurt
      children:
        - type: condition
          check: mob_health_below
          percent: 25
        - type: action
          do: flee

    # Combat cascade
    - type: selector
      event: mob_combat_round
      children:
        # Opening: conviction-armor when buff is missing
        - type: sequence
          children:
            - type: decorator
              mod: invert
              child:
                type: condition
                check: mob_has_buff
                buff_id: 5
            - type: action
              do: cast
              spell: conviction-armor
```

(Verify `buff_id: 5` matches the conviction-armor buff; adjust during implementation if different.)

### Step 5: Create `boss_soren.yaml`

```yaml
# boss_soren archetype
#
# Soren (mob 286) — named bandit camp leader. Extends the
# `leader` archetype's rally/warcry pattern with a low-HP
# call_for_help trigger that's specific to Soren (not
# universal to all leader-archetype mobs).
#
# Migrated from inline tactics in chunk 2.6.

tree:
  type: selector
  children:
    # Panic-flee at critical HP
    - type: sequence
      event: mob_hurt
      children:
        - type: condition
          check: mob_health_below
          percent: 25
        - type: action
          do: flee

    # Soren-specific: call for help at HP<30
    - type: sequence
      event: mob_hurt
      children:
        - type: condition
          check: mob_health_below
          percent: 30
        - type: action
          do: command
          cmd: callforhelp

    # Below: identical to the `leader` archetype's combat behavior.
    - type: sequence
      event: packmate_hurt
      children:
        - type: action
          do: command_best_of
          cmds: [rally, warcry]
        - type: action
          do: attack

    - type: sequence
      event: mob_hurt
      children:
        - type: action
          do: command_best_of
          cmds: [rally, warcry]
        - type: action
          do: attack
```

### Step 6: Create `boss_chrysalis_phantom.yaml`

```yaml
# boss_chrysalis_phantom archetype
#
# Chrysalis Phantom (mob 272) — eldritch melee bruiser. Flees
# below 20% HP and trips casting targets mid-combat. The
# generic `ambusher` archetype almost covers this (after the
# chunk 2.6 augmentation it gains target_casting→trip too), but
# the Phantom has a tighter panic threshold (HP<20 vs the
# augmented archetype's HP<25 panic-flee) and uses `trip`
# (single-cmd) rather than `command_best_of`.
#
# Migrated from inline tactics in chunk 2.6.

tree:
  type: selector
  children:
    # Panic-flee at HP<20 (tighter than the generic 25%)
    - type: sequence
      event: mob_hurt
      children:
        - type: condition
          check: mob_health_below
          percent: 20
        - type: action
          do: flee

    # Interrupt: trip a casting target mid-combat
    - type: sequence
      event: mob_combat_round
      children:
        - type: condition
          check: target_is_casting
        - type: action
          do: command_best_of
          cmds: [trip]
```

### Step 7: Boot the server to verify all 6 archetypes parse

Run: `go build ./...` — clean.

```bash
go run . > /tmp/boot_task2.log 2>&1 &
```

Wait ~15s. Check:
```bash
grep -E "behaviors.LoadDataFiles|panic|Server Ready" /tmp/boot_task2.log
```

Expected:
- `behaviors.LoadDataFiles() loadedCount=<baseline + 6>` (the 6 new archetypes)
- `Server Ready`
- No panics

Kill the server.

### Step 8: Commit

```bash
git add _datafiles/world/dogmud/behaviors/archetypes/
git commit -m "feat(behaviors): add defensive_caster + 5 per-boss archetypes"
```

---

## Task 3: Migrate preset-using mobs (27 mobs)

**Files:** 27 mob YAMLs across many zone folders.

This task strips `tactic_preset:`, `tactical_discipline:`, and `reaction_delay:` from each mob. Three of the four `defensive_caster` mobs + the queen also need their `behavior_archetype:` reassigned. The other 23 keep their existing archetype.

### Step 1: Strip preset fields from the 23 mobs that keep their archetype

For each mob below, remove these YAML keys (top-level, sibling to `mobid:`):
- `tactic_preset:`
- `tactical_discipline:`
- `reaction_delay:`

Do NOT change their `behavior_archetype:`. The augmented archetype (Task 1) absorbs the preset's behavior.

**`aggressive_melee` preset mobs (11 — strip preset only; the 2 other aggressive_melee mobs, Rhett (242) and Soren (286), are handled in Task 4 because they have inline tactics + become bosses):**
- `_datafiles/world/dogmud/mobs/instance_arena/324-arena_champion.yaml`
- `_datafiles/world/dogmud/mobs/instance_planar_oasis/320-elemental_king.yaml`
- `_datafiles/world/dogmud/mobs/instance_planar_oasis/322-elemental_prince.yaml`
- `_datafiles/world/dogmud/mobs/ironwind_steppe/218-goblin_scrapper.yaml`
- `_datafiles/world/dogmud/mobs/ironwind_steppe/226-deep_gnawer.yaml`
- `_datafiles/world/dogmud/mobs/ironwind_steppe/229-windscour_wyrm.yaml` — also has inline tactics; strip both
- `_datafiles/world/dogmud/mobs/labyrinth_of_low_tunnels/73-warren_warrior.yaml` — also has inline tactics; strip both
- `_datafiles/world/dogmud/mobs/marches_spur_road/254-bandit_leader.yaml` — also has inline tactics; strip both
- `_datafiles/world/dogmud/mobs/north_road/284-bandit_fighter.yaml` — also has inline tactics; strip both
- `_datafiles/world/dogmud/mobs/north_road/287-bloodline_agent.yaml`
- `_datafiles/world/dogmud/mobs/stillwater/332-sump_dweller.yaml` — also has inline tactics; strip both

**`tank` preset mobs (4):**
- Find them: `grep -l "tactic_preset: tank" _datafiles/world/dogmud/mobs/**/*.yaml`. Strip the preset + discipline + delay fields from each. If any of these mobs ALSO has an inline `tactics:` block, strip that too. None of them are bosses (no archetype reassignment).

**`ambusher` preset mobs (5 — strip preset only; the 6th ambusher mob, Chrysalis Phantom (272), is handled in Task 4 because she has inline tactics + becomes a boss):**
- Find them: `grep -l "tactic_preset: ambusher" _datafiles/world/dogmud/mobs/**/*.yaml`. Exclude 272-chrysalis_phantom from this list — she's handled in Task 4. Strip preset + discipline + delay from the remaining 5.

### Step 2: Reassign archetype + strip preset for the 4 defensive_caster/caster_backline mobs

Each gets `behavior_archetype: defensive_caster` (replaces whatever's currently there) AND has `tactic_preset:`, `tactical_discipline:`, `reaction_delay:` stripped.

| File | Old archetype | New archetype |
|---|---|---|
| `_datafiles/world/dogmud/mobs/ironwind_steppe/219-goblin_shaman.yaml` | (none — preset only) | defensive_caster |
| `_datafiles/world/dogmud/mobs/labyrinth_of_low_tunnels/74-tunnel_shaman.yaml` | (none — preset only) | defensive_caster |
| `_datafiles/world/dogmud/mobs/north_road/285-bandit_caster.yaml` | (none — preset only) | defensive_caster |
| `_datafiles/world/dogmud/mobs/instance_planar_oasis/321-elemental_queen.yaml` | (none — caster_backline preset only) | defensive_caster |

If any of these has no existing `behavior_archetype:` line, ADD one (top-level, alongside `mobid:`):

```yaml
behavior_archetype: defensive_caster
```

If one already exists (which the spec didn't anticipate), replace its value.

### Step 3: Boot the server

Run: `go build ./...` — clean.

```bash
go run . > /tmp/boot_task3.log 2>&1 &
```

Wait ~15s. Check:
```bash
grep -E "mobs.LoadDataFiles|panic|Server Ready|unknown field|cannot unmarshal" /tmp/boot_task3.log
```

Expected:
- `mobs.LoadDataFiles() loadedCount=<unchanged>`
- `Server Ready`
- No panics
- Possibly warnings about unknown fields if the YAML parser surfaces stripped-but-still-defined-in-Mob-struct fields. The fields are still in the Mob struct at this point; the stripped YAML just won't populate them. No warning expected — `omitempty` on YAML side means absence is fine.

Kill the server.

### Step 4: Commit

```bash
git add _datafiles/world/dogmud/mobs/
git commit -m "refactor(mobs): strip tactic_preset from 27 preset-using mobs"
```

---

## Task 4: Migrate inline-tactics mobs (17 mobs)

**Files:** 17 mob YAMLs.

12 generic mobs strip their inline `tactics:` field (their tactics' behavior is absorbed by the augmented archetypes from Task 1). 5 named bosses get reassigned to per-boss archetypes.

### Step 1: Reassign archetype + strip tactics for the 5 named bosses

For each boss below, change `behavior_archetype:` to the listed value AND remove `tactic_preset:`, `tactics:`, `tactical_discipline:`, `reaction_delay:`.

| File | New `behavior_archetype:` |
|---|---|
| `_datafiles/world/dogmud/mobs/marches_spur_road/275-old_edrin.yaml` | boss_edrin |
| `_datafiles/world/dogmud/mobs/ironwind_steppe/241-windwarden_sylara.yaml` | boss_sylara |
| `_datafiles/world/dogmud/mobs/ironwind_steppe/242-geomancer_rhett.yaml` | boss_rhett |
| `_datafiles/world/dogmud/mobs/north_road/286-soren.yaml` | boss_soren |
| `_datafiles/world/dogmud/mobs/thornwall_city/272-chrysalis_phantom.yaml` | boss_chrysalis_phantom |

### Step 2: Strip tactics from the 12 generic inline-tactics mobs

These keep their existing `behavior_archetype:`. Remove `tactic_preset:` (if present), `tactics:`, `tactical_discipline:`, `reaction_delay:`.

| File |
|---|
| `_datafiles/world/dogmud/mobs/ashwick/262-the_forager.yaml` |
| `_datafiles/world/dogmud/mobs/dustwalk_road/80-dustwalk_bandit.yaml` |
| `_datafiles/world/dogmud/mobs/ironwind_steppe/217-goblin_scout.yaml` |
| `_datafiles/world/dogmud/mobs/ironwind_steppe/224-cave_crawler.yaml` |
| `_datafiles/world/dogmud/mobs/ironwind_steppe/225-pale_lurker.yaml` |
| `_datafiles/world/dogmud/mobs/ironwind_steppe/227-blind_stalker.yaml` |
| `_datafiles/world/dogmud/mobs/labyrinth_of_low_tunnels/72-warren_scout.yaml` |
| `_datafiles/world/dogmud/mobs/labyrinth_of_low_tunnels/75-warren_chieftain.yaml` |
| `_datafiles/world/dogmud/mobs/labyrinth_of_low_tunnels/78-spore_crawler.yaml` |
| `_datafiles/world/dogmud/mobs/marches_spur_road/253-road_bandit.yaml` |
| `_datafiles/world/dogmud/mobs/north_road/283-bandit_lookout.yaml` |
| `_datafiles/world/dogmud/mobs/stillwater/331-drowned_hunter.yaml` |
| `_datafiles/world/dogmud/mobs/thornwall_outskirts/90-thornwall_highwayman.yaml` |

If after stripping, a mob's behavior would be noticeably degraded (e.g., a uniquely-tuned panic flee threshold at 15% instead of the archetype's 25%), note it in the commit message but proceed — the spec accepted this as a tuning trade-off.

### Step 3: Final verification — no remaining tactics references in mob YAMLs

Run:
```bash
grep -rln "^tactic_preset:\|^tactics:\|^reaction_delay:\|^tactical_discipline:" _datafiles/world/dogmud/mobs/ 2>/dev/null
```

Expected: zero matches. If any mob still has these fields, find and clean.

### Step 4: Boot the server

```bash
go run . > /tmp/boot_task4.log 2>&1 &
```

Wait ~15s. Check:
```bash
grep -E "mobs.LoadDataFiles|panic|Server Ready" /tmp/boot_task4.log
```

Expected: server reaches Server Ready, no panics, mob count unchanged.

Kill the server.

### Step 5: Commit

```bash
git add _datafiles/world/dogmud/mobs/
git commit -m "refactor(mobs): migrate 17 inline-tactics mobs to btree archetypes"
```

---

## Task 5: Delete legacy tactics engine + Mob struct fields + hook callers

**Files:**
- Delete: `internal/mobai/tactics.go`
- Delete: `internal/mobai/reactor.go`
- Delete: `internal/mobai/actions.go`
- Delete: `internal/mobai/types.go`
- Delete: all `internal/mobai/*_test.go`
- Modify: `internal/mobs/mobs.go` (remove 4 fields)
- Modify: hook callers (TBD via grep; likely `internal/hooks/*.go`)

This is the engine deletion. Must happen AFTER all mob YAMLs are clean (Tasks 3 + 4). Order within this task matters because of mutual dependencies between the Mob struct field types and the mobai package.

### Step 1: Find all mobai imports

Run:
```bash
grep -rln "internal/mobai" internal/ 2>/dev/null
```

Note every file. Likely:
- `internal/mobs/mobs.go` (imports for `mobai.TacticRule`)
- `internal/hooks/*.go` (calls into `mobai.Reaction`, `mobai.ProcessPendingReactions`)
- Possibly `internal/usercommands/*.go` (unlikely — verify)

### Step 2: Remove the 4 Mob struct fields

Open `internal/mobs/mobs.go`. Find lines 121-124 (use grep if line numbers drift):

```go
ReactionDelay           float64             `yaml:"reaction_delay,omitempty"`      // Seconds before executing a reactive tactic (default 1.5)
TacticalDiscipline      float64             `yaml:"tactical_discipline,omitempty"` // 0.0-1.0, how reliably mob follows tactics (default 0.5)
TacticPreset            string              `yaml:"tactic_preset,omitempty"`       // Named preset: "aggressive_melee", "defensive_caster", "ambusher"
Tactics                 []mobai.TacticRule  `yaml:"tactics,omitempty"`             // Per-mob tactic overrides
```

Remove all 4 lines.

### Step 3: Remove mobai usage sites in mobs.go

Search the file for `m.Tactics`, `m.TacticPreset`, `m.ReactionDelay`, `m.TacticalDiscipline`, `mobai.`. Remove each line/block. Common patterns to look for:

- `base := m.TacticalDiscipline` (and similar reads)
- Any function that calls `mobai.GetEffectiveReactionDelay(m.ReactionDelay, ...)`
- Any function that builds tactics from `mobai.GetPreset(m.TacticPreset)` or merges `m.Tactics`

If a function existed solely to manage tactics (e.g., `m.BuildTactics()` or similar), delete it entirely.

After this step, `internal/mobs/mobs.go` should have no references to `mobai`.

### Step 4: Remove mobai callers in hooks

For each hook file in `internal/hooks/` that imports `mobai`:

- Find calls like `mobai.Reaction(...)`, `mobai.ProcessPendingReactions(...)`, `mobai.ClearPendingForMob(...)`, `mobai.ExecuteAction(...)`.
- Delete each call site. If a function exists solely to dispatch signals to the engine (e.g., `signalCombatStart(mob, ...)` that calls `mobai.Reaction`), delete the entire function.
- If a hook file becomes empty after deletion, delete the file entirely.

Tactic-related hook calls to look for (based on spec):
- combat_start signal emission
- action_complete signal emission
- player_entered signal emission
- ProcessPendingReactions every-turn tick

### Step 5: Delete the mobai package files

```bash
rm internal/mobai/tactics.go
rm internal/mobai/reactor.go
rm internal/mobai/actions.go
rm internal/mobai/types.go
rm internal/mobai/*_test.go
```

Then check if the directory is empty:
```bash
ls internal/mobai/ 2>/dev/null
```

If empty, remove the directory:
```bash
rmdir internal/mobai/
```

### Step 6: Build and verify

Run: `go build ./...`
Expected: clean. Any leftover `mobai.*` reference produces a compile error pointing to the call site — fix by removing.

Run: `go test ./...`
Expected: no FAILs. Some pre-existing tests in `internal/mobai/` are deleted along with the files; that's expected.

### Step 7: Final grep verification

```bash
grep -rln "internal/mobai\|mobai\." internal/ 2>/dev/null
```

Expected: zero matches.

```bash
ls internal/mobai/ 2>/dev/null
```

Expected: directory doesn't exist (or returns "No such file or directory").

### Step 8: Commit

```bash
git add -A
git commit -m "refactor(mobs): delete legacy mobai tactics engine + Mob struct fields"
```

---

## Task 6: Update context.md files

**Files:**
- Modify: `internal/mobs/context.md`
- Modify: `internal/behaviortree/context.md`

### Step 1: Update `internal/mobs/context.md`

Search for any reference to the tactics engine, `TacticPreset`, `TacticalDiscipline`, `ReactionDelay`, `Tactics` field, or `internal/mobai`. Remove all such references.

If a dedicated "Tactics System" or similar section exists, replace it with:

```markdown
### Mob Behavior (chunk 2.6 update)

Mob behavior is driven entirely by the behavior tree (btree) system —
see `internal/behaviortree/context.md`. The legacy `internal/mobai/`
tactics engine was removed in chunk 2.6; the Mob struct no longer
carries `Tactics`, `TacticPreset`, `ReactionDelay`, or
`TacticalDiscipline` fields. Mob YAMLs no longer support
`tactic_preset:`, `tactics:`, `reaction_delay:`, or
`tactical_discipline:` keys (they're silently ignored if present).

Design: `docs/superpowers/specs/2026-05-12-mob-aliveness-2.6-sunset-tactics-engine-design.md`
```

If no such section exists, add the above as a new subsection in the appropriate location (likely under a "Behavior" or "Combat" subsection of the file).

### Step 2: Update `internal/behaviortree/context.md`

Add a new section documenting the chunk-2.6 archetype additions + the panic-flee pattern:

```markdown
### Panic-Flee Pattern (chunk 2.6)

A shared `mob_hurt + mob_health_below:N → flee` branch is the FIRST
child of the top-level selector in five core archetypes:
`generic_fighter`, `predator`, `leader`, `lookout`, `tank_taunter`.
Threshold defaults to 25% HP. Emergency flee outranks any combat
action because it's the first matching branch in the selector
evaluation order.

Mobs that need a different threshold (e.g., Chrysalis Phantom at
20%, Edrin's heal-at-50% sequence) author a per-boss archetype that
overrides the default.

### New Archetypes (chunk 2.6)

Added in the legacy tactics-engine sunset migration:

- **`defensive_caster`** — Caster pattern with self-preservation:
  panic-flee at HP<30, panic-buff (chrysalis-cocoon when buff 2
  missing), AoE on multiple targets (conviction-barrage),
  single-target spike (conviction-spike). Used by goblin_shaman
  (219), tunnel_shaman (74), bandit_caster (285),
  elemental_queen (321). Absorbed the legacy `defensive_caster`
  and `caster_backline` tactic presets.
- **`boss_edrin`** — Old Edrin's fragile-caster rotation with
  fold-recall at HP<30, heal at HP<50, opening conviction-ward,
  mind-spike on casters, hemorrhagic-burst on multi, pyretic-
  surge single-target.
- **`boss_sylara`** — Windwarden Sylara's heal-at-30 + panic
  chrysalis-cocoon + conviction-ward opener + bash interrupt.
- **`boss_rhett`** — Geomancer Rhett's defense-only opener
  (conviction-armor) + panic-flee.
- **`boss_soren`** — Soren's leader-archetype combat plus a
  call_for_help at HP<30 branch.
- **`boss_chrysalis_phantom`** — Tight panic-flee (HP<20) +
  target_casting → trip interrupt.

Spec: `docs/superpowers/specs/2026-05-12-mob-aliveness-2.6-sunset-tactics-engine-design.md`
```

Place this section in the file's existing flow — likely near other archetype documentation. If there's a "Combat" or "Archetypes" subsection, append there.

### Step 3: Commit

```bash
git add internal/mobs/context.md internal/behaviortree/context.md
git commit -m "docs: document mobai sunset and new chunk-2.6 archetypes"
```

---

## Task 7: Roadmap + MEMORY cleanup

**Files:**
- Modify: `MOB_ALIVENESS_ROADMAP.md`
- Delete: `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\project_tactics_cast_preemption.md`

### Step 1: Update the progress tracker

Locate the chunk 2.6 row. Change from:

```markdown
| 2.6 | Tactical | Tactics-cast preemption fix | S | — | Not started |
```

To:

```markdown
| 2.6 | Tactical | Sunset legacy tactics engine | L | — | Done |
```

(Title + size both change. Size is L because of the engine deletion + 44 mob migrations.)

### Step 2: Update the roll-up line

Change:

```markdown
**Roll-up:** 13 / 41 done • 0 in progress • 28 not started.
```

To:

```markdown
**Roll-up:** 14 / 41 done • 0 in progress • 27 not started.
```

### Step 3: Update the 2.6 mini-brief

Locate `### 2.6 Tactics-cast preemption fix`. Replace the entire section with:

```markdown
### 2.6 Sunset legacy tactics engine
**Status:** Done (2026-05-12) • **Size:** L

- **Goal:** Delete the legacy `internal/mobai/` tactics engine and migrate all 44 tactic-using mobs to the behavior tree (btree) system.
- **In:** Reframed from the original "fix the Edrin priority race" band-aid into the structural fix. Btree now the single mob-behavior substrate. Five existing archetypes gain a shared panic-flee branch (generic_fighter, predator, leader, lookout, tank_taunter). tank_taunter additionally gets a call_for_help branch (absorbing the `tank` preset); ambusher gets a target_casting→trip branch (absorbing the `ambusher` preset). One new `defensive_caster` archetype absorbs 4 mobs from the old `defensive_caster` and `caster_backline` presets. Five per-boss archetypes for named encounter mobs (Edrin, Sylara, Rhett, Soren, Chrysalis Phantom) preserve their unique spell rotations.
- **Out:** Boss encounter tuning (faithful translation only); generic-mob inline-tactic preservation beyond what the augmented archetypes cover (acceptable loss).
- **Depends on:** —
- **Why:** Eliminated the dual-system architectural smell. The original Edrin priority-race bug became structurally impossible (btree selectors are inherently priority-ordered, no async reaction queue racing `InitiateCast`). ~400-500 lines of legacy code deleted.
- **Shipped:** Zero new btree primitives (mob_has_buff + invert decorator covers missing_buff). 6 new archetypes (defensive_caster + 5 boss). 5 archetype augmentations. 44 mob YAML migrations (27 preset-only + 17 inline-tactic). Engine deletion: `internal/mobai/tactics.go`, `reactor.go`, `actions.go`, `types.go`, plus all `*_test.go`. Mob struct fields `Tactics`, `TacticPreset`, `ReactionDelay`, `TacticalDiscipline` removed. Hook callers in `internal/hooks/` cleaned. `internal/mobs/context.md` + `internal/behaviortree/context.md` updated. `project_tactics_cast_preemption.md` MEMORY entry deleted. Spec at `docs/superpowers/specs/2026-05-12-mob-aliveness-2.6-sunset-tactics-engine-design.md`, plan at `docs/superpowers/plans/2026-05-12-mob-aliveness-2.6-sunset-tactics-engine.md`.
```

### Step 4: Delete the MEMORY entry

```bash
rm "C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\project_tactics_cast_preemption.md"
```

Also check if it's referenced in MEMORY.md (the index):

```bash
grep -n "tactics_cast_preemption" "C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\MEMORY.md"
```

If yes, remove the line from MEMORY.md too.

### Step 5: Commit

```bash
git add MOB_ALIVENESS_ROADMAP.md
git commit -m "docs(roadmap): mark chunk 2.6 (sunset tactics engine) as Done"
```

(The MEMORY file deletion is local to the user's `.claude/projects/` and not git-tracked — no commit needed for that step.)

---

## Task 8: Smoke validation

**Files:** None modified — pure validation.

### Step 1: Full build + test suite

Run: `go build ./...` — clean.
Run: `go test ./...` — no FAILs.

### Step 2: Verify no mobai references

```bash
grep -rln "internal/mobai\|mobai\." internal/ *.go 2>/dev/null
```

Expected: zero matches.

### Step 3: Verify no tactics fields in mob YAMLs

```bash
grep -rln "^tactic_preset:\|^tactics:\|^reaction_delay:\|^tactical_discipline:" _datafiles/world/dogmud/mobs/ 2>/dev/null
```

Expected: zero matches.

### Step 4: Boot the server

```bash
go run . > /tmp/chunk26_smoke.log 2>&1 &
```

Wait ~20 seconds.

```bash
grep -E "(behaviors|mobs).LoadDataFiles|Server Ready|panic|unknown field" /tmp/chunk26_smoke.log
```

Expected:
- `behaviors.LoadDataFiles() loadedCount=<baseline + 6>` (6 new archetypes)
- `mobs.LoadDataFiles() loadedCount=<unchanged>` (~225)
- `Server Ready`
- No panics
- No "unknown field" warnings

### Step 5: Spot-check via admin or AI tester

The smoketester account is admin-flagged from chunk 2.5. Reuse if convenient.

**Edrin priority-race (the original bug):**
- Spawn old_edrin (mob 275) in a test room
- Engage him; admin-drop his HP to <30%
- Expected: `fold-recall` fires reliably. Pre-chunk, this could be silently dropped if conviction-ward had been queued earlier.

**Sylara opener:**
- Spawn Sylara (mob 241). Engage.
- First combat round: `conviction-ward` fires (buff lands).
- Second round onward: opener no longer fires (buff present → invert(mob_has_buff) returns false → branch skipped). Next branch in the cascade fires instead.

**Tank call_for_help:**
- Spawn any tank_taunter-archetype mob. Drop HP below 20%.
- Expected: `callforhelp` emote/action fires.

**Ambusher mid-combat trip:**
- Spawn a bandit lookout (mob 283) — uses the augmented `ambusher` archetype after migration.
- Begin casting any spell in the room.
- Expected: lookout trips you mid-combat.

**Panic-flee universality:**
- Spawn any generic_fighter / predator / leader / lookout mob.
- Beat them below 25% HP.
- Expected: they flee.

**defensive_caster:**
- Spawn goblin_shaman (219). Engage.
- Expected on first round: chrysalis-cocoon casts (buff missing). On subsequent rounds (after buff lands): single-target spike or multi-target barrage depending on enemy count.

### Step 6: Kill test servers cleanly

```bash
ps -ef 2>/dev/null | grep -E '(go run|dogmud)' | grep -v grep | awk '{print $2}' | xargs -r kill -9 2>/dev/null
```

Verify clean:
```bash
ps -ef 2>/dev/null | grep -E '(go run|dogmud)' | grep -v grep || echo "no servers"
```

### Step 7: Final commit log review

```bash
git log --oneline -15
```

Expected sequence ending in the roadmap commit:
- `feat(behaviors): augment archetypes with panic-flee + call_for_help + trip branches`
- `feat(behaviors): add defensive_caster + 5 per-boss archetypes`
- `refactor(mobs): strip tactic_preset from 27 preset-using mobs`
- `refactor(mobs): migrate 17 inline-tactics mobs to btree archetypes`
- `refactor(mobs): delete legacy mobai tactics engine + Mob struct fields`
- `docs: document mobai sunset and new chunk-2.6 archetypes`
- `docs(roadmap): mark chunk 2.6 (sunset tactics engine) as Done`

---

## Out of scope (per spec)

- Boss encounter quality / fun tuning (faithful translation only this chunk)
- `combat_just_started` btree event type (not needed; missing-buff self-gating covers all openers)
- Archetype parameter / spell-id YAML configurability (defensive_caster hardcodes its spell rotation; add later if needed)
- `submit` action support in btree (legacy `aggressive_melee` preset's third rule; dropped — generic_fighter's existing cascade is different but acceptable)
- Per-mob behavior_extras composition mechanism (rejected in brainstorming; per-boss archetypes cover unique cases)
- Player-side cast feedback changes
- Backfilling `tactical_discipline`-as-randomness via `random_chance` btree condition (acceptable loss for v1)
