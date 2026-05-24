# Mob Aliveness 2.9 — Mob Forage + Salvage Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Lift forage and salvage into the actor pattern (`actions.Forage`, single-tick `actions.Salvage`). Add btree primitives `try_forage`, `try_salvage`, `wander_territory`, and condition `forager_state_is_foraging`. Refactor `forager_step` to dissolve only the Foraging-state per-tick loop into YAML, keeping multi-state daily cycle in Go. Migrate three forager mobs (Tova, Halix, Kessa) from per-mob behavior YAMLs to a shared `forager` archetype.

**Architecture:** Three actions in `internal/actions/` via the actor pattern (Forage, Salvage single-tick). Thin player + mob wrappers. The player-side multi-round salvage activity keeps its initiator (`usercommands/salvage.go` untouched) but the per-tick resolve in `hooks/NewRound_UserRoundTick.go` is refactored to call `actions.Salvage`. New shared `forager` archetype YAML composes `try_forage` + `try_salvage` + `wander_territory` for the Foraging-state per-tick loop; `forager_step` handles the multi-state daily cycle for the other five states.

**Tech Stack:** Go, GoMud engine, YAML data files, TDD with `go test`.

---

## Spec discrepancies (resolved in this plan)

1. **`usercommands/salvage.go` is the INITIATOR only, not the per-tick caller.** The spec's wording "usercommands/salvage.go keeps its multi-round CraftingState scheduling; each tick calls actions.Salvage" is misleading — the per-tick logic actually lives in `internal/hooks/NewRound_UserRoundTick.go`'s `resolveSalvageFromData` and `resolveCorpseSalvage`. **This plan refactors those hook functions, not `usercommands/salvage.go`.**

2. **Salvage has no cooldown today.** The spec proposed a 2-round cooldown on the "salvage" key, but neither player nor mob salvage paths currently enforce one (Activity machine throttles the player; mob path is single-tick by design). This plan drops the cooldown to preserve existing behavior. `SalvageOptions` does not include a cooldown field. `SalvageResult.OnCooldown` is omitted.

## Implementation order rationale

1. **Actions package lifts first (Tasks 1-2):** TDD per file. Source-of-truth code for the two verbs. Self-contained.
2. **Player + hook refactors (Tasks 3-4):** Thin player wrapper for forage; refactor the per-tick salvage resolve in hooks.
3. **Mob wrappers (Task 5):** Tiny.
4. **Btree primitives (Tasks 6-7):** Three actions + one condition + registrations.
5. **Forager state machine refactor (Task 8):** Trickiest — surgery on `actForagerStep` + deletion of `tickForagerForaging`.
6. **Archetype + migration (Tasks 9-10):** YAML authoring + mob YAML flips + per-mob behavior YAML deletions.
7. **Docs + smoke (Tasks 11-12):** context.md updates + boot/smoke/roadmap.

## Task table of contents

| # | Task | Files | Deps |
|---|------|-------|------|
| 1 | Lift Forage into `internal/actions/` (TDD) | actions/forage.go + test | — |
| 2 | Lift Salvage single-tick core into `internal/actions/` (TDD) | actions/salvage.go + test | — |
| 3 | Thin `usercommands/skill.forage.go` | usercommands/skill.forage.go | 1 |
| 4 | Refactor hooks/NewRound_UserRoundTick.go to call actions.Salvage | hooks/NewRound_UserRoundTick.go | 2 |
| 5 | Add mob wrappers + register `forage` | mobcommands/{forage,salvage}.go + mobcommands.go | 1, 2 |
| 6 | Add btree primitives `try_forage` / `try_salvage` / `wander_territory` + condition `forager_state_is_foraging` | behaviortree/actions_forager_verbs.go + conditions_forager.go | 1, 2 |
| 7 | Register btree primitives in init() | behaviortree/actions.go + conditions.go | 6 |
| 8 | Refactor `forager_step` (Foraging-state guards at top, delete tick handler) | behaviortree/actions_forager.go | 7 |
| 9 | Author `forager.yaml` archetype | behaviors/archetypes/forager.yaml | 8 |
| 10 | Migrate 3 forager mobs + delete 3 per-mob behavior YAMLs | 6 YAML files | 9 |
| 11 | Update context.md files | behaviortree + actions context.md | 8, 9 |
| 12 | Boot + smoke + mark roadmap done | server boot, MOB_ALIVENESS_ROADMAP.md | all |

---

## Phase 1 — Actions package lifts

### Task 1: Lift Forage into `internal/actions/` (TDD)

**Files:**
- Create: `internal/actions/forage.go`
- Create: `internal/actions/forage_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/actions/forage_test.go`. Follow the established local-fake-actor pattern (see `scan_test.go` from chunk 2.8 for the canonical shape — `scanFakeActor` struct + local helpers per-file, no shared test helpers).

```go
package actions

import (
	"testing"
)

// TestForage_NonForagableBiomeReturnsEmpty confirms the action
// returns Found=false with a reason when the actor's biome has
// no entry in forager.ForageYields.
func TestForage_NonForagableBiomeReturnsEmpty(t *testing.T) {
	// Use a biome ID known not to be in ForageYields (e.g., "indoor").
	room := newForageTestRoom(t, 9301, "indoor")
	user := newForageFakeActor(t, "ForageTester", room, true, 1)

	result := Forage(user, ForageOptions{})

	if result.Found {
		t.Error("expected Found=false for non-foragable biome")
	}
	if result.Reason == "" {
		t.Error("expected Reason to be set on biome failure")
	}
	if result.RollHappened {
		t.Error("expected RollHappened=false (no roll on biome failure)")
	}
}

// TestForage_CooldownGate confirms a second call within 6 rounds
// returns OnCooldown=true and skips the roll.
func TestForage_CooldownGate(t *testing.T) {
	room := newForageTestRoom(t, 9302, "forest")
	user := newForageFakeActor(t, "ForageTester2", room, true, 2)

	_ = Forage(user, ForageOptions{})
	second := Forage(user, ForageOptions{})

	if !second.OnCooldown {
		t.Error("second call within 6-round window should return OnCooldown=true")
	}
	if second.RollHappened {
		t.Error("OnCooldown path should not roll")
	}
}

// TestForage_MobActorSilent confirms MobActor.SendText is not called.
func TestForage_MobActorSilent(t *testing.T) {
	room := newForageTestRoom(t, 9303, "forest")
	mob := newForageTestMob(t, 9999, "ForageMob", room.RoomId)
	actor := newForageMobActor(t, mob, room)

	_ = Forage(actor, ForageOptions{})
	// scanFakeActor-style: check actor.sent slice is empty.
	if len(actor.sent) != 0 {
		t.Errorf("MobActor should be silent; got %d messages", len(actor.sent))
	}
}
```

