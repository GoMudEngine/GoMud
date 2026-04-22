# DOGMud Progression Data Analysis

## 1. Admin Dashboard Architecture

### Web Handler Structure
- **Entry Point**: `/c/Users/Calabe Davis/workspace/DOGMud/internal/web/web.go` (line 288-345)
  - Router setup with authentication middleware `doBasicAuth()`
  - All handlers wrapped with `RunWithMUDLocked()` for thread-safe MUD access
  - Plugin system via `WebPlugin` interface for extending nav/pages

### Existing Admin Tabs (as defined in _header.html)
1. **Dashboard** → `/admin/` (handler: `adminIndex`)
2. **Items** → `/admin/items/` (handler: `itemsIndex`, `itemData`)
3. **Species** → `/admin/species/` (handler: `speciesIndex`, `speciesData`)
4. **Mobs** → `/admin/mobs/` (handler: `mobsIndex`, `mobData`)
5. **Mutators** → `/admin/mutators/` (handler: `mutatorsIndex`, `mutatorData`)
6. **Rooms** → `/admin/rooms/` (handler: `roomsIndex`, `roomData`)
7. **Combat Stats** → `/admin/combat-stats/` (handler: `combatStatsIndex`, `combatStatsAPI`)

### Template System
- **Header**: `/c/Users/Calabe Davis/workspace/DOGMud/_datafiles/html/admin/_header.html` (defines sidebar nav, includes Bootstrap 4 + HTMX)
- **Index**: `/c/Users/Calabe Davis/workspace/DOGMud/_datafiles/html/admin/index.html` (minimal, just includes header+footer)
- **Footer**: `/c/Users/Calabe Davis/workspace/DOGMud/_datafiles/html/admin/_footer.html`
- **Combat Stats Template**: `/c/Users/Calabe Davis/workspace/DOGMud/_datafiles/html/admin/combatstats/index.html` (example pattern with summary cards, filters, charts)

### Key Handler Pattern (Combat Stats as example)
1. **Index Handler** (`combatStatsIndex`): Renders HTML template
   - `/c/Users/Calabe Davis/workspace/DOGMud/internal/web/admin.combatstats.go` line 71
   - Parses: `_header.html` → `combatstats/index.html` → `_footer.html`
   - Passes empty data struct to template

2. **API Handler** (`combatStatsAPI`): Returns JSON
   - `/c/Users/Calabe Davis/workspace/DOGMud/internal/web/admin.combatstats.go` line 101
   - Accepts query params (filters)
   - Calls business logic (`combat.GetFilteredSummary()`)
   - Returns `combatStatsAPIResponse` struct (line 14-37)

3. **HTML + JavaScript**: Uses HTMX + Chart.js for frontend
   - Filters trigger `fetchData()` → GET `/admin/api/combat-stats/` → JSON → Charts updated

---

## 2. Skill/Stat Progression Storage

### Character Structure
**File**: `/c/Users/Calabe Davis/workspace/DOGMud/internal/characters/character.go` (line 51-119)

#### Stats Storage (line 58)
```go
Stats            stats.Statistics               // Character stats
```

**StatInfo structure** (`/c/Users/Calabe Davis/workspace/DOGMud/internal/stats/stats.go` line 20-27):
```go
type StatInfo struct {
    Training int `yaml:"training,omitempty"` // Use-based progression gains
    Value    int `yaml:"-"`                  // Final calculated value (Base + Training + Mods)
    ValueAdj int `yaml:"-"`                  // Adjusted value (after softcap)
    Racial   int `yaml:"-"`                  // Racial benefits
    Base     int `yaml:"base,omitempty"`     // Base starting value
    Mods     int `yaml:"-"`                  // Equipment/buff modifiers
}
```

**Statistics struct** (6 stats):
- `Strength`, `Dexterity`, `Perception`, `Vitality`, `Willpower`, `Charisma`
- All calculated via `Recalculate()` (line 40-58) which applies softcap formula

#### Skills Storage (line 92)
```go
Skills                   map[string]int                 `yaml:"skills,omitempty"`        // skill_tag → rank level
```
- Keys: skill tags like `"weapon-combat"`, `"spellcasting"`, `"salvage"`, etc.
- Values: rank level (integer, starts at 1)

#### Progression Tracking (line 106-107)
```go
SkillUseCount    map[string]int                 `yaml:"skillusecount,omitempty"` // Tracks uses per skill
StatUseCount     map[string]int                 `yaml:"statusecount,omitempty"`  // Tracks uses per stat (typo: "statusecount")
```

### Progression Functions
**File**: `/c/Users/Calabe Davis/workspace/DOGMud/internal/characters/progression.go`

#### Stat Use Tracking (line 210-221)
```go
func (c *Character) OnStatUse(statName string, userId int) bool {
    c.TrackStatUse(statName)                    // Increment StatUseCount[statName]
    if configs.GetGamePlayConfig().UseSkillProgression {
        return c.CheckStatProgression(statName, userId, 1.0)
    }
    return false
}

func (c *Character) TrackStatUse(statName string) {
    if c.StatUseCount == nil {
        c.StatUseCount = make(map[string]int)
    }
    c.StatUseCount[statName]++
}

func (c *Character) GetStatUseCount(statName string) int {
    if c.StatUseCount == nil {
        return 0
    }
    return c.StatUseCount[statName]
}
```

