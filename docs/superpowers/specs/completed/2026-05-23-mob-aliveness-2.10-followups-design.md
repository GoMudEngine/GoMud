# Mob Aliveness 2.10 Followups (Design)

**Status:** Approved (brainstorming) — ready for `writing-plans`
**Roadmap chunk:** Bundle of 5 followups surfaced by chunk 2.10's audit, code review, and triage. No new roadmap row; this chunk closes out chunk 2.10's deferred work and resolves several `project_*` memory entries.
**Size:** L (largest mob-aliveness chunk so far)
**Branch:** `feature/mob-aliveness-2.10-followups`

## Goal

Close the loop on chunk 2.10's deferred work and ship the scope
expansions agreed during user triage. Five distinct items, executed
sequentially (refactors first, features last) on a single feature
branch.

## Non-goals

- Throwable mobs (gated on a future ranged-weapon-system chunk that
  doesn't yet exist).
- Withdrawing items from the forager chests (v1 is deposit-only).
- Player-collectible chest stock as a separate economy hook (was
  rejected during chest-purpose triage in favor of overflow cache).
- Re-tuning sonic-shout / toxic-bite to match prior raw-arithmetic
  magnitudes (we accept whatever the pipeline produces; tuning
  followup if smoke surfaces issues).
- **Vendor backfill from forager chests** — the critical other half
  of the overflow design. Without this, chests are one-way drains.
  Scope is too large for this chunk; followup memory entry sketches
  the design space.

## In scope (5 items, execution order)

1. **`charge.go` trip-math duplication cleanup** (refactor, S, ~50 LoC)
2. **Surprise-attack unification refactor** (refactor, S-M, ~200 LoC)
3. **`try_any_active_mutation` btree action** (new mechanism, M, ~150 LoC)
4. **Mutation damage pipeline routing** (refactor + balance, M, ~80 LoC)
5. **Forager locked-chest workflow** (new feature, M-L, ~300+ LoC + YAML)

## Item 1 — `charge.go` trip-math duplication cleanup

**Files:** `internal/mobcommands/charge.go`, possibly minor extension to
`internal/actions/combat_trip.go`.

**Current state:** `charge.go` reimplements `actions.ExecuteTrip`'s
knockdown roll + prone application instead of calling it. Charge-specific
bits (distance approach narration, post-impact aggro behavior) are
tangled with the trip math.

**Fix:** Read `charge.go` and `actions.ExecuteTrip` side-by-side.
Extract the trip-specific lines and replace with
`actions.ExecuteTrip(attacker, target, opts)`. Charge keeps only its
decoration (charge-specific messaging, distance closing, aggro
behavior). If `ExecuteTrip`'s current signature can't accommodate
charge's needs (e.g., charge wants to control the knockdown roll
outcome), extend with an `Opts` field; don't fork.

**Tests:** Existing combat tests should cover the trip mechanics. Add
one charge-specific test verifying the decoration still fires when
knockdown succeeds.

**Memory entry resolved:** `project_charge_trip_math_duplication.md`
marked Done with commit hash.

## Item 2 — Surprise-attack unification refactor

**Files:**
- Create: `internal/actions/surprise_attack.go`
- Create: `internal/actions/surprise_attack_test.go`
- Modify: `internal/usercommands/attack.go` (replace `executeSurpriseAttack`
  helper with call-through)
- Modify: `internal/mobcommands/attack.go:64` (replace inline branch with
  call-through)

**Pattern:** Same lift-to-actions shape as chunk 2.10's mutation lifts.

```go
// internal/actions/surprise_attack.go

type SurpriseAttackOpts struct {
    Target Actor
}

type SurpriseAttackResult struct {
    Triggered    bool
    StrikeCount  int    // per-weapon strikes that landed
    BlockReason  string // "not-hidden", "cooldown", "no-target"
}

func SurpriseAttack(actor Actor, opts SurpriseAttackOpts) SurpriseAttackResult
```

**Action owns:** hidden-state check, special-move cooldown, per-weapon
iteration, per-weapon attack roll, Awareness_Cascades hand-off (Hidden
→ Revealing on round end). The Awareness_Cascades coordination stays
where it is (it's a state-transition policy, not attack-resolution
mechanic) — the action just hands the state machine the right signal.

**Wrappers own:** target resolution + actor adaptation.