You'll need to author local helpers (`newForageTestRoom`, `newForageFakeActor`, `newForageTestMob`, `newForageMobActor`) modeled exactly on `scan_test.go`'s helpers. The fake actor needs a real `*characters.Character` so cooldowns work (see `searchFakeActor` from `search_test.go` for the pattern).

For `newForageTestRoom`, the biome needs to be settable to a string ID. Use whatever the rooms package's API is for constructing a test room with a specific biome ID — check how `scan_test.go` / `search_test.go` construct rooms and adapt.

- [ ] **Step 2: Run the test, verify it fails (Forage undefined)**

Run: `go test ./internal/actions/ -run TestForage -v`

Expected: build error or "undefined: Forage".

- [ ] **Step 3: Implement actions/forage.go**

Create `internal/actions/forage.go`:

```go
package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/forager"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/skills"
)

// ForageOptions parameterizes a forage attempt.
// Empty v1 — biome derives from actor.GetRoom().GetBiome().
type ForageOptions struct{}

// ForageResult is the structured outcome.
type ForageResult struct {
	Found        bool
	ItemId       int
	ItemName     string
	Reason       string
	OnCooldown   bool
	RollHappened bool
}

// Forage runs a Perception+Search forage attempt scoped to the
// actor's current room biome. Cooldown key "forage" shared with
// the player path (6 rounds). UserActor emits the existing
// snooping emote + "you find X" message; MobActor SendText is a
// no-op (silent).
func Forage(actor Actor, opts ForageOptions) ForageResult {
	result := ForageResult{}

	char := actor.GetCharacter()
	room := actor.GetRoom()
	if char == nil || room == nil {
		result.Reason = "no character or room"
		return result
	}

	biome := room.GetBiome()
	if _, ok := forager.ForageYields[biome.BiomeId]; !ok {
		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem,
				`There is nothing here worth foraging. Try an outdoor area.`)
		}
		result.Reason = "wrong biome"
		return result
	}

	if !char.TryCooldown("forage", "6 rounds") {
		result.OnCooldown = true
		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem,
				fmt.Sprintf("You need to wait %d more rounds before you can forage again.",
					char.GetCooldown("forage")))
		}
		return result
	}

	searchScore := CalcSearchScore(char)

	if actor.IsPlayer() {
		actor.SendText(messaging.CategorySystem,
			`You crouch low and begin searching the ground carefully...`)
		room.SendTextVisual(messaging.CategoryMobEmote,
			fmt.Sprintf(`<ansi fg="username">%s</ansi> is searching the ground for something.`,
				char.Name),
			actor.GetUserId(),
		)
	}

	coreResult := forager.ForageCore(forager.ForageAttempt{
		Biome:       biome.BiomeId,
		SearchScore: searchScore,
		AtNight:     gametime.IsNight(),
	})

	result.RollHappened = true

	if !coreResult.Found {
		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem,
				`You find nothing of use this time.`)
		}
		return result
	}

	newItem := items.New(coreResult.ItemId)
	if !newItem.IsValid() {
		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem,
				`You find something, but it crumbles in your hands.`)
		}
		result.Reason = "item invalid"
		return result
	}

	char.StoreItem(newItem)
	if actor.GetUserId() != 0 {
		events.AddToQueue(events.ItemOwnership{
			UserId: actor.GetUserId(),
			Item:   newItem,
			Gained: true,
		})
	}
	actor.OnSkillUse(string(skills.Search))

	if actor.IsPlayer() {
		actor.SendText(messaging.CategorySystem,
			fmt.Sprintf(`You find a <ansi fg="itemname">%s</ansi>.`, newItem.DisplayName()))
	}

	result.Found = true
	result.ItemId = coreResult.ItemId
	result.ItemName = newItem.DisplayName()
	return result
}
```

- [ ] **Step 4: Run the test, verify it passes**

Run: `go test ./internal/actions/ -run TestForage -v`

Expected: all 3 tests PASS.

- [ ] **Step 5: Run full actions package tests**

Run: `go test ./internal/actions/...`

Expected: all tests pass, no regressions.

- [ ] **Step 6: Commit**

```bash
git add internal/actions/forage.go internal/actions/forage_test.go
git commit -m "$(cat <<'EOF'
feat(actions): lift Forage into actions/ via actor pattern (2.9)

Forage(actor, opts) ForageResult. UserActor emits the existing
snooping emote + yield message; MobActor silent. Cooldown key
"forage" shared with the player path (6 rounds). Skill
progression on every roll-happened path.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 2: Lift Salvage single-tick core into `internal/actions/` (TDD)

**Files:**
- Create: `internal/actions/salvage.go`
- Create: `internal/actions/salvage_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/actions/salvage_test.go` using the local-helper pattern:

```go
package actions

import (
	"testing"
)

// TestSalvage_NoTargetFails confirms an empty options struct
// (no corpse, no item) returns a reason without rolling.
func TestSalvage_NoTargetFails(t *testing.T) {
	room := newSalvageTestRoom(t, 9401)
	user := newSalvageFakeActor(t, "SalvageTester", room, true, 1)

	result := Salvage(user, SalvageOptions{})

	if result.Succeeded {
		t.Error("expected Succeeded=false with no target")
	}
	if result.Reason == "" {
		t.Error("expected Reason to be set")
	}
	if result.RollHappened {
		t.Error("expected RollHappened=false with no target")
	}
}

// TestSalvage_CorpseModeNoCorpse confirms TargetCorpse with no
// eligible corpse in the room returns Failure cleanly.
func TestSalvage_CorpseModeNoCorpse(t *testing.T) {
	room := newSalvageTestRoom(t, 9402)
	mob := newSalvageTestMob(t, 9998, "SalvageMob", room.RoomId)
	actor := newSalvageMobActor(t, mob, room)

	result := Salvage(actor, SalvageOptions{TargetCorpse: true})

	if result.Succeeded {
		t.Error("expected Succeeded=false with no corpse in room")
	}
}

// TestSalvage_ItemModeInvalidUuid confirms TargetItemUuid pointing
// at a non-existent UUID returns Failure with a reason.
func TestSalvage_ItemModeInvalidUuid(t *testing.T) {
	room := newSalvageTestRoom(t, 9403)
	user := newSalvageFakeActor(t, "SalvageTester2", room, true, 2)

	result := Salvage(user, SalvageOptions{TargetItemUuid: "nonexistent-uuid"})

	if result.Succeeded {
		t.Error("expected Succeeded=false for invalid UUID")
	}
}

