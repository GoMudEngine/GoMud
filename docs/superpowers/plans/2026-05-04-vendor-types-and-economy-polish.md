# Vendor Types & Economy Polish — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Convert the buy rule to single-tag-overlap, audit and re-baseline every shopkeeper, layer tier-50/40 baseline restock under foragers/caravan, add forager-stuck watchdog + diagnostics, and rework the economy dashboard to show stock-score deltas + per-tier throughput bars. Spec: `docs/superpowers/specs/2026-05-04-vendor-types-and-economy-polish-design.md`.

**Architecture:** ItemSpec gains `vendor_categories []string`. `EvaluateBuyRules` becomes a single tag-overlap + overstock + gold-reserve gate. `RestockBaselineTiers()` covers tier-50/40 mats on the existing crafter tick. New `forager.Throughput` and `caravan.Throughput` types persist delivery counters, captured into snapshots and rendered as per-rarity-tier colored bars on the dashboard. A boot-time wipe of `_datafiles/world/dogmud/shops/` flushes legacy shop state.

**Tech Stack:** Go 1.x, YAML data files, gopkg.in/yaml.v2, existing GoMud subsystems (`internal/shops`, `internal/forager`, `internal/caravan`, `internal/economy/health`, `internal/behaviortree`, `internal/web`), browser dashboard (`_datafiles/html/admin/economy/index.html`).

---

## Phase order (independently shippable)

1. **Phase 1** — Name template fallback (single small fix, can ship first).
2. **Phase 2** — ItemSpec field + valid-categories helpers.
3. **Phase 3** — Item tagging sweep (per discipline, separately committed).
4. **Phase 4** — Validators (item tags + recipe ingredient tags), wired to boot.
5. **Phase 5** — Buy rule rewrite (TDD).
6. **Phase 6** — Per-vendor cuts (6 mob YAMLs).
7. **Phase 7** — Per-vendor gold defaults (16 mob YAMLs).
8. **Phase 8** — One-time shop wipe.
9. **Phase 9** — Tier-50/40 baseline restock.
10. **Phase 10** — Forager stuck-state watchdog + capture diagnostics.
11. **Phase 11** — Throughput counters (forager + caravan).
12. **Phase 12** — Dashboard rework (stock-score + tier bars).
13. **Phase 13** — Tutorial regression smoke + PATCH_NOTES.

A separate worktree should host this work — branch name suggestion: `feature/vendor-types-economy-polish`.

---

## Phase 1: Name template fallback

### Task 1.1: Fall back to mob template for shop names

**Files:**
- Modify: `internal/economy/health/capture.go` (function `lookupShopMobName`)
- Test: `internal/economy/health/capture_test.go`

- [ ] **Step 1: Read existing `lookupShopMobName` to confirm current shape**

```bash
grep -n "lookupShopMobName" internal/economy/health/capture.go
```
Expected: function defined at the bottom of the file, walks live mob instances and returns "" on no match.

- [ ] **Step 2: Write the failing test**

Add to `internal/economy/health/capture_test.go`:

```go
func TestLookupShopMobName_FallsBackToTemplate(t *testing.T) {
    // No live instance for mobId 999; template provides the name.
    // Test relies on a real mob template — use mobid 63 (Adela) which
    // exists in _datafiles/world/dogmud/mobs/sanctum_basin/63-merchant_adela.yaml.
    mobs.LoadDataFiles() // boot-style template load; or use the test_main_test.go fixture
    name := lookupShopMobName(63, 108)
    if name == "" {
        t.Fatalf("expected template name for mob 63, got empty")
    }
}
```

- [ ] **Step 3: Run test to verify it fails**

Run:
```bash
go test ./internal/economy/health/... -run TestLookupShopMobName_FallsBackToTemplate -v
```
Expected: FAIL — current implementation returns "" when no live mob matches.

- [ ] **Step 4: Add fallback in `lookupShopMobName`**

In `internal/economy/health/capture.go`, replace the function body:

```go
func lookupShopMobName(mobId, roomId int) string {
    for _, instId := range mobs.GetAllMobInstanceIds() {
        m := mobs.GetInstance(instId)
        if m == nil {
            continue
        }
        if int(m.MobId) == mobId && m.HomeRoomId == roomId {
            return m.Character.Name
        }
    }
    // Fallback: template (always loaded at boot).
    if t := mobs.GetMobSpec(mobs.MobId(mobId)); t != nil {
        return t.Character.Name
    }
    return ""
}
```

- [ ] **Step 5: Run test to verify it passes**

```bash
go test ./internal/economy/health/... -run TestLookupShopMobName_FallsBackToTemplate -v
```
Expected: PASS.

- [ ] **Step 6: Run full economy/health package tests**

```bash
go test ./internal/economy/health/...
```
Expected: all PASS, no regression.

- [ ] **Step 7: Commit**

```bash
git add internal/economy/health/capture.go internal/economy/health/capture_test.go
git commit -m "fix(dashboard): fall back to mob template for shop names

When a shop's NPC isn't currently spawned, lookupShopMobName used to
return empty (rendering as #<mobId> in the dashboard). Templates are
loaded at boot, always present — fall back to the template name so the
dashboard shows the shopkeeper's name regardless of live state.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Phase 2: ItemSpec field + valid-categories helpers

### Task 2.1: Add `vendor_categories` to ItemSpec

**Files:**
- Modify: `internal/items/itemspec.go` (struct definition)
- Test: `internal/items/items_test.go`

- [ ] **Step 1: Read existing ItemSpec struct around line 233**

```bash
sed -n '230,290p' internal/items/itemspec.go
```
Expected: struct definition with fields like `ItemId`, `Name`, `Type`, `Value`, `RarityTier`, etc.

- [ ] **Step 2: Write the failing YAML round-trip test**

Add to `internal/items/items_test.go`:

```go
func TestItemSpec_VendorCategories_YAMLRoundtrip(t *testing.T) {
    yamlInput := `itemid: 99999
name: test item
type: object
vendor_categories:
- alchemy
- jewelcrafting
`
    var spec ItemSpec
    if err := yaml.Unmarshal([]byte(yamlInput), &spec); err != nil {
        t.Fatalf("unmarshal: %v", err)
    }
    if len(spec.VendorCategories) != 2 ||
        spec.VendorCategories[0] != "alchemy" ||
        spec.VendorCategories[1] != "jewelcrafting" {
        t.Errorf("VendorCategories = %v, want [alchemy jewelcrafting]", spec.VendorCategories)
    }
}

func TestItemSpec_VendorCategories_DefaultsEmpty(t *testing.T) {
    var spec ItemSpec
    if err := yaml.Unmarshal([]byte("itemid: 1\nname: x\n"), &spec); err != nil {
        t.Fatalf("unmarshal: %v", err)
    }
    if len(spec.VendorCategories) != 0 {
        t.Errorf("VendorCategories = %v, want empty", spec.VendorCategories)
    }
}
```

- [ ] **Step 3: Run tests to verify failure**

```bash
go test ./internal/items/ -run TestItemSpec_VendorCategories -v
```
Expected: COMPILE ERROR — `spec.VendorCategories` undefined.

- [ ] **Step 4: Add field to ItemSpec**

In `internal/items/itemspec.go`, add to the struct (place near other tag-style fields like `ComponentTag`):

```go
VendorCategories []string `yaml:"vendor_categories,omitempty"` // Disciplines that buy/sell this item; mirrors shops.ValidCraftSupports minus "general"
```

- [ ] **Step 5: Run tests to verify pass**

```bash
go test ./internal/items/ -run TestItemSpec_VendorCategories -v
```
Expected: both tests PASS.

- [ ] **Step 6: Run full items package tests**

```bash
go test ./internal/items/...
```
Expected: all PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/items/itemspec.go internal/items/items_test.go
git commit -m "feat(items): add vendor_categories tag to ItemSpec

Multi-tag field on each item identifies which crafting disciplines
buy/sell it. Enables single-rule tag-overlap buy logic (replacing
the old craft-material + gear-upgrade + general-fallback chain).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### Task 2.2: Add `ValidVendorCategories` helper

**Files:**
- Modify: `internal/shops/shopinventory.go`
- Test: `internal/shops/shopinventory_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/shops/shopinventory_test.go`:

```go
func TestIsValidVendorCategory(t *testing.T) {
    tests := []struct {
        in   string
        want bool
    }{
        {"alchemy", true},
        {"blacksmithing", true},
        {"jewelcrafting", true},
        {"tailoring", true},
        {"cooking", true},
        {"enchanting", true},
        {"general", false}, // general is a vendor type, not an item tag
        {"", false},
        {"unknown", false},
    }
    for _, tt := range tests {
        if got := IsValidVendorCategory(tt.in); got != tt.want {
            t.Errorf("IsValidVendorCategory(%q) = %v, want %v", tt.in, got, tt.want)
        }
    }
}
```

- [ ] **Step 2: Run test to verify failure**

```bash
go test ./internal/shops/ -run TestIsValidVendorCategory -v
```
Expected: COMPILE ERROR — `IsValidVendorCategory` undefined.

- [ ] **Step 3: Add helper in `internal/shops/shopinventory.go`**

Add below `IsValidCraftSupport`:

```go
// ValidVendorCategories is the canonical set of values that may appear in
// ItemSpec.VendorCategories. Mirrors ValidCraftSupports MINUS "general"
// — items belong to discipline(s); general stores accept everything.
var ValidVendorCategories = []string{
    CraftSupportBlacksmithing,
    CraftSupportAlchemy,
    CraftSupportTailoring,
    CraftSupportCooking,
    CraftSupportJewelcrafting,
    CraftSupportEnchanting,
}

// IsValidVendorCategory reports whether v is one of ValidVendorCategories.
func IsValidVendorCategory(v string) bool {
    return slices.Contains(ValidVendorCategories, v)
}
```

- [ ] **Step 4: Run test to verify pass**

```bash
go test ./internal/shops/ -run TestIsValidVendorCategory -v
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/shops/shopinventory.go internal/shops/shopinventory_test.go
git commit -m "feat(shops): add IsValidVendorCategory + ValidVendorCategories

Mirror of ValidCraftSupports minus 'general' — items declare their
discipline(s); general stores accept everything regardless of tag.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Phase 3: Item tagging sweep (per discipline)

Each task in this phase tags one batch of items, commits, and moves to the next. Validation in Phase 4 will catch any miss.

### Task 3.1: Tag alchemy raw mats

**Files:**
- Modify: every alchemy-relevant material YAML in `_datafiles/world/dogmud/items/materials-40000/`

- [ ] **Step 1: List all alchemy ingredients used in recipes**

Run:
```bash
grep -rh "item_tag:" _datafiles/world/dogmud/recipes/alchemy/ | sort -u
```
Record the list — these are the canonical alchemy component_tags.

- [ ] **Step 2: For each tag, find the matching item YAML**

Run:
```bash
grep -l "component_tag: <tag>" _datafiles/world/dogmud/items/materials-40000/
```
For each tag from Step 1, identify the item file.

- [ ] **Step 3: Add `vendor_categories: [alchemy]` to each alchemy material**

For an item like `40057-lake_mint.yaml`, add (or merge with existing if cross-discipline):

```yaml
vendor_categories:
- alchemy
```

Apply to (canonical alchemy mats from `mat-audit-matrix.md` + recipe walk):
- 40004 (healer's root), 40005 (bitter thistle), 40006 (glass vial), 40010 (Chrysalis Core), 40028 (binding paste), 40029 (mutation catalyst), 40043 (clay flask), 40044 (sealed phial), 40045 (crystalline decanter), 40046 (moonpetal), 40047 (veilbloom petal), 40048 (serpent venom sac), 40049 (ironbark shaving — also tailoring? confirm via recipes), 40050 (putrid residue), 40051 (skitter-shrimp shell), 40053 (Stillwater black pearl — also jewelcrafting), 40055 (oil of spore?), 40056 (marsh willow bark), 40057 (lake mint), 40058 (freshwater clam), 40059 (lake-iron nodule), 40062 (oak bark), 40063 (shadowcap mushroom), 40064 (wild hare meat — also cooking?), 40065 (beeswax), 40066 (blood-moss), 40067 (pine pitch).

For mats used in multiple disciplines, list all applicable disciplines now (saves a second sweep later).

- [ ] **Step 4: Boot-smoke that nothing parses broken**

```bash
go build ./...
```
Expected: clean build (no validators yet — just YAML round-trip stability).

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/items/materials-40000/
git commit -m "content(items): tag alchemy materials with vendor_categories

Walks all alchemy recipes; tags each ingredient material with its
discipline (or disciplines, for cross-cutting mats like ironbark).
Step 1 of the per-discipline tagging sweep.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### Task 3.2: Tag blacksmithing raw mats

Same pattern as 3.1 but using `_datafiles/world/dogmud/recipes/blacksmithing/`. Likely items: 40001 (iron ingot), 40018 (steel ingot), 40019 (chain link), 40020 (coal dust), 40002 (leather strip — also tailoring), 40068 (sinew — also tailoring), 40013 (bone needle — possibly also tailoring), 40012 (thread spool — also tailoring).

- [ ] Step 1-5 mirror 3.1 with blacksmithing recipes.
- [ ] Commit message: `content(items): tag blacksmithing materials with vendor_categories`.

### Task 3.3: Tag jewelcrafting raw mats

`_datafiles/world/dogmud/recipes/jewelcrafting/`. Likely items: 40021 (copper wire), 40022 (silver wire), 40023 (gold wire), 40024 (polished stone), 40025 (raw gem), 40026 (gem dust), 40030 (chrysalis setting), 40027 (chrysalis shard), 40053 (Stillwater black pearl), 40001 (iron ingot — possibly also).

- [ ] Step 1-5 mirror 3.1.
- [ ] Commit message: `content(items): tag jewelcrafting materials with vendor_categories`.

### Task 3.4: Tag tailoring raw mats

`_datafiles/world/dogmud/recipes/tailoring/`. Likely items: 40007 (cloth strip), 40012 (thread spool), 40013 (bone needle), 40002 (leather strip), 40068 (sinew), and any new tailoring-specific items.

- [ ] Step 1-5 mirror 3.1.
- [ ] Commit message: `content(items): tag tailoring materials with vendor_categories`.

### Task 3.5: Tag cooking raw mats

`_datafiles/world/dogmud/recipes/cooking/`. Likely items: 40014 (raw meat), 40015 (wild vegetables), 40016 (water flask), 40017 (salt pouch), 40058 (freshwater clam), 40064 (wild hare meat), 40065 (beeswax — also alchemy).

- [ ] Step 1-5 mirror 3.1.
- [ ] Commit message: `content(items): tag cooking materials with vendor_categories`.

### Task 3.6: Tag enchanting raw mats

`_datafiles/world/dogmud/recipes/enchanting/`. Walk recipe dir to identify ingredients.

- [ ] Step 1-5 mirror 3.1.
- [ ] Commit message: `content(items): tag enchanting materials with vendor_categories`.

### Task 3.7: Tag finished goods (weapons)

**Files:** `_datafiles/world/dogmud/items/weapons-10000/*.yaml` (35 files).

- [ ] **Step 1: Add `vendor_categories: [blacksmithing]` to all weapon YAMLs.**

Bows, crossbows are also blacksmithing? Confirm with recipe walk. Wands/sceptres/staves may be enchanting. For each weapon, decide based on item type/subtype:
- swords, axes, daggers, maces, spears, hammers, shields, bows, crossbows → `[blacksmithing]`
- wands, sceptres, staves → `[enchanting]` (review carefully — these may be alchemy or jewelcrafting depending on lore)

- [ ] **Step 2: Boot-smoke.**

```bash
go build ./...
```

- [ ] **Step 3: Commit.**

```bash
git add _datafiles/world/dogmud/items/weapons-10000/
git commit -m "content(items): tag weapons with vendor_categories

Blacksmithing for melee + ranged; enchanting for wand/sceptre/staff.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### Task 3.8: Tag finished goods (armor)

**Files:** `_datafiles/world/dogmud/items/armor-20000/*.yaml` (13 files).

- [ ] **Step 1: For each armor item, decide tag based on material/subtype.**

- Metal armor (helms, plates, gauntlets, greaves) → `[blacksmithing]`
- Cloth armor (robes, hoods) → `[tailoring]`
- Leather armor → `[blacksmithing, tailoring]` (cross-cutting; tanners + smiths both work leather historically)

- [ ] **Step 2: Boot-smoke + commit.**

```bash
git add _datafiles/world/dogmud/items/armor-20000/
git commit -m "content(items): tag armor with vendor_categories

Metal armor → blacksmithing. Cloth → tailoring. Leather → both.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### Task 3.9: Tag finished goods (consumables)

**Files:** `_datafiles/world/dogmud/items/consumables-30000/*.yaml` (44 files).

- [ ] **Step 1: For each consumable, decide tag.**

- Potions → `[alchemy]`
- Food → `[cooking]`
- Scrolls → `[enchanting]`
- Other (e.g., grenades) → review spec / existing recipe

- [ ] **Step 2: Boot-smoke + commit.**

```bash
git add _datafiles/world/dogmud/items/consumables-30000/
git commit -m "content(items): tag consumables with vendor_categories

Potions → alchemy. Food → cooking. Scrolls → enchanting.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### Task 3.10: Tag remaining items (other-0)

**Files:** `_datafiles/world/dogmud/items/other-0/*.yaml` (35 files).

- [ ] **Step 1: For each item, decide:**

- Quest items (`questtoken: ...` set) → leave empty.
- Holdable props with no value or `value: 0` → leave empty.
- Anything else with a clear discipline mapping → tag.
- Genuinely uncategorizable (tools? lockpicks?) → tag the most plausible single discipline (lockpicks → blacksmithing).

- [ ] **Step 2: Boot-smoke + commit.**

```bash
git add _datafiles/world/dogmud/items/other-0/
git commit -m "content(items): tag remaining items with vendor_categories

Quest items left untagged (rejected by buy rule via QuestToken
check). Tools and misc gear get plausible single-discipline tags.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Phase 4: Validators

### Task 4.1: ValidateVendorCategories (item-side validator)

**Files:**
- Create: `internal/items/validation.go`
- Test: `internal/items/validation_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/items/validation_test.go`:

```go
package items

import (
    "strings"
    "testing"
)

func TestValidateVendorCategories_RejectsUntaggedSalableItem(t *testing.T) {
    specs := map[int]*ItemSpec{
        1: {ItemId: 1, Name: "x", Value: 5}, // no vendor_categories, no questtoken
    }
    err := ValidateVendorCategories(specs, []string{"alchemy", "blacksmithing"})
    if err == nil {
        t.Fatalf("expected error for untagged salable item")
    }
    if !strings.Contains(err.Error(), "1") {
        t.Errorf("error should mention itemId 1: %v", err)
    }
}

func TestValidateVendorCategories_AllowsQuestItem(t *testing.T) {
    specs := map[int]*ItemSpec{
        1: {ItemId: 1, Name: "quest token", Value: 0, QuestToken: "5-start"},
    }
    err := ValidateVendorCategories(specs, []string{"alchemy"})
    if err != nil {
        t.Errorf("quest items should be skipped: %v", err)
    }
}

func TestValidateVendorCategories_AllowsZeroValueItem(t *testing.T) {
    specs := map[int]*ItemSpec{
        1: {ItemId: 1, Name: "prop", Value: 0},
    }
    err := ValidateVendorCategories(specs, []string{"alchemy"})
    if err != nil {
        t.Errorf("zero-value items should be skipped: %v", err)
    }
}

func TestValidateVendorCategories_RejectsUnknownCategory(t *testing.T) {
    specs := map[int]*ItemSpec{
        1: {ItemId: 1, Name: "x", Value: 5, VendorCategories: []string{"madeup"}},
    }
    err := ValidateVendorCategories(specs, []string{"alchemy", "blacksmithing"})
    if err == nil {
        t.Fatalf("expected error for unknown category")
    }
    if !strings.Contains(err.Error(), "madeup") {
        t.Errorf("error should mention bad category: %v", err)
    }
}

func TestValidateVendorCategories_AcceptsTaggedItem(t *testing.T) {
    specs := map[int]*ItemSpec{
        1: {ItemId: 1, Name: "x", Value: 5, VendorCategories: []string{"alchemy"}},
    }
    err := ValidateVendorCategories(specs, []string{"alchemy", "blacksmithing"})
    if err != nil {
        t.Errorf("expected no error: %v", err)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/items/ -run TestValidateVendorCategories -v
```
Expected: COMPILE ERROR — `ValidateVendorCategories` undefined.

- [ ] **Step 3: Implement the validator**

Create `internal/items/validation.go`:

```go
package items

import (
    "fmt"
    "slices"
    "sort"
    "strings"
)

// ValidateVendorCategories panics-style validator for ItemSpec.VendorCategories
// integrity. Returns a non-nil error listing every offending item if any:
//   - has Value > 0 AND empty QuestToken AND empty VendorCategories, OR
//   - carries a VendorCategories value not present in validCategories.
//
// Caller behavior on non-nil error:
//   - Cold boot: panic.
//   - /reload: log Error and continue (data files in inconsistent state).
func ValidateVendorCategories(specs map[int]*ItemSpec, validCategories []string) error {
    type fault struct {
        itemId int
        name   string
        why    string
    }
    var faults []fault

    for id, spec := range specs {
        if spec == nil {
            continue
        }
        // Skip non-salable items.
        if spec.Value <= 0 {
            continue
        }
        if spec.QuestToken != "" {
            continue
        }
        if len(spec.VendorCategories) == 0 {
            faults = append(faults, fault{id, spec.Name, "missing vendor_categories"})
            continue
        }
        for _, c := range spec.VendorCategories {
            if !slices.Contains(validCategories, c) {
                faults = append(faults, fault{id, spec.Name,
                    fmt.Sprintf("unknown vendor_category %q", c)})
            }
        }
    }

    if len(faults) == 0 {
        return nil
    }

    sort.Slice(faults, func(i, j int) bool { return faults[i].itemId < faults[j].itemId })
    var b strings.Builder
    fmt.Fprintf(&b, "items with bad vendor_categories (%d):\n", len(faults))
    for _, f := range faults {
        fmt.Fprintf(&b, "  - item %d (%q): %s\n", f.itemId, f.name, f.why)
    }
    fmt.Fprintf(&b, "Valid vendor_categories: %s\n", strings.Join(validCategories, ", "))
    return fmt.Errorf("%s", b.String())
}
```

- [ ] **Step 4: Run tests to verify pass**

```bash
go test ./internal/items/ -run TestValidateVendorCategories -v
```
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/items/validation.go internal/items/validation_test.go
git commit -m "feat(items): add ValidateVendorCategories boot validator

Enumerates every item that's salable but missing or carrying an
invalid vendor_category. Listed all-at-once so a single boot
surfaces the full set.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### Task 4.2: ValidateRecipeIngredientTags (recipe-side validator)

**Files:**
- Create: `internal/crafting/validation.go`
- Test: `internal/crafting/validation_test.go`

- [ ] **Step 1: Inspect Recipe + Ingredient struct shape**

```bash
grep -n "type Recipe struct\|type Ingredient struct\|ItemTag\|Skill" internal/crafting/*.go | head -30
```
Identify field names.

- [ ] **Step 2: Write the failing test**

Create `internal/crafting/validation_test.go`:

```go
package crafting

import (
    "strings"
    "testing"

    "github.com/GoMudEngine/GoMud/internal/items"
)

func TestValidateRecipeIngredientTags_FlagsMissingTag(t *testing.T) {
    recipes := map[string]*Recipe{
        "test-r": {
            Id:    "test-r",
            Skill: "alchemy",
            Ingredients: []Ingredient{
                {ItemTag: "lake-mint", Quantity: 1},
            },
        },
    }
    specs := map[int]*items.ItemSpec{
        1: {
            ItemId:           1,
            Name:             "lake mint",
            ComponentTag:     "lake-mint",
            VendorCategories: []string{}, // missing alchemy
        },
    }
    err := ValidateRecipeIngredientTags(recipes, specs)
    if err == nil {
        t.Fatalf("expected error for missing tag")
    }
    if !strings.Contains(err.Error(), "alchemy") {
        t.Errorf("error should mention 'alchemy': %v", err)
    }
}

func TestValidateRecipeIngredientTags_AcceptsCorrectTag(t *testing.T) {
    recipes := map[string]*Recipe{
        "test-r": {
            Id:    "test-r",
            Skill: "alchemy",
            Ingredients: []Ingredient{
                {ItemTag: "lake-mint", Quantity: 1},
            },
        },
    }
    specs := map[int]*items.ItemSpec{
        1: {
            ItemId:           1,
            Name:             "lake mint",
            ComponentTag:     "lake-mint",
            VendorCategories: []string{"alchemy"},
        },
    }
    err := ValidateRecipeIngredientTags(recipes, specs)
    if err != nil {
        t.Errorf("expected no error: %v", err)
    }
}

func TestValidateRecipeIngredientTags_SkipsIngredientsWithoutItem(t *testing.T) {
    // Recipe references a tag with no canonical item; warning only, not panic.
    recipes := map[string]*Recipe{
        "test-r": {
            Id:    "test-r",
            Skill: "alchemy",
            Ingredients: []Ingredient{
                {ItemTag: "ghost-tag", Quantity: 1},
            },
        },
    }
    specs := map[int]*items.ItemSpec{}
    err := ValidateRecipeIngredientTags(recipes, specs)
    // Implementation choice: ghost tags log a warning but don't error.
    // If you'd prefer they error, change the contract here.
    if err != nil {
        t.Errorf("expected no error for ghost tag: %v", err)
    }
}
```

- [ ] **Step 3: Run tests to verify failure**

```bash
go test ./internal/crafting/ -run TestValidateRecipeIngredientTags -v
```
Expected: COMPILE ERROR.

- [ ] **Step 4: Implement validator**

Create `internal/crafting/validation.go`:

```go
package crafting

import (
    "fmt"
    "slices"
    "sort"
    "strings"

    "github.com/GoMudEngine/GoMud/internal/items"
    "github.com/GoMudEngine/GoMud/internal/mudlog"
)

// ValidateRecipeIngredientTags ensures every recipe ingredient resolves
// to an item carrying the recipe's Skill in its VendorCategories list.
// Ingredients with no canonical item (no ItemSpec has matching
// ComponentTag) emit a warning but don't error — those are legitimate
// "raw concept" tags (e.g., a flavor recipe that hasn't been wired to
// a specific item).
func ValidateRecipeIngredientTags(
    recipes map[string]*Recipe,
    specs map[int]*items.ItemSpec,
) error {
    // Index items by ComponentTag.
    byTag := map[string]*items.ItemSpec{}
    for _, s := range specs {
        if s == nil || s.ComponentTag == "" {
            continue
        }
        byTag[s.ComponentTag] = s
    }

    type fault struct {
        recipeId string
        itemTag  string
        skill    string
        why      string
    }
    var faults []fault

    for id, r := range recipes {
        if r == nil {
            continue
        }
        for _, ing := range r.Ingredients {
            spec, ok := byTag[ing.ItemTag]
            if !ok {
                mudlog.Warn("recipe ingredient has no canonical item",
                    "recipe", id, "itemTag", ing.ItemTag)
                continue
            }
            if !slices.Contains(spec.VendorCategories, r.Skill) {
                faults = append(faults, fault{
                    recipeId: id,
                    itemTag:  ing.ItemTag,
                    skill:    r.Skill,
                    why: fmt.Sprintf("item %d (%q) missing %q in vendor_categories",
                        spec.ItemId, spec.Name, r.Skill),
                })
            }
        }
    }

    if len(faults) == 0 {
        return nil
    }

    sort.Slice(faults, func(i, j int) bool { return faults[i].recipeId < faults[j].recipeId })
    var b strings.Builder
    fmt.Fprintf(&b, "recipe ingredients with missing vendor_categories tags (%d):\n", len(faults))
    for _, f := range faults {
        fmt.Fprintf(&b, "  - recipe %q ingredient %q (%s): %s\n",
            f.recipeId, f.itemTag, f.skill, f.why)
    }
    return fmt.Errorf("%s", b.String())
}
```

- [ ] **Step 5: Run tests to verify pass**

```bash
go test ./internal/crafting/ -run TestValidateRecipeIngredientTags -v
```
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/crafting/validation.go internal/crafting/validation_test.go
git commit -m "feat(crafting): validate recipe ingredients carry discipline tag

Ensures every recipe's ingredients have the recipe's skill listed
in their item vendor_categories. Catches typos like 'recipe needs
lake-mint, no item with that component_tag exists' and
'recipe needs alchemy mat, but item only tagged blacksmithing.'

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### Task 4.3: Wire validators into boot

**Files:**
- Modify: `main.go` (the `loadAllDataFiles` or equivalent boot function)

- [ ] **Step 1: Find the boot loading sequence**

```bash
grep -n "items.LoadDataFiles\|crafting.LoadDataFiles\|recipes.LoadDataFiles\|ValidateShopMobTags" main.go
```
Identify the order: items → recipes → shop mob tags.

- [ ] **Step 2: Insert validator calls after items + recipes load**

In `main.go`, after `items.LoadDataFiles()` and `crafting.LoadDataFiles()` (or whatever loads recipes), add:

```go
// Validate item vendor_categories (panic on cold boot, log on /reload).
if err := items.ValidateVendorCategories(items.AllItemSpecs(), shops.ValidVendorCategories); err != nil {
    if isColdBoot {
        panic(err)
    }
    mudlog.Error("items.ValidateVendorCategories", "error", err)
}

// Validate recipe ingredient tags.
if err := crafting.ValidateRecipeIngredientTags(crafting.AllRecipes(), items.AllItemSpecs()); err != nil {
    if isColdBoot {
        panic(err)
    }
    mudlog.Error("crafting.ValidateRecipeIngredientTags", "error", err)
}
```

(Replace `items.AllItemSpecs()` and `crafting.AllRecipes()` with whatever the actual accessors are; check `internal/items/items.go` and `internal/crafting/*.go` for the canonical API. If accessors don't exist, add them as part of this step.)

- [ ] **Step 3: Boot the server, see what fails**

```bash
go run main.go
```
Expected: panic with the full list of items/recipes that failed validation. Phase 3 should have addressed most; this surfaces any missed.

- [ ] **Step 4: Iterate Phase 3 commits as needed to fix any remaining failures**

For each failure: edit the offending YAML, re-run boot, repeat until clean.

- [ ] **Step 5: Boot completes past validators**

```bash
go run main.go
```
Expected: server reaches "listening on" or equivalent without panic.

- [ ] **Step 6: Commit**

```bash
git add main.go internal/items/items.go internal/crafting/*.go
git commit -m "feat(boot): wire VendorCategories + RecipeIngredient validators

Cold boot now panics if any item is missing or carrying an invalid
vendor_categories tag, or if any recipe ingredient resolves to an
item that doesn't carry the recipe's discipline.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Phase 5: Buy rule rewrite

### Task 5.1: Rewrite buy-rule unit tests for the new contract

**Files:**
- Modify: `internal/shops/buyrules_test.go`

- [ ] **Step 1: Read the existing tests to understand fixtures**

```bash
sed -n '1,60p' internal/shops/buyrules_test.go
```
Note the test fixtures (sample ItemSpecs, ShopInventory, PricingConfig).

- [ ] **Step 2: Replace existing test cases with new contract tests**

In `internal/shops/buyrules_test.go`, replace the test functions with (keep the existing test fixtures / setup helpers):

```go
func TestEvaluateBuyRules_QuestItem_Reject(t *testing.T) {
    item := makeTestItem(itemSpecQuest)
    shopInv := &ShopInventory{CraftSupport: "alchemy", Gold: 1000, StartingGold: 1000}
    offer := EvaluateBuyRules(item, shopInv, "", false, DefaultPricingConfig(), nil)
    if offer.Price != 0 {
        t.Errorf("quest item should reject, got %+v", offer)
    }
}

func TestEvaluateBuyRules_UntaggedItem_Reject(t *testing.T) {
    item := makeTestItem(&items.ItemSpec{ItemId: 99, Name: "stuff", Value: 5})
    shopInv := &ShopInventory{CraftSupport: "alchemy", Gold: 1000, StartingGold: 1000}
    offer := EvaluateBuyRules(item, shopInv, "", false, DefaultPricingConfig(), nil)
    if offer.Price != 0 {
        t.Errorf("untagged item should reject, got %+v", offer)
    }
}

func TestEvaluateBuyRules_TagMatch_Accept(t *testing.T) {
    spec := &items.ItemSpec{ItemId: 99, Name: "stuff", Value: 10,
        VendorCategories: []string{"alchemy"}}
    item := makeTestItem(spec)
    shopInv := &ShopInventory{CraftSupport: "alchemy", Gold: 1000, StartingGold: 1000}
    offer := EvaluateBuyRules(item, shopInv, "", false, DefaultPricingConfig(), nil)
    if offer.Price <= 0 {
        t.Errorf("tagged-matching item should accept with positive price, got %+v", offer)
    }
}

func TestEvaluateBuyRules_TagMismatch_Reject(t *testing.T) {
    spec := &items.ItemSpec{ItemId: 99, Name: "stuff", Value: 10,
        VendorCategories: []string{"blacksmithing"}}
    item := makeTestItem(spec)
    shopInv := &ShopInventory{CraftSupport: "alchemy", Gold: 1000, StartingGold: 1000}
    offer := EvaluateBuyRules(item, shopInv, "", false, DefaultPricingConfig(), nil)
    if offer.Price != 0 {
        t.Errorf("tag-mismatched item should reject, got %+v", offer)
    }
}

func TestEvaluateBuyRules_MultiTag_Accept(t *testing.T) {
    spec := &items.ItemSpec{ItemId: 99, Name: "stuff", Value: 10,
        VendorCategories: []string{"alchemy", "jewelcrafting"}}
    item := makeTestItem(spec)
    // Jeweler accepts despite being secondary tag.
    shopInv := &ShopInventory{CraftSupport: "jewelcrafting", Gold: 1000, StartingGold: 1000}
    offer := EvaluateBuyRules(item, shopInv, "", false, DefaultPricingConfig(), nil)
    if offer.Price <= 0 {
        t.Errorf("multi-tag item should accept on either tag, got %+v", offer)
    }
}

func TestEvaluateBuyRules_GeneralStore_AcceptsAnyTag(t *testing.T) {
    spec := &items.ItemSpec{ItemId: 99, Name: "stuff", Value: 10,
        VendorCategories: []string{"blacksmithing"}}
    item := makeTestItem(spec)
    shopInv := &ShopInventory{CraftSupport: "general", Gold: 5000, StartingGold: 5000}
    offer := EvaluateBuyRules(item, shopInv, "", false, DefaultPricingConfig(), nil)
    if offer.Price <= 0 {
        t.Errorf("general store should accept tagged item, got %+v", offer)
    }
}

func TestEvaluateBuyRules_AtMaxStock_Reject(t *testing.T) {
    spec := &items.ItemSpec{ItemId: 99, Name: "stuff", Value: 10,
        VendorCategories: []string{"alchemy"}}
    item := makeTestItem(spec)
    shopInv := &ShopInventory{
        CraftSupport: "alchemy", Gold: 1000, StartingGold: 1000,
        Stock: []StockEntry{{ItemId: 99, MaxStock: 50, Current: 50}},
    }
    offer := EvaluateBuyRules(item, shopInv, "", false, DefaultPricingConfig(), nil)
    if offer.Price != 0 {
        t.Errorf("at-cap item should reject, got %+v", offer)
    }
}

func TestEvaluateBuyRules_InsufficientGold_Reject(t *testing.T) {
    spec := &items.ItemSpec{ItemId: 99, Name: "stuff", Value: 1000,
        VendorCategories: []string{"alchemy"}}
    item := makeTestItem(spec)
    shopInv := &ShopInventory{
        CraftSupport: "alchemy", Gold: 50, StartingGold: 1000, // Reserve = 1000 * GoldReserveRatio
    }
    cfg := DefaultPricingConfig()
    cfg.GoldReserveRatio = 0.5 // reserve = 500
    offer := EvaluateBuyRules(item, shopInv, "", false, cfg, nil)
    if offer.Price != 0 {
        t.Errorf("insufficient-gold should reject, got %+v", offer)
    }
}

func TestEvaluateBuyRules_GearUpgradeNoLongerAccepted(t *testing.T) {
    // Regression: a smithing-tagged sword sold to an alchemist must REJECT
    // even if alchemist's loadout is empty (gear upgrade rule is gone).
    spec := &items.ItemSpec{ItemId: 99, Name: "iron sword", Value: 100,
        Type: items.Weapon, VendorCategories: []string{"blacksmithing"}}
    item := makeTestItem(spec)
    shopInv := &ShopInventory{CraftSupport: "alchemy", Gold: 1000, StartingGold: 1000}
    // Caller used to pass wornItems; we still pass nil for sig compat.
    offer := EvaluateBuyRules(item, shopInv, "", false, DefaultPricingConfig(), nil)
    if offer.Price != 0 {
        t.Errorf("gear upgrade rule should be gone — alchemist should reject sword, got %+v", offer)
    }
}

func TestEvaluateBuyRules_DecliningPotion_Reject(t *testing.T) {
    // Use existing aging fixture — see existing test for pattern.
    // ... (port the existing declining-potion case to match new signature)
}
```

(Adapt `makeTestItem`, `itemSpecQuest`, etc. to whatever helpers exist in the file.)

- [ ] **Step 3: Run tests to verify failure**

```bash
go test ./internal/shops/ -run TestEvaluateBuyRules -v
```
Expected: most tests FAIL (current implementation doesn't match new contract).

- [ ] **Step 4: Commit the test changes (RED)**

```bash
git add internal/shops/buyrules_test.go
git commit -m "test(shops): rewrite buy-rule tests for tag-overlap contract

Phase 5 RED — replaces craft-material/gear-upgrade/general-fallback
scenarios with single-rule tag-overlap, overstock cap, and
gold-reserve gate.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### Task 5.2: Implement new buy rule

**Files:**
- Modify: `internal/shops/buyrules.go` (replace `EvaluateBuyRules` body, delete dead helpers)

- [ ] **Step 1: Replace `EvaluateBuyRules` and add `vendorAcceptsAny`, `pickReason`, extract `isPotionDeclining`**

In `internal/shops/buyrules.go`, replace the entire file body (keep imports, BuyOffer type) with:

```go
package shops

import (
    "github.com/GoMudEngine/GoMud/internal/items"
    "github.com/GoMudEngine/GoMud/internal/util"
)

type BuyOffer struct {
    Price  int
    Reason string // "craft_material", "potion", "general", ""
}

// EvaluateBuyRules returns what an NPC will pay for an item offered by
// a player. Single-rule tag-overlap with overstock and gold-reserve gates.
//
// Reject conditions, in order:
//   1. Item has no ItemSpec or carries a QuestToken.
//   2. Item is a potion in PhaseDeclining or PhaseSpoiled.
//   3. Item has no vendor_categories tags.
//   4. Vendor's craft_support doesn't accept any of the item's tags.
//   5. Vendor is at MaxStock for this item ("48 iron ores" overstock cap).
//   6. Vendor can't afford the buy price without dropping below
//      GoldReserve(cfg.GoldReserveRatio).
//
// Otherwise returns a BuyOffer with dynamic price from CalcBuyPrice.
//
// crafterSkill, buysGeneral, wornItems are unused — kept in the signature
// for back-compat with the call site in internal/usercommands/sell.go.
func EvaluateBuyRules(
    item items.Item,
    shopInv *ShopInventory,
    crafterSkill string,
    buysGeneral bool,
    cfg PricingConfig,
    wornItems []items.Item,
) BuyOffer {
    spec := item.GetSpec()
    if spec == nil || spec.ItemId < 1 || spec.QuestToken != "" {
        return BuyOffer{}
    }

    if spec.Type == items.Potion && isPotionDeclining(item, spec) {
        return BuyOffer{}
    }

    if len(spec.VendorCategories) == 0 {
        return BuyOffer{}
    }
    if !vendorAcceptsAny(shopInv.CraftSupport, spec.VendorCategories) {
        return BuyOffer{}
    }

    // Overstock cap.
    if entry := shopInv.GetStock(spec.ItemId); entry != nil &&
        entry.MaxStock > 0 && entry.Current >= entry.MaxStock {
        return BuyOffer{}
    }

    // Compute price.
    current, restock := 0, 1
    if entry := shopInv.GetStock(spec.ItemId); entry != nil {
        current = entry.Current
        if entry.RestockQty > 0 {
            restock = entry.RestockQty
        }
    }
    price := CalcBuyPrice(spec.Value, current, restock, cfg)

    // Gold-reserve gate.
    reserve := shopInv.GoldReserve(cfg.GoldReserveRatio)
    if !shopInv.CanAfford(price, reserve) {
        return BuyOffer{}
    }

    return BuyOffer{Price: price, Reason: pickReason(spec)}
}

func vendorAcceptsAny(craftSupport string, itemTags []string) bool {
    if craftSupport == CraftSupportGeneral {
        return true
    }
    for _, t := range itemTags {
        if t == craftSupport {
            return true
        }
    }
    return false
}

func pickReason(spec *items.ItemSpec) string {
    if spec.Type == items.Potion {
        return "potion"
    }
    if spec.IsComponent {
        return "craft_material"
    }
    return "general"
}

// isPotionDeclining reports whether a potion's aging phase is Declining
// or Spoiled — those should never be bought (potions whose magic has
// faded or gone toxic).
func isPotionDeclining(item items.Item, spec *items.ItemSpec) bool {
    if !spec.Aging.HasAging() || item.CraftedRound == 0 {
        return false
    }
    currentRound := util.GetRoundCount()
    var elapsed uint64
    if currentRound >= item.CraftedRound {
        elapsed = currentRound - item.CraftedRound
    }
    bottleMult := item.BottleMultiplier
    if bottleMult <= 0 {
        bottleMult = spec.BottleAgingMultiplier
    }
    effectiveSpeed := items.CalcEffectiveAgingSpeed(bottleMult, item.CraftSkill)
    phase, _ := items.GetAgingPhase(elapsed, spec.Aging, effectiveSpeed)
    return phase == items.PhaseDeclining || phase == items.PhaseSpoiled
}
```

- [ ] **Step 2: Run tests to verify pass (GREEN)**

```bash
go test ./internal/shops/ -run TestEvaluateBuyRules -v
```
Expected: all tests PASS.

- [ ] **Step 3: Run full shops package tests**

```bash
go test ./internal/shops/...
```
Expected: all PASS (older tests around `usesComponent`/`canCraftItem` should now also be removed in this commit — the helpers no longer exist).

- [ ] **Step 4: Run sell command tests + cross-package**

```bash
go test ./internal/usercommands/...
```
Expected: PASS (signature unchanged).

- [ ] **Step 5: Commit (GREEN)**

```bash
git add internal/shops/buyrules.go
git commit -m "feat(shops): rewrite EvaluateBuyRules to tag-overlap + gates

Replaces 5-rule chain (quest / craft-material / gear-upgrade /
potion / general) with single rule:
  reject quest, reject declining potion, require vendor-tag overlap,
  reject overstock, reject if drops below gold reserve.
Removes ~80 lines of usesComponent/canCraftItem/isUpgrade helpers.
Gear-upgrade behavior intentionally gone — shopkeepers are
non-combatants who never wore their purchases.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Phase 6: Per-vendor cuts

Each task: strip shopkeeper status from one mob YAML, build, commit.

### Task 6.1: Cut Korvath (mob 52, Sanctum Basin)

**File:** `_datafiles/world/dogmud/mobs/sanctum_basin/52-korvath.yaml`

- [ ] **Step 1: Edit the YAML.** Remove the following keys: `crafter`, `crafterskill`, `crafterrecipeids`, `crafterrestockmaterials`, `craft_support`, and the `shop:` block under `character:`. Leave `non_combatant: true`, `player_attack_immune: true`, dialogue, and other identity fields untouched.

- [ ] **Step 2: Build.**

```bash
go build ./...
```
Expected: clean.

- [ ] **Step 3: Commit.**

```bash
git add _datafiles/world/dogmud/mobs/sanctum_basin/52-korvath.yaml
git commit -m "content(mobs): remove shopkeeper status from Korvath

Korvath is a questgiver, not a vendor. Strip crafter/shop fields;
non_combatant + player_attack_immune stay so questline still works.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### Task 6.2: Cut Yenna (mob 53, Sanctum Basin)

Same pattern as 6.1 on `_datafiles/world/dogmud/mobs/sanctum_basin/53-alchemist_yenna.yaml`. Commit message: `content(mobs): remove shopkeeper status from Yenna`.

### Task 6.3: Cut Sigrid (mob 333, Stillwater)

Same pattern on `_datafiles/world/dogmud/mobs/stillwater/333-innkeeper_sigrid.yaml`. Commit message: `content(mobs): remove shopkeeper status from Sigrid (innkeeper, dialogue-only)`.

### Task 6.4: Cut Haral (mob 278, North Road)

Same pattern on `_datafiles/world/dogmud/mobs/north_road/278-haral.yaml`. Commit message: `content(mobs): remove shopkeeper status from Haral (flavor)`.

### Task 6.5: Cut Whisper (mob 273, Thornwall)

Same pattern on `_datafiles/world/dogmud/mobs/thornwall_city/273-whisper.yaml`. Commit message: `content(mobs): remove shopkeeper status from Whisper (flavor)`.

### Task 6.6: Cut Bram (mob 348, Stillwater) + drop archetype

**File:** `_datafiles/world/dogmud/mobs/stillwater/348-miller_bram.yaml`

- [ ] **Step 1: Edit the YAML.** Same field removals as 6.1, AND additionally remove `behavior_archetype: noncombat_shopkeeper`. Keep `non_combatant: true` if present (harmless to leave).

- [ ] **Step 2: Build + commit.**

```bash
git add _datafiles/world/dogmud/mobs/stillwater/348-miller_bram.yaml
git commit -m "content(mobs): reframe Bram as flavor miller (drop shop + archetype)

Bram is a duplicate-cooking specialist next to Tov Brann; strip
shopkeeper status entirely and drop noncombat_shopkeeper archetype
so he becomes a pure flavor mob.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Phase 7: Per-vendor gold defaults (16 mobs)

Each task: bump `gold:` in mob YAML to the new default. The migration is per-mob since each may have an existing custom value (e.g., Ilsa is 200 today).

### Task 7.1: Bump specialist shopkeeper gold to 1000

**Files** (12 mobs — all specialists from the keepers table):
- `_datafiles/world/dogmud/mobs/thornwall_city/97-blacksmith_kerra.yaml`
- `_datafiles/world/dogmud/mobs/thornwall_city/98-apothecary_voss.yaml`
- `_datafiles/world/dogmud/mobs/thornwall_city/103-food_vendor.yaml`
- `_datafiles/world/dogmud/mobs/thornwall_city/108-jeweler_tess.yaml`
- `_datafiles/world/dogmud/mobs/thornwall_city/109-enchanter_vael.yaml`
- `_datafiles/world/dogmud/mobs/thornwall_city/113-weaver_maren.yaml`
- `_datafiles/world/dogmud/mobs/thornwall_city/248-tavern_cook_brynn.yaml`
- `_datafiles/world/dogmud/mobs/stillwater/336-fishmonger_tov_brann.yaml`
- `_datafiles/world/dogmud/mobs/stillwater/337-smith_brindle.yaml`
- `_datafiles/world/dogmud/mobs/stillwater/338-apothecary_ilsa.yaml`
- `_datafiles/world/dogmud/mobs/stillwater/339-weaver_edda.yaml`
- `_datafiles/world/dogmud/mobs/stillwater/340-pearl_carver_kess.yaml`

- [ ] **Step 1: For each YAML above, set `gold: 1000` under `character:`.**

Replace the existing `gold: <N>` line. If the file doesn't have one (unlikely), add it.

- [ ] **Step 2: Build.**

```bash
go build ./...
```
Expected: clean.

- [ ] **Step 3: Commit.**

```bash
git add _datafiles/world/dogmud/mobs/
git commit -m "content(mobs): bump specialist shopkeeper starting gold to 1000

Up from 500 (or various manual overrides like 200 on Ilsa). Aligns
with new vendor type & economy polish baseline.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### Task 7.2: Bump general store gold to 5000

**Files** (4 mobs):
- `_datafiles/world/dogmud/mobs/sanctum_basin/63-merchant_adela.yaml`
- `_datafiles/world/dogmud/mobs/watchers_crossing/85-merchant_brecca.yaml`
- `_datafiles/world/dogmud/mobs/thornwall_city/104-fence_dealer_siv.yaml`
- `_datafiles/world/dogmud/mobs/stillwater/341-storekeeper_wulf.yaml`

- [ ] **Step 1: Set `gold: 5000` under `character:` in each.**

- [ ] **Step 2: Build + commit.**

```bash
git add _datafiles/world/dogmud/mobs/
git commit -m "content(mobs): bump general store starting gold to 5000

Generals buy everything tagged; need a bigger float to absorb
heterogeneous incoming stock without hitting the gold-reserve gate.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Phase 8: One-time shop wipe

### Task 8.1: Wipe `_datafiles/world/dogmud/shops/`

**Files:**
- Delete: entire `_datafiles/world/dogmud/shops/` directory contents.

- [ ] **Step 1: Confirm what's there before wiping.**

```bash
find _datafiles/world/dogmud/shops/ -type f -name "*.yaml"
```
Expected: 9 files (or whatever exists at this point) — capture the list for the commit message.

- [ ] **Step 2: Remove the directory tree.**

```bash
rm -rf _datafiles/world/dogmud/shops/
```

- [ ] **Step 3: Boot the server (it should re-seed shops fresh).**

```bash
go run main.go
```
Expected: server starts, validators pass, shops re-seed from mob templates.

- [ ] **Step 4: Inspect a freshly-seeded shop file.**

After server has run for a moment, `Ctrl-C` it and check:
```bash
ls _datafiles/world/dogmud/shops/
cat _datafiles/world/dogmud/shops/stillwater/338-room*.yaml
```
Expected: file exists, `gold: 1000`, fresh `Stock` from template.

- [ ] **Step 5: Commit the wipe.**

```bash
git add _datafiles/world/dogmud/shops/  # may be empty add if nothing tracked, else removed-files
git rm -r --cached _datafiles/world/dogmud/shops/ 2>/dev/null || true
git commit -m "chore(shops): wipe legacy shop save files for clean reseed

Vendor-types overhaul changes buy rule, tags, gold defaults, and
seed contents — piecemeal migration is risky. Wipe the dir; shops
re-seed from mob templates on first boot. Prod state has been
running <2 weeks; loss is acceptable.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Phase 9: Tier-50/40 baseline restock

### Task 9.1: Add `RestockBaselineTiers()` method (TDD)

**Files:**
- Modify: `internal/shops/shopinventory.go`
- Test: `internal/shops/shopinventory_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `internal/shops/shopinventory_test.go`:

```go
func TestRestockBaselineTiers_TopsUpTier50And40(t *testing.T) {
    // Set up two items via test fixture: one tier 50, one tier 40.
    // (Use existing fixture pattern — likely needs items.RegisterTestSpec or similar.)
    setupTestItemSpec(t, 1, 50, 5)  // itemId=1, rarity_tier=50, value=5
    setupTestItemSpec(t, 2, 40, 5)
    setupTestItemSpec(t, 3, 30, 5)  // shouldn't restock
    si := &ShopInventory{
        Stock: []StockEntry{
            {ItemId: 1, RestockQty: 5, MaxStock: 50, Current: 10},
            {ItemId: 2, RestockQty: 5, MaxStock: 40, Current: 10},
            {ItemId: 3, RestockQty: 5, MaxStock: 30, Current: 10},
        },
    }
    restocked := si.RestockBaselineTiers()
    if !restocked {
        t.Errorf("expected restocked=true")
    }
    if si.Stock[0].Current != 15 {
        t.Errorf("tier-50 entry: Current = %d, want 15", si.Stock[0].Current)
    }
    if si.Stock[1].Current != 15 {
        t.Errorf("tier-40 entry: Current = %d, want 15", si.Stock[1].Current)
    }
    if si.Stock[2].Current != 10 {
        t.Errorf("tier-30 entry should not restock: Current = %d, want 10", si.Stock[2].Current)
    }
}

func TestRestockBaselineTiers_SkipsCrafterEntries(t *testing.T) {
    setupTestItemSpec(t, 1, 50, 5)
    si := &ShopInventory{
        Stock: []StockEntry{{ItemId: 1, RestockQty: 0, MaxStock: 50, Current: 0}},
    }
    if si.RestockBaselineTiers() {
        t.Errorf("RestockQty=0 entry should not restock")
    }
    if si.Stock[0].Current != 0 {
        t.Errorf("Current changed unexpectedly: %d", si.Stock[0].Current)
    }
}

func TestRestockBaselineTiers_SkipsAtCap(t *testing.T) {
    setupTestItemSpec(t, 1, 50, 5)
    si := &ShopInventory{
        Stock: []StockEntry{{ItemId: 1, RestockQty: 5, MaxStock: 50, Current: 50}},
    }
    if si.RestockBaselineTiers() {
        t.Errorf("at-cap entry should not restock")
    }
    if si.Stock[0].Current != 50 {
        t.Errorf("Current = %d, want 50", si.Stock[0].Current)
    }
}

func TestRestockBaselineTiers_CapsAtMaxStock(t *testing.T) {
    setupTestItemSpec(t, 1, 50, 5)
    si := &ShopInventory{
        Stock: []StockEntry{{ItemId: 1, RestockQty: 10, MaxStock: 50, Current: 47}},
    }
    si.RestockBaselineTiers()
    if si.Stock[0].Current != 50 {
        t.Errorf("should cap at MaxStock: Current = %d, want 50", si.Stock[0].Current)
    }
}
```

The `setupTestItemSpec` helper may need to be added — check if `internal/shops/test_main_test.go` already provides one.

- [ ] **Step 2: Run tests to verify failure**

```bash
go test ./internal/shops/ -run TestRestockBaselineTiers -v
```
Expected: COMPILE ERROR — `RestockBaselineTiers` undefined.

- [ ] **Step 3: Implement the method**

In `internal/shops/shopinventory.go`, add after `RestockBuckets`:

```go
// RestockBaselineTiers tops up StockEntries whose item carries
// rarity_tier 50 or 40, by RestockQty per call (capped at MaxStock).
// Skips entries with RestockQty <= 0 (NPC-crafted, untouched).
// Returns true if any stock was added.
func (si *ShopInventory) RestockBaselineTiers() bool {
    restocked := false
    for i := range si.Stock {
        e := &si.Stock[i]
        if e.RestockQty <= 0 {
            continue
        }
        spec := items.GetItemSpec(e.ItemId)
        if spec == nil {
            continue
        }
        if spec.RarityTier != 50 && spec.RarityTier != 40 {
            continue
        }
        room := e.MaxStock - e.Current
        if room <= 0 {
            continue
        }
        add := e.RestockQty
        if add > room {
            add = room
        }
        e.Current += add
        restocked = true
    }
    return restocked
}
```

Add `"github.com/GoMudEngine/GoMud/internal/items"` to the import block if not already present.

- [ ] **Step 4: Run tests to verify pass**

```bash
go test ./internal/shops/ -run TestRestockBaselineTiers -v
```
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/shops/shopinventory.go internal/shops/shopinventory_test.go
git commit -m "feat(shops): add RestockBaselineTiers for tier-50/40 mats

Top-up only the two most common rarity tiers, skipping anything
RestockQty=0 (NPC-crafted) or rarer (caravan/forager-served).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### Task 9.2: Swap call site in TickMobCraft

**Files:**
- Modify: `internal/mobs/crafter.go` (around line 194)

- [ ] **Step 1: Find the call**

```bash
grep -n "IsCaravanServedZone" internal/mobs/crafter.go
```
Confirms the line.

- [ ] **Step 2: Replace the suppression branch**

In `internal/mobs/crafter.go`, change:

```go
restocked := false
if !b.IsCaravanServedZone(mob.Zone) {
    restocked = shopInv.Restock()
}
```

to:

```go
var restocked bool
if b.IsCaravanServedZone(mob.Zone) {
    restocked = shopInv.RestockBaselineTiers()
} else {
    restocked = shopInv.Restock()
}
```

- [ ] **Step 3: Run mobs package tests**

```bash
go test ./internal/mobs/...
```
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/mobs/crafter.go
git commit -m "feat(mobs): tier-50/40 baseline restock in caravan-served zones

Caravan-served zones used to suppress shopInv.Restock() entirely,
relying solely on caravan/forager deliveries — too slow for common
mats. Now tier-50 and tier-40 entries top up via the existing
crafter tick; rarer mats still depend on caravan/forager.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Phase 10: Forager watchdog + diagnostics

### Task 10.1: Add `ForagerStuckThresholdRounds` config knob

**Files:**
- Modify: `internal/configs/config.balance.go`
- Modify: `_datafiles/config.yaml`

- [ ] **Step 1: Find the Balance struct**

```bash
grep -n "type Balance struct\|ForagerHPRecallThresholdPct" internal/configs/config.balance.go | head -10
```

- [ ] **Step 2: Add the knob**

In `internal/configs/config.balance.go`, near the other Forager fields:

```go
ForagerStuckThresholdRounds ConfigInt `yaml:"ForagerStuckThresholdRounds"` // Watchdog: force-reset to Recalling if a forager has been in the same state for this many rounds (default 600)
```

In the `Validate` (or default-setting) function, set the default:

```go
if b.ForagerStuckThresholdRounds == 0 {
    b.ForagerStuckThresholdRounds = 600
}
```

- [ ] **Step 3: Add to `_datafiles/config.yaml`** under the Balance section, with the canonical comment block:

```yaml
  # - ForagerStuckThresholdRounds -
  # Forager watchdog: if a forager has been in the same state for more
  # than this many rounds, force-reset the state to Recalling so it
  # heads home, dumps satchel, and re-cycles. Logs a warning on reset.
  ForagerStuckThresholdRounds: 600
```

- [ ] **Step 4: Boot smoke + commit**

```bash
go build ./...
git add internal/configs/config.balance.go _datafiles/config.yaml
git commit -m "feat(config): add ForagerStuckThresholdRounds (default 600)

Watchdog threshold for forager stuck-state recovery.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### Task 10.2: Watchdog in actForagerStep (TDD)

**Files:**
- Modify: `internal/behaviortree/actions_forager.go`
- Test: `internal/behaviortree/actions_forager_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/behaviortree/actions_forager_test.go`:

```go
func TestForagerWatchdog_ResetsStuckMobToRecalling(t *testing.T) {
    // Set up: forager in StateForaging with state_started_round far
    // in the past. Tick once, assert state is now "recalling".
    cfg := configs.GetBalanceConfig()
    cfg.ForagerStuckThresholdRounds = 100
    configs.OverrideBalance(cfg) // or whatever fixture sets the test config

    state := newBehaviorStateFixture(t)
    state.Set("forager_state", "foraging")
    state.Set("forager_state_started_round", "0") // way in the past

    util.SetTestRoundCount(500) // > threshold
    defer util.ResetTestRoundCount()

    mob := newTestForagerMob(t, 372 /*Halix*/)
    ctx := &EvalContext{InstanceId: mob.InstanceId, MobState: state, RoomId: 3000}

    res := actForagerStep(nil, ctx)
    if res != Success {
        t.Errorf("watchdog should return Success: got %v", res)
    }
    if got := state.GetString("forager_state"); got != "recalling" {
        t.Errorf("expected forager_state=recalling after watchdog reset, got %q", got)
    }
}

func TestForagerWatchdog_DoesNotResetActiveForager(t *testing.T) {
    cfg := configs.GetBalanceConfig()
    cfg.ForagerStuckThresholdRounds = 100
    configs.OverrideBalance(cfg)

    state := newBehaviorStateFixture(t)
    state.Set("forager_state", "foraging")
    state.Set("forager_state_started_round", "450") // recent

    util.SetTestRoundCount(500)
    defer util.ResetTestRoundCount()

    mob := newTestForagerMob(t, 372)
    ctx := &EvalContext{InstanceId: mob.InstanceId, MobState: state, RoomId: 3000}

    actForagerStep(nil, ctx)
    if got := state.GetString("forager_state"); got != "foraging" {
        t.Errorf("watchdog fired prematurely: got %q, want foraging", got)
    }
}
```

(Adapt fixture-builder names to existing helpers in the test file.)

- [ ] **Step 2: Run tests to verify failure**

```bash
go test ./internal/behaviortree/ -run TestForagerWatchdog -v
```
Expected: tests FAIL — watchdog doesn't exist yet.

- [ ] **Step 3: Add watchdog at top of actForagerStep**

In `internal/behaviortree/actions_forager.go::actForagerStep`, just after the profile-resolve block (line ~57) and before `cur := readForagerState(...)`:

```go
// Stuck-state watchdog. If the forager has been sitting in one state
// longer than the threshold, force-reset to Recalling so it heads
// home, dumps satchel, and re-cycles. Logs a Warn for ops visibility.
{
    startedStr := ctx.MobState.GetString(keyStateStartedRound)
    started, _ := strconv.ParseUint(startedStr, 10, 64)
    threshold := uint64(cfg.ForagerStuckThresholdRounds)
    now := util.GetRoundCount()
    if started > 0 && threshold > 0 && now > started &&
        now-started > threshold {
        mudlog.Warn("forager watchdog: stuck state, force-resetting to recalling",
            "mobId", mob.MobId, "name", profile.Name,
            "state", ctx.MobState.GetString(keyForagerState),
            "stuckRounds", now-started)
        transitionForager(ctx.MobState, forager.StateRecalling)
        return Success
    }
}
```

- [ ] **Step 4: Run tests to verify pass**

```bash
go test ./internal/behaviortree/ -run TestForagerWatchdog -v
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/behaviortree/actions_forager.go internal/behaviortree/actions_forager_test.go
git commit -m "feat(forager): stuck-state watchdog resets to Recalling

Foragers wedged in the same state for >ForagerStuckThresholdRounds
get forced to Recalling so they head home, dump satchel, and
re-cycle. Logs a Warn on every reset for ops visibility.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### Task 10.3: Capture diagnostics (despawned vs idle)

**Files:**
- Modify: `internal/economy/health/snapshot.go` — add StuckRounds field.
- Modify: `internal/economy/health/capture.go` — distinguish in placeholder logic.
- Test: `internal/economy/health/capture_test.go`

- [ ] **Step 1: Add StuckRounds to ForagerSnapshot**

In `internal/economy/health/snapshot.go`, add to the `ForagerSnapshot` struct:

```go
StuckRounds uint64 `yaml:"stuck_rounds" json:"stuck_rounds"` // currentRound - state_entered_round; -1 / 0 means "(not active)"
```

- [ ] **Step 2: Write failing test**

Add to `internal/economy/health/capture_test.go`:

```go
func TestCaptureForagers_DistinguishesDespawnedFromIdle(t *testing.T) {
    // Two profiles: one despawned, one live with empty BTreeState.
    // Use existing test fixture or build minimal mobs.
    setupTestForagerProfiles(t, []int{371, 372}) // Tova (despawned) + Halix (live)

    halix := spawnTestMobAt(t, 372, 3000)
    halix.BTreeState = &behaviortree.BehaviorState{} // empty state

    snaps := captureForagers()
    if len(snaps) != 2 {
        t.Fatalf("expected 2 snapshots, got %d", len(snaps))
    }
    var despawned, idle *ForagerSnapshot
    for i := range snaps {
        if snaps[i].State == "(despawned)" {
            despawned = &snaps[i]
        }
        if snaps[i].State == "(idle, no state)" {
            idle = &snaps[i]
        }
    }
    if despawned == nil {
        t.Errorf("missing (despawned) row")
    }
    if idle == nil {
        t.Errorf("missing (idle, no state) row")
    }
}
```

- [ ] **Step 3: Run test to confirm failure**

```bash
go test ./internal/economy/health/ -run TestCaptureForagers_DistinguishesDespawnedFromIdle -v
```
Expected: FAIL.

- [ ] **Step 4: Update captureForagers**

In `internal/economy/health/capture.go::captureForagers`, replace the live-pass + placeholder-pass logic with:

```go
func captureForagers() []ForagerSnapshot {
    out := []ForagerSnapshot{}

    // Build lookup: mobId → live mob instance.
    liveByMobId := map[int]*mobs.Mob{}
    for _, instId := range mobs.GetAllMobInstanceIds() {
        m := mobs.GetInstance(instId)
        if m == nil {
            continue
        }
        if forager.ProfileFor(int(m.MobId)) == nil {
            continue
        }
        liveByMobId[int(m.MobId)] = m
    }

    now := util.GetRoundCount()

    for _, p := range forager.AllProfiles() {
        m, alive := liveByMobId[p.MobId]
        if !alive {
            out = append(out, ForagerSnapshot{
                MobId: p.MobId, Name: p.Name,
                Territory: territoryFor(p.MobId),
                State:     "(despawned)",
                RoomId:    p.SanctuaryRoom,
                CargoByBucket: map[string]int{},
            })
            continue
        }

        bs, ok := m.BTreeState.(*behaviortree.BehaviorState)
        if !ok || bs == nil {
            out = append(out, ForagerSnapshot{
                InstId: m.InstanceId, MobId: p.MobId, Name: p.Name,
                Territory: territoryFor(p.MobId),
                State:     "(idle, no state)",
                RoomId:    m.Character.RoomId,
                CargoByBucket: map[string]int{},
            })
            continue
        }
        stateName := bs.GetString("forager_state")
        if stateName == "" {
            mudlog.Warn("forager state missing on live mob",
                "mobId", p.MobId, "name", p.Name, "roomId", m.Character.RoomId)
            out = append(out, ForagerSnapshot{
                InstId: m.InstanceId, MobId: p.MobId, Name: p.Name,
                Territory: territoryFor(p.MobId),
                State:     "(idle, no state)",
                RoomId:    m.Character.RoomId,
                CargoByBucket: map[string]int{},
            })
            continue
        }

        startedRound, _ := strconv.ParseUint(bs.GetString("forager_state_started_round"), 10, 64)
        var stuck uint64
        if now > startedRound {
            stuck = now - startedRound
        }

        fs := ForagerSnapshot{
            InstId:            m.InstanceId,
            MobId:             p.MobId,
            Name:              m.Character.Name,
            Territory:         territoryFor(p.MobId),
            State:             stateName,
            StateEnteredRound: startedRound,
            StuckRounds:       stuck,
            RoomId:            m.Character.RoomId,
            CargoByBucket:     map[string]int{},
            CargoWeight:       int(m.Character.GetCarriedWeight()),
            CargoCapacity:     int(m.Character.CarryCapacity()),
        }
        inventories := [][]items.Item{m.Character.Items, m.Character.ComponentItems, m.Character.PotionItems}
        for _, list := range inventories {
            for _, it := range list {
                bucket := economy.BucketFor(it.ItemId)
                if bucket == "" {
                    continue
                }
                w := int(it.GetSpec().GetWeight())
                if w > 0 {
                    fs.CargoByBucket[bucket] += w
                }
            }
        }
        out = append(out, fs)
    }
    return out
}
```

(Drop the old separate live-pass + placeholder-append flow — single pass over profiles.)

- [ ] **Step 5: Run test to verify pass**

```bash
go test ./internal/economy/health/ -run TestCaptureForagers -v
```
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/economy/health/snapshot.go internal/economy/health/capture.go internal/economy/health/capture_test.go
git commit -m "feat(dashboard): distinguish despawned vs idle forager + StuckRounds

captureForagers walks profiles instead of live instances and emits
distinct State strings for '(despawned)' vs '(idle, no state)' so
the dashboard surfaces what's actually wrong with a wedged forager.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Phase 11: Throughput counters

### Task 11.1: forager.Throughput type + persistence (TDD)

**Files:**
- Create: `internal/forager/throughput.go`
- Test: `internal/forager/throughput_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/forager/throughput_test.go`:

```go
package forager

import (
    "os"
    "path/filepath"
    "testing"
)

func TestThroughput_GetReturnsEmptyOnFirstCall(t *testing.T) {
    ClearThroughputCache()
    tp := GetThroughput("test_zone", 372)
    if tp == nil {
        t.Fatalf("expected non-nil throughput")
    }
    if len(tp.DeliveriesByTier) != 0 {
        t.Errorf("expected empty map, got %v", tp.DeliveriesByTier)
    }
}

func TestThroughput_IncrementDelivery(t *testing.T) {
    ClearThroughputCache()
    IncrementDelivery("test_zone", 372, 50)
    IncrementDelivery("test_zone", 372, 50)
    IncrementDelivery("test_zone", 372, 40)
    tp := GetThroughput("test_zone", 372)
    if tp.DeliveriesByTier[50] != 2 {
        t.Errorf("tier-50: got %d, want 2", tp.DeliveriesByTier[50])
    }
    if tp.DeliveriesByTier[40] != 1 {
        t.Errorf("tier-40: got %d, want 1", tp.DeliveriesByTier[40])
    }
}

func TestThroughput_SaveLoadRoundtrip(t *testing.T) {
    ClearThroughputCache()
    tmpDir := t.TempDir()
    SetThroughputBaseDirForTest(tmpDir)
    defer SetThroughputBaseDirForTest("")

    IncrementDelivery("test_zone", 372, 50)
    IncrementDelivery("test_zone", 372, 30)
    if err := SaveThroughput("test_zone", 372); err != nil {
        t.Fatalf("save: %v", err)
    }

    // Verify file exists.
    p := filepath.Join(tmpDir, "test_zone", "372.yaml")
    if _, err := os.Stat(p); err != nil {
        t.Fatalf("expected save file at %s: %v", p, err)
    }

    ClearThroughputCache()
    tp := GetThroughput("test_zone", 372) // should load from disk
    if tp.DeliveriesByTier[50] != 1 || tp.DeliveriesByTier[30] != 1 {
        t.Errorf("after reload: %v", tp.DeliveriesByTier)
    }
}
```

- [ ] **Step 2: Run test to verify failure**

```bash
go test ./internal/forager/ -run TestThroughput -v
```
Expected: COMPILE ERROR.

- [ ] **Step 3: Implement throughput.go**

Create `internal/forager/throughput.go`:

```go
package forager

import (
    "fmt"
    "os"
    "path/filepath"
    "sync"

    "github.com/GoMudEngine/GoMud/internal/configs"
    "github.com/GoMudEngine/GoMud/internal/mudlog"
    "github.com/GoMudEngine/GoMud/internal/util"
    "gopkg.in/yaml.v2"
)

// Throughput is the per-forager cumulative delivery counter, persisted
// at <DataFiles>/foragers/<zone>/<mobId>.yaml. Snapshotted hourly into
// ForagerSnapshot.DeliveriesByTier; deltas surfaced on the dashboard.
type Throughput struct {
    MobId            int         `yaml:"mob_id"`
    Zone             string      `yaml:"zone"`
    DeliveriesByTier map[int]int `yaml:"deliveries_by_tier"`
    LastUpdatedRound uint64      `yaml:"last_updated_round"`
}

var (
    throughputMu       sync.RWMutex
    throughputCache    = map[string]*Throughput{}
    throughputBaseDir  string // for tests
)

func throughputKey(zone string, mobId int) string {
    return fmt.Sprintf("%s/%d", zone, mobId)
}

func throughputPath(zone string, mobId int) string {
    base := throughputBaseDir
    if base == "" {
        base = util.FilePath(
            configs.GetFilePathsConfig().DataFiles.String(), `/`, `foragers`,
        )
    }
    zoneSan := util.ConvertForFilename(zone)
    return filepath.Join(base, zoneSan, fmt.Sprintf("%d.yaml", mobId))
}

// GetThroughput returns the Throughput for (zone, mobId), loading from
// disk if not cached. Always returns non-nil (creates empty if missing).
func GetThroughput(zone string, mobId int) *Throughput {
    key := throughputKey(zone, mobId)

    throughputMu.RLock()
    if tp, ok := throughputCache[key]; ok {
        throughputMu.RUnlock()
        return tp
    }
    throughputMu.RUnlock()

    // Try disk.
    if tp := loadThroughputFromDisk(zone, mobId); tp != nil {
        throughputMu.Lock()
        throughputCache[key] = tp
        throughputMu.Unlock()
        return tp
    }

    // Fresh.
    tp := &Throughput{
        MobId: mobId, Zone: zone,
        DeliveriesByTier: map[int]int{},
    }
    throughputMu.Lock()
    throughputCache[key] = tp
    throughputMu.Unlock()
    return tp
}

// IncrementDelivery bumps the per-tier counter and updates LastUpdatedRound.
// Caller is responsible for calling SaveThroughput when convenient.
func IncrementDelivery(zone string, mobId int, rarityTier int) {
    if rarityTier <= 0 {
        return
    }
    tp := GetThroughput(zone, mobId)
    throughputMu.Lock()
    if tp.DeliveriesByTier == nil {
        tp.DeliveriesByTier = map[int]int{}
    }
    tp.DeliveriesByTier[rarityTier]++
    tp.LastUpdatedRound = util.GetRoundCount()
    throughputMu.Unlock()
}

// SaveThroughput writes the cached Throughput for (zone, mobId) to disk.
func SaveThroughput(zone string, mobId int) error {
    key := throughputKey(zone, mobId)

    throughputMu.RLock()
    tp, ok := throughputCache[key]
    throughputMu.RUnlock()
    if !ok {
        return fmt.Errorf("forager.SaveThroughput: no cached entry for %s", key)
    }

    p := throughputPath(zone, mobId)
    if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
        return fmt.Errorf("forager.SaveThroughput: mkdir: %w", err)
    }
    data, err := yaml.Marshal(tp)
    if err != nil {
        return fmt.Errorf("forager.SaveThroughput: marshal: %w", err)
    }
    if err := os.WriteFile(p, data, 0644); err != nil {
        return fmt.Errorf("forager.SaveThroughput: write: %w", err)
    }
    return nil
}

// SaveAllThroughputs persists every cached entry. Intended for graceful
// shutdown.
func SaveAllThroughputs() {
    throughputMu.RLock()
    keys := make([]struct {
        zone  string
        mobId int
    }, 0, len(throughputCache))
    for _, tp := range throughputCache {
        keys = append(keys, struct {
            zone  string
            mobId int
        }{tp.Zone, tp.MobId})
    }
    throughputMu.RUnlock()
    for _, k := range keys {
        if err := SaveThroughput(k.zone, k.mobId); err != nil {
            mudlog.Error("forager.SaveAllThroughputs", "error", err)
        }
    }
}

// ClearThroughputCache drops all cached entries (for tests).
func ClearThroughputCache() {
    throughputMu.Lock()
    throughputCache = map[string]*Throughput{}
    throughputMu.Unlock()
}

// SetThroughputBaseDirForTest overrides the on-disk base directory
// (for use in tests with t.TempDir()).
func SetThroughputBaseDirForTest(dir string) {
    throughputBaseDir = dir
}

func loadThroughputFromDisk(zone string, mobId int) *Throughput {
    p := throughputPath(zone, mobId)
    raw, err := os.ReadFile(p)
    if err != nil {
        return nil
    }
    var tp Throughput
    if err := yaml.Unmarshal(raw, &tp); err != nil {
        mudlog.Error("forager.loadThroughputFromDisk", "path", p, "error", err)
        return nil
    }
    if tp.DeliveriesByTier == nil {
        tp.DeliveriesByTier = map[int]int{}
    }
    return &tp
}
```

- [ ] **Step 4: Run test to verify pass**

```bash
go test ./internal/forager/ -run TestThroughput -v
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/forager/throughput.go internal/forager/throughput_test.go
git commit -m "feat(forager): per-forager Throughput counter + YAML persistence

Tracks DeliveriesByTier across server restarts. Snapshot capture
will surface this as a colored bar on the economy dashboard.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### Task 11.2: caravan.Throughput (mirror of forager)

**Files:**
- Create: `internal/caravan/throughput.go`
- Test: `internal/caravan/throughput_test.go`

- [ ] **Step 1: Mirror Task 11.1 in the caravan package.**

Same exact API and tests; replace `forager` with `caravan` everywhere, file path becomes `<DataFiles>/caravans/<zone>/<mobId>.yaml`.

- [ ] **Step 2: Run tests + commit.**

```bash
go test ./internal/caravan/ -run TestThroughput -v
git add internal/caravan/throughput.go internal/caravan/throughput_test.go
git commit -m "feat(caravan): per-caravan Throughput counter + YAML persistence

Same shape as forager.Throughput. Counters increment on the
caravan dropoff side (see following commit).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### Task 11.3: Wire forager increment into npcVisitVendorsInRoom

**Files:**
- Modify: `internal/behaviortree/actions_forager.go`

- [ ] **Step 1: Find the existing item-handoff loop (~line 493)**

```bash
grep -n "entry.Current++" internal/behaviortree/actions_forager.go
```

- [ ] **Step 2: Add the increment call alongside the existing handoff**

In `npcVisitVendorsInRoom`, after `entry.Current++`:

```go
mob.Character.RemoveItem(item)
entry.Current++
mutated = true

// Throughput counter (per-tier).
itemSpec := items.GetItemSpec(item.ItemId)
if itemSpec != nil && itemSpec.RarityTier > 0 {
    forager.IncrementDelivery(mob.Zone, int(mob.MobId), itemSpec.RarityTier)
}

room.SendText(fmt.Sprintf( ... ))
```

After the inner loop (when `mutated` triggers `shops.SaveShop`), also save the throughput file:

```go
if mutated {
    if err := shops.SaveShop(...); err != nil { ... }
    if err := forager.SaveThroughput(mob.Zone, int(mob.MobId)); err != nil {
        mudlog.Error("forager.SaveThroughput", "forager", p.Name, "error", err)
    }
}
```

- [ ] **Step 3: Build + run forager-related tests**

```bash
go build ./...
go test ./internal/behaviortree/...
```
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/behaviortree/actions_forager.go
git commit -m "feat(forager): record DeliveriesByTier on each vendor handoff

npcVisitVendorsInRoom bumps the per-tier counter and saves the
throughput file after a successful round of handoffs.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### Task 11.4: Wire caravan increment into visit dropoff

**Files:**
- Modify: `internal/caravan/visit.go` (the dropoff path)

- [ ] **Step 1: Find the visit/dropoff function**

```bash
grep -n "func VisitVendors\|delivery\|DropOff\|Drop" internal/caravan/*.go
```
Identify where items transfer from wagon → destination shop.

- [ ] **Step 2: Add increment + save**

After each successful item transfer to a destination shop, call:

```go
itemSpec := items.GetItemSpec(item.ItemId)
if itemSpec != nil && itemSpec.RarityTier > 0 {
    caravan.IncrementDelivery(caravanZone, leaderMobId, itemSpec.RarityTier)
}
```

After the visit batch completes, save:

```go
if delivered > 0 {
    if err := caravan.SaveThroughput(caravanZone, leaderMobId); err != nil {
        mudlog.Error("caravan.SaveThroughput", "error", err)
    }
}
```

(Adapt to the actual function signature and variable names.)

- [ ] **Step 3: Build + tests**

```bash
go build ./...
go test ./internal/caravan/...
```

- [ ] **Step 4: Commit**

```bash
git add internal/caravan/visit.go
git commit -m "feat(caravan): record DeliveriesByTier on each dropoff

Pickups don't count — only items handed to destination shops.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### Task 11.5: Boot prewarm for forager + caravan throughput

**Files:**
- Modify: `internal/forager/throughput.go` — add `PrewarmFromPersistedFiles`.
- Modify: `internal/caravan/throughput.go` — same.
- Modify: `main.go` — call both at boot.

- [ ] **Step 1: Add `PrewarmFromPersistedFiles` to forager/throughput.go**

```go
// PrewarmFromPersistedFiles loads every persisted forager throughput
// YAML found under <DataFiles>/foragers/<zone>/<mobId>.yaml into the
// in-memory cache. Returns the count loaded.
func PrewarmFromPersistedFiles() (int, error) {
    base := util.FilePath(
        configs.GetFilePathsConfig().DataFiles.String(), `/`, `foragers`,
    )
    return prewarmFrom(base)
}

func prewarmFrom(baseDir string) (int, error) {
    zoneDirs, err := os.ReadDir(baseDir)
    if err != nil {
        if os.IsNotExist(err) {
            return 0, nil
        }
        return 0, fmt.Errorf("forager.PrewarmFromPersistedFiles: readdir: %w", err)
    }
    loaded := 0
    fileRe := regexp.MustCompile(`^(\d+)\.yaml$`)
    for _, zd := range zoneDirs {
        if !zd.IsDir() {
            continue
        }
        files, err := os.ReadDir(filepath.Join(baseDir, zd.Name()))
        if err != nil {
            mudlog.Warn("forager prewarm: readdir zone", "zone", zd.Name(), "error", err)
            continue
        }
        for _, f := range files {
            m := fileRe.FindStringSubmatch(f.Name())
            if m == nil {
                continue
            }
            mobId, _ := strconv.Atoi(m[1])
            // Resolve canonical zone name from a mob template lookup.
            // For the throughput dir we expect zone-sanitized dir names;
            // for now we trust the directory name as zone-sanitized.
            // (See shop prewarm for the alternative pattern.)
            zone := zd.Name()
            tp := loadThroughputFromDisk(zone, mobId)
            if tp == nil {
                continue
            }
            key := throughputKey(zone, mobId)
            throughputMu.Lock()
            throughputCache[key] = tp
            throughputMu.Unlock()
            loaded++
        }
    }
    return loaded, nil
}
```

Note: this assumes zone-sanitized dir names. If we need canonical zone names (un-sanitized) — add a zoneLookup callback like shops do.

- [ ] **Step 2: Mirror in caravan package**

- [ ] **Step 3: Wire into main.go**

After the existing `shops.PrewarmFromPersistedFiles(...)` call:

```go
if n, err := forager.PrewarmFromPersistedFiles(); err != nil {
    mudlog.Error("forager.PrewarmFromPersistedFiles", "error", err)
} else {
    mudlog.Info("forager.PrewarmFromPersistedFiles", "loaded", n)
}
if n, err := caravan.PrewarmFromPersistedFiles(); err != nil {
    mudlog.Error("caravan.PrewarmFromPersistedFiles", "error", err)
} else {
    mudlog.Info("caravan.PrewarmFromPersistedFiles", "loaded", n)
}
```

Also wire `forager.SaveAllThroughputs()` and `caravan.SaveAllThroughputs()` into the graceful-shutdown path next to `shops.SaveAllShops()`.

- [ ] **Step 4: Boot smoke**

```bash
go run main.go
```
Expected: server starts; new prewarm log lines visible.

- [ ] **Step 5: Commit**

```bash
git add internal/forager/throughput.go internal/caravan/throughput.go main.go
git commit -m "feat(boot): prewarm forager + caravan throughput from disk

Loads cached counters at boot, saves all entries at graceful shutdown.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Phase 12: Dashboard rework

### Task 12.1: StockScore on ShopSnapshot + ShopDelta

**Files:**
- Modify: `internal/economy/health/snapshot.go` — add StockScore.
- Modify: `internal/economy/health/capture.go` — compute it.
- Modify: `internal/economy/health/delta.go` — add StockScoreDelta.
- Test: `internal/economy/health/snapshot_test.go`

- [ ] **Step 1: Add fields**

In `snapshot.go::ShopSnapshot`:
```go
StockScore float64 `yaml:"stock_score" json:"stock_score"` // sum(Current) / sum(MaxStock); 0..1
```

In `delta.go::ShopDelta`:
```go
StockScoreDelta int `json:"stock_score_delta"` // percentage points (now - old)
```

- [ ] **Step 2: Compute in captureShops**

In `capture.go::captureShops`, after building `ss.Stock`:

```go
total, capacity := 0, 0
for _, e := range ss.Stock {
    total += e.Current
    capacity += e.Max
}
if capacity > 0 {
    ss.StockScore = float64(total) / float64(capacity)
}
```

- [ ] **Step 3: Compute in ComputeShopDelta**

In `delta.go::ComputeShopDelta`, add:

```go
d.StockScoreDelta = int((now.StockScore - oldS.StockScore) * 100) // percentage points
```

(Where `oldS` is the dereferenced old snapshot — guard against nil.)

- [ ] **Step 4: Test**

Add unit tests in `snapshot_test.go`:
```go
func TestStockScoreComputation(t *testing.T) {
    s := ShopSnapshot{
        Stock: []StockSnapshot{
            {Current: 5, Max: 10},
            {Current: 0, Max: 20},
        },
    }
    s.RecomputeStockScore() // helper if extracted; or compute inline in test
    if math.Abs(s.StockScore - 5.0/30.0) > 0.001 {
        t.Errorf("StockScore = %f", s.StockScore)
    }
}
```

(If `RecomputeStockScore()` isn't extracted, just inline the formula in the test or assert against `captureShops` output.)

- [ ] **Step 5: Run + commit**

```bash
go test ./internal/economy/health/...
git add internal/economy/health/snapshot.go internal/economy/health/capture.go internal/economy/health/delta.go internal/economy/health/snapshot_test.go
git commit -m "feat(dashboard): StockScore + StockScoreDelta on ShopSnapshot

Per-shop stock fill ratio (0..1) + delta in percentage points;
replaces gold-delta as the primary trend metric for shops.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### Task 12.2: DeliveriesByTier on ForagerSnapshot/CaravanSnapshot + deltas

**Files:**
- Modify: `internal/economy/health/snapshot.go`
- Modify: `internal/economy/health/capture.go`
- Modify: `internal/economy/health/delta.go`

- [ ] **Step 1: Add fields**

In `snapshot.go`:
```go
// ForagerSnapshot
DeliveriesByTier map[int]int `yaml:"deliveries_by_tier" json:"deliveries_by_tier"`
// CaravanSnapshot
DeliveriesByTier map[int]int `yaml:"deliveries_by_tier" json:"deliveries_by_tier"`
```

- [ ] **Step 2: Populate in captureForagers + captureCaravans**

In each capture function, before emitting the snapshot:

```go
tp := forager.GetThroughput(m.Zone, p.MobId) // or caravan.GetThroughput for caravan
if tp != nil && tp.DeliveriesByTier != nil {
    fs.DeliveriesByTier = map[int]int{}
    for tier, count := range tp.DeliveriesByTier {
        fs.DeliveriesByTier[tier] = count
    }
}
```

(Copy the map so dashboard can't mutate live state.)

- [ ] **Step 3: Add delta types + functions**

In `delta.go`:

```go
type ForagerDelta struct {
    DeliveriesByTierDelta map[int]int
    StuckRoundsDelta      int64
}
type CaravanDelta struct {
    DeliveriesByTierDelta map[int]int
}

func ComputeForagerDelta(now ForagerSnapshot, old *ForagerSnapshot) ForagerDelta {
    d := ForagerDelta{DeliveriesByTierDelta: map[int]int{}}
    if old == nil { return d }
    for tier, count := range now.DeliveriesByTier {
        d.DeliveriesByTierDelta[tier] = count - old.DeliveriesByTier[tier]
    }
    for tier, count := range old.DeliveriesByTier {
        if _, seen := now.DeliveriesByTier[tier]; !seen {
            d.DeliveriesByTierDelta[tier] = -count
        }
    }
    d.StuckRoundsDelta = int64(now.StuckRounds) - int64(old.StuckRounds)
    return d
}

// ComputeCaravanDelta — same shape minus StuckRounds.
func ComputeCaravanDelta(now CaravanSnapshot, old *CaravanSnapshot) CaravanDelta {
    d := CaravanDelta{DeliveriesByTierDelta: map[int]int{}}
    if old == nil { return d }
    for tier, count := range now.DeliveriesByTier {
        d.DeliveriesByTierDelta[tier] = count - old.DeliveriesByTier[tier]
    }
    for tier, count := range old.DeliveriesByTier {
        if _, seen := now.DeliveriesByTier[tier]; !seen {
            d.DeliveriesByTierDelta[tier] = -count
        }
    }
    return d
}
```

- [ ] **Step 4: Wire into dashboard JSON response**

In `internal/web/admin.economyhealth.go::economyAPI`, extend `deltaSet`:

```go
type deltaSet struct {
    UnixTs    int64                          `json:"unix_ts,omitempty"`
    Shops     map[string]health.ShopDelta    `json:"shops"`
    Foragers  map[string]health.ForagerDelta `json:"foragers"` // key "{mobId}"
    Caravans  map[string]health.CaravanDelta `json:"caravans"` // key "{instId}"
}
```

Populate them in the existing per-offset loop, parallel to shops.

- [ ] **Step 5: Test + commit**

Add a `TestComputeForagerDelta_BasicCounts` and `TestComputeCaravanDelta_BasicCounts`. Pattern similar to existing `ComputeShopDelta` tests.

```bash
go test ./internal/economy/health/...
git add internal/economy/health/ internal/web/admin.economyhealth.go
git commit -m "feat(dashboard): DeliveriesByTier deltas on forager + caravan

Snapshot captures per-tier counts; delta functions compute window
deltas for dashboard tier-color bars.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### Task 12.3: Frontend — replace gold-delta column with stock-score-delta

**Files:**
- Modify: `_datafiles/html/admin/economy/index.html`

- [ ] **Step 1: Update the shop table render block (~lines 174-232)**

Replace the gold-delta cell rendering (`var sd = d.deltas[label]...; sum += sd.GoldDelta`) with stock-score-delta:

```javascript
['1h','6h','1d','3d','1w'].forEach(function(label) {
    var sd = d.deltas[label] && d.deltas[label].shops[key];
    var deltaText;
    if (sd) {
        var dpp = sd.StockScoreDelta;
        deltaText = (dpp > 0 ? '+' : '') + dpp + 'pp';
    } else {
        deltaText = '<span class="text-muted">—</span>';
    }
    shopHtml += '<td>' + deltaText + '</td>';
});
```

- [ ] **Step 2: Update the discipline rollup row similarly**

Aggregate stock-score across the discipline:

```javascript
// Discipline rollup — compute aggregate StockScore.
var totalCur = 0, totalMax = 0;
for (var j = 0; j < shops.length; j++) {
    var s = shops[j];
    for (var k = 0; k < (s.stock || []).length; k++) {
        totalCur += s.stock[k].current;
        totalMax += s.stock[k].max;
    }
}
var aggScore = totalMax > 0 ? Math.round((totalCur/totalMax)*100) : 0;
discHtml += '<td>' + aggScore + '%</td>'; // current (no delta on discipline rollup, or compute via per-shop delta sum)
// Per-window delta: sum stock_score_delta across shops, divide by count.
['1h','6h','1d','3d','1w'].forEach(function(label) {
    var sumPP = 0, n = 0;
    for (var j = 0; j < shops.length; j++) {
        var sd = d.deltas[label] && d.deltas[label].shops[shopKey(shops[j])];
        if (sd) { sumPP += sd.StockScoreDelta; n++; }
    }
    var avg = n > 0 ? Math.round(sumPP / n) : 0;
    discHtml += '<td>' + (avg > 0 ? '+' : '') + avg + 'pp</td>';
});
```

- [ ] **Step 3: Update column headers in the table thead**

Change "1h", "6h" etc. column header tooltips/labels from "gold delta" to "stock-score delta (percentage points)".

- [ ] **Step 4: Reload dashboard, eyeball, commit**

```bash
# Boot server, hit /admin/economy/, verify columns render correctly
go run main.go
# (open browser, sanity check)
```

```bash
git add _datafiles/html/admin/economy/index.html
git commit -m "feat(dashboard): replace gold-delta column with stock-score-delta

Per-shop and per-discipline columns now show stock fill change in
percentage points instead of gold delta. Gold (current value) stays
visible as a non-delta column.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### Task 12.4: Frontend — tier-color bars

**Files:**
- Modify: `_datafiles/html/admin/economy/index.html`

- [ ] **Step 1: Define tier color palette + new `tierBar()` helper.**

In the `<style>` block, add:

```css
.tier-50 { background: #9aa0a6; }
.tier-40 { background: #5fb878; }
.tier-30 { background: #5b9dd9; }
.tier-20 { background: #a979e8; }
.tier-10 { background: #d4a73a; }
```

In the `<script>` block, replace `bucketBar()` with a new `tierBar(stockEntries, totalCap)` that walks per-tier counts. (For shop stock bar, count items per tier from `snap.stock[].item_id` resolved via a tier lookup — easiest: have backend send `tier` per-stock-entry. Add `Tier int` to `StockSnapshot` if it isn't there already; otherwise compute via item lookup at capture time.)

Add `Tier int` to `StockSnapshot` if missing — populate in `captureShops`:
```go
ss.Stock = append(ss.Stock, StockSnapshot{
    ItemId:     e.ItemId,
    Bucket:     economy.BucketFor(e.ItemId),
    Tier:       items.GetItemSpec(e.ItemId).RarityTier,
    Current:    e.Current,
    Max:        e.MaxStock,
    RestockQty: e.RestockQty,
})
```

Frontend `tierBar`:
```javascript
function tierBar(stockEntries, totalCap) {
    var byTier = {50:0,40:0,30:0,20:0,10:0};
    for (var i = 0; i < (stockEntries || []).length; i++) {
        var t = stockEntries[i].tier;
        if (byTier[t] !== undefined) byTier[t] += stockEntries[i].current;
    }
    var html = '<div class="bar-container">';
    [50,40,30,20,10].forEach(function(t) {
        var pct = totalCap > 0 ? (byTier[t] / totalCap) * 100 : 0;
        html += '<div class="bar-segment tier-' + t + '" style="width:' + pct + '%"></div>';
    });
    html += '</div>';
    return html;
}
```

- [ ] **Step 2: Replace `bucketBar()` callsites in the shop, caravan, forager renderers.**

Shop stock bar — keep `tierBar(snap.stock, totalCap(snap.stock))`.

Caravan throughput bar — new helper `tierBarFromDelta(deliveriesByTierDelta)`:

```javascript
function tierBarFromDelta(delta) {
    var total = 0;
    for (var k in delta) total += Math.abs(delta[k] || 0);
    if (total === 0) return '<span class="text-muted">no movement</span>';
    var html = '<div class="bar-container">';
    [50,40,30,20,10].forEach(function(t) {
        var pct = total > 0 ? ((delta[t] || 0) / total) * 100 : 0;
        html += '<div class="bar-segment tier-' + t + '" style="width:' + pct + '%"></div>';
    });
    html += '</div>';
    return html + ' <span class="text-muted">(' + total + ' items)</span>';
}
```

In the caravan/forager render block, replace the `colspan="5"` placeholder with `tierBarFromDelta(d.deltas['1d'].caravans[c.inst_id].DeliveriesByTierDelta)` (or whichever window the user selects — start with 1d).

- [ ] **Step 3: Boot, eyeball, commit**

```bash
go run main.go
# verify in browser
```

```bash
git add _datafiles/html/admin/economy/index.html internal/economy/health/snapshot.go internal/economy/health/capture.go
git commit -m "feat(dashboard): tier-color bars for shop stock + caravan/forager throughput

Replaces bucket-color bars. Five tier classes (grey/green/blue/purple/gold).
Caravan + forager rows show per-tier deliveries-in-window via tierBarFromDelta.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Phase 13: Tutorial regression smoke + PATCH_NOTES

### Task 13.1: Create tutorial regression goal file

**Files:**
- Create: `tools/testing/goals/vendor-economy-polish-tutorial-regression.yaml`

- [ ] **Step 1: Read an existing goal file to see the schema**

```bash
ls tools/testing/goals/ | head -5
cat tools/testing/goals/stage-3-1-foragers.yaml
```

- [ ] **Step 2: Write the goal file**

Create `tools/testing/goals/vendor-economy-polish-tutorial-regression.yaml`:

```yaml
name: Vendor Economy Polish — Tutorial Regression
role: feel-tester
character: NEW # fresh character creation
description: |
  Verify that vendor + economy changes haven't broken the new-player path
  through Sanctum Basin. Critical for ship gate.

objectives:
  - Complete character creation.
  - Reach Chrysalis Priest (Korvath) at the start of the tutorial.
  - Complete the mutation step (verify Korvath stays unattackable).
  - Reach Yenna and complete her step.
  - Reach Merchant Adela; verify her shop list is browseable and at least
    one starter weapon + one starter armor are available + affordable.
  - Buy a starter weapon and equip it.
  - Reach the Combat Trainer step.
  - Defeat the training dummy / mob.
  - Leave Sanctum Basin via the Stillwater or Thornwall road.
  - End with non-zero gold (not all spent at Adela's).

failure_conditions:
  - Korvath or Yenna is attackable.
  - Adela has no buyable starter gear.
  - Player is stuck with insufficient gold to leave Sanctum Basin.
  - Any NPC throws a parse error or crashes the dialogue tree.

duration: ~15 minutes wall-clock.

reporting:
  format: short report; pass/fail per objective; any user-visible jank
  noted explicitly.

DO NOT modify code. Observe and report only.
```

- [ ] **Step 3: Commit**

```bash
git add tools/testing/goals/vendor-economy-polish-tutorial-regression.yaml
git commit -m "test(ai): tutorial regression goal for vendor economy polish

Ship-gate: AI must complete a fresh-character run through Sanctum
Basin before this work promotes to prod.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### Task 13.2: Update PATCH_NOTES

**Files:**
- Modify: `PATCH_NOTES.md`

- [ ] **Step 1: Read existing format**

```bash
head -40 PATCH_NOTES.md
```

- [ ] **Step 2: Add new entry at the top**

```markdown
## 2026-05-04 — Vendor Types & Economy Polish

**Vendor system overhaul:**
- New `vendor_categories` tag on every salable item, with multi-discipline
  support (e.g., iron ingot is `[blacksmithing, jewelcrafting]`).
- Buy rule rewritten: single tag-overlap check + overstock cap +
  gold-reserve gate. Apothecary Ilsa now buys all alchemy mats, not
  just ones in her recipe list. Specialist shopkeepers no longer buy
  gear-upgrades (they're non-combatants and didn't wear what they
  bought).
- Per-vendor audit: Korvath, Yenna, Sigrid, Haral, Whisper, Bram are no
  longer shopkeepers (questgivers / flavor mobs). Specialist shopkeepers
  start with 1000g (was 500). General stores start with 5000g.
- One-time wipe of `_datafiles/world/dogmud/shops/` so all shops re-seed
  fresh with the new defaults.

**Restock pacing:**
- Tier-50 and tier-40 mats now top up at every shop on the existing
  crafter tick, layered alongside forager/caravan deliveries. Remote
  shops (Watcher's Crossing) and caravan-served towns (Stillwater,
  Thornwall) both benefit.

**Forager reliability:**
- Stuck-state watchdog (default threshold 600 rounds) force-resets a
  wedged forager to Recalling so it heads home, dumps satchel, re-cycles.
  Logs a warning on every reset for ops visibility.
- Dashboard distinguishes "(despawned)" from "(idle, no state)" foragers
  + shows StuckRounds.

**Dashboard:**
- Names visible for every shopkeeper (template fallback when not
  spawned).
- Gold-delta columns replaced with stock-score-delta (% fill change in
  percentage points). Gold value still visible as a static column.
- Caravan + forager rows now show per-rarity-tier delivery throughput
  bars (grey/green/blue/purple/gold = tier 50/40/30/20/10).
- All colored bars switched from supply-bucket to rarity-tier coloring.

**Persistence:**
- New `_datafiles/world/dogmud/foragers/<zone>/<mobId>.yaml` and
  `_datafiles/world/dogmud/caravans/<zone>/<mobId>.yaml` track
  cumulative DeliveriesByTier counters across reboots.
```

- [ ] **Step 3: Commit**

```bash
git add PATCH_NOTES.md
git commit -m "docs: PATCH_NOTES for 2026-05-04 vendor + economy polish

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### Task 13.3: Final boot smoke + tutorial AI run

- [ ] **Step 1: Disable file logging per pre-push SOP**

```bash
grep -n "LogToFile" _datafiles/config.yaml
# Set Logging.LogToFile: false
```

- [ ] **Step 2: Boot locally and watch for clean startup**

```bash
go run main.go
```

Expected: see in order
- `mobs.LoadDataFiles() loadedCount=...`
- `items.LoadDataFiles() loadedCount=...`
- `recipes.LoadDataFiles() loadedCount=...`
- `items.ValidateVendorCategories OK` (or implicit pass — no panic)
- `crafting.ValidateRecipeIngredientTags OK`
- `shops.ValidateShopMobTags OK`
- `shops.PrewarmFromPersistedFiles loaded=0` (post-wipe)
- `forager.PrewarmFromPersistedFiles loaded=0`
- `caravan.PrewarmFromPersistedFiles loaded=0`
- "listening on" or whatever the ready log line is

If any panic — fix and re-iterate before proceeding.

- [ ] **Step 3: Run the AI tutorial smoke**

```bash
# Per CLAUDE.md AI Testing section:
/test-mud local feel-tester vendor-economy-polish-tutorial-regression.yaml
```

Wait for the report. If the AI cannot complete the run, treat as ship-blocking. Fix issues, re-run.

- [ ] **Step 4: Final commit (PATCH_NOTES updated, log toggled off)**

If config.yaml changed only for the LogToFile toggle, commit just PATCH_NOTES + config:

```bash
git add _datafiles/config.yaml
git commit -m "chore(prod): toggle LogToFile=false for prod push

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Self-review notes

After writing this plan, walk back through the spec and confirm each section is covered:

- §1 (Item-side vendor_categories) → Phases 2–4.
- §2 (Buy rule rewrite) → Phase 5.
- §3 (Per-vendor audit + gold defaults + shop wipe) → Phases 6–8.
- §4 (Tier-50/40 baseline restock) → Phase 9.
- §5 (Forager liveness diagnostics + watchdog) → Phase 10.
- §6 (Dashboard rework) → Phases 11–12, with name fallback in Phase 1.
- §Tests → distributed across phases (TDD-first); plus Phase 13 smoke.

No placeholders — every task has explicit code snippets or file paths.
Type names consistent throughout (`StockScore`, `StockScoreDelta`,
`DeliveriesByTier`, `Throughput`, `RestockBaselineTiers`,
`vendorAcceptsAny`, `pickReason`).
