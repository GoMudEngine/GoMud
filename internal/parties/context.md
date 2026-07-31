# Party System Context

## Overview

The `internal/parties` package provides a group management system for the GoMud game engine. It enables players to form parties for cooperative gameplay, with features including leadership, positioning, combat coordination, and invitation management.

## Key Components

### Core Files
- **parties.go**: Complete party management functionality

### Key Structures

#### Party
```go
type Party struct {
    LeaderUserId  int
    UserIds       []int
    InviteUserIds []int
    AutoAttackers []int
    Position      map[int]string
}
```
Represents a player party with the following features:
- **LeaderUserId**: User ID of the party leader with administrative privileges
- **UserIds**: List of all current party members including leader
- **InviteUserIds**: List of users with pending party invitations
- **AutoAttackers**: List of party members who automatically join combat
- **Position**: Map of user IDs to their tactical positions (front/middle/back)

### Global State
- **partyMap**: `map[int]*Party` - Maps leader user IDs to their party instances

## Core Functions

### Party Management
- **New(userId int) *Party**: Creates new party with specified leader
  - Returns nil if user already leads a party
  - Initializes party with leader as first member
  - Creates empty invitation and position maps
  - Registers party in global party map

- **Get(userId int) *Party**: Retrieves party by leader user ID
  - Returns party instance if found
  - Returns nil if no party exists for the user ID
  - Used for party lookup and validation

### Combat Integration
- **ChanceToBeTargetted(userId int) int**: Calculates targeting probability based on position
  - **Front Position**: Returns 2 (high chance to be targeted)
  - **Back Position**: Returns 0 (protected from targeting)
  - **Middle Position**: Returns 1 (moderate chance to be targeted)
  - Used by combat system for tactical positioning

### Position Management
- **GetRank(userId int) string**: Retrieves tactical position of party member
  - Returns position string: "front", "middle", or "back"
  - Used for combat calculations and tactical display
  - Determines combat role and targeting priority

## Party Features

### Leadership System
- **Single Leader**: One designated leader per party
- **Administrative Control**: Leader manages invitations and membership
- **Party Dissolution**: Party dissolves when leader leaves
- **Leadership Transfer**: Potential for leadership changes (implementation dependent)

### Membership Management
- **Dynamic Membership**: Members can join and leave during gameplay
- **Invitation System**: Formal invitation process for new members
- **Member Tracking**: Maintains list of all current party members
- **Size Limits**: Configurable party size restrictions (implementation dependent)

### Tactical Positioning
- **Three Positions**: Front, middle, and back tactical positions
- **Combat Impact**: Position affects targeting probability and combat effectiveness
- **Strategic Gameplay**: Encourages tactical thinking and coordination
- **Role Specialization**: Different positions suit different character builds

### Combat Coordination
- **Auto-Attack System**: Members can automatically join combat when party engages
- **Targeting Logic**: Position-based targeting system for balanced combat
- **Group Combat**: Coordinated combat actions
- **Protection Mechanics**: Back position provides protection from direct targeting

## Dependencies

### Internal Dependencies
- None directly - serves as foundational system for other components

### Integration Points
- **Combat System**: Position-based targeting and auto-attack coordination
- **User System**: Integration with user management for member validation
- **Progression System**: Skill progression through group combat
- **Command System**: Party-related commands for management and coordination

## Usage Patterns

### Party Creation
```go
// Create new party with leader
party := parties.New(leaderUserId)
if party == nil {
    // User already leads a party
    return errors.New("already leading a party")
}
```

### Party Lookup
```go
// Find party by leader ID
party := parties.Get(leaderUserId)
if party == nil {
    // No party found
    return errors.New("party not found")
}
```

### Combat Positioning
```go
// Check targeting chance for combat
targetChance := party.ChanceToBeTargetted(userId)
if targetChance > 0 {
    // Member can be targeted based on position
    // Apply combat targeting logic
}
```

### Position Management
```go
// Set member position
party.Position[userId] = "front"  // Place member in front position

// Check member position
position := party.GetRank(userId)
switch position {
case "front":
    // Handle front-line combat role
case "middle":
    // Handle support role
case "back":
    // Handle protected role (casters, healers)
}
```

## Integration Points

### Combat System
- **Targeting Logic**: Position-based enemy targeting calculations
- **Auto-Attack**: Automatic combat participation for designated members
- **Damage Distribution**: Position affects damage taken and dealt
- **Combat Roles**: Position determines combat effectiveness and specialization

### Experience System
- **Shared Experience**: Experience distribution among party members
- **Bonus Experience**: Group bonuses for party activities
- **Level Balancing**: Experience scaling based on party composition
- **Skill Development**: Group skill advancement opportunities

### Communication System
- **Party Chat**: Dedicated communication channels for party members
- **Status Updates**: Automatic status sharing among members
- **Coordination**: Command coordination and tactical communication
- **Notifications**: Party event notifications and alerts

### Quest System
- **Group Quests**: Quests requiring party cooperation
- **Shared Progress**: Quest progress shared among party members
- **Group Objectives**: Multi-player quest objectives and rewards
- **Coordination Requirements**: Quests requiring tactical positioning

## Performance Considerations

### Memory Efficiency
- **Lightweight Structure**: Minimal memory footprint per party
- **Efficient Lookups**: O(1) party lookup by leader ID
- **Compact Storage**: Simple data structures for fast access
- **Garbage Collection**: Clean party cleanup when disbanded

### Scalability
- **Concurrent Parties**: Support for multiple simultaneous parties
- **Member Limits**: Configurable limits prevent resource exhaustion
- **Efficient Updates**: Fast member addition/removal operations
- **State Synchronization**: Efficient state updates across party members

## Error Handling

### Graceful Degradation
- **Invalid Operations**: Handles invalid party operations gracefully
- **Missing Parties**: Safe handling of non-existent party references
- **Member Conflicts**: Resolves membership conflicts and edge cases
- **State Consistency**: Maintains consistent party state during errors

### Recovery Mechanisms
- **Automatic Cleanup**: Cleans up orphaned or invalid party data
- **State Repair**: Repairs inconsistent party states when detected
- **Rollback Support**: Ability to rollback problematic party changes
- **Logging**: Comprehensive logging for debugging and analysis
