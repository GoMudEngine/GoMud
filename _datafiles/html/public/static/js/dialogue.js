"use strict";
/*
 * dialogue.js — the dialogue editor (admin web-building 5b), a dedicated
 * panel opened from the mob editor's Dialogue… button. Consumes
 * Build.Dialogue GMCP (file + enums) and drives Build.Dialogue.Update/
 * Create/Delete. The server refuses SOP violations naming each offending
 * node/pattern and returns non-blocking warnings on success; this panel
 * renders both. Field names on the wire mirror the YAML names (the dialogue
 * types carry matching json tags).
 *
 * The tree renders as an ORDERED list with up/down controls, not a graph:
 * matching walks tree.nodes in file order, so order IS semantics — grant
 * nodes must come first, and the UI keeps that property visible.
 */
(function () {
  function ce(tag, attrs, kids) {
    var e = document.createElement(tag);
    if (attrs) for (var k in attrs) {
      if (k === "text") e.textContent = attrs[k];
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

  var Panel = {
    active: false,
    mobId: 0,
    zone: "",
    enums: {},
    dirty: false,
    saving: false,
    deleting: false,
  };

  Panel.open = function (mobId, zone) {
    Panel.active = true;
    Panel.mobId = mobId;
    Panel.zone = zone;
    gmcp("Build.Dialogue.Get", { mobId: mobId, zone: zone });
  };

  Panel.close = function () {
    Panel.active = false;
    Panel.dirty = false;
    // Restore the mob form by re-fetching it.
    gmcp("Build.Mob.Get", { mobId: Panel.mobId });
  };

  function markDirty() { Panel.dirty = true; var b = document.getElementById("dlg-save"); if (b) b.disabled = false; }

  // ---- small reusable editors ------------------------------------------

  // chips: a growable list of single-token text inputs with a shared datalist.
  function chips(vals, datalistId) {
    var box = ce("div", { "class": "chips" });
    function row(v) {
      var i = ce("input", { type: "text", style: "width:130px;margin:2px;" });
      if (datalistId) i.setAttribute("list", datalistId);
      i.value = v || "";
      i.addEventListener("input", markDirty);
      var rm = ce("button", { "class": "mini rm", text: "✕" });
      var r = ce("span", {}, [i, rm]);
      rm.addEventListener("click", function () { box.removeChild(r); markDirty(); });
      r._i = i;
      return r;
    }
    (vals || []).forEach(function (v) { box.appendChild(row(v)); });
    var add = ce("button", { "class": "mini", text: "+" });
    add.addEventListener("click", function () { box.insertBefore(row(""), add); markDirty(); });
    box.appendChild(add);
    return {
      el: box,
      get: function () {
        var out = [];
        Array.prototype.forEach.call(box.children, function (r) {
          if (r._i && r._i.value.trim()) out.push(r._i.value.trim());
        });
        return out;
      }
    };
  }

  // lines: a growable list of full-width text inputs (responses etc.).
  function lines(vals) {
    var box = ce("div", {});
    function row(v) {
      var i = ce("input", { type: "text", style: "width:92%;margin:2px 0;" });
      i.value = v || "";
      i.addEventListener("input", markDirty);
      var rm = ce("button", { "class": "mini rm", text: "✕" });
      var r = ce("div", {}, [i, rm]);
      rm.addEventListener("click", function () { box.removeChild(r); markDirty(); });
      r._i = i;
      return r;
    }
    (vals || []).forEach(function (v) { box.appendChild(row(v)); });
    var add = ce("button", { "class": "mini", text: "+ line" });
    add.addEventListener("click", function () { box.insertBefore(row(""), add); markDirty(); });
    box.appendChild(add);
    return {
      el: box,
      get: function () {
        var out = [];
        Array.prototype.forEach.call(box.children, function (r) {
          if (r._i && r._i.value.trim()) out.push(r._i.value.trim());
        });
        return out;
      }
    };
  }

  function textArea(v) {
    var t = ce("textarea", { style: "width:95%;min-height:52px;" });
    t.value = v || "";
    t.addEventListener("input", markDirty);
    return t;
  }

  function numInput(v, datalistId) {
    var i = ce("input", { type: "number", step: "1", style: "width:110px;" });
    if (datalistId) i.setAttribute("list", datalistId);
    i.value = v ? String(v) : "";
    i.addEventListener("input", markDirty);
    return i;
  }

  function moodSelect(v) {
    var s = ce("select", {});
    s.appendChild(ce("option", { value: "", text: "(none)" }));
    (Panel.enums.moods || []).forEach(function (m) {
      var o = ce("option", { value: m, text: m });
      if (m === v) o.selected = true;
      s.appendChild(o);
    });
    s.addEventListener("change", markDirty);
    return s;
  }

  // kv rows for quest-flag gates: key + value with datalist assist.
  function flagRows(m) {
    var box = ce("div", {});
    function row(k, v) {
      var ki = ce("input", { type: "text", placeholder: "flag key", style: "width:130px;" });
      ki.setAttribute("list", "dlg-flagkey-dl");
      ki.value = k || "";
      var vi = ce("input", { type: "text", placeholder: "value", style: "width:110px;" });
      vi.value = v || "";
      ki.addEventListener("input", markDirty); vi.addEventListener("input", markDirty);
      var rm = ce("button", { "class": "mini rm", text: "✕" });
      var r = ce("div", {}, [ki, vi, rm]);
      rm.addEventListener("click", function () { box.removeChild(r); markDirty(); });
      r._k = ki; r._v = vi;
      return r;
    }
    Object.keys(m || {}).forEach(function (k) { box.appendChild(row(k, m[k])); });
    var add = ce("button", { "class": "mini", text: "+ flag" });
    add.addEventListener("click", function () { box.insertBefore(row("", ""), add); markDirty(); });
    box.appendChild(add);
    return {
      el: box,
      get: function () {
        var out = {};
        Array.prototype.forEach.call(box.children, function (r) {
          if (r._k && r._k.value.trim()) out[r._k.value.trim()] = r._v.value;
        });
        return Object.keys(out).length ? out : null;
      }
    };
  }

  function repRows(vals) {
    var box = ce("div", {});
    function row(f, d) {
      var fi = ce("input", { type: "text", placeholder: "faction", style: "width:130px;" });
      fi.value = f || "";
      var di = ce("input", { type: "number", step: "1", placeholder: "delta", style: "width:70px;" });
      di.value = (d || d === 0) ? String(d) : "";
      fi.addEventListener("input", markDirty); di.addEventListener("input", markDirty);
      var rm = ce("button", { "class": "mini rm", text: "✕" });
      var r = ce("div", {}, [fi, di, rm]);
      rm.addEventListener("click", function () { box.removeChild(r); markDirty(); });
      r._f = fi; r._d = di;
      return r;
    }
    (vals || []).forEach(function (b) { box.appendChild(row(b.faction, b.delta)); });
    var add = ce("button", { "class": "mini", text: "+ rep" });
    add.addEventListener("click", function () { box.insertBefore(row("", ""), add); markDirty(); });
    box.appendChild(add);
    return {
      el: box,
      get: function () {
        var out = [];
        Array.prototype.forEach.call(box.children, function (r) {
          if (r._f && r._f.value.trim()) out.push({ faction: r._f.value.trim(), delta: Math.round(parseFloat(r._d.value) || 0) });
        });
        return out.length ? out : null;
      }
    };
  }

  // ---- the shared quest-gate drawer ------------------------------------

  function gateDrawer(g) {
    g = g || {};
    var body = ce("div", { style: "display:none;padding:6px 0 2px 10px;border-left:2px solid var(--tooled);" });
    var head = ce("button", { "class": "mini", text: "▸ Quest gating & effects" });
    head.addEventListener("click", function () {
      var open = body.style.display !== "none";
      body.style.display = open ? "none" : "";
      head.textContent = (open ? "▸" : "▾") + " Quest gating & effects";
    });

    var grants = ce("input", { type: "text", placeholder: "e.g. 10-start" });
    grants.setAttribute("list", "dlg-token-dl");
    grants.value = g.grantsQuest || "";
    grants.addEventListener("input", markDirty);
    body.appendChild(field("Grants quest", grants, "must also appear in questExcluded along with its -end token"));

    var qReq = chips(g.questRequired, "dlg-token-dl");
    body.appendChild(field("Quest required", qReq.el));
    var qExc = chips(g.questExcluded, "dlg-token-dl");
    body.appendChild(field("Quest excluded", qExc.el));

    var fReq = flagRows(g.questFlagRequired);
    body.appendChild(field("Flag required", fReq.el));
    var fExc = flagRows(g.questFlagExcluded);
    body.appendChild(field("Flag excluded", fExc.el));

    var sfKey = ce("input", { type: "text", placeholder: "key" });
    sfKey.setAttribute("list", "dlg-flagkey-dl");
    var sfVal = ce("input", { type: "text", placeholder: "value" });
    if (g.setsQuestFlag) { sfKey.value = g.setsQuestFlag.key || ""; sfVal.value = g.setsQuestFlag.value || ""; }
    sfKey.addEventListener("input", markDirty); sfVal.addEventListener("input", markDirty);
    body.appendChild(field("Sets quest flag", ce("span", {}, [sfKey, sfVal])));

    var reqItem = numInput(g.requiresItem, "dlg-item-dl");
    body.appendChild(field("Requires item", reqItem));
    var givItem = numInput(g.givesItem, "dlg-item-dl");
    body.appendChild(field("Gives item", givItem));
    var gold = numInput(g.givesGold);
    body.appendChild(field("Gives gold", gold));
    var mw = numInput(g.masterworkRequired);
    body.appendChild(field("Masterwork required", mw));
    var rep = repRows(g.bumpsRep);
    body.appendChild(field("Bumps rep", rep.el));

    return {
      el: ce("div", {}, [head, body]),
      gather: function (into) {
        into.grantsQuest = grants.value.trim();
        into.questRequired = qReq.get();
        into.questExcluded = qExc.get();
        into.questFlagRequired = fReq.get();
        into.questFlagExcluded = fExc.get();
        if (sfKey.value.trim()) into.setsQuestFlag = { key: sfKey.value.trim(), value: sfVal.value };
        into.requiresItem = Math.round(parseFloat(reqItem.value) || 0);
        into.givesItem = Math.round(parseFloat(givItem.value) || 0);
        into.givesGold = Math.round(parseFloat(gold.value) || 0);
        into.masterworkRequired = Math.round(parseFloat(mw.value) || 0);
        into.bumpsRep = rep.get();
        return into;
      }
    };
  }

  // ---- collapsible row shell -------------------------------------------

  function collapsible(box, summaryFn, buildBody, ordered) {
    var body = ce("div", { style: "display:none;padding:4px 0 8px 10px;" });
    var sum = ce("span", { style: "cursor:pointer;", text: summaryFn() });
    var head = ce("div", { "class": "irow" });
    head.appendChild(sum);
    if (ordered) {
      var up = ce("button", { "class": "mini", text: "↑", title: "move up" });
      var dn = ce("button", { "class": "mini", text: "↓", title: "move down" });
      up.addEventListener("click", function (ev) { ev.stopPropagation(); var w = head.parentNode; var prev = w.previousElementSibling; if (prev && prev._isRow) { w.parentNode.insertBefore(w, prev); markDirty(); } });
      dn.addEventListener("click", function (ev) { ev.stopPropagation(); var w = head.parentNode; var next = w.nextElementSibling; if (next && next._isRow) { w.parentNode.insertBefore(next, w); markDirty(); } });
      head.appendChild(up); head.appendChild(dn);
    }
    var rm = ce("button", { "class": "mini rm", text: "✕", title: "remove" });
    head.appendChild(rm);
    var wrap = ce("div", {}, [head, body]);
    wrap._isRow = true;
    rm.addEventListener("click", function (ev) { ev.stopPropagation(); wrap.parentNode.removeChild(wrap); markDirty(); });
    sum.addEventListener("click", function () {
      var open = body.style.display !== "none";
      body.style.display = open ? "none" : "";
      sum.textContent = summaryFn();
    });
    var built = buildBody(body);
    wrap._gather = built.gather;
    wrap._refreshSummary = function () { sum.textContent = summaryFn(); };
    box.appendChild(wrap);
    return wrap;
  }

  // ---- render ----------------------------------------------------------

  Panel.renderDetail = function (obj) {
    if (!Panel.active) return;
    Panel.enums = (obj && obj.enums) || {};
    var insp = document.getElementById("inspector");
    insp.innerHTML = "";
    Panel.dirty = false;

    var head = ce("div", {});
    head.appendChild(ce("h2", { text: "Dialogue: mob " + Panel.mobId + " (" + Panel.zone + ")" }));
    var closeBtn = ce("button", { "class": "mini", text: "◂ Back to mob" });
    closeBtn.addEventListener("click", function () {
      if (Panel.dirty && !window.confirm("Discard unsaved dialogue changes?")) return;
      Panel.close();
    });
    head.appendChild(closeBtn);
    insp.appendChild(head);

    // datalists shared by every picker in the panel
    var tokenDL = ce("datalist", { id: "dlg-token-dl" });
    (Panel.enums.questTokens || []).forEach(function (qt) {
      tokenDL.appendChild(ce("option", { value: qt.token, text: qt.questName }));
    });
    insp.appendChild(tokenDL);
    var flagDL = ce("datalist", { id: "dlg-flagkey-dl" });
    var seenKeys = {};
    (Panel.enums.questFlags || []).forEach(function (qf) {
      Object.keys(qf.flags || {}).forEach(function (k) {
        if (!seenKeys[k]) { seenKeys[k] = true; flagDL.appendChild(ce("option", { value: k, text: qf.questName + ": " + (qf.flags[k] || []).join("/") })); }
      });
    });
    insp.appendChild(flagDL);
    var itemDL = ce("datalist", { id: "dlg-item-dl" });
    ((window.Builder && window.Builder.itemRows) || []).forEach(function (r) {
      itemDL.appendChild(ce("option", { value: String(r.id), text: r.name || "" }));
    });
    insp.appendChild(itemDL);

    if (!obj || !obj.found) {
      insp.appendChild(ce("div", { "class": "empty", text: "This mob has no dialogue file. Talking to it does nothing." }));
      var create = ce("button", { text: "Create dialogue file" });
      create.addEventListener("click", function () {
        gmcp("Build.Dialogue.Create", { mobId: Panel.mobId, zone: Panel.zone });
      });
      insp.appendChild(create);
      return;
    }

    var f = obj.file || {};

    // persistent warnings area (filled by onResult)
    insp.appendChild(ce("div", { id: "dlg-warnings", style: "color:#c90;font-size:12px;white-space:pre-wrap;" }));
    insp.appendChild(ce("div", { id: "dlg-errors", style: "color:var(--danger);font-size:12px;white-space:pre-wrap;" }));

    // ---- identity & mood ----
    insp.appendChild(sectionTitle("Identity & mood"));
    var mood = moodSelect(f.defaultMood);
    insp.appendChild(field("Default mood", mood));
    var expiry = ce("input", { type: "text" });
    expiry.value = (f.memory && f.memory.expiryPeriod) || "";
    expiry.addEventListener("input", markDirty);
    insp.appendChild(field("Memory expiry", expiry,
      "leave EMPTY except for deliberately timed quests — expiring memory can brick quest chains"));

    // ---- greetings ----
    insp.appendChild(sectionTitle("Greetings (on player arrival)"));
    var greetBox = ce("div", {});
    insp.appendChild(greetBox);
    function greetingRow(g) {
      g = g || {};
      collapsible(greetBox, function () { return "greeting: " + ((g.text || "(new)").slice(0, 60)); }, function (body) {
        var t = textArea(g.text);
        body.appendChild(field("Text", t));
        var m = chips(g.moods, "dlg-mood-dl");
        body.appendChild(field("Moods (empty = any)", m.el));
        return { gather: function () { return { text: t.value, moods: m.get() }; } };
      });
    }
    (f.greetings || []).forEach(greetingRow);
    var addGreet = ce("button", { "class": "mini", text: "+ greeting" });
    addGreet.addEventListener("click", function () { greetingRow(null); markDirty(); });
    insp.appendChild(addGreet);
    var moodDL = ce("datalist", { id: "dlg-mood-dl" });
    (Panel.enums.moods || []).forEach(function (m) { moodDL.appendChild(ce("option", { value: m })); });
    insp.appendChild(moodDL);

    // ---- patterns ----
    insp.appendChild(sectionTitle("Patterns (keyword responses)"));
    var patBox = ce("div", {});
    insp.appendChild(patBox);
    function patternRow(p) {
      p = p || {};
      collapsible(patBox, function () { return "pattern: " + ((p.keywords || []).join(", ") || "(new)"); }, function (body) {
        var kw = chips(p.keywords);
        body.appendChild(field("Keywords", kw.el, "empty-string keyword = catch-all"));
        var mo = chips(p.moods, "dlg-mood-dl");
        body.appendChild(field("Moods (empty = any)", mo.el));
        var resp = lines(p.responses);
        body.appendChild(field("Responses", resp.el));
        var mc = moodSelect(p.moodChange);
        body.appendChild(field("Mood change", mc));
        var gd = gateDrawer(p);
        body.appendChild(gd.el);
        return { gather: function () {
          return gd.gather({ keywords: kw.get(), moods: mo.get(), responses: resp.get(), moodChange: mc.value });
        } };
      });
    }
    (f.patterns || []).forEach(patternRow);
    var addPat = ce("button", { "class": "mini", text: "+ pattern" });
    addPat.addEventListener("click", function () { patternRow(null); markDirty(); });
    insp.appendChild(addPat);

    // ---- tree ----
    insp.appendChild(sectionTitle("Conversation tree"));
    var tree = f.tree || {};
    var root = tree.root || {};
    var rootText = textArea(root.text);
    insp.appendChild(field("Root text (the talk opener)", rootText));
    var rootHints = textArea(root.hints);
    insp.appendChild(field("Root hints (narrator voice: \"You could ask…\")", rootHints));

    insp.appendChild(ce("div", { style: "font-size:11px;color:var(--gold-dim);margin:4px 0;",
      text: "Root variants: quest-gated replacements for the root, first match wins." }));
    var varBox = ce("div", {});
    insp.appendChild(varBox);
    function variantRow(v) {
      v = v || {};
      collapsible(varBox, function () { return "variant: " + ((v.text || "(new)").slice(0, 50)); }, function (body) {
        var t = textArea(v.text);
        body.appendChild(field("Text", t));
        var h = textArea(v.hints);
        body.appendChild(field("Hints", h));
        var gd = gateDrawer(v);
        body.appendChild(gd.el);
        return { gather: function () { return gd.gather({ text: t.value, hints: h.value }); } };
      }, true);
    }
    (root.variants || []).forEach(variantRow);
    var addVar = ce("button", { "class": "mini", text: "+ variant" });
    addVar.addEventListener("click", function () { variantRow(null); markDirty(); });
    insp.appendChild(addVar);

    insp.appendChild(ce("div", { style: "font-weight:bold;color:var(--gold-dim);font-size:11px;margin-top:8px;",
      text: "Nodes match in ORDER — quest-grant nodes must come first." }));
    var nodeBox = ce("div", {});
    insp.appendChild(nodeBox);
    var nodeDL = ce("datalist", { id: "dlg-node-dl" });
    ((tree.nodes) || []).forEach(function (n) { nodeDL.appendChild(ce("option", { value: n.id })); });
    insp.appendChild(nodeDL);
    function nodeRow(n) {
      n = n || {};
      collapsible(nodeBox, function () {
        var id = n.id || "(new)";
        return "node " + id + (n.grantsQuest ? " [grants " + n.grantsQuest + "]" : "") + " · " + ((n.triggers || []).join(","));
      }, function (body) {
        var id = ce("input", { type: "text" });
        id.value = n.id || "";
        id.addEventListener("input", markDirty);
        body.appendChild(field("Node id", id));
        var trig = chips(n.triggers);
        body.appendChild(field("Triggers", trig.el, "each must be discoverable in a hint, text, or the root — the save warns per undiscoverable trigger"));
        var req = chips(n.requires, "dlg-node-dl");
        body.appendChild(field("Requires (node ids)", req.el));
        var unl = chips(n.unlocks, "dlg-node-dl");
        body.appendChild(field("Unlocks (node ids)", unl.el));
        var t = textArea(n.text);
        body.appendChild(field("Text (NPC first person)", t));
        var h = textArea(n.hints);
        body.appendChild(field("Hints (narrator voice)", h));
        var mc = moodSelect(n.moodChange);
        body.appendChild(field("Mood change", mc));
        var gd = gateDrawer(n);
        body.appendChild(gd.el);
        return { gather: function () {
          return gd.gather({ id: id.value.trim(), triggers: trig.get(), requires: req.get(),
            unlocks: unl.get(), text: t.value, hints: h.value, moodChange: mc.value });
        } };
      }, true);
    }
    ((tree.nodes) || []).forEach(nodeRow);
    var addNode = ce("button", { "class": "mini", text: "+ node" });
    addNode.addEventListener("click", function () { nodeRow(null); markDirty(); });
    insp.appendChild(addNode);

    // ---- save / delete ----
    var save = ce("button", { id: "dlg-save", text: "Save dialogue" });
    save.addEventListener("click", function () {
      Panel.saving = true;
      gmcp("Build.Dialogue.Update", gatherFile());
    });
    var del = ce("button", { "class": "mini rm", text: "Delete dialogue", style: "margin-left:12px;" });
    del.addEventListener("click", function () {
      if (!window.confirm("Delete this mob's dialogue file? The NPC will be MUTE until a new one is created.")) return;
      Panel.deleting = true;
      gmcp("Build.Dialogue.Delete", { mobId: Panel.mobId, zone: Panel.zone });
    });
    insp.appendChild(ce("div", { "class": "save-row", style: "margin-top:12px;" }, [save, del]));

    function gatherFile() {
      var greets = [];
      Array.prototype.forEach.call(greetBox.children, function (r) { if (r._gather) { var g = r._gather(); if (g.text.trim()) greets.push(g); } });
      var pats = [];
      Array.prototype.forEach.call(patBox.children, function (r) { if (r._gather) pats.push(r._gather()); });
      var vars = [];
      Array.prototype.forEach.call(varBox.children, function (r) { if (r._gather) vars.push(r._gather()); });
      var nodes = [];
      Array.prototype.forEach.call(nodeBox.children, function (r) { if (r._gather) nodes.push(r._gather()); });
      var out = {
        mobid: Panel.mobId, zone: Panel.zone, defaultMood: mood.value,
        greetings: greets, patterns: pats,
        memory: { expiryPeriod: expiry.value.trim() }
      };
      if (rootText.value.trim() || nodes.length || vars.length) {
        out.tree = { root: { text: rootText.value, hints: rootHints.value, variants: vars }, nodes: nodes };
      }
      return out;
    }
  };

  Panel.onResult = function (res) {
    if (!Panel.active) return;
    var errEl = document.getElementById("dlg-errors");
    var warnEl = document.getElementById("dlg-warnings");
    if (res && res.ok) {
      if (Panel.deleting) { Panel.deleting = false; toast("Dialogue deleted — the NPC is mute.", false); Panel.close(); return; }
      Panel.saving = false; Panel.dirty = false;
      toast("Dialogue saved — live immediately, no reboot.", false);
      if (warnEl) warnEl.textContent = (res.warnings && res.warnings.length)
        ? ("Warnings:\n" + res.warnings.join("\n")) : "";
      if (errEl) errEl.textContent = "";
      return;
    }
    Panel.saving = false; Panel.deleting = false;
    var msg = (res && res.error) || "Dialogue error";
    if (errEl) errEl.textContent = msg;
    toast("Dialogue refused — see the errors above the form.", true);
  };

  window.Builder = window.Builder || {};
  window.Builder.DialoguePanel = Panel;
})();
