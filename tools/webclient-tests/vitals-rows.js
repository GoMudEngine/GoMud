#!/usr/bin/env node
/*
 * Headless regression test for the web client's vitals panel row rendering.
 *
 *   node tools/webclient-tests/vitals-rows.js
 *
 * Zero dependencies. Not wired into CI — the web client has no JS test
 * infrastructure, and this exists because that gap let two real bugs ship:
 *
 *   1. Two same-named companions ("Earth Elemental") produced rows sharing
 *      one dataset.name, so dismissing one never pruned its row — the panel
 *      kept a stale, frozen duplicate.
 *   2. In a party, the companion row displayed the raw server key
 *      "Earth Elemental#7" — an internal instance id shown to the player.
 *
 * Both came from conflating row IDENTITY with row LABEL. The server keys
 * Party.Vitals as "Name#instanceId" (so same-named pets stay distinct) and
 * carries the clean name in the payload's .name field. The client must key
 * rows on the former and render the latter.
 *
 * This file parses the real functions out of webclient-pure.html rather than
 * copying them, so it cannot drift from shipping code. It locates them by
 * function name, not line number.
 */
'use strict';

const fs = require('fs');
const path = require('path');

const HTML = path.join(__dirname, '..', '..', '_datafiles', 'html', 'public', 'webclient-pure.html');

// ---------------------------------------------------------------- extraction
function extract(src, startMarker, endMarker) {
  const s = src.indexOf(startMarker);
  const e = src.indexOf(endMarker, s + 1);
  if (s < 0) throw new Error('marker not found in webclient-pure.html: ' + startMarker);
  if (e < 0) throw new Error('end marker not found in webclient-pure.html: ' + endMarker);
  return src.slice(s, e);
}

const html = fs.readFileSync(HTML, 'utf8');
const code =
  extract(html, 'function availablePct(', 'function evalTriggerCondition(') +
  '\n' +
  extract(html, 'function buildVitalsSeg(', 'function handleGMCP(');

// ---------------------------------------------------------------- DOM stub
function El(tag) {
  this.tag = tag; this.className = ''; this.dataset = {}; this.children = [];
  this._text = ''; this.style = {};
  Object.defineProperty(this, 'textContent', {
    get() { return this._text; }, set(v) { this._text = v; }
  });
}
El.prototype.appendChild = function (c) { c.parent = this; this.children.push(c); return c; };
El.prototype.insertBefore = function (n, ref) {
  const cur = this.children.indexOf(n); if (cur >= 0) this.children.splice(cur, 1);
  let i = this.children.indexOf(ref); if (i < 0) i = this.children.length;
  this.children.splice(i, 0, n); n.parent = this; return n;
};
El.prototype.remove = function () {
  if (!this.parent) return;
  const i = this.parent.children.indexOf(this);
  if (i >= 0) this.parent.children.splice(i, 1);
};
El.prototype._all = function (out) {
  out = out || [];
  for (const c of this.children) { out.push(c); c._all(out); }
  return out;
};
function unescapeSel(s) { return s.split('\\"').join('"').split('\\\\').join('\\'); }
function matches(el, sel) {
  let m = sel.match(/^\.([a-z-]+)$/);
  if (m) return (' ' + el.className + ' ').includes(' ' + m[1] + ' ');
  m = sel.match(/^\.([a-z-]+)\[data-name="(.*)"\]$/);
  if (m) {
    return (' ' + el.className + ' ').includes(' ' + m[1] + ' ') &&
      el.dataset.name === unescapeSel(m[2]);
  }
  return false;
}
El.prototype.querySelectorAll = function (sel) {
  return this._all().filter((e) => matches(e, sel));
};
El.prototype.querySelector = function (sel) { return this.querySelectorAll(sel)[0] || null; };

const container = new El('div');
global.document = {
  createElement: (t) => new El(t),
  getElementById: (id) => (id === 'vitals-bars' ? container : null)
};
global.GMCPStructs = {};
global.window = {};

