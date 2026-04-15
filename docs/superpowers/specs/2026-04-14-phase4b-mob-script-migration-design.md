# Phase 4b: Mob Script Migration — Design Spec

**Date:** 2026-04-14
**Goal:** Migrate all remaining dogmud mob JS scripts (except Sable) to
behavior trees, upgrade two boss mobs with improved AI, implement
perception-scaled reaction delays, and add cross-mob interaction.

**Depends on:** Phase 4a (behavior tree engine, merged into development)
**Deferred to 4c:** Sable (315) — portal logic is room-bound, not mob-bound

---

## Scope

7 mob JS scripts migrated + 1 existing behavior tree upgraded + 1 empty
stub deleted. Three tiers of work:

| Tier | Mobs | Work |
|------|------|------|
| 1 | Tessara (83), Pell (99), Rhett (242), Sylara (241), Dummy (58), Mage (3) | YAML-only migration using existing features |
| 2 | (engine) | New events, conditions, actions, reaction delays |
| 3 | Edrin (275), Phantom (272), Dal (117) | Upgraded AI using new engine features |

---

## Tier 1: Quest NPC Migrations

These mobs have dialogue YAMLs that handle conversation. The JS handles
`onGive` for quest items. The quest engine triggers already cover core
quest advancement. Behavior trees add item rejection and edge cases.

### Tessara (83) — Dustwalk Road

**JS behavior:** Accepts bandit's purse (item 16) for Q4. Returns wrong
items.

**Behavior tree:** `player_give` event handler.
- If item 16 + has Q4-start + missing Q4-end: respond with flavor text,
  grant Q4-end. (Quest engine may already handle this — tree adds the
  flavor text and item-rejection fallback.)
- If wrong item: say rejection line, return item to player.
- Dialogue stays in dialogue YAML.

### Pell (99) — Thornwall City

**JS behavior:** Accepts bridge report (item 31) for Q6, redirects tithe
ledger (item 29) for Q9 back to Olen.

**Behavior tree:** `player_give` event handler with item discrimination.
- If item 31 + has Q6-start + missing Q6-report: respond, grant Q6-report.
- If item 29 + has Q9-start + missing Q9-end: redirect to Olen, return
  ledger to player via `give_item`.
- If item 29 without active Q9: reject, return ledger.
- Default: reject with bureaucratic flavor.

### Rhett (242) — Ironwind Steppe

**JS behavior:** Accepts windstone sample (item 40032) for Q11. Returns
wrong items.

**Behavior tree:** `player_give` event handler.
- If item 40032 + has Q11-start + missing Q11-end: respond, grant Q11-end.
- If wrong item: reject, return to player.
- Dialogue stays in dialogue YAML.

### Sylara (241) — Ironwind Steppe

**JS behavior:** Two handlers:
1. `onGive`: Accepts wolf totem (item 40033) for Q11. Returns wrong items.
2. `onAsk`: Dispenses spirit fetishes (item 40031) post-Q12. First-time
   bonus of 4, subsequent asks give 1. Gated by Q12-end or knowing
   summon-steppe-spirit spell. Tracks bonus via `GetMiscCharacterData`.

**Behavior tree:** Two event branches.
- `player_give`: Same pattern as Rhett — item 40033 for Q11-end, reject
  others.
- `player_ask`: Keyword match for fetish/spirit/summon/component/more.
  Sequence: check Q12-end or has-spell, check bonus flag via
  `player_has_misc_data`, branch on first-time (give 4 via
  `give_item_multiple`) vs. subsequent (give 1). Set bonus flag via
  `set_misc_data`. Check if player already carrying fetish before giving.

### Training Dummy (58) — Tutorial

**Current JS:** On death, sends room text and triggers teacher mob (57)
to speak.

**Simplified behavior tree:** `mob_die` event handler.
- `grant_quest_to_user` to advance tutorial token for the killer.
- No cross-mob communication. Teacher mob (57) reacts via its own dialogue
  variants gated on the quest token.
- Check: verify what quest token the dummy death should grant. Read the
  tutorial quest YAML to confirm.

### Apprentice Mage (3) — Startland

Empty stub JS file (`// placeholder`). Delete it.

---

## Tier 2: Engine Features

### New Events

**`mob_die`**
- Fires when a mob reaches 0 HP, before corpse creation.
- Wire in the death handler hook (search for mob death processing in
  `internal/hooks/`).
- EventContext: `EventType: "mob_die"`, `UserId` = killer's user ID
  (or 0 if killed by a mob), `RoomId` = death room.

