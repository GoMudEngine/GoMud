# Mob Authoring Editor (Admin Web-Building Sub-Project 3) — Design

**Date:** 2026-07-23
**Status:** Approved design, pre-implementation
**Epic:** Admin web-building (sub-project 3 of 5; follows 1a coord model,
1b room builder page, 2 item authoring)
**Predecessor specs:** `2026-07-22-admin-builder-page-1b-design.md`,
`2026-07-23-item-authoring-2-design.md`, `2026-07-23-item-advanced-behaviors-design.md`

## Goal

Full-parity mob-template authoring in the `/build` admin web page: create,
edit, delete, and test-spawn mob templates without touching YAML or logging
into the game. This is net-new plumbing — unlike rooms (`SaveRoomTemplate`)
and items (`SaveItemSpec`), **no mob-template writer exists anywhere in the
engine**; `SaveMobInstance` only persists a small progression blob to
`mobs.instances/`.

Scope decisions (user-confirmed):

- **Full field parity** with the mob YAML, organized like the item editor:
  core sections always visible, collapsible Advanced for
  commerce/aliveness/hooks/overrides.
- **Template-only, plus targeted test-spawn.** Spawn lists remain
  sub-project 4. The editor includes a test-spawn action that spawns one
  instance into an **admin-chosen zone + room** picked in the web UI — the
  admin is not assumed to be logged into the game.
- Dialogue, behavior-tree, and quest authoring stay deferred (sub-project 5).

## 1. Writer seam — `internal/mobs/save.go`

Mirrors `internal/items/save.go`:

- **`SaveMobSpec(mob Mob) error`** — validate (below) → if name or zone
  changed, remove the old file first (filenames embed
  `<mobid>-<ConvertForFilename(name)>.yaml` under the zone folder, so a
  rename or re-zone moves the path; a stale duplicate must not linger to
  boot as a second copy) → `fileloader.SaveFlatFile` (honoring
  `CarefulSaveFiles`) → update the in-memory template cache under its lock.
- **`DeleteMobSpec(mobId MobId) error`** — remove file + cache entry.
  Callers must run the reference scan first (§3).
- **`CreateNewMobFile(zone string) (MobId, error)`** — seed a stub at the
  next free mob ID (filesystem scan, per the verify-IDs SOP — never trust a
  cached counter) with **boot-safe defaults**: placeholder name ("New Mob"
  disambiguated by ID), empty description, `speciesid` human, small
  statpool, `non_combatant: false`, `auto_aggro: false`, no schedule/patrol,
  no shop. A stub left unedited must never fail a boot validator.
- **`mobsBasePath`** package var (tests point it at a temp dir), mirroring
  `itemsBasePath`.

**Accepted churn (same class the room saver accepted):** a full re-marshal
drops hand-authored YAML comments (e.g. `# lake-iron nodule` annotations in
shop lists) and reflows description block scalars (`>` → `|`, semantically
identical). Editing a mob in the web editor rewrites its file in canonical
marshal form. Documented cost, not a bug.

### Save-time validation (boot-brick prevention)

Every boot-time check that can panic (or silently mis-load) is replicated at
save so the editor cannot create a file that bricks the next boot — the
vendor-categories lesson from sub-project 2, applied from day one:

- Filename/name-field consistency is **derived, not validated**: the file
  path is computed from the canonical name, so a mismatch is impossible.
- Zone folder must exist (`ConvertForFilename` of a real zone).
- `schedule_id` resolves to a loaded schedule; `patrol_id` to a loaded
  patrol; `behavior_archetype` to `behaviors/archetypes/<name>.yaml`.
- All `buffids` exist; all `loot_pool`, `character.shop`,
  `character.equipment`, and `character.items` item IDs exist (equipment
  items must be equippable in the assigned slot).
- `speciesid` is a valid race; `archetype` ∈ {`fighting`, `casting`, ``};
  `aiprofile` ∈ the Stage-8.9 set; `submission_policy` / `surrender_policy`
  ∈ their enum sets; `craft_support` ∈ `shops.ValidCraftSupports`;
  `crafterskill` a real craft skill and `crafterrecipeids` real recipes when
  `crafter: true`.
- Numeric sanity: `statpool ≥ 0`, `activitylevel` 0–100,
  `ItemDropChance`/`mutationchance`/`specialmovechance` 0–100.

Validation lives on the mobs side (a `ValidateSpec`-style func callable by
both `SaveMobSpec` and tests), returning player-readable errors that name
the field and list valid values.

## 2. GMCP protocol — `Build.Mob.*`

New handlers in the `modules/gmcp` Build family, following the item-editor
architecture exactly:

- **Admin-gated** (`requireAdmin`, silent refusal for non-admins — both
  layers, page auth and per-message).
- **Dispatched to MainWorker via event** — never run on the connection
  goroutine (the 1b concurrent-map-write crash lesson).
- **`mobDeps` seam**: all world access behind a small interface so handlers
  unit-test against a fake world, mirroring `itemDeps`.

Messages:

| Message | Payload → Response |
|---|---|
| `Build.Mob.List` | zone → `[{mobid, name, zone, statpool, non_combatant, has_schedule, has_shop}]` summaries |
| `Build.Mob.Get` | mobid → full editable spec + the **enums payload** |
| `Build.Mob.Create` | zone → creates stub via `CreateNewMobFile`, returns its detail |
| `Build.Mob.Update` | full spec → validate + `SaveMobSpec`, `buildErr` on rejection |
| `Build.Mob.Delete` | mobid → reference scan (§3); refuse-with-list or delete |
| `Build.Mob.Spawn` | `{mobid, roomid}` → validates room exists, spawns ONE instance into it |

**Enums payload** (served with `Get`/`Create` so the form never offers free
text for a reference field): races (id + name), archetypes, AI profiles,
behavior archetypes, schedule IDs and patrol IDs (grouped by zone), buff
list (id + name), item list (id + name, for loot-pool and shop pickers),
craft supports, craft skills + recipe IDs, submission/surrender policies,
and the observed set of `groups` values for suggestions.

**Range hints:** numeric fields ship observed min–max across existing mobs
(statpool, activitylevel, gold, drop chance …), reusing the item editor's
ranges mechanism, so authors see what values are normal.

**Test-spawn semantics:** `Build.Mob.Spawn` reuses the same spawn path as
the in-game `mob spawn` admin command. The instance is ordinary and
transient — it wanders/idles per its template, is killable, and does not
survive restarts (no instance-save implications; charmed/progression rules
don't apply to a fresh spawn). Response confirms mob name + room so the
admin gets feedback without being in-game.

## 3. Delete reference scan

`Build.Mob.Delete` refuses deletion while the template is referenced,
returning the reference list (same UX as the item delete's world-wide scan).
Scanned surfaces:

- Room templates' spawn lists (`spawninfo` mob IDs) — all zones.
- Other mobs' `relationships:` edges (`to: <mobId>`).
- Quest YAMLs (mob references in triggers/steps, e.g. `mob_death`).
- Dialogue: existence of `dialogue/<zone>/<mobId>.yaml`.
- Conversation pair files (`conversations/pairs/<lower>_<higher>.yaml`).
- Mob-for-hire lists.

The scan is read-only over loaded template data (not the filesystem) where
caches exist; dialogue/conversation checks may hit the filesystem since
those are keyed by ID/pair filename.

## 4. Web UI — mob panel in `/build`

Third editor panel alongside rooms and items (`mobs.js`, following
`items.js` structure): a list (current zone + name search) and an inspector
form with explicit Save. Un-stuck Save/Delete bar per `310d97538`.

**Core sections (always visible):**

- **Identity** — name, description (textarea), zone (dropdown), species
  (dropdown), groups (chips w/ suggestions).
- **Stats & Combat** — statpool, per-stat training overrides
  (`character.stats.<stat>.training`), level, archetype, auto_aggro,
  AI profile, special-move chance, activity level, maxwander, pack
  routine + routine_links, hates.
- **Equipment** — per-slot worn-item picker (`character.equipment`,
  slot → itemid) and carried-items list (`character.items`).
- **Flavor** — idle / combat / angry command list editors (add/remove/reorder
  rows; blank rows allowed — they're real "do nothing" pool entries).
- **Loot** — item drop chance, loot-pool item picker, gold.

**Advanced (collapsible, item-editor pattern):**

- **Commerce** — shop table editor (rows of item picker / quantitymax /
  price), buys_general, stock_multiplier, craft_support, crafter block
  (crafter, crafterskill, crafterrecipeids, crafterrestockmaterials).
- **Aliveness** — schedule_id (dropdown, filtered to the mob's zone),
  patrol_id (dropdown), relationships row editor (to-mob picker / type /
  subtype), knows_facts, default_disposition, fold_anchor_room,
  storage_chest_room.
- **Hooks** — scripttag, behavior_archetype (dropdown), buffids (picker),
  questflags, spawnmutations, mutationchance, LLM profile sub-form.
- **Overrides & flags** — carry_capacity, health_max, stamina_max,
  corpse_name/corpse_description, hide_equipment_slots, charm_immune,
  non_combatant, player_attack_immune, pack_flee_immune. The legacy
  `hostile:` field is NOT exposed — auto_aggro is the canonical field
  (§5).

**Test Spawn control:** zone dropdown + room picker (room list for the
chosen zone with id + title; typing an ID confirms the room name before
enabling the button) → `Build.Mob.Spawn` → toast with result.

## 5. Errors & guardrails

- Server-authoritative validation; every rejection is a `buildErr` naming
  the field and listing valid values (vendor-category style).
- Client renders toasts; the form never blocks on client-side validation
  alone.
- Reference fields (schedule, patrol, species, buffs, items, archetypes,
  recipes) are **dropdowns/pickers fed by the enums payload** — free text
  cannot reach them.
- `hostile:` (LegacyHostile) is never written by the editor; `auto_aggro:`
  is the only exposed aggression field. Loading a legacy mob that still has
  `hostile: true` shows it as auto_aggro (Validate() already copies it), and
  a subsequent save writes the canonical field.

## 6. Testing & verification gate

- **Unit (mobs pkg):** ValidateSpec rejections (bad schedule/patrol/buff/
  shop-item/loot-item/archetype/species refs, out-of-range numerics);
  SaveMobSpec rename removes the old file; re-zone moves the file; stub
  creation is boot-safe; DeleteMobSpec removes file + cache.
- **Unit (gmcp):** each `Build.Mob.*` handler against a fake `mobDeps` —
  admin gate, list/get/create/update round-trip, delete refused while
  referenced, spawn validates the room.
- **Boot test** after implementation (pre-push SOP) — plus a
  create-stub-then-boot check proving an unedited stub loads clean.
- **Headless end-to-end:** drive the WS→GMCP flow (login bridge →
  create → edit → save → spawn → delete) exactly as 1b/2 were verified.
- **In-game half:** after a web test-spawn, verify via telnet/harness that
  the mob exists in the target room, idles/wanders per template, and is
  killable.
- **Owed to user before prod:** browser visual/UX eyeball of the mob panel
  (I cannot drive a browser), same as 1b and 2.

## Out of scope

- Spawn lists / room `spawninfo` editing — **sub-project 4**.
- Dialogue trees, behavior trees, quests — **sub-project 5**.
- Editing live mob *instances* (the editor writes templates only; the
  instance-save shadowing SOP still applies to smoke tests).
- Fixing the nondeterministic `mapper.GetReciprocalExit` engine bug (tracked
  from 1b, unrelated to mobs).
