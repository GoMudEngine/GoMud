# Icon Generation Spec — gap icons for DOGMud

The 81 gap items (see `ICON_GAPS.md`) collapse to **~24 new icons** plus **~6 that
can borrow existing pack `other/` icons** (papers/books). One icon per *type*
covers many items (a single "metal ingot" serves iron + steel, etc.).

**Shared style (match the pack):** 64×64 PNG · transparent background · one
centered object · painted / semi-realistic RPG-inventory style · soft top-left
directional light, subtle drop shading, faint dark outline. Reference:
`weapons/dagger.png`, `consumables/small_red_potion.png`.

## Armor (5 new — DOGMud-only slots)
| icon | covers | description |
|------|--------|-------------|
| `armor/back/cloak` | wool / cattail-down / bounty-hunter's cloak | draped hooded cloak with a clasp |
| `armor/back/backpack` | leather backpack, reinforced travel pack | leather rucksack with straps |
| `armor/shoulders/pauldron` | chitin spaulders, mist pauldrons | single armored shoulder plate |
| `armor/tail/tail_guard` | bladed/spiked/weighted tail items | segmented armored tail-band/sheath |
| `armor/wrist/bracer` | storm bracer, resin-laced bracers, engraved bracelet | forearm bracer / wrist cuff |

## Materials (~19 new)
| icon | covers |
|------|--------|
| `metal_ingot` | iron ingot, steel ingot |
| `ore_nodule` | lake-iron nodule, windstone sample, polished stone |
| `raw_gem` | raw gem |
| `pearl` | Stillwater black pearl, freshwater clam (pearl-in-shell) |
| `dust_pouch` | coal dust, gem dust, salt pouch, putrid residue |
| `wire_coil` | copper / gold / silver wire (tint per metal) |
| `tree_bark` | oak bark, marsh willow bark, ironbark shaving |
| `resin_glob` | pine pitch, beeswax, binding paste |
| `wood_plank` | wooden plank, tally stick |
| `hide` | drowned-hunter hide, leather strip, sinew |
| `cloth_strip` | cloth strip, thread spool |
| `raw_meat` | raw meat, wild hare meat |
| `gland_sac` | serpent venom sac, spore sac |
| `shell` | skitter-shrimp shell |
| `tooth_trophy` | leviathan-tooth trophy |
| `herb_sprig` | bitter thistle, blood-moss, dustwalk herb, forest herbs, healer's root, lake mint, moonpetal, veilbloom petal |
| `mushroom` | shadowcap mushroom |
| `chrysalis_crystal` (glowing, themed) | Chrysalis Core, chrysalis setting, chrysalis shard, Hive Fragment, mutation catalyst |
| `misc_craft` | bone needle, chain link, oil lantern → (or split if wanted) |
| `key` | strongbox key |
| `crate` | freight crate |
| `carved_figurine` | carved wolf totem, spirit fetish, Elgar's carved kingfisher |

## Borrow from existing pack `other/` (no new art)
Papers/books/documents → reuse `other/` note/book/map icons:
Elgar's last journal entry, bribe ledger, creased letter, guard captain's
commendation, herbalism recipe page, water flask/clay flask/glass vial/sealed
phial/crystalline decanter (→ if the pack potion/bottle icons fit, reuse those).

## Resolved rendering method — GENERATED 2026-06-08
Method chosen: **(b) image-gen MCP** (`image-gen-mcp`, OpenAI `gpt-image-2`,
1024×1024, `background: transparent`, **quality `low`**), then downscaled to
64×64 PNG with Pillow (autocrop → 12% pad → LANCZOS).

**Cost/quality finding (important):** at `low` quality each icon bills ~$0.0055
and is indistinguishable once downscaled to 64px. `high`/`medium` transparent
renders **time out** in the MCP — and OpenAI **still bills** the timed-out request
at the expensive tier (~$0.30–0.60 for nothing). **Stay on `low`.** Do not retry
on `high`/`medium` until the MCP request timeout is raised.

### Generated (30 icons — covers all 81 gap items)
Armor (5): `armor/back/cloak`, `armor/back/backpack`, `armor/shoulders/pauldron`,
`armor/tail/tail_guard`, `armor/wrist/bracer`.
Materials/misc (25, under `materials/`): `metal_ingot`, `ore_nodule`, `raw_gem`,
`pearl`, `dust_pouch`, `wire_coil`, `tree_bark`, `resin_glob`, `wood_plank`,
`hide`, `cloth_strip`, `raw_meat`, `gland_sac`, `shell`, `tooth_trophy`,
`shadowcap_mushroom`, `carved_figurine`, `chrysalis_crystal`, `key`, `crate`,
`oil_lantern`, `chain_link`, `bone_needle`, `flask`, `glass_vial`, `herb_sprig`.

### Borrowed from `other/` (no new art)
Papers/books → `other/note.png` (creased letter, bribe ledger, guard captain's
commendation), `other/history_of_frostfang.png` or `other/the_shadow_herbarium.png`
(Elgar's last journal entry, herbalism recipe page). Carved kingfisher → the
generated `materials/carved_figurine`. Existing `other/*_key.png` also available
alongside the new generic `materials/key`.
