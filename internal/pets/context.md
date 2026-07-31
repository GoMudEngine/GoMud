# Pet System Context

## Overview

The `internal/pets` package provides a comprehensive pet/companion system for the GoMud game engine. It manages pet ownership, feeding mechanics, combat abilities, inventory management, and stat modifications, creating engaging companion gameplay with care and maintenance requirements.

## Key Components

### Core Files
- **pets.go**: Complete pet management and functionality system
- **food.go**: Pet feeding and nutrition system

### Key Structures

#### Pet
```go
type Pet struct {
    Name          string            `yaml:"name,omitempty"`
    NameStyle     string            `yaml:"namestyle,omitempty"`
    Type          string            `yaml:"type"`
    Food          Food              `yaml:"food,omitempty"`
    LastMealRound uint8             `yaml:"lastmealround,omitempty"`
    Damage        items.Damage      `yaml:"damage,omitempty"`
    StatMods      statmods.StatMods `yaml:"statmods,omitempty"`
    BuffIds       []int             `yaml:"buffids,omitempty"`
    Capacity      int               `yaml:"capacity,omitempty"`
    Items         []items.Item      `yaml:"items,omitempty"`
}
```
Represents a player's pet with comprehensive features:
- **Name/NameStyle**: Custom pet naming with color pattern support
- **Type**: Pet species/breed identifier
- **Food/LastMealRound**: Feeding system with hunger tracking
- **Damage**: Combat capabilities and attack patterns
- **StatMods**: Stat bonuses provided to owner
- **BuffIds**: Permanent buffs granted to owner
- **Capacity/Items**: Pet inventory system for item carrying

#### Food
```go
type Food struct {
    // Food-related properties for pet nutrition
}
```
Manages pet feeding mechanics and nutritional requirements.

### Global State
- **petTypes**: `map[string]*Pet` - Registry of available pet types and specifications

## Core Functions

### Pet Management
- **GetPetCopy(petId string) Pet**: Returns copy of pet specification
  - Creates new pet instance from template
  - Used for spawning new pets of specific type
  - Returns empty Pet if type doesn't exist

- **GetPetSpec(petId string) Pet**: Returns pet type specification
  - Provides access to base pet configuration
  - Used for validation and reference information
  - Returns empty Pet if type doesn't exist

### Pet Properties
- **Exists() bool**: Checks if pet is valid (has type)
- **DisplayName() string**: Returns formatted pet name with color styling
  - Uses custom name if set, otherwise uses type name
  - Applies color patterns from NameStyle field
  - Falls back to default pet name color if no style specified

- **StatMod(statName string) int**: Returns stat modification value
  - Provides stat bonuses to pet owner
  - Integrates with character stat system
  - Supports various stat types and modifications

### Combat Integration
- **GetDiceRoll() (attacks, dCount, dSides, bonus int, buffOnCrit []int)**: Returns combat statistics
  - Provides pet's combat capabilities
  - Returns attack count, dice configuration, and damage bonus
  - Includes critical hit buff effects
  - Integrates with combat system for pet participation

### Inventory Management
- **StoreItem(i items.Item) bool**: Adds item to pet inventory
  - Validates pet capacity and item validity
  - Prevents storage if capacity is full or item is invalid
  - Returns success/failure status

- **RemoveItem(i items.Item) bool**: Removes specific item from pet inventory
  - Searches for exact item match
  - Removes first matching item found
  - Returns success/failure status

- **FindItem(itemName string) (items.Item, bool)**: Searches pet inventory
  - Supports fuzzy matching for item names
  - Returns best match or close match if found
  - Integrates with item search system

### Buff System
- **GetBuffs() []int**: Returns copy of pet's buff IDs
  - Provides permanent buffs granted to owner
  - Returns defensive copy to prevent modification
  - Integrates with character buff system

## Pet Features

### Customization System
- **Custom Naming**: Players can name their pets
- **Color Styling**: Custom color patterns for pet names
- **Type Variety**: Multiple pet types with different characteristics
- **Stat Specialization**: Different pets provide different stat bonuses

