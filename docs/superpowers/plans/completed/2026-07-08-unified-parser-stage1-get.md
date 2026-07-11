# Unified Parser Seam — Stage 1 (`get` composition) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Retire `get.go`'s hand-rolled container/corpse/pet **detection ladder**
(the code that decides which trailing span is a container vs. corpse vs. pet —
the exact site of the 2026-07-08 corpse-loot bug) by routing it through a shared
`parser.SplitTrailingContainer` helper, **preserving every gate, message, and
transfer path byte-for-byte.**

**Architecture:** This is a **behavior-preserving refactor**, not a feature.
`get.go` keeps all of its command policy — `all`, gold, bag/bandolier moves,
stash, corpse ownership/loot-mode gates, hidden-container discovery gate,
exploding-item guard, encumbrance, events. Only the *detection* of a trailing
container/corpse/pet (and the item-span split) moves into the parser. The
existing branch bodies that apply gates and transfer items run unchanged.

**Tech Stack:** Go, `testify`, the `internal/parser` package (Stage 0), the
existing `internal/usercommands` test harness (`seedAllRegistries`).

**Spec:** `docs/superpowers/specs/2026-07-08-unified-parser-seam-design.md`
(see "Divergences Discovered During Implementation" — this stage is C/bug-
prevention, NOT a multi-word feature; single-token multi-word already works).

---

## Why a *split* helper, not a full `ResolveItem` swap

`get.go` is ~650 lines with many entangled paths (gold, `all`, bag/bandolier,
stash, per-source gates). A wholesale swap to `ResolveItem` would have to
reproduce all of that and is high-risk. Instead we extract only the piece that
was actually buggy — "given `get`'s input, is there a trailing container/corpse/
pet, and what's the item span?" — into `parser.SplitTrailingContainer`. `get.go`
feeds its existing `containerName` / `corpseIdx` / `petUserId` / `rest` variables
from the helper's result, and every downstream branch stays as-is. This kills
the duplicated, bug-prone detection while touching the least surface.

## Gate-Preservation Checklist (MUST stay behavior-identical)

The refactor MUST NOT change any of these — they stay in `get.go`:
1. `all` / `all.item` handling (component bag, bandolier, `all <container>`,
   `all <corpse>`, `all` floor).
2. Gold: `get gold`, `get gold from <container|corpse>`.
3. `get X from bag/case/pouch/bandolier` (inventory-to-inventory moves) and the
   "item whose name ends in a container word" disambiguation (the Vitalis
   Bandolier case).
4. `stash` / `from stash` / `ground` / `from ground` modifiers.
5. Corpse: `canLootCorpse` (kill ownership) + `CanTakeItem` (loot-mode) gates,
   and their exact refusal messages.
6. Room container: the **hidden-container discovery gate** (`c.Hidden &&
   !HasDiscovery`).
7. Floor: the **exploding-item guard** and stash auto-detect.
8. `CancelBuffsWithFlag(Hidden)`, `ItemOwnership` events, encumbrance warnings.

The regression backbone is the existing suite: `TestGet`, `TestGet_CorpseLoot`,
and the bandolier/container-word subtests in `usercommands_test.go`. They must
stay green with zero edits.

---

## Task 1: Expose `SplitTrailingContainer` in the parser + complete pet source

Stage 0's `ResolveItem` already contains the trailing-container detection loop
internally. Extract it into an exported helper that returns the split (item span
+ container/corpse/pet Match) *without* resolving the item inside — `get.go`'s
branches do their own item resolution + gating. Also add `KindPet` to the
detection (the deferred Stage-0 pet branch).

**Files:**
- Modify: `internal/parser/helpers.go`
- Test: `internal/parser/helpers_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/parser/helpers_test.go`:

```go
func TestSplitTrailingContainer_Corpse(t *testing.T) {
	s, cleanup := seedParserTest(t)
	defer cleanup()
	s.Room.Corpses = []rooms.Corpse{{
		MobId:     1,
		Character: characters.Character{Name: "Skeleton"},
		Loot:      rooms.Container{Items: []items.Item{items.New(10001)}},
	}}

	// "sword corpse" (no "from") splits into item="sword", corpse container.
	itemPart, cm, ok := SplitTrailingContainer(s, "sword corpse")
	require.True(t, ok)
	assert.Equal(t, "sword", itemPart)
	assert.Equal(t, KindCorpse, cm.Kind)
	assert.Equal(t, 0, cm.CorpseIdx)
}

func TestSplitTrailingContainer_ExplicitFrom(t *testing.T) {
	s, cleanup := seedParserTest(t)
	defer cleanup()
	s.Room.Containers = map[string]rooms.Container{
		"wooden chest": {Items: []items.Item{items.New(10001)}},
	}
	itemPart, cm, ok := SplitTrailingContainer(s, "iron sword from wooden chest")
	require.True(t, ok)
	assert.Equal(t, "iron sword", itemPart)
	assert.Equal(t, KindRoomContainer, cm.Kind)
	assert.Equal(t, "wooden chest", cm.ContainerName)
}

func TestSplitTrailingContainer_NoContainer(t *testing.T) {
	s, cleanup := seedParserTest(t)
	defer cleanup()
	// Nothing trailing resolves to a container/corpse/pet.
	_, _, ok := SplitTrailingContainer(s, "lake iron nodule")
	assert.False(t, ok)
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/parser/ -run TestSplitTrailingContainer -v`
Expected: FAIL — `undefined: SplitTrailingContainer`.

- [ ] **Step 3: Refactor `ResolveItem` to use the new exported helper**

In `internal/parser/helpers.go`, replace the no-"from" loop inside `ResolveItem`
and add the exported helper. The final shape:

```go
// SplitTrailingContainer detects whether input ends in a container / corpse /
// pet and, if so, returns the leading item span plus the container Match. It
// does NOT resolve the item inside — callers (e.g. get.go) apply their own
// gates and item lookup. Handles both "X from Y" and "X Y" forms.
func SplitTrailingContainer(s Scope, input string) (itemPart string, cm Match, ok bool) {
	// Explicit "from <container>".
	if left, right, found := splitOnConnective(input, "from"); found {
		if m, matched := Resolve(s, right, KindRoomContainer, KindCorpse, KindPet); matched {
			return left, m, true
		}
		return "", Match{}, false
	}
	// No "from": try each "<item> <container>" split, longest item first.
	tokens := strings.Fields(input)
	for start := 1; start < len(tokens); start++ {
		left := strings.Join(tokens[:start], " ")
		right := strings.Join(tokens[start:], " ")
		if m, matched := Resolve(s, right, KindRoomContainer, KindCorpse, KindPet); matched {
			return left, m, true
		}
	}
	return "", Match{}, false
}

// ResolveItem is the shared get/drop/look-item ladder. It resolves an item that
// may live in a trailing container / corpse ("get X from Y" or "get X Y"), or on
// the floor / in inventory.
func ResolveItem(s Scope, input string) (Match, bool) {
	if itemPart, cm, ok := SplitTrailingContainer(s, input); ok {
		if m, ok2 := lootFromContainer(s, cm, itemPart); ok2 {
			return m, true
		}
		// A trailing container matched but held no such item — fall through so a
		// bare floor/inventory item of that literal name can still resolve.
	}
	return Resolve(s, input, KindFloorItem, KindInventoryItem)
}
```

