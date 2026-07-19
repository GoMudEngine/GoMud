# Newcomer Tutorial Chain Quest — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rebuild the route-1 newcomer tutorial as a linear chain quest (id 28,
"Waking to Gaius") the player follows one hand-held step at a time — adding the
missing `command_issued` quest primitive so the tutorial is quest-driven, not
behavior-tree-driven.

**Architecture:** A new non-intercepting `command_issued` quest event fires from
`TryCommand` for every dispatched command. Quest 28 uses it (plus `room_enter`)
to advance; Dewey speaks via `npc_say`; the six room behavior trees shrink to a
single exit-intercept branch each (gated on quest tokens). The effigy is made
unkillable. No change to the existing `command` event or any existing quest.

**Tech Stack:** Go (quest engine + command dispatch); YAML quest/mob/behavior
data files; `go test`; the `mudagent` playtest harness.

**Spec:** `docs/superpowers/specs/2026-07-19-newcomer-tutorial-quest-design.md`

---

### Task 1: Engine — the `command_issued` quest primitive (TDD)

**Files:**
- Modify: `internal/questengine/loader.go` (whitelist the event)
- Modify: `internal/usercommands/usercommands.go` (fire it in `TryCommand`)
- Test: `internal/questengine/loader_test.go` (event accepted) and
  `internal/questengine/engine_test.go` (trigger matches)

- [ ] **Step 1: Write the failing loader test** — `command_issued` is a valid event.

In `internal/questengine/loader_test.go`, add:
```go
func TestCommandIssuedIsValidEvent(t *testing.T) {
	q := &QuestData{
		QuestId: 999,
		Name:    "T",
		Steps:   []StepDef{{Id: "start"}},
		Triggers: []TriggerDef{{
			Event:   "command_issued",
			Command: "look",
			Actions: []ActionDef{{Grant: "999-start"}},
		}},
	}
	if err := validateQuest(q); err != nil {
		t.Fatalf("command_issued should be a valid event, got: %v", err)
	}
}
```
(Confirm the real validator function name — it is the function in `loader.go`
that owns the `validEvents` map, near the "invalid event" error. Match the
existing `QuestData`/`StepDef`/`TriggerDef` field names used elsewhere in
`loader_test.go`.)

- [ ] **Step 2: Run it — expect FAIL** (invalid event `"command_issued"`).

Run: `go test ./internal/questengine/ -run TestCommandIssuedIsValidEvent -v`
Expected: FAIL — `trigger 0 has invalid event "command_issued"`.

- [ ] **Step 3: Whitelist the event.**

In `internal/questengine/loader.go`, add `command_issued` to `validEvents`:
```go
	validEvents := map[string]bool{
		"room_enter": true, "item_give": true, "skill_use": true,
		"mob_death": true, "command": true, "item_gain": true,
		"dialogue": true, "quest_granted": true, "room_interact": true,
		"command_issued": true,
	}
```

- [ ] **Step 4: Run it — expect PASS.**

Run: `go test ./internal/questengine/ -run TestCommandIssuedIsValidEvent -v`
Expected: PASS.

- [ ] **Step 5: Write the failing engine match test** — a `command_issued` Notify
  advances a quest whose trigger matches on command + noun.

In `internal/questengine/engine_test.go`, add a test modeled on the existing
Notify tests in that file (reuse their fixture/setup helpers). It should:
register a quest with a `command_issued`/`command: look`/`noun: guide` trigger
granting a token, `Notify("command_issued", EventDetails{Command:"look",
Noun:"guide", ...})`, and assert the token was granted / `result.Handled` is
true. Also assert a `Notify` with `Noun:"grey"` does NOT match (noun filter).
```go
func TestCommandIssuedTriggerMatchesNoun(t *testing.T) {
	// ... build engine with a quest:
	//   trigger{event:"command_issued", command:"look", noun:"guide",
	//           actions:[grant "<qid>-examined"]}
	// grant the player the prior step so IsTokenAfter allows the advance.
	// Notify("command_issued", {Command:"look", Noun:"guide"}) -> Handled==true,
	//   QuestProgress advanced to "examined".
	// Notify("command_issued", {Command:"look", Noun:"grey"}) -> not handled.
}
```
(Fill the body using the same construction the neighbouring tests use — do not
invent new helpers.)

- [ ] **Step 6: Run it — expect FAIL** (no wiring yet is fine; this tests the
  matcher, which already supports Command/Noun, so it may PASS immediately —
  that is acceptable, it locks the contract). If it fails for a fixture reason,
  fix the fixture, not the engine.

