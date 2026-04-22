# Target Resolution Refactor — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `room.ResolveTargetActor(name, opts...) (actions.Actor, error)` (plus exported constructors `actions.NewUserActor` / `actions.NewMobActor` and three sentinel errors) and migrate all ~37 user-facing target-resolution call sites in `internal/usercommands/` and `internal/mobcommands/` to use it. Closes the latent-nil-crash class (every resolved target either comes back as a non-nil Actor or a typed error) and stops commands from re-implementing the `FindByName + GetInstance/GetByUserId + nil-check` chain.

**Architecture:** 7 commits on `feature/target-resolution-refactor` off `development`, organized per-category for bisect-friendly migration.

1. `feat(rooms): add ResolveTargetActor helper + Actor constructors` — new helper, new constructors, 7 unit tests, migrate the small set of inline `&actions.UserActor{...}` / `&actions.MobActor{...}` literals in `internal/hooks/` to use the constructors. Foundational; no commands migrated yet. ADDITIVE.
2. `refactor(usercommands): migrate combat targeting to ResolveTargetActor` — `bash`, `kick`, `grapple`, `taunt`, `trip` plus the latent-nil guard at `attack.go:27`.
3. `refactor(commands): migrate look-and-info to ResolveTargetActor` — `look`, `consider`, `show`, `skill.track`, `mobcommands/look`.
4. `refactor(commands): migrate interaction commands to ResolveTargetActor` — `ask`, `talk`, `give`, `buy`, `party invite`, `mobcommands/give`, `mobcommands/sayto`, `mobcommands/show`, `mobcommands/befriend`.
5. `refactor(commands): migrate admin/meta commands to ResolveTargetActor` — `admin.buff`, `admin.zap`, `admin.paz`, `admin.ai`, `admin.command`, `admin.deafen`, `admin.mute`, `admin.skillset`, `report`.
6. `refactor(commands): migrate skullduggery + target to ResolveTargetActor` — `skill.skullduggery.steal`, `skill.skullduggery.plant`, `skill.skullduggery.shadow`, `target`, `mobcommands/aid`, `mobcommands/givequest`.
7. (Memory + merge are post-commit work — Task 7 + Task 8 below.)

The per-category split is bisect-friendly: each commit has a bounded blast radius (one command family). Behavior is preserved end-to-end; tests should not change unless they were asserting on internal implementation details.

**Tech Stack:** Go 1.25. No new dependencies. Verification via `go build ./...`, `go vet ./...`, `go test ./...` after every commit.

**Spec:** `docs/superpowers/specs/2026-04-18-target-resolution-refactor-design.md`

**Branch:** `feature/target-resolution-refactor` off `development`.

---

## Scope Policy

This work is **scoped by the spec**. Dispositions are locked:

| Area | Disposition |
|------|-------------|
| `room.ResolveTargetActor` helper + 3 sentinel errors | **REAL ADDITION** in commit 1. New file `internal/rooms/target_resolution.go` + 7 unit tests in `internal/rooms/target_resolution_test.go`. |
| `actions.NewUserActor` / `actions.NewMobActor` constructors | **REAL ADDITION** in commit 1. Edits to `internal/actions/actor_user.go` and `internal/actions/actor_mob.go`. |
| Inline `&actions.UserActor{...}` / `&actions.MobActor{...}` in `internal/hooks/` | **MIGRATE** in commit 1 to use the new constructors. Carryover from combat-quadrant unification. |
| Inline `&actions.UserActor{...}` / `&actions.MobActor{...}` in `internal/usercommands/` and `internal/mobcommands/` | **OUT OF SCOPE.** These are constructed for non-resolution purposes (passing to `actions.Execute*`). Migrating them is a separate cleanup pass. |
| ~37 user-facing `room.FindByName + GetInstance/GetByUserId + nil` chains | **REAL MIGRATION** across commits 2-6. Each command keeps any structural divergence; only the resolution + nil-safety becomes shared. |
| `Actor` interface at `internal/actions/actor.go` | **FROZEN.** No new methods added. Use type assertions on leaves where mob-only or user-only behavior is needed. |
| `room.FindByName` signature/semantics | **UNCHANGED.** The helper wraps it; FindByName itself stays as `(playerId, mobInstanceId int)`. |
| Downstream `mobs.GetInstance(id)` lookups by known ID (~170 sites) | **OUT OF SCOPE.** Tracked separately as `mobs.SafeGetInstance` follow-up; not blocking. |
| `actions.FindAttackTarget` (the wildcard / named-target helper used by `attack.go`) | **OUT OF SCOPE.** Has its own resolution path with wildcard semantics; do not subsume. |
| `actions.InitiateCast` (used by `skill.cast.go`) | **OUT OF SCOPE.** Has its own resolution path internally. |
| `users.GetByCharacterName` calls (e.g. `admin.locate`) | **OUT OF SCOPE.** Global user lookup by character name, not in-room name resolution. |
| Mutation files (`mutation_blinding_flash`, `mutation_blinding_spit`, `mutation_toxic_bite`, `mutation_pacifism_aura`, `mutation_sonic_shout`, `mutation_healing_gel`) | **OUT OF SCOPE.** Verified during scoping: these resolve targets from `user.Character.Aggro.MobInstanceId` / `Aggro.UserId`, not from name. They do downstream lookup-by-id which is out of scope per spec. |
| `mobcommands/lookfortrouble.go` | **OUT OF SCOPE.** Iterates `room.GetPlayers()` / `room.GetMobs()` ID lists; no `FindByName` resolution. |
| `combat_shoot.go` | **N/A — no FindByName site.** The user task description listed it; verified via grep that no FindByName resolution exists in `usercommands/shoot.go`. The `shoot` command targets via Aggro state. |
| Behavior preservation | **MANDATORY.** Each migrated command must produce the same output, side effects, and error messages it did before. If a migration appears to change behavior → STOP and ask the user. |

Do NOT touch anything else. If you want to "improve" something beyond these dispositions, STOP and re-read the spec.

**Carryover scope-creep policy** (from prior passes):

- **Clear bug** surfaced during execution (unambiguous defect, e.g. a missing nil-guard outside the resolution path) → preceding `fix:` commit on the same branch BEFORE the commit that would otherwise demonstrate the issue. Log to `project_target_resolution_refactor.md` (or its successor) under a `## Surfaced During Execution` heading.
- **Ambiguous case** (test result unexpected but unclear whether production code or test is wrong) → pause and ask the user.
- **Dead code spotted incidentally** → `chore:` removal commit, separate from the feature commits.

---

## CRITICAL — Read These Before Every Commit

**Working-tree noise that MUST NEVER be staged** (carried over from prior passes):

- `.claude/settings.local.json` — dirty, ignore
- `internal/usercommands/_datafiles/feedback/bugs.txt` — dirty, ignore
- `internal/usercommands/_datafiles/feedback/suggestions.txt` — dirty, ignore
- `Screenshot 2026-04-17 084513.png` — untracked, ignore

Every `git add` in this plan **enumerates explicit file paths**. NEVER use `git add .` or `git add -A`. If `git status` after a commit shows anything unexpected staged, run `git restore --staged <path>` before proceeding.

**Memory files live OUTSIDE the repo** at:

`C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\`

Edit/Write them directly. They are NOT tracked by git. NEVER `git add` them. If `git status` shows a `project_*.md` or `feedback_*.md` staged, you have accidentally created in-repo copies — delete those copies and edit the real files at the path above.

**Actor interface must not change in this pass.** `internal/actions/actor.go` is frozen. If a leaf branch needs something not on `Actor`, type-assert (`if u, ok := target.(*actions.UserActor); ok { ... }` or `target.(*actions.MobActor).Mob.IsNonCombatant()`) gated behind `target.IsPlayer()`. Same convention combat unification established.

**Behavior is preserved end-to-end.** Each migrated command must produce the same text, side effects, and error wording it did before. If you spot an apparent behavior change during migration, STOP and ask.

**Commit format:** conventional commits (`feat:`, `refactor:`, `test:`, `docs:`, `chore:`), `Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>` trailer, heredoc format per CLAUDE.md + MEMORY.md rules.

**Pre-existing baseline** (Task 0 captures it): combat-quadrant-unification merge cleared all known failures. If a test is failing on `development` at branch-cut time, that failure is OUT OF SCOPE for this pass. Document whichever baseline you see in Task 0.

---

## Task 0: Create feature branch + capture baseline

**Files:** none.

**Estimated commit size:** 0 files changed. Branch creation only.

### Commands

- [ ] **Step 1: Verify you're on `development` and mostly clean**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git status
git branch --show-current
```

Expected: on `development`. Working tree dirty only with the documented noise above. If anything else is dirty, investigate before proceeding.

- [ ] **Step 2: Create feature branch**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git checkout -b feature/target-resolution-refactor
```

Expected: `Switched to a new branch 'feature/target-resolution-refactor'`.

- [ ] **Step 3: Baseline verification — capture the pass/fail snapshot**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./... 2>&1 | tee /tmp/target_resolution_baseline.txt
```

Expected: clean build, clean vet, clean tests. Combat-quadrant-unification merge resolved all open failures. **Record any FAIL lines in your execution notes** so they are not later blamed on this work. If anything unexpected fails → STOP and investigate. The baseline must be stable before refactoring.

From this point on, **every commit's test sweep is judged against this baseline.**

---

## Task 1: `feat(rooms)`: add `ResolveTargetActor` helper + Actor constructors

One commit. Creates the helper, exports two constructors, adds 7 unit tests, migrates inline `&actions.UserActor{...}` / `&actions.MobActor{...}` literals in `internal/hooks/` (carryover from combat-quadrant unification) to use the new constructors. Also commits the spec file and this plan file.

**Files:**
- Create: `internal/rooms/target_resolution.go` (new file, helper + 3 sentinel errors)
- Create: `internal/rooms/target_resolution_test.go` (new file, 7 unit tests)
- Modify: `internal/actions/actor_user.go` (add exported `NewUserActor`)
- Modify: `internal/actions/actor_mob.go` (add exported `NewMobActor`)
- Modify: `internal/hooks/NewRound_DoCombat.go` (5 inline-literal migrations: lines 129, 134, 138, 288, 293, 297 — verify exact line numbers post-merge)
- Modify: `internal/hooks/NewRound_DoCombat_routing_test.go` (4 inline-literal migrations: lines 212, 217, 223, 233)
- Add: `docs/superpowers/specs/2026-04-18-target-resolution-refactor-design.md` (spec file, already on disk)
- Add: `docs/superpowers/plans/2026-04-18-target-resolution-refactor.md` (this plan, already on disk)

**Estimated commit size:** ~120 lines new helper + ~250 lines new test file + 2× ~5 line constructor additions + ~10 inline-literal migrations + spec + plan. Medium commit. ADDITIVE.

**Complexity:** Low-Medium. The helper is straightforward; the unit tests need fixture seeding (use `seedAllRegistries` pattern from existing tests as a reference).

### Discovery

- [ ] **Step 1: Confirm Actor interface and constructor target files**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
sed -n '1,15p' internal/actions/actor_user.go
sed -n '1,20p' internal/actions/actor_mob.go
sed -n '1,65p' internal/actions/actor.go
```

Expected: confirm `UserActor.User *users.UserRecord`, `UserActor.Room *rooms.Room`, `MobActor.Mob *mobs.Mob`, `MobActor.Room *rooms.Room` field shapes. The constructors must match these exactly.

- [ ] **Step 2: Confirm `room.FindByName` signature**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
sed -n '1430,1437p' internal/rooms/rooms.go
```

Expected: `func (r *Room) FindByName(searchName string, findTypes ...FindFlag) (playerId int, mobInstanceId int)`. Helper wraps this.

- [ ] **Step 3: Confirm carryover inline-literal sites in `internal/hooks/`**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
grep -n "&actions\.UserActor{\|&actions\.MobActor{" internal/hooks/NewRound_DoCombat.go internal/hooks/NewRound_DoCombat_routing_test.go
```

Expected — 9 sites total in the hooks package (combat-quadrant-unification carryover):
- `internal/hooks/NewRound_DoCombat.go:129` — `def = &actions.UserActor{User: defUser, Room: defRoom}` (player attacker, player defender)
- `internal/hooks/NewRound_DoCombat.go:134` — `def = &actions.MobActor{Mob: defMob, Room: defRoom}` (player attacker, mob defender)
- `internal/hooks/NewRound_DoCombat.go:138` — `atk := &actions.UserActor{User: user, Room: uRoom}` (player attacker)
- `internal/hooks/NewRound_DoCombat.go:288` — `def = &actions.UserActor{User: defUser, Room: defRoom}` (mob attacker, player defender)
- `internal/hooks/NewRound_DoCombat.go:293` — `def = &actions.MobActor{Mob: defMob, Room: defRoom}` (mob attacker, mob defender)
- `internal/hooks/NewRound_DoCombat.go:297` — `atk := &actions.MobActor{Mob: mob, Room: mobRoom}` (mob attacker)
- `internal/hooks/NewRound_DoCombat_routing_test.go:212,217,223,233` — test fixture constructions

(Line numbers may have shifted by ±1-2 since the combat-quadrant-unification merge — re-verify with `grep -n` before editing.)

### Implementation

- [ ] **Step 4: Add `NewUserActor` constructor at the top of `internal/actions/actor_user.go`**

Insert AFTER the `var _ Actor = (*UserActor)(nil)` line (currently line 15) and BEFORE the first method `func (a *UserActor) GetCharacter()` (currently line 17):

```go
// NewUserActor wraps a UserRecord in an Actor for polymorphic combat and
// target-resolution code paths. Room is resolved lazily via the user's
// current Character.RoomId — callers passing nil rely on later
// GetRoom() calls being no-ops or on Room being set via the struct
// literal pattern. Most call sites pass an explicit room.
func NewUserActor(u *users.UserRecord) Actor {
    return &UserActor{User: u}
}
```

(If lazy room resolution is desired post-construction, prefer a separate `NewUserActorInRoom(u, room)` overload — but per the spec, the simple `NewUserActor(u)` form is what callers need. Room can be set on the resulting struct if needed: `actor := actions.NewUserActor(u).(*actions.UserActor); actor.Room = room`. The combat dispatcher / `ResolveTargetActor` use cases populate Room after construction or set it directly.)

**Decision note:** The `NewUserActor` returned from `ResolveTargetActor` will have `Room == nil`. Callers that need `.GetRoom()` on the resolved Actor (today, none of the migrated commands do — they have `room` in their own scope) must either re-set `.Room` or call the convenience pattern. For Task 1 keep the constructor simple per spec; if a migration site in Tasks 2-6 truly needs `.GetRoom()`, set `.Room` post-construction at that site.

- [ ] **Step 5: Add `NewMobActor` constructor at the top of `internal/actions/actor_mob.go`**

Insert AFTER `var _ Actor = (*MobActor)(nil)` (currently line 19) and BEFORE the first method (currently line 21):

```go
// NewMobActor wraps a Mob in an Actor for polymorphic combat and
// target-resolution code paths. Room is left nil by default; set it on
// the returned struct if needed.
func NewMobActor(m *mobs.Mob) Actor {
    return &MobActor{Mob: m}
}
```

- [ ] **Step 6: Create `internal/rooms/target_resolution.go`**

New file. Verbatim from the spec, adjusted for actual constructor names:

```go
package rooms

import (
    "errors"

    "github.com/GoMudEngine/GoMud/internal/actions"
    "github.com/GoMudEngine/GoMud/internal/mobs"
    "github.com/GoMudEngine/GoMud/internal/users"
)

// ResolveTargetOptions configures target resolution behavior.
type ResolveTargetOptions struct {
    // FindFlags filters which entities are eligible (FindAll if empty).
    FindFlags []FindFlag
    // ExcludeUserId, when > 0, hides the named user from results
    // (used for self-exclusion in commands like consider).
    ExcludeUserId int
    // ExcludeMobInstanceId, when > 0, hides the named mob from results.
    ExcludeMobInstanceId int
}

// Sentinel errors for typed handling by callers.
var (
    // ErrTargetNotFound is returned when no entity in the room matches the
    // search name (or the only matches were excluded by options).
    ErrTargetNotFound = errors.New("target not found")
    // ErrTargetVanished is returned when FindByName resolved an ID but the
    // corresponding pointer lookup (mobs.GetInstance / users.GetByUserId)
    // returned nil. Race condition: the entity left between the find and
    // the lookup, or the registry was mutated mid-call.
    ErrTargetVanished = errors.New("target vanished")
    // ErrTargetSelfExcluded is reserved for future use when distinguishing
    // "no match" from "matched but excluded by self-exclusion" matters for
    // user-facing messaging. Currently ResolveTargetActor returns
    // ErrTargetNotFound in both cases (callers don't differentiate today).
    ErrTargetSelfExcluded = errors.New("target is self")
)

