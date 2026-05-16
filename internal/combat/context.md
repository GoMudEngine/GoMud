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
// - Grapple position: `c.IsController()` + IsStandingGrapple -0.2, IsGroundGrapple -0.4 (chunk 4b R1)
// - Backstab: guaranteed crit on first pass
```

### Dual Wielding
- Skill level 2: 50% chance to use both weapons
- Skill level 3+: Always dual wield
- Claws (natural weapons): Always dual wield
- Hit penalty: continuous scale from 50 (skill 0) to 10 (skill 50)
  - Formula: `penalty = 50 - (dualWieldLevel / 50.0) * 40`
  - Floor: minimum 10% penalty
- Extra arms (mutation): `+20` penalty for 3rd weapon, `+40` for 4th

## Stage 7.5: Tactical Combat Enhancements

### Layered Defense System (Best-of-All)
Combat defense uses a best-of-all system where ALL available defenses are
rolled and the one that won by the widest margin is selected. This replaced
the old sequential short-circuit approach. Benefits: every defense type gets
fair representation in combat text, and having multiple defense types is
always better (wider net).

**Defense Types:**
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

**Defense Floor:** `MinDefenseChance` (default 0.15) ensures even massively
outclassed defenders have a 15% chance to avoid any swing.

**Stamina Costs:**
- Only the winning defense (best margin) costs stamina — losing defenses are free
- Low stamina drains smooth penalty via `ResourceMultiplier`
- Defense costs configured in `config.gameplay.yaml`

**Implementation:**
- `runBestOfAllDefense()` in `combat_helpers.go` rolls every defense
- `resolveDefenseOutcome()` processes the best result and floor saves
- Two floors: `MinDefenseChance` (defender saves when attack wins) AND
  `MinAttackHitChance` (attacker hits when defense wins)
- Defense crit detection (z > 2.0): parry crit → disarm, dodge crit → grapple opportunity

### Prone / Supine Knockdown System (Position FSM, chunks 4a + 4b)

Characters can be knocked to the ground by special combat moves
(bash/trip/kick) or spell knockdowns, applying severe combat penalties.
Chunk 4a split the legacy single "prone" state into **Prone** (face-down)
and **Supine** (face-up); chunk 4b cut over every writer and most readers
to the new Position FSM.

**Down state (Position FSM):**
- Canonical source: `Character.Position` (`*position.Machine`). Use
  `c.IsProne()` and `c.IsSupine()` predicates; rollup `c.IsOnFloor()`
  covers either + ground grapples.
- Per-state data: `ProneData` / `SupineData` carry
  `MinRecoveryRounds` (replaces legacy `PositionRoundsMin`),
  `KnockdownSource` (the attacker's `ActorRef`), and the
  `TransitionReason`.
- The legacy `CombatPosition` / `PositionRoundsMin` parallel-writes are
  removed (T21 sunset). The legacy enum collapsed Prone/Supine into one
  bucket; the FSM distinguishes them.
- Visual indicator: "prone" adjective added via `GetAdjectives()` (still
  enum-driven; helpfile content enhancement deferred to chunk 4f).

**Why split Prone vs Supine?** Submission paths and recovery
mechanics diverge: Prone is back-take-vulnerable and harder to
recover from; Supine can pull guard (`TransitionToGuard`) and recovers
more easily. Mechanically (4b): both states share the same combat
penalty profile via `IsProne() || IsSupine()` reads in
`combat_helpers.go` (R1). Submission-engine divergence is chunk 4d.

**Combat modifiers** (applied in `combat_helpers.go`, all migrated to
`IsProne() || IsSupine()` in chunk 4b R1):
- Attacker prone: `dmgMean *= ProneDamagePenalty` (config),
  `attackScore *= ProneAttackMultiplier`
- Defender prone: `attackScore *= ProneVulnerabilityMultiplier`
- Dodge/parry/block penalties: `ProneDodgePenalty` / `ProneParryPenalty`
  / `ProneBlockPenalty` (defense penalty switch reads
  `IsProne() || IsSupine()` → prone bucket).

**Behavioral restrictions** (chunk 4b R3, reads
`IsProne() || IsSupine()`):
- Cannot flee from combat (`mobcommands/flee.go` + `handlePlayerFlee`
  apply a 0.5x flee-score penalty; grapple states block flee entirely).
- Cannot move between rooms (enforced in movement commands —
  unmigrated reader, scheduled in the broader sweep).

**Recovery mechanics** (chunk 4b W6 / W7 cutover):

1. **Automatic recovery** — stat-based logarithmic formula
   `min(90, 25 + 20 × ln(DEX/25))`. Implemented in
   `Character.AttemptRecovery(statValue int)`. Gates on
   `IsProne() || IsSupine()` and reads `MinRecoveryRounds` from
   `ProneData` / `SupineData`. Decrements via
   `Position.ConsumeRecoveryRound()` (mutates the per-state slot in
   place; analogous to `MutateGrappleControlLevel`). On success fires
   `Position.TransitionToStanding(TriggerRecoveryRoll)` plus the legacy
   parallel-write. Called every round via `NewRound_UserRoundTick`
   and `NewRound_MobRoundTick`. Failed attempts add
   `ConditionRecoveryPenalty` (limits attacks to 1).

2. **Manual recovery** — `stand` command (`internal/usercommands/stand.go`)
   - Costs `StandStaminaCost` (config, 15% of max). Requires
     `StandMinStamina` remaining.
   - Bypasses `MinRecoveryRounds`. Fires
     `Position.TransitionToStanding(TriggerStandCommand)` BEFORE
     deducting stamina so an FSM-edge failure bails without charge.
   - The legacy `CombatPosition` / `PositionRoundsMin` parallel-writes
     are removed (T21 sunset); the FSM transition is the sole write.

### Grapple mechanics (Position FSM control axis, chunk 4b)

The 11 grapple states (Clinch, BackStanding, Mount, SideControl,
KneeOnBelly, NorthSouth, Crucifix, BackGround, HalfGuard, Guard,
Turtle) and the per-grappler **control axis**
(`ControlLevel: InControl ↔ LosingControl ↔ Neutral ↔ BecomingControlled ↔ Controlled`)
live in `internal/state/position/`. Canonical doc:
`internal/state/position/context.md`. Brief summary of how the combat
package interacts:

- **Per-round drift** — `Position_GrappleTick.go` (hooks package) fires
  the opposed Strength + Unarmed-combat roll each round, scaled by
  stamina + encumbrance curves, and shifts `ControlLevel` via
  `MutateGrappleControlLevel` (no FSM transition — the table forbids
  `Mount→Mount`). Threshold crossings can fire follow-up position
  transitions.
- **Per-round stamina cost** — `GrappleStaminaCostPerRound` × a
  per-role multiplier (controller 1.0x, controlled 2.0x by default;
  asymmetry is the "smother" feedback loop).
- **Third-party defense filter** — `IsThirdPartyAttack` (chunk 4b R2)
  now reads `target.IsGrappling()` + `GrappleData.Partner` instead of
  the deleted `CombatPosition.IsGrapplePosition()` + `GrappleControllerId`
  fields. Zero-Partner (solo Turtle) preserves the "no controller → not
  third-party" semantics; 4e refines.
- **Crit-threshold bonuses** — controller in any grapple grants a
  crit boost (-0.2 standing, -0.4 ground); reads `c.IsController()`
  (chunk 4b R1, replaces `HasCondition(ConditionGrappleController)`).

The legacy `CombatPosition` enum is fully removed (T21 sunset). All
readers in `combat_helpers.go`, `ai.go`, `grapple.go`, and across
the codebase were migrated in the chunk-4b reader sweep before deletion.

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
- All three moves share a single cooldown (config: `SpecialMoveCooldown`, currently 4 rounds)
- Tracked in `Character.Cooldowns` map with key "combat-special"
- Cooldowns automatically decrement via `RoundTick()` called in combat hooks

**Intentional: Cooldown-blocked specials still initiate combat.**
If a player opens a fight with a special move (kick, bash, trip) while
on cooldown, the move itself fizzles but combat still starts. This is
by design — the player committed to an aggressive action and the target
noticed. This prevents risk-free cooldown probing (try special on a
passive mob, walk away if on cooldown, repeat). The player learns to
track their cooldown timing.
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

## Power Scoring & Gear Contribution

`combat.PowerScore(char)` combines six terms: Offense, Defense,
Durability, Skills, Mutations, and KD ratio. Equipment
contribution flows through the standard pipes; there is no
separate "gear quality" axis.

| PowerScore term | Equipment field(s) that feed it |
|---|---|
| Offense (physAtk per-swing) | weapon `DamageMultiplier`, `SpeedMultiplier`; offhand + ExtraArm weapons |
| Offense (magAtk caster) | equipped weapon `SpellDamageMultiplier` |
| Offense (any stat-derived) | equipment `StatMods` → `Stats.X.ValueAdj` |
| Defense (mitigation) | equipment `PhysicalMitigation` / `MagicalMitigation` / `ConvictionMitigation` summed by `char.Get*Mitigation()` |
| Defense (avoidance) | equipment-driven dodge/parry/block via `char.GetDefenseScore(...)` |
| Durability | `char.HealthMax.Value` / `StaminaMax.Value` / `ConvictionMax.Value` — all reflect equipment stat boosts |
| Skills | not gear-driven |
| Mutations | not gear-driven |
| KD ratio | not gear-driven |

A player swapping a steel sword for an iron one will see
PowerScore drop because (a) the weapon's `DamageMultiplier`
changes (physAtk) and (b) any stat-mod difference flows through
`ValueAdj` into multiple terms. The Incorporeal mutation (chunk
2.2a) further scales gear contributions via
`mutations.GearEffectivenessMultiplier` — an ethereal wraith's
PowerScore reflects gear at the rank-determined fraction.

Consumers: `actions.Consider` (player + mob `consider`),
behavior tree conditions `target_power_ratio_above` and
`target_power_ratio_below`, behavior tree action
`target_weakest_mob_in_room`.

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

## Typical Combat Round Walkthrough (Player vs Mob)

This traces one full round for a player with sword + shield fighting an
armed mob with a shield. The player uses bash on cooldown and occasionally
casts offensive spells. Both characters are standing, not grappled.

### 1. Round Tick Fires: `DoCombat()`

`hooks/NewRound_DoCombat.go` — The engine emits a `NewRound` event each
tick. `DoCombat` is the listener:

```
DoCombat(evt)
  handlePlayerCombat(evt)    // all players act first
  handleMobCombat(evt)       // then all mobs act
  handleAffected(...)        // death/disable resolution
