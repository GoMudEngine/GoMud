# Phase 5: JS Scripting Bridge Removal Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the entire JavaScript scripting system — Go/JS bridge, goja dependency, all JS files, admin creation commands — leaving zero JS scripting infrastructure.

**Architecture:** Four sequential streams — (1) remove all scripting caller code from 30 Go files, (2) delete the scripting package + goja dependency, (3) delete JS files + admin creation commands, (4) update documentation.

**Tech Stack:** Go, YAML

**Spec:** `docs/superpowers/specs/2026-04-15-phase5-js-bridge-removal-design.md`

## File Structure

```
Files DELETED entirely:
  internal/scripting/              — entire directory (18 files)
  internal/mobs/newmobfile.go      — CreateNewMobFile (admin mob create)
  internal/spells/newspellfile.go   — CreateNewSpellFile (admin spell create)
  internal/hooks/NewRound_PruneVMs.go — PruneVMs hook (only calls scripting)
  _datafiles/world/default/**/*.js  — 88 JS files
  _datafiles/world/empty/**/*.js    — 19 JS files
  _datafiles/sample-scripts/**/*.js — 8 JS files
  _datafiles/world/dogmud/**/*.js.bak — 31 backup JS files

Files EDITED (scripting calls removed):
  internal/usercommands/ask.go
  internal/usercommands/talk.go
  internal/usercommands/give.go
  internal/usercommands/show.go
  internal/usercommands/go.go
  internal/usercommands/start.go
  internal/usercommands/admin.teleport.go
  internal/usercommands/skill.cast.go
  internal/usercommands/use.go
  internal/usercommands/equip.go
  internal/usercommands/buy.go
  internal/usercommands/usercommands.go
  internal/usercommands/admin.mob.go
  internal/usercommands/admin.spell.go
  internal/mobcommands/cast.go
  internal/mobcommands/aid.go
  internal/mobcommands/go.go
  internal/mobcommands/suicide.go
  internal/hooks/spell_resolution.go
  internal/hooks/NewRound_UserRoundTick.go
  internal/hooks/NewRound_MobRoundTick.go
  internal/hooks/NewRound_IdleMobs.go
  internal/hooks/NewRound_DoCombat_helpers.go
  internal/hooks/NewTurn_PruneBuffs.go
  internal/hooks/Buff_ApplyBuffs.go
  internal/hooks/MobIdle_HandleIdleMobs.go
  internal/hooks/ItemOwnership_CheckItemQuests.go
  internal/hooks/PlayerDrop_HandlePlayerDrop.go
  internal/hooks/PlayerSpawn_HandleJoin.go
  internal/hooks/RoomChange_CleanupEphemeralRooms.go
  internal/hooks/hooks.go
  internal/hooks/hooks_test.go
  internal/plugins/plugins.go
  internal/plugins/plugincallbacks.go
  main.go
  go.mod / go.sum

Documentation UPDATED:
  internal/mobs/context.md
  internal/spells/context.md
  internal/items/context.md
  internal/buffs/context.md
  internal/hooks/context.md
  CLAUDE.md
  .claude/commands/new-quest.md
  .claude/commands/sketch-quest.md
  _datafiles/world/dogmud/templates/admincommands/help/command.mob.template
  _datafiles/world/dogmud/templates/admincommands/help/command.spell.template
```

---

## Task 1: Remove scripting calls from user commands (12 files)

All line numbers are APPROXIMATE. Read each file to find exact locations before editing.

- [ ] **Step 1.1: ask.go** — Read `internal/usercommands/ask.go`. Find the `scripting.TryMobScriptEvent("onAsk", ...)` if-block (~line 123). Delete the entire if-block. Remove `"github.com/GoMudEngine/GoMud/internal/scripting"` from imports if no other scripting references remain.

- [ ] **Step 1.2: talk.go** — Read `internal/usercommands/talk.go`. Find the `scripting.TryMobScriptEvent("onAsk", ...)` if-block (~line 60). Delete it. Remove scripting import.

- [ ] **Step 1.3: give.go** — Read `internal/usercommands/give.go`. Find the `scripting.TryMobScriptEvent("onGive", ...)` if-block (~line 240). Delete it. Remove scripting import.

