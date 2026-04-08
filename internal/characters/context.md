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

### Difficulty-Scaled Progression
`OnSkillUseScaled(skillName, userId, bonusMultiplier)` accepts a difficulty
bonus that flows into `CheckSkillProgression`. `OnSkillUse` delegates with
1.0 for backwards compatibility. Spell resolution passes
`1.0 + difficulty * SpellDifficultyProgressionScale`, craft completion passes
`1.0 + skillMinimum * CraftDifficultyProgressionScale`.

### Regen-Based Stat Progression
Every regen tick (every 3 rounds), each resource pool has a small chance to
trigger stat progression based on how depleted it is. This replaced the old
hard 25%-threshold `OnLowResource` system.

**Formula:** `chance = RegenProgressionBase × (1 - current/max) ^ RegenProgressionCurve`

**Config knobs:** `RegenProgressionBase` (default 0.01), `RegenProgressionCurve` (default 3.0)

**Resource → Stat Mappings:**
- Health → Vitality, Willpower (enduring injury toughens body + mind)
- Stamina → Strength, Vitality (exertion builds power + endurance)
- Conviction → Willpower, Charisma (mental strain sharpens will + presence)

The existing `StatProgressionMultipliers` still apply on top.
Mob progression uses `MobProgressionRate` as a multiplier.

**Key methods:**
- `OnRegenTick(current, max, relatedStats, userId)` — computes chance, calls CheckRegenProgression per stat
- `CheckRegenProgression(statName, userId, chance)` — applies mob gating, multipliers, rolls

### Equipment System (`worn.go`)
- **Equipment slots**: Weapon, Offhand, Head, Neck, Body, Belt, Gloves, Ring, Legs, Feet
- **Stat modifications**: Equipment provides stat bonuses aggregated across all slots
- **Item management**: Worn item tracking and validation

### Character States and Modifiers
- **Aggro system** (`aggro.go`): Combat targeting and threat management
- **Buffs integration**: Status effects that modify character capabilities
- **Cooldowns** (`cooldowns.go`): Time-based ability restrictions
- **Prone system** (Stage 7.5): Knockdown condition with stat-based recovery mechanics

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
- **Adjectives system**: Visual indicators for character states (sleeping, charmed, poisoned, prone, etc.)
- **Quest indicators**: Visual markers for quest-relevant NPCs

## Stage 7.5: Prone Condition System

### Prone State Fields
The prone condition is tracked via three fields in the `Character` struct:

```go
Prone                    bool   `yaml:"-"`  // Currently knocked down
ProneRoundsRemaining     int    `yaml:"-"`  // Minimum prone duration counter
RecoveryPenaltyThisRound bool   `yaml:"-"`  // Limits attacks to 1 during recovery attempt
```

**Field Descriptions:**
- `Prone`: Boolean flag indicating character is knocked to the ground
- `ProneRoundsRemaining`: Countdown for minimum prone duration (set to 2 when knocked down)
  - Must reach 0 before auto-recovery attempts begin
  - Decremented each round in combat hook processing
- `RecoveryPenaltyThisRound`: Flag set during failed recovery attempts
  - Reduces character's attacks to 1 for the current round
  - Represents struggling to stand while fighting
  - Cleared at end of each round tick

### Prone Adjective Display
The `GetAdjectives()` method in `character.go` includes "prone" when `c.Prone == true`:

```go
func (c *Character) GetAdjectives() []string {
    retAdjectives := []string{}

    if c.Health < 1 {
        retAdjectives = append(retAdjectives, `downed`)
    }
    if c.Prone {
        retAdjectives = append(retAdjectives, `prone`)
    }
    // ... other adjectives
}
```

This makes prone status visible in character descriptions and room listings.

### Automatic Recovery System
The `AttemptRecovery(statValue int)` method implements stat-based recovery with logarithmic scaling:

```go
func (c *Character) AttemptRecovery(statValue int) (bool, bool) {
    // Returns: (attemptMade, success)

    if !c.Prone {
        return false, false  // Not prone, no recovery needed
    }

    if c.ProneRoundsRemaining > 0 {
        c.ProneRoundsRemaining--
        c.RecoveryPenaltyThisRound = true
        return false, false  // Still in minimum duration, no messages
    }

    // Calculate recovery chance: min(90, 25 + 20 × ln(stat/25))
    chance := 25.0 + 20.0*math.Log(float64(statValue)/25.0)
    if chance > 90.0 {
        chance = 90.0  // Cap at 90% to keep some uncertainty
    }

    roll := dice.Roll(50, 15.0)
    success := roll.Value < chance

    if success {
        c.Prone = false
        c.ProneRoundsRemaining = 0
    } else {
        c.RecoveryPenaltyThisRound = true
    }

    return true, success  // Attempt made, return success status
}
```

**Recovery Formula Rationale:**
- Logarithmic scaling provides smooth progression without overpowering high stats
- 25 stat (low) = 25% chance, 100 stat (average) = 53%, 300 stat (high) = 75%
- 90% cap maintains tactical uncertainty even at extreme stats
- Generic `statValue` parameter allows future use for other conditions (grapple uses Strength, etc.)

**Integration with Combat Hooks:**
Called in `NewRound_UserRoundTick` and `NewRound_MobRoundTick`:

```go
// After cooldown ticks, attempt recovery if prone
if attemptMade, success := user.Character.AttemptRecovery(user.Character.Stats.Dexterity.ValueAdj); attemptMade {
    if success {
        user.SendText("You scramble to your feet!")
        room.SendText("<user> clambers to their feet in a rushed panic.", user.UserId)
    } else {
        user.SendText("You attempt to stand, but slip back down!")
        room.SendText("<user> attempts to stand, but slips and falls.", user.UserId)
    }
}

// Clear recovery penalty flag at end of round
user.Character.RecoveryPenaltyThisRound = false
```

### Cooldown System Usage
The cooldown system (`cooldowns.go`) is used for special combat moves:

**Special Move Cooldown:**
- Key: `"combat-special"`
- Duration: 5 rounds (config: `SpecialMoveCooldown`)
- Shared across bash, trip, and kick commands

**Usage Pattern in Commands:**
```go
// Check cooldown before executing special move
if !user.Character.Cooldowns.Try("combat-special", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
    user.SendText(fmt.Sprintf("You can't use special moves yet! (%d rounds remaining)",
        user.Character.Cooldowns.Get("combat-special")))
    return true, nil
}

// Execute special move...
```

**Cooldown Mechanics:**
- Stored in `Character.Cooldowns` map (map[string]int)
- Auto-decremented via `RoundTick()` called in combat hooks
- `Try(key, period)` checks if cooldown expired and resets if action performed
- `Get(key)` returns remaining rounds for display purposes

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

## Shop Inventory Decoupling (Living Economy)

Merchant NPCs separate trade inventory from character inventory:

- **`ShopInventory`** (in `internal/shops/`) is the live trade state — stock
  levels, dynamic prices, NPC gold for transactions, restock timers. This is
  what `buy`/`sell` commands interact with.
- **`Character.Shop`** (the legacy `[]ShopItem` slice) remains as template /
  seed data and a fallback for non-migrated merchants. It is NOT the live
  inventory.
- **`Character.Gold`** is the NPC's personal gold (loot on death). NPC gold
  for trade transactions is tracked in `ShopInventory.Gold`, not here.
- **`Character.Items`** (backpack) is NOT used for merchant trade stock.
  Crafter mobs do use the backpack transiently to hold raw materials between
  restock and craft, but finished goods go directly into `ShopInventory`.

When reading or writing merchant code, always distinguish between these three
gold/inventory sources to avoid double-counting or routing items to the
wrong pool.

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
