# Item Editor — Advanced (Pinnacle) Behaviors Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let admins author the advanced `ItemSpec` fields (combat procs, sentient voice, resource reserves, hunger, mutation-drip, worn-buffs) in the `/build` item editor, behind a collapsible "Advanced" section that auto-opens when an item already carries such data.

**Architecture:** Extend the existing `Build.Item.*` GMCP contract — add fields to `itemUpdateReq`/`specToReq`/`reqToSpec` and enum lists (valid triggers/effects/voices) to `itemDetail`, exactly mirroring how `Types`/`Stats`/`VendorCats` already work. The frontend renders a collapsible section reusing the existing `numField`/`selectField`/`checkField` helpers plus a repeatable proc-row editor (like stat-mods). No new boot-brick class: `SaveItemSpec → Validate()` already enforces proc/reserve/mutation constraints, surfacing bad entries as the same red error toast.

**Tech Stack:** Go (`internal/items`, `internal/itemvoices`, `modules/gmcp`), vanilla JS (`_datafiles/html/public/static/js/items.js`).

## File structure

- `internal/items/itemspec.go` — add `ValidProcTriggers()` / `ValidProcEffects()` exported accessors.
- `internal/itemvoices/itemvoices.go` — add `AllVoiceIds()` accessor.
- `internal/items/save_test.go` — add: `SaveItemSpec` rejects an item with an invalid proc.
- `internal/items/itemspec_test.go` (create if absent) — accessor tests.
- `internal/itemvoices/itemvoices_test.go` — `AllVoiceIds` test.
- `modules/gmcp/gmcp.Item.go` — `procRow` type; advanced fields on `itemUpdateReq`; `specToReq`/`reqToSpec` mappings; enum fields on `itemDetail`; `procTriggerIds`/`procEffectIds`/`itemVoiceIds` providers; populate in `buildItemGet`.
- `modules/gmcp/gmcp.Item_test.go` — round-trip + enum tests.
- `_datafiles/html/public/static/js/items.js` — collapsible Advanced section + proc-row editor + `gather()` additions.

---

### Task 1: Export valid proc triggers/effects and voice ids

**Files:**
- Modify: `internal/items/itemspec.go` (after the `validProcEffects` var, ~line 251)
- Modify: `internal/itemvoices/itemvoices.go` (near `GetVoice`, ~line 79)
- Test: `internal/items/itemspec_test.go`, `internal/itemvoices/itemvoices_test.go`

- [ ] **Step 1: Write the failing tests**

In `internal/items/itemspec_test.go` (create if it does not exist; package `items`):

```go
package items

import (
	"slices"
	"testing"
)

func TestValidProcTriggers(t *testing.T) {
	got := ValidProcTriggers()
	for _, want := range []string{"on_hit", "on_kill", "on_block", "on_grapple", "on_spell_hit"} {
		if !slices.Contains(got, want) {
			t.Errorf("missing trigger %q in %v", want, got)
		}
	}
	if !slices.IsSorted(got) {
		t.Errorf("triggers must be sorted: %v", got)
	}
}

func TestValidProcEffects(t *testing.T) {
	got := ValidProcEffects()
	for _, want := range []string{"lifesteal", "steal_pool", "aoe_stun", "apply_condition"} {
		if !slices.Contains(got, want) {
			t.Errorf("missing effect %q in %v", want, got)
		}
	}
}
```

In `internal/itemvoices/itemvoices_test.go` (package `itemvoices`):

```go
package itemvoices

import (
	"slices"
	"testing"
)

func TestAllVoiceIds(t *testing.T) {
	restore := SeedVoicesForTest(map[string]*VoiceSpec{
		"blackrazor": {VoiceId: "blackrazor"},
		"aegis":      {VoiceId: "aegis"},
	})
	defer restore()
	got := AllVoiceIds()
	if !slices.Equal(got, []string{"aegis", "blackrazor"}) {
		t.Errorf("want sorted [aegis blackrazor], got %v", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/items/ ./internal/itemvoices/ -run 'ValidProc|AllVoiceIds'`
