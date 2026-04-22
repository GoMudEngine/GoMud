# Code Cleanup 1.6: Test Coverage for New Systems — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add 23 Go unit tests across four files (3 new, 1 appended) to lock in the behavior of five recently-shipped subsystems with zero dedicated coverage today: the room-level behavior tree engine entry point, Phase 4c condition functions, Phase 4c action functions, the quest-engine ↔ btree `player_give` handoff in `give.go`, and the hostile branch of `actSummonCompanion`. This is an additive substage — no production code changes unless a test surfaces a clear bug per scope-creep policy.

**Architecture:** 7 commits on `feature/stage-1.6-test-coverage`. Commit 1 establishes shared test fixtures (helpers wrapping the existing `Seed*ForTest` primitives). Commits 2–5 add behavior-tree tests in dependency order (engine → conditions → actions → companion summon). Commit 6 adds the cross-package `give.go` regression test. Commit 7 is the docs flip. Each commit independently revertable; commits 2/4/5 depend on commit 1 (the helpers).

**Tech Stack:** Go 1.25. No new dependencies. Verification via `go build`/`go vet`/`go test ./...` after every commit, plus a `-race` pass on `internal/behaviortree/...` after the final commit (the room-state double-checked-locking test needs `-race` to be meaningful).

**Spec:** `docs/superpowers/specs/2026-04-17-code-cleanup-1.6-test-coverage-design.md`

**Branch:** `feature/stage-1.6-test-coverage` off `development`.

---

## Task 0: Create feature branch

**Files:** none.

- [ ] **Step 1: Verify you're on `development` and clean**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git status
git branch --show-current
```

Expected: `development`. Working tree should be clean EXCEPT for the following known unrelated working-tree noise:

- `.claude/settings.local.json` — dirty
- `internal/usercommands/_datafiles/feedback/bugs.txt`, `suggestions.txt` — dirty
- `"Screenshot 2026-04-17 084513.png"` — untracked

These are **out of scope** for 1.6. Do NOT stage or commit them at any point in this plan. If `git status` shows anything else dirty, investigate before proceeding.

- [ ] **Step 2: Create feature branch**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git checkout -b feature/stage-1.6-test-coverage
```

Expected: `Switched to a new branch 'feature/stage-1.6-test-coverage'`.

- [ ] **Step 3: Baseline verification**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./...
```

Expected: clean build, clean vet, and all tests pass EXCEPT for one known pre-existing failure:

- `TestRoom_AddTemporaryExit/duplicate_name_rejected` in `internal/rooms/`

**This failure is known, pre-existing, and out of scope for 1.6.** It is documented in the memory file `project_rooms_package_audit_needed.md` and belongs to a future dedicated rooms-package pass. Do NOT attempt to fix it on this branch. Confirm before proceeding that it is the ONLY failing test. If any other test fails, STOP and investigate — the baseline is broken and must be green (aside from `TestRoom_AddTemporaryExit`) before you start adding tests.

Per-commit verification below treats that single failure as expected noise. If any commit causes additional test failures, STOP and revert.

---

## Scope Policy: Additive Tests Only

This is a **test-only substage**. The default disposition is "do not change production code." Test additions, fixtures, and helpers are all that should appear in the diff aside from the docs flip in Task 7.

**The exception:** if writing a test surfaces a clear, reproducible bug in the system under test, the scope-creep policy from 1.2b/1.5 applies:

- **Clear bug** (test demonstrates a panic, silent state loss, or contract violation reachable in normal gameplay) → preceding `fix:` commit on the same branch, before the `test:` commit that demonstrates the fix. Memory file note in `project_error_handling_audit_findings.md` (carry-over from 1.5) under the `## Resolved` or `## Open / In-Progress` section, flagged as "surfaced during 1.6 test author pass."
- **Ambiguous case** (test result is unexpected but unclear whether the production code or the test is wrong) → pause and ask the user. Log to `project_error_handling_audit_findings.md` under `## Pending Decision` with the test name and observation.
- **Dead code spotted incidentally** (e.g., unreferenced helper while reading a target file) → `chore:` removal commit, separate from the test commit. Same memory-file note convention.

**Do NOT spray defensive test coverage to "feel safe."** Each test must justify itself per the spec's D1 depth bar (happy path + 1–2 obvious failure modes). If a test would only verify a defensive guard you wrote yourself, it doesn't belong here.

**No PATCH_NOTES.md entry by default.** Audit findings and tests are internal; zero player-facing impact. Exception: if a `fix:` commit ships a player-visible bug fix, that one finding gets a PATCH_NOTES line at execution time.

**Pre-existing baseline noise to ignore at every task boundary:**
- `.claude/settings.local.json`
- `internal/usercommands/_datafiles/feedback/*.txt`
- `Screenshot 2026-04-17 084513.png`
- `TestRoom_AddTemporaryExit/duplicate_name_rejected` test failure (per `project_rooms_package_audit_needed.md`)

---

## Test Patterns

Six copy-paste-ready patterns the executor uses. Don't invent variations; the byte-consistent pattern keeps test diffs reviewable.

### Pattern A — `seedTestMob` / `seedTestUser` / `seedTestRoom` helpers

These three helpers live in `internal/behaviortree/test_helpers_test.go` (created in Task 1). Each wraps the existing `Seed*ForTest` primitive in the corresponding domain package and returns a cleanup function. The caller composes only the helpers it needs (counter-pattern: a single `seedAll()` mega-fixture per test — that's what `internal/hooks/hooks_test.go` does, and it's overkill for 1.6).

**Underlying primitives (already exist; do NOT duplicate):**

- `mobs.SeedMobsForTest(specs map[int]*Mob, instances map[int]*Mob) func()` — `internal/mobs/test_helpers.go:6`
- `users.SeedUsersForTest(testUsers map[int]*UserRecord) func()` — `internal/users/test_helpers.go:11`
- `users.NewTestUser(userId int, username string, charName string, connId uint64) *UserRecord` — `internal/users/test_helpers.go:38`
- `rooms.SeedRoomsForTest(roomMap map[int]*Room, zoneMap map[string]*ZoneConfig) func()` — `internal/rooms/test_helpers.go:6`
- `rooms.MarkRoomOccupancy(roomId int, playerCt int, mobCt int)` — `internal/rooms/test_helpers.go:27`

**Helper signatures (executor implements; sketch only):**

```go
// seedTestMob seeds a single mob spec at templateId and a single instance
// at instanceId placed in homeRoomId. Returns a cleanup function.
func seedTestMob(t *testing.T, templateId int, instanceId int, homeRoomId int, name string) func()

// seedTestUser seeds a single user (UserId, username, charName, RoomId).
// Returns a cleanup function.
func seedTestUser(t *testing.T, userId int, username string, charName string, roomId int) func()

// seedTestRoom seeds a single room with the supplied id and zone.
// Caller can call room.AddPlayer / room.AddMob after — the seed only
// installs the *Room into the registry. Returns a cleanup function.
func seedTestRoom(t *testing.T, roomId int, zone string) func()
```

**Caller pattern (rote):**

```go
func TestActFoo_HappyPath(t *testing.T) {
    cleanupRoom := seedTestRoom(t, 1, "TestZone")
    defer cleanupRoom()
    cleanupMob := seedTestMob(t, 1, 100, 1, "Skeleton")
    defer cleanupMob()
    cleanupUser := seedTestUser(t, 1, "alice", "Aliceia", 1)
    defer cleanupUser()
    // ... arrange / act / assert ...
}
```

