# GoMud Quests System Context

## Overview

The GoMud quests system provides a comprehensive quest management framework with support for multi-step quest progression, diverse reward types, secret quests, and token-based progress tracking. It features step validation, automatic quest discovery, and flexible reward distribution including items, gold, experience, skills, and teleportation.

## Architecture

The quests system is built around several key components:

### Core Components

**Quest Definition System:**
- Unique quest identification with integer IDs
- YAML-based storage with automatic loading and validation
- Multi-step progression with named quest steps
- Secret quest support for hidden progress tracking

**Quest Token System:**
- Token-based progress tracking using `{questId}-{stepId}` format
- Step validation and progression logic
- Automatic quest discovery and step verification
- Support for single-step and multi-step quests

**Reward System:**
- Multiple reward types (gold, items, experience, skills, buffs)
- Player and room messaging for quest completion
- Teleportation rewards for quest outcomes
- Chained quest support through quest rewards

## Key Features

### 1. **Flexible Quest Structure**
- Multi-step quest progression with named steps
- Optional quest descriptions and hints for each step
- Secret quest support for background progress tracking
- Single-step quest support for simple objectives

### 2. **Comprehensive Reward System**
- **Gold Rewards**: Direct gold distribution
- **Item Rewards**: Specific item distribution by ID
- **Experience Rewards**: Character experience points
- **Skill Rewards**: Skill level advancement in format `skillId:level`
- **Buff Rewards**: Temporary or permanent buff application
- **Quest Rewards**: Chain to new quests for storylines
- **Teleportation Rewards**: Move player to specific room
- **Messaging Rewards**: Custom player and room messages

### 3. **Token-Based Progress Tracking**
- Standardized token format for quest progress
- Step validation and progression logic
- Support for quest branching and completion checking
- Integration with character quest flag system

## Quest Structure

### Quest Definition Structure
```go
type Quest struct {
    QuestId     int         // Unique quest identifier
    Name        string      // Quest display name
    Description string      // Quest description
    Secret      bool        // Hidden from player quest lists
    Steps       []QuestStep // Ordered quest progression steps
    Rewards     QuestReward // Completion rewards
}
```

### Quest Step Structure
```go
type QuestStep struct {
    Id          string // Step identifier (e.g., "start", "middle", "end")
    Description string // Step description for players
    Hint        string // Optional hint for completing step
}
```

### Quest Reward Structure
```go
type QuestReward struct {
    QuestId       string // New quest to give (format: "{id}-{step}")
    Gold          int    // Gold amount to award
    ItemId        int    // Item to give by ID
    BuffId        int    // Buff to apply by ID
    Experience    int    // Experience points to award
    SkillInfo     string // Skill advancement (format: "skillId:level")
    PlayerMessage string // Message displayed to player
    RoomMessage   string // Message displayed to room
    RoomId        int    // Room to teleport player to
}
```

## Quest Token System

### Token Format and Parsing
```go
const QuestTokenSeparator = "-"

// Convert quest ID and step to token format
func PartsToToken(questId int, questStep string) string {
    return fmt.Sprintf("%d%s%s", questId, QuestTokenSeparator, questStep)
}

// Parse token into quest ID and step
func TokenToParts(questToken string) (questId int, questStep string) {
    parts := strings.Split(questToken, QuestTokenSeparator)
    questId, _ = strconv.Atoi(parts[0])
    
    if len(parts) > 1 {
        questStep = parts[1]
    } else {
        questStep = "start" // Default to start step
    }
    
    return questId, questStep
}
```

### Quest Progress Validation
```go
// Check if next token represents valid progression from current token
func IsTokenAfter(currentToken string, nextToken string) bool {
    currentId, currentStep := TokenToParts(currentToken)
    nextId, nextStep := TokenToParts(nextToken)
    
    // No current progress - can only start quests
    if currentStep == "" {
        if nextStep == "start" {
            return true
        } else if nextStep == "end" {
            // Single-step quest can be completed immediately
            if questInfo := GetQuest(nextToken); questInfo != nil {
                if len(questInfo.Steps) == 1 {
                    return true
                }
            }
        }
        return false
    }
    
    // Must be same quest and different step
    if currentId != nextId || currentStep == nextStep {
        return false
    }
    
    // Validate step progression
    questInfo := GetQuest(currentToken)
    if questInfo == nil {
        return false
    }
    
    // Find current step and check if next step follows
    result := false
    startLooking := false
    
    for _, step := range questInfo.Steps {
        if step.Id == currentStep {
            startLooking = true
        }
        if startLooking && step.Id == nextStep {
            result = true
            break
        }
    }
    
    return result
}
```

