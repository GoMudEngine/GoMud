# Trail-Quest Per-Step `map_target` Pass — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `map_target` to every non-terminal step of the seven Pothole Coulee trail quests so the web minimap's focused-quest marker is always lit and points at the room where the player makes progress.

**Architecture:** Content-only. Each quest step gains one `map_target: <roomid>` line, consumed by the existing `questengine.ResolveQuestTarget` → `Char.Quests` GMCP payload → web `RoomGridSVG` marker. No Go code changes. Follows the exemplar already in `32-first_blood.yaml` (`start` step, `map_target: 5227`).

**Tech Stack:** YAML quest data files under `_datafiles/world/dogmud/quests/`; Go server boot for the load check; the `mudagent` playtest harness for the GMCP smoke.

**Spec:** `docs/superpowers/specs/2026-07-19-trail-quest-map-targets-design.md`

**Placement convention:** `map_target:` is indented 4 spaces (aligned with `description:`) and placed immediately after the step's `- id:` line, exactly as in `32-first_blood.yaml:10-12`.

---

### Task 1: First Blood (32) — mark the four mid-steps

The `start` step already has `map_target: 5227`. Add the same target to the four
in-yard steps so the marker stays lit on the Drill Yard through the turn-in.
The `end` step gets none (terminal).

**Files:**
- Modify: `_datafiles/world/dogmud/quests/32-first_blood.yaml`

- [ ] **Step 1: Add `map_target: 5227` to the `strike` step**

Find:
```yaml
  - id: strike
    description: "You have traded blows with the training dummy and
```
Replace with:
```yaml
  - id: strike
    map_target: 5227
    description: "You have traded blows with the training dummy and
```

- [ ] **Step 2: Add `map_target: 5227` to the `special` step**

Find:
```yaml
  - id: special
    description: "You landed a special move on the dummy. Kicks and
```
Replace with:
```yaml
  - id: special
    map_target: 5227
    description: "You landed a special move on the dummy. Kicks and
```

- [ ] **Step 3: Add `map_target: 5227` to the `consider` step**

Find:
```yaml
  - id: consider
    description: "You sized up the training dummy. Consider tells you
```
Replace with:
```yaml
  - id: consider
    map_target: 5227
    description: "You sized up the training dummy. Consider tells you
```

- [ ] **Step 4: Add `map_target: 5227` to the `verbosity` step**

Find:
```yaml
  - id: verbosity
    description: "You have tuned your combat output. The Drill Yard
```
Replace with:
```yaml
  - id: verbosity
    map_target: 5227
    description: "You have tuned your combat output. The Drill Yard
```

---

### Task 2: First Heat (35) — forge (5245)

Both non-terminal steps happen at the forge where Smith Rusk stands (room 5245).

**Files:**
- Modify: `_datafiles/world/dogmud/quests/35-first_heat.yaml`

- [ ] **Step 1: Add `map_target: 5245` to the `start` step**

Find:
```yaml
  - id: start
    description: "The forge is where iron learns its shape. Buy an iron
```
Replace with:
```yaml
  - id: start
    map_target: 5245
    description: "The forge is where iron learns its shape. Buy an iron
```

- [ ] **Step 2: Add `map_target: 5245` to the `craft` step**

Find:
```yaml
  - id: craft
    description: "You worked iron at the forge for the first time
```
Replace with:
```yaml
  - id: craft
    map_target: 5245
    description: "You worked iron at the forge for the first time
```

---

### Task 3: First Brew (38) — Birna's pool (5265)

All three non-terminal steps point at Birna's bench room. Note this file uses
`>-` block scalars, so the line after `- id:` is `description: >-`.

**Files:**
- Modify: `_datafiles/world/dogmud/quests/38-first_brew.yaml`

- [ ] **Step 1: Add `map_target: 5265` to the `start` step**

Find:
```yaml
  - id: start
    description: >-
      The alchemy bench at the sheltered pool is where brewing begins.
```
Replace with:
```yaml
  - id: start
    map_target: 5265
    description: >-
      The alchemy bench at the sheltered pool is where brewing begins.
```

- [ ] **Step 2: Add `map_target: 5265` to the `brew` step**

Find:
```yaml
  - id: brew
    description: >-
      You ground the root, steeped it, and sealed a healing salve in the
```
Replace with:
```yaml
  - id: brew
    map_target: 5265
    description: >-
      You ground the root, steeped it, and sealed a healing salve in the
```

- [ ] **Step 3: Add `map_target: 5265` to the `drink` step**

