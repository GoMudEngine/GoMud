# Package: internal/state/presence

Per-character Presence state machine for player AFK / connection
lifecycle and mob boredom / despawn lifecycle.
Chunk 5 (2026-05-19).

## Overview

The `internal/state/presence` package is the sixth consumer of the
`internal/state` framework, after combatphase, awareness, life,
activity, and position. It centralizes "is this character
meaningfully present?" into one canonical FSM per character.

**Design:** one `Machine[State]` type, two transition tables (one for
players, one for mobs), and a single union enum with 8 states. Per-actor
polymorphism is enforced by the transition tables, not the type system
— both actor types carry a `*presence.Machine` on their Character.
`Active` is the only shared state.

**Sunsets (chunk 5):** `ManualAFK` / `AFKMessage` on `UserRecord`;
`BoredomCounter` / `PreventIdle` on `Mob`; `MaxMobBoredom` config knob.

---

## States

| State | Actor | Semantics |
|-------|-------|-----------|
| `Active` | Both | Normal — ticking, receivable, eligible for combat. |
| `Connecting` | Player | Logged in but not yet placed in a room. |
| `Idle` | Player | No input for `PresenceIdleAfterRounds`; still receivable. |
| `AFK` | Player | No input for `PresenceAFKAfterRounds`, or manual `afk` cmd. |
| `Disconnected` | Player | TCP gone. Terminal until reconnect. |
| `Spawning` | Mob | Freshly created; auto-advances to Active on next tick. |
| `Dormant` | Mob | Bored (`lastTargetFoundRound` stale) or zone has no players. |
| `Despawning` | Mob | Scheduled for removal; terminal one-tick hold. |

**Receivability note:** Idle, AFK, and Dormant characters are all still
targetable and take hits. Going AFK in a dangerous room is the player's
choice. Only Disconnected (no room ID, character in graveyard) and
Despawning are removed from the targetable pool via the CombatPhase
veto.

---

## Key Components

- **presence.go** — `State` enum (8 values), `AFKData` struct, `Machine`
  wrapper with `TransitionTo` / `TransitionToAFK` / `RegisterVeto` /
  `RegisterObserver` / `Inner`. `NewPlayerPresence()` starts in
  `Connecting`; `NewMobPresence()` starts in `Spawning`.
- **transitions.go** — `playerTransitions` and `mobTransitions` tables
  (two separate `TransitionTable[State]` constants) + 13 trigger string
  constants (`TriggerInputReceived`, `TriggerManualAFK`, etc.).
- **presence_test.go** — Behavior Matrix unit tests PR-001 through
  PR-031 (one test per matrix row from the spec).
- **integration_test.go** — Integration tests for the CombatPhase veto
  (PR-030) and the scheduler-cancel observer (PR-031).

---

## Construction

```go
func NewPlayerPresence() *Machine  // starts in Connecting
func NewMobPresence()   *Machine   // starts in Spawning
```

`NewPlayerPresence` is called in `characters.New()` for the player
code path. `NewMobPresence` is called in `mobs.Mob.Validate()` after a
fresh shallow copy from `newMobByIdInternal`. Both are wrapped in
`characters.OnCharacterCreated` wire callbacks (T2) so hook observers
are registered immediately on first Validate.

---

## Transition Tables

### Player

```
Connecting   → Active (entered room)
Active       → Idle, AFK, Disconnected
Idle         → Active, AFK, Disconnected
AFK          → Active, Disconnected
Disconnected → (terminal — new Machine on reconnect)
```

`[any non-Disconnected] --input received--> Active`
`[Active/Idle]          --manual afk cmd--> AFK`
`[any]                  --TCP closed------> Disconnected`

### Mob

```
Spawning   → Active (auto next tick)
Active     → Dormant, Despawning
Dormant    → Active, Despawning
Despawning → (terminal — removed next tick)
```

Vetoes registered on `Active→Dormant` and `Active→Despawning` return
`ErrVetoed` when `IsEssential() || IsCharmed()`. Shopkeepers, foragers,
caravan crew, charmed companions never leave Active.

---

## Integration Points

### T6 — CombatPhase veto on `Idle→Engaging`

Block list: `Disconnected` and `Despawning`. Wired by
`hooks.wireCombatPhaseVetoes` via `RegisterTargetPresenceCheck` on the
target's CombatPhase machine. When the attacker calls
`CombatPhase.TransitionToEngaging`, the veto fires on the target's
Presence state and returns `ErrVetoed` for either of the two terminal
states. Idle / AFK / Dormant targets are NOT blocked — they are still
attackable. See `internal/state/combatphase/context.md`.