```

### 2. Player's Turn: `handlePlayerCombat()`

`hooks/NewRound_DoCombat.go` — Loops every online user. For each player
in combat:

#### 2a. NoCombat Buff Check
If the player has a `NoCombat` buff flag, skip the entire combat turn
(including shield decay). This check happens before anything else.

#### 2b. Shield Decay
`handlePlayerShieldDecay(user)` — If the player has a `ConditionShield`
(from Minor Shield spell), its duration ticks down. At 0, removed.

#### 2c. Fold Casting Check
`handlePlayerFoldCasting(user, userId)` — If the player typed
`cast fireball` last round, `c.IsCasting()` is true (Activity machine
is in Casting state):

1. Prone/disabled check — breaks concentration immediately.
2. Conviction cost — proportional to folds gained this round:
   `roundCost = (totalConvictionCost * foldDelta) / foldsNeeded`.
   If conviction is too low, the spell fizzles.
3. Fold accumulation — folds double each iteration:
   0 -> 1 -> 2 -> 4 -> 8 -> ... until `foldsNeeded` is reached.
4. When complete, calls `resolveSpell()`:
   - Harm spell vs mob: `spellAttack = Willpower + SpellcastingSkill`,
     opposed roll vs target's defense (usually Willpower).
   - Damage: `CalcRawDamage(Willpower, spellcastingRank,
     spellDmgMult * weaponSpellDmgMult, ChannelMagical)`.
   - Applies `GetMagicalMitigation()` from target's gear.
   - Z-score <= -2.0 = backfire (damages caster).
   - Z-score >= 2.0 = crit (double damage, bypasses mitigation).
5. Triggers spell discovery chance, skill progression.
6. Returns `true` — player skips normal melee this round.

If the player is NOT casting, flow continues to melee.

#### 2d. Aggro Check
If `user.Character.Aggro == nil`, skip (player not in combat).

#### 2e. Cancel Combat-Incompatible Buffs
`CancelBuffsWithFlag(buffs.CancelIfCombat)` — strips buffs like stealth.

#### 2f. Flee Check
`handlePlayerFlee(user, uRoom, userId)` — if the player typed `flee`,
attempts escape. On success, player leaves combat.

#### 2g. PvM Dispatch: `handlePlayerVsMob()`

`hooks/NewRound_DoCombat_helpers.go` — The main player attack sequence:

1. **Target validation** — mob exists, same room (or reachable via exit
   for ranged).
2. **Rounds waiting** — If `RoundsWaiting > 0` (from bash or weapon
   wind-up), decrement and show a preparation message via
   `GetWaitMessages()`. This is how bash costs a round: last round the
   player used bash, it set `RoundsWaiting = 1`, so this round the
   player only sees a flavor message. Skip to mob's turn.
3. **Grapple progression** — `processGrappleProgression()` handles
   clinch to ground advancement.
4. **Moon mods** — `applyMoonMods()` temporarily adjusts stats for
   mutated characters based on moon phase.
5. **THE ATTACK** — `combat.AttackPlayerVsMob(user, defMob)` (see
   Section 3 below).
6. **Post-attack bonuses:**
   - Conviction Surge buff: +15% damage if active.
   - Adrenaline Surge mutation: bonus damage when low HP.
7. **Crit effects** — `applyPvMCritEffects()`: parry crits attempt
   disarm, dodge crits create grapple opportunity.
8. **Apply buffs** from attack result (crit buffs on target).
9. **Dispatch messages** to player, room, defender room.
10. **Mob concentration break** — if the mob was casting and got hit,
    roll Willpower vs damage% to see if concentration holds.
11. **Scripting hook** — `onHurt` mob script fires.
12. **Hostility** — mob's group becomes hostile to the player.
13. **Mob retaliates** — if mob wasn't already aggro'd, it attacks back.
14. **End check** — if either is at 0 HP, end aggro. If player won,
    `handleAutoRetargetPlayer()` finds the next mob.

### 3. The Core Attack: `combat.AttackPlayerVsMob()`

`combat/combat.go` — Wrapper that calls `calculateCombat()` then applies
side effects:

```
attackResult = calculateCombat(*user.Character, mob.Character, User, Mob)
user.Character.DeductAttackStamina()
mob.Character.ApplyHealthChange(-totalDmg)
mob.Character.TrackPlayerDamage(userId, dmg)  // loot attribution
user.Character.OnStatUse("strength")          // progression
user.Character.OnStatUse("dexterity")         // progression
if hit: user.Character.OnSkillUse(combatSkill)
if crit: user.Character.OnCriticalSuccess(combatSkill)
if fumble: user.Character.OnCriticalFailure(combatSkill)
if dualWield: extra OnSkillUse(WeaponCombat)
user.PlaySound("hit-other"/"miss", "combat")  // MSP sound events
```

### 3a. Inside `calculateCombat()` — Step by Step

`combat/combat.go` — Orchestrator calling helpers in `combat_helpers.go`:

**Step 0: StatMod Bonuses**
```
statModDBonus = sourceChar.StatMod("damage")   // flat bonus to damage
extraAttacks  = sourceChar.StatMod("attacks")  // extra attack passes
backstabCrit  = (Aggro.Type == BackStab)        // first pass auto-crits
```

**Step 1: Attack Count** — `calcAttackCount()`
```
attackCount = 1 + floor(Dexterity / 50) + extraAttacks
  e.g., DEX 120: 1 + 2 + 0 = 3 attack passes

