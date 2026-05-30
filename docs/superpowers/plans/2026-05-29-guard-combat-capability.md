# Guard Combat Capability Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the three active Thornwall guards combat-capable and a credible deterrent, while preserving Captain Velk's quest role — so Town Justice 5.1a's attack path and 5.1c's arrest-resist path actually function.

**Architecture:** Two rank-and-file guards swap to the existing `tank_taunter` combat archetype; Captain Velk gets a new hybrid `guard_captain` archetype (tank_taunter combat nodes + noncombat_questgiver dialogue/quest fallthrough). All three switch their stat-distribution `archetype` field to `tank` and get a `statpool` bump. No engine code changes — archetypes auto-load from the data directory.

**Tech Stack:** YAML data files (mob specs, behavior-tree archetype); Go btree loader (read-only); boot + in-game inspection for verification.

**Spec:** `docs/superpowers/specs/2026-05-29-guard-combat-capability-design.md`

**Branch:** `feature/guard-combat-capability` (off master; has 5.1a + 5.1b).

**Verified facts (confirm any path/line by reading before editing):**
- Archetypes auto-load by walking `_datafiles/world/dogmud/behaviors/archetypes/` (`internal/behaviortree/archetype_loader.go`). A new `guard_captain.yaml` there needs NO Go registration.
- Stat value = `Racial + Training + Mods` (`internal/stats/stats.go` `Recalculate`). Human (speciesid 1) `base: 100` per stat (`_datafiles/world/dogmud/species/1-human.yaml`) → guards floor at 100. `statpool` adds Training, distributed by the mob's `archetype` field weighting at `internal/mobs/mobs.go` ~line 447. `tank` weighting: Cha 25% / Vit 20% / Str 15% / Dex 15% / Wil 15% / Per 10%.
- The three in-scope mob YAMLs and current fields:
  - `_datafiles/world/dogmud/mobs/thornwall_city/106-city_guard.yaml` — `archetype: fighting`, `behavior_archetype: noncombat_questgiver`, `statpool: 60`, `schedule_id: thornwall_city_guard_dayshift`, no quests/dialogue.
  - `_datafiles/world/dogmud/mobs/thornwall_outskirts/92-city_gate_guard.yaml` — `archetype: fighting`, `behavior_archetype: noncombat_questgiver`, `statpool: 75`, no quests/dialogue.
  - `_datafiles/world/dogmud/mobs/thornwall_city/94-guard_captain_velk.yaml` — `archetype: fighting`, `behavior_archetype: noncombat_questgiver`, `statpool: 120`, `charm_immune: true`, `maxwander: 0`. **Quest NPC**: quests `10-the_drowning_posts_debt` + `14-the_undertow` deliver items to `mob: 94` via `item_give`.
- Source archetypes to copy from: `tank_taunter.yaml` (combat: `mob_hurt`→flee<15%, `packmate_hurt`→attack, `mob_combat_round` cascade taunt/bash/trip/kick/grapple, `mob_idle`→try_goal_planner, default_goals survival+protect_allies) and `noncombat_questgiver.yaml` (`mob_idle`→try_goal_planner, `player_enter`→emote, `player_attack_rejected`→emote, `player_give`→emote+return_item).
- 335 constable_drunn is OUT of scope.

---

## File Structure

| File | Change |
|------|--------|
| `_datafiles/world/dogmud/behaviors/archetypes/guard_captain.yaml` | NEW hybrid archetype (combat + questgiver) |
| `_datafiles/world/dogmud/mobs/thornwall_city/106-city_guard.yaml` | btree→tank_taunter, archetype→tank, statpool→240 |
| `_datafiles/world/dogmud/mobs/thornwall_outskirts/92-city_gate_guard.yaml` | btree→tank_taunter, archetype→tank, statpool→240 |
| `_datafiles/world/dogmud/mobs/thornwall_city/94-guard_captain_velk.yaml` | btree→guard_captain, archetype→tank, statpool→300 |
| `internal/behaviortree/archetype_guard_captain_test.go` (optional) | load+shape test if the pattern is real (Task 5) |

