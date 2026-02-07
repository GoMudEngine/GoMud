# DOGMud Spells System Context

## Overview

The DOGMud spells system provides a comprehensive magic framework with support for multiple spell types, schools of magic, difficulty scaling, scripting integration, and flexible targeting systems. It features spell discovery, automatic script template generation, and seamless integration with the character and combat systems.

**DOGMud Differences from upstream GoMud:**
- Mana removed — spells currently cast without resource cost (Conviction costs coming in Phase 6)
- Spellcasting skill (DOG) replaces legacy Cast skill for combat resolution
- Psionics skill added for mental powers

## Architecture

The spells system is built around several key components:

### Core Components

**Spell Data Management:**
- Unique spell identification with string-based IDs
- YAML-based storage with automatic loading and validation
- Spell discovery by name or ID with fuzzy matching
- In-memory caching for fast spell lookups

**Spell Classification System:**
- Type-based targeting (single, multi, area, neutral)
- School-based categorization (restoration, illusion, conjuration)
- Harm/help classification for spell effects
- Difficulty scaling for success calculations

**Scripting Integration:**
- JavaScript implementation for spell effects
- Automatic script template generation by spell type
- Custom script path resolution and loading
- Event-driven spell execution

## Key Features

### 1. **Comprehensive Spell Types**
- **Neutral**: No specific target requirements
- **HarmSingle**: Single target harmful spells (magic missile)
- **HarmMulti**: Multi-target harmful spells (chain lightning)
- **HelpSingle**: Single target beneficial spells (heal)
- **HelpMulti**: Group beneficial spells (mass heal)
- **HarmArea**: Area-of-effect harmful spells (fireball)
- **HelpArea**: Area-of-effect beneficial spells (sanctuary)

### 2. **Magic Schools System**
- **Restoration**: Healing and condition curing
- **Illusion**: Light, darkness, invisibility, blink effects
- **Conjuration**: Summoning and teleportation magic

### 3. **Flexible Targeting System**
- Automatic target selection based on spell type
- Default targeting for harmful spells (current aggro)
- Default targeting for helpful spells (self or party)
- Area effects that bypass stealth and allegiance

### 4. **Script Template System**
- Automatic generation of spell scripts based on type
- Template copying from sample script library
- Type-specific script templates for consistent behavior

## Spell Structure

### Spell Data Structure
```go
type SpellData struct {
    SpellId     string      // Unique spell identifier
    Name        string      // Display name
    Description string      // Spell description
    Type        SpellType   // Targeting and effect type
    School      SpellSchool // Magic school classification
    Cost        int         // Resource cost (currently unused — Conviction costs coming in Phase 6)
    WaitRounds  int         // Casting delay in rounds
    Difficulty  int         // Success modifier (0-100%)
}
```

### Spell Type Enumeration
```go
type SpellType string

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

### Magic School Enumeration
```go
type SpellSchool string

const (
    SchoolRestoration SpellSchool = "restoration" // Healing and curing
    SchoolIllusion    SpellSchool = "illusion"    // Light, stealth, vision
    SchoolConjuration SpellSchool = "conjuration" // Summoning, teleportation
)
```

## Spell Discovery and Lookup

### Spell Finding System
```go
// Find spell by name or ID with exact and fuzzy matching
func FindSpell(spellName string) string {
    // Check for exact ID match first
    if sd, ok := allSpells[spellName]; ok {
        return sd.SpellId
    }
    
    // Check for exact name match
    for _, spellInfo := range allSpells {
        if strings.ToLower(spellInfo.Name) == spellName {
            return spellInfo.SpellId
        }
    }
    
    return ""
}

// Advanced spell search with prefix matching
func FindSpellByName(spellName string) *SpellData {
    var closestMatch *SpellData = nil
    spellName = strings.ToLower(spellName)
    
    for _, spellData := range allSpells {
        testName := strings.ToLower(spellData.Name)
        
        // Exact match takes priority
        if testName == spellName {
            return spellData
        }
        
        // Store first prefix match as fallback
        if closestMatch == nil && strings.HasPrefix(testName, spellName) {
            closestMatch = spellData
        }
    }
    
    return closestMatch
}

// Get spell by exact ID
func GetSpell(spellId string) *SpellData {
    if sd, ok := allSpells[spellId]; ok {
        return sd
    }
    return nil
}

