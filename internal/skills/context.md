# DOGMud Skills System Context

## Overview

The DOGMud skills system provides a dual-layer character development framework combining 10 core DOG skills with 15 legacy GoMud skills, organized into 10 professions. Skills improve through use-based progression (not training points or level-ups). The system features profession-based titles, weapon-aware combat skill routing, and a progression curve with exponential decay.

**DOGMud Differences from upstream GoMud:**
- 10 new DOG skills added (5 combat + 5 non-combat)
- Use-based progression replaces manual training point allocation
- Combat skills are weapon-aware (auto-select based on equipped weapon)
- Skill name aliasing infrastructure exists for legacy→DOG mapping (not yet active)
- All characters start with every registered skill at rank 1

## Skill Sets

### DOG Combat & Magic Skills (Stage 3.7)
| Skill Tag | Covers |
|-----------|--------|
| `weapon-combat` | Melee attack & defense with weapons, parrying |
| `unarmed-combat` | Fist/body attacks & defense, grappling |
| `ranged-combat` | Bows, crossbows, thrown weapons |
| `spellcasting` | All magic — elemental, enhancement, vital schools |
| `psionics` | Mental powers — telepathy, telekinesis, illusion |

### DOG Non-Combat Skills (Stage 3.8+)
| Skill Tag | Covers |
|-----------|--------|
| `first-aid` | Healing others, treating wounds, stabilizing |
| `stealth` | Sneaking, hiding, avoiding detection |
| `search` | Finding hidden objects, creatures, exits; tracking and foraging |
| `bartering` | Trade prices, negotiation, appraisal |

### Legacy GoMud Skills (13 total — 2 removed in discovery system)
```go
Cast, DualWield, Map, Enchant, Peep, Inspect, Portal,
Skulduggery, Brawling, Scribe, Protection, Tame, Trading
```

**Removed in search system:**
- `Track` — merged into `search` skill
- `Foraging` — merged into `search` skill

Legacy skill `Skulduggery` still exists separately (overlaps with `stealth` but aliasing is deferred).

## Architecture

### Skill Tag System
```go
type SkillTag string

// Skills use kebab-case string tags: "weapon-combat", "first-aid", etc.
// Subtag support: skill.Sub(":fireball") → "cast:fireball"
```

### Skill Registration
```go
func init() {
    // 1. Collect legacy skills from Professions map
    // 2. Explicitly register all 10 DOG skills:
    //    WeaponCombat, UnarmedCombat, RangedCombat, Spellcasting, Psionics,
    //    FirstAid, Stealth, Tracking, Bartering, Foraging
    // Total: ~25 registered skills
}
```

### Character Integration
- All characters start with every registered skill at rank 1 (`initAllSkills()`)
- Skills stored as `map[string]int` (skill name → rank)
- `GetSkillLevel(tag)` returns current rank
- `GetCombatSkillTag()` routes to weapon-appropriate combat skill
- `GetCombatSkillLevel()` returns combat skill rank with Brawling fallback
- **Migration on load**: `search = max(tracking, foraging)`, old skills removed (runs before `ensureAllSkills()` in `Validate()`)

## Progression System (`characters/progression.go`)

### Use-Based Progression
- Skills improve through gameplay use, not manual training
- `OnSkillUse(skillName, userId)` triggers progression check
- Exponential decay curve: ~50% at rank 0, ~2.5% at soft cap (rank 50)
- Critical successes get 2x progression bonus
- Virtual ranks: every 10 uses = 1 virtual rank for progression math

### Progression Curve
```
CalculateProgressionChance(currentRank, softCap=50):
  Rank 0:  50%
  Rank 10: ~27%
  Rank 25: ~11%
  Rank 40: ~4.5%
  Rank 50: ~2.5% (at soft cap)
  Rank 60: ~0.6% (above soft cap)
```

### Skill Name Aliasing
```go
// skillNameMap in progression.go can map legacy → DOG names
// Currently empty — aliasing deferred to future stage
var skillNameMap = map[string]string{}
```

## Profession System

### Profession Definitions
```go
var Professions = map[string][]SkillTag{
    "treasure hunter": {Map, Search, Peep, Inspect, Trading},
    "assassin":        {Skulduggery, DualWield, Track},
    "explorer":        {Map, Portal, Scribe},
    "arcane scholar":  {Enchant, Scribe, Inspect},
    "warrior":         {Brawling, DualWield},
    "paladin":         {Protection, Brawling},
    "ranger":          {Map, Search, Track},
    "monster hunter":  {Tame, Track},
    "sorcerer":        {Cast, Enchant},
    "merchant":        {Peep, Trading},
}
```

### Experience Titles
Based on profession completion percentage:
- `scrub` (< 10%)
- `novice` (≥ 10%)
- `apprentice` (≥ 30%)
- `journeyman` (≥ 60%)
- `expert` (≥ 90%)
- `demigod` (all professions mastered)

## Search Skill Details

The `search` skill replaces and consolidates the legacy `Track` and `Foraging` skills:

### Governing Stat
- **Perception** — base attribute driving discovery effectiveness

### Commands Using Search
- **`search`** — Find hidden objects, containers, exits, creatures in current room
- **`track`** — Follow hidden tracks of creatures/players
- **`forage`** — Gather hidden resources from environment

All three use the unified search score formula: `dice.RollStat(Perception + SkillMultiplier(searchRank) * 25.0)`

### Migration Path
When a character loads from disk:
1. Check if legacy `Track` and/or `Foraging` skills exist
2. Set `search = max(track_rank, foraging_rank)`
3. Remove `Track` and `Foraging` from skills map
4. This runs in `Validate()` **before** `ensureAllSkills()` so migration completes before missing skills are auto-added

### Soft Cap
- Default soft cap: 50 (same as other skills)
- Progression curve unchanged from other use-based skills

## Dependencies

- `strings` - String manipulation for skill tags and profession names
