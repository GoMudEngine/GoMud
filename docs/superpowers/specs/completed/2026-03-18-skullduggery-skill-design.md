# Skullduggery Skill Consolidation Design

**Date:** 2026-03-18
**Status:** Draft

---

## Overview

Consolidate the existing `stealth` skill and its sub-commands (sneak,
pickpocket) plus the mob-only backstab into a single **skullduggery**
skill with seven sub-commands. All sub-commands progress the one skill.
All roll-based checks use `Dex + (SkillMultiplier(rank) * 25)` scaled
for 100-baseline stats.

**Governing stat:** Dexterity

**PvP policy:** Steal and plant are NPC/mob/container only. The old
pickpocket PvP code is intentionally removed. Players cannot steal
from or plant items on other players.

---

## Sub-Commands

| Command | Description | Skill Gate | Roll-Based |
|---------|-------------|------------|------------|
| sneak | Enter hidden state | Rank 0+ | Yes (opposed) |
| steal | Take from NPCs/containers | Rank 1+ | Yes (opposed) |
| plant | Slip items onto NPCs/containers | Rank 1+ | Yes (opposed) |
| picklock | Existing lock minigame | Rank 1+ | No (minigame) |
| shadow | Follow target between rooms while hidden | Rank 2+ | Yes (opposed) |
| surprise attack | Auto-triggered bonus hit from stealth | Rank 0+ | No (auto) |
| defuse | Disable traps (stub) | Rank 3+ | Future |

---

## Skill Progression

All opposed rolls (sneak, steal, plant, shadow room-entry, shadow
target detection) trigger `CheckSkillProgression` on completion,
regardless of success or failure. This matches the search skill
pattern where any roll against something counts as skill use.

Picklock triggers progression on successful lock completion only.

Surprise attack triggers progression when it fires.

---

## Section 1: Sneak

### Current Behavior
Type `sneak` in a calm room -> auto-grants hidden buff (ID 9). No
roll, no counterplay, no feedback message.

### New Behavior

**Empty room:** Auto-success. Grants hidden buff.
Message: "You slip into the shadows."

**Room with observers:** Opposed roll per hostile/neutral observer:
- Attacker: `Dex + (SkillMultiplier(skullduggeryRank) * 25)`
- Defender: `observer's Perception + (SkillMultiplier(searchRank) * 25)`
- Uses `dice.OpposedRollStat()` for each observer
- Party members excluded from observer list

**Must beat every hostile/neutral observer** to enter hidden state.
If any observer wins:
- Sneaker sees: "You try to blend into the shadows but \<name\>
  notices you."
- Observer (if player) sees: "\<username\> tries to hide but you
  notice them."
- No room broadcast -- others who didn't spot you don't know.

**No calm-room restriction.** Can attempt anywhere. Mid-combat rooms
are naturally harder (more alert observers).

