# Zone Rename — Phase 2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rename a zone from the `/build` Zones panel — moving all ten directories it owns, rewriting the zone name inside every file that stores it, and rekeying the in-memory caches — refusing while any player is in the zone.

**Architecture:** `rooms.RenameZone` executes **validate → dry-run → move → rewrite → rekey** on MainWorker, recording a move manifest so a mid-flight failure can be reversed best-effort. A new `Build.Zone.Rename` verb sits behind the existing `zoneDeps` seam. Content rewriting is a **targeted line edit**, never a YAML re-marshal.

**Tech Stack:** Go, `internal/fileloader`, testify `assert` in `internal/rooms`, plain `testing` in `modules/gmcp`, vanilla JS in `zones.js`.

**Spec:** `docs/superpowers/specs/2026-07-25-zone-lifecycle-config-design.md` §4

**Phase 1 is merged (`a245a63d2`).** This builds on it.

---

## Verified reference — what a rename must touch

Established by inspection of the live world before planning. **Do not re-derive; do not assume symmetry.**

**A zone owns ten directories**, each named by the sanitized zone name
(`rooms.ZoneToFolder`). `rooms.zoneAllDirs()` already returns exactly this list:

```
rooms  rooms.instances  mobs  mobs.instances  dialogue
behaviors  schedules  caravans  foragers  shops
```

**Only FIVE of them store the zone name inside file content** — a top-level
`zone:` key, in practice on line 2:

| tree | `zone:` in content? |
|------|--------------------|
| `rooms/` | **yes** |
| `mobs/` | **yes** |
| `dialogue/` | **yes** |
| `caravans/` | **yes** |
| `foragers/` | **yes** |
| `behaviors/`, `schedules/`, `shops/`, both `.instances/` | no — folder-only |

Moving the directory is sufficient for the folder-only five. The other five
need both the move and a content rewrite, or their loaded objects keep the old
zone name and the world splits in half.

**In-memory state that must be rekeyed:**

- `roomManager.zones` — delete the old key, insert the config under the new one, set `cfg.Name`.
- `roomManager.rooms[id].Zone` — every loaded room in the zone.
- `roomManager.roomIdToFileCache[id]` — stores `<zoneFolder>/<id>.yaml`. Stale entries point at the pre-move path. `GetFilePath` self-heals by walking the whole rooms tree when an entry is missing (`searchForRoomFile`), so **deleting** the entries is correct but costs a tree walk per room; rewriting the prefix is cheap and precise. This plan rewrites.
- Mob templates' in-memory `.Zone`. **`mobNameCache` is keyed MobId→name and is zone-independent** (`mobs.go:46,1163`), and `Mob.Filepath()` derives the folder from `m.Zone` at call time (`mobs.go:1185`) — so there is no zone-keyed mob cache to invalidate.
- `mapper.ClearCache()` — **at the GMCP layer**, not in `rooms`. `internal/rooms` cannot import `internal/mapper` (mapper imports rooms).

**Known hazard:** `internal/mobs` has its own unqualified `ZoneNameSanitize`
(`mobs.go:1186`) separate from `rooms.ZoneNameSanitize`. They must agree or a
renamed mob lands in a different folder than its room. Task 6 asserts this.

**Signatures verified present:**

```go
rooms.ValidateZoneName(zone string) error   // roommanager.go:636 — returns nil for ""
rooms.ZoneFolderCollision(newName string, existing []string) string  // Phase 1
rooms.ZoneNameSanitize(zone string) string  // roommanager.go:620
rooms.ZoneToFolder(zone string) string      // roommanager.go:630
rooms.zoneAllDirs() []string                // Phase 1, unexported
rooms.GetAllZoneNames() []string
rooms.GetZoneConfig(zone string) *ZoneConfig
rooms.SaveZoneConfig(cfg *ZoneConfig) error
rooms.GetAllRoomIds() []int
rooms.LoadRoomTemplate(roomId int) *Room    // reads from DISK every call
(r *Room) GetPlayers(...FindFlag) []int     // rooms.go:1490
mapper.ClearCache()                          // mapper.go:110
```

## Non-goals (locked by the spec — do not implement)

- **Player fog-of-war is NOT migrated.** `Character.VisitedRooms` is keyed by
  zone display name in every player save. Players re-explore the renamed zone
  on the web map. This is a recorded decision, not an oversight.
- No renaming while any player is in the zone.
- No rename of the in-game `build zone` command's behaviour.

---

## File Structure