## Quest Discovery and Management

### Quest Retrieval
```go
// Get quest by token (validates step existence)
func GetQuest(questToken string) *Quest {
    questId, questStep := TokenToParts(questToken)
    
    quest := quests[questId]
    if quest == nil {
        return nil
    }
    
    // Special case: return full quest info
    if questStep == "all+" {
        return quest
    }
    
    // Validate step exists in quest
    stepIsValid := true
    if len(questStep) > 0 {
        stepIsValid = false
        for _, step := range quest.Steps {
            if step.Id == questStep {
                stepIsValid = true
                break
            }
        }
    }
    
    if stepIsValid {
        return quest
    }
    
    return nil
}

// Get all available quests (returns copies)
func GetAllQuests() []Quest {
    ret := []Quest{}
    for _, q := range quests {
        ret = append(ret, *q)
    }
    return ret
}
```

### Quest Statistics
```go
// Get count of quests (optionally including secret quests)
func GetQuestCt(includeSecret bool) int {
    ret := 0
    for _, q := range quests {
        if includeSecret || !q.Secret {
            ret++
        }
    }
    return ret
}
```

## File Management and Validation

### Quest File Operations
```go
// Generate filename for quest
func (r *Quest) Filename() string {
    filename := util.ConvertForFilename(r.Name)
    return fmt.Sprintf("%d-%s.yaml", r.Id(), filename)
}

// Get file path for quest
func (r *Quest) Filepath() string {
    return r.Filename()
}

// Get unique identifier
func (r *Quest) Id() int {
    return r.QuestId
}

// Validate quest configuration
func (r *Quest) Validate() error {
    return nil // Placeholder for validation logic
}
```

### Data Loading
```go
// Load all quest files from data directory
func LoadDataFiles() {
    start := time.Now()
    
    tmpQuests, err := fileloader.LoadAllFlatFiles[int, *Quest](
        configs.GetFilePathsConfig().DataFiles.String() + "/quests"
    )
    if err != nil {
        panic(err)
    }
    
    quests = tmpQuests
    
    mudlog.Info("quests.LoadDataFiles()", 
        "loadedCount", len(quests), 
        "Time Taken", time.Since(start))
}
```

## Quest Progression Patterns

### Single-Step Quests
```go
// Simple quest with immediate completion
quest := Quest{
    QuestId: 1001,
    Name:    "Deliver Message",
    Steps: []QuestStep{
        {Id: "start", Description: "Deliver the message to the guard"},
    },
    Rewards: QuestReward{
        Gold:          50,
        Experience:    100,
        PlayerMessage: "The guard thanks you for the message.",
        RoomMessage:   "The guard nods appreciatively.",
    },
}

// Token progression: "" -> "1001-start" (completion)
```

### Multi-Step Quests
```go
// Complex quest with multiple stages
quest := Quest{
    QuestId: 2001,
    Name:    "The Lost Artifact",
    Steps: []QuestStep{
        {Id: "start", Description: "Speak to the archaeologist"},
        {Id: "search", Description: "Search the ancient ruins"},
        {Id: "retrieve", Description: "Retrieve the artifact"},
        {Id: "return", Description: "Return to the archaeologist"},
        {Id: "end", Description: "Complete the quest"},
    },
    Rewards: QuestReward{
        ItemId:        501, // Ancient Artifact
        Experience:    500,
        QuestId:       "3001-start", // Chain to next quest
        PlayerMessage: "You have uncovered an ancient mystery!",
    },
}

// Token progression: 
// "" -> "2001-start" -> "2001-search" -> "2001-retrieve" -> "2001-return" -> "2001-end"
```

### Secret Quests
```go
// Hidden progress tracking quest
quest := Quest{
    QuestId: 9001,
    Name:    "Hidden Achievement",
    Secret:  true, // Not shown in quest lists
    Steps: []QuestStep{
        {Id: "progress", Description: "Make progress toward hidden goal"},
        {Id: "complete", Description: "Achieve hidden objective"},
    },
    Rewards: QuestReward{
        BuffId:        101, // Special achievement buff
        PlayerMessage: "You feel a sense of accomplishment!",
    },
}
```

