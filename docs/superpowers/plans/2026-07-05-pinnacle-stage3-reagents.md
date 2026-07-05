# Pinnacle Items Stage 3: The Reagents — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Seed the 18 ultra-rare crafting reagents that the nine pinnacle
items require into the world — via existing endgame mob drops, a new
zone+weather forage overlay, and one new giant-spider zone (The Foldweave)
that drops Folded-Space Silk.

**Architecture:** One small engine primitive (a player-only zone+weather
forage-yield overlay), 18 reagent item YAMLs, low-chance `character.items`
drops appended to five existing endgame mobs, forage/weather placements in
the new overlay tables, and a new ~12-room `non_cartesian` cave zone with a
giant-spider ecology reachable via a hidden search-gated descent off the
Fernway South.

**Tech Stack:** Go (1 forage-overlay task), YAML data (items, mobs, rooms,
zone-config), the mapper's non_cartesian exemption, harness playtest.

**Spec:** `docs/superpowers/specs/2026-07-04-pinnacle-chase-items-design.md`
(§5 bills of materials, §7 placement). **Author refs:**
`docs/schemas/pinnacle-items.md`, `docs/schemas/item.md`.

**Branch:** `feature/pinnacle-stage3-reagents` off `master`.

```bash
git checkout master && git checkout -b feature/pinnacle-stage3-reagents
```

---

## Locked IDs (allocated via id_inventory 2026-07-05 — do NOT reassign)

**Reagent items 40190-40207** (all flat in
`_datafiles/world/dogmud/items/materials-40000/` — ItemFolder hardcodes the
40000 band; NO subdir). **Rooms 6426-6441** (The Foldweave; ~12 used +
spares). **Mobs 9580-9587** (Foldweave spider ecology).

| ID | Reagent | Serves item | Source | vendor_cat (best-guess; Stage 4 finalizes) |
|---|---|---|---|---|
| 40190 | Chrysalis Filter-Membrane | Vitalis Bandolier | Core Guardian (9562) | alchemy |
| 40191 | Resonant Vox-Core | Aegis of Mockery | Core Guardian (9562) | jewelcrafting |
| 40192 | Hollowed Voice-Box | Hollow Choir Staff | Core Guardian (9562) | enchanting |
| 40193 | Unmaking Distillate | Phial of Second Birth | Core Guardian (9562) | alchemy |
| 40194 | Seed-Crystal of the Breach | Seething Prism | Core Guardian (9562) | jewelcrafting |
| 40195 | Void-Quenched Obsidian Core | The Blackrazor | Warden-Prime (9561) | blacksmithing |
| 40196 | Warden Chassis-Loom | Wayfarer's Pack | Warden-Prime (9561) | tailoring |
| 40197 | Whisper of the Old White | The Blackrazor | The Old White (9570) | blacksmithing |
| 40198 | Still-Glass Rosette | Vitalis Bandolier | Stillwater Marsh forage | alchemy |
| 40199 | Mockingbird Amber | Aegis of Mockery | Ironwind Steppe forage | jewelcrafting |
| 40200 | Ironwood Thorn-Heart | Thornwall Harness | The Fernway South forage | tailoring |
| 40201 | Bloom-Saturated Geode | Seething Prism | Labyrinth of Low Tunnels forage | jewelcrafting |
| 40202 | First-Bloom Nectar | Phial of Second Birth | Stillwater Marsh forage | alchemy |
| 40203 | Chorus-Shard | Hollow Choir Staff | Confluence forage | enchanting |
| 40204 | Stormfront Residue | Zephyr Treads | storm-weather forage (mountains) | alchemy |
| 40205 | Scab-Chitin Plate | Thornwall Harness | A Pale Creeper drop (9568) | tailoring |
| 40206 | Gale-Sinew of the Steppe | Zephyr Treads | Windscour Wyrm drop (229) | tailoring |
| 40207 | Folded-Space Silk | Wayfarer's Pack | The Foldweaver drop (new zone) | tailoring |

**The Blackrazor's third ultra-rare is the EXISTING item 40166 "Pale-Grey
Casting"** (already dropped by The Sentinel 9552 @3%). Do NOT create it —
Stage 4's Blackrazor recipe references 40166.

## Reagent item schema (verified against 40171-hull_filament.yaml)

Every reagent YAML:
```yaml
itemid: 4019X
name: <Reagent Name>
namesimple: <one noun>
vendor_categories:
- <craft skill from table>
description: >-
  <AUTHORED — see per-reagent brief; folded >-, ~68-72 col, world.md voice,
  no numbers, no lore-boundary leaks. "Chrysalis"/"grey material"/named
  wardens-as-known-things OK; the cosmic truth NOT.>
type: object
subtype: mundane
component_tag: <kebab-case-tag>
weight: 0.1-0.6
value: 1200-1400
rarity_tier: 82
is_component: true
```
`component_tag` (kebab-case) is what Stage 4 recipes match. NO
`salvage_returns` (not a real field). Filename =
`4019X-<ConvertForFilename(name)>.yaml`.

---

### Task 1: Engine — player-only zone+weather forage overlay

Forage yields are Go-hardcoded by biome (`internal/forager/forage_core.go`
`ForageYields`), with no zone-exclusivity or weather-gating. Add two overlay
maps, spliced exactly like the existing `NightForageYields`, fed by new
`ForageAttempt` fields. **Player forage path only** — the NPC-forager path
must NOT pass zone/weather, so these ultra-rares can never be foraged by
NPCs and leaked into vendor stock.

**Files:**
- Modify: `internal/forager/forage_core.go` (ForageAttempt + 2 maps + splice)
- Modify: `internal/actions/forage.go` (thread room.Zone + GetWeather export)
- Test: `internal/forager/forage_overlay_test.go` (new)

- [ ] **Step 1: Write the failing test**

```go
package forager

import "testing"

func TestZoneAndStormForageOverlays(t *testing.T) {
	// Seed overlay tables for the test (these are package vars).
	ZoneForageYields["Test Zone"] = []int{999001}
	StormForageYields["swamp"] = []int{999002}
	defer func() {
		delete(ZoneForageYields, "Test Zone")
		delete(StormForageYields, "swamp")
	}()

	// A zone-forage item is reachable only when the zone matches.
	if !poolContains(buildForagePool("swamp", "Test Zone", "clear", false), 999001) {
		t.Fatal("zone overlay item missing when zone matches")
	}
	if poolContains(buildForagePool("swamp", "Other Zone", "clear", false), 999001) {
		t.Fatal("zone overlay item leaked into a non-matching zone")
	}
	// A storm-forage item is reachable only during a storm in that biome.
	if !poolContains(buildForagePool("swamp", "Other Zone", "storm", false), 999002) {
		t.Fatal("storm overlay item missing during storm")
	}
	if poolContains(buildForagePool("swamp", "Other Zone", "clear", false), 999002) {
		t.Fatal("storm overlay item leaked in clear weather")
	}
}

func poolContains(p []int, id int) bool {
	for _, x := range p {
		if x == id {
			return true
		}
	}
	return false
}
```

(This test assumes a small extracted helper `buildForagePool(biome, zone,
weather string, atNight bool) []int` that assembles the candidate slice —
extract it from ForageCore so both the RNG pick and this test use it. If you
prefer not to extract, adapt the test to call ForageCore many times and
assert reachability statistically — but the pure helper is cleaner and
deterministic. Match the real biome/zone/weather string types.)

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/forager/ -run TestZoneAndStormForageOverlays -v`
Expected: FAIL (funcs/maps undefined)

- [ ] **Step 3: Implement in forage_core.go**

Add package vars beside `NightForageYields`:
```go
// ZoneForageYields adds zone-exclusive forageables (keyed by zone display
// name). Appended only when the player forages in that exact zone. Used for
// the pinnacle ultra-rare reagents (single entry among many biome commons =
// rarest outcome). Player-forage only — NOT applied to NPC foragers.
var ZoneForageYields = map[string][]int{}

// StormForageYields adds weather-gated forageables (keyed by biome), appended
// only when the zone's current weather is "storm". Player-forage only.
var StormForageYields = map[string][]int{}
```

Extract the pool assembly (mirror the NightForageYields splice):
```go
// buildForagePool assembles the candidate yield slice for a forage attempt:
// the biome base + night overlay + zone overlay + storm overlay. Duplicate
// entries raise probability; a single appended ultra-rare is the rarest
// possible outcome.
func buildForagePool(biome, zone, weather string, atNight bool) []int {
	base := ForageYields[biome]
	pool := append([]int{}, base...)
	if atNight {
		pool = append(pool, NightForageYields[biome]...)
	}
	if zone != "" {
		pool = append(pool, ZoneForageYields[zone]...)
	}
	if weather == "storm" {
		pool = append(pool, StormForageYields[biome]...)
	}
	return pool
}
```

Add the fields to `ForageAttempt` (Zone, Weather strings) and rewrite
`ForageCore` to use `buildForagePool(a.Biome, a.Zone, a.Weather, a.AtNight)`
in place of the inline `yields` assembly — preserving the existing
difficulty-gate and `yields[util.Rand(len(yields))]` pick. An empty pool
(no biome match) still returns `ForageResult{}`.

- [ ] **Step 4: Thread zone+weather in the PLAYER path only** (`internal/actions/forage.go`)

At the `forager.ForageCore(...)` call, add Zone from `room.Zone` and Weather
via the weather plugin export (the pattern the playtest module uses):
```go
	weatherType := ""
	if f, ok := plugins.GetPluginRegistry().GetExportedFunction("GetWeather"); ok {
		if getW, ok := f.(func(string) map[string]any); ok {
			if wx := getW(room.Zone); wx != nil {
				weatherType, _ = wx["type"].(string)
			}
		}
	}
	coreResult := forager.ForageCore(forager.ForageAttempt{
		Biome:       biome.BiomeId,
		SearchScore: searchScore,
		AtNight:     gametime.IsNight(),
		Zone:        room.Zone,
		Weather:     weatherType,
	})
```
(Verify `plugins.GetPluginRegistry().GetExportedFunction` import + signature
against `modules/playtest/beacons.go:31` and `go.go`'s SendGMCPEvent use.
Verify `room.Zone` field access via `actor.GetRoom()` return type. Do NOT
touch `internal/behaviortree/actions_forager.go` — leaving its ForageAttempt
without Zone/Weather is the intentional guard against NPC forage leaking the
reagents.)

- [ ] **Step 5: Run + build**

Run: `go test ./internal/forager/ ./internal/actions/ -count=1` → green;
`go build ./...` clean.

- [ ] **Step 6: Commit**

```bash
git add internal/forager/ internal/actions/forage.go
git commit -m "feat(pinnacle): player-only zone+weather forage overlay (Stage 3 reagent placement)"
```

---

### Task 2: The 10 mob-drop reagent items (40190-40197, 40205, 40206)

Author the reagent YAMLs that drop from mobs. (Forage reagents + silk are
Task 5.) One file each; descriptions authored to the briefs below.

**Files:** Create 10 YAMLs in `_datafiles/world/dogmud/items/materials-40000/`:
`40190-chrysalis_filter_membrane.yaml`, `40191-resonant_vox_core.yaml`,
`40192-hollowed_voice_box.yaml`, `40193-unmaking_distillate.yaml`,
`40194-seed_crystal_of_the_breach.yaml`,
`40195-void_quenched_obsidian_core.yaml`, `40196-warden_chassis_loom.yaml`,
`40197-whisper_of_the_old_white.yaml`, `40205-scab_chitin_plate.yaml`,
`40206-gale_sinew_of_the_steppe.yaml`.

- [ ] **Step 1: Author each YAML** to the schema above. component_tags
(kebab-case, distinct): `chrysalis-filter-membrane`, `resonant-vox-core`,
`hollowed-voice-box`, `unmaking-distillate`, `seed-crystal-breach`,
`void-quenched-obsidian`, `warden-chassis-loom`, `whisper-old-white`,
`scab-chitin-plate`, `gale-sinew-steppe`. Description briefs (5-8 lines each):
  - **Chrysalis Filter-Membrane**: a translucent grey membrane that holds a
    breath of liquid at its perfect moment; harvested from the Core Guardian.
  - **Resonant Vox-Core**: a fist-sized resonating node that catches sound
    and throws it back louder; torn from a warden's chest.
  - **Hollowed Voice-Box**: a hollow grey throat that hums a chord below
    hearing; taken from the Core Guardian.
  - **Unmaking Distillate**: a phial of grey solvent that dissolves whatever
    the Chrysalis wrote; sister to the Catalyst of Unmaking.
  - **Seed-Crystal of the Breach**: a wet, warm crystal seed that stirs
    Chrysalis-matter around it; from the Breach's core.
  - **Void-Quenched Obsidian Core**: a black core quenched in a cold the
    smiths won't name; the heart of a Warden-Prime.
  - **Warden Chassis-Loom**: a machined grey frame, too clean-jointed to
    have come off any living loom.
  - **Whisper of the Old White**: something between a scale and a sound,
    lifted from the great pale thing in the deep sewers.
  - **Scab-Chitin Plate**: a scabbed, mineral-crusted carapace plate,
    wrong-jointed and warm.
  - **Gale-Sinew of the Steppe**: a length of wind-cured sinew from the
    apex of the Windscour, still faintly humming with motion.

- [ ] **Step 2: Boot smoke** (as every content task): wipe instance saves,
`go build -o gomud_smoke.exe . && ./gomud_smoke.exe` (background ~90s),
confirm itemLoadedCount +10 (→396), zero panics, Server Ready; then
`taskkill /IM gomud_smoke.exe /F` + `Get-Process gomud_smoke` returns
NOTHING; delete the exe + any log.

- [ ] **Step 3: Commit**

```bash
git add "_datafiles/world/dogmud/items/materials-40000/"
git commit -m "content(pinnacle): 10 mob-drop reagent items (40190-40197, 40205, 40206)"
```

---

### Task 3: Append reagent drops to 5 existing endgame mobs

Add one `character.items` entry per reagent at a low `dropchance` (percent).
Verified field: `character.items[].dropchance` is chance-in-100.

**Files (modify `character.items:` lists):**
- `_datafiles/world/dogmud/mobs/crash_site_interior/9562-the_core_guardian.yaml` — add 40190, 40191, 40192, 40193, 40194 each `dropchance: 4`
- `_datafiles/world/dogmud/mobs/crash_site_interior/9561-warden_prime.yaml` — add 40195, 40196 each `dropchance: 5`
- `_datafiles/world/dogmud/mobs/new_plymouth_sewers/9570-the_old_white.yaml` — add 40197 `dropchance: 4`
- `_datafiles/world/dogmud/mobs/new_plymouth_sewers/9568-a_pale_creeper.yaml` — add 40205 `dropchance: 5` (Scab-Chitin Plate; statpool 400 aberration — thematically the "scab-chitin" carapace)
- `_datafiles/world/dogmud/mobs/ironwind_steppe/229-windscour_wyrm.yaml` — add 40206 `dropchance: 5` (Gale-Sinew; already carries a rare-drop entry to mirror)

- [ ] **Step 1: Edit each mob** — append to the existing `character.items:`
list (do not disturb existing entries), e.g. on the Core Guardian:
```yaml
character:
  items:
    # ...existing entries unchanged...
    - itemid: 40190
      dropchance: 4
    - itemid: 40191
      dropchance: 4
    - itemid: 40192
      dropchance: 4
    - itemid: 40193
      dropchance: 4
    - itemid: 40194
      dropchance: 4
```
(READ each mob file first; match its exact indentation/entry style — some
use flow `{itemid: X, dropchance: Y}`, some block. Preserve the file's
style. Do NOT touch `loot_pool` — that's affix gear, not reagents.)

- [ ] **Step 2: Boot smoke** (as Task 2 Step 2): mobs count unchanged (606),
zero panics — the drops don't change mob count, this proves the YAML still
parses. Kill + verify + delete.

- [ ] **Step 3: Commit**

```bash
git add "_datafiles/world/dogmud/mobs/"
git commit -m "content(pinnacle): reagent drops on Core Guardian, Warden-Prime, Old White, Pale Creeper, Windscour Wyrm"
```

---

### Task 4: The 7 forage/weather reagent items (40198-40204)

Author the forage-sourced reagent YAMLs (same schema as Task 2).

**Files:** Create 7 YAMLs: `40198-still_glass_rosette.yaml`,
`40199-mockingbird_amber.yaml`, `40200-ironwood_thorn_heart.yaml`,
`40201-bloom_saturated_geode.yaml`, `40202-first_bloom_nectar.yaml`,
`40203-chorus_shard.yaml`, `40204-stormfront_residue.yaml`.

- [ ] **Step 1: Author each** (component_tags: `still-glass-rosette`,
`mockingbird-amber`, `ironwood-thorn-heart`, `bloom-saturated-geode`,
`first-bloom-nectar`, `chorus-shard`, `stormfront-residue`). Briefs:
  - **Still-Glass Rosette**: a rosette of marsh-glass that never stops slowly
    turning; found in the still water of Stillwater Marsh.
  - **Mockingbird Amber**: amber with a trapped birdsong you can almost hear;
    from the Ironwind Steppe.
  - **Ironwood Thorn-Heart**: the dense heartwood-thorn of a Fernway ironwood,
    hard as forged iron.
  - **Bloom-Saturated Geode**: a geode whose crystals drank deep Chrysalis
    bloom; from the low tunnels.
  - **First-Bloom Nectar**: pale nectar from the first bloom of a deep-marsh
    flower; smells like rain on nothing.
  - **Chorus-Shard**: a shard that hums a chord in sympathy with running
    water; from the Confluence.
  - **Stormfront Residue**: a scrap of charged, ozone-smelling matter left in
    the wake of a highland storm.

- [ ] **Step 2: Boot smoke** (itemLoadedCount +7 → 403). Kill + verify + delete.

- [ ] **Step 3: Commit**

```bash
git add "_datafiles/world/dogmud/items/materials-40000/"
git commit -m "content(pinnacle): 7 forage/weather reagent items (40198-40204)"
```

---

### Task 5: Populate the forage overlay tables with the reagents

Now that the items (Task 4) and the overlay (Task 1) exist, wire them.
**Verify the exact zone-name strings first** by reading each zone's
`zone-config.yaml` (`name:` field) — the map keys must match byte-for-byte.

**Files:** Modify `internal/forager/forage_core.go` (populate the two maps).
- Test: extend `internal/forager/forage_overlay_test.go`.

- [ ] **Step 1: Verify zone names.** Read the `name:` from:
`rooms/stillwater_marsh/zone-config.yaml`,
`rooms/ironwind_steppe/zone-config.yaml`,
`rooms/the_fernway_south/zone-config.yaml`,
`rooms/labyrinth_of_low_tunnels/zone-config.yaml`, and the Confluence zone
(grep `name: The Confluence` under rooms/). Record the exact strings.

- [ ] **Step 2: Populate** (use the verified strings; biome for storm is
`mountains`):
```go
var ZoneForageYields = map[string][]int{
	"Stillwater Marsh":         {40198, 40202}, // Still-Glass Rosette, First-Bloom Nectar
	"Ironwind Steppe":          {40199},        // Mockingbird Amber
	"The Fernway South":        {40200},        // Ironwood Thorn-Heart
	"Labyrinth of Low Tunnels": {40201},        // Bloom-Saturated Geode
	"<exact Confluence zone>":  {40203},        // Chorus-Shard
}

var StormForageYields = map[string][]int{
	"mountains": {40204}, // Stormfront Residue — highland storms
}
```
(If the Confluence zone name has no forageable wilderness rooms, note it and
pick the nearest forageable Confluence-region zone; Chorus-Shard just needs a
water/river zone the player can forage in.)

- [ ] **Step 3: Extend the test** — assert each real reagent ID is reachable
in its zone and NOT in a different zone; Stormfront reachable in mountains
during storm, not in clear. `go test ./internal/forager/ -count=1` green.

- [ ] **Step 4: Commit**

```bash
git add internal/forager/
git commit -m "content(pinnacle): seed 6 zone-forage + 1 storm-forage reagents into the overlay"
```

---

### Task 6: The Foldweave — zone-config + rooms + hidden Fernway entrance

A ~12-room `non_cartesian` cave zone off the Fernway South, reached by a
hidden search-gated descent. Endgame difficulty, space-folding flavor. IDs:
rooms 6426-6437 (12 used; 6438-6441 spare), root/entry 6426.

**Files:**
- Create: `_datafiles/world/dogmud/rooms/the_foldweave/zone-config.yaml`
- Create: `_datafiles/world/dogmud/rooms/the_foldweave/6426.yaml` … `6437.yaml`
- Modify: `_datafiles/world/dogmud/rooms/the_fernway_south/4161.yaml` (hidden entrance)

- [ ] **Step 1: zone-config.yaml** (model on labyrinth_of_low_tunnels):
```yaml
name: The Foldweave
roomid: 6426
defaultbiome: cave
region: Windward Marches
non_cartesian: true
```
(non_cartesian exempts the maze from cartcheck collision/reciprocity panics —
verified in mapper.consistency.go. Author overlapping coords / looping exits
freely for the space-folding effect.)

- [ ] **Step 2: The hidden entrance** on Fernway South room **4161 (Pine
Stand)** — it already hints at a large predator (`wolf-sign`, "a deep cough
that is not a deer"), the perfect breadcrumb. READ 4161.yaml first, then add
a secret `down` exit into The Foldweave, plus a `hidden_nouns`/description
breadcrumb (a search-revealed "a gap between roots breathing cold, sweetish
air"):
```yaml
exits:
  # ...existing west exit unchanged...
  down:
    roomid: 6426
    secret: true
    zone: The Foldweave
```
And 6426's reciprocal `up` → 4161 (`zone: The Fernway South`). Secret exits
reveal on a Tier-1 search (DC 125 vs Perception+Search) — endgame players
clear it, wanderers don't. (Optionally add a `lock:` on the down exit for a
harder gate — the thornwall_city:487 pattern — but a plain secret is enough
for Stage 3; the danger IS the gate.)

- [ ] **Step 3: The 12 rooms** (6426-6437). Author cave rooms, `zone: The
Foldweave`, `biome: cave`, with:
  - **6426 The Foldthreshold** (entry): the descent lands here; cold sweet
    air, first webs; `up` → 4161. An unmissable "the space is wrong here"
    beat + a survivor-warning or bone-litter (no NPC needed).
  - A looping mid-section (6427-6435) of web galleries, egg-hung chambers,
    and impossible-distance tunnels — use `non_cartesian` freedom: exits that
    loop back, a corridor longer inside than out, a chamber reached by two
    different directions. Space-folding descriptions (a strand running past
    the horizon of a small cave; a room you can see the far side of but not
    reach directly). No numbers, no lore-boundary leaks.
  - **6436/6437 The Foldweaver's Core** (boss lair, 2 rooms): the apex
    Foldweaver's nest; the deepest fold. Mob spawns wired in Task 7.
  - Every room 78-col wrapped descriptions in world.md voice; `nouns:` for
    the interactable web/fold/egg features; `idlemessages:` (skittering,
    strands thrumming, something big shifting).

- [ ] **Step 4: Boot smoke + cartcheck.** Wipe instances, boot; confirm the
new zone loads (room count up ~12), `ValidateZoneConsistency errors=0`
(non_cartesian must produce zero hard errors — if it panics, a coord/exit
issue slipped the exemption; fix), zero panics, Server Ready. Kill + verify +
delete. (Optionally `cartcheck the foldweave` in-game if an admin session is
handy — but the boot validator is the gate.)

- [ ] **Step 5: Commit**

```bash
git add "_datafiles/world/dogmud/rooms/the_foldweave/" "_datafiles/world/dogmud/rooms/the_fernway_south/4161.yaml"
git commit -m "content(pinnacle): The Foldweave — non_cartesian spider cave off the Fernway"
```

---

### Task 7: The Foldweave mobs + Folded-Space Silk drop + Folded-Space Silk item

The spider ecology and the silk it yields. Mobs 9580-9587 (use ~5-6).

**Files:**
- Create: `_datafiles/world/dogmud/items/materials-40000/40207-folded_space_silk.yaml`
- Create: mob YAMLs in `_datafiles/world/dogmud/mobs/the_foldweave/` —
  e.g. `9580-web_skitterer.yaml`, `9581-fold_broodling.yaml`,
  `9582-silk_lurker.yaml`, `9583-the_foldweaver.yaml` (+ spares 9584-9587
  if you want variety/adds)
- Modify: The Foldweave room YAMLs (add `spawninfo:` lists)

- [ ] **Step 1: Folded-Space Silk item** (40207): schema as Task 2/4;
`component_tag: folded-space-silk`; brief: a length of spider-silk that holds
more length than it has, folding back on itself; a single strand fills a room
and weighs nothing. vendor_categories: tailoring.

- [ ] **Step 2: Spider mobs** (model on coulee_spider 220 for lessers,
windscour_wyrm 229 / sentinel 9552 for the boss). Endgame statpools:
  - **Web Skitterer** (9580): statpool ~150, `aiprofile: ambush_predator`,
    `groups: [cave-pack]`, `hostile: true`, low/no loot.
  - **Fold Broodling** (9581): statpool ~200, ambush, `groups: [cave-pack]`.
  - **Silk Lurker** (9582): statpool ~280, tougher, maybe a rare trickle of
    40207 at `dropchance: 1`.
  - **The Foldweaver** (9583): the apex boss — statpool ~600-800,
    `behavior_archetype: leader`, `aiprofile: brute`,
    `submission_policy: lethal`, `surrender_policy: never`, `groups:
    [cave-pack]`, spawnmutations (large / thick-hide), `combatcommands`.
    `character.items: [{itemid: 40207, dropchance: 5}]` — the ultra-rare
    Folded-Space Silk. (Match `itemdropchance` to the boss's; the 5% is the
    pinnacle-reagent standard.)
  - Verify `speciesid` for spiders (coulee spider uses 17) — reuse it.

- [ ] **Step 3: Wire spawns** — add `spawninfo:` to the Foldweave rooms:
lessers (9580/9581) scattered through the galleries (repeat entries +
`respawnrate: "12 real minutes"`; `StatPoolMod` to vary), the Silk Lurker
(9582) deeper, and **The Foldweaver (9583) in 6436/6437 only** (single spawn,
the boss). Use the real `respawnrate:` field (NOT the dead `cooldown:` key).

- [ ] **Step 4: Boot smoke** — itemLoadedCount +1 (→404), mobs +~4-6, zero
panics, the Foldweave loads with its mobs, `errors=0`. Kill + verify + delete.

- [ ] **Step 5: Commit**

```bash
git add "_datafiles/world/dogmud/items/materials-40000/40207-folded_space_silk.yaml" "_datafiles/world/dogmud/mobs/the_foldweave/" "_datafiles/world/dogmud/rooms/the_foldweave/"
git commit -m "content(pinnacle): Foldweave spider ecology + Folded-Space Silk drop"
```

---

### Task 8: Full-suite + boot + world-critic pass

- [ ] **Step 1:** `go test -timeout 300s -count=1 ./...` → green.
- [ ] **Step 2:** Instance-wipe + boot: itemLoadedCount 404 (18 reagents +
the Stage-2 386), mobs up by the Foldweave count, room count up ~12,
`ValidateZoneConsistency errors=0 mode=panic`, zero panics, Server Ready.
Kill + verify + delete.
- [ ] **Step 3: World-critic** — review all 18 reagent descriptions + all
Foldweave room descriptions against world.md (lore-boundary leaks, register,
78-col wrap, no numbers). The Foldweave especially: the space-folding flavor
must not tip into naming the cosmic truth. Fix inline; re-boot if any room/
mob YAML changed.
- [ ] **Step 4: Schema-doc update** — append a "Stage 3 reagents" section to
`docs/schemas/pinnacle-items.md`: the reagent→component_tag→source table (so
Stage 4 recipe authors have the tags), the forage-overlay note (player-only,
zone/storm keyed), and The Foldweave zone entry.
- [ ] **Step 5: Commit**

```bash
git add _datafiles/ internal/ docs/schemas/pinnacle-items.md
git commit -m "content(pinnacle): Stage 3 world-critic polish + reagent/source registry"
```

---

### Task 9: Harness playtest (live verification)

Goals file `tools/playtest/goals/pinnacle-stage3-reagents.yaml`, then
`/playtest local feature-tester pinnacle-stage3-reagents.yaml`. Verify with
the admin smoketester:
- **Forage overlay**: in Stillwater Marsh, forage repeatedly and confirm
  Still-Glass Rosette / First-Bloom Nectar CAN appear and don't appear in a
  different zone; confirm Stormfront Residue only forageable in a mountains
  zone during a storm (may need to wait for/force weather — note if not
  observable, the unit test covers it).
- **Mob drops**: `mob spawn 9583` (Foldweaver) and kill it a few times /
  admin-inspect its loot table; spawn Core Guardian / Old White and confirm
  the new reagents are in their drop tables (killing to a 4-5% drop is slow —
  inspect the mob's items or spawn-and-kill a handful; note this is a
  low-chance verification).
- **The Foldweave**: search 4161 to reveal the descent, go down, confirm the
  zone loads, the space-folding rooms render, the spiders are hostile and
  endgame-tough, and the Foldweaver drops Folded-Space Silk (admin-force the
  drop or kill repeatedly).
- Triage; fix content-level findings on the branch.

Report to `tools/playtest/reports/` (gitignored). Commit the goals file.

---

## Explicitly OUT of Stage 3

- Component sub-recipes + assembly recipes (Stage 4 — the recipes that
  CONSUME these reagents via their component_tags).
- Veyra, the workshop, commission quests (Stage 4).
- Any change to how the pinnacle ITEMS behave (that was Stages 1-2).
- True per-item forage rarity tuning knobs — the single-entry-among-commons
  gives ultra-rarity; revisit only if playtest shows it's too common/rare.
- Weather climate registration for The Foldweave (it's a cave — no weather;
  Stormfront is gated on the mountains biome, which has live weather).
