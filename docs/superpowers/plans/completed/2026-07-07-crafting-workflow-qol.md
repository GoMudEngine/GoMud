# Crafting Workflow QOL Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Smarter `craft` menu (craftable-now default + sectioned list), `craft` auto-pulls missing components from the room's storage when at the station, and a short `stow` command to deposit only crafting components.

**Architecture:** A pure `crafting.PlanStoragePull` helper computes which storage items complete a recipe (reused by both the auto-pull and the menu's craftable-now check). The command layer (`craft.go`, `stow.go`) does the item moves + player text. Ergonomics on existing plumbing — no new data, no save migration.

**Tech Stack:** Go (GoMud). Unit tests (`internal/crafting` has `_test.go`). Boot + in-game (`craft`, `stow`) verification.

**Spec:** `docs/superpowers/specs/2026-07-07-crafting-workflow-qol-design.md`

---

## Verified code facts (2026-07-07)

- **Ingredient matching** (`internal/crafting/crafting.go`): `HasIngredients(inv, componentInv, recipe) (bool, firstMissingTag)` counts `componentTagOf(item)` across `componentInv`+`inv` vs `recipe.Ingredients` (each `{ItemTag string, Quantity int}`). `ConsumeIngredients(inv, componentInv, recipe)` removes exactly the needed qty (component bag first). `componentTagOf` is **unexported** in the crafting package — the pull helper must live here.
- **Craft resolver** (`internal/actions/craft.go` `InitiateCraft(actor, recipeName)`, :66): Actor-generic (mobs use it too — NO storage access). Gate order: AlreadyCrafting → RecipeNotFound → RecipeNotKnown → SkillTooLow → **WrongStation** (`recipe.Station != "" && room.Station != recipe.Station`, :104) → **MissingIngredients** (`HasIngredients(char.Items, char.ComponentItems, recipe)`, :111) → CheckOwnComponents → consume. So the auto-pull must run in the COMMAND layer BEFORE `InitiateCraft`.
- **Craft command** (`internal/usercommands/craft.go`): `Craft` — bare/`list` → `craftList` (:221, groups known recipes by skill, annotates via `recipeStatus`); a recipe name → `actions.InitiateCraft(actor, rest)` (:32) then a `switch` on result codes. Helpers: `recipeStatus(user, room, r, skillLevel) (indicator, reason)`, `ingredientSummary(r)`, `craftTimeDesc`, `titleCase`.
- **Storage** (`internal/usercommands/storage.go`, gated `room.IsStorage` :20): `storage add all` (:101) deposits `GetAllBackpackItems()`+`ComponentItems` via `storageAddQuiet(user, itm)` with a `user.ItemStorage.SlotCount() >= storageCap` guard. `storageCap` from `room.StorageCapacity`.
- **ItemStorage** (`internal/users/storage.go`): `Storage` with `GetItems() []items.Item`, `GetSlots() []StorageSlot`, `FindItem(name)`, `AddItem(i) bool`, `RemoveItem(i) bool`, `SlotCount()`.
- **Component predicate**: `item.GetSpec().IsComponent` (`internal/items/itemspec.go:316`). Component-bag routing = `Character.ComponentItems` (all components); backpack components have `IsComponent` true.
- **Command registration**: `internal/usercommands/usercommands.go` command map. `stow`/`stock`/`mats` are unused (free). `deposit` is taken by the gold bank.
- **`Character.StoreItem(item)`** routes a component to the component bag (used by storage retrieval).

---

## Task 1: `stow` command — deposit only crafting components

**Files:**
- Create: `internal/usercommands/stow.go`
- Modify: `internal/usercommands/usercommands.go` (register `stow`)
- Create: `_datafiles/world/dogmud/templates/help/stow.template`

- [ ] **Step 1: Implement `Stow`.** Mirror the `storage add all` block (`storage.go:101`) but deposit ONLY components: every `user.Character.ComponentItems` plus backpack items where `item.GetSpec().IsComponent`. Reuse `storageAddQuiet` (same package) + the `storageCap` capacity guard.

```go
package usercommands

import (
	"fmt"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Stow deposits all crafting components (component bag + is_component backpack
// items) into the room's storage, leaving gear/consumables/quest items behind.
// Shorthand for the component-only case of `storage add all`.
func Stow(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	if !room.IsStorage {
		user.SendText(messaging.CategorySystem, `There is nowhere to stow anything here.`)
		return true, nil
	}
	storageCap := room.StorageCapacity
	if storageCap <= 0 {
		storageCap = 20
	}

	comps := append([]items.Item{}, user.Character.ComponentItems...)
	for _, itm := range user.Character.GetAllBackpackItems() {
		if itm.GetSpec().IsComponent {
			comps = append(comps, itm)
		}
	}

	stowed := 0
	full := false
	for _, itm := range comps {
		if user.ItemStorage.SlotCount() >= storageCap {
			full = true
			break
		}
		if storageAddQuiet(user, itm) {
			stowed++
		}
	}
	switch {
	case stowed == 0 && full:
		user.SendText(messaging.CategorySystem, `Storage is full.`)
	case stowed == 0:
		user.SendText(messaging.CategorySystem, `You have no crafting components to stow.`)
	case full:
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You stow %d component(s); storage filled up before the rest.`, stowed))
	default:
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You stow %d crafting component(s).`, stowed))
	}
	return true, nil
}
```

