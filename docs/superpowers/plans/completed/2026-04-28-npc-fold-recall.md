# NPC Fold-Recall Implementation Plan (Stage 3.0d)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Generalize `fold-anchor` and `fold-recall` resolvers to accept `actions.Actor`, wire mobs into the same spell dispatch as players, add a YAML `fold_anchor_room` field for spawn-time anchor pre-stamping, and ship Edrin + caravan crew with anchors set.

**Architecture:** Refactor the player-only resolvers in `internal/hooks/spell_foldanchor.go` and `internal/hooks/spell_foldrecall.go` to take `actions.Actor`. Branch the teleport step on `actor.IsPlayer()` — players go through `rooms.MoveToRoom`; mobs go through `oldRoom.RemoveMob → newRoom.AddMob → mob.Character.RoomId = anchor`. Add a Go-hook switch at the top of `resolveMobSpell` that wraps the mob in `actions.NewMobActorInRoom` and dispatches to the same shared resolvers; update the player path to wrap user in `actions.NewUserActorInRoom`. Add `FoldAnchorRoom int` to the Mob struct, stamp it into `MiscData["fold-anchor-room"]` at spawn.

**Tech Stack:** Go (server). YAML data files. Existing systems: `characters.MiscData`, `actions.Actor` adapters, `mobs.NewMobByIdFresh`, the mob tactics dispatcher (which already routes `cast <spell>` through `mob.Command(...)` to `resolveMobSpell`).

**Spec:** `docs/superpowers/specs/2026-04-28-npc-fold-recall-design.md`

---

## File Structure

| Action | File | Responsibility |
|---|---|---|
| MODIFY | `internal/mobs/mobs.go` | Add `FoldAnchorRoom int` field to the Mob struct; stamp MiscData in `NewMobByIdFresh` |
| MODIFY | `internal/hooks/spell_foldanchor.go` | Resolver takes `actions.Actor` instead of `*users.UserRecord` |
| MODIFY | `internal/hooks/spell_foldrecall.go` | Validator + resolver take `actions.Actor`; new `teleportActor(actor, toRoomId)` helper branches on player/mob |
| MODIFY | `internal/hooks/spell_resolution.go` | Player path wraps user in `NewUserActorInRoom`; `resolveMobSpell` adds Go-hook switch wrapping mob in `NewMobActorInRoom` |
| CREATE | `internal/hooks/spell_foldanchor_test.go` | Unit tests covering both UserActor and MobActor cases |
| CREATE | `internal/hooks/spell_foldrecall_test.go` | Unit tests for validate + resolve covering missing-anchor, anchor-at-current-room, allow_recall block, successful teleport (player + mob) |
| MODIFY | `_datafiles/world/dogmud/mobs/marches_spur_road/275-old_edrin.yaml` | Add `fold_anchor_room: 4037`, spellbook entry, panic-recall tactic |
| MODIFY | `_datafiles/world/dogmud/mobs/thornwall_city/357-ketil.yaml` | Add `fold_anchor_room: 465`, spellbook entry, panic-recall tactic |
| MODIFY | `_datafiles/world/dogmud/mobs/thornwall_city/358-marta.yaml` | Same treatment as Ketil |
| MODIFY | `_datafiles/world/dogmud/mobs/thornwall_city/359-lars.yaml` | Same treatment as Ketil |
| MODIFY | `docs/schemas/mob.md` | Document the new `fold_anchor_room` field |
| MODIFY | `PATCH_NOTES.md` | Stage 3.0d dev-only entry |

---

## Task 1: Add `FoldAnchorRoom` field to Mob struct + schema doc

**Files:**
- Modify: `internal/mobs/mobs.go` (Mob struct, around line 79)
- Modify: `docs/schemas/mob.md` (mob field reference table)

This is a structural change with no behavior. Stamping the MiscData happens in Task 2.

- [ ] **Step 1: Locate the Mob struct**

Run: `grep -n "^type Mob struct\|^	Groups\b" internal/mobs/mobs.go | head -5`
Expected: a `type Mob struct` line and a `Groups` field line (~line 79). Note both line numbers.

- [ ] **Step 2: Add the new field below `Groups`**

In `internal/mobs/mobs.go`, immediately AFTER the `Groups []string` field declaration in the Mob struct, insert:

```go
	FoldAnchorRoom  int      `yaml:"fold_anchor_room,omitempty"` // Spawn-time fold-recall anchor (room ID)
```

Match the existing field-comment style. Verify alignment of the struct tag column with neighboring fields.

- [ ] **Step 3: Verify build still compiles**

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 4: Document the field in mob.md**

In `docs/schemas/mob.md`, find the table row for `groups`. Immediately AFTER that row, insert:

```
| `fold_anchor_room` | int | no | Room ID stamped into `MiscData["fold-anchor-room"]` at spawn. Lets the mob `cast fold-recall` to that room without first casting `fold-anchor`. Resolver: `internal/hooks/spell_foldrecall.go`. |
```

- [ ] **Step 5: Commit**

```bash
git add internal/mobs/mobs.go docs/schemas/mob.md
git commit -m "$(cat <<'EOF'
feat(mobs): add FoldAnchorRoom YAML field

Pre-stamps a mob's fold-recall anchor at spawn so NPCs can use the
existing fold-recall spell without first casting fold-anchor. Stage
3.0d (NPC fold-recall).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Stamp `MiscData["fold-anchor-room"]` at mob spawn

**Files:**
- Modify: `internal/mobs/mobs.go` (extract a `stampFoldAnchor` helper + call it from `newMobByIdInternal`)
- Create: `internal/mobs/fold_anchor_spawn_test.go`

TDD on a small extracted helper to keep the unit test independent of the global mob registry, world boot, and `Validate()` side effects.

- [ ] **Step 1: Write the failing test**

Create `internal/mobs/fold_anchor_spawn_test.go`:

```go
package mobs

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/stretchr/testify/assert"
)

// stampFoldAnchor must write the room ID into MiscData when the field is
// set. Unit-tested in isolation from the rest of the spawn pipeline so
// the test doesn't depend on world boot / Validate / RegisterMobShop.
func TestStampFoldAnchor_Stamps(t *testing.T) {
	c := characters.New()
	stampFoldAnchor(c, 4037)

	got := c.GetMiscData("fold-anchor-room")
	assert.Equal(t, 4037, got, "MiscData should hold the anchor room ID")
}

// stampFoldAnchor must be a no-op when the field is zero (the YAML default)
// — otherwise every mob would get a spurious anchor at room 0.
func TestStampFoldAnchor_NoOpWhenZero(t *testing.T) {
	c := characters.New()
	stampFoldAnchor(c, 0)

	got := c.GetMiscData("fold-anchor-room")
	assert.Nil(t, got, "MiscData must NOT be set when FoldAnchorRoom is zero")
}

// stampFoldAnchor must also no-op for negative values (defensive — YAML
// authors might typo a negative).
func TestStampFoldAnchor_NoOpWhenNegative(t *testing.T) {
	c := characters.New()
	stampFoldAnchor(c, -1)

	got := c.GetMiscData("fold-anchor-room")
	assert.Nil(t, got, "MiscData must NOT be set for negative anchor IDs")
}
```

- [ ] **Step 2: Run the test and confirm it fails**

Run: `go test ./internal/mobs/ -run TestStampFoldAnchor -v`
Expected: FAIL with a compile error like `undefined: stampFoldAnchor`.

- [ ] **Step 3: Add the helper + call site**

In `internal/mobs/mobs.go`, add the helper function. A good location is immediately AFTER `newMobByIdInternal` ends (around line 555) — co-located with its only caller:

```go
// stampFoldAnchor pre-stamps a mob's fold-recall anchor in MiscData. Called
// from newMobByIdInternal at spawn time. No-op when anchorRoom <= 0 so the
// default YAML value (omitted field) doesn't create a spurious anchor at
// room 0. Stage 3.0d.
func stampFoldAnchor(c *characters.Character, anchorRoom int) {
	if anchorRoom <= 0 {
		return
	}
	c.SetMiscData("fold-anchor-room", anchorRoom)
}
```

Then wire the call site. In `newMobByIdInternal`, after the line `RegisterMobShop(&mob)` (~line 545) and BEFORE `mobInstancesMu.Lock()` (~line 548), insert:

```go
		// Stage 3.0d: pre-stamp fold-recall anchor if the YAML template set one.
		stampFoldAnchor(&mob.Character, m.FoldAnchorRoom)

```

Note: `m` is the template (line 338 — `m, ok := mobs[int(mobId)]`); `mob` is the freshly constructed instance (line 348 — `mob := *m`). Match the existing tab indentation (this code is inside the `if ok {` block, so two tab levels deep).

- [ ] **Step 4: Re-run the tests and confirm they pass**

Run: `go test ./internal/mobs/ -run TestStampFoldAnchor -v`
Expected: all 3 tests PASS.

- [ ] **Step 5: Run the full mobs package suite**

Run: `go test ./internal/mobs/...`
Expected: all PASS, no regressions.

- [ ] **Step 6: Commit**

```bash
git add internal/mobs/mobs.go internal/mobs/fold_anchor_spawn_test.go
git commit -m "$(cat <<'EOF'
feat(mobs): stamp fold-anchor MiscData at spawn

New stampFoldAnchor helper reads the FoldAnchorRoom YAML field and
writes MiscData["fold-anchor-room"] on the freshly spawned mob
instance, so it can immediately cast fold-recall without first
casting fold-anchor. Stage 3.0d.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Refactor `resolveFoldAnchor` to take `actions.Actor`

**Files:**
- Modify: `internal/hooks/spell_foldanchor.go`
- Create: `internal/hooks/spell_foldanchor_test.go`

TDD. Write the failing tests first; the player + mob assertions both exercise the new actor signature.

- [ ] **Step 1: Write the failing tests**

Create `internal/hooks/spell_foldanchor_test.go`:

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
)

// fakeActor is a minimal Actor implementation for unit tests. It records
// SendText / SendRoomText calls so assertions can verify behavior without
// touching the user/mob packages.
type fakeActor struct {
	char       *characters.Character
	room       *rooms.Room
	name       string
	isPlayer   bool
	userId     int
	mobInstId  int
	selfTexts  []string
	roomTexts  []string
}

func (f *fakeActor) GetCharacter() *characters.Character { return f.char }
func (f *fakeActor) GetRoom() *rooms.Room                { return f.room }
func (f *fakeActor) SendText(msg string)                 { f.selfTexts = append(f.selfTexts, msg) }
func (f *fakeActor) SendRoomText(msg string, excludeSelf bool) {
	f.roomTexts = append(f.roomTexts, msg)
}
func (f *fakeActor) SendRoomCommunication(msg string, excludeSelf bool) {}
func (f *fakeActor) GetName() string                                    { return f.name }
func (f *fakeActor) IsPlayer() bool                                     { return f.isPlayer }
func (f *fakeActor) GetUserId() int                                     { return f.userId }
func (f *fakeActor) GetMobInstanceId() int                              { return f.mobInstId }
func (f *fakeActor) AddBuff(buffId int, source string)                  {}
func (f *fakeActor) OnSkillUse(skillName string) bool                   { return false }
func (f *fakeActor) OnStatUse(statName string) bool                     { return false }
func (f *fakeActor) OnCriticalSuccess(skillName string)                 {}
func (f *fakeActor) OnCriticalFailure(skillName string)                 {}

// compile-time check
var _ actions.Actor = (*fakeActor)(nil)

// Resolving fold-anchor must write the actor's current room ID into
// MiscData["fold-anchor-room"]. Works for both player and mob actors.
func TestResolveFoldAnchor_PlayerActor_SetsMiscData(t *testing.T) {
	c := characters.New()
	c.RoomId = 4036

	a := &fakeActor{
		char:     c,
		name:     "TestPlayer",
		isPlayer: true,
		userId:   42,
	}

	resolveFoldAnchor(a)

	got := c.GetMiscData("fold-anchor-room")
	assert.Equal(t, 4036, got, "MiscData should hold the actor's current room ID")
	assert.Len(t, a.selfTexts, 1, "player should receive one self message")
	assert.Len(t, a.roomTexts, 1, "room should receive one shimmer broadcast")
}

func TestResolveFoldAnchor_MobActor_SetsMiscData(t *testing.T) {
	c := characters.New()
	c.RoomId = 4036

	a := &fakeActor{
		char:      c,
		name:      "Old Edrin",
		isPlayer:  false,
		mobInstId: 99,
	}

	resolveFoldAnchor(a)

	got := c.GetMiscData("fold-anchor-room")
	assert.Equal(t, 4036, got, "MiscData should hold the actor's current room ID")
	assert.Len(t, a.roomTexts, 1, "room should still get the shimmer broadcast for mob actors")
}
```

- [ ] **Step 2: Run the test and confirm it fails**

Run: `go test ./internal/hooks/ -run TestResolveFoldAnchor -v`
Expected: FAIL with a compile error like `cannot use a (*hooks.fakeActor) as *users.UserRecord` because `resolveFoldAnchor` still has the old signature.

- [ ] **Step 3: Refactor `resolveFoldAnchor` to actor signature**

Replace the entire body of `internal/hooks/spell_foldanchor.go` with:

```go
package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
)

func resolveFoldAnchor(actor actions.Actor) {
	char := actor.GetCharacter()
	if char == nil {
		return
	}
	roomId := char.RoomId
	char.SetMiscData("fold-anchor-room", roomId)

	actor.SendText(`A Chrysalis anchor locks into place here. ` +
		`Cast <ansi fg="command">fold-recall</ansi> from elsewhere to return.`)

	actor.SendRoomText(fmt.Sprintf(
		`A faint shimmer marks where <ansi fg="username">%s</ansi> has set an anchor.`,
		actor.GetName()), true)
}
```

Note the imports change — drop `rooms` and `users`, add `actions`.

- [ ] **Step 4: Update the player call site so the build still passes**

In `internal/hooks/spell_resolution.go`, find the case for `fold-anchor` (around line 201):

```go
		case "fold-anchor":
			resolveFoldAnchor(user)
			return
```

Change it to wrap the user in an actor:

```go
		case "fold-anchor":
			resolveFoldAnchor(actions.NewUserActorInRoom(user, room))
			return
```

If `actions` isn't yet in the file's import block, add `"github.com/GoMudEngine/GoMud/internal/actions"` to it.

- [ ] **Step 5: Re-run the tests and confirm they pass**

Run: `go build ./... && go test ./internal/hooks/ -run TestResolveFoldAnchor -v`
Expected: build clean, both tests PASS.

- [ ] **Step 6: Run the full hooks package suite**

Run: `go test ./internal/hooks/...`
Expected: all PASS, no regressions.

- [ ] **Step 7: Commit**

```bash
git add internal/hooks/spell_foldanchor.go internal/hooks/spell_foldanchor_test.go internal/hooks/spell_resolution.go
git commit -m "$(cat <<'EOF'
refactor(hooks): generalize resolveFoldAnchor to actions.Actor

Player path wraps user in NewUserActorInRoom; mobs will go through
NewMobActorInRoom in the next task. Adds fakeActor test helper for
unit-testing actor-shaped resolvers in isolation. Stage 3.0d.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Refactor `validateFoldRecall` + `resolveFoldRecall` to take `actions.Actor` (with `teleportActor` helper)

**Files:**
- Modify: `internal/hooks/spell_foldrecall.go`
- Create: `internal/hooks/spell_foldrecall_test.go`

TDD. The teleport branches on `actor.IsPlayer()` so the tests cover both cases.

- [ ] **Step 1: Write the failing tests**

Create `internal/hooks/spell_foldrecall_test.go`:

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/stretchr/testify/assert"
)

// validateFoldRecall must reject when no anchor has been set.
func TestValidateFoldRecall_NoAnchor_Rejects(t *testing.T) {
	c := characters.New()
	c.RoomId = 4036

	a := &fakeActor{char: c, name: "TestPlayer", isPlayer: true, userId: 42}

	ok := validateFoldRecall(a)

	assert.False(t, ok, "validate must reject when no anchor exists")
	assert.NotEmpty(t, a.selfTexts, "actor should be told why it failed")
}

// validateFoldRecall must reject when the actor is already in the anchor room.
func TestValidateFoldRecall_AlreadyAtAnchor_Rejects(t *testing.T) {
	c := characters.New()
	c.RoomId = 4037
	c.SetMiscData("fold-anchor-room", 4037)

	a := &fakeActor{char: c, name: "TestPlayer", isPlayer: true, userId: 42}

	ok := validateFoldRecall(a)

	assert.False(t, ok, "validate must reject when actor is already on the anchor")
	assert.NotEmpty(t, a.selfTexts, "actor should be told why it failed")
}

// validateFoldRecall passes when an anchor is set and not the current room.
func TestValidateFoldRecall_HappyPath_Passes(t *testing.T) {
	c := characters.New()
	c.RoomId = 4036
	c.SetMiscData("fold-anchor-room", 4037)

	a := &fakeActor{char: c, name: "TestPlayer", isPlayer: true, userId: 42}

	ok := validateFoldRecall(a)

	assert.True(t, ok, "validate must pass when anchor exists and is not the current room")
}

// resolveFoldRecall: short-circuit when no anchor is set (defensive — validate
// should have caught it). Confirm the resolver doesn't blow up.
func TestResolveFoldRecall_NoAnchor_NoOp(t *testing.T) {
	c := characters.New()
	c.RoomId = 4036

	a := &fakeActor{char: c, name: "TestPlayer", isPlayer: true, userId: 42}

	// must not panic
	resolveFoldRecall(a)

	assert.NotEmpty(t, a.selfTexts, "actor should receive a 'fold collapses' message")
}
```

These are the easy unit-testable cases. The full teleport (room mutation) is awkward to unit-test against the live `rooms` package without spinning up the world; we cover that at the integration level via the in-game smoke test in Task 7.

- [ ] **Step 2: Run the tests and confirm they fail**

Run: `go test ./internal/hooks/ -run TestValidateFoldRecall -v`
Expected: FAIL with compile error or signature mismatch — `validateFoldRecall` still takes `*users.UserRecord`.

- [ ] **Step 3: Refactor `spell_foldrecall.go` to actor signature + teleport helper**

Replace the entire contents of `internal/hooks/spell_foldrecall.go` with:

```go
package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// validateFoldRecall is called during the onCast phase. Returns false to
// abort the spell.
func validateFoldRecall(actor actions.Actor) bool {
	char := actor.GetCharacter()
	if char == nil {
		return false
	}
	currentRoomId := char.RoomId

	// Check if recall is blocked in the current room (instanced zones with
	// allow_recall: false).
	if currentRoom := rooms.LoadRoom(currentRoomId); currentRoom != nil {
		if blocked, ok := currentRoom.GetTempData("allow_recall").(bool); ok && !blocked {
			actor.SendText("Something about this place prevents you from recalling.")
			return false
		}
	}

	anchorRoom := getMiscDataInt(char, "fold-anchor-room")
	if anchorRoom <= 0 {
		actor.SendText(`You reach for the Veil, but there is no anchor to ` +
			`pull you. Set one first with ` +
			`<ansi fg="command">cast fold-anchor</ansi>.`)
		return false
	}

	if anchorRoom == currentRoomId {
		actor.SendText("You are already standing on your anchor.")
		return false
	}

	return true
}

// resolveFoldRecall is called during the onMagic phase.
func resolveFoldRecall(actor actions.Actor) {
	char := actor.GetCharacter()
	if char == nil {
		return
	}
	anchorRoom := getMiscDataInt(char, "fold-anchor-room")
	currentRoomId := char.RoomId

	if anchorRoom <= 0 || anchorRoom == currentRoomId {
		actor.SendText("The fold collapses — no valid anchor found.")
		return
	}

	// Clear combat state before teleporting.
	char.EndAggro()

	// Departure broadcast on the current room.
	actor.SendRoomText(fmt.Sprintf(
		`<ansi fg="username">%s</ansi> folds through the Veil and vanishes!`,
		actor.GetName()), true)

	// Move the actor.
	if !teleportActor(actor, anchorRoom) {
		actor.SendText("The fold collapses — no valid anchor found.")
		return
	}

	actor.SendText("You fold through the Veil and arrive at your anchor point!")

	// Arrival broadcast on the new room.
	if newRoom := rooms.LoadRoom(anchorRoom); newRoom != nil {
		newRoom.SendText(fmt.Sprintf(
			`<ansi fg="username">%s</ansi> folds through the Veil and appears!`,
			actor.GetName()), actor.GetUserId())
	}
}

// teleportActor moves the actor to the destination room. For players this
// goes through rooms.MoveToRoom (handles cross-zone bookkeeping). For mobs
// it manipulates room membership directly. Returns false if the destination
// room can't be loaded.
func teleportActor(actor actions.Actor, toRoomId int) bool {
	if actor.IsPlayer() {
		// Players: existing helper handles the cross-zone case.
		if err := rooms.MoveToRoom(actor.GetUserId(), toRoomId); err != nil {
			return false
		}
		return true
	}

	// Mobs: manual room membership update.
	char := actor.GetCharacter()
	fromRoom := rooms.LoadRoom(char.RoomId)
	toRoom := rooms.LoadRoom(toRoomId)
	if toRoom == nil {
		return false
	}
	instId := actor.GetMobInstanceId()
	if fromRoom != nil {
		fromRoom.RemoveMob(instId)
	}
	toRoom.AddMob(instId)
	char.RoomId = toRoomId
	return true
}

// getMiscDataInt retrieves an integer stored in MiscData, handling both int
// and float64 (the latter can occur after YAML round-trips).
func getMiscDataInt(char *characters.Character, key string) int {
	val := char.GetMiscData(key)
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case int:
		return v
	case float64:
		return int(v)
	}
	return 0
}
```

Add the missing import for `getMiscDataInt`:

```go
import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)
```

- [ ] **Step 4: Update the player call site so the build passes**

In `internal/hooks/spell_resolution.go`, find the case for `fold-recall` (around line 204):

```go
		case "fold-recall":
			resolveFoldRecall(user)
			return
```

Change it to wrap the user in an actor:

```go
		case "fold-recall":
			resolveFoldRecall(actions.NewUserActorInRoom(user, room))
			return
```

(The `actions` import was added in Task 3; verify it's still there.)

- [ ] **Step 5: Confirm there are no `validateFoldRecall` callers**

`validateFoldRecall` is currently dead code — defined but never invoked (the cast pipeline goes straight to `resolveFoldRecall`, which has its own no-anchor early return). Verify:

```bash
grep -rn "validateFoldRecall" internal/
```

Expected: only the function definition itself in `internal/hooks/spell_foldrecall.go`. No call sites to update. Refactoring it to the actor signature is still useful (future callers can use it; the tests in this task exercise it), but no other files need to change.

- [ ] **Step 6: Build + run tests**

Run: `go build ./... && go test ./internal/hooks/ -run "TestValidateFoldRecall|TestResolveFoldRecall" -v`
Expected: build clean, all 4 tests PASS.

- [ ] **Step 7: Run full hooks suite**

Run: `go test ./internal/hooks/...`
Expected: all PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/hooks/spell_foldrecall.go internal/hooks/spell_foldrecall_test.go internal/hooks/spell_resolution.go
git commit -m "$(cat <<'EOF'
refactor(hooks): generalize fold-recall validate+resolve to actions.Actor

New teleportActor helper branches: players go through rooms.MoveToRoom,
mobs through manual room membership updates. The room-side broadcast
and the existing allow_recall gate behave identically for both actor
types. Stage 3.0d.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Add Go-hook switch to `resolveMobSpell`

**Files:**
- Modify: `internal/hooks/spell_resolution.go` (`resolveMobSpell` around line 844)

`resolveMobSpell` currently has no spell-id dispatch — it routes by `spellData.Type`. Fold-anchor and fold-recall don't fit that model (they're position-mutating, not damage/heal). Add a Go-hook switch at the top of the function, mirroring the pattern in the player resolver (`spell_resolution.go:200-216`).

- [ ] **Step 1: Locate `resolveMobSpell`**

Run: `grep -n "^func resolveMobSpell" internal/hooks/spell_resolution.go`
Expected: a single match around line 844.

- [ ] **Step 2: Insert the Go-hook switch at the top of the function**

Find the line right after the function signature:

```go
func resolveMobSpell(mob *mobs.Mob, cs *characters.CastingState, spellData *spells.SpellData, room *rooms.Room) {
	skillLevel := mob.Character.GetSkillLevel(skills.Spellcasting)
```

Insert this block BEFORE `skillLevel := ...`:

```go
	// Go spell hooks — dispatch position-mutating / non-target spells before
	// the type-based effect routing below. Mirrors the player path in
	// resolveSpell. Stage 3.0d.
	switch cs.SpellId {
	case "fold-anchor":
		resolveFoldAnchor(actions.NewMobActorInRoom(mob, room))
		return
	case "fold-recall":
		resolveFoldRecall(actions.NewMobActorInRoom(mob, room))
		return
	}

```

If the `actions` import isn't yet in this file's import block, add it:

```go
"github.com/GoMudEngine/GoMud/internal/actions"
```

- [ ] **Step 3: Build + run hooks tests**

Run: `go build ./... && go test ./internal/hooks/...`
Expected: clean build, all PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/spell_resolution.go
git commit -m "$(cat <<'EOF'
feat(hooks): wire fold-anchor/fold-recall into resolveMobSpell

Adds a Go-hook switch at the top of resolveMobSpell that wraps the
casting mob in NewMobActorInRoom and dispatches to the same shared
resolvers the player path uses. Mob-cast parity for the two
position-mutating spells. Stage 3.0d.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Wire YAML data — Edrin + caravan crew

**Files:**
- Modify: `_datafiles/world/dogmud/mobs/marches_spur_road/275-old_edrin.yaml`
- Modify: `_datafiles/world/dogmud/mobs/thornwall_city/357-ketil.yaml`
- Modify: `_datafiles/world/dogmud/mobs/thornwall_city/358-marta.yaml`
- Modify: `_datafiles/world/dogmud/mobs/thornwall_city/359-lars.yaml`

- [ ] **Step 1: Edit Edrin (mob 275)**

In `_datafiles/world/dogmud/mobs/marches_spur_road/275-old_edrin.yaml`:

(a) Add `fold_anchor_room: 4037` near the top of the file, after the `groups:` block (or wherever it fits the existing field ordering). For example, between `scriptag:` (line 15) and `idlecommands:` (line 16).

(b) In the `tactics:` list, ADD a new entry at the top (above the existing `health_below:25 → flee` tactic):

```yaml
  - trigger: health_below:30
    action: cast fold-recall
    priority: 13
```

Keep all existing tactics entries untouched below it.

(c) In the `spellbook:` block (lines ~83-90), ADD a new entry:

```yaml
  fold-recall: 30
```

(Slot in alphabetical or by-skill-level order — match the existing convention.)

- [ ] **Step 2: Edit Ketil (mob 357)**

In `_datafiles/world/dogmud/mobs/thornwall_city/357-ketil.yaml`:

(a) Add `fold_anchor_room: 465` near the top of the file (consistent with Edrin's placement).

(b) Add to `spellbook:` (create the block if Ketil doesn't have one):

```yaml
spellbook:
  fold-recall: 20
```

(c) Add to `tactics:` (create the block if Ketil doesn't have one). Set the priority high enough that it fires above any existing flee/heal triggers — bump to `priority: 13` to match Edrin:

```yaml
tactics:
  - trigger: health_below:30
    action: cast fold-recall
    priority: 13
```

If Ketil already has a `tactics:` block with a `flee` at priority 12 or lower, this insert is correct. If he has a higher-priority entry, raise this one accordingly so recall wins.

- [ ] **Step 3: Edit Marta (mob 358)**

Same three edits to `_datafiles/world/dogmud/mobs/thornwall_city/358-marta.yaml`:

```yaml
fold_anchor_room: 465
```

```yaml
spellbook:
  fold-recall: 20
```

```yaml
tactics:
  - trigger: health_below:30
    action: cast fold-recall
    priority: 13
```

- [ ] **Step 4: Edit Lars (mob 359)**

Same three edits to `_datafiles/world/dogmud/mobs/thornwall_city/359-lars.yaml`:

```yaml
fold_anchor_room: 465
```

```yaml
spellbook:
  fold-recall: 20
```

```yaml
tactics:
  - trigger: health_below:30
    action: cast fold-recall
    priority: 13
```

- [ ] **Step 5: Boot test**

Run: `timeout 30 go run . 2>&1 | grep -E "mobs.LoadDataFiles|panic" | head -5`
Expected: a line `mobs.LoadDataFiles() loadedCount=N` (count unchanged from baseline — we're editing existing mobs). No panic.

If the loader panics complaining about unknown YAML field `fold_anchor_room`, that means Task 1 didn't land — fix that first. If the panic mentions an unknown spell `fold-recall`, the spell YAML file is missing (but `_datafiles/world/dogmud/spells/fold-recall.yaml` exists per scout — verify it's still there).

- [ ] **Step 6: Commit**

```bash
git add _datafiles/world/dogmud/mobs/marches_spur_road/275-old_edrin.yaml _datafiles/world/dogmud/mobs/thornwall_city/357-ketil.yaml _datafiles/world/dogmud/mobs/thornwall_city/358-marta.yaml _datafiles/world/dogmud/mobs/thornwall_city/359-lars.yaml
git commit -m "$(cat <<'EOF'
feat(mobs): wire fold-recall on Edrin + caravan crew

- Edrin: fold_anchor_room: 4037 (back room, 1-west of his cottage),
  fold-recall in spellbook at skill 30, panic-recall tactic above flee
- Ketil/Marta/Lars: fold_anchor_room: 465 (Thornwall Market Square
  depot), fold-recall at skill 20, panic-recall tactic at priority 13

Edrin is the smoke-test rig; caravan crew get fold-recall as wipe
insurance. Stage 3.0d.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: PATCH_NOTES + smoke verification

**Files:**
- Modify: `PATCH_NOTES.md`

- [ ] **Step 1: Insert Stage 3.0d entry at the top of PATCH_NOTES.md**

After the title `# DOGMud Patch Notes`, insert a new entry above the existing 2026-04-28 Stage 3.0e entry:

```markdown
## 2026-04-28 — Stage 3.0d: NPC Fold-Recall (dev only)

**Note:** Dev-only landing. The full economy stack ships to prod (`master`)
as a coherent update once Stage 3.4 lands.

- `fold-anchor` and `fold-recall` resolvers now accept `actions.Actor`
  rather than `*users.UserRecord`. Mobs can cast both spells via the
  existing tactics dispatcher and the new Go-hook switch in
  `resolveMobSpell`. Player behavior is unchanged.
- New mob YAML field `fold_anchor_room: <roomId>` pre-stamps a mob's
  fold-recall anchor at spawn. The runtime is then identical to a
  player who already cast `fold-anchor`.
- Old Edrin (mob 275) gets `fold-recall` as a panic spell at
  `health_below:30` priority above his existing flee — he recalls to
  the cluttered back room (4037) when injured. Useful smoke-test rig
  for the new pipeline.
- Caravan crew Ketil/Marta/Lars (mobs 357-359) get the same treatment
  with anchor at the Thornwall Market Square depot (465). Wipe
  insurance for the bandit camp ambush — if their HP drops they
  recall instead of dying, keeping the restock service running.
- Stage 3.0d does NOT add forager NPCs or logistic recall triggers
  (e.g., `inventory_full → cast fold-recall`). Those are Stage 3.1's
  job. Caravan recall is individual, not group-aware: each crew
  member recalls on their own panic threshold.

```

- [ ] **Step 2: Build + boot**

Run: `go build ./... && timeout 30 go run . 2>&1 | head -50`
Expected: clean build, server passes the data-file load phase without panic.

- [ ] **Step 3: Manual smoke test — Edrin recall**

Connect to the local server. Travel to Edrin (Marches Spur Road, room 4036, "The Hermit's Cottage"). Then:

1. `look` — confirm Old Edrin is present in the room.
2. `attack edrin` — start combat. Use whatever attacks you have. Edrin will heal himself; that's fine, just keep hitting until his HP drops below 30%.
3. Watch for the room broadcast: "Old Edrin folds through the Veil and vanishes!"
4. After the broadcast, run `look` — confirm Edrin is no longer in the room.
5. Walk west into room 4037 (Cluttered Back Room). Run `look`.
6. Confirm Edrin is now in the back room. He may try to heal / re-buff per his other tactics — that's fine; just confirms he survived the recall.

Expected outcome: Edrin teleports to 4037 mid-combat, alive.

If Edrin DOESN'T fold-recall (he just flees, or dies outright):
- Verify the YAML edits landed (run `git diff HEAD~1 -- _datafiles/world/dogmud/mobs/marches_spur_road/275-old_edrin.yaml`).
- Verify the priority of fold-recall is HIGHER than flee (recall must win the tactic resolution at the same HP threshold).
- Verify the spell exists in his spellbook.
- Restart the server (mob templates are loaded at boot).

- [ ] **Step 4: Manual smoke test — caravan recall (optional)**

This requires positioning yourself at the bandit-camp ambush room on the North Road and waiting for the caravan to come through. Skip if testing is taking too long; the Edrin smoke test exercises the same code path. If you want to verify:

1. Watch the caravan in Thornwall Market Square (room 465). They should be present during `thornwall_dwell`.
2. Travel to the North Road bandit camp room (where the caravan engages bandits during transit).
3. Wait for the caravan to arrive and start combat.
4. Watch for any caravan crew member to fold-recall back to room 465 if their HP drops.

If the caravan wins the bandit fight cleanly without anyone dropping below 30% HP, the test is inconclusive but not failing. Skip and rely on Edrin smoke.

- [ ] **Step 5: Commit PATCH_NOTES**

```bash
git add PATCH_NOTES.md
git commit -m "$(cat <<'EOF'
docs(3.0d): patch notes for NPC fold-recall

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Out of scope reminder (from spec)

Push back if scope creep tries to add:
- Forager NPC wiring (Stage 3.1)
- Logistic recall trigger (`inventory_full → cast fold-recall`) — also 3.1
- Group-aware recall (caravan leader pulls followers along) — followers each have their own anchor + own panic trigger
- Cooldowns beyond the existing per-spell casting cost
- Anchor-room-side `allow_recall` validation (player parity = no validation on anchor side)
- Player UI for inspecting NPC anchors

## Done = ?

All 7 tasks complete, all commits landed on `development` branch, smoke test (Edrin recall) green. Per the multi-stage caravan/economy effort: this lands on `development` only. Nothing ships to `master` until Stage 3.4 lands.
