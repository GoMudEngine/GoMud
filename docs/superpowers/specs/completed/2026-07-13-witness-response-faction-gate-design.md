# Witness-Response Faction Gate — Design

**Date:** 2026-07-13
**Status:** Approved (brainstorm), ready for implementation plan
**Scope:** small, targeted bug fix + consistency alignment

## Problem

A feel-tester (Fernling, 2026-07-13) found that **Drillmaster Vorn — a tutorial
NPC — flees when the player attacks the practice dummy he just assigned them to
hit**, emitting `"recoils and cries out, then hurries for the nearest way out"`
and stepping toward an exit. A drillmaster panicking at the sparring session he
staged is immersion-breaking. The intent is a broader fix (not a Vorn one-off):
attacking a *sanctioned/unaligned* target should not make bystanders react as if
a crime occurred.

## Root cause

Two independent systems react to a player attacking a mob:

1. **Crime record** (`internal/crimes/`) — rep loss + guard enforcement. This is
   **already faction-gated**: `crimes.Record(...)` and
   `crimes.WitnessesInRoom(factionIds, room, ...)` only involve mobs whose
   `Groups` overlap the *victim's* registered factions
   (`factions.GetDefinition(g) != nil`). Attacking a factionless mob records no
   crime.
2. **Witness-response seeder** (`internal/seeders/aggressive_action_to_revenge.go`)
   — on the `PlayerAttackedMob` event it seeds a reaction into the attacked mob
   (victim) and **every** other non-`AutoAggro` mob in the room (witnesses), via
   `seedWitnessResponse` → guard = report-only, combat-capable = revenge goal,
   non-combatant = the alarm emote + one step toward an exit. **This path is NOT
   faction-gated** — it reacts to all bystanders regardless of allegiance.

The training dummy's only group is `construct`, which is **not** a registered
faction, so the dummy is factionless — the crime record correctly ignores it, but
the witness-response seeder does not, so Vorn (an unaligned bystander) reacts.
This is a latent bug affecting *any* factionless victim (dummies, wildlife such as
the passive boar/bees/turtle, monsters, unaligned tutorial mobs), not just Vorn.

## Design

Gate the witness-response seeder on the **victim's faction**, mirroring the rule
the crime record already uses. No new mob flag, no per-mob authoring.

In `aggressiveActionToRevenge` (fires on `PlayerAttackedMob`):

1. Resolve the attacked victim's **factions** = its `Groups` filtered to those
   that are registered faction definitions (`factions.GetDefinition(g) != nil`) —
   the same membership test `crimes.WitnessesInRoom` uses.
2. **If the victim is factionless → return early.** No victim reaction, no
   witness alarm/revenge. Covers training dummies, wildlife, monsters, and
   unaligned tutorial mobs — attacking them is not a social crime.
3. **If the victim has ≥1 faction:**
   - Seed the victim's own reaction (unchanged — it has a faction and, per the
     existing top-of-function guard, is not `AutoAggro`).
   - Seed witnesses **only if they share ≥1 of the victim's factions**. Reuse
     the group-overlap logic (either call `crimes.WitnessesInRoom(victimFactionIds,
     room, victimInstanceId)` and iterate its result, or an equivalent shared
     helper). Continue to skip `AutoAggro` witnesses (they already attack on
     sight; revenge is redundant noise — existing behavior).

The existing `AutoAggro` early-skip (attacking an already-hostile mob is not a
crime) stays as-is.

### Implementation notes
- The victim's-faction resolution and the witness group-overlap are the same
  membership test already implemented inline in `crimes.WitnessesInRoom`. Prefer
  reusing it (pass the victim's faction ids + `excludeInstanceId = victim
  instance`) over duplicating the loop; if a small exported helper reads cleaner
  (e.g. `crimes.FactionIdsForMob(mob) []string`), add it there. The plan resolves
  the exact seam and confirms no `seeders → crimes` import cycle (crimes does not
  import seeders, so this direction is safe).
- `classifyWitnessResponse` / `seedWitnessResponse` / `alarmReaction` are
  unchanged — only *who* gets a response seeded changes.

## What does NOT change
- **Combat-flee** (`mobcommands/flee.go`, behavior-tree `actFlee`): a mob already
  in combat disengaging at low HP / when cornered. Untouched — it was not flagged
  and is working as intended.
- **Crime record + guard enforcement**: already faction-gated; unchanged.
- **The alarm/revenge reactions themselves**: a real crime against a faction still
  makes that faction's members react exactly as today.
- A **factioned** victim being attacked still reacts and still alerts its
  faction-mates.

## Consequences / consistency
- **Fixes a class, not a case:** no unaligned target (dummy/wildlife/monster/
  tutorial NPC) triggers bystander alarm or revenge anymore.
- **Aligns the two systems:** the witness-response now obeys the same faction rule
  as the crime record. Previously they disagreed (crime ignored cross-faction;
  witness-response reacted to everyone).
- **Deliberate behavioral change:** a bystander of a *different* faction than the
  victim no longer reacts to a real crime (e.g. a Merchant NPC won't panic when
  you mug a Thornwall citizen). This matches how the crime record already behaves.
  The only thing removed is "any violence alarms any civilian regardless of
  allegiance," which is not consistently modeled today. If universal violence-fear
  is ever wanted, it is a separate deliberate feature — out of scope here.

## Testing
- Unit tests on `aggressiveActionToRevenge` (table-driven, using the existing
  seeder test harness if present):
  - **Factionless victim** (e.g. a `construct`-group dummy): attacking it seeds
    **no** witness response into a factionless bystander (the Vorn case) — the
    regression guard for this bug.
  - **Factioned victim + same-faction witness:** witness gets a response seeded.
  - **Factioned victim + different-faction witness:** witness gets **no**
    response.
  - **`AutoAggro` victim / `AutoAggro` witness:** still skipped (existing behavior
    preserved).
- Boot smoke: server starts clean.
- Optional live check: attack the Drill Yard training dummy (9109) with Vorn (9108)
  present → Vorn no longer flees.

## Files
- `internal/seeders/aggressive_action_to_revenge.go` — add the victim-faction gate.
- Possibly `internal/crimes/crimes.go` — small exported faction-ids helper if it
  reads cleaner than reusing `WitnessesInRoom` inline.
- `internal/seeders/aggressive_action_to_revenge_test.go` (or existing seeder test
  file) — the table-driven tests above.
