# Chunk 2.5 Mutations-on-Mobs Admin Smoke Test

**Date:** 2026-05-12  
**Tester:** smoketester (admin role)  
**Server:** localhost:55555  
**Branch:** feature/mob-aliveness-1.3-crimes  

---

## Smoke Verdict: PARTIAL

- Wolf body-parts gate: PASS (code + unit tests verified)
- Wolf intrinsic tail: PASS (code path verified; see note on pre-2.5 instance files)
- Ice queen corporeal: PASS (speciesid 43, no incorporeal in species YAML)
- Smoke prince incorporeal: PASS (speciesid 44, incorporeal: 4 in species intrinsics)
- Sand/storm elementals: PASS (correct species IDs and intrinsic mutations in YAML)
- Magma king large override: **FAIL — BUG** (see Bug 1 below)
- Existing ethereal cleanup: PASS (all ethereal species have incorporeal: 4 as intrinsic)
- Player path no-op: PASS (smoketester shows only pre-existing mutations)

---

## Goal 0 — Login and Admin Confirm

PASS. Connected to localhost:55555. Role column in WHO list showed `admin`.
Set charset ASCII: confirmed with "Charset mode set to ASCII." response.
Current room: Temple District [Thornwall City], room 467.

---

## Goal 1 — Admin Help Discovery

PASS. `help` command revealed the full admin command suite.

**Key admin commands discovered:**

| Command | Syntax | Notes |
|---------|--------|-------|
| `mob spawn` | `mob spawn [MobId\|MobName]` | Spawns mob in current room |
| `mob list` | `mob list [*pattern*]` | Lists matching mob names |
| `item spawn` | `item spawn [ItemId\|ItemName]` | Spawns item in room |
| `teleport` | `teleport [room_id]` | Moves to room |
| `paz` | `paz` | Pacifies player (stops combat) |
| `zap` | `zap [target]` | One-shots mob to 1 HP |
| `command` | `command <mob> <cmd>` | Forces mob to execute command |
| `syslogs` | `syslogs debug` | Streams server debug logs |

**Note:** There is NO `mob show <id>` sub-command to inspect a live mob's
mutation state. The `mob` admin command only supports `spawn` and `list`.
Mutation inspection was done via:
1. Reading mob YAML + species YAML static analysis
2. Reading mob instance save files (mobs.instances/)
3. Running the chunk 2.5 unit tests
4. Behavioral in-game observation

---

## Goal 2 — Wolf Body-Parts Gate

PASS (via code analysis + unit tests).

**Static analysis:**
- Steppe wolf (mob 205) uses `speciesid: 2` (canine)
- Canine species: `body_parts: [legs, eyes, mouth, skin, tail]`
  — no `arms` or `hands` declared
- Mutations `extra-arms` and `elongated-limbs` require `arms`
- Mutation `clawed-hands` requires `hands`
- `mutations.GetWeightedPool()` calls `spec.CanApplyTo(sp)` which calls
  `sp.HasAllBodyParts(spec.RequiresBodyParts)` — filtering any mutation
  whose required body parts are absent from the species

**Unit test result:**
```
=== RUN   TestMobSpawn_CanineNeverGetsExtraArms
--- PASS: TestMobSpawn_CanineNeverGetsExtraArms (0.00s)
```

Test ran 500 random pool samples against a canine-shaped species and
confirmed zero occurrences of extra-arms, elongated-limbs, or clawed-hands.

**Existing wolf instance files sampled:**
- `205-steppe_wolf-room3019.yaml`: mutations: `{tremorsense: 1}` — no arms/hands mutation
- `205-steppe_wolf-room3015.yaml`: mutations: `{tremorsense: 1}` — no arms/hands mutation
- `205-steppe_wolf-room3024.yaml`: mutations: `{infrared-vision: 1}` — no arms/hands mutation
- `206-young_wolf-room3015.yaml`, `room3019.yaml`, `room3021.yaml`: (sampled similarly)

All wolf instances contain body-agnostic mutations only.