#### Skill Use Tracking (line 194-200, 227-250)
```go
func (c *Character) OnSkillUse(skillName string, userId int) bool {
    c.TrackSkillUse(skillName)                  // Increment SkillUseCount[skillName]
    gained := false
    if configs.GetGamePlayConfig().UseSkillProgression {
        gained = c.CheckSkillProgression(skillName, userId, 1.0)
    }
    // Auto-track primary stat for the skill
    if primaryStat := skills.GetSkillPrimaryStat(skillName); primaryStat != "" {
        c.OnStatUse(primaryStat, userId)
    }
    return gained
}

func (c *Character) TrackSkillUse(skillName string) {
    if c.SkillUseCount == nil {
        c.SkillUseCount = make(map[string]int)
    }
    c.SkillUseCount[skillName]++
}

func (c *Character) GetSkillUseCount(skillName string) int {
    if c.SkillUseCount == nil {
        return 0
    }
    return c.SkillUseCount[skillName]
}
```

#### Progression Check (line 63-142)
```go
func (c *Character) CheckSkillProgression(skillName string, userId int, bonusMultiplier float64) bool {
    // 1. Mob-specific gating check
    // 2. Calculate virtual rank: adjustedUseCount / UsesPerRank (config)
    // 3. Roll against chance using: CalculateProgressionChance(virtualRank, softCap)
    //    - Exponential decay: base(0.30) * exp(-decayBelow * ratio)
    //    - Includes: mutation bonuses, buff multipliers
    // 4. On success: call c.IncreaseSkill(skillName)
    //    - Increments Skills[skillName] by 1
    //    - Checks if rank description changed
    //    - Sends event message to player
    return true/false
}

func (c *Character) CheckStatProgression(statName string, userId int, bonusMultiplier float64) bool {
    // Same flow as skill progression
    // 1. Calculate virtual rank from StatUseCount
    // 2. Roll against CalculateProgressionChance()
    // 3. On success: call c.IncreaseStat(statName, 1)
    //    - Increments Stats.<StatName>.Training by 1
    //    - Calls c.Validate() to recalculate derived values
    return true/false
}
```

#### Stat/Skill Increase (character.go line 1951-1982)
```go
func (c *Character) IncreaseSkill(skillName string) bool {
    if c.Skills == nil {
        c.Skills = make(map[string]int)
    }
    oldLevel := c.Skills[skillName]
    c.Skills[skillName] = oldLevel + 1
    newLevel := c.Skills[skillName]
    // Returns true if rank description changed (e.g., "Novice" → "Apprentice")
    return skills.GetSkillRankDescription(newLevel) != skills.GetSkillRankDescription(oldLevel)
}

func (c *Character) IncreaseStat(statName string, amount int) bool {
    // Updates: Stats.<StatName>.Training += amount
    // Calls: c.Validate() to recalculate Value/ValueAdj with softcap
    switch statName {
    case "strength":
        c.Stats.Strength.Training += amount
    // ... (6 cases)
    }
    c.Validate()
    return true
}
```

---

## 3. Data Available for Display

### Per-Player Character Data
Access: `/c/Users/Calabe Davis/workspace/DOGMud/internal/users/users.go`

#### Getting All Users
```go
func GetAllActiveUsers() []*UserRecord {
    ret := []*UserRecord{}
    for _, userPtr := range userManager.Users {
        if !userPtr.isZombie {
            ret = append(ret, userPtr)
        }
    }
    return ret
}

func GetByUserId(userId int) *UserRecord {
    if user, ok := userManager.Users[userId]; ok {
        return user
    }
    return nil
}
```

#### UserRecord Structure
**File**: `/c/Users/Calabe Davis/workspace/DOGMud/internal/users/userrecord.go`
- Contains: `Character *characters.Character` field
- Can extract: `user.Character.Stats`, `user.Character.Skills`, `user.Character.SkillUseCount`, `user.Character.StatUseCount`

### Available Progression Metrics

#### Stat Data
For each of 6 stats (strength, dexterity, perception, vitality, willpower, charisma):
- **Current Rank**: `c.Stats.<Stat>.Value` (final value after mods/softcap)
- **Training Points**: `c.Stats.<Stat>.Training` (use-based gains)
- **Use Count**: `c.GetStatUseCount("<stat-name>")` → total times stat was checked
- **Virtual Rank**: `useCount / UsesPerRank` (config-driven)
- **Progress to Next**: `useCount % UsesPerRank` (for progress bar)
- **Base**: `c.Stats.<Stat>.Base`
- **Mods**: `c.Stats.<Stat>.Mods` (from equipment/buffs)