Run: `go test ./internal/questengine/ -run TestCommandIssuedTriggerMatchesNoun -v`

- [ ] **Step 7: Fire `command_issued` from `TryCommand`.**

In `internal/usercommands/usercommands.go`, in the `userCommands[cmd]` dispatch
block, replace the run-and-return (currently around line 520–522):
```go
			// Run the command here
			handled, err := cmdInfo.Func(rest, user, room, flags)
			return handled, err
```
with:
```go
			// Run the command here
			handled, err := cmdInfo.Func(rest, user, room, flags)
			// Quest engine: non-intercepting "command_issued" — the player typed
			// this command (distinct from the action-success "command" event).
			// Lets any quest gate a step on a command without instrumenting each
			// handler. Fire only on a clean dispatch; ignore the result (the
			// command has already run — this never intercepts).
			if handled && err == nil {
				if qRoom := rooms.LoadRoom(user.Character.RoomId); qRoom != nil {
					b := questengine.NewGameBridge(user, qRoom.RoomId)
					questengine.GetEngine().Notify("command_issued", questengine.EventDetails{
						UserId:  user.UserId,
						RoomId:  qRoom.RoomId,
						Command: cmd,
						Noun:    strings.ToLower(strings.TrimSpace(rest)),
					}, b, b)
				}
			}
			return handled, err
```
(`questengine`, `rooms`, and `strings` are already imported in this file — the
existing `room_interact` Notify above uses the same pattern.)

- [ ] **Step 8: Build + run the questengine tests.**

Run: `go build ./... && go test ./internal/questengine/ ./internal/usercommands/ 2>&1 | tail -20`
Expected: build OK; both packages pass.

- [ ] **Step 9: Commit.**

```bash
git add internal/questengine/loader.go internal/questengine/loader_test.go \
        internal/questengine/engine_test.go internal/usercommands/usercommands.go
git commit -m "feat(questengine): non-intercepting command_issued event

Fires from TryCommand for every dispatched command (Command + Noun), letting
quests gate a step on 'the player typed X' without instrumenting each handler.
Distinct from the action-success 'command' event; no existing quest affected."
```

---

### Task 2: Quest 28 — steps definition (loads, hint works)

**Files:**
- Create: `_datafiles/world/dogmud/quests/28-waking_to_gaius.yaml`

- [ ] **Step 1: Confirm id 28 is free.**

Run: `python tools/id_inventory.py --type quests`
Expected: 28 listed in the free gap (22–28).

- [ ] **Step 2: Write the quest with its 22 steps (no triggers yet).**

Create `_datafiles/world/dogmud/quests/28-waking_to_gaius.yaml`:
```yaml
questid: 28
name: Waking to Gaius
description: >-
  You wake in the grey between-place with a guide named Dewey beside you.
  Learn the first handful of things every soul needs before it steps into
  the world -- one at a time, at your own pace.
secret: false
linear: true

steps:
  - id: start
    description: "You have woken, and Dewey has marked your path."
    hint: "Type hint to see your next step -- or quests to see the whole journey."
  - id: hinted
    description: "You read your first hint. That command is your compass now."
    hint: "Take stock of where you stand -- type look."
  - id: looked
    description: "You took in the room around you."
    hint: "Look closer at one thing -- type look guide."
  - id: examined
    description: "You read the guide closely. Reading is the first survival skill."
    hint: "The way north is open -- type north."
  - id: status_prompt
    description: "You stepped into the place where you meet yourself."
    hint: "Read yourself -- type status."
  - id: statused
    description: "You read your six stats and three pools."
    hint: "The way north is open -- type north."
  - id: carry_prompt
    description: "Small things matter out there."
    hint: "Pick that up -- type get token."
  - id: got
    description: "The token is yours now."
    hint: "See what you carry -- type inventory."
  - id: invd
    description: "You checked what you are carrying."
    hint: "When you forget a command, the game remembers -- type help."
  - id: helped
    description: "You opened the help index -- your patient, permanent reference."
    hint: "The way north is open -- type north."
  - id: speaks_prompt
    description: "Out in the world, people answer when you speak."
    hint: "Speak up -- type say hello."
  - id: said
    description: "You spoke, and the world can hear you."
    hint: "Ask me something directly -- type ask dewey world."
  - id: asked
    description: "You learned to ask -- half of what matters is learned that way."
    hint: "The way north is open -- type north."
  - id: proving_prompt
    description: "The practice ring -- where you make your mistakes safely."
    hint: "Steel yourself before a fight -- type warcry."
  - id: warcried
    description: "Your war cry steadied your nerve -- a condition, riding on you."
    hint: "See what is riding on you -- type conditions."
  - id: saw_conditions
    description: "You read your conditions. They cut both ways, good and bad."
    hint: "See which big moves are resting -- type cooldowns."
  - id: checked_cooldowns
    description: "You checked your cooldowns -- your big moves share one well."
    hint: "Now fight -- type attack effigy."
  - id: attacked
    description: "You closed with the effigy. Your steady swings are free."
    hint: "Drive belief into it -- type cast spike."
  - id: cast_spike
    description: "You loosed a spike of conviction. Belief is a weapon here."
    hint: "Take its legs the body's way -- type trip."
  - id: tripped
    description: "You put the effigy down -- voice, belief, and body, all yours."
    hint: "Not every fight is worth finishing -- type flee."
  - id: fled
    description: "You broke away. Running is how you live to win the next one."
    hint: "One step left, and it is not a walk -- type talk dewey."
  - id: end
    description: "Dewey sent you through to the world. Gaius is waiting."

rewards:
  playermessage: >-
    Dewey's voice stays with you a moment after the grey lets go -- steady,
    warm, already a little proud. You are awake now, and the world is real.
```

