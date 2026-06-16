# Chunk 9 Phase 1 — Repeatable Quests Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add native repeatable-quest support to the engine (a `repeatable` flag + a per-quest cooldown), then author one repeatable "practice loop" quest per spoke (IDs 53–59) offered off each spoke's existing capstone trainer.

**Architecture:** All quest-token grants funnel through the single hook `HandleQuestUpdate` (`internal/hooks/Quest_HandleQuestUpdate.go`, registered at `hooks.go:58`). On a repeatable quest's `-end`, the hook grants rewards as today, then clears the quest's `QuestProgress` entry (so it can be re-taken) and stamps a cooldown timestamp into the character's persisted `MiscData`. On a repeatable quest's `start`, the hook refuses the grant while the cooldown is still active. Cooldown read/write is encapsulated as `Character` methods so the hook needs no new imports and the logic is unit-testable in the `characters` package using the existing `util.SetRoundCountForTest` seam.

**Tech Stack:** Go; `gopkg.in/yaml.v2` quest loader; existing `Character.QuestProgress map[int]string`, `Character.MiscData map[string]any`, `ClearQuestToken`, `SetMiscData`/`GetMiscData`, `util.GetRoundCount`.

---

## Context the engineer needs (verified 2026-06-16)

- **Quest struct:** `internal/quests/quests.go:58`. Fields are tag-less except `Flags` (`yaml:"flags,omitempty"`). Tag-less fields load from the lowercased Go field name (`QuestId`→`questid`). Explicit yaml tags ARE honored (including underscores) — `Flags` proves it.
- **QuestReward gotcha** (quests.go:31): reward-block subfields must use no-underscore keys (`gold`, `skillinfo`, `itemid`, `playermessage`). This applies to the `rewards:` block only, NOT to new top-level Quest fields.
- **Completion hook:** `internal/hooks/Quest_HandleQuestUpdate.go`. The `-end` branch (`stepName == "end"`, starts ~line 209) grants all rewards; the faction-rep block ends ~line 370 just before `} else {` at line 371. `stepName` is parsed at line 200 (after `GiveQuestToken` at line 193). The handler imports `quests`, `users`, `messaging`, `fmt`, etc. — it does NOT currently import `util`.
- **Character quest API:** `internal/characters/quests.go` — `ClearQuestToken(token)` (line 104, deletes the `QuestProgress[id]` entry), `GiveQuestToken`, `HasQuest`, `IsQuestDone`. `QuestProgress` is `map[int]string` keyed by quest id (`character.go:253`).
- **MiscData:** `SetMiscData(key, any)` / `GetMiscData(key) any` (`character.go:487`/`500`). `MiscData` is persisted (`yaml:"miscdata,omitempty"`, character.go:258). On YAML reload an integer value typically returns as `int`, so reads must coerce across int/int64/uint64/float64.
- **Round counter:** `util.GetRoundCount() uint64` (util.go:123). Test seams: `util.SetRoundCountForTest(r)` and `util.ResetRoundCountForTest()` (util.go:128/133).
- **`characters` already imports `internal/util`** (e.g. alts.go) — safe to use `util.GetRoundCount()` there.
- **Quest YAML template:** see `_datafiles/world/dogmud/quests/32-first_blood.yaml` (multi-step quest with `triggers:` of `mob_death` and `command`+`room` events). Reward block uses no-underscore keys.
- **Capstone trainers (mob ids) + dialogue files** (dialogue files are bare `<mobid>.yaml`):
  - A Martial — Garve **9112** — `dialogue/pothole_coulee/9112.yaml`
  - B Forge — Ovell **9117** — `9117.yaml`
  - C Alchemy — Falv **9129** — `9129.yaml`
  - D Wilderness — Delk **9137** — `9137.yaml`
  - E Folding — Orrin **9145** — `9145.yaml`
  - F Lore — Sere **9160** — `9160.yaml`
  - G Ranged — Bryn **9162** — `9162.yaml`
  (Confirm each path with `ls` before editing — Task 4 step 1.)

