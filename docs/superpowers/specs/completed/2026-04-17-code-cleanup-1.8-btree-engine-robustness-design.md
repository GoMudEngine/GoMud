# Code Cleanup 1.8: Behavior Tree Engine Robustness — Design Spec

## Goal

Targeted hardening pass on the behavior tree engine after heavy use in
Phase 4b/4c. Three robustness gaps identified in the Stage 1 overview's
1.8 section get real fixes: a panic-safe wrapper around `QueueDelayed`
closures (so a destroyed mob/room/user doesn't take the engine down when
its delayed action fires), an `EvictRoomBTreeState` API (so a future
rooms-package pass can break the `roomStates` map's permanent-retention
contract without touching the engine), and a documented "this is
correct" note on the `noTree` / `noRoomTree` negative cache (no
hot-reload, no stale-cache bug). One investigation item — tree parse
error visibility — is verified clean at read-time and closed with zero
code change.

This is the **final substage of Stage 1**. No new features; no
player-facing change unless `QueueDelayed` is currently panicking in
production (in which case commit 2 becomes a `PATCH_NOTES` candidate at
execution time). The deliverable is a smaller engine surface, fewer
hidden crash paths, and one new API ready for the rooms-package follow-up.

## Scope

**In scope (4 investigation areas with locked dispositions):**

| # | Area | Disposition |
|---|------|-------------|
| 1 | `QueueDelayed` closures dereferencing dead mobs/rooms/users | **REAL FIX.** Add a `safeExecuteDelayed(name, fn)` helper that recovers panics + logs to `mudlog.Error`. Apply at the engine level (inside `DrainQueue`) so all current and future call sites get coverage automatically. Tests: 2 in `room_engine_test.go`. |
| 2 | Negative cache (`noTree` / `noRoomTree`) ever stale | **DEFER (no code change).** No hot-reload support today; the cache is correct as long as files don't change at runtime. Add a one-line comment + `TODO(hot-reload)` near the negative-cache accessors in `engine.go`. Memory file note: revisit when/if hot-reload lands. |
| 3 | Room state memory leak — destroyed rooms retain `BehaviorState` | **REAL FIX (partial).** Add `EvictRoomBTreeState(roomId int)` to `internal/behaviortree/room_state.go` with two in-package tests. Do **not** wire it into `internal/rooms/` from 1.8 — wire-up belongs to a future rooms-package pass and is already captured in `project_rooms_package_audit_needed.md`. |
| 4 | Tree parse errors — surfaced or silently swallowed? | **VERIFIED CLEAN (no code change).** Read-pass at spec time confirmed both `TryRoomBehavior` (`internal/behaviortree/helpers.go:102`) and `TryMobBehavior` (`internal/behaviortree/helpers.go:151`) log parse failures at `mudlog.Error` with the room/mob id and the underlying error. The 1.6 test `TestTryRoomBehavior_LoadParseError_FailureNotCached` already exercises one parse-error branch. Memory file note records the verification result. |

**Out of scope:**

- Wiring `EvictRoomBTreeState` into `internal/rooms/InstanceRegistry.Remove` (or wherever ephemeral-zone room teardown happens). The cross-reference is already captured in `project_rooms_package_audit_needed.md`; the wire-up is a rooms-package concern, not a btree concern.
- Hot-reload of behavior tree YAML on disk change. The negative cache is correct only because files are static; if hot-reload is ever added, the negative cache invalidation strategy is the bigger problem.
- Mob-level `EnsureBTreeState` eviction. Mob instance state lives on the mob struct itself (per `EnsureBTreeState` in `internal/behaviortree/helpers.go`) and is collected when the mob instance is collected — no map-level leak. Only the room-state map leaks.
- Behavior tree parser changes, action/condition lookup table changes, decorator semantics. The compile path in `loader.go` is left exactly as-is.
- Pre-existing baseline failure `internal/rooms/TestRoom_AddTemporaryExit/duplicate_name_rejected` — still intentional, still tracked in `project_rooms_package_audit_needed.md`, still must not be "fixed" during 1.8.

## Decisions Locked During Brainstorming

- **Item 1 (delayed action queue) is a real fix.** `Engine.QueueDelayed` (`internal/behaviortree/engine.go:130`) accepts arbitrary closures, and `DrainQueue` calls them with no recovery. The two known call sites both live in `internal/behaviortree/actions.go` (lines 111 and 131) and both close over an `EvalContext` whose `MobId` / `InstanceId` / `RoomId` may refer to entities that have died between queue time and execute time. A nil deref inside any one of those closures kills the entire round tick. The fix is panic-recovery at the engine level (one helper, applied uniformly inside `DrainQueue`) rather than per-call-site nil guards — the closures are user-supplied through `actions.go` and through any future action that wants delayed dispatch, so the engine is the right enforcement boundary.

