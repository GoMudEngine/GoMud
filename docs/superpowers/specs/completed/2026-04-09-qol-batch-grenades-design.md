# QOL Batch + Grenade System Design

**Date:** 2026-04-09
**Scope:** 7 independent QOL improvements + new grenade subsystem

---

## 1. Sort Command: Bandolier Support

**Goal:** Extend `sort` to auto-move potions into the equipped bandolier,
mirroring how it already moves components into the component bag.

**Changes:**
- `internal/usercommands/sort.go` — after `SortComponentItems()`, call a
  new `SortPotionItems()` method
- `internal/characters/character.go` — add `SortPotionItems()`:
  - Iterate backpack items where `itemSpec.Subtype == items.Drinkable`
  - Move each into `Character.PotionItems` if a bandolier is equipped
    (`Equipment.Belt` with `IsBandolier == true`) and capacity allows
  - Return count of items moved
- Player message: `"Sorted N potion(s) into your bandolier."` (only if N > 0)
- Combined output example: `"Sorted 3 component(s) into your component bag
  and 2 potion(s) into your bandolier."`

---

## 2. Taunt Pulls Aggro (Tank Taunt)

**Goal:** A successful taunt forces the target mob to switch aggro to the
taunter. Extends existing conviction damage mechanic with an aggro-pull
effect.

### Mechanic
- Opposed roll: Charisma + Rhetoric (attacker) vs Willpower + Rhetoric
  (defender) -- this is the existing taunt roll, no change needed
- On **hit or crit**: if the target mob is currently fighting someone else
  (different `Aggro.UserId` or companion), force `SetAggro` to the taunter
- On **miss or fumble**: no aggro change (existing behavior)
- The conviction damage still applies as before -- aggro pull is additive

### Messaging
Extend `_datafiles/world/dogmud/taunt-messages/rhetoric.yaml` with
aggro-pull variants. When a taunt lands AND pulls aggro (target was
fighting someone else), append additional flavor text after the normal
hit/crit message:

**Attacker sees (appended to hit message):**
- `"Your words cut deep -- {target} turns away from its prey and fixes its fury on you!"`
- `"The insult lands true. {target} abandons its quarry and lunges toward you!"`
- `"Your mockery is unbearable -- {target} wheels around to face you!"`
- `"{target} snarls and shifts its full attention to you."`

**Defender sees (appended to hit message):**
- `"Something about {source}'s words fills you with rage. You can't ignore them!"`
- `"The barrage of insults overwhelms your focus -- you turn on {source} in fury!"`
- `"{source}'s taunts are maddening. You abandon your target and charge!"`

**Room sees (appended to hit message):**
- `"{target} breaks off and turns its fury on {source}!"`
- `"Enraged by {source}'s taunts, {target} shifts its attention!"`
- `"{target} abandons its quarry and charges at {source}!"`

When taunt hits but the target was already fighting the taunter, no extra
message -- standard hit messaging only.

### Implementation
- `internal/actions/combat_taunt.go` — add `AggroPulled bool` to
  `TauntResult` struct. After a successful hit/crit, check if target's
  current aggro differs from attacker; if so, call `SetAggro` on the mob
  to target the taunter and set `AggroPulled = true`
- `internal/usercommands/taunt.go` — when `result.AggroPulled`, load and
  send the aggro-pull append messages from a new YAML section or inline
- Works for both player taunting mob AND mob taunting player (symmetry)

### Golem Companion Taunt
- Add `taunt` to flesh golem's `combatcommands` list in
  `_datafiles/world/dogmud/mobs/summons/305-flesh_golem.yaml`
- The mob taunt command (`internal/mobcommands/taunt.go`) already exists
  and calls the same `ExecuteTaunt()` — just needs the aggro-pull logic
  added to the shared function
- Golem becomes the "tank pet" archetype: high Vitality + taunt in rotation

### Fighter AI Integration
- Add `taunt` to the `combatcommands` of fighter-archetype hostile mobs
  where it makes sense (bandit leaders, guards, etc.)
