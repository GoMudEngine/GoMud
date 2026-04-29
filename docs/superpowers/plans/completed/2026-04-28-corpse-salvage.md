# Corpse Salvage Implementation Plan (Stage 3.0e)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend the existing `salvage` command to accept room-resident corpses, recover cloth/leather/sinew based on the mob's `groups`, and consume the corpse on success. Adds `sinew` (40068) as a new mat with two recipe homes.

**Architecture:** Reuse the existing `CraftingState` activity machinery and `resolveSalvage` hook. Add a new `salvage-corpse:<mobId>` `RecipeId` prefix so the round-tick dispatcher routes corpse resolution to a new `resolveCorpseSalvage` function. A small static table in `internal/crafting/corpse_salvage.go` maps mob `groups` to salvage returns; `LookupCorpseSalvage(groups)` keeps the API extensible for future creature types. The corpse is removed from the room on completion (matches existing tagged-item salvage that fully consumes the target).

**Tech Stack:** Go (server). YAML data files. Existing systems: `crafting`, `items`, `mobs`, `rooms`, `characters.CraftingState`, `skills.Salvage`.

**Spec:** `docs/superpowers/specs/2026-04-28-corpse-salvage-design.md`

---

## File Structure

| Action | File | Responsibility |
|---|---|---|
| CREATE | `_datafiles/world/dogmud/items/materials-40000/40068-sinew.yaml` | New animal-tendon mat |
| CREATE | `internal/crafting/corpse_salvage.go` | Group → salvage returns table + `LookupCorpseSalvage(groups)` helper |
| CREATE | `internal/crafting/corpse_salvage_test.go` | Unit tests for the table lookup |
| MODIFY | `internal/usercommands/salvage.go` | After `FindItem` fails, fall through to `room.FindCorpse`; validate via `LookupCorpseSalvage`; start a `salvage-corpse:<mobId>` activity |
| MODIFY | `internal/hooks/NewRound_UserRoundTick.go` | Add `salvage-corpse:` prefix routing in the crafting dispatcher; new `resolveCorpseSalvage` function that rolls returns, removes the corpse, and grants materials |
| MODIFY | `_datafiles/world/dogmud/recipes/tailoring/artisans-satchel.yaml` | Add 1× sinew to ingredients |
| MODIFY | `_datafiles/world/dogmud/recipes/blacksmithing/lake-iron-hook-spear.yaml` | Add 1× sinew to ingredients |
| MODIFY | `docs/economy/mat-audit-matrix.md` | Add 40068 sinew row; reclassify 40002 leather strip and 40007 cloth strip from "Defer to 3.0e" to "Mid-tier overlap (corpse-salvage sourced)" |
| MODIFY | `docs/schemas/mob.md` | One-line note on `groups` row that it drives corpse salvage |
| MODIFY | `PATCH_NOTES.md` | Stage 3.0e dev-only entry |

---

## Task 1: Add sinew material YAML

**Files:**
- Create: `_datafiles/world/dogmud/items/materials-40000/40068-sinew.yaml`

- [ ] **Step 1: Confirm 40068 is unused**

Run: `ls _datafiles/world/dogmud/items/materials-40000/40068*.yaml 2>/dev/null`
Expected: no output (file does not exist).

- [ ] **Step 2: Create the file**

```yaml
itemid: 40068
name: sinew
namesimple: sinew
description: A length of dried tendon stripped clean from an
  animal carcass and stretched between two pegs to cure. Tough
  enough to bind a haft against generations of use; supple
  enough to draw a hunting bow without splintering. Sewers
  reach for it when a needle and thread won't hold the seam
  through a hard winter.
type: object
subtype: mundane
component_tag: sinew
weight: 0.05
value: 25
is_component: true
```

- [ ] **Step 3: Verify the build still compiles**

Run: `go build ./...`
Expected: clean build, no errors.

- [ ] **Step 4: Boot test the loader**

Run: `go run . 2>&1 | grep -E "items.LoadDataFiles|panic" | head -10`
Expected: a line `items.LoadDataFiles() loadedCount=N` where N is one greater than before. No panic. Kill the server (Ctrl-C) once you see the line.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/items/materials-40000/40068-sinew.yaml
git commit -m "$(cat <<'EOF'
feat(items): add sinew (40068) for corpse salvage

