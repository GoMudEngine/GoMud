# Endgame Combat Calibration (#20 / #21) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Re-tune Cascade Pass (#20 Pass-Apex) and Eastern Highlands (#21 Sentinel) into a graduated solo→duo difficulty ramp, using statpool for threat and `NeverDrops` gear for durability, plus one light "Rouse the Wards" hook on the Sentinel.

**Architecture:** Data-only where possible. New `never_drops` mitigation items equipped on the two bosses (durability axis), `statpool` bumps (threat axis), and one new behavior-tree file for the Sentinel that reuses the proven Warden-Prime telegraph→`summon_companion` pattern. No new engine primitives. Difficulty is dialed by an empirical harness loop against explicit success criteria, cross-checked with DPS/EHP math.

**Tech Stack:** GoMud YAML data files (mobs, items, behaviors); the `/playtest-scenario` multi-agent harness; `id_inventory.py`; local server boot-validation.

**Spec:** `docs/superpowers/specs/2026-07-07-endgame-combat-calibration-2021-design.md`

---

## Key facts verified against current code (2026-07-07)

- **Meirok baseline** (`prod_meirok.yaml`, user 24): HP ~610, stats 93–115,
  weapon-combat 69 / unarmed 57 / spellcasting 51 / rhetoric 55, extra-arms L1
  (triple drowned-claw, mixed physical+magical+conviction attacker), conviction-
  ward shield, rally/warcry/surge self-buffs, summon steppe-spirit + undead
  companions. "1 Meirok + companions" = one such char plus its pets.
- **#20 Pass-Apex** = mob **9541** (`mobs/cascade_pass_road/9541-the_pass_apex.yaml`),
  `aiprofile: predator`, `behavior_archetype: leader`, `statpool: 550`, no
  equipment, no btree. Fauna 9538–9540 @275, 9542 @220.
- **#21 Sentinel** = mob **9552** (`mobs/eastern_highlands/9552-the_sentinel.yaml`),
  `aiprofile: brute`, `behavior_archetype: leader`, `statpool: 1200`, `speciesid: 37`
  (earth-elemental), no equipment, **no btree**. Adds already exist: **Roused Ward
  9550** (`aggressive`, 300), **Watcher-Shard 9551** (`skirmisher`, 300).
- **Nodrop gear pattern** (`items/materials-40000/40227-arc_welded_repair_housing.yaml`):
  `type: <slot>`, `subtype: wearable`, `physical_mitigation`/`magical_mitigation`/
  `conviction_mitigation` ints, `never_drops: true`, `not_salable: true`, weight 0,
  value 0. Equipped via `character.equipment.<slot>.itemid: <id>` (see Core
  Guardian `9562`, body 40225 + neck 40226).
- **Telegraph→summon pattern** (`behaviors/crash_site_interior/9561-warden_prime.yaml`):
  a `sequence` on `event: mob_hurt` → `condition: mob_health_below percent: N` →
  inverted `state_equals` guard → `set_state` → `send_room_text` (the telegraph)
  → `action: do: summon_companion mob_id: N count: 1 base_pool: P hostile: true`.
  **Use `summon_companion`, never `spawn_mob`** (spawn_mob never calls room.AddMob
  → ghost mobs).
- **Next free item ID:** global ≥ **40230** (`id_inventory.py --type items`).
- **Instance-save shadow:** overworld mob stat edits are shadowed by stale saves —
  nuke `mobs.instances/*` before every boot (do NOT touch `shops/`).

---

## Task 0: Baseline analysis & test-rig confirmation

**Files:** none (analysis + environment prep only).

- [ ] **Step 1: Confirm the harness party-run mechanism.**

Read the playtest-scenario skill and confirm a **2-character party run** is
runnable locally (the #22 3-Meirok party is precedent — accounts quester4/5/6 →
chars Vael/Ryn/Doss). Confirm the accounts/chars still exist:

Run: `ls _datafiles/world/dogmud/users/ | sort -n` and grep the Vael/Ryn/Doss
char files for `aiflag`/AI-enabled + gold + roomid.
Expected: three Meirok-clone chars exist and are AI-flagged. If missing, note the
recreate step (clone `prod_meirok.yaml` → users, per the #22 rig) before Task 4.

- [ ] **Step 2: Record the Meirok DPS/EHP baseline.**

From `prod_meirok.yaml`, write a short scratch note (scratchpad, not committed)
capturing: HP ~610, effective mitigation from equipped gear (sum
`physical_mitigation` across equipped slots), triple-claw attack count, and the
spell/taunt channels. This is the yardstick for "is 900 statpool + 30 mitigation
tough for one of these?" — it prevents tuning purely by trial and error.

- [ ] **Step 3: Record starting seed values** (used by later tasks, refined by the
  loops):
  - #20 Apex: `statpool 550 → 900`; nodrop body gear phys/mag/conv mitigation
    **30 / 20 / 20**.
  - #21 Sentinel: `statpool 1200 → 1900`; nodrop body **35 / 30 / 25** + neck
    **0 / 20 / 20**; Rouse-the-Wards at **50% HP**.

  These are explicit starting points, not final values. The loops (Task 4, Task 10)
  move them.

- [ ] **Step 4: Commit the scratch note pointer only if useful** (optional; the
  note itself lives in scratchpad and is not committed).

---

## Task 1: Create the #20 Pass-Apex nodrop mitigation item

**Files:**
- Create: `_datafiles/world/dogmud/items/materials-40000/40230-apex_thick_hide.yaml`

- [ ] **Step 1: Write the item YAML.**

```yaml
itemid: 40230
name: Apex Thick-Hide
namesimple: hide
description: >-
  The layered, scar-thick hide of a beast that has outlived every rival in
  the pass -- gnarled with old wounds gone hard, dense enough to turn a
  blade that would open a lesser animal. It is not armour and was never
  worn; it is simply what a thing this old is made of.
type: body
subtype: wearable
physical_mitigation: 30
magical_mitigation: 20
conviction_mitigation: 20
never_drops: true
not_salable: true
weight: 0.0
value: 0
```

- [ ] **Step 2: Validate the YAML parses.**

Run: `python -c "import yaml,sys; yaml.safe_load(open(r'_datafiles/world/dogmud/items/materials-40000/40230-apex_thick_hide.yaml'))" && echo OK`
Expected: `OK`

- [ ] **Step 3: Commit.**

```bash
git add _datafiles/world/dogmud/items/materials-40000/40230-apex_thick_hide.yaml
git commit -m "content(#20): nodrop mitigation hide for Pass-Apex durability tuning"
```

---

## Task 2: Bump the Pass-Apex statpool and equip the hide

**Files:**
- Modify: `_datafiles/world/dogmud/mobs/cascade_pass_road/9541-the_pass_apex.yaml`

- [ ] **Step 1: Raise statpool 550 → 900.**

Change the top-level `statpool: 550` line to `statpool: 900`.

- [ ] **Step 2: Add an equipment block** under `character:` (mirroring Core
  Guardian 9562). Insert a `character.equipment` block; it must be a sibling of
  `character.name`/`character.items`, at the same indent:

```yaml
  equipment:
    body:
      itemid: 40230
```

- [ ] **Step 3: Validate the mob YAML parses.**

Run: `python -c "import yaml; yaml.safe_load(open(r'_datafiles/world/dogmud/mobs/cascade_pass_road/9541-the_pass_apex.yaml'))" && echo OK`
Expected: `OK`

- [ ] **Step 4: Commit.**

```bash
git add _datafiles/world/dogmud/mobs/cascade_pass_road/9541-the_pass_apex.yaml
git commit -m "tune(#20): Pass-Apex statpool 550->900 + equip nodrop hide (seed)"
```

---

## Task 3: Boot-validate the #20 changes

**Files:** none (validation).

- [ ] **Step 1: Nuke instance saves.**

Run: `rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*`

- [ ] **Step 2: Build a fresh binary** (NOT `go build ./...` — that does not
  produce the runnable exe; the #22 calibration lost two runs to a stale binary):

Run: `go build -o gomud_smoke.exe . && echo BUILT`
Expected: `BUILT`

- [ ] **Step 3: Boot and watch for a clean load.**

Boot `gomud_smoke.exe` (background), watch the log for `items ... loadedCount`,
`mobs ... loadedCount`, `ValidateZoneConsistency errors=0 mode=panic`, and **no
panics** (casing, filepath, duplicate-id, dangling-equipment).
Expected: clean boot, item 40230 loaded, no panic. Kill the server after confirm.

- [ ] **Step 4: No commit** (validation only). If a panic appears, fix the
  offending file and re-run Step 2–3 before proceeding.

---

## Task 4: #20 tuning loop — 1 Meirok + companions vs the Pass-Apex

**Files:** iterates on
`_datafiles/world/dogmud/mobs/cascade_pass_road/9541-the_pass_apex.yaml` and (if
needed) `40230-apex_thick_hide.yaml`.

This is an **empirical loop**, not a one-shot. Repeat Steps 1–4 until the success
criteria hold.

- [ ] **Step 1: Set the test char at the Apex encounter.**

Put one Meirok-clone (Vael) at the Pass-Apex spawn room in Cascade Pass Road
(edit the char's `roomid`/`zone` before boot; the Apex spawns in the high-pass
section, room per `9541` spawninfo — grep `cascade_pass_road/rooms` for `9541`).
Give the char full endgame gold/consumables (it is the Meirok baseline).

- [ ] **Step 2: Run a single-agent harness session** driving that char to engage
  and fight the Apex to a decision (win or death).

Run (per the playtest skill): `/playtest local feature-tester` with a goals file
that says "travel to the Pass-Apex and fight it to the death; report ending HP,
consumables used, and cooldowns burned." Capture the report + `Playtest.Round`
beacons (hp/sp/cp per round).

- [ ] **Step 3: Judge against the success criteria** (from the spec):
  - PASS if: the Meirok **wins** but ends **< ~30% HP** and burned potions/
    cooldowns (shield re-casts, summons, surge).
  - Under-tuned if: win at > 70% HP, no cooldowns → raise statpool (+150 steps)
    and/or mitigation (+10 steps).
  - Over-tuned if: the Meirok **dies** → lower statpool and/or mitigation.
  - Remember the AI plays sub-optimally: land the harness result slightly on the
    "hard" side, and sanity-check the swing math against the Task 0 baseline.

- [ ] **Step 4: Adjust one axis at a time**, re-nuke instances, `go build -o
  gomud_smoke.exe .`, reboot, and repeat from Step 2. **Change threat (statpool)
  and durability (mitigation) separately** so you can attribute the effect.

- [ ] **Step 5: Commit the settled values** once the criteria hold.

```bash
git add _datafiles/world/dogmud/mobs/cascade_pass_road/9541-the_pass_apex.yaml _datafiles/world/dogmud/items/materials-40000/40230-apex_thick_hide.yaml
git commit -m "tune(#20): Pass-Apex calibrated tough-but-winnable for 1 Meirok+companions"
```

---

## Task 5: Create the #21 Sentinel nodrop mitigation items

**Files:**
- Create: `_datafiles/world/dogmud/items/materials-40000/40231-sentinel_grey_carapace.yaml`
- Create: `_datafiles/world/dogmud/items/materials-40000/40232-sentinel_warding_core.yaml`

- [ ] **Step 1: Write the body carapace (40231).**

```yaml
itemid: 40231
name: Sentinel Grey-Carapace
namesimple: carapace
description: >-
  A shell of the same seamless grey as the buried thing the Sentinel guards
  -- denser than stone, older than reckoning, and cold to a degree that has
  nothing to do with the vault. Blows land on it and stop, as though the
  material simply refuses the idea of being broken.
type: body
subtype: wearable
physical_mitigation: 35
magical_mitigation: 30
conviction_mitigation: 25
never_drops: true
not_salable: true
weight: 0.0
value: 0
```

- [ ] **Step 2: Write the warding core (40232).**

```yaml
itemid: 40232
name: Sentinel Warding-Core
namesimple: core
description: >-
  A second, deeper socket set behind the Sentinel's watching eye, banked
  with the same cold fire -- a ward turned inward, bleeding off the force of
  spell and shout and conviction alike before it can reach the frame it
  protects.
type: neck
subtype: wearable
physical_mitigation: 0
magical_mitigation: 20
conviction_mitigation: 20
never_drops: true
not_salable: true
weight: 0.0
value: 0
```

- [ ] **Step 3: Validate both parse.**

Run: `for f in 40231-sentinel_grey_carapace 40232-sentinel_warding_core; do python -c "import yaml; yaml.safe_load(open(r'_datafiles/world/dogmud/items/materials-40000/'+'$f'+'.yaml'))"; done && echo OK`
Expected: `OK`

- [ ] **Step 4: Commit.**

```bash
git add _datafiles/world/dogmud/items/materials-40000/40231-sentinel_grey_carapace.yaml _datafiles/world/dogmud/items/materials-40000/40232-sentinel_warding_core.yaml
git commit -m "content(#21): nodrop mitigation gear for Sentinel durability tuning"
```

---

## Task 6: Bump the Sentinel statpool and equip its gear

**Files:**
- Modify: `_datafiles/world/dogmud/mobs/eastern_highlands/9552-the_sentinel.yaml`

- [ ] **Step 1: Raise statpool 1200 → 1900.**

Change `statpool: 1200` to `statpool: 1900`.

- [ ] **Step 2: Add the equipment block** under `character:` (sibling of
  `character.items`/`character.name`, same indent as `stats:`):

```yaml
  equipment:
    body:
      itemid: 40231
    neck:
      itemid: 40232
```

- [ ] **Step 3: Validate the mob YAML parses.**

Run: `python -c "import yaml; yaml.safe_load(open(r'_datafiles/world/dogmud/mobs/eastern_highlands/9552-the_sentinel.yaml'))" && echo OK`
Expected: `OK`

- [ ] **Step 4: Commit.**

```bash
git add _datafiles/world/dogmud/mobs/eastern_highlands/9552-the_sentinel.yaml
git commit -m "tune(#21): Sentinel statpool 1200->1900 + equip nodrop gear (seed)"
```

---

## Task 7: Write the Sentinel "Rouse the Wards" behavior tree

**Files:**
- Create: `_datafiles/world/dogmud/behaviors/eastern_highlands/9552-the_sentinel.yaml`

The btree filename must equal the mobid (`9552`). This reuses the Warden-Prime
telegraph→`summon_companion` pattern verbatim, state-gated so it fires once.

- [ ] **Step 1: Write the tree.**

```yaml
# The Sentinel -- Eastern Highlands (#21) duo optional-boss.
# ONE light hook: "Rouse the Wards." At 50% HP the Sentinel telegraphs,
# then summons its two dormant guardians (Roused Ward 9550, Watcher-Shard
# 9551) to split a duo's focus. State-gated so it fires exactly once even
# if the Sentinel's HP crosses 50% more than once. Modeled on
# behaviors/crash_site_interior/9561-warden_prime.yaml (proven pattern:
# summon_companion, NOT spawn_mob -- the latter never calls room.AddMob
# and produces ghost mobs).
tree:
  type: selector
  children:

    # ── Rouse the Wards: one-shot telegraphed add-summon at 50% HP ──
    - type: sequence
      event: mob_hurt
      children:
        - type: condition
          check: mob_health_below
          percent: 50
        - type: decorator
          mod: invert
          child:
            type: condition
            check: state_equals
            key: wards_roused
            value: "true"
        - type: action
          do: set_state
          key: wards_roused
          value: "true"
        - type: action
          do: send_room_text
          text: >-
            The light in the Sentinel's socket flares white and splits --
            and along the vault walls two banked guardians grind awake and
            drop from their niches to stand between you and the thing it
            keeps.
        - type: action
          do: summon_companion
          mob_id: 9550
          count: 1
          base_pool: 300
          hostile: true
        - type: action
          do: summon_companion
          mob_id: 9551
          count: 1
          base_pool: 300
          hostile: true
```

- [ ] **Step 2: Validate the YAML parses.**

Run: `python -c "import yaml; yaml.safe_load(open(r'_datafiles/world/dogmud/behaviors/eastern_highlands/9552-the_sentinel.yaml'))" && echo OK`
Expected: `OK`

- [ ] **Step 3: Commit.**

```bash
git add _datafiles/world/dogmud/behaviors/eastern_highlands/9552-the_sentinel.yaml
git commit -m "feat(#21): Sentinel 'Rouse the Wards' btree hook (50% HP add-summon)"
```

---

## Task 8: Ensure the wards come from the rouse, not ambient vault spawn

**Files:**
- Possibly modify: the Sentinel's vault room file(s) in
  `_datafiles/world/dogmud/rooms/eastern_highlands/` that carry `spawninfo` for
  9550 / 9551.

- [ ] **Step 1: Find where 9550/9551 currently spawn.**

Run: `grep -rn "9550\|9551" _datafiles/world/dogmud/rooms/eastern_highlands/`
Expected: identify whether the Roused Ward / Watcher-Shard are pre-spawned in the
Sentinel's vault room (making the fight start 3-up) or only referenced elsewhere.

- [ ] **Step 2: Decide + apply.**
  - If 9550/9551 are pre-spawned **in the Sentinel's vault room**, remove those
    two `spawninfo` entries from that room so the adds arrive only via the rouse
    (cleaner theater, and avoids a 3-mob opening the tuning didn't account for).
    Leave any 9550/9551 spawns in *other* rooms untouched.
  - If they are not pre-spawned in the vault, no change — note it and move on.

- [ ] **Step 3: Validate any edited room YAML parses** (if changed).

Run: `python -c "import yaml; yaml.safe_load(open(r'<edited room path>'))" && echo OK`
Expected: `OK`

- [ ] **Step 4: Commit** (only if a room was edited).

```bash
git add _datafiles/world/dogmud/rooms/eastern_highlands/<edited>.yaml
git commit -m "content(#21): wards arrive via Sentinel rouse, not ambient vault spawn"
```

---

## Task 9: Boot-validate the #21 changes

**Files:** none (validation).

- [ ] **Step 1: Nuke instance saves.**

Run: `rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*`

- [ ] **Step 2: Build fresh.**

Run: `go build -o gomud_smoke.exe . && echo BUILT`
Expected: `BUILT`

- [ ] **Step 3: Boot and watch for clean load.**

Boot `gomud_smoke.exe`, watch for `items ... loadedCount` (40231/40232 loaded),
`mobs`, behavior-tree load for 9552, `ValidateZoneConsistency errors=0
mode=panic`, and **no panics**. Kill after confirm.
Expected: clean boot, no panic, Sentinel btree loaded.

- [ ] **Step 4: No commit** (validation only). Fix + re-run on any panic.

---

## Task 10: #21 tuning loop — 2 Meiroks + companions vs the Sentinel (+ solo control)

**Files:** iterates on `9552-the_sentinel.yaml`, `40231`/`40232`, and the btree
threshold if the rouse timing feels off.

Empirical loop — repeat until criteria hold.

- [ ] **Step 1: Stage a 2-Meirok party** (Vael + Ryn) at the Sentinel's vault,
  full endgame gold/consumables.

- [ ] **Step 2: Run a 2-agent party harness session** (`/playtest-scenario`,
  party mode) with a goals file: "as a party, fight the Sentinel to a decision;
  report each member's ending HP, whether the wards roused at ~50%, consumables/
  cooldowns used, and win/wipe." Capture reports + beacons.

- [ ] **Step 3: Judge against the success criteria:**
  - PASS if: the duo **wins** but ends **< ~30% HP** (at least one member low),
    burned cooldowns, and the **Rouse-the-Wards** beat fired and mattered.
  - Under/over-tuned: adjust statpool (±200 steps) and/or mitigation (±10) — one
    axis at a time. If the rouse fires too late/early, adjust the btree `percent`.

- [ ] **Step 4: Run the solo control** (one Meirok, same encounter) at the
  settled values. Expected: the solo **loses or must disengage** — this confirms
  the lower bound (the duo gate is real). If a solo Meirok wins comfortably, the
  Sentinel is under-tuned for a duo gate; raise durability and re-check the duo.

- [ ] **Step 5: Commit the settled values.**

```bash
git add _datafiles/world/dogmud/mobs/eastern_highlands/9552-the_sentinel.yaml _datafiles/world/dogmud/items/materials-40000/40231-sentinel_grey_carapace.yaml _datafiles/world/dogmud/items/materials-40000/40232-sentinel_warding_core.yaml _datafiles/world/dogmud/behaviors/eastern_highlands/9552-the_sentinel.yaml
git commit -m "tune(#21): Sentinel calibrated tough-but-winnable for 2 Meiroks+companions"
```

---

## Task 11: Final verification, docs, and SOP memory

**Files:**
- Modify: `docs/superpowers/specs/2026-07-07-endgame-combat-calibration-2021-design.md`
  (record the final settled numbers).
- Modify: `PATCH_NOTES.md` (only when the user later pushes — flag, don't push here).

- [ ] **Step 1: Full clean boot** with both zones' final values.

Run: `rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/* && go build -o gomud_smoke.exe . && echo BUILT`
Then boot and confirm `errors=0 mode=panic`, both zones' items/mobs/btree loaded,
no panic.
Expected: clean.

- [ ] **Step 2: Record the final numbers** in the spec's "Open items" section
  (Apex statpool + mitigation; Sentinel statpool + mitigation + rouse threshold),
  turning them from "empirical" into the shipped record.

```bash
git add docs/superpowers/specs/2026-07-07-endgame-combat-calibration-2021-design.md
git commit -m "docs(calibration): record settled #20/#21 tuning values"
```

- [ ] **Step 3: Author the endgame-combat tuning reference doc** (user request,
  2026-07-07). Create `docs/ENDGAME_COMBAT_TUNING.md` — a best-practices author
  guide that establishes **"Meirok" as the standard difficulty unit** and the
  nodrop-gear method as the standing way to build endgame fights. It MUST contain:
  - **The difficulty unit.** One geared **Meirok + companions** = 1 unit (define
    it from `prod_meirok.yaml`: HP, stat band, key skills, the companion kit).
    A fight is specified as targeting **N Meiroks** (1 / 2 / 3 …).
  - **The two-axis method.** `statpool` sets threat (damage/accuracy);
    `NeverDrops` mitigation gear sets durability (EHP). **Never `base_pool`** (it
    scales melee and turns supports into killers). Include the nodrop-item recipe
    (the `never_drops`/`not_salable`/mitigation-fields shape) and the equip block.
  - **The three empirical anchors** — the settled numbers from this effort and
    #22, as the baseline table to interpolate from going forward:
    | Target | Encounter | statpool | nodrop mitigation | mechanic |
    |--------|-----------|----------|-------------------|----------|
    | 1 Meirok | #20 Pass-Apex | *(settled)* | *(settled)* | none |
    | 2 Meirok | #21 Sentinel | *(settled)* | *(settled)* | 1 light hook |
    | 3 Meirok | #22 Core Guardian | *(from #22)* | *(from #22)* | full apparatus |
  - **Success criteria** (win-but-under-~30%, one-smaller-loses) and the
    **mechanical-depth ramp** (0 hooks → 1 light hook → full apparatus).
  - **The test method** — the manual N-bridge conductor rig (one driver
    puppeteering N geared quester chars round-by-round; N = target Meirok count),
    the instance-save nuke, and the "AI plays sub-optimally → tune to the hard
    side" caveat.
  Fill the *(settled)* cells with the actual Task 4 / Task 10 results, and pull
  the #22 Core Guardian row from its shipped values (statpool ×7 × gold, its
  nodrop gear 40225/40226). Commit:

```bash
git add docs/ENDGAME_COMBAT_TUNING.md
git commit -m "docs: endgame-combat tuning reference (Meirok-unit + nodrop-gear method)"
```

- [ ] **Step 4: Bank the nodrop-gear SOP + Meirok-unit metric to memory.** Update
  `MEMORY.md` + a topic file so the two-axis method (statpool = threat,
  `never_drops` gear = durability, never `base_pool`) and the Meirok-multiple
  difficulty unit are standing practice, pointing at `docs/ENDGAME_COMBAT_TUNING.md`
  as the reference. Cross-link the crashsite + zone-expansion project memories.

- [ ] **Step 5: Flag pre-push, do NOT push.** Note in the final report that a prod
  push is owed and gated behind the pre-push SOP (PATCH_NOTES entry,
  `Logging.LogToFile: false`, boot-test, droplet deploy + perf datapoint). The
  user runs the push.

---

## Notes on execution order & dependencies

- Tasks 1–4 (#20) and Tasks 5–10 (#21) are independent zones; do #20 fully first
  (simpler, proves the rig) then #21.
- Tasks 1, 5, 7 (create files) can be authored up front; the **tuning loops
  (4, 10) are the real time sink** and cannot be parallelized against each other
  cleanly (shared server boot). Run them serially.
- No new Go code, so no Go tests — validation is boot-clean + harness runs.
- If the harness party path (Task 0 Step 1) turns out not to support a 2-agent
  run, fall back to: tune the Sentinel against a single Meirok scaled by the
  measured duo-DPS/EHP multiplier from the Task 0 baseline, and note the reduced
  confidence in the final report.
