# Code Cleanup 1.5: Error Handling Audit — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Audit the targets listed in `code_cleanup_stage_1_overview.md`
Stage 1.5 — a set of code paths added during ~500 commits *after* the
Phase 37.3a/b error-handling sweep — and fix the findings that matter.
"Matter" is defined by severity tiers (Critical / High / Medium / Low).
Low-severity / defensive-only findings are logged to a memory file for a
future pass rather than fixed here. This is an audit, not a refactor;
per-finding fix lists materialize at execution time from discovery.

**Architecture:** Up to 9 commits on
`feature/stage-1.5-error-handling-audit`. Commit 1 establishes the
memory file. Commits 2–3 sweep the biggest target area (behaviortree
actions + conditions, room entry points). Commits 4–5 are the hooks
batch (spell hooks, type-assertion safety). Commit 6 is the
highest-risk single audit (Sable portal refund paths). Commit 7 is the
dashboard. Commit 8 replaces the three `mudlog` startup panics with
fatal-log-and-exit (the one intentional behavior change). Commit 9
flips the overview and summarizes finding counts in the memory file.
Each target-area commit independently revertable.

**Tech Stack:** Go 1.25. No new dependencies. Verification via existing
tests + `go build`/`go vet`/`go test ./...` + targeted smoke tests
(Sable portal refund, mudlog fatal exit, dashboard load, 5-min test-mud
AI autonomous run).

**Spec:** `docs/superpowers/specs/2026-04-17-code-cleanup-1.5-error-handling-audit-design.md`

**Branch:** `feature/stage-1.5-error-handling-audit` off `development`.

---

## Task 0: Create feature branch

**Files:** none.

- [ ] **Step 1: Verify you're on `development` and clean**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git status
git branch --show-current
```

Expected: `development`. Working tree should be clean EXCEPT for the
following known unrelated working-tree noise:

- `.claude/settings.local.json` — dirty
- `internal/usercommands/_datafiles/feedback/bugs.txt`, `suggestions.txt` — dirty
- `"Screenshot 2026-04-17 084513.png"` — untracked
- `docs/superpowers/specs/2026-04-17-code-cleanup-1.5-error-handling-audit-design.md` — untracked (the spec itself; will be added in Task 1 or equivalent; see note)

These are **out of scope** for 1.5. Do NOT stage or commit them at any
point in this plan. If `git status` shows anything else dirty,
investigate before proceeding.

- [ ] **Step 2: Create feature branch**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git checkout -b feature/stage-1.5-error-handling-audit
```

Expected: `Switched to a new branch 'feature/stage-1.5-error-handling-audit'`.

- [ ] **Step 3: Baseline verification**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./...
```

Expected: clean build, clean vet, and all tests pass EXCEPT for one
known pre-existing failure:

- `TestRoom_AddTemporaryExit/duplicate_name_rejected` in `internal/rooms/`

**This failure is known, pre-existing, and out of scope for 1.5.** Do
NOT attempt to fix it as part of this branch. Confirm before proceeding
that it is the ONLY failing test. If any other test fails, STOP and
investigate — the baseline is broken and must be green (aside from
`TestRoom_AddTemporaryExit`) before you start fixing audit findings.

Per-commit verification below treats that single failure as expected
noise. If any future commit causes additional test failures, STOP and
revert.

---

## Scope Policy: Severity-Tier-Driven Disposition

The spec's severity tiers drive every finding's disposition. Do NOT
spray defensive nil-checks. Do NOT fix findings you can't justify with
the decision tree below.

**Severity tiers (from spec):**

| Tier | Definition | Disposition |
|------|------------|-------------|
| **Critical** | Deref of a pointer that can be nil, on a path reachable in normal gameplay, with no guard. Panic crashes server or kills a hook/goroutine. Includes unsafe type assertions on values that can plausibly be a different type (including `nil`). | **Fix in this substage.** Commit to the correct target-area commit. If discovered outside the target areas, still fix (with `fix:` commit) and note in memory file. |
| **High** | Error path silently drops user-visible state: gold, item, quest progress, buff application, progression delta. Or ignored error return from a mutation call where the caller can't know if the mutation landed. | **Fix in this substage.** Sable portal refund paths live here. |
| **Medium** | Logs but continues with degraded state. No crash, no data loss, but e.g. a room message never went out. | **Fix in this substage unless out-of-scope.** Out-of-scope Mediums go to the memory file. |
| **Low** | Defensive-only — the pointer is provably non-nil per the "one-function provable non-nil" rule. | **Log to memory file, do not fix.** |

**Triage decision tree (apply to every grep match):**

1. **Is the file in an in-scope directory?**
   - No → log to memory file as "out-of-scope finding", skip.
   - Yes → continue.
2. **Is there an existing nil check on this value earlier in the function, with no goroutine boundary and no callback in between?**
   - Yes → provably non-nil. Low severity. Log to memory file. Skip.
   - No → continue.
3. **What is the worst plausible failure?**
   - Panic → **Critical**. Add guard. Fix in this substage.
   - Silent state loss → **High**. Add guard + correct error path. Fix.
   - Degraded state, no data loss → **Medium**. Fix if target area in scope.
   - Defensive-only → **Low**. Log.
4. **Type assertion specifically:** is the asserted type the *only* type the value can hold?
   - Yes (private map, single-source-of-truth) → Low. Log.
   - No (YAML-loaded data, config options, user input) → Critical or Medium. Fix.
5. **Goroutine specifically:** does it already have `defer func() { if r := recover(); r != nil { mudlog.Error(...) } }()`?
   - Yes → no action.
   - No → Critical. Add the canonical recover block (below).

**Ambiguous cases:** pause. Do NOT guess. Ask the user. Log to the
memory file with status "pending decision".

**Scope-creep policy carries over from 1.2a:** clear bug encountered
outside the target area → preceding `fix:` commit. Ambiguous → pause
and ask. Dead code encountered incidentally → `chore:` removal commit.

**No PATCH_NOTES.md entry by default.** Audit findings are internal;
zero player-facing impact on happy paths. Exception: if the Sable
portal refund audit (Task 6) uncovers a refund bug, that one finding
gets a PATCH_NOTES line at execution time.

---

## Canonical Fix Patterns

The executor applies these snippets by rote when they match a finding.
Do NOT invent variations; keep the pattern byte-consistent so diffs
review cleanly.

### Pattern 1 — Missing nil check on registry lookup (entry-point)

When an action function receives a `mobInstanceId`, `userId`, or `roomId`
via `ctx`/`evt` and the lookup result is dereferenced without a guard:

```go
mob := mobs.GetInstance(ctx.InstanceId)
if mob == nil {
    return Failure // or `return false` / `return events.Cancel` per the function's contract
}
```

```go
user := users.GetByUserId(ctx.Event.UserId)
if user == nil {
    return Failure
}
```

```go
room := rooms.LoadRoom(ctx.RoomId)
if room == nil {
    return Failure
}
```

The guard always **returns early** — never log-and-continue, never
silently succeed. A silent-fail nil-check is itself a finding (see
spec risk register).

### Pattern 2 — Unsafe type assertion → two-form

```go
// BEFORE (Critical if value can plausibly be a different type):
if hintsOpt != nil && !hintsOpt.(bool) { ... }

