# Item Authoring (2) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** An **Items** tab in the `/build` page where an admin browses/searches item templates, edits any field with a type-aware form, creates and deletes items (delete gated by a world-wide reference scan), all persisting to item YAML and live in-game.

**Architecture:** Reuse 1b's admin `/build` GMCP session. New RoleAdmin-gated `Build.Item.*` packages run on MainWorker via the existing `GMCPBuildOp` event, mutating item templates through the `items` package and replying with `Build.Item`/`Build.Items`/`Build.Result`. Create reuses `items.CreateNewItemFile`; update/delete add `items.SaveItemSpec`/`DeleteItemSpec`. The delete reference scan lives in the gmcp layer (avoids an items↔mobs/rooms/quests import cycle) behind an injected seam so it's unit-testable.

**Tech Stack:** Go (GoMud fork), GMCP over WebSocket, vanilla JS (matching `builder.js`/`build.html`), `go test`.

**Spec:** `docs/superpowers/specs/2026-07-23-item-authoring-2-design.md`

**Verified seams:**
- `ItemSpec` struct `internal/items/itemspec.go:254`; weapon damage = `DamageMultiplier` float (Damage dice struct is legacy); `RarityTier`, `PhysicalMitigation`/`MagicalMitigation`/`ConvictionMitigation`, `SpellDamageMultiplier`, `StatMods` (`statmods.StatMods = map[string]int`), `AgingThresholds` (`aging.go:15`: FermentRounds/PeakRounds/DecayRounds/SpoilRounds), `SalvageReturn` (`itemspec.go:339`: ItemTag/Quantity).
- `items.CreateNewItemFile(ItemSpec)(int,error)` `newitemfile.go:10`; `getNextItemId(type)` `:47`; `ItemSpec.Filepath()`=`ItemFolder()+"/"+Filename()`; `Filename()`=`<id>-<ConvertForFilename(name)>.yaml`; armor subfolders `armor-20000/<type>`.
- `items.GetAllItemSpecs()[]ItemSpec` `:467`; `GetItemSpec(id)*ItemSpec` `:667`; `ItemTypes()`/`ItemSubtypes()` `:41`/`:80`; the package map `items map[int]*ItemSpec` `:29`.
- Reference sources: `mobs.AllMobTemplates()[]*Mob` (`Mob.LootPool`/`PrizeItemIds`/`AcceptedItemIds`/`CrafterRestockMaterials` []int); `crafting.GetAllRecipes()map[string]*RecipeSpec` + `RecipeSpec.Output.ItemId` + `Ingredients[].ItemId`; `quests.GetAllQuests()[]Quest` + `Quest.Rewards.ItemId`/`ItemInfo`; `rooms.GetAllRoomIds()`+`LoadRoom(id).Containers[name].Items[].ItemId` + `.Recipes`.
- 1b patterns to MIRROR: `modules/gmcp/gmcp.Build.go` (`BuildResult`, `requireAdmin`, `buildDeps` seam, `GMCPBuildOp`, `handleBuildOp`, `sendBuildResult`); `gmcp.go` HandleIAC grouped `Build.*` dispatch case (`gmcp.go:485` area); `build.html` toolbar/inspector/`onGMCP`; `builder.js`.

---

## File Structure

- `internal/items/save.go` *(new)* — `SaveItemSpec`, `DeleteItemSpec`. (Task 1)
- `internal/items/save_test.go` *(new)* — tests. (Task 1)
- `modules/gmcp/gmcp.Item.go` *(new)* — `Build.Item.*` payloads, detail, `itemDeps` seam, core funcs, reference scan, senders, `handleBuildOp` cases. (Tasks 2,3)
- `modules/gmcp/gmcp.Item_test.go` *(new)* — unit tests. (Tasks 2,3)
- `modules/gmcp/gmcp.go` — add `Build.Item.*` to the deferred dispatch + route in `handleBuildOp`. (Task 2)
- `_datafiles/html/public/static/js/items.js` *(new)* — items list + type-aware form. (Tasks 4,5)
- `_datafiles/html/public/build.html` — `Rooms | Items` toggle, load `items.js`, route `Build.Item*` GMCP. (Tasks 4,5)

---

## Task 1: Item persistence seams — `SaveItemSpec` + `DeleteItemSpec`

**Files:** Create `internal/items/save.go`, `internal/items/save_test.go`.

- [ ] **Step 1: Failing test.** The one non-trivial behavior is **file relocation on rename/retype** (item filenames embed the name; armor is per-type-subfoldered). Test that a rename removes the old file and writes the new, and the cache updates. Use `t.TempDir()` as the data root.

```go
package items

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

// pointDataFilesAt sets FilePaths.DataFiles to dir for the duration of a test.
// SetVal persists to the override file; capture + restore the prior value.
func pointDataFilesAt(t *testing.T, dir string) {
	t.Helper()
	prev := configs.GetFilePathsConfig().DataFiles.String()
	if err := configs.SetVal("FilePaths.DataFiles", dir); err != nil {
		t.Fatalf("set DataFiles: %v", err)
	}
	// items live under <dir>/items/<folder>/
	if err := os.MkdirAll(filepath.Join(dir, "items"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { configs.SetVal("FilePaths.DataFiles", prev) })
}

func TestSaveItemSpec_RelocatesFileOnRename(t *testing.T) {
	dir := t.TempDir()
	pointDataFilesAt(t, dir)

	// Seed a weapon (id 10001) in the cache + on disk.
	spec := ItemSpec{ItemId: 10001, Name: "Iron Sword", Type: Weapon, Description: "A sword.", Hands: 1, DamageMultiplier: 0.8}
	items[spec.ItemId] = &spec
	if err := SaveItemSpec(spec); err != nil {
		t.Fatalf("initial save: %v", err)
	}
	oldPath := filepath.Join(dir, "items", spec.Filepath())
	if _, err := os.Stat(oldPath); err != nil {
		t.Fatalf("expected %s to exist: %v", oldPath, err)
	}

	// Rename -> filename changes -> old file must be removed, new written.
	renamed := spec
	renamed.Name = "Steel Sword"
	if err := SaveItemSpec(renamed); err != nil {
		t.Fatalf("rename save: %v", err)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Errorf("old file %s should be gone after rename", oldPath)
	}
	newPath := filepath.Join(dir, "items", renamed.Filepath())
	if _, err := os.Stat(newPath); err != nil {
		t.Errorf("new file %s should exist: %v", newPath, err)
	}
	if items[10001].Name != "Steel Sword" {
		t.Errorf("cache not updated: %q", items[10001].Name)
	}

	delete(items, 10001) // clean the package-global cache
}

func TestDeleteItemSpec_RemovesFileAndCache(t *testing.T) {
	dir := t.TempDir()
	pointDataFilesAt(t, dir)
	spec := ItemSpec{ItemId: 10002, Name: "Bronze Dagger", Type: Weapon, Description: "A dagger.", Hands: 1, DamageMultiplier: 0.5}
	items[spec.ItemId] = &spec
	if err := SaveItemSpec(spec); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "items", spec.Filepath())
	if _, err := os.Stat(p); err != nil {
		t.Fatal(err)
	}
	if err := DeleteItemSpec(10002); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Errorf("file %s should be gone", p)
	}
	if _, ok := items[10002]; ok {
		t.Error("cache entry should be gone")
	}
}
```

- [ ] **Step 2: Run — FAIL** (undefined SaveItemSpec/DeleteItemSpec). `go test ./internal/items/ -run 'TestSaveItemSpec_RelocatesFileOnRename|TestDeleteItemSpec_RemovesFileAndCache'`

- [ ] **Step 3: Implement `save.go`.**

```go
package items

import (
	"fmt"
	"os"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// itemsBasePath is the root SaveFlatFile joins with ItemSpec.Filepath().
func itemsBasePath() string {
	return configs.GetFilePathsConfig().DataFiles.String() + `/items`
}

// SaveItemSpec validates and writes an existing/edited item template. Item
// filenames embed the name (<id>-<name>.yaml) and armor lives in per-type
// subfolders, so a rename or retype changes the path — the OLD file is removed
// first so a stale duplicate can't linger (and boot as a second copy). Updates
// the in-memory cache.
func SaveItemSpec(spec ItemSpec) error {
	if err := spec.Validate(); err != nil {
		return err
	}
	base := itemsBasePath()
	newRel := spec.Filepath()

	if old, ok := items[spec.ItemId]; ok {
		if oldRel := old.Filepath(); oldRel != newRel {
			oldFull := util.FilePath(base + `/` + oldRel)
			if err := os.Remove(oldFull); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}

	saveModes := []fileloader.SaveOption{}
	if configs.GetFilePathsConfig().CarefulSaveFiles {
		saveModes = append(saveModes, fileloader.SaveCareful)
	}
	if err := fileloader.SaveFlatFile[*ItemSpec](base, &spec, saveModes...); err != nil {
		return err
	}
	cp := spec
	items[spec.ItemId] = &cp
	return nil
}

// DeleteItemSpec removes an item template's file and cache entry. Callers must
// first confirm the item is unreferenced (the web builder's reference scan).
func DeleteItemSpec(itemId int) error {
	spec, ok := items[itemId]
	if !ok {
		return fmt.Errorf(`item %d not found`, itemId)
	}
	full := util.FilePath(itemsBasePath() + `/` + spec.Filepath())
	if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
		return err
	}
	delete(items, itemId)
	return nil
}
```

- [ ] **Step 4: Run — PASS.** `go test ./internal/items/ -run 'TestSaveItemSpec_RelocatesFileOnRename|TestDeleteItemSpec_RemovesFileAndCache'`

- [ ] **Step 5: Run full items tests + gofmt.** `go test ./internal/items/ && gofmt -l internal/items/save.go` (expect empty).

- [ ] **Step 6: Commit.**
```bash
git add internal/items/save.go internal/items/save_test.go
git commit -m "feat(items): SaveItemSpec + DeleteItemSpec (builder persistence)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: `Build.Item.*` GMCP core — list / get / create / update

**Files:** Create `modules/gmcp/gmcp.Item.go`, `modules/gmcp/gmcp.Item_test.go`; Modify `modules/gmcp/gmcp.go`.

Mirror `gmcp.Build.go`: reuse `BuildResult`, `requireAdmin`, `sendBuildResult`, and the `GMCPBuildOp` → `handleBuildOp` deferral. Item mutations touch the shared `items` map + files, so they MUST run on MainWorker (never the connection goroutine).

- [ ] **Step 1: Failing tests** for the core seams behind an `itemDeps` struct (fake world). Cover: create assigns an id and returns detail; update round-trips fields + relocates via save; get maps fields; non-admin refused (pure `roleIsAdmin`, already in gmcp.Build.go — just assert an item op path respects it in Task 4 routing, not here).

```go
package gmcp

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

type fakeItemWorld struct {
	specs   map[int]*items.ItemSpec
	saved   []items.ItemSpec
	deleted []int
	nextId  int
}

func newFakeItemWorld() *fakeItemWorld {
	return &fakeItemWorld{specs: map[int]*items.ItemSpec{}, nextId: 10000}
}

func (w *fakeItemWorld) deps() itemDeps {
	return itemDeps{
		load: func(id int) *items.ItemSpec { return w.specs[id] },
		save: func(s items.ItemSpec) error {
			w.saved = append(w.saved, s)
			cp := s
			w.specs[s.ItemId] = &cp
			return nil
		},
		del: func(id int) error { w.deleted = append(w.deleted, id); delete(w.specs, id); return nil },
		create: func(s items.ItemSpec) (int, error) {
			w.nextId++
			s.ItemId = w.nextId
			cp := s
			w.specs[s.ItemId] = &cp
			return s.ItemId, nil
		},
		references: func(id int) []itemRef { return nil }, // Task 3 exercises this
	}
}

func TestBuildItemUpdate_RoundTripsFields(t *testing.T) {
	w := newFakeItemWorld()
	w.specs[10001] = &items.ItemSpec{ItemId: 10001, Name: "Old", Type: items.Weapon, Description: "d", Hands: 1}
	res := buildItemUpdate(w.deps(), itemUpdateReq{
		ItemId: 10001, Name: "Keen Blade", Type: string(items.Weapon), Description: "Sharp.",
		Value: 55, Weight: 3, DamageMultiplier: 0.9, Hands: 1, RarityTier: 30,
	})
	if !res.Ok {
		t.Fatalf("update should succeed, got %+v", res)
	}
	if len(w.saved) != 1 {
		t.Fatalf("expected 1 save, got %d", len(w.saved))
	}
	got := w.saved[0]
	if got.Name != "Keen Blade" || got.Description != "Sharp." || got.DamageMultiplier != 0.9 || got.RarityTier != 30 {
		t.Errorf("fields not round-tripped: %+v", got)
	}
}

func TestBuildItemCreate_AssignsIdAndReturnsDetail(t *testing.T) {
	w := newFakeItemWorld()
	res := buildItemCreate(w.deps(), string(items.Weapon))
	if !res.Ok || res.ItemId == 0 {
		t.Fatalf("create should return an id, got %+v", res)
	}
	if w.specs[res.ItemId] == nil {
		t.Fatal("created item not stored")
	}
}

func TestBuildItemGet_MapsFields(t *testing.T) {
	w := newFakeItemWorld()
	w.specs[10005] = &items.ItemSpec{ItemId: 10005, Name: "Hammer", Type: items.Weapon, Subtype: items.Bludgeoning,
		Description: "Heavy.", Value: 40, Weight: 6, DamageMultiplier: 1.1, Hands: 2, RarityTier: 20}
	d, ok := buildItemGet(w.deps(), 10005)
	if !ok {
		t.Fatal("expected found")
	}
	if d.Name != "Hammer" || d.Type != string(items.Weapon) || d.DamageMultiplier != 1.1 || d.RarityTier != 20 {
		t.Errorf("detail wrong: %+v", d)
	}
}
```

- [ ] **Step 2: Run — FAIL.** `go test ./modules/gmcp/ -run TestBuildItem`

- [ ] **Step 3a: Extend `BuildResult`** (in `gmcp.Build.go`) with an item id field the item results use:
```go
	ItemId int `json:"itemId,omitempty"` // e.g. the created/updated item
```
(The `Refs []itemRef` field is added in Task 3.)

- [ ] **Step 3b: Implement `gmcp.Item.go`** (payloads, detail, seam, list/get/create/update — delete added in Task 3).

```go
package gmcp

import (
	"sort"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/statmods"
)

// ---- client -> server payloads ----
type itemGetReq struct{ ItemId int `json:"itemId"` }
type itemCreateReq struct{ Type string `json:"type"` }
type itemDeleteReq struct{ ItemId int `json:"itemId"` }
type itemUpdateReq struct {
	ItemId      int    `json:"itemId"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	NameSimple  string `json:"nameSimple"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Subtype     string `json:"subtype"`
	Value       int    `json:"value"`
	Weight      float64 `json:"weight"`
	Uses        int    `json:"uses"`
	Quantity    int    `json:"quantity"`
	RarityTier  int    `json:"rarityTier"`
	VendorCategories []string `json:"vendorCategories"`
	NotSalable  bool `json:"notSalable"`
	NeverDrops  bool `json:"neverDrops"`
	Restricted  bool `json:"restricted"`
	Cursed      bool `json:"cursed"`
	QuestToken  string `json:"questToken"`
	StatMods    map[string]int `json:"statMods"`
	// weapon
	DamageMultiplier      float64 `json:"damageMultiplier"`
	SpellDamageMultiplier float64 `json:"spellDamageMultiplier"`
	Hands       int     `json:"hands"`
	ParryRating int     `json:"parryRating"`
	SpeedMultiplier float64 `json:"speedMultiplier"`
	StaminaCost int     `json:"staminaCost"`
	WaitRounds  int     `json:"waitRounds"`
	MinStrength int     `json:"minStrength"`
	Reach       float64 `json:"reach"`
	GrappleModifier float64 `json:"grappleModifier"`
	Element     string  `json:"element"`
	AmmoTag     string  `json:"ammoTag"`
	// armor
	PhysicalMitigation   int     `json:"physicalMitigation"`
	MagicalMitigation    int     `json:"magicalMitigation"`
	ConvictionMitigation int     `json:"convictionMitigation"`
	BlockRating int     `json:"blockRating"`
	EscapeModifier float64 `json:"escapeModifier"`
	// consumable
	BuffIds  []int `json:"buffIds"`
	Toxicity int   `json:"toxicity"`
	FermentRounds int `json:"fermentRounds"`
	PeakRounds    int `json:"peakRounds"`
	DecayRounds   int `json:"decayRounds"`
	SpoilRounds   int `json:"spoilRounds"`
	BottleAgingMultiplier float64 `json:"bottleAgingMultiplier"`
	IsBandolier bool `json:"isBandolier"`
	BandolierCapacity int `json:"bandolierCapacity"`
	// component
	IsComponent bool    `json:"isComponent"`
	ComponentTag string `json:"componentTag"`
	WeightReduction float64 `json:"weightReduction"`
	BagCapacity int     `json:"bagCapacity"`
	SalvageReturns []itemSalvageRow `json:"salvageReturns"`
	// key
	KeyLockId string `json:"keyLockId"`
}
type itemSalvageRow struct {
	ItemTag  string `json:"itemTag"`
	Quantity int    `json:"quantity"`
}

// ---- server -> client detail (Build.Item) ----
type itemDetail struct {
	itemUpdateReq          // same field set, echoed back
	Types    []string `json:"types"`
	Subtypes []string `json:"subtypes"`
	Elements []string `json:"elements"`
	Stats    []string `json:"stats"`
}

// itemListRow is one row of Build.Items.
type itemListRow struct {
	Id      int    `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Subtype string `json:"subtype"`
	Rarity  int    `json:"rarity"`
}

// ---- dependency seam ----
type itemRef struct {
	Kind string `json:"kind"` // mob | recipe | quest | container
	Id   string `json:"id"`   // e.g. "mob 9538", "quest 10"
}
type itemDeps struct {
	load       func(id int) *items.ItemSpec
	save       func(s items.ItemSpec) error
	del        func(id int) error
	create     func(s items.ItemSpec) (int, error)
	references func(id int) []itemRef
}

func realItemDeps() itemDeps {
	return itemDeps{
		load:       items.GetItemSpec,
		save:       items.SaveItemSpec,
		del:        items.DeleteItemSpec,
		create:     items.CreateNewItemFile,
		references: scanItemReferences, // Task 3
	}
}

// ---- core ----
func buildItemList(d itemDeps) []itemListRow {
	all := items.GetAllItemSpecs()
	rows := make([]itemListRow, 0, len(all))
	for _, s := range all {
		rows = append(rows, itemListRow{s.ItemId, s.Name, string(s.Type), string(s.Subtype), s.RarityTier})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Id < rows[j].Id })
	return rows
}

func specToReq(s *items.ItemSpec) itemUpdateReq {
	req := itemUpdateReq{
		ItemId: s.ItemId, Name: s.Name, DisplayName: s.DisplayName, NameSimple: s.NameSimple,
		Description: s.Description, Type: string(s.Type), Subtype: string(s.Subtype),
		Value: s.Value, Weight: s.Weight, Uses: s.Uses, Quantity: s.Quantity, RarityTier: s.RarityTier,
		VendorCategories: s.VendorCategories, NotSalable: s.NotSalable, NeverDrops: s.NeverDrops,
		Restricted: s.Restricted, Cursed: s.Cursed, QuestToken: s.QuestToken, StatMods: s.StatMods,
		DamageMultiplier: s.DamageMultiplier, SpellDamageMultiplier: s.SpellDamageMultiplier,
		Hands: s.Hands, ParryRating: s.ParryRating, SpeedMultiplier: s.SpeedMultiplier,
		StaminaCost: s.StaminaCost, WaitRounds: s.WaitRounds, MinStrength: s.MinStrength,
		Reach: s.Reach, GrappleModifier: s.GrappleModifier, Element: string(s.Element), AmmoTag: s.AmmoTag,
		PhysicalMitigation: s.PhysicalMitigation, MagicalMitigation: s.MagicalMitigation,
		ConvictionMitigation: s.ConvictionMitigation, BlockRating: s.BlockRating, EscapeModifier: s.EscapeModifier,
		BuffIds: s.BuffIds, Toxicity: s.Toxicity,
		FermentRounds: s.Aging.FermentRounds, PeakRounds: s.Aging.PeakRounds,
		DecayRounds: s.Aging.DecayRounds, SpoilRounds: s.Aging.SpoilRounds,
		BottleAgingMultiplier: s.BottleAgingMultiplier, IsBandolier: s.IsBandolier, BandolierCapacity: s.BandolierCapacity,
		IsComponent: s.IsComponent, ComponentTag: s.ComponentTag, WeightReduction: s.WeightReduction,
		BagCapacity: s.BagCapacity, KeyLockId: s.KeyLockId,
	}
	for _, sr := range s.SalvageReturns {
		req.SalvageReturns = append(req.SalvageReturns, itemSalvageRow{sr.ItemTag, sr.Quantity})
	}
	return req
}

func buildItemGet(d itemDeps, itemId int) (itemDetail, bool) {
	s := d.load(itemId)
	if s == nil {
		return itemDetail{}, false
	}
	return itemDetail{itemUpdateReq: specToReq(s), Types: itemTypeIds(), Subtypes: itemSubtypeIds(), Elements: itemElementIds(), Stats: statModNames()}, true
}

func reqToSpec(base *items.ItemSpec, req itemUpdateReq) items.ItemSpec {
	s := *base // preserve fields the form doesn't cover (procs, reserves, buffids-worn, etc.)
	s.Name, s.DisplayName, s.NameSimple, s.Description = req.Name, req.DisplayName, req.NameSimple, req.Description
	s.Type, s.Subtype = items.ItemType(req.Type), items.ItemSubType(req.Subtype)
	s.Value, s.Weight, s.Uses, s.Quantity, s.RarityTier = req.Value, req.Weight, req.Uses, req.Quantity, req.RarityTier
	s.VendorCategories, s.QuestToken, s.StatMods = req.VendorCategories, req.QuestToken, statmods.StatMods(req.StatMods)
	s.NotSalable, s.NeverDrops, s.Restricted, s.Cursed = req.NotSalable, req.NeverDrops, req.Restricted, req.Cursed
	s.DamageMultiplier, s.SpellDamageMultiplier, s.Hands = req.DamageMultiplier, req.SpellDamageMultiplier, req.Hands
	s.ParryRating, s.SpeedMultiplier, s.StaminaCost, s.WaitRounds = req.ParryRating, req.SpeedMultiplier, req.StaminaCost, req.WaitRounds
	s.MinStrength, s.Reach, s.GrappleModifier, s.Element, s.AmmoTag = req.MinStrength, req.Reach, req.GrappleModifier, items.Element(req.Element), req.AmmoTag
	s.PhysicalMitigation, s.MagicalMitigation, s.ConvictionMitigation = req.PhysicalMitigation, req.MagicalMitigation, req.ConvictionMitigation
	s.BlockRating, s.EscapeModifier = req.BlockRating, req.EscapeModifier
	s.BuffIds, s.Toxicity = req.BuffIds, req.Toxicity
	s.Aging = items.AgingThresholds{FermentRounds: req.FermentRounds, PeakRounds: req.PeakRounds, DecayRounds: req.DecayRounds, SpoilRounds: req.SpoilRounds}
	s.BottleAgingMultiplier, s.IsBandolier, s.BandolierCapacity = req.BottleAgingMultiplier, req.IsBandolier, req.BandolierCapacity
	s.IsComponent, s.ComponentTag, s.WeightReduction, s.BagCapacity = req.IsComponent, req.ComponentTag, req.WeightReduction, req.BagCapacity
	s.KeyLockId = req.KeyLockId
	s.SalvageReturns = nil
	for _, r := range req.SalvageReturns {
		if r.ItemTag != "" {
			s.SalvageReturns = append(s.SalvageReturns, items.SalvageReturn{ItemTag: r.ItemTag, Quantity: r.Quantity})
		}
	}
	return s
}

func buildItemUpdate(d itemDeps, req itemUpdateReq) BuildResult {
	base := d.load(req.ItemId)
	if base == nil {
		return buildErr("item %d not found", req.ItemId)
	}
	if req.Name == "" || req.Type == "" {
		return buildErr("name and type are required")
	}
	if err := d.save(reqToSpec(base, req)); err != nil {
		return buildErr("could not save item %d: %s", req.ItemId, err.Error())
	}
	return BuildResult{Ok: true, ItemId: req.ItemId}
}

func buildItemCreate(d itemDeps, itemType string) BuildResult {
	if itemType == "" {
		return buildErr("choose an item type")
	}
	seed := items.ItemSpec{Type: items.ItemType(itemType), Name: "new item", Description: "An unfinished item.", Hands: 1}
	id, err := d.create(seed)
	if err != nil {
		return buildErr("%s", err.Error())
	}
	return BuildResult{Ok: true, ItemId: id}
}

// enum providers
func itemTypeIds() []string {
	out := []string{}
	for _, t := range items.ItemTypes() {
		out = append(out, t.Type)
	}
	return dedupeSorted(out)
}
func itemSubtypeIds() []string {
	out := []string{}
	for _, t := range items.ItemSubtypes() {
		out = append(out, t.Type)
	}
	return dedupeSorted(out)
}
func itemElementIds() []string {
	return []string{"", "fire", "water", "ice", "electricity", "acid", "life", "death"}
}
func statModNames() []string {
	return []string{"strength", "dexterity", "perception", "vitality", "willpower", "charisma",
		"weapon-combat", "unarmed-combat", "ranged-combat", "spellcasting", "rhetoric", "manifestation", "skullduggery", "salvage"}
}
func dedupeSorted(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

// ---- senders ----
func sendItemList(uid int)              { events.AddToQueue(GMCPOut{UserId: uid, Module: "Build.Items", Payload: buildItemList(realItemDeps())}) }
func sendItemDetail(uid int, d itemDetail) { events.AddToQueue(GMCPOut{UserId: uid, Module: "Build.Item", Payload: d}) }
```

> **Implementer note:** `ItemSubtypes()` may not expose every subtype constant (unarmed/bite/etc.); if the form needs the full list, extend `itemSubtypeIds()` with the constants from `itemspec.go:145-171`. Confirm `crafting.GetAllRecipes` exists (Task 3) — it returns `allRecipes`; if the exported name differs, add a thin `GetAllRecipes()` accessor.

- [ ] **Step 4: Run item core tests — PASS.** `go test ./modules/gmcp/ -run TestBuildItem`

- [ ] **Step 5: Route in `handleBuildOp` + HandleIAC.** In `gmcp.Build.go` `handleBuildOp`'s switch add cases (item ops that only READ still go through MainWorker for consistency + safe map access):

```go
	case `Build.Item.List`:
		sendItemList(uid)
	case `Build.Item.Get`:
		var req itemGetReq
		if json.Unmarshal(evt.Payload, &req) != nil { sendBuildResult(uid, buildErr("bad Build.Item.Get payload")); break }
		if d, ok := buildItemGet(realItemDeps(), req.ItemId); ok { sendItemDetail(uid, d) } else { sendBuildResult(uid, buildErr("item %d not found", req.ItemId)) }
	case `Build.Item.Create`:
		var req itemCreateReq
		if json.Unmarshal(evt.Payload, &req) != nil { sendBuildResult(uid, buildErr("bad Build.Item.Create payload")); break }
		res := buildItemCreate(realItemDeps(), req.Type)
		sendBuildResult(uid, res)
		sendItemList(uid)
	case `Build.Item.Update`:
		var req itemUpdateReq
		if json.Unmarshal(evt.Payload, &req) != nil { sendBuildResult(uid, buildErr("bad Build.Item.Update payload")); break }
		sendBuildResult(uid, buildItemUpdate(realItemDeps(), req))
		sendItemList(uid)
	// Build.Item.Delete — Task 3
```

Then in `gmcp.go` add the new commands to the deferred `Build.*` dispatch case list: `Build.Item.List`, `Build.Item.Get`, `Build.Item.Create`, `Build.Item.Update`, `Build.Item.Delete`.

- [ ] **Step 6: Build + tests + gofmt.** `go build ./... && go test ./modules/gmcp/ && gofmt -l modules/gmcp/gmcp.Item.go`

- [ ] **Step 7: Commit.**
```bash
git add modules/gmcp/gmcp.Item.go modules/gmcp/gmcp.Item_test.go modules/gmcp/gmcp.go
git commit -m "feat(gmcp): Build.Item.* list/get/create/update (admin-gated)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: Reference scan + `Build.Item.Delete`

**Files:** Modify `modules/gmcp/gmcp.Item.go`, `modules/gmcp/gmcp.Item_test.go`, `gmcp.Build.go` (route).

The scan lives in gmcp (imports items+mobs+rooms+quests+crafting without a cycle). `buildItemDelete` uses the injected `references` seam so it's unit-testable; `scanItemReferences` is the real implementation.

- [ ] **Step 1: Failing tests.** (a) `buildItemDelete` blocks when refs exist and deletes when clean; (b) `scanItemReferences` finds a mob LootPool ref via injected iterators.

```go
func TestBuildItemDelete_BlocksWhenReferenced(t *testing.T) {
	w := newFakeItemWorld()
	w.specs[10001] = &items.ItemSpec{ItemId: 10001, Name: "Used Sword", Type: items.Weapon}
	d := w.deps()
	d.references = func(id int) []itemRef { return []itemRef{{Kind: "mob", Id: "mob 9538"}} }
	res := buildItemDelete(d, 10001)
	if res.Ok {
		t.Fatal("delete should be blocked when referenced")
	}
	if len(w.deleted) != 0 {
		t.Error("nothing should be deleted when blocked")
	}
	if len(res.Refs) != 1 || res.Refs[0].Id != "mob 9538" {
		t.Errorf("blocked result must carry the references, got %+v", res.Refs)
	}
}

func TestBuildItemDelete_DeletesWhenClean(t *testing.T) {
	w := newFakeItemWorld()
	w.specs[10002] = &items.ItemSpec{ItemId: 10002, Name: "Unused", Type: items.Weapon}
	res := buildItemDelete(w.deps(), 10002) // fake references() returns nil
	if !res.Ok {
		t.Fatalf("clean delete should succeed, got %+v", res)
	}
	if len(w.deleted) != 1 || w.deleted[0] != 10002 {
		t.Errorf("expected delete of 10002, got %+v", w.deleted)
	}
}

func TestScanItemReferences_FindsMobLootPool(t *testing.T) {
	refs := scanItemReferencesWith(40163, refIterators{
		mobs: func(yield func(mobRef)) {
			yield(mobRef{id: 9538, name: "a wolf", ids: []int{40163}})
		},
	})
	if len(refs) != 1 || refs[0].Kind != "mob" {
		t.Fatalf("expected one mob ref, got %+v", refs)
	}
}
```

- [ ] **Step 2: Run — FAIL.**

- [ ] **Step 3: Implement.** Add `Refs` to `BuildResult` (in `gmcp.Build.go`): `Refs []itemRef \`json:"refs,omitempty"\``. Then in `gmcp.Item.go`:

```go
func buildItemDelete(d itemDeps, itemId int) BuildResult {
	if d.load(itemId) == nil {
		return buildErr("item %d not found", itemId)
	}
	if refs := d.references(itemId); len(refs) > 0 {
		return BuildResult{Error: "item is still referenced — remove these first", Refs: refs}
	}
	if err := d.del(itemId); err != nil {
		return buildErr("could not delete item %d: %s", itemId, err.Error())
	}
	return BuildResult{Ok: true, ItemId: itemId}
}

// refIterators lets the scan be tested without the real world packages.
type mobRef struct{ id int; name string; ids []int }
type refIterators struct {
	mobs      func(yield func(mobRef))
	recipes   func(yield func(recipeId string, outputId int, ingredientIds []int))
	quests    func(yield func(token string, rewardIds []int))
	containers func(yield func(roomId int, itemIds []int))
}

func scanItemReferencesWith(itemId int, it refIterators) []itemRef {
	out := []itemRef{}
	if it.mobs != nil {
		it.mobs(func(m mobRef) {
			for _, id := range m.ids {
				if id == itemId {
					out = append(out, itemRef{Kind: "mob", Id: fmt.Sprintf("mob %d (%s)", m.id, m.name)})
					break
				}
			}
		})
	}
	if it.recipes != nil {
		it.recipes(func(rid string, outputId int, ing []int) {
			if outputId == itemId || containsInt(ing, itemId) {
				out = append(out, itemRef{Kind: "recipe", Id: "recipe " + rid})
			}
		})
	}
	if it.quests != nil {
		it.quests(func(token string, rewardIds []int) {
			if containsInt(rewardIds, itemId) {
				out = append(out, itemRef{Kind: "quest", Id: "quest " + token})
			}
		})
	}
	if it.containers != nil {
		it.containers(func(roomId int, itemIds []int) {
			if containsInt(itemIds, itemId) {
				out = append(out, itemRef{Kind: "container", Id: fmt.Sprintf("room %d chest", roomId)})
			}
		})
	}
	return out
}

func containsInt(s []int, v int) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// scanItemReferences wires the real world iterators. Shop stock is dynamic
// living state (self-heals on restock), so it is NOT scanned.
func scanItemReferences(itemId int) []itemRef {
	return scanItemReferencesWith(itemId, refIterators{
		mobs: func(yield func(mobRef)) {
			for _, m := range mobs.AllMobTemplates() {
				ids := append(append(append([]int{}, m.LootPool...), m.PrizeItemIds...), m.AcceptedItemIds...)
				ids = append(ids, m.CrafterRestockMaterials...)
				yield(mobRef{id: int(m.MobId), name: m.Character.Name, ids: ids})
			}
		},
		recipes: func(yield func(string, int, []int)) {
			for id, r := range crafting.GetAllRecipes() {
				ing := []int{}
				for _, in := range r.Ingredients {
					if in.ItemId > 0 {
						ing = append(ing, in.ItemId)
					}
				}
				yield(id, r.Output.ItemId, ing)
			}
		},
		quests: func(yield func(string, []int)) {
			for _, q := range quests.GetAllQuests() {
				ids := []int{}
				if q.Rewards.ItemId > 0 {
					ids = append(ids, q.Rewards.ItemId)
				}
				ids = append(ids, parseItemInfoIds(q.Rewards.ItemInfo)...)
				yield(q.QuestId, ids)
			}
		},
		containers: func(yield func(int, []int)) {
			for _, roomId := range rooms.GetAllRoomIds() {
				r := rooms.LoadRoom(roomId)
				if r == nil || len(r.Containers) == 0 {
					continue
				}
				ids := []int{}
				for _, c := range r.Containers {
					for _, it := range c.Items {
						ids = append(ids, it.ItemId)
					}
				}
				if len(ids) > 0 {
					yield(roomId, ids)
				}
			}
		},
	})
}
```

Add imports (`fmt`, `mobs`, `crafting`, `quests`, `rooms`) and a `parseItemInfoIds(s string) []int` helper that parses the `"itemid[:qty],itemid[:qty]"` format (split on `,`, take the pre-`:` int). **Verify** the exact field names on `Mob` (`MobId`, `Character.Name`), `RecipeSpec` (`QuestId`? use `Quest.QuestId`), and `Quest` via codegraph before writing; adjust if they differ.

- [ ] **Step 4: Run — PASS.** `go test ./modules/gmcp/ -run 'TestBuildItemDelete|TestScanItemReferences'`

- [ ] **Step 5: Route delete** in `handleBuildOp`:
```go
	case `Build.Item.Delete`:
		var req itemDeleteReq
		if json.Unmarshal(evt.Payload, &req) != nil { sendBuildResult(uid, buildErr("bad Build.Item.Delete payload")); break }
		res := buildItemDelete(realItemDeps(), req.ItemId)
		sendBuildResult(uid, res)
		if res.Ok { sendItemList(uid) }
```

- [ ] **Step 6: Build + tests + gofmt.** `go build ./... && go test ./modules/gmcp/ && gofmt -l modules/gmcp/gmcp.Item.go`

- [ ] **Step 7: Commit.**
```bash
git add modules/gmcp/
git commit -m "feat(gmcp): Build.Item.Delete with world-wide reference scan

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 4: Items-mode UI — tab toggle + item list

**Files:** Create `_datafiles/html/public/static/js/items.js`; Modify `build.html`.

- [ ] **Step 1: Add the mode toggle + Items panel shell to `build.html`.** In the toolbar (near the brand), add:
```html
<span class="mode-toggle">
  <button id="tb-mode-rooms" class="active">Rooms</button>
  <button id="tb-mode-items">Items</button>
</span>
```
Add two containers in `#main`: keep `#canvas` for Rooms; add `<div id="itemlist" style="display:none;flex:0 0 320px;overflow-y:auto;background:var(--leather2);border-right:2px solid var(--tooled);"></div>`. Load `items.js` after `builder.js`:
```html
<script src="{{ .CONFIG.FilePaths.WebCDNLocation }}/static/js/items.js"></script>
```

- [ ] **Step 2: Mode switching.** In the inline script, wire the two buttons: Rooms mode shows `#canvas` + the zone/plane toolbar controls and hides `#itemlist`; Items mode hides them, shows `#itemlist`, sends `Build.Item.List`, and `clearInspector()`. Track `window.Builder.mode`.

- [ ] **Step 3: `items.js` list rendering.** `ItemsPanel` object with `render(rows)` that draws a search `<input>` + a type `<select>` (populated from the rows' distinct types) + a scrollable list of rows (`#<id> · <name> · <type>/<subtype> · T<rarity>`), filtered live by the search text + type. Row click → `sendGMCP("Build.Item.Get",{itemId})`. A `+ New Item` button → a type picker → `sendGMCP("Build.Item.Create",{type})`.

- [ ] **Step 4: Route `Build.Items` GMCP** in `build.html` `onGMCP`: `if (ns==="Build.Items") window.Builder.ItemsPanel.render(obj);`. (`Build.Item` handled in Task 5.)

- [ ] **Step 5: Verify in browser.** On `/build`, click **Items** → the list loads and filters by search + type; `+ New Item` creates one and it appears in the list. Server log shows `Build.Item.Create`.

- [ ] **Step 6: Commit.**
```bash
git add _datafiles/html/public/static/js/items.js _datafiles/html/public/build.html
git commit -m "feat(build): Items tab + searchable item list

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 5: Type-aware item form + Save / Create-load / Delete

**Files:** Modify `items.js`, `build.html`.

- [ ] **Step 1: Render the form on `Build.Item`.** In `build.html` `onGMCP`, add `if (ns==="Build.Item") window.Builder.ItemsPanel.renderForm(obj);`. In `items.js`, `renderForm(detail)` builds (into `#inspector`, reusing the room inspector's `ce()` helper pattern) the **Common** section (name, display-name, simple-name, description, type dropdown [detail.types], subtype dropdown [detail.subtypes], value, weight, uses, quantity, RarityTier dropdown [50/40/30/20/10/0], vendor categories, flag checkboxes, quest-token) + a **StatMods** repeatable rows editor (stat select [detail.stats] → int), + the type-specific section chosen from `detail.type`:
  - weapon → DamageMultiplier, hands, parry, speed, stamina-cost, wait-rounds, min-strength, reach, grapple-mod, element; if subtype ∈ {wand,sceptre,staff} → SpellDamageMultiplier; if subtype=shooting → ammoTag.
  - armor slots → physical/magical/conviction mitigation, block-rating, escape-mod.
  - potion/food/drink → buffIds, toxicity, ferment/peak/decay/spoil rounds, bottle mult, bandolier flags.
  - component/componentbag → isComponent, componentTag, weightReduction, bagCapacity, salvageReturns rows.
  - key → keyLockId. ammo → ammoTag.
  - **Advanced** collapsible (leave the uncommon fields — procs, mutation, voice, hunger — as a "raw JSON" textarea in v1 so they round-trip untouched; note this in the section header).

- [ ] **Step 2: Save.** A **Save** button gathers all fields into an `itemUpdateReq` shape and sends one `Build.Item.Update {itemId, …}`. On `Build.Result{ok}` clear dirty + toast; on error show inline. Reuse the room inspector's dirty-tracking + toast pattern.

- [ ] **Step 3: Create → auto-load.** When a `Build.Item.Create` returns `Build.Result{ok,itemId}`, the list refresh (Build.Items) arrives; auto-select the new item by sending `Build.Item.Get{itemId}` so the form opens ready to edit (placeholder name "new item").

- [ ] **Step 4: Delete + reference block.** A **Delete item** button (confirm) → `Build.Item.Delete{itemId}`. On `Build.Result{ok}` clear the form + toast. On a blocked result (`obj.refs` present), render the reference list inline ("Still used by: mob 9538 (a wolf), quest 10 — remove those first") and do NOT clear.

- [ ] **Step 5: Verify in browser.** Create a weapon → set name/DamageMultiplier/RarityTier → Save (persists — check the YAML in `items/weapons-10000/` and that renaming relocates the file). Create an armor + a potion + a component, edit type-specific fields + Save. Attempt to delete a referenced item (blocked, refs shown) and an unreferenced one (deletes). All reflect in the list.

- [ ] **Step 6: Commit.**
```bash
git add _datafiles/html/public/static/js/items.js _datafiles/html/public/build.html
git commit -m "feat(build): type-aware item form + save/create/delete

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 6: End-to-end + content/UX gate (REQUIRED)

**Files:** none (verification).

- [ ] **Step 1: Full build + tests.** `go build ./... && go test ./internal/items/ ./modules/gmcp/`
- [ ] **Step 2: gofmt + vet.** `gofmt -l internal/items/ modules/gmcp/gmcp.Item.go` (empty) and `go vet ./internal/items/ ./modules/gmcp/`.
- [ ] **Step 3: Boot-clean.** Wipe instances; boot with `MapConsistencyEnforce=panic`; confirm items load (`itemspec.LoadDataFiles() itemLoadedCount=…`) and `ValidateZoneConsistency errors=0`, no panic. **Local dev runs on port 8090** (config-overrides `HttpPort`/`config.yaml` skip-worktree — do NOT re-add port 80).
- [ ] **Step 4: REQUIRED browser playtest** (CLAUDE.md gate). On `/build` → **Items**: create one of each class (weapon / an armor slot / a potion / a component), edit fields incl. **RarityTier** and **DamageMultiplier** + Save; confirm each **persists to YAML AND is usable in-game** (in a play client: spawn/give the item, equip the weapon and confirm the multiplier applies, drink the potion). Rename an item and confirm the file relocates (old `<id>-oldname.yaml` gone). Attempt a **blocked delete** (an item in a mob LootPool / recipe / quest) and a **clean delete**. Read every interaction as a confused human; fix what it surfaces, re-run.
- [ ] **Step 5: Report.** Summarize findings + fixes. Do not claim done on boot-clean alone.

---

## Self-Review notes
- **Spec coverage:** persistence (SaveItemSpec/DeleteItemSpec)→T1; Build.Item protocol list/get/create/update→T2; reference-scan delete→T3; Items tab + list→T4; type-aware form + save/create/delete→T5; e2e + gate→T6. RarityTier + DamageMultiplier(not dice) carried in the payload/detail/form. Instancing + LootPool explicitly deferred.
- **Type consistency:** `itemUpdateReq` is defined once (T2) and reused as the Save payload and embedded in `itemDetail`; `itemDeps`/`itemRef`/`refIterators` defined in T2/T3 and used by the routing in T2/T3; `BuildResult.Refs` added in T3 and consumed by the T5 delete-block UI.
- **Risk notes:** (1) item filenames embed the name + armor subfolders — `SaveItemSpec` relocation is the key correctness point (T1 test). (2) reference scan must live in gmcp (import-cycle) — injected iterators keep it testable (T3). (3) `reqToSpec` starts from the loaded spec so form-omitted fields (procs/reserves/worn-buffs) survive a Save; the Advanced raw-JSON textarea (T5) covers the rest. (4) verify `Mob`/`RecipeSpec`/`Quest` field names via codegraph before T3.
```