Animal-tendon mat sourced from corpse salvage on animal-group mobs.
Mid-low value (25g) reflects the salvage gating. Stage 3.0e.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Create corpse_salvage table + lookup helper + tests

**Files:**
- Create: `internal/crafting/corpse_salvage.go`
- Create: `internal/crafting/corpse_salvage_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/crafting/corpse_salvage_test.go`:

```go
package crafting

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

func TestLookupCorpseSalvage_Animal(t *testing.T) {
	got := LookupCorpseSalvage([]string{"animal", "canine", "predator"})
	want := []items.SalvageReturn{
		{ItemTag: "leather-strip", Quantity: 2},
		{ItemTag: "sinew", Quantity: 1},
	}
	if !equalReturns(got, want) {
		t.Errorf("animal: got %+v, want %+v", got, want)
	}
}

func TestLookupCorpseSalvage_Humanoid(t *testing.T) {
	got := LookupCorpseSalvage([]string{"bandit", "humanoid"})
	want := []items.SalvageReturn{
		{ItemTag: "cloth-strip", Quantity: 2},
		{ItemTag: "leather-strip", Quantity: 1},
	}
	if !equalReturns(got, want) {
		t.Errorf("humanoid: got %+v, want %+v", got, want)
	}
}

func TestLookupCorpseSalvage_NoMatch(t *testing.T) {
	got := LookupCorpseSalvage([]string{"chrysalis", "elemental"})
	if got != nil {
		t.Errorf("no-match: got %+v, want nil", got)
	}
}

func TestLookupCorpseSalvage_EmptyGroups(t *testing.T) {
	got := LookupCorpseSalvage(nil)
	if got != nil {
		t.Errorf("nil groups: got %+v, want nil", got)
	}
	got = LookupCorpseSalvage([]string{})
	if got != nil {
		t.Errorf("empty groups: got %+v, want nil", got)
	}
}

func TestLookupCorpseSalvage_FirstTableEntryWins(t *testing.T) {
	// animal appears before humanoid in the table — if a mob
	// somehow has both, animal wins.
	got := LookupCorpseSalvage([]string{"humanoid", "animal"})
	want := []items.SalvageReturn{
		{ItemTag: "leather-strip", Quantity: 2},
		{ItemTag: "sinew", Quantity: 1},
	}
	if !equalReturns(got, want) {
		t.Errorf("multi-group: got %+v, want %+v", got, want)
	}
}

func equalReturns(a, b []items.SalvageReturn) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].ItemTag != b[i].ItemTag || a[i].Quantity != b[i].Quantity {
			return false
		}
	}
	return true
}
```

- [ ] **Step 2: Run the test and confirm it fails**

Run: `go test ./internal/crafting/ -run TestLookupCorpseSalvage -v`
Expected: FAIL with `undefined: LookupCorpseSalvage`.

- [ ] **Step 3: Create the corpse_salvage.go file**

Create `internal/crafting/corpse_salvage.go`:

```go
package crafting

import (
	"github.com/GoMudEngine/GoMud/internal/items"
)

// corpseSalvageEntry pairs a mob group key with the salvage returns
// players recover when they salvage a corpse from that group.
type corpseSalvageEntry struct {
	Group   string
	Returns []items.SalvageReturn
}

// corpseSalvageTable is the static lookup table for corpse salvage.
// Order matters: LookupCorpseSalvage returns the first matching entry.
// Future expansion (bird, insect, chrysalis, etc.) just appends here.
var corpseSalvageTable = []corpseSalvageEntry{
	{
		Group: "animal",
		Returns: []items.SalvageReturn{
			{ItemTag: "leather-strip", Quantity: 2},
			{ItemTag: "sinew", Quantity: 1},
		},
	},
	{
		Group: "humanoid",
		Returns: []items.SalvageReturn{
			{ItemTag: "cloth-strip", Quantity: 2},
			{ItemTag: "leather-strip", Quantity: 1},
		},
	},
}

// LookupCorpseSalvage returns the salvage returns for the first matching
// group in the table, or nil if no group matches. The mob's full groups
// slice is passed in; iteration order is the table's declaration order.
func LookupCorpseSalvage(groups []string) []items.SalvageReturn {
	for _, entry := range corpseSalvageTable {
		for _, g := range groups {
			if g == entry.Group {
				return entry.Returns
			}
		}
	}
	return nil
}
```

