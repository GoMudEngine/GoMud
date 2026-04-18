# Rooms Package Pass — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Targeted fix bundle for `internal/rooms/` after Stage 1 cleanup surfaced three concrete contract violations and an unfinished cleanup chain. Three known issues get real fixes — `Room.AddTemporaryExit`'s "always overwrite" behavior is replaced by a three-path rule (overwrite only ephemeral→ephemeral); the TTL-triggered instance cleanup chain is consolidated into the existing per-tick `CheckPortalTimers` (boot players → deregister → evict btree state → free ephemeral chunk); and a 30-minute light audit pass on the rest of `internal/rooms/*.go` looks for similar contract/stub patterns. Outcomes: a previously-failing test now passes (no test edit), instance/btree-state/room-map memory stops leaking forever in production, the dead `currentRound >= expiryRound { continue }` line is replaced with the real cleanup, and the dead refund branch in `actOpenInstancePortal` becomes reachable.

**Architecture:** 4–6 commits on `fix/rooms-package-pass` off `development`. Commit 1 fixes `AddTemporaryExit` (independent). Commit 2 is the only commit with substantive behavior change — adds the consolidated cleanup chain in `CheckPortalTimers`, deletes `CleanupEmptyInstances` and its `world.go:787` call site, and wires `behaviortree.EvictRoomBTreeState` into the chain via a startup-registered callback (because `internal/rooms` cannot import `internal/behaviortree` — see Risk Register below). Commit 3 is tests-only for the cleanup chain. Commit 4 is the time-capped audit pass. Commit 5 is memory-file cleanup (no git work). Commit 6 is conditional `fix:` per scope-creep policy.

