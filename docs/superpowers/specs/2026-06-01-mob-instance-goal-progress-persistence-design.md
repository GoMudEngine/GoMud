# Mob Instance Goal-Progress Persistence — Design

**Date:** 2026-06-01
**Status:** Approved (design); ready for plan
**Author:** Claude + Calabe

## Problem

Strategic-goal mobs (chunk 4.4 / 5.2 / 5.3) pursue goals that accumulate
*progress*: a `wealth-gold` mob sells junk to save up gold; an
`upgrade-gear` mob saves up, walks to a vendor, and buys a better item.
The goal itself (the `upgrade-gear` Goal) persists across despawn — but the
**progress toward it does not**.

Mob instance persistence (`internal/mobs/instance_save.go`,
`MobInstanceData`) saves *only* stat training, skills, use counts, and
mutations. It does **not** persist gold, equipment, or planner working
state. So when a goal-bearing mob despawns (by-design performance despawn
on room unload — `rooms.removeRoomFromMemory` →
`mobs.DestroyInstance`) and later respawns, it is rebuilt from its YAML
template:

- Gold reverts to the template value — saved-up gold is gone.
- Equipment reverts to the template loadout — a bought upgrade vanishes.
- Planner working state (`plan:`-prefixed MiscData) is gone — the mob
  forgets which shop room it was walking to.

The result is a Sisyphean loop: a mob can spend its whole life saving for
an upgrade, despawn one tick before (or after) the purchase, and come back
with nothing to show for it. The MOTD-advertised "NPCs pursue their own
goals / show up in better gear next time" never materializes for any mob
that has despawned at least once — which, given the performance despawn,
is effectively all of them.

## Goal

Persist the *outcomes and in-flight state* of goal pursuit — **gold,
equipment, and planner working state** — across the performance despawn,
so a respawned mob resumes where it left off. Killing or admin-despawning
a mob still resets it (those paths already wipe the save).

Out of scope: persisting backpack inventory (vendor trash). Accepted
consequence: junk a mob is carrying but hasn't sold yet vanishes on
despawn (a small one-time gold loss for that NPC).

## Key Mechanics (verified)

- **Perf-despawn does NOT delete the save.** `removeRoomFromMemory`
  (`internal/rooms/roommanager.go:475`) calls `mobs.DestroyInstance` per
  mob, which only drops the in-memory instance (`mobs.go:693`,
  `delete(mobInstances, …)`). The instance file from the last periodic
  save stays on disk. Only **death** (`Death_MobInstanceCleanup.go`),
  **admin `despawn`** (`mobcommands/despawn.go`), and **`suicide`** call
  `DeleteMobInstance` (wipe). So preservation across perf-despawn is
  possible without touching the delete policy.
- **Saves are periodic only** today (`NewRound_MobRoundTick.go`, every
  `MobSaveIntervalRounds`) — *not* at despawn. A purchase made between the
  last periodic save and the despawn would be lost unless we also save at
  despawn.
- **Restore overlay** lives at `mobs.go:417` (inside `NewMobById`),
  applied after the mob is built from template when a saved instance
  exists.
- **Save gate** is `hasProgression(mob)` (`instance_save.go:211`) — true
  only for stat training / skills / use counts / mutations. Mobs whose
  only change is gold/equipment currently never save at all.
- **Equipment serializes cleanly.** `characters.Worn`
  (`characters/worn.go:8`) is a flat struct of YAML-tagged `items.Item`
  slots; `items.Item` (`items/items.go:40`) carries all enchant/affix
  state (`enchanttier`, `enchanttype`, `enchantuses`, `adjectives`,
  `overrides`). `UUID` is `yaml:"-"` (regenerated on load — runtime
  identity, not persistent state). A marshal round-trip preserves the
  full loadout. `mobs` already imports `characters`.
- **Planner working state is namespaced.** Every planner stores
  intermediate state under the `plan:` MiscData prefix
  (`planners.PlanKeyPrefix`, `planners/state.go:12`), wiped by
  `ClearPlanState` on goal switch. Persisting plan state = persisting all
  `plan:`-prefixed MiscData keys.
- **Mobs do not drop gold to killers.** No `GoldDrop` spec field and no
  kill-gold-reward path exists anywhere in the codebase. Persisting mob
  gold therefore creates **no kill-farm exploit** — a hoarding mob killed
  by a player yields the player nothing. Persisted gold only matters for
  the mob's own shopping, and re-enters circulation only when the mob buys
  (a conserved transfer). No inflation.
- **Template baseline is available.** `GetMobSpec(mobId)`
  (`mobs.go:643`) returns a copy of the template mob, so the save gate can
  compare live gold/equipment against template values cheaply.

## Design

### 1. Extend `MobInstanceData`

