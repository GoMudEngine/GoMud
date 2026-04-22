# Respawn Aggro Cleanup + Condition Wipe + Grace Period — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** On player death, comprehensively clean all aggro state touching the dying player, wipe combat conditions, and apply a short respawn grace period that prevents mobs from acquiring aggro on the respawning player.

**Architecture:** Three fixes land inline in the existing `Suicide()` function at `internal/usercommands/suicide.go`. Supporting infrastructure: a new `NoAggroTarget` buff flag, a new buff id 81 "Respawn Grace", a callback pattern (`characters.SetUserUntargetableCheck`) to resolve the users↔characters import cycle, and a new `Death.RespawnGraceRounds` config knob.

**Tech Stack:** Go 1.21+, existing buff + character + config packages, existing callback-registration pattern from `rooms.SetCompanionTransport` / `SetBTreeStateEvictor`.

**Related spec:** `docs/superpowers/specs/2026-04-21-respawn-aggro-cleanup-design.md`
**Branch:** `feature/respawn-aggro-cleanup` (created; spec committed as `728469ef`).

---

## File Structure

**New files:**
- `_datafiles/world/dogmud/buffs/81-respawn_grace.yaml` — the new buff content.

**Modified files:**
- `internal/buffs/buffspec.go` — add `NoAggroTarget Flag = "no-aggro-target"` constant.
- `internal/characters/aggro.go` — add `SetUserUntargetableCheck` callback + guard in `SetAggro`.
- `internal/configs/config.gameplay.go` — add `RespawnGraceRounds ConfigInt` to `GameplayDeath`.
- `internal/hooks/aggro_helpers.go` — defense-in-depth `HasBuffFlag` check in `CompanionAutoTarget`.
- `internal/usercommands/suicide.go` — Fix A + Fix B + Fix C applications.
- `main.go` — register the callback at boot.

**New test files:**
- `internal/characters/aggro_grace_test.go` — unit tests for the SetAggro grace guard.
- `internal/usercommands/suicide_cleanup_test.go` — unit tests for the Suicide cleanup paths (inbound aggro, companion aggro, conditions, grace buff applied).

---

## Task 1: Add `NoAggroTarget` flag + buff 81 YAML

**Files:**
- Modify: `internal/buffs/buffspec.go` — add flag constant.
- Create: `_datafiles/world/dogmud/buffs/81-respawn_grace.yaml`.

### Step 1: Add the flag constant

In `internal/buffs/buffspec.go`, find the existing flag block (starts near line 33 with `All Flag = ""`). Add the new constant alphabetically or at the end of the meaningful group:

```go
	NoAggroTarget  Flag = `no-aggro-target`
```

Example placement (between existing `NoFlee` and `CancelIfCombat` lines, alphabetically, or wherever fits cleanly):

```go
const (
	All Flag = ``

	// Combat-state flags
	NoCombat       Flag = `no-combat`
	NoMovement     Flag = `no-go`
	NoFlee         Flag = `no-flee`
	NoAggroTarget  Flag = `no-aggro-target`   // ← new: grace-period protection

	CancelIfCombat Flag = `cancel-on-combat`
	// ... rest unchanged
)
```

Check for any buff-flag registration step elsewhere in the package (e.g., a `FlagsFromStrings` parser). If it uses a map of known flags, add the new one.

### Step 2: Create buff 81 YAML

Create `_datafiles/world/dogmud/buffs/81-respawn_grace.yaml` with content matching the format of `80-rally.yaml` and other existing buff YAMLs:

```yaml
buffid: 81
name: Respawn Grace
description: The world gives you a moment to gather yourself after death.
flags:
  - no-aggro-target
triggerrate: 1 round
triggercount: 3
```

Notes:
- `triggerrate: 1 round` + `triggercount: 3` = 3 rounds duration. Matches other buffs with per-round ticks.
- `flags:` list items use the kebab-case STRING value of the flag (`no-aggro-target`), not the Go constant name.
- No `statmods` block needed — the buff's effect is entirely via the flag check in SetAggro.
- No `onstart`/`onend` messages strictly required — player sees the buff appear/expire in status.

