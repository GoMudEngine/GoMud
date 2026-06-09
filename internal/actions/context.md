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

### Special-Move Actions & Anatomy Gating (Phase 2)

The shared special-move actions (`ExecuteGrapple`, `ExecuteBash`,
`ExecuteTrip`, `ExecuteKick`, `ExecuteHamstring`) each carry a
defense-in-depth anatomy guard mirroring the AI `CanUse*` gate in
`internal/combat/ai.go` and the parity check in `command_readiness.go`:
grapple/submit need `arms`, trip/kick need `legs`, bash needs
`(shield|NaturalBash) AND (arms|NaturalBash)`. These three sites form a
`// SYNC POINT` triad — change one, change all; `command_readiness_drift_test.go`
asserts the gate and `CommandIsReady` stay in agreement (the `*_no_arms`
/`*_no_legs` rows). The action-entry guard is unreachable for players
(always humanoid) and reuses the nearest existing failure flag
(`GrappleImmune`, `NoShield`, `NoTarget`) rather than minting a new one.

**Retired:** `ExecuteBite`/`BiteResult` were removed — biting is now the
Phase-1 basic attack for fanged species (species `NaturalAttack`).
`mobOnlyCommands` no longer lists `bite`. `toxic-bite` (mutation) stays.

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

### Scan

**Function:** `Scan(actor, opts) ScanResult`

Sweeps adjacent rooms for visible entities (non-hidden mobs/players).

**Mechanics:**
- **Adjacent rooms:** Scans in all four cardinal directions; lists any
  non-hidden mobs and players in the returned room descriptions.
- **Visibility:** Does not bypass hidden state — only visible entities are
  reported. Mobs/players who are hidden (buff 9) are not seen.
- **UserActor behavior:** Renders a "You sense:" list of adjacent-room
  entities with flavor text.
- **MobActor behavior:** Silent (no feedback).
- **Hostile-only mode:** `opts.HostileOnly = true` filters results to only
  entities the actor hates.

**Messaging:** UserActor receives "You sense: [adjacent rooms with entities]."
MobActor silent.

**Progression:** No stat/skill use triggered.

**Cooldown:** No cooldown.

**Result struct:**
```go
type ScanResult struct {
	Success        bool
	SightingFound  bool  // true if at least one entity seen
	Message        string
}
```

### Search

**Function:** `Search(actor, opts) SearchResult`

Three-tier discovery system: exits, stashed/hidden objects, and nouns.

**Mechanics:**
- **Tier 1 (Exits):** Lists all room exits (always succeeds for UserActor).
- **Tier 2 (Stashed items):** Searches containers and ground for hidden items.
- **Tier 3 (Hidden entities):** Detects hidden mobs/players in the room.
  Promoted to `ctx.SoftTarget` if hostile.
- **Ignores non-hostile Tier-3 hits:** If a hidden entity is not hostile,
  it is not set as SoftTarget.
- **UserActor behavior:** Renders progressive search feedback (what tiers
  were discovered, what was found).
- **MobActor behavior:** Silent; just seeds SoftTarget if hostile found.

**Messaging:** UserActor receives discovery feedback per tier. MobActor silent.

**Progression:** No stat/skill use triggered.

**Cooldown:** Shares the `search` key (configurable duration, typically
  2 rounds).

**Result struct:**
```go
type SearchResult struct {
	Success         bool
	HiddenHostile   bool  // true if hidden entity promoted to SoftTarget
	Message         string
}
```

---

## Sleep Action (chunk 3.3)

### Sleep

**Function:** `Sleep(actor, opts) SleepResult`

Applies buff 15 (Sleeping) to the actor. Combat-gated: fails if the
actor is currently in combat (`Aggro != nil`). Idempotent: if the
actor already has the Sleeping buff, returns `SleepResult.AlreadyAsleep
= true` with no additional buff applied.

- **Success:** Actor receives buff 15; returns `SleepResult.Success = true`.
- **Failure (combat):** Returns `Success = false` with messaging.
- **Messaging:** UserActor receives a "You lie down and close your eyes."
  message; the room sees "<Actor> lies down to sleep." MobActor messaging
  goes to the room only.
- **Progression:** No stat/skill progression triggered.
- **Cooldown:** None.

Entry points that call `Sleep`:
- `usercommands/sleep.go` — player `sleep` command
- `mobcommands/sleep.go` — mob `sleep` command
- `hooks/NewRound_IdleMobs_schedule.go` — schedule executor for
  `activity: sleeping` segments

**Result struct:**
```go
type SleepResult struct {
    Success       bool
    AlreadyAsleep bool
    Message       string
}
```

---

## Foraging & Salvage Actions (chunk 2.9)

### Forage

**Function:** `Forage(actor, opts) ForageResult`

Single forage attempt in the current room's biome. Gated by biome availability
and shared cooldown.

**Mechanics:**
- **Biome check:** Actor must have forager profile data for the room's biome.
  Returns Failure if biome not found in profile (wrong terrain type).
- **Cooldown:** Shares the `forage` key (6 rounds, config:
  `ForageActionCooldown`).
- **Item discovery:** On success, generates item based on biome's forage table.
  Returns `ForageResult.ItemId` and `ItemName`. On miss (roll failure) or
  cooldown, returns Failure.
- **Progression:** Triggers `actor.OnSkillUse("foraging")`.

**Result struct:**
```go
type ForageResult struct {
	Success   bool
	ItemId    int
	ItemName  string
	Message   string
}
```

### Salvage

**Function:** `Salvage(actor, opts) SalvageResult`

Single-tick salvage attempt. Modes: default targets first eligible corpse
(mob death items in room), optional `ItemUuid` targets a specific item.

**Mechanics:**
- **Corpse mode** (`opts.ItemUuid` empty): Scans room for corpse items
  (items with `on_corpse: true`). Salvages the first match. Returns Failure
  if no corpse items found.
- **Item mode** (`opts.ItemUuid` set): Salvages the specified item UUID
  directly (player per-tick invocation path).
- **Multi-round activity:** Salvage takes 1-5 rounds depending on ingredient
  gold value. Each ingredient is rolled independently per the skill-based
  recovery table.
- **Progression:** Triggers `actor.OnStatUse("perception")` and
  `actor.OnSkillUse("salvage")`.
- **Result:** Returns `SalvageResult` with success flag and recovered
  materials list (empty if roll failures on all ingredients).

**Result struct:**
```go
type SalvageResult struct {
	Success     bool
	Message     string
	RecoveredCount int  // number of materials recovered
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
  under key `"shadow-target-mob"` or `"shadow-target-user"` depending on
  target type. Also applies buff 87 (Shadow status buff).
- **Auto-follow:** When the target moves to a new room, the actor's
  auto-follow system (in `modules/follow/`) automatically moves the actor
  with them if the actor carries buff 87 (`HasBuff(87)` gating in
  `usercommands/go.go`), maintaining the hidden state.
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

### Track

**Function:** `Track(actor, opts) TrackResult`

Trail-read (passive sniffing) or active tracking on a resolved target.

**Mechanics:**
- **Trail-sniff (no-arg):** `opts.TargetNoun` empty; reads the room's trail
  data (bloodstain, scent, tracks left by previous passage). Returns success
  if a trail exists; failure if none.
- **Active track (target noun):** `opts.TargetNoun` or target from Event/Aggro;
  enters tracking mode on the target. On success (adjacent trail/recent
  sighting), applies buff 86 (Track status) and misc-data pair
  (`tracking-<userId>` or `tracking-<mobId>` with arrival timestamp).
  Seeds `ctx.SoftTarget` for downstream scout actions.
- **UserActor behavior:** Renders trail-sniff results or tracking status.
- **MobActor behavior:** Silent; just applies buff/misc-data and seeds
  SoftTarget.

**Messaging:** UserActor receives trail feedback or tracking status. MobActor
silent.

**Progression:** No stat/skill use triggered (scout actions planned as
  skill-less in Phase 1).

**Cooldown:** Shares the `search` key (typically 2 rounds).

**Result struct:**
```go
type TrackResult struct {
	Success        bool
	TrailFound     bool  // true if trail exists (sniff) or track active (active)
	TargetName     string
	Message        string
}
```

---

## Sell Action (chunk 5.4)

### Sell

**Function:** `Sell(seller Actor, opts SellOptions) SellResult`

Shared seller entry point for players and mobs. Resolves the first willing
merchant in the seller's room, then sells matching items.

**Two sell models — important distinction:**

1. **SALE (gold transfer) via `actions.Sell`**: The merchant evaluates the
   item via `EvaluateBuyRules` / `GetSellPrice`, returns an offer price, and
   the item transfers to the shop's stock. For **player** sellers the merchant's
   gold pool is drawn down by the sale price (`shopInv.Gold -= sellValue` or
   `mob.Character.Gold -= sellValue`), and a merchant-broke gate prevents
   sales the merchant can't afford. For **mob** sellers the seller is credited
   (`char.Gold += sellValue`) but **shop gold is not touched** — mob sales
   mint gold without bankrupting the shop (gated on `seller.IsPlayer()`).

2. **SUPPLY HANDOFF (free, no gold) via `forager.SellToVendor` and
   `forager.BackfillVendorFromChests`**: Forager delivery and chest backfill
   transfer items directly into vendor stock with no price computation and no
   gold movement. These are supply-side operations, not market transactions.

**SellAllSellable mode** (`opts.SellAllSellable = true`): Mob inventory-sweep.
Iterates every item in the seller's backpack that is not a quest token, not a
crafting material, and has `Value > 0`; calls `sellOneToMerchant` for each.
Used by the goal planner's wealth-gold sell step.

**Options struct:**
```go
type SellOptions struct {
    ItemName        string // ignored when SellAllSellable
    Quantity        int    // 1, N, or UnlimitedSell (math.MaxInt)
    SellAllSellable bool   // mob inventory-sweep mode
    MerchantName    string // optional target; "" = first willing merchant
}
```

**Result struct:**
```go
type SellResult struct {
    Sold         int
    TotalGold    int
    Reason       SellStopReason
    LastItemName string
}
```

`SellStopReason` values: `SellStopSoldAll` (normal), `SellStopNoItem`,
`SellStopNoMerchant`, `SellStopMerchantBroke` (player path only),
`SellStopRejected`.

**Messaging:** Player sellers receive "You don't have that item." and merchant
speech lines synchronously (not via the mob's async command queue — see
`merchantSay` in `sell.go`). Mob sellers receive no feedback text.

**Progression:** Triggers `seller.OnSkillUse("bartering")` and
`mob.Character.OnStatUse("charisma")` on each successful sale.

Entry points that call `Sell`:
- `usercommands/sell.go` — player `sell` command
- `mobcommands/sell.go` — mob `sell` command
- `internal/planners/` — goal planner's wealth-gold save-up sell step

**See also:** `internal/forager/vendor_sell.go` (`forager.SellToVendor`) and
`internal/forager/chest_backfill.go` (`forager.BackfillVendorFromChests`) for
the free supply-handoff paths — these are NOT routed through `actions.Sell`.

---

## Available Actions Summary

| Action | Package | Actor→Target | Returns | Messaging | Cooldown |
|--------|---------|---|---|---|---|
| Consider | actions | self vs target | ConsiderResult | player only | none |
| Defuse | actions | self vs trap | DefuseResult | varies | none |
| Forage | actions | self vs biome | ForageResult | varies | shared |
| Plant | actions | self vs mob/container | PlantResult | varies | shared |
| Salvage | actions | self vs corpse/item | SalvageResult | varies | none |
| Scan | actions | self → adjacent | ScanResult | user only | none |
| Search | actions | self vs room | SearchResult | user only | shared |
| Shadow | actions | self→target | ShadowResult | varies | none |
| Sneak | actions | self vs room | SneakResult | silent | shared |
| Steal | actions | self vs mob/player/container | StealResult | varies | shared |
| Sell | actions | self vs merchant | SellResult | player only | none |
| Sleep | actions | self | SleepResult | varies | none |
| Track | actions | self vs trail/target | TrackResult | user only | shared |

---

## Options Structs

All chunk 2.7+ actions expose `<Verb>Options` structs for caller-side target
structuring:

```go
type DefuseOptions struct {
	ContainerId int    // container with trap
	Direction   string // cardinal (north/south/east/west)
	ExitName    string // friendly exit name
	UseKit      bool   // whether to consume disarm kit
	// ContainerId checked first; if 0, uses Direction+ExitName
}

type PlantOptions struct {
	ItemTag           string // noun phrase from command
	TargetMobId       int    // mob to plant on
	RoomContainerId   int    // container to plant in
	// Only one of the two should be set; TargetMobId checked first
}

type ScanOptions struct {
	HostileOnly bool // if true, only return entities actor hates
}

type SearchOptions struct {
	// No options — Search discovers all tiers in the room
}

type ShadowOptions struct {
	TargetUserId string // player to shadow
	TargetMobId  int    // mob to shadow
	// Only one should be set; TargetUserId checked first
}

type SneakOptions struct {
	// No options — Sneak targets self against all observers in room
}

type StealOptions struct {
	TargetMobId       int    // mob to pickpocket
	TargetUserId      string // player to pickpocket
	RoomContainerId   int    // container to rob
	// Only one of the three should be set; first non-zero wins
}

type TrackOptions struct {
	TargetNoun string // optional target noun for active track
	// Empty = trail-sniff; non-empty = active track on resolved target
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