Expected: FAIL — `ValidProcTriggers`, `ValidProcEffects`, `AllVoiceIds` undefined.

- [ ] **Step 3: Implement the accessors**

In `internal/items/itemspec.go`, immediately after the `validProcEffects` map (line ~251):

```go
// ValidProcTriggers / ValidProcEffects return the accepted proc trigger and
// effect ids, sorted — for the web item editor's dropdowns and any external
// validation. Mirrors the validProcTriggers/validProcEffects maps used by
// ItemSpec.Validate.
func ValidProcTriggers() []string { return sortedBoolKeys(validProcTriggers) }
func ValidProcEffects() []string  { return sortedBoolKeys(validProcEffects) }

func sortedBoolKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
```

Confirm `internal/items/itemspec.go` already imports `sort` (it does — used by other code). If not, add it.

In `internal/itemvoices/itemvoices.go`, after `GetVoice` (line ~86):

```go
// AllVoiceIds returns every loaded voice id, sorted — for the item editor's
// sentient-voice dropdown (so only resolvable voices are offered).
func AllVoiceIds() []string {
	out := make([]string, 0, len(allVoices))
	for id := range allVoices {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}
```

Add `"sort"` to the `internal/itemvoices/itemvoices.go` import block if not present.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/items/ ./internal/itemvoices/ -run 'ValidProc|AllVoiceIds'`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/items/itemspec.go internal/items/itemspec_test.go internal/itemvoices/itemvoices.go internal/itemvoices/itemvoices_test.go
git commit -m "feat(items): export valid proc trigger/effect sets + AllVoiceIds"
```

---

### Task 2: Backend request/response fields for advanced behaviors

**Files:**
- Modify: `modules/gmcp/gmcp.Item.go` (`itemUpdateReq`, `specToReq`, `reqToSpec`)
- Test: `modules/gmcp/gmcp.Item_test.go`

- [ ] **Step 1: Write the failing test**

Add to `modules/gmcp/gmcp.Item_test.go`:

```go
func TestBuildItemUpdate_RoundTripsAdvancedFields(t *testing.T) {
	w := newFakeItemWorld()
	w.specs[10001] = &items.ItemSpec{ItemId: 10001, Name: "Old", Type: items.Weapon, Hands: 2}
	res := buildItemUpdate(w.deps(), itemUpdateReq{
		ItemId: 10001, Name: "Blackrazor", Type: "weapon", Description: "d", NotSalable: true,
		Procs: []procRow{{Trigger: "on_hit", Effect: "lifesteal", Chance: 100, CooldownRounds: 2,
			Params: map[string]float64{"ratio": 0.25}}},
		ReserveHealthPct: 0.25, VoiceId: "blackrazor", TauntPull: true,
		HungerRounds: 50, HungerDrainPct: 0.01,
		MutationTickInterval: 10, MutationTickChance: 5, MutationRarityFloor: 3,
		WornBuffIds: []int{7, 9},
	})
	if !res.Ok {
		t.Fatalf("update should succeed, got %+v", res)
	}
	got := w.saved[0]
	if len(got.Procs) != 1 || got.Procs[0].Trigger != "on_hit" || got.Procs[0].Effect != "lifesteal" ||
		got.Procs[0].Chance != 100 || got.Procs[0].CooldownRounds != 2 || got.Procs[0].Params["ratio"] != 0.25 {
		t.Errorf("proc not round-tripped: %+v", got.Procs)
	}
	if got.ReserveHealthPct != 0.25 || got.VoiceId != "blackrazor" || !got.TauntPull ||
		got.HungerRounds != 50 || got.HungerDrainPct != 0.01 ||
		got.MutationTickInterval != 10 || got.MutationTickChance != 5 || got.MutationRarityFloor != 3 {
		t.Errorf("advanced scalars not round-tripped: %+v", got)
	}
	if len(got.WornBuffIds) != 2 || got.WornBuffIds[0] != 7 {
		t.Errorf("worn buffs not round-tripped: %+v", got.WornBuffIds)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./modules/gmcp/ -run RoundTripsAdvancedFields`