## Reward System Integration

### Skill Rewards
```go
// Skill advancement reward format: "skillId:level"
reward := QuestReward{
    SkillInfo: "map:2", // Advance map skill to level 2
}

// Parsing skill reward:
parts := strings.Split(reward.SkillInfo, ":")
skillId := parts[0]    // "map"
level, _ := strconv.Atoi(parts[1]) // 2
```

### Chained Quests
```go
// Quest completion triggers new quest
reward := QuestReward{
    QuestId: "1002-start", // Start quest 1002 at "start" step
    PlayerMessage: "A new adventure awaits!",
}

// Quest branching based on choices
reward1 := QuestReward{
    QuestId: "2001-good", // Good path
}

reward2 := QuestReward{
    QuestId: "2002-evil", // Evil path
}
```

### Teleportation Rewards
```go
// Transport player to specific location
reward := QuestReward{
    RoomId:        100, // Teleport to room 100
    PlayerMessage: "You are whisked away to a new location!",
    RoomMessage:   "A portal opens and someone steps through!",
}
```

## Integration Patterns

### Character System Integration
```go
// Quests integrate with character quest flags
- character.QuestFlags[]           // Current quest progress tokens
- character.HasQuestFlag()         // Check specific quest progress
- character.AddQuestFlag()         // Add new quest progress
- character.Experience            // Experience rewards
- character.Gold                  // Gold rewards
```

### Item System Integration
```go
// Quest rewards can give items
if reward.ItemId > 0 {
    item := items.New(reward.ItemId)
    character.GiveItem(item)
}
```

### Skill System Integration
```go
// Quest rewards can advance skills
if reward.SkillInfo != "" {
    parts := strings.Split(reward.SkillInfo, ":")
    skillId := parts[0]
    level, _ := strconv.Atoi(parts[1])
    character.SetSkillLevel(skillId, level)
}
```

### Buff System Integration
```go
// Quest rewards can apply buffs
if reward.BuffId > 0 {
    character.Buffs.AddBuff(reward.BuffId, false)
}
```

## Usage Examples

### Quest Progress Tracking
```go
// Check if player can advance quest
currentProgress := "1001-start"
nextStep := "1001-middle"

if quests.IsTokenAfter(currentProgress, nextStep) {
    // Player can advance to next step
    character.RemoveQuestFlag(currentProgress)
    character.AddQuestFlag(nextStep)
    
    // Check if quest is complete
    if nextStep == "1001-end" {
        quest := quests.GetQuest(nextStep)
        if quest != nil {
            distributeRewards(character, quest.Rewards)
        }
    }
}
```

### Quest Discovery
```go
// Find quest by token
questToken := "2001-search"
quest := quests.GetQuest(questToken)

if quest != nil {
    fmt.Printf("Quest: %s\n", quest.Name)
    fmt.Printf("Description: %s\n", quest.Description)
    
    // Find current step
    _, stepId := quests.TokenToParts(questToken)
    for _, step := range quest.Steps {
        if step.Id == stepId {
            fmt.Printf("Current Step: %s\n", step.Description)
            if step.Hint != "" {
                fmt.Printf("Hint: %s\n", step.Hint)
            }
            break
        }
    }
}
```

### Reward Distribution
```go
// Distribute quest completion rewards
func distributeRewards(character *characters.Character, rewards QuestReward) {
    // Gold reward
    if rewards.Gold > 0 {
        character.Gold += rewards.Gold
    }
    
    // Experience reward
    if rewards.Experience > 0 {
        character.Experience += rewards.Experience
    }
    
    // Item reward
    if rewards.ItemId > 0 {
        item := items.New(rewards.ItemId)
        character.GiveItem(item)
    }
    
    // Skill reward
    if rewards.SkillInfo != "" {
        parts := strings.Split(rewards.SkillInfo, ":")
        if len(parts) == 2 {
            skillId := parts[0]
            level, _ := strconv.Atoi(parts[1])
            character.SetSkillLevel(skillId, level)
        }
    }
    
    // Buff reward
    if rewards.BuffId > 0 {
        character.Buffs.AddBuff(rewards.BuffId, false)
    }
    
    // Chained quest reward
    if rewards.QuestId != "" {
        character.AddQuestFlag(rewards.QuestId)
    }
    
    // Teleportation reward
    if rewards.RoomId > 0 {
        character.RoomId = rewards.RoomId
    }
    
    // Messages
    if rewards.PlayerMessage != "" {
        user.SendText(rewards.PlayerMessage)
    }
    
    if rewards.RoomMessage != "" {
        room.SendText(rewards.RoomMessage, user)
    }
}
```