Find:
```yaml
  - id: drink
    description: >-
      The salve went to work, warm and steady. Tell Birna how it felt.
```
Replace with:
```yaml
  - id: drink
    map_target: 5265
    description: >-
      The salve went to work, warm and steady. Tell Birna how it felt.
```

---

### Task 4: First Sign (41) — Open Steppe (5284)

All three non-terminal steps mark the steppe work-area (both objectives act
there; Tarn is one room away at 5283). Deliberate exception to the turn-in rule
— see the spec's "First Sign asymmetry" note.

**Files:**
- Modify: `_datafiles/world/dogmud/quests/41-first_sign.yaml`

- [ ] **Step 1: Add `map_target: 5284` to the `start` step**

Find:
```yaml
  - id: start
    description: >-
      The steppe is your first lesson in reading the ground. Track a
```
Replace with:
```yaml
  - id: start
    map_target: 5284
    description: >-
      The steppe is your first lesson in reading the ground. Track a
```

- [ ] **Step 2: Add `map_target: 5284` to the `track` step**

Find:
```yaml
  - id: track
    description: >-
      You picked up the hare's trail and followed it across the steppe.
```
Replace with:
```yaml
  - id: track
    map_target: 5284
    description: >-
      You picked up the hare's trail and followed it across the steppe.
```

- [ ] **Step 3: Add `map_target: 5284` to the `forage` step**

Find:
```yaml
  - id: forage
    description: >-
      You worked the steppe for forage. Track a hare if you have not yet.
```
Replace with:
```yaml
  - id: forage
    map_target: 5284
    description: >-
      You worked the steppe for forage. Track a hare if you have not yet.
```

---

### Task 5: First Casting (44) — Star Chamber (5305) then Grieve's hall (5303)

The genuine two-room case: cast at the Star Chamber, then walk back down to
Grieve in the Observatory Hall.

**Files:**
- Modify: `_datafiles/world/dogmud/quests/44-first_casting.yaml`

- [ ] **Step 1: Add `map_target: 5305` to the `start` step**

Find:
```yaml
  - id: start
    description: >-
      The Star Chamber is where you first turn the mind. Stand there and
```
Replace with:
```yaml
  - id: start
    map_target: 5305
    description: >-
      The Star Chamber is where you first turn the mind. Stand there and
```

- [ ] **Step 2: Add `map_target: 5303` to the `cast` step**

Find:
```yaml
  - id: cast
    description: >-
      You shaped a spell and loosed it at the mark. Return to Grieve in the
```
Replace with:
```yaml
  - id: cast
    map_target: 5303
    description: >-
      You shaped a spell and loosed it at the mark. Return to Grieve in the
```

---

### Task 6: First Words (47) — Wenna's farmstead (5324)

Both non-terminal steps point at Wenna (the `help` command works anywhere; the
marker guides the player back to her for the hand-in).

**Files:**
- Modify: `_datafiles/world/dogmud/quests/47-first_words.yaml`

- [ ] **Step 1: Add `map_target: 5324` to the `start` step**

Find:
```yaml
  - id: start
    description: >-
      The way of the folk is mostly the way of asking. Hear Elder Wenna out
```
Replace with:
```yaml
  - id: start
    map_target: 5324
    description: >-
      The way of the folk is mostly the way of asking. Hear Elder Wenna out
```

- [ ] **Step 2: Add `map_target: 5324` to the `help` step**

Find:
```yaml
  - id: help
    description: >-
      You opened the help reference -- your surest guide when memory fails.
```
Replace with:
```yaml
  - id: help
    map_target: 5324
    description: >-
      You opened the help reference -- your surest guide when memory fails.
```

---

### Task 7: First Shot (50) — Long Terrace (5351) then Iden's range (5350)

`start` and `reload` point at the Long Terrace (shoot objective); `shoot` points
back at Iden for the hand-in.

**Files:**
- Modify: `_datafiles/world/dogmud/quests/50-first_shot.yaml`

- [ ] **Step 1: Add `map_target: 5351` to the `start` step**

Find:
```yaml
  - id: start
    description: >-
      You have a sling -- equip it, get a pouch of shot, reload it, and
```
Replace with:
```yaml
  - id: start
    map_target: 5351
    description: >-
      You have a sling -- equip it, get a pouch of shot, reload it, and
```

- [ ] **Step 2: Add `map_target: 5351` to the `reload` step**

Find:
```yaml
  - id: reload
    description: >-
      You chambered a stone. Now loose it at the practice butt down-range if
```
Replace with:
```yaml
  - id: reload
    map_target: 5351
    description: >-
      You chambered a stone. Now loose it at the practice butt down-range if
```

- [ ] **Step 3: Add `map_target: 5350` to the `shoot` step**

