# GoMud Hooks System Context

## Overview

The GoMud hooks system provides comprehensive event-driven game logic through a collection of 39 specialized event listeners that handle everything from combat rounds to quest progression. It serves as the primary integration layer between the event system and game mechanics, implementing core gameplay features like combat resolution, mob AI, player lifecycle management, and system maintenance tasks.

## Architecture

The hooks system is built around several key categories:

### Core Components

**Event Registration System:**
- Centralized listener registration in `RegisterListeners()`
- Type-safe event handling with proper casting
- Ordered execution with priority support (events.Last)
- Comprehensive coverage of all game events

**Game Loop Hooks:**
- **NewRound Events**: Combat, healing, mob AI, player ticks
- **NewTurn Events**: Autosave, cleanup, buff management
- **Player Lifecycle**: Spawn, despawn, character changes
- **System Maintenance**: VM pruning, zombie cleanup, respawns

**Gameplay Integration:**
- **Combat System**: Full combat round processing with multi-target support
- **Quest System**: Progress tracking and reward distribution
- **Buff System**: Application, expiration, and effect processing
- **Audio System**: MSP sound effects and location-based music

## Key Features

### 1. **Comprehensive Game Loop Management**
- **Round Processing**: 15 different NewRound event handlers
- **Turn Processing**: 4 NewTurn event handlers for maintenance
- **Combat Integration**: Complete combat round resolution
- **Mob AI Processing**: Idle behavior and action execution

### 2. **Player Lifecycle Management**
- **Join/Leave Handling**: Player spawn and despawn processing
- **Character Updates**: Broadcasting character changes
- **Skill Progression**: Skill-use notifications and guide spawning (Level-up disabled in DOGMud)
- **Connection Management**: Zombie cleanup and inactive player handling

### 3. **Quest and Progression Systems**
- **Quest Processing**: Multi-step quest advancement and rewards
- **Item Integration**: Quest item requirements and rewards
- **Skill Advancement**: Skill-based quest completion
- **Progression Distribution**: Skill progression rewards and notifications

### 4. **System Maintenance and Optimization**
- **Automatic Cleanup**: Zombie connections, expired buffs, ephemeral rooms
- **Resource Management**: VM pruning, memory optimization
- **Data Persistence**: Automatic user saves and data integrity
- **Performance Monitoring**: Event processing and system health

## Event Listener Categories

### NewRound Event Handlers (14 handlers)
```go
// Core game loop processing every round
events.RegisterListener(events.NewRound{}, InactivePlayers)       // Handle AFK players
events.RegisterListener(events.NewRound{}, UpdateZoneMutators)    // Update zone effects
events.RegisterListener(events.NewRound{}, CheckNewDay)           // Day/night cycle
events.RegisterListener(events.NewRound{}, SpawnLootGoblin)       // Special mob spawning
events.RegisterListener(events.NewRound{}, UserRoundTick)         // Player round processing
events.RegisterListener(events.NewRound{}, MobRoundTick)          // NPC round processing
events.RegisterListener(events.NewRound{}, HandleRespawns)        // Mob respawning
events.RegisterListener(events.NewRound{}, DoCombat)              // Combat resolution
events.RegisterListener(events.NewRound{}, AutoHeal)              // Natural healing
events.RegisterListener(events.NewRound{}, IdleMobs)              // Mob idle behavior
```

### NewTurn Event Handlers (4 handlers)
```go
// System maintenance every turn (multiple rounds)
events.RegisterListener(events.NewTurn{}, CleanupZombies)         // Remove disconnected users
events.RegisterListener(events.NewTurn{}, AutoSave)               // Automatic data saves
events.RegisterListener(events.NewTurn{}, PruneBuffs)             // Remove expired buffs
events.RegisterListener(events.NewTurn{}, ActionPoints)           // Regenerate action points
```

### Player Lifecycle Handlers
```go
// Player connection and character management
events.RegisterListener(events.PlayerSpawn{}, HandleJoin)         // Player login processing
events.RegisterListener(events.PlayerDespawn{}, HandleLeave, events.Last) // Player logout (final)
events.RegisterListener(events.PlayerDrop{}, HandlePlayerDrop)    // Unexpected disconnection
events.RegisterListener(events.CharacterCreated{}, BroadcastNewChar) // New character announcements
events.RegisterListener(events.CharacterChanged{}, BroadcastNewChar) // Character update announcements
```

