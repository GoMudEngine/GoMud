// ── Leather drawing toolkit (emboss / hide-fray / craquelure / icons) ─────────
// Ported verbatim from docs/superpowers/specs/2026-06-06-mapper-leather-mockups/
// 03-emboss-craquelure.html and 02-connection-types.html.
// None of these are called by the existing render path; they are helpers for
// the aged tooled-leather style rewrite (future tasks).

var LEATHER_NS = "http://www.w3.org/2000/svg";

// Emboss highlight / shadow colors
var LEATHER_HI = "#efce8c", LEATHER_SH = "#140b04";

// Connection icon colors
var LEATHER_INK = "#c9a86a", LEATHER_LOCK = "#d0633f";

/** Seeded PRNG — returns a closure that produces deterministic floats in [0,1). */
function rng(seed) {
  return function() {
    seed |= 0;
    seed = seed + 0x6D2B79F5 | 0;
    var t = Math.imul(seed ^ seed >>> 15, 1 | seed);
    t = t + Math.imul(t ^ t >>> 7, 61 | t) ^ t;
    return ((t ^ t >>> 14) >>> 0) / 4294967296;
  };
}

/** SVG element factory using LEATHER_NS. */
function lEl(tag, attrs) {
  var e = document.createElementNS(LEATHER_NS, tag);
  for (var k in attrs) e.setAttribute(k, attrs[k]);
  return e;
}

/** SVG text element factory using LEATHER_NS. */
function lTxt(attrs, s) {
  var e = lEl("text", attrs);
  e.textContent = s;
  return e;
}

// ── Emboss helpers (highlight up-left, shadow down-right, face on top) ────────

function embLine(g, x1, y1, x2, y2, col, w, op, d) {
  g.appendChild(lEl("line", { x1: x1 + d, y1: y1 + d, x2: x2 + d, y2: y2 + d, stroke: LEATHER_SH, "stroke-width": w, opacity: 0.55 * op, "stroke-linecap": "round" }));
  g.appendChild(lEl("line", { x1: x1 - d, y1: y1 - d, x2: x2 - d, y2: y2 - d, stroke: LEATHER_HI, "stroke-width": w, opacity: 0.45 * op, "stroke-linecap": "round" }));
  g.appendChild(lEl("line", { x1: x1, y1: y1, x2: x2, y2: y2, stroke: col, "stroke-width": w, opacity: op, "stroke-linecap": "round" }));
}

function embCirc(g, cx, cy, r, faceStroke, faceFill, w, d) {
  g.appendChild(lEl("circle", { cx: cx + d, cy: cy + d, r: r, fill: "none", stroke: LEATHER_SH, "stroke-width": w, opacity: "0.55" }));
  g.appendChild(lEl("circle", { cx: cx - d, cy: cy - d, r: r, fill: "none", stroke: LEATHER_HI, "stroke-width": w, opacity: "0.4" }));
  g.appendChild(lEl("circle", { cx: cx, cy: cy, r: r, fill: faceFill, stroke: faceStroke, "stroke-width": w }));
}

function embText(g, x, y, attrs, s, face, d) {
  function mk(xx, yy, col, op) {
    var a = {};
    for (var k in attrs) a[k] = attrs[k];
    a.x = xx; a.y = yy; a.fill = col; a.opacity = op;
    g.appendChild(lTxt(a, s));
  }
  mk(x + d, y + d, LEATHER_SH, 0.5);
  mk(x - d, y - d, LEATHER_HI, 0.4);
  mk(x, y, face, attrs.opacity != null ? attrs.opacity : 1);
}

// ── Hide-path (frayed border outline) ─────────────────────────────────────────

function hidePath(W, H, m, fray, nickP, rnd) {
  var pts = [], step = 12;
  function edge(x0, y0, x1, y1, nx, ny) {
    var len = Math.hypot(x1 - x0, y1 - y0), n = Math.max(3, Math.round(len / step));
    for (var i = 0; i < n; i++) {
      var t = i / n, x = x0 + (x1 - x0) * t, y = y0 + (y1 - y0) * t, off = rnd() * fray;
      if (rnd() < nickP) off += fray * (1.4 + rnd() * 1.8);
      pts.push([x + nx * off, y + ny * off]);
    }
  }
  edge(m, m, W - m, m, 0, 1);
  edge(W - m, m, W - m, H - m, -1, 0);
  edge(W - m, H - m, m, H - m, 0, -1);
  edge(m, H - m, m, m, 1, 0);
  return "M" + pts.map(function(p) { return p[0].toFixed(1) + "," + p[1].toFixed(1); }).join(" L") + " Z";
}

// ── Craquelure (fine crack network pressed into the leather surface) ───────────

function craquelure(g, W, H, step, jit, rnd) {
  var nx = Math.floor((W - 48) / step), ny = Math.floor((H - 48) / step), grid = [];
  for (var iy = 0; iy <= ny; iy++) {
    grid[iy] = [];
    for (var ix = 0; ix <= nx; ix++)
      grid[iy][ix] = [24 + ix * step + (rnd() - 0.5) * jit, 24 + iy * step + (rnd() - 0.5) * jit];
  }
  function seg(a, b) {
    if (rnd() < 0.16) return;
    var mx = (a[0] + b[0]) / 2 + (rnd() - 0.5) * jit * 0.7,
        my = (a[1] + b[1]) / 2 + (rnd() - 0.5) * jit * 0.7,
        w  = (0.3 + rnd() * 0.4).toFixed(2),
        pp = a[0].toFixed(1) + "," + a[1].toFixed(1) + " " + mx.toFixed(1) + "," + my.toFixed(1) + " " + b[0].toFixed(1) + "," + b[1].toFixed(1);
    // dark groove + faint light lip just below-right for realism
    g.appendChild(lEl("polyline", { points: pp, fill: "none", stroke: "#120a04", "stroke-width": w, opacity: "0.34", "stroke-linejoin": "round", "stroke-linecap": "round" }));
    if (rnd() < 0.5)
      g.appendChild(lEl("polyline", {
        points: (a[0] + 0.5).toFixed(1) + "," + (a[1] + 0.5).toFixed(1) + " " +
                (mx + 0.5).toFixed(1) + "," + (my + 0.5).toFixed(1) + " " +
                (b[0] + 0.5).toFixed(1) + "," + (b[1] + 0.5).toFixed(1),
        fill: "none", stroke: "#caa86a", "stroke-width": "0.3", opacity: "0.10"
      }));
  }
  for (var iy2 = 0; iy2 <= ny; iy2++)
    for (var ix2 = 0; ix2 <= nx; ix2++) {
      if (ix2 < nx) seg(grid[iy2][ix2], grid[iy2][ix2 + 1]);
      if (iy2 < ny) seg(grid[iy2][ix2], grid[iy2 + 1][ix2]);
    }
}

// ── Connection icon helpers ───────────────────────────────────────────────────

/** Arrowhead at pixel (x,y) pointing along angle `ang` (radians), in color `col`. */
function arrowhead(g, x, y, ang, col) {
  var b = 6, c2 = 4, px1 = -Math.sin(ang), py1 = Math.cos(ang);
  [1, -1].forEach(function(sg) {
    g.appendChild(lEl("line", {
      x1: x, y1: y,
      x2: x - Math.cos(ang) * b + px1 * c2 * sg,
      y2: y - Math.sin(ang) * b + py1 * c2 * sg,
      stroke: col, "stroke-width": 1.6, "stroke-linecap": "round"
    }));
  });
}

/** Linear interpolation between two [x,y] points at parameter t. */
function mid(a, b, t) {
  return [a[0] + (b[0] - a[0]) * t, a[1] + (b[1] - a[1]) * t];
}

/**
 * Draw a locked-door icon at the midpoint of segment a→b.
 * a, b are pixel-coordinate arrays [x, y].
 * Extracted from drawConn type==="locked" in 02-connection-types.html.
 */
