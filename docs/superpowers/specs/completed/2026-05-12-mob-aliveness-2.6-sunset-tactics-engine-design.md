# Mob Aliveness 2.6 — Sunset Legacy Tactics Engine

> **Phase 2 tactical (sixth chunk).** Reframed from the original
> "fix tactics-cast preemption race" band-aid into the structural
> fix: delete the legacy `internal/mobai` tactics engine entirely
> and consolidate mob behavior under the behavior tree (btree)
> system. The Edrin priority-race bug becomes structurally
> impossible once tactics are gone — btree selector node-order
> IS priority, with no asynchronous reaction queue to race
> against `InitiateCast`.

## Goal

DOGMud currently has TWO competing mob-behavior systems:

- **Legacy tactics engine** (`internal/mobai/`) — priority-ordered
  trigger→action rules. 44 mobs use it via `tactic_preset:` or
  inline `tactics:` YAML. Dispatches through a `pendingReactions`
  FIFO queue, which is the root cause of the priority-race bug
  filed in `project_tactics_cast_preemption.md`.
- **Behavior tree (btree)** system
  (`internal/behaviortree/`) — selector/sequence/decorator/
  condition/action node graphs, declared per-archetype. 205 mobs
  use it via `behavior_archetype:`. Active development since
  chunk 2.2; chunks 2.3 and 2.4 added several new primitives.

