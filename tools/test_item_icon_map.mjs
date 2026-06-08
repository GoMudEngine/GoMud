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
