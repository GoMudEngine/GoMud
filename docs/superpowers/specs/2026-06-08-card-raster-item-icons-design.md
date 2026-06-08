# Raster item-icons in the inventory/equipment card

**Date:** 2026-06-08
**Status:** Design — approved for planning
**Author:** Calabe Davis + Claude

## Problem

The web client's inventory/equipment card renders each item with a
**monochrome inline-SVG glyph** keyed on item type/subtype
(`static/js/item-icons.js`, `itemIconSVG(type, subtype)`). Hands-on
feedback flagged the glyphs as "tiny and abstract." We now have a
painted raster icon set: the community GoMud asset pack (112 icons) plus
**31 newly generated gap icons** (`GoMudAssetsPack/`, committed
2026-06-08 at `6f0efc81`). This work wires the new raster icons into the
card so real items show painted icons, with the existing SVG glyphs as a
guaranteed fallback.

This is step #3 ("serving the icons in the card") of the icon pipeline
tracked in the next-session plan. It is the **first pass**: it stands up
the serving mechanism and maps the 31 new gap icons. Mapping the full
261-item catalogue (and the 112 community icons) is an explicit
follow-up that reuses this same mechanism.

## Constraints / context

- **No item id over GMCP.** Each rendered item exposes only `type`,
  `subtype`, `name`, `handle`, `uses`, `usesMax`
  (`webclient-pure.html` `renderInventory()`). Icon resolution must key
  off **name (+ type/subtype)**, not an id or tag.
- **Static serving root** is `_datafiles/html/public/static/`
  (already serves `css/`, `js/`, `audio/`, `images/`). The asset pack
  lives at repo-root `GoMudAssetsPack/`, which is **not** web-served, so
  icons must be physically copied under the static tree (also required
  for prod, which deploys the `html` tree).
- **Zero regression for unmapped items.** Any item without a raster
  match must render exactly as it does today (SVG glyph).
- Icon files are 64×64 transparent PNGs, painted style with their own
  light/shadow (see `GoMudAssetsPack/ICON_GENERATION_SPEC.md`).

## Architecture

Four small, independent units plus one wire-up:

### 1. Asset sync script — `tools/sync_item_icons.py`
Copies `GoMudAssetsPack/**/*.png` into
`_datafiles/html/public/static/images/items/` as **flat, keyed
filenames** (the icon's basename, e.g. `metal_ingot.png`,
`cloak.png`). Downscales anything larger than 64×64 to 64×64
(LANCZOS, preserve alpha); already-64×64 icons are copied as-is.
Idempotent — safe to re-run. Prints a summary (copied / skipped /
resized counts).

- First pass: the 31 new gap icons are what the mapping references, but
  the script copies the **entire pack** (cheap, ~tens of KB each) so the
  community-icon follow-up needs only mapping-table additions, no
  re-plumbing.
- Filename collisions across pack subfolders (same basename in two
  categories): first copy wins, the collision is logged as a warning,
  and the source paths are listed so it can be resolved by hand. The 31
  first-pass gap icons have unique basenames, so the mapping is
  unaffected; collisions only matter when the full pack is later mapped.
- The synced `static/images/items/` PNGs are committed (prod needs
  them); the script is the regenerator, not a build-time-only step.

### 2. Icon-URL lookup module — `static/js/item-icon-map.js`
Exposes `window.itemIconURL(item) -> string|null`. Pure data + one
function; no DOM, no network.

Resolution order (first hit wins):
1. **Exact-name table** `NAME_MAP`: normalized `item.name` →
   icon basename. Built from each icon's "covers" list in
   `ICON_GENERATION_SPEC.md` (e.g. `"iron ingot"`, `"steel ingot"` →
   `metal_ingot`).
2. **Keyword rules** `KEYWORD_RULES`: ordered `[regex, icon]` pairs
   tested against the normalized name (e.g. `/\bingot\b/→metal_ingot`,
   `/\bwire\b/→wire_coil`, `/bracer|bracelet/→bracer`,
   `/\bcloak\b/→cloak`). Catches authored variants the exact table
   misses.
3. Return `null` → caller falls back to `itemIconSVG`.

Normalization: lowercase, collapse whitespace, strip a leading article
("a "/"an "/"the "). Returns a path string
`"/static/images/items/<icon>.png"` on match.