| File | Responsibility |
|------|----------------|
| `internal/rooms/zone_rename.go` **(create)** | `ZoneRenameBlockers`, `ValidateZoneRename`, `planZoneRename`/`zoneMoveManifest`, `rewriteZoneField`, `RenameZone` |
| `internal/rooms/zone_rename_test.go` **(create)** | Unit + filesystem tests |
| `modules/gmcp/gmcp.Zone.go` **(modify)** | `zoneDeps.rename`, `zoneRenameReq`, `buildZoneRename` |
| `modules/gmcp/gmcp.Zone_test.go` **(modify)** | Handler tests |
| `modules/gmcp/gmcp.Build.go` **(modify)** | `Build.Zone.Rename` dispatch + `mapper.ClearCache()` |
| `modules/gmcp/gmcp.go` **(modify)** | Route + admin-gate the verb |
| `_datafiles/html/public/static/js/zones.js` **(modify)** | Rename control |

---

### Task 1: Rename blockers (players in zone)

**Files:** Create `internal/rooms/zone_rename.go`, `internal/rooms/zone_rename_test.go`

- [ ] **Step 1: Write the failing test**

```go
package rooms

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZoneRenameBlockers_ReportsPlayersOnly(t *testing.T) {
	// Rooms, mobs and content do NOT block a rename — only players, because
	// their in-memory room pointers and the file move would race.
	src := zoneRenameSources{
		playersInZone: func(z string) []string { return []string{"2 player(s) in room 101"} },
	}
	got := ZoneRenameBlockersWith("Testzone", src)
	assert.Len(t, got, 1)
	assert.Equal(t, "player", got[0].Kind)
}

func TestZoneRenameBlockers_QuietZoneIsRenameable(t *testing.T) {
	src := zoneRenameSources{
		playersInZone: func(z string) []string { return nil },
	}
	assert.Empty(t, ZoneRenameBlockersWith("Testzone", src))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/rooms/ -run TestZoneRenameBlockers -v`
Expected: FAIL — `undefined: zoneRenameSources`

- [ ] **Step 3: Write minimal implementation**

Create `internal/rooms/zone_rename.go`:

```go
package rooms

// zoneRenameSources injects the world lookups the rename guard needs, so the
// policy is testable without a loaded world.
type zoneRenameSources struct {
	playersInZone func(zone string) []string
}

// ZoneRenameBlockersWith reports why a zone cannot be renamed right now.
//
// Only players block. Rooms, mobs and authored content are all rewritten by
// the rename itself; a player standing in the zone is different, because the
// files move under their feet and their in-memory room pointer would go stale.
func ZoneRenameBlockersWith(zone string, src zoneRenameSources) []ZoneBlocker {
	out := []ZoneBlocker{}
	for _, p := range src.playersInZone(zone) {
		out = append(out, ZoneBlocker{Kind: "player", Id: p})
	}
	return out
}

// ZoneRenameBlockers is the production wiring.
func ZoneRenameBlockers(zone string) []ZoneBlocker {
	return ZoneRenameBlockersWith(zone, zoneRenameSources{
		playersInZone: func(z string) []string {
			out := []string{}
			for _, id := range GetAllRoomIds() {
				r := LoadRoomTemplate(id)
				if r == nil || r.Zone != z {
					continue
				}
				if n := len(r.GetPlayers()); n > 0 {
					out = append(out, fmt.Sprintf("%d player(s) in room %d", n, id))
				}
			}
			return out
		},
	})
}
```