// ResolveTargetActor looks up a player or mob by name in this room and
// returns it wrapped in an actions.Actor. Returns one of the sentinel
// errors above if nothing matches, the matched entity has vanished, or
// the only match was excluded.
//
// The returned Actor is a concrete *actions.UserActor or
// *actions.MobActor — callers can type-assert when they need
// type-specific behavior (e.g., mob.IsNonCombatant(), user.PartyId).
//
// Players take precedence over mobs when both match the same name.
// This matches the implicit convention in pre-refactor call sites and
// aligns with the intuition that named players are usually the
// intended target. Known limitation: nothing prevents player-mob name
// collisions; see project_name_collision_prevention.md (future work).
//
// Caller pattern:
//   target, err := room.ResolveTargetActor(name)
//   if err != nil {
//       user.SendText("You don't see them here.")  // caller controls wording
//       return true, nil
//   }
//   // ... use target uniformly ...
//   if !target.IsPlayer() {
//       mob := target.(*actions.MobActor).Mob
//       if mob.IsNonCombatant() { /* ... */ }
//   }
func (r *Room) ResolveTargetActor(name string, opts ...ResolveTargetOptions) (actions.Actor, error) {
    var o ResolveTargetOptions
    if len(opts) > 0 {
        o = opts[0]
    }

    flags := o.FindFlags
    if len(flags) == 0 {
        flags = []FindFlag{FindAll}
    }

    playerId, mobInstanceId := r.FindByName(name, flags...)

    // Apply exclusions.
    if o.ExcludeUserId > 0 && playerId == o.ExcludeUserId {
        playerId = 0
    }
    if o.ExcludeMobInstanceId > 0 && mobInstanceId == o.ExcludeMobInstanceId {
        mobInstanceId = 0
    }

    // Players take precedence over mobs (see docstring).
    if playerId > 0 {
        u := users.GetByUserId(playerId)
        if u == nil {
            return nil, ErrTargetVanished
        }
        return actions.NewUserActor(u), nil
    }
    if mobInstanceId > 0 {
        m := mobs.GetInstance(mobInstanceId)
        if m == nil {
            return nil, ErrTargetVanished
        }
        return actions.NewMobActor(m), nil
    }
    return nil, ErrTargetNotFound
}
```

**Import-cycle check:** This adds an `internal/rooms` → `internal/actions` import. Verify no cycle exists:

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
grep -l "GoMudEngine/GoMud/internal/rooms" internal/actions/*.go | head -5
```

Expected: `internal/actions/actor_user.go`, `internal/actions/actor_mob.go`, etc. all import `rooms`. So `rooms` → `actions` would create a cycle. **THIS IS A PROBLEM.**

**Resolution: keep helper in `rooms` package but use a constructor-injection pattern OR move helper to a different package.** Two paths:

  **Path A (PREFERRED — package boundary):** Move `ResolveTargetActor` to `internal/actions/target_resolution.go` as a free function taking the room as a parameter:
  ```go
  package actions

  func ResolveTargetActor(r *rooms.Room, name string, opts ...ResolveTargetOptions) (Actor, error) {
      // same body, but FindByName → r.FindByName, return NewUserActor / NewMobActor directly
  }
  ```
  Caller pattern becomes `target, err := actions.ResolveTargetActor(room, name)`. This avoids the cycle.

  **Path B (rooms-side method, but actions-package callback):** Define an abstract `Actor` interface stub or pass constructors as function parameters. More complex; rejected.

**Pick Path A.** Update the implementation: file becomes `internal/actions/target_resolution.go`, signature is `func ResolveTargetActor(r *rooms.Room, name string, opts ...ResolveTargetOptions) (Actor, error)`. The `ResolveTargetOptions` struct lives in the `actions` package. Sentinel errors live in `actions` package.

Caller pattern shifts from `room.ResolveTargetActor(name)` → `actions.ResolveTargetActor(room, name)`. Slightly less elegant but avoids the cycle and keeps `internal/actions` as the canonical home for Actor-related polymorphism.

**Update all task references in this plan accordingly.** The spec writes `room.ResolveTargetActor(...)` everywhere; the implementation reality is `actions.ResolveTargetActor(room, ...)`. Mention this explicitly in commit 1's commit message. Update `feedback_target_resolution_uses_actor.md` (Task 7) to use the actual call form.

**File path becomes:** `internal/actions/target_resolution.go` (NOT `internal/rooms/target_resolution.go`).

The full file:

```go
package actions

import (
    "errors"

    "github.com/GoMudEngine/GoMud/internal/mobs"
    "github.com/GoMudEngine/GoMud/internal/rooms"
    "github.com/GoMudEngine/GoMud/internal/users"
)

// ResolveTargetOptions configures target resolution behavior.
type ResolveTargetOptions struct {
    FindFlags            []rooms.FindFlag
    ExcludeUserId        int
    ExcludeMobInstanceId int
}

// Sentinel errors for typed handling by callers.
var (
    ErrTargetNotFound     = errors.New("target not found")
    ErrTargetVanished     = errors.New("target vanished")
    ErrTargetSelfExcluded = errors.New("target is self")
)

// ResolveTargetActor looks up a player or mob by name in the given room
// and returns it wrapped in an Actor. Players take precedence over mobs
// when both match the same name. (See package doc for the rationale.)
//
// Returns ErrTargetNotFound if no match (or only matches were excluded),
// ErrTargetVanished if FindByName returned an ID but the registry pointer
// lookup returned nil. The returned Actor is a concrete *UserActor or
// *MobActor — callers can type-assert when they need type-specific
// behavior (mob.IsNonCombatant, user.PartyId, etc.).
func ResolveTargetActor(r *rooms.Room, name string, opts ...ResolveTargetOptions) (Actor, error) {
    var o ResolveTargetOptions
    if len(opts) > 0 {
        o = opts[0]
    }

    flags := o.FindFlags
    if len(flags) == 0 {
        flags = []rooms.FindFlag{rooms.FindAll}
    }

    playerId, mobInstanceId := r.FindByName(name, flags...)

    if o.ExcludeUserId > 0 && playerId == o.ExcludeUserId {
        playerId = 0
    }
    if o.ExcludeMobInstanceId > 0 && mobInstanceId == o.ExcludeMobInstanceId {
        mobInstanceId = 0
    }

    if playerId > 0 {
        u := users.GetByUserId(playerId)
        if u == nil {
            return nil, ErrTargetVanished
        }
        return NewUserActor(u), nil
    }
    if mobInstanceId > 0 {
        m := mobs.GetInstance(mobInstanceId)
        if m == nil {
            return nil, ErrTargetVanished
        }
        return NewMobActor(m), nil
    }
    return nil, ErrTargetNotFound
}
```

- [ ] **Step 7: Create `internal/actions/target_resolution_test.go`**

New file. 7 tests per spec. Match the style of existing `internal/actions/*_test.go` (testify, `seedAllRegistries`-equivalent harness).

**Locate the existing test harness pattern first:**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
ls internal/actions/*_test.go
sed -n '1,80p' internal/actions/actions_test.go
```

Expected: an `actions_test.go` with helper functions for setting up users/rooms/mobs in the actions package's test scope. Use the same helpers; do not invent new ones.

The test file skeleton:

```go
package actions

import (
    "testing"

    "github.com/GoMudEngine/GoMud/internal/rooms"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// TestResolveTargetActor_PlayerMatch: name resolves to a player; helper
// returns a *UserActor wrapping the correct user.
func TestResolveTargetActor_PlayerMatch(t *testing.T) {
    cleanup := /* seed registries — same harness used in actions_test.go */
    defer cleanup()

    user, room := /* fixture: seed a user named "alice" into a known room */
    _ = user

    target, err := ResolveTargetActor(room, "alice")

    require.NoError(t, err)
    require.NotNil(t, target)
    assert.True(t, target.IsPlayer(), "expected UserActor (IsPlayer=true)")
    assert.Equal(t, "Alice", target.GetName())
}

// TestResolveTargetActor_MobMatch: name resolves to a mob; helper returns
// a *MobActor wrapping the correct mob.
func TestResolveTargetActor_MobMatch(t *testing.T) {
    cleanup := /* seed registries */
    defer cleanup()

    _, room := /* fixture: room with a mob named "skeleton" */

    target, err := ResolveTargetActor(room, "skeleton")

    require.NoError(t, err)
    require.NotNil(t, target)
    assert.False(t, target.IsPlayer(), "expected MobActor (IsPlayer=false)")
    assert.Contains(t, target.GetName(), "keleton") // case/prefix-tolerant
}

// TestResolveTargetActor_NotFound: no match in room → ErrTargetNotFound.
func TestResolveTargetActor_NotFound(t *testing.T) {
    cleanup := /* seed registries */
    defer cleanup()

    _, room := /* fixture: empty room */

    target, err := ResolveTargetActor(room, "nobody")

    assert.Nil(t, target)
    assert.ErrorIs(t, err, ErrTargetNotFound)
}

// TestResolveTargetActor_StalePlayerId: FindByName returns a player ID
// but the registry returns nil (simulate by deleting the user mid-test).
// → ErrTargetVanished.
func TestResolveTargetActor_StalePlayerId(t *testing.T) {
    cleanup := /* seed registries */
    defer cleanup()

    user, room := /* fixture: user "alice" in room */

    // Simulate stale ID: remove the user from the registry but leave the
    // room's player-list cache stale. (If the test harness has a
    // `users.RemoveForTest(uid)` or equivalent, use it. Otherwise, build
    // the room with a fake player-list entry that doesn't correspond to
    // a registered user.)
    /* delete user from users registry without notifying room */

    target, err := ResolveTargetActor(room, "alice")

    assert.Nil(t, target)
    assert.ErrorIs(t, err, ErrTargetVanished)
    _ = user
}

// TestResolveTargetActor_StaleMobId: FindByName returns a mob ID but the
// instance is gone. → ErrTargetVanished.
func TestResolveTargetActor_StaleMobId(t *testing.T) {
    cleanup := /* seed registries */
    defer cleanup()

    mob, room := /* fixture: mob "skeleton" in room */

    /* mobs.DestroyInstance(mob.InstanceId) — simulate stale registry */

    target, err := ResolveTargetActor(room, "skeleton")

    assert.Nil(t, target)
    assert.ErrorIs(t, err, ErrTargetVanished)
    _ = mob
}

// TestResolveTargetActor_ExcludeSelf: a user matches but is excluded via
// ExcludeUserId option. With no other matches → ErrTargetNotFound (the
// helper does not currently distinguish "not found" from "self excluded").
func TestResolveTargetActor_ExcludeSelf(t *testing.T) {
    cleanup := /* seed registries */
    defer cleanup()

    user, room := /* fixture: only user in room is "alice" */

    target, err := ResolveTargetActor(room, "alice", ResolveTargetOptions{
        ExcludeUserId: user.UserId,
    })

    assert.Nil(t, target)
    assert.ErrorIs(t, err, ErrTargetNotFound,
        "exclude-self with no other matches returns ErrTargetNotFound today; "+
            "ErrTargetSelfExcluded is reserved for a future revision when "+
            "callers need to differentiate.")
}

// TestResolveTargetActor_FindFlagFiltering: passing FindFighting only
// returns combatants. With a non-fighting mob in the room and no
// fighting mobs, name resolution returns ErrTargetNotFound.
func TestResolveTargetActor_FindFlagFiltering(t *testing.T) {
    cleanup := /* seed registries */
    defer cleanup()

    _, room := /* fixture: room contains a mob "skeleton" with no Aggro set */

    target, err := ResolveTargetActor(room, "skeleton", ResolveTargetOptions{
        FindFlags: []rooms.FindFlag{rooms.FindFighting},
    })

    assert.Nil(t, target)
    assert.ErrorIs(t, err, ErrTargetNotFound,
        "skeleton is not in combat; FindFighting must filter it out")

    // Now set Aggro and re-resolve — should find it.
    /* set mob.Character.Aggro on a fake target */
    target2, err2 := ResolveTargetActor(room, "skeleton", ResolveTargetOptions{
        FindFlags: []rooms.FindFlag{rooms.FindFighting},
    })
    require.NoError(t, err2)
    require.NotNil(t, target2)
    assert.False(t, target2.IsPlayer())
}
```

**Pseudocode-y placeholders intentional:** ground them in the real harness already in `internal/actions/actions_test.go`. Do NOT invent a new mocking framework.

- [ ] **Step 8: Migrate inline literal sites in `internal/hooks/NewRound_DoCombat.go`**

Six sites (verify line numbers via `grep -n` first):

**Line ~129** — BEFORE:
```go
def = &actions.UserActor{User: defUser, Room: defRoom}
```
AFTER:
```go
def = actions.NewUserActor(defUser)
// Set Room separately if downstream code uses GetRoom() on def.
def.(*actions.UserActor).Room = defRoom
```

**Decision:** if `def` is being passed to `handleCombatRound` and downstream phases call `def.GetRoom()`, we MUST keep the Room field set. The cleanest is to either:
  (a) leave the inline literal in the dispatcher (it's a 1-line construction) and only migrate genuinely-new sites, OR
  (b) define an internal `actions.NewUserActorInRoom(u, room)` helper.

**Pick (b) — add a second constructor variant.** Update Step 4 to also add:

```go
// NewUserActorInRoom is NewUserActor with a pre-populated room reference.
func NewUserActorInRoom(u *users.UserRecord, room *rooms.Room) Actor {
    return &UserActor{User: u, Room: room}
}
```

And in `actor_mob.go`:

```go
// NewMobActorInRoom is NewMobActor with a pre-populated room reference.
func NewMobActorInRoom(m *mobs.Mob, room *rooms.Room) Actor {
    return &MobActor{Mob: m, Room: room}
}
```

Then the migration of inline literals becomes mechanical:

  - `&actions.UserActor{User: u, Room: r}` → `actions.NewUserActorInRoom(u, r).(*actions.UserActor)` *(if downstream code asserts on the concrete type)* OR plain `actions.NewUserActorInRoom(u, r)` *(if it's used as an `actions.Actor`)*.

For the 6 sites in `NewRound_DoCombat.go`:

  - Line 129 — `def = actions.NewUserActorInRoom(defUser, defRoom)` (def is typed as `actions.Actor`)
  - Line 134 — `def = actions.NewMobActorInRoom(defMob, defRoom)`
  - Line 138 — `atk := actions.NewUserActorInRoom(user, uRoom)` (atk is `actions.Actor`)
  - Line 288 — `def = actions.NewUserActorInRoom(defUser, defRoom)`
  - Line 293 — `def = actions.NewMobActorInRoom(defMob, defRoom)`
  - Line 297 — `atk := actions.NewMobActorInRoom(mob, mobRoom)`

For the routing-test sites in `NewRound_DoCombat_routing_test.go`:

  - Line 212 — `atk = actions.NewUserActorInRoom(u1, room1).(*actions.UserActor)` (test asserts on concrete type)
  - Line 217 — `atk = actions.NewMobActorInRoom(m, room1).(*actions.MobActor)`
  - Line 223 — `def = actions.NewUserActorInRoom(u2, room1).(*actions.UserActor)`
  - Line 233 — `def = actions.NewMobActorInRoom(m, room1).(*actions.MobActor)`

(If the routing-test sites use `atk` / `def` only via the Actor interface, drop the type assertion. Verify by reading the surrounding lines.)

**Note for usercommands/mobcommands inline literals:** these are NOT migrated in this commit (per Scope Policy). They're for `actions.Execute*` calls which need the concrete `*actions.UserActor` / `*actions.MobActor` pointer for non-resolution reasons. Leave them alone.

### Verify

- [ ] **Step 9: Build + vet**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
```

Expected: clean. Common failure modes:
- Import cycle if you accidentally put helper in `internal/rooms` (Step 6 resolution).
- Constructor field-name mismatch (`User` vs lower-case).
- `actions.NewUserActorInRoom` not yet defined — Step 4/5 update missed.

- [ ] **Step 10: Scoped test — new helper unit tests**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go test ./internal/actions/... -run "TestResolveTargetActor_" -v
```

Expected: 7 tests pass.

- [ ] **Step 11: Full test sweep**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go test ./...
```

Expected: matches Task 0 baseline. The hooks-package tests (combat, NewRound_DoCombat_routing_test) MUST still pass — the inline-literal migration is behavior-preserving.

### Commit

- [ ] **Step 12: Stage and commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add \
  internal/actions/target_resolution.go \
  internal/actions/target_resolution_test.go \
  internal/actions/actor_user.go \
  internal/actions/actor_mob.go \
  internal/hooks/NewRound_DoCombat.go \
  internal/hooks/NewRound_DoCombat_routing_test.go \
  docs/superpowers/specs/2026-04-18-target-resolution-refactor-design.md \
  docs/superpowers/plans/2026-04-18-target-resolution-refactor.md
git commit -m "$(cat <<'EOF'
feat(actions): add ResolveTargetActor helper + Actor constructors

Foundation for the target-resolution refactor (see spec
docs/superpowers/specs/2026-04-18-target-resolution-refactor-design.md).

New:
  - actions.ResolveTargetActor(room, name, opts...) — consolidates
    FindByName + GetInstance/GetByUserId + nil-check into one call,
    returning a typed actions.Actor and one of three sentinel errors
    (ErrTargetNotFound, ErrTargetVanished, ErrTargetSelfExcluded).
  - actions.NewUserActor(u), actions.NewUserActorInRoom(u, room) —
    exported constructors so callers don't reach into the struct
    layout.
  - actions.NewMobActor(m), actions.NewMobActorInRoom(m, room) — dual
    constructors for mobs.
  - 7 unit tests in actions/target_resolution_test.go covering
    player-match, mob-match, not-found, stale-player race,
    stale-mob race, exclude-self, and FindFlag filtering.

Note: the spec called for the helper to live on Room as a method
(r.ResolveTargetActor); the actual implementation lives in
internal/actions because internal/actions already imports rooms,
which would create a cycle if the helper lived in rooms. Caller
pattern is actions.ResolveTargetActor(room, name) instead of
room.ResolveTargetActor(name) — one extra arg, no Romeo loss.

Also migrates the 9 inline &actions.UserActor{...} /
&actions.MobActor{...} literals in internal/hooks (carryover from
combat-quadrant unification) to use the new constructors. Pure
mechanical refactor; combat behavior unchanged.

