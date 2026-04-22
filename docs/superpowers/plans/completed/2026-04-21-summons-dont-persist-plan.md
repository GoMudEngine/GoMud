# Summons Don't Persist — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the redundant `mobs.instances/summons/` persistence layer that causes companion-type mobs (summoned / raised / conjured / charmed) to inherit progression across dismiss + re-summon cycles. CompanionInfo on the owner's user YAML is already the correct persistence boundary.

**Architecture:**
1. Guard `SaveMobInstance` so charmed mobs never write instance files (defense in depth).
2. Split `NewMobById` into two constructors: the existing one keeps the organic-spawn behavior (reads `LoadMobInstance`), and a new `NewMobByIdFresh` skips the read path for companion-spawn call sites.
3. Add `NukeSummonsInstances` boot cleanup that wipes all files under `mobs.instances/summons/` on startup.
4. Migrate all 6 companion-creating call sites to `NewMobByIdFresh`.

**Tech Stack:** Go 1.21+, testify for assertions, YAML file-based persistence, existing `mudlog` package.

**Related spec:** `docs/superpowers/specs/2026-04-21-summons-dont-persist-design.md`

**Branch:** `feature/fix-summons-persistence` (already created; spec committed as `aa1366e6`).

---

## File Structure

**Modified files:**
- `internal/mobs/instance_save.go` — add `IsCharmed()` guard to `SaveMobInstance`; add `NukeSummonsInstances` function.
- `internal/mobs/mobs.go` — split `NewMobById` constructor to add `NewMobByIdFresh`.
- `internal/mobs/instance_save_test.go` *(new file)* — tests for guard + nuke.
- `internal/mobs/mobs_test.go` — tests for `NewMobByIdFresh`.
- `main.go` — wire `NukeSummonsInstances` call at line ~201.
- `internal/hooks/companion_summon.go` — migrate caller (line 127).
- `internal/hooks/PlayerSpawn_HandleJoin.go` — migrate caller (line 62).
- `internal/behaviortree/actions_mob.go` — migrate caller (line 74).
- `internal/usercommands/buy.go` — migrate caller (line ~544 — the `newMob := mobs.NewMobById(...)` call just above line 564).
- `internal/usercommands/character.go` — migrate caller (line 328).

**Note:** `internal/usercommands/character.go:421` is a charm-existing-mob path, not a mob construction — already excluded per spec Section 2.

---

## Task 1: Guard SaveMobInstance for charmed mobs

**Files:**
- Modify: `internal/mobs/instance_save.go:46-57`
- Create: `internal/mobs/instance_save_test.go`

The guard is the belt-and-suspenders layer. Even if a caller forgets to switch to `NewMobByIdFresh`, no file is written for a charmed mob.

- [ ] **Step 1: Write the failing test**

Create `internal/mobs/instance_save_test.go` with:

```go
package mobs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/stretchr/testify/assert"
)

// TestSaveMobInstance_CharmedMobSkipsWrite verifies the guard added to
// SaveMobInstance: any mob charmed to a user must not write to
// mobs.instances/ because its progression lives on CompanionInfo on the
// owner's user YAML.
func TestSaveMobInstance_CharmedMobSkipsWrite(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	mob := NewMobById(1, 100)
	if mob == nil {
		t.Fatal("NewMobById returned nil")
	}
	// Give the mob some progression so it WOULD normally persist.
	mob.Character.Stats.Strength.Training = 10
	// Charm it to a user — this is the signal that it's a companion.
	mob.Character.Charm(42, 99999, "")

	err := SaveMobInstance(mob)
	assert.NoError(t, err)

	// Assert no file was written.
	path := instancePath(mob.MobId, mob.Zone, mob.Character.Name, mob.HomeRoomId)
	_, statErr := os.Stat(path)
	assert.True(t, os.IsNotExist(statErr),
		"expected no file at %s for charmed mob, got stat err %v",
		path, statErr)

	// Cleanup in case the test fails and a file was written.
	_ = os.Remove(path)
	_ = os.Remove(filepath.Dir(path))
}

// TestSaveMobInstance_UncharmedMobWritesFile verifies the inverse: an
// uncharmed mob with progression still gets its file written, so genuine
// world-mob persistence is unaffected by the guard.
func TestSaveMobInstance_UncharmedMobWritesFile(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	mob := NewMobById(1, 100)
	if mob == nil {
		t.Fatal("NewMobById returned nil")
	}
	mob.Character.Stats.Strength.Training = 10
	// NOT charming — this is an organic world mob.

	err := SaveMobInstance(mob)
	assert.NoError(t, err)

	path := instancePath(mob.MobId, mob.Zone, mob.Character.Name, mob.HomeRoomId)
	_, statErr := os.Stat(path)
	assert.NoError(t, statErr, "expected file at %s for uncharmed mob", path)

	// Cleanup.
	_ = os.Remove(path)
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test -run 'TestSaveMobInstance_' ./internal/mobs/`
Expected: `TestSaveMobInstance_CharmedMobSkipsWrite` **FAILS** (file gets written because no guard yet). `TestSaveMobInstance_UncharmedMobWritesFile` passes.

