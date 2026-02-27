# GoMud JavaScript Scripting System Context

## Overview

The GoMud scripting system provides JavaScript runtime integration using the Goja JavaScript engine, enabling dynamic game logic, event handling, and content scripting. It supports scripting for rooms, mobs, items, spells, and buffs, with comprehensive API access to game systems including messaging, room manipulation, character interaction, and event management.

## Architecture

The scripting system is built on the Goja JavaScript engine with several key components:

### Core Components

**JavaScript Runtime Management:**
- `VMWrapper` - Caches compiled functions for performance optimization
- Individual VM caches for different entity types (rooms, mobs, items, spells, buffs)
- Timeout protection and memory management
- Script compilation and execution isolation

**Script Types:**
- **Room Scripts** - Handle room-specific events, commands, and interactions
- **Mob Scripts** - Control NPC behavior, AI responses, and mob-specific events  
- **Item Scripts** - Manage item events (purchase, found, lost, use)
- **Spell Scripts** - Control magic casting, waiting, and spell effects
- **Buff Scripts** - Handle status effect application and removal

**Function Categories:**
- **Actor Functions** - User and mob interaction and manipulation
- **Room Functions** - Room management, spawning, exits, and mapping
- **Item Functions** - Item creation and management
- **Messaging Functions** - Communication and text output
- **Utility Functions** - Time, dice, configuration, and helper functions

## Key Features

### 1. **Multi-Entity Scripting Support**
- Room-based event handling and command processing
- NPC AI scripting with autonomous behavior
- Item interaction and lifecycle events
- Spell casting and magical effect scripting
- Buff/debuff application and management

### 2. **Performance Optimization**
- Function caching in `VMWrapper` for repeated calls
- Separate VM instances per entity type
- Configurable timeouts (load: 1000ms, execution: 50ms)
- Memory pruning and cleanup mechanisms

### 3. **Event-Driven Integration**
- Seamless integration with GoMud's event system
- Script events triggered by game actions
- Custom event raising from JavaScript
- Timeout protection and error handling

### 4. **Comprehensive API Access**
- Full access to room manipulation and spawning
- Character stats, inventory, and skill management
- Combat system integration
- Quest and party system interaction
- Configuration and time management

### 5. **Security and Isolation**
- Timeout protection prevents infinite loops
- Memory limits and VM isolation
- Error handling with detailed logging
- Script compilation validation

## Script Entry Points and Events

### Room Script Events
```javascript
// Called when room is first loaded
function onLoad(room) {
    // Initialization logic
}

// Called periodically when room has no activity
function onIdle(room) {
    // Background processing
}

// Called for specific commands: onCommand_<commandname>
function onCommand_pull(rest, user, room) {
    // Handle 'pull' command
    return true; // Indicates command was handled
}

// Generic command handler
function onCommand(cmd, rest, user, room) {
    // Handle any unmatched command
    return false; // Allow other handlers to process
}

// Custom events triggered by game systems
function onPlayerEnter(user, room) {
    // Player entered room
}
```

### Mob Script Events
```javascript
// Called when mob is spawned
function onLoad(mob) {
    // Mob initialization
}

// Mob command handling
function onCommand_greet(rest, source, mob, room) {
    // Handle 'greet' command directed at mob
    return true;
}

// Combat and interaction events
function onPlayerDowned(downedPlayer, mob, room) {
    // Player was defeated by this mob
}

function onDeath(mob, room) {
    // Mob death handling
}
```

### Item Script Events
```javascript
// Item lifecycle events
function onFound(user, item, room) {
    // Item was picked up
}

function onLost(user, item, room) {
    // Item was dropped/lost
}

function onPurchase(user, item, room) {
    // Item was purchased
    return true; // Allow purchase
}

function onUse(user, item, room) {
    // Item was used/activated
}
```

### Spell Script Events
```javascript
// Spell casting phases
function onCast(caster, targets, room) {
    // Spell is being cast
    return true; // Allow casting
}

function onMagic(caster, targets, room) {
    // Spell effect execution
}

function onWait(caster, room) {
    // Waiting between spell rounds
}
```