- [ ] **Step 4: Run the tests and confirm they pass**

Run: `go test ./internal/crafting/ -run TestLookupCorpseSalvage -v`
Expected: all 5 tests PASS.

- [ ] **Step 5: Run the full crafting test suite**

Run: `go test ./internal/crafting/...`
Expected: PASS (no regressions in existing tests).

- [ ] **Step 6: Commit**

```bash
git add internal/crafting/corpse_salvage.go internal/crafting/corpse_salvage_test.go
git commit -m "$(cat <<'EOF'
feat(crafting): add corpse salvage group→returns lookup

Static table mapping mob groups (animal, humanoid) to salvage returns.
LookupCorpseSalvage(groups) returns the first matching entry; the API
shape is designed for future creature types (bird, insect, chrysalis).
Stage 3.0e.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Extend salvage command + resolver to handle corpses

**Files:**
- Modify: `internal/usercommands/salvage.go` (add corpse fall-through after `FindItem`)
- Modify: `internal/hooks/NewRound_UserRoundTick.go` (add `salvage-corpse:` dispatcher branch + `resolveCorpseSalvage` function)

This task touches two files because salvage start (parser) and salvage finish (resolver) live in different packages. Implement both together so the build stays green between commits.

### Part A: Extend `salvage.go` parser

- [ ] **Step 1: Add the corpse fall-through in salvage.go**

In `internal/usercommands/salvage.go`, replace the existing `FindItem` block (lines 33–45) and the imports block. The full updated file is below — diff against current to apply.

Updated imports (top of file, replaces lines 1–15):

```go
package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)
```

REPLACE the existing `FindItem` block (current lines 33–39) with the block below. This is a single replacement — do not insert separately and then delete; that would double-declare `itm`, `source`, `found`.

Old (lines 33–39):
```go
	// Find item in backpack (not equipped — must unequip first)
	itm, source, found := user.Character.FindItem(rest)
	if !found {
		user.SendText(fmt.Sprintf(
			`<ansi fg="red">You don't have "%s".</ansi>`, rest))
		return true, nil
	}
```

New:
```go
	// Try inventory item first; fall through to room corpses if no
	// inventory match.
	itm, source, found := user.Character.FindItem(rest)
	if !found {
		corpse, corpseFound := room.FindCorpse(rest)
		if !corpseFound {
			user.SendText(fmt.Sprintf(
				`<ansi fg="red">You don't have "%s" and there's no corpse of that name here.</ansi>`, rest))
			return true, nil
		}
		return startCorpseSalvage(user, room, corpse)
	}