- [ ] **Step 1.4: show.go** — Read `internal/usercommands/show.go`. Find the `scripting.TryMobScriptEvent("onShow", ...)` if-block (~line 100). Delete it. Remove scripting import.

- [ ] **Step 1.5: go.go** — Read `internal/usercommands/go.go`. Find TWO scripting calls: `scripting.TryRoomScriptEvent("onExit", ...)` (~line 268) and `scripting.TryRoomScriptEvent("onEnter", ...)` (~line 572). Delete both if-blocks. Remove scripting import.

- [ ] **Step 1.6: start.go** — Read `internal/usercommands/start.go`. Find TWO `scripting.TryRoomScriptEvent("onEnter", ...)` calls (~lines 147, 186). Delete both if-blocks. Remove scripting import.

- [ ] **Step 1.7: admin.teleport.go** — Read `internal/usercommands/admin.teleport.go`. Find TWO scripting calls (~lines 100, 149). Delete both if-blocks. Remove scripting import.

- [ ] **Step 1.8: skill.cast.go** — Read `internal/usercommands/skill.cast.go`. Find `scripting.TrySpellScriptEvent("onCast", ...)` (~line 291). Delete the if-block. Remove scripting import.

- [ ] **Step 1.9: use.go** — Read `internal/usercommands/use.go`. Find `scripting.TryItemScriptEvent("onUse", ...)` (~line 107). Delete the if-block. Remove scripting import.

- [ ] **Step 1.10: equip.go** — Read `internal/usercommands/equip.go`. Find TWO `scripting.TryBuffScriptEvent("onStart", ...)` calls (~lines 196, 254). Delete both if-blocks. Remove scripting import.

- [ ] **Step 1.11: buy.go** — Read `internal/usercommands/buy.go`. Find `scripting.TryItemScriptEvent("onPurchase", ...)` (~line 534). Delete the if-block. Remove scripting import.

- [ ] **Step 1.12: usercommands.go** — Read `internal/usercommands/usercommands.go`. This requires special handling:
  - Find `TryRoomScripts()` function (~lines 542-587). **Keep** the quest engine notification block (lines ~546-563 with `questengine.GetEngine().Notify("room_interact", ...)`). **Delete** the `scripting.TryRoomCommand(alias, rest, userId)` call and the entire GMCP WrongDir handling block that follows it (lines ~566-586). After removal, the function should just run the quest engine check and return `false, nil` if not handled.
  - Find the two calls to `TryRoomScripts(...)` in the main command dispatch (~line 351 and the fallthrough ~line 382 area). Keep these calls since the function still does quest engine work.
  - Find `scripting.TryItemCommand(...)` call (~line 382). Delete this if-block.
  - Remove `"github.com/GoMudEngine/GoMud/internal/scripting"` import. Also remove `"fmt"` import if it was only used by the GMCP WrongDir block — check carefully.

- [ ] **Step 1.13: Build verification.** Run:
  ```bash
  cd "C:/Users/Calabe Davis/workspace/DOGMud" && go build ./...
  ```
  Fix any compilation errors.

- [ ] **Step 1.14: Commit.**
  ```bash
  git add internal/usercommands/ask.go internal/usercommands/talk.go \
    internal/usercommands/give.go internal/usercommands/show.go \
    internal/usercommands/go.go internal/usercommands/start.go \
    internal/usercommands/admin.teleport.go internal/usercommands/skill.cast.go \
    internal/usercommands/use.go internal/usercommands/equip.go \
    internal/usercommands/buy.go internal/usercommands/usercommands.go
  git commit -m "$(cat <<'EOF'
  refactor: remove scripting calls from user commands

  Remove all scripting.TryMobScriptEvent, TryRoomScriptEvent,
  TrySpellScriptEvent, TryBuffScriptEvent, and TryItemScriptEvent
  calls from 12 usercommand files. TryRoomScripts retains its
  quest engine notification but no longer calls scripting.TryRoomCommand.

  Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
  EOF
  )"
  ```

---

