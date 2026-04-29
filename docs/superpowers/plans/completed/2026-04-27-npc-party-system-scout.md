# Scout Report: Bandit Camp Room & Existing Bandit BTree State
**Date:** 2026-04-25  
**Task:** Stage 1 of NPC Party System implementation  
**Status:** Complete

## Findings

### Bandit Camp Room
- **Room ID:** 4052
- **Title:** "Bandit Camp"
- **Zone:** North Road
- **Description:** A clearing hacked from scrub with firepit, bedrolls in circle, lean-to shelter with supplies. Working camp with military discipline.
- **Current Spawn:** mobid 284 (bandit_fighter), mobid 285 (bandit_caster), mobid 286 (Soren — leader)
- **Exits:** east to room 4043 (North Road)
- **Coordinate:** x: -19, y: -10, z: 0

### Lookout's Home Room
- **Room ID:** 4043
- **Title:** "North Road"
- **Zone:** North Road
- **Description:** Road wider with wagon traffic, stand of scrub trees on west side. Figure loiters near treeline watching road.
- **Current Spawn:** mobid 283 (bandit_lookout)
- **Exits:** south to 4042, north to 4044 (Drainage Ditch), west to 4052 (Bandit Camp)
- **Coordinate:** x: -18, y: -10, z: 0
- **Note:** The lookout's behavior_archetype is `lookout` with routine `watch_north_road` and routine_links to `bandit_camp_guard`. It has `pack_flee_immune: true` and `maxwander: 1`.

### Bandit Mob Definitions
All bandit mobs are in `/c/Users/Calabe Davis/workspace/DOGMud/_datafiles/world/dogmud/mobs/north_road/`:
- **283-bandit_lookout.yaml** — archetype: fighting, behavior_archetype: lookout, pack_flee_immune: true
- **284-bandit_fighter.yaml**
- **285-bandit_caster.yaml**
- **286-soren.yaml** — the pack leader

### Existing Bandit Behavior Tree Files

**North Road:**
- No behavior tree directory exists yet for `north_road/` zone.
- No per-mob behavior files for mobs 283, 284, 285, 286.
- Bandits currently rely on archetype behavior + ad-hoc hooks.

**Marches Spur Road (other bandit group):**
- **254-bandit_leader.yaml** exists in `/c/Users/Calabe Davis/workspace/DOGMud/_datafiles/world/dogmud/behaviors/marches_spur_road/`
- Mob 254 (bandit_leader) is in room 4007 of marches_spur_road
- This BTree implements timed aggro: waits 5 rounds if player is in room without talking, then attacks
- Dialogue cancels countdown via `negotiating` state flag

### MobDeath_PackFlee.go Summary
**Location:** `internal/hooks/MobDeath_PackFlee.go`

**Behavior:** When any mob dies, the hook checks if it has a species_id and queues a `flee` command on all other mobs in the room that share the same species AND don't have exemptions. Exemptions:
- Charmed/companion mobs skip flee
- Non-combatant mobs never scatter (merchants, quest NPCs)
- Mobs with `pack_flee_immune: true` hold ground
- Only mobs with matching MobId or matching species_id flee

**Key Output:** Visual room text when pack scatters: "Sensing the death of their packmate, the remaining {species} scatter!"

**Stage 42.8 Integration:** Also calls `mobs.HandleAlphaDeath()` if pack roaming is enabled, which scatters the entire pack hierarchy.

**Significance for NPC Party System:**
- This hook currently drives bandit flee behavior
- Task 3 will migrate this to the party system's `on_member_death` handler
- The lookout's `pack_flee_immune: true` prevents it from fleeing, making it an implicit "leader" role that can call the pack back

## Files Ready for Task 2 Implementation

The following paths are accurate for Task 2 (implement party system + wire bandits):

- Room file to edit: `/c/Users/Calabe Davis/workspace/DOGMud/_datafiles/world/dogmud/rooms/north_road/4052.yaml` (bandit camp)
- Mob files: `/c/Users/Calabe Davis/workspace/DOGMud/_datafiles/world/dogmud/mobs/north_road/283-286.yaml` (4 files)
- Reference behavior file: `/c/Users/Calabe Davis/workspace/DOGMud/_datafiles/world/dogmud/behaviors/marches_spur_road/254-bandit_leader.yaml` (timed aggro pattern)
- Hook to preserve/migrate: `internal/hooks/MobDeath_PackFlee.go`

## No Issues Found

All room titles, mob IDs, and behavior patterns are consistent and discoverable. The bandit camp room is clearly titled and described as a working camp with living bandits. The lookout's `pack_flee_immune` flag already hints at its leadership role.
