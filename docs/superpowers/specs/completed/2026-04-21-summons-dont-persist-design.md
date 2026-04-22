# Summons Don't Persist — Design

**Date:** 2026-04-21
**Status:** Approved
**Related memory:** `project_instance_save_exits_corruption.md` (Fix D)

## Problem

`mobs.instances/summons/*.yaml` files persist runtime progression
(stat training, skills, skill use counts, mutations, mutation
progress) of summoned / conjured / raised / charmed companions.
These files are keyed by `(mobId, zone, mobName, homeRoomId)` — they
are tied to a *room*, not to an *owner*.

Observed symptom (2026-04-20): user dismisses air elemental, summons
a new one, and the new summon inherits the previous one's mutations
and training. Air elemental template edits (archetype, statpool)
also failed to take effect on running instances, because the stale
instance file overwrites template values at mob construction.

Additional repro path: a second player summoning the same template
in the same room inherits the first player's progression.

## Root cause

Two independent persistence systems carry the same data, and they
disagree in exactly the scenarios where they're supposed to cooperate.

1. **`CompanionInfo` on the owner's user YAML.** Carries
   `StatTraining`, `Skills`, `SkillUseCount`, `Mutations`,
   `SpellBook`, `MutationProgress`. Keyed by owner + instance id.
   Synced on logout (`saveCompanionState`), applied on login
   (`applyCompanionState`), cleared on death / dismiss / despawn
   (`RemoveCompanion`). This is the *correct* layer.

2. **`mobs.instances/summons/*.yaml` files.** Written every round
   tick from `NewRound_MobRoundTick.go:36` via `SaveMobInstance`.
   Read at spawn time from `mobs.go:345` via `LoadMobInstance` inside
   `NewMobById`. Keyed by room, not owner. This is the *wrong* layer
   for companion data, and only gets deleted on the single happy
   path of `despawn` on a clean dismiss of a player-crafted summon.
   Death in combat, dismiss-to-hostile of a charmed wild creature,
   server shutdown, and crash all leak the file.

The file layer is vestigial for companion-type mobs. CompanionInfo
already does the owner-scoping and lifecycle-clearing the file layer
tries (and mostly fails) to do. Layering them creates the bug.

## Design rule

> **If a mob is charmed to any user, it is a companion — its
> persistence lives on the owner's `CompanionInfo`, not in
> `mobs.instances/`. Full stop.**

"Charmed to any user" reliably identifies every current companion
source type (Summoned, Raised, Conjured, Charmed, future Pet
migration) — every code path that registers a companion calls
`mob.Character.Charm(user.UserId, …)` unconditionally. Mob-summoned
mobs (charm with `UserId == 0`) are also excluded from file
persistence, which is correct — they are ephemeral by nature.

The rule applies in both directions: don't write, don't read.

## Code changes

### 1. `internal/mobs/instance_save.go` — guard `SaveMobInstance`

Early-return at the top of `SaveMobInstance`:

```go
if mob.Character.IsCharmed() {
    return nil
}
```

Belt-and-suspenders. If any caller forgets the guard below, no file
is written.

### 2. `internal/mobs/mobs.go` — split constructors

`NewMobById` currently always calls `LoadMobInstance` at line 345
and applies saved progression. Split the read path:

- `NewMobById(mobId, roomId, statPool …)` — unchanged behavior for
  organic world spawns. Still loads instance data.
- `NewMobByIdFresh(mobId, roomId, statPool …)` — skips
  `LoadMobInstance`. Used by every companion-spawning path.

Callers to switch to `NewMobByIdFresh`:

- `internal/hooks/companion_summon.go:127` (summon / raise / conjure)
- `internal/hooks/PlayerSpawn_HandleJoin.go:62` (login respawn of
  previously-saved companions)
- `internal/behaviortree/actions_mob.go:~87`
  (mob-summoning-mob action, `summon_companion`)
- `internal/usercommands/buy.go:~564` (companion-vending-NPC)
- `internal/usercommands/character.go:~331,~421` (admin / suicide-
  vanish paths that create a charmed mob)

`internal/hooks/charm_spell.go` is **not** on this list: charm
operates on an existing `targetMob` already present in the room, no
new construction. Charmed wild creatures that organically progressed
before being charmed are the edge case covered under "Edge cases
consciously accepted" below.

