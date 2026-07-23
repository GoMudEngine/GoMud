"use strict";
/*
 * mobs.js — the mob-template list (admin web-building 3), a third mode of the
 * /build page. Consumes Build.Mobs (list) GMCP and drives Build.Mob.Get /
 * Build.Mob.Create. The sectioned mob form (Build.Mob detail, Save/Delete) is
 * Task 6; the test-spawn control is Task 7 — this file is list-only for now.
 */
(function () {
  function ce(tag, attrs, kids) {
    var e = document.createElement(tag);
    if (attrs) for (var k in attrs) {
      if (k === "text") e.textContent = attrs[k];
      else if (k === "html") e.innerHTML = attrs[k];
      else e.setAttribute(k, attrs[k]);
    }
    (kids || []).forEach(function (c) { if (c) e.appendChild(c); });
    return e;
  }
  function gmcp(pkg, obj) { if (window.Builder && window.Builder.sendGMCP) window.Builder.sendGMCP(pkg, obj); }

  var Panel = {
    rows: [],
    search: "",
    zoneFilter: "",
    zones: [],       // distinct zones observed across the current rows
    selectedId: 0,
    listBody: null,
  };

  // ---- list ----
  Panel.render = function (rows) {
    this.rows = rows || [];
    var host = document.getElementById("moblist");
    if (!host) return;
    host.innerHTML = "";

    var distinct = {};
    this.rows.forEach(function (r) { distinct[r.zone] = true; });
    this.zones = Object.keys(distinct).sort();

    var newBtn = ce("button", { "class": "newitem", text: "+ New Mob" });
    newBtn.addEventListener("click", promptNewMob);
    host.appendChild(newBtn);

    var filters = ce("div", { "class": "filters" });
    var search = ce("input", { type: "text", placeholder: "search id or name" });
    search.value = this.search;
    search.addEventListener("input", function () { Panel.search = search.value; Panel.drawRows(); });
    var zoneSel = ce("select", {});
    zoneSel.appendChild(ce("option", { value: "", text: "all zones" }));
    this.zones.forEach(function (z) {
      var o = ce("option", { value: z, text: z });
      if (z === Panel.zoneFilter) o.selected = true;
      zoneSel.appendChild(o);
    });
    zoneSel.addEventListener("change", function () { Panel.zoneFilter = zoneSel.value; Panel.drawRows(); });
    filters.appendChild(search);
    filters.appendChild(zoneSel);
    host.appendChild(filters);

    this.listBody = ce("div", {});
    host.appendChild(this.listBody);
    this.drawRows();
  };

  Panel.drawRows = function () {
    if (!this.listBody) return;
    this.listBody.innerHTML = "";
    var q = this.search.trim().toLowerCase();
    var zf = this.zoneFilter;
    var shown = 0;
    this.rows.forEach(function (r) {
      if (zf && r.zone !== zf) return;
      if (q && String(r.id).indexOf(q) === -1 && (r.name || "").toLowerCase().indexOf(q) === -1) return;
      shown++;
      var row = ce("div", { "class": "irow" + (r.id === Panel.selectedId ? " sel" : "") });
      row.appendChild(ce("span", { "class": "iid", text: "#" + r.id + " " }));
      row.appendChild(document.createTextNode(r.name || "(unnamed)"));
      row.appendChild(ce("span", { "class": "mzone", text: "  " + r.zone + " · pool " + r.statPool }));
      if (r.nonCombatant) row.appendChild(ce("span", { "class": "mbadge", text: "non-combatant" }));
      if (r.hasSchedule) row.appendChild(ce("span", { "class": "mbadge", text: "schedule" }));
      if (r.hasShop) row.appendChild(ce("span", { "class": "mbadge", text: "shop" }));
      row.addEventListener("click", function () { Panel.selectItem(r.id); });
      Panel.listBody.appendChild(row);
    });
    if (!shown) this.listBody.appendChild(ce("div", { "style": "color:var(--gold-dim);font-style:italic;padding:8px;", text: "no mobs match" }));
  };

  Panel.selectItem = function (id) {
    this.selectedId = id;
    this.drawRows();
    gmcp("Build.Mob.Get", { mobId: id });
  };

  function promptNewMob() {
    var zones = Panel.zones.length ? Panel.zones : distinctZones();
    var dflt = Panel.zoneFilter || (zones.length ? zones[0] : "");
    var z = window.prompt("New mob — which zone?\n(" + zones.join(", ") + ")", dflt);
    if (!z) return;
    z = z.trim();
    if (!z) return;
    gmcp("Build.Mob.Create", { zone: z });
  }
  function distinctZones() {
    var d = {};
    Panel.rows.forEach(function (r) { d[r.zone] = true; });
    return Object.keys(d).sort();
  }

  // ---- inspector placeholder ----
  // The sectioned mob form (Build.Mob detail) is Task 6; until then, entering
  // Mobs mode or clicking a row just clears the inspector to a placeholder.
  function clearMobInspector() {
    var insp = document.getElementById("inspector");
    if (!insp) return;
    insp.innerHTML = "";
    insp.appendChild(ce("h2", { text: "Mobs" }));
    insp.appendChild(ce("div", { "class": "empty", text: "Select a mob on the left, or + New Mob." }));
  }
  Panel.clear = clearMobInspector;

  window.Builder = window.Builder || {};
  window.Builder.MobsPanel = Panel;
})();