Add `import "fmt"`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/rooms/ -run TestZoneRenameBlockers -v`
Expected: PASS (both)

- [ ] **Step 5: Commit**

```bash
git add internal/rooms/zone_rename.go internal/rooms/zone_rename_test.go
git commit -m "feat(rooms): zone rename blocker policy"
```

---

### Task 2: Rename validation

**Files:** Modify `internal/rooms/zone_rename.go`, `internal/rooms/zone_rename_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestValidateZoneRename(t *testing.T) {
	existing := []string{"Amber Valley", "Stillwater"}

	// Happy path.
	assert.NoError(t, ValidateZoneRename("Stillwater", "Quiet Water", existing))

	// Empty / too short. ValidateZoneName returns nil for "", so the emptiness
	// check must be our own.
	assert.Error(t, ValidateZoneRename("Stillwater", "", existing))
	assert.Error(t, ValidateZoneRename("Stillwater", "Q", existing))

	// Illegal characters (ValidateZoneName allows letters/digits/space/_ only).
	assert.Error(t, ValidateZoneRename("Stillwater", "Bad/Name", existing))

	// Renaming to an existing zone.
	assert.Error(t, ValidateZoneRename("Stillwater", "Amber Valley", existing))

	// Different display name that sanitizes onto a LIVE zone's folder.
	assert.Error(t, ValidateZoneRename("Stillwater", "Amber_Valley", existing))

	// Renaming a zone to a different capitalisation of ITSELF is allowed —
	// it collides only with its own folder, which is the one being moved.
	assert.NoError(t, ValidateZoneRename("Stillwater", "StillWater", existing))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/rooms/ -run TestValidateZoneRename -v`
Expected: FAIL — `undefined: ValidateZoneRename`

- [ ] **Step 3: Write minimal implementation**

Append to `internal/rooms/zone_rename.go`:

```go
// ValidateZoneRename checks a proposed new zone name against the existing set.
//
// The folder check matters as much as the name check: ZoneNameSanitize only
// lowercases and turns spaces into underscores, so "Amber Valley" and
// "Amber_Valley" are different display names occupying the SAME directory.
// Renaming onto a live zone's folder would collide on disk.
func ValidateZoneRename(oldName, newName string, existing []string) error {
	newName = strings.TrimSpace(newName)
	if len(newName) < 2 {
		return errors.New("zone name must be at least 2 characters")
	}
	if err := ValidateZoneName(newName); err != nil {
		return err
	}
	if newName == oldName {
		return errors.New("new name is the same as the current name")
	}
	for _, z := range existing {
		if z == newName {
			return fmt.Errorf("zone %q already exists", newName)
		}
	}
	// Exclude the zone being renamed: its own folder is the one moving, so
	// re-casing a name (Stillwater -> StillWater) is legal.
	others := make([]string, 0, len(existing))
	for _, z := range existing {
		if z != oldName {
			others = append(others, z)
		}
	}
	if clash := ZoneFolderCollision(newName, others); clash != "" {
		return fmt.Errorf("zone folder %q is already used by zone %q", ZoneNameSanitize(newName), clash)
	}
	return nil
}
```

Add `errors` and `strings` to the imports.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/rooms/ -run TestValidateZoneRename -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/rooms/zone_rename.go internal/rooms/zone_rename_test.go
git commit -m "feat(rooms): zone rename validation incl. sanitized-folder collision"
```

---

### Task 3: The `zone:` content rewrite

The single most dangerous edit in this plan. A YAML re-marshal would reflow
every room description in the zone. This rewrites one line and touches nothing
else.

**Files:** Modify `internal/rooms/zone_rename.go`, `internal/rooms/zone_rename_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestRewriteZoneField_TouchesOnlyTheTopLevelKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "room.yaml")

	// CRLF, a description mentioning "zone:" inside a block scalar, and a
	// nested key that also ends in "zone:" — none of which may be rewritten.
	original := "roomid: 5\r\n" +
		"zone: Old Name\r\n" +
		"title: A Room\r\n" +
		"description: >-\r\n" +
		"  A sign reads: zone: Old Name. The prose must survive verbatim.\r\n" +
		"nouns:\r\n" +
		"  subzone: something\r\n"
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	if err := rewriteZoneField(path, "New Name"); err != nil {
		t.Fatalf("rewriteZoneField: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(got)

	assert.Contains(t, s, "zone: New Name\r\n", "top-level zone must be rewritten")
	assert.Contains(t, s, "A sign reads: zone: Old Name.", "prose must be untouched")
	assert.Contains(t, s, "  subzone: something", "nested keys must be untouched")
	assert.Contains(t, s, "\r\n", "CRLF line endings must be preserved")
	assert.NotContains(t, s, "zone: Old Name\r\n", "old top-level value must be gone")
}

func TestRewriteZoneField_NoTopLevelZoneIsNotAnError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "schedule.yaml")
	original := "id: arn\r\nsegments: []\r\n"
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}
	// behaviors/schedules/shops carry no zone: key. Rewriting must be a no-op,
	// not a failure.
	assert.NoError(t, rewriteZoneField(path, "New Name"))
	got, _ := os.ReadFile(path)
	assert.Equal(t, original, string(got))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/rooms/ -run TestRewriteZoneField -v`
Expected: FAIL — `undefined: rewriteZoneField`

- [ ] **Step 3: Write minimal implementation**

Append to `internal/rooms/zone_rename.go`:

```go
// rewriteZoneField replaces the value of the top-level `zone:` key in a YAML
// file, leaving every other byte alone.
//
// This is deliberately a text edit, not a load-and-re-marshal. Round-tripping
// a room through yaml reflows every description and noun block (`>-` becomes
// `|`, wrapping changes), producing an enormous diff for a one-word change.
//
// "Top-level" means column zero: a `zone:` inside a description block scalar,
// or a nested `subzone:`, must survive untouched. Only the first match is
// replaced — a YAML document cannot legally have two top-level `zone:` keys,
// and stopping at the first avoids corrupting anything odd further down.
//
// A file with no top-level `zone:` is not an error: five of the ten zone trees
// (behaviors, schedules, shops, and both .instances) are folder-only.
func rewriteZoneField(path, newZone string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(raw)

	eol := "\n"
	if strings.Contains(content, "\r\n") {
		eol = "\r\n"
	}

	lines := strings.Split(content, eol)
	for i, line := range lines {
		// Column zero only — no leading whitespace.
		if strings.HasPrefix(line, "zone:") {
			lines[i] = "zone: " + newZone
			return os.WriteFile(path, []byte(strings.Join(lines, eol)), 0644)
		}
	}
	return nil // folder-only tree; nothing to rewrite
}
```

Add `os` and `path/filepath` to imports as needed; the test needs `os` and `path/filepath` too.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/rooms/ -run TestRewriteZoneField -v`
Expected: PASS (both)

- [ ] **Step 5: Commit**

```bash
git add internal/rooms/zone_rename.go internal/rooms/zone_rename_test.go
git commit -m "feat(rooms): targeted zone: line rewrite that never reflows prose"
```

---

### Task 4: Move manifest + dry run

**Files:** Modify `internal/rooms/zone_rename.go`, `internal/rooms/zone_rename_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestPlanZoneRename_SkipsAbsentTreesAndDetectsCollision(t *testing.T) {
	base := t.TempDir()
	// Only three of the ten trees exist for this zone.
	for _, d := range []string{"rooms", "mobs", "shops"} {
		if err := os.MkdirAll(filepath.Join(base, d, "old_zone"), 0755); err != nil {
			t.Fatal(err)
		}
	}

	mv, err := planZoneRename(base, "Old Zone", "New Zone")
	assert.NoError(t, err)
	assert.Len(t, mv, 3, "only existing trees are planned")

	// A pre-existing target directory must abort the whole plan.
	if err := os.MkdirAll(filepath.Join(base, "rooms", "new_zone"), 0755); err != nil {
		t.Fatal(err)
	}
	_, err = planZoneRename(base, "Old Zone", "New Zone")
	assert.Error(t, err, "an occupied target path must abort before anything moves")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/rooms/ -run TestPlanZoneRename -v`
Expected: FAIL — `undefined: planZoneRename`

- [ ] **Step 3: Write minimal implementation**

```go
// zoneMove is one directory rename in a zone-rename manifest.
type zoneMove struct {
	From string
	To   string
}

// planZoneRename builds the move manifest and verifies every target path is
// free BEFORE anything is touched, so a doomed rename aborts having changed
// nothing. Trees that do not exist for this zone are skipped, not created.
func planZoneRename(base, oldName, newName string) ([]zoneMove, error) {
	oldFolder := ZoneNameSanitize(oldName)
	newFolder := ZoneNameSanitize(newName)

	out := []zoneMove{}
	for _, d := range zoneAllDirs() {
		from := util.FilePath(base, "/", d, "/", oldFolder)
		if _, err := os.Stat(from); err != nil {
			continue // this tree does not exist for this zone
		}
		to := util.FilePath(base, "/", d, "/", newFolder)
		if _, err := os.Stat(to); err == nil {
			return nil, fmt.Errorf("target path already exists: %s", to)
		}
		out = append(out, zoneMove{From: from, To: to})
	}
	return out, nil
}

// applyZoneMoves renames each planned directory, returning the moves that
// succeeded so a failure can be reversed.
func applyZoneMoves(mv []zoneMove) (done []zoneMove, err error) {
	for _, m := range mv {
		if err = os.Rename(m.From, m.To); err != nil {
			return done, fmt.Errorf("moving %s -> %s: %w", m.From, m.To, err)
		}
		done = append(done, m)
	}
	return done, nil
}

// reverseZoneMoves undoes completed moves, best effort. It reports what it
// could NOT undo — a half-renamed zone must be loud, never silent.
func reverseZoneMoves(done []zoneMove) []string {
	stuck := []string{}
	for i := len(done) - 1; i >= 0; i-- {
		if err := os.Rename(done[i].To, done[i].From); err != nil {
			stuck = append(stuck, done[i].To)
		}
	}
	return stuck
}
```

- [ ] **Step 4: Widen the import block**

`planZoneRename` introduces `util.FilePath`, and Task 5 will add
`configs.GetFilePathsConfig`. By the end of Task 5 `internal/rooms/zone_rename.go`
needs exactly:

```go
import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/util"
)
```

Match these import paths against the existing block in
`internal/rooms/zone_lifecycle.go`, which already imports both `configs` and
`util`. The TEST file additionally needs `os` and `path/filepath` (Task 3), and
`configs` + `util` for Task 5's filesystem test.

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/rooms/ -run TestPlanZoneRename -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/rooms/zone_rename.go internal/rooms/zone_rename_test.go
git commit -m "feat(rooms): zone rename move manifest with pre-flight target check"
```

---

### Task 5: `RenameZone` orchestration

**Files:** Modify `internal/rooms/zone_rename.go`

- [ ] **Step 1: Write the failing filesystem test**

```go
func TestRenameZone_MovesRewritesAndRekeys(t *testing.T) {
	if os.Getenv(`DOGMUD_BOOT_SMOKE`) == `` {
		t.Skip("set DOGMUD_BOOT_SMOKE=1 to run the filesystem rename test")
	}
	// Test binaries run with CWD = their own package dir; ReloadConfig reads
	// _datafiles/config.yaml relative to CWD. Same dance as
	// internal/web/auth_test.go and TestDeleteZone_RemovesEveryTree.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := configs.ReloadConfig(); err != nil {
		t.Fatal(err)
	}

	const oldName = "Rename Probe Zone"
	const newName = "Renamed Probe Zone"
	t.Cleanup(func() {
		base := configs.GetFilePathsConfig().DataFiles.String()
		for _, d := range zoneAllDirs() {
			_ = os.RemoveAll(util.FilePath(base, "/", d, "/", ZoneNameSanitize(oldName)))
			_ = os.RemoveAll(util.FilePath(base, "/", d, "/", ZoneNameSanitize(newName)))
		}
		delete(roomManager.zones, oldName)
		delete(roomManager.zones, newName)
	})

	roomId, err := CreateZone(oldName)
	assert.NoError(t, err)

	assert.NoError(t, RenameZone(oldName, newName))

	base := configs.GetFilePathsConfig().DataFiles.String()
	oldDir := util.FilePath(base, "/", "rooms", "/", ZoneNameSanitize(oldName))
	newDir := util.FilePath(base, "/", "rooms", "/", ZoneNameSanitize(newName))
	_, oldErr := os.Stat(oldDir)
	assert.True(t, os.IsNotExist(oldErr), "old rooms dir must be gone")
	_, newErr := os.Stat(newDir)
	assert.NoError(t, newErr, "new rooms dir must exist")

	// Content rewritten.
	r := LoadRoomTemplate(roomId)
	assert.NotNil(t, r)
	assert.Equal(t, newName, r.Zone, "room zone: must be rewritten")

	// Caches rekeyed.
	assert.NotContains(t, GetAllZoneNames(), oldName)
	assert.Contains(t, GetAllZoneNames(), newName)
	if cfg := GetZoneConfig(newName); assert.NotNil(t, cfg) {
		assert.Equal(t, newName, cfg.Name)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `DOGMUD_BOOT_SMOKE=1 go test ./internal/rooms/ -run TestRenameZone_MovesRewritesAndRekeys -v`
Expected: FAIL — `undefined: RenameZone`

- [ ] **Step 3: Write minimal implementation**

```go
// contentZoneDirs are the trees whose FILES store the zone name in a top-level
// `zone:` key. The other five trees a zone owns are folder-only, so moving the
// directory is all they need. Verified against the live world 2026-07-25.
func contentZoneDirs() []string {
	return []string{"rooms", "mobs", "dialogue", "caravans", "foragers"}
}

// RenameZone moves every directory a zone owns, rewrites the zone name inside
// the files that store it, and rekeys the in-memory caches.
//
// Order is validate -> plan -> move -> rewrite -> rekey. The plan verifies all
// target paths are free first, so an impossible rename changes nothing. If a
// move fails partway the completed moves are reversed; if a reversal ALSO
// fails the error names the directories left stranded, because a half-renamed
// zone must never be reported as success.
//
// The caller is responsible for mapper.ClearCache() afterwards; internal/rooms
// cannot import internal/mapper.
func RenameZone(oldName, newName string) error {
	cfg, ok := roomManager.zones[oldName]
	if !ok {
		return fmt.Errorf("zone %q does not exist", oldName)
	}
	if err := ValidateZoneRename(oldName, newName, GetAllZoneNames()); err != nil {
		return err
	}
	if b := ZoneRenameBlockers(oldName); len(b) > 0 {
		return fmt.Errorf("zone %q cannot be renamed right now (%d blockers)", oldName, len(b))
	}

	// Collect the zone's rooms BEFORE the move — LoadRoomTemplate reads from
	// disk, so afterwards the old paths are gone.
	roomIds := []int{}
	for _, id := range GetAllRoomIds() {
		if r := LoadRoomTemplate(id); r != nil && r.Zone == oldName {
			roomIds = append(roomIds, id)
		}
	}

	base := configs.GetFilePathsConfig().DataFiles.String()
	planned, err := planZoneRename(base, oldName, newName)
	if err != nil {
		return err
	}
	done, err := applyZoneMoves(planned)
	if err != nil {
		if stuck := reverseZoneMoves(done); len(stuck) > 0 {
			return fmt.Errorf("%w; ALSO FAILED TO REVERSE, these directories are stranded: %v", err, stuck)
		}
		return err
	}

	// Rewrite the zone name inside every file that stores it.
	newFolder := ZoneNameSanitize(newName)
	for _, d := range contentZoneDirs() {
		dir := util.FilePath(base, "/", d, "/", newFolder)
		entries, readErr := os.ReadDir(dir)
		if readErr != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
				continue
			}
			if err := rewriteZoneField(util.FilePath(dir, "/", e.Name()), newName); err != nil {
				return fmt.Errorf("rewriting %s: %w (directories are already moved — the zone is half-renamed)", e.Name(), err)
			}
		}
	}

	// Rekey in-memory state.
	cfg.Name = newName
	delete(roomManager.zones, oldName)
	roomManager.zones[newName] = cfg

	oldFolder := ZoneNameSanitize(oldName)
	for _, id := range roomIds {
		if r, ok := roomManager.rooms[id]; ok && r != nil {
			r.Zone = newName
		}
		// roomIdToFileCache stores "<zoneFolder>/<id>.yaml"; a stale entry
		// points at the pre-move path. GetFilePath would self-heal by walking
		// the whole rooms tree, but rewriting the prefix is cheap and exact.
		if p, ok := roomManager.roomIdToFileCache[id]; ok {
			roomManager.roomIdToFileCache[id] = strings.Replace(p, oldFolder, newFolder, 1)
		}
	}

	return SaveZoneConfig(cfg)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `DOGMUD_BOOT_SMOKE=1 go test ./internal/rooms/ -run TestRenameZone_MovesRewritesAndRekeys -v`
