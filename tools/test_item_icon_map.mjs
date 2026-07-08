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

// Legendary BIS gear (rarity_tier 82) — NAME_MAP must win over generic keywords.
eq(itemIconURL({ name: "The Blackrazor", type: "weapon", subtype: "slashing" }), urlFor("blackrazor"), "bis: blackrazor (article)");
eq(itemIconURL({ name: "Blackrazor", type: "weapon", subtype: "slashing" }), urlFor("blackrazor"), "bis: blackrazor (displayname)");
eq(itemIconURL({ name: "Staff of the Hollow Choir", type: "weapon", subtype: "staff" }), urlFor("staff_of_the_hollow_choir"), "bis: hollow choir beats generic staff");
eq(itemIconURL({ name: "Phial of Second Birth", type: "potion", subtype: "drinkable" }), urlFor("phial_of_second_birth"), "bis: phial");
eq(itemIconURL({ name: "Vitalis Bandolier", type: "belt" }), urlFor("vitalis_bandolier"), "bis: bandolier");
eq(itemIconURL({ name: "Wayfarer's Bottomless Pack", type: "back" }), urlFor("wayfarers_bottomless_pack"), "bis: pack");
eq(itemIconURL({ name: "Aegis of Mockery", type: "offhand" }), urlFor("aegis_of_mockery"), "bis: aegis");
eq(itemIconURL({ name: "Thornwall Harness", type: "body" }), urlFor("thornwall_harness"), "bis: harness");
eq(itemIconURL({ name: "Seething Prism", type: "neck" }), urlFor("seething_prism"), "bis: prism");
eq(itemIconURL({ name: "Zephyr Treads", type: "feet" }), urlFor("zephyr_treads"), "bis: treads");
eq(itemIconURL({ name: "Warden's Lash", type: "weapon" }), urlFor("wardens_lash"), "bis: lash");
eq(itemIconURL({ name: "Relic Sidearm", type: "weapon" }), urlFor("relic_sidearm"), "bis: sidearm");
eq(itemIconURL({ name: "Ironhorn Warbow", type: "weapon", subtype: "ranged" }), urlFor("ironhorn_warbow"), "bis: warbow beats generic bow");
eq(itemIconURL({ name: "Coldlight Lance", type: "weapon" }), urlFor("coldlight_lance"), "bis: lance");
eq(itemIconURL({ name: "Warden-Step Greaves", type: "legs" }), urlFor("warden_step_greaves"), "bis: greaves (hyphen)");
eq(itemIconURL({ name: "Grey Warden Helm", type: "head" }), urlFor("grey_warden_helm"), "bis: helm");
eq(itemIconURL({ name: "Greyfield Striders", type: "feet" }), urlFor("greyfield_striders"), "bis: striders");
eq(itemIconURL({ name: "Hull-Plate Cuirass", type: "body" }), urlFor("hull_plate_cuirass"), "bis: cuirass (hyphen)");
eq(itemIconURL({ name: "Coldlight Mantle", type: "back" }), urlFor("coldlight_mantle"), "bis: mantle");

// Arena + Elemental Oasis instance loot pool (near-BIS).
eq(itemIconURL({ name: "Earthshaker Warhammer", type: "weapon", subtype: "bludgeoning" }), urlFor("earthshaker_warhammer"), "inst: warhammer beats generic hammer");
eq(itemIconURL({ name: "Champion Mace", type: "weapon", subtype: "bludgeoning" }), urlFor("champion_mace"), "inst: champion mace beats generic mace");
eq(itemIconURL({ name: "Crystal Sceptre", type: "weapon", subtype: "sceptre" }), urlFor("crystal_sceptre"), "inst: crystal sceptre");
eq(itemIconURL({ name: "Wind Scimitar", type: "weapon", subtype: "slashing" }), urlFor("wind_scimitar"), "inst: wind scimitar");
eq(itemIconURL({ name: "Drowned Claws", type: "weapon", subtype: "claws" }), urlFor("drowned_claws"), "inst: drowned claws");
eq(itemIconURL({ name: "Arena Tower Shield", type: "offhand" }), urlFor("arena_tower_shield"), "inst: tower shield");
eq(itemIconURL({ name: "Arena Chain Gloves", type: "gloves" }), urlFor("arena_chain_gloves"), "inst: chain gloves");
eq(itemIconURL({ name: "Arena Iron Greaves", type: "legs" }), urlFor("arena_iron_greaves"), "inst: iron greaves");
eq(itemIconURL({ name: "Volcanic Plate", type: "body" }), urlFor("volcanic_plate"), "inst: volcanic plate");
eq(itemIconURL({ name: "Ice Crown", type: "head" }), urlFor("ice_crown"), "inst: ice crown");
eq(itemIconURL({ name: "Mist Pauldrons", type: "shoulders" }), urlFor("mist_pauldrons"), "inst: mist pauldrons (repointed)");
eq(itemIconURL({ name: "Stone Ring", type: "ring" }), urlFor("stone_ring"), "inst: stone ring");
eq(itemIconURL({ name: "Storm Bracer", type: "wrist" }), urlFor("storm_bracer"), "inst: storm bracer (repointed)");
eq(itemIconURL({ name: "Tidal Torc", type: "neck" }), urlFor("tidal_torc"), "inst: tidal torc");

