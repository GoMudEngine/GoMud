# Code Cleanup 1.4: Dead Code Sweep (Go code) — Design Spec

## Goal

Remove Go code made dead by the JS scripting bridge removal in
Phase 5. Three categories: orphaned `GetScript`/`HasScript`/
`GetScriptPath` methods across 6 packages, the unused `Scripting`
config struct, and the unused `_datafiles/world/empty/` world
directory.

Pure dead-code removal — no behavior change.

## Scope

**In scope:**
- Remove `GetScript`/`HasScript`/`GetScriptPath` methods from
  `internal/buffs`, `internal/spells`, `internal/rooms`,
  `internal/mobs`, `internal/items`
- Remove `Scripting` config struct and `GetScriptingConfig()`
- Remove `Scripting:` block from `_datafiles/config.yaml`
- Remove `Scripting Scripting` field from `configs.Config` struct
- Remove `c.Scripting.Validate()` call from `Config.Validate()`
- Delete `_datafiles/world/empty/` directory entirely

**Out of scope:**
- Help template audit (Stage 1.4b)
- Removing `script:` or `scriptPath:` YAML fields from data files
  (those are data, not Go code)
- `util.Hash` removal (still used by description cache and
  password migration path)
- `_datafiles/world/default/` (actively referenced by
  `ValidateWorldFiles`)

## Categories

### Category 1: Orphaned GetScript methods

**Every single `GetScript`/`HasScript`/`GetScriptPath` method in the
codebase has zero non-test callers.** Verified via grep:
```bash
grep -rn "GetScript\|HasScript\|GetScriptPath" internal/ --include="*.go"
```
Only the definitions themselves and self-calls (e.g., `GetScript()`
calls `GetScriptPath()`) appear. Nothing else uses them.

**Methods to delete:**

| Package | File | Methods |
|---------|------|---------|
| `buffs` | `buffspec.go` | `GetScript()`, `GetScriptPath()` |
| `spells` | `spells.go` | `GetScript()`, `GetScriptPath()` |
| `rooms` | `rooms.go` | `GetScript()`, `GetScriptPath()` |
| `mobs` | `mobs.go` | `GetScript()`, `GetScriptPath()`, `HasScript()` |
| `items` | `items.go` | `GetScript()` |
| `items` | `itemspec.go` | `GetScript()`, `GetScriptPath()` |

**Also remove any tests** that exclusively test these methods. If a
test function has other assertions mixed in, keep those assertions
and only remove the `GetScript`-related ones.

### Category 2: Dead Scripting config

The `Scripting` config struct was used by the goja JS VM to set
per-script timeouts. With JS removed, nothing reads these fields.

**Files to modify:**

- **Delete entirely:** `internal/configs/config.scripting.go` (contains
  the `Scripting` struct, its `Validate()`, and `GetScriptingConfig()`)
- **Modify:** `internal/configs/configs.go`
  - Remove `Scripting Scripting` field from `Config` struct (~line 47)
  - Remove `c.Scripting.Validate()` call from `Config.Validate()` (~line 205)
- **Modify:** `_datafiles/config.yaml`
  - Remove the entire `Scripting:` section and its fields
  - Remove the `# SCRIPTING` comment block header

### Category 3: Unused empty world directory

`_datafiles/world/empty/` contains ~110 YAML files that duplicate
the structure of `default/` and `dogmud/`. No Go code references
this directory.

**Verification:** `grep -rn "world/empty" internal/ _datafiles/config.yaml main.go` returns only doc references, no code.

**Action:** Delete the entire `_datafiles/world/empty/` directory.

## Architecture

This is a deletion-only change. No new code. No refactored
interfaces. Each removal is verified by:
1. Grep confirms zero non-test callers
2. `go build ./...` succeeds after removal
3. `go test ./...` passes

Since every removal is independent, they can be grouped by category
and each category gets its own commit for clear history.

## Constraints

- **Zero behavior change.** Nothing depends on the removed code.
- **Build must pass after each category.** Don't batch all deletions
  into one commit — category-level commits make reverting easier
  if we discover a hidden caller.
- **Don't touch data-file `script:` fields.** Items, mobs, spells,
  rooms, and buffs may have `script:` or `scriptPath:` fields in
  their YAML definitions. Leaving those alone — they're dormant
  metadata that no Go code reads.
- **Don't touch util.Hash.** Still needed for character description
  caching and the SHA256→bcrypt migration path.

## Testing

After each category:

```bash
cd /c/Users/Calabe\ Davis/workspace/DOGMud
go build ./...
go vet ./...
go test ./...
```

After Category 3 (directory deletion), also verify the server starts:

```bash
go run .
# Expect: "Server Ready" without errors
# Kill with Ctrl+C
```

## Risk Assessment

**Low risk.** Every item in scope has been verified dead via grep.
The only way this breaks something:
- A reflection-based or string-based lookup (e.g., a `reflect.ValueOf`
  finding a method by name). Unlikely in this codebase — we don't
  use reflection in hot paths.
- An external consumer of the Go API (e.g., a plugin) that imports
  our packages. DOGMud is not a library, so this doesn't apply.
- The `_datafiles/world/empty/` directory being referenced by a
  deployment script or external tool. Verified: grep returned no
  references.

Mitigated by: category-level commits allow easy revert if something
breaks on prod.

## Execution Plan

Each task is one category, one commit:

**Task 1: Remove GetScript methods (Category 1)**
- Delete methods from 6 files across 5 packages
- Remove any test functions that exclusively tested them
- Build + test + commit

**Task 2: Remove Scripting config (Category 2)**
- Delete `config.scripting.go`
- Edit `configs.go` (remove field + Validate call)
- Edit `config.yaml` (remove block)
- Build + test + commit

**Task 3: Delete empty world directory (Category 3)**
- `rm -rf _datafiles/world/empty/`
- Build + test + smoke-test server startup
- Commit

**Task 4: Final verification**
- Grep confirms zero `GetScript`/`HasScript`/`GetScriptPath`
  references remain
- Grep confirms zero `Scripting` config references
- Full build + test suite
- Manual server startup smoke test

## What we're NOT doing

Stated explicitly because these may come up:

- **Help template audit** — Stage 1.4b handles this. Orphaned help
  templates are a separate category.
- **YAML `script:` field cleanup** — Data files may have dormant
  script references. Removing them requires a data migration and
  risks breaking world file loading. Deferred indefinitely; harmless
  as-is.
- **`util.Hash` removal** — Still has legitimate callers
  (description cache, password migration). Cannot remove until both
  are replaced.
- **`_datafiles/world/default/` cleanup** — `ValidateWorldFiles` in
  `internal/util/util.go` reads this directory as a structural
  template. Required, keep it.

## Success criteria

After this stage completes:
- `grep -rn "GetScript\|HasScript\|GetScriptPath" internal/ --include="*.go"`
  returns zero results
- `grep -rn "Scripting" internal/configs/ --include="*.go"` returns
  zero results
- `ls _datafiles/world/` does not show `empty/`
- Server starts and runs normally
- All tests pass
