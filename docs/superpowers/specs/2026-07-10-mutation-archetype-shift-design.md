# Mutation-Triggered Archetype Shift — Design

**Date:** 2026-07-10
**Status:** Approved
**Origin:** Deferred from chunk 4.2 brainstorming (2026-05-27); unblocked by
4.4/4.5/4.6 shipping. Memory: `project_mutation_triggered_archetype_shift`.

## Why

Mutations currently change *what abilities* a mob has but not *how it picks
fights*. A wolf that goes incorporeal still tries to melee; a boar that sprouts
extra arms still acts like a boar. Shifting the mob's behavior archetype on
acquisition makes mutations transformative rather than additive — and because
mobs only acquire mutations **in combat** (`tickMobMutationAcquisition` gates on
`IsInCombat`), every shift happens mid-fight, in front of a player.

## Decisions (from brainstorm Q&A)

| Question | Decision |
|---|---|
| Which mobs can shift? | Generic combat archetypes only (+ archetype-less mobs). Authored specialists never shift. |
| When does a shift fire? | On mutation **acquisition** (level 1) only. Deepening never triggers a shift. |
| Multiple pull-mutations? | **Highest rarity wins**, alphabetical key tiebreak. Re-evaluated on each acquisition. |
| Spell-less mobs shifting to caster? | Moot — **all mobs get the player baseline spellbook at spawn** (full actor parity). |
| `small` mutation direction | Hit-and-run/rogueish (`ambusher`), NOT skittish prey — small ≠ weak. |

## 1. The `archetype_pull` mapping (content)

New **optional** mutation YAML field:

```yaml
archetype_pull: generic_fighter
```

Loader validation **panics at boot** when a pull names a nonexistent archetype
or one outside the target whitelist (same convention as schedule validators;
caught by the pre-push boot test).

### Starter mapping (10 pulls; all other mutations have none)

| Mutation | Pull | Rationale |
|---|---|---|
| incorporeal (r10) | defensive_caster | Drifted from the physical — avoids melee, channels will |
| extra-arms (r9) | generic_fighter | Four-armed brawler |
| clawed-hands | predator | Built to hunt |
| toxic-bite | predator | Venom hunter (dispatchable via `try_mutation_active_at_target`) |
| blinding-spit | predator | Same family |
| sonic-shout | tank_taunter | Loud, attention-grabbing, aggro-pulling |
| dense-muscles | tank_taunter | Wall of meat |
| large | generic_fighter | Imposing bruiser |
| small | ambusher | Hit-and-run — strike from hiding, flee, re-hide |
| pacifism-aura | combat_passive | It literally suppresses fighting |

### Eligibility sets

**Shift-eligible (FROM):** `generic_fighter`, `predator`, `prey`,
`combat_passive`, `""` (no archetype — a shift grants one; strict upgrade).

**Target whitelist (TO):** `generic_fighter`, `predator`, `prey`,
`combat_passive`, `tank_taunter`, `defensive_caster`, `pure_caster`,
`ambusher`.

The asymmetry is deliberate: the FROM set protects authored behavior (bosses,
leader, casters, thief/lookout/scout, noncombat_* never shift); the TO set is
about what any mob can credibly play. `archer` is excluded from TO — it needs
a ranged weapon + ammo we can't conjure. `ambusher` is TO-only: any mob can
take the Hidden buff, but authored ambushers keep their tuning.

## 2. Shift mechanics

`behaviortree.ReevaluateArchetypeShift(mob)` — new function in
`internal/behaviortree` (it owns archetype knowledge, the `BehaviorState`
type, and already imports `mutations` for rarity). Called from
`tickMobMutationAcquisition` (internal/hooks/NewRound_MobRoundTick.go)
immediately after `mob.Character.Mutations[mutId] = 1` in the **acquisition**
branch only.

Logic, in order:

1. **Eligibility gate.** Current `BehaviorArchetype` must be in the FROM set,
   and the mob must have no per-mob btree file (per-mob trees shadow
   archetypes entirely — check via the engine's existing tree/negative-cache
   lookup before an `os.Stat` fallback).
2. **Pick the winner.** Collect owned mutations whose spec has an
   `archetype_pull`; sort by rarity descending, alphabetical key tiebreak
   (existing `mutationRarity` helper + the codebase's standard sort). Empty →
   return. Winner's pull == current archetype → silent no-op.
3. **The swap.**
   - `mob.BehaviorArchetype = target`
   - `mob.BTreeState = nil` (EnsureBTreeState lazily re-inits; mid-combat is
     fine — the next event evaluates the new tree)
   - Re-derive `SubmissionPolicy` / `SurrenderPolicy` via
     `DefaultSubmissionPolicyForArchetype` etc., using the same
     author-override guard the spawn path uses (never clobber explicit YAML)
   - **Merge** the target archetype's `default_goals` into the mob's goals —
     dedupe by goal type, keep existing goals (learned/reactive goals
     survive). Goal *weights* need no work: `goals/lookup.go` keys off
     `mob.BehaviorArchetype` dynamically.
4. **Flavor.** One room-visible line per target archetype from a small Go map
   ("X settles into a fighter's stance", "X's movements turn predatory", …),
   generic fallback ("Something in X's bearing changes."). Sight-gated via the
   existing visual room-send used by the acquisition messages. No new
   world-event type — acquisition already emits `MobMutationGained`.

## 3. Baseline spellbook parity

In `Mob.Validate()` (runs at template load and instance spawn): initialize
`SpellBook` if nil, then add each of `conviction-spike`, `chrysalis-glow`,
`identify` at value **1** — **only if the key is absent**. Authored spellbooks
(e.g. bandit_caster's `nerve-disruption: 50`) are never modified; value 1
matches a fresh player's proficiency. Every mob is caster-capable from birth;
non-caster trees never cast, so this is inert until a shift (or future caster
behavior) makes it matter.

## 4. Persistence

`internal/mobs/instance_save.go` gains `behavior_archetype` on the saved
instance, written **only when the live value differs from the template's**
(same selective pattern as `MutationProgress`), restored on load. A shifted
mob survives a server restart shifted. Instance saves are not deployed to
prod, so this is purely runtime state.

## 5. Validation & testing

- **Loader validation:** unknown/non-whitelisted `archetype_pull` → boot
  panic. Covered by a unit test and the pre-push boot test.
- **behaviortree unit tests:** specialist/noncombat mobs never shift;
  rarest-wins with alphabetical tiebreak; same-target no-op; per-mob-tree
  shadowing blocks the shift; btree state reset; policy re-derivation
  respects authored overrides; goals merged, not replaced.
- **mobs unit tests:** spellbook seeding merges under authored spells;
  instance-save round-trip preserves a shifted archetype.
- **Manual smoke:** SOP boot test; locally force-feed a wolf `extra-arms`
  (admin mutation grant) and confirm visible mid-fight behavior change +
  flavor line.

## Out of scope

- **Inverse relax-on-removal.** Mobs never lose mutations today; revisit if a
  mob-facing scour mechanic ever lands.
- **Deepening-triggered shifts** (level thresholds) — acquisition-only, per
  decision.
- **`archer` as a shift target** — capability we can't grant.
- **Per-mob-btree mobs** — shadowed by design; they keep their authored trees.
- **Wiring `try_mutation_active_at_target` into archetype trees** — separate
  content/balance pass (per the defer-tuning-to-post-build-playtest SOP).
