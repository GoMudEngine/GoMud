# Newcomer Tutorial Guide Redesign — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rebuild the route-1 "complete noob" antechamber (rooms 6258–6262 + one new room) into a strict, guide-led tutorial: a named mentor (Dewey) walks the player through look/examine/move, status + progression, get/inventory/help, say/ask, a combat primer teaching conditions + the shared cooldown + all three playstyles, and flee — then hands off to the Awakening Pool.

**Architecture:** Almost entirely **data** (room YAML + room behavior-tree YAML + mob YAML + dialogue YAML). The engine loads these; per-instance `set_state` flags gate each lesson. One small **Go** addition: a `grant_progression` behavior action so the player sees a real progression banner once. No changes to `start.go`, quests 21/30/31, Cleric Hadwen, or Pothole Coulee.

**Tech Stack:** Go (behavior-tree action), YAML data files (rooms, room behavior trees, mobs, dialogue, goals), the GoMud room-behavior-tree runtime (`internal/behaviortree`).

**Design spec:** `docs/superpowers/specs/2026-07-17-newcomer-tutorial-guide-redesign-design.md`

---

## Shared conventions (read once, apply in every task)

**Command highlighting.** Every command word Dewey speaks is wrapped
`<ansi fg="command">look</ansi>` (color alias `command`, cyan). Never use the old
plain-double-space style.

**Dewey speaks via `mob_say`.** In room behavior trees, Dewey's lines are authored
as `do: mob_say`, `mob_id: 9491`, `text: ...`. The engine renders them as
`Dewey says, "..."`. Dewey (mob 9491) must be in every antechamber room's
`spawninfo`.

**Gating idiom.** Each room tree is a `selector` of `sequence` children, each child
`event: room_command`. Lesson branches come first (specific → general), then a
movement/exit-block branch last. Lesson branches set a per-instance flag with
`set_state` and generally do **not** `intercept` (so the taught command still runs
normally). The block branch fires only while the gate flag is unset and always
`intercept`s. Per-instance state is keyed by the ephemeral instance room, so each
player's copy gates independently.

**Room IDs (forward order):** 6258 → 6259 → 6260 → 6261 → 6262 → **6467** (new;
6263 is taken by `east_road_to_greenford`). Each room (including room 5, 6262) has
exactly **one forward exit, `north`, to the next room**. This matters: the
antechamber runs as a private **ephemeral instance**, and the instance runtime
remaps real exits to each player's private copy. In-instance hops must therefore
use the real `north` exit — **never** `move_player(templateId)` between two
in-instance rooms (that would pull the player into the shared template room). Only
the final hop from room 6 (6467) to room 5200 (a real, non-instanced Pothole Coulee
room) uses `move_player`, exactly as the old terminus did. Because room 5's only
exit is `north`, `flee` there can only carry the player forward. Mob IDs: Dewey =
9491 (re-themed), effigy = **9614** (new).

**Zone folder:** `_datafiles/world/dogmud/rooms/newcomer_antechamber/`,
`.../behaviors/rooms/newcomer_antechamber/`, `.../mobs/newcomer_antechamber/`,
`.../dialogue/newcomer_antechamber/`, `.../goals/`.

**Before any smoke test, wipe instance saves** (SOP):
```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
```

---

## Task 1: `grant_progression` behavior action (Go, TDD)

Adds a room-usable action that forces exactly one real skill-progression event
(with the standard banner) for the triggering player. Mirrors the guardrail-test
style already used by `actions_move_player_test.go` in this package (happy path is
covered by the live walk-through in Task 11).

**Files:**
- Create: `internal/behaviortree/actions_progression.go`
- Modify: `internal/behaviortree/actions.go` (register the action, ~line 63 block)
- Test: `internal/behaviortree/actions_progression_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/behaviortree/actions_progression_test.go`:

```go
package behaviortree

import "testing"

// grant_progression forces one real skill-progression event for the triggering
// player (used by the newcomer tutorial to show the banner once). These pin the
// guardrails; the happy path (real user, skill actually increments + banner) is
// covered by the live tutorial walk-through, matching actions_move_player_test.go.

func TestGrantProgression_RegisteredInActionRegistry(t *testing.T) {
	if _, ok := actionRegistry["grant_progression"]; !ok {
		t.Fatal("grant_progression not registered in actionRegistry")
	}
}

func TestGrantProgression_NoTriggeringPlayer_Failure(t *testing.T) {
	ctx := &EvalContext{Event: EventContext{EventType: "room_command", UserId: 0}}
	if res := actGrantProgression(map[string]any{"skill": "spellcasting"}, ctx); res != Failure {
		t.Errorf("expected Failure with no triggering player, got %v", res)
	}
}

func TestGrantProgression_MissingSkillParam_Failure(t *testing.T) {
	// UserId 42 is not registered; but the missing-skill guard should fire first.
	ctx := &EvalContext{Event: EventContext{EventType: "room_command", UserId: 42}}
	if res := actGrantProgression(map[string]any{}, ctx); res != Failure {
		t.Errorf("expected Failure with no skill param, got %v", res)
	}
}

func TestGrantProgression_UnknownUser_Failure(t *testing.T) {
	ctx := &EvalContext{Event: EventContext{EventType: "room_command", UserId: 42}}
	if res := actGrantProgression(map[string]any{"skill": "spellcasting"}, ctx); res != Failure {
		t.Errorf("expected Failure for unknown user, got %v", res)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/behaviortree/ -run TestGrantProgression -v`
Expected: FAIL — `undefined: actGrantProgression` (compile error).

- [ ] **Step 3: Write the implementation**

Create `internal/behaviortree/actions_progression.go`:

```go
package behaviortree

import (
	"github.com/GoMudEngine/GoMud/internal/users"
)

// actGrantProgression forces exactly one real skill-progression event for the
// triggering player, emitting the standard SKILL ADVANCEMENT banner. It calls the
// genuine CheckSkillProgression with a large bonus multiplier: the chance clamps
// to 1.0 (progression.go), so the roll is guaranteed, and the real path does the
// IncreaseSkill + tier bookkeeping and queues the banner. The tutorial guards this
// to fire once via a room set_state flag.
//
// params: skill (string) — the skill tag to advance (e.g. "spellcasting").
func actGrantProgression(params map[string]any, ctx *EvalContext) Result {
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}
	skill := getStringParam(params, "skill")
	if skill == "" {
		return Failure
	}
	user.Character.CheckSkillProgression(skill, user.UserId, 1000.0)
	return Success
}
```

- [ ] **Step 4: Register the action**

In `internal/behaviortree/actions.go`, in the "New actions for room behavior trees"
block (near line 63, alongside `send_user_text`/`mob_say`), add:

