# State Chunk 5 — Presence (Design)

**Status:** Draft 2026-05-19 — awaiting user review before writing-plans handoff
**Branch:** `feature/mob-aliveness-1.3-crimes` (continuation from chunks 4a-4f)
**Predecessor chunks:** 4f (closed 2026-05-19), 4e, 4d, 4c, 4b-fixup-2, 4b-fixup, 4b, 4a, 3, 2, 1, 0
**Successor chunks:** 6 (Perception). Chunk 5 closes the combat-state-machines arc except for Perception.
**Master spec:** `docs/superpowers/specs/2026-05-13-combat-state-machines-design.md` §6

---

## 1. Problem Statement

Player AFK state is computed ad-hoc from `lastInputRound` and a `ManualAFK` bool, with no formal state machine. Mob "boredom" and despawning is driven by counters (`BoredomCounter`, `WanderCount`, `PreventIdle`) and a one-shot `mob.Command("despawn ...")` call from `NewRound_IdleMobs.go`. Multiple scattered call sites read these signals to decide aggro, visibility, and tick eligibility. The headline bug from the master spec: an AFK player in a dangerous room can still take aggro and combat starts mid-fold, despite no input ever arriving — the existing checks aren't applied consistently.

Chunk 5 introduces the Presence state machine (the sixth and last of the combat-state-machines arc, before Perception). It centralizes "is this character meaningfully present?" into one canonical FSM per character, with per-actor polymorphism in the state list. Combat Phase gating, scheduled-transition cancellation, and the existing essential-mob exclusions all migrate to the new machine.

---

## 2. Design Goals

1. **One canonical Presence machine per character, with per-actor state lists encoded in a union enum.** Same `Machine[State]` type as other chunks; different transition tables for player vs mob actors.
2. **Combat Phase gating with intent-driven block list.** Disconnected and Despawning block `Idle→Engaging` transitions. Idle and AFK players still take hits — going AFK in a dangerous room is the player's choice.
3. **Auto-wake on attack for Dormant mobs.** A targeted attack from any actor transitions Dormant → Active before damage resolves. Receivability stays intact while the mob's per-round tick is skipped.
4. **Essential-mob veto.** Shopkeepers, foragers, caravan crew, charmed companions never transition out of Active. Veto handlers wrap `IsEssential() || IsCharmed()` at the framework level.
5. **Scheduled-transition cleanup on terminal states.** Despawning and Disconnected cascade to `RoundScheduler.CancelAllFor(character)`, wiping pending Activity / Position / etc. timers.
6. **Sunset the legacy ad-hoc fields and call sites.** `ManualAFK`, `AFKMessage`, `BoredomCounter`, `PreventIdle` deleted at chunk end. UI compat shim for `OnlineInfo.IsAFK` preserves observable behavior.

---

## 3. State List and Transition Tables

### 3.1 Union enum

```go
package presence

type State int

const (
    Active       State = iota  // both actors — normal "in world, ticking"

    // Player-only
    Connecting                  // logged in but not yet in a room
    Idle                        // no input for N rounds (soft)
    AFK                         // no input for M rounds OR manual `afk` cmd
    Disconnected                // TCP gone, character in graveyard

    // Mob-only
    Spawning                    // freshly created; auto → Active next tick
    Dormant                     // bored OR zone has no nearby players
    Despawning                  // scheduled for removal next tick
)
```

Active is shared between actors. The remaining states are per-actor — the transition tables make this explicit; nothing prevents a mob from being passed a `Connecting` state value at the type level, but the mob transition table won't accept it.

### 3.2 Player transition table

```
Connecting --in-world (character entered room)--> Active
Active     --N rounds no input------------------> Idle
Idle       --M rounds no input------------------> AFK
AFK        --hours OR TCP timeout---------------> Disconnected

[any non-Disconnected] --input received---------> Active
[any non-Connecting]   --manual `afk` cmd-------> AFK (AFKData.Manual=true)
[any]                  --TCP closed-------------> Disconnected
```

### 3.3 Mob transition table

```
Spawning   --auto next tick-------------------> Active
Active     --bored N rounds && !essential-----> Dormant
Active     --zone has no nearby players-------> Dormant
Dormant    --player nearby OR attacked--------> Active
Dormant    --too long alone && !essential-----> Despawning
Despawning --next tick------------------------> (removed)
```