- **Item 2 (negative cache) defers — no code change beyond a comment.** The negative cache is keyed on `mobId` / `roomId`; entries are only set when `os.Stat` returns an error or `LoadTree` fails. Both calls happen at lazy-load time inside `TryMobBehavior` / `TryRoomBehavior` (helpers.go). Once set, an entry only clears on a successful subsequent `LoadTree` / `LoadRoomTree`. With no hot-reload, the file-not-on-disk and parse-error states are stable for the process lifetime, so the cache is correct. Adding invalidation today is dead code. The TODO comment makes the assumption explicit for future hot-reload work.

- **Item 3 (room state leak) gets the API but not the wire-up.** `roomStates` (in `room_state.go`) grows monotonically. With ephemeral zones (Sable portal instances) destroying rooms, the `BehaviorState` for those room ids is leaked. The fix is two halves: (a) the eviction API on the btree side, (b) calling it from the rooms-package teardown path. 1.8 ships only (a) because (b) belongs to a rooms-package audit pass that's already on deck (per `project_rooms_package_audit_needed.md`, updated 2026-04-17 to call out the eviction wire-up specifically). Shipping (a) without (b) is correct: the API exists, is tested, and is ready when the rooms pass picks it up; in the meantime nothing changes (no caller).

- **Item 4 (parse error visibility) is verified clean.** Read-pass during spec authoring confirmed both `TryMobBehavior` and `TryRoomBehavior` already emit `mudlog.Error` on parse failure. The 1.6 test `TestTryRoomBehavior_LoadParseError_FailureNotCached` exercises one branch and locks the "no negative-cache write on parse error" contract. No additional code or tests needed.

- **Engine-level panic recovery, not per-closure nil guards.** Considered: rewriting both `actions.go` call sites to nil-check `mob.GetInstance(InstanceId)` / `users.GetByUserId(UserId)` / `rooms.LoadRoom(RoomId)` inside each closure before dereferencing. Rejected: it's per-closure boilerplate, doesn't cover unknown future closures, and doesn't help if the deref panic lives deeper in an action function than the closure body. The engine-level wrapper is one place, covers everything, and matches the existing recovery pattern in `integrations/discord/client.go:131`, `internal/llm/client.go:32`, `internal/inputhandlers/systemcommands.go:141`, `internal/questengine/engine.go:168` (`defer recover()` + `mudlog.Error`).

- **`safeExecuteDelayed` is unexported.** It's a single-package helper. If a future package wants similar protection, it can either move the helper to `mudlog` (where the recovery pattern already arguably belongs) or copy the small body. Don't pre-export.

- **Eviction is no-op on missing key.** Standard Go `delete(map, key)` semantics. No error return, no panic. Tests lock this contract.

- **`EvictRoomBTreeState` takes the write lock unconditionally.** A double-check (RLock → check → Unlock → return early if absent) saves nothing; eviction is rare and the contended path is `EnsureRoomBTreeState`, which already does the read-fast-path correctly.

- **Test file: append to `internal/behaviortree/room_engine_test.go`.** Created in 1.6, already contains the `EnsureRoomBTreeState` tests, already has the `clearEngineRoomState` / `clearRoomStates` helpers and `seedTestRoom` access. The 4 new tests (2 for the `QueueDelayed` wrapper, 2 for `EvictRoomBTreeState`) fit as siblings. Naming: `TestQueueDelayed_*`, `TestEvictRoomBTreeState_*`.

- **Scope-creep policy carries over.** Clear bug found while writing the wrapper or eviction → preceding `fix:` commit. Ambiguous → pause and ask. Dead code spotted incidentally → `chore:` removal commit.

- **No `PATCH_NOTES.md` entry by default.** Internal robustness, zero player-visible change unless `QueueDelayed` is currently panicking in production. If logs from any current deployment show a "delayed action panicked" trace, commit 2 becomes a `PATCH_NOTES` candidate (re-decide at execution).

## Test Inventory

All 4 tests append to `internal/behaviortree/room_engine_test.go`.

