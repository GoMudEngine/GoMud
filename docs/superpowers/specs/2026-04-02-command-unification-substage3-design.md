# Command Unification — Substage 3: Combat Commands

**Date:** 2026-04-02
**Goal:** Extract shared combat command logic for attack initiation
and special moves (bash, kick, trip, grapple, shoot). Fix combat
asymmetries where mobs lack position-aware variants that players have.

## Current State

The combat *calculation* layer is already unified — all four
matchup types (PvM, PvP, MvP, MvM) converge at a single
`calculateCombat()` function in `internal/combat/`. The
duplication lives in the **command entry points**: target
resolution, cooldown checks, and messaging.

### Duplication Inventory

| Pattern | Duplicated In | Lines Each |
|---------|--------------|------------|
| Target resolver (aggro → character) | bash, kick, trip, grapple, shoot (x2 each) | ~25 lines |
| Cooldown check + message | bash, kick, trip, grapple (x2 each) | ~8 lines |
| Wildcard target selection | attack (x2) | ~45 lines |
| Analytics + round cost tail | all special moves (x2 each) | ~5 lines |

### Commands In Scope

| Command | User LOC | Mob LOC | Shareability |
|---------|----------|---------|-------------|
| attack | 335 | 168 | ~50% (wildcard resolution) |
| bash | 154 | 133 | ~80% (nearly identical) |
| kick | 300 | 128 | ~60% (variants user-only) |
| trip | 206 | 128 | ~60% (tail user-only) |
| grapple | 150 | 114 | ~80% (already uses shared engine) |
| shoot | 133 | 104 | ~85% (nearly identical) |

### Commands NOT In Scope

- **flee** — intentionally asymmetric (player enters flee state,
  mob flees instantly). Different by design.
- **taunt** — user-only, no mob equivalent. Would require new
  mob AI for conviction damage. Defer to substage 5.

## Approach

### Shared Combat Helpers

Extract into `internal/actions/combat_helpers.go`:

**1. ResolveAggroTarget** — the ~25-line target resolver block
that appears in every special move. Takes an aggro struct,
returns the defender's `*characters.Character`, name, and IDs.

```go
type AggroTarget struct {
    Char          *characters.Character
    Name          string
    UserId        int
    MobInstanceId int
    Found         bool
}

func ResolveAggroTarget(aggro *characters.Aggro) AggroTarget
```

Both user and mob special moves call this instead of duplicating
the `Aggro.UserId > 0` / `Aggro.MobInstanceId > 0` branching
with `users.GetByUserId` / `mobs.GetInstance` lookups.

**2. TryCombatCooldown** — the cooldown check pattern.

```go
func TryCombatCooldown(char *characters.Character, moveName string, cooldownRounds int) (bool, string)
```

Returns (onCooldown bool, message string). Both sides call this
instead of duplicating the `char.Cooldowns.Try(...)` pattern.

**3. RecordAndWait** — the analytics + round cost tail.

```go
func RecordAndWait(char *characters.Character, moveName string, sourceType int)
```

Calls `combat.RecordSpecialMove(...)` and sets
`char.Aggro.RoundsWaiting = 1`.

### Per-Command Shared Actions

Each special move gets a shared action that handles the core
mechanic (resolve target, check cooldown, call ExecuteSkillMove,
record analytics). The wrappers handle messaging and actor-specific
concerns.

**Pattern for each special move:**

```go
type SkillMoveResult struct {
    Target    AggroTarget
    Success   bool
    OnCooldown bool
    CooldownMsg string
    MoveResult combat.SkillMoveResult  // from ExecuteSkillMove
    // Command-specific fields (e.g., kick variant)
}

func ExecuteBash(actor Actor) SkillMoveResult
func ExecuteKick(actor Actor) SkillMoveResult
func ExecuteTrip(actor Actor) SkillMoveResult
```

### Kick Variant Parity

Currently only the user side checks combat positions for
stomp (target prone) and knee (grapple + controller). Mobs
should also use these variants — a mob fighting a prone
target should stomp, not kick.

The shared `ExecuteKick` will:
1. Check `defender.CombatPosition` for prone → stomp variant
2. Check attacker position + grapple controller → knee variant
3. Apply the correct damage/knockdown config per variant
4. Return which variant was used so wrappers can pick messages

### Trip / Tailsweep Parity

Currently only the user side checks for tail mutation to use
tailsweep. Mobs with the tail mutation should also tailsweep.

The shared `ExecuteTrip` will:
1. Check `attacker.Character.Mutations["tail"]` for tail mutation
2. Apply tailsweep stats if tail mutation present
3. Return which variant was used

### Attack Initiation

The attack command is more complex than special moves. The shared
core extracts:

1. **Wildcard target resolution** — the `*`/`*mob`/`*user` logic
   that selects a random valid target from the room. Currently
   ~45 lines duplicated verbatim.

2. **Named target resolution** — find target by name in room.

3. **Self-attack guard** — prevent attacking yourself.

The user wrapper keeps: party logic, PvP guard, target-switching,
surprise attack cooldown, charm assist.

The mob wrapper keeps: `PlayerAttacked()` tracking, simplified
random targeting.

### Hidden Mob Fix (already applied)

The hidden mob combat bug (mobs staying invisible after attacking)
is already fixed in this substage — `CancelBuffsWithFlag(Hidden)`
is now called immediately when a mob commits to attacking, rather
than waiting for the next combat loop tick.

## Testing Strategy

### Parity Tests

For each shared combat helper:
- Test ResolveAggroTarget with both user and mob aggro targets
- Test TryCombatCooldown with fresh and on-cooldown states
- Test that kick variant selection produces the correct variant
  for each combat position (standing, prone, grapple)
- Test that trip variant selection detects tail mutation

### Conservation (less critical here)

Combat commands don't move items, so conservation invariants
don't apply. The key assertions are:
- Same SkillMoveParams for same inputs regardless of actor type
- Variant selection matches combat position
- Cooldown state is properly consumed

## Order of Implementation

1. **Shared combat helpers** — ResolveAggroTarget, TryCombatCooldown,
   RecordAndWait
2. **Bash** — simplest special move, proves the pattern
3. **Kick** — add variant parity for mobs
4. **Trip** — add tailsweep parity for mobs
5. **Grapple** — already mostly shared, thin extraction
6. **Shoot** — nearly identical, straightforward
7. **Attack** — most complex, wildcard resolution extraction
8. **Tests**

## What This Does NOT Change

- Combat calculation engine (already unified in internal/combat/)
- Combat loop in NewRound_DoCombat.go
- Flee (intentionally asymmetric)
- Taunt (deferred to substage 5)
- Message formatting (stays in wrappers — different ANSI colors,
  darkness awareness for mobs, player feedback text)
- Skill progression events (user-only — mobs don't progress)
- Party/charm assist in attack (user-only)

## Risks

1. **Kick variant for mobs:** Mobs currently don't check combat
   positions. Adding stomp/knee means mobs will deal more damage
   to prone targets (stomp) and in grapples (knee). This is a
   balance change — intentional and desired, but worth noting.

2. **Tailsweep for mobs:** Mobs with tail mutations will now
   tailsweep instead of trip. This is also a balance change.
   Currently no mobs have the tail mutation, so this is future-
   proofing rather than an immediate change.

3. **Message formatting divergence:** User and mob sides use
   different ANSI color tags (username vs mobname) and darkness
   awareness. The shared action returns result data; the wrapper
   formats messages. This keeps the shared core clean but means
   message formatting stays duplicated (by design).
