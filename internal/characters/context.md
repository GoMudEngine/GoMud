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
- **Dynamic modifiers**: Equipment, buffs, pets, and mutations affect final stats
- **Use-based improvement**: Stats improve organically through gameplay

**Gear-effectiveness integration (chunk 2.2a):** `Character.StatMod()` multiplies
the Equipment portion of `Mods` by `mutations.GearEffectivenessMultiplier(c.Mutations)`
before summing with Buffs and Pet contributions. This cascades through `RecalculateStats()`
into all downstream consumers (stat values, mitigation, recovery, skills, spells).

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

## Mitigation System (Three Channels)

The character package provides three mitigation getter methods that compute
total damage reduction across all equipped items and modifications:

**Three Methods:**
- `GetPhysicalMitigation()` — defends against physical damage
- `GetMagicalMitigation()` — defends against spells
- `GetConvictionMitigation()` — defends against taunt/conviction damage

**Gear-effectiveness integration (chunk 2.2a):** Each method separates
gear-derived contributions (equipment slot mitigation) from non-gear
contributions (natural armor from mutations, species baseline, shield spell
magnitude, buff stat mods). The gear portion is multiplied by
`mutations.GearEffectivenessMultiplier(c.Mutations)` before summing.

**Slot coverage:** All 25 equipment slots are included in the three
mitigation getters, completed during chunk 2.2a:
- Physical mitigation: Shoulders, Back, Wrist1/2, Ring, Ring2, ExtraWrist1-4,
  ExtraArm3-4, ComponentBag (all physical-type armor items).
- Magical mitigation: same slots (all items can carry magical mitigation).
- Conviction mitigation: same slots.

This ensures characters with many-armed mutations or high-value jewelry can
leverage their full equipment potential for defense.

## Intrinsic Mutations (chunk 2.5)

`Character.ApplyIntrinsicMutations(species *species.Species)` merges
the species's intrinsic mutations additively into `Character.Mutations`.
No-op on nil species or empty intrinsic map. Cap-aware via
`MutationMaxRank = 4` (matches chunk-2.2a convention; no per-mutation
max field exists today).

Called once at character init AFTER all other mutation logic:
1. Curated SpawnMutations from mob YAML (mob spawn only)
2. Random-roll mutation acquisition (mob spawn + player round tick)
3. Persistent acquired mutations from save file (players only)
4. `ApplyIntrinsicMutations(species)` — this call

Stacks ADDITIVELY: a wolf species with `intrinsic_mutations: { tail: 1 }`
that also rolls `tail` rank 1 ends up with effective rank 2 in
`Character.Mutations`.

File: `internal/characters/intrinsic.go`

Design: `docs/superpowers/specs/2026-05-12-mob-aliveness-2.5-mutations-on-mobs-design.md`

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

## Combat Phase Machine Integration (chunk 0)

### New field: CombatPhase

```go
CombatPhase *combatphase.Machine `yaml:"-"`
```

Initialized in `New()` and lazily in `Validate()` (for characters loaded
from YAML without a direct `New()` path). `RegisterMachine` is called
immediately after allocation so inbound-attacker tracking is active from
the first combat action.

### New flag: NonCombatant

```go
NonCombatant bool `yaml:"non_combatant,omitempty"`
```

`true` = character is immune to combat targeting (shops, quest-givers,
etc.). Set from `Mob.NonCombatant` during `Mob.Validate()` for mob
characters; set directly in player creation for any exempt player
archetype.

```go
func (c *Character) IsCombatant() bool { return !c.NonCombatant }
```

The `RegisterCombatantVeto` wiring in `CombatPhase_Vetoes.go` calls this
to block `TransitionToEngaging` for non-combatants.

### Internal guard: combatPhaseWired

```go
combatPhaseWired bool `yaml:"-"`
```

Set to `true` the first time `fireCharacterCreated` runs. The `Validate()`
path checks this flag to avoid double-firing `OnCharacterCreated` callbacks
when `Validate()` is called multiple times during a character's lifetime.

