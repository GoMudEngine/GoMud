# life — Package Documentation

## Overview

The `internal/state/life` package is the third consumer of the
`internal/state` framework, after `combatphase` and `awareness`.
It defines the **Life state machine**, replacing scattered
death-cleanup logic that previously lived in:
- `usercommands/suicide.go` (thin handler now, delegates to `Die`)
- `mobcommands/suicide.go` (same pattern)
- `MobDeath_*` hooks (collapsed into cascade observers)

Life has three states:

| State | Meaning |
|-------|---------|
| `Alive` | Normal operation; health above zero. |
| `Dead` | Character has died; cleanup cascade in flight. |
| `Respawning` | Player only; en route to respawn room. |

The machine is mob/player symmetric: both player and mob `Character`
instances carry a `*life.Machine` field. Mobs stay in `Dead` after
the transition fires; the instance-cleanup observer handles despawn
and mobs never enter `Respawning`. Players chain all three states in
a single `Die()` call.

---

## Key Components

### Core Files

- **life.go** — `State` enum, `AliveData`/`DeadData`/`RespawningData`
  types, `Machine` wrapper with predicate methods, machine registry.
- **transitions.go** — `validTransitions` table constant and Trigger
  string constants.
- **rules.go** — `TransitionToDead`, `TransitionToRespawning`,
  `TransitionToAlive`, `ForceAlive`.
- **life_test.go** — Behavior Matrix tests LI-001 through LI-027.

---

## State Diagram

```
Alive ──(health zero / suicide / admin kill)──> Dead
Dead  ──(respawn ready)──> Respawning ──(respawn complete)──> Alive
Dead  ──(force alive — admin)──> Alive
```

Dead → Alive direct path: admin restoration only. Standard player
respawn always goes through Respawning.

---

## Per-State Data

### AliveData

```go
type AliveData struct{}
```

Empty — the Alive state has no metadata.

### DeadData

```go
type DeadData struct {
    Reason    state.TransitionReason
    Killer    state.ActorRef
    DamageMap map[int]int  // userId → damage dealt
}
```

`DamageMap` is a **snapshot** of `Character.PlayerDamage` taken at
transition time by `characters.Die()` before the cascade fires.
Observers (kill credit, faction rep, party share) consume the
snapshot. `Life_Cascades.go` clears the live `PlayerDamage` field
during the `Dead → Respawning` transition, so the snapshot ensures
downstream consumers always see a stable value regardless of
cascade ordering.

### RespawningData

```go
type RespawningData struct {
    Reason     state.TransitionReason
    DestRoomId int  // graveyard or home room ID
}
```

Player-only. `DestRoomId` is resolved by `c.ResolveRespawnRoom()`
before `TransitionToRespawning` is called. The
`Respawn_PlayerTeleport.go` observer reads this field to move the
player to the correct room.

---

## Trigger Constants

Defined in `transitions.go`. Always use these constants instead of
inline string literals for stable identifiers across the codebase.

| Constant | Value | Purpose |
|----------|-------|---------|
| `TriggerHealthZero` | `"health_zero"` | Health dropped to zero or below during combat or condition tick |
| `TriggerSuicide` | `"suicide_command"` | Player typed `suicide` command |
| `TriggerAdminKill` | `"admin_kill"` | Admin used `kill` / `die` command on the target |
| `TriggerCleanupReady` | `"cleanup_ready"` | Internal: cascade cleanup complete (reserved) |
| `TriggerRespawnReady` | `"respawn_ready"` | `Die()` helper advances Dead → Respawning |
| `TriggerRespawnComplete` | `"respawn_complete"` | `Die()` helper advances Respawning → Alive |
| `TriggerForceAlive` | `"force_alive"` | Admin restoration skips normal cascade |

---

## Key Functions

### TransitionToDead

```go
func (m *Machine) TransitionToDead(d DeadData,
    r state.TransitionReason) error
```

Primary death entry point. All death paths (damage application,
suicide command, admin kill) route through here. Stores `DeadData`
before delegating to the inner framework so observers can read it
via `m.DeadData()` in their `AfterTransition` callback.

### TransitionToRespawning

