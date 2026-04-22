# Code Cleanup 1.5: Error Handling Audit — Design Spec

## Goal

Audit the targets listed in `code_cleanup_stage_1_overview.md` Stage 1.5 — a set of code paths added during ~500 commits *after* the Phase 37.3a/b error-handling sweep — and fix the findings that matter. "Matter" is defined by severity tiers (below); low-severity / defensive-only findings are logged to a memory file for a future pass rather than fixed here.

This is an audit, not a refactor. Deliverables come from discovery. The spec commits to the *process* (what to look for, how to triage, how to cut commits); the per-finding fix list materializes during execution.

## Scope

**In scope — directories targeted by this audit:**

| Area | Rationale |
|------|-----------|
| `internal/behaviortree/actions_*.go` + `conditions_*.go` | 30+ action functions, most added in Phase 4b/4c. Heavy reliance on `mobs.GetInstance`, `users.GetByUserId`, `rooms.LoadRoom`. |
| `internal/behaviortree/helpers.go` (TryRoomBehavior, TryMobBehavior) + `room_state.go` (EnsureRoomBTreeState) | Top-level entry points that every btree evaluation passes through. |
| `internal/hooks/spell_foldanchor.go`, `spell_foldrecall.go`, `spell_purgeaffliction.go`, `charm_spell.go` | New Go spell hooks, small (20–130 LOC each), post-37.3. |
| `internal/hooks/Buff_ApplyBuffs.go` | Buff-apply pathway, exercised by every buff including meditating. |
| `internal/hooks/NewRound_BroadcastHints.go`, `RedrawPrompt_SendRedraw.go` | Known unchecked type-assertion sites flagged by discovery grep (`hintsOpt.(bool)`, `oldCmdPrompt.(string)`). |
| `internal/web/admin.progression.go` (`buildPlayerOverview` and related) | Dashboard code; 737 LOC; mostly new since 37.3. |
| `internal/mudlog/mudlog.go` (`SetupLogger`) | Three `panic(fmt.Errorf(...))` sites for missing log directory / bad log path. Panics at startup are fine-ish but these have been called out explicitly in the overview. |
| `internal/web/web.go` goroutines (lines 392, 454) | Verify `defer recover()` is present and wired to `mudlog.Error`, matching the existing convention (`integrations/discord/client.go:131`, `llm/client.go:32`, `inputhandlers/systemcommands.go:141`, `questengine/engine.go:168`). |

**Out of scope — explicitly not this audit:**

