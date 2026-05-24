# Combat State Machines — Chunk 2: Life

> **Third chunk of the combat-state-machines redesign**
> (master spec: `2026-05-13-combat-state-machines-design.md`).
> Builds the third machine — `Alive | Dead | Respawning` — on
> the chunk-0 framework. Replaces the scattered death-cleanup
> logic in `suicide.go` (player) and `MobDeath_*` hooks (mob)
> with cascade-driven cross-machine cleanup. Rips out unused
> permadeath + extra-lives systems while we're touching the
> code. Adds auto-look-after-respawn for cleaner UX.

## Goal

Today, when a character dies, the cleanup logic is split across
`internal/usercommands/suicide.go` (~250 lines for player death),
several `internal/hooks/MobDeath_*.go` files for mob-specific
side effects, and inline death-handling in
`internal/characters/resources.go` (health-drops-to-zero path).
Buff cancellation, aggro clearing, state-machine resets, loot
drops, teleports, grace buffs, stat decay, kill stat tracking,
PvP/PvE counters, party notifications, and aliveness substrate
events all fire from these scattered sites.

This chunk:

- **Builds the Life state machine** (`Alive | Dead | Respawning`)
  with explicit transitions, framework-driven cascade handlers,
  and observer subscriptions for the per-actor cleanup effects.
- **Consolidates cross-machine cleanup into one cascade.** Combat
  Phase → Idle, Awareness → Visible, Activity → Free (chunk 3
  pre-wire), Position → Standing (chunk 4 pre-wire), buffs
  canceled, conditions cleared — all fire from the Life
  Alive → Dead cascade, not from inline death code.
- **Routes scattered death effects through observers.** Loot
  drop, graveyard teleport, grace buff, stat decay, PvP/PvE
  tracking, party notifications, MobDeath event firing — each
  becomes a small file with a single responsibility
  subscribing to the appropriate Life transition.
- **Rips out permadeath + extra lives.** Dead content the user
  never intends to invoke (inherited from upstream). Mechanic
  simplification falls out for free.
- **Adds auto-look after respawn teleport.** Players who die
  and respawn at the graveyard see the room description
  automatically. Same fix lands as a followup for fold-recall
  and future teleports (logged as
  `project_auto_look_after_room_change.md`).
- **Sunsets the existing `suicide.go` cleanup logic.** File
  shrinks from ~250 lines to ~30 lines (just command parsing
  and the Life transition call). All cleanup moves to cascade
  handlers and observers.

## Non-goals

- **Combat math.** Damage formulas, mitigation curves, kill-blow
  attribution math — all unchanged.
- **Respawn location logic.** Graveyard determination, fallback
  rooms, instance-room respawn semantics — preserved as-is.
- **Mob respawn timer.** New instances spawn from the room's
  spawn point on its existing schedule; Life machine doesn't
  participate in that lifecycle.
- **Player UX during the brief Respawning state.** No "death
  screen" or wait-pause; respawn remains same-tick (Alive → Dead
  → Respawning → Alive within one tick of player input).
- **ReviveOnDeath buff.** This is a separate one-shot resurrection
  mechanic (spell/item) and stays.
- **Stat/skill decay system.** Confirmed during brainstorm as a
  normal-death cleanup mechanic (NOT tied to permadeath); stays
  active, just invoked from a Death observer rather than inline.

## Architectural musts

Confirmed during brainstorm:

1. **Unified state list for players and mobs.** `Alive | Dead |
   Respawning`. Mobs follow Alive → Dead → (instance destroyed,
   machine deallocates with it). Players follow Alive → Dead →
   Respawning → Alive within one tick. Same machine, same
   transitions; mob's never-reached Respawning state is a no-op
   slot.

2. **Life owns state + cross-machine cascades; observers own
   game effects.** Chunks 0/1 precedent. Combat Phase / Awareness
   / Activity / Position forced to terminal states by the Life
   cascade. Concrete effects (loot drop, teleport, decay) live
   in observer files subscribing to Life transitions.

3. **Permadeath + extra lives ripped out entirely.** No
   `Character.ExtraLives`, no permadeath path in `suicide.go`,
   no permadeath-specific cleanup, no character lockout. Mob
   despawn + respawn handled by existing spawn mechanism
   (unaffected).

4. **Stat/skill decay stays.** Confirmed as normal-death system;
   invoked from `Death_PlayerCleanup.go` observer.

