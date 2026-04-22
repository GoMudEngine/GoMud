# Rooms Package Pass — Design Spec

## Goal

Targeted fix bundle for `internal/rooms/` after Stage 1 cleanup surfaced
three concrete contract violations and an unfinished cleanup chain. Three
known issues get real fixes — `AddTemporaryExit`'s "always overwrite"
behavior contradicts both its declared contract and a failing test;
`InstanceRegistry.Remove` has no production callers; `behaviortree.EvictRoomBTreeState`
(shipped in Stage 1.8 commit 3) has no production callers — and the
ephemeral-room cleanup chain is consolidated into the existing per-tick
`CheckPortalTimers` so instances and ephemeral rooms stop leaking
forever in production. A 30-minute light audit pass on the rest of
`internal/rooms/*.go` looks for similar "test exists, code doesn't" or
"always returns true" stub patterns; findings either fix in this branch
(Critical/High) or log to a memory file (Low).

This is a **dedicated rooms-package pass**, not a sub-stage of Stage 1.
The deliverable is a smaller leak surface (instance memory, btree state
map, room map), one previously-failing test now passing, the Sable
portal refund branch in `actOpenInstancePortal` finally reachable, and
two long-standing memory-file flags retired.

## Scope

**In scope (3 known issues + 1 audit pass):**

| # | Area | Disposition |
|---|------|-------------|
| 1 | `Room.AddTemporaryExit` duplicate-rejection contract | **REAL FIX (three-path rule).** Allow overwrite only when both the existing AND new exit's `RoomId` are ephemeral (instance-portal upgrade case). Reject in all other cases. The pre-existing failing test `TestRoom_AddTemporaryExit/duplicate_name_rejected` continues to assert reject (its inputs are non-ephemeral 100/200) and starts passing. New tests cover the ephemeral-overwrite-allowed and mixed-type-rejected paths. |
| 2 | TTL-triggered instance cleanup chain (`InstanceRegistry.Remove` + `EvictRoomBTreeState` + `TryEphemeralCleanup` wire-up) | **REAL FIX (consolidated chain in `CheckPortalTimers`).** On TTL expiry: boot players to `OverworldRoomId` with flavor message; call `Remove(inst)`; call `behaviortree.EvictRoomBTreeState(ephId)` for each ephemeral room id; call `TryEphemeralCleanup(any-eph-id)` to free the ephemeral chunk. Drop or annotate `CleanupEmptyInstances` (now redundant). Replaces the dead `currentRound >= expiryRound { continue }` no-op. |
| 3 | Light audit pass on `internal/rooms/*.go` | **AUDIT, ~30-min cap.** Grep `return true$` / `return false$` and adjacent test files. Per-finding triage: real contract violation or real bug → fix in this branch; cosmetic / unchecked return → log to new `project_rooms_package_audit_findings.md` memory file as Low. Anything bigger than the time cap gets logged for a future pass. |

**Out of scope:**

- Re-architecting the instance lifecycle (e.g., a separate goroutine for cleanup, lifecycle events). The consolidated `CheckPortalTimers` chain is the surgical fix that restores leak-free behavior without changing the world-tick architecture.
- Hot-reload of zone YAML at runtime. Independent concern.
- Mob-instance lifecycle. Mob state lives on the mob struct; only room-level btree state map needs eviction.
- Behavior tree engine internals (`actions_*.go`, `conditions_*.go`). Stage 1.8 closed that audit territory.
- `internal/scripting/room_func.go` `ScriptRoom.AddTemporaryExit` wrapper — its `result := r.roomRecord.AddTemporaryExit(...)` already propagates the bool, so the new contract automatically reaches scripts. No change needed.
- Refunding Sable gold on TTL expiry — the gold paid is the price of the instance regardless of how it ends. Boot is the consequence of TTL; refund is not.

## Decisions Locked During Brainstorming

