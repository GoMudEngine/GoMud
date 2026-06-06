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
      this.verticalTickColor = options.verticalTickColor || "#8a6a3a"; // up/down tick color
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
          entry.defaultColor = defaultColor;

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

          // move & update label
          const txtEl = this.svg.querySelector(`g[data-room-id="${id}"] text`);
          txtEl.setAttribute('x', room.x * this.spacing);
          txtEl.setAttribute('y', room.y * this.spacing + s * 0.25 + 1);
          txtEl.textContent = room.symbol || '';

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
      const rect = document.createElementNS(this.svg.namespaceURI, 'rect');
      rect.setAttribute('width', s);
      rect.setAttribute('height', s);
      rect.setAttribute('x', cx - s / 2);
      rect.setAttribute('y', cy - s / 2);
      rect.setAttribute('stroke', this.roomEdgeColor);
      rect.setAttribute('stroke-width', '1');
      rect.setAttribute('rx', '4');
      rect.setAttribute('ry', '4');
      rect.setAttribute('data-room-rect', id);
      rect.setAttribute('fill', room.Color || this.tintFor(room.biome));
      rect.style.cursor = 'pointer';
      rect.addEventListener('click', () => this.onRoomClick(room));
      g.appendChild(rect);

      // faint glyph (room.symbol) centered in the node
      const glyph = document.createElementNS(this.svg.namespaceURI, 'text');
      glyph.setAttribute('x', cx);
      glyph.setAttribute('y', cy + s * 0.25 + 1);
      glyph.setAttribute('text-anchor', 'middle');
      glyph.setAttribute('font-size', s * 0.6);
      glyph.setAttribute('fill', this.glyphColor);
      glyph.setAttribute('opacity', '0.85');
      glyph.setAttribute('pointer-events', 'none');
      glyph.textContent = room.symbol || '';
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
  setZoneSnapshot(zone, snapshotRooms) {
      if (this._zone !== zone) {
          this.reset();
          this._zone = zone;
      }
      // Pass 1: place/update every room node first (defer edge drawing) so that
      // when connectors are drawn, every endpoint already has its real coords.
      // Drawing an edge to a not-yet-placed room would anchor it at a stale
      // position and the edge-dedup would never repair it (the "streak" bug).
      snapshotRooms.forEach(r => {
          this.addRoom({
              RoomId: r.num,
              x: r.x,
              y: r.y,
              z: r.z,
              symbol: r.symbol,
              biome: r.biome,
              Exits: (r.exits || []).map(e => ({ RoomId: e.to, kind: e.kind, dx: e.dx, dy: e.dy, dz: e.dz })),
              ExitsMeta: r.exits || [] // raw per-exit {to,dx,dy,dz,kind}; consumed by wrap-stub + vertical-tick rendering
          }, /* deferEdges = */ true);
      });
      // Pass 2: all nodes exist now — draw connectors/stubs/ticks (each dedups,
      // so repeated same-zone resends never duplicate or misplace lines).
      snapshotRooms.forEach(r => this._drawEdgesForRoom(r.num));
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
      t.setAttribute('font-size', s * 0.32);
      t.setAttribute('fill', this.verticalTickColor);
      t.setAttribute('opacity', '0.6');
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
