# Cooking Supply Chain Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make town cooks craft meals from forager-supplied ingredients — corpse salvage yields meat, foragers deliver it, the global chest backfill spreads forageables, and the three cooks become crafters that turn ingredients into the meals players buy.

**Architecture:** Mostly data (salvage table, forage table, economy bucket, mob/shop YAML) plus two small Go changes (globalize the 5.4 chest backfill; extend the salvage yield table). The forager hunt→salvage/forage→deliver pipe (chunks 3.8/5.4) already runs and already visits all three cooks' rooms.

**Tech Stack:** Go (tests: `testing` + `testify/assert`). Data: YAML. Build `go build ./...`; test `go test ./internal/<pkg>/...`. Boot smoke per CLAUDE.md SOP (wipe instance saves, `go run .`, watch for `InputWorker ... Started` and no panic).

**Spec:** `docs/superpowers/specs/2026-06-02-cooking-supply-chain-design.md`

---

## Verified facts (confirmed before writing — trust over memory)

- **Corpse salvage table:** `internal/crafting/corpse_salvage.go` — `var corpseSalvageTable []corpseSalvageEntry{Group string; Returns []items.SalvageReturn}`. `LookupCorpseSalvage(groups []string)` returns the **first** table entry whose `Group` is in the mob's groups. Item tags must be valid `component_tag`s: raw-meat=40014, wild-hare-meat=40064, leather-strip=40002, sinew=40068, cloth-strip=40007. Test file: `corpse_salvage_test.go`.
- **Economy buckets:** `internal/economy/buckets.go` — a `map[int]string` literal; `40064: "fernway", // wild hare meat`. `BucketFor(itemId)` reads it.
- **Forage yields:** `internal/forager/forage_core.go:29` — `var ForageYields = map[string][]int{ "forest": {40004,40004,40005,40005,40049,40049,40067}, ... }`. shadowcap=40063, blood-moss=40066 are NOT in any biome list. Test: `forage_core_test.go`.
- **Chest backfill (5.4, to globalize):** `internal/forager/chest_backfill.go` — `BackfillVendorFromChests(vendorMob, shopInv)` calls `chestPoolForZone(vendorMob.Zone)` (zone-scoped — the bug). `chestPoolForZone(zone)` uses `ChestRoomsForZone(zone)` + the `loadRoomFn` seam. Chest index: `internal/forager/chest_index.go` — `var chestIndex map[string]map[int]bool` (zone→roomset), `ChestRoomsForZone(zone) []int` (sorted). `selectBackfillTransfers(si, pool)` is pure (neediest-gap-first, vendor-stocks-it, MaxStock cap). Tests: `chest_backfill_test.go`, `chest_index_test.go`.
- **Crafter system:** Mob struct (`internal/mobs/mobs.go:138-141`): `Crafter bool` (`crafter`), `CrafterSkill string` (`crafterskill`), `CrafterRecipeIds []string` (`crafterrecipeids`), `CrafterRestockMaterials []int` (`crafterrestockmaterials`). `cooking` is a valid `skills.SkillTag` (`internal/skills/skills.go:40`). `TickMobCraft` (`crafter.go:195`) → `EvaluateCraftOptions` (`craftdecision.go:141`): crafts when output entry is absent OR `Current < MaxStock`, all ingredients present (≥ `CrafterIngredientReservePct` reserve), and profit > 0. **Cook needs a `skills: cooking: N` character entry** (else `GetSkillLevel`=0 fails recipe `skill_minimum`). `CrafterRestockMaterials` items get auto-managed stock entries (RestockQty 3 / MaxStock 10 per `crafter.go:38`). Reference crafter: `_datafiles/world/dogmud/mobs/stillwater/338-apothecary_ilsa.yaml`.
- **Cooking recipes** (`_datafiles/world/dogmud/recipes/cooking/`), recipe-id → output → inputs:
  - `grilled-meat` → 30022 ← raw-meat, salt-pouch
  - `trail-rations` → 30021 ← raw-meat, salt-pouch
  - `hearty-stew` → 30023 ← raw-meat, wild-hare-meat, wild-vegetables, water-flask
  - `herbal-tea` → 30024 ← healers-root, shadowcap, water-flask
  - `stillwater-lake-chowder` → 30061 ← freshwater-clam×3, wild-vegetables, lake-mint, blood-moss, water-flask
  - `antidote-broth` → 30027 ← wild-vegetables, healers-root, water-flask
  - `energy-bread` → 30026 ← wild-vegetables×2, water-flask, salt-pouch
  - `spiced-wine` → 30025 ← wild-vegetables, water-flask, salt-pouch
