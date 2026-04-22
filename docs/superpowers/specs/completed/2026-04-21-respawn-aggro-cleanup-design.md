# Respawn Aggro Cleanup + Condition Wipe + Grace Period — Design

**Date:** 2026-04-21
**Status:** Approved
**Related memory:** `project_respawn_aggro_death_loop.md`
**Repro evidence:** `bug_log.txt` (Duard double-death on prod)

## Problem

On 2026-04-21 a prod incident showed a player (Duard) dying, respawning
at 5% HP per the Death.RespawnPoolFraction rule, and then dying AGAIN
before any further attack line appeared in the log — followed by
temple priest Olen aggroing. Analysis of `internal/usercommands/suicide.go`
uncovered three independent gaps in the current death flow:

1. **Inbound aggro is not cleared.** `user.Character.EndAggro()` clears
   the dying player's *outbound* aggro (line 179), but every mob in
   the room whose `Aggro.UserId == dyingUserId` keeps its aggro
   through the respawn + room move. When the player respawns, those
   mobs (or an auto-retarget chain sourced from them) can immediately
   re-engage.

2. **Charmed companions retain their aggro through the respawn.**
   Companions follow the owner on room change via `TransportCompanions`.
   If a skeleton companion was attacking Mob X in the old room, it
   arrives in the home room with `Aggro = {Mob X}` still set. The next
   tick's `ValidateAggro` → `RetargetOrEnd` → `CompanionAutoTarget`
   chain can pick an unintended new target in the home room (e.g., an
   otherwise-neutral temple priest).

3. **Combat conditions are not cleared on death.** Suicide.go clears
   `Buffs` via `CancelBuffsWithFlag(buffs.All)` at line 178 but
   leaves the `Conditions` slice intact. A player respawning at 5% HP
   who was poisoned at death is still poisoned — the next DoT tick
   kills them again, and the log shows a second "has DIED" line with
   no attacker named. This matches the Duard repro exactly.

There is no `PlayerDeath` event listener in `internal/hooks/` — the
event fires from suicide.go but only Discord listens. All game-state
cleanup lives inline in Suicide(), and the fix extends it rather than
splitting into a new listener.

## Design rule

> **On player death, every aggro relationship touching the dying
> player (outbound, inbound, companion) is cleared before the
> respawn room move, every combat condition is wiped, and the
> respawning player is protected by a short grace period during
> which no mob can acquire aggro on them.**

## Scope

Three fixes, landing together in one branch:

- **Fix A — Comprehensive aggro cleanup on death.** Three categories:
  the dying player's outbound aggro (already cleared), every mob in
  the pre-respawn room aggro'd on the player, every companion mob's
  own aggro.
- **Fix B — Clear all combat conditions on death.** Matches the
  existing `CancelBuffsWithFlag(buffs.All)` precedent — `Conditions`
  slice set to `nil`.
- **Fix C — Post-respawn grace period.** A new `NoAggroTarget` buff
  flag, a new buff (id 81, "Respawn Grace", duration configurable via
  `Death.RespawnGraceRounds` default 3), and guards in three sites
  that attempt aggro: `Character.SetAggro`, `RetargetOrEnd`, and
  (verification-pass) `CompanionAutoTarget`.

## Code changes

### Fix A — Comprehensive aggro cleanup in `internal/usercommands/suicide.go`

After the existing `user.Character.EndAggro()` at line 179, before
`MoveToRoom` at line 225 (current positions):

```go
// Inbound aggro: any mob in the pre-respawn room that was targeting
// this player stops targeting.
for _, mobInstId := range room.GetMobs(rooms.FindFighting) {
    mob := mobs.GetInstance(mobInstId)
    if mob == nil || mob.Character.Aggro == nil {
        continue
    }
    if mob.Character.Aggro.UserId == user.UserId {
        mob.Character.EndAggro()
    }
}

// Companion aggro: the dying player's charmed mobs' own aggro.
// Cleared here so when TransportCompanions moves them to home room,
// they arrive with blank aggro state and can't auto-retarget via
// CompanionAutoTarget (which only fires if someone IS attacking the
// owner — no false positive in home room).
for _, compInstId := range user.Character.GetCharmIds() {
    comp := mobs.GetInstance(compInstId)
    if comp == nil {
        continue
    }
    comp.Character.EndAggro()
}
```

### Fix B — Clear combat conditions in `internal/usercommands/suicide.go`

Immediately after the existing `CancelBuffsWithFlag(buffs.All)` at
line 178:

```go
user.Character.CancelBuffsWithFlag(buffs.All)
user.Character.Conditions = nil   // wipe poison, bleeding, rally,
                                  // warcry, shield, regen, etc.
user.Character.EndAggro()
```