### Game Mechanics Handlers
```go
// Core gameplay systems
events.RegisterListener(events.Quest{}, HandleQuestUpdate)        // Quest progression
events.RegisterListener(events.Buff{}, ApplyBuffs)               // Buff application
events.RegisterListener(events.LevelUp{}, SendLevelNotifications) // Level-up messages
events.RegisterListener(events.LevelUp{}, CheckGuide)             // Guide NPC spawning
events.RegisterListener(events.ItemOwnership{}, CheckItemQuests)  // Item-based quests
events.RegisterListener(events.MobIdle{}, HandleIdleMobs)         // Mob AI behavior
```

## Combat System Integration

### Combat Round Processing
```go
func DoCombat(e events.Event) events.ListenerReturn {
    evt := e.(events.NewRound)
    
    // Process all active combat encounters
    for _, user := range users.GetAllActiveUsers() {
        if user.Character.IsAggro() {
            // Handle player combat
            processCombatRound(user)
        }
    }
    
    // Process mob vs mob combat
    for _, mobInstanceId := range mobs.GetAllMobInstanceIds() {
        mob := mobs.GetInstance(mobInstanceId)
        if mob != nil && mob.Character.IsAggro() {
            processMobCombat(mob)
        }
    }
    
    return events.Continue
}

// Combat processing includes:
// - Multi-target combat resolution
// - Weapon durability and breakage
// - Death handling and consequences
// - Experience and loot distribution
// - Combat state management
```

## Quest System Integration

### Quest Progress Handling
```go
func HandleQuestUpdate(e events.Event) events.ListenerReturn {
    evt := e.(events.Quest)
    
    user := users.GetByUserId(evt.UserId)
    if user == nil {
        return events.Cancel
    }
    
    // Validate quest progression
    if !quests.IsTokenAfter(user.Character.GetCurrentQuestToken(), evt.QuestToken) {
        return events.Cancel
    }
    
    // Update quest progress
    user.Character.SetQuestFlag(evt.QuestToken)
    
    // Check for quest completion
    quest := quests.GetQuest(evt.QuestToken)
    if quest != nil && isQuestComplete(quest, evt.QuestToken) {
        distributeQuestRewards(user, quest)
    }
    
    return events.Continue
}

// Quest processing includes:
// - Multi-step quest validation
// - Item requirement checking
// - Skill-based quest completion
// - Reward distribution (gold, items, experience, skills)
// - Chained quest activation
```

## Player Lifecycle Management

### Player Join Processing
```go
func HandleJoin(e events.Event) events.PlayerSpawn {
    evt := e.(events.PlayerSpawn)
    
    user := users.GetByUserId(evt.UserId)
    if user == nil {
        return events.Cancel
    }
    
    // Handle first-time login
    if user.Character.Level == 1 && user.Character.Experience == 0 {
        handleNewPlayerSetup(user)
    }
    
    // Broadcast join message
    broadcastPlayerJoin(user)
    
    return events.Continue
}
```

### Player Leave Processing
```go
func HandleLeave(e events.Event) events.ListenerReturn {
    evt := e.(events.PlayerDespawn)
    
    user := users.GetByUserId(evt.UserId)
    if user == nil {
        return events.Cancel
    }
    
    // Save user data
    if err := user.Save(); err != nil {
        mudlog.Error("HandleLeave", "userId", evt.UserId, "error", err)
    }
    
    // Clean up combat state
    user.Character.ClearAggro()
    
    // Broadcast leave message
    broadcastPlayerLeave(user)
    
    return events.Continue
}
```

## System Maintenance Hooks

### Automatic Cleanup
```go
// Zombie connection cleanup
func CleanupZombies(e events.Event) events.ListenerReturn {
    evt := e.(events.NewTurn)
    
    expirationTurn := evt.TurnNumber - configs.GetNetworkConfig().LogoutRounds
    expiredZombies := users.GetExpiredZombies(expirationTurn)
    
    for _, userId := range expiredZombies {
        user := users.GetByUserId(userId)
        if user != nil {
            user.Save()
            users.RemoveUser(userId)
        }
    }
    
    return events.Continue
}

// Buff expiration management
func PruneBuffs(e events.Event) events.ListenerReturn {
    evt := e.(events.NewTurn)
    
    // Prune user buffs
    for _, user := range users.GetAllActiveUsers() {
        prunedBuffs := user.Character.Buffs.Prune()
        for _, buff := range prunedBuffs {
            notifyBuffExpiration(user, buff)
        }
    }
    
    // Prune mob buffs
    for _, mobInstanceId := range mobs.GetAllMobInstanceIds() {
        mob := mobs.GetInstance(mobInstanceId)
        if mob != nil {
            mob.Character.Buffs.Prune()
        }
    }
    
    return events.Continue
}
```