function drawLockedDoor(g, a, b, emboss) {
  var x1 = a[0], y1 = a[1], x2 = b[0], y2 = b[1];
  var m = mid(a, b, 0.5);
  var d = (emboss != null) ? emboss : 0.6; // emboss offset
  // near half (player side) normal; far half (beyond the locked door) rust-red
  g.appendChild(lEl("line", { x1: x1, y1: y1, x2: m[0], y2: m[1], stroke: "#9c8048", "stroke-width": 1.4, "stroke-dasharray": "3 3", opacity: "0.8" }));
  g.appendChild(lEl("line", { x1: m[0], y1: m[1], x2: x2, y2: y2, stroke: LEATHER_LOCK, "stroke-width": 1.4, "stroke-dasharray": "3 3", opacity: "0.9" }));
  // door silhouette (shadow, face, highlight)
  function dp(ox, oy) {
    var L = m[0] - 3.6 + ox, Rt = m[0] + 3.6 + ox, bot = m[1] + 5.4 + oy, sh = m[1] - 2 + oy, top = m[1] - 5.6 + oy, cxx = m[0] + ox;
    return "M" + L + "," + bot + " L" + L + "," + sh + " Q" + L + "," + top + " " + cxx + "," + top + " Q" + Rt + "," + top + " " + Rt + "," + sh + " L" + Rt + "," + bot + " Z";
  }
  g.appendChild(lEl("path", { d: dp(d, d), fill: LEATHER_SH, opacity: "0.5" }));
  g.appendChild(lEl("path", { d: dp(0, 0), fill: "#2a1d12", stroke: LEATHER_INK, "stroke-width": 1.1 }));
  g.appendChild(lEl("path", { d: dp(-d, -d), fill: "none", stroke: LEATHER_HI, "stroke-width": 0.8, opacity: "0.5" }));
  g.appendChild(lEl("line", { x1: m[0], y1: m[1] - 4.4, x2: m[0], y2: m[1] + 4.8, stroke: LEATHER_INK, "stroke-width": 0.5, opacity: "0.55" })); // plank seam
  g.appendChild(lEl("circle", { cx: m[0] + 1.7, cy: m[1] + 0.4, r: 0.95, fill: LEATHER_LOCK }));                                                   // keyhole circle
  g.appendChild(lEl("line", { x1: m[0] + 1.7, y1: m[1] + 0.8, x2: m[0] + 1.7, y2: m[1] + 2.4, stroke: LEATHER_LOCK, "stroke-width": 0.7 }));       // keyhole slot
}

/**
 * Draw an archway gate icon at the midpoint of segment a→b.
 * a, b are pixel-coordinate arrays [x, y].
 * Extracted from drawConn type==="gate" in 02-connection-types.html.
 */
function drawArchGate(g, a, b, emboss) {
  var x1 = a[0], y1 = a[1], x2 = b[0], y2 = b[1];
  var ang = Math.atan2(y2 - y1, x2 - x1);
  var m = mid(a, b, 0.5);
  var pxn = -Math.sin(ang), pyn = Math.cos(ang);
  var ca = Math.cos(ang), sa = Math.sin(ang);
  var d = (emboss != null) ? emboss : 0.6; // emboss offset
  // passage line (embossed)
  embLine(g, x1, y1, x2, y2, LEATHER_INK, 2.2, 0.92, d);
  // arch curve
  var f1 = [m[0] + pxn * 6, m[1] + pyn * 6], f2 = [m[0] - pxn * 6, m[1] - pyn * 6];
  var bow = 7, ctl = [m[0] + ca * bow, m[1] + sa * bow];
  function ap(ox, oy) {
    return "M" + (f1[0] + ox) + "," + (f1[1] + oy) +
           " Q" + (ctl[0] + ox) + "," + (ctl[1] + oy) +
           " " + (f2[0] + ox) + "," + (f2[1] + oy);
  }
  g.appendChild(lEl("path", { d: ap(d, d), fill: "none", stroke: LEATHER_SH, "stroke-width": 1.7, opacity: "0.55", "stroke-linecap": "round" }));
  g.appendChild(lEl("path", { d: ap(-d, -d), fill: "none", stroke: LEATHER_HI, "stroke-width": 1.7, opacity: "0.45", "stroke-linecap": "round" }));
  g.appendChild(lEl("path", { d: ap(0, 0), fill: "none", stroke: LEATHER_INK, "stroke-width": 1.7, "stroke-linecap": "round" }));
  // little feet/posts at the arch base
  [f1, f2].forEach(function(ff) {
    g.appendChild(lEl("line", { x1: ff[0] - ca * 2, y1: ff[1] - sa * 2, x2: ff[0] + ca * 2, y2: ff[1] + sa * 2, stroke: LEATHER_INK, "stroke-width": 1.6, "stroke-linecap": "round" }));
  });
}

// ─────────────────────────────────────────────────────────────────────────────

// Global instance counter for unique defs IDs when multiple map windows exist.
var _RGSVG_instanceCounter = 0;

class RoomGridSVG {
  constructor(selector, options = {}) {
      // ── Configurable options & defaults ───────────────────────────────
      this.cellSize = options.cellSize || 100;     // legacy
      this.cellMargin = options.cellMargin || 20;  // legacy
      this.roomSize = options.roomSize || 24;
      this.spacing = options.spacing || this.roomSize * 2;
      this.viewCells = options.viewCells || 6;
      this.zoomStep = options.zoomStep || 1.2;
      this.zoomLevel = options.initialZoom || 1;
      this.onRoomClick = options.onRoomClick || (() => {});
      this.zoomButtonSize = options.zoomButtonSize || 25;
      this.controlsMargin = options.controlsMargin || 10;
      this.biomeTints = options.biomeTints || RoomGridSVG.DEFAULT_BIOME_TINTS;

      // ── Instance id for unique defs (two map windows must not clash) ──
      this._iid = ++_RGSVG_instanceCounter;

      // ── Internal state ────────────────────────────────────────────────
      this.rooms = new Map();
      this.drawnEdges = new Set();
      this.drawnWrapStubs = new Set();
      this.drawnVerticalTicks = new Set();
      this.currentCenterId = null;
      // Centre requested before the room existed on the map (zone crossing:
      // Room.Info arrives before Zone.Map). Applied by setZoneSnapshot.
      this._pendingCenterId = null;
      this._zone = null;
      this._z = null;
      this._party = {}; // keyed by roomId (number or string)
      this._questMarker = null; // {targetRoom, nextRoom, nextDir, name} or null
      // Instance-stable def IDs (mirror the +iid convention of the leather defs)
      // so a second coexisting map instance never collides on these ids.
      this._questBrassId = "questBrass" + this._iid;
      this._questHeadId = "questHead" + this._iid;
      this._questArrowEl = null; // the single next-step arrow <line>, so we replace not accumulate

      // ── Build container ────────────────────────────────────────────────
      this.container = document.querySelector(selector);
      this.container.style.position = 'relative';

      // ── Outer SVG — fixed 320×300 design canvas ───────────────────────
      // The leather surface is drawn here in design coords (320×300).
      // A nested <svg> (worldSvg) holds the pannable/zoomable world.
      this.svg = document.createElementNS(LEATHER_NS, 'svg');
      this.svg.setAttribute('viewBox', '0 0 320 300');
      this.svg.setAttribute('preserveAspectRatio', 'xMidYMid meet');
      this.svg.style.width = '100%';
      this.svg.style.height = '100%';
      this.container.appendChild(this.svg);

      // Surface group — leather frame drawn in 320×300 design coords, never transformed.
      this.surfaceGroup = document.createElementNS(LEATHER_NS, 'g');
      this.svg.appendChild(this.surfaceGroup);

      // worldSvg — nested SVG for the pannable/zoomable room world.
      // Positioned inside the border interior: x=26, y=48, w=268, h=232.
      // The zoom viewBox is applied to this element.
      this.worldSvg = document.createElementNS(LEATHER_NS, 'svg');
      this.worldSvg.setAttribute('x', '26');
      this.worldSvg.setAttribute('y', '48');
      this.worldSvg.setAttribute('width', '268');
      this.worldSvg.setAttribute('height', '232');
      this.worldSvg.setAttribute('preserveAspectRatio', 'xMidYMid meet');
      // Default tiny viewBox until rooms exist:
      this.worldSvg.setAttribute('viewBox', '0 0 1 1');
      this.svg.appendChild(this.worldSvg);

      // Connections under rooms (both inside worldSvg, world coords):
      this.connectionsGroup = document.createElementNS(LEATHER_NS, 'g');
      this.worldSvg.appendChild(this.connectionsGroup);
      this.roomsGroup = document.createElementNS(LEATHER_NS, 'g');
      this.worldSvg.appendChild(this.roomsGroup);

      // overlayGroup — legend + compass, drawn in 320×300 design coords on
      // TOP of worldSvg so the pannable room world never covers them.
      this.overlayGroup = document.createElementNS(LEATHER_NS, 'g');
      this.svg.appendChild(this.overlayGroup);

      // ── HTML overlay zoom controls ────────────────────────────────────
      this._createHTMLControls();
  }

