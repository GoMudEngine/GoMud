# DOGMud Spells System Context

## Overview

The DOGMud spells system provides a comprehensive magic framework with 45 spells across
multiple types and schools, a fold-based casting mechanic, use-based spell discovery, and
flexible targeting. Spells use Conviction (not mana) as their resource.

**DOGMud Differences from upstream GoMud:**
- Mana removed — spells use Conviction resource
- Optional Health costs for life-force magic (vital school spells may sacrifice health)
- Spell cost scaling via config multipliers (SpellConvictionCostMultiplier, SpellHealthCostMultiplier)
- Spellcasting skill (DOG) replaces legacy Cast skill for combat resolution
- Schools changed from single value to array (can have multiple schools)
- Four DOG schools: Elemental, Enhancement, Mental, Vital
- Use-based spell discovery (Phase 25) — no trainers, spells emerge through casting practice
- New effect types: dot, knockdown, purge (Phase 25)
- Starting spells reduced to 1 (Conviction Spike only)

---

## Architecture

### Core Components

**Spell Data Management:**
- Unique spell identification with string-based IDs
- YAML-based storage with automatic loading and validation
- Spell discovery by name or ID with fuzzy matching
- In-memory caching for fast spell lookups

**Spell Classification System:**
- Type-based targeting (single, multi, area, neutral)
- School-based categorization (elemental, enhancement, mental, vital)
- Harm/help classification for spell effects
- Difficulty scaling for success calculations

---

## SpellData Structure

```go
type SpellData struct {
    SpellId           string    // Unique spell identifier (also filename base)
    Name              string    // Display name
    Description       string    // Spell description
    Type              SpellType // Targeting and effect type
    Schools           []string  // Magic school classification (can have multiple)
    Cost              int       // Conviction cost
    HealthCost        int       // Optional Health cost for vital school
    WaitRounds        int       // Casting delay in rounds
    Difficulty        int       // Success modifier (0-100%)
    PrimaryStat       string    // Stat used for spell power scaling
    BaseFolds         int       // Number of folds required to cast
    TargetDefenseType string    // "physical", "mental", or "none"
    EffectType        string    // "damage", "heal", "shield", "dot", "knockdown", "purge", "none"
    EffectMagnitude   int       // Base power of the effect
    EffectDuration    int       // For DoT: number of tick cycles
}
```

### Spell Difficulty and Target Types

**Difficulty Field:** The `Difficulty` integer (range 0–75) affects skill
progression via difficulty-scaled bonus multiplier (applied in spell resolution).
Values: 0 (utility), 1–15 (weak combat), 15–30 (moderate), 30–50 (strong),
50–75 (apex combat spells).

**TargetTypeString:** `Neutral` spells now return "Self" instead of "Unknown"
for display purposes.

### Spell Types
```go
const (
    Neutral    SpellType = "neutral"    // No expected target
    HarmSingle SpellType = "harmsingle" // Single harmful target
    HarmMulti  SpellType = "harmmulti"  // Multiple harmful targets
    HelpSingle SpellType = "helpsingle" // Single beneficial target
    HelpMulti  SpellType = "helpmulti"  // Multiple beneficial targets
    HarmArea   SpellType = "harmarea"   // Area harmful effect
    HelpArea   SpellType = "helparea"   // Area beneficial effect
)
```

### Magic Schools
```go
const (
    SchoolElemental   = "elemental"   // Fire, ice, lightning, earth — offensive elemental magic
    SchoolEnhancement = "enhancement" // Buffs, shields, enchantments — augmentation magic
    SchoolMental      = "mental"      // Illusions, charms, telepathy — mind-affecting magic
    SchoolVital       = "vital"       // Healing, curing, life/death manipulation
)
```

---

## Effect Types (spell_resolution.go)

| EffectType | Behavior |
|------------|----------|
| `damage` | Direct HP damage to target(s), scaled by EffectMagnitude and PrimaryStat |
| `heal` | Direct HP restoration to target(s) |
| `shield` | Applies ConditionShield with magnitude = damage absorbed |
| `dot` | Applies ConditionPoisoned; ticks for EffectDuration cycles (each cycle = 3 rounds in AutoHeal) |
| `knockdown` | Deals damage + knocks the target down. Chunk 4b W5 cutover fires `Position.TransitionToSupine(MinRecoveryRounds: 1, TriggerKnockdownSpell)` (face-up "slams to the ground" default) alongside the legacy `CombatPosition = PositionProne` parallel-write. Future work may add a direction config to distinguish blast (Supine) from shockwave (Prone). |
| `purge` | Removes poison buffs and ConditionPoisoned from target(s) |
| `none` | No automatic effect — spell behavior handled in Go hooks (used by buff spells, summons, utility) |