Expected: PASS

If it fails leaving directories behind, remove
`_datafiles/world/dogmud/*/rename[d]_probe_zone/` by hand before re-running.

- [ ] **Step 5: Run the whole package**

Run: `go test ./internal/rooms/`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/rooms/zone_rename.go internal/rooms/zone_rename_test.go
git commit -m "feat(rooms): RenameZone — move, rewrite, rekey"
```

---

### Task 6: Mob zone agreement guard

`internal/mobs` has its own unqualified `ZoneNameSanitize` (`mobs.go:1186`),
separate from `rooms.ZoneNameSanitize`. If they ever disagree, a renamed mob
lands in a different folder than its rooms and silently vanishes from the zone.

**Files:** Modify `internal/rooms/zone_rename_test.go` (or a new `internal/mobs` test)

- [ ] **Step 1: Write the failing test**

Create `internal/mobs/zone_sanitize_agreement_test.go`:

```go
package mobs

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// internal/mobs carries its own ZoneNameSanitize, used by Mob.Filepath to pick
// the folder a mob template is written to. rooms.ZoneNameSanitize picks the
// folder everything else uses. If they drift, a zone rename moves the rooms one
// way and the mobs another, and the mobs silently disappear from the zone.
func TestZoneNameSanitize_AgreesWithRoomsPackage(t *testing.T) {
	for _, in := range []string{
		"Stillwater", "Amber Valley", "amber valley", "Amber_Valley",
		"New Plymouth Common", "A", "", "Zone With  Double Space",
	} {
		if got, want := ZoneNameSanitize(in), rooms.ZoneNameSanitize(in); got != want {
			t.Errorf("ZoneNameSanitize(%q): mobs=%q rooms=%q — the two must agree", in, got, want)
		}
	}
}
```

- [ ] **Step 2: Run it**

Run: `go test ./internal/mobs/ -run TestZoneNameSanitize_AgreesWithRoomsPackage -v`

**If it PASSES:** good — the guard is now in place against future drift. Commit and move on.

**If it FAILS:** you have found a real pre-existing bug. Do NOT paper over it by changing the test. Report the exact divergence to the controller before proceeding — it means mob templates and room files already disagree about folder naming for some inputs.

**If it fails to compile** because `internal/mobs` importing `internal/rooms` is a cycle: drop the cross-package comparison and instead assert the mobs version against a hardcoded expectations table matching `rooms.ZoneNameSanitize`'s documented behaviour (lowercase; spaces → underscores), and note the cycle in the test comment.

- [ ] **Step 3: Commit**

```bash
git add internal/mobs/zone_sanitize_agreement_test.go
git commit -m "test(mobs): pin zone-folder sanitization agreement with rooms"
```

---

### Task 7: `Build.Zone.Rename` handler

**Files:** Modify `modules/gmcp/gmcp.Zone.go`, `modules/gmcp/gmcp.Zone_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestBuildZoneRename_RefusesWhenBlocked(t *testing.T) {
	w := newFakeZoneWorld()
	d := w.deps()
	d.renameBlockers = func(z string) []rooms.ZoneBlocker {
		return []rooms.ZoneBlocker{{Kind: "player", Id: "1 player(s) in room 100"}}
	}
	renameCalls := 0
	d.rename = func(oldName, newName string) error { renameCalls++; return nil }

	res := buildZoneRename(d, zoneRenameReq{Zone: "Testzone", NewName: "Quiet Water"})
	if res.Ok {
		t.Error("rename must be refused while a player is in the zone")
	}
	if len(res.ZoneRefs) != 1 {
		t.Errorf("expected the blocker surfaced, got %d", len(res.ZoneRefs))
	}
	if renameCalls != 0 {
		t.Error("d.rename must not be called when blocked")
	}
}

