# GoMud Quests System Context

## Overview

The GoMud quests system provides a comprehensive quest management framework with support for multi-step quest progression, diverse reward types, secret quests, and token-based progress tracking. It features step validation, automatic quest discovery, and flexible reward distribution including items, gold, skills, stats, recipes, spells, and teleportation.

DOGMud has no experience points or character levels: progression is
use-based only (see CLAUDE.md). Quest rewards that used to grant XP were
removed; skill and stat rewards take their place.

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
- Multiple reward types (gold, items, skills, stats, recipes, spells, buffs, faction reputation)
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
- **Item Rewards**: A single item by ID, or a multi-item stockpile grant
- **Skill Rewards**: Floor-raise skill advancement, one or more skills at once
- **Stat Rewards**: Additive, permanent stat increases, one or more stats at once
- **Recipe Rewards**: Grants known crafting recipes
- **Spell Rewards**: Teaches a spell
- **Buff Rewards**: Temporary or permanent buff application
- **Quest Rewards**: Chain to new quests for storylines
- **Teleportation Rewards**: Move player to specific room
- **Faction Reputation Rewards**: Adjust standing with a named faction
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
    QuestId        int            // Unique quest identifier
    Name           string         // Quest display name
    Description    string         // Quest description
    Secret         bool           // Hidden from player quest lists
    Steps          []QuestStep    // Ordered quest progression steps
    Rewards        QuestReward    // Completion rewards
    Triggers       []TriggerDef   // Event-driven step/action rules (see triggers.go)
    Flags          []QuestFlagDef // Declared flag keys + allowed values (see Quest Flags System below)
    Repeatable     bool           // If true, completing the quest clears progress so it can be retaken
    CooldownRounds int            // Rounds after completion before a repeatable quest can be retaken
}
```

### Quest Step Structure
```go
type QuestStep struct {
    Id           string // Step identifier (e.g., "start", "middle", "end")
    Description  string // Step description for players
    Hint         string // Optional hint for completing step
    MapTarget    int    // Room the minimap marker points at during this step
                         // (0 = infer from room_enter triggers, -1 = deliberate no marker)
    MapTargetMob int    // Points the minimap marker at a unique NPC's CURRENT room
                         // instead of a fixed room (falls back to MapTarget if unresolved)
}
```

### Quest Reward Structure
```go
type QuestReward struct {
    QuestId       string // New quest to give (format: "{id}-{step}")
    Gold          int    // Gold amount to award
    ItemId        int    // Single item to give by ID
    BuffId        int    // Buff to apply by ID
    SkillInfo     string // Skill advancement, one or more: "skill:level[,skill:level]"
    StatInfo      string // Stat increase, one or more: "stat:amount[,stat:amount]"
    RecipeInfo    string // Recipe(s) to grant, comma-separated recipe IDs
    ItemInfo      string // Item stockpile grant: "itemid[:qty][,itemid[:qty]]"
    SpellId       string // Spell to teach on completion
    PlayerMessage string // Message displayed to player
    RoomMessage   string // Message displayed to room
    RoomId        int    // Room to teleport player to
    RepFaction    string // Faction whose reputation should change
    RepAmount     int    // Amount to bump RepFaction's reputation by
}
```

There is no experience/XP field. DOGMud removed character levels and XP
entirely; skill and stat rewards (`SkillInfo`, `StatInfo`) are the
progression-granting rewards. `SkillInfo` grants are floor-raises: a grant
never downgrades a player already at or above the target level, so a
veteran replaying a newbie quest keeps their rank. `StatInfo` grants are
always additive.

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
// Get a quest definition by bare numeric id — no token parsing, no step
// check. Used by the editor's load path, since not every quest's tokens
// share a common valid step.
func GetQuestById(questId int) *Quest {
    return quests[questId]
}

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

// Validate implements fileloader.Loadable — it runs on EVERY parse (boot
// and editor save), so a broken quest file fails to load instead of
// loading half-formed. Checks: QuestId > 0, Name non-empty, at least one
// step, every trigger has a known event and at least one action, no
// duplicate step ids, every declared flag has a non-empty key and at
// least one allowed value with no duplicate flag keys, and every
// same-quest grant token in a trigger action names a real step.
func (r *Quest) Validate() error {
    // ...
}
```