### Automatic Saves
```go
func AutoSave(e events.Event) events.ListenerReturn {
    evt := e.(events.NewTurn)
    
    // Save all active users periodically
    if evt.TurnNumber%configs.GetGamePlayConfig().AutoSaveFrequency == 0 {
        for _, user := range users.GetAllActiveUsers() {
            if err := user.Save(); err != nil {
                mudlog.Error("AutoSave", "userId", user.UserId, "error", err)
            }
        }
    }
    
    return events.Continue
}
```

## Audio and Visual Effects

### MSP Sound System
```go
func PlaySound(e events.Event) events.ListenerReturn {
    evt := e.(events.MSP)
    
    user := users.GetByUserId(evt.UserId)
    if user == nil || !user.ClientSettings().IsMsp() {
        return events.Continue
    }
    
    // Send MSP sound command
    soundCommand := fmt.Sprintf("!!SOUND(%s)", evt.SoundFile)
    user.SendText(soundCommand)
    
    return events.Continue
}

// Location-based music changes
func LocationMusicChange(e events.Event) events.ListenerReturn {
    evt := e.(events.RoomChange)
    
    user := users.GetByUserId(evt.UserId)
    if user == nil {
        return events.Continue
    }
    
    room := rooms.LoadRoom(evt.RoomId)
    if room != nil && room.MusicFile != "" {
        if user.LastMusic != room.MusicFile {
            user.PlayMusic(room.MusicFile)
            user.LastMusic = room.MusicFile
        }
    }
    
    return events.Continue
}
```

## Mob Round Tick (`NewRound_MobRoundTick.go`)

The MobRoundTick handler runs every round and processes per-mob updates including
buff triggers, stat/skill progression, pack scaling, and mutation acquisition.

### Pack Scaling (before per-mob loop)
```go
// TickPackSurvival returns []PackBonus — data structs to avoid import cycle
// with the rooms package. The hook handles room messaging and world events.
if b.PackScalingEnabled {
    for _, bonus := range mobs.TickPackSurvival() {
        // Emit room message: "The <group> pack moves with renewed coordination."
        // Emit WorldEvent{Type: PackStrengthened}
        // Significance: first bonus → Local, reaching max → Regional
    }
}
```

### Mob Mutation Acquisition (inside per-mob loop)
After buff triggers and before `Validate()`:
```go
// Guard: MobMutationEnabled && mob.Character.Aggro != nil
// Progress: += MutationProgressGainPerRound * MobMutationRate
// Threshold: MutationBaseProgress * MutationProgressScale^mutationLoad
// On acquire/deepen:
//   - Room flavor text
//   - EmitWorldEvent(MobMutationGained/Advanced)
//   - Significance based on mutation rarity (>=8 Global, >=5 Regional, else Local)
//   - Deepening to level 3 bumps significance one tier
```

### Per-Mob Loop Order
1. Buff trigger checks
2. Stat/skill progression (`MobProgressionEnabled`)
3. **Mutation acquisition** (`MobMutationEnabled`)
4. `Character.Validate()`

---

## Mob AI and Behavior