**Deterministic IDs:** UserId 1, MobId 1 (template), MobInstanceId 100, RoomId 1 by default — matches `hooks_test.go`. Tests needing more pick obvious extensions (UserId 2, RoomId 2, etc.).

### Pattern B — Building an in-memory behavior tree from YAML bytes

The engine exposes `LoadTreeFromBytes([]byte) (Node, error)` at `internal/behaviortree/loader.go:32`. The room-engine tests need this to load a tree without a YAML file on disk. To wire the in-memory tree into the engine for `TryRoomBehavior`, the test must call `GetEngine().LoadRoomTree(roomId, path)` with a file path — but `LoadRoomTree` reads the file. The test must therefore write a temporary tree YAML to a tempdir and point `LoadRoomTree` at it (the engine's public surface is file-based).

**Snippet (sketch):**

```go
yamlData := []byte(`
tree:
  type: action
  do: intercept
`)
// For unit tests on tree evaluation only (no engine state):
node, err := behaviortree.LoadTreeFromBytes(yamlData)
require.NoError(t, err)
ctx := &behaviortree.EvalContext{ /* fields per Pattern E */ }
result := node.Evaluate(ctx)

// For tests that exercise TryRoomBehavior (requires engine cache):
tmpDir := t.TempDir()
treePath := filepath.Join(tmpDir, "1.yaml")
require.NoError(t, os.WriteFile(treePath, yamlData, 0644))
require.NoError(t, behaviortree.GetEngine().LoadRoomTree(1, treePath))
```

**Engine state caveat:** the global `behaviortree.GetEngine()` carries cache state across tests. Tests that load trees should also clear them at cleanup time. See Task 2's helper for the room-tree clear pattern (write directly into the engine map only via the engine's public surface — `LoadRoomTree` overwrites; for "clear" we need a `t.Cleanup` that loads an empty tree or accepts the carry-over). If carry-over breaks an isolated test, the test author may need an `EngineResetForTest` exported helper (out of scope unless required — note in memory file as a deferred follow-up).

### Pattern C — Asserting against a mob's queued commands via events listener

`Mob.Command` (`internal/mobs/mobs.go:637`) does NOT keep a per-mob queue. It pushes `events.Input{MobInstanceId, InputText, ReadyTurn}` onto the global event queue (`internal/events/events.go:90`). To verify "mob X had command Y queued," tests register an event listener that captures the `events.Input` events into a slice keyed by `MobInstanceId`, then assert on the captured slice after the action returns.

**Snippet (sketch):**

```go
var captured []events.Input
listenerId := events.RegisterListener(events.Input{}, func(e events.Event) events.ListenerReturn {
    if in, ok := e.(events.Input); ok {
        captured = append(captured, in)
    }
    return events.Continue
})
defer events.UnregisterListener(listenerId) // verify the unregister API exists
// ... call action under test ...
// Assertion:
found := false
for _, in := range captured {
    if in.MobInstanceId == 100 && strings.Contains(in.InputText, "say hello") {
        found = true
    }
}
require.True(t, found, "expected 'say hello' queued on mob 100")
```

**Verify at execution time:** does `events.UnregisterListener` (or equivalent) exist? If not, the test must scope the listener for the test's lifetime via a `t.Cleanup` that swaps an `enabled` flag in the closure to short-circuit further captures. Do NOT leak listeners between tests.

**Note:** `Mob.Command` calls `events.AddToQueue` directly — it does not actually invoke the listener until `events.ProcessEvents()` runs. The listener fires on `AddToQueue` only if `RegisterListener` taps the enqueue path (verify by reading `internal/events/events.go` listener mechanics; if listeners only fire during `ProcessEvents`, the test must call `events.ProcessEvents()` between act and assert).

### Pattern D — Asserting on user-visible text

`UserRecord.SendText` (`internal/users/userrecord.go:288`) pushes `events.Message{UserId, Text}` onto the queue. Same model as Pattern C.

**Two-tier fallback (decide per test):**

1. **Preferred:** capture `events.Message` events via Pattern C and assert on the `Text` field for the expected user.
2. **Fallback (if Pattern C is too brittle for a one-line test):** assert the action returned `Success`/`Failure` per the spec's D1 bar. The spec's risk register flags this explicitly: where `SendText` doesn't have a clean tap, falling back to "verify return value" is acceptable.

`Room.SendText` (`internal/rooms/rooms.go:255`) similarly fans out via the events queue. Same capture mechanism applies.

### Pattern E — Constructing a minimal `EvalContext`

From `internal/behaviortree/types.go:33`, the struct has:

```go
type EvalContext struct {
    Event       EventContext
    MobState    *BehaviorState
    MobId       int    // template ID
    InstanceId  int    // instance ID
    RoomId      int
    MobName     string
    Intercepted bool
}
```

And `EventContext` (line 13) carries `EventType`, `UserId`, `MobId`, `Text`, `ItemId`, `RoomId`, `Extra`, `Command`, `Rest`, `Direction`.

**Recipe per test type:**

- **Condition tests with no fixtures (e.g., `command_matches`, `command_rest_contains`):** populate only `Event.Command` / `Event.Rest`. Pass `&EvalContext{Event: EventContext{Command: "look"}}`.
- **Condition tests with room fixtures (`mob_in_room`):** populate `RoomId`, leave `MobState` nil unless needed.
- **Action tests on the triggering player (`give_gold`, `send_user_text`, `remove_buff`, `move_player`, `grant_mutation`):** populate `Event.UserId` and `RoomId`.
- **Action tests on the calling mob (`mob_say`, `mob_emote`, `summon_companion`):** populate `InstanceId`, `RoomId`, and `Event.UserId` (for hostile-summon target).
- **`intercept` action:** populate nothing meaningful — just create `&EvalContext{}` and assert `Intercepted == true` after.

**Direct-dispatch convention:** call the action function via the registry, not the wrapping `ActionNode.Evaluate`, to bypass the perception-scaled reaction-delay layer in `actions.go:88-138`. Example:

```go
fn := behaviortree.LookupAction("mob_say")
require.NotNil(t, fn, "mob_say not registered")
result := fn(params, ctx)
```

This matches the existing `conditions_test.go` style (line 8: `fn := LookupCondition("keyword_match")`).

### Pattern F — Quest-engine handoff for the give.go regression test

**Read first:** `internal/usercommands/give.go:175-237`. The mob-receive branch flow is:

1. Build a `questengine.GameBridge` for the user.
2. Call `questengine.GetEngine().Notify("item_give", ...)` — returns `NotifyResult{Handled, ConsumeItem}`.
3. If `qResult.Handled && qResult.ConsumeItem` → remove item from player only, `return true, nil` immediately. Never transfers; never calls btree.
4. Else → atomic transfer to mob, then `behaviortree.TryMobBehavior(m.InstanceId, ...)` with `EventType: "player_give"` and the post-transfer `giveItem.ItemId`.

**Extension point:** there is NO swap-in function variable for the quest engine. `questengine.GetEngine()` (`internal/questengine/loader.go:112`) returns the package-private `globalEngine *Engine`, lazily initialized. There is also NO public `SetEngine` / `EngineResetForTest`. However, existing `loader_test.go:98` directly assigns `globalEngine = nil` from within the package; the test cannot do that from `usercommands`.

**Therefore, the test does NOT mock the quest engine.** Instead, it seeds a real quest definition into the global engine via `questengine.GetEngine().RegisterQuest(...)` (`engine.go:25`) with a single trigger:

- Event: `item_give`
- Match `MobId` and `ItemId` to the test's seeded values
- Action with `ConsumeItem > 0` (per `engine.go:104`, this drives `result.ConsumeItem = true`)
- Action that grants something testable so `result.Handled = true` (any action that runs causes `Handled = true` per `engine.go:99`)

