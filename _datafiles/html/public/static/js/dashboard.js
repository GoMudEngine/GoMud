/* dashboard.js — Dashboard controller + responsive column-stack reflow
 * Three breakpoints:
 *   wide  >= 1100px : default three-column layout (no changes)
 *   rail  700–1100px: left+right cols hidden; side panels in a tabbed rail
 *   phone < 700px   : center fills; fixed bottom tab-bar; slide-up drawer
 */
"use strict";

(function () {

  var SIDE_PANELS = ["map", "vitals", "art", "chat", "status", "trig"];

  // Icons for the phone tab-bar (order matches SIDE_PANELS)
  var PANEL_ICONS = {
    map:    "🗺",
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
    resetLayout: function () {
      try { localStorage.removeItem("dogmud.dashboard.layout.v1"); } catch (e) {}
      location.reload();
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
      // Move every side panel back to its home column in original order.
      // SIDE_PANELS is defined in home-col grouping order
      // (map/vitals/art → left, chat/status/trig → right)
      SIDE_PANELS.forEach(function (name) {
        var panel = document.getElementById("panel-" + name);
        if (!panel) return;
        var homeId = panel.dataset.homeCol;
        if (!homeId) return;
        var col = document.getElementById(homeId);
        if (col) col.appendChild(panel);
        panel.style.display = "";
      });

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
        panel.style.display = (pName === name) ? "" : "none";
      });
    },

    _exitRail: function () {
      // Move panels back to home cols (wide will re-place them)
      var self = this;
      SIDE_PANELS.forEach(function (name) {
        var panel = document.getElementById("panel-" + name);
        if (!panel) return;
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
  };

  window.Dashboard = Dashboard;

  document.addEventListener("DOMContentLoaded", function () {
    Dashboard.init();
  });

})();
