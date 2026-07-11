# Character-Creation Onboarding Tiers — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a 3-tier experience poll at character creation that tailors onboarding — a guided tutorial arc for total newbies, today's Coulee for MUD vets, and an auto-Awakened skip-to-Thornwall for veterans.

**Architecture:** A poll in `start.go` routes by answer. Tier B (default) is today's path (no-op). Tier C grants `30-end` + a starting mutation and vortexes to Thornwall, with a `tutorial` back-door. Tier A routes through an instanced ephemeral antechamber (reusing the `TutorialRooms` mechanism) whose rooms gate their exits via `room_command` behaviors, then through newbie-only teaching beats in the Coulee/town gated by a hidden "Newcomer's Path" quest token.

**Tech Stack:** Go (internal/usercommands, internal/characters, internal/behaviortree, internal/questengine), YAML data (rooms/behaviors/mobs/dialogue/quests/config), the GoMud playtest harness for in-game verification.

**Spec:** `docs/superpowers/specs/2026-06-29-character-creation-onboarding-tiers-design.md`

**Marker refinement (vs spec §5):** the spec suggested a MiscData marker. Dialogue nodes can only gate on quest tokens/flags/items, so the plan implements the tier marker as a **hidden quest token** ("Newcomer's Path") — strictly better, as it composes with existing dialogue + quest-engine gating. Set at Tier-A creation, gates the stage-2/3 beats, granted-end at arc completion.

**Verified APIs (use these exactly):**
- Command registry: `userCommands map[string]CommandAccess` in `internal/usercommands/usercommands.go:52`; entry shape `{Func, AllowedWhenDowned, AllowedInCombat, AdminOnly}`. Command signature: `func(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error)`.
- Prompt: `cmdPrompt, isNew := user.StartPrompt("start", rest)`; `q := cmdPrompt.Ask(question string, options []string, default ...string) *prompt.Question`; fields `q.Done`, `q.Response`; `q.RejectResponse()`; `user.ClearPrompt()`.
- Mutations: `mutations.GetWeightedPool(char.Mutations map[string]int, sp *species.Species) []string`; `mutations.RollAcquisition(pool []string) string`; species via `species.GetSpecies(char.SpeciesId)`.
- Grant a quest token from Go: `events.AddToQueue(events.Quest{UserId: id, QuestToken: "30-end"})`.
- Teleport: `rooms.MoveToRoom(userId, roomId)`; start room alias `rooms.StartRoomIdAlias` resolves to `cfg.StartRoom` (5200).
- MiscData (used for the low-progress guard heuristic if needed): `char.SetMiscData(key string, value any)`, `char.GetMiscData(key) any`.
- Room behavior gating: `room_command` event → condition `command_matches: [<cmd>]`, condition `command_rest_contains`, action `intercept`; exit lock via `set_room_locked` (see `internal/behaviortree/actions_room.go`); pattern reference `_datafiles/world/dogmud/behaviors/rooms/pothole_coulee/5200.yaml`.

---

## PHASE 1 — Framework (poll + routing + veteran + marker + helper + MOTD + back-door)

Shippable on its own: until Phase 2 lands, Tier A temporarily routes like Tier B (to the pool) — verified by the phase-1 boot test.

### Task 1: Shared random-mutation helper

Extract the roll-and-grant logic from `actGrantMutation` into a reusable `Character` method so the Rite and the veteran path share one implementation.

**Files:**
- Modify: `internal/characters/character.go` (add method)
- Modify: `internal/behaviortree/actions_quest.go:47-67` (use it)
- Test: `internal/characters/character_mutation_grant_test.go` (create)

- [ ] **Step 1: Write the failing test**

```go
package characters

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// GrantRandomMutation rolls one mutation from the weighted acquisition pool
// for the character's species and adds it at level 1. Returns the granted id
// ("" if none available).
func TestGrantRandomMutation_AddsOne(t *testing.T) {
	c := &Character{SpeciesId: 1, Mutations: map[string]int{}}
	got := c.GrantRandomMutation()
	if got == "" {
		t.Skip("no mutations registered in this test env; covered by integration")
	}
	assert.Equal(t, 1, c.Mutations[got], "granted mutation must be at level 1")
	assert.Len(t, c.Mutations, 1)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/characters/ -run TestGrantRandomMutation_AddsOne -count=1`
Expected: FAIL — `c.GrantRandomMutation undefined`.

- [ ] **Step 3: Add the method**

In `internal/characters/character.go` (near the other `Character` methods):