### T5 — Essential-mob veto on `Active→Dormant` / `Active→Despawning`

Wired by `hooks.wireMobPresenceVetoes` in
`internal/hooks/Presence_MobVetoes.go`. The veto reads
`mob.IsEssential() || mob.Character.IsCharmed()`. Shopkeepers,
foragers, caravan crew, and charmed companions return to Active
indefinitely.

### T8 — Scheduler-cancel observer on terminal-state entry

An `AfterTransition` observer fires when `to == Disconnected` or
`to == Despawning`. It calls `Character.CancelAllScheduled()`, which
cancels pending Activity / Position / etc. scheduled transitions for
that character. Wired by `hooks.wirePresenceSchedulerObserver` via
`OnCharacterCreated`.

### T4 — `NewRound_PresenceTick` hook

Fires once per round (after `DoCombat`, before `IdleMobs`). For each
player it reads `roundNow - lastInputRound` and compares to the three
player thresholds. For each mob it reads
`roundNow - lastTargetFoundRound` and compares to the two mob
thresholds. Fires the relevant `TransitionTo` call through the machine.
See `internal/hooks/context.md` for ordering rationale.

### T11 — `RoomChange_PresencePlayerEntry` wake for Dormant mobs

Registered as a `RoomChange` event listener. When a player enters a
room, any Dormant mob in that room is transitioned `Dormant→Active`
with trigger `TriggerPlayerEntry`. Wired in
`internal/hooks/Presence_MobWake.go`.

### T7 — Auto-wake on attack

Attack paths (`AttackPlayerVsMob`, `AttackMobVsMob`) call
`target.Presence.TransitionTo(Active, ...)` with trigger
`TriggerAttacked` when the target is Dormant. Fires BEFORE damage
resolves so the mob is Active when the round's logic runs. Single
call site in `actions.ResolveTargetActor` callers.

### T9 — Connection lifecycle

- `HandleJoin` in `users/users.go`: fires `Connecting→Active` when
  the character is placed in its starting room (`TriggerEnteredRoom`).
- `LogOutUserByConnectionId` in `users/users.go`: fires any→`Disconnected`
  on TCP close (`TriggerTCPClosed`).
- `SetLastInputRound` in `userrecord.go`: calls
  `Presence.TransitionTo(Active, ...)` on any non-Disconnected state
  (`TriggerInputReceived`), replacing the old manual-AFK clear shim.

---

## Configuration

Five knobs under `Server` in `_datafiles/config.yaml`:

| Knob | Default | Effect |
|------|---------|--------|
| `PresenceIdleAfterRounds` | 8 (~30s) | `Active→Idle` |
| `PresenceAFKAfterRounds` | 75 (~5min) | `Idle→AFK` |
| `PresenceDisconnectAfterRounds` | 900 (~1hr) | `AFK→Disconnected` |
| `PresenceMobDormantAfterRounds` | 30 | `Active→Dormant` |
| `PresenceMobDespawnAfterRounds` | 60 | `Dormant→Despawning` |

`MaxMobBoredom` (old `Memory` section) is removed and superseded by
`PresenceMobDormantAfterRounds`.

---

## Sunset List

| Deleted artifact | Notes |
|---|---|
| `UserRecord.ManualAFK bool` | Replaced by `Presence.State() == AFK` |
| `UserRecord.AFKMessage string` | Replaced by `AFKData.Message` |
| Ad-hoc `isAfk` computation in `userrecord.go:543` | Replaced by `Presence.State() == AFK` |
| `Mob.BoredomCounter uint8` | Semantics live in `Character.LastTargetFoundRound` + Presence |
| `Mob.PreventIdle bool` | Subsumed by the essential-mob veto on `Active→Dormant` |
| `Memory.MaxMobBoredom` config knob | Replaced by `PresenceMobDormantAfterRounds` |
| `mob.Command("despawn ...")` call | Replaced by Presence terminal-tick removal |

**Preserved:** `Mob.WanderCount` / `MaxWander` (orthogonal),
`Mob.IsEssential()` (now the veto policy hook), `Mob.Despawns()`
(called inside the veto), `UserRecord.lastInputRound` (source of truth
for input timing), `OnlineInfo.IsAFK` (compat shim reading
`Presence.State() == AFK`).
