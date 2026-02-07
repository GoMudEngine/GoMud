# DOGMud Combat System Context

## Overview

The DOGMud combat system provides comprehensive turn-based combat mechanics with support for player vs player, player vs mob, and mob vs mob encounters. It features skill-based damage calculations, dual wielding, critical hits, backstab mechanics, pet participation, alignment-based consequences, and detailed combat messaging with cross-room attack support.

**DOGMud Differences from upstream GoMud:**
- Combat uses skill ranks (weapon-combat, unarmed-combat, ranged-combat) instead of character Level
- No Level-based combat formulas — all calculations use stats and skills
- Weapon-aware skill routing: melee → weapon-combat, bows → ranged-combat, fists → unarmed-combat
- Stat names: Dexterity (was Speed), Perception (was Smarts), Willpower (was Mysticism)

## Architecture

The combat system is built around several key components:

### Core Components

**Combat Resolution Engine:**
- Turn-based combat with dexterity-based attack frequency
- Multi-attack system based on dexterity differentials
- Weapon-based damage calculations with species bonuses
- Defense reduction and damage mitigation
- Critical hit system with buff effects

**Attack Result System:**
- Comprehensive result tracking for damage, hits, and effects
- Multi-target messaging system for attacker, defender, and rooms
- Support for cross-room combat with directional messaging
- Buff application tracking for combat effects

**Combat Calculations (`calculations.go`):**
- Hit chance calculations based on dexterity statistics
- Critical hit probability based on stats and combat skill ranks
- Damage reduction through defense statistics
- Power ranking system for combat assessment
- Alignment change calculations for PvP consequences

## Combat Skill Resolution

### Weapon-Aware Skill Routing
```go
// GetCombatSkillTag() selects the appropriate DOG combat skill:
// - Equipped ranged weapon (bow/crossbow) → ranged-combat
// - Equipped melee weapon (sword/axe/etc.) → weapon-combat
// - No weapon or claws → unarmed-combat

// GetCombatSkillLevel() returns the rank of the appropriate skill:
// 1. Try the weapon-appropriate DOG skill
// 2. Fall back to legacy Brawling skill
// 3. Minimum return: 1
```

### Combat Types
- Player vs Mob combat with damage tracking
- Player vs Player combat with alignment consequences
- Mob vs Player combat with AI integration
- Mob vs Mob combat with charm attribution

## Key Features

### Advanced Combat Mechanics
- Dexterity-based multiple attacks per round
- Dual wielding with skill-based penalties
- Backstab mechanics with guaranteed critical hits
- Pet participation in combat (20% chance)
- Cross-room combat support with directional messaging

### Hit Calculation System
```go
// Hit chance based on dexterity statistics
// hitChance = 30 + (attackDex / (attackDex + defendDex)) * 70
// Clamped between 5% and 95%
```

### Critical Hit System
```go
// Critical hit probability uses combat skill ranks, not Level
// Base crit chance modified by:
// - Strength + Dexterity stats
// - Combat skill rank differential
// - Accuracy buff (doubles crit chance)
// - Blink buff on target (halves crit chance)
```

### Dual Wielding
- Skill level 2: 50% chance to use both weapons
- Skill level 3+: Always dual wield
- Claws (natural weapons): Always dual wield
- Hit penalty: 35% (25% at mastery level 4)

## Combat Messaging System
- Dynamic message selection based on damage percentage
- Token-based message customization ({source}, {target}, {weapon}, etc.)
- Separate messaging for same-room vs cross-room combat
- Critical hit and backstab message highlighting

## Power Ranking System
```
Combat assessment weights:
- Damage output: 40%
- Dexterity comparison: 30%
- Health comparison: 20%
- Defense comparison: 10%
```

## Dependencies

- `internal/characters` - Character stats, equipment, and abilities
- `internal/items` - Weapon specifications and combat messaging
- `internal/users` - Player character management and state
- `internal/mobs` - NPC character management and AI integration
- `internal/buffs` - Status effects that modify combat
- `internal/skills` - Skill system for combat skills and dual wielding
- `internal/species` - Species bonuses and unarmed combat specifications
- `internal/rooms` - Room management for cross-room combat
- `internal/util` - Dice rolling and random number generation
- `internal/configs` - Configuration for combat behavior and messaging