## Task 2: Remove scripting calls from mob commands (4 files)

- [ ] **Step 2.1: cast.go** — Read `internal/mobcommands/cast.go`. Find `scripting.TryMobScriptEvent` and/or `scripting.TrySpellScriptEvent` calls (~line 84). Delete the if-block(s). Remove scripting import.

- [ ] **Step 2.2: aid.go** — Read `internal/mobcommands/aid.go`. Find `scripting.TryMobScriptEvent` (~line 77). Delete the if-block. Remove scripting import.

- [ ] **Step 2.3: go.go** — Read `internal/mobcommands/go.go`. Find `scripting.TryMobScriptEvent("onPath")` (~line 231). Delete the if-block. Remove scripting import.

- [ ] **Step 2.4: suicide.go** — Read `internal/mobcommands/suicide.go`. Find `scripting.TryMobScriptEvent("onDie")` (~line 147). Delete the if-block. Remove scripting import.

- [ ] **Step 2.5: Build verification.** Run:
  ```bash
  cd "C:/Users/Calabe Davis/workspace/DOGMud" && go build ./...
  ```

- [ ] **Step 2.6: Commit.**
  ```bash
  git add internal/mobcommands/cast.go internal/mobcommands/aid.go \
    internal/mobcommands/go.go internal/mobcommands/suicide.go
  git commit -m "$(cat <<'EOF'
  refactor: remove scripting calls from mob commands

  Remove scripting.TryMobScriptEvent and TrySpellScriptEvent calls
  from cast.go, aid.go, go.go, and suicide.go in mobcommands.

  Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
  EOF
  )"
  ```

---

## Task 3: Remove scripting calls from hooks (12 files)

- [ ] **Step 3.1: Delete NewRound_PruneVMs.go.** This file's only purpose is calling `scripting.PruneVMs()`. Delete it entirely:
  ```bash
  git rm internal/hooks/NewRound_PruneVMs.go
  ```