---

## File Structure

- **Modify** `internal/quests/quests.go` — add two fields to the `Quest` struct.
- **Modify** `internal/quests/quests_test.go` — add a YAML-load test for the new fields.
- **Modify** `internal/characters/quests.go` — add cooldown methods + a private coercion helper.
- **Create** `internal/characters/questcooldown_test.go` — unit tests for the cooldown methods.
- **Modify** `internal/hooks/Quest_HandleQuestUpdate.go` — add the start-gate and end-clear/stamp; add the `util` import.
- **Create** `_datafiles/world/dogmud/quests/53-…​.yaml` … `59-…​.yaml` — the 7 repeatable quests.
- **Modify** the 7 trainer dialogue files — add a repeatable offer-node each.

---

## Task 1: Add `Repeatable` + `CooldownRounds` to the Quest struct

**Files:**
- Modify: `internal/quests/quests.go:58-66` (the `Quest` struct)
- Test: `internal/quests/quests_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/quests/quests_test.go`:

```go
func TestQuest_RepeatableFieldsLoad(t *testing.T) {
	src := []byte("questid: 9999\nname: Repeat Me\nrepeatable: true\ncooldown_rounds: 200\n")
	var q Quest
	if err := yaml.Unmarshal(src, &q); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !q.Repeatable {
		t.Errorf("Repeatable: want true, got false")
	}
	if q.CooldownRounds != 200 {
		t.Errorf("CooldownRounds: want 200, got %d", q.CooldownRounds)
	}
}

func TestQuest_RepeatableDefaultsFalse(t *testing.T) {
	src := []byte("questid: 9998\nname: One Shot\n")
	var q Quest
	if err := yaml.Unmarshal(src, &q); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if q.Repeatable {
		t.Errorf("Repeatable: want false (default), got true")
	}
	if q.CooldownRounds != 0 {
		t.Errorf("CooldownRounds: want 0 (default), got %d", q.CooldownRounds)
	}
}
```

If `quests_test.go` does not already import the yaml package, add `yaml "gopkg.in/yaml.v2"` to its imports.

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/quests/ -run TestQuest_Repeatable -v`
Expected: FAIL — `q.Repeatable undefined` (compile error) or the field is missing.

- [ ] **Step 3: Add the fields**

In `internal/quests/quests.go`, change the `Quest` struct (currently lines 58-66) to add the two fields after `Flags`:

```go
type Quest struct {
	QuestId        int
	Name           string
	Description    string
	Secret         bool        // Secret quests are useful for marking some progress without making it known to the player
	Steps          []QuestStep // String identifiers for each step required to complete the quest
	Rewards        QuestReward
	Flags          []QuestFlagDef `yaml:"flags,omitempty"`
	Repeatable     bool           `yaml:"repeatable,omitempty"`      // if true, completing the quest clears its progress so it can be taken again (after CooldownRounds)
	CooldownRounds int            `yaml:"cooldown_rounds,omitempty"` // rounds that must pass after completion before a repeatable quest can be re-taken
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/quests/ -run TestQuest_Repeatable -v`
Expected: PASS (both tests).

- [ ] **Step 5: Commit**

```bash
git add internal/quests/quests.go internal/quests/quests_test.go
git commit -m "feat(quests): add Repeatable + CooldownRounds fields to Quest struct"
```

---

## Task 2: Cooldown methods on Character

**Files:**
- Modify: `internal/characters/quests.go` (add methods + helper at end of file, before EOF)
- Test: `internal/characters/questcooldown_test.go` (create)

- [ ] **Step 1: Write the failing test**

Create `internal/characters/questcooldown_test.go`:

```go
package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/util"
)

