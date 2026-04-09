# Command Unification Design Spec

**Date:** 2026-04-02
**Goal:** Unify usercommands and mobcommands via shared core logic,
fix magic system parity, simplify combat code, and establish
guardrails to prevent future drift.

## Problem Statement

The user command system (151 files, ~170 commands) and mob command
system (56 files, ~50 commands) have drifted significantly apart.
They have different function signatures, different registration
metadata, and 12 of the 25 shared commands have diverged in
behavior. Most critically:

- Mobs cast spells instantly while players use fold accumulation
- Economy commands (buy, sell, give) are missing or incomplete for mobs
- Combat commands (attack, bash, kick, trip) have diverged
- Craft, forage, and sneak don't exist for mobs
- No mechanism prevents continued drift

This blocks planned work on NPC behavior, the attitude/faction
system, and the MUD economy.

## Architecture

### Approach: Shared Core (Approach A)

Shared logic lives in `internal/actions/`. Each action is a
function that operates on an Actor interface. User and mob
commands remain in their own packages as thin wrappers.

### Actor Interface

```go
// internal/actions/actor.go
type Actor interface {
    GetCharacter() *characters.Character
    GetRoom() *rooms.Room
    SendText(msg string)
    GetInstanceId() int
    IsPlayer() bool
    // Room-broadcast helper — sends text to everyone else in room
    SendRoomText(msg string, excludeSelf bool)
}
```

Both `*users.UserRecord` and `*mobs.Mob` implement this interface.

- `UserRecord.SendText()` → sends to the player's terminal
- `Mob.SendText()` → no-op (or debug log if mob debug enabled)
- `UserRecord.SendRoomText()` → broadcasts to room via events
- `Mob.SendRoomText()` → broadcasts to room via mob event helpers
- `UserRecord.IsPlayer()` → true
- `Mob.IsPlayer()` → false

### Shared Action Functions

```go
// internal/actions/say.go
func Say(actor Actor, text string) (bool, error)

// internal/actions/get.go
func Get(actor Actor, itemName string, fromContainer string) (bool, error)

// internal/actions/attack.go
func InitiateCombat(actor Actor, targetId int) (bool, error)
```

### Wrapper Pattern

```go
// usercommands/say.go
func Say(rest string, user *users.UserRecord, room *rooms.Room,
    flags events.EventFlag) (bool, error) {
    // User-specific: check muted status
    // User-specific: drunk text transformation
    return actions.Say(user, rest)
    // User-specific: GMCP comm update (if needed)
}

// mobcommands/say.go
func Say(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
    // Mob-specific: skip if no players in room
    return actions.Say(mob, rest)
}
```

### Concern Separation for Say/Emote

The shared action is a **verb** — it handles "make words appear
in the room." The decision systems that trigger it are separate:

| System | Drives | Untouched by unification |
|--------|--------|--------------------------|
| Dialogue engine | Quest/NPC scripted responses | Yes |
| Idle scripts | Ambient flavor (gossip, barks) | Yes |
| Combat hooks | Taunts, incantations, death cries | Yes |
| Player input | Direct typing | Yes |

All four call the same shared `actions.Say()` underneath.

## Substages

### Substage 1 — Actor Interface + Registry Audit + Pattern

**Deliverables:**
- `internal/actions/` package with Actor interface
- Actor implementation on UserRecord and Mob
- Registry audit: startup check comparing user vs mob command
  sets, with allowlist for intentional gaps (admin commands,
  mob-only AI commands like `lookfortrouble`)
- Extract 3 simple commands as proof of pattern: say, emote, go
- Shared test framework: tests that exercise the shared action
  with both a mock user-actor and mock mob-actor

**Intentional divergences allowlist (initial):**
- Admin commands (user-only): all commands with AdminOnly=true
- AI commands (mob-only): lookfortrouble, wander, idle, converse
- UI commands (user-only): status, prompt, set, help, inventory
  display, map, score, quest log
- Note: `train` command to be removed (vestigial, does nothing).
  `meditate` and `levelup` already removed from codebase.

### Substage 2 — Economy Commands

Extract shared core for:
- `get` / `drop` — item pickup and placement
- `give` — item transfer between actors
- `buy` / `sell` — merchant transactions
- `equip` / `remove` — equipment management

