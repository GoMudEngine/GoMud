// Node test for RoomGridSVG.controlsModeForSize (no framework).
// Run: node tools/test_rgsvg_controls.mjs
//
// The map controls overlay the SVG; on a small map panel they cover the
// map. controlsModeForSize decides when the overlay switches to the
// compact/translucent treatment. This tests that pure threshold logic.
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import path from "node:path";
import vm from "node:vm";

const here = path.dirname(fileURLToPath(import.meta.url));
const repo = path.dirname(here);
const modPath = path.join(repo, "_datafiles/html/public/static/js/gmcp.js");

// gmcp.js is a browser classic-script: it has no top-level DOM execution,
// so it loads in a bare sandbox that just provides a `window` for its
// test export. ResizeObserver/document are only touched at runtime.
const sandbox = { window: {}, console };
vm.createContext(sandbox);
vm.runInContext(readFileSync(modPath, "utf8"), sandbox);
const RoomGridSVG = sandbox.window.RoomGridSVG;

let fails = 0;
function eq(actual, expected, label) {
  const ok = actual === expected;
  if (!ok) { fails++; console.error(`FAIL ${label}: got ${JSON.stringify(actual)} want ${JSON.stringify(expected)}`); }
  else console.log(`ok   ${label}`);
}

const f = RoomGridSVG.controlsModeForSize;
eq(typeof f, "function", "controlsModeForSize exists");
eq(f(320, 300), "normal", "full-size panel -> normal");
eq(f(240, 180), "normal", "threshold corner -> normal");
eq(f(239, 180), "compact", "just-narrow -> compact");
eq(f(300, 179), "compact", "just-short -> compact");
eq(f(220, 300), "compact", "narrow panel -> compact");
eq(f(320, 150), "compact", "short panel -> compact");
eq(f(120, 110), "compact", "tiny panel -> compact");

console.log(fails ? `\n${fails} FAILURES` : "\nALL PASS");
process.exit(fails ? 1 : 0);