```go
	actionRegistry["grant_progression"] = actGrantProgression
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `go test ./internal/behaviortree/ -run TestGrantProgression -v`
Expected: PASS (all four tests).

- [ ] **Step 6: Build the whole engine**

Run: `go build ./...`
Expected: no errors.

- [ ] **Step 7: Commit**

```bash
git add internal/behaviortree/actions_progression.go internal/behaviortree/actions.go internal/behaviortree/actions_progression_test.go
git commit -m "feat(behaviortree): grant_progression room action for tutorial banner"
```

---

## Task 2: Config — add room 6467 to TutorialRooms + fix stale comment

**Files:**
- Modify: `_datafiles/config.yaml` (SpecialRooms block, ~line 1297-1303)

- [ ] **Step 1: Edit the TutorialRooms array and comment**

Replace:
```yaml
  #   Sanctum Basin replaces the old tutorial zone; TutorialRooms is intentionally empty.
  TutorialRooms: ["6258", "6259", "6260", "6261", "6262"]
```
with:
```yaml
  #   The Newcomer Antechamber (rooms 6258-6262 + 6467) is the route-1 guided
  #   tutorial. The first id is where a new "complete noob" player is placed.
  TutorialRooms: ["6258", "6259", "6260", "6261", "6262", "6467"]
```

- [ ] **Step 2: Verify YAML parses**

Run: `python -c "import yaml,sys; yaml.safe_load(open('_datafiles/config.yaml'))" && echo OK`
Expected: `OK`

- [ ] **Step 3: Commit**

```bash
git add _datafiles/config.yaml
git commit -m "chore(config): add tutorial room 6467 to TutorialRooms; fix stale comment"
```

---

## Task 3: Re-theme the guide mob into Dewey

The existing guide (mob 9491) becomes **Dewey**, a warm named mentor. The mob file
must be renamed to match the new name (`ConvertForFilename("Dewey")` → `dewey`), or
the loader panics at startup. The goals file is renamed alongside it.

**Files:**
- Delete: `_datafiles/world/dogmud/mobs/newcomer_antechamber/9491-the_guide.yaml`
- Create: `_datafiles/world/dogmud/mobs/newcomer_antechamber/9491-dewey.yaml`
- Rename: `_datafiles/world/dogmud/goals/9491-the_guide.yaml` → `.../goals/9491-dewey.yaml`

- [ ] **Step 1: Create the new Dewey mob file**

`_datafiles/world/dogmud/mobs/newcomer_antechamber/9491-dewey.yaml`:
```yaml
mobid: 9491
zone: Newcomer Antechamber
behavior_archetype: noncombat_passive
non_combatant: true
charm_immune: true
hostile: false
statpool: 30
maxwander: 0
groups:
  - humanoid
character:
  name: Dewey
  description: |
    A wiry figure with laugh-lines and patient eyes, dressed in plain,
    travel-worn clothes. Unlike everything else in this grey place, Dewey
    is solid and warm -- a person, not a trick of the light. Dewey came
    through here once, frightened and brand-new, and never quite left the
    threshold: someone has to catch the next ones as they wake.
  speciesid: 1
  level: 1
```

- [ ] **Step 2: Delete the old guide mob file**

```bash
git rm _datafiles/world/dogmud/mobs/newcomer_antechamber/9491-the_guide.yaml
```

- [ ] **Step 3: Rename the goals file**

```bash
git mv _datafiles/world/dogmud/goals/9491-the_guide.yaml _datafiles/world/dogmud/goals/9491-dewey.yaml
```
(Content stays: `mob_id: 9491`, `next_goal_id: 1`, `seeded_from_archetype: true`,
`goals: []` — no edit needed; the file is keyed by `mob_id` internally.)

- [ ] **Step 4: Commit** (boot verification happens in Task 11)

```bash
git add _datafiles/world/dogmud/mobs/newcomer_antechamber/9491-dewey.yaml
git commit -m "feat(tutorial): re-theme guide mob 9491 into Dewey"
```

---

## Task 4: Create the straw effigy practice mob (9614)

Attackable but harmless: `combat_passive` (can be attacked, never initiates),
`hostile: false`, `maxwander: 0` (stationary), high vitality so it survives the
lesson, species 19 (the dummy species — zero base damage). NOT `non_combatant`
(that flag makes a mob unattackable).

**Files:**
- Create: `_datafiles/world/dogmud/mobs/newcomer_antechamber/9614-straw_effigy.yaml`
- Create: `_datafiles/world/dogmud/goals/9614-straw_effigy.yaml`

- [ ] **Step 1: Create the effigy mob file**

`_datafiles/world/dogmud/mobs/newcomer_antechamber/9614-straw_effigy.yaml`:
```yaml
mobid: 9614
zone: Newcomer Antechamber
behavior_archetype: combat_passive
hostile: false
maxwander: 0
pack_flee_immune: true
statpool: 3
groups:
  - construct
activitylevel: 3
idlecommands:
  - 'emote sags on its post, straw whispering, then settles upright again'
  - ''
  - ''
character:
  name: Straw Effigy
  description: |
    A crude man-shape of bound straw and sackcloth lashed to a fixed post,
    faceless but for two charcoal smudges where eyes might be. It does not
    breathe and does not flinch until struck -- and it rights itself with a
    dry rustle every time, no worse for the beating. A safe thing to learn
    a fight against.
  speciesid: 19
  level: 1
  gold: 0
  stats:
    # Species 19 (dummy) has base vitality 200 and zero base damage. Add a
    # large vitality bump so the effigy cannot be killed during the short
    # scripted lesson (no engine "invulnerable" flag exists; high HP is the
    # mechanism). It deals no damage, so there is no time pressure on the player.
    vitality:
      training: 300
```

- [ ] **Step 2: Create the effigy goals file**

`_datafiles/world/dogmud/goals/9614-straw_effigy.yaml`:
```yaml
mob_id: 9614
next_goal_id: 1
seeded_from_archetype: true
goals: []
```

- [ ] **Step 3: Verify both files parse**

Run: `python -c "import yaml; yaml.safe_load(open('_datafiles/world/dogmud/mobs/newcomer_antechamber/9614-straw_effigy.yaml')); yaml.safe_load(open('_datafiles/world/dogmud/goals/9614-straw_effigy.yaml'))" && echo OK`
Expected: `OK`

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/mobs/newcomer_antechamber/9614-straw_effigy.yaml _datafiles/world/dogmud/goals/9614-straw_effigy.yaml
git commit -m "feat(tutorial): add straw effigy practice mob 9614"
```

---

## Task 5: Room 1 — The Threshold (6258): look → examine → move

Teaches `look`, then `look <noun>` (examine), then movement. Rich nouns because
this room *is* the examine lesson.

**Files:**
- Modify (overwrite): `_datafiles/world/dogmud/rooms/newcomer_antechamber/6258.yaml`
- Modify (overwrite): `_datafiles/world/dogmud/behaviors/rooms/newcomer_antechamber/6258.yaml`

