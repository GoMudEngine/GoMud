# Code Cleanup 1.1: Behavior Tree Package Split — Design Spec

## Goal

Split the two monolithic files in `internal/behaviortree/`
(`actions.go` at 1005 lines and `conditions.go` at 408 lines) into
themed files. Extract duplicated parameter helpers into a shared
file. Reduce the per-file cognitive load so future behavior tree
work can focus on one theme at a time.

Pure structural change — zero behavior impact.

## Scope

**In scope:**
- Split `actions.go` into 6 themed files + thin registry
- Split `conditions.go` into 4 themed files + thin registry
- Extract `getIntParam`/`getStringParam` into shared `params.go`
- Move `splitTwo`/`parseIntStr` helpers alongside their caller

**Out of scope:**
- Any behavior change
- New actions, conditions, or types
- Changes to `engine.go`, `helpers.go`, `loader.go`, `state.go`,
  `decorators.go`, `structural.go`, `room_state.go`, `types.go`
- Renaming any action/condition/function

## Architecture

### Shared Helpers

**New file: `params.go`**

Contains parameter lookup helpers used by both actions and conditions
(currently duplicated in both files). Also used by `loader.go` line
140 for decorator compilation.

- `getIntParam(params map[string]any, key string) int`
- `getStringParam(params map[string]any, key string) string`

### Action Files

**Registry file: `actions.go` (thin)**

Retains only the coordination layer:

- `ActionFunc` type alias
- `actionRegistry` package-level var
- `delayedActions` package-level map
- `ActionNode` struct
- `ActionNode.Evaluate()` method
- `LookupAction()` function
- `init()` function — registers all 37 action names

**Themed action files:**

`actions_combat.go`
- `actAttack`, `actFlee`, `actCast`, `actAddBuff`, `actRemoveBuff`

`actions_dialogue.go`
- `actRespond`, `actSay`, `actEmote`
- `actSendUserText`, `actSendRoomText`
- `actMobSay`, `actMobEmote`

`actions_room.go`
- `actSetRoomLocked`, `actSpawnItemInRoom`, `actAddTempExit`
- `actMovePlayer`, `actIntercept`

`actions_quest.go`
- `actGrantQuest` (also used via `grant_quest_to_user` alias)
- `actSetQuestFlag`, `actGrantMutation`
- `actGiveGold`, `actTakeGold`
- `actGiveItem`, `actReturnItem`, `actTakeItem`
- `actGiveItemMultiple`, `actSetMiscData`

`actions_mob.go`
- `actSpawnMob`, `actSummonCompanion`
- `actCommand`, `actCommandMob`, `actMove`
- `actOpenInstancePortal`, `actCreateInstance`
- `splitTwo`, `parseIntStr` (helpers only used by `actOpenInstancePortal`)

`actions_state.go`
- `actSetState`, `actIncrementState`, `actDecrementState`

### Condition Files

**Registry file: `conditions.go` (thin)**

Retains only:

- `ConditionFunc` type alias
- `conditionRegistry` package-level var
- `ConditionNode` struct
- `ConditionNode.Evaluate()` method
- `LookupCondition()` function
- `init()` function — registers all 23 condition names

**Themed condition files:**

`conditions_mob.go`
- `condMobInCombat`, `condMobHealthBelow`, `condMobAtHome`
- `condMobHasBuff`, `condMobInRoom`

`conditions_player.go`
- `condPlayerHasQuest`, `condPlayerMissingQuest`
- `condPlayerHasItem`, `condPlayerHasGold`, `condPlayerHasFlag`
- `condPlayerHasSpell`, `condPlayerHasMiscData`
- `condPlayersInRoom`, `condMultipleEnemies`

`conditions_room.go`
- `condCommandMatches`, `condCommandRestContains`

`conditions_state.go`
- `condStateEquals`, `condStateGreaterThan`
- `condKeywordMatch`, `condItemMatches`
- `condTimeOfDay`, `condRoundMod`, `condRandomChance`

### File Size Targets

| File | Estimated lines | Current source |
|------|----------------|----------------|
| `actions.go` | ~120 | reduced from 1005 |
| `actions_combat.go` | ~100 | extracted |
| `actions_dialogue.go` | ~180 | extracted |
| `actions_room.go` | ~120 | extracted |
| `actions_quest.go` | ~220 | extracted |
| `actions_mob.go` | ~260 | extracted + helpers |
| `actions_state.go` | ~60 | extracted |
| `conditions.go` | ~80 | reduced from 408 |
| `conditions_mob.go` | ~80 | extracted |
| `conditions_player.go` | ~180 | extracted |
| `conditions_room.go` | ~40 | extracted |
| `conditions_state.go` | ~130 | extracted |
| `params.go` | ~40 | new shared file |

## Constraints

- **Zero behavior change.** No logic modifications, no renames, no
  new types.
- All exported names (`LookupAction`, `LookupCondition`, `ActionFunc`,
  `ConditionFunc`, `ActionNode`, `ConditionNode`) unchanged.
- All YAML action/condition names unchanged.
- All function signatures unchanged.
- `init()` functions registering everything stay in `actions.go`
  and `conditions.go` so there's one place to see all registered
  names.
- The `delayedActions` map stays in `actions.go` since it's closely
  tied to `ActionNode.Evaluate`.

## Testing

After each file is extracted:
- `go build ./internal/behaviortree/...` — clean
- `go vet ./internal/behaviortree/...` — no new warnings
- `go build ./...` — full project build clean
- `go test ./internal/behaviortree/...` — all existing tests pass

After the full split:
- Start the server locally, enter a room with a mob that has a
  behavior tree (e.g., Dal in Thornwall, Edrin on Marches Spur)
- Verify behavior tree still fires correctly (mob emotes, responds
  to ask/give, combat AI works)
- Restart and verify trees load without errors

## Risk Assessment

**Low risk.** This is a pure file-reorganization refactor. Go's
package system means all files in the same package share the same
namespace — splitting a file is equivalent to keeping everything in
one file from the compiler's perspective.

The only way to break something: accidentally delete or misplace code
during the split. Mitigated by:
- Tests running after each extraction
- Manual smoke test at the end
- Git diff review before commit

## Execution Order

1. Create `params.go`, remove duplicates from `actions.go` and
   `conditions.go`. Verify build + tests.
2. Extract `actions_state.go` (smallest, simplest). Verify.
3. Extract `actions_room.go`. Verify.
4. Extract `actions_combat.go`. Verify.
5. Extract `actions_dialogue.go`. Verify.
6. Extract `actions_quest.go`. Verify.
7. Extract `actions_mob.go` (includes helper move). Verify.
8. Extract `conditions_room.go`. Verify.
9. Extract `conditions_state.go`. Verify.
10. Extract `conditions_mob.go`. Verify.
11. Extract `conditions_player.go`. Verify.
12. Final build, tests, manual smoke test, commit.
