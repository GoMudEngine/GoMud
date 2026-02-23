# Mob Schema Reference

## 1. Filename & Location

**Path formula:**
```
_datafiles/world/dogmud/mobs/{zone_folder}/{mobid}-{ConvertForFilename(name)}.yaml
```

- `{zone_folder}` = `ConvertForFilename(zone display name)` — lowercase, keep a-z/0-9, drop apostrophes, all other chars → underscore.
- `{ConvertForFilename(name)}` — same conversion applied to the mob's character name.

**Worked example:**
- Zone: `Sanctum Basin`, Mob ID: `12`, Name: `"Cave Troll"`
- Path: `_datafiles/world/dogmud/mobs/sanctum_basin/12-cave_troll.yaml`

**Optional JS script:**
```
_datafiles/world/dogmud/mobs/{zone_folder}/scripts/{mobid}-{ConvertForFilename(name)}-{scripttag}.js
```

**Existing zone folders:**
- `startland/`
- `sanctum_basin/`
- `endless_trashheap/`
- `test_arena/`
- `tutorial/`
- `test/`

---

## 2. Field Reference

### Top-level Fields

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `mobid` | int | **yes** | Unique integer. Must match filename. |
| `zone` | string | **yes** | Display name of the zone (e.g. `"Sanctum Basin"`). |
| `hostile` | bool | no | Whether this mob attacks players on sight. Default: false. |
| `maxwander` | int | no | Max rooms the mob will wander from its home room. 0 = stationary. |
| `activitylevel` | int | no | 1–100. How often the mob executes idle commands. Higher = more active. |
| `itemdropchance` | int | no | Percent chance (0–100) to drop carried items on death. |
| `statpool` | int | no | Stat points randomly distributed across stats on spawn. |
| `groups` | list | no | Group membership (e.g. `[rats, animal]`). Used for teamwork and hates logic. |
| `hates` | list | no | Group names or species this mob will attack on sight. |
| `buffids` | list | no | Buff IDs always applied when mob spawns. |
| `questflags` | list | no | Quest flag strings set on this mob. |
| `scripttag` | string | no | Tag appended to the script filename. Must match the `.js` file. |
| `aiprofile` | string | no | Combat AI profile: `"default"`, `"aggressive"`, `"defensive"`, `"grappler"`, `"brawler"`, `"tactical"`. |
| `specialmovechance` | int | no | Base % chance to use special moves in combat (0–100). |
| `idlecommands` | list | no | Commands executed while not in combat. Use `""` for empty (wait) turns. |
| `combatcommands` | list | no | Commands executed while in combat. |
| `character` | object | **yes** | Character sub-object. See below. |
| `llmprofile` | object | no | LLM-driven dialogue. See below. |

### Character Sub-object

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `name` | string | **yes** | Mob's display name. Must match filename (`ConvertForFilename(name)`). |
| `description` | string | **yes** | Text shown when players look at the mob. |
| `speciesid` | int | **yes** | Species template. See known species below. |
| `level` | int | no | Starting level. Default: 1. |
| `gold` | int | no | Gold the mob carries (can be looted). |
| `stats` | map | no | Stat overrides applied at spawn. Each stat has a `training` key (positive or negative integer adjustment). |
| `items` | list | no | Items the mob spawns with. Each entry: `itemid: N`. |

**Known species IDs** (from `_datafiles/world/dogmud/species/`):
| speciesid | Name |
|-----------|------|
| 0 | ghostly spirit |
| 1 | human |
| 10 | rodent |
| 19 | dummy |
| 20 | orb |
| 22 | bat |

### LLMProfile Sub-object (Stage 18.3)

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `model` | string | **yes** | Ollama model name (e.g. `"llama3.2:3b"`). |
| `systemprompt` | string | **yes** | Character persona, world facts, speech rules. Use YAML block scalar (`\|`). |
| `maxwords` | int | no | Max words per LLM response. Default: 100. |
| `cachettl` | string | no | How long to cache identical responses. Format: `"1hour"`, `"30minutes"`. |
| `defaultmood` | string | no | Starting mood: `"friendly"`, `"neutral"`, `"cautious"`, `"hostile"`. |

---

## 3. Annotated Example

```yaml
# _datafiles/world/dogmud/mobs/startland/1-rat.yaml
mobid: 1                     # Must match filename (1-rat.yaml)
zone: Startland              # Display name; folder = startland/
itemdropchance: 10           # 10% chance to drop carried items on death
hostile: false               # Does not attack on sight
maxwander: 8                 # Will wander up to 8 rooms from home
groups:
  - rats
  - animal
idlecommands:
  - 'emote wiggles its nose'
  - 'wander'
  - ''                       # Empty string = skip this turn (adds variance)
activitylevel: 10            # Fairly slow/quiet mob

character:
  name: rat                  # Must match filename segment (rat → 1-rat.yaml)
  description: 'The rat''s sleek, mottled fur...'   # Single quotes; escape ' as ''
  level: 1
  speciesid: 10              # rodent species
  stats:
    vitality:
      training: -2           # Slightly less HP than baseline
  items:
    - itemid: 40002          # Spawns carrying this item
```

**Example with LLMProfile (Elder Saris):**
```yaml
# _datafiles/world/dogmud/mobs/sanctum_basin/55-elder_saris.yaml
mobid: 55
zone: Sanctum Basin
hostile: false
maxwander: 0                 # Stationary
activitylevel: 10
idlecommands:
  - 'emote observes the moons through the bronze sighting device'
  - 'emote makes a small notation in a slim leather-bound journal'
  - ''
  - ''

llmprofile:
  model: "llama3.2:3b"
  maxwords: 100
  cachettl: "1hour"
  defaultmood: "neutral"
  systemprompt: |
    You are Elder Saris, the oldest living resident of Sanctum Basin...
    # (Full persona prompt here — 1–4 paragraphs max for 3b models)

character:
  name: Elder Saris
  description: 'Elder Saris is old in the way that certain basalt formations are old...'
  speciesid: 1               # human
  level: 5
  gold: 0
```

---

## 4. Gotchas

**Filename must match character.name exactly via `ConvertForFilename`.**
If mob name is `"Cave Troll"`, filename must be `{id}-cave_troll.yaml`. A mismatch causes either a startup panic or the mob loading silently under the wrong key.

**Zone folder must use underscores.**
`sanctum-basin/` will panic. Always `sanctum_basin/`.

**Script tag must match `.js` filename.**
If `scripttag: patrol`, the JS file must be named `{mobid}-{name}-patrol.js`. Mismatches cause the script to never load (no error, silent failure).

**LLMProfile is optional — dialogue file is the fallback.**
If Ollama is unreachable, the engine falls back to the mob's dialogue YAML (if one exists). Always provide at minimum a dialogue file for important NPCs. See `docs/schemas/dialogue.md`.

**`statpool` distributes randomly.**
Stats in `statpool` are distributed at spawn, so identical mob templates will vary. Use explicit `stats:` overrides when a specific stat spread matters.

**`level` in `character:` sets baseline — statpool modifies it.**
The engine calls `AutoTrain()` after distributing statpool points. Do not set both a high level and a large statpool expecting them to stack cleanly.