Vetoes registered on `Active→Dormant` and `Active→Despawning` return `ErrVetoed` when `mob.IsEssential() || mob.Character.IsCharmed()`. Existing `Despawns()` predicate becomes the veto's policy hook. Shopkeepers, foragers, caravan crew, charmed companions stay Active permanently.

### 3.4 Per-state data

Only one data struct:

```go
type AFKData struct {
    Message string  // set by manual `afk <message>`; empty for auto-AFK
    Manual  bool    // distinguishes manual vs timeout
}
```

All other states are stateless. Transition reasons carry the trigger (`"manual_afk"`, `"timeout_idle"`, `"player_entry"`, `"attacked"`, `"tcp_closed"`, etc.) and any per-event metadata via `TransitionReason.Metadata`.

---

## 4. Integration with Other Machines

### 4.1 Combat Phase veto

Block list: **Disconnected**, **Despawning**.

Connecting players have no room ID — structurally untargetable, no veto needed. Idle / AFK / Dormant players accept incoming combat. (For Dormant mobs see §4.2.)

```go
character.CombatPhase.RegisterVeto(
    combatphase.Idle, combatphase.Engaging,
    func(reason TransitionReason) error {
        switch character.Presence.State() {
        case Disconnected, Despawning:
            return ErrVetoed
        }
        return nil
    })
```

The veto fires on the DEFENDER's CombatPhase — the attacker's `Idle→Engaging` is not gated by their own Presence (already Active by virtue of having issued the attack command).

### 4.2 Auto-wake for Dormant mobs

Single transition fired from the lowest-common-ancestor target-resolution path (likely `actions.ResolveTargetActor` or its caller). When a Dormant target is resolved and an attack lands:

```go
if target.Presence.State() == presence.Dormant {
    target.Presence.TransitionTo(presence.Active,
        TransitionReason{Trigger: "attacked", Actor: attackerRef})
}
// Combat continues normally
```

The mob's per-round tick was being skipped; receivability was intact. The wake fires BEFORE damage resolves so the target is in Active when the round's logic runs.

### 4.3 Scheduled-transition cancellation

The framework's `RoundScheduler` is keyed by character. Presence registers an observer on terminal-state entry:

```go
character.Presence.RegisterObserver(func(evt Event) {
    if evt.To == presence.Disconnected || evt.To == presence.Despawning {
        scheduler.CancelAllFor(character)
    }
})
```

This wipes pending scheduled transitions across ALL of the character's machines (Activity casting timers, Position recovery timers, drift rolls, etc.).

---

## 5. Round-Driver Hook

One new hook: `NewRound_PresenceTick`. Position in the chain:

```
NewRound_IsBoss
NewRound_DoCombat           ← attacks resolve here
NewRound_PresenceTick       ← NEW: timeout-driven transitions
NewRound_IdleMobs           ← Despawning mobs get terminal-tick removal here
NewRound_Mutations
...
```

The PresenceTick hook walks active characters:

- For each player: read `roundNow - lastInputRound`, compare to `PresenceIdleAfterRounds` / `PresenceAFKAfterRounds` / `PresenceDisconnectAfterRounds`, fire the relevant transition through the machine.
- For each mob: read the equivalent "rounds since last meaningful event" counter (folded from BoredomCounter), compare to `PresenceMobDormantAfterRounds` / `PresenceMobDespawnAfterRounds`, fire transition.

The hook is short — the framework handles vetoes and observers automatically. Idle-mobs cleanup in the next hook (`NewRound_IdleMobs`) sees Despawning mobs and runs the terminal-tick removal.

Ordering rationale:
- After `NewRound_DoCombat`: an attack that landed this round transitions Dormant→Active in the wake-on-attack path BEFORE PresenceTick runs, so the now-Active mob isn't immediately bounced back to Dormant by the timeout check.
- Before `NewRound_IdleMobs`: mobs that PresenceTick transitions to Despawning get their terminal-tick removal cleanly in the same round.

---

## 6. Config Knobs

New knobs under `Server`:

| Knob | Default | Notes |
|---|---|---|
| `PresenceIdleAfterRounds` | 8 (~30s) | Active → Idle |
| `PresenceAFKAfterRounds` | 75 (~5min) | Idle → AFK |
| `PresenceDisconnectAfterRounds` | 900 (~1hr) | AFK → Disconnected |
| `PresenceMobDormantAfterRounds` | (= current `MaxMobBoredom`, candidate 30) | Active → Dormant |
| `PresenceMobDespawnAfterRounds` | 60 | Dormant → Despawning |

Existing `MaxMobBoredom` is removed and replaced by `PresenceMobDormantAfterRounds`. The despawn threshold gets a new knob since Dormant is a new intermediate tier the old single-counter design didn't have.

---

## 7. Behavior Matrix (Intent-Driven Tests)

Every row becomes a RED-phase test before implementation. PR = Presence.

| ID | Actor | From | Trigger | Conditions | To | Notes |
|---|---|---|---|---|---|---|
| PR-001 | Player | (init) | login complete | — | Connecting | Constructor |
| PR-002 | Player | Connecting | entered room | — | Active | Auto |
| PR-003 | Player | Active | command sent | — | Active | No-op, keeps lastInput fresh |
| PR-004 | Player | Active | round tick | `roundNow - lastInput >= IdleThreshold` | Idle | Auto |
| PR-005 | Player | Idle | command sent | — | Active | Auto |
| PR-006 | Player | Idle | round tick | `roundNow - lastInput >= AFKThreshold` | AFK | Auto |
| PR-007 | Player | Active | `afk <message>` cmd | — | AFK | AFKData.Manual=true |
| PR-008 | Player | AFK | command sent (non-`afk`) | — | Active | Clears AFKData |
| PR-009 | Player | AFK | round tick | `roundNow - lastInput >= DisconnectThreshold` | Disconnected | Auto |
| PR-010 | Player | (any) | TCP closed | — | Disconnected | Connection observer |
| PR-011 | Player | Disconnected | (any) | — | (no transition) | Terminal until reconnect |
| PR-020 | Mob | (init) | NewMobByIdFresh | — | Spawning | Constructor |
| PR-021 | Mob | Spawning | next round tick | — | Active | Auto |
| PR-022 | Mob | Active | round tick | bored >= threshold && !essential | Dormant | Veto on essential |
| PR-023 | Mob | Active | round tick | zone has no players nearby && !essential | Dormant | Veto on essential |
| PR-024 | Mob | Active | round tick | essential | (no transition) | Essential veto |
| PR-025 | Mob | Dormant | player enters room | — | Active | Player-entry observer |
| PR-026 | Mob | Dormant | attacked | — | Active | Wake-on-attack |
| PR-027 | Mob | Dormant | round tick | dormant too long && !essential | Despawning | Veto on essential |
| PR-028 | Mob | Despawning | next round tick | — | (removed) | Terminal cascade |
| PR-030 | Either | (any) | CombatPhase.Idle→Engaging | Presence is Disconnected/Despawning | VETO | Block list |
| PR-031 | Either | (any) | scheduled transition pending | Presence→Disconnected/Despawning | CancelAllFor(char) | Scheduler observer |

---

## 8. New Files

| Path | Responsibility |
|---|---|
| `internal/state/presence/presence.go` | State enum, transition tables (one per actor), constructor functions `NewPlayerPresence()` + `NewMobPresence()` |
| `internal/state/presence/afk_data.go` | `AFKData` struct |
| `internal/state/presence/triggers.go` | Trigger string constants (`TriggerInputReceived`, `TriggerManualAFK`, `TriggerTCPClosed`, `TriggerTimeoutIdle`, `TriggerTimeoutAFK`, `TriggerTimeoutDisconnect`, `TriggerPlayerEntry`, `TriggerAttacked`, `TriggerBored`, `TriggerZoneEmpty`, `TriggerDormantTooLong`) |
| `internal/state/presence/presence_test.go` | Behavior Matrix unit tests, one row per matrix ID |
| `internal/state/presence/integration_test.go` | Veto + observer integration with CombatPhase and the scheduler |
| `internal/state/presence/context.md` | Package documentation |
| `internal/hooks/NewRound_PresenceTick.go` | Round-driver hook (player + mob timeout transitions) |

## 9. Modified Files

