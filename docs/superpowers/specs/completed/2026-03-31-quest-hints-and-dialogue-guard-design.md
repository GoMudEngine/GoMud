# Quest Hints & Dialogue Self-Reference Guard — Design Spec

## Overview

Two related features that improve quest discoverability and prevent a
recurring content bug.

1. A `hint` command that shows the player a fourth-wall hint for their
   current quest step, formatted distinctly from in-character dialogue.
2. Build-time validation tests that catch NPC dialogue referring to the
   speaking NPC in third person, and that enforce full hint coverage on
   every quest step.
3. Strengthened content-generation skills to prevent the third-person
   bug at authoring time.

## Goals

- Players always have a way to get unstuck: `hint`
- Quest authors are required to write a hint for every step
- NPCs never refer to themselves by name in their own dialogue
- Content generation skills enforce these rules before output

---

## Feature 1: `hint` Command

### Behavior

| Input | Result |
|-------|--------|
| `hint` | Show hint for the most recently progressed quest |
| `hint <questname>` | Show hint for the named quest (partial match) |
| `hint <number>` | Show hint for quest with that ID |
| No active quests | "You don't have any active quests." |
| Named quest not found | "You don't have an active quest by that name." |

### Output Format

```
Quest Hint (The Sanctum Trials): Ask the combat trainer about fighting techniques.
```

ANSI formatting:
- `<ansi fg="yellow">Quest Hint</ansi>` for the label
- `<ansi fg="yellow-bold">(<quest name>)</ansi>` for the quest name
- `<ansi fg="white-bold">hint text</ansi>` for the hint body

### Data Source

The `QuestStep.Hint` field already exists in `questengine/types.go`:

```go
type QuestStep struct {
    Id          string `yaml:"id"`
    Description string `yaml:"description,omitempty"`
    Hint        string `yaml:"hint,omitempty"`
}
```

Quest YAML authors populate it per step:

```yaml
steps:
  - id: combat_defeat
    description: "Defeat the training dummy"
    hint: "Ask the combat trainer about fighting techniques."
```

### Most-Recent-Quest Tracking

Add a `LastQuestId int` field to `Character` (persisted in YAML save).
Updated whenever `GiveQuestToken` succeeds. The `hint` command reads
this field to determine which quest to show by default.

### Current Step Resolution

Given a quest ID and the player's `QuestProgress` map, the current step
is the one AFTER the player's recorded progress. For example, if
`QuestProgress[1] = "mutation"`, the current step is the one after
`mutation` in the quest's `Steps` list. The hint displayed is the hint
for that next step.

If the player's progress is on the last step (`end`), the quest is
complete — skip it and fall back to the next most recent active quest.

### Lookup Flow

```
hint command
  │
  ├─ No argument: use character.LastQuestId
  ├─ Numeric argument: use as quest ID
  ├─ String argument: search quest names (case-insensitive partial match)
  │
  ├─ Load QuestDef from questengine
  ├─ Get player's current step from QuestProgress
  ├─ Find the NEXT step in the Steps list
  ├─ Display that step's Hint field
  └─ Done
```

### Tutorial NPC Discovery

One of the early tutorial NPCs (e.g., the Chrysalis Priest or Combat
Trainer) should mention the `hint` command in dialogue. Example:

> "If you ever feel lost, try typing `hint` — it may point you in the
> right direction."

The `hint` command should also be mentioned in the existing hints/tips
broadcast rotation.

---

## Feature 2: Hint Coverage Validation Test

### What It Tests

Every `QuestStep` in every loaded quest definition MUST have a non-empty
`Hint` field. No exceptions.

### Implementation

In `internal/questengine/validation_test.go`:

```go
func TestAllQuestStepsHaveHints(t *testing.T) {
    // Load all quest definitions
    // For each quest, for each step:
    //   assert step.Hint != ""
    //   with message: "quest %d (%s) step %q has no hint"
}
```

This test loads quest YAML files using the same loader infrastructure as
the engine. It runs at build time and fails the build if any step is
missing a hint.

---

## Feature 3: Dialogue Self-Reference Guard Test

### What It Tests

No NPC dialogue file should contain the NPC's own name in its `text` or
`hints` fields. This catches the pattern where an NPC refers to
themselves in third person (e.g., Sylara saying "Ask Sylara about...").

### Implementation

In `internal/questengine/validation_test.go` (or a new
`internal/dialogue/validation_test.go` if better scoped):

```
For each dialogue YAML file:
  1. Extract mob ID from filename
  2. Look up mob spec to get mob name
  3. Split name into significant words
     (skip common words: "the", "old", "a", "an")
  4. For each tree node and pattern:
     - Scan `text` and `hints` fields (case-insensitive)
     - Flag if any significant name word appears
  5. Report: file, field type, node/pattern index, offending word
```

### Skip List

Common words that appear in mob names but are also normal speech:

```
the, old, a, an, of, in, on, at, to, for, and, with
```

This list may grow over time as false positives are discovered.

### What It Catches

- Sylara's dialogue mentioning "Sylara"
- "Guard Captain Velk" dialogue mentioning "Velk"
- "Elder Saris" dialogue mentioning "Saris"

### What It Doesn't Catch

Pronoun self-references ("Ask her about...") — acceptable miss. The
false positive rate for pronoun detection is too high to be useful.
The `hint` command system reduces the need for dialogue to carry
discovery hints, which should naturally reduce the pronoun problem.

---

## Feature 4: Content Generation Skill Updates

### Skills to Update

- `/new-mob` — add self-review checklist for dialogue
- `/new-quest` — add self-review checklist for dialogue + hint coverage
- `/sketch-quest` — add hint field to quest step planning output

### Self-Review Checklist (added to each skill)

Before outputting dialogue for any NPC:

1. Does any `text` field mention the NPC's own name? If yes, rewrite.
2. Does any `hints` field use third-person pronouns (he/she/they) that
   refer to the speaking NPC? If yes, rewrite from player perspective.
3. Does every quest step have a `hint` field? If no, add one.
4. Are `hints` fields written from the player's perspective ("You could
   ask about...") not narrator perspective ("Ask her about...")?

---

## Character Field Change

### New Field

```go
type Character struct {
    // ... existing fields ...
    LastQuestId int `yaml:"lastquestid,omitempty"`
}
```

### Update Point

In `Character.GiveQuestToken()`, after a successful token grant:

```go
questId, _ := quests.TokenToParts(questToken)
c.LastQuestId = questId
```

---

## Files Touched

### New files
- `internal/usercommands/hint.go` — hint command implementation
- Help template: `_datafiles/.../templates/help/hint.template`

### Modified files
- `internal/characters/character.go` — add `LastQuestId` field + update
  in `GiveQuestToken`
- `internal/questengine/validation_test.go` — hint coverage test +
  dialogue self-reference test
- `internal/usercommands/usercommands.go` — register `hint` command
- Content generation skills (`/new-mob`, `/new-quest`, `/sketch-quest`)
- Tutorial NPC dialogue (Chrysalis Priest or Combat Trainer)
- Hints/tips broadcast rotation — add `hint` command mention

### Quest YAML updates
- All 17 quest definitions need `hint` fields on every step (will fail
  the validation test until populated)
