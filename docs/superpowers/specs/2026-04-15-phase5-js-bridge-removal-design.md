# Phase 5: JS Scripting Bridge Removal — Design Spec

## Goal

Remove the entire JavaScript scripting system from DOGMud — the Go/JS
bridge, the goja dependency, all remaining JS files, and admin
creation commands. After this phase, the codebase has zero JS
scripting infrastructure.

## Scope

**In scope:**
- Remove all `scripting.*` calls from 30 Go caller files
- Delete `internal/scripting/` package (17 files)
- Delete hooks that only exist for JS (PruneVMs, etc.)
- Delete all JS files in `default/`, `empty/`, `sample-scripts/`
- Delete all `.js.bak` files in `dogmud/`
- Delete admin mob/spell creation commands + help files
- Remove `github.com/dop251/goja` from go.mod
- Update context.md files, CLAUDE.md, and slash commands

**Out of scope:**
- Replacing admin builder commands (future work)
- Modifying behavior tree engine
- Any gameplay changes

## Architecture

### Stream 1: Remove Scripting Callers (30 Go files)

Every `scripting.Try*` call in the codebase is dead code — no dogmud
JS files exist to load. Remove each call and its surrounding
error-handling block. Where a behavior tree dispatch precedes the
scripting call, keep the behavior tree dispatch.

**48 dead event calls across these files:**

User commands:
- `ask.go` — remove TryMobScriptEvent("onAsk")
- `give.go` — remove TryMobScriptEvent("onGive")
- `talk.go` — remove TryMobScriptEvent("onAsk")
- `show.go` — remove TryMobScriptEvent("onShow")
- `go.go` — remove TryRoomScriptEvent("onExit"/"onEnter")
- `start.go` — remove TryRoomScriptEvent("onEnter")
- `admin.teleport.go` — remove TryRoomScriptEvent
- `skill.cast.go` — remove TrySpellScriptEvent("onCast")
- `use.go` — remove TryItemScriptEvent("onUse")
- `equip.go` — remove TryBuffScriptEvent("onStart")
- `buy.go` — remove TryItemScriptEvent("onPurchase")
- `usercommands.go` — remove TryRoomCommand, TryItemCommand,
  TryRoomScripts helper function

Mob commands:
- `cast.go` — remove TryMobScriptEvent/TrySpellScriptEvent
- `aid.go` — remove TryMobScriptEvent
- `go.go` — remove TryMobScriptEvent
- `suicide.go` — remove TryMobScriptEvent("onDie")

Hooks:
- `spell_resolution.go` — remove TrySpellScriptEvent("onMagic")
- `NewRound_UserRoundTick.go` — remove TryRoomIdleEvent,
  TryBuffScriptEvent
- `NewRound_MobRoundTick.go` — remove TryBuffScriptEvent
- `NewRound_IdleMobs.go` — remove TryMobScriptEvent("onPath")
- `NewRound_DoCombat_helpers.go` — remove TryMobScriptEvent,
  TrySpellScriptEvent, TryRoomScriptEvent
- `NewTurn_PruneBuffs.go` — remove TryBuffScriptEvent("onEnd")
- `Buff_ApplyBuffs.go` — remove TryBuffScriptEvent
- `MobIdle_HandleIdleMobs.go` — remove TryMobScriptEvent("onIdle")
- `ItemOwnership_CheckItemQuests.go` — remove TryItemScriptEvent
- `PlayerDrop_HandlePlayerDrop.go` — remove TryPlayerDownedEvent
- `PlayerSpawn_HandleJoin.go` — remove TryRoomScriptEvent
- `RoomChange_CleanupEphemeralRooms.go` — remove PruneRoomVMs

Also remove `scripting` from import blocks in all affected files.

### Stream 2: Delete Scripting Package + Dependency

Delete `internal/scripting/` entirely (17 files including context.md).

Delete hooks that only exist for scripting:
- `internal/hooks/NewRound_PruneVMs.go` — only calls PruneVMs()

Check if any other hooks become empty after stream 1 removals and
delete those too.

Remove from `go.mod`:
- `github.com/dop251/goja`
- Any transitive dependencies only used by goja

Run `go mod tidy` to clean up.

### Stream 3: Delete Files + Admin Commands

**JS file deletion:**
- `_datafiles/world/default/buffs/*.js` (38 files)
- `_datafiles/world/default/mobs/*/scripts/*.js` (18 files)
- `_datafiles/world/default/rooms/*/*.js` (26 files)
- `_datafiles/world/default/spells/*.js` (6 files)
- `_datafiles/world/empty/buffs/*.js` (3 files)
- `_datafiles/world/empty/mobs/*/scripts/*.js` (1 file)
- `_datafiles/world/empty/rooms/*/*.js` (8 files)
- `_datafiles/world/empty/spells/*.js` (7 files)
- `_datafiles/sample-scripts/**/*.js` (8 files)
- `_datafiles/world/dogmud/**/*.js.bak` (all backup files)

Keep folder structure in `default/` intact so `ValidateWorldFiles()`
still works (it only checks directory existence).

**Admin command deletion:**
- `internal/mobs/newmobfile.go` — delete
- `internal/spells/newspellfile.go` — delete
- Any associated admin command registration
- Help templates for admin creation commands (if they exist)

### Stream 4: Documentation Updates

**Delete:**
- `internal/scripting/context.md` (deleted with package)

**Update context.md files — remove scripting sections:**
- `internal/mobs/context.md` — remove "Scripting Integration"
- `internal/spells/context.md` — remove "Scripting Integration"
- `internal/items/context.md` — remove "Scripting Support"
- `internal/buffs/context.md` — remove "Scripting and Customization"
- `internal/hooks/context.md` — remove PruneVMs references

**Update project instructions:**
- `CLAUDE.md` — update quest item delivery section to reference
  behavior trees instead of `onGive` scripts. Remove any references
  to JS scripting. Update the "Quest Item Delivery" section to note
  that behavior trees handle item rejection and the quest engine
  handles quest advancement via `item_give` triggers.
- `.claude/commands/new-quest.md` — replace `onGive` script
  references with behavior tree pattern for item-give handling
- `.claude/commands/sketch-quest.md` — replace room/mob script
  references with behavior tree references

## Execution Order

1. Stream 1: Remove scripting callers (must compile after)
2. Stream 2: Delete package + dependency (must compile after)
3. Stream 3: Delete JS files + admin commands (must compile after)
4. Stream 4: Documentation updates (no compilation needed)

## Risk Assessment

**Low risk.** Every scripting call is confirmed dead code — no dogmud
JS files exist. The behavior tree system and quest engine handle all
game logic previously handled by JS. The only compilation risk is
missing an import removal or leaving a reference to a deleted
function.

## Validation

After all streams complete:
- `go build ./...` — clean build
- `go test ./...` — all tests pass
- `find _datafiles -name "*.js" -not -path "*/html/*"` — zero
  non-web JS files
- `grep -r "scripting" internal/ --include="*.go"` — zero references
- Server starts and runs without errors
