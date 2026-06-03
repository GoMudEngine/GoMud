# Instanced Jail Cells (Design)

**Date:** 2026-06-03
**Status:** Design approved 2026-06-03
**Size:** M
**Area:** Town Justice (chunk 5.1) + instance-zone machinery

## Purpose

Replace the shared static "holding cell" rooms introduced in chunk 5.1 with a
**per-prisoner, single-room instanced jail cell**, reusing the existing
instance-zone machinery. Each arrested prisoner gets their own private cell that
only they can enter, created at arrest and torn down on release.

### Why

The static shared-cell approach (rooms 5106 Stillwater, the Thornwall barracks
cell) has structural problems:
- Two prisoners share one cell and can see/interact.
- Other players or mobs can wander/pursue into it (the 5.1c smoke BUG-04: a guard
  followed the prisoner into the cell and re-declared combat — patched with locked
  exits + the `NoAggroTarget` buff, but the cell is still a shared public room).
- It's a fixed room rather than a confinement primitive.

Per-prisoner instances solve all three: isolation, an auth-gate that only admits
the prisoner, and clean teardown. The instance machinery already exists; this is
mostly wiring.

## Current state (verified 2026-06-03)

- **Jail record:** `JailRecord` (`internal/justice/arrest.go:195`) stored on
  `Character.MiscData` via keys `justice_jail_until_round`,
  `justice_jail_fine_original`, `justice_jail_decay_per_round`,
  `justice_jail_faction`, `justice_jail_cell_room` (+ crime ids). Read back via
  `JailInfo()`. Persisted with the character.
- **Arrest:** `ExecuteArrest()` (`arrest.go:240`) computes fine/sentence, stamps
  the record, adds buff 88 (Jailed, carries `buffs.NoAggroTarget`), `EndAggro()`,
  then teleports the player to `FactionDefinition.HoldingCellRoom` via a move seam.
- **Release:** `ResolveDetention()` (`arrest.go:348`) — fired by timer expiry
  (`UntilRound <= now`) or fine payment (`internal/usercommands/jail.go`
  Fine/PayFine). Clears crimes (incl. faction allies), removes buff 88, clears
  jail keys, teleports to `FactionDefinition.ReleaseRoom` (fallback barracks 473).
- **Cells (data):** rooms 5106 (Stillwater) and the Thornwall barracks cell; exits
  locked (commit 5df39d90).
- **Instance machinery (`internal/rooms/instances.go`):**
  `CreateZoneInstance(zoneName, goldPaid, ownerUserId, authorizedUsers, overworldRoomId) (*ZoneInstance, error)`
  clones a zone marked `instanced: true` (`ZoneConfig`, `zoneconfig.go`) into
  ephemeral room ids, registers it in `instanceRegistry` (room-id → instance),
  adds a return-portal exit to the overworld room, and stamps room temp-data.
  `MoveToRoom()` (`roommanager.go:~310`) blocks movement into ephemeral rooms for
  non-authorized users. `CheckPortalTimers()` (`instances.go:144`) evicts +
  tears down on a TTL derived from `PortalDuration` — and **skips instances whose
  `PortalDuration` is empty** (`instances.go:162` `continue`). Teardown =
  `instanceRegistry.Remove(inst)` + `TryEphemeralCleanup(roomId)`.

## Locked design decisions

- **Rollout:** instanced cells are the **default** for all arrests; the static
  holding-cell room is a **failure-fallback** only (if instance creation fails).
  Static cell rooms are retained (not deleted) as that fallback.
- **Flavor:** **one generic** `instance_jail_cell` template room, instanced per
  prisoner; the arresting **faction name is woven into the cell description** via
  room temp-data.
- **Death in jail = released** (`death_policy: ejected`).
- **No-TTL lifetime:** the cell instance uses an **empty `portal_duration`**, so
  `CheckPortalTimers` never evicts or warns. Lifetime is owned explicitly by
  release / despawn teardown.
- **Instance creation extended** via an options struct (defaults preserve current
  dungeon behavior), not a parallel jail-only cloner.

---

## Section A — Jail Cell template zone (data)

A new zone `instance_jail_cell` with a `ZoneConfig` (YAML):

