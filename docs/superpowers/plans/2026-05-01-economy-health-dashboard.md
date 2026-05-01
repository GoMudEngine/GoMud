# Economy Health Dashboard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship `/admin/economy/` web dashboard with hourly disk snapshots, health scores per shop / craft discipline / caravan / forager / overall, and delta columns at 1h/6h/1d/3d/1w. Answers the question "is the NPC economy supporting player crafting?" by rolling shop scores up by the discipline each shop's stock supports.

**Architecture:** New `internal/economy/health/` package owns snapshot capture, persistence, and scoring. A wall-clock ticker in `main.go` writes hourly snapshots to `_datafiles/economy/snapshots/{unix-ts}.yaml` (gitignored). Web layer (`internal/web/admin.economyhealth.go` + HTML template) renders tables and bars from JSON API. Mirrors existing Combat Stats / Progression dashboard pattern.

**Tech Stack:** Go (existing engine), YAML (gopkg.in/yaml.v2 — already in go.mod), Bootstrap 4 + vanilla JS (already loaded by `_header.html`), no new dependencies.

**Spec:** `docs/superpowers/specs/2026-05-01-economy-health-dashboard-design.md`

---

## File Structure

**New files:**
- `internal/economy/health/snapshot.go` — Snapshot struct + types
- `internal/economy/health/capture.go` — `CaptureSnapshot()` walks shops/caravans/foragers
- `internal/economy/health/persistence.go` — read/write/list/prune snapshots
- `internal/economy/health/scoring.go` — score formulas + insufficient-history handling
- `internal/economy/health/delta.go` — closest-snapshot picker, delta math
- `internal/economy/health/snapshot_test.go` / `capture_test.go` / `persistence_test.go` / `scoring_test.go` / `delta_test.go`
- `internal/web/admin.economyhealth.go` — HTTP handlers
- `_datafiles/html/admin/economy/index.html` — dashboard template
- `docs/economy/dashboard-runbook.md` — short note on how to read it

**Modified files:**
- `internal/shops/shopinventory.go` — add `CraftSupport string` field + valid-value constants
- `internal/shops/persistence.go` — add `AllShops()` accessor + RegisterShop auto-migration of empty CraftSupport from template
- `internal/shops/validation.go` (new) — `ValidateShopMobTags(mobs)` walks all shop-bearing mobs, panics with full list if any lack `craft_support:`
- `internal/mobs/mobs.go` — add `ShopCraftSupport string` field on the Mob struct
- `internal/mobs/crafter.go` — pass `mob.ShopCraftSupport` into the seeded template
- `main.go` — call `shops.ValidateShopMobTags()` after mob load
- `internal/caravan/visit.go` (or new `internal/caravan/wagon.go`) — expose `WagonMobId` const + `FindWagonInRoom(roomId int)`
- `internal/behaviortree/actions_caravan.go` — delegate `findWagonInRoom` to `caravan.FindWagonInRoom`
- `internal/configs/config.balance.go` — add 5 knobs
- `internal/configs/config.balance.misc.go` — defaults for the new knobs
- `_datafiles/config.yaml` — surface the new knobs
- `_datafiles/world/dogmud/mobs/**/*.yaml` (22 files) — add `craft_support:` to every mob with a shop or `crafter: true`. Full file list below in Task 2.
- `internal/web/web.go` — register 3 new routes
- `_datafiles/html/admin/_header.html` — add sidebar entry
- `main.go` — start hourly snapshot ticker
- `.gitignore` — exclude `_datafiles/economy/snapshots/`
- `PATCH_NOTES.md` — describe the dashboard

**Note on backfill:** The 7 files in `_datafiles/world/dogmud/shops/` are persisted *runtime instances* — written when a shop is first transacted on. The full set of shop NPCs is the union of mobs with `character.shop:` and mobs with `crafter: true` (22 mob YAMLs). The `craft_support:` tag lives on the mob YAML so new shops inherit it automatically when they materialize. Existing persisted runtime files are auto-migrated by RegisterShop (Task 1, Step 4): if `loaded.CraftSupport == ""`, stamp `template.CraftSupport` onto it. Startup validation panics if any shop-bearing mob is missing the tag.

**Tag taxonomy (`craft_support:`).** One value per shop, no lists:
- `blacksmithing`, `alchemy`, `tailoring`, `cooking`, `jewelcrafting`, `enchanting` — single-discipline shops
- `general` — multi-discipline / mixed merchants (avoids needing to maintain a list of every craft a general store contributes to)

---

## Conventions for every task

- **TDD where the code has logic:** test before implementation, run-fail before run-pass.
- **No TDD for pure-content edits:** YAML data, HTML template, sidebar link, gitignore — those are configuration, not logic.
- **Test runner:** `go test ./internal/economy/health/... -v -run <TestName>`. Whole-package: `go test ./internal/economy/health/...`.
- **Build check after each task:** `go build ./...` to confirm compile.
- **Commits:** conventional format (`feat:`, `refactor:`, `test:`, `chore:`). Co-authored trailer per CLAUDE.md.
- **Memory:** the test data path is per-test temp dir via `t.TempDir()`. Do not write into `_datafiles/economy/` from tests.

---

## Task 1: Plumb `CraftSupport` from mob template through to ShopInventory

The CraftSupport tag's source of truth is the mob YAML (`craft_support:` field). The mob loader copies it onto `mobs.Mob.ShopCraftSupport`. `crafter.go::registerCrafterShop` stamps it onto the seeded template. `RegisterShop` then auto-migrates any persisted runtime shop whose tag is empty (covers the 7 existing runtime files without manual edits). A startup validation pass (Task 1.5) panics if any shop-bearing mob lacks a valid tag — no silent fallback.

**Files:**
- Modify: `internal/shops/shopinventory.go` — add field + constants
- Modify: `internal/shops/persistence.go` — auto-migrate empty CraftSupport from template
- Modify: `internal/mobs/mobs.go` — add `ShopCraftSupport` field
- Modify: `internal/mobs/crafter.go` — copy mob.ShopCraftSupport into template
- Test: `internal/shops/persistence_test.go` (append) — auto-migration test

- [ ] **Step 1: Add CraftSupport constants + field on ShopInventory**

Edit `internal/shops/shopinventory.go`. Above `ShopInventory`, add:

```go
// CraftSupport tags a shop with the crafting discipline its stock
// supports. Single-discipline shops use the matching skill tag;
// multi-discipline / mixed merchants use "general". Empty is INVALID
// — a startup validator panics if any shop-bearing mob is missing
// the tag (see ValidateShopMobTags).
const (
	CraftSupportBlacksmithing = "blacksmithing"
	CraftSupportAlchemy       = "alchemy"
	CraftSupportTailoring     = "tailoring"
	CraftSupportCooking       = "cooking"
	CraftSupportJewelcrafting = "jewelcrafting"
	CraftSupportEnchanting    = "enchanting"
	CraftSupportGeneral       = "general"
)

// ValidCraftSupports is the canonical set. Mirrors the player crafting
// skills in internal/skills/skills.go plus "general" for mixed shops.
var ValidCraftSupports = []string{
	CraftSupportBlacksmithing,
	CraftSupportAlchemy,
	CraftSupportTailoring,
	CraftSupportCooking,
	CraftSupportJewelcrafting,
	CraftSupportEnchanting,
	CraftSupportGeneral,
}

// IsValidCraftSupport reports whether v is one of ValidCraftSupports.
func IsValidCraftSupport(v string) bool {
	for _, s := range ValidCraftSupports {
		if s == v {
			return true
		}
	}
	return false
}
```

In the `ShopInventory` struct, add a field after `KnownRecipes`:

```go
	CraftSupport string `yaml:"craft_support,omitempty"` // Discipline this shop's stock supports — see ValidCraftSupports
```

- [ ] **Step 2: Add ShopCraftSupport field on Mob**

Edit `internal/mobs/mobs.go`. Find the `Mob` struct (around line 72-130). Add a field next to other top-level YAML-loaded mob fields (near `Crafter` / `CrafterRestockMaterials` for visual grouping):

```go
	ShopCraftSupport string `yaml:"craft_support,omitempty"` // Crafting discipline this shop supports (one of shops.ValidCraftSupports)
```

- [ ] **Step 3: Stamp CraftSupport into the seeded template in crafter.go**

Edit `internal/mobs/crafter.go`. Find the `registerCrafterShop` function (around line 60-87). Locate the `template := shops.ShopInventory{...}` declaration and add the field:

```go
template := shops.ShopInventory{
    // ... existing fields ...
    CraftSupport: mob.ShopCraftSupport,
}
```

If `template` is built piecemeal (not in a single struct literal), just set `template.CraftSupport = mob.ShopCraftSupport` on a separate line before the `shops.RegisterShop(...)` call.

There may be more than one shop-registration call site. Search:

Run: `grep -rn "shops.RegisterShop" internal/`

For every callsite, ensure the template carries the tag. If there's a separate non-crafter shop registration, copy the same one-line stamp.

- [ ] **Step 4: Auto-migrate persisted shops with empty CraftSupport**

Edit `internal/shops/persistence.go`. Find `RegisterShop` (around line 70). The cache short-circuit at the top of RegisterShop is the migration's danger zone — ensure the migration runs even on cache hit:

```go
shopCacheMu.RLock()
inv, ok := shopCache[key]
shopCacheMu.RUnlock()
if ok {
    if inv.CraftSupport == "" && template.CraftSupport != "" {
        inv.CraftSupport = template.CraftSupport
        if err := saveToDisk(zone, mobId, roomId, inv); err != nil {
            mudlog.Warn("RegisterShop CraftSupport migration save", "key", key, "error", err)
        }
    }
    return inv
}
```

After the disk-load branch (when `inv` is freshly loaded from YAML or seeded from template), add the same check before storing into cache:

```go
if inv.CraftSupport == "" && template.CraftSupport != "" {
    inv.CraftSupport = template.CraftSupport
    if err := saveToDisk(zone, mobId, roomId, inv); err != nil {
        mudlog.Warn("RegisterShop CraftSupport migration save", "key", key, "error", err)
    }
}
```

If `saveToDisk` is not the existing helper name, find the actual one: `grep -n "func.*[Ss]ave\|writeShop\|persistShop" internal/shops/persistence.go`. Use whatever the existing function called by `SaveShop()` is named.

- [ ] **Step 5: Write the failing migration test**

Append to `internal/shops/persistence_test.go`:

```go
func TestRegisterShop_MigratesEmptyCraftSupport(t *testing.T) {
	ClearCache()
	t.Cleanup(ClearCache)

	// Seed a cached shop with empty CraftSupport.
	tmplNoTag := makeTemplate()
	tmplNoTag.CraftSupport = ""
	inv := RegisterShop("testzone", 9001, 1, tmplNoTag)
	if inv.CraftSupport != "" {
		t.Fatalf("setup precondition: got %q, want \"\"", inv.CraftSupport)
	}

	// Re-register with a tagged template — should migrate the cached one.
	tmplWithTag := makeTemplate()
	tmplWithTag.CraftSupport = CraftSupportBlacksmithing
	inv2 := RegisterShop("testzone", 9001, 1, tmplWithTag)
	if inv2.CraftSupport != CraftSupportBlacksmithing {
		t.Errorf("got %q after migration, want %q", inv2.CraftSupport, CraftSupportBlacksmithing)
	}
}

func TestIsValidCraftSupport(t *testing.T) {
	for _, v := range ValidCraftSupports {
		if !IsValidCraftSupport(v) {
			t.Errorf("ValidCraftSupports contains %q but IsValidCraftSupport returns false", v)
		}
	}
	if IsValidCraftSupport("nonsense") {
		t.Error("IsValidCraftSupport(\"nonsense\") = true, want false")
	}
	if IsValidCraftSupport("") {
		t.Error("IsValidCraftSupport(\"\") = true, want false (empty is INVALID)")
	}
}
```

`makeTemplate()` is the existing test helper in this file — reuse it.

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/shops/ -v -run "TestRegisterShop_MigratesEmptyCraftSupport|TestIsValidCraftSupport"`
Expected: PASS.

- [ ] **Step 7: Run all shop tests for regression**

Run: `go test ./internal/shops/...`
Expected: PASS.

- [ ] **Step 8: Build to confirm**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 9: Commit**

```bash
git add internal/shops/shopinventory.go internal/shops/persistence.go internal/shops/persistence_test.go internal/mobs/mobs.go internal/mobs/crafter.go
git commit -m "feat(shops): plumb CraftSupport from mob template to ShopInventory

CraftSupport tags a shop with the crafting discipline its stock
supports — one of {blacksmithing, alchemy, tailoring, cooking,
jewelcrafting, enchanting, general}. Source of truth is the mob YAML's
craft_support: field, copied into mobs.Mob.ShopCraftSupport, flowed
through crafter.go's registerCrafterShop into the seeded template,
and auto-migrated onto persisted runtime shops by RegisterShop.

Empty is invalid — startup validation in Task 1.5 panics if any
shop-bearing mob is missing the tag.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 1.5: Startup validation of `craft_support:` tags

A defensive panic at startup catches any shop-bearing mob that lacks `craft_support:` (or has an invalid value). Without this, a forgotten tag silently lands the shop in the dashboard's "general" bucket — exactly the kind of dataset drift the user wants to prevent.

**Files:**
- Create: `internal/shops/validation.go`
- Create: `internal/shops/validation_test.go`
- Modify: `main.go` — call validator after `mobs.LoadDataFiles()`

- [ ] **Step 1: Write the failing tests**

Create `internal/shops/validation_test.go`:

```go
package shops

import (
	"strings"
	"testing"
)

// fakeMob satisfies the minimal interface ValidateShopMobTags needs.
// Reuse production type Mob if it's package-importable; this is a
// stand-in if circular imports prevent that.
type fakeMob struct {
	mobId            int
	name             string
	zone             string
	hasShop          bool
	isCrafter        bool
	shopCraftSupport string
}

func (f fakeMob) MobId() int             { return f.mobId }
func (f fakeMob) Name() string           { return f.name }
func (f fakeMob) Zone() string           { return f.zone }
func (f fakeMob) HasShop() bool          { return f.hasShop }
func (f fakeMob) IsCrafter() bool        { return f.isCrafter }
func (f fakeMob) ShopCraftSupport() string { return f.shopCraftSupport }

func TestValidateShopMobTags_AllValid(t *testing.T) {
	mobs := []ShopBearingMob{
		fakeMob{mobId: 1, hasShop: true, shopCraftSupport: CraftSupportBlacksmithing},
		fakeMob{mobId: 2, isCrafter: true, shopCraftSupport: CraftSupportGeneral},
	}
	if err := ValidateShopMobTags(mobs); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateShopMobTags_MissingTag(t *testing.T) {
	mobs := []ShopBearingMob{
		fakeMob{mobId: 99, name: "broken", zone: "z", hasShop: true, shopCraftSupport: ""},
	}
	err := ValidateShopMobTags(mobs)
	if err == nil {
		t.Fatal("expected error for missing tag, got nil")
	}
	if !strings.Contains(err.Error(), "99") || !strings.Contains(err.Error(), "broken") {
		t.Errorf("error should reference the offending mob id and name; got: %v", err)
	}
}

func TestValidateShopMobTags_InvalidTag(t *testing.T) {
	mobs := []ShopBearingMob{
		fakeMob{mobId: 100, hasShop: true, shopCraftSupport: "knitting"},
	}
	err := ValidateShopMobTags(mobs)
	if err == nil || !strings.Contains(err.Error(), "knitting") {
		t.Errorf("expected error mentioning invalid tag; got: %v", err)
	}
}

func TestValidateShopMobTags_NonShopMobsIgnored(t *testing.T) {
	mobs := []ShopBearingMob{
		fakeMob{mobId: 5, hasShop: false, isCrafter: false, shopCraftSupport: ""},
	}
	if err := ValidateShopMobTags(mobs); err != nil {
		t.Fatalf("non-shop mobs should not be validated; got: %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/shops/ -v -run TestValidateShopMobTags`