---

### Task 1: Verify the Velk quest/dialogue × btree interaction (keystone — investigation first)

The hybrid only works if we preserve exactly what Velk needs. Establish that BEFORE authoring the archetype.

- [ ] **Step 1** — Read the quest `item_give` handling path. Find where the quest engine processes an `item_give` step (grep `item_give` in `internal/quests/` or `internal/` broadly) and confirm whether it fires **independent of the mob's behavior_archetype** (i.e., the quest engine sees the give regardless of btree). Read quests `10-the_drowning_posts_debt.yaml` and `14-the_undertow.yaml` to see exactly what Velk must do (receive an item via give; any dialogue node).

- [ ] **Step 2** — Read how `player_ask` dialogue dispatches relative to the btree. The noncombat_questgiver archetype comment says "the btree returns Failure for ask so the dispatcher falls through to dialogue patterns." Confirm that mechanism (grep the dispatcher: `player_ask` handling in `internal/` — likely a hooks or mobcommand path that tries the btree then falls through to dialogue YAML). Determine what the hybrid btree must do on `player_ask` (return Failure / not consume it) to keep dialogue working.

- [ ] **Step 3** — Write findings to the task report:
  - Is `item_give` btree-independent? (expected: yes)
  - What must the hybrid tree include/omit for `player_ask` dialogue + `player_give` to keep working?
  - **If `item_give` or dialogue turns out to be btree-coupled in a way the hybrid can't cleanly preserve, STOP and report BLOCKED** — the captain archetype design needs revisiting.

No commit (investigation only); findings drive Task 3.

---

### Task 2: Rank-and-file guards → tank_taunter (106, 92)

- [ ] **Step 1** — In `_datafiles/world/dogmud/mobs/thornwall_city/106-city_guard.yaml`, change three fields:
  - `archetype: fighting` → `archetype: tank`
  - `behavior_archetype: noncombat_questgiver` → `behavior_archetype: tank_taunter`
  - `statpool: 60` → `statpool: 240`
  Leave `schedule_id`, `idlecommands`, `groups`, equipment, everything else untouched.

- [ ] **Step 2** — In `_datafiles/world/dogmud/mobs/thornwall_outskirts/92-city_gate_guard.yaml`, the same three changes:
  - `archetype: fighting` → `archetype: tank`
  - `behavior_archetype: noncombat_questgiver` → `behavior_archetype: tank_taunter`
  - `statpool: 75` → `statpool: 240`

- [ ] **Step 3** — Commit:
```bash
git add _datafiles/world/dogmud/mobs/thornwall_city/106-city_guard.yaml _datafiles/world/dogmud/mobs/thornwall_outskirts/92-city_gate_guard.yaml
git commit -m "feat(guards): city + gate guard -> tank_taunter, tank stats, statpool 240

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: New hybrid `guard_captain` archetype + Velk swap

**Files:** Create `_datafiles/world/dogmud/behaviors/archetypes/guard_captain.yaml`; modify `94-guard_captain_velk.yaml`.

- [ ] **Step 1** — Create `_datafiles/world/dogmud/behaviors/archetypes/guard_captain.yaml`. Combat reflex/round nodes (copied verbatim from `tank_taunter.yaml` — copy from the actual file to get exact indentation) sit ABOVE the questgiver non-combat nodes. Shape:
```yaml
# guard_captain archetype
#
# Hybrid: a quest-giving guard captain who ALSO fights. Combat nodes
# (from tank_taunter) take priority when engaged; the questgiver
# non-combat nodes (idle goal pursuit, dialogue fallthrough, give
# handling) run when idle so quest item-give + ask-dialogue keep working.
#
# Used by: guard_captain_velk (94). Precursor to Town Justice 5.1c.