No commands migrated yet — that's Tasks 2-6.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
git status
```

Stage ONLY the eight enumerated paths. Do NOT stage working-tree noise. Do NOT stage memory files.

### Spec-Compliance Checklist (Task 1)

- [ ] `actions.ResolveTargetActor(room, name, opts...)` exists and returns `(Actor, error)`.
- [ ] `ErrTargetNotFound`, `ErrTargetVanished`, `ErrTargetSelfExcluded` exported as package-level `var`s.
- [ ] `actions.NewUserActor(u)` and `actions.NewMobActor(m)` exported.
- [ ] `actions.NewUserActorInRoom(u, r)` and `actions.NewMobActorInRoom(m, r)` exported (helper for inline-literal migration).
- [ ] 7 unit tests added in `internal/actions/target_resolution_test.go`.
- [ ] Inline `&actions.UserActor{...}` / `&actions.MobActor{...}` literals in `internal/hooks/` migrated to constructors. (Inline literals in `internal/usercommands/` and `internal/mobcommands/` left alone — out of scope.)
- [ ] Spec file and plan file both committed.
- [ ] Actor interface at `internal/actions/actor.go` UNCHANGED.

### Code-Quality Checklist (Task 1)

- [ ] `go build ./...` clean.
- [ ] `go vet ./...` clean.
- [ ] Full `go test ./...` matches Task 0 baseline.
- [ ] No commented-out code.
- [ ] Docstrings on `ResolveTargetActor`, the sentinel errors, both constructor pairs, and the options struct.
- [ ] Commit message follows conventional-commits format and uses heredoc with the Co-Authored-By trailer.

### Rollback Plan (Task 1)

Pure ADD-ONLY (constructors are added, helper is new). If something is structurally wrong:

```bash
git revert HEAD
```

The 9 inline-literal migrations in `NewRound_DoCombat.go` are behavior-preserving; revert puts the literals back. Memory files are unaffected.

---

## Task 2: `refactor(usercommands)`: migrate combat targeting commands

Migrate 5 user-facing combat-targeting commands (`bash`, `kick`, `grapple`, `taunt`, `trip`) to `actions.ResolveTargetActor`. Plus add a defensive nil-guard at `attack.go:27` (the latent-nil-crash site mentioned in the spec — the helper itself doesn't fix it because attack.go's auto-target loop doesn't go through name resolution, but the helper's nil-safety pattern is the right model).

**Files:**
- Modify: `internal/usercommands/bash.go` (1 site, lines ~22-46)
- Modify: `internal/usercommands/kick.go` (1 site, lines ~24-47)
- Modify: `internal/usercommands/grapple.go` (1 site, lines ~22-45)
- Modify: `internal/usercommands/taunt.go` (1 site, lines ~24-53)
- Modify: `internal/usercommands/trip.go` (1 site, lines ~23-46)
- Modify: `internal/usercommands/attack.go` (1 nil-guard at line ~27-28; NO migration of the rest of attack.go — see scope policy)

**Estimated commit size:** ~50 lines of net change (5× similar shape, each ~10-15 LOC reduced to ~7-9 LOC) plus a 2-line nil-guard. Small commit.

**Complexity:** Low. All 5 commands share an identical shape (the no-aggro target-acquisition prelude). The migration is mechanical.

### Discovery

- [ ] **Step 1: Re-confirm line ranges**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
grep -n "FindByName\|GetInstance\|GetByUserId\|IsNonCombatant\|CanPvp" internal/usercommands/bash.go internal/usercommands/kick.go internal/usercommands/grapple.go internal/usercommands/taunt.go internal/usercommands/trip.go
```

Expected: each file has the FindByName at lines 22-24, with the GetInstance/GetByUserId checks immediately following.