Then multiplied by:
  ResourceMultiplier(stamina)    // smooth penalty as stamina drains
  encumbrance penalty            // if overloaded
  minimum 1
```

**Step 2: Collect Weapons** — `collectAttackWeapons()`
- Main hand: sword (ItemId > 0) — added.
- Offhand: shield (Type = Shield, NOT Weapon) — not added.
- Extra arms (mutation): `ExtraArm1`/`ExtraArm2` added as additional weapons.
- Result: 1 weapon (the sword). No dual-wield penalty applies.

**Step 3: Per-pass loop** (3 passes for DEX 120):

For each pass, for each weapon (just the sword):

**Step 3a: Build Weapon Setup** — `buildWeaponSetup()`
```
ws.attacks     = weapon.GetDistributionDamage() attacks (e.g. 2)
ws.baseDmg     = weapon base damage
ws.weaponDmgMult = item's damage_multiplier (e.g. 1.2)
ws.weaponSpeed = item's speed multiplier (e.g. 1.0)
ws.attacks     = GetModifiedAttackCount(attacks, speed)  // skill-modified
ws.attacks    *= c.GetPositionSpeedMultiplier()         // position modifier (Position FSM, chunk 4b R1)
// ConditionRecoveryPenalty: forces attacks = 1
// Racial bonus: weapon.StatMod(RacialBonusPrefix + targetSpecies)
// Hard cap: max 4 swings per weapon per pass
```

**Step 3b: Build Damage Params** — `buildDamageParams()`
```
rawDmg = CalcRawDamage(Strength, combatSkillLevel, weaponDmgMult,
                        ChannelPhysical)
       = Strength * SkillMultiplier(rank) * weaponDmgMult * 0.30