- **The three cooks** (all `behavior_archetype: noncombat_shopkeeper`, no crafter fields yet):
  | Cook | Zone | mob YAML | shop file | stocks (40xxx ingredients) | stocks (30xxx meals) |
  |---|---|---|---|---|---|
  | 336 Tov Brann | Stillwater | `mobs/stillwater/336-fishmonger_tov_brann.yaml` | `shops/stillwater/336-room4102.yaml` | 40058 clam, 40051 shrimp-shell, 40014 raw-meat, 40017 salt | none |
  | 103 food vendor | Thornwall | `mobs/thornwall_city/103-food_vendor.yaml` | `shops/thornwall_city/103-room464.yaml` | 40058, 40063 shadowcap, 40064 hare-meat, 40014, 40015 veg | 30022,30023,30024,30021 |
  | 248 tavern cook Brynn | Thornwall | `mobs/thornwall_city/248-tavern_cook_brynn.yaml` | `shops/thornwall_city/248-room481.yaml` | 40057 mint,40058,40063,40064,40066 blood-moss,40004 healers-root,40014,40015,40016 water,40017 salt | none |
- **Prey mob groups** — confirm in Task 1 by reading `_datafiles/world/dogmud/mobs/**/360-*.yaml` (wild hare) and `367-*.yaml` (marsh rat): wild hare is `[animal, rodent, prey]`; marsh rat carries 40064 (mis-tag to clean up).

---

## File structure

| File | Change |
|------|--------|
| `internal/crafting/corpse_salvage.go` (+ test) | add raw-meat / wild-hare-meat yields; rodent entry ordered before animal |
| `internal/economy/buckets.go` (+ test if present) | 40064 `fernway` → `overlap` |
| `internal/forager/chest_index.go` (+ test) | add `ChestRoomsAll()` |
| `internal/forager/chest_backfill.go` (+ test) | global pool; `BackfillVendorFromChests` uses it |
| `internal/forager/forage_core.go` (+ test) | add 40063, 40066 to `forest` yields |
| 3 cook mob YAMLs | crafter fields + `skills: cooking` |
| 3 cook shop YAMLs | cooked-meal entries → RestockQty 0 (+ add missing meal entries) |
| `_datafiles/world/dogmud/mobs/**/367-*.yaml` | marsh-rat salvage cleanup (Task 1) |
| context.md (crafting, forager) + roadmap/memory | docs |

---

## Task 1: Salvage yields meat

**Files:** Modify `internal/crafting/corpse_salvage.go`; Test `internal/crafting/corpse_salvage_test.go`.

- [ ] **Step 1: Confirm prey group tags**

Read `_datafiles/world/dogmud/mobs/**/360-*.yaml` (wild hare), `367-*.yaml` (marsh rat), and 1-2 other forager prey (Tova prey 367/368; Kessa/Tova hare 360). Note each mob's `groups:` list. Decide the small-game key: use a tag the hares carry that generic animals lack (expected `rodent`). Record which prey have which groups (you'll need this so the ordering in Step 3 maps hares→hare-meat and other animals→raw-meat). If marsh rat (367) has `rodent`, it would match the hare-meat entry — decide its yield (see Step 5 cleanup).

- [ ] **Step 2: Write the failing test**

In `internal/crafting/corpse_salvage_test.go` add:

```go
func TestLookupCorpseSalvage_AnimalYieldsRawMeat(t *testing.T) {
	got := LookupCorpseSalvage([]string{"animal", "predator"})
	tags := map[string]int{}
	for _, r := range got {
		tags[r.ItemTag] = r.Quantity
	}
	if tags["raw-meat"] < 1 {
		t.Errorf("animal corpse should yield raw-meat, got %v", got)
	}
}

func TestLookupCorpseSalvage_SmallGameYieldsHareMeat(t *testing.T) {
	// rodent listed before animal → small game yields hare meat, not raw-meat
	got := LookupCorpseSalvage([]string{"animal", "rodent", "prey"})
	tags := map[string]int{}
	for _, r := range got {
		tags[r.ItemTag] = r.Quantity
	}
	if tags["wild-hare-meat"] < 1 {
		t.Errorf("small-game corpse should yield wild-hare-meat, got %v", got)
	}
}
```

(If Step 1 found a different small-game tag than `rodent`, use that tag in the second test.)

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/crafting/ -run TestLookupCorpseSalvage -v`
Expected: FAIL (raw-meat / wild-hare-meat not yielded yet).

- [ ] **Step 4: Implement the yield additions**

Replace `corpseSalvageTable` in `internal/crafting/corpse_salvage.go` (rodent BEFORE animal so the first-match returns hare meat for small game):

```go
var corpseSalvageTable = []corpseSalvageEntry{
	{Group: "rodent", Returns: []items.SalvageReturn{ // small game → hare meat
		{ItemTag: "wild-hare-meat", Quantity: 1},
		{ItemTag: "leather-strip", Quantity: 1},
	}},
	{Group: "animal", Returns: []items.SalvageReturn{ // generic game → raw meat
		{ItemTag: "raw-meat", Quantity: 1},
		{ItemTag: "leather-strip", Quantity: 2},
		{ItemTag: "sinew", Quantity: 1},
	}},
	{Group: "humanoid", Returns: []items.SalvageReturn{ // unchanged
		{ItemTag: "cloth-strip", Quantity: 2},
		{ItemTag: "leather-strip", Quantity: 1},
	}},
}
```

Use the actual small-game tag from Step 1 if not `rodent`.

- [ ] **Step 5: Marsh-rat content cleanup**

If Step 1 found marsh rat (367) has the small-game tag (so it would yield hare meat — wrong, rats aren't hares), and/or 367 carries item 40064 as a carried item: edit `367-*.yaml` so its salvage/loot yields generic `raw-meat` instead — simplest is to ensure 367's `groups` does NOT include the small-game tag (so it falls through to `animal`→raw-meat), and remove a mis-tagged `40064` carried item if present. Keep this minimal; if 367 already maps cleanly to `animal`, skip.

- [ ] **Step 6: Run tests + build**

Run: `go test ./internal/crafting/ -run TestLookupCorpseSalvage -v && go build ./...`
Expected: PASS, clean build.

- [ ] **Step 7: Commit**

```bash
git add internal/crafting/corpse_salvage.go internal/crafting/corpse_salvage_test.go
git commit -m "feat(cooking): corpse salvage yields raw-meat + wild-hare-meat"
```
(Include the 367 YAML in the add if you edited it.)

---

## Task 2: Re-bucket wild-hare-meat to overlap

**Files:** Modify `internal/economy/buckets.go`; Test `internal/economy/buckets_test.go` (if present).

- [ ] **Step 1: Check for other dependencies on the fernway bucket**

Run: `grep -rn "40064\|wild-hare-meat" internal/ _datafiles/world/dogmud/caravans/` — confirm nothing relies on 40064 being `fernway` (caravan cargo lists, forager throughput). If a caravan cargo references it, note it (re-bucketing only changes forager direct-delivery eligibility; caravan cargo is separate).

- [ ] **Step 2: Write/adjust the failing test**

If `internal/economy/buckets_test.go` exists, add:

```go
func TestBucketFor_WildHareMeatOverlap(t *testing.T) {
	if got := BucketFor(40064); got != "overlap" {
		t.Errorf("wild-hare-meat bucket = %q, want overlap", got)
	}
}
```

If no buckets test file exists, create `internal/economy/buckets_test.go` with `package economy` + this test.

- [ ] **Step 3: Run to verify it fails**

Run: `go test ./internal/economy/ -run TestBucketFor_WildHareMeatOverlap -v`
Expected: FAIL (currently `fernway`).

- [ ] **Step 4: Change the bucket**

In `internal/economy/buckets.go`, change the line `40064: "fernway", // wild hare meat (3.0b)` to:
```go
	40064: "overlap", // wild hare meat — forager-deliverable to cooks (cooking chunk)
```