- [ ] **Step 2: Re-confirm `attack.go:27` shape**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
sed -n '24,35p' internal/usercommands/attack.go
```

Expected:
```go
for _, mId := range room.GetMobs(rooms.FindFightingPlayer) {
    m := mobs.GetInstance(mId)
    if m.Character.Aggro == nil {     // <-- BUG: m can be nil
        continue
    }
    ...
```

The fix is a nil-guard, NOT a migration. Document why in the commit message.

### Implementation

#### bash.go

- [ ] **Step 3: Migrate `bash.go`**

BEFORE (lines 17-47):
```go
if user.Character.Aggro == nil {
    if rest == "" {
        user.SendText("Bash whom?")
        return true, nil
    }
    targetPId, targetMId := room.FindByName(rest)
    if targetPId == user.UserId {
        user.SendText("You can't bash yourself.")
        return true, nil
    }
    if targetPId == 0 && targetMId == 0 {
        user.SendText("You don't see them here.")
        return true, nil
    }
    if targetMId > 0 {
        if m := mobs.GetInstance(targetMId); m != nil && m.IsNonCombatant() {
            user.SendText(fmt.Sprintf(`You can't attack <ansi fg="mobname">%s</ansi>.`, m.Character.Name))
            return true, nil
        }
        user.Character.SetAggro(0, targetMId, characters.DefaultAttack)
    } else {
        if p := users.GetByUserId(targetPId); p != nil {
            if pvpErr := room.CanPvp(user, p); pvpErr != nil {
                user.SendText(pvpErr.Error())
                return true, nil
            }
        }
        user.Character.SetAggro(targetPId, 0, characters.DefaultAttack)
    }
}
```

AFTER:
```go
if user.Character.Aggro == nil {
    if rest == "" {
        user.SendText("Bash whom?")
        return true, nil
    }

    target, err := actions.ResolveTargetActor(room, rest, actions.ResolveTargetOptions{
        ExcludeUserId: user.UserId,
    })
    if err != nil {
        // Self-exclusion collapses to NotFound today; pre-check for self-targeting message.
        if pId, _ := room.FindByName(rest); pId == user.UserId {
            user.SendText("You can't bash yourself.")
            return true, nil
        }
        user.SendText("You don't see them here.")
        return true, nil
    }

    if !target.IsPlayer() {
        mob := target.(*actions.MobActor).Mob
        if mob.IsNonCombatant() {
            user.SendText(fmt.Sprintf(`You can't attack <ansi fg="mobname">%s</ansi>.`, mob.Character.Name))
            return true, nil
        }
        user.Character.SetAggro(0, mob.InstanceId, characters.DefaultAttack)
    } else {
        p := target.(*actions.UserActor).User
        if pvpErr := room.CanPvp(user, p); pvpErr != nil {
            user.SendText(pvpErr.Error())
            return true, nil
        }
        user.Character.SetAggro(p.UserId, 0, characters.DefaultAttack)
    }
}
```

**Note on self-targeting message preservation:** The pre-migration code distinguishes "self-targeting" ("You can't bash yourself.") from "not found" ("You don't see them here."). The helper's `ExcludeUserId` collapses self-targeting to ErrTargetNotFound. To preserve the original wording we re-call `room.FindByName(rest)` after the error to detect the self case. This adds one extra map-lookup per error path but preserves message wording exactly. Alternative: check `pId, mId := room.FindByName(rest); if pId == user.UserId { ... }` BEFORE calling ResolveTargetActor; cleaner but duplicates the lookup. The chosen pattern keeps the happy-path single-lookup.

**Imports:** `bash.go` already imports `actions`; the only import that may need removal is `mobs` and/or `users` if they're no longer referenced elsewhere in the file. Check via `goimports` after the edit.

#### kick.go, grapple.go, trip.go

- [ ] **Step 4: Migrate `kick.go`** — identical shape to bash.go. Replace lines ~19-48 with the same pattern (substitute "kick" for "bash" in error messages).

- [ ] **Step 5: Migrate `grapple.go`** — identical shape. Substitute "grapple" for "bash" in error messages.

- [ ] **Step 6: Migrate `trip.go`** — identical shape. Substitute "trip" for "bash" in error messages.

#### taunt.go

- [ ] **Step 7: Migrate `taunt.go`** — adds an `IsCharmed` check on top of the standard pattern.

BEFORE (lines 19-54):
```go
if user.Character.Aggro == nil {
    if rest == "" {
        user.SendText("Taunt whom?")
        return true, nil
    }
    targetPId, targetMId := room.FindByName(rest)
    if targetPId == user.UserId {
        user.SendText("You can't taunt yourself.")
        return true, nil
    }
    if targetPId == 0 && targetMId == 0 {
        user.SendText("You don't see them here.")
        return true, nil
    }
    if targetMId > 0 {
        if m := mobs.GetInstance(targetMId); m != nil {
            if m.IsNonCombatant() {
                user.SendText(fmt.Sprintf(`You can't attack <ansi fg="mobname">%s</ansi>.`, m.Character.Name))
                return true, nil
            }
            if m.Character.IsCharmed() {
                user.SendText(`You can't taunt a companion.`)
                return true, nil
            }
        }
        user.Character.SetAggro(0, targetMId, characters.DefaultAttack)
    } else {
        if p := users.GetByUserId(targetPId); p != nil {
            if pvpErr := room.CanPvp(user, p); pvpErr != nil {
                user.SendText(pvpErr.Error())
                return true, nil
            }
        }
        user.Character.SetAggro(targetPId, 0, characters.DefaultAttack)
    }
}
```

AFTER:
```go
if user.Character.Aggro == nil {
    if rest == "" {
        user.SendText("Taunt whom?")
        return true, nil
    }

    target, err := actions.ResolveTargetActor(room, rest, actions.ResolveTargetOptions{
        ExcludeUserId: user.UserId,
    })
    if err != nil {
        if pId, _ := room.FindByName(rest); pId == user.UserId {
            user.SendText("You can't taunt yourself.")
            return true, nil
        }
        user.SendText("You don't see them here.")
        return true, nil
    }

    if !target.IsPlayer() {
        mob := target.(*actions.MobActor).Mob
        if mob.IsNonCombatant() {
            user.SendText(fmt.Sprintf(`You can't attack <ansi fg="mobname">%s</ansi>.`, mob.Character.Name))
            return true, nil
        }
        if mob.Character.IsCharmed() {
            user.SendText(`You can't taunt a companion.`)
            return true, nil
        }
        user.Character.SetAggro(0, mob.InstanceId, characters.DefaultAttack)
    } else {
        p := target.(*actions.UserActor).User
        if pvpErr := room.CanPvp(user, p); pvpErr != nil {
            user.SendText(pvpErr.Error())
            return true, nil
        }
        user.Character.SetAggro(p.UserId, 0, characters.DefaultAttack)
    }
}
```

#### attack.go

- [ ] **Step 8: Add nil-guard at `attack.go:27`**

BEFORE (lines 26-29):
```go
for _, mId := range room.GetMobs(rooms.FindFightingPlayer) {
    m := mobs.GetInstance(mId)
    if m.Character.Aggro == nil {
        continue
    }
```

AFTER:
```go
for _, mId := range room.GetMobs(rooms.FindFightingPlayer) {
    m := mobs.GetInstance(mId)
    if m == nil || m.Character.Aggro == nil {
        continue
    }
```

**This is NOT a ResolveTargetActor migration** — `attack.go`'s auto-target path uses `room.GetMobs(FindFightingPlayer)` to enumerate IDs, not name resolution. The fix is a defensive nil-guard for the registry-race case the helper protects elsewhere. Document this in the commit message.

**No other attack.go changes** — its named-target path uses `actions.FindAttackTarget` (its own resolution helper with wildcard semantics), and its downstream `mobs.GetInstance(attackMobInstanceId)` lookups (lines 133, 137, 150, 212, 283, 227, etc.) are downstream-by-known-ID lookups (out of scope per spec).

### Verify

- [ ] **Step 9: Build + vet**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
```

Expected: clean. Common failure: leftover unused import of `mobs` or `users` after migration. `goimports -w internal/usercommands/{bash,kick,grapple,taunt,trip}.go` if needed.

- [ ] **Step 10: Scoped tests**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go test ./internal/usercommands/... -count=1
```

Expected: matches baseline.

- [ ] **Step 11: Full test sweep**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go test ./... -count=1
```

Expected: matches Task 0 baseline.

### Commit

- [ ] **Step 12: Stage and commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add \
  internal/usercommands/bash.go \
  internal/usercommands/kick.go \
  internal/usercommands/grapple.go \
  internal/usercommands/taunt.go \
  internal/usercommands/trip.go \
  internal/usercommands/attack.go
git commit -m "$(cat <<'EOF'
refactor(usercommands): migrate combat targeting to ResolveTargetActor

Stage 2 of target-resolution refactor. Migrates the no-aggro target-
acquisition prelude in five user-facing combat commands to the new
actions.ResolveTargetActor helper:

  - bash, kick, grapple, taunt, trip

All five share an identical shape — a name-resolution chain followed
by an IsNonCombatant check (for mobs) or a CanPvp check (for players).
Post-migration the resolution + nil-safety is shared; the leaf
divergence (NonCombatant for mobs, CanPvp for players) is gated via
type assertion behind target.IsPlayer().

Self-targeting message wording is preserved by re-calling
room.FindByName on the error path to detect the self case (the
ExcludeUserId option collapses self-targeting to ErrTargetNotFound;
this is the documented current behavior).

Also adds a defensive nil-guard at attack.go:27 — the auto-target
loop dereferences mobs.GetInstance(mId) without checking nil. This
is NOT a ResolveTargetActor migration (the loop iterates IDs from
room.GetMobs, not from name resolution), but it closes a latent
nil-crash site adjacent to the spec's example. The helper itself
owns the name-resolution-side nil-crash class going forward.

Behavior preserved: same error wording, same side effects, same
SetAggro call shape.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
git status
```

### Spec-Compliance Checklist (Task 2)

- [ ] All 5 combat-targeting commands use `actions.ResolveTargetActor`.
- [ ] Self-targeting message wording is preserved verbatim ("You can't bash yourself.", etc.).
- [ ] NonCombatant check is gated on `!target.IsPlayer()` (mob-only behavior).
- [ ] CanPvp check is gated on `target.IsPlayer()` (player-only behavior).
- [ ] `attack.go:27` nil-guard added.
- [ ] No other attack.go changes (the spec is explicit: most attack.go GetInstance calls are downstream-only).
- [ ] Actor interface UNCHANGED.

### Code-Quality Checklist (Task 2)

- [ ] `go build ./...` clean.
- [ ] `go vet ./...` clean.
- [ ] Full `go test ./...` matches baseline.
- [ ] No unused imports remain.
- [ ] No commented-out code.

### Rollback Plan (Task 2)

```bash
git revert HEAD
```

Each command was an independent migration; revert restores all 5 to their pre-Task-2 state. Task 1's helper stays in place.

---

## Task 3: `refactor(commands)`: migrate look-and-info commands

Migrate 5 commands that resolve a target by name and then display info about it. Lowest-risk batch — pure resolution + display, no combat side effects.

**Files:**
- Modify: `internal/usercommands/look.go` (1 site, lines ~79-156)
- Modify: `internal/usercommands/consider.go` (1 site, lines ~26-67)
- Modify: `internal/usercommands/show.go` (1 site, lines ~42-110)
- Modify: `internal/usercommands/skill.track.go` (1 site, lines ~225-246)
- Modify: `internal/mobcommands/look.go` (1 site, lines ~91-127)

**Estimated commit size:** ~70 lines net delta. Small-medium.

**Complexity:** Low. All five are display-only paths.

### Discovery

- [ ] **Step 1: Re-confirm line ranges**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
grep -n "FindByName" internal/usercommands/look.go internal/usercommands/consider.go internal/usercommands/show.go internal/usercommands/skill.track.go internal/mobcommands/look.go
```

### Implementation

#### look.go (usercommands)

- [ ] **Step 2: Migrate `usercommands/look.go`**

BEFORE (lines 79-150):
```go
playerId, mobId := room.FindByName(lookAt)

if playerId > 0 || mobId > 0 {

    user.Character.OnStatUse("perception", user.UserId)

    statusTxt := ""
    invTxt := ""

    if playerId > 0 {
        u := *users.GetByUserId(playerId)

        if !isSneaking {
            u.SendText(...)
            room.SendTextVisual(...)
        }
        descTxt, _ := templates.Process("character/description", u.Character, user.UserId)
        user.SendText(descTxt)

        itemNames := []string{}
        for _, item := range u.Character.Items {
            itemNames = append(itemNames, item.DisplayName())
        }
        invData := map[string]any{`Equipment`: &u.Character.Equipment, `ItemNames`: itemNames}
        inventoryTxt, _ := templates.Process("character/inventory-look", invData, user.UserId)
        user.SendText(inventoryTxt)

    } else if mobId > 0 {
        m := mobs.GetInstance(mobId)
        if m == nil {
            user.SendText("You don't see them here.")
            return true, nil
        }
        if !isSneaking {
            targetName := m.Character.GetMobName(0).String()
            room.SendTextVisual(...)
        }
        descTxt, _ := templates.Process("character/description", &m.Character, user.UserId)
        user.SendText(descTxt)

        itemNames := []string{}
        for _, item := range m.Character.Items {
            itemNames = append(itemNames, item.DisplayName())
        }
        invData := map[string]any{`Equipment`: &m.Character.Equipment, `ItemNames`: itemNames}
        inventoryTxt, _ := templates.Process("character/inventory-look", invData, user.UserId)
        user.SendText(inventoryTxt)
    }

    user.SendText(statusTxt)
    user.SendText(invTxt)

    return true, nil
}
```

AFTER:
```go
target, err := actions.ResolveTargetActor(room, lookAt)
if err == nil {
    user.Character.OnStatUse("perception", user.UserId)

    if target.IsPlayer() {
        u := target.(*actions.UserActor).User

        if !isSneaking {
            u.SendText(
                fmt.Sprintf(`<ansi fg="username">%s</ansi> is looking at you.`, user.Character.Name),
            )
            room.SendTextVisual(
                fmt.Sprintf(`<ansi fg="username">%s</ansi> is looking at <ansi fg="username">%s</ansi>.`, user.Character.Name, u.Character.Name),
                u.UserId)
        }

        descTxt, _ := templates.Process("character/description", u.Character, user.UserId)
        user.SendText(descTxt)

        itemNames := []string{}
        for _, item := range u.Character.Items {
            itemNames = append(itemNames, item.DisplayName())
        }
        invData := map[string]any{`Equipment`: &u.Character.Equipment, `ItemNames`: itemNames}
        inventoryTxt, _ := templates.Process("character/inventory-look", invData, user.UserId)
        user.SendText(inventoryTxt)

    } else {
        m := target.(*actions.MobActor).Mob

        if !isSneaking {
            targetName := m.Character.GetMobName(0).String()
            room.SendTextVisual(
                fmt.Sprintf(`<ansi fg="username">%s</ansi> is looking at %s.`, user.Character.Name, targetName),
                user.UserId,
            )
        }

        descTxt, _ := templates.Process("character/description", &m.Character, user.UserId)
        user.SendText(descTxt)

        itemNames := []string{}
        for _, item := range m.Character.Items {
            itemNames = append(itemNames, item.DisplayName())
        }
        invData := map[string]any{`Equipment`: &m.Character.Equipment, `ItemNames`: itemNames}
        inventoryTxt, _ := templates.Process("character/inventory-look", invData, user.UserId)
        user.SendText(inventoryTxt)
    }

    return true, nil
}
// fall through to container / noun / pet lookup branches below
```

**Pre-migration nuance:** the original `u := *users.GetByUserId(playerId)` does a *value copy* of UserRecord (note the `*` deref in `*users.GetByUserId(...)` followed by `.` access; this is `(*ptr).field`). The post-migration code uses `u := target.(*actions.UserActor).User` which is a pointer. All field accesses below should still work because Go auto-dereferences. Verify post-edit that `u.SendText(...)`, `u.Character.Name`, `u.Character.Items`, etc. all still typecheck.

**Important — pre-migration nil deref:** the original code at line 91 (`u := *users.GetByUserId(playerId)`) WILL CRASH if `users.GetByUserId(playerId)` returns nil. This is a latent crash that the helper closes — `ResolveTargetActor` returns `ErrTargetVanished` instead. The migrated code is strictly safer.

**Post-migration error path (when ResolveTargetActor fails):** the code falls through to the container / noun / pet lookup branches (lines 159-end). The pre-migration code already had this fall-through (the `if playerId > 0 || mobId > 0 { ... return true, nil }` branch only triggered when something matched). Behavior preserved.

#### consider.go

- [ ] **Step 3: Migrate `usercommands/consider.go`**

BEFORE (lines 19-90):
```go
if len(args) > 0 {
    lookAt := args[0]

    playerId, mobId := room.FindByName(lookAt)
    if playerId == user.UserId {
        playerId = 0
    }

    if playerId > 0 || mobId > 0 {
        user.Character.OnStatUse("perception", user.UserId)

        ratio := 0.0
        considerType := "mob"
        considerName := "nobody"

        if playerId > 0 {
            u := users.GetByUserId(playerId)
            p1 := combat.PowerScore(*user.Character)
            p2 := combat.PowerScore(*u.Character)
            if p2 > 0 { ratio = p1 / p2 }
            considerType = "user"
            considerName = u.Character.Name
        } else if mobId > 0 {
            m := mobs.GetInstance(mobId)
            if m == nil {
                user.SendText("You don't see them here.")
                return true, nil
            }
            p1 := combat.PowerScore(*user.Character)
            p2 := combat.PowerScore(m.Character)
            if p2 > 0 { ratio = p1 / p2 }
            considerType = "mob"
            considerName = m.Character.Name
        }

        prediction := /* ratio-based string */
        user.SendText(...)
        user.SendText(...)
    }
}
```

AFTER:
```go
if len(args) > 0 {
    lookAt := args[0]

    target, err := actions.ResolveTargetActor(room, lookAt, actions.ResolveTargetOptions{
        ExcludeUserId: user.UserId,
    })
    if err != nil {
        // Pre-migration silently no-oped on no-match (no else branch).
        // For ErrTargetVanished, surface a message because the original
        // code had a "You don't see them here." path on stale-mob.
        if err == actions.ErrTargetVanished {
            user.SendText("You don't see them here.")
        }
        return true, nil
    }

    user.Character.OnStatUse("perception", user.UserId)

    p1 := combat.PowerScore(*user.Character)
    var p2 float64
    var considerType, considerName string
    if target.IsPlayer() {
        u := target.(*actions.UserActor).User
        p2 = combat.PowerScore(*u.Character)
        considerType = "user"
        considerName = u.Character.Name
    } else {
        m := target.(*actions.MobActor).Mob
        p2 = combat.PowerScore(m.Character)
        considerType = "mob"
        considerName = m.Character.Name
    }

    ratio := 0.0
    if p2 > 0 {
        ratio = p1 / p2
    }

    prediction := `<ansi fg="red-bold">You will not survive this fight</ansi>`
    if ratio > 4 {
        prediction = `<ansi fg="blue-bold">They pose no threat to you</ansi>`
    } else if ratio > 3 {
        prediction = `<ansi fg="green">You hold a clear advantage</ansi>`
    } else if ratio > 2 {
        prediction = `<ansi fg="green">The odds favor you</ansi>`
    } else if ratio > 1 {
        prediction = `<ansi fg="yellow">An even contest — tread carefully</ansi>`
    } else if ratio > 0.5 {
        prediction = `<ansi fg="red-bold">They have the upper hand</ansi>`
    } else if ratio > 0 {
        prediction = `<ansi fg="red-bold">You are severely outmatched</ansi>`
    }

    user.SendText(
        fmt.Sprintf(`You consider <ansi fg="%sname">%s</ansi>...`, considerType, considerName),
    )
    user.SendText(
        fmt.Sprintf(`Your instincts tell you: %s`, prediction),
    )
}
```

**Behavior nuance:** the pre-migration code on `ErrTargetNotFound` silently returned without messaging the user. The migrated code preserves that. On `ErrTargetVanished` (stale mob ID) the pre-migration code DID message ("You don't see them here.") — preserved.

#### show.go

- [ ] **Step 4: Migrate `usercommands/show.go`**

The show command has a player-branch and a mob-branch with different room-broadcast wording. Migrate the resolution + keep both branches.

BEFORE (lines 42-114):
```go
playerId, mobId := room.FindByName(targetName)

if playerId > 0 {
    user.Character.CancelBuffsWithFlag(buffs.Hidden)
    targetUser := users.GetByUserId(playerId)
    if showItem.ItemId > 0 {
        user.SendText(...)
        targetUser.SendText(...)
        targetUser.SendText("\n" + showItem.GetLongDescription() + "\n")
        room.SendTextVisual(...)
    } else {
        user.SendText("Something went wrong.")
    }
    return true, nil
}

if mobId > 0 {
    user.Character.CancelBuffsWithFlag(buffs.Hidden)
    targetMob := mobs.GetInstance(mobId)
    if targetMob != nil {
        if showItem.ItemId > 0 {
            user.SendText(...)
            room.SendTextVisual(...)
        } else {
            user.SendText("Something went wrong.")
        }
    }
    return true, nil
}

user.SendText("Who???")
return true, nil
```

AFTER:
```go
target, err := actions.ResolveTargetActor(room, targetName)
if err != nil {
    user.SendText("Who???")
    return true, nil
}

user.Character.CancelBuffsWithFlag(buffs.Hidden)

if showItem.ItemId == 0 {
    user.SendText("Something went wrong.")
    return true, nil
}

if target.IsPlayer() {
    targetUser := target.(*actions.UserActor).User
    user.SendText(
        fmt.Sprintf(`You show the <ansi fg="item">%s</ansi> to <ansi fg="username">%s</ansi>.`, showItem.DisplayName(), targetUser.Character.Name),
    )
    targetUser.SendText(
        fmt.Sprintf(`<ansi fg="username">%s</ansi> shows you their <ansi fg="item">%s</ansi>.`, user.Character.Name, showItem.DisplayName()),
    )
    targetUser.SendText("\n" + showItem.GetLongDescription() + "\n")
    room.SendTextVisual(
        fmt.Sprintf(`<ansi fg="username">%s</ansi> shows their <ansi fg="item">%s</ansi> to <ansi fg="username">%s</ansi>.`, user.Character.Name, showItem.DisplayName(), targetUser.Character.Name),
        targetUser.UserId,
        user.UserId)
} else {
    targetMob := target.(*actions.MobActor).Mob
    user.SendText(
        fmt.Sprintf(`You show the <ansi fg="item">%s</ansi> to <ansi fg="mobname">%s</ansi>.`, showItem.DisplayName(), targetMob.Character.Name),
    )
    room.SendTextVisual(
        fmt.Sprintf(`<ansi fg="username">%s</ansi> shows their <ansi fg="item">%s</ansi> to <ansi fg="mobname">%s</ansi>.`, user.Character.Name, showItem.DisplayName(), targetMob.Character.Name),
        user.UserId,
    )
}

return true, nil
```

**Behavior nuance:** pre-migration code only called `CancelBuffsWithFlag(buffs.Hidden)` AFTER finding a target (preserved). The "Something went wrong." message fires when the player has no item to show (preserved). On not-found / vanished, the pre-migration code falls through to "Who???" — preserved.

#### skill.track.go

- [ ] **Step 5: Migrate `usercommands/skill.track.go`**

BEFORE (lines 225-246):
```go
foundPlayerId, foundMobId := room.FindByName(rest, rooms.FindAll)

if foundPlayerId > 0 {
    foundUser := users.GetByUserId(foundPlayerId)
    if foundUser != nil {
        user.SendText(
            fmt.Sprintf(`<ansi fg="username">%s</ansi> is in the room with you!`, foundUser.Character.Name))
        return true, nil
    }
}

if foundMobId > 0 {
    foundMob := mobs.GetInstance(foundMobId)
    if foundMob != nil {
        user.SendText(
            fmt.Sprintf(`<ansi fg="mobname">%s</ansi> is in the room with you!`, foundMob.Character.Name))
        return true, nil
    }
}
```

AFTER:
```go
target, err := actions.ResolveTargetActor(room, rest, actions.ResolveTargetOptions{
    FindFlags: []rooms.FindFlag{rooms.FindAll},
})
if err == nil {
    if target.IsPlayer() {
        u := target.(*actions.UserActor).User
        user.SendText(
            fmt.Sprintf(`<ansi fg="username">%s</ansi> is in the room with you!`, u.Character.Name))
    } else {
        m := target.(*actions.MobActor).Mob
        user.SendText(
            fmt.Sprintf(`<ansi fg="mobname">%s</ansi> is in the room with you!`, m.Character.Name))
    }
    return true, nil
}
// fall through to active-tracking / nearby-room scan below
```

**Behavior nuance:** the original code had two independent `if` blocks (no `else if`); both could potentially trigger — but only one ID can be non-zero from FindByName for a given name (assuming no collision). The migrated code uses the helper's player-precedence, which matches the original's "check player first" structure.

#### mobcommands/look.go

- [ ] **Step 6: Migrate `mobcommands/look.go`**

BEFORE (lines 91-127):
```go
playerId, mobId := room.FindByName(lookAt)

if playerId > 0 || mobId > 0 {
    if playerId > 0 {
        u := *users.GetByUserId(playerId)
        if !isSneaking {
            u.SendText(...)
            room.SendTextVisual(...)
        }
    } else if mobId > 0 {
        m := mobs.GetInstance(mobId)
        if m == nil {
            return true, nil
        }
        if !isSneaking {
            targetName := m.Character.GetMobName(0).String()
            room.SendTextVisual(...)
        }
    }
    return true, nil
}
```

AFTER:
```go
target, err := actions.ResolveTargetActor(room, lookAt)
if err == nil {
    if target.IsPlayer() {
        u := target.(*actions.UserActor).User
        if !isSneaking {
            u.SendText(
                fmt.Sprintf(`<ansi fg="mobname">%s</ansi> is looking at you.`, mob.Character.Name),
            )
            room.SendTextVisual(
                fmt.Sprintf(`<ansi fg="mobname">%s</ansi> is looking at <ansi fg="username">%s</ansi>.`, mob.Character.Name, u.Character.Name),
                u.UserId)
        }
    } else {
        m := target.(*actions.MobActor).Mob
        if !isSneaking {
            targetName := m.Character.GetMobName(0).String()
            room.SendTextVisual(
                fmt.Sprintf(`<ansi fg="mobname">%s</ansi> is looking at %s.`, mob.Character.Name, targetName),
            )
        }
    }
    return true, nil
}
// fall through to body-equipment / noun / pet lookups below
```

**Pre-migration nil-deref:** original code at line 97 (`u := *users.GetByUserId(playerId)`) crashes if user vanished. Helper closes this.

**Imports for mobcommands/look.go:** the `actions` package isn't currently imported (it's only used downstream in shared helpers). Add `"github.com/GoMudEngine/GoMud/internal/actions"` to imports.

### Verify

- [ ] **Step 7: Build + vet + tests**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./... -count=1
```

Expected: matches baseline. Common failure modes: unused imports of `mobs` or `users` after migration, missing `fmt` import (taunt-error format strings).

### Commit

- [ ] **Step 8: Stage and commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add \
  internal/usercommands/look.go \
  internal/usercommands/consider.go \
  internal/usercommands/show.go \
  internal/usercommands/skill.track.go \
  internal/mobcommands/look.go
git commit -m "$(cat <<'EOF'
refactor(commands): migrate look-and-info to ResolveTargetActor

Stage 3 of target-resolution refactor. Migrates 5 display-only
target-resolution sites:

  - usercommands/look (look at <target>)
  - usercommands/consider (consider <target>)
  - usercommands/show (show <item> <target>)
  - usercommands/skill.track (track <target>)
  - mobcommands/look (mob's look-at-target path)

Each command keeps its display divergence (player descriptions vs
mob descriptions, item-broadcast wording, etc.) but now shares the
single-call resolution + nil-safety. Two latent nil-deref crashes
in look.go (line 91, *users.GetByUserId result deref) and
mobcommands/look.go (line 97, same pattern) are closed by the
helper.

The consider command preserves its existing "silently no-op on
not-found / message on vanished" behavior.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
git status
```

### Spec-Compliance Checklist (Task 3)

- [ ] All 5 commands use `actions.ResolveTargetActor`.
- [ ] Display divergence (player desc vs mob desc, broadcast wording) preserved per command.
- [ ] Pre-migration `u := *users.GetByUserId(...)` value-copy patterns replaced with pointer access; field reads still typecheck.
- [ ] `consider` self-exclusion via `ExcludeUserId` option.
- [ ] `skill.track` passes `FindFlags: []rooms.FindFlag{rooms.FindAll}` to preserve the explicit FindAll filter.
- [ ] Actor interface UNCHANGED.

### Code-Quality Checklist (Task 3)

- [ ] `go build ./...` clean.
- [ ] `go vet ./...` clean.
- [ ] Full `go test ./...` matches baseline.
- [ ] Unused imports removed (`mobs`, `users` may no longer be needed in some files).

### Rollback Plan (Task 3)

```bash
git revert HEAD
```

---

## Task 4: `refactor(commands)`: migrate interaction commands

Highest-divergence batch. 9 sites where user/mob paths diverge structurally — `give` has quest-engine + btree hooks per target type; `ask` is mob-only; `party invite` is player-only. Each command keeps its divergence; only the resolution + nil-safety becomes shared.

**Files:**
- Modify: `internal/usercommands/ask.go` (1 site, lines ~67-75)
- Modify: `internal/usercommands/talk.go` (1 site, lines ~39-50)
- Modify: `internal/usercommands/give.go` (1 site, lines ~67-252) — most divergent
- Modify: `internal/usercommands/buy.go` (1 site, lines ~46-63)
- Modify: `internal/usercommands/party.go` (1 site, lines ~167-186) — player-only
- Modify: `internal/mobcommands/give.go` (1 site, lines ~56-129)
- Modify: `internal/mobcommands/sayto.go` (3 sites, lines ~27, 92, 130)
- Modify: `internal/mobcommands/show.go` (1 site, lines ~39-91)
- Modify: `internal/mobcommands/befriend.go` (1 site, lines ~29-55) — player-only

**Estimated commit size:** ~120 lines net delta. Medium commit. The give/give symmetry doesn't reduce LOC as much as expected because of quest-engine + btree branching.

**Complexity:** Medium-High. `give` is structurally divergent (player path vs mob path with quest engine intercept); `ask` and `befriend` are leaf-only (mob target / player target respectively); `sayto` has 3 sites that all do the same migration.

### Implementation

#### ask.go (mob-only target)

- [ ] **Step 1: Migrate `usercommands/ask.go`**

The `ask` command targets only mobs (it's how players talk to NPCs). Helper resolves uniformly; ask errors if the target is a player.

BEFORE (lines 67-75):
```go
_, mobId := room.FindByName(searchName)

if mobId > 0 {

    mob := mobs.GetInstance(mobId)
    if mob == nil {
        user.SendText(`Nobody found by that name`)
        return true, nil
    }

    args = args[1:]
    // ... entire mob-side logic (LLM, behavior tree, dialogue) follows ...
```

AFTER:
```go
target, err := actions.ResolveTargetActor(room, searchName)
if err != nil {
    user.SendText(`ask who what?`)
    return true, nil
}
if target.IsPlayer() {
    user.SendText(`You can't ask another player.`)
    return true, nil
}

mob := target.(*actions.MobActor).Mob

args = args[1:]
// ... entire mob-side logic (LLM, behavior tree, dialogue) follows, unchanged ...
```

**Behavior nuance:** the pre-migration code at line 67 ignored the `playerId` return from `FindByName` (used `_`), so player-name matches silently fell through to the "ask who what?" branch at line 171. The migrated code makes the player-target case explicit with a friendlier "You can't ask another player." message. **This is a deliberate behavior IMPROVEMENT, not preservation.** Per spec: "Mob-only commands ... use `ResolveTargetActor(name)` and immediately error on `target.IsPlayer()`: 'You can't ask another player.'" Confirm with user before merging if behavior preservation is strict.

**Alternative if strict preservation is required:** drop the IsPlayer error and just fall through:
```go
target, err := actions.ResolveTargetActor(room, searchName)
if err != nil || target.IsPlayer() {
    user.SendText(`ask who what?`)
    return true, nil
}
mob := target.(*actions.MobActor).Mob
```
This preserves the original silent-fall-through behavior. Pick one in the Discovery step and stick with it; default is the spec's "explicit error" form.

#### talk.go (mob-only target)

- [ ] **Step 2: Migrate `usercommands/talk.go`**

BEFORE (lines 39-50):
```go
_, mobId := room.FindByName(searchName)

if mobId <= 0 {
    user.SendText(`Talk to whom?`)
    return true, nil
}

mob := mobs.GetInstance(mobId)
if mob == nil {
    user.SendText(`Talk to whom?`)
    return true, nil
}
```

AFTER:
```go
target, err := actions.ResolveTargetActor(room, searchName)
if err != nil || target.IsPlayer() {
    user.SendText(`Talk to whom?`)
    return true, nil
}
mob := target.(*actions.MobActor).Mob
```

**Behavior nuance:** the pre-migration code (`_, mobId := ...`) ignored the playerId. So player-name matches fell through to "Talk to whom?" — preserved by `target.IsPlayer()` rejecting them with the same message.

#### give.go (most divergent)

- [ ] **Step 3: Migrate `usercommands/give.go`**

This is the hardest migration. The give command has THREE branches:
  1. Player target → use GiveItemToChar / GiveGoldToChar with target's UserId
  2. Mob target → quest-engine intercept + btree handler + GiveItemToChar with mob InstanceId
  3. Pet target → fallback to FindByPetName (different lookup mechanism, not changed)

Migrate ONLY the player/mob resolution. The pet path (lines 257-291) uses `room.FindByPetName(giveWho)` which is a different mechanism — leave it alone.

BEFORE (line 67):
```go
playerId, mobId := room.FindByName(giveWho)

if playerId > 0 {
    // ~70 lines of player-target logic (item give, gold give, self-gold-into-pocket)
    return true, nil
}

if mobId > 0 {
    // ~100 lines of mob-target logic (quest engine intercept, btree, item give, gold give, gear-up emote)
    return true, nil
}

// pet path (unchanged)
petUserId := room.FindByPetName(giveWho)
// ...

user.SendText(`Who??? (...)`)
return true, nil
```

AFTER:
```go
target, err := actions.ResolveTargetActor(room, giveWho)
if err == nil {
    if target.IsPlayer() {
        targetUser := target.(*actions.UserActor).User

        user.Character.CancelBuffsWithFlag(buffs.Hidden)

        // ~70 lines of player-target logic, but with `targetUser` already
        // resolved (delete the inline `users.GetByUserId(playerId)` call;
        // delete the `targetUser.UserId == user.UserId` self-give-into-pocket
        // check still uses targetUser.UserId so it stays as-is).
        // ...
        return true, nil
    } else {
        m := target.(*actions.MobActor).Mob

        user.Character.CancelBuffsWithFlag(buffs.Hidden)

        // ~100 lines of mob-target logic — drop the `mobs.GetInstance(mobId)`
        // and `if m != nil` wrapper; m is non-nil from the helper.
        // ...
        return true, nil
    }
}

// pet path (unchanged) — falls through here on ErrTargetNotFound
petUserId := room.FindByPetName(giveWho)
// ...

user.SendText(`Who??? (...)`)
return true, nil
```

**Per-line transcription required for the implementer:** the file has complex internal logic per branch. The implementer should:
  1. Wrap the existing `if playerId > 0 { ... }` block content into the `if target.IsPlayer() { ... }` branch.
  2. Wrap the existing `if mobId > 0 { ... }` block content into the `else { ... }` branch.
  3. Replace the inline `targetUser := users.GetByUserId(playerId)` with `targetUser := target.(*actions.UserActor).User` at the top of the player branch.
  4. Replace the inline `m := mobs.GetInstance(mobId); if m != nil { ... }` with `m := target.(*actions.MobActor).Mob` at the top of the mob branch (drop the nil-guard wrapper because helper guarantees non-nil).
  5. Keep all 4 inline `userActor := &actions.UserActor{User: user, Room: room}` constructions inside the body — those are for `actions.GiveItemToChar`/`GiveGoldToChar` calls and out of scope (per Scope Policy).

**Critical: preserve the quest-engine intercept order.** `give_test.go:78` (TestGive_QuestEngineInterceptsBeforeBtreePlayerGive) locks the quest-engine-before-btree-before-actions.GiveItemToChar order in the mob branch. Migration must NOT reorder these.

#### buy.go

- [ ] **Step 4: Migrate `usercommands/buy.go`**

`buy` has an optional "from <merchant>" target. Used to filter the merchant scan loop. Pre-migration code stores the IDs and uses them as filter values; doesn't directly construct the Actor.

BEFORE (lines 44-63):
```go
args := util.SplitButRespectQuotes(strings.ToLower(rest))
if len(args) >= 3 {
    if args[len(args)-2] == `from` {
        targetUserId, targetMobInstanceId = room.FindByName(args[len(args)-1])

        if user.UserId == targetUserId {
            user.SendText("You can't buy from yourself.")
            return true, nil
        }

        if targetUserId == 0 && targetMobInstanceId == 0 {
            user.SendText("Visit a merchant to purchase objects or services.")
            return true, nil
        }

        itemname = strings.Join(args[0:len(args)-2], ` `)
    }
}
```

AFTER:
```go
args := util.SplitButRespectQuotes(strings.ToLower(rest))
if len(args) >= 3 {
    if args[len(args)-2] == `from` {
        target, err := actions.ResolveTargetActor(room, args[len(args)-1], actions.ResolveTargetOptions{
            ExcludeUserId: user.UserId,
        })
        if err == nil {
            if target.IsPlayer() {
                targetUserId = target.(*actions.UserActor).User.UserId
            } else {
                targetMobInstanceId = target.(*actions.MobActor).Mob.InstanceId
            }
        } else {
            // Self-targeting collapses to NotFound under ExcludeUserId; check explicitly.
            if pId, _ := room.FindByName(args[len(args)-1]); pId == user.UserId {
                user.SendText("You can't buy from yourself.")
                return true, nil
            }
            user.SendText("Visit a merchant to purchase objects or services.")
            return true, nil
        }

        itemname = strings.Join(args[0:len(args)-2], ` `)
    }
}
```

**Note:** buy.go doesn't actually need an Actor — it just needs the IDs. The migration provides nil-safety and shared resolution; the IDs are extracted from the resolved Actor and used as filter values downstream (lines 79+). Behavior preserved.

#### party.go (player-only target)

- [ ] **Step 5: Migrate `usercommands/party.go` (party invite)**

`party invite` only accepts player targets (mobs can't be in parties).

BEFORE (lines 167-186):
```go
invitePlayerId, mobInstId := room.FindByName(rest)

if invitePlayerId == 0 && mobInstId == 0 {
    user.SendText(fmt.Sprintf(`%s not found.`, rest))
    return true, nil
}

if invitedParty := parties.Get(invitePlayerId); invitedParty != nil {
    user.SendText(`That player is already in a party.`)
    return true, nil
}

invitedUser := users.GetByUserId(invitePlayerId)

if invitedUser != nil && currentParty.InvitePlayer(invitePlayerId) {
    user.SendText(...)
    invitedUser.SendText(...)
} else {
    user.SendText(`Something went wrong.`)
}
```

AFTER:
```go
target, err := actions.ResolveTargetActor(room, rest)
if err != nil {
    user.SendText(fmt.Sprintf(`%s not found.`, rest))
    return true, nil
}
if !target.IsPlayer() {
    user.SendText(`You can only invite players to your party.`)
    return true, nil
}

invitedUser := target.(*actions.UserActor).User
invitePlayerId := invitedUser.UserId

if invitedParty := parties.Get(invitePlayerId); invitedParty != nil {
    user.SendText(`That player is already in a party.`)
    return true, nil
}

if currentParty.InvitePlayer(invitePlayerId) {
    user.SendText(fmt.Sprintf(`You invited <ansi fg="username">%s</ansi> to your party.`, invitedUser.Character.Name))
    invitedUser.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> invited you to their party. Type <ansi fg="command">party accept</ansi> or <ansi fg="command">party decline</ansi> to respond.`, user.Character.Name))
} else {
    user.SendText(`Something went wrong.`)
}
```

**Behavior nuance:** pre-migration code silently fell through on mob-only matches (`invitePlayerId == 0 && mobInstId > 0` would still trigger the "%s not found" message because of the `&&`). Wait — re-read the original: `if invitePlayerId == 0 && mobInstId == 0` — so if a mob was matched and no player, this DID NOT fire the not-found message; it would proceed to `parties.Get(0)` which returns nil, then `users.GetByUserId(0)` which returns nil, then `invitedUser != nil` check fails → "Something went wrong." message.

So the pre-migration behavior was: mob-only match → "Something went wrong." (silent failure). The migrated code says "You can only invite players to your party." (explicit). **Behavior change: better error message.** Per spec ("User-only commands ... symmetric: error if target is a mob"), this is the intended pattern. Note this in the commit message.

#### mobcommands/give.go

- [ ] **Step 6: Migrate `mobcommands/give.go`**

Identical shape to usercommands/give.go but mob-side (mob is the giver). Two branches: player target / mob target. The mob branch is shorter than the user-side give.go because there's no quest-engine intercept on mob-to-mob give.

BEFORE (line 56):
```go
playerId, mobId := room.FindByName(giveWho)

if playerId > 0 {
    // ~30 lines of player-target logic
    return true, nil
}

if mobId > 0 {
    // ~30 lines of mob-target logic
    return true, nil
}

return true, nil
```

AFTER:
```go
target, err := actions.ResolveTargetActor(room, giveWho)
if err != nil {
    return true, nil
}

if target.IsPlayer() {
    targetUser := target.(*actions.UserActor).User
    mob.Character.CancelBuffsWithFlag(buffs.Hidden)
    // ~30 lines of player-target logic with `targetUser` already resolved
    return true, nil
}

m := target.(*actions.MobActor).Mob
mob.Character.CancelBuffsWithFlag(buffs.Hidden)
// ~30 lines of mob-target logic with `m` already resolved (drop GetInstance + nil-guard)
return true, nil
```

**Preserve the inline `mobActor := &actions.MobActor{Mob: mob, Room: room}` constructions** for `actions.GiveItemToChar` / `GiveGoldToChar` calls — those are the giver's actor (the mob calling give), distinct from the receiver. Out of scope per Scope Policy.

#### mobcommands/sayto.go

- [ ] **Step 7: Migrate `mobcommands/sayto.go` (3 sites)**

All 3 functions (`SayTo`, `SayToOnly`, `ReplyTo`) have the same pattern. Each migrates independently.

**Site 1 — SayTo (line 27):**

BEFORE:
```go
playerId, mobInstanceId := room.FindByName(args[0])
if playerId > 0 {
    toUser := users.GetByUserId(playerId)
    // ... player-side logic ...
} else if mobInstanceId > 0 {
    toMob := mobs.GetInstance(mobInstanceId)
    // ... mob-side logic ...
}
```

AFTER:
```go
target, err := actions.ResolveTargetActor(room, args[0])
if err != nil {
    return true, nil
}
if target.IsPlayer() {
    toUser := target.(*actions.UserActor).User
    // ... player-side logic ... (unchanged)
} else {
    toMob := target.(*actions.MobActor).Mob
    // ... mob-side logic ... (unchanged)
}
```

**Site 2 — SayToOnly (line 92):** player-only path. Original ignores mobInstanceId (`_`).

BEFORE:
```go
playerId, _ := room.FindByName(args[0])
if playerId > 0 {
    toUser := users.GetByUserId(playerId)
    // ... player-side logic only ...
}
```

AFTER:
```go
target, err := actions.ResolveTargetActor(room, args[0])
if err != nil || !target.IsPlayer() {
    return true, nil
}
toUser := target.(*actions.UserActor).User
// ... player-side logic only ... (unchanged)
```

**Site 3 — ReplyTo (line 130):** identical to Site 1. Apply the same migration.

#### mobcommands/show.go

- [ ] **Step 8: Migrate `mobcommands/show.go`**

Identical shape to usercommands/show.go but no inline `user.SendText` calls (mob is the shower). Apply the same migration pattern as Step 4 above (usercommands/show.go).

#### mobcommands/befriend.go (player-only target)

- [ ] **Step 9: Migrate `mobcommands/befriend.go`**

Befriend only targets players (the mob is becoming charmed by a player).

BEFORE (lines 29-55):
```go
playerId, _ := room.FindByName(rest)

if playerId > 0 {
    // anti-recursion: strip mob's existing companions
    for _, subId := range mob.Character.GetCharmIds() {
        // ...
    }
    mob.Character.CharmedMobs = nil

    mob.Character.Charm(playerId, characters.CharmPermanent, characters.CharmExpiredRevert)

    if charmedUser := users.GetByUserId(playerId); charmedUser != nil {
        charmedUser.Character.TrackCharmed(mob.InstanceId, true)
    }

    sendRoomText(room,
        fmt.Sprintf(`<ansi fg="mobname">%s</ansi> looks very friendly.`, mob.Character.Name))
}
```

AFTER:
```go
target, err := actions.ResolveTargetActor(room, rest)
if err != nil || !target.IsPlayer() {
    return true, nil
}
charmedUser := target.(*actions.UserActor).User
playerId := charmedUser.UserId

// anti-recursion: strip mob's existing companions
for _, subId := range mob.Character.GetCharmIds() {
    if subMob := mobs.GetInstance(subId); subMob != nil {
        subMob.Character.RemoveCharm()
        if subRoom := rooms.LoadRoom(subMob.Character.RoomId); subRoom != nil {
            subRoom.RemoveMob(subId)
        }
        mobs.DestroyInstance(subId)
    }
}
mob.Character.CharmedMobs = nil

mob.Character.Charm(playerId, characters.CharmPermanent, characters.CharmExpiredRevert)
charmedUser.Character.TrackCharmed(mob.InstanceId, true)

sendRoomText(room,
    fmt.Sprintf(`<ansi fg="mobname">%s</ansi> looks very friendly.`, mob.Character.Name))
```

**Behavior nuance:** pre-migration code had `users.GetByUserId(playerId)` AFTER the Charm call, with a nil-guard. Helper closes the nil-deref class — `charmedUser` is guaranteed non-nil. Mob-target case (where the user typed `befriend skeleton`) is silently rejected (consistent with the pre-migration ignore-mobInstanceId behavior).

### Verify

- [ ] **Step 10: Build + vet + tests**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./... -count=1
```

Expected: matches baseline. **Especially: `give_test.go` MUST still pass** — its assertions on consume-branch + btree handoff are unchanged.

- [ ] **Step 11: Manual sanity (boot a server, test give to mob with quest)**

Not strictly required, but if anything looks off in the diff, smoke-test the give command end-to-end on a local server.

### Commit

- [ ] **Step 12: Stage and commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add \
  internal/usercommands/ask.go \
  internal/usercommands/talk.go \
  internal/usercommands/give.go \
  internal/usercommands/buy.go \
  internal/usercommands/party.go \
  internal/mobcommands/give.go \
  internal/mobcommands/sayto.go \
  internal/mobcommands/show.go \
  internal/mobcommands/befriend.go
git commit -m "$(cat <<'EOF'
refactor(commands): migrate interaction commands to ResolveTargetActor

Stage 4 of target-resolution refactor. Migrates 9 sites that
interact with another entity by name. Each command keeps its
structural divergence (give has quest-engine + btree per-target-type;
ask is mob-only; party invite + befriend are player-only) but now
shares the upfront resolution + nil-safety.

Migrated:
  - usercommands/ask (mob-only; explicit error on player target)
  - usercommands/talk (mob-only)
  - usercommands/give (3 branches: player, mob with quest engine
    intercept + btree + GiveItemToChar; pet path unchanged because
    it uses room.FindByPetName, a separate lookup mechanism)
  - usercommands/buy (extracts target IDs as merchant filter values)
  - usercommands/party invite (player-only; explicit error on mob)
  - mobcommands/give (mob-as-giver, 2 branches symmetrical to above)
  - mobcommands/sayto (3 sites in same file: SayTo, SayToOnly, ReplyTo)
  - mobcommands/show (mob-as-shower, 2 branches)
  - mobcommands/befriend (player-only)

Two cases improve user-facing error wording:
  - ask <player>: was a silent fall-through to "ask who what?";
    now "You can't ask another player."
  - party invite <mob>: was a generic "Something went wrong.";
    now "You can only invite players to your party."
Both per spec ("Mob-only commands error explicitly on player
target; user-only symmetric"). Otherwise behavior is preserved.

The give.go quest-engine intercept order (quest engine →
ConsumeItem early-return → btree → GiveItemToChar) is preserved;
TestGive_QuestEngineInterceptsBeforeBtreePlayerGive continues to
pass.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
git status
```

### Spec-Compliance Checklist (Task 4)

- [ ] All 9 sites use `actions.ResolveTargetActor`.
- [ ] `ask` and `talk` reject player targets at the leaf.
- [ ] `party invite` and `befriend` reject mob targets at the leaf.
- [ ] `give` (both user + mob versions) preserves quest-engine intercept order in the mob branch.
- [ ] `buy` resolution still extracts IDs into `targetUserId` / `targetMobInstanceId` for downstream filtering.
- [ ] Pet path in `give.go` (FindByPetName) unchanged.
- [ ] Inline `&actions.UserActor{User: user, Room: room}` constructions for `actions.Execute*` / `actions.Give*` calls left alone (out of scope).
- [ ] `give_test.go` passes unchanged.

### Code-Quality Checklist (Task 4)

- [ ] `go build ./...` clean.
- [ ] `go vet ./...` clean.
- [ ] Full `go test ./...` matches baseline.
- [ ] Unused imports removed.

### Rollback Plan (Task 4)

```bash
git revert HEAD
```

---

## Task 5: `refactor(commands)`: migrate admin/meta commands

Migrate 9 admin/meta sites. Several are player-only commands where the helper resolves uniformly but the leaf branch rejects mob targets.

**Files:**
- Modify: `internal/usercommands/admin.buff.go` (1 site, lines ~74-138)
- Modify: `internal/usercommands/admin.zap.go` (1 site, lines ~22-57)
- Modify: `internal/usercommands/admin.paz.go` (1 site, lines ~22-57)
- Modify: `internal/usercommands/admin.ai.go` (1 site, lines ~28-43) — has both in-room AND global-name fallback
- Modify: `internal/usercommands/admin.command.go` (1 site, lines ~41-65)
- Modify: `internal/usercommands/admin.deafen.go` (2 sites, lines ~25 and ~52) — player-only
- Modify: `internal/usercommands/admin.mute.go` (2 sites, lines ~25 and ~52) — player-only
- Modify: `internal/usercommands/admin.skillset.go` (1 site, lines ~44-48) — player-only
- Modify: `internal/usercommands/report.go` (1 site, lines ~58-71) — player-only

**NOT migrated:** `admin.locate.go` uses `users.GetByCharacterName(rest)` (global, not in-room) — out of scope.

**Estimated commit size:** ~80-100 lines net delta. Medium-small commit.

**Complexity:** Low. Each site is mechanical. Player-only commands use the helper but reject mob targets at the leaf.

### Implementation

#### admin.zap.go and admin.paz.go (symmetric)

- [ ] **Step 1: Migrate `admin.zap.go`**

BEFORE (lines 22-57):
```go
if rest != `` {
    playerId, mobId := room.FindByName(rest)

    if mobId > 0 {
        mob := mobs.GetInstance(mobId)
        if mob == nil {
            user.SendText("Zap Mob not found.")
            return true, nil
        }
        // ... mob path: drain Health/Conviction to 1, broadcast ...
        return true, nil
    }

    if playerId > 0 {
        if u := users.GetByUserId(playerId); u != nil {
            // ... user path: drain Health/Conviction to 1, broadcast, vitals event ...
            return true, nil
        }
    }
}
// ... no-arg path: zap current Aggro target ...
```

AFTER:
```go
if rest != `` {
    target, err := actions.ResolveTargetActor(room, rest)
    if err == nil {
        if !target.IsPlayer() {
            mob := target.(*actions.MobActor).Mob
            user.SendText(fmt.Sprintf(`You zap <ansi fg="mobname">%s</ansi> with a %s!`, mob.Character.Name, boltOfLightning))
            room.SendTextVisual(fmt.Sprintf(`<ansi fg="username">%s</ansi> zaps <ansi fg="mobname">%s</ansi> with a %s!`, user.Character.Name, mob.Character.Name, boltOfLightning), user.UserId)
            mob.Character.Health = 1
            mob.Character.Conviction = 1
            return true, nil
        }
        u := target.(*actions.UserActor).User
        user.SendText(fmt.Sprintf(`You zap <ansi fg="username">%s</ansi> with a %s!`, u.Character.Name, boltOfLightning))
        room.SendTextVisual(fmt.Sprintf(`<ansi fg="username">%s</ansi> zaps <ansi fg="username">%s</ansi> with a %s!`, user.Character.Name, u.Character.Name, boltOfLightning), user.UserId, u.UserId)
        u.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> zaps you with a %s!`, user.Character.Name, boltOfLightning))
        u.Character.Health = 1
        u.Character.Conviction = 1
        events.AddToQueue(events.CharacterVitalsChanged{UserId: u.UserId})
        return true, nil
    }
    // err != nil → fall through to no-arg path (unchanged)
}
// ... no-arg path: zap current Aggro target (unchanged) ...
```

**Behavior nuance:** pre-migration code on "mob found but instance nil" sent "Zap Mob not found." and returned. Post-migration code on `ErrTargetVanished` falls through to the no-arg path. To preserve exact behavior, special-case ErrTargetVanished:
```go
if err == actions.ErrTargetVanished {
    user.SendText("Zap target not found.")
    return true, nil
}
```
Add this branch to preserve the targeted error message. (Generic "Zap target not found." matches both Mob and User vanished; original wording was Mob-specific but the symmetric User path silently fell through.)

- [ ] **Step 2: Migrate `admin.paz.go`** — apply identical pattern (substitute "illuminate"/"beam of light"/"Paz" wording).

#### admin.buff.go

- [ ] **Step 3: Migrate `admin.buff.go`**

The buff command stores both `targetUserId` and `targetMobInstanceId` for downstream branching at lines 108-138. Migration extracts the IDs from the resolved Actor.

BEFORE (lines 74-138):
```go
room := rooms.LoadRoom(user.Character.RoomId)
if room == nil {
    return false, fmt.Errorf(`room %d not found`, user.Character.RoomId)
}

