# Clans System Context

## Overview

The `internal/clans` package provides data structures and types for a clan/guild system in the GoMud game engine. It defines the organizational structure for player groups, including membership management, ranks, applications, and donation tracking.

## Key Components

### Core Files
- **clans.go**: Clan data structures and type definitions

### Key Structures

#### ClanInfo
```go
type ClanInfo struct {
    Zone         string       `json:"zone"`         // Zone the clan controls
    ClanTag      string       `json:"clantag"`      // Abbreviated clan name (up to 4 chars)
    ClanName     string       `json:"clanname"`     // Full clan name
    Upkeep       int          `json:"upkeep"`       // Daily cost in gold
    MemberUpkeep int          `json:"memberupkeep"` // Daily gold cost per member
    Members      []ClanMember `json:"members"`      // Current clan members
    Applications []ClanMember `json:"applications"` // Pending applications
    Donations    []ClanMember `json:"donations"`    // Donation history
}
```
Main clan structure containing all clan-related information including territory, finances, and membership.

#### ClanMember
```go
type ClanMember struct {
    UserId        int       `json:"userid"`        // User ID
    CharacterName string    `json:"charactername"` // Character name
    Joined        time.Time `json:"joined"`        // Join date/time
    Rank          ClanRank  `json:"rank"`          // Member rank
}
```
Represents individual clan members with their metadata and rank information.

#### Donation
```go
type Donation struct {
    UserId int        `json:"userid"` // Donor user ID
    Gold   int        `json:"gold"`   // Gold amount donated
    Item   items.Item `json:"item"`   // Item donated
    Date   time.Time  `json:"date"`   // Donation timestamp
}
```
Tracks individual donations to the clan treasury.

### Clan Rank System

#### ClanRank Type
```go
type ClanRank string
```

#### Rank Levels
- **ClanRankMember** (`"member"`): Normal members with no special privileges
- **ClanRankLieutenant** (`"lieutenant"`): Can accept applications
- **ClanRankLeader** (`"leader"`): Full privileges - invite, kick, accept applications, promote members

## Dependencies

### Internal Dependencies
- `internal/items`: For donation item tracking
- Standard library: `time` for timestamp management

### External Dependencies
- JSON serialization support through struct tags

## Clan Management Features

### Territory Control
- **Zone Ownership**: Clans can control specific game zones
- **Territorial Benefits**: Implied benefits from zone control

### Financial System
- **Base Upkeep**: Daily gold cost to maintain clan
- **Member Upkeep**: Additional cost per clan member
- **Automatic Disbanding**: Clans disband if upkeep isn't paid
- **Donation Tracking**: Members can donate gold and items

### Membership Management
- **Application System**: Players can apply to join clans
- **Rank-Based Permissions**: Different ranks have different privileges
- **Member History**: Join dates and rank progression tracking

### Permission Structure
- **Members**: Basic clan membership, no administrative privileges
- **Lieutenants**: Can accept new member applications
- **Leaders**: Full administrative control over clan operations

## Data Persistence

### JSON Serialization
- All structures use JSON tags for data persistence
- Compatible with standard Go JSON marshaling/unmarshaling
- Supports clan data storage and retrieval

### Time Tracking
- Member join dates for seniority tracking
- Donation timestamps for financial history
- Potential for upkeep payment scheduling

## Integration Points

### Game World Integration
- **Zone Control**: Links to game world territory system
- **Item System**: Integration with item donation mechanics
- **User System**: Links to player character management

### Economic System
- **Gold Management**: Integration with game economy
- **Upkeep Costs**: Daily financial obligations
- **Resource Sharing**: Clan treasury and donation system

### Social Features
- **Group Identity**: Clan tags and names for player identification
- **Hierarchy**: Rank-based social structure
- **Collaboration**: Shared resources and territory

## Usage Patterns

### Clan Creation
```go
clan := ClanInfo{
    Zone:         "frostfang",
    ClanTag:      "QC",
    ClanName:     "Questing Cajuns",
    Upkeep:       100,
    MemberUpkeep: 10,
    Members:      []ClanMember{},
    Applications: []ClanMember{},
    Donations:    []ClanMember{},
}
```

### Member Management
```go
newMember := ClanMember{
    UserId:        123,
    CharacterName: "PlayerName",
    Joined:        time.Now(),
    Rank:          ClanRankMember,
}
clan.Members = append(clan.Members, newMember)
```

### Donation Tracking
```go
donation := Donation{
    UserId: 123,
    Gold:   500,
    Item:   someItem,
    Date:   time.Now(),
}
```

## Design Considerations

### Scalability
- Structure supports multiple clans
- Efficient member lookup capabilities
- Manageable data size for JSON persistence

### Flexibility
- Extensible rank system
- Configurable upkeep costs
- Support for various donation types

### Data Integrity
- Time-based tracking for audit trails
- Clear ownership and permission models
- Structured financial tracking