- **Decision 1 — `AddTemporaryExit` duplicate behavior is the third path.** The failing test asserts reject; the comment-doc says "always replaces"; both are true depending on context. The legitimate use case for replace is the "instance portal upgrade" (Sable opens a second portal of the same name to a different ephemeral entry — old portal still in TTL window). The illegitimate case is any other duplicate. The discriminator is `IsEphemeralRoomId(existing.RoomId) && IsEphemeralRoomId(t.RoomId)` — both must point to ephemeral targets to qualify as a portal-upgrade. Anything else returns false. The existing failing test stays unedited (its inputs are RoomId 100/200, both non-ephemeral); the new contract makes it pass. Three new tests cover the new branches. `IsEphemeralRoomId` and `TemporaryRoomExit.RoomId` are both already exported, so no new API surface.

- **Decision 2 — TTL cleanup consolidates into `CheckPortalTimers`.** The existing per-tick path runs every world tick under `util.LockMud()` (world.go line 786). It already iterates `ir.instances` and computes `expiryRound`. The line `if currentRound >= expiryRound { continue // already expired — cleanup handled elsewhere }` is **provably dead** — there is no "elsewhere." The fix replaces that `continue` with the four-step cleanup chain. Per-tick cost is O(N_instances) outer × O(M_rooms) inner; both are constant-time map reads at typical scale (1–5 instances, 5–64 rooms each). No perf concern. Lock contention with `roomsWithUsers` is **not a concern** — `CheckPortalTimers` runs inside `util.LockMud()` (the MUD-wide lock that also serializes player room-change), so reading `roomManager.roomsWithUsers` for O(1) populated-room detection is safe. The existing `ir.mu.RLock()` does not need to upgrade to a write lock for the per-instance work, but `Remove(inst)` takes its own `ir.mu.Lock()` — locking will need to drop and reacquire (or the cleanup loop builds an "expired" snapshot under RLock and then mutates under Lock). Executor decides at implementation; both shapes are correct.

- **Decision 3 — `CleanupEmptyInstances` is redundant after the chain consolidates.** The user's source brief stated it has zero callers — that's incorrect. It's invoked from `world.go:787` directly after `CheckPortalTimers`. Its semantics ("any room id that no longer loads → remove instance") are now subsumed by the TTL path: TTL expiry calls `Remove(inst)` and `TryEphemeralCleanup` directly, no rely-on-rooms-already-gone heuristic needed. Disposition: delete the function and its `world.go:787` call site as part of commit 2. (Conservative alternative: leave with a `// TODO: redundant after consolidated cleanup; remove next round if no edge case surfaces` comment. Executor's choice; the spec recommends delete because dead-code-with-TODO tends to outlive the TODO.)

- **Decision 4 — Boot policy is teleport to `OverworldRoomId` with flavor.** When TTL expires with players still inside, send each affected player to the room where the portal was created (the `OverworldRoomId` already stored on `ZoneInstance`). Flavor message: `"The portal's shimmer collapses around you — the instance unravels."` Standard MUD convention; doesn't punish players for the system reaping their instance. Implementation uses the existing `MoveToRoom(userId, toRoomId)` (rooms.go line 256) which is already lock-safe under `util.LockMud()`. Players in different ephemeral rooms of the same instance all land in the same overworld room — by design, since that's where the portal was.

- **Audit pass severity rules carry over from 1.5.** Real contract violation (test asserts behavior code doesn't honor) or real bug (silent failure where caller can't observe) → fix in this branch with a `fix:` commit. Cosmetic / never-checked return value → log to memory file as Low, do not fix. Time cap: 30 minutes. Anything bigger gets logged for a future pass.

- **Memory file pattern matches 1.5 audit.** New file `project_rooms_package_audit_findings.md` mirrors `project_error_handling_audit_findings.md`. Old file `project_rooms_package_audit_needed.md` retires after this pass — its three known items are all addressed here, plus the audit completes.

- **Scope-creep policy carries over.** Clear bug found while writing the cleanup chain → preceding `fix:` commit. Ambiguous → pause and ask. Dead code spotted incidentally → `chore:` removal commit.

