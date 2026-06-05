# Revenge-Witness: Report vs React Split

**Status:** Design approved — pending spec review → implementation plan
**Type:** Aliveness followup (4.5 rule 5 + the assault sibling), now unblocked by 5.1
**Date:** 2026-06-05
**Depends on:** 4.5 (reactive seeders), 5.1 (crimes substrate + guard enforcement — done)

## Problem

When a player steals from or attacks a mob, two reactive seeders loop over **every
mob in the room** and seed a personal `revenge-mob` goal into each:

- `internal/seeders/witness_of_theft_to_revenge.go` (`OnTheft`) — victim (pri 90) +
  all room witnesses (pri 60).
- `internal/seeders/aggressive_action_to_revenge.go` — attacked mob (pri 75) + all
  non-`AutoAggro` room witnesses (pri 50).

Seeding revenge into *every* witness is wrong for two archetypes:
- **Guards** — a guard should enforce the law (warn → arrest, via the 5.1 crime log
  + `RunGuardEnforcement`), not pursue a personal vendetta. The stray revenge goal
  can derail proper enforcement.
- **Noncombatant civilians** — a shopkeeper/townsperson can't act on a revenge goal
  (they can't fight). They should react with alarm and let the law handle it.

The "report" path **already exists** (5.1): the steal/attack actions call
`crimes.Record(...)` + `justice.MaybeDeclareBounty(...)` when faction-aligned
witnesses identify the perp, and `RunGuardEnforcement` acts on that log. So this
followup is mostly about **gating the revenge seed by witness type** and adding a
light alarm reaction for civilians — not building a new report mechanism.

## Scope

**In:** the two room-witness seeders — theft (`OnTheft`) **and** assault
(`aggressive_action_to_revenge`). They share the same indiscriminate room-witness
pattern, so both route through one shared classifier.

**Out (principled, not arbitrary):** `friend_killed_to_revenge` (the kin/murder
path). It seeds revenge only into **relationship-scoped kin** (`RelationsOf`), not
random room witnesses — already correctly targeted (kin *should* avenge their own,
matching the 6.5c pack/warren behavior). It does not sweep in uninvolved
guards/civilians, so it has neither defect. (A minor optional follow-up could skip
*noncombatant* kin, but that's separate and small.)

## Design

### Witness-response classifier (shared)

A new shared helper routes each witness (and the direct victim) to one of three
responses by type:

```
seedWitnessResponse(target *mobs.Mob, playerId int, revengePriority int):
    if mobs.IsGuardMob(target.Groups):        // law enforcement
        return                                 # REPORT-ONLY — seed nothing;
                                               # 5.1 crime record + RunGuardEnforcement handle it
    if target.IsNonCombatant():               // can't fight
        alarmReaction(target, playerId)        # ALARM — fright emote + one step toward an exit
        return
    seedRevengeGoalIfAbsent(target, "player", playerId, revengePriority)  # REVENGE (unchanged)
```

- Both `OnTheft` and `aggressiveActionToRevenge` replace their per-witness
  `seedRevengeGoalIfAbsent(...)` call with `seedWitnessResponse(witness, playerId,
  <witnessPriority>)` (theft 60, assault 50).
- The **victim/attacked mob** also routes through `seedWitnessResponse` at the
  victim priority (theft 90, assault 75). The victim is never noncombatant (you
  cannot steal from or attack a `non_combatant` mob — engine rejects it), so the
  victim only ever hits the guard or revenge branch. A **guard** that is itself the
  victim → report-only (its combat btree still handles self-defense independently
  of the goal seed; enforcement handles the crime).
- The assault seeder keeps its existing `AutoAggro` skip (those mobs already attack
  on sight; no seeding needed) — apply it before calling `seedWitnessResponse`.

### Guard detection — lift `isGuardMob` to shared

`isGuardMob(groups []string) bool` currently lives package-private in
`internal/hooks/NewRound_MobRoundTick.go` (checks `groups` for the `"guard"`
marker). Lift it to `mobs.IsGuardMob(groups []string) bool` (exported, in the
`mobs` package); update the hooks caller to use it; the seeders call it too.
**Do NOT use `Character.IsGuard()`** — that is the combat-stance ("Guard" position)
predicate, an unrelated false friend.

### Alarm reaction (noncombat civilians)

`alarmReaction(target, playerId)` is an immediate, momentary reaction — no
persistent goal (deliberately avoids the `survival`-goal-pruned-at-full-HP bug,
`project_survival_goal_pruned_when_healthy`):
1. A room-visible fright emote in the mob's voice (e.g. "recoils and cries out" /
   "hurries away, calling for the watch").
2. One step toward a random valid exit of the current room (issue the exit
   command); if the room has no exits, the emote alone.

The 5.1 crime record (fired by the steal/attack action when the civilian is
faction-aligned, e.g. townsfolk/shopkeeper/citizen) is the actual "report" — no
new reporting mechanism is built here.

## What stays unchanged

- The 5.1 crime-recording + bounty path in `actions/steal.go` and the attack path
  (already correct).
- `seedRevengeGoalIfAbsent` semantics + the revenge-mob goal itself.
- `friend_killed_to_revenge` (kin/murder).
- Priorities (90/60 theft, 75/50 assault) and the `AutoAggro` skip.

## Testing

Go unit tests (the seeders package is unit-tested — see existing
`internal/seeders/*_test.go`):
- `seedWitnessResponse`: a guard-group mob → no goal seeded; a `non_combatant` mob →
  no goal seeded (alarm path); a plain combat mob → revenge goal seeded at the given
  priority. (Use mob fixtures with `Groups: ["guard"]`, `NonCombatant: true`, and a
  plain fighter.)
- `mobs.IsGuardMob`: true for `["guard"]`, false otherwise; hooks caller still
  compiles/behaves.
- Regression: `OnTheft` / `aggressiveActionToRevenge` still seed revenge into a
  plain combat witness, and now skip a guard + a noncombatant witness.
- Boot smoke (deferred to user OK): steal from a mob in a guarded town square —
  the guard enforces (no personal chase), a shopkeeper bystander recoils/steps away,
  a combat thug bystander pursues.

## Files touched (anticipated)

- New/edit: `internal/mobs/` — exported `IsGuardMob(groups []string) bool` (lifted).
- Edit: `internal/hooks/NewRound_MobRoundTick.go` — use `mobs.IsGuardMob`.
- New: `internal/seeders/witness_response.go` — `seedWitnessResponse` + `alarmReaction`.
- Edit: `internal/seeders/witness_of_theft_to_revenge.go` — route witness + victim
  through `seedWitnessResponse`.
- Edit: `internal/seeders/aggressive_action_to_revenge.go` — same.
- Tests: `internal/seeders/witness_response_test.go` (+ touch existing seeder tests
  if they assert the old blanket-seed behavior).
- `internal/seeders/context.md` + `internal/mobs/context.md` notes.

## Out of scope

- Assault/murder beyond the two room-witness seeders; `friend_killed` kin path.
- An active "fetch the guards" / alarm-propagation mechanism (the lighter alarm
  reaction is enough; 5.1 enforcement already responds to the crime log).
- Renaming the `witness_of_theft_to_revenge.go` file (the name is now slightly
  broad, but renaming is churn — leave it, note in context.md).