- [ ] **Step 3: Boot-test that the quest loads.**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . > /c/tmp/tut-boot.log 2>&1 &
# wait for the quest-load line, then check no panic:
until grep -qa "quests.LoadDataFiles" /c/tmp/tut-boot.log; do sleep 2; done
grep -aE "quests.LoadDataFiles|panic:|Waking to Gaius|quest 28" /c/tmp/tut-boot.log | head
taskkill //F //IM GoMud.exe 2>/dev/null; taskkill //F //IM go.exe 2>/dev/null; true
```
Expected: `loadedCount` +1 vs. before; no panic. A linear quest with steps and no
triggers loads fine (advancement comes from Task 3 + the btree grant of `start`).

- [ ] **Step 4: Commit.**

```bash
git add _datafiles/world/dogmud/quests/28-waking_to_gaius.yaml
git commit -m "content(quest): Waking to Gaius (28) tutorial steps + hints"
```

---

### Task 3: Quest 28 — triggers (advancement + Dewey's voice + handoff)

**Files:**
- Modify: `_datafiles/world/dogmud/quests/28-waking_to_gaius.yaml` (add `triggers:`)

Trigger archetypes (each `command:` value is the alias-resolved command; `has`/
`missing` enforce order defensively on top of `linear`):

**A. Room-arrival prompt** (grant the room's first token + Dewey greets). Example
for 6259:
```yaml
  - event: room_enter
    room: 6259
    conditions:
      has: [28-examined]
      missing: [28-status_prompt]
    actions:
      - grant: 28-status_prompt
      - npc_say:
          mob: 9491
          lines:
            - {delay: 1, text: "This is where you meet yourself. When you are
                ready, read yourself -- type status."}
```

**B. Command lesson** (advance + Dewey reacts, pointing at the single next
action). Example for the `status` step:
```yaml
  - event: command_issued
    command: status
    conditions:
      has: [28-status_prompt]
      missing: [28-statused]
    actions:
      - grant: 28-statused
      - npc_say:
          mob: 9491
          lines:
            - {delay: 0, text: "There you are -- all of you on one page. Those
                grow by use, not by any level. The way north is open."}
```

**C. Quest grant on first arrival** (6258):
```yaml
  - event: room_enter
    room: 6258
    conditions:
      missing: [28-start, 28-end]
    actions:
      - grant: 28-start
      - npc_say:
          mob: 9491
          lines:
            - {delay: 1, text: "There you are -- waking up at last. Easy, now.
                Nothing here can harm you."}
            - {delay: 3, text: "I have marked your path already. Type hint to
                see your first step -- I will be right here."}
```

**D. Combat close** (`tripped` step — paced multi-beat, teaches progression as a
truth, no fabricated event):
```yaml
  - event: command_issued
    command: trip
    conditions:
      has: [28-cast_spike]
      missing: [28-tripped]
    actions:
      - grant: 28-tripped
      - npc_say:
          mob: 9491
          lines:
            - {delay: 1, text: "Down it goes -- and now it wears a condition of
                its own, knocked flat. Voice, belief, and body, all three yours."}
            - {delay: 4, text: "Out here you grow by doing -- channel belief and
                your Willpower deepens, close with a blade and your Dexterity
                sharpens. You never pick it. You earn it."}
            - {delay: 8, text: "Last thing, and the most important -- not every
                fight is worth finishing. Type flee to break away and live."}
```

**E. Handoff** (`end` step on `talk` in 6467 — farewell, teleport, grant 30):
```yaml
  - event: command_issued
    command: talk
    room: 6467
    conditions:
      has: [28-fled]
      missing: [28-end]
    actions:
      - grant: 28-end
      - npc_say:
          mob: 9491
          lines:
            - {delay: 1, text: "That is the last of it. Steady, now -- this part
                feels strange the first time, but it is gentle. I have you."}
      - teleport: 5200
      - grant: 30-start
```

- [ ] **Step 1: Author all 22 triggers** into the `triggers:` block, following
  archetypes A–E and the spec's step table. One trigger per row of that table
  (steps 5/7/11/14 use archetype A; 2/3/4/6/8/9/10/12/13/15/16/17/18/19 use B;
  step 1 uses C; step 20 uses D; step 22 uses E; step 21 `fled` uses B on
  `command: flee` — its `npc_say` is the "running isn't losing" line, and no
  lock is needed since flee itself moves the player to 6467).

  **Dewey line inventory** (one `npc_say` per step; de-compounded — each names at
  most the single next action; adapt from the current room behavior-tree text
  which is already in his voice). Authoring rules: **no semicolons, no literal
  asterisks** (both break/garble the underlying `say`); 80-col; first-person;
  no hard numbers. Use `has:`/`missing:` on every trigger as shown so an
  out-of-order `command_issued` cannot skip a step.

- [ ] **Step 2: Boot-test the full quest loads with triggers.**

Same boot procedure as Task 2 Step 3. Expected: no panic; `command_issued` and
`room_enter` triggers accepted (Task 1 whitelisted the event).

- [ ] **Step 3: Commit.**

```bash
git add _datafiles/world/dogmud/quests/28-waking_to_gaius.yaml
git commit -m "content(quest): Waking to Gaius triggers -- Dewey voice + handoff"
```

---

### Task 4: Effigy — make it unkillable

**Files:**
- Modify: `_datafiles/world/dogmud/mobs/newcomer_antechamber/9614-straw_effigy.yaml`

- [ ] **Step 1: Raise the effigy's vitality far past any tutorial session.**

Find:
```yaml
    vitality:
      training: 1000
```
Replace with:
```yaml
    vitality:
      # Effectively unkillable for the lesson. The effigy deals no damage, and
      # the player is meant to practice on and FLEE it, never kill it -- a live
      # target must remain for the cast-spike and trip steps. (Was 1000; a slow,
      # confused player could still drop that, stranding the fight -- Malia's bug.)
      training: 100000
```

- [ ] **Step 2: Commit.**

```bash
git add _datafiles/world/dogmud/mobs/newcomer_antechamber/9614-straw_effigy.yaml
git commit -m "fix(tutorial): make the straw effigy unkillable during the lesson"
```

---

### Task 5: Behavior trees — reduce to exit-intercept only

The six room trees no longer detect commands or speak (the quest does both). Each
keeps ONLY a movement-intercept branch that blocks a premature exit until the
room's last lesson token is held, with a single-action Dewey nudge pointing at
the tool.

**Files:**
- Modify: `_datafiles/world/dogmud/behaviors/rooms/newcomer_antechamber/`
  `{6258,6259,6260,6261,6262,6467}.yaml`

Per-room "last lesson token" to gate on (block north while this token is MISSING):

| Room | Gate on missing token |
|---|---|
| 6258 | `28-examined` |
| 6259 | `28-statused` |
| 6260 | `28-helped` |
| 6261 | `28-asked` |
| 6262 | `28-tripped` |
| 6467 | `28-end` |

- [ ] **Step 1: Replace each of the six files with an exit-intercept-only tree.**

Template (example for 6259; substitute the room's gate token):
```yaml
# 6259 Knowing Yourself — movement gate only. All lessons + Dewey's voice live
# in quest 28. Block the exit until the room's lesson (28-statused) is done.
tree:
  type: selector
  children:
    - type: sequence
      event: room_command
      children:
        - type: condition
          check: command_matches
          commands: [north, n, go, south, s, east, e, west, w, up, u, down, d]
        - type: condition
          check: player_missing_quest
          quest: "28-statused"
        - type: action
          do: mob_say
          mob_id: 9491
          text: >-
            Not through yet -- if you have lost the thread, type
            <ansi fg="command">hint</ansi> and it will point you at the next step.
        - type: action
          do: intercept
```
For **6467**, gate on `28-end` and change the nudge to point at `talk dewey`
(the handoff), since that room's "exit" is the `talk` handoff, not a walk:
```yaml
          text: >-
            One step left, and it is not a walk -- type
            <ansi fg="command">talk dewey</ansi> and I will carry you through.
```
(Confirm `player_missing_quest` takes a `quest:` param in the room-tree schema —
it is registered in `conditions.go`; match the param key other trees use.)

- [ ] **Step 2: Boot-test — trees load, no panic.**

Same boot procedure. Expected: all six trees load; no panic.

- [ ] **Step 3: Commit.**

```bash
git add _datafiles/world/dogmud/behaviors/rooms/newcomer_antechamber/
git commit -m "refactor(tutorial): room trees keep only the exit gate; quest 28 drives the rest"
```

---

### Task 6: Verification — boot clean + existing-quest regression check

**Files:** none.

- [ ] **Step 1: Full boot test (instance saves nuked).**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . > /c/tmp/tut-boot.log 2>&1 &
until grep -qa "quests.LoadDataFiles" /c/tmp/tut-boot.log; do sleep 2; done
sleep 5
grep -acE "panic:" /c/tmp/tut-boot.log   # expect 0
grep -aE "Waking to Gaius|loadedCount" /c/tmp/tut-boot.log | head
```
Leave the server running for Step 2.

- [ ] **Step 2: Confirm the existing `command` event still fires (no regression).**

Drive the harness as admin (`quester4` won't work — use the local `smoketester`
elevated to admin as in the trail-quest run, then revert) OR simply grant First
Blood tokens and confirm its `command: kick`/`trip` steps still advance — the
`command` event and its handlers were untouched, so this is a smoke confirmation.
Kill the server after.

---

### Task 7: Adversarial content playtest (mandatory SOP gate) + fixes

**Files:** none (drives fixes back into Tasks 2–5 as needed).

- [ ] **Step 1: Fresh-newbie route-1 playtest.**

Run `/playtest local bug-finder` (or drive `mudagent` directly) creating a BRAND
NEW character via the new-player flow so it lands in the antechamber (6258) as a
"complete noob". Converge ASCII. With a critical, adversarial mandate, walk the
ENTIRE tutorial reading every line, and verify:
- On arrival `hint` and `quests` immediately show "Waking to Gaius" and step 1.
- Every Dewey prompt is a SINGLE action (no compound instructions).
- `hint` re-shows the current step at any point — test it mid-combat, after the
  `attack effigy` step.
- The effigy cannot be killed; `cast spike` and `trip` always have a target.
- A premature `north` in each room is blocked with the "type hint" nudge, and
  passes once the room's lesson is done.
- The `talk dewey` handoff teleports to 5200, grants quest 30, completes quest
  28, and Cleric Hadwen is present.

- [ ] **Step 2: Fix findings, re-run until clean.**

Drive fixes into the quest triggers (Task 3), effigy (Task 4), or gate trees
(Task 5). Re-boot (nuke instances) and re-run Step 1 until the walk is clean.

- [ ] **Step 3: Report + hand to the user for their own playtest.**

Write the report to `tools/playtest/reports/`. Do NOT claim done on a clean boot
alone — the human walk is the gate. Then hand to the user.

---

## Notes for the implementer

- **Instance-save SOP:** nuke `mobs.instances/*` + `rooms.instances/*` before
  every smoke test. Never delete `shops/`/`guilds/`.
- **Windows env:** `go run .` / server / `git` are windowless and safe to run
  from the main loop; do not spawn shell-denied subagents for them.
- **Dewey's voice:** adapt the existing room-behavior-tree `mob_say` text (it is
  already in his warm first-person voice) — the job is to *de-compound* it (one
  action per line) and move it into the quest `npc_say` actions, not to rewrite
  his character.
- **Verify before asserting:** the `command_issued` wiring's real proof is the
  Task 7 walk (does typing `look` actually advance the quest) — the unit tests
  cover the engine contract, the playtest covers the dispatch wiring.