Matches the wipe-everything semantic of the existing buff line.

### Fix C — Respawn grace period

**(c.1) New buff flag** in `internal/buffs/flags.go` (or wherever
existing buff flags live — check for the current file):

```go
NoAggroTarget Flag = "NoAggroTarget"
```

Add to the flag registry/parsing alongside existing flags like
`Poison`, `All`, `ReviveOnDeath`.

**(c.2) New buff YAML** `_datafiles/world/dogmud/buffs/81-respawn_grace.yaml`:

```yaml
buffid: 81
name: Respawn Grace
description: The world gives you a moment to gather yourself after death.
flags:
  - NoAggroTarget
roundinterval: 1
triggercount: 3
statmods: {}
messages:
  end: The world returns its attention to you.
```

`triggercount` is the default duration (3 rounds). The `Death.RespawnGraceRounds`
config knob can override in future if we want per-environment tuning;
v1 uses the YAML default.

**(c.3) Enforcement sites** (three):

**Cycle constraint:** `internal/users` already imports `internal/characters`,
so `characters/aggro.go` cannot directly import `users` to look up a
target user's buff flag. The codebase has a pattern for this exact
shape: `rooms.SetCompanionTransport` / `SetBTreeStateEvictor`
register a callback at boot that the characters/rooms package calls
without needing to know the caller's type.

**(c.3.a) Callback interface in `characters/aggro.go`:**

Add a package-level function variable and a public setter:

```go
// userUntargetableFn is registered from hooks at boot. Returns true
// if the user with the given id is protected from incoming aggro
// (e.g., post-respawn grace). Called from SetAggro before setting
// aggro on a player target. nil = no check (safe default for tests).
var userUntargetableFn func(userId int) bool

// SetUserUntargetableCheck registers the grace-flag check. Called
// from main.go at boot. Repeated registrations overwrite (last wins).
func SetUserUntargetableCheck(fn func(userId int) bool) {
    userUntargetableFn = fn
}
```

Then `Character.SetAggro` gates at the top:

```go
func (c *Character) SetAggro(userId int, mobInstanceId int, ...) {
    if userId > 0 && userUntargetableFn != nil && userUntargetableFn(userId) {
        return // target user is grace-protected
    }
    // ... existing logic
}
```

**(c.3.b) Register the callback in `main.go`:**

At boot, after the `rooms.SetCompanionTransport(...)` line (grep to
find the exact location — Stage 4 Phase 4 work added that one):

```go
characters.SetUserUntargetableCheck(func(userId int) bool {
    user := users.GetByUserId(userId)
    if user == nil {
        return false
    }
    return user.Character.HasBuffFlag(buffs.NoAggroTarget)
})
```

**(c.3.c) `RetargetOrEnd` in `internal/hooks/aggro_helpers.go`:**

Because the scanning character uses `char.SetAggro(...)` at lines 86,
103, 118, those calls will now short-circuit via the callback. **No
direct change to RetargetOrEnd is strictly required** — the SetAggro
guard catches it. But for clarity and to avoid scanning the room at
all when the scanner is looking for an already-grace-protected target
(not the common case but still possible), consider a defense-in-depth
skip. Task 2 of the plan evaluates.

**(c.3.d) `CompanionAutoTarget` in `internal/hooks/aggro_helpers.go`:**

This function makes a companion attack whoever is attacking the
companion's owner. After Fix A + Fix C.c.3.a, the owner's aggressors
are already filtered at the SetAggro level — no mob should have the
owner targeted during the grace period. So CompanionAutoTarget's
owner-defense scan will naturally find no candidates. **No change
needed**, but Task 2 adds a one-line early-return as defense-in-depth:

```go
if owner.Character.HasBuffFlag(buffs.NoAggroTarget) {
    return
}
```

**(c.4) Apply the grace buff in `suicide.go`** after the "Darkness swallows
you" message (line 217), before `MoveToRoom` (line 225):

```go
graceRounds := int(config.Death.RespawnGraceRounds)
if graceRounds > 0 {
    user.Character.AddBuff(81, false)
}
```

**(c.5) New config knob** `Death.RespawnGraceRounds` in the GamePlay
config section (wherever `Death.RespawnPoolFraction` lives — likely
`internal/configs/config.gameplay.go` or similar). Default `3`.

Setting the knob to `0` disables the grace mechanic (tests for A+B
alone, or operators who prefer no grace period).

## Edge cases accepted

