# Mob Commands Package Context

## Overview
The `internal/mobcommands` package implements the AI command system for non-player characters (NPCs/mobs) in GoMud. It provides intelligent behavior patterns, combat AI, social interactions, and autonomous decision-making capabilities that bring the game world to life through sophisticated NPC behaviors.

## Key Components

### Command Architecture (`mobcommands.go`)
- **MobCommand function signature**: Standardized interface for all mob AI commands
  ```go
  type MobCommand func(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error)
  ```
- **CommandAccess structure**: Defines command availability and restrictions for mobs
- **Command registry**: Central mapping of AI command names to implementations
- **Autonomous execution**: Commands executed by AI logic rather than player input

### AI Behavior Categories

#### **Combat Intelligence**
- **Threat assessment**: `lookfortrouble` - Advanced hostility detection and target selection
- **Combat actions**: `attack`, `backstab`, `shoot`, `throw` - Offensive capabilities
- **Special moves**: `bash`, `trip`, `kick`, `grapple`, `hamstring` - selected
  by `combat.ChooseSpecialMove` and dispatched here via `mob.Command(name)`.
  Each is anatomy-gated by the actor's species `body_parts` (grapple needs
  `arms`, trip/kick need `legs`, hamstring needs `legs` + a `Bite`/`Claws`
  natural attack; bash needs a shield-or-`NaturalBash` plus arms-or-`NaturalBash`).
  The dedicated `bite` command was **retired** — biting is now the basic
  attack for fanged species (see `internal/combat` Natural-Attack Subtype
  Resolution). `toxic-bite` (mutation) is unaffected.
- **Beast moves (Phase 3)**: `rake` (clawed), `maul` (fanged), `pounce`
  (quadruped predator, not grappling), `gore` (horned), `drain` (lifedrain
  species flag), `throttle` (fanged) — registered as both mob and user
  commands for full parity. Selected by `combat.ChooseSpecialMove` via the
  new `predator`, `ambush_predator`, and `brute` AI profiles (also weighted
  in `default`/`aggressive`). Each is gated by the exported predicates
  `combat.SpeciesIsFanged` / `SpeciesIsClawed` / `SpeciesIsHorned` /
  `SpeciesHasLifeDrain` / `SpeciesIsQuadrupedPredator` at three sync points:
  `CanUse*` in `ai.go`, `CommandIsReady`, and the action entry. Drift rows
  in `command_readiness_drift_test.go` keep all three in sync.
- **Tactical support**: `callforhelp` - Coordinated group combat behaviors
- **Self-preservation**: Retreat and defensive behaviors

#### **Social and Communication AI**
- **Conversation system**: `converse` - Dynamic NPC-to-NPC dialogue
- **Player interaction**: `sayto`, `say`, `shout` - Contextual communication
- **Emotional expression**: `emote` - Rich behavioral expressions
- **Quest integration**: `givequest` - Dynamic quest assignment

#### **Autonomous Movement**
- **Wandering behavior**: `wander` - Intelligent exploration with constraints
- **Pathfinding**: `pathto` - Goal-directed navigation
- **Zone awareness**: Movement restricted by zone boundaries
- **Home behavior**: Return-to-home mechanics for territorial mobs

#### **Item and Resource Management**
- **Inventory control**: `get`, `drop`, `put`, `give` - Intelligent item handling
- **Equipment management**: `equip`, `remove`, `gearup` - Automated gear optimization
- **Resource consumption**: `eat`, `drink` - Survival behaviors
- **Alchemy and crafting**: `alchemy` - Automated production behaviors

#### **Support and Utility Behaviors**
- **Healing assistance**: `aid`, `lookforaid` - Medical support AI
- **Environmental interaction**: `look`, `show` - Awareness and demonstration
- **Magic usage**: `cast`, `portal` - Spellcasting AI with tactical considerations
- **Stealth operations**: `sneak` - Covert movement capabilities

### Advanced AI Features

#### **Hostility and Threat Management** (`lookfortrouble.go`)
- **Multi-factor threat assessment**: Race hatred, alignment conflicts, group hostilities
- **Player party awareness**: Sophisticated targeting that considers party dynamics
- **Charmed mob handling**: Different behaviors for player-controlled NPCs
- **Escalation prevention**: Boredom counters to prevent endless aggression