Example at STR=120, rank=10, weaponDmgMult=1.2:
  SkillMultiplier(10) = 1.0 + (3.0-1.0) * sqrt(10/50) = 1.894
  rawDmg = 120 * 1.894 * 1.2 * 0.30 = 81.8

dmgMean = ApplyMitigation(rawDmg, mob.GetPhysicalMitigation(), 0.75)
  e.g., mob has 20% phys mitigation: dmgMean = 81.8 * 0.80 = 65.4

dmgVariance = dmgMean * RollSpread = 65.4 * 0.15 = 9.8

Further multiplied by:
  ResourceMultiplier(health)         // HP-based melee penalty
  ProneAttackMultiplier (if prone)   // 0.80x
  Mutation damage multiplier         // if any
```

**Step 3c: Per-swing loop** (e.g. 2 swings per pass):

For each swing:

**i. Attack Score** — `calcAttackScore()`
```
attackScore = Dexterity + combatSkillLevel - dualWieldPenalty
            = 120 + 10 - 0 = 130

Then multiplied by:
  ResourceMultiplier(stamina)          // stamina penalty
  ProneAttackMultiplier (if prone)     // 0.80x
  ProneVulnerabilityMultiplier         // 1.15x if target is prone
```

**ii. Fumble Check**
```
initialAttackRoll = dice.RollStat(130)
  mean=130, stdDev = 130 * RollSpread = 19.5