- **No `PATCH_NOTES.md` entry by default.** Internal infra; player-facing impact is implicit ("the world stops leaking memory" / "instance portals expire as designed"). If a player has reported in-the-wild a stuck-in-instance-after-portal-expiry symptom, the boot fix becomes a `PATCH_NOTES` candidate at execution time.

- **`TryEphemeralCleanup` will now have two call sites** (the existing `RoomChange_CleanupEphemeralRooms` hook and the new TTL chain). No double-free risk — the function self-protects via "if any room has players OR an active instance, return early" guards. The two paths are complementary: hook handles "last player left, no TTL yet" (typical); TTL handles "TTL expired, players may or may not still be inside" (the leak case). Document the now-overlapping call sites in commit 2's body.

## Test Inventory

All test work appends to existing `internal/rooms/rooms_test.go` and `internal/rooms/instances_test.go`. No new test files.

| Test name | Verifies |
|-----------|---------|
| `TestRoom_AddTemporaryExit/duplicate_name_rejected` (existing) | Pre-existing failing test. Inputs are non-ephemeral RoomId 100/200; new three-path rule rejects. Flips FAIL → PASS automatically with no test edit. |
| `TestRoom_AddTemporaryExit/ephemeral_overwrite_allowed` (NEW) | Both existing AND new exits target ephemeral RoomIds. Function returns true and the new exit replaces the old. |
| `TestRoom_AddTemporaryExit/mixed_existing_ephemeral_new_normal_rejected` (NEW) | Existing exit ephemeral, new exit non-ephemeral. Returns false. Don't stomp a portal with a non-portal. |
| `TestRoom_AddTemporaryExit/mixed_existing_normal_new_ephemeral_rejected` (NEW) | Existing exit non-ephemeral, new exit ephemeral. Returns false. Don't stomp a regular temp exit with a portal. |
| `TestCheckPortalTimers_TtlExpiryDeregisters` (NEW) | Set up an instance with `CreatedRound + duration < currentRound`. Run `CheckPortalTimers`. Assert `registry.FindByRoomId(entryRoomId)` returns nil afterward. |
| `TestCheckPortalTimers_TtlExpiryEvictsBtreeState` (NEW) | Same setup. Pre-seed `EnsureRoomBTreeState(ephId)` for each room in the instance, capture pointers. Run `CheckPortalTimers`. Assert `EnsureRoomBTreeState(ephId)` returns a *different* pointer (proves eviction). |
| `TestCheckPortalTimers_TtlExpiryBootsPlayers` (NEW) | Populate one ephemeral room with a player via test helpers (set `roomsWithUsers[ephId] = 1`, add to `room.players`). Expire instance. Run `CheckPortalTimers`. Assert player ended up in `OverworldRoomId`. |
| `TestCheckPortalTimers_NotExpiredNoOp` (NEW) | Instance with TTL still in the future. Run `CheckPortalTimers`. Registry still contains the instance, btree state pointers unchanged, no player movement. Locks the no-false-positive contract. |
| Audit-pass tests | Per-finding from the audit. Names finalize at execution. Likely zero or one — most `return true|false` sites in the package are legitimate search idioms, not stubs. |

**Total: 7 new test cases minimum (3 AddTemporaryExit + 4 CheckPortalTimers), plus 1 unedited test that flips PASS, plus audit-driven additions.**

## Architecture

**Execution order — 4–6 commits on `fix/rooms-package-pass` off `development`:**