- Not all fighters -- just mobs that thematically would taunt (sentient
  humanoids with rhetoric capability)

---

## 3. Auto-Eject Spoiled Potions from Bandolier

**Goal:** Spoiled potions automatically move from bandolier to backpack
with a notification, so players don't accidentally drink rotten potions.

**Changes:**
- In the regen tick handler (where potion aging already advances), after
  calculating aging phase for each potion in `PotionItems`:
  - If `phase == PhaseSpoiled`, remove from `PotionItems` and add to
    backpack via `StoreItem()`
  - Send notification: `"<ansi fg=\"yellow\">Your <itemname> has spoiled
    and falls out of your bandolier.</ansi>"`
- Only fires once per potion (on the tick it transitions to spoiled)
- If backpack is full / over carry capacity, still eject -- the potion
  goes to backpack regardless (same as unequipping items when over capacity)

---

## 4. Food Spoiling + Grenades

### 4a. Food Aging System

**Goal:** Extend the potion aging system to food items (`Edible` subtype).

**Changes:**
- `internal/items/aging.go` — `GetAgingPhase()` and
  `CalcEffectiveAgingSpeed()` already work on any item with `aging:`
  thresholds. No changes needed to the core aging engine.
- Food item YAMLs get `aging:` fields (same format as potions):
  ```yaml
  aging:
    ferment_rounds: 500
    peak_rounds: 1000
    decay_rounds: 2000
    spoil_rounds: 3000
  ```
- Food does NOT use bottles, so `BottleMultiplier` defaults to 1.0
- `internal/usercommands/eat.go` — add spoiled food check mirroring
  `drink.go`:
  - If `phase == PhaseSpoiled`, block consumption with message:
    `"<ansi fg=\"red\">The food has gone bad! It reeks of decay and is
    clearly inedible.</ansi>"`
  - Unlike potions, spoiled food is NOT consumed when attempted -- just
    refused. Player must salvage or drop it.
- Existing food items that lack `aging:` fields continue to work unchanged
  (no aging = never spoils)

### Crafted Food: Aging Stamp at Craft Time

Food items produced by crafting recipes MUST get `CraftedRound` stamped
(same as potions already do) so the aging clock starts at creation time.
The crafting system already stamps `CraftedRound` on potion outputs -- the
same path needs to apply to any output item with `aging:` thresholds,
regardless of subtype.

**Changes:**
- In the crafting output logic (where `CraftedRound` is set for potions),
  broaden the condition: stamp `CraftedRound = currentRound` on ANY crafted
  item whose ItemSpec has non-zero `AgingThresholds`, not just `Drinkable`
  subtype items.
- Food item templates (YAMLs) define the `aging:` thresholds. The crafted
  instance inherits them from the spec and starts aging from `CraftedRound`.
- Pre-existing food items in player inventories have `CraftedRound == 0`,
  which means `GetAgingPhase()` returns `PhaseFresh` with zero elapsed
  rounds -- they effectively never age. This is the desired backward
  compatibility: old food is grandfathered, new crafted food spoils.
- Shop-purchased food (non-crafted) also has `CraftedRound == 0` and will
  not age. This is intentional -- only player-crafted food spoils.

### 4b. Putrid Residue (Salvage Material)

**New item:**
- **Item ID:** 40050
- **Name:** Putrid Residue
- **component_tag:** `putrid-residue`
- **is_component:** true
- **Description:** A vile, viscous paste that forms when food breaks down.
  Useful in volatile alchemical mixtures.
- **Value:** 1 gold

**Salvage integration:**
- All spoilable food items get `salvage_returns:` in their ItemSpec:
  ```yaml
  salvage_returns:
    - item_tag: putrid-residue
      quantity: 1
  ```
- Spoiled potions also get `salvage_returns` for putrid residue (1 each)
- Salvage works on spoiled items using the existing salvage system

### 4c. Grenade Items

Three new throwable items crafted via alchemy:

| Grenade | Item ID | Effect | Damage/Debuff |
|---------|---------|--------|---------------|
| Flashbang | 30057 | AoE blind debuff | Buff: 3-round blind (no damage) |
| Firebomb | 30058 | AoE burst damage | Physical damage, Dex-scaled |
| Toxic Flask | 30059 | AoE poison DoT | Buff: 5-round poison tick |

**New buff IDs:**
- 77: Flashbang Blindness (3 rounds, `no-combat` flag for 1 round + perception penalty)
- 78: Toxic Cloud (5 rounds, DoT ticking health damage each round)

**Alchemy recipes** (3 new recipes):
- Flashbang: Putrid Residue + Glass Vial + Mineral Salt
- Firebomb: Putrid Residue + Flask of Oil + Glass Vial
- Toxic Flask: Putrid Residue + Venom Sac + Glass Vial

Each grenade is `type: object`, `subtype: throwable`, single-use (`uses: 1`).

### 4d. Throw Command

**New command:** `throw <item>` or `throw <item> at <target>`

**Mechanics:**
- Requires combat (must have aggro) OR a specified target
- Consumes the thrown item (1 use)
- AoE resolution: opposed roll per mob in room
  - Attacker: Dexterity + Skullduggery skill
  - Defender: Dexterity + Perception skill (each mob rolls independently)
- On hit: apply grenade effect (damage or debuff)
- On fumble (ZScore <= -2.0): effect hits the thrower instead
- Shares `special-move` cooldown
- Progresses Dexterity stat and Skullduggery skill on use
- Companion protection: skip friendly companions (same as other AoE)
- PvP check: respect PvP settings for player targets

**Damage scaling (Firebomb):**
- `raw = Dexterity * SkillMultiplier(skullduggery) * 0.30` (physical channel scale)
- Apply target physical mitigation, then `dice.RollStat(final)` for variance
- Distributed evenly -- each mob takes full damage (like AoE spells, not split)

**Debuff scaling (Flashbang/Toxic Flask):**
- Duration scales with Skullduggery skill: `baseDuration * (1 + skill/100)`
- Opposed roll still determines hit/miss per target

---

## 5. Sell from Bandolier / Component Bag

**Goal:** `sell` command searches bandolier and component bag as fallbacks
when the item isn't found in backpack.

**Changes to `internal/usercommands/sell.go`:**
```
item, found := FindInBackpack(rest)
if !found {
    item, found = FindInPotions(rest)   // bandolier
}
if !found {
    item, found = FindInComponents(rest) // component bag
}
```
- Removal already works across all storage types via existing remove methods
- No change to sell pricing or merchant logic

---

## 6. Companion SP/CP Pool Fix

**Goal:** Fix companion stamina burnout by raising mob regen rates and
improving companion pool scaling.

### Diagnosis
A fire elemental companion with a 100-Cha summoner has ~102 SP with
1 SP/tick regen. At 4 SP per unarmed swing, it runs dry in ~25 swings
while only regenerating ~1 SP/round. A player with similar stats has
~405 SP -- 4x more sustainable.

### Changes

**Raise mob regen rates** (in `_datafiles/config.yaml` under `Balance`):
- `MobStaminaRegenPct`: 0.01 -> 0.02 (2% per tick)
- `MobConvictionRegenPct`: 0.01 -> 0.02 (2% per tick)
- `MobHealthRegenPct`: stays at 0.01 (health regen already feels right)

This affects ALL mobs (hostile + companions):
- Companions sustain longer in combat
- Hostile mobs are harder to chip down with hit-and-run tactics
- A fire elemental now regens ~2 SP/tick instead of 1, making net cost
  ~2 SP/round instead of 3 -- extends combat endurance by ~50%

**Improve companion stat pool scaling:**
- `ManifestStatScaleChaFactor`: 200 -> 150 (Charisma contributes more)
- At 100 Cha + 10 manifestation: scale goes from 1.7x to 1.87x
- Fire elemental SP pool: ~102 -> ~112 SP
- Combined with 2% regen: ~2.2 SP/tick, net cost ~1.8 SP/round
- Rough combat endurance: ~62 rounds (up from ~34)