**This requires read access to `questengine.QuestDef` / `TriggerDef` / `ActionDef` types** at `internal/questengine/types.go`. The executor reads those types to construct a minimal valid in-memory quest. If the type wiring proves too involved within the 60-min wall-clock cap (see Risk Register), fall back to the smaller-scope test (see Task 6 decision tree).

**Inverse path (if cheap to add):** unregister the test quest, call Give again, verify the btree gets a `player_give` event. The btree side needs a per-mob behavior tree fixture — possibly skipped to keep the test focused. Decide at execution.

**Cleanup:** there is no public "remove a quest" API. The test must save the original `globalEngine` pointer via reflection or by directly writing a sibling test in the `questengine` package that exposes a `ResetForTest`. Easier path: in `give_test.go`, call `questengine.GetEngine()`, snapshot its `quests` map size or contents, do work, and restore by re-creating a fresh engine if the package exposes that constructor. Verify `questengine.NewEngine()` (`internal/questengine/engine.go:17`) exists and is exported — yes — but `globalEngine` is package-private, so `usercommands` can't reset it.

**Decision at execution time:** if the cleanup story is too brittle, add a single one-liner exported helper `questengine.ResetEngineForTest()` in a new `internal/questengine/test_helpers.go` and commit it as a `chore:` precursor. That's a minor scope expansion explicitly allowed by the scope-creep policy when needed for test wiring (and it parallels the pattern set by `mobs.SeedMobsForTest`, etc.).

---

## Task 1: Add shared behavior-tree test helpers

Create the test-only fixture helpers that all subsequent behaviortree tests use. Tiny commit; nothing else changes. Establishes the helper contract before any test depends on it.

**Files:**
- Create: `internal/behaviortree/test_helpers_test.go`

**Complexity:** Trivial (~30 LOC).

### Discovery

- [ ] **Step 1: Re-read the wrapped primitives** to confirm the signatures haven't drifted.

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
sed -n '1,50p' internal/mobs/test_helpers.go
sed -n '1,70p' internal/users/test_helpers.go
sed -n '1,50p' internal/rooms/test_helpers.go
```

Expected:
- `mobs.SeedMobsForTest(specs, instances) func()` — both maps int-keyed (template id and instance id).
- `users.SeedUsersForTest(map[int]*UserRecord) func()` and `users.NewTestUser(userId, username, charName, connId)`.
- `rooms.SeedRoomsForTest(roomMap, zoneMap) func()` plus `rooms.MarkRoomOccupancy(roomId, playerCt, mobCt)`.

### Implementation

- [ ] **Step 2: Write `internal/behaviortree/test_helpers_test.go`**

Write three helpers per Pattern A. Each helper:

1. Constructs the minimum spec/instance/UserRecord/Room needed.
2. Calls the wrapped `Seed*ForTest` primitive.
3. Returns the cleanup function.

Add a `TestMain(m *testing.M)` if needed for `mudlog.SetupLogger(nil, "", "", false)` (mirrors `internal/hooks/hooks_test.go:24`). If a `TestMain` already exists in another `*_test.go` in this package, do NOT add a second one (Go forbids it).

Required imports: `testing`, the three domain packages, `characters`, `buffs` (for the buff slot init in `NewTestUser`-style construction), `exit` (for room exits if needed).

### Verify

- [ ] **Step 3: Build + vet (no tests yet)**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./internal/behaviortree/...
go vet ./internal/behaviortree/...
go test ./internal/behaviortree/...
```

Expected: clean build/vet, existing `conditions_test.go` and `engine_test.go` still pass.

- [ ] **Step 4: Run full project test sweep** to confirm no cross-package regression.

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go test ./...
```

Expected: clean (treating `TestRoom_AddTemporaryExit/duplicate_name_rejected` as baseline noise).

### Commit

- [ ] **Step 5: Commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add internal/behaviortree/test_helpers_test.go
git commit -m "$(cat <<'EOF'
test(behaviortree): add test_helpers_test.go shared fixtures

Three helpers (seedTestMob, seedTestUser, seedTestRoom) wrap the
existing mobs/users/rooms SeedForTest primitives. Each returns a
cleanup function so tests compose only the fixtures they need
without dragging in the hooks-package seedAllRegistries() mega-fixture.

No tests yet — that's commits 2 onward. This commit is the helper
contract; the rest of stage 1.6 depends on it.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

Stage ONLY the helper file. Do NOT stage working-tree noise.

---

## Task 2: Room behavior tree engine entry-point coverage

Add 6 tests covering `TryRoomBehavior` (`internal/behaviortree/helpers.go:84-125`) and `EnsureRoomBTreeState` (`internal/behaviortree/room_state.go:12-27`). Highest-value single area: zero current coverage on the room-engine entry point that every Phase 4c room script depends on.

**Files:**
- Create: `internal/behaviortree/room_engine_test.go`

**Complexity:** Medium. 6 tests; needs Pattern B (in-memory tree) and a tempdir to point `LoadRoomTree` at.

### Discovery

- [ ] **Step 1: Re-read the entry points**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
sed -n '80,125p' internal/behaviortree/helpers.go
sed -n '1,30p' internal/behaviortree/room_state.go
sed -n '76,112p' internal/behaviortree/engine.go
```

Confirm:
- `TryRoomBehavior(roomId, event)` returns `false` on nil room (line 86-88), and writes `noRoomTree[roomId] = true` only via the `os.Stat` failure branch (line 97-99) — NOT on a parse error (line 101-104, which logs but does not negative-cache). The `TestTryRoomBehavior_LoadParseError_FailureNotCached` test asserts this contract.
- For `EventType == "room_command"` it returns `ctx.Intercepted`; otherwise `result == Success`.
- `EnsureRoomBTreeState` uses the double-checked lock at lines 13-23.