**`mob_flee`**
- Fires when a mob successfully flees combat.
- Wire in the flee command handler (`internal/mobcommands/flee.go` or
  equivalent). Fires AFTER the flee succeeds (mob has moved rooms).
- EventContext: `EventType: "mob_flee"`, `UserId` = 0 (no target),
  `RoomId` = new room (where mob fled to).

**`player_enter`**
- Fires when a player enters a room containing mobs with behavior trees.
- Wire in the movement/go handler where players arrive in a new room.
- For each mob in the destination room, dispatch `player_enter` event.
- EventContext: `EventType: "player_enter"`, `UserId` = entering player,
  `RoomId` = room entered.

### New Conditions

**`mob_has_buff`**
- Params: `buff_id` (int)
- Checks if the mob instance has the specified buff active.
- Uses: `mob.Character.HasBuff(buffId)` or equivalent.

**`player_has_spell`**
- Params: `spell` (string)
- Checks if the triggering player knows the named spell.
- Uses: `user.Character.HasSpell(spellName)` — verify exact method name.

**`player_has_misc_data`**
- Params: `key` (string), `value` (string)
- Checks `user.Character.GetMiscCharacterData(key) == value`.
- For Sylara's bonus fetish tracking.

**`state_greater_than`**
- Params: `key` (string), `value` (int)
- Returns Success if `BehaviorState.GetInt(key) > value`.
- For counter comparisons (Phantom's recovery countdown).

**`multiple_enemies`**
- No params. Returns Success if more than one hostile entity is in the
  mob's room (players + their companions).
- Count: `len(room.GetPlayers()) + companion count > 1`.
- For Edrin's AoE spell decisions.

### New Actions

**`summon_companion`**
- Params: `mob_id` (int), `count` (int, default 1)
- Spawns companion(s) using the same path as player summoning spells:
  `SpawnMobScaled` + charm registration.
- Scales stat pool off the summoning mob's manifestation skill rank +
  willpower, using the same formula as player companions.
- Companions auto-assist (follow master's aggro), despawn on master
  death.
- Uses the consolidated companion spawn function from Phase 2
  (`internal/hooks/spell_resolution.go`).

**`set_room_locked`**
- Params: `direction` (string), `locked` (bool)
- Locks or unlocks an exit in the mob's current room.
- Uses: `room.SetLocked(direction, locked)` — verify exact API.

**`spawn_item_in_room`**
- Params: `item_id` (int), `room_id` (int), `chance` (int, 0-100,
  default 100)
- Creates item and places it in the specified room.
- Rolls `util.Rand(100) < chance` before spawning.
- For Edrin's probabilistic loot in the back room.

**`add_buff`**
- Params: `buff_id` (int)
- Applies the specified buff to the mob itself.
- For Phantom's hidden buff re-application and Edrin's self-buffing.

**`command_mob`**
- Params: `mob_id` (int), `cmd` (string)
- Finds a mob with the given template ID in the same room and executes
  the command as that mob.
- Uses: find mob in room by MobId, call `mob.Command(cmd)`.
- For Dal's patron heckling interactions.

**`give_item_multiple`**
- Params: `item_id` (int), `count` (int)
- Gives N copies of an item to the triggering player.
- For Sylara's first-time 4x fetish bonus.

**`set_misc_data`**
- Params: `key` (string), `value` (string)
- Sets `user.Character.SetMiscCharacterData(key, value)` on the
  triggering player.
- For Sylara's bonus fetish tracking flag.

**`increment_state` / `decrement_state`**
- Params: `key` (string), `amount` (int, default 1)
- Adds/subtracts from a BehaviorState integer value.
- For countdown patterns (Phantom's flee recovery).

**`grant_quest_to_user`**
- Params: `quest` (string token)
- Grants a quest token to the player identified by `ctx.Event.UserId`.
  Works for `mob_die` events where the killer's userId is in the event
  details. Functionally identical to existing `grant_quest` — implement
  as an alias to the same function. Both use `ctx.Event.UserId`.

### Reaction Delay Implementation

Actions are categorized as instant or delayed:

**Instant (execute synchronously):**
- Quest actions: `grant_quest`, `grant_quest_to_user`, `set_quest_flag`
- Commerce: `give_gold`, `take_gold`, `give_item`, `take_item`,
  `give_item_multiple`
- State: `set_state`, `increment_state`, `decrement_state`,
  `set_misc_data`
- Spawning: `summon_companion`, `spawn_item_in_room`
- Room state: `set_room_locked`, `add_temp_exit`

**Reaction-delayed (perception-scaled):**
- Dialogue: `respond`, `say`, `emote`
- Combat: `attack`, `flee`, `cast`
- Movement: `move`
- Buff application: `add_buff`
- Cross-mob: `command_mob`

**Delay formula:**
```
delay = MobBTreeReactionBase - (perception / MobBTreeReactionPerceptionScale)
clamped to [MobReactionDelayMin, MobReactionDelayMax]
```

Config defaults (already in Balance from Phase 4a):
- `MobBTreeReactionBase`: 3.0 seconds
- `MobBTreeReactionPerceptionScale`: 100
- `MobReactionDelayMin`: 0.25 seconds
- `MobReactionDelayMax`: 3.5 seconds

A perception-100 mob reacts in ~2s. A perception-150 mob in ~1.5s.
A perception-50 mob in ~2.5s.

**Implementation:** In each action function, check if the action is in
the delayed category. If so, compute delay from the mob's perception,
wrap the action execution in a closure, and call
`GetEngine().QueueDelayed(delay, closure)`. Return `Success` immediately
(the action is committed, just not yet visible). The delayed action queue
is already drained each round tick (wired in Phase 4a).

### Negative Caching

`TryMobBehavior` currently calls `os.Stat` every idle tick for mobs
without behavior trees. Add a `noTree` map to the Engine:

```go
type Engine struct {
    mu     sync.RWMutex
    trees  map[int]Node
    noTree map[int]bool  // mob IDs confirmed to have no behavior file
    queue  []DelayedAction
}
```

On `os.Stat` failure, add the mob ID to `noTree`. Check `noTree` before
`os.Stat`. Reset `noTree` on `LoadTree` calls (in case files are added
at runtime).

### Behavior Tree Context Documentation

Create `internal/behaviortree/context.md` with:
- Package overview and architecture
- Full node type reference (conditions, actions, decorators)
- Event types and when they fire
- Instant vs. delayed action table
- YAML format guide with examples
- BehaviorState patterns (counters, flags, timers)
- File path convention for behavior tree YAMLs

---

## Tier 3: Upgraded Boss AI

### Old Edrin (275) — Multi-Phase Caster Boss

**Mob YAML updates:**
- Add `manifestation` skill rank (e.g., 15-20) to scale companions
- Add spellbook entries for self-buffs (conviction-ward, chrysalis-cocoon
  or similar defensive spells)
- Set `archetype: casting`
- Ensure willpower training is high (drives companion stat scaling)

**Behavior tree phases:**

**Phase 0 — Preemptive (`player_enter`):**
When a player enters room 4036 (Edrin's room), Edrin begins self-buffing.
Reaction-delayed cast of defensive spells (conviction-ward). Also summons
3 elemental companions via `summon_companion` (scaled off manifestation +
willpower). No dramatic text — just quiet preparation. Companions stand
idle since Edrin isn't hostile.

Use cooldown decorator to prevent re-summoning if player leaves and
re-enters.

**Phase 1 — Reveal (`mob_hurt`, one-time):**
First damage sets state `revealed=true`. Dramatic emote sequence: stoop
vanishes, eyes clear, speaks warning. Perception-scaled delay on the
emotes. His companions auto-assist via the companion system — no explicit
attack commands needed.

**Phase 2 — Tactical Combat (`mob_idle` while revealed + in combat):**
- Check `multiple_enemies` condition: if true, use AoE spells. If false,
  use single-target spells.
- Re-buff if conviction-ward drops (check `mob_has_buff`).
- Perception-scaled reaction timing on all combat actions.
- Existing `combatcommands` in mob YAML provide baseline combat behavior;
  the behavior tree adds the tactical layer.

**Phase 3 — Death (`mob_die`):**
- `set_room_locked`: unlock west exit.
- `spawn_item_in_room`: 4 items at 75% chance each in room 4037.
- Room text describing elementals dissolving (companions auto-despawn).
- Death flavor text.

### Chrysalis Phantom (272) — Hit-and-Run Assassin

**Mob YAML updates:**
- Set `search` skill rank to 20+ (for reliable tracking)
- Verify `buffids: [9]` (hidden buff on spawn)
- Ensure high perception (fast reaction delays)
- Ensure high dexterity (flee success rate)

**Behavior tree — the surprise strike loop:**

**1. Ambush (`mob_idle`, hidden, players in room):**
- Condition: `mob_has_buff(9)` (hidden) AND `players_in_room`
- After perception-scaled delay: `attack` (surprise strike since hidden)
- Store target name in state for later tracking
- Set state `struck=true`

**2. Immediate Flee (`mob_idle`, in combat, struck=true):**
- Condition: `mob_in_combat` AND `state_equals(struck, true)`
- Action: `command: flee`
- Set state `struck=false`
- If flee fails (mob still in combat next tick), try again

**3. Recovery (`mob_flee` event):**
- Set state `recovery=3`
- Store target player name in state

**4. Recovery Countdown (`mob_idle`, recovery > 0):**
- `decrement_state(recovery)`
- At recovery=2: `command: sneak` + `add_buff(9)` (re-hide)
- At recovery=1: Wait (sneak/hide cooldown window)
- At recovery=0: `command: track <targetname>` to hunt target
- Perception-scaled delays on all recovery actions

**5. Suppress idle emotes while hidden:**
- Top of idle branch: if `mob_has_buff(9)`, skip all emote branches
- Prevents hidden Phantom from broadcasting its presence

**6. Target lost:**
- If target is no longer in the zone (track fails repeatedly), return
  home via `command: pathto home`, reset all state, wait for next victim

### Barmaid Dal (117) — Patrol Upgrade

**Existing behavior tree:** Room cycling (home ↔ back room) + gossip
redirect on ask.

**Upgrade:** Add patron heckling on arrival at back room (484).

When Dal arrives in the back room:
- 40% chance (`random` decorator, percent 40) to trigger interaction
- Pick a random patron from mob IDs 114, 115, 116 via `command_mob`
- Patron delivers heckle line (perception-delayed)
- Dal responds with comeback (perception-delayed)
- 4 heckle/response pairs, randomly selected

Interaction pairs (from original JS):
1. Patron: `emote nudges his cup forward with one finger, grinning.`
   Dal: `emote smacks the cup back without looking. "Ask nicer."`
2. Patron: `say Dal, you get prettier every year.`
   Dal: `emote does not look up from the tray. "And you get older..."`
3. Patron: `emote reaches for the bread and accidentally brushes her arm.`
   Dal: `emote pulls her arm back and gives him a look that could curdle milk.`
4. Patron: `say How about a smile with that ale, Dal?`
   Dal: `emote sets the cup down hard enough to slosh. "How about a tip?"`

All interactions perception-delayed for natural pacing.

**Perception notes for Dal and patrons:**
- Dal has Perception training 6 — moderate reaction speed
- Old regulars (114-116) should have low-moderate perception — they're
  old men, not quick on the draw. Reactions feel like natural barroom
  back-and-forth.

---

## Files Changed

**New/modified in `internal/behaviortree/`:**
- `conditions.go` — add 4 new conditions
- `actions.go` — add ~10 new actions
- `engine.go` — add negative caching (`noTree` map)
- `helpers.go` — check `noTree` before `os.Stat`
- `context.md` — full package documentation

**New/modified in `internal/hooks/`:**
- Wire `mob_die` event in death handler
- Wire `mob_flee` event in flee handler
- Wire `player_enter` event in movement/go handler

**New behavior tree YAMLs in `_datafiles/world/dogmud/behaviors/`:**
- `dustwalk_road/83-road_warden_tessara.yaml`
- `thornwall_city/99-records_clerk_pell.yaml`
- `ironwind_steppe/242-geomancer_rhett.yaml`
- `ironwind_steppe/241-windwarden_sylara.yaml`
- `tutorial/58-training_dummy.yaml`
- `marches_spur_road/275-old_edrin.yaml`
- `thornwall_city/272-chrysalis_phantom.yaml`

**Modified behavior tree:**
- `thornwall_city/117-barmaid_dal.yaml` — add patron heckling

**Modified mob YAMLs:**
- `marches_spur_road/275-old_edrin.yaml` — manifestation skill, spellbook,
  casting archetype
- `thornwall_city/272-chrysalis_phantom.yaml` — search skill 20+, verify
  buffids and perception

**JS files removed (renamed to .bak):**
- `dustwalk_road/scripts/83-road_warden_tessara.js`
- `thornwall_city/scripts/99-records_clerk_pell.js`
- `ironwind_steppe/scripts/242-geomancer_rhett.js`
- `ironwind_steppe/scripts/241-windwarden_sylara.js`
- `tutorial/scripts/58-training_dummy.js`
- `marches_spur_road/scripts/275-old_edrin.js`
- `thornwall_city/scripts/272-chrysalis_phantom.js`

**JS file deleted:**
- `startland/3-apprentice_mage.js` (empty stub)

---

## Not In Scope

- Sable (315) — deferred to Phase 4c (room behavior trees)
- Room scripts — Phase 4c
- Remaining spell/buff scripts — Phase 4c
- Go/JS bridge removal — Phase 5
- Behavior trees for mobs that currently have no JS (new content)
