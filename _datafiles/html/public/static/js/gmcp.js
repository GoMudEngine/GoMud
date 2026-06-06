// ── Leather drawing toolkit (dormant — consumed by later tasks) ───────────────
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
function drawLockedDoor(g, a, b) {
  var x1 = a[0], y1 = a[1], x2 = b[0], y2 = b[1];
  var m = mid(a, b, 0.5);
  var d = 0.6; // emboss offset
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
function drawArchGate(g, a, b) {
  var x1 = a[0], y1 = a[1], x2 = b[0], y2 = b[1];
  var ang = Math.atan2(y2 - y1, x2 - x1);
  var m = mid(a, b, 0.5);
  var pxn = -Math.sin(ang), pyn = Math.cos(ang);
  var ca = Math.cos(ang), sa = Math.sin(ang);
  var d = 0.6; // emboss offset
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

class RoomGridSVG {
  constructor(selector, options = {}) {
      // ── Configurable options & defaults ───────────────────────────────
      // cellSize + cellMargin set the grid pitch (spacing between node centres).
      // Node display size is roomSize; cellSize itself is not the node size.
      this.cellSize = options.cellSize || 100;     // legacy; no longer drives node size
      this.cellMargin = options.cellMargin || 20;  // legacy
      this.roomSize = options.roomSize || 24;      // hybrid node size (world units)
      this.spacing = options.spacing || this.roomSize * 2; // grid pitch ~2x node => node fills ~half the gap
      this.viewCells = options.viewCells || 6;     // cells across the default local view (stable; not whole-world)
      this.zoomStep = options.zoomStep || 1.2;
      this.zoomLevel = options.initialZoom || 1;
      this.onRoomClick = options.onRoomClick || (() => {});
      this.zoomButtonSize = options.zoomButtonSize || 25;
      this.controlsMargin = options.controlsMargin || 10;
      this.roomEdgeColor = options.roomEdgeColor || "#1c6b60";
      this.visitingColor = options.visitingColor || "#c20000";
      // (roomSize / spacing / viewCells defined above)
      this.connectionColor = options.connectionColor || "#b8893f"; // amber
      this.connectionWidth = options.connectionWidth || 1.6;
      this.glyphColor = options.glyphColor || "#c9b48f";
      this.wrapColor = options.wrapColor || "#3fb0a0";          // toroidal wrap edge-stub color
      this.verticalTickColor = options.verticalTickColor || "#5ad4e6"; // up/down tick (bright, stands out)
      this.serviceColor = options.serviceColor || "#e8b94a";    // bank/shop/trainer marker (gold)
      this.biomeTints = options.biomeTints || RoomGridSVG.DEFAULT_BIOME_TINTS;
      // ── Internal state ────────────────────────────────────────────────
      // rooms: Map<RoomId, { room, group, defaultColor }>
      this.rooms = new Map();
      this.drawnEdges = new Set(); // to avoid dup lines
      // Dedup sets for wrap stubs and vertical ticks (keyed by "id:dx:dy" /
      // "id:u" / "id:d") — mirrors drawnEdges so repeated same-zone snapshots
      // do NOT accumulate duplicate DOM elements (approach (a) from the task spec).
      this.drawnWrapStubs = new Set();
      this.drawnVerticalTicks = new Set();
      this.currentCenterId = null; // for highlight
      this._zone = null; // current zone of the loaded snapshot (Zone.Map)

      // ── Build container & SVG ─────────────────────────────────────────
      this.container = document.querySelector(selector);
      this.container.style.position = 'relative';

      this.svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
      this.svg.setAttribute('preserveAspectRatio', 'xMidYMid meet');
      this.svg.style.width = '100%';
      this.svg.style.height = '100%';
      this.container.appendChild(this.svg);

      // Connections under rooms:
      this.connectionsGroup = document.createElementNS(this.svg.namespaceURI, 'g');
      this.svg.appendChild(this.connectionsGroup);
      // Rooms on top:
      this.roomsGroup = document.createElementNS(this.svg.namespaceURI, 'g');
      this.svg.appendChild(this.roomsGroup);

      // Default tiny viewBox until rooms exist:
      this.svg.setAttribute('viewBox', '0 0 1 1');

      // ── HTML overlay zoom controls ────────────────────────────────────
      this._createHTMLControls();
  }

  // ── Public API ───────────────────────────────────────────────────────

  /**
   * Add or update a room.
   * - Pre-adds any Exits given as {RoomId,x,y,…}
   * - If room already exists, updates its position, color, text, & redraws edges.
   */
  addRoom(room, deferEdges = false) {
      const id = room.RoomId;

      // 1) Pre-add exit-defined rooms — ONLY when the exit carries real coords.
      //    Snapshot exits carry no x/y (the room arrives as its own entry), so
      //    pre-adding them would create a ghost node at undefined coords.
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

      // prepare defaults
      const defaultColor = room.Color || this.tintFor(room.biome);

      // 2) UPDATE existing
      if (this.rooms.has(id)) {
          const entry = this.rooms.get(id);
          // update stored data
          entry.room.x = room.x;
          entry.room.y = room.y;
          entry.room.Exits = Array.isArray(room.Exits) ? room.Exits : [];
          entry.room.ExitsMeta = room.ExitsMeta || [];
          entry.room.Color = room.Color;
          entry.room.Text = room.Text;
          entry.room.tags = room.tags;
          entry.room.name = room.name;
          entry.defaultColor = defaultColor;

          const svc = this._serviceFor(room.tags);

          // move & recolor rect (centered small node)
          const s = this.roomSize;
          const rect = this.svg.querySelector(`rect[data-room-rect="${id}"]`);
          rect.setAttribute('x', room.x * this.spacing - s / 2);
          rect.setAttribute('y', room.y * this.spacing - s / 2);
          if (this.currentCenterId === id) {
              rect.setAttribute('fill', this.visitingColor);
          } else {
              rect.setAttribute('fill', defaultColor);
          }

          // move & update glyph (keep the service marker if this is a service room)
          const txtEl = this.svg.querySelector(`g[data-room-id="${id}"] text`);
          txtEl.setAttribute('x', room.x * this.spacing);
          txtEl.setAttribute('y', room.y * this.spacing + s * 0.25 + 1);
          txtEl.textContent = svc ? svc.glyph : (room.symbol || '');

          // redraw any new edges (skipped during deferred two-pass placement)
          if (!deferEdges) this._drawEdgesForRoom(id);

          // refresh bounds & view
          this._updateBounds();
          this._applyZoom();
          return;
      }

      // 3) NEW room → draw group
      const g = document.createElementNS(this.svg.namespaceURI, 'g');
      g.setAttribute('data-room-id', id);

      // hybrid small centered node
      const s = this.roomSize;
      const cx = room.x * this.spacing;
      const cy = room.y * this.spacing;
      const svc = this._serviceFor(room.tags); // bank/shop/trainer/storage marker (or null)
      const rect = document.createElementNS(this.svg.namespaceURI, 'rect');
      rect.setAttribute('width', s);
      rect.setAttribute('height', s);
      rect.setAttribute('x', cx - s / 2);
      rect.setAttribute('y', cy - s / 2);
      rect.setAttribute('stroke', svc ? this.serviceColor : this.roomEdgeColor);
      rect.setAttribute('stroke-width', svc ? '2' : '1');
      rect.setAttribute('rx', '4');
      rect.setAttribute('ry', '4');
      rect.setAttribute('data-room-rect', id);
      rect.setAttribute('fill', room.Color || this.tintFor(room.biome));
      rect.style.cursor = 'pointer';
      rect.addEventListener('click', () => this.onRoomClick(room));
      g.appendChild(rect);

      // hover tooltip with the room name (restores at-a-glance identify)
      const title = document.createElementNS(this.svg.namespaceURI, 'title');
      title.textContent = room.name || '';
      g.appendChild(title);

      // glyph: gold service marker if any (bold), else the faint biome symbol
      const glyph = document.createElementNS(this.svg.namespaceURI, 'text');
      glyph.setAttribute('x', cx);
      glyph.setAttribute('y', cy + s * 0.25 + 1);
      glyph.setAttribute('text-anchor', 'middle');
      glyph.setAttribute('font-size', s * 0.6);
      glyph.setAttribute('fill', svc ? this.serviceColor : this.glyphColor);
      glyph.setAttribute('opacity', svc ? '1' : '0.85');
      if (svc) glyph.setAttribute('font-weight', 'bold');
      glyph.setAttribute('pointer-events', 'none');
      glyph.textContent = svc ? svc.glyph : (room.symbol || '');
      g.appendChild(glyph);

      this.roomsGroup.appendChild(g);
      this.rooms.set(id, {
          room,
          group: g,
          defaultColor
      });

      // draw edges for this new room (skipped during deferred two-pass placement)
      if (!deferEdges) this._drawEdgesForRoom(id);

      // refresh bounds & view
      this._updateBounds();
      this._applyZoom();
  }

  /**
   * Bulk‐set rooms (wipes existing).
   */
  setRooms(arr) {
      this.reset();
      arr.forEach(r => this.addRoom(r));
  }

  /**
   * Clear everything.
   */
  reset() {
      this.rooms.clear();
      this.drawnEdges.clear();
      this.drawnWrapStubs.clear();
      this.drawnVerticalTicks.clear();
      this.currentCenterId = null;
      this._z = null;
      this.zoomLevel = 1;
      this.svg.setAttribute('viewBox', '0 0 1 1');
      this.roomsGroup.innerHTML = '';
      this.connectionsGroup.innerHTML = '';
  }

  /**
   * Center & highlight a room.  Previous one reverts to its default color.
   */
  centerOnRoom(id) {
      const entry = this.rooms.get(id);
      if (!entry) return;

      // un-highlight previous
      if (this.currentCenterId != null) {
          const prevRect = this.svg.querySelector(
              `rect[data-room-rect="${this.currentCenterId}"]`
          );
          if (prevRect) {
              const prevEntry = this.rooms.get(this.currentCenterId);
              prevRect.setAttribute('fill', prevEntry.defaultColor);
          }
      }

      // compute new view center (rooms are centered on grid coords)
      this.center = {
          x: entry.room.x * this.spacing,
          y: entry.room.y * this.spacing
      };
      this._applyZoom();

      // highlight new
      const newRect = this.svg.querySelector(
          `rect[data-room-rect="${id}"]`
      );
      if (newRect) newRect.setAttribute('fill', this.visitingColor);

      this.currentCenterId = id;
  }

  /** Ingest a Zone.Map snapshot: [{num,x,y,z,symbol,biome,exits:[{to,dx,dy,dz,kind}]}]. */
  setZoneSnapshot(zone, snapshotRooms, currentZ) {
      currentZ = currentZ || 0;
      // The map shows one floor at a time: reset on zone change OR floor (z) change.
      if (this._zone !== zone || this._z !== currentZ) {
          this.reset();
          this._zone = zone;
          this._z = currentZ;
      }
      // Only render rooms on the current floor; up/down exits show as ▲/▼ ticks.
      const floor = snapshotRooms.filter(r => (r.z || 0) === currentZ);
      // Pass 1: place/update every room node first (defer edge drawing) so that
      // when connectors are drawn, every endpoint already has its real coords.
      // Drawing an edge to a not-yet-placed room would anchor it at a stale
      // position and the edge-dedup would never repair it (the "streak" bug).
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
              ExitsMeta: r.exits || [] // raw per-exit {to,dx,dy,dz,kind}; consumed by wrap-stub + vertical-tick rendering
          }, /* deferEdges = */ true);
      });
      // Pass 2: all nodes exist now — draw connectors/stubs/ticks (each dedups,
      // so repeated same-zone resends never duplicate or misplace lines).
      floor.forEach(r => this._drawEdgesForRoom(r.num));
      this._applyZoom();
  }

  /** Zoom out so the whole explored map fits in view. */
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
      // choose the zoom that makes the local window encompass the whole world
      const zW = span / Math.max(this.worldWidth, 1);
      const zH = (span * ar) / Math.max(this.worldHeight, 1);
      this.zoomLevel = Math.min(zW, zH) * 0.92; // small margin around the edges
      this._applyZoom();
  }

  zoomIn() {
      this.zoomLevel *= this.zoomStep;
      this._applyZoom();
  }
  zoomOut() {
      this.zoomLevel /= this.zoomStep;
      this._applyZoom();
  }

  drawConnection(a, b) {
      if (!this.rooms.has(a) || !this.rooms.has(b)) return;
      this._drawEdge(a, b);
      this._applyZoom();
  }

  // ── Private draw helpers ───────────────────────────────────────────────

  _createHTMLControls() {
      const div = document.createElement('div');
      div.style.cssText = `
    position:absolute;
    bottom:${this.controlsMargin}px;
    left:${this.controlsMargin}px;
    display:flex; gap:5px;
    z-index:5;
  `;
      const mk = (lbl, cb) => {
          const b = document.createElement('button');
          b.textContent = lbl;
          b.style.cssText = `
      width:${this.zoomButtonSize}px;
      height:${this.zoomButtonSize}px;
      font-size:${this.zoomButtonSize*0.6}px;
      line-height:1;
    `;
          b.addEventListener('click', cb);
          return b;
      };
      div.append(
          mk('fit',    () => this.fit()),
          mk('ctr',   () => this.centerOnRoom(this.currentCenterId)),
          mk('−',     () => this.zoomOut()),
          mk('+',     () => this.zoomIn())
      );
      this.container.appendChild(div);
  }

  _drawEdgesForRoom(id) {
      const me = this.rooms.get(id).room;

      if (Array.isArray(me.ExitsMeta) && me.ExitsMeta.length) {
          // Kind-routing path: ExitsMeta is set by setZoneSnapshot (Zone.Map).
          // Each entry is {to, dx, dy, dz, kind} where kind ∈ normal|long|wrap|vertical.
          me.ExitsMeta.forEach(e => {
              if (e.kind === 'vertical') { this._drawVerticalTick(id, e.dz); return; }
              if (e.kind === 'wrap')     { this._drawWrapStub(id, e.dx, e.dy); return; }
              // normal + long: full connector line (drawnEdges dedups already)
              if (this.rooms.has(e.to))  { this._drawEdge(id, e.to); }
          });
      } else {
          // Fallback path: rooms added via Room.Info (no ExitsMeta), use Exits array.
          const exits = Array.isArray(me.Exits) ? me.Exits : [];

          // draw its own exits
          exits.forEach(e => {
              const to = (typeof e === 'object') ? e.RoomId : e;
              if (this.rooms.has(to)) this._drawEdge(id, to);
          });

          // draw others' exits back to it
          this.rooms.forEach(({ room }, otherId) => {
              if (otherId === id) return;
              const oe = Array.isArray(room.Exits) ? room.Exits : [];
              if (oe.some(x => ((typeof x === 'object') ? x.RoomId : x) === id)) {
                  this._drawEdge(otherId, id);
              }
          });
      }
  }

  _drawEdge(a, b) {
      const key = a < b ? `${a}-${b}` : `${b}-${a}`;
      if (this.drawnEdges.has(key)) return;
      this.drawnEdges.add(key);

      const ra = this.rooms.get(a).room;
      const rb = this.rooms.get(b).room;
      // rooms are centered on grid coords — no cellSize offset needed
      const x1 = ra.x * this.spacing;
      const y1 = ra.y * this.spacing;
      const x2 = rb.x * this.spacing;
      const y2 = rb.y * this.spacing;

      const line = document.createElementNS(this.svg.namespaceURI, 'line');
      line.setAttribute('x1', x1);
      line.setAttribute('y1', y1);
      line.setAttribute('x2', x2);
      line.setAttribute('y2', y2);
      line.setAttribute('stroke', this.connectionColor);
      line.setAttribute('stroke-width', this.connectionWidth);
      this.connectionsGroup.appendChild(line);
  }

  /**
   * Draw a teal stub + chevron for a wrap exit pointing off room's edge.
   * Dedup: skipped if this room+direction combo was already drawn this zone
   * (mirrors drawnEdges pattern — see drawnWrapStubs in constructor).
   */
  _drawWrapStub(id, dx, dy) {
      const me = this.rooms.get(id); if (!me) return;
      // Dedup: same approach as drawnEdges (approach (a)) — keyed by id:dx:dy
      const key = `${id}:${dx}:${dy}`;
      if (this.drawnWrapStubs.has(key)) return;
      this.drawnWrapStubs.add(key);

      const cx = me.room.x * this.spacing, cy = me.room.y * this.spacing;
      let ux = dx === 0 ? 0 : (dx > 0 ? 1 : -1);
      let uy = dy === 0 ? 0 : (dy > 0 ? 1 : -1);
      const mag = Math.hypot(ux, uy) || 1;
      ux /= mag; uy /= mag;
      const len = this.roomSize * 1.4, start = this.roomSize * 0.55;
      const ex = cx + ux * (start + len), ey = cy + uy * (start + len);
      const WC = this.wrapColor;
      const line = document.createElementNS(this.svg.namespaceURI, 'line');
      line.setAttribute('x1', cx + ux * start); line.setAttribute('y1', cy + uy * start);
      line.setAttribute('x2', ex); line.setAttribute('y2', ey);
      line.setAttribute('stroke', WC); line.setAttribute('stroke-width', this.connectionWidth);
      line.setAttribute('stroke-linecap', 'round');
      this.connectionsGroup.appendChild(line);
      // Chevron arms: perpendicular to the outward direction
      const px = -uy, py = ux, c = this.roomSize * 0.28, b = this.roomSize * 0.34;
      [1, -1].forEach(sgn => {
          const ch = document.createElementNS(this.svg.namespaceURI, 'line');
          ch.setAttribute('x1', ex); ch.setAttribute('y1', ey);
          ch.setAttribute('x2', ex - ux * b + px * c * sgn);
          ch.setAttribute('y2', ey - uy * b + py * c * sgn);
          ch.setAttribute('stroke', WC); ch.setAttribute('stroke-width', this.connectionWidth);
          ch.setAttribute('stroke-linecap', 'round');
          this.connectionsGroup.appendChild(ch);
      });
  }

  /**
   * Draw a faint ▲/▼ tick on the room for vertical exits (up/down).
   * Tick lives in the room's own <g> so it moves/clears with the room.
   * Dedup: skipped if this room+dz-sign combo was already drawn this zone
   * (mirrors drawnEdges pattern — see drawnVerticalTicks in constructor).
   */
  _drawVerticalTick(id, dz) {
      const me = this.rooms.get(id); if (!me) return;
      // Dedup: keyed by id:sign(dz) — one up-tick and one down-tick max per room
      const sign = dz > 0 ? 'u' : 'd';
      const key = `${id}:${sign}`;
      if (this.drawnVerticalTicks.has(key)) return;
      this.drawnVerticalTicks.add(key);

      const s = this.roomSize;
      const cx = me.room.x * this.spacing, cy = me.room.y * this.spacing;
      const t = document.createElementNS(this.svg.namespaceURI, 'text');
      t.setAttribute('x', cx + s * 0.42);
      t.setAttribute('y', cy + (dz > 0 ? -s * 0.30 : s * 0.46));
      t.setAttribute('text-anchor', 'middle');
      t.setAttribute('font-size', s * 0.5);
      t.setAttribute('fill', this.verticalTickColor);
      t.setAttribute('opacity', '0.95');
      t.setAttribute('font-weight', 'bold');
      t.setAttribute('pointer-events', 'none');
      t.textContent = dz > 0 ? '▲' : '▼'; // ▲ up / ▼ down
      me.group.appendChild(t);
  }

  _updateBounds() {
      if (!this.rooms.size) {
          this.bounds = {
              minX: 0,
              maxX: 0,
              minY: 0,
              maxY: 0
          };
      } else {
          const xs = [...this.rooms.values()].map(e => e.room.x);
          const ys = [...this.rooms.values()].map(e => e.room.y);
          this.bounds = {
              minX: Math.min(...xs),
              maxX: Math.max(...xs),
              minY: Math.min(...ys),
              maxY: Math.max(...ys)
          };
      }
      this.worldWidth = (this.bounds.maxX - this.bounds.minX + 1) * this.spacing;
      this.worldHeight = (this.bounds.maxY - this.bounds.minY + 1) * this.spacing;

      if (!this.center && this.rooms.size) {
          this.center = {
              x: this.bounds.minX * this.spacing + this.worldWidth / 2,
              y: this.bounds.minY * this.spacing + this.worldHeight / 2
          };
      }
  }

  _applyZoom() {
      // Stable local window: show ~viewCells cells across, centred on the current
      // room, regardless of how much of the zone has been explored. zoomLevel
      // scales this window; fit() widens it to cover the whole world.
      const baseW = (this.viewCells * this.spacing) / this.zoomLevel;
      const cw = this.container.clientWidth || 1;
      const ch = this.container.clientHeight || 1;
      const ar = (cw > 1 && ch > 1) ? ch / cw : 1;
      const baseH = baseW * ar;
      const cx = this.center ? this.center.x : 0;
      const cy = this.center ? this.center.y : 0;
      this.svg.setAttribute('viewBox', `${cx - baseW / 2} ${cy - baseH / 2} ${baseW} ${baseH}`);
  }
}

// ── Biome tint lookup table ───────────────────────────────────────────────────
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
  emboss: 0.6, fray: 3.4, nickP: 0.08, crackStep: 24, crackJit: 9, vig: 0.5
};

// ── Service-room markers (bank / shop / trainer / storage) ────────────────────
RoomGridSVG.SERVICE_GLYPHS = { bank: '$', shop: 'S', trainer: 'T', storage: '▢' };
RoomGridSVG.SERVICE_ORDER  = ['bank', 'shop', 'trainer', 'storage'];
RoomGridSVG.prototype._serviceFor = function (tags) {
  if (!Array.isArray(tags) || !tags.length) return null;
  for (let i = 0; i < RoomGridSVG.SERVICE_ORDER.length; i++) {
    const k = RoomGridSVG.SERVICE_ORDER[i];
    if (tags.indexOf(k) !== -1) return { key: k, glyph: RoomGridSVG.SERVICE_GLYPHS[k] };
  }
  return null;
};