### Data Loading
```go
// Load all quest files from data directory
func LoadDataFiles() {
    start := time.Now()

    dataPath := questsDataRoot() // DataFiles config path + "/quests"
    tmpQuests, err := fileloader.LoadAllFlatFiles[int, *Quest](dataPath)
    if err != nil {
        panic(errors.Wrap(err, `filepath: `+dataPath))
    }

    quests = tmpQuests

    // Rebuild the flag registry from the freshly loaded quests so
    // ValidateFlag has an up to date view of every declared flag key.
    flagRegistry = map[string][]string{}
    for _, q := range quests {
        RegisterFlags(q.QuestId, q.Flags)
    }

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
// Skill advancement reward format: "skill:level", comma-separated for
// several skills in one reward.
reward := QuestReward{
    SkillInfo: "weapon-combat:1,unarmed-combat:1",
}

// The real parser (internal/hooks/Quest_HandleQuestUpdate.go, not this
// package) splits on "," then ":" per entry:
for _, entry := range strings.Split(reward.SkillInfo, ",") {
    details := strings.Split(strings.TrimSpace(entry), ":")
    skillName := strings.ToLower(strings.TrimSpace(details[0]))
    level, _ := strconv.Atoi(strings.TrimSpace(details[1]))
    // grant is a floor-raise: only applied if level > character's current level
}
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
`internal/characters` (see `internal/characters/quests.go`) is the actual
consumer of quest tokens and flags — this package never touches
`*characters.Character` directly:
```go
- character.QuestProgress               // map[int]string: questId -> current step id
- character.HasQuest(token)             // true if progress is at or past token's step
- character.GiveQuestToken(token) bool  // advances progress if token is a valid next step
- character.ClearQuestToken(token)      // removes progress (used by repeatable quests)
- character.QuestFlags                  // map[string]string: branch/choice flags, key "{questId}-{flagName}"
- character.SetQuestFlag(key, value)    // set a flag (validated against the quest's declared values)
- character.GetQuestFlag(key) string    // read a flag; "" if unset
- character.HasQuestFlag(key) bool      // true if the flag key is set at all
- character.ClearQuestFlag(key)
- character.Gold                        // int: gold reward target
```

### Item System Integration
```go
// Single item reward
if reward.ItemId > 0 {
    item := items.New(reward.ItemId)
    character.StoreItem(item)
}

// Multi-item stockpile reward (ItemInfo): see parseItemGrants in
// internal/hooks/Quest_HandleQuestUpdate.go for the real grant loop.
```

### Skill System Integration
```go
// Quest rewards can advance skills — a floor-raise, never a downgrade
if reward.SkillInfo != "" {
    // ... parse "skill:level[,skill:level]" per the Skill Rewards example above ...
    if character.GetSkillLevel(skills.SkillTag(skillName)) < level {
        character.TrainSkill(skillName, level)
    }
}
```

### Buff System Integration
```go
// Quest rewards can apply buffs
if reward.BuffId > 0 {
    character.AddBuff(reward.BuffId, false) // isPermanent=false
}
```

## Usage Examples

### Quest Progress Tracking
```go
// GiveQuestToken checks quests.IsTokenAfter internally, so callers don't
// need to call it themselves before advancing.
nextStep := "1001-middle"