Expected: FAIL — `procRow` undefined; `itemUpdateReq` has no `Procs`/`ReserveHealthPct`/etc.

- [ ] **Step 3: Add the `procRow` type and the fields**

In `modules/gmcp/gmcp.Item.go`, add `procRow` near the top payload types (after `itemSalvageRow`):

```go
type procRow struct {
	Trigger        string             `json:"trigger"`
	Effect         string             `json:"effect"`
	Chance         int                `json:"chance"`
	CooldownRounds int                `json:"cooldownRounds"`
	Params         map[string]float64 `json:"params"`
}
```

Append these fields to `itemUpdateReq` (after `KeyLockId`):

```go
	// advanced / pinnacle
	Procs                []procRow `json:"procs"`
	ReserveHealthPct     float64   `json:"reserveHealthPct"`
	ReserveStaminaPct    float64   `json:"reserveStaminaPct"`
	ReserveConvictionPct float64   `json:"reserveConvictionPct"`
	VoiceId              string    `json:"voiceId"`
	TauntPull            bool      `json:"tauntPull"`
	HungerRounds         int       `json:"hungerRounds"`
	HungerDrainPct       float64   `json:"hungerDrainPct"`
	MutationTickInterval int       `json:"mutationTickInterval"`
	MutationTickChance   int       `json:"mutationTickChance"`
	MutationRarityFloor  int       `json:"mutationRarityFloor"`
	WornBuffIds          []int     `json:"wornBuffIds"`
```

- [ ] **Step 4: Map the fields in `specToReq` and `reqToSpec`**

In `specToReq`, before `return req` (after the salvage loop):

```go
	req.ReserveHealthPct, req.ReserveStaminaPct, req.ReserveConvictionPct = s.ReserveHealthPct, s.ReserveStaminaPct, s.ReserveConvictionPct
	req.VoiceId, req.TauntPull = s.VoiceId, s.TauntPull
	req.HungerRounds, req.HungerDrainPct = s.HungerRounds, s.HungerDrainPct
	req.MutationTickInterval, req.MutationTickChance, req.MutationRarityFloor = s.MutationTickInterval, s.MutationTickChance, s.MutationRarityFloor
	req.WornBuffIds = s.WornBuffIds
	for _, p := range s.Procs {
		req.Procs = append(req.Procs, procRow{Trigger: p.Trigger, Effect: p.Effect, Chance: p.Chance, CooldownRounds: p.CooldownRounds, Params: p.Params})
	}
```

In `reqToSpec`, before `return s` (after the salvage loop):

```go
	s.ReserveHealthPct, s.ReserveStaminaPct, s.ReserveConvictionPct = req.ReserveHealthPct, req.ReserveStaminaPct, req.ReserveConvictionPct
	s.VoiceId, s.TauntPull = req.VoiceId, req.TauntPull
	s.HungerRounds, s.HungerDrainPct = req.HungerRounds, req.HungerDrainPct
	s.MutationTickInterval, s.MutationTickChance, s.MutationRarityFloor = req.MutationTickInterval, req.MutationTickChance, req.MutationRarityFloor
	s.WornBuffIds = req.WornBuffIds
	s.Procs = nil
	for _, p := range req.Procs {
		if p.Trigger == "" && p.Effect == "" {
			continue // skip blank rows the form may emit
		}
		s.Procs = append(s.Procs, items.ItemProc{Trigger: p.Trigger, Chance: p.Chance, CooldownRounds: p.CooldownRounds, Effect: p.Effect, Params: p.Params})
	}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./modules/gmcp/ -run RoundTripsAdvancedFields`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add modules/gmcp/gmcp.Item.go modules/gmcp/gmcp.Item_test.go