```go
func (m *Machine) TransitionToRespawning(d RespawningData,
    r state.TransitionReason) error
```

Advances `Dead → Respawning`. Caller supplies `DestRoomId` (resolved
by `c.ResolveRespawnRoom()`). Clears `dead` pointer — callers that
held a snapshot from `DeadData()` before this call are unaffected.

### TransitionToAlive

```go
func (m *Machine) TransitionToAlive(r state.TransitionReason) error
```

Advances `Respawning → Alive` (normal respawn) or `Dead → Alive`
(admin restoration edge case). Clears all per-state data.

### ForceAlive

```go
func (m *Machine) ForceAlive(r state.TransitionReason)
```

Idempotent transition to `Alive` from any state. Used by admin
restoration commands. Does not fire the normal cascade; skips
directly to Alive.

### DeadData

```go
func (m *Machine) DeadData() (DeadData, bool)
```

Returns the death context while the machine is in `Dead` state.
Returns the zero value + `false` if not Dead.

### RespawningData

```go
func (m *Machine) RespawningData() (RespawningData, bool)
```

Returns the respawn context while the machine is in `Respawning`
state. Returns the zero value + `false` if not Respawning.

### Inner

```go
func (m *Machine) Inner() *state.Machine[State]
```

Returns the underlying `state.Machine[State]`. Used by `rules.go`
and hooks to register `AfterTransition` observers. Not part of the
stable caller API.

---

## Cascade Integration

Observers register via `m.Inner().AfterTransition(fn)` at
character-creation time (wired through
`characters.OnCharacterCreated`). Each observer receives the
`(from, to, reason)` triple and calls `m.DeadData()` or
`m.RespawningData()` to access state-specific context.

### Life_Cascades.go (cross-machine cleanup)

Fires on two transitions:

**Alive → Dead:**
- Forces Combat Phase to `Idle` (clears aggro)
- Forces Awareness to `Visible` (clears hidden state)
- Nils `CastingState` and `CraftingState`
- Resets `CombatPosition` to Standing
- Clears grapple controller
- Cancels all non-permanent buffs
- Clears active combat conditions