---

## 7. Rhetoric Shouts: Warcry + Rally

**Goal:** Two new rhetoric-based AoE buff shouts that buff all friendly
characters in the room.

### Warcry (Offense Buff)
- Applies buff to all friendlies in room (player + companions + party)
- Buff effect: `physical_damage_multiplier` bonus
- **Buff ID:** 79
- Duration: ~25 rounds base
- Shares `special-move` cooldown

### Rally (Defense Buff)
- Applies buff to all friendlies in room (player + companions + party)
- Buff effect: avoidance modifier (dodge, parry, block bonuses)
- **Buff ID:** 80
- Duration: ~25 rounds base
- Shares `special-move` cooldown

### Magnitude Curve
```
bonus = 0.05 + 0.15 * sqrt((rhetoric / 75.0) * (charisma / 175.0))
```
Clamped to [0.05, 0.20] range.

| Rhetoric | Charisma | Bonus |
|----------|----------|-------|
| 1        | 100      | ~5.5% |
| 25       | 120      | ~11%  |
| 50       | 150      | ~16%  |
| 75       | 175      | 20%   |

### Buff YAML Design

**Warcry (Buff 79):**
```yaml
buffid: 79
name: Warcry
description: A rallying battle cry bolsters your fighting spirit.
triggerrate: 1 round
triggercount: 25
statmods:
  physical_damage_multiplier: <dynamic, set by shout magnitude>
```
The `triggercount` and `physical_damage_multiplier` are set dynamically
via `AddBuffScaled()` based on the shouter's stats at cast time.

**Rally (Buff 80):**
```yaml
buffid: 80
name: Rally
description: An inspiring shout steadies your defenses.
triggerrate: 1 round
triggercount: 25
statmods:
  dodge: <dynamic>
  parry: <dynamic>
  block: <dynamic>
```

### Progression
- `OnStatUse(Charisma)` and `OnSkillUse(Rhetoric)` on every cast
- In combat: full progression chance (standard `OnSkillUseScaled`)
- Out of combat: 50% progression multiplier via a halved difficulty bonus
  (soft incentive to use in combat without hard-blocking out-of-combat use)

### Messaging

**Warcry caster:**
`"<ansi fg=\"yellow-bold\">You let loose a thundering warcry!</ansi>"`

**Warcry room:**
`"<ansi fg=\"yellow-bold\">{source} lets loose a thundering warcry that
stirs your blood!</ansi>"`

**Rally caster:**
`"<ansi fg=\"cyan-bold\">You shout words of encouragement to your
allies!</ansi>"`

**Rally room:**
`"<ansi fg=\"cyan-bold\">{source} shouts words of encouragement --
you feel steadier on your feet!</ansi>"`

### Command Names
- `warcry` — offense shout
- `rally` — defense shout

Both registered as user commands with `special-move` cooldown check.

---

## ID Allocation Summary

| Type | ID | Name |
|------|----|------|
| Item | 40050 | Putrid Residue |
| Item | 30057 | Flashbang |
| Item | 30058 | Firebomb |
| Item | 30059 | Toxic Flask |
| Buff | 77 | Flashbang Blindness |
| Buff | 78 | Toxic Cloud |
| Buff | 79 | Warcry |
| Buff | 80 | Rally |

---

## Implementation Order

These are independent and can be parallelized, but a natural order:

1. **Sort bandolier** (smallest, isolated)
2. **Sell from bandolier/component bag** (smallest, isolated)
3. **Auto-eject spoiled potions** (small, isolated)
4. **Companion SP/CP fix** (config changes + scaling tweak)
5. **Taunt pulls aggro** (moderate, self-contained)
6. **Rhetoric shouts** (moderate, new commands + buffs)
7. **Food spoiling + grenades** (largest, depends on aging system + new
   throw command + new items + recipes)