- [ ] **Step 3: Add the guard**

In `internal/mobs/instance_save.go`, modify the start of `SaveMobInstance` (currently at line 48):

```go
// SaveMobInstance writes a mob's progression data to disk so it survives
// server restarts. Only called for mobs with progression enabled.
//
// Charmed mobs are skipped — their progression is the owner's
// responsibility (CompanionInfo on the owner's user YAML). The file
// layer would otherwise be a redundant, room-keyed second persistence
// that leaks across player-summon cycles. See
// docs/superpowers/specs/2026-04-21-summons-dont-persist-design.md.
func SaveMobInstance(mob *Mob) error {
	// Companions live on CompanionInfo, not in mobs.instances/.
	if mob.Character.IsCharmed() {
		return nil
	}

	b := configs.GetBalanceConfig()
	if !bool(b.MobProgressionEnabled) {
		return nil
	}
	// ... rest unchanged
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test -run 'TestSaveMobInstance_' ./internal/mobs/`
Expected: both tests PASS.

- [ ] **Step 5: Run the full mobs package test suite to verify nothing else broke**

Run: `go test ./internal/mobs/`
Expected: all tests PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/mobs/instance_save.go internal/mobs/instance_save_test.go
git commit -m "$(cat <<'EOF'
fix(mobs): skip SaveMobInstance for charmed companions

A charmed mob's progression lives on CompanionInfo on the owner's
user YAML. Writing a parallel file under mobs.instances/summons/
was a redundant persistence layer that leaked across summon-
dismiss-resummon cycles because the file was keyed by room, not
by owner, and was not deleted in every teardown path.

Belt-and-suspenders guard at the persistence boundary: if a mob
is charmed to any user, SaveMobInstance returns nil without
writing. Genuine world-mob persistence is unchanged.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Add NewMobByIdFresh constructor

**Files:**
- Modify: `internal/mobs/mobs.go:304-411` (split `NewMobById` via internal helper)
- Modify: `internal/mobs/mobs_test.go` (append new test)

Currently `NewMobById` always calls `LoadMobInstance` at line 345. We want two behaviors: the existing one for organic world spawns, and a fresh variant for companion spawns that skips the load.

**Implementation strategy:** extract the body of `NewMobById` into an unexported helper that takes a `skipInstanceLoad bool`, and have both public functions call the helper. Prevents code duplication.

- [ ] **Step 1: Write the failing test**

Append to `internal/mobs/mobs_test.go` (bottom of file, after the `PathQueue` tests):