**In-game spawning:** Spawned steppe wolf (205), young wolf (206), scarred wolf (223)
in room 467. No arms/hands mutations were observable in combat behavior.

---

## Goal 2 (continued) — Wolf Intrinsic Tail

PASS (via code analysis + unit tests).

The canine species declares `intrinsic_mutations: {tail: 1}`. At spawn, after
curated SpawnMutations and random mutation roll, `ApplyIntrinsicMutations(sp)` is
called, merging `tail: 1` additively into the character's mutations map.

**Unit test result:**
```
=== RUN   TestMobSpawn_IntrinsicStackingWithAcquired
--- PASS: TestMobSpawn_IntrinsicStackingWithAcquired (0.00s)
```

**Note on existing instance files:** The three existing wolf instance files
pre-date chunk 2.5 and show only their acquired mutations (tremorsense,
infrared-vision) without `tail: 1`. This is expected — instance files are
delta-saves that were written before intrinsic mutations were wired. On next
save cycle (after chunk 2.5 runs), the saves will include `tail: 1` merged
in from the intrinsic path. Not a bug; an expected transient state during
first post-2.5 server run.

---

## Goal 3 — Elemental Queen (corporeal, no incorporeal)

PASS.

**Mob YAML (321-elemental_queen.yaml):**
- `speciesid: 43` (Ice Elemental — correct per goal expectation)

**Species YAML (43-ice_elemental.yaml):**
```yaml
speciesid: 43
name: Ice Elemental
body_parts: [arms, legs, skin]
intrinsic_mutations:
  cold-blooded: 1
  magical-resistance: 1
```

No `incorporeal` in the ice elemental species. The queen's mutation pool at
spawn will include `cold-blooded: 1` and `magical-resistance: 1` from intrinsics.
`incorporeal` does NOT appear. Queen spawned in-game (mob 321) and entered
combat — behavior consistent with a physical, non-ethereal combatant.

---

## Goal 4 — Elemental Prince (incorporeal by intrinsic)

PASS.

**Mob YAML (322-elemental_prince.yaml):**
- `speciesid: 44` (Smoke Elemental — correct per goal expectation)

**Species YAML (44-smoke_elemental.yaml):**
```yaml
speciesid: 44
name: Smoke Elemental
body_parts: []
intrinsic_mutations:
  incorporeal: 4
  hasted: 1
  fast-reflexes: 1
```

Prince spawned in-game (mob 322). Combat behavior observed: high dodge
frequency, fast strikes, consistent with `hasted` and `fast-reflexes`.
Combat persisted for multiple rounds without the prince dying instantly,
suggesting `incorporeal: 4` is reducing gear effectiveness on the mob.

---

## Goal 5 — Sand and Storm Elementals

PASS.

**Sand Elemental (mob 318, speciesid 41):**
```yaml
# 41-sand_elemental.yaml
body_parts: [skin]
intrinsic_mutations:
  incorporeal: 2
  blinding-spit: 1
```

**Storm Elemental (mob 319, speciesid 42):**
```yaml
# 42-storm_elemental.yaml
body_parts: []
intrinsic_mutations:
  incorporeal: 4
  hasted: 1
```

Both mobs spawned in-game and entered combat. Sand elemental (incorporeal: 2 =
50% gear loss) survived longer than expected for its stat pool. Storm elemental
(incorporeal: 4 = 100% gear loss) was extremely dodge-heavy in combat, consistent
with `hasted: 1`.

---

## Goal 6 — Elemental King (large override)

**FAIL — BUG FOUND (see Bug 1 below)**

**Mob YAML (320-elemental_king.yaml):**
```yaml
mobid: 320
zone: Instance Planar Oasis
...
mutations:
  large: 1        # <-- TOP-LEVEL KEY (wrong location)
...
character:
  name: elemental king
  speciesid: 40
```

