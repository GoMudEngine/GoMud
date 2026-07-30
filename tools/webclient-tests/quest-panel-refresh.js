#!/usr/bin/env node
/*
 * EXECUTES the real client dispatch to prove the Quests panel refreshes on a
 * full "Char" push — not merely that the delegation line exists.
 *
 *   node tools/webclient-tests/quest-panel-refresh.js
 *
 * Zero dependencies. Parses the real GMCPUpdateHandlers map, handleGMCP and
 * renderQuests out of webclient-pure.html, stubs just enough DOM, then feeds
 * the two payload shapes the server actually sends:
 *
 *   "Char.Quests"  — sent ONLY when a quest is granted or advances
 *   "Char"         — sent on login / refresh / reconnect, and contains the
 *                    same quest data (GetCharNode: all = gmcpModule == "Char")
 *
 * Before the 2026-07-30 fix the second case rendered nothing, because
 * handleGMCP stops at the first matching handler and the generic "Char" one
 * had no Quests branch. Players saw "No active quests" for everything they
 * already had.
 */
'use strict';

const fs = require('fs');
const path = require('path');

const HTML = path.join(__dirname, '..', '..', '_datafiles', 'html', 'public', 'webclient-pure.html');
const html = fs.readFileSync(HTML, 'utf8');

// ---------------------------------------------------------------- extraction
function sliceBalanced(src, startIdx, open, close) {
  const s = src.indexOf(open, startIdx);
  let depth = 0;
  for (let i = s; i < src.length; i++) {
    if (src[i] === open) depth++;
    else if (src[i] === close) {
      depth--;
      if (depth === 0) return src.slice(s, i + 1);
    }
  }
  throw new Error('unbalanced ' + open + close);
}
function funcSrc(src, signature) {
  const i = src.indexOf(signature);
  if (i < 0) throw new Error('not found: ' + signature);
  return src.slice(i, i + sliceBalanced(src, i, '{', '}').length + (src.indexOf('{', i) - i));
}

const handlersMap = sliceBalanced(html, html.indexOf('let GMCPUpdateHandlers ='), '{', '}');
const renderQuestsSrc = funcSrc(html, 'function renderQuests(');
const handleGMCPSrc = funcSrc(html, 'function handleGMCP(');
const escapeHTMLSrc = funcSrc(html, 'function escapeHTML(');

// ---------------------------------------------------------------- DOM stub
function El(tag) {
  this.tag = tag; this.className = ''; this.dataset = {}; this.children = [];
  this._html = ''; this._text = ''; this.style = {};
  Object.defineProperty(this, 'innerHTML', {
    get() { return this._html; },
    set(v) { this._html = v; this.children = []; }
  });
  Object.defineProperty(this, 'textContent', {
    get() { return this._text; }, set(v) { this._text = v; }
  });
}
El.prototype.appendChild = function (c) { c.parent = this; this.children.push(c); this._html += (c._html || c._text || ''); return c; };
El.prototype.insertBefore = function (n) { return this.appendChild(n); };
El.prototype.remove = function () {};
El.prototype.querySelector = function () { return null; };
El.prototype.querySelectorAll = function () { return []; };
El.prototype.addEventListener = function () {};
El.prototype.setAttribute = function () {};

const els = {};
function el(id) { if (!els[id]) els[id] = new El('div'); return els[id]; }

global.document = {
  createElement: (t) => new El(t),
  getElementById: (id) => el(id),
  querySelector: () => null,
  querySelectorAll: () => [],
  addEventListener: () => {},
};
global.window = { addEventListener: () => {} };
global.GMCPStructs = {};
global.console_ = console;

// Globals the handler map touches that we do not care about here.
const noop = () => {};
for (const name of [
  'updateStatusPanel', 'renderInventory', 'renderCommsTabs', 'updateVitalsWindow',
  'renderAutomation', 'rebuildTickTimers', 'renderRoom', 'setConnected',
  'renderMutationToast', 'onActionResult', 'onActionInterrupted', 'renderCommands',
  'SendGMCP', 'appendGameText', 'updateMapPanelTitle', 'renderParty',
  'drainQueue', 'updateActionBar', 'setCooldowns', 'refreshCommandList',
]) {
  if (!(name in global)) global[name] = noop;
}
// The map renderer: record marker calls instead of drawing. `rooms` mirrors
// RoomGridSVG's Map of the rooms currently ON the map — Zone.Map only carries
// the zone you are standing in, so a cross-zone target is legitimately absent.
const markerCalls = [];
const mapRooms = new Map();
global.gr = { setQuestMarker: (m) => markerCalls.push(m), rooms: mapRooms };