if ZScore <= -2.0: FUMBLE (miss, ~2.3% chance)
```

**iii. Best-of-All Defense** — `runBestOfAllDefense()`

The mob has defenses: dodge (always), parry (has weapon), block (has
shield). All three are rolled simultaneously:

```
For each defense in [dodge, parry, block]:
  1. Deduct defense stamina cost.
  2. defenseScore = mob.GetDefenseScore(defenseType)
       dodge: DEX-based
       parry: weapon parry rating
       block: shield block rating
  3. Multiply by effectiveness (DodgeEffectiveness, ParryEffectiveness,
     BlockEffectiveness from config).
  4. Multiply by prone penalties if applicable.
  5. Opposed roll: dice.OpposedRollStat(attackScore, defenseScore)
  6. margin = defenseRoll.Value - hitRoll.Value
  7. Keep the defense with the HIGHEST margin.
```

**iv. Resolve Defense** — `resolveDefenseOutcome()`
```
if best.margin > 0:
  Defense succeeded — but check attack floor first:
    MinAttackHitChance: attacker still has a small chance to hit anyway.
    If attack floor triggers: HIT despite defense winning.
  Otherwise: Attack avoided.
  Send defense message ("The goblin blocks your attack!")
  Check for defense crits (defRoll.ZScore > 2.0):
    Parry crit -> disarm opportunity
    Dodge crit -> grapple opportunity
  Trigger skill progression for the defense type.
  return hit=false

