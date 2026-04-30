# Room Schema Reference

## 1. Filename & Location

**Path formula:**
```
_datafiles/world/dogmud/rooms/{zone_folder}/{roomid}.yaml
```

- `{zone_folder}` = `ConvertForFilename(zone display name)` — lowercase, keep a-z/0-9, drop apostrophes, all other chars → underscore.
  Example: `"Sanctum Basin"` → `sanctum_basin/`
- `{roomid}` = the integer roomid. The filename IS the ID — no name conversion.

**Worked example:**
- Zone: `Startland`, Room ID: `42`
- Path: `_datafiles/world/dogmud/rooms/startland/42.yaml`

**Existing zone folders:**
- `startland/`
- `sanctum_basin/`
- `a_dark_forest/`
- `endless_trashheap/`
- `shadow_realm/`
- `test_arena/`
- `world_road/`

---

## 2. Field Reference

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `roomid` | int | **yes** | Unique across all zones. Filename must match. |
| `zone` | string | **yes** | Display name of the zone (e.g. `"Sanctum Basin"`). |
| `title` | string | **yes** | Short room title shown to players (e.g. `"Town Square"`). |
| `description` | string | **yes** | Multi-sentence room description. Use YAML block scalar (`\|` or `>`) for long text. |
| `biome` | string | no | Affects weather/ambient. Valid values: `city`, `desert`, `forest`, `cave`, `water`, `plains`, `arctic`, `swamp`, `dungeon`, `mountain`, `ruins`. |
| `mapsymbol` | string | no | Single character shown on the automap (e.g. `T`, `*`, `#`). |
| `maplegend` | string | no | One-word label for the map legend (e.g. `"Townsquare"`). |
| `pvp` | bool | no | If config pvp is `limited`, this overrides per-room. Default false. |
| `isbank` | bool | no | Players can deposit/withdraw gold here. |
| `isstorage` | bool | no | Players can stash/retrieve items here. |
| `ischaracterroom` | bool | no | Players can create/swap characters here. |
| `musicfile` | string | no | Background music filename. |
| `station` | string | no | Crafting station present: `"forge"`, `"alchemy"`, `"workshop"`, etc. |
| `gold` | int | no | Gold coins on the floor at room load. |
| `exits` | map | **yes** | At minimum an empty map `exits: {}`. See Exit sub-fields below. |
| `spawninfo` | list | no | Mob/item spawns for this room. See SpawnInfo sub-fields below. |
| `idlemessages` | list | no | Flavor text messages displayed periodically. Supports `<ansi>` tags. |
| `nouns` | map | no | Keyword → description. Players can `look {noun}` to read. |
| `containers` | map | no | Named containers (e.g. `chest:`) holding items. |
| `hidden_nouns` | map | no | Hidden noun objects discovered via search. See Hidden Nouns sub-section below. Marked `instance:"skip"`. |
| `signs` | list | no | Readable signs. Each has a `title` and `body`. |
| `skilltraining` | map | no | Skill → `{min: N, max: N}` range. Allows players to train here. |
| `mutators` | list | no | Mutator tags applied when the room spawns. Each entry is `- mutatorid: <tag>`. Mutators can append flavor text, modify regen (`regenmultiplier` field on the mutator spec — e.g. `sanctuary` 5x), apply buffs, or override PvP. See `_datafiles/world/dogmud/mutators/`. |

### Exit Sub-fields

```yaml
exits:
  north:          # direction: north/south/east/west/up/down/enter/etc.
    roomid: 42    # (required) destination room ID
    lock: true    # (optional) door is locked
    key: 10001    # (optional) item ID that unlocks this exit
    cost: 5       # (optional) gold cost to use this exit
```

Valid directions: `north`, `south`, `east`, `west`, `up`, `down`, `enter`, `leave`, `northwest`, `northeast`, `southwest`, `southeast`

### SpawnInfo Sub-fields

```yaml
spawninfo:
  - mobid: 5             # (required if spawning mob) mob ID to spawn
    itemid: 10001        # (required if spawning item) item ID to drop
    message: "..."       # (optional) message shown when mob spawns
    respawnrate: "5 real minutes"   # (optional) how often to respawn
    levelmod: 2          # (optional) level modifier applied to spawned mob
    questflags:          # (optional) quest flags required for this spawn
      - someflag
    buffids:             # (optional) extra buffs applied to spawned mob
      - 3
    forcehostile: true   # (optional) override mob's default hostility
    maxwander: 3         # (optional) override mob's max wander distance
    idlecommands:        # (optional) override mob's idle commands for this room
      - "say hello"
    name: "Guard Captain" # (optional) override mob name for this instance
    scripttag: patrol    # (optional) override mob script tag
```