- [ ] **Step 5: Run + build + commit**

Run: `go test ./internal/economy/ -run TestBucketFor_WildHareMeatOverlap -v && go build ./...`
Expected: PASS, clean.
```bash
git add internal/economy/buckets.go internal/economy/buckets_test.go
git commit -m "feat(cooking): re-bucket wild-hare-meat fernway->overlap for forager delivery"
```

---

## Task 3: Globalize the chest backfill (5.4 correction)

**Files:** Modify `internal/forager/chest_index.go`, `internal/forager/chest_backfill.go`; Test both `_test.go`.

- [ ] **Step 1: Write the failing cross-zone test**

In `internal/forager/chest_backfill_test.go` add (uses the `loadRoomFn` seam + a chest registered in a DIFFERENT zone than the vendor):

```go
func TestBackfill_GlobalCrossZone(t *testing.T) {
	const chestZone, chestRoom = "zoneA-5.4cook", 51001
	RegisterChestRoom(chestZone, chestRoom)

	orig := loadRoomFn
	defer func() { loadRoomFn = orig }()
	loadRoomFn = func(id int) *rooms.Room {
		if id != chestRoom {
			return nil
		}
		return &rooms.Room{Containers: map[string]rooms.Container{
			"lockbox": {Items: []items.Item{{ItemId: 40063}, {ItemId: 40063}}},
		}}
	}

	// pool aggregated globally must see the chest even though its zone != "zoneB"
	pool, rooms2 := chestPoolAll()
	if pool[40063] != 2 {
		t.Fatalf("global pool should aggregate cross-zone chest, got %v", pool)
	}
	if len(rooms2) == 0 {
		t.Fatalf("expected chest room in global list")
	}
}
```

Add `github.com/GoMudEngine/GoMud/internal/{items,rooms}` to test imports if missing. (Scope the zone name uniquely — the index is a package global.)

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/forager/ -run TestBackfill_GlobalCrossZone -v`
Expected: FAIL (`chestPoolAll` undefined).

- [ ] **Step 3: Add `ChestRoomsAll` to the index**

In `internal/forager/chest_index.go` add (after `ChestRoomsForZone`):

```go
// ChestRoomsAll returns every registered chest room across all zones, stable-
// sorted and deduped. Used by the global chest backfill so forager chests are
// aggregated as one group (not per-zone).
func ChestRoomsAll() []int {
	chestIndexMu.RLock()
	defer chestIndexMu.RUnlock()
	seen := map[int]bool{}
	var out []int
	for _, set := range chestIndex {
		for id := range set {
			if !seen[id] {
				seen[id] = true
				out = append(out, id)
			}
		}
	}
	sort.Ints(out)
	return out
}
```

(`sort` is already imported in `chest_index.go`.)

- [ ] **Step 4: Add `chestPoolAll` and switch the backfill to it**

In `internal/forager/chest_backfill.go`, add `chestPoolAll` next to `chestPoolForZone` (factor the shared body into a helper to stay DRY):

```go
// chestPoolFromRooms aggregates item counts across the given chest rooms.
func chestPoolFromRooms(chestRooms []int) (pool map[int]int, nonEmpty []int) {
	pool = map[int]int{}
	for _, chestRoom := range chestRooms {
		room := loadRoomFn(chestRoom)
		if room == nil {
			continue
		}
		key := room.FindContainerByName("lockbox")
		if key == "" {
			continue
		}
		c := room.Containers[key]
		empty := true
		for _, it := range c.Items {
			pool[it.ItemId]++
			empty = false
		}
		if !empty {
			nonEmpty = append(nonEmpty, chestRoom)
		}
	}
	return pool, nonEmpty
}