**Cooldown:** On failure only. `SneakFailCooldown` rounds (default 3)
before the player can attempt sneak again. Uses
`TryCooldown("skullduggery:sneak", ...)`. On success, no cooldown --
player can sneak again immediately (though they're already hidden).

### Stealth Detection on Room Entry

Implemented in `go.go` movement handler. After destination room is
determined but before the move completes:

1. Enumerate all hostile/neutral NPCs and non-party players in the
   destination room
2. For each observer: `dice.OpposedRollStat(sneakerScore, observerScore)`
3. If **any observer wins**: hidden buff drops, player arrives
   normally with messages:
   - Sneaker: "You slip into the room but \<name\> notices you."
   - Spotter (if player): "\<username\> slips into the room but
     you notice them."
   - Room: normal "\<username\> arrives." broadcast
4. If **all observers fail**: invisible arrival, no messages to
   anyone in the destination room

### Detection When Others Enter Your Room

When a new NPC/player enters a room where hidden players are present,
the newcomer rolls against each hidden player (same opposed roll).
- Newcomer fails: they don't notice the hidden player.
- Newcomer succeeds: hidden player is revealed. Both parties get
  appropriate messages.

Implemented in the same `go.go` movement handler, after the mover
arrives in the room.

### Hidden as a Condition

Hidden status shows in the `condition` command output via the existing
buff system (buff ID 9).

---

## Section 2: Surprise Attack

### Trigger

Automatic. When a hidden player initiates combat (`attack <target>`):
1. Player has the hidden buff
2. `special-move` cooldown is available

Surprise attack fires as an **extra hit** on top of the normal combat
round. Not a replacement -- bonus damage.

### Implementation Location

The surprise attack check and execution happens in the `attack`
command handler (`attack.go`), BEFORE combat begins and BEFORE the
hidden buff's `cancel-on-combat` flag fires. Sequence:

1. Player types `attack <target>`
2. `attack.go` checks: is player hidden? Is special-move cooldown
   available?
3. If yes: execute surprise attack (extra crit swings, consume
   special-move cooldown)
4. Then: initiate normal combat (which triggers `cancel-on-combat`
   and removes the hidden buff)
5. Normal combat round proceeds

This avoids the sequencing problem entirely -- the surprise attack
fires in the attack command, not in the combat round loop.

### Damage

Swings all equipped weapons in sequence (same weapon enumeration as
normal combat). Each swing:
- Auto-crit (same crit behavior as existing backstab)
- Damage multiplied by:
  `surpriseMultiplier = max(1.0, (Dexterity + skillRank) / 100)`
- Stacking hit penalties per weapon (config-driven):

| Weapon | Config Key | Default |
|--------|-----------|---------|
| Primary | (none) | 0% |
| Offhand | `SurpriseAttackOffhandPenalty` | 0.10 |
| Extra arm 1 | `SurpriseAttackExtraArm1Penalty` | 0.25 |
| Extra arm 2 | `SurpriseAttackExtraArm2Penalty` | 0.40 |

### Scaling Examples

| Dex | Skill Rank | Multiplier |
|-----|-----------|------------|
| 100 | 0 | 1.00 |
| 115 | 15 | 1.30 |
| 130 | 30 | 1.60 |
| 140 | 40 | 1.80 |

### Cooldown

Consumes the shared `special-move` cooldown (same slot as bash, kick,
trip, cast).

### Party Auto-Assist

When the initiator's surprise attack triggers, all hidden party
members in the same room who also have `special-move` off cooldown
get their own surprise attacks on the same target. Each member rolls
independently (own weapons, own multiplier, own hit penalties).

The auto-assist system must be explicitly verified to work with
surprise attack triggering.

### Messages

Attacker: `*[SURPRISE ATTACK]* You strike <target> from the shadows!`
Room: `*[SURPRISE ATTACK]* <username> strikes <target> from the
shadows!`

Crit flavor text from weapon-type combat messages applies on top.

### Mob Support

Mobs with skullduggery can also surprise attack. Replaces the old
`backstab` mob command. `characters.BackStab` aggro type renamed to
`characters.SurpriseAttack`. No YAML or script changes needed -- aggro
types are not serialized in data files. Check for any mob AI scripts
that reference the `backstab` command and update them.

---

## Section 3: Steal

**Usage:** `steal <target>` (NPC/mob) or `steal <container>`

**NPC/mob and container only.** Cannot steal from players. Attempting
to steal from a player: "You can't steal from other players." The old
pickpocket PvP code is intentionally removed.

### Roll Formula

- Attacker: `(Dex + (SkillMultiplier(skullduggeryRank) * 25)) * StealSkillMultiplier`
- Hidden bonus: `+StealHiddenBonus` (config, default 25) added to
  attacker score if sneaking
- Config: `StealSkillMultiplier` (default 1.0) for tuning

### Creature Steal

- Opposed roll: attacker score vs target's Perception
- Success: steal gold (random portion) and/or a random item (same
  loot logic as current pickpocket)
- Failure: target notices, hidden buff cancelled, target attacks
- Must not be in combat

### Container Steal

- Opposed roll: attacker score vs highest Perception among
  hostile/neutral observers in room
- Empty room: auto-success
- Success: take an item without anyone noticing
- Failure: observers notice, hidden buff cancelled

### Error Messages

- No target: "Steal from whom?"
- Target is player: "You can't steal from other players."
- In combat: "You can't do that while in combat!"

### Cooldown

`StealCooldown` (config, default "1 real minute"). Shared with plant.
Uses real-world time (AFK counts down).

---

## Section 4: Plant

**Usage:** `plant <item> <target>` (NPC/mob) or
`plant <item> <container>`

**NPC/mob and container only.** Cannot plant on players. Attempting
to plant on a player: "You can't plant items on other players."

Mechanically a mirror of steal:
- Same roll formula (attacker score vs target Perception or room
  observers)
- Same config knobs (`StealSkillMultiplier`, `StealHiddenBonus`)
- Same failure consequences (spotted, hidden cancelled, observers
  become hostile -- same as steal failure)

**Success:** Item moves from backpack to target's inventory or
container without anyone noticing.

### Error Messages

- No target: "Plant on whom?"
- No item specified: "Plant what?"
- Item not in backpack: "You don't have that."
- Target is player: "You can't plant items on other players."

**Cooldown:** Shares the steal cooldown.

---

## Section 5: Shadow

**Usage:** `shadow <target>`

**Requirements:**
- Must be hidden (have the hidden buff)
- Skill rank 2+
- Target must be in the same room

**Effect:** Automatically follow the target between rooms while
maintaining stealth.

### Per-Room Transition

Two checks happen in sequence on each room transition:

1. **General room-entry stealth check** -- same as normal stealth-on-
   move (roll against all hostile/neutral observers in new room).
   If this fails: shadow ends, you're spotted, hidden drops.

2. **Target-specific detection** (only if step 1 passed) -- opposed
   roll: your `Dex + (SkillMultiplier(skullduggeryRank) * 25)` vs
   target's `Perception + (SkillMultiplier(searchRank) * 25)`. If
   target wins: "You sense someone following you." sent to target.
   Shadow continues but target is alerted.

### Shadow Ends When

- Any room-entry detection check fails (you're spotted)
- You type `shadow stop` or any non-movement command
- Hidden buff expires naturally
- Target logs off
- Target enters a room you can't follow into (locked door, etc.)
- Target dies

### Cooldown

`ShadowCooldown` rounds (config, default 5). Applies whenever shadow
ends for any reason (spotted, target dies, player cancels, target
enters inaccessible room, etc.). Per-player cooldown, not per-target.
Uses `TryCooldown("skullduggery:shadow", ...)`.

---

## Section 6: Picklock

**Existing minigame stays as-is.** Changes:
- Add skill gate: requires skullduggery rank 1+
- Add skill progression: successful picks call
  `CheckSkillProgression` for skullduggery
- Update skill reference from `stealth` to `skullduggery`

No formula or gameplay changes.

---

## Section 7: Defuse (Stub)

**Usage:** `defuse <target>`

Requires skullduggery rank 3+. Checks for traps on a container or
exit.

**Current implementation:** "You don't detect any traps here." (traps
as interactable objects don't exist yet)

**Future:** When traps are implemented, would use a roll:
`Dex + (SkillMultiplier(skullduggeryRank) * 25)` vs trap difficulty.

---

## Section 8: Config Knobs

All new values in `config.balance.go`:

| Config | Type | Default | Purpose |
|--------|------|---------|---------|
| `SneakFailCooldown` | int | 3 | Rounds before retry after failed sneak |
| `SurpriseAttackOffhandPenalty` | float64 | 0.10 | Hit penalty for offhand |
| `SurpriseAttackExtraArm1Penalty` | float64 | 0.25 | Hit penalty for extra arm 1 |
| `SurpriseAttackExtraArm2Penalty` | float64 | 0.40 | Hit penalty for extra arm 2 |
| `StealSkillMultiplier` | float64 | 1.0 | Tuning knob for steal/plant rolls |
| `StealHiddenBonus` | int | 25 | Bonus to attacker score when hidden |
| `StealCooldown` | string | "1 real minute" | Cooldown for steal/plant (real time) |
| `ShadowCooldown` | int | 5 | Rounds before re-shadowing |

---

## Section 9: Skill Registration & Renames

### Skill System

- `Stealth` tag -> `Skullduggery` everywhere in `skills.go`
- Primary stat: Dexterity (unchanged)
- Progression multiplier: 2.0 (unchanged)
- Rogue profession: `{Skullduggery, WeaponCombat}`

### Aggro Type

- `characters.BackStab` -> `characters.SurpriseAttack`
- Rename in `characters/aggro.go`. No YAML or data file changes
  needed (aggro types are not serialized in data files).

### Files to Delete

- `internal/usercommands/skill.stealth.go`
- `internal/usercommands/skill.stealth.pickpocket.go`

### Files to Create

- `internal/usercommands/skill.skullduggery.sneak.go`
- `internal/usercommands/skill.skullduggery.steal.go`
- `internal/usercommands/skill.skullduggery.plant.go`
- `internal/usercommands/skill.skullduggery.shadow.go`
- `internal/usercommands/skill.skullduggery.defuse.go`

### Files to Modify

- `internal/usercommands/attack.go` -- surprise attack check + execution
- `internal/combat/combat.go` -- rename backstab references
- `internal/mobcommands/backstab.go` -- rename to surprise attack
- `internal/mobcommands/sneak.go` -- update skill reference
- `internal/usercommands/picklock.go` -- add skill gate + progression
- `internal/usercommands/usercommands.go` -- register new commands
- `internal/skills/skills.go` -- rename skill
- `internal/usercommands/go.go` -- stealth-on-move checks, shadow
  follow logic, detection when others enter room
- `internal/characters/aggro.go` -- rename BackStab to SurpriseAttack
- `internal/configs/config.balance.go` -- new config knobs
- `_datafiles/world/default/buffs/9-hidden.yaml` -- verify flags

### Help Files

- Delete: `help stealth` (if exists)
- Create: `help skullduggery` (overview of all sub-commands)
- Create: `help sneak`, `help steal`, `help plant`, `help shadow`,
  `help defuse`
- Update: `help picklock` (mention skullduggery skill gate)
- Update: `hints.yaml` (add tips for new commands)

### Broadcast Tips (hints.yaml)

Add the following tips to the rotation:

- "The <ansi fg="skill">skullduggery</ansi> skill covers sneaking,
  stealing, lockpicking, and more. Type
  <ansi fg="command">help skullduggery</ansi> for details."
- "Use <ansi fg="command">sneak</ansi> to enter the shadows. Be
  careful -- sharp-eyed NPCs may notice you trying to hide."
- "Hidden and attacking? If your special move cooldown is ready,
  you'll land a devastating surprise attack automatically."
- "Use <ansi fg="command">shadow</ansi> to tail someone between
  rooms without being seen. Requires rank 2 skullduggery."
- "You can <ansi fg="command">steal</ansi> from NPCs or
  <ansi fg="command">plant</ansi> items on them. Being hidden
  improves your chances."

Remove or update any existing tips that reference the old `stealth`
skill name.

---

## Out of Scope

- Sap/knockout (future sub-command -- CC from stealth)
- Poison (fits under alchemy, not skullduggery)
- Disguise (fits under rhetoric)
- Trap creation (separate system)
- PvP steal/plant (explicitly excluded -- intentional design choice)

---

## Future Work Notes

- **Disguise** could be a rhetoric sub-command (social stealth)
- **Poison** could be an alchemy sub-command
- **Traps** as placeable objects would enable the defuse sub-command
- **Sap** as a CC-from-stealth option alongside surprise attack