// TestSalvage_MobActorSilent confirms MobActor.SendText is not called.
func TestSalvage_MobActorSilent(t *testing.T) {
	room := newSalvageTestRoom(t, 9404)
	mob := newSalvageTestMob(t, 9997, "SalvageSilentMob", room.RoomId)
	actor := newSalvageMobActor(t, mob, room)

	_ = Salvage(actor, SalvageOptions{TargetCorpse: true})

	if len(actor.sent) != 0 {
		t.Errorf("MobActor should be silent; got %d messages", len(actor.sent))
	}
}
```

Author local helpers (`newSalvageTestRoom`, `newSalvageFakeActor`, etc.) mirroring `scan_test.go`'s pattern. The fake actor needs a real `*characters.Character` with an `Activity` machine for the salvage path to work cleanly — see `mobcommands/salvage.go:63` for how it nil-guards `Activity`.

- [ ] **Step 2: Run the test, verify it fails**

Run: `go test ./internal/actions/ -run TestSalvage -v`

Expected: build error or "undefined: Salvage".

- [ ] **Step 3: Implement actions/salvage.go**

Create `internal/actions/salvage.go`:

```go
package actions

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
)

// SalvageOptions identifies the salvage target.
//
//   - TargetCorpse: salvage an eligible corpse in the room. Default
//     mode is "first eligible". When TargetCorpseMobId is also set
//     (non-zero), filter for the specific corpse with that MobId +
//     RoundCreated (used by the player path, which started the
//     activity against a specific corpse).
//   - TargetItemUuid: salvage a specific item from actor inventory
//     by UUID. Used by the player path (resolved on the final
//     activity tick from SalvagingData.ItemUuid).
//   - SpoiledPotion: hint that this is a spoiled-potion salvage —
//     overrides recipe lookup and yields binding paste. Set by the
//     player wrapper when starting the activity.
//
// Exactly one of TargetCorpse or TargetItemUuid!="" should be set.
type SalvageOptions struct {
	TargetCorpse              bool
	TargetCorpseMobId         int    // 0 = first eligible; non-zero = specific corpse
	TargetCorpseRoundCreated  uint64 // disambiguator paired with TargetCorpseMobId
	TargetItemUuid            string
	SpoiledPotion             bool
}

// SalvageResult is the structured outcome of one salvage tick.
type SalvageResult struct {
	Succeeded    bool
	MaterialIds  []int
	Reason       string
	RollHappened bool
}

// Salvage runs one tick of the salvage roll. Single-tick by design
// — player-side multi-round UX wraps this via the Activity machine
// + per-tick hook in NewRound_UserRoundTick.go. UserActor emits
// per-tick progress text; MobActor silent. Skill progression via
// actor.OnSkillUse("salvage").
func Salvage(actor Actor, opts SalvageOptions) SalvageResult {
	result := SalvageResult{}

	char := actor.GetCharacter()
	room := actor.GetRoom()
	if char == nil || room == nil {
		result.Reason = "no character or room"
		return result
	}

	if !opts.TargetCorpse && opts.TargetItemUuid == "" {
		result.Reason = "no target"
		return result
	}

	bal := configs.GetBalanceConfig()
	salvageSkill := char.GetSkillLevel(skills.Salvage)
	chance := crafting.CalcSalvageChance(salvageSkill,
		float64(bal.SalvageMinChance),
		float64(bal.SalvageMaxChance),
		int(bal.SalvageSoftCap))

	if opts.TargetCorpse {
		return salvageCorpse(actor, room, opts, chance)
	}
	return salvageItem(actor, opts.TargetItemUuid, opts.SpoiledPotion, chance)
}