  // ── Public API ───────────────────────────────────────────────────────

  /**
   * Add or update a room in leather token style.
   */
  addRoom(room, deferEdges = false) {
      const id = room.RoomId;

      // 1) Pre-add exit-defined rooms — only when exit carries real coords.
      if (Array.isArray(room.Exits)) {
          room.Exits.forEach(e => {
              if (e && typeof e === 'object' && e.RoomId != null &&
                  Number.isFinite(e.x) && Number.isFinite(e.y)) {
                  if (this.rooms.has(e.RoomId)) return;
                  this.addRoom({
                      RoomId: e.RoomId,
                      Text: e.Text != null ? e.Text : String(e.RoomId),
                      x: e.x,
                      y: e.y,
                      Exits: Array.isArray(e.Exits) ? e.Exits : []
                  });
              }
          });
      }

      const isCurrent = (this.currentCenterId === id);
      const svc = this._serviceFor(room.tags);

      // 2) UPDATE existing
      if (this.rooms.has(id)) {
          const entry = this.rooms.get(id);
          entry.room.x = room.x;
          entry.room.y = room.y;
          entry.room.Exits = Array.isArray(room.Exits) ? room.Exits : [];
          entry.room.ExitsMeta = room.ExitsMeta || [];
          entry.room.Color = room.Color;
          entry.room.Text = room.Text;
          entry.room.tags = room.tags;
          entry.room.name = room.name;
          entry.room.biome = room.biome;

          // Re-draw the room token by clearing and rebuilding group contents
          while (entry.group.firstChild) entry.group.removeChild(entry.group.firstChild);
          this._buildRoomTokenInGroup(entry.group, room, isCurrent, svc);

          if (!deferEdges) this._drawEdgesForRoom(id);
          this._updateBounds();
          this._applyZoom();
          return;
      }

      // 3) NEW room
      const g = document.createElementNS(LEATHER_NS, 'g');
      g.setAttribute('data-room-id', id);
      g.style.cursor = 'pointer';
      g.addEventListener('click', () => this.onRoomClick(room));

      this._buildRoomTokenInGroup(g, room, isCurrent, svc);

      this.roomsGroup.appendChild(g);
      this.rooms.set(id, { room, group: g });

      if (!deferEdges) this._drawEdgesForRoom(id);
      this._updateBounds();
      this._applyZoom();
  }

  /**
   * Bulk-set rooms (wipes existing).
   */
  setRooms(arr) {
      this.reset();
      arr.forEach(r => this.addRoom(r));
  }

  /**
   * Clear world state (rooms + connections). Surface stays intact unless
   * caller also calls _renderSurface again.
   */
  reset() {
      this.rooms.clear();
      this.drawnEdges.clear();
      this.drawnWrapStubs.clear();
      this.drawnVerticalTicks.clear();
      this.currentCenterId = null;
      this._z = null;
      this.zoomLevel = 1;
      this.worldSvg.setAttribute('viewBox', '0 0 1 1');
      this.roomsGroup.innerHTML = '';
      this.connectionsGroup.innerHTML = '';
  }

  /**
   * Center & highlight a room. Re-draws that room as the current token.
   */
  centerOnRoom(id) {
      const entry = this.rooms.get(id);
      if (!entry) {
          // The room is not on the map YET. This is the normal case when
          // crossing a zone boundary: Room.Info arrives before Zone.Map, so we
          // are asked to centre on a room the new snapshot has not delivered.
          // Remember the request — setZoneSnapshot applies it once the room
          // exists. Without this the centre stayed null for the whole of that
          // room's visit (reset() clears it), which left the current room
          // un-highlighted and killed the quest arrow, since it is drawn FROM
          // the centre. The next in-zone move papered over it, which is why the
          // arrow only ever went missing on the step that crossed a border.
          this._pendingCenterId = id;
          return;
      }
      this._pendingCenterId = null;

      // Set the new center FIRST so _fog() computes distances from the new room
      // when we rebuild the previous room's token below.
      const prevCenterId = this.currentCenterId;
      this.currentCenterId = id;

      // un-highlight previous — rebuild its token as non-current (fog now correct)
      if (prevCenterId != null && prevCenterId !== id) {
          const prevEntry = this.rooms.get(prevCenterId);
          if (prevEntry) {
              const prevSvc = this._serviceFor(prevEntry.room.tags);
              while (prevEntry.group.firstChild) prevEntry.group.removeChild(prevEntry.group.firstChild);
              this._buildRoomTokenInGroup(prevEntry.group, prevEntry.room, false, prevSvc);
          }
      }

      // re-draw new current room as raised token
      const svc = this._serviceFor(entry.room.tags);
      while (entry.group.firstChild) entry.group.removeChild(entry.group.firstChild);
      this._buildRoomTokenInGroup(entry.group, entry.room, true, svc);

      this.center = {
          x: entry.room.x * this.spacing,
          y: entry.room.y * this.spacing
      };

      // The arrow is drawn FROM the current room, so it must be (re)drawn once
      // that room is known. Crossing a zone boundary calls reset(), which nulls
      // currentCenterId; setZoneSnapshot then rebuilds the map and calls
      // _drawQuestArrow() while the centre is still null, so it bailed out —
      // and centerOnRoom, which runs afterwards, never redrew it. The result
      // was the quest arrow vanishing on exactly the step that crosses into a
      // new zone, and staying gone until the next in-zone move.
      this._drawQuestArrow();

      this._applyZoom();
  }

  /**
   * Ingest a Zone.Map snapshot.
   * party = array of roomIds that hold party members (from Zone.Map payload).
   */
  setZoneSnapshot(zone, snapshotRooms, currentZ, party) {
      currentZ = currentZ || 0;

      // Build party lookup
      this._party = {};
      if (Array.isArray(party)) {
          party.forEach(rid => { this._party[rid] = true; });
      }

      // On zone OR floor change: reset world + re-render surface.
      if (this._zone !== zone || this._z !== currentZ) {
          this.reset();
          this._zone = zone;
          this._z = currentZ;
          // Clear surface + overlay groups and re-render the leather frame
          this.surfaceGroup.innerHTML = '';
          this.overlayGroup.innerHTML = '';
          this._renderSurface(zone, currentZ);
      }

      // Only render rooms on the current floor
      const floor = snapshotRooms.filter(r => (r.z || 0) === currentZ);

      // Pass 1: place/update every room node (defer edge drawing)
      floor.forEach(r => {
          this.addRoom({
              RoomId: r.num,
              x: r.x,
              y: r.y,
              z: r.z,
              symbol: r.symbol,
              biome: r.biome,
              name: r.name,
              tags: r.tags,
              Exits: (r.exits || []).map(e => ({ RoomId: e.to, kind: e.kind, dx: e.dx, dy: e.dy, dz: e.dz })),
              ExitsMeta: r.exits || []
          }, /* deferEdges = */ true);
      });

      // Pass 2: draw connectors/stubs/ticks for newly-seen rooms (edge dedup
      // skips ones already drawn), then the quest next-step arrow on top.
      // Tokens were already (re)built in Pass 1 — don't rebuild them again.
      floor.forEach(r => this._drawEdgesForRoom(r.num));

      // Apply a centre request that arrived before this snapshot did (see
      // centerOnRoom). Must run before the arrow is drawn — the arrow starts at
      // the centre room, so with the centre still null it would draw nothing.
      if (this._pendingCenterId != null && this.rooms.get(this._pendingCenterId)) {
          this.centerOnRoom(this._pendingCenterId);
      }

      this._drawQuestArrow();
      this._applyZoom();
  }