---

## Spell Discovery System (Phase 25)

Spells are learned through casting, not trainers. After each successful cast:

1. Base discovery chance: ~5% per cast
2. Scaled down by known spell count: `chance = baseChance / (1 + knownCount * 0.1)`
3. `GetEligibleSpells()` returns unlearned spells whose `BaseFolds` ≤ skill-gated threshold
4. Random selection from eligible pool
5. Player receives: "A new pattern crystallizes in your mind: <spell name>"

### Fold Threshold Table (MaxFoldsForSkill)
| Skill Level | Max Discoverable Folds |
|-------------|----------------------|
| 1–4 | ≤ 4 |
| 5–9 | ≤ 6 |
| 10–19 | ≤ 8 |
| 20–29 | ≤ 10 |
| 30–39 | ≤ 12 |
| 40–49 | ≤ 16 |
| 50–59 | ≤ 20 |
| 60–69 | ≤ 24 |
| 70–79 | ≤ 28 |
| 80+ | ≤ 32 |

### Starting Spells
New characters begin with only `mm` (Conviction Spike). All other spells are
discovered through casting practice.

---

## Buff Integration (Phase 25.3)

Several buff flags affect spell and combat systems:

| Flag | Effect | Applied In |
|------|--------|-----------|
| `damage-bonus` | +15% physical damage | NewRound_DoCombat.go |
| `haste` | Speed effects | Various |
| `slow` | Movement penalty | Various |
| `skill-progress` | 2x skill progression chance | progression.go |
| `mutation-rate` | 2x mutation progress gain | UserRoundTick.go |

---

## Summon Spells (Phase 25.4)

Two permanent summon spells use components, resolved in Go hooks:
- `chrysalis-construct` (20 folds) — requires Chrysalis Core (item 40010), spawns mob 110
- `summon-hive-swarm` (24 folds) — requires Hive Fragment (item 40011), spawns mob 111

Summons persist until killed, one per type per caster. Go hooks in
`spell_resolution.go` handle component checking/consumption, mob spawning,
and permanent charm via `CharmSet(userId, 99999)`.

---

## All Spells (45 total)

### Re-Themed Original Spells (14)
mm (Conviction Spike), fire-bolt (Pyretic Surge), heal (Mend Flesh),
fireball (Hemorrhagic Burst), minor-shield (Conviction Ward),
curepoison (Purge Affliction), tame (Empathic Bond), sparks (Conviction Sparks),
stun (Neural Stun), blind (Sensory Veil), throw-stone (Kinetic Hurl),
illum (Chrysalis Glow), healall (Mend All), aidskill (Chrysalis Aid).

### Damage/Heal/DoT/Shield Spells (12)
mind-spike, kinetic-shove, blood-boil, hemorrhagic-wave, synaptic-overload,
veil-rend, mend-wounds, communion-of-flesh, chrysalis-cocoon, neural-toxin,
conviction-barrage, cleansing-wave.

### Buff/Debuff/Utility Spells (17)
conviction-surge, iron-will, chrysalis-haste, mind-fog, nerve-disruption,
empathic-shroud, vital-surge, chrysalis-regeneration, skill-attunement,
mutation-catalyst, psychic-anchor, sensory-overload, conviction-armor,
veil-sight, fold-anchor (set/recall toggle), mass-mend.

### Summon Spells (2)
chrysalis-construct, summon-hive-swarm.

---

## Hook Integration Points

| Hook File | What It Does |
|-----------|-------------|
| `internal/hooks/spell_resolution.go` | Effect dispatch (damage, heal, shield, dot, knockdown, purge), HelpArea targeting |
| `internal/hooks/NewRound_DoCombat.go` | Spell discovery after cast, DamageBonus buff check |
| `internal/hooks/NewRound_AutoHeal.go` | Mob poison DoT ticking |
| `internal/hooks/NewRound_UserRoundTick.go` | MutationRate buff check |
| `internal/characters/progression.go` | SkillProgress buff check (2x casting skill gain) |

---

## Files in This Package

| File | Purpose |
|------|---------|
| `spells.go` | SpellData struct, registry, loader, GetEligibleSpells(), MaxFoldsForSkill() |
| `context.md` | This file — package overview for Claude Code |

---

## Stage Roadmap

- **Phase 25.1** (complete) — Re-themed 14 spells, Go infrastructure (dot/knockdown/purge), spell discovery, HelpArea fix
- **Phase 25.2** (complete) — 12 new damage/heal/DoT/shield spells
- **Phase 25.3** (complete) — 13 new buffs, 17 new buff/debuff/utility spells, hook integration
- **Phase 25.4** (complete) — 2 summon spells with component items and permanent charm
