"use strict";
/*
 * builder.js — the editor-grade SVG canvas for the admin World Builder (/build).
 *
 * It consumes the unfogged Zone.Map snapshot pushed over Build.* GMCP and draws
 * one (plane, floor) at a time: biome-tinted room nodes on an explicit grid,
 * styled exit lines, and dashed "+" ghost cells in each open compass direction.
 * Clicking a room selects it (fires Builder.onRoomSelected for the inspector);
 * clicking a ghost cell sends Build.Room.Create. Pan by dragging; zoom by wheel.
 *
 * Authored coords use the engine frame (north = y-1), which maps directly to
 * SVG's y-down axis, so no vertical flip is needed.
 */
(function () {
  var SVGNS = "http://www.w3.org/2000/svg";
  var CELL = 96;   // grid pitch in world units
  var RW = 62, RH = 46; // room node size

  var BIOME_TINTS = {
    "city": "#3a342c", "town": "#3a342c",
    "forest": "#25382a", "swamp": "#243226", "marsh": "#243226",
    "water": "#243246", "lake": "#243246", "river": "#243246",
    "hills": "#3e3422", "mountain": "#3e3422",
    "cave": "#2c2530", "dungeon": "#2c2530",
    "desert": "#3e3622", "road": "#3a3226",
    "_default": "#2a2018"
  };
  function biomeFill(b) {
    if (!b) return BIOME_TINTS._default;
    return BIOME_TINTS[String(b).toLowerCase()] || BIOME_TINTS._default;
  }

  // Compass deltas in the engine frame (north = y-1). Mirrors internal/mapper.
  var DIRS = {
    north: [0, -1], south: [0, 1], west: [-1, 0], east: [1, 0],
    northeast: [1, -1], northwest: [-1, -1],
    southeast: [1, 1], southwest: [-1, 1]
  };

  function el(name, attrs) {
    var e = document.createElementNS(SVGNS, name);
    if (attrs) for (var k in attrs) e.setAttribute(k, attrs[k]);
    return e;
  }

  function BuilderCanvas(host) {
    this.host = host;
    this.rooms = [];            // all rooms from the snapshot
    this.byId = {};             // roomId -> room
    this.zone = "";
    this.plane = null;
    this.floor = 0;             // z-level within the plane
    this.selectedId = 0;
    this.tx = 40; this.ty = 40; this.scale = 1; // pan/zoom of the world group

    this.svg = el("svg", { width: "100%", height: "100%" });
    this.svg.style.display = "block";
    this.svg.style.touchAction = "none";
    this.world = el("g", {});
    this.svg.appendChild(this.world);
    host.innerHTML = "";
    host.appendChild(this.svg);

    this._wirePanZoom();
  }

  BuilderCanvas.prototype.planesPresent = function () {
    var set = {};
    for (var i = 0; i < this.rooms.length; i++) set[this.rooms[i].plane || 0] = true;
    return Object.keys(set).map(Number).sort(function (a, b) { return a - b; });
  };

  BuilderCanvas.prototype.floorsForPlane = function (plane) {
    var set = {};
    for (var i = 0; i < this.rooms.length; i++) {
      var r = this.rooms[i];
      if ((r.plane || 0) === plane) set[r.z || 0] = true;
    }
    return Object.keys(set).map(Number).sort(function (a, b) { return a - b; });
  };

  BuilderCanvas.prototype.setSnapshot = function (payload) {
    this.zone = payload.zone || "";
    this.rooms = (payload.rooms || []).slice();
    this.byId = {};
    for (var i = 0; i < this.rooms.length; i++) this.byId[this.rooms[i].num] = this.rooms[i];

    var planes = this.planesPresent();
    if (this.plane === null || planes.indexOf(this.plane) === -1) {
      this.plane = planes.length ? planes[0] : 0;
    }
    var floors = this.floorsForPlane(this.plane);
    if (floors.indexOf(this.floor) === -1) this.floor = floors.length ? floors[0] : 0;

    if (typeof this.onPlanes === "function") this.onPlanes(planes, this.plane, floors, this.floor);
    // Keep the selected room highlighted if it survived a refresh.
    if (this.selectedId && !this.byId[this.selectedId]) this.selectedId = 0;
    this.render();
  };

  BuilderCanvas.prototype.setPlane = function (plane) {
    this.plane = plane;
    var floors = this.floorsForPlane(plane);
    if (floors.indexOf(this.floor) === -1) this.floor = floors.length ? floors[0] : 0;
    if (typeof this.onPlanes === "function") this.onPlanes(this.planesPresent(), this.plane, floors, this.floor);
    this.render();
  };

  BuilderCanvas.prototype.setFloor = function (z) { this.floor = z; this.render(); };

  BuilderCanvas.prototype.visibleRooms = function () {
    var out = [], p = this.plane, z = this.floor;
    for (var i = 0; i < this.rooms.length; i++) {
      var r = this.rooms[i];
      if ((r.plane || 0) === p && (r.z || 0) === z) out.push(r);
    }
    return out;
  };

  // Occupancy over the whole plane (all z) so ghost cells never overlap a room
  // stacked on another floor at the same x,y.
  BuilderCanvas.prototype.cellOccupied = function (x, y) {
    for (var i = 0; i < this.rooms.length; i++) {
      var r = this.rooms[i];
      if ((r.plane || 0) === this.plane && r.x === x && r.y === y && (r.z || 0) === this.floor) return true;
    }
    return false;
  };

  function cx(r) { return r.x * CELL + RW / 2; }
  function cy(r) { return r.y * CELL + RH / 2; }

  BuilderCanvas.prototype.render = function () {
    var self = this;
    while (this.world.firstChild) this.world.removeChild(this.world.firstChild);
    this.world.setAttribute("transform", "translate(" + this.tx + "," + this.ty + ") scale(" + this.scale + ")");

    var vis = this.visibleRooms();

    // ---- exits (drawn under nodes) ----
    for (var i = 0; i < vis.length; i++) {
      var r = vis[i];
      var exits = r.exits || [];
      for (var j = 0; j < exits.length; j++) this._drawExit(r, exits[j]);
    }

    // ---- ghost cells ----
    for (i = 0; i < vis.length; i++) this._drawGhosts(vis[i]);

    // ---- room nodes ----
    for (i = 0; i < vis.length; i++) this._drawRoom(vis[i]);

    // (re)apply selection ring on top
    if (this.selectedId && this.byId[this.selectedId]) {
      var sr = this.byId[this.selectedId];
      if ((sr.plane || 0) === this.plane && (sr.z || 0) === this.floor) {
        this.world.appendChild(el("rect", {
          x: sr.x * CELL - 4, y: sr.y * CELL - 4, width: RW + 8, height: RH + 8, rx: 9,
          fill: "none", stroke: "#c9a24b", "stroke-width": 3, opacity: 0.95
        }));
      }
    }
    void self;
  };

  BuilderCanvas.prototype._drawExit = function (r, ex) {
    var dz = ex.dz || 0;
    // Vertical exit: draw a ▲/▼ tick on the room instead of a connector.
    if (dz !== 0 && (ex.dx || 0) === 0 && (ex.dy || 0) === 0) {
      this.world.appendChild(el("text", {
        x: cx(r) + (dz > 0 ? -8 : 8), y: r.y * CELL + 12,
        "text-anchor": "middle", "font-size": 11, fill: "#c9a24b"
      })).textContent = dz > 0 ? "▲" : "▼";
      return;
    }
    var dst = this.byId[ex.to];
    var x1 = cx(r), y1 = cy(r), x2, y2;
    if (dst && (dst.plane || 0) === this.plane && (dst.z || 0) === this.floor) {
      x2 = cx(dst); y2 = cy(dst);
    } else {
      // Off-plane / unknown target (portal, cross-zone, or other floor): stub.
      var dx = ex.dx || 0, dy = ex.dy || 0;
      if (dx === 0 && dy === 0) { dx = 0; dy = -1; } // portal with no delta -> nub upward
      var len = Math.sqrt(dx * dx + dy * dy) || 1;
      x2 = x1 + (dx / len) * (CELL * 0.42);
      y2 = y1 + (dy / len) * (CELL * 0.42);
    }
    var attrs = {
      x1: x1, y1: y1, x2: x2, y2: y2,
      stroke: ex.secret ? "#8a6d3b" : "#b98f3a",
      "stroke-width": 2, "stroke-linecap": "round", opacity: 0.85
    };
    if (ex.secret) attrs["stroke-dasharray"] = "4 4";
    if (ex.stub || !dst) { attrs["stroke-dasharray"] = "2 5"; attrs.opacity = 0.6; }
    this.world.appendChild(el("line", attrs));
    if (ex.oneway) {
      // small arrowhead toward the destination
      var mx = (x1 + x2) / 2, my = (y1 + y2) / 2;
      this.world.appendChild(el("circle", { cx: mx, cy: my, r: 2.4, fill: "#b98f3a" }));
    }
    if (ex.locked) {
      this.world.appendChild(el("circle", {
        cx: (x1 + x2) / 2, cy: (y1 + y2) / 2, r: 3.4, fill: "#2a1d12", stroke: "#c9a24b", "stroke-width": 1
      }));
    }
  };

  BuilderCanvas.prototype._drawGhosts = function (r) {
    var self = this;
    var have = {};
    var exits = r.exits || [];
    for (var j = 0; j < exits.length; j++) {
      var e = exits[j];
      // Mark the compass slot this exit occupies (by its delta).
      for (var d in DIRS) {
        if (DIRS[d][0] === (e.dx || 0) && DIRS[d][1] === (e.dy || 0) && (e.dz || 0) === 0) have[d] = true;
      }
    }
    for (var dir in DIRS) {
      if (have[dir]) continue;
      var nx = r.x + DIRS[dir][0], ny = r.y + DIRS[dir][1];
      if (this.cellOccupied(nx, ny)) continue;
      (function (dir, nx, ny) {
        var gx = nx * CELL + RW / 2, gy = ny * CELL + RH / 2;
        var g = el("g", { style: "cursor:pointer" });
        g.appendChild(el("rect", {
          x: nx * CELL, y: ny * CELL, width: RW, height: RH, rx: 8,
          fill: "none", stroke: "#6f5a34", "stroke-width": 1.4, "stroke-dasharray": "5 4", opacity: 0.7
        }));
        var plus = el("text", {
          x: gx, y: gy + 6, "text-anchor": "middle", "font-size": 20, fill: "#8a7038", opacity: 0.85
        });
        plus.textContent = "+";
        g.appendChild(plus);
        g.addEventListener("click", function (ev) {
          ev.stopPropagation();
          self._createRoom(r, dir, nx, ny);
        });
        self.world.appendChild(g);
      })(dir, nx, ny);
    }
  };

  BuilderCanvas.prototype._createRoom = function (from, dir, nx, ny) {
    if (!window.Builder || !window.Builder.sendGMCP) return;
    this.pendingSelectDir = { from: from.num }; // auto-select the created room on refresh
    window.Builder.sendGMCP("Build.Room.Create", {
      fromRoomId: from.num, dir: dir,
      plane: this.plane, x: nx, y: ny, z: this.floor
    });
  };

  BuilderCanvas.prototype._drawRoom = function (r) {
    var self = this;
    var g = el("g", { style: "cursor:pointer" });
    var rect = el("rect", {
      x: r.x * CELL, y: r.y * CELL, width: RW, height: RH, rx: 8,
      fill: biomeFill(r.biome), stroke: "#0e0703", "stroke-width": 1.4
    });
    g.appendChild(rect);
    // subtle top highlight
    g.appendChild(el("rect", {
      x: r.x * CELL + 2, y: r.y * CELL + 2, width: RW - 4, height: RH - 4, rx: 6,
      fill: "none", stroke: "#5c4326", "stroke-width": 0.6, opacity: 0.6
    }));
    var idT = el("text", {
      x: cx(r), y: r.y * CELL + 16, "text-anchor": "middle",
      "font-family": "monospace", "font-size": 10, fill: "#c9a24b"
    });
    idT.textContent = r.num;
    g.appendChild(idT);
    if (r.name) {
      var nm = el("text", {
        x: cx(r), y: r.y * CELL + 31, "text-anchor": "middle",
        "font-family": "Georgia,serif", "font-size": 8, fill: "#e7d9b8"
      });
      nm.textContent = r.name.length > 12 ? r.name.slice(0, 11) + "…" : r.name;
      g.appendChild(nm);
      var title = el("title"); title.textContent = r.name + " (#" + r.num + ")"; g.appendChild(title);
    }
    g.addEventListener("click", function (ev) {
      ev.stopPropagation();
      self.select(r.num);
    });
    this.world.appendChild(g);
  };

  BuilderCanvas.prototype.select = function (roomId) {
    this.selectedId = roomId;
    this.render();
    if (typeof window.Builder.onRoomSelected === "function") {
      window.Builder.onRoomSelected(this.byId[roomId] || null);
    }
  };

  // After a refresh following a ghost-create, auto-select the new room (the one
  // now occupying the pending cell / the highest new id neighbouring `from`).
  BuilderCanvas.prototype.afterRefresh = function (createdRoomId) {
    if (createdRoomId && this.byId[createdRoomId]) {
      var nr = this.byId[createdRoomId];
      this.plane = nr.plane || 0; this.floor = nr.z || 0;
      this.select(createdRoomId);
    }
  };

  BuilderCanvas.prototype._wirePanZoom = function () {
    var self = this, dragging = false, lx = 0, ly = 0;
    this.svg.addEventListener("pointerdown", function (e) {
      dragging = true; lx = e.clientX; ly = e.clientY;
      self.svg.setPointerCapture(e.pointerId);
    });
    this.svg.addEventListener("pointermove", function (e) {
      if (!dragging) return;
      self.tx += (e.clientX - lx); self.ty += (e.clientY - ly);
      lx = e.clientX; ly = e.clientY;
      self.world.setAttribute("transform", "translate(" + self.tx + "," + self.ty + ") scale(" + self.scale + ")");
    });
    function stop(e) { dragging = false; try { self.svg.releasePointerCapture(e.pointerId); } catch (x) {} }
    this.svg.addEventListener("pointerup", stop);
    this.svg.addEventListener("pointercancel", stop);
    this.svg.addEventListener("wheel", function (e) {
      e.preventDefault();
      var rect = self.svg.getBoundingClientRect();
      var mx = e.clientX - rect.left, my = e.clientY - rect.top;
      var factor = e.deltaY < 0 ? 1.1 : 1 / 1.1;
      var ns = Math.max(0.25, Math.min(3, self.scale * factor));
      // zoom toward the cursor
      self.tx = mx - (mx - self.tx) * (ns / self.scale);
      self.ty = my - (my - self.ty) * (ns / self.scale);
      self.scale = ns;
      self.world.setAttribute("transform", "translate(" + self.tx + "," + self.ty + ") scale(" + self.scale + ")");
    }, { passive: false });
    // click on empty canvas clears selection
    this.svg.addEventListener("click", function () {
      if (self.selectedId) { self.selectedId = 0; self.render(); if (typeof window.Builder.onRoomSelected === "function") window.Builder.onRoomSelected(null); }
    });
  };

  window.Builder = window.Builder || {};
  window.Builder.BuilderCanvas = BuilderCanvas;
})();