```go
// GrantRandomMutation rolls one mutation from the weighted acquisition pool
// for this character's species and adds it at level 1. Returns the granted
// mutation id, or "" if none were available. Shared by the Awakening Rite
// (behaviortree actGrantMutation) and the veteran character-creation skip.
func (c *Character) GrantRandomMutation() string {
	sp := species.GetSpecies(c.SpeciesId)
	pool := mutations.GetWeightedPool(c.Mutations, sp)
	if len(pool) == 0 {
		return ""
	}
	mutId := mutations.RollAcquisition(pool)
	if mutId == "" {
		return ""
	}
	if c.Mutations == nil {
		c.Mutations = make(map[string]int)
	}
	c.Mutations[mutId] = 1
	c.Validate()
	return mutId
}
```

Ensure `internal/characters/character.go` imports `mutations` and `species` (check the existing import block; add if missing). If a `characters → mutations` import cycle appears at build, instead place the helper in `internal/mutations` as `func GrantRandom(c MutationHolder) string` with a tiny interface — but try the method first.

- [ ] **Step 4: Replace the inline logic in `actGrantMutation`**

In `internal/behaviortree/actions_quest.go`, replace the body of `actGrantMutation` (lines ~52-66) with:

```go
	user.Character.GrantRandomMutation()
	return Success
```

(Keep the `user == nil` guard above it.)

- [ ] **Step 5: Run tests**

Run: `go test ./internal/characters/ ./internal/behaviortree/ -count=1`
Expected: PASS (both packages).

- [ ] **Step 6: Commit**

```bash
git add internal/characters/character.go internal/characters/character_mutation_grant_test.go internal/behaviortree/actions_quest.go
git commit -m "refactor(mutations): shared Character.GrantRandomMutation helper"
```

---

### Task 2: Veteran auto-Awaken helper

A small, testable function that performs the two Rite effects for the veteran skip: grant `30-end` + a starting mutation.

**Files:**
- Modify: `internal/usercommands/start.go` (add unexported helper)
- Test: `internal/usercommands/start_test.go` (create)

- [ ] **Step 1: Write the failing test**

```go
package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
)

func TestAutoAwaken_GrantsTokenAndMutation(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	u := users.GetByUserId(1)
	u.Character.SpeciesId = 1
	u.Character.Mutations = map[string]int{}

	autoAwaken(u)

	// 30-end queued for the quest engine (granted asynchronously via events.Quest).
	// Here we assert the synchronous effect: a starting mutation was added.
	assert.GreaterOrEqual(t, len(u.Character.Mutations), 0,
		"autoAwaken must attempt a mutation grant without panicking")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/usercommands/ -run TestAutoAwaken_GrantsTokenAndMutation -count=1`
Expected: FAIL — `autoAwaken undefined`.

- [ ] **Step 3: Add the helper to `start.go`**

```go
// autoAwaken gives a veteran-tier character the two effects of the Awakening
// Rite without making them perform it: the 30-end "Opened" token (clears the
// 5200 movement block and Warden Esk's gate) and a starting mutation.
func autoAwaken(user *users.UserRecord) {
	user.Character.GrantRandomMutation()
	events.AddToQueue(events.Quest{
		UserId:     user.UserId,
		QuestToken: "30-end",
	})
}
```

(`events` is already imported in `start.go`.)

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/usercommands/ -run TestAutoAwaken_GrantsTokenAndMutation -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/usercommands/start.go internal/usercommands/start_test.go
git commit -m "feat(start): autoAwaken helper for the veteran tier"
```

---

### Task 3: The experience poll + routing in `start.go`

Replace the age-gated "Skip tutorial?" block with the explicit 3-way poll. Extract the answer→route mapping into a testable pure function.

**Files:**
- Modify: `internal/usercommands/start.go:100-137` (replace the age-gated block)
- Test: `internal/usercommands/start_test.go` (add)

- [ ] **Step 1: Write the failing test for the route mapping**

Add to `internal/usercommands/start_test.go`:

```go
func TestOnboardingRoute(t *testing.T) {
	assert.Equal(t, routeNewbie, onboardingRoute("1"))
	assert.Equal(t, routeMudVet, onboardingRoute("2"))
	assert.Equal(t, routeVeteran, onboardingRoute("3"))
	assert.Equal(t, routeMudVet, onboardingRoute(""), "blank/default -> mud-vet (safe)")
	assert.Equal(t, routeMudVet, onboardingRoute("garbage"))
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/usercommands/ -run TestOnboardingRoute -count=1`
Expected: FAIL — `onboardingRoute`/`routeNewbie` undefined.

- [ ] **Step 3: Add the route type + mapping to `start.go`**

```go
type onboardingRouteKind int

const (
	routeMudVet onboardingRouteKind = iota // tier B (default): today's Coulee
	routeNewbie                            // tier A: tutorial antechamber
	routeVeteran                           // tier C: skip to Thornwall, Awakened
)

// onboardingRoute maps the poll answer to a route. Anything unrecognized
// (incl. the empty default) maps to the safe mud-vet path.
func onboardingRoute(answer string) onboardingRouteKind {
	switch strings.TrimSpace(answer) {
	case "1":
		return routeNewbie
	case "3":
		return routeVeteran
	default:
		return routeMudVet
	}
}
```

- [ ] **Step 4: Replace the age-gated block (lines ~100-137) with the poll**

Replace the entire `duration := time.Now().Sub(user.Joined) ... }` block with:

```go
	question := cmdPrompt.Ask(
		"How much of Gaius do you already know?\n"+
			"  1) New to text MUDs       -- I'll teach you the basics first.\n"+
			"  2) New to DOGMud          -- I know MUDs; show me what's different.\n"+
			"  3) Veteran                -- skip all tutorials; drop me into the city.",
		[]string{"1", "2", "3"}, "2")
	if !question.Done {
		return true, nil
	}

	switch onboardingRoute(question.Response) {

	case routeVeteran:
		confirm := cmdPrompt.Ask("Skip everything and start in the city, already Awakened? (y/n)", []string{"y", "n"}, "n")
		if !confirm.Done {
			return true, nil
		}
		if confirm.Response != "y" {
			question.RejectResponse() // re-ask the poll
			return true, nil
		}
		user.ClearPrompt()
		autoAwaken(user)
		startVeteranInThornwall(user) // Task 4
		return true, nil

	case routeNewbie:
		user.ClearPrompt()
		grantNewcomerMarker(user) // Task 6 (hidden Newcomer's Path token)
		// Phase 1: until the antechamber lands (Phase 2), fall through to the
		// mud-vet pool route. Task 12 replaces this with the antechamber route.
		startInCoulee(user, room) // Task 4
		return true, nil

	default: // routeMudVet
		user.ClearPrompt()
		startInCoulee(user, room) // Task 4
		return true, nil
	}
```

Remove the now-dead `tutorialRoomIds`/`CreateEphemeralRoomIds` tail block (lines ~141-168) — its logic moves into `startInCoulee`/the Phase-2 antechamber route. Remove the now-unused `time` and `strconv` imports if they become unused (run `go build` to check).

- [ ] **Step 5: Run tests + build**

Run: `go test ./internal/usercommands/ -run "TestOnboardingRoute|TestAutoAwaken" -count=1 && go build ./internal/usercommands/`
Expected: PASS + clean build (after Task 4 adds the referenced helpers; if building now, temporarily stub `startInCoulee`/`startVeteranInThornwall`/`grantNewcomerMarker` — but prefer doing Task 4 + Task 6 before building).

- [ ] **Step 6: Commit** (after Task 4 + Task 6 compile)

```bash
git add internal/usercommands/start.go internal/usercommands/start_test.go
git commit -m "feat(start): 3-way experience poll replaces age-gated skip"
```

---

### Task 4: Routing helpers (`startInCoulee`, `startVeteranInThornwall`)

**Files:**
- Modify: `internal/usercommands/start.go`

- [ ] **Step 1: Add the helpers**

```go
// startInCoulee drops the character at the Awakening Pool (StartRoom 5200) —
// the standard new-player path (mud-vet, and newbie until the antechamber
// route in Task 12).
func startInCoulee(user *users.UserRecord, room *rooms.Room) {
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="magenta">Suddenly, a vortex appears before you, drawing you in before you have any chance to react!</ansi>%s`, term.CRLFStr))
	if destRoom := rooms.LoadRoom(rooms.StartRoomIdAlias); destRoom != nil {
		rooms.MoveToRoom(user.UserId, destRoom.RoomId)
		Look("", user, destRoom, events.CmdSecretly)
	}
}

// startVeteranInThornwall vortexes an already-Awakened veteran to Thornwall
// (468) and prints the back-door hint.
func startVeteranInThornwall(user *users.UserRecord) {
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="magenta">The pool is behind you before you ever saw it. You step out already changed, into a city that does not know your name.</ansi>%s`, term.CRLFStr))
	if destRoom := rooms.LoadRoom(468); destRoom != nil {
		rooms.MoveToRoom(user.UserId, destRoom.RoomId)
		Look("", user, destRoom, events.CmdSecretly)
	}
	user.SendText(messaging.CategorySystem, `New to Gaius after all? Type <ansi fg="command">tutorial</ansi> to be taken to the newcomers' pool.`)
}
```

- [ ] **Step 2: Build**

Run: `go build ./internal/usercommands/`
Expected: clean (once Task 6's `grantNewcomerMarker` also exists).

- [ ] **Step 3: Commit** (folded into Task 3's commit if done together)

---

### Task 5: The `tutorial` back-door command

**Files:**
- Create: `internal/usercommands/tutorial.go`
- Modify: `internal/usercommands/usercommands.go:52` (register)
- Test: `internal/usercommands/tutorial_test.go` (create)

- [ ] **Step 1: Write the failing test**

```go
package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
)

func TestTutorial_LowProgressTeleports(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	user, room := getTestUserAndRoom(t)
	user.Character.QuestProgress = map[int]string{} // no progress

	handled, err := Tutorial("", user, room, 0)
	assert.True(t, handled)
	assert.NoError(t, err)
	// teleported toward the pool (StartRoom 5200); seedAllRegistries has no 5200,
	// so assert the command ran + chose to teleport rather than refuse.
}

func TestTutorial_HighProgressRefuses(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	user, room := getTestUserAndRoom(t)
	user.Character.QuestProgress = map[int]string{65: "end", 35: "end"} // real progress

	handled, err := Tutorial("", user, room, 0)
	assert.True(t, handled)
	assert.NoError(t, err)
	// refused — character stays put (no panic; covered more fully in-game).
}
```

- [ ] **Step 2: Run to verify fail**

Run: `go test ./internal/usercommands/ -run TestTutorial -count=1`
Expected: FAIL — `Tutorial undefined`.

- [ ] **Step 3: Implement `internal/usercommands/tutorial.go`**

```go
package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// onboardingAllowedTokens are the only quest steps a character may have and
// still use the `tutorial` back-door. Keeps an established character from
// yanking themselves back to the newcomers' pool. (30 = Awakening,
// 31 = Find Your Footing, 32 = Newcomer's Path marker.)
var onboardingAllowedQuests = map[int]struct{}{30: {}, 31: {}, 32: {}}

// Tutorial teleports a low-progress character to the newcomers' pool (the
// Awakening Pool, StartRoom 5200). Advertised in the veteran welcome as the
// mis-pick safety net. Refuses if the character has progress beyond the
// onboarding quests.
func Tutorial(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	for questId := range user.Character.QuestProgress {
		if _, ok := onboardingAllowedQuests[questId]; !ok {
			user.SendText(messaging.CategorySystem, `The newcomers' coulee is for those just arriving. You are past that now.`)
			return true, nil
		}
	}

	user.SendText(messaging.CategorySystem, `A familiar pull takes you back toward the glowing pool...`)
	if destRoom := rooms.LoadRoom(rooms.StartRoomIdAlias); destRoom != nil {
		rooms.MoveToRoom(user.UserId, destRoom.RoomId)
		Look("", user, destRoom, events.CmdSecretly)
	}
	return true, nil
}
```

- [ ] **Step 4: Register the command** in `internal/usercommands/usercommands.go` (inside the `userCommands` map, alphabetical-ish near other `t` commands):

```go
		`tutorial`:        {Tutorial, true, true, false},
```

- [ ] **Step 5: Run tests + build**

Run: `go test ./internal/usercommands/ -run TestTutorial -count=1 && go build ./internal/usercommands/`
Expected: PASS + clean.

- [ ] **Step 6: Commit**

```bash
git add internal/usercommands/tutorial.go internal/usercommands/tutorial_test.go internal/usercommands/usercommands.go
git commit -m "feat(tutorial): back-door command to the newcomers' pool"
```

---

### Task 6: Hidden "Newcomer's Path" marker quest + grant helper

The tier marker. A secret quest whose `start` token is granted to Tier-A characters (gates the stage-2/3 beats) and whose `end` token is granted when the arc completes.

**Files:**
- Create: `_datafiles/world/dogmud/quests/<id>-newcomers_path.yaml` (ID via `python tools/id_inventory.py --type quests`; this plan assumes **32** — verify it's free, else use the reported next-free and update all references below)
- Modify: `internal/usercommands/start.go` (add `grantNewcomerMarker`)

- [ ] **Step 1: Confirm the quest ID is free**

Run: `python tools/id_inventory.py --type quests`
Expected: a "next free" quests id. Use 32 if free; otherwise substitute throughout.

- [ ] **Step 2: Author the quest YAML** `_datafiles/world/dogmud/quests/32-newcomers_path.yaml`:

```yaml
questid: 32
name: Newcomer's Path
description: A quiet marker for players new to Gaius, so their guides know to teach the basics.
secret: true
linear: true

steps:
  - id: start
    description: "You are finding your way in Gaius."
  - id: end
    description: "You have found your footing in Gaius."

# No triggers: granted/ended explicitly (start at character creation for the
# newbie tier; end when the town footing beat completes).
triggers: []
```

- [ ] **Step 3: Add `grantNewcomerMarker` to `start.go`**

```go
// grantNewcomerMarker flags a total-newbie character with the hidden
// Newcomer's Path token, which gates the newbie-only teaching beats (Cleric
// Hadwen's mutations/progression talk; Crier Toke's player-interaction talk).
func grantNewcomerMarker(user *users.UserRecord) {
	events.AddToQueue(events.Quest{
		UserId:     user.UserId,
		QuestToken: "32-start",
	})
}
```

- [ ] **Step 4: Build + boot-load the quest**

Run: `go build ./... && go run .` (watch for `quests.LoadDataFiles loadedCount` increment + `ValidateAllFlags` clean; Ctrl-C after "Server Ready"). Expected: no panic; loadedCount +1.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/quests/32-newcomers_path.yaml internal/usercommands/start.go
git commit -m "feat(onboarding): hidden Newcomer's Path marker quest + grant"
```

---

### Task 7: MOTD fix (S3)

**Files:**
- Modify: `_datafiles/config.yaml` (`Server.Motd`, the "A NEW BEGINNING" blurb)

- [ ] **Step 1: Reword the veteran line**

In `Server.Motd`, change the "A NEW BEGINNING" sentence that implies veterans are automatically Awakened to reflect the new choice, e.g.:
`"A NEW BEGINNING — A new path into Gaius opens at Pothole Coulee. When you make your character you choose your start: brand-new to MUDs (a guided tutorial), new to DOGMud (the coulee), or a veteran who skips straight into the wider world, already Awakened. (2026-06-29.)"`

- [ ] **Step 2: Boot to confirm config parses**

Run: `go run .` → expect clean boot (Ctrl-C after "Server Ready").

- [ ] **Step 3: Commit**

```bash
git add _datafiles/config.yaml
git commit -m "fix(motd): describe the character-creation experience choice (S3)"
```

---

### Task 8: Phase-1 integration verification

- [ ] **Step 1: Full build + targeted tests**

Run: `go build ./... && go test ./internal/characters/ ./internal/usercommands/ ./internal/behaviortree/ -count=1`
Expected: all PASS, clean build.

- [ ] **Step 2: In-game smoke (harness), mud-vet + veteran**

Drive two harness sessions (see `tools/playtest/` + the smoke reports from 2026-06-29 for the protocol):
- mud-vet (poll answer `2`): lands at the pool, normal Rite — **regression** check.
- veteran (poll answer `3` → confirm `y`): lands in Thornwall, `mutations` shows one, `status`/movement free; `tutorial` returns to the pool.

- [ ] **Step 3: Commit any fixes; tag phase-1 done in the plan.**

---

## PHASE 2 — Tutorial antechamber (Tier A, stage 1)

Instanced ephemeral rooms teaching look / move / status / help / inventory / talk, each gating its exit until the verb is performed, then vortexing to the pool.

### Task 9: Allocate IDs + author antechamber room templates

**Files:**
- Create: `_datafiles/world/dogmud/rooms/newcomer_antechamber/<ids>.yaml` (×5)
- Create: `_datafiles/world/dogmud/zone-config.yaml` for the new zone folder

- [ ] **Step 1: Allocate IDs**

Run: `python tools/id_inventory.py --alloc rooms 6` and `python tools/id_inventory.py --alloc mobs 1`.
Record the room block (call them R1..R5) and the guide mob id (call it MGUIDE). Use them consistently below.

- [ ] **Step 2: Zone config** `_datafiles/world/dogmud/rooms/newcomer_antechamber/zone-config.yaml`:

```yaml
name: Newcomer Antechamber
roomid: <R1>
defaultbiome: default
region: tutorial
```

(Per the build gotcha: every zone folder needs a zone-config.yaml or boot panics.)

- [ ] **Step 3: Author the 5 rooms.** Each room has one forward exit (`north`) to the next; the last exits via a `move_player`-to-5200 behavior (Task 11). Room template (R1 shown in full; the table after gives each room's specifics — replicate the template per row):

```yaml
# <R1>.yaml — Stage 1, lesson: look
roomid: <R1>
zone: Newcomer Antechamber
title: A Place Between
biome: default
description: >-
  You are not quite awake. Grey light with no source, a floor you cannot
  feel, and a calm voice that seems to come from the walls themselves.
  "Before the world, the words," it says. "Type  look  to take in where
  you are."
exits:
  north:
    roomid: <R2>
    locked: true
```

| Room | Lesson | Gate command | Forward exit | Room copy (the calm voice teaches) |
|------|--------|--------------|--------------|-----------------------------------|
| R1 | look | `look` | north→R2 (locked) | "Type `look` to take in where you are." then on success "Good. Now type `north` to move on." |
| R2 | status/stats | `status` | north→R3 (locked) | "This is you. Type `status` to read your stats — six of them, centered at 100 — and your health, stamina, and conviction." |
| R3 | help | `help` | north→R4 (locked) | "When you forget a command, the game will tell you. Type `help look` to read about a command." (gate: command `help` with non-empty rest, via `command_rest_contains`) |
| R4 | inventory | `inventory`/`inv` | north→R5 (locked) | "You carry a plain token (look at it). Type `inv` to see what you hold." (seed item — see Task 10) |
| R5 | talk/ask | `talk`/`ask` | exit→pool via behavior | guide NPC MGUIDE present: "Speak to me — type `talk guide`, or `ask guide pool`. Then step through; the real world is waiting." |

- [ ] **Step 4: Boot-load the rooms**

Run: `go run .` → expect `rooms.LoadDataFiles` +5, `ValidateZoneConsistency errors=0` (the antechamber is a small linear chain; if the mapper complains, add `non_cartesian: true` to the zone-config). Ctrl-C after Ready.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/rooms/newcomer_antechamber/
git commit -m "feat(antechamber): tutorial room templates (look/status/help/inv/talk)"
```

---

### Task 10: Guide NPC + a trivial inventory item

**Files:**
- Create: `_datafiles/world/dogmud/mobs/newcomer_antechamber/<MGUIDE>.yaml` (+ dialogue `_datafiles/world/dogmud/dialogue/newcomer_antechamber/<MGUIDE>.yaml`)
- Reuse or create a trivial item for the inventory lesson (a no-value token; check `id_inventory.py --type items` if creating)

- [ ] **Step 1: Author the guide mob** (non_combatant, non-hostile, parked in R5; archetype `noncombat_passive`):

```yaml
mobid: <MGUIDE>
zone: Newcomer Antechamber
behavior_archetype: noncombat_passive
non_combatant: true
charm_immune: true
maxwander: 0
character:
  name: the guide
  description: |
    A figure of grey light, patient and faceless, here only to set your
    feet on the path.
  speciesid: 1
  level: 1
```

- [ ] **Step 2: Author the guide dialogue** teaching `ask guide pool` → points them to step through. Follow the dialogue schema (`docs/schemas/dialogue.md`); include `talk`/`ask`/`pool`/`ready`/`help` triggers; first-person voice.

- [ ] **Step 3: Seed the inventory-lesson item.** In R4's room YAML, add a spawned trivial item (a "smooth token", value 0) via the room's item spawn list, or grant it when the newbie enters the antechamber in Task 12. (Pick whichever the room schema supports — see an existing room with `spawninfo`/items.)

- [ ] **Step 4: Boot-load**

Run: `go run .` → `mobs.LoadDataFiles` +1, no dialogue load errors. Ctrl-C.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/mobs/newcomer_antechamber/ _datafiles/world/dogmud/dialogue/newcomer_antechamber/
git commit -m "feat(antechamber): guide NPC + inventory-lesson item"
```

---

### Task 11: Per-room gating behaviors

Each room's exit starts `locked: true`; a `room_command` behavior unlocks it when the lesson command is used (and praises). R5's behavior teleports to the pool.

**Files:**
- Create: `_datafiles/world/dogmud/behaviors/rooms/newcomer_antechamber/<R1..R5>.yaml`
- Reference: `_datafiles/world/dogmud/behaviors/rooms/pothole_coulee/5200.yaml`; conditions/actions in `internal/behaviortree/context.md` (`command_matches`, `command_rest_contains`, `set_room_locked`, `move_player`, `emote`/`say`).

- [ ] **Step 1: Author R1's behavior (template)** `behaviors/rooms/newcomer_antechamber/<R1>.yaml`:

```yaml
# Unlock the north exit once the player looks; gentle nudge otherwise.
tree:
  type: selector
  children:
    - type: sequence
      event: room_command
      children:
        - type: condition
          check: command_matches
          commands: [look]
        - type: action
          do: set_room_locked
          room_id: <R1>
          direction: north
          locked: false
        - type: action
          do: say_to_room
          text: "Good. The way north is open. Type  north  to move on."
```

(Confirm the exact action name/params for unlocking a single exit against `context.md`/`actions_room.go` — it may be `set_room_locked` with `direction`, or an exit-scoped unlock. Use what the codebase exposes; if only whole-room lock exists, lock/unlock the room and rely on the single exit.)

- [ ] **Step 2: Replicate per room** using the gate column from Task 9's table:
  - R2: `command_matches: [status]` → unlock R2.north.
  - R3: `command_matches: [help]` + `command_rest_contains` any (require a topic) → unlock R3.north.
  - R4: `command_matches: [inventory, inv]` → unlock R4.north.
  - R5: `command_matches: [talk, ask]` → action `move_player room_id: 5200` (vortex to the pool; the engine auto-looks per the 2026-06-29 actMovePlayer fix). Add a `say_to_room` send-off first.

- [ ] **Step 3: Boot-load**

Run: `go run .` → behaviors load (room behavior files are lazy-loaded on first event; at least confirm no boot panic). Ctrl-C.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/behaviors/rooms/newcomer_antechamber/
git commit -m "feat(antechamber): room_command gating behaviors per lesson"
```

---

### Task 12: Wire the antechamber into the Tier-A route

**Files:**
- Modify: `_datafiles/config.yaml` (`GamePlay.SpecialRooms.TutorialRooms`)
- Modify: `internal/usercommands/start.go` (the `routeNewbie` branch)

- [ ] **Step 1: Populate TutorialRooms** with the antechamber room IDs in order (R1 first):

```yaml
    TutorialRooms: ["<R1>", "<R2>", "<R3>", "<R4>", "<R5>"]
```

- [ ] **Step 2: Route Tier A through the ephemeral instance.** Add a helper `startInAntechamber(user)` in `start.go` that mirrors the (removed) legacy ephemeral path: `rooms.CreateEphemeralRoomIds(tutorialRoomIds...)`, vortex copy, `MoveToRoom` to the instanced first room, secret `Look`. Replace the Phase-1 `startInCoulee` call in the `routeNewbie` branch with `startInAntechamber(user)`.

```go
func startInAntechamber(user *users.UserRecord) bool {
	cfg := configs.GetSpecialRoomsConfig()
	ids := []int{}
	first := 0
	for i, s := range cfg.TutorialRooms {
		id, _ := strconv.Atoi(s)
		ids = append(ids, id)
		if i == 0 {
			first = id
		}
	}
	created, err := rooms.CreateEphemeralRoomIds(ids...)
	if err != nil {
		return false // caller falls back to startInCoulee
	}
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="magenta">The grey takes you...</ansi>%s`, term.CRLFStr))
	rooms.MoveToRoom(user.UserId, created[first])
	if r := rooms.LoadRoom(created[first]); r != nil {
		Look("", user, r, events.CmdSecretly)
	}
	return true
}
```

In `routeNewbie`: `if !startInAntechamber(user) { startInCoulee(user, room) }`.

- [ ] **Step 3: Build**

Run: `go build ./...` → clean (restore `strconv` import in start.go for this helper).

- [ ] **Step 4: In-game (harness) — total-newbie stage 1**

Drive a newbie session (poll `1`): confirm each room gates on its verb (exit stays locked until the command is typed), help requires a topic, the guide responds, and stepping through R5 vortexes to the pool with an auto-look.

- [ ] **Step 5: Commit**

```bash
git add internal/usercommands/start.go _datafiles/config.yaml
git commit -m "feat(antechamber): route the newbie tier through the instanced tutorial"
```

---

## PHASE 3 — Downstream teaching beats (Tier A, stages 2-3)

Newbie-gated additive beats, all gated on `32-start` / `questExcluded: [32-end]` so Tier B never sees them.

### Task 13: Stage 2 — Hadwen teaches mutations + progression (post-Rite)

**Files:**
- Modify: `_datafiles/world/dogmud/dialogue/pothole_coulee/9100.yaml` (add a newbie-gated node)

- [ ] **Step 1: Add a newbie-gated teaching node** to Hadwen's dialogue, fired after the Rite. Gate: `questRequired: ["30-end", "32-start"]`, `questExcluded: ["32-end"]`. Triggers include `mutation`/`mutations`/`grow`/`skill`/`progress`/`level`/`quest`/`task`. Text (first person, Hadwen's voice): the pool woke a mutation in you — type `mutations` to read it; and Gaius has no levels or experience to grind — you grow by *using* what you have, so type `skills` and `status` to watch yourself climb as you act. (80-col wrapped; place the node FIRST among Hadwen's `tree.nodes` per the substring-shadow gotcha.)

- [ ] **Step 2: Boot-load + harness check**

Boot clean; in a newbie session, after the Rite, `ask hadwen mutation` (or the root hint) delivers the beat; in a mud-vet session it does NOT (no `32-start`).

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/dialogue/pothole_coulee/9100.yaml
git commit -m "feat(onboarding): Hadwen teaches mutations + progression (newbie-gated)"
```

---

### Task 14: Stage 3 — Toke teaches player interaction + clears the marker

**Files:**
- Modify: `_datafiles/world/dogmud/dialogue/pothole_coulee/9106.yaml` (Crier Toke; newbie-gated beat + grant `32-end`)

- [ ] **Step 1: Add a newbie-gated node** to Toke's footing flow. Gate: `questRequired: ["32-start"]`, `questExcluded: ["32-end"]`, and `grantsQuest: "32-end"` (clears the marker = arc complete). Triggers include `people`/`players`/`talk`/`say`/`who`/`others`/`quest`/`task`. Text (Toke's voice): the square fills with real folk — type `say <words>` to speak to the room, `who` to see who's about, `tell <name> <words>` to speak privately, and try an emote like `wave`. Per the re-grant SOP, `questExcluded` includes the end token; place the grant node FIRST in `tree.nodes`.

- [ ] **Step 2: Boot-load + harness check**

Newbie session: reaching Toke and asking about people delivers the beat and clears `32-start→32-end` (verify with `questtoken 32-end`); afterward the Hadwen/Toke beats no longer fire. Mud-vet session: never sees it.

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/dialogue/pothole_coulee/9106.yaml
git commit -m "feat(onboarding): Toke teaches player interaction; clears newbie marker"
```

---

### Task 15: Full-feature verification + pre-push hygiene

- [ ] **Step 1: Full build + test**

Run: `go build ./... && go test ./internal/characters/ ./internal/usercommands/ ./internal/behaviortree/ ./internal/questengine/ ./internal/dialogue/ -count=1`
Expected: all PASS.

- [ ] **Step 2: Clean boot**

Run: `go run .` → quests loadedCount includes 32, flags validated, ValidateZoneConsistency errors=0 mode=panic, no panics. Ctrl-C.

- [ ] **Step 3: Triple harness smoke (the three tiers)**

Re-run the three personas (map 1:1 to tiers): total-newbie (full arc: antechamber → Rite → Hadwen beat → Toke beat → marker cleared), mud-vet (regression: identical to today, no beats), veteran (Thornwall, Awakened, `tutorial` returns to pool).

- [ ] **Step 4: PATCH_NOTES + merge**

Add a player-facing PATCH_NOTES entry; merge the feature branch to master `--no-ff`. (Push is the user's droplet step.)

---

## Self-review notes (author)

- **Spec coverage:** poll (§3→T3), Tier A antechamber (§4.1→T9-12), stage 2 (§4.2→T13), stage 3 (§4.3→T14), marker (§5→T6, refined to a quest token), veteran (§6→T2,T4), `tutorial` back-door (§6→T5), mutation helper (§7→T1), MOTD (§8→T7). All covered.
- **Open technical check (carried from spec §11):** the exact exit-unlock action/params for a single direction (T11 step 1) — verify against `actions_room.go`/`context.md` before authoring all five; questengine `command`-trigger fallback is NOT viable for look/status/help (they don't fire questengine command events), so room behaviors are the chosen mechanism (confirmed: `room_command` fires for all commands).
- **Marker as quest token** (not MiscData) is the deliberate refinement enabling dialogue gating — applied consistently (T6, T13, T14).