### 3. Boot cleanup — `mobs.NukeSummonsInstances`

New function in `internal/mobs/instance_save.go`:

```go
// NukeSummonsInstances removes every file under
// _datafiles/.../mobs.instances/summons/ at boot. Companion
// persistence lives on CompanionInfo on the owner's user YAML; any
// file in this directory is stale and will poison the next summon of
// the same template in the same room.
func NukeSummonsInstances()
```

Implementation: `filepath.Walk`, count files removed, log one line:
`"mobs.NukeSummonsInstances pruned N files"`. No dry-run, no
per-file reporting.

Call site: `main.go:201`, immediately before (or after) the existing
`mobs.PruneStaleInstances(…)` call.

### 4. No changes needed to

- `CompanionInfo` struct, `saveCompanionState`, `applyCompanionState`,
  `respawnCompanions`, `RemoveCompanion`. These already do the right
  thing.
- `DeleteMobInstance` — still valid for non-companion mobs. Leave it.
- `PruneStaleInstances` — unchanged.

## Edge cases consciously accepted

- **Non-summons-zone mob with stale instance file from a prior
  charm session.** E.g., a thornwall wolf that was charmed last
  week — its instance file under `thornwall/` still persists,
  because the nuke only targets `summons/`. It will only poison the
  next respawn of that exact mob at that exact home room. The
  `PruneStaleInstances` age sweep handles it eventually. Accepted as
  rare / bounded.
- **Crash between `IsCharmed()` check and actual write.** No longer
  possible — the charm state is set synchronously at summon time
  (`companion_summon.go:135`) before any round tick fires, so the
  first `SaveMobInstance` call for a fresh summon will already see
  `IsCharmed() == true`.
- **A mob that *becomes* charmed mid-life.** Previous round ticks
  may have written a file before the charm. After the charm, no
  more writes. The old file on disk is orphaned; next respawn of
  that mob at that home room could still read it. Mitigation: the
  prune-on-age sweep. Accepted.

## Out of scope / follow-up

- **`CompanionConjured` source type is declared but never assigned.**
  All conjure spells currently register as `CompanionSummoned`
  because they flow through the same `companion_summon.go` path and
  the branch at line 141-143 only distinguishes raised from
  summoned. Separate cleanup commit.
- **Rename `IsCharmed` → `IsCompanion`.** The function name lags the
  semantic — "charmed" is the mechanism, "companion" is the
  meaning. Touches many call sites; scope creep here. Separate PR.
- **Fix A (rooms-exits instance save corruption).** Different code
  path, same root-cause family. Already investigated in
  `project_instance_save_exits_corruption.md`. Will be landed
  alongside this work but not part of this spec.
- **`mobs.instances/` for genuine non-charmed world mobs.** Still
  operates as designed. Zone wolves, etc., continue to progress and
  persist. No change.

## Testing

### Unit tests (Go)

1. **`TestSaveMobInstance_CharmedMobSkipsWrite`** — construct mob,
   charm to userId=1, mutate progression, call `SaveMobInstance`,
   assert no file written.
2. **`TestSaveMobInstance_UncharmedMobWritesFile`** — construct mob,
   *don't* charm, mutate progression, assert file written.
3. **`TestNewMobByIdFresh_IgnoresInstanceFile`** — write a deliberate
   instance file, call `NewMobByIdFresh`, assert progression is
   template-default (not file values).
4. **`TestNukeSummonsInstances_RemovesAll`** — seed 3 fake files
   under a temp `summons/`, call nuke, assert empty directory.
5. **`TestNukeSummonsInstances_IgnoresOtherZones`** — seed files in
   `summons/` AND `thornwall/`, assert only `summons/` gets nuked.

### Smoke tests (server)

1. Summon air elemental, train skills, dismiss, summon again —
   confirm second summon has clean stats.
2. Summon air elemental, train it, log out cleanly, log back in —
   confirm companion respawns with training preserved (CompanionInfo
   layer still works).
3. Two characters summon air elemental in the same room back to
   back — confirm no cross-pollination.
4. Charm wild creature, train with it, dismiss (turns hostile),
   restart server — confirm next wild creature of that type spawns
   fresh.