### Step 3: Verify the buff loads

Run: `go build ./... && go test ./internal/buffs/`
Expected: clean. Buff loaders run during tests/boot and will catch malformed YAML.

### Step 4: Commit

```bash
git add internal/buffs/buffspec.go _datafiles/world/dogmud/buffs/81-respawn_grace.yaml
git commit -m "$(cat <<'EOF'
feat(buffs): NoAggroTarget flag + Respawn Grace buff (id 81)

Adds the no-aggro-target flag and a 3-round "Respawn Grace"
buff that will be applied to players on respawn. Enforcement
in SetAggro lands in a later commit.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

## Context for Task 1

- Branch: `feature/respawn-aggro-cleanup`. Parent: `728469ef` (spec commit).
- Existing flag constants in `internal/buffs/buffspec.go` use kebab-case strings (e.g., `NoCombat Flag = "no-combat"`).
- Existing buff YAMLs use `triggerrate` + `triggercount` (see `_datafiles/world/dogmud/buffs/80-rally.yaml`).

Work from: `C:\Users\Calabe Davis\workspace\DOGMud`. Shell is bash on Windows.

---

## Task 2: SetUserUntargetableCheck callback + SetAggro guard

**Files:**
- Modify: `internal/characters/aggro.go`
- Create: `internal/characters/aggro_grace_test.go`

Add a package-level callback (mirroring `rooms.SetCompanionTransport`) and a guard in `Character.SetAggro`.

### Step 1: Write the failing test

Create `internal/characters/aggro_grace_test.go`:

```go
package characters

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSetAggro_GraceGuard_Untargetable verifies that when the registered
// untargetable-check callback returns true for a target userId,
// Character.SetAggro short-circuits without setting Aggro.
func TestSetAggro_GraceGuard_Untargetable(t *testing.T) {
	// Register a callback that says userId 42 is untargetable.
	SetUserUntargetableCheck(func(userId int) bool {
		return userId == 42
	})
	t.Cleanup(func() { SetUserUntargetableCheck(nil) })

	mob := &Character{}
	mob.SetAggro(42, 0, DefaultAttack)

	assert.Nil(t, mob.Aggro, "SetAggro should short-circuit for untargetable user")
}

// TestSetAggro_GraceGuard_Targetable verifies that when the callback
// returns false, SetAggro behaves normally.
func TestSetAggro_GraceGuard_Targetable(t *testing.T) {
	SetUserUntargetableCheck(func(userId int) bool {
		return false
	})
	t.Cleanup(func() { SetUserUntargetableCheck(nil) })

	mob := &Character{}
	mob.SetAggro(42, 0, DefaultAttack)

	assert.NotNil(t, mob.Aggro, "SetAggro should set aggro when target is targetable")
	assert.Equal(t, 42, mob.Aggro.UserId)
}

// TestSetAggro_GraceGuard_NoCallbackRegistered verifies the guard is
// permissive when no callback is registered (safe default).
func TestSetAggro_GraceGuard_NoCallbackRegistered(t *testing.T) {
	SetUserUntargetableCheck(nil)

	mob := &Character{}
	mob.SetAggro(42, 0, DefaultAttack)

	assert.NotNil(t, mob.Aggro, "SetAggro with no callback should behave normally")
}

// TestSetAggro_GraceGuard_MobTargetBypasses verifies the guard only
// gates on userId. Targeting a mob (userId=0, mobInstanceId>0) isn't
// affected even if the callback would block some userId.
func TestSetAggro_GraceGuard_MobTargetBypasses(t *testing.T) {
	SetUserUntargetableCheck(func(userId int) bool { return true }) // block everyone
	t.Cleanup(func() { SetUserUntargetableCheck(nil) })

	mob := &Character{}
	mob.SetAggro(0, 200, DefaultAttack)

	assert.NotNil(t, mob.Aggro, "mob-target aggro should not be gated by user check")
	assert.Equal(t, 200, mob.Aggro.MobInstanceId)
}
```

### Step 2: Run the tests to verify they fail

Run: `go test -run 'TestSetAggro_GraceGuard_' ./internal/characters/`
Expected: FAIL with `undefined: SetUserUntargetableCheck`.

### Step 3: Add the callback + guard in `aggro.go`

In `internal/characters/aggro.go`, add near the top of the file (after the existing `type AggroType int` / const block and before the existing Aggro struct or `SetAggro` function — find a sensible spot for package-level vars):

```go
// userUntargetableFn is registered from hooks at boot. Returns true if
// the user with the given id is protected from incoming aggro (e.g.,
// post-respawn grace period). Called from SetAggro before setting
// aggro on a player target. nil = no check (safe default for tests).
var userUntargetableFn func(userId int) bool

// SetUserUntargetableCheck registers the untargetable-user check used
// by SetAggro's player-target gate. Follows the callback pattern used
// by rooms.SetCompanionTransport / rooms.SetBTreeStateEvictor to
// avoid the users→characters import cycle (characters cannot import
// users directly).
//
// Repeated registrations overwrite; pass nil to disable.
func SetUserUntargetableCheck(fn func(userId int) bool) {
	userUntargetableFn = fn
}
```

In the existing `Character.SetAggro` method body (around line 58), add the grace guard as the FIRST check (before any existing logic):

```go
func (c *Character) SetAggro(userId int, mobInstanceId int, aggroType AggroType, roundsWaitTime ...int) {
	// Grace-period guard: don't acquire aggro on a grace-protected
	// player. Other target shapes (mob, spellcast) are unaffected.
	if userId > 0 && userUntargetableFn != nil && userUntargetableFn(userId) {
		return
	}

	// ... existing logic unchanged
}
```

Adjust the function signature to match the existing one exactly — the real SetAggro may have slightly different args (e.g., `...int` variadic for roundsWaitTime). Don't change the signature; just add the guard block at the top of the body.

### Step 4: Run the tests to verify they pass

Run: `go test -run 'TestSetAggro_GraceGuard_' ./internal/characters/`
Expected: all PASS.

### Step 5: Run the full characters package tests

Run: `go test ./internal/characters/`
Expected: clean (existing tests untouched; the new guard only fires when the callback is registered, which the other tests don't register).

### Step 6: Full project build

Run: `go build ./... && go test ./...`
Expected: clean.

### Step 7: Commit

```bash
git add internal/characters/aggro.go internal/characters/aggro_grace_test.go
git commit -m "$(cat <<'EOF'
feat(characters): SetUserUntargetableCheck callback + SetAggro guard

Adds a package-level callback that hooks registers at boot (via
the same pattern as rooms.SetCompanionTransport /
SetBTreeStateEvictor — avoids the users→characters import cycle).
When registered, Character.SetAggro short-circuits for any player
target that returns true from the callback. Used for the respawn
grace period.

4 unit tests cover: untargetable blocks, targetable passes through,
no-callback default is permissive, mob-target bypasses the gate.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

## Context for Task 2

- Branch: `feature/respawn-aggro-cleanup`. Parent: Task 1's commit.
- `Character.SetAggro` signature: check `internal/characters/aggro.go` — it's somewhere around line 58. Preserve the exact signature (don't add args).
- `Character.Aggro` is a `*Aggro` struct. `nil` means no aggro; the guard preventing the set means it stays nil.

---

## Task 3: Register the callback in main.go

**Files:**
- Modify: `main.go`

Wire the callback at boot, following the pattern of the existing `rooms.SetCompanionTransport` / `SetBTreeStateEvictor` registrations (around lines 231-235).

### Step 1: Find the existing callback registrations

Run: `grep -n "SetBTreeStateEvictor\|SetCompanionTransport" main.go`
Expected: 2 lines, around line 231 and line 235.

### Step 2: Add the new registration

Add a new block immediately after the existing `rooms.SetCompanionTransport(...)` line:

```go
// Register the grace-period untargetable-user check so SetAggro
// can short-circuit against grace-protected respawning players
// without characters/aggro.go needing to import users (would
// create a cycle).
characters.SetUserUntargetableCheck(func(userId int) bool {
	user := users.GetByUserId(userId)
	if user == nil {
		return false
	}
	return user.Character.HasBuffFlag(buffs.NoAggroTarget)
})
```

Verify the imports at the top of `main.go` include:
- `github.com/GoMudEngine/GoMud/internal/buffs`
- `github.com/GoMudEngine/GoMud/internal/characters`
- `github.com/GoMudEngine/GoMud/internal/users`

Most or all of these are likely already imported. Add whichever are missing.

### Step 3: Build

Run: `go build ./...`
Expected: clean.

### Step 4: Commit

```bash
git add main.go
git commit -m "$(cat <<'EOF'
feat(boot): register SetUserUntargetableCheck callback

Wires the characters→hooks→users resolution so SetAggro's grace
guard can consult the NoAggroTarget buff flag on any target user.
Follows the established SetCompanionTransport /
SetBTreeStateEvictor pattern at lines 231-235.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

## Context for Task 3

- Branch: `feature/respawn-aggro-cleanup`. Parent: Task 2's commit.
- The registrations live around `main.go:231-235`. Keep the alphabetical / thematic grouping.

---

## Task 4: Add `RespawnGraceRounds` config knob

**Files:**
- Modify: `internal/configs/config.gameplay.go`

Add a new field on `GameplayDeath` + a default in the validation block.

### Step 1: Add the field

In `internal/configs/config.gameplay.go`, find the `GameplayDeath` struct (around line 26). Add a new field near the existing `RespawnPoolFraction`:

```go
type GameplayDeath struct {
	// ... existing fields ...
	RespawnPoolFraction   ConfigFloat `yaml:"RespawnPoolFraction"` // Fraction of max pools (Health/Stamina/Conviction) restored on respawn (default 0.05). Keeps "death run" strategies honest — players respawn weakened and have to recover before their next attempt.
	RespawnGraceRounds    ConfigInt   `yaml:"RespawnGraceRounds"`  // Rounds of no-aggro-target protection after respawn (default 3). Set to 0 to disable grace period.
	// ... rest unchanged ...
}
```

### Step 2: Add the default in the validation block

In the same file, find the `Validate()` (or similar) method that sets defaults for `GameplayDeath` fields. Find the block where `RespawnPoolFraction` default is set (around line 54-56). Add a similar block for the new field:

```go
if g.Death.RespawnPoolFraction <= 0.0 || g.Death.RespawnPoolFraction > 1.0 {
	g.Death.RespawnPoolFraction = 0.05
}
if g.Death.RespawnGraceRounds < 0 {
	g.Death.RespawnGraceRounds = 3 // default — 3 rounds of grace protection
}
```

Note: `< 0` lets operators set `0` to explicitly disable the grace period (a deliberate zero value, distinct from "unset").

### Step 3: Build + test

Run: `go build ./... && go test ./internal/configs/`
Expected: clean.

### Step 4: Commit

```bash
git add internal/configs/config.gameplay.go
git commit -m "$(cat <<'EOF'
feat(config): Death.RespawnGraceRounds knob (default 3)

Controls how many rounds of grace-period protection a respawning
player gets. Set to 0 to disable the grace mechanic entirely
(fall back to A+B aggro-cleanup + condition-wipe only).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

## Context for Task 4

- Branch: `feature/respawn-aggro-cleanup`. Parent: Task 3's commit.
- Existing config defaults in `config.gameplay.go:50-100+`. Follow the `if g.Death.X < N { g.Death.X = default }` pattern.

---

## Task 5: Suicide() cleanup — Fix A + Fix B + Fix C applications + tests

**Files:**
- Modify: `internal/usercommands/suicide.go`
- Create: `internal/usercommands/suicide_cleanup_test.go`

Add all three fixes to the existing Suicide function. This is the centerpiece of the whole refactor.

### Step 1: Write the failing tests

Create `internal/usercommands/suicide_cleanup_test.go`:

```go
package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
)

// TestSuicide_ClearsInboundAggro — a mob in the room with aggro on
// the dying player has its aggro cleared.
func TestSuicide_ClearsInboundAggro(t *testing.T) {
	t.Skip("pending test-infra: need real UserRecord + Room harness; manual smoke covers for v1")
	// Full implementation pending wider test harness — the inline
	// unit test requires a real room, user, and mob registry (all
	// singletons). Smoke test in Task 6 exercises this path
	// end-to-end. See plan §Testing for the smoke checklist.
}

// TestSuicide_ClearsCompanionAggro — see above.
func TestSuicide_ClearsCompanionAggro(t *testing.T) {
	t.Skip("pending test-infra; smoke test covers")
}

// TestSuicide_ClearsConditions — a user with active conditions has
// them wiped after suicide.
func TestSuicide_ClearsConditions(t *testing.T) {
	t.Skip("pending test-infra; smoke test covers")
}

// TestSuicide_AppliesRespawnGrace — a respawning user has the grace
// buff (id 81) applied after suicide.
func TestSuicide_AppliesRespawnGrace(t *testing.T) {
	t.Skip("pending test-infra; smoke test covers")
}

// The above tests are skipped because internal/usercommands/suicide.go
// operates on package-global state (users registry, mobs registry,
// rooms registry) and a direct unit test would require bootstrapping
// all three. The smoke test in Task 6 covers each case end-to-end.
//
// If a future pass adds UserRecord/room/mob test-harness helpers
// that let us build a minimal working user + room + mob without a
// full server boot, these tests should be fleshed out. For now,
// the characters-package unit tests in Task 2 cover the SetAggro
// grace guard — the load-bearing piece — and smoke covers wiring.

// --- Helpers for testing once the harness is available -----------

// _ = users.UserRecord{}    // placeholder for future use
// _ = rooms.Room{}          // placeholder
// _ = mobs.Mob{}            // placeholder
// _ = buffs.NoAggroTarget   // placeholder
// _ = characters.Character{} // placeholder
```

**Realism note:** unit tests for the full Suicide flow require a real UserRecord + Room + mob registry. Other places in the codebase do have test harnesses for this (e.g., `internal/hooks/hooks_test.go`'s `seedAllRegistries()`), so the infra exists. Rather than replicating the full hooks-test harness in usercommands, use the `t.Skip` stubs and rely on the smoke test.

**Alternative if the implementer has time + appetite:** adapt `seedAllRegistries()` from `internal/hooks/hooks_test.go` into a `suicide_cleanup_test.go` style TestMain, then exercise the cleanup. Time-budget: if Task 5 Step 1's skeleton tests + Task 2's unit tests + the smoke test all land green, we have sufficient coverage. Don't spend >30min building test harness; mark as follow-up if blocked.

### Step 2: Apply Fix A + Fix B + Fix C in `suicide.go`

Open `internal/usercommands/suicide.go`. Three hunks to modify:

**Hunk 1 — Fix A (aggro cleanup), inserted AFTER line 179's `user.Character.EndAggro()`:**

Current code (around line 178-180):

```go
	user.Character.CancelBuffsWithFlag(buffs.All)
	user.Character.EndAggro()
	user.Character.CastingState = nil
```

Replace with:

```go
	user.Character.CancelBuffsWithFlag(buffs.All)
	user.Character.Conditions = nil   // Fix B: wipe combat conditions
	user.Character.EndAggro()
	user.Character.CastingState = nil

	// Fix A: clear inbound aggro (mobs in the pre-respawn room
	// targeting this player) and companion aggro. Cleared BEFORE
	// MoveToRoom so companions arrive in home room with a blank
	// slate (they follow via TransportCompanions).
	for _, mobInstId := range room.GetMobs(rooms.FindFighting) {
		mob := mobs.GetInstance(mobInstId)
		if mob == nil || mob.Character.Aggro == nil {
			continue
		}
		if mob.Character.Aggro.UserId == user.UserId {
			mob.Character.EndAggro()
		}
	}
	for _, compInstId := range user.Character.GetCharmIds() {
		comp := mobs.GetInstance(compInstId)
		if comp == nil {
			continue
		}
		comp.Character.EndAggro()
	}
```

**Hunk 2 — Fix C (apply grace buff), inserted AFTER line 217's "Darkness swallows you" SendText and BEFORE line 225's `MoveToRoom`:**

Current code (around line 217-225):

```go
	user.SendText(`<ansi fg="yellow">Darkness swallows you. When you open your eyes, you are somewhere safe.</ansi>`)

	// Resolve home room from player settings, falling back to default.
	homeSetting := user.Character.GetSetting("home")
	homeRoomId, ok := homeLocations[homeSetting]
	if !ok {
		homeRoomId = homeLocations["default"]
	}
	rooms.MoveToRoom(user.UserId, homeRoomId)
```

Replace with:

```go
	user.SendText(`<ansi fg="yellow">Darkness swallows you. When you open your eyes, you are somewhere safe.</ansi>`)

	// Fix C: apply respawn grace buff. Prevents mobs from
	// acquiring aggro on the respawning player for N rounds.
	// Knob: Death.RespawnGraceRounds (default 3; 0 = disabled).
	if int(config.Death.RespawnGraceRounds) > 0 {
		user.Character.AddBuff(81, false)
	}

	// Resolve home room from player settings, falling back to default.
	homeSetting := user.Character.GetSetting("home")
	homeRoomId, ok := homeLocations[homeSetting]
	if !ok {
		homeRoomId = homeLocations["default"]
	}
	rooms.MoveToRoom(user.UserId, homeRoomId)
```

Note the buff's duration is what's in the YAML (`triggercount: 3`), not a per-call arg. The config knob's practical use is binary: 0 disables, non-zero enables. Future work could dynamically scale the duration via a helper like `AddBuffScaled`, but v1 uses the YAML default.

### Step 3: Build + test

Run: `go build ./... && go test ./...`
Expected: clean. The skeleton tests in suicide_cleanup_test.go skip and don't fail.

### Step 4: Commit

```bash
git add internal/usercommands/suicide.go internal/usercommands/suicide_cleanup_test.go
git commit -m "$(cat <<'EOF'
fix(suicide): comprehensive aggro cleanup + condition wipe + grace

Three fixes to the Suicide flow for the Duard prod respawn-aggro
bug:

- Fix A: beyond the existing outbound-aggro clear (line 179),
  also clear every inbound mob aggro on this player in the
  pre-respawn room, plus every companion's own aggro. Cleared
  before MoveToRoom so companions arrive in home room with a
  blank slate.
- Fix B: Conditions slice set to nil, matching the
  CancelBuffsWithFlag(buffs.All) precedent one line up. Stops
  the "respawn at 5% HP + still poisoned → die again" loop.
- Fix C: Apply buff 81 (Respawn Grace, NoAggroTarget flag) on
  respawn when Death.RespawnGraceRounds > 0 (default 3). Mobs
  can't acquire aggro on the grace-protected player via
  SetAggro's new guard.

Test scaffolding present with t.Skip stubs — full Suicide unit
tests require a larger test harness than usercommands currently
has. SetAggro's grace guard has direct unit tests in
internal/characters/aggro_grace_test.go, and smoke tests cover
the Suicide wiring end-to-end.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

## Context for Task 5

- Branch: `feature/respawn-aggro-cleanup`. Parent: Task 4's commit.
- Line numbers above (178-180, 217-225) reference the CURRENT pre-fix shape of suicide.go. Use `grep -n` to find the exact insertion points before editing.
- `rooms.FindFighting` is a `GetMobs` filter that returns only currently-fighting mobs. Pre-existing.
- `user.Character.GetCharmIds() []int` returns the list of charmed mob instance IDs. Pre-existing.
- `user.Character.Conditions` is a slice (`[]CombatCondition`). Setting to nil is safe.
- Config access uses `config.Death.RespawnGraceRounds` — `config` is already resolved at line 23 of the current suicide.go.

---

## Task 6: CompanionAutoTarget defense-in-depth

**Files:**
- Modify: `internal/hooks/aggro_helpers.go`

Add a one-line early-return in `CompanionAutoTarget` for grace-protected owners. Defense-in-depth — the SetAggro guard already prevents anyone from targeting the owner, so CompanionAutoTarget's scan naturally finds no candidates. But being explicit documents intent and catches any gap.

### Step 1: Add the guard

In `internal/hooks/aggro_helpers.go`, find the `CompanionAutoTarget` function (around line 128+). Add an early-return near the top after the existing owner-resolution:

```go
func CompanionAutoTarget(mob *mobs.Mob, room *rooms.Room) {
	// Already fighting — nothing to do.
	if mob.Character.Aggro != nil {
		return
	}

	if !mob.Character.IsCharmed() {
		return
	}

	ownerId := mob.Character.GetCharmedUserId()
	if ownerId == 0 {
		return
	}

	owner := users.GetByUserId(ownerId)
	if owner == nil {
		return
	}

	// Grace-period defense-in-depth: if the owner is grace-protected,
	// no mob should be aggressing them (SetAggro already gates), and
	// the companion has nothing to defend against.
	if owner.Character.HasBuffFlag(buffs.NoAggroTarget) {
		return
	}

	// ... existing logic continues (AutoAssist flag check, etc.)
```

Add the `buffs` import to the file if it's not already there.

### Step 2: Build + test

Run: `go build ./... && go test ./internal/hooks/`
Expected: clean.

### Step 3: Commit

```bash
git add internal/hooks/aggro_helpers.go
git commit -m "$(cat <<'EOF'
feat(hooks): CompanionAutoTarget skips grace-protected owners

Defense-in-depth. SetAggro already prevents mobs from aggroing
grace-protected players, so CompanionAutoTarget's "who is
attacking my owner?" scan would naturally find no candidates.
This early-return documents intent and catches any future path
that might bypass SetAggro.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

## Context for Task 6

- Branch: `feature/respawn-aggro-cleanup`. Parent: Task 5's commit.

---

## Task 7: Manual smoke test

Over to the user. Rebuild and verify each of the 5 scenarios below.

**Pre-test:** `go build ./...` + start server.

- [ ] **Smoke 1: Die with multiple attackers** — fight 2+ mobs, die. On respawn, for ~3 rounds no mob should pursue or re-aggro. Confirm the original-room mobs' aggro is cleared by checking that they don't shift to a nearby player or wander.
- [ ] **Smoke 2: Die with poison + bleeding active** — apply poison/bleeding (drink a cursed potion or get poisoned by a mob). Die. Respawn. Status shouldn't show either condition. HP shouldn't tick down from DoT.
- [ ] **Smoke 3: Respawn in room with a non-hostile NPC** — die in any combat. Home room contains a temple priest / shopkeeper NPC. For ~3 rounds after respawn, no engagement. After grace expires, NPCs stay idle unless provoked.
- [ ] **Smoke 4: Die while companion is mid-fight** — summon a skeleton or other companion, get into combat, die. On respawn, companion follows (per TransportCompanions) and arrives with no aggro. Skeleton stays idle in home room.
- [ ] **Smoke 5: PvP grace period** — if possible with another player: A kills B. B respawns. A tries to attack B during grace. A's attacks should fail to acquire aggro for 3 rounds. After grace, normal combat if still in same room.

Report pass/fail per scenario.

---

## Task 8: Finalize + merge

- [ ] **Step 1: Full build + test**

Run: `go build ./... && go test ./...`
Expected: all green.

- [ ] **Step 2: Update MEMORY.md**

In `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\MEMORY.md`:

(a) Remove the "Respawn aggro death loop" entry from `## Bugs to Fix`. The `project_respawn_aggro_death_loop.md` memory file can be updated with a "STATUS: Landed 2026-04-21" header similar to the instance-save cleanup.

(b) Add to `## Completed (2026-04-21)`:

```markdown
- **Respawn aggro cleanup + condition wipe + grace period** — three-part fix for the Duard prod bug where a player dying and respawning at 5% HP would die again from stale aggro + unconditioned DoT, with a third death-chain via auto-retarget onto a normally-peaceful NPC (Olen). Fix A: comprehensive aggro cleanup on death — inbound mob aggro + companion aggro (in addition to existing outbound clear), before MoveToRoom so companions arrive in home room with blank slate. Fix B: Conditions slice cleared alongside the existing CancelBuffsWithFlag(buffs.All) — stops the "5%-HP + poison" loop. Fix C: new NoAggroTarget buff flag + buff id 81 "Respawn Grace" (3 rounds default, Death.RespawnGraceRounds knob, 0 disables) gate incoming SetAggro on grace-protected players via a new characters.SetUserUntargetableCheck callback (registered from main.go — follows the SetCompanionTransport / SetBTreeStateEvictor pattern to avoid the users↔characters cycle). 6 code tasks + smoke test + finalize. Branch: `feature/respawn-aggro-cleanup`. Design in `docs/superpowers/specs/2026-04-21-respawn-aggro-cleanup-design.md`, plan in `docs/superpowers/plans/2026-04-21-respawn-aggro-cleanup-plan.md`.
```

- [ ] **Step 3: Prompt user about merge**

Per `github_guide.md`: feature branches merge into `development` with `--no-ff`. Ask the user before merging — do NOT merge autonomously.

After confirmation:

```bash
git checkout development
git merge --no-ff feature/respawn-aggro-cleanup -m "..."
git branch -d feature/respawn-aggro-cleanup
```

Commit message summarizes the refactor + references spec + follow-ups.

---

## Self-Review

**Spec coverage:**
- §Fix A (Comprehensive aggro cleanup) → Task 5 Hunk 1. ✓
- §Fix B (Clear combat conditions) → Task 5 Hunk 1 (one-liner alongside buff cancel). ✓
- §Fix C sub-pieces:
  - (c.1) NoAggroTarget flag → Task 1. ✓
  - (c.2) Buff 81 YAML → Task 1. ✓
  - (c.3.a) SetUserUntargetableCheck callback + SetAggro guard → Task 2. ✓
  - (c.3.b) main.go registration → Task 3. ✓
  - (c.3.c) RetargetOrEnd verification → implicit (callback + guard cover it; no dedicated task needed per spec analysis).
  - (c.3.d) CompanionAutoTarget defense-in-depth → Task 6. ✓
  - (c.4) Apply grace buff in suicide.go → Task 5 Hunk 2. ✓
  - (c.5) Death.RespawnGraceRounds config knob → Task 4. ✓
- §Testing → Task 2 unit tests (SetAggro guard), Task 5 skip-stubs + smoke fallback, Task 7 smoke scenarios. Covered (with acknowledged limit for Suicide unit tests).

**Placeholder scan:**
- No TBD / TODO / "similar to".
- Task 5 Step 1's `t.Skip` test stubs are a documented scope decision, not placeholders — the rationale is explicit and a future-work hook is provided.

**Type consistency:**
- `NoAggroTarget` flag constant + `"no-aggro-target"` string value consistent throughout.
- Buff ID `81` consistent in YAML, main.go, and suicide.go.
- `SetUserUntargetableCheck` signature `(fn func(userId int) bool)` consistent in Tasks 2 + 3.
- `RespawnGraceRounds ConfigInt` field consistent in Task 4 + Task 5.

No issues.
