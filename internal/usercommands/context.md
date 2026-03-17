# User Commands Package Context

## Overview
The `internal/usercommands` package implements the complete command system for player interactions in GoMud. It defines all player-executable commands, from basic movement and communication to complex skills, combat actions, and administrative functions.

## Key Components

### Command Architecture (`usercommands.go`)
- **UserCommand function signature**: Standardized interface for all commands
  ```go
  type UserCommand func(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error)
  ```
- **CommandAccess structure**: Defines command permissions and restrictions
- **Command registry**: Central mapping of command names to implementations
- **Permission system**: Admin-only commands and downed-state restrictions

### Command Categories

#### **Basic Interaction Commands**
- **Movement**: `go`, `flee` - Navigation and escape mechanics
- **Communication**: `say`, `shout`, `whisper`, `emote`, `broadcast` - Player communication
- **Observation**: `look`, `inspect`, `consider`, `who`, `online` - Information gathering
- **Inventory**: `inventory`, `get`, `drop`, `give`, `put` - Item management

#### **Combat Commands**
- **Direct combat**: `attack`, `shoot`, `throw` - Offensive actions
- **Combat skills**: `disarm`, `tackle`, `backstab`, `recover` - Specialized combat techniques
- **Defensive**: `flee`, `aid` - Escape and assistance mechanics

#### **Skill-Based Commands**
- **Magic system**: `cast`, `enchant`, `unenchant`, `prepare` - Spellcasting mechanics
- **Stealth**: `sneak`, `picklock`, `pickpocket`, `peep` - Stealth and thievery
- **Utility skills**: `map`, `track`, `search`, `portal` - Exploration and navigation
- **Crafting**: Various skill-based creation and modification commands

#### **Economic Commands**
- **Trading**: `buy`, `sell`, `list`, `offer`, `appraise` - Commerce mechanics
- **Banking**: `bank` - Financial management
- **Services**: `train` - Character development

#### **Social and Party Commands**
- **Party system**: `party` - Group management and coordination
- **Pets**: `pet`, `tame` - Animal companion system
- **Character management**: `character`, `set`, `alias` - Character customization

#### **Administrative Commands** (Admin-only)
- **World building**: `room`, `build`, `zone` - Environment creation and modification
- **Entity management**: `mob`, `item`, `spawn` - Game object manipulation
- **Server management**: `server`, `reload`, `teleport` - System administration
- **Player management**: `grant`, `modify`, `mute`, `deafen` - Player administration
- **Combat analytics**: `combatstats` - Combat analytics dashboard — view, filter, reset, export combat event data

### Command Processing Features

#### **Input Parsing and Validation**
- **Argument parsing**: Sophisticated parsing with quote respect for complex arguments
- **Target resolution**: Finding players, mobs, and objects by name or partial match
- **State validation**: Checking combat status, buffs, and other restrictions

#### **Permission and Security**
- **Role-based access**: Admin commands restricted by user permissions
- **State restrictions**: Commands blocked when downed, in combat, or affected by buffs
- **Cooldown management**: Time-based restrictions on command usage

#### **Event Integration**
- **Event flags**: Commands can be executed secretly or with special modifiers
- **Event emission**: Commands trigger events for logging and system integration
- **Combat integration**: Commands interact with combat state and aggro systems

### Skill Integration

#### **Skill-Based Commands** (`skill.*.go` files)
- **Cast system**: Magic spell casting with proficiency scaling
- **Brawling skills**: Physical combat techniques (disarm, tackle, throw)
- **Utility skills**: Map creation, portal magic, inspection abilities
- **Protection skills**: Aid and defensive capabilities
- **Search skill**: Discovery of hidden objects, containers, exits, and creatures

#### **Skill Validation**
- **Level requirements**: Commands check character skill levels
- **Proficiency effects**: Higher skill levels improve command effectiveness
- **Training integration**: Skills can be improved through use and training

### Administrative System

#### **World Management**
- **Room editing**: Comprehensive room modification capabilities
- **Zone management**: Creating and managing game world zones (note: `zone set autoscale` was removed in Phase 21; mob difficulty is now per-mob via `statpool`)
- **Spawn control**: Managing mob and item spawning

#### **Player Administration**
- **Character modification**: Changing player stats, levels, and properties
- **Punishment system**: Muting, deafening, and other disciplinary actions
- **Server monitoring**: System status and performance monitoring