// Get all available spells (returns copy)
func GetAllSpells() map[string]*SpellData {
    retSpellBook := make(map[string]*SpellData)
    for k, v := range allSpells {
        retSpellBook[k] = v
    }
    return retSpellBook
}
```

## Spell Type Classification

### Harm/Help Classification
```go
// Determine if spell is harmful, helpful, or neutral
func (s SpellType) HelpOrHarmString() string {
    switch s {
    case Neutral:
        return "Neutral"
    case HelpSingle, HelpMulti, HelpArea:
        return "Helpful"
    case HarmSingle, HarmMulti, HarmArea:
        return "Harmful"
    }
    return "Unknown"
}
```

### Target Type Classification
```go
// Get targeting description (short or long form)
func (s SpellType) TargetTypeString(short ...bool) string {
    // Short form for UI display
    if len(short) > 0 && short[0] {
        switch s {
        case Neutral:
            return "Unknown"
        case HelpSingle, HarmSingle:
            return "Single"
        case HelpMulti, HarmMulti:
            return "Group"
        case HelpArea, HarmArea:
            return "Area"
        }
        return "Unknown"
    }
    
    // Long form for detailed descriptions
    switch s {
    case Neutral:
        return "Unknown"
    case HelpSingle, HarmSingle:
        return "Single Target"
    case HelpMulti, HarmMulti:
        return "Group Target"
    case HelpArea, HarmArea:
        return "Area Target"
    }
    return "Unknown"
}
```

## Spell Creation and Management

### New Spell Creation
```go
// Create new spell with automatic script template
func CreateNewSpellFile(newSpellInfo SpellData) (string, error) {
    // Check for existing spell
    if sp := GetSpell(newSpellInfo.SpellId); sp != nil {
        return "", errors.New("Spell already exists.")
    }
    
    // Validate spell data
    if err := newSpellInfo.Validate(); err != nil {
        return "", err
    }
    
    // Configure save options
    saveModes := []fileloader.SaveOption{}
    if configs.GetFilePathsConfig().CarefulSaveFiles {
        saveModes = append(saveModes, fileloader.SaveCareful)
    }
    
    // Save spell to file system
    if err := fileloader.SaveFlatFile[*SpellData](
        string(configs.GetFilePathsConfig().DataFiles)+"/spells", 
        &newSpellInfo, 
        saveModes...
    ); err != nil {
        return "", err
    }
    
    // Update in-memory cache
    allSpells[newSpellInfo.Id()] = &newSpellInfo
    
    // Create script directory and copy template
    newScriptPath := newSpellInfo.GetScriptPath()
    os.MkdirAll(filepath.Dir(newScriptPath), os.ModePerm)
    
    // Copy type-specific script template
    templatePath := util.FilePath("_datafiles/sample-scripts/spells/" + string(newSpellInfo.Type) + ".js")
    fileloader.CopyFileContents(templatePath, newScriptPath)
    
    return newSpellInfo.SpellId, nil
}
```

### Spell Validation
```go
// Validate spell configuration
func (s *SpellData) Validate() error {
    // Clamp difficulty to valid range (0-100%)
    if s.Difficulty < 0 {
        s.Difficulty = 0
    } else if s.Difficulty > 100 {
        s.Difficulty = 100
    }
    
    return nil
}

// Get validated difficulty value
func (s *SpellData) GetDifficulty() int {
    return s.Difficulty
}
```

## Scripting Integration

### Script System Support
```go
// Load spell script content
func (s *SpellData) GetScript() string {
    scriptPath := s.GetScriptPath()
    
    if _, err := os.Stat(scriptPath); err == nil {
        if bytes, err := os.ReadFile(scriptPath); err == nil {
            return string(bytes)
        }
    }
    
    return ""
}

// Generate script file path
func (s *SpellData) GetScriptPath() string {
    return strings.Replace(
        string(configs.GetFilePathsConfig().DataFiles)+"/spells/"+s.Filepath(), 
        ".yaml", 
        ".js", 
        1
    )
}
```

### File Path Management
```go
// Generate YAML file path for spell
func (s *SpellData) Filepath() string {
    return util.FilePath(fmt.Sprintf("%s.yaml", s.SpellId))
}

// Get unique identifier
func (s *SpellData) Id() string {
    return s.SpellId
}
```

## Summoning System

### Summoning Framework
```go
// Summoning spell implementation framework
func Summon(sourceUserId int, sourceMobId int, details any) (bool, error) {
    // details contains summoning specifics (creature type, duration, etc.)
    // Implementation would handle:
    // - Creature selection based on details
    // - Summoning location determination
    // - Duration and control mechanics
    // - Integration with mob system
    
    return false, nil // Placeholder implementation
}
```

### Summoning Integration Points
- **Mob System Integration**: Create temporary mob instances
- **Duration Management**: Time-limited summons with automatic cleanup
- **Control Mechanics**: Charmed mob behavior for summoned creatures
- **Targeting System**: Summoning location and creature selection

## Spell Type Behavior Patterns

### Neutral Spells
```go
// Neutral spells have no default targeting
// Examples: Light, Detect Magic, Identify
// Usage: cast light, cast detect magic
// Behavior: Applied to caster or specified target/object
```

### Single Target Spells
```go
// HarmSingle: Targets current combat opponent by default
// Examples: Magic Missile, Lightning Bolt, Charm Person
// Usage: cast magic missile [target], cast lightning bolt
// Behavior: Auto-targets aggro mob if in combat, requires target if not

// HelpSingle: Targets self by default
// Examples: Heal, Shield, Bless
// Usage: cast heal [target], cast shield
// Behavior: Self-cast by default, can specify other targets
```

### Multi-Target Spells
```go
// HarmMulti: Targets all current combat opponents
// Examples: Chain Lightning, Fireball (targeted), Sleep
// Usage: cast chain lightning, cast sleep
// Behavior: Affects all mobs player is fighting