Type/subtype → category-icon resolution is intentionally **out of
scope for this pass** (the SVG glyphs already cover type/subtype well);
the module is structured to allow adding a `TYPE_MAP` tier later.

### 3. Renderer helper — in `webclient-pure.html`
Extract the icon-building block (currently **duplicated** at the worn
branch `:748` and the container branch `:823`) into one helper:

```
function renderTileIcon(tile, item) {
  var url = (typeof window.itemIconURL === "function")
    ? window.itemIconURL(item) : null;
  if (url) {
    var img = document.createElement("img");
    img.className = "inv-img";
    img.src = url;                 // src, not innerHTML — no injection surface
    img.alt = item.name || "";
    img.loading = "lazy";
    img.onerror = function () {    // missing file → graceful SVG fallback
      img.remove();
      var svg = window.itemIconSVG ? window.itemIconSVG(item.type, item.subtype) : "";
      // insertAdjacentHTML (NOT innerHTML): onerror is async and fires after
      // the charge meter / stack badge were appended — don't clobber them.
      if (svg) tile.insertAdjacentHTML("afterbegin", svg);
    };
    tile.appendChild(img);
  } else {
    var svg = (typeof window.itemIconSVG === "function")
      ? window.itemIconSVG(item.type, item.subtype) : "";
    if (svg) tile.innerHTML = svg;     // trusted local string — safe
  }
}
```

Both call sites become `renderTileIcon(tile, item);`. The helper runs
**before** the charge meter / stack badge / dataset assignments so those
remain siblings appended after the icon (unchanged ordering).

### 4. CSS — `static/css/dashboard.css`
Add alongside the existing `.inv-tile > svg` rule:

```
.inv-tile > img.inv-img {
  width: 82%;
  height: 82%;
  object-fit: contain;
  pointer-events: none;   /* clicks fall through to the tile (action menu) */
}
```

Raster icons read better filling ~82% vs the line-glyphs' 58%. The
charge-meter/stack-badge/hover rules are untouched.

### 5. Wire-up
Add `<script src="/static/js/item-icon-map.js"></script>` after the
existing `item-icons.js` include in `webclient-pure.html`.

## Data flow

GMCP `Char.Inventory` → `renderInventory()` iterates worn / container
items → for each, `renderTileIcon(tile, item)` → `itemIconURL(item)`
resolves name → either an `<img>` from `/static/images/items/…` or the
SVG glyph fallback → charge meter / stack badge appended → tile added to
grid.

## Error handling

- **Unmapped item** → `itemIconURL` returns `null` → SVG glyph (today's
  behavior).
- **Mapped but file missing** (sync not run / typo) → `<img>.onerror`
  removes the img and swaps in the SVG glyph. No broken-image icon.
- **`itemIconURL` / `itemIconSVG` absent** (script load failure) →
  guarded by `typeof` checks; tile renders without an icon rather than
  throwing.

## Testing

- **Unit (Node, no browser):** a small harness over `item-icon-map.js`
  asserting representative names resolve to the right icon
  (`"iron ingot"→metal_ingot`, `"bounty hunter's cloak"→cloak`,
  `"copper wire"→wire_coil`), that an unmapped name (`"rusty sword"`)
  returns `null`, and that normalization handles articles/case.
- **Manual local review (acceptance):** boot the server locally
  (instance-save wipe per SOP not needed — no data-file changes), open
  the web client, acquire/equip items whose names hit the map, and
  confirm painted icons render in the Worn / Components / Backpack tabs
  with correct sizing, transparent edges on the tile gradient, and
  intact charge meters / stack badges. Confirm an unmapped item still
  shows its SVG glyph.

## Out of scope (explicit follow-ups)

- Mapping the full 261-item catalogue and the 112 community icons (same
  mechanism, mapping-table additions only).
- Type/subtype `TYPE_MAP` category tier.
- Any server-side / GMCP change (this is entirely client + a copy
  script).
- Reserved-pool vitals viz and other unrelated card polish.

## Files touched

- **new** `tools/sync_item_icons.py`
- **new** `static/js/item-icon-map.js` (+ a Node test harness)
- **new (generated)** `_datafiles/html/public/static/images/items/*.png`
- **edit** `_datafiles/html/public/webclient-pure.html`
  (helper + dedupe + script include)
- **edit** `_datafiles/html/public/static/css/dashboard.css` (img rule)