5. **Auto-look after respawn teleport.** New observer
   `Respawn_PlayerAutoLook.go` fires a `look` command on
   the player's connection on `Respawning → Alive`. Parallel
   fold-recall fix logged as followup memory.

6. **Hard cutover within chunk.** Move suicide.go cleanups to
   cascade observers; delete the inline cleanup blocks; verify
   the migration is complete before chunk close.

7. **Players and mobs share the same Life machine type.** No
   per-actor polymorphism on the state list. Per-actor effects
   are observer-side (player observers ignore mob transitions
   and vice versa, gated by `actor.IsPlayer()` checks or by
   subscribing only to the relevant character types).

8. **Death triggered from any health-to-zero site routes
   through `c.Life.TransitionToDead(reason)`.** Damage
   application (`ApplyHealthChange`), suicide command, admin
   kill — all call the same entry point. Cascade fires
   regardless of trigger. `TransitionReason.Trigger` carries
   the cause for observers to branch on (kill-credit handling,
   special suicide messaging, etc.).

9. **Per-state data slot for DeadData carries killer + damage
   map.** Observers reading DeadData can compute kill credit,
   faction rep, party-share, etc. without re-discovering the
   damage attribution.

10. **Persistence: Life state lazily reverts to Alive on
    boot.** Long-lived player records that were in Dead or
    Respawning at server shutdown boot to Alive at full health.
    Matches existing engine behavior (no die-and-stay-dead
    across restarts). Chunk-0 Combat Phase boots Idle; chunk-1
    Awareness boots Visible — Life follows the same "fresh
    state on boot" pattern.

## Life machine design

### States

```go
package life

type State int

const (
    Alive State = iota
    Dead
    Respawning
)

func (s State) String() string {
    switch s {
    case Alive:      return "Alive"
    case Dead:       return "Dead"
    case Respawning: return "Respawning"
    }
    return "Unknown"
}
```

### Per-state data

```go
// AliveData is empty — default state.
type AliveData struct{}

// DeadData carries killer + damage attribution.
// Observers consume this for kill credit, faction rep,
// party-share, kill stats, etc.
type DeadData struct {
    Reason     state.TransitionReason
    Killer     state.ActorRef           // who landed the final blow
    DamageMap  map[int]int              // userId → damage; for kill-credit and party-share
}

// RespawningData captures the in-flight respawn cycle.
// Player-only (mobs don't reach Respawning).
type RespawningData struct {
    Reason     state.TransitionReason
    DestRoomId int                       // graveyard or home room id
}
```

### Transition table

```go
var validTransitions = state.TransitionTable[State]{
    Alive:      {Dead},
    Dead:       {Respawning, Alive},     // Alive direct for cleanup-restart edge case
    Respawning: {Alive},
}
```

(Dead → Alive is allowed for edge cases like admin restoration or hypothetical revive-from-Dead spell. Standard player respawn flow goes Dead → Respawning → Alive.)

### Trigger reasons

```go
const (
    TriggerHealthZero    = "health_zero"      // damage drove health to 0
    TriggerSuicide       = "suicide_command"  // player explicit
    TriggerAdminKill     = "admin_kill"       // admin command
    TriggerCleanupReady  = "cleanup_ready"    // Dead → Respawning auto-advance
    TriggerRespawnReady  = "respawn_ready"    // Respawning → Alive auto-advance
    TriggerForceAlive    = "force_alive"      // edge-case admin restore
)
```

## Behavior Matrix

Each row drives a RED-phase test. ID prefix `LI-` = Life.

### Alive → Dead triggers

| ID | Start | Trigger | Conditions | Outcome | Rationale |
|----|-------|---------|------------|---------|-----------|
| LI-001 | Alive | Health drops to 0 (damage application) | none | → Dead; DeadData populated with Killer + DamageMap from damage context | Primary death path: combat damage drives health to zero. |
| LI-002 | Alive | `suicide` command | none | → Dead with Trigger=TriggerSuicide | Explicit player-initiated death; no killer attribution. |
| LI-003 | Alive | Admin `kill` command targeting this character | admin auth confirmed | → Dead with Trigger=TriggerAdminKill, Killer=admin | Admin override; bypasses the normal damage path. |

### Player respawn cycle

| ID | Start | Trigger | Conditions | Outcome | Rationale |
|----|-------|---------|------------|---------|-----------|
| LI-004 | Dead (player) | cleanup cascade complete | player actor | → Respawning same-tick; DestRoomId set | After cross-machine + per-actor cleanup completes, advance to respawn. |
| LI-005 | Respawning (player) | teleport + grace buff applied | player actor | → Alive same-tick; auto-look fires | Respawn cycle completes within one tick from player perspective. |