if best.margin <= 0:
  Check defense floor: MinDefenseChance (15%).
  Random roll: 15% chance to save anyway ("narrowly blocks").
  If floor save: hit=false.
  Otherwise: HIT. return hit=true.
```

**v. Momentum** — `sourceChar.UpdateMomentum(hit)` — consecutive
hits/misses affect stance display text.

**vi. If HIT** — `calcHitDamage()`
```
Crit check: hitRoll.ZScore >= critThreshold (default 2.0)?
  CRIT: damage = roll(rawDmgForCrit, variance)  // PRE-mitigation!
        Apply crit buffs to target.

Normal hit:
  damage = roll(dmgMean, variance)  // POST-mitigation
  Round to nearest int, minimum 0.
```

**vii. Build Messages** — `buildAttackMessages()`
```
Select message template by weapon subtype + damage percentage.
Apply token replacements ({source}, {target}, {itemname},
  {damage description}, {stance}, {position}, {momentum}).
Wrap in *** *** for crits, !!! !!! for fumbles.
Send to attacker, defender, room observers.
```

**viii. Pet Damage** — `applyPetDamage()` — 20% chance the player's pet
joins in with bonus damage.

**Step 4: Accumulate** — all damage from all passes, swings, and weapons
adds up in `AttackResult.DamageToTarget`.

### 4. Mob's Turn: `handleMobCombat()`

`hooks/NewRound_DoCombat.go` — Loops every mob instance:

#### 4a. Pre-checks
Mob alive? Has aggro? Not in NoCombat buff? Load room. Cancel
combat-incompatible buffs. Shield decay (symmetric with player —
inline in `handleMobCombat`, not extracted to a helper).

#### 4b. Fold Casting Check
`handleMobFoldCasting(mob, mobRoom)` — Same fold system as players. If
`mob.Character.IsCasting()` is true (Activity machine is in Casting state),
folds accumulate. On completion, spell resolves via `resolveMobSpell()`.

#### 4c. AI Decision: `handleMobAIDecision()`

`hooks/NewRound_DoCombat_helpers.go` — The mob decides whether to use a
special ability instead of basic melee:

```
if rand(100) < mob.ActivityLevel:
  1. Try ChooseCastAction(mob) -- picks a spell if available, off cooldown
  2. If no spell: ChooseSpecialMove(mob, target) -- bash, grapple, etc.
  3. If chosen: mob.Command(chosenMove), return true (skip melee)

Fallback: CombatCommands list
  if rand(100) < mob.ActivityLevel:
    Pick random CombatCommand, execute, return true
