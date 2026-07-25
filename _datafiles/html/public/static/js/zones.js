"use strict";
/*
 * zones.js — the zone-config editor (admin web-building 4), a fourth mode of
 * the /build page. Consumes Build.Zones (list) + Build.Zone (detail) GMCP and
 * drives Build.Zone.Create/Update/Delete. The "+ New Zone" button at the top
 * of the list opens an in-panel create form (mirroring Rooms mode's own
 * "+ Zone" flow, which is left in place — both hit the same
 * Build.Zone.Create verb). This panel also edits an existing zone's config
 * and deletes empty ones, and renames one via the guarded Rename control.
 * Mirrors mobs.js's module shape, field helpers, and save/delete/refs flow.
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
  function toast(m, e) { if (window.Builder && window.Builder.toast) window.Builder.toast(m, e); }

  function field(labelText, input, hint) {
    var kids = [ce("label", { text: labelText }), input];
    if (hint) kids.push(ce("div", { style: "font-size:10px;color:var(--gold-dim);margin-top:2px;", text: hint }));
    return ce("div", {}, kids);
  }
  function sectionTitle(t) { return ce("h3", { text: t }); }
  function markDirty() { Panel.setDirty(true); }

  // Exact copy the server rejects Build.Zone.Update with server-side (Task 6's
  // guard in gmcp.Zone.go) — shown here BEFORE the author hits Save so the
  // problem is explained up front. The server remains authoritative; this is
  // advisory only. A non-Cartesian zone marks its coordinate plane
  // non-Euclidean, and the plane registry accumulates that with a sticky OR —
  // so on plane 0 it marks the entire overworld non-Euclidean and silently
  // disables map-consistency enforcement for the whole world.
  var NON_CARTESIAN_WARNING = "A non-Cartesian zone needs its own plane.";

  var Panel = {
    rows: [],
    selectedZone: "",
    listBody: null,
    detail: null,
    dirty: false,
    fields: null,     // gather closures for the current form
    saving: false,
    deleting: false,
    creating: false,      // a Build.Zone.Create is in flight
    renaming: false,      // a Build.Zone.Rename is in flight
    pendingRenameName: "", // remembered rename target — Build.Result for
                           // Zone.Rename does not echo the new name either
    pendingCreateName: "", // remembered create-form name — Build.Result for
                           // Zone.Create doesn't echo it, only roomId
    lastEnums: null,      // enums from the most recently viewed zone detail —
                           // fallback biome source if window.Builder.biomes
                           // hasn't been populated yet (Zone.Map not seen)
    guardBlocked: false, // non-Cartesian-on-plane-0 — Save stays disabled regardless of dirty
  };

  // ---- list ----
  Panel.renderList = function (obj) {
    this.rows = (obj && obj.zones) || [];
    var host = document.getElementById("zonelist");
    if (!host) return;
    host.innerHTML = "";
    var newBtn = ce("button", { "class": "newitem", text: "+ New Zone" });
    newBtn.addEventListener("click", function () { Panel.renderCreateForm(); });
    host.appendChild(newBtn);
    this.listBody = ce("div", {});
    host.appendChild(this.listBody);
    this.drawRows();
  };

  Panel.drawRows = function () {
    if (!this.listBody) return;
    this.listBody.innerHTML = "";
    var self = this;
    var sorted = this.rows.slice().sort(function (a, b) { return (a.zone || "").localeCompare(b.zone || ""); });
    sorted.forEach(function (r) {
      var row = ce("div", { "class": "irow" + (r.zone === self.selectedZone ? " sel" : "") });
      row.appendChild(document.createTextNode(r.zone || "(unnamed)"));
      row.appendChild(ce("span", { "class": "mzone", text: "  " + r.roomCount + " room" + (r.roomCount === 1 ? "" : "s") }));
      if (r.instanced) row.appendChild(ce("span", { "class": "mbadge", text: "instanced" }));
      row.addEventListener("click", function () { Panel.selectZone(r.zone); });
      self.listBody.appendChild(row);
    });
    if (!sorted.length) this.listBody.appendChild(ce("div", { "style": "color:var(--gold-dim);font-style:italic;padding:8px;", text: "no zones" }));
  };

  Panel.selectZone = function (zone) {
    if (this.dirty && !window.confirm("Discard unsaved changes to this zone?")) return;
    this.selectedZone = zone;
    this.drawRows();
    gmcp("Build.Zone.Get", { zone: zone });
  };

  // ---- create ----
  // Mirrors build.html's Rooms-mode renderNewZoneForm (same fields, same
  // copy) but lives entirely in this panel so it can run its own onResult
  // flow — build.html routes Build.Result to ZP.onResult whenever
  // window.Builder.mode === "zones" BEFORE the Rooms-mode create handling
  // ever sees it. Deliberately does NOT carry the edit form's non-Cartesian/
  // plane-0 guard: buildZoneCreate always allocates a fresh plane, so a new
  // zone can never land on plane 0.
  Panel.renderCreateForm = function () {
    if (this.dirty && !window.confirm("Discard unsaved changes to this zone?")) return;
    this.detail = null; this.fields = null; this.dirty = false; this.selectedZone = "";
    this.saving = false; this.deleting = false; this.guardBlocked = false;
    this.drawRows();

    var insp = document.getElementById("inspector");
    insp.innerHTML = "";
    insp.appendChild(ce("h2", { text: "New Zone" }));
    insp.appendChild(ce("div", { "class": "tag", "style": "color:var(--gold-dim);font-size:11px;margin-bottom:6px;",
      text: "Creates a zone on its own plane with one entrance room." }));

    var name = ce("input", { type: "text", placeholder: "e.g. Sunken Crypt" });
    insp.appendChild(field("Zone name", name));

    // window.Builder.biomes is populated globally from the Zone.Map GMCP
    // snapshot the server pushes right after login (before an author could
    // ever reach this form) — but fall back to the biome enum cached off the
    // most recently viewed zone detail in case that snapshot hasn't landed.
    var biomeList = (window.Builder.biomes && window.Builder.biomes.length)
      ? window.Builder.biomes
      : ((this.lastEnums && this.lastEnums.biomes) || []);
    var biome = ce("select", {});
    biome.appendChild(ce("option", { value: "", text: "(default)" }));
    biomeList.forEach(function (b) { biome.appendChild(ce("option", { value: b, text: b })); });
    insp.appendChild(field("Default biome", biome));

    var region = ce("input", { type: "text", placeholder: "e.g. Thornwall Region (optional)" });
    insp.appendChild(field("Region", region));

    var nc = ce("input", { type: "checkbox" });
    insp.appendChild(ce("div", { "class": "flags", "style": "margin-top:8px;" }, [
      ce("label", { "class": "chk" }, [nc, ce("span", { text: " Non-Cartesian (maze / toroidal — skips grid checks)" })])
    ]));

    var errEl = ce("div", { id: "zone-nz-err", "style": "color:var(--danger);font-size:12px;min-height:1em;margin-top:8px;" });
    insp.appendChild(errEl);

    var create = ce("button", { text: "Create zone" });
    create.addEventListener("click", function () {
      var n = name.value.trim();
      if (n.length < 2) { errEl.textContent = "Zone name must be at least 2 characters."; return; }
      errEl.textContent = "";
      Panel.creating = true;
      Panel.pendingCreateName = n;
      gmcp("Build.Zone.Create", { name: n, biome: biome.value, region: region.value.trim(), nonCartesian: nc.checked });
    });
    var cancel = ce("button", { "class": "mini", text: "Cancel", "style": "width:100%;margin-top:6px;padding:6px;" });
    cancel.addEventListener("click", function () { Panel.clear(); });
    insp.appendChild(ce("div", { "class": "save-row" }, [create, cancel]));
    name.focus();
  };

  // ---- form ----
  Panel.renderDetail = function (detail) {
    this.detail = detail;
    this.selectedZone = detail.name;
    this.drawRows();
    var insp = document.getElementById("inspector");
    insp.innerHTML = "";
    this.fields = {};
    var F = this.fields;
    var enums = detail.enums || {};
    this.lastEnums = enums;

    insp.appendChild(ce("h2", { text: "Zone: " + detail.name }));

    // ---- field builders bound to this render's F/markDirty closure ----
    function textField(label, key, val, hint) {
      var i = ce("input", { type: "text" }); i.value = val == null ? "" : val;
      i.addEventListener("input", markDirty);
      F[key] = function () { return i.value; };
      return field(label, i, hint);
    }
    function numField(label, key, val, hint) {
      // Integer fields snap to a whole number on blur (Go rejects a fractional
      // number into an int field and fails the WHOLE Save payload) — same
      // discipline as items.js/mobs.js.
      var i = ce("input", { type: "number", step: "1" }); i.value = (val === 0 ? "0" : (val || ""));
      i.addEventListener("input", markDirty);
      i.addEventListener("blur", function () { if (i.value !== "") i.value = String(Math.round(parseFloat(i.value) || 0)); });
      F[key] = function () { return Math.round(parseFloat(i.value) || 0); };
      return field(label, i, hint);
    }
    function checkField(label, key, val) {
      var cb = ce("input", { type: "checkbox" }); cb.checked = !!val; cb.addEventListener("change", markDirty);
      F[key] = function () { return cb.checked; };
      return ce("label", { "class": "chk" }, [cb, ce("span", { text: " " + label })]);
    }
    function selectField(label, key, val, opts, hint) {
      var s = ce("select", {});
      opts.forEach(function (o) {
        var ov = (typeof o === "object") ? o.v : o, ot = (typeof o === "object") ? o.t : o;
        var op = ce("option", { value: ov, text: (ot === "" || ot == null) ? "(default)" : ot });
        if (String(ov) === String(val)) op.selected = true;
        s.appendChild(op);
      });
      s.addEventListener("change", markDirty);
      F[key] = function () { return s.value; };
      return field(label, s, hint);
    }
    // Free-text field with a suggestion datalist (region names observed
    // across other zones) — not a closed set like biome/deathPolicy.
    function datalistField(label, key, val, dlOpts, hint) {
      var i = ce("input", { type: "text" }); i.value = val == null ? "" : val;
      i.addEventListener("input", markDirty);
      var kids = [i];
      if (dlOpts && dlOpts.length) {
        var dlid = "dl-zone-" + key;
        var dl = ce("datalist", { id: dlid });
        dlOpts.forEach(function (o) { dl.appendChild(ce("option", { value: o })); });
        i.setAttribute("list", dlid);
        kids.push(dl);
      }
      F[key] = function () { return i.value; };
      return field(label, ce("div", {}, kids), hint);
    }
    // Free-text line list (idle messages). Blank lines are dropped on
    // gather — unlike mobs.js's flavor command pools, a blank idle message
    // isn't a meaningful "do nothing" entry here, it's just an empty row the
    // author never filled in.
    function listField(label, key, vals, hint) {
      var wrap = ce("div", {});
      wrap.appendChild(ce("label", { text: label }));
      var box = ce("div", {});
      var rows = [];
      function addRow(v) {
        var i = ce("input", { type: "text" }); i.value = v == null ? "" : v;
        i.addEventListener("input", markDirty);
        var rm = ce("button", { type: "button", "class": "mini rm", text: "✕" });
        var row = ce("div", { "class": "kv" }, [i, rm]);
        rm.addEventListener("click", function () { box.removeChild(row); rows.splice(rows.indexOf(row), 1); markDirty(); });
        row._i = i;
        rows.push(row); box.appendChild(row);
      }
      (vals || []).forEach(addRow);
      wrap.appendChild(box);
      var addBtn = ce("button", { type: "button", "class": "mini", text: "+ message" });
      addBtn.addEventListener("click", function () { addRow(""); markDirty(); });
      wrap.appendChild(addBtn);
      if (hint) wrap.appendChild(ce("div", { style: "font-size:10px;color:var(--gold-dim);margin-top:2px;", text: hint }));
      F[key] = function () {
        return rows.map(function (r) { return r._i.value; }).filter(function (v) { return v.trim(); });
      };
      return wrap;
    }

    // ---- Identity ----
    insp.appendChild(sectionTitle("Identity"));

    // Name is not part of the Save payload — renaming moves ten directories
    // and rewrites the zone name inside every file that stores it, so it is
    // its own guarded action rather than an editable field.
    var nameRow = ce("div", {});
    nameRow.appendChild(ce("label", { text: "Name" }));
    var nameShown = ce("div", { style: "padding:5px 0;color:var(--parch);", text: detail.name });
    nameRow.appendChild(nameShown);

    var renameBox = ce("div", { style: "display:none;margin-top:4px;" });
    var renameInput = ce("input", { type: "text" });
    renameInput.value = detail.name;
    renameBox.appendChild(renameInput);
    renameBox.appendChild(ce("div", {
      style: "font-size:10px;color:var(--gold-dim);margin-top:4px;line-height:1.4;",
      text: "Moves every directory this zone owns and rewrites its name inside "
        + "each file. Blocked while anyone is in the zone. Players' explored-map "
        + "history for this zone is not carried over — it will look unexplored "
        + "to them again. That is expected, not a bug."
    }));
    var renameErr = ce("div", { id: "zone-rn-err", style: "color:var(--danger);font-size:12px;min-height:1em;margin-top:4px;" });
    renameBox.appendChild(renameErr);

    var renameBtn = ce("button", { "class": "mini", text: "Rename…", style: "margin-top:4px;padding:4px 10px;" });
    renameBtn.addEventListener("click", function () {
      var showing = renameBox.style.display !== "none";
      renameBox.style.display = showing ? "none" : "";
      renameBtn.textContent = showing ? "Rename…" : "Cancel rename";
      if (!showing) { renameInput.value = detail.name; renameErr.textContent = ""; renameInput.focus(); }
    });

    var confirmBtn = ce("button", { text: "Confirm rename", style: "margin-top:6px;" });
    confirmBtn.addEventListener("click", function () {
      var n = renameInput.value.trim();
      if (n.length < 2) { renameErr.textContent = "Zone name must be at least 2 characters."; return; }
      if (n === detail.name) { renameErr.textContent = "That is already the zone's name."; return; }
      renameErr.textContent = "";
      Panel.renaming = true;
      // Build.Result does not echo the new name, so remember what we sent in
      // order to reselect the zone afterwards — same as the create flow.
      Panel.pendingRenameName = n;
      gmcp("Build.Zone.Rename", { zone: detail.name, newName: n });
    });
    renameBox.appendChild(confirmBtn);

    nameRow.appendChild(renameBtn);
    nameRow.appendChild(renameBox);
    insp.appendChild(nameRow);
    insp.appendChild(numField("Root room id", "roomId", detail.roomId));
    var biomeOpts = [{ v: "", t: "" }].concat(enums.biomes || []);
    insp.appendChild(selectField("Default biome", "defaultBiome", detail.defaultBiome, biomeOpts));
    insp.appendChild(datalistField("Region", "region", detail.region, enums.regions || []));
    insp.appendChild(textField("Music file", "musicFile", detail.musicFile));
    insp.appendChild(listField("Idle messages", "idleMessages", detail.idleMessages));

    // ---- Map (non-Cartesian / plane guard — built by hand, not via the
    // generic helpers above, because the two controls need to watch each
    // other) ----
    insp.appendChild(sectionTitle("Map"));
    var planeInput = ce("input", { type: "number", step: "1" });
    planeInput.value = (detail.defaultPlane === 0 ? "0" : (detail.defaultPlane || ""));
    var ncInput = ce("input", { type: "checkbox" }); ncInput.checked = !!detail.nonCartesian;
    var ncWarn = ce("div", { style: "font-size:11px;color:var(--danger);margin-top:4px;display:none;", text: NON_CARTESIAN_WARNING });

    function recomputeGuard() {
      var plane = Math.round(parseFloat(planeInput.value) || 0);
      var blocked = ncInput.checked && plane === 0;
      Panel.guardBlocked = blocked;
      ncWarn.style.display = blocked ? "" : "none";
      Panel.applySaveState();
    }
    planeInput.addEventListener("input", function () { markDirty(); recomputeGuard(); });
    planeInput.addEventListener("blur", function () { if (planeInput.value !== "") planeInput.value = String(Math.round(parseFloat(planeInput.value) || 0)); });
    ncInput.addEventListener("change", function () { markDirty(); recomputeGuard(); });

    F.defaultPlane = function () { return Math.round(parseFloat(planeInput.value) || 0); };
    F.nonCartesian = function () { return ncInput.checked; };

    insp.appendChild(field("Default plane", planeInput));
    insp.appendChild(ce("div", { "class": "flags" }, [
      ce("label", { "class": "chk" }, [ncInput, ce("span", { text: " Non-Cartesian (maze / toroidal — skips grid checks)" })])
    ]));
    insp.appendChild(ncWarn);

    // ---- Instanced ----
    insp.appendChild(sectionTitle("Instanced"));
    var instInput = ce("input", { type: "checkbox" }); instInput.checked = !!detail.instanced;
    F.instanced = function () { return instInput.checked; };
    insp.appendChild(ce("div", { "class": "flags" }, [
      ce("label", { "class": "chk" }, [instInput, ce("span", { text: " Instanced (own entry room, death policy, recall)" })])
    ]));

    var instBlock = ce("div", {});
    instBlock.appendChild(numField("Entry room id", "entryRoom", detail.entryRoom));
    instBlock.appendChild(textField("Portal duration", "portalDuration", detail.portalDuration,
      "free text, engine-parsed — e.g. \"10m\", \"1h\""));
    var deathPolicyOpts = [{ v: "", t: "" }].concat(enums.deathPolicies || []);
    instBlock.appendChild(selectField("Death policy", "deathPolicy", detail.deathPolicy, deathPolicyOpts));
    instBlock.appendChild(ce("div", { "class": "flags" }, [checkField("Allow recall", "allowRecall", detail.allowRecall)]));
    insp.appendChild(instBlock);

    function toggleInstancedBlock() { instBlock.style.display = instInput.checked ? "" : "none"; }
    instInput.addEventListener("change", toggleInstancedBlock);
    toggleInstancedBlock();

    // ---- Save + delete ----
    var save = ce("button", { id: "zone-save", text: "Save Zone", disabled: "disabled" });
    save.addEventListener("click", function () { Panel.save(); });
    var del = ce("button", { "class": "mini rm", text: "Delete zone", "style": "width:100%;margin-top:6px;padding:6px;" });
    del.addEventListener("click", function () { Panel.del(); });
    insp.appendChild(ce("div", { "class": "save-row" }, [save, del]));
    insp.appendChild(ce("div", { id: "zone-refs", "style": "color:var(--danger);font-size:11px;margin-top:6px;" }));

    recomputeGuard();
    this.setDirty(false);
  };

  // Save stays disabled while either nothing has changed OR the non-Cartesian
  // guard is tripped — the guard check overrides plain dirty-tracking.
  Panel.applySaveState = function () {
    var s = document.getElementById("zone-save");
    if (s) s.disabled = !this.dirty || this.guardBlocked;
  };
  Panel.setDirty = function (d) {
    this.dirty = d;
    this.applySaveState();
  };

  // ---- mutations ----
  Panel.gather = function () {
    var F = this.fields || {};
    var d = this.detail || {};
    function g(k, dflt) { return F[k] ? F[k]() : (d[k] !== undefined ? d[k] : dflt); }
    return {
      name: d.name, // not editable via Save — renaming is its own guarded action
      roomId: g("roomId", 0),
      defaultBiome: g("defaultBiome", ""),
      region: g("region", ""),
      musicFile: g("musicFile", ""),
      idleMessages: g("idleMessages", []),
      instanced: g("instanced", false),
      deathPolicy: g("deathPolicy", ""),
      portalDuration: g("portalDuration", ""),
      entryRoom: g("entryRoom", 0),
      allowRecall: g("allowRecall", false),
      nonCartesian: g("nonCartesian", false),
      defaultPlane: g("defaultPlane", 0)
    };
  };

  Panel.save = function () {
    if (!this.detail) return;
    if (this.guardBlocked) { toast(NON_CARTESIAN_WARNING, true); return; }
    var req = this.gather();
    this.saving = true;
    gmcp("Build.Zone.Update", req);
  };
  Panel.del = function () {
    if (!this.detail) return;
    if (!window.confirm("Delete zone \"" + this.detail.name + "\"? This cannot be undone.")) return;
    this.deleting = true;
    gmcp("Build.Zone.Delete", { zone: this.detail.name });
  };

  // Build.Result routing from build.html (mode === "zones").
  Panel.onResult = function (obj) {
    if (obj && obj.ok) {
      if (this.deleting) {
        this.deleting = false; this.detail = null; this.selectedZone = ""; this.clear();
        toast("Zone deleted.", false);
        return;
      }
      if (this.saving) {
        this.saving = false; this.setDirty(false); toast("Saved.", false);
        // Reload the persisted config so the form reflects exactly what was
        // stored.
        if (this.selectedZone) gmcp("Build.Zone.Get", { zone: this.selectedZone });
        return;
      }
      if (this.renaming) {
        this.renaming = false;
        var renamedTo = this.pendingRenameName;
        this.pendingRenameName = "";
        toast("Zone renamed.", false);
        this.detail = null;
        this.selectedZone = renamedTo;
        this.clear();
        gmcp("Build.Zone.List", {});
        if (renamedTo) gmcp("Build.Zone.Get", { zone: renamedTo });
        return;
      }
      if (this.creating) {
        this.creating = false;
        // Build.Result for Zone.Create only echoes { ok, roomId } — the zone
        // name has to come from what we remembered submitting — so select by
        // the name we sent, not anything off obj.
        var createdName = this.pendingCreateName;
        this.pendingCreateName = "";
        toast("Zone created.", false);
        this.clear();
        gmcp("Build.Zone.List", {});
        if (createdName) gmcp("Build.Zone.Get", { zone: createdName });
        return;
      }
    } else {
      this.saving = false; this.deleting = false;
      if (this.renaming) {
        this.renaming = false;
        this.pendingRenameName = "";
        // Leave the rename box open and populated so the author can fix
        // the name, or read who is standing in the zone, and retry.
        var rnErr = document.getElementById("zone-rn-err");
        if (rnErr) rnErr.textContent = (obj && obj.error) || "Rename failed";
      }
      if (this.creating) {
        this.creating = false;
        var errEl = document.getElementById("zone-nz-err");
        if (errEl) errEl.textContent = (obj && obj.error) || "Zone error";
        // Deliberately do NOT clear pendingCreateName or re-render the
        // form here — the create form's inputs are untouched in the DOM so
        // the author can fix the name/fields and resubmit without retyping.
      }
      if (obj && obj.zoneRefs && obj.zoneRefs.length) this.showDeleteRefs(obj.zoneRefs);
      toast((obj && obj.error) || "Zone error", true);
    }
  };

  Panel.showDeleteRefs = function (refs) {
    var el = document.getElementById("zone-refs");
    if (!el) return;
    el.textContent = "Still referenced: " + refs.map(function (r) { return r.kind + " — " + r.id; }).join("; ") + " — remove these first.";
  };

  // ---- inspector placeholder ----
  function clearZoneInspector() {
    var insp = document.getElementById("inspector");
    if (!insp) return;
    insp.innerHTML = "";
    insp.appendChild(ce("h2", { text: "Zones" }));
    insp.appendChild(ce("div", { "class": "empty", text: "Select a zone on the left to edit it, or + New Zone." }));
  }
  Panel.clear = function () {
    this.detail = null; this.fields = null; this.dirty = false; this.selectedZone = "";
    this.saving = false; this.deleting = false; this.creating = false; this.renaming = false; this.guardBlocked = false;
    clearZoneInspector();
  };

  window.Builder = window.Builder || {};
  window.Builder.ZonesPanel = Panel;
})();