- [ ] **Step 1: Overwrite the room file**

`_datafiles/world/dogmud/rooms/newcomer_antechamber/6258.yaml`:
```yaml
roomid: 6258
zone: Newcomer Antechamber
title: The Threshold
description: >
  Grey light with no source, a floor you cannot quite feel -- and then, warm
  and solid where nothing else is, someone waiting for you. 'There you are,'
  Dewey says gently. 'You're waking up. First thing anyone learns here: type
  <ansi fg="command">look</ansi> to take in wherever you stand. Go on.'
biome: default
exits:
  north:
    roomid: 6259
spawninfo:
  - mobid: 9491
nouns:
  hands: >-
    You turn your hands over. They are half-there -- the grey light shows
    through them at the edges, firming up even as you watch, as if you are
    being remembered back into a body. Whatever this place is, you are not
    finished arriving.
  grey: >-
    The grey has no walls and no distance; it simply is, in every direction.
    Not death, Dewey will tell you, and not quite life yet either -- the
    threshold between, where the newly-woken gather themselves before the
    world.
  stone: >-
    A single worn threshold-stone sits underfoot, the one solid thing you can
    feel. Its surface is polished to a shine, hollowed slightly at the center
    by the passage of every soul who ever woke here and stepped through.
  dewey: >-
    Wiry, plain-dressed, patient -- and warm in a way nothing else here is.
    Dewey watches you with the easy attention of someone who has done this
    many times and still means it every time.
idlemessages:
  - The grey light pulses, slow and soundless, like a breath.
```

- [ ] **Step 2: Overwrite the room behavior tree**

`_datafiles/world/dogmud/behaviors/rooms/newcomer_antechamber/6258.yaml`:
```yaml
# 6258 The Threshold — gates: look, then look <noun>, then north.
# Branch order matters (selector = first success wins): the specific
# examine branch precedes the plain-look branch so `look hands` advances
# the examine step once `looked` is set. No branch intercepts a lesson
# command, so look/examine still run normally.

tree:
  type: selector
  children:

    # ── examine a noun (requires plain look done first) ──
    - type: sequence
      event: room_command
      children:
        - type: condition
          check: command_matches
          commands: [look, l, examine, exam, x]
        - type: condition
          check: command_rest_contains
          keywords: [hands, grey, stone, dewey]
        - type: condition
          check: state_equals
          key: looked
          value: "1"
        - type: condition
          check: state_equals
          key: examined
          value: ""
        - type: action
          do: set_state
          key: examined
          value: "1"
        - type: action
          do: mob_say
          mob_id: 9491
          text: >-
            That's the whole trick of this world -- the ones who read what's
            in front of them live longest. The way <ansi fg="command">north</ansi>
            is open. Type <ansi fg="command">north</ansi> and I'll be right there
            with you.

    # ── plain look ──
    - type: sequence
      event: room_command
      children:
        - type: condition
          check: command_matches
          commands: [look, l]
        - type: condition
          check: state_equals
          key: looked
          value: ""
        - type: action
          do: set_state
          key: looked
          value: "1"
        - type: action
          do: mob_say
          mob_id: 9491
          text: >-
            Good. Now look *closer* at something -- your own
            <ansi fg="command">look hands</ansi> will do. Descriptions here
            reward a second look.

    # ── movement before both lessons: nudge + block ──
    - type: sequence
      event: room_command
      children:
        - type: condition
          check: command_matches
          commands: [north, n, go, south, s, east, e, west, w, up, u, down, d]
        - type: condition
          check: state_equals
          key: examined
          value: ""
        - type: action
          do: mob_say
          mob_id: 9491
          text: >-
            Not yet. Type <ansi fg="command">look</ansi> to take the room in,
            then <ansi fg="command">look hands</ansi> to read something closely.
            Then north.
        - type: action
          do: intercept
```

- [ ] **Step 3: Verify both files parse**

Run: `python -c "import yaml; yaml.safe_load(open('_datafiles/world/dogmud/rooms/newcomer_antechamber/6258.yaml')); yaml.safe_load(open('_datafiles/world/dogmud/behaviors/rooms/newcomer_antechamber/6258.yaml'))" && echo OK`
Expected: `OK`

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/rooms/newcomer_antechamber/6258.yaml _datafiles/world/dogmud/behaviors/rooms/newcomer_antechamber/6258.yaml
git commit -m "feat(tutorial): room 1 The Threshold (look/examine/move)"
```

---

## Task 6: Room 2 — Knowing Yourself (6259): status + progression teaser

**Files:**
- Modify (overwrite): `_datafiles/world/dogmud/rooms/newcomer_antechamber/6259.yaml`
- Modify (overwrite): `_datafiles/world/dogmud/behaviors/rooms/newcomer_antechamber/6259.yaml`

- [ ] **Step 1: Overwrite the room file**

`_datafiles/world/dogmud/rooms/newcomer_antechamber/6259.yaml`:
```yaml
roomid: 6259
zone: Newcomer Antechamber
title: Knowing Yourself
description: >
  The grey brightens over a wide, still surface, dark as poured glass. 'This
  is where you meet yourself,' Dewey says. 'Type <ansi fg="command">status</ansi>
  to read your six stats and the three pools beneath them -- health, stamina,
  and conviction. Knowing yourself is the first armor.'
biome: default
exits:
  north:
    roomid: 6260
spawninfo:
  - mobid: 9491
nouns:
  water: >-
    The still surface shows you back to yourself -- and, far down beneath your
    reflection, a faint blue-green glow, steady and patient. It is the same
    light you'll one day find in the Awakening Pool. For now it only watches.
  pattern: >-
    Six faint points of light hang in the depths in a slow ring -- Strength,
    Dexterity, Perception, Vitality, Willpower, Charisma. Every soul that wakes
    here carries all six, each one centered on the same human measure. What you
    *do* with them is what makes you different.
  embers: >-
    Three small embers drift beneath the surface: one red, one green, one gold
    -- your health, your stamina, your conviction. They dim when you spend them
    and warm again when you rest.
  dewey: >-
    Dewey stands a little back from the water, giving you room to look. 'Take
    your time. There's no clock on you here.'
idlemessages:
  - The still surface trembles once, then settles glass-flat again.
```

- [ ] **Step 2: Overwrite the room behavior tree**

`_datafiles/world/dogmud/behaviors/rooms/newcomer_antechamber/6259.yaml`:
```yaml
# 6259 Knowing Yourself — gate: status before leaving north.
# The status branch also plants the progression teaser (paid off in room 5).