// AFTER:
if b, ok := hintsOpt.(bool); ok && !b { ... }
```

```go
// BEFORE:
if oldCmdPrompt != nil && oldCmdPrompt.(string) == newCmdPrompt { ... }

// AFTER:
if s, ok := oldCmdPrompt.(string); ok && s == newCmdPrompt { ... }
```

### Pattern 3 — Missing goroutine recover (canonical block)

Transcribed verbatim from `internal/integrations/discord/client.go:129-138`
and `internal/llm/client.go:30-39` (the repo's established convention):

```go
go func() {
    defer func() {
        if r := recover(); r != nil {
            mudlog.Error("PANIC", "error", r)
            s := string(debug.Stack())
            for _, str := range strings.Split(s, "\n") {
                mudlog.Error("PANIC", "stack", str)
            }
        }
    }()
    // ...goroutine body...
}()
```

Imports required: `runtime/debug` (for `debug.Stack`), `strings`, and
`github.com/GoMudEngine/GoMud/internal/mudlog`.

### Pattern 4 — Sable portal refund path

Each refund branch **must** refund gold BEFORE emitting the apology
message:

```go
user.Character.Gold += goldAmount // refund
mob.Command("say Something went wrong. <...reason...>. Your gold is returned.")
return Success
```

Any early return between `user.Character.Gold -= goldAmount` and the
final `return Success` **without** a refund is a **High**-severity
finding.

### Pattern 5 — `mudlog.SetupLogger` panic → fatal-log exit

Pre-`slogInstance` setup, slog is not yet available. Use stdlib `log`:

```go
// BEFORE:
panic(fmt.Errorf("log file path is a directory: %s", logPath))

// AFTER:
log.Fatalf("log file path is a directory: %s", logPath)
```

`log.Fatalf` writes the message to stderr and calls `os.Exit(1)`.
Import: `"log"` (stdlib). Apply to all three panic sites. The commit
message documents that this replaces a stack trace with a clean exit.

---

## Task 1: Add error-handling audit memory file

Create the memory file that every later commit appends to. The commit
is tiny but locks the workflow — deferred findings have a place to land
from the start.

**Files:**
- Create: `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\project_error_handling_audit_findings.md`
- Modify: `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\MEMORY.md` — add one index line.

### Write the memory file

- [ ] **Step 1: Create `project_error_handling_audit_findings.md`**

Contents (frontmatter + empty sections):

```markdown
---
name: Error-handling audit findings (Stage 1.5)
description: Running log of Low-severity / deferred / out-of-scope / ambiguous findings discovered during the Stage 1.5 error-handling audit, for a future pass
type: project
---
# Error-Handling Audit Findings (Stage 1.5)

**Why:** Stage 1.5 audits code added after the Phase 37.3a/b sweep.
Findings are triaged by severity tier (Critical/High/Medium fix
in-substage; Low and out-of-scope log here). This file preserves
audit signal for a future pass and keeps 1.5 diffs focused.

**Entry format:**
`- file.go:LINE — <pattern summary> — <severity> — <disposition: fixed|logged|skipped|pending> — <rationale>`

## Open / In-Progress

_(temporary working area during 1.5 execution; empty at merge time)_

## Low-Severity Findings (Not Fixed)

_(defensive-only checks, provably non-nil per the one-function rule,
redundant error-return checks — NOT fixed, logged for a future pass if
the codebase style shifts to require them)_

## Out-of-Scope Findings (Logged During 1.5)

_(findings discovered in files outside the 1.5 target areas; deferred
to the owning substage)_

## Pending Decision (Ambiguous)

_(findings where the executor could not classify without user input;
revisit before closing 1.5)_

## Resolved

_(findings that were initially logged as pending but resolved during
1.5; keep for audit trail)_

## Summary Counts

_(populated by Task 9 at close-out)_
```

- [ ] **Step 2: Update `MEMORY.md` index**

Open `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\MEMORY.md`. Find the
"Future Work" section (around line 111). Add a bullet following the
existing pattern (same style as the `project_pvm_mvp_parity_gaps.md`
line at index line 115):

```markdown
- [Error-handling audit findings (Stage 1.5)](project_error_handling_audit_findings.md) — Low-severity / deferred / out-of-scope findings from the 1.5 audit, for a future pass
```

### Verify

- [ ] **Step 3: Build / vet / test baseline still clean**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./...
```

Expected: no change from Task 0 baseline (same single pre-existing
failure, nothing else).

### Commit

- [ ] **Step 4: Commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add "/c/Users/Calabe Davis/.claude/projects/C--Users-Calabe-Davis-workspace-DOGMud/memory/project_error_handling_audit_findings.md" "/c/Users/Calabe Davis/.claude/projects/C--Users-Calabe-Davis-workspace-DOGMud/memory/MEMORY.md"
git commit -m "$(cat <<'EOF'
docs: add error-handling audit memory file

Create project_error_handling_audit_findings.md with empty sections for
Low / Out-of-Scope / Pending Decision / Resolved / Summary. Every later
commit in this branch appends to this file as findings are triaged.

Index link added to MEMORY.md Future Work section.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

Stage ONLY those two files. Do NOT stage working-tree noise.

---

## Task 2: Behaviortree actions + conditions nil-check audit

Sweep all `actions_*.go` and `conditions_*.go` files in
`internal/behaviortree/`. Entry-point nil checks on `mobs.GetInstance`,
`users.GetByUserId`, `rooms.LoadRoom` at function boundary. Unsafe type
assertions on param maps converted to two-form. This is the biggest
task in the branch.

**Files audited (read in Discovery):**
- `internal/behaviortree/actions_combat.go` (83 LOC)
- `internal/behaviortree/actions_dialogue.go` (142 LOC)
- `internal/behaviortree/actions_mob.go` (355 LOC, excluding `actOpenInstancePortal` lines 223-328 which is Task 6)
- `internal/behaviortree/actions_quest.go` (189 LOC)
- `internal/behaviortree/actions_room.go` (104 LOC)
- `internal/behaviortree/actions_state.go` (53 LOC)
- `internal/behaviortree/conditions_mob.go` (74 LOC)
- `internal/behaviortree/conditions_player.go` (131 LOC)
- `internal/behaviortree/conditions_room.go` (43 LOC)
- `internal/behaviortree/conditions_state.go` (103 LOC)

**Files possibly modified:** subset of the above that have Critical /
High / Medium findings after triage.

### Discovery