func TestBuildZoneRename_RenamesQuietZone(t *testing.T) {
	w := newFakeZoneWorld()
	d := w.deps()
	var gotOld, gotNew string
	d.rename = func(oldName, newName string) error { gotOld, gotNew = oldName, newName; return nil }

	res := buildZoneRename(d, zoneRenameReq{Zone: "Testzone", NewName: "Quiet Water"})
	if !res.Ok {
		t.Fatalf("quiet zone should rename, got %+v", res)
	}
	if gotOld != "Testzone" || gotNew != "Quiet Water" {
		t.Errorf("rename called with (%q,%q)", gotOld, gotNew)
	}
}

func TestBuildZoneRename_UnknownZone(t *testing.T) {
	w := newFakeZoneWorld()
	if res := buildZoneRename(w.deps(), zoneRenameReq{Zone: "Nowhere", NewName: "X Y"}); res.Ok {
		t.Error("unknown zone must not report success")
	}
}

func TestBuildZoneRename_SurfacesRenameError(t *testing.T) {
	w := newFakeZoneWorld()
	d := w.deps()
	d.rename = func(oldName, newName string) error { return fmt.Errorf("target path already exists") }
	res := buildZoneRename(d, zoneRenameReq{Zone: "Testzone", NewName: "Quiet Water"})
	if res.Ok {
		t.Error("a rename error must not report success")
	}
	if res.Error == "" {
		t.Error("expected the error surfaced to the author")
	}
}
```

Add `"fmt"` to the test file's imports if not already present.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./modules/gmcp/ -run TestBuildZoneRename -v`
Expected: FAIL — `undefined: buildZoneRename`