**Tests:** Mirror chunk 2.10 B2 pattern. Cover: not-hidden block,
cooldown block, no-target block, success-triggered, per-weapon strike
count.

**Memory entry resolved:** `project_surprise_attack_unification.md`
marked Done.

## Item 3 — `try_any_active_mutation` btree action

**Files:**
- Extend: `internal/behaviortree/actions_mutation.go` (already exists
  from chunk 2.10)
- Modify: `internal/behaviortree/actions.go` (register new action)
- Modify: `internal/behaviortree/context.md` (document new action)
- Extend: `internal/behaviortree/actions_mutation_test.go`

**Action shape:**

```yaml
# Author opts in per node:
- type: try_any_active_mutation
  # no required params — enumerates mob's current mutations
```

**Algorithm:**

1. Get mob + room. Return Failure if either nil.
2. Walk `mob.Character.Mutations` map (live state, includes
   runtime-evolved entries).
3. For each mutation key the mob has: look up in `mutationTriggers`
   map (chunk-2.10 dispatch table, restricted to self/AoE mutations:
   `blinding-flash`, `healing-gel`, `pacifism-aura`, `sonic-shout`).
   Skip if not in map.
4. Sort qualifying candidates by `rarity` field from
   `_datafiles/world/dogmud/mutations/<key>.yaml`, descending.
   Tie-break alphabetically by key for determinism.
5. For each candidate in sorted order: call
   `trigger(actor, MutationOpts{})`. If `Triggered`, return `Success`.
   Otherwise (block reason fall-through), try next.
6. If no candidate fires: return `Failure`.

**Coexistence with `try_mutation_active`:** Authors choose.
Explicit-keys version stays the default for archetypes that want
curation; dynamic version is opt-in for archetypes that want autonomous
behavior.

**Single-target mutations excluded:** Same restriction as
`try_mutation_active` — `blinding-spit` and `toxic-bite` need a
target-resolving primitive that's still deferred (new followup memory
entry, see Cross-cutting section).

**Tests:**
- Registered-in-registry
- Mob-lacks-any-eligible-mutation → Failure
- Mob-has-multiple → rarity-descending order verified
- Mob-has-only-single-target-mutations → Failure (no eligible candidates)

**Memory entry resolved:**
`project_mutation_active_runtime_evolution_btree.md` marked Done
(Path B + rarity-descending shipped).

## Item 4 — Mutation damage pipeline routing

**Files:**
- Modify: `internal/actions/mutation_sonic_shout.go`
- Modify: `internal/actions/mutation_toxic_bite.go`
- Update: corresponding `_test.go` files

**Sonic-shout fix:**

```go
// OLD: dmg := int(char.Stats.Willpower.ValueAdj) * 8 / 100

// NEW:
raw := combat.CalcRawDamage(
    float64(char.Stats.Willpower.ValueAdj),
    float64(char.GetSkillLevel(skills.UnarmedCombat)),
    1.0,                       // no item multiplier
    combat.ChannelConviction,  // routes through conviction_mitigation
)
mitigated := combat.ApplyMitigation(raw, target.GetConvictionMitigation(),
    cfg.ConvictionMitigationCap)
dmg := dice.RollStat(mitigated).Value
```

Willpower stays as the input stat (preserves "willful shout" character)
but defense routes through Conviction (mental resilience). Damage
description via `combat.GetDamageDescription` — no raw numbers in
player text.

**Toxic-bite fix (damage portion only):**

```go
// OLD: dmg := int(char.Stats.Strength.ValueAdj) * 6 / 100

// NEW:
raw := combat.CalcRawDamage(
    float64(char.Stats.Strength.ValueAdj),
    float64(char.GetSkillLevel(skills.UnarmedCombat)),
    1.0,
    combat.ChannelPhysical,
)
mitigated := combat.ApplyMitigation(raw, target.GetPhysicalMitigation(),
    cfg.PhysicalMitigationCap)
dmg := dice.RollStat(mitigated).Value
```

**Poison DoT magnitude stays raw for v1.** `Vit × 0.04` continues to
drive the poison condition's magnitude. DoT magnitudes work differently
from instant damage; routing those through the pipeline is a separate
concern (new followup memory entry).

**Balance expectation:** Magnitudes will shift vs the old raw
arithmetic. We accept whatever the pipeline produces. Patch notes
flag this as a behavioral change. If smoke surfaces imbalance, a
tuning followup adjusts `ChannelScale` knobs.

**Tests:** Update existing tests to assert damage flows through the
pipeline (verify mitigation was applied — e.g., target with high
mitigation takes less than target with zero). Don't assert exact damage
numbers (they vary with the pipeline's roll spread).

**Memory entry partially resolved:**
`project_mutation_damage_pipeline_bypass.md` marked partially-Done —
sonic-shout + toxic-bite damage routed; poison DoT magnitude stays raw,
note in the resolution.

## Item 5 — Forager locked-chest workflow

This is the most substantive piece. Five sub-items.

### 5a. Mob `lock` and `unlock` verbs

Lift the player implementation if straightforward. Read
`internal/usercommands/lock.go` (156 LoC) and `unlock.go` (151 LoC).
If the core logic ports cleanly to the Actor abstraction, extract
into `internal/actions/lock.go` and `unlock.go` with the standard
Actor-based signature; both wrappers (user and mob) call through.

If lifting proves more invasive than helpful (e.g., heavy reliance
on user-specific keyring state that doesn't translate to mobs),
fall back to standalone mob versions in
`internal/mobcommands/lock.go` and `unlock.go` that reimplement the
mob-relevant subset. The decision is made during implementation
based on actual code shape; either approach satisfies the
mob-verb-registration requirement.

Either way: register `"lock"` and `"unlock"` in `mobcommands.go`.
Key-matching for mobs uses the same `key.UniqueId == lock.UniqueId`
model — mob inventory checked for matching key item.

### 5b. New btree primitive `try_store_excess`

**File:** Extend `internal/behaviortree/actions_forager.go` (or
create a new `actions_forager_storage.go` if the file is already
crowded).

**YAML shape:**

```yaml
- type: try_store_excess
  chest_room: <room-id>
```

The `chest_room` parameter comes from the mob's `storage_chest_room`
field (set per-mob in YAML); the btree node can either reference it
inline or read from mob state. Both approaches valid; implementer's
choice based on the existing forager-archetype pattern.

**Algorithm:**

1. Mob has items in satchel? Return `Failure` if empty (Storing
   state's selector falls through to Recalling).
2. Mob is in `chest_room`? If not, issue `mob.Command("pathto <chest_room>")`
   and return `Success` (this tick consumed; next tick re-checks).
3. Lock state of the chest's lockbox: locked? Issue `mob.Command("unlock lockbox")`
   and return `Success`. (Next tick continues to deposit step.)
4. After unlock: iterate satchel items, issue `mob.Command(fmt.Sprintf("put %s in lockbox", item.Name()))`
   for each. The engine's existing `put`-into-container path handles
   capacity failures gracefully — if the chest is full mid-deposit,
   unsuccessful `put` commands no-op and the items stay in the
   satchel for the next cycle's attempt. Returns `Success` after
   the deposit pass completes regardless of how many items landed.
5. After deposit: issue `mob.Command("lock lockbox")` and return `Success`.
6. The state machine transitions out of Storing back to Recalling
   on the next tick after `lock` succeeds (or after a hard turn
   counter if the workflow stalls).

**Multi-tick state:** The primitive completes the full workflow over
multiple ticks (pathto → unlock → put → lock). Each tick advances one
step; the btree re-tick model handles the rest. No additional state
machine inside the primitive.

### 5c. Tova's new private dwelling room

**File:** New room file at
`_datafiles/world/dogmud/rooms/stillwater_marsh/<NEW_ID>.yaml`.

**ID allocation:** Use `python tools/id_inventory.py` to allocate a
free room ID in the marsh zone.

**Coordinate placement — REQUIRED zone-scan check (per `[[feedback_zone_coord_planning]]`):**
Before writing the room file, grep all existing room YAMLs across the
marsh zone AND geographically-adjacent zones (Stillwater, Stillwater
Marsh, anything that shares cardinal coords with the marsh) to
enumerate occupied `coord: {x, y, z}` triples. Pick coordinates
adjacent to one of Tova's existing wander territory rooms that are
GUARANTEED not to collide with any other room. Implementation plan
will issue the grep command verbatim as a prerequisite step.

**Room shape:** Model on Kessa's "Forager's Camp" (room 4197) — same
ingredients:
- Title: something like "Tova's Marsh Hut" or "Reedwoven Hut"
- Description: damp marsh-shelter on stilts, with bedding pallet,
  bundles of dried plants, ironbound lockbox
- `mutators: [sanctuary]` (matches Halix/Kessa)
- `nouns:` entry for `lockbox` and any other flavor items
- `idlemessages:` for ambient flavor
- `spawninfo:` includes `mobid: 371` (Tova)
- `containers:` block:
  ```yaml
  containers:
    lockbox:
      lock:
        difficulty: 10
        relockinterval: 24 hours
        rotationseed: 1
      items: []
  ```
- One exit back to the adjacent existing marsh room

**Update Tova's mob YAML:** Change `fold_anchor_room: 4123` (Temple of
Stillwater) to the new room ID. Add `storage_chest_room: <NEW_ID>`.
The Temple stays as a wander destination but isn't her home.

### 5d. Keys

**Model:** Each chest's lock gets a unique `keyid` (or matched by
`RotationSeed` if that's the actual mechanism). Each forager's mob
YAML gains the corresponding key item in inventory/equipment.

**Allocation:** Three new key items (one per forager). Use
`id_inventory.py` to allocate three free item IDs in the keys
category. Item YAMLs include `keyid` matching the chest's lock
identifier.

**Halix and Kessa update:** add `storage_chest_room: <existing-room-id>`
field on their mob YAMLs (3040 and 4197 respectively). Add the new
key items to their equipment.

**Tova update:** Add `storage_chest_room: <NEW_ID>` and the new key
item to her equipment.

**Item filter for what goes in the chest:** All unsold items at end
of vendor trip. Simplest possible — no bucket filtering, no per-vendor
exclusion. If the forager returns home with nothing (vendor took
everything), Storing state skips itself.

### 5e. State machine wiring

**File:** `internal/forager/state.go`.

Insert `StateStoring` between `StateDelivering` (existing) and
`StateRecalling` (existing) in the state enum + transition table.

**Existing transitions:**
- `StateDelivering → StateRecalling` (current)

**New transitions:**
- `StateDelivering → StateStoring` (when forager has chest + has items)
- `StateDelivering → StateRecalling` (when no chest OR no items)
- `StateStoring → StateRecalling` (always, after deposit completes
  or after a hard turn-count watchdog to prevent infinite loops)

**Forager YAML archetype update:** Add a new branch (or extend
existing) in `_datafiles/world/dogmud/behaviors/<forager-archetype>.yaml`
that handles `StateStoring` by calling `try_store_excess`. If the mob
has no `storage_chest_room` field, the Storing state is never entered
(transition straight from Delivering to Recalling).

**Per-archetype detection of `storage_chest_room`:** the forager state
machine's `tickForagerDelivering` (or its wrapper) checks the mob
field at Delivering-end to decide whether to transition to Storing or
Recalling.

## Cross-cutting concerns

### Branch + commit shape

Single feature branch: `feature/mob-aliveness-2.10-followups`.

Approximate commits (in order):
1. `refactor(combat): charge.go delegates trip math to actions.ExecuteTrip`
2. `refactor(actions): lift surprise_attack into actions package`
3. `feat(btree): try_any_active_mutation dynamic-dispatch action`
4. `refactor(actions): route mutation_sonic_shout damage through pipeline`
5. `refactor(actions): route mutation_toxic_bite damage through pipeline`
6. `feat(actions): lock/unlock mob verbs` (or standalone-mob-cmd variant)
7. `feat(forager): try_store_excess btree primitive + StateStoring`
8. `feat(content): Tova's marsh hut + lockbox + keys for all foragers`
9. `chore(memory): close 5 resolved followups + log 3 new ones`
10. `docs(2.10-followups): patch notes + roadmap closeout`

Roughly 10 commits + per-item test commits where TDD applies. ~12-15
commits total.

### Testing strategy

| Item | Unit tests | Manual smoke |
|---|---|---|
| 1. charge.go dup | New charge-decoration test + existing trip tests cover the math | Trigger charge in-game; verify knockdown behavior unchanged |
| 2. Surprise-attack unification | TDD per chunk-2.10 B2 pattern: 4 block paths + success + per-weapon strike count | Hidden mob ambush; verify per-weapon strikes still fire |
| 3. `try_any_active_mutation` | Registry presence, mob-lacks-any → Failure, rarity-descending ordering, single-target-only → Failure | Spawn mob with 2+ active mutations; observe btree picks rare one first |
| 4. Mutation damage pipeline | Update existing tests; assert mitigation reduces damage (don't assert exact numbers) | **REQUIRED** — magnitudes shift; user validates feel |
| 5. Forager workflow | Mob lock/unlock smoke tests; state-machine transition tests; `try_store_excess` algorithm tests | **REQUIRED** — boot, watch Tova/Halix/Kessa complete a full cycle including chest visit |

Per chunk 2.9/2.10 precedent, **in-game smoke is deferred to the user
post-merge.** The implementation plan will list a step-by-step smoke
checklist.

### Balance changes worth flagging in PATCH_NOTES

- **Mutation damage magnitudes shift** (Item 4): sonic-shout +
  toxic-bite numbers change as they route through the pipeline.
  High-mitigation targets take less than before; zero-mitigation
  targets take roughly the pipeline's `ChannelScale`-derived value.
- **Trip math via charge** (Item 1): if `actions.ExecuteTrip` produces
  different knockdown probabilities than `charge.go`'s prior
  reimplementation, charge's prone-application rate may shift slightly.
  Verify during smoke.

### Memory entries to write at chunk end

**To resolve** (mark Done in MEMORY.md, update file frontmatter):
- `project_charge_trip_math_duplication.md`
- `project_surprise_attack_unification.md`
- `project_mutation_active_runtime_evolution_btree.md`
- `project_mutation_damage_pipeline_bypass.md` (partial — see new
  followup below)
- `project_forager_locked_chest_workflow.md`

**To create (new followups surfaced by this chunk):**
- **Vendor backfill from forager chests** — the missing other half
  of the overflow design. Without this, chests are one-way drains.
  Sketches: forager-withdraws-on-next-vendor-trip, courier NPC,
  vendor restock querying nearby chests, scheduled bulk transfer.
- **Poison DoT magnitude pipeline routing** — toxic-bite's poison
  portion still uses raw `Vit × 0.04`. Future cleanup to route DoT
  magnitudes through a similar pipeline (likely needs a new pipeline
  variant for over-time damage).
- **Mob single-target mutation dispatch** — `blinding-spit` and
  `toxic-bite` still aren't fireable from `try_mutation_active` or
  `try_any_active_mutation`. Need a target-resolving btree primitive
  (`try_mutation_active_at_target` per chunk 2.10's followup note).

### Roadmap update

This chunk doesn't get its own progress-tracker row — it bundles
followups from chunk 2.10's closeout. After merge:
- No tracker row change.
- Append a "Followup chunk shipped" note to chunk 2.10's `**Shipped:**`
  paragraph in `MOB_ALIVENESS_ROADMAP.md` with date + commit hash.
- Update relevant Companion Phase 5 / loose-followup entries in
  MEMORY.md.

### Smoke checklist (post-merge, user-driven)

Step-by-step checklist provided in the implementation plan. Covers:
- One charge combat round (verify decoration + math unchanged)
- One hidden-mob surprise attack round (verify per-weapon strikes)
- A mob spawn with 2 active mutations using `try_any_active_mutation`
  (verify rare-mutation-first ordering)
- Sonic-shout + toxic-bite damage in combat (verify pipeline-mitigated
  output feels reasonable)
- One full forager cycle for Tova specifically (verify the new room
  + chest workflow end-to-end)

## Open questions

None at spec time — all clarifications captured during brainstorming,
scope locked.

## References

- Originating chunk: `docs/superpowers/specs/2026-05-23-mob-aliveness-2.10-pvm-mvp-parity-audit-design.md`
- Deferred-gap review (where these followups were triaged):
  `docs/superpowers/specs/2026-05-23-mob-aliveness-2.10-deferred-gaps-review.md`
- Actions-lift precedent: chunk 2.1 (`actions.Buy`), chunk 2.9
  (`actions.Forage`, `actions.Salvage`), chunk 2.10 mutation lifts
- Unified damage pipeline: CLAUDE.md "Unified Damage & Mitigation
  Pipeline (Stage 34)" section
- Forager state machine: `internal/forager/state.go`,
  `internal/behaviortree/actions_forager.go`
- Existing lockbox infrastructure: `internal/gamelock/`,
  `_datafiles/world/dogmud/rooms/ironwind_steppe/3040.yaml`
  (Halix's setup), `_datafiles/world/dogmud/rooms/the_fernway_south/4197.yaml`
  (Kessa's setup)
- Coord-planning rule: `[[feedback_zone_coord_planning]]`
