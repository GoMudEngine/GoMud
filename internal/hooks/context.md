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

### NewRound Event Handlers (15 handlers)
```go
// Core game loop processing every round
events.RegisterListener(events.NewRound{}, PruneVMs)              // Clean up JavaScript VMs
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
    
    // Execute join scripts
    if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
        if room.HasScript() {
            scripting.TryRoomScriptEvent("onPlayerEnter", room.RoomId, evt.UserId, 0)
        }
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
    
    // Execute leave scripts
    if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
        if room.HasScript() {
            scripting.TryRoomScriptEvent("onPlayerLeave", room.RoomId, evt.UserId, 0)
        }
    }
    
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
    
    // Execute idle command
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
- scripting.TryRoomScriptEvent()  // Scripting system integration
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

## Dependencies

- `internal/events` - Event system for listener registration and event processing
- `internal/users` - User management for player-related hooks
- `internal/mobs` - NPC management for mob-related hooks
- `internal/combat` - Combat system for battle resolution
- `internal/quests` - Quest system for progression tracking
- `internal/rooms` - Room management for location-based events
- `internal/scripting` - JavaScript runtime for script execution
- `internal/buffs` - Status effects for buff management
- `internal/configs` - Configuration management for system settings
- `internal/mudlog` - Logging system for debugging and monitoring

This comprehensive hooks system provides the core game logic implementation through event-driven architecture, handling everything from basic gameplay mechanics to complex system maintenance tasks while maintaining clean separation of concerns and extensible design patterns.