tree:
  type: selector
  children:

    # ── status lesson ──
    - type: sequence
      event: room_command
      children:
        - type: condition
          check: command_matches
          commands: [status, stat, st, score]
        - type: condition
          check: state_equals
          key: lesson_done
          value: ""
        - type: action
          do: set_state
          key: lesson_done
          value: "1"
        - type: action
          do: mob_say
          mob_id: 9491
          text: >-
            There you are -- all of you, on one page. Those stats and pools
            *grow*, but not the way you might expect: no levels, no experience
            to grind. I'll show you how when we spar. For now, the way
            <ansi fg="command">north</ansi> is open.

    # ── movement before lesson: nudge + block ──
    - type: sequence
      event: room_command
      children:
        - type: condition
          check: command_matches
          commands: [north, n, go, south, s, east, e, west, w, up, u, down, d]
        - type: condition
          check: state_equals
          key: lesson_done
          value: ""
        - type: action
          do: mob_say
          mob_id: 9491
          text: >-
            Meet yourself first -- type <ansi fg="command">status</ansi>, then
            go north.
        - type: action
          do: intercept
```

- [ ] **Step 3: Verify both files parse**

Run: `python -c "import yaml; yaml.safe_load(open('_datafiles/world/dogmud/rooms/newcomer_antechamber/6259.yaml')); yaml.safe_load(open('_datafiles/world/dogmud/behaviors/rooms/newcomer_antechamber/6259.yaml'))" && echo OK`
Expected: `OK`

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/rooms/newcomer_antechamber/6259.yaml _datafiles/world/dogmud/behaviors/rooms/newcomer_antechamber/6259.yaml
git commit -m "feat(tutorial): room 2 Knowing Yourself (status + progression teaser)"
```

---

## Task 7: Room 3 — What You Carry (6260): get token → inventory → help

The grey token (`itemid: 2`) rests on the floor here (relocated from the old
6261).

**Files:**
- Modify (overwrite): `_datafiles/world/dogmud/rooms/newcomer_antechamber/6260.yaml`
- Modify (overwrite): `_datafiles/world/dogmud/behaviors/rooms/newcomer_antechamber/6260.yaml`

- [ ] **Step 1: Overwrite the room file**

`_datafiles/world/dogmud/rooms/newcomer_antechamber/6260.yaml`:
```yaml
roomid: 6260
zone: Newcomer Antechamber
title: What You Carry
description: >
  A shallow alcove of grey light, and a smooth token resting in it. 'Small
  things matter out there,' Dewey says. 'Type <ansi fg="command">get token</ansi>
  to pick it up, then <ansi fg="command">inventory</ansi> -- or just
  <ansi fg="command">inv</ansi> -- to see what you're carrying.'
biome: default
exits:
  north:
    roomid: 6261
spawninfo:
  - mobid: 9491
items:
  - itemid: 2
nouns:
  token: >-
    A smooth grey disc, cool and featureless, no bigger than a coin. It has no
    obvious use -- Dewey just wants you to hold something, so that picking a
    thing up stops being a mystery before the world hands you things that
    matter.
  alcove: >-
    A shallow niche worn into the grey, the sort of place a traveler leaves
    something for whoever comes next. Others have stood exactly here, learning
    exactly this.
  dewey: >-
    'Everyone drops something and forgets it eventually,' Dewey says with a
    small grin. 'Better you learn where things go now, while it's only a token.'
idlemessages:
  - Motes of pale light drift through the alcove and wink out.
```

- [ ] **Step 2: Overwrite the room behavior tree**

`_datafiles/world/dogmud/behaviors/rooms/newcomer_antechamber/6260.yaml`:
```yaml
# 6260 What You Carry — gates: get token, then inventory, then help.

tree:
  type: selector
  children:

    # ── get the token ──
    - type: sequence
      event: room_command
      children:
        - type: condition
          check: command_matches
          commands: [get, take, g]
        - type: condition
          check: command_rest_contains
          keywords: [token]
        - type: condition
          check: state_equals
          key: got
          value: ""
        - type: action
          do: set_state
          key: got
          value: "1"
        - type: action
          do: mob_say
          mob_id: 9491
          text: >-
            Yours now. See what you're carrying -- type
            <ansi fg="command">inventory</ansi>, or just
            <ansi fg="command">inv</ansi>.

    # ── inventory ──
    - type: sequence
      event: room_command
      children:
        - type: condition
          check: command_matches
          commands: [inventory, inv, i]
        - type: condition
          check: state_equals
          key: got
          value: "1"
        - type: condition
          check: state_equals
          key: invd
          value: ""
        - type: action
          do: set_state
          key: invd
          value: "1"
        - type: action
          do: mob_say
          mob_id: 9491
          text: >-
            There it is. And when you forget a command -- you will, everyone
            does -- the game remembers for you. Type
            <ansi fg="command">help</ansi> to see the whole index.

    # ── help ──
    - type: sequence
      event: room_command
      children:
        - type: condition
          check: command_matches
          commands: [help]
        - type: condition
          check: state_equals
          key: invd
          value: "1"
        - type: condition
          check: state_equals
          key: helped
          value: ""
        - type: action
          do: set_state
          key: helped
          value: "1"
        - type: action
          do: mob_say
          mob_id: 9491
          text: >-
            That list is always there -- <ansi fg="command">help</ansi> for the
            index, <ansi fg="command">help</ansi> and a topic for one thing.
            Never be afraid to use it. The way <ansi fg="command">north</ansi>
            is open.

    # ── movement before all three: nudge + block ──
    - type: sequence
      event: room_command
      children:
        - type: condition
          check: command_matches
          commands: [north, n, go, south, s, east, e, west, w, up, u, down, d]
        - type: condition
          check: state_equals
          key: helped
          value: ""
        - type: action
          do: mob_say
          mob_id: 9491
          text: >-
            A few small things first: <ansi fg="command">get token</ansi>, then
            <ansi fg="command">inv</ansi>, then <ansi fg="command">help</ansi>.
            Then north.
        - type: action
          do: intercept
```

- [ ] **Step 3: Verify both files parse**

Run: `python -c "import yaml; yaml.safe_load(open('_datafiles/world/dogmud/rooms/newcomer_antechamber/6260.yaml')); yaml.safe_load(open('_datafiles/world/dogmud/behaviors/rooms/newcomer_antechamber/6260.yaml'))" && echo OK`
Expected: `OK`

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/rooms/newcomer_antechamber/6260.yaml _datafiles/world/dogmud/behaviors/rooms/newcomer_antechamber/6260.yaml
git commit -m "feat(tutorial): room 3 What You Carry (get/inventory/help)"
```

---

## Task 8: Room 4 — The World Speaks (6261): say → ask dewey

**Files:**
- Modify (overwrite): `_datafiles/world/dogmud/rooms/newcomer_antechamber/6261.yaml`
- Modify (overwrite): `_datafiles/world/dogmud/behaviors/rooms/newcomer_antechamber/6261.yaml`

- [ ] **Step 1: Overwrite the room file**

`_datafiles/world/dogmud/rooms/newcomer_antechamber/6261.yaml`:
```yaml
roomid: 6261
zone: Newcomer Antechamber
title: The World Speaks
description: >
  Sound seeps into the silence here -- far voices, wind, the distant clatter
  of a living world. 'You're almost through,' Dewey says. 'Out there, people
  and creatures answer when you speak. Try it: <ansi fg="command">say hello</ansi>.
  Then ask me something -- <ansi fg="command">ask dewey world</ansi>.'
