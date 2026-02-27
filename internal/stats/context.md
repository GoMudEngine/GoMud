# DOGMud Stats System Context

## Overview

The DOGMud stats system provides a character attribute framework with six core statistics that govern all character capabilities. Stats improve organically through use (skill-based progression) rather than through level-ups. The system features training point allocation, equipment modifications, and a diminishing returns system for balanced character development.

**DOGMud Differences from upstream GoMud:**
- Stats renamed: Speed → Dexterity, Smarts → Perception, Mysticism → Willpower, old Perception → Charisma
- Level-based progression removed — stats improve through use and training
- Mana removed — spells use Conviction instead
- Three resource pools: Health, Stamina, Conviction (no Mana)

## Architecture

The stats system is built around several key components:

### Core Components

**Six Primary Statistics:**
- **Strength**: Physical power affecting damage output and carrying capacity
- **Dexterity**: Agility and reflexes affecting combat speed and dodging
- **Perception**: Awareness affecting detection and observation abilities
- **Vitality**: Health and stamina affecting hit points and endurance
- **Willpower**: Mental fortitude affecting conviction capacity and spell effectiveness
- **Charisma**: Social influence affecting NPC interactions and charm abilities

**Multi-Layer Calculation System:**
- **Base Values**: Species starting statistics
- **Training Points**: Player-allocated stat improvements (gained through use-based progression)
- **Equipment Modifiers**: Temporary bonuses from gear and effects
- **Diminishing Returns**: Balanced scaling for high-end statistics

## Key Features

### 1. **Use-Based Progression**
- Stats improve through gameplay — using Strength-related actions improves Strength
- Exponential decay progression curve (easy early gains, harder at high values)
- Soft cap at virtual rank 150 for stats
- No level-up stat grants

### 2. **Species Differentiation**
- All players are Human with balanced base stats
- NPC species have varied base stats
- Base stats influence starting capabilities

### 3. **Flexible Modification System**
- **Equipment Bonuses**: Gear provides temporary stat improvements
- **Spell Effects**: Magic can enhance or reduce statistics
- **Buff Integration**: Status effects modify character capabilities
- **Dynamic Recalculation**: Stats update automatically when modifiers change

## Stat Structure

### Statistics Collection
```go
type Statistics struct {
    Strength   StatInfo // Physical power and damage
    Dexterity  StatInfo // Agility, reflexes, and combat speed
    Perception StatInfo // Awareness, detection, and observation
    Vitality   StatInfo // Health, stamina, and endurance
    Willpower  StatInfo // Mental fortitude and conviction capacity
    Charisma   StatInfo // Social influence and charm
}
```

### Individual Stat Information
```go
type StatInfo struct {
    Training int // Player-allocated training points (persistent)
    Value    int // Final calculated value (runtime)
    ValueAdj int // Adjusted value with diminishing returns (runtime)
    Base     int // Species base value (persistent)
    Mods     int // Equipment/effect modifiers (runtime)
}
```

## Stat Calculation System

### Diminishing Returns
```
Below StatSoftCapThreshold (105): no adjustment
105 to StatSoftCap (150): linear (stats earned = stats kept)
Above StatSoftCap (150):
  Adjusted = SoftCap + (Raw - SoftCap)^0.75 × Multiplier

Examples (with default multiplier 2.0):
- Value 120 → Adjusted 120 (below soft cap, linear)
- Value 150 → Adjusted 150 (at soft cap)
- Value 160 → Adjusted 161 (10^0.75 × 2 ≈ 11)
- Value 200 → Adjusted 188 (50^0.75 × 2 ≈ 38)
- Value 300 → Adjusted 236 (150^0.75 × 2 ≈ 86)

This allows meaningful progression up to the soft cap while
preventing runaway scaling beyond it.
```

## Stat Applications

### Combat Integration
```go
// Strength affects damage output
baseDamage := weaponDamage + (character.Stats.Strength.ValueAdj / 10)

// Dexterity affects hit chance and attack frequency
hitBonus := character.Stats.Dexterity.ValueAdj - target.Stats.Dexterity.ValueAdj

// Vitality affects health capacity
healthMax := baseHealth + (character.Stats.Vitality.ValueAdj * 2)
```

### Resource Pools
```go
// Three resource pools (no Mana):
// - Health: based on Vitality
// - Stamina: based on Vitality (physical endurance)
// - Conviction: based on Willpower + Charisma (mental/magical resource)
```

### Skill System Integration
```go
// Perception affects detection and awareness
detectionRange := baseRange + (character.Stats.Perception.ValueAdj / 20)

// Charisma affects social interactions
barteringBonus := character.Stats.Charisma.ValueAdj / 10
```

## Dependencies

- `math` - Mathematical functions for diminishing returns calculations