- [ ] **Step 2: Confirm room behavior path layout**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
sed -n '72,79p' internal/behaviortree/helpers.go
```

`GetRoomBehaviorPath` builds `{dataFiles}/behaviors/rooms/{zoneSafe}/{roomId}.yaml`. The tests must either set the test room's zone such that this path resolves to a tempdir-controlled location, OR write the yaml to whatever `dataFiles` resolves to in test config and clean up.

**Recommended:** in `TryRoomBehavior` happy-path tests, override the data-files path via `configs.SetTestDataFilesPath(...)` if such helper exists; otherwise call `LoadRoomTree(roomId, tempPath)` directly to pre-populate the engine cache, bypassing the os.Stat path. The negative-cache and parse-error tests still exercise the os.Stat / load branches, so they need a real path.

**Verify the configs helper at execution time.** If no such test override exists, the parse-error and missing-file tests can use `t.Setenv` plus a `configs` reload, OR they can directly seed `engine.noRoomTree[roomId]` by reading the unexported field via a same-package test. Same-package tests CAN read unexported fields — the easier route.

### Implementation

- [ ] **Step 3: Write `internal/behaviortree/room_engine_test.go`**

Each test below uses Pattern A for fixtures and Pattern B for tree construction. Test bodies follow arrange/act/assert; do NOT paste full Go bodies — sketch only.

| Test | Setup | Act | Assert |
|---|---|---|---|
| `TestTryRoomBehavior_NoRoom_ReturnsFalse` | Empty `rooms.SeedRoomsForTest(nil, nil)` so `LoadRoom(99)` returns nil. | `TryRoomBehavior(99, EventContext{})`. | Returns false; engine's `noRoomTree[99]` NOT set (verify via direct map read). |
| `TestTryRoomBehavior_NoTreeFile_NegativeCache` | Seed room 1 in zone "TestZone" but no YAML on disk for it. Pre-condition: `HasNoRoomTree(1)` returns false. | First call `TryRoomBehavior(1, …)`. Then second call. | First returns false, sets `noRoomTree[1]=true`. Second returns false without re-reading os.Stat (assert via a counter or by deleting the would-be path between calls and verifying second still returns false). |
| `TestTryRoomBehavior_LoadParseError_FailureNotCached` | Seed room 1, write malformed YAML at the resolved path. | `TryRoomBehavior(1, …)`. | Returns false; `HasNoRoomTree(1)` is still false (parse errors are NOT cached, per current room-side code at helpers.go:101-104; this differs from the mob path at line 152). |
| `TestTryRoomBehavior_HappyPath_Success` | Seed room 1. Call `engine.LoadRoomTree(1, tmpYamlPath)` directly with a Pattern B tree like `tree: { type: action, do: set_state, key: t, value: ok }`. | `TryRoomBehavior(1, EventContext{EventType: "anything_not_room_command"})`. | Returns true. |
| `TestTryRoomBehavior_RoomCommand_ReturnsIntercepted` | Seed room 1, load a tree with the `intercept` action. | `TryRoomBehavior(1, EventContext{EventType: "room_command", Command: "look"})`. Then load a tree WITHOUT intercept and call again. | First returns true (intercept set). Second returns false (Intercepted not set, regardless of Success). |
| `TestEnsureRoomBTreeState_PersistsAcrossCalls` | None. | Two calls with same roomId. Then a third call writing `state.Set("k", "v")` between them; verify the value survives. Also: spawn 50 goroutines via `sync.WaitGroup` calling `EnsureRoomBTreeState(42)` simultaneously; collect the returned pointers. | Both returned pointers identical. Concurrent calls all return the same pointer. The value persists. |

**Cleanup:** each test that loads a tree via `engine.LoadRoomTree` MUST defer a clear by direct-map mutation (same-package access to `engine.roomTrees`/`engine.noRoomTree` is allowed). Add a small private helper `clearEngineRoomState(t *testing.T, roomIds ...int)` in this file that strips the room IDs from both maps under the engine's mutex.

**Reference style:** match `internal/behaviortree/engine_test.go` for assertion idioms (use `t.Errorf`/`t.Fatalf`, not `assert.*` — this package doesn't import testify).

### Verify

- [ ] **Step 4: Build + vet + scoped test**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./internal/behaviortree/...
go vet ./internal/behaviortree/...
go test ./internal/behaviortree/... -run "TestTryRoomBehavior|TestEnsureRoomBTreeState"
```

Expected: 6 new tests pass, all existing tests still pass.

- [ ] **Step 5: Full project test sweep**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go test ./...
```

Expected: only `TestRoom_AddTemporaryExit/duplicate_name_rejected` fails (baseline).

### Commit

- [ ] **Step 6: Commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add internal/behaviortree/room_engine_test.go
git commit -m "$(cat <<'EOF'
test(behaviortree): room engine entry-point coverage

Add room_engine_test.go with 6 tests for TryRoomBehavior and
EnsureRoomBTreeState. Covers nil-room early-return, missing-YAML
negative-cache, malformed-YAML parse error (not cached, by current
contract), happy path Success, room_command interception via
ctx.Intercepted, and double-checked-lock state persistence under
concurrent invocation.

Tests use the seedTest* helpers from commit 1 and load in-memory
trees via LoadTreeFromBytes / LoadRoomTree against tempdir paths.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Phase 4c condition coverage

Append 6 tests to the existing `conditions_test.go` for `command_matches`, `command_rest_contains`, and `mob_in_room`.

**Files:**
- Modify: `internal/behaviortree/conditions_test.go` (append only — do NOT touch existing tests).

**Complexity:** Trivial-medium. The first two conditions need only an `EvalContext`; the third needs a room fixture (Pattern A).

### Discovery

- [ ] **Step 1: Re-read the conditions**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
sed -n '1,75p' internal/behaviortree/conditions_room.go
sed -n '60,75p' internal/behaviortree/conditions_mob.go
```

Confirm:
- `condCommandMatches` (`conditions_room.go:7-23`) reads `params["commands"]` as `[]any`, lowercases each entry, compares to lowercased `ctx.Event.Command`.
- `condCommandRestContains` (`conditions_room.go:27-43`) reads `params["keywords"]`, lowercases entries, uses `strings.Contains` against lowercased `ctx.Event.Rest`.
- `condMobInRoom` (`conditions_mob.go:61-74`) reads `params["mob_id"]`, calls `rooms.LoadRoom(ctx.RoomId)`, scans `room.GetMobs(rooms.FindAll)` for an instance with matching template MobId.

- [ ] **Step 2: Re-read the existing test style**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
sed -n '1,50p' internal/behaviortree/conditions_test.go
sed -n '193,212p' internal/behaviortree/conditions_test.go
```

Match the `TestCond<Name>_<Scenario>` naming and the `fn := LookupCondition(...)` direct-call pattern.

### Implementation

- [ ] **Step 3: Append the 6 tests**

| Test | Notes |
|---|---|
| `TestCondCommandMatches_Hit` | `params["commands"] = []any{"look", "examine"}`, `Event.Command = "look"`. Assert Success. |
| `TestCondCommandMatches_Miss` | Same params, `Event.Command = "east"`. Assert Failure. Add a sub-case (or sibling test `TestCondCommandMatches_MissingParam`) where `params` lacks `commands` → Failure. |
| `TestCondCommandRestContains_Hit` | `params["keywords"] = []any{"chest"}`, `Event.Rest = "open the wooden chest"`. Assert Success. Verify case-insensitivity by using `Event.Rest = "OPEN THE WOODEN CHEST"`. |
| `TestCondCommandRestContains_EmptyRest` | Same params, `Event.Rest = ""`. Assert Failure. |
| `TestCondMobInRoom_Hit` | Use Pattern A: `seedTestRoom(t, 1, "TestZone")`, `seedTestMob(t, 5, 105, 1, "Goblin")`. Then `rooms.LoadRoom(1).AddMob(105)`. `params["mob_id"] = 5`, `ctx.RoomId = 1`. Assert Success. |
| `TestCondMobInRoom_NoRoom` | `rooms.SeedRoomsForTest(nil, nil)`. `params["mob_id"] = 5`, `ctx.RoomId = 99`. Assert Failure (no panic). |

**Per Pattern E:** for `command_matches` and `command_rest_contains`, the `EvalContext` only needs `Event`. For `mob_in_room`, also set `RoomId`.

### Verify

- [ ] **Step 4: Build + vet + scoped test**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./internal/behaviortree/...
go vet ./internal/behaviortree/...
go test ./internal/behaviortree/... -run "TestCondCommandMatches|TestCondCommandRestContains|TestCondMobInRoom"
```

Expected: 6 new tests pass.

