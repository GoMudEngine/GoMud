# DOGMud Combat System Context

## Overview

The DOGMud combat system provides comprehensive turn-based combat mechanics with support for player vs player, player vs mob, and mob vs mob encounters. It features skill-based damage calculations, layered defenses (dodge/parry/block), dual wielding, critical hits, special combat moves with knockdown mechanics, prone condition system, backstab mechanics, pet participation, alignment-based consequences, and detailed combat messaging with cross-room attack support.

**DOGMud Differences from upstream GoMud:**
- Combat uses skill ranks (weapon-combat, unarmed-combat, ranged-combat) instead of character Level
- No Level-based combat formulas — all calculations use stats and skills
- Weapon-aware skill routing: melee → weapon-combat, bows → ranged-combat, fists → unarmed-combat
- Stat names: Dexterity (was Speed), Perception (was Smarts), Willpower (was Mysticism)
- Layered defense system with stamina costs (dodge/parry/block)
- Prone condition with combat modifiers and stat-based recovery
- Special combat moves (bash/trip/kick) with shared cooldowns
- Target switching mid-combat

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

## Stage 7.5: Tactical Combat Enhancements

### Layered Defense System
Combat defense uses a three-tier system where attacks must overcome multiple defense layers:

**Defense Order (checked sequentially until one succeeds):**
1. **Dodge** - Unarmed Combat + Dexterity, costs 2 stamina, always available
   - Reduced by heavy armor and encumbrance
   - -50% effectiveness when prone
2. **Parry** - Weapon Combat + weapon parry rating, costs 4 stamina, requires weapon
   - Two-handed weapons get bonus to parry
   - Can parry with either hand when dual wielding
   - -30% effectiveness when prone
3. **Block** - Weapon Combat + shield block rating, costs 5 stamina, requires shield
   - Most stamina-intensive but highly effective
   - -20% effectiveness when prone (shield still works from ground)

**Stamina Costs:**
- All defense attempts cost stamina even on failure
- Low stamina (<50% of max) imposes attack and defense penalties
- Defense costs configured in `config.gameplay.yaml` (DodgeStaminaCost, ParryStaminaCost, BlockStaminaCost)

**Implementation:**
- Defense calculations in `combat.go` lines 517-570
- Stamina costs applied regardless of defense success
- First successful defense stops attack, remaining defenses not attempted

### Prone Condition System
Characters can be knocked to the ground by special combat moves (bash/trip/kick), applying severe combat penalties:

**Prone State:**
- Tracked via `Character.Prone` boolean field
- Minimum prone duration: 2 rounds (`Character.ProneRoundsRemaining`)
- Visual indicator: "prone" adjective added via `GetAdjectives()`

**Combat Modifiers (applied in `combat.go`):**
- Attack penalty: -30 to attack score
- Damage penalty: ×0.80 (20% reduction)
- Vulnerability: Attackers get +20 to hit prone targets
- Dodge penalty: ×0.50 (50% reduction)
- Parry penalty: ×0.70 (30% reduction)
- Block penalty: ×0.80 (20% reduction)

**Behavioral Restrictions:**
- Cannot flee from combat (enforced in flee.go)
- Cannot move between rooms (enforced in movement commands)

**Recovery Mechanics:**
1. **Automatic Recovery** - Stat-based logarithmic formula
   - Formula: `min(90, 25 + 20 × ln(DEX/25))` where DEX is Dexterity stat
   - Implemented in `Character.AttemptRecovery(statValue int)`
   - Called automatically each round via `NewRound_UserRoundTick` and `NewRound_MobRoundTick` hooks
   - Recovery attempts only after minimum prone duration expires
   - Failed recovery attempts set `RecoveryPenaltyThisRound = true` (limits attacks to 1)
   - Success rate examples: 25 DEX = 25%, 100 DEX = 53%, 300 DEX = 75%, capped at 90%

2. **Manual Recovery** - Stand command
   - Costs 15% of maximum stamina (config: `StandStaminaCost`)
   - Requires minimum 15% stamina remaining (config: `StandMinStamina`)
   - Guaranteed success, bypasses minimum duration
   - Immediately removes prone state and resets `ProneRoundsRemaining`

**Future-Proofing:**
- `AttemptRecovery()` accepts generic `statValue` parameter for other conditions (grapple, entangle)
- Can be called with Strength for grapple recovery, different stats for other effects

### Special Combat Moves
Three tactical combat abilities with knockdown mechanics and shared cooldown:

**Bash Command** (`usercommands/bash.go`):
- Requirements: Shield equipped (checked via `HasShield()`), in active combat
- Damage: 50% of Strength stat (config: `BashDamagePercent`)
- Knockdown chance: 40% base (config: `BashKnockdownChance`)
- Opposed check: Weapon Combat + Strength vs. target's combat skill + Dexterity
- Skill progression: Weapon Combat
- On knockdown: Sets `target.Prone = true`, `ProneRoundsRemaining = 2`

**Trip Command** (`usercommands/trip.go`):
- Requirements: In active combat
- Damage: 25% of Strength stat (config: `TripDamagePercent`)
- Knockdown chance: 60% base (config: `TripKnockdownChance`) - highest of the three
- Opposed check: Unarmed Combat + Dexterity vs. target's combat skill + Dexterity
- Skill progression: Unarmed Combat
- Tactical: Low damage, high setup potential

**Kick Command** (`usercommands/kick.go`):
- Requirements: In active combat
- Damage: 40% of Strength stat (config: `KickDamagePercent`)
- Knockdown chance: 35% base (config: `KickKnockdownChance`)
- Opposed check: Unarmed Combat + Strength vs. target's combat skill + Dexterity
- Skill progression: Unarmed Combat
- Balanced: Moderate damage and knockdown

**Shared Cooldown System:**
- All three moves share a single 5-round cooldown (config: `SpecialMoveCooldown`)
- Tracked in `Character.Cooldowns` map with key "combat-special"
- Cooldowns automatically decrement via `RoundTick()` called in combat hooks
- Prevents knockdown spam, encourages tactical timing

### Target Switching
Players can switch combat targets mid-fight using `attack <new-target>`:

**Implementation** (`usercommands/attack.go`):
- When already in combat (`user.Character.Aggro != nil`), retargets to new enemy
- Validates new target is in room and attackable
- Updates `Aggro.UserId` or `Aggro.MobInstanceId` to new target
- Automatic retargeting when current target dies (handled in combat death processing)
- Party coordination: Multiple players can target same enemy

**Messaging:**
- Informs user of target switch: "You shift your focus to <new-target>!"
- Room sees: "<player> shifts their focus to <new-target>!"

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

## Combat Analytics (Stage 30.1–30.2)

### Architecture
The analytics subsystem uses a ring buffer (`eventBuffer`) to capture every
combat action in real time. When the buffer reaches `maxEvents` (configured
via `Analytics.MaxEvents`), the oldest event is dropped (FIFO). A periodic
flush cycle (controlled by `Analytics.FlushIntervalSec`) aggregates the
buffer into an `AnalyticsSummary` and writes it as a single JSONL line to
the configured log path. The log is rotated by lumberjack (50 MB, 10
backups, compressed).

### CombatEvent Fields
- `SourceType` / `TargetType` — `User` or `Mob`
- `AttackType` — e.g. "unarmed", "weapon", "spell", "bash", "kick", "trip"
- `Hit`, `Crit`, `Fumble`, `Backfire`, `Fizzle` — outcome booleans
- `DamageDealt`, `DamageReduced` — integer damage values
- `DefenseUsed` — "dodge", "parry", "block", or ""
- `AttackZScore`, `DefenseZScore` — z-scores from opposed rolls
- `SourcePosition`, `TargetPosition` — "standing", "prone", etc.
- `SourceIsGrappleController`, `TargetIsGrappleController` — booleans
- `RoundNumber` — combat round counter

### AnalyticsSummary Fields
Aggregated totals (hits, misses, crits, fumbles, backfires, fizzles, total
damage), per-attack-type breakdowns (`ByAttackType`), defense success
counts (dodge/parry/block), matchup counts (PvM/MvP/PvP/MvM), position
hit rates, grapple controller hit rates, average z-scores, and round range.

### Recording Functions
- `RecordAttack()` — standard auto-attacks
- `RecordSpecialMove()` — bash, kick, trip, grapple, mutations
- `RecordSpell()` — spell resolution events

### Query Functions (Stage 30.2)
- `GetSummary()` — full aggregated summary of all buffered events
- `GetFilteredSummary(attackType)` — summary filtered to one attack type
- `GetBufferLen()` — current event count in buffer
- `ResetBuffer()` — clears buffer, returns count cleared
- `ExportNow()` — immediate flush to log file
- `GetAttackTypes()` — map of attack type → event count

### Admin Command: `combatstats` (Stage 30.2)
Subcommands: `summary [type]`, `types`, `matchups`, `defense`, `position`,
`reset`, `export`. All output uses `templates.GetTable()` for tabular
display. See `internal/usercommands/admin.combatstats.go`.