### Special Features

#### **Command Suggestions**
- **Fuzzy matching**: Suggesting similar commands for typos
- **Context-aware help**: Relevant command suggestions based on situation
- **Admin filtering**: Different suggestions for admin vs regular users

#### **Dialogue–Quest Integration** (`talk.go`, `ask.go`)
- **PlayerState construction**: `buildPlayerState(user)` creates a
  `dialogue.PlayerState` with callbacks for `HasQuest`, `HasItem`,
  `RemoveItem`, and `GiveQuest` — passed to all dialogue engine calls
  so NPC dialogue can be gated on quest progress and inventory
- **Quest context for LLM**: `buildQuestContext(user, mobId)` returns
  human-readable quest summaries injected into the LLM system prompt
  via `llm.ConversationContext.QuestContext`
- **Item consumption**: `requiresItem` on dialogue nodes removes the
  item from the player's backpack on activation (via `RemoveItem`)
- **Quest advancement**: `grantsQuest` fires `events.Quest` to
  advance quest state; the quest event handler processes rewards

#### **Scripting Integration**
- **JavaScript exposure**: Commands can be called from game scripts
- **Function export**: Command functions available to scripting system
- **Event-driven execution**: Commands can be triggered by game events

#### **Alias System**
- **Custom shortcuts**: Players can create command aliases
- **Macro support**: Complex command sequences through aliases
- **Personal customization**: Per-character alias storage

## Dependencies
- `internal/users`: User management and character data
- `internal/rooms`: Room system for location-based commands
- `internal/events`: Event system for command effects and logging
- `internal/mobs`: NPC interaction and combat
- `internal/items`: Item manipulation and inventory management
- `internal/skills`: Skill system integration
- `internal/spells`: Magic system integration
- `internal/buffs`: Status effect checking and application
- `internal/scripting`: JavaScript runtime integration

## Usage Patterns
- Commands follow consistent signature and return conventions
- State validation occurs before command execution
- Events are emitted for logging and system integration
- Permission checks prevent unauthorized access
- Error handling provides user feedback and system logging

## Testing
The package includes comprehensive testing for:
- Command parsing and argument handling
- Permission and access control
- State validation and restrictions
- Integration with other game systems
- Administrative functionality

## Architecture Benefits
- **Modular design**: Each command is self-contained and focused
- **Consistent interface**: All commands follow the same signature pattern
- **Extensible system**: New commands can be easily added to the registry
- **Permission control**: Granular access control for different user types
- **Event integration**: Commands seamlessly integrate with the game's event system

## Search Skill System

### Overview
The `search` command discovers hidden objects in rooms, including hidden containers, hidden nouns, secret exits, and hidden mobs. Uses Perception-based rolls with per-discovery granularity.

### Search Roll Formula
```
searchScore = dice.RollStat(Perception + SkillMultiplier(searchRank) * 25.0)
```

- **Perception**: Character's current Perception stat (~100 baseline)
- **SkillMultiplier**: Sqrt curve from current search rank to soft cap (rank 50)
- **dice.RollStat**: Applies global `RollSpread` factor for variance
- **searchScore**: Single roll covers all discoveries in one `search` command

### Tier Difficulty Targets
| Target | Hidden Type | Examples |
|--------|------------|----------|
| 125 | Secret exits, hidden containers | Doors behind tapestries, false walls |
| 135 | Stashed items, hidden creatures | Boxes under beds, camouflaged mobs |
| 175 | Hidden nouns | Faint carvings, ancient runes |

### Per-Discovery Rolls
- Each hidden object in the room gets compared against `searchScore` individually
- Players with `searchScore ≥ target` discover that specific object
- Multiple discoveries possible in one `search` if roll is high enough
- Each discovery shows unique flavor text and adds to discovery tracking

### Anti-Botting Protection
- **Progression guard**: Search skill only gains progression if at least one undiscovered object was rolled against
- If all objects in the room are already discovered, skill use doesn't trigger progression
- Prevents skill grinding on discovered-only rooms

### Related Commands
- **`track`**: Uses search skill formula to find hidden tracks
- **`forage`**: Uses search skill formula to gather hidden resources
- All three commands use the same unified search score calculation

This package serves as the primary interface between players and the game world, providing a rich and comprehensive command system that supports all aspects of gameplay from basic interaction to advanced administrative functions.