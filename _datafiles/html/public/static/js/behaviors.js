"use strict";
/*
 * behaviors.js — the behavior-tree editor (admin web-building 5d), a sixth
 * mode of the /build page. Three tree families (archetypes / per-mob / room)
 * ride Build.Behavior.* kind-routed verbs. The tree itself is a RECURSIVE
 * collapsible node editor: child order IS evaluation order for selectors and
 * sequences, so children render as ordered lists with ↑/↓.
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
  function markDirty() { Panel.dirty = true; }
  function toInt(v) { var n = Math.round(parseFloat(v)); return isNaN(n) ? 0 : n; }

  var MAX_DEPTH = 12; // soft cap: warn, don't refuse
  var DECORATOR_PARAM = { cooldown: "rounds", repeat: "times", random: "percent", delay: "rounds", invert: null };

  var Panel = {
    data: { archetypes: [], mobTrees: [], roomTrees: [] },
    search: "",
    selected: null, // {kind, name|mobId|roomId, zone}
    enums: {},
    dirty: false,
    saving: false,
    deleting: false,
    mobRows: [],
    zoneRows: [],
    roomsByZone: {},
    _pendingRoomZone: "",
    _roomSelects: []
  };

  function textInput(val) {
    var i = ce("input", { type: "text" }); i.value = val == null ? "" : val;
    i.addEventListener("input", markDirty); return i;
  }
  function textArea(val) {
    var t = ce("textarea", {}); t.value = val || "";
    t.addEventListener("input", markDirty); return t;
  }
  function numInput(val) {
    var i = ce("input", { type: "text", style: "max-width:90px;" });
    i.value = (val || val === 0) && val !== "" ? String(val) : "";
    i.addEventListener("input", markDirty); return i;
  }
  function pick(val, dl) {
    var i = ce("input", { type: "text" });
    if (dl) i.setAttribute("list", dl);
    i.value = val || "";
    i.addEventListener("input", markDirty); return i;
  }

  // generic params key/value rows — the registries declare no schemas, so
  // check/action params are freeform (see docs/schemas/behavior.md).
  function paramRows(m) {
    var box = ce("div", {});
    function row(k, v) {
      var key = ce("input", { type: "text", placeholder: "param", style: "max-width:140px;" });
      key.value = k || "";
      var val = ce("input", { type: "text", placeholder: "value", style: "max-width:160px;" });
      val.value = v == null ? "" : String(v);
      key.addEventListener("input", markDirty); val.addEventListener("input", markDirty);
      var rm = ce("button", { "class": "mini rm", text: "×" });
      var wrap = ce("div", {}, [key, val, rm]);
      rm.addEventListener("click", function () { box.removeChild(wrap); markDirty(); });
      wrap._kv = function () {
        var kk = key.value.trim();
        if (!kk) return null;
        var vv = val.value;
        // numbers stay numbers on the wire (rounds: 3, chance: 25)
        return [kk, vv !== "" && !isNaN(parseFloat(vv)) && String(parseFloat(vv)) === vv.trim() ? parseFloat(vv) : vv];
      };
      box.insertBefore(wrap, addBtn);
    }
    var addBtn = ce("button", { "class": "mini", text: "+ param" });
    addBtn.addEventListener("click", function () { row("", ""); markDirty(); });
    box.appendChild(addBtn);
    if (m) for (var k in m) row(k, m[k]);
    return {
      el: box,
      get: function () {
        var out = {};
        Array.prototype.forEach.call(box.children, function (c) {
          if (c._kv) { var kv = c._kv(); if (kv) out[kv[0]] = kv[1]; }
        });
        return out;
      }
    };
  }

  // ---- the recursive node editor ---------------------------------------

  function nodeSummary(def) {
    var bits = [def.type || "?"];
    if (def.event) bits.push("on " + def.event);
    if (def.check) bits.push("check " + def.check);
    if (def.do) bits.push("do " + def.do);
    if (def.mod) bits.push("mod " + def.mod);
    return bits.join(" · ");
  }

  // nodeRow renders one node as a collapsible row into box; returns nothing —
  // the row's wrap._gather() produces the node object.
  function nodeRow(box, def, depth, startOpen) {
    def = def || { type: "action" };
    var current = { type: def.type || "action" };

    var body = ce("div", { style: "display:none;padding:4px 0 8px 12px;border-left:1px dotted var(--tooled);" });
    var sum = ce("span", { text: "▸ " + nodeSummary(def) });
    var head = ce("div", { "class": "irow", style: "cursor:pointer;" });
    head.appendChild(sum);
    var up = ce("button", { "class": "mini", text: "↑", title: "move up (evaluation order)" });
    var dn = ce("button", { "class": "mini", text: "↓", title: "move down" });
    up.addEventListener("click", function (ev) { ev.stopPropagation(); var w = head.parentNode; var prev = w.previousElementSibling; if (prev && prev._isRow) { w.parentNode.insertBefore(w, prev); markDirty(); } });
    dn.addEventListener("click", function (ev) { ev.stopPropagation(); var w = head.parentNode; var next = w.nextElementSibling; if (next && next._isRow) { w.parentNode.insertBefore(next, w); markDirty(); } });
    head.appendChild(up); head.appendChild(dn);
    var rm = ce("button", { "class": "mini rm", text: "✕", title: "remove node (and its subtree)" });
    head.appendChild(rm);
    var wrap = ce("div", {}, [head, body]);
    wrap._isRow = true;
    rm.addEventListener("click", function (ev) {
      ev.stopPropagation();
      if ((def.children || []).length || def.child) {
        if (!window.confirm("Remove this node AND everything nested under it?")) return;
      }
      wrap.parentNode.removeChild(wrap); markDirty();
    });
    var open = !!startOpen;
    function setOpen(o) {
      open = o;
      body.style.display = o ? "" : "none";
      sum.textContent = (o ? "▾ " : "▸ ") + nodeSummary(gatherShallow());
    }
    head.addEventListener("click", function () { setOpen(!open); });

    // ---- body ----
    var typeSel = ce("select", {});
    (Panel.enums.nodeTypes || ["selector", "sequence", "condition", "action", "decorator"]).forEach(function (tp) {
      var o = ce("option", { value: tp, text: tp });
      if ((def.type || "action") === tp) o.selected = true;
      typeSel.appendChild(o);
    });
    body.appendChild(field("Type", typeSel));

    var evSel = ce("select", {});
    evSel.appendChild(ce("option", { value: "", text: "(no event gate)" }));
    (Panel.enums.events || []).forEach(function (evName) {
      var o = ce("option", { value: evName, text: evName });
      if (def.event === evName) o.selected = true;
      evSel.appendChild(o);
    });
    evSel.addEventListener("change", markDirty);
    body.appendChild(field("Event gate (node only runs for this event)", evSel));

    var note = textInput(def.note);
    body.appendChild(field("Note (design rationale — survives saves; # comments do not)", note));

    // type-specific zone, rebuilt on type change
    var typed = ce("div", {});
    body.appendChild(typed);
    var readers = {};
    function renderTyped(tp) {
      typed.innerHTML = "";
      readers = {};
      if (tp === "condition") {
        var check = pick(def.check, "bt-cond-dl");
        typed.appendChild(field("Check", check, "registered condition name"));
        readers.check = check;
        var cp = paramRows(def.params);
        typed.appendChild(field("Params", cp.el));
        readers.params = cp;
      } else if (tp === "action") {
        var doInp = pick(def.do, "bt-act-dl");
        typed.appendChild(field("Do", doInp, "registered action name"));
        readers.do = doInp;
        var ap = paramRows(def.params);
        typed.appendChild(field("Params", ap.el));
        readers.params = ap;
      } else if (tp === "decorator") {
        var modSel = ce("select", {});
        (Panel.enums.decoratorMods || []).forEach(function (m) {
          var o = ce("option", { value: m.key, text: m.key + " — " + m.description });
          if (def.mod === m.key) o.selected = true;
          modSel.appendChild(o);
        });
        modSel.addEventListener("change", function () { renderModParam(); markDirty(); });
        typed.appendChild(field("Mod", modSel));
        readers.mod = modSel;
        var modParamHost = ce("div", {});
        typed.appendChild(modParamHost);
        function renderModParam() {
          modParamHost.innerHTML = "";
          var pkey = DECORATOR_PARAM[modSel.value];
          if (pkey) {
            var pv = numInput(def.params && def.params[pkey]);
            modParamHost.appendChild(field(pkey, pv));
            readers.modParamKey = pkey;
            readers.modParamVal = pv;
          } else {
            readers.modParamKey = null;
          }
        }
        renderModParam();
        typed.appendChild(ce("div", { style: "font-size:10px;color:var(--gold-dim);margin:4px 0 2px;", text: "Child (the decorated node):" }));
        var childBox = ce("div", {});
        typed.appendChild(childBox);
        if (def.child) nodeRow(childBox, def.child, depth + 1, false);
        var addChild = ce("button", { "class": "mini", text: "+ set child" });
        addChild.addEventListener("click", function () {
          if (childRows(childBox).length) { toast("A decorator has exactly one child — remove it first.", true); return; }
          nodeRow(childBox, { type: "action" }, depth + 1, true); markDirty();
        });
        typed.appendChild(addChild);
        readers.childBox = childBox;
      } else { // selector / sequence
        typed.appendChild(ce("div", { style: "font-weight:bold;color:var(--gold-dim);font-size:11px;margin:4px 0;", text: "Children — run in ORDER (" + (tp === "selector" ? "first success wins" : "all must succeed") + "):" }));
        var kidsBox = ce("div", {});
        typed.appendChild(kidsBox);
        if (depth >= MAX_DEPTH) {
          typed.appendChild(ce("div", { style: "color:#c90;font-size:11px;", text: "Deeply nested (" + depth + " levels) — consider restructuring." }));
        }
        (def.children || []).forEach(function (chDef) { nodeRow(kidsBox, chDef, depth + 1, false); });
        var addKid = ce("button", { "class": "mini", text: "+ child" });
        addKid.addEventListener("click", function () { nodeRow(kidsBox, { type: "action" }, depth + 1, true); markDirty(); });
        typed.appendChild(addKid);
        readers.kidsBox = kidsBox;
      }
    }
    renderTyped(current.type);
    typeSel.addEventListener("change", function () {
      // Re-anchor def to what's currently gathered so switching types keeps
      // shared fields; type-specific fields reset by design.
      def = gatherShallow();
      current.type = typeSel.value;
      renderTyped(current.type);
      markDirty();
    });

    function childRows(host) {
      var out = [];
      Array.prototype.forEach.call(host.children, function (c) { if (c._gather) out.push(c); });
      return out;
    }

    function gatherShallow() {
      var out = { type: typeSel.value };
      if (evSel.value) out.event = evSel.value;
      if (note.value.trim()) out.note = note.value.trim();
      if (readers.check) out.check = readers.check.value.trim();
      if (readers.do) out.do = readers.do.value.trim();
      if (readers.mod) out.mod = readers.mod.value;
      return out;
    }

    wrap._gather = function () {
      var out = gatherShallow();
      var tp = out.type;
      if (tp === "condition" || tp === "action") {
        var p = readers.params.get();
        if (Object.keys(p).length) out.params = p;
      } else if (tp === "decorator") {
        if (readers.modParamKey && readers.modParamVal) {
          var params = {};
          params[readers.modParamKey] = toInt(readers.modParamVal.value);
          out.params = params;
        }
        var rows = childRows(readers.childBox);
        if (rows.length) out.child = rows[0]._gather();
      } else {
        out.children = childRows(readers.kidsBox).map(function (r) { return r._gather(); });
      }
      return out;
    };
    wrap._refreshSummary = function () {
      sum.textContent = (open ? "▾ " : "▸ ") + nodeSummary(gatherShallow());
    };
    box.appendChild(wrap);
    if (startOpen) setOpen(true);
    return wrap;
  }

  // ---- list -------------------------------------------------------------

  Panel.render = function (payload) {
    this.data = payload || { archetypes: [], mobTrees: [], roomTrees: [] };
    var host = document.getElementById("behaviorlist");
    if (!host) return;
    host.innerHTML = "";

    var search = ce("input", { type: "text", placeholder: "search name or id" });
    search.value = this.search;
    search.addEventListener("input", function () { Panel.search = search.value; Panel.render(Panel.data); });
    host.appendChild(ce("div", { "class": "filters" }, [search]));
    var q = this.search.trim().toLowerCase();

    function section(title, addBtn) {
      var h = ce("div", { style: "font-weight:bold;color:var(--gold);margin:10px 0 4px;" , text: title });
      host.appendChild(h);
      if (addBtn) host.appendChild(addBtn);
    }

    // Archetypes
    var newArch = ce("button", { "class": "newitem", text: "+ New archetype" });
    newArch.addEventListener("click", function () {
      var name = window.prompt("New archetype — filesystem-safe name (e.g. pack_howler):", "");
      if (name === null || !name.trim()) return;
      gmcp("Build.Behavior.Create", { kind: "archetype", name: name.trim() });
    });
    section("Archetypes (shared)", newArch);
    (this.data.archetypes || []).forEach(function (a) {
      if (q && a.name.toLowerCase().indexOf(q) === -1) return;
      var row = ce("div", { "class": "irow" + (Panel.isSel("archetype", a.name) ? " sel" : "") });
      row.appendChild(document.createTextNode(a.name));
      row.appendChild(ce("span", { "class": "mzone", text: "  used by " + a.usedBy + " mob" + (a.usedBy === 1 ? "" : "s") }));
      if (a.hasHandComments) row.appendChild(ce("span", { "class": "mbadge", text: "# comments" }));
      row.addEventListener("click", function () { Panel.open({ kind: "archetype", name: a.name }); });
      host.appendChild(row);
    });

    // Mob trees
    var newMob = ce("button", { "class": "newitem", text: "+ New mob tree (specialize an archetype)" });
    newMob.addEventListener("click", function () { promptNewMobTree(); });
    section("Per-mob trees (override the mob's archetype)", newMob);
    host.appendChild(ce("div", { style: "font-size:10px;color:var(--gold-dim);margin:2px 0 6px;",
      text: "A per-mob tree is a private COPY of an archetype, forked for one mob to customize. To simply assign a mob to a shared archetype, use the mob form's Behavior archetype dropdown instead." }));
    (this.data.mobTrees || []).forEach(function (m) {
      if (q && String(m.mobId).indexOf(q) === -1 && (m.mobName || "").toLowerCase().indexOf(q) === -1) return;
      var row = ce("div", { "class": "irow" + (Panel.isSel("mob", m.mobId) ? " sel" : "") });
      row.appendChild(ce("span", { "class": "iid", text: "#" + m.mobId + " " }));
      row.appendChild(document.createTextNode(m.mobName || "(unknown mob)"));
      row.appendChild(ce("span", { "class": "mzone", text: "  " + m.zone }));
      row.addEventListener("click", function () { Panel.open({ kind: "mob", mobId: m.mobId }); });
      host.appendChild(row);
    });

    // Room trees
    var newRoom = ce("button", { "class": "newitem", text: "+ New room tree" });
    newRoom.addEventListener("click", function () { promptNewRoomTree(); });
    section("Room trees", newRoom);
    (this.data.roomTrees || []).forEach(function (r) {
      if (q && String(r.roomId).indexOf(q) === -1 && (r.title || "").toLowerCase().indexOf(q) === -1) return;
      var row = ce("div", { "class": "irow" + (Panel.isSel("room", r.roomId) ? " sel" : "") });
      row.appendChild(ce("span", { "class": "iid", text: "#" + r.roomId + " " }));
      row.appendChild(document.createTextNode(r.title || "(untitled)"));
      row.appendChild(ce("span", { "class": "mzone", text: "  " + r.zone }));
      row.addEventListener("click", function () { Panel.open({ kind: "room", roomId: r.roomId, zone: r.zone }); });
      host.appendChild(row);
    });
  };

  Panel.isSel = function (kind, key) {
    var s = this.selected;
    if (!s || s.kind !== kind) return false;
    return (kind === "archetype" && s.name === key) || (kind === "mob" && s.mobId === key) || (kind === "room" && s.roomId === key);
  };

  Panel.open = function (sel) {
    if (this.dirty && !window.confirm("Discard unsaved behavior changes?")) return;
    this.selected = sel;
    this.render(this.data);
    gmcp("Build.Behavior.Get", sel);
  };

  function promptNewMobTree() {
    var mobId = toInt(window.prompt("Mob id to specialize (see the Mobs tab):", ""));
    if (!mobId) return;
    var archNames = (Panel.data.archetypes || []).map(function (a) { return a.name; }).join(", ");
    var from = window.prompt("Seed from which archetype? (copy to start from; blank = empty tree)\n(" + archNames + ")", "");
    if (from === null) return;
    gmcp("Build.Behavior.Create", { kind: "mob", mobId: mobId, fromArchetype: from.trim() });
  }

  function promptNewRoomTree() {
    var roomId = toInt(window.prompt("Room id:", ""));
    if (!roomId) return;
    var zone = window.prompt("Zone name (as shown in the Zones tab):", "");
    if (zone === null || !zone.trim()) return;
    gmcp("Build.Behavior.Create", { kind: "room", roomId: roomId, zone: zone.trim() });
  }

  // ---- detail -----------------------------------------------------------

  Panel.renderDetail = function (obj) {
    Panel.enums = (obj && obj.enums) || Panel.enums || {};
    var insp = document.getElementById("inspector");
    insp.innerHTML = "";
    Panel.dirty = false;

    var title = "";
    if (obj.kind === "archetype") title = "Archetype: " + obj.name;
    else if (obj.kind === "mob") title = "Mob tree: #" + obj.mobId + " " + (obj.mobName || "");
    else title = "Room tree: #" + obj.roomId + " (" + (obj.zone || "") + ")";
    insp.appendChild(ce("h2", { text: title }));

    if (!obj.found) {
      insp.appendChild(ce("div", { "class": "empty", text: "No behavior file found." }));
      return;
    }

    // datalists for check/do pickers
    var condDl = ce("datalist", { id: "bt-cond-dl" });
    (Panel.enums.conditions || []).forEach(function (c) { condDl.appendChild(ce("option", { value: c })); });
    insp.appendChild(condDl);
    var actDl = ce("datalist", { id: "bt-act-dl" });
    (Panel.enums.actions || []).forEach(function (a) { actDl.appendChild(ce("option", { value: a })); });
    insp.appendChild(actDl);

    if (obj.hasHandComments) {
      insp.appendChild(ce("div", { style: "border:1px solid #c90;color:#c90;font-size:12px;padding:6px 10px;border-radius:4px;margin:6px 0;",
        text: "This file carries hand-written # comments. The FIRST editor save drops them all (a marshal cannot keep comments) — move anything worth keeping into the note/notes fields before saving." }));
    }
    insp.appendChild(ce("div", { style: "font-size:11px;color:var(--gold-dim);margin:2px 0 8px;",
      text: "Saves are live immediately — the engine hot-reloads this tree; no reboot. Node order IS evaluation order." }));

    insp.appendChild(ce("div", { id: "bt-warnings", style: "color:#c90;font-size:12px;white-space:pre-wrap;" }));
    insp.appendChild(ce("div", { id: "bt-errors", style: "color:var(--danger);font-size:12px;white-space:pre-wrap;" }));

    var f = obj.file || {};
    var notes = textArea(f.notes);
    insp.appendChild(field("Notes (file-level design rationale)", notes));

    // archetype extras
    var gw = null, dg = null;
    if (obj.kind === "archetype") {
      insp.appendChild(sectionTitle("Goal weights (goal type → multiplier)"));
      gw = kvRows(f.goal_weights, "goal type", "multiplier");
      insp.appendChild(gw.el);
      insp.appendChild(sectionTitle("Default goals"));
      dg = goalRows(f.default_goals);
      insp.appendChild(dg.el);
      if ((obj.usedBy || []).length) {
        insp.appendChild(sectionTitle("Used by " + obj.usedBy.length + " mob(s)"));
        obj.usedBy.forEach(function (u) {
          insp.appendChild(ce("div", { style: "font-size:11px;color:var(--gold-dim);", text: u }));
        });
      }
    }

    insp.appendChild(sectionTitle("Tree"));
    var rootBox = ce("div", {});
    insp.appendChild(rootBox);
    nodeRow(rootBox, f.tree || { type: "selector", children: [] }, 0, true);

    var save = ce("button", { id: "bt-save", text: "Save behavior" });
    save.addEventListener("click", function () {
      Panel.saving = true;
      var rows = [];
      Array.prototype.forEach.call(rootBox.children, function (c) { if (c._gather) rows.push(c); });
      if (!rows.length) { toast("The tree needs a root node.", true); return; }
      var file = { notes: notes.value, tree: rows[0]._gather() };
      if (gw) file.goal_weights = gw.getFloats();
      if (dg) file.default_goals = dg.get();
      var req = { kind: obj.kind, file: file };
      if (obj.kind === "archetype") req.name = obj.name;
      if (obj.kind === "mob") req.mobId = obj.mobId;
      if (obj.kind === "room") { req.roomId = obj.roomId; req.zone = obj.zone; }
      gmcp("Build.Behavior.Update", req);
    });
    var del = ce("button", { "class": "mini rm", text: "Delete behavior", style: "margin-left:12px;" });
    del.addEventListener("click", function () {
      var msg = obj.kind === "archetype"
        ? "Delete archetype \"" + obj.name + "\"? Anything referencing it blocks the delete and will be listed."
        : obj.kind === "mob"
          ? "Delete this mob tree? The mob falls back to its archetype."
          : "Delete this room tree? The room loses its scripted behavior.";
      if (!window.confirm(msg)) return;
      Panel.deleting = true;
      var req = { kind: obj.kind, name: obj.name, mobId: obj.mobId, roomId: obj.roomId, zone: obj.zone };
      gmcp("Build.Behavior.Delete", req);
    });
    insp.appendChild(ce("div", { "class": "save-row", style: "margin-top:12px;display:flex;align-items:center;" },
      [save, ce("span", { style: "flex:1 1 auto;" }), del]));
  };

  function kvRows(m, kPlaceholder, vPlaceholder) {
    var box = ce("div", {});
    function row(k, v) {
      var key = ce("input", { type: "text", placeholder: kPlaceholder, style: "max-width:160px;" });
      key.value = k || "";
      var val = ce("input", { type: "text", placeholder: vPlaceholder, style: "max-width:90px;" });
      val.value = v == null ? "" : String(v);
      key.addEventListener("input", markDirty); val.addEventListener("input", markDirty);
      var rm = ce("button", { "class": "mini rm", text: "×" });
      var wrap = ce("div", {}, [key, val, rm]);
      rm.addEventListener("click", function () { box.removeChild(wrap); markDirty(); });
      wrap._kv = function () { return key.value.trim() ? [key.value.trim(), parseFloat(val.value) || 0] : null; };
      box.insertBefore(wrap, addBtn);
    }
    var addBtn = ce("button", { "class": "mini", text: "+ add" });
    addBtn.addEventListener("click", function () { row("", ""); markDirty(); });
    box.appendChild(addBtn);
    if (m) for (var k in m) row(k, m[k]);
    return {
      el: box,
      getFloats: function () {
        var out = {};
        Array.prototype.forEach.call(box.children, function (c) {
          if (c._kv) { var kv = c._kv(); if (kv) out[kv[0]] = kv[1]; }
        });
        return out;
      }
    };
  }

  function goalRows(goals) {
    var box = ce("div", {});
    function row(g) {
      g = g || {};
      var tp = ce("input", { type: "text", placeholder: "goal type", style: "max-width:160px;" });
      tp.value = g.type || "";
      var pr = ce("input", { type: "text", placeholder: "priority", style: "max-width:70px;" });
      pr.value = g.priority ? String(g.priority) : "";
      tp.addEventListener("input", markDirty); pr.addEventListener("input", markDirty);
      var params = paramRows(g.params);
      var rm = ce("button", { "class": "mini rm", text: "×" });
      var wrap = ce("div", { style: "border:1px solid var(--tooled);border-radius:4px;padding:4px;margin:3px 0;" },
        [ce("div", {}, [tp, pr, rm]), params.el]);
      rm.addEventListener("click", function () { box.removeChild(wrap); markDirty(); });
      wrap._goal = function () {
        if (!tp.value.trim()) return null;
        var out = { type: tp.value.trim(), priority: toInt(pr.value) };
        var p = params.get();
        if (Object.keys(p).length) out.params = p;
        return out;
      };
      box.insertBefore(wrap, addBtn);
    }
    var addBtn = ce("button", { "class": "mini", text: "+ goal" });
    addBtn.addEventListener("click", function () { row(null); markDirty(); });
    box.appendChild(addBtn);
    (goals || []).forEach(row);
    return {
      el: box,
      get: function () {
        var out = [];
        Array.prototype.forEach.call(box.children, function (c) {
          if (c._goal) { var g = c._goal(); if (g) out.push(g); }
        });
        return out;
      }
    };
  }

  // ---- GMCP feeds -------------------------------------------------------

  Panel.onResult = function (res) {
    var errEl = document.getElementById("bt-errors");
    var warnEl = document.getElementById("bt-warnings");
    if (res && res.ok) {
      if (Panel.deleting) { Panel.deleting = false; Panel.selected = null; toast("Behavior deleted.", false); gmcp("Build.Behavior.List", {}); var insp = document.getElementById("inspector"); if (insp) insp.innerHTML = ""; return; }
      Panel.saving = false; Panel.dirty = false;
      toast("Behavior saved — live immediately, no reboot.", false);
      if (warnEl) warnEl.textContent = (res.warnings && res.warnings.length)
        ? ("Warnings:\n" + res.warnings.join("\n")) : "";
      if (errEl) errEl.textContent = "";
      gmcp("Build.Behavior.List", {});
      return;
    }
    Panel.saving = false; Panel.deleting = false;
    var msg = (res && res.error) || "Behavior error";
    if (res && res.behaviorRefs && res.behaviorRefs.length) {
      msg = (res.error || "Still referenced:") ;
    }
    if (errEl) errEl.textContent = msg;
    toast("Behavior refused — see the errors above the form.", true);
  };

  window.Builder = window.Builder || {};
  window.Builder.BehaviorsPanel = Panel;
})();