### Predicate methods

All read from `CombatPhase` exclusively; they do not read the legacy
`Aggro` field.

```go
func (c *Character) IsEngaged() bool
    // true when Combat Phase == Engaged (actively fighting)

func (c *Character) IsInCombat() bool
    // true when Combat Phase != Idle (any non-idle combat state)

func (c *Character) IsDisengaging() bool
    // true when Combat Phase == Disengaging (flee in progress)

func (c *Character) EngagedTarget() state.ActorRef
    // current target when Engaged; zero when not Engaged

func (c *Character) CurrentCombatTarget() state.ActorRef
    // current target across all non-Idle states (Engaging/Engaged/Disengaging)

func (c *Character) Attackers() []state.ActorRef
    // snapshot of inbound attacker list from CombatPhase
```

### Legacy Aggro field (compat surface)

The `Aggro *Aggro` field is kept in `combat_state_compat.go` for the
~200 direct field reads in usercommands, hooks, combat, and mob-commands
that were not migrated in chunk 0. **Do not add new reads against
`Character.Aggro`** — use the predicate methods above.

All writes go through `SetAggro` / `EndAggro`, which dual-write to both
`Aggro` and `CombatPhase.TransitionToEngaging` / `ForceIdle`. Direct
mutation of `Character.Aggro` (bypassing the wrappers) is forbidden.

Field removal is scheduled for a cleanup chunk after chunks 1-5 land and
the remaining reads are migrated.

### OnCharacterCreated callback registry

```go
func OnCharacterCreated(fn func(*Character))
```

Registers a callback that fires once per `Character` the first time it
is fully initialized (after `New()` or on first `Validate()` if loaded
from YAML). Used by the hooks package to wire state-machine vetoes and
observers without creating an import cycle (characters cannot import
hooks; hooks import characters).

Current registrations (all in `internal/hooks/`):
- `wireCombatPhaseVetoes` — wires the seven veto closures
- `wireCombatPhaseBtreeEvents` — wires the btree transition cascade
- `wireCompanionAssist` — subscribes to Attackers-change events

## Awareness Machine Integration (chunk 1)

### New field: Awareness

```go
Awareness *awareness.Machine `yaml:"-"`
```

Initialized in `New()` and lazily in `Validate()` (for characters loaded
from YAML without a direct `New()` path). The awareness machine tracks
whether a character is currently hidden and coordinates state transitions
for sneak attempts, detection, and revealing. It operates independently
of Combat Phase but cascades through the same hook framework.

### New predicate: IsHidden()

```go
func (c *Character) IsHidden() bool
    // true when Awareness == Hidden
    // replacement for the old HasBuffFlag(buffs.Hidden) pattern
```

The only canonical way to check if a character is hidden. It reads directly
from the Awareness machine's state, not from buff #9 (which is now a
side-effect carrier only).

### Cascade pattern: Awareness to Buff #9

The `Awareness_Cascades.go` hook ensures buff #9 ("Hidden" status effect)
stays synchronized with the Awareness machine:

- When Awareness transitions to `Hidden` state, the hook applies buff #9
  to the character (providing stat mods and room broadcast text).
- When Awareness transitions away from `Hidden`, the hook removes buff #9.

This maintains backward compatibility with systems that check for buff #9
while keeping the Awareness machine as the canonical state source.

### Hidden movement stamina scaling

When a character is `Hidden`, movement stamina cost is multiplied by
`HiddenMoveStaminaMultiplier` (config default 1.0, tunable at runtime).
This is read in `GetMovementStaminaCost()` and applied before returning
the movement cost to the caller.

### Integration with Combat Phase

The Awareness machine subscribes to Combat Phase's `OnEndOfRoundIfSurprise`
callback (wired in `Awareness_Cascades.go`). When a surprise engagement
completes its first round of swings, the Awareness machine triggers a
reveal cascade (`Hidden → Revealing → Visible`), forcing surprise-attacked
sneakers out of hiding. The full cascade completes before the next round
begins, ensuring surprised attackers are visible for retaliation.

