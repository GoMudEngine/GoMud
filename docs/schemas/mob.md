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

**Workflow:** new mobs are usually built as part of a zone — see
`docs/CONTENT_GENERATION_GUIDE.md` Section 2 for the full zone-build
SOP including the `behavior_archetype` priority order.

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
| `statpool` | int | no | Stat points distributed across stats on spawn (weighted by archetype). |
| `archetype` | string | no | Stat distribution archetype: `"fighting"` (80% physical), `"casting"` (80% mental), or `""` (even). |
| `groups` | list | no | Group membership (e.g. `[rats, animal]`). Used for teamwork and hates logic. |
| `hates` | list | no | Group names or species this mob will attack on sight. |
| `buffids` | list | no | Buff IDs always applied when mob spawns. |
| `questflags` | list | no | Quest flag strings set on this mob. |
| `scripttag` | string | no | Tag appended to the script filename. Must match the `.js` file. |
| `behavior_archetype` | string | no | Filename (without `.yaml`) of an archetype in `_datafiles/world/dogmud/behaviors/archetypes/`. Drives the mob's behavior tree. **Strongly preferred over legacy `aiprofile`/`combatcommands`/`tactic_preset` for new mobs.** See "Behavior Archetypes" below. |
| `aiprofile` | string | no | Legacy combat AI profile: `"default"`, `"aggressive"`, `"defensive"`, `"grappler"`, `"brawler"`, `"tactical"`. (legacy — prefer `behavior_archetype`) |
| `specialmovechance` | int | no | Base % chance to use special moves in combat (0–100). |
| `tactic_preset` | string | no | Reactive AI preset: `"aggressive_melee"`, `"defensive_caster"`, `"ambusher"`, `"tank"`. See Reactive AI below. (legacy — prefer `behavior_archetype`) |
| `tactics` | list | no | Custom tactic rules. Each entry: `trigger`, `action`, `priority`. Merged with preset. (legacy — prefer `behavior_archetype`) |
| `reaction_delay` | float | no | Seconds before reactive AI acts (0.25–4.0). Lower = faster reactions. (legacy — prefer `behavior_archetype`) |
| `tactical_discipline` | float | no | 0.0–1.0. Probability the mob follows through on a tactic. Higher = more reliable. (legacy — prefer `behavior_archetype`) |
| `idlecommands` | list | no | Commands executed while not in combat. Use `""` for empty (wait) turns. |
| `combatcommands` | list | no | Commands executed while in combat. |
| `spawnmutations` | list | no | Mutation IDs always granted at level 1 on spawn (Phase 24.3). |
| `mutationchance` | int | no | % chance (0–100) to gain 1 random bonus mutation on spawn (Phase 24.3). |
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

**Reactive AI fields require at least one tactic source.**
A mob needs either `tactic_preset` or `tactics` (or both) for the reactive AI to fire. Without either, the mob uses only the legacy `aiprofile` + `combatcommands` system. `reaction_delay` and `tactical_discipline` are only meaningful when tactics exist.

**Tactic presets:**
- `aggressive_melee` — kick prone targets, bash casters, submit grapplers
- `defensive_caster` — self-shield, AoE on multiple targets, flee when low, single-target spell
- `ambusher` — flee after engagement, hide when out of combat, trip casters
- `tank` — bash casters, kick prone targets, call for help when low

**Available triggers:** `combat_start`, `health_below:N`, `target_casting`, `target_prone`, `target_grappled`, `multiple_targets`, `single_target`, `no_aggro`, `not_hidden`, `after_action:X`, `player_entered`, `has_buff:N`, `missing_buff:N`

**Available actions:** `flee`, `hide`, `kick`, `bash`, `trip`, `call_for_help`, `retarget_strongest`, `cast <spell>`, `track_memory`, `recall`

**`statpool` distributes by archetype.**
Stats in `statpool` are weighted by `archetype` at spawn: `"fighting"` favors Str/Dex/Vit (80/20 split), `"casting"` favors Per/Wil/Cha (20/80 split), and default is uniform. Identical mob templates will still vary. Use explicit `stats:` overrides when a specific stat spread matters.

**`level` in `character:` sets baseline — statpool modifies it.**
The engine calls `AutoTrain()` after distributing statpool points. Do not set both a high level and a large statpool expecting them to stack cleanly.

---

## 5. Behavior Archetypes

`behavior_archetype` selects a behavior-tree YAML from
`_datafiles/world/dogmud/behaviors/archetypes/`. The archetype drives
combat decision-making, packmate awareness, and reactive AI without
needing to author per-mob `combatcommands`/`tactics`.

### Available archetypes

| Archetype | Role |
|-----------|------|
| `generic_fighter` | Melee with bash/trip/grapple toolkit. Default for non-tank fighters. |
| `tank_taunter` | Melee with signature taunt + self-buffs. For high-priority threats. |
| `melee_self_buff` | Melee fighter who pre-buffs before engaging. |
| `ambusher` | Hidden until engagement; high opening burst. |
| `pure_caster` | Spell-focused; flees from melee, kites with damage. |
| `support_caster` | Buffs/heals packmates; rarely the front-line target. |
| `leader` | Commands packmates, calls for help, coordinates. |
| `prey` | Flees on engagement; non-aggressive. |
| `lookout` | Stationary observer; calls for help when triggered. |
| `combat_passive` | In combat but doesn't attack — atmospheric or quest fodder. |
| `noncombat_passive` | Walks idles, no combat behavior. |
| `noncombat_questgiver` | Stationary, dialogue-only NPC. |
| `noncombat_shopkeeper` | Stationary shop NPC. |

### Priority for new mobs

When authoring a new mob:

1. **Reuse** an existing archetype if one fits.
2. **Author a new archetype YAML** under `behaviors/archetypes/` if
   the behavior pattern is reusable across multiple mobs.
3. **Custom legacy** (`aiprofile` + `combatcommands` + `tactic_preset`)
   ONLY for bosses or signature one-off NPCs.

See `docs/CONTENT_GENERATION_GUIDE.md` "Building a Full Zone" for the
full zone-build workflow including the smoke-test checklist.

### Pairing with stat distribution

`behavior_archetype` and `archetype` (stat distribution) usually pair
naturally:

- `pure_caster` / `support_caster` → `archetype: "casting"`
- `generic_fighter` / `tank_taunter` / `ambusher` /
  `melee_self_buff` → `archetype: "fighting"`
- `prey` / `noncombat_*` → `archetype: ""` (uniform)