- [ ] **Step 3: Write minimal implementation**

Extend `zoneDeps` in `modules/gmcp/gmcp.Zone.go` with two fields:

```go
	rename         func(oldName, newName string) error
	renameBlockers func(zone string) []rooms.ZoneBlocker
```

Wire them in `realZoneDeps()`:

```go
		rename:         rooms.RenameZone,
		renameBlockers: rooms.ZoneRenameBlockers,
```

Wire them in the test fake's `deps()` too:

```go
		rename:         func(oldName, newName string) error { w.renamed = append(w.renamed, oldName+"->"+newName); return nil },
		renameBlockers: func(z string) []rooms.ZoneBlocker { return nil },
```

(add a `renamed []string` field to `fakeZoneWorld`).

Then the handler:

```go
type zoneRenameReq struct {
	Zone    string `json:"zone"`
	NewName string `json:"newName"`
}

func buildZoneRename(d zoneDeps, req zoneRenameReq) BuildResult {
	if d.load(req.Zone) == nil {
		return buildErr("zone %q not found", req.Zone)
	}
	if b := d.renameBlockers(req.Zone); len(b) > 0 {
		return BuildResult{
			Error:    "zone cannot be renamed right now — these must clear first",
			ZoneRefs: b,
		}
	}
	if err := d.rename(req.Zone, req.NewName); err != nil {
		return buildErr("could not rename zone %q: %s", req.Zone, err.Error())
	}
	return BuildResult{Ok: true, Message: "zone renamed to " + req.NewName}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./modules/gmcp/ -run TestBuildZone -v`