| Test name | Verifies |
|-----------|---------|
| `TestQueueDelayed_RecoversFromPanic` | Submit a closure that panics. `DrainQueue` returns normally; the engine remains usable for subsequent `QueueDelayed`/`DrainQueue` cycles. The panic is captured (no goroutine death). Verified by submitting a second non-panicking closure after the panicking one and confirming the second runs. |
| `TestQueueDelayed_RunsSucceededClosure` | Sanity check: a normal closure submitted via `QueueDelayed(0, fn)` runs exactly once when `DrainQueue` is called. Locks the happy-path contract so the wrapper doesn't accidentally swallow non-panic returns. |
| `TestEvictRoomBTreeState_RemovesEntry` | Seed state via `EnsureRoomBTreeState(roomId)`; capture the pointer. Call `EvictRoomBTreeState(roomId)`. Call `EnsureRoomBTreeState(roomId)` again; verify the returned pointer is **different** from the captured one (proves the prior entry was actually removed, not just overwritten). |
| `TestEvictRoomBTreeState_NoOpOnMissing` | Call `EvictRoomBTreeState(roomId)` for a roomId that was never seeded. No panic, no error, no side effect. Subsequent `EnsureRoomBTreeState(roomId)` still works normally. |

**Total: 4 new tests, 1 file appended (no new files).**

## Architecture

**Execution order — 4 commits on `feature/stage-1.8-btree-engine-robustness` off `development`:**

| # | Commit | Description |
|---|--------|-------------|
| 1 | `chore(behaviortree): document negative cache hot-reload assumption` | Adds a one-paragraph comment near `HasNoTree` / `HasNoRoomTree` (or the first negative-cache accessor in `engine.go`) explaining that the cache is correct only because behavior tree files don't change at runtime, with a `// TODO(hot-reload): bust cache on file change if/when hot-reload is added`. Also appends to memory file `project_btree_engine_audit_findings.md` (created in this commit if absent) noting the deferred status of cache invalidation. No tests, no code logic change. |
| 2 | `fix(behaviortree): panic-safe DrainQueue execution` | Adds an unexported `safeExecuteDelayed(name string, fn func())` helper in `engine.go` (or a new `engine_safety.go` if `engine.go` is getting too long — decide at execution). Modifies `DrainQueue` to wrap each ready action in the helper. Adds `TestQueueDelayed_RecoversFromPanic` and `TestQueueDelayed_RunsSucceededClosure` to `room_engine_test.go`. Verifies the existing two call sites (`actions.go:111`, `actions.go:131`) need no modification — they keep submitting raw closures, the engine wraps. |
| 3 | `feat(behaviortree): add EvictRoomBTreeState API` | Adds `EvictRoomBTreeState(roomId int)` to `room_state.go` under `roomStateMu` write lock. No-op on missing key. Adds `TestEvictRoomBTreeState_RemovesEntry` and `TestEvictRoomBTreeState_NoOpOnMissing` to `room_engine_test.go`. Confirms zero current callers (intentional — wire-up belongs to future rooms-package pass). Memory file `project_rooms_package_audit_needed.md` updated to note the new API is now available for the rooms-package wire-up. |
| 4 | `docs: mark code cleanup 1.8 complete` | Flips the 1.8 row in `code_cleanup_stage_1_overview.md` from `Not started` to `Complete`. If the overview tracks Stage 1's overall completion state at the top of the file, flips that to `Complete` as well (1.8 is the final substage). Memory file `project_btree_engine_audit_findings.md` gets the verified-clean note for Item 4. No `PATCH_NOTES.md` entry. |

**Why this order:**

- Commit 1 is the cheapest documentation-only commit; doing it first lands the "we considered this" record before any logic change so reviewers see the full disposition picture from the first commit.
- Commit 2 is the only commit with a possible behavior change (panic recovery surface). Isolated so a revert exactly undoes the engine-level wrapper without touching the eviction API.
- Commit 3 is a pure addition (new function, new tests, no callers). Independent of commit 2 — could ship in either order, but after-2 keeps the engine-internals commit before the room-state API commit, matching dependency direction (engine is more foundational).
- Commit 4 is the conventional docs flip. Last because Stage 1's overall-complete flag depends on commit-3 success.

**Commit cadence rule:** after each commit, `go build ./... && go vet ./... && go test ./...` must be clean (with the documented `internal/rooms/TestRoom_AddTemporaryExit/duplicate_name_rejected` failure unchanged). No `--no-verify`. If a hook fails, fix and create a NEW commit.

**Commit message format:** Conventional commits, heredoc, trailing `Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>`.

## Engine-Robustness Patterns

Two patterns introduced by this substage; both follow conventions already established elsewhere in the codebase.

### Pattern 1 — `safeExecuteDelayed(name, fn)` (commit 2)

Mirror of the `defer recover()` + `mudlog.Error` pattern used in `integrations/discord/client.go:131`, `internal/llm/client.go:32`, `internal/inputhandlers/systemcommands.go:141`, and `internal/questengine/engine.go:168`. The helper is unexported, lives next to `DrainQueue`, and applies inside `DrainQueue`'s ready-loop so all current and future closures get coverage.

Contract:
- A panicking closure does not crash the engine, does not abort the remaining ready actions in the same `DrainQueue` call, and does not leak the panic up the call stack.
- A panic is logged via `mudlog.Error` with `name` (a static label like `"behaviortree.delayed_action"`) and the recovered value.
- Non-panicking closures execute exactly once with no behavior change.

Why engine-level (not per-closure):
- One enforcement point covers the two known call sites in `actions.go` and any future delayed action.
- Closures often reference entities (mob, user, room) whose nil-ness can change between queue time and execute time. A per-closure nil guard catches some derefs but not all (an action function may dereference a stale pointer two layers deep). Recovery is a categorical answer: any panic, any depth.

### Pattern 2 — `EvictRoomBTreeState(roomId int)` API contract (commit 3)

Mirror of standard map-key removal. The function:
- Acquires `roomStateMu.Lock()` (write lock — eviction is a write operation regardless of whether the key is present).
- Calls `delete(roomStates, roomId)`.
- Releases the lock.
- Returns nothing. No-op on missing key.

Contract:
- After eviction, the next `EnsureRoomBTreeState(roomId)` returns a freshly-allocated `*BehaviorState` (proved by `TestEvictRoomBTreeState_RemovesEntry`).
- Eviction of an unseeded roomId is safe and silent (proved by `TestEvictRoomBTreeState_NoOpOnMissing`).
- Concurrent eviction with `EnsureRoomBTreeState` is mutex-safe; the worst case is "evicted then immediately re-seeded by a concurrent ensure call" which is benign.

Caller policy (not enforced by 1.8):
- The intended caller is the rooms-package teardown for ephemeral zones (e.g., `InstanceRegistry.Remove` per `project_rooms_package_audit_needed.md`).
- Static / persistent rooms must never be evicted — their `BehaviorState` is meant to live for the process lifetime. Caller is responsible for distinguishing.
- Calling eviction on a still-active room is correctness-safe (the next event re-seeds) but wastes any accumulated state (cooldown counters, delay rounds, etc.).

## Verification & Rollout

**Per-commit verification:**
```bash
go build ./...
go vet ./...
go test ./internal/behaviortree/...
go test ./...
```

Any failure other than the pre-existing `internal/rooms/TestRoom_AddTemporaryExit/duplicate_name_rejected` is a hard block.

