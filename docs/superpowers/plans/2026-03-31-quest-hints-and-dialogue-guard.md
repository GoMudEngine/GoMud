# Quest Hints & Dialogue Self-Reference Guard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `hint` command that shows quest step hints, validation tests for hint coverage and NPC self-reference, and update content generation skills.

**Architecture:** The `hint` command reads quest definitions from the quest engine, resolves the player's current step, and displays the step's `Hint` field. A new `LastQuestId` field on Character tracks the most recently progressed quest. Two validation tests enforce data quality: one requires every quest step to have a hint, the other flags dialogue files where NPCs mention themselves by name.

**Tech Stack:** Go, YAML quest/dialogue data files, existing template/help system

---

## File Structure

| File | Responsibility |
|------|---------------|
| **Create:** `internal/usercommands/hint.go` | `hint` command implementation |
| **Create:** `_datafiles/world/dogmud/templates/help/hint.template` | Help file for hint command |
| **Create:** `_datafiles/world/default/templates/help/hint.template` | Default help file |
| **Create:** `_datafiles/world/empty/templates/help/hint.template` | Empty world help file |
| **Create:** `internal/questengine/validation_test.go` | Hint coverage + dialogue self-reference tests |
| **Modify:** `internal/characters/character.go` | Add `LastQuestId` field |
| **Modify:** `internal/usercommands/usercommands.go` | Register `hint` command |
| **Modify:** `_datafiles/world/dogmud/hints.yaml` | Add hint command to broadcast rotation |

---

### Task 1: Add LastQuestId to Character + Update GiveQuestToken

**Files:**
- Modify: `internal/characters/character.go`

- [ ] **Step 1: Add LastQuestId field to Character struct**

In `internal/characters/character.go`, find the `QuestProgress` field (line 97) and add `LastQuestId` after it:

```go
	QuestProgress    map[int]string                 `yaml:"questprogress,omitempty"` // quest progress tracking
	LastQuestId      int                            `yaml:"lastquestid,omitempty"`   // most recently progressed quest
```

- [ ] **Step 2: Update GiveQuestToken to set LastQuestId**

In `GiveQuestToken` (around line 2221), after `c.QuestProgress[questId] = newStep` on line 2233, add:

```go
		c.LastQuestId = questId
```

So the full success block becomes:

```go
	if quests.IsTokenAfter(currentToken, questToken) {
		c.QuestProgress[questId] = newStep
		c.LastQuestId = questId
		return true
	}
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/characters/`
Expected: Compiles without errors

- [ ] **Step 4: Commit**

```bash
git add internal/characters/character.go
git commit -m "feat: add LastQuestId field to Character, set on quest token grant"
```

---

### Task 2: Implement hint Command

**Files:**
- Create: `internal/usercommands/hint.go`

- [ ] **Step 1: Create hint.go**