  /**
   * Set (or clear) the focused-quest destination marker + next-step arrow.
   * marker = {targetRoom, nextRoom, nextDir, name} or null. Only the room(s)
   * whose marker actually changed are rebuilt — never the whole map.
   */
  setQuestMarker(marker) {
      marker = marker || null;
      const a = this._questMarker, b = marker;
      const same = (!a && !b) || (!!a && !!b
          && a.targetRoom === b.targetRoom && a.nextRoom === b.nextRoom
          && a.nextDir === b.nextDir && a.name === b.name);
      if (same) { return; }
      const oldTarget = a ? a.targetRoom : 0;
      this._questMarker = marker;
      const newTarget = marker ? marker.targetRoom : 0;
      // Rebuild only the room tokens whose destination marker appears/clears.
      if (oldTarget && oldTarget !== newTarget) { this._rebuildRoomToken(oldTarget); }
      if (newTarget) { this._rebuildRoomToken(newTarget); }
      // Redraw the next-step arrow (removes any prior arrow).
      this._drawQuestArrow();
  }

  /** Rebuild a single room's token in place (e.g. to add/remove its marker). */
  _rebuildRoomToken(id) {
      const entry = this.rooms.get(id) || this.rooms.get(String(id)) || this.rooms.get(parseInt(id, 10));
      if (!entry) { return; }
      const isCurrent = (this.currentCenterId === entry.room.RoomId || this.currentCenterId === id);
      const svc = this._serviceFor(entry.room.tags);
      while (entry.group.firstChild) entry.group.removeChild(entry.group.firstChild);
      this._buildRoomTokenInGroup(entry.group, entry.room, isCurrent, svc);
  }

  /**
   * Draw a gold arrow from the current room toward the focused quest's
   * next step. If next_room is on the map, aim at its node; else, if a
   * compass dir is supplied (next_room still fogged), draw a short stub.
   */
  _drawQuestArrow() {
      // Replace, never accumulate: drop any prior arrow first.
      if (this._questArrowEl && this._questArrowEl.parentNode) {
          this._questArrowEl.parentNode.removeChild(this._questArrowEl);
      }
      this._questArrowEl = null;
      const qm = this._questMarker;
      if (!qm || this.currentCenterId == null || !qm.nextRoom) { return; }
      const fromE = this.rooms.get(this.currentCenterId);
      if (!fromE) { return; }
      // Never point at the room the player is already standing in. A marker can
      // name the current room if it went stale (the server recomputes next_room
      // from the player's position, so any lag between moving and the refresh
      // lands here), and the arrow would then be a zero-length line drawn on
      // their own node — which reads as "the marker is pointing at me".
      if (String(qm.nextRoom) === String(this.currentCenterId)) { return; }
      const sp = this.spacing;
      const from = { x: fromE.room.x * sp, y: fromE.room.y * sp };
      let to = null;
      const toE = this.rooms.get(qm.nextRoom) || this.rooms.get(String(qm.nextRoom))
                  || this.rooms.get(parseInt(qm.nextRoom, 10));
      if (toE) {
          to = { x: toE.room.x * sp, y: toE.room.y * sp };
      } else if (qm.nextDir) {
          const off = { north: [0, -1], south: [0, 1], east: [1, 0], west: [-1, 0], up: [0, 0], down: [0, 0] }[String(qm.nextDir).toLowerCase()];
          if (off && (off[0] || off[1])) {
              const stub = sp * 0.7;
              to = { x: from.x + off[0] * stub, y: from.y + off[1] * stub };
          }
      }
      if (!to) { return; }
      const dx = to.x - from.x, dy = to.y - from.y, len = Math.hypot(dx, dy) || 1;
      const ux = dx / len, uy = dy / len;
      const L = RoomGridSVG.LEATHER;
      const arrow = lEl("line", {
          x1: from.x + ux * 9, y1: from.y + uy * 9,
          x2: to.x - ux * 6, y2: to.y - uy * 6,
          stroke: L.questGold, "stroke-width": 2.6, "stroke-linecap": "round",
          "marker-end": "url(#" + this._questHeadId + ")"
      });
      this.connectionsGroup.appendChild(arrow);
      this._questArrowEl = arrow;
  }

  /** Zoom out so the whole map fits in view. */
  fit() {
      this._updateBounds();
      this.center = {
          x: this.bounds.minX * this.spacing + this.worldWidth / 2,
          y: this.bounds.minY * this.spacing + this.worldHeight / 2
      };
      const cw = this.container.clientWidth || 1;
      const ch = this.container.clientHeight || 1;
      const ar = (cw > 1 && ch > 1) ? ch / cw : 1;
      const span = this.viewCells * this.spacing;
      const zW = span / Math.max(this.worldWidth, 1);
      const zH = (span * ar) / Math.max(this.worldHeight, 1);
      this.zoomLevel = Math.min(zW, zH) * 0.92;
      this._applyZoom();
  }

  zoomIn()  { this.zoomLevel *= this.zoomStep; this._applyZoom(); }
  zoomOut() { this.zoomLevel /= this.zoomStep; this._applyZoom(); }

  drawConnection(a, b) {
      if (!this.rooms.has(a) || !this.rooms.has(b)) return;
      this._drawEdge(a, b);
      this._applyZoom();
  }

  // ── Private: surface rendering (fixed leather frame in 320×300) ───────────