| # | Commit | Description |
|---|--------|-------------|
| 1 | `fix(rooms): AddTemporaryExit allows ephemeral-portal overwrite, rejects others` | Implement Decision 1's three-path rule in `Room.AddTemporaryExit` (rooms.go line 449). Update the function's doc comment to describe the new contract. Add the three NEW test cases. The pre-existing failing `duplicate_name_rejected` test now passes with no test edit. |
| 2 | `feat(rooms): TTL-triggered instance cleanup chain in CheckPortalTimers` | Replace the dead `currentRound >= expiryRound { continue }` line in `CheckPortalTimers` with the four-step chain: boot players (using `roomManager.roomsWithUsers` for O(1) populated-room detection + `MoveToRoom` to `OverworldRoomId` with flavor message); call `Remove(inst)`; call `behaviortree.EvictRoomBTreeState(ephId)` for each room id in `inst.RoomIdMap`; call `TryEphemeralCleanup(inst.EntryRoomId)`. Delete `CleanupEmptyInstances` and its `world.go:787` call site (or annotate per Decision 3). Add `behaviortree` import to `internal/rooms/instances.go`. Document the now-overlapping `TryEphemeralCleanup` call sites in the commit body. |
| 3 | `test(rooms): cleanup-chain coverage` | Add the four `TestCheckPortalTimers_*` tests to `instances_test.go`. May require small additions to `test_helpers.go` for setting up an instance with a controllable expiry (e.g., a helper that pre-seeds `CreatedRound`). |
| 4 | `audit(rooms): light pass on contract / stub patterns` | Per-finding fixes from the 30-minute audit pass. If no findings, this commit is docs-only — creates `project_rooms_package_audit_findings.md` with the "audit clean" note. |
| 5 | `docs: retire project_rooms_package_audit_needed.md` | Memory cleanup — delete the now-addressed flag file. Update `project_btree_engine_audit_findings.md` to mark `EvictRoomBTreeState` wire-up as complete (was logged as "wire-up pending" by 1.8). |
| 6 | (optional) `fix:` for any clear bug surfaced during commit 2 implementation | Per scope-creep policy. Only if encountered. |

**Why this order:**

- Commit 1 is independent of the cleanup chain and can revert in isolation if the three-path rule has an unforeseen interaction with `actOpenInstancePortal` or the script wrapper.
- Commit 2 is the largest behavioral change (real cleanup runs in production for the first time). Isolated so a revert exactly undoes the chain without touching the AddTemporaryExit fix.
- Commit 3 is tests-only after commit 2 ships the implementation. Could be folded into commit 2; kept separate for review readability and because the tests touch test infrastructure (`test_helpers.go`) that would muddy the implementation diff.
- Commit 4 is the audit pass. Independent of the prior commits; can run in any order but placed after the substantive changes so the audit context is fresh from the spec read-through.
- Commit 5 is the memory-file cleanup. Last because it depends on commits 1–4 being merged-and-staying-merged.

**Commit cadence rule:** after each commit, `go build ./... && go vet ./... && go test ./...` must be clean. After commit 1 the previously-failing `TestRoom_AddTemporaryExit/duplicate_name_rejected` flips to PASS; after that point there are no documented baseline failures from `internal/rooms/`. No `--no-verify`. If a hook fails, fix and create a NEW commit.

**Commit message format:** Conventional commits, heredoc, trailing `Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>`.

## Cleanup-Chain Patterns

Two patterns introduced by this pass; both follow conventions already established elsewhere in the codebase.

### Pattern 1 — TTL-triggered consolidated cleanup chain (commit 2)

Mirror of the standard MUD-wide-tick cleanup pattern (e.g., `RoomMaintenance`, `EphemeralRoomMaintenance`, both invoked from `world.go` under `util.LockMud()`). The chain runs inside `CheckPortalTimers` in the per-tick path that already iterates instances.

Per-instance, on TTL expiry (`currentRound >= expiryRound`):

1. **Boot phase.** For each room id in `inst.RoomIdMap`, check `roomManager.roomsWithUsers[ephId]`. If positive, call `LoadRoom(ephId)`, iterate `room.GetPlayers()`, send each player the flavor message, call `MoveToRoom(userId, inst.OverworldRoomId)`. Skip rooms with no players (the map check is O(1) — no `LoadRoom`/`GetPlayers` per empty room).
2. **Deregister phase.** `instanceRegistry.Remove(inst)`. (Note: the outer loop is over `ir.instances` — mutating the slice during iteration requires either (a) snapshot-then-mutate, or (b) deferred removal list. Executor picks at implementation; both are standard Go idioms.)
3. **Btree eviction phase.** For each room id in `inst.RoomIdMap`, call `behaviortree.EvictRoomBTreeState(ephId)`. The API is no-op-on-missing-key (locked by `TestEvictRoomBTreeState_NoOpOnMissing` from 1.8), so always-call is safe.
4. **Ephemeral chunk free phase.** `TryEphemeralCleanup(inst.EntryRoomId)`. The function self-protects (returns `[]int{}` if any room has players or an active instance), so calling after step 1+2 is correct: players are gone, instance is deregistered, the function proceeds to free the chunk.