- [ ] **Step 5: Full project test sweep**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go test ./...
```

Expected: only the baseline failure.

### Commit

- [ ] **Step 6: Commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add internal/behaviortree/conditions_test.go
git commit -m "$(cat <<'EOF'
test(behaviortree): Phase 4c conditions

Append 6 tests to conditions_test.go covering command_matches,
command_rest_contains, and mob_in_room. Hit + miss + edge case
(missing param, empty rest, nil room) per the spec D1 bar.

mob_in_room uses the seedTest* helpers from commit 1; the other two
need only an EvalContext.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Phase 4c action coverage

Add 9 tests for the Phase 4c actions in a new `actions_test.go` file. Largest test count of any single commit in this stage.

**Files:**
- Create: `internal/behaviortree/actions_test.go`

**Complexity:** Medium. 9 tests; all need fixtures from Pattern A; some need event-listener taps (Pattern C/D).

### Discovery

- [ ] **Step 1: Re-read each action**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
sed -n '104,142p' internal/behaviortree/actions_dialogue.go
sed -n '45,78p'  internal/behaviortree/actions_quest.go
sed -n '82,103p' internal/behaviortree/actions_dialogue.go
sed -n '88,105p' internal/behaviortree/actions_room.go
sed -n '73,84p'  internal/behaviortree/actions_combat.go
```

Confirm contracts (each is a `func(params map[string]any, ctx *EvalContext) Result`):

- `actMobSay` / `actMobEmote` — find a room mob with matching `mob_id`, call `m.Command("say "/"emote "+text)`. Failure on nil room or no matching mob.
- `actGrantMutation` — Failure on nil user. Returns Success when pool empty (line 51-53). When pool non-empty, picks one and writes to `user.Character.Mutations` map.
- `actGiveGold` — Failure on nil user or `amount <= 0`. Else `user.Character.Gold += amount` and `user.SendText("You receive %d gold.\n", amount)`.
- `actSendUserText` — Failure on nil user. Else `user.SendText(text)`.
- `actSendRoomText` — Failure on nil room. Else `room.SendText(text)`.
- `actIntercept` — sets `ctx.Intercepted = true`, returns Success unconditionally.
- `actRemoveBuff` — Failure on nil user. Calls `user.Character.RemoveBuff(buffId)`. Note: does NOT validate buff_id presence; missing buff_id results in calling `RemoveBuff(0)` which is a no-op.
- `actMovePlayer` — Failure on `room_id == 0`. Calls `rooms.MoveToRoom(ctx.Event.UserId, roomId)`.