  /**
   * Render the aged tooled-leather surface into this.surfaceGroup.
   * Ported from 01-surface-and-rooms.html render() — everything before
   * the room loop.
   */
  _renderSurface(zone, level) {
      const W = 320, H = 300;
      const g = this.surfaceGroup;
      const ov = this.overlayGroup;   // legend + compass float here, atop rooms
      const L = RoomGridSVG.LEATHER;
      const iid = this._iid;
      const questBrassId = this._questBrassId, questHeadId = this._questHeadId;

      // Deterministic seed from zone+level string so fray/craquelure are
      // stable per zone-floor.
      let seedVal = 0;
      const seedStr = String(zone) + String(level);
      for (let i = 0; i < seedStr.length; i++) seedVal = (seedVal * 31 + seedStr.charCodeAt(i)) | 0;
      const rnd = rng(seedVal);

      // ── <defs> ──────────────────────────────────────────────────────
      const defs = lEl("defs", {});

      // Radial gradient (leather background)
      const bgId = "lgbg" + iid;
      const bg = lEl("radialGradient", { id: bgId, cx: "45%", cy: "36%", r: "82%" });
      [["0%", "#43301d"], ["55%", "#2c1d11"], ["100%", "#170e07"]].forEach(function(s) {
          bg.appendChild(lEl("stop", { offset: s[0], "stop-color": s[1] }));
      });
      defs.appendChild(bg);

      // Grain filter
      const grainId = "lggrain" + iid;
      const f = lEl("filter", { id: grainId });
      f.appendChild(lEl("feTurbulence", { type: "fractalNoise", baseFrequency: "0.85", numOctaves: "2", stitchTiles: "stitch", result: "n" }));
      f.appendChild(lEl("feColorMatrix", { in: "n", type: "matrix", values: "0 0 0 0 0.05  0 0 0 0 0.035  0 0 0 0 0.02  0 0 0 0.6 0" }));
      defs.appendChild(f);

      // Vignette gradient
      const vigId = "lgvig" + iid;
      const vg = lEl("radialGradient", { id: vigId, cx: "50%", cy: "48%", r: "60%" });
      vg.appendChild(lEl("stop", { offset: "48%", "stop-color": "#000", "stop-opacity": "0" }));
      vg.appendChild(lEl("stop", { offset: "100%", "stop-color": "#000", "stop-opacity": L.vig.toFixed(2) }));
      defs.appendChild(vg);

      // Sheen gradient
      const sheenId = "lgsheen" + iid;
      const shg = lEl("radialGradient", { id: sheenId, cx: "28%", cy: "20%", r: "55%" });
      shg.appendChild(lEl("stop", { offset: "0%", "stop-color": "#fff4d8", "stop-opacity": "0.09" }));
      shg.appendChild(lEl("stop", { offset: "100%", "stop-color": "#fff4d8", "stop-opacity": "0" }));
      defs.appendChild(shg);

      // Hide clipPath
      const hd = hidePath(W, H, 4, L.fray, L.nickP, rnd);
      const clipId = "lgclip" + iid;
      const cp = lEl("clipPath", { id: clipId });
      cp.appendChild(lEl("path", { d: hd }));
      defs.appendChild(cp);

      // Quest brass radial gradient (destination marker ring / pin fill).
      const qbrass = lEl("radialGradient", { id: questBrassId, cx: "34%", cy: "26%", r: "70%" });
      [["0%", "#f4dd92"], ["46%", "#cb9f42"], ["100%", "#8a6620"]].forEach(function(s) {
          qbrass.appendChild(lEl("stop", { offset: s[0], "stop-color": s[1] }));
      });
      defs.appendChild(qbrass);

      // Quest next-step arrowhead marker.
      const qhead = lEl("marker", {
          id: questHeadId, markerWidth: 7, markerHeight: 7, refX: 5.2, refY: 3,
          orient: "auto", markerUnits: "strokeWidth", viewBox: "0 0 6 6"
      });
      qhead.appendChild(lEl("path", { d: "M0,0.4 L6,3 L0,5.6 L1.7,3 Z", fill: "#e8b84a" }));
      defs.appendChild(qhead);

      g.appendChild(defs);

      // ── Hide background ──────────────────────────────────────────────
      g.appendChild(lEl("path", { d: hd, fill: "url(#" + bgId + ")" }));
      g.appendChild(lEl("path", { d: hd, fill: "none", stroke: "#0e0703", "stroke-width": 2.4, "stroke-linejoin": "round", opacity: "0.9" }));

      // ── Clipped texture group (grain + sheen + craquelure) ───────────
      const tex = lEl("g", { "clip-path": "url(#" + clipId + ")" });
      g.appendChild(tex);
      tex.appendChild(lEl("rect", { x: 0, y: 0, width: W, height: H, filter: "url(#" + grainId + ")", opacity: "0.4" }));
      tex.appendChild(lEl("rect", { x: 0, y: 0, width: W, height: H, fill: "url(#" + sheenId + ")" }));
      craquelure(tex, W, H, L.crackStep, L.crackJit, rnd);

      // ── Embossed double border ───────────────────────────────────────
      const d = L.emboss;
      const bm = 18;
      [[bm, bm, W - bm, bm], [W - bm, bm, W - bm, H - bm],
       [W - bm, H - bm, bm, H - bm], [bm, H - bm, bm, bm]].forEach(function(ln) {
          embLine(g, ln[0], ln[1], ln[2], ln[3], L.ink, 2.2, 1, d);
      });
      // Inner hairline
      const bi = 24;
      [[bi, bi, W - bi, bi], [W - bi, bi, W - bi, H - bi],
       [W - bi, H - bi, bi, H - bi], [bi, H - bi, bi, bi]].forEach(function(ln) {
          g.appendChild(lEl("line", { x1: ln[0], y1: ln[1], x2: ln[2], y2: ln[3], stroke: L.ink, "stroke-width": 0.8, opacity: "0.8" }));
      });

      // ── Corner studs ─────────────────────────────────────────────────
      [[bm, bm, 1, 1], [W - bm, bm, -1, 1], [bm, H - bm, 1, -1], [W - bm, H - bm, -1, -1]].forEach(function(c4) {
          embCirc(g, c4[0] + c4[2] * 16, c4[1] + c4[3] * 16, 2.6, L.ink, L.ink, 0.6, d * 0.7);
      });

      // ── Title + subtitle ─────────────────────────────────────────────
      embText(g, W / 2, 36, {
          "text-anchor": "middle", "font-family": "Georgia,serif",
          "font-size": 13, "font-style": "italic", "font-weight": "bold"
      }, String(zone), L.title, d);
      g.appendChild(lTxt({
          x: W / 2, y: 46, "text-anchor": "middle",
          "font-family": "Georgia,serif", "font-size": 8, fill: L.ink
      }, "~ Level " + level + " ~"));

      // ── Embossed compass rose (top-right corner, floats atop rooms) ────
      const ccx = 279, ccy = 43, Rr = 9;
      ov.appendChild(lEl("circle", { cx: ccx, cy: ccy, r: Rr + 4, fill: L.legendBg, stroke: L.ink, "stroke-width": 1, opacity: "0.9" }));
      embCirc(ov, ccx, ccy, Rr, L.ink, "none", 0.8, d * 0.6);
      [[0, -1], [1, 0], [0, 1], [-1, 0]].forEach(function(dz) {
          embLine(ov, ccx, ccy, ccx + dz[0] * Rr, ccy + dz[1] * Rr, L.ink, 1.4, 1, d * 0.6);
      });
      embText(ov, ccx, ccy - Rr - 2, {
          "text-anchor": "middle", "font-family": "Georgia,serif",
          "font-size": 6.5, "font-weight": "bold"
      }, "N", L.ink, d * 0.5);

      // ── Legend card (bottom-right corner, floats atop rooms) ──────────
      // Rows: Party member | Bank/Shop/Store | Stairs | Road/trail/water
      const lw = 104, lh = 68, lx = 188, ly = 221;
      ov.appendChild(lEl("rect", { x: lx, y: ly, width: lw, height: lh, rx: 3, fill: L.legendBg, stroke: L.ink, "stroke-width": 1, opacity: "0.92" }));
      ov.appendChild(lEl("rect", { x: lx + 2.5, y: ly + 2.5, width: lw - 5, height: lh - 5, rx: 2, fill: "none", stroke: L.ink, "stroke-width": 0.4 }));
      embText(ov, lx + lw / 2, ly + 11, {
          "text-anchor": "middle", "font-family": "Georgia,serif",
          "font-size": 8, "font-weight": "bold", "font-style": "italic"
      }, "Legend", L.label, d * 0.5);

      const legendRows = [
          ["party",  "Party member"],
          ["svc",    "Bank / Shop / Store"],
          ["stairs", "Stairs ▲▼"],
          ["road",   "Road / trail / water"],
          ["quest",  "Quest destination"],
          ["qstep",  "Next step"]
      ];
      legendRows.forEach(function(rw, i) {
          var yy = ly + 20 + i * 8.5, sxc = lx + 11;
          if (rw[0] === "party") {
              // Tiny figure (verdigris)
              ov.appendChild(lEl("circle", { cx: sxc, cy: yy - 3, r: 1.2, fill: L.party }));
              ov.appendChild(lEl("path", { d: "M" + (sxc - 2) + "," + yy + " Q" + sxc + "," + (yy - 2.6) + " " + (sxc + 2) + "," + yy + " Z", fill: L.party }));
          } else if (rw[0] === "svc") {
              ov.appendChild(lEl("circle", { cx: sxc, cy: yy - 2, r: 3.4, fill: L.roomFill, stroke: L.ink, "stroke-width": 0.9 }));
              ov.appendChild(lTxt({ x: sxc, y: yy + 0.2, "text-anchor": "middle", "font-size": 4.4, "font-weight": "bold", fill: L.ink }, "$"));
          } else if (rw[0] === "stairs") {
              ov.appendChild(lTxt({ x: sxc, y: yy + 1, "text-anchor": "middle", "font-size": 7, "font-weight": "bold", fill: L.ink }, "▲"));
          } else if (rw[0] === "quest") {
              // Brass destination ring + patina core
              ov.appendChild(lEl("circle", { cx: sxc, cy: yy - 2, r: 3.2, fill: L.roomFill, stroke: "url(#" + questBrassId + ")", "stroke-width": 1.4 }));
              ov.appendChild(lEl("circle", { cx: sxc, cy: yy - 2, r: 1.2, fill: L.questPatina }));
          } else if (rw[0] === "qstep") {
              // Short gold next-step line with arrowhead
              ov.appendChild(lEl("line", { x1: sxc - 5, y1: yy - 2, x2: sxc + 3, y2: yy - 2, stroke: L.questGold, "stroke-width": 1.4, "marker-end": "url(#" + questHeadId + ")" }));
          } else {
              // Road sample line
              ov.appendChild(lEl("line", { x1: sxc - 5, y1: yy - 2, x2: sxc + 5, y2: yy - 2, stroke: L.road, "stroke-width": 2.6 }));
              ov.appendChild(lEl("line", { x1: sxc - 5, y1: yy - 2, x2: sxc + 5, y2: yy - 2, stroke: "#241810", "stroke-width": 0.6, "stroke-dasharray": "1 3" }));
          }
          ov.appendChild(lTxt({ x: lx + 22, y: yy + 1, "font-family": "Georgia,serif", "font-size": 6.6, fill: L.label }, rw[1]));
      });

      // ── Vignette (drawn last, on top, clipped to hide shape) ────────
      g.appendChild(lEl("rect", { x: 0, y: 0, width: W, height: H, fill: "url(#" + vigId + ")", "clip-path": "url(#" + clipId + ")", "pointer-events": "none" }));
  }