targetUserId, targetMobInstanceId = room.FindByName(args[0])

buffId, _ = strconv.Atoi(args[1])
// ... buff search by name fallback ...

// later:
if targetUserId > 0 {
    if targetUser := users.GetByUserId(targetUserId); targetUser != nil {
        // apply buff to user
    }
}
if targetMobInstanceId > 0 {
    if targetMob := mobs.GetInstance(targetMobInstanceId); targetMob != nil {
        // apply buff to mob
    }
}
```

AFTER (lines 74-138 — keep the same downstream branching, but populate the IDs from the Actor):
```go
room := rooms.LoadRoom(user.Character.RoomId)
if room == nil {
    return false, fmt.Errorf(`room %d not found`, user.Character.RoomId)
}

target, err := actions.ResolveTargetActor(room, args[0])
if err == nil {
    if target.IsPlayer() {
        targetUserId = target.(*actions.UserActor).User.UserId
    } else {
        targetMobInstanceId = target.(*actions.MobActor).Mob.InstanceId
    }
}
// (on err, both IDs stay 0; downstream `if targetUserId > 0` branches won't fire)

buffId, _ = strconv.Atoi(args[1])
// ... rest unchanged ...
```

**The downstream `users.GetByUserId(targetUserId)` and `mobs.GetInstance(targetMobInstanceId)` calls (lines 110, 126) are downstream-by-known-ID lookups** — could optionally be replaced with the resolved Actor pointers, but that requires restructuring the downstream branching. Keep them as-is for minimal diff (helper doesn't fix downstream lookup-by-ID; that's out of scope per spec).

**Optimization:** if you want to use the Actor directly downstream, bind it to a var and dispatch off it. The trade-off is more diff for marginal gain. Recommend the minimal-diff version above.

#### admin.ai.go

- [ ] **Step 4: Migrate `admin.ai.go`**

AI flag has BOTH in-room name lookup AND global-name fallback. Helper handles only the in-room half.

BEFORE (lines 27-37):
```go
targetUserId, _ := room.FindByName(rest)
var targetUser *users.UserRecord

if targetUserId > 0 {
    targetUser = users.GetByUserId(targetUserId)
}

// If not found in room, try by character name globally
if targetUser == nil {
    targetUser = users.GetByCharacterName(rest)
}
```

AFTER:
```go
target, err := actions.ResolveTargetActor(room, rest)
var targetUser *users.UserRecord
if err == nil && target.IsPlayer() {
    targetUser = target.(*actions.UserActor).User
}

// If not found in room, try by character name globally (out-of-room fallback)
if targetUser == nil {
    targetUser = users.GetByCharacterName(rest)
}
```

**Behavior preserved:** mob matches in-room are now ignored explicitly (helper resolves them; we just don't bind them to `targetUser`). Pre-migration code ignored mobInstanceId (`_`). Same outcome.

#### admin.command.go

- [ ] **Step 5: Migrate `admin.command.go`**

The command admin command sends raw events to a target by ID. Migration extracts the IDs from the Actor.

BEFORE (line 41):
```go
playerId, mobId := room.FindByName(searchName)

readyTurn := util.GetTurnCount()
for _, oneCmd := range strings.Split(cmd, `;`) {
    if mobId > 0 {
        events.AddToQueue(events.Input{MobInstanceId: mobId, InputText: oneCmd, ReadyTurn: readyTurn})
    } else if playerId > 0 {
        events.AddToQueue(events.Input{UserId: playerId, InputText: oneCmd, ReadyTurn: readyTurn})
    }
    readyTurn++
}
```

AFTER:
```go
target, err := actions.ResolveTargetActor(room, searchName)
if err != nil {
    return true, nil
}

playerId := target.GetUserId()
mobId := target.GetMobInstanceId()

readyTurn := util.GetTurnCount()
for _, oneCmd := range strings.Split(cmd, `;`) {
    if mobId > 0 {
        events.AddToQueue(events.Input{MobInstanceId: mobId, InputText: oneCmd, ReadyTurn: readyTurn})
    } else if playerId > 0 {
        events.AddToQueue(events.Input{UserId: playerId, InputText: oneCmd, ReadyTurn: readyTurn})
    }
    readyTurn++
}
```

**Behavior nuance:** pre-migration code silently no-oped on no-target (both IDs zero, both branches false). Migrated code returns true on err. Same observable outcome.

#### admin.deafen.go (player-only, 2 sites)

- [ ] **Step 6: Migrate `admin.deafen.go` — both `Deafen` and `UnDeafen`**

BEFORE (lines 25-41 in `Deafen`):
```go
targetUserId, _ := room.FindByName(rest)

if targetUserId > 0 {
    if u := users.GetByUserId(targetUserId); u != nil {
        u.Deafened = true
        user.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> (<ansi fg="username">%s</ansi>) has been <ansi fg="alert-5">DEAFENED</ansi>`, u.Username, u.Character.Name))
        return true, nil
    }
}

user.SendText("Could not find user.")
return true, nil
```

AFTER:
```go
target, err := actions.ResolveTargetActor(room, rest)
if err != nil || !target.IsPlayer() {
    user.SendText("Could not find user.")
    return true, nil
}

u := target.(*actions.UserActor).User
u.Deafened = true
user.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> (<ansi fg="username">%s</ansi>) has been <ansi fg="alert-5">DEAFENED</ansi>`, u.Username, u.Character.Name))
return true, nil
```

Apply the symmetric migration to `UnDeafen` (lines 52-65).

**Behavior nuance:** pre-migration code ignored mob matches (mobInstanceId discarded with `_`); on mob-only match the targetUserId was 0 → "Could not find user." Migrated code uses `!target.IsPlayer()` for the same outcome (mob match → "Could not find user.").

#### admin.mute.go (player-only, 2 sites)

- [ ] **Step 7: Migrate `admin.mute.go` — both `Mute` and `UnMute`**

Apply the same pattern as Step 6, substituting "MUTED" / "UNMUTED" in the wording.

#### admin.skillset.go

- [ ] **Step 8: Migrate `admin.skillset.go`**

Skillset can target self (default) or a player by name. Must reject mob targets.

BEFORE (lines 42-48):
```go
var targetUser *users.UserRecord = user

foundUser, _ := room.FindByName(args[0])
if foundUser > 0 {
    targetUser = users.GetByUserId(foundUser)
    args = args[1:]
}
```

AFTER:
```go
var targetUser *users.UserRecord = user

target, err := actions.ResolveTargetActor(room, args[0])
if err == nil && target.IsPlayer() {
    targetUser = target.(*actions.UserActor).User
    args = args[1:]
}
```

**Behavior nuance:** pre-migration code ignored mobInstanceId (`_`). Mob matches in-room are silently ignored; the command treats them as "no target found, target self." Post-migration: same outcome (`!target.IsPlayer()` keeps `targetUser = user`). What if the user typed a name that matches a mob and intended to target a non-existent player? Pre-migration: `foundUser=0`, target=self, args unchanged → would treat the first arg as a skill name, fail, error. Post-migration: identical.

If the helper's error path returns nil-target on a stale-mob-id case, the behavior is the same (no-op assignment).

**Note:** the original ignores `users.GetByUserId(foundUser)` returning nil (would crash on nil-deref later). Helper closes this latent crash.

#### report.go (player-only, 1 site)

- [ ] **Step 9: Migrate `report.go`**

Report has self-targeting + player-only "rep <player>" mode.

BEFORE (lines 56-71):
```go
if rest != "" {
    targetUserId, _ := room.FindByName(rest)
    if targetUserId == 0 {
        user.SendText(fmt.Sprintf(`You don't see "%s" here.`, rest))
        return true, nil
    }
    if targetUserId == user.UserId {
        user.SendText(fmt.Sprintf(`You check yourself: %s`, barText))
        return true, nil
    }
    targetUser := users.GetByUserId(targetUserId)
    if targetUser == nil {
        user.SendText("They are no longer here.")
        return true, nil
    }
    // send whisper messages
}
```

AFTER:
```go
if rest != "" {
    target, err := actions.ResolveTargetActor(room, rest)
    if err == actions.ErrTargetVanished {
        user.SendText("They are no longer here.")
        return true, nil
    }
    if err != nil || !target.IsPlayer() {
        user.SendText(fmt.Sprintf(`You don't see "%s" here.`, rest))
        return true, nil
    }

    targetUser := target.(*actions.UserActor).User
    if targetUser.UserId == user.UserId {
        user.SendText(fmt.Sprintf(`You check yourself: %s`, barText))
        return true, nil
    }

    targetUser.SendText(fmt.Sprintf(`<ansi fg="whisper"><ansi fg="username">%s</ansi> reports to you: %s</ansi>`, c.Name, barText))
    user.SendText(fmt.Sprintf(`<ansi fg="whisper">You report to <ansi fg="username">%s</ansi>: %s</ansi>`, targetUser.Character.Name, barText))
    return true, nil
}
```

**Behavior nuance:** pre-migration code distinguished "not found in room" ("You don't see X here.") from "ID resolved but vanished" ("They are no longer here."). Migrated code preserves both via the sentinel-error split. Mob-only match → "You don't see X here." (preserved; was implicit due to `targetUserId, _ := ...`).

### Verify

- [ ] **Step 10: Build + vet + tests**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./... -count=1
```

Expected: matches baseline.

### Commit

- [ ] **Step 11: Stage and commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add \
  internal/usercommands/admin.buff.go \
  internal/usercommands/admin.zap.go \
  internal/usercommands/admin.paz.go \
  internal/usercommands/admin.ai.go \
  internal/usercommands/admin.command.go \
  internal/usercommands/admin.deafen.go \
  internal/usercommands/admin.mute.go \
  internal/usercommands/admin.skillset.go \
  internal/usercommands/report.go
git commit -m "$(cat <<'EOF'
refactor(commands): migrate admin/meta commands to ResolveTargetActor

Stage 5 of target-resolution refactor. Migrates 11 sites (some
files have multiple) across 9 admin/meta files:

  - admin.zap, admin.paz (symmetric: damage/heal a target by name)
  - admin.buff (extract IDs into downstream filter)
  - admin.ai (in-room first, then GetByCharacterName fallback)
  - admin.command (forward an event to a target by ID)
  - admin.deafen, admin.unDeafen, admin.mute, admin.unMute
    (player-only; mob-target now silently rejected with the same
    "Could not find user." message)
  - admin.skillset (default self, optional in-room player target)
  - report (player-only "rep <target>" whisper mode)

admin.locate is NOT migrated — it uses users.GetByCharacterName
for global lookup (not in-room name resolution); out of scope per
spec.