Verify `storageAddQuiet` / `GetAllBackpackItems` signatures by reading `storage.go` + `inventory.go`; adjust if they differ.

- [ ] **Step 2: Register `stow`.** In `usercommands.go`, add to the command map next to `storage` (match the tuple shape used there — e.g. `{Stow, false, true, false}`; confirm the field meaning from neighbors).

- [ ] **Step 3: Help template.** Create `stow.template` (80-col, player-facing): what `stow` does (dump all crafting mats into storage here), where it works (a storage room), and note `storage` for the full/selective controls. Run the help-completeness test.

- [ ] **Step 4: Build + verify the help test.**

Run: `go build -o gomud_smoke.exe . && go test ./internal/usercommands/ -run TestHelpFile 2>&1 | tail -3`
Expected: BUILT + PASS.

- [ ] **Step 5: Commit.**

```bash
git add internal/usercommands/stow.go internal/usercommands/usercommands.go _datafiles/world/dogmud/templates/help/stow.template
git commit -m "feat(craft-qol): 'stow' command deposits only crafting components to storage"
```

---

## Task 2: `crafting.PlanStoragePull` — pure helper

**Files:**
- Modify: `internal/crafting/crafting.go`
- Test: `internal/crafting/crafting_storagepull_test.go` (create)

- [ ] **Step 1: Write the failing test.** Given a recipe needing 3 of tag `iron`, the player holding 1 (pack/bag) and storage holding 2+ → the plan pulls exactly 2 iron items and reports complete; if storage holds only 1 → not complete and pulls nothing.

```go
func TestPlanStoragePull(t *testing.T) {
	// Build a recipe needing 3 "iron"; seed test item specs so componentTagOf
	// returns "iron" for the test items (grep how crafting_test.go seeds tags).
	recipe := &RecipeSpec{Ingredients: []RecipeIngredient{{ItemTag: "iron", Quantity: 3}}}
	onHandComp := []items.Item{ironItem()} // 1 on hand
	storage := []items.Item{ironItem(), ironItem(), ironItem()} // 3 in storage

	pull, complete := PlanStoragePull(recipe, nil, onHandComp, storage)
	if !complete || len(pull) != 2 {
		t.Fatalf("expected complete with 2 pulled, got complete=%v pulled=%d", complete, len(pull))
	}

	// Not enough in storage → no pull, not complete.
	pull2, complete2 := PlanStoragePull(recipe, nil, onHandComp, []items.Item{ironItem()})
	if complete2 || len(pull2) != 0 {
		t.Fatalf("expected incomplete + no pull, got complete=%v pulled=%d", complete2, len(pull2))
	}
}
```

