# Code Cleanup 1.1: Behavior Tree Package Split Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development
> (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps
> use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Split `internal/behaviortree/actions.go` (1005 lines) into 6 themed files, split
`conditions.go` (408 lines) into 4 themed files, extract duplicated param helpers into shared
`params.go`. Zero behavior change.

**Architecture:** Pure file-reorganization refactor. All files remain in the `behaviortree`
package, so Go's namespace is unchanged. Each task extracts one themed file, verifies build +
tests pass, then commits. Registry (`init()`), `ActionNode`, `ConditionNode`, and shared state
stay in the thin `actions.go`/`conditions.go`.

**Tech Stack:** Go, existing behaviortree package

**Spec:** `docs/superpowers/specs/2026-04-16-code-cleanup-1.1-btree-package-split-design.md`

---

## Full File/Function Inventory

### actions.go (1005 lines, 37 registered actions)

**Action functions:**
- `actRespond`, `actSay`, `actEmote` — dialogue
- `actGrantQuest`, `actSetQuestFlag`, `actGiveItem`, `actReturnItem`, `actTakeItem`,
  `actGiveGold`, `actTakeGold`, `actGiveItemMultiple`, `actSetMiscData`,
  `actGrantMutation` — quest
- `actMove`, `actAttack`, `actFlee`, `actCast`, `actAddBuff`, `actRemoveBuff` — combat
- `actSpawnMob`, `actAddTempExit`, `actSetState`, `actCommand` — mixed
- `actSummonCompanion`, `actSetRoomLocked`, `actSpawnItemInRoom`,
  `actCommandMob` — mixed
- `actIncrementState`, `actDecrementState` — state
- `actMobSay`, `actMobEmote` — dialogue
- `actSendUserText`, `actSendRoomText`, `actIntercept` — room
- `actMovePlayer` — room
- `actCreateInstance`, `actOpenInstancePortal` — mob
- Helpers: `splitTwo`, `parseIntStr` (only used by `actOpenInstancePortal`)

**Package-level:**
- Package helpers: `getIntParam`, `getStringParam` (DUPLICATED in conditions.go)
- Type: `ActionNode` with `Evaluate()` method
- Package vars: `actionRegistry`, `delayedActions`
- Exports: `LookupAction`
- `init()` registering all 37 action names

`delayedActions` map contains: respond, say, emote, attack, flee, cast, move,
add_buff, command_mob, mob_say, mob_emote

### conditions.go (408 lines, 23 registered conditions)

**Condition functions:**
- `condKeywordMatch`, `condItemMatches`, `condTimeOfDay`, `condRoundMod`,
  `condRandomChance`, `condStateEquals`, `condStateGreaterThan` — state/general
- `condPlayerHasQuest`, `condPlayerMissingQuest`, `condPlayerHasItem`,
  `condPlayerHasGold`, `condPlayerHasFlag`, `condPlayerHasSpell`,
  `condPlayerHasMiscData`, `condPlayersInRoom`, `condMultipleEnemies` — player
- `condMobInCombat`, `condMobHealthBelow`, `condMobAtHome`, `condMobHasBuff`,
  `condMobInRoom` — mob
- `condCommandMatches`, `condCommandRestContains` — room

**Package-level:**
- Package helpers: `getIntParam`, `getStringParam` (DUPLICATED from actions.go)
- Type: `ConditionNode` with `Evaluate()` method
- Package var: `conditionRegistry`
- Exports: `LookupCondition`
- `init()` registering all 23 conditions

### loader.go

Line ~140 uses `getIntParam()` — params.go must exist before duplicates are removed.

---

## Target File Layout

After all tasks complete:

```
internal/behaviortree/
  actions.go           # ActionFunc type, actionRegistry, delayedActions,
                       #   ActionNode+Evaluate, LookupAction, init() (37 registrations)
  actions_combat.go    # actAttack, actFlee, actCast, actAddBuff, actRemoveBuff
  actions_dialogue.go  # actRespond, actSay, actEmote, actSendUserText,
                       #   actSendRoomText, actMobSay, actMobEmote
  actions_mob.go       # actSpawnMob, actSummonCompanion, actCommand, actCommandMob,
                       #   actMove, actOpenInstancePortal, actCreateInstance,
                       #   splitTwo, parseIntStr
  actions_quest.go     # actGrantQuest, actSetQuestFlag, actGrantMutation,
                       #   actGiveGold, actTakeGold, actGiveItem, actReturnItem,
                       #   actTakeItem, actGiveItemMultiple, actSetMiscData
  actions_room.go      # actSetRoomLocked, actSpawnItemInRoom, actAddTempExit,
                       #   actMovePlayer, actIntercept
  actions_state.go     # actSetState, actIncrementState, actDecrementState
  conditions.go        # ConditionFunc type, conditionRegistry,
                       #   ConditionNode+Evaluate, LookupCondition, init() (23 registrations)
  conditions_mob.go    # condMobInCombat, condMobHealthBelow, condMobAtHome,
                       #   condMobHasBuff, condMobInRoom
  conditions_player.go # condPlayerHasQuest, condPlayerMissingQuest,
                       #   condPlayerHasItem, condPlayerHasGold, condPlayerHasFlag,
                       #   condPlayerHasSpell, condPlayerHasMiscData,
                       #   condPlayersInRoom, condMultipleEnemies
  conditions_room.go   # condCommandMatches, condCommandRestContains
  conditions_state.go  # condStateEquals, condStateGreaterThan, condKeywordMatch,
                       #   condItemMatches, condTimeOfDay, condRoundMod,
                       #   condRandomChance
  params.go            # getIntParam, getStringParam
  loader.go            # (unchanged)
  ... (other existing files unchanged)
```

---

## Tasks

---

### Task 1: Create params.go (extract shared helpers)

**Files:**
- Create: `internal/behaviortree/params.go`
- Modify: `internal/behaviortree/actions.go` (remove `getIntParam`/`getStringParam`)
- Modify: `internal/behaviortree/conditions.go` (remove `getIntParam`/`getStringParam`)

- [ ] **Step 1: Read current helper implementations**
  Read `internal/behaviortree/actions.go`. Locate `getIntParam` and `getStringParam`
  function bodies. Note the EXACT signatures and logic — the conditions.go versions
  may differ slightly.

- [ ] **Step 2: Create params.go**
  Create `internal/behaviortree/params.go` with `package behaviortree` and both
  helpers copied verbatim from actions.go. If conditions.go has a divergent
  implementation of either helper, reconcile them (prefer the more complete version)
  and use that single version in params.go.

  Structure (copy function bodies verbatim from actions.go):
  ```go
  package behaviortree

  // getIntParam reads an integer parameter from the params map.
  // Handles both int and float64 YAML values. Returns 0 if missing.
  func getIntParam(params map[string]any, key string) int {
      // ... copy from actions.go verbatim ...
  }

  // getStringParam reads a string parameter from the params map.
  // Returns empty string if missing or wrong type.
  func getStringParam(params map[string]any, key string) string {
      // ... copy from actions.go verbatim ...
  }
  ```

- [ ] **Step 3: Remove from actions.go**
  Delete the `getIntParam` and `getStringParam` function bodies from
  `internal/behaviortree/actions.go`.

- [ ] **Step 4: Remove from conditions.go**
  Delete the `getIntParam` and `getStringParam` function bodies from
  `internal/behaviortree/conditions.go`.

- [ ] **Step 5: Build**
  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go build ./...
  ```
  Fix any errors before proceeding.

- [ ] **Step 6: Test**
  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go test ./internal/behaviortree/...
  ```

- [ ] **Step 7: Commit**
  ```bash
  git add internal/behaviortree/params.go \
          internal/behaviortree/actions.go \
          internal/behaviortree/conditions.go
  git commit -m "$(cat <<'EOF'
  refactor(behaviortree): extract shared param helpers into params.go

  Removes duplicated getIntParam/getStringParam from actions.go and
  conditions.go into a single shared params.go. Zero behavior change.

  Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
  EOF
  )"
  ```

---

### Task 2: Extract actions_state.go

**Files:**
- Create: `internal/behaviortree/actions_state.go`
- Modify: `internal/behaviortree/actions.go` (remove the 3 functions)

**Functions to move:** `actSetState`, `actIncrementState`, `actDecrementState`

- [ ] **Step 1: Read functions from actions.go**
  Open `internal/behaviortree/actions.go`. Locate `actSetState`, `actIncrementState`,
  and `actDecrementState`. Note their full signatures, bodies, and which packages
  they import.

- [ ] **Step 2: Create actions_state.go**
  Create `internal/behaviortree/actions_state.go`. Add `package behaviortree` at
  top. Add an `import` block containing only the packages actually used by these
  three functions. Copy each function body verbatim from actions.go.

- [ ] **Step 3: Remove functions from actions.go**
  Delete all three function bodies from `internal/behaviortree/actions.go`. Remove
  any imports that are no longer needed by the remaining code in actions.go.

- [ ] **Step 4: Build**
  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go build ./...
  ```

- [ ] **Step 5: Test**
  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go test ./internal/behaviortree/...
  ```

- [ ] **Step 6: Commit**
  ```bash
  git add internal/behaviortree/actions_state.go \
          internal/behaviortree/actions.go
  git commit -m "$(cat <<'EOF'
  refactor(behaviortree): extract state actions into actions_state.go

  Moves actSetState, actIncrementState, actDecrementState out of the
  monolithic actions.go. Zero behavior change.

  Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
  EOF
  )"
  ```

---

### Task 3: Extract actions_room.go

**Files:**
- Create: `internal/behaviortree/actions_room.go`
- Modify: `internal/behaviortree/actions.go` (remove the 5 functions)

**Functions to move:** `actSetRoomLocked`, `actSpawnItemInRoom`, `actAddTempExit`,
`actMovePlayer`, `actIntercept`

- [ ] **Step 1: Read functions from actions.go**
  Open `internal/behaviortree/actions.go`. Locate all five functions. Note their
  full bodies and imports used.

- [ ] **Step 2: Create actions_room.go**
  Create `internal/behaviortree/actions_room.go`. Add `package behaviortree` at top.
  Add an `import` block for only the packages used by these five functions. Copy
  each function body verbatim.

- [ ] **Step 3: Remove functions from actions.go**
  Delete all five function bodies from `internal/behaviortree/actions.go`. Clean up
  any now-unused imports.

- [ ] **Step 4: Build**
  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go build ./...
  ```

- [ ] **Step 5: Test**
  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go test ./internal/behaviortree/...
  ```

- [ ] **Step 6: Commit**
  ```bash
  git add internal/behaviortree/actions_room.go \
          internal/behaviortree/actions.go
  git commit -m "$(cat <<'EOF'
  refactor(behaviortree): extract room actions into actions_room.go

  Moves actSetRoomLocked, actSpawnItemInRoom, actAddTempExit, actMovePlayer,
  actIntercept out of the monolithic actions.go. Zero behavior change.

  Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
  EOF
  )"
  ```

---

### Task 4: Extract actions_combat.go

**Files:**
- Create: `internal/behaviortree/actions_combat.go`
- Modify: `internal/behaviortree/actions.go` (remove the 5 functions)

**Functions to move:** `actAttack`, `actFlee`, `actCast`, `actAddBuff`, `actRemoveBuff`

- [ ] **Step 1: Read functions from actions.go**
  Open `internal/behaviortree/actions.go`. Locate all five functions. Note their
  full bodies and imports used.

- [ ] **Step 2: Create actions_combat.go**
  Create `internal/behaviortree/actions_combat.go`. Add `package behaviortree` at
  top. Add an `import` block for only the packages used by these five functions.
  Copy each function body verbatim.

- [ ] **Step 3: Remove functions from actions.go**
  Delete all five function bodies from `internal/behaviortree/actions.go`. Clean up
  any now-unused imports.

- [ ] **Step 4: Build**
  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go build ./...
  ```

- [ ] **Step 5: Test**
  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go test ./internal/behaviortree/...
  ```

- [ ] **Step 6: Commit**
  ```bash
  git add internal/behaviortree/actions_combat.go \
          internal/behaviortree/actions.go
  git commit -m "$(cat <<'EOF'
  refactor(behaviortree): extract combat actions into actions_combat.go

  Moves actAttack, actFlee, actCast, actAddBuff, actRemoveBuff out of
  the monolithic actions.go. Zero behavior change.

  Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
  EOF
  )"
  ```

---

### Task 5: Extract actions_dialogue.go

**Files:**
- Create: `internal/behaviortree/actions_dialogue.go`
- Modify: `internal/behaviortree/actions.go` (remove the 7 functions)

**Functions to move:** `actRespond`, `actSay`, `actEmote`, `actSendUserText`,
`actSendRoomText`, `actMobSay`, `actMobEmote`

- [ ] **Step 1: Read functions from actions.go**
  Open `internal/behaviortree/actions.go`. Locate all seven functions. Note their
  full bodies and imports used.

- [ ] **Step 2: Create actions_dialogue.go**
  Create `internal/behaviortree/actions_dialogue.go`. Add `package behaviortree` at
  top. Add an `import` block for only the packages used by these seven functions.
  Copy each function body verbatim.

- [ ] **Step 3: Remove functions from actions.go**
  Delete all seven function bodies from `internal/behaviortree/actions.go`. Clean up
  any now-unused imports.

- [ ] **Step 4: Build**
  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go build ./...
  ```

- [ ] **Step 5: Test**
  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go test ./internal/behaviortree/...
  ```

- [ ] **Step 6: Commit**
  ```bash
  git add internal/behaviortree/actions_dialogue.go \
          internal/behaviortree/actions.go
  git commit -m "$(cat <<'EOF'
  refactor(behaviortree): extract dialogue actions into actions_dialogue.go

  Moves actRespond, actSay, actEmote, actSendUserText, actSendRoomText,
  actMobSay, actMobEmote out of the monolithic actions.go. Zero behavior change.

  Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
  EOF
  )"
  ```

---

### Task 6: Extract actions_quest.go

**Files:**
- Create: `internal/behaviortree/actions_quest.go`
- Modify: `internal/behaviortree/actions.go` (remove the 10 functions)

**Functions to move:** `actGrantQuest`, `actSetQuestFlag`, `actGrantMutation`,
`actGiveGold`, `actTakeGold`, `actGiveItem`, `actReturnItem`, `actTakeItem`,
`actGiveItemMultiple`, `actSetMiscData`

Note: `grant_quest_to_user` is an ALIAS for `actGrantQuest` registered in `init()`.
It is a registry entry only — there is no separate function to move for it. The
`init()` registration stays in `actions.go`.

- [ ] **Step 1: Read functions from actions.go**
  Open `internal/behaviortree/actions.go`. Locate all ten functions. Note their
  full bodies and imports used.

- [ ] **Step 2: Create actions_quest.go**
  Create `internal/behaviortree/actions_quest.go`. Add `package behaviortree` at
  top. Add an `import` block for only the packages used by these ten functions.
  Copy each function body verbatim.

- [ ] **Step 3: Remove functions from actions.go**
  Delete all ten function bodies from `internal/behaviortree/actions.go`. Clean up
  any now-unused imports.

- [ ] **Step 4: Build**
  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go build ./...
  ```

- [ ] **Step 5: Test**
  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go test ./internal/behaviortree/...
  ```

- [ ] **Step 6: Commit**
  ```bash
  git add internal/behaviortree/actions_quest.go \
          internal/behaviortree/actions.go
  git commit -m "$(cat <<'EOF'
  refactor(behaviortree): extract quest actions into actions_quest.go

  Moves actGrantQuest, actSetQuestFlag, actGrantMutation, actGiveGold,
  actTakeGold, actGiveItem, actReturnItem, actTakeItem, actGiveItemMultiple,
  actSetMiscData out of the monolithic actions.go. Zero behavior change.

  Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
  EOF
  )"
  ```

---

### Task 7: Extract actions_mob.go

**Files:**
- Create: `internal/behaviortree/actions_mob.go`
- Modify: `internal/behaviortree/actions.go` (remove the 7 functions + 2 helpers)

**Functions to move:** `actSpawnMob`, `actSummonCompanion`, `actCommand`,
`actCommandMob`, `actMove`, `actOpenInstancePortal`, `actCreateInstance`

**Helpers to move:** `splitTwo`, `parseIntStr`
(These are only called by `actOpenInstancePortal`, so they travel with it.)

After this task, `actions.go` should contain ONLY:
- `ActionFunc` type
- `actionRegistry` var
- `delayedActions` map
- `ActionNode` struct + `Evaluate()` method
- `LookupAction` function
- `init()` with all 37 registrations

- [ ] **Step 1: Read functions from actions.go**
  Open `internal/behaviortree/actions.go`. Locate all seven action functions plus
  `splitTwo` and `parseIntStr`. Note their full bodies and imports used.

- [ ] **Step 2: Create actions_mob.go**
  Create `internal/behaviortree/actions_mob.go`. Add `package behaviortree` at top.
  Add an `import` block for only the packages used by these functions. Copy each
  function body verbatim, including `splitTwo` and `parseIntStr`.

- [ ] **Step 3: Remove functions from actions.go**
  Delete all nine function bodies (7 actions + 2 helpers) from
  `internal/behaviortree/actions.go`. Clean up any now-unused imports. Verify
  that what remains is only the package-level declarations described above.

- [ ] **Step 4: Build**
  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go build ./...
  ```

- [ ] **Step 5: Test**
  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go test ./internal/behaviortree/...
  ```

- [ ] **Step 6: Verify actions.go is now thin**
  ```bash
  wc -l internal/behaviortree/actions.go
  ```
  Expected: roughly 60-100 lines (type, vars, Evaluate, LookupAction, init).

- [ ] **Step 7: Commit**
  ```bash
  git add internal/behaviortree/actions_mob.go \
          internal/behaviortree/actions.go
  git commit -m "$(cat <<'EOF'
  refactor(behaviortree): extract mob actions into actions_mob.go

  Moves actSpawnMob, actSummonCompanion, actCommand, actCommandMob, actMove,
  actOpenInstancePortal, actCreateInstance (+ splitTwo/parseIntStr helpers)
  out of the monolithic actions.go. actions.go is now a thin registry file.
  Zero behavior change.

  Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
  EOF
  )"
  ```

---

### Task 8: Extract conditions_room.go

**Files:**
- Create: `internal/behaviortree/conditions_room.go`
- Modify: `internal/behaviortree/conditions.go` (remove the 2 functions)

**Functions to move:** `condCommandMatches`, `condCommandRestContains`

- [ ] **Step 1: Read functions from conditions.go**
  Open `internal/behaviortree/conditions.go`. Locate both functions. Note their
  full bodies and imports used.

- [ ] **Step 2: Create conditions_room.go**
  Create `internal/behaviortree/conditions_room.go`. Add `package behaviortree` at
  top. Add an `import` block for only the packages used by these two functions.
  Copy each function body verbatim.

- [ ] **Step 3: Remove functions from conditions.go**
  Delete both function bodies from `internal/behaviortree/conditions.go`. Clean up
  any now-unused imports.

- [ ] **Step 4: Build**
  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go build ./...
  ```

- [ ] **Step 5: Test**
  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go test ./internal/behaviortree/...
  ```

- [ ] **Step 6: Commit**
  ```bash
  git add internal/behaviortree/conditions_room.go \
          internal/behaviortree/conditions.go
  git commit -m "$(cat <<'EOF'
  refactor(behaviortree): extract room conditions into conditions_room.go

  Moves condCommandMatches, condCommandRestContains out of the monolithic
  conditions.go. Zero behavior change.

  Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
  EOF
  )"
  ```

---

### Task 9: Extract conditions_state.go

**Files:**
- Create: `internal/behaviortree/conditions_state.go`
- Modify: `internal/behaviortree/conditions.go` (remove the 7 functions)

**Functions to move:** `condStateEquals`, `condStateGreaterThan`, `condKeywordMatch`,
`condItemMatches`, `condTimeOfDay`, `condRoundMod`, `condRandomChance`

- [ ] **Step 1: Read functions from conditions.go**
  Open `internal/behaviortree/conditions.go`. Locate all seven functions. Note their
  full bodies and imports used.

- [ ] **Step 2: Create conditions_state.go**
  Create `internal/behaviortree/conditions_state.go`. Add `package behaviortree` at
  top. Add an `import` block for only the packages used by these seven functions.
  Copy each function body verbatim.

- [ ] **Step 3: Remove functions from conditions.go**
  Delete all seven function bodies from `internal/behaviortree/conditions.go`. Clean
  up any now-unused imports.

- [ ] **Step 4: Build**
  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go build ./...
  ```

- [ ] **Step 5: Test**
  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go test ./internal/behaviortree/...
  ```

- [ ] **Step 6: Commit**
  ```bash
  git add internal/behaviortree/conditions_state.go \
          internal/behaviortree/conditions.go
  git commit -m "$(cat <<'EOF'
  refactor(behaviortree): extract state/general conditions into conditions_state.go

  Moves condStateEquals, condStateGreaterThan, condKeywordMatch, condItemMatches,
  condTimeOfDay, condRoundMod, condRandomChance out of the monolithic
  conditions.go. Zero behavior change.

  Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
  EOF
  )"
  ```

---

### Task 10: Extract conditions_mob.go

**Files:**
- Create: `internal/behaviortree/conditions_mob.go`
- Modify: `internal/behaviortree/conditions.go` (remove the 5 functions)

**Functions to move:** `condMobInCombat`, `condMobHealthBelow`, `condMobAtHome`,
`condMobHasBuff`, `condMobInRoom`

- [ ] **Step 1: Read functions from conditions.go**
  Open `internal/behaviortree/conditions.go`. Locate all five functions. Note their
  full bodies and imports used.

- [ ] **Step 2: Create conditions_mob.go**
  Create `internal/behaviortree/conditions_mob.go`. Add `package behaviortree` at
  top. Add an `import` block for only the packages used by these five functions.
  Copy each function body verbatim.

- [ ] **Step 3: Remove functions from conditions.go**
  Delete all five function bodies from `internal/behaviortree/conditions.go`. Clean
  up any now-unused imports.

- [ ] **Step 4: Build**
  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go build ./...
  ```

- [ ] **Step 5: Test**
  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go test ./internal/behaviortree/...
  ```

- [ ] **Step 6: Commit**
  ```bash
  git add internal/behaviortree/conditions_mob.go \
          internal/behaviortree/conditions.go
  git commit -m "$(cat <<'EOF'
  refactor(behaviortree): extract mob conditions into conditions_mob.go

  Moves condMobInCombat, condMobHealthBelow, condMobAtHome, condMobHasBuff,
  condMobInRoom out of the monolithic conditions.go. Zero behavior change.

  Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
  EOF
  )"
  ```

---

### Task 11: Extract conditions_player.go

**Files:**
- Create: `internal/behaviortree/conditions_player.go`
- Modify: `internal/behaviortree/conditions.go` (remove the 9 functions)

**Functions to move:** `condPlayerHasQuest`, `condPlayerMissingQuest`,
`condPlayerHasItem`, `condPlayerHasGold`, `condPlayerHasFlag`, `condPlayerHasSpell`,
`condPlayerHasMiscData`, `condPlayersInRoom`, `condMultipleEnemies`

After this task, `conditions.go` should contain ONLY:
- `ConditionFunc` type
- `conditionRegistry` var
- `ConditionNode` struct + `Evaluate()` method
- `LookupCondition` function
- `init()` with all 23 registrations

- [ ] **Step 1: Read functions from conditions.go**
  Open `internal/behaviortree/conditions.go`. Locate all nine functions. Note their
  full bodies and imports used.

- [ ] **Step 2: Create conditions_player.go**
  Create `internal/behaviortree/conditions_player.go`. Add `package behaviortree` at
  top. Add an `import` block for only the packages used by these nine functions.
  Copy each function body verbatim.

- [ ] **Step 3: Remove functions from conditions.go**
  Delete all nine function bodies from `internal/behaviortree/conditions.go`. Clean
  up any now-unused imports. Verify that what remains is only the package-level
  declarations described above.

- [ ] **Step 4: Build**
  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go build ./...
  ```

- [ ] **Step 5: Test**
  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go test ./internal/behaviortree/...
  ```

- [ ] **Step 6: Verify conditions.go is now thin**
  ```bash
  wc -l internal/behaviortree/conditions.go
  ```
  Expected: roughly 40-70 lines (type, var, Evaluate, LookupCondition, init).

- [ ] **Step 7: Commit**
  ```bash
  git add internal/behaviortree/conditions_player.go \
          internal/behaviortree/conditions.go
  git commit -m "$(cat <<'EOF'
  refactor(behaviortree): extract player conditions into conditions_player.go

  Moves condPlayerHasQuest, condPlayerMissingQuest, condPlayerHasItem,
  condPlayerHasGold, condPlayerHasFlag, condPlayerHasSpell, condPlayerHasMiscData,
  condPlayersInRoom, condMultipleEnemies out of the monolithic conditions.go.
  conditions.go is now a thin registry file. Zero behavior change.

  Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
  EOF
  )"
  ```

---

### Task 12: Final verification

- [ ] **Step 1: Full build**
  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go build ./...
  ```
  Must produce zero errors or warnings.

- [ ] **Step 2: All behaviortree tests**
  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go test ./internal/behaviortree/...
  ```
  Must pass.

- [ ] **Step 3: Confirm no duplicate helpers**
  ```bash
  grep -l "func getIntParam" internal/behaviortree/*.go
  ```
  Must return only `internal/behaviortree/params.go`.

  ```bash
  grep -l "func getStringParam" internal/behaviortree/*.go
  ```
  Must return only `internal/behaviortree/params.go`.

- [ ] **Step 4: Confirm registry count unchanged**
  ```bash
  grep -c "actionRegistry\[" internal/behaviortree/actions.go
  ```
  Expected: 38 (37 action registrations + 1 map variable declaration).

  ```bash
  grep -c "conditionRegistry\[" internal/behaviortree/conditions.go
  ```
  Expected: 24 (23 condition registrations + 1 map variable declaration).

- [ ] **Step 5: Verify file count**
  ```bash
  ls internal/behaviortree/*.go
  ```
  Expected new files: `params.go`, `actions_state.go`, `actions_room.go`,
  `actions_combat.go`, `actions_dialogue.go`, `actions_quest.go`, `actions_mob.go`,
  `conditions_room.go`, `conditions_state.go`, `conditions_mob.go`,
  `conditions_player.go` (11 new files total).

- [ ] **Step 6: Verify each themed file has package declaration**
  ```bash
  for f in internal/behaviortree/actions_*.go internal/behaviortree/conditions_*.go \
            internal/behaviortree/params.go; do
    head -1 "$f"
  done
  ```
  Every line must be `package behaviortree`.

- [ ] **Step 7: Final commit**
  ```bash
  git commit --allow-empty -m "$(cat <<'EOF'
  refactor(behaviortree): complete package split — 11 themed files extracted

  actions.go split into 6 files: actions_state, actions_room, actions_combat,
  actions_dialogue, actions_quest, actions_mob. conditions.go split into 4 files:
  conditions_state, conditions_room, conditions_mob, conditions_player.
  Shared helpers extracted to params.go. All 37 actions and 23 conditions
  remain registered. Zero behavior change.

  Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
  EOF
  )"
  ```
  Note: if no unstaged changes remain after Task 11's commit, this step can be
  skipped — the previous commits already capture the full refactor.

---

## Implementation Notes

### Imports per file
Each extracted file needs its own `import` block. The implementer must determine
exact imports by reading what each function actually references. Common packages:

- `github.com/GoMudEngine/GoMud/internal/mobs`
- `github.com/GoMudEngine/GoMud/internal/rooms`
- `github.com/GoMudEngine/GoMud/internal/users`
- `github.com/GoMudEngine/GoMud/internal/characters`
- `github.com/GoMudEngine/GoMud/internal/items`
- `github.com/GoMudEngine/GoMud/internal/buffs`
- `github.com/GoMudEngine/GoMud/internal/quests`
- `fmt`, `strings`, `strconv`

Do NOT add unused imports — Go will refuse to compile. Do NOT omit needed imports.
The safest approach: copy the function, attempt build, add missing imports from
the error output.

### Function bodies must be verbatim
Do NOT paraphrase or rewrite function logic. Copy the exact bytes. This is a
file-split refactor, not a rewrite. Any logic change is a bug.

### Build must pass between every task
If a build fails mid-task, do NOT proceed to the commit step. Fix the error first.
Acceptable causes: missing import (add it), stale unused import (remove it).
Unacceptable cause: logic changes needed — that means something was misidentified.

### actions.go / conditions.go always stay valid
Because all files share the `behaviortree` package, the compiler sees them as one
unit. Removing a function from actions.go while it is still referenced from another
file in the same package is fine — the reference now resolves to the new themed file.
This is why build passes after every move even before init() is touched.