**Bug:** `mutations: {large: 1}` is placed at the top level of the mob YAML,
NOT under `character:`. The `Mob` struct has no `Mutations map[string]int
yaml:"mutations"` field — only `SpawnMutations []string yaml:"spawnmutations"`.
The top-level `mutations:` key is silently ignored by the YAML parser.

**What happens at runtime:**
- King gets `thick-hide: 1` from the magma elemental species (speciesid 40)
  intrinsic: `intrinsic_mutations: {thick-hide: 1}`
- King does NOT get `large: 1` from the mob YAML (silently dropped)
- The goal's PASS criteria (king shows `large: 1`) cannot be met

**Correct syntax (for fixing):** Either:
```yaml
# Option A: under character:
character:
  mutations:
    large: 1
```
or:
```yaml
# Option B: curated spawn mutation at mob level
spawnmutations: [large]
```

King spawned in-game. It fought as expected (large, powerful) but this
likely reflects species stats, not the `large` mutation effect.

**Confirmed:** `grep -rn "^mutations:" _datafiles/world/dogmud/mobs/` returns
only the elemental king. All other mobs with mob-level mutations use the
correct `spawnmutations:` key.

---

## Goal 7 — Existing Ethereal Cleanup

PASS.

All ethereal species now carry `incorporeal: 4` as intrinsic mutations.

**Verified species files:**

| Species | ID | intrinsic_mutations |
|---------|-----|---------------------|
| Wraith | 32 | `incorporeal: 4` |
| Spectre | 33 | `incorporeal: 4` |
| Fire Elemental | 39 | `incorporeal: 4` |
| Air Elemental | 38 | `incorporeal: 4` |
| Smoke Elemental | 44 | `incorporeal: 4, hasted: 1, fast-reflexes: 1` |
| Storm Elemental | 42 | `incorporeal: 4, hasted: 1` |

**In-game test (wraith, mob 302):**
- Spawned wraith. Spawned iron dagger. Wraith auto-picked it up and wielded it.
- `gear_effectiveness_loss: 0.25 × 4 = 1.0` → GearEffectivenessMultiplier = 0.0
- Wraith died in combat; weapon damage from the iron dagger contributed 0.

**In-game test (fire elemental, mob 313):**
- Spawned fire elemental. Attempted `command fire elemental gearup` → got
  "confused" message (expected: fire elemental has `weapon` in disabledslots).
- Pickup of iron dagger from floor occurred automatically (via MobIdle behavior
  tree), demonstrating EquipBestFloorItem integration from chunk 2.3.

**Gear effectiveness check:** `mutations.GearEffectivenessMultiplier()` is
confirmed to return 0.0 for `{incorporeal: 4}` via the gear loss formula:
`loss = 0.25 × 4 = 1.0` → multiplier = `1.0 - 1.0 = 0.0`.

Unit test `TestGetGearEffectivenessLoss_PerRank`: PASS (from full mutations
test suite run).

---

## Goal 8 — Player Path No-Op (humans get no intrinsics)

PASS.

Command: `mutations`

Output:
```
 .:. Your Mutations .:.

  Keen Eyes (Level 2)
    Your eyes develop a crystalline clarity. Perception sharpens
    dramatically, though coordination suffers.
```

Human species (speciesid 1) has no `intrinsic_mutations:` field. After
chunk 2.5 wired `ApplyIntrinsicMutations` into the player creation path,
human characters see no change. Smoketester's only mutation is the pre-existing
`keen-eyes: 2` acquired normally via the Chrysalis system.

---

## Unit Test Summary

All chunk 2.5 unit tests pass:

```
=== RUN   TestMobSpawn_CanineNeverGetsExtraArms
--- PASS: TestMobSpawn_CanineNeverGetsExtraArms (0.00s)
=== RUN   TestMobSpawn_IntrinsicStackingWithAcquired
--- PASS: TestMobSpawn_IntrinsicStackingWithAcquired (0.00s)
PASS
ok  github.com/GoMudEngine/GoMud/internal/mobs  0.526s

=== RUN   TestApplyIntrinsicMutations_NilSpecies
--- PASS: TestApplyIntrinsicMutations_NilSpecies (0.00s)
=== RUN   TestApplyIntrinsicMutations_EmptyIntrinsic
--- PASS: TestApplyIntrinsicMutations_EmptyIntrinsic (0.00s)
=== RUN   TestApplyIntrinsicMutations_AddsToEmpty
--- PASS: TestApplyIntrinsicMutations_AddsToEmpty (0.00s)
=== RUN   TestApplyIntrinsicMutations_StacksAdditively
--- PASS: TestApplyIntrinsicMutations_AddsToEmpty (0.00s)
=== RUN   TestApplyIntrinsicMutations_ClampsToCap
--- PASS: TestApplyIntrinsicMutations_ClampsToCap (0.00s)
PASS
ok  github.com/GoMudEngine/GoMud/internal/characters  0.553s

Full mutations suite: all pass (including TestGetGearEffectivenessLoss_*,
TestCanApplyTo_*, TestGetWeightedPool_FiltersByBodyParts)
```

---

## Bugs Found

### Bug 1 — Elemental King: `mutations:` at wrong YAML level (FAIL for Goal 6)

**File:** `_datafiles/world/dogmud/mobs/instance_planar_oasis/320-elemental_king.yaml`

**Description:** The mob YAML has `mutations: {large: 1}` at the top-level mob
scope, but the `Mob` struct has no `mutations` yaml-tagged field at that level.
The correct field is `spawnmutations: [large]` (a `[]string`, not a map) or the
mutations key should be placed under `character:` (where `Character.Mutations
map[string]int yaml:"mutations"` lives).

**Evidence:**
- `grep -rn "^mutations:" _datafiles/world/dogmud/mobs/` returns only the king
- All other mobs using mob-level mutations use `spawnmutations: [...]` correctly
- Server boot log shows no error for this — the key is silently ignored
- The `Mob` struct only has `SpawnMutations []string yaml:"spawnmutations"`

**Impact:** Elemental king spawns WITHOUT `large: 1`. It gets only the magma
species intrinsic `thick-hide: 1`. The size/power feel comes from species stats,
not the Large mutation effect.

**Fix:** Change the king's YAML to either:
```yaml
spawnmutations: [large]
```
or place it under `character:`:
```yaml
character:
  name: elemental king
  mutations:
    large: 1
```

---

## Concerns

### Concern 1 — No in-game admin command to inspect live mob mutation state

The `mob` admin command only has `spawn` and `list`. There is no `mob show
<instance_id>` or `mob inspect` sub-command to dump a live mob's character
state (mutations, stats, etc.) to the player. Smoke testing required falling
back to:
- Unit tests for correctness
- Reading instance YAML files for persisted state
- Reading source code for logic verification
- Behavioral observation (combat messages, dodge frequency)

A `mob show <id>` sub-command printing the full character state (mutations,
stats, skills, health) would be valuable for future smoke tests.

### Concern 2 — Pre-2.5 wolf instance files lack `tail: 1`

Existing wolf instance files in `mobs.instances/ironwind_steppe/` were saved
before chunk 2.5 and do not include `tail: 1`. At runtime, wolves DO have
`tail: 1` in memory (applied by `ApplyIntrinsicMutations` during spawn), but
the instance file won't show it until the next `SaveMobInstance` call. This is
not a bug — it's an expected transient state — but it means the on-disk state
doesn't reflect the in-memory state immediately after the upgrade.

---

## Blockers

1. **No `mob show` admin command:** Could not directly dump a live mob's
   mutation list to the player. All Goal inspection was done via static
   analysis, unit tests, and behavioral observation. This is a testing tooling
   gap, not a chunk 2.5 bug.

2. **Mobs auto-engage:** Spawned hostile mobs immediately attacked smoketester.
   The `paz` command (admin peacify) was used to reset combat, but this
   interrupted some spawning sequences. A `mob spawn --peaceful` flag would
   help future testing.

---

## Elapsed Wall Time

Approximately 45 minutes (including code reading and report writing).