- [ ] **Step 1: Re-run grep sweep (confirm line numbers haven't drifted)**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
grep -n "mobs\.GetInstance(" internal/behaviortree/actions_*.go internal/behaviortree/conditions_*.go
grep -n "users\.GetByUserId(" internal/behaviortree/actions_*.go internal/behaviortree/conditions_*.go
grep -n "rooms\.LoadRoom(" internal/behaviortree/actions_*.go internal/behaviortree/conditions_*.go
grep -nE "\.\([A-Za-z][A-Za-z_.]*\)[^,{]" internal/behaviortree/actions_*.go internal/behaviortree/conditions_*.go
grep -n "go func(" internal/behaviortree/actions_*.go internal/behaviortree/conditions_*.go
```

Expected shape (from plan research — executor confirms):
- `mobs.GetInstance(`: ~19 matches across actions_combat / dialogue / mob / conditions_mob.
- `users.GetByUserId(`: ~21 matches — heavy in actions_quest.go (11 matches) and conditions_player.go (7 matches).
- `rooms.LoadRoom(`: ~14 matches.
- Unsafe type assertions: ~4 matches — `actions.go:92` `staticDelay.(type)` (switch — safe), `actions_mob.go:250` `zonesRaw.(type)` (switch — safe, in Sable scope for Task 6), `conditions_room.go:18,38` `v.(string)` (already two-form — Low).
- `go func(`: run against these dirs — if any, each needs Pattern 3.

- [ ] **Step 2: Read-pass per file**

For each file in the list, read top-to-bottom (they're small — 43 to
355 LOC). For each grep match, classify per the severity-tier decision
tree. Pay particular attention to these patterns common in behaviortree
code:

Function entry pattern (most actions_*/conditions_*):
```go
func actFoo(params map[string]any, ctx *EvalContext) Result {
    mob := mobs.GetInstance(ctx.InstanceId)
    // then mob.Character.X used — check if nil guard exists
```

If the guard is absent and the next deref panics → **Critical**. Apply
Pattern 1.

If the guard is absent but the next use is itself nil-tolerant (e.g.,
passed to another function that handles nil), re-evaluate — may be
**Low**.

For `ctx.Event.UserId == 0` paths where `users.GetByUserId(0)` is
expected to return nil, verify downstream code treats nil as valid.

**Known clean reference points (do NOT re-fix):**

- `internal/behaviortree/helpers.go:21-22` — `calcReactionDelay` already
  nil-checks `mob`.
- `internal/behaviortree/helpers.go:85-88` — `TryRoomBehavior` already
  nil-checks `room`.
- `internal/behaviortree/helpers.go:130-133` — `TryMobBehavior` already
  nil-checks `mob`.

These set the house style; new guards should match this exact shape
(return early, no log, no continuation).

- [ ] **Step 3: Read the two switch-based type assertions (safe but verify)**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
sed -n '88,98p' internal/behaviortree/actions.go
sed -n '14,45p' internal/behaviortree/conditions_room.go
```

Expected: both are already `switch v := x.(type)` or `if s, ok := v.(string); ok` — safe, log as Low.

### Triage

For each classified finding:

- [ ] **Step 4: Critical + High + in-scope Medium → fix inline** using Pattern 1 (registry lookup) or Pattern 2 (type assertion).
- [ ] **Step 5: Low findings → append to memory file** under `## Low-Severity Findings (Not Fixed)` with format:
  `- internal/behaviortree/<file>.go:<LINE> — <pattern> — Low — logged — <rationale>`
- [ ] **Step 6: Ambiguous → stop, ask the user, log under Pending Decision.**

### Verify

- [ ] **Step 7: Build + vet + test**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./internal/behaviortree/...
go test ./...
```

Expected: clean (treating the single `TestRoom_AddTemporaryExit` pre-existing failure as baseline).

### Commit

- [ ] **Step 8: Commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add internal/behaviortree/actions_combat.go internal/behaviortree/actions_dialogue.go internal/behaviortree/actions_mob.go internal/behaviortree/actions_quest.go internal/behaviortree/actions_room.go internal/behaviortree/actions_state.go internal/behaviortree/conditions_mob.go internal/behaviortree/conditions_player.go internal/behaviortree/conditions_room.go internal/behaviortree/conditions_state.go "/c/Users/Calabe Davis/.claude/projects/C--Users-Calabe-Davis-workspace-DOGMud/memory/project_error_handling_audit_findings.md"
git commit -m "$(cat <<'EOF'
fix(behaviortree): nil-check audit on action + condition functions

Sweep actions_*.go and conditions_*.go for missing entry-point nil
checks on mobs.GetInstance / users.GetByUserId / rooms.LoadRoom, and
for unsafe type assertions on param-map values.

Fixed: {{CRITICAL_COUNT}} Critical, {{HIGH_COUNT}} High, {{MEDIUM_COUNT}} Medium.
Logged to memory file: {{LOW_COUNT}} Low, {{DEFERRED_COUNT}} deferred.

Sable portal (actOpenInstancePortal) excluded — audited separately in
its own commit (Task 6) because it has real refund behavior to verify.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

Stage ONLY the behaviortree files actually touched + the memory file.
Drop unchanged files from the `git add` list if the executor verified
nothing changed there. Do NOT stage any working-tree noise.

---

## Task 3: Room behavior-tree entry-point audit (TryRoomBehavior / EnsureRoomBTreeState)

Verify top-level entry points are nil-safe. Plan research found both
`TryRoomBehavior` (`internal/behaviortree/helpers.go:84-125`) and
`EnsureRoomBTreeState` (`internal/behaviortree/room_state.go:12-27`)
already correctly handle nil / absent state:

- `TryRoomBehavior` nil-checks `room := rooms.LoadRoom(roomId)` at
  line 85 and returns `false` on nil (line 86-88).
- `TryMobBehavior` nil-checks `mob := mobs.GetInstance(...)` at line 130.
- `EnsureRoomBTreeState` uses a double-checked RWMutex pattern and
  never returns nil.

**Expected outcome: zero code change.** If that holds, this task
commits an audit-verification note to the memory file only (no code
commit). If the executor's read-pass surfaces a real finding not caught
by plan research, fix it here as a normal audit commit.

**Files audited (read in full):**
- `internal/behaviortree/helpers.go` (174 LOC)
- `internal/behaviortree/room_state.go` (27 LOC)

**Files possibly modified:** likely `project_error_handling_audit_findings.md` only.

### Discovery

- [ ] **Step 1: Read both files in full and verify the claims above**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
sed -n '1,174p' internal/behaviortree/helpers.go
sed -n '1,27p' internal/behaviortree/room_state.go
```

For each `mobs.GetInstance` / `rooms.LoadRoom` call, confirm the
following line is a `if X == nil { return ... }` guard.

For `EnsureRoomBTreeState`, verify: RWMutex double-check pattern
correctly preserves the "one writer" invariant, and no caller ever
receives a nil `*BehaviorState`.

- [ ] **Step 2: Look for any `go func(` inside these files**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
grep -n "go func(" internal/behaviortree/helpers.go internal/behaviortree/room_state.go
```

Expected: zero matches. If any match found, apply Pattern 3.

### Triage

- [ ] **Step 3: If the read-pass surfaces NO findings**
  - Append one bullet to the memory file under `## Resolved`:
    `- internal/behaviortree/helpers.go + room_state.go — full audit read-pass — N/A — logged — verified clean, no code change needed`
  - Commit the memory-file update only (message below).

- [ ] **Step 4: If the read-pass surfaces ANY finding**
  - Apply the Triage decision tree. Critical/High/Medium → fix inline.
  - Commit the code change + memory file update using the same commit message, adjusting the `Fixed:` line.
  - If the only findings are Low, log them and commit only the memory file.

**Alternative:** if the read-pass is clean AND the memory-file update
is trivial, the executor MAY fold this task's memory bullet into
Task 2's memory-file append and skip the standalone commit. Decide at
execution time. If folded, skip Step 5 and note the skip in the final
verification (Task 10).

### Verify

- [ ] **Step 5: Build + vet + test**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./internal/behaviortree/...
```

Expected: clean.

### Commit

- [ ] **Step 6: Commit (only if not folded into Task 2)**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add internal/behaviortree/helpers.go internal/behaviortree/room_state.go "/c/Users/Calabe Davis/.claude/projects/C--Users-Calabe-Davis-workspace-DOGMud/memory/project_error_handling_audit_findings.md"
git commit -m "$(cat <<'EOF'
fix(behaviortree): TryRoomBehavior / EnsureRoomBTreeState audit

Full read-pass audit of the top-level behavior-tree entry points.
TryRoomBehavior already nil-checks rooms.LoadRoom (helpers.go:85-88);
TryMobBehavior already nil-checks mobs.GetInstance (helpers.go:130-133);
EnsureRoomBTreeState uses a double-checked RWMutex pattern and cannot
return nil.

Fixed: {{CRITICAL_COUNT}} Critical, {{HIGH_COUNT}} High, {{MEDIUM_COUNT}} Medium.
Logged to memory file: {{LOW_COUNT}} Low.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

Stage ONLY those files. Do NOT include feedback/*.txt etc.

---

## Task 4: Spell hook nil-check + type-assert audit

Audit the small post-37.3 spell hooks and the Buff_ApplyBuffs path.
Each file is 20-131 LOC; most already have the guard pattern inline.
The one known structural finding is `spell_purgeaffliction.go`'s
reliance on `target` being non-nil without checking at the function
boundary.

**Files audited:**
- `internal/hooks/spell_foldanchor.go` (20 LOC)
- `internal/hooks/spell_foldrecall.go` (85 LOC)
- `internal/hooks/spell_purgeaffliction.go` (38 LOC) — **has known finding**
- `internal/hooks/charm_spell.go` (129 LOC)
- `internal/hooks/Buff_ApplyBuffs.go` (131 LOC)

**Files possibly modified:** subset above. The two "unsafe assertion"
files `NewRound_BroadcastHints.go` and `RedrawPrompt_SendRedraw.go` are
in Task 5, NOT here.

### Discovery

- [ ] **Step 1: Re-run grep sweep**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
grep -n "mobs\.GetInstance\|users\.GetByUserId\|rooms\.LoadRoom" internal/hooks/spell_foldanchor.go internal/hooks/spell_foldrecall.go internal/hooks/spell_purgeaffliction.go internal/hooks/charm_spell.go internal/hooks/Buff_ApplyBuffs.go
grep -nE "\.\([A-Za-z][A-Za-z_.]*\)[^,{]" internal/hooks/spell_foldanchor.go internal/hooks/spell_foldrecall.go internal/hooks/spell_purgeaffliction.go internal/hooks/charm_spell.go internal/hooks/Buff_ApplyBuffs.go
```

Expected shape (from plan research):
- `spell_foldanchor.go:15` — `if room := rooms.LoadRoom(roomId); room != nil` — safe Low.
- `spell_foldrecall.go:15,52,64` — all three guarded with `if room != nil`. Safe.
- `spell_foldrecall.go:16` — `currentRoom.GetTempData("allow_recall").(bool)` is two-form (`ok`). Safe.
- `spell_foldrecall.go:78` — `switch v := val.(type)` — safe.
- `spell_purgeaffliction.go:13` — `rooms.LoadRoom(user.Character.RoomId)` — dereffed only inside `if room != nil` branches (lines 22, 29). `user.Character.Name` accessed at line 21, and **`target.Character.Name` accessed at lines 18, 25 with NO nil-check on `target`**.
- `charm_spell.go:97` — `if companion := mobs.GetInstance(charmId); companion != nil` — safe.
- `Buff_ApplyBuffs.go:37,46,71,79,94` — every one is guarded. Safe.

- [ ] **Step 2: Read `spell_purgeaffliction.go` in full**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
sed -n '1,38p' internal/hooks/spell_purgeaffliction.go
```

Note: the function signature is
`resolvePurgeAffliction(user *users.UserRecord, target *users.UserRecord)`.
The callers (spell resolution in `internal/hooks/spell_resolution.go`)
*should* always pass non-nil — verify but don't fan out the audit.
The contract is not enforced by type; a defensive guard at the function
boundary is the correct fix.

**Severity:** Critical (panic if `target == nil`) — apply Pattern 1-style guard:

```go
func resolvePurgeAffliction(user *users.UserRecord, target *users.UserRecord) {
    if user == nil || target == nil {
        mudlog.Error("resolvePurgeAffliction", "error", "nil user or target")
        return
    }
    // ... rest unchanged
}
```

Requires adding `"github.com/GoMudEngine/GoMud/internal/mudlog"` to
imports. The `mudlog.Error` call is load-bearing per spec risk
register — silent returns are themselves a finding.

- [ ] **Step 3: Read the remaining four files**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
sed -n '1,20p' internal/hooks/spell_foldanchor.go
sed -n '1,85p' internal/hooks/spell_foldrecall.go
sed -n '1,129p' internal/hooks/charm_spell.go
sed -n '1,131p' internal/hooks/Buff_ApplyBuffs.go
```

For each function, confirm: every registry lookup is followed by a nil
guard OR the call is inside an `if X := f(); X != nil {` binding.
Confirm `charm_spell.go:23` `resolveCharmSpell(user, targetMob, room)`
takes three non-nil params — if the audit finds the callers all pass
non-nil and type-system guarantees it, log as Low. If callers cannot
be verified, add defensive boundary checks same as purgeaffliction.

### Triage

- [ ] **Step 4: Apply fixes per Pattern 1 / Pattern 2**
  - `spell_purgeaffliction.go` boundary guard (Critical).
  - Any other finding surfaced during read-pass.
- [ ] **Step 5: Log Low findings to memory file**
  - Most of this file set is already safe — expect most entries to be Low.

### Verify

- [ ] **Step 6: Build + vet + test**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./internal/hooks/...
go test ./...
```

Expected: clean.

### Commit

- [ ] **Step 7: Commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add internal/hooks/spell_foldanchor.go internal/hooks/spell_foldrecall.go internal/hooks/spell_purgeaffliction.go internal/hooks/charm_spell.go internal/hooks/Buff_ApplyBuffs.go "/c/Users/Calabe Davis/.claude/projects/C--Users-Calabe-Davis-workspace-DOGMud/memory/project_error_handling_audit_findings.md"
git commit -m "$(cat <<'EOF'
fix(hooks): spell hook nil-check + type-assert audit

Audit spell_foldanchor, spell_foldrecall, spell_purgeaffliction,
charm_spell, Buff_ApplyBuffs. Add defensive boundary guards where the
function contract is not type-enforced and callers might in principle
pass nil (notably resolvePurgeAffliction, which dereferences
target.Character.Name without a boundary check).

Fixed: {{CRITICAL_COUNT}} Critical, {{HIGH_COUNT}} High, {{MEDIUM_COUNT}} Medium.
Logged to memory file: {{LOW_COUNT}} Low.

Type-assertion fixes for NewRound_BroadcastHints and
RedrawPrompt_SendRedraw deferred to the next commit (narrower concern).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

Stage ONLY files actually modified + memory file.

---

## Task 5: Type-assertion safety on opt-out reads

Two known unsafe single-form type assertions discovered by plan
research. This commit is narrow on purpose — single-concern, clean
revert surface.

**Files audited:**
- `internal/hooks/NewRound_BroadcastHints.go` (95 LOC)
- `internal/hooks/RedrawPrompt_SendRedraw.go` (44 LOC)

**Files possibly modified:** both of the above.

### Discovery

- [ ] **Step 1: Confirm line numbers**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
grep -nE "hintsOpt\.\(bool\)|oldCmdPrompt\.\(string\)" internal/hooks/NewRound_BroadcastHints.go internal/hooks/RedrawPrompt_SendRedraw.go
```

Expected:
- `internal/hooks/NewRound_BroadcastHints.go:80` — `if hintsOpt != nil && !hintsOpt.(bool)`
- `internal/hooks/RedrawPrompt_SendRedraw.go:29` — `if oldCmdPrompt != nil && oldCmdPrompt.(string) == newCmdPrompt`

Both are single-form assertions on values read from `GetConfigOption` /
`GetTempData` (both `any`-valued). If the stored value is of a
different type (YAML round-trip, stale data, player config tamper),
the assertion panics. Severity: **Critical**.

- [ ] **Step 2: Read the two blocks in context**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
sed -n '70,90p' internal/hooks/NewRound_BroadcastHints.go
sed -n '20,45p' internal/hooks/RedrawPrompt_SendRedraw.go
```

### Fix

- [ ] **Step 3: Apply Pattern 2 to `NewRound_BroadcastHints.go:80`**

```go
// BEFORE (line 80):
if hintsOpt != nil && !hintsOpt.(bool) {
    continue
}

// AFTER:
if b, ok := hintsOpt.(bool); ok && !b {
    continue
}
```

Semantics: only skip if the value is explicitly `false`. If the value
is nil OR a different type, hints are ON by default (matches the
pre-fix intent per the comment on line 78 "hints default to ON").

- [ ] **Step 4: Apply Pattern 2 to `RedrawPrompt_SendRedraw.go:29`**

```go
// BEFORE (line 29):
if oldCmdPrompt != nil && oldCmdPrompt.(string) == newCmdPrompt {
    return events.Continue
}

// AFTER:
if s, ok := oldCmdPrompt.(string); ok && s == newCmdPrompt {
    return events.Continue
}
```

Semantics: only short-circuit if the stored prompt matches exactly as a
string. If it's missing or wrong type, we re-draw (correct fail-open
behavior — the redraw is idempotent).

### Verify

- [ ] **Step 5: Build + vet + test**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./internal/hooks/...
```

Expected: clean.

### Commit

- [ ] **Step 6: Commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add internal/hooks/NewRound_BroadcastHints.go internal/hooks/RedrawPrompt_SendRedraw.go
git commit -m "$(cat <<'EOF'
fix(hooks): type-assertion safety on opt-out reads

Two single-form type assertions on any-valued config / temp data:
- NewRound_BroadcastHints.go:80  hintsOpt.(bool)
- RedrawPrompt_SendRedraw.go:29  oldCmdPrompt.(string)

Convert to two-form. Both fail-open correctly:
- hints default to ON if the stored value is missing or non-bool
- prompt redraws if the cached prompt is missing or non-string (redraw
  is idempotent)

Fixed: 2 Critical. Logged: 0.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

Stage ONLY these two files.

---

## Task 6: Sable portal refund-path hardening

**HIGHEST-RISK TASK IN THE BRANCH.** The one place in the audit where a
real gold-loss bug could live and a real refund-path behavior change
could ship if done wrong. Plan research statically traced the three
refund paths and found them correctly structured; this task's job is to
(1) verify that holds, (2) look for any fourth gold-touching error
path that plan research might have missed, (3) smoke-test each refund
branch, (4) commit either an audit-verification note (no code change)
or a refund fix (with mandatory PATCH_NOTES.md entry).

**Files audited:**
- `internal/behaviortree/actions_mob.go` — `actOpenInstancePortal` (lines 223-328)

**Files possibly modified:** the same file if a real bug is found;
`PATCH_NOTES.md` if a player-visible refund bug is fixed.

### Discovery

- [ ] **Step 1: Read `actOpenInstancePortal` top-to-bottom**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
sed -n '208,328p' internal/behaviortree/actions_mob.go
```

Confirm the function boundaries (`func actOpenInstancePortal` starts at
line 223, closes around line 328 with `return Success`).

- [ ] **Step 2: Enumerate the three known refund paths**

Per plan research:

1. **`rooms.CreateZoneInstance` err** (~line 293-298):
   ```go
   inst, err := rooms.CreateZoneInstance(templateZone, goldAmount, user.UserId, authorizedUsers, ctx.RoomId)
   if err != nil {
       mudlog.Error("actOpenInstancePortal", "zone", templateZone, "userId", user.UserId, "error", err)
       user.Character.Gold += goldAmount // refund
       mob.Command("say Something went wrong. The planes resist me. Your gold is returned.")
       return Success
   }
   ```
   **Verify:** refund (`+= goldAmount`) happens BEFORE the `mob.Command` say.

2. **`rooms.LoadRoom(ctx.RoomId)` nil** (~line 305-310):
   ```go
   room := rooms.LoadRoom(ctx.RoomId)
   if room == nil {
       user.Character.Gold += goldAmount // refund
       mob.Command("say Something went wrong. Your gold is returned.")
       return Success
   }
   ```
   **Verify:** refund happens.

3. **`room.AddTemporaryExit(...)` returns false** (~line 317-321):
   ```go
   if !added {
       user.Character.Gold += goldAmount // refund
       mob.Command("say The archway is already sustaining too many rifts. Your gold is returned.")
       return Success
   }
   ```
   **Verify:** refund happens.

- [ ] **Step 3: Look for ANY early return / abort between `user.Character.Gold -= goldAmount` (line 281) and final `return Success` (line 327) that does NOT refund**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
sed -n '281,327p' internal/behaviortree/actions_mob.go
grep -n "return " internal/behaviortree/actions_mob.go | awk -F: '$2>=281 && $2<=327'
```

Every early return between line 281 and 327 MUST have a preceding
`user.Character.Gold += goldAmount`. If any does NOT, it's a **High**
severity gold-loss finding. Fix by adding the refund line above the
`return`.

- [ ] **Step 4: Verify the gold CHARGE happens exactly once on line 281 and not duplicated or skipped**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
grep -n "Character\.Gold" internal/behaviortree/actions_mob.go
```

Expected: one `-= goldAmount` at ~281, three `+= goldAmount` at ~295, ~307, ~318. No others.

- [ ] **Step 5: Verify the pre-charge validation is complete**

Confirm these checks precede the charge (lines 267-278):
- `minGold` guard (~271-274)
- insufficient-gold guard (~275-278)

If either is weak (e.g., off-by-one, wrong operator), it's a **High**
finding.

### Triage

- [ ] **Step 6: If all three refund paths verified correct AND no fourth path found**
  - Zero code change. Memory-file note under `## Resolved`:
    `- internal/behaviortree/actions_mob.go:223-328 — actOpenInstancePortal refund paths audit — N/A — logged — verified correct on all three paths (CreateZoneInstance err / LoadRoom nil / AddTemporaryExit false); no refund bug detected`
  - Commit the memory-file update with the commit message below (adjusting the `Fixed:` line to 0/0/0).

- [ ] **Step 7: If any finding is discovered**
  - Fix using Pattern 4 (refund BEFORE the apology message).
  - Commit the code change + memory file update.
  - If the finding is a real gold-loss bug affecting players: **add a PATCH_NOTES.md entry** describing the bug and the fix.

### Verify

- [ ] **Step 8: Build + vet + test**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./...
```

Expected: clean.

- [ ] **Step 9: Smoke test — Sable portal refund scenarios (MANDATORY)**

Start local server with test-mud data. Log in, travel to Sable, perform:

1. **Happy path:** ask for a known zone with enough gold (e.g., `ask sable arena 200`).
   - Portal opens.
   - Gold deducted exactly once.
   - Enter portal; arrive in instance.
2. **Insufficient gold:** ask with gold < your wallet but < minGold (e.g., `ask sable arena 50` if minGold is 100).
   - Mob says "That barely covers the runes…"
   - Gold unchanged.
3. **Below-minimum gold:** ask with gold > 0 but < minGold.
   - Same as above.
4. **Insufficient wallet:** ask with gold > wallet.
   - Mob says "You do not have that much gold."
   - Gold unchanged.
5. **Forced failure of `CreateZoneInstance`:** temporarily rename a template zone file (or cause the CreateZoneInstance call to error via a data mutation). Ask for that zone.
   - Mob says "Something went wrong. The planes resist me. Your gold is returned."
   - **Gold refunded to exactly the starting amount.** Verify via `inv` or similar.
   - Restore the renamed file.

Path 2 (`LoadRoom` nil) and path 3 (`AddTemporaryExit` false) are hard
to force from player input. If the executor cannot cheaply construct a
local harness, rely on diff inspection + code review + existing
test-mud AI run. Document the decision in the commit message.

Any deviation = STOP, diagnose before committing.

### Commit

- [ ] **Step 10: Commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add internal/behaviortree/actions_mob.go "/c/Users/Calabe Davis/.claude/projects/C--Users-Calabe-Davis-workspace-DOGMud/memory/project_error_handling_audit_findings.md"
# If a refund bug was found and fixed, also:
# git add PATCH_NOTES.md
git commit -m "$(cat <<'EOF'
fix(behaviortree): Sable portal refund-path hardening

Audit actOpenInstancePortal's three refund paths:
- rooms.CreateZoneInstance err (~line 293-298)
- rooms.LoadRoom nil (~line 305-310)
- room.AddTemporaryExit false (~line 317-321)

Each verified to refund user.Character.Gold += goldAmount BEFORE the
mob's apology message. No fourth gold-touching early-return path
discovered between the charge at line 281 and the final return at
line 327.

Fixed: {{CRITICAL_COUNT}} Critical, {{HIGH_COUNT}} High, {{MEDIUM_COUNT}} Medium.
Logged to memory file: {{LOW_COUNT}} Low.

Verified by manual smoke: happy path + insufficient gold + below-min
gold + forced CreateZoneInstance failure (gold refunded exactly).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

If this commit fixed a real player-visible refund bug, the executor
MUST prepend the PATCH_NOTES.md entry to the staged files and update
the commit body to call out the bug fix explicitly.

---

## Task 7: Dashboard nil-check audit on buildPlayerOverview

Dashboard code is 737 LOC, mostly new since Phase 37.3. Most `u.Character`
accesses in the file already have explicit nil guards (plan research
counted ~12 `if u.Character == nil` checks). Expect mostly Low findings.

**Files audited:**
- `internal/web/admin.progression.go` (737 LOC)

**Files possibly modified:** the same file if any real finding surfaces;
otherwise memory file only.

### Discovery

- [ ] **Step 1: Re-run grep sweep**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
grep -n "u\.Character" internal/web/admin.progression.go
grep -n "if u\.Character == nil" internal/web/admin.progression.go
grep -nE "^func " internal/web/admin.progression.go
grep -nE "\.\([A-Za-z][A-Za-z_.]*\)[^,{]" internal/web/admin.progression.go
grep -n "go func(" internal/web/admin.progression.go
```

Expected shape (from plan research):
- Existing nil checks: 12 matches at lines 185, 204, 282, 369, 393, 459, 494, 536, 565, 601, 710 (one duplicate per scope).
- Functions: `loadRecentUserFiles` (150), `collectPlayers` (195), `buildSkillHealth` (259), `buildStatHealth` (358), `playerTotalActivity` (392), `buildSpellHealth` (438), `buildRecipeHealth` (519), `buildPlayerOverview` (592), `progressionIndex` (677), `progressionAPI` (693).
- Type assertions: likely zero or safe (map iterators only).
- Goroutines: likely zero in this file.

- [ ] **Step 2: Read each `u.Character` usage in context**

For each `u.Character.X` reference, verify there's a prior
`if u.Character == nil { continue }` (or equivalent) within the same
function, with no goroutine boundary between.

Known reference points (do NOT re-fix):
- Line 601: `if u.Character == nil { continue }` in `buildPlayerOverview` — correctly guards all subsequent `c := u.Character` usage.
- Lines 185, 204: guards in `loadRecentUserFiles` and `collectPlayers`.
- Lines 282, 369: guards inside the per-player loops of `buildSkillHealth` and `buildStatHealth`.
- Line 393: guard in `playerTotalActivity` at the top of the function.
- Lines 459, 494, 536, 565: guards in `buildSpellHealth` / `buildRecipeHealth`.
- Line 710: guard in `progressionAPI`.

- [ ] **Step 3: Audit `playerTotalActivity(u)` call sites**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
grep -n "playerTotalActivity(" internal/web/admin.progression.go
```

`playerTotalActivity` nil-checks at the top (line 393). Safe.

- [ ] **Step 4: Audit the HTTP handlers for panic recovery**

The Go `net/http` server wraps handler panics per-request (stdlib
behavior), so the dashboard is panic-isolated by the framework. No
additional recover block needed in handlers. Confirm the two handlers
(`progressionIndex` line 677, `progressionAPI` line 693) don't spawn
naked goroutines inside.

### Triage

- [ ] **Step 5: For each `u.Character.X` access missing a guard**
  - If panic-reachable: **Critical**. Add Pattern 1-style guard.
  - If the caller loop pre-filters nil: **Low**. Log.

- [ ] **Step 6: For any other finding, apply the triage tree.**

Expected outcome: zero or very few real findings. Most will be Low.

### Verify

- [ ] **Step 7: Build + vet + test**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./internal/web/...
go test ./...
```

Expected: clean.

- [ ] **Step 8: Smoke test — dashboard load**

Start local server. Load `/admin/progression` in a browser. Verify:
- Page renders without HTTP 500.
- No panic in the server log.
- Run test-mud AI autonomous activity concurrently for 1-2 minutes and
  reload the dashboard during activity — still clean.

### Commit

- [ ] **Step 9: Commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add internal/web/admin.progression.go "/c/Users/Calabe Davis/.claude/projects/C--Users-Calabe-Davis-workspace-DOGMud/memory/project_error_handling_audit_findings.md"
git commit -m "$(cat <<'EOF'
fix(web): dashboard nil-check audit on buildPlayerOverview

Audit admin.progression.go (737 LOC) for missing u.Character guards,
unsafe type assertions, and panic-reachable paths. Existing guard
pattern (if u.Character == nil { continue }) already in place at 12
sites; read-pass confirms no gap on happy paths.

Fixed: {{CRITICAL_COUNT}} Critical, {{HIGH_COUNT}} High, {{MEDIUM_COUNT}} Medium.
Logged to memory file: {{LOW_COUNT}} Low.

Verified by smoke: /admin/progression loads cleanly during concurrent
test-mud AI activity.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

If the read-pass finds zero code change needed, commit only the memory
file with the same message (set counts to zero).

---

## Task 8: mudlog SetupLogger panics → fatal-log exit

Replace the three `panic(fmt.Errorf(...))` sites in
`internal/mudlog/mudlog.go`'s `SetupLogger` with `log.Fatalf` (stdlib
`log` package). The rationale: these are startup-time "config wrong"
errors on the main goroutine pre-serve; a stack trace adds no
operator value over a clear error line. This is the **one intentional
behavior change** in the branch and gets its own commit for clean
revert surface.

**Files audited:**
- `internal/mudlog/mudlog.go` (100 LOC)

**Files possibly modified:** the same file.

### Discovery

- [ ] **Step 1: Re-confirm the three sites**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
grep -n "panic(fmt\." internal/mudlog/mudlog.go
```

Expected:
- `internal/mudlog/mudlog.go:56` — `panic(fmt.Errorf("log file path is a directory: %s", logPath))`
- `internal/mudlog/mudlog.go:63` — `panic(fmt.Errorf("directory for log file does not exist: %s", dir))`
- `internal/mudlog/mudlog.go:67` — `panic(fmt.Errorf("error accessing log file path: %v", err))`

- [ ] **Step 2: Confirm no goroutine wraps this**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
grep -n "go func\|SetupLogger" internal/mudlog/mudlog.go
```

Expected: `SetupLogger` is called synchronously from `main`. No
goroutine needed; no recover needed.

### Fix

- [ ] **Step 3: Apply Pattern 5 to each panic site**

Edit `internal/mudlog/mudlog.go`:

Add `"log"` to the imports (stdlib — already have `"log/slog"` but
`log.Fatalf` is on the base `log` package, a separate import).

Replace the three panics:

```go
// Line 56 (was panic):
if fileInfo.IsDir() {
    log.Fatalf("log file path is a directory: %s", logPath)
}

// Line 63 (was panic):
if _, err := os.Stat(dir); os.IsNotExist(err) {
    log.Fatalf("directory for log file does not exist: %s", dir)
}

// Line 67 (was panic):
} else {
    log.Fatalf("error accessing log file path: %v", err)
}
```

If adding `import "log"` creates a conflict with the `slog` rename
(unlikely — different identifiers), the executor renames accordingly:
`import stdlog "log"` and uses `stdlog.Fatalf`.

After the fix, `fmt` may no longer be needed in this file. Check:

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
grep -n "fmt\." internal/mudlog/mudlog.go
```

If `fmt` has zero remaining uses, `go vet` or `go build` will flag the
unused import — remove it.

### Verify

- [ ] **Step 4: Build + vet + test**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./...
```

Expected: clean.

- [ ] **Step 5: Smoke test — mudlog fatal exit scenarios**

Three scenarios (manual):

1. **Bad log path (directory):** configure `config.yaml` with
   `LogToFile: true` and `LogPath` pointing at an existing *directory*
   (not a file). Start the server. Expect: clean fatal log line
   `"log file path is a directory: <path>"` and `os.Exit(1)`, NOT a
   panic stack trace.

2. **Missing parent dir:** configure `LogPath` under a non-existent
   directory. Start server. Expect: fatal log line
   `"directory for log file does not exist: <dir>"` and exit 1.

3. **Normal path:** configure valid `LogPath`. Start server. Expect:
   normal startup, log file created, no crash.

4. **LogToFile false:** configure `LogToFile: false` (empty `LogPath`).
   Start server. Expect: normal startup logging to stderr.

Revert config changes between scenarios.

### Commit

- [ ] **Step 6: Commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add internal/mudlog/mudlog.go
git commit -m "$(cat <<'EOF'
fix(mudlog): replace SetupLogger panics with fatal-log exit

Three panic(fmt.Errorf(...)) sites at startup become log.Fatalf — the
only intentional behavior change in the 1.5 audit. A stack trace here
adds no operator value over a clear error line; log.Fatalf writes to
stderr and calls os.Exit(1).

Sites:
- mudlog.go:56  log file path is a directory
- mudlog.go:63  directory for log file does not exist
- mudlog.go:67  error accessing log file path

SetupLogger runs synchronously from main pre-serve; no goroutine
recover needed.

Verified by smoke: bad log path (directory) and missing parent dir
both produce a clean fatal line + exit 1 instead of a stack trace.
Normal startup and LogToFile: false unaffected.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 9: Mark 1.5 complete in overview + finalize memory file

Flip the Stage 1.5 status in the overview doc and append the final
summary count section to the memory file. Docs-only commit.

**Files:**
- Modify: `docs/superpowers/code_cleanup_stage_1_overview.md`
- Modify: `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\project_error_handling_audit_findings.md`

### Update

- [ ] **Step 1: Flip 1.5 status in overview**

Open `docs/superpowers/code_cleanup_stage_1_overview.md`. Find the
Stage 1.5 row in the summary table (around line 21):

```markdown
| 1.5 | Error Handling Audit | 4h | Medium | Not started |
```

Change `Not started` to `Complete`. Do NOT touch other rows.

- [ ] **Step 2: Populate `## Summary Counts` in the memory file**

Open `project_error_handling_audit_findings.md`. Find the `## Summary Counts`
heading at the bottom. Replace the placeholder body with:

```markdown
## Summary Counts

**Fixed (in-substage):**
- Critical: {{TOTAL_CRITICAL}}
- High: {{TOTAL_HIGH}}
- Medium: {{TOTAL_MEDIUM}}

**Logged (deferred):**
- Low: {{TOTAL_LOW}}
- Out-of-scope: {{TOTAL_OUT_OF_SCOPE}}
- Pending decision: {{TOTAL_PENDING}}

**Commits:**
- Task 1 (memory file setup)
- Task 2 (behaviortree actions + conditions): {{TASK2_FIX_COUNT}} fixed, {{TASK2_LOG_COUNT}} logged
- Task 3 (room btree entry points): {{TASK3_FIX_COUNT}} fixed, {{TASK3_LOG_COUNT}} logged (or folded into Task 2)
- Task 4 (spell hooks): {{TASK4_FIX_COUNT}} fixed, {{TASK4_LOG_COUNT}} logged
- Task 5 (type-assertion safety): {{TASK5_FIX_COUNT}} fixed, 0 logged
- Task 6 (Sable portal): {{TASK6_FIX_COUNT}} fixed, {{TASK6_LOG_COUNT}} logged
- Task 7 (dashboard): {{TASK7_FIX_COUNT}} fixed, {{TASK7_LOG_COUNT}} logged
- Task 8 (mudlog fatal exit): 3 fixed (Critical), 0 logged

**Closeout date:** 2026-04-17 (or execution date).
```

Executor fills placeholders with real counts drawn from the per-commit
bodies.

- [ ] **Step 3: Verify no PATCH_NOTES.md entry required**

PATCH_NOTES.md only gets an entry if Task 6 uncovered a real
player-facing refund bug. If Task 6 committed zero code change,
PATCH_NOTES.md is NOT touched here.

If Task 6 DID ship a refund-bug fix, the PATCH_NOTES line was added in
Task 6's commit; no additional update here.

### Verify

- [ ] **Step 4: Build / vet / test (docs-only should be no-op)**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./...
```

Expected: unchanged from Task 8 baseline.

### Commit

- [ ] **Step 5: Commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add docs/superpowers/code_cleanup_stage_1_overview.md "/c/Users/Calabe Davis/.claude/projects/C--Users-Calabe-Davis-workspace-DOGMud/memory/project_error_handling_audit_findings.md"
git commit -m "$(cat <<'EOF'
docs: mark code cleanup 1.5 complete

Flip Stage 1.5 status to Complete in code_cleanup_stage_1_overview.md.
Populate Summary Counts section in project_error_handling_audit_findings.md
with per-task finding counts.

Total fixed: {{TOTAL_CRITICAL}} Critical, {{TOTAL_HIGH}} High, {{TOTAL_MEDIUM}} Medium.
Total logged: {{TOTAL_LOW}} Low, {{TOTAL_OUT_OF_SCOPE}} out-of-scope, {{TOTAL_PENDING}} pending.

One intentional behavior change: mudlog SetupLogger panics replaced
with log.Fatalf (Task 8). No other player-visible change.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 10: Final verification

- [ ] **Step 1: Confirm branch commit count and order**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git log --oneline feature/stage-1.5-error-handling-audit ^development
```

Expected: 8-9 commits, in this order (newest first):

1. `docs: mark code cleanup 1.5 complete`
2. `fix(mudlog): replace SetupLogger panics with fatal-log exit`
3. `fix(web): dashboard nil-check audit on buildPlayerOverview`
4. `fix(behaviortree): Sable portal refund-path hardening`
5. `fix(hooks): type-assertion safety on opt-out reads`
6. `fix(hooks): spell hook nil-check + type-assert audit`
7. `fix(behaviortree): TryRoomBehavior / EnsureRoomBTreeState audit` (may be absent if folded into #8)
8. `fix(behaviortree): nil-check audit on action + condition functions`
9. `docs: add error-handling audit memory file`

If Task 3 was folded into Task 2, expect 8 commits; otherwise 9. If
Task 6 was a no-op (zero code change but memory-file update), it still
counts as 1 commit.

- [ ] **Step 2: Verify branch diff shape**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git diff --stat development...feature/stage-1.5-error-handling-audit
```

Expected files changed:
- Subset of `internal/behaviortree/actions_*.go` + `conditions_*.go` (depending on findings)
- `internal/behaviortree/actions_mob.go` (Sable audit — may be zero code change)
- `internal/hooks/spell_purgeaffliction.go` (boundary guard added)
- `internal/hooks/NewRound_BroadcastHints.go` (line 80 fix)
- `internal/hooks/RedrawPrompt_SendRedraw.go` (line 29 fix)
- `internal/web/admin.progression.go` (may be zero code change)
- `internal/mudlog/mudlog.go` (three panic → log.Fatalf)
- `docs/superpowers/code_cleanup_stage_1_overview.md` (1-line status flip)
- `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\project_error_handling_audit_findings.md` (created + appended)
- `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\MEMORY.md` (1-line index add)
- Optionally `PATCH_NOTES.md` (only if Task 6 fixed a refund bug)

No other files should appear. If `.claude/settings.local.json`,
`feedback/*.txt`, or `Screenshot*.png` show up: STOP, `git restore --staged`
them, and investigate.

- [ ] **Step 3: Final whole-project verification**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./...
```

Expected: clean. The single pre-existing
`TestRoom_AddTemporaryExit/duplicate_name_rejected` failure remains
(baseline noise). No other test regressions.

- [ ] **Step 4: Verify memory file has summary counts**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
grep -n "## Summary Counts\|^\*\*Fixed\|^\*\*Logged\|^\*\*Commits" "/c/Users/Calabe Davis/.claude/projects/C--Users-Calabe-Davis-workspace-DOGMud/memory/project_error_handling_audit_findings.md"
```

Expected: the `## Summary Counts` section populated (not just the
placeholder body). All placeholders `{{...}}` must be replaced with
real numbers.

- [ ] **Step 5: Targeted smoke (repeat critical scenarios from spec §Verification)**

1. **Full server startup** with test-mud data. Confirm no panics at boot.
2. **5-minute test-mud AI autonomous run.** Review logs for panics or unexpected errors.
3. **Behavior-tree exercise:** walk into Sanctum Basin tutorial rooms. Confirm room trees fire without panics.
4. **Spell exercise:** cast fold-anchor, travel, cast fold-recall (both happy path and blocked-zone path). Cast purge-affliction on self and on another player.
5. **Dashboard:** open `/admin/progression` during concurrent test-mud AI activity. No panics.
6. **Sable portal happy-path + forced-failure refund** (Task 6 smoke, repeated end-to-end).
7. **mudlog fatal-exit** (Task 8 smoke, at least bad-log-path scenario).

Any deviation = STOP. Revert the offending commit and diagnose.

---

## Merge to development (after user review)

Do NOT merge until the user has reviewed the branch. Once approved:

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git checkout development
git merge --no-ff feature/stage-1.5-error-handling-audit -m "$(cat <<'EOF'
merge: stage 1.5 error handling audit

Sweeps post-37.3 additions (behaviortree actions/conditions, spell
hooks, dashboard, mudlog, Sable portal refund paths, unsafe type
assertions) under the severity-tier triage rubric. Critical / High /
in-scope Medium findings fixed inline; Low and out-of-scope findings
logged to project_error_handling_audit_findings.md for a future pass.

One intentional behavior change: mudlog SetupLogger panics replaced
with log.Fatalf for cleaner operator output on startup config errors.

Zero player-visible change on happy paths. Verified by targeted smoke
+ 5-min test-mud AI autonomous run + full go test ./... clean.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

Do NOT push to origin. Let 1.5 bake on `development` for a few days of
test-mud exposure before rolling to `master`. Any Sable-portal or
mudlog-fatal issue surfaced during the soak → revert the specific
commit (Task 6 or Task 8) without losing the rest of the audit.