| Path | Change |
|---|---|
| `internal/characters/character.go` | Add `Presence *presence.Machine` field; initialization in NewCharacter (or actor-specific constructors) |
| `internal/users/userrecord.go` | Drop `ManualAFK`, `AFKMessage` fields. `OnlineInfo.IsAFK` compat shim reads `Presence.State() == AFK`. Drop ad-hoc `isAfk` computation. |
| `internal/users/users.go` | TCP-close observer fires `Presence.TransitionTo(Disconnected)`. Login-complete fires `Connecting`; entry-to-room fires `Active`. |
| `internal/usercommands/afk.go` | Rewrite: cmd transitions to AFK with AFKData; second invocation transitions back to Active. |
| `internal/usercommands/usercommands.go` | Drop the `ManualAFK` clear-on-next-cmd shim (subsumed by Presence). Any command transitions to Active. |
| `internal/usercommands/online.go` | Continue reading `onlineInfo.IsAFK`. The shim in `userrecord.go` populates that bool from `Presence.State() == AFK`. UI behavior unchanged. |
| `internal/rooms/roomdetails.go` | Read manual-AFK + message from `AFKData` via the Presence machine. |
| `internal/mobs/mobs.go` | Drop `BoredomCounter`, `PreventIdle`. Add `NewMobPresence()` to the mob constructor flow. |
| `internal/hooks/NewRound_IdleMobs.go` | Despawn path: read `Presence.State() == Despawning` and run the terminal-tick removal. No more `mob.BoredomCounter` access. |
| `internal/hooks/MobIdle_HandleIdleMobs.go` | Drop `BoredomCounter` reset patterns; rely on Presence transitions. `IsCaravanServedZone()` check moves into the Active→Dormant veto. |
| `internal/mobcommands/lookfortrouble.go` | Drop `BoredomCounter` increment. Instead, the mob's PresenceTick reads "rounds since last target_found event" via a per-mob counter that doesn't bleed through the rest of the codebase. |
| `internal/rooms/rooms.go:2144` | Drop `mob.BoredomCounter = 0` reset; rely on Presence's player-entry observer. |
| `internal/state/control/context.md`, `internal/state/position/context.md` (if cross-link useful) | Note that CombatPhase gating now consults Presence. |
| `internal/state/combatphase/context.md` | Document the Presence veto on Idle→Engaging. |
| `internal/configs/config.server.go` | Add the five new threshold knobs. Drop `MaxMobBoredom`. |

---

## 10. Sunset List (Chunk-End)

**Player-side fields:**
- `UserRecord.ManualAFK bool`
- `UserRecord.AFKMessage string`
- Ad-hoc `isAfk := u.ManualAFK || (roundNow - u.GetLastInputRound() >= afkRounds)` in `userrecord.go:543`

**Mob-side fields:**
- `Mob.BoredomCounter uint8`
- `Mob.PreventIdle bool`

**Config:**
- `Memory.MaxMobBoredom`

**Functions:**
- The `mob.Command("despawn ...")` call from `NewRound_IdleMobs.go:47` — replaced by the Presence machine's terminal-tick handler on Despawning.

**Preserved:**
- `Mob.WanderCount` / `MaxWander` — orthogonal wander-budget concern.
- `Mob.IsEssential()` — now the veto policy hook.
- `Mob.Despawns()` — kept as the existing predicate; the veto handler calls it.
- `IsCaravanServedZone()` — wrapped in the veto.
- `UserRecord.lastInputRound` — still the source of truth for input timing.
- `OnlineInfo.IsAFK` — compat shim, computed from Presence.

---

## 11. Out of Scope (Explicit)

- **`Mob.WanderCount` / `MaxWander` migration.** Orthogonal wander-budget concern, stays.
- **Helpfile rewrites for `afk`.** Single-line touchup if it's stale; no broader sweep.
- **Idle-mob behavior templates** (wandering messages, etc.).
- **`mob_spawned` BTree cascade re-routing.** Existing on_spawn handlers stay at their call sites. Spawning is a one-tick observer-firing beat in the framework; no new BTree integration.
- **Per-player AFK thresholds.** Single set of config knobs for all players in v1.
- **Charmed-companion auto-Dormant when owner Disconnects.** Companions stay Active even when their owner is gone. Future polish if needed.
- **Idle-aware faction / quest hooks.** Some quest engines / faction observers might want to react to AFK / Disconnected; not in v1.