// Equip-keyword tier (weapon/armor nouns, safe on any name)
eq(itemIconURL({ name: "rusted broadsword" }), urlFor("finely_crafted_shortsword"), "equip-kw: broadsword->sword");
eq(itemIconURL({ name: "gnarled quarterstaff" }), urlFor("staff"), "equip-kw: staff");
eq(itemIconURL({ name: "hook-spear" }), urlFor("war_spear"), "equip-kw: spear");
eq(itemIconURL({ name: "iron knuckles" }), urlFor("spiked_knuckles"), "equip-kw: knuckles");

// Material-keyword tier
eq(itemIconURL({ name: "copper wire" }), urlFor("wire_coil"), "mat-kw: wire");
eq(itemIconURL({ name: "gnarled oak bark" }), urlFor("tree_bark"), "mat-kw: bark");

// Equipment-type-beats-material: a material word in an EQUIPMENT name must
// NOT hijack the icon (the bug the screenshot exposed).
eq(itemIconURL({ name: "chrysalis knuckles", type: "weapon", subtype: "fist" }), urlFor("spiked_knuckles"), "equip>mat: chrysalis knuckles (kw)");
eq(itemIconURL({ name: "chrysalis-forged greaves", type: "legs" }), urlFor("leather_pants"), "equip>mat: chrysalis greaves (type)");
eq(itemIconURL({ name: "Edrin's gnarled staff", type: "weapon", subtype: "staff" }), urlFor("staff"), "equip>mat: edrin staff");
eq(itemIconURL({ name: "satchel of bits", type: "componentbag" }), urlFor("component_satchel"), "equip>mat: component bag");
eq(itemIconURL({ name: "chain-link gloves", type: "gloves" }), urlFor("torn_gloves"), "equip>mat: chain gloves -> gloves");

// Keyword-shadow regressions (adjectives must not hijack later rules)
eq(itemIconURL({ name: "dusty scroll" }), urlFor("note"), "kw-shadow: dusty scroll -> note");
eq(itemIconURL({ name: "pitched battleaxe" }), urlFor("glowing_battleaxe"), "kw-shadow: pitched battleaxe -> axe");

// Null/undefined item contract
eq(itemIconURL(null), null, "null item -> null");
eq(itemIconURL(undefined), null, "undefined item -> null");

// Type/subtype tier (non-equip + equip)
eq(itemIconURL({ name: "mystery brew", type: "potion" }), urlFor("small_red_potion"), "type: potion");
eq(itemIconURL({ name: "odd hat", type: "head" }), urlFor("leather_cap"), "type: head");
eq(itemIconURL({ name: "big chopper", type: "weapon", subtype: "axe" }), urlFor("glowing_battleaxe"), "type: weapon-axe");
eq(itemIconURL({ name: "plain wooden pole", type: "weapon", subtype: "staff" }), urlFor("staff"), "type: weapon-staff");

// Fallback
eq(itemIconURL({ name: "ineffable thing", type: "nonsense" }), null, "fallback: unknown -> null");

// Manifest cross-check: every referenced icon basename must be a served file.
const manifest = new Set(JSON.parse(readFileSync(manifestPath, "utf8")));
const referenced = new Set();
for (const v of Object.values(TABLES.NAME_MAP)) referenced.add(v);
for (const [, v] of TABLES.EQUIP_KEYWORDS) referenced.add(v);
for (const [, v] of TABLES.MATERIAL_KEYWORDS) referenced.add(v);
for (const v of Object.values(TABLES.TYPE_MAP)) referenced.add(v);
for (const icon of referenced) {
  eq(manifest.has(icon + ".png"), true, `manifest has ${icon}.png`);
}

console.log(fails ? `\n${fails} FAILURES` : "\nALL PASS");
process.exit(fails ? 1 : 0);
