# Mob/Player Progression & Command Parity

**Date:** 2026-04-04
**Goal:** Eliminate all progression asymmetries between mobs and players so
both entity types advance stats and skills identically through combat and
special moves. Consolidate duplicate commands (howl/taunt), remove deprecated
commands (roar, throw, backstab), and rename the misleading alchemy command.

---

## 1. Actor Interface Expansion

**File:** `internal/actions/actor.go`

Add four progression methods to the Actor interface:

```go
// OnSkillUse triggers skill progression (and the skill's governing stat).
// UserActor fires events.SkillUsed{}; MobActor calls Character.OnSkillUse directly.
OnSkillUse(skillName string) bool

// OnStatUse triggers stat progression.
// UserActor calls Character.OnStatUse(stat, userId); MobActor uses userId=0.
OnStatUse(statName string) bool

// OnCriticalSuccess records a critical hit for progression bonuses.
OnCriticalSuccess(skillName string)

// OnCriticalFailure records a fumble for progression tracking.
OnCriticalFailure(skillName string)
```

### Implementation

**Key finding:** `Character.OnSkillUse(skillName, userId)` already fires
`events.SkillUsed{}` internally when `userId > 0` (line 234 of
progression.go). The existing wrapper-level `events.AddToQueue` calls in
usercommands are **redundant duplicates**. This simplifies the Actor
implementation — both UserActor and MobActor just delegate to the Character
method with the appropriate userId.

**`actor_user.go` (UserActor):**
- `OnSkillUse(skill)` → `char.OnSkillUse(skill, userId)` (fires quest
  event internally via userId > 0)
- `OnStatUse(stat)` → `char.OnStatUse(stat, userId)`
- `OnCriticalSuccess(skill)` → `char.OnCriticalSuccess(skill, userId)`
- `OnCriticalFailure(skill)` → `char.OnCriticalFailure(skill, userId)`

**`actor_mob.go` (MobActor):**
- `OnSkillUse(skill)` → `char.OnSkillUse(skill, 0)` (userId=0 skips event)
- `OnStatUse(stat)` → `char.OnStatUse(stat, 0)`
- `OnCriticalSuccess(skill)` → `char.OnCriticalSuccess(skill, 0)`
- `OnCriticalFailure(skill)` → `char.OnCriticalFailure(skill, 0)`

---

## 2. Combat Loop Parity

**File:** `internal/combat/combat.go`

### Problem

`AttackPlayerVsMob` (line 34) and `AttackPlayerVsPlayer` (line 82) call
OnStatUse(strength), OnStatUse(dexterity), OnSkillUse(combatSkill),
OnCriticalSuccess, and OnCriticalFailure for the attacker.

`AttackMobVsPlayer` (line 131) only tracks the *defender's* dexterity.
`AttackMobVsMob` (line 161) tracks nothing at all.

### Fix

Add matching progression calls to both mob-attacker functions. Since
combat.go doesn't use the Actor interface (it takes raw `*mobs.Mob` /
`*users.UserRecord`), we create a thin helper that wraps the mob into a
MobActor for the progression calls:

```go
// trackMobAttackProgression mirrors the player progression in
// AttackPlayerVsMob/AttackPlayerVsPlayer for a mob attacker.
func trackMobAttackProgression(mob *mobs.Mob, room *rooms.Room, result AttackResult) {
    actor := &actions.MobActor{Mob: mob, Room: room}
    actor.OnStatUse("strength")
    actor.OnStatUse("dexterity")
    if result.Hit {
        combatSkill := string(mob.Character.GetCombatSkillTag())
        actor.OnSkillUse(combatSkill)
        if result.Crit {
            actor.OnCriticalSuccess(combatSkill)
        }
        // Dual-wield weapon-combat tracking (same logic as player path)
        if mob.Character.Equipment.Weapon.ItemId > 0 &&
           mob.Character.Equipment.Offhand.ItemId > 0 &&
           mob.Character.Equipment.Offhand.GetSpec().Type == items.Weapon {
            actor.OnSkillUse(string(skills.WeaponCombat))
        }
    } else if result.Fumble {
        combatSkill := string(mob.Character.GetCombatSkillTag())
        actor.OnCriticalFailure(combatSkill)
    }
}
```

**AttackMobVsPlayer:** Add `trackMobAttackProgression(mob, room, attackResult)`
after the existing defender dexterity tracking (line 151).

**AttackMobVsMob:** Add `trackMobAttackProgression(mobAtk, room, attackResult)`
after the charmed-user tracking block (line 179). Also add defender dexterity
tracking for the defending mob (mirrors what AttackMobVsPlayer does for
player defenders).

Note: `trackMobAttackProgression` needs to load the room (or accept it as a
parameter). Since both attack functions already load the room into `room`
variable, pass it through.

---

## 3. Shared Action Progression Migration

Move OnSkillUse calls from user/mob wrappers INTO the shared ExecuteX
functions via the Actor interface.

### 3a. ExecuteBash (`internal/actions/combat_bash.go`)

Add after the RecordAndWait call (line 100):

```go
if result.Hit {
    actor.OnSkillUse(string(skills.WeaponCombat))
}
```

**Remove from:**
- `usercommands/bash.go` — the `events.AddToQueue(events.SkillUsed{...})`
- `mobcommands/bash.go` — the `mob.Character.OnSkillUse(...)` call

### 3b. ExecuteKick (`internal/actions/combat_kick.go`)

Add after RecordAndWait:

```go
if result.Hit {
    actor.OnSkillUse(string(skills.UnarmedCombat))
}
```

**Remove from:** `usercommands/kick.go` and `mobcommands/kick.go`.

### 3c. ExecuteTrip (`internal/actions/combat_trip.go`)

Same pattern — add `actor.OnSkillUse(string(skills.UnarmedCombat))` on hit.

**Remove from:** `usercommands/trip.go` and `mobcommands/trip.go`.

### 3d. ExecuteGrapple (`internal/actions/combat_grapple.go`)

Same pattern — add `actor.OnSkillUse(string(skills.UnarmedCombat))` on hit.

**Remove from:** `usercommands/grapple.go` and `mobcommands/grapple.go`.

### 3e. InitiateCast (`internal/actions/cast.go`)

Add `actor.OnSkillUse(castSkillName)` after successful cast initiation,
where `castSkillName` is "spellcasting" or "manifestation" depending on
spell type. The Actor parameter is already passed to InitiateCast.

**Remove from:** `usercommands/skill.cast.go` — the
`events.AddToQueue(events.SkillUsed{...})` block.

Result: mob casting now gets identical progression to player casting.

---

## 4. New Shared Actions

### 4a. ExecuteTaunt (`internal/actions/combat_taunt.go`)

Unify `usercommands/taunt.go` and `mobcommands/howl.go` into a shared
conviction-damage action.

**Signature:**
```go
type TauntResult struct {
    Target    AggroTarget
    Executed  bool
    OnCooldown bool
    NoTarget  bool
    Hit       bool
    Crit      bool
    Fumble    bool
    Damage    int
    DmgDesc   string
}

func ExecuteTaunt(actor Actor) TauntResult
```

**Core mechanics (identical to existing taunt.go):**
1. Check aggro exists → `NoTarget`
2. Check special-move cooldown → `OnCooldown`
3. Resolve target via `ResolveAggroTarget`
4. Conviction attack: Charisma + rhetoric (weighted by SkillWeight) vs
   Willpower + rhetoric
5. Apply conviction depletion penalty via `ResourceMultiplier`
6. Opposed roll via `dice.OpposedRollStat`
7. Fumble (ZScore <= -2.0): self-conviction damage (Charisma/10, min 1)
8. Hit: `CalcRawDamage(Charisma, rhetoric, TauntBaseMult, ChannelConviction)`
   with conviction depletion multiplier
9. Crit (ZScore >= 2.0): bypass mitigation
10. Normal hit: apply `GetConvictionMitigation()` + `MitigationCap`
11. Apply damage to defender's conviction pool
12. `actor.OnSkillUse("rhetoric")` — automatic parity via Actor interface
13. RecordAndWait with analytics
14. Set `Aggro.RoundsWaiting = 1`

**Config:** Uses existing `TauntBaseMult` (0.5) for both player taunt and
mob howl. Single knob.

**Callers:**

`usercommands/taunt.go` becomes a thin wrapper:
- Handle out-of-combat target resolution (player-only: "Taunt whom?")
- Call `ExecuteTaunt(actor)`
- Call `sendTauntMessages(...)` based on result (player-facing messages)
- `sendTauntMessages` stays in usercommands — it handles the data-driven
  YAML taunt message system with player-specific formatting

`mobcommands/howl.go` becomes a thin wrapper:
- Call `ExecuteTaunt(actor)`
- Send mob-flavored messages (howl text, darkness-aware)
- The mob command stays registered as `"howl"` so existing mob YAML
  `scriptcommands` that reference `howl` keep working

### 4b. ExecuteBite (`internal/actions/combat_bite.go`)

Move bite from `mobcommands/bite.go` into a shared action for future player
use (species-gated ability).

**Signature:**
```go
type BiteResult struct {
    Target     AggroTarget
    MoveResult combat.SkillMoveResult
    Executed   bool
    OnCooldown bool
    NoTarget   bool
    DrainAmount int  // HP drained on hit
}

func ExecuteBite(actor Actor) BiteResult
```

**Core mechanics (from existing bite.go):**
1. Check aggro, cooldown, target resolution (same pattern as ExecuteBash)
2. `ExecuteSkillMove` with UnarmedCombat, 60% damage
3. On hit: drain 50% of damage as healing (cap at max HP)
4. `actor.OnSkillUse(string(skills.UnarmedCombat))` on hit
5. RecordAndWait

**Callers:**
- `mobcommands/bite.go` — thin wrapper, mob-flavored messages
- Future: `usercommands/bite.go` — player wrapper (not in this spec)

### 4c. ExecuteHamstring (`internal/actions/combat_hamstring.go`)

Move hamstring from `mobcommands/hamstring.go` into a shared action.

**Signature:**
```go
type HamstringResult struct {
    Target     AggroTarget
    MoveResult combat.SkillMoveResult
    Executed   bool
    OnCooldown bool
    NoTarget   bool
    BleedDmg   int  // per-tick bleed damage applied
}

func ExecuteHamstring(actor Actor) HamstringResult
```

**Core mechanics (from existing hamstring.go):**
1. Check aggro, cooldown, target resolution
2. `ExecuteSkillMove` with Dexterity attack, UnarmedCombat skill,
   TripDamagePercent damage, no knockdown
3. On hit: apply ConditionBleeding (duration 5, magnitude Strength/10 min 2)
4. `actor.OnSkillUse(string(skills.UnarmedCombat))` on hit
5. RecordSpecialMove for analytics

**Callers:**
- `mobcommands/hamstring.go` — thin wrapper, mob-flavored messages
- Future: `usercommands/hamstring.go` — player wrapper (not in this spec)

---

## 5. Command Removals

### 5a. Delete `mobcommands/roar.go`

Roar is a copy-paste of howl with different flavor text. With howl now
calling the shared `ExecuteTaunt`, roar is redundant.

- Delete `internal/mobcommands/roar.go`
- Remove `"roar": {Roar, false}` from command registry in `mobcommands.go`
- Remove `"roar"` entry from `mobOnlyCommands` in `divergences.go`
- No dogmud mob YAML references `roar` in combat scripts (only in prose
  descriptions which are unaffected)

### 5b. Delete `mobcommands/throw.go`

- Delete `internal/mobcommands/throw.go`
- Remove from command registry
- Remove from `mobOnlyCommands` in `divergences.go`
- No dogmud mob YAML references `throw` in combat scripts

### 5c. Delete `mobcommands/backstab.go`

- Delete `internal/mobcommands/backstab.go`
- Remove from command registry
- Remove from `mobOnlyCommands` in `divergences.go`

**Data file impact:** The following mobs reference `backstab` in their
scripts:
- `world/dogmud/mobs/thornwall_city/scripts/272-chrysalis_phantom.js`
  (line 37: `mob.Command('backstab')`)
- `world/default/mobs/mirror_caves/24-cave_stalker.yaml` (scriptcommands)
- `world/default/mobs/frostfang_slums/41-shadow_trainee.yaml`
- `world/default/mobs/frostfang_slums/30-shadow_master.yaml`

**Fix:** Replace `backstab` with `attack` in the chrysalis phantom JS
script. The `world/default/` mobs are upstream GoMud content — we don't
modify those; they'll get a harmless "unknown command" log when backstab
fires and fall through to their normal attack cycle.

### 5d. Keywords cleanup

Remove `backstab` and `throw` aliases from
`_datafiles/world/dogmud/keywords.yaml` (lines 222, 229). Leave
`world/default/` and `world/empty/` keywords alone (upstream).

---

## 6. Rename: alchemy -> selljunk

**File:** `internal/mobcommands/alchemy.go` → `internal/mobcommands/selljunk.go`

- Rename file
- Rename function `Alchemy` → `Selljunk`
- Update command registry: `"selljunk": {Selljunk, false}`
- Update `mobOnlyCommands` in `divergences.go`:
  `"selljunk": "mob-ai: converts inventory items to gold"`
- Remove old `"alchemy"` entries from both maps

No mob YAML in dogmud references the `alchemy` command in combat scripts.

---

## 7. Divergences.go Updates

After all changes, `mobOnlyCommands` should reflect the new state:

**Remove:**
- `"backstab"` — deleted
- `"roar"` — deleted
- `"throw"` — deleted
- `"alchemy"` — renamed
- `"howl"` future-work comment — now uses shared taunt

**Update:**
- `"howl": "mob-ai: shared taunt with mob flavor text"` — stays mob-only
  in registry but calls shared ExecuteTaunt
- `"selljunk": "mob-ai: converts inventory items to gold"` — renamed from
  alchemy

**Add (if not present):**
- `"bite"` and `"hamstring"` keep their `"mob-ai"` category but note they
  now use shared actions (future player abilities)

---

## 8. Files Changed Summary

### New files
| File | Purpose |
|------|---------|
| `internal/actions/combat_taunt.go` | Shared ExecuteTaunt |
| `internal/actions/combat_bite.go` | Shared ExecuteBite |
| `internal/actions/combat_hamstring.go` | Shared ExecuteHamstring |
| `internal/mobcommands/selljunk.go` | Renamed from alchemy.go |

### Modified files
| File | Changes |
|------|---------|
| `internal/actions/actor.go` | Add 4 progression methods to interface |
| `internal/actions/actor_user.go` | Implement progression methods |
| `internal/actions/actor_mob.go` | Implement progression methods |
| `internal/actions/combat_bash.go` | Add actor.OnSkillUse on hit |
| `internal/actions/combat_kick.go` | Add actor.OnSkillUse on hit |
| `internal/actions/combat_trip.go` | Add actor.OnSkillUse on hit |
| `internal/actions/combat_grapple.go` | Add actor.OnSkillUse on hit |
| `internal/actions/cast.go` | Add actor.OnSkillUse on successful cast |
| `internal/combat/combat.go` | Add trackMobAttackProgression helper; call in AttackMobVsPlayer + AttackMobVsMob |
| `internal/usercommands/taunt.go` | Thin wrapper calling ExecuteTaunt |
| `internal/usercommands/bash.go` | Remove OnSkillUse call |
| `internal/usercommands/kick.go` | Remove OnSkillUse call |
| `internal/usercommands/trip.go` | Remove OnSkillUse call |
| `internal/usercommands/grapple.go` | Remove OnSkillUse call |
| `internal/usercommands/skill.cast.go` | Remove OnSkillUse call |
| `internal/mobcommands/howl.go` | Thin wrapper calling ExecuteTaunt |
| `internal/mobcommands/bite.go` | Thin wrapper calling ExecuteBite |
| `internal/mobcommands/hamstring.go` | Thin wrapper calling ExecuteHamstring |
| `internal/mobcommands/cast.go` | Remove (progression now in shared layer) |
| `internal/mobcommands/mobcommands.go` | Update registry (remove roar/throw/backstab, rename alchemy→selljunk) |
| `internal/actions/divergences.go` | Update both allowlists |
| `_datafiles/world/dogmud/mobs/thornwall_city/scripts/272-chrysalis_phantom.js` | backstab → attack |
| `_datafiles/world/dogmud/keywords.yaml` | Remove backstab/throw aliases |

### Deleted files
| File | Reason |
|------|--------|
| `internal/mobcommands/roar.go` | Redundant with shared taunt |
| `internal/mobcommands/throw.go` | Deprecated, future player ability |
| `internal/mobcommands/backstab.go` | Redundant with surprise strike |
| `internal/mobcommands/alchemy.go` | Renamed to selljunk.go |

---

## 9. Testing Strategy

1. **Compile check** — `go build ./...` must pass with no errors
2. **Unit tests** — `go test ./internal/actions/...` and
   `go test ./internal/combat/...` must pass
3. **Manual smoke test:**
   - Mob in combat with player: verify mob stats/skills advance after
     basic attacks (check via admin `mob` command or combatstats)
   - Mob using howl: verify conviction damage + rhetoric progression
   - Mob using bite: verify life drain + unarmed-combat progression
   - Mob using hamstring: verify bleed + unarmed-combat progression
   - Player using bash/kick/trip/grapple/taunt/cast: verify progression
     still works (no regressions from removing wrapper-level calls)
   - Wolf mobs with `howl` in scriptcommands: verify they still use it
   - Chrysalis phantom: verify it attacks instead of backstabbing
   - `selljunk` command: verify mobs can still convert items to gold
4. **Parity audit** — run server startup, confirm `AuditCommandParity`
   logs no unexpected warnings