```

The mob might bash the player (using the same `Bash()` function, which
sets `RoundsWaiting = 1` on the mob's aggro), cast a spell, or execute a
scripted combat command.

#### 4d. MvP Dispatch: `handleMobVsPlayer()`

`hooks/NewRound_DoCombat_helpers.go` — If the mob does normal melee:

1. **Target validation** — player exists, same room.
2. **Downed grace** — `handleMobDownedGrace()`: if the player is
   disabled (HP/STA/CONV <= 0), the mob circles for
   `CoupDeGraceRounds` before delivering a finishing blow.
3. **Hidden check** — can't hit hidden players.
4. **Reciprocal aggro** — if player wasn't already fighting this mob,
   set their aggro.
5. **Party auto-attack** — `handlePartyAutoAttack()`: party members
   auto-engage the mob.
6. **Grapple progression** — same as player side.
7. **Target switch AI** — 10% chance the mob switches to a different
   player in the room (requires combat skill >= 30).
8. **Weapon pickup** — if disarmed, tries to equip a weapon from
   inventory.
9. **Rounds waiting** — if mob used bash last round, show wind-up
   message and skip this round.
10. **Moon mods** — apply to the defending player (mutated characters
    get stat boosts).
11. **THE ATTACK** — `combat.AttackMobVsPlayer(mob, defUser)`:
    Same `calculateCombat()` pipeline but with Mob as source, User as
    target. `MobDamageMultiplier` config scales mob damage. Player's
    defense sequence: dodge (DEX), parry (weapon), block (shield) —
    the shield is critical here, it provides `DefenseBlock` with a
    high effectiveness.
    - **Defender progression:** Player gets `OnStatUse("dexterity")`
      for reacting to attacks.
    - **Resource depletion progression:** Moved to regen tick in
      `NewRound_AutoHeal.go` — smooth curve replaces old 25% threshold.
      See `characters/context.md` for details.
12. **Minor Shield reduction** — if player has `ConditionShield`, flat
    damage reduction.
13. **Adrenaline Surge** — mutation check for bonus damage.
14. **Crit effects (defender is player):**
    - Parry crit: player attempts to disarm the mob.
    - Dodge crit: player gets a grapple opportunity.
15. **Charmed mob assist** — charmed mobs in room help the player.
16. **Apply buffs and messages** to player and room.
17. **Concentration break** — if player was casting and got hit,
    Willpower vs damage% check to see if concentration holds.
18. **Offhand break** — chance the player's shield gets damaged.
19. **Mob attacker progression** (Stage 38.3) — if `MobProgressionEnabled`:
    - `mob.Character.OnStatUse("strength")` / `OnStatUse("dexterity")`
    - If hit: `mob.Character.OnSkillUse(combatSkill)`
    - If crit: `mob.Character.OnCriticalSuccess(combatSkill)`
    - If fumble: `mob.Character.OnCriticalFailure(combatSkill)`
20. **End check** — if either at 0 HP, end aggro.

### 5. Resolution: `handleAffected()`

`hooks/NewRound_DoCombat.go` — After all player and mob combat resolves:

**For each affected player:**
```
if Health <= -10 OR Stamina <= -10 OR Conviction <= -10:
  user.Command("suicide")  // death: drops items, land of the dead

else if IsDisabled() (any pool <= 0):
  events.AddToQueue(PlayerDrop)  // drop items, bleeding out state
```

**For each affected mob:**
```
if Health < 1:
  mob.Command("suicide")  // death: drops loot, despawns