Contract:
- A still-in-TTL instance is untouched (`TestCheckPortalTimers_NotExpiredNoOp`).
- An expired empty instance: deregistered, btree state evicted, chunk freed.
- An expired populated instance: players booted to overworld with flavor, then same as empty.
- Concurrent player movement is impossible: the entire chain runs under `util.LockMud()` (MUD-wide lock).

Replaces:
- The dead `if currentRound >= expiryRound { continue // already expired — cleanup handled elsewhere }` line in `instances.go` (current line ~183-184). Deleted in commit 2.
- The `CleanupEmptyInstances` function and its `world.go:787` call site. Deleted in commit 2.

### Pattern 2 — `AddTemporaryExit` three-path rule (commit 1)

Modifies `Room.AddTemporaryExit` (rooms.go line 449). The function:

1. Looks up `existing, present := r.ExitsTemp[exitName]`.
2. If `!present`, sets the new exit and returns `true`.
3. If `present`, applies the discriminator: `if IsEphemeralRoomId(existing.RoomId) && IsEphemeralRoomId(t.RoomId)` → overwrite, return `true`. Else → no change, return `false`.
4. Both `IsEphemeralRoomId` (ephemeral.go line 154) and `TemporaryRoomExit.RoomId` are already exported; no new API surface.

Contract:
- New exit (no name conflict): added, returns `true`. (Locked by existing `TestRoom_AddTemporaryExit/add_first_exit`.)
- Duplicate name, both ephemeral: overwrite, returns `true`. (Locked by NEW `ephemeral_overwrite_allowed`.)
- Duplicate name, mixed ephemeral/non-ephemeral or both non-ephemeral: rejected, returns `false`. (Locked by existing `duplicate_name_rejected` and NEW `mixed_*_rejected` tests.)

Caller policy:
- `actOpenInstancePortal` (`internal/behaviortree/actions_mob.go:311`) already checks the bool return and refunds gold on false — the previously-dead refund branch becomes reachable when a Sable customer requests a duplicate-name portal to a non-ephemeral target. (In practice this is rare since Sable always targets ephemeral instances; but the contract is now honored.)
- `mobcommands/portal.go:83` already checks the bool. Behavior is correct under the new rule.
- `cubegen.go:195`, `instances.go:343` (the return-portal exit) currently ignore the return. Both target ephemeral RoomIds, so they pass under the new rule. (The cubegen one is a fresh exit on a fresh ephemeral room — never a duplicate. The instances.go one is similarly fresh.) No change needed.
- `scripting/room_func.go:363` propagates the bool to scripts. No change needed.

## Verification & Rollout

**Per-commit verification:**
```bash
go build ./...
go vet ./...
go test ./internal/rooms/...
go test ./internal/behaviortree/...
go test ./...
```

After commit 1, the previously-failing `TestRoom_AddTemporaryExit/duplicate_name_rejected` flips to PASS — verify the failure-classification step no longer skips it (i.e., remove any "expected failure" marker if present). After commit 2+, the new `TestCheckPortalTimers_*` tests must pass.

**After all commits:**
- `go test ./... -count=1` to defeat any stale cache.
- `go test ./internal/rooms/... -race` (where supported — environment may lack gcc; if so, document and skip).
- **Local server smoke (manual).** This is the gating smoke for this pass:
  1. Boot, walk to Sable, ask for a known instance with enough gold → portal opens.
  2. Walk through portal → enter ephemeral instance.
  3. Watch the warning broadcasts at 5 min and 1 min before TTL.
  4. **Stay inside past TTL** → expect the flavor message (`"The portal's shimmer collapses around you — the instance unravels."`) and find yourself in the overworld room where the portal was created.
  5. Verify the entry portal in that overworld room is gone (ExitsTemp cleared).
  6. Verify the instance no longer appears in any admin "list instances" debug command (if one exists; otherwise inspect via debug log or by attempting to FindByRoomId on a known ephemeral id).
  7. **Negative smoke for AddTemporaryExit:** ask Sable twice for the same zone (back to back) before the first portal expires → second request should refund gold (now-reachable refund branch in `actOpenInstancePortal`). Confirm gold balance returns to pre-second-request level.

**Pre-push:**
- All commits pass per-commit verification.
- All smoke scenarios pass.
- Memory file `project_rooms_package_audit_needed.md` deleted (commit 5).
- Memory file `project_rooms_package_audit_findings.md` exists and carries any audit-pass Low findings (or the "audit clean" note).
- Memory file `project_btree_engine_audit_findings.md` updated to mark `EvictRoomBTreeState` wire-up as complete.
- `PATCH_NOTES.md` — no entry by default. Re-decide at execution if a player has reported a stuck-in-instance-after-expiry symptom in the wild.

**Working-tree noise to ignore (do not stage):** `.claude/settings.local.json`, `internal/usercommands/_datafiles/feedback/*.txt`, `Screenshot 2026-04-17 084513.png`.

**Branch strategy:**
- Branch: `fix/rooms-package-pass` off `development`.
- Merge `--no-ff` into `development`.
- Do not push to origin until user OKs.

**Rollback:**
- Each commit independently revertable.
- Commit 1 (AddTemporaryExit) and commit 2 (cleanup chain) have no inter-dependency — either can revert without affecting the other.
- Commit 3 (tests) is test-only; revert is purely cosmetic.
- Commit 4 (audit) per-finding fixes, each cleanly bounded; revert by file.
- Commit 5 (docs) is documentation-only; revert is purely cosmetic.

