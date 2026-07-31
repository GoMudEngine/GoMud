# /new-mob

Generate a new mob YAML file (and optional JS script stub) for DOGMud.

## Instructions

You are generating a new mob data file for the DOGMud MUD. Follow these steps exactly.

### Step 1 — Load context

Read these files before generating anything:
1. `docs/world.md` — lore, tone, world facts (Gaius, Chrysalis, the three moons, etc.)
2. `docs/schemas/mob.md` — complete field reference, filename formula, gotchas
3. List archetypes: glob `_datafiles/world/dogmud/behaviors/archetypes/*.yaml`. The 13 filenames here are the valid values for `behavior_archetype` in Step 4 below.

Then glob 2 existing mob files as few-shot examples:
- `_datafiles/world/dogmud/mobs/sanctum_basin/55-elder_saris.yaml` (complex NPC with LLMProfile)
- `_datafiles/world/dogmud/mobs/ashwick/264-timber_wolf.yaml` (simple hostile creature with `behavior_archetype: generic_fighter`)

### Step 2 — Determine the next available mobid

Glob all mob YAML files:
```
_datafiles/world/dogmud/mobs/**/*.yaml
```
Find the highest existing mobid and suggest the next integer. Confirm with the user if there is any ambiguity.

### Step 3 — Determine the zone folder

From the description in `$ARGUMENTS`, identify the zone. Apply `ConvertForFilename` to the zone display name to get the folder:
- Lowercase everything
- Keep a-z, 0-9
- Drop apostrophes
- All other characters → underscore

Example: `"Sanctum Basin"` → `sanctum_basin`

If the zone folder does not exist yet, inform the user: "Zone folder `{folder}/` does not exist. It will need to be created, or choose an existing zone."

### Step 4 — Generate the mob YAML

Using `$ARGUMENTS` as the creative brief, fill in the YAML. The
fields below are listed in priority order — start with behavior, then
stats, then flavor.

#### (a) Choose `behavior_archetype` — most important field

The 13 available archetypes (loaded in Step 1):

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

**Priority order** (from `docs/CONTENT_GENERATION_GUIDE.md` Section
2):

1. **Reuse** an existing archetype if one fits the brief.
2. **Author a new archetype YAML** under `behaviors/archetypes/` if
   the behavior is reusable across multiple mobs. Tell the user this
   is what you're doing — they will need to author the new archetype
   YAML separately.
3. **Custom legacy** (`aiprofile` + `combatcommands` +
   `tactic_preset`) ONLY for bosses or signature one-off NPCs.

Picking option 3 should be rare and deliberate. If `$ARGUMENTS`
describes a generic creature, option 1 is the answer.

#### (b) Choose stat distribution `archetype`

| Value | Stat split | Use when |
|-------|------------|----------|
| `"fighting"` | 80% physical (Str/Dex/Vit) | Brawlers, beasts, melee-focused humanoids |
| `"casting"` | 80% mental (Per/Wil/Cha) | Spellcasters, scholars, charisma-driven NPCs |
| `""` (default) | uniform random | Mixed roles, generic NPCs |

Pair with `behavior_archetype` sensibly — a `pure_caster` should
almost always have stat archetype `"casting"`.

#### (c) Decide on `spawnmutations` and `mutationchance`

Mutations differentiate mobs of the same archetype. A
`generic_fighter` goblin and a `generic_fighter` ogre share AI but
feel different because one has thick-skin and the other has rage.

- `spawnmutations: [42, 18]` — guaranteed mutation IDs on every
  spawn. IDs from `_datafiles/world/dogmud/mutations/`.
- `mutationchance: 25` — % chance (0–100) to gain ONE extra random
  mutation on top.

Use `spawnmutations` for signature traits (a "stone goblin" always
has Stone Skin); use `mutationchance` for variety (most spawned
wolves are normal, occasional pack-leader has bonus mutations).

#### (d) Standard fields

`character.name`, `character.description` (2–4 sentences, physical
only, no personality narration), `character.speciesid`, `hostile`,
`maxwander`, `activitylevel`, `groups`, `hates`, `idlecommands`. See
`docs/schemas/mob.md` for the full reference.

**Naming:** fits the world's tone (no modern slang, no fantasy
clichés).

**Description:** physical details, no behavior/personality narration.
No raw numbers (damage, armor, etc.).

#### (e) Skip these unless you've chosen the legacy path (option 3)

`aiprofile`, `combatcommands`, `tactic_preset`, `tactics`,
`reaction_delay`, `tactical_discipline`. With a `behavior_archetype`
set, these are usually unnecessary — the archetype YAML drives
behavior. Including them alongside `behavior_archetype` is
redundant and potentially confusing.

**Do not add an LLMProfile unless the user explicitly requests it.**
LLMProfile requires Ollama to be running.

### Step 5 — Verify before writing

Check:
1. `{mobid}-{ConvertForFilename(name)}.yaml` — filename matches name exactly
2. `mobid` is unique (not already in use)
3. Zone folder name uses underscores, not hyphens
4. All required fields present: `mobid`, `zone`, `character.name`, `character.description`, `character.speciesid`
5. If `scripttag` is set, note that a JS file will also be needed

### Step 6 — Write the file

Write to:
```
_datafiles/world/dogmud/mobs/{zone_folder}/{mobid}-{ConvertForFilename(name)}.yaml
```

### Step 7 — Optional JS stub

If the mob needs script behavior (custom combat logic, quest interactions, triggered actions), offer to generate a JS stub. The stub filename must be:
```
_datafiles/world/dogmud/mobs/{zone_folder}/scripts/{mobid}-{ConvertForFilename(name)}-{scripttag}.js
```

### Step 8 — Remind the user

After writing:
> Mob file written. To see it in-game: restart the server. If you want this mob to spawn somewhere, add a `spawninfo` entry to the relevant room YAML and restart again.
>
> If you want a dialogue YAML for this mob, run `/new-room` is not needed — just create `_datafiles/world/dogmud/dialogue/{zone_folder}/{mobid}.yaml` using `docs/schemas/dialogue.md` as reference.
>
> Once your zone has all its mobs and items in place, run the smoke
> checklist (in `docs/CONTENT_GENERATION_GUIDE.md` Section 2) before
> starting `/sketch-quest` for any quests in this zone.

---

## Usage

```
/new-mob "cave troll, aggressive, stone-skin natural armor, Sanctum Basin"
/new-mob "elderly herbalist NPC, friendly, sells remedies, Startland"
/new-mob "feral bat swarm, hostile, caves near Sanctum Basin"
```

Arguments are a freeform description. Include: creature type, temperament, any special traits, and the zone.