**After all commits:**
- `go test ./... -count=1` to defeat any stale cache.
- `go test ./internal/behaviortree/... -race` (where supported — environment may lack gcc; if so, document and skip).
- Local server smoke: boot, walk into a Phase 4c room with a delayed action (e.g., `mob_say` with `delay: 2`), trigger it, kill the mob mid-delay (admin command), confirm the round tick after the delay fires does not crash the server. If reproducing the kill-mid-delay is hard locally, settle for: trigger the delayed action, confirm normal-path delivery still works (sanity check that the wrapper doesn't accidentally swallow normal returns).

**Pre-push:**
- All commits pass per-commit verification.
- `code_cleanup_stage_1_overview.md` Stage 1.8 row flipped to Complete; Stage 1 overall-status flipped to Complete (commit 4).
- Memory file `project_btree_engine_audit_findings.md` carries the negative-cache deferral note (commit 1) and the parse-error verified-clean note (commit 4).
- Memory file `project_rooms_package_audit_needed.md` carries the `EvictRoomBTreeState`-now-available note (commit 3).
- `PATCH_NOTES.md` — no entry by default. Re-decide at execution if production logs show evidence of a `QueueDelayed` panic in the wild (commit 2 would then become a player-facing fix worth noting).

**Working-tree noise to ignore (do not stage):** `.claude/settings.local.json`, `internal/usercommands/_datafiles/feedback/*.txt`, `Screenshot 2026-04-17 084513.png`.

**Branch strategy:**
- Branch: `feature/stage-1.8-btree-engine-robustness` off `development`.
- Merge `--no-ff` into `development`.
- Do not push to origin until user OKs.

**Rollback:**
- Each commit independently revertable.
- Commit 2 (wrapper) and commit 3 (eviction API) have no inter-dependency — either can revert without affecting the other.
- Commit 1 (negative-cache comment) is documentation-only; revert is purely cosmetic.
- Commit 4 (docs flip) is documentation-only; revert is purely cosmetic.

## Risk Register

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| The `safeExecuteDelayed` wrapper masks a real bug — a closure that *should* panic in development now silently logs and continues | Low-Medium | The recovery logs at `mudlog.Error` with the `name` and the recovered value — bugs are visible in logs, not silently dropped. The pattern matches existing recovers in `discord/client.go`, `llm/client.go`, etc., which have not caused this problem in practice. If a developer wants strict-panic behavior in tests, they can call the closure directly rather than through `QueueDelayed`. |
| `EvictRoomBTreeState` is added but never wired up; the leak persists indefinitely | Medium | The leak is small (one `BehaviorState` struct per ever-instanced ephemeral room id) and not a 1.8 acceptance gate. The wire-up is captured in `project_rooms_package_audit_needed.md`; the rooms-package pass is the right place. 1.8 making the API available is the dependency lift. |
| A test calling `EnsureRoomBTreeState` from another package starts seeing different pointers because some test or hook called `EvictRoomBTreeState` mid-run | Low | Zero current callers of `EvictRoomBTreeState`. The 1.6 test `TestEnsureRoomBTreeState_PersistsAcrossCalls` is unaffected (it doesn't evict). The new eviction tests use the existing `clearRoomStates(t, roomIds...)` defer pattern to clean up after themselves. |
| Wrapper changes the timing of `DrainQueue` enough to cause a flaky test elsewhere | Low | Wrapper adds one `defer`/`recover` per closure — nanosecond cost, not observable. No existing test asserts on `DrainQueue` wall time. |
| Tree parse error verification (Item 4) was based on a current snapshot; a regression could re-introduce silent swallowing later | Low | The 1.6 test `TestTryRoomBehavior_LoadParseError_FailureNotCached` exercises the parse-error path and asserts the no-negative-cache contract. If a future change silently swallows the error, that test still catches the contract violation. Adding a log-assertion test is over-engineering for 1.8's budget. |
| Pre-existing `TestRoom_AddTemporaryExit/duplicate_name_rejected` failure starts to be confused with a 1.8 regression | Low | Verify script grep-excludes that test name from the failure-classification step. Memory file `project_rooms_package_audit_needed.md` already documents the intentional skip. |
| `roomStateMu` is held during `delete(roomStates, roomId)` while a concurrent `EnsureRoomBTreeState` is mid-double-check; data race or stale read | Low | The double-check pattern in `EnsureRoomBTreeState` already handles concurrent inserts safely. `delete` under the same write lock is the canonical safe pattern. `-race` test pass is the gate (where environment supports). |

## Success Criteria

- 4 new tests appended to `internal/behaviortree/room_engine_test.go`; all pass.
- `go build ./...`, `go vet ./...`, `go test ./...` clean after every commit (with the documented `TestRoom_AddTemporaryExit/duplicate_name_rejected` failure unchanged).
- `go test ./internal/behaviortree/... -race` passes (or documented as deferred if environment lacks gcc).
- `Engine.DrainQueue` is panic-safe: a panicking closure does not crash the round tick; the panic is logged at `mudlog.Error` with a stable label.
- `EvictRoomBTreeState(roomId int)` exists in `internal/behaviortree/room_state.go`, respects `roomStateMu`, and is no-op on missing key.
- `engine.go` carries an explicit comment documenting the negative-cache hot-reload assumption with a `TODO(hot-reload)` marker.
- Memory file `project_btree_engine_audit_findings.md` exists and records: (a) the negative-cache deferral, (b) the parse-error verified-clean result for Item 4.
- Memory file `project_rooms_package_audit_needed.md` updated to reference the new `EvictRoomBTreeState` API as the dependency for the future rooms-package wire-up.
- `code_cleanup_stage_1_overview.md` Stage 1.8 row flipped to `Complete`; if the overview tracks Stage 1's overall status, that is also flipped to `Complete` (1.8 is the final substage of Stage 1).
- `feature/stage-1.8-btree-engine-robustness` merged `--no-ff` into `development`; not pushed to origin until user OK.
- No production code modified outside the four commits described above. Any `fix:` commit added during execution per the scope-creep policy is documented in the merge commit body.
- No player-visible behavior change (or, if `QueueDelayed` is currently panicking in production, a `PATCH_NOTES` line added at execution time noting the crash-fix).