biome: default
exits:
  north:
    roomid: 6262
spawninfo:
  - mobid: 9491
nouns:
  voices: >-
    Faint and far, the sound of a whole world going about its business --
    hawkers, footsteps, a snatch of song. That is Gaius, waiting just past this
    last stretch of grey. It sounds busy. It sounds alive.
  doorway: >-
    A ragged doorway of brighter light hangs in the grey, and through it you
    catch half-glimpses: a rooftop, a stretch of pale coulee wall, sky. The
    real world, close enough now to smell on the wind.
  wind: >-
    A real breeze moves through, carrying dust and warmth and the green smell
    of open country -- the first honest air since you woke.
  dewey: >-
    'Ask me anything while you can,' Dewey says. 'World, the pools, the Opened,
    how you'll grow -- I've got a moment, and out there everyone's busy.'
idlemessages:
  - A gust carries a scrap of far-off laughter through the doorway of light.
```

- [ ] **Step 2: Overwrite the room behavior tree**

`_datafiles/world/dogmud/behaviors/rooms/newcomer_antechamber/6261.yaml`:
```yaml
# 6261 The World Speaks — gates: say, then ask. Neither intercepts, so `say`
# broadcasts and `ask dewey <topic>` still hits Dewey's dialogue tree.

tree:
  type: selector
  children:

    # ── say ──
    - type: sequence
      event: room_command
      children:
        - type: condition
          check: command_matches
          commands: [say, '"', shout, yell]
        - type: condition
          check: state_equals
          key: said
          value: ""
        - type: action
          do: set_state
          key: said
          value: "1"
        - type: action
          do: mob_say
          mob_id: 9491
          text: >-
            Ha -- and the world will answer, out there. Now ask *me* something
            directly. Try <ansi fg="command">ask dewey world</ansi>, or ask
            about the pools, or how you'll grow.

    # ── ask ──
    - type: sequence
      event: room_command
      children:
        - type: condition
          check: command_matches
          commands: [ask, talk]
        - type: condition
          check: state_equals
          key: said
          value: "1"
        - type: condition
          check: state_equals
          key: asked
          value: ""
        - type: action
          do: set_state
          key: asked
          value: "1"
        - type: action
          do: mob_say
          mob_id: 9491
          text: >-
            That's how you'll learn half of what matters out there -- just ask.
            One last thing before I let you go, and it's the one that keeps you
            alive. The way <ansi fg="command">north</ansi> is open.

    # ── movement before both: nudge + block ──
    - type: sequence
      event: room_command
      children:
        - type: condition
          check: command_matches
          commands: [north, n, go, south, s, east, e, west, w, up, u, down, d]
        - type: condition
          check: state_equals
          key: asked
          value: ""
        - type: action
          do: mob_say
          mob_id: 9491
          text: >-
            Speak first -- <ansi fg="command">say hello</ansi>, then
            <ansi fg="command">ask dewey world</ansi>. Then north.
        - type: action
          do: intercept
```

- [ ] **Step 3: Verify both files parse**

Run: `python -c "import yaml; yaml.safe_load(open('_datafiles/world/dogmud/rooms/newcomer_antechamber/6261.yaml')); yaml.safe_load(open('_datafiles/world/dogmud/behaviors/rooms/newcomer_antechamber/6261.yaml'))" && echo OK`
Expected: `OK`

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/rooms/newcomer_antechamber/6261.yaml _datafiles/world/dogmud/behaviors/rooms/newcomer_antechamber/6261.yaml
git commit -m "feat(tutorial): room 4 The World Speaks (say/ask)"
```

---

## Task 9: Room 5 — The Proving (6262): the combat primer