Note: `SplitTrailingContainer` includes `KindPet`; `lootFromContainer` still only
handles container+corpse (pet item-lookup lives in `get.go`), so `ResolveItem`'s
pet case falls through to floor/inventory — acceptable, since `get.go` (not
`ResolveItem`) owns pet loot. This closes the Stage-0 pet deferral at the split
level.

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/parser/ -v`
Expected: PASS (new tests + all Stage-0 tests, including `TestResolveItem_*`).

- [ ] **Step 5: Commit**

```bash
git add internal/parser/helpers.go internal/parser/helpers_test.go
git commit -m "feat(parser): expose SplitTrailingContainer (item/container split) + pet in detection"
```

---

## Task 2: Lock `get`'s composition behavior with characterization tests

Before touching `get.go`, add any missing regression coverage for the paths the
refactor will run through, so drift is caught. `TestGet_CorpseLoot` already
covers corpse composition; add room-container `get` + the hidden-container gate.

**Files:**
- Create: `internal/usercommands/get_container_test.go`

- [ ] **Step 1: Write the characterization tests**

`internal/usercommands/get_container_test.go`:

```go
package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGet_RoomContainer_Composition locks the "get <item> <container>" and
// "get <item> from <container>" paths that Stage 1 refactors.
func TestGet_RoomContainer_Composition(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	user, room := getTestUserAndRoom(t)

	seedChest := func() {
		room.Containers = map[string]rooms.Container{
			"wooden chest": {Items: []items.Item{items.New(10001)}}, // Iron Sword
		}
	}

	t.Run("no_from", func(t *testing.T) {
		seedChest()
		user.Character.Items = nil
		handled, err := Get("sword wooden chest", user, room, 0)
		assert.True(t, handled)
		assert.NoError(t, err)
		assert.Len(t, user.Character.Items, 1, "`get sword wooden chest` should take from the chest")
	})

	t.Run("explicit_from", func(t *testing.T) {
		seedChest()
		user.Character.Items = nil
		handled, err := Get("iron sword from wooden chest", user, room, 0)
		assert.True(t, handled)
		assert.NoError(t, err)
		assert.Len(t, user.Character.Items, 1)
	})
}

// TestGet_HiddenContainerGate locks that an undiscovered hidden container is NOT
// lootable — the discovery gate must survive the refactor.
func TestGet_HiddenContainerGate(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	user, room := getTestUserAndRoom(t)

	room.Containers = map[string]rooms.Container{
		"secret cache": {Hidden: true, Items: []items.Item{items.New(10001)}},
	}
	user.Character.Items = nil

	handled, err := Get("sword secret cache", user, room, 0)
	assert.True(t, handled)
	assert.NoError(t, err)
	require.Len(t, user.Character.Items, 0, "an undiscovered hidden container must not be lootable")
}
```

- [ ] **Step 2: Run — these characterize CURRENT behavior, so they should PASS now**

Run: `go test ./internal/usercommands/ -run 'TestGet_RoomContainer_Composition|TestGet_HiddenContainerGate' -v`
Expected: PASS against the current (pre-refactor) `get.go`. If any fails, STOP —
that reveals current behavior differs from the assumption; investigate before
refactoring.

- [ ] **Step 3: Commit**

```bash
git add internal/usercommands/get_container_test.go
git commit -m "test(get): characterize room-container + hidden-gate composition before refactor"
```

---

## Task 3: Route `get`'s detection through `SplitTrailingContainer`

Replace **only** the container/corpse/pet detection block in `get.go` (the code
that sets `containerName` / `corpseIdx` / `petUserId` / `rest` from the trailing
args) with a single `parser.SplitTrailingContainer` call. Keep every branch body
(corpse loot, container, pet, floor) and all gates exactly as they are.

**Files:**
- Modify: `internal/usercommands/get.go` (the detection block, currently
  ~lines 243–308: the `FindContainerByName` + pet + corpse-detection ladder)

- [ ] **Step 1: Replace the detection block**

Find the block that currently begins with
`containerName = room.FindContainerByName(args[len(args)-1])` and ends after the
corpse-detection `if containerName == `` && petUserId == 0 { ... }` (the
"from"-split + trailing-word corpse logic). Replace that whole block with:

```go
	// Composition detection (Stage 1): the parser owns "is the trailing span a
	// container / corpse / pet, and what's the item span?". The branch bodies
	// below (gates + transfer) are unchanged.
	if len(args) >= 2 {
		scope := parser.Scope{User: user, Room: room}
		if itemPart, cm, ok := parser.SplitTrailingContainer(scope, rest); ok {
			switch cm.Kind {
			case parser.KindRoomContainer:
				// Preserve the hidden-container discovery gate.
				if c, exists := room.Containers[cm.ContainerName]; !exists ||
					(c.Hidden && (user == nil || !user.Character.HasDiscovery(room.RoomId, cm.ContainerName))) {
					// Not lootable / undiscovered: leave detection unset so the
					// input falls through to the floor/noun path unchanged.
				} else {
					containerName = cm.ContainerName
					getFromStash = false
					rest = itemPart
				}
			case parser.KindCorpse:
				corpseIdx = cm.CorpseIdx
				getFromStash = false
				rest = itemPart
			case parser.KindPet:
				petUserId = cm.UserId
				if petUserId > 0 && petUserId != user.UserId {
					user.SendText(messaging.CategorySystem, `You can't do that!`)
					return true, nil
				}
				getFromStash = false
				rest = itemPart
			}
		}
	}