### Care and Maintenance
- **Feeding System**: Pets require regular feeding to maintain health
- **Hunger Tracking**: Last meal round tracking for feeding schedules
- **Nutrition Management**: Food system for pet care requirements
- **Health Consequences**: Potential penalties for neglecting pet care

### Combat Participation
- **Combat Stats**: Pets have attack capabilities and damage values
- **Critical Effects**: Special buffs applied on critical hits
- **Owner Support**: Pets provide combat assistance to owners
- **Stat Bonuses**: Combat-related stat modifications for owners

### Utility Functions
- **Item Carrying**: Pets can carry items for players
- **Inventory Extension**: Expands player storage capacity
- **Item Management**: Pet-based item storage and retrieval
- **Capacity Limits**: Balanced carrying capacity per pet type

## Dependencies

### Internal Dependencies
- `internal/colorpatterns`: For pet name color styling
- `internal/configs`: For configuration file paths
- `internal/fileloader`: For pet data loading and saving
- `internal/items`: For pet inventory and damage systems
- `internal/mudlog`: For logging pet operations
- `internal/statmods`: For pet stat modification system
- `internal/util`: For utility functions and file operations

### External Dependencies
- `gopkg.in/yaml.v2`: For YAML serialization of pet data

## Data Persistence

### File Operations
- **Save() error**: Saves pet data to YAML file
  - Creates pet-specific file based on pet name
  - Handles file creation and error reporting
  - Supports individual pet data persistence

- **Filename() string**: Generates filename for pet data
- **Filepath() string**: Returns relative file path for pet storage

### YAML Integration
- **Serialization**: Complete pet state saved to YAML format
- **Loading**: Pet types loaded from configuration files
- **Validation**: Data validation during load/save operations
- **Configuration**: Pet specifications defined in YAML files

## Usage Patterns

### Pet Creation and Management
```go
// Get pet template and create instance
petSpec := pets.GetPetCopy("wolf")
if petSpec.Exists() {
    // Customize pet
    petSpec.Name = "Fluffy"
    petSpec.NameStyle = ":rainbow"
}
```

### Pet Interaction
```go
// Check pet capabilities
statBonus := pet.StatMod("strength")
attacks, dice, sides, bonus, crits := pet.GetDiceRoll()

// Manage pet inventory
success := pet.StoreItem(someItem)
foundItem, exists := pet.FindItem("sword")
```

### Pet Care
```go
// Check if pet needs feeding
if pet.Food.NeedsFeeding() {
    // Feed the pet
    pet.Feed(foodItem)
    pet.LastMealRound = currentRound
}
```

## Integration Points

### Character System
- **Stat Bonuses**: Pets provide stat modifications to owners
- **Buff Application**: Permanent buffs granted through pet ownership
- **Combat Enhancement**: Pet-based combat improvements
- **Inventory Extension**: Additional storage capacity through pets

### Combat System
- **Pet Participation**: Pets can participate in combat alongside owners
- **Damage Calculation**: Pet damage integrated into combat calculations
- **Critical Effects**: Special effects triggered on pet critical hits
- **Tactical Options**: Pets provide additional combat strategies

### Item System
- **Inventory Management**: Pets serve as mobile storage containers
- **Item Interaction**: Pets can carry and manage items for players
- **Capacity Management**: Pet carrying capacity affects item distribution
- **Item Access**: Players can access pet inventories for item management

### Economy System
- **Pet Trading**: Pets as valuable tradeable assets
- **Care Costs**: Economic impact of pet feeding and maintenance
- **Breeding System**: Potential pet breeding and genetics (future enhancement)
- **Pet Services**: Pet-related services and businesses

## Performance Considerations

### Memory Management
- **Efficient Storage**: Compact pet data structures
- **Lazy Loading**: Pet types loaded on demand
- **Memory Pools**: Efficient allocation for pet instances
- **Garbage Collection**: Proper cleanup of unused pet data

### File I/O Optimization
- **Batch Operations**: Efficient saving of multiple pets
- **Incremental Updates**: Save only changed pet data
- **Compression**: Compressed storage for large pet datasets
- **Caching**: In-memory caching of frequently accessed pet data
