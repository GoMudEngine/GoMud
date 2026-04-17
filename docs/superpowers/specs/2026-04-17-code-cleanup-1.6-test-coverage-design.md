# Code Cleanup 1.6: Test Coverage for New Systems — Design Spec

## Goal

Add Go unit tests for five recently-shipped subsystems that have zero
dedicated coverage today. The targets are the room-level behavior tree
engine, the Phase 4c condition + action set, the quest-engine /
behavior-tree handoff in `give.go`, and the hostile branch of
`actSummonCompanion`. Tests use real fixtures (no mocks), happy path
plus 1–2 obvious failure modes per function, and live alongside (or
extend) the existing `internal/behaviortree/*_test.go` files.

This is an additive substage. No production code changes. The point is
to lock current behavior in place so 1.7's perf pass and any future
btree work has a regression net.

## Scope

**In scope (Option B — top 5):**

1. **Room behavior tree engine** — `TryRoomBehavior`,
   `EnsureRoomBTreeState`, state persistence across calls,
   command-interception via `ctx.Intercepted`.
   Files: `internal/behaviortree/helpers.go:81-125`,
   `internal/behaviortree/room_state.go`.
2. **Phase 4c conditions** — `command_matches`, `command_rest_contains`,
   `mob_in_room`. Files: `internal/behaviortree/conditions_room.go`,
   `internal/behaviortree/conditions_mob.go:61`.
3. **Phase 4c actions** — `mob_say`, `mob_emote`, `grant_mutation`,
   `give_gold`, `send_user_text`, `send_room_text`, `intercept`,
   `remove_buff`, `move_player`. Files:
   `internal/behaviortree/actions_dialogue.go`,
   `internal/behaviortree/actions_quest.go`,
   `internal/behaviortree/actions_room.go`,
   `internal/behaviortree/actions_combat.go:75`.
4. **Quest engine `item_give` vs btree `player_give` handoff** —
   one regression test exercising `give.go`'s ordering: quest engine
   intercept first (consume + return), behavior tree second when not
   intercepted. Architecture per CLAUDE.md "Quest Item Delivery —
   give.go Gotcha" (lines 204–214); established by
   commit `f7a647b3 feat: quest engine intercepts item_give before
   transfer in give.go` and refined by
   commit `c3c48a7c fix: smoke test bugfixes for Phase 4b/4c` (which
   removed redundant `grant_quest` from btrees because the quest
   engine handles advancement).
5. **`actSummonCompanion` with `hostile: "true"`** — verify aggro
   set on triggering player + `lookfortrouble` engage command queued
   on the spawned companion. File:
   `internal/behaviortree/actions_mob.go:48-101`.

**Out of scope:**

- Bcrypt SHA256 → bcrypt migration tests (overview line 260) — defer.
- File-logging config tests (overview line 261) — covered by 1.5
  smoke (mudlog fatal-path).
- Pre-existing failure
  `internal/rooms/TestRoom_AddTemporaryExit/duplicate_name_rejected`
  — option (i): leave untouched. The intent is captured in the
  `project_rooms_package_audit_needed.md` memory file; addressing it
  belongs to a future dedicated rooms-package pass, not 1.6.
- Branch coverage, table-driven exhaustiveness, property-based, or
  fuzz tests. Each test is hand-written: happy path + 1–2 failure
  modes (D1 depth).
- Full integration tests of the btree engine end-to-end (parser →
  evaluator → side effects). Engine-level coverage already exists in
  `engine_test.go` and `conditions_test.go`; 1.6 extends, not
  re-architects.
- Modifying any production code. Any defect surfaced by these tests
  pauses per the scope-creep policy and either gets a prior `fix:`
  commit (clear bug) or a memory-file note (ambiguous).

## Decisions Locked During Brainstorming

- **Scope is Option B (top 5).** Bcrypt and file-logging are
  background-priority and the budget is 4h; fitting the top 5 well
  beats covering all 7 thinly.
