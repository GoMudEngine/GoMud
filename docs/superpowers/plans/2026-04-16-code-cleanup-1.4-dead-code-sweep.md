# Code Cleanup 1.4: Dead Code Sweep Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove Go code made dead by the JS scripting bridge removal — orphaned GetScript methods across 6 packages, unused Scripting config struct, and the unused `_datafiles/world/empty/` directory.

**Architecture:** Pure deletion. Each category is removed independently with build + test verification after each step. Three independent categories → three tasks → three commits.

**Tech Stack:** Go, YAML

**Spec:** `docs/superpowers/specs/2026-04-16-code-cleanup-1.4-dead-code-sweep-design.md`

---

## Task 1: Remove orphaned GetScript/HasScript/GetScriptPath methods

**Files to modify:**
- `internal/buffs/buffspec.go` — remove `GetScript()`, `GetScriptPath()`
- `internal/spells/spells.go` — remove `GetScript()`, `GetScriptPath()`
- `internal/rooms/rooms.go` — remove `GetScript()`, `GetScriptPath()`
- `internal/mobs/mobs.go` — remove `GetScript()`, `GetScriptPath()`, `HasScript()`
- `internal/items/items.go` — remove `GetScript()`
- `internal/items/itemspec.go` — remove `GetScript()`, `GetScriptPath()`

Also remove any tests that exclusively test these methods.

- [ ] **Step 1: Verify zero non-test callers**

```bash
cd /c/Users/Calabe\ Davis/workspace/DOGMud
grep -rn "GetScript\|HasScript\|GetScriptPath" internal/ --include="*.go" | grep -v "_test.go"
```

Expected: only the definitions themselves (lines in buffspec.go, spells.go, rooms.go, mobs.go, items.go, itemspec.go) and one `config.scripting.go:20 GetScriptingConfig` which is a different function (unrelated, handled in Task 2). No other files should reference these methods.

- [ ] **Step 2: Remove from buffspec.go**

Read `internal/buffs/buffspec.go` around lines 249-270. Delete the two functions `GetScript()` and `GetScriptPath()`. If they import anything (like `configs` for the DataFiles path), check whether those imports are still used elsewhere in the file after removal — if not, remove them.

- [ ] **Step 3: Remove from spells.go**

Read `internal/spells/spells.go` around lines 251-270. Delete `GetScript()` and `GetScriptPath()`. Clean up imports.

- [ ] **Step 4: Remove from rooms.go**

Read `internal/rooms/rooms.go` around lines 418-440. Delete `GetScript()` and `GetScriptPath()`. Clean up imports.

- [ ] **Step 5: Remove from mobs.go**

Read `internal/mobs/mobs.go` around lines 908-940. Delete `HasScript()`, `GetScript()`, `GetScriptPath()`. Clean up imports.

- [ ] **Step 6: Remove from items.go**

Read `internal/items/items.go` around lines 79-82. Delete `GetScript()`. Clean up imports.

- [ ] **Step 7: Remove from itemspec.go**

Read `internal/items/itemspec.go` around lines 556-580. Delete `GetScript()` and `GetScriptPath()`. Clean up imports.

- [ ] **Step 8: Remove related test functions**

```bash
grep -rn "GetScript\|HasScript\|GetScriptPath" internal/ --include="*_test.go"
```

For each result, open the file and:
- If the test function exclusively tests these methods (its entire body checks `GetScript()` or `GetScriptPath()`), delete the whole test function.
- If the test function tests other things and only incidentally references these methods, delete just the relevant assertions.

- [ ] **Step 9: Build**

```bash
cd /c/Users/Calabe\ Davis/workspace/DOGMud
go build ./...
```

Expected: clean build, no errors.

- [ ] **Step 10: Test**

```bash
go test ./internal/buffs/... ./internal/spells/... ./internal/rooms/... ./internal/mobs/... ./internal/items/...
```

Expected: all pass.

- [ ] **Step 11: Final grep to confirm**

```bash
grep -rn "GetScript\|HasScript\|GetScriptPath" internal/ --include="*.go" | grep -v "GetScriptingConfig"
```

Expected: zero results.

- [ ] **Step 12: Commit**