tree:
  type: selector
  children:
    # ── COMBAT (from tank_taunter) ──
    # panic-flee at critical HP
    - type: sequence
      event: mob_hurt
      children:
        - type: condition
          check: mob_health_below
          percent: 15
        - type: action
          do: flee

    # packmate_hurt: engage the attacker
    - type: action
      event: packmate_hurt
      do: attack

    # mob_combat_round: taunt → interrupt → kick → knockdown cascade
    - type: selector
      event: mob_combat_round
      children:
        - type: sequence
          children:
            - type: condition
              check: target_is_casting
            - type: action
              do: command_best_of
              cmds: [bash, trip]
        - type: sequence
          children:
            - type: condition
              check: target_not_standing
            - type: action
              do: command_best_of
              cmds: [kick]
        - type: action
          do: command_best_of
          cmds: [bash]
        - type: sequence
          children:
            - type: decorator
              mod: invert
              child:
                type: condition
                check: multiple_enemies
            - type: action
              do: command_best_of
              cmds: [grapple]
        - type: action
          do: command_best_of
          cmds: [trip]

    # taunt maintenance when nothing better
    - type: action
      event: mob_combat_round
      do: command
      cmd: taunt

    # ── QUESTGIVER / IDLE (from noncombat_questgiver) ──
    # strategic goal pursuit (idle)
    - type: action
      event: mob_idle
      do: try_goal_planner

    - type: action
      event: player_enter
      do: emote
      text: "looks up from the duty roster as you enter."

    - type: action
      event: player_attack_rejected
      do: emote
      text: "sets his jaw and says nothing."

    - type: sequence
      event: player_give
      children:
        - type: action
          do: emote
          text: "takes it with a curt nod."
        - type: action
          do: return_item

default_goals:
  - type: survival
    priority: 80
  - type: protect_allies
    priority: 70
```
**IMPORTANT — adjust per Task 1 findings:**
- If Task 1 found that quest `item_give` needs Velk to NOT return the item (the quest engine consumes it), then OMIT the `player_give`→`return_item` node, or gate it — copy whatever the quest flow requires. The `return_item` node above is the questgiver default (decline gifts); a quest-receiver may need to KEEP quest items. **Verify quests 10/14 don't break from return_item before keeping it.** (See the give.go gotcha in CLAUDE.md: give transfers the item before handlers fire; quest `item_give` triggers handle delivery. If returning the item would undo a quest delivery, omit the give node so the quest engine's own handling stands.)
- Confirm `player_ask` is not consumed by any node here (no `player_ask` node → btree returns Failure → dialogue dispatcher falls through, preserving `ask velk`).

- [ ] **Step 2** — In `_datafiles/world/dogmud/mobs/thornwall_city/94-guard_captain_velk.yaml`:
  - `archetype: fighting` → `archetype: tank`
  - `behavior_archetype: noncombat_questgiver` → `behavior_archetype: guard_captain`
  - `statpool: 120` → `statpool: 300`
  Leave `charm_immune`, `maxwander`, equipment, groups untouched.

- [ ] **Step 3** — `gofmt`/YAML lint is N/A; verify YAML parses by booting (Task 4). Commit:
```bash
git add _datafiles/world/dogmud/behaviors/archetypes/guard_captain.yaml _datafiles/world/dogmud/mobs/thornwall_city/94-guard_captain_velk.yaml
git commit -m "feat(guards): hybrid guard_captain archetype + Velk -> combat-capable

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: Boot smoke + stat/combat inspection (tuning)

- [ ] **Step 1** — Wipe instances (SOP): `rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*`. Build (`go build -o dogmud-guardtest.exe .`) and run in background, capturing the log.

- [ ] **Step 2** — Confirm clean boot: the new `guard_captain` archetype loads (grep the log for archetype-load lines or absence of parse errors), all three guard mobs load, `Server Ready`, no panic. A YAML indentation error in `guard_captain.yaml` will panic here — fix and rebuild if so.

