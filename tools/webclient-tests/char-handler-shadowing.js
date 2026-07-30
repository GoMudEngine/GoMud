#!/usr/bin/env node
/*
 * Guards the GMCP handler-shadowing trap in the web client.
 *
 *   node tools/webclient-tests/char-handler-shadowing.js
 *
 * Zero dependencies. Static analysis of webclient-pure.html — it does not
 * execute the UI, so it needs no DOM.
 *
 * THE BUG THIS EXISTS FOR (2026-07-30)
 * ------------------------------------
 * handleGMCP dispatches to the FIRST matching handler, longest path first, and
 * then RETURNS:
 *
 *     for (let i = nameParts.length-1; i >= 0; i--) {
 *         var path = nameParts.slice(0, i+1).join(".");
 *         if (GMCPUpdateHandlers[path]) { GMCPUpdateHandlers[path](); return; }
 *     }
 *
 * So a push named "Char" runs ONLY the generic "Char" handler. Every
 * "Char.<Sub>" handler is shadowed. The server names a sub-package (e.g.
 * "Char.Quests") only on the event that changes it; on login, refresh and
 * reconnect it sends the whole "Char" node instead — and GetCharNode sets
 * all = (gmcpModule == "Char"), so that payload contains every sub-object.
 *
 * Result: the Quests panel and the map's quest marker stayed empty for every
 * quest a player already had, while the `quests` command listed them fine.
 * Vitals and Inventory worked only because the "Char" handler happens to
 * delegate for those two.
 *
 * So: every Char.<Sub> handler must either be delegated to from the generic
 * "Char" handler, or be explicitly listed below as event-only (never present
 * in a full Char payload). A new one that is neither fails this test rather
 * than silently going stale on login.
 */
'use strict';

const fs = require('fs');
const path = require('path');

const HTML = path.join(__dirname, '..', '..', '_datafiles', 'html', 'public', 'webclient-pure.html');
const html = fs.readFileSync(HTML, 'utf8');

// Sub-packages that legitimately never arrive inside a full "Char" push, with
// the reason. Adding to this list is a deliberate claim — check GetCharNode in
// modules/gmcp/gmcp.Char.go before you do.
const EVENT_ONLY = {
  'Char.Automation': 'separate module push; the in-code NOTE explains re-rendering here causes DOM churn and resets tick timers',
  'Char.Mutation': 'one-shot reveal event (Gained), not part of the Char node',
  'Char.Action.Result': 'per-action event push',
  'Char.Action.Interrupted': 'per-action event push',
  'Char.Conditions': 'covered by updateStatusPanel(), which the Char handler calls unconditionally',
};

// ---------------------------------------------------------------- extraction
function handlerKeys(src) {
  // Keys of the GMCPUpdateHandlers object literal, e.g.  "Char.Quests": function() {
  const keys = new Set();
  const re = /"([A-Za-z][A-Za-z0-9.]*)"\s*:\s*function\s*\(/g;
  let m;
  while ((m = re.exec(src)) !== null) keys.add(m[1]);
  return keys;
}

function charHandlerBody(src) {
  const start = src.indexOf('"Char":function()');
  if (start < 0) throw new Error('could not find the generic "Char" handler — has it been renamed?');
  // Walk braces from the handler's opening { to its matching close.
  const open = src.indexOf('{', start);
  let depth = 0;
  for (let i = open; i < src.length; i++) {
    if (src[i] === '{') depth++;
    else if (src[i] === '}') {
      depth--;
      if (depth === 0) return src.slice(open, i + 1);
    }
  }
  throw new Error('unbalanced braces while reading the "Char" handler');
}

// ---------------------------------------------------------------- checks
const keys = handlerKeys(html);
const body = charHandlerBody(html);

const subKeys = [...keys].filter((k) => k.startsWith('Char.')).sort();
if (subKeys.length === 0) {
  console.error('FAIL  found no Char.* handlers at all — the parser is broken, not the client');
  process.exit(1);
}

let fails = 0;
console.log('Char.* handlers found: ' + subKeys.length + '\n');

for (const key of subKeys) {
  const sub = key.slice('Char.'.length); // e.g. Quests
  // Delegated either explicitly by key, or by calling a renderer while
  // guarding on the sub-object (the renderInventory / obj.Inventory shape).
  const delegatesByKey = body.includes(`GMCPUpdateHandlers['${key}']`) ||
                         body.includes(`GMCPUpdateHandlers["${key}"]`);
  const guardsOnSubObject = new RegExp('obj\\.' + sub.split('.')[0] + '\\b').test(body);
  const delegated = delegatesByKey || guardsOnSubObject;

  if (delegated) {
    console.log('  OK        ' + key + (delegatesByKey ? '  (delegates)' : '  (guards on obj.' + sub.split('.')[0] + ')'));
  } else if (EVENT_ONLY[key]) {
    console.log('  OK        ' + key + '  (event-only: ' + EVENT_ONLY[key] + ')');
  } else {
    fails++;
    console.log('  FAIL      ' + key);
    console.log('              A full "Char" push will NOT refresh this panel, because');
    console.log('              handleGMCP stops at the generic "Char" handler. Either');
    console.log('              delegate to it from that handler, or add it to EVENT_ONLY');
    console.log('              with a reason if it truly never rides in a Char payload.');
  }
}

// The specific regression: Quests must be delegated, not merely tolerated.
if (!body.includes("GMCPUpdateHandlers['Char.Quests']")) {
  fails++;
  console.log('\n  FAIL      the "Char" handler no longer delegates to Char.Quests');
  console.log('              This is the exact 2026-07-30 bug: quests a player already');
  console.log('              had showed nothing in the panel or on the map until one of');
  console.log('              them advanced.');
}

console.log(fails === 0 ? '\nALL CHECKS PASSED' : '\n' + fails + ' CHECK(S) FAILED');
process.exit(fails ? 1 : 0);