func chestPoolForZone(zone string) (map[int]int, []int) { return chestPoolFromRooms(ChestRoomsForZone(zone)) }
func chestPoolAll() (map[int]int, []int)               { return chestPoolFromRooms(ChestRoomsAll()) }
```

(Replace the old standalone `chestPoolForZone` body with the delegating one above.)

Then in `BackfillVendorFromChests`, change the pool source from zone to global:
```go
	pool, chestRooms := chestPoolAll()   // was: chestPoolForZone(vendorMob.Zone)
```
Leave the rest (`selectBackfillTransfers`, the transfer loop, `LastGrewRound` stamp, `SaveShop`) unchanged.

- [ ] **Step 5: Run tests**

Run: `go test ./internal/forager/ -run 'TestBackfill|TestSelectBackfillTransfers|TestChestPoolForZone|TestChestIndex' -v && go build ./...`
Expected: PASS (existing per-zone pool tests still pass — `chestPoolForZone` still works; the new global test passes).

- [ ] **Step 6: Update context + commit**

Edit `internal/forager/context.md`: change the backfill description from "in its zone" to "across all forager chests (global aggregation)".
```bash
git add internal/forager/chest_index.go internal/forager/chest_backfill.go internal/forager/chest_backfill_test.go internal/forager/context.md
git commit -m "fix(5.4): aggregate chest backfill across all forager chests (global, not per-zone)"
```

---

## Task 4: Forage-yield additions (shadowcap + blood-moss)

**Files:** Modify `internal/forager/forage_core.go`; Test `internal/forager/forage_core_test.go`.

- [ ] **Step 1: Write the failing test**

In `internal/forager/forage_core_test.go` add:

```go
func TestForageYields_ForestHasCookingFlora(t *testing.T) {
	forest := ForageYields["forest"]
	has := func(id int) bool {
		for _, x := range forest {
			if x == id {
				return true
			}
		}
		return false
	}
	if !has(40063) {
		t.Error("forest forage should include shadowcap (40063)")
	}
	if !has(40066) {
		t.Error("forest forage should include blood-moss (40066)")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/forager/ -run TestForageYields_ForestHasCookingFlora -v`
Expected: FAIL.

- [ ] **Step 3: Add the items to the forest yield**

In `internal/forager/forage_core.go`, change the `"forest"` line of `ForageYields` to append 40063 + 40066:
```go
	"forest":    {40004, 40004, 40005, 40005, 40049, 40049, 40067, 40063, 40066}, // +40063 shadowcap, +40066 blood-moss (cooking chunk)
```

- [ ] **Step 4: Run + build**

Run: `go test ./internal/forager/ -run TestForageYields -v && go build ./...`
Expected: PASS. (If an existing forage_core_test asserts an exact forest set, update it to include the two new ids.)

- [ ] **Step 5: Commit**

```bash
git add internal/forager/forage_core.go internal/forager/forage_core_test.go
git commit -m "feat(cooking): forage yields shadowcap + blood-moss (forest); reach cooks via global backfill"
```

---

## Task 5: Convert the three cooks to crafters

This is the core fix. Per cook: add crafter fields + a `skills: cooking` entry to the mob YAML, ensure every chosen recipe's ingredients are available (existing stock OR `crafterrestockmaterials`), and set cooked-meal stock entries to `RestockQty: 0` in the shop file (adding entries for meals not yet stocked). No unit test (crafting requires a running server); verified by boot + in-game smoke.

**Files:** Modify the three cook mob YAMLs + three shop YAMLs (paths in Verified facts).

- [ ] **Step 1: Determine recipe skill_minimum**

Read the 8 cooking recipe YAMLs and note each recipe's `skill_minimum` (or equivalent gate field). Set each cook's `skills: cooking: N` to **N = max(skill_minimum over that cook's recipes) + a margin** (if all are 0, use 10 to be safe; mirror Ilsa's `alchemy: 12`). Record the chosen N per cook.

- [ ] **Step 2: Convert 336 Tov Brann (Stillwater) — grilled-meat + trail-rations**

He stocks raw-meat (40014) + salt (40017) → can make grilled-meat (30022) + trail-rations (30021). In `mobs/stillwater/336-fishmonger_tov_brann.yaml`, add (preserve all existing fields; mirror Ilsa's structure):
```yaml
crafter: true
crafterskill: cooking
crafterrecipeids:
  - grilled-meat
  - trail-rations
crafterrestockmaterials:
  - 40017   # salt-pouch (staple; auto-stocked)
```
And in the `character:` `skills:` block add `cooking: <N>`. (raw-meat already has a RestockQty-5 shop entry as the floor; foragers top it up — do NOT add 40014 to crafterrestockmaterials so it stays forager/cart-sourced.)

In `shops/stillwater/336-room4102.yaml`, ADD cooked-meal stock entries (so the meals have a cap + appear in lists), RestockQty 0:
```yaml
- item_id: 30022
  restock_qty: 0
  max_stock: 20
  current: 0
- item_id: 30021
  restock_qty: 0
  max_stock: 20
  current: 0
```

- [ ] **Step 3: Convert 103 food vendor (Thornwall) — its 4 meals**

It stocks meals 30022/30023/30024/30021 + ingredients raw-meat/veg/hare-meat/shadowcap/clam, but LACKS salt (40017), water (40016), healers-root (40004). In `mobs/thornwall_city/103-food_vendor.yaml` add:
```yaml
crafter: true
crafterskill: cooking
crafterrecipeids:
  - grilled-meat
  - trail-rations
  - hearty-stew
  - herbal-tea
crafterrestockmaterials:
  - 40017   # salt-pouch
  - 40016   # water-flask
  - 40004   # healers-root
  - 40015   # wild-vegetables (ensure present)
```
Add `cooking: <N>` to its `skills:` block. (raw-meat 40014, hare-meat 40064, shadowcap 40063 keep their existing RestockQty-5 floor + forager/backfill top-up — not in crafterrestockmaterials.)

In `shops/thornwall_city/103-room464.yaml`, set the four cooked-meal entries (30022, 30023, 30024, 30021) to `restock_qty: 0` (leave their max_stock; keep current as-is).

- [ ] **Step 4: Convert 248 tavern cook Brynn (Thornwall) — full menu**

She stocks nearly every ingredient (mint, clam, shadowcap, hare-meat, blood-moss, healers-root, raw-meat, veg, water, salt) but NO cooked-meal entries. In `mobs/thornwall_city/248-tavern_cook_brynn.yaml` add:
```yaml
crafter: true
crafterskill: cooking
crafterrecipeids:
  - hearty-stew
  - herbal-tea
  - stillwater-lake-chowder
  - antidote-broth
  - energy-bread
  - spiced-wine
crafterrestockmaterials: []   # she already stocks all needed ingredients (RestockQty 5)
```
(If any chosen recipe needs an ingredient she lacks, add that id to `crafterrestockmaterials` instead. Verify against her stock + the recipe inputs in Verified facts. `crafterrestockmaterials: []` is valid; omit the key if empty per YAML style.)
Add `cooking: <N>` to her `skills:` block.

In `shops/thornwall_city/248-room481.yaml`, ADD cooked-meal stock entries (RestockQty 0, max_stock 20, current 0) for each meal she crafts: 30023, 30024, 30061, 30027, 30026, 30025.

- [ ] **Step 5: Build + boot smoke (clean instances)**

Run: `go build ./...`
Then (CLAUDE.md SOP — shop files are persistent state and are NOT wiped, only instances):
```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
```
Boot the server; confirm `InputWorker ... Started`, no panic, and **no crafter-config validation panic** for the three cooks (watch for messages about unknown recipe ids / invalid crafter skill). Confirm `mobs.LoadDataFiles loadedCount` unchanged (228).

- [ ] **Step 6: Commit**

```bash
git add _datafiles/world/dogmud/mobs/stillwater/336-fishmonger_tov_brann.yaml _datafiles/world/dogmud/mobs/thornwall_city/103-food_vendor.yaml _datafiles/world/dogmud/mobs/thornwall_city/248-tavern_cook_brynn.yaml _datafiles/world/dogmud/shops/stillwater/336-room4102.yaml _datafiles/world/dogmud/shops/thornwall_city/103-room464.yaml _datafiles/world/dogmud/shops/thornwall_city/248-room481.yaml
git commit -m "feat(cooking): convert Tov Brann/food vendor/Brynn to cooking crafters"
```

---

## Task 6: Docs, full verification, roadmap/memory

**Files:** context.md (crafting), `MOB_ALIVENESS_ROADMAP.md` or memory.

- [ ] **Step 1: Docs**

- `internal/crafting/context.md`: note corpse salvage now yields raw-meat (animal) + wild-hare-meat (small game), feeding cooking.
- (forager context.md already updated in Task 3.)

- [ ] **Step 2: Full build + test sweep**

Run: `go build ./... && go test ./...`
Expected: all packages PASS. Classify any failure as cooking-chunk-related vs pre-existing; do not fix unrelated breakage.

- [ ] **Step 3: Boot smoke (clean instances)**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
```
Boot; confirm `InputWorker ... Started`, no panic.

- [ ] **Step 4: Roadmap/memory note**

Note the cooking half of the economy-fix split done; log follow-ups: (a) the enchanting half (decayed potions → enchanter), (b) the Fernway caravan empty `deliveries_by_tier`. Update [[project_store_restock_considered_fix]] to mark cooking addressed (enchanting + general still open).

- [ ] **Step 5: Commit**

```bash
git add internal/crafting/context.md MOB_ALIVENESS_ROADMAP.md
git commit -m "docs(cooking): context + roadmap for cooking supply chain"
```

---

## In-game smoke checklist (deferred to user, per precedent)

- [ ] A forager salvages a prey kill → gains raw-meat (generic animal) / wild-hare-meat (hare); a hunted hare yields hare meat, a generic animal yields raw-meat.
- [ ] Forager delivers meat to a cook (cook's raw-meat / hare-meat stock rises after a forager visit — Tova→336, Halix→103/248).
- [ ] Each cook crafts meals over time (cooked-meal stock rises from 0); players can buy the meals at all three cooks (including Stillwater's 336 now).
- [ ] A forager-gathered shadowcap/blood-moss reaches a Thornwall cook (103/248) via the **global** chest backfill (cross-zone) and the cook crafts herbal-tea / chowder.
- [ ] New-player cooking grind isn't more expensive (raw-meat value unchanged at 1).
- [ ] Watch that `RestockQty: 0` meals don't sit permanently empty (ingredient supply keeps up); if a meal starves, raise that ingredient's floor or add it to the cook's `crafterrestockmaterials`.

---

## Self-review notes (flagged for the implementer)

- **`skills: cooking: N` is load-bearing** (Task 5 Step 1): without a cooking skill entry the cook's `GetSkillLevel` is 0 and recipes with `skill_minimum > 0` never craft. Confirm each recipe's gate field name + value when setting N.
- **Crafted-output stock entry for cooks that don't stock meals** (336, 248): `EvaluateCraftOptions` crafts even with no output entry, but adding explicit `RestockQty: 0` entries (Task 5) gives the meal a defined `max_stock` and makes it appear in `list`. Confirm the meal item ids (30025/30026/30027/30061) exist as item specs.
- **`crafterrestockmaterials` vs forager supply:** living-economy ingredients (raw-meat, hare-meat, shadowcap, blood-moss) are intentionally LEFT OUT of `crafterrestockmaterials` so they depend on forager/cart supply, not crafter auto-stock. Mundane staples (salt/water/veg/healers-root) go in `crafterrestockmaterials` (auto-stocked at RestockQty 3). If smoke shows a cook starving on a living ingredient, its existing RestockQty-5 cart entry is the floor — confirm that floor exists in the shop file.
- **Global backfill is a 5.4 correction** (Task 3) and lands here because the cooking forageables depend on it; the per-zone `chestPoolForZone` is kept (still used by nothing else now, but harmless) — or inline it if the reviewer prefers removing the dead path.
