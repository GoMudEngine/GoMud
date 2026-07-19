/* dashboard.js — Dashboard controller + responsive column-stack reflow
 * Three breakpoints:
 *   wide  >= 1100px : default three-column layout (no changes)
 *   rail  700–1100px: left+right cols hidden; side panels in a tabbed rail
 *   phone < 700px   : center fills; fixed bottom tab-bar; slide-up drawer
 */
"use strict";

(function () {

  var SIDE_PANELS = ["map", "quests", "vitals", "art", "chat", "status", "trig"];

  // Icons for the phone tab-bar (order matches SIDE_PANELS)
  var PANEL_ICONS = {
    map:    "🗺",
    quests: "❖",
    vitals: "❤",
    art:    "🖼",
    chat:   "💬",
    status: "⚔",
    trig:   "⏱",
  };

  var Dashboard = {
    mode: null,         // current mode: "wide" | "rail" | "phone"
    activeSide: "map",  // last-active side panel name

    // DOM refs created once
    _rail: null,
    _railTabs: null,
    _railBody: null,
    _tabbar: null,
    _drawer: null,
    _drawerBody: null,
    _drawerHead: null,

    // ---------------------------------------------------------------
    // Public entry point
    // ---------------------------------------------------------------
    init: function () {
      // Record each side panel's home column so we can restore later.
      // Order within each col is preserved by the original DOM order.
      SIDE_PANELS.forEach(function (name) {
        var panel = document.getElementById("panel-" + name);
        if (!panel) return;
        var col = panel.closest(".dash-col");
        if (col) panel.dataset.homeCol = col.id;
      });

      // Build rail (inside #dashboard, in the grid area "rail")
      this._buildRail();

      // Build tab-bar and drawer (OUTSIDE #dashboard, appended to body
      // parent so position:fixed is not clipped by the grid container)
      this._buildPhone();

      // Apply initial mode
      this._apply();

      // Window resize listener
      var self = this;
      window.addEventListener("resize", function () {
        self._apply();
      });

      // Optional hooks for later tasks (splitters, rearrange, popout, layout restore)
      this.initSplitters  && this.initSplitters();
      this.initRearrange  && this.initRearrange();
      this.initPopout     && this.initPopout();
      this.restoreLayout  && this.restoreLayout();
    },

    // ---------------------------------------------------------------
    // Reset layout
    // ---------------------------------------------------------------
    // Reset the dashboard to its default arrangement IN PLACE — no page reload,
    // so the websocket (and the player's session) stays connected. Falls back to
    // the old hard reload only if something unexpected throws.
    resetLayout: function () {
      var self = this;
      try {
        // 1. Dock any popped-out panels (closing the WinBox re-docks via onclose).
        if (this._popped) {
          Object.keys(this._popped).forEach(function (name) {
            var wb = self._popped[name];
            if (wb && typeof wb.close === "function") wb.close();
            else if (self.redock) self.redock(name);
          });
        }
        // 2. Un-collapse every panel.
        document.querySelectorAll(".dash-panel.collapsed").forEach(function (p) {
          p.classList.remove("collapsed");
        });
        // 3. Re-home panels in default order (clear the saved arrangement first
        //    so _enterWide uses the default SIDE_PANELS order, not the custom one).
        this._savedArrange = null;
        document.querySelectorAll(".dash-panel").forEach(function (p) { p.style.display = ""; });
        if (this.mode === "rail" && this._enterRail) this._enterRail();
        else if (this.mode === "phone" && this._enterPhone) this._enterPhone();
        else if (this._enterWide) this._enterWide();
        // 4. Forget the persisted layout so a later reload also starts clean.
        try { localStorage.removeItem("dogmud.dashboard.layout.v1"); } catch (e) {}
        // 5. Confirm — no disconnect this time.
        this._toast("Layout reset to default.");
      } catch (e) {
        // Anything unexpected: fall back so the button always does *something*.
        try { localStorage.removeItem("dogmud.dashboard.layout.v1"); } catch (e2) {}
        location.reload();
      }
    },

    // Brief, self-dismissing notification (bottom-center, leather-styled).
    _toast: function (msg) {
      var t = document.createElement("div");
      t.className = "dash-toast";
      t.textContent = msg;
      document.body.appendChild(t);
      requestAnimationFrame(function () { t.classList.add("show"); });
      setTimeout(function () {
        t.classList.remove("show");
        setTimeout(function () { if (t.parentNode) t.parentNode.removeChild(t); }, 400);
      }, 2600);
    },

    // ---------------------------------------------------------------
    // Mode application
    // ---------------------------------------------------------------
    _computeMode: function () {
      var w = document.getElementById("dashboard").clientWidth;
      if (w >= 1100) return "wide";
      if (w >= 700)  return "rail";
      return "phone";
    },

    _apply: function () {
      var newMode = this._computeMode();
      if (newMode === this.mode) return; // guard: no-op when mode unchanged

      var prev = this.mode;
      this.mode = newMode;

      // Tear down previous mode before setting up new one
      if (prev === "rail")  this._exitRail();
      if (prev === "phone") this._exitPhone();

      document.getElementById("dashboard").dataset.mode = newMode;

      if (newMode === "wide")  this._enterWide();
      if (newMode === "rail")  this._enterRail();
      if (newMode === "phone") this._enterPhone();
    },

    // ---------------------------------------------------------------
    // Wide mode
    // ---------------------------------------------------------------
    _enterWide: function () {
      // Re-home side panels honoring the saved arrangement order when available
      // (prevents a wide→rail→wide round-trip from resetting a custom layout).
      // Falls back to the default SIDE_PANELS order when no saved arrange exists.
      var arrange = this._savedArrange;
      if (arrange && typeof arrange === "object") {
        // Replay saved column order: for each column append panels in saved sequence.
        // Any panel not listed in saved arrange falls through to the default below.
        var placed = {};
        ["dash-col-left", "dash-col-center", "dash-col-right"].forEach(function (colId) {
          var col = document.getElementById(colId);
          if (!col) return;
          var names = arrange[colId];
          if (!Array.isArray(names)) return;
          names.forEach(function (name) {
            var panel = document.getElementById("panel-" + name);
            if (!panel) return;
            if (panel.dataset.popped === "1") return;
            col.appendChild(panel);
            panel.dataset.homeCol = colId;
            panel.style.display = "";
            placed[name] = true;
          });
        });
        // Catch any panels not covered by the saved arrange (new panels added after save)
        SIDE_PANELS.forEach(function (name) {
          if (placed[name]) return;
          var panel = document.getElementById("panel-" + name);
          if (!panel) return;
          if (panel.dataset.popped === "1") return;
          var homeId = panel.dataset.homeCol;
          if (!homeId) return;
          var col = document.getElementById(homeId);
          if (col) col.appendChild(panel);
          panel.style.display = "";
        });
      } else {
        // No saved arrangement — use default SIDE_PANELS order
        // (map/vitals/art → left, chat/status/trig → right)
        SIDE_PANELS.forEach(function (name) {
          var panel = document.getElementById("panel-" + name);
          if (!panel) return;
          if (panel.dataset.popped === "1") return; // skip popped-out panels
          var homeId = panel.dataset.homeCol;
          if (!homeId) return;
          var col = document.getElementById(homeId);
          if (col) col.appendChild(panel);
          panel.style.display = "";
        });
      }

      // Hide rail, tabbar, drawer
      if (this._rail)   this._rail.style.display   = "none";
      if (this._tabbar) this._tabbar.style.display  = "none";
      if (this._drawer) {
        this._drawer.classList.remove("open");
        this._drawer.style.display = "none";
      }
    },

    // ---------------------------------------------------------------
    // Rail mode
    // ---------------------------------------------------------------
    _buildRail: function () {
      var rail = document.createElement("div");
      rail.id = "dash-rail";
      rail.style.display = "none";

      var tabs = document.createElement("div");
      tabs.className = "rail-tabs";
      rail.appendChild(tabs);

      var body = document.createElement("div");
      body.className = "rail-body";
      rail.appendChild(body);

      document.getElementById("dashboard").appendChild(rail);

      this._rail     = rail;
      this._railTabs = tabs;
      this._railBody = body;
    },

    _enterRail: function () {
      var self = this;

      // Show rail, hide phone chrome
      this._rail.style.display = "";
      if (this._tabbar) this._tabbar.style.display = "none";
      if (this._drawer) {
        this._drawer.classList.remove("open");
        this._drawer.style.display = "none";
      }

      // Move all side panels into rail body
      SIDE_PANELS.forEach(function (name) {
        var panel = document.getElementById("panel-" + name);
        if (!panel) return;
        if (panel.dataset.popped === "1") return; // skip popped-out panels
        // Return from drawer if it was there
        if (self._drawerBody && self._drawerBody.contains(panel)) {
          self._drawerBody.removeChild(panel);
        }
        self._railBody.appendChild(panel);
        panel.style.display = "none"; // will show only the active tab's panel
      });

      // Build tab strip
      this._railTabs.innerHTML = "";
      SIDE_PANELS.forEach(function (name) {
        var btn = document.createElement("button");
        btn.className = "brass rail-tab";
        btn.dataset.panel = name;
        // Label: capitalize first letter of panel name
        btn.textContent = name.charAt(0).toUpperCase() + name.slice(1);
        btn.addEventListener("click", function () {
          self._setRailActive(name);
        });
        self._railTabs.appendChild(btn);
      });

      // Restore last-active or default to "map"
      this._setRailActive(this.activeSide);
    },

    _setRailActive: function (name) {
      this.activeSide = name;

      // Update tab buttons
      var buttons = this._railTabs.querySelectorAll(".rail-tab");
      buttons.forEach(function (btn) {
        btn.classList.toggle("rail-active", btn.dataset.panel === name);
      });

      // Show only the active panel
      SIDE_PANELS.forEach(function (pName) {
        var panel = document.getElementById("panel-" + pName);
        if (!panel) return;
        if (panel.dataset.popped === "1") return; // skip popped-out panels
        panel.style.display = (pName === name) ? "" : "none";
      });
    },

    _exitRail: function () {
      // Move panels back to home cols (wide will re-place them)
      var self = this;
      SIDE_PANELS.forEach(function (name) {
        var panel = document.getElementById("panel-" + name);
        if (!panel) return;
        if (panel.dataset.popped === "1") return; // skip popped-out panels
        if (self._railBody && self._railBody.contains(panel)) {
          self._railBody.removeChild(panel);
          var homeId = panel.dataset.homeCol;
          if (homeId) {
            var col = document.getElementById(homeId);
            if (col) col.appendChild(panel);
          }
        }
        panel.style.display = "";
      });

      this._railTabs.innerHTML = "";
      this._rail.style.display = "none";
    },

    // ---------------------------------------------------------------
    // Phone mode
    // ---------------------------------------------------------------
    _buildPhone: function () {
      var self = this;
      var parent = document.getElementById("dashboard").parentNode;

      // Tab-bar
      var tabbar = document.createElement("div");
      tabbar.id = "dash-tabbar";
      tabbar.style.display = "none";

      SIDE_PANELS.forEach(function (name) {
        var btn = document.createElement("div");
        btn.className = "tabbar-btn";
        btn.dataset.panel = name;
        btn.title = name.charAt(0).toUpperCase() + name.slice(1);
        btn.textContent = PANEL_ICONS[name] || name.charAt(0).toUpperCase();
        btn.addEventListener("click", function () {
          // Toggle: if drawer is open for same panel, close it
          if (self._drawer.classList.contains("open") &&
              self._drawerActive === name) {
            self._closeDrawer();
          } else {
            self._openDrawer(name);
          }
        });
        tabbar.appendChild(btn);
      });

      parent.appendChild(tabbar);
      this._tabbar = tabbar;

      // Drawer
      var drawer = document.createElement("div");
      drawer.id = "dash-drawer";

      var drawerHead = document.createElement("div");
      drawerHead.className = "drawer-head";
      var closeBtn = document.createElement("button");
      closeBtn.className = "brass";
      closeBtn.textContent = "✕ Close";
      closeBtn.addEventListener("click", function () {
        self._closeDrawer();
      });
      drawerHead.appendChild(closeBtn);
      drawer.appendChild(drawerHead);

      var drawerBody = document.createElement("div");
      drawerBody.className = "drawer-body";
      drawer.appendChild(drawerBody);

      parent.appendChild(drawer);
      this._drawer     = drawer;
      this._drawerHead = drawerHead;
      this._drawerBody = drawerBody;
      this._drawerActive = null;
    },

    _enterPhone: function () {
      // Show tab-bar; hide rail
      if (this._tabbar) this._tabbar.style.display = "";
      if (this._rail)   this._rail.style.display   = "none";

      // Ensure all side panels are in their home cols (hidden via CSS)
      var self = this;
      SIDE_PANELS.forEach(function (name) {
        var panel = document.getElementById("panel-" + name);
        if (!panel) return;
        if (panel.dataset.popped === "1") return; // skip popped-out panels
        // If it's in rail body, move it home
        if (self._railBody && self._railBody.contains(panel)) {
          self._railBody.removeChild(panel);
        }
        // If it's in drawer, move it home
        if (self._drawerBody && self._drawerBody.contains(panel)) {
          self._drawerBody.removeChild(panel);
        }
        var homeId = panel.dataset.homeCol;
        if (homeId) {
          var col = document.getElementById(homeId);
          if (col && !col.contains(panel)) col.appendChild(panel);
        }
        panel.style.display = "";
      });

      // Close drawer if open
      if (this._drawer) {
        this._drawer.classList.remove("open");
        this._drawerActive = null;
      }

      this._updateTabbarActive(null);
    },

    _exitPhone: function () {
      // Return any drawer panel to its home col
      this._closeDrawer();

      if (this._tabbar) this._tabbar.style.display = "none";
      if (this._drawer) this._drawer.style.display  = "none";
    },

    _openDrawer: function (name) {
      // Return current drawer panel to its home col first
      if (this._drawerActive && this._drawerActive !== name) {
        var prev = document.getElementById("panel-" + this._drawerActive);
        if (prev && this._drawerBody.contains(prev)) {
          this._drawerBody.removeChild(prev);
          var prevHomeId = prev.dataset.homeCol;
          if (prevHomeId) {
            var prevCol = document.getElementById(prevHomeId);
            if (prevCol) prevCol.appendChild(prev);
          }
        }
      }

      var panel = document.getElementById("panel-" + name);
      if (!panel) return;

      // Move panel into drawer body
      this._drawerBody.appendChild(panel);
      panel.style.display = "";

      this._drawerActive = name;
      this._drawer.style.display = "";
      // Force reflow so CSS transition fires
      this._drawer.getBoundingClientRect();
      this._drawer.classList.add("open");

      this._updateTabbarActive(name);
    },

    _closeDrawer: function () {
      if (!this._drawerActive) return;

      var panel = document.getElementById("panel-" + this._drawerActive);
      if (panel && this._drawerBody && this._drawerBody.contains(panel)) {
        this._drawerBody.removeChild(panel);
        var homeId = panel.dataset.homeCol;
        if (homeId) {
          var col = document.getElementById(homeId);
          if (col) col.appendChild(panel);
        }
        panel.style.display = "";
      }

      this._drawerActive = null;
      if (this._drawer) {
        this._drawer.classList.remove("open");
      }
      this._updateTabbarActive(null);
    },

    _updateTabbarActive: function (activeName) {
      if (!this._tabbar) return;
      var buttons = this._tabbar.querySelectorAll(".tabbar-btn");
      buttons.forEach(function (btn) {
        btn.classList.toggle("active", btn.dataset.panel === activeName);
      });
    },

    // ---------------------------------------------------------------
    // Layout persistence (localStorage key: dogmud.dashboard.layout.v1)
    // ---------------------------------------------------------------

    // In-memory copy of the last-saved arrange block so _enterWide can
    // honor it after a wide→rail→wide round-trip without re-reading LS.
    _savedArrange: null,

    saveLayout: function () {
      try {
        var el = document.getElementById("dashboard");

        // Column widths (always safe to capture from the #dashboard style)
        var cols = {
          l: el ? el.style.getPropertyValue("--col-l-w") : "",
          r: el ? el.style.getPropertyValue("--col-r-w") : ""
        };

        // Collapsed panels (class survives mode changes — always safe)
        var collapsed = [];
        document.querySelectorAll(".dash-panel.collapsed").forEach(function (p) {
          if (p.dataset.panel) collapsed.push(p.dataset.panel);
        });

        // Popped panels (always safe)
        var popped = Object.keys(this._popped || {});

        // Arrange: only valid in wide mode. In rail/phone the panels are
        // relocated away from their columns, so DOM order is not the real
        // arrangement. Preserve whatever was previously saved for arrange
        // and only overwrite it when we're actually in wide mode.
        var arrange;
        if (this.mode === "wide") {
          arrange = {};
          ["dash-col-left", "dash-col-center", "dash-col-right"].forEach(function (colId) {
            var col = document.getElementById(colId);
            if (!col) return;
            arrange[colId] = [];
            col.querySelectorAll(".dash-panel").forEach(function (p) {
              if (p.dataset.panel) arrange[colId].push(p.dataset.panel);
            });
          });
          this._savedArrange = arrange;
        } else {
          // Read whatever was already persisted so we can write it back unchanged
          try {
            var existing = JSON.parse(localStorage.getItem("dogmud.dashboard.layout.v1") || "{}");
            arrange = existing.arrange || this._savedArrange || null;
          } catch (e) {
            arrange = this._savedArrange || null;
          }
        }

        var payload = { cols: cols, collapsed: collapsed, popped: popped };
        if (arrange) payload.arrange = arrange;

        localStorage.setItem("dogmud.dashboard.layout.v1", JSON.stringify(payload));
      } catch (e) {
        // localStorage may be unavailable (private browsing, quota, etc.) — ignore
      }
    },

    restoreLayout: function () {
      var self = this;
      var data;
      try {
        var raw = localStorage.getItem("dogmud.dashboard.layout.v1");
        if (!raw) return;
        data = JSON.parse(raw);
        if (!data || typeof data !== "object") return;
      } catch (e) {
        return;
      }

      // 1. Column widths
      try {
        var el = document.getElementById("dashboard");
        if (el && data.cols) {
          if (data.cols.l) el.style.setProperty("--col-l-w", data.cols.l);
          if (data.cols.r) el.style.setProperty("--col-r-w", data.cols.r);
        }
      } catch (e) {}

      // 2. Arrange — only apply if we're currently in wide mode
      try {
        if (this.mode === "wide" && data.arrange && typeof data.arrange === "object") {
          this._savedArrange = data.arrange;
          var arrange = data.arrange;
          ["dash-col-left", "dash-col-center", "dash-col-right"].forEach(function (colId) {
            var col = document.getElementById(colId);
            if (!col) return;
            var names = arrange[colId];
            if (!Array.isArray(names)) return;
            names.forEach(function (name) {
              var panel = document.getElementById("panel-" + name);
              if (!panel) return;
              col.appendChild(panel);
              panel.dataset.homeCol = colId;
            });
          });
        } else if (data.arrange) {
          // Cache it so _enterWide can use it later even if we're in rail/phone now
          this._savedArrange = data.arrange;
        }
      } catch (e) {}

      // 3. Collapsed panels
      try {
        if (Array.isArray(data.collapsed)) {
          data.collapsed.forEach(function (name) {
            var panel = document.getElementById("panel-" + name);
            if (panel) panel.classList.add("collapsed");
          });
        }
      } catch (e) {}

      // 4. Popped panels — do this LAST (depends on panels being in place)
      try {
        if (Array.isArray(data.popped)) {
          data.popped.forEach(function (name) {
            self.popout && self.popout(name);
          });
        }
      } catch (e) {}
    },

    // ---------------------------------------------------------------
    // Pop-out panels into floating WinBox windows
    // ---------------------------------------------------------------
    initPopout: function () {
      var self = this;
      this._popped = this._popped || {};
      this._dockAnchor = this._dockAnchor || {};

      // Wire every pop-out button (side panels AND the game feed)
      document.querySelectorAll("#dashboard .ph-popout").forEach(function (btn) {
        var panel = btn.closest(".dash-panel");
        if (!panel) return;
        btn.addEventListener("click", function (ev) {
          ev.stopPropagation();
          self.popout(panel.dataset.panel);
        });
      });
    },

    popout: function (name) {
      var self = this;
      // Toggle: the ⧉ now lives in the floating window, so a second press re-docks.
      if (this._popped[name]) { this._popped[name].close(); return; }

      var panel = document.getElementById("panel-" + name);
      if (!panel) return;

      // Remember the slot so we can re-dock in the SAME position (not the bottom).
      this._dockAnchor[name] = panel.nextElementSibling || null;

      // Mark before creating the WinBox so reflow guards see it immediately
      panel.dataset.popped = "1";

      var titleEl = panel.querySelector(".ph-title");
      var title = titleEl ? titleEl.textContent.trim() : name;

      var wb = new WinBox({
        title: title,
        mount: panel,
        class: "dash-winbox",
        width: 360,
        height: 300,
        onclose: function () {
          self.redock(name);
          // return undefined → WinBox proceeds with close normally
        }
      });

      this._popped[name] = wb;
      this.saveLayout && this.saveLayout();
    },

    redock: function (name) {
      var panel = document.getElementById("panel-" + name);
      if (panel) {
        panel.removeAttribute("style");      // drop WinBox inline sizing → CSS flex governs again
        panel.removeAttribute("data-popped");
        var homeId = panel.dataset.homeCol;
        var col = homeId ? document.getElementById(homeId) : null;
        if (col) {
          var anchor = this._dockAnchor ? this._dockAnchor[name] : null;
          if (anchor && anchor.parentNode === col) col.insertBefore(panel, anchor);
          else col.appendChild(panel);
        }
      }
      if (this._dockAnchor) delete this._dockAnchor[name];
      delete this._popped[name];
      this.saveLayout && this.saveLayout();
    },

    // ---------------------------------------------------------------
    // Column-width splitters (wide mode only)
    // ---------------------------------------------------------------
    initSplitters: function () {
      var self = this;
      var el = document.getElementById("dashboard");

      function clamp(v, min, max) {
        return Math.max(min, Math.min(max, v));
      }

      function makeSplitter(id, edge) {
        var s = document.createElement("div");
        s.id = id;
        s.className = "dash-splitter col";
        s.dataset.edge = edge;
        el.appendChild(s);

        var rect = null;

        s.addEventListener("pointerdown", function (ev) {
          if (self.mode !== "wide") return;
          rect = el.getBoundingClientRect();
          s.classList.add("dragging");
          s.setPointerCapture(ev.pointerId);
          ev.preventDefault();
        });

        s.addEventListener("pointermove", function (ev) {
          if (!s.classList.contains("dragging")) return;
          if (edge === "l") {
            var w = clamp(ev.clientX - rect.left, 180, rect.width * 0.45);
            el.style.setProperty("--col-l-w", w + "px");
          } else {
            var w = clamp(rect.right - ev.clientX, 180, rect.width * 0.45);
            el.style.setProperty("--col-r-w", w + "px");
          }
        });

        s.addEventListener("pointerup", function (ev) {
          if (!s.classList.contains("dragging")) return;
          s.classList.remove("dragging");
          s.releasePointerCapture(ev.pointerId);
          self.saveLayout && self.saveLayout();
        });
      }

      makeSplitter("dash-splitter-l", "l");
      makeSplitter("dash-splitter-r", "r");
    },

    // ---------------------------------------------------------------
    // Drag-to-swap panel rearrange + collapse toggle (wide mode only)
    // ---------------------------------------------------------------
    initRearrange: function () {
      var self = this;
      var dragSourcePanel = null; // data-panel value of the panel being dragged

      // Helper: swap two <section> elements in the DOM, regardless of
      // whether they live in the same or different parent columns.
      function swapNodes(a, b) {
        var marker = document.createComment("swap");
        a.parentNode.insertBefore(marker, a);
        b.parentNode.insertBefore(a, b);
        marker.parentNode.insertBefore(b, marker);
        marker.parentNode.removeChild(marker);
      }

      // Update data-homeCol for a panel to reflect its current column.
      function updateHomeCol(panel) {
        var col = panel.closest(".dash-col");
        if (col) panel.dataset.homeCol = col.id;
      }

      var panels = document.querySelectorAll(".dash-panel");

      panels.forEach(function (panel) {
        var head = panel.querySelector(".dash-panel-head");
        if (!head) return;

        // --- Drag handle: mark header as draggable ---
        head.setAttribute("draggable", "true");

        // Prevent the small button spans from initiating a swap drag
        var btnSpans = head.querySelectorAll(".ph-collapse, .ph-popout");
        btnSpans.forEach(function (span) {
          span.setAttribute("draggable", "false");
        });

        // dragstart — only if wide mode
        head.addEventListener("dragstart", function (ev) {
          if (self.mode !== "wide") {
            ev.preventDefault();
            return;
          }
          dragSourcePanel = panel.dataset.panel;
          ev.dataTransfer.effectAllowed = "move";
          ev.dataTransfer.setData("text/plain", dragSourcePanel);
        });

        head.addEventListener("dragend", function () {
          dragSourcePanel = null;
          // clear any lingering drop highlight (e.g. drag released off-window)
          var hl = document.querySelectorAll("#dashboard .dash-panel.drag-over");
          for (var i = 0; i < hl.length; i++) hl[i].classList.remove("drag-over");
        });

        // --- Drop target: the entire panel section ---
        panel.addEventListener("dragover", function (ev) {
          if (!dragSourcePanel || self.mode !== "wide") return;
          if (panel.dataset.panel === dragSourcePanel) return;
          ev.preventDefault();
          panel.classList.add("drag-over");
        });

        panel.addEventListener("dragleave", function () {
          panel.classList.remove("drag-over");
        });

        panel.addEventListener("drop", function (ev) {
          ev.preventDefault();
          panel.classList.remove("drag-over");

          if (!dragSourcePanel || self.mode !== "wide") return;
          var targetPanelName = panel.dataset.panel;
          if (targetPanelName === dragSourcePanel) return;

          var srcPanel = document.getElementById("panel-" + dragSourcePanel);
          var dstPanel = panel;
          if (!srcPanel || !dstPanel) return;

          swapNodes(srcPanel, dstPanel);

          // Update data-homeCol for both panels (critical for responsive restore)
          updateHomeCol(srcPanel);
          updateHomeCol(dstPanel);

          dragSourcePanel = null;
          self.saveLayout && self.saveLayout();
        });

        // --- Collapse toggle ---
        var collapseBtn = head.querySelector(".ph-collapse");
        if (collapseBtn) {
          collapseBtn.addEventListener("click", function (ev) {
            ev.stopPropagation();
            panel.classList.toggle("collapsed");
            self.saveLayout && self.saveLayout();
          });
        }
      });
    },
  };

  window.Dashboard = Dashboard;

  document.addEventListener("DOMContentLoaded", function () {
    Dashboard.init();
  });

})();
