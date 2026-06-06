class RoomGridSVG {
  constructor(selector, options = {}) {
      // ── Configurable options & defaults ───────────────────────────────
      // cellSize + cellMargin set the grid pitch (spacing between node centres).
      // Node display size is roomSize; cellSize itself is not the node size.
      this.cellSize = options.cellSize || 100;
      this.cellMargin = options.cellMargin || 20;
      this.spacing = this.cellSize + this.cellMargin;
      this.zoomStep = options.zoomStep || 1.2;
      this.zoomLevel = options.initialZoom || 1;
      this.onRoomClick = options.onRoomClick || (() => {});
      this.zoomButtonSize = options.zoomButtonSize || 25;
      this.controlsMargin = options.controlsMargin || 10;
      this.roomEdgeColor = options.roomEdgeColor || "#1c6b60";
      this.visitingColor = options.visitingColor || "#c20000";
      this.roomSize = options.roomSize || 16;              // hybrid: small nodes
      this.connectionColor = options.connectionColor || "#b8893f"; // amber
      this.connectionWidth = options.connectionWidth || 1.6;
      this.glyphColor = options.glyphColor || "#c9b48f";
      this.biomeTints = options.biomeTints || RoomGridSVG.DEFAULT_BIOME_TINTS;
      // ── Internal state ────────────────────────────────────────────────
      // rooms: Map<RoomId, { room, group, defaultColor }>
      this.rooms = new Map();
      this.drawnEdges = new Set(); // to avoid dup lines
      this.currentCenterId = null; // for highlight

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
  addRoom(room) {
      const id = room.RoomId;

      // 1) Pre-add exit-defined rooms
      if (Array.isArray(room.Exits)) {
          room.Exits.forEach(e => {
              if (e && typeof e === 'object' && e.RoomId != null) {

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

          // redraw any new edges
          this._drawEdgesForRoom(id);

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
      glyph.setAttribute('font-size', s * 0.5);
      glyph.setAttribute('fill', this.glyphColor);
      glyph.setAttribute('opacity', '0.75');
      glyph.setAttribute('pointer-events', 'none');
      glyph.textContent = room.symbol || '';
      g.appendChild(glyph);

      this.roomsGroup.appendChild(g);
      this.rooms.set(id, {
          room,
          group: g,
          defaultColor
      });

      // draw edges for this new room
      this._drawEdgesForRoom(id);

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
    top:${this.controlsMargin}px;
    right:${this.controlsMargin}px;
    display:flex; gap:5px;
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
      div.append(mk('−', () => this.zoomOut()), mk('+', () => this.zoomIn()));
      this.container.appendChild(div);
  }

  _drawEdgesForRoom(id) {
      const me = this.rooms.get(id)
          .room;
      const exits = Array.isArray(me.Exits) ? me.Exits : [];

      // draw its own exits
      exits.forEach(e => {
          const to = (typeof e === 'object') ? e.RoomId : e;
          if (this.rooms.has(to)) this._drawEdge(id, to);
      });

      // draw others’ exits back to it
      this.rooms.forEach(({
          room
      }, otherId) => {
          if (otherId === id) return;
          const oe = Array.isArray(room.Exits) ? room.Exits : [];
          if (oe.some(x => ((typeof x === 'object') ? x.RoomId : x) === id)) {
              this._drawEdge(otherId, id);
          }
      });
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
      const hw = this.worldWidth / (2 * this.zoomLevel);
      const hh = this.worldHeight / (2 * this.zoomLevel);
      const x0 = (this.center ? this.center.x : this.worldWidth / 2) - hw;
      const y0 = (this.center ? this.center.y : this.worldHeight / 2) - hh;
      this.svg.setAttribute('viewBox', `${x0} ${y0} ${hw*2} ${hh*2}`);
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
