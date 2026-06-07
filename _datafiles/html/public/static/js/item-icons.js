// item-icons.js — Monochrome inline-SVG icon set for DOGMud inventory tiles.
// All icons authored in-house (not adapted from external sources).
// SVGs use stroke="currentColor", fill="none" so the tile CSS controls tinting.
// viewBox 0 0 100 100; stroke-width 6; stroke-linejoin/linecap round.
// Keys match the exact ItemType / ItemSubType string values from
// internal/items/itemspec.go that GMCP sends as item.type / item.subtype.
"use strict";
(function (global) {

  var S = 'stroke="currentColor" stroke-width="6" fill="none" stroke-linejoin="round" stroke-linecap="round"';
  function svg(inner) {
    return '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100" ' + S + '>' + inner + '</svg>';
  }

  var ICONS = {

    // ── Weapons (generic) ──────────────────────────────────────────────────
    // Diagonal blade with cross-guard
    "weapon": svg(
      '<path d="M30 80 L70 30 M62 22 q16 4 16 20 q-16 -4 -20 -16 Z"/>' +
      '<path d="M28 76 L24 80 M38 68 L16 78"/>'
    ),

    // ── Weapon subtypes ────────────────────────────────────────────────────
    // sword — classic double-edged blade
    "weapon-sword": svg(
      '<path d="M30 80 L70 30 M62 22 q16 4 16 20 q-16 -4 -20 -16 Z"/>' +
      '<path d="M28 76 L24 80 M38 68 L16 78"/>'
    ),

    // dagger — short blade, visible crossguard + pommel
    "weapon-dagger": svg(
      '<path d="M50 82 L50 28"/>' +
      '<path d="M50 28 L42 38 M50 28 L58 38"/>' +
      '<path d="M36 72 L64 72 M50 82 L44 88 L50 94 L56 88 Z"/>'
    ),

    // axe — single-bit axe head
    "weapon-axe": svg(
      '<path d="M40 80 L60 30"/>' +
      '<path d="M60 30 C75 20 82 36 72 46 C62 56 54 44 60 30 Z"/>' +
      '<path d="M38 76 L34 82"/>'
    ),

    // bludgeoning — mace/club with flanged head
    "weapon-bludgeoning": svg(
      '<path d="M50 82 L50 48"/>' +
      '<rect x="36" y="26" width="28" height="22" rx="4"/>' +
      '<path d="M38 26 L38 18 M50 26 L50 16 M62 26 L62 18"/>' +
      '<path d="M36 72 L64 72"/>'
    ),

    // cleaving — broad chopping blade
    "weapon-cleaving": svg(
      '<path d="M42 80 L62 28"/>' +
      '<path d="M62 28 C80 18 88 48 70 54 C52 60 44 42 62 28 Z"/>' +
      '<path d="M40 76 L34 82"/>'
    ),

    // stabbing — thin thrusting blade (rapier/spear tip)
    "weapon-stabbing": svg(
      '<path d="M50 86 L50 22 L42 36 M50 22 L58 36"/>' +
      '<path d="M36 72 L64 72"/>'
    ),

    // slashing — curved scimitar-style blade
    "weapon-slashing": svg(
      '<path d="M36 82 C30 50 46 26 68 22 C60 42 58 62 64 78"/>' +
      '<path d="M64 78 L36 82"/>' +
      '<path d="M34 78 L28 84"/>'
    ),

    // shooting — bow (arc + string)
    "weapon-shooting": svg(
      '<path d="M30 20 C16 50 30 80 30 80"/>' +
      '<path d="M30 20 L30 80"/>' +
      '<path d="M50 50 L26 50"/>' +
      '<path d="M50 50 L74 46 L68 52 L74 58 Z"/>'
    ),

    // claws — three curved claw marks
    "weapon-claws": svg(
      '<path d="M32 28 C28 50 34 72 38 82"/>' +
      '<path d="M50 22 C48 46 52 68 54 82"/>' +
      '<path d="M68 28 C72 50 66 72 62 82"/>'
    ),

    // fist — knuckle duster / gauntlet fist
    "weapon-fist": svg(
      '<rect x="28" y="44" width="44" height="28" rx="6"/>' +
      '<path d="M36 44 L36 30 Q36 24 42 24 Q48 24 48 30 L48 44"/>' +
      '<path d="M48 44 L48 28 Q48 22 54 22 Q60 22 60 28 L60 44"/>' +
      '<path d="M28 60 L22 60"/>'
    ),

    // whipping — curling whip line
    "weapon-whipping": svg(
      '<path d="M30 28 C24 36 36 44 28 54 C20 64 32 70 50 82"/>' +
      '<path d="M30 22 L36 34"/>'
    ),

    // wand — slender wand with star tip
    "weapon-wand": svg(
      '<path d="M62 28 L38 78"/>' +
      '<path d="M62 22 L62 34 M56 28 L68 28"/>' +
      '<circle cx="62" cy="28" r="6"/>'
    ),

    // sceptre — ornate rod with orb
    "weapon-sceptre": svg(
      '<path d="M50 82 L50 42"/>' +
      '<circle cx="50" cy="30" r="14"/>' +
      '<path d="M38 70 L62 70"/>'
    ),

    // staff — two-handed staff with crystal tip
    "weapon-staff": svg(
      '<path d="M38 86 L62 20"/>' +
      '<path d="M62 20 L56 28 L62 34 L68 28 Z"/>'
    ),

    // ── Armour / equipment slots ───────────────────────────────────────────
    // offhand — heater shield
    "offhand": svg(
      '<path d="M50 18 C75 26 80 26 80 26 C80 60 66 78 50 86 C34 78 20 60 20 26 C20 26 25 26 50 18 Z"/>'
    ),

    // head — open-faced helm (dome + cheek line)
    "head": svg(
      '<path d="M22 58 a28 28 0 0 1 56 0 Z"/>' +
      '<path d="M30 58 v10 h40 v-10"/>' +
      '<path d="M38 58 L38 64 M62 58 L62 64"/>'
    ),

    // neck — collar/amulet pendant
    "neck": svg(
      '<path d="M30 30 Q50 20 70 30"/>' +
      '<path d="M30 30 L26 50 Q26 60 38 60"/>' +
      '<path d="M70 30 L74 50 Q74 60 62 60"/>' +
      '<path d="M38 60 L50 76 L62 60"/>'
    ),

    // shoulders — symmetrical pauldrons
    "shoulders": svg(
      '<path d="M50 40 C34 36 20 44 18 58 C18 68 28 74 38 70 L42 56 Q50 50 58 56 L62 70 C72 74 82 68 82 58 C80 44 66 36 50 40 Z"/>'
    ),

    // body — tabard/shirt with collar + sleeves
    "body": svg(
      '<path d="M32 22 L20 34 L30 44 L34 38 V80 H66 V38 L70 44 L80 34 L68 22 L58 22 q-8 8 -16 0 Z"/>'
    ),

    // back — cloak silhouette
    "back": svg(
      '<path d="M30 30 Q50 18 70 30 L74 80 Q50 92 26 80 Z"/>' +
      '<path d="M30 30 L26 80 M70 30 L74 80"/>'
    ),

    // belt — horizontal belt with central buckle
    "belt": svg(
      '<path d="M16 46 H84 V58 H16 Z"/>' +
      '<rect x="42" y="44" width="16" height="14" rx="2"/>' +
      '<path d="M50 44 L50 58"/>'
    ),

    // wrist — bracelet band
    "wrist": svg(
      '<ellipse cx="50" cy="56" rx="26" ry="12"/>' +
      '<ellipse cx="50" cy="46" rx="26" ry="12"/>' +
      '<path d="M24 56 L24 46 M76 56 L76 46"/>'
    ),

    // ring — band with gemstone
    "ring": svg(
      '<circle cx="50" cy="58" r="22"/>' +
      '<path d="M40 32 L50 44 L60 32 Z"/>' +
      '<circle cx="50" cy="30" r="6"/>'
    ),

    // legs — pair of greaves/trousers
    "legs": svg(
      '<path d="M34 20 H66 L62 82 H52 L50 44 L48 82 H38 Z"/>'
    ),

    // feet — boot side profile
    "feet": svg(
      '<path d="M38 20 V60 Q38 74 52 74 H78 V62 H56 V20 Z"/>'
    ),

    // tail — curling tail shape
    "tail": svg(
      '<path d="M34 80 C28 60 36 36 54 30 C70 24 82 38 78 56 C74 70 60 76 50 68"/>' +
      '<path d="M50 68 L46 76 L54 76 Z"/>'
    ),

    // componentbag — drawstring pouch with label
    "componentbag": svg(
      '<path d="M40 18 h20 v8 h-20 Z"/>' +
      '<path d="M46 26 v40 a4 4 0 0 0 8 0 V26"/>' +
      '<path d="M32 34 C28 42 28 68 36 76 H64 C72 68 72 42 68 34 Z"/>'
    ),

    // ── Consumables ────────────────────────────────────────────────────────
    // potion — round-bottomed bottle
    "potion": svg(
      '<path d="M40 18 H60 V30 C70 36 76 50 76 62 C76 74 64 84 50 84 C36 84 24 74 24 62 C24 50 30 36 40 30 Z"/>' +
      '<path d="M40 18 H60"/>' +
      '<path d="M38 52 C42 46 58 46 62 52"/>'
    ),

    // food — drumstick / bread loaf
    "food": svg(
      '<path d="M28 72 C18 62 20 44 34 38 C38 36 42 38 44 42 L68 24 C74 18 84 24 80 34 L58 56 C62 58 64 62 62 68 C56 82 38 82 28 72 Z"/>'
    ),

    // drink — cup/goblet
    "drink": svg(
      '<path d="M30 20 H70 L62 56 H38 Z"/>' +
      '<path d="M38 56 L34 68 H66 L62 56"/>' +
      '<path d="M34 68 H66"/>'
    ),

    // scroll — rolled scroll
    "scroll": svg(
      '<rect x="24" y="28" width="52" height="44" rx="4"/>' +
      '<path d="M24 36 C18 36 18 64 24 64 M76 36 C82 36 82 64 76 64"/>' +
      '<path d="M34 42 H66 M34 52 H66 M34 62 H58"/>'
    ),

    // grenade — round bomb with fuse
    "grenade": svg(
      '<circle cx="50" cy="60" r="26"/>' +
      '<path d="M50 34 L50 22 C50 16 60 16 60 22 L54 26"/>'
    ),

    // junk — broken/cracked box
    "junk": svg(
      '<rect x="24" y="40" width="52" height="42" rx="3"/>' +
      '<path d="M24 52 H76 M40 40 L36 20 H64 L60 40"/>' +
      '<path d="M44 56 L50 64 L56 56"/>'
    ),

    // ── Other types ────────────────────────────────────────────────────────
    // readable — open book
    "readable": svg(
      '<path d="M50 24 L50 80"/>' +
      '<path d="M50 24 C40 20 22 22 18 26 L18 76 C22 72 40 70 50 74"/>' +
      '<path d="M50 24 C60 20 78 22 82 26 L82 76 C78 72 60 70 50 74"/>'
    ),

    // key — classic key shape
    "key": svg(
      '<circle cx="38" cy="44" r="18"/>' +
      '<circle cx="38" cy="44" r="8"/>' +
      '<path d="M52 52 L78 78"/>' +
      '<path d="M66 66 L72 60 M74 70 L80 64"/>'
    ),

    // object — generic box
    "object": svg(
      '<rect x="22" y="36" width="56" height="46" rx="4"/>' +
      '<path d="M22 36 L50 22 L78 36"/>' +
      '<path d="M50 22 L50 82 M22 36 H78"/>'
    ),

    // gemstone — faceted gem
    "gemstone": svg(
      '<path d="M50 82 L18 44 L32 22 H68 L82 44 Z"/>' +
      '<path d="M18 44 H82 M32 22 L50 82 M68 22 L50 82 M50 22 L50 82"/>'
    ),

    // lockpicks — set of picks
    "lockpicks": svg(
      '<path d="M30 80 L66 24 M66 24 C72 18 78 22 74 28 L38 82"/>' +
      '<path d="M42 72 L72 32 M72 32 C76 26 82 30 78 36 L46 78"/>'
    ),

    // botanical — stylized leaf/herb
    "botanical": svg(
      '<path d="M50 82 L50 40"/>' +
      '<path d="M50 40 C50 20 74 16 76 34 C78 50 58 58 50 52"/>' +
      '<path d="M50 52 C42 46 24 52 26 68 C28 82 50 82 50 76"/>'
    ),

    // ammo — two arrows
    "ammo": svg(
      '<path d="M26 74 L68 26 L74 32 M68 26 L62 26 M68 26 L68 32"/>' +
      '<path d="M34 78 L72 34 L78 40 M72 34 L66 34 M72 34 L72 40"/>'
    ),

    // service — ticket/star badge
    "service": svg(
      '<path d="M50 18 L58 38 H80 L62 52 L70 72 L50 60 L30 72 L38 52 L20 38 H42 Z"/>'
    ),

    // ── Fallback ───────────────────────────────────────────────────────────
    "_fallback": svg(
      '<rect x="22" y="22" width="56" height="56" rx="6"/>' +
      '<path d="M22 22 L78 78 M78 22 L22 78"/>'
    )
  };

  /**
   * Returns the best-match icon SVG string for an item's type/subtype.
   * Falls back to type-only, then "_fallback".
   *
   * @param {string} type    — item.type from GMCP (e.g. "weapon", "potion")
   * @param {string} subtype — item.subtype from GMCP (e.g. "sword", "drinkable")
   * @returns {string} inline SVG markup
   */
  global.itemIconSVG = function (type, subtype) {
    type    = (type    || "").toLowerCase();
    subtype = (subtype || "").toLowerCase();
    var specific = type + "-" + subtype;
    if (subtype && ICONS[specific]) return ICONS[specific];
    if (ICONS[type])                return ICONS[type];
    return ICONS["_fallback"];
  };

  global.ITEM_ICONS = ICONS;

})(window);