## API Reference

### Actor/Character Functions
```javascript
// Get user or mob actors
var user = GetUser(userId);
var mob = GetMob(mobInstanceId);

// Actor properties and methods
user.GetName();              // Character name
user.GetRoomId();           // Current room ID
user.GetLevel();            // Character level (deprecated in DOGMud — returns 0)
user.GetGold();             // Gold amount
user.GetHealth();           // Current health
user.GetMana();             // DEPRECATED (Mana removed in DOGMud)

// Character manipulation
user.SetHealth(amount);
user.SetMana(amount);
user.GiveGold(amount);
user.TakeGold(amount);
user.GiveItem(itemId);
user.TakeItem(itemId);

// Communication
user.SendText("Hello!");
user.Command("look");       // Execute command as character
```

### Room Functions
```javascript
// Get room instance
var room = GetRoom(roomId);

// Room properties
room.GetPlayers();          // Array of players in room
room.GetMobs(mobId);        // Array of mobs (optionally filtered)
room.GetItems();            // Array of items on floor
room.GetExits();            // Array of exit information

// Room manipulation
room.SpawnMob(mobId);       // Create new mob
room.SpawnItem(itemId);     // Create new item
room.SendText("message");   // Send to all in room
room.SendTextToExits("msg", false); // Send to adjacent rooms

// Dynamic exits
room.AddTemporaryExit("portal", "shimmering portal", targetRoomId, "1h");
room.RemoveTemporaryExit("portal", "shimmering portal", targetRoomId);

// Data persistence
room.SetTempData("key", value);    // Temporary data
room.GetTempData("key");
room.SetPermData("key", value);    // Persistent data
room.GetPermData("key");
```

### Messaging Functions
```javascript
// Direct messaging
SendUserMessage(userId, "Private message");
SendRoomMessage(roomId, "Room announcement");
SendRoomExitsMessage(roomId, "Sound echoes from nearby", false);
SendBroadcast("Server-wide message");

// Console logging (for debugging)
console.log("Debug message");
console.error("Error message");
```

### Utility Functions
```javascript
// Time and rounds
UtilGetRoundNumber();               // Current game round
UtilGetTime();                      // Game time object
UtilIsDay();                        // Boolean day/night check
UtilSetTimeDay();                   // Force daytime
UtilSetTimeNight();                 // Force nighttime

// Dice and randomization
UtilDiceRoll(3, 6);                 // Roll 3d6

// String matching
var match = UtilFindMatchIn("sw", ["north", "south", "southwest"]);
// Returns: {found: true, exact: "", close: "southwest"}

// Color and formatting
ColorWrap("text", "red", "blue");   // Apply ANSI colors
UtilApplyColorPattern("text", "rainbow");

// Configuration access
var config = UtilGetConfig();

// Event system
RaiseEvent("custom-event", {data: "value"});
```

### Advanced Room Functions
```javascript
// Instance management
var instanceMap = CreateInstancesFromRoomIds([1, 2, 3]);
var zoneInstances = CreateInstancesFromZone("dungeon");

// Map generation
var mapHtml = GetMap(
    roomId,           // Center room
    2,                // Zoom level
    15,               // Height
    25,               // Width
    "Area Map",       // Title
    true,             // Show secrets
    "123,×,You are here"  // Custom markers
);

// Quest integration
var questUsers = room.HasQuest("dragon-slayer");
var missingQuest = room.MissingQuest("tutorial", userId);

// Lock management
room.IsLocked("north");
room.SetLocked("north", true);
```

## Integration Patterns

### Command Handling Priority
1. **Specific Command Functions** - `onCommand_<command>` functions are tried first
2. **Generic Command Handler** - `onCommand` function handles unmatched commands
3. **Mob Command Processing** - Mobs in room can intercept commands
4. **Exit Name Matching** - Commands matching exit names are handled