### Mob death lifecycle

| ID | Start | Trigger | Conditions | Outcome | Rationale |
|----|-------|---------|------------|---------|-----------|
| LI-006 | Alive (mob) | health drops to 0 | mob actor | → Dead; observers fire loot drop + MobDeath event + corpse setup; instance scheduled for cleanup; machine never reaches Respawning | Mob instances are ephemeral; the respawn machinery lives on the spawn point, not the instance. |

### Cross-machine cascade on Alive → Dead

| ID | Start | Trigger | Conditions | Outcome | Rationale |
|----|-------|---------|------------|---------|-----------|
| LI-007 | Alive | → Dead transition | (cascade) | Combat Phase forced to Idle (ForceIdle); Attackers list cleared bidirectionally | Death ends combat; both outbound and inbound aggro cleared. |
| LI-008 | Alive | → Dead transition | (cascade) | Awareness forced to Visible (ForceVisible) | Dead characters don't stay hidden; room broadcast fires via the Awareness cascade. |
| LI-009 | Alive | → Dead transition | (cascade) | Activity → Free: CastingState nil, CraftingState nil (chunk 3 pre-wire) | Casting and crafting end on death. Chunk 3 will repoint to real Activity machine. |
| LI-010 | Alive | → Dead transition | (cascade) | Position → Standing: CombatPosition reset; GrappleControllerId cleared (chunk 4 pre-wire) | Dead characters not prone or grappled. |
| LI-011 | Alive | → Dead transition | (cascade) | All non-permanent buffs canceled via CancelBuffsWithFlag(buffs.All) | Death wipes status effects. Permabuffs preserved. |
| LI-012 | Alive | → Dead transition | (cascade) | Conditions slice cleared | Combat conditions (recovery penalty, shield, etc.) reset. |

### Dead → Respawning cascade (player only)