#### **Coordinated Group Behaviors** (`callforhelp.go`)
- **Range-based assistance**: Configurable help radius for tactical support
- **Selective recruitment**: Target-specific ally summoning
- **Communication integration**: Emotive calls for help with custom messages
- **Strategic positioning**: Intelligent movement for optimal combat support

#### **Intelligent Navigation** (`wander.go`)
- **Goal-oriented wandering**: Seeking loot, players, or specific objectives
- **Territorial constraints**: Respecting home zones and wander limits
- **Return-home logic**: Automatic navigation back to spawn points
- **Environmental awareness**: Zone-restricted movement patterns

#### **Dynamic Conversations** (`converse.go`)
- **Context-aware dialogue**: Conversations based on mob types and situations
- **Multi-participant support**: Complex NPC-to-NPC interaction chains
- **State management**: Conversation tracking to prevent conflicts
- **Scripted flexibility**: Support for forced conversation scenarios

### AI Decision Making

#### **Behavioral Prioritization**
- **Combat override**: Combat takes precedence over other activities
- **State-based decisions**: Different behaviors based on health, buffs, and conditions
- **Environmental factors**: Room conditions influence behavior choices
- **Social awareness**: Presence of players and other mobs affects decisions

#### **Intelligent Targeting**
- **Threat assessment algorithms**: Complex scoring for target selection
- **Party dynamics**: Understanding player group relationships
- **Alignment considerations**: Moral and ethical targeting preferences
- **Race-based hostilities**: Cultural and species-based conflicts

#### **Resource Management**
- **Inventory optimization**: Automatic equipment and item management
- **Survival priorities**: Food, drink, and healing behaviors
- **Economic behaviors**: Trading and resource acquisition patterns
- **Crafting automation**: Intelligent use of alchemy and creation skills

### Integration with Game Systems

#### **Character System Integration**
- **Buff awareness**: Behaviors modified by active status effects
- **Skill utilization**: AI uses mob skills and abilities appropriately
- **Health monitoring**: Behavior changes based on health status
- **Charm handling**: Different behaviors for player-controlled mobs

#### **Combat System Integration**
- **Aggro management**: Sophisticated threat and targeting systems
- **Tactical awareness**: Understanding of combat mechanics and timing
- **Group coordination**: Multi-mob combat strategies
- **Defensive behaviors**: Retreat and evasion capabilities

#### **Quest System Integration**
- **Dynamic quest giving**: NPCs can assign quests based on conditions
- **Progress awareness**: Mobs understand player quest states
- **Reward distribution**: Intelligent quest completion handling
- **Story integration**: Behaviors that support narrative elements

### Performance and Efficiency

#### **Optimized Execution**
- **Conditional processing**: Commands only execute when relevant
- **State caching**: Efficient tracking of mob states and conditions
- **Range limitations**: Bounded search areas for performance
- **Selective activation**: Behaviors triggered only when needed

#### **Memory Management**
- **Lightweight state tracking**: Minimal memory footprint for AI state
- **Efficient pathfinding**: Optimized navigation algorithms
- **Conversation management**: Proper cleanup of dialogue states
- **Resource pooling**: Shared resources for common AI operations

## Dependencies
- `internal/mobs`: Core mob management and state
- `internal/rooms`: Room system for spatial awareness
- `internal/characters`: Character system for mob properties
- `internal/users`: Player interaction and targeting
- `internal/buffs`: Status effect awareness
- `internal/conversations`: Dynamic dialogue system
- `internal/mapper`: Pathfinding and navigation
- `internal/parties`: Player group dynamics understanding

## Usage Patterns
- Commands executed autonomously by mob AI systems
- State validation ensures appropriate behavior selection
- Environmental awareness drives decision making
- Social dynamics influence interaction patterns
- Combat priorities override other behaviors

## Architecture Benefits
- **Autonomous intelligence**: Mobs behave independently without constant scripting
- **Scalable AI**: System supports complex behaviors across many NPCs simultaneously
- **Flexible behaviors**: Easy to add new AI patterns and capabilities
- **Performance optimized**: Efficient execution suitable for large-scale deployment
- **Integrated design**: Seamless interaction with all game systems

This package transforms static NPCs into dynamic, intelligent entities that create a living, breathing game world through sophisticated AI behaviors and decision-making systems.