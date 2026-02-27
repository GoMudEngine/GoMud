# Rooms Package Context

## Overview
The `internal/rooms` package is the core world management system for GoMud, handling all aspects of game world rooms, zones, and spatial relationships. It provides comprehensive room functionality including dynamic loading/unloading, ephemeral room creation, biome management, and complex room state tracking.

## Key Components

### Core Room Structure (`rooms.go`)
- **Room struct**: The main room entity containing all room data and state
- **Room properties**: Title, description, exits, items, NPCs, players, and environmental settings
- **Special room types**: Banks, storage rooms, character creation rooms, PvP areas
- **Dynamic state**: Player/mob tracking, visitor history, temporary data storage
- **Room features**: Containers, signs, skill training areas, spawn points

### Room Management System (`roommanager.go`)
- **RoomManager**: Singleton manager for all room operations and caching
- **Memory management**: Automatic loading/unloading of rooms based on occupancy
- **Zone management**: Organizing rooms into logical zones with metadata
- **File system integration**: Room data persistence and template loading
- **Cache optimization**: Room file path caching and efficient lookups

### Room Details and Presentation (`roomdetails.go`)
- **RoomTemplateDetails**: Rich room information for client rendering
- **Dynamic content**: Visible players, mobs, corpses, and exits
- **Environmental context**: Day/night cycles, lighting, biome effects
- **User-specific views**: Personalized room information based on character state
- **Room alerts**: Special notifications for banks, training, storage, etc.

### Biome System (`biomes.go`)
- **BiomeInfo**: Environmental definitions affecting room behavior
- **Lighting system**: Dark areas, lit areas, and visibility mechanics
- **Environmental effects**: Symbols, descriptions, and special properties
- **Item requirements**: Biomes that require specific items to navigate safely
- **Dynamic loading**: File-based biome definitions with validation

### Spawn Management (`spawninfo.go`)
- **SpawnInfo**: Comprehensive mob and item spawning system
- **Spawn configuration**: Mob templates, items, gold, and containers
- **Respawn mechanics**: Time-based respawning with configurable rates
- **Spawn customization**: Level modifications, hostility, scripting overrides
- **Quest integration**: Quest flags and buff assignments for spawned entities

### Container System (`container.go`)
- **Container**: In-room storage with locking mechanisms
- **Item management**: Adding, removing, and searching container contents
- **Lock system**: Difficulty-based locks requiring skills to open
- **Recipe system**: Crafting recipes that trigger when ingredients are present
- **Temporary containers**: Time-limited containers that despawn automatically

### Ephemeral Rooms (`ephemeral.go`)
- **Dynamic room creation**: Runtime creation of temporary room copies
- **Chunk management**: Efficient allocation of ephemeral room ID ranges
- **Memory optimization**: Automatic cleanup when rooms are no longer needed
- **Zone duplication**: Creating temporary copies of entire zones
- **ID mapping**: Tracking relationships between original and ephemeral rooms

## Key Features

### Dynamic World Management
- **Memory efficiency**: Rooms load/unload based on player presence
- **Visitor tracking**: History of who has visited rooms and when
- **State persistence**: Automatic saving of room changes and contents
- **Zone organization**: Logical grouping of related rooms

### Environmental Systems
- **Biome integration**: Environmental effects on room behavior and appearance
- **Day/night cycles**: Time-based lighting and atmospheric changes
- **Weather integration**: Biome-based weather effects and descriptions
- **Lighting mechanics**: Dark rooms, light sources, and visibility

### Interactive Elements
- **Containers**: Lockable storage with crafting recipe support
- **Signs**: Player-created messages and room annotations
- **Skill training**: Designated areas for character skill development
- **Special services**: Banking, storage, and character management rooms

### Spawn and Population
- **Flexible spawning**: Mobs, items, and gold with complex configuration
- **Respawn timing**: Configurable respawn rates and conditions
- **Population limits**: Preventing overcrowding through spawn management
- **Per-mob stat pools**: Mob difficulty set via `statpool` in mob YAML or per-spawn `statpool`/`statpoolmod` in room spawn info (zone-level autoscaling was removed in Phase 21)

### Performance Optimization
- **Chunk-based ephemeral rooms**: Efficient temporary room management
- **Lazy loading**: Rooms load only when needed
- **Memory cleanup**: Automatic removal of unused rooms and data
- **Cache management**: File path caching and lookup optimization

## Dependencies
- `internal/characters`: Character and mob management
- `internal/items`: Item system integration
- `internal/mobs`: NPC spawning and management
- `internal/exit`: Room connection and movement system
- `internal/gametime`: Time-based mechanics and scheduling
- `internal/mutators`: Room effect modifiers
- `internal/buffs`: Status effects in rooms
- `internal/configs`: Configuration management
- `internal/fileloader`: Data file loading system

## Usage Patterns
- Room loading through manager functions with automatic caching
- Player/mob tracking through room occupancy methods
- Dynamic content through spawn system and container management
- Environmental effects through biome integration
- Temporary content through ephemeral room system

## Testing
Comprehensive test coverage in `*_test.go` files covering:
- Room loading and caching mechanisms
- Spawn system functionality
- Container and lock mechanics
- Ephemeral room creation and cleanup
- Visitor tracking and room state management

## Special Considerations
- **Memory management**: Rooms automatically unload when empty to conserve memory
- **Ephemeral limits**: Maximum of 100 chunks with 250 rooms each for temporary content
- **Thread safety**: Room operations are designed for concurrent access
- **Data persistence**: Room changes are automatically saved to maintain world state

This package serves as the foundation for the entire game world, providing a rich and dynamic environment system that supports complex gameplay mechanics while maintaining optimal performance through intelligent memory management.