Expected: PASS (all zone tests, including the pre-existing ones)

- [ ] **Step 5: Commit**

```bash
git add modules/gmcp/gmcp.Zone.go modules/gmcp/gmcp.Zone_test.go
git commit -m "feat(build): Build.Zone.Rename handler"
```

---

### Task 8: Route, dispatch, and invalidate the mapper

**Files:** Modify `modules/gmcp/gmcp.go`, `modules/gmcp/gmcp.Build.go`

- [ ] **Step 1: Add the verb to the routed list**

In `modules/gmcp/gmcp.go` (~line 486), extend the Zone line:

```go
		`Build.Zone.Create`, `Build.Zone.List`, `Build.Zone.Get`, `Build.Zone.Update`, `Build.Zone.Delete`, `Build.Zone.Rename`,
```

- [ ] **Step 2: Add the dispatch case**

In `modules/gmcp/gmcp.Build.go`, after the `Build.Zone.Delete` case:

```go
	case `Build.Zone.Rename`:
		var req zoneRenameReq
		if json.Unmarshal(evt.Payload, &req) != nil {
			sendBuildResult(uid, buildErr("bad Build.Zone.Rename payload"))
			break
		}
		res := buildZoneRename(realZoneDeps(), req)
		if res.Ok {
			// Every cached map keyed by the old zone name is now wrong.
			mapper.ClearCache()
		}
		sendBuildResult(uid, res)
		sendZoneList(uid)
```

