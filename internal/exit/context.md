# Exit System Context

## Overview

The `internal/exit` package provides data structures for managing room exits in the GoMud game engine. It defines both permanent and temporary exit types, supporting features like secret passages, locks, exit messages, and magical portals.

## Key Components

### Core Files
- **exit.go**: Exit data structures and functionality

### Key Structures

#### RoomExit
```go
type RoomExit struct {
    RoomId       int
    Secret       bool          `yaml:"secret,omitempty"`
    MapDirection string        `yaml:"mapdirection,omitempty"`
    ExitMessage  string        `yaml:"exitmessage,omitempty"`
    Lock         gamelock.Lock `yaml:"lock,omitempty"`
    OneWay       bool          `yaml:"oneway,omitempty"`
}
```
Represents a permanent room exit with the following features:
- **RoomId**: Destination room identifier
- **Secret**: Hidden exits not visible in normal room descriptions
- **MapDirection**: Optional mapping direction for cartography systems
- **ExitMessage**: Custom message displayed before traversing the exit
- **Lock**: Locking mechanism with difficulty-based security
- **OneWay**: Marks an intentional one-way spatial exit (skips reciprocity check)

#### TemporaryRoomExit
```go
type TemporaryRoomExit struct {
    RoomId       int
    Title        string
    UserId       int
    SpawnedRound uint64 `yaml:"-"`
    Expires      string
}
```
Represents temporary exits like magical portals:
- **RoomId**: Destination room identifier
- **Title**: Special display name for the exit
- **UserId**: Creator of the temporary exit
- **SpawnedRound**: Creation time (not persisted)
- **Expires**: Expiration time specification

### Key Methods

#### RoomExit Methods
- **HasLock() bool**: Determines if the exit has an active lock
  - Returns true if lock difficulty is greater than 0
  - Used for exit interaction validation

## Dependencies

### Internal Dependencies
- `internal/gamelock`: For lock mechanism implementation

### External Dependencies
- YAML serialization support through struct tags

## Exit Features

### Permanent Exits
- **Standard Navigation**: Basic room-to-room movement
- **Secret Passages**: Hidden exits requiring discovery
- **Locked Doors**: Exits requiring keys or lockpicking
- **Custom Messages**: Special flavor text for unique exits
- **Mapping Integration**: Direction information for map generation

### Temporary Exits
- **Magical Portals**: Player-created transportation methods
- **Time-Limited**: Automatic expiration and cleanup
- **User Attribution**: Tracking of portal creators
- **Custom Titles**: Descriptive names for special exits

### Lock System Integration
- **Difficulty-Based**: Locks with varying difficulty levels
- **Key Support**: Integration with game lock mechanisms
- **Skill Checks**: Lockpicking and key usage validation

## Usage Patterns

### Creating Permanent Exits
```go
exit := RoomExit{
    RoomId:       123,
    Secret:       false,
    MapDirection: "north",
    ExitMessage:  "You step through the ancient doorway...",
    Lock:         gamelock.Lock{Difficulty: 5},
}
```

### Creating Temporary Exits
```go
portal := TemporaryRoomExit{
    RoomId:       456,
    Title:        "magical portal of Chuckles",
    UserId:       789,
    SpawnedRound: currentRound,
    Expires:      "5 minutes",
}
```

### Lock Checking
```go
if exit.HasLock() {
    // Handle locked exit logic
    // Check for keys, attempt lockpicking, etc.
}
```

## Integration Points

### Room System Integration
- **Exit Lists**: Rooms contain collections of RoomExit objects
- **Navigation**: Core movement system uses exit definitions
- **Discovery**: Secret exit revelation mechanics

### Game Lock System
- **Lock Validation**: Integration with key and lockpicking systems
- **Difficulty Scaling**: Lock difficulty affects success rates
- **Security Mechanics**: Prevents unauthorized access

### Mapping System
- **Direction Mapping**: MapDirection field supports cartography
- **Exit Visualization**: Map generation uses exit information
- **Navigation Aids**: Player mapping tools integration

### Temporal Systems
- **Round Tracking**: Temporary exits track creation time
- **Expiration**: Automatic cleanup of expired portals
- **Duration Management**: Flexible expiration time formats

## Data Persistence

### YAML Serialization
- **Permanent Exits**: Full YAML serialization support
- **Optional Fields**: Uses `omitempty` for clean configuration files
- **Temporary Exclusions**: SpawnedRound excluded from persistence

### Configuration Integration
- **Room Definitions**: Exits defined in room YAML files
- **Dynamic Loading**: Support for runtime exit modification
- **Backup and Restore**: Standard configuration management

## Exit Types and Behaviors

### Standard Exits
- **Bidirectional**: Can be configured in both directions
- **Unidirectional**: One-way passages and drops
- **Conditional**: Exits that appear/disappear based on conditions

### Special Exits
- **Secret Passages**: Hidden until discovered
- **Locked Doors**: Require keys or skills to open
- **Magical Portals**: Temporary transportation methods
- **Conditional Exits**: State-dependent availability

### Exit Messages
- **Flavor Text**: Immersive descriptions for special exits
- **Delay Mechanics**: Messages followed by traversal delays
- **Atmospheric Enhancement**: Rich storytelling through exit descriptions

## Security and Access Control

### Lock Mechanisms
- **Difficulty Levels**: Numeric difficulty for lockpicking
- **Key Requirements**: Integration with item-based keys
- **Skill Validation**: Player skill checks for access

### Secret Exits
- **Discovery Mechanics**: Hidden until found through exploration
- **Search Integration**: Player search commands reveal secrets
- **Progressive Revelation**: Gradual discovery of hidden areas

## Performance Considerations

### Memory Efficiency
- **Lightweight Structures**: Minimal memory footprint per exit
- **Optional Fields**: Reduced storage for unused features
- **Efficient Lookups**: Fast exit resolution for navigation

### Temporary Exit Management
- **Automatic Cleanup**: Expired portals removed automatically
- **Memory Management**: Prevents accumulation of old portals
- **Round-Based Tracking**: Efficient temporal management

## Future Enhancements

### Potential Features
- **Conditional Exits**: Exits that appear based on game state
- **Animated Exits**: Time-based exit state changes
- **Group Restrictions**: Exits limited to certain player groups
- **Skill Requirements**: Exits requiring specific abilities
- **Dynamic Destinations**: Exits with variable destinations

### Advanced Lock Features
- **Multiple Keys**: Exits requiring multiple keys
- **Combination Locks**: Number-based security systems
- **Biometric Locks**: Character-specific access control
- **Time Locks**: Exits that lock/unlock on schedules

### Enhanced Temporary Exits
- **Portal Networks**: Connected portal systems
- **Conditional Expiration**: Context-based portal duration
- **Portal Stability**: Varying reliability of magical exits
- **Resource Costs**: Magic or energy requirements for portals