| ID | Start | Trigger | Conditions | Outcome | Rationale |
|----|-------|---------|------------|---------|-----------|
| LI-013 | Dead → Respawning | (cascade) | player actor | Health/Stamina/Conviction reset to 5% of max | Preserves existing behavior; players respawn weak. |
| LI-014 | Dead → Respawning | (cascade) | player actor | Grace buff NoAggroTarget (#81) applied for ~3 rounds | Players get a brief protection window after respawn. |
| LI-015 | Dead → Respawning | (cascade) | player actor | Teleport to graveyard / home room; DestRoomId set in RespawningData | Player moves to respawn location. |

### Respawning → Alive cascade

| ID | Start | Trigger | Conditions | Outcome | Rationale |
|----|-------|---------|------------|---------|-----------|
| LI-016 | Respawning → Alive | (cascade) | player actor | `look` command fired on player connection so room description renders | UX fix — players don't have to type `look` after teleport. |

### Killer attribution + damage map

| ID | Start | Trigger | Conditions | Outcome | Rationale |
|----|-------|---------|------------|---------|-----------|
| LI-017 | Alive → Dead | health-zero trigger | killer ref available from damage context | DeadData.Killer = killer | Last-blow attribution preserved for observers. |
| LI-018 | Alive → Dead | health-zero trigger | character has PlayerDamage map | DeadData.DamageMap = copy of PlayerDamage | Damage attribution preserved for party-share and faction rep observers. |
| LI-019 | Dead | observers read DeadData | (cascade) | Observers consume DeadData; Killer + DamageMap drive kill credit, faction rep, party notifications | One read source of truth post-Death; no re-discovery. |

### Mob-specific death observers

| ID | Start | Trigger | Conditions | Outcome | Rationale |
|----|-------|---------|------------|---------|-----------|
| LI-020 | Alive (mob) → Dead | (observer fires) | mob actor | Loot drop per ItemDropChance; gold drops to room; corpse name/desc set | Existing mob-death behavior preserved. |
| LI-021 | Alive (mob) → Dead | (observer fires) | mob actor | events.MobDeath fires; existing aliveness subscribers (faction rep, opinion, crime, knowledge, bounty) react | Aliveness substrate unchanged; just rerouted through the Life observer. |
| LI-022 | Alive (mob) → Dead | (observer fires) | mob actor | Instance scheduled for cleanup (existing despawn timer) | Mob instance lifecycle preserved. |

### Player-specific death observers

| ID | Start | Trigger | Conditions | Outcome | Rationale |
|----|-------|---------|------------|---------|-----------|
| LI-023 | Alive (player) → Dead | (observer fires) | player actor | Stat decay + skill rust applied (existing system, preserved per user direction) | Normal-death cleanup, NOT permadeath-only. |
| LI-024 | Alive (player) → Dead | (observer fires) | player actor | KD stats updated: deaths++, PvP/PvE attribution per Killer.IsPlayer() | Kill/death counters preserved. |
| LI-025 | Alive (player) → Dead | (observer fires) | player has PartyId | Party members notified via the existing channel | Existing party-notification behavior preserved. |

### Persistence + fresh-machine state

| ID | Start | Trigger | Conditions | Outcome | Rationale |
|----|-------|---------|------------|---------|-----------|
| LI-026 | (n/a) | NewMachine() | (construction) | State() == Alive | Sensible default at construction. |
| LI-027 | (any) | Server restart | (boot) | All characters boot Alive (Life state does not persist) | Matches chunk-0/1 pattern; in-flight Dead/Respawning chars resolve to Alive on restart. No die-and-stay-dead across reboots. |

## Cleanup cascade ownership

**Life machine OWNS (cascade handlers):**
- Combat Phase ForceIdle on Dead entry
- Awareness ForceVisible on Dead entry
- Activity → Free (CastingState/CraftingState nil) on Dead entry — chunk 3 pre-wire
- Position → Standing on Dead entry — chunk 4 pre-wire
- Buffs CancelBuffsWithFlag(buffs.All) on Dead entry
- Conditions slice clear on Dead entry
- Resources reset to 5% of max on Dead → Respawning entry (player only)
- Grace buff (NoAggroTarget #81) on Dead → Respawning entry (player only)

**Observer files (separate files, one per concern):**

| File | Subscribes to | Responsibility |
|------|---------------|----------------|
| `internal/hooks/Death_PlayerCleanup.go` | Player Alive → Dead | Stat decay + skill rust; KD stat tracking (PvP/PvE); party notifications |
| `internal/hooks/Death_MobLoot.go` | Mob Alive → Dead | Loot drop per ItemDropChance; corpse name/description; gold drop to room |
| `internal/hooks/Death_AlivenessSubstrate.go` | Mob Alive → Dead | Fire events.MobDeath (existing faction/opinion/crime/knowledge/bounty subscribers fire downstream) |
| `internal/hooks/Death_MobInstanceCleanup.go` | Mob Alive → Dead | Schedule instance for despawn (existing cleanup machinery) |
| `internal/hooks/Respawn_PlayerTeleport.go` | Player Dead → Respawning | Graveyard / home-room teleport; DestRoomId set in RespawningData |
| `internal/hooks/Respawn_PlayerAutoLook.go` | Player Respawning → Alive | Fire `look` command on player connection |

**Order matters within cascade.** Cascade handlers fire in registration order. Required ordering:
1. Cross-machine cleanups (Combat Phase, Awareness, Activity, Position, buffs, conditions)
2. Per-actor cleanups (stat decay, loot drop, aliveness events)
3. State transitions Dead → Respawning, fires resource reset + grace buff + teleport
4. State transitions Respawning → Alive, fires auto-look

Init() order in hook files enforces this. Document the dependency clearly so future contributors don't reorder.

## Migration approach

### `internal/usercommands/suicide.go`

**Before:** ~250 lines including:
- Permadeath path (extra-lives check, character lockout)
- Buff cancellation
- Conditions clear
- Aggro/EndAggro
- CastingState nil
- Health/Stamina/Conviction reset
- Stat/skill decay
- Graveyard teleport
- Grace buff
- KD stat tracking
- Party notifications

**After:** ~30 lines:
- Command parse (rest, user, room, flags)
- `user.Character.Life.TransitionToDead(state.TransitionReason{Trigger: life.TriggerSuicide, Actor: state.ActorRef{UserId: user.UserId}})`
- Return

The cascade + observers do everything else.

### `internal/characters/resources.go` `ApplyHealthChange`

Today: if health <= 0, calls into the death sequence inline. Migration:

```go
if c.Health < 1 {
    c.Life.TransitionToDead(state.TransitionReason{
        Trigger: life.TriggerHealthZero,
        Actor:   killerRef,
        Metadata: map[string]any{
            "damage_map": c.PlayerDamage,
        },
    })
}
```

Killer ref + damage map propagate through the transition; the Life machine populates DeadData from them. Observers consume.

### Mob death paths

Damage-application sites in `internal/combat/` that drive mob health to zero get the same migration:

```go
mob.Character.Life.TransitionToDead(state.TransitionReason{
    Trigger: life.TriggerHealthZero,
    Actor:   killerRef,
    Metadata: map[string]any{
        "damage_map": mob.PlayerDamage,
    },
})
```

The `events.MobDeath` event continues to fire — but from
`Death_AlivenessSubstrate.go` observer on the Life Dead
transition, NOT from inline code in the damage path.

### File structure

| File | Status | Responsibility |
|------|--------|----------------|
| `internal/state/life/life.go` | NEW | State enum, data types, Machine wrapper |
| `internal/state/life/transitions.go` | NEW | Valid-transition table, trigger constants |
| `internal/state/life/rules.go` | NEW | Transition method implementations |
| `internal/state/life/life_test.go` | NEW | Behavior Matrix tests (LI-001 through LI-027) |
| `internal/state/life/context.md` | NEW | Package documentation |
| `internal/characters/character.go` | MODIFY | Add `Life *life.Machine` field; `IsAlive()` / `IsDead()` predicates; init in `New()` |
| `internal/characters/validate.go` | MODIFY | Nil-guard init of `Life` for YAML-loaded characters |
| `internal/hooks/Life_Cascades.go` | NEW | Cross-machine cascade wiring (Combat, Awareness, Activity, Position, buffs, conditions) |
| `internal/hooks/Death_PlayerCleanup.go` | NEW | Stat decay, KD tracking, party notifications observer |
| `internal/hooks/Death_MobLoot.go` | NEW | Loot drop, corpse setup observer |
| `internal/hooks/Death_AlivenessSubstrate.go` | NEW | events.MobDeath firing observer |
| `internal/hooks/Death_MobInstanceCleanup.go` | NEW | Instance cleanup-scheduling observer |
| `internal/hooks/Respawn_PlayerTeleport.go` | NEW | Graveyard teleport observer |
| `internal/hooks/Respawn_PlayerAutoLook.go` | NEW | Auto-look after respawn observer |
| `internal/usercommands/suicide.go` | MODIFY | Thin command handler; calls Life.TransitionToDead |
| `internal/mobcommands/suicide.go` | MODIFY | Thin command handler (if exists) |
| `internal/characters/resources.go` | MODIFY | `ApplyHealthChange` calls Life.TransitionToDead instead of inline death-sequence |
| `internal/combat/` damage-application sites | MODIFY | Route mob deaths through Life.TransitionToDead |
| `internal/characters/character.go` | MODIFY | Delete `Character.ExtraLives` field (and any related fields) |
| Permadeath-specific YAML | DELETE | Audit + delete if any |
| `internal/characters/context.md` | MODIFY | Document Life field, IsAlive/IsDead predicates |
| `internal/hooks/context.md` | MODIFY | Document new Life cascade + Death/Respawn observer files |
| `COMBAT_STATE_ROADMAP.md` | MODIFY | Mark chunk 2 Done |

## Smoke scenarios

In-game validation. Each maps to one or more Behavior Matrix rows.

1. **Player suicide command (LI-002, LI-013, LI-015, LI-016).**
   Run `suicide`. Verify: death message → graveyard room
   description via auto-look → resources at 5% → grace buff
   applied.

2. **Player dies in combat (LI-001, LI-017, LI-018).** Engage
   a stronger mob; die. Verify: killer attribution captured;
   normal respawn flow; stat decay observable.

3. **Mob dies (LI-006, LI-020, LI-021).** Player kills a mob.
   Verify: loot drops; corpse appears; faction rep / opinion
   bumps fire (chunk-1.x observers still work).

4. **Mid-cast death (LI-009).** Player starts a multi-round cast,
   takes lethal damage mid-cast. Verify: CastingState cleared on
   death; on respawn, casting state is nil.

5. **Hidden death (LI-008).** Hidden player takes lethal damage.
   Verify: Awareness ForceVisible cascade fires; room broadcast
   "emerges from shadows" appears just before death message.

6. **Grappled death (LI-010).** Player grappled by mob takes
   lethal damage. Verify: GrappleControllerId cleared; position
   reset; no orphan grapple state on respawn.

7. **Stat decay still applies (LI-023).** Track skill levels
   pre-death; die normally; verify rust applied on respawn.

8. **Multi-killer kill credit (LI-018).** Two players damage a
   mob to death. Verify DamageMap has both attackers; party-share
   / faction rep observers see both.

9. **Permadeath path removed.** Attempt to invoke permadeath via
   any admin command or YAML setup. Confirm: no such path exists;
   character cannot be permanently removed via this mechanism.

10. **Chunk 0/1 regression.** Run the chunk 0 thief smoke and
    chunk 1 awareness smoke after chunk 2 lands. Both should
    still pass — no Combat Phase or Awareness regression.

## Sunset list

- `Character.ExtraLives` field (and any related lives-counter
  YAML deserialization)
- Permadeath path in `suicide.go` (~50 lines including
  extra-lives check, lockout, permanent character removal)
- Permadeath-specific cleanup code paths (audit and delete)
- Any permadeath YAML configuration (audit and delete; likely
  none, but verify)
- Inline death-cleanup blocks in `suicide.go` (buff cancel,
  aggro clear, casting nil, conditions clear) — moved to Life
  cascade
- Inline death-cleanup in mob death paths in `internal/combat/`
  — moved to Life cascade
- The "permadeath path" branches in stat decay (decay applies
  uniformly on normal death now; permadeath-specific harder
  decay disappears)

Preserved (NOT sunset):
- `applyStatDecay` / `applySkillRust` functions — invoked from
  `Death_PlayerCleanup.go`
- `ReviveOnDeath` buff — separate one-shot resurrection mechanic
- Mob spawn timer / respawn mechanism — unaffected
- Grace buff #81 (NoAggroTarget) — applied from Life Dead →
  Respawning cascade

## Risks / known limitations

- **Death from damage application is the hottest path.** Every
  combat round can fire health-to-zero checks. Routing through
  Life.TransitionToDead adds a few function calls + cascade
  handler invocations. Profile during smoke; current chunk-0/1
  benchmarks show no regression from similar cascade work, but
  damage application runs more frequently.
- **Cascade handler ordering is enforced by registration order.**
  If a future contributor reorders the init() calls, the
  cleanup order changes. Document clearly in
  `internal/hooks/context.md`; consider adding a comment in
  each init() linking to the order requirement.
- **Player-vs-mob observers double-fire risk.** The
  `Death_PlayerCleanup.go` and `Death_MobLoot.go` observers
  both subscribe to Life Dead transitions. Each must gate on
  `actor.IsPlayer()` / `actor.IsMob()` to avoid firing on the
  wrong actor type.
- **Permadeath sunset may leave dangling test fixtures.** Any
  test that exercises permadeath behavior needs updating or
  deleting. Audit during implementation.
- **Multi-killer kill-credit calculation.** Today's PlayerDamage
  map drives party-share and faction rep. Migration must
  preserve this map snapshot on the Life Dead transition (via
  DeadData.DamageMap) — observers consume the snapshot, not
  the live (now-cleared) field.

## Open questions

- **Mob `suicide` command — does it exist?** Players have
  `suicide`; verify whether mobs have an equivalent (admin
  `mob.do suicide`?). If so, ensure the migration routes it
  through Life.TransitionToDead. If not, skip.
- **Admin `kill` command location.** Confirm where the existing
  admin kill / despawn admin command lives; migrate to call
  Life.TransitionToDead with TriggerAdminKill.

## Roadmap impact

- Master spec
  `2026-05-13-combat-state-machines-design.md` references this
  as Chunk 2.
- On completion: chunk 2 marked Done in
  `COMBAT_STATE_ROADMAP.md`; chunk 3 (Activity machine)
  brainstorm begins.
- Aliveness work stays paused per master spec.

## Resumption criteria

Chunk 2 is complete when:
1. All ~27 Behavior Matrix tests pass.
2. `suicide.go` reduced from ~250 to ~30 lines.
3. All inline death-cleanup blocks routed through Life cascade
   or observers.
4. Permadeath + extra lives fully ripped (grep returns zero
   production references).
5. `ApplyHealthChange` calls `Life.TransitionToDead` instead of
   inline death sequence.
6. Mob death paths in `internal/combat/` route through
   `Life.TransitionToDead`.
7. Auto-look after respawn teleport works.
8. Stat decay / skill rust still fires on normal player death.
9. Chunk 0 (Combat Phase 32/32 tests) + chunk 1 (Awareness
   29 PASS + 4 SKIP) still green.
10. Full server boot clean; full test suite green.