**Dead → Respawning:**
- Refills all resource pools to 5% of max
- Applies `NoAggroTarget` grace buff (#81)
- Clears `PlayerDamage` (live field; snapshot in `DeadData` is stable)
- Queues `CharacterVitalsChanged` event

### Death observers

| File | Purpose |
|------|---------|
| `Death_PlayerCleanup.go` | Stat decay, skill rust, KD tracking, party notifications |
| `Death_PlayerAnnouncement.go` | Room broadcast, global broadcast, `events.PlayerDeath` queue, worldevents PvE emit, weakened/darkness text, instance ejection |
| `Death_PlayerCorpse.go` | Player corpse creation in the death room |
| `Death_InboundAggroCleanup.go` | Clears mobs and companions targeting the dying actor; fires for both player and mob deaths |
| `Death_MobLoot.go` | Carried/equipped item drop, gold drop, dark-room sound, mob corpse creation |
| `Death_AlivenessSubstrate.go` | Fires `events.MobDeath`; downstream subscribers handle faction rep, opinion update, crime recording, knowledge propagation, bounty resolution |
| `Death_MobInstanceCleanup.go` | `DeleteMobInstance`, `DestroyInstance`, `CleanupMobSpawns`, `RemoveMob` |
| `Death_MobBroadcast.go` | Room "X has died" broadcast, Guide tempdata, worldevents `MobKilledByPlayer` |
| `Death_MobBehaviorTree.go` | Fires `mob_die` btree event with primary killer's `UserId` |
| `Death_MobKillCredit.go` | `EndAggro` on killers, `KD.AddMobKill`, `OnFirstMobKill`, party kill credit |
| `Death_MobCharmCleanup.go` | `TrackRecentDeath`, `RemoveCharm`, reverse-track player `TrackCharmed` |

### Respawn observers

| File | Purpose |
|------|---------|
| `Respawn_PlayerTeleport.go` | `rooms.MoveToRoom` to `ResolveRespawnRoom()` destination; belt-and-suspenders `EndAggro` |
| `Respawn_PlayerAutoLook.go` | Fires `u.Command("look")` for room-render UX after teleport |

### Wiring pattern

Each observer registers via `characters.OnCharacterCreated(wireXxx)`
at `init()` time in the hooks package. This avoids import cycles
(the `characters` package cannot import `hooks`; `hooks` imports
`characters` and registers via callback).

Gating conventions:
- Player-only observers check `c.GetUserId() != 0`
- Mob-only observers check `c.MobInstanceId != 0`

---

## Character Helper: Die()

`die.go` in `internal/characters` provides a convenience wrapper:

```go
func (c *Character) Die(killer state.ActorRef, trigger string)
```

Chains all three transitions in the correct order for the character
type:
- **Players:** `TransitionToDead` → `TransitionToRespawning` →
  `TransitionToAlive` (all same-tick; observers fire synchronously
  via `AfterTransition`)
- **Mobs:** Only `TransitionToDead` fires; the mob stays Dead until
  the instance-cleanup observer despawns it

Pre-conditions callers must satisfy before calling `Die`:
1. Check `ReviveOnDeath` buff and bail if present
2. Dedupe against `LastSuicideRound` (if the call site can double-fire)
3. Shadow Realm zone guard (player call sites only)

`Die()` is idempotent: if the machine is already Dead or Respawning
it returns immediately.

## Character Helper: ResolveRespawnRoom()

`respawn_home.go` in `internal/characters`:

```go
func (c *Character) ResolveRespawnRoom() int
```

Reads the player's `"home"` setting, looks it up in
`HomeLocations`, and falls back to `"default"` (Sanctum Basin
entrance, room 0) if unset or unrecognized.

`HomeLocations` and `HomeLocationNames` are exported maps consumed
by `sethome.go` (to validate setting values) and by the
`Respawn_PlayerTeleport.go` observer (to resolve the destination).

---

## Machine Registry

```go
var machineRegistry = map[state.ActorRef]*Machine{}
```

Guarded by `registryMu`. Populated by `RegisterMachine` at
character creation; cleared by `UnregisterMachine` on logout or
mob despawn.

The registry is the bridge between `ActorRef` (the identity type
used in cascade payloads) and the live `Machine` pointer. It allows
cross-character cascade lookups (e.g., inbound-aggro cleanup reading
the dying character's machine) without callers holding raw pointers.

---

## Character Field: MobInstanceId

```go
MobInstanceId int `yaml:"-"`
```

Non-persisted field added in chunk 2. Set to the mob's live
`InstanceId` value when a mob `Character` is initialized. Used as a
cheap gating check in Life observers: `c.MobInstanceId != 0` means
the character is a mob, without requiring a cast or lookup.

---

## Testing Notes

### life_test.go — Behavior Matrix

Tests follow the LI-NNN naming scheme from the chunk 2 spec. Each
test exercises one cell of the state × trigger matrix.

| Range | Area |
|-------|------|
| LI-001 – LI-003 | Basic death transitions (health zero, suicide, admin kill) |
| LI-004 – LI-006 | Respawning + Alive transitions |
| LI-007 – LI-015 | Cross-machine cascade (combat phase idle, awareness visible, buffs, conditions) |
| LI-015 – LI-016 | Respawn observers (teleport, auto-look) |
| LI-017 – LI-019 | ForceAlive from Dead and Respawning; idempotent Alive |
| LI-020 – LI-022 | Mob death observers (loot, MobDeath event, instance cleanup) |
| LI-023 – LI-025 | Player death observers (stat decay, KD, party) |
| LI-026 – LI-027 | DeadData snapshot stability; DamageMap cleared on Respawning |

All tests use local `Machine` instances; no server or database setup
required. Cross-machine cascade tests that touch `hooks` wiring sit
in `internal/hooks/Life_Cascades_test.go` if that file exists.

---

## Sunset Notes

The permadeath and extra-lives systems were removed in chunk 2,
Task 11. `events.PlayerDeath.Permanent` field remains in the event
struct for upstream compatibility but is always `false`. The
`GiveExtraLife()` scripting API function is a no-op stub retained
for upstream parity.