## Risk Register

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Booting players mid-instance feels abrupt or causes confusion | Medium | Flavor message frames the event as in-fiction ("the portal's shimmer collapses"). Standard MUD convention. The 5-min and 1-min warnings already exist (current `CheckPortalTimers` emits them) — the boot is the natural conclusion players have been warned about. |
| `roomsWithUsers` race: `CheckPortalTimers` reads it while another goroutine mutates | **Non-issue** | Verified during research: `CheckPortalTimers` runs inside `util.LockMud()` (world.go line 783–788). All `roomsWithUsers` mutations also happen under `util.LockMud()` (e.g., `MoveToRoom` line 256). The MUD-wide lock serializes both. The internal `ir.mu` lock is for inter-instance-registry coordination only (e.g., admin debug commands), not for room-state coordination. |
| Mutating `ir.instances` while iterating it in commit 2's outer loop causes a slice-mutation-during-range bug | Medium | Pattern is well-known: iterate to build an "expired" snapshot under RLock, then drop RLock and call `Remove(inst)` per snapshot entry under the function's own Lock. Or: defer removals to a post-loop pass. Executor decides; both are correct. Test `TestCheckPortalTimers_TtlExpiryDeregisters` exercises the multi-instance-expiry case to lock the contract. |
| The dead `currentRound >= expiryRound { continue }` line is intentional protection against double-cleanup that I'm misreading | Low | Verified during research. The comment says "cleanup handled elsewhere" — there is no "elsewhere" (grepped all callers of `Remove`, `TryEphemeralCleanup`, `CleanupEmptyInstances`). The line is dead. Reviewers may miss the deletion since it's a `continue`-to-real-work flip; commit 2 message must explicitly call out "this was a no-op; replacing with the real cleanup." |
| `EvictRoomBTreeState` accidentally called on a static (non-ephemeral) room id | **Non-issue** | Loop iterates `inst.RoomIdMap`, which by construction contains only ephemeral room ids (the map is built by `CreateEphemeralZone` / `CreateEphemeralRoomIds`, both of which produce ephemeral ids ≥ `ephemeralRoomIdMinimum`). |
| `TryEphemeralCleanup` double-called in the same tick (once from boot-then-TTL chain, once from `RoomChange_CleanupEphemeralRooms` hook) | **Non-issue** | The function self-protects: returns `[]int{}` if any room in the chunk still has players or an active instance. Second call is no-op (the chunk is already empty). Document the now-overlapping call sites in commit 2's body so future reviewers don't re-litigate. |
| Audit pass scope creep — executor finds 10+ "interesting" patterns and starts fixing all of them | Medium | Strict 30-min cap. Severity rule: only Critical (real bug, real contract violation) and High fix in this branch. Everything else logs to memory file. The user's brief was unambiguous: "we're making the effort, let's fix it all" applies to the three known issues, not to the audit pass which is explicitly time-capped. |
| Pre-existing `TestRoom_AddTemporaryExit/duplicate_name_rejected` flip from FAIL→PASS reads as "tests broken" by reviewers | Low | Commit 1 message explicitly notes the flip and references the spec. Memory file `project_rooms_package_audit_needed.md` documented the intentional baseline failure; that file gets retired in commit 5 with a note in the deletion message. |
| `world.go:787` `CleanupEmptyInstances()` removal breaks something unforeseen (e.g., a test seeded an instance whose rooms got destroyed without `Remove` being called) | Low | If a test depends on the heuristic "destroyed-rooms imply remove the instance," it's testing the wrong contract under the new model — instances should always be `Remove()`'d explicitly. If such a test surfaces, fix the test to use the explicit teardown path. (The conservative fallback is to leave `CleanupEmptyInstances` annotated, per Decision 3; commit 2 picks one approach at implementation.) |

## Success Criteria

- Pre-existing failing `internal/rooms/TestRoom_AddTemporaryExit/duplicate_name_rejected` now passes with no test edit.
- All 7 new tests pass (3 `AddTemporaryExit` cases + 4 `CheckPortalTimers` cases).
- `go build ./... && go vet ./... && go test ./...` clean after every commit. No remaining baseline failures from `internal/rooms/`.
- `go test ./internal/rooms/... -race` passes (or documented as deferred if environment lacks gcc).
- Local server smoke passes:
  - Sable portal expires with a player inside → player teleported to overworld with flavor message.
  - Entry portal exit cleared from overworld room.
  - Instance no longer findable via `FindByRoomId` on any of its ephemeral ids.
  - Duplicate Sable request before first portal expires → gold refunded (now-reachable refund branch fires).
- `Room.AddTemporaryExit` doc comment accurately describes the three-path rule.
- The dead `currentRound >= expiryRound { continue }` line in `CheckPortalTimers` is gone.
- `CleanupEmptyInstances` is deleted (or annotated, per executor's choice in Decision 3) and its `world.go:787` call site cleaned up.
- `behaviortree.EvictRoomBTreeState` is called from `internal/rooms/instances.go` for each ephemeral room in an expiring instance — completing the wire-up that Stage 1.8 commit 3 deferred.
- Memory file `project_rooms_package_audit_needed.md` deleted (its three flagged items are all addressed; audit completed).
- Memory file `project_rooms_package_audit_findings.md` exists and carries any audit-pass Low findings (or "audit clean" note).
- Memory file `project_btree_engine_audit_findings.md` updated to mark the `EvictRoomBTreeState` wire-up as complete (was previously logged as "wire-up pending").
- `fix/rooms-package-pass` merged `--no-ff` into `development`; not pushed to origin until user OK.
- No production code modified outside the four commits described above. Any `fix:` commit added during execution per the scope-creep policy is documented in the merge commit body.
- No `PATCH_NOTES.md` entry (or one entry, if production logs / player reports show the leak / stuck-in-instance has been observed in the wild).