### Logout cleanup

The `Logout_AwarenessCleanup.go` hook calls `ForceVisible()` on logout,
ensuring the awareness machine doesn't leak state or block future character
reuses (edge case safety).

## Life Machine Integration (chunk 2)

### New field: Life

```go
Life *life.Machine `yaml:"-"`
```

Initialized in `New()` and lazily in `Validate()` (for characters
loaded from YAML without a direct `New()` path). `RegisterMachine`
is called immediately after allocation. The Life machine is the
canonical source of truth for "is this character alive?".

### Predicate methods

```go
func (c *Character) IsAlive() bool
    // true when Life == Alive

func (c *Character) IsDead() bool
    // true when Life == Dead

func (c *Character) IsRespawning() bool
    // true when Life == Respawning (player only)
```

Note: these predicates call through to the Life machine. Tests that
exercise code paths gated by these predicates must initialize the
Life machine (via `Validate()` or direct `NewMachine()` assignment)
or the call will panic on a nil pointer.

### Die helper (die.go)

```go
func (c *Character) Die(killer state.ActorRef, trigger string)
```

Chains all Life transitions in the correct order. Players complete
all three states (`Dead → Respawning → Alive`) same-tick via
synchronous `AfterTransition` observer chains. Mobs only transition
to `Dead`; the instance-cleanup observer fires synchronously and
despawns the mob.

Callers MUST pre-check before calling `Die`:
1. `ReviveOnDeath` buff (prevents death; callers bail early if set)
2. `LastSuicideRound` dedupe (if the call site can double-fire)
3. Shadow Realm zone guard (player call sites only)

`Die` is idempotent: if the Life machine is already `Dead` or
`Respawning` it returns immediately without firing observers.

### ResolveRespawnRoom (respawn_home.go)

```go
func (c *Character) ResolveRespawnRoom() int
```

Reads the player's `"home"` setting, looks it up in
`HomeLocations`, and falls back to `"default"` (Sanctum Basin
entrance, room 0) if unset or unrecognized.

`HomeLocations` maps setting key → room ID. `HomeLocationNames`
maps setting key → display string. Both are exported maps consumed
by `sethome.go` (key validation) and by `Respawn_PlayerTeleport.go`
(destination resolution).

Current entries:

| Key | Room ID | Display Name |
|-----|---------|--------------|
| `"default"` | 0 | Sanctum Basin |
| `"thornwall"` | 468 | Thornwall City (Temple Interior) |
| `"stillwater"` | 4123 | Stillwater (Temple of Stillwater) |

### MobInstanceId field

```go
MobInstanceId int `yaml:"-"`
```

Non-persisted field set to the mob's live `InstanceId` at
character initialization. Used as a cheap gating check in Life
machine observers (`c.MobInstanceId != 0` = mob) without requiring
a cast or registry lookup.

### OnCharacterCreated additions (chunk 2)

The `OnCharacterCreated` registry gains Life-machine wire callbacks.
New registrations (all in `internal/hooks/`):
- `wireLifeMachine` — registers the Life machine and all Death +
  Respawn observer chains

## Activity Machine Integration (chunk 3)

### New field: Activity

```go
Activity *activity.Machine `yaml:"-"`
```

Initialized in `New()` and nil-guarded in `Validate()` (for characters
loaded from YAML without a direct `New()` path). The Activity machine
is the canonical source of truth for "what multi-round action is this
character locked into right now?"

### Predicate methods

