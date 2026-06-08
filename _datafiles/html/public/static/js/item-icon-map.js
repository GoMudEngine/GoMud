// item-icon-map.js — maps a GMCP item to a painted raster icon URL.
// Returns null when no pack icon fits, so the caller falls back to the
// monochrome SVG glyph from item-icons.js.
//
// Resolution order (first hit wins):
//   1. NAME_MAP         exact normalized item name -> icon basename
//   2. EQUIP_KEYWORDS   weapon/armor nouns (dagger, staff, cloak…) — safe
//                       to match on ANY item name
//   3. TYPE_MAP (if the item's type is equipment) — so e.g. "chrysalis
//      knuckles" resolves as a fist weapon, NOT the chrysalis material
//   4. MATERIAL_KEYWORDS  material/consumable nouns (ingot, ore, herb…)
//   5. TYPE_MAP (catch-all for non-equipment types: potion, food, key…)
//   6. null  -> SVG glyph fallback
//
// The equipment-type tier (step 3) sits BEFORE material keywords so a
// material word in an equipment NAME can't hijack the icon.
// Icons are served from window.ITEM_ICON_BASE (set by the page template),
// defaulting to "/static/images/items".
"use strict";
(function (global) {

  // Exact item-name -> icon basename. The 31 generated gap icons (from
  // ICON_GENERATION_SPEC covers-lists) plus generic community names.
  var NAME_MAP = {
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
    // generic community names (match if DOGMud has same-named item)
    "dagger": "dagger", "crowbar": "crowbar", "sling": "sling", "cudgel": "cudgel",
    "rope": "rope", "waterskin": "waterskin", "mug of ale": "mug_of_ale",
    "cheese sandwich": "cheese_sandwich", "mutton stew": "mutton_stew",
    "leather cap": "leather_cap", "leather vest": "leather_vest", "leather robe": "leather_robe",
    "cotton shirt": "cotton_shirt", "leather pants": "leather_pants", "worn boots": "worn_boots",
    "iron shield": "iron_shield", "wooden shield": "wooden_shield",
    "copper ring": "copper_ring", "cloth belt": "cloth_belt"
  };

  // Weapon / armor nouns. Safe to test against ANY item name because these
  // words don't appear in material names. Checked BEFORE material keywords.
  var EQUIP_KEYWORDS = [
    [/knuckle|cestus/, "spiked_knuckles"],
    [/\bdagger\b|\bdirk\b|\bstiletto\b/, "dagger"],
    [/broadsword|longsword|shortsword|\bsword\b|\bblade\b|\bsabre\b|scimitar|\brapier\b/, "finely_crafted_shortsword"],
    [/battleaxe|\baxe\b|hatchet/, "glowing_battleaxe"],
    [/\bspear\b|halberd|glaive|\bpike\b|polearm|trident|\blance\b|naginata/, "war_spear"],
    [/\bwhip\b|\blash\b|\bflail\b/, "long_whip"],
    [/\bsling\b|\bbow\b|crossbow|longbow|shortbow/, "sling"],
    [/\bclaw\b|\btalon\b/, "wooden_claw"],
    [/\bstaff\b|\bstave\b|quarterstaff/, "staff"],
    [/scepter|sceptre|\bwand\b|\brod\b/, "ancient_royal_scepter"],
    [/cudgel|\bclub\b|\bmace\b|mallet|\bbludgeon\b|warhammer|\bhammer\b/, "cudgel"],
    [/\bshield\b|buckler/, "iron_shield"],
    [/\bcloak\b|\bcape\b|\bmantle\b/, "cloak"],
    [/backpack|rucksack|travel pack/, "backpack"],
    [/pauldron|spaulder/, "pauldron"],
    [/bracer|bracelet|vambrace/, "bracer"],
    [/glove|gauntlet|\bmitt/, "torn_gloves"],
    [/tail (cap|band|sheath|guard)/, "tail_guard"]
  ];

  // Material / consumable nouns. Only reached for items whose type is NOT
  // equipment (so they can't override a weapon/armor icon).
  var MATERIAL_KEYWORDS = [
    [/\bingot\b/, "metal_ingot"],
    [/nodule|\bore\b/, "ore_nodule"],
    [/\bwire\b/, "wire_coil"],
    [/\bbark\b/, "tree_bark"],
    [/\bpitch\b|\bresin\b|beeswax|binding paste/, "resin_glob"],
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
    [/\bdust\b|residue|\bsalt\b/, "dust_pouch"],
    [/\bcrate\b|freight/, "crate"],
    [/lantern/, "oil_lantern"],
    [/chain link|\bchain\b/, "chain_link"],
    [/\bneedle\b|\bawl\b/, "bone_needle"],
    [/totem|fetish|figurine/, "carved_figurine"],
    [/satchel|component bag|component pouch/, "component_satchel"],
    [/\bkey\b/, "key"],
    [/\bvial\b|\bphial\b|decanter/, "glass_vial"],
    [/\bflask\b/, "flask"],
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

  // Item types that are worn/wielded equipment. For these, TYPE_MAP is
  // consulted before MATERIAL_KEYWORDS. Keys mirror the GMCP item.type
  // values (ItemType strings in internal/items/itemspec.go).
  var EQUIPMENT_TYPES = {
    "weapon": 1, "offhand": 1, "head": 1, "neck": 1, "shoulders": 1,
    "body": 1, "back": 1, "belt": 1, "wrist": 1, "gloves": 1, "ring": 1,
    "legs": 1, "feet": 1, "tail": 1, "componentbag": 1
  };

  // type-subtype (then type) -> representative icon. Keys mirror the SVG
  // glyph key scheme in item-icons.js so every glyphed item gets a raster.
  var TYPE_MAP = {
    "weapon": "guardsmans_broadsword",
    "weapon-sword": "finely_crafted_shortsword",
    "weapon-dagger": "dagger",
    "weapon-axe": "glowing_battleaxe",
    "weapon-bludgeoning": "cudgel",
    "weapon-cleaving": "captains_broadsword",
    "weapon-stabbing": "war_spear",
    "weapon-slashing": "captains_broadsword",
    "weapon-shooting": "sling",
    "weapon-claws": "wooden_claw",
    "weapon-fist": "spiked_knuckles",
    "weapon-whipping": "long_whip",
    "weapon-wand": "ancient_royal_scepter",
    "weapon-sceptre": "ancient_royal_scepter",
    "weapon-staff": "staff",
    "offhand": "iron_shield",
    "head": "leather_cap",
    "neck": "students_amulet",
    "shoulders": "pauldron",
    "body": "leather_vest",
    "back": "cloak",
    "belt": "cloth_belt",
    "wrist": "bracer",
    "gloves": "torn_gloves",
    "ring": "copper_ring",
    "legs": "leather_pants",
    "feet": "worn_boots",
    "tail": "tail_guard",
    "componentbag": "component_satchel",
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
  };

  function normalize(name) {
    return (name || "")
      .toLowerCase()
      .replace(/\s+/g, " ")
      .trim()
      .replace(/^(a|an|the) /, "");
  }

  function matchRules(rules, name) {
    for (var i = 0; i < rules.length; i++) {
      if (rules[i][0].test(name)) return rules[i][1];
    }
    return null;
  }

  function typeIcon(type, subtype) {
    if (!type) return null;
    var k = type + "-" + subtype;
    if (subtype && TYPE_MAP[k]) return TYPE_MAP[k];
    if (TYPE_MAP[type]) return TYPE_MAP[type];
    return null;
  }

  function lookupIcon(item) {
    var name = normalize(item && item.name);
    if (name && NAME_MAP[name]) return NAME_MAP[name];

    var hit = name ? matchRules(EQUIP_KEYWORDS, name) : null;
    if (hit) return hit;

    var type = ((item && item.type) || "").toLowerCase();
    var subtype = ((item && item.subtype) || "").toLowerCase();

    // Equipment types resolve via their type icon BEFORE material keywords,
    // so a material word in the name can't hijack a weapon/armor icon.
    if (EQUIPMENT_TYPES[type]) {
      hit = typeIcon(type, subtype);
      if (hit) return hit;
    }

    if (name) {
      hit = matchRules(MATERIAL_KEYWORDS, name);
      if (hit) return hit;
    }

    return typeIcon(type, subtype);
  }

  global.itemIconURL = function (item) {
    var icon = lookupIcon(item);
    if (!icon) return null;
    var base = global.ITEM_ICON_BASE || "/static/images/items";
    return base + "/" + icon + ".png";
  };

  // Exposed for the Node test harness (manifest cross-check).
  global.ITEM_ICON_TABLES = {
    NAME_MAP: NAME_MAP,
    EQUIP_KEYWORDS: EQUIP_KEYWORDS,
    MATERIAL_KEYWORDS: MATERIAL_KEYWORDS,
    TYPE_MAP: TYPE_MAP
  };

})(typeof window !== "undefined" ? window : globalThis);