Add three fields with **explicit-presence semantics** so that a legitimate
"spent all gold" (Gold = 0) or "stripped gear" (empty Worn) state restores
correctly, and so old training-only save files migrate transparently
(all three fields nil → no overlay → identical to today's behavior):

```go
type MobInstanceData struct {
    // … existing training/skills/mutations fields …

    // Goal-progress persistence (2026-06-01). Pointers / nil-able so that
    // "absent in the save" (old file or non-goal mob) is distinguishable
    // from a real zero value (mob spent all gold / stripped all gear).
    Gold      *int             `yaml:"gold,omitempty"`
    Equipment *characters.Worn `yaml:"equipment,omitempty"`
    PlanState map[string]any   `yaml:"plan_state,omitempty"`
}
```

### 2. Capture (`SaveMobInstance`)

Inside the existing `SaveMobInstance`, after the training/skills/mutations
block, capture the three new fields. They are captured whenever the mob is
deemed persistable (see gate); the captured value is the live truth, so
capturing unconditionally (even when equal to template) is harmless — the
restore overlay is then a no-op.

```go
gold := mob.Character.Gold
data.Gold = &gold

eq := mob.Character.Equipment // value copy of the Worn struct
data.Equipment = &eq

if planState := collectPlanState(mob); len(planState) > 0 {
    data.PlanState = planState
}
```

`collectPlanState` scans `mob.Character.MiscData` for `plan:`-prefixed
keys:

```go
// planKeyPrefix MUST match planners.PlanKeyPrefix. Duplicated here (not
// imported) because planners imports mobs — referencing it the other way
// would form an import cycle.
const planKeyPrefix = "plan:"

func collectPlanState(mob *Mob) map[string]any {
    if mob.Character.MiscData == nil {
        return nil
    }
    out := map[string]any{}
    for k, v := range mob.Character.MiscData {
        if strings.HasPrefix(k, planKeyPrefix) {
            out[k] = v
        }
    }
    return out
}
```

### 3. Broaden the save gate

Replace the `hasProgression(mob)` early-return in `SaveMobInstance` with
`hasPersistableState(mob)`. This keeps the gate *meaningful* — without it,
broadening to "has gold or equipment" would write a file for nearly every
mob in the world every save interval (a disk/perf regression the despawn
optimization exists to avoid). The template comparison runs only at save
cadence + despawn, so its cost is negligible.

```go
func hasPersistableState(mob *Mob) bool {
    if hasProgression(mob) {
        return true
    }
    if len(collectPlanState(mob)) > 0 {
        return true
    }
    tmpl := GetMobSpec(mob.MobId)
    if tmpl == nil {
        return false
    }
    if mob.Character.Gold != tmpl.Character.Gold {
        return true
    }
    return equipmentDiffers(mob.Character.Equipment, tmpl.Character.Equipment)
}
```

`equipmentDiffers` compares the two `Worn` values by their marshaled YAML
bytes. `UUID` is `yaml:"-"` (excluded) and the unexported `tempDataStore`
is not marshaled, so byte-equality is a clean value comparison that
ignores per-instance identity and correctly detects a changed itemId or
enchant tier in any slot:

```go
func equipmentDiffers(a, b characters.Worn) bool {
    ab, _ := yaml.Marshal(&a)
    bb, _ := yaml.Marshal(&b)
    return !bytes.Equal(ab, bb)
}
```

### 4. Save-at-despawn

In `removeRoomFromMemory` (`internal/rooms/roommanager.go`), save each mob
immediately before destroying it, so a buy/sell/equip since the last
periodic save isn't lost on room unload. `SaveMobInstance` is gated
internally (no-op for unchanged mobs) and idempotent, so saving in both
existing destroy loops is safe. Save failures are logged, not fatal —
room unload must not block on a save error.

```go
for _, mobInstanceId := range room.mobs {
    if m := mobs.GetInstance(mobInstanceId); m != nil {
        if err := mobs.SaveMobInstance(m); err != nil {
            mudlog.Error("removeRoomFromMemory save", "instanceId", mobInstanceId, "error", err)
        }
    }
    mobs.DestroyInstance(mobInstanceId)
}
// … SpawnInfo loop: SaveMobInstance(m) before its DestroyInstance(…) …
```

### 5. Restore overlay (`mobs.go:~437`)

After the existing training/skills/mutations overlay inside the
`savedInstance != nil` branch, overlay the three new fields, each guarded
by presence (nil → leave the template value untouched):

```go
if savedInstance.Gold != nil {
    mob.Character.Gold = *savedInstance.Gold
}
if savedInstance.Equipment != nil {
    mob.Character.Equipment = *savedInstance.Equipment
}
if savedInstance.PlanState != nil {
    if mob.Character.MiscData == nil {
        mob.Character.MiscData = map[string]any{}
    }
    for k, v := range savedInstance.PlanState {
        mob.Character.MiscData[k] = v
    }
}
```

Raw field assignment mirrors how templates load equipment — no `Wear()`
re-validation is needed. Mutation-gated slots (extra-arm gear) restore
correctly because `Mutations` are overlaid in the same pass, and the
overlay bypasses slot-gating entirely (the loadout was already legal when
saved). Equipment stat bonuses are computed on demand via `Worn.StatMod`,
so no recompute is required.

### 6. Reset paths unchanged

Death, admin `despawn`, and `suicide` still call `DeleteMobInstance`
(wipe). Only the perf-despawn (room unload) preserves. So a *killed* mob
respawns template-fresh; a *room-unloaded* mob respawns with its hoard +
bought gear. Charmed mobs remain skipped by the existing `IsCharmed`
guard at the top of `SaveMobInstance`.

## Consequences

- **No gold inflation, no kill-farm.** Buy/sell are conserved transfers;
  persistence just remembers a conserved balance. Mobs don't drop gold to
  killers.
- **Session-bounded hoards.** Death resets a mob's save; prod redeploys
  don't ship `mobs.instances/`, so all hoards reset on deploy. A mob
  cannot accumulate unbounded gold across the lifetime of the world.
- **Per-save cost** grows by one small equipment block + plan-state map
  per *changed* mob. Unchanged mobs still skip entirely (the gate). The
  added despawn-time save is gated identically.
- **Vendor-trash loss.** A mob's un-sold backpack items vanish on
  despawn (not persisted). Accepted.
- **Coupled to `MobProgressionEnabled`.** Gold/equipment/plan-state
  persistence rides the same master switch as stat-training persistence
  (the early-return in `SaveMobInstance`). Disabling mob progression makes
  mobs fully static, including their goal progress. This is intentional
  (one "mobs evolve over time" switch); a separate knob can be split out
  later if needed (YAGNI for now).
- **Migration is transparent.** Old training-only save files unmarshal
  with `Gold`/`Equipment`/`PlanState` all nil → restore overlays nothing
  new → identical to current behavior. No migration step required.

## Dependencies & Coordination

- **Ghost-guards fix** (`fix/ghost-guards-spawn-schedule-mismatch`,
  `project_ghost_guards_spawn_schedule_mismatch`) also touches
  `removeRoomFromMemory`. Land/rebase so the save-at-despawn insertion and
  the ghost-guards change compose cleanly. Functionally orthogonal (one
  adds a save call, the other fixes a duplicate room-list entry), but they
  edit adjacent code.
- The **named-combat-mob despawn** issue
  (`project_goal_planner_inert_for_combat_mobs`) is what makes this fix
  observable end-to-end in game (a named mob must stay alive long enough to
  save up, then despawn-and-return geared). The unit tests below stand on
  their own; the in-game smoke depends on a live combat mob, which the
  ghost-guards fix enables.

## Testing

### Unit (package `mobs`, `instance_save_test.go`)

1. **Round-trip — gold.** Save a mob with non-template gold; load; assert
   restored gold equals saved (including the `Gold = 0` case via the
   pointer).
2. **Round-trip — equipment with enchant.** Equip an item with an enchant
   tier/type; save; load; assert the slot's itemId + enchant tier/type
   survive.
3. **Round-trip — plan state.** Set two `plan:`-prefixed MiscData keys +
   one non-`plan:` key; save; load; assert only the `plan:` keys are
   restored.
4. **Gate.** `hasPersistableState` is false for a fresh template mob;
   true after (a) gold change, (b) equipment change, (c) a `plan:` key,
   (d) training (existing behavior preserved).
5. **Migration.** A `MobInstanceData` with all three new fields nil
   (old-format) overlays nothing — template gold/equipment/MiscData
   preserved.
6. **Stripped-gear / spent-gold.** A mob saved with empty `Worn` and
   `Gold = 0` restores to empty equipment and 0 gold (not template
   values), proving presence-semantics work.

### Integration / smoke

7. **Save-at-despawn.** Drive a changed mob into a room with no players,
   trigger `removeRoomFromMemory`, assert an instance file exists with the
   mob's current gold/equipment.
8. **In-game end-to-end (manual, blocked on ghost-guards):** a live combat
   mob with `upgrade-gear` current saves up, buys a vendor upgrade, the
   room unloads, the mob respawns still wearing the upgrade with leftover
   gold. Document in the chunk smoke report.

## Files Touched

- `internal/mobs/instance_save.go` — struct fields, capture,
  `collectPlanState`, `hasPersistableState`, `equipmentDiffers`,
  `planKeyPrefix` const, new imports (`bytes`, `strings`).
- `internal/mobs/mobs.go` — restore overlay in the `savedInstance != nil`
  branch.
- `internal/rooms/roommanager.go` — save-at-despawn in
  `removeRoomFromMemory` (both destroy loops).
- `internal/mobs/instance_save_test.go` — new unit tests.
