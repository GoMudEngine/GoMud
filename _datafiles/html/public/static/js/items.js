"use strict";
/*
 * items.js — the item-template editor (admin web-building 2), a second mode of
 * the /build page. Consumes Build.Items (list) + Build.Item (detail) GMCP and
 * drives Build.Item.Create/Update/Delete. The form morphs by item type; fields
 * the form doesn't cover (procs, sentient/voice, hunger) round-trip untouched
 * because the server rebuilds the spec from the loaded copy.
 */
(function () {
  var ARMOR_SLOTS = ["offhand", "head", "neck", "body", "belt", "gloves", "ring",
    "wrist", "back", "shoulders", "legs", "feet", "tail", "componentbag"];
  var CONSUMABLE = ["potion", "food", "drink"];
  var RARITY = [
    { v: 0, t: "untiered" }, { v: 10, t: "10 — legendary" }, { v: 20, t: "20 — epic" },
    { v: 30, t: "30 — rare" }, { v: 40, t: "40 — uncommon" }, { v: 50, t: "50 — common" }
  ];
  var CASTER = { wand: 1, sceptre: 1, staff: 1 };

  // Per-field hints (units + whether it's a whole number, a multiplier, a
  // percent, etc.) so the editor makes the field's meaning obvious instead of
  // presenting a bare number box. Keyed by the form field key.
  var HINTS = {
    value: "gold value — whole number (auto-set if left 0)",
    weight: "pounds (decimals ok)",
    uses: "charges — whole number (0 = unlimited)",
    description: "shown to players — wraps at ~78 chars",
    vendorCategories: "comma-separated shop categories",
    // weapon
    damageMultiplier: "multiplier — 1.0 = baseline",
    spellDamageMultiplier: "multiplier — 1.0 = baseline",
    speedMultiplier: "multiplier — 1.0 = baseline",
    hands: "1 or 2 — whole number",
    parryRating: "flat parry bonus — whole number",
    staminaCost: "per swing — whole number",
    waitRounds: "extra rounds — whole number",
    minStrength: "whole number",
    reach: "meters (e.g. 1.0)",
    grappleModifier: "additive modifier (decimals ok)",
    // armor
    physicalMitigation: "percent — whole number (0–75)",
    magicalMitigation: "percent — whole number (0–75)",
    convictionMitigation: "percent — whole number (0–75)",
    blockRating: "flat block bonus — whole number",
    escapeModifier: "additive modifier (decimals ok)",
    // consumable
    buffIds: "comma-separated buff ids applied on use",
    toxicity: "whole number",
    bottleAgingMultiplier: "multiplier",
    fermentRounds: "rounds — whole number",
    peakRounds: "rounds — whole number",
    decayRounds: "rounds — whole number",
    spoilRounds: "rounds — whole number",
    bandolierCapacity: "whole number",
    // component
    weightReduction: "fraction 0–1 (0.30 = 30% lighter)",
    bagCapacity: "whole number",
    // advanced / pinnacle
    reserveHealthPct: "fraction 0–1 of max HP locked while equipped",
    reserveStaminaPct: "fraction 0–1 of max stamina locked while equipped",
    reserveConvictionPct: "fraction 0–1 of max conviction locked while equipped",
    hungerRounds: "rounds without a kill before it feeds on the wielder (0 = never)",
    hungerDrainPct: "fraction 0–1 of max HP drained per hungry round",
    mutationTickInterval: "rounds between mutation rolls while worn (0 = never)",
    mutationTickChance: "percent chance per roll (0–100)",
    mutationRarityFloor: "min mutation rarity in the pool (0–10)",
    wornBuffIds: "comma-separated buff ids applied while worn",
  };

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

  // Bare DOM builders used by the proc-row editor (which lives outside the
  // renderForm closure that owns numField/selectField).
  function selBox(opts, val) {
    var s = document.createElement("select");
    opts.forEach(function (o) {
      var op = document.createElement("option"); op.value = o; op.textContent = o === "" ? "(none)" : o;
      if (o === val) op.selected = true; s.appendChild(op);
    });
    return s;
  }
  function numBox(val, step) {
    var i = document.createElement("input"); i.type = "number"; i.step = step || "1";
    i.value = (val === 0 ? "0" : (val || "")); return i;
  }
  function labelWrap(text, input) {
    var d = document.createElement("div"); d.style.flex = "1 1 auto"; d.style.minWidth = "0";
    var l = document.createElement("label"); l.textContent = text; d.appendChild(l); d.appendChild(input); return d;
  }

  var Panel = {
    rows: [],
    search: "",
    typeFilter: "",
    types: [],       // all type ids (from a detail; falls back to list's distinct)
    selectedId: 0,
    detail: null,
    dirty: false,
    fields: null,    // gather closures for the current form
    pendingSelect: 0,
    saving: false,
    deleting: false,
    advancedOpen: false,
    _advItemId: 0,
  };

  // ---- list ----
  Panel.render = function (rows) {
    this.rows = rows || [];
    var host = document.getElementById("itemlist");
    if (!host) return;
    host.innerHTML = "";

    var distinct = {};
    this.rows.forEach(function (r) { distinct[r.type] = true; });
    var typeOpts = Object.keys(distinct).sort();

    var newBtn = ce("button", { "class": "newitem", text: "+ New Item" });
    newBtn.addEventListener("click", promptNewItem);
    host.appendChild(newBtn);

    var filters = ce("div", { "class": "filters" });
    var search = ce("input", { type: "text", placeholder: "search id or name" });
    search.value = this.search;
    search.addEventListener("input", function () { Panel.search = search.value; Panel.drawRows(); });
    var typeSel = ce("select", {});
    typeSel.appendChild(ce("option", { value: "", text: "all types" }));
    typeOpts.forEach(function (t) {
      var o = ce("option", { value: t, text: t });
      if (t === Panel.typeFilter) o.selected = true;
      typeSel.appendChild(o);
    });
    typeSel.addEventListener("change", function () { Panel.typeFilter = typeSel.value; Panel.drawRows(); });
    filters.appendChild(search);
    filters.appendChild(typeSel);
    host.appendChild(filters);

    this.listBody = ce("div", {});
    host.appendChild(this.listBody);
    this.drawRows();
  };

  Panel.drawRows = function () {
    if (!this.listBody) return;
    this.listBody.innerHTML = "";
    var q = this.search.trim().toLowerCase();
    var tf = this.typeFilter;
    var shown = 0;
    this.rows.forEach(function (r) {
      if (tf && r.type !== tf) return;
      if (q && String(r.id).indexOf(q) === -1 && (r.name || "").toLowerCase().indexOf(q) === -1) return;
      shown++;
      var row = ce("div", { "class": "irow" + (r.id === Panel.selectedId ? " sel" : "") });
      row.appendChild(ce("span", { "class": "iid", text: "#" + r.id + " " }));
      row.appendChild(document.createTextNode(r.name || "(unnamed)"));
      row.appendChild(ce("span", { "class": "ity", text: "  " + r.type + (r.subtype ? "/" + r.subtype : "") + (r.rarity ? " · T" + r.rarity : "") }));
      row.addEventListener("click", function () { Panel.selectItem(r.id); });
      Panel.listBody.appendChild(row);
    });
    if (!shown) this.listBody.appendChild(ce("div", { "style": "color:var(--gold-dim);font-style:italic;padding:8px;", text: "no items match" }));
  };

  Panel.selectItem = function (id) {
    if (this.dirty && !window.confirm("Discard unsaved changes to this item?")) return;
    this.selectedId = id;
    this.drawRows();
    gmcp("Build.Item.Get", { itemId: id });
  };

  function promptNewItem() {
    var types = Panel.types.length ? Panel.types : distinctTypes();
    var t = window.prompt("New item — type?\n(" + types.join(", ") + ")", "weapon");
    if (!t) return;
    t = t.trim().toLowerCase();
    if (types.indexOf(t) === -1) { toast("Unknown item type: " + t, true); return; }
    gmcp("Build.Item.Create", { type: t });
  }
  function distinctTypes() {
    var d = {};
    Panel.rows.forEach(function (r) { d[r.type] = true; });
    return Object.keys(d).sort();
  }

  // ---- form ----
  Panel.renderForm = function (detail) {
    this.detail = detail;
    this.types = detail.types || this.types;
    this.selectedId = detail.itemId;
    this.drawRows();
    var insp = document.getElementById("inspector");
    insp.innerHTML = "";
    this.fields = {};
    var F = this.fields;

    // Compose a field's hint: its static description plus, when the server sent
    // an observed min–max for this field across items of the same type, a
    // "seen 0.30–3.50" span so the author knows what values are normal.
    var RANGES = detail.ranges || {};
    function fmtNum(n, isInt) { return isInt ? String(Math.round(n)) : String(Math.round(n * 100) / 100); }
    function hintFor(key, isInt) {
      var base = HINTS[key] || "";
      var r = RANGES[key];
      if (r && r.length === 2) {
        var span = "seen " + fmtNum(r[0], isInt) + "–" + fmtNum(r[1], isInt);
        base = base ? base + " · " + span : span;
      }
      return base;
    }

    insp.appendChild(ce("h2", { text: "Item #" + detail.itemId }));

    // helpers bound to F
    function textField(label, key, val) {
      var i = ce("input", { type: "text" }); i.value = val == null ? "" : val;
      i.addEventListener("input", markDirty);
      F[key] = function () { return i.value; };
      return field(label, i, hintFor(key, false));
    }
    function numField(label, key, val, step) {
      // Fields created without a fractional step are integer-typed on the
      // server (parryRating, hands, mitigations, ...). Go's json rejects a
      // fractional number into an int field and fails the WHOLE Save payload,
      // so integer fields snap to a whole number on blur — visible, next to a
      // "whole number" hint — instead of a silent gather-time surprise. Gather
      // rounds again as a final guard.
      var isInt = !step || step === "1";
      var i = ce("input", { type: "number", step: step || "1" }); i.value = (val === 0 ? "0" : (val || ""));
      i.addEventListener("input", markDirty);
      if (isInt) i.addEventListener("blur", function () {
        if (i.value !== "") i.value = String(Math.round(parseFloat(i.value) || 0));
      });
      F[key] = function () { var n = parseFloat(i.value) || 0; return isInt ? Math.round(n) : n; };
      return field(label, i, hintFor(key, isInt));
    }
    function checkField(label, key, val) {
      var cb = ce("input", { type: "checkbox" }); cb.checked = !!val; cb.addEventListener("change", markDirty);
      F[key] = function () { return cb.checked; };
      return ce("label", { "class": "chk" }, [cb, ce("span", { text: " " + label })]);
    }
    function selectField(label, key, val, opts) {
      var s = ce("select", {});
      opts.forEach(function (o) {
        var ov = (typeof o === "object") ? o.v : o, ot = (typeof o === "object") ? o.t : o;
        var op = ce("option", { value: ov, text: ot === "" ? "(none)" : ot });
        if (String(ov) === String(val)) op.selected = true;
        s.appendChild(op);
      });
      s.addEventListener("change", markDirty);
      if (key === "type") s.addEventListener("change", function () { rerenderTypeSections(s.value); });
      F[key] = function () { var v = s.value; return v; };
      return field(label, s, hintFor(key, false));
    }

    // Common
    insp.appendChild(sectionTitle("Common"));
    insp.appendChild(textField("Name", "name", detail.name));
    insp.appendChild(textField("Display name", "displayName", detail.displayName));
    insp.appendChild(textField("Simple name", "nameSimple", detail.nameSimple));
    var desc = ce("textarea", {}); desc.value = detail.description || ""; desc.addEventListener("input", markDirty);
    F.description = function () { return desc.value; };
    insp.appendChild(field("Description", desc, hintFor("description", false)));
    insp.appendChild(selectField("Type", "type", detail.type, detail.types || []));
    insp.appendChild(selectField("Subtype", "subtype", detail.subtype, [""].concat(detail.subtypes || [])));
    insp.appendChild(ce("div", { "class": "row" }, [numField("Value", "value", detail.value), numField("Weight", "weight", detail.weight, "0.1")]));
    insp.appendChild(ce("div", { "class": "row" }, [numField("Uses", "uses", detail.uses), selectField("Rarity", "rarityTier", detail.rarityTier, RARITY)]));
    // Vendor categories as checkboxes from the valid set — a salable item needs
    // at least one, and a bad value would brick the server at boot, so no free text.
    var vcSel = {}; (detail.vendorCategories || []).forEach(function (c) { vcSel[c] = true; });
    var vcChecks = [];
    var vcBox = ce("div", { "class": "flags" });
    (detail.vendorCats || []).forEach(function (c) {
      var cb = ce("input", { type: "checkbox" }); cb.checked = !!vcSel[c]; cb.addEventListener("change", markDirty);
      vcBox.appendChild(ce("label", {}, [cb, ce("span", { text: " " + c })]));
      vcChecks.push({ cat: c, cb: cb });
    });
    F.vendorCategories = function () { return vcChecks.filter(function (x) { return x.cb.checked; }).map(function (x) { return x.cat; }); };
    insp.appendChild(ce("div", {}, [ce("label", { text: "Vendor categories" }), vcBox,
      ce("div", { style: "font-size:10px;color:var(--gold-dim);margin-top:2px;", text: "Shop disciplines that stock this item. A salable item needs at least one — otherwise check “not-salable” below." })]));
    var flags = ce("div", { "class": "flags" }, [
      checkField("not-salable", "notSalable", detail.notSalable),
      checkField("never-drops", "neverDrops", detail.neverDrops),
      checkField("restricted", "restricted", detail.restricted),
      checkField("cursed", "cursed", detail.cursed)
    ]);
    insp.appendChild(flags);
    insp.appendChild(ce("div", { style: "font-size:10px;color:var(--gold-dim);margin-top:2px;", text: "not-salable: never stocked by shops (quest items, drops, unique gear) — exempt from the vendor-category requirement." }));
    insp.appendChild(textField("Quest token", "questToken", detail.questToken));

    // StatMods
    insp.appendChild(sectionTitle("Stat modifiers"));
    var smBox = ce("div", {});
    var smRows = [];
    function addStatRow(name, value) {
      var sel = ce("select", {}); sel.style.flex = "1";
      (detail.stats || []).forEach(function (st) { var o = ce("option", { value: st, text: st }); if (st === name) o.selected = true; sel.appendChild(o); });
      var num = ce("input", { type: "number" }); num.value = value || 0; num.style.width = "70px";
      var rm = ce("button", { "class": "mini rm", text: "✕" });
      var row = ce("div", { "class": "kv" }, [sel, num, rm]);
      sel.addEventListener("change", markDirty); num.addEventListener("input", markDirty);
      rm.addEventListener("click", function () { smBox.removeChild(row); smRows.splice(smRows.indexOf(row), 1); markDirty(); });
      row._sel = sel; row._num = num;
      smRows.push(row); smBox.appendChild(row);
    }
    var sm = detail.statMods || {};
    Object.keys(sm).forEach(function (k) { addStatRow(k, sm[k]); });
    insp.appendChild(smBox);
    var addSm = ce("button", { "class": "mini", text: "+ stat" });
    addSm.addEventListener("click", function () { addStatRow((detail.stats || ["strength"])[0], 0); markDirty(); });
    insp.appendChild(addSm);
    F.statMods = function () {
      var out = {};
      smRows.forEach(function (r) { var n = r._sel.value; var v = parseInt(r._num.value, 10) || 0; if (n && v) out[n] = v; });
      return out;
    };

    // Type-specific sections live in a container we can re-render on type change.
    this.typeSections = ce("div", {});
    insp.appendChild(this.typeSections);
    var self = this;
    function rerenderTypeSections(type) { self.buildTypeSections(self.typeSections, type, detail, F, markDirty, field, numField, textField, checkField, selectField, hintFor); }
    rerenderTypeSections(detail.type);

    // Advanced (sentient & procs) — collapsible, auto-open when populated.
    this.buildAdvancedSection(insp, detail, F, markDirty, field, numField, textField, checkField, selectField, hintFor);

    // Save + delete row
    var save = ce("button", { id: "item-save", text: "Save Item", disabled: "disabled" });
    save.addEventListener("click", function () { Panel.save(); });
    var del = ce("button", { "class": "mini rm", text: "Delete item", "style": "width:100%;margin-top:6px;padding:6px;" });
    del.addEventListener("click", function () { Panel.del(); });
    insp.appendChild(ce("div", { "class": "save-row" }, [save, del]));
    insp.appendChild(ce("div", { id: "item-refs", "style": "color:var(--danger);font-size:11px;margin-top:6px;" }));

    this.setDirty(false);
  };

  function field(labelText, input, hint) {
    var kids = [ce("label", { text: labelText }), input];
    if (hint) kids.push(ce("div", { style: "font-size:10px;color:var(--gold-dim);margin-top:2px;", text: hint }));
    return ce("div", {}, kids);
  }
  function sectionTitle(t) { return ce("h3", { text: t }); }
  function markDirty() { Panel.setDirty(true); }
  Panel.setDirty = function (d) {
    this.dirty = d;
    var s = document.getElementById("item-save");
    if (s) s.disabled = !d;
  };

  Panel.buildTypeSections = function (host, type, detail, F, markDirty, field, numField, textField, checkField, selectField, hintFor) {
    host.innerHTML = "";
    var isArmor = ARMOR_SLOTS.indexOf(type) !== -1;
    var isConsum = CONSUMABLE.indexOf(type) !== -1;

    if (type === "weapon") {
      host.appendChild(sectionTitle("Weapon"));
      host.appendChild(ce("div", { "class": "row" }, [numField("Damage mult", "damageMultiplier", detail.damageMultiplier, "0.05"), numField("Hands", "hands", detail.hands || 1)]));
      host.appendChild(ce("div", { "class": "row" }, [numField("Parry", "parryRating", detail.parryRating), numField("Speed mult", "speedMultiplier", detail.speedMultiplier, "0.05")]));
      host.appendChild(ce("div", { "class": "row" }, [numField("Stamina cost", "staminaCost", detail.staminaCost), numField("Wait rounds", "waitRounds", detail.waitRounds)]));
      host.appendChild(ce("div", { "class": "row" }, [numField("Min strength", "minStrength", detail.minStrength), numField("Reach", "reach", detail.reach, "0.1")]));
      host.appendChild(ce("div", { "class": "row" }, [numField("Grapple mod", "grappleModifier", detail.grappleModifier, "0.05"), selectField("Element", "element", detail.element, [""].concat(detail.elements || []))]));
      if (CASTER[detail.subtype]) host.appendChild(numField("Spell dmg mult", "spellDamageMultiplier", detail.spellDamageMultiplier, "0.05"));
      if (detail.subtype === "shooting") host.appendChild(textField("Ammo tag", "ammoTag", detail.ammoTag));
    }
    if (isArmor) {
      host.appendChild(sectionTitle("Armor"));
      host.appendChild(ce("div", { "class": "row" }, [numField("Phys mitig %", "physicalMitigation", detail.physicalMitigation), numField("Magic mitig %", "magicalMitigation", detail.magicalMitigation)]));
      host.appendChild(ce("div", { "class": "row" }, [numField("Convic mitig %", "convictionMitigation", detail.convictionMitigation), numField("Block", "blockRating", detail.blockRating)]));
      host.appendChild(numField("Escape mod", "escapeModifier", detail.escapeModifier, "0.05"));
    }
    if (isConsum) {
      host.appendChild(sectionTitle("Consumable"));
      var bf = ce("input", { type: "text", placeholder: "comma buff ids" }); bf.value = (detail.buffIds || []).join(", ");
      bf.addEventListener("input", markDirty);
      F.buffIds = function () { return bf.value.split(",").map(function (s) { return parseInt(s.trim(), 10); }).filter(function (n) { return !isNaN(n); }); };
      host.appendChild(field("Buff ids", bf, hintFor("buffIds", false)));
      host.appendChild(ce("div", { "class": "row" }, [numField("Toxicity", "toxicity", detail.toxicity), numField("Bottle aging ×", "bottleAgingMultiplier", detail.bottleAgingMultiplier, "0.05")]));
      host.appendChild(sectionTitle("Aging (rounds)"));
      host.appendChild(ce("div", { "class": "row" }, [numField("Ferment", "fermentRounds", detail.fermentRounds), numField("Peak", "peakRounds", detail.peakRounds)]));
      host.appendChild(ce("div", { "class": "row" }, [numField("Decay", "decayRounds", detail.decayRounds), numField("Spoil", "spoilRounds", detail.spoilRounds)]));
      host.appendChild(ce("div", { "class": "flags" }, [checkField("bandolier", "isBandolier", detail.isBandolier)]));
      host.appendChild(numField("Bandolier capacity", "bandolierCapacity", detail.bandolierCapacity));
    }
    // Component / bag (flag-driven; available on any type)
    host.appendChild(sectionTitle("Component / bag"));
    host.appendChild(ce("div", { "class": "flags" }, [checkField("is-component", "isComponent", detail.isComponent)]));
    host.appendChild(textField("Component tag", "componentTag", detail.componentTag));
    host.appendChild(ce("div", { "class": "row" }, [numField("Weight reduction", "weightReduction", detail.weightReduction, "0.05"), numField("Bag capacity", "bagCapacity", detail.bagCapacity)]));
    var salBox = ce("div", {});
    var salRows = [];
    function salRow(tag, qty) {
      var t = ce("input", { type: "text", placeholder: "item_tag" }); t.value = tag || "";
      var q = ce("input", { type: "number", placeholder: "qty" }); q.value = qty || 0; q.style.width = "60px";
      var rm = ce("button", { "class": "mini rm", text: "✕" });
      var row = ce("div", { "class": "kv" }, [t, q, rm]);
      t.addEventListener("input", markDirty); q.addEventListener("input", markDirty);
      rm.addEventListener("click", function () { salBox.removeChild(row); salRows.splice(salRows.indexOf(row), 1); markDirty(); });
      row._t = t; row._q = q; salRows.push(row); salBox.appendChild(row);
    }
    (detail.salvageReturns || []).forEach(function (s) { salRow(s.itemTag, s.quantity); });
    host.appendChild(ce("div", {}, [ce("label", { text: "Salvage returns" }), salBox]));
    var addSal = ce("button", { "class": "mini", text: "+ salvage" });
    addSal.addEventListener("click", function () { salRow("", 1); markDirty(); });
    host.appendChild(addSal);
    F.salvageReturns = function () {
      var out = [];
      salRows.forEach(function (r) { var tg = r._t.value.trim(); if (tg) out.push({ itemTag: tg, quantity: parseInt(r._q.value, 10) || 0 }); });
      return out;
    };

    if (type === "key") host.appendChild(textField("Key lock id", "keyLockId", detail.keyLockId));
    if (type === "ammo") host.appendChild(textField("Ammo tag", "ammoTag", detail.ammoTag));
  };

  Panel.buildAdvancedSection = function (insp, detail, F, markDirty, field, numField, textField, checkField, selectField, hintFor) {
    var self = this;
    var hasAdv = (detail.procs && detail.procs.length) || detail.voiceId ||
      detail.reserveHealthPct || detail.reserveStaminaPct || detail.reserveConvictionPct ||
      detail.hungerRounds || detail.hungerDrainPct || detail.tauntPull ||
      detail.mutationTickInterval || detail.mutationTickChance || detail.mutationRarityFloor ||
      (detail.wornBuffIds && detail.wornBuffIds.length);
    // Recompute open state only when a different item is selected; preserve the
    // author's toggle across same-item re-renders (e.g. the post-save re-Get).
    if (detail.itemId !== this._advItemId) { this.advancedOpen = !!hasAdv; this._advItemId = detail.itemId; }

    function headText() { return (self.advancedOpen ? "▾ " : "▸ ") + "Advanced — sentient & procs"; }
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

    this.buildProcEditor(body, detail, F, markDirty);

    body.appendChild(sectionTitle("Reserves"));
    body.appendChild(ce("div", { "class": "row" }, [
      numField("Reserve HP", "reserveHealthPct", detail.reserveHealthPct, "0.05"),
      numField("Reserve SP", "reserveStaminaPct", detail.reserveStaminaPct, "0.05")]));
    body.appendChild(numField("Reserve CP", "reserveConvictionPct", detail.reserveConvictionPct, "0.05"));

    body.appendChild(sectionTitle("Sentient"));
    body.appendChild(selectField("Voice", "voiceId", detail.voiceId, [""].concat(detail.voices || [])));
    body.appendChild(ce("div", { "class": "flags" }, [checkField("taunt-pull", "tauntPull", detail.tauntPull)]));

    body.appendChild(sectionTitle("Hunger"));
    body.appendChild(ce("div", { "class": "row" }, [
      numField("Hunger rounds", "hungerRounds", detail.hungerRounds),
      numField("Hunger drain", "hungerDrainPct", detail.hungerDrainPct, "0.01")]));

    body.appendChild(sectionTitle("Mutation drip"));
    body.appendChild(ce("div", { "class": "row" }, [
      numField("Tick interval", "mutationTickInterval", detail.mutationTickInterval),
      numField("Tick chance", "mutationTickChance", detail.mutationTickChance)]));
    body.appendChild(numField("Rarity floor", "mutationRarityFloor", detail.mutationRarityFloor));

    body.appendChild(sectionTitle("Worn buffs"));
    var wb = ce("input", { type: "text", placeholder: "comma buff ids" });
    wb.value = (detail.wornBuffIds || []).join(", ");
    wb.addEventListener("input", markDirty);
    F.wornBuffIds = function () { return wb.value.split(",").map(function (s) { return parseInt(s.trim(), 10); }).filter(function (n) { return !isNaN(n); }); };
    body.appendChild(field("Worn buff ids", wb, hintFor("wornBuffIds", false)));
  };

  Panel.buildProcEditor = function (body, detail, F, markDirty) {
    body.appendChild(sectionTitle("Procs"));
    var procBox = ce("div", {});
    body.appendChild(procBox);
    var procRows = [];

    function addProc(p) {
      p = p || { trigger: "", effect: "", chance: 100, cooldownRounds: 0, params: {} };
      var trig = selBox([""].concat(detail.procTriggers || []), p.trigger);
      var eff = selBox([""].concat(detail.procEffects || []), p.effect);
      var chance = numBox(p.chance, "1"); chance.style.width = "60px";
      var cd = numBox(p.cooldownRounds, "1"); cd.style.width = "60px";
      var rm = ce("button", { "class": "mini rm", text: "✕ proc" });

      var paramBox = ce("div", { style: "margin:3px 0 3px 10px;" });
      var paramRows = [];
      function addParam(k, v) {
        var name = ce("input", { type: "text", placeholder: "param" }); name.value = k || ""; name.style.flex = "1";
        var val = ce("input", { type: "number", step: "0.05" }); val.value = (v === 0 ? "0" : (v || "")); val.style.width = "70px";
        var prm = ce("button", { "class": "mini rm", text: "✕" });
        var prow = ce("div", { "class": "kv" }, [name, val, prm]);
        name.addEventListener("input", markDirty); val.addEventListener("input", markDirty);
        prm.addEventListener("click", function () { paramBox.removeChild(prow); paramRows.splice(paramRows.indexOf(prow), 1); markDirty(); });
        prow._name = name; prow._val = val; paramRows.push(prow); paramBox.appendChild(prow);
      }
      Object.keys(p.params || {}).forEach(function (k) { addParam(k, p.params[k]); });
      var addParamBtn = ce("button", { "class": "mini", text: "+ param" });
      addParamBtn.addEventListener("click", function () { addParam("", 0); markDirty(); });

      var row = ce("div", { style: "border:1px solid var(--tooled);border-radius:4px;padding:6px;margin:4px 0;" }, [
        ce("div", { "class": "row" }, [labelWrap("Trigger", trig), labelWrap("Effect", eff)]),
        ce("div", { "class": "row" }, [labelWrap("Chance", chance), labelWrap("Cooldown", cd)]),
        ce("div", {}, [ce("label", { text: "Params (e.g. ratio 0.25)" }), paramBox, addParamBtn]),
        rm
      ]);
      trig.addEventListener("change", markDirty); eff.addEventListener("change", markDirty);
      chance.addEventListener("input", markDirty); cd.addEventListener("input", markDirty);
      rm.addEventListener("click", function () { procBox.removeChild(row); procRows.splice(procRows.indexOf(row), 1); markDirty(); });
      row._get = function () {
        var params = {};
        paramRows.forEach(function (pr) { var n = pr._name.value.trim(); if (n) params[n] = parseFloat(pr._val.value) || 0; });
        return { trigger: trig.value, effect: eff.value, chance: parseInt(chance.value, 10) || 0, cooldownRounds: parseInt(cd.value, 10) || 0, params: params };
      };
      procRows.push(row); procBox.appendChild(row);
    }

    (detail.procs || []).forEach(addProc);
    var addBtn = ce("button", { "class": "mini", text: "+ proc" });
    addBtn.addEventListener("click", function () { addProc(); markDirty(); });
    body.appendChild(addBtn);

    F.procs = function () {
      return procRows.map(function (r) { return r._get(); })
        .filter(function (p) { return p.trigger || p.effect; });
    };
  };

  // ---- mutations ----
  Panel.gather = function () {
    var F = this.fields || {};
    var d = this.detail || {};
    function g(k, dflt) { return F[k] ? F[k]() : (d[k] !== undefined ? d[k] : dflt); }
    return {
      itemId: d.itemId,
      name: g("name", ""), displayName: g("displayName", ""), nameSimple: g("nameSimple", ""),
      description: g("description", ""), type: g("type", ""), subtype: g("subtype", ""),
      value: g("value", 0), weight: g("weight", 0), uses: g("uses", 0), rarityTier: parseInt(g("rarityTier", 0), 10) || 0,
      vendorCategories: g("vendorCategories", []), notSalable: g("notSalable", false), neverDrops: g("neverDrops", false),
      restricted: g("restricted", false), cursed: g("cursed", false), questToken: g("questToken", ""),
      statMods: g("statMods", {}),
      damageMultiplier: g("damageMultiplier", 0), spellDamageMultiplier: g("spellDamageMultiplier", 0), hands: g("hands", 0),
      parryRating: g("parryRating", 0), speedMultiplier: g("speedMultiplier", 0), staminaCost: g("staminaCost", 0),
      waitRounds: g("waitRounds", 0), minStrength: g("minStrength", 0), reach: g("reach", 0), grappleModifier: g("grappleModifier", 0),
      element: g("element", ""), ammoTag: g("ammoTag", ""),
      physicalMitigation: g("physicalMitigation", 0), magicalMitigation: g("magicalMitigation", 0),
      convictionMitigation: g("convictionMitigation", 0), blockRating: g("blockRating", 0), escapeModifier: g("escapeModifier", 0),
      buffIds: g("buffIds", []), toxicity: g("toxicity", 0),
      fermentRounds: g("fermentRounds", 0), peakRounds: g("peakRounds", 0), decayRounds: g("decayRounds", 0), spoilRounds: g("spoilRounds", 0),
      bottleAgingMultiplier: g("bottleAgingMultiplier", 0), isBandolier: g("isBandolier", false), bandolierCapacity: g("bandolierCapacity", 0),
      isComponent: g("isComponent", false), componentTag: g("componentTag", ""), weightReduction: g("weightReduction", 0),
      bagCapacity: g("bagCapacity", 0), salvageReturns: g("salvageReturns", []), keyLockId: g("keyLockId", ""),
      procs: g("procs", []),
      reserveHealthPct: g("reserveHealthPct", 0), reserveStaminaPct: g("reserveStaminaPct", 0), reserveConvictionPct: g("reserveConvictionPct", 0),
      voiceId: g("voiceId", ""), tauntPull: g("tauntPull", false),
      hungerRounds: g("hungerRounds", 0), hungerDrainPct: g("hungerDrainPct", 0),
      mutationTickInterval: g("mutationTickInterval", 0), mutationTickChance: g("mutationTickChance", 0), mutationRarityFloor: g("mutationRarityFloor", 0),
      wornBuffIds: g("wornBuffIds", [])
    };
  };

  Panel.save = function () {
    if (!this.detail) return;
    var req = this.gather();
    if (!req.name || !req.type) { toast("Name and type are required", true); return; }
    this.saving = true;
    gmcp("Build.Item.Update", req);
  };
  Panel.del = function () {
    if (!this.detail) return;
    if (!window.confirm("Delete item #" + this.detail.itemId + "?")) return;
    this.deleting = true;
    gmcp("Build.Item.Delete", { itemId: this.detail.itemId });
  };

  // Build.Result routing from build.html.
  Panel.onResult = function (obj) {
    var refsEl = document.getElementById("item-refs");
    if (obj && obj.ok) {
      if (this.deleting) { this.deleting = false; this.detail = null; this.selectedId = 0; clearItemInspector(); toast("Item deleted.", false); return; }
      if (this.saving) {
        this.saving = false; this.setDirty(false); toast("Saved.", false);
        // Reload the persisted spec so the form reflects exactly what was
        // stored (e.g. a name the server canonicalized) — visible proof the
        // save took, not just a fire-and-forget confirmation.
        if (this.selectedId) gmcp("Build.Item.Get", { itemId: this.selectedId });
        return;
      }
      if (obj.itemId) { this.pendingSelect = obj.itemId; toast("Created.", false); } // create → auto-select on next list
    } else {
      this.saving = false; this.deleting = false;
      if (obj && obj.refs && obj.refs.length && refsEl) {
        refsEl.textContent = "Still used by: " + obj.refs.map(function (r) { return r.id; }).join(", ") + " — remove those first.";
      }
      toast((obj && obj.error) || "Item error", true);
    }
  };

  function clearItemInspector() {
    var insp = document.getElementById("inspector");
    insp.innerHTML = "";
    insp.appendChild(ce("h2", { text: "Items" }));
    insp.appendChild(ce("div", { "class": "empty", text: "Select an item on the left, or + New Item." }));
  }
  Panel.clear = clearItemInspector;

  // Called after a Build.Items refresh to consume a pending create-select.
  Panel.afterListRefresh = function () {
    if (this.pendingSelect) {
      var id = this.pendingSelect; this.pendingSelect = 0;
      this.selectItem(id);
    }
  };

  window.Builder = window.Builder || {};
  window.Builder.ItemsPanel = Panel;
})();