  // ── Private: room token construction ──────────────────────────────────────

  /**
   * Build leather room token DOM elements into group g.
   * g lives in roomsGroup (worldSvg world coords).
   * isCurrent = true for the player's current room.
   */
  _buildRoomTokenInGroup(g, room, isCurrent, svc) {
      const id = room.RoomId;
      const cx = room.x * this.spacing;
      const cy = room.y * this.spacing;
      const L = RoomGridSVG.LEATHER;
      const d = L.emboss;

      // Fog by Chebyshev distance from current center
      const fog = this._fog(room);

      // Tooltip
      const title = lEl("title", {});
      title.textContent = room.name || String(id);
      g.appendChild(title);

      if (isCurrent) {
          // Whisper warm glow
          g.appendChild(lEl("circle", { cx: cx, cy: cy, r: 13, fill: "#e8c98a", opacity: "0.07" }));
          // Outer emphasis ring
          embCirc(g, cx, cy, 10, L.ink, "none", 0.7, d * 1.3);
          // Raised token (deep emboss, lighter face)
          embCirc(g, cx, cy, 7.5, L.ink, "#3a2a18", 1.5, d * 2.4);
          // Italic name label beneath
          g.appendChild(lTxt({
              x: cx, y: cy + 20,
              "text-anchor": "middle", "font-family": "Georgia,serif",
              "font-size": 8, "font-style": "italic", fill: L.label
          }, room.name || ""));
      } else if (svc) {
          embCirc(g, cx, cy, 7, L.ink, L.roomFill, 1.3, d * 0.7);
          g.appendChild(lEl("circle", { cx: cx, cy: cy, r: 9.5, fill: "none", stroke: L.ink, "stroke-width": 0.8 }));
          embText(g, cx, cy + 3.2, {
              "text-anchor": "middle", "font-family": "Georgia,serif",
              "font-size": 9, "font-weight": "bold"
          }, svc.glyph, L.ink, d * 0.6);
      } else {
          embCirc(g, cx, cy, 6.5, L.ink, L.roomFill, 1.2, d * 0.7);
      }

      // Apply fog opacity to whole group (skip for current room — always fully visible)
      if (!isCurrent) {
          g.setAttribute("opacity", fog);
      }

      // Vertical stair ticks (▲/▼) — drawn from ExitsMeta vertical exits.
      // We just add placeholders here; the actual ticks are added by _drawVerticalTick
      // called from _drawEdgesForRoom. So nothing needed here.

      // Party marker (verdigris figure) at room's upper-right corner
      if (this._party[id] || this._party[String(id)]) {
          const mx = cx + 8, my = cy - 8;
          // Emboss shadow
          g.appendChild(lEl("circle", { cx: mx + 0.5, cy: my + 0.5, r: 3.4, fill: LEATHER_SH, opacity: "0.5" }));
          // Figure silhouette
          g.appendChild(lEl("circle", { cx: mx, cy: my, r: 3.4, fill: "#1c2e2a", stroke: L.partyDk, "stroke-width": 0.6 }));
          g.appendChild(lEl("circle", { cx: mx, cy: my - 1.1, r: 1.25, fill: L.party }));
          g.appendChild(lEl("path", { d: "M" + (mx - 2) + "," + (my + 2.6) + " Q" + mx + "," + (my - 0.4) + " " + (mx + 2) + "," + (my + 2.6) + " Z", fill: L.party }));
      }

      // Focused-quest destination marker (brass ring + copper-patina glow +
      // brass pin + italic name) on the target room.
      const qm = this._questMarker;
      if (qm && qm.targetRoom && (qm.targetRoom === id || qm.targetRoom === String(id))) {
          g.appendChild(lEl("circle", { cx: cx, cy: cy, r: 12, fill: L.questPatina, opacity: "0.30" }));
          g.appendChild(lEl("circle", { cx: cx, cy: cy, r: 8.5, fill: L.roomFill, stroke: "url(#" + this._questBrassId + ")", "stroke-width": 2.2 }));
          g.appendChild(lEl("circle", { cx: cx, cy: cy, r: 3, fill: L.questPatina }));
          g.appendChild(lEl("path", { d: "M" + cx + "," + (cy - 14) + " L" + cx + "," + (cy - 24), stroke: "#cb9f42", "stroke-width": 1.6 }));
          g.appendChild(lEl("path", { d: "M" + cx + "," + (cy - 24) + " L" + (cx + 8) + "," + (cy - 21) + " L" + cx + "," + (cy - 18) + " Z", fill: "url(#" + this._questBrassId + ")" }));
          if (qm.name) {
              g.appendChild(lTxt({
                  x: cx, y: cy + 21, "text-anchor": "middle",
                  "font-family": "Georgia,serif", "font-size": "7.5",
                  "font-style": "italic", fill: L.questPatina
              }, qm.name));
          }
      }
  }

  /**
   * Chebyshev fog: d0=1.0, d1=0.92, d2+=0.74.
   */
  _fog(room) {
      if (this.currentCenterId == null) return 1;
      const center = this.rooms.get(this.currentCenterId);
      if (!center) return 1;
      const dd = Math.max(Math.abs(room.x - center.room.x), Math.abs(room.y - center.room.y));
      return dd === 0 ? 1 : dd === 1 ? 0.92 : 0.74;
  }

  // ── Private: connection rendering ─────────────────────────────────────────

  /**
   * Determine edge kind from the biomes of two rooms.
   * Water > trail > road > ridge > plain (precedence).
   */
  _edgeKind(biomeA, biomeB) {
      function biomeClass(b) {
          if (!b) return "plain";
          const bl = b.toLowerCase();
          if (bl === "water" || bl === "lake" || bl === "river") return "water";
          if (bl === "forest" || bl === "swamp" || bl === "marsh") return "trail";
          if (bl === "city" || bl === "town" || bl === "road") return "road";
          if (bl === "hills" || bl === "mountain") return "ridge";
          return "plain";
      }
      const ka = biomeClass(biomeA), kb = biomeClass(biomeB);
      if (ka === "water" || kb === "water") return "water";
      if (ka === "trail" || kb === "trail") return "trail";
      if (ka === "road"  || kb === "road")  return "road";
      if (ka === "ridge" || kb === "ridge") return "ridge";
      return "plain";
  }