```

The remaining body (`source != "in your backpack"` check through the end of the function) stays unchanged. `itm` and `source` are still used downstream.

- [ ] **Step 2: Add the `startCorpseSalvage` helper at the bottom of the same file**

Append at the end of `internal/usercommands/salvage.go` (after `userHasSalvageKit`):

```go
// startCorpseSalvage initiates a corpse salvage activity. Called from
// Salvage when the player's argument matches a room corpse rather than
// an inventory item.
func startCorpseSalvage(user *users.UserRecord, room *rooms.Room, corpse rooms.Corpse) (bool, error) {

	// Player corpses are out of scope for v1.
	if corpse.MobId <= 0 {
		user.SendText(`<ansi fg="red">You can't bring yourself to salvage that.</ansi>`)
		return true, nil
	}

	mobSpec := mobs.GetMobSpec(mobs.MobId(corpse.MobId))
	if mobSpec == nil {
		user.SendText(`<ansi fg="red">Something is wrong with that corpse.</ansi>`)
		return true, nil
	}

	returns := crafting.LookupCorpseSalvage(mobSpec.Groups)
	if len(returns) == 0 {
		user.SendText(`<ansi fg="red">There's nothing useful to recover here.</ansi>`)
		return true, nil
	}

	// Salvage kit always required for corpses.
	if !userHasSalvageKit(user) {
		user.SendText(`<ansi fg="red">You need a salvage kit to skin a corpse.</ansi>`)
		return true, nil
	}

	bal := configs.GetBalanceConfig()
	totalGold := crafting.CalcSalvageReturnGoldValue(returns)
	rounds := crafting.CalcSalvageRounds(totalGold,
		int(bal.SalvageGoldPerRound), int(bal.SalvageMaxRounds))

	// Stash corpse identity for the resolver. mobid + roundCreated
	// uniquely identifies the corpse within the room. Store as int to
	// avoid type-assertion issues if MiscData ever round-trips through
	// YAML (uint64 can come back coerced).
	user.Character.SetMiscData("salvage_corpse_mobid", corpse.MobId)
	user.Character.SetMiscData("salvage_corpse_round_created", int(corpse.RoundCreated))
	user.Character.SetMiscData("salvage_corpse_name", corpse.Character.Name)
	user.Character.SetMiscData("salvage_uses_kit", true)

	user.Character.CraftingState = &characters.CraftingState{
		RecipeId:    fmt.Sprintf("salvage-corpse:%d", corpse.MobId),
		RoundsTotal: rounds,
	}

	user.SendText(fmt.Sprintf(
		`<ansi fg="yellow">You begin carefully working over the <ansi fg="mobname">%s corpse</ansi>...</ansi>`,
		corpse.Character.Name))

	return true, nil
}
```

- [ ] **Step 3: Verify the build**

Run: `go build ./...`
Expected: clean build. Fix any unused-import errors before continuing.

### Part B: Add resolver branch + function

- [ ] **Step 4: Update the dispatcher in NewRound_UserRoundTick.go**

In `internal/hooks/NewRound_UserRoundTick.go`, locate the salvage dispatcher block at line 308–318. Replace it with the version below.

Old (lines 307–318):
```go
							progressMsg := cs.RecipeId
							if len(cs.RecipeId) > 8 && cs.RecipeId[:8] == "salvage:" {
								progressMsg = "salvaging"
							}
							user.SendText(fmt.Sprintf(
								`<ansi fg="yellow">You continue working on %s... (%d/%d)</ansi>`,
								progressMsg, cs.RoundsComplete, cs.RoundsTotal))
						} else if len(cs.RecipeId) > 8 && cs.RecipeId[:8] == "salvage:" {
							// Salvage completion
							itemIdStr := cs.RecipeId[8:]
							user.Character.CraftingState = nil
							resolveSalvage(user, itemIdStr)
						} else {
```

New:
```go
							progressMsg := cs.RecipeId
							if strings.HasPrefix(cs.RecipeId, "salvage:") || strings.HasPrefix(cs.RecipeId, "salvage-corpse:") {
								progressMsg = "salvaging"
							}
							user.SendText(fmt.Sprintf(
								`<ansi fg="yellow">You continue working on %s... (%d/%d)</ansi>`,
								progressMsg, cs.RoundsComplete, cs.RoundsTotal))
						} else if strings.HasPrefix(cs.RecipeId, "salvage-corpse:") {
							mobIdStr := strings.TrimPrefix(cs.RecipeId, "salvage-corpse:")
							user.Character.CraftingState = nil
							resolveCorpseSalvage(user, mobIdStr)
						} else if strings.HasPrefix(cs.RecipeId, "salvage:") {
							// Salvage completion
							itemIdStr := strings.TrimPrefix(cs.RecipeId, "salvage:")
							user.Character.CraftingState = nil
							resolveSalvage(user, itemIdStr)
						} else {
```

Note: `strings` is already imported in this file. Verify by checking the existing imports block at the top.

- [ ] **Step 5: Add the `resolveCorpseSalvage` function**

Append at the end of `internal/hooks/NewRound_UserRoundTick.go`, after `resolveSalvage`:

```go
// resolveCorpseSalvage handles corpse salvage completion when CraftingState
// finishes. The corpse is removed from the room on completion (matches
// existing tagged-item salvage that fully consumes its target).
func resolveCorpseSalvage(user *users.UserRecord, mobIdStr string) {
	var mobId int
	fmt.Sscanf(mobIdStr, "%d", &mobId)

	// Pull stashed corpse identity.
	roundCreatedInt, _ := user.Character.GetMiscData("salvage_corpse_round_created").(int)
	corpseName, _ := user.Character.GetMiscData("salvage_corpse_name").(string)
	usesKit, _ := user.Character.GetMiscData("salvage_uses_kit").(bool)
	roundCreated := uint64(roundCreatedInt)
	user.Character.SetMiscData("salvage_corpse_mobid", nil)
	user.Character.SetMiscData("salvage_corpse_round_created", nil)
	user.Character.SetMiscData("salvage_corpse_name", nil)
	user.Character.SetMiscData("salvage_uses_kit", nil)

	// Consume salvage kit (corpses always use the kit).
	if usesKit {
		for _, itm := range user.Character.Items {
			if itm.GetSpec().ComponentTag == "salvage-kit" {
				user.Character.RemoveItem(itm)
				break
			}
		}
	}

	room := rooms.LoadRoom(user.Character.RoomId)
	if room == nil {
		user.SendText(`<ansi fg="red">Something went wrong with your salvage attempt.</ansi>`)
		return
	}

	// Locate the corpse by mobId + roundCreated (unique within a room).
	var target rooms.Corpse
	found := false
	for _, c := range room.Corpses {
		if c.MobId == mobId && c.RoundCreated == roundCreated && !c.Prunable {
			target = c
			found = true
			break
		}
	}
	if !found {
		user.SendText(fmt.Sprintf(
			`<ansi fg="red">The %s corpse is no longer here.</ansi>`, corpseName))
		return
	}

	mobSpec := mobs.GetMobSpec(mobs.MobId(mobId))
	if mobSpec == nil {
		user.SendText(`<ansi fg="red">Something went wrong with your salvage attempt.</ansi>`)
		return
	}

	returns := crafting.LookupCorpseSalvage(mobSpec.Groups)
	if len(returns) == 0 {
		user.SendText(`<ansi fg="red">There's nothing useful to recover here.</ansi>`)
		room.RemoveCorpse(target)
		return
	}

	// Skill chance.
	bal := configs.GetBalanceConfig()
	salvageSkill := user.Character.GetSkillLevel(skills.Salvage)
	chance := crafting.CalcSalvageChance(salvageSkill,
		float64(bal.SalvageMinChance), float64(bal.SalvageMaxChance),
		int(bal.SalvageSoftCap))

	recovered := crafting.RollSalvageReturnsFromSpec(returns, chance)

	// Remove the corpse regardless of roll outcome (matches tagged-item
	// salvage behavior — the activity has cost regardless of result).
	room.RemoveCorpse(target)

	if len(recovered) > 0 {
		var parts []string
		for _, ing := range recovered {
			for i := 0; i < ing.Quantity; i++ {
				matSpec := items.FindSpecByComponentTag(ing.ItemTag)
				if matSpec != nil {
					newItem := items.New(matSpec.ItemId)
					user.Character.StoreItem(newItem)
				}
			}
			parts = append(parts, fmt.Sprintf("%dx %s", ing.Quantity, ing.ItemTag))
		}
		user.SendText(fmt.Sprintf(
			`<ansi fg="green">You finish working over the <ansi fg="mobname">%s corpse</ansi> and recover: %s.</ansi>`,
			corpseName,
			strings.Join(parts, ", ")))
	} else {
		user.SendText(fmt.Sprintf(
			`<ansi fg="red">You work over the <ansi fg="mobname">%s corpse</ansi> but recover nothing useful.</ansi>`,
			corpseName))
	}

	user.Character.OnSkillUse("salvage", user.UserId)
}
```

- [ ] **Step 6: Add the `mobs` import to NewRound_UserRoundTick.go**

The new function references `rooms.LoadRoom`, `rooms.Corpse`, `mobs.GetMobSpec`, `mobs.MobId`. `rooms` and `skills` are already imported but `mobs` is NOT. Add it to the import block:

```go
"github.com/GoMudEngine/GoMud/internal/mobs"
```

Verify with:
```bash
grep -E '"github.com/GoMudEngine/GoMud/internal/mobs"' internal/hooks/NewRound_UserRoundTick.go
```
Expected: one match.

- [ ] **Step 7: Build the whole tree**

Run: `go build ./...`
Expected: clean build, no errors.

- [ ] **Step 8: Run all tests**

Run: `go test ./...`
Expected: all PASS (no regressions).

- [ ] **Step 9: Commit**

```bash
git add internal/usercommands/salvage.go internal/hooks/NewRound_UserRoundTick.go
git commit -m "$(cat <<'EOF'
feat(salvage): allow salvaging corpses for cloth/leather/sinew

`salvage <corpse>` now extends the existing salvage activity to room-
resident corpses. The mob's `groups` field drives the returns table
(animal → leather + sinew, humanoid → cloth + leather). Salvage kit
required. The corpse is consumed on successful completion, mirroring
how tagged-item salvage fully consumes its target.

Stage 3.0e.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Wire sinew into 2 existing recipes

**Files:**
- Modify: `_datafiles/world/dogmud/recipes/tailoring/artisans-satchel.yaml`
- Modify: `_datafiles/world/dogmud/recipes/blacksmithing/lake-iron-hook-spear.yaml`

We add sinew to one tailoring and one blacksmithing recipe — picked because sinew plausibly fits a heavy-duty seam (satchel) and a haft binding (hook-spear). No existing ingredient is removed.

- [ ] **Step 1: Add sinew to artisans-satchel.yaml**

In `_datafiles/world/dogmud/recipes/tailoring/artisans-satchel.yaml`, append a new ingredient under the `ingredients:` list. The full updated file:

```yaml
id: artisans-satchel
name: Artisan's Satchel
skill: tailoring
skill_minimum: 15
station: loom
time_rounds: 5
ingredients:
  - item_tag: leather-strip
    quantity: 2
  - item_tag: cloth-strip
    quantity: 1
  - item_tag: thread-spool
    quantity: 2
  - item_tag: bone-needle
    quantity: 1
  - item_tag: sinew
    quantity: 1
output:
  item_id: 30034
  quantity: 1
success_message: "You stitch padded dividers into a leather satchel and fit a brass clasp, lashing the seams with sinew for a hard-wearing finish. An artisan's satchel!"
failure_message: "The dividers won't sit straight and the clasp warps. The materials are wasted."
```

- [ ] **Step 2: Add sinew to lake-iron-hook-spear.yaml**

In `_datafiles/world/dogmud/recipes/blacksmithing/lake-iron-hook-spear.yaml`, append a new ingredient. The full updated file:

```yaml
id: lake-iron-hook-spear
name: Lake-Iron Hook-Spear
skill: blacksmithing
skill_minimum: 22
station: forge
time_rounds: 5
ingredients:
  - item_tag: lake-iron-nodule
    quantity: 3
  - item_tag: iron-ingot
    quantity: 1
  - item_tag: leather-strip
    quantity: 1
  - item_tag: sinew
    quantity: 1
output:
  item_id: 10031
  quantity: 1
success_message: "You smelt the lake-iron nodules in a hot fire and draw the metal out long and keen, then weld a back-curving hook to the socket. You wrap the haft in leather and lash it tight with sinew. The leaf-blade rings true on the anvil. A lake-iron hook-spear!"
failure_message: "The lake-iron will not weld evenly to the socket. The hook tears free under the first hammer-blow and the work is wasted."
```

- [ ] **Step 3: Verify recipe loader picks up the changes**

Run: `go build ./...`
Expected: clean build (recipes are YAML-loaded at runtime, not compiled).

- [ ] **Step 4: Boot test the recipe loader**

Run: `go run . 2>&1 | grep -E "recipes.LoadDataFiles|panic" | head -5`
Expected: a line `recipes.LoadDataFiles() loadedCount=N` (count unchanged from baseline — we're editing existing recipes, not adding). No panic. Kill the server.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/recipes/tailoring/artisans-satchel.yaml _datafiles/world/dogmud/recipes/blacksmithing/lake-iron-hook-spear.yaml
git commit -m "$(cat <<'EOF'
feat(recipes): require sinew in artisans satchel + lake-iron hook-spear

Wires demand for sinew (the new corpse-salvage mat) across 2 craft
schools. Tailoring: heavy-duty seam binding on the satchel. Smithing:
haft lashing on the hook-spear. Stage 3.0e.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Update audit matrix + schema docs + PATCH_NOTES

**Files:**
- Modify: `docs/economy/mat-audit-matrix.md`
- Modify: `docs/schemas/mob.md`
- Modify: `PATCH_NOTES.md`

- [ ] **Step 1: Update mat-audit-matrix.md**

Make three edits to `docs/economy/mat-audit-matrix.md`:

(a) Reclassify 40002 leather strip — change the row at line 35 to:
```
| 40002 | leather strip | Mid-tier overlap | corpse-salvage sourced | Reclassified by 3.0e: dropped by salvaging animal- and humanoid-group corpses |
```

(b) Reclassify 40007 cloth strip — change the row at line 40 to:
```
| 40007 | cloth strip | Mid-tier overlap | corpse-salvage sourced | Reclassified by 3.0e: dropped by salvaging humanoid-group corpses |
```

(c) Add a new row for 40068 sinew — append after the 40067 pine pitch row:
```
| **40068** | **sinew** | **Mid-tier overlap** | **corpse-salvage sourced** | **NEW (Stage 3.0e); dropped by salvaging animal-group corpses; demand wired into tailoring + blacksmithing** |
```

(d) Update the "Bucket summary" table:
- "Mid-tier overlap" count: 8 → 11; append `40002, 40007, 40068` to the IDs list (sorted in numeric position)
- "Defer to 3.0e" count: 4 → 2; remove `40002, 40007` from the IDs list (leaving 40052, 40055)

(e) Update the row-count comment below the table:
```
> Row count: 68 total. 40001-40067 = 67 existing mats; 40068 added by
> Stage 3.0e (corpse salvage). All 68 rows appear in the audit table
> above, including 15 quest/specialty items that are out of the supply
> pipeline.
```

- [ ] **Step 2: Update mob.md schema doc**

In `docs/schemas/mob.md`, change the `groups` row (line 54) from:
```
| `groups` | list | no | Group membership (e.g. `[rats, animal]`). Used for teamwork and hates logic. |
```

to:
```
| `groups` | list | no | Group membership (e.g. `[rats, animal]`). Used for teamwork and hates logic, and drives corpse salvage returns (see `internal/crafting/corpse_salvage.go`). |
```

- [ ] **Step 3: Update PATCH_NOTES.md**

Insert a new top entry above the existing 2026-04-28 Stage 3.0b entry:

```markdown
## 2026-04-28 — Stage 3.0e: Corpse Salvage (dev only)

**Note:** Dev-only landing. The full economy stack ships to prod (`master`)
as a coherent update once Stage 3.4 lands.

- `salvage <corpse>` now works on room-resident corpses, not just
  inventory items. Animal-group mobs yield leather strip + sinew;
  humanoid-group mobs yield cloth strip + leather strip. Each material
  rolls independently against the salvage skill curve. Salvage kit
  required (sold by Fence Dealer Siv, 1g).
- The corpse is consumed on completion (mirrors tagged-item salvage
  behavior — the activity has cost regardless of roll outcome). If the
  activity is interrupted (combat, movement) the corpse stays untouched.
- Added **sinew** (40068), a tough animal-tendon mat sourced from
  corpse salvage on animals. Wired into 2 existing recipes: tailoring's
  Artisan's Satchel (heavy-duty seam binding) and blacksmithing's
  Lake-Iron Hook-Spear (haft lashing).
- 40002 leather strip and 40007 cloth strip reclassified in the audit
  matrix from "Defer to 3.0e" → "Mid-tier overlap (corpse-salvage
  sourced)". Source pipeline now decided. Vendor inventories continue
  to NOT stock these mats — corpse salvage is the v1 source.

```

- [ ] **Step 4: Verify everything builds + boots**

Run: `go build ./...`
Expected: clean build.

Run: `go run . 2>&1 | head -50`
Expected: server passes the data-file load phase without panic. Kill the server.

- [ ] **Step 5: Commit**

```bash
git add docs/economy/mat-audit-matrix.md docs/schemas/mob.md PATCH_NOTES.md
git commit -m "$(cat <<'EOF'
docs(3.0e): audit matrix + schema + patch notes for corpse salvage

- Mat audit matrix: sinew (40068) added; leather strip + cloth strip
  reclassified from Defer-to-3.0e to Mid-tier overlap
  (corpse-salvage sourced). Bucket summary updated.
- Mob schema: note that `groups` drives corpse salvage returns.
- Patch notes: Stage 3.0e dev-only entry.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: In-game smoke verification

**Files:** none modified — manual playtest. If any smoke step fails, stop and fix the underlying issue (likely in T3) before claiming done.

- [ ] **Step 1: Boot the server**

Run: `go run .`
Expected: server reaches the prompt without panic. Note the loaded counts:
- `items.LoadDataFiles() loadedCount=N` — N should be one greater than baseline (sinew added)
- `recipes.LoadDataFiles() loadedCount=M` — M unchanged (97 expected)
- `mobs.LoadDataFiles() loadedCount=K` — unchanged

- [ ] **Step 2: Test animal corpse salvage**

In a connected client:
1. Travel to a zone with animal-group mobs (e.g., Ashwick — wild dogs, timber wolves: mob 264).
2. Kill one (`attack wolf`). Confirm a corpse appears in the room (`look`).
3. Confirm you have a salvage kit (`buy salvage kit` from Fence Dealer Siv if not).
4. Run `salvage corpse` (or `salvage wolf`).
5. Expected: yellow message "You begin carefully working over the <wolf> corpse...", then a multi-round activity.
6. On completion: green message recovering some leather-strip and possibly sinew. The corpse disappears from the room (`look` to confirm).

- [ ] **Step 3: Test humanoid corpse salvage**

1. Travel to dustwalk_road. Kill a bandit (mob 80).
2. Run `salvage corpse`.
3. Expected: recovers cloth-strip and possibly leather-strip. Corpse disappears.

- [ ] **Step 4: Test no-match corpse**

1. Find a mob whose `groups` contain neither `animal` nor `humanoid` (e.g., a chrysalis-touched creature in Thornwall). Kill it.
2. Run `salvage corpse`.
3. Expected: red message "There's nothing useful to recover here." Activity does NOT start; corpse stays in the room.

- [ ] **Step 5: Test interruption**

1. Kill an animal mob.
2. Run `salvage corpse`.
3. Before the activity completes, run `north` (or any movement).
4. Expected: salvage cancels (existing crafting cancellation message). Walk back; corpse is still in the room (NOT consumed).

- [ ] **Step 6: Test missing salvage kit**

1. Drop your salvage kit (`drop kit`).
2. Kill an animal mob.
3. Run `salvage corpse`.
4. Expected: red message "You need a salvage kit to skin a corpse." No activity starts.

- [ ] **Step 7: Test recipe wiring**

1. Pick up the salvage kit again.
2. Confirm `recipes` lists Artisan's Satchel and Lake-Iron Hook-Spear with sinew in the ingredient list.
3. Gather all ingredients (use `give` from a creative session if testing as admin).
4. Craft each recipe at the appropriate station (loom for satchel, forge for hook-spear). Both should succeed.

- [ ] **Step 8: Test skill progression**

1. Run `status` (or `skills`) — note your salvage skill level.
2. Salvage 5–10 corpses.
3. Re-check skill — confirm it has a chance of progressing (existing OnSkillUse path; not deterministic but should fire occasionally).

- [ ] **Step 9: Mark task complete**

If all 8 smoke steps pass, T6 is done. No commit (manual test only).

If any step fails: stop, capture the failure, and fix in the relevant earlier task. Do not claim completion until all steps pass.

---

## Out of scope reminder (from spec)

These are explicitly NOT part of this plan — push back if scope creep tries to add them:
- Per-mob `salvage_returns` overrides
- Bird/insect/elemental drop tables (option C)
- New cloth/leather/sinew recipes invented (only existing recipe edits)
- Re-adding cloth/leather slots to caravan-served vendor inventories
- Tanning/curing tiers
- Skinning as a separate skill
- Corpse drag/move mechanics
- Salvage failure refund

## Done = ?

All 6 tasks complete, all commits landed on `development` branch, smoke test (T6) all green. Per the multi-stage caravan/economy effort: this lands on `development` only. Nothing ships to `master` until Stage 3.4 lands.