git commit -m "feat(build): advanced item fields in the Build.Item payload"
```

---

### Task 3: Ship valid trigger/effect/voice enums in Build.Item

**Files:**
- Modify: `modules/gmcp/gmcp.Item.go` (`itemDetail`, providers, `buildItemGet`, imports)
- Test: `modules/gmcp/gmcp.Item_test.go`

- [ ] **Step 1: Write the failing test**

Add to `modules/gmcp/gmcp.Item_test.go`:

```go
func TestBuildItemGet_ShipsAdvancedEnums(t *testing.T) {
	restore := itemvoices.SeedVoicesForTest(map[string]*itemvoices.VoiceSpec{"blackrazor": {VoiceId: "blackrazor"}})
	defer restore()
	w := newFakeItemWorld()
	w.specs[10005] = &items.ItemSpec{ItemId: 10005, Name: "Sword", Type: items.Weapon}
	d, ok := buildItemGet(w.deps(), 10005)
	if !ok {
		t.Fatal("expected found")
	}
	if len(d.ProcTriggers) == 0 || len(d.ProcEffects) == 0 {
		t.Error("detail must ship proc trigger/effect enums")
	}
	if len(d.Voices) == 0 || d.Voices[0] != "blackrazor" {
		t.Errorf("detail must ship the voice list, got %v", d.Voices)
	}
}
```

Add `"github.com/GoMudEngine/GoMud/internal/itemvoices"` to the test's imports.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./modules/gmcp/ -run ShipsAdvancedEnums`
Expected: FAIL — `itemDetail` has no `ProcTriggers`/`ProcEffects`/`Voices`.

- [ ] **Step 3: Add enum fields, providers, and populate them**

In `modules/gmcp/gmcp.Item.go`, add `"github.com/GoMudEngine/GoMud/internal/itemvoices"` to the imports.

Add to the `itemDetail` struct (after `VendorCats`):

```go
	ProcTriggers  []string              `json:"procTriggers"`
	ProcEffects   []string              `json:"procEffects"`
	Voices        []string              `json:"voices"`
```

Add providers near the other enum providers:

```go
func procTriggerIds() []string { return items.ValidProcTriggers() }
func procEffectIds() []string  { return items.ValidProcEffects() }
func itemVoiceIds() []string   { return itemvoices.AllVoiceIds() }
```

In `buildItemGet`, extend the returned `itemDetail`:

```go
	return itemDetail{
		itemUpdateReq: specToReq(s),
		Types:         itemTypeIds(), Subtypes: itemSubtypeIds(), Elements: itemElementIds(), Stats: statModNames(),
		VendorCats:   shops.ValidVendorCategories,
		ProcTriggers: procTriggerIds(), ProcEffects: procEffectIds(), Voices: itemVoiceIds(),
		Ranges: ranges,
	}, true
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./modules/gmcp/ -run ShipsAdvancedEnums`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add modules/gmcp/gmcp.Item.go modules/gmcp/gmcp.Item_test.go
git commit -m "feat(build): ship proc trigger/effect/voice enums in Build.Item"
```

---

### Task 4: Confirm the save guard rejects invalid procs

**Files:**
- Test: `internal/items/save_test.go`

The production save path (`realItemDeps.save = items.SaveItemSpec`) calls `spec.Validate()`, which rejects invalid procs. This task documents/locks that guard so a bad proc from the editor returns an error (red toast) rather than persisting.

- [ ] **Step 1: Write the failing test**

Add to `internal/items/save_test.go`:

```go
func TestSaveItemSpec_RejectsInvalidProc(t *testing.T) {
	dir := t.TempDir()
	pointItemsAt(t, dir)
	spec := ItemSpec{ItemId: 10009, Name: "Bad Blade", Type: Weapon, Description: "d", Hands: 1,
		NotSalable: true, DamageMultiplier: 1.0,
		Procs: []ItemProc{{Trigger: "on_wobble", Effect: "lifesteal", Chance: 50}}, // bad trigger
	}
	items[spec.ItemId] = &spec
	t.Cleanup(func() { delete(items, 10009) })
	if err := SaveItemSpec(spec); err == nil {
		t.Fatal("expected SaveItemSpec to reject an invalid proc trigger")
	}
}
```

- [ ] **Step 2: Run test to verify it passes immediately**

Run: `go test ./internal/items/ -run RejectsInvalidProc`
Expected: PASS (the guard already exists via `Validate`). This is a characterization test; if it FAILS, `SaveItemSpec` is not calling `Validate` — stop and fix that before continuing.

- [ ] **Step 3: Commit**

```bash
git add internal/items/save_test.go
git commit -m "test(items): SaveItemSpec rejects an invalid proc"
```

---

### Task 5: Frontend — collapsible Advanced section (non-proc fields)

**Files:**
- Modify: `_datafiles/html/public/static/js/items.js`

- [ ] **Step 1: Add the advanced-open state to the Panel object**

In the `Panel` object literal (top of file), add:

```js
    advancedOpen: false,
    _advItemId: 0,
