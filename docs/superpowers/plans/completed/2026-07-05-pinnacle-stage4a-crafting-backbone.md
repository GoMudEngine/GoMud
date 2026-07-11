# Pinnacle Items Stage 4a: The Crafting Backbone — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

> **⚠ PENDING USER CONFIRMATION (asked 2026-07-05, user away — proceeding on
> best-judgment defaults):** (1) Stage 4 is SPLIT into 4a (this — the crafting
> backbone) and 4b (Veyra dialogue + quests + gold/masterwork engine gaps).
> (2) The three engine gaps get small faithful additions, not proxies — the
> only one in 4a's scope is the **learn_only recipe flag** (Task 1). If the
> user prefers proxies, drop Task 1 and the `learn_only: true` line from the
> assembly recipes (they'd become discoverable, gated only by their rare
> components). Confirm both before executing.

**Goal:** Build the full crafting backbone for the nine pinnacle items — the
17 intermediate component items, their 17 single-skill component recipes, the
9 assembly recipes (self-craft-enforced, learn-only), and Veyra's workshop
(rooms + the 6 crafting stations + Veyra as a station-tending crafter) — so
that a character who knows the recipes and holds the materials can craft every
pinnacle item. Quest-gating, dialogue, and recipe-learning are Stage 4b.

**Architecture:** One small engine addition (a `learn_only` recipe flag that
excludes a recipe from craft-discovery so it's quest-taught only). Everything
else is data on existing systems: component items (`is_component` +
`component_tag`), recipes (`recipes/<skill>/<slug>.yaml`, `require_own_components`
already exists from Stage 1), crafting stations (all 6 needed types exist), and
a scheduled non-combatant crafter NPC. A 3-room workshop annex off the
Confluence's Artisans' Quarter houses the stations.

**Tech Stack:** Go (1 recipe-flag task), YAML data (items, recipes, rooms,
mob, schedule), existing crafting/station/schedule systems.

**Spec:** `docs/superpowers/specs/2026-07-04-pinnacle-chase-items-design.md`
(§3 Veyra frame, §4.4 skill gating, §5 bills of materials). **Refs:**
`docs/schemas/pinnacle-items.md` (reagent tags), `docs/schemas/room.md`,
`docs/schemas/schedule.md`.

**Branch:** `feature/pinnacle-stage4a-crafting-backbone` off `master`.

---

## Locked IDs (allocated 2026-07-05)

**Component items 40208-40224** (17 used; 40225 spare) in
`items/materials-40000/`. **Rooms 6438-6441** (3 workshop rooms; 1 spare).
**Veyra mob 9584** (9585 spare). Recipe IDs are **string slugs** (no int
allocation) in `recipes/<skill>/<slug>.yaml`.

### The 17 new component items + their recipes

| ID | Component | Recipe skill (50) | Serves | assembly-tag |
|---|---|---|---|---|
| 40208 | Reinforced Harness | tailoring | Bandolier | reinforced-harness |
| 40209 | Preservation Runes | enchanting | Bandolier | preservation-runes |
| 40210 | Hungering Guard | jewelcrafting | Blackrazor | hungering-guard |
| 40211 | Obsidian Edge-Resin | alchemy | Blackrazor | obsidian-edge-resin |
| 40212 | Reinforced Frame | blacksmithing | Pack | reinforced-frame |
| 40213 | Spatial Stitching | enchanting | Pack | spatial-stitching |
| 40214 | Voice-Amber Housing | jewelcrafting | Aegis | voice-amber-housing |
| 40215 | Resonance Lacquer | alchemy | Aegis | resonance-lacquer |
| 40216 | Barbed Spike-Plates | blacksmithing | Harness | barbed-spike-plates |
| 40217 | Anti-Corrosion Quench | alchemy | Harness | anti-corrosion-quench |
| 40218 | Containment Lattice | enchanting | Prism | containment-lattice |
| 40219 | Nutrient Suspension | alchemy | Prism | nutrient-suspension |
| 40220 | Quicksilver Soles | alchemy | Treads | quicksilver-soles |
| 40221 | Windlace Bindings | enchanting | Treads | windlace-bindings |
| 40222 | Conductor Core | blacksmithing | Staff | conductor-core |
| 40223 | Choir-Focus Gems | jewelcrafting | Staff | choir-focus-gems |
| 40224 | Reduction Base | cooking | Phial | reduction-base |

The **Phial's second component is the existing Crystalline Decanter 40045**
(jewelcrafting item — verify its component_tag at build time; do NOT recreate).

### The 9 assembly recipes (65+ primary skill, `require_own_components: true`, `learn_only: true`)

Each consumes its 2 components + its ultra-rare reagents (by Stage-3
component_tag) + bulk existing goods. Reagent tags (from
`docs/schemas/pinnacle-items.md`):

| Item (output id) | Primary skill 65 | Components | Ultra-rare reagent tags |
|---|---|---|---|
| Vitalis Bandolier (40182) | alchemy | reinforced-harness, preservation-runes | chrysalis-filter-membrane, still-glass-rosette |
| The Blackrazor (40183) | blacksmithing | hungering-guard, obsidian-edge-resin | void-quenched-obsidian, grey-relic (40166), whisper-old-white |
| Wayfarer's Pack (40184) | tailoring | reinforced-frame, spatial-stitching | folded-space-silk, warden-chassis-loom |
| Aegis of Mockery (40185) | blacksmithing | voice-amber-housing, resonance-lacquer | resonant-vox-core, mockingbird-amber |
| Thornwall Harness (40186) | tailoring | barbed-spike-plates, anti-corrosion-quench | ironwood-thorn-heart, scab-chitin-plate |
| Seething Prism (40187) | jewelcrafting | containment-lattice, nutrient-suspension | seed-crystal-breach, bloom-saturated-geode |
| Zephyr Treads (40188) | tailoring | quicksilver-soles, windlace-bindings | stormfront-residue, gale-sinew-steppe |
| Hollow Choir Staff (40189) | enchanting | conductor-core, choir-focus-gems | hollowed-voice-box, chorus-shard |
| Phial of Second Birth (40181) | alchemy | reduction-base, <decanter tag 40045> | unmaking-distillate, first-bloom-nectar |

(**Verify `grey-relic` is 40166's real component_tag** at build time —
the recon reported it; confirm before writing the Blackrazor recipe.)

---

### Task 1: Engine — `learn_only` recipe flag

Pinnacle recipes must NOT be craft-discoverable (a 65-skill crafter would
otherwise auto-discover the Blackrazor assembly, bypassing Veyra's
commission). Add a flag that excludes a recipe from discovery; it can only be
taught via the quest engine's `learn_recipe` action (Stage 4b).

**Files:**
- Modify: `internal/crafting/crafting.go` (RecipeSpec field + GetEligibleRecipes filter)
- Test: `internal/crafting/crafting_learn_only_test.go` (new)

- [ ] **Step 1: Write the failing test**

```go
package crafting

import "testing"

func TestLearnOnlyExcludedFromDiscovery(t *testing.T) {
	defer seedRecipesForTest(map[string]*RecipeSpec{
		"normal-x": {RecipeId: "normal-x", Name: "Normal X", Skill: "blacksmithing", SkillMinimum: 10},
		"pinnacle-x": {RecipeId: "pinnacle-x", Name: "Pinnacle X", Skill: "blacksmithing", SkillMinimum: 65, LearnOnly: true},
	})()

	// A blacksmith at skill 70 who knows neither: discovery offers normal-x, never pinnacle-x.
	eligible := GetEligibleRecipes("blacksmithing", 70, map[string]bool{})
	sawNormal, sawPinnacle := false, false
	for _, r := range eligible {
		if r.RecipeId == "normal-x" { sawNormal = true }
		if r.RecipeId == "pinnacle-x" { sawPinnacle = true }
	}
	if !sawNormal { t.Fatal("normal recipe should be discoverable") }
	if sawPinnacle { t.Fatal("learn_only recipe must NOT be discoverable") }
}
```

(Adapt to the real `GetEligibleRecipes` signature — read crafting.go:290-308
first; the args/known-map shape may differ. Find/confirm the recipe test
seeder (`crafting_test.go` has one — match its name/pattern). If knowing a
recipe is tracked differently, mirror the real API.)

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/crafting/ -run TestLearnOnlyExcludedFromDiscovery -v`
Expected: FAIL (LearnOnly undefined)

- [ ] **Step 3: Implement**

RecipeSpec (crafting.go, near RequireOwnComponents):
```go
	LearnOnly bool `yaml:"learn_only,omitempty"` // excluded from craft-discovery; taught only via quest learn_recipe
```
In `GetEligibleRecipes` (crafting.go:290-308), skip recipes where
`recipe.LearnOnly` (a learn_only recipe is never offered as a discovery
candidate; it can still be crafted once explicitly learned). Confirm the
craft path itself (`craft.go`) checks "known" not "eligible" so a taught
learn_only recipe still crafts.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/crafting/ ./internal/usercommands/ -count=1` → green;
`go build ./...` clean.

- [ ] **Step 5: Commit**

```bash
git add internal/crafting/
git commit -m "feat(pinnacle): learn_only recipe flag (pinnacle recipes are quest-taught, not discoverable)"
```

---

### Task 2: The 17 component item YAMLs (40208-40224)

**Files:** Create 17 YAMLs in `_datafiles/world/dogmud/items/materials-40000/`
(filenames = `4020X-<ConvertForFilename(name)>.yaml`).

- [ ] **Step 1: Author each** to the component schema (study
`40018-steel_ingot.yaml` + a Stage-3 reagent like
`40190-chrysalis_filter_membrane.yaml`):
```yaml
itemid: 4020X
name: <Component Name>
namesimple: <one noun>
vendor_categories:
- <the recipe skill that MAKES it>
description: >-
  <AUTHORED — folded >-, ~68-72 col, world.md voice, no numbers, no
  lore-boundary leaks. These are intermediate crafted goods (a rune-plate,
  a resin, a lattice) — describe the crafted object.>
type: object
subtype: mundane
component_tag: <assembly-tag from the table>
weight: 0.1-1.0
value: 400-900
rarity_tier: 78
is_component: true
```
Use the exact `component_tag`s from the components table. rarity_tier 78
(a notch below the 82 reagents — these are craftable intermediates, not
world-drops). Brief per component (2-3 sensory lines each): e.g. Hungering
Guard = a jeweled crossguard set with a small hollow that seems to draw the
eye in; Obsidian Edge-Resin = a jar of black volcanic resin that never fully
sets; Containment Lattice = a fine enchanted cage of silver wire that hums;
Reduction Base = a cooked-down culinary reduction, bitter and clarifying;
etc. (author to each name).

- [ ] **Step 2: Boot smoke** — itemLoadedCount +17 (→421), zero panics, killed.
- [ ] **Step 3: Commit**

```bash
git add "_datafiles/world/dogmud/items/materials-40000/"
git commit -m "content(pinnacle): 17 intermediate component items (40208-40224)"
```

---

### Task 3: The 17 component recipes (single-skill 50)

**Files:** Create 17 recipe YAMLs in `_datafiles/world/dogmud/recipes/<skill>/`
(one per component, filed under its skill dir). Recipe schema (study
`recipes/blacksmithing/steel-longsword.yaml`):
```yaml
id: <component-slug>
name: <Component Name>
skill: <skill>
skill_minimum: 50
station: <forge|alchemy_bench|loom|jeweler_bench|enchanting_circle|cooking_fire>
time_rounds: 8
ingredients:
  - item_tag: <existing raw-material tag>
    quantity: <n>
  # ... 2-4 existing materials
output:
  item_id: <4020X>
  quantity: 1
success_message: "..."
failure_message: "..."
```

- [ ] **Step 1: Verify existing raw-material tags.** These recipes consume
EXISTING materials (ingots, hides, gems, gems, alchemy reagents, silk, thread).
Grep `component_tag:` across `items/materials-40000/` to get the real tags
(e.g. steel-ingot, iron-ingot, gold-ingot, silver-ingot, leather-strip,
cured-hide, cut-gem, garnet, silk-bolt, etc.). Record the ones you'll use.
Do NOT invent material tags — use ones that exist. If a needed material
doesn't exist (e.g. "garnet"), substitute the nearest existing gem/material.

- [ ] **Step 2: Author the 17 recipes.** skill_minimum 50, station per the
component's skill, indicative existing ingredients per the spec's component
notes (§5). station map: tailoring→loom, enchanting→enchanting_circle,
jewelcrafting→jeweler_bench, alchemy→alchemy_bench, blacksmithing→forge,
cooking→cooking_fire. NO `learn_only` on component recipes — components ARE
discoverable/craftable by any 50-skill crafter (only the ASSEMBLY is
Veyra-gated). NO `require_own_components` on component recipes.

- [ ] **Step 3: Boot smoke** — recipes load (grep the loader log for recipe
count up by 17), zero panics, killed. (A recipe with a bad item_tag or
missing output item won't panic at load but will fail validation — check the
boot log for recipe validation warnings/errors.)

- [ ] **Step 4: Commit**

```bash
git add "_datafiles/world/dogmud/recipes/"
git commit -m "content(pinnacle): 17 component recipes (single-skill 50)"
```

---

### Task 4: The 9 assembly recipes (65+, require_own_components, learn_only)

**Files:** Create 9 recipe YAMLs in `recipes/<primary-skill>/` (slugs like
`assemble-the-blackrazor`).
```yaml
id: assemble-<item-slug>
name: <Item Name>
skill: <primary skill>
skill_minimum: 65
station: <station for the primary skill>
require_own_components: true
learn_only: true
time_rounds: 20
ingredients:
  - item_tag: <component A tag>
    quantity: 1
  - item_tag: <component B tag>
    quantity: 1
  - item_tag: <ultra-rare reagent tag 1>
    quantity: 1
  - item_tag: <ultra-rare reagent tag 2>
    quantity: 1
  # + reagent 3 where applicable (Blackrazor has 3)
  # + bulk existing goods (e.g. iron-ingot x40, coal x60) per spec §5 bills
output:
  item_id: <pinnacle item id 40181-40189>
  quantity: 1
success_message: "..."
failure_message: "..."
```

- [ ] **Step 1: Author the 9 assembly recipes** using the assembly table
above (components + reagent tags + bulk). Verify `grey-relic` (40166's tag)
for the Blackrazor. Bulk goods per the spec §5 "Bulk (indicative)" lines
(iron ingots, coal, cured hides, etc.) — use real existing tags. Each sets
BOTH `require_own_components: true` (components must carry the assembler's
MakerName — the self-craft enforcement) AND `learn_only: true` (Veyra teaches
it in 4b). time_rounds ~20 (the marquee craft).

- [ ] **Step 2: Boot smoke** — recipes load (+9), zero panics. Confirm the
assembly recipes validate (output item ids 40181-40189 exist; all ingredient
tags resolve to real component_tags — a typo'd reagent tag = an
uncraftable item). Killed.

- [ ] **Step 3: Commit**

```bash
git add "_datafiles/world/dogmud/recipes/"
git commit -m "content(pinnacle): 9 assembly recipes (65+, require_own_components, learn_only)"
```

---

### Task 5: Veyra's workshop — 3 rooms + the 6 stations, off the Confluence

**Files:**
- Create: `_datafiles/world/dogmud/rooms/the_confluence/6438.yaml` … `6440.yaml`
- Modify: `_datafiles/world/dogmud/rooms/the_confluence/6233.yaml` (Upper Craft Row — add the workshop entrance)

- [ ] **Step 1: The entrance** on 6233 Upper Craft Row (free exits: north,
up, down). READ 6233 first. Add an exit (e.g. `up` — an upper-floor workshop,
or `north`) → 6438, reciprocal back. NOT secret — Veyra is discoverable
(design pillar 1: visible early). A doorway with a quasi-legal hint ("a
narrow stair to a shuttered upper workshop; the sign is painted over").

- [ ] **Step 2: The 3 workshop rooms** (6438-6440), `zone: The Confluence`,
`biome: city` (or `indoor` — match Artisans' Quarter rooms). Distribute the 6
stations across the 3 rooms (one `station:` per room, but a room has only ONE
station field — so with 3 rooms you cover 3 stations; the other 3 disciplines
must reuse the Confluence's… wait, the Confluence has NO working stations).

  **RESOLUTION:** a room can carry only one `station`. Veyra's workshop needs
  all 6 (blacksmithing/alchemy/tailoring/jewelcrafting/enchanting/cooking).
  Allocate the 4th room (6441, from the spare) and lay out the workshop as
  **4 rooms covering the primary disciplines** most items need
  (forge, alchemy_bench, loom, jeweler_bench — these cover 8 of 9 items'
  primary skills) + note that enchanting (Staff assembly) and cooking (Phial
  component) stations are ALSO needed. So: use ALL 4 allocated rooms
  (6438-6441) with 4 stations, and add a 5th+6th... — this needs 6 rooms for 6
  stations. **DECISION for the builder:** either (a) request 2 more room IDs
  (id_inventory --alloc rooms 2) to make a 6-room workshop (one station each),
  or (b) place the 6 stations across 6438-6441 + reuse 2 existing
  Artisans'-Quarter rooms by ADDING a `station:` field to them (e.g. add
  `station: forge` to 6239 The Smithy which is already forge-flavored, and
  `station: loom` to 6235 The Weaver's). Option (b) is cleaner lore-wise (the
  Quarter's flavored rooms become functional) and uses fewer new rooms — but
  it makes those stations public (not Veyra-gated), which is FINE (stations
  aren't the gate; the recipes + require_own_components + learn_only are).
  **Recommend (b):** Veyra's 3-4 annex rooms carry the rarer stations
  (jeweler_bench, enchanting_circle, alchemy_bench, cooking_fire) + Veyra
  herself; add forge + loom to the existing flavored Quarter rooms. Confirm
  the exact station-to-room layout at build time and document it.

- [ ] **Step 3: Author the rooms** — title, description (world.md voice,
Veyra's quasi-legal workshop — cluttered with half-finished convergence
works, the smell of six crafts at once), exits, nouns, idlemessages. One room
is Veyra's (her spawn + the room she tends).

- [ ] **Step 4: Boot smoke** — rooms load, stations present (no station
validation errors), cartcheck errors=0 (Confluence is Cartesian — the annex
must be coordinate-consistent OR use a non-spatial/named exit), killed.

- [ ] **Step 5: Commit**

```bash
git add "_datafiles/world/dogmud/rooms/the_confluence/"
git commit -m "content(pinnacle): Veyra's workshop rooms + crafting stations off the Confluence"
```

---

### Task 6: Veyra Coil-Tongue — the crafter NPC (mob 9584 + schedule)

**Files:**
- Create: `_datafiles/world/dogmud/mobs/the_confluence/9584-veyra_coil_tongue.yaml`
- Create: `_datafiles/world/dogmud/schedules/the_confluence/veyra.yaml`
- Modify: the workshop room YAML (add Veyra's `spawninfo`)

- [ ] **Step 1: Veyra mob** (model on `mobs/pothole_coulee/9116-smith_rusk.yaml`
+ `mobs/the_confluence/9476-the_smith.yaml`):
```yaml
mobid: 9584
zone: The Confluence
schedule_id: veyra
behavior_archetype: noncombat_questgiver
non_combatant: true
charm_immune: true
hostile: false
statpool: 30
maxwander: 0
crafter: true          # tends her stations (verify field name vs smith_rusk)
character:
  name: Veyra Coil-Tongue
  description: |
    <AUTHORED — a reclusive legendary artificer; quasi-legal; sole keeper of
    the convergence crafts. She asks few questions and answers fewer. No
    lore-boundary leaks.>
  speciesid: 1
```
(NO dialogue/quest wiring here — that's 4b. She exists as a station-tending
NPC now; her dialogue file `dialogue/the_confluence/9584.yaml` comes in 4b.)

- [ ] **Step 2: Schedule** (model on `schedules/the_confluence/cf_smith.yaml`)
— a stationary work/sleep schedule pinned to her workshop room, covering all
24h (validators panic on gaps). `activity: craft` during work hours so she
visibly works.

- [ ] **Step 3: Spawn** — add Veyra's `spawninfo` (mobid 9584, respawnrate) to
her workshop room.

- [ ] **Step 4: Boot smoke** — Veyra loads (mobs +1 → 611), her schedule
validates (no coverage-gap panic, target_room reachable), she's non-combatant,
zero panics, killed.

- [ ] **Step 5: Commit**

```bash
git add "_datafiles/world/dogmud/mobs/the_confluence/9584-veyra_coil_tongue.yaml" "_datafiles/world/dogmud/schedules/the_confluence/veyra.yaml" "_datafiles/world/dogmud/rooms/the_confluence/"
git commit -m "content(pinnacle): Veyra Coil-Tongue crafter NPC + schedule"
```

---

### Task 7: Full-suite + boot + end-to-end craft verification + world-critic

- [ ] **Step 1:** `go test -timeout 300s -count=1 ./...` → green (note the
pre-existing TestAttackInCombat grapple flake if it surfaces — unrelated).
- [ ] **Step 2:** Instance-wipe + boot: items 421 (404 + 17), mobs 611, rooms
up ~3-4, all recipes load + validate, Veyra + schedule OK, cartcheck errors=0,
zero panics. Kill + verify + delete.
- [ ] **Step 3: End-to-end craft test (the load-bearing verification).** Via
the harness or a scripted admin session: give an admin character skill 65 in a
test discipline, admin-`learn` an assembly recipe (learn_only, so it must be
explicitly learned) + its 2 component recipes, spawn the raw materials + the
ultra-rare reagents, craft both components at the stations (confirm they get
the crafter's MakerName), then assemble the pinnacle item (confirm
require_own_components passes with own components, and the pinnacle item is
produced). Do this for AT LEAST ONE item (e.g. the Wayfarer's Pack — cheapest,
tailoring). Confirm require_own_components REFUSES if a component is
foreign-made (spawn a foreign-MakerName component and confirm the refusal
message). This proves the whole backbone works end-to-end.
- [ ] **Step 4: World-critic** — the 17 component descriptions + workshop rooms
+ Veyra's description against world.md (lore leaks, register, 78-col, no
numbers). Fix inline; re-boot if any room/mob YAML changed.
- [ ] **Step 5: Schema-doc update** — append a "Stage 4a: recipes" section to
`docs/schemas/pinnacle-items.md`: the component→recipe→assembly map + the
station layout + the learn_only flag note (so 4b's quests know which recipe
slugs to teach via learn_recipe).
- [ ] **Step 6: Commit**

```bash
git add _datafiles/ internal/ docs/schemas/pinnacle-items.md
git commit -m "content(pinnacle): Stage 4a world-critic polish + recipe/station registry"
```

---

## Explicitly OUT of Stage 4a (→ Stage 4b)

- Veyra's dialogue tree, the intro quest "The Convergence", the 9 commission
  quests.
- The quest-gated recipe LEARNING (learn_recipe actions on the commission
  quests) — in 4a, the recipes exist + are admin-learnable/testable but no
  player path teaches them yet.
- The two 4b engine gaps: `charge_gold` action + `has_gold` condition (staged
  fees); the masterwork/skill entry-gate condition.
- The staged gold fees, the "show a masterwork" gate, the truth-knower variant
  lines.

## Notes / build-time decisions to resolve

- **Station layout** (Task 5 Step 2): recommend adding `station:` to 1-2
  existing Artisans'-Quarter rooms (The Smithy→forge, The Weaver's→loom) +
  Veyra's annex rooms for the rarer 4 stations. Confirm + document.
- **Existing material tags** (Task 3 Step 1): verify every raw-material
  `item_tag` exists before writing recipes; substitute nearest existing where
  a spec-named material is absent.
- **`grey-relic`** (40166's tag): verify before the Blackrazor assembly recipe.
- **Crystalline Decanter 40045** tag: verify for the Phial assembly.