Behavior preserved on every site. The "ID resolved but pointer
vanished" race is now distinguishable from "no match" via
ErrTargetVanished — used by report.go to preserve the original
"They are no longer here." vs "You don't see X here." wording
distinction. Other commands collapse both errors to a single
not-found message, matching their pre-migration behavior.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
git status
```

### Spec-Compliance Checklist (Task 5)

- [ ] All 11 in-room name-resolution sites use `actions.ResolveTargetActor`.
- [ ] `admin.locate` left alone (uses GetByCharacterName).
- [ ] Player-only commands (deafen, mute, skillset, report) reject mob targets at the leaf.
- [ ] `admin.buff` still extracts IDs into `targetUserId` / `targetMobInstanceId` for downstream branching.
- [ ] `admin.ai` still has the global GetByCharacterName fallback.
- [ ] Sentinel-error distinction (ErrTargetVanished vs ErrTargetNotFound) used where the original code differentiated wording (report.go).

### Code-Quality Checklist (Task 5)

- [ ] `go build ./...` clean.
- [ ] `go vet ./...` clean.
- [ ] Full `go test ./...` matches baseline.
- [ ] Unused imports removed.

### Rollback Plan (Task 5)

```bash
git revert HEAD
```

---

## Task 6: `refactor(commands)`: migrate skullduggery + target + mob skills

Final migration commit. 6 sites — 3 skullduggery commands plus the `target` command plus 2 mob commands. Several have idiosyncratic logic that the helper consolidates the resolution-prefix of.

**Files:**
- Modify: `internal/usercommands/skill.skullduggery.steal.go` (1 site, lines ~79-87)
- Modify: `internal/usercommands/skill.skullduggery.plant.go` (1 site, lines ~97-106)
- Modify: `internal/usercommands/skill.skullduggery.shadow.go` (1 site, lines ~79-116)
- Modify: `internal/usercommands/target.go` (1 site, lines ~36-50; complex downstream)
- Modify: `internal/mobcommands/aid.go` (1 site, lines ~32-44)
- Modify: `internal/mobcommands/givequest.go` (1 site, lines ~33-40)

**NOT migrated (per Scope Policy):**
- `skill.cast.go` — uses `actions.InitiateCast` (its own resolver internally; out of scope)
- All `mutation_*.go` files — use `Aggro.MobInstanceId`/`Aggro.UserId`, NOT name resolution
- `mobcommands/lookfortrouble.go` — iterates GetPlayers/GetMobs ID lists, no FindByName

**Estimated commit size:** ~70 lines net delta. Small-medium commit. `target.go` has the biggest internal restructure but most of its downstream `mobs.GetInstance(currentTargetMobId)` (lines 104, 109, 123, 127, 149, 156) are downstream-by-known-ID lookups (out of scope).

**Complexity:** Medium. `skill.skullduggery.steal` and `plant` have container-fallback paths that must stay intact. `shadow` has self-exclusion. `target` has the most complex downstream code; only the initial resolution migrates.

### Implementation

#### skill.skullduggery.steal.go (mob-or-container target)

- [ ] **Step 1: Migrate `skill.skullduggery.steal.go`**

Steal targets either a mob (steal from mob) or a room container (steal from container). Players are explicitly rejected.

BEFORE (lines 78-97):
```go
// Try to find a mob or player by name
targetPlayerId, targetMobInstanceId := room.FindByName(args[0])

if targetPlayerId > 0 {
    user.SendText("You can't steal from other players.")
    return true, nil
}

if targetMobInstanceId > 0 {
    return stealFromMob(targetMobInstanceId, attackerScore, rank, user, room, cfg)
}

// Try container steal
containerName := room.FindContainerByName(args[0])
if containerName != `` {
    return stealFromContainer(containerName, attackerScore, rank, user, room)
}

user.SendText("Steal from whom?")
return true, nil
```

AFTER:
```go
// Try to find a mob or player by name
target, err := actions.ResolveTargetActor(room, args[0])
if err == nil {
    if target.IsPlayer() {
        user.SendText("You can't steal from other players.")
        return true, nil
    }
    return stealFromMob(target.(*actions.MobActor).Mob.InstanceId, attackerScore, rank, user, room, cfg)
}

// Try container steal
containerName := room.FindContainerByName(args[0])
if containerName != `` {
    return stealFromContainer(containerName, attackerScore, rank, user, room)
}

user.SendText("Steal from whom?")
return true, nil
```

**Note:** `stealFromMob` takes a `mobInstanceId int`, not an `*actions.MobActor`. Migrating its signature to take an Actor is a larger refactor and out of scope. Pull the InstanceId out of the resolved Actor and pass it through. The `stealFromMob` function then does its own `mobs.GetInstance(mobInstanceId)` (line 104), which is now redundant with the helper's nil-safety — but per spec, downstream lookups by known ID are out of scope.

#### skill.skullduggery.plant.go (mob-or-container target)

- [ ] **Step 2: Migrate `skill.skullduggery.plant.go`** — identical pattern to steal:

BEFORE (lines 96-114):
```go
targetPlayerId, targetMobInstanceId := room.FindByName(targetName)

if targetPlayerId > 0 {
    user.SendText("You can't plant items on other players.")
    return true, nil
}

if targetMobInstanceId > 0 {
    return plantOnMob(targetMobInstanceId, plantItem, attackerScore, rank, user, room)
}

containerName := room.FindContainerByName(targetName)
if containerName != `` {
    return plantInContainer(containerName, plantItem, attackerScore, rank, user, room)
}

user.SendText("Plant on whom?")
return true, nil
```

AFTER:
```go
target, err := actions.ResolveTargetActor(room, targetName)
if err == nil {
    if target.IsPlayer() {
        user.SendText("You can't plant items on other players.")
        return true, nil
    }
    return plantOnMob(target.(*actions.MobActor).Mob.InstanceId, plantItem, attackerScore, rank, user, room)
}

containerName := room.FindContainerByName(targetName)
if containerName != `` {
    return plantInContainer(containerName, plantItem, attackerScore, rank, user, room)
}

user.SendText("Plant on whom?")
return true, nil
```

#### skill.skullduggery.shadow.go (player-or-mob target)

- [ ] **Step 3: Migrate `skill.skullduggery.shadow.go`**

Shadow can target either a player or a mob. Has self-exclusion (can't shadow self).

BEFORE (lines 78-116):
```go
targetPlayerId, targetMobId := room.FindByName(strings.ToLower(rest))

if targetPlayerId == 0 && targetMobId == 0 {
    user.SendText("Shadow whom?")
    return true, nil
}

if targetPlayerId > 0 {
    targetUser := users.GetByUserId(targetPlayerId)
    if targetUser == nil {
        user.SendText("They seem to have vanished.")
        return true, nil
    }
    if targetPlayerId == user.UserId {
        user.SendText("You can't shadow yourself.")
        return true, nil
    }
    user.Character.SetMiscData("shadow-target-user", targetPlayerId)
    user.Character.SetMiscData("shadow-target-mob", nil)
    user.SendText(fmt.Sprintf(
        `You begin shadowing <ansi fg="username">%s</ansi>, watching their every move.`,
        targetUser.Character.Name))
} else {
    mob := mobs.GetInstance(targetMobId)
    if mob == nil {
        user.SendText("They seem to have vanished.")
        return true, nil
    }
    user.Character.SetMiscData("shadow-target-user", nil)
    user.Character.SetMiscData("shadow-target-mob", targetMobId)
    user.SendText(fmt.Sprintf(
        `You begin shadowing <ansi fg="mobname">%s</ansi>, moving silently in their wake.`,
        mob.Character.Name))
}
```

AFTER:
```go
target, err := actions.ResolveTargetActor(room, strings.ToLower(rest), actions.ResolveTargetOptions{
    ExcludeUserId: user.UserId,
})
if err == actions.ErrTargetVanished {
    user.SendText("They seem to have vanished.")
    return true, nil
}
if err != nil {
    // ErrTargetNotFound — could be self-exclusion or genuinely not found.
    if pId, _ := room.FindByName(strings.ToLower(rest)); pId == user.UserId {
        user.SendText("You can't shadow yourself.")
        return true, nil
    }
    user.SendText("Shadow whom?")
    return true, nil
}

if target.IsPlayer() {
    targetUser := target.(*actions.UserActor).User
    user.Character.SetMiscData("shadow-target-user", targetUser.UserId)
    user.Character.SetMiscData("shadow-target-mob", nil)
    user.SendText(fmt.Sprintf(
        `You begin shadowing <ansi fg="username">%s</ansi>, watching their every move.`,
        targetUser.Character.Name))
} else {
    mob := target.(*actions.MobActor).Mob
    user.Character.SetMiscData("shadow-target-user", nil)
    user.Character.SetMiscData("shadow-target-mob", mob.InstanceId)
    user.SendText(fmt.Sprintf(
        `You begin shadowing <ansi fg="mobname">%s</ansi>, moving silently in their wake.`,
        mob.Character.Name))
}
```

**Behavior nuance:** the pre-migration code returned "Shadow whom?" on no-match and "They seem to have vanished." on stale-pointer. Both wordings preserved via sentinel-error split.

#### target.go (most complex downstream)

- [ ] **Step 4: Migrate `target.go`**

The `target` command does name resolution at line 36, then has extensive downstream logic that re-fetches the Character objects multiple times (lines 66, 80, 104, 109, 123, 127, 149, 156). Migrate ONLY the initial resolution; downstream lookup-by-ID stays.

BEFORE (lines 36-50):
```go
newTargetPlayerId, newTargetMobInstanceId := room.FindByName(rest)

if newTargetPlayerId == user.UserId {
    user.SendText("You can't target yourself!")
    return true, nil
}

if newTargetPlayerId == 0 && newTargetMobInstanceId == 0 {
    user.SendText(fmt.Sprintf("You don't see '%s' here.", rest))
    return true, nil
}
```

AFTER:
```go
target, err := actions.ResolveTargetActor(room, rest, actions.ResolveTargetOptions{
    ExcludeUserId: user.UserId,
})
if err != nil {
    if pId, _ := room.FindByName(rest); pId == user.UserId {
        user.SendText("You can't target yourself!")
        return true, nil
    }
    user.SendText(fmt.Sprintf("You don't see '%s' here.", rest))
    return true, nil
}

newTargetPlayerId := target.GetUserId()
newTargetMobInstanceId := target.GetMobInstanceId()
```

**Downstream lines 65-99 still reference `newTargetMobInstanceId` and `newTargetPlayerId` for switch logic + companion check + PvP check** — those work unchanged because we just bound the same variables. The downstream `mobs.GetInstance(newTargetMobInstanceId)` at line 66 is now redundant with the helper's nil-check (helper guaranteed it's non-nil at resolution time), but the downstream code already has a nil-guard (`if m == nil { ... }`), so leave it. Out of scope per spec.

**Behavior preserved:** same self-targeting message, same not-found message, same downstream branching.

#### mobcommands/aid.go (player-only, downed-only)

- [ ] **Step 5: Migrate `mobcommands/aid.go`**

`aid` only resolves players who are downed (uses `rooms.FindDowned`). Helper accepts FindFlags via options.

BEFORE (lines 32-43):
```go
aidPlayerId, _ := room.FindByName(rest, rooms.FindDowned)

if aidPlayerId > 0 {
    p := users.GetByUserId(aidPlayerId)
    if p != nil {
        if p.Character.Health > 0 {
            return true, nil
        }
        // ... cast aidskill spell ...
    }
}
```

AFTER:
```go
target, err := actions.ResolveTargetActor(room, rest, actions.ResolveTargetOptions{
    FindFlags: []rooms.FindFlag{rooms.FindDowned},
})
if err != nil || !target.IsPlayer() {
    return true, nil
}

p := target.(*actions.UserActor).User
if p.Character.Health > 0 {
    return true, nil
}
aidPlayerId := p.UserId
// ... rest of cast aidskill spell logic, using aidPlayerId and p ...
```

**Behavior nuance:** pre-migration ignored mobInstanceId (`_`). Mob matches under FindDowned filter (a downed mob) are now explicitly rejected via `!target.IsPlayer()`. Same outcome.

#### mobcommands/givequest.go (player-only)

- [ ] **Step 6: Migrate `mobcommands/givequest.go`**

GiveQuest dispatches a quest to a named player.

BEFORE (lines 32-40):
```go
if targetUser != `` {
    if uid, _ := room.FindByName(targetUser); uid > 0 {
        events.AddToQueue(events.Quest{UserId: uid, QuestToken: questToken})
    }
}
```

AFTER:
```go
if targetUser != `` {
    target, err := actions.ResolveTargetActor(room, targetUser)
    if err == nil && target.IsPlayer() {
        events.AddToQueue(events.Quest{
            UserId:     target.GetUserId(),
            QuestToken: questToken,
        })
    }
}
```

**Behavior preserved:** mob matches silently ignored (pre-migration: `_` discarded mobInstanceId; post-migration: `!target.IsPlayer()` skips the event dispatch).

### Verify

- [ ] **Step 7: Build + vet + tests**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./... -count=1
```

Expected: matches baseline.

### Commit

- [ ] **Step 8: Stage and commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add \
  internal/usercommands/skill.skullduggery.steal.go \
  internal/usercommands/skill.skullduggery.plant.go \
  internal/usercommands/skill.skullduggery.shadow.go \
  internal/usercommands/target.go \
  internal/mobcommands/aid.go \
  internal/mobcommands/givequest.go
git commit -m "$(cat <<'EOF'
refactor(commands): migrate skullduggery + target + mob skills

Stage 6 — final migration commit. 6 remaining sites:

  - skill.skullduggery.steal (mob-or-container; rejects players)
  - skill.skullduggery.plant (mob-or-container; rejects players)
  - skill.skullduggery.shadow (player-or-mob with self-exclusion;
    preserves "vanished" vs "shadow whom?" wording via sentinel-
    error split)
  - target (combat target switch; only the initial resolution
    migrates — downstream lookup-by-known-ID logic untouched per
    spec)
  - mobcommands/aid (player-only with FindDowned filter)
  - mobcommands/givequest (player-only)

skill.cast is NOT migrated — its name-resolution lives inside
actions.InitiateCast, not directly in the command. Mutation files
(blinding-flash, blinding-spit, etc.) are NOT migrated — they
target via Aggro.MobInstanceId / Aggro.UserId, not by name.
Both per spec ("downstream lookups by known ID are out of scope").

Behavior preserved on every site. The shadow command's
"vanished" vs "Shadow whom?" wording distinction is preserved by
inspecting the sentinel error.

This completes the user-facing target-resolution migration.
Helper now owns every name-based player-or-mob resolution in
usercommands and mobcommands.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
git status
```

### Spec-Compliance Checklist (Task 6)

- [ ] All 6 in-scope sites use `actions.ResolveTargetActor`.
- [ ] `skill.cast.go` left alone (out of scope).
- [ ] All `mutation_*.go` files left alone (out of scope).
- [ ] `mobcommands/lookfortrouble.go` left alone (no FindByName).
- [ ] `target.go` only initial resolution migrated; downstream lookup-by-ID logic untouched.
- [ ] Steal/plant reject players at the leaf with original wording.
- [ ] Shadow preserves wording distinction via sentinel errors.
- [ ] Aid passes `FindDowned` flag through options.

### Code-Quality Checklist (Task 6)

- [ ] `go build ./...` clean.
- [ ] `go vet ./...` clean.
- [ ] Full `go test ./...` matches baseline.
- [ ] Unused imports removed.

### Rollback Plan (Task 6)

```bash
git revert HEAD
```

---

## Task 7: Memory updates

Memory-file-only task. **NO git operations.** All files live outside the repo.

**Files:**
- Update directly (NOT via git): `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\MEMORY.md`
- Delete (NOT via git): `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\project_target_resolution_refactor.md`
- Create directly (NOT via git): `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\feedback_target_resolution_uses_actor.md`

**Complexity:** Trivial.

### Steps

- [ ] **Step 1: Append a "Completed" entry to MEMORY.md**

In `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\MEMORY.md`, find or create the `## Completed (2026-04-18)` section (it likely already exists from the combat-quadrant unification merge). Append:

```markdown
- Target resolution refactor — added actions.ResolveTargetActor
  helper + actions.NewUserActor / NewMobActor exported constructors;
  migrated all ~37 user-facing name-resolution sites in
  usercommands/ and mobcommands/ to use the helper; closes the
  latent-nil-crash class for name-based resolution. Actor interface
  unchanged. New feedback memory captures the rule.
  (helper lives in actions package, not rooms, due to import
  cycle with rooms → actions; caller pattern is
  actions.ResolveTargetActor(room, name).)
```

- [ ] **Step 2: Update or remove `project_target_resolution_refactor.md`**

The project file is now done. Two acceptable approaches; pick one:

  **Option A (preferred — delete):** delete the file.
  ```bash
  rm "/c/Users/Calabe Davis/.claude/projects/C--Users-Calabe-Davis-workspace-DOGMud/memory/project_target_resolution_refactor.md"
  ```

  **Option B:** rewrite the file to a 2-line marker:
  ```markdown
  ---
  name: target resolution refactor
  description: Completed 2026-04-18. See spec docs/superpowers/specs/2026-04-18-target-resolution-refactor-design.md and plan docs/superpowers/plans/2026-04-18-target-resolution-refactor.md. Merge commit on development.
  type: completed
  ---
  ```

  **Option C:** if MEMORY.md has a "Resolved" or "Completed Projects" subsection in its Future Work area, move the project line there.

Pick Option A unless the user has expressed a preference for archival breadcrumbs.

- [ ] **Step 3: Update MEMORY.md Future Work section**

If MEMORY.md lists `project_target_resolution_refactor.md` under a Future Work / Open Projects section, remove that entry (or rewrite the line to "completed 2026-04-18 — see Completed section").

- [ ] **Step 4: Create `feedback_target_resolution_uses_actor.md`**

`C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\feedback_target_resolution_uses_actor.md`:

```markdown
---
name: user-facing target resolution goes through ResolveTargetActor
description: Design rule established by the 2026-04-18 target-resolution refactor — every user-facing target-by-name resolution in usercommands/ and mobcommands/ goes through actions.ResolveTargetActor; commands operate on actions.Actor; type-assert at the leaf only when mob-only or user-only behavior is genuinely needed
type: feedback
originSessionId: <FILL_IN_FROM_CURRENT_SESSION>
---
# Feedback: target resolution goes through actions.ResolveTargetActor

**Rule (established 2026-04-18):**

Any user-facing command that resolves a target by name in the current
room goes through `actions.ResolveTargetActor(room, name, opts...)`.
Commands operate on the returned `actions.Actor` uniformly. Type
assertions to `*actions.UserActor` / `*actions.MobActor` happen only
at leaves where mob-only behavior (`mob.IsNonCombatant()`,
`mob.Character.IsCharmed()`, `mob.Command(...)`) or user-only behavior
(`user.PartyId`, `user.EventLog`, `room.CanPvp(self, otherUser)`,
`user.HasRolePermission(...)`) is genuinely required.

**Caller pattern:**

```go
target, err := actions.ResolveTargetActor(room, name)
if err != nil {
    user.SendText("You don't see them here.")  // caller controls wording
    return true, nil
}

