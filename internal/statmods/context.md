# Stat Modifications System Context

## Overview

The `internal/statmods` package provides a centralized system for managing character statistic modifications in the DOGMud game engine. It defines standardized stat names, modification structures, and provides utilities for applying temporary and permanent stat changes from various sources like items, buffs, species bonuses, and skills.

**DOGMud note:** The internal stat key strings (e.g., `"speed"`, `"smarts"`, `"mysticism"`) are used as map keys in the engine and may not match the player-facing stat names (Dexterity, Perception, Willpower). The mapping is: speed→Dexterity, smarts→Perception, mysticism→Willpower, perception→Charisma. Mana-related keys are deprecated.

## Key Components

### Core Files
- **statmods.go**: Complete stat modification system and standardized stat name definitions

### Key Structures

#### StatMods
```go
type StatMods map[string]int
```
Core map-based structure for storing stat modifications:
- **Key**: Stat name as string identifier
- **Value**: Modification amount (positive for bonuses, negative for penalties)
- **Flexible**: Supports any stat name for extensibility
- **Efficient**: Direct map lookup for O(1) access performance

#### StatName
```go
type StatName string
```
Type-safe string wrapper for standardized stat name constants, ensuring consistency across the codebase.

## Standardized Stat Names

### Core Character Stats
- **Strength**: `"strength"` - Physical power affecting melee damage and carrying capacity
- **Speed/Dexterity**: `"speed"` - Agility and reaction time affecting initiative and dodge (displayed as "Dexterity")
- **Smarts/Perception**: `"smarts"` - Awareness affecting detection and observation (displayed as "Perception")
- **Vitality**: `"vitality"` - Constitution affecting health and resistance
- **Mysticism/Willpower**: `"mysticism"` - Mental fortitude affecting conviction and spell effectiveness (displayed as "Willpower")
- **Perception/Charisma**: `"perception"` - Social influence affecting NPC interactions (displayed as "Charisma")

### Derived Stats
- **HealthMax**: `"healthmax"` - Maximum health points
- **ManaMax**: `"manamax"` - DEPRECATED (Mana removed in DOGMud)
- **HealthRecovery**: `"healthrecovery"` - Health regeneration rate modifier
- **ManaRecovery**: `"manarecovery"` - DEPRECATED (Mana removed in DOGMud)

### Skill-Specific Stats
- **Tame**: `"tame"` - Animal taming skill effectiveness
- **Picklock**: `"picklock"` - Lock picking skill effectiveness
- **Casting**: `"casting"` - General spell casting effectiveness
- **CastingPrefix**: `"casting-"` - Prefix for school-specific casting bonuses

### Special Modifiers
- **XPScale**: `"xpscale"` - Experience gain multiplier
- **RacialBonusPrefix**: `"racial-bonus-"` - Prefix for racial stat bonuses

## Core Methods

### Stat Retrieval
- **Get(statName ...string) int**: Retrieves total modification for specified stat(s)
  - Supports single stat lookup: `statMods.Get("strength")`
  - Supports multiple stat aggregation: `statMods.Get("strength", "vitality")`
  - Returns sum of all requested stat modifications
  - Returns 0 for empty StatMods or non-existent stats
  - Thread-safe for concurrent read access

## Usage Patterns

### Basic Stat Modification
```go
// Create stat modifications
mods := statmods.StatMods{
    "strength":     5,
    "vitality":     3,
    "healthmax":    25,
    "manamax":      -10,
}

// Retrieve individual stat
strengthBonus := mods.Get("strength")  // Returns 5

// Retrieve multiple stats
physicalBonus := mods.Get("strength", "vitality")  // Returns 8
```

### Prefix-Based Modifications
```go
// School-specific casting bonuses
mods := statmods.StatMods{
    "casting-fire":    10,
    "casting-ice":     5,
    "casting-healing": 15,
}

// Racial bonuses
racialMods := statmods.StatMods{
    "racial-bonus-strength": 2,
    "racial-bonus-vitality": 1,
}
```

### Integration with Game Systems
```go
// Item-based stat modifications
item := items.Item{
    StatMods: statmods.StatMods{
        "strength": 3,
        "speed":    2,
    },
}

// Apply item bonuses
totalStrength := baseStrength + item.StatMods.Get("strength")
```

## Integration Points

### Item System
- **Equipment Bonuses**: Items provide stat modifications when equipped
- **Consumable Effects**: Temporary stat modifications from consumables
- **Enchantments**: Magical enhancements adding stat bonuses
- **Set Bonuses**: Multiple item combinations providing additional bonuses

