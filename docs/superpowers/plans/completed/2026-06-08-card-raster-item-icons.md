# Raster Item-Icons in the Inventory/Equipment Card — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the web client's inventory/equipment card render painted raster icons from the GoMud asset pack (143 icons), falling back to the existing monochrome SVG glyphs only when no pack icon fits.

**Architecture:** A Python sync script copies the repo-root `GoMudAssetsPack/` PNGs into the web-served `static/images/items/` and writes a `manifest.json`. A pure-JS module `item-icon-map.js` resolves an item to an icon URL via three data tiers (exact-name → keyword → type/subtype category), returning `null` to signal SVG fallback. The card's tile renderer is refactored into one `renderTileIcon` helper that prefers an `<img>` and falls back to `itemIconSVG`. A CSS rule sizes the raster icons.

**Tech Stack:** Python 3 + Pillow (sync script), vanilla ES5 JS (browser module, matches existing `item-icons.js` style), Node 24 (test harness, no framework), Go HTML template (`webclient-pure.html`), CSS.

**Spec:** `docs/superpowers/specs/2026-06-08-card-raster-item-icons-design.md`

---

## File Structure

- **new** `tools/sync_item_icons.py` — copy/downscale pack → `static/images/items/`, emit `manifest.json`. One responsibility: asset sync.
- **new** `_datafiles/html/public/static/js/item-icon-map.js` — `window.itemIconURL(item)` + the three mapping tables. One responsibility: name/type → icon-URL resolution. Node-loadable for testing.
- **new** `tools/test_item_icon_map.mjs` — Node test harness over the module + manifest cross-check.
- **generated** `_datafiles/html/public/static/images/items/*.png` + `manifest.json` — produced by the sync script, committed (prod serves them).
- **edit** `_datafiles/html/public/webclient-pure.html` — `renderTileIcon` helper, replace the two duplicated icon blocks, add `ITEM_ICON_BASE` + `<script>` include.
- **edit** `_datafiles/html/public/static/css/dashboard.css` — `.inv-tile > img.inv-img` rule.

Pack layout reference (source of representative icons used in the TYPE_MAP):
`weapons/` (dagger, finely_crafted_shortsword, guardsmans_broadsword, captains_broadsword, glowing_battleaxe, cudgel, dancing_needle, obsidian_dagger, long_whip, sling, wooden_claw, ancient_royal_scepter, crowbar, …); `armor/<slot>/` (head: leather_cap; body: leather_vest, leather_robe, cotton_shirt; offhand: iron_shield, wooden_shield, lantern; neck: students_amulet, leather_choker; belt: cloth_belt; gloves: torn_gloves; ring: copper_ring; legs: leather_pants; feet: worn_boots; back: cloak, backpack [new]; shoulders: pauldron [new]; tail: tail_guard [new]; wrist: bracer [new]); `consumables/` (small_red_potion, mug_of_ale, cheese_sandwich, mutton_stew, waterskin, mushroom); `other/` (note, map, rope, history_of_frostfang, lockpick_kit, pinecone_grenade, spellbound_projectiles, junk, crypt_key, stat_coupon); `materials/` (the 31 new icons).

---

## Task 1: Asset sync script

**Files:**
- Create: `tools/sync_item_icons.py`
- Output (generated): `_datafiles/html/public/static/images/items/*.png`, `.../manifest.json`

- [ ] **Step 1: Write the sync script**

Create `tools/sync_item_icons.py`:

```python
#!/usr/bin/env python3
"""Sync GoMudAssetsPack PNG icons into the web-served static dir.

Copies every GoMudAssetsPack/**/*.png into
_datafiles/html/public/static/images/items/ as a flat, basename-keyed
file, downscaling anything larger than 64x64 to 64x64 (LANCZOS, alpha
preserved). Writes manifest.json (sorted list of served basenames).
Idempotent. First-copy-wins on basename collisions, with a warning.

Run: python tools/sync_item_icons.py
"""
import json
import os
import sys

try:
    from PIL import Image
except ImportError:
    sys.exit("Pillow required: pip install pillow")

REPO = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
SRC = os.path.join(REPO, "GoMudAssetsPack")
DST = os.path.join(REPO, "_datafiles", "html", "public", "static", "images", "items")
SIZE = 64


def main():
    os.makedirs(DST, exist_ok=True)
    served = {}        # basename -> source relpath (first wins)
    collisions = []
    copied = resized = 0

    for root, _dirs, files in os.walk(SRC):
        for fn in sorted(files):
            if not fn.lower().endswith(".png"):
                continue
            src = os.path.join(root, fn)
            rel = os.path.relpath(src, SRC)
            if fn in served:
                collisions.append((fn, served[fn], rel))
                continue
            served[fn] = rel
            im = Image.open(src).convert("RGBA")
            if im.size != (SIZE, SIZE):
                im = im.resize((SIZE, SIZE), Image.LANCZOS)
                resized += 1
            im.save(os.path.join(DST, fn))
            copied += 1

    manifest = sorted(served.keys())
    with open(os.path.join(DST, "manifest.json"), "w") as f:
        json.dump(manifest, f, indent=0)

    print(f"synced {copied} icons ({resized} resized) -> {DST}")
    if collisions:
        print(f"WARNING: {len(collisions)} basename collision(s) (first wins):")
        for fn, kept, dropped in collisions:
            print(f"  {fn}: kept {kept}, dropped {dropped}")
    print(f"manifest.json: {len(manifest)} entries")


if __name__ == "__main__":
    main()
```

- [ ] **Step 2: Run the script**

Run: `python tools/sync_item_icons.py`
Expected: `synced 143 icons (... resized) -> ...items` and `manifest.json: 143 entries` (count may differ slightly if collisions exist; any collision is printed — note it but proceed).

- [ ] **Step 3: Verify outputs exist**

Run: `python -c "import json,os; m=json.load(open('_datafiles/html/public/static/images/items/manifest.json')); print(len(m)); print('metal_ingot.png' in m, 'dagger.png' in m, 'leather_cap.png' in m)"`
Expected: a count (~143) then `True True True`.

- [ ] **Step 4: Commit**

```bash
git add tools/sync_item_icons.py _datafiles/html/public/static/images/items
git commit -m "feat(web): sync script + served raster item-icons (pack -> static)"
```

---

## Task 2: Icon-URL lookup module

**Files:**
- Create: `_datafiles/html/public/static/js/item-icon-map.js`
- Test: `tools/test_item_icon_map.mjs`

- [ ] **Step 1: Write the failing test harness**

Create `tools/test_item_icon_map.mjs`:

```js
// Node test harness for item-icon-map.js (no framework).
// Run: node tools/test_item_icon_map.mjs
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import path from "node:path";
import vm from "node:vm";

const here = path.dirname(fileURLToPath(import.meta.url));
const repo = path.dirname(here);
const modPath = path.join(repo, "_datafiles/html/public/static/js/item-icon-map.js");
const manifestPath = path.join(repo, "_datafiles/html/public/static/images/items/manifest.json");

// Load the browser module into a sandbox with a fake `window`.
const sandbox = { window: {}, console };
vm.createContext(sandbox);
vm.runInContext(readFileSync(modPath, "utf8"), sandbox);
const itemIconURL = sandbox.window.itemIconURL;
const TABLES = sandbox.window.ITEM_ICON_TABLES;

let fails = 0;
function eq(actual, expected, label) {
  const ok = actual === expected;
  if (!ok) { fails++; console.error(`FAIL ${label}: got ${JSON.stringify(actual)} want ${JSON.stringify(expected)}`); }
  else console.log(`ok   ${label}`);
}
function urlFor(icon) { return "/static/images/items/" + icon + ".png"; }

// Exact-name tier
eq(itemIconURL({ name: "iron ingot" }), urlFor("metal_ingot"), "exact: iron ingot");
eq(itemIconURL({ name: "Steel Ingot" }), urlFor("metal_ingot"), "exact: case-insensitive");
eq(itemIconURL({ name: "a wool cloak" }), urlFor("cloak"), "exact: leading article stripped");
eq(itemIconURL({ name: "bounty hunter's cloak" }), urlFor("cloak"), "exact: apostrophe name");

// Keyword tier
eq(itemIconURL({ name: "rusted broadsword" }), urlFor("finely_crafted_shortsword"), "kw: broadsword->sword");
eq(itemIconURL({ name: "copper wire" }), urlFor("wire_coil"), "kw: wire");
eq(itemIconURL({ name: "gnarled oak bark" }), urlFor("tree_bark"), "kw: bark");

// Type/subtype tier
eq(itemIconURL({ name: "mystery brew", type: "potion" }), urlFor("small_red_potion"), "type: potion");
eq(itemIconURL({ name: "odd hat", type: "head" }), urlFor("leather_cap"), "type: head");
eq(itemIconURL({ name: "big chopper", type: "weapon", subtype: "axe" }), urlFor("glowing_battleaxe"), "type: weapon-axe");

// Fallback
eq(itemIconURL({ name: "ineffable thing", type: "nonsense" }), null, "fallback: unknown -> null");

// Manifest cross-check: every referenced icon basename must be a served file.
const manifest = new Set(JSON.parse(readFileSync(manifestPath, "utf8")));
const referenced = new Set();
for (const v of Object.values(TABLES.NAME_MAP)) referenced.add(v);
for (const [, v] of TABLES.KEYWORD_RULES) referenced.add(v);
for (const v of Object.values(TABLES.TYPE_MAP)) referenced.add(v);
for (const icon of referenced) {
  eq(manifest.has(icon + ".png"), true, `manifest has ${icon}.png`);
}

console.log(fails ? `\n${fails} FAILURES` : "\nALL PASS");
process.exit(fails ? 1 : 0);
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `node tools/test_item_icon_map.mjs`
Expected: FAIL — throws because `item-icon-map.js` does not exist yet (cannot read module file).

- [ ] **Step 3: Write the module**

Create `_datafiles/html/public/static/js/item-icon-map.js`:

```js
// item-icon-map.js — maps a GMCP item to a painted raster icon URL.
// Returns null when no pack icon fits, so the caller falls back to the
// monochrome SVG glyph from item-icons.js. Three tiers, all data:
//   1. NAME_MAP    exact normalized item name -> icon basename
//   2. KEYWORD_RULES ordered [regex, icon] on the normalized name
//   3. TYPE_MAP    "type-subtype" then "type" -> representative icon
// Icons are served from window.ITEM_ICON_BASE (set by the page template),
// defaulting to "/static/images/items".
"use strict";
(function (global) {

  // Exact item-name -> icon basename. Covers the 31 new gap icons (from
  // ICON_GENERATION_SPEC covers-lists) plus generic community names.
  var NAME_MAP = {
    // --- new gap icons ---
    "iron ingot": "metal_ingot", "steel ingot": "metal_ingot",
    "lake-iron nodule": "ore_nodule", "windstone sample": "ore_nodule", "polished stone": "ore_nodule",
    "raw gem": "raw_gem",
    "stillwater black pearl": "pearl", "freshwater clam": "pearl",
    "coal dust": "dust_pouch", "gem dust": "dust_pouch", "salt pouch": "dust_pouch", "putrid residue": "dust_pouch",
    "copper wire": "wire_coil", "gold wire": "wire_coil", "silver wire": "wire_coil",
    "oak bark": "tree_bark", "marsh willow bark": "tree_bark", "ironbark shaving": "tree_bark",
    "pine pitch": "resin_glob", "beeswax": "resin_glob", "binding paste": "resin_glob",
    "wooden plank": "wood_plank", "tally stick": "wood_plank",
    "drowned-hunter hide": "hide", "leather strip": "hide", "sinew": "hide",
    "cloth strip": "cloth_strip", "thread spool": "cloth_strip",
    "raw meat": "raw_meat", "wild hare meat": "raw_meat",
    "serpent venom sac": "gland_sac", "spore sac": "gland_sac",
    "skitter-shrimp shell": "shell",
    "leviathan-tooth trophy": "tooth_trophy",
    "shadowcap mushroom": "shadowcap_mushroom",
    "carved wolf totem": "carved_figurine", "spirit fetish": "carved_figurine", "elgar's carved kingfisher": "carved_figurine",
    "chrysalis core": "chrysalis_crystal", "chrysalis setting": "chrysalis_crystal", "chrysalis shard": "chrysalis_crystal",
    "hive fragment": "chrysalis_crystal", "mutation catalyst": "chrysalis_crystal",
    "strongbox key": "key",
    "freight crate": "crate",
    "oil lantern": "oil_lantern",
    "chain link": "chain_link",
    "bone needle": "bone_needle",
    "water flask": "flask", "clay flask": "flask",
    "glass vial": "glass_vial", "sealed phial": "glass_vial", "crystalline decanter": "glass_vial",
    "wool cloak": "cloak", "cattail-down cloak": "cloak", "bounty hunter's cloak": "cloak",
    "leather backpack": "backpack", "reinforced travel pack": "backpack",
    "chitin spaulders": "pauldron", "mist pauldrons": "pauldron",
    "weighted tail cap": "tail_guard", "spiked tail band": "tail_guard", "bladed tail sheath": "tail_guard",
    "engraved bracelet": "bracer", "resin-laced bracers": "bracer", "storm bracer": "bracer",
    "bitter thistle": "herb_sprig", "blood-moss": "herb_sprig", "dustwalk herb": "herb_sprig",
    "forest herbs": "herb_sprig", "healer's root": "herb_sprig", "lake mint": "herb_sprig",
    "moonpetal": "herb_sprig", "veilbloom petal": "herb_sprig",
    // --- generic community names (match if DOGMud has same-named item) ---
    "dagger": "dagger", "crowbar": "crowbar", "sling": "sling", "cudgel": "cudgel",
    "rope": "rope", "waterskin": "waterskin", "mug of ale": "mug_of_ale",
    "cheese sandwich": "cheese_sandwich", "mutton stew": "mutton_stew",
    "leather cap": "leather_cap", "leather vest": "leather_vest", "leather robe": "leather_robe",
    "cotton shirt": "cotton_shirt", "leather pants": "leather_pants", "worn boots": "worn_boots",
    "iron shield": "iron_shield", "wooden shield": "wooden_shield",
    "copper ring": "copper_ring", "cloth belt": "cloth_belt"
  };

  // Ordered keyword rules — first match wins. Extends specific-named
  // community icons to whole item families.
  var KEYWORD_RULES = [
    [/\bingot\b/, "metal_ingot"],
    [/nodule|\bore\b/, "ore_nodule"],
    [/\bwire\b/, "wire_coil"],
    [/\bbark\b/, "tree_bark"],
    [/pitch|resin|beeswax|binding paste/, "resin_glob"],
    [/plank|tally stick|\blumber\b/, "wood_plank"],
    [/\bhide\b|leather strip|\bsinew\b|\bpelt\b/, "hide"],
    [/cloth strip|thread spool|\bbandage\b/, "cloth_strip"],
    [/venom sac|spore sac|\bgland\b/, "gland_sac"],
    [/\bmeat\b|\bflesh\b/, "raw_meat"],
    [/\bshell\b|carapace|chitin/, "shell"],
    [/\btooth\b|\bfang\b|\btusk\b/, "tooth_trophy"],
    [/\bpearl\b|\bclam\b/, "pearl"],
    [/chrysalis|hive fragment|mutation catalyst/, "chrysalis_crystal"],
    [/shadowcap/, "shadowcap_mushroom"],
    [/dust|residue|\bsalt\b/, "dust_pouch"],
    [/\bcrate\b|freight/, "crate"],
    [/lantern/, "oil_lantern"],
    [/chain link|\bchain\b/, "chain_link"],
    [/\bneedle\b|\bawl\b/, "bone_needle"],
    [/totem|fetish|figurine|carved/, "carved_figurine"],
    [/\bkey\b/, "key"],
    [/\bcloak\b/, "cloak"],
    [/backpack|rucksack|travel pack/, "backpack"],
    [/pauldron|spaulder/, "pauldron"],
    [/bracer|bracelet/, "bracer"],
    [/tail (cap|band|sheath|guard)/, "tail_guard"],
    [/\bvial\b|\bphial\b|decanter/, "glass_vial"],
    [/\bflask\b/, "flask"],
    [/\bdagger\b|\bdirk\b|\bstiletto\b/, "dagger"],
    [/broadsword|longsword|shortsword|\bsword\b|\bblade\b|\bsabre\b|scimitar/, "finely_crafted_shortsword"],
    [/battleaxe|\baxe\b|hatchet/, "glowing_battleaxe"],
    [/\bwhip\b|\blash\b/, "long_whip"],
    [/\bsling\b|\bbow\b/, "sling"],
    [/\bclaw\b/, "wooden_claw"],
    [/\bstaff\b|\bstave\b/, "ancient_royal_scepter"],
    [/scepter|sceptre|\bwand\b/, "ancient_royal_scepter"],
    [/cudgel|\bclub\b|\bmace\b|mallet|\bbludgeon\b/, "cudgel"],
    [/\bshield\b|buckler/, "iron_shield"],
    [/\bale\b|\bbeer\b|\bmead\b|\bgrog\b/, "mug_of_ale"],
    [/\bstew\b|\bsoup\b|\bbroth\b/, "mutton_stew"],
    [/sandwich|\bbread\b|\bcheese\b|\bration\b/, "cheese_sandwich"],
    [/waterskin|\bwater\b|\btea\b/, "waterskin"],
    [/\bpotion\b|elixir|tonic|draught|philter|brew/, "small_red_potion"],
    [/\bscroll\b/, "note"],
    [/\bbook\b|journal|ledger|tome|herbarium|recipe page|commendation/, "history_of_frostfang"],
    [/\bletter\b|\bnote\b/, "note"],
    [/\bmap\b/, "map"],
    [/\brope\b|\bcord\b|twine/, "rope"],
    [/\bgem\b|\bcrystal\b|\bjewel\b/, "raw_gem"],
    [/\bmushroom\b|\bfungus\b/, "mushroom"],
    [/\bherb\b|thistle|\bmoss\b|petal|\bmint\b|\broot\b|bloom|sprig|leaf/, "herb_sprig"]
  ];

  // type-subtype (then type) -> representative icon. Keys mirror the SVG
  // glyph key scheme in item-icons.js so every glyphed item gets a raster.
  var TYPE_MAP = {
    "weapon": "guardsmans_broadsword",
    "weapon-sword": "finely_crafted_shortsword",
    "weapon-dagger": "dagger",
    "weapon-axe": "glowing_battleaxe",
    "weapon-bludgeoning": "cudgel",
    "weapon-cleaving": "captains_broadsword",
    "weapon-stabbing": "dancing_needle",
    "weapon-slashing": "captains_broadsword",
    "weapon-shooting": "sling",
    "weapon-claws": "wooden_claw",
    "weapon-fist": "wooden_claw",
    "weapon-whipping": "long_whip",
    "weapon-wand": "ancient_royal_scepter",
    "weapon-sceptre": "ancient_royal_scepter",
    "weapon-staff": "ancient_royal_scepter",
    "offhand": "iron_shield",
    "head": "leather_cap",
    "neck": "students_amulet",
    "shoulders": "pauldron",
    "body": "leather_vest",
    "back": "cloak",
    "belt": "cloth_belt",
    "wrist": "bracer",
    "ring": "copper_ring",
    "legs": "leather_pants",
    "feet": "worn_boots",
    "tail": "tail_guard",
    "potion": "small_red_potion",
    "food": "cheese_sandwich",
    "drink": "mug_of_ale",
    "scroll": "note",
    "grenade": "pinecone_grenade",
    "junk": "junk",
    "readable": "history_of_frostfang",
    "key": "key",
    "object": "crate",
    "gemstone": "raw_gem",
    "lockpicks": "lockpick_kit",
    "botanical": "herb_sprig",
    "ammo": "spellbound_projectiles",
    "service": "stat_coupon"
    // NOTE: "componentbag" intentionally omitted -> SVG glyph (no clean
    // bag icon in the pack).
  };

  function normalize(name) {
    return (name || "")
      .toLowerCase()
      .replace(/\s+/g, " ")
      .trim()
      .replace(/^(a|an|the) /, "");
  }

  function lookupIcon(item) {
    var name = normalize(item && item.name);
    if (name && NAME_MAP[name]) return NAME_MAP[name];
    if (name) {
      for (var i = 0; i < KEYWORD_RULES.length; i++) {
        if (KEYWORD_RULES[i][0].test(name)) return KEYWORD_RULES[i][1];
      }
    }
    var type = ((item && item.type) || "").toLowerCase();
    var subtype = ((item && item.subtype) || "").toLowerCase();
    if (type) {
      var k = type + "-" + subtype;
      if (subtype && TYPE_MAP[k]) return TYPE_MAP[k];
      if (TYPE_MAP[type]) return TYPE_MAP[type];
    }
    return null;
  }

  global.itemIconURL = function (item) {
    var icon = lookupIcon(item);
    if (!icon) return null;
    var base = global.ITEM_ICON_BASE || "/static/images/items";
    return base + "/" + icon + ".png";
  };

  // Exposed for the Node test harness (manifest cross-check).
  global.ITEM_ICON_TABLES = { NAME_MAP: NAME_MAP, KEYWORD_RULES: KEYWORD_RULES, TYPE_MAP: TYPE_MAP };

})(typeof window !== "undefined" ? window : globalThis);
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `node tools/test_item_icon_map.mjs`
Expected: every `ok …` line, ending `ALL PASS`, exit 0. If a `manifest has X.png` line FAILS, the referenced icon isn't in the synced set — fix the basename in the table (or confirm Task 1 ran), do not weaken the test.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/html/public/static/js/item-icon-map.js tools/test_item_icon_map.mjs
git commit -m "feat(web): itemIconURL three-tier item->icon lookup (+ node test)"
```

---

## Task 3: Renderer helper + page wiring

**Files:**
- Modify: `_datafiles/html/public/webclient-pure.html` (script include + `ITEM_ICON_BASE` near line 197; `renderTileIcon` helper + two call sites near lines 706-849)

- [ ] **Step 1: Add the script include and icon base**

In `_datafiles/html/public/webclient-pure.html`, immediately AFTER the existing line 197
`<script src="{{ .CONFIG.FilePaths.WebCDNLocation }}/static/js/item-icons.js"></script>`
insert:

```html
    <script src="{{ .CONFIG.FilePaths.WebCDNLocation }}/static/js/item-icon-map.js"></script>
    <script>window.ITEM_ICON_BASE = "{{ .CONFIG.FilePaths.WebCDNLocation }}/static/images/items";</script>
```

- [ ] **Step 2: Add the `renderTileIcon` helper**

In the same file, immediately BEFORE the `function renderInventory() {` declaration (currently line 706), insert this helper:

```js
        // Render an item's icon into a tile: prefer a painted raster from
        // itemIconURL, fall back to the monochrome SVG glyph. Must run
        // BEFORE the charge meter / stack badge are appended (the onerror
        // path uses insertAdjacentHTML so it never clobbers those siblings).
        function renderTileIcon(tile, item) {
            var url = (typeof window.itemIconURL === "function")
                ? window.itemIconURL(item) : null;
            if (url) {
                var img = document.createElement("img");
                img.className = "inv-img";
                img.src = url;                 // src, not innerHTML — no injection surface
                img.alt = item.name || "";
                img.loading = "lazy";
                img.onerror = function () {
                    img.remove();
                    var svg = (typeof window.itemIconSVG === "function")
                        ? window.itemIconSVG(item.type, item.subtype) : "";
                    if (svg) tile.insertAdjacentHTML("afterbegin", svg);
                };
                tile.appendChild(img);
            } else {
                var svg = (typeof window.itemIconSVG === "function")
                    ? window.itemIconSVG(item.type, item.subtype) : "";
                if (svg) tile.innerHTML = svg;
            }
        }
```

- [ ] **Step 3: Replace the worn-branch icon block**

Find this block in the worn branch (currently around lines 747-751):

```js
                    // Icon from trusted local map — innerHTML is safe here
                    var iconSVG = (typeof window.itemIconSVG === "function")
                        ? window.itemIconSVG(item.type, item.subtype)
                        : "";
                    if (iconSVG) tile.innerHTML = iconSVG;
```

Replace it with:

```js
                    renderTileIcon(tile, item);
```

- [ ] **Step 4: Replace the container-branch icon block**

Find the identical block in the container branch (currently around lines 823-826):

```js
                    var iconSVG = (typeof window.itemIconSVG === "function")
                        ? window.itemIconSVG(item.type, item.subtype)
                        : "";
                    if (iconSVG) tile.innerHTML = iconSVG;
```

Replace it with:

```js
                    renderTileIcon(tile, item);
```

- [ ] **Step 5: Verify no stray references and the helper is single-source**

Run: `grep -n "renderTileIcon\|itemIconSVG\|itemIconURL\|item-icon-map" _datafiles/html/public/webclient-pure.html`
Expected: the `renderTileIcon` definition once, two call sites, the `itemIconSVG` uses only inside the helper, plus the two `<script>`/`ITEM_ICON_BASE` includes. No leftover `var iconSVG` assignments in the render branches.

- [ ] **Step 6: Commit**

```bash
git add _datafiles/html/public/webclient-pure.html
git commit -m "feat(web): card tiles render raster icons via renderTileIcon (dedupe + fallback)"
```

---

## Task 4: CSS for raster icons

**Files:**
- Modify: `_datafiles/html/public/static/css/dashboard.css` (after the `.inv-tile > svg` rule, around line 432)

- [ ] **Step 1: Add the img rule**

In `_datafiles/html/public/static/css/dashboard.css`, immediately AFTER the closing `}` of the `.inv-tile > svg { … }` rule (line 432), insert:

```css
/* Painted raster icons from itemIconURL — fill more of the tile than the
   line-glyphs (58%); transparent PNGs sit on the tile gradient. */
.inv-tile > img.inv-img {
  width: 82%;
  height: 82%;
  object-fit: contain;
  pointer-events: none;   /* clicks fall through to the tile action menu */
}
```

- [ ] **Step 2: Sanity-check the CSS edit**

Run: `grep -n "inv-img\|inv-tile > svg" _datafiles/html/public/static/css/dashboard.css`
Expected: the existing `.inv-tile > svg` rule and the new `.inv-tile > img.inv-img` rule both present.

- [ ] **Step 3: Commit**

```bash
git add _datafiles/html/public/static/css/dashboard.css
git commit -m "style(web): size raster item-icons in inventory tiles"
```

---

## Task 5: Local smoke test (manual acceptance)

**Files:** none (verification only)

- [ ] **Step 1: Re-run the JS test (regression guard)**

Run: `node tools/test_item_icon_map.mjs`
Expected: `ALL PASS`.

- [ ] **Step 2: Build + boot the server locally**

Per the project pre-push SOP (no data-file changes here, so the instance-save wipe is optional). Run: `go build ./...` then start the server (e.g. `go run .`), and watch it boot past data-file loading without panics.

- [ ] **Step 3: Open the web client and inspect the card**

Open the local web client in a browser, log in, and open the inventory/equipment card. Acquire/equip a spread of items and check each tab (Worn / Components / Backpack / Bandolier):
- Items with name/keyword/type matches show **painted raster icons** (e.g. an ingot → metal_ingot, a sword → a sword icon, a potion → small_red_potion, a helm → leather_cap).
- Icons sit centered at ~82% of the tile with transparent edges on the dark gradient.
- **Charge meters and xN stack badges** still render over/under the icon correctly.
- An item with no pack match (a type omitted from TYPE_MAP, e.g. a component bag) still shows its **SVG glyph** — no broken-image box.
- Open devtools Network: no 404s for `/static/images/items/*.png` on rendered tiles.

- [ ] **Step 4: Note results**

If everything renders correctly, the feature is done. If a specific item shows the wrong icon, that's a mapping-data tweak (edit `NAME_MAP`/`KEYWORD_RULES`/`TYPE_MAP` and re-run the Node test) — not a logic change.

---

## Self-Review notes (addressed)

- **Spec coverage:** sync script + manifest (Task 1) ✓; three-tier `itemIconURL` with SVG fallback (Task 2) ✓; deduped `renderTileIcon` + `<img>` + onerror fallback + script wiring (Task 3) ✓; `.inv-img` CSS at 82% (Task 4) ✓; unit (Node) + manual smoke testing (Tasks 2 & 5) ✓; all-143-icon scope via NAME_MAP + KEYWORD_RULES + TYPE_MAP ✓.
- **Type consistency:** the module exposes `window.itemIconURL` and `window.ITEM_ICON_TABLES` (with `NAME_MAP`, `KEYWORD_RULES`, `TYPE_MAP`); the test harness reads exactly those names; the renderer calls `window.itemIconURL` and `window.itemIconSVG`; CSS selector `.inv-tile > img.inv-img` matches the `img.className = "inv-img"` appended as a direct child of `tile`.
- **Fallback contract:** unmapped items and missing files both resolve to the existing SVG glyph; `insertAdjacentHTML("afterbegin", …)` in `onerror` preserves the later-appended charge meter / stack badge.