### SkillTraining Sub-fields

```yaml
skilltraining:
  unarmed:
    min: 0
    max: 10
  survival:
    min: 5
    max: 25
```

### Hidden Nouns Sub-fields

Hidden noun objects are discovered via the `search` command and provide optional flavor or worldbuilding details. Each hidden noun has a description players see after discovery and a hidden description appended to the room description for discoverers.

```yaml
hidden_nouns:
  scratchmarks:                    # noun key (used with `look scratchmarks`)
    description: |
      Deep claw marks gouged into the wooden wall, three in parallel.
      They're old, weathered by time.
    hidden_description: |
      You notice strange scratchmarks on the wall here.
  passage:
    description: |
      A narrow gap behind the tapestry, barely wide enough to squeeze
      through. Darkness lies beyond.
    hidden_description: |
      There's a hidden passage here!
```

**Rules:**
- `description` — what `look <noun>` displays after discovery
- `hidden_description` — appended to the room description for players who have discovered this noun
- No formal parent link; references to parent objects are written as prose in `hidden_description`
- Marked `instance:"skip"` — always loaded from template, never instance-saved

---

## 3. Annotated Example

```yaml
# _datafiles/world/dogmud/rooms/startland/1.yaml
roomid: 1                    # Must match filename (1.yaml)
zone: Startland              # Display name; folder = startland/
title: Town Square
description: You stand at the town square of startland. This is the first room you
  enter upon completing training and beginning the mud.
mapsymbol: T                 # Shows as 'T' on map
maplegend: Townsquare        # Legend label
biome: city                  # Affects ambient weather

exits:
  north:
    roomid: 2                # Just a roomid reference — not a title
  south:
    roomid: 200
  east:
    roomid: 5
  west:
    roomid: 7

spawninfo:
- mobid: 2                   # Town guard mob
  message: A town guard emerges from a nearby building.
  idlecommands:              # This instance overrides the mob's default idle commands
  - say did you know there's a sign in the Townsquare with a map of the area?
  - ""
  - ""
  - ""
  - wander
  levelmod: 10               # Mob spawns 10 levels higher than its template
  respawnrate: 5 real minutes

idlemessages:                # Flavor — shown periodically to players in the room
- A <ansi fg="mobname">citizen</ansi> walks up and examines the sign.
- A <ansi fg="mobname">guard</ansi> is looking at the sign.
```

---

### Container Sub-fields

```yaml
containers:
  chest:                           # container name (referenced in exits/items)
    title: An oaken chest
    description: A sturdy wooden chest with iron bands.
    gold: 50                       # optional gold coins inside
    items:                         # optional items inside
      - 10001
      - 10002
    lock:                          # optional lock
      difficulty: 8
      key: 10003
    hidden: false                  # if true, container hidden until discovered
```

**The `hidden` field:**
- If `true`, the container is invisible in normal room descriptions and `look` output
- Players can only interact with it after discovering it via the `search` command
- Once discovered, the container appears in room details and `look` output for all players who have found it
- The container noun should be subtly color-highlighted in the room description with `<ansi fg="itemname">noun</ansi>` for discoverability hints

---

## 4. Gotchas

**Instance saves silently override template edits.**
The engine loads `rooms/{zone}/{roomid}.yaml` first, then overwrites with data from `rooms.instances/{zone}/{roomid}.yaml` if that file exists. If you edit a room template and the change doesn't appear in-game, check `_datafiles/world/dogmud/rooms.instances/{zone_folder}/` for a stale instance file and delete it.

**Zone folder name must match `ConvertForFilename(zone)`.**
If the folder is named `sanctum-basin/` instead of `sanctum_basin/`, the engine will panic on startup with:
```
filesystem path "..." did not end in Filepath() "..."
```
Always use underscores, never hyphens.

**Exits reference roomids, not titles.**
Never put a room title in an exit field. Only integers.

**Roomid must be globally unique.**
Before assigning a new ID, scan all zone folders for the highest existing roomid to avoid collisions.

**`spawninfo` is skipped in instance saves** (tagged `instance:"skip"`), so spawn configuration always comes from the template. Instance saves store current state (items on floor, gold, etc.).