### Buff System
- **Temporary Effects**: Buffs apply time-limited stat modifications
- **Permanent Buffs**: Long-term or permanent stat changes
- **Stacking Rules**: Multiple buff sources combining stat effects
- **Dispel Effects**: Removal of buff-based stat modifications

### Race System
- **Racial Bonuses**: Inherent stat modifications based on character race
- **Racial Penalties**: Balanced disadvantages for certain races
- **Cultural Variants**: Sub-racial stat variations
- **Evolution Effects**: Racial development affecting stats over time

### Character System
- **Base Stats**: Foundation stats modified by various sources
- **Derived Calculations**: Stats affecting secondary attributes
- **Level Progression**: Stat modifications from character advancement
- **Class Bonuses**: Class-specific stat modifications

### Spell System
- **Casting Effectiveness**: School-specific casting bonuses
- **Spell Power**: Stat modifications affecting spell damage/healing
- **Mana Efficiency**: Modifications to mana costs and regeneration
- **Spell Resistance**: Defensive stat modifications against magic

## Performance Considerations

### Efficient Access
- **O(1) Lookups**: Map-based storage provides constant-time access
- **Minimal Allocation**: Direct map operations without additional allocations
- **Batch Operations**: Multiple stat retrieval in single method call
- **Memory Efficiency**: Compact storage using native Go map

### Scalability
- **Unlimited Stats**: No hard limits on number of stat types
- **Dynamic Extension**: Easy addition of new stat types
- **Prefix Support**: Hierarchical stat organization through prefixes
- **Concurrent Safe**: Read operations safe for concurrent access

## Extensibility Features

### Dynamic Stat Names
- **Runtime Addition**: New stat types can be added without code changes
- **Plugin Support**: Plugins can define custom stat modifications
- **Modular Design**: Independent stat systems can coexist
- **Forward Compatibility**: New stats don't break existing systems

### Prefix System
- **Hierarchical Organization**: Related stats grouped by prefixes
- **Scalable Categories**: Easy addition of new stat categories
- **Flexible Matching**: Prefix-based stat retrieval and filtering
- **Namespace Management**: Prevents stat name conflicts

## Validation and Safety

### Data Integrity
- **Type Safety**: StatName type prevents string typos
- **Bounds Checking**: Validation of stat modification ranges
- **Consistency**: Standardized stat names across all systems
- **Error Prevention**: Compile-time checking of stat name constants

### Balance Protection
- **Modification Limits**: Configurable limits on stat modifications
- **Overflow Protection**: Prevention of integer overflow in calculations
- **Negative Handling**: Proper handling of negative stat modifications
- **Range Validation**: Ensuring stat values remain within valid ranges

## Future Enhancements

### Advanced Features
- **Percentage Modifiers**: Support for percentage-based stat modifications
- **Conditional Modifiers**: Stats that change based on conditions
- **Time-Based Decay**: Stat modifications that decrease over time
- **Scaling Modifiers**: Modifications that scale with character level

### Enhanced Integration
- **Database Storage**: Persistent storage of stat modifications
- **Network Synchronization**: Stat modification synchronization across clients
- **Real-time Updates**: Live updates of stat modifications
- **Audit Trails**: Tracking of stat modification sources and changes

### Administrative Tools
- **Stat Analysis**: Tools for analyzing stat distribution and balance
- **Modification Tracking**: Monitoring sources of stat modifications
- **Balance Reports**: Analysis of stat modification effectiveness
- **Debug Tools**: Tools for debugging stat calculation issues

### Performance Optimizations
- **Caching**: Cached calculation of frequently accessed stats
- **Batch Processing**: Optimized batch stat calculations
- **Memory Pooling**: Efficient memory management for stat operations
- **Lazy Evaluation**: Deferred calculation of complex stat combinations

## Security Considerations

### Exploit Prevention
- **Input Validation**: Validation of stat modification inputs
- **Range Limits**: Prevention of extreme stat modifications
- **Source Verification**: Verification of stat modification sources
- **Anti-Cheating**: Protection against stat modification exploits

### Data Protection
- **Integrity Checks**: Validation of stat modification data integrity
- **Tamper Detection**: Detection of unauthorized stat modifications
- **Rollback Capability**: Ability to revert problematic stat changes
- **Audit Logging**: Comprehensive logging of stat modifications

## Development Guidelines

### Best Practices
- **Consistent Naming**: Use standardized StatName constants
- **Documentation**: Clear documentation of custom stat purposes
- **Testing**: Comprehensive testing of stat modification logic
- **Performance**: Consider performance impact of stat calculations

### Code Organization
- **Centralized Definitions**: All stat names defined in statmods package
- **Modular Design**: Separate stat logic from application logic
- **Clear Interfaces**: Well-defined interfaces for stat operations
- **Error Handling**: Robust error handling for stat operations