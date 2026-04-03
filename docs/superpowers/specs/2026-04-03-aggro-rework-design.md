# Aggro/Targeting Rework — Design Spec

**Date:** 2026-04-03
**Goal:** Centralize aggro state management into 4 well-defined
functions. Eliminate scattered stale-aggro patches and inconsistent
cleanup. Handle companion combat groups cleanly.

## Problem

Aggro state is managed in 20+ scattered locations with inconsistent
cleanup. Some use `EndAggro()` (cleans grapple), some use raw
`Aggro = nil` (skips grapple cleanup). Auto-retarget only fires
from some kill paths. Companions make it worse — their kills leave
the owner with stale aggro, and idle companions don't pick up new
threats to their owner.

## Solution: 4 Centralized Functions

### 1. EndAggro() — always, no raw nil

Replace ALL `Aggro = nil` with `EndAggro()` across the codebase.
`EndAggro()` already clears grapple state. No exceptions.

~20 call sites need updating (see audit in research).

### 2. ValidateAggro() — single stale check

New method on `*Character`:

```go
func (c *Character) ValidateAggro() bool
```

Returns true if aggro is valid, false if cleared. Called once per
round at the top of both `handlePlayerCombat` and `handleMobCombat`.
Replaces all scattered stale checks.

Logic:
- Aggro nil → return false
- MobInstanceId > 0: check `mobs.GetInstance` exists and Health > 0
- UserId > 0: check `users.GetByUserId` exists and Health > 0
- If invalid: `EndAggro()`, return false

This removes the stale checks from:
- `NewRound_DoCombat.go:64-75` (pre-loop player check)
- `NewRound_UserRoundTick.go:125-129` (round tick safety net)
- `handlePlayerVsMob:1047,1053,1060` (inline checks)
- `handlePlayerVsPlayer:897-926` (inline checks)
- `handleMobVsMob:1465-1469` (inline checks)

### 3. RetargetOrEnd() — smart retarget after kills

New function (in `internal/hooks/` or `internal/actions/`):

```go
func RetargetOrEnd(char *Character, room *Room) bool
```

Called whenever a target dies. Replaces `handleAutoRetargetPlayer`
and extends it to work for both players and mobs (companions).

Logic:
1. `char.EndAggro()`
2. Scan room for mobs attacking this character (or attacking the
   character's charm owner, for companions)
3. If found: `char.SetAggro(target)`, return true
4. Scan room for players attacking this character
5. If found: `char.SetAggro(target)`, return true
6. Return false (out of combat)

For companion mobs, "attacking my owner" counts as a valid retarget.

### 4. CompanionAutoTarget() — idle companion threat scan

Runs in the mob round tick (or combat loop) for charmed mobs with:
- `AutoAssist = true` (from CompanionInfo)
- `Aggro == nil` (currently idle)
- Owner is in combat or being attacked

Logic:
1. Get charm owner
2. If owner has aggro: companion attacks owner's target
3. Else if any mob in room is attacking owner: companion attacks
   that mob
4. Else: stay idle

This catches the case where a new mob attacks the owner while
the companion just finished a kill, or where a mob attacks the
owner and the companion hasn't been pulled into combat yet.

## What Changes

### Combat loop (`NewRound_DoCombat.go`):

Before:
```
for each player:
  [scattered stale checks]
  handlePlayerVsX(...)
```

After:
```
for each player:
  if !player.ValidateAggro():
    RetargetOrEnd(player, room)
    if player.Aggro == nil: continue
  handlePlayerVsX(...)
```

Same pattern for mob loop.

### Kill paths (all 4 combat handlers):

Before:
```
if defenderDead:
  attacker.EndAggro()
  defender.EndAggro()
  handleAutoRetargetPlayer(...)  // only sometimes
```

After:
```
if defenderDead:
  defender.EndAggro()
  RetargetOrEnd(attacker, room)
```

### Raw `Aggro = nil` sites:

All ~20 sites converted to `EndAggro()`. Grep for `\.Aggro = nil`
and replace each one.

### Companion idle scan:

Add `CompanionAutoTarget()` call in `handleMobCombat` loop for
charmed mobs with nil aggro, or in `MobIdle` handling.

## Import Cycle Consideration

`ValidateAggro()` needs `mobs.GetInstance` and `users.GetByUserId`.
The `characters` package may not import `mobs` or `users`. If so,
`ValidateAggro` should be in `internal/hooks/` or `internal/actions/`
as a standalone function taking `*Character` + lookup functions.

## Testing

- Companion kills mob → owner retargets to next attacker
- All mobs die → player exits combat cleanly
- New mob attacks player mid-fight → idle companion picks it up
- Party: two players + companions vs 4 mobs → correct targeting
- Grapple state clears on all combat exit paths
- No "can't do that while fighting" after all enemies dead