```

- [ ] **Step 2: Add HINTS entries for the new fields**

In the `HINTS` map add:

```js
    reserveHealthPct: "fraction 0–1 of max HP locked while equipped",
    reserveStaminaPct: "fraction 0–1 of max stamina locked while equipped",
    reserveConvictionPct: "fraction 0–1 of max conviction locked while equipped",
    hungerRounds: "rounds without a kill before it feeds on the wielder (0 = never)",
    hungerDrainPct: "fraction 0–1 of max HP drained per hungry round",
    mutationTickInterval: "rounds between mutation rolls while worn (0 = never)",
    mutationTickChance: "percent chance per roll (0–100)",
    mutationRarityFloor: "min mutation rarity in the pool (0–10)",
    wornBuffIds: "comma-separated buff ids applied while worn",
    chance: "percent per trigger (1–100)",
    cooldownRounds: "internal cooldown in rounds (0 = none)",
```

- [ ] **Step 3: Build the Advanced section in `renderForm`**

In `renderForm`, replace the Save+delete row block's lead-in by inserting a call to `buildAdvancedSection` just before the Save row is created (i.e. after `rerenderTypeSections(detail.type);`):

```js
    // Advanced (sentient & procs) — collapsible, auto-open when populated.
    this.buildAdvancedSection(insp, detail, F, markDirty, field, numField, textField, checkField, selectField, hintFor);
```

Add this method after `Panel.buildTypeSections`:

```js
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

    var head = ce("h3", { text: (this.advancedOpen ? "▾ " : "▸ ") + "Advanced — sentient & procs" });
    head.style.cursor = "pointer";
    var body = ce("div", {});
    body.style.display = this.advancedOpen ? "" : "none";
    head.addEventListener("click", function () {
      self.advancedOpen = !self.advancedOpen;
      body.style.display = self.advancedOpen ? "" : "none";
      head.textContent = (self.advancedOpen ? "▾ " : "▸ ") + "Advanced — sentient & procs";
    });
    insp.appendChild(head);
    insp.appendChild(body);

    // Procs (Task 6 fills this in).
    this.buildProcEditor(body, detail, F, markDirty);

    // Reserves.
    body.appendChild(sectionTitle("Reserves"));
    body.appendChild(ce("div", { "class": "row" }, [
      numField("Reserve HP", "reserveHealthPct", detail.reserveHealthPct, "0.05"),
      numField("Reserve SP", "reserveStaminaPct", detail.reserveStaminaPct, "0.05")]));
    body.appendChild(numField("Reserve CP", "reserveConvictionPct", detail.reserveConvictionPct, "0.05"));

    // Sentient.
    body.appendChild(sectionTitle("Sentient"));
    body.appendChild(selectField("Voice", "voiceId", detail.voiceId, [""].concat(detail.voices || [])));
    body.appendChild(ce("div", { "class": "flags" }, [checkField("taunt-pull", "tauntPull", detail.tauntPull)]));

    // Hunger.
    body.appendChild(sectionTitle("Hunger"));
    body.appendChild(ce("div", { "class": "row" }, [
      numField("Hunger rounds", "hungerRounds", detail.hungerRounds),
      numField("Hunger drain", "hungerDrainPct", detail.hungerDrainPct, "0.01")]));

    // Mutation drip.
    body.appendChild(sectionTitle("Mutation drip"));
    body.appendChild(ce("div", { "class": "row" }, [
      numField("Tick interval", "mutationTickInterval", detail.mutationTickInterval),
      numField("Tick chance", "mutationTickChance", detail.mutationTickChance)]));
    body.appendChild(numField("Rarity floor", "mutationRarityFloor", detail.mutationRarityFloor));

    // Worn buffs.
    body.appendChild(sectionTitle("Worn buffs"));
    var wb = ce("input", { type: "text", placeholder: "comma buff ids" });
    wb.value = (detail.wornBuffIds || []).join(", ");
    wb.addEventListener("input", markDirty);
    F.wornBuffIds = function () { return wb.value.split(",").map(function (s) { return parseInt(s.trim(), 10); }).filter(function (n) { return !isNaN(n); }); };
    body.appendChild(field("Worn buff ids", wb, hintFor("wornBuffIds", false)));
  };
```

Note: `sectionTitle` is already a module-level helper in this file.

- [ ] **Step 4: Add a no-op `buildProcEditor` stub so the file parses**

Temporarily, so Step 5 can run before Task 6:

```js
  Panel.buildProcEditor = function (body, detail, F, markDirty) {
    F.procs = function () { return detail.procs || []; };
  };
```

- [ ] **Step 5: Add the non-proc fields to `gather()`**

In `Panel.gather`, before the closing `}` of the returned object (after `keyLockId`), add:

```js
      , reserveHealthPct: g("reserveHealthPct", 0), reserveStaminaPct: g("reserveStaminaPct", 0), reserveConvictionPct: g("reserveConvictionPct", 0)
      , voiceId: g("voiceId", ""), tauntPull: g("tauntPull", false)
      , hungerRounds: g("hungerRounds", 0), hungerDrainPct: g("hungerDrainPct", 0)
      , mutationTickInterval: g("mutationTickInterval", 0), mutationTickChance: g("mutationTickChance", 0), mutationRarityFloor: g("mutationRarityFloor", 0)
      , wornBuffIds: g("wornBuffIds", []), procs: g("procs", [])
```

(Match the existing object's trailing-comma style; if the last existing field has no trailing comma, merge these onto that line instead.)

- [ ] **Step 6: Verify syntax**

Run: `node --check _datafiles/html/public/static/js/items.js`
Expected: no output (valid).

- [ ] **Step 7: Commit**

```bash
git add _datafiles/html/public/static/js/items.js
git commit -m "feat(build): collapsible Advanced item section (reserves/hunger/mutation/voice/worn)"
```

---

### Task 6: Frontend — proc row editor with nested params

**Files:**
- Modify: `_datafiles/html/public/static/js/items.js`

- [ ] **Step 1: Replace the `buildProcEditor` stub with the real editor**

```js
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
```

- [ ] **Step 2: Add the small local helpers used above**

Add near the top of the IIFE (after `ce`):

```js
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
```

- [ ] **Step 3: Verify syntax**

Run: `node --check _datafiles/html/public/static/js/items.js`
Expected: no output (valid).

- [ ] **Step 4: Commit**

```bash
git add _datafiles/html/public/static/js/items.js
git commit -m "feat(build): proc row editor with nested key->value params"
```

---

### Task 7: Full verification + live Blackrazor smoke

**Files:** none (verification only). Follow the local-dev conventions: `config.yaml` `HttpPort` is skip-worktree'd; boot a test server on port 8091 without disturbing the user's 8090 server, and restore the port after the server reads it.

- [ ] **Step 1: gofmt + build + unit tests**

Run:
```bash
gofmt -w modules/gmcp/gmcp.Item.go internal/items/*.go internal/itemvoices/*.go
gofmt -l modules/gmcp/gmcp.Item.go internal/items/*.go internal/itemvoices/*.go   # expect empty
go build ./...
go test ./internal/items/... ./internal/itemvoices/... ./modules/gmcp/...
node --check _datafiles/html/public/static/js/items.js
```
Expected: gofmt clean, build clean, tests PASS, JS valid.

- [ ] **Step 2: Boot a test server on 8091 and confirm clean boot**

```bash
sed -i 's/^  HttpPort: 8090$/  HttpPort: 8091/' _datafiles/config.yaml
go build -o buildsrv.exe .
./buildsrv.exe > "$LOCALAPPDATA/Temp/claude/itemsrv.log" 2>&1 &
# once "Starting http server" appears, restore the port:
sed -i 's/^  HttpPort: 8091$/  HttpPort: 8090/' _datafiles/config.yaml
# wait for "Server Ready"; confirm no panic.
```

- [ ] **Step 3: Live Blackrazor round-trip smoke**

Write a throwaway WS smoke under `tools/_buildsmoke/` (mirroring the earlier item smokes): log in as an admin, `Build.Item.Get {itemId:40183}`, and assert the returned `Build.Item` carries the proc (`trigger:"on_hit"`, `effect:"lifesteal"`, `params.ratio:0.25`), `voiceId:"blackrazor"`, `reserveHealthPct:0.25`, `hungerRounds:50`, and non-empty `procTriggers`/`procEffects`/`voices`. Then `Build.Item.Update` the same payload back and assert `Build.Result.ok`, and a follow-up `Build.Item.Get` still shows the proc — proving persistence. Remove `tools/_buildsmoke` and `buildsrv.exe` after.

Run: `go run ./tools/_buildsmoke --user <admin> --pass <pw> --port 8091 --get 40183`
Expected: `OK` line showing the proc + voice + reserve round-tripped.

- [ ] **Step 4: Confirm the test server still boots clean after the round-trip**

Check the log for `Server Ready` with no `panic` / `ValidateVendorCategories` / `casing:` — the Blackrazor save must not brick boot. Kill the server (`taskkill //F //IM buildsrv.exe`), remove `buildsrv.exe`.

- [ ] **Step 5: Browser playtest gate (owed to the user)**

Per the DOGMud content SOP, hand to the user for a browser eyeball: open a plain item (Advanced collapsed), open the Blackrazor (Advanced auto-open showing its proc/voice/reserve/hunger), add a proc with a param + save + reopen (persists), add an invalid proc (bad chance) + save (clear red toast, nothing saved), collapse/expand the section.

- [ ] **Step 6: Final commit (if any verification fixups were needed)**

```bash
git add -A
git commit -m "chore(build): advanced item behaviors verified (blackrazor round-trip + boot-clean)"
```

---

## Self-review

- **Spec coverage:** Procs (Task 2/6), reserves/hunger/mutation/voice/taunt/worn-buffs (Task 2/5), enums (Task 3), collapsible auto-open panel (Task 5), generic key→value params (Task 6), save-guard via Validate (Task 4), Blackrazor smoke (Task 7). All spec sections mapped.
- **Type consistency:** `procRow` JSON keys match the frontend gather (`trigger/effect/chance/cooldownRounds/params`); `itemDetail` enum fields (`procTriggers/procEffects/voices`) match the frontend reads (`detail.procTriggers` etc.); `reqToSpec` maps `procRow → items.ItemProc` with the correct field order.
- **No placeholders:** every code step shows the actual code. The Task 5 `buildProcEditor` stub is intentional and replaced in Task 6.
