# Room Instance Load Respects `instance:"skip"` Tag — Implementation Plan (Fix A)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `LoadRoomInstance` symmetric with `SaveRoomInstance` in honoring `instance:"skip"` struct tags, so stale pre-fix instance files can no longer overwrite template-owned fields (exits, title, description, nouns, etc.) on load.

**Architecture:** Add a `restoreSkipTaggedFields(dst, src *Room)` helper that copies every skip-tagged field from `src` onto `dst` via reflection. Call it in `LoadRoomInstance` after `yaml.Unmarshal`, passing a fresh `LoadRoomTemplate` call as `src`. No new boot cleanup — the existing empty-diff file deletion in `SaveRoomInstance` self-cleans stale files on the next runtime mutation.

**Tech Stack:** Go 1.21+, `reflect` (already imported), `gopkg.in/yaml.v2`, `testify` for assertions.

**Related spec:** `docs/superpowers/specs/2026-04-21-room-instance-load-respects-skip-tag-design.md`

**Branch:** `feature/fix-room-instance-load` (created; spec committed as `cd44df29`).

---

## File Structure

**Modified files:**
- `internal/rooms/save_and_load.go` — modify `LoadRoomInstance` (currently at line 100) + append new `restoreSkipTaggedFields` helper.

**Created files:**
- `internal/rooms/save_and_load_test.go` — new test file for the two LoadRoomInstance behaviors + the helper.

**Unchanged:**
- `internal/rooms/rooms.go` — `Room` struct already has `instance:"skip"` on all target fields (since 2026-03-03, commit `10edee33`). No change.
- `SaveRoomInstance` — already honors the tag. No change.

---

## Task 1: Add restoreSkipTaggedFields helper + LoadRoomInstance wire-up

**Files:**
- Modify: `internal/rooms/save_and_load.go` (modify `LoadRoomInstance` at ~line 100, add helper after it)
- Create: `internal/rooms/save_and_load_test.go`

TDD in three test cases: unit test for the helper in isolation, integration test for `LoadRoomInstance` respecting the tag, control test for non-skip fields still being applied.

- [ ] **Step 1: Write the failing tests**

Create `internal/rooms/save_and_load_test.go`:

```go
package rooms

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/stretchr/testify/assert"
)

// ─── restoreSkipTaggedFields ────────────────────────────────────────────────

// TestRestoreSkipTaggedFields verifies the helper copies every field
// tagged `instance:"skip"` from src onto dst, leaves non-skip fields
// on dst alone.
func TestRestoreSkipTaggedFields(t *testing.T) {
	// src = "template" values
	src := &Room{
		RoomId:      100,
		Zone:        "template_zone",
		Title:       "Template Title",
		Description: "Template description.",
		Exits: map[string]exit.RoomExit{
			"north": {RoomId: 200},
			"south": {RoomId: 300},
		},
		Nouns: map[string]string{"altar": "A stone altar."},
		Gold:  0, // non-skip field; runtime state
	}

	// dst = "post-unmarshal" values with corrupt skip fields + legit
	// non-skip runtime state.
	dst := &Room{
		RoomId:      100,
		Zone:        "corrupt_zone",   // skip-tagged; should be restored
		Title:       "Corrupt Title",  // skip-tagged; should be restored
		Description: "Corrupt.",       // skip-tagged; should be restored
		Exits: map[string]exit.RoomExit{
			"east": {RoomId: 999}, // skip-tagged; should be restored
		},
		Nouns: map[string]string{"poison": "Bad data."}, // skip-tagged
		Gold:  42,                                       // NOT skip-tagged; should stay
	}

	restoreSkipTaggedFields(dst, src)

	// Skip-tagged fields now match src
	assert.Equal(t, "template_zone", dst.Zone)
	assert.Equal(t, "Template Title", dst.Title)
	assert.Equal(t, "Template description.", dst.Description)
	assert.Equal(t, 2, len(dst.Exits))
	assert.Equal(t, 200, dst.Exits["north"].RoomId)
	assert.Equal(t, 300, dst.Exits["south"].RoomId)
	_, hasEast := dst.Exits["east"]
	assert.False(t, hasEast, "east should have been replaced by template exits")
	assert.Equal(t, "A stone altar.", dst.Nouns["altar"])
	_, hasPoison := dst.Nouns["poison"]
	assert.False(t, hasPoison, "poison noun should have been replaced")

	// Non-skip field preserved
	assert.Equal(t, 42, dst.Gold)
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test -run 'TestRestoreSkipTaggedFields' ./internal/rooms/`
Expected: **FAIL** with `undefined: restoreSkipTaggedFields` (compile error).