```yaml
instanced: true
portal_duration: ""        # empty → CheckPortalTimers skips it (no TTL, no warnings)
allow_recall: false        # no recalling out of jail
death_policy: ejected      # dying in jail = released
```

One template room "Holding Cell":
- **No usable exits** (the prisoner cannot walk out; combined with suppressed
  return portal below).
- Atmospheric, no-numbers description. A `{cell_faction}`-style token (or
  temp-data substitution) renders the arresting faction's flavor — e.g. "the
  Stillwater constabulary cell" vs "a Thornwall guard-barracks cell". The faction
  string is stamped on the ephemeral room at instance creation (Section C).
- No mobs, no aggressive spawns. Optional flavor item (a pallet/bucket).

Filename/zone-folder follows the engine's `ConvertForFilename` convention
(`instance_jail_cell`), mirroring the existing `instance_planar_oasis` /
`instance_arena` zones.

## Section B — Instance-creation options (`internal/rooms`)

Extend `CreateZoneInstance` to accept options without changing existing-caller
behavior. Add an options struct (or trailing variadic option), with one flag this
feature needs now:

- **`SuppressReturnPortal bool`** — when true, skip adding the
  return-to-overworld exit. A dungeon needs that exit (it's the way out); a cell
  must NOT have it, or the prisoner escapes by walking through it.

The no-TTL behavior needs **no new flag** — it falls out of an empty
`portal_duration` (the Section-A zone config) via the existing `instances.go:162`
skip. Existing dungeon callers pass the zero-value options (return portal added,
as today).

Implementation note: keep one creation/registration path (DRY). The current
function already special-cases the Oasis cube vs the generic
`CreateEphemeralZone` clone; the jail cell uses the **generic** single-room clone
path. `goldPaid` is irrelevant (no mobs to scale) — pass a safe non-zero value
(e.g. 1) to avoid any zero-multiplier edge in `ScaleSpawnStatPools`.

## Section C — `JailRecord.InstanceId` + `ExecuteArrest`

- Add `InstanceId int` to `JailRecord` and a `justice_jail_instance_id` MiscData
  key. `0` = not instanced (legacy/fallback).
- In `ExecuteArrest`, after fine/sentence computation and before the move:
  1. Attempt `CreateZoneInstance("Instance Jail Cell", goldPaid=1,
     ownerUserId=prisonerUserId, authorizedUsers=[]int{prisonerUserId},
     overworldRoomId=releaseRoomFn(faction), opts{SuppressReturnPortal:true})`.
  2. On success: stamp the arresting faction's display name onto the ephemeral
     cell room's temp-data (for the description token); set `cell =
     inst.EntryRoomId`; stamp `InstanceId = inst.EntryRoomId` (the registry keys
     on room id).
  3. On failure (e.g. ephemeral chunks exhausted): fall back to
     `cellRoomFn(faction)` (the static room), `InstanceId = 0`.
  4. If neither an instance nor a static cell is available (`cell == 0`): no
     arrest (existing behavior).
- The rest is unchanged: stamp the jail keys, add buff 88 scaled to `rounds`,
  `EndAggro()`, move the prisoner to `cell`. The move passes the `MoveToRoom`
  auth gate because the prisoner is in `authorizedUsers`.

## Section D — `ResolveDetention` teardown (dual-path)

On release (timer expiry OR fine paid), before/around the existing teleport:
- If `InstanceId > 0`: `inst := instanceRegistry.FindByRoomId(InstanceId)`; if
  found, `instanceRegistry.Remove(inst)` then `TryEphemeralCleanup(InstanceId)`.
- Clear the `justice_jail_instance_id` key alongside the other jail keys.
- Teleport to `releaseRoomFn(faction)` (unchanged). `InstanceId == 0` → legacy
  static path, teleport only. Handles both old and new prisoners.

## Section E — Abnormal-exit teardown (despawn / death / logout)

A no-TTL confinement instance leaks if the prisoner leaves by any path other than
`ResolveDetention`. Add a despawn/death hook (mirroring the existing
`PlayerDespawn_TrackingCleanup` pattern from chunk 2.8):

When a character with `justice_jail_instance_id > 0` **despawns** (logout or
disconnect) or **dies** in the cell:
1. Tear down the cell instance (`Remove` + `TryEphemeralCleanup`) and clear the
   stale `InstanceId` key (the ephemeral room is gone).
2. **Death** → treat as release: run `ResolveDetention` semantics (clear jail
   state, normal respawn). `death_policy: ejected` supports this.
3. **Logout/disconnect** → **keep** the rest of the jail record
   (`UntilRound`/fine/faction/crime-ids) so the sentence persists, and **rewrite
   the character's saved `RoomId`** to the faction's static fallback cell room so
   the character can never be saved pointing at a now-dead ephemeral room
   (limbo-proofing). Restoration happens on login (Section G).

## Section G — Login restore (logout/restart resilience)

The sentence clock `UntilRound` is an absolute game-round number on persisted
MiscData; the world ticks while the player is offline, so time served continues.
A login hook (fires on character load / enter-world) checks the jail record:

- **No active jail record** → normal login.
- **Jail record present and `UntilRound <= GetRoundCount()`** → sentence elapsed
  while away → run `ResolveDetention` (teleport to release room, clear state).
  The player returns **free**.
- **Jail record present and `UntilRound > GetRoundCount()`** → still serving →
  **re-create a fresh cell instance** (same `CreateZoneInstance` call as arrest),
  refresh buff 88 to the remaining rounds (`UntilRound - now`), stamp the new
  `InstanceId`, and place the prisoner inside. Sentence resumes in a fresh private
  cell.

The login hook is **authoritative on placement** for an active prisoner, so a
stale saved ephemeral `RoomId` never matters. This also covers a **server
restart** for free: all instances vanish on boot; the first login (or the
despawn-save before shutdown rewriting RoomId to the fallback) lands the player in
a real room, then the hook re-instances or releases based on persisted
`UntilRound`.

## Section F — Static-cell retention & validation

- Keep rooms 5106 and the Thornwall barracks cell as the harmless
  failure-fallback target; keep `FactionDefinition.HoldingCellRoom`/`ReleaseRoom`.
  No data deletion in this chunk.
- **Boot-validate** the new `instance_jail_cell` zone loads and is recognized as
  instanceable (no panic).
- **Unit tests** (the lifecycle is testable without a live client):
  - Arrest creates an instance, prisoner authorized, prisoner in the ephemeral
    entry room, `InstanceId` stamped.
  - Release (both timer and fine paths) removes the instance + frees the chunk +
    clears `InstanceId`.
  - Fallback path: when `CreateZoneInstance` fails, arrest uses the static cell
    (`InstanceId == 0`) and still works.
  - Logout-despawn keeps the jail record, tears down the instance, rewrites saved
    RoomId; login with `UntilRound > now` re-instances; login with
    `UntilRound <= now` releases.
  - Death-in-jail releases + tears down.
- **Manual in-game smoke (deferred to user):** commit a crime, get arrested,
  confirm you're alone in a cell you can't leave (no exits, recall blocked, no
  guard follows), pay the fine → released + (admin) confirm the ephemeral chunk
  was freed; log out mid-sentence and back in → resume in a fresh cell; wait out a
  sentence offline → return free.

## Obstacles & mitigations (summary)

| Obstacle | Mitigation |
|----------|-----------|
| Return portal lets prisoner walk out | `SuppressReturnPortal` option (Section B) |
| TTL eviction + "portal collapsing" warnings | Empty `portal_duration` → `CheckPortalTimers` skips (existing) |
| Forced placement vs auth gate | Prisoner is in `authorizedUsers`; move passes |
| Instance leak on non-release exit | Despawn/death teardown hook (Section E) |
| Saved into a dead ephemeral room | Despawn rewrites saved RoomId to static fallback; login hook authoritative (E + G) |
| Server restart with active prisoners | Persisted `UntilRound` + login restore (Section G) |
| Backward compat with any static prisoner | `InstanceId == 0` dual-path in release (Section D) |

## Out of scope

- Multi-room or shared jail instances (single room only).
- Deleting the static cell rooms / removing the legacy path (kept as fallback).
- Changes to fine economics, sentence length, or crime/verdict logic.
- Player-visible "you have N rounds left" UI changes beyond what 5.1 already does.