Find:
```yaml
  - id: shoot
    description: >-
      You put a stone into the butt. Make sure you have worked the reload
```
Replace with:
```yaml
  - id: shoot
    map_target: 5350
    description: >-
      You put a stone into the butt. Make sure you have worked the reload
```

---

### Task 8: Boot test — confirm the seven quests still load

**Files:** none (verification only)

- [ ] **Step 1: Nuke instance saves (SOP) so template edits are not shadowed**

Run:
```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* \
       _datafiles/world/dogmud/rooms.instances/*
```

- [ ] **Step 2: Boot the server and capture the quest-load line**

Run (windowless, safe to run directly on this Windows box):
```bash
go run . 2>&1 | tee /c/tmp/trailquest-boot.log &
sleep 45 && grep -E "quests.LoadDataFiles\(\) loadedCount|panic" /c/tmp/trailquest-boot.log
```
Expected: a `quests.LoadDataFiles() loadedCount=<N>` line with the same count as
before this change (map_target is a field addition, not a new quest), and **no
`panic`**. A YAML indent typo would panic here with a decode/`Filepath()` error.

- [ ] **Step 3: Stop the server**

Run:
```bash
taskkill //F //IM GoMud.exe 2>/dev/null; taskkill //F //IM go.exe 2>/dev/null; true
```
(Do NOT run this while a background feel-tester is live — none is at this point.)

---

### Task 9: Commit the seven edits

**Files:** the seven quest YAMLs from Tasks 1-7.

- [ ] **Step 1: Stage and commit**

Run:
```bash
git add _datafiles/world/dogmud/quests/32-first_blood.yaml \
        _datafiles/world/dogmud/quests/35-first_heat.yaml \
        _datafiles/world/dogmud/quests/38-first_brew.yaml \
        _datafiles/world/dogmud/quests/41-first_sign.yaml \
        _datafiles/world/dogmud/quests/44-first_casting.yaml \
        _datafiles/world/dogmud/quests/47-first_words.yaml \
        _datafiles/world/dogmud/quests/50-first_shot.yaml
git commit -m "content(quests): per-step map_target on the 7 newbie trail quests

Lights the web minimap focused-quest marker on every step of First Blood,
Heat, Brew, Sign, Casting, Words, and Shot (only First Blood's start step
carried one before). Objective steps target the action room; turn-in steps
target the trailhead NPC. First Sign marks the shared steppe on all steps
(parallel objectives). Content-only; ResolveQuestTarget already reads the field.

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 10: GMCP smoke + adversarial content playtest (harness gate)

The ASCII harness cannot see the SVG marker, but it CAN read the outbound
`Char.Quests` payload's `target_room`, and it must confirm no prose/step
regression. This is the automated portion of the content-playtest SOP gate.

**Files:** none (verification only)

- [ ] **Step 1: Run the route-2 newcomer bug-finder playtest**

Run:
```
/playtest local bug-finder
```
with an explicitly adversarial, critical mandate: spawn a fresh character, take
the footing hand-off so all seven trail quests are granted, focus each trail in
turn, and for each focused quest read the `Char.Quests` GMCP payload and confirm
`target_room` matches the plan table (32→5227, 35→5245, 38→5265, 41→5284,
44 start→5305, 47→5324, 50 start→5351). Report every prose/step/pacing problem
bluntly.

- [ ] **Step 2: Fix anything the harness surfaces, re-run if needed**

If the smoke shows a wrong `target_room`, a load issue, or a regression,
correct the offending YAML, re-boot (Task 8), and re-run this task. Only proceed
once the payloads match and no regression remains.

---

### Task 11: Hand off to the user for the browser playtest (visual gate)

**Files:** none (handoff).

- [ ] **Step 1: Report status and request the browser walk-through**

The marker glyph, patina destination pin, and gold next-step arrow render only
in the web client and are invisible to the harness (SVG-blind, and it cannot
send inbound `Char.Quests.Focus`). Tell the user the branch is ready and ask
them to walk a fresh character through a couple of trails in the browser to
confirm each marker lights and the arrow points correctly — the same visual gate
the original minimap-marker feature used. Do not claim the feature "done" on the
boot + GMCP smoke alone.

---

## Notes for the implementer

- **Instance-save SOP:** always nuke `mobs.instances/*` + `rooms.instances/*`
  before a smoke test (Task 8 Step 1). Never delete `shops/` or `guilds/`.
- **Windows env:** `go run .` / server / `git` runs are windowless and fine to
  run directly from the main loop; do not spawn shell-denied subagents for them.
- **No prose changes:** this pass adds only `map_target` lines. If you find
  yourself editing a `description:` or `hint:`, stop — that is out of scope.