---

## 12. Risks / Open Questions

- **`NewRound_PresenceTick` hook ordering.** Must be after `DoCombat` (so attacks-this-round transition Dormant→Active before PresenceTick re-evaluates timeouts) and before `IdleMobs` (so Despawning mobs get terminal-tick removal in the same round). Existing `NewRound_*.go` files have a clear convention.

- **`OnlineInfo.IsAFK` semantics.** Today the UI shows "(afk)" for both manual-AFK and timeout-AFK. With Idle as a new state, do we show "(idle)" too? Probably NO in v1 — keep `OnlineInfo.IsAFK = (Presence.State() == AFK)` and let Idle be invisible. Future polish.

- **The wake-on-attack site.** Multiple attack paths (player→mob, mob→player, mob→mob). Need a single lowest-common-ancestor for the Dormant→Active transition. Likely `actions.ResolveTargetActor` or its immediate caller. Confirmed during writing-plans.

- **Connecting state coverage.** Need to find the right spot in `users/users.go` to fire `Connecting → Active`. Probably the `loginCharacterTo()` or `MoveToRoom` call that places the character in their starting room. Confirmed during writing-plans.

- **Mob "bored" counter equivalent.** BoredomCounter is being sunset. The PresenceTick still needs SOME signal of "rounds since this mob found a target". Options: (a) keep a per-mob `lastTargetFoundRound uint64` and compute on the fly (analogous to player `lastInputRound`); (b) add a `RoundsSinceActive int` field reset by every relevant event. Probably (a) — symmetric to the player side.

- **Backwards compat for in-progress play.** If a server upgrades mid-session, mobs already in flight have no `Presence` machine. Initialization at `NewMobByIdFresh` covers new spawns; existing instances at load time need a one-time `Presence = NewMobPresence()` injection. Player records on load have the same requirement.

---

## 13. Success Criteria

1. AFK player still takes hits in a dangerous room (intentional, not a regression).
2. Manual `afk <message>` command transitions to AFK with the message stored on the machine. `online` list shows the player as AFK.
3. Mob spawned via `mob spawn 105` enters Spawning → Active on next tick. Existing on_spawn handlers fire.
4. Mob in a room with no players for `PresenceMobDormantAfterRounds` rounds enters Dormant. A player attacking that mob auto-wakes it to Active before damage resolves.
5. Shopkeeper, forager, caravan crew, charmed companion: NEVER transition out of Active. Veto fires correctly. Verified by 4 unit tests + one smoke spot-check.
6. Despawning mob disappears on the next round-driver tick. Pending scheduled transitions for that mob cancel via the scheduler observer.
7. Disconnected player: scheduled transitions cancel; if reconnected, they enter Connecting → Active. AFK state does NOT persist across login.
8. Behavior Matrix: every PR-NNN row has a test; all tests PASS.
9. Sunset deletions all land: `ManualAFK`, `AFKMessage`, `BoredomCounter`, `PreventIdle`, `MaxMobBoredom`. `go build ./...` clean.
10. AI smoke: no panics, no missing-template debug strings, observed behavior matches the Behavior Matrix.

---

## 14. Implementation Order (preview for writing-plans)

1. New `internal/state/presence/` package + Behavior Matrix unit tests (RED).
2. Implement the package (GREEN).
3. Wire `Character.Presence` field + per-actor constructors.
4. `NewRound_PresenceTick` hook + config knobs.
5. Veto on `Active→Dormant`/`Active→Despawning` (essential gate).
6. Veto on `CombatPhase.Idle→Engaging` (block list).
7. Auto-wake on attack in target-resolution path.
8. Scheduler observer for terminal-state cancellation.
9. Connection lifecycle hooks (login → Connecting → Active; TCP close → Disconnected).
10. Rewrite `afk` command + drop `ManualAFK`/`AFKMessage`.
11. Cutover existing call sites (NewRound_IdleMobs, MobIdle_HandleIdleMobs, lookfortrouble, rooms.go:2144, online.go, roomdetails.go).
12. Drop sunset fields and config knob.
13. Context.md sweep.
14. AI smoke pass.
15. Roadmap + patch notes close-out.