- [ ] **Step 3: Verify build and package tests**

Run: `go build ./... && go test ./modules/gmcp/`
Expected: build clean, PASS

- [ ] **Step 4: Commit**

```bash
git add modules/gmcp/gmcp.go modules/gmcp/gmcp.Build.go
git commit -m "feat(build): route Build.Zone.Rename and clear the mapper cache"
```

---

### Task 9: Rename control in the Zones panel

**Files:** Modify `_datafiles/html/public/static/js/zones.js`

Read `zones.js` fully first — you are modifying the existing detail form.

- [ ] **Step 1: Replace the read-only name treatment**

The name currently renders as static text with a note that renaming is not yet
supported. Replace that note with a **Rename** button beside the name.

- [ ] **Step 2: Rename flow**

Clicking Rename reveals a text input pre-filled with the current name, plus
Confirm and Cancel. Confirm sends:

```javascript
gmcp("Build.Zone.Rename", { zone: <currentName>, newName: <input value> });
```

Client-side pre-checks before sending (the server re-checks all of them):
- at least 2 characters
- different from the current name

- [ ] **Step 3: Result handling**

Mirror the create flow's `pendingCreateName` pattern: remember the submitted
new name, and on `ok` refresh the list and select the zone under its NEW name
(the result does not echo it). On failure show `obj.error` inline and render
`obj.zoneRefs` as `kind — id` lines — the realistic failure is a player
standing in the zone, and the author needs to know who.

- [ ] **Step 4: Warn about what rename does NOT do**

Put a short line in the rename UI: renaming does not carry over players'
explored-map history for the zone, so it will look unexplored to them again.
This is intended behaviour and authors should not report it as a bug.

- [ ] **Step 5: Syntax check**

Run: `node --check _datafiles/html/public/static/js/zones.js`
Expected: no output

- [ ] **Step 6: Commit**

```bash
git add _datafiles/html/public/static/js/zones.js
git commit -m "feat(build): zone rename control in the Zones panel"
```

---

### Task 10: Full verification gate

- [ ] **Step 1: Full suite**

Run: `go test ./...`
Expected: no failures.

- [ ] **Step 2: Format and vet**

Run: `gofmt -l internal modules && go vet ./...`
Expected: no output from either.

- [ ] **Step 3: Filesystem rename test**

Run: `DOGMUD_BOOT_SMOKE=1 go test ./internal/rooms/ -run TestRenameZone -v`
Expected: PASS, and `git status --short` clean afterwards (the test cleans up
its own directories).

- [ ] **Step 4: Boot under the prod setting**

Confirm `_datafiles/config.yaml` has `MapConsistencyEnforce: panic` (note: this
file carries git **skip-worktree**, so local edits never appear in
`git status`).

Run: `go run .` — expect
`mapper.ValidateZoneConsistency errors=0 warnings=0 mode="panic"` and no
`ERROR: PANIC` lines. Ctrl-C once the HTTP server starts.

- [ ] **Step 5: Unknown-key drift gate**

Run: `DOGMUD_BOOT_SMOKE=1 go test . -run TestSmoke_NoNewSilentlyIgnoredYAMLKeys -v`
Expected: PASS with `new: 0`.

- [ ] **Step 6: Confirm no prose churn**

Run: `git status --short` — clean. If any room YAML shows as modified after the
rename test, the content rewrite reflowed prose and Task 3's approach has
regressed. Investigate before proceeding.

- [ ] **Step 7: PATCH_NOTES.md**

Add a dated staff-facing entry in the house voice (prose, no identifiers, no
raw numbers) describing zone renaming and its one caveat (explored-map history
resets for that zone).

- [ ] **Step 8: Commit**

```bash
git add PATCH_NOTES.md
git commit -m "docs(patch-notes): zone rename"
```

- [ ] **Step 9: Browser gate — hand to the user**

Renaming is the most destructive operation in the editor. Ask the user to
drive `/build` → Zones → Rename on a throwaway zone before this is called done.