### Idle Mob Processing
```go
func IdleMobs(e events.Event) events.ListenerReturn {
    evt := e.(events.NewRound)

    for _, mobInstanceId := range mobs.GetAllMobInstanceIds() {
        mob := mobs.GetInstance(mobInstanceId)
        if mob == nil || mob.Character.IsAggro() {
            continue
        }

        // Check activity level for idle behavior
        if util.Rand(100) < mob.ActivityLevel {
            events.AddToQueue(events.MobIdle{
                MobInstanceId: mobInstanceId,
            })
        }
    }

    return events.Continue
}

func HandleIdleMobs(e events.Event) events.ListenerReturn {
    evt := e.(events.MobIdle)

    mob := mobs.GetInstance(evt.MobInstanceId)
    if mob == nil {
        return events.Continue
    }

    // --- Crafter tick (fires on restock cycle, not every idle tick) ---
    // TickMobCraft returns a CraftResult only when a craft is attempted.
    // The hook handles room messaging and world event emission to avoid
    // import cycles in the mobs package.
    if result := mobs.TickMobCraft(mob); result != nil {
        // Emit room flavor text (success/failure)
        // Emit MobCraftedRare world event if SkillMinimum >= CrafterRareThreshold
    }

    // Execute idle command (runs alongside crafting)
    idleCommand := mob.GetIdleCommand()
    if idleCommand != "" {
        mob.Command(idleCommand)
    }

    return events.Continue
}
```

## Integration Patterns

### Event System Integration
```go
// All hooks integrate with the event system
- events.RegisterListener()        // Register event handlers
- events.AddToQueue()             // Queue new events from handlers
- events.Continue/Cancel          // Control event processing flow
```

### Cross-System Communication
```go
// Hooks coordinate between systems
- users.GetByUserId()             // User management integration
- rooms.LoadRoom()                // Room system integration
- mobs.GetInstance()              // Mob system integration
- combat.AttackPlayerVsMob()      // Combat system integration
```

## Usage Examples

### Custom Hook Registration
```go
// Register custom event listener
func RegisterCustomHook() {
    events.RegisterListener(events.CustomEvent{}, func(e events.Event) events.ListenerReturn {
        evt := e.(events.CustomEvent)
        
        // Custom processing logic
        processCustomEvent(evt)
        
        return events.Continue
    })
}
```

### Event Processing Flow
```go
// Example of event flow through hooks
// 1. Player attacks mob
events.AddToQueue(events.Combat{
    AttackerId: userId,
    TargetId:   mobInstanceId,
})

// 2. Combat hook processes attack
func DoCombat(e events.Event) events.ListenerReturn {
    // Resolve combat
    result := combat.AttackPlayerVsMob(user, mob)
    
    // Check for death
    if mob.Character.Health <= 0 {
        events.AddToQueue(events.MobDeath{
            MobInstanceId: mobInstanceId,
            KillerId:      userId,
        })
    }
    
    return events.Continue
}
```

### System Maintenance
```go
// Hooks handle automatic system maintenance
func SystemMaintenance(e events.Event) events.ListenerReturn {
    evt := e.(events.NewTurn)
    
    // Periodic maintenance tasks
    if evt.TurnNumber%100 == 0 {
        // Clean up resources
        cleanupExpiredData()
        
        // Optimize performance
        optimizeMemoryUsage()
        
        // Update statistics
        updateSystemStats()
    }
    
    return events.Continue
}
```

## Combat State Machine Integration (chunk 0)

Four files in the hooks package wire the Combat Phase machine into the
engine without creating import cycles (the characters package cannot
import hooks; hooks import characters and register via `OnCharacterCreated`).

### CombatPhase_Vetoes.go

Registers the seven veto callbacks on every new `Character` via
`characters.OnCharacterCreated(wireCombatPhaseVetoes)`.

Each veto reads the current character field for its concern. Future
chunks replace each closure body as the corresponding machine lands
(e.g., `RegisterLifeCheck` will read `c.LifeMachine.State() == Alive`
once the Life machine ships in chunk 2).

| Veto registration | Reads |
|-------------------|-------|
| `RegisterCombatantVeto` | `c.IsCombatant()` |
| `RegisterActivityCheck` | `c.IsActing()` (negated) — queries Activity machine |
| `RegisterLifeCheck` | `c.Health > 0` |
| `RegisterPositionCheck` | `c.IsStanding()` (Position FSM, chunk 4b R5) |
| `RegisterTargetCombatantCheck` | target's `IsCombatant()` via users/mobs lookup |
| `RegisterTargetLifeCheck` | target's `Health > 0` via users/mobs lookup |
| `RegisterTargetPresenceCheck` | player grace buff (`NoAggroTarget`) check |

### CombatPhase_BtreeEvents.go

Registers an `AfterTransition` cascade that fires btree transition events
whenever a mob's Combat Phase state changes. Player characters also have
`CombatPhase` but the btree system only fires for mob instances.