```bash
git add internal/buffs/ internal/spells/ internal/rooms/ internal/mobs/ internal/items/
git commit -m "$(cat <<'EOF'
refactor: remove orphaned GetScript/HasScript/GetScriptPath methods

After JS bridge removal in Phase 5, these accessor methods had zero
non-test callers across 6 packages (buffs, spells, rooms, mobs,
items, itemspec). Pure dead-code removal, no behavior change.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Remove Scripting config struct

**Files:**
- Delete: `internal/configs/config.scripting.go`
- Modify: `internal/configs/configs.go` — remove Scripting field and Validate call
- Modify: `_datafiles/config.yaml` — remove Scripting: block

- [ ] **Step 1: Verify no non-internal callers**

```bash
cd /c/Users/Calabe\ Davis/workspace/DOGMud
grep -rn "GetScriptingConfig\|configs.Scripting\|\.Scripting\b" internal/ main.go --include="*.go" | grep -v "config.scripting.go\|config.scripting_test"
```

Expected: only `internal/configs/configs.go:47` (struct field) and `internal/configs/configs.go:205` (Validate call). Nothing else.

- [ ] **Step 2: Delete config.scripting.go**

```bash
rm internal/configs/config.scripting.go
```

If there's a `config.scripting_test.go`, delete it too:

```bash
rm -f internal/configs/config.scripting_test.go
```

- [ ] **Step 3: Remove Scripting field from Config struct**

Open `internal/configs/configs.go`. Find the Config struct around line 35-60. Locate the line:

```go
	Scripting    Scripting    `yaml:"Scripting"`
```

Delete that entire line.

- [ ] **Step 4: Remove Scripting.Validate() call**

Still in `internal/configs/configs.go`. Find the `Validate()` method around line 193. Locate the line:

```go
	c.Scripting.Validate()
```

Delete that entire line.

- [ ] **Step 5: Remove Scripting block from config.yaml**

Open `_datafiles/config.yaml`. Find the section starting with:

```
################################################################################
#
#   SCRIPTING
#   Configurations to put limits on run away scripts etc.
#
################################################################################
Scripting:
  # - LoadTimeoutMs -
  ...
```

Delete the entire section — the comment header banner, the `Scripting:` key, and all its children (LoadTimeoutMs, RoomTimeoutMs, and any other fields). End at the line before the next `################################################################################` banner (which will be the next config section).

- [ ] **Step 6: Build**

```bash
cd /c/Users/Calabe\ Davis/workspace/DOGMud
go build ./...
```

Expected: clean build.

- [ ] **Step 7: Test**

```bash
go test ./internal/configs/...
```

Expected: all pass.

- [ ] **Step 8: Full test suite**

```bash
go test ./...
```

Expected: all pass.

- [ ] **Step 9: Final grep to confirm**

```bash
grep -rn "GetScriptingConfig\|\"Scripting\"\|configs.Scripting" internal/ _datafiles/config.yaml main.go --include="*.go" 2>/dev/null
```

Expected: zero results (the YAML grep might match comments in other files — check that the main `_datafiles/config.yaml` has no `Scripting:` block remaining).

- [ ] **Step 10: Commit**

```bash
git add internal/configs/config.scripting.go internal/configs/configs.go _datafiles/config.yaml
git commit -m "$(cat <<'EOF'
refactor: remove dead Scripting config struct

The Scripting struct (LoadTimeoutMs, RoomTimeoutMs) was used by the
goja JS VM for per-script timeouts. After JS removal in Phase 5, no
code reads these fields. Delete the struct, its accessor, the Config
field, the Validate() call, and the yaml block.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

Note: `git add` with a deleted file won't work — use `git rm` instead for the deleted file, or use `git add -A internal/configs/` to stage both the deletion and the modification.

```bash
git add -A internal/configs/ _datafiles/config.yaml
git commit -m "$(cat <<'EOF'
refactor: remove dead Scripting config struct

The Scripting struct (LoadTimeoutMs, RoomTimeoutMs) was used by the
goja JS VM for per-script timeouts. After JS removal in Phase 5, no
code reads these fields. Delete the struct, its accessor, the Config
field, the Validate() call, and the yaml block.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Delete _datafiles/world/empty/ directory

**Files:**
- Delete: `_datafiles/world/empty/` (entire directory, ~110 YAML files)

- [ ] **Step 1: Verify no code references**

```bash
cd /c/Users/Calabe\ Davis/workspace/DOGMud
grep -rn "world/empty\|/empty/" internal/ main.go --include="*.go"
grep -n "empty" _datafiles/config.yaml
```

Expected: zero non-comment results from Go code. The config.yaml grep may hit unrelated uses of "empty" (e.g., "empty room"); scan each hit manually to confirm none are paths.