Expected: FAIL — `ValidateShopMobTags` and `ShopBearingMob` undefined.

- [ ] **Step 3: Implement validator**

Create `internal/shops/validation.go`:

```go
package shops

import (
	"fmt"
	"sort"
	"strings"
)

// ShopBearingMob is the minimal interface ValidateShopMobTags needs.
// Production code wraps mobs.Mob in this interface (see main.go).
type ShopBearingMob interface {
	MobId() int
	Name() string
	Zone() string
	HasShop() bool
	IsCrafter() bool
	ShopCraftSupport() string
}

// ValidateShopMobTags walks all candidate mobs and returns a non-nil
// error if any shop-bearing mob (HasShop or IsCrafter) lacks a valid
// craft_support: tag. The error message lists every offending mob so
// a single restart surfaces the full set.
//
// Callers should panic on non-nil return to fail fast at startup.
func ValidateShopMobTags(mobs []ShopBearingMob) error {
	type fault struct {
		mobId int
		name  string
		zone  string
		tag   string
		why   string
	}
	var faults []fault

	for _, m := range mobs {
		if !m.HasShop() && !m.IsCrafter() {
			continue
		}
		tag := m.ShopCraftSupport()
		if tag == "" {
			faults = append(faults, fault{m.MobId(), m.Name(), m.Zone(), tag, "missing craft_support:"})
			continue
		}
		if !IsValidCraftSupport(tag) {
			faults = append(faults, fault{m.MobId(), m.Name(), m.Zone(), tag, "invalid value (not in ValidCraftSupports)"})
		}
	}

	if len(faults) == 0 {
		return nil
	}

	sort.Slice(faults, func(i, j int) bool { return faults[i].mobId < faults[j].mobId })
	var b strings.Builder
	fmt.Fprintf(&b, "shop-bearing mobs with bad craft_support tags (%d):\n", len(faults))
	for _, f := range faults {
		fmt.Fprintf(&b, "  - mob %d (%s, zone=%s): %s (got %q)\n", f.mobId, f.name, f.zone, f.why, f.tag)
	}
	b.WriteString("Valid values: ")
	b.WriteString(strings.Join(ValidCraftSupports, ", "))
	return fmt.Errorf("%s", b.String())
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/shops/ -v -run TestValidateShopMobTags`
Expected: PASS (4 subtests).

- [ ] **Step 5: Wire into main.go after mob load**

Edit `main.go`. Find `mobs.LoadDataFiles()` (search for it). Immediately after it returns successfully, add a validation call. Because `mobs.Mob` lives in another package, wrap it inline in a helper or implement the `ShopBearingMob` interface methods on `*mobs.Mob` directly.

Easiest path: add the methods on `*mobs.Mob` in `internal/mobs/mobs.go`:

```go
// Methods to satisfy shops.ShopBearingMob — used by startup validation.
func (m *Mob) HasShop() bool             { return len(m.Character.Shop) > 0 }
func (m *Mob) IsCrafter() bool           { return m.Crafter }
// MobId() is already a field-via-method or use direct field access; if
// the test interface requires a method, add a thin one:
//   func (m *Mob) GetMobId() int { return int(m.MobId) }
// and have the interface use GetMobId. Or rename interface methods to
// match the existing field-access pattern.
```

Check whether `mobs.Mob.Character.Shop` is the right field for "has a shop." If shops are detected differently (e.g. via `m.Character.Shop != nil` or another flag), use the equivalent. Open `internal/mobs/mobs.go` to confirm before writing.

The interface method names in `validation.go` may need to be adjusted to match what `*mobs.Mob` already exposes. Pick the path of least resistance — if the Mob struct uses field access for `MobId`, change the interface to `GetMobId() int` and add that method.

In `main.go`, after `mobs.LoadDataFiles()`:

```go
// Validate that every shop-bearing mob declares craft_support:
allMobs := mobs.AllMobTemplates() // or whatever returns the loaded slice
adapted := make([]shops.ShopBearingMob, 0, len(allMobs))
for _, m := range allMobs {
    adapted = append(adapted, m)
}
if err := shops.ValidateShopMobTags(adapted); err != nil {
    panic(fmt.Sprintf("shops.ValidateShopMobTags failed:\n%v", err))
}
```

If `mobs.AllMobTemplates()` doesn't exist, find the existing accessor (search: `grep -n "func.*\[\]\*Mob\|func.*\[\]Mob\b\|allMobs\|mobTemplates" internal/mobs/`). If no public accessor exists, add a tiny one:

```go
// AllMobTemplates returns every mob template loaded from disk. Used
// by startup validators (e.g. shops.ValidateShopMobTags).
func AllMobTemplates() []*Mob {
    // adapt to the existing template store (mobsByMobId, etc.)
}
```

- [ ] **Step 6: Boot the server, expect a clean startup once Task 2 is done**

Run: `go build ./...`
Expected: clean. Server boot will panic until Task 2 backfills the tags. That's the desired behavior — the panic message will list all 22 mobs that need the tag, which is exactly the worklist for Task 2.

- [ ] **Step 7: Commit**

```bash
git add internal/shops/validation.go internal/shops/validation_test.go internal/mobs/mobs.go main.go
git commit -m "feat(shops): startup validation of craft_support tags

Panics if any shop-bearing mob lacks a valid craft_support: tag.
The panic message lists every offender at once so a single restart
surfaces the full backfill worklist. Server will refuse to boot
until Task 2's mob YAML backfill is complete.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: Backfill `craft_support:` on the 22 mob YAMLs

The mob YAML is the source of truth. Crafter mobs have an explicit `crafterskill:` field that maps 1:1 to a craft_support value — those are deterministic. Non-crafter merchants need a judgment call from the engineer based on their stock list. After Task 1.5 ships, the server will refuse to boot until every shop-bearing mob has a valid tag — the panic message lists all 22 and serves as the worklist.

**Mapping (decided up front so this task is mechanical):**

| Mob ID | File | crafterskill | Decision | Reasoning |
|--------|------|--------------|----------|-----------|
| 52 | sanctum_basin/52-korvath.yaml | (read file) | (decide) | Inspect mob name + stock |
| 53 | sanctum_basin/53-alchemist_yenna.yaml | alchemy (likely) | `alchemy` | Apothecary by name |
| 63 | sanctum_basin/63-merchant_adela.yaml | (none) | `general` | "merchant" = mixed |
| 85 | watchers_crossing/85-merchant_brecca.yaml | (none) | `general` | "merchant" = mixed |
| 97 | thornwall_city/97-blacksmith_kerra.yaml | blacksmithing | `blacksmithing` | crafterskill maps |
| 98 | thornwall_city/98-apothecary_voss.yaml | alchemy | `alchemy` | crafterskill maps |
| 103 | thornwall_city/103-food_vendor.yaml | (none) | `cooking` | Sells food = supports cooking |
| 104 | thornwall_city/104-fence_dealer_siv.yaml | (none) | `general` | Fence = mixed loot |
| 108 | thornwall_city/108-jeweler_tess.yaml | jewelcrafting | `jewelcrafting` | crafterskill maps |
| 109 | thornwall_city/109-enchanter_vael.yaml | enchanting (likely) | `enchanting` | Inspect to confirm crafterskill |
| 113 | thornwall_city/113-weaver_maren.yaml | tailoring | `tailoring` | crafterskill maps |
| 248 | thornwall_city/248-tavern_cook_brynn.yaml | (none/cooking?) | `cooking` | Tavern cook |
| 273 | thornwall_city/273-whisper.yaml | (read file) | (decide) | Inspect mob name + stock |
| 278 | north_road/278-haral.yaml | (read file) | (decide) | Inspect mob name + stock |
| 333 | stillwater/333-innkeeper_sigrid.yaml | (none) | `cooking` | Innkeeper sells food |
| 336 | stillwater/336-fishmonger_tov_brann.yaml | (none) | `cooking` | Fish = cooking ingredient |
| 337 | stillwater/337-smith_brindle.yaml | blacksmithing | `blacksmithing` | crafterskill maps |
| 338 | stillwater/338-apothecary_ilsa.yaml | alchemy | `alchemy` | crafterskill maps |
| 339 | stillwater/339-weaver_edda.yaml | tailoring | `tailoring` | crafterskill maps |
| 340 | stillwater/340-pearl_carver_kess.yaml | jewelcrafting | `jewelcrafting` | crafterskill maps |
| 341 | stillwater/341-storekeeper_wulf.yaml | (none) | `general` | Sells base materials across multiple crafts |
| 348 | stillwater/348-miller_bram.yaml | (none) | `cooking` | Miller = grain/flour for cooking |

Five mobs marked "inspect" need a quick read by the engineer — open the YAML, look at `name:` and `character.shop:` items + any `crafterrestockmaterials:`, then pick from the same set. Default to `general` when in doubt.

- [ ] **Step 1: Read the five "inspect" mobs and finalize the table**

For each of {52, 109, 273, 278} confirm the decision. (109 should almost certainly be `enchanting` based on the name — verify there's a `crafterskill: enchanting` in the YAML.)

Capture the final table in your scratch notes — it goes into the commit message.

- [ ] **Step 2: Add `craft_support:` to each of the 22 mob YAMLs**

For each mob YAML, add `craft_support: <value>` as a top-level field. Place it just after `zone:` for visual grouping with other shop-related metadata.

Example for `_datafiles/world/dogmud/mobs/stillwater/337-smith_brindle.yaml`:

```yaml
mobid: 337
zone: Stillwater
craft_support: blacksmithing       # ← add this
behavior_archetype: noncombat_shopkeeper
statpool: 100
...
```

- [ ] **Step 3: Run mobs package tests**

Run: `go test ./internal/mobs/...`
Expected: PASS.

- [ ] **Step 4: Boot server — should now start cleanly**

Run: `go run main.go` and watch the startup logs for `mobs.LoadDataFiles() loadedCount=...` and the validator output. The Task 1.5 panic should NOT fire.
Expected: clean startup, no panics. `Ctrl-C` to stop.

If the validator still panics, the message lists the offending mob IDs — re-edit them and try again.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/mobs/
git commit -m "chore(mobs): backfill craft_support on 22 shop-bearing mobs

Tags every merchant or crafter mob with the crafting discipline its
stock supports. Crafter mobs derive from existing crafterskill: field
(blacksmithing, alchemy, tailoring, cooking, jewelcrafting,
enchanting); non-crafters use 'general' for mixed-stock or the most
relevant single discipline (e.g. innkeeper -> cooking, food vendor ->
cooking, fishmonger -> cooking).

Persisted runtime shops in _datafiles/world/dogmud/shops/ auto-migrate
from these on next boot (RegisterShop migration in earlier commit).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: Promote `findWagonInRoom` to `internal/caravan` package

The snapshot system needs to find each caravan's wagon (mob 374) to read cargo. The lookup currently lives as a private helper inside `internal/behaviortree`. Move it so both packages can use it.

**Files:**
- Create: `internal/caravan/wagon.go`
- Modify: `internal/behaviortree/actions_caravan.go` (delete local helper, call new exported one)
- Test: `internal/caravan/wagon_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/caravan/wagon_test.go`:

```go
package caravan_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/caravan"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func TestFindWagonInRoom_ReturnsWagon(t *testing.T) {
	// Set up a room and place mob 374 in it.
	r := &rooms.Room{RoomId: 9999}
	rooms.SetRoom(r)
	t.Cleanup(func() { rooms.RemoveRoom(9999) })

	wagon, _ := mobs.NewMobByIdFresh(caravan.WagonMobId, 9999)
	if wagon == nil {
		t.Fatalf("test fixture: wagon mobId %d failed to instantiate", caravan.WagonMobId)
	}
	r.AddMob(wagon.InstanceId)
	t.Cleanup(func() { mobs.RemoveMobInstance(wagon.InstanceId) })

	got := caravan.FindWagonInRoom(9999)
	if got == nil {
		t.Fatalf("FindWagonInRoom(9999): got nil, want wagon mob %d", caravan.WagonMobId)
	}
	if int(got.MobId) != caravan.WagonMobId {
		t.Errorf("FindWagonInRoom(9999): got mobId %d, want %d", got.MobId, caravan.WagonMobId)
	}
}

func TestFindWagonInRoom_AbsentReturnsNil(t *testing.T) {
	r := &rooms.Room{RoomId: 9998}
	rooms.SetRoom(r)
	t.Cleanup(func() { rooms.RemoveRoom(9998) })

	if got := caravan.FindWagonInRoom(9998); got != nil {
		t.Errorf("FindWagonInRoom(9998) on empty room: got %v, want nil", got)
	}
}