#### Skill Data
For each skill in `c.Skills` map:
- **Current Rank**: `c.Skills[skillName]` (integer level)
- **Rank Description**: `skills.GetSkillRankDescription(rank)` (e.g., "Novice", "Adept")
- **Use Count**: `c.GetSkillUseCount(skillName)` → total times skill was used
- **Virtual Rank**: `useCount / UsesPerRank`
- **Progress to Next**: `useCount % UsesPerRank`
- **Primary Stat**: `skills.GetSkillPrimaryStat(skillName)` (e.g., weapon-combat → strength)

#### Progression Configuration
**File**: `/c/Users/Calabe Davis/workspace/DOGMud/internal/characters/progression.go` (line 43-61)

Key config values (from `balance.yaml`):
- `UsesPerRank`: How many uses before virtual rank increases
- `BaseProgressionChance`: Base roll chance (approx 30%)
- `ProgressionDecayBelowCap`: Exponential decay rate below soft cap
- `ProgressionDecayAboveCap`: Decay rate above soft cap
- `StatSoftCap`: Hard cap for linear growth (default 150)
- `SkillSoftCap`: Hard cap for linear growth (default 50)
- `StatSoftCapThreshold`: Threshold before softcap math applies (default 105)
- `MobProgressionEnabled`, `MobStatCap`, `MobSkillCap`: Mob-specific gating
- `UseSkillProgression`: Global on/off toggle

#### Progression Chance Formula
```
CalculateProgressionChance(virtualRank, softCap) →
  base = 0.30 (30%)
  if rank ≤ softCap:
    chance = base × exp(-decayBelow × (rank / softCap))
  else:
    chance = aboveCapFloor × exp(-decayAbove × (rank - softCap) / softCap)
```
Results in: approx 30% at rank 0, approx 1.5% at soft cap, asymptoting thereafter.

### Combat Analytics Structure (Existing Pattern)
**File**: `/c/Users/Calabe Davis/workspace/DOGMud/internal/combat/analytics.go`

Can follow same pattern for progression:
```go
type CombatEvent struct {
    // Tracks per-event metrics
}

var (
    analyticsReady bool
    eventBuffer    []CombatEvent
    maxEvents      int
    logWriter      *lumberjack.Logger
)

// Event recording, ring buffer, JSON export
```

---

## 4. Key Integration Points

### User Retrieval in Web Handlers
```go
// From any web handler (locked with util.LockMud()):
users := users.GetAllActiveUsers()  // []*UserRecord
for _, user := range users {
    char := user.Character
    skills := char.Skills              // map[string]int
    skillUses := char.SkillUseCount    // map[string]int
    statUses := char.StatUseCount      // map[string]int
    stats := char.Stats                // stats.Statistics
}
```

### Progression Events
When progression fires:
1. `CheckSkillProgression()` or `CheckStatProgression()` returns `true`
2. Event added to queue: `events.AddToQueue(events.Message{...})`
3. Messages NOT persisted — only show to current session
4. No built-in history/logging system exists (opportunity for new feature)

### Config Access
```go
b := configs.GetBalanceConfig()  // progression config
g := configs.GetGamePlayConfig() // UseSkillProgression toggle
```

---

## 5. Data File Locations

### User Character Files
**Path**: `_datafiles/users/{userId}.yaml`
- YAML serialization includes: `Skills`, `SkillUseCount`, `StatUseCount`
- Stats stored via `Training` field (only persisted value)
- Other fields (`Value`, `ValueAdj`) recalculated on load

### Combat Analytics Logs
**Path**: Configured in `analytics.yaml` (e.g., `logs/analytics.jsonl`)
- JSON Lines format (one summary per flush)
- Ring buffer system with configurable max events

---

## 6. What's NOT Tracked

- **Progression history**: No timestamp/log of when skills/stats increased
- **Event details**: No per-progression-check logging
- **Player analytics**: No per-player metrics aggregation
- **Regen progression**: Smooth curve, separate from use-based (OnRegenTick)
- **Critical events**: Tracked separately (OnCriticalSuccess, OnCriticalFailure, OnFirstMobKill)

---

## Summary: Data Available for Progression Dashboard

| Item | Storage | Access | Notes |
|------|---------|--------|-------|
| Character | User → Character | `users.GetAllActiveUsers()` | Pointer to active user |
| Skill Level | `Character.Skills[name]` | Direct access | Integer rank |
| Skill Uses | `Character.SkillUseCount[name]` | `GetSkillUseCount()` | Incremented on use |
| Stat Value | `Character.Stats.<Stat>.Value` | Direct access | Final value (after softcap) |
| Stat Training | `Character.Stats.<Stat>.Training` | Direct access | Use-based gains only |
| Stat Uses | `Character.StatUseCount[name]` | `GetStatUseCount()` | Incremented on use |
| Progression Config | Balance config | `configs.GetBalanceConfig()` | All formulas/thresholds |
| Progression Events | Events queue | Hook into `OnSkillUse`, `OnStatUse` | Not persisted |

All data is **in-memory during session**. No persistent progression history unless you implement new logging.