### Script Execution Flow
```javascript
// Example room script with multiple handlers
function onLoad(room) {
    room.SetTempData("initialized", true);
    console.log("Room " + room.RoomId() + " loaded");
}

function onCommand_search(rest, user, room) {
    if (rest === "desk") {
        user.SendText("You find a hidden compartment!");
        room.SpawnItem(123); // Spawn hidden item
        return true; // Command handled
    }
    return false; // Let other handlers try
}

function onCommand(cmd, rest, user, room) {
    // Log all unhandled commands for debugging
    console.log("Unhandled command: " + cmd + " " + rest);
    return false; // Don't block other processing
}

function onIdle(room) {
    // Periodic maintenance every few rounds
    var roundNum = UtilGetRoundNumber();
    if (roundNum % 100 === 0) {
        room.SendText("The room hums with magical energy.");
    }
}
```

### Error Handling and Timeouts
- **Load Timeout**: 1000ms for script compilation and `onLoad` execution
- **Execution Timeout**: 50ms for event handlers and commands
- **Automatic Cleanup**: VMs are pruned when entities are unloaded
- **Exception Logging**: JavaScript exceptions are logged with full context

### Performance Considerations
- **Function Caching**: `VMWrapper` caches compiled functions for repeated calls
- **VM Reuse**: VMs are cached per entity and reused across events
- **Memory Management**: Automatic pruning prevents memory leaks
- **Timeout Protection**: Prevents runaway scripts from blocking the server

## Usage Examples

### Interactive Room with Puzzles
```javascript
function onLoad(room) {
    room.SetPermData("puzzle-state", "unsolved");
    room.SetTempData("last-attempt", 0);
}

function onCommand_push(rest, user, room) {
    if (rest === "button" && room.GetPermData("puzzle-state") === "unsolved") {
        var sequence = room.GetTempData("button-sequence") || [];
        sequence.push("button");
        room.SetTempData("button-sequence", sequence);
        
        if (sequence.length === 3) {
            room.SetPermData("puzzle-state", "solved");
            room.SendText("The puzzle clicks into place!");
            room.SpawnItem(456); // Reward item
        }
        return true;
    }
    return false;
}
```

### Dynamic NPC Merchant
```javascript
function onCommand_trade(rest, user, mob, room) {
    var gold = user.GetGold();
    if (gold < 100) {
        user.SendText("You need at least 100 gold to trade with me.");
        return true;
    }
    
    user.TakeGold(100);
    user.GiveItem(789); // Trade item
    user.SendText("Here's something special for you!");
    return true;
}

function onPlayerDowned(downedPlayer, mob, room) {
    mob.Command("emote looks disappointed");
    mob.Command("say That's what happens when you don't pay your debts!");
}
```

### Spell with Area Effects
```javascript
function onCast(caster, targets, room) {
    // Check if spell should be allowed
    if (caster.GetMana() < 50) {
        caster.SendText("You don't have enough mana!");
        return false; // Block casting
    }
    return true; // Allow casting
}

function onMagic(caster, targets, room) {
    // Apply spell effects
    var players = room.GetPlayers();
    for (var i = 0; i < players.length; i++) {
        var player = players[i];
        if (player.UserId() !== caster.UserId()) {
            player.TakeDamage(UtilDiceRoll(2, 6));
            player.SendText("You are caught in the magical explosion!");
        }
    }
    
    room.SendText("Magical energy erupts throughout the area!");
}
```

## Dependencies

- `github.com/dop251/goja` - JavaScript runtime engine
- `internal/rooms` - Room management and world state
- `internal/users` - User session and character management
- `internal/mobs` - NPC management and AI systems
- `internal/items` - Item system and inventory management
- `internal/events` - Event-driven architecture integration
- `internal/templates` - Template processing for dynamic content
- `internal/mudlog` - Logging and debugging support

## Testing and Debugging

The scripting system includes comprehensive benchmarking tests comparing:
- Direct Goja function calls vs cached wrapper calls
- Performance with and without function caching
- Missing vs found function lookup performance

**Debugging Features:**
- Console logging support (`console.log`, `console.error`)
- Detailed error logging with script context
- Performance timing for all script executions
- VM cache statistics and pruning logs

This JavaScript scripting system enables GoMud to provide rich, dynamic content and sophisticated game mechanics while maintaining performance and security through proper isolation and timeout management.