**Tech Stack:** Go 1.25. No new dependencies. Verification via `go build`/`go vet`/`go test ./...` after every commit, plus a `-race` pass on `internal/rooms/...` after the final commit (environment-aware: skip with a documented note if gcc isn't available).

**Spec:** `docs/superpowers/specs/2026-04-18-rooms-package-pass-design.md`

**Branch:** `fix/rooms-package-pass` off `development`.

---

## Task 0: Create feature branch

**Files:** none.

- [ ] **Step 1: Verify you're on `development` and clean**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git status
git branch --show-current
```

Expected: `development`. Working tree should be clean EXCEPT for the following known unrelated working-tree noise (carried over from 1.5/1.6/1.7/1.8 execution):

- `.claude/settings.local.json` — dirty
- `internal/usercommands/_datafiles/feedback/bugs.txt`, `suggestions.txt` — dirty
- `"Screenshot 2026-04-17 084513.png"` — untracked

These are **out of scope** for this pass. Do NOT stage or commit them at any point in this plan. If `git status` shows anything else dirty, investigate before proceeding.

- [ ] **Step 2: Create feature branch**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git checkout -b fix/rooms-package-pass
```

Expected: `Switched to a new branch 'fix/rooms-package-pass'`.

- [ ] **Step 3: Baseline verification**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./...
```

Expected: clean build, clean vet, and all tests pass EXCEPT for one known pre-existing failure:

- `TestRoom_AddTemporaryExit/duplicate_name_rejected` in `internal/rooms/`

**This failure is the ONLY currently-failing test on `development`.** It is documented in the memory file `project_rooms_package_audit_needed.md`. **After Task 1 ships in this plan, the failure flips to PASS** (the new three-path rule rejects the non-ephemeral RoomId 100/200 inputs the existing test uses) — at that point there will be no documented baseline failures from `internal/rooms/`. Document this expected transition in your execution notes; reviewers may otherwise read it as "tests broken."

If any other test fails at Step 3, STOP and investigate — the baseline is broken and must be otherwise green before you start.

Per-commit verification below treats that single failure as expected noise UP TO commit 1 (Task 1). After commit 1, every test must pass.

---

## Scope Policy

This is a **targeted rooms-package fix bundle**. The default disposition is "make exactly the changes described in the spec and nothing more." The four areas have **locked dispositions** (per the spec):

| Area | Disposition |
|------|-------------|
| `Room.AddTemporaryExit` duplicate-rejection contract | **REAL FIX** in commit 1 (three-path rule + 3 new tests + 1 existing test flips PASS). |
| TTL-triggered instance cleanup chain | **REAL FIX** in commit 2 (boot → deregister → btree-evict → free-chunk inside `CheckPortalTimers`; delete `CleanupEmptyInstances` + its `world.go:787` call site; wire btree eviction via callback registered at startup). |
| Cleanup-chain test coverage | **REAL FIX** in commit 3 (4 new `TestCheckPortalTimers_*` tests). |
| Light audit pass on `internal/rooms/*.go` | **TIME-CAPPED AUDIT** in commit 4 (30 min wall-clock; Critical/High → fix in this commit; Low → log to memory file). |

Do NOT touch anything else in `internal/rooms/`. If you find yourself wanting to "improve" anything outside the locked dispositions, STOP and re-read the spec.

**Carryover scope-creep policy** (from 1.5/1.6/1.8, applies if something surfaces unexpectedly during execution):

- **Clear bug** (an unambiguous defect found while writing the cleanup chain — e.g., `Remove` already nil-derefs, or `roomsWithUsers` mutation racing somewhere we missed) → preceding `fix:` commit on the same branch, BEFORE the commit that would otherwise demonstrate the issue. Memory-file note in `project_rooms_package_audit_findings.md` (created in Task 4) under a `## Surfaced During rooms-pass Execution` heading. This is the optional Task 6 commit.
- **Ambiguous case** (test result unexpected but unclear whether the production code or the test is wrong) → pause and ask the user. Log to `project_rooms_package_audit_findings.md` under `## Pending Decision`.
- **Dead code spotted incidentally** (e.g., an unreferenced helper while reading `instances.go`) → `chore:` removal commit, separate from the feature commits. Same memory-file note convention.

**Audit-pass severity rule** (commit 4 only): Critical = real bug, gameplay-affecting → fix in this commit. High = real contract violation that callers depend on → fix in this commit. Low = inverted/awkward contract that's correctly used everywhere → log to memory file, do NOT fix.

**Do NOT spray defensive guards.** The TTL chain is the categorical fix for instance leaks (commit 2) — adding per-instance lifecycle hooks elsewhere is rejected by the spec.

**No `PATCH_NOTES.md` entry by default.** Internal infra; player-facing impact is implicit. **Re-decide at commit 2 execution time:** if a player has reported a stuck-in-instance-after-TTL symptom in the wild, commit 2 ships a player-facing fix and warrants a `PATCH_NOTES.md` line.

**Pre-existing baseline noise to ignore at every task boundary:**
- `.claude/settings.local.json`
- `internal/usercommands/_datafiles/feedback/*.txt`
- `Screenshot 2026-04-17 084513.png`
- `TestRoom_AddTemporaryExit/duplicate_name_rejected` test failure — **only until commit 1 lands**, after which it flips to PASS.

---

## Cleanup-Chain Patterns

Two patterns introduced by this pass; both follow conventions already established elsewhere in the codebase. Copy-paste-ready snippets below.

### Pattern 1 — TTL-triggered consolidated cleanup chain (commit 2)

Mirror of the standard MUD-wide-tick cleanup pattern (e.g., `RoomMaintenance`, `EphemeralRoomMaintenance` — both invoked from `world.go:784-785` under `util.LockMud()`). The chain runs inside `CheckPortalTimers` in the per-tick path that already iterates instances.

**Critical: the cleanup chain cannot directly call `behaviortree.EvictRoomBTreeState`.** `internal/rooms` cannot import `internal/behaviortree` — production behaviortree files (`actions_combat.go`, `actions_dialogue.go`, `actions_mob.go`, `actions_room.go`, `conditions_mob.go`, `conditions_player.go`, `helpers.go`) all import `internal/rooms`. A direct import creates a cycle. Wire-up uses a registered callback (mirroring `internal/mobai/reactor.go:130-137`'s `mobResolver`/`roomLoader` callback pattern).

**Sketch — callback registration in `instances.go`:**

```go
// btreeStateEvictor is set at startup by world.go to wire the
// rooms→behaviortree dependency direction without an import cycle.
// nil-safe: a nil evictor is a no-op (used in tests that don't care
// about btree state).
var btreeStateEvictor func(roomId int)

// SetBTreeStateEvictor registers the per-room btree state eviction
// callback. Called once at startup from world.go with
// behaviortree.EvictRoomBTreeState.
func SetBTreeStateEvictor(fn func(int)) {
    btreeStateEvictor = fn
}
```

**Sketch — startup wire-up in `world.go`:**

```go
// (in main / startup, before the world loop starts)
rooms.SetBTreeStateEvictor(behaviortree.EvictRoomBTreeState)
```

**Sketch — chain shape inside `CheckPortalTimers`:**

```go
// Phase A: under RLock, snapshot the expired instances.
ir.mu.RLock()
c := configs.GetConfig()
currentRound := util.GetRoundCount()
fiveMinRounds := uint64(c.Timing.MinutesToRounds(5))
oneMinRounds := uint64(c.Timing.MinutesToRounds(1))

var expired []*ZoneInstance
for _, inst := range ir.instances {
    if inst.PortalDuration == `` {
        continue
    }
    g := gametime.GetDate(inst.CreatedRound)
    expiryRound := g.AddPeriod(inst.PortalDuration)

    if currentRound >= expiryRound {
        expired = append(expired, inst)
        continue
    }
    // (existing 5-min / 1-min warning broadcast code stays here, unchanged)
    remainingRounds := expiryRound - currentRound
    var msg string
    switch remainingRounds {
    case fiveMinRounds:
        msg = `<ansi fg="yellow">The portal flickers — it won't hold much longer.</ansi>`
    case oneMinRounds:
        msg = `<ansi fg="red">The portal is barely a shimmer now. Leave soon or find your own way out.</ansi>`
    }
    if msg == `` {
        continue
    }
    for _, ephId := range inst.RoomIdMap {
        if room := LoadRoom(ephId); room != nil {
            room.SendText(msg)
        }
    }
}
ir.mu.RUnlock()

// Phase B: process each expired instance OUTSIDE the RLock.
const collapseMsg = `<ansi fg="red">The portal's shimmer collapses around you — the instance unravels.</ansi>`
for _, inst := range expired {
    // 1. Boot phase — O(1) populated-room check via roomsWithUsers.
    for _, ephId := range inst.RoomIdMap {
        if roomManager.roomsWithUsers[ephId] == 0 {
            continue
        }
        room := LoadRoom(ephId)
        if room == nil {
            continue
        }
        for _, userId := range room.GetPlayers() {
            if u := users.GetByUserId(userId); u != nil {
                u.SendText(collapseMsg)
            }
            // Move AFTER sending text — so the player sees the
            // message in the ephemeral room context, not after teleport.
            MoveToRoom(userId, inst.OverworldRoomId)
        }
    }

    // 2. Deregister phase (Remove takes its own ir.mu.Lock).
    ir.Remove(inst)

    // 3. Btree eviction phase — callback-mediated to avoid import cycle.
    if btreeStateEvictor != nil {
        for _, ephId := range inst.RoomIdMap {
            btreeStateEvictor(ephId)
        }
        // Also evict the entry-room id (it's already covered by RoomIdMap
        // because instances.go:325 stores it as a self-mapping; verify
        // during execution and skip the duplicate if so).
    }

    // 4. Ephemeral chunk free phase. Self-protects (returns []int{} if
    // any room has players or an active instance). After steps 1+2 both
    // gates are clear, so the chunk frees.
    TryEphemeralCleanup(inst.EntryRoomId)
}
```

**Imports required by `instances.go` after the chain lands:** add `"github.com/GoMudEngine/GoMud/internal/users"` (for `users.GetByUserId` in the boot phase). `LoadRoom` and `MoveToRoom` and `roomManager` are same-package; no new import for them.

**Contract:**
- A still-in-TTL instance is untouched (locked by `TestCheckPortalTimers_NotExpiredNoOp`).
- An expired empty instance: deregistered, btree state evicted, chunk freed.
- An expired populated instance: players booted to overworld with flavor, then same as empty.
- Concurrent player movement is impossible: the entire chain runs under `util.LockMud()` (the MUD-wide lock that also serializes `MoveToRoom` and all `roomsWithUsers` mutations).
- `btreeStateEvictor == nil` is safe (tests that don't register a callback simply don't evict — covered by `TestCheckPortalTimers_TtlExpiryEvictsBtreeState`'s setup which registers the callback explicitly).

**Replaces:**
- The dead `if currentRound >= expiryRound { continue // already expired — cleanup handled elsewhere }` line at `instances.go:183-184`. Removed in commit 2.
- The `CleanupEmptyInstances` function (`instances.go:122-157`) and its `world.go:787` call site. Removed in commit 2.

### Pattern 2 — `AddTemporaryExit` three-path rule (commit 1)

Modifies `Room.AddTemporaryExit` (`rooms.go:449`). The function:

1. Sets `t.SpawnedRound = util.GetRoundCount()` and lazy-initializes `r.ExitsTemp` (existing behavior, keep).
2. If `len(t.Title) == 0 { t.Title = exitName }` (existing behavior, keep).
3. Look up `existing, present := r.ExitsTemp[exitName]`.
4. If `!present`: store and return `true`.
5. If `present`: apply discriminator `if IsEphemeralRoomId(existing.RoomId) && IsEphemeralRoomId(t.RoomId) { overwrite, return true } else { no change, return false }`.

`IsEphemeralRoomId` (`ephemeral.go:154`) is in the same package. No new import.

**Sketch (executor uses verbatim modulo formatting):**

```go
// AddTemporaryExit adds a temporary exit by exitName. Returns true if
// the exit was stored, false if rejected.
//
// Three-path rule:
//   - No existing exit with this name → store, return true.
//   - Existing exit with this name AND both the existing and new
//     exits target ephemeral RoomIds (instance-portal upgrade case,
//     e.g. Sable opening a second portal of the same name to a new
//     ephemeral entry while the old portal is still in TTL) →
//     overwrite, return true.
//   - Existing exit with this name in any other case → leave the
//     existing exit alone, return false. Don't stomp a regular temp
//     exit with a portal, and don't stomp a portal with a non-portal.
func (r *Room) AddTemporaryExit(exitName string, t exit.TemporaryRoomExit) bool {

    t.SpawnedRound = util.GetRoundCount()

    if r.ExitsTemp == nil {
        r.ExitsTemp = make(map[string]exit.TemporaryRoomExit)
    }

    if len(t.Title) == 0 {
        t.Title = exitName
    }

    if existing, present := r.ExitsTemp[exitName]; present {
        if !(IsEphemeralRoomId(existing.RoomId) && IsEphemeralRoomId(t.RoomId)) {
            return false
        }
    }

    r.ExitsTemp[exitName] = t
    return true
}
```

**Contract:**
- New exit (no name conflict): added, returns `true`. (Locked by existing `TestRoom_AddTemporaryExit/add_first_exit`.)
- Duplicate name, both ephemeral: overwrite, returns `true`. (Locked by NEW `ephemeral_overwrite_allowed`.)
- Duplicate name, mixed ephemeral/non-ephemeral or both non-ephemeral: rejected, returns `false`. (Locked by existing `duplicate_name_rejected` (now passing) and NEW `mixed_*_rejected` tests.)

**Caller policy (NOT modified by this pass — verified per spec):**
- `actOpenInstancePortal` (`internal/behaviortree/actions_mob.go:311`) checks the bool and refunds gold on false (lines 317-321) — previously-dead refund branch becomes reachable.
- `mobcommands/portal.go:83` already checks the bool. Behavior correct under new rule.
- `cubegen.go:195`, `instances.go:343`, `actions_room.go:84`, `mobcommands/portal.go:93` ignore the return; all target ephemeral RoomIds on fresh ephemeral rooms (no name conflict possible). No change needed.
- `scripting/room_func.go:363` propagates the bool to scripts. No change needed.

---

## Task 1: `fix(rooms): AddTemporaryExit allows ephemeral-portal overwrite, rejects others`

Smallest substantive commit. Replaces the function body with the three-path rule, updates the doc comment, and adds three new sub-tests inside the existing `TestRoom_AddTemporaryExit`.

**Files:**
- Modify: `internal/rooms/rooms.go` (`AddTemporaryExit` at line 449 + its doc comment at line 446-448)
- Modify: `internal/rooms/rooms_test.go` (append 3 new sub-tests inside the existing `TestRoom_AddTemporaryExit` at line 1363)

**Complexity:** Low.

### Discovery

- [ ] **Step 1: Re-read the current `AddTemporaryExit` and its doc comment**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
sed -n '446,462p' internal/rooms/rooms.go
```

Confirm:
- Line 446-448: doc comment claims "(or replaces)" — outdated under the new rule.
- Line 449-461: function body unconditionally writes `r.ExitsTemp[exitName] = t` and returns `true`.
- `IsEphemeralRoomId` is in `ephemeral.go:154` — same package, no import needed.

- [ ] **Step 2: Re-read the existing `TestRoom_AddTemporaryExit` family**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
sed -n '1361,1383p' internal/rooms/rooms_test.go
```

Confirm:
- 3 existing sub-tests: `add first exit` (passes today), `duplicate name rejected` (FAILS today — uses non-ephemeral RoomId 100/200), `empty title defaults to exit name` (passes today).
- After this commit: 5 sub-tests pass (1 existing add + 1 existing duplicate-now-passes + 3 new + 1 existing empty-title).
- The existing tests use `exit.TemporaryRoomExit{RoomId: 100/200/300, ...}`. `IsEphemeralRoomId` returns true only when `roomId >= ephemeralRoomIdMinimum` (1,000,000 on 32-bit, 1,000,000,000 on 64-bit). All three existing IDs are non-ephemeral.

### Implementation

- [ ] **Step 3: Replace the function body with the three-path rule**

Use Pattern 2's sketch from the Cleanup-Chain Patterns section above. Update both the doc comment (lines 446-448) and the function body (lines 449-461). Preserve `t.SpawnedRound = util.GetRoundCount()`, the lazy `r.ExitsTemp` init, and the `t.Title` default — only the duplicate-handling logic is new.

- [ ] **Step 4: Append 3 new sub-tests inside the existing `TestRoom_AddTemporaryExit`**

Insert AFTER the existing `duplicate name rejected` sub-test (line 1376) and BEFORE the `empty title defaults to exit name` sub-test (line 1378). Match the existing testify-`assert` style and the `t.Run("...", func(t *testing.T) { ... })` shape.

To ensure the test uses an ephemeral RoomId, use `ephemeralRoomIdMinimum + N` directly (the constant is package-private but in scope for `_test.go` files in the same package).

| Sub-test name | Setup | Act | Assert |
|---|---|---|---|
| `ephemeral_overwrite_allowed` | Fresh `r2 := &Room{}`. Add an ephemeral-target exit first: `r2.AddTemporaryExit("eph_portal", exit.TemporaryRoomExit{RoomId: ephemeralRoomIdMinimum + 10, UserId: 1})`. | `added := r2.AddTemporaryExit("eph_portal", exit.TemporaryRoomExit{RoomId: ephemeralRoomIdMinimum + 20, UserId: 2})` | `assert.True(t, added)`. `assert.Equal(t, ephemeralRoomIdMinimum + 20, r2.ExitsTemp["eph_portal"].RoomId)` (proves overwrite happened). |
| `mixed_existing_ephemeral_new_normal_rejected` | Fresh `r3 := &Room{}`. Add an ephemeral-target exit first: `r3.AddTemporaryExit("eph_portal", exit.TemporaryRoomExit{RoomId: ephemeralRoomIdMinimum + 30, UserId: 1})`. | `added := r3.AddTemporaryExit("eph_portal", exit.TemporaryRoomExit{RoomId: 500, UserId: 2})` (non-ephemeral RoomId). | `assert.False(t, added)`. `assert.Equal(t, ephemeralRoomIdMinimum + 30, r3.ExitsTemp["eph_portal"].RoomId)` (proves no overwrite). |
| `mixed_existing_normal_new_ephemeral_rejected` | Fresh `r4 := &Room{}`. Add a non-ephemeral exit first: `r4.AddTemporaryExit("portal", exit.TemporaryRoomExit{RoomId: 600, UserId: 1})`. | `added := r4.AddTemporaryExit("portal", exit.TemporaryRoomExit{RoomId: ephemeralRoomIdMinimum + 40, UserId: 2})`. | `assert.False(t, added)`. `assert.Equal(t, 600, r4.ExitsTemp["portal"].RoomId)` (proves no overwrite). |

**Use fresh `Room` instances per sub-test** (`r2`, `r3`, `r4`) to avoid cross-test contamination from the shared `r := &Room{}` at line 1364. The existing sub-tests share `r` because they cooperate (`add first exit` then `duplicate name rejected`) — the new sub-tests have independent setups so they get fresh state.

The pre-existing `duplicate_name_rejected` sub-test at line 1372-1376 stays **unedited**. Its inputs are RoomId 100 (existing, non-ephemeral) and 200 (new, non-ephemeral); under the new rule the discriminator returns false, so the sub-test's `assert.False(t, added)` and `assert.Len(t, r.ExitsTemp, 1)` start passing automatically.

### Verify

- [ ] **Step 5: Scoped test**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go test ./internal/rooms/... -run "TestRoom_AddTemporaryExit" -v
```

Expected: 5 sub-tests pass — `add first exit`, `duplicate name rejected` (now PASSING), `ephemeral_overwrite_allowed`, `mixed_existing_ephemeral_new_normal_rejected`, `mixed_existing_normal_new_ephemeral_rejected`, `empty title defaults to exit name`. (6 total counting the parent test.)

If `duplicate_name_rejected` still FAILS, re-read the discriminator condition — the `!(IsEphemeralRoomId(existing.RoomId) && IsEphemeralRoomId(t.RoomId))` flip is the load-bearing parenthesization.

- [ ] **Step 6: Build + vet + full test sweep**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./...
```

Expected: clean. **No baseline failures remain after this commit** — the previously-failing `TestRoom_AddTemporaryExit/duplicate_name_rejected` is now passing; from this point on every commit must produce a fully green test sweep.

If any other test fails, STOP — investigate before proceeding. Likely candidate: `internal/rooms/integration_zones_test.go:136` calls `r.AddTemporaryExit("portal", tempExit)`. Read it; if its inputs are non-ephemeral and it asserts `added == true`, you've discovered a contract-violating test that needs either the test data fix (use ephemeral inputs) or a `fix:` precursor commit per scope-creep policy.

### Commit

- [ ] **Step 7: Commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add internal/rooms/rooms.go internal/rooms/rooms_test.go
git commit -m "$(cat <<'EOF'
fix(rooms): AddTemporaryExit allows ephemeral-portal overwrite, rejects others

Replace the unconditional-overwrite behavior with a three-path rule:
- No existing exit with this name → store, return true.
- Existing exit AND both existing+new target ephemeral RoomIds
  (instance-portal upgrade case) → overwrite, return true.
- Existing exit in any other case → no change, return false.

The previously-failing TestRoom_AddTemporaryExit/duplicate_name_rejected
(its inputs are non-ephemeral RoomId 100/200) now passes with no test
edit — the new contract returns false for that case, which the test
already asserts.

Three new sub-tests cover the new branches:
- ephemeral_overwrite_allowed
- mixed_existing_ephemeral_new_normal_rejected
- mixed_existing_normal_new_ephemeral_rejected

This makes the previously-dead refund branch in
internal/behaviortree/actions_mob.go actOpenInstancePortal (line 317-321)
reachable: a Sable customer requesting a duplicate-name portal targeting
a non-ephemeral RoomId now gets the gold refund as the contract intended.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

Stage ONLY the two listed files. Do NOT stage working-tree noise.

---

## Task 2: `feat(rooms): TTL-triggered instance cleanup chain in CheckPortalTimers`

The largest commit. Adds the consolidated cleanup chain inside `CheckPortalTimers`, wires the btree-eviction callback at startup (because of the import-cycle constraint), deletes `CleanupEmptyInstances`, deletes its `world.go:787` call site.

**Files:**
- Modify: `internal/rooms/instances.go` (rewrite `CheckPortalTimers` lines 161-207, delete `CleanupEmptyInstances` lines 122-157, add `btreeStateEvictor` var + `SetBTreeStateEvictor` setter, add `users` import)
- Modify: `internal/rooms/ephemeral.go` (update stale comment at line 173-174)
- Modify: `world.go` (delete the `CleanupEmptyInstances()` call at line 787 — this file is the per-tick worker loop, not the startup site)
- Modify: `main.go` (add `behaviortree` import; add `rooms.SetBTreeStateEvictor(behaviortree.EvictRoomBTreeState)` near `hooks.RegisterListeners()` at line 224 — the startup-registration cluster)

**Important file-distinction:** `world.go` (repo root) is the World worker loop — it owns the per-tick maintenance call sites (lines 783-788). `main.go` (repo root) is the startup site — it owns one-time registrations like `hooks.RegisterListeners()`. The cleanup-call deletion goes in `world.go`; the wire-up goes in `main.go`. They are two different files.

**Complexity:** Medium. Chain logic is mechanical; the callback wire-up needs a clean startup site that imports both packages.

### Discovery

- [ ] **Step 1: Re-read `CheckPortalTimers` in full**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
sed -n '159,207p' internal/rooms/instances.go
```

Confirm:
- Line 162-163: `ir.mu.RLock()` / `defer ir.mu.RUnlock()`.
- Line 165-167: short-circuit on empty.
- Line 169-173: pulls config, current round, and 5-min/1-min thresholds.
- Line 175-206: outer loop over `ir.instances`.
- Line 183-184: **DEAD `continue`** — the surgical replacement point.
- Line 187-205: existing warning broadcast logic (5-min / 1-min); **keep verbatim** in Phase A of the new chain.
- Line 162's `defer` cannot stand if Phase B mutates the registry — the chain needs to drop the RLock before calling `Remove(inst)` (which takes its own write lock) to avoid a deadlock between `RLock` and `Lock`.

- [ ] **Step 2: Re-read `Remove` and `TryEphemeralCleanup`**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
sed -n '82,101p' internal/rooms/instances.go
sed -n '158,210p' internal/rooms/ephemeral.go
```

Confirm:
- `Remove` (line 83-101): takes `ir.mu.Lock()`, filters the slice, deletes from `roomIndex`. Safe to call from outside Phase A's RLock once the RLock is released.
- `TryEphemeralCleanup` (ephemeral.go line 158-210): self-protects via "if any room has players OR is in `instanceRegistry.FindByRoomId`, return `[]int{}`" guards (lines 169-177). Calling after Phase A (boot) and Phase B (Remove) is safe — both gates are clear.

- [ ] **Step 3: Re-confirm the import-cycle constraint**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
grep -l "github.com/GoMudEngine/GoMud/internal/rooms" internal/behaviortree/*.go | grep -v _test.go
```

Expected: 7 files matched (`actions_combat.go`, `actions_dialogue.go`, `actions_mob.go`, `actions_room.go`, `conditions_mob.go`, `conditions_player.go`, `helpers.go`). This means `internal/rooms` **cannot** import `internal/behaviortree` — would form a cycle. The chain uses the callback pattern (Pattern 1, sketch above).

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
grep -l "github.com/GoMudEngine/GoMud/internal/behaviortree" internal/rooms/*.go
```

Expected: zero matches. Confirms `internal/rooms` doesn't currently import `behaviortree`. The new `btreeStateEvictor` var keeps it that way.

- [ ] **Step 4: Verify the startup wire-up site in `main.go`**

`world.go` is the World struct's worker loop (per-tick maintenance at lines 783-788). The actual startup-registration cluster is in `main.go` at the repo root, around line 224 where `hooks.RegisterListeners()` is called.

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
sed -n '220,232p' main.go
grep -n "behaviortree\|rooms\b" main.go | head -10
```

Confirm:
- `main.go:224` is `hooks.RegisterListeners()` — the canonical startup registration spot.
- `main.go:57` already imports `internal/rooms` ✓.
- `main.go` does NOT currently import `internal/behaviortree` — needs adding.

The wire-up goes adjacent to `hooks.RegisterListeners()` (immediately before or after — executor picks). Follow the existing import grouping style in `main.go` for the new behaviortree import.

### Implementation

#### Sub-step 2A — `instances.go` cleanup-chain rewrite

- [ ] **Step 5: Add `users` import to `instances.go`**

The boot phase needs `users.GetByUserId(userId).SendText(...)`. Current `instances.go` imports: `fmt`, `sync`, `configs`, `exit`, `gametime`, `util`. Add:

```go
"github.com/GoMudEngine/GoMud/internal/users"
```

Group placement: after `gametime`, before `util`, alphabetical within the internal-package group.

- [ ] **Step 6: Add the callback var and setter near the top of `instances.go`**

Place between the package-level `instanceRegistry` declaration (currently line 210) and `GetInstanceRegistry` (line 213). Use the sketch from Pattern 1:

```go
// btreeStateEvictor is set at startup by world.go to wire the
// rooms→behaviortree dependency direction without an import cycle.
// nil-safe: a nil evictor is a no-op (used in tests that don't care
// about btree state).
var btreeStateEvictor func(roomId int)

// SetBTreeStateEvictor registers the per-room btree state eviction
// callback. Called once at startup from world.go with
// behaviortree.EvictRoomBTreeState. Safe to leave unregistered in
// tests that don't exercise btree state.
func SetBTreeStateEvictor(fn func(int)) {
    btreeStateEvictor = fn
}
```

- [ ] **Step 7: Rewrite `CheckPortalTimers` body**

Use Pattern 1's sketch verbatim (Phase A snapshot + Phase B per-instance cleanup chain). Replace the entire current body (lines 162-207). Keep the function signature and the existing doc comment text but extend the doc comment to describe the new chain:

```go
// CheckPortalTimers runs the per-tick instance lifecycle pass:
//   - Broadcasts 5-minute and 1-minute warning messages to players
//     inside instances whose portal is about to expire.
//   - On TTL expiry, runs the consolidated cleanup chain: boot any
//     remaining players to OverworldRoomId with a flavor message,
//     deregister the instance, evict each ephemeral room's btree
//     state via the registered callback, and free the ephemeral
//     chunk via TryEphemeralCleanup.
//
// Runs inside util.LockMud() (called from world.go's per-tick loop),
// so concurrent player movement is serialized against the chain.
//
// The TryEphemeralCleanup call here overlaps with the existing
// RoomChange_CleanupEphemeralRooms hook (which fires when a player
// leaves an ephemeral room with no remaining players). Both paths
// are correct: the hook handles "last player left, no TTL yet"
// (typical), and this TTL chain handles "TTL expired, players may or
// may not still be inside" (the leak case). The function self-
// protects against double-free via its instance-active and
// players-present guards.
func (ir *InstanceRegistry) CheckPortalTimers() {
    // ... Pattern 1 sketch body ...
}
```

**Implementation note on `inst.RoomIdMap`:** the map's keys are *original* room IDs and values are *ephemeral* room IDs (per the field comment `// original → ephemeral room ID mapping` at instances.go:26). Iterate as `for _, ephId := range inst.RoomIdMap`. Do NOT iterate the keys — they're not ephemeral IDs.

**Implementation note on `inst.EntryRoomId`:** also an ephemeral room ID. Per `CreateZoneInstance` line 314-325, `EntryRoomId == InstanceId == roomIdMap[zCfg.EntryRoom]` — the entry room is included in `RoomIdMap` as one of its ephemeral values. So btree eviction iterates `RoomIdMap` only; no separate `EntryRoomId` eviction needed. Verify this during implementation by adding a one-line debug log if uncertain.

**Send order in boot phase:** `user.SendText(collapseMsg)` BEFORE `MoveToRoom(userId, inst.OverworldRoomId)`. Players see the message in the ephemeral room context (where the portal was), not after teleport (which would be confusing — the overworld room they land in didn't have a "shimmer" to collapse).

- [ ] **Step 8: Delete `CleanupEmptyInstances`**

Remove lines 122-157 (the entire function and its doc comment). Per Decision 3 in the spec, delete cleanly — no `TODO`-annotated dead-code-with-marker fallback. If the executor encounters a test that exercises `CleanupEmptyInstances` directly, that test is testing the wrong contract under the new model and should be updated to use the explicit `Remove(inst)` path (or deleted). Such a test is unlikely; check via:

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
grep -rn "CleanupEmptyInstances" --include="*.go" .
```

After deletion, expected matches: zero (everything was call-site-only, deleted in sub-step 2B).

- [ ] **Step 9: Update the stale comment in `ephemeral.go`**

Line 173-174 of `ephemeral.go` currently reads:

```go
// Don't clean up rooms belonging to an active instance —
// instance cleanup is handled separately via CleanupEmptyInstances.
```

Update to reflect the new architecture:

```go
// Don't clean up rooms belonging to an active instance —
// instance cleanup is handled by the TTL chain in CheckPortalTimers
// (which calls Remove(inst) before calling TryEphemeralCleanup).
```

#### Sub-step 2B — `world.go` deletion + wire-up

- [ ] **Step 10: Delete the `CleanupEmptyInstances()` call site**

Remove `world.go:787` (`rooms.GetInstanceRegistry().CleanupEmptyInstances()`). Leave `CheckPortalTimers()` at line 786 untouched — it now does the consolidated cleanup work itself.

- [ ] **Step 11: Add the btree-eviction wire-up in `main.go` at startup**

In `main.go`, adjacent to the `hooks.RegisterListeners()` call (around line 224), add:

```go
// Wire the rooms package's btree-eviction callback to behaviortree's
// implementation. Avoids an internal/rooms → internal/behaviortree
// import cycle (behaviortree already imports rooms).
rooms.SetBTreeStateEvictor(behaviortree.EvictRoomBTreeState)
```

Add `"github.com/GoMudEngine/GoMud/internal/behaviortree"` to `main.go`'s imports. Match the existing import grouping style — the file groups internal packages alphabetically; the new import slots between `internal/badinputtracker` (or wherever the alphabet places it).

Do NOT add this to `world.go` — that file is the per-tick worker loop, not the startup site, and adding a one-time registration there would either fire repeatedly (if placed in MainWorker) or never (if placed in a function never called).

### Verify

- [ ] **Step 12: Build (this catches the import cycle if the wire-up is wrong)**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
```

Expected: clean. If you see `import cycle not allowed`, the chain accidentally imports `behaviortree` directly — re-read Pattern 1 and use the `btreeStateEvictor` callback var instead.

- [ ] **Step 13: Vet + full test sweep**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go vet ./...
go test ./...
```

Expected: clean. The chain is exercised end-to-end only by Task 3's tests; this commit may surface no new test failures (good) or surface an integration-test that depended on `CleanupEmptyInstances` (bad — STOP and investigate per scope-creep policy).

### Commit

- [ ] **Step 14: Commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add internal/rooms/instances.go internal/rooms/ephemeral.go world.go main.go
git commit -m "$(cat <<'EOF'
feat(rooms): TTL-triggered instance cleanup chain in CheckPortalTimers

Replace the dead "currentRound >= expiryRound { continue }" no-op in
CheckPortalTimers with a four-step consolidated cleanup chain:

  1. Boot phase. For each ephemeral room with players (O(1) check via
     roomManager.roomsWithUsers), send a flavor message
     ("The portal's shimmer collapses around you — the instance
     unravels.") and MoveToRoom each player to inst.OverworldRoomId.
  2. Deregister phase. instanceRegistry.Remove(inst).
  3. Btree eviction phase. Call the registered btree-state evictor
     callback for each ephemeral room id in inst.RoomIdMap.
  4. Ephemeral chunk free phase. TryEphemeralCleanup(inst.EntryRoomId)
     — its self-protective guards now pass because steps 1+2 cleared
     them.

The chain runs under util.LockMud() like the rest of CheckPortalTimers,
serializing it against player movement.

Wire-up: internal/rooms cannot import internal/behaviortree (the latter
already imports the former in actions_*.go, conditions_*.go, helpers.go
— direct import would form a cycle). The chain uses a package-level
btreeStateEvictor func(int) callback registered at startup from
world.go via rooms.SetBTreeStateEvictor(behaviortree.EvictRoomBTreeState).
This mirrors the mobResolver/roomLoader callback pattern in
internal/mobai/reactor.go.

Removals:
- CleanupEmptyInstances function (instances.go) — its semantics
  ("any room id that no longer loads → remove instance") are subsumed
  by the explicit Remove(inst) call in step 2.
- CleanupEmptyInstances() call at world.go:787.
- Stale comment in ephemeral.go:174 referencing CleanupEmptyInstances.

Now-overlapping TryEphemeralCleanup call sites (the existing
RoomChange_CleanupEphemeralRooms hook + the new TTL chain) are safe by
construction: the function self-protects against double-free via its
"any room has players OR an active instance" guards. The two paths are
complementary — hook handles "last player left, no TTL yet" (typical);
TTL handles "TTL expired, players may or may not still be inside"
(the leak case).

This completes the EvictRoomBTreeState wire-up that 1.8 commit 3
deferred (logged in project_btree_engine_audit_findings.md as
"wire-up pending").

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

Stage ONLY the four listed files (`instances.go`, `ephemeral.go`, `world.go`, `main.go`). Do NOT stage working-tree noise.

---

## Task 3: `test(rooms): cleanup-chain coverage`

Tests-only commit. Adds 4 new `TestCheckPortalTimers_*` tests to `instances_test.go`. May require minor additions to `test_helpers.go`.

**Files:**
- Modify: `internal/rooms/instances_test.go` (append 4 new tests)
- Maybe modify: `internal/rooms/test_helpers.go` (small helper additions if needed)

**Complexity:** Medium. The trickiest test is the "boots players" case because seeding a real `users.UserRecord` is heavy.

### Discovery

- [ ] **Step 1: Re-read `instances_test.go` style**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
sed -n '1,50p' internal/rooms/instances_test.go
```

Confirm:
- Uses `testify/assert`.
- Tests construct `ZoneInstance` literals directly with arbitrary `EntryRoomId`, `RoomIdMap`, etc. — no factory helper needed for plain construction.
- Uses `NewInstanceRegistry()` for isolated registries when needed (e.g., `TestInstanceRegistry_Remove` line 110-149) — but `CheckPortalTimers` is a method on `*InstanceRegistry`, so we can use a fresh registry per test rather than mutating the package-level `instanceRegistry`.

**Important:** `CheckPortalTimers` reads the package-level `roomManager.roomsWithUsers` and the package-level `btreeStateEvictor`. Test isolation strategies:
- For `roomsWithUsers`: use `SeedRoomsForTest` from `test_helpers.go:6` to install a fresh `roomManager`; remember to call the returned cleanup function via `defer`.
- For `btreeStateEvictor`: snapshot the current value, set the test callback, and restore via `defer`. (Or accept that tests mutate the global; document.)

- [ ] **Step 2: Re-read `test_helpers.go`**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
sed -n '1,46p' internal/rooms/test_helpers.go
```

Confirm:
- `SeedRoomsForTest(roomMap, zoneMap)` returns a cleanup function. Use this to install a fresh `roomManager` for each cleanup-chain test.
- `MarkRoomOccupancy(roomId, playerCt, mobCt)` is the public helper for setting `roomsWithUsers[roomId] = N`. Use this in the "boots players" test setup.

- [ ] **Step 3: Decide on a `seedExpiredInstance` helper**

The 4 cleanup tests all need to construct an instance whose `currentRound >= expiryRound`. The easiest shape: pre-set `CreatedRound` to a deterministic past value (e.g., 0) and use a `PortalDuration` that resolves to a small number of rounds (e.g., `"1m"`). Then `util.GetRoundCount()` returns the actual current round (will be ≥ 1m worth of rounds during tests; if not, the test can advance the round counter via existing helpers — check if such a helper exists by grepping `util.SetRoundCount` or `util.GetRoundCount` test usage).

Alternative: construct the instance with `CreatedRound: util.GetRoundCount() - 999999` and `PortalDuration: "1m"` — guaranteed past expiry regardless of test environment.

If a `seedExpiredInstance` helper would simplify the 4 tests substantially, add it to `test_helpers.go`. Likely shape:

```go
// SeedExpiredInstance constructs a ZoneInstance whose TTL has already
// elapsed at the current round count. Useful for testing
// CheckPortalTimers' TTL-cleanup branch.
func SeedExpiredInstance(entryRoomId, overworldRoomId int, ephRoomMap map[int]int) *ZoneInstance {
    return &ZoneInstance{
        InstanceId:      entryRoomId,
        TemplateZone:    "TestZone",
        AuthorizedUsers: []int{},
        OwnerUserId:     0,
        CreatedRound:    0, // long past
        PortalDuration:  "1m", // any non-empty value < currentRound
        OverworldRoomId: overworldRoomId,
        EntryRoomId:     entryRoomId,
        RoomIdMap:       ephRoomMap,
    }
}
```

If existing patterns are flexible enough (4 tests × ~5 lines of inline construction), skip the helper. Executor decides at implementation.

### Implementation

- [ ] **Step 4: Append 4 new tests to `instances_test.go`**

Place at the end of the file (after `TestScaleSpawnStatPools_NoCap` at line 318). Match the existing testify-`assert` style.

| Test name | Setup | Act | Assert |
|---|---|---|---|
| `TestCheckPortalTimers_TtlExpiryDeregisters` | Fresh `ir := NewInstanceRegistry()`. (Note: the production `CheckPortalTimers` is a method — we can call it on the fresh registry.) Add an expired instance via `SeedExpiredInstance` (or inline construction). Use ephemeral RoomIds (e.g., `ephemeralRoomIdMinimum + 100`). `ir.Add(inst)`. | `ir.CheckPortalTimers()` | `assert.Nil(t, ir.FindByRoomId(inst.EntryRoomId))`. `assert.Len(t, ir.All(), 0)`. |
| `TestCheckPortalTimers_TtlExpiryEvictsBtreeState` | Same instance setup as above. Snapshot current `btreeStateEvictor` and `defer` restore. Set up an in-test eviction tracker: `evicted := []int{}; SetBTreeStateEvictor(func(id int) { evicted = append(evicted, id) })`. | `ir.CheckPortalTimers()` | `assert.Contains(t, evicted, ephemeralRoomIdMinimum + 100)` (and any other ephemeral IDs in the map). Length matches `len(inst.RoomIdMap)`. |
| `TestCheckPortalTimers_TtlExpiryBootsPlayers` | Same instance setup. Use `SeedRoomsForTest` to install a roomManager populated with the ephemeral rooms (each with an empty `players` slice initially) AND the overworld destination room. Add a player to one ephemeral room: `roomManager.rooms[ephId].AddPlayer(99); MarkRoomOccupancy(ephId, 1, 0)`. **Caveat:** `MoveToRoom` calls `users.GetByUserId(99)` and dereferences the result. If the test cannot easily seed a real user, skip the move-assertion and instead assert that `roomsWithUsers[ephId]` got cleared OR that `LoadRoom(ephId)` is no longer in `roomManager.rooms`. Decide at implementation by reading `users.GetByUserId` to see if it's nil-safe under no-user. | `ir.CheckPortalTimers()` | If user-seeding worked: `assert.Equal(t, overworldRoomId, users.GetByUserId(99).Character.RoomId)`. Otherwise: `assert.Equal(t, 0, roomManager.roomsWithUsers[ephId])` (player was removed during boot). |
| `TestCheckPortalTimers_NotExpiredNoOp` | Fresh `ir := NewInstanceRegistry()`. Add a NOT-expired instance: `CreatedRound: util.GetRoundCount(), PortalDuration: "999 real hours"`. Snapshot `btreeStateEvictor` per above and install a tracker. | `ir.CheckPortalTimers()` | `assert.NotNil(t, ir.FindByRoomId(inst.EntryRoomId))` (still registered). `assert.Len(t, ir.All(), 1)`. `assert.Empty(t, evicted)` (no eviction on non-expired). |

**On the "BootsPlayers" caveat:** if seeding a real user is too painful, the test can still lock the boot contract by instrumenting an alternative observable. E.g., assert that `inst` is no longer findable in the registry AFTER the chain (which proves Phase B ran), and rely on `TtlExpiryEvictsBtreeState` to prove Phase C ran. The boot phase's player-movement is the hardest to assert without a full user fixture; the executor's call.

**Concrete fallback for BootsPlayers (if user-seeding is unreasonable):** rename the test to `TestCheckPortalTimers_TtlExpiryClearsRoomsWithUsers` and assert that `roomManager.roomsWithUsers` no longer contains the ephemeral room id after the chain. (The chain's `MoveToRoom` deletes from the map via `delete(roomManager.roomsWithUsers, currentRoom.RoomId)` at `roommanager.go:319`.) This proves the boot loop ran and observable side-effects landed without requiring a real user record.

- [ ] **Step 5: Test isolation cleanup**

Each test must:
- `defer` cleanup of `btreeStateEvictor` if it set one.
- Use a fresh `NewInstanceRegistry()` (do NOT mutate the package-level `instanceRegistry`).
- If using `SeedRoomsForTest`, `defer` the returned cleanup function.

### Verify

- [ ] **Step 6: Scoped test**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go test ./internal/rooms/... -run "TestCheckPortalTimers" -v
```

Expected: 4 PASS.

- [ ] **Step 7: Full test sweep**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go test ./...
```

Expected: clean.

### Commit

- [ ] **Step 8: Commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add internal/rooms/instances_test.go internal/rooms/test_helpers.go
git commit -m "$(cat <<'EOF'
test(rooms): cleanup-chain coverage

Add 4 new tests for the TTL-triggered cleanup chain in
CheckPortalTimers:

- TestCheckPortalTimers_TtlExpiryDeregisters — expired instance is
  removed from the registry.
- TestCheckPortalTimers_TtlExpiryEvictsBtreeState — the registered
  btree-eviction callback fires once per ephemeral room id.
- TestCheckPortalTimers_TtlExpiryBootsPlayers (or
  _ClearsRoomsWithUsers fallback) — populated ephemeral room is
  drained during the boot phase.
- TestCheckPortalTimers_NotExpiredNoOp — not-yet-expired instance is
  untouched (registry intact, no eviction calls).

Tests construct fresh InstanceRegistry instances per case and snapshot/
restore the package-level btreeStateEvictor. Helper additions to
test_helpers.go (if any) are minimal and follow the existing
SeedRoomsForTest / MarkRoomOccupancy patterns.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: `audit(rooms): light pass on contract / stub patterns`

Time-capped audit pass (30 min wall-clock) on `internal/rooms/*.go`. Per-finding triage: Critical/High → fix in this commit; Low → log to memory file. If no Critical/High findings, this commit is the memory-file creation only with an "audit clean" note (still gets a chore-style commit message reflecting "no code change").

**Files:**
- Discovery first. Modify production files only on Critical/High findings.
- Create / update directly (NOT via git): `~/.claude/projects/C--Users-Calabe-Davis-workspace-DOGMud/memory/project_rooms_package_audit_findings.md`

**Complexity:** Low → unbounded (capped at 30 min).

### Discovery

- [ ] **Step 1: Set the wall-clock cap**

Note the wall-clock start time. STOP the audit at +30 minutes regardless of completeness. Anything not triaged by the cap goes into the memory file as "deferred — needs another pass."

- [ ] **Step 2: Find candidate sites**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
grep -n "return true$\|return false$" internal/rooms/*.go | grep -v _test.go
```

Each match is a candidate. Most are legitimate search idioms (e.g., `for _, x := range xs { if cond { return true } }; return false`). Spot-check each for the inverted-contract pattern.

- [ ] **Step 3: Spot-check the borderline `AddSign` case from spec research**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
grep -n "func.*AddSign\|func.*AddItem\|func.*AddPlayer\|func.*AddMob" internal/rooms/rooms.go
```

For each "Add"-style function, read the body and check whether returning `false` represents "rejected" or "replaced an existing entry." If the contract is inverted (returns `true` only on replace), eyeball callers via:

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
grep -rn "<funcname>(" --include="*.go" internal/ | head -20
```

If callers correctly handle both branches, log as Low. If callers ignore the bool, the contract is moot — log as Low. Critical only if a caller depends on the (incorrectly-inverted) bool.

- [ ] **Step 4: Severity triage**

Per finding:
- **Critical** = real bug, gameplay-affecting → fix in this commit. Include a test if practical.
- **High** = real contract violation that callers depend on → fix in this commit.
- **Low** = inverted/awkward contract that's correctly used everywhere OR cosmetic → log to memory file, do NOT fix.

### Implementation (conditional)

- [ ] **Step 5 (conditional): Per-finding fixes**

For each Critical/High finding, apply the fix and (if applicable) add a regression test. Keep all fixes scoped to `internal/rooms/*.go` — anything bigger (e.g., a multi-package contract violation) gets logged for a separate pass per scope-creep policy.

- [ ] **Step 6: Create / update the audit-findings memory file**

Open `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\project_rooms_package_audit_findings.md` directly with your file tool (NOT via git). Mirror the 1.5 audit-findings memory file pattern.

If no Critical/High findings:

```markdown
# Rooms Package Audit Findings

Tracker for findings surfaced during the rooms-package pass
(2026-04-18) and any future rooms-package audit passes.

## Audit Pass Summary

Date: 2026-04-18
Wall-clock: 30 minutes (capped per scope policy)
Files reviewed: internal/rooms/*.go (excluding _test.go)
Candidate sites grepped: `return true$` / `return false$` in non-test files
Findings: N candidates reviewed, no Critical or High findings.

## Resolved (Critical / High)

(none — audit clean)

## Low Findings (logged, not fixed)

- Add per-finding entries here. Format:
  - **<filename>:<line> — <function name>**: <one-line description>.
    Why Low: <rationale>. Action: none (or "revisit if X").

## Surfaced During rooms-pass Execution

(none unless a scope-creep fix commit was added — see commit log)

## Pending Decision

(none)
```

If Critical/High found and fixed in this commit, fill the **Resolved** section with one entry per fix (file/line, problem, fix, test added).

**CRITICAL — memory file is NOT git-tracked.** Edit it directly with your file tool. Do NOT `git add` it. Do NOT include it in any commit's staged file list. Memory files live outside the repo at `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\` and are never committed to the DOGMud repo. (1.5 / 1.6 plans had this bug; do not repeat it.)

### Verify

- [ ] **Step 7: Build + vet + full test sweep**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./...
```

Expected: clean. If any audit-fix introduced a regression, STOP — investigate per scope-creep policy.

### Commit

- [ ] **Step 8: Commit (always — even if no code change, the commit documents the audit completion)**

If Critical/High findings were fixed:

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add internal/rooms/<modified-files...>
git commit -m "$(cat <<'EOF'
audit(rooms): light pass on contract / stub patterns

30-minute time-capped audit pass on internal/rooms/*.go. Per-finding
fixes:

- <file>:<line> — <function>: <problem>. Fix: <description>. Test:
  <test name in rooms_test.go / instances_test.go>.

(repeat per finding)

Low findings logged to project_rooms_package_audit_findings.md (memory
file, lives outside repo).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

If no Critical/High findings:

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git commit --allow-empty -m "$(cat <<'EOF'
audit(rooms): light pass on contract / stub patterns

30-minute time-capped audit pass on internal/rooms/*.go. N candidate
sites reviewed (`return true$` / `return false$` patterns and
adjacent Add-/Remove-style contracts). All candidates are legitimate
search idioms or contracts that callers correctly handle — no
Critical or High findings.

Audit log captured in project_rooms_package_audit_findings.md (memory
file, lives outside repo). No production code change.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

(Use `--allow-empty` only if there's truly no code change; otherwise stage the modified files and commit normally. Do NOT stage the memory file.)

---

## Task 5: Memory cleanup (no commit)

Memory-only step. Retires the now-addressed audit-needed flag, marks the btree wire-up complete, and trims the MEMORY.md index. **No git commit** — memory files live outside the repo.

**Files (all outside the repo at `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\`):**
- DELETE: `project_rooms_package_audit_needed.md`
- MODIFY: `project_btree_engine_audit_findings.md` (flip "wire-up pending" → "wire-up complete")
- MODIFY: `MEMORY.md` (remove the index line for the retired file)

### Steps

- [ ] **Step 1: Delete the retired flag file**

```bash
rm "/c/Users/Calabe Davis/.claude/projects/C--Users-Calabe-Davis-workspace-DOGMud/memory/project_rooms_package_audit_needed.md"
```

(This is the ONE place a non-git `rm` of a memory file is appropriate. Do NOT `git rm`.)

- [ ] **Step 2: Update `project_btree_engine_audit_findings.md`**

Open the file and find the Item 3 / `EvictRoomBTreeState` entry that says "wire-up pending" (or similar). Update it to:

```markdown
- **Item 3 — Room state map leak.** Wire-up complete in the
  rooms-package pass (2026-04-18). The TTL chain in
  CheckPortalTimers calls behaviortree.EvictRoomBTreeState (via the
  rooms.SetBTreeStateEvictor callback registered at startup in
  world.go) for each ephemeral room id of an expiring instance.
  Tested by TestCheckPortalTimers_TtlExpiryEvictsBtreeState in
  internal/rooms/instances_test.go.
```

If the file's structure differs from this exact wording, adapt — the load-bearing change is "pending" → "complete" with a date and reference to the test that locks the contract.

- [ ] **Step 3: Update `MEMORY.md` index**

Open `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\MEMORY.md`. Find the line referencing `project_rooms_package_audit_needed.md` and remove it. If `MEMORY.md` has descriptions or status flags for memory files, also remove the description.

If `project_rooms_package_audit_findings.md` is NOT yet in the index, add an entry for it (mirroring the format of other entries).

**Verification:**
```bash
ls "/c/Users/Calabe Davis/.claude/projects/C--Users-Calabe-Davis-workspace-DOGMud/memory/"
```

Expected: no `project_rooms_package_audit_needed.md`. Should still see `project_rooms_package_audit_findings.md` (created/updated in Task 4) and `project_btree_engine_audit_findings.md` (updated above).

**No commit.** This task touches only files outside the repo.

---

## Task 6 (optional): Scope-creep `fix:` commit

Only if execution of Tasks 1-4 surfaced a clear bug that needed a precursor `fix:` commit per the scope-creep policy. If used:

- Place the commit BEFORE the commit that would otherwise demonstrate the issue (e.g., before Task 2 if Task 2's chain implementation surfaced a `Remove` bug).
- Document the issue and fix in the commit body.
- Add a memory-file note to `project_rooms_package_audit_findings.md` under `## Surfaced During rooms-pass Execution`.

Skip this task if Tasks 1-4 went clean.

---

## Task 7: Final verification + merge

Confirm branch shape, run full test sweep, race-mode pass with environment fallback, manual smoke, then merge `--no-ff` into `development`. Do NOT push.

- [ ] **Step 1: Confirm branch commit count and order**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git log --oneline fix/rooms-package-pass ^development
```

Expected: 4 commits in this order (newest first), assuming no scope-creep precursor:

1. `audit(rooms): light pass on contract / stub patterns`
2. `test(rooms): cleanup-chain coverage`
3. `feat(rooms): TTL-triggered instance cleanup chain in CheckPortalTimers`
4. `fix(rooms): AddTemporaryExit allows ephemeral-portal overwrite, rejects others`

Task 5 contributes ZERO commits (memory-only). Optional Task 6 adds 1 if it fired.

If commit count differs, document the actual count and the reason in the merge commit body.

- [ ] **Step 2: Verify branch diff shape**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git diff --stat development...fix/rooms-package-pass
```

Expected files changed:
- `internal/rooms/rooms.go` — `AddTemporaryExit` body + doc comment (commit 1)
- `internal/rooms/rooms_test.go` — 3 new sub-tests (commit 1)
- `internal/rooms/instances.go` — chain rewrite + delete `CleanupEmptyInstances` + add callback var/setter + add `users` import (commit 2)
- `internal/rooms/ephemeral.go` — stale-comment update (commit 2)
- `world.go` — delete `CleanupEmptyInstances()` call + add `SetBTreeStateEvictor` registration + maybe add `behaviortree` import (commit 2)
- `internal/rooms/instances_test.go` — 4 new tests (commit 3)
- `internal/rooms/test_helpers.go` — possibly minor helper additions (commit 3)
- `internal/rooms/<files>` — only if commit 4 fixed Critical/High audit findings

NO other production source files should appear. NO new files should be created in the repo.

If `.claude/settings.local.json`, `feedback/*.txt`, or `Screenshot*.png` show up: STOP, investigate, `git restore --staged` them.

If any memory file (`project_rooms_package_audit_needed.md`, `project_rooms_package_audit_findings.md`, `project_btree_engine_audit_findings.md`, `MEMORY.md`) appears in the diff: STOP — those live OUTSIDE the repo at `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\` and must NEVER appear in `git diff`. If they do, your file tool wrote to in-repo paths by mistake — investigate and revert.

- [ ] **Step 3: Final whole-project verification**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./... -count=1
```

Expected: completely clean. The previously-failing `TestRoom_AddTemporaryExit/duplicate_name_rejected` is now passing. No remaining baseline failures from `internal/rooms/`. `-count=1` defeats stale cache.

- [ ] **Step 4: Race-mode pass on rooms (environment-aware)**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go test ./internal/rooms/... -race
```

Expected: clean. The new `TestCheckPortalTimers_*` tests exercise the chain's lock-drop-then-Remove pattern (Phase A RLock → release → Phase B per-instance work including `Remove` which takes write lock). The chain's correctness depends on the MUD-wide `util.LockMud()` for `roomsWithUsers` coordination, which `-race` doesn't directly observe — but `-race` will flag any in-package mu mishandling.

**Environment fallback (carryover from 1.6/1.8 precedent):** `-race` requires CGO and a working gcc on Windows. If the run fails with a `cgo: C compiler "gcc" not found` style error, document the skip in the merge commit body — do NOT block on it.

If `-race` does run and reports a real race (not a known flake), STOP — investigate and either add a `fix:` commit or skip the offending test with a memory-file note before merging.

- [ ] **Step 5: Confirm new test count**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go test ./internal/rooms/... -run "TestRoom_AddTemporaryExit|TestCheckPortalTimers" -v 2>&1 | grep -c "^=== RUN"
```

Expected: 11 `=== RUN` lines (1 parent `TestRoom_AddTemporaryExit` + 5 sub-tests + 4 `TestCheckPortalTimers_*` + 1 parent for the table = roughly 10-11; precise count depends on how Go reports sub-tests via `=== RUN`). Adjust the regex if needed; the load-bearing check is "all 4 new TestCheckPortalTimers tests + all 5 sub-tests under TestRoom_AddTemporaryExit ran and passed."

- [ ] **Step 6: Confirm memory files are correct (sanity check, not a git operation)**

```bash
ls "/c/Users/Calabe Davis/.claude/projects/C--Users-Calabe-Davis-workspace-DOGMud/memory/"
```

Expected:
- `project_rooms_package_audit_needed.md` — **NOT present** (deleted in Task 5).
- `project_rooms_package_audit_findings.md` — present (created/updated in Task 4).
- `project_btree_engine_audit_findings.md` — present, with Item 3 "wire-up complete" entry.
- `MEMORY.md` — present, no longer referencing `audit_needed`.

If any of those is wrong, edit directly to fix. Do NOT git-add.

- [ ] **Step 7: Manual smoke test (gating for this pass)**

Boot a local server. Walk through the 7-step Sable portal scenario from the spec:

1. Boot, walk to Sable, ask for a known instance with enough gold → portal opens.
2. Walk through portal → enter ephemeral instance.
3. Watch the warning broadcasts at 5 min and 1 min before TTL.
4. **Stay inside past TTL** → expect the flavor message (`"The portal's shimmer collapses around you — the instance unravels."`) and find yourself in the overworld room where the portal was created.
5. Verify the entry portal in that overworld room is gone (`ExitsTemp` cleared — try walking the portal direction; should fail).
6. Verify the instance no longer appears in any admin "list instances" debug command (if one exists; otherwise inspect via debug log).
7. **Negative smoke for `AddTemporaryExit`:** ask Sable twice for the same zone (back to back) before the first portal expires → second request should refund gold (now-reachable refund branch in `actOpenInstancePortal`). Confirm gold balance returns to pre-second-request level.

If any step fails, STOP — investigate before merging.

If local-boot is impractical, document the skip in the merge commit body. The unit tests in commits 1-3 lock the contracts; the smoke is a final functional sanity check.

---

## Merge to development (after user review)

Do NOT merge until the user has reviewed the branch. Once approved:

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git checkout development
git merge --no-ff fix/rooms-package-pass -m "$(cat <<'EOF'
merge: rooms-package pass

Targeted fix bundle for internal/rooms/ after Stage 1 cleanup
surfaced three concrete contract violations and an unfinished
cleanup chain.

- AddTemporaryExit: replaced unconditional-overwrite behavior with a
  three-path rule (overwrite only ephemeral→ephemeral; reject
  otherwise). Previously-failing duplicate_name_rejected test now
  passes. Three new sub-tests cover the new branches. Now-reachable
  refund branch in actOpenInstancePortal honors the contract for
  Sable duplicate-portal requests.

- TTL-triggered instance cleanup chain consolidated into the existing
  per-tick CheckPortalTimers: boot players to OverworldRoomId with
  flavor message → Remove(inst) → behaviortree.EvictRoomBTreeState
  per ephemeral room (via callback registered at startup to avoid
  the rooms→behaviortree import cycle) → TryEphemeralCleanup. Replaces
  the dead `currentRound >= expiryRound { continue }` no-op. Deletes
  the redundant CleanupEmptyInstances function and its world.go:787
  call site. Four new TestCheckPortalTimers_* tests lock the contract.

- 30-minute audit pass on internal/rooms/*.go: <N findings; per-finding
  results in commit log>. Low findings logged to
  project_rooms_package_audit_findings.md (memory file, outside repo).

Memory housekeeping: project_rooms_package_audit_needed.md retired
(its three flagged items are all addressed); project_btree_engine_
audit_findings.md updated to mark EvictRoomBTreeState wire-up as
complete.

Verified by go build / go vet / go test ./... clean across every
commit (no remaining baseline failures from internal/rooms/) plus
go test -race on internal/rooms/... (or a documented gcc-unavailable
skip per 1.6/1.8 precedent). Manual Sable-portal smoke per spec
section "Verification & Rollout."

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

If a scope-creep `fix:` precursor was added during execution (Task 6), mention it in the merge body (one bullet). If the audit pass surfaced and fixed Critical/High findings, expand the audit-pass bullet with one sub-bullet per fix.

If `-race` was skipped due to environment, add a final paragraph: `Race-mode test deferred: gcc not available in this environment. The cleanup chain's correctness depends on util.LockMud() (the MUD-wide lock) for roomsWithUsers serialization; the in-package ir.mu transitions (RLock snapshot → release → per-instance Remove under Lock) follow the standard snapshot-then-mutate Go pattern.`

If the manual smoke was skipped, document why in the merge body.

**Do NOT push to origin.** User reviews merge locally and pushes when ready.

---

## Done

After merge, the rooms package has a leak-free instance lifecycle, a contract-honoring `AddTemporaryExit`, a documented audit trail, and one fewer baseline-failing test. The Stage 1.8 `EvictRoomBTreeState` API is finally wired up. Two long-standing memory-file flags are retired.