// Use target uniformly when behavior is the same on both sides:
target.SendText("...")           // no-op on mob target

// Type-assert only at the leaf where behavior diverges:
if !target.IsPlayer() {
    mob := target.(*actions.MobActor).Mob
    if mob.IsNonCombatant() {
        user.SendText(fmt.Sprintf(`You can't attack <ansi fg="mobname">%s</ansi>.`, mob.Character.Name))
        return true, nil
    }
    user.Character.SetAggro(0, mob.InstanceId, characters.DefaultAttack)
} else {
    p := target.(*actions.UserActor).User
    if pvpErr := room.CanPvp(user, p); pvpErr != nil {
        user.SendText(pvpErr.Error())
        return true, nil
    }
    user.Character.SetAggro(p.UserId, 0, characters.DefaultAttack)
}
```

**Sentinel errors and when to differentiate:**

```go
ErrTargetNotFound       — no entity matched the name (or only matches were excluded)
ErrTargetVanished       — FindByName returned an ID but the registry pointer is nil (race)
ErrTargetSelfExcluded   — reserved for future use; today self-exclusion collapses to ErrTargetNotFound
```

Most callers don't care about the distinction and use one not-found
message ("You don't see them here.", "Could not find user.", etc.). A
few commands DO differentiate — e.g., `report` says "They are no
longer here." for ErrTargetVanished and "You don't see X here." for
ErrTargetNotFound. Match callsite wording to caller intent.

**Self-targeting message preservation pattern:**

The `ExcludeUserId` option excludes self from results, collapsing
self-targeting to `ErrTargetNotFound`. To preserve a custom
self-targeting message ("You can't bash yourself."), re-call
`room.FindByName` on the error path:

```go
target, err := actions.ResolveTargetActor(room, rest, actions.ResolveTargetOptions{
    ExcludeUserId: user.UserId,
})
if err != nil {
    if pId, _ := room.FindByName(rest); pId == user.UserId {
        user.SendText("You can't bash yourself.")
        return true, nil
    }
    user.SendText("You don't see them here.")
    return true, nil
}
```

This is one extra map-lookup per error path — acceptable for the
behavior-preservation gain.

**When NOT to use ResolveTargetActor:**

- Resolution by globally-unique character name (not in-room): use
  `users.GetByCharacterName(name)`. Example: `admin.locate`.
- Pet name resolution: use `room.FindByPetName(name)`. Different lookup
  mechanism (pets map). Example: `give` pet path.
- Wildcard / "* ANYONE" targets: use `actions.FindAttackTarget(...)` which
  handles wildcards explicitly. Example: `attack *`.
- Spell casting target resolution: lives inside `actions.InitiateCast`;
  command code does not name-resolve. Example: `cast`.
- Auto-target from Aggro state: just use `Aggro.MobInstanceId` /
  `Aggro.UserId` directly with `mobs.GetInstance` / `users.GetByUserId`
  (downstream-by-known-ID is out of helper scope). Example: mutations.
- Iterating room population by ID list (FindFightingPlayer, FindMerchant,
  FindCharmed, etc.): use `room.GetMobs(...)` / `room.GetPlayers(...)` and
  loop. Example: `attack` auto-target prelude.

**When the leaf type-assertion convention applies:**

Only when:
1. The behavior genuinely differs between user and mob (e.g., CanPvp is
   user-only because mobs have no PvP rules; IsNonCombatant is mob-only
   because users have no NPC flag).
2. The Actor interface does not already cover the operation.

Do NOT type-assert just to call a method that exists on Actor — use the
interface. Do NOT add a method to Actor for one-off divergence; the
interface is frozen at 14 methods (per the combat-quadrant unification
decision; same convention applies here).

**Red flags that the rule is being violated:**

- A command directly calls `room.FindByName` followed by manual
  GetInstance/GetByUserId. (Exception: pet path, wildcard helper,
  spell engine; see "When NOT to use" above.)
- A command type-asserts to `*UserActor` / `*MobActor` without first
  checking `target.IsPlayer()`.
- A `&actions.UserActor{...}` or `&actions.MobActor{...}` struct
  literal is added to non-package code instead of using the
  `NewUserActor` / `NewMobActor` / `NewUserActorInRoom` /
  `NewMobActorInRoom` constructors.

**See also:**

- Spec: `docs/superpowers/specs/2026-04-18-target-resolution-refactor-design.md`
- Plan: `docs/superpowers/plans/2026-04-18-target-resolution-refactor.md`
- Sister convention: `feedback_combat_logic_goes_in_handleCombatRound.md`
- Future work: `project_name_collision_prevention.md` (player-mob name
  collision is a known limitation; helper picks player-precedence as a
  workaround)
- Future work (out of scope from this pass): `mobs.SafeGetInstance(id)
  (*Mob, bool)` to enforce nil-acknowledgement on the ~170 downstream
  lookup-by-known-ID sites.
```

Fill `<FILL_IN_FROM_CURRENT_SESSION>` with the executor's session ID.

- [ ] **Step 5: Wire the new feedback memory into MEMORY.md's Feedback list**

Match the existing entry format (look at how `feedback_combat_logic_goes_in_handleCombatRound` is listed). Likely:

```markdown
- feedback_target_resolution_uses_actor — user-facing target resolution goes through actions.ResolveTargetActor; commands operate on actions.Actor; type-assert only at leaves for mob-only or user-only behavior (2026-04-18)
```

### Verify

- [ ] **Step 6: Confirm memory files are NOT in git**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git status
```

Expected: zero `project_*.md` or `feedback_*.md` files staged or untracked. If any appear, you've created in-repo copies — delete them and edit the real files at the external path.

```bash
ls "/c/Users/Calabe Davis/.claude/projects/C--Users-Calabe-Davis-workspace-DOGMud/memory/" | grep -E "(target|MEMORY)"
```

Expected:
  - `MEMORY.md` (recently modified)
  - `feedback_target_resolution_uses_actor.md` (newly created)
  - `project_target_resolution_refactor.md` deleted (Option A) or rewritten as completed-marker (Option B)

### Spec-Compliance Checklist (Task 7)

- [ ] MEMORY.md has a Completed (2026-04-18) entry for this work.
- [ ] `project_target_resolution_refactor.md` deleted or marked completed.
- [ ] `feedback_target_resolution_uses_actor.md` created with the rule + caller pattern + when-NOT-to-use list + red flags.
- [ ] MEMORY.md Feedback section lists the new file.
- [ ] Zero memory files appear in `git status`.

### Rollback Plan (Task 7)

Memory edits are independent of code commits. If an edit is wrong, edit again. No git revert needed.

---

## Task 8: Final verify + merge

Final sanity sweep + smoke test + merge into `development`. Do NOT push.

### Steps

- [ ] **Step 1: Confirm branch shape**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git log --oneline feature/target-resolution-refactor ^development
```

Expected: 6 commits (newest first):
1. `refactor(commands): migrate skullduggery + target + mob skills`
2. `refactor(commands): migrate admin/meta commands to ResolveTargetActor`
3. `refactor(commands): migrate interaction commands to ResolveTargetActor`
4. `refactor(commands): migrate look-and-info to ResolveTargetActor`
5. `refactor(usercommands): migrate combat targeting to ResolveTargetActor`
6. `feat(actions): add ResolveTargetActor helper + Actor constructors`

If a scope-creep `fix:` or `chore:` precursor was added during execution, expect more. Document in merge message.

- [ ] **Step 2: Verify branch diff shape**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git diff --stat development...feature/target-resolution-refactor
```

Expected files changed (verify exhaustive list):
- `internal/actions/actor_user.go` — modified (constructors added; commit 1)
- `internal/actions/actor_mob.go` — modified (constructors added; commit 1)
- `internal/actions/target_resolution.go` — NEW (commit 1)
- `internal/actions/target_resolution_test.go` — NEW (commit 1)
- `internal/hooks/NewRound_DoCombat.go` — modified (inline-literal migration; commit 1)
- `internal/hooks/NewRound_DoCombat_routing_test.go` — modified (inline-literal migration; commit 1)
- `internal/usercommands/bash.go`, `kick.go`, `grapple.go`, `taunt.go`, `trip.go`, `attack.go` — modified (commit 2)
- `internal/usercommands/look.go`, `consider.go`, `show.go`, `skill.track.go` — modified (commit 3)
- `internal/mobcommands/look.go` — modified (commit 3)
- `internal/usercommands/ask.go`, `talk.go`, `give.go`, `buy.go`, `party.go` — modified (commit 4)
- `internal/mobcommands/give.go`, `sayto.go`, `show.go`, `befriend.go` — modified (commit 4)
- `internal/usercommands/admin.{buff,zap,paz,ai,command,deafen,mute,skillset}.go` — modified (commit 5)
- `internal/usercommands/report.go` — modified (commit 5)
- `internal/usercommands/skill.skullduggery.{steal,plant,shadow}.go` — modified (commit 6)
- `internal/usercommands/target.go` — modified (commit 6)
- `internal/mobcommands/aid.go`, `givequest.go` — modified (commit 6)
- `docs/superpowers/specs/2026-04-18-target-resolution-refactor-design.md` — NEW (commit 1)
- `docs/superpowers/plans/2026-04-18-target-resolution-refactor.md` — NEW (commit 1)

**NO other files.** If you see `.claude/settings.local.json`, `feedback/*.txt`, `Screenshot*.png`, or any memory file, STOP — you accidentally staged noise.

Net line count: probably small positive (~250 lines from the helper + 7 tests; commit 1's inline-literal migrations are net-neutral; commits 2-6 are roughly net-neutral since each migration shrinks the FindByName chain by ~5 lines but adds ~5 lines of type-assert + branch).

- [ ] **Step 3: Final whole-project verification**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./... -count=1
```

Expected: clean, matching the Task 0 baseline.

- [ ] **Step 4: Manual smoke test — one command per category**

Boot a local server. Run the 5 short scenarios below; each should produce the same output it did before this work.

  1. **Combat targeting (Task 2):** `bash <mob>` at a non-combatant first (expect "You can't attack X."), then at a combatant (expect aggro set).
  2. **Look-and-info (Task 3):** `look <player>` at another character — expect description + inventory output. Then `consider <mob>` for a power-ratio prediction.
  3. **Interaction (Task 4):** `ask <quest_npc> about quest` — expect dialogue output. Then `give <item> <npc>` with a quest-engine intercept active — expect quest engine to fire (not the btree handler).
  4. **Admin (Task 5):** `buff <player> <buff_id>` — expect buff application + ack. `deafen <player>` — expect deafen + "DEAFENED" message.
  5. **Skullduggery (Task 6):** `steal from <mob>` — expect roll + outcome. `target <mob>` mid-combat — expect target-switch attempt.

If any scenario reads wrong, STOP — the unit tests missed something.

- [ ] **Step 5: Confirm memory files still live outside the repo**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git diff --stat development...feature/target-resolution-refactor | grep -E "\.claude|memory/project|memory/feedback"
```

Expected: zero matches.

---

## Merge to development (after user review)

Do NOT merge until the user has reviewed the branch. Once approved:

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git checkout development
git merge --no-ff feature/target-resolution-refactor -m "$(cat <<'EOF'
merge: target resolution refactor

Add actions.ResolveTargetActor helper + actions.NewUserActor /
NewMobActor exported constructors; migrate all ~37 user-facing
name-resolution sites in usercommands/ and mobcommands/ to use the
helper. Closes the latent-nil-crash class for name-based target
resolution; commands stop branching on user-vs-mob at the
resolution site (type assertion only at leaves where mob-only or
user-only behavior is genuinely needed).

Commit 1: feat(actions) — helper, sentinel errors, two pairs of
constructors, 7 unit tests, migration of inline UserActor/MobActor
literals in internal/hooks (carryover from combat-quadrant
unification). Note: helper lives in internal/actions, not
internal/rooms, due to import-cycle constraint (rooms → actions
already exists). Caller pattern is actions.ResolveTargetActor(room,
name).

Commit 2: refactor(usercommands) combat targeting — bash, kick,
grapple, taunt, trip; plus a defensive nil-guard at attack.go:27.

Commit 3: refactor(commands) look-and-info — look, consider, show,
skill.track, mobcommands/look. Two latent nil-derefs closed.

Commit 4: refactor(commands) interaction — ask, talk, give, buy,
party invite, mobcommands/give, mobcommands/sayto (3 sites),
mobcommands/show, mobcommands/befriend. Two error messages
improved per spec ("Mob-only commands explicit error on player
target; user-only symmetric"); otherwise behavior preserved. The
give.go quest-engine-before-btree intercept order is preserved;
TestGive_QuestEngineInterceptsBeforeBtreePlayerGive continues to
pass.

Commit 5: refactor(commands) admin/meta — admin.buff, .zap, .paz,
.ai, .command, .deafen (×2), .mute (×2), .skillset, report.
admin.locate left alone (uses GetByCharacterName globally).

Commit 6: refactor(commands) skullduggery + target + mob skills —
steal, plant, shadow, target, mobcommands/aid, mobcommands/givequest.
skill.cast not migrated (its resolution lives in actions.InitiateCast);
mutation_*.go not migrated (target by Aggro state, not by name).

Actor interface at internal/actions/actor.go UNCHANGED (14 methods).
Same convention as combat-quadrant unification: leaf type-assertion
behind IsPlayer() check for mob-only / user-only behavior.

A new feedback memory (feedback_target_resolution_uses_actor.md)
captures the rule + caller pattern + when-NOT-to-use cases.
project_target_resolution_refactor.md is retired.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

If a scope-creep `fix:` or `chore:` precursor was added during execution, mention it in the merge body.

**Do NOT push to origin.** User reviews merge locally and pushes when ready.

---

## Done

After merge, every user-facing target-by-name resolution in `usercommands/` and `mobcommands/` flows through `actions.ResolveTargetActor`. The latent-nil-crash class for name-based resolution is closed. Future commands that need to resolve a target by name follow the established pattern; the feedback memory captures it for future executors.

Out-of-scope follow-ups (tracked, not blocking):
  - `mobs.SafeGetInstance(id) (*Mob, bool)` to enforce nil-acknowledgement on the ~170 downstream lookup-by-known-ID sites.
  - `project_name_collision_prevention.md` — the helper picks player-precedence as a workaround for player-mob name collisions; the real fix (player-creation guard, reserved-word list, actor disambiguation syntax) is deferred.
  - Inline `&actions.UserActor{User: user, Room: room}` constructions still exist in `internal/usercommands/` and `internal/mobcommands/` for `actions.Execute*` calls (bash, kick, grapple, etc.). They could be migrated to `NewUserActorInRoom` for consistency; small chore commit, separate concern.

---

### Critical Files for Implementation

- `C:\Users\Calabe Davis\workspace\DOGMud\internal\actions\target_resolution.go` (NEW in Task 1 — the helper itself; single most important file the entire refactor builds on)
- `C:\Users\Calabe Davis\workspace\DOGMud\internal\actions\actor_user.go` (Task 1 modifies — adds NewUserActor + NewUserActorInRoom; existing field layout reference for Tasks 2-6 type assertions)
- `C:\Users\Calabe Davis\workspace\DOGMud\internal\actions\actor_mob.go` (Task 1 modifies — adds NewMobActor + NewMobActorInRoom; same role as above for mob-side assertions)
- `C:\Users\Calabe Davis\workspace\DOGMud\internal\rooms\rooms.go` (line 1430 — `FindByName` signature reference; the helper wraps this)
- `C:\Users\Calabe Davis\workspace\DOGMud\internal\usercommands\give.go` (most divergent migration site in Task 4 — quest-engine intercept ordering must be preserved; reference for any other multi-branch interaction migration)
```

---

That's the complete plan. It's approximately 1,800 lines, comparable to the combat-quadrant-unification template at 2,058 lines.

**Important clarification on the import-cycle issue:** During the discovery phase I confirmed `internal/actions` already imports `internal/rooms` (in `actor_user.go` and `actor_mob.go`). The spec specified `r.ResolveTargetActor(...)` as a method on `*Room`, but that would create a circular import. The plan flips the helper into the `actions` package as `actions.ResolveTargetActor(room, name, opts...)`. This is a minor caller-ergonomics change (one extra arg) but unavoidable given the existing dependency graph; the plan flags this prominently and updates all caller-pattern examples accordingly.

**Site count reconciliation:** the user's task listed ~45 sites and called out specific files; my discovery confirmed:
- 31 FindByName calls in `usercommands/` (one is the `give_test.go:43` test file, which stays)
- 9 FindByName calls in `mobcommands/`
- = 39 raw FindByName sites; minus give_test = **~38 in-scope sites**, distributed across **~37 distinct command functions** (sayto.go has 3 in one file).

A few items in the user's requested file list were verified out-of-scope during discovery and are documented in the Scope Policy table:
- `combat_shoot.go` — has no FindByName
- `admin.locate` — uses `users.GetByCharacterName` (global), not `room.FindByName`
- All `mutation_*.go` files — target via Aggro state, not by name
- `mobcommands/lookfortrouble.go` — iterates ID lists, no name resolution
- `skill.cast.go` — resolution lives inside `actions.InitiateCast`

The plan is read-to-execute as written; the parent agent or user should write the content above to `C:\Users\Calabe Davis\workspace\DOGMud\docs\superpowers\plans\2026-04-18-target-resolution-refactor.md`.