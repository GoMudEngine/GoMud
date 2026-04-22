# Code Cleanup 1.8: Behavior Tree Engine Robustness — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Targeted hardening pass on the behavior tree engine after Phase 4b/4c. Three robustness gaps from the Stage 1 overview's 1.8 section get fixes: a panic-safe wrapper around `QueueDelayed` closures (Item 1, real fix), an `EvictRoomBTreeState(roomId int)` API on `room_state.go` ready for a future rooms-package wire-up (Item 3, API only), and a documented "this is correct" note on the `noTree` / `noRoomTree` negative cache (Item 2, deferred). One investigation item — tree parse error visibility (Item 4) — is verified clean at read-time and closes with zero code change. This is the **final substage of Stage 1**.

**Architecture:** 4 commits on `feature/stage-1.8-btree-engine-robustness`. Commit 1 is documentation-only (negative-cache hot-reload comment). Commit 2 is the only commit with possible behavior change (engine-level panic recovery in `DrainQueue`). Commit 3 is a pure addition (new function, new tests, no callers — wire-up is a future rooms-package concern). Commit 4 flips the 1.8 row in the Stage 1 overview. Each commit is independently revertable; commits 2 and 3 have no inter-dependency.

**Tech Stack:** Go 1.25. No new dependencies. Verification via `go build`/`go vet`/`go test ./...` after every commit, plus a `-race` pass on `internal/behaviortree/...` after the final commit (environment-aware: skip with a documented note if gcc isn't available).

**Spec:** `docs/superpowers/specs/2026-04-17-code-cleanup-1.8-btree-engine-robustness-design.md`

**Branch:** `feature/stage-1.8-btree-engine-robustness` off `development`.

---

## Task 0: Create feature branch

**Files:** none.

- [ ] **Step 1: Verify you're on `development` and clean**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git status
git branch --show-current
```

Expected: `development`. Working tree should be clean EXCEPT for the following known unrelated working-tree noise (carried over from 1.5/1.6/1.7 execution):

- `.claude/settings.local.json` — dirty
- `internal/usercommands/_datafiles/feedback/bugs.txt`, `suggestions.txt` — dirty
- `"Screenshot 2026-04-17 084513.png"` — untracked

These are **out of scope** for 1.8. Do NOT stage or commit them at any point in this plan. If `git status` shows anything else dirty, investigate before proceeding.

- [ ] **Step 2: Create feature branch**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git checkout -b feature/stage-1.8-btree-engine-robustness
```

Expected: `Switched to a new branch 'feature/stage-1.8-btree-engine-robustness'`.

- [ ] **Step 3: Baseline verification**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./...
```

Expected: clean build, clean vet, and all tests pass EXCEPT for one known pre-existing failure:

- `TestRoom_AddTemporaryExit/duplicate_name_rejected` in `internal/rooms/`

**This failure is known, pre-existing, and out of scope for 1.8.** It is documented in the memory file `project_rooms_package_audit_needed.md` and belongs to a future dedicated rooms-package pass. Do NOT attempt to fix it on this branch. Confirm before proceeding that it is the ONLY failing test. If any other test fails, STOP and investigate — the baseline is broken and must be green (aside from `TestRoom_AddTemporaryExit`) before you start.

Per-commit verification below treats that single failure as expected noise. If any commit causes additional test failures, STOP and revert.

---

## Scope Policy

This is a **targeted robustness substage**. The default disposition is "make exactly the changes described in the spec and nothing more." The four investigation items have **locked dispositions** (per the spec):

| Item | Disposition |
|------|-------------|
| 1 — `QueueDelayed` panic safety | **REAL FIX** in commit 2 (`safeExecuteDelayed` wrapper inside `DrainQueue` + 2 tests). |
| 2 — Negative cache (`noTree`/`noRoomTree`) staleness | **DEFER** in commit 1 (one-paragraph comment + `TODO(hot-reload)` near the negative-cache accessors; memory-file note). |
| 3 — Room state map leak (`roomStates` grows monotonically) | **REAL FIX (partial)** in commit 3 (`EvictRoomBTreeState(roomId int)` API + 2 tests; rooms-package wire-up is OUT of scope per the spec — captured in `project_rooms_package_audit_needed.md`). |
| 4 — Tree parse error visibility | **VERIFIED CLEAN** — both `helpers.go:102` (room) and `helpers.go:151` (mob) already log at `mudlog.Error` with id + underlying error. No code change. Memory-file note in commit 4. |

Do NOT touch anything else in `engine.go`, `room_state.go`, `helpers.go`, `actions.go`, or anywhere else in the package. If you find yourself wanting to "improve" anything outside the locked dispositions, STOP and re-read the spec.

**Carryover scope-creep policy** (from 1.5 and 1.6, applies if something surfaces unexpectedly during execution):

- **Clear bug** (an unambiguous defect found while writing the wrapper or the eviction API — e.g., `QueueDelayed` already nil-derefs a struct field at queue time, or `roomStateMu` is held wrong somewhere) → preceding `fix:` commit on the same branch, BEFORE the commit that would otherwise demonstrate the issue. Memory-file note in `project_btree_engine_audit_findings.md` (created in Task 1) under a `## Surfaced During 1.8 Execution` heading.
- **Ambiguous case** (test result unexpected but unclear whether the production code or the test is wrong) → pause and ask the user. Log to `project_btree_engine_audit_findings.md` under `## Pending Decision`.
- **Dead code spotted incidentally** (e.g., an unreferenced helper while reading `engine.go`) → `chore:` removal commit, separate from the feature commits. Same memory-file note convention.

**Do NOT spray defensive guards.** The engine-level wrapper covers panics categorically (Item 1) — adding per-closure nil checks is rejected by the spec.

**No `PATCH_NOTES.md` entry by default.** Internal robustness, zero player-visible change. **Re-decide at execution time** for commit 2 only: if production logs from any current deployment show evidence of a `QueueDelayed` panic in the wild (e.g., a "delayed action" stack trace), commit 2 becomes a player-facing crash fix and gets a `PATCH_NOTES.md` line at execution.

**Pre-existing baseline noise to ignore at every task boundary:**
- `.claude/settings.local.json`
- `internal/usercommands/_datafiles/feedback/*.txt`
- `Screenshot 2026-04-17 084513.png`
- `TestRoom_AddTemporaryExit/duplicate_name_rejected` test failure (per `project_rooms_package_audit_needed.md`)

---

## Engine-Robustness Patterns

Two patterns introduced by this substage. Both follow conventions already used elsewhere in the codebase. Copy-paste-ready snippets below.

### Pattern 1 — `safeExecuteDelayed(name, fn)` (commit 2)

Mirror of the `defer recover()` + `mudlog.Error` pattern used in `integrations/discord/client.go:131`, `internal/llm/client.go:32`, `internal/inputhandlers/systemcommands.go:141`, and `internal/questengine/engine.go:168`. The helper is **unexported**, lives in `engine.go` next to `DrainQueue`, and applies inside `DrainQueue`'s ready-loop so all current and future closures get coverage.

**Sketch (executor implements; do NOT paste-and-walk-away — read the helper imports first):**

```go
// safeExecuteDelayed runs a delayed-action closure with panic recovery so a
// single misbehaving action (e.g., one closing over a mob/user/room that has
// since been destroyed) does not crash the engine round tick. Panics are
// logged at mudlog.Error with the supplied name as the operation label.
func safeExecuteDelayed(name string, fn func()) {
    defer func() {
        if r := recover(); r != nil {
            mudlog.Error(name, "error", fmt.Sprintf("delayed action panicked: %v", r))
        }
    }()
    fn()
}
```

**Apply inside `DrainQueue`** (currently at `engine.go:142-160`). The existing loop is:

```go
for _, da := range ready {
    da.Action()
}
```

becomes:

```go
for _, da := range ready {
    safeExecuteDelayed("behaviortree.delayed_action", da.Action)
}
```

**Required imports check:** `engine.go` does NOT currently import `fmt` or `mudlog`. Add both. Match the import grouping style already used in the package (`helpers.go` has both — copy that block style).

**Contract:**
- A panicking closure does not crash the engine, does not abort the remaining ready actions in the same `DrainQueue` call, and does not leak the panic up the call stack.
- A panic is logged via `mudlog.Error` with `name` (a static label like `"behaviortree.delayed_action"`) and the recovered value.
- Non-panicking closures execute exactly once with no behavior change.
- The helper is unexported. If a future package wants similar protection, it can move the helper to `mudlog` (where the recovery pattern arguably belongs) or copy the small body — do NOT pre-export.

**Why engine-level (not per-closure):** one enforcement point covers the two known call sites in `actions.go` (lines 111 and 131) and any future delayed action. Per-closure nil guards are per-call-site boilerplate that don't help if the deref panic lives deeper than the closure body.

### Pattern 2 — `EvictRoomBTreeState(roomId int)` API (commit 3)

Standard map-key removal under the existing `roomStateMu` write lock. Lives in `room_state.go` immediately after `EnsureRoomBTreeState`.

**Sketch (executor uses verbatim; the body is genuinely this small):**

```go
// EvictRoomBTreeState removes the cached BehaviorState for the given room.
// No-op if the room has no cached state. The next EnsureRoomBTreeState call
// for this roomId will allocate a fresh BehaviorState.
//
// Intended caller: rooms-package teardown for ephemeral zone instances
// (e.g., InstanceRegistry.Remove for portal-spawned rooms). Static rooms
// must NOT be evicted — their state is meant to live for the process
// lifetime; eviction loses any accumulated state (cooldown counters,
// delay rounds, etc.).
//
// 1.8 ships only the API. Wire-up from the rooms package is captured in
// project_rooms_package_audit_needed.md and belongs to a future rooms-
// package audit pass.
func EvictRoomBTreeState(roomId int) {
    roomStateMu.Lock()
    defer roomStateMu.Unlock()
    delete(roomStates, roomId)
}
```

**Contract:**
- After eviction, the next `EnsureRoomBTreeState(roomId)` returns a freshly-allocated `*BehaviorState` (proved by `TestEvictRoomBTreeState_RemovesEntry`).
- Eviction of an unseeded roomId is safe and silent (proved by `TestEvictRoomBTreeState_NoOpOnMissing`).
- Concurrent eviction with `EnsureRoomBTreeState` is mutex-safe; the worst case is "evicted then immediately re-seeded by a concurrent ensure call" which is benign.
- Takes `roomStateMu.Lock()` unconditionally. A double-check (`RLock` → check → unlock → return early if absent) saves nothing; eviction is rare and the contended path is `EnsureRoomBTreeState`, which already does the read-fast-path correctly.

**Caller policy (NOT enforced by 1.8):** the caller (future rooms-package code) is responsible for distinguishing ephemeral-instance rooms from static rooms before calling. Calling eviction on a still-active room is correctness-safe (the next event re-seeds) but wastes any accumulated state.

---

## Task 1: `chore(behaviortree)`: document negative cache hot-reload assumption

Smallest commit. Documentation-only on the negative cache, plus the new memory file. No tests, no logic change. Lands first so reviewers see the deferred-disposition record before any logic change.

**Files:**
- Modify: `internal/behaviortree/engine.go` (add comment block above the first negative-cache accessor)
- Update directly (NOT via git): `~/.claude/projects/C--Users-Calabe-Davis-workspace-DOGMud/memory/project_btree_engine_audit_findings.md` (CREATE — this file does not exist yet)

**Complexity:** Trivial.

### Discovery

- [ ] **Step 1: Re-confirm the negative-cache layout in `engine.go`**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
sed -n '50,110p' internal/behaviortree/engine.go
```

Expected: four functions cluster the negative cache — `HasNoTree` (line ~52), `SetNoTree` (~63), and the room equivalents `HasNoRoomTree` (~92), `SetNoRoomTree` (~101). The mob-side `LoadTree` (~40) clears `delete(e.noTree, mobId)` on success; `LoadRoomTree` (~78) does the same for rooms.

- [ ] **Step 2: Pick the comment site**

The cleanest single anchor is **immediately above `HasNoTree`** (the first negative-cache accessor in the file). The comment documents the assumption once and applies to all four functions in the cluster. Do NOT scatter the comment across all four functions; do NOT add it inside `init()` or the struct definition.

### Implementation

- [ ] **Step 3: Add the comment block above `HasNoTree`**

Insert immediately above the existing `// HasNoTree reports ...` doc comment (so the new block is the first thing the reader sees when scrolling to the negative-cache cluster). Preserve the existing `// HasNoTree ...` doc comment unchanged below the new block.

Suggested wording (executor may rephrase, must keep the `TODO(hot-reload)` line verbatim so future grep finds it):

```go
// Negative cache (noTree / noRoomTree) — design note.
//
// The negative cache records mob/room ids whose behavior tree YAML does
// not exist on disk (or whose load failed at file-stat time). Once set,
// an entry only clears on a successful subsequent LoadTree / LoadRoomTree.
//
// This is correct ONLY because behavior tree files are static for the
// process lifetime — there is no hot-reload of YAML on disk change. If
// hot-reload is ever added, the negative cache becomes a stale-cache bug:
// a file appearing on disk after the negative entry is set will be
// invisible until the engine restarts.
//
// TODO(hot-reload): bust cache on file change if/when hot-reload is added.
```

- [ ] **Step 4: Create the memory file**

Create `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\project_btree_engine_audit_findings.md` with initial content along these lines (executor refines):

```markdown
# Behavior Tree Engine Audit Findings

Tracker for findings surfaced during Code Cleanup 1.8 (Behavior Tree Engine
Robustness) and any future btree-engine audit passes.

## Resolved

- **Item 1 — QueueDelayed panic safety.** Real fix shipped in 1.8 commit 2:
  added `safeExecuteDelayed` helper inside `DrainQueue` (`engine.go`).
  Panics in delayed-action closures (typically caused by closures over
  destroyed mobs/rooms/users) are now recovered and logged at
  `mudlog.Error` instead of crashing the engine round tick.

## Deferred

- **Item 2 — Negative cache (noTree/noRoomTree) staleness.** The cache is
  correct only because behavior tree files are static for the process
  lifetime. No hot-reload, no stale-cache bug today. Adding invalidation
  now is dead code. Documented in `engine.go` with a `TODO(hot-reload)`
  comment near `HasNoTree`. Revisit if/when hot-reload is added.

## Open / In-Progress

(none)

## Pending Decision

(none)
```

The Item 3 (eviction API) and Item 4 (parse-error verified clean) entries are added in Tasks 3 and 4 respectively; do NOT pre-populate them here.

**CRITICAL — memory file is NOT git-tracked.** Edit it directly with your file tool. Do NOT `git add` it. Do NOT include it in any commit's staged file list. Memory files live outside the repo at `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\` and are never committed to the DOGMud repo. (The 1.5 and 1.6 plans had this bug; do not repeat it.)

### Verify

- [ ] **Step 5: Build + vet + test**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./...
```

Expected: clean (with the documented `TestRoom_AddTemporaryExit/duplicate_name_rejected` baseline noise unchanged). The change is comment-only; no test should observe any difference.

### Commit

- [ ] **Step 6: Commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add internal/behaviortree/engine.go
git commit -m "$(cat <<'EOF'
chore(behaviortree): document negative cache hot-reload assumption

Add a design-note block comment above HasNoTree explaining that the
noTree / noRoomTree negative cache is correct ONLY because behavior
tree files are static for the process lifetime. If hot-reload is ever
added, the cache becomes a stale-cache bug. Includes a
TODO(hot-reload) marker so future hot-reload work surfaces this.

No logic change. Disposition for Item 2 of the 1.8 robustness audit;
the cache is correct today, deferral is intentional.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

Stage ONLY `internal/behaviortree/engine.go`. Do NOT stage the memory file (it lives outside the repo). Do NOT stage working-tree noise (`.claude/settings.local.json`, etc.).

---

## Task 2: `fix(behaviortree)`: panic-safe DrainQueue execution

The only commit in 1.8 with a possible behavior change. Adds the `safeExecuteDelayed` helper and wraps the ready-loop in `DrainQueue`. Adds 2 tests to `room_engine_test.go`.

**Files:**
- Modify: `internal/behaviortree/engine.go` (add helper + wrap loop in `DrainQueue`)
- Modify: `internal/behaviortree/room_engine_test.go` (append 2 tests)

**Complexity:** Low. Helper is ~6 lines; loop change is one-line; tests are short.

### Discovery

- [ ] **Step 1: Re-read `QueueDelayed` and `DrainQueue`**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
sed -n '130,160p' internal/behaviortree/engine.go
```

Confirm:
- `QueueDelayed(delay time.Duration, action func())` at line 131. Appends a `DelayedAction{ExecuteAt, Action}` to `e.queue` under `e.mu.Lock`.
- `DrainQueue()` at line 142. Splits `e.queue` into `ready` (executable now) and `remaining` (still-pending) under `e.mu.Lock`, releases the lock, then iterates `for _, da := range ready { da.Action() }`. The loop runs **outside** the engine mutex (good — and the wrapper preserves this).
- `DelayedAction` struct has fields `ExecuteAt time.Time` and `Action func()`. The closure field is `Action` (not `Fn`).

- [ ] **Step 2: Re-read the two existing call sites in `actions.go`**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
sed -n '95,140p' internal/behaviortree/actions.go
```

Confirm both `GetEngine().QueueDelayed(dur, func() { fn(params, evalCtx) })` (line 111) and `GetEngine().QueueDelayed(delay, func() { fn(params, evalCtx) })` (line 131) submit raw closures. **Neither call site needs modification** — the engine-level wrapper applies inside `DrainQueue` regardless of who submitted the closure.

- [ ] **Step 3: Confirm the imports needed in `engine.go`**

`engine.go` currently imports `sync` and `time` only. The wrapper needs:
- `fmt` (for `fmt.Sprintf` in the recover branch)
- `github.com/GoMudEngine/GoMud/internal/mudlog` (for `mudlog.Error`)

Match the import grouping style in `helpers.go` (stdlib first, blank line, then internal packages).

### Implementation

- [ ] **Step 4: Add the `safeExecuteDelayed` helper to `engine.go`**

Place the helper at the bottom of `engine.go` (after `DrainQueue`). Use the sketch from Pattern 1 above verbatim.

- [ ] **Step 5: Wrap the ready-loop in `DrainQueue`**

Change:

```go
for _, da := range ready {
    da.Action()
}
```

to:

```go
for _, da := range ready {
    safeExecuteDelayed("behaviortree.delayed_action", da.Action)
}
```

The `name` argument is a stable label so log searches grep for `"behaviortree.delayed_action"` to find every panic recovery in production logs. Do NOT vary it per-closure (we don't have a per-closure name available, and a stable label is more useful for log triage).

- [ ] **Step 6: Add the imports**

Add `fmt` and `github.com/GoMudEngine/GoMud/internal/mudlog` to the import block. Order to match the existing convention in `helpers.go`:

```go
import (
    "fmt"
    "sync"
    "time"

    "github.com/GoMudEngine/GoMud/internal/mudlog"
)
```

- [ ] **Step 7: Append 2 tests to `internal/behaviortree/room_engine_test.go`**

Match the existing test style in this file: `TestQueueDelayed_<Scenario>` naming, `t.Errorf`/`t.Fatalf` (no testify), and any cleanup goes in a `defer`. Place both tests at the end of the file, AFTER `TestEnsureRoomBTreeState_PersistsAcrossCalls`.

| Test | Setup | Act | Assert |
|---|---|---|---|
| `TestQueueDelayed_RecoversFromPanic` | `e := GetEngine()`. Track two booleans (`firstRan`, `secondRan` — declare outside the closures so the test can read them after `DrainQueue`). | `e.QueueDelayed(0, func() { firstRan = true; panic("intentional test panic") })`. Then `e.QueueDelayed(0, func() { secondRan = true })`. Then `e.DrainQueue()`. | `DrainQueue` returns normally (no panic propagated to test). `firstRan == true` (panicker actually started). `secondRan == true` (the post-panic closure still ran — proves the panic in closure #1 did NOT abort the rest of the loop). |
| `TestQueueDelayed_RunsSucceededClosure` | `e := GetEngine()`. Track `ran int = 0` and `var mu sync.Mutex`. | `e.QueueDelayed(0, func() { mu.Lock(); ran++; mu.Unlock() })`. Then `e.DrainQueue()`. | `ran == 1` (happy-path: closure ran exactly once, wrapper did not swallow normal return). |

**Test cleanup:** `DrainQueue` consumes `ready` entries from `e.queue`, so no leftover state remains in the queue after the test. No defer needed. If a test ever needs to clear leftover queue state, add a sibling helper `clearEngineQueue(t)` modeled on `clearEngineRoomState` at `room_engine_test.go:15`.

**Reference style:** match the existing `TestEnsureRoomBTreeState_PersistsAcrossCalls` test (line 213) for `t` usage and assertion idioms. The package does not import testify.

**Test isolation note:** the `Engine` is a global singleton. Other tests in this package call `QueueDelayed` indirectly through `actions.go` action functions — but only when an `ActionNode.Evaluate` runs through the delayed-action branch (`actions.go:88-138`). `actions_test.go` tests dispatch via `LookupAction` directly (per the 1.6 plan's Pattern E), bypassing the queue path entirely. Therefore the queue is empty between tests and the new tests don't need to drain stale entries before acting. If a future test starts using `QueueDelayed` via `ActionNode.Evaluate`, this assumption may break — note in the test file with a comment.

### Verify

- [ ] **Step 8: Build + vet + scoped test**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./internal/behaviortree/...
go vet ./internal/behaviortree/...
go test ./internal/behaviortree/... -run "TestQueueDelayed"
```

Expected: 2 new tests pass, all existing behaviortree tests still pass.

- [ ] **Step 9: Full project test sweep**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go test ./...
```

Expected: only the baseline `TestRoom_AddTemporaryExit/duplicate_name_rejected` failure. The wrapper adds nanoseconds per closure — no timing-sensitive test should observe any change.

- [ ] **Step 10: (Spot check) Confirm panic message format in logs**

Run the panic test verbosely to eyeball the log line shape:

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go test ./internal/behaviortree/... -run "TestQueueDelayed_RecoversFromPanic" -v
```

Expected: a log line containing `"delayed action panicked: intentional test panic"` with the `behaviortree.delayed_action` label. If the log format looks off (e.g., the recovered value isn't visible), tweak the `fmt.Sprintf` format and re-run. This is a one-time visual check; no automated assertion needed.

### Commit

- [ ] **Step 11: Commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add internal/behaviortree/engine.go internal/behaviortree/room_engine_test.go
git commit -m "$(cat <<'EOF'
fix(behaviortree): panic-safe DrainQueue execution

Add unexported safeExecuteDelayed(name, fn) helper in engine.go and
wrap each ready closure inside DrainQueue with it. A panic in any
delayed-action closure (typically caused by closures over destroyed
mobs/rooms/users — see the two call sites in actions.go:111 and
actions.go:131 that close over an EvalContext whose MobId/InstanceId/
RoomId may refer to entities that have died between queue time and
execute time) is now recovered and logged at mudlog.Error with the
"behaviortree.delayed_action" label, instead of crashing the engine
round tick.

Mirrors the defer-recover pattern already used in
integrations/discord/client.go, internal/llm/client.go,
internal/inputhandlers/systemcommands.go, and
internal/questengine/engine.go.

Tests in room_engine_test.go:
- TestQueueDelayed_RecoversFromPanic: a panicking closure does not
  crash DrainQueue or abort subsequent ready closures in the same
  call.
- TestQueueDelayed_RunsSucceededClosure: happy-path locks the
  contract that the wrapper does not swallow non-panic returns.

The two existing call sites in actions.go are unchanged — they keep
submitting raw closures, the engine wraps.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

Stage ONLY the two listed files. Do NOT stage working-tree noise.

**PATCH_NOTES decision point:** at execution time, check production logs (or recent dev logs) for any "delayed action panicked" trace or any unexplained `QueueDelayed`-related crash signature. If found, this commit ships a player-facing crash fix and warrants a `PATCH_NOTES.md` line. Default: no entry.

---

## Task 3: `feat(behaviortree)`: add EvictRoomBTreeState API

Pure addition. New function in `room_state.go`, two tests in `room_engine_test.go`. Zero callers (intentional — wire-up belongs to a future rooms-package pass).

**Files:**
- Modify: `internal/behaviortree/room_state.go` (add `EvictRoomBTreeState`)
- Modify: `internal/behaviortree/room_engine_test.go` (append 2 tests)
- Update directly (NOT via git): `~/.claude/projects/C--Users-Calabe-Davis-workspace-DOGMud/memory/project_rooms_package_audit_needed.md` (append a note that the API is now available)

**Complexity:** Trivial.

### Discovery

- [ ] **Step 1: Re-read `room_state.go`**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
sed -n '1,28p' internal/behaviortree/room_state.go
```

Confirm:
- `roomStateMu sync.RWMutex` and `roomStates = make(map[int]*BehaviorState)` at the top of the file.
- `EnsureRoomBTreeState(roomId int) *BehaviorState` uses a double-checked lock (RLock fast-path → Lock slow-path → re-check → allocate).
- The file is short (28 lines). The new function fits cleanly at the bottom.

- [ ] **Step 2: Re-confirm zero current callers of any future eviction**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
grep -rn "EvictRoomBTreeState\|delete(roomStates" internal/ 2>/dev/null
```

Expected: zero matches (the function doesn't exist yet, and the test helper `clearRoomStates` at `room_engine_test.go:28` uses direct `delete(roomStates, id)` under `roomStateMu` — not via a public API). After the new function lands, the only matches should be the new function itself, the two new tests, and the test helper.

### Implementation

- [ ] **Step 3: Add `EvictRoomBTreeState` to `room_state.go`**

Append at the bottom of the file (after `EnsureRoomBTreeState`). Use the sketch from Pattern 2 above verbatim. The body is genuinely four lines (`Lock`, `defer Unlock`, `delete`).

- [ ] **Step 4: Append 2 tests to `internal/behaviortree/room_engine_test.go`**

Place after the `TestQueueDelayed_*` tests added in Task 2. Match the existing test style (no testify, `t.Errorf`/`t.Fatalf`).

| Test | Setup | Act | Assert |
|---|---|---|---|
| `TestEvictRoomBTreeState_RemovesEntry` | `const roomId = 99908`. `defer clearRoomStates(t, roomId)`. | `state1 := EnsureRoomBTreeState(roomId)` → capture pointer. `EvictRoomBTreeState(roomId)`. `state2 := EnsureRoomBTreeState(roomId)` → capture pointer. | `state1 != nil`. `state2 != nil`. `state1 != state2` (proves the prior entry was actually removed and a fresh one allocated, not just overwritten in place). |
| `TestEvictRoomBTreeState_NoOpOnMissing` | `const roomId = 99909`. `defer clearRoomStates(t, roomId)`. | `EvictRoomBTreeState(roomId)` (key was never seeded — must not panic). Then `state := EnsureRoomBTreeState(roomId)`. | No panic. `state != nil` (subsequent ensure works normally on the previously-evicted-empty key). |

**Pick fresh roomIds** to avoid cross-test contamination. The 1.6 tests in this file use IDs 99901-99907; pick 99908 and 99909 for these. Each test's `defer clearRoomStates(t, roomId)` ensures cleanup regardless of pass/fail.

**Reference style:** match `TestEnsureRoomBTreeState_PersistsAcrossCalls` (line 213) for the pointer-equality assertion idiom (`if state1 != state2 { t.Errorf("expected different pointer: got %p vs %p", state1, state2) }`).

- [ ] **Step 5: Append the API-availability note to `project_rooms_package_audit_needed.md`**

Open `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\project_rooms_package_audit_needed.md` directly with your file tool (NOT via git). Append (or update an existing) note under the relevant section indicating that the eviction API is now available:

```markdown
## EvictRoomBTreeState — Wire-up Pending (added 2026-04-17 by 1.8)

`internal/behaviortree.EvictRoomBTreeState(roomId int)` shipped in
Code Cleanup 1.8 commit 3. The API is no-op-on-missing-key and takes
the existing `roomStateMu` write lock.

**Wire-up TODO for the rooms-package audit pass:**
- Identify ephemeral-instance room teardown sites (likely
  `internal/rooms/InstanceRegistry.Remove` or wherever portal-spawned
  rooms are reaped).
- Call `behaviortree.EvictRoomBTreeState(roomId)` for each room being
  destroyed.
- Do NOT call eviction for static / persistent rooms — their
  BehaviorState is meant to live for the process lifetime; eviction
  loses accumulated state (cooldown counters, delay rounds, etc.).
- Caller is responsible for distinguishing ephemeral vs static rooms
  before calling.
```

If a similar section already exists in the file (from earlier 1.8-spec authoring), update the date and confirm the API name + signature are accurate. **CRITICAL — memory file is NOT git-tracked. Edit directly, do NOT `git add` it.**

### Verify

- [ ] **Step 6: Build + vet + scoped test**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./internal/behaviortree/...
go vet ./internal/behaviortree/...
go test ./internal/behaviortree/... -run "TestEvictRoomBTreeState"
```

Expected: 2 new tests pass.

- [ ] **Step 7: Full project test sweep**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go test ./...
```

Expected: only the baseline `TestRoom_AddTemporaryExit/duplicate_name_rejected` failure.

### Commit

- [ ] **Step 8: Commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add internal/behaviortree/room_state.go internal/behaviortree/room_engine_test.go
git commit -m "$(cat <<'EOF'
feat(behaviortree): add EvictRoomBTreeState API

Add EvictRoomBTreeState(roomId int) to room_state.go. Removes the
cached BehaviorState for the given room under the existing
roomStateMu write lock; no-op on missing key. The next
EnsureRoomBTreeState call for the same roomId allocates fresh state.

Intended caller: rooms-package teardown for ephemeral zone instances
(e.g., portal-spawned rooms reaped by InstanceRegistry.Remove). 1.8
ships only the API; the rooms-package wire-up is captured in
project_rooms_package_audit_needed.md and belongs to a future
rooms-package audit pass.

Tests in room_engine_test.go:
- TestEvictRoomBTreeState_RemovesEntry: post-eviction Ensure returns
  a different pointer (proves removal, not in-place overwrite).
- TestEvictRoomBTreeState_NoOpOnMissing: eviction of an unseeded id
  does not panic and does not break a subsequent Ensure call.

No current callers — by design.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

Stage ONLY the two listed files. Do NOT stage working-tree noise. Do NOT stage the memory file (it lives outside the repo).

---

## Task 4: `docs`: mark code cleanup 1.8 complete

Flip the 1.8 row in `code_cleanup_stage_1_overview.md` from `Not started` to `Complete`. Append the verified-clean note for Item 4 to `project_btree_engine_audit_findings.md`.

**Files:**
- Modify: `docs/superpowers/code_cleanup_stage_1_overview.md` (1-line status flip on the 1.8 row)
- Update directly (NOT via git): `~/.claude/projects/C--Users-Calabe-Davis-workspace-DOGMud/memory/project_btree_engine_audit_findings.md` (append the Item 4 note + the Item 3 wire-up-pending note)

**Complexity:** Trivial.

### Discovery

- [ ] **Step 1: Re-read the overview row layout**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
sed -n '12,30p' docs/superpowers/code_cleanup_stage_1_overview.md
```

Expected: line 24 (the 1.8 row) reads:

```markdown
| 1.8 | Behavior Tree Engine Robustness | 4h | Low | Not started |
```

There is **no top-level Stage 1 overall-completion flag** at the top of the file (verified during spec authoring) — just the per-row `Status` column. Therefore Task 4 only flips the 1.8 row; do NOT add or modify any other row, banner, or summary line. (The spec mentions a possible Stage 1 overall flag conditionally — confirmed absent in the actual file.)

### Update

- [ ] **Step 2: Flip the 1.8 status**

Change the 1.8 row's `Status` cell from `Not started` to `Complete`. Do NOT touch the Effort, Risk, or any other column. Do NOT touch other rows. Do NOT add a footer or summary.

After the change, line 24 reads:

```markdown
| 1.8 | Behavior Tree Engine Robustness | 4h | Low | Complete |
```

- [ ] **Step 3: Append the verified-clean and wire-up notes to `project_btree_engine_audit_findings.md`**

Edit the memory file directly (do NOT git-add). Add the following entries to bring it up to date with all four items:

Under `## Resolved`, ADD a new entry beneath the existing Item 1 entry:

```markdown
- **Item 3 — Room state map leak.** Real fix (partial) shipped in 1.8
  commit 3: added `EvictRoomBTreeState(roomId int)` to
  `room_state.go`. API is no-op-on-missing, takes `roomStateMu` write
  lock, and has 2 tests in `room_engine_test.go`. Wire-up from the
  rooms-package teardown path is OUT of scope for 1.8 and tracked in
  `project_rooms_package_audit_needed.md`.
```

Add a new top-level section:

```markdown
## Verified Clean (no code change)

- **Item 4 — Tree parse error visibility.** Read-pass at 1.8 spec time
  and re-verified at execution: both `TryRoomBehavior`
  (`internal/behaviortree/helpers.go:102`) and `TryMobBehavior`
  (`internal/behaviortree/helpers.go:151`) log parse failures at
  `mudlog.Error` with the room/mob id and the underlying error. The
  1.6 test `TestTryRoomBehavior_LoadParseError_FailureNotCached`
  exercises the room parse-error branch and locks the "no negative-
  cache write on parse error" contract. No additional code or tests
  needed.
```

If the memory file as authored in Task 1 is structured differently, adapt the heading wording to match — but the content must cover (a) Item 3 resolution, (b) Item 4 verified clean.

**CRITICAL — memory file is NOT git-tracked. Edit directly, do NOT `git add` it.**

### Verify

- [ ] **Step 4: Build + vet + test (docs-only should be no-op)**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./...
```

Expected: unchanged from Task 3 baseline (only the documented `TestRoom_AddTemporaryExit/duplicate_name_rejected` failure).

### Commit

- [ ] **Step 5: Commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add docs/superpowers/code_cleanup_stage_1_overview.md
git commit -m "$(cat <<'EOF'
docs: mark code cleanup 1.8 complete

Flip Stage 1.8 status to Complete in code_cleanup_stage_1_overview.md.

1.8 (Behavior Tree Engine Robustness) shipped:
- Item 1: panic-safe DrainQueue (commit 2 — safeExecuteDelayed wrapper)
- Item 2: negative-cache hot-reload assumption documented (commit 1)
- Item 3: EvictRoomBTreeState API added (commit 3 — wire-up pending in
  the future rooms-package audit pass)
- Item 4: tree parse error visibility verified clean — no code change

4 new tests appended to internal/behaviortree/room_engine_test.go.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

Stage ONLY `docs/superpowers/code_cleanup_stage_1_overview.md`. Do NOT stage the memory file (lives outside repo). Do NOT stage working-tree noise.

No `PATCH_NOTES.md` entry by default (re-decided at Task 2 execution time per the policy in the Scope section above).

---

## Task 5: Final verification + merge

Confirm branch shape, run full test sweep, race-mode pass on behaviortree (with environment fallback), then merge `--no-ff` into `development`. Do NOT push.

- [ ] **Step 1: Confirm branch commit count and order**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git log --oneline feature/stage-1.8-btree-engine-robustness ^development
```

Expected: 4 commits in this order (newest first):

1. `docs: mark code cleanup 1.8 complete`
2. `feat(behaviortree): add EvictRoomBTreeState API`
3. `fix(behaviortree): panic-safe DrainQueue execution`
4. `chore(behaviortree): document negative cache hot-reload assumption`

If a `fix:` or `chore:` precursor was added during execution per the scope-creep policy, expect more — document the actual count and the reason in the eventual merge commit body.

- [ ] **Step 2: Verify branch diff shape**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git diff --stat development...feature/stage-1.8-btree-engine-robustness
```

Expected files changed:
- `internal/behaviortree/engine.go` — modified (comment block in commit 1, helper + loop wrap in commit 2)
- `internal/behaviortree/room_state.go` — modified (EvictRoomBTreeState in commit 3)
- `internal/behaviortree/room_engine_test.go` — modified (4 tests appended across commits 2 and 3)
- `docs/superpowers/code_cleanup_stage_1_overview.md` — 1-line status flip (commit 4)

NO other production source files (`*.go` outside `*_test.go`) should appear. NO new files should be created in the repo.

If `.claude/settings.local.json`, `feedback/*.txt`, or `Screenshot*.png` show up: STOP, investigate, `git restore --staged` them.

If memory files (`project_btree_engine_audit_findings.md`, `project_rooms_package_audit_needed.md`) show up: STOP — those live outside the repo at `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\` and must NEVER appear in `git diff`. If they do, your working tree has the wrong file paths and you've accidentally created in-repo copies.

- [ ] **Step 3: Final whole-project verification**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./... -count=1
```

Expected: clean. The single pre-existing `TestRoom_AddTemporaryExit/duplicate_name_rejected` failure remains (baseline noise per `project_rooms_package_audit_needed.md`). No other test regressions. `-count=1` defeats stale cache.

- [ ] **Step 4: Race-mode pass on behaviortree (environment-aware)**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go test ./internal/behaviortree/... -race
```

Expected: clean. The new `TestEvictRoomBTreeState_*` tests and the existing `TestEnsureRoomBTreeState_PersistsAcrossCalls` concurrent sub-case are the meaningful coverage under `-race`.

**Environment fallback (carryover from 1.6 precedent):** `-race` requires CGO and a working gcc on Windows. If the run fails with a `cgo: C compiler "gcc" not found` style error (or equivalent), document the skip in the merge commit body — do NOT block on it. The wrapper's wrap site is single-goroutine (`DrainQueue` runs the ready-loop on the caller's goroutine), and `EvictRoomBTreeState` uses the same `roomStateMu` write lock that `EnsureRoomBTreeState` already proves race-safe under load. If `-race` is available, run it and record the result.

If `-race` does run and reports a real race (not a known flake), STOP — investigate and either add a `fix:` commit or skip the offending test with a memory-file note before merging.

- [ ] **Step 5: Confirm new test count**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go test ./internal/behaviortree/... -run "TestQueueDelayed|TestEvictRoomBTreeState" -v 2>&1 | grep -c "^=== RUN"
```

Expected: `4` (TestQueueDelayed_RecoversFromPanic, TestQueueDelayed_RunsSucceededClosure, TestEvictRoomBTreeState_RemovesEntry, TestEvictRoomBTreeState_NoOpOnMissing). Spec says 4 new tests; match exactly.

- [ ] **Step 6: Confirm memory files were edited (sanity check, not a git operation)**

Open both memory files with your file tool (do NOT shell out for this — the files live outside the repo at the path the spec references):

- `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\project_btree_engine_audit_findings.md` — should have all four items represented (Item 1 resolved, Item 2 deferred, Item 3 resolved, Item 4 verified clean).
- `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\project_rooms_package_audit_needed.md` — should have the `EvictRoomBTreeState` wire-up-pending section dated 2026-04-17.

If either is missing content, edit directly to add it. Do NOT git-add.

- [ ] **Step 7: (Optional) Smoke test — Phase 4c room behavior still fires**

Boot a local server, walk a player into a Phase 4c room with a delayed action (e.g., a `mob_say` or `mob_emote` with `delay: 2`), trigger it, confirm the round tick after the delay fires does not crash. If reproducing a kill-mid-delay scenario locally is hard, settle for: trigger a delayed action, confirm normal-path delivery still works (sanity-check that the wrapper doesn't accidentally swallow normal returns).

This is a final functional sanity check; the unit tests in commit 2 already lock the contract. Skip if local-boot is awkward.

---

## Merge to development (after user review)

Do NOT merge until the user has reviewed the branch. Once approved:

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git checkout development
git merge --no-ff feature/stage-1.8-btree-engine-robustness -m "$(cat <<'EOF'
merge: stage 1.8 behavior tree engine robustness

Final substage of Stage 1. Targeted hardening pass on the behavior
tree engine after Phase 4b/4c.

- Item 1 (panic-safe DrainQueue): added safeExecuteDelayed helper +
  wrapped the ready-loop. A panicking delayed-action closure
  (typically caused by closures over destroyed mobs/rooms/users) no
  longer crashes the engine round tick — the panic is logged at
  mudlog.Error with the "behaviortree.delayed_action" label and the
  remaining ready actions in the same DrainQueue call still execute.
  Mirrors the defer-recover pattern already used in discord/llm/
  inputhandlers/questengine.
- Item 2 (negative cache deferral): documented in engine.go with a
  TODO(hot-reload) marker. The noTree/noRoomTree cache is correct
  ONLY because behavior tree files are static; if hot-reload lands,
  the cache becomes a stale-cache bug.
- Item 3 (room state leak): added EvictRoomBTreeState(roomId int) to
  room_state.go. No-op on missing key, takes roomStateMu write lock.
  Wire-up from the rooms-package teardown is OUT of scope for 1.8 and
  tracked in project_rooms_package_audit_needed.md.
- Item 4 (parse error visibility): verified clean — both helpers.go
  load paths already log at mudlog.Error.

4 new tests appended to internal/behaviortree/room_engine_test.go.
No new files. No new dependencies. Verified by go build / go vet /
go test ./... clean (baseline TestRoom_AddTemporaryExit failure
unchanged) plus go test -race on internal/behaviortree/... (or a
documented gcc-unavailable skip per 1.6 precedent).

Stage 1 of the post-launch code cleanup is now complete (1.1 - 1.8).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

If a scope-creep `fix:` or `chore:` precursor was added during execution, mention it in the merge body (one bullet per extra commit).

If `-race` was skipped due to environment, add a final paragraph: `Race-mode test deferred: gcc not available in this environment. The wrapper site runs single-goroutine inside DrainQueue and the eviction API uses the same roomStateMu write lock that the existing EnsureRoomBTreeState concurrent test exercises.`

**Do NOT push to origin.** User reviews merge locally and pushes when ready.

---

## Done

After merge, Stage 1 of the post-launch code cleanup is complete (1.1 through 1.8). The behavior tree engine is panic-safe at its delayed-action surface, the room-state map has a teardown API ready for the future rooms-package audit, and the negative-cache hot-reload assumption is explicit. Time to move to Stage 2 (or whatever the next post-launch milestone is).