```go
// ─── NewMobByIdFresh ──────────────────────────────────────────────────────

// TestNewMobByIdFresh_IgnoresInstanceFile verifies that NewMobByIdFresh
// does not load any existing mobs.instances/ file for the mob — companion
// spawns must always start from the template.
func TestNewMobByIdFresh_IgnoresInstanceFile(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	// First: create a mob the ORGANIC way and stuff it with progression,
	// then persist. This seeds an instance file that would normally be
	// loaded by the next NewMobById call.
	organic := NewMobById(1, 100)
	if organic == nil {
		t.Fatal("NewMobById returned nil")
	}
	organic.Character.Stats.Strength.Training = 999
	// Bypass the guard by using SaveMobInstance directly — at this point
	// the mob is uncharmed, so the guard doesn't fire. (If the guard fires
	// here, the test premise is broken; the assertion below will catch it.)
	if err := SaveMobInstance(organic); err != nil {
		t.Fatalf("seed SaveMobInstance: %v", err)
	}
	defer func() {
		_ = os.Remove(instancePath(organic.MobId, organic.Zone,
			organic.Character.Name, organic.HomeRoomId))
	}()

	// Now call NewMobByIdFresh with the SAME mobId and homeRoomId.
	fresh := NewMobByIdFresh(1, 100)
	if fresh == nil {
		t.Fatal("NewMobByIdFresh returned nil")
	}

	// Fresh mob must NOT inherit the 999 training value.
	assert.NotEqual(t, 999, fresh.Character.Stats.Strength.Training,
		"NewMobByIdFresh must not load from mobs.instances/")
}

// TestNewMobById_StillLoadsInstanceFile is the control case — existing
// organic spawn behavior is preserved.
func TestNewMobById_StillLoadsInstanceFile(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	seed := NewMobById(1, 100)
	if seed == nil {
		t.Fatal("NewMobById returned nil")
	}
	seed.Character.Stats.Strength.Training = 777
	if err := SaveMobInstance(seed); err != nil {
		t.Fatalf("seed SaveMobInstance: %v", err)
	}
	defer func() {
		_ = os.Remove(instancePath(seed.MobId, seed.Zone,
			seed.Character.Name, seed.HomeRoomId))
	}()

	organic := NewMobById(1, 100)
	if organic == nil {
		t.Fatal("NewMobById returned nil")
	}
	assert.Equal(t, 777, organic.Character.Stats.Strength.Training,
		"NewMobById must still load from mobs.instances/")
}
```

Ensure the `os` import exists at the top of `mobs_test.go`; add it if needed.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test -run 'TestNewMobByIdFresh|TestNewMobById_StillLoadsInstanceFile' ./internal/mobs/`
Expected: `TestNewMobByIdFresh_IgnoresInstanceFile` **FAILS** (undefined: `NewMobByIdFresh`). `TestNewMobById_StillLoadsInstanceFile` passes (or also fails to compile — same reason).

- [ ] **Step 3: Split the constructor**

In `internal/mobs/mobs.go`, replace the current `NewMobById` (at line 304) with:

```go
// NewMobById creates a new mob instance from the template `mobId`,
// placed at `homeRoomId`. If a saved instance file exists for this
// (mobId, zone, mobName, homeRoomId) tuple, its progression is loaded
// onto the new mob. This is the constructor for organic world spawns.
//
// Companion-spawning callers (summon / raise / conjure / charm-respawn)
// must use NewMobByIdFresh instead — their progression lives on
// CompanionInfo, not on the file system.
func NewMobById(mobId MobId, homeRoomId int, forceStatPool ...int) *Mob {
	return newMobByIdInternal(mobId, homeRoomId, false, forceStatPool...)
}

// NewMobByIdFresh creates a mob instance from the template without
// reading any saved progression file. Used by companion-spawning code
// paths (summon / raise / conjure / login-respawn of companions /
// companion-vending NPCs / admin suicide-vanish). Template defaults
// (including random stat pool distribution) apply as if no instance
// file existed.
func NewMobByIdFresh(mobId MobId, homeRoomId int, forceStatPool ...int) *Mob {
	return newMobByIdInternal(mobId, homeRoomId, true, forceStatPool...)
}