// HelpMulti: Targets party members
// Examples: Mass Heal, Group Bless, Party Shield
// Usage: cast mass heal, cast group bless
// Behavior: Affects all party members in range
```

### Area Effect Spells
```go
// HarmArea: Affects everyone in room (including friendlies)
// Examples: Earthquake, Meteor Swarm, Poison Cloud
// Usage: cast earthquake, cast meteor swarm
// Behavior: Indiscriminate area damage, bypasses stealth

// HelpArea: Affects everyone in room beneficially
// Examples: Sanctuary, Mass Blessing, Healing Aura
// Usage: cast sanctuary, cast mass blessing
// Behavior: Benefits all present, including hidden entities
```

## Data Loading and Management

### Spell Data Loading
```go
// Load all spell files from data directory
func LoadSpellFiles() {
    start := time.Now()
    
    tmpAllSpells, err := fileloader.LoadAllFlatFiles[string, *SpellData](
        string(configs.GetFilePathsConfig().DataFiles) + "/spells"
    )
    if err != nil {
        panic(err)
    }
    
    allSpells = tmpAllSpells
    
    mudlog.Info("spells.loadAllSpells()", 
        "loadedCount", len(allSpells), 
        "Time Taken", time.Since(start))
}
```

## Integration Patterns

### Character System Integration
```go
// Spells integrate with character magic abilities
- character.Mana                    // Mana cost deduction
- character.GetSkillLevel()         // Magic skill levels
- character.Stats.Intelligence      // Spell success calculation
- character.Buffs.AddBuff()         // Spell effect application
```

### Combat System Integration
```go
// Spells participate in combat mechanics
- combat.AttackPlayerVsMob()        // Harmful spell damage
- character.IsAggro()               // Target selection for harm spells
- character.Party                   // Target selection for help spells
- room.GetMobs()                    // Area effect target selection
```

### Event System Integration
```go
// Spells trigger through event system
events.AddToQueue(events.SpellCast{
    UserId:      userId,
    SpellId:     spellId,
    TargetType:  targetType,
    TargetId:    targetId,
    WaitRounds:  spell.WaitRounds,
})
```

## Usage Examples

### Basic Spell Casting
```go
// Find and cast a spell
spellId := spells.FindSpell("magic missile")
if spellId != "" {
    spell := spells.GetSpell(spellId)
    if spell != nil {
        // Check mana cost
        if user.Character.Mana >= spell.Cost {
            // Deduct mana
            user.Character.Mana -= spell.Cost
            
            // Queue spell casting event
            events.AddToQueue(events.SpellCast{
                UserId:     user.UserId,
                SpellId:    spellId,
                WaitRounds: spell.WaitRounds,
            })
        }
    }
}
```

### Spell Discovery
```go
// Fuzzy spell name matching
spell := spells.FindSpellByName("mag mis") // Finds "Magic Missile"
if spell != nil {
    user.SendText(fmt.Sprintf("Found spell: %s (%s)", spell.Name, spell.SpellId))
    user.SendText(fmt.Sprintf("Type: %s", spell.Type.TargetTypeString()))
    user.SendText(fmt.Sprintf("School: %s", spell.School))
    user.SendText(fmt.Sprintf("Cost: %d mana", spell.Cost))
}
```

### Creating New Spells
```go
// Create new spell with automatic script template
newSpell := spells.SpellData{
    SpellId:     "fireball",
    Name:        "Fireball",
    Description: "A blazing sphere of fire that explodes on impact.",
    Type:        spells.HarmArea,
    School:      spells.SchoolEvocation,
    Cost:        25,
    WaitRounds:  4,
    Difficulty:  15,
}

spellId, err := spells.CreateNewSpellFile(newSpell)
if err != nil {
    return err
}

// Script template automatically created at:
// _datafiles/spells/fireball.js
```

### Spell Type Behavior
```go
// Different targeting behaviors based on spell type
spell := spells.GetSpell(spellId)

switch spell.Type {
case spells.HarmSingle:
    // Target current combat opponent or require explicit target
    if user.Character.IsInCombat() {
        target = user.Character.GetAggroTarget()
    } else {
        target = parseTargetFromCommand(command)
    }

case spells.HelpSingle:
    // Default to self, allow explicit targeting
    target = user.Character
    if explicitTarget := parseTargetFromCommand(command); explicitTarget != nil {
        target = explicitTarget
    }

case spells.HarmArea:
    // Affect all entities in room
    targets = room.GetAllEntities()

case spells.HelpMulti:
    // Affect party members
    targets = user.Character.Party.GetMembers()
}
```

## Dependencies

- `internal/configs` - Configuration management for file paths and settings
- `internal/fileloader` - YAML file loading, validation, and template copying
- `internal/util` - Utility functions for file operations and path management
- `internal/mudlog` - Logging system for debugging and monitoring

This comprehensive spells system provides a flexible magic framework with sophisticated targeting, type-based behavior, scripting integration, and seamless integration with character and combat systems.