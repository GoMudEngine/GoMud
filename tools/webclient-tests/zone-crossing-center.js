#!/usr/bin/env node
/*
 * Replays a zone crossing through the REAL RoomGridSVG to prove the map keeps a
 * current room — and therefore keeps drawing the quest arrow — on the step that
 * crosses a boundary.
 *
 *   node tools/webclient-tests/zone-crossing-center.js
 *
 * THE BUG (2026-07-30, reported from a 7-frame walk)
 * --------------------------------------------------
 * The quest arrow was correct in Watchers Crossing, VANISHED in the first room
 * of Dustwalk Road, then came back correctly one room later. "Shows up
 * different depending on my direction of travel."
 *
 * On a zone change the client receives Room.Info BEFORE Zone.Map. So:
 *   1. centerOnRoom(409) runs while 409 is not on the map yet -> returned early,
 *      leaving currentCenterId pointing at the OLD room.
 *   2. setZoneSnapshot() calls reset(), which nulls currentCenterId.
 *   3. _drawQuestArrow() bails on a null centre — it draws FROM the centre.
 *   4. Nothing calls centerOnRoom again, so it stayed null for that whole room.
 * The next in-zone move re-centred normally, which is why only the crossing
 * step lost the arrow.
 *
 * Fix: centerOnRoom remembers a centre it could not apply; setZoneSnapshot
 * applies it once the room arrives, before drawing the arrow.
 */
'use strict';

const fs = require('fs');
const path = require('path');

const JS = path.join(__dirname, '..', '..', '_datafiles', 'html', 'public', 'static', 'js', 'gmcp.js');
const src = fs.readFileSync(JS, 'utf8');

// ---------------------------------------------------------------- SVG stub
function El(tag) {
  this.tag = tag; this.children = []; this.attrs = {}; this.style = {};
  this._html = '';
  Object.defineProperty(this, 'innerHTML', {
    get() { return this._html; }, set(v) { this._html = v; this.children = []; }
  });
  Object.defineProperty(this, 'firstChild', { get() { return this.children[0] || null; } });
  Object.defineProperty(this, 'parentNode', { get() { return this._parent || null; } });
}
El.prototype.setAttribute = function (k, v) { this.attrs[k] = v; };
El.prototype.getAttribute = function (k) { return this.attrs[k]; };
El.prototype.appendChild = function (c) { c._parent = this; this.children.push(c); return c; };
El.prototype.removeChild = function (c) {
  const i = this.children.indexOf(c); if (i >= 0) this.children.splice(i, 1); return c;
};
El.prototype.append = function () {
  for (const c of arguments) { if (c && typeof c === 'object') this.appendChild(c); }
};
El.prototype.remove = function () { if (this._parent) this._parent.removeChild(this); };
El.prototype.addEventListener = function () {};
El.prototype.querySelector = function () { return null; };
El.prototype.querySelectorAll = function () { return []; };
El.prototype.getBoundingClientRect = function () { return { width: 400, height: 400 }; };
Object.defineProperty(El.prototype, 'clientWidth', { get() { return 400; } });
Object.defineProperty(El.prototype, 'clientHeight', { get() { return 400; } });

global.document = {
  createElement: (t) => new El(t),
  createElementNS: (ns, t) => new El(t),
  getElementById: () => new El('div'),
  querySelector: () => new El('div'),
  querySelectorAll: () => [],
  addEventListener: () => {},
};
global.window = { addEventListener: () => {}, devicePixelRatio: 1 };
global.requestAnimationFrame = (fn) => fn();

// eslint-disable-next-line no-eval
(0, eval)(src + '\n;globalThis.RoomGridSVG = RoomGridSVG;');

// ---------------------------------------------------------------- helpers
function mkGrid() {
  const g = new globalThis.RoomGridSVG(new El('div'), {});
  return g;
}
// Minimal snapshot room in the shape setZoneSnapshot expects.
function room(num, x, y, exits) {
  return { num, x, y, z: 0, symbol: '.', biome: 'road', name: 'r' + num, tags: [], exits: exits || [] };
}

let fails = 0;
function check(name, cond, detail) {
  if (cond) console.log('  PASS  ' + name);
  else { fails++; console.log('  FAIL  ' + name + '\n          ' + detail); }
}
function arrowCount(g) {
  // The arrow is the only <line> appended to connectionsGroup by _drawQuestArrow.
  return g._questArrowEl ? 1 : 0;
}

// ---------------------------------------------------------------- scenario
console.log('Walking west out of Watchers Crossing into Dustwalk Road.\n');

const g = mkGrid();

// --- In Watchers Crossing, standing at 420, quest points west to 409 ---
g.setZoneSnapshot('Watchers Crossing', [
  room(421, -7, 0, [{ to: 420, kind: 'normal', dx: -1, dy: 0, dz: 0 }]),
  room(420, -8, 0, [{ to: 421, kind: 'normal', dx: 1, dy: 0, dz: 0 }]),
], 0, []);
g.centerOnRoom(420);
g.setQuestMarker({ targetRoom: 301, nextRoom: 409, nextDir: 'west', name: 'The Warren Compact' });
check('in the old zone the centre is set', g.currentCenterId === 420, 'centre=' + g.currentCenterId);

// --- Move west: Room.Info arrives FIRST, naming a room not yet on the map ---
g.centerOnRoom(409);
check('centre request for an unknown room is remembered, not dropped',
  g._pendingCenterId === 409, '_pendingCenterId=' + g._pendingCenterId);

// --- Then Zone.Map for the new zone arrives (this reset()s the world) ---
g.setZoneSnapshot('Dustwalk Road', [
  room(409, 1, 5, [{ to: 406, kind: 'normal', dx: -1, dy: 0, dz: 0 }]),
  room(406, 0, 5, [{ to: 409, kind: 'normal', dx: 1, dy: 0, dz: 0 }]),
], 0, []);

check('centre is restored after the zone rebuild',
  g.currentCenterId === 409,
  'centre=' + g.currentCenterId + ' — null here is the reported bug: the room is not ' +
  'highlighted and the arrow cannot be drawn');

// --- Fresh marker for the new room: next hop is 406, west ---
g.setQuestMarker({ targetRoom: 301, nextRoom: 406, nextDir: 'west', name: 'The Warren Compact' });
check('the quest arrow is drawn in the room just across the border',
  arrowCount(g) === 1,
  'no arrow — this is exactly what the screenshots showed at Dustwalk Road, Eastern End');

// --- One more move within the zone: this always worked, guard it stays working ---
g.centerOnRoom(406);
g.setQuestMarker({ targetRoom: 301, nextRoom: 405, nextDir: 'north', name: 'The Warren Compact' });
check('still drawn after an in-zone move', arrowCount(g) === 1, 'arrow lost on a normal move');

console.log(fails === 0 ? '\nALL CHECKS PASSED' : '\n' + fails + ' CHECK(S) FAILED');
process.exit(fails ? 1 : 0);