- [ ] **Step 3 — stat inspection (tuning):** connect as an admin (or use an admin inspection command) and spawn/inspect each guard's rolled stats. Confirm they land in the "serious deterrent" band (rank-and-file: most stats ~125-160, all ≥ ~120; Velk proportionally higher). If the random roll materially under/overshoots, adjust `statpool` on the affected mob(s) and re-boot. Record the observed stat lines in the task report.

- [ ] **Step 4 — combat check:** attack a guard and confirm combat rounds actually fire and the guard swings back (BUG-3 gone — previously stalled). If feasible, confirm a rank-and-file guard flees at low HP and that taunt/bash appear in combat output.

- [ ] **Step 5 — Velk quest/dialogue check (keystone):** confirm `ask velk` still produces dialogue, and (if a quest item is available) that giving Velk a quest item still advances quest 10 or 14. If dialogue or give is broken, return to Task 3 and adjust the hybrid tree per Task 1 findings.

- [ ] **Step 6** — Kill the server, remove `dogmud-guardtest.exe` and the log. If `statpool` was tuned, commit the adjustment:
```bash
git add _datafiles/world/dogmud/mobs/thornwall_city/106-city_guard.yaml _datafiles/world/dogmud/mobs/thornwall_outskirts/92-city_gate_guard.yaml _datafiles/world/dogmud/mobs/thornwall_city/94-guard_captain_velk.yaml
git commit -m "tune(guards): adjust statpool after boot-inspect

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```
(Skip this commit if no tuning was needed.)

---

### Task 5: Archetype load test (optional — only if the pattern is real)

- [ ] **Step 1** — Check `internal/behaviortree/archetype_noncombat_test.go` and siblings. If they contain real (non-skipped) load+shape assertions, add `internal/behaviortree/archetype_guard_captain_test.go` mirroring that pattern: load `guard_captain` and assert it has the expected combat + idle nodes. If the existing tests are skipped placeholders (boot smoke is the real coverage), SKIP this task — boot smoke (Task 4) already verifies the archetype loads.

- [ ] **Step 2** — If a test was added: `go test ./internal/behaviortree/ -run GuardCaptain` → PASS. Commit:
```bash
git add internal/behaviortree/archetype_guard_captain_test.go
git commit -m "test(guards): guard_captain archetype load+shape

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 6: Final verification

- [ ] **Step 1** — `go build ./...` clean (no Go changed, but confirms nothing broke).
- [ ] **Step 2** — Re-confirm the boot smoke from Task 4 is green on the final state (config/archetype/mobs load, no panic).
- [ ] **Step 3** — Report completion. Note for the user: functional in-game smoke (commit a crime → guard fights → win/lose; full 5.1c readiness) is theirs to run; this build did boot + stat/combat inspection.

---

## Notes for the implementer

- **No Go engine changes expected.** This is data: 3 mob YAMLs + 1 new archetype YAML. The only Go is an optional test (Task 5). If you find yourself editing engine code, stop and reconsider.
- **Task 1 is the keystone** — author the hybrid only after confirming what Velk's quests/dialogue actually require. The `return_item` give node is the riskiest piece (it could undo a quest delivery); verify against quests 10/14 before keeping it.
- **Copy archetype nodes from the real `tank_taunter.yaml`** for exact YAML indentation; don't hand-transcribe from this plan (indentation errors panic at boot).
- **Stat outcomes are random** — the `statpool` numbers are targets; Task 4 inspects and tunes. Don't trust the numbers without the boot-inspect.
- **Followups to log at finish** (MEMORY): this resolves the `project_guard_combat_capability` followup — mark it done; 5.1c can now resume. 335 constable_drunn still needs the same treatment when `stillwater_guards` lands (note in `project_town_justice_5_1_followups`).
- **Don't push** — local-only per project convention; user runs the prod-push SOP separately.