- `internal/combat/`, `internal/characters/`, `internal/items/`, `internal/spells/`, `internal/rooms/` — touched heavily in 1.2a/b/c and 1.7; re-auditing them now would overlap those substages' review.
- `internal/usercommands/` *except* `admin.*` files that were flagged by Phase 4b/4c wiring. Regular user commands are covered by existing tests and per-command review; re-auditing all ~70 files is out of the 4h budget.
- `internal/hooks/NewRound_DoCombat*.go`, `spell_resolution.go`, `combat_shared_helpers.go`, `aggro_helpers.go`, `MobAI_Reactor.go` — just refactored in 1.2a (and the PvM/MvP handlers already carry per-step nil guards from 1.2a's scope-creep policy).
- `internal/characters/` `RecalculateStats`, `Wear`, `Validate` — characterization-tested in 1.2b; not re-auditing.
- Tests (`*_test.go`), vendored deps (`vendor/`), generated code, data files.
- Behavior-tree engine robustness (negative cache, delayed-action nil guards, `roomStates` leak) — that is **Stage 1.8**. Any engine-level finding discovered here gets logged to the memory file and left for 1.8.
- Test coverage additions for the audited code paths — that is **Stage 1.6**.

## Decisions Locked During Brainstorming

- **Audit is scoped by directory, not by pattern.** Running `grep mobs.GetInstance(` across the tree produces >350 hits; most are in already-audited areas. Every finding in this audit must come from a file in the in-scope table. The pattern grep is the *discovery tool* inside those directories, not the scope definition.
- **Severity tiers drive fix-now vs log-for-later.** Critical and High always fix. Medium fix in-substage unless the fix requires touching out-of-scope code. Low always logged, never fixed.
- **Ambiguous nil-check rule ("one-function provable non-nil").** If a pointer is returned from a registry call (`GetInstance`, `GetByUserId`, `LoadRoom`) and used in the *same function* without crossing a goroutine boundary and without any `events.AddToQueue`/`room.SendText`/other callback that could re-enter and mutate the registry, a subsequent use within the same function is provably non-nil *only if* the prior nil check exists on the returned value. If no prior check exists, the check is a Critical or Medium finding depending on whether the deref panics or degrades. If a prior check exists earlier in the function on the same value, subsequent uses don't need a redundant check. This rule means we don't spray defensive nil-checks everywhere — we add them at the *boundary* where the value enters the function.
- **Commit cadence: one commit per target area, not per finding.** Per-finding commits would fragment history for findings that cluster in one file. Per-whole-audit mega-commits lose revertability. Target-area commits (e.g., "behaviortree action nil-check audit", "spell hook nil-check audit", "Sable portal refund-path hardening", "file logger panic replaced with fatal-log") give clean review surfaces that each touch one concern. Each commit is independently revertable.
- **Memory file for deferred findings.** Mirror the 1.2a pattern: `~/.claude/projects/C--Users-Calabe-Davis-workspace-DOGMud/memory/project_error_handling_audit_findings.md`. Appended to during execution; contents are Low-severity findings, out-of-scope findings, and any Medium findings that the triage pushes to a future pass. The file is the audit's long-term memory, not a deliverable of 1.5.
- **No new characterization tests.** Audit fixes shouldn't change happy-path behavior. Existing tests + `go build`/`go vet`/`go test ./...` + targeted smoke (Sable portal, fold-anchor/fold-recall cast, purge-affliction self + other, buff application, dashboard load, server startup with intentionally bad log path) are the verification net. Sable portal refund smoke is the one place where a Medium/High fix could ship with observable behavior change if done wrong — so its smoke is mandatory.
- **Scope-creep policy carries over from 1.2a verbatim.** Clear bug → preceding `fix:` commit. Ambiguous → pause and ask. Dead code encountered incidentally → `chore:` removal commit.
- **No PATCH_NOTES.md entry by default.** Audit findings are internal; zero player-facing impact on happy paths. Exceptions: if the Sable portal refund audit uncovers a refund bug (gold silently lost), that one finding does get a PATCH_NOTES line. Flag at execution time.

## Architecture

**Execution order — up to 9 commits on `feature/stage-1.5-error-handling-audit`:**

| # | Commit | Description |
|---|--------|-------------|
| 1 | `docs: add error-handling audit memory file` | Create the `project_error_handling_audit_findings.md` memory file with empty sections (Low, Deferred, Out-of-Scope). Every later commit appends to this file as findings are triaged. Commit is tiny but locks the workflow. |
| 2 | `fix(behaviortree): nil-check audit on action + condition functions` | Sweep `actions_*.go` and `conditions_*.go`. Entry-point nil checks on `mobs.GetInstance`, `users.GetByUserId`, `rooms.LoadRoom` at function boundary. Unsafe type assertions on param maps converted to two-form (`v, ok := x.(T)`). Log Low-severity redundant-check findings to memory file. |
| 3 | `fix(behaviortree): TryRoomBehavior / EnsureRoomBTreeState audit` | Verify top-level entry paths. Likely zero code change (current `TryRoomBehavior` already nil-checks `rooms.LoadRoom`; `EnsureRoomBTreeState` is lock-safe). If truly zero change, fold into commit 2 and skip this commit — decide at execution. |
| 4 | `fix(hooks): spell hook nil-check + type-assert audit` | `spell_foldanchor.go`, `spell_foldrecall.go`, `spell_purgeaffliction.go`, `charm_spell.go`, `Buff_ApplyBuffs.go`. `purgeaffliction` in particular does not currently nil-check `target` before dereferencing `target.Character.Name`; audit confirms the caller always passes non-nil *but* the signature doesn't enforce it — add a defensive check and log-and-return if violated. |
| 5 | `fix(hooks): type-assertion safety on opt-out reads` | Two known unsafe assertions: `NewRound_BroadcastHints.go:80` `hintsOpt.(bool)` and `RedrawPrompt_SendRedraw.go:29` `oldCmdPrompt.(string)`. Convert to two-form. This commit is narrow on purpose — it's the cleanest single-concern fix in the batch. |
| 6 | `fix(behaviortree): Sable portal refund-path hardening` | `actOpenInstancePortal` in `actions_mob.go`. Audit the three existing refund paths (`CreateZoneInstance` err, `LoadRoom` nil, `AddTemporaryExit` false). Verify each refunds gold before emitting the "something went wrong" message. Look for unhandled error returns between `user.Character.Gold -= goldAmount` and the final Success — any early return in that window without a refund is a **High**-severity finding. If refund is correctly scoped in all three paths, commit is the audit-verification commit with no code change (and a note in the memory file). |
| 7 | `fix(web): dashboard nil-check audit on buildPlayerOverview` | `admin.progression.go`. Verify every `u.Character` access is nil-checked (line 601 already does). Grep for `u.Character.X` patterns. Audit `playerTotalActivity(u)` and helpers for the same pattern. Likely small — the existing code pattern is reasonable; any real finding is a Low that gets logged. |
| 8 | `fix(mudlog): replace SetupLogger panics with fatal-log exit` | Three `panic(fmt.Errorf(...))` sites become `mudlog.Error(...)` + `os.Exit(1)` (or a single `log.Fatal(...)` via the stdlib `log` package before slog is set up — server can't start without a log path so os.Exit is fine). A panic here produces a stack trace that's noise; a clean Fatal message is friendlier to operators. This is the one behavior change in the audit — document in commit body. No goroutine recover needed (this runs in main goroutine pre-serve). |
| 9 | `docs: mark code cleanup 1.5 complete` | Flip status in `code_cleanup_stage_1_overview.md`. Summarize finding counts per severity in the memory file at the same time. |

**Why this order:**

- Commit 1 establishes the memory file before any fix commit, so deferred findings have a place to land from the start.
- Commits 2 → 3 cover the biggest target (behaviortree, 30+ functions) first. Doing the heaviest area first means later commits work in smaller, well-scoped files.
- Commits 4 → 5 are the hooks batch. Commit 5 is separated from commit 4 because its two findings are well-defined and narrow; keeping it on its own means a revert of commit 4 doesn't lose the type-assert fixes.
- Commit 6 (Sable portal) is its own commit because it's the one place in the audit where a real bug could live and a real refund behavior could change. If post-merge a player reports gold loss, reverting exactly this commit isolates the change.
- Commit 7 (dashboard) is isolated because it touches `internal/web/`, a different package boundary from hooks/btree.
- Commit 8 (file logger) is its own commit because it has the only intentional behavior change (panic → fatal-log-and-exit). Clean revert surface.

**Commit cadence rule:** after each commit, `go build ./... && go vet ./... && go test ./...` must be clean. No `--no-verify`. If a hook fails, fix and create a new commit (per repo convention).

## Severity Tiers

Each finding gets one tier. The tier determines disposition.

| Tier | Definition | Disposition |
|------|------------|-------------|
| **Critical** | Deref of a pointer that can be nil, on a path reachable in normal gameplay, with no guard. Panic crashes the server or kills a hook/goroutine that should never die. Includes unsafe type assertions on values that can plausibly be a different type (including `nil`). | **Fix in this substage.** Commit to the correct target-area commit. If discovered outside the target areas, still fix (with `fix:` commit) and note in memory file. |
| **High** | Error path silently drops user-visible state: gold, item, quest progress, buff application, progression delta. Or ignored error return from a mutation call where the caller can't know if the mutation landed. | **Fix in this substage.** Sable portal refund paths live here. |
| **Medium** | Logs but continues with degraded state. Example: `rooms.LoadRoom` returns nil and the code pretends the broadcast happened. No state loss, no crash, but the room message never went out. | **Fix in this substage unless out-of-scope.** Out-of-scope Mediums go to the memory file for a future pass. |
| **Low** | Defensive-only — the pointer is provably non-nil per the "one-function provable non-nil" rule, but a reader might want the check for symmetry. Or error return on a path where the function's contract guarantees the call succeeds. | **Log to memory file, do not fix.** The point is to preserve audit signal-to-noise; over-defensive checks pollute diffs. |

Borderline between Critical and Medium: is the deref on the exact nil value, or on something derived from it? If the code does `if u != nil { u.SendText(...) }` around the deref but forgets the check elsewhere — Critical. If the code does `room := rooms.LoadRoom(roomId); room.SendText(...)` with no check at all — Critical. If the code does `if room := rooms.LoadRoom(roomId); room != nil { ... }` and inside the block every deref is fine but the function continues after the block *pretending* the broadcast happened — Medium.

## Discovery Workflow

Per target area, the executor runs the following in order:

1. **Grep sweep** inside the target area's directory(-ies) for each pattern:
   ```bash
   grep -n "mobs\.GetInstance(" <dir>        # instance lookup
   grep -n "rooms\.LoadRoom(" <dir>          # room lookup
   grep -n "users\.GetByUserId(" <dir>       # user lookup
   grep -nE "\\.\\([A-Za-z][A-Za-z_.]*\\)[^,{]" <dir>  # single-form type assertions
   grep -n "go func(" <dir>                  # goroutines
   grep -nE "^[[:space:]]*_, _ =" <dir>      # explicitly-ignored error tuples
   ```
2. **Read-pass per match.** For each match, read ±10 lines around it. Classify by severity tier. If the same function appears twice, treat the function as one finding with the worst-seen severity.
3. **Triage against the decision tree below.** Fix-in-commit / log-to-memory / skip.
4. **Commit.** One commit per target area once the pass is complete. Commit message summarizes finding counts by severity ("fixed: 4 Critical, 2 High, 1 Medium; logged: 3 Low").
5. **Append to memory file.** Each deferred/out-of-scope/Low finding gets one bullet with: file:line, pattern, severity, reason-deferred.

**Expected volume** (from the grep counts done during spec research):

| Area | `GetInstance` | `LoadRoom` | `GetByUserId` | Notes |
|------|-------|-------|------|-------|
| `internal/behaviortree/` | 26 | 15 | 22 | 30+ action fns — biggest area. |
| `internal/hooks/` (in-scope subset only) | ~10 (spell hooks + Buff_ApplyBuffs) | ~5 | ~5 | Most `internal/hooks/` counts are in files that are OUT of scope (combat). |
| `internal/web/` dashboard | (via UserRecord deref patterns) | — | — | `u.Character` deref pattern is the relevant one here. |

These counts set the realistic volume: ~80 grep hits in in-scope code, of which probably 15–25 are real findings (Critical/High/Medium) after triage and the rest are Low (provably non-nil per the rule above).

## Triage Decision Tree

For each grep match or flagged pattern:

1. **Is the file in an in-scope directory?**
   - No → log to memory file as "out-of-scope finding", skip.
   - Yes → continue.
2. **Is there an existing nil check on this value earlier in the function (no goroutine boundary, no callback in between)?**
   - Yes → provably non-nil. Low severity. Log to memory file (not fixed). Skip.
   - No → continue.
3. **What is the worst plausible failure?**
   - Panic (server crash, goroutine death) → **Critical**. Add guard. Fix in this substage.
   - Silent state loss (gold, item, progression) → **High**. Add guard + correct error path. Fix in this substage.
   - Degraded state but no data loss → **Medium**. Add guard if target area is in scope; otherwise log.
   - Defensive-only → **Low**. Log.
4. **Type assertion specifically:** is the asserted type the *only* type the value can hold?
   - Yes (e.g., a private map where every insertion is a single type) → Low. Log.
   - No (YAML-loaded data, config options, user input) → Critical (for panic risk) or Medium (if nil-value happens to satisfy assertion). Fix in this substage.
5. **Goroutine specifically:** does it already have `defer func() { if r := recover(); r != nil { mudlog.Error(...) } }()`?
   - Yes → no action.
   - No → Critical. Add the recover block matching the existing convention (see `internal/integrations/discord/client.go:131`, `internal/llm/client.go:32`, `internal/inputhandlers/systemcommands.go:141`, `internal/questengine/engine.go:168`).

**Ambiguous cases:** pause. Do not guess. Ask the user. Log the case to the memory file with status "pending decision".

## Verification & Rollout

**Per-commit verification:**
```bash
go build ./...
go vet ./...
go test ./...
```

Any failure = fix before committing. For commits 2–4, run `go test ./internal/behaviortree/... ./internal/hooks/...` on every save; full suite before commit.

**Targeted smoke after commit 6 (Sable portal):**
- Log in, travel to Sable, ask for a known zone with enough gold → portal opens, gold deducted exactly once.
- Ask with insufficient gold → mob says so, gold unchanged.
- Ask with below-minimum gold → mob says so, gold unchanged.
- Force a failing `CreateZoneInstance` (e.g., by renaming a template zone file temporarily) → mob says "the planes resist me", gold refunded exactly to starting amount.
- Verify `LoadRoom` refund path by temporarily setting ctx.RoomId to an invalid id in a local test harness (only if cheap; otherwise rely on code review + diff inspection).

**Targeted smoke after commit 8 (mudlog):**
- Start server with `LogToFile: true` and a bogus log path (non-existent directory) → expect clean fatal log line, not a stack trace.
- Start server with `LogToFile: false` → expect normal startup.
- Start server with valid log path → expect normal startup and file rotation behavior unchanged.

**Targeted smoke after all commits:**
- Full server startup with the test mud data, 5 minutes of test-mud AI autonomous exploration.
- Behavior-tree exercise: walk into Sanctum Basin tutorial rooms (Phase 4b/4c migrated), confirm room trees fire without panics.
- Spell exercise: cast fold-anchor, travel, cast fold-recall (both happy path and blocked-zone path). Cast purge-affliction on self and on another player.
- Dashboard: open `/admin/progression` while test-mud AI is running, verify no panics in the log.

**Pre-push:**
- All commits pass per-commit verification.
- All targeted smoke passes.
- Memory file updated with every Low / deferred / pending finding.
- `code_cleanup_stage_1_overview.md` — 1.5 flipped to Complete.
- `PATCH_NOTES.md` — entry only if a real player-facing bug was fixed (likely none, but Sable refund bug is the candidate).

**Branch strategy:**
- Branch: `feature/stage-1.5-error-handling-audit` off `development`.
- Merge `--no-ff` into `development`.
- No urgent prod push. Settle on `development` for a few days before rolling to `master`.

**Rollback:**
- Each target-area commit independently revertable. Commit 6 (Sable portal) and commit 8 (mudlog) are the two with real behavior change; isolate so either can be reverted without losing the rest of the audit.

**Working-tree noise to ignore:** `.claude/settings.local.json`, `internal/usercommands/_datafiles/feedback/*.txt`, `Screenshot 2026-04-17 084513.png`. Do not stage.

## Risk Register

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Audit spirals: executor starts "fixing" defensive Low findings to make the diff feel substantial | **High** | Severity tier rules are strict. Low = log, never fix. If the executor finds themself writing a defensive nil-check with no prior-function-boundary justification, it goes to the memory file instead of the code. |
| False-positive fixes mask a real underlying contract violation (e.g., adding a nil-check on something that shouldn't be nil lets a bug pass silently) | Medium | Every added nil-check that triggers must log at `mudlog.Error` or return visibly — never silently succeed. Silent-fail nil-checks are themselves a finding. |
| Sable portal refund fix changes the gold accounting path and the manual smoke doesn't catch the new bug | Medium | Commit 6 isolated. Smoke script explicitly includes the three refund paths. If refund math changes, PATCH_NOTES entry is mandatory. Diff review on commit 6 by eye before push. |
| mudlog `panic → Fatal` change swallows a stack trace that was operationally useful | Low | The three panic sites are all "config wrong at startup" — stack trace adds nothing over a clear error message. If an operator needs a trace, they still get one from any panic outside SetupLogger. |
| Missing the actual important issue while fixing trivial ones (the audit doesn't cover a code path that's already silently failing in prod) | Medium | Discovery grep is comprehensive across in-scope dirs. Any finding surfaced during smoke that wasn't caught by grep → pause audit, add the missed pattern, rerun grep. Memory file's "pending" section captures these. |
| Scope creep — "while I'm in here" fixes pull in unrelated refactors | Low | Scope-creep policy. `fix:` commits allowed only for clear bugs; ambiguous cases pause. |
| Phase-4b/4c recently-wired hooks I don't know about get missed | Low | Commit 2's grep in `internal/behaviortree/` catches them — all new actions are registered in `actionRegistry`/`conditionRegistry` and the grep sweeps the whole dir. |

## Success Criteria

- Memory file `project_error_handling_audit_findings.md` exists with every Low finding, every deferred finding, and every out-of-scope finding logged with file:line, pattern, severity, reason.
- Every Critical finding in an in-scope directory has a guard + test pass.
- Every High finding has its state-loss path closed. Sable portal refund paths verified correct on all three branches.
- Every in-scope Medium finding is either fixed or explicitly logged with reason-deferred.
- `go build ./...`, `go vet ./...`, `go test ./...` clean after every commit.
- Server starts cleanly; test-mud AI 5-minute run produces no panics in the log.
- Sable portal refund smoke passes all four scenarios.
- mudlog fatal path smoke passes (bad log dir produces clean exit, not panic trace).
- Dashboard loads with concurrent test-mud AI activity — no crash in `buildPlayerOverview`.
- `code_cleanup_stage_1_overview.md` Stage 1.5 row flipped to Complete.
- `PATCH_NOTES.md` entry only if a real player-facing bug was fixed.
- No player-visible behavior change on any happy path.