func TestFindWagonInRoom_UnknownRoomReturnsNil(t *testing.T) {
	if got := caravan.FindWagonInRoom(424242); got != nil {
		t.Errorf("FindWagonInRoom(unknown room): got %v, want nil", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/caravan/... -v -run TestFindWagonInRoom`
Expected: FAIL — `caravan.WagonMobId` and `caravan.FindWagonInRoom` don't exist yet.

- [ ] **Step 3: Implement the helper**

Create `internal/caravan/wagon.go`:

```go
package caravan

import (
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// WagonMobId is the mob template ID of the caravan wagon — the
// cargo-bearing follower that travels with each caravan leader.
// Its inventory is the source of truth for caravan cargo.
const WagonMobId = 374

// FindWagonInRoom returns the wagon mob (WagonMobId) co-located in
// the given room, or nil if the wagon is not present (mid-respawn,
// followers lagging, or wagon wiped).
//
// Callers decide whether nil is fatal or recoverable.
func FindWagonInRoom(roomId int) *mobs.Mob {
	room := rooms.LoadRoom(roomId)
	if room == nil {
		return nil
	}
	for _, instId := range room.GetMobs(rooms.FindAll) {
		m := mobs.GetInstance(instId)
		if m == nil {
			continue
		}
		if int(m.MobId) == WagonMobId {
			return m
		}
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/caravan/... -v -run TestFindWagonInRoom`
Expected: PASS (3 subtests).

- [ ] **Step 5: Update behaviortree to delegate**

Edit `internal/behaviortree/actions_caravan.go`. Delete the `findWagonInRoom` function (lines ~445-461 — the body shown in plan-prep was 17 lines). Replace any callsites in this file:

```go
// Before:
wagon := findWagonInRoom(nextRoom)

// After:
wagon := caravan.FindWagonInRoom(nextRoom)
```

The `caravan` import is already present in this file.

- [ ] **Step 6: Run all caravan-touching tests**

Run: `go test ./internal/caravan/... ./internal/behaviortree/...`
Expected: PASS.

- [ ] **Step 7: Build to confirm clean compile**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 8: Commit**

```bash
git add internal/caravan/wagon.go internal/caravan/wagon_test.go internal/behaviortree/actions_caravan.go
git commit -m "refactor(caravan): expose WagonMobId + FindWagonInRoom

Promotes findWagonInRoom from behaviortree-private to caravan-public
so the upcoming economy/health snapshot system can reuse it. Logic
unchanged.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 4: Define snapshot types in new health package

**Files:**
- Create: `internal/economy/health/snapshot.go`
- Test: `internal/economy/health/snapshot_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/economy/health/snapshot_test.go`:

```go
package health

import (
	"testing"

	"gopkg.in/yaml.v2"
)

func TestSnapshot_YAMLRoundTrip(t *testing.T) {
	in := Snapshot{
		Timestamp: "2026-05-01T13:00:00Z",
		UnixTs:    1746104400,
		Round:     12345,
		Manual:    true,
		ManualLabel: "pre stage-3.4",
		Shops: []ShopSnapshot{
			{
				Zone: "stillwater", MobId: 341, RoomId: 4105,
				Name: "Storekeeper Wulf", CraftSupport: "general",
				Gold: 487, StartingGold: 500, LastRestockRound: 12000,
				Stock: []StockSnapshot{
					{ItemId: 40001, Bucket: "base", Current: 8, Max: 20, RestockQty: 5},
				},
			},
		},
		Caravans: []CaravanSnapshot{
			{
				InstId: 42, Name: "Caravan Master Borric",
				State: "outbound_transit", StateEnteredRound: 12100, RoomId: 1500,
				CargoWeight: 850, CargoCapacity: 5000,
				CargoByBucket: map[string]int{"base": 300, "stillwater": 550},
			},
		},
		Foragers: []ForagerSnapshot{
			{
				InstId: 88, Name: "Storekeeper Wulf",
				Territory: "stillwater_marsh",
				State: "foraging", StateEnteredRound: 12200, RoomId: 4520,
				CargoWeight: 14, CargoCapacity: 60,
				CargoByBucket: map[string]int{"stillwater": 14},
			},
		},
	}

	bytes, err := yaml.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var out Snapshot
	if err := yaml.Unmarshal(bytes, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if out.UnixTs != in.UnixTs {
		t.Errorf("UnixTs: got %d, want %d", out.UnixTs, in.UnixTs)
	}
	if len(out.Shops) != 1 || out.Shops[0].MobId != 341 {
		t.Errorf("Shops round-trip mismatch: got %+v", out.Shops)
	}
	if len(out.Caravans) != 1 || out.Caravans[0].State != "outbound_transit" {
		t.Errorf("Caravans round-trip mismatch: got %+v", out.Caravans)
	}
	if got := out.Foragers[0].CargoByBucket["stillwater"]; got != 6 {
		t.Errorf("Foragers cargo bucket: got %d, want 6", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/economy/health/ -v -run TestSnapshot_YAMLRoundTrip`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Implement the types**

Create `internal/economy/health/snapshot.go`:

```go
// Package health captures and scores point-in-time snapshots of the
// NPC economy (shops, caravans, foragers). Snapshots persist as YAML
// and feed the /admin/economy/ dashboard.
package health

// Snapshot is the single payload written hourly to disk. Both yaml
// (disk) and json (web API) tags are present — the dashboard JS
// expects lowercase JSON field names.
type Snapshot struct {
	Timestamp   string `yaml:"timestamp"                 json:"timestamp"`
	UnixTs      int64  `yaml:"unix_ts"                   json:"unix_ts"`
	Round       uint64 `yaml:"round"                     json:"round"`
	Manual      bool   `yaml:"manual"                    json:"manual"`
	ManualLabel string `yaml:"manual_label,omitempty"    json:"manual_label,omitempty"`

	Shops    []ShopSnapshot    `yaml:"shops"    json:"shops"`
	Caravans []CaravanSnapshot `yaml:"caravans" json:"caravans"`
	Foragers []ForagerSnapshot `yaml:"foragers" json:"foragers"`
}

// ShopSnapshot captures one merchant's economic state.
type ShopSnapshot struct {
	Zone             string          `yaml:"zone"               json:"zone"`
	MobId            int             `yaml:"mob_id"             json:"mob_id"`
	RoomId           int             `yaml:"room_id"            json:"room_id"`
	Name             string          `yaml:"name"               json:"name"`
	CraftSupport     string          `yaml:"craft_support"      json:"craft_support"`
	Gold             int             `yaml:"gold"               json:"gold"`
	StartingGold     int             `yaml:"starting_gold"      json:"starting_gold"`
	LastRestockRound uint64          `yaml:"last_restock_round" json:"last_restock_round"`
	Stock            []StockSnapshot `yaml:"stock"              json:"stock"`
}

// StockSnapshot is a per-item entry. Bucket comes from economy.BucketFor().
type StockSnapshot struct {
	ItemId     int    `yaml:"item_id"     json:"item_id"`
	Bucket     string `yaml:"bucket"      json:"bucket"`
	Current    int    `yaml:"current"     json:"current"`
	Max        int    `yaml:"max"         json:"max"`
	RestockQty int    `yaml:"restock_qty" json:"restock_qty"`
}

// CaravanSnapshot captures one caravan-leader instance + its co-located wagon's cargo.
type CaravanSnapshot struct {
	InstId            int            `yaml:"inst_id"             json:"inst_id"`
	Name              string         `yaml:"name"                json:"name"`
	State             string         `yaml:"state"               json:"state"`
	StateEnteredRound uint64         `yaml:"state_entered_round" json:"state_entered_round"`
	RoomId            int            `yaml:"room_id"             json:"room_id"`
	CargoWeight       int            `yaml:"cargo_weight"        json:"cargo_weight"`   // pounds
	CargoCapacity     int            `yaml:"cargo_capacity"      json:"cargo_capacity"` // pounds
	CargoByBucket     map[string]int `yaml:"cargo_by_bucket"     json:"cargo_by_bucket"`
}

// ForagerSnapshot captures one forager NPC's state + backpack composition.
type ForagerSnapshot struct {
	InstId            int            `yaml:"inst_id"             json:"inst_id"`
	Name              string         `yaml:"name"                json:"name"`
	Territory         string         `yaml:"territory"           json:"territory"`
	State             string         `yaml:"state"               json:"state"`
	StateEnteredRound uint64         `yaml:"state_entered_round" json:"state_entered_round"`
	RoomId            int            `yaml:"room_id"             json:"room_id"`
	CargoWeight       int            `yaml:"cargo_weight"        json:"cargo_weight"`   // pounds
	CargoCapacity     int            `yaml:"cargo_capacity"      json:"cargo_capacity"` // pounds
	CargoByBucket     map[string]int `yaml:"cargo_by_bucket"     json:"cargo_by_bucket"`
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/economy/health/ -v -run TestSnapshot_YAMLRoundTrip`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/economy/health/snapshot.go internal/economy/health/snapshot_test.go
git commit -m "feat(economy/health): define Snapshot types

Snapshot is the YAML-serializable shape captured hourly by the
upcoming dashboard ticker. Round-trip test pins the schema.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 5: Implement `CaptureSnapshot()` for shops

Walk `shops.shopCache` and produce `[]ShopSnapshot`. The shop cache is package-private; we add `shops.AllShops()` to expose it.

**Files:**
- Modify: `internal/shops/persistence.go` (add `AllShops()` accessor)
- Create: `internal/economy/health/capture.go`
- Test: `internal/economy/health/capture_test.go`

- [ ] **Step 1: Add `AllShops()` to shops package**

Edit `internal/shops/persistence.go`. Append after the existing exported funcs:

```go
// AllShops returns a snapshot of every registered ShopInventory in
// the cache. The returned slice contains pointers to the cached
// inventories — callers must not mutate them. Used by the
// economy/health dashboard for hourly capture.
func AllShops() []*ShopInventory {
	shopCacheMu.RLock()
	defer shopCacheMu.RUnlock()
	out := make([]*ShopInventory, 0, len(shopCache))
	for _, inv := range shopCache {
		out = append(out, inv)
	}
	return out
}
```

- [ ] **Step 2: Build to verify**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 3: Write the failing test for shop capture**

Create `internal/economy/health/capture_test.go`:

```go
package health_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/economy/health"
	"github.com/GoMudEngine/GoMud/internal/shops"
)

func TestCaptureSnapshot_Shops(t *testing.T) {
	shops.ClearCache()
	t.Cleanup(shops.ClearCache)

	tmpl := shops.ShopInventory{
		Gold:         500,
		StartingGold: 500,
		CraftSupport: shops.CraftSupportGeneral,
		Stock: []shops.StockEntry{
			{ItemId: 40001, RestockQty: 5, MaxStock: 20, Current: 8}, // base
			{ItemId: 40051, RestockQty: 3, MaxStock: 10, Current: 4}, // stillwater
		},
	}
	shops.RegisterShop("stillwater", 341, 4105, tmpl)

	snap := health.CaptureSnapshot()

	if len(snap.Shops) != 1 {
		t.Fatalf("Shops: got %d, want 1", len(snap.Shops))
	}
	got := snap.Shops[0]
	if got.MobId != 341 || got.RoomId != 4105 {
		t.Errorf("location: got %d/%d, want 341/4105", got.MobId, got.RoomId)
	}
	if got.CraftSupport != "general" {
		t.Errorf("craft_support: got %q, want general", got.CraftSupport)
	}
	if len(got.Stock) != 2 {
		t.Fatalf("stock entries: got %d, want 2", len(got.Stock))
	}
	if got.Stock[0].Bucket != "base" {
		t.Errorf("first stock bucket: got %q, want base", got.Stock[0].Bucket)
	}
}
```

- [ ] **Step 4: Run test to verify it fails**

Run: `go test ./internal/economy/health/ -v -run TestCaptureSnapshot_Shops`
Expected: FAIL — `health.CaptureSnapshot` not defined.

- [ ] **Step 5: Implement shop capture**

Create `internal/economy/health/capture.go`:

```go
package health

import (
	"time"

	"github.com/GoMudEngine/GoMud/internal/economy"
	"github.com/GoMudEngine/GoMud/internal/shops"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// CaptureSnapshot walks every live shop, caravan leader, and forager
// and produces a Snapshot suitable for serialization or scoring.
//
// Caravans and foragers are populated by separate helpers (see
// captureCaravans, captureForagers in this file).
func CaptureSnapshot() Snapshot {
	now := time.Now().UTC()
	snap := Snapshot{
		Timestamp: now.Format(time.RFC3339),
		UnixTs:    now.Unix(),
		Round:     util.GetRoundCount(),
	}
	snap.Shops = captureShops()
	snap.Caravans = captureCaravans()
	snap.Foragers = captureForagers()
	return snap
}

func captureShops() []ShopSnapshot {
	all := shops.AllShops()
	out := make([]ShopSnapshot, 0, len(all))
	for _, inv := range all {
		ss := ShopSnapshot{
			Zone:             inv.Zone,
			MobId:            inv.MobId,
			RoomId:           inv.RoomId,
			CraftSupport:     inv.CraftSupport,
			Gold:             inv.Gold,
			StartingGold:     inv.StartingGold,
			LastRestockRound: inv.LastRestock,
			Stock:            make([]StockSnapshot, 0, len(inv.Stock)),
		}
		ss.Name = lookupMobName(inv.MobId, inv.RoomId) // shop is bound to a specific mob instance
		for _, e := range inv.Stock {
			ss.Stock = append(ss.Stock, StockSnapshot{
				ItemId:     e.ItemId,
				Bucket:     economy.BucketFor(e.ItemId),
				Current:    e.Current,
				Max:        e.MaxStock,
				RestockQty: e.RestockQty,
			})
		}
		out = append(out, ss)
	}
	return out
}

// captureCaravans is implemented in Task 6.
func captureCaravans() []CaravanSnapshot { return nil }

// captureForagers is implemented in Task 7.
func captureForagers() []ForagerSnapshot { return nil }
```

Add a helper at the bottom of `capture.go`:

```go
// lookupMobName resolves a shop's display name by walking live mob
// instances for one matching mobId+roomId. Returns "" if the mob is
// not currently spawned (shouldn't happen for registered shops, but
// gracefully degrades).
func lookupMobName(mobId, roomId int) string {
	// Direct lookup avoids importing mobs at package level and keeps
	// capture testable.
	for _, instId := range mobsAllInstanceIds() {
		m := mobsGetInstance(instId)
		if m == nil {
			continue
		}
		if int(m.MobId) == mobId && m.HomeRoomId == roomId {
			return m.Character.Name
		}
	}
	return ""
}
```

We expose tiny indirections so tests can stub mob iteration. Add at bottom of `capture.go`:

```go
// mobs-package indirections for testability. Production wires these
// to the real mobs package in init.go.
var (
	mobsAllInstanceIds = func() []int { return nil }
	mobsGetInstance    = func(int) *mobInstance { return nil }
)

type mobInstance interface {
	// no methods required at this layer; capture.go reads concrete fields
}
```

This indirection looks awkward — drop it and import mobs directly. Simpler approach below.

- [ ] **Step 6: Replace the indirection with a direct mobs import**

Replace the bottom of `capture.go` with a direct import. The whole file should now import `internal/mobs`. Final `capture.go`:

```go
package health

import (
	"time"

	"github.com/GoMudEngine/GoMud/internal/economy"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/shops"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func CaptureSnapshot() Snapshot {
	now := time.Now().UTC()
	snap := Snapshot{
		Timestamp: now.Format(time.RFC3339),
		UnixTs:    now.Unix(),
		Round:     util.GetRoundCount(),
	}
	snap.Shops = captureShops()
	snap.Caravans = captureCaravans()
	snap.Foragers = captureForagers()
	return snap
}

func captureShops() []ShopSnapshot {
	all := shops.AllShops()
	out := make([]ShopSnapshot, 0, len(all))
	for _, inv := range all {
		ss := ShopSnapshot{
			Zone:             inv.Zone,
			MobId:            inv.MobId,
			RoomId:           inv.RoomId,
			CraftSupport:     inv.CraftSupport,
			Gold:             inv.Gold,
			StartingGold:     inv.StartingGold,
			LastRestockRound: inv.LastRestock,
			Stock:            make([]StockSnapshot, 0, len(inv.Stock)),
			Name:             lookupShopMobName(inv.MobId, inv.RoomId),
		}
		for _, e := range inv.Stock {
			ss.Stock = append(ss.Stock, StockSnapshot{
				ItemId:     e.ItemId,
				Bucket:     economy.BucketFor(e.ItemId),
				Current:    e.Current,
				Max:        e.MaxStock,
				RestockQty: e.RestockQty,
			})
		}
		out = append(out, ss)
	}
	return out
}

func captureCaravans() []CaravanSnapshot { return nil } // Task 6
func captureForagers() []ForagerSnapshot { return nil } // Task 7

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
	return ""
}
```

- [ ] **Step 7: Run test to verify it passes**

Run: `go test ./internal/economy/health/ -v -run TestCaptureSnapshot_Shops`
Expected: PASS. (Name will be "" because no mob is spawned in the test — the assertion only checks the fields that don't depend on a live mob instance.)

- [ ] **Step 8: Build all packages**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 9: Commit**

```bash
git add internal/shops/persistence.go internal/economy/health/capture.go internal/economy/health/capture_test.go
git commit -m "feat(economy/health): capture shops into snapshot

Adds shops.AllShops() accessor and the captureShops() helper. Caravan
and forager capture follow in subsequent tasks (currently return nil).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 6: Implement caravan capture

Walk every mob; identify caravan leaders by `BTreeState.GetString("caravan_state") != ""`; for each leader, find the wagon in the same room (via `caravan.FindWagonInRoom`) and read its cargo.

**Files:**
- Modify: `internal/economy/health/capture.go`
- Test: `internal/economy/health/capture_test.go` (append)

- [ ] **Step 1: Write the failing test**

Append to `internal/economy/health/capture_test.go`:

```go
import (
	// add to imports if not present:
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/caravan"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func TestCaptureSnapshot_Caravans(t *testing.T) {
	r := &rooms.Room{RoomId: 9000}
	rooms.SetRoom(r)
	t.Cleanup(func() { rooms.RemoveRoom(9000) })

	// Spawn the wagon and stuff one base-bucket item into its cargo.
	wagon, _ := mobs.NewMobByIdFresh(caravan.WagonMobId, 9000)
	if wagon == nil {
		t.Fatal("wagon failed to instantiate")
	}
	r.AddMob(wagon.InstanceId)
	wagon.Character.StoreItem(items.New(40001)) // iron ingot, "base" bucket
	t.Cleanup(func() { mobs.RemoveMobInstance(wagon.InstanceId) })

	// Spawn the real caravan leader (mob 357 — Ketil) and stamp
	// caravan_state on its BehaviorState. Using a distinct mobId from
	// the wagon avoids ambiguity in FindWagonInRoom.
	const caravanLeaderMobId = 357
	leader, _ := mobs.NewMobByIdFresh(caravanLeaderMobId, 9000)
	if leader == nil {
		t.Fatalf("leader fixture failed — mob %d missing from fixtures", caravanLeaderMobId)
	}
	bs := &behaviortree.BehaviorState{}
	bs.Set("caravan_state", "outbound_transit")
	bs.Set("caravan_state_started_round", strconv.FormatUint(12100, 10))
	leader.BTreeState = bs
	r.AddMob(leader.InstanceId)
	t.Cleanup(func() { mobs.RemoveMobInstance(leader.InstanceId) })

	snap := health.CaptureSnapshot()

	if len(snap.Caravans) != 1 {
		t.Fatalf("Caravans: got %d, want 1", len(snap.Caravans))
	}
	c := snap.Caravans[0]
	if c.State != "outbound_transit" {
		t.Errorf("state: got %q, want outbound_transit", c.State)
	}
	if c.StateEnteredRound != 12100 {
		t.Errorf("state_entered_round: got %d, want 12100", c.StateEnteredRound)
	}
	if c.CargoCapacity != 5000 {
		t.Errorf("cargo_capacity: got %d, want 5000 (override set in fixture)", c.CargoCapacity)
	}
	if c.CargoByBucket == nil {
		t.Error("cargo_by_bucket: got nil map, want initialized map")
	}
}
```

(Mob 357 is Ketil — the real caravan leader for the Thornwall-Stillwater route. Mob 374 is the wagon. They're distinct mobIds in production and `FindWagonInRoom` correctly returns mob 374 only.)

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/economy/health/ -v -run TestCaptureSnapshot_Caravans`
Expected: FAIL — `captureCaravans` returns nil.

- [ ] **Step 3: Implement `captureCaravans`**

Replace the stub in `internal/economy/health/capture.go`:

```go
import (
	// add:
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/caravan"
)

func captureCaravans() []CaravanSnapshot {
	out := []CaravanSnapshot{}
	for _, instId := range mobs.GetAllMobInstanceIds() {
		m := mobs.GetInstance(instId)
		if m == nil {
			continue
		}
		bs, ok := m.BTreeState.(*behaviortree.BehaviorState)
		if !ok || bs == nil {
			continue
		}
		stateName := bs.GetString("caravan_state")
		if stateName == "" {
			continue
		}

		startedRound, _ := strconv.ParseUint(bs.GetString("caravan_state_started_round"), 10, 64)

		cs := CaravanSnapshot{
			InstId:            instId,
			Name:              m.Character.Name,
			State:             stateName,
			StateEnteredRound: startedRound,
			RoomId:            m.Character.RoomId,
			CargoByBucket:     map[string]int{},
		}

		// Wagon co-located with leader is the cargo source. Both
		// CargoWeight and CargoCapacity are pounds — carry weight is
		// what actually limits the wagon, so the dashboard's "is the
		// wagon filling up?" question reads honestly as a weight
		// ratio. Per-bucket also sums weights, not item counts.
		wagon := caravan.FindWagonInRoom(m.Character.RoomId)
		if wagon != nil {
			cs.CargoWeight = int(wagon.Character.GetCarriedWeight())
			cs.CargoCapacity = int(wagon.Character.CarryCapacity())
			for _, it := range wagon.Character.Items {
				bucket := economy.BucketFor(it.ItemId)
				if bucket == "" {
					continue
				}
				w := int(it.GetSpec().GetWeight())
				if w > 0 {
					cs.CargoByBucket[bucket] += w
				}
			}
		}

		out = append(out, cs)
	}
	return out
}
```

If `Character.GetCarryCapacity()` is named differently (e.g. `CarryCapacity()`), discover the correct name from `internal/characters/character.go` before writing — read the file and adapt.

- [ ] **Step 4: Confirm the carry-capacity method name**

Run: `grep -rn "func.*CarryCapacity" internal/characters/ | head -5`
Pick the public method that returns the strength-derived capacity. Substitute it into the line above.

- [ ] **Step 5: Run the test to verify it passes**

Run: `go test ./internal/economy/health/ -v -run TestCaptureSnapshot_Caravans`
Expected: PASS.

- [ ] **Step 6: Run full health package tests**

Run: `go test ./internal/economy/health/...`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/economy/health/capture.go internal/economy/health/capture_test.go
git commit -m "feat(economy/health): capture caravan leaders + wagon cargo

Identifies leaders by BTreeState.GetString(\"caravan_state\") != \"\"
(same convention as the existing caravan reset admin command).
Cargo is read from the wagon mob co-located with the leader.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 7: Implement forager capture

Same shape as Task 6. Foragers carry their own cargo (no separate wagon mob); the cargo source is `forager.Character.Items` directly.

**Files:**
- Modify: `internal/economy/health/capture.go`
- Test: `internal/economy/health/capture_test.go` (append)

- [ ] **Step 1: Write the failing test**

Append to `internal/economy/health/capture_test.go`:

```go
func TestCaptureSnapshot_Foragers(t *testing.T) {
	r := &rooms.Room{RoomId: 9100}
	rooms.SetRoom(r)
	t.Cleanup(func() { rooms.RemoveRoom(9100) })

	// Marsh forager (Tova, mob 371). The forager package's
	// ProfileFor(371) returns KindMarsh, which captureForagers
	// translates to "stillwater_marsh" in the Territory field.
	const marshForagerMobId = 371
	forager, _ := mobs.NewMobByIdFresh(marshForagerMobId, 9100)
	if forager == nil {
		t.Fatalf("forager fixture failed — mob %d missing", marshForagerMobId)
	}
	bs := &behaviortree.BehaviorState{}
	bs.Set("forager_state", "foraging")
	bs.Set("forager_state_started_round", strconv.FormatUint(12200, 10))
	forager.BTreeState = bs
	forager.Character.StoreItem(items.New(40051)) // skitter-shrimp shell, "stillwater"
	r.AddMob(forager.InstanceId)
	t.Cleanup(func() { mobs.RemoveMobInstance(forager.InstanceId) })

	snap := health.CaptureSnapshot()

	if len(snap.Foragers) != 1 {
		t.Fatalf("Foragers: got %d, want 1", len(snap.Foragers))
	}
	f := snap.Foragers[0]
	if f.State != "foraging" {
		t.Errorf("state: got %q, want foraging", f.State)
	}
	if f.Territory != "stillwater_marsh" {
		t.Errorf("territory: got %q, want stillwater_marsh", f.Territory)
	}
	if f.CargoByBucket["stillwater"] != 1 {
		t.Errorf("cargo_by_bucket[stillwater]: got %d, want 1", f.CargoByBucket["stillwater"])
	}
}
```

(Territory is derived from `forager.ProfileFor(mobId).Kind`, not a BTreeState key — `internal/forager/territory.go` defines `KindMarsh`/`KindSteppe`/`KindFernway` per mobId. The capture function translates Kind to a stable string.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/economy/health/ -v -run TestCaptureSnapshot_Foragers`
Expected: FAIL — `captureForagers` returns nil.

- [ ] **Step 3: Implement `captureForagers`**

Replace the stub in `capture.go`:

```go
func captureForagers() []ForagerSnapshot {
	out := []ForagerSnapshot{}
	for _, instId := range mobs.GetAllMobInstanceIds() {
		m := mobs.GetInstance(instId)
		if m == nil {
			continue
		}
		bs, ok := m.BTreeState.(*behaviortree.BehaviorState)
		if !ok || bs == nil {
			continue
		}
		stateName := bs.GetString("forager_state")
		if stateName == "" {
			continue
		}

		startedRound, _ := strconv.ParseUint(bs.GetString("forager_state_started_round"), 10, 64)

		fs := ForagerSnapshot{
			InstId:            instId,
			Name:              m.Character.Name,
			Territory:         territoryFor(int(m.MobId)),
			State:             stateName,
			StateEnteredRound: startedRound,
			RoomId:            m.Character.RoomId,
			CargoByBucket:     map[string]int{},
			CargoWeight:       int(m.Character.GetCarriedWeight()),
			CargoCapacity:     int(m.Character.CarryCapacity()),
		}
		// Per-bucket: sum item weights by bucket. Skip items with no
		// bucket or zero weight. Same convention as captureCaravans.
		for _, it := range m.Character.Items {
			bucket := economy.BucketFor(it.ItemId)
			if bucket == "" {
				continue
			}
			w := int(it.GetSpec().GetWeight())
			if w > 0 {
				fs.CargoByBucket[bucket] += w
			}
		}
		out = append(out, fs)
	}
	return out
}

// territoryFor returns the stable string label for a forager's territory,
// derived from forager.ProfileFor(mobId).Kind. Returns "" for non-foragers
// or unrecognized profiles.
func territoryFor(mobId int) string {
	p := forager.ProfileFor(mobId)
	if p == nil {
		return ""
	}
	switch p.Kind {
	case forager.KindMarsh:
		return "stillwater_marsh"
	case forager.KindSteppe:
		return "thornwall_steppe"
	case forager.KindFernway:
		return "fernway"
	}
	return ""
}
```

Add `"github.com/GoMudEngine/GoMud/internal/forager"` to the imports at the top of `capture.go`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/economy/health/ -v -run TestCaptureSnapshot_Foragers`
Expected: PASS.

- [ ] **Step 5: Run full package**

Run: `go test ./internal/economy/health/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/economy/health/capture.go internal/economy/health/capture_test.go
git commit -m "feat(economy/health): capture foragers and their cargo

Identifies foragers by BTreeState.GetString(\"forager_state\") != \"\".
Cargo is read from the forager's own inventory (no separate wagon).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 8: Snapshot persistence (write/load/list/prune)

**Files:**
- Create: `internal/economy/health/persistence.go`
- Test: `internal/economy/health/persistence_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/economy/health/persistence_test.go`:

```go
package health_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/economy/health"
)

func TestPersistence_WriteThenLoad(t *testing.T) {
	dir := t.TempDir()
	in := health.Snapshot{UnixTs: 1746104400, Round: 12345, Manual: true, ManualLabel: "x"}

	if err := health.WriteSnapshotTo(dir, in); err != nil {
		t.Fatalf("WriteSnapshotTo: %v", err)
	}

	out, err := health.LoadSnapshotFrom(dir, in.UnixTs)
	if err != nil {
		t.Fatalf("LoadSnapshotFrom: %v", err)
	}
	if out.UnixTs != in.UnixTs || out.ManualLabel != "x" {
		t.Errorf("round-trip mismatch: %+v vs %+v", in, out)
	}
}

func TestPersistence_List(t *testing.T) {
	dir := t.TempDir()
	for _, ts := range []int64{1000, 2000, 3000} {
		health.WriteSnapshotTo(dir, health.Snapshot{UnixTs: ts})
	}
	metas := health.ListSnapshotsFrom(dir)
	if len(metas) != 3 {
		t.Fatalf("got %d, want 3 metas", len(metas))
	}
	if metas[0].UnixTs != 3000 || metas[2].UnixTs != 1000 {
		t.Errorf("not sorted desc: %v", metas)
	}
}

func TestPersistence_Prune_KeepsManual(t *testing.T) {
	dir := t.TempDir()
	old := time.Now().Add(-100 * 24 * time.Hour).Unix()
	recent := time.Now().Unix()

	health.WriteSnapshotTo(dir, health.Snapshot{UnixTs: old, Manual: false})
	health.WriteSnapshotTo(dir, health.Snapshot{UnixTs: old + 1, Manual: true})
	health.WriteSnapshotTo(dir, health.Snapshot{UnixTs: recent, Manual: false})

	pruned, err := health.PruneSnapshotsIn(dir, 30) // 30-day retention
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if pruned != 1 {
		t.Errorf("pruned: got %d, want 1 (only the old non-manual)", pruned)
	}

	left, _ := os.ReadDir(dir)
	if len(left) != 2 {
		t.Errorf("files left: got %d, want 2 (manual + recent)", len(left))
	}
	// Confirm manual file survived.
	if _, err := os.Stat(filepath.Join(dir, snapshotFilename(old+1))); err != nil {
		t.Errorf("manual snapshot was pruned: %v", err)
	}
}

// snapshotFilename — match the persistence helper's naming. Test-only mirror.
func snapshotFilename(ts int64) string {
	return health.SnapshotFilename(ts)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/economy/health/ -v -run TestPersistence`
Expected: FAIL — symbols undefined.

- [ ] **Step 3: Implement persistence**

Create `internal/economy/health/persistence.go`:

```go
package health

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"gopkg.in/yaml.v2"
)

// SnapshotFilename returns the canonical filename for a snapshot at
// the given unix-ts. Exposed for test parity.
func SnapshotFilename(unixTs int64) string {
	return fmt.Sprintf("%d.yaml", unixTs)
}

// snapshotDir returns the production snapshot directory. Tests use
// the *To variants below to redirect storage to t.TempDir().
func snapshotDir() string {
	return filepath.Join(configs.GetFilePathsConfig().DataFiles.String(), "economy", "snapshots")
}

// WriteSnapshot writes to the production directory.
func WriteSnapshot(s Snapshot) error { return WriteSnapshotTo(snapshotDir(), s) }

// LoadSnapshot reads from the production directory.
func LoadSnapshot(unixTs int64) (*Snapshot, error) { return LoadSnapshotFrom(snapshotDir(), unixTs) }

// ListSnapshots lists from the production directory.
func ListSnapshots() []SnapshotMeta { return ListSnapshotsFrom(snapshotDir()) }

// PruneSnapshots prunes from the production directory.
func PruneSnapshots(retentionDays int) (int, error) {
	return PruneSnapshotsIn(snapshotDir(), retentionDays)
}

// WriteSnapshotTo writes a snapshot YAML to the given directory.
// Creates the directory if missing. Filename is "{unix_ts}.yaml".
func WriteSnapshotTo(dir string, s Snapshot) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %q: %w", dir, err)
	}
	bytes, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	path := filepath.Join(dir, SnapshotFilename(s.UnixTs))
	return os.WriteFile(path, bytes, 0o644)
}

// LoadSnapshotFrom reads "{unix_ts}.yaml" from dir.
func LoadSnapshotFrom(dir string, unixTs int64) (*Snapshot, error) {
	path := filepath.Join(dir, SnapshotFilename(unixTs))
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Snapshot
	if err := yaml.Unmarshal(bytes, &s); err != nil {
		return nil, fmt.Errorf("unmarshal %q: %w", path, err)
	}
	return &s, nil
}

// SnapshotMeta is a lightweight directory entry. Used for fast listing
// without parsing every snapshot.
type SnapshotMeta struct {
	UnixTs      int64
	Manual      bool
	ManualLabel string
}

// ListSnapshotsFrom returns metas sorted by timestamp descending. The
// Manual + ManualLabel fields require a one-line peek into each YAML;
// the cost is small at hourly cadence.
func ListSnapshotsFrom(dir string) []SnapshotMeta {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	out := make([]SnapshotMeta, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		base := strings.TrimSuffix(e.Name(), ".yaml")
		ts, err := strconv.ParseInt(base, 10, 64)
		if err != nil {
			continue
		}
		meta := SnapshotMeta{UnixTs: ts}
		// Peek for manual flag — cheap parse of a few keys.
		if s, err := LoadSnapshotFrom(dir, ts); err == nil && s != nil {
			meta.Manual = s.Manual
			meta.ManualLabel = s.ManualLabel
		}
		out = append(out, meta)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UnixTs > out[j].UnixTs })
	return out
}

// PruneSnapshotsIn deletes auto-snapshots in dir older than
// retentionDays. Manual snapshots are never pruned. Returns the number
// of files deleted.
func PruneSnapshotsIn(dir string, retentionDays int) (int, error) {
	if retentionDays <= 0 {
		return 0, nil
	}
	cutoff := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour).Unix()
	deleted := 0
	for _, meta := range ListSnapshotsFrom(dir) {
		if meta.Manual {
			continue
		}
		if meta.UnixTs >= cutoff {
			continue
		}
		path := filepath.Join(dir, SnapshotFilename(meta.UnixTs))
		if err := os.Remove(path); err != nil {
			return deleted, fmt.Errorf("remove %q: %w", path, err)
		}
		deleted++
	}
	return deleted, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/economy/health/ -v -run TestPersistence`
Expected: PASS (3 subtests).

- [ ] **Step 5: Commit**

```bash
git add internal/economy/health/persistence.go internal/economy/health/persistence_test.go
git commit -m "feat(economy/health): snapshot read/write/list/prune

Persistence is split into bare-directory helpers (*To/*From/*In) for
tests + thin wrappers that bind to the configured data path for prod.
PruneSnapshots leaves manual snapshots untouched.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 9: Per-shop and per-discipline scoring

**Files:**
- Create: `internal/economy/health/scoring.go`
- Test: `internal/economy/health/scoring_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/economy/health/scoring_test.go`:

```go
package health_test

import (
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/economy/health"
)

func TestScore_PerShop_WeightedByRestockQty(t *testing.T) {
	shop := health.ShopSnapshot{
		Stock: []health.StockSnapshot{
			{ItemId: 1, RestockQty: 5, Current: 5, Max: 10},  // 50% fill, weight 5
			{ItemId: 2, RestockQty: 10, Current: 10, Max: 10}, // 100% fill, weight 10
		},
	}
	score := health.PerShopScore(shop)
	// weighted: (5*0.5 + 10*1.0) / (5+10) = 12.5/15 = 0.8333 → 83.33
	if math.Abs(score-83.33) > 0.5 {
		t.Errorf("got %.2f, want ~83.33", score)
	}
}

func TestScore_PerShop_NoStockReturnsNil(t *testing.T) {
	shop := health.ShopSnapshot{}
	if v, ok := health.PerShopScoreOpt(shop); ok {
		t.Errorf("got %v, want (_, false) for empty shop", v)
	}
}

func TestScore_PerCraftSupport_MeanOfShops(t *testing.T) {
	snap := health.Snapshot{
		Shops: []health.ShopSnapshot{
			{CraftSupport: "blacksmithing", Stock: []health.StockSnapshot{{RestockQty: 1, Current: 4, Max: 10}}}, // 40
			{CraftSupport: "blacksmithing", Stock: []health.StockSnapshot{{RestockQty: 1, Current: 8, Max: 10}}}, // 80
			{CraftSupport: "cooking", Stock: []health.StockSnapshot{{RestockQty: 1, Current: 5, Max: 10}}},        // 50
		},
	}
	scores := health.PerCraftSupportScores(snap)
	if math.Abs(scores["blacksmithing"]-60) > 0.01 {
		t.Errorf("blacksmithing: got %.2f, want 60", scores["blacksmithing"])
	}
	if math.Abs(scores["cooking"]-50) > 0.01 {
		t.Errorf("cooking: got %.2f, want 50", scores["cooking"])
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/economy/health/ -v -run TestScore`
Expected: FAIL — scoring symbols undefined.

- [ ] **Step 3: Implement scoring**

Create `internal/economy/health/scoring.go`:

```go
package health

// PerShopScore returns a 0-100 health score for one shop, weighted by
// RestockQty. Shops with no stock entries return 0; callers that want
// to distinguish "no data" from "zero" should use PerShopScoreOpt.
func PerShopScore(s ShopSnapshot) float64 {
	v, _ := PerShopScoreOpt(s)
	return v
}

// PerShopScoreOpt returns (score, true) when the shop has stock data
// to score, or (0, false) when there is no signal.
func PerShopScoreOpt(s ShopSnapshot) (float64, bool) {
	if len(s.Stock) == 0 {
		return 0, false
	}
	var weightedSum float64
	var totalWeight float64
	for _, e := range s.Stock {
		if e.Max <= 0 {
			continue
		}
		fill := float64(e.Current) / float64(e.Max)
		if fill < 0 {
			fill = 0
		}
		if fill > 1 {
			fill = 1
		}
		weight := float64(e.RestockQty)
		if weight < 1 {
			weight = 1
		}
		weightedSum += weight * fill
		totalWeight += weight
	}
	if totalWeight == 0 {
		return 0, false
	}
	return 100 * weightedSum / totalWeight, true
}

// PerCraftSupportScores returns mean per-shop score grouped by the
// craft discipline each shop supports. Shops with empty CraftSupport
// roll into key "" (should never happen in production thanks to
// startup validation; surfaces clearly in the UI as "(uncategorized)"
// if it ever does).
func PerCraftSupportScores(snap Snapshot) map[string]float64 {
	type bucket struct {
		sum   float64
		count int
	}
	buckets := map[string]*bucket{}
	for _, s := range snap.Shops {
		score, ok := PerShopScoreOpt(s)
		if !ok {
			continue
		}
		b, exists := buckets[s.CraftSupport]
		if !exists {
			b = &bucket{}
			buckets[s.CraftSupport] = b
		}
		b.sum += score
		b.count++
	}
	out := map[string]float64{}
	for k, b := range buckets {
		if b.count > 0 {
			out[k] = b.sum / float64(b.count)
		}
	}
	return out
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/economy/health/ -v -run TestScore`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/economy/health/scoring.go internal/economy/health/scoring_test.go
git commit -m "feat(economy/health): per-shop and per-discipline scoring

Shop score weights item fills by RestockQty so high-throughput items
dominate. Per-discipline score is the mean of shop scores grouped by
CraftSupport tag — the answer to 'is the supply chain supporting
discipline X?'. Empty-stock shops return (_, false) instead of
misleading zero.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 10: Caravan + forager scoring (cycle counting from history)

**Files:**
- Modify: `internal/economy/health/scoring.go`
- Test: `internal/economy/health/scoring_test.go` (append)

- [ ] **Step 1: Write failing tests**

Append to `internal/economy/health/scoring_test.go`:

```go
func TestScore_Caravan_CycleCount(t *testing.T) {
	// Build a 4-snapshot history that contains exactly one ThornwallDwell→
	// ThornwallDwell transition.
	hist := []*health.Snapshot{
		{Caravans: []health.CaravanSnapshot{{InstId: 1, State: "thornwall_dwell"}}},     // t-3
		{Caravans: []health.CaravanSnapshot{{InstId: 1, State: "outbound_transit"}}},     // t-2
		{Caravans: []health.CaravanSnapshot{{InstId: 1, State: "stillwater_dwell"}}},     // t-1
		{Caravans: []health.CaravanSnapshot{{InstId: 1, State: "thornwall_dwell"}}},     // now
	}
	cycles := health.CountCaravanCycles(1, hist)
	if cycles != 1 {
		t.Errorf("got %d cycles, want 1", cycles)
	}
}

func TestScore_Forager_CycleCount(t *testing.T) {
	hist := []*health.Snapshot{
		{Foragers: []health.ForagerSnapshot{{InstId: 7, State: "resting"}}},
		{Foragers: []health.ForagerSnapshot{{InstId: 7, State: "foraging"}}},
		{Foragers: []health.ForagerSnapshot{{InstId: 7, State: "resting"}}},
		{Foragers: []health.ForagerSnapshot{{InstId: 7, State: "foraging"}}},
		{Foragers: []health.ForagerSnapshot{{InstId: 7, State: "resting"}}},
	}
	cycles := health.CountForagerCycles(7, hist)
	if cycles != 2 {
		t.Errorf("got %d cycles, want 2", cycles)
	}
}

func TestScore_Caravan_InsufficientHistory(t *testing.T) {
	cur := health.Snapshot{Caravans: []health.CaravanSnapshot{{InstId: 1, State: "thornwall_dwell"}}}
	score, ok := health.PerCaravanScore(1, cur, nil) // no history
	if ok {
		t.Errorf("got (%v, true), want (_, false) for no history", score)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/economy/health/ -v -run TestScore_Caravan -run TestScore_Forager`
Expected: FAIL.

- [ ] **Step 3: Implement cycle counting**

Append to `internal/economy/health/scoring.go`:

```go
// CountCaravanCycles returns the number of ThornwallDwell→ThornwallDwell
// cycles for instId across history. History must be ordered oldest→
// newest. A cycle is one entry into thornwall_dwell preceded by a
// non-thornwall_dwell state.
func CountCaravanCycles(instId int, history []*Snapshot) int {
	return countStateReturns(history, "thornwall_dwell", func(s *Snapshot) string {
		for _, c := range s.Caravans {
			if c.InstId == instId {
				return c.State
			}
		}
		return ""
	})
}

// CountForagerCycles returns the number of Resting→Resting transitions
// for instId across history.
func CountForagerCycles(instId int, history []*Snapshot) int {
	return countStateReturns(history, "resting", func(s *Snapshot) string {
		for _, f := range s.Foragers {
			if f.InstId == instId {
				return f.State
			}
		}
		return ""
	})
}

func countStateReturns(history []*Snapshot, target string, lookup func(*Snapshot) string) int {
	cycles := 0
	prev := ""
	for _, s := range history {
		cur := lookup(s)
		if cur == target && prev != "" && prev != target {
			cycles++
		}
		if cur != "" {
			prev = cur
		}
	}
	return cycles
}

// PerCaravanScore returns (score, true) if there's enough history to
// compute. Insufficient history (fewer than minHistory entries) returns
// (_, false). Score = cycleScore - stuckPenalty, clamped to [0, 100].
const (
	minHistoryForCycles = 24   // ~24 hourly samples = 1 day baseline
	stuckThresholdRounds = 5000 // any state held longer than this triggers the penalty
	stuckPenalty         = 30  // points deducted when stuck
)

func PerCaravanScore(instId int, cur Snapshot, history []*Snapshot) (float64, bool) {
	if len(history) < minHistoryForCycles {
		return 0, false
	}
	cycles := CountCaravanCycles(instId, history)
	expectedPerWindow := float64(len(history)) / 24.0 // 1 cycle/day default
	if expectedPerWindow <= 0 {
		expectedPerWindow = 1
	}
	score := 100 * float64(cycles) / expectedPerWindow
	if score > 100 {
		score = 100
	}

	// Stuck penalty: any caravan whose current state has been held
	// longer than stuckThresholdRounds loses points.
	for _, c := range cur.Caravans {
		if c.InstId == instId && c.StateEnteredRound > 0 {
			if cur.Round > c.StateEnteredRound &&
				cur.Round-c.StateEnteredRound > stuckThresholdRounds {
				score -= stuckPenalty
			}
			break
		}
	}
	if score < 0 {
		score = 0
	}
	return score, true
}

// PerForagerScore mirrors PerCaravanScore for foragers (Resting→Resting
// cycle counting + same stuck-penalty logic).
func PerForagerScore(instId int, cur Snapshot, history []*Snapshot) (float64, bool) {
	if len(history) < minHistoryForCycles {
		return 0, false
	}
	cycles := CountForagerCycles(instId, history)
	expectedPerWindow := float64(len(history)) / 8.0 // ~3 cycles/day
	if expectedPerWindow <= 0 {
		expectedPerWindow = 1
	}
	score := 100 * float64(cycles) / expectedPerWindow
	if score > 100 {
		score = 100
	}

	for _, f := range cur.Foragers {
		if f.InstId == instId && f.StateEnteredRound > 0 {
			if cur.Round > f.StateEnteredRound &&
				cur.Round-f.StateEnteredRound > stuckThresholdRounds {
				score -= stuckPenalty
			}
			break
		}
	}
	if score < 0 {
		score = 0
	}
	return score, true
}
```

The fixed `stuckThresholdRounds = 5000` is an MVP shortcut — the spec called for `2 × expectedDuration(state)` per-state, but the per-state expected durations aren't yet a single config knob. 5000 rounds is roughly an order of magnitude longer than the current dwell timer, matching the spirit of "if it's stuck this long, something's wrong." Tune later when per-state config exists.

The 24h-cycle and 8h-forager-cycle assumptions are MVP defaults. The spec calls for reading "expected cycle length" from existing config, but no such single knob exists yet — the caravan dwell + transit duration combine into total cycle length implicitly. Use the hardcoded defaults for MVP and document the simplification in the commit message.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/economy/health/ -v -run TestScore`
Expected: PASS (5+ subtests).

- [ ] **Step 5: Commit**

```bash
git add internal/economy/health/scoring.go internal/economy/health/scoring_test.go
git commit -m "feat(economy/health): caravan + forager cycle counting

Counts state-return transitions across snapshot history (Thornwall→
Thornwall for caravans, Resting→Resting for foragers). Insufficient
history (<24 samples) yields (_, false). Expected cadence is hardcoded
per MVP — wiring to config-driven values is deferred.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 11: Overall economy score + Score() entry point

**Files:**
- Modify: `internal/economy/health/scoring.go`
- Test: `internal/economy/health/scoring_test.go` (append)

- [ ] **Step 1: Write failing test**

Append to `scoring_test.go`:

```go
func TestScore_OverallWeightsShopsHeaviest(t *testing.T) {
	cur := health.Snapshot{
		Shops: []health.ShopSnapshot{
			{CraftSupport: "blacksmithing", Stock: []health.StockSnapshot{{RestockQty: 1, Current: 5, Max: 10}}}, // 50
		},
		Caravans: []health.CaravanSnapshot{{InstId: 1, State: "thornwall_dwell"}},
		Foragers: []health.ForagerSnapshot{{InstId: 7, State: "resting"}},
	}

	// Build 24 history entries that yield ~24 caravan cycles and ~24
	// forager cycles (each entry alternates state). With the default
	// expected cadence, both score = 100.
	hist := make([]*health.Snapshot, 0, 24)
	caravanThornwall := &health.Snapshot{Caravans: []health.CaravanSnapshot{{InstId: 1, State: "thornwall_dwell"}}, Foragers: []health.ForagerSnapshot{{InstId: 7, State: "resting"}}}
	caravanTransit := &health.Snapshot{Caravans: []health.CaravanSnapshot{{InstId: 1, State: "outbound_transit"}}, Foragers: []health.ForagerSnapshot{{InstId: 7, State: "foraging"}}}
	for i := 0; i < 24; i++ {
		if i%2 == 0 {
			hist = append(hist, caravanThornwall)
		} else {
			hist = append(hist, caravanTransit)
		}
	}

	scores := health.Score(&cur, hist)

	if scores.PerShop[0].Score != 50 {
		t.Errorf("PerShop[0]: got %.2f, want 50", scores.PerShop[0].Score)
	}
	// With shop=50, caravan~100, forager~100 and weights 0.6/0.2/0.2:
	// overall = (0.6*50 + 0.2*100 + 0.2*100) / 1.0 = 70.
	// Allow ±15 for variation in cycle counting against the chosen pattern.
	if scores.OverallScore < 55 || scores.OverallScore > 85 {
		t.Errorf("OverallScore: got %.2f, want in [55, 85] (shops weighted heaviest)", scores.OverallScore)
	}
	// Shops weighted heaviest sanity: overall should sit closer to MeanShop
	// (50) than to the unweighted mean of (50, MeanCaravan, MeanForager).
	unweightedMean := (scores.MeanShop + scores.MeanCaravan + scores.MeanForager) / 3
	if abs(scores.OverallScore-scores.MeanShop) > abs(scores.OverallScore-unweightedMean) {
		t.Errorf("Overall %.2f is closer to unweighted mean %.2f than to MeanShop %.2f — shops aren't weighted heaviest",
			scores.OverallScore, unweightedMean, scores.MeanShop)
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/economy/health/ -v -run TestScore_Overall`
Expected: FAIL — `Score` and `Scores` types undefined.

- [ ] **Step 3: Implement `Score()` entry point + `Scores` type**

Append to `scoring.go`:

```go
// Scores is the bundle returned by Score(). Each per-entity score
// has a HasScore flag for "insufficient history" cases.
type Scores struct {
	OverallScore float64

	PerShop      []ShopScoreRow
	PerCraftSupport map[string]float64
	PerCaravan   []EntityScoreRow
	PerForager   []EntityScoreRow

	// Component aggregate scores, surfaced as the four Bootstrap cards
	// at the top of the dashboard.
	MeanShop    float64
	MeanCaravan float64
	MeanForager float64
}

type ShopScoreRow struct {
	Zone         string
	MobId        int
	RoomId       int
	Name         string
	CraftSupport string
	Score        float64
	HasScore     bool
}

type EntityScoreRow struct {
	InstId   int
	Name     string
	Score    float64
	HasScore bool
}

// Score is the dashboard's main scoring entry point.
func Score(cur *Snapshot, history []*Snapshot) Scores {
	out := Scores{}
	if cur == nil {
		return out
	}

	// Per-shop scores.
	out.PerShop = make([]ShopScoreRow, 0, len(cur.Shops))
	var shopSum float64
	var shopCount int
	for _, s := range cur.Shops {
		score, ok := PerShopScoreOpt(s)
		out.PerShop = append(out.PerShop, ShopScoreRow{
			Zone: s.Zone, MobId: s.MobId, RoomId: s.RoomId, Name: s.Name,
			CraftSupport: s.CraftSupport, Score: score, HasScore: ok,
		})
		if ok {
			shopSum += score
			shopCount++
		}
	}
	if shopCount > 0 {
		out.MeanShop = shopSum / float64(shopCount)
	}

	out.PerCraftSupport = PerCraftSupportScores(*cur)

	// Per-caravan / per-forager.
	out.PerCaravan = make([]EntityScoreRow, 0, len(cur.Caravans))
	var caravanSum float64
	var caravanCount int
	for _, c := range cur.Caravans {
		score, ok := PerCaravanScore(c.InstId, *cur, history)
		out.PerCaravan = append(out.PerCaravan, EntityScoreRow{
			InstId: c.InstId, Name: c.Name, Score: score, HasScore: ok,
		})
		if ok {
			caravanSum += score
			caravanCount++
		}
	}
	if caravanCount > 0 {
		out.MeanCaravan = caravanSum / float64(caravanCount)
	}

	out.PerForager = make([]EntityScoreRow, 0, len(cur.Foragers))
	var foragerSum float64
	var foragerCount int
	for _, f := range cur.Foragers {
		score, ok := PerForagerScore(f.InstId, *cur, history)
		out.PerForager = append(out.PerForager, EntityScoreRow{
			InstId: f.InstId, Name: f.Name, Score: score, HasScore: ok,
		})
		if ok {
			foragerSum += score
			foragerCount++
		}
	}
	if foragerCount > 0 {
		out.MeanForager = foragerSum / float64(foragerCount)
	}

	// Overall: weighted mean. Renormalize over components that have
	// data (avoids dragging the score to 0 when history is short).
	const wShop, wCaravan, wForager = 0.6, 0.2, 0.2
	var wSum, weighted float64
	if shopCount > 0 {
		weighted += wShop * out.MeanShop
		wSum += wShop
	}
	if caravanCount > 0 {
		weighted += wCaravan * out.MeanCaravan
		wSum += wCaravan
	}
	if foragerCount > 0 {
		weighted += wForager * out.MeanForager
		wSum += wForager
	}
	if wSum > 0 {
		out.OverallScore = weighted / wSum
	}

	return out
}
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/economy/health/ -v -run TestScore`
Expected: PASS for all subtests.

- [ ] **Step 5: Commit**

```bash
git add internal/economy/health/scoring.go internal/economy/health/scoring_test.go
git commit -m "feat(economy/health): overall economy score + Scores bundle

Score() is the dashboard's main entry point. OverallScore weights
shops 0.6 / caravans 0.2 / foragers 0.2 and renormalizes when one
component has insufficient history.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 12: Closest-snapshot picker + delta computation

**Files:**
- Create: `internal/economy/health/delta.go`
- Test: `internal/economy/health/delta_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/economy/health/delta_test.go`:

```go
package health_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/economy/health"
)

func TestPickClosest_PrefersWithinTolerance(t *testing.T) {
	metas := []health.SnapshotMeta{
		{UnixTs: 100}, {UnixTs: 200}, {UnixTs: 300}, {UnixTs: 1000},
	}
	got := health.PickClosestSnapshot(metas, 250, 60)
	if got == nil || got.UnixTs != 200 {
		t.Errorf("got %v, want 200 (within tolerance 60)", got)
	}
}

func TestPickClosest_NoneInTolerance(t *testing.T) {
	metas := []health.SnapshotMeta{{UnixTs: 100}, {UnixTs: 1000}}
	got := health.PickClosestSnapshot(metas, 500, 60)
	if got != nil {
		t.Errorf("got %v, want nil (no snapshot within 60s of 500)", got)
	}
}

func TestComputeShopDelta_GoldAndStock(t *testing.T) {
	now := health.ShopSnapshot{
		Zone: "z", MobId: 1, RoomId: 1, Gold: 600,
		Stock: []health.StockSnapshot{{ItemId: 40001, Bucket: "base", Current: 5, Max: 10}},
	}
	old := health.ShopSnapshot{
		Zone: "z", MobId: 1, RoomId: 1, Gold: 500,
		Stock: []health.StockSnapshot{{ItemId: 40001, Bucket: "base", Current: 8, Max: 10}},
	}
	d := health.ComputeShopDelta(now, &old)
	if d.GoldDelta != 100 {
		t.Errorf("GoldDelta: got %d, want 100", d.GoldDelta)
	}
	if d.BucketDeltas["base"] != -3 {
		t.Errorf("base bucket: got %d, want -3", d.BucketDeltas["base"])
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/economy/health/ -v -run TestPickClosest -run TestComputeShopDelta`
Expected: FAIL.

- [ ] **Step 3: Implement delta module**

Create `internal/economy/health/delta.go`:

```go
package health

// PickClosestSnapshot returns the meta whose UnixTs is nearest to
// target, but only within toleranceSeconds. Returns nil if no meta
// is within tolerance. metas is expected to be sorted descending by
// UnixTs (the order ListSnapshots returns).
func PickClosestSnapshot(metas []SnapshotMeta, target int64, toleranceSeconds int64) *SnapshotMeta {
	var best *SnapshotMeta
	var bestDelta int64 = -1
	for i := range metas {
		m := metas[i]
		delta := m.UnixTs - target
		if delta < 0 {
			delta = -delta
		}
		if toleranceSeconds > 0 && delta > toleranceSeconds {
			continue
		}
		if bestDelta < 0 || delta < bestDelta {
			bestDelta = delta
			best = &metas[i]
		}
	}
	return best
}

// ShopDelta is the per-shop delta vs a comparison snapshot. Bucket
// deltas are sums of per-bucket Current values (this snapshot − old).
type ShopDelta struct {
	GoldDelta    int
	BucketDeltas map[string]int
}

// ComputeShopDelta returns the shop's delta against old. If old is
// nil, returns a zero-value delta with empty BucketDeltas (the UI
// should distinguish "no comparison snapshot" from "zero change").
func ComputeShopDelta(now ShopSnapshot, old *ShopSnapshot) ShopDelta {
	d := ShopDelta{BucketDeltas: map[string]int{}}
	if old == nil {
		return d
	}
	d.GoldDelta = now.Gold - old.Gold

	bucketSum := func(s ShopSnapshot) map[string]int {
		out := map[string]int{}
		for _, e := range s.Stock {
			if e.Bucket == "" {
				continue
			}
			out[e.Bucket] += e.Current
		}
		return out
	}
	curBuckets := bucketSum(now)
	oldBuckets := bucketSum(*old)
	for b, n := range curBuckets {
		d.BucketDeltas[b] = n - oldBuckets[b]
	}
	for b, n := range oldBuckets {
		if _, seen := curBuckets[b]; !seen {
			d.BucketDeltas[b] = -n
		}
	}
	return d
}

// FindShopInSnapshot returns a pointer to the matching shop in s, or
// nil if absent. Match key is (Zone, MobId, RoomId).
func FindShopInSnapshot(s *Snapshot, zone string, mobId, roomId int) *ShopSnapshot {
	if s == nil {
		return nil
	}
	for i := range s.Shops {
		if s.Shops[i].Zone == zone && s.Shops[i].MobId == mobId && s.Shops[i].RoomId == roomId {
			return &s.Shops[i]
		}
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/economy/health/ -v -run TestPickClosest -run TestComputeShopDelta`
Expected: PASS.

- [ ] **Step 5: Run full health package**

Run: `go test ./internal/economy/health/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/economy/health/delta.go internal/economy/health/delta_test.go
git commit -m "feat(economy/health): closest-snapshot picker + shop deltas

PickClosestSnapshot finds the comparison snapshot for each delta
column (1h/6h/1d/3d/1w). ComputeShopDelta diffs gold + bucket totals.
Caravan/forager cycle counters are computed in scoring.go and don't
need a delta path.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 13: Add config knobs

**Files:**
- Modify: `internal/configs/config.balance.go`
- Modify: `internal/configs/config.balance.misc.go`
- Modify: `_datafiles/config.yaml`

- [ ] **Step 1: Add fields to Balance struct**

Edit `internal/configs/config.balance.go`. Append a new section near the bottom:

```go
	// ── ECONOMY HEALTH DASHBOARD ─────────────────────────────────────────────
	EconomySnapshotIntervalHours ConfigInt   `yaml:"EconomySnapshotIntervalHours"` // Wall-clock cadence (default 1)
	EconomySnapshotRetentionDays ConfigInt   `yaml:"EconomySnapshotRetentionDays"` // Auto-snapshot retention (default 30)
	EconomyScoreWeightShop       ConfigFloat `yaml:"EconomyScoreWeightShop"`       // Overall-score weight for shops (default 0.6)
	EconomyScoreWeightCaravan    ConfigFloat `yaml:"EconomyScoreWeightCaravan"`    // (default 0.2)
	EconomyScoreWeightForager    ConfigFloat `yaml:"EconomyScoreWeightForager"`    // (default 0.2)
```

- [ ] **Step 2: Add defaults to validateMisc()**

Edit `internal/configs/config.balance.misc.go`. Append inside `validateMisc`:

```go
	// ── ECONOMY HEALTH DASHBOARD ─────────────────────────────────────────────
	if b.EconomySnapshotIntervalHours <= 0 {
		b.EconomySnapshotIntervalHours = 1
	}
	if b.EconomySnapshotRetentionDays <= 0 {
		b.EconomySnapshotRetentionDays = 30
	}
	if b.EconomyScoreWeightShop <= 0 {
		b.EconomyScoreWeightShop = 0.6
	}
	if b.EconomyScoreWeightCaravan <= 0 {
		b.EconomyScoreWeightCaravan = 0.2
	}
	if b.EconomyScoreWeightForager <= 0 {
		b.EconomyScoreWeightForager = 0.2
	}
```

- [ ] **Step 3: Surface in config.yaml**

Edit `_datafiles/config.yaml`. Find the `Balance:` section and append:

```yaml
  EconomySnapshotIntervalHours: 1
  EconomySnapshotRetentionDays: 30
  EconomyScoreWeightShop: 0.6
  EconomyScoreWeightCaravan: 0.2
  EconomyScoreWeightForager: 0.2
```

- [ ] **Step 4: Build and run config tests**

Run: `go build ./... && go test ./internal/configs/...`
Expected: clean + PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/configs/config.balance.go internal/configs/config.balance.misc.go _datafiles/config.yaml
git commit -m "feat(configs): economy health dashboard knobs

Cadence (1h), retention (30d), and overall-score component weights.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 14: Wire snapshot ticker in main.go

**Files:**
- Modify: `main.go`
- Modify: `.gitignore`

- [ ] **Step 1: Add gitignore entry**

Edit `.gitignore`. Append:

```
# Economy health snapshots — runtime state, regenerated by the server.
_datafiles/economy/snapshots/
```

- [ ] **Step 2: Add ticker goroutine to main.go**

Locate where the existing background goroutines start in `main.go` (around the `// Start a goroutine to accept incoming connections` block at line ~1010, and the long-running goroutine at line 390). Add a new goroutine after the others:

```go
// Hourly economy-health snapshot capture. Runs while the server is
// up; uses workerShutdownChan to halt cleanly. Pruning runs once
// per day.
go func() {
	defer func() {
		if r := recover(); r != nil {
			mudlog.Error("PANIC", "where", "economy_snapshot_ticker", "error", r)
		}
	}()
	b := configs.GetBalanceConfig()
	hours := int(b.EconomySnapshotIntervalHours)
	if hours <= 0 {
		hours = 1
	}
	ticker := time.NewTicker(time.Duration(hours) * time.Hour)
	defer ticker.Stop()
	pruneTicker := time.NewTicker(24 * time.Hour)
	defer pruneTicker.Stop()
	for {
		select {
		case <-workerShutdownChan:
			return
		case <-ticker.C:
			snap := health.CaptureSnapshot()
			if err := health.WriteSnapshot(snap); err != nil {
				mudlog.Error("economy snapshot write", "error", err)
			}
		case <-pruneTicker.C:
			retention := int(configs.GetBalanceConfig().EconomySnapshotRetentionDays)
			if _, err := health.PruneSnapshots(retention); err != nil {
				mudlog.Error("economy snapshot prune", "error", err)
			}
		}
	}
}()
```

Add the import at the top of `main.go`: `"github.com/GoMudEngine/GoMud/internal/economy/health"`. The `time` and `mudlog` and `configs` imports are already present.

- [ ] **Step 3: Build to confirm**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 4: Boot server briefly to confirm no panic**

Run: `go run main.go &` then `sleep 8 && pkill -f "go run main.go" || pkill -f "main.go"`. Inspect output for any panics or "economy snapshot" log lines.
Expected: clean startup, no panics. (No snapshot will fire in 8s — the ticker fires after 1h. Goal is just verifying no init-time crash.)

- [ ] **Step 5: Commit**

```bash
git add main.go .gitignore
git commit -m "feat(economy/health): hourly snapshot ticker

Runs CaptureSnapshot+WriteSnapshot every EconomySnapshotIntervalHours
(default 1h) and PruneSnapshots once per day. Snapshots are gitignored
runtime state.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 15: Web handlers (server-side)

**Files:**
- Create: `internal/web/admin.economyhealth.go`

- [ ] **Step 1: Implement handlers**

Create `internal/web/admin.economyhealth.go`:

```go
package web

import (
	"encoding/json"
	"net/http"
	"text/template"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/economy/health"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

func economyIndex(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("index.html").Funcs(funcMap).ParseFiles(
		configs.GetFilePathsConfig().AdminHtml.String()+"/_header.html",
		configs.GetFilePathsConfig().AdminHtml.String()+"/economy/index.html",
		configs.GetFilePathsConfig().AdminHtml.String()+"/_footer.html",
	)
	if err != nil {
		mudlog.Error("HTML Template", "error", err)
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, struct{}{}); err != nil {
		mudlog.Error("HTML Execute", "error", err)
	}
}

// economyAPIResponse is the JSON shape the dashboard fetches.
type economyAPIResponse struct {
	Snapshot health.Snapshot          `json:"snapshot"`
	Scores   health.Scores            `json:"scores"`
	Deltas   map[string]deltaSet      `json:"deltas"` // keyed by "1h", "6h", "1d", "3d", "1w"
}

type deltaSet struct {
	UnixTs int64                            `json:"unix_ts,omitempty"`
	Shops  map[string]health.ShopDelta      `json:"shops"` // key "{zone}/{mobId}/{roomId}"
}

func economyAPI(w http.ResponseWriter, r *http.Request) {
	cur := health.CaptureSnapshot()
	metas := health.ListSnapshots()

	// Caller-provided history window (last 168 entries — 7 days at 1h).
	historyMetas := metas
	if len(historyMetas) > 168 {
		historyMetas = historyMetas[:168]
	}
	history := make([]*health.Snapshot, 0, len(historyMetas))
	for i := len(historyMetas) - 1; i >= 0; i-- { // oldest first
		s, err := health.LoadSnapshot(historyMetas[i].UnixTs)
		if err != nil || s == nil {
			continue
		}
		history = append(history, s)
	}

	scores := health.Score(&cur, history)

	// Deltas at five offsets.
	now := time.Now().Unix()
	deltas := map[string]deltaSet{}
	offsets := map[string]int64{
		"1h": 3600,
		"6h": 6 * 3600,
		"1d": 24 * 3600,
		"3d": 3 * 24 * 3600,
		"1w": 7 * 24 * 3600,
	}
	for label, off := range offsets {
		target := now - off
		// Tolerance: ±50% of the offset, so the picker still works for sparse history.
		tolerance := off / 2
		if tolerance < 1800 {
			tolerance = 1800
		}
		meta := health.PickClosestSnapshot(metas, target, tolerance)
		ds := deltaSet{Shops: map[string]health.ShopDelta{}}
		if meta != nil {
			ds.UnixTs = meta.UnixTs
			old, err := health.LoadSnapshot(meta.UnixTs)
			if err == nil && old != nil {
				for _, s := range cur.Shops {
					oldShop := health.FindShopInSnapshot(old, s.Zone, s.MobId, s.RoomId)
					key := fmt.Sprintf("%s/%d/%d", s.Zone, s.MobId, s.RoomId)
					ds.Shops[key] = health.ComputeShopDelta(s, oldShop)
				}
			}
		}
		deltas[label] = ds
	}

	resp := economyAPIResponse{
		Snapshot: cur,
		Scores:   scores,
		Deltas:   deltas,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		mudlog.Error("economy API encode", "error", err)
	}
}

func economySnapshotAPI(w http.ResponseWriter, r *http.Request) {
	label := r.URL.Query().Get("label")
	snap := health.CaptureSnapshot()
	snap.Manual = true
	snap.ManualLabel = label
	if err := health.WriteSnapshot(snap); err != nil {
		mudlog.Error("manual snapshot write", "error", err)
		http.Error(w, "write failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"unix_ts": snap.UnixTs, "manual": true})
}
```

Add `"fmt"` to the imports.

- [ ] **Step 2: Build to verify**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 3: Commit**

```bash
git add internal/web/admin.economyhealth.go
git commit -m "feat(web/admin): economy health handlers

Three endpoints: HTML index, JSON API (current + scores + deltas),
manual snapshot trigger. Wiring follows in next commit.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 16: Register routes + sidebar entry

**Files:**
- Modify: `internal/web/web.go`
- Modify: `_datafiles/html/admin/_header.html`

- [ ] **Step 1: Register routes**

Edit `internal/web/web.go`. After the existing Combat Stats / Progression registrations (around line ~344), add:

```go
	// Economy Health Admin
	http.HandleFunc("GET /admin/economy/", RunWithMUDLocked(
		doBasicAuth(economyIndex),
	))
	http.HandleFunc("GET /admin/api/economy/", RunWithMUDLocked(
		doBasicAuth(economyAPI),
	))
	http.HandleFunc("POST /admin/api/economy/snapshot", RunWithMUDLocked(
		doBasicAuth(economySnapshotAPI),
	))
```

- [ ] **Step 2: Add sidebar link**

Edit `_datafiles/html/admin/_header.html`. After the `Progression` link (around line 87), add:

```html
                    <a class="list-group-item list-group-item-action list-group-item-light p-3" href="/admin/economy/">Economy</a>
```

- [ ] **Step 3: Build to verify**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add internal/web/web.go _datafiles/html/admin/_header.html
git commit -m "feat(web/admin): wire /admin/economy/ routes + sidebar

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 17: HTML template

**Files:**
- Create: `_datafiles/html/admin/economy/index.html`

- [ ] **Step 1: Create template**

Create `_datafiles/html/admin/economy/index.html` modeled on `combatstats/index.html`. Key elements:

```html
{{template "header" .}}

<div class="container-fluid">
    <h1 class="mt-4">Economy Health Dashboard</h1>

    <!-- Five summary cards -->
    <div class="row mb-3">
        <div class="col-md-2">
            <div class="card text-center"><div class="card-body p-2">
                <h6 class="card-subtitle text-muted">Economy</h6>
                <h4 id="score-overall" class="card-title mb-0">—</h4>
            </div></div>
        </div>
        <div class="col-md-2">
            <div class="card text-center"><div class="card-body p-2">
                <h6 class="card-subtitle text-muted">Shops</h6>
                <h4 id="score-shop" class="card-title mb-0">—</h4>
            </div></div>
        </div>
        <div class="col-md-2">
            <div class="card text-center"><div class="card-body p-2">
                <h6 class="card-subtitle text-muted">Caravans</h6>
                <h4 id="score-caravan" class="card-title mb-0">—</h4>
            </div></div>
        </div>
        <div class="col-md-2">
            <div class="card text-center"><div class="card-body p-2">
                <h6 class="card-subtitle text-muted">Foragers</h6>
                <h4 id="score-forager" class="card-title mb-0">—</h4>
            </div></div>
        </div>
        <div class="col-md-4">
            <div class="card text-center"><div class="card-body p-2">
                <h6 class="card-subtitle text-muted">Last Snapshot</h6>
                <span id="last-ts">—</span>
                <input id="manual-label" type="text" placeholder="(optional label)"
                       class="form-control form-control-sm d-inline-block ml-2" style="width:auto">
                <button class="btn btn-sm btn-outline-info ml-2"
                        onclick="doSnapshot()">Snapshot Now</button>
            </div></div>
        </div>
    </div>

    <!-- Auto-refresh controls -->
    <div class="row mb-3">
        <div class="col-auto">
            <button class="btn btn-sm btn-primary" onclick="fetchData()">Refresh</button>
            <button id="btn-auto" class="btn btn-sm btn-outline-secondary ml-2"
                    onclick="toggleAuto()">Auto: OFF</button>
            <select id="sel-interval" class="custom-select custom-select-sm ml-2"
                    style="width:auto;display:inline-block;" onchange="updateInterval()">
                <option value="30" selected>30s</option>
                <option value="60">60s</option>
                <option value="120">2m</option>
            </select>
        </div>
    </div>

    <!-- Section A: Shopkeepers — by Craft Discipline -->
    <h3 class="mt-3">Shopkeepers — by Craft Discipline</h3>
    <table class="table table-sm table-striped">
        <thead><tr>
            <th>Discipline</th><th>Shops</th><th>Score</th>
            <th>Total Gold</th>
            <th>Δ1h</th><th>Δ6h</th><th>Δ1d</th><th>Δ3d</th><th>Δ1w</th>
        </tr></thead>
        <tbody id="tbl-discipline"></tbody>
    </table>

    <h4 class="mt-3">Per-Shop Detail</h4>
    <table class="table table-sm table-striped">
        <thead><tr>
            <th>Shop</th><th>Discipline</th><th>Score</th>
            <th>Gold</th>
            <th>Δ1h</th><th>Δ6h</th><th>Δ1d</th><th>Δ3d</th><th>Δ1w</th>
            <th>Stock by Bucket</th>
        </tr></thead>
        <tbody id="tbl-shops"></tbody>
    </table>

    <!-- Section B: Caravans -->
    <h3 class="mt-3">Caravans</h3>
    <table class="table table-sm table-striped">
        <thead><tr>
            <th>Name</th><th>Score</th><th>State</th><th>Time-in-State</th>
            <th>Room</th><th>Cargo</th>
            <th>Cycles 1h</th><th>6h</th><th>1d</th><th>3d</th><th>1w</th>
        </tr></thead>
        <tbody id="tbl-caravans"></tbody>
    </table>

    <!-- Section C: Foragers -->
    <h3 class="mt-3">Foragers</h3>
    <table class="table table-sm table-striped">
        <thead><tr>
            <th>Name</th><th>Score</th><th>Territory</th><th>State</th>
            <th>Time-in-State</th><th>Room</th><th>Cargo</th>
            <th>Cycles 1h</th><th>6h</th><th>1d</th><th>3d</th><th>1w</th>
        </tr></thead>
        <tbody id="tbl-foragers"></tbody>
    </table>
</div>

<script>
(function() {
    var autoTimer = null;
    var autoOn = false;

    var BUCKET_COLORS = {
        base: "#4e79a7", stillwater: "#76b7b2", thornwall: "#f28e2b",
        fernway: "#59a14f", overlap: "#bab0ac"
    };

    function colorScore(v) {
        if (v < 40) return "text-danger";
        if (v < 70) return "text-warning";
        return "text-success";
    }

    function fmtScore(score, hasScore) {
        if (!hasScore) return '<span class="text-muted">—</span>';
        return '<span class="' + colorScore(score) + '">' + score.toFixed(0) + '</span>';
    }

    function fmtDelta(v) {
        if (v === 0 || v === undefined) return '<span class="text-muted">0</span>';
        var sign = v > 0 ? '+' : '';
        var cls = v > 0 ? 'text-success' : 'text-danger';
        return '<span class="' + cls + '">' + sign + v + '</span>';
    }

    function bucketBar(stockEntries, totalCap) {
        if (!stockEntries || stockEntries.length === 0) return '';
        var sumByBucket = {};
        for (var i = 0; i < stockEntries.length; i++) {
            var e = stockEntries[i];
            if (!e.bucket) continue;
            sumByBucket[e.bucket] = (sumByBucket[e.bucket] || 0) + e.current;
        }
        var bar = '<div style="display:flex;height:14px;width:140px;">';
        for (var b in sumByBucket) {
            var pct = totalCap > 0 ? (sumByBucket[b] / totalCap * 100) : 0;
            var color = BUCKET_COLORS[b] || "#999";
            bar += '<div title="' + b + ': ' + sumByBucket[b] + '" ' +
                'style="width:' + pct.toFixed(0) + '%;background:' + color + '"></div>';
        }
        bar += '</div>';
        return bar;
    }

    function shopKey(s) { return s.zone + '/' + s.mob_id + '/' + s.room_id; }

    function totalCap(stockEntries) {
        var s = 0;
        for (var i = 0; i < (stockEntries || []).length; i++) s += stockEntries[i].max;
        return s;
    }

    function totalGold(shops) {
        var s = 0;
        for (var i = 0; i < shops.length; i++) s += shops[i].gold;
        return s;
    }

    function render(data) {
        var d = data;

        document.getElementById('score-overall').innerHTML = fmtScore(d.scores.OverallScore, true);
        document.getElementById('score-shop').innerHTML = fmtScore(d.scores.MeanShop, d.scores.MeanShop > 0);
        document.getElementById('score-caravan').innerHTML = fmtScore(d.scores.MeanCaravan, d.scores.MeanCaravan > 0);
        document.getElementById('score-forager').innerHTML = fmtScore(d.scores.MeanForager, d.scores.MeanForager > 0);
        document.getElementById('last-ts').textContent = d.snapshot.timestamp;

        // Group shops by craft discipline they support.
        var byDisc = {};
        for (var i = 0; i < d.snapshot.shops.length; i++) {
            var s = d.snapshot.shops[i];
            var disc = s.craft_support || "(uncategorized)";
            if (!byDisc[disc]) byDisc[disc] = [];
            byDisc[disc].push(s);
        }
        // Render discipline rollup. Stable order: known disciplines
        // first, then "general", then anything else (which would be
        // a validation gap).
        var DISC_ORDER = ['blacksmithing','alchemy','tailoring','cooking',
                          'jewelcrafting','enchanting','general'];
        var discHtml = '';
        var seen = {};
        var discKeys = [];
        for (var i = 0; i < DISC_ORDER.length; i++) {
            if (byDisc[DISC_ORDER[i]]) { discKeys.push(DISC_ORDER[i]); seen[DISC_ORDER[i]] = true; }
        }
        Object.keys(byDisc).sort().forEach(function(k){
            if (!seen[k]) discKeys.push(k);
        });
        for (var i = 0; i < discKeys.length; i++) {
            var disc = discKeys[i];
            var shops = byDisc[disc];
            var score = d.scores.PerCraftSupport[disc] || 0;
            var gold = totalGold(shops);
            discHtml += '<tr><td>' + disc + '</td><td>' + shops.length + '</td>' +
                        '<td>' + fmtScore(score, score > 0) + '</td>' +
                        '<td>' + gold + '</td>';
            // Sum gold deltas across shops in this discipline.
            ['1h','6h','1d','3d','1w'].forEach(function(label) {
                var sum = 0;
                for (var j = 0; j < shops.length; j++) {
                    var key = shopKey(shops[j]);
                    var sd = d.deltas[label] && d.deltas[label].shops[key];
                    if (sd) sum += sd.GoldDelta;
                }
                discHtml += '<td>' + fmtDelta(sum) + '</td>';
            });
            discHtml += '</tr>';
        }
        document.getElementById('tbl-discipline').innerHTML = discHtml;

        // Render per-shop rows.
        var shopHtml = '';
        for (var i = 0; i < d.scores.PerShop.length; i++) {
            var row = d.scores.PerShop[i];
            var snap = d.snapshot.shops[i];
            var key = shopKey(snap);
            shopHtml += '<tr><td>' + (row.Name || '#'+row.MobId) + '</td>' +
                        '<td>' + (row.CraftSupport || '—') + '</td>' +
                        '<td>' + fmtScore(row.Score, row.HasScore) + '</td>' +
                        '<td>' + snap.gold + '</td>';
            ['1h','6h','1d','3d','1w'].forEach(function(label) {
                var sd = d.deltas[label] && d.deltas[label].shops[key];
                shopHtml += '<td>' + (sd ? fmtDelta(sd.GoldDelta) : '<span class="text-muted">—</span>') + '</td>';
            });
            shopHtml += '<td>' + bucketBar(snap.stock, totalCap(snap.stock)) + '</td></tr>';
        }
        document.getElementById('tbl-shops').innerHTML = shopHtml;

        // Caravans.
        var caravanHtml = '';
        for (var i = 0; i < (d.snapshot.caravans || []).length; i++) {
            var c = d.snapshot.caravans[i];
            var sc = d.scores.PerCaravan[i] || {};
            var bar = bucketBar(
                Object.keys(c.cargo_by_bucket || {}).map(function(b){ return {bucket:b, current:c.cargo_by_bucket[b]}; }),
                c.cargo_capacity || 1
            );
            caravanHtml += '<tr><td>' + c.name + '</td>' +
                           '<td>' + fmtScore(sc.Score, sc.HasScore) + '</td>' +
                           '<td>' + c.state + '</td>' +
                           '<td>round +' + (d.snapshot.round - c.state_entered_round) + '</td>' +
                           '<td>' + c.room_id + '</td>' +
                           '<td>' + bar + ' ' + c.cargo_weight + '/' + c.cargo_capacity + ' lbs' + '</td>' +
                           '<td colspan="5" class="text-muted">cycles: see scoring</td></tr>';
        }
        document.getElementById('tbl-caravans').innerHTML = caravanHtml || '<tr><td colspan="11" class="text-muted">No caravans active.</td></tr>';

        // Foragers.
        var foragerHtml = '';
        for (var i = 0; i < (d.snapshot.foragers || []).length; i++) {
            var f = d.snapshot.foragers[i];
            var sc = d.scores.PerForager[i] || {};
            var bar = bucketBar(
                Object.keys(f.cargo_by_bucket || {}).map(function(b){ return {bucket:b, current:f.cargo_by_bucket[b]}; }),
                f.cargo_capacity || 1
            );
            foragerHtml += '<tr><td>' + f.name + '</td>' +
                           '<td>' + fmtScore(sc.Score, sc.HasScore) + '</td>' +
                           '<td>' + (f.territory || '—') + '</td>' +
                           '<td>' + f.state + '</td>' +
                           '<td>round +' + (d.snapshot.round - f.state_entered_round) + '</td>' +
                           '<td>' + f.room_id + '</td>' +
                           '<td>' + bar + ' ' + f.cargo_weight + '/' + f.cargo_capacity + ' lbs' + '</td>' +
                           '<td colspan="5" class="text-muted">cycles: see scoring</td></tr>';
        }
        document.getElementById('tbl-foragers').innerHTML = foragerHtml || '<tr><td colspan="12" class="text-muted">No foragers active.</td></tr>';
    }

    window.fetchData = function() {
        fetch('/admin/api/economy/')
            .then(function(r){ return r.json(); })
            .then(render)
            .catch(function(err){ console.error(err); });
    };

    window.toggleAuto = function() {
        autoOn = !autoOn;
        var btn = document.getElementById('btn-auto');
        if (autoOn) {
            btn.textContent = 'Auto: ON';
            btn.classList.remove('btn-outline-secondary');
            btn.classList.add('btn-success');
            startTimer();
        } else {
            btn.textContent = 'Auto: OFF';
            btn.classList.remove('btn-success');
            btn.classList.add('btn-outline-secondary');
            stopTimer();
        }
    };

    window.updateInterval = function() {
        if (autoOn) { stopTimer(); startTimer(); }
    };

    function startTimer() {
        var secs = parseInt(document.getElementById('sel-interval').value, 10);
        autoTimer = setInterval(fetchData, secs * 1000);
    }
    function stopTimer() { if (autoTimer) { clearInterval(autoTimer); autoTimer = null; } }

    window.doSnapshot = function() {
        var label = document.getElementById('manual-label').value || '';
        fetch('/admin/api/economy/snapshot?label=' + encodeURIComponent(label), { method: 'POST' })
            .then(function(r){ return r.json(); })
            .then(function(d){
                alert('Snapshot saved: ' + d.unix_ts);
                fetchData();
            })
            .catch(function(err){ alert('Snapshot failed: ' + err); });
    };

    fetchData();
})();
</script>

{{template "footer" .}}
```

- [ ] **Step 2: Boot the server, hit the dashboard manually**

Run: `go run main.go &` then in a browser go to `http://localhost:<port>/admin/economy/`. The default admin port follows the existing convention — confirm it from `_datafiles/config.yaml`'s `WebPort`.

Expected: page renders. Five score cards show "—" initially (no history). Shop table populated with current shops. Per-discipline rollup populated with one row per discipline that has shops backing it.

`pkill -f "go run main.go"` to stop.

- [ ] **Step 3: Commit**

```bash
git add _datafiles/html/admin/economy/index.html
git commit -m "feat(web/admin): economy dashboard HTML template

Renders five summary cards, per-discipline rollup, per-shop detail with
bucket-stacked stock bars, caravan and forager tables. Auto-refresh
30s/60s/2m. Manual snapshot button.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 18: Smoke test + runbook + patch notes

**Files:**
- Create: `docs/economy/dashboard-runbook.md`
- Modify: `PATCH_NOTES.md`

- [ ] **Step 1: Force a snapshot via the manual button**

Boot server, hit `/admin/economy/`, type a label like "smoke-test", click "Snapshot Now". Confirm:
- Alert shows the unix_ts.
- Visit `_datafiles/economy/snapshots/` on disk and confirm the file exists.
- Read the YAML and confirm shops/caravans/foragers populated as expected.

- [ ] **Step 2: Write the runbook**

Create `docs/economy/dashboard-runbook.md`:

```markdown
# Economy Health Dashboard Runbook

URL: `/admin/economy/` (basic auth — same credentials as Combat Stats).

## What it shows

- **Five score cards** at the top: Economy (overall), Shops, Caravans,
  Foragers, plus the most recent snapshot timestamp and a "Snapshot Now"
  button. Scores are 0-100, colored red <40 / yellow 40-70 / green >70.
- **Per-discipline rollup** of shops grouped by `craft_support:` tag —
  one row each for blacksmithing, alchemy, tailoring, cooking,
  jewelcrafting, enchanting, and general. Then per-shop detail with
  stock bars colored by supply bucket (blue=base, cyan=stillwater,
  orange=thornwall, green=fernway, gray=overlap). The discipline
  rollup is the answer to "is the supply chain supporting player
  crafting?" — a low score on `alchemy` means alchemists are
  starving for materials.
- **Caravan + forager tables** with state, time-in-state, cargo bar,
  per-instance score, and cycle counters.

## How snapshots work

The server writes a snapshot to `_datafiles/economy/snapshots/{unix_ts}.yaml`
every hour (config: `Balance.EconomySnapshotIntervalHours`). The
dashboard's "now" data is captured live on each fetch — disk snapshots
are only used for delta columns and cycle counting.

Auto-snapshots are pruned past `Balance.EconomySnapshotRetentionDays`
(default 30). Manual snapshots (the "Snapshot Now" button) are never
pruned and are useful for "before/after" comparisons across config
changes.

## Troubleshooting

- **Scores show "—":** insufficient history. Caravan/forager scores
  need at least 24 snapshot entries (one full day). Shop scores need
  no history.
- **Empty caravan/forager tables:** confirm the entities are alive.
  Caravan leader is identified by `BTreeState["caravan_state"] != ""`,
  forager by `BTreeState["forager_state"] != ""`.
- **Shop discipline shows "(uncategorized)":** the startup validator
  should have prevented this. If it slipped through, add `craft_support:`
  to the mob YAML and restart.

- **Server panics at boot with "shops.ValidateShopMobTags failed":**
  the panic message lists every shop-bearing mob missing its
  `craft_support:` tag. Add the tag to each listed mob YAML and
  restart. Use `general` for mixed-stock merchants, otherwise pick
  the matching crafting discipline.

## Adding a new vendor discipline

The `ValidCraftSupports` list mirrors the player crafting skills in
`internal/skills/skills.go`. If a new player-facing crafting skill is
added there:

1. Add a matching constant to `internal/shops/shopinventory.go`
   (`CraftSupport<Foo>`) and append to `ValidCraftSupports`.
2. Tag the relevant mobs with `craft_support: foo`.
3. Restart server — dashboard rolls up the new discipline automatically.

If you just want to subdivide an existing discipline (e.g. split
`tailoring` into `cloth` + `leather`), prefer keeping the existing
tag and using the per-shop bucket bar to read the difference.
```

- [ ] **Step 3: Update PATCH_NOTES.md**

Edit `PATCH_NOTES.md`. Add a new dated entry at the top (or under the next prod-cut bucket — match the existing format):

```markdown
### 2026-05-XX (development)

- **Economy Health Dashboard:** new `/admin/economy/` web dashboard
  surfaces shopkeeper, caravan, and forager inventory health. Hourly
  snapshots persist to disk for delta comparisons at 1h / 6h / 1d /
  3d / 1w. Per-shop, per-discipline (blacksmithing/alchemy/tailoring/
  cooking/jewelcrafting/enchanting/general), per-caravan, and
  per-forager scores plus an overall Economy Health score. Manual "Snapshot Now"
  for ad-hoc before/after checks.
```

(Substitute the actual date when shipping.)

- [ ] **Step 4: Commit**

```bash
git add docs/economy/dashboard-runbook.md PATCH_NOTES.md
git commit -m "docs: economy dashboard runbook + patch notes entry

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Verification before declaring done

- [ ] `go build ./...` clean.
- [ ] `go test ./internal/economy/health/... ./internal/shops/... ./internal/caravan/... ./internal/behaviortree/...` PASS.
- [ ] Server boots, sees no panics, dashboard renders at `/admin/economy/`.
- [ ] Snapshot Now writes a file. Disk file matches what the dashboard shows.
- [ ] Sidebar entry highlights when on the economy page.
- [ ] `_datafiles/economy/snapshots/` is gitignored.
