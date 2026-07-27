"use strict";
/*
 * mobs.js — the mob-template editor (admin web-building 3), a third mode of
 * the /build page. Consumes Build.Mobs (list) + Build.Mob (detail) GMCP and
 * drives Build.Mob.Create/Update/Delete. Mirrors items.js's field/hint/dirty-
 * tracking helpers and its collapsible-Advanced pattern.
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
  function labelWrap(text, input) {
    var d = document.createElement("div"); d.style.flex = "1 1 auto"; d.style.minWidth = "0";
    var l = document.createElement("label"); l.textContent = text; d.appendChild(l); d.appendChild(input); return d;
  }
  function cap(s) { return s ? s.charAt(0).toUpperCase() + s.slice(1) : s; }
  function markDirty() { Panel.setDirty(true); }

  var STAT_KEYS = ["strength", "dexterity", "perception", "vitality", "willpower", "charisma"];
  // Sentinel option value for the mob-list zone filter's "(unzoned)" bucket —
  // distinct from "" (which the filter select already uses for "all zones").
  var UNZONED_FILTER = " unzoned";

  var Panel = {
    rows: [],
    search: "",
    zoneFilter: "",
    zones: [],       // distinct zones observed across the current rows
    selectedId: 0,
    listBody: null,
    detail: null,
    dirty: false,
    fields: null,    // gather closures for the current form
    saving: false,
    deleting: false,
    creating: false,
    spawning: false, // a Build.Mob.Spawn is in flight
    _spawnUI: null,  // { zoneSel, roomSel, spawnBtn, statusEl, pendingZone } for the current form
    advancedOpen: false,
    _advMobId: 0,
  };

  // ---- list ----
  Panel.render = function (rows) {
    this.rows = rows || [];
    var host = document.getElementById("moblist");
    if (!host) return;
    host.innerHTML = "";

    var distinct = {};
    this.rows.forEach(function (r) { distinct[r.zone || ""] = true; });
    // Real zone names only — the "(unzoned)" bucket (zone === "") is a fixed
    // option added separately below, not sorted in among real zone names.
    this.zones = Object.keys(distinct).filter(function (z) { return z; }).sort();
    var hasUnzoned = !!distinct[""];

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
    if (hasUnzoned) {
      var uo = ce("option", { value: UNZONED_FILTER, text: "(unzoned)" });
      if (Panel.zoneFilter === UNZONED_FILTER) uo.selected = true;
      zoneSel.appendChild(uo);
    }
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
      if (zf === UNZONED_FILTER) { if (r.zone) return; }
      else if (zf && r.zone !== zf) return;
      if (q && String(r.id).indexOf(q) === -1 && (r.name || "").toLowerCase().indexOf(q) === -1) return;
      shown++;
      var row = ce("div", { "class": "irow" + (r.id === Panel.selectedId ? " sel" : "") });
      row.appendChild(ce("span", { "class": "iid", text: "#" + r.id + " " }));
      row.appendChild(document.createTextNode(r.name || "(unnamed)"));
      row.appendChild(ce("span", { "class": "mzone", text: "  " + (r.zone || "(unzoned)") + " · pool " + r.statPool }));
      if (r.nonCombatant) row.appendChild(ce("span", { "class": "mbadge", text: "non-combatant" }));
      if (r.hasSchedule) row.appendChild(ce("span", { "class": "mbadge", text: "schedule" }));
      if (r.hasShop) row.appendChild(ce("span", { "class": "mbadge", text: "shop" }));
      row.addEventListener("click", function () { Panel.selectItem(r.id); });
      Panel.listBody.appendChild(row);
    });
    if (!shown) this.listBody.appendChild(ce("div", { "style": "color:var(--gold-dim);font-style:italic;padding:8px;", text: "no mobs match" }));
  };

  Panel.selectItem = function (id) {
    if (this.dirty && !window.confirm("Discard unsaved changes to this mob?")) return;
    this.selectedId = id;
    this.drawRows();
    gmcp("Build.Mob.Get", { mobId: id });
  };

  function promptNewMob() {
    if (Panel.dirty && !window.confirm("Discard unsaved changes to this mob?")) return;
    var zones = Panel.zones.length ? Panel.zones : distinctZones();
    var dflt = (Panel.zoneFilter && Panel.zoneFilter !== UNZONED_FILTER)
      ? Panel.zoneFilter
      : (Panel.zoneFilter === UNZONED_FILTER ? "" : (zones.length ? zones[0] : ""));
    // window.prompt returns null on Cancel but "" on OK with an empty field —
    // the latter is a deliberate "(no zone)" choice (a summon-only / not-yet-
    // placed template) and must NOT be treated as a cancel.
    var z = window.prompt("New mob — which zone? Leave blank for (no zone).\n(" + zones.join(", ") + ")", dflt);
    if (z === null) return;
    z = z.trim();
    Panel.creating = true;
    gmcp("Build.Mob.Create", { zone: z });
  }
  function distinctZones() {
    var d = {};
    Panel.rows.forEach(function (r) { if (r.zone) d[r.zone] = true; });
    return Object.keys(d).sort();
  }

  // ---- test spawn (Task 7; reworked in the browser-eyeball fix round) ----
  // A "Test spawn" row at the top of the form: pick a zone (defaults to the
  // mob's own, or the first available zone for a zoneless mob template, which
  // has none of its own to preselect) and a room from that zone's own
  // dropdown, then Build.Mob.Spawn it in. The room list (Build.Room.List ->
  // Build.Rooms) is scoped to the selected zone and existence-guaranteed by
  // construction, so there is nothing left to "Check" — picking a room from
  // the list is itself the confirmation a raw typed room id used to need.
  Panel.buildTestSpawnRow = function (detail, enums) {
    var self = this;
    var wrap = ce("div", { style: "border:1px solid var(--tooled);border-radius:5px;padding:8px;margin-bottom:12px;" });
    wrap.appendChild(ce("h3", { text: "Test spawn", style: "margin-top:0;" }));
    wrap.appendChild(ce("div", { style: "font-size:10px;color:var(--gold-dim);margin:2px 0 6px;",
      text: "Admin test tool only — drops a disposable copy wherever you pick. Where this mob REALLY spawns is authored on rooms: Rooms tab → the room → Spawns." }));

    var zoneSel = ce("select", {});
    (enums.zones || []).forEach(function (z) {
      var o = ce("option", { value: z, text: z });
      if (z === detail.zone) o.selected = true;
      zoneSel.appendChild(o);
    });

    var roomSel = ce("select", {});
    var spawnBtn = ce("button", { type: "button", text: "Spawn", disabled: "disabled" });
    spawnBtn.style.cssText = "background:var(--gold);color:var(--ink);font-weight:bold;" +
      "border:none;border-radius:4px;padding:4px 12px;cursor:pointer;";
    var statusEl = ce("span", { style: "font-size:11px;color:var(--gold-dim);" });

    var ui = { zoneSel: zoneSel, roomSel: roomSel, spawnBtn: spawnBtn, statusEl: statusEl, pendingZone: "" };
    this._spawnUI = ui;

    function fillRoomPlaceholder(text) {
      roomSel.innerHTML = "";
      roomSel.appendChild(ce("option", { value: "", text: text, disabled: "disabled", selected: "selected" }));
    }

    // Re-requests the room list whenever the zone changes (and once on first
    // render below). pendingZone guards against a stale Build.Rooms response
    // (from a since-abandoned zone selection) repopulating the dropdown.
    function requestRooms() {
      spawnBtn.disabled = true;
      statusEl.textContent = "";
      if (!zoneSel.value) { fillRoomPlaceholder("(no zone selected)"); ui.pendingZone = ""; return; }
      ui.pendingZone = zoneSel.value;
      fillRoomPlaceholder("loading rooms…");
      gmcp("Build.Room.List", { zone: zoneSel.value });
    }
    zoneSel.addEventListener("change", requestRooms);
    roomSel.addEventListener("change", function () { spawnBtn.disabled = !roomSel.value; });

    spawnBtn.addEventListener("click", function () {
      var roomId = parseInt(roomSel.value, 10);
      if (!roomId || !detail.mobId) return;
      self.spawning = true;
      spawnBtn.disabled = true;
      gmcp("Build.Mob.Spawn", { mobId: detail.mobId, roomId: roomId });
      // No accidental double-spawn: hold the button disabled for 2s regardless
      // of when/whether the result arrives. Only re-enable if the picked room
      // is still the one this click fired for (the dropdown wasn't changed).
      setTimeout(function () {
        if (parseInt(roomSel.value, 10) === roomId) spawnBtn.disabled = false;
      }, 2000);
    });

    wrap.appendChild(ce("div", { "class": "row" }, [labelWrap("Zone", zoneSel), labelWrap("Room", roomSel)]));
    wrap.appendChild(ce("div", { style: "margin-top:6px;display:flex;align-items:center;gap:8px;" },
      [spawnBtn, statusEl]));
    fillRoomPlaceholder("— pick a room —");
    requestRooms();
    return wrap;
  };

  // Build.Rooms (from Build.Room.List), routed here by build.html — populates
  // the test-spawn room dropdown for the currently-selected zone.
  Panel.onRoomList = function (rows) {
    var ui = this._spawnUI;
    if (!ui) return;
    // A response for a zone the admin has since changed away from — drop it
    // rather than repopulating the dropdown with the wrong zone's rooms.
    if (ui.pendingZone !== ui.zoneSel.value) return;
    ui.roomSel.innerHTML = "";
    ui.roomSel.appendChild(ce("option", { value: "", text: "— pick a room —", disabled: "disabled", selected: "selected" }));
    (rows || []).forEach(function (r) {
      ui.roomSel.appendChild(ce("option", { value: String(r.id), text: "#" + r.id + " — " + (r.title || "(untitled)") }));
    });
    ui.spawnBtn.disabled = true;
    ui.statusEl.textContent = (rows || []).length ? "" : "no rooms in this zone";
  };

  // ---- form ----
  Panel.renderForm = function (detail) {
    this.detail = detail;
    this.selectedId = detail.mobId;
    this.drawRows();
    var insp = document.getElementById("inspector");
    insp.innerHTML = "";
    this.fields = {};
    var F = this.fields;
    var enums = detail.enums || {};

    insp.appendChild(ce("h2", { text: "Mob #" + detail.mobId }));
    insp.appendChild(this.buildTestSpawnRow(detail, enums));
    // 5b: the dialogue editor opens in a dedicated panel for this mob.
    var dlgBtn = ce("button", { "class": "mini", text: detail.hasDialogue ? "Dialogue…" : "Add dialogue…" });
    dlgBtn.addEventListener("click", function () {
      if (Panel.dirty && !window.confirm("Discard unsaved mob changes?")) return;
      if (window.Builder.DialoguePanel) window.Builder.DialoguePanel.open(detail.mobId, detail.zone);
    });
    insp.appendChild(dlgBtn);

    // Shared id-picker datalists: items from the login-time Build.Items
    // prefetch, buffs from the server enums.
    var itemDl = ce("datalist", { id: "dl-mob-items" });
    ((window.Builder && window.Builder.itemRows) || []).forEach(function (r) {
      itemDl.appendChild(ce("option", { value: String(r.id), text: r.name || "" }));
    });
    insp.appendChild(itemDl);
    var buffDl = ce("datalist", { id: "dl-mob-buffs" });
    (enums.buffs || []).forEach(function (b) {
      buffDl.appendChild(ce("option", { value: String(b.id), text: b.name || "" }));
    });
    insp.appendChild(buffDl);

    // ---- field builders bound to this render's F/markDirty closure ----
    function textField(label, key, val, hint) {
      var i = ce("input", { type: "text" }); i.value = val == null ? "" : val;
      i.addEventListener("input", markDirty);
      F[key] = function () { return i.value; };
      return field(label, i, hint);
    }
    function textAreaField(label, key, val, hint) {
      var t = ce("textarea", {}); t.value = val || "";
      t.addEventListener("input", markDirty);
      F[key] = function () { return t.value; };
      return field(label, t, hint);
    }
    function numField(label, key, val, step) {
      // Integer fields snap to a whole number on blur (Go rejects a fractional
      // number into an int field and fails the WHOLE Save payload) — same
      // discipline as items.js.
      var isInt = !step || step === "1";
      var i = ce("input", { type: "number", step: step || "1" }); i.value = (val === 0 ? "0" : (val || ""));
      i.addEventListener("input", markDirty);
      if (isInt) i.addEventListener("blur", function () {
        if (i.value !== "") i.value = String(Math.round(parseFloat(i.value) || 0));
      });
      F[key] = function () { var n = parseFloat(i.value) || 0; return isInt ? Math.round(n) : n; };
      return field(label, i);
    }
    function checkField(label, key, val) {
      var cb = ce("input", { type: "checkbox" }); cb.checked = !!val; cb.addEventListener("change", markDirty);
      F[key] = function () { return cb.checked; };
      return ce("label", { "class": "chk" }, [cb, ce("span", { text: " " + label })]);
    }
    function selectField(label, key, val, opts, asInt) {
      var s = ce("select", {});
      opts.forEach(function (o) {
        var ov = (typeof o === "object") ? o.v : o, ot = (typeof o === "object") ? o.t : o;
        var op = ce("option", { value: ov, text: (ot === "" || ot == null) ? "(none)" : ot });
        if (String(ov) === String(val)) op.selected = true;
        s.appendChild(op);
      });
      s.addEventListener("change", markDirty);
      F[key] = function () { return asInt ? (parseInt(s.value, 10) || 0) : s.value; };
      return field(label, s);
    }

    // Chip editor for string-array fields (adjectives, groups, routineLinks,
    // hates, knowsFacts, questFlags, spawnMutations, crafterRecipeIds). An
    // optional datalist gives autocomplete suggestions (e.g. observed groups).
    function chipsField(label, key, vals, hint, datalistOpts) {
      var wrap = ce("div", {});
      wrap.appendChild(ce("label", { text: label }));
      var chipBox = ce("div", { "class": "chips" });
      var current = (vals || []).slice();
      function redraw() {
        chipBox.innerHTML = "";
        current.forEach(function (v, idx) {
          var chip = ce("span", { "class": "chip" }, [document.createTextNode(v)]);
          var rm = ce("button", { type: "button", "class": "chip-rm", text: "✕" });
          rm.addEventListener("click", function () { current.splice(idx, 1); redraw(); markDirty(); });
          chip.appendChild(rm);
          chipBox.appendChild(chip);
        });
      }
      redraw();
      wrap.appendChild(chipBox);
      var inputRow = ce("div", { "class": "row" });
      var inp = ce("input", { type: "text", placeholder: "add value, press Enter" });
      if (datalistOpts && datalistOpts.length) {
        var dlid = "dl-" + key;
        var dl = ce("datalist", { id: dlid });
        datalistOpts.forEach(function (o) { dl.appendChild(ce("option", { value: o })); });
        wrap.appendChild(dl);
        inp.setAttribute("list", dlid);
      }
      function addFromInput() {
        var v = inp.value.trim();
        if (!v) return;
        current.push(v);
        inp.value = "";
        redraw();
        markDirty();
      }
      inp.addEventListener("keydown", function (e) { if (e.key === "Enter") { e.preventDefault(); addFromInput(); } });
      var addBtn = ce("button", { type: "button", "class": "mini", text: "+ add" });
      addBtn.addEventListener("click", addFromInput);
      inputRow.appendChild(inp); inputRow.appendChild(addBtn);
      wrap.appendChild(inputRow);
      if (hint) wrap.appendChild(ce("div", { style: "font-size:10px;color:var(--gold-dim);margin-top:2px;", text: hint }));
      // Text typed into the add-box but never committed via Enter/+add would
      // otherwise be silently lost on Save — flush it in as if the add
      // routine had been called (this also updates the visible chip list).
      F[key] = function () {
        if (inp.value.trim()) addFromInput();
        return current.slice();
      };
      return wrap;
    }

    // Repeatable id rows (lootPool, carriedItems, buffIds,
    // crafterRestockMaterials) — id pickers backed by a datalist (dl), so
    // authors see names instead of typing bare numbers (epic followup #3).
    function idRowsField(label, key, vals, hint, dl) {
      var wrap = ce("div", {});
      wrap.appendChild(ce("label", { text: label }));
      var box = ce("div", {});
      var rows = [];
      function addRow(v) {
        var i = ce("input", { type: "text", placeholder: "id" });
        if (dl) i.setAttribute("list", dl);
        i.value = (v || v === 0) ? v : "";
        i.addEventListener("input", markDirty);
        i.addEventListener("blur", function () { if (i.value !== "") i.value = String(Math.round(parseFloat(i.value) || 0)); });
        var rm = ce("button", { type: "button", "class": "mini rm", text: "✕" });
        var row = ce("div", { "class": "kv" }, [i, rm]);
        rm.addEventListener("click", function () { box.removeChild(row); rows.splice(rows.indexOf(row), 1); markDirty(); });
        row._i = i;
        rows.push(row); box.appendChild(row);
      }
      (vals || []).forEach(addRow);
      wrap.appendChild(box);
      var addBtn = ce("button", { type: "button", "class": "mini", text: "+ id" });
      addBtn.addEventListener("click", function () { addRow(0); markDirty(); });
      wrap.appendChild(addBtn);
      if (hint) wrap.appendChild(ce("div", { style: "font-size:10px;color:var(--gold-dim);margin-top:2px;", text: hint }));
      F[key] = function () {
        return rows.map(function (r) { return parseInt(r._i.value, 10) || 0; }).filter(function (n) { return n > 0; });
      };
      return wrap;
    }

    // Flavor command-list rows (idle/combat/angry). Blank rows are real
    // "do nothing" pool entries and are PRESERVED (not filtered) — only an
    // explicit ✕ removes a row. Up/down reorder in place.
    function commandListField(label, key, vals, hint) {
      var wrap = ce("div", {});
      wrap.appendChild(ce("label", { text: label }));
      var box = ce("div", {});
      var rows = [];
      function addRow(v) {
        var i = ce("input", { type: "text" }); i.value = v == null ? "" : v;
        i.addEventListener("input", markDirty);
        var up = ce("button", { type: "button", "class": "mini", text: "↑" });
        var down = ce("button", { type: "button", "class": "mini", text: "↓" });
        var rm = ce("button", { type: "button", "class": "mini rm", text: "✕" });
        var row = ce("div", { "class": "kv" }, [i, up, down, rm]);
        up.addEventListener("click", function () {
          var idx = rows.indexOf(row);
          if (idx <= 0) return;
          var prev = rows[idx - 1];
          box.insertBefore(row, prev);
          rows[idx] = prev; rows[idx - 1] = row;
          markDirty();
        });
        down.addEventListener("click", function () {
          var idx = rows.indexOf(row);
          if (idx === -1 || idx >= rows.length - 1) return;
          var next = rows[idx + 1];
          box.insertBefore(next, row);
          rows[idx] = next; rows[idx + 1] = row;
          markDirty();
        });
        rm.addEventListener("click", function () { box.removeChild(row); rows.splice(rows.indexOf(row), 1); markDirty(); });
        row._i = i;
        rows.push(row); box.appendChild(row);
      }
      (vals || []).forEach(function (v) { addRow(v); });
      wrap.appendChild(box);
      var addBtn = ce("button", { type: "button", "class": "mini", text: "+ line" });
      addBtn.addEventListener("click", function () { addRow(""); markDirty(); });
      wrap.appendChild(addBtn);
      if (hint) wrap.appendChild(ce("div", { style: "font-size:10px;color:var(--gold-dim);margin-top:2px;", text: hint }));
      F[key] = function () { return rows.map(function (r) { return r._i.value; }); }; // blanks preserved intentionally
      return wrap;
    }

    // Shop inventory rows (Commerce).
    function shopRowsField(label, key, vals, hint) {
      var wrap = ce("div", {});
      wrap.appendChild(ce("label", { text: label }));
      var box = ce("div", {});
      var rows = [];
      function addRow(r) {
        r = r || { itemId: 0, quantityMax: 0, price: 0 };
        var iid = ce("input", { type: "text", placeholder: "item id", list: "dl-mob-items" }); iid.value = r.itemId || "";
        var qty = ce("input", { type: "number", step: "1", placeholder: "qty max" }); qty.value = r.quantityMax || "";
        var price = ce("input", { type: "number", step: "1", placeholder: "price" }); price.value = r.price || "";
        [iid, qty, price].forEach(function (el) { el.addEventListener("input", markDirty); });
        var rm = ce("button", { type: "button", "class": "mini rm", text: "✕" });
        var row = ce("div", { "class": "kv" }, [iid, qty, price, rm]);
        rm.addEventListener("click", function () { box.removeChild(row); rows.splice(rows.indexOf(row), 1); markDirty(); });
        row._iid = iid; row._qty = qty; row._price = price;
        rows.push(row); box.appendChild(row);
      }
      (vals || []).forEach(addRow);
      wrap.appendChild(box);
      var addBtn = ce("button", { type: "button", "class": "mini", text: "+ shop item" });
      addBtn.addEventListener("click", function () { addRow(); markDirty(); });
      wrap.appendChild(addBtn);
      if (hint) wrap.appendChild(ce("div", { style: "font-size:10px;color:var(--gold-dim);margin-top:2px;", text: hint }));
      F[key] = function () {
        return rows.map(function (r) {
          return { itemId: parseInt(r._iid.value, 10) || 0, quantityMax: parseInt(r._qty.value, 10) || 0, price: parseInt(r._price.value, 10) || 0 };
        }).filter(function (row) { return row.itemId > 0; });
      };
      return wrap;
    }

    // Relationship rows (Aliveness).
    function relRowsField(label, key, vals, hint) {
      var wrap = ce("div", {});
      wrap.appendChild(ce("label", { text: label }));
      var box = ce("div", {});
      var rows = [];
      function addRow(r) {
        r = r || { to: 0, type: "", subtype: "" };
        var to = ce("input", { type: "number", step: "1", placeholder: "mob id" }); to.value = r.to || "";
        var type = ce("input", { type: "text", placeholder: "type" }); type.value = r.type || "";
        var subtype = ce("input", { type: "text", placeholder: "subtype" }); subtype.value = r.subtype || "";
        [to, type, subtype].forEach(function (el) { el.addEventListener("input", markDirty); });
        var rm = ce("button", { type: "button", "class": "mini rm", text: "✕" });
        var row = ce("div", { "class": "kv" }, [to, type, subtype, rm]);
        rm.addEventListener("click", function () { box.removeChild(row); rows.splice(rows.indexOf(row), 1); markDirty(); });
        row._to = to; row._type = type; row._subtype = subtype;
        rows.push(row); box.appendChild(row);
      }
      (vals || []).forEach(addRow);
      wrap.appendChild(box);
      var addBtn = ce("button", { type: "button", "class": "mini", text: "+ relationship" });
      addBtn.addEventListener("click", function () { addRow(); markDirty(); });
      wrap.appendChild(addBtn);
      if (hint) wrap.appendChild(ce("div", { style: "font-size:10px;color:var(--gold-dim);margin-top:2px;", text: hint }));
      F[key] = function () {
        return rows.map(function (r) {
          return { to: parseInt(r._to.value, 10) || 0, type: r._type.value.trim(), subtype: r._subtype.value.trim() };
        }).filter(function (row) { return row.to > 0; });
      };
      return wrap;
    }

    // Equipment: one numeric item-id input per worn slot (v1 — no searchable
    // item picker; see Task 6 plan note).
    function equipmentField(slots, equipMap) {
      var wrap = ce("div", {});
      var rows = [];
      var equip = equipMap || {};
      for (var i = 0; i < slots.length; i += 2) {
        var rowSlots = slots.slice(i, i + 2);
        var cols = rowSlots.map(function (s) {
          var inp = ce("input", { type: "text", placeholder: "item id", list: "dl-mob-items" });
          inp.value = equip[s] || "";
          inp.addEventListener("input", markDirty);
          rows.push({ slot: s, input: inp });
          return labelWrap(s, inp);
        });
        wrap.appendChild(ce("div", { "class": "row" }, cols));
      }
      F.equipment = function () {
        var out = {};
        rows.forEach(function (r) {
          var v = parseInt(r.input.value, 10) || 0;
          if (v > 0) out[r.slot] = v;
        });
        return out;
      };
      return wrap;
    }

    // ---- Identity ----
    insp.appendChild(sectionTitle("Identity"));
    insp.appendChild(textField("Name", "name", detail.name));
    insp.appendChild(textAreaField("Description", "description", detail.description, "shown to players — wraps at ~78 chars"));
    // "(no zone)" is a summon-only / not-yet-placed template — a legal state
    // the server accepts (Filepath() routes it to the unzoned/ folder).
    var zoneOpts = [{ v: "", t: "(no zone)" }].concat(enums.zones || []);
    insp.appendChild(selectField("Zone", "zone", detail.zone, zoneOpts));
    var speciesOpts = Object.keys(enums.species || {}).map(function (id) { return { v: id, t: enums.species[id] }; });
    insp.appendChild(selectField("Species", "speciesId", detail.speciesId, speciesOpts, true));
    insp.appendChild(chipsField("Adjectives", "adjectives", detail.adjectives));
    insp.appendChild(chipsField("Groups", "groups", detail.groups, "", enums.groups || []));

    // ---- Stats & Combat ----
    insp.appendChild(sectionTitle("Stats & Combat"));
    insp.appendChild(numField("Stat pool", "statPool", detail.statPool));
    var st = detail.statTraining || {};
    var statRows = [];
    for (var si = 0; si < STAT_KEYS.length; si += 2) {
      var pair = STAT_KEYS.slice(si, si + 2);
      var statCols = pair.map(function (sk) {
        var inp = ce("input", { type: "number", step: "1" }); inp.value = st[sk] || 0;
        inp.addEventListener("input", markDirty);
        inp.addEventListener("blur", function () { if (this.value !== "") this.value = String(Math.round(parseFloat(this.value) || 0)); });
        statRows.push({ key: sk, input: inp });
        return labelWrap(cap(sk) + " training", inp);
      });
      insp.appendChild(ce("div", { "class": "row" }, statCols));
    }
    F.statTraining = function () {
      var out = {};
      statRows.forEach(function (r) { out[r.key] = parseInt(r.input.value, 10) || 0; });
      return out;
    };
    insp.appendChild(selectField("Archetype", "archetype", detail.archetype, enums.archetypes || [""]));
    insp.appendChild(checkField("Auto-aggro", "autoAggro", detail.autoAggro));
    insp.appendChild(selectField("AI profile", "aiProfile", detail.aiProfile, enums.aiProfiles || [""]));
    insp.appendChild(ce("div", { "class": "row" }, [
      numField("Special move %", "specialMoveChance", detail.specialMoveChance),
      numField("Activity level", "activityLevel", detail.activityLevel)]));
    insp.appendChild(ce("div", { "class": "row" }, [
      numField("Max wander", "maxWander", detail.maxWander),
      textField("Routine", "routine", detail.routine)]));
    insp.appendChild(chipsField("Routine links", "routineLinks", detail.routineLinks));
    insp.appendChild(chipsField("Hates", "hates", detail.hates));
    insp.appendChild(selectField("Submission policy", "submissionPolicy", detail.submissionPolicy, enums.submissionPolicies || [""]));
    insp.appendChild(textField("Surrender policy", "surrenderPolicy", detail.surrenderPolicy,
      'never | always | "auto-tap-below N"'));
    insp.appendChild(checkField("Pack flee immune", "packFleeImmune", detail.packFleeImmune));

    // ---- Equipment ----
    insp.appendChild(sectionTitle("Equipment"));
    insp.appendChild(equipmentField(enums.wornSlots || [], detail.equipment || {}));
    insp.appendChild(idRowsField("Carried items", "carriedItems", detail.carriedItems, "item ids the mob carries in inventory", "dl-mob-items"));

    // ---- Flavor ----
    insp.appendChild(sectionTitle("Flavor"));
    insp.appendChild(commandListField("Idle commands", "idleCommands", detail.idleCommands,
      "blank rows are intentional “do nothing” pool entries — don't delete them to tidy up"));
    insp.appendChild(commandListField("Combat commands", "combatCommands", detail.combatCommands));
    insp.appendChild(commandListField("Angry commands", "angryCommands", detail.angryCommands));

    // ---- Loot ----
    insp.appendChild(sectionTitle("Loot"));
    insp.appendChild(numField("Item drop chance %", "itemDropChance", detail.itemDropChance));
    insp.appendChild(idRowsField("Loot pool", "lootPool", detail.lootPool, "", "dl-mob-items"));
    insp.appendChild(numField("Gold", "gold", detail.gold));

    // ---- Advanced (collapsible) ----
    var H = {
      field: field, sectionTitle: sectionTitle, textField: textField, textAreaField: textAreaField,
      numField: numField, checkField: checkField, selectField: selectField, chipsField: chipsField,
      idRowsField: idRowsField, shopRowsField: shopRowsField, relRowsField: relRowsField
    };
    this.buildAdvancedSection(insp, detail, F, H);

    // ---- Save + delete ----
    var save = ce("button", { id: "mob-save", text: "Save Mob", disabled: "disabled" });
    save.addEventListener("click", function () { Panel.save(); });
    var del = ce("button", { "class": "mini rm", text: "Delete mob", "style": "width:100%;margin-top:6px;padding:6px;" });
    del.addEventListener("click", function () { Panel.del(); });
    insp.appendChild(ce("div", { "class": "save-row", style: "display:flex;align-items:center;" }, [save, ce("span", { style: "flex:1 1 auto;" }), del]));
    insp.appendChild(ce("div", { id: "mob-refs", "style": "color:var(--danger);font-size:11px;margin-top:6px;" }));

    this.setDirty(false);
  };

  Panel.buildAdvancedSection = function (insp, detail, F, H) {
    var self = this;
    var enums = detail.enums || {};
    var hasAdv = detail.scheduleId || detail.patrolId || (detail.shop && detail.shop.length) ||
      detail.crafter || (detail.relationships && detail.relationships.length) ||
      (detail.buffIds && detail.buffIds.length) || detail.scriptTag || detail.llmProfileJson ||
      detail.carryCapacity || detail.healthMax || detail.staminaMax || detail.nonCombatant;
    // Recompute open state only when a different mob is selected; preserve the
    // author's toggle across same-mob re-renders (e.g. the post-save re-Get).
    if (detail.mobId !== this._advMobId) { this.advancedOpen = !!hasAdv; this._advMobId = detail.mobId; }

    function headText() { return (self.advancedOpen ? "▾ " : "▸ ") + "Advanced"; }
    var head = ce("h3", { text: headText() });
    head.style.cursor = "pointer";
    var body = ce("div", {});
    body.style.display = this.advancedOpen ? "" : "none";
    head.addEventListener("click", function () {
      self.advancedOpen = !self.advancedOpen;
      body.style.display = self.advancedOpen ? "" : "none";
      head.textContent = headText();
    });
    insp.appendChild(head);
    insp.appendChild(body);

    // Commerce
    body.appendChild(H.sectionTitle("Commerce"));
    body.appendChild(H.shopRowsField("Shop inventory", "shop", detail.shop));
    body.appendChild(H.checkField("Buys general goods", "buysGeneral", detail.buysGeneral));
    body.appendChild(H.numField("Stock multiplier", "stockMultiplier", detail.stockMultiplier, "0.05"));
    body.appendChild(H.selectField("Craft support", "craftSupport", detail.craftSupport, [""].concat(enums.craftSupports || [])));
    body.appendChild(H.checkField("Crafter", "crafter", detail.crafter));
    body.appendChild(H.selectField("Crafter skill", "crafterSkill", detail.crafterSkill, [""].concat(enums.crafterSkills || [])));
    body.appendChild(H.chipsField("Crafter recipe ids", "crafterRecipeIds", detail.crafterRecipeIds));
    body.appendChild(H.idRowsField("Crafter restock materials", "crafterRestockMaterials", detail.crafterRestockMaterials, "", "dl-mob-items"));

    // Aliveness
    body.appendChild(H.sectionTitle("Aliveness"));
    body.appendChild(H.selectField("Schedule", "scheduleId", detail.scheduleId, [""].concat(enums.scheduleIds || [])));
    body.appendChild(H.selectField("Patrol", "patrolId", detail.patrolId, [""].concat(enums.patrolIds || [])));
    body.appendChild(H.relRowsField("Relationships", "relationships", detail.relationships));
    body.appendChild(H.chipsField("Knows facts", "knowsFacts", detail.knowsFacts));
    body.appendChild(ce("div", { "class": "row" }, [
      H.numField("Default disposition", "defaultDisposition", detail.defaultDisposition),
      H.numField("Fold anchor room", "foldAnchorRoom", detail.foldAnchorRoom)]));
    body.appendChild(H.numField("Storage chest room", "storageChestRoom", detail.storageChestRoom));

    // Hooks
    body.appendChild(H.sectionTitle("Hooks"));
    body.appendChild(H.textField("Script tag", "scriptTag", detail.scriptTag));
    body.appendChild(H.selectField("Behavior archetype", "behaviorArchetype", detail.behaviorArchetype, [""].concat(enums.behaviorArchetypes || [])));
    // 5d: jump into the behavior editor for this mob's AI.
    var btArchBtn = ce("button", { "class": "mini", text: "Edit archetype tree…" });
    btArchBtn.addEventListener("click", function () {
      var arch = Panel.fields && Panel.fields.behaviorArchetype ? Panel.fields.behaviorArchetype() : detail.behaviorArchetype;
      if (!arch) { toast("This mob has no behavior archetype set.", true); return; }
      if (window.Builder.setMode) window.Builder.setMode("behaviors");
      gmcp("Build.Behavior.Get", { kind: "archetype", name: arch });
    });
    var btMobBtn = ce("button", { "class": "mini", text: "Mob tree…", style: "margin-left:6px;" });
    btMobBtn.addEventListener("click", function () {
      if (window.Builder.setMode) window.Builder.setMode("behaviors");
      gmcp("Build.Behavior.Get", { kind: "mob", mobId: detail.mobId });
    });
    body.appendChild(ce("div", { style: "margin:4px 0 8px;" }, [btArchBtn, btMobBtn]));
    body.appendChild(H.idRowsField("Buff ids", "buffIds", detail.buffIds, "", "dl-mob-buffs"));
    body.appendChild(H.chipsField("Quest flags", "questFlags", detail.questFlags));
    body.appendChild(H.chipsField("Spawn mutations", "spawnMutations", detail.spawnMutations));
    body.appendChild(H.numField("Mutation chance %", "mutationChance", detail.mutationChance));
    body.appendChild(H.textAreaField("LLM profile (raw JSON)", "llmProfileJson", detail.llmProfileJson,
      "advanced — raw JSON; leave empty for none"));

    // Overrides & flags
    body.appendChild(H.sectionTitle("Overrides & flags"));
    body.appendChild(ce("div", { "class": "row" }, [
      H.numField("Carry capacity", "carryCapacity", detail.carryCapacity, "0.05"),
      H.numField("Health max", "healthMax", detail.healthMax)]));
    body.appendChild(H.numField("Stamina max", "staminaMax", detail.staminaMax));
    body.appendChild(H.textField("Corpse name", "corpseName", detail.corpseName));
    body.appendChild(H.textAreaField("Corpse description", "corpseDescription", detail.corpseDescription));
    body.appendChild(ce("div", { "class": "flags" }, [
      H.checkField("Hide equipment slots", "hideEquipmentSlots", detail.hideEquipmentSlots),
      H.checkField("Charm immune", "charmImmune", detail.charmImmune),
      H.checkField("Non-combatant", "nonCombatant", detail.nonCombatant),
      H.checkField("Player-attack immune", "playerAttackImmune", detail.playerAttackImmune)]));
  };

  Panel.setDirty = function (d) {
    this.dirty = d;
    var s = document.getElementById("mob-save");
    if (s) s.disabled = !d;
  };

  // ---- mutations ----
  // Any mobUpdateReq field the form doesn't expose a control for (e.g.
  // movePreferences) falls back to the value the server sent, so a save never
  // silently blanks a field the UI can't edit yet.
  Panel.gather = function () {
    var F = this.fields || {};
    var d = this.detail || {};
    function g(k, dflt) { return F[k] ? F[k]() : (d[k] !== undefined ? d[k] : dflt); }
    return {
      mobId: d.mobId,
      zone: g("zone", d.zone || ""),
      name: g("name", ""), description: g("description", ""),
      adjectives: g("adjectives", []), speciesId: g("speciesId", 0), gold: g("gold", 0),
      statTraining: g("statTraining", {}), equipment: g("equipment", {}),
      carriedItems: g("carriedItems", []), shop: g("shop", []),
      statPool: g("statPool", 0), archetype: g("archetype", ""), autoAggro: g("autoAggro", false),
      aiProfile: g("aiProfile", ""), specialMoveChance: g("specialMoveChance", 0),
      movePreferences: g("movePreferences", {}), activityLevel: g("activityLevel", 0),
      maxWander: g("maxWander", 0), routine: g("routine", ""), routineLinks: g("routineLinks", []),
      hates: g("hates", []), groups: g("groups", []), submissionPolicy: g("submissionPolicy", ""),
      surrenderPolicy: g("surrenderPolicy", ""), packFleeImmune: g("packFleeImmune", false),
      idleCommands: g("idleCommands", []), combatCommands: g("combatCommands", []), angryCommands: g("angryCommands", []),
      itemDropChance: g("itemDropChance", 0), lootPool: g("lootPool", []),
      buysGeneral: g("buysGeneral", false), stockMultiplier: g("stockMultiplier", 0),
      craftSupport: g("craftSupport", ""), crafter: g("crafter", false), crafterSkill: g("crafterSkill", ""),
      crafterRecipeIds: g("crafterRecipeIds", []), crafterRestockMaterials: g("crafterRestockMaterials", []),
      scheduleId: g("scheduleId", ""), patrolId: g("patrolId", ""),
      relationships: g("relationships", []), knowsFacts: g("knowsFacts", []),
      defaultDisposition: g("defaultDisposition", 0), foldAnchorRoom: g("foldAnchorRoom", 0), storageChestRoom: g("storageChestRoom", 0),
      scriptTag: g("scriptTag", ""), behaviorArchetype: g("behaviorArchetype", ""),
      buffIds: g("buffIds", []), questFlags: g("questFlags", []), spawnMutations: g("spawnMutations", []),
      mutationChance: g("mutationChance", 0), llmProfileJson: g("llmProfileJson", ""),
      carryCapacity: g("carryCapacity", 0), healthMax: g("healthMax", 0), staminaMax: g("staminaMax", 0),
      corpseName: g("corpseName", ""), corpseDescription: g("corpseDescription", ""),
      hideEquipmentSlots: g("hideEquipmentSlots", false), charmImmune: g("charmImmune", false),
      nonCombatant: g("nonCombatant", false), playerAttackImmune: g("playerAttackImmune", false)
    };
  };

  Panel.save = function () {
    if (!this.detail) return;
    var req = this.gather();
    if (!req.name) { toast("Name is required", true); return; }
    // Zone is intentionally NOT required — a mob template can be authored
    // with no home zone (a summon-only kind, or a not-yet-placed new mob).
    this.saving = true;
    gmcp("Build.Mob.Update", req);
  };
  Panel.del = function () {
    if (!this.detail) return;
    if (!window.confirm("Delete mob #" + this.detail.mobId + "?")) return;
    this.deleting = true;
    gmcp("Build.Mob.Delete", { mobId: this.detail.mobId });
  };

  // Build.Result routing from build.html (mode === "mobs").
  Panel.onResult = function (obj) {
    if (obj && obj.ok) {
      if (this.deleting) {
        this.deleting = false; this.detail = null; this.selectedId = 0; this.clear();
        toast("Mob deleted.", false);
        return;
      }
      if (this.saving) {
        this.saving = false; this.setDirty(false); toast("Saved.", false);
        // Reload the persisted spec so the form reflects exactly what was
        // stored (e.g. a name the server canonicalized).
        if (this.selectedId) gmcp("Build.Mob.Get", { mobId: this.selectedId });
        return;
      }
      if (this.creating) {
        this.creating = false;
        toast("Created.", false);
        if (obj.mobId) { this.selectedId = obj.mobId; gmcp("Build.Mob.Get", { mobId: obj.mobId }); }
        return;
      }
      if (this.spawning) {
        this.spawning = false;
        toast(obj.message || "Mob spawned.", false);
        return;
      }
    } else {
      this.saving = false; this.deleting = false; this.creating = false; this.spawning = false;
      if (obj && obj.mobRefs && obj.mobRefs.length) this.showDeleteRefs(obj.mobRefs);
      toast((obj && obj.error) || "Mob error", true);
    }
  };

  Panel.showDeleteRefs = function (refs) {
    var el = document.getElementById("mob-refs");
    if (!el) return;
    el.textContent = "Still referenced: " + refs.map(function (r) { return r.kind + " " + r.id; }).join(", ") + " — remove these first.";
  };

  // ---- inspector placeholder ----
  function clearMobInspector() {
    var insp = document.getElementById("inspector");
    if (!insp) return;
    insp.innerHTML = "";
    insp.appendChild(ce("h2", { text: "Mobs" }));
    insp.appendChild(ce("div", { "class": "empty", text: "Select a mob on the left, or + New Mob." }));
  }
  Panel.clear = function () {
    this.detail = null; this.fields = null; this.dirty = false; this.selectedId = 0;
    this.saving = false; this.deleting = false; this.creating = false;
    this._spawnUI = null; this.spawning = false;
    clearMobInspector();
  };

  window.Builder = window.Builder || {};
  window.Builder.MobsPanel = Panel;
})();
