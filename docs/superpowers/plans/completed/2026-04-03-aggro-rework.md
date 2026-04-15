# Aggro/Targeting Rework — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Centralize aggro state management. Replace ~20 raw `Aggro = nil` with `EndAggro()`. Add `ValidateAggro()`, `RetargetOrEnd()`, and `CompanionAutoTarget()`.

**Architecture:** New aggro helper functions in `internal/hooks/` (avoids import cycles with characters→mobs/users). All scattered stale checks and retarget logic consolidated into single call sites.

**Tech Stack:** Go, existing combat/aggro system

---

## File Structure

| File | Responsibility |
|------|---------------|
| `internal/hooks/aggro_helpers.go` | ValidateAggro, RetargetOrEnd, CompanionAutoTarget |
| `internal/hooks/NewRound_DoCombat.go` | Simplified combat loop with centralized aggro checks |
| `internal/hooks/NewRound_DoCombat_helpers.go` | Kill paths use RetargetOrEnd, raw nil → EndAggro |
| `internal/hooks/NewRound_UserRoundTick.go` | Remove stale aggro safety net (handled by combat loop now) |
| `internal/hooks/NewRound_IdleMobs.go` | Raw nil → EndAggro |
| `internal/mobcommands/break.go` | Raw nil → EndAggro |
| `internal/mobcommands/flee.go` | Raw nil → EndAggro |
| `internal/mobcommands/submit.go` | Raw nil → EndAggro |
| `internal/mobcommands/suicide.go` | Raw nil → EndAggro |
| `internal/usercommands/break.go` | Raw nil → EndAggro |
| `internal/usercommands/go.go` | Raw nil → EndAggro |
| `internal/usercommands/submit.go` | Raw nil → EndAggro |
| `internal/usercommands/suicide.go` | Raw nil → EndAggro |
| `internal/usercommands/mutation_pacifism_aura.go` | Raw nil → EndAggro |
| `internal/hooks/aggro_helpers_test.go` | Tests |

---

### Task 1: Create Aggro Helper Functions

**Files:**
- Create: `internal/hooks/aggro_helpers.go`

- [ ] **Step 1: Create ValidateAggro**

```go
package hooks

// ValidateAggro checks if a character's aggro target still exists
// and is alive. If invalid, calls EndAggro() and returns false.
// Returns true if aggro is valid, false if cleared.
func ValidateAggro(char *characters.Character) bool
```

Logic:
1. If `char.Aggro == nil` → return false
2. If `char.Aggro.MobInstanceId > 0`:
   - `target := mobs.GetInstance(MobInstanceId)`
   - If nil or `target.Character.Health < 1` → `char.EndAggro()`, return false
3. If `char.Aggro.UserId > 0`:
   - `target := users.GetByUserId(UserId)`
   - If nil or `target.Character.Health < 1` → `char.EndAggro()`, return false
4. Return true

- [ ] **Step 2: Create RetargetOrEnd**

```go
// RetargetOrEnd clears current aggro and attempts to find a new
// target in the room. For companions, also considers mobs attacking
// the charm owner. Returns true if a new target was found.
func RetargetOrEnd(char *characters.Character, room *rooms.Room,
    userId int, mobInstanceId int) bool
```

Logic:
1. `char.EndAggro()`
2. Scan `room.GetMobs(rooms.FindFighting)` for mobs with aggro
   pointing at this character (by userId or mobInstanceId)
3. If found: `char.SetAggro(0, mobInstanceId, DefaultAttack)`, return true
4. For companion mobs: also check mobs attacking the charm owner
   - If `char.IsCharmed()`: get owner userId, scan for mobs
     attacking owner
5. Scan `room.GetPlayers(rooms.FindFighting)` for players with
   aggro pointing at this character
6. If found: `char.SetAggro(userId, 0, DefaultAttack)`, return true
7. Return false (out of combat)

- [ ] **Step 3: Create CompanionAutoTarget**

```go
// CompanionAutoTarget checks if an idle companion should enter
// combat to defend its owner. Only fires when the companion has
// AutoAssist=true and Aggro==nil.
func CompanionAutoTarget(mob *mobs.Mob, room *rooms.Room)
```

Logic:
1. If `mob.Character.Aggro != nil` → return (already fighting)
2. If `!mob.Character.IsCharmed()` → return (not a companion)
3. Get owner: `ownerId := mob.Character.GetCharmedUserId()`
4. Get owner user: `owner := users.GetByUserId(ownerId)`
5. Check AutoAssist: find CompanionInfo by instanceId, check flag
6. If owner has aggro: companion attacks owner's target
   - `mob.Command(fmt.Sprintf("attack @%d", owner.Aggro.UserId))`
     or `"attack #%d"` for mob targets
7. Else: scan room for mobs attacking owner
   - Issue attack command on first found

- [ ] **Step 4: Verify build + commit**

```bash
git commit -m "feat: centralized aggro helpers — ValidateAggro, RetargetOrEnd, CompanionAutoTarget"
```

---

### Task 2: Replace Raw `Aggro = nil` with `EndAggro()`

**Files:** ~15 files with raw nil assignments

- [ ] **Step 1: Find and replace all raw nil sites**

Search for `\.Aggro = nil` across the codebase. For each site,
replace with `.EndAggro()`. The sites from the audit:

**`internal/hooks/NewRound_DoCombat.go`:** line 124 (mob no room)
**`internal/hooks/NewRound_DoCombat_helpers.go`:**
- line 467 (player flee)
- line 843 (coup de grace config=0)
- lines 892, 911, 917, 924 (PvP stale checks)
- lines 1036, 1047, 1053, 1060 (PvM stale checks)
- lines 1255, 1261 (MvP null checks)
- lines 1468, 1474, 1481 (MvM null checks)

**`internal/hooks/NewRound_IdleMobs.go`:** line 71

**`internal/mobcommands/break.go`:** line 13
**`internal/mobcommands/flee.go`:** line 15
**`internal/mobcommands/submit.go`:** lines 106-110
**`internal/mobcommands/suicide.go`:** line 128

**`internal/usercommands/break.go`:** line 14
**`internal/usercommands/go.go`:** line 54
**`internal/usercommands/submit.go`:** lines 115-119
**`internal/usercommands/suicide.go`:** lines 179, 205
**`internal/usercommands/mutation_pacifism_aura.go`:** lines 44, 64

Read each file, find the exact line, replace `.Aggro = nil` with
`.EndAggro()`. Some sites assign to a variable (e.g.,
`mob.Character.Aggro = nil`) — replace with
`mob.Character.EndAggro()`.

NOTE: The `handleCharmedMobAssist` two-step pattern
(`Aggro = &Aggro{...}`) is NOT a nil assignment — leave those alone.

- [ ] **Step 2: Verify build**

Run: `go build ./...`

- [ ] **Step 3: Commit**

```bash
git commit -m "refactor: replace all raw Aggro = nil with EndAggro() for consistent grapple cleanup"
```

---

### Task 3: Integrate ValidateAggro into Combat Loop

**Files:**
- Modify: `internal/hooks/NewRound_DoCombat.go`
- Modify: `internal/hooks/NewRound_DoCombat_helpers.go`
- Modify: `internal/hooks/NewRound_UserRoundTick.go`

- [ ] **Step 1: Replace player combat pre-loop check**

In `NewRound_DoCombat.go`, the `handlePlayerCombat` loop currently
has a stale aggro check at lines 64-75. Replace with:

```go
if !ValidateAggro(user.Character) {
    uRoom := rooms.LoadRoom(user.Character.RoomId)
    if uRoom != nil {
        RetargetOrEnd(user.Character, uRoom, user.UserId, 0)
    }
    if user.Character.Aggro == nil {
        continue
    }
}
```

- [ ] **Step 2: Add mob combat pre-loop check**

In the `handleMobCombat` loop, add the same pattern before
dispatching to handlers:

```go
if !ValidateAggro(&mob.Character) {
    mobRoom := rooms.LoadRoom(mob.Character.RoomId)
    if mobRoom != nil {
        RetargetOrEnd(&mob.Character, mobRoom, 0, mob.InstanceId)
    }
    if mob.Character.Aggro == nil {
        continue
    }
}
```

- [ ] **Step 3: Remove scattered stale checks from combat handlers**

In `handlePlayerVsMob`, `handlePlayerVsPlayer`, `handleMobVsMob`:
remove the inline stale-target checks that duplicate what
`ValidateAggro` already did. These are the `GetInstance == nil`,
`RoomId mismatch`, `Health < 1` checks at the top of each handler.

Be careful — some of these checks also handle room-mismatch
(target left the room). `ValidateAggro` doesn't check room
mismatch. Either add room-mismatch to `ValidateAggro` or keep
those specific checks.

- [ ] **Step 4: Remove UserRoundTick stale check**

Remove the stale aggro check we added in `NewRound_UserRoundTick.go`
(lines 125-129). It's no longer needed since the combat loop handles
it and the ValidateAggro call is comprehensive.

- [ ] **Step 5: Verify build + commit**

```bash
git commit -m "refactor: centralized ValidateAggro in combat loop, remove scattered checks"
```

---

### Task 4: Integrate RetargetOrEnd into Kill Paths

**Files:**
- Modify: `internal/hooks/NewRound_DoCombat_helpers.go`

- [ ] **Step 1: Replace kill-path retarget logic**

In each of the 4 combat handlers, find the death block
(`if attackerDead || defenderDead`). Replace the current pattern:

Before:
```go
if defenderDead:
    attacker.EndAggro()
    defender.EndAggro()
    handleAutoRetargetPlayer(user, room)
```

After:
```go
if defenderDead:
    defender.EndAggro()
    RetargetOrEnd(attacker, room, attackerId, attackerMobId)
```

Apply to:
- `handlePlayerVsPlayer` (line ~1008)
- `handlePlayerVsMob` (line ~1238)
- `handleMobVsPlayer` (line ~1454)
- `handleMobVsMob` (line ~1600)

- [ ] **Step 2: Remove handleAutoRetargetPlayer**

After all callers are converted to `RetargetOrEnd`, the old
`handleAutoRetargetPlayer` function can be deleted.

- [ ] **Step 3: Send retarget message**

`RetargetOrEnd` should send a message when it retargets:
- Player: "You turn your attention to [name]!"
- Mob: no message (silent retarget)

Pass the user record or a message callback so the function can
notify the player.

- [ ] **Step 4: Verify build + commit**

```bash
git commit -m "refactor: RetargetOrEnd replaces handleAutoRetargetPlayer in all kill paths"
```

---

### Task 5: Integrate CompanionAutoTarget

**Files:**
- Modify: `internal/hooks/NewRound_DoCombat.go` (mob combat loop)

- [ ] **Step 1: Add CompanionAutoTarget to mob combat loop**

In `handleMobCombat`, after the ValidateAggro check, add:

```go
// Idle companions with autoassist scan for threats to owner
if mob.Character.Aggro == nil && mob.Character.IsCharmed() {
    CompanionAutoTarget(mob, mobRoom)
    if mob.Character.Aggro == nil {
        continue // Still no target
    }
}
```

This runs after ValidateAggro (which may have cleared stale aggro)
and before combat dispatch. If the companion finds a target via
autoassist, it enters combat this round.

- [ ] **Step 2: Verify build + commit**

```bash
git commit -m "feat: CompanionAutoTarget — idle companions scan for owner threats"
```

---

### Task 6: Tests

**Files:**
- Create: `internal/hooks/aggro_helpers_test.go`

- [ ] **Step 1: Write tests**

Tests needed:
1. `TestValidateAggro_Nil` — nil aggro returns false
2. `TestValidateAggro_ValidMob` — valid mob target returns true
3. `TestValidateAggro_DeadMob` — dead mob target → EndAggro, false
4. `TestValidateAggro_MissingMob` — despawned mob → EndAggro, false
5. `TestRetargetOrEnd_FindsAttacker` — hostile in room → retargets
6. `TestRetargetOrEnd_NoHostiles` — empty room → stays out of combat

Note: these tests may need full mob/user registry seeding. If too
complex, test the logic with mocks or skip with explanation.

- [ ] **Step 2: Run tests + commit**

```bash
git commit -m "test: aggro helper tests"
```

---

### Task 7: Final Verification

- [ ] **Step 1: Full build + tests**

Run: `go build ./...`
Run: `go test ./... -count=1 -timeout 300s`

- [ ] **Step 2: Manual smoke test**

- Fight 3 mobs with a companion → companion kills one → player
  auto-retargets to next mob attacking them
- All mobs die → player cleanly exits combat, no "can't do that"
- Companion idle after kill → picks up new mob attacking owner
- Player in party with companion → party member gets attacked →
  companion assists
- Dismiss companion mid-fight → companion turns hostile, player
  retargets correctly
- Flee from combat → aggro fully cleared, can act normally

- [ ] **Step 3: Commit any fixups**