```go
func (c *Character) IsFree() bool
    // true when Activity == Free (no activity in flight)

func (c *Character) IsCasting() bool
    // true when Activity == Casting
    // replaces the old c.CastingState != nil check

func (c *Character) IsCrafting() bool
    // true when Activity == Crafting
    // replaces the old c.CraftingState != nil check

func (c *Character) IsSalvaging() bool
    // true when Activity == Salvaging

func (c *Character) IsActing() bool
    // true when Activity != Free (any non-Free state)
    // canonical "is busy" gate replacing the old IsCrafting() gate
    // at special-moves check sites (13 call sites rewired in chunk 3)
```

`IsActing()` is preferred for "should this action be blocked because
the character is busy?" checks. Use the specific predicates only when
you need to distinguish which activity is running (e.g., the craft
command's own re-entrancy check).

### OnCharacterCreated additions (chunk 3)

The `OnCharacterCreated` registry gains the Activity machine wire
callback. New registration (in `internal/hooks/`):
- `wireActivityCrossMachineCascades` — subscribes `activity_life_dead`
  observer to the Life machine; wires the Activity machine's identity
  via `RegisterMachine`.

### Sunset notes (chunk 3)

The following fields and files were deleted in chunk 3:
- `Character.CastingState *characters.CastingState` field
- `Character.CraftingState *characters.CraftingState` field
- `internal/characters/casting.go` — `CastingState` struct
- `internal/characters/crafting.go` — `CraftingState` struct
- `CraftingState.MiscData["salvage_item_uuid"]` key pattern

All call sites that read `c.CastingState != nil` or
`c.CraftingState != nil` were migrated to `IsCasting()` / `IsCrafting()`
/ `IsSalvaging()` / `IsFree()` / `IsActing()` predicates.

## Position Machine Integration (chunk 4a — scaffold)

### New field: Position

```go
Position *position.Machine `yaml:"-"`
```

Initialized in `New()` and nil-guarded in `Validate()` (for characters
loaded from YAML without a direct `New()` path). The Position machine
is the canonical source of truth for body geometry and grapple state.
Chunk 4a scaffolded the machine DORMANT; **chunk 4b wired every
production writer and migrated most readers** (writers W1-W8: grapple
entry, legacy progression delete, submission outcomes, trip/bash,
spell knockdown, auto-recovery, stand command; readers R1/R2/R3/R5/R6:
combat math, third-party defense filter, flee blockers, CombatPhase
position check, prompt `{pos}` token). R4 (delete Life cascade Position
pre-wire) and S1-S5 (legacy field sunsets) are **deferred** pending a
broader CombatPosition reader sweep — see memory entry
`project_chunk_4b_r4_blocked_on_reader_sweep.md`.

### Predicate methods (chunk 4a + 4b)

**Chunk 4a — 19 predicates** in `position_predicates.go` delegate to
the underlying machine with nil guards. Nil-guard convention:
`IsStanding()` returns `true` on a nil machine (matches `NewMachine()`
default); all others return `false`.

14 per-state predicates: `IsStanding`, `IsProne`, `IsSupine`,
`IsClinch`, `IsBackStanding`, `IsMount`, `IsSideControl`,
`IsKneeOnBelly`, `IsNorthSouth`, `IsCrucifix`, `IsBackGround`,
`IsHalfGuard`, `IsGuard`, `IsTurtle`.

5 rollup predicates: `IsGrappling`, `IsStandingGrapple`,
`IsGroundGrapple`, `IsTopDominant`, `IsOnFloor`.

**Chunk 4b — 4 control-axis predicates and helpers:**

- `IsController()` — true when the character is on the controller side
  of a grapple pair (reads `Position.GrappleData().ControlLevel`
  via `IsControllerLevel`). Replaces the legacy
  `HasCondition(ConditionGrappleController)` check; sunset target S4.
- `IsBeingControlled()` — true when the character is on the controlled
  side (symmetric to `IsController`).
- `IsLowGrappleStamina()` — true when stamina fraction is below
  `GrappleStaminaLowThreshold` (config, default 0.25). Used by
  `mob_low_grapple_stamina` btree primitive and by
  `Position_Messaging` for the once-per-grapple "you're getting
  gassed" warning.
- `GetPositionSpeedMultiplier()` — replaces the legacy
  `CombatPosition.GetSpeedMultiplier()` helper (sunset S5). Switches
  on `Position.State()`: Standing 1.0, Prone/Supine/Turtle 0.5,
  Clinch/BackStanding 0.6, ground grapples 0.3.

**Legacy enum coexistence — status:** the chunk 4b reader sweep is
**in progress** but not complete. Combat math (`combat_helpers.go`,
all 6 reader sites) and the third-party defense filter
(`IsThirdPartyAttack`) read the FSM. ~25 readers across `combat/ai.go`,
`combat/grapple.go`, `actions/combat_kick.go`,
`actions/command_readiness.go`, `behaviortree/conditions_mob.go`,
`hooks/combat_shared_helpers.go`, `mobcommands/submit.go`,
`usercommands/submit.go`, `characters/combat_state_compat.go` are
still unmigrated. The legacy `CombatPosition` enum, its
`IsGroundPosition()` / `IsGrapplePosition()` / `GetSpeedMultiplier()` /
`GetPositionColor()` helpers, the `PositionRoundsMin` /
`GrappleControllerId` fields, and the `ConditionGrappleController`
constant remain in place until S1-S5 land. The mapping table for
migrators:

| Legacy reader | New FSM predicate |
|---------------|-------------------|
| `== PositionProne` | `IsProne() \|\| IsSupine()` |
| `== PositionClinched` | `IsStandingGrapple()` |
| `== PositionGrounded` | `IsGroundGrapple()` |
| `!= PositionStanding` | `!IsStanding()` |
| `.IsGrapplePosition()` | `IsGrappling()` |
| `.IsGroundPosition()` | `IsOnFloor()` |
| `.GetSpeedMultiplier()` | `GetPositionSpeedMultiplier()` |
| `HasCondition(GrappleController)` | `IsController()` |

### Prompt helpers (chunk 4b R6)

The `{pos}` prompt-token cutover added two private helpers in
`internal/users/userrecord.prompt.go`:

- `positionPromptColor(position.State) string` — returns the ANSI
  color name. Standing white, Prone/Supine yellow, Clinch/BackStanding
  orange, ground grapples red. Replaces the legacy
  `CombatPosition.GetPositionColor()`.
- `positionPromptAbbrev(position.State) string` — abbreviates long
  state names: BackStanding→B.Std, BackGround→B.Gnd, SideControl→SC,
  KneeOnBelly→KOB, NorthSouth→N-S, HalfGuard→H.Gd. Other states
  render verbatim via `State.String()`.

These live in the users package (not characters) because they format
the prompt-substitution output, not the underlying state.

### OnCharacterCreated additions (chunk 4a + 4b)

The `OnCharacterCreated` registry gains four Position-related wire
callbacks across chunks 4a and 4b:

- **4a `wirePositionCrossMachineCascades`** — subscribes the
  `position_life_dead` observer to the Life machine; handles
  `Alive → Dead` cascade that resets Position to `Standing`.
- **4b `wirePositionGrappleTick`** — registers the per-round drift
  observer that fires opposed control rolls + grapple stamina cost +
  threshold-triggered position transitions.
- **4b `wirePositionMessaging`** — registers the per-round messaging
  observer that fires gradient ("getting controlled"), transition
  ("you scramble out of mount"), and stamina-warning text with
  per-grapple cooldowns.
- **4b `wirePositionConsistencyCheck`** — registers the periodic
  invariant checker (`ValidateGrapplePair`) that catches pair drift
  (e.g. controller's partner ref doesn't match controlled's ref).

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
- `internal/state/combatphase`: Combat Phase state machine (chunk 0)
- `internal/state/awareness`: Awareness state machine (chunk 1)
- `internal/state/life`: Life state machine (chunk 2)
- `internal/state/activity`: Activity state machine (chunk 3)
- `internal/state/position`: Position state machine (chunks 4a + 4b)