- [ ] **Step 3.2: hooks.go** — Read `internal/hooks/hooks.go`. Remove the listener registration line:
  ```go
  events.RegisterListener(events.NewRound{}, PruneVMs)
  ```
  This is ~line 18. No scripting import to remove (hooks.go doesn't import scripting directly — the PruneVMs function was in the deleted file).

- [ ] **Step 3.3: hooks_test.go** — Read `internal/hooks/hooks_test.go`. Delete the entire PruneVMs test section (~lines 1110-1128):
  ```go
  // --- PruneVMs ---
  func TestPruneVMs_NonDivisibleRound(...)
  func TestPruneVMs_DivisibleRound(...)
  func TestPruneVMs_ZeroRound(...)
  ```

- [ ] **Step 3.4: RoomChange_CleanupEphemeralRooms.go** — Read `internal/hooks/RoomChange_CleanupEphemeralRooms.go`. Remove ONLY the `scripting.PruneRoomVMs(removedRoomIds...)` line (~line 26) and the `"github.com/GoMudEngine/GoMud/internal/scripting"` import. Keep the rest of the function (the ephemeral room cleanup logic). The if-block checking `len(removedRoomIds) > 0` can be removed too since it now has no body, but the `rooms.TryEphemeralCleanup` call above it must stay. Result:
  ```go
  func CleanupEphemeralRooms(e events.Event) events.ListenerReturn {
      evt := e.(events.RoomChange)
      if evt.UserId == 0 {
          return events.Continue
      }
      if rooms.IsEphemeralRoomId(evt.FromRoomId) {
          rooms.TryEphemeralCleanup(evt.FromRoomId)
      }
      return events.Continue
  }
  ```

- [ ] **Step 3.5: spell_resolution.go** — Read `internal/hooks/spell_resolution.go`. Find `scripting.TrySpellScriptEvent("onMagic", ...)` (~line 204). Delete the if-block. Remove scripting import if unused.

- [ ] **Step 3.6: NewRound_UserRoundTick.go** — Read `internal/hooks/NewRound_UserRoundTick.go`. Find TWO scripting calls: `scripting.TryRoomIdleEvent` (~line 45) and `scripting.TryBuffScriptEvent` (~line 205). Delete both if-blocks. Remove scripting import.

- [ ] **Step 3.7: NewRound_MobRoundTick.go** — Read `internal/hooks/NewRound_MobRoundTick.go`. Find `scripting.TryBuffScriptEvent` (~line 157). Delete the if-block. Remove scripting import.

- [ ] **Step 3.8: NewRound_IdleMobs.go** — Read `internal/hooks/NewRound_IdleMobs.go`. Find TWO `scripting.TryMobScriptEvent("onPath")` calls (~lines 88, 131). Delete both if-blocks. Remove scripting import.

- [ ] **Step 3.9: NewRound_DoCombat_helpers.go** — Read `internal/hooks/NewRound_DoCombat_helpers.go`. Find SIX scripting calls at approximate lines:
  - ~line 275: `scripting.TrySpellScriptEvent`
  - ~line 379: `scripting.TrySpellScriptEvent`
  - ~line 568: `scripting.TryRoomScriptEvent`
  - ~line 580: `scripting.TryRoomScriptEvent`
  - ~line 1332: `scripting.TryMobScriptEvent`
  - ~line 1782: `scripting.TryMobScriptEvent`
  Delete all six if-blocks. Remove scripting import. This is a large file — search carefully for ALL occurrences.

- [ ] **Step 3.10: NewTurn_PruneBuffs.go** — Read `internal/hooks/NewTurn_PruneBuffs.go`. Find TWO `scripting.TryBuffScriptEvent("onEnd")` calls (~lines 61, 106). Delete both if-blocks. Remove scripting import.

- [ ] **Step 3.11: Buff_ApplyBuffs.go** — Read `internal/hooks/Buff_ApplyBuffs.go`. Find TWO `scripting.TryBuffScriptEvent` calls (~lines 115, 123). Delete both if-blocks. Remove scripting import.

- [ ] **Step 3.12: MobIdle_HandleIdleMobs.go** — Read `internal/hooks/MobIdle_HandleIdleMobs.go`. Find `scripting.TryMobScriptEvent("onIdle")` (~line 159). Delete the if-block. Remove scripting import.

- [ ] **Step 3.13: ItemOwnership_CheckItemQuests.go** — Read `internal/hooks/ItemOwnership_CheckItemQuests.go`. Find TWO `scripting.TryItemScriptEvent` calls (~lines 38, 52). Delete both if-blocks. Remove scripting import.

- [ ] **Step 3.14: PlayerDrop_HandlePlayerDrop.go** — Read `internal/hooks/PlayerDrop_HandlePlayerDrop.go`. Find `scripting.TryPlayerDownedEvent` (~line 61). Delete the if-block. Remove scripting import.

- [ ] **Step 3.15: PlayerSpawn_HandleJoin.go** — Read `internal/hooks/PlayerSpawn_HandleJoin.go`. Find `scripting.TryRoomScriptEvent` (~line 187). Delete the if-block. Remove scripting import.

- [ ] **Step 3.16: Build verification.** Run:
  ```bash
  cd "C:/Users/Calabe Davis/workspace/DOGMud" && go build ./...
  ```

- [ ] **Step 3.17: Commit.**
  ```bash
  git add internal/hooks/
  git commit -m "$(cat <<'EOF'
  refactor: remove scripting calls from hooks

  Remove all scripting bridge calls from 11 hook files. Delete
  NewRound_PruneVMs.go entirely (only purpose was scripting.PruneVMs).
  Remove PruneVMs listener registration and tests.
  CleanupEphemeralRooms retains room cleanup but drops VM pruning.

  Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
  EOF
  )"
  ```

---

## Task 4: Remove scripting from plugins + main.go + delete scripting package

- [ ] **Step 4.1: plugins.go** — Read `internal/plugins/plugins.go`. Make three changes:
  1. Delete the `scriptCommands` pre-population in the `New()` function (~lines 91-98). The `version` lambda and comment about forcing module registration are no longer needed.
  2. Delete the `AddScriptingFunction()` method (~lines 252-259).
  3. Delete the `for nameSpace, funcMap := range p.Callbacks.scriptCommands` loop (~lines 454-458) that calls `scripting.AddModlueFunction`.
  4. Remove `"github.com/GoMudEngine/GoMud/internal/scripting"` from imports.

- [ ] **Step 4.2: plugincallbacks.go** — Read `internal/plugins/plugincallbacks.go`. Remove the `scriptCommands` field from the `PluginCallbacks` struct (~line 13) and its initialization in `newPluginCallbacks()` (~line 25).

- [ ] **Step 4.3: main.go** — Read `main.go`. Remove two scripting calls:
  1. `scripting.Setup(int(c.Scripting.LoadTimeoutMs), int(c.Scripting.RoomTimeoutMs))` (~line 251)
  2. `scripting.PruneVMs(true)` (~line 1045)
  3. Remove `"github.com/GoMudEngine/GoMud/internal/scripting"` from imports (~line 58).

- [ ] **Step 4.4: Build verification** (before deleting the package, to confirm no remaining references):
  ```bash
  cd "C:/Users/Calabe Davis/workspace/DOGMud" && grep -r '"github.com/GoMudEngine/GoMud/internal/scripting"' --include="*.go" .
  ```
  This should return zero results. If any remain, fix them before proceeding.

- [ ] **Step 4.5: Delete the scripting package.** Remove the entire directory:
  ```bash
  cd "C:/Users/Calabe Davis/workspace/DOGMud" && git rm -r internal/scripting/
  ```

- [ ] **Step 4.6: Remove goja from go.mod and tidy dependencies.**
  ```bash
  cd "C:/Users/Calabe Davis/workspace/DOGMud" && go mod tidy
  ```
  Verify that `github.com/dop251/goja`, `github.com/dlclark/regexp2`, `github.com/go-sourcemap/sourcemap`, and `github.com/google/pprof` are no longer in go.mod (pprof may survive if something else uses it — check).

- [ ] **Step 4.7: Build verification.**
  ```bash
  cd "C:/Users/Calabe Davis/workspace/DOGMud" && go build ./...
  ```

- [ ] **Step 4.8: Run tests.**
  ```bash
  cd "C:/Users/Calabe Davis/workspace/DOGMud" && go test ./...
  ```
  Fix any failures. The PruneVMs tests were already deleted in Task 3.

- [ ] **Step 4.9: Commit.**
  ```bash
  git add internal/plugins/plugins.go internal/plugins/plugincallbacks.go \
    main.go go.mod go.sum
  git commit -m "$(cat <<'EOF'
  refactor: delete scripting package and goja dependency

  Remove scripting bridge from plugins and main.go. Delete entire
  internal/scripting/ directory (18 files). Remove goja and its
  transitive dependencies from go.mod.

  Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
  EOF
  )"
  ```

---

## Task 5: Delete JS files + admin creation commands

- [ ] **Step 5.1: Delete all JS files from default world.**
  ```bash
  cd "C:/Users/Calabe Davis/workspace/DOGMud" && find _datafiles/world/default -name "*.js" -exec git rm {} +
  ```

- [ ] **Step 5.2: Delete all JS files from empty world.**
  ```bash
  cd "C:/Users/Calabe Davis/workspace/DOGMud" && find _datafiles/world/empty -name "*.js" -exec git rm {} +
  ```

- [ ] **Step 5.3: Delete all JS files from sample-scripts.**
  ```bash
  cd "C:/Users/Calabe Davis/workspace/DOGMud" && find _datafiles/sample-scripts -name "*.js" -exec git rm {} +
  ```

- [ ] **Step 5.4: Delete all .js.bak files from dogmud world.**
  ```bash
  cd "C:/Users/Calabe Davis/workspace/DOGMud" && find _datafiles/world/dogmud -name "*.js.bak" -exec git rm {} +
  ```

- [ ] **Step 5.5: Verify no JS files remain in game data.** Check that the only remaining `.js` files are web assets (if any):
  ```bash
  cd "C:/Users/Calabe Davis/workspace/DOGMud" && find _datafiles -name "*.js" -o -name "*.js.bak" | head -20
  ```
  Any results should be web-related only (webclient JS). If game-logic JS files remain, delete them.

- [ ] **Step 5.6: Delete newmobfile.go.**
  ```bash
  cd "C:/Users/Calabe Davis/workspace/DOGMud" && git rm internal/mobs/newmobfile.go
  ```

- [ ] **Step 5.7: Delete newspellfile.go.**
  ```bash
  cd "C:/Users/Calabe Davis/workspace/DOGMud" && git rm internal/spells/newspellfile.go
  ```

- [ ] **Step 5.8: admin.mob.go** — Read `internal/usercommands/admin.mob.go`. Remove the `mob create` route from the `Mob()` dispatcher (~lines 38-47):
  ```go
  // Create a new mob
  if args[0] == `create` {
      if !user.HasRolePermission(`mob.create`) {
          user.SendText(`you do not have <ansi fg="command">mob.create</ansi> permission`)
          return true, nil
      }
      return mob_Create(strings.TrimSpace(rest[6:]), user, room, flags)
  }
  ```
  Delete the entire `mob_Create()` function (~line 163 to its end). Remove the `mob.create` permission comment from the header. Remove any imports that become unused (check `configs`, `species`, `mudlog`, `strconv` — they may only be used by `mob_Create`).

- [ ] **Step 5.9: admin.spell.go** — Read `internal/usercommands/admin.spell.go`. Remove the `spell create` route from the `Spell()` dispatcher (~lines 36-44):
  ```go
  if args[0] == `create` {
      if !user.HasRolePermission(`spell.create`) {
          user.SendText(`you do not have <ansi fg="command">spell.create</ansi> permission`)
          return true, nil
      }
      return spell_Create(strings.TrimSpace(rest[6:]), user, room, flags)
  }
  ```
  Delete the entire `spell_Create()` function (~line 92 to its end). Remove the `spell.create` permission comment. Remove unused imports.

- [ ] **Step 5.10: Update admin help templates.** Read and edit:
  - `_datafiles/world/dogmud/templates/admincommands/help/command.mob.template` — remove the `create` subcommand documentation.
  - `_datafiles/world/dogmud/templates/admincommands/help/command.spell.template` — remove the `create` subcommand documentation.

- [ ] **Step 5.11: Build verification.**
  ```bash
  cd "C:/Users/Calabe Davis/workspace/DOGMud" && go build ./...
  ```

- [ ] **Step 5.12: Run tests.**
  ```bash
  cd "C:/Users/Calabe Davis/workspace/DOGMud" && go test ./...
  ```

- [ ] **Step 5.13: Commit.**
  ```bash
  git add -A
  git commit -m "$(cat <<'EOF'
  chore: delete JS files and admin creation commands

  Remove 88 default JS files, 19 empty JS files, 8 sample scripts,
  and 31 dogmud .js.bak backups. Delete newmobfile.go and
  newspellfile.go. Remove mob create and spell create admin
  subcommands and their help documentation.

  Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
  EOF
  )"
  ```

---

## Task 6: Update documentation

- [ ] **Step 6.1: internal/mobs/context.md** — Read the file. Remove or rewrite all references to:
  - Scripting integration, script path resolution, ScriptTag, GetScriptPath
  - `.js` file generation and script template copying
  - Runtime data storage for scripts
  Replace with references to behavior trees where appropriate (mobs now use behavior trees in `_datafiles/world/dogmud/behaviors/mobs/`).

- [ ] **Step 6.2: internal/spells/context.md** — Read the file. Remove references to:
  - Automatic script template generation by spell type
  - Custom script path resolution and loading
  - `none` type description mentioning "JS script handles everything"
  - Component-consuming summon spells using JS scripting
  Replace with references to Go hooks in `internal/hooks/spell_*.go`.

- [ ] **Step 6.3: internal/items/context.md** — Read the file. Remove references to:
  - Scripting integration, blob storage for scripting
  - `.js` file path generation from `.yaml`

- [ ] **Step 6.4: internal/buffs/context.md** — Read the file. Remove references to:
  - JavaScript scripting support for custom buff behaviors
  - Script path resolution and loading
  - `.js` file generation from buff YAML paths
  Replace with references to config-driven buff ticks and Go hooks.

- [ ] **Step 6.5: internal/hooks/context.md** — Read the file. Remove references to:
  - `scripting.TryRoomScriptEvent()` integration
  - `internal/scripting` package dependency
  - Any hook descriptions that mention firing JS scripts

- [ ] **Step 6.6: CLAUDE.md** — Read `CLAUDE.md`. Make these changes:
  1. **"Data File Naming Convention" section**: Remove the bullet about confirming `.js` stub filename matches `.yaml` filename (~line 161).
  2. **"Quest Item Delivery — give.go Gotcha" section** (~lines 206-211): Replace `onGive script` references. The section should read:
     ```
     - Every NPC that should accept a quest item needs either a quest engine
       item_give trigger or a behavior tree player_give handler (otherwise
       the mob does the default "considers the item" emote and the quest
       doesn't advance)
     - NPCs that should NOT keep the item need a behavior tree player_give
       handler that returns the item via the quest engine or dialogue system
     ```
  3. Remove any other stray JS/scripting references throughout the file.

- [ ] **Step 6.7: .claude/commands/new-quest.md** — Read the file. Update step 4e (~line 85):
  Replace:
  ```
  **4e. Mob scripts** — create `.js` files for `onGive` handlers.
  ```
  With:
  ```
  **4e. Mob behavior trees** — create behavior tree YAML for `player_give` handlers.
  ```
  Also update the surrounding text that references `onGive` scripts to use behavior tree terminology. Update step 4d (~line 81) to replace `.js` room script references with room behavior tree references.
  Remove lines referencing item delivery needing an `onGive` script and replace with behavior tree or quest engine patterns.

- [ ] **Step 6.8: .claude/commands/sketch-quest.md** — Read the file. Update:
  - Line ~74: Replace `mob script (onGive)` with `behavior tree (player_give)`
  - Line ~74: Replace `room script (onCommand)` with `room behavior tree (room_command)`
  - Lines ~88-89: Replace `room script onCommand` references with `room behavior tree`
  - Line ~132: Replace `mob script onGive handler` with `behavior tree player_give handler`
  - Lines ~171-172: Replace the CREATE file table entries for `.js` files with behavior tree YAML entries
  - Lines ~205-210: Replace `onGive script` references with behavior tree references
  - Line ~219-220: Replace room script references with room behavior tree references

- [ ] **Step 6.9: Commit.**
  ```bash
  git add internal/mobs/context.md internal/spells/context.md \
    internal/items/context.md internal/buffs/context.md \
    internal/hooks/context.md CLAUDE.md \
    .claude/commands/new-quest.md .claude/commands/sketch-quest.md
  git commit -m "$(cat <<'EOF'
  docs: remove JS scripting references from all documentation

  Update context.md files for mobs, spells, items, buffs, and hooks
  to remove scripting integration references. Update CLAUDE.md to
  replace onGive script references with behavior tree patterns.
  Update new-quest and sketch-quest slash commands to use behavior
  tree terminology instead of JS scripts.

  Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
  EOF
  )"
  ```

---

## Verification Checklist

After all tasks complete:

- [ ] `go build ./...` passes with zero errors
- [ ] `go test ./...` passes with zero failures
- [ ] `grep -r "scripting" --include="*.go" internal/ main.go` returns zero results (excluding comments/strings that aren't import paths)
- [ ] `grep -r '"github.com/GoMudEngine/GoMud/internal/scripting"' --include="*.go" .` returns zero results
- [ ] `grep -r "goja" go.mod` returns zero results
- [ ] `find _datafiles -name "*.js" | grep -v webclient` returns zero results (no game-logic JS files remain)
- [ ] `find _datafiles -name "*.js.bak"` returns zero results
- [ ] `ls internal/scripting/` returns "No such file or directory"
- [ ] `ls internal/mobs/newmobfile.go` returns "No such file or directory"
- [ ] `ls internal/spells/newspellfile.go` returns "No such file or directory"
- [ ] `ls internal/hooks/NewRound_PruneVMs.go` returns "No such file or directory"