Events fired (once per state transition, not per round):
- `mob_engaging` — `Idle → Engaging`
- `mob_engaged` — `Engaging → Engaged` (after `RoundsUntil` countdown)
- `mob_disengaging` — `Engaged → Disengaging` (flee initiated)
- `mob_combat_ended` — any → `Idle` (target died, flee succeeded, etc.)

Tick events (`mob_combat_round`, `mob_idle`) fire from the round driver
via `DispatchTickEvent`, not from this file.

Mob ownership is resolved via `findMobOwningCharacter`, an O(N) scan
over all mob instances that compares `Character` pointer identity. This
is acceptable because transition events fire at most once per state
change (not per round).

### CombatPhase_CompanionAssist.go

Registers `SubscribeAttackersChange` on every character. When a charmed
companion's inbound attacker list grows (new attacker recorded), the
handler reactively directs the companion's owner and sibling companions
to join the fight — without waiting for the next round tick.

Behavioral parity with the old polling path in `NewRound_DoCombat`:
- Same `AutoAssist` flag check on the companion entry
- Same `NoAggroTarget` grace-period guard on the owner
- Sibling companions in the same room are also assisted

The polling `CompanionAutoTarget` in `combat_retarget.go` remains as a
fallback. Duplicate attack commands are benign (second attempt is vetoed
by the already-fighting state).

### combat_retarget.go

Contains three functions moved from the deleted `aggro_helpers.go` in
chunk 0's sunset pass. Still consumed by `NewRound_DoCombat`.

- **`ValidateAggro(char)`** — checks if the character's `Aggro` target
  still exists and is alive in the same room; calls `EndAggro()` and
  returns false if stale.