```

Keep everything before this block (the `all` branch, `all.item`, stash/ground
detection, bag/bandolier gets) and everything after it (the `corpseIdx >= 0`,
pet, `containerName != ""`, and floor/inventory branches) exactly as-is. Add
`"github.com/GoMudEngine/GoMud/internal/parser"` to the imports.

Note: the bag/bandolier-source detection (`isBagGet`/`isBandolierGet`) runs
BEFORE this block and `return`s on match, so component-bag/bandolier gets are
untouched. This block only runs when those did not claim the input.

- [ ] **Step 2: Run the full `get` regression suite**

Run: `go test ./internal/usercommands/ -run 'TestGet' -v`
Expected: PASS — `TestGet` (incl. bandolier/container-word subtests),
`TestGet_CorpseLoot`, `TestGet_RoomContainer_Composition`,
`TestGet_HiddenContainerGate` all green. If any fail, a gate was dropped —
compare against the checklist and the pre-refactor branch bodies.

- [ ] **Step 3: Build + broader smoke**

Run: `go build ./... && go test ./internal/usercommands/ ./internal/parser/`
Expected: build OK, both packages green.

- [ ] **Step 4: Commit**

```bash
git add internal/usercommands/get.go
git commit -m "refactor(get): route container/corpse/pet detection through parser.SplitTrailingContainer"
```

---

## Task 4: Remove the now-dead detection helpers from `get.go`

After Task 3, the old inline detection may leave unused locals or now-redundant
logic. Remove only what is provably dead; do not touch branch bodies.

**Files:**
- Modify: `internal/usercommands/get.go`

- [ ] **Step 1: Identify dead code**

Run: `go vet ./internal/usercommands/` and `go build ./internal/usercommands/`
Look for "declared and not used" or now-unreachable detection code left behind
by Task 3 (e.g. an orphaned `fromIdx` loop). Remove only those.

- [ ] **Step 2: Verify still green**

Run: `go test ./internal/usercommands/ -run 'TestGet' && go build ./...`
Expected: PASS, build OK.

- [ ] **Step 3: Commit (skip if nothing was dead)**

```bash
git add internal/usercommands/get.go
git commit -m "refactor(get): drop dead detection code superseded by the parser split"
```

---

## Definition of Done (Stage 1)

- `get.go`'s container/corpse/pet detection is a single `SplitTrailingContainer`
  call; the branch bodies and all gates are unchanged.
- `TestGet`, `TestGet_CorpseLoot`, `TestGet_RoomContainer_Composition`,
  `TestGet_HiddenContainerGate`, and all `internal/parser` tests are green.
- `go build ./...` clean. No behavior change for any existing input; the
  corpse-loot composition (`get all corpse` / `get <item> corpse`) still works.

## Divergences From Spec (this stage)

- **Split helper, not full `ResolveItem` swap.** Per the composition-only
  re-scope, `get` keeps all its policy; only the detection ladder moves to the
  parser. `ResolveItem` remains for future callers but `get` uses the lighter
  `SplitTrailingContainer` to preserve its existing gate/transfer branches.
- **Pet deferral closed at the split level:** `SplitTrailingContainer` detects
  `KindPet`; the pet *item-lookup* stays in `get.go` (where the ownership check
  lives), so `lootFromContainer` still needs no pet branch.
- **Gates stay in the command** (hidden-container discovery re-checked in
  `get.go` after the parser resolves the container) — consistent with the spec's
  "resolver finds, command gates" principle.