  /**
   * Draw one connection from srcRoom's ExitsMeta entry e.
   * All drawing goes into this.connectionsGroup (world coords).
   */
  _drawConnection(srcId, e) {
      const L = RoomGridSVG.LEATHER;
      const d = L.emboss;
      const me = this.rooms.get(srcId);
      if (!me) return;
      const cx = me.room.x * this.spacing;
      const cy = me.room.y * this.spacing;

      // ── Vertical exits ───────────────────────────────────────────────
      if (e.kind === "vertical") {
          this._drawVerticalTick(srcId, e.dz || (e.dy < 0 ? 1 : -1));
          return;
      }

      // ── Cross-zone / stub / wrap exits ────────────────────────────────
      // Wrap exits (non_cartesian/toroidal zones) connect two placed rooms
      // across a large coordinate gap; render them as a short outbound edge
      // stub in the nominal direction rather than a misleading cross-map line.
      if (e.kind === "wrap" || e.stub || !this.rooms.has(e.to)) {
          const key = srcId + ":" + e.dx + ":" + e.dy;
          if (this.drawnWrapStubs.has(key)) return;
          this.drawnWrapStubs.add(key);

          let ux = e.dx === 0 ? 0 : (e.dx > 0 ? 1 : -1);
          let uy = e.dy === 0 ? 0 : (e.dy > 0 ? 1 : -1);
          const mag = Math.hypot(ux, uy) || 1;
          ux /= mag; uy /= mag;
          const s2 = this.spacing * 0.3, L2 = this.spacing * 0.45;
          const ex = cx + ux * (s2 + L2), ey = cy + uy * (s2 + L2);
          this.connectionsGroup.appendChild(lEl("line", {
              x1: cx + ux * s2, y1: cy + uy * s2, x2: ex, y2: ey,
              stroke: L.ink, "stroke-width": 1.2, "stroke-dasharray": "2 3", opacity: "0.7"
          }));
          // Cross-zone label
          if (e.tozone) {
              this.connectionsGroup.appendChild(lTxt({
                  x: ex + ux * 4, y: ey + uy * 4 + 3,
                  "text-anchor": ux > 0 ? "start" : (ux < 0 ? "end" : "middle"),
                  "font-family": "Georgia,serif", "font-size": 7, "font-style": "italic", fill: L.ink
              }, "→ " + e.tozone));
          }
          return;
      }

      // ── Full connections (srcId ↔ e.to, both placed) ─────────────────
      // Dedup: use canonical a<b key
      const a = srcId, b = e.to;
      const edgeKey = a < b ? (a + "-" + b) : (b + "-" + a);
      if (this.drawnEdges.has(edgeKey)) return;
      this.drawnEdges.add(edgeKey);

      const ra = me.room;
      const rbEntry = this.rooms.get(b);
      if (!rbEntry) return;
      const rb = rbEntry.room;
      const x1 = cx, y1 = cy;
      const x2 = rb.x * this.spacing, y2 = rb.y * this.spacing;
      const pt1 = [x1, y1], pt2 = [x2, y2];

      // Fog for connection = min of two endpoint fogs
      const fog = Math.min(this._fog(ra), this._fog(rb));

      // ── Special connection types (override biome styling) ────────────
      if (e.locked) {
          drawLockedDoor(this.connectionsGroup, pt1, pt2, RoomGridSVG.LEATHER.emboss);
          return;
      }
      if (e.secret) {
          this.connectionsGroup.appendChild(lEl("line", {
              x1: x1, y1: y1, x2: x2, y2: y2,
              stroke: "#8a7a55", "stroke-width": 1.1, "stroke-dasharray": "1 4", opacity: (0.6 * fog).toFixed(2)
          }));
          const m2 = mid(pt1, pt2, 0.5);
          this.connectionsGroup.appendChild(lTxt({
              x: m2[0], y: m2[1] - 3, "text-anchor": "middle",
              "font-family": "Georgia,serif", "font-size": 10, "font-weight": "bold",
              fill: "#8a7a55", opacity: (0.85 * fog).toFixed(2)
          }, "?"));
          return;
      }
      if (e.oneway) {
          const ang = Math.atan2(y2 - y1, x2 - x1);
          embLine(this.connectionsGroup, x1, y1, x2, y2, L.ink, 1.6, 0.9 * fog, d);
          const ap2 = mid(pt1, pt2, 0.72);
          arrowhead(this.connectionsGroup, ap2[0], ap2[1], ang, L.ink);
          return;
      }
      if (e.gate) {
          drawArchGate(this.connectionsGroup, pt1, pt2, RoomGridSVG.LEATHER.emboss);
          return;
      }

      // ── Biome-based styling ───────────────────────────────────────────
      const kind = this._edgeKind(ra.biome, rb.biome);
      if (kind === "road") {
          embLine(this.connectionsGroup, x1, y1, x2, y2, L.road, 3.0, 0.95 * fog, d);
          this.connectionsGroup.appendChild(lEl("line", {
              x1: x1, y1: y1, x2: x2, y2: y2,
              stroke: "#241810", "stroke-width": 0.8, "stroke-dasharray": "1 3", opacity: (0.8 * fog).toFixed(2)
          }));
          // Long road: two league tick marks
          if (e.kind === "long") {
              const ang = Math.atan2(y2 - y1, x2 - x1);
              const pxn = -Math.sin(ang), pyn = Math.cos(ang);
              [0.38, 0.62].forEach(function(tt) {
                  const p = mid(pt1, pt2, tt);
                  this.connectionsGroup.appendChild(lEl("line", {
                      x1: p[0] + pxn * 4, y1: p[1] + pyn * 4,
                      x2: p[0] - pxn * 4, y2: p[1] - pyn * 4,
                      stroke: L.road, "stroke-width": 1.2, opacity: fog
                  }));
              }, this);
          }
      } else if (kind === "trail") {
          this.connectionsGroup.appendChild(lEl("line", {
              x1: x1 + d * 0.6, y1: y1 + d * 0.6, x2: x2 + d * 0.6, y2: y2 + d * 0.6,
              stroke: LEATHER_SH, "stroke-width": 1.5, "stroke-dasharray": "4 3", opacity: (0.4 * fog).toFixed(2)
          }));
          this.connectionsGroup.appendChild(lEl("line", {
              x1: x1, y1: y1, x2: x2, y2: y2,
              stroke: L.trail, "stroke-width": 1.5, "stroke-dasharray": "4 3", opacity: (0.9 * fog).toFixed(2)
          }));
      } else if (kind === "water") {
          this.connectionsGroup.appendChild(lEl("line", {
              x1: x1, y1: y1, x2: x2, y2: y2,
              stroke: L.water, "stroke-width": 1.7, "stroke-dasharray": "2 3", opacity: (0.9 * fog).toFixed(2)
          }));
      } else if (kind === "ridge") {
          embLine(this.connectionsGroup, x1, y1, x2, y2, L.ridge, 1.7, 0.9 * fog, d);
      } else {
          // plain
          embLine(this.connectionsGroup, x1, y1, x2, y2, L.plain, 1.5, 0.85 * fog, d * 0.6);
      }
  }

  _drawEdgesForRoom(id) {
      const me = this.rooms.get(id);
      if (!me) return;
      const room = me.room;

      if (Array.isArray(room.ExitsMeta) && room.ExitsMeta.length) {
          room.ExitsMeta.forEach(e => {
              this._drawConnection(id, e);
          });
      } else {
          // Fallback path: no ExitsMeta — use Exits array with plain connectors
          const exits = Array.isArray(room.Exits) ? room.Exits : [];
          exits.forEach(e => {
              const to = (typeof e === 'object') ? e.RoomId : e;
              if (this.rooms.has(to)) this._drawEdge(id, to);
          });
          this.rooms.forEach(({ room: otherRoom }, otherId) => {
              if (otherId === id) return;
              const oe = Array.isArray(otherRoom.Exits) ? otherRoom.Exits : [];
              if (oe.some(x => ((typeof x === 'object') ? x.RoomId : x) === id)) {
                  this._drawEdge(otherId, id);
              }
          });
      }
  }

  /**
   * Legacy fallback plain connector (no ExitsMeta).
   */
  _drawEdge(a, b) {
      const key = a < b ? (a + "-" + b) : (b + "-" + a);
      if (this.drawnEdges.has(key)) return;
      this.drawnEdges.add(key);

      const ra = this.rooms.get(a).room;
      const rb = this.rooms.get(b).room;
      const x1 = ra.x * this.spacing, y1 = ra.y * this.spacing;
      const x2 = rb.x * this.spacing, y2 = rb.y * this.spacing;
      const L = RoomGridSVG.LEATHER;

      embLine(this.connectionsGroup, x1, y1, x2, y2, L.plain, 1.5, 0.85, L.emboss * 0.6);
  }

  /**
   * Draw a ▲/▼ tick on the room token for a vertical exit.
   * Tick appended to the room's own group (moves with the room).
   */
  _drawVerticalTick(id, dz) {
      const me = this.rooms.get(id); if (!me) return;
      const sign = dz > 0 ? 'u' : 'd';
      const key = id + ":" + sign;
      if (this.drawnVerticalTicks.has(key)) return;
      this.drawnVerticalTicks.add(key);

      const L = RoomGridSVG.LEATHER;
      const cx = me.room.x * this.spacing, cy = me.room.y * this.spacing;
      me.group.appendChild(lTxt({
          x: cx + 11, y: cy + (dz > 0 ? -5 : 8),
          "text-anchor": "middle", "font-size": 9, "font-weight": "bold",
          fill: L.ink, opacity: "0.95"
      }, dz > 0 ? "▲" : "▼"));
  }