```

### 6. Bash Cooldown Flow (Specific Example)

**Round N:** Player types `bash`.
- `usercommands.Bash()` executes immediately (not during the round tick).
- Checks shield equipped, checks `special-move` cooldown.
- Opposed roll: `(WeaponCombat + Strength)` vs `(mobSkill + DEX)`.
- If hit: `CalcRawDamage(STR, weaponCombatRank, BashDamagePercent,
  ChannelPhysical)` with mitigation applied.
- If knockdown roll succeeds: mob goes `PositionProne` for 2+ rounds.
- Sets `user.Character.Aggro.RoundsWaiting = 1` (the attack cost).

**Round N+1:** `handlePlayerVsMob()` fires during the tick.
- `RoundsWaiting > 0`: decrements to 0, shows wind-up message via
  `GetWaitMessages()`. Player does NOT get a normal attack this round.

**Round N+2:** Player attacks normally again (`RoundsWaiting == 0`).
- Full `calculateCombat()` runs.
- If mob is still `PositionProne` from the bash:
  - Player gets `ProneVulnerabilityMultiplier` (1.15x) bonus to attack.
  - Mob defense scores get `ProneDodge/Parry/BlockPenalty`.
  - Mob attack gets `ProneAttackMultiplier` (0.80x) and
    `ProneDamagePenalty`.

### 7. Difficulty Display (`descriptions.go`)

The `GetDifficultyDescription(difficulty int)` function converts spell
difficulty integers (0-75) into qualitative labels for player-facing display:
trivial, simple, moderate, challenging, demanding, formidable, masterwork.
Used in spell UX to communicate challenge without exposing numeric difficulty
values directly.

### 8. File Map After Refactor (Stage 37.1a+)

| File | Contents |
|------|----------|
| `combat/combat.go` | `calculateCombat()` orchestrator (~80 lines), `AttackPlayerVsMob`, `AttackPlayerVsPlayer`, `AttackMobVsPlayer`, `AttackMobVsMob`, `GetWaitMessages` |
| `combat/combat_helpers.go` | Extracted helpers: `calcAttackCount`, `collectAttackWeapons`, `buildWeaponSetup`, `buildDamageParams`, `calcAttackScore`, `calcCritThreshold`, `calcDualWieldPenalty`, `filterDefensesForThirdParty`, `runBestOfAllDefense`, `resolveDefenseOutcome`, `calcHitDamage`, `buildAttackMessages`, `applyPetDamage` |
| `combat/damage_pipeline.go` | `CalcRawDamage`, `ApplyMitigation`, `SkillMultiplier`, `ResourceMultiplier`, `MitigationCap`, `DamageScale` |
| `combat/attackresult.go` | `AttackResult` struct (includes `DefenseAttempts`, `AttackZScore`, `DefenseZScore`, `ParryCritDetected`, `DodgeCritDetected`) and message helpers |
| `combat/ai.go` | `ChooseSpecialMove`, `ChooseCastAction`, `GetAIProfile`, AI profiles, viability checks (`CanUseBash`, `CanUseKick`, etc.), scoring functions |
| `combat/criteffects.go` | `AttemptCritDisarm`, `SetGrappleOpportunity`, `HasGrappleOpportunity`, `GetGrappleOpportunityBonus`, `ClearGrappleOpportunity` |
| `combat/grapple.go` | `AttemptGrapple`, `ApplyGrappleResult`, `CheckClinchProgression`, `CheckGroundedEscape`, `ApplyPositionProgression`, `IsThirdPartyAttack`, `AttemptSubmission` |
| `combat/grapple_move.go` | `ExecuteGrappleMove`, `GrappleMoveResult`, `GrappleMoveDisarmWeapon` |
| `combat/skill_moves.go` | `ExecuteSkillMove`, `SkillMoveResult`, `SkillMoveParams` |
| `combat/calculations.go` | Hit chance, crit probability, power ranking, alignment calculations |
| `combat/descriptions.go` | `GetDamageDescription`, `GetHealDescription`, `GetDifficultyDescription` helpers |
| `combat/taunt_messages.go` | Taunt/conviction combat messages |
| `combat/analytics.go` | Ring buffer, `CombatEvent`, `AnalyticsSummary`, recording + query functions |
| `hooks/NewRound_DoCombat.go` | `DoCombat`, `handlePlayerCombat` (~50 lines), `handleMobCombat` (~50 lines), `processGrappleProgression`, `handleAffected`, `applyMoonMods` |
| `hooks/NewRound_DoCombat_helpers.go` | All extracted helpers: `handlePlayerShieldDecay`, `handlePlayerFoldCasting`, `handleMobFoldCasting`, `handlePlayerFlee`, `handlePlayerVsPlayer`, `handlePlayerVsMob`, `handleMobVsPlayer`, `handleMobVsMob`, `handleMobAIDecision`, `handleMobTargetSwitch`, `handleMobWeaponPickup`, `handleMobDownedGrace`, `handlePartyAutoAttack`, `handleCharmedMobAssist`, `handleAutoRetargetPlayer`, `handlePlayerConcentrationBreak`, `dispatchCombatMessages`, `handleOffhandBreakUserDef`, `handleOffhandBreakMobDef` |
| `hooks/combat_shared_helpers.go` | `simulateFoldRound`, `calcFoldConvictionCost`, `advanceFolds`, `checkConcentrationBreak`, `tryWeaponBreak`, `applyCritEffects`, `CritEffectResult`, `calcSpellDamageForCharacter` |
| `hooks/spell_resolution.go` | `resolveSpell`, `resolveAgainstMob`, `resolveAgainstPlayer`, `applyPlayerEffect` |