Most tactics-using mobs (27 of 44) also use a btree archetype.
The dual system creates author-facing confusion ("which one do
I use?"), maintenance overhead, and the actual priority-race
bug that motivated this chunk.

The goal: **delete the tactics engine, migrate all 44 mobs to
btree, retire the dual-system smell.**

## Architectural musts

Brainstorming refined the framing:

1. **Btree subsumes tactics functionally.** Every tactic trigger
   has a btree equivalent already:
   - `health_below:N` → existing `mob_health_below` condition
   - `target_casting` → existing `target_is_casting` condition
   - `target_prone` → existing `target_not_standing` condition
   - `multiple_targets` → existing `multiple_enemies` condition
   - `combat_start → cast X` → btree `mob_combat_round + invert(mob_has_buff) → cast X` (self-gates: once the buff lands, the branch stops firing)
   - `missing_buff:N` → existing `mob_has_buff` + `decorator: invert` (no new primitive)
   - `no_aggro → hide` → existing `mob_in_combat` condition + invert; ambusher archetype already does this
   - `after_action:X → flee` → not directly expressible today but only used by ambusher preset; absorbed via existing ambusher archetype's `mob_hurt → flee`
   - `single_target` → existing `multiple_enemies` + invert
   - `target_grappled` → covered by existing `target_not_standing`
   `submit` action (a grapple yield) is dropped — generic_fighter's existing grapple/trip cascade handles the same situation differently and the loss is negligible.

2. **Zero new btree primitives required.** All trigger semantics
   are expressible with existing conditions/decorators. This is
   the surprise win — the btree library is already complete for
   this scope.

3. **Three preset bundles get DELETED outright, not migrated.**
   - `aggressive_melee` (13 mobs) — its three rules
     (`target_prone→kick`, `target_casting→bash`,
     `target_grappled→submit`) are already in
     `generic_fighter`'s `mob_combat_round` cascade. Mobs using
     this preset already have `generic_fighter` or `leader` as
     their archetype, so dropping the preset is a no-op.
   - `caster_backline` (1 mob) — single user; migrates inline
     into that mob's archetype assignment.

4. **Two preset bundles get FOLDED into existing archetypes.**
   - `tank` (4 mobs) — adds `health_below:20 → call_for_help`
     to `tank_taunter`. Single new branch on the existing
     archetype.
   - `ambusher` (6 mobs) — adds
     `mob_combat_round + target_is_casting → trip` to the
     existing `ambusher` btree archetype. The other two preset
     rules (`after_action:surprise-strike→flee`,
     `no_aggro→hide`) are already covered by the existing
     ambusher archetype's mob_hurt→flee and idle→add_buff
     branches.

5. **`defensive_caster` preset (3 mobs) gets ONE shared new
   archetype, not 3 inline trees.** The three mobs share the
   same pattern (panic-buff → AoE → single-target → panic flee)
   with different spell choices. The shared archetype expresses
   the pattern; each mob's specific spell IDs come from the
   archetype tree referencing well-known spells via per-archetype
   constants. ALTERNATIVELY: this archetype takes spell-id
   params via mob YAML if we add that mechanism. v1 ships
   archetype-with-hardcoded-spells, since all 3 mobs use the
   exact same 4 spells (chrysalis-cocoon, conviction-barrage,
   conviction-spike, flee).

6. **Generic panic-flee branch added to common archetypes.**
   `generic_fighter`, `predator`, `tank_taunter`, `leader`,
   `lookout` all get a shared pattern:
   `mob_hurt + mob_health_below:25 → flee`
   This absorbs the bulk of the 12 generic mobs' inline tactic
   rules. Threshold = 25% is a reasonable default; mobs needing
   different thresholds use their per-boss archetype.

7. **Five per-boss archetypes for named encounter mobs.**
   Mobs with genuinely unique spell rotations get their own
   archetype rather than losing behavior or absorbing into a
   generic one. These are content authoring, not engine work:
   - `boss_edrin` — Edrin (275): fold-recall on HP<30, heal on
     HP<50, flee on HP<25. Plus base combat from generic_fighter.
   - `boss_sylara` — Windwarden Sylara (241): heal on HP<30,
     panic chrysalis-cocoon on missing-buff, opening
     conviction-ward, base from pure_caster.
   - `boss_rhett` — Geomancer Rhett (242): TBD per current
     tactics inspection.
   - `boss_soren` — Soren (286): named bandit leader; tactics
     vary from `leader` archetype.
   - `boss_chrysalis_phantom` — Chrysalis Phantom (272):
     panic-flee + interrupt-trip; differs enough from generic
     `ambusher` to warrant per-mob authoring.

   The per-boss archetypes are concretely distinct (different
   spell IDs, different priorities) and satisfy the
   "don't create virtually identical archetypes" constraint.

8. **Legacy engine deletion is part of the chunk.** Not a
   follow-up. Files removed:
   - `internal/mobai/tactics.go` (~70 lines — presets + EvaluateTactics)
   - `internal/mobai/reactor.go` (~190 lines — pendingReactions queue + Reaction + ProcessPendingReactions)
   - `internal/mobai/actions.go` (~120 lines — ExecuteAction + helpers)
   - Plus tests for all three
   - The `Mob` struct fields at `internal/mobs/mobs.go:121-124`:
     `ReactionDelay float64`, `TacticalDiscipline float64`,
     `TacticPreset string`, `Tactics []mobai.TacticRule`
   - Any imports of `internal/mobai` from hooks
   Roughly **400-500 lines of legacy code removed**.

9. **`reaction_delay` and `tactical_discipline` fields are
   discarded.** Btree already provides perception-scaled
   reaction delays via the action-execution pipeline (see
   `internal/behaviortree/context.md` "Instant vs Delayed
   Action Table"). Discipline-as-randomness is expressible
   via the existing `random_chance` btree condition if any
   future mob needs it; v1 drops the field outright.

10. **The original Edrin priority-race bug becomes
    structurally impossible.** Btree selectors evaluate
    children in YAML-declared order on every relevant signal
    (mob_combat_round, mob_hurt, etc.). There's no
    asynchronous queue, no FIFO race, no `InitiateCast`
    contention. The first child branch whose conditions all
    match and whose actions succeed wins. The
    `surface-silent-exit` logging in `mobcommands/cast.go`
    stays — useful for any future cast-failure diagnostics
    — but the underlying bug it was meant to make
    diagnosable is now gone.

## Architecture & module layout

| File | Status | Responsibility |
|------|--------|----------------|
| `internal/mobai/tactics.go` | DELETE | Presets + EvaluateTactics |
| `internal/mobai/reactor.go` | DELETE | pendingReactions queue, Reaction(), ProcessPendingReactions, signal dispatch |
| `internal/mobai/actions.go` | DELETE | ExecuteAction + helpers (doRetargetStrongest, doCallForHelp, doTrackMemory, doRecall) — the live equivalents are btree actions |
| `internal/mobai/*_test.go` | DELETE | Tests for the deleted files |
| `internal/mobai/types.go` | DELETE or MODIFY | If only used by the engine, delete. If exporting types still used elsewhere (e.g., `TriggerContext`, `Signal`), prune to just what survives. (Likely fully deletable — types are engine-internal.) |
| `internal/mobs/mobs.go` | MODIFY | Remove `Tactics`, `TacticPreset`, `ReactionDelay`, `TacticalDiscipline` fields from `Mob` struct. Remove `mobai.Reaction` and `mobai.ProcessPendingReactions` callers. |
| `internal/hooks/*.go` | MODIFY | Any hook calling into `internal/mobai` gets the call deleted. Likely candidates: a signal-emitting hook (combat-start, action-complete) that feeds the tactics engine. |
| `_datafiles/world/dogmud/behaviors/archetypes/tank_taunter.yaml` | MODIFY | Add `mob_hurt + mob_health_below:20 → call_for_help` branch |
| `_datafiles/world/dogmud/behaviors/archetypes/ambusher.yaml` | MODIFY | Add `mob_combat_round + target_is_casting → trip` branch |
| `_datafiles/world/dogmud/behaviors/archetypes/generic_fighter.yaml` | MODIFY | Add `mob_hurt + mob_health_below:25 → flee` panic-flee branch |
| `_datafiles/world/dogmud/behaviors/archetypes/predator.yaml` | MODIFY | Add same panic-flee branch |
| `_datafiles/world/dogmud/behaviors/archetypes/leader.yaml` | MODIFY | Add same panic-flee branch |
| `_datafiles/world/dogmud/behaviors/archetypes/lookout.yaml` | MODIFY | Add same panic-flee branch |
| `_datafiles/world/dogmud/behaviors/archetypes/defensive_caster.yaml` | NEW | Caster pattern: missing-buff → cocoon; multi-target → barrage; single-target → spike; HP<30 → flee. Base from pure_caster |
| `_datafiles/world/dogmud/behaviors/archetypes/boss_edrin.yaml` | NEW | Edrin's per-boss tree |
| `_datafiles/world/dogmud/behaviors/archetypes/boss_sylara.yaml` | NEW | Sylara's per-boss tree |
| `_datafiles/world/dogmud/behaviors/archetypes/boss_rhett.yaml` | NEW | Rhett's per-boss tree |
| `_datafiles/world/dogmud/behaviors/archetypes/boss_soren.yaml` | NEW | Soren's per-boss tree |
| `_datafiles/world/dogmud/behaviors/archetypes/boss_chrysalis_phantom.yaml` | NEW | Chrysalis Phantom's per-boss tree |
| 44 mob YAMLs across many zone folders | MODIFY | Drop `tactic_preset:`, drop `tactics:`, drop `reaction_delay:`, drop `tactical_discipline:`; reassign `behavior_archetype:` for the per-boss and `defensive_caster` cases |
| `internal/behaviortree/context.md` | MODIFY | Note the panic-flee pattern + the new boss/defensive_caster archetypes |
| `internal/mobs/context.md` | MODIFY | Document that tactics engine is removed; legacy fields gone |
| `MOB_ALIVENESS_ROADMAP.md` | MODIFY | Mark 2.6 Done with the reframed title; roll-up 13/41 → 14/41 |
| `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\project_tactics_cast_preemption.md` | DELETE | Bug resolved by migration; memory note is obsolete |

## New archetypes

### `defensive_caster.yaml`

```yaml
# defensive_caster archetype
#
# Caster pattern with self-preservation: opens with conviction-ward,
# applies chrysalis-cocoon when the cocoon buff is missing, casts
# conviction-barrage on multiple targets, conviction-spike on single
# target, flees at low HP.
#
# Used by: chrysalis-phantom variants and similar caster bosses.
# (Specific mob assignment in chunk 2.6 migration table.)

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

### Per-boss archetypes

Each per-boss archetype follows this template: panic-flee at the boss-specific HP threshold, mob_combat_round cascade with boss-specific spells/abilities. Concrete trees authored during implementation by reading the existing inline tactics on each mob YAML and translating rule-by-rule into btree branches.

**Stand-in templates** (final spell IDs taken from the mobs' existing inline tactics during migration):

- `boss_edrin.yaml` — generic_fighter base + panic-flee at HP<25 + `cast fold-recall` at HP<30 + `cast heal` at HP<50
- `boss_sylara.yaml` — pure_caster base + panic-flee at HP<30 + `cast heal` at HP<30 + missing-buff `cast chrysalis-cocoon` + opening `cast conviction-ward`
- `boss_rhett.yaml` — content authored from current tactics inspection
- `boss_soren.yaml` — leader base + soren-specific behaviors from current tactics
- `boss_chrysalis_phantom.yaml` — ambusher base + extra panic/interrupt rules

The implementation pass will populate each tree by reading the original mob's inline `tactics:` block and writing the equivalent btree branches verbatim.

## Augmented existing archetypes

### Panic-flee branch (added to 5 archetypes)

Shared YAML branch, copied into `generic_fighter`, `predator`, `tank_taunter`, `leader`, `lookout`:

```yaml
    # NEW: panic-flee at critical HP
    - type: sequence
      event: mob_hurt
      children:
        - type: condition
          check: mob_health_below
          percent: 25
        - type: action
          do: flee
```

Inserted as the FIRST child of the top-level selector — emergency flee outranks any combat action.

### tank_taunter additional branch (call_for_help)

```yaml
    # NEW: call for help at low HP (absorbed from the tank tactic preset)
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

Insert after the panic-flee branch (above) but before the mob_combat_round cascade.

### ambusher additional branch (target_casting → trip)

```yaml
    # NEW: interrupt casting targets (absorbed from the ambusher tactic preset)
    - type: sequence
      event: mob_combat_round
      children:
        - type: condition
          check: target_is_casting
        - type: action
          do: command_best_of
          cmds: [trip]
```

Inserted in the existing ambusher tree — place after the hide-related branches but before the catch-all flee on hurt.

## Mob migration table

The 44 mobs categorized by current preset + new archetype assignment.

### aggressive_melee preset (13 mobs) — drop preset only

| Mob | Current `behavior_archetype` | New `behavior_archetype` |
|---|---|---|
| 218-goblin_scrapper | generic_fighter | generic_fighter (unchanged; tactic_preset stripped) |
| 226-deep_gnawer | generic_fighter | generic_fighter |
| 229-windscour_wyrm | generic_fighter | generic_fighter |
| 242-geomancer_rhett | noncombat_questgiver | boss_rhett (overrides — Rhett is also in inline-tactics list) |
| 73-warren_warrior | generic_fighter | generic_fighter |
| 254-bandit_leader | leader | leader |
| 284-bandit_fighter | generic_fighter | generic_fighter |
| 286-soren | leader | boss_soren (overrides — Soren is also in inline-tactics list) |
| 287-bloodline_agent | generic_fighter | generic_fighter |
| 320-elemental_king | leader | leader |
| 322-elemental_prince | leader | leader |
| 324-arena_champion | leader | leader |
| 332-sump_dweller | (none / TBD) | generic_fighter |

(13 entries; some may also appear in inline-tactics — those rows resolved by their boss assignment.)

### tank preset (4 mobs) — keep tank_taunter, drop preset

The augmented `tank_taunter` archetype absorbs the call_for_help rule. Mobs strip the preset.

### ambusher preset (6 mobs) — keep ambusher, drop preset

The augmented `ambusher` archetype absorbs the target_casting → trip rule. Mobs strip the preset.

### defensive_caster preset (3 mobs) — switch to defensive_caster archetype

| Mob | New archetype |
|---|---|
| 219-goblin_shaman | defensive_caster |
| 74-tunnel_shaman | defensive_caster |
| 285-bandit_caster | defensive_caster |

### caster_backline preset (1 mob) — absorb into defensive_caster

| Mob | New archetype |
|---|---|
| 321-elemental_queen | defensive_caster |

The queen is the only mob using `caster_backline`; her behavior fits the same defensive_caster pattern (panic-buff, AoE on multi, single-target spike, flee at low HP). Folding her into the shared defensive_caster archetype eliminates the 1-mob `caster_backline` preset entirely without authoring a dedicated queen archetype.

### Inline-tactics named bosses (5 mobs) — per-boss archetypes

| Mob | New archetype |
|---|---|
| 275-old_edrin | boss_edrin |
| 241-windwarden_sylara | boss_sylara |
| 242-geomancer_rhett | boss_rhett |
| 286-soren | boss_soren |
| 272-chrysalis_phantom | boss_chrysalis_phantom |

### Inline-tactics generic mobs (~12 mobs) — absorbed via augmented archetypes

| Mob | New archetype |
|---|---|
| 262-the_forager | (existing — typically forager or generic_fighter) |
| 80-dustwalk_bandit | generic_fighter (panic-flee absorbs inline) |
| 217-goblin_scout | (existing — scout/lookout?) |
| 224-cave_crawler | generic_fighter / predator |
| 225-pale_lurker | predator / ambusher |
| 227-blind_stalker | predator |
| 253-road_bandit | generic_fighter |
| 72-warren_scout | (lookout / generic_fighter) |
| 75-warren_chieftain | leader (with panic-flee) |
| 78-spore_crawler | predator / generic_fighter |
| 283-bandit_lookout | lookout (already migrated; panic-flee adds) |
| 90-thornwall_highwayman | generic_fighter |
| 331-drowned_hunter | predator / generic_fighter |
| 332-sump_dweller | generic_fighter |

Exact assignments resolved during the migration pass by reading each mob's existing inline tactics and picking the archetype whose combat-cascade + panic-flee branches cover the same behavior. Where a generic mob has unique inline tactics that the augmented archetype doesn't cover, the migration plan flags it for either (a) a small archetype tweak or (b) acceptance of minor behavior loss.

## Engine deletion plan

Order matters — the deletion has to happen AFTER all mobs have migrated. Otherwise the boot panics when it tries to evaluate a `tactic_preset:` field that no longer parses.

1. **All YAML migrations land first.** Every mob YAML is clean of `tactic_preset:`, `tactics:`, `reaction_delay:`, `tactical_discipline:` after this phase. Server still boots with the engine present but unused.
2. **Engine code deletion.** Delete the three Go files + tests. Remove the four `Mob` struct fields (mob YAML loader will then silently ignore stripped fields — they're all already-removed by phase 1, so this is purely a Go-side cleanup).
3. **Hook-layer cleanup.** Remove any `mobai.Reaction(...)` or `mobai.ProcessPendingReactions(...)` callers. Likely in `internal/hooks/NewTurn_*.go` or similar combat-round hooks. Find via `grep -rn "mobai\." internal/`.
4. **Boot validation.** Server starts clean; `go build ./...` clean; no orphaned imports.

## Data flow / signaling — how btree replaces tactic dispatch

The old flow:

```
combat_start signal  →  mobai.Reaction()  →  EvaluateTactics  →
   pendingReactions queue  →  ProcessPendingReactions (delayed)  →
      ExecuteAction  →  mob.Command("cast X")  →  ...
```

The new flow:

```
mob combat round tick  →  behaviortree.TryMobBehavior(mob_combat_round)  →
   archetype tree's selector → first matching sequence/action → ...
mob takes damage  →  TryMobBehavior(mob_hurt)  →  panic-flee branch fires if HP threshold met
mob killed in idle  →  TryMobBehavior(mob_idle)  →  ...
```

Btree is already wired in for `mob_combat_round`, `mob_hurt`, `mob_idle`, `player_enter`, `packmate_hurt`, `heard_callforhelp`, `mob_die`, `player_give`, `player_ask`. No new event types needed. The `combat_start` semantics that tactics used are expressible via:

- `mob_combat_round` selector branches that fire on the first matching condition (e.g., missing-buff opener casts itself only on the first round because the buff lands after the cast)
- OR for true "first-tick-of-combat" detection, an existing BehaviorState mechanism could track it — but none of the migrated mobs actually need this; the missing-buff self-gating covers all opener cases.

## Validation / testing

No new unit tests for the deletions themselves — the absence of a regression is verified by smoke. New tests for:

1. **Augmented archetype branches** — verify the panic-flee + call_for_help + target_casting→trip branches fire under expected conditions. Synthetic tests in `internal/behaviortree/` using existing test helpers (seedTestMob, seedTestRoom).
2. **Per-boss archetype trees parse correctly** — `behaviors.LoadDataFiles()` panics at boot on malformed YAML; smoke test catches it.
3. **No mob YAML still references the legacy engine fields** — boot succeeds, and a grep across `_datafiles/world/dogmud/mobs/` returns zero matches for `tactic_preset:` and `tactics:`.
4. **Original Edrin race no longer reproduces** — manual smoke: spawn Edrin, drop him below 30% HP via admin, confirm fold-recall fires reliably (no longer dropped by an earlier-queued conviction-ward).

## Smoke test

After all migrations + deletions land:

1. `go build ./...` clean
2. `go test ./...` no FAILs (some `internal/mobai/` tests are deleted along with the package)
3. Boot the server:
   - `behaviors.LoadDataFiles()` increments loadedCount by ~6 (defensive_caster + 5 boss archetypes)
   - `mobs.LoadDataFiles()` count unchanged
   - No panics, no warnings about unknown YAML fields
4. Spot-check via admin / AI tester:
   - **Edrin recall:** spawn Edrin, drop HP via admin, observe fold-recall fires
   - **Sylara opener:** spawn Sylara, engage, observe conviction-ward cast on first round
   - **Tank call_for_help:** spawn a tank-archetype mob, drop HP below 20%, observe callforhelp emote/action
   - **Ambusher trip:** spawn a bandit lookout, have a player begin casting in the room, observe trip during combat
   - **Panic-flee universality:** spawn a road bandit, beat it to HP<25, observe flee
5. `grep -rn "mobai\." internal/` returns zero matches outside `_test.go` files
6. `grep -rln "tactic_preset:\|^tactics:" _datafiles/world/dogmud/mobs/` returns zero matches

## Out of scope / deferred

- **Per-boss archetype quality polish.** The 5 boss archetypes faithfully translate the existing inline tactics; tuning the bosses for "actually fun encounter" is a content-pass task, not 2.6's job.
- **`combat_just_started` btree event type.** Not needed for the migration since missing-buff self-gating covers all current openers. Add later if a future encounter wants true one-shot-at-combat-start logic that doesn't ride on a buff.
- **Archetype parameters / spell-id config knobs.** The defensive_caster archetype hardcodes spell IDs (chrysalis-cocoon, conviction-barrage, conviction-spike). If multiple casters with different spell rotations emerge later, we can add a YAML param mechanism then.
- **`submit` action support in btree.** The grapple-yield action from `aggressive_melee` is dropped. The 13 mobs using `aggressive_melee` don't get a 1:1 behavioral replacement for `submit` — generic_fighter's existing grapple cascade is different. Acceptable loss given how rarely `submit` is the right tactical choice.
- **`internal/mobai/` package fully deleted.** If any types from the package are imported elsewhere for non-engine purposes (e.g., shared signal definitions), this chunk doesn't preserve them. Grep first; rescue what's actually used.
- **Behavior extras / per-mob composition.** Rejected during brainstorming. Per-boss archetypes cover the unique cases; the composition mechanism isn't worth the ongoing complexity.
- **Player-side cast feedback changes.** The `surface-silent-exit` debug logging in `mobcommands/cast.go` stays. Players don't see it. Future work could route the same info into a more visible flow for debug builds, but not in 2.6.
- **Tactical_discipline-as-randomness preservation.** Some mobs had `tactical_discipline: 0.85` etc. — a probability of *not* following the tactic. The btree equivalent (`random_chance: 85` condition) isn't backfilled per-mob; bosses can author it explicitly if needed, generic mobs lose the variance. Acceptable.

## Roadmap touchpoints

This chunk:

- Closes chunk **2.6** on `MOB_ALIVENESS_ROADMAP.md`. Roll-up moves from 13/41 → 14/41.
- **Title reframe:** from "Tactics-cast preemption fix" to "Sunset legacy tactics engine" since the original band-aid framing was wrong. Documented in the mini-brief.
- Resolves the open MEMORY follow-up
  `project_tactics_cast_preemption.md` (deleted alongside the
  engine).
- Unblocks future Phase 3 (Routine layer) and Phase 4
  (Strategic layer) chunks — those build on btree as the single
  mob-behavior substrate; with tactics gone, there's no
  competing system to integrate with.
- Reduces ~400-500 lines of legacy code from the project.