The capstone. Gated sub-step chain: `warcried` → `saw_conditions` →
`checked_cooldowns` → `attacked` → `cast_spike` → `recast_blocked` → `tripped`
→ (grant_progression + primer) → `flee`/`north` out. The effigy (9614) and Dewey
(9491) both spawn here. The room has a **single forward `north` exit to 6467** (the
instance remaps it to the player's private copy). `flee`, taught as the last
lesson, moves the player along that only exit — forward, into The Landing — and
because the effigy is a floor-stat non-retaliator, flee succeeds reliably. No
lesson branch intercepts, so `warcry`/`conditions`/`cooldowns`/`attack`/`cast`/
`trip`/`flee` all run for real (the immediate recast of `cast spike` is genuinely
blocked by the engine, showing the real recover message). Movement and premature
`flee` are blocked only until `tripped`.

**Files:**
- Modify (overwrite): `_datafiles/world/dogmud/rooms/newcomer_antechamber/6262.yaml`
- Modify (overwrite): `_datafiles/world/dogmud/behaviors/rooms/newcomer_antechamber/6262.yaml`

- [ ] **Step 1: Overwrite the room file**

`_datafiles/world/dogmud/rooms/newcomer_antechamber/6262.yaml`:
```yaml
roomid: 6262
zone: Newcomer Antechamber
title: The Proving
description: >
  A worn practice ring scuffed into the grey, and a straw effigy standing
  quiet at its center. 'Last lessons, and the ones that keep you breathing,'
  Dewey says. 'Before any fight, steel yourself -- type
  <ansi fg="command">warcry</ansi> and let the sound harden your nerve.'
biome: default
exits:
  north:
    roomid: 6467
spawninfo:
  - mobid: 9491
  - mobid: 9614
nouns:
  effigy: >-
    A man-shape of bound straw on a fixed post, charcoal smudges for eyes. It
    will stand and take whatever you give it and never lift a hand back -- the
    safest sparring partner you will ever have.
  ring: >-
    A rough circle scuffed into the grey floor by countless feet. Every soul
    who came through here learned to throw a punch, a spell, a shout in this
    same ring. The wear is oddly comforting.
  dewey: >-
    Dewey rolls both shoulders loose and nods you toward the effigy. 'No one
    ever died against this fellow. Best place there is to make your mistakes.'
idlemessages:
  - The effigy sways on its post and settles, straw whispering.
```

- [ ] **Step 2: Overwrite the room behavior tree**

`_datafiles/world/dogmud/behaviors/rooms/newcomer_antechamber/6262.yaml`:
```yaml
# 6262 The Proving — the combat primer. Sub-steps advance in strict order via
# per-instance flags. No lesson branch intercepts, so each taught command runs
# for real (including the deliberately-blocked recast). Only `flee` (after the
# lesson) is intercepted, to hand off to room 6467. There are no compass exits,
# so `flee` is the sole way out and is fully deterministic.

tree:
  type: selector
  children:

    # 1 ── warcry: applies the Warcry condition + starts the shared cooldown ──
    - type: sequence
      event: room_command
      children:
        - type: condition
          check: command_matches
          commands: [warcry]
        - type: condition
          check: state_equals
          key: warcried
          value: ""
        - type: action
          do: set_state
          key: warcried
          value: "1"
        - type: action
          do: mob_say
          mob_id: 9491
          text: >-
            Feel that lift? You just put an *effect* on yourself -- a good one.
            We call those *conditions*: some raise you up like this, others drag
            you down. Type <ansi fg="command">conditions</ansi> to see what's
            riding on you, and how long it lasts.

    # 2 ── conditions: inspect the buff just applied ──
    - type: sequence
      event: room_command
      children:
        - type: condition
          check: command_matches
          commands: [conditions, cond, affects, buffs]
        - type: condition
          check: state_equals
          key: warcried
          value: "1"
        - type: condition
          check: state_equals
          key: saw_conditions
          value: ""
        - type: action
          do: set_state
          key: saw_conditions
          value: "1"
        - type: action
          do: mob_say
          mob_id: 9491
          text: >-
            There's your war cry, ticking down. But shouting spent something,
            too -- your *focus*. Your biggest moves all draw from one well, and
            it takes a few rounds to refill. Type
            <ansi fg="command">cooldowns</ansi> to watch it come back.

    # 3 ── cooldowns: see the shared special-move timer ──
    - type: sequence
      event: room_command
      children:
        - type: condition
          check: command_matches
          commands: [cooldowns, cooldown, cd]
        - type: condition
          check: state_equals
          key: saw_conditions
          value: "1"
        - type: condition
          check: state_equals
          key: checked_cooldowns
          value: ""
        - type: action
          do: set_state
          key: checked_cooldowns
          value: "1"
        - type: action
          do: mob_say
          mob_id: 9491
          text: >-
            Good. Now, with your blood up -- let's fight. Type
            <ansi fg="command">attack effigy</ansi>. Your body swings on its own
            each round, and those normal swings never tire, well or no well.

    # 4 ── attack: enter combat with the effigy. Require "effigy" in the args so
    #      the flag only advances on a real target (a bare `attack` starts no
    #      combat, which would leave `cast spike` with nothing to hit).
    - type: sequence
      event: room_command
      children:
        - type: condition
          check: command_matches
          commands: [attack, kill, k, hit]
        - type: condition
          check: command_rest_contains
          keywords: [effigy]
        - type: condition
          check: state_equals
          key: checked_cooldowns
          value: "1"
        - type: condition
          check: state_equals
          key: attacked
          value: ""
        - type: action
          do: set_state
          key: attacked
          value: "1"
        - type: action
          do: mob_say
          mob_id: 9491
          text: >-
            Good -- those steady swings are yours for free. Now spend your focus
            on *belief*: type <ansi fg="command">cast spike</ansi> to drive a
            spike of raw conviction into it. No robes, no order, no permission --
            in Gaius, belief is a weapon.

    # 5 ── cast spike (first cast): belief ──
    - type: sequence
      event: room_command
      children:
        - type: condition
          check: command_matches
          commands: [cast, c]
        - type: condition
          check: command_rest_contains
          keywords: [spike]
        - type: condition
          check: state_equals
          key: attacked
          value: "1"
        - type: condition
          check: state_equals
          key: cast_spike
          value: ""
        - type: action
          do: set_state
          key: cast_spike
          value: "1"
        - type: action
          do: mob_say
          mob_id: 9491
          text: >-
            You felt that leave you. Now try <ansi fg="command">cast spike</ansi>
            again, straightaway -- go on.

    # 6 ── cast spike (immediate recast): blocked by the shared well ──
    - type: sequence
      event: room_command
      children:
        - type: condition
          check: command_matches
          commands: [cast, c]
        - type: condition
          check: command_rest_contains
          keywords: [spike]
        - type: condition
          check: state_equals
          key: cast_spike
          value: "1"
        - type: condition
          check: state_equals
          key: recast_blocked
          value: ""
        - type: action
          do: set_state
          key: recast_blocked
          value: "1"
        - type: action
          do: mob_say
          mob_id: 9491
          text: >-
            See? Same well you felt with the war cry. Belief just drew it down,
            so your next big move has to wait a breath. Let it fill -- then take
            its legs the body's way: type <ansi fg="command">trip</ansi>.

    # 7 ── trip: body move → prone (enemy condition) → progression + primer ──
    - type: sequence
      event: room_command
      children:
        - type: condition
          check: command_matches
          commands: [trip]
        - type: condition
          check: state_equals
          key: recast_blocked
          value: "1"
        - type: condition
          check: state_equals
          key: tripped
          value: ""
        - type: action
          do: set_state
          key: tripped
          value: "1"
        - type: action
          do: mob_say
          mob_id: 9491
          delay: 1.0
          text: >-
            Down it goes -- and now *it* wears a condition: knocked flat, easy
            pickings. Conditions cut both ways. And look what you've done: voice,
            belief, and body, all three, all from the same well. Every one of
            them is *yours* -- no class picks one and locks the rest away.
        - type: action
          do: grant_progression
          skill: spellcasting
          delay: 2.5
        - type: action
          do: mob_say
          mob_id: 9491
          delay: 3.5
          text: >-
            See that? You just got *better* -- not from any tally of kills or a
            level you climbed. There are no levels here, and no class boxing you
            in. You grow by *doing*: channel belief and your Willpower deepens;
            close in with fist or blade and your Dexterity sharpens; bend others
            with your voice and your Charisma rises. Over a longer road, the way
            you fight even pulls mutations toward you. You never pick it -- you
            earn it. <ansi fg="command">help skills</ansi> and
            <ansi fg="command">help mutations</ansi> hold the long version.
        - type: action
          do: mob_say
          mob_id: 9491
          delay: 5.0
          text: >-
            Last thing, and the most important: not every fight is worth
            finishing. When one turns against you, don't die proving a point --
            type <ansi fg="command">flee</ansi> to break away and run.

    # 8 ── flee (after the lesson): the farewell. NOT intercepted — the real
    #      flee command runs and carries the player out the room's only exit
    #      (north → the instance's 6467), so it is instance-safe and needs no
    #      move_player. The effigy is a floor-stat non-retaliator, so flee
    #      succeeds reliably; if it ever fails, the player simply retries (or
    #      walks north, which is now unblocked).
    - type: sequence
      event: room_command
      children:
        - type: condition
          check: command_matches
          commands: [flee, run, escape]
        - type: condition
          check: state_equals
          key: tripped
          value: "1"
        - type: action
          do: mob_say
          mob_id: 9491
          text: >-
            That's it -- running isn't losing. It's how you live long enough to
            win the next one. Go on, then -- I'll see you through on the far side.

    # 9 ── flee too early: block until the lesson is done ──
    - type: sequence
      event: room_command
      children:
        - type: condition
          check: command_matches
          commands: [flee, run, escape]
        - type: condition
          check: state_equals
          key: tripped
          value: ""
        - type: action
          do: mob_say
          mob_id: 9491
          text: >-
            Not yet -- finish the lesson and I'll teach you the smartest move
            there is. Keep at the effigy.
        - type: action
          do: intercept

    # 10 ── movement before the lesson is done: nudge + block. After `tripped`
    #      is set this branch no longer matches, so `north` (and a successful
    #      `flee`) carry the player forward to 6467.
    - type: sequence
      event: room_command
      children:
        - type: condition
          check: command_matches
          commands: [north, n, go, south, s, east, e, west, w, up, u, down, d]
        - type: condition
          check: state_equals
          key: tripped
          value: ""
        - type: action
          do: mob_say
          mob_id: 9491
          text: >-
            Not through here yet -- finish at the effigy first. When you're done,
            I'll show you how to leave a fight the smart way.
        - type: action
          do: intercept
```

- [ ] **Step 3: Verify both files parse**

Run: `python -c "import yaml; yaml.safe_load(open('_datafiles/world/dogmud/rooms/newcomer_antechamber/6262.yaml')); yaml.safe_load(open('_datafiles/world/dogmud/behaviors/rooms/newcomer_antechamber/6262.yaml'))" && echo OK`
Expected: `OK`

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/rooms/newcomer_antechamber/6262.yaml _datafiles/world/dogmud/behaviors/rooms/newcomer_antechamber/6262.yaml
git commit -m "feat(tutorial): room 5 The Proving (conditions/cooldown/playstyle/flee primer)"
```

---

## Task 10: Room 6 — The Landing (6467): handoff to the Awakening Pool

New room. The player arrives here via `flee` from room 5 (the `move_player`
auto-queues a `look`, so the arrival prose shows immediately). The prose carries
Dewey's arrival line + the final `talk dewey` instruction; the tree intercepts
`talk`/`say` to vortex the player to room 5200 (mirrors the old 6262 terminus).

**Files:**
- Create: `_datafiles/world/dogmud/rooms/newcomer_antechamber/6467.yaml`
- Create: `_datafiles/world/dogmud/behaviors/rooms/newcomer_antechamber/6467.yaml`

- [ ] **Step 1: Create the room file**

`_datafiles/world/dogmud/rooms/newcomer_antechamber/6467.yaml`:
```yaml
roomid: 6467
zone: Newcomer Antechamber
title: The Landing
description: >
  The grey thins to almost nothing here, and ahead a pale blue-green light
  pools like the mouth of a well -- the real world, close enough to step into.
  Dewey is already waiting, unhurried. 'See? Still breathing,' Dewey says,
  smiling. 'You've got everything you need to start. When you're ready, type
  <ansi fg="command">talk dewey</ansi> and I'll send you through. The world's
  waiting.'
biome: default
exits: {}
spawninfo:
  - mobid: 9491
nouns:
  light: >-
    A pool of steady blue-green light hangs where the grey gives out -- the
    same glow you glimpsed under the still water, but whole and near now. It is
    the Awakening Pool, and the world around it, waiting for you to arrive.
  archway: >-
    Not so much a door as a thinning, a place where this grey between-world
    wears through into the real one. Step through it and you wake for good.
  dewey: >-
    Dewey stands easy at the threshold, hands in pockets. 'I stay here. There's
    always another one waking up. But you -- you go on. You'll do fine.'
idlemessages:
  - The pool of light ahead pulses once, slow and warm, like a welcome.
```

- [ ] **Step 2: Create the room behavior tree**

`_datafiles/world/dogmud/behaviors/rooms/newcomer_antechamber/6467.yaml`:
```yaml
# 6467 The Landing — terminus. talk/say → farewell → vortex to the Awakening
# Pool (5200), where quest 30 begins with Cleric Hadwen. Intercept so the
# original talk/say doesn't re-run in 5200. Movement is caught and redirected.

tree:
  type: selector
  children:

    # ── handoff ──
    - type: sequence
      event: room_command
      children:
        - type: condition
          check: command_matches
          commands: [talk, say]
        - type: action
          do: mob_say
          mob_id: 9491
          text: >-
            That's the last of it -- and welcome to Gaius. Go on, now.
        - type: action
          do: move_player
          room_id: 5200
        - type: action
          do: intercept

    # ── movement: redirect to the handoff (catches `go <dir>` too) ──
    - type: sequence
      event: room_command
      children:
        - type: condition
          check: command_matches
          commands: [north, n, go, south, s, east, e, west, w, up, u, down, d]
        - type: action
          do: mob_say
          mob_id: 9491
          text: >-
            One step left, and it's not a walk -- type
            <ansi fg="command">talk dewey</ansi> and I'll carry you through.
        - type: action
          do: intercept
```

- [ ] **Step 3: Verify both files parse**

Run: `python -c "import yaml; yaml.safe_load(open('_datafiles/world/dogmud/rooms/newcomer_antechamber/6467.yaml')); yaml.safe_load(open('_datafiles/world/dogmud/behaviors/rooms/newcomer_antechamber/6467.yaml'))" && echo OK`
Expected: `OK`

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/rooms/newcomer_antechamber/6467.yaml _datafiles/world/dogmud/behaviors/rooms/newcomer_antechamber/6467.yaml
git commit -m "feat(tutorial): room 6 The Landing (handoff to Awakening Pool)"
```

---

## Task 11: Re-theme the guide dialogue tree into Dewey's voice

The dialogue file (keyed by mob id, filename stays `9491.yaml`) powers
`ask dewey <topic>` in room 4 and anywhere Dewey stands. Re-voice it as Dewey
(first person, warm) and add the topics the tutorial invites: world, pools, help,
look, conditions, growth/no-levels.

**Files:**
- Modify (overwrite): `_datafiles/world/dogmud/dialogue/newcomer_antechamber/9491.yaml`

- [ ] **Step 1: Overwrite the dialogue file**

`_datafiles/world/dogmud/dialogue/newcomer_antechamber/9491.yaml`:
```yaml
mobid: 9491
zone: Newcomer Antechamber
defaultMood: friendly

patterns:
  - keywords: ["hello", "hi", "greet", "hey", "hail"]
    responses:
      - "Hello yourself. Ask me anything while you've got me -- the world, the
          pools, how you'll grow, or just tell me you're ready."

  - keywords: ["world", "gaius", "outside", "beyond"]
    responses:
      - "Gaius. A whole living world -- a city, roads, people who lie and
          help and everything between. It changes while you're not looking.
          You'll love it and it'll scare you, both. That's right and proper."

  - keywords: ["pool", "pools", "health", "stamina", "conviction"]
    responses:
      - "Three pools sit under your six stats: health, stamina, conviction.
          Spend them and they refill with time and rest. Type
          <ansi fg=\"command\">status</ansi> anytime to read them."

  - keywords: ["help", "command", "commands", "forget", "lost"]
    responses:
      - "Forget a command and the game remembers for you. Type
          <ansi fg=\"command\">help</ansi> for the whole index, or
          <ansi fg=\"command\">help</ansi> and a topic for one thing. No shame
          in it -- I still look things up."

  - keywords: ["look", "see", "examine", "read"]
    responses:
      - "Looking is the first skill and the last one you'll outgrow. Type
          <ansi fg=\"command\">look</ansi> in any room, and
          <ansi fg=\"command\">look</ansi> at anything that catches your eye.
          This world hides its best parts in plain sight."

  - keywords: ["condition", "conditions", "buff", "effect"]
    responses:
      - "Conditions are the temporary things riding on you -- a war cry
          steadying your hand, a poison in your blood, a leg swept out from
          under you. Good and bad both. Type
          <ansi fg=\"command\">conditions</ansi> to see what's on you."

  - keywords: ["grow", "growth", "level", "levels", "class", "progress", "mutation", "mutations"]
    responses:
      - "No levels here, and no class to cage you. You grow by *doing* --
          swing a blade and you sharpen, cast and you deepen, shout and you
          steady. Over time the way you fight even pulls mutations toward you.
          You never pick what you become. You earn it."

  - keywords: ["ready", "go", "leave", "through", "step", "done"]
    responses:
      - "Then don't let me keep you. Step on through when you reach the
          landing -- and welcome to Gaius. I'll be right behind you in spirit."

  - keywords: [""]
    responses:
      - "Ask me about the world, the pools, conditions, how you'll grow -- or
          tell me you're ready."
      - "I'm patient, and I've got nowhere to be. Ask what you need."

tree:
  root:
    text: "Ask me anything -- the world, the pools, conditions, how you grow --
      or tell me you're ready and I'll see you through."
    hints: "You could ask Dewey about the world, the pools, conditions, or how
      you'll grow. When you're ready, say so."

memory:
  expiryPeriod: ""
```

- [ ] **Step 2: Verify the file parses**

Run: `python -c "import yaml; yaml.safe_load(open('_datafiles/world/dogmud/dialogue/newcomer_antechamber/9491.yaml'))" && echo OK`
Expected: `OK`

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/dialogue/newcomer_antechamber/9491.yaml
git commit -m "feat(tutorial): re-voice guide dialogue as Dewey + add topics"
```

---

## Task 12: Boot test + full walk-through verification

Data-file errors (name/filename mismatch, bad trigger, ID collision) only surface
at server startup and in play — this is the real acceptance gate.

- [ ] **Step 1: Full test + build**

Run: `go build ./... && go test ./internal/behaviortree/ -run TestGrantProgression`
Expected: build clean; tests PASS.

- [ ] **Step 2: Wipe instance saves (SOP)**

Run:
```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
```

- [ ] **Step 3: Boot the server and watch for a clean load**

Run: `go run . 2>&1 | tee /tmp/dogmud-boot.log` (stop after startup completes)
Expected: no panic; lines like `mobs.LoadDataFiles() loadedCount=...`,
`rooms.LoadDataFiles() ...`, `dialogue ... loadedCount=...` appear. Specifically
confirm **no** `filesystem path ... did not end in Filepath()` panic (would mean
the Dewey mob/goals filename doesn't match the name field) and **no** unknown
behavior action/condition errors for `grant_progression`.

- [ ] **Step 4: Live walk-through (fresh character, route 1)**

Connect a client, create a new character, and choose **"1) New to text MUDs"**.
Walk all six rooms and confirm each gate + line. Verify in order:

- Room 6258: `look` → Dewey praises + prompts `look hands`; `look hands` → advances,
  north opens. Try `north` before examining → blocked with a nudge. Each noun
  (`look hands`, `look grey`, `look stone`, `look dewey`) returns lore prose.
  Every command word in Dewey's lines renders cyan.
- Room 6259: `status` → progression teaser; `north` gated until `status`.
- Room 6260: `get token` → `inventory` → `help`, each gated in order; `north`
  gated until `help`.
- Room 6261: `say hello` → `ask dewey world` (Dewey's dialogue answers AND the
  lesson advances); `north` gated until asked.
- Room 6262: `warcry` → the **Warcry** condition applies; `conditions` lists it
  with a countdown; `cooldowns` shows the special-move timer; `attack effigy`
  starts combat; `cast spike` hits; an **immediate** `cast spike` recast is
  **blocked** with the real recover message; after the well refills, `trip` lands
  and the effigy is **prone**; the **SKILL ADVANCEMENT** banner appears **once**;
  Dewey delivers the primer + flee prompt; `flee` carries you to room 6467. Confirm
  the effigy never dies and never flees, and that `flee` before `trip` is blocked.
- Room 6467: arrival prose shows Dewey's line; `talk dewey` vortexes you to room
  5200 (The Awakening Pool) with Cleric Hadwen present and quest 30 active
  (`quests` shows "The Awakening").

- [ ] **Step 5: If any step fails, fix the responsible file and re-run from Step 2.**
Common causes: a stale instance save shadowing an edit (re-run Step 2); a mismatched
mob filename vs `name:` (Task 3/4); a wrong command alias in a `command_matches`
list; `cast spike` needing an explicit target (if so, change the room-5 prompts and
`command_rest_contains` handling to `cast spike effigy`).

- [ ] **Step 6: Final commit (if fixes were needed)**

```bash
git add -A
git commit -m "fix(tutorial): walk-through corrections for the newcomer antechamber"
```

---

## Notes for the implementer

- **Data > code here.** Only Task 1 is Go. Everything else is YAML the engine
  hot-loads at boot; there is no unit-test harness for room content — the boot
  test + walk-through (Task 12) is the acceptance gate. Do NOT skip the instance-
  save wipe before each boot, or stale saves will shadow your edits.
- **`cast spike` targeting.** In combat the caster should auto-target the current
  enemy. If the live walk-through shows `cast spike` demanding an explicit target,
  switch the room-5 copy and lesson to `cast spike effigy` (the
  `command_rest_contains: [spike]` check still matches). Flagged in Task 12 Step 5.
- **Effigy longevity.** If the walk-through ever kills the effigy, raise its
  `vitality.training` further (Task 4). There is no invulnerability flag; HP is the
  only lever.
- **Delays in room 5.** The `delay:` params on the post-`trip` lines sequence the
  "down it goes" line, the banner, the primer, and the flee prompt so they land in
  order after the trip resolves (same mechanism as the Hadwen rite). Adjust the
  offsets if the pacing feels off in play.
```