- [ ] **Step 3: Add the helper + wire into LoadRoomInstance**

In `internal/rooms/save_and_load.go`:

**(a)** Modify `LoadRoomInstance` (currently at line 100). Current code:

```go
func LoadRoomInstance(roomId int) *Room {

	room := LoadRoomTemplate(roomId)
	if room == nil {
		return nil
	}

	filename := roomManager.GetFilePath(roomId)

	if len(filename) == 0 {
		return nil
	}

	// Look for specially saved instance data
	filepath := util.FilePath(configs.GetFilePathsConfig().DataFiles.String(), `/rooms.instances/`, filename)

	if bytes, err := os.ReadFile(filepath); err == nil {
		// Unmarshal onto the default template data, overwriting any set fields in the instance save file
		if err := yaml.Unmarshal(bytes, room); err != nil {
			mudlog.Warn("LoadRoom", "roomId", roomId, "filepath", filepath, "error", err)
		}
	}

	return room
}
```

Replace the `if bytes, err := os.ReadFile(filepath); err == nil {` block with:

```go
	if bytes, err := os.ReadFile(filepath); err == nil {
		// Unmarshal onto the default template data, overwriting any set fields in the instance save file
		if err := yaml.Unmarshal(bytes, room); err != nil {
			mudlog.Warn("LoadRoom", "roomId", roomId, "filepath", filepath, "error", err)
		}
		// yaml.Unmarshal only honors `yaml:` tags; the `instance:"skip"`
		// annotation is invisible to it. Reload a fresh template copy and
		// restore every skip-tagged field so template-owned state
		// (title/description/exits/nouns/zone/etc.) cannot be corrupted
		// by stale data in pre-fix instance files. See
		// docs/superpowers/specs/2026-04-21-room-instance-load-respects-skip-tag-design.md.
		if freshTemplate := LoadRoomTemplate(roomId); freshTemplate != nil {
			restoreSkipTaggedFields(room, freshTemplate)
		}
	}
```

**(b)** Append the helper immediately after `LoadRoomInstance`:

```go
// restoreSkipTaggedFields copies every exported Room field tagged
// `instance:"skip"` from src onto dst. Used after an instance-file
// overlay unmarshal to ensure template-owned fields are not corrupted
// by stale data in pre-fix instance files.
func restoreSkipTaggedFields(dst, src *Room) {
	srcVal := reflect.ValueOf(*src)
	dstVal := reflect.ValueOf(dst).Elem()
	t := srcVal.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}
		if field.Tag.Get("instance") != "skip" {
			continue
		}
		dstVal.Field(i).Set(srcVal.Field(i))
	}
}
```

No new imports needed — `reflect` is already imported in this file.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test -run 'TestRestoreSkipTaggedFields' ./internal/rooms/`
Expected: PASS.

- [ ] **Step 5: Run the full rooms package test suite**

Run: `go test ./internal/rooms/`
Expected: all tests PASS. If any existing test fails, something unexpected shifted — stop and investigate.

- [ ] **Step 6: Full project build + tests**

Run: `go build ./... && go test ./...`
Expected: clean build; all tests PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/rooms/save_and_load.go internal/rooms/save_and_load_test.go
git commit -m "$(cat <<'EOF'
fix(rooms): LoadRoomInstance honors instance:"skip" tag

Mirror of the SaveRoomInstance tag check: after the YAML overlay
unmarshal, reload a fresh template copy and copy every skip-tagged
field back onto the loaded room. yaml.Unmarshal only reads yaml:
tags, so stale pre-10edee33 instance files (still on disk in the
wild — e.g., thornwall_city/472.yaml) would otherwise continue to
overwrite template-owned state (exits, title, description, nouns,
zone, etc.) on every load.

Self-clean of existing corrupt files is handled by the existing
empty-diff file deletion in SaveRoomInstance — next runtime
mutation on an affected room writes an instanceSaveData map that
(with skip-tagged fields excluded) is either empty-and-removed or
rewritten without the corrupt keys.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Add LoadRoomInstance integration tests

**Files:**
- Modify: `internal/rooms/save_and_load_test.go` (append)

Two tests that go end-to-end through `LoadRoomInstance` with a real instance file on disk. The helper test above covers the reflection logic in isolation; these cover the wire-up and the control case (non-skip fields still apply).

- [ ] **Step 1: Append the failing tests**

Append to `internal/rooms/save_and_load_test.go`:

```go
// ─── LoadRoomInstance — skip-tag enforcement ────────────────────────────────