if character.GiveQuestToken(nextStep) {
    // Progress advanced.

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
The real distribution logic lives in
`internal/hooks/Quest_HandleQuestUpdate.go`, not in this package (this
package only defines the data shapes). Condensed and simplified from that
file:
```go
// Distribute quest completion rewards
func distributeRewards(questUser *users.UserRecord, rewards QuestReward) {
    character := questUser.Character

    // Gold reward
    if rewards.Gold > 0 {
        character.Gold += rewards.Gold
    }

    // Item reward
    if rewards.ItemId > 0 {
        item := items.New(rewards.ItemId)
        character.StoreItem(item)
    }
    // Item-stockpile reward (rewards.ItemInfo) grants N of each listed
    // item — see parseItemGrants.

    // Skill reward(s) — floor-raise, never downgrades
    for _, grant := range parseSkillGrants(rewards.SkillInfo) {
        if character.GetSkillLevel(skills.SkillTag(grant.skill)) < grant.level {
            character.TrainSkill(grant.skill, grant.level)
        }
    }

    // Stat reward(s) — always additive
    for _, grant := range parseStatGrants(rewards.StatInfo) {
        character.IncreaseStat(grant.stat, grant.amount)
    }

    // Recipe reward(s)
    for _, recipeId := range parseRecipeGrants(rewards.RecipeInfo) {
        character.LearnRecipe(recipeId)
    }

    // Spell reward
    if rewards.SpellId != "" {
        character.LearnSpell(rewards.SpellId)
    }

    // Buff reward
    if rewards.BuffId > 0 {
        character.AddBuff(rewards.BuffId, false)
    }

    // Chained quest reward
    if rewards.QuestId != "" {
        character.GiveQuestToken(rewards.QuestId)
    }

    // Teleportation reward
    if rewards.RoomId > 0 {
        rooms.MoveToRoom(questUser.UserId, rewards.RoomId)
    }

    // Faction reputation reward
    if rewards.RepFaction != "" && rewards.RepAmount != 0 {
        factions.BumpRep(rewards.RepFaction, questUser.UserId, rewards.RepAmount)
    }

    // Messages
    if rewards.PlayerMessage != "" {
        questUser.SendText(messaging.CategoryLoot, rewards.PlayerMessage)
    }
    if rewards.RoomMessage != "" {
        sendVisualRoomText(room, messaging.CategoryEmote, rewards.RoomMessage, questUser.UserId)
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
    HasQuest         func(token string) bool   // checks character.HasQuest()
    HasItem          func(itemId int) bool     // checks backpack for item
    RemoveItem       func(itemId int) bool     // removes item from backpack
    GiveQuest        func(token string)        // fires events.Quest to advance quest
    GiveItem         func(itemId int) bool     // returns whether the item actually reached the player (false e.g. over carry capacity)
    GetQuestFlag     func(key string) string
    SetQuestFlag     func(key, value string)
    BumpRep          func(faction string, delta int)
    GiveGold         func(amount int)
    HasOwnMasterwork func(skillMin int) bool
}
```
`GiveItem` returning false aborts the node's OTHER effects too
(`grantsQuest`, `requiresItem` removal, flags, rep, gold) so a
`givesItem` + `grantsQuest` node can never burn its quest token on an
item that didn't actually arrive; the node stays re-triggerable once the
player makes room.

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

This comprehensive quests system provides flexible quest management with
multi-step progression, diverse reward types, secret quest support, and
seamless integration with character progression, item distribution, and
skill advancement systems.

---

## Quest Flags System

Quest flags are a key-value metadata store attached to a player's quest
progress. They complement the token system: tokens track *where* a player
is in a quest (which step), while flags track *how* they got there (which
branch, which choice, whose side they took).

### Flags vs. Tokens — When to Use Each

| Mechanism | Use for | Storage |
|-----------|---------|---------|
| Quest token (`{id}-{step}`) | Step progress, gating future steps | `character.QuestProgress` |
| Quest flag (`{id}-{name}`) | Branching choices, cross-quest state | `character.QuestFlags` |

Use **tokens** when you want to gate an NPC interaction on whether a player
has reached a certain quest stage. Use **flags** when you need to remember
a *choice* the player made — especially when that choice affects a different
NPC or a later quest in the same chain.

Flags never expire and are never consumed. They are permanent markers on the
character until explicitly cleared by a script.

### Flag Declaration in Quest YAML

Every flag key that any dialogue file or quest condition references MUST be
declared in the quest's YAML under a top-level `flags:` list. **Referencing
an undeclared flag key causes a server panic at startup.** This is
intentional — it catches typos in dialogue files before players ever see
them in production.

```yaml
questid: 11
name: "The Schism at Veltara"
flags:
  - key: branch
    values: [sylara, rhett]
    description: "Which NPC the player sided with"
steps:
  - id: start
    description: "Speak to Sylara or Rhett about the schism."
  - id: middle
    description: "Carry out your chosen ally's request."
  - id: end
    description: "Return to your ally with proof."
```

**Key naming convention:** `"{questId}-{flagName}"` — e.g., `"11-branch"`.
Always namespace the flag with the quest ID to prevent collisions between
quests. The YAML `key:` field stores only the short name (`branch`); the
full namespaced key is constructed at runtime as `"{questId}-{key}"`.

**Values list:** All legal values must be listed. The engine validates that
any `setsQuestFlag` value is in the declared list at startup.

### Dialogue Integration

Three new fields on `TreeNode` and `Pattern` support flag-driven gating:

#### `setsQuestFlag`
Sets a flag when this dialogue node fires. Evaluated after `grantsQuest` so
both the token and the flag are written atomically from the player's
perspective.

```yaml
- id: side_with_rhett
  triggers: ["rhett", "side", "help"]
  questRequired: ["11-start"]
  questExcluded: ["11-middle"]
  grantsQuest: "11-middle"
  setsQuestFlag:
    key: "11-branch"
    value: "rhett"
  text: "Then we are agreed. Bring me the seal and we are done."
```

#### `questFlagRequired`
Gates the node on an exact flag value. Accepts a map of `key: value` pairs.
All pairs must match for the node to be eligible.

```yaml
- id: rhett_midquest_check
  triggers: ["progress", "seal", "status"]
  questRequired: ["11-middle"]
  questFlagRequired:
    "11-branch": "rhett"
  text: "Have you found the seal yet? Time is short."
```

#### `questFlagExcluded`
Hides the node if ANY of the listed key/value pairs match. Useful for
suppressing irrelevant dialogue on wrong-path players.

```yaml
- id: sylara_reward
  triggers: ["reward", "done", "finished"]
  questRequired: ["11-end"]
  questFlagExcluded:
    "11-branch": "rhett"
  text: "You have done well. The Circle will remember this."
```

### Quest Engine Condition/Action Integration

Quest step conditions and actions (used in quest YAML `steps[].conditions`)
support flag checks:

```yaml
steps:
  - id: middle
    description: "Complete your ally's task."
    conditions:
      - has_flag: {"11-branch": "rhett"}
        hint: "Rhett wants you to retrieve the obsidian seal."
      - has_flag: {"11-branch": "sylara"}
        hint: "Sylara wants you to destroy the seal before Rhett can use it."
```

**Condition types:**
- `has_flag: {key: value}` — true if flag equals value
- `missing_flag: {key: value}` — true if flag does NOT equal value (or
  is unset)

**Action type (for use in reward blocks or triggered events):**
- `set_flag: {key: "11-branch", value: "rhett"}` — programmatically set a
  flag (used in JS scripts when dialogue-layer flag setting isn't sufficient)

### Admin and Scripting Interface

**Admin commands (via `questtoken` command):**

```
questtoken flags              # list all flags on your character
questtoken flag 11-branch     # show current value of one flag
questtoken flag 11-branch rhett  # manually set a flag (admin use only)
```

**JavaScript scripting API** (available in all mob `.js` scripts via the
`user` object):

```javascript
// Read a flag
var branch = user.GetQuestFlag("11-branch");
if (branch === "rhett") {
    user.SendText("Rhett's ally — I know your face.");
}

// Set a flag (rare — prefer setsQuestFlag in dialogue YAML)
user.SetQuestFlag("11-branch", "sylara");

// Check existence (flag is unset if empty string)
if (user.HasQuestFlag("11-branch")) {
    // branch is already chosen
}
```

Prefer `setsQuestFlag` in dialogue YAML over `SetQuestFlag` in scripts
wherever possible. Script-side flag setting is best reserved for `onGive`
handlers or combat hooks where dialogue YAML can't fire.

### Worked Example: Quest 11 — The Schism at Veltara

Quest 11 is a two-NPC branching quest. Sylara and Rhett both offer the
quest; the player can only side with one. The flag `11-branch` records the
choice and gates all subsequent dialogue on both NPCs.

**Quest YAML (`11-the_schism_at_veltara.yaml`):**

```yaml
questid: 11
name: "The Schism at Veltara"
flags:
  - key: branch
    values: [sylara, rhett]
    description: "Which NPC the player sided with"
steps:
  - id: start
    description: "Speak to Sylara or Rhett about the schism."
    hint: "Ask Sylara or Rhett about the schism."
  - id: middle
    description: "Carry out your chosen ally's request."
  - id: end
    description: "Return to your ally with the proof."
rewards:
  gold: 200
  playerMessage: "The schism is resolved — for better or worse."
```

**Sylara's dialogue tree (abbreviated):**

```yaml
tree:
  root:
    text: "The Circle fractures around us. Choose your allegiance wisely."
    variants:
      # Mid-quest: player is on Sylara's branch
      - questRequired: ["11-middle"]
        questFlagRequired: {"11-branch": "sylara"}
        text: "You serve the Circle still. Bring me the proof."
      # Mid-quest: player sided with Rhett — show dismissal
      - questRequired: ["11-middle"]
        questFlagRequired: {"11-branch": "rhett"}
        text: "You walk Rhett's path. We have nothing to discuss."
      # Quest done
      - questRequired: ["11-end"]
        text: "It is done. The Circle holds."

  nodes:
    # DISMISSAL NODE — must appear FIRST so it blocks keyword matches
    # for players already committed to Rhett's branch
    - id: rhett_dismiss
      triggers: ["schism", "quest", "task", "help", "side", "sylara"]
      questRequired: ["11-middle"]
      questFlagRequired: {"11-branch": "rhett"}
      text: "You have chosen Rhett. I have nothing for you."

    # Quest offer — only shows before player has started
    - id: offer_quest
      triggers: ["schism", "quest", "task", "help"]
      questExcluded: ["11-start", "11-middle", "11-end"]
      grantsQuest: "11-start"
      setsQuestFlag:
        key: "11-branch"
        value: "sylara"
      text: "Then stand with me. The seal must be destroyed."

    # Mid-quest check — only for Sylara's branch
    - id: progress_check
      triggers: ["seal", "progress", "proof"]
      questRequired: ["11-middle"]
      questFlagRequired: {"11-branch": "sylara"}
      questExcluded: ["11-end"]
      text: "The seal must be destroyed before Rhett can use it."

    # Completion — only for Sylara's branch
    - id: complete
      triggers: ["done", "finished", "proof", "seal"]
      questRequired: ["11-middle"]
      questFlagRequired: {"11-branch": "sylara"}
      questExcluded: ["11-end"]
      requiresItem: 4422
      grantsQuest: "11-end"
      text: "The fragments. It is done. You have my gratitude."
```

**Rhett's dialogue tree follows the mirror pattern** with `questFlagRequired:
{"11-branch": "rhett"}` and a dismissal node gating `{"11-branch": "sylara"}`
players out of his quest flow.

### Common Pitfalls

#### 1. Missing dismissal nodes
The most common mistake. If a player already committed to Rhett's branch
visits Sylara, her `schism`/`quest`/`task` keyword nodes will still match
unless a dismissal node sits above them in the nodes list. Dismissal nodes
**must appear first** in the `nodes:` list so the engine evaluates them
before reaching the quest-offer nodes.

#### 2. Missing mid-quest root variants
Without a variant for "player is on the other path," the NPC gives their
default greeting to wrong-path players, which often contains hints about the
quest. This leaks quest text and confuses players. Always add a root variant
with `questFlagRequired` for both branches during the active quest phase.

#### 3. `is_component` on quest items
Items with `is_component: true` auto-route to the component bag on pickup.
Quest items must NOT have `is_component: true` — they belong in the regular
backpack where quest scripts and `requiresItem` checks can find them.

#### 4. Flag key typos bypassing validation
The startup panic for undeclared flag keys only fires if the quest YAML is
loaded AND the dialogue YAML references a `questFlagRequired`/
`questFlagExcluded`/`setsQuestFlag` key that the quest didn't declare.
If you create a flag reference in dialogue before writing the quest YAML,
the server will panic at start. Write the quest YAML `flags:` block first.

#### 5. Double-completion guard missing
Nodes that fire `grantsQuest` for the final step should always include
`questExcluded: ["{id}-end"]` to prevent re-triggering if the player talks
to the NPC again after completion. This is the same guard used for all
completion nodes (see CLAUDE.md "Quest Re-Grant Prevention SOP").

#### 6. Cross-quest flag references
If quest 14 needs to know which branch the player took in quest 11, it reads
`"11-branch"` directly. There is no need to copy the flag into a new key.
Document the dependency in both quest YAMLs with a comment so future
maintainers know the quests are coupled.
## Files

| File | Purpose |
|------|---------|
| `quests.go` | Every quest definition type — the single owner of the quest file parse |
| `triggers.go` | Trigger and action definition shapes |
| `save.go` | Quest file persistence |
| `validate_refs.go` | Cross-reference validation (flags, tokens, ids) |

**This package owns the data; `internal/questengine` owns the evaluation.**
Since the 5c-pre unification, `questengine` only aliases these types. Change a
YAML field here; change what a trigger *does* there.
