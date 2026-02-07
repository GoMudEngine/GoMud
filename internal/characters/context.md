# DOGMud Characters Package Context

## Overview
The `internal/characters` package is the core character system for DOGMud, handling both player characters (PCs) and non-player characters (NPCs/mobs). It provides a comprehensive character model with stats, equipment, skills, combat mechanics, and various character states.

**DOGMud Differences from upstream GoMud:**
- Level system disabled — progression is skill/stat-use-based
- Mana removed — spells use Conviction resource pool
- Three resource pools: Health, Stamina, Conviction
- Six stats renamed: Strength, Dexterity, Perception, Vitality, Willpower, Charisma
- Species system replaces races (all players are Human)
- 10 core DOG skills + 15 legacy GoMud skills coexist

## Key Components

### Core Character Structure (`character.go`)
- **Character struct**: The main character entity containing all character data
- **Character creation and management**: Factory functions and lifecycle management
- **Stat calculations**: Dynamic stat computation with buffs, equipment, and species modifiers
- **Skill-based progression**: Skills and stats improve through use (`progression.go`)
- **Persistence**: Character data serialization/deserialization

### Character Statistics System
- **Six core stats**: Strength, Dexterity, Perception, Vitality, Willpower, Charisma
- **Stat scaling**: Stats over 100 use `SQRT(overage)*2` formula for diminishing returns
- **Dynamic modifiers**: Equipment, buffs, and species bonuses affect final stats
- **Use-based improvement**: Stats improve organically through gameplay

### Skill System (`progression.go`)
- **Use-based progression**: Skills improve through gameplay use, not training points
- **Exponential decay curve**: ~50% chance at rank 0, ~2.5% at soft cap (rank 50)
- **Skill aliasing**: `skillNameMap` supports mapping legacy skill names to DOG equivalents
- **10 core DOG skills**: 5 combat (weapon-combat, unarmed-combat, ranged-combat, spellcasting, psionics) + 5 non-combat (first-aid, stealth, tracking, bartering, foraging)
- **15 legacy GoMud skills**: Still functional alongside DOG skills
- **Combat skill routing**: `GetCombatSkillTag()` selects weapon-appropriate skill

### Equipment System (`worn.go`)
- **Equipment slots**: Weapon, Offhand, Head, Neck, Body, Belt, Gloves, Ring, Legs, Feet
- **Stat modifications**: Equipment provides stat bonuses aggregated across all slots
- **Item management**: Worn item tracking and validation

### Character States and Modifiers
- **Alignment system** (`alignment.go`): Good/neutral/evil alignment with numeric values (-100 to +100)
- **Aggro system** (`aggro.go`): Combat targeting and threat management
- **Buffs integration**: Status effects that modify character capabilities
- **Cooldowns** (`cooldowns.go`): Time-based ability restrictions

### Resource Pools
- **Health**: Physical hitpoints, based on Vitality
- **Stamina**: Physical endurance, based on Vitality (used for movement and combat actions)
- **Conviction**: Mental/magical resource, based on Willpower + Charisma (used for spells)
- Mana has been removed entirely

### Combat and Interaction Systems
- **Kill/Death statistics** (`kdstats.go`): PvP and PvE combat tracking
- **Charm system** (`charminfo.go`): Mind control and pet mechanics
- **Mob mastery** (`mobmastery.go`): Character proficiency with specific creature types
- **Shop system** (`shop.go`): NPC merchant capabilities with restocking mechanics

### Character Presentation
- **Formatted names** (`formattedname.go`): Rich text rendering with adjectives and color coding
- **Adjectives system**: Visual indicators for character states (sleeping, charmed, poisoned, etc.)
- **Quest indicators**: Visual markers for quest-relevant NPCs

## Key Features

### Character Persistence
- YAML-based character data storage
- Automatic saving with configurable intervals
- Character creation timestamps and history tracking
- Room history for movement tracking

### Dynamic Stat System
- Base stats from species definitions
- Equipment stat modifications
- Buff/debuff effects
- Use-based stat improvement through gameplay
- Calculated maximums for Health, Stamina, and Conviction

### Social and Economic Systems
- Gold and banking system
- Player shops and merchant NPCs
- Clan membership support
- Pet ownership and management
- Quest progress tracking

## Dependencies
- `internal/stats`: Core statistics definitions
- `internal/items`: Item system integration
- `internal/buffs`: Status effect system
- `internal/species`: Character species definitions
- `internal/skills`: Skill system integration
- `internal/spells`: Magic system integration
- `internal/quests`: Quest system integration
- `internal/pets`: Pet system integration
- `internal/gametime`: Time-based mechanics
- `internal/colorpatterns`: Text formatting and colors