// salvageCorpse handles the corpse-target path. Finds the target
// corpse (specific by MobId+RoundCreated, or first eligible),
// rolls returns, removes the corpse, stores materials.
func salvageCorpse(actor Actor, room *rooms.Room, opts SalvageOptions, chance float64) SalvageResult {
	result := SalvageResult{}

	var target rooms.Corpse
	found := false
	for _, c := range room.Corpses {
		if c.Prunable {
			continue
		}
		if c.MobId <= 0 {
			continue
		}
		// Specific-corpse filter (player path).
		if opts.TargetCorpseMobId != 0 {
			if c.MobId != opts.TargetCorpseMobId ||
				c.RoundCreated != opts.TargetCorpseRoundCreated {
				continue
			}
		}
		mobSpec := mobs.GetMobSpec(mobs.MobId(c.MobId))
		if mobSpec == nil {
			continue
		}
		if len(crafting.LookupCorpseSalvage(mobSpec.Groups)) > 0 {
			target = c
			found = true
			break
		}
	}

	if !found {
		result.Reason = "no eligible corpse"
		return result
	}

	result.RollHappened = true

	mobSpec := mobs.GetMobSpec(mobs.MobId(target.MobId))
	returns := crafting.LookupCorpseSalvage(mobSpec.Groups)
	recovered := crafting.RollSalvageReturnsFromSpec(returns, chance)

	room.RemoveCorpse(target)
	actor.OnSkillUse(string(skills.Salvage))

	storeRecovered(actor, recovered, &result)

	if actor.IsPlayer() {
		if len(recovered) > 0 {
			actor.SendText(messaging.CategorySystem, fmt.Sprintf(
				`<ansi fg="green">You salvage the <ansi fg="mobname">%s corpse</ansi> and recover: %s.</ansi>`,
				target.Character.Name,
				formatRecovered(recovered)))
		} else {
			actor.SendText(messaging.CategorySystem, fmt.Sprintf(
				`<ansi fg="red">You attempt to salvage the <ansi fg="mobname">%s corpse</ansi> but recover nothing useful.</ansi>`,
				target.Character.Name))
		}
	} else if room.PlayerCt() > 0 {
		// Mob-side room flavor (matches existing mobcommands/salvage.go).
		room.SendTextVisual(messaging.CategoryMobIdle, fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi> kneels by the carcass and cuts strips of hide from it.`,
			actor.GetName()))
	}

	result.Succeeded = true
	return result
}

// salvageItem handles the item-target path (by UUID). Mirrors the
// logic of the prior resolveSalvageFromData in
// hooks/NewRound_UserRoundTick.go, adapted for the actor interface.
func salvageItem(actor Actor, uuid string, spoiledPotion bool, chance float64) SalvageResult {
	result := SalvageResult{}
	char := actor.GetCharacter()

	// Find item in inventory by UUID.
	var targetItem items.Item
	found := false
	for _, itm := range char.Items {
		if itm.UUID.String() == uuid {
			targetItem = itm
			found = true
			break
		}
	}
	if !found {
		if actor.IsPlayer() {
			actor.SendText(messaging.CategoryError,
				`<ansi fg="red">The item you were salvaging is no longer in your backpack.</ansi>`)
		}
		result.Reason = "item not found"
		return result
	}

	itemId := targetItem.ItemId
	spec := items.GetItemSpec(itemId)
	if spec == nil {
		if actor.IsPlayer() {
			actor.SendText(messaging.CategoryError,
				`<ansi fg="red">Something went wrong with your salvage attempt.</ansi>`)
		}
		result.Reason = "spec not found"
		return result
	}

	// Roll returns from spoiled-potion branch, recipe lookup, or
	// tagged salvage_returns.
	var recovered []crafting.RecipeIngredient
	if spoiledPotion {
		qty := 1
		if chance > 0.5 {
			qty = 2
		}
		recovered = []crafting.RecipeIngredient{
			{ItemTag: "binding-paste", Quantity: qty},
		}
	} else {
		recipe := crafting.GetRecipeByOutputItemId(itemId)
		if recipe != nil {
			recovered = crafting.RollSalvageReturns(recipe.Ingredients, chance)
		} else if len(spec.SalvageReturns) > 0 {
			recovered = crafting.RollSalvageReturnsFromSpec(spec.SalvageReturns, chance)
		}
	}

	result.RollHappened = true

	// Always destroy the item (matches existing behavior).
	char.RemoveItem(targetItem)
	actor.OnSkillUse(string(skills.Salvage))

	storeRecovered(actor, recovered, &result)

	if actor.IsPlayer() {
		if len(recovered) > 0 {
			actor.SendText(messaging.CategorySystem, fmt.Sprintf(
				`<ansi fg="green">You salvage the <ansi fg="itemname">%s</ansi> and recover: %s.</ansi>`,
				targetItem.DisplayName(),
				formatRecovered(recovered)))
		} else {
			actor.SendText(messaging.CategorySystem, fmt.Sprintf(
				`<ansi fg="red">You attempt to salvage the <ansi fg="itemname">%s</ansi> but recover nothing useful.</ansi>`,
				targetItem.DisplayName()))
		}
	}

	result.Succeeded = true
	return result
}

// storeRecovered creates the material items and stores them in the
// actor's inventory, populating result.MaterialIds.
func storeRecovered(actor Actor, recovered []crafting.RecipeIngredient, result *SalvageResult) {
	char := actor.GetCharacter()
	for _, ing := range recovered {
		for i := 0; i < ing.Quantity; i++ {
			matSpec := items.FindSpecByComponentTag(ing.ItemTag)
			if matSpec == nil {
				continue
			}
			newItem := items.New(matSpec.ItemId)
			char.StoreItem(newItem)
			result.MaterialIds = append(result.MaterialIds, matSpec.ItemId)
		}
	}
}

// formatRecovered builds the comma-separated yield list for player
// flavor text. Matches the format used by the prior player path in
// hooks/NewRound_UserRoundTick.go.
func formatRecovered(recovered []crafting.RecipeIngredient) string {
	parts := make([]string, 0, len(recovered))
	for _, ing := range recovered {
		parts = append(parts, fmt.Sprintf("%dx %s", ing.Quantity, ing.ItemTag))
	}
	return strings.Join(parts, ", ")
}
```

- [ ] **Step 4: Run the test, verify it passes**

Run: `go test ./internal/actions/ -run TestSalvage -v`

Expected: all 4 tests PASS.

- [ ] **Step 5: Run full actions package tests**

Run: `go test ./internal/actions/...`

Expected: all tests pass, no regressions.

- [ ] **Step 6: Commit**

```bash
git add internal/actions/salvage.go internal/actions/salvage_test.go
git commit -m "$(cat <<'EOF'
feat(actions): lift Salvage single-tick core into actions/ (2.9)

Salvage(actor, opts) SalvageResult. Two modes:
  - TargetCorpse: find first eligible corpse in room, roll yield.
  - TargetItemUuid: find item in inventory by UUID, roll yield.
UserActor emits flavor text; MobActor silent. Skill progression on
every roll-happened path. The player multi-round activity calls
this on its final tick via the hook in NewRound_UserRoundTick.go
(refactor lands in Task 4).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Phase 2 — Player + hook refactors

### Task 3: Thin `usercommands/skill.forage.go`

**Files:**
- Modify: `internal/usercommands/skill.forage.go` (full rewrite, ~30 LoC)

- [ ] **Step 1: Rewrite the player forage wrapper**

Replace the entire contents of `internal/usercommands/skill.forage.go` with:

```go
package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Forage is a thin wrapper over actions.Forage. The action handles
// biome check, cooldown, score calculation, rendering, and item
// store. This wrapper handles the quest-engine command notification.
func Forage(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	actor := &actions.UserActor{User: user, Room: room}
	_ = actions.Forage(actor, actions.ForageOptions{})

	// Quest engine: command notification (preserved from prior behavior).
	bridge := questengine.NewGameBridge(user, room.RoomId)
	questengine.GetEngine().Notify("command", questengine.EventDetails{
		UserId:  user.UserId,
		RoomId:  room.RoomId,
		Command: "forage",
	}, bridge, bridge)

	return true, nil
}
```

- [ ] **Step 2: Build to confirm**

Run: `go build ./...`

Expected: clean exit.

- [ ] **Step 3: Run usercommands tests**

Run: `go test ./internal/usercommands/...`

Expected: all tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/usercommands/skill.forage.go
git commit -m "$(cat <<'EOF'
refactor(forage): thin player wrapper over actions.Forage (2.9)

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 4: Refactor `hooks/NewRound_UserRoundTick.go` to call `actions.Salvage`

**Files:**
- Modify: `internal/hooks/NewRound_UserRoundTick.go` — `resolveSalvageFromData` (line ~499) and `resolveCorpseSalvage` (line ~587)

- [ ] **Step 1: Refactor `resolveSalvageFromData`**

Open `internal/hooks/NewRound_UserRoundTick.go` and locate `resolveSalvageFromData`. Replace its body (lines 499-582) with a call to `actions.Salvage`:

```go
func resolveSalvageFromData(user *users.UserRecord, sd activity.SalvagingData) {
	actor := &actions.UserActor{
		User: user,
		Room: rooms.LoadRoom(user.Character.RoomId),
	}
	_ = actions.Salvage(actor, actions.SalvageOptions{
		TargetItemUuid: sd.ItemUuid,
		SpoiledPotion:  sd.SpoiledPotion,
	})
}
```

The action handles all messaging, material storage, and skill progression. The 80 lines previously inline are now in `actions.Salvage`'s `salvageItem` branch.

- [ ] **Step 2: Refactor `resolveCorpseSalvage`**

In the same file, locate `resolveCorpseSalvage` (around line 587). Replace its body with a call to `actions.Salvage` using the `TargetCorpseMobId` + `TargetCorpseRoundCreated` fields (declared in Task 2's `SalvageOptions`) to identify the specific corpse the player started salvaging:

```go
func resolveCorpseSalvage(user *users.UserRecord, mobIdStr string) {
	var mobId int
	fmt.Sscanf(mobIdStr, "%d", &mobId)

	// Pull stashed corpse identity (existing logic).
	roundCreatedInt, _ := user.Character.GetMiscData("salvage_corpse_round_created").(int)
	user.Character.SetMiscData("salvage_corpse_round_created", nil)
	user.Character.SetMiscData("salvage_corpse_name", nil)

	actor := &actions.UserActor{
		User: user,
		Room: rooms.LoadRoom(user.Character.RoomId),
	}
	_ = actions.Salvage(actor, actions.SalvageOptions{
		TargetCorpse:             true,
		TargetCorpseMobId:        mobId,
		TargetCorpseRoundCreated: uint64(roundCreatedInt),
	})
}
```

The action's `salvageCorpse` branch handles the corpse-finding (filtering for the specific MobId+RoundCreated pair), yield rolling, corpse removal, material storage, and messaging.

- [ ] **Step 3: Prune now-unused imports**

After the body deletions, several imports in `NewRound_UserRoundTick.go` will become unused (e.g., `crafting`, possibly `items` if no other consumer in the file). Run `go build ./...` and prune any "imported and not used" errors.

- [ ] **Step 4: Build + test**

Run: `go build ./... && go test ./internal/hooks/... && go test ./internal/actions/...`

Expected: clean exit, all tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/hooks/NewRound_UserRoundTick.go internal/actions/salvage.go
git commit -m "$(cat <<'EOF'
refactor(salvage): unify player per-tick resolve on actions.Salvage (2.9)

resolveSalvageFromData and resolveCorpseSalvage in
NewRound_UserRoundTick.go now delegate to actions.Salvage instead
of inlining the roll + storage + messaging. SalvageOptions
extended with TargetCorpseMobId + TargetCorpseRoundCreated for
the specific-corpse-identification path the player needs.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Phase 3 — Mob wrappers

### Task 5: Add mob wrappers + register `forage`

**Files:**
- Create: `internal/mobcommands/forage.go`
- Modify: `internal/mobcommands/salvage.go` (full rewrite, thin wrapper)
- Modify: `internal/mobcommands/mobcommands.go` (register `forage`)

- [ ] **Step 1: Create `mobcommands/forage.go`**

```go
package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Forage runs a single forage attempt for the mob. Thin wrapper
// over actions.Forage. Mob actor is silent — the structured result
// is consumed by the btree primitive try_forage.
func Forage(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	actor := actions.NewMobActorInRoom(mob, room)
	_ = actions.Forage(actor, actions.ForageOptions{})
	return true, nil
}
```

- [ ] **Step 2: Rewrite `mobcommands/salvage.go`**

Replace the entire contents with:

```go
package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Salvage runs a single-tick corpse salvage for the mob. Thin
// wrapper over actions.Salvage with TargetCorpse=true. The action
// handles corpse-finding, yield rolling, material storage, and
// room flavor (when players are present).
func Salvage(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	actor := actions.NewMobActorInRoom(mob, room)
	_ = actions.Salvage(actor, actions.SalvageOptions{TargetCorpse: true})
	return true, nil
}
```

The existing 119-line `mobcommands/salvage.go` collapses to ~17 lines. All the corpse-finding + Activity transitions + yield + flavor logic moves into `actions.Salvage`.

NOTE: the existing mob salvage code did Activity machine transitions (`TransitionToSalvaging` and `TransitionToFree`) for state-machine parity. These need to either:
- Stay in `actions.Salvage`'s mob path (action handles Activity transitions for mob actors), OR
- Be dropped from the mob path entirely if not load-bearing.

Inspect what consumes mob Activity state. If nothing in `actions_forager.go` or other btree code reads mob Activity for salvage decisions, the transitions are pure ceremony and can be dropped. If they're load-bearing, port them into `actions.Salvage`'s corpse branch when `!actor.IsPlayer()`.

- [ ] **Step 3: Register `forage` in `mobcommands/mobcommands.go`**

Edit `internal/mobcommands/mobcommands.go`. Add to the `mobCommands` map alphabetically (insert between `flee` and `gearup`):

```go
		"forage":         {Forage, false},
```

`salvage` and `scan` etc. are already registered from prior chunks; only `forage` needs adding.

- [ ] **Step 4: Build**

Run: `go build ./...`

Expected: clean exit.

- [ ] **Step 5: Run mobcommands tests**

Run: `go test ./internal/mobcommands/...`

Expected: all tests pass. The existing `mobcommands/salvage_test.go` may need updates if it tested internals that moved into actions/. Inspect test failures and adjust:
- If a test asserted intermediate state (e.g., "Activity transitioned to Salvaging mid-call"), either move the assertion to `actions/salvage_test.go` or update it to assert the post-call state.
- If a test ran the function and checked output messages, it should still pass via the action delegation.

- [ ] **Step 6: Commit**

```bash
git add internal/mobcommands/forage.go internal/mobcommands/salvage.go internal/mobcommands/mobcommands.go internal/mobcommands/salvage_test.go
git commit -m "$(cat <<'EOF'
feat(mobcommands): add forage + thin salvage wrappers (2.9)

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Phase 4 — Btree primitives

### Task 6: Add btree primitives + condition

**Files:**
- Create: `internal/behaviortree/actions_forager_verbs.go`
- Create: `internal/behaviortree/conditions_forager.go`

- [ ] **Step 1: Create `actions_forager_verbs.go`**

```go
package behaviortree

// actions_forager_verbs.go — btree action primitives for chunk 2.9
// forager-suite verbs: try_forage, try_salvage, wander_territory.
//
// try_forage / try_salvage delegate to actions.Forage / actions.Salvage.
// wander_territory delegates to npcWanderTerritoryNeighbor (the
// existing helper in actions_forager.go) so territory-aware
// movement is preserved when the foraging tick loop runs in YAML.

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/forager"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// actTryForage runs one forage attempt via actions.Forage.
// Returns Success on item found; Failure on miss/cooldown/wrong-biome.
func actTryForage(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return Failure
	}
	actor := actions.NewMobActorInRoom(mob, room)
	result := actions.Forage(actor, actions.ForageOptions{})
	if result.Found {
		return Success
	}
	return Failure
}

// actTrySalvage runs one salvage tick via actions.Salvage. Default
// mode targets the first eligible corpse in the room.
//
// Optional param "item_uuid" overrides to item-target mode (string).
func actTrySalvage(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return Failure
	}

	opts := actions.SalvageOptions{TargetCorpse: true}
	if uuid := getStringParam(params, "item_uuid"); uuid != "" {
		opts = actions.SalvageOptions{TargetItemUuid: uuid}
	}

	actor := actions.NewMobActorInRoom(mob, room)
	result := actions.Salvage(actor, opts)
	if result.Succeeded {
		return Success
	}
	return Failure
}

// actWanderTerritory delegates to the existing
// npcWanderTerritoryNeighbor helper in actions_forager.go for
// territory-aware adjacent-room movement. Returns Failure on
// mobs without a forager profile.
func actWanderTerritory(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	profile := forager.ProfileFor(int(mob.MobId))
	if profile == nil {
		return Failure
	}
	npcWanderTerritoryNeighbor(profile, mob, ctx)
	return Success
}
```

- [ ] **Step 2: Create `conditions_forager.go`**

```go
package behaviortree

import (
	"github.com/GoMudEngine/GoMud/internal/forager"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// condForagerStateIsForaging returns Success when the mob's
// forager state machine is currently in StateForaging. Lets the
// archetype YAML branch the per-tick foraging loop.
func condForagerStateIsForaging(params map[string]any, ctx *EvalContext) Result {
	if ctx.MobState == nil {
		return Failure
	}
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	if readForagerState(ctx.MobState) == forager.StateForaging {
		return Success
	}
	return Failure
}
```

`readForagerState` is the existing helper in `actions_forager.go` (line ~114). It's package-private and already accessible from this new file (same package).

- [ ] **Step 3: Build**

Run: `go build ./...`

Expected: clean. The new functions will be flagged as unused (no registration yet — that's Task 7), but that's a lint hint, not an error.

- [ ] **Step 4: Commit**

```bash
git add internal/behaviortree/actions_forager_verbs.go internal/behaviortree/conditions_forager.go
git commit -m "$(cat <<'EOF'
feat(behaviortree): add forager-suite primitives (2.9)

Adds try_forage, try_salvage, wander_territory actions plus the
forager_state_is_foraging condition. Primitives delegate to
actions.Forage / actions.Salvage; wander_territory wraps the
existing npcWanderTerritoryNeighbor helper for territory-aware
movement. Registration lands in Task 7.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 7: Register btree primitives in init()

**Files:**
- Modify: `internal/behaviortree/actions.go` (add 3 entries to `actionRegistry`)
- Modify: `internal/behaviortree/conditions.go` (add 1 entry to `conditionRegistry`)

- [ ] **Step 1: Register actions**

In `internal/behaviortree/actions.go`, locate the `init()` function and add at the end (after the existing scout-suite registrations from chunk 2.8):

```go
	// Forager suite (2.9)
	actionRegistry["try_forage"] = actTryForage
	actionRegistry["try_salvage"] = actTrySalvage
	actionRegistry["wander_territory"] = actWanderTerritory
```

- [ ] **Step 2: Register condition**

In `internal/behaviortree/conditions.go`, locate the `init()` function and add at the end:

```go
	// Forager suite (2.9)
	conditionRegistry["forager_state_is_foraging"] = condForagerStateIsForaging
```

- [ ] **Step 3: Build + test**

Run: `go build ./... && go test ./internal/behaviortree/...`

Expected: clean exit, all tests pass. The previously-unused diagnostics from Task 6 should clear.

- [ ] **Step 4: Commit**

```bash
git add internal/behaviortree/actions.go internal/behaviortree/conditions.go
git commit -m "$(cat <<'EOF'
feat(behaviortree): register forager-suite primitives in init() (2.9)

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Phase 5 — Forager state machine refactor

### Task 8: Refactor `forager_step` — Foraging-state guards + delete tick handler

**Files:**
- Modify: `internal/behaviortree/actions_forager.go` (delete `tickForagerForaging`, add Foraging-state guards at top of `actForagerStep`, verify `npcAttemptForage` no longer used)

- [ ] **Step 1: Verify `npcAttemptForage` and `tickForagerForaging` callers**

Run Grep on `internal/` for `npcAttemptForage` and `tickForagerForaging`. If references appear outside `actions_forager.go`, STOP and report — the deletion is unsafe.

Expected: both functions are only referenced within `actions_forager.go` and its sibling test file `actions_forager_test.go`.

- [ ] **Step 2: Add Foraging-state guards at top of `actForagerStep`**

Open `internal/behaviortree/actions_forager.go`. Locate the existing `actForagerStep` function (around line 49). After the HP-emergency check and stuck-state watchdog block, BEFORE the existing `switch cur` dispatch, insert:

```go
	// Foraging-state per-tick coordination. The per-tick foraging
	// loop (forage roll, salvage, wander) now runs in YAML via
	// try_forage / try_salvage / wander_territory primitives.
	// The state machine still owns the transition triggers OUT of
	// Foraging — fatigue limit or carry threshold sends the mob
	// to TravelingToDropoff.
	if cur == forager.StateForaging {
		// Fatigue tick (was inside tickForagerForaging).
		fatigue := getIntFromState(ctx.MobState, keyFatigueTimer) + 1
		ctx.MobState.Set(keyFatigueTimer, strconv.Itoa(fatigue))

		// Carry-cap or fatigue → head to dropoff.
		if fatigue >= fatigueLimit ||
			carryRatio(mob) >= float64(cfg.ForagerCarryThresholdPct) {
			transitionForager(ctx.MobState, forager.StateTravelingToDropoff)
			return Success
		}

		// YAML handles try_forage / try_salvage / wander_territory.
		return Success
	}
```

- [ ] **Step 3: Remove the `case forager.StateForaging` arm from the dispatch switch**

Locate the `switch cur` dispatch in `actForagerStep`. Delete the case that calls `tickForagerForaging`:

```go
	case forager.StateForaging:
		return tickForagerForaging(profile, mob, ctx, cfg)
```

The other cases (Resting, TravelingToTerritory, TravelingToDropoff, Delivering, Recalling) stay.

- [ ] **Step 4: Delete `tickForagerForaging`**

In `actions_forager.go`, locate `tickForagerForaging` (line ~211). Delete the entire function. Its responsibilities (fatigue tick, carry check, forage roll, salvage corpse, wander) are now distributed:
- Fatigue tick + carry check → top of `actForagerStep` (Step 2).
- Forage roll → `actions.Forage` via `try_forage` btree primitive.
- Salvage corpse → `actions.Salvage` via `try_salvage` btree primitive.
- Wander → `npcWanderTerritoryNeighbor` via `wander_territory` btree primitive.

- [ ] **Step 5: Delete `npcAttemptForage` if unused**

After deleting `tickForagerForaging`, search for remaining `npcAttemptForage` callers. If zero, delete the function. If any remain (unlikely), STOP and report.

- [ ] **Step 6: Build + test**

Run: `go build ./... && go test ./internal/behaviortree/...`

Expected: clean. If any test in `actions_forager_test.go` referenced `tickForagerForaging` or `npcAttemptForage` directly, those tests need updates. Common adjustments:
- Tests that asserted a Foraging-state tick produced a specific result → adapt to assert the new transition-only behavior (forager_step returns Success in Foraging with fatigue ticked).
- Tests that asserted forage rolls fired → those tests now belong in `actions/forage_test.go`'s domain.

If tests need adjustment, inspect them and make the minimum changes to preserve coverage. Don't delete coverage — migrate it.

- [ ] **Step 7: Commit**

```bash
git add internal/behaviortree/actions_forager.go internal/behaviortree/actions_forager_test.go
git commit -m "$(cat <<'EOF'
refactor(behaviortree): dissolve Foraging-state per-tick loop (2.9)

forager_step's Foraging-state arm replaced with transition-only
guards at the top of the function: increment fatigue, check
fatigue/carry thresholds, transition to TravelingToDropoff when
applicable, otherwise return Success and let the YAML handle the
per-tick verbs. tickForagerForaging and npcAttemptForage deleted.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Phase 6 — Archetype + migration

### Task 9: Author `forager.yaml` archetype

**Files:**
- Create: `_datafiles/world/dogmud/behaviors/archetypes/forager.yaml`

- [ ] **Step 1: Create the archetype YAML**

```yaml
# forager archetype
#
# Routine NPC foragers (Tova 371, Halix 372, Kessa 373). The
# high-level daily cycle (Resting → Traveling → Foraging →
# Delivering → Recalling) stays in the Go state machine via
# forager_step. The per-tick Foraging loop runs here in YAML so
# the verb calls (try_forage, try_salvage) flow through the
# unified actions.Forage / actions.Salvage pipeline.
#
# Spec: docs/superpowers/specs/
#       2026-05-22-mob-aliveness-2.9-mob-forage-salvage-design.md

tree:
  type: selector
  children:

    # 1. Self-defense — fight back if attacked.
    - type: sequence
      event: mob_hurt
      children:
        - type: action
          do: attack

    # 2. Foraging-state per-tick loop: salvage opportunistically,
    # then try a forage roll, then wander within territory. Each
    # step returns Failure cleanly when nothing's available, so
    # the inner selector falls through.
    - type: sequence
      event: mob_idle
      children:
        - type: condition
          check: forager_state_is_foraging
        - type: selector
          children:
            - type: action
              do: try_salvage
            - type: action
              do: try_forage
            - type: action
              do: wander_territory

    # 3. All non-Foraging states (Resting, TravelingToTerritory,
    # TravelingToDropoff, Delivering, Recalling) flow through the
    # Go state machine.
    - type: sequence
      event: mob_idle
      children:
        - type: action
          do: forager_step
```

- [ ] **Step 2: Boot the server, confirm archetype loads**

Run: `go run . 2>&1 | head -120`

Expected: clean boot. No panics. Kill the server after verification (per CLAUDE.md SOP).

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/behaviors/archetypes/forager.yaml
git commit -m "$(cat <<'EOF'
feat(archetypes): add forager archetype (2.9)

Shared archetype for routine foragers (Tova, Halix, Kessa).
Foraging-state tick loop in YAML via try_forage / try_salvage /
wander_territory; multi-state daily cycle delegates to
forager_step.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 10: Migrate 3 forager mobs + delete per-mob behavior YAMLs

**Files:**
- Modify: `_datafiles/world/dogmud/mobs/stillwater_marsh/371-tova.yaml`
- Modify: `_datafiles/world/dogmud/mobs/ironwind_steppe/372-halix.yaml`
- Modify: `_datafiles/world/dogmud/mobs/the_fernway_south/373-kessa.yaml`
- Delete: `_datafiles/world/dogmud/behaviors/stillwater_marsh/371-tova.yaml`
- Delete: `_datafiles/world/dogmud/behaviors/ironwind_steppe/372-halix.yaml`
- Delete: `_datafiles/world/dogmud/behaviors/the_fernway_south/373-kessa.yaml`

- [ ] **Step 1: Flip mob YAMLs to `behavior_archetype: forager`**

For each of the three mob YAMLs (371-tova, 372-halix, 373-kessa), use Edit tool to change:

```yaml
behavior_archetype: ""
```

to:

```yaml
behavior_archetype: forager
```

All other fields preserved. Read each file first to confirm the exact current state of `behavior_archetype`.

- [ ] **Step 2: Delete the three per-mob behavior YAMLs**

```bash
rm "_datafiles/world/dogmud/behaviors/stillwater_marsh/371-tova.yaml"
rm "_datafiles/world/dogmud/behaviors/ironwind_steppe/372-halix.yaml"
rm "_datafiles/world/dogmud/behaviors/the_fernway_south/373-kessa.yaml"
```

If the directories `behaviors/stillwater_marsh`, `behaviors/ironwind_steppe`, `behaviors/the_fernway_south` are now empty, leave them empty — don't delete the directories themselves (the loader may scan them and an empty dir is fine).

- [ ] **Step 3: Boot the server, confirm clean load**

Run: `go run . 2>&1 | head -150`

Expected: clean boot. `mobs.LoadDataFiles()` loads the three foragers without errors. The behaviors loader picks up the new `forager` archetype reference for each. Kill the server.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/mobs/stillwater_marsh/371-tova.yaml _datafiles/world/dogmud/mobs/ironwind_steppe/372-halix.yaml _datafiles/world/dogmud/mobs/the_fernway_south/373-kessa.yaml _datafiles/world/dogmud/behaviors/stillwater_marsh/371-tova.yaml _datafiles/world/dogmud/behaviors/ironwind_steppe/372-halix.yaml _datafiles/world/dogmud/behaviors/the_fernway_south/373-kessa.yaml
git commit -m "$(cat <<'EOF'
feat(foragers): migrate Tova/Halix/Kessa to forager archetype (2.9)

Three forager mobs now use the shared `forager` behavior_archetype.
Per-mob behavior YAMLs deleted — the archetype covers their full
behavior.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Phase 7 — Docs + smoke

### Task 11: Update context.md files

**Files:**
- Modify: `internal/behaviortree/context.md`
- Modify: `internal/actions/context.md`

- [ ] **Step 1: Update `behaviortree/context.md`**

Add entries for the new primitives and condition, matching existing formatting (alphabetical insertion, brief descriptions):

- `try_forage` — invokes actions.Forage; Success on item found, Failure on miss/cooldown/wrong-biome
- `try_salvage` — invokes actions.Salvage; default mode targets first eligible corpse in room, optional `item_uuid` param for item-target
- `wander_territory` — delegates to npcWanderTerritoryNeighbor for territory-aware movement; Failure without forager profile
- `forager_state_is_foraging` (condition) — true when forager state machine is in Foraging state

If the file has an "Instant vs Delayed" classification table (per chunk 2.8 precedent), classify: try_forage instant, try_salvage instant, wander_territory delayed (dispatches movement).

- [ ] **Step 2: Update `actions/context.md`**

Add entries for the new lifted actions, matching the existing entries (Sneak, Steal, Shadow, Scan, etc.):

- `Forage(actor, opts) ForageResult` — single forage attempt; biome-gated, 6-round cooldown shared with player path
- `Salvage(actor, opts) SalvageResult` — single-tick salvage; modes: TargetCorpse (mob default), TargetItemUuid (player per-tick from hook)

- [ ] **Step 3: Commit**

```bash
git add internal/behaviortree/context.md internal/actions/context.md
git commit -m "$(cat <<'EOF'
docs(2.9): update context.md for forager-suite primitives + actions

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 12: Final boot + smoke + roadmap update

**Files:**
- Modify: `MOB_ALIVENESS_ROADMAP.md`

- [ ] **Step 1: Final clean build + full test suite**

Run: `go build ./... && go test ./...`

Expected: clean exit, all tests pass.

- [ ] **Step 2: Boot the server cleanly**

Run: `go run . 2>&1 | head -200`

Expected: clean boot past all `LoadDataFiles()` calls. The new `forager` archetype loads; the three forager mobs resolve their archetype reference cleanly. Kill the server.

- [ ] **Step 3: In-game smoke (deferred to user)**

The 9-scenario in-game smoke plan in the spec requires live gameplay and cannot be run by an autonomous subagent. Note this in the final report.

- [ ] **Step 4: Update `MOB_ALIVENESS_ROADMAP.md`**

1. Progress tracker row: change `| 2.9 | Tactical | Mob \`forage\` as a command | S | — | Not started |` to `| 2.9 | Tactical | Mob \`forage\` as a command | M | — | Done |` (size S → M).

2. Roll-up line: change `**Roll-up:** 16 / 41 done • 0 in progress • 25 not started.` to `**Roll-up:** 17 / 41 done • 0 in progress • 24 not started.`.

3. Chunk 2.9 mini-brief: change `**Status:** Not started • **Size:** S` to `**Status:** Done (2026-05-22) • **Size:** M (originally scoped S; expanded during brainstorming to include salvage parallel + forager archetype migration + state-machine refactor)`. Append a `**Shipped:**` bullet:

```
- **Shipped:** Two actions lifted into `internal/actions/` (`Forage`, `Salvage` single-tick core). Three btree primitives (`try_forage`, `try_salvage`, `wander_territory`) + one condition (`forager_state_is_foraging`). Hybrid forager state-machine refactor: Foraging-state per-tick loop dissolved into YAML, multi-state daily cycle preserved in Go via `forager_step`. New shared `forager` archetype replaces three per-mob behavior YAMLs for Tova (371), Halix (372), Kessa (373). Player per-tick salvage resolve in `hooks/NewRound_UserRoundTick.go` refactored to call `actions.Salvage`. Spec at `docs/superpowers/specs/2026-05-22-mob-aliveness-2.9-mob-forage-salvage-design.md`, plan at `docs/superpowers/plans/2026-05-22-mob-aliveness-2.9-mob-forage-salvage.md`.
```

- [ ] **Step 5: Final commit**

```bash
git add MOB_ALIVENESS_ROADMAP.md
git commit -m "$(cat <<'EOF'
docs(2.9): mark mob-aliveness 2.9 Done

Roll-up: 17 / 41 done. Size: S → M (scope expanded during
brainstorming to include salvage parallel + forager archetype
migration + state-machine refactor). In-game smoke testing
deferred to user.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Self-review checklist (for the implementing engineer)

After all 12 tasks land, before declaring 2.9 done:

- [ ] Both actions lifted via the actor pattern (`internal/actions/{forage,salvage}.go`).
- [ ] Player forage wrapper thinned to ~25 LoC.
- [ ] `usercommands/salvage.go` UNCHANGED (it's just the initiator).
- [ ] `hooks/NewRound_UserRoundTick.go`'s `resolveSalvageFromData` + `resolveCorpseSalvage` delegate to `actions.Salvage`.
- [ ] Mob wrappers added (`forage`, `salvage`) and registered in `mobcommands.go`.
- [ ] Three btree actions + one condition registered in init().
- [ ] `forager_step` refactored: Foraging-state guards at top + dispatch arm removed + `tickForagerForaging` deleted.
- [ ] `forager.yaml` archetype authored.
- [ ] Three mob YAMLs flipped to `behavior_archetype: forager`; three per-mob behavior YAMLs deleted.
- [ ] context.md files updated (behaviortree + actions).
- [ ] Roadmap row + roll-up + mini-brief updated; size noted as S → M.
- [ ] `go build ./...` clean.
- [ ] `go test ./...` clean.
- [ ] Server boots cleanly past all data files.
- [ ] No remaining `tickForagerForaging` or `npcAttemptForage` references.