func newMobByIdInternal(mobId MobId, homeRoomId int, skipInstanceLoad bool, forceStatPool ...int) *Mob {

	mobsMu.RLock()
	m, ok := mobs[int(mobId)]
	mobsMu.RUnlock()

	if ok {

		mobInstancesMu.Lock()
		instanceCounter++
		newInstanceId := instanceCounter
		mobInstancesMu.Unlock()

		mob := *m // Make a copy of the mob
		mob.InstanceId = newInstanceId

		mob.HomeRoomId = homeRoomId
		mob.Character.RoomId = homeRoomId
		mob.Character.IsMob = true
		mob.Character.PlayerDamage = make(map[int]int)

		// Deep copy maps to prevent shared state with template.
		// Go shallow copy shares map backing data — mutations to an
		// instance's skills/spellbook would contaminate the template.
		if mob.Character.Skills != nil {
			skillsCopy := make(map[string]int, len(mob.Character.Skills))
			for k, v := range mob.Character.Skills {
				skillsCopy[k] = v
			}
			mob.Character.Skills = skillsCopy
		}
		if mob.Character.SpellBook != nil {
			spellCopy := make(map[string]int, len(mob.Character.SpellBook))
			for k, v := range mob.Character.SpellBook {
				spellCopy[k] = v
			}
			mob.Character.SpellBook = spellCopy
		}

		// Try to load a saved instance unless the caller requested a
		// fresh spawn (companion-spawn paths skip this).
		var savedInstance *MobInstanceData
		if !skipInstanceLoad {
			savedInstance = LoadMobInstance(mob.MobId, mob.Zone, mob.Character.Name, homeRoomId)
		}
		if savedInstance != nil {
			// Restore saved progression
			mob.Character.Stats.Strength.Training = savedInstance.StrengthTraining
			mob.Character.Stats.Dexterity.Training = savedInstance.DexterityTraining
			mob.Character.Stats.Perception.Training = savedInstance.PerceptionTraining
			mob.Character.Stats.Vitality.Training = savedInstance.VitalityTraining
			mob.Character.Stats.Willpower.Training = savedInstance.WillpowerTraining
			mob.Character.Stats.Charisma.Training = savedInstance.CharismaTraining
			if savedInstance.Skills != nil {
				mob.Character.Skills = savedInstance.Skills
			}
			if savedInstance.SkillUseCount != nil {
				mob.Character.SkillUseCount = savedInstance.SkillUseCount
			}
			if savedInstance.StatUseCount != nil {
				mob.Character.StatUseCount = savedInstance.StatUseCount
			}
			if savedInstance.Mutations != nil {
				mob.Character.Mutations = savedInstance.Mutations
			}
			mob.Character.MutationProgress = savedInstance.MutationProgress
		} else {
			// No saved instance (or skipInstanceLoad set) — randomize stat pool as usual
			statPool := mob.StatPool
			if len(forceStatPool) > 0 && forceStatPool[0] > 0 {
				statPool = forceStatPool[0]
			}
			// [rest of stat-pool distribution + trailing logic unchanged —
			//  copy verbatim from the current NewMobById body]
```

**Important:** only the top of the function changes. The rest of the body (from `for i := 0; i < statPool; i++` through the final `return &mob`) must be copied **verbatim** into `newMobByIdInternal`. The public `NewMobById` now delegates; `NewMobByIdFresh` is new.

**Refactor notes:**
- Keep all existing behavior identical when `skipInstanceLoad == false`.
- The `forceStatPool` variadic is preserved on both public constructors.
- No change to return type or signature semantics beyond adding the new function.

- [ ] **Step 4: Run the new tests**

Run: `go test -run 'TestNewMobByIdFresh|TestNewMobById_StillLoadsInstanceFile' ./internal/mobs/`
Expected: both PASS.

- [ ] **Step 5: Run the full mobs package test suite**

Run: `go test ./internal/mobs/`
Expected: all tests PASS. (If any test fails, the verbatim-copy of the function body was incomplete — diff against the pre-refactor version.)

- [ ] **Step 6: Run the full project test suite**

Run: `go build ./... && go test ./...`
Expected: build succeeds; all tests pass.

- [ ] **Step 7: Commit**

```bash
git add internal/mobs/mobs.go internal/mobs/mobs_test.go
git commit -m "$(cat <<'EOF'
refactor(mobs): split NewMobById into organic + Fresh constructors

Introduces NewMobByIdFresh for callers that should not load any
saved instance file. Implemented via a shared unexported helper
so the existing NewMobById behavior is identical (template load
when a file exists, random stat-pool distribution otherwise).

Caller migrations land in the next commit.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Add NukeSummonsInstances boot cleanup

**Files:**
- Modify: `internal/mobs/instance_save.go` (add function)
- Modify: `internal/mobs/instance_save_test.go` (append tests)

One-time purge at boot of the `summons/` zone under `mobs.instances/`. Any file there is stale by definition after this change — companions persist via CompanionInfo, not files.

- [ ] **Step 1: Write the failing tests**

Append to `internal/mobs/instance_save_test.go`:

```go
// TestNukeSummonsInstances_RemovesAllFiles verifies the boot-cleanup
// nuke — every file under mobs.instances/summons/ is removed, and the
// count is returned for logging.
func TestNukeSummonsInstances_RemovesAllFiles(t *testing.T) {
	baseDir := filepath.Join(
		configs.GetFilePathsConfig().DataFiles.String(),
		"mobs.instances", "summons")

	// Seed three fake files under summons/.
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	defer os.RemoveAll(baseDir)

	for _, name := range []string{"1-foo-room1.yaml", "2-bar-room2.yaml", "3-baz-room3.yaml"} {
		if err := os.WriteFile(filepath.Join(baseDir, name), []byte("x"), 0644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	pruned := NukeSummonsInstances()
	assert.Equal(t, 3, pruned, "expected 3 files nuked")

	// Directory should be empty (may still exist).
	entries, err := os.ReadDir(baseDir)
	if err == nil {
		assert.Empty(t, entries, "summons/ should have no files remaining")
	}
}

// TestNukeSummonsInstances_IgnoresOtherZones verifies the nuke only
// targets summons/ — legitimate world-mob instance files under other
// zones are untouched.
func TestNukeSummonsInstances_IgnoresOtherZones(t *testing.T) {
	base := filepath.Join(
		configs.GetFilePathsConfig().DataFiles.String(),
		"mobs.instances")

	summonsDir := filepath.Join(base, "summons")
	worldDir := filepath.Join(base, "thornwall_city")

	if err := os.MkdirAll(summonsDir, 0755); err != nil {
		t.Fatalf("mkdir summons: %v", err)
	}
	if err := os.MkdirAll(worldDir, 0755); err != nil {
		t.Fatalf("mkdir world: %v", err)
	}
	defer os.RemoveAll(summonsDir)
	defer os.RemoveAll(worldDir)

	_ = os.WriteFile(filepath.Join(summonsDir, "1-foo-room1.yaml"), []byte("x"), 0644)
	worldFile := filepath.Join(worldDir, "2-wolf-room200.yaml")
	_ = os.WriteFile(worldFile, []byte("x"), 0644)

	pruned := NukeSummonsInstances()
	assert.Equal(t, 1, pruned)

	// World-zone file must still exist.
	_, err := os.Stat(worldFile)
	assert.NoError(t, err, "world-mob instance file must not be touched")
}

// TestNukeSummonsInstances_NoDirectory verifies the nuke is a no-op
// (no panic, returns 0) when the summons/ directory doesn't exist.
func TestNukeSummonsInstances_NoDirectory(t *testing.T) {
	// Ensure directory does not exist.
	base := filepath.Join(
		configs.GetFilePathsConfig().DataFiles.String(),
		"mobs.instances", "summons")
	_ = os.RemoveAll(base)

	pruned := NukeSummonsInstances()
	assert.Equal(t, 0, pruned)
}
```

Ensure `"github.com/GoMudEngine/GoMud/internal/configs"` is imported in the test file.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test -run 'TestNukeSummonsInstances_' ./internal/mobs/`
Expected: all three **FAIL** with `undefined: NukeSummonsInstances`.

- [ ] **Step 3: Add the function**

Append to `internal/mobs/instance_save.go` (after `PruneStaleInstances`):

```go
// NukeSummonsInstances removes every file under
// _datafiles/.../mobs.instances/summons/ at boot. Companion persistence
// lives on CompanionInfo on the owner's user YAML; any file in this
// directory is stale and would poison the next summon of the same
// template in the same room. No-op if the directory doesn't exist.
// Returns the count of files removed (for logging).
func NukeSummonsInstances() int {
	baseDir := util.FilePath(
		configs.GetFilePathsConfig().DataFiles.String(), `/`, `mobs.instances`, `/`, `summons`,
	)

	pruned := 0
	filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors (e.g., directory doesn't exist)
		}
		if info.IsDir() {
			return nil
		}
		if removeErr := os.Remove(path); removeErr == nil {
			pruned++
		}
		return nil
	})

	if pruned > 0 {
		mudlog.Info("mobs.NukeSummonsInstances", "pruned", pruned)
	}

	return pruned
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test -run 'TestNukeSummonsInstances_' ./internal/mobs/`
Expected: all three PASS.

- [ ] **Step 5: Run the full mobs package test suite**

Run: `go test ./internal/mobs/`
Expected: all tests PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/mobs/instance_save.go internal/mobs/instance_save_test.go
git commit -m "$(cat <<'EOF'
feat(mobs): NukeSummonsInstances boot cleanup

Every file under mobs.instances/summons/ is stale after the
persistence boundary change — companions live on CompanionInfo
on the owner's user YAML, not in the file system. NukeSummonsInstances
wipes the directory at boot to purge existing corruption. No-op
if the directory doesn't exist.

Wired into main.go in the next commit.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Wire NukeSummonsInstances into boot

**Files:**
- Modify: `main.go:201`

- [ ] **Step 1: Confirm the current boot call**

Run: `grep -n "PruneStaleInstances\|NukeSummonsInstances" main.go`
Expected output: `main.go:201: mobs.PruneStaleInstances(int(configs.GetBalanceConfig().MobInstanceMaxAgeDays))`
(only one line — Nuke not yet wired.)

- [ ] **Step 2: Add the nuke call**

In `main.go`, find the line:

```go
mobs.PruneStaleInstances(int(configs.GetBalanceConfig().MobInstanceMaxAgeDays))
```

Insert a new call **before** it:

```go
mobs.NukeSummonsInstances()
mobs.PruneStaleInstances(int(configs.GetBalanceConfig().MobInstanceMaxAgeDays))
```

- [ ] **Step 3: Build the project to verify**

Run: `go build ./...`
Expected: build succeeds.

- [ ] **Step 4: Commit**

```bash
git add main.go
git commit -m "$(cat <<'EOF'
feat(boot): wire NukeSummonsInstances at startup

Runs before PruneStaleInstances so the summons/ zone is always
clean on boot. Logs pruned count.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Migrate companion-spawn callers to NewMobByIdFresh

**Files:**
- Modify: `internal/hooks/companion_summon.go:127`
- Modify: `internal/hooks/PlayerSpawn_HandleJoin.go:62`
- Modify: `internal/behaviortree/actions_mob.go:74`
- Modify: `internal/usercommands/buy.go` (locate the `newMob := mobs.NewMobById(...)` call; around line 544)
- Modify: `internal/usercommands/character.go:328`

One-line replacement at each site. No behavior change for organic world spawns — only companion-spawn paths switch.

- [ ] **Step 1: Migrate companion_summon.go**

Find the line:

```go
	mob := mobs.NewMobById(mobs.MobId(spellData.SummonMobId), room.RoomId, pool)
```

Replace with:

```go
	mob := mobs.NewMobByIdFresh(mobs.MobId(spellData.SummonMobId), room.RoomId, pool)
```

- [ ] **Step 2: Migrate PlayerSpawn_HandleJoin.go**

Find the line:

```go
		mob := mobs.NewMobById(mobs.MobId(comp.MobId), user.Character.RoomId)
```

Replace with:

```go
		mob := mobs.NewMobByIdFresh(mobs.MobId(comp.MobId), user.Character.RoomId)
```

- [ ] **Step 3: Migrate behaviortree/actions_mob.go**

Find the line:

```go
		companion := mobs.NewMobById(mobs.MobId(mobId), room.RoomId, pool)
```

Replace with:

```go
		companion := mobs.NewMobByIdFresh(mobs.MobId(mobId), room.RoomId, pool)
```

Both the hostile and charmed branches go through this one call — one edit covers both.

- [ ] **Step 4: Migrate buy.go**

Locate the call (around line 544, just above `newMob.Character.Charm(...)` at line 564). The exact current call is a `mobs.NewMobById(...)` that produces `newMob`. Change to `mobs.NewMobByIdFresh`.

Run: `grep -n "NewMobById" internal/usercommands/buy.go` to find the exact line.
Replace `mobs.NewMobById(` with `mobs.NewMobByIdFresh(` at that line only (if there's more than one match, pick the one that feeds into `newMob.Character.Charm` at line ~564).

- [ ] **Step 5: Migrate character.go**

Find the line (at line 328):

```go
		m := mobs.NewMobById(59, user.Character.RoomId)
```

Replace with:

```go
		m := mobs.NewMobByIdFresh(59, user.Character.RoomId)
```

The `m := ...` call at line ~421 is a **different** control flow — it's charming an already-constructed mob, not constructing a new one. Do NOT change that one.

Run: `grep -n "NewMobById" internal/usercommands/character.go` to confirm there's only one construction site to change.

- [ ] **Step 6: Verify no other companion-spawn sites were missed**

Run: `grep -rn "mobs.NewMobById(" --include='*.go' internal/`

Scan the output. Any line that constructs a mob and then calls `Character.Charm(user.UserId, ...)` (or is inside a summon/raise/conjure/companion-respawn flow) should be using `NewMobByIdFresh`. Organic spawns (zone wanders, pack spawns, room mob loading) should stay on `NewMobById`.

Expected: after this task, only organic-spawn call sites use `NewMobById`. Document in the commit message if any new site was found beyond the 5 listed above.

- [ ] **Step 7: Build + test**

Run: `go build ./... && go test ./...`
Expected: build succeeds; all tests pass.

- [ ] **Step 8: Commit**

```bash
git add internal/hooks/companion_summon.go internal/hooks/PlayerSpawn_HandleJoin.go internal/behaviortree/actions_mob.go internal/usercommands/buy.go internal/usercommands/character.go
git commit -m "$(cat <<'EOF'
fix(mobs): companion spawns use NewMobByIdFresh

Five call sites migrated: summon/raise/conjure spell resolution,
login respawn of saved companions, mob-summoning-mob btree action,
companion-vending NPC (buy.go), and the admin suicide-vanish path
in character.go.

These mobs' progression lives on CompanionInfo on the owner's
user YAML, so reading an existing mobs.instances/ file at spawn
time would poison the fresh summon with a prior mob's stats. The
write side is already guarded; this closes the read side.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Manual smoke test checklist

Tests above cover the persistence boundary in isolation. These smoke tests verify the end-to-end behavior per the spec.

**Pre-test setup:**
- Build and run the server locally: `go run . -c _datafiles/config.yaml`
- Use a test character with access to a summon spell (e.g., `summon-air-elemental` if the character can cast it, or use admin tooling to grant the spell).
- Open a terminal tail on the server log to watch for `mobs.NukeSummonsInstances` at boot.

Each checkbox is a manual scenario; record pass/fail in the commit message of Task 7 or in the PR body.

- [ ] **Smoke 1: Boot-time nuke log appears when summons/ has files**

Steps:
1. Stop server.
2. `touch _datafiles/world/dogmud/mobs.instances/summons/99-test-room99.yaml`
3. Start server.
4. Look for log line: `mobs.NukeSummonsInstances pruned 1` (or similar).
5. Verify file is gone: `ls _datafiles/world/dogmud/mobs.instances/summons/`

Expected: log line present, directory is empty.

- [ ] **Smoke 2: Summon + dismiss + re-summon shows clean stats**

Steps:
1. Log in test character.
2. Summon air elemental. Record its stat profile (`consider <name>`).
3. Spar with it or play long enough for a mutation/skill to develop.
4. Dismiss it.
5. Immediately summon another air elemental.
6. Check: `ls _datafiles/world/dogmud/mobs.instances/summons/` — should have zero files while the summon is alive (because `SaveMobInstance` is guarded) AND after dismiss.
7. The new elemental should have a fresh stat profile — no inherited mutations / skills from the previous one.

Expected: no file written during the companion's lifetime; re-summon rolls fresh.

- [ ] **Smoke 3: Companion progression persists across clean logout**

Steps:
1. Summon air elemental, train it (combat to pump weapon/spell/spell skill).
2. Record skill levels via `consider` or admin command.
3. Cleanly log out (`quit`).
4. Log back in.
5. Companion should respawn at owner's room with the same skill levels — CompanionInfo layer still works.

Expected: progression preserved across logout.

- [ ] **Smoke 4: Cross-player room collision (the original bug repro)**

Steps:
1. Character A summons air elemental in room X, trains it, then dismisses.
2. Character B summons air elemental in room X.
3. Character B's elemental should have fresh stats — NOT inherit from A's.

Expected: no cross-player leakage.

- [ ] **Smoke 5: Genuine world-mob progression still works**

Steps:
1. Identify a world mob zone that has `MobProgressionEnabled` set (e.g., thornwall wolves).
2. Allow a zone mob to gain progression.
3. Shut down server cleanly.
4. Inspect: `ls _datafiles/world/dogmud/mobs.instances/<zone>/` — should contain files.
5. Restart. Verify progression persists on that zone mob (e.g., via admin inspect).

Expected: world mobs still write + read instance files. Only `summons/` is nuked.

- [ ] **Smoke 6: Charmed wild creature scenario**

Steps:
1. Character charms a wild animal (e.g., wolf via charm spell).
2. While charmed, verify no instance file is written for that mob
   (`ls` the wolf's zone, confirm the specific mob is absent).
3. Dismiss (turns hostile).
4. The wolf may still have an old-session instance file in its zone if one existed before the charm — this is the documented accepted edge case.

Expected: no new file written during the charm; old file (if any) is handled by `PruneStaleInstances` age sweep.

---

## Task 7: Finalize + update memory

- [ ] **Step 1: Confirm all tests pass on the branch**

Run: `go build ./... && go test ./...`
Expected: all green.

- [ ] **Step 2: Update MEMORY.md**

In `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\MEMORY.md`:

1. Move the "Instance-save corruption — rooms AND mobs" entry out of "Bugs to Fix".
2. Add a new "Completed (2026-04-21)" section with a summary of the fix.
3. Leave Fix A (rooms exits) in "Bugs to Fix" if not yet addressed this session.

Also update `project_instance_save_exits_corruption.md` — mark Fix D as landed, leave Fix A status as-is.

- [ ] **Step 3: Ask the user whether to merge to development now or open a PR first**

Per `github_guide.md` and CLAUDE.md, feature branches merge into `development` with `--no-ff`. Ask the user before doing this — do NOT merge autonomously.

---

## Self-Review Checklist

Run through after the plan is complete, before handing off for execution:

**Spec coverage:**
- Section 1 (scope) — Task 1 covers write-side guard, Tasks 2 + 5 cover read-side, Task 3 + 4 cover boot nuke. Full coverage. ✓
- Section 2 (architecture rule + code changes) — all four code-change bullets in the spec are covered by Tasks 1-5. ✓
- Section 3 (boot cleanup + edge cases) — Task 3 implements the nuke, Tasks 1+2 handle the in-process prevention. Edge cases are documented, not implemented (accepted as-is per spec). ✓
- Testing section — unit tests 1-5 all appear in Tasks 1-3; smoke tests 1-4 appear in Task 6 (plus two extra scenarios for belt-and-suspenders).

**Placeholder scan:**
- No "TBD", "TODO", "etc." in any task.
- Every code block is complete and runnable.
- File paths are exact; line numbers are exact where known, `~` marked where approximate.

**Type consistency:**
- `NewMobByIdFresh(mobId MobId, homeRoomId int, forceStatPool ...int)` — used consistently across Tasks 2, 5.
- `NukeSummonsInstances() int` — signature consistent across Tasks 3, 4.
- Guard clause `if mob.Character.IsCharmed()` — used in Task 1, referenced in spec. Consistent.

No issues found. Plan is ready for execution.