- [ ] **Step 2: Verify the events listener API for Pattern C/D**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
grep -n "^func RegisterListener\|^func UnregisterListener\|^func DoListeners" internal/events/*.go
```

Expected: `RegisterListener(eventProto, fn, …) ListenerId` and `UnregisterListener(id)` exist. If `UnregisterListener` is missing, fall back to a closure-flag approach (Pattern C note). Document the actual signatures in the test file.

- [ ] **Step 3: Verify mutations seeding strategy**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
grep -n "SeedMutations\|^var allMutations" internal/mutations/*.go
```

Expected: `allMutations` is a package-private map at `mutations.go:106`; there is NO public `SeedMutationsForTest`. The `actGrantMutation` test has two viable paths:

1. **Empty-pool happy path** — assert that with an empty pool the action still returns Success (per the comment at `actions_quest.go:52`).
2. **Non-empty pool path** — requires a mutations seed helper. If the executor judges this a 5-min add (one-line `SeedMutationsForTest(map[string]*MutationSpec) func()` mirroring `buffs.SeedBuffsForTest`), add it as a `chore:` precursor commit before this task; otherwise skip the non-empty path and rely on a same-package `mutations_test.go:22` style direct-mutation IF testing in-package — which we are NOT (we're in `behaviortree`).

**Decision at execution:** if no helper exists and adding one feels intrusive, the test asserts only the empty-pool Success path AND the nil-user Failure path. Document the deferred non-empty-pool coverage in `project_error_handling_audit_findings.md` under `## Open / In-Progress` with `surfaced during 1.6`.

### Implementation

- [ ] **Step 4: Write `internal/behaviortree/actions_test.go`**

| Test | Pattern | Notes |
|---|---|---|
| `TestActMobSay_FindsMobInRoomAndQueuesCommand` | A + C | Seed room 1, mob (template id 5, instance 105) in room 1. `params = {mob_id: 5, text: "hello"}`, `ctx.RoomId = 1`. Capture `events.Input` events, assert one was queued for InstanceId 105 with text containing "say hello". Empty-room sub-test: room with no mobs → Failure. |
| `TestActMobEmote_FindsMobInRoomAndQueuesCommand` | A + C | Same shape, `text: "waves"`, expect "emote waves". |
| `TestActGrantMutation_AddsMutationToCharacter` | A | See Step 3 decision tree — minimum: empty-pool returns Success, nil-user returns Failure. Stretch: non-empty pool writes a key. |
| `TestActGiveGold_IncreasesGoldAndNotifies` | A + D | User starts with Gold 100. `params = {amount: 25}`. After: Gold == 125. Capture `events.Message` events, assert one for the user containing "25 gold". `amount <= 0` sub-test → Failure (try `amount: 0` and `amount: -5`). |
| `TestActSendUserText_DeliversToUser` | A + D | Capture `events.Message`, assert text matches `params["text"]`. Nil user (UserId 99 not seeded) → Failure. |
| `TestActSendRoomText_BroadcastsToRoom` | A + D | Capture `events.Message` for any user in the room. Nil room (RoomId 99) → Failure. Per Pattern D fallback, if the per-user message routing makes this brittle, accept "action returned Success" as the assertion. |
| `TestActIntercept_SetsCtxIntercepted` | E only | `ctx := &EvalContext{}`. Call action, assert `ctx.Intercepted == true` and result == Success. |
| `TestActRemoveBuff_RemovesBuffFromUser` | A + buffs.SeedBuffsForTest | Seed buff 100. Apply to user via `user.Character.AddBuff(100, "test")`. Pre-assert `HasBuff(100) == true`. `params = {buff_id: 100}` → action returns Success, post-assert `HasBuff(100) == false`. Missing user (UserId 99) → Failure. |
| `TestActMovePlayer_TeleportsUser` | A | Seed two rooms (1 and 2), user in room 1. `params = {room_id: 2}` → user.Character.RoomId == 2. `room_id: 0` → Failure. |

**Direct-call dispatch per Pattern E.** Use `LookupAction("mob_say")` etc. — bypass `ActionNode.Evaluate` so the perception-scaled delay layer never fires and we don't need to worry about `delayedActions` (`actions.go:74`).

**Listener cleanup:** at the top of any test using the events queue, install the listener with a `t.Cleanup` to unregister or set a closure flag to ignore further events. NEVER leak listeners between tests — they bleed into other tests in the same `go test` run.

**Buffs dependency:** `actRemoveBuff` test needs `buffs.SeedBuffsForTest` (`internal/buffs/test_helpers.go:6`). Import that directly; do NOT add it to the `seedTest*` helper trio (those wrap mob/user/room only).

### Verify

- [ ] **Step 5: Build + vet + scoped test**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./internal/behaviortree/...
go vet ./internal/behaviortree/...
go test ./internal/behaviortree/... -run "^TestAct(MobSay|MobEmote|GrantMutation|GiveGold|SendUserText|SendRoomText|Intercept|RemoveBuff|MovePlayer)"
```

Expected: 9 new tests pass.

- [ ] **Step 6: Full project test sweep**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go test ./...
```

Expected: only the baseline failure.

### Commit

- [ ] **Step 7: Commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add internal/behaviortree/actions_test.go
git commit -m "$(cat <<'EOF'
test(behaviortree): Phase 4c actions

Add actions_test.go with 9 tests covering mob_say, mob_emote,
grant_mutation, give_gold, send_user_text, send_room_text,
intercept, remove_buff, and move_player. Each test is happy path +
1-2 obvious failure modes (D1 depth).

Tests dispatch the action via LookupAction() (bypassing the
delayedActions layer) and assert via either direct state read or
via an events.RegisterListener tap on events.Input/events.Message.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: actSummonCompanion hostile branch

Add the one cross-subsystem test for `actSummonCompanion` (`internal/behaviortree/actions_mob.go:48-101`) hostile path. Same file as Task 4 (`actions_test.go`).

**Files:**
- Modify: `internal/behaviortree/actions_test.go` (append).

**Complexity:** Medium. One test, but touches three subsystems: mob spawn (`mobs.NewMobById`), aggro (`Character.SetAggro`), command queue (`companion.Command("lookfortrouble", 4)`).

### Discovery

- [ ] **Step 1: Re-read the action**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
sed -n '48,101p' internal/behaviortree/actions_mob.go
```

Confirm at line 71: `hostile := getStringParam(params, "hostile") == "true"`. The test MUST pass `params["hostile"] = "true"` as a STRING LITERAL, not a Go bool. **This is the gotcha** documented in the memory file `project_actsummoncompanion_hostile_should_be_bool.md` (created today). The future fix is to switch to a bool helper; this test locks current behavior.

- [ ] **Step 2: Re-read `mobs.NewMobById`**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
sed -n '303,400p' internal/mobs/mobs.go
```

Note: `NewMobById` calls `LoadMobInstance` (line 344) which may hit disk. In tests with no on-disk instance file, `LoadMobInstance` should return nil and the function continues with the `forceStatPool`/random branch. Verify at execution time by running the test once and checking for unexpected disk-load logs.

If `LoadMobInstance` fails in test mode, the spec risk register accepts asserting only `room.AddMob` invocation + aggro state, NOT the spawn lifecycle. Decision at execution; default to full assertion.

- [ ] **Step 3: Verify Character.Aggro and skill registration**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
sed -n '1,40p' internal/characters/aggro.go
grep -n "skills.Manifestation\b" internal/skills/*.go internal/characters/*.go
```

Confirm `Character.Aggro` is a `*Aggro` struct with `UserId int` field (`aggro.go:26-30`). The action calls `mob.Character.GetSkillLevel(skills.Manifestation)` (`actions_mob.go:65`); if `skills.Manifestation` requires registry initialization that's not present in the test, the test seed must initialize it OR the calling mob can have `Stats.Charisma.ValueAdj` set such that the formula still produces a valid pool with `manifestSkill = 0`.

### Implementation

- [ ] **Step 4: Append `TestActSummonCompanion_HostileSetsAggroAndEngages` to `actions_test.go`**

Setup:
- Seed room 1.
- Seed two mob templates: caller (template 1, instance 100, in room 1) AND companion template (template 7) — the companion needs a spec in `mobs` so `NewMobById` can clone it.
- Seed user 1 (charname "Aliceia") in room 1.
- Set `EvalContext`: `InstanceId = 100`, `RoomId = 1`, `Event.UserId = 1`.
- Set `params = map[string]any{"mob_id": 7, "hostile": "true", "count": 1, "base_pool": 50}` — note **string literal "true"** per the gotcha.

Capture pre-action state:
- Snapshot the room's mob list (`room.GetMobs(rooms.FindAll)`) — should contain only instance 100.

Install an `events.Input` listener (Pattern C) to capture commands queued on whichever new instance shows up.

Act:
- Call `LookupAction("summon_companion")(params, ctx)`. Assert result == Success.

Assert:
1. Room mob list now has one MORE entry. Find the new instance ID by set-difference against the pre-snapshot.
2. The new instance's `Character.Aggro != nil` and `Character.Aggro.UserId == 1`.
3. The captured `events.Input` slice contains an entry with `MobInstanceId == newInstanceId` and `InputText == "lookfortrouble"`.

Cleanup: remove the new instance from the registry (call `mobs.RemoveMobInstance(...)` if exposed, else don't worry — `SeedMobsForTest`'s cleanup wipes `mobInstances` entirely).

**Do NOT cover the non-hostile (charmed companion) branch** — out of scope per the spec.

### Verify

- [ ] **Step 5: Build + vet + scoped test**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./internal/behaviortree/...
go vet ./internal/behaviortree/...
go test ./internal/behaviortree/... -run "TestActSummonCompanion_HostileSetsAggroAndEngages"
```

Expected: pass.

- [ ] **Step 6: Full project test sweep**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go test ./...
```

Expected: only the baseline failure.

### Commit

- [ ] **Step 7: Commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add internal/behaviortree/actions_test.go
git commit -m "$(cat <<'EOF'
test(behaviortree): actSummonCompanion hostile branch

Append TestActSummonCompanion_HostileSetsAggroAndEngages to
actions_test.go. Verifies that with hostile="true" (string literal,
matching current parsing in actions_mob.go:71) the action:
1. spawns a new mob instance into the room
2. sets the new mob's Aggro to the triggering player
3. queues "lookfortrouble" on the new mob

Non-hostile / charmed-companion branch deliberately not covered —
out of scope per spec. The string-literal "true" param is a known
ergonomics issue tracked in
project_actsummoncompanion_hostile_should_be_bool.md.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

**Note on commit fold:** if Steps 4-6 turned out trivial and the executor preferred to fold this commit into Task 4, that's allowed per spec — the rationale must be noted in the Task 4 commit body if folded ("includes the actSummonCompanion hostile-branch test from Task 5"). Default: keep separate for revert isolation.

---

## Task 6: give.go quest-engine vs btree handoff regression

Hardest task in this stage. Adds the single regression test for `give.go`'s ordering: quest engine `item_give` intercept first; behavior tree `player_give` second when not intercepted. Locks the architecture established by `f7a647b3 feat: quest engine intercepts item_give before transfer in give.go` and reinforced by `c3c48a7c fix: smoke test bugfixes for Phase 4b/4c`.

**Files:**
- Create: `internal/questengine/test_helpers.go` — required precursor (Pattern F).
  `globalEngine` at `loader.go:109` is package-private; `usercommands` is an
  external package and cannot touch it. The existing `loader_test.go:98` does
  `globalEngine = nil` from inside the package — that path is closed to us.
- Create: `internal/usercommands/give_test.go`
- Reference (read first, do NOT modify): `internal/usercommands/usercommands_test.go`
  and `internal/usercommands/helpfile_completeness_test.go` already exist in this
  package. Skim them for any cross-cutting fixture/helper pattern before
  bootstrapping fresh.

**Complexity:** Medium-high. The setup pulls in actions, events, items, mobs, rooms, users, and questengine. **60-min wall-clock cap on setup** (per spec risk register). If the executor is past 60 min on setup, fall back to the smaller-scope test described below.

### Decision tree (apply BEFORE writing any code)

1. **Read `give.go:175-237` end-to-end.** Trace the call sequence for a player giving an item to a mob.
2. **Read `internal/questengine/types.go`** in full. Inventory the minimum field set on `QuestDef` / `TriggerDef` / `ActionDef` needed for `Notify("item_give", ...)` to return `{Handled:true, ConsumeItem:true}`. Cross-reference with `engine.go:43-130` (the Notify and evaluate functions) and `engine.go:99-109` (`Handled` and `ConsumeItem` set conditions).
3. **Engine reset strategy is locked: Pattern F precursor commit is REQUIRED.** Add a tiny exported `ResetEngineForTest()` helper in `internal/questengine/test_helpers.go` (snapshot+restore via `questengine.NewEngine()`). See Step 4 for the full code + commit message. The precursor lands BEFORE `give_test.go`.
4. **Plan the inputs to `Give(rest, user, room, flags)`:**
   - `rest = "questitem mobname"` (must split via `util.StripPrepositions` and `SplitButRespectQuotes` correctly)
   - `user` = a seeded UserRecord with a quest item in their backpack
   - `room` = a seeded room containing the user and the mob (mob found by name)
   - `flags` = `events.EventFlag(0)` or whatever default is fine

### Discovery

- [ ] **Step 1: Read `give.go` and the quest-engine wiring**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
sed -n '1,60p'    internal/usercommands/give.go
sed -n '145,245p' internal/usercommands/give.go
sed -n '1,50p'    internal/questengine/types.go
sed -n '40,135p'  internal/questengine/engine.go
sed -n '105,140p' internal/questengine/loader.go
```

Confirm:
- `give.go:179-185` calls `questengine.GetEngine().Notify("item_give", EventDetails{UserId, RoomId, MobId, ItemId}, bridge, bridge)`.
- `give.go:187-207` is the consume branch (early return).
- `give.go:230-237` is the btree branch (`TryMobBehavior` with `EventType: "player_give"`).
- `questengine.RegisterQuest(*QuestDef)` (`engine.go:25`) is exported; the test can register a fake quest into the global engine.

- [ ] **Step 2: Find an existing quest YAML to mimic for the trigger shape**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
find _datafiles/quests -type f -name "*.yaml" 2>/dev/null | head -3
```

Read one to understand the `ConsumeItem` field layout.

- [ ] **Step 3: Verify items.Item construction for the test**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
grep -n "^func New\|^func.*StoreItem\|^func.*FindInBackpack" internal/items/*.go internal/characters/*.go | head -10
```

Confirm `items.New(itemId int) Item` and `Character.StoreItem(item) bool`.
**Prefer `StoreItem` over direct slice append** — it matches production-path
behavior (capacity / weight checks). Direct append is acceptable only if a
specific test scenario requires bypassing those checks (none do here).

### Implementation

- [ ] **Step 4: (REQUIRED) Add `questengine.ResetEngineForTest`**

Create `internal/questengine/test_helpers.go`. This is a precursor commit, NOT
conditional — `globalEngine` at `loader.go:109` is package-private and the
existing in-package `loader_test.go:98` `globalEngine = nil` swap is unreachable
from `usercommands`. The exported helper is the only path:

```go
package questengine

// ResetEngineForTest replaces the global engine with a fresh empty Engine
// and returns a cleanup function that restores the original. Test-only.
func ResetEngineForTest() func() {
    orig := globalEngine
    globalEngine = NewEngine()
    return func() {
        globalEngine = orig
    }
}
```

Commit this as a precursor with message:

```
chore(questengine): add ResetEngineForTest helper

Mirrors mobs.SeedMobsForTest / users.SeedUsersForTest / rooms.SeedRoomsForTest
pattern. Required by internal/usercommands/give_test.go to reset the
global quest engine between tests without leaking state.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
```

Commit BEFORE writing `give_test.go`. Use the chore commit message above
verbatim.

- [ ] **Step 5: Write `internal/usercommands/give_test.go`**

Test name: `TestGive_QuestEngineInterceptsBeforeBtreePlayerGive`

Setup (per Pattern A, plus questengine seeding):
- `seedTestRoom(t, 1, "TestZone")`.
- `seedTestUser(t, 1, "alice", "Aliceia", 1)` — but reach into the underlying record to add a quest item to `Character.Items`.
- `seedTestMob(t, 1, 100, 1, "Mister Quester")` — caller for `room.FindByName("quester")`.
- Reset the questengine: `cleanupEngine := questengine.ResetEngineForTest(); defer cleanupEngine()`.
- Build a single `*QuestDef` whose `Triggers[0]` matches `Event = "item_give"`, `MobId` = 1, `ItemId` = the test item's ID, and `Actions[0].ConsumeItem = 1` (or whatever the type expects). Register via `questengine.GetEngine().RegisterQuest(quest)`.
- Place player and mob in room 1: `room.AddPlayer(1)`, `room.AddMob(100)`.

Pre-assertion:
- Player has the quest item in their backpack (count = 1).

Act:
- Call `Give("questitem quester", user, room, events.EventFlag(0))`.

Assert:
1. Item removed from player's backpack (count = 0).
2. Mob does NOT have the item (it was consumed, not transferred). Verify by inspecting `m.Character.Items`.
3. The behavior tree `player_give` event was NOT fired. **Use Approach A
   (positive observable absence), not Approach B (absence of setup).**
   - **Approach A (recommended):** Seed a behavior tree on the mob with a
     `player_give` handler that mutates observable state — for example, a
     `set_state: { key: "btree_fired", value: 1 }` action. After `Give(...)`
     returns, assert `mob.BTreeState`'s `btree_fired` is still 0. This
     **actively** proves the handler did not dispatch.
   - **Approach B (fragile, do not use):** Don't seed a btree at all and rely
     on the absence of any btree call mattering. This passes even when the
     regression is real (no btree → no event observable either way), so it
     doesn't actually verify the consume branch's early return. Reject.
   - Tree YAML for the handler is tiny — see `behaviortree.LoadTreeFromBytes`
     used by Pattern B in Task 2. If the tree fixture is awkward (e.g., needs
     `set_state` action support not present in the action registry — verify
     during Step 1), fall back to a different observable side-effect action.
     Do NOT fall back to Approach B.

**Inverse path (optional, if cheap):**
- Reset quest engine to empty (no consume).
- Call `Give(...)` again with a fresh item.
- Assert: item transferred to mob; btree gets a chance (no btree → mob does the default `emote considers...`).

If the wall-clock cap (60 min) is hit during setup, drop the inverse path and ship the consume-only assertion.

### Verify

- [ ] **Step 6: Build + vet + scoped test**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./internal/usercommands/...
go vet ./internal/usercommands/...
go test ./internal/usercommands/... -run "TestGive_QuestEngineInterceptsBeforeBtreePlayerGive"
```

Expected: pass.

- [ ] **Step 7: Full project test sweep**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go test ./...
```

Expected: only the baseline failure. The give-test must not leak state into subsequent tests (this is what `ResetEngineForTest` cleanup is for).

### Commit

- [ ] **Step 8: Commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add internal/usercommands/give_test.go
git commit -m "$(cat <<'EOF'
test(usercommands): give.go quest-engine vs btree handoff regression

Add give_test.go with TestGive_QuestEngineInterceptsBeforeBtreePlayerGive.
Locks the architecture established by f7a647b3 (feat: quest engine
intercepts item_give before transfer in give.go) and reinforced by
c3c48a7c (fix: smoke test bugfixes for Phase 4b/4c, which removed
duplicate grant_quest from Tessara/Pell btrees because the quest
engine handles advancement).

The test seeds a quest with item_give → ConsumeItem trigger, then
calls Give() and verifies:
- the item is removed from the player's inventory
- the item is NOT transferred to the mob
- the behavior tree player_give handler is NOT invoked

Future refactors that re-order give.go's consume-then-btree flow,
or re-introduce double-handling, will fail this test.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

If the inverse path was added, the commit body should include a "Plus: inverse path verifies normal transfer + btree dispatch when no quest is consuming" line.

---

## Task 7: Mark 1.6 complete in overview

Flip the Stage 1.6 status. Docs-only commit. No memory-file populate (1.6 didn't accumulate per-task findings the way 1.5 did — unless `fix:`/`chore:` precursors fired).

**Files:**
- Modify: `docs/superpowers/code_cleanup_stage_1_overview.md`
- (Optional) Modify: `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\project_error_handling_audit_findings.md` if any 1.6 findings landed there during execution — append a `## 1.6 Carryover` section noting the test author's perspective findings.

**Complexity:** Trivial.

### Update

- [ ] **Step 1: Flip 1.6 status in overview**

Open `docs/superpowers/code_cleanup_stage_1_overview.md`. Line 22:

```markdown
| 1.6 | Test Coverage for New Systems | 5h | Low | Not started |
```

Change `Not started` to `Complete`. Do NOT touch other rows.

- [ ] **Step 2: (Conditional) Update `project_error_handling_audit_findings.md`**

If any test in commits 2-6 surfaced a finding (clear bug → preceding fix commit, or ambiguous → pending), append a section under the existing memory file:

```markdown
## 1.6 Carryover (test-author findings)

- internal/behaviortree/<file>.go:<LINE> — <observation> — <severity> — <fixed|pending|deferred> — <rationale>
```

If 1.6 surfaced zero findings, skip this step.

### Verify

- [ ] **Step 3: Build + vet + test (docs-only should be no-op)**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./...
```

Expected: unchanged from Task 6 baseline (only the documented `TestRoom_AddTemporaryExit` failure).

### Commit

- [ ] **Step 4: Commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add docs/superpowers/code_cleanup_stage_1_overview.md
# (also stage memory file if Step 2 was non-empty)
git commit -m "$(cat <<'EOF'
docs: mark code cleanup 1.6 complete

Flip Stage 1.6 status to Complete in code_cleanup_stage_1_overview.md.

23 new tests across 4 files (3 new: room_engine_test.go,
actions_test.go, give_test.go; 1 appended: conditions_test.go).
Test-only substage; no production code change.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

No PATCH_NOTES.md entry (test additions are internal, zero player impact).

---

## Task 8: Final verification

Confirm branch shape, run the full test sweep with `-race` on the behaviortree package, and verify the count of new tests matches the spec.

- [ ] **Step 1: Confirm branch commit count and order**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git log --oneline feature/stage-1.6-test-coverage ^development
```

Expected: 7 commits in this order (newest first), or 8 if Task 6 added the precursor `chore(questengine): add ResetEngineForTest helper`:

1. `docs: mark code cleanup 1.6 complete`
2. `test(usercommands): give.go quest-engine vs btree handoff regression`
3. (optional) `chore(questengine): add ResetEngineForTest helper`
4. `test(behaviortree): actSummonCompanion hostile branch` (or folded into 5)
5. `test(behaviortree): Phase 4c actions`
6. `test(behaviortree): Phase 4c conditions`
7. `test(behaviortree): room engine entry-point coverage`
8. `test(behaviortree): add test_helpers_test.go shared fixtures`

If the executor folded Task 5 into Task 4, expect 6 commits; if a `fix:` precursor was added during execution, expect more. Document the actual count.

- [ ] **Step 2: Verify branch diff shape**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git diff --stat development...feature/stage-1.6-test-coverage
```

Expected files changed:
- `internal/behaviortree/test_helpers_test.go` (created)
- `internal/behaviortree/room_engine_test.go` (created)
- `internal/behaviortree/conditions_test.go` (modified — appended only)
- `internal/behaviortree/actions_test.go` (created)
- `internal/usercommands/give_test.go` (created)
- (optional) `internal/questengine/test_helpers.go` (created)
- `docs/superpowers/code_cleanup_stage_1_overview.md` (1-line status flip)
- (optional) `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\project_error_handling_audit_findings.md` (only if 1.6 surfaced findings)

No production source files (`*.go` outside `*_test.go`) should appear EXCEPT the optional `questengine/test_helpers.go`. If anything else shows up: STOP, investigate (probably an in-scope `fix:` precursor — verify it has a corresponding commit).

If `.claude/settings.local.json`, `feedback/*.txt`, or `Screenshot*.png` show up: STOP, `git restore --staged` them, and investigate.

- [ ] **Step 3: Final whole-project verification**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./... -count=1
```

Expected: clean. The single pre-existing `TestRoom_AddTemporaryExit/duplicate_name_rejected` failure remains (baseline noise per `project_rooms_package_audit_needed.md`). No other test regressions. `-count=1` defeats stale cache.

- [ ] **Step 4: Race-mode pass on behaviortree**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go test ./internal/behaviortree/... -race
```

Expected: clean. The `TestEnsureRoomBTreeState_PersistsAcrossCalls` concurrent sub-case from Task 2 needs `-race` to be meaningful — if it fires a race report, that's a real finding requiring a `fix:` commit (the spec risk register notes this) OR a `t.Skip("race-mode only — known flaky on Windows")` with a memory file note.

- [ ] **Step 5: Confirm new test count**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go test ./internal/behaviortree/... -run "TestTryRoomBehavior|TestEnsureRoomBTreeState|TestCondCommandMatches|TestCondCommandRestContains|TestCondMobInRoom|TestActMobSay|TestActMobEmote|TestActGrantMutation|TestActGiveGold|TestActSendUserText|TestActSendRoomText|TestActIntercept|TestActRemoveBuff|TestActMovePlayer|TestActSummonCompanion_HostileSetsAggroAndEngages" -v 2>&1 | grep -c "^=== RUN"
go test ./internal/usercommands/... -run "TestGive_QuestEngineInterceptsBeforeBtreePlayerGive" -v 2>&1 | grep -c "^=== RUN"
```

Expected: total of 22-23 new top-level tests (depending on whether `TestCondCommandMatches_MissingParam` is a sibling test or a sub-test of `_Miss`). The spec says **23 total** — match within ±1 acceptable.

- [ ] **Step 6: Smoke test — Phase 4c room behavior still fires**

Boot a local server, walk a player into Sanctum Basin tutorial (or any Phase 4c room with a behavior tree), run `look` and `ask` to confirm room trees still fire. The 1.6 tests don't replace functional smoke; this is a final sanity check.

Any deviation from expected = STOP. Revert the offending commit and diagnose.

---

## Merge to development (after user review)

Do NOT merge until the user has reviewed the branch. Once approved:

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git checkout development
git merge --no-ff feature/stage-1.6-test-coverage -m "$(cat <<'EOF'
merge: stage 1.6 test coverage for new systems

Adds 23 new tests across four files locking in the behavior of five
recently-shipped subsystems with zero prior coverage:
- Room behavior tree engine entry point (TryRoomBehavior,
  EnsureRoomBTreeState)
- Phase 4c conditions (command_matches, command_rest_contains,
  mob_in_room)
- Phase 4c actions (mob_say, mob_emote, grant_mutation, give_gold,
  send_user_text, send_room_text, intercept, remove_buff, move_player)
- actSummonCompanion hostile branch
- give.go quest-engine vs btree player_give handoff (regression test
  for the architecture established by f7a647b3 + c3c48a7c)

Test-only; zero production code change. Verified by go test ./...
+ go test ./internal/behaviortree/... -race + targeted Phase 4c
room smoke.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

Do NOT push to origin. Let 1.6 ride on `development` alongside the rest of the Stage 1 backlog. Future `fix:` work in the targets covered by these tests will surface regressions immediately.

---

## Summary

- **Commits:** 7 (or 8 if `chore(questengine): add ResetEngineForTest` precursor was needed; or 6 if Task 5 folded into Task 4).
- **Test count:** 23 new tests across `internal/behaviortree/test_helpers_test.go` (helpers only), `room_engine_test.go` (6), `conditions_test.go` (+6), `actions_test.go` (10 = 9 + companion), `internal/usercommands/give_test.go` (1).
- **Production code change:** 0 lines (unless a test surfaced a clear bug per scope-creep policy, in which case a preceding `fix:` commit is permitted).
- **Verification gate:** `go build && go vet && go test ./...` clean after every commit; `-race` clean on behaviortree at the end.