- **`RetargetOrEnd(char, room, userId, mobInstanceId)`** — clears current
  aggro and scans the room for a new target already attacking the
  character (or the character's companions). Returns true if a new target
  was found and `SetAggro` was called.
- **`CompanionAutoTarget(mob, room)`** — polling fallback for companion
  auto-assist. Runs once per round in `NewRound_DoCombat`. Directs idle
  companions to join the owner's fight or intercept mobs attacking the
  owner.

### Round driver dispatch (NewRound_DoCombat.go)

The round driver reads Combat Phase state instead of legacy `Aggro`:

- `c.IsInCombat()` replaces `c.Aggro != nil` in the "who is fighting?"
  loop.
- `c.CombatPhase.OnRoundTick()` advances `Engaging` → `Engaged` when
  `RoundsUntil` hits zero.
- `c.CombatPhase.DispatchTickEvent()` fires `mob_combat_round` or
  `mob_idle` btree events per character per round.
- `c.CombatPhase.OnCombatRoundEnd()` clears the `SurpriseLeft` flag
  at end-of-round for surprise engagements.

## Awareness State Machine Integration (chunk 1)

Four files in the hooks package wire the Awareness machine into the
engine without creating import cycles (the characters package cannot
import hooks; hooks import characters and register via `OnCharacterCreated`).

### Awareness_Vetoes.go

Registers the activity check and detection-roll veto callbacks on every new
`Character` via `characters.OnCharacterCreated(wireAwarenessVetoes)`.

Each veto reads the current character field for its concern.

| Veto registration | Reads |
|-------------------|-------|
| `RegisterActivityCheck` | `c.IsActing()` (negated) — queries Activity machine |
| `RegisterDetectionCheck` | validates sneak attempt is proceeding (scaffold) |

### Awareness_Cascades.go

Registers an `AfterTransition` callback on the Awareness machine. When
the machine transitions away from or into the `Hidden` state, the hook
applies or removes buff #9 to keep the visible effect synchronized with
the invisible state.

Also subscribes to Combat Phase's `OnEndOfRoundIfSurprise` callback. When
a surprise engagement completes its first round, the hook triggers the
Awareness reveal cascade (`Hidden → Revealing → Visible`), forcing any
hidden characters out of hiding.

Events and cascades (per state transition, not per round):
- Awareness `Visible → Hidden`: apply buff #9 + room text "sneaks away"
- Awareness `Hidden → Visible`: remove buff #9 + room text "emerges from hiding"
- Combat Phase end-of-surprise round: trigger Awareness reveal cascade

### Awareness_LightChange.go

Scaffolding for future light-source re-roll mechanics. Registers a
`OnCharacterCreated` callback to set up the listener registration hooks
for light-state-change events. Today a no-op pending full light-system
design; the file exists to document the integration point for future
chapters.

### Logout_AwarenessCleanup.go

Registers an `OnPlayerDespawn` listener that calls `character.Awareness.ForceVisible()`
to ensure the awareness machine is reset on logout. Prevents stale awareness
state or leaks if a character is reused or respawned.

## Life Machine Cascade + Death/Respawn Observers (chunk 2)

Fourteen files in the hooks package wire the Life machine into the
engine without creating import cycles. Each file registers its
observer via `characters.OnCharacterCreated(wireXxx)` at `init()`
time. Player-only observers gate on `c.GetUserId() != 0`; mob-only
observers gate on `c.MobInstanceId != 0`.

### Life_Cascades.go

Cross-machine cleanup that fires on two Life transitions:

**Alive → Dead:**
- Forces Combat Phase to `Idle` (`ForceIdle`)
- Forces Awareness to `Visible` (`ForceVisible`)
- Transitions Activity machine to `Free` (via separate `activity_life_dead`
  observer in `Activity_Cascades.go` — see Activity Machine section below)
- (The legacy `CombatPosition` reset and `GrappleControllerId` clear
  that previously lived here were deleted in chunk 4b R4. The
  `position_life_dead` observer in `Position_Cascades.go` owns the
  Position FSM death cascade.)
- Cancels all non-permanent active buffs
- Clears active combat conditions

**Dead → Respawning:**
- Refills all resource pools to 5% of max
- Applies `NoAggroTarget` grace buff (#81)
- Clears live `PlayerDamage` map (snapshot already in `DeadData`)
- Queues `CharacterVitalsChanged` event

### Death observers

| File | Purpose |
|------|---------|
| `Death_PlayerCleanup.go` | Stat decay + skill rust penalties, KD tracking (death count), party death notifications |
| `Death_PlayerAnnouncement.go` | Room broadcast, global broadcast, `events.PlayerDeath` queue, worldevents PvE emit, weakened/darkness text, instance ejection |
| `Death_PlayerCorpse.go` | Player corpse creation in the death room |
| `Death_InboundAggroCleanup.go` | Clears mobs and companions that were targeting the dying actor; fires for both player and mob deaths |
| `Death_MobLoot.go` | Carried and equipped item drop, gold drop, dark-room sound cue, mob corpse creation |
| `Death_AlivenessSubstrate.go` | Fires `events.MobDeath`; downstream subscribers handle faction rep, opinion update, crime recording, knowledge propagation, bounty resolution |
| `Death_MobInstanceCleanup.go` | `DeleteMobInstance`, `DestroyInstance`, `CleanupMobSpawns`, `RemoveMob` |
| `Death_MobBroadcast.go` | Room "X has died" broadcast, Guide tempdata, worldevents `MobKilledByPlayer` |
| `Death_MobBehaviorTree.go` | Fires `mob_die` btree event with primary killer's `UserId` |
| `Death_MobKillCredit.go` | `EndAggro` on killers, `KD.AddMobKill`, `OnFirstMobKill`, party kill credit |
| `Death_MobCharmCleanup.go` | `TrackRecentDeath`, `RemoveCharm`, reverse-track player `TrackCharmed` |

### Respawn observers

| File | Purpose |
|------|---------|
| `Respawn_PlayerTeleport.go` | `rooms.MoveToRoom` to `c.ResolveRespawnRoom()` destination; belt-and-suspenders `EndAggro` |
| `Respawn_PlayerAutoLook.go` | Fires `u.Command("look")` for room-render UX after respawn teleport |

### Wiring pattern

All fourteen files follow the same registration pattern:

```go
func init() {
    characters.OnCharacterCreated(wireXxx)
}

func wireXxx(c *characters.Character) {
    c.Life.Inner().AfterTransition(func(from, to life.State,
        r state.TransitionReason) {
        if from != life.Alive || to != life.Dead {
            return
        }
        // ... observer logic, gated by c.GetUserId() != 0
        // or c.MobInstanceId != 0 as appropriate
    })
}
```

The `AfterTransition` callbacks on the `state.Machine[State]` inner
framework call all registered observers synchronously before
returning control to the caller. This means by the time
`c.Life.TransitionToDead(...)` returns, all death-cascade side
effects have already fired.

## Activity Machine Cascade + Observers (chunk 3)

One file in the hooks package wires the Activity machine into the engine
without creating import cycles (same pattern as chunks 0-2).

### Activity_Cascades.go

Registers one `AfterTransition` observer via
`characters.OnCharacterCreated(wireActivityCrossMachineCascades)`.

**`activity_life_dead` handler — Life `Alive → Dead` → Activity `→ Free`:**

When the Life machine transitions `Alive → Dead`, the handler calls
`c.Activity.TransitionToFree(TriggerDeath)` if any activity is in
flight. This repoints the chunk-2 pre-wire in `Life_Cascades.go` (which
niled `CastingState` and `CraftingState` directly) onto a proper
Activity-side observer. All three active states (Casting, Crafting,
Salvaging) transition to Free; there is no casting exemption for the
death cascade.

**Combat-entry cancellation — implemented via veto, not cascade:**

Crafting and Salvaging block the character from entering combat
(`Idle → Engaging`). This is implemented in `CombatPhase_Vetoes.go` —
`RegisterActivityCheck` returns `!c.IsCrafting() && !c.IsSalvaging()`,
so the veto fires only when one of those two activities is active.
Casting is exempt (cast IS a combat action — the character continues
casting through combat entry, with damage handled separately via the
concentration-break path). A separate `AfterTransition` cascade for
combat-entry was evaluated and removed as unreachable (the veto fires
before the transition succeeds for craft/salvage; nothing to cascade
for casting).

### Call-site wirings (not AfterTransition)

Movement and damage interrupts do not fit the machine-to-machine
`AfterTransition` pattern; they are wired directly at their call sites:

| Interrupt | Location | Trigger fired |
|-----------|----------|---------------|
| Movement (Crafting/Salvaging) | `internal/usercommands/go.go` | `TriggerMovementInterrupt` |
| Damage taken (Crafting/Salvaging) | `cancelCraftOrSalvageOnDamage` in `combat_shared_helpers.go` | `TriggerDamageInterrupt` |
| Damage taken (Casting) | `clearCastingActivity` in `combat_shared_helpers.go` | `TriggerConcentrationBreak` on roll failure |

Completion triggers are fired by per-tick consumers after a successful
`Advance*` call:

| Completion | Location | Trigger fired |
|------------|----------|---------------|
| Cast completes | `processFoldRound` in `NewRound_UserRoundTick.go` | `TriggerCastComplete` |
| Craft completes (player) | inline craft-tick block in `NewRound_UserRoundTick.go` | `TriggerCraftComplete` |
| Craft completes (mob) | inline craft-tick block in `NewRound_MobRoundTick.go` | `TriggerCraftComplete` |
| Salvage completes (player) | inline salvage-tick block in `NewRound_UserRoundTick.go` | `TriggerSalvageComplete` |
| Salvage completes (mob) | inline salvage-tick block in `NewRound_MobRoundTick.go` | `TriggerSalvageComplete` |

## Position Cascade + Observers (chunks 4a + 4b)

Four files in the hooks package wire the Position machine into the
engine (same import-cycle-free pattern as chunks 0-3). One file
scaffolded the cascade in 4a; three more landed in 4b with the
control-axis cutover.

### Position_Cascades.go (chunk 4a)

Registers one `AfterTransition` observer on the Life machine via
`characters.OnCharacterCreated(wirePositionCrossMachineCascades)`.

**`position_life_dead` handler — Life `Alive → Dead` → Position → Standing:**

When the Life machine transitions `Alive → Dead`, the handler calls
`c.Position.TransitionToStanding(TriggerDeath)` if the Position machine
is non-nil and not already `Standing`. This ensures that a character
who dies while grappled or knocked down returns to the `Standing` default.

This observer is now the sole Position reset on death. Chunk 4b R4
deleted the chunk-2 `Life_Cascades.go` pre-wire that previously reset
`c.CombatPosition = PositionStanding` and `c.GrappleControllerId = 0`
directly. Those legacy fields no longer exist (T21 sunset).

**Integration tests** in `Position_Cascades_test.go` cover four scenarios:
- PO-037: Standing at death → remains Standing (no-op observer path)
- PO-038: Mount at death → cascades to Standing
- PO-039: Guard at death → cascades to Standing
- PO-040: BackGround at death → cascades to Standing

### Position_GrappleTick.go (chunk 4b)

Per-round drift observer registered via
`wirePositionGrappleTick` on character creation. Fires from the
NewRound tick walker and drives three things for every character
currently in a grapple:

1. **Opposed control rolls** — Strength + Unarmed-combat for both
   sides, modified by `grappleStaminaMultiplier` (curve config
   `GrappleStaminaPenaltyMax` / `Curve`) and the encumbrance multiplier
   (curve `GrappleEncumbrancePenaltyMax` / `Curve`). The winning
   margin's `ZScore` shifts `ControlLevel` along the
   InControl ↔ LosingControl ↔ Neutral ↔ BecomingControlled ↔ Controlled
   axis via `MutateGrappleControlLevel` (without firing a state
   transition — the FSM forbids `Mount→Mount`, so per-round drift
   mutates the shared `GrappleData` directly).
2. **Per-round stamina cost** — `GrappleStaminaCostPerRound`, scaled
   by `GrappleControllerCostMultiplier` (default 1.0) for the
   controller side or `GrappleControlledCostMultiplier` (default 2.0)
   for the controlled side. The asymmetry creates the "smother" feedback
   loop: controlled side gases out first.
3. **Threshold-triggered position transitions** — when `ControlLevel`
   crosses `InControl`/`Controlled` thresholds, fires a follow-up
   position transition (e.g. Mount controller crosses to deeper
   control → still Mount but with `ControlLevel: InControl` set; a
   controlled grappler crossing to `Controlled` may transition out via
   `TransitionToTurtle` defensive curl).

Smother edge case: when stamina hits 0 the character keeps grappling
(no FSM transition) and the penalty curve maxes out — the controlled
side simply gets gassed faster, reinforcing the feedback loop.

### Position_Messaging.go (chunk 4b)

Per-round messaging observer. Subscribes via `wirePositionMessaging`
to the GrappleTick walker and the Position FSM's `AfterTransition`.
Generates three message classes with per-grapple cooldowns
(`Character.PerGrappleMessageCooldowns map[string]int`):

- **Gradient messages** — "you're losing control of the grapple",
  "your grip is slipping", etc. Fire once per grapple per
  `ControlLevel` crossing.
- **Transition messages** — "you scramble out of mount and into
  guard", "they pull you down to half-guard". Fire on every Position
  state change while grappling; have controller / controlled / room
  variants.
- **Stamina warnings** — "you're getting gassed" — fire once per
  grapple when stamina drops below `GrappleStaminaLowThreshold`
  (config, default 0.25). `IsLowGrappleStamina()` is the predicate.

Cooldowns reset when the grapple ends (any `TransitionToStanding` via
escape, break, or death).

### Position_ConsistencyCheck.go (chunk 4b)

Periodic invariant checker registered via
`wirePositionConsistencyCheck`. Walks character pairs and calls
`position.ValidateGrapplePair(a, b)` to verify:

- If `a.IsGrappling()` and references `b` via `GrappleData.Partner`,
  then `b.IsGrappling()` and references `a` symmetrically.
- ControlLevel relationship is consistent with the pair role (one
  controller / one controlled / mutual neutral).
- No orphan grapples (character in a grapple state with no Partner
  except Turtle).

Logs WARN on any invariant violation. Cheap to run (small pair
universe in any one room); intended as a safety net during 4b's
parallel-write window.

## Dependencies

- `internal/events` - Event system for listener registration and event processing
- `internal/users` - User management for player-related hooks
- `internal/mobs` - NPC management for mob-related hooks
- `internal/combat` - Combat system for battle resolution
- `internal/quests` - Quest system for progression tracking
- `internal/rooms` - Room management for location-based events
- `internal/buffs` - Status effects for buff management
- `internal/configs` - Configuration management for system settings
- `internal/mutations` - Mutation system for mob mutation acquisition
- `internal/worldevents` - World event recording for emergent behavior milestones
- `internal/mudlog` - Logging system for debugging and monitoring
- `internal/state/combatphase` - Combat Phase state machine (chunk 0)
- `internal/state/awareness` - Awareness state machine (chunk 1)
- `internal/state/life` - Life state machine (chunk 2)
- `internal/state/position` - Position state machine (chunks 4a + 4b)