### Quest Statistics
```go
// Get quest completion statistics
totalQuests := quests.GetQuestCt(false) // Exclude secret quests
totalWithSecret := quests.GetQuestCt(true) // Include secret quests

fmt.Printf("Public quests: %d\n", totalQuests)
fmt.Printf("Total quests: %d\n", totalWithSecret)
fmt.Printf("Secret quests: %d\n", totalWithSecret-totalQuests)

// List all available quests
allQuests := quests.GetAllQuests()
for _, quest := range allQuests {
    if !quest.Secret {
        fmt.Printf("Quest %d: %s\n", quest.QuestId, quest.Name)
    }
}
```

## Dependencies

- `internal/configs` - Configuration management for file paths
- `internal/fileloader` - YAML file loading and validation system
- `internal/util` - Utility functions for file operations and string conversion
- `internal/mudlog` - Logging system for debugging and monitoring

## Dialogue–Quest Integration (Phase 27)

The dialogue system can gate conversation options on quest state and advance
quests through NPC dialogue, connecting YAML-driven dialogue trees with the
token-based quest progression system.

### How It Works

The dialogue engine accepts an optional `PlayerState` callback struct that
provides quest and inventory checks without importing `characters` or `users`
(avoids circular deps):

```go
// internal/dialogue/types.go
type PlayerState struct {
    HasQuest   func(token string) bool   // checks character.HasQuest()
    HasItem    func(itemId int) bool     // checks backpack for item
    RemoveItem func(itemId int) bool     // removes item from backpack
    GiveQuest  func(token string)        // fires events.Quest to advance quest
}
```

Call sites in `talk.go` and `ask.go` build a `PlayerState` from the user's
character before calling dialogue engine functions.

### Dialogue YAML Fields for Quest Gating

Both `TreeNode` and `Pattern` structs support:

| Field | Type | Effect |
|-------|------|--------|
| `questRequired` | `[]string` | Player must have these quest tokens |
| `questExcluded` | `[]string` | Player must NOT have these tokens |
| `grantsQuest` | `string` | Quest token granted on match |
| `requiresItem` | `int` | Item ID the player must hold (consumed) |

### Greeting Variants

`TreeRoot` has an optional `Variants` list of `QuestGreeting` entries. Each
variant has `questRequired`/`questExcluded` conditions and alternate `text`/
`hints`. `Greet()` checks variants first; the first match wins.

### Worked Example: Tolva (Mob 84) — Quest 5

```yaml
# Quest 5 steps: start → ledger → evidence → end

tree:
  root:
    text: "Default greeting..."
    variants:
      - questRequired: ["5-start"]
        questExcluded: ["5-end"]
        text: "Any luck with that ledger?"
      - questRequired: ["5-end"]
        text: "Thanks to you, the magistrate will hear about this."

  nodes:
    - id: help_quest
      triggers: ["help", "quest"]
      requires: ["toll_problem"]
      questExcluded: ["5-start"]       # only shows before quest is accepted
      grantsQuest: "5-start"           # accepting starts the quest
      text: "Get me that ledger..."

    - id: ledger_return
      triggers: ["ledger", "evidence"]
      requires: ["help_quest"]
      questRequired: ["5-evidence"]    # only shows at evidence step
      questExcluded: ["5-end"]         # hidden after quest is done
      requiresItem: 21                 # consumes the crossing ledger
      grantsQuest: "5-end"             # completing fires quest rewards
      text: "No commission. No charter..."
```

When `grantsQuest` fires, it calls `events.AddToQueue(events.Quest{...})`
which is processed by `Quest_HandleQuestUpdate.go`. That handler distributes
all rewards (gold, items, buffs) defined in the quest YAML — no separate
reward mechanism is needed in the dialogue system.

### LLM Quest Context

When an NPC has an LLM profile, `talk.go` and `ask.go` populate
`llm.ConversationContext.QuestContext` with human-readable summaries of the
player's active quests. These are injected into the LLM system prompt so
Ollama-powered NPCs can reference quest state naturally.

This comprehensive quests system provides flexible quest management with multi-step progression, diverse reward types, secret quest support, and seamless integration with character progression, item distribution, and skill advancement systems.