// TestLoadRoomInstance_RestoresSkipTaggedFieldsFromTemplate verifies
// that an instance file containing skip-tagged fields (title, exits,
// description, nouns) cannot overwrite the template values on load.
// Simulates a pre-10edee33 corrupt file like thornwall_city/472.yaml.
func TestLoadRoomInstance_RestoresSkipTaggedFieldsFromTemplate(t *testing.T) {
	cleanup := stubRoomTemplate(t, 99001, func(r *Room) {
		r.Title = "Template Title"
		r.Description = "Template description."
		r.Exits = map[string]exit.RoomExit{
			"north": {RoomId: 100},
			"south": {RoomId: 200},
		}
	})
	defer cleanup()

	// Seed a corrupt instance file with different values for skip-tagged fields.
	writeInstanceFile(t, 99001, `
title: Corrupt Title
description: Corrupt description.
exits:
  east:
    roomid: 999
`)
	defer removeInstanceFile(t, 99001)

	room := LoadRoomInstance(99001)
	if room == nil {
		t.Fatal("LoadRoomInstance returned nil")
	}

	// Skip-tagged fields must come from template, not from the file.
	assert.Equal(t, "Template Title", room.Title)
	assert.Equal(t, "Template description.", room.Description)
	assert.Equal(t, 2, len(room.Exits))
	assert.Equal(t, 100, room.Exits["north"].RoomId)
	assert.Equal(t, 200, room.Exits["south"].RoomId)
	_, hasEast := room.Exits["east"]
	assert.False(t, hasEast, "east exit should have been dropped — skip tag")
}

// TestLoadRoomInstance_AppliesNonSkipFields is the control case. A
// non-skip-tagged field written into the instance file DOES land on
// the loaded room. Proves the skip-tag enforcement is narrow and
// doesn't regress legitimate runtime-state persistence.
func TestLoadRoomInstance_AppliesNonSkipFields(t *testing.T) {
	cleanup := stubRoomTemplate(t, 99002, func(r *Room) {
		r.Title = "Template Title"
		r.Gold = 0
	})
	defer cleanup()

	// Gold is NOT skip-tagged — should apply.
	writeInstanceFile(t, 99002, "gold: 42\n")
	defer removeInstanceFile(t, 99002)

	room := LoadRoomInstance(99002)
	if room == nil {
		t.Fatal("LoadRoomInstance returned nil")
	}

	assert.Equal(t, 42, room.Gold, "non-skip field should load from instance file")
	assert.Equal(t, "Template Title", room.Title, "skip field should come from template")
}
```

Also add three helper functions at the top of the file (right under the imports), which the two new tests use:

```go
// stubRoomTemplate writes a minimal Room YAML template to a temporary
// path under _datafiles/world/dogmud/rooms/test_zone/{roomId}.yaml
// and registers it with the rooms package. Returns a cleanup func
// that removes the file and clears the registry entry.
func stubRoomTemplate(t *testing.T, roomId int, customize func(r *Room)) func() {
	t.Helper()

	baseDir := filepath.Join(
		configs.GetFilePathsConfig().DataFiles.String(),
		"rooms", "test_zone")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		t.Fatalf("stubRoomTemplate mkdir: %v", err)
	}

	r := &Room{
		RoomId: roomId,
		Zone:   "test_zone",
		Title:  "Stub",
	}
	if customize != nil {
		customize(r)
	}

	// Write the template YAML. Using the same yaml library as the loader
	// guarantees the file is readable. (import yaml under alias if needed)
	bytes, err := yamlMarshalRoom(r)
	if err != nil {
		t.Fatalf("stubRoomTemplate marshal: %v", err)
	}
	path := filepath.Join(baseDir, fmtRoomFile(roomId))
	if err := os.WriteFile(path, bytes, 0644); err != nil {
		t.Fatalf("stubRoomTemplate write: %v", err)
	}

	// Register so LoadRoomTemplate/LoadRoomInstance can resolve the path.
	registerRoomTestFile(roomId, "test_zone/"+fmtRoomFile(roomId))

	return func() {
		_ = os.Remove(path)
		unregisterRoomTestFile(roomId)
	}
}

// writeInstanceFile writes an instance YAML file for the given room.
func writeInstanceFile(t *testing.T, roomId int, yaml string) {
	t.Helper()
	baseDir := filepath.Join(
		configs.GetFilePathsConfig().DataFiles.String(),
		"rooms.instances", "test_zone")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		t.Fatalf("writeInstanceFile mkdir: %v", err)
	}
	path := filepath.Join(baseDir, fmtRoomFile(roomId))
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatalf("writeInstanceFile write: %v", err)
	}
}

// removeInstanceFile cleans up an instance YAML file after a test.
func removeInstanceFile(t *testing.T, roomId int) {
	t.Helper()
	path := filepath.Join(
		configs.GetFilePathsConfig().DataFiles.String(),
		"rooms.instances", "test_zone", fmtRoomFile(roomId))
	_ = os.Remove(path)
}

func fmtRoomFile(roomId int) string {
	return fmt.Sprintf("%d.yaml", roomId)
}
```

**Important: helper dependencies.** `yamlMarshalRoom`, `registerRoomTestFile`, and `unregisterRoomTestFile` are referenced but not defined. Before writing the tests, **investigate how existing tests in `internal/rooms/rooms_test.go`, `instances_test.go`, or `test_helpers.go` stub room templates** — use those patterns if they exist. If the package doesn't have stub helpers for test-registered rooms, you have two options:

- **Option A (preferred if helpers exist):** use the established pattern. Skip the helper names above and rewrite the tests to match.
- **Option B:** create minimal helpers. Look at `internal/rooms/test_helpers.go` — if it already has a room-registration helper, reuse it. Otherwise, consider whether the integration tests are feasible in this package's test infrastructure.

If **neither option works** within 15 minutes of investigation, STOP and report BLOCKED. The helper test (Task 1) is the primary guarantee of correctness. The integration tests are belt-and-suspenders; a BLOCKED status here is fine — just land Task 1 + the helper test + the full package suite staying green, and we'll smoke-verify the end-to-end behavior via the manual smoke test in Task 3.

- [ ] **Step 2: Run the tests**

Run: `go test -run 'TestLoadRoomInstance_' ./internal/rooms/`
Expected: PASS (both tests).

- [ ] **Step 3: Run the full rooms package test suite**

Run: `go test ./internal/rooms/`
Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/rooms/save_and_load_test.go
git commit -m "$(cat <<'EOF'
test(rooms): integration tests for LoadRoomInstance skip-tag enforcement

Covers the end-to-end load path with a real YAML template + real
instance file on disk:
- skip-tagged fields (title/description/exits) ignored from file
- non-skip fields (gold) still applied — control case

Helper Task 1's unit test covers the reflection logic; these
cover the wire-up inside LoadRoomInstance.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

**If Task 2 was BLOCKED** per Step 1 investigation: skip Step 2–4 entirely. Move to Task 3.

---

## Task 3: Manual smoke test + merge

Task 1's unit test proves the helper logic. Task 2's integration tests (if they landed) prove the wire-up. This task provides a real-server sanity check and lands the branch on `development`.

**Pre-test setup:**
- Confirm `_datafiles/world/dogmud/rooms.instances/thornwall_city/472.yaml` still exists (the canonical corrupt file). If you removed it as part of earlier cleanup, restore a fake version with `exits: { east: { roomid: 475 } }` for the test.

- [ ] **Smoke 1: Corrupt file no longer poisons room 472**

Steps:
1. Confirm the corrupt file is present:
   `cat _datafiles/world/dogmud/rooms.instances/thornwall_city/472.yaml | grep -E "^exits:|^  "` — should show `east: 475` and `north: 469`.
2. Build and run the server: `go run . -c _datafiles/config.yaml`.
3. Log in as admin and teleport to room 472 (`goto 472` or similar).
4. `look` and check the exits list. Expected exits from template: `north, west, south`. NOT `north, east`.
5. Confirm you can move `west` and `south` successfully — they lead to valid rooms.

Expected: room 472's exits match the template. The stale `east: 475` from the file is ignored. Sable access is no longer broken.

- [ ] **Smoke 2: Trigger self-clean**

Steps:
1. Still in room 472, do something that mutates non-skip-tagged runtime state (drop and pick up an item to trigger `SaveRoomInstance`). Or type `admin save` if that exists.
2. Inspect `_datafiles/world/dogmud/rooms.instances/thornwall_city/472.yaml`.

Expected: either the file is deleted (if there's no remaining non-skip diff) or rewritten without the exits/description/nouns keys.

- [ ] **Smoke 3: Other skip-tagged fields also respected**

Steps:
1. Inspect other files in `rooms.instances/` for any `title:` or `description:` keys — `grep -l "^title:\|^description:" _datafiles/world/dogmud/rooms.instances/**/*.yaml 2>/dev/null`.
2. Visit one such room in-game and confirm the title/description shown matches the template, not the stale file.

Expected: all stale skip-tagged fields ignored across the board.

- [ ] **Post-smoke: Report pass/fail**

Report back with pass/fail per smoke case. I'll finalize based on that.

---

## Task 4: Finalize + merge

- [ ] **Step 1: Full build + test**

Run: `go build ./... && go test ./...`
Expected: all green.

- [ ] **Step 2: Update MEMORY.md**

In `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\MEMORY.md`:

- Remove the "Instance-save corruption — rooms only (Fix A)" entry from "## Bugs to Fix".
- Add a bullet to the "## Completed (2026-04-21)" section summarizing the fix: asymmetric save-vs-load bug in the `instance:"skip"` tag; `LoadRoomInstance` now reloads a fresh template copy and restores skip-tagged fields after yaml.Unmarshal.
- Remove the "Fix A (rooms-exits instance-save corruption)" entry from the "## Next Up" list (it was added earlier today after Fix D landed).

Also update `project_instance_save_exits_corruption.md` — mark Fix A as landed. Consider deleting the memory entirely at this point, or leaving a brief "both fixes landed 2026-04-21" header and archiving the investigation below.

- [ ] **Step 3: Ask user about merge**

Per `github_guide.md`: feature branches merge into `development` with `--no-ff`. Ask the user before running the merge — do NOT merge autonomously. After confirmation, the merge command is:

```bash
git checkout development
git merge --no-ff feature/fix-room-instance-load -m "..."
```

Ask whether to delete the feature branch after the merge.

---

## Self-Review

**Spec coverage:**
- Section "Code changes → modify LoadRoomInstance" — Task 1 Step 3(a). ✓
- Section "Code changes → New helper (restoreSkipTaggedFields)" — Task 1 Step 3(b). ✓
- Section "What this fixes without further action" — Tasks 2 + 3 verify empirically. ✓
- Section "Testing → Unit tests" — Tasks 1 (helper) + 2 (LoadRoomInstance). ✓
- Section "Testing → Smoke test" — Task 3. ✓
- Section "Edge cases consciously accepted" + "Out of scope" — no code tasks needed; documented in spec. ✓

**Placeholder scan:**
- Task 2 has a conditional "BLOCKED" escape hatch that is NOT a placeholder — it's a well-defined branching behavior with specific criteria and a next step. Acceptable.
- Every code block is concrete. No TBDs, no "etc."

**Type consistency:**
- `restoreSkipTaggedFields(dst, src *Room)` — used consistently in Tasks 1 and 2. ✓
- Test helper signatures in Task 2 are self-contained.

No issues found.