// eslint-disable-next-line no-eval
(0, eval)([
  escapeHTMLSrc,
  renderQuestsSrc,
  handlersMap.replace(/^\{/, 'globalThis.GMCPUpdateHandlers = {') + ';',
  handleGMCPSrc,
].join('\n'));

// ---------------------------------------------------------------- helpers
const QUESTS = [
  { id: 2, name: 'The Warren Compact', completion: 50, focused: true,
    hint: 'Bring a healing salve to the shaman.', target_room: 301, next_room: 300, next_dir: 'north' },
  { id: 6, name: "The Collector's Burden", completion: 33, focused: false, hint: 'Deliver to Clerk Pell.' },
];
function panelText() { return el('quests-list').innerHTML || ''; }

// The panel ships with this markup in webclient-pure.html (the #quests-list
// div). Seeding it matters: if the stub started empty, "does not contain
// 'No active quests'" would pass VACUOUSLY on a panel nothing ever touched —
// which is exactly the bug being tested for.
const PANEL_INITIAL = '<div class="quests-empty">No active quests.</div>';
function reset() {
  const e = new El('div');
  e.innerHTML = PANEL_INITIAL;
  els['quests-list'] = e;
  markerCalls.length = 0;
  global.GMCPStructs = {};
}

let fails = 0;
function check(name, cond, detail) {
  if (cond) console.log('  PASS  ' + name);
  else { fails++; console.log('  FAIL  ' + name + '\n          ' + detail); }
}

// ---------------------------------------------------------------- scenarios
console.log('SCENARIO 1 — "Char.Quests" push (a quest was granted or advanced)');
reset();
handleGMCP('Char.Quests', QUESTS);
console.log('   panel: ' + JSON.stringify(panelText().slice(0, 70)));
check('panel is not the empty state', !panelText().includes('No active quests'), panelText().slice(0, 120));
check('a quest name rendered', panelText().includes('Warren Compact'), panelText().slice(0, 120));
check('map marker set from the focused quest', markerCalls.length > 0 && markerCalls[markerCalls.length - 1] &&
  markerCalls[markerCalls.length - 1].targetRoom === 301, JSON.stringify(markerCalls));

console.log('SCENARIO 2 — full "Char" push (login / refresh / reconnect) — THE BUG');
reset();
handleGMCP('Char', {
  Info: { Name: 'Meirok' },
  Vitals: { hp: 100, hp_max: 100 },
  Quests: QUESTS,
});
console.log('   panel: ' + JSON.stringify(panelText().slice(0, 70)));
check('panel refreshed from the full Char payload', !panelText().includes('No active quests'),
  'panel still shows the empty state: ' + JSON.stringify(panelText().slice(0, 120)));
check('a quest name rendered', panelText().includes('Warren Compact'), panelText().slice(0, 120));
check('map marker set on login too', markerCalls.length > 0 && markerCalls[markerCalls.length - 1] &&
  markerCalls[markerCalls.length - 1].targetRoom === 301, JSON.stringify(markerCalls));

console.log('SCENARIO 3 — full "Char" push with NO quests clears the panel and marker');
reset();
handleGMCP('Char', { Info: { Name: 'Meirok' }, Quests: [] });
check('empty state shown', panelText().includes('No active quests'), JSON.stringify(panelText().slice(0, 120)));

console.log('SCENARIO 4 — SAME-zone target: the destination really is on the map');
reset();
mapRooms.clear();
mapRooms.set(301, { room: { RoomId: 301 } }); // target is in this zone's snapshot
handleGMCP('Char.Quests', QUESTS);
console.log('   label: ' + JSON.stringify((panelText().match(/qonmap">([^<]*)/) || [])[1] || '(none)'));
check('claims "marked on map"', panelText().includes('marked on map'), panelText().slice(0, 200));

console.log('SCENARIO 5 — CROSS-zone target: room 301 is NOT on the Watchers Crossing map');
reset();
mapRooms.clear();
mapRooms.set(420, { room: { RoomId: 420 } }); // only local rooms are on the map
handleGMCP('Char.Quests', QUESTS);
console.log('   label: ' + JSON.stringify((panelText().match(/qonmap">([^<]*)/) || [])[1] || '(none)'));
check('does NOT claim a marker that cannot be drawn',
  !panelText().includes('marked on map'),
  'panel promises a dot for an off-map room: ' + panelText().slice(0, 220));
check('shows the next-step direction instead', panelText().includes('head north'),
  panelText().slice(0, 220));

console.log(fails === 0 ? '\nALL CHECKS PASSED' : '\n' + fails + ' CHECK(S) FAILED');
process.exit(fails ? 1 : 0);
