# actions — Package Documentation

## Overview

The `actions` package provides the core runtime implementations of in-game
actions — both simple queries (Consider) and complex multi-phase operations
(Combat, Skill, Craft, etc.). Each action is implemented as a public function
that accepts an `Actor` (player or mob abstraction) and target context, then
returns a `*Result` struct with outcome data.

**Structure:**
- Actions are called by user commands (`internal/usercommands/`), mob commands
  (`internal/mobcommands/`), and behavior tree primitives
  (`internal/behaviortree/`).
- Each action returns a structured result for programmatic consumption.
- Messages to players/mobs are sent by actions themselves, not by callers
  (callers only dispatch; actions own messaging).
- Skill progression (`OnStatUse`, `OnSkillUse`) is triggered within actions,
  not by callers.

---

## Actor Abstraction

The `Actor` interface unifies player and mob behavior:

```go
type Actor interface {
	GetCharacter() *characters.Character
	GetMobId() int                  // 0 for players
	GetUserId() string              // "" for mobs
	SendText(text string)
	SendRoom(text string)
	OnStatUse(stat string)
	OnSkillUse(skill string)
	OnCriticalSuccess(skill string)
	OnCriticalFailure(skill string)
}
```

- **UserActor** (`actor_user.go`): wraps a `*users.User`, sends text via
  `user.SendText()`, skill progression goes through `user.Character.OnSkillUse()`.
- **MobActor** (`actor_mob.go`): wraps a `*mobs.Mob`, sends text via room's
  `BroadcastSub()` (mob has no direct text sender), skill progression goes through
  `mob.Character.OnSkillUse()`.

---

## Combat Actions

### Basic Attack

**Function:** `combat.AttackPlayerVsMob(user, mob)`, `combat.AttackMobVsPlayer(mob, user)`, etc.

Handled by the combat system (`internal/combat/`), not the actions package.
See `internal/combat/context.md` for full details.

---

## Skill Actions

### Consider

**Function:** `Consider(actor, target) ConsiderResult`

Computes a power-ratio assessment of `target` from `actor`'s perspective.
- Returns `ConsiderResult` with `Ratio` (self power / target power), both
  absolute power values, and target name/type.
- **Progression:** Triggers `actor.OnStatUse("perception")`.
- **Messaging:** Sends a colored difficulty string to the actor (e.g.,
  "an easy opponent"). Mobs receive no feedback (SendText is a no-op).

---

## Skullduggery Actions (chunk 2.7)

### Sneak

**Function:** `Sneak(actor) SneakResult`

Applies the Hidden buff (ID 9) to the actor after an opposed roll against
all observers in the room.

- **Roll:** `actor Perception + Stealth` vs `each observer's Dexterity +
  Skullduggery`.
- **Success:** Actor gains the Hidden buff; returns `SneakResult.Success =
  true`.
- **Failure:** Actor fails to sneak; returns `Success = false`. (No "you
  were seen" message — the stealth fails silently.)
- **Progression:** Triggers `actor.OnStatUse("perception")` and
  `actor.OnSkillUse("stealth")`.
- **Cooldown:** Shares the `skullduggery` cooldown key (10 rounds, config:
  `SkullduggeryActionCooldown`).

### Steal

**Function:** `Steal(actor, opts) StealResult`

Pickpockets a target mob or player, or robs an item from a room container.

**Three paths:**

1. **Mob pickpocket** (`opts.TargetMobId` set):
   - Rolls `actor Dexterity + Skullduggery` vs `mob Dexterity +
     Skullduggery`.
   - If win: picks a random item from mob inventory.
   - If succeed: returns `StealResult.Success = true` and item ID.
   - If fail: returns `Success = false`.
   - **Messaging:** Always silent on the thief side (no feedback).

2. **Player pickpocket** (`opts.TargetUserId` set):
   - Rolls `actor Dexterity + Skullduggery` vs `player Perception +
     Skullduggery` (note: Perception, not Dex).
   - If win: picks a random item from player inventory.
   - **Detection roll (extra):** If the steal succeeds, rolls `actor
     Dexterity + Skullduggery` vs `player Perception + Skullduggery` again
     to determine if the player **notices**. If player notices, they receive
     "You notice someone trying to pickpocket you!" message. Theft still
     succeeds either way.
   - **Messaging:** Actor always gets silent feedback. Player gets the
     detection message only if the second roll fails.

3. **Container rob** (`opts.RoomContainerId` set):
   - No opposed roll. Opens the container and removes an item.
   - Always succeeds if the container has items.
   - Returns item ID.

**Progression:** Triggers `actor.OnStatUse("dexterity")` and
`actor.OnSkillUse("skullduggery")`.

**Cooldown:** Shares the `skullduggery` cooldown key (10 rounds).

**Result struct:**
```go
type StealResult struct {
	Success    bool
	ItemId     int
	ItemName   string
	Message    string  // feedback message
}
```

### Plant

**Function:** `Plant(actor, opts) PlantResult`

Slips an item from the actor's backpack onto a target mob or into a room
container.

**Two paths:**

1. **Plant on mob** (`opts.TargetMobId` set):
   - Rolls `actor Dexterity + Skullduggery` vs `mob Dexterity +
     Skullduggery`.
   - If win: removes item from actor backpack, adds to mob inventory.
   - If fail: item stays with actor, plant fails.
   - **Messaging:** Actor gets success/failure feedback. Mob (if aware) may
     receive a discovery message on next interaction (not immediate).

2. **Plant in container** (`opts.RoomContainerId` set):
   - No opposed roll. Removes item from actor backpack, adds to container
     inventory.
   - Always succeeds if the container exists.

**Item lookup:** `opts.ItemTag` is a space-separated noun (e.g., "copper
coin"). The function searches actor backpack for a matching item by display
name / simple name.

**Progression:** Triggers `actor.OnStatUse("dexterity")` and
`actor.OnSkillUse("skullduggery")`.

**Cooldown:** Shares the `skullduggery` cooldown key (10 rounds).

**Result struct:**
```go
type PlantResult struct {
	Success bool
	ItemId  int
	Message string
}
```

### Defuse

**Function:** `Defuse(actor, opts) DefuseResult`

Disarms a trap on a room container or exit. Optionally consumes a disarm kit
from the actor's backpack if `opts.UseKit` is true.

**Two paths:**

1. **Container trap** (`opts.ContainerId` set):
   - Finds the container, rolls opposed check: `actor Dexterity +
     Skullduggery` vs `container.TrapDifficulty`.
   - If win: removes the trap (sets `TrapId = 0`). Container is now safe.
   - If fail: actor takes damage (trap detonates). `actor.ApplyHealthChange(-damage)`.

2. **Exit trap** (`opts.ExitName` and `opts.Direction` set):
   - Finds the exit, rolls opposed check: same formula as containers.
   - On success: trap removed.
   - On fail: actor takes damage.

**Disarm kit consumption:** If `opts.UseKit` is true, the function searches
the actor's backpack for an item tagged "disarm kit" and consumes it on
success only. If no kit found, the action proceeds without it.

**Progression:** Triggers `actor.OnStatUse("dexterity")` and
`actor.OnSkillUse("skullduggery")`.

**Cooldown:** No cooldown (can defuse multiple traps per turn).

**Result struct:**
```go
type DefuseResult struct {
	Success          bool
	Message          string
	TrapDetonated    bool  // true if failed and trap triggered
	DamageDealt      int
}
```

### Shadow

**Function:** `Shadow(actor, opts) ShadowResult`

Follow a target while hidden. The actor must already be hidden (carries buff
ID 9) for Shadow to succeed.

**Mechanics:**
- **Prerequisite:** `actor.HasBuff(9)` must be true. If not, returns
  `Success = false`.
- **Target resolution:** `opts.TargetUserId` or `opts.TargetMobId` sets the
  follow target.
- **Storage:** On success, stores the target ID in the actor's misc-data
  under key `"shadow:target"`.
- **Auto-follow:** When the target moves to a new room, the actor's
  auto-follow system (in `modules/follow/`) automatically moves the actor
  with them, maintaining the hidden state.
- **Reveal on attack:** If the hidden actor attacks before Shadow completes,
  the Hidden buff is cancelled and Shadow ends.

**Messaging:** On success, actor receives "You begin stalking [target]." On
failure, "You are not hidden."

**Progression:** No stat/skill use triggered (Shadow is a passive follow
mechanic).

**Cooldown:** No cooldown.

**Result struct:**
```go
type ShadowResult struct {
	Success    bool
	TargetName string
	Message    string
}
```

---

## Available Actions Summary

| Action | Package | Actor→Target | Returns | Messaging | Cooldown |
|--------|---------|---|---|---|---|
| Sneak | actions | self vs room | SneakResult | silent | shared |
| Steal | actions | self vs mob/player/container | StealResult | varies | shared |
| Plant | actions | self vs mob/container | PlantResult | varies | shared |
| Defuse | actions | self vs trap | DefuseResult | varies | none |
| Shadow | actions | self→target | ShadowResult | varies | none |
| Consider | actions | self vs target | ConsiderResult | player only | none |

---

## Options Structs

All chunk 2.7 actions expose `<Verb>Options` structs for caller-side target
structuring:

```go
type SneakOptions struct {
	// No options — Sneak targets self against all observers in room
}

type StealOptions struct {
	TargetMobId       int    // mob to pickpocket
	TargetUserId      string // player to pickpocket
	RoomContainerId   int    // container to rob
	// Only one of the three should be set; first non-zero wins
}

type PlantOptions struct {
	ItemTag           string // noun phrase from command
	TargetMobId       int    // mob to plant on
	RoomContainerId   int    // container to plant in
	// Only one of the two should be set; TargetMobId checked first
}

type DefuseOptions struct {
	ContainerId int    // container with trap
	Direction   string // cardinal (north/south/east/west)
	ExitName    string // friendly exit name
	UseKit      bool   // whether to consume disarm kit
	// ContainerId checked first; if 0, uses Direction+ExitName
}

type ShadowOptions struct {
	TargetUserId string // player to shadow
	TargetMobId  int    // mob to shadow
	// Only one should be set; TargetUserId checked first
}
```

---

## Caller Integration

**User commands** (`internal/usercommands/`): Parse CLI args into
`<Verb>Options`, call the action function, process the result struct.

**Mob commands** (`internal/mobcommands/`): Build options from command args
or script context, call the action function.

**Behavior trees** (`internal/behaviortree/`): BTree action primitives
(`try_sneak`, `try_steal`, etc.) populate options from `EvalContext.Event`
and `mob.Character.Aggro` context, call the action function, return
Success/Failure based on the result.

---

## Cooldown System

Skullduggery actions (Sneak, Steal, Plant) share a single cooldown key
(`"skullduggery"`). Config: `SkullduggeryActionCooldown` (default 10 rounds).

- Tracked in `Character.Cooldowns` map (string → int remaining rounds).
- Cooldowns decrement each round via combat hooks.
- Expired cooldowns are cleaned up lazily when checked.

---

## Dependencies

- `internal/characters` — Character stats, buffs, inventory, cooldowns
- `internal/combat` — Power calculations (Consider)
- `internal/users` — Player character management
- `internal/mobs` — NPC management
- `internal/rooms` — Room context, containers, exits
- `internal/items` — Item specs, damage calculations
- `internal/buffs` — Buff system (Hidden buff for Sneak/Shadow)
- `internal/skills` — Skill progression and names
- `internal/modules/follow` — Auto-follow (used by Shadow)

---