- [ ] **Step 2: List what's being deleted (for the commit message)**

```bash
find _datafiles/world/empty -type f | wc -l
du -sh _datafiles/world/empty
```

Note the file count and size for the commit message.

- [ ] **Step 3: Delete the directory**

```bash
rm -rf _datafiles/world/empty
```

- [ ] **Step 4: Verify it's gone**

```bash
ls _datafiles/world/
```

Expected: shows `default/` and `dogmud/` only, no `empty/`.

- [ ] **Step 5: Build**

```bash
go build ./...
```

Expected: clean build.

- [ ] **Step 6: Run server smoke test**

```bash
go run . &
SERVER_PID=$!
sleep 15
# Expect to see "Server Ready" in stdout
kill $SERVER_PID
wait $SERVER_PID 2>/dev/null
```

Or, if running `go run` in background is inconvenient, just run `go run .` in a terminal, wait for "Server Ready", and Ctrl+C. The server must start without errors about missing `empty` world files.

- [ ] **Step 7: Full test suite**

```bash
go test ./...
```

Expected: all pass.

- [ ] **Step 8: Commit**

```bash
git add -A _datafiles/world/empty
git commit -m "$(cat <<'EOF'
chore: remove unused _datafiles/world/empty/ directory

The empty world (~110 YAML files) was an old template for a
minimal world, never referenced by any Go code or config. The
configured world is _datafiles/world/dogmud, and the structural
template validator uses _datafiles/world/default. No callers, safe
to delete.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Final verification

**No files modified** — just verification that Stage 1.4 is complete.

- [ ] **Step 1: Confirm all GetScript references are gone**

```bash
cd /c/Users/Calabe\ Davis/workspace/DOGMud
grep -rn "GetScript\|HasScript\|GetScriptPath" internal/ --include="*.go" | grep -v "GetScriptingConfig"
```

Expected: zero results.

- [ ] **Step 2: Confirm Scripting config is gone**

```bash
grep -rn "GetScriptingConfig\|configs.Scripting\b" internal/ main.go --include="*.go"
ls internal/configs/config.scripting.go 2>&1
grep -c "^Scripting:" _datafiles/config.yaml
```

Expected:
- First grep: zero results
- `ls`: "No such file or directory"
- Third grep: 0

- [ ] **Step 3: Confirm empty world is gone**

```bash
ls -d _datafiles/world/empty 2>&1
```

Expected: "No such file or directory"

- [ ] **Step 4: Full build + test + server**

```bash
go build ./...
go vet ./...
go test ./...
```

All three must pass clean.

- [ ] **Step 5: Line counts**

Observe the savings:

```bash
# Lines removed from config.scripting.go (file was ~30 lines)
# Lines removed from 6 files containing GetScript methods (~140 total)
# Files removed from _datafiles/world/empty/ (~110 files)

git diff --stat $(git log --oneline | grep "dead code sweep" | tail -1 | awk '{print $1}')~1..HEAD -- internal/ _datafiles/ | tail -1
```

Note: this only works if the dead code sweep is a contiguous series of commits on this branch. Otherwise, count manually from the three task commits.

- [ ] **Step 6: No commit needed**

This task is verification only. If fixes are required, amend the relevant task's commit.

---

## Important constraints

- **Don't touch `util.Hash`.** It has legitimate callers in
  `Character.CacheDescription` (description dedup) and the SHA256→bcrypt
  password migration path.
- **Don't touch `_datafiles/world/default/`.** It's actively used by
  `ValidateWorldFiles()` in `internal/util/util.go` as a structural
  template at startup.
- **Don't touch YAML `script:` or `scriptPath:` fields** in data files.
  Those are dormant metadata — no Go code reads them, but cleaning
  them up would require a data migration out of scope for this stage.
- **Don't touch help templates.** Orphaned help templates are Stage 1.4b.
- **Don't bundle deletions into a mega-commit.** Each category gets its
  own commit so reverts are clean if a hidden caller surfaces.

## What success looks like

After this plan:
- `grep -rn "GetScript\|HasScript\|GetScriptPath" internal/ --include="*.go"` returns zero results
- `ls internal/configs/config.scripting.go` errors with "No such file"
- `ls _datafiles/world/empty/` errors with "No such file"
- Full build + tests pass
- Server starts cleanly

Estimated code removed: ~150 lines Go, ~30 lines YAML, ~110 YAML data files.