// Indirect eval — runs in global scope so the extracted function declarations
// become globals. A direct eval() under 'use strict' keeps them module-local
// and every call below would throw ReferenceError.
// eslint-disable-next-line no-eval
(0, eval)(code);

// ---------------------------------------------------------------- helpers
function render() {
  // eslint-disable-next-line no-undef
  updateVitalsWindow();
  return container.querySelectorAll('.vitals-row').map((r) => {
    const n = r.querySelector('.vitals-name');
    return { key: r.dataset.name, label: n ? n.textContent : '' };
  });
}
function selfBase() {
  GMCPStructs.Char = {
    Vitals: {
      hp: 100, hp_max: 100, stamina: 50, stamina_max: 50,
      conviction: 10, conviction_max: 10
    },
    Info: { name: 'Ryn' }
  };
  GMCPStructs.Room = { Info: { name: 'A Room' } };
}
function pet(name, hp) {
  return { name, health: hp, stamina: 50, conviction: 0, location: 'A Room' };
}
let fails = 0;
function check(name, cond, detail) {
  if (cond) console.log('  PASS  ' + name);
  else { fails++; console.log('  FAIL  ' + name + '\n          ' + detail); }
}
const labels = (rows) => JSON.stringify(rows.map((x) => x.label));
const leaksId = (rows) => rows.some((x) => /#\d/.test(x.label));

// ---------------------------------------------------------------- scenarios
console.log('SCENARIO 1 — solo player, two same-named companions');
container.children = [];
selfBase();
GMCPStructs.Party = { Vitals: {
  'Earth Elemental#7': pet('Earth Elemental', 80),
  'Earth Elemental#9': pet('Earth Elemental', 30)
} };
const r1 = render();
console.log('   rows: ' + JSON.stringify(r1));
const c1 = r1.filter((x) => x.label === 'Earth Elemental');
check('both companions get a row', c1.length === 2, 'got ' + c1.length);
check('rows have DISTINCT keys', c1.length === 2 && c1[0].key !== c1[1].key, JSON.stringify(c1));
check('no internal id shown to the player', !leaksId(r1), labels(r1));
check('idempotent across re-renders', render().length === r1.length, 'row count changed');

console.log('SCENARIO 2 — in a party (NOTE: gated on Party.Members[{name}] + Party.Leader;');
console.log('             a wrong shape silently falls through to the solo branch)');
container.children = [];
selfBase();
GMCPStructs.Party = {
  Leader: 'Ryn',
  Members: [{ name: 'Ryn' }, { name: 'Malia' }],
  Vitals: {
    Ryn: { name: 'Ryn', health: 100, stamina: 100, conviction: 100, location: 'A Room' },
    Malia: { name: 'Malia', health: 90, stamina: 90, conviction: 90, location: 'A Room' },
    'Earth Elemental#7': pet('Earth Elemental', 80)
  }
};
const r2 = render();
console.log('   rows: ' + JSON.stringify(r2));
check('companion label has no "#id"', !leaksId(r2), labels(r2));
check('party members still render', r2.some((x) => x.label === 'Malia'), labels(r2));

console.log('SCENARIO 3 — dismissing one of two same-named companions prunes its row');
container.children = [];
selfBase();
GMCPStructs.Party = { Vitals: {
  'Earth Elemental#7': pet('Earth Elemental', 80),
  'Earth Elemental#9': pet('Earth Elemental', 30)
} };
render();
GMCPStructs.Party = { Vitals: { 'Earth Elemental#9': pet('Earth Elemental', 30) } };
const r3 = render();
console.log('   rows after dismissing #7: ' + JSON.stringify(r3));
const c3 = r3.filter((x) => x.label === 'Earth Elemental');
check('exactly one companion row remains', c3.length === 1, JSON.stringify(r3));
check('the survivor is #9', c3.length === 1 && c3[0].key === 'Earth Elemental#9', JSON.stringify(c3));

console.log(fails === 0 ? '\nALL CHECKS PASSED' : '\n' + fails + ' CHECK(S) FAILED');
process.exit(fails ? 1 : 0);
