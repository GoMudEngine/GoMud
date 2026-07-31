# /new-item

Generate a new item YAML file for DOGMud.

## Instructions

You are generating a new item data file for the DOGMud MUD. Follow these steps exactly.

### Step 1 — Load context

Read these files before generating anything:
1. `docs/world.md` — lore, tone, world aesthetics (no anachronistic materials or references)
2. `docs/schemas/item.md` — complete field reference, ID ranges, folder layout, filename formula

Then glob 2–3 existing items as examples:
- Glob `_datafiles/world/dogmud/items/weapons-10000/*.yaml`, read one result
- Glob `_datafiles/world/dogmud/items/armor-20000/body/*.yaml` or similar, read one result
- If generating a consumable: read `_datafiles/world/dogmud/items/consumables-30000/30001-small_red_potion.yaml`

### Step 2 — Determine item type and ID range

From `$ARGUMENTS`, determine the item type and correct ID range:

| Type | ID range | Folder |
|------|----------|--------|
| Weapon | 10000–19999 | `weapons-10000/` |
| Armor (any slot) | 20000–29999 | `armor-20000/{slot}/` |
| Consumable | 30000–39999 | `consumables-30000/` |
| Material/misc | 40000+ | `materials-40000/` or as appropriate |

Glob all item YAMLs within the relevant folder to find the highest existing itemid in that range. Suggest the next available ID.

### Step 3 — Generate the item YAML

Using `$ARGUMENTS` as the creative brief:
- Write a name that fits the world (no generic fantasy names unless they fit — prefer specific, grounded names)
- Write a `namesimple` (one or two words players use to refer to it: `longsword`, `potion`, `belt`)
- Write a description: 1–2 sentences. Focus on physical appearance and feel, not game stats.
- Set `type` and `subtype` from the valid values in `docs/schemas/item.md`

**For weapons:**
- Set `hands` (1 or 2)
- Set `damage.basedamage` and `damage.variance` — keep values balanced relative to existing weapons
- Set `speedmultiplier` (reference existing weapons; lighter weapons should be faster)
- Set `staminacost` (reference existing weapons)
- Set `parryrating` if appropriate for weapon type

**For armor:**
- Set `damagereduction` consistent with the slot and material tier (cloth < leather < metal)
- Set `statmods` only if the armor has a meaningful enchantment or special property
- Set `wornbuffids` for magical armor effects

**For consumables:**
- Set `uses` (usually 1)
- Reference an existing buff ID for `buffids`, or note that a new buff will be needed

**No hard numbers in descriptions.** Describe feel, not mechanics. "The blade feels unusually light" not "speedmultiplier: 1.2".

### Step 4 — Determine subfolder for armor

For armor items, identify the correct slot subfolder:
- `head/`, `body/`, `legs/`, `feet/`, `hands/`, `neck/`, `ring/`, `belt/`, `offhand/`

### Step 5 — Verify before writing

Check:
1. `{itemid}-{ConvertForFilename(name)}.yaml` — filename matches name exactly
2. `itemid` is unique and in the correct range for the type
3. Folder path is correct (weapons vs armor vs consumables, correct slot for armor)
4. All required fields present: `itemid`, `name`, `description`, `type`, `subtype`
5. No raw stats in player-visible text fields

### Step 6 — Write the file

Write to the correct path:
```
_datafiles/world/dogmud/items/{type_folder}/{slot_if_armor}/{itemid}-{ConvertForFilename(name)}.yaml
```

### Step 7 — Remind the user

After writing:
> Item written. To use it in-game:
> - Add the `itemid` to a mob's `character.items` list to make them carry it
> - Add a `spawninfo` entry with `itemid` in a room YAML to place it in the world
> - Restart the server to load the new item.
>
> If this item references a buff ID that doesn't exist yet, create the buff first using `docs/schemas/buff.md`.

---

## Usage

```
/new-item "one-handed obsidian blade, medium damage, slashing, slightly faster than iron"
/new-item "woven grass cloak, light armor, body slot, mild warmth"
/new-item "bitter root tincture, consumable, applies a temporary perception boost"
/new-item "iron key, opens the north gate of Sanctum Basin room 112"
```

Arguments are a freeform description. Include: item category, material, key stats or properties, and any special behaviors.