func TestQuestCooldown_ActiveThenExpires(t *testing.T) {
	defer util.ResetRoundCountForTest()
	util.SetRoundCountForTest(1000)

	c := &Character{}
	// No cooldown set yet -> not active.
	if c.QuestCooldownActive(53) {
		t.Fatalf("no cooldown set: want inactive, got active")
	}

	// Set a 200-round cooldown at round 1000 -> available again at 1200.
	c.SetQuestCooldown(53, 200)
	if !c.QuestCooldownActive(53) {
		t.Fatalf("just-set cooldown at round 1000: want active, got inactive")
	}

	// Still inside the window.
	util.SetRoundCountForTest(1199)
	if !c.QuestCooldownActive(53) {
		t.Fatalf("round 1199 (< 1200): want active, got inactive")
	}

	// At/after the threshold -> expired.
	util.SetRoundCountForTest(1200)
	if c.QuestCooldownActive(53) {
		t.Fatalf("round 1200 (>= 1200): want inactive, got active")
	}
}

func TestQuestCooldown_CoercesReloadedInt(t *testing.T) {
	// MiscData round-trips through YAML as an int; QuestCooldownActive must
	// coerce a plain int value, not only the uint64 it was written as.
	defer util.ResetRoundCountForTest()
	util.SetRoundCountForTest(500)

	c := &Character{}
	c.SetMiscData("questcd-53", int(1000)) // simulate a reloaded value
	if !c.QuestCooldownActive(53) {
		t.Fatalf("int-typed cooldown 1000 at round 500: want active, got inactive")
	}
	util.SetRoundCountForTest(1000)
	if c.QuestCooldownActive(53) {
		t.Fatalf("int-typed cooldown 1000 at round 1000: want inactive, got active")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/characters/ -run TestQuestCooldown -v`
Expected: FAIL — `c.QuestCooldownActive undefined`, `c.SetQuestCooldown undefined`.

- [ ] **Step 3: Implement the methods**

Append to `internal/characters/quests.go` (the file already imports `github.com/GoMudEngine/GoMud/internal/quests` and `mudlog`; add `fmt` and `github.com/GoMudEngine/GoMud/internal/util` to its import block):

```go
// questCooldownKey is the MiscData key under which a repeatable quest's
// "available again at round N" timestamp is stored.
func questCooldownKey(questId int) string {
	return fmt.Sprintf("questcd-%d", questId)
}

// SetQuestCooldown records that quest questId may not be re-taken until
// cooldownRounds rounds from now. Stored in persisted MiscData so the
// cooldown survives logout.
func (c *Character) SetQuestCooldown(questId int, cooldownRounds uint64) {
	c.SetMiscData(questCooldownKey(questId), util.GetRoundCount()+cooldownRounds)
}

// QuestCooldownActive reports whether quest questId is still inside its
// post-completion cooldown window (current round < stored "available at").
func (c *Character) QuestCooldownActive(questId int) bool {
	v := c.GetMiscData(questCooldownKey(questId))
	if v == nil {
		return false
	}
	return util.GetRoundCount() < miscDataToUint64(v)
}

// miscDataToUint64 coerces a MiscData value to uint64. MiscData stores any,
// and an integer written as uint64 may return from YAML reload as int,
// int64, or float64 — handle all of them; anything else yields 0.
func miscDataToUint64(v any) uint64 {
	switch n := v.(type) {
	case uint64:
		return n
	case int:
		if n < 0 {
			return 0
		}
		return uint64(n)
	case int64:
		if n < 0 {
			return 0
		}
		return uint64(n)
	case float64:
		if n < 0 {
			return 0
		}
		return uint64(n)
	default:
		return 0
	}
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/characters/ -run TestQuestCooldown -v`
Expected: PASS (both tests).

- [ ] **Step 5: Commit**

```bash
git add internal/characters/quests.go internal/characters/questcooldown_test.go
git commit -m "feat(characters): add repeatable-quest cooldown methods (MiscData-backed)"
```

---

## Task 3: Wire the hook — start-gate + end-clear/stamp

**Files:**
- Modify: `internal/hooks/Quest_HandleQuestUpdate.go` (imports; the `remove`/start region ~line 185-200; the end of the `-end` block ~line 370)

No unit test for this task — the handler needs the full events/users plumbing and the existing reward-grant logic is not unit-tested in this file (it is verified by the boot test + scripted walkthrough in Task 5). The cooldown logic it calls is fully unit-tested in Task 2.

- [ ] **Step 1: Add the `util` import**

In `internal/hooks/Quest_HandleQuestUpdate.go`, add to the import block:

```go
	"github.com/GoMudEngine/GoMud/internal/util"
```

- [ ] **Step 2: Add the start-gate**

The current code (lines 185-200) is:

```go
	if remove {
		questUser.Character.ClearQuestToken(evt.QuestToken)
		return events.Continue
	}
	// Try to advance the quest. If it fails, check whether the quest engine
	// already set this token (GrantQuest sets it synchronously for chain
	// evaluation, then fires this event for rewards). In that case, proceed
	// with reward processing.
	if !questUser.Character.GiveQuestToken(evt.QuestToken) {
		// Already at this step? The quest engine pre-set it. Continue to rewards.
		if !questUser.Character.HasQuest(evt.QuestToken) {
			return events.Continue
		}
	}

	_, stepName := quests.TokenToParts(evt.QuestToken)
```

Insert the gate immediately after the `if remove { … }` block and BEFORE the `GiveQuestToken` call:

```go
	if remove {
		questUser.Character.ClearQuestToken(evt.QuestToken)
		return events.Continue
	}

	// Repeatable-quest cooldown gate: refuse to (re)start a repeatable quest
	// whose post-completion cooldown has not yet elapsed. Parse the step
	// up-front so we can check before advancing the token.
	{
		_, incomingStep := quests.TokenToParts(evt.QuestToken)
		if incomingStep == `start` && questInfo.Repeatable &&
			questUser.Character.QuestCooldownActive(questInfo.QuestId) {
			questUser.SendText(messaging.CategorySystem,
				`You have put in the work recently. Rest a while before taking this on again.`)
			return events.Continue
		}
	}

	// Try to advance the quest. If it fails, check whether the quest engine
	...
```

(Leave the existing `_, stepName := quests.TokenToParts(evt.QuestToken)` at line 200 as-is.)

- [ ] **Step 3: Add the end-clear/stamp**

Find the end of the `else if stepName == "end"` block — the faction-rep reward is the last block before `} else {`:

```go
		// Faction reputation reward?
		if questInfo.Rewards.RepFaction != "" && questInfo.Rewards.RepAmount != 0 {
			factions.BumpRep(questInfo.Rewards.RepFaction, questUser.UserId, questInfo.Rewards.RepAmount)
		}
	} else {
```

Insert the repeatable reset immediately after the faction-rep `if` block and before the closing `}` of the `end` branch:

```go
		// Faction reputation reward?
		if questInfo.Rewards.RepFaction != "" && questInfo.Rewards.RepAmount != 0 {
			factions.BumpRep(questInfo.Rewards.RepFaction, questUser.UserId, questInfo.Rewards.RepAmount)
		}

		// Repeatable quest: reset progress so it can be taken again, and stamp
		// a cooldown so it cannot be farmed back-to-back. (Non-repeatable
		// quests are unaffected — Repeatable defaults false.)
		if questInfo.Repeatable {
			questUser.Character.ClearQuestToken(evt.QuestToken)
			questUser.Character.SetQuestCooldown(questInfo.QuestId, uint64(questInfo.CooldownRounds))
		}
	} else {
```

- [ ] **Step 4: Build and run the quests/characters/hooks tests**

Run: `go build ./... && go test ./internal/quests/ ./internal/characters/ ./internal/hooks/`
Expected: build exit 0; all three packages PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/hooks/Quest_HandleQuestUpdate.go
git commit -m "feat(quests): repeatable-quest reset+cooldown wiring in HandleQuestUpdate"
```

---

## Task 4: Author the 7 repeatable quests + trainer offer-nodes

This is data authoring against the verified engine. **No new mobs/items/rooms** — every kill/craft/forage target reuses the spoke's existing committed content. Author sequentially (ID-collision SOP) or dispatch one content subagent per spoke sequentially.

**Per-spoke parameters** (foe/room ids: confirm against each spoke's existing cert quests before writing — they are the source of truth):

| Quest | Spoke | Trainer (mobid / dialogue file) | Loop | Verify ids from |
|-------|-------|---------------------------------|------|-----------------|
| 53 | A Martial | Garve 9112 / `9112.yaml` | kill 2× wash bandits (9110 / 9111) | `quests/33-hold_the_wash.yaml` |
| 54 | B Forge | Ovell 9117 / `9117.yaml` | `craft iron dagger` at the forge (station room) | `quests/35-first_heat.yaml` |
| 55 | C Alchemy | Falv 9129 / `9129.yaml` | craft a healing salve + `drink` it | `quests/38-first_brew.yaml` |
| 56 | D Wilderness | Delk 9137 / `9137.yaml` | kill game (hare 9138 / pronghorn 9139) + `forage` | `quests/41-first_sign.yaml`, `42-the_hunt.yaml` |
| 57 | E Folding | Orrin 9145 / `9145.yaml` | kill a grove foe (9147 / 9148) via `cast` | `quests/45-the_unquiet_grove.yaml` |
| 58 | F Lore | Sere 9160 / `9160.yaml` | `search` the standing stones / reliquary room | `quests/48-the_standing_stones.yaml`, `49-the_old_shrine.yaml` |
| 59 | G Ranged | Bryn 9162 / `9162.yaml` | `shoot` down canyon targets (foe 9165 or the practice butt) | `quests/51-across_the_canyon.yaml`, `50-first_shot.yaml` |

- [ ] **Step 1: Verify ids + dialogue files + next free quest id**

```bash
cd "<worktree>"
python tools/id_inventory.py --type quests   # confirm 53-59 are free
ls _datafiles/world/dogmud/dialogue/pothole_coulee/9112.yaml \
   _datafiles/world/dogmud/dialogue/pothole_coulee/9117.yaml \
   _datafiles/world/dogmud/dialogue/pothole_coulee/9129.yaml \
   _datafiles/world/dogmud/dialogue/pothole_coulee/9137.yaml \
   _datafiles/world/dogmud/dialogue/pothole_coulee/9145.yaml \
   _datafiles/world/dogmud/dialogue/pothole_coulee/9160.yaml \
   _datafiles/world/dogmud/dialogue/pothole_coulee/9162.yaml
```

Expected: id_inventory reports 53 as the next free quest id; all 7 dialogue files exist.

- [ ] **Step 2: Write quest 53 (the exemplar) — `_datafiles/world/dogmud/quests/53-cull_the_wash.yaml`**

Open `quests/33-hold_the_wash.yaml` first and copy the exact `mob:` ids and `room:` ids it uses for the wash bandits. Then write:

```yaml
questid: 53
name: Cull the Wash
description: Garve always has more work clearing squatters out of the wash.
  Thin them out and report back.
secret: false
repeatable: true
cooldown_rounds: 300

steps:
  - id: start
    description: "Garve has asked you to cull the squatters infesting the
      wash. Put a couple down and come back."
    hint: "Head into the wash and kill the bandits, then return: ask garve report."
  - id: end
    description: "You thinned the wash again. Garve will have more work
      next time you have the stomach for it."

rewards:
  gold: 6
  skillinfo: "weapon-combat:1"
  playermessage: >-
    Garve counts your tally with a grunt and flips you a few coins.
    "Good. They breed faster than I can swing. Come back when you have
    the stomach for more."
  roommessage: "Garve gives you a short, approving nod."

triggers:
  # First wash foe down (use the bandit mob ids verified from quest 33).
  - event: mob_death
    mob: <BANDIT_ID_1>      # e.g. 9110 — confirm from quest 33
    conditions:
      has: ["53-start"]
      missing: ["53-firstkill"]
    actions:
      - grant: "53-firstkill"
      - send_text: "One down. One more should satisfy Garve."
  # Second wash foe down -> complete.
  - event: mob_death
    mob: <BANDIT_ID_2>      # e.g. 9111 — confirm from quest 33
    conditions:
      has: ["53-firstkill"]
      missing: ["53-end"]
    actions:
      - grant: "53-end"
```

Add a `firstkill` step between `start` and `end` if you use the two-kill pattern above (mirror quest 32's intermediate steps). Keep `skillinfo` a floor-raise of 1 (the handler never downgrades a higher-ranked veteran). Reward block keys are no-underscore per the loader gotcha.

- [ ] **Step 3: Write quests 54–59**

Author each from the Step-2 template, substituting the per-spoke loop from the parameter table. Rules for every file:
- `repeatable: true`, `cooldown_rounds: 300` (uniform for now; tuned later).
- `gold:` in the 4–8 range (repeatables pay less than the 15–50g certs).
- `skillinfo:` = the spoke's skill at level 1 (floor-raise). For 56 use `search:1`; 57 `spellcasting:1`; 58 `rhetoric:1`; 59 `ranged-combat:1`; 54 `blacksmithing:1`; 55 `alchemy:1`.
- Craft loops (54, 55) use a `command: craft` trigger gated by `room:` (the station room) — copy the exact craft trigger + room id from the spoke's cert quest (35 / 38). For 55 add a `command: drink` step (the `drink` Notify already exists).
- Cast loop (57) uses `mob_death` of the grove foe (the kill-by-casting beat) — copy the foe id + grove room from quest 45.
- Search loop (58) uses a `command: search` trigger + room id from quest 48/49.
- Shoot loop (59) uses a `command: shoot` trigger + room id from quest 50/51 (the `shoot` Notify exists).
- Every quest's start node is granted by dialogue (Step 4); the `mob_death`/`command` triggers advance it.

- [ ] **Step 4: Add a repeatable offer-node to each trainer's dialogue file**

For each trainer dialogue file, add ONE tree node that grants the spoke's repeatable `start`. Pattern (example for Garve `9112.yaml`, quest 53) — place it among the `nodes:` list:

```yaml
  - keywords: ["work", "again", "more", "job", "quest", "task"]
    questRequired: "<spoke-capstone>-end"   # only after the player finished the spoke (e.g. 34-end for A)
    grantsQuest: "53-start"
    questExcluded: ["53-start", "53-firstkill", "53-end"]
    triggers: ["work", "again", "more", "job", "quest", "task"]
    text: >-
      There is always more work in the wash. Thin the squatters out and
      come back to me.
    hint: "You could take on more work -- type ask garve work."
```

SOP checklist for every offer-node:
- Include `"quest"` and `"task"` in both `keywords` and `triggers` (Quest NPC Dialogue SOP).
- `questExcluded` lists ALL of the quest's tokens (start + any mid step + end). Because the engine clears progress on completion, this exclusion blocks re-offer only while the quest is in-progress, not after — exactly right for a repeatable.
- `text` is first-person NPC voice; `hint` is narrator/player perspective and names the exact command (Dialogue Voice SOP).
- Gate on the spoke's capstone `-end` via `questRequired` so the repeatable only appears once the player has finished that spoke.
- The cooldown "rest a while" message comes from the engine (Task 3) when a player asks again too soon — the NPC node itself does not need cooldown awareness.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/quests/53-*.yaml _datafiles/world/dogmud/quests/54-*.yaml \
        _datafiles/world/dogmud/quests/55-*.yaml _datafiles/world/dogmud/quests/56-*.yaml \
        _datafiles/world/dogmud/quests/57-*.yaml _datafiles/world/dogmud/quests/58-*.yaml \
        _datafiles/world/dogmud/quests/59-*.yaml \
        _datafiles/world/dogmud/dialogue/pothole_coulee/9112.yaml \
        _datafiles/world/dogmud/dialogue/pothole_coulee/9117.yaml \
        _datafiles/world/dogmud/dialogue/pothole_coulee/9129.yaml \
        _datafiles/world/dogmud/dialogue/pothole_coulee/9137.yaml \
        _datafiles/world/dogmud/dialogue/pothole_coulee/9145.yaml \
        _datafiles/world/dogmud/dialogue/pothole_coulee/9160.yaml \
        _datafiles/world/dogmud/dialogue/pothole_coulee/9162.yaml
git commit -m "content(newbie): 7 repeatable practice-loop quests (53-59) + trainer offer-nodes"
```

---

## Task 5: Verification gate (boot + scripted walkthrough)

This is the Phase-1 review gate. Runs only in a popup-tolerant / user-run window (no-console-popups constraint).

- [ ] **Step 1: Build + full test of touched packages**

Run: `go build ./... && go test ./internal/quests/ ./internal/characters/ ./internal/hooks/`
Expected: build exit 0; all PASS.

- [ ] **Step 2: Nuke instance saves, then boot**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
# start the server (user-run); watch the load lines
```
Expected boot, no panics: `quests.LoadDataFiles() loadedCount=51` (was 44 — +7 repeatables), `ValidateAllFlags` clean, `ValidateZoneConsistency errors=0 warnings=0` (panic mode), `Server Ready`.

- [ ] **Step 3: Scripted AI-port walkthrough — confirm the novel behavior**

Drive a character who has already completed at least one spoke's capstone (e.g. A → `34-end`). For that spoke's repeatable (53):
1. `ask garve work` → quest 53 granted (`53-start` in questprogress).
2. Do the loop (kill the two wash foes) → `53-end` fires; reward gold + skill; player sees the completion + reward.
3. Immediately `ask garve work` again → engine refuses with "You have put in the work recently. Rest a while…"; questprogress does NOT contain 53 (cleared) and is NOT re-set to start.
4. (Optional, fast) Temporarily set `cooldown_rounds: 1` in quest 53 or advance rounds, then confirm `ask garve work` re-grants `53-start` — proving the cooldown expiry path.

Capture the transcript as the artifact (`newbie-c9-phase1-walkthrough.txt`). The save is ground truth where the transcript is ambiguous, but note the AI-port `quit` may not flush the save — verify via transcript.

- [ ] **Step 4: Manifest check (no new rooms, but quests added)**

Run: `python tools/newbie_manifest_check.py`
Expected: 0 FAIL. (If the checker has a quest manifest section, extend it with 53-59; if not, this confirms rooms/mobs unchanged.)

- [ ] **Step 5: Commit any walkthrough fixes, then hand to user review**

Commit fixes found during the walkthrough with a `fix(...)` message. Then stop for the Phase-1 review gate. Exclude the dirty `config.yaml` dev settings + runtime YAMLs from commits (as in C2–C8).

---

## Self-review notes (author)

- **Spec coverage:** This plan covers spec §1 (Phase 1 — engine feature §1a + the 7 repeatables §1b) in full. Spec Phase 2 (schedules/conversations) and Phase 3 (audits/balance) are separate plans authored at their gates, per the chunk's per-phase review cadence.
- **Cooldown storage** (the spec's one open item) is resolved: persisted `MiscData` key `questcd-<id>` holding the "available-again" round, with int/float coercion for YAML reload. No new dialogue/questengine surface — the single `HandleQuestUpdate` choke point carries both the reset and the gate.
- **No-regression:** `Repeatable` defaults false; the end-clear/stamp and start-gate are both guarded by `questInfo.Repeatable`, so all existing quests 1–52 behave identically.
- **Type consistency:** `SetQuestCooldown(int, uint64)` / `QuestCooldownActive(int) bool` / `miscDataToUint64(any) uint64` / `questCooldownKey(int) string` are used consistently across Tasks 2 and 3. `cooldown_rounds` (yaml) ↔ `CooldownRounds` (Go) consistent across Tasks 1 and 4.