(Use the existing crafting test's tag-seeding for `ironItem()` — grep `crafting_test.go` for how `componentTagOf`/`ItemTag` test items are built.)

- [ ] **Step 2: Run — expect FAIL.**

Run: `go test ./internal/crafting/ -run TestPlanStoragePull`
Expected: FAIL (undefined).

- [ ] **Step 3: Implement `PlanStoragePull`.**

```go
// PlanStoragePull computes which storage items would complete recipe's
// ingredients given what the actor already holds (inv = backpack, componentInv
// = component bag). Returns the exact storage items to pull and whether pulling
// them makes the recipe craftable. All-or-nothing: if storage can't cover the
// full shortfall, returns (nil, false) — pull nothing.
func PlanStoragePull(recipe *RecipeSpec, inv, componentInv, storage []items.Item) ([]items.Item, bool) {
	// shortfall per tag = needed - on hand (bag + pack)
	shortfall := make(map[string]int)
	for _, ing := range recipe.Ingredients {
		shortfall[ing.ItemTag] = ing.Quantity
	}
	for _, it := range componentInv {
		if t := componentTagOf(it); shortfall[t] > 0 {
			shortfall[t]--
		}
	}
	for _, it := range inv {
		if t := componentTagOf(it); shortfall[t] > 0 {
			shortfall[t]--
		}
	}

	pull := []items.Item{}
	for _, it := range storage {
		t := componentTagOf(it)
		if shortfall[t] > 0 {
			pull = append(pull, it)
			shortfall[t]--
		}
	}
	for _, remaining := range shortfall {
		if remaining > 0 {
			return nil, false // storage can't complete it — pull nothing
		}
	}
	return pull, true
}
```

- [ ] **Step 4: Run — expect PASS** (+ package).

Run: `go test ./internal/crafting/`
Expected: PASS.

- [ ] **Step 5: Commit.**

```bash
git add internal/crafting/crafting.go internal/crafting/crafting_storagepull_test.go
git commit -m "feat(craft-qol): crafting.PlanStoragePull computes storage items to complete a recipe"
```

---

## Task 3: Auto-pull components from storage in `craft`

**Files:**
- Modify: `internal/usercommands/craft.go`

- [ ] **Step 1: Add the pull before `InitiateCraft`.** In `Craft`, after `recipe := crafting.FindRecipeByName(rest)` resolves (and it's a normal, not enchanting, recipe), and BEFORE `actions.InitiateCraft`, run the storage pull when eligible:

```go
// Auto-pull from storage: if we're at a storage room that satisfies the
// recipe's station and we're short on components, draw exactly what's needed
// from storage so the craft can proceed. All-or-nothing (PlanStoragePull).
if recipe != nil && room.IsStorage &&
	(recipe.Station == "" || room.Station == recipe.Station) &&
	user.Character.HasRecipe(recipe.RecipeId) {

	if ok, _ := crafting.HasIngredients(user.Character.Items, user.Character.ComponentItems, recipe); !ok {
		pull, complete := crafting.PlanStoragePull(recipe, user.Character.Items, user.Character.ComponentItems, user.ItemStorage.GetItems())
		if complete {
			for _, itm := range pull {
				if user.ItemStorage.RemoveItem(itm) {
					user.Character.StoreItem(itm) // routes components to the component bag
					user.SendText(messaging.CategoryLoot, fmt.Sprintf(`You draw <ansi fg="item">%s</ansi> from storage.`, itm.DisplayName()))
				}
			}
		}
	}
}
```

Place this so it runs for the recipe-name path only (not `craft list`), and after the enchanting-recipe routing (enchanting has its own `craftEnchanting` path — don't double-handle). Then the existing `actions.InitiateCraft(actor, rest)` sees the pulled components and proceeds; if the pull didn't complete it, `InitiateCraft` returns `MissingIngredients` as before.

- [ ] **Step 2: Build + manual/harness verify.** At a room that is both storage and the right station, with the components in storage (not pack): `craft <recipe>` draws them from storage and completes; `look`/`storage` confirm the components left storage. Away from storage, behavior is unchanged.

Run: `go build -o gomud_smoke.exe . && echo BUILT` then drive it (mudagent; admin at a storage+station room — e.g. set up a test room, or use an existing bank/workshop that IsStorage + has a station).
Expected: pull message + successful craft.

- [ ] **Step 3: Commit.**

```bash
git add internal/usercommands/craft.go
git commit -m "feat(craft-qol): craft auto-pulls missing components from storage at the station"
```

---

## Task 4: Smarter craft menu — craftable-now + sectioned list

**Files:**
- Modify: `internal/usercommands/craft.go` (`craftList` + a bare-vs-list split)

- [ ] **Step 1: Split bare `craft` from `craft list`.** In `Craft`, route no-args to a new `craftCraftableNow(user, room)` and `list`/`all` to the sectioned `craftList`. (Currently both call `craftList` — read the dispatch at the top of `Craft`.)

- [ ] **Step 2: A craftability classifier.** Add a helper that classifies a known recipe for the current room into one of: `ready` (skill OK + station satisfied + ingredients on hand **or** completable from storage when `room.IsStorage`), `missing` (skill + station OK but lacking mats even from storage), `locked` (wrong/absent station or skill too low). Reuse `recipeStatus`/`HasIngredients` + `PlanStoragePull`:

```go
func classifyRecipe(user *users.UserRecord, room *rooms.Room, r *crafting.RecipeSpec) string {
	lvl := user.Character.GetSkillLevel(skills.SkillTag(r.Skill))
	if lvl < r.SkillMinimum {
		return "locked"
	}
	if r.Station != "" && room.Station != r.Station {
		return "locked"
	}
	if ok, _ := crafting.HasIngredients(user.Character.Items, user.Character.ComponentItems, r); ok {
		return "ready"
	}
	if room.IsStorage {
		if _, complete := crafting.PlanStoragePull(r, user.Character.Items, user.Character.ComponentItems, user.ItemStorage.GetItems()); complete {
			return "ready"
		}
	}
	return "missing"
}
```

- [ ] **Step 3: Bare `craft` (`craftCraftableNow`).** List only `ready` known recipes (station-aware via the classifier), formatted like today's rows minus the red-`[X]` noise. Empty state: `Nothing you can craft here right now — try` + `craft list`.

- [ ] **Step 4: Sectioned `craft list`.** Reorganize `craftList` into three labeled sections in order — **Ready to craft** (green), **Missing ingredients** (with the missing tags), **Locked** (with the station/skill reason) — using `classifyRecipe`. Keep the existing per-recipe annotation (`ingredientSummary`, station, `craftTimeDesc`); within a section, keep the skill grouping if it reads well, else flat. Preserve the known/total completion counts somewhere (header or per-skill).

- [ ] **Step 5: Build + manual verify** each: bare `craft` at a station with mats → shows only makeable; `craft list` → three sections; at a storage+station room, a recipe whose mats are only in storage shows under **Ready**.

Run: `go build -o gomud_smoke.exe . && go vet ./internal/usercommands/ && echo OK`
Expected: OK + the in-game views.

- [ ] **Step 6: Commit.**

```bash
git add internal/usercommands/craft.go
git commit -m "feat(craft-qol): bare craft = craftable-now (station+storage aware); craft list sectioned Ready/Missing/Locked"
```

---

## Task 5: Discoverability + full verification + memory

**Files:**
- Modify: storage/bank help text (`storage`/`bank` help templates or the command's hint output)
- Modify: memory

- [ ] **Step 1: Surface `stow`.** Mention `stow` in the `storage` command's help/list header and in the storage/bank room's command hints (grep how `storage` advertises its subcommands + where a bank/storage room lists actions). Keep it one line: "`stow` — deposit all crafting components here."

- [ ] **Step 2: Full suite + clean boot.**

Run: `go test ./... 2>&1 | grep -vE "^ok|no test files" | head` (no new failures) then
`rm -rf _datafiles/world/dogmud/mobs.instances/* && go build -o gomud_smoke.exe .`, boot, confirm no panic + recipes/spells load.

- [ ] **Step 3: In-game matrix** (mudagent, admin at a storage+station room — set up or find one; a crafting-district workshop that IsStorage, or temporarily flag a room):
  - `stow` deposits only components (gear stays in pack).
  - `craft <recipe>` with mats only in storage → "You draw … from storage" → craft completes; storage decremented.
  - bare `craft` shows only makeable recipes (incl. storage-completable ones); at no station, shows station-free recipes only.
  - `craft list` shows the three sections.
  - `craft <recipe>` away from storage still errors with "You are missing: …".

- [ ] **Step 4: Update memory.** Mark [[project_crafting_qol_backlog]] BUILT (the `stow` command, PlanStoragePull, auto-pull, sectioned menu); note pre-push SOP owed.

- [ ] **Step 5: Commit.**

```bash
git add _datafiles/world/dogmud/templates/help/ internal/usercommands/
git commit -m "docs(craft-qol): surface stow in storage/bank help + verification"
```

---

## Notes for the executor

- **Auto-pull is command-layer only** — `InitiateCraft` is Actor-generic (mobs share it) and must not gain storage access; do the pull in `craft.go` before calling it.
- **All-or-nothing pull**: `PlanStoragePull` returns `(nil, false)` when storage can't fully cover the shortfall, so nothing is pulled on a doomed craft.
- **Reuse, don't reimplement**: `PlanStoragePull` (Task 2) is the single source of truth for "completable from storage," used by both the auto-pull (Task 3) and the menu classifier (Task 4).
- **`componentTagOf` is unexported** → the pull-planning logic lives in the `crafting` package, not the command layer.
- **Don't touch enchanting** — it has its own `craftEnchanting` path; the auto-pull + classifier are for normal recipes (enchanting targets an equipped item, not component ingredients).
