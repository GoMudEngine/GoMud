# Item Authoring — Design (admin web-building, sub-project 2)

**Date:** 2026-07-23
**Status:** approved design, pre-plan
**Epic:** Admin web-building. **Sub-project 2** — the item-template editor, built as
a second mode of the **1b `/build` page** (rooms done, merged; adjacent-zone
context added). Instancing (instanced-zone authoring in the room editor) and mob
`LootPool` assignment are the following sub-projects (2.5 / 3), not this one.

## Motivation

1b delivered a graphical room builder. Item templates (`ItemSpec` definitions)
are still authored only by hand-editing YAML or the `/new-item` slash command,
and the read-only `/admin/items` page has dead Submit buttons. Sub-project 2
delivers a real item-template editor: an admin opens the builder, switches to an
**Items** tab, browses/searches all items, edits any field of any item with a
type-aware form, creates new items, deletes them (with a reference safety
check), and everything persists to the item YAML templates and is live in-game
without a restart.

Scope is **item templates only**. Placing items (room containers, mob drops),
mob `LootPool` assignment, and instanced-zone authoring are later sub-projects.
Any authored item can already serve as an instance-loot base — the scaling lives
on the mob (`LootPool`) and the zone (entry `gold_paid`), so nothing item-side is
needed to make an item "an instance item."

## Architecture

### Page + session (reuse 1b)
- The existing `/build` page and its admin GMCP session are reused. A slim
  **`Rooms | Items` toggle** in the toolbar switches modes.
- **Items mode** replaces the SVG map canvas (left ~70%) with a **searchable item
  list**, and reuses the right-hand inspector (~260px) for the **type-aware
  form**. The zone/plane/floor toolbar controls hide in Items mode; a search box
  + type filter take their place.

### `Build.Item.*` GMCP protocol (client→server, RoleAdmin-gated)
Added as cases in `HandleIAC` (`modules/gmcp/gmcp.go`), each deferred to
**MainWorker** via the existing `GMCPBuildOp` event (item mutations write the
shared `items` map + files, so they must not run on the connection goroutine —
same concurrency fix as `Build.Room.*`).

| Package | Payload | Server action |
|---------|---------|---------------|
| `Build.Item.List` | `{}` | return a lightweight index of every item: `{id, name, type, subtype, rarity}` for the browse list |
| `Build.Item.Get` | `{itemId}` | full `ItemSpec` detail (all editable fields) + enum lists for dropdowns |
| `Build.Item.Create` | `{type}` | `CreateNewItemFile` assigns the next free id in the type's range → return the new detail (form loads it) |
| `Build.Item.Update` | `{itemId, …fields}` | mutate the loaded spec → `SaveItemSpec` (validate + write + relocate file on rename/retype) |
| `Build.Item.Delete` | `{itemId}` | reference scan → block + list refs if used, else `DeleteItemSpec` |

**Server → client:** `Build.Item` (full detail, mirrors `Build.Room`),
`Build.Items` (the list index), and `Build.Result {ok, error, itemId}` after
mutations. A `Build.Result` error carries the reference list on a blocked delete.

### Server plumbing (`internal/items`)
- **Create:** reuse `CreateNewItemFile(ItemSpec) (int, error)` — already assigns
  `getNextItemId(type)`, validates, saves via `SaveFlatFile`, caches. Wrap it.
- **Update — new `SaveItemSpec(ItemSpec) error`:** `Validate()`, then write to the
  spec's `Filepath()`. **File-relocation gotcha:** item filenames embed the name
  (`<id>-<ConvertForFilename(name)>.yaml`) and armor sits in per-`Type`
  subfolders (`armor-20000/<type>/`), so a Save that changes the name or type
  changes the path — `SaveItemSpec` must remove the OLD file before writing the
  new one (compare the cached spec's old `Filepath()` to the new one). Update the
  in-memory `items[id]`.
- **Delete — new `DeleteItemSpec(itemId) error`:** remove the file + `delete(items, id)`.
- **Enum providers:** `ItemTypes()` / `ItemSubtypes()` already exist; expose valid
  types, subtypes, elements, stat-mod names, and vendor categories for dropdowns.

## The item browser (Items-mode left panel)
- A **search box** (matches id or name substring) + a **type filter** dropdown.
- Rows: `#<id> · <name> · <type>/<subtype> · T<rarity>`. Click → `Build.Item.Get`
  loads the form.
- A **`+ New Item`** button opens a small type picker → `Build.Item.Create`.
- The list is refreshed (`Build.Item.List`) on open and after any create/delete.

## The type-aware form (inspector)
Sections render based on the item's `Type`; the client holds the field-groups
(consistent with how the room inspector hard-codes fields), populated from
`Build.Item.Get`. **Weapon damage is `DamageMultiplier` (float) — the legacy
`Damage` dice struct is NOT exposed.**

- **Common (all types):** name, display-name, simple-name, description, type
  (dropdown), subtype (dropdown, filtered by type), value, weight, uses,
  quantity, **`RarityTier`** (dropdown: 50/40/30/20/10 / untiered), vendor
  categories (multi), flags (not-salable, never-drops, restricted, cursed),
  quest-token. Plus a generic **StatMods** editor (stat → value rows).
- **Weapon (`Type=weapon`):** `DamageMultiplier`, hands (1/2), parry-rating,
  speed-multiplier, stamina-cost, wait-rounds, min-strength, reach,
  grapple-modifier, element; **caster subtypes** (wand/sceptre/staff) →
  `SpellDamageMultiplier`; **ranged** (shooting) → ammo-tag.
- **Armor (any of the ~14 wearable slots):** physical/magical/conviction
  mitigation, block-rating (shields/offhand), escape-modifier,
  reserve-{health,stamina,conviction}-pct.
- **Consumable (potion/food/drink):** buff ids, toxicity, aging thresholds,
  bottle-aging-multiplier; bandolier flags (is-bandolier, capacity,
  preserves-contents, ambient-potions).
- **Component / bag (is-component or componentbag):** is-component,
  component-tag, weight-reduction, bag-capacity, salvage-returns (item_tag +
  quantity rows).
- **Ammo / Key:** ammo-tag / key-lock-id.
- **Advanced (collapsible, rare/sentient):** procs (trigger/chance/effect/params),
  mutation-tick fields, voice-id, hunger fields, taunt-pull, worn-buff ids,
  break-chance, on-use-train fields.

**Save model:** explicit **Save** (batched, one `Build.Item.Update`), mirroring
the room inspector. Create is immediate (a blank valid item); field edits batch
until Save.

## Delete + reference scan
`Delete item` (confirm-gated) → `Build.Item.Delete` → the server scans
**in-memory** data for the item id across:
- **mobs:** `LootPool` (instance-loot bases) and drop tables,
- **shops:** stock lists,
- **crafting recipes:** outputs and id-referenced ingredients,
- **quests:** reward `itemid` (and dialogue `givesItem`),
- **rooms:** container item lists (authored chest contents; room `spawninfo` spawns
  mobs, not items, so it isn't scanned).

If any references exist → **block** and return them (`Build.Result` error listing
"mob 9538, quest 10, …"). If clean → `DeleteItemSpec` + refresh the list.
Editing never dangles references (item id is stable), so only delete is gated.

## Out of scope (explicit)
- **Instanced-zone authoring** (mark a zone `instanced`, entry room/cost/duration)
  — the next piece (2.5), a room-editor extension; it's what makes entry gold
  scale loot.
- **Mob `LootPool` assignment** and mob drop tables — the mob editor (3).
- Item **placement** in rooms/containers — sub-project 4.
- Bulk/offline item editing (the `/admin/items` app's niche).
- Attack-message / defense-message group authoring (separate data files).

## Testing & verification
- **Unit** (`internal/items` + `modules/gmcp`): `SaveItemSpec` round-trips fields
  through YAML; **relocates the file** when name or type changes (old path gone,
  new path present); `getNextItemId` assigns within the type's range; the
  reference scan finds a referenced item and passes a clean one; a non-admin
  `Build.Item.*` is refused.
- **Boot-clean** after any struct/loader change (items load without panic;
  `MapConsistencyEnforce=panic` still errors=0).
- **REQUIRED content/UX gate** (CLAUDE.md): drive the real `/build` Items tab as
  an admin — create one item of each major class (weapon, an armor slot, a
  potion, a component), edit fields incl. RarityTier + DamageMultiplier + Save,
  confirm each **persists to YAML AND is usable in-game** (spawn/give it, equip a
  weapon and see the multiplier apply); attempt a **blocked delete** (an item in
  use) and a **clean delete**. Read every interaction as a confused human would;
  fix what it surfaces, re-run.