- **Real fixtures, no mocks.** Matches the existing pattern in
  `internal/hooks/hooks_test.go` (`seedAllRegistries()`) and the user
  feedback in `feedback_btree_death_actions.md` /
  "no mocking the database". Tests construct real `Mob`,
  `UserRecord`, `Room`, `BehaviorState` values via the existing
  `Seed*ForTest` helpers in `internal/mobs`, `internal/users`,
  `internal/rooms`. The only "fake" piece is the `EvalContext` itself,
  which the action functions take by pointer and is already a plain
  struct.
- **One test file per concern, append to existing where possible.**
  Phase 4c condition tests append to
  `internal/behaviortree/conditions_test.go` (existing). Phase 4c
  action tests go in a new sibling
  `internal/behaviortree/actions_test.go`. Room-engine tests go in a
  new sibling `internal/behaviortree/room_engine_test.go`. The quest +
  btree handoff lives in `internal/usercommands/give_test.go` (new
  file, package-local). The companion test goes in the same
  `actions_test.go` file as Item 3.
- **Coverage depth = D1 (happy + 1–2 failures), not branch
  coverage.** Per-function targets: confirm the happy path Success,
  one nil-input Failure, one wrong-type-param Failure where param
  parsing matters. No exhaustive enumeration of param combinations.
- **Helper file vs inline fixtures.** The action/condition tests
  need a much smaller fixture than `seedAllRegistries()` (one mob,
  one user, one room is plenty for most). Rather than reuse
  `seedAllRegistries()` from the `hooks` package (cross-package import
  cycle risk, plus it seeds buffs/biomes/spells we don't need), 1.6
  introduces a parallel `internal/behaviortree/test_helpers_test.go`
  with three helpers: `seedTestMob(...)`, `seedTestUser(...)`,
  `seedTestRoom(...)`. Each returns a cleanup function. Composable
  per test; no monolith. The helpers wrap the existing
  `mobs.SeedMobsForTest` / `users.SeedUsersForTest` /
  `rooms.SeedRoomsForTest` so the underlying registry-seeding logic
  is shared, not duplicated.
- **Test naming follows existing `conditions_test.go` convention.**
  `TestCond<Name>_<Scenario>` for conditions, `TestAct<Name>_<Scenario>`
  for actions, `TestTryRoomBehavior_<Scenario>` for engine entry
  points.
- **No `time.Sleep` in tests.** Engine evaluation is synchronous; if
  any candidate test wants to sleep, that's a sign the test is
  wrong-shaped — refactor or skip. Delayed-action queues are out of
  scope for 1.6 (they're 1.8's territory per the 1.5 spec).
- **Critical-failure rule during Verify.** A regression in any
  pre-existing test is a hard block. A failing 1.6 test on first
  commit is acceptable mid-stage as long as it's resolved before
  that commit lands; per-commit verification (`go test ./...` clean)
  applies to every commit. The pre-existing
  `TestRoom_AddTemporaryExit/duplicate_name_rejected` failure is
  documented as out of scope and does not block 1.6 (it was already
  failing before this branch).
- **Scope-creep policy carried over from 1.2b/1.5.** Clear bug found
  while writing a test → preceding `fix:` commit. Ambiguous case →
  pause and ask. Dead code spotted incidentally → `chore:` removal
  commit.
- **Commit cadence: one commit per scope-item area + one helper
  commit + one docs commit = 7 commits.** See Architecture below for
  rationale on why 4c conditions and 4c actions stay separated rather
  than folded — they touch sibling files but the action tests need
  the new test helpers from commit 1, while condition tests don't,
  so a clean dependency edge falls between them.

## Test Inventory

### Item 1 — Room btree engine (6 tests, in `room_engine_test.go`)

| Test name | Verifies |
|-----------|---------|
| `TestTryRoomBehavior_NoRoom_ReturnsFalse` | `rooms.LoadRoom` nil → returns false, no panic, no negative-cache write. |
| `TestTryRoomBehavior_NoTreeFile_NegativeCache` | Missing YAML on disk → `HasNoRoomTree(roomId)` becomes true; second call short-circuits without touching `os.Stat`. |
| `TestTryRoomBehavior_LoadParseError_FailureNotCached` | Malformed YAML → returns false; verify negative cache state matches actual implementation behavior (current code does NOT set it on parse error for rooms — locks that as the contract). |
| `TestTryRoomBehavior_HappyPath_Success` | Valid tree (built via `LoadTreeFromBytes`, registered in engine via `LoadRoomTree`) evaluates to Success → returns true. |
| `TestTryRoomBehavior_RoomCommand_ReturnsIntercepted` | EventType `room_command`; tree calls `intercept` action → returns true. Same tree without `intercept` and `room_command` event → returns false (regardless of Success). |
| `TestEnsureRoomBTreeState_PersistsAcrossCalls` | Two calls with same roomId → identical pointer; state values written between calls survive. Concurrent invocation under `sync.WaitGroup` → no double-init (locks the double-checked locking contract). |

### Item 2 — Phase 4c conditions (6 tests, appended to `conditions_test.go`)

| Test name | Verifies |
|-----------|---------|
| `TestCondCommandMatches_Hit` | `commands: ["look", "examine"]` against `Event.Command="look"` → Success. |
| `TestCondCommandMatches_Miss` | `commands: ["look"]` against `Event.Command="east"` → Failure. (Plus: missing `commands` param → Failure.) |
| `TestCondCommandRestContains_Hit` | `keywords: ["chest"]` against `Event.Rest="open the wooden chest"` → Success (case-insensitive). |
| `TestCondCommandRestContains_EmptyRest` | Rest is empty → Failure (no keyword can match empty string except "" which we don't allow). |
| `TestCondMobInRoom_Hit` | Room contains an instance of mob_id 5 → Success. |
| `TestCondMobInRoom_NoRoom` | Nil room (LoadRoom returns nil) → Failure, no panic. |

### Item 3 — Phase 4c actions (9 tests in `actions_test.go`)

| Test name | Verifies |
|-----------|---------|
| `TestActMobSay_FindsMobInRoomAndQueuesCommand` | Room has mob with target mob_id; action calls `mob.Command("say <text>")`; verify by inspecting `mob.GetCommandQueue()` (or equivalent observable state). Empty room → Failure. |
| `TestActMobEmote_FindsMobInRoomAndQueuesCommand` | Same shape as `mob_say`, command becomes `emote <text>`. |
| `TestActGrantMutation_AddsMutationToCharacter` | User has empty `Mutations` map; pool seeded with one mutation; after action → map contains that mutation key. Returns Success even when pool is empty (per code comment). |
| `TestActGiveGold_IncreasesGoldAndNotifies` | Gold before = 100, amount = 25 → Gold after = 125; user gets one SendText line containing "25 gold". `amount <= 0` → Failure. |
| `TestActSendUserText_DeliversToUser` | User receives the exact `text` param via `SendText`. Nil user (UserId not in registry) → Failure. |
| `TestActSendRoomText_BroadcastsToRoom` | Room receives text. Inspect via captured-write helper on `Room.SendText` if available, else by asserting room's outbound queue. Nil room → Failure. |
| `TestActIntercept_SetsCtxIntercepted` | Pre: `ctx.Intercepted = false`; after: `ctx.Intercepted = true`; returns Success. |
| `TestActRemoveBuff_RemovesBuffFromUser` | User has buff 100 applied; action with `buff_id: 100` → buff removed. Missing user → Failure. |
| `TestActMovePlayer_TeleportsUser` | `room_id: 2` against user in room 1 → user's `Character.RoomId == 2` after action. `room_id: 0` → Failure. |

### Item 4 — Quest engine + btree player_give handoff (1 test in `internal/usercommands/give_test.go`)

| Test name | Verifies |
|-----------|---------|
| `TestGive_QuestEngineInterceptsBeforeBtreePlayerGive` | Player gives a quest item to a quest mob. The test mock-bridges the quest engine to declare `Handled=true, ConsumeItem=true` on the `item_give` notify. Verify: (1) item is removed from the player's inventory, (2) the mob does NOT receive the item (it was consumed, not transferred), (3) the behavior tree's `player_give` handler is NOT invoked (since `give.go` returns true after the consume branch). Inverse path is sketched as a sub-case if cheap: when `Handled=false`, the item transfers AND the btree gets a chance — verify the btree event fires with the live (post-transfer) `giveItem.ItemId`. |

This is the "regression" test for the architecture established by
commit `f7a647b3` and reinforced by commit `c3c48a7c`'s removal of
duplicate `grant_quest` from the affected behavior trees (Tessara,
Pell). Future refactors that re-order the calls in `give.go` or
re-introduce double-handling will fail this test.

### Item 5 — `actSummonCompanion` hostile branch (1 test in `actions_test.go`)

| Test name | Verifies |
|-----------|---------|
| `TestActSummonCompanion_HostileSetsAggroAndEngages` | Caller mob present, target user in same room, params `mob_id: <spawnable>`, `hostile: "true"` (string, not bool — matches current implementation). After action: (1) a new mob instance was added to the room, (2) the new mob's `Character.Aggro` targets the calling user's UserId with `DefaultAttack`, (3) the new mob has `lookfortrouble` queued in its command queue. The non-hostile branch (omit `hostile`) is NOT covered here — that's outside the scope item, which targets the hostile path specifically. |

**Total: 23 new tests across 4 files (3 new files, 1 appended).**

## Architecture

**Execution order — 7 commits on `feature/stage-1.6-test-coverage` off
`development`:**

| # | Commit | Description |
|---|--------|-------------|
| 1 | `test(behaviortree): add test_helpers_test.go shared fixtures` | Three helpers: `seedTestMob`, `seedTestUser`, `seedTestRoom` (each returns a cleanup func). Plus a tiny `withFixtures(t, …)` combiner that defers all returned cleanups. No tests in this commit — just the helper file. Verifies: `go build ./internal/behaviortree/...` clean. |
| 2 | `test(behaviortree): room engine entry-point coverage` | Adds `room_engine_test.go` with the 6 tests from Item 1. Uses the helpers from commit 1 (mob+room) and constructs trees via `LoadTreeFromBytes` for in-memory tree fixtures. |
| 3 | `test(behaviortree): Phase 4c conditions` | Appends 6 tests to existing `conditions_test.go` for `command_matches`, `command_rest_contains`, `mob_in_room`. The first two need only an `EvalContext` (no fixtures); `mob_in_room` uses commit 1's helpers. |
| 4 | `test(behaviortree): Phase 4c actions` | Adds `actions_test.go` with the 9 action tests from Item 3. Uses helpers throughout. |
| 5 | `test(behaviortree): actSummonCompanion hostile branch` | Adds `TestActSummonCompanion_HostileSetsAggroAndEngages` to the same `actions_test.go` (one related test, same file boundary). Could be folded into commit 4 if execution finds the test trivially short; defaults to its own commit for clean revert surface. |
| 6 | `test(usercommands): give.go quest-engine vs btree handoff regression` | Adds `internal/usercommands/give_test.go` with the one regression test from Item 4. Uses `mobs.SeedMobsForTest` / `users.SeedUsersForTest` / `rooms.SeedRoomsForTest` directly (or duplicates the small subset needed) — no btree helper reuse since this lives in a different package. |
| 7 | `docs: mark code cleanup 1.6 complete` | Flips the 1.6 row in `code_cleanup_stage_1_overview.md` from `Not started` to `Complete`. No PATCH_NOTES entry (test-only). |

**Why this order:**

- Commit 1 isolated so the helpers compile + format correctly before
  any test depends on them. Cheap and clean revert if the helper
  shape needs to change.
- Commit 2 is the highest-value single area (room engine has zero
  current coverage and is exercised by every Phase 4c room script);
  ship it next so that any iteration on the helpers from commit 1
  surfaces immediately.
- Commit 3 is independent of the helpers for two of three conditions;
  intentionally cheap and isolated. Keeps the conditions_test.go
  growth tied to a single commit per Phase batch (matches the
  existing file's apparent convention — see
  `TestCondAllNewRegistered`).
- Commits 4 and 5 share `actions_test.go` and could collapse, but
  `actSummonCompanion` is the one test that touches three subsystems
  at once (aggro, room mob list, command queue) and is most likely
  to need a follow-up. Separate commit = isolated revert.
- Commit 6 lives in `internal/usercommands` because the test target
  is `give.go`, not a btree action. Cleanly different package from
  commits 1–5.
- Commit 7 is the conventional docs flip.

**Commit cadence rule:** after each commit,
`go build ./... && go vet ./... && go test ./...` must be clean. No
`--no-verify`. If a hook fails, fix and create a NEW commit (per repo
convention; never `--amend`).

**Commit message format:** Conventional commits, heredoc, trailing
`Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>`.

## Test Patterns

Three patterns are used across all 1.6 tests:

1. **Tiny per-test fixture, not monolith.** Each test calls one or
   two of `seedTestMob` / `seedTestUser` / `seedTestRoom` for exactly
   what it needs and `defer cleanup()`s. The helpers
   in `test_helpers_test.go` (commit 1) wrap
   `mobs.SeedMobsForTest` etc. to keep test bodies short. Counter-
   pattern to avoid: a single `seedAllRegistries()` mega-fixture per
   test (per the 1.6 budget, that's overkill and obscures what each
   test depends on).

2. **Deterministic IDs.** UserId 1, MobId 1, MobInstanceId 100,
   RoomId 1 by default — matches `hooks_test.go`'s convention so the
   two test packages stay readable together. Tests that need more
   instances pick obvious extensions (UserId 2, RoomId 2, etc.).

3. **No time-dependent assertions.** `time.Sleep`, wall-clock
   comparisons, and goroutine fan-out are out. Engine evaluation is
   synchronous; the tested actions either set state immediately or
   queue commands on a mob (which is observable via the mob's
   command queue, not via wall time). If a test wants to verify a
   delayed action, it skips — that's 1.8 territory.

The `actSummonCompanion` test (Item 5) needs one extra
consideration: `mobs.NewMobById` requires the mob spec to be present
in the registry. Test pre-seeds a minimal spec for the chosen
companion mob_id; `room.AddMob` is then implicitly called by the
action. Verify the resulting instance with `room.GetMobs(rooms.FindAll)`
and find the new instanceId by exclusion (it wasn't there before the
action ran).

## Verification & Rollout

**Per-commit verification:**
```bash
go build ./...
go vet ./...
go test ./internal/behaviortree/...    # commits 1–5
go test ./internal/usercommands/...    # commit 6
go test ./...                          # all commits
```

Any failure (other than the pre-existing
`TestRoom_AddTemporaryExit/duplicate_name_rejected`) is a hard block.

**After all commits:**
- `go test ./... -count=1` to defeat any stale cache.
- `go test ./internal/behaviortree/... -race` to surface any
  fixture-related data race (the room state double-checked locking
  test from Item 1 needs `-race` to be meaningful).
- Boot a local server, walk into a Phase 4c room (Sanctum Basin
  tutorial), run a `look` and an `ask` to confirm the room's tree
  still fires (sanity smoke; the 1.6 tests don't replace functional
  smoke).

**Pre-push:**
- All commits pass per-commit verification.
- `code_cleanup_stage_1_overview.md` Stage 1.6 row flipped to
  Complete (commit 7).
- `PATCH_NOTES.md` — no entry (test additions are internal, zero
  player impact).
- Memory file `project_rooms_package_audit_needed.md` left as-is —
  the failing rooms test is intentionally still failing and that
  memory file already captures the followup intent.

**Working-tree noise to ignore (do not stage):**
`.claude/settings.local.json`,
`internal/usercommands/_datafiles/feedback/*.txt`,
`Screenshot 2026-04-17 084513.png`.

**Branch strategy:**
- Branch: `feature/stage-1.6-test-coverage` off `development`.
- Merge `--no-ff` into `development`.
- Do not push to origin until user OKs.

**Rollback:**
- Each commit independently revertable. Commit 1 (helpers) is the
  base for commits 2/4/5; commits 3 and 6 are independent of the
  helpers. A revert of commit 1 forces a revert of 2/4/5 — accept
  that coupling rather than duplicate fixture code per test file.
- Commits 6 and 7 are independent of all earlier commits.

## Risk Register

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| A 1.6 test surfaces a real bug in a Phase 4c action (e.g., `mob_in_room` returns Success for a nil mob in the list) → blocks the test commit | Medium | Scope-creep policy. Clear bug → preceding `fix:` commit; ambiguous → pause and ask. Spec budget assumes zero defects; one defect adds maybe 30 min. |
| `internal/usercommands/give_test.go` setup is more involved than expected (give.go pulls in actions, events, items, mobs, rooms, users, questengine) | Medium-High | Use `seedAllRegistries()`-style approach scoped to give_test.go (parallel pattern, not import). If the test ends up taking >1h to wire, drop to a smaller-scope test that calls the inner question-engine notify branch directly (still proves the architecture, less coverage). Decision point: 60-min wall-clock cap on commit 6's setup. |
| `Room.SendText` doesn't have a tap point for assertions, forcing a real connection mock | Medium | The existing `hooks_test.go` does NOT mock connections; it relies on the absence of a connection meaning SendText quietly drops. For action tests, we either (a) verify the action's Success/Failure return is enough, or (b) add a tap via the existing `users.UserRecord.SendText` indirection. Decide at execution; default to (a) where possible. |
| `mobs.NewMobById` has hidden side effects (e.g., reads from disk for default loadout) that don't work in test fixtures | Low-Medium | The companion test (Item 5) is the only one calling `NewMobById`. If it pulls disk data, the test seeds a minimal spec via `SeedMobsForTest` and verifies no further loading is attempted. If a true block, the test asserts only the `room.AddMob` invocation + aggro state, not the spawn lifecycle. |
| `EnsureRoomBTreeState`'s sync test (Item 1, last test) is racy in a way that doesn't reproduce on Windows | Low | Use `-race` on the test target. If still flaky, mark as `t.Skip("race-mode only")` with a clear reason and log to memory file. |
| Fold of commit 5 into commit 4 happens silently and the rationale is lost | Low | Spec calls out the fold decision explicitly; commit message must note "folded test 5 into commit 4 because trivial" if that decision is made at execution. |
| Pre-existing `TestRoom_AddTemporaryExit` failure starts to be confused with a 1.6 regression | Low | Verify script grep-excludes `TestRoom_AddTemporaryExit` from the failure-classification step. Memory file `project_rooms_package_audit_needed.md` documents the intentional skip. |

## Success Criteria

- 23 new tests across 4 files; all pass on the merged branch.
- `go build ./...`, `go vet ./...`, `go test ./...` clean after every
  commit (with the documented `TestRoom_AddTemporaryExit` failure
  unchanged).
- `go test ./internal/behaviortree/... -race` passes.
- `code_cleanup_stage_1_overview.md` Stage 1.6 row flipped to
  Complete.
- No production code modified. Any `fix:` commit added during
  execution is documented in the merge commit body.
- `feature/stage-1.6-test-coverage` merged `--no-ff` into
  `development`; not pushed to origin until user OK.
- Quest-engine vs btree-player_give handoff is locked in by
  `TestGive_QuestEngineInterceptsBeforeBtreePlayerGive`. Future
  changes to `give.go`'s ordering surface as a test failure rather
  than a player-reported quest break.
- No player-visible behavior change (this is test-only).