Mob wrappers created for any missing commands. Tests for
behavioral parity (same item, same room → same result).

### Substage 3 — Combat Unification

Extract into shared core:
- `attack` — target resolution, aggro initiation
- `bash` / `kick` / `trip` — combat techniques
- Defense resolution (already partly shared in internal/combat/)
- Damage application pipeline (already shared)

Simplify combat_helpers.go by extracting actor-agnostic logic
into internal/combat/ functions that both sides call.

### Substage 4 — Magic System Parity

**This is the biggest piece.** Currently:
- Users: fold-based accumulation over multiple rounds
- Mobs: instant single-round execution

**Target state:** Both use fold-based casting.

- Extract fold accumulation engine into shared core
- Mob casting interface: `actions.StartCasting(actor, spell)` —
  fold count comes from the spell's `BaseFolds` field (same as
  players), accumulation rate from Perception + skill level via
  `CalcFoldsPerRound()`
- Spell interruption works symmetrically (damage during casting
  can disrupt either side)
- Spell resolution (damage, healing, buffs) uses same shared
  functions for both actor types

Mob AI for spell selection (not fold count — that's per-spell):
- Simple mobs: pick from known spells based on situation
- Caster-archetype mobs: higher Perception → faster fold rate
  → effectively shorter cast times
- Boss mobs: know powerful spells with high BaseFolds
  (interruptible — creates tactical counterplay)

### Substage 5 — Skills + NPC Behavior Foundation

Extract into shared core:
- `craft` — crafting at stations
- `search` / `forage` — resource gathering
- `sneak` — stealth movement and hiding

Simplify combat code using patterns from substages 3-4.

Add initial NPC behavior hooks:
- Mobs can evaluate "should I buy/sell/craft/forage" as idle
  behaviors
- Foundation for attitude system: Actor interface includes
  `GetAttitudeToward(targetId int) Attitude` (stub for now)

### Substage 6 — Tests + Guardrails

- Parity test suite: for every shared action, test with both
  user-actor and mob-actor, verify same mechanical outcome
- Registry audit covers all tiers, runs at startup
- `INTENTIONAL_DIVERGENCES.md` documents every allowlisted gap
  with rationale
- CI integration: tests run on every push

## Testing Strategy

### Parity Tests

For each shared action, a test creates a mock user-actor and a
mock mob-actor in the same room with the same stats/items, runs
the action on both, and asserts the mechanical outcome matches:
- Same items moved
- Same damage dealt
- Same buffs applied
- Same room state changes

User-specific side effects (GMCP, prompt, quest triggers) and
mob-specific side effects (AI logging, empty-room skips) are
NOT tested for parity — those are intentional divergences.

### Registry Audit

At startup, compare command registries. For each command in
either registry:
1. Check if it exists in the other registry
2. If not, check the allowlist
3. If not allowlisted, log a warning (dev) or panic (release)

The allowlist is a Go map in `internal/actions/divergences.go`,
not a YAML file, so it's version-controlled and code-reviewed.

## What This Does NOT Change

- Command dispatch (TryCommand) stays in each package
- Alias resolution stays user-specific
- Room script interception stays user-specific
- Dialogue engine untouched
- Idle behavior scripts untouched
- Event flags stay user-specific (mob wrappers don't need them)
- The mob command AI (lookfortrouble, wander, etc.) untouched

## Risks

1. **Character field difference:** UserRecord has `*Character`
   (pointer), Mob has `Character` (value). The Actor interface
   returns `*Character` which works for both (Go takes address
   of value field automatically via method receiver).

2. **Room broadcast asymmetry:** User events use SourceUserId,
   mob events use SourceMobInstanceId. The shared action needs
   to handle this — likely via IsPlayer() branching inside the
   SendRoomText implementation, not in the action itself.

3. **Fold system complexity:** Adding folds to mobs is the
   riskiest substage. Needs careful testing to ensure combat
   balance doesn't shift dramatically. Mob fold counts need
   tuning.

4. **Performance:** Mob idle behavior (buy/sell/craft) running
   through the full shared action path could be heavier than
   the current lightweight mob commands. Profile if needed.