- **Companion aggro follows owner correctly today.** The companion's
  aggro-chain cleanup we add is preventive, not curative for an
  existing live bug we can point at. It closes one of three plausible
  root-cause paths for the Duard 2nd-death. The other paths (DoT
  conditions, inbound aggro) are covered by Fix B and Fix A's inbound
  loop.
- **`ValidateAggro` next-tick fallback.** Even if Fix A's explicit
  clearing has gaps (e.g., a mob outside the room with aggro on the
  player), `ValidateAggro` catches stale aggro on the mob's next
  combat tick via the "target not in my current room" check. Fix A's
  in-room cleanup covers the common case; `ValidateAggro` is the
  backstop for weird setups.
- **Grace-period time-only, not action-based.** A player who attacks
  during the 3-round grace gets their attack through but remains
  grace-protected from retaliatory aggro. This is fine for PvE; in
  PvP it gives the respawning player a 3-round window to retaliate
  with impunity. If this becomes grief-abuse, action-based expiry can
  be added as a follow-up.
- **NoAggroTarget only applies to SetAggro/RetargetOrEnd.** Direct
  attack commands by the player (e.g., `attack <mob>`) aren't gated
  — those are player-initiated. The grace flag only prevents
  INCOMING aggro. Matches the usual MMO grace-period semantic.
- **Clearing all conditions also wipes buff-like conditions (rally,
  warcry, regen).** Matches the existing "wipe all buffs" precedent
  one line up. Death is a hard reset; this is by design.

## Testing

### Unit tests (Go)

1. **`TestSuicide_ClearsInboundAggro`** — Seed a mob with `Aggro.UserId = user.UserId`. Call Suicide flow. Assert `mob.Character.Aggro == nil`.
2. **`TestSuicide_ClearsCompanionAggro`** — Give user a charmed companion with `Aggro` set to some prior target. Call Suicide. Assert companion's `Aggro == nil`.
3. **`TestSuicide_ClearsConditions`** — User has `ConditionPoisoned` + `ConditionBleeding` + `ConditionRegen`. Call Suicide. Assert `len(user.Character.Conditions) == 0`.
4. **`TestSuicide_AppliesRespawnGrace`** — Call Suicide. Assert `user.Character.HasBuff(81) == true` and `user.Character.HasBuffFlag(buffs.NoAggroTarget) == true`.
5. **`TestSetAggro_SkipsGraceProtectedUser`** — Target user has grace buff applied. Have a mob call `SetAggro(targetUserId, 0, ...)`. Assert the mob's `Aggro == nil` (unchanged).
6. **`TestRetargetOrEnd_BehavesCorrectly`** — Grace-protected player is in the room; a mob with stale aggro calls `RetargetOrEnd`. Whether the mob picks the grace-protected player depends on Task 2's decision; assert matches the chosen behavior.
7. **`TestCompanionAutoTarget_SkipsGraceProtectedOwner`** — A companion whose owner is grace-protected and who has no aggro runs `CompanionAutoTarget`. Assert the companion stays idle (or `Aggro == nil`).

### Smoke tests (in-game)

1. Die in a combat room with multiple mobs attacking. Respawn. For ~3 rounds, mobs in the pre-respawn room don't follow / don't continue attacking via chain. Home-room mobs (if any) don't engage either.
2. Die with poison active. Respawn. Confirm the condition is gone (not on the status line, not ticking HP down).
3. Die in home room containing a normally-peaceful NPC (Olen-style). Respawn. During grace, no engagement. After grace expires, only engagement if the player provokes first.
4. Die while your companion is mid-fight against a target. Respawn. Companion arrives in home room, stays idle.
5. PvP: player A kills player B. Player B respawns. Player A tries to continue attack during grace — attacks should be blocked (no new aggro lock). After grace, normal combat resumes if the players are still in the same room.

## Out of scope

- **Action-based grace expiry** (player attacks during grace → grace ends). V1 is time-only. Easy add-on later.
- **Grace UI indicator / prompt flag.** The buff shows in `status` output normally. No dedicated indicator beyond that.
- **`PlayerDeath` event listener in `internal/hooks/`.** Suicide centralizes death housekeeping; adding an event listener would duplicate. Could refactor later if death logic splits across concerns.
- **Mob-to-mob aggro cleanup on mob death.** Parallel concern (if mob A dies, mob B aggro'd on A should clean up via `ValidateAggro` next tick) — out of scope for this player-death fix. Already partially covered by `handleCombatRound`'s "end aggro when target dies" path.
- **Root-cause identification for "Duard's aggro auto-switched to Olen."** Whatever the specific chain (RetargetOrEnd path, CompanionAutoTarget path, or something else entirely), the 3-layer cleanup + grace period makes it moot. If post-fix smoke shows a specific code path still doing this, a separate ticket investigates.
