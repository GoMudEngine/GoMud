# Quest Flags Core Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a general-purpose quest flags system that enables branching quests, startup validation of flag references, and extensible quest metadata.

**Architecture:** Quest flags are stored as `map[string]string` on Character, keyed by `"{questId}-{flagName}"`. Quests declare expected flags in YAML with allowed values. The dialogue engine and quest engine both get flag conditions and actions. Undeclared flag references cause a startup panic. The scripting API and admin commands are extended for debugging.

**Tech Stack:** Go, YAML data files, existing dialogue/quest engine infrastructure

---

### Task 1: Character QuestFlags — Data Layer

**Files:**
- Modify: `internal/characters/character.go`
- Test: `internal/characters/character_test.go` (or create `internal/characters/questflags_test.go`)

- [ ] **Step 1: Write tests for QuestFlag methods**

Create `internal/characters/questflags_test.go`:

```go
package characters

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetAndGetQuestFlag(t *testing.T) {
	c := &Character{}
	c.SetQuestFlag("11-branch", "rhett")
	assert.Equal(t, "rhett", c.GetQuestFlag("11-branch"))
}

func TestGetQuestFlag_Missing(t *testing.T) {
	c := &Character{}
	assert.Equal(t, "", c.GetQuestFlag("11-branch"))
}

func TestHasQuestFlag(t *testing.T) {
	c := &Character{}
	assert.False(t, c.HasQuestFlag("11-branch"))
	c.SetQuestFlag("11-branch", "rhett")
	assert.True(t, c.HasQuestFlag("11-branch"))
}

func TestClearQuestFlag(t *testing.T) {
	c := &Character{}
	c.SetQuestFlag("11-branch", "rhett")
	c.ClearQuestFlag("11-branch")
	assert.False(t, c.HasQuestFlag("11-branch"))
	assert.Equal(t, "", c.GetQuestFlag("11-branch"))
}

func TestSetQuestFlag_Overwrite(t *testing.T) {
	c := &Character{}
	c.SetQuestFlag("11-branch", "rhett")
	c.SetQuestFlag("11-branch", "sylara")
	assert.Equal(t, "sylara", c.GetQuestFlag("11-branch"))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/characters/ -run TestSetAndGetQuestFlag -v`
Expected: FAIL — `SetQuestFlag` not defined

- [ ] **Step 3: Add QuestFlags field to Character struct**

In `internal/characters/character.go`, add after the `QuestProgress` field (around line 97):

```go
QuestFlags    map[string]string              `yaml:"questflags,omitempty"`   // quest flag tracking (e.g., "11-branch" → "rhett")
```

- [ ] **Step 4: Implement QuestFlag methods**

Add to `internal/characters/character.go` after the existing quest methods (after `ClearQuestToken`):

```go
func (c *Character) SetQuestFlag(key, value string) {
	if c.QuestFlags == nil {
		c.QuestFlags = make(map[string]string)
	}
	c.QuestFlags[key] = value
}

func (c *Character) GetQuestFlag(key string) string {
	if c.QuestFlags == nil {
		return ""
	}
	return c.QuestFlags[key]
}

func (c *Character) HasQuestFlag(key string) bool {
	if c.QuestFlags == nil {
		return false
	}
	_, ok := c.QuestFlags[key]
	return ok
}

func (c *Character) ClearQuestFlag(key string) {
	if c.QuestFlags == nil {
		return
	}
	delete(c.QuestFlags, key)
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/characters/ -run TestQuestFlag -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/characters/character.go internal/characters/questflags_test.go
git commit -m "feat: add QuestFlags field and methods to Character"
```

---

### Task 2: Quest YAML Flag Declarations

**Files:**
- Modify: `internal/quests/quests.go`
- Modify: `internal/questengine/types.go`
- Modify: `internal/questengine/loader.go`

- [ ] **Step 1: Add QuestFlagDef type and Flags field to quests.Quest**

In `internal/quests/quests.go`, add the type and modify Quest struct:

```go
type QuestFlagDef struct {
	Key         string   `yaml:"key"`
	Values      []string `yaml:"values"`
	Description string   `yaml:"description,omitempty"`
}
```

Add `Flags []QuestFlagDef` field to the `Quest` struct:

```go
type Quest struct {
	QuestId     int
	Name        string
	Description string
	Secret      bool
	Steps       []QuestStep
	Rewards     QuestReward
	Flags       []QuestFlagDef `yaml:"flags,omitempty"`
}
```

- [ ] **Step 2: Add Flags field to QuestDef in questengine types**

In `internal/questengine/types.go`, add to `QuestDef`:

```go
type QuestDef struct {
	QuestId     int            `yaml:"questid"`
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Secret      bool           `yaml:"secret,omitempty"`
	Linear      bool           `yaml:"linear,omitempty"`
	Steps       []QuestStep    `yaml:"steps"`
	Rewards     QuestRewards   `yaml:"rewards,omitempty"`
	Triggers    []TriggerDef   `yaml:"triggers"`
	Flags       []QuestFlagDef `yaml:"flags,omitempty"`
}

type QuestFlagDef struct {
	Key         string   `yaml:"key"`
	Values      []string `yaml:"values"`
	Description string   `yaml:"description,omitempty"`
}
```

- [ ] **Step 3: Add flag declaration validation to questengine loader**

In `internal/questengine/loader.go`, add to `Validate()` after the duplicate step check:

```go
// Validate flag declarations
flagKeys := make(map[string]bool)
for _, f := range q.Flags {
	if f.Key == "" {
		return fmt.Errorf("quest %d (%s): flag has empty key", q.QuestId, q.Name)
	}
	if len(f.Values) == 0 {
		return fmt.Errorf("quest %d (%s): flag %q has no allowed values", q.QuestId, q.Name, f.Key)
	}
	fullKey := fmt.Sprintf("%d-%s", q.QuestId, f.Key)
	if flagKeys[fullKey] {
		return fmt.Errorf("quest %d (%s): duplicate flag key %q", q.QuestId, q.Name, f.Key)
	}
	flagKeys[fullKey] = true
}
```

- [ ] **Step 4: Build a global flag registry**

Add to `internal/quests/quests.go`:

```go
// flagRegistry maps fully-qualified flag keys ("11-branch") to allowed values.
var flagRegistry = map[string][]string{}

// RegisterFlags populates the flag registry from a quest's flag declarations.
func RegisterFlags(questId int, flags []QuestFlagDef) {
	for _, f := range flags {
		key := fmt.Sprintf("%d-%s", questId, f.Key)
		flagRegistry[key] = f.Values
	}
}

// ValidateFlag checks if a key-value pair is declared in the registry.
// Returns an error describing the mismatch, or nil if valid.
func ValidateFlag(key, value string) error {
	allowed, ok := flagRegistry[key]
	if !ok {
		return fmt.Errorf("undeclared quest flag %q (not defined in any quest's flags section)", key)
	}
	if value == "" {
		return nil // existence check only, any value is fine
	}
	for _, v := range allowed {
		if v == value {
			return nil
		}
	}
	return fmt.Errorf("quest flag %q has invalid value %q (allowed: %v)", key, value, allowed)
}

// GetFlagRegistry returns a copy of the flag registry for inspection.
func GetFlagRegistry() map[string][]string {
	out := make(map[string][]string, len(flagRegistry))
	for k, v := range flagRegistry {
		out[k] = v
	}
	return out
}
```

- [ ] **Step 5: Call RegisterFlags during quest loading**

In `internal/quests/quests.go`, inside `LoadDataFiles()`, after `quests = tmpQuests`, add:

```go
// Build the flag registry from all quest flag declarations
flagRegistry = map[string][]string{}
for _, q := range quests {
	RegisterFlags(q.QuestId, q.Flags)
}
```

Also in `internal/questengine/loader.go`, inside `LoadDataFiles()`, after the RegisterQuest loop, add:

```go
// Register flags from quest engine definitions too
for _, q := range tmpQuests {
	quests.RegisterFlags(q.QuestId, convertFlags(q.Flags))
}
```

Add the conversion helper:

```go
func convertFlags(flags []QuestFlagDef) []quests.QuestFlagDef {
	out := make([]quests.QuestFlagDef, len(flags))
	for i, f := range flags {
		out[i] = quests.QuestFlagDef{Key: f.Key, Values: f.Values, Description: f.Description}
	}
	return out
}
```

Note: Both `quests.QuestFlagDef` and `questengine.QuestFlagDef` have the same fields. If this duplication feels wrong, the `questengine` package can import and use `quests.QuestFlagDef` directly instead of defining its own. Choose whichever avoids circular imports.

- [ ] **Step 6: Run build**

Run: `go build ./...`
Expected: Clean build

- [ ] **Step 7: Commit**

```bash
git add internal/quests/quests.go internal/questengine/types.go internal/questengine/loader.go
git commit -m "feat: quest flag declarations with registry and validation"
```

---

### Task 3: Dialogue Engine — Flag Conditions and Actions

**Files:**
- Modify: `internal/dialogue/types.go`
- Modify: `internal/dialogue/engine.go`
- Test: `internal/dialogue/integration_dialogue_test.go`

- [ ] **Step 1: Add flag fields to dialogue types**

In `internal/dialogue/types.go`, add to `PlayerState`:

```go
type PlayerState struct {
	HasQuest      func(token string) bool
	HasItem       func(itemId int) bool
	RemoveItem    func(itemId int) bool
	GiveQuest     func(token string)
	GiveItem      func(itemId int)
	GetQuestFlag  func(key string) string
	SetQuestFlag  func(key, value string)
}
```

Add flag fields to `Pattern`:

```go
QuestFlagRequired map[string]string `yaml:"questFlagRequired,omitempty"`
QuestFlagExcluded map[string]string `yaml:"questFlagExcluded,omitempty"`
SetsQuestFlag     *QuestFlagSet     `yaml:"setsQuestFlag,omitempty"`
```

Add flag fields to `TreeNode` (same fields):

```go
QuestFlagRequired map[string]string `yaml:"questFlagRequired,omitempty"`
QuestFlagExcluded map[string]string `yaml:"questFlagExcluded,omitempty"`
SetsQuestFlag     *QuestFlagSet     `yaml:"setsQuestFlag,omitempty"`
```

Add flag fields to `QuestGreeting`:

```go
QuestFlagRequired map[string]string `yaml:"questFlagRequired,omitempty"`
QuestFlagExcluded map[string]string `yaml:"questFlagExcluded,omitempty"`
```

Add the flag set type:

```go
type QuestFlagSet struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}
```

- [ ] **Step 2: Update checkQuestGate to include flags**

In `internal/dialogue/engine.go`, update `checkQuestGate` signature and add flag checks. Add the two new map parameters:

```go
func checkQuestGate(questRequired, questExcluded []string, requiresItem int, flagRequired, flagExcluded map[string]string, ps *PlayerState) bool {
	if ps == nil {
		return true
	}

	for _, token := range questRequired {
		if !ps.HasQuest(token) {
			return false
		}
	}

	for _, token := range questExcluded {
		if ps.HasQuest(token) {
			return false
		}
	}

	if requiresItem > 0 && !ps.HasItem(requiresItem) {
		return false
	}

	// Quest flag checks
	if ps.GetQuestFlag != nil {
		for key, val := range flagRequired {
			if ps.GetQuestFlag(key) != val {
				return false
			}
		}
		for key, val := range flagExcluded {
			if ps.GetQuestFlag(key) == val {
				return false
			}
		}
	}

	return true
}
```

- [ ] **Step 3: Update applyQuestEffects to set flags**

Add a `flagSet` parameter to `applyQuestEffects`:

```go
func applyQuestEffects(grantsQuest string, requiresItem int, givesItem int, flagSet *QuestFlagSet, ps *PlayerState) {
	if ps == nil {
		return
	}
	if requiresItem > 0 {
		ps.RemoveItem(requiresItem)
	}
	if grantsQuest != "" {
		ps.GiveQuest(grantsQuest)
	}
	if givesItem > 0 && ps.GiveItem != nil {
		ps.GiveItem(givesItem)
	}
	if flagSet != nil && ps.SetQuestFlag != nil {
		ps.SetQuestFlag(flagSet.Key, flagSet.Value)
	}
}
```

- [ ] **Step 4: Update all call sites in engine.go**

Update `Match` function — the `checkQuestGate` call (line 81) and `applyQuestEffects` call (line 113):

```go
if !checkQuestGate(p.QuestRequired, p.QuestExcluded, p.RequiresItem, p.QuestFlagRequired, p.QuestFlagExcluded, ps) {
```

```go
applyQuestEffects(matched.GrantsQuest, matched.RequiresItem, matched.GivesItem, matched.SetsQuestFlag, ps)
```

Update `TreeAdvance` — the `checkQuestGate` call (line 165) and `applyQuestEffects` call (line 170):

```go
if !checkQuestGate(node.QuestRequired, node.QuestExcluded, node.RequiresItem, node.QuestFlagRequired, node.QuestFlagExcluded, ps) {
```

```go
applyQuestEffects(node.GrantsQuest, node.RequiresItem, node.GivesItem, node.SetsQuestFlag, ps)
```

Update `Greet` — the `checkQuestGate` call (line 205):

```go
if checkQuestGate(v.QuestRequired, v.QuestExcluded, 0, v.QuestFlagRequired, v.QuestFlagExcluded, ps) {
```

- [ ] **Step 5: Update buildPlayerState in talk.go**

In `internal/usercommands/talk.go`, add flag callbacks to `buildPlayerState`:

```go
GetQuestFlag: func(key string) string {
	return user.Character.GetQuestFlag(key)
},
SetQuestFlag: func(key, value string) {
	user.Character.SetQuestFlag(key, value)
},
```

- [ ] **Step 6: Run build and existing tests**

Run: `go build ./... && go test ./internal/dialogue/... -v`
Expected: Clean build and tests pass

- [ ] **Step 7: Write a dialogue integration test for flags**

Add a test to `internal/dialogue/integration_dialogue_test.go` that creates a dialogue file with `questFlagRequired` and verifies gating works.

- [ ] **Step 8: Commit**

```bash
git add internal/dialogue/types.go internal/dialogue/engine.go internal/usercommands/talk.go
git commit -m "feat: dialogue engine supports quest flag conditions and actions"
```

---

### Task 4: Quest Engine — Flag Conditions and Actions

**Files:**
- Modify: `internal/questengine/types.go`
- Modify: `internal/questengine/conditions.go`
- Modify: `internal/questengine/actions.go`
- Modify: `internal/questengine/bridge.go`
- Test: `internal/questengine/conditions_test.go`, `internal/questengine/actions_test.go`

- [ ] **Step 1: Add flag fields to Conditions and ActionDef**

In `internal/questengine/types.go`, add to `Conditions`:

```go
HasFlag     map[string]string `yaml:"has_flag,omitempty"`
MissingFlag map[string]string `yaml:"missing_flag,omitempty"`
```

Add to `ActionDef`:

```go
SetFlag *QuestFlagAction `yaml:"set_flag,omitempty"`
```

Add the type:

```go
type QuestFlagAction struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}
```

- [ ] **Step 2: Add flag methods to PlayerState interface**

In `internal/questengine/conditions.go`, extend `PlayerState`:

```go
type PlayerState interface {
	HasQuest(token string) bool
	HasItem(itemId int) bool
	GetRoomId() int
	GetQuestFlag(key string) string
}
```

- [ ] **Step 3: Update EvalConditions for flags**

In `internal/questengine/conditions.go`, add flag checks to `EvalConditions` before the final `return true`:

```go
for key, val := range c.HasFlag {
	if p.GetQuestFlag(key) != val {
		return false
	}
}
for key, val := range c.MissingFlag {
	if p.GetQuestFlag(key) == val {
		return false
	}
}
```

- [ ] **Step 4: Add SetQuestFlag to ActionContext interface and ExecuteAction**

In `internal/questengine/actions.go`, add to `ActionContext`:

```go
SetQuestFlag(key, value string)
```

Add to `ExecuteAction`, before the final error return:

```go
if a.SetFlag != nil {
	LogVerboseF(ctx.GetUserId(), "set quest flag %s=%s", a.SetFlag.Key, a.SetFlag.Value)
	ctx.SetQuestFlag(a.SetFlag.Key, a.SetFlag.Value)
	return nil
}
```

- [ ] **Step 5: Implement flag methods on GameBridge**

In `internal/questengine/bridge.go`, add:

```go
func (b *GameBridge) GetQuestFlag(key string) string {
	return b.user.Character.GetQuestFlag(key)
}

func (b *GameBridge) SetQuestFlag(key, value string) {
	b.user.Character.SetQuestFlag(key, value)
}
```

- [ ] **Step 6: Update mock contexts in test files**

In `internal/questengine/conditions_test.go`, add to `mockPlayer`:

```go
flags map[string]string
```

Add method:

```go
func (m *mockPlayer) GetQuestFlag(key string) string { return m.flags[key] }
```

Initialize in `newMockPlayer`:

```go
flags: make(map[string]string),
```

In `internal/questengine/actions_test.go` and `engine_test.go`, add `SetQuestFlag` and `GetQuestFlag` to mock contexts:

```go
func (m *mockActionContext) SetQuestFlag(key, value string) {}
```

```go
func (p *fullMockPlayer) GetQuestFlag(key string) string { return p.flags[key] }
```

- [ ] **Step 7: Write tests for flag conditions and actions**

Add to `internal/questengine/conditions_test.go`:

```go
func TestEvalConditions_HasFlag(t *testing.T) {
	p := newMockPlayer(100)
	p.flags["11-branch"] = "rhett"
	assert.True(t, EvalConditions(Conditions{HasFlag: map[string]string{"11-branch": "rhett"}}, p))
	assert.False(t, EvalConditions(Conditions{HasFlag: map[string]string{"11-branch": "sylara"}}, p))
}

func TestEvalConditions_MissingFlag(t *testing.T) {
	p := newMockPlayer(100)
	p.flags["11-branch"] = "rhett"
	assert.True(t, EvalConditions(Conditions{MissingFlag: map[string]string{"11-branch": "sylara"}}, p))
	assert.False(t, EvalConditions(Conditions{MissingFlag: map[string]string{"11-branch": "rhett"}}, p))
}
```

- [ ] **Step 8: Run all tests**

Run: `go test ./internal/questengine/... -v`
Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add internal/questengine/
git commit -m "feat: quest engine supports flag conditions and set_flag action"
```

---

### Task 5: Startup Flag Validation

**Files:**
- Modify: `internal/questengine/loader.go`
- Modify: `internal/dialogue/engine.go` (or new file `internal/dialogue/validation.go`)

- [ ] **Step 1: Add dialogue flag scanning**

Create `internal/dialogue/validation.go`:

```go
package dialogue

// CollectFlagReferences scans a DialogueFile for all quest flag references.
// Returns two slices: required flags (key→value pairs from questFlagRequired,
// questFlagExcluded) and set flags (key→value from setsQuestFlag).
func CollectFlagReferences(df *DialogueFile) (refs []FlagRef, sets []FlagRef) {
	if df == nil {
		return
	}

	for _, p := range df.Patterns {
		for k, v := range p.QuestFlagRequired {
			refs = append(refs, FlagRef{Key: k, Value: v, Source: "pattern"})
		}
		for k, v := range p.QuestFlagExcluded {
			refs = append(refs, FlagRef{Key: k, Value: v, Source: "pattern"})
		}
		if p.SetsQuestFlag != nil {
			sets = append(sets, FlagRef{Key: p.SetsQuestFlag.Key, Value: p.SetsQuestFlag.Value, Source: "pattern"})
		}
	}

	if df.Tree != nil {
		for _, v := range df.Tree.Root.Variants {
			for k, val := range v.QuestFlagRequired {
				refs = append(refs, FlagRef{Key: k, Value: val, Source: "root variant"})
			}
			for k, val := range v.QuestFlagExcluded {
				refs = append(refs, FlagRef{Key: k, Value: val, Source: "root variant"})
			}
		}
		for _, n := range df.Tree.Nodes {
			for k, v := range n.QuestFlagRequired {
				refs = append(refs, FlagRef{Key: k, Value: v, Source: "node " + n.Id})
			}
			for k, v := range n.QuestFlagExcluded {
				refs = append(refs, FlagRef{Key: k, Value: v, Source: "node " + n.Id})
			}
			if n.SetsQuestFlag != nil {
				sets = append(sets, FlagRef{Key: n.SetsQuestFlag.Key, Value: n.SetsQuestFlag.Value, Source: "node " + n.Id})
			}
		}
	}

	return
}

type FlagRef struct {
	Key    string
	Value  string
	Source string
}
```

- [ ] **Step 2: Add validation to startup sequence**

In `internal/questengine/loader.go`, add a `ValidateAllFlags()` function that:
1. Loads all dialogue files
2. Calls `CollectFlagReferences` on each
3. Also scans quest engine trigger conditions for `has_flag`/`missing_flag` references
4. Validates each reference against `quests.ValidateFlag()`
5. Panics with a clear message listing all violations

Call this at the end of `LoadDataFiles()`:

```go
func ValidateAllFlags() {
	errors := []string{}

	// Scan quest engine triggers
	for _, q := range globalEngine.quests {
		for i, t := range q.Triggers {
			for k, v := range t.Conditions.HasFlag {
				if err := quests.ValidateFlag(k, v); err != nil {
					errors = append(errors, fmt.Sprintf("quest %d trigger %d: %s", q.QuestId, i, err))
				}
			}
			for k, v := range t.Conditions.MissingFlag {
				if err := quests.ValidateFlag(k, v); err != nil {
					errors = append(errors, fmt.Sprintf("quest %d trigger %d: %s", q.QuestId, i, err))
				}
			}
			for _, a := range t.Actions {
				if a.SetFlag != nil {
					if err := quests.ValidateFlag(a.SetFlag.Key, a.SetFlag.Value); err != nil {
						errors = append(errors, fmt.Sprintf("quest %d trigger %d set_flag: %s", q.QuestId, i, err))
					}
				}
			}
		}
	}

	// Scan dialogue files — load all zones from the dialogue directory
	// and call dialogue.CollectFlagReferences on each.
	// (Implementation depends on how dialogue files are discovered —
	//  iterate the dialogue directory similar to how quest files are loaded)

	if len(errors) > 0 {
		panic(fmt.Sprintf("Quest flag validation failed:\n  %s", strings.Join(errors, "\n  ")))
	}
}
```

Add `ValidateAllFlags()` call at the end of `LoadDataFiles()`.

- [ ] **Step 3: Add runtime validation to Character.SetQuestFlag**

In `internal/characters/character.go`, modify `SetQuestFlag` to log a warning for undeclared flags:

```go
func (c *Character) SetQuestFlag(key, value string) {
	if c.QuestFlags == nil {
		c.QuestFlags = make(map[string]string)
	}
	if err := quests.ValidateFlag(key, value); err != nil {
		mudlog.Error("SetQuestFlag", "warning", err.Error())
	}
	c.QuestFlags[key] = value
}
```

- [ ] **Step 4: Run build and verify startup doesn't panic (no flags declared yet)**

Run: `go build ./...`
Expected: Clean build. Server should start fine since no flags are referenced yet.

- [ ] **Step 5: Commit**

```bash
git add internal/dialogue/validation.go internal/questengine/loader.go internal/characters/character.go
git commit -m "feat: startup validation panics on undeclared quest flag references"
```

---

### Task 6: Scripting API + Admin Command

**Files:**
- Modify: `internal/scripting/actor_func.go`
- Modify: `internal/usercommands/admin.questtoken.go`
- Modify: `_datafiles/world/dogmud/templates/admincommands/help/command.questtoken.template`

- [ ] **Step 1: Add flag methods to ScriptActor**

In `internal/scripting/actor_func.go`, add after `HasQuest`:

```go
func (a ScriptActor) GetQuestFlag(key string) string {
	return a.characterRecord.GetQuestFlag(key)
}

func (a ScriptActor) SetQuestFlag(key, value string) {
	a.characterRecord.SetQuestFlag(key, value)
}

func (a ScriptActor) HasQuestFlag(key string) bool {
	return a.characterRecord.HasQuestFlag(key)
}
```

- [ ] **Step 2: Extend admin questtoken command**

In `internal/usercommands/admin.questtoken.go`, add `flags` and `flag` subcommands. Inside the `else if` chain after `args[0] == "all"`:

```go
} else if args[0] == "flags" {

	allFlags := user.Character.QuestFlags
	headers := []string{"Flag Key", "Value"}
	rows := [][]string{}

	if len(allFlags) == 0 {
		rows = append(rows, []string{"None", ""})
	} else {
		for k, v := range allFlags {
			rows = append(rows, []string{k, v})
		}
	}

	searchResultsTable := templates.GetTable("Quest Flags", headers, rows)
	tplTxt, _ := templates.Process("tables/generic", searchResultsTable, user.UserId)
	user.SendText(tplTxt)

} else if args[0] == "flag" {

	if len(args) < 2 {
		user.SendText("Usage: questtoken flag <key> [value]")
	} else if len(args) == 2 {
		val := user.Character.GetQuestFlag(args[1])
		if val == "" {
			user.SendText(fmt.Sprintf("Flag %q is not set.", args[1]))
		} else {
			user.SendText(fmt.Sprintf("Flag %q = %q", args[1], val))
		}
	} else {
		user.Character.SetQuestFlag(args[1], args[2])
		user.SendText(fmt.Sprintf("Set flag %q = %q", args[1], args[2]))
	}

} else {
```

- [ ] **Step 3: Update help template**

Update `_datafiles/world/dogmud/templates/admincommands/help/command.questtoken.template` to add:

```
<ansi fg="command">questtoken flags</ansi>
  Show all quest flags on your character.

<ansi fg="command">questtoken flag <key></ansi>
  Show a specific flag's value.

<ansi fg="command">questtoken flag <key> <value></ansi>
  Set a quest flag manually.
```

- [ ] **Step 4: Build and verify**

Run: `go build ./...`
Expected: Clean build

- [ ] **Step 5: Commit**

```bash
git add internal/scripting/actor_func.go internal/usercommands/admin.questtoken.go _datafiles/world/dogmud/templates/admincommands/help/command.questtoken.template
git commit -m "feat: scripting API and admin command for quest flags"
```

---

### Task 7: Quest 11 — Flag-Based Branching

**Files:**
- Modify: `_datafiles/world/dogmud/quests/11-the_windwardens_dilemma.yaml`
- Modify: `_datafiles/world/dogmud/dialogue/ironwind_steppe/241.yaml`
- Modify: `_datafiles/world/dogmud/dialogue/ironwind_steppe/242.yaml`

- [ ] **Step 1: Add flag declarations to Quest 11 YAML**

```yaml
questid: 11
name: The Windwarden's Dilemma
description: A conflict is brewing on the Ironwind Steppe between a
  druid who guards the ancient stone circle and a Thornwall scholar
  who wants to extract rare crystal from beneath it. You must choose
  a side.
secret: false
flags:
  - key: branch
    values: [sylara, rhett]
    description: "Which NPC the player sided with"
steps:
  - id: start
    description: "A conflict is brewing on the Ironwind Steppe. Visit
      Windwarden Sylara at the stone circle or Geomancer Rhett at the
      ridge approach and choose a side."
    hint: "Sylara is at the stone circle (east from the ridge trail).
      Rhett is at the ridge approach (east from the ridge trail then
      north). Ask either about the conflict to choose a side."
  - id: end
    description: "You have made your choice and completed the task.
      The consequences remain to be seen."
rewards:
  playermessage: "You have chosen your path. The steppe will
    remember."
  gold: 10
```

- [ ] **Step 2: Update Sylara's dialogue (mob 241) to set flag**

On the `her_cause` node, add `setsQuestFlag`:

```yaml
    - id: her_cause
      triggers: ["circle", "conflict", "wind", "problem", "sacred", "cause", "why", "quest", "task", "help"]
      questExcluded: ["11-start"]
      grantsQuest: "11-start"
      setsQuestFlag:
        key: "11-branch"
        value: "sylara"
```

Update `offer_covenant` to use flag gating:

```yaml
    - id: offer_covenant
      triggers: ["covenant", "quest", "task", "teach", "offer", "spirits"]
      questRequired: ["11-end"]
      questExcluded: ["12-start"]
      questFlagRequired:
        "11-branch": "sylara"
      grantsQuest: "12-start"
```

Add a dismissive root variant for Rhett-path players visiting after Q11:

```yaml
      - questRequired: ["11-end"]
        questExcluded: ["12-start", "13-start"]
        questFlagRequired:
          "11-branch": "rhett"
        text: "Sylara regards you with cold respect. 'You helped the
          scholar dig his crystal from the ridge. The wind-paths felt
          it.' She turns back to the standing stones. 'The covenant
          is not for those who crack the earth for trinkets.'"
      - questRequired: ["11-end"]
        questExcluded: ["12-start", "13-start"]
        questFlagRequired:
          "11-branch": "sylara"
        text: "Sylara regards you with newfound respect. 'You proved
          yourself on the steppe. The spirits noticed.' Her voice
          softens. 'I could teach you the old covenant -- to call
          upon the wolf spirits that guard the wind-paths. It is not
          offered lightly.'"
        hints: "You could ask about the covenant."
```

- [ ] **Step 3: Update Rhett's dialogue (mob 242) to set flag**

On the `his_cause` node, add `setsQuestFlag`:

```yaml
    - id: his_cause
      triggers: ["windstone", "crystal", "conflict", "cause", "why", "problem", "quest", "task", "help"]
      questExcluded: ["11-start"]
      grantsQuest: "11-start"
      setsQuestFlag:
        key: "11-branch"
        value: "rhett"
```

Update `offer_gambit` to use flag gating:

```yaml
    - id: offer_gambit
      triggers: ["work", "extract", "gambit", "help", "quest", "task"]
      questRequired: ["11-end"]
      questExcluded: ["13-start", "12-start"]
      questFlagRequired:
        "11-branch": "rhett"
      grantsQuest: "13-start"
```

Add root variants with flag gating:

```yaml
      - questRequired: ["11-end"]
        questExcluded: ["12-start", "13-start"]
        questFlagRequired:
          "11-branch": "sylara"
        text: "Rhett's smile is strained. 'You sided with the
          Windwarden in the end. I understand -- tradition has a
          gravity to it.' He turns back to his notes. 'The crystal
          will wait. It has waited this long.'"
      - questRequired: ["11-end"]
        questExcluded: ["12-start", "13-start"]
        questFlagRequired:
          "11-branch": "rhett"
        text: "Rhett's eyes light up. 'Those samples confirmed
          everything I suspected.' He leans in conspiratorially.
          'I have been preparing an extraction site. Ask me about
          the work if you are interested.'"
        hints: "You could ask about further work with the windstone."
```

- [ ] **Step 4: Verify startup doesn't panic**

Restart server. The flag declarations in Quest 11 should match the references in dialogues 241/242. If any typo exists, the server panics with a clear message.

- [ ] **Step 5: Manual test**

New character → talk to Rhett → ask about quest → should grant 11-start + set 11-branch=rhett → `questtoken flags` shows the flag → complete Q11 → `ask rhett quest` should offer gambit → visiting Sylara shows dismissive text.

- [ ] **Step 6: Commit**

```bash
git add _datafiles/world/dogmud/quests/11-the_windwardens_dilemma.yaml _datafiles/world/dogmud/dialogue/ironwind_steppe/241.yaml _datafiles/world/dogmud/dialogue/ironwind_steppe/242.yaml
git commit -m "feat: Quest 11 uses flags for branch tracking"
```

---

### Task 8: Migration — Existing Players

**Files:**
- Modify: `internal/users/userrecord.go`

- [ ] **Step 1: Find the migration chain**

Search for `Migrate` calls in `userrecord.go` and add `MigrateQuestFlags` at the end.

- [ ] **Step 2: Implement MigrateQuestFlags**

```go
func MigrateQuestFlags(user *UserRecord) {
	if user.Character.QuestFlags != nil {
		return // already migrated or set
	}

	// Infer Q11 branch from Q12/Q13 progress
	q12Progress := user.Character.QuestProgress[12]
	q13Progress := user.Character.QuestProgress[13]

	if q12Progress != "" {
		user.Character.SetQuestFlag("11-branch", "sylara")
	} else if q13Progress != "" {
		user.Character.SetQuestFlag("11-branch", "rhett")
	}
	// If neither Q12 nor Q13 started, leave unset —
	// the player will pick a branch when they next interact.
}
```

- [ ] **Step 3: Add to migration chain**

Call `MigrateQuestFlags(user)` in the migration sequence in `LoadUser` or wherever the other `Migrate*` calls are.

- [ ] **Step 4: Commit**

```bash
git add internal/users/userrecord.go
git commit -m "feat: migration infers Q11 branch flags from Q12/Q13 progress"
```
