# /new-mob

Generate a new mob YAML file (and optional JS script stub) for DOGMud.

## Instructions

You are generating a new mob data file for the DOGMud MUD. Follow these steps exactly.

### Step 1 — Load context

Read these files before generating anything:
1. `world.md` — lore, tone, world facts (Gaius, Chrysalis, the three moons, etc.)
2. `docs/schemas/mob.md` — complete field reference, filename formula, gotchas

Then glob 2 existing mob files as few-shot examples:
- `_datafiles/world/dogmud/mobs/sanctum_basin/55-elder_saris.yaml` (complex NPC with LLMProfile)
- `_datafiles/world/dogmud/mobs/startland/1-rat.yaml` (simple hostile creature)

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

Using `$ARGUMENTS` as the creative brief:
- Choose a name that fits the world's tone (no modern slang, no fantasy clichés)
- Write a description in 2–4 sentences — physical details only, no personality narration
- Set `speciesid` from the known list in `docs/schemas/mob.md`
- Set `hostile`, `maxwander`, `activitylevel` consistent with the creature's behavior
- Set `groups` (e.g. `animal`, `undead`, `humanoid`) and `hates` if the creature has natural enemies
- Write `idlecommands` that feel alive (emotes, wandering, brief spoken fragments)
- If the creature has combat behavior, add `aiprofile` and `combatcommands`
- If the creature is a speaking NPC (not purely hostile), ask: "Should I also generate a dialogue YAML? (y/n)"

**Do not add an LLMProfile unless the user explicitly requests it.** LLMProfile requires Ollama to be running.

**No hard numbers in description or messages.** Describe behavior and appearance; never expose stat values.

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

---

## Usage

```
/new-mob "cave troll, aggressive, stone-skin natural armor, Sanctum Basin"
/new-mob "elderly herbalist NPC, friendly, sells remedies, Startland"
/new-mob "feral bat swarm, hostile, caves near Sanctum Basin"
```

Arguments are a freeform description. Include: creature type, temperament, any special traits, and the zone.