  _updateBounds() {
      if (!this.rooms.size) {
          this.bounds = { minX: 0, maxX: 0, minY: 0, maxY: 0 };
      } else {
          const xs = [...this.rooms.values()].map(e => e.room.x);
          const ys = [...this.rooms.values()].map(e => e.room.y);
          this.bounds = {
              minX: Math.min(...xs), maxX: Math.max(...xs),
              minY: Math.min(...ys), maxY: Math.max(...ys)
          };
      }
      this.worldWidth  = (this.bounds.maxX - this.bounds.minX + 1) * this.spacing;
      this.worldHeight = (this.bounds.maxY - this.bounds.minY + 1) * this.spacing;

      if (!this.center && this.rooms.size) {
          this.center = {
              x: this.bounds.minX * this.spacing + this.worldWidth / 2,
              y: this.bounds.minY * this.spacing + this.worldHeight / 2
          };
      }
  }

  /**
   * Apply zoom by setting the viewBox on worldSvg (the interior panel).
   * The surface (surfaceGroup) never moves — it is always in 320×300 design coords.
   */
  _applyZoom() {
      const baseW = (this.viewCells * this.spacing) / this.zoomLevel;
      // worldSvg is 268×232; use its aspect ratio for the viewBox
      const ar = 232 / 268;
      const baseH = baseW * ar;
      const cx = this.center ? this.center.x : 0;
      const cy = this.center ? this.center.y : 0;
      this.worldSvg.setAttribute('viewBox', (cx - baseW / 2) + " " + (cy - baseH / 2) + " " + baseW + " " + baseH);
  }

  // ── Private: HTML overlay controls ────────────────────────────────────────

  _createHTMLControls() {
      // Brass map-instrument buttons — embossed engraved glyphs on a polished
      // brass face, to match the tooled-leather surface. Stylesheet injected
      // once per document; div positioning stays inline (per-instance margin).
      if (!document.getElementById('rgsvg-ctl-style')) {
          const st = document.createElement('style');
          st.id = 'rgsvg-ctl-style';
          st.textContent =
".rgsvg-brass{min-width:24px;height:24px;padding:0 6px;font-family:Georgia,serif;" +
"font-weight:bold;font-size:12px;line-height:1;color:#3b2a10;cursor:pointer;" +
"border:1px solid #5e431a;border-radius:5px;" +
"background:radial-gradient(circle at 34% 26%,#f4dd92 0%,#cb9f42 46%,#8a6620 100%);" +
"text-shadow:0 1px 0 rgba(255,244,206,0.55);" +
"box-shadow:0 2px 3px rgba(0,0,0,0.55),inset 0 1px 1px rgba(255,246,212,0.7)," +
"inset 0 -2px 3px rgba(74,52,16,0.55);" +
"transition:filter .08s ease,box-shadow .08s ease,transform .08s ease;}" +
".rgsvg-brass:hover{filter:brightness(1.08);}" +
".rgsvg-brass:active{transform:translateY(1px);" +
"box-shadow:0 1px 1px rgba(0,0,0,0.4),inset 0 2px 4px rgba(74,52,16,0.7)," +
"inset 0 -1px 1px rgba(255,246,212,0.4);}" +
".rgsvg-controls{transition:opacity .15s ease,transform .15s ease;}" +
".rgsvg-controls.compact{opacity:0.32;transform:scale(0.8);transform-origin:bottom left;}" +
".rgsvg-controls.compact:hover,.rgsvg-controls.compact:focus-within{opacity:1;}";
          document.head.appendChild(st);
      }

      const div = document.createElement('div');
      div.className = 'rgsvg-controls';
      this.controlsEl = div;
      div.style.cssText =
          `position:absolute;bottom:${this.controlsMargin}px;left:${this.controlsMargin}px;` +
          `display:flex;gap:5px;z-index:5;`;
      const mk = (lbl, cb) => {
          const b = document.createElement('button');
          b.textContent = lbl;
          b.className = 'rgsvg-brass';
          b.addEventListener('click', cb);
          return b;
      };
      div.append(
          mk('fit',  () => this.fit()),
          mk('ctr',  () => this.centerOnRoom(this.currentCenterId)),
          mk('−', () => this.zoomOut()),
          mk('+',    () => this.zoomIn())
      );
      this.container.appendChild(div);

      // Keep the overlay from covering the map when the panel is small:
      // re-evaluate the compact/normal mode whenever the container resizes.
      this._applyControlsMode();
      if (typeof ResizeObserver !== 'undefined' && !this._controlsRO) {
          this._controlsRO = new ResizeObserver(() => this._applyControlsMode());
          this._controlsRO.observe(this.container);
      }
  }

  // Apply the size-driven controls mode (see RoomGridSVG.controlsModeForSize).
  _applyControlsMode() {
      if (!this.controlsEl) return;
      const mode = RoomGridSVG.controlsModeForSize(
          this.container.clientWidth, this.container.clientHeight);
      this.controlsEl.className = 'rgsvg-controls ' + mode;
  }
}

// ── Biome tint lookup table (legacy — kept for tintFor compat) ────────────────
RoomGridSVG.DEFAULT_BIOME_TINTS = {
  "city":     "#3a342c", "town":     "#3a342c",
  "forest":   "#25382a", "swamp":    "#243226", "marsh":    "#243226",
  "water":    "#243246", "lake":     "#243246", "river":    "#243246",
  "hills":    "#3e3422", "mountain": "#3e3422",
  "cave":     "#2c2530", "dungeon":  "#2c2530",
  "desert":   "#3e3622", "road":     "#3a3226",
  "_default": "#2a2018"
};

RoomGridSVG.prototype.tintFor = function (biome) {
  if (!biome) return this.biomeTints._default;
  const key = String(biome).toLowerCase();
  return this.biomeTints[key] || this.biomeTints._default;
};

// ── Leather style palette & render parameters ─────────────────────────────────
RoomGridSVG.LEATHER = {
  ink: "#c9a86a", ink2: "#9c8048", title: "#e8d2a0", roomFill: "#2a1d12",
  label: "#e8d8b8", locked: "#d0633f", water: "#6f99c0", trail: "#a98a55",
  road: "#c9a86a", ridge: "#a98a55", plain: "#9c8048",
  legendBg: "#241810", party: "#6bb0a0", partyDk: "#243f3a",
  questGold: "#e8b84a", questPatina: "#6bb0a0",
  emboss: 0.6, fray: 3.4, nickP: 0.08, crackStep: 24, crackJit: 9, vig: 0.5
};

// ── Service-room markers (bank / shop / storage) ──────────────────────────────
RoomGridSVG.SERVICE_GLYPHS = { bank: '$', shop: 'S', storage: '▢' };
RoomGridSVG.SERVICE_ORDER  = ['bank', 'shop', 'storage'];
RoomGridSVG.prototype._serviceFor = function (tags) {
  if (!Array.isArray(tags) || !tags.length) return null;
  for (let i = 0; i < RoomGridSVG.SERVICE_ORDER.length; i++) {
    const k = RoomGridSVG.SERVICE_ORDER[i];
    if (tags.indexOf(k) !== -1) return { key: k, glyph: RoomGridSVG.SERVICE_GLYPHS[k] };
  }
  return null;
};

// ── Responsive controls mode ──────────────────────────────────────────────────
// The fit/zoom controls are a fixed-pixel overlay while the map SVG scales to
// fit its container. On a small map panel the full-size buttons cover too much
// of the map, so below these thresholds the overlay switches to a compact,
// translucent-at-rest treatment (full opacity on hover/focus). Pure function so
// the threshold is unit-testable without a DOM.
RoomGridSVG.controlsModeForSize = function (w, h) {
  return (w < 240 || h < 180) ? 'compact' : 'normal';
};

// Test/diagnostic export — harmless in the browser (RoomGridSVG is already a
// classic-script global); lets the Node harness reach the static above.
if (typeof window !== 'undefined') { window.RoomGridSVG = RoomGridSVG; }
