# Command Unification — Substage 3: Combat Commands

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract shared combat command logic for attack, bash, kick, trip, grapple, and shoot. Add kick variant parity (stomp/knee) and tailsweep parity for mobs. The combat calculation engine is already unified — this substage unifies the command entry points.

**Architecture:** Shared combat helpers in `internal/actions/combat_helpers.go` eliminate the most duplicated patterns (target resolution, cooldowns, analytics). Per-command shared actions handle the core mechanic. Wrappers handle messaging and actor-specific concerns (party, PvP, charm assist, darkness, skill progression).

**Tech Stack:** Go, testify/assert, existing combat engine

---

## File Structure

| File | Responsibility |
|------|---------------|
| `internal/actions/combat_helpers.go` | ResolveAggroTarget, TryCombatCooldown, RecordAndWait |
| `internal/actions/combat_bash.go` | Shared ExecuteBash |
| `internal/actions/combat_kick.go` | Shared ExecuteKick with variant selection |
| `internal/actions/combat_trip.go` | Shared ExecuteTrip with tailsweep detection |
| `internal/actions/combat_grapple.go` | Shared ExecuteGrapple |
| `internal/actions/combat_shoot.go` | Shared ExecuteShoot |
| `internal/actions/combat_attack.go` | Shared attack target resolution helpers |
| `internal/actions/combat_test.go` | Tests for combat helpers and variant selection |
| `internal/usercommands/bash.go` | Thin wrapper (modify) |
| `internal/usercommands/kick.go` | Thin wrapper (modify) |
| `internal/usercommands/trip.go` | Thin wrapper (modify) |
| `internal/usercommands/grapple.go` | Thin wrapper (modify) |
| `internal/usercommands/shoot.go` | Thin wrapper (modify) |
| `internal/usercommands/attack.go` | Thin wrapper (modify) |
| `internal/mobcommands/bash.go` | Thin wrapper (modify) |
| `internal/mobcommands/kick.go` | Thin wrapper (modify) |
| `internal/mobcommands/trip.go` | Thin wrapper (modify) |
| `internal/mobcommands/grapple.go` | Thin wrapper (modify) |
| `internal/mobcommands/shoot.go` | Thin wrapper (modify) |
| `internal/mobcommands/attack.go` | Thin wrapper (modify) |

---

### Task 1: Shared Combat Helpers

**Files:**
- Create: `internal/actions/combat_helpers.go`

- [ ] **Step 1: Read the existing target resolver pattern**

Read both `internal/usercommands/bash.go` and `internal/mobcommands/bash.go`
to see the exact duplicated target resolution block. It looks like:

```go
// Get the aggro target
if char.Aggro.UserId > 0 {
    u := users.GetByUserId(char.Aggro.UserId)
    if u != nil {
        defender = u.Character
        defenderName = u.Character.Name
        // ...
    }
} else if char.Aggro.MobInstanceId > 0 {
    m := mobs.GetInstance(char.Aggro.MobInstanceId)
    if m != nil {
        defender = &m.Character
        defenderName = m.Character.Name
        // ...
    }
}
```

Also read the cooldown check pattern and the analytics tail.

- [ ] **Step 2: Create combat_helpers.go**

```go
package actions

import (
    "fmt"

    "github.com/GoMudEngine/GoMud/internal/characters"
    "github.com/GoMudEngine/GoMud/internal/combat"
    "github.com/GoMudEngine/GoMud/internal/mobs"
    "github.com/GoMudEngine/GoMud/internal/users"
)

// AggroTarget holds the resolved target from an aggro reference.
type AggroTarget struct {
    Char          *characters.Character
    Name          string
    UserId        int
    MobInstanceId int
    Found         bool
}

// ResolveAggroTarget resolves the current aggro target to a
// character pointer, name, and IDs. Works for both user and
// mob aggressors — the aggro struct is the same.
func ResolveAggroTarget(aggro *characters.Aggro) AggroTarget {
    if aggro == nil {
        return AggroTarget{Found: false}
    }
    if aggro.UserId > 0 {
        u := users.GetByUserId(aggro.UserId)
        if u != nil {
            return AggroTarget{
                Char:   u.Character,
                Name:   u.Character.Name,
                UserId: aggro.UserId,
                Found:  true,
            }
        }
    }
    if aggro.MobInstanceId > 0 {
        m := mobs.GetInstance(aggro.MobInstanceId)
        if m != nil {
            return AggroTarget{
                Char:          &m.Character,
                Name:          m.Character.Name,
                MobInstanceId: aggro.MobInstanceId,
                Found:         true,
            }
        }
    }
    return AggroTarget{Found: false}
}

// TryCombatCooldown checks if the character is on cooldown for
// a special move. Returns true if on cooldown (move blocked).
func TryCombatCooldown(char *characters.Character, cooldownRounds int) bool {
    return !char.Cooldowns.Try("special-move",
        fmt.Sprintf("%d rounds", cooldownRounds))
}

// RecordAndWait records a special move for analytics and sets
// the round wait penalty.
func RecordAndWait(char *characters.Character, moveName string, sourceType int) {
    combat.RecordSpecialMove(moveName, sourceType)
    char.Aggro.RoundsWaiting = 1
}
```

Note: Read `internal/characters/cooldowns.go` to verify the
`Cooldowns.Try` signature. Read `internal/combat/` to verify
`RecordSpecialMove` signature and the source type constants
(`combat.User`, `combat.Mob` or similar).

- [ ] **Step 3: Verify build**

Run: `go build ./...`

- [ ] **Step 4: Commit**

```bash
git add internal/actions/combat_helpers.go
git commit -m "feat: shared combat helpers — target resolution, cooldowns, analytics"
```

---

### Task 2: Shared Bash Action

**Files:**
- Create: `internal/actions/combat_bash.go`
- Modify: `internal/usercommands/bash.go`
- Modify: `internal/mobcommands/bash.go`

Bash is the simplest special move — use it to establish the pattern.

- [ ] **Step 1: Read both bash files fully**

Read `internal/usercommands/bash.go` and `internal/mobcommands/bash.go`
to understand the full flow.

- [ ] **Step 2: Create shared bash action**

```go
package actions

import (
    "github.com/GoMudEngine/GoMud/internal/characters"
    "github.com/GoMudEngine/GoMud/internal/combat"
    "github.com/GoMudEngine/GoMud/internal/configs"
)

// SkillMoveResult contains the outcome of a special move.
type SkillMoveResult struct {
    Target      AggroTarget
    MoveResult  combat.SkillMoveResult
    Executed    bool
    OnCooldown  bool
    NoTarget    bool
    NoShield    bool  // bash-specific
}

// ExecuteBash performs the bash special move.
// Requires: actor is in combat (Aggro != nil), has a shield equipped.
// Returns result for wrapper to format messages.
func ExecuteBash(actor Actor) SkillMoveResult {
    char := actor.GetCharacter()

    if char.Aggro == nil {
        return SkillMoveResult{NoTarget: true}
    }

    // Shield check
    if !char.HasShield() {
        return SkillMoveResult{NoShield: true}
    }

    // Cooldown check
    cfg := configs.GetGamePlayConfig()
    if TryCombatCooldown(char, int(cfg.SpecialMoveCooldown)) {
        return SkillMoveResult{OnCooldown: true}
    }

    // Resolve target
    target := ResolveAggroTarget(char.Aggro)
    if !target.Found {
        return SkillMoveResult{NoTarget: true}
    }

    // Execute the move
    result := combat.ExecuteSkillMove(combat.SkillMoveParams{
        // Fill in the exact params from the existing bash code.
        // Read both bash.go files to see what params are used —
        // they should be identical between user and mob.
    })

    // Record and set round wait
    sourceType := combat.Mob
    if actor.IsPlayer() {
        sourceType = combat.User
    }
    RecordAndWait(char, "bash", sourceType)

    return SkillMoveResult{
        Target:     target,
        MoveResult: result,
        Executed:   true,
    }
}
```

IMPORTANT: Read both existing bash files to get the exact
`combat.SkillMoveParams` struct fields. They should be identical
between user and mob. Copy the exact field values.

- [ ] **Step 3: Update user bash wrapper**

Refactor to:
1. Allow initiating from non-combat (user-specific: parse `rest`
   arg, SetAggro if needed)
2. Call `actions.ExecuteBash(actor)`
3. Handle result: NoShield → "need a shield", OnCooldown → "need
   a moment to recover", NoTarget → "not fighting anyone"
4. Format hit/miss/knockdown messages with user ANSI colors
5. Fire `events.SkillUsed` for progression (user-only)

- [ ] **Step 4: Update mob bash wrapper**

Refactor to:
1. Call `actions.ExecuteBash(actor)`
2. Handle result with mob messaging (darkness-aware, room-only)
3. No skill progression event

- [ ] **Step 5: Verify build**

Run: `go build ./...`

- [ ] **Step 6: Commit**

```bash
git add internal/actions/combat_bash.go \
  internal/usercommands/bash.go internal/mobcommands/bash.go
git commit -m "refactor: shared Bash action with combat helpers"
```

---

### Task 3: Shared Kick Action (with variant parity)

**Files:**
- Create: `internal/actions/combat_kick.go`
- Modify: `internal/usercommands/kick.go`
- Modify: `internal/mobcommands/kick.go`

- [ ] **Step 1: Read both kick files fully**

The user kick.go has ~300 lines with stomp/knee variants and
flavor text arrays. The mob kick.go has ~128 lines with no
variants. Understand the variant selection logic:
- Standing target → regular kick (KickDamagePercent, KickKnockdownChance)
- Prone target → stomp (StompDamagePercent, higher knockdown)
- Grapple + controller → knee (KneeDamagePercent)

- [ ] **Step 2: Create shared kick action**

```go
type KickVariant int
const (
    KickStandard KickVariant = iota
    KickStomp    // target is prone
    KickKnee     // attacker in grapple, has control
)

type KickMoveResult struct {
    SkillMoveResult
    Variant KickVariant
}

func ExecuteKick(actor Actor) KickMoveResult
```

ExecuteKick should:
1. Standard aggro/cooldown/target checks
2. Determine variant based on defender and attacker combat positions
3. Select the correct config values per variant
4. Call `combat.ExecuteSkillMove` with variant-specific params
5. If stomp: extend prone duration (`defender.PositionRoundsMin += 1`)
6. Return which variant was used

This gives mobs kick variant parity — a mob fighting a prone
player will now stomp instead of regular kick.

- [ ] **Step 3: Update user kick wrapper**

Refactor to use `actions.ExecuteKick()`. Keep user-specific:
- Initiation from non-combat
- Variant-specific flavor text arrays
- Skill progression event
- User SendText formatting

- [ ] **Step 4: Update mob kick wrapper**

Refactor to use `actions.ExecuteKick()`. Now mobs get stomp/knee
variants automatically. Add appropriate mob-side messages for
each variant (simpler than user side — no flavor arrays, just
"mob stomps you" / "mob knees you" style).

- [ ] **Step 5: Verify build**

Run: `go build ./...`

- [ ] **Step 6: Commit**

```bash
git add internal/actions/combat_kick.go \
  internal/usercommands/kick.go internal/mobcommands/kick.go
git commit -m "refactor: shared Kick action with stomp/knee variant parity for mobs"
```

---

### Task 4: Shared Trip Action (with tailsweep parity)

**Files:**
- Create: `internal/actions/combat_trip.go`
- Modify: `internal/usercommands/trip.go`
- Modify: `internal/mobcommands/trip.go`

- [ ] **Step 1: Read both trip files fully**

User trip checks `user.Character.Mutations["tail"]` for tailsweep
variant with different stats (damagePercent=0.40, knockdownChance=70).

- [ ] **Step 2: Create shared trip action**

```go
type TripVariant int
const (
    TripStandard TripVariant = iota
    TripTailsweep // attacker has tail mutation
)

type TripMoveResult struct {
    SkillMoveResult
    Variant TripVariant
}

func ExecuteTrip(actor Actor) TripMoveResult
```

ExecuteTrip should:
1. Standard aggro/cooldown/target checks
2. Check `actor.GetCharacter().Mutations` for tail mutation
3. Select standard trip or tailsweep params accordingly
4. Call `combat.ExecuteSkillMove`
5. Return which variant was used

- [ ] **Step 3: Update user trip wrapper**

Refactor to use `actions.ExecuteTrip()`. Keep: variant messages,
skill progression, user formatting.

- [ ] **Step 4: Update mob trip wrapper**

Refactor to use `actions.ExecuteTrip()`. Mobs with tail mutation
will now tailsweep. Add mob-side tailsweep message.

- [ ] **Step 5: Verify build + commit**

```bash
git add internal/actions/combat_trip.go \
  internal/usercommands/trip.go internal/mobcommands/trip.go
git commit -m "refactor: shared Trip action with tailsweep parity for mobs"
```

---

### Task 5: Shared Grapple Action

**Files:**
- Create: `internal/actions/combat_grapple.go`
- Modify: `internal/usercommands/grapple.go`
- Modify: `internal/mobcommands/grapple.go`

Grapple already uses the shared `combat.ExecuteGrappleMove` engine.
The extraction here is thin — just the target resolution and
cooldown wrapper.

- [ ] **Step 1: Read both grapple files, create shared action**

```go
type GrappleMoveResult struct {
    Target   AggroTarget
    Result   combat.GrappleResult // or whatever ExecuteGrappleMove returns
    Executed bool
    OnCooldown bool
    NoTarget   bool
}

func ExecuteGrapple(actor Actor) GrappleMoveResult
```

- [ ] **Step 2: Update both wrappers**

- [ ] **Step 3: Verify build + commit**

```bash
git commit -m "refactor: shared Grapple action with combat helpers"
```

---

### Task 6: Shared Shoot Action

**Files:**
- Create: `internal/actions/combat_shoot.go`
- Modify: `internal/usercommands/shoot.go`
- Modify: `internal/mobcommands/shoot.go`

Shoot is nearly identical on both sides (~85% shared). It parses
a direction, finds targets in adjacent rooms, and sets remote aggro.

- [ ] **Step 1: Read both shoot files, create shared action**

The shared action should handle:
1. Check weapon type is `items.Shooting`
2. Parse direction from args
3. Find exit, load adjacent room
4. Find valid targets in adjacent room
5. Set `SetAggroRemote(exitName, ...)`

The user wrapper keeps: party protection, PvP checks, user SendText.
The mob wrapper keeps: room-only messaging.

- [ ] **Step 2: Update both wrappers**

- [ ] **Step 3: Verify build + commit**

```bash
git commit -m "refactor: shared Shoot action with combat helpers"
```

---

### Task 7: Shared Attack Target Resolution

**Files:**
- Create: `internal/actions/combat_attack.go`
- Modify: `internal/usercommands/attack.go`
- Modify: `internal/mobcommands/attack.go`

Attack is the most complex. Extract the wildcard and named target
resolution that's duplicated between both sides.

- [ ] **Step 1: Read both attack files fully**

- [ ] **Step 2: Create shared attack helpers**

```go
// ResolveAttackTarget finds a target in the room by name or
// wildcard pattern. Returns the target's user/mob IDs.
type AttackTarget struct {
    UserId        int
    MobInstanceId int
    Name          string
    Found         bool
    IsSelf        bool
}

// FindAttackTarget resolves a target from the input string.
// Handles wildcards (*, *mob, *user, *ANYONE) and named targets.
func FindAttackTarget(actor Actor, rest string, room *rooms.Room) AttackTarget
```

This extracts the ~45-line wildcard resolution and the named
target lookup.

- [ ] **Step 3: Update user attack wrapper**

Use `actions.FindAttackTarget()` for target resolution. Keep:
party auto-assist, target-switching, PvP guard, surprise attack
cooldown, charm assist, user messaging.

- [ ] **Step 4: Update mob attack wrapper**

Use `actions.FindAttackTarget()` for target resolution. Keep:
`PlayerAttacked()` tracking, mob messaging, darkness awareness.

Note: The hidden mob fix is already in this file — make sure it's
preserved when refactoring.

- [ ] **Step 5: Verify build + commit**

```bash
git commit -m "refactor: shared Attack target resolution"
```

---

### Task 8: Combat Parity Tests

**Files:**
- Create: `internal/actions/combat_test.go`

- [ ] **Step 1: Write tests for combat helpers**

Tests needed:

1. **TestResolveAggroTarget_User** — aggro pointing at user ID
2. **TestResolveAggroTarget_Mob** — aggro pointing at mob instance ID
3. **TestResolveAggroTarget_Nil** — nil aggro returns Found=false
4. **TestResolveAggroTarget_InvalidId** — user/mob not found
5. **TestTryCombatCooldown_Fresh** — not on cooldown returns false
6. **TestTryCombatCooldown_Active** — on cooldown returns true
7. **TestKickVariant_Standing** — standard kick for standing target
8. **TestKickVariant_Prone** — stomp variant for prone target
9. **TestKickVariant_Grapple** — knee variant for grapple controller
10. **TestTripVariant_NoTail** — standard trip without tail mutation
11. **TestTripVariant_WithTail** — tailsweep with tail mutation

Note: Some of these tests may require setting up characters with
combat state (aggro, positions, mutations). Read the existing test
infrastructure to understand how to create test characters with
the right state. If the tests can't easily test ExecuteKick etc.
(because they require full combat state), test the variant
selection logic in isolation instead.

- [ ] **Step 2: Run tests**

Run: `go test ./internal/actions/ -v -run "Combat|Kick|Trip|Resolve|Cooldown"`

- [ ] **Step 3: Commit**

```bash
git commit -m "test: combat helper and variant parity tests"
```

---

### Task 9: Final Verification

- [ ] **Step 1: Full build**

Run: `go build ./...`

- [ ] **Step 2: Run all tests**

Run: `go test ./internal/actions/ ./internal/usercommands/ ./internal/mobcommands/ ./internal/combat/ -count=1 -timeout 120s`

- [ ] **Step 3: Manual smoke test**

Test in-game:
- `bash` works against a mob (player has shield)
- Mob bashes player back
- `kick` while target is prone → stomp message
- `trip` works, mob trips player
- Hidden mob attacks → revealed immediately, player can fight back
- `shoot` from adjacent room works
- `grapple` initiates grapple correctly

- [ ] **Step 4: Commit any fixups**