```go
package usercommands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Hint(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	engine := questengine.GetEngine()
	progress := user.Character.GetQuestProgress()

	if len(progress) == 0 {
		user.SendText("You don't have any active quests.")
		return true, nil
	}

	var questId int

	if rest == "" {
		// Default: most recently progressed quest
		questId = user.Character.LastQuestId
		if questId == 0 {
			// Fallback: pick any active quest
			for qId := range progress {
				questId = qId
				break
			}
		}
	} else if num, err := strconv.Atoi(rest); err == nil {
		// Numeric argument: quest ID
		questId = num
	} else {
		// String argument: search by quest name (case-insensitive partial match)
		search := strings.ToLower(rest)
		for qId := range progress {
			qDef := engine.GetQuest(qId)
			if qDef != nil && strings.Contains(strings.ToLower(qDef.Name), search) {
				questId = qId
				break
			}
		}
		if questId == 0 {
			user.SendText("You don't have an active quest by that name.")
			return true, nil
		}
	}

	// Verify the player actually has this quest in progress
	currentStep, hasQuest := progress[questId]
	if !hasQuest {
		user.SendText("You don't have an active quest by that name.")
		return true, nil
	}

	qDef := engine.GetQuest(questId)
	if qDef == nil {
		user.SendText("You don't have any active quests.")
		return true, nil
	}

	// Quest is complete (on last step "end")
	if currentStep == "end" {
		user.SendText(fmt.Sprintf(
			`<ansi fg="yellow">Quest Hint</ansi> <ansi fg="yellow-bold">(%s)</ansi>: This quest is complete!`,
			qDef.Name))
		return true, nil
	}

	// Find the next step (the one the player needs to do)
	nextHint := ""
	foundCurrent := false
	for _, step := range qDef.Steps {
		if foundCurrent {
			nextHint = step.Hint
			break
		}
		if step.Id == currentStep {
			foundCurrent = true
		}
	}

	// If player has no progress on this quest yet (just started), show first step hint
	if !foundCurrent && currentStep == "" {
		if len(qDef.Steps) > 0 {
			nextHint = qDef.Steps[0].Hint
		}
	}

	if nextHint == "" {
		user.SendText(fmt.Sprintf(
			`<ansi fg="yellow">Quest Hint</ansi> <ansi fg="yellow-bold">(%s)</ansi>: No hint available for this step.`,
			qDef.Name))
		return true, nil
	}

	user.SendText(fmt.Sprintf(
		`<ansi fg="yellow">Quest Hint</ansi> <ansi fg="yellow-bold">(%s)</ansi>: <ansi fg="white-bold">%s</ansi>`,
		qDef.Name, nextHint))

	return true, nil
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/usercommands/`
Expected: Compiles without errors

- [ ] **Step 3: Commit**

```bash
git add internal/usercommands/hint.go
git commit -m "feat: implement hint command for quest step hints"
```

---

### Task 3: Register hint Command + Help Files

**Files:**
- Modify: `internal/usercommands/usercommands.go`
- Create: `_datafiles/world/dogmud/templates/help/hint.template`
- Create: `_datafiles/world/default/templates/help/hint.template`
- Create: `_datafiles/world/empty/templates/help/hint.template`

- [ ] **Step 1: Register hint command in usercommands.go**

In `internal/usercommands/usercommands.go`, find the command table (alphabetical order). Add between `help` and `identify` (or wherever `h` falls alphabetically):

```go
		`hint`:        {Hint, true, true, false},
```

- [ ] **Step 2: Create help template for dogmud**

Create `_datafiles/world/dogmud/templates/help/hint.template`:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">hint</ansi>

The <ansi fg="command">hint</ansi> command gives you a hint for your
current quest step. Useful when you're not sure what to do next.

<ansi fg="yellow">Usage: </ansi>

  <ansi fg="command">hint</ansi>              Show hint for your most recent quest
  <ansi fg="command">hint [name]</ansi>       Show hint for a specific quest
  <ansi fg="command">hint [number]</ansi>     Show hint for a quest by ID
```

- [ ] **Step 3: Create help template for default**

Create `_datafiles/world/default/templates/help/hint.template` with identical content to the dogmud template.

- [ ] **Step 4: Create help template for empty**

Create `_datafiles/world/empty/templates/help/hint.template` with identical content to the dogmud template.

- [ ] **Step 5: Verify compilation**

Run: `go build ./internal/usercommands/`
Expected: Compiles without errors

- [ ] **Step 6: Commit**

```bash
git add internal/usercommands/usercommands.go
git add _datafiles/world/dogmud/templates/help/hint.template
git add _datafiles/world/default/templates/help/hint.template
git add _datafiles/world/empty/templates/help/hint.template
git commit -m "feat: register hint command and add help files"
```

---

### Task 4: Add hint to Broadcast Hints Rotation

**Files:**
- Modify: `_datafiles/world/dogmud/hints.yaml`

- [ ] **Step 1: Add hint command to hints.yaml**

Add a new entry to the `hints` list in `_datafiles/world/dogmud/hints.yaml`:

```yaml
  - >-
    Stuck on a quest? Type <ansi fg="command">hint</ansi> for a nudge
    in the right direction.
```

- [ ] **Step 2: Commit**

```bash
git add _datafiles/world/dogmud/hints.yaml
git commit -m "feat: add hint command to broadcast hints rotation"
```

---

### Task 5: Hint Coverage Validation Test

**Files:**
- Create: `internal/questengine/validation_test.go`

- [ ] **Step 1: Write the hint coverage test**

```go
package questengine_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/stretchr/testify/assert"
)

func loadAllQuestDefs(t *testing.T) []*questengine.QuestDef {
	t.Helper()
	dataPath := configs.GetFilePathsConfig().DataFiles.String() + `/quests`
	quests, err := fileloader.LoadAllFlatFiles[int, *questengine.QuestDef](dataPath)
	if err != nil {
		// If no quest files exist yet, skip
		if strings.Contains(err.Error(), "no such file") ||
			strings.Contains(err.Error(), "cannot find") {
			t.Skip("No quest data files found, skipping validation")
		}
		t.Fatalf("Failed to load quest definitions: %v", err)
	}
	return quests
}

func TestAllQuestStepsHaveHints(t *testing.T) {
	quests := loadAllQuestDefs(t)

	if len(quests) == 0 {
		t.Skip("No quest definitions loaded, skipping hint coverage test")
	}

	var missing []string
	for _, q := range quests {
		for _, step := range q.Steps {
			if strings.TrimSpace(step.Hint) == "" {
				missing = append(missing,
					fmt.Sprintf("quest %d (%s) step %q has no hint", q.QuestId, q.Name, step.Id))
			}
		}
	}

	if len(missing) > 0 {
		t.Errorf("Quest steps missing hints:\n  %s", strings.Join(missing, "\n  "))
	}
}
```

- [ ] **Step 2: Run the test**

Run: `go test ./internal/questengine/ -run TestAllQuestStepsHaveHints -v`
Expected: Either SKIP (no quest files with triggers yet) or FAIL (existing quests lack hints). Both are acceptable at this stage — the test will enforce the rule as quests are ported in Phase 3+.

- [ ] **Step 3: Commit**

```bash
git add internal/questengine/validation_test.go
git commit -m "feat: validation test requiring hints on all quest steps"
```

---

### Task 6: Dialogue Self-Reference Guard Test

**Files:**
- Modify: `internal/questengine/validation_test.go`

- [ ] **Step 1: Write the self-reference test**

Append to `internal/questengine/validation_test.go`:

```go
import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/mobs"
	"gopkg.in/yaml.v2"
)

// skipWords are common words that appear in mob names but are also
// normal speech. Don't flag these as self-references.
var skipWords = map[string]bool{
	"the": true, "old": true, "a": true, "an": true,
	"of": true, "in": true, "on": true, "at": true,
	"to": true, "for": true, "and": true, "with": true,
}

// dialogueFileForTest is a minimal struct for loading dialogue YAML in tests.
type dialogueFileForTest struct {
	Tree struct {
		Root struct {
			Text  string `yaml:"text"`
			Hints string `yaml:"hints"`
		} `yaml:"root"`
		Nodes []struct {
			Text  string `yaml:"text"`
			Hints string `yaml:"hints"`
		} `yaml:"nodes"`
	} `yaml:"tree"`
	Patterns []struct {
		Text  string `yaml:"text"`
		Hints string `yaml:"hints"`
	} `yaml:"patterns"`
}

func TestDialogueSelfReference(t *testing.T) {
	dataPath := configs.GetFilePathsConfig().DataFiles.String()
	dialogueRoot := filepath.Join(dataPath, "dialogue")

	if _, err := os.Stat(dialogueRoot); os.IsNotExist(err) {
		t.Skip("No dialogue directory found, skipping self-reference test")
	}

	var violations []string

	// Walk all dialogue YAML files
	filepath.Walk(dialogueRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}

		// Extract mob ID from filename (e.g., "107.yaml" → 107)
		base := strings.TrimSuffix(filepath.Base(path), ".yaml")
		mobId, err := strconv.Atoi(base)
		if err != nil {
			return nil // skip non-numeric filenames
		}

		// Get mob spec to find the name
		mobSpec := mobs.GetMobSpec(mobs.MobId(mobId))
		if mobSpec == nil {
			return nil // mob spec not loaded in test env, skip
		}

		mobName := mobSpec.Character.Name
		if mobName == "" {
			return nil
		}

		// Split name into significant words
		nameWords := []string{}
		for _, word := range strings.Fields(mobName) {
			lower := strings.ToLower(word)
			if !skipWords[lower] && len(lower) > 2 {
				nameWords = append(nameWords, lower)
			}
		}
		if len(nameWords) == 0 {
			return nil
		}

		// Load and parse dialogue file
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		var df dialogueFileForTest
		if err := yaml.Unmarshal(data, &df); err != nil {
			return nil
		}

		// Check all text and hints fields
		relPath, _ := filepath.Rel(dataPath, path)
		checkField := func(fieldType string, index int, content string) {
			lower := strings.ToLower(content)
			for _, word := range nameWords {
				if strings.Contains(lower, word) {
					violations = append(violations,
						fmt.Sprintf("%s: %s[%d] contains mob's own name word %q (mob: %s)",
							relPath, fieldType, index, word, mobName))
				}
			}
		}

		checkField("tree.root.text", 0, df.Tree.Root.Text)
		checkField("tree.root.hints", 0, df.Tree.Root.Hints)
		for i, node := range df.Tree.Nodes {
			checkField("tree.node.text", i, node.Text)
			checkField("tree.node.hints", i, node.Hints)
		}
		for i, pat := range df.Patterns {
			checkField("pattern.text", i, pat.Text)
			checkField("pattern.hints", i, pat.Hints)
		}

		return nil
	})

	if len(violations) > 0 {
		t.Errorf("NPC dialogue self-references found (%d):\n  %s",
			len(violations), strings.Join(violations, "\n  "))
	}
}
```

Note: Merge all import blocks into the single import at the top of validation_test.go. The final file should have one `import` block.

- [ ] **Step 2: Run the test**

Run: `go test ./internal/questengine/ -run TestDialogueSelfReference -v`
Expected: Either SKIP (no mob specs loaded in test env) or PASS/FAIL depending on current dialogue data. If it fails, the violations list shows exactly which files need fixing.

- [ ] **Step 3: Commit**

```bash
git add internal/questengine/validation_test.go
git commit -m "feat: validation test for NPC dialogue third-person self-references"
```

---

### Task 7: Full Build + Test Verification

- [ ] **Step 1: Full build**

Run: `go build ./...`
Expected: Compiles without errors

- [ ] **Step 2: Run questengine tests**

Run: `go test ./internal/questengine/ -v -count=1`
Expected: All PASS (validation tests may SKIP if no data files in test env)

- [ ] **Step 3: Run full test suite**

Run: `go test ./... -count=1`
Expected: All PASS

- [ ] **Step 4: Commit (if any fixes were needed)**

Only if test failures required fixes.

---

## Notes for Later Phases

**Quest YAML hint population:** When quests are ported in Phase 3+, every step MUST include a `hint` field. The validation test will enforce this.

**Tutorial NPC mention:** When porting Quest 1 (Sanctum Trials) in Phase 3, add a line to the Chrysalis Priest or Combat Trainer dialogue mentioning the `hint` command.

**Content generation skill updates:** Update `/new-mob`, `/new-quest`, and `/sketch-quest` skills to include the self-review checklist from the spec. This is a skill file edit, not a code change — do it when those skills are next modified.
