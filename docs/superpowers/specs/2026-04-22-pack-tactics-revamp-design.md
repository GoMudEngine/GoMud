# Pack Tactics Revamp — Design

**Date:** 2026-04-22
**Status:** Approved
**Related:**
- Bug surfaced 2026-04-22 (Duard's wife, Oriana): temple priest Olen
  aggroes a respawning player in Thornwall after the player had fought
  bandits in North Road.
- Fix A shipped 2026-04-22 as a bandaid
  (`fix(mobcommands): LookForTrouble skips grace-protected players`,
  commit `d93744a7`). This spec is the follow-up full revamp.
- Memory follow-up: `project_pack_tactics_revamp.md`.
- Prior art: Companion Phase 4 (2026-04-20) established behavior-tree
  archetypes. Companion Phase 5 (on the project roadmap) intends to
  extend archetypes from companions to wild mobs; this revamp *is*
  that extension plus pack-reaction semantics.

## Goal

Replace `mobs.MakeHostile` / `mobs.IsHostile` — a broad-group
hostility-propagation mechanism — with a narrow, routine-scoped pack
reaction system driven by behavior-tree archetypes. The existing system
propagates hostility through every taxonomic group a mob belongs to,
so attacking one "humanoid bandit" makes every "humanoid priest" in
town hostile on sight. The replacement propagates only to same-room
(or adjacent-room via call-for-help) mobs that share a narrow routine
declaration, and each mob reacts per its archetype.

As a bonus, the replacement leaves clean hooks for a later
"impression graph" upgrade (tracked as v2 in this spec) that gives
mobs persistent feelings about other mobs and players, emerging
naturally from the routine/role structure.

## Non-goals

- Not building a reputation, faction-standing, citizenship, or bounty
  system. City-wide / zone-wide alerts will come later from that
  separate system.
- Not changing the `pack_roaming.go` movement system. Pack-roaming
  (who-moves-with-whom) stays keyed off `groups` overlap. Pack-combat
  (who-reacts-when-attacked) is what this spec revamps. They are
  orthogonal concerns.
- Not implementing the v2 impression graph in this revamp. Only
  leaving a clean hook for it.

## Key decisions

1. **No new "role" field.** "Role" was a weak abstraction recreating
   the archetype concept. The existing `behavior_archetype` YAML field
   already holds a btree; it just gains a new `packmate_hurt` handler.
2. **New narrow declarative fields on mobs:** `routine` (freeform
   string) and optional `routine_links` (list of other routines this
   mob also reacts to).
3. **Pack identification** is *routine match* (same routine, or
   victim's routine appears in this mob's `routine_links`), scoped to
   *same room* by default. `callforhelp` extends it by one adjacent
   room.
4. **Reaction dispatch** is via a new btree event `packmate_hurt`
   fired by the combat-unification layer immediately after the
   defender aggro is established.
5. **MakeHostile / IsHostile / `mobsHatePlayers` are deleted.** The
   `LookForTrouble` group-hostility branch goes with them.
6. **Migration is all-or-nothing per feature branch.** No partial
   rollout. Infrastructure + default archetype handlers + content
   migration for priority zones + deletion ship as one unit to avoid
   a window of missing pack behavior.

## Architecture

### Mob YAML fields

- `behavior_archetype` (existing) — now also declares the mob's
  `packmate_hurt` response.
- `routine` (new, optional string) — e.g. `bandit_camp_guard`,
  `temple_service`, `wolf_pack_ironwind`. Freeform; the engine only
  string-compares it.
- `routine_links` (new, optional list of strings) — other routines
  this mob considers allied. E.g., a bandit lookout on the road has
  `routine: watch_north_road` and
  `routine_links: [bandit_camp_guard]` so a call from the camp pulls
  it in and vice versa.

Mobs without a routine participate in no pack. Mobs without an
archetype dispatch no `packmate_hurt` to their packmates. Both fields
are optional and zero-default safely.

### Packmate identification

New helper `mobs.FindPackmatesInRoom(victim *Mob) []*Mob` returns all
same-room mobs `m` such that:

- `m.InstanceId != victim.InstanceId`, and
- `m.Character.Health > 0`, and
- `!m.Character.IsCharmed()` (charmed mobs follow their owner, not
  packs — matches the `pack_roaming.go` exclusion), and
- (`m.Routine == victim.Routine` *and* both non-empty) *or*
  (`victim.Routine` appears in `m.RoutineLinks`), *or*
  (`m.Routine` appears in `victim.RoutineLinks`).

Lives in `internal/mobs/packmates.go`.

### Event dispatch

New function `dispatchPackmateHurt(victim *Mob, attackerUserId int,
attackerMobInstanceId int)` in `internal/hooks/packmate_hurt.go`.
Called from `handleAggroAndAssist` (`NewRound_DoCombat_unified.go`)
immediately after the defender-side aggro is established, replacing
the current `mobs.MakeHostile` call.

Behavior:
- Calls `mobs.FindPackmatesInRoom(victim)`.
- For each packmate, fires the new `packmate_hurt` btree event with
  context `{victim_instance_id, attacker_user_id,
  attacker_mob_instance_id}`.
- Packmates without an archetype get no event (no-op). Packmates with
  an archetype but no handler for `packmate_hurt` get no action
  (btree selector Failure fall-through).

### New btree event: `packmate_hurt`

Registered in the engine's event registry alongside existing events
(`mob_hurt`, `player_enter`, `mob_combat_round`, etc.). Consumed by
archetypes via `event: packmate_hurt` selector children.

### Adjacent-room callforhelp propagation

Extend `internal/mobcommands/callforhelp.go`:

- After the existing in-room announcement, the command iterates the
  current room's exits.
- For each adjacent room, fires a new `heard_callforhelp` btree event
  on every mob in that room whose routine matches the caller's or is
  in the caller's `routine_links`.
- Event context: `{caller_instance_id, caller_routine,
  from_exit_name}`.

Default archetype response to `heard_callforhelp`: `go <from_exit>`
back toward the caller, then fire the archetype's own in-room
pack-reaction on arrival (achieved by the normal combat loop — once
the mob arrives and sees an active combat, its `mob_combat_round` or
`packmate_hurt`-on-next-attack handlers fire).

### Archetype work

**Existing archetypes gain a `packmate_hurt` handler:**
- `generic_fighter` (wolf 243, zombie 301): engage the attacker
  (`attack @<attacker>`).
- `tank_taunter` (flesh golem 305, earth elemental 311, magma
  elemental 314): taunt + engage.
- `melee_self_buff` (vampire 304, air elemental 312, fire elemental
  313): cast best self-buff, then engage.

**New archetypes:**
- `lookout` — first tick: `callforhelp`. Next tick: engage.
- `healer_support` — find the most-wounded same-room packmate (HP
  ratio < threshold); cast heal on them; if no wounded packmates,
  fall through to engage.
- `leader` — rally/warcry (existing commands); then engage. Shares
  most of `tank_taunter`'s body but without the mandatory taunt.

**No `civilian` archetype needed.** A non-reacting mob has no
archetype; the engine skips dispatch.

### Deletion list

- `internal/mobs/mobs.go`: delete `MakeHostile`, `IsHostile`,
  `ReduceHostility`, `mobsHatePlayers` map, `mobsHatePlayersMu` mutex.
- `internal/mobs/memory.go:20-22`: delete the `mobsHatePlayers`
  memory-usage reporting block.
- `internal/hooks/NewRound_MobRoundTick.go:46`: delete the
  `mobs.ReduceHostility()` call.
- `internal/hooks/NewRound_DoCombat_unified.go`: remove the
  `MakeHostile` call at lines 666–669 in the PvM branch. Replace with
  `dispatchPackmateHurt(defMob, atk.GetUserId(), 0)`.
- `internal/mobcommands/lookfortrouble.go`: delete the group-hostility
  branch (the `for _, groupName := range mob.Groups` block that calls
  `mobs.IsHostile`). Keep Fix A's NoAggroTarget early-continue intact.
  Keep `mob.Hostile`, `HatesSpecies`, `HatesMob` branches intact.
- `internal/mobs/mobs_test.go`: delete `TestMakeHostileAndIsHostile`,
  `TestReduceHostility`, and any other tests that exercise the
  removed functions.

## Data flow

```
Player attacks bandit_caster
  ├─ existing: handleAggroAndAssist fires (PvM branch)
  ├─ existing: defMob aggro set, defMob issues attack-or-go command
  ├─ REMOVED: mobs.MakeHostile("humanoid"/"bandit", playerId, N)
  ├─ NEW:     dispatchPackmateHurt(bandit_caster, playerId, 0)
  │      ├─ FindPackmatesInRoom → [bandit_fighter, soren]
  │      │  (shared routine bandit_camp_guard, same room, alive,
  │      │   not charmed)
  │      └─ for each packmate: fire packmate_hurt btree event
  │         ├─ bandit_fighter (generic_fighter) → attack player
  │         └─ soren           (leader)         → rally → attack player
  └─ if any packmate invokes `callforhelp`:
       └─ callforhelp broadcasts heard_callforhelp to adjacent rooms
          └─ matching-routine mobs receive event, navigate in
```

Olen at the Temple: `routine: temple_service`, no `routine_links` to
`bandit_camp_guard`. `FindPackmatesInRoom(bandit_caster)` scans the
bandit's room — Olen isn't there. Even if he were, his routine
wouldn't match. He receives no event. Fix A's early-continue in
`LookForTrouble` is a secondary safety net (still valuable for future
direct-aggro edges); the primary fix is that Olen is outside the pack
entirely.

## v2 impression hook

Each archetype's `packmate_hurt` handler computes an impression
score before branching:

```
impression := impressionScore(self, victim)    // v1: always 0
if impression > strong_positive:
    react_hard()       // go out of way to help
elif impression < strong_negative:
    ignore()           // I don't like this packmate, let them bleed
else:
    default_for_archetype()
```

v1 stubs `impressionScore` to return 0 unconditionally so the default
branch always fires. No storage, no tick work. v2 replaces the stub
with a real sparse map and interaction hooks — no structural change
required here.

The routine structure is what makes impressions tractable later:
impressions accumulate from sharing a routine over time. Two wolves
with `routine: wolf_pack_ironwind` build positive impressions from
cohabitation. The bandit leader's subordinates accumulate *negative*
impressions from being bossed around (natural "tired of this guy"
emergence).

## Error handling / edge cases

- **Charmed mobs**: excluded from `FindPackmatesInRoom`. They follow
  their player owner, not their former wild pack. Matches the
  existing exclusion in `pack_roaming.go`.
- **Dead/downed victim**: still fires the event. Packmates can
  legitimately react to a comrade's death (often more aggressively).
- **Victim has no routine**: `FindPackmatesInRoom` returns empty
  (no one shares "no routine"). Solo mobs have no pack, which is
  correct.
- **Victim has routine but no packmates in room**: empty list, no
  event dispatched. Mob fights alone as before.
- **Ephemeral rooms (instanced zones)**: the same logic applies —
  same-room packmate scan uses the room regardless of whether it's
  ephemeral.
- **PvP (player-vs-player)**: pack dispatch runs only in the PvM
  branch of `handleAggroAndAssist`. PvP doesn't involve mob packs.
- **Race in `dispatchPackmateHurt`**: the function snapshots the
  packmate list before firing events, so a mob entering/leaving the
  room during dispatch won't affect this round's fan-out.

## Testing

### Unit tests (new)

- `TestFindPackmatesInRoom_SameRoutine` — two mobs, same room, same
  routine → packmate match.
- `TestFindPackmatesInRoom_RoutineLinks` — victim's routine appears
  in candidate's `routine_links` → match.
- `TestFindPackmatesInRoom_LinksReverse` — candidate's routine
  appears in victim's `routine_links` → match.
- `TestFindPackmatesInRoom_DifferentRoutine_NoMatch` — different
  routines, no links → no match.
- `TestFindPackmatesInRoom_DifferentRoom_NoMatch` — same routine,
  different rooms → no match.
- `TestFindPackmatesInRoom_CharmedExcluded` — charmed mob with
  matching routine → excluded.
- `TestFindPackmatesInRoom_DeadExcluded` — mob with Health=0 → excluded.
- `TestFindPackmatesInRoom_NoRoutineVictim` — victim has no routine
  → empty result.
- `TestDispatchPackmateHurt_FiresOnAllPackmates` — asserts the event
  fires on every packmate, none on non-packmates.
- `TestCallforhelpAdjacentRoomPropagation` — caller in room A with
  routine X; neighbor room B has one matching-routine mob + one
  non-matching mob; only the matching one receives
  `heard_callforhelp`.

### Regression tests (must continue passing)

- `TestLookForTrouble_SkipsGraceProtectedPlayer` (Fix A).
- All `pack_roaming.go` tests — no change to roaming logic.

### Tests to delete alongside removed code

- `mobs_test.go`: `TestMakeHostileAndIsHostile` and every other test
  that depends on `MakeHostile` / `IsHostile`.

### In-game smoke test plan

- Die near North Road bandits (quester9 scenario). Respawn in
  Thornwall. Temple priest Olen does *not* engage. (Tests Fix A stays
  holding + Olen is outside the new pack entirely.)
- Attack one bandit at the camp. Remaining bandits engage in the
  same round. Lookout (adjacent room) receives callforhelp and
  walks in.
- Attack a wolf in a pack. Other wolves in the room engage.
- Attack a civilian merchant. No packmate reaction. Merchant's own
  retaliation (via `handleAggroAndAssist`'s defender-aggro path) is
  unchanged.

## Migration path

Single feature branch. One merge.

1. **Infrastructure** — Mob struct fields (`Routine`,
   `RoutineLinks`), YAML load wiring, `FindPackmatesInRoom`,
   `dispatchPackmateHurt`, register `packmate_hurt` event, extend
   `callforhelp` with adjacent-room broadcast.
2. **Archetype authoring** —
   - Add `packmate_hurt` handlers to `generic_fighter`,
     `tank_taunter`, `melee_self_buff`.
   - Create `lookout`, `healer_support`, `leader` archetypes as new
     YAML files in `_datafiles/world/dogmud/behaviors/archetypes/`.
3. **Content migration — priority zones:**
   - `north_road`: bandits (fighter, caster, soren the leader),
     bloodline agent (remains unarchetyped, solo mob).
   - `ironwind_steppe`: goblin scouts, packs, wolves if grouped.
   - Instanced zones: arena champions, planar elementals (already
     have archetypes from Phase 4 — just add routines).
   - `dustwalk_road`: bandits in the ambush group.
   - Civilians / quest NPCs across all zones: no routine, no
     archetype. They stay passive.
4. **Deletion** — remove `MakeHostile`, `IsHostile`,
   `mobsHatePlayers`, the `MakeHostile` call in combat-unified, the
   group-hostility branch in `LookForTrouble`, and associated tests.
5. **Smoke-test in-game**, merge as one unit into development.

## Files touched (approximate)

Go:
- `internal/mobs/mobs.go` — `Routine`, `RoutineLinks` fields on
  `Mob`; delete `MakeHostile`, `IsHostile`, `ReduceHostility`,
  `mobsHatePlayers`, `mobsHatePlayersMu`.
- `internal/mobs/memory.go` — drop the `mobsHatePlayers` reporting
  block.
- `internal/hooks/NewRound_MobRoundTick.go` — drop the
  `ReduceHostility` call.
- `internal/mobs/packmates.go` (new) — `FindPackmatesInRoom`.
- `internal/mobs/mobs_test.go` — delete tests for removed functions;
  add `TestFindPackmatesInRoom_*`.
- `internal/hooks/NewRound_DoCombat_unified.go` — replace the
  `MakeHostile` call with `dispatchPackmateHurt`.
- `internal/hooks/packmate_hurt.go` (new) — dispatcher + adjacent-room
  callforhelp broadcast helper.
- `internal/mobcommands/callforhelp.go` — extend with adjacent-room
  event fan-out.
- `internal/mobcommands/lookfortrouble.go` — delete the
  group-hostility branch. Keep Fix A's grace guard.
- `internal/behaviortree/events.go` (or wherever events are
  registered) — register `packmate_hurt` and `heard_callforhelp`.

YAML:
- `_datafiles/world/dogmud/behaviors/archetypes/generic_fighter.yaml`,
  `tank_taunter.yaml`, `melee_self_buff.yaml` — add `packmate_hurt`
  handlers.
- `_datafiles/world/dogmud/behaviors/archetypes/lookout.yaml`,
  `healer_support.yaml`, `leader.yaml` — new files.
- `_datafiles/world/dogmud/mobs/north_road/*.yaml`,
  `ironwind_steppe/*.yaml`, `instance_arena/*.yaml`,
  `instance_planar_oasis/*.yaml`, `dustwalk_road/*.yaml` — add
  `routine` (and `routine_links` where appropriate) to pack-forming
  mobs.

## Success criteria

- Quester9 dies near North Road bandits, respawns in Thornwall: Olen
  never engages. (Primary bug fixed.)
- Attacking one bandit in the camp triggers same-room bandits to
  engage same round. Lookout in adjacent room receives callforhelp,
  walks in, engages.
- `TestLookForTrouble_SkipsGraceProtectedPlayer` still passes.
- `grep -rn "MakeHostile\|IsHostile\|mobsHatePlayers" internal/`
  returns no hits in production code paths (tests may reference
  deletion work).
- Full test suite green. Build green.
