# Critical Blockers Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix three critical prod bugs: death loop in Shadow Realm, broken spell scripts for player casts, and missing quest spell rewards.

**Architecture:** Three independent fixes sharing one engine improvement (spell script wiring). The death loop fix is purely Go-side. Spell scripts require wiring `TrySpellScriptEvent` into the player cast pipeline. Quest rewards add a `SpellId` field to the reward struct.

**Tech Stack:** Go, JavaScript (goja VM), YAML data files

---

## Task 1: Death Loop — Health Guard on Reciprocal Aggro

**Files:**
- Modify: `internal/hooks/NewRound_DoCombat_helpers.go:1165`

- [ ] **Step 1: Add health check to reciprocal aggro assignment**

In `internal/hooks/NewRound_DoCombat_helpers.go`, find line 1165:

```go
	// Reciprocal aggro
	if defUser.Character.Aggro == nil {
		defUser.Character.SetAggro(0, mob.InstanceId, characters.DefaultAttack)
	}
```

Replace with:

```go
	// Reciprocal aggro — skip dead/downed players to prevent stale aggro in Shadow Realm
	if defUser.Character.Health > 0 && defUser.Character.Aggro == nil {
		defUser.Character.SetAggro(0, mob.InstanceId, characters.DefaultAttack)
	}
```

- [ ] **Step 2: Build and verify**

Run: `go build ./...`
Expected: Clean build, no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/hooks/NewRound_DoCombat_helpers.go
git commit -m "fix: guard reciprocal aggro against dead players

Mobs could re-aggro a player who hit 0 HP earlier in the same
combat round, before handleAffected() ran suicide to clear it.
This caused players to arrive in the Shadow Realm with stale
aggro, permanently stuck.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: Death Loop — Safety Net in suicide.go

**Files:**
- Modify: `internal/usercommands/suicide.go:200`

- [ ] **Step 1: Add post-move aggro clear**

In `internal/usercommands/suicide.go`, find line 200:

```go
	rooms.MoveToRoom(user.UserId, int(configs.GetSpecialRoomsConfig().DeathRecoveryRoom))
```

Add after it:

```go
	// Belt-and-suspenders: re-clear aggro after room move in case any
	// code path (e.g., mob combat round processing) assigned aggro
	// between our first clear (line 179) and the room move.
	user.Character.Aggro = nil
```

- [ ] **Step 2: Build and verify**

Run: `go build ./...`
Expected: Clean build, no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/usercommands/suicide.go
git commit -m "fix: re-clear aggro after death room move as safety net

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: Death Loop — Escape Hatch in go.go

**Files:**
- Modify: `internal/usercommands/go.go:41-44`

- [ ] **Step 1: Skip combat check in death recovery room**

In `internal/usercommands/go.go`, find lines 41-44:

```go
	if user.Character.Aggro != nil {
		user.SendText("You can't do that! You are in combat!")
		return true, nil
	}
```

Replace with:

```go
	if user.Character.Aggro != nil {
		// Always allow movement out of the death recovery room —
		// stale aggro must never trap a player in the Shadow Realm.
		deathRoom := int(configs.GetSpecialRoomsConfig().DeathRecoveryRoom)
		if user.Character.RoomId != deathRoom {
			user.SendText("You can't do that! You are in combat!")
			return true, nil
		}
		// Force-clear the stale aggro so it doesn't follow them out.
		user.Character.Aggro = nil
	}
```

- [ ] **Step 2: Build and verify**

Run: `go build ./...`
Expected: Clean build, no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/usercommands/go.go
git commit -m "fix: allow movement from death recovery room even with stale aggro

Players stuck in the Shadow Realm with stale combat state can now
always use the portal. Aggro is force-cleared on exit.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 4: Wire onMagic into resolveSpell

**Files:**
- Modify: `internal/hooks/spell_resolution.go:77-81`

- [ ] **Step 1: Add scripting import**

In `internal/hooks/spell_resolution.go`, find the import block (lines 2-20) and add the scripting import. Find:

```go
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/templates"
```

Replace with:

```go
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/templates"
```

- [ ] **Step 2: Call onMagic before component consumption**

In `internal/hooks/spell_resolution.go`, find lines 77-81:

```go
	// --- Consume component if required ---
	if spellData.ComponentTag != "" {
		consumeSpellComponent(user, spellData.ComponentTag)
	}
}
```

Replace with:

```go
	// --- Run spell script onMagic (if present) ---
	spellAggro := characters.SpellAggroInfo{
		SpellId:              cs.SpellId,
		SpellRest:            cs.SpellRest,
		TargetUserIds:        cs.TargetUserIds,
		TargetMobInstanceIds: cs.TargetMobInstanceIds,
	}
	scripting.TrySpellScriptEvent("onMagic", user.UserId, 0, spellAggro)

	// --- Consume component if required ---
	if spellData.ComponentTag != "" {
		consumeSpellComponent(user, spellData.ComponentTag)
	}
}
```

- [ ] **Step 3: Build and verify**

Run: `go build ./...`
Expected: Clean build, no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/spell_resolution.go
git commit -m "feat: call spell script onMagic for player casts

TrySpellScriptEvent('onMagic') was only wired for mob casts.
Player spell resolution now invokes it too, enabling script-based
spells like fold-anchor, chrysalis-aid, and summon spells.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 5: Wire onCast into Cast Command

**Files:**
- Modify: `internal/usercommands/skill.cast.go:242-251`

- [ ] **Step 1: Add scripting import**

In `internal/usercommands/skill.cast.go`, add the scripting import to the import block. Find:

```go
	"github.com/GoMudEngine/GoMud/internal/rooms"
```

Add after it:

```go
	"github.com/GoMudEngine/GoMud/internal/scripting"
```

(If `scripting` is already imported, skip this sub-step.)

- [ ] **Step 2: Call onCast after CastingState creation**

In `internal/usercommands/skill.cast.go`, find lines 242-251:

```go
	}

	// 10. Announce and fire skill-used event
	events.AddToQueue(events.SkillUsed{
		UserId:  user.UserId,
		Skill:   skills.Spellcasting,
		Details: spellInfo.Name,
	})

	user.SendText(`<ansi fg="cyan">` + spells.GetCastMessage("cast_started", spellInfo.Name) + `</ansi>`)
```

Replace with:

```go
	}

	// 10a. Fire onCast spell script (if present) — can cancel the cast
	spellAggro := characters.SpellAggroInfo{
		SpellId:              spellInfo.SpellId,
		SpellRest:            spellRest,
		TargetUserIds:        targetUserIds,
		TargetMobInstanceIds: targetMobInstanceIds,
	}
	if allowContinue, err := scripting.TrySpellScriptEvent("onCast", user.UserId, 0, spellAggro); err == nil && !allowContinue {
		user.Character.CastingState = nil
		return true, nil
	}

	// 10b. Announce and fire skill-used event
	events.AddToQueue(events.SkillUsed{
		UserId:  user.UserId,
		Skill:   skills.Spellcasting,
		Details: spellInfo.Name,
	})

	user.SendText(`<ansi fg="cyan">` + spells.GetCastMessage("cast_started", spellInfo.Name) + `</ansi>`)
```

- [ ] **Step 3: Build and verify**

Run: `go build ./...`
Expected: Clean build, no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/usercommands/skill.cast.go
git commit -m "feat: call spell script onCast when player initiates casting

Allows spell scripts to show cast-initiation flavor text and
optionally cancel the cast by returning false.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 6: Wire onWait into Fold Accumulation

**Files:**
- Modify: `internal/hooks/NewRound_DoCombat_helpers.go:256-289`

- [ ] **Step 1: Add scripting import**

In `internal/hooks/NewRound_DoCombat_helpers.go`, add to the import block. Find:

```go
	"github.com/GoMudEngine/GoMud/internal/rooms"
```

Add after it:

```go
	"github.com/GoMudEngine/GoMud/internal/scripting"
```

(If `scripting` is already imported, skip this sub-step.)

- [ ] **Step 2: Call onWait during fold accumulation**

In `internal/hooks/NewRound_DoCombat_helpers.go`, find the fold accumulation block. After lines 254-255:

```go
	user.Character.Conviction -= roundCost
	cs.ConvictionSpent += roundCost
```

And before line 257:

```go
	// Advance folds — resolve spell if complete
	if advanceFolds(cs) {
```

Insert between them:

```go
	// Run spell script onWait (if present) — flavor text during fold accumulation
	spellAggro := characters.SpellAggroInfo{
		SpellId:              cs.SpellId,
		SpellRest:            cs.SpellRest,
		TargetUserIds:        cs.TargetUserIds,
		TargetMobInstanceIds: cs.TargetMobInstanceIds,
	}
	scripting.TrySpellScriptEvent("onWait", user.UserId, 0, spellAggro)

```

- [ ] **Step 3: Build and verify**

Run: `go build ./...`
Expected: Clean build, no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/NewRound_DoCombat_helpers.go
git commit -m "feat: call spell script onWait during fold accumulation

Allows spell scripts to display flavor text each round while
folds are building up.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 7: Split Fold Anchor into Two Spells — Data Files

**Files:**
- Modify: `_datafiles/world/dogmud/spells/fold-anchor.yaml`
- Modify: `_datafiles/world/dogmud/spells/fold-anchor.js`
- Create: `_datafiles/world/dogmud/spells/fold-recall.yaml`
- Create: `_datafiles/world/dogmud/spells/fold-recall.js`
- Modify: `_datafiles/world/dogmud/templates/help/fold-anchor.template`
- Create: `_datafiles/world/dogmud/templates/help/fold-recall.template`

- [ ] **Step 1: Update fold-anchor.yaml description**

Replace the full contents of `_datafiles/world/dogmud/spells/fold-anchor.yaml` with:

```yaml
spellid: fold-anchor
name: Fold Anchor
description: Sets a Chrysalis anchor at your current location.
type: helpsingle
schools:
  - enhancement
cost: 50
waitrounds: 1
difficulty: 0
primarystat: willpower
base_folds: 6
```

- [ ] **Step 2: Update fold-anchor.js to set-only logic**

Replace the full contents of `_datafiles/world/dogmud/spells/fold-anchor.js` with:

```javascript
// Fold Anchor spell script — sets a Chrysalis anchor at current location

function onCast(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(),
        'You weave a Chrysalis anchor into the fabric of this location.');
    SendRoomMessage(sourceActor.GetRoomId(),
        sourceActor.GetCharacterName(true) +
        ' traces a complex pattern in the air that briefly glows and fades.',
        sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(),
        'The anchor takes shape, binding to the Veil...');
}

function onMagic(sourceActor, targetActor) {
    var currentRoom = sourceActor.GetRoomId();
    sourceActor.SetMiscCharacterData('fold-anchor-room', currentRoom);
    SendUserMessage(sourceActor.UserId(),
        'A Chrysalis anchor locks into place here. ' +
        'Cast <ansi fg="command">fold-recall</ansi> from elsewhere to return.');
    SendRoomMessage(currentRoom,
        'A faint shimmer marks where ' +
        sourceActor.GetCharacterName(true) +
        ' has set an anchor.',
        sourceActor.UserId());
}
```

- [ ] **Step 3: Create fold-recall.yaml**

Create `_datafiles/world/dogmud/spells/fold-recall.yaml`:

```yaml
spellid: fold-recall
name: Fold Recall
description: Folds the Veil and teleports you to your Chrysalis anchor.
type: helpsingle
schools:
  - enhancement
cost: 50
waitrounds: 1
difficulty: 0
primarystat: willpower
base_folds: 6
```

- [ ] **Step 4: Create fold-recall.js**

Create `_datafiles/world/dogmud/spells/fold-recall.js`:

```javascript
// Fold Recall spell script — teleport to your Chrysalis anchor

function onCast(sourceActor, targetActor) {
    var anchorRoom = Number(
        sourceActor.GetMiscCharacterData('fold-anchor-room')) || 0;

    if (anchorRoom <= 0) {
        SendUserMessage(sourceActor.UserId(),
            'You reach for the Veil, but there is no anchor to ' +
            'pull you. Set one first with ' +
            '<ansi fg="command">cast fold-anchor</ansi>.');
        return false;
    }

    var currentRoom = sourceActor.GetRoomId();
    if (anchorRoom == currentRoom) {
        SendUserMessage(sourceActor.UserId(),
            'You are already standing on your anchor.');
        return false;
    }

    SendUserMessage(sourceActor.UserId(),
        'You reach through the Veil toward your anchor point...');
    SendRoomMessage(currentRoom,
        sourceActor.GetCharacterName(true) +
        ' reaches into the Veil, reality blurring around them.',
        sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(),
        'The Veil thins as you pull yourself toward the anchor...');
}

function onMagic(sourceActor, targetActor) {
    var anchorRoom = Number(
        sourceActor.GetMiscCharacterData('fold-anchor-room')) || 0;
    var currentRoom = sourceActor.GetRoomId();

    if (anchorRoom <= 0 || anchorRoom == currentRoom) {
        SendUserMessage(sourceActor.UserId(),
            'The fold collapses — no valid anchor found.');
        return;
    }

    SendRoomMessage(currentRoom,
        sourceActor.GetCharacterName(true) +
        ' folds through the Veil and vanishes!',
        sourceActor.UserId());
    sourceActor.MoveRoom(anchorRoom);
    SendUserMessage(sourceActor.UserId(),
        'You fold through the Veil and arrive at your anchor point!');
    SendRoomMessage(anchorRoom,
        sourceActor.GetCharacterName(true) +
        ' folds through the Veil and appears!',
        sourceActor.UserId());
}
```

- [ ] **Step 5: Update fold-anchor help template**

Replace the full contents of `_datafiles/world/dogmud/templates/help/fold-anchor.template`:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">fold-anchor</ansi> spell

<ansi fg="white-bold">Fold Anchor</ansi> sets a Chrysalis anchor at your current location.
You can only have one anchor at a time. Casting again at a new
location overwrites the previous anchor.

Use <ansi fg="command">cast fold-recall</ansi> from anywhere else to teleport back
to your anchor.

<ansi fg="yellow">Usage: </ansi>

  <ansi fg="command">cast fold-anchor</ansi>     Set anchor at current location.

<ansi fg="yellow">Base Folds:  </ansi> <ansi fg="white-bold">6</ansi>
<ansi fg="yellow">Target:      </ansi> <ansi fg="white-bold">Self</ansi>
<ansi fg="yellow">Conv. Cost:  </ansi> <ansi fg="white-bold">50</ansi>
<ansi fg="yellow">School:      </ansi> <ansi fg="white-bold">Enhancement</ansi>

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help fold-recall</ansi>, <ansi fg="command">help cast</ansi>, <ansi fg="command">help spells</ansi>
```

- [ ] **Step 6: Create fold-recall help template**

Create `_datafiles/world/dogmud/templates/help/fold-recall.template`:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">fold-recall</ansi> spell

<ansi fg="white-bold">Fold Recall</ansi> teleports you back to your Chrysalis anchor.
You must first set an anchor with <ansi fg="command">cast fold-anchor</ansi>.

If you have no anchor set, or you are already standing on it,
the cast will fail and no conviction is spent.

<ansi fg="yellow">Usage: </ansi>

  <ansi fg="command">cast fold-recall</ansi>     Teleport to your anchor.

<ansi fg="yellow">Base Folds:  </ansi> <ansi fg="white-bold">6</ansi>
<ansi fg="yellow">Target:      </ansi> <ansi fg="white-bold">Self</ansi>
<ansi fg="yellow">Conv. Cost:  </ansi> <ansi fg="white-bold">50</ansi>
<ansi fg="yellow">School:      </ansi> <ansi fg="white-bold">Enhancement</ansi>

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help fold-anchor</ansi>, <ansi fg="command">help cast</ansi>, <ansi fg="command">help spells</ansi>
```

- [ ] **Step 7: Commit**

```bash
git add _datafiles/world/dogmud/spells/fold-anchor.yaml \
        _datafiles/world/dogmud/spells/fold-anchor.js \
        _datafiles/world/dogmud/spells/fold-recall.yaml \
        _datafiles/world/dogmud/spells/fold-recall.js \
        _datafiles/world/dogmud/templates/help/fold-anchor.template \
        _datafiles/world/dogmud/templates/help/fold-recall.template
git commit -m "feat: split fold-anchor into separate set and recall spells

fold-anchor now only sets the anchor. fold-recall teleports back.
Each has its own help file. onCast in fold-recall validates that
an anchor exists before spending folds.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 8: Paired Spell Learning + Migration

**Files:**
- Modify: `internal/characters/character.go:808-814`

- [ ] **Step 1: Add paired spell map and update LearnSpell**

In `internal/characters/character.go`, find the `LearnSpell` function at line 808:

```go
func (c *Character) LearnSpell(spellName string) bool {
	if _, ok := c.SpellBook[spellName]; !ok {
		c.SpellBook[spellName] = 1
		return true
	}
	return false
}
```

Replace with:

```go
// pairedSpells maps spells that are always learned together.
// Learning either spell automatically grants the other.
var pairedSpells = map[string]string{
	"fold-anchor": "fold-recall",
	"fold-recall": "fold-anchor",
}

func (c *Character) LearnSpell(spellName string) bool {
	if _, ok := c.SpellBook[spellName]; ok {
		return false
	}
	c.SpellBook[spellName] = 1

	// Grant paired spell if one exists
	if paired, ok := pairedSpells[spellName]; ok {
		if _, known := c.SpellBook[paired]; !known {
			c.SpellBook[paired] = 1
		}
	}

	return true
}
```

- [ ] **Step 2: Add migration function**

In `internal/characters/character.go`, add the migration function directly after `LearnSpell`:

```go
// MigratePairedSpells is a one-time migration that grants missing
// paired spells to existing characters. Call on character load.
// Uses MiscData flag to run only once per character.
func (c *Character) MigratePairedSpells() {
	const migrationKey = "migration-fold-pair-done"
	if c.GetMiscData(migrationKey) != nil {
		return
	}
	for spell, paired := range pairedSpells {
		if _, known := c.SpellBook[spell]; known {
			if _, hasPartner := c.SpellBook[paired]; !hasPartner {
				c.SpellBook[paired] = 1
			}
		}
	}
	c.SetMiscData(migrationKey, "1")
}
```

Note: Verify the exact method names for MiscData access. The scripting layer uses `GetMiscCharacterData`/`SetMiscCharacterData`, but the Go Character struct may use `GetMiscData`/`SetMiscData`. Check `character.go` for the correct names and adjust accordingly.

- [ ] **Step 3: Wire migration into character load path**

In `internal/users/users.go`, in the `LoadUser` function, add the migration call after validation (around line 340, after the `Validate` block):

Find:

```go
	if loadedUser.Joined.IsZero() {
		loadedUser.Joined = time.Now()
	}
```

Add before it:

```go
	// One-time migrations
	loadedUser.Character.MigratePairedSpells()
```

- [ ] **Step 4: Build and verify**

Run: `go build ./...`
Expected: Clean build, no errors.

- [ ] **Step 5: Commit**

```bash
git add internal/characters/character.go internal/users/users.go
git commit -m "feat: paired spell learning and one-time migration

LearnSpell now auto-grants paired spells (fold-anchor <-> fold-recall).
Existing characters get the missing spell on next load via migration.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 9: Add SpellId to QuestReward

**Files:**
- Modify: `internal/quests/quests.go:24-33`
- Modify: `internal/hooks/Quest_HandleQuestUpdate.go:128-151`

- [ ] **Step 1: Add SpellId field to QuestReward struct**

In `internal/quests/quests.go`, find lines 24-33:

```go
type QuestReward struct {
	QuestId       string // new questId to give ( {id}-{step} format )
	Gold          int    // zero or more gold to give.
	ItemId        int    // itemId to give
	BuffId        int    // buffId to apply
	SkillInfo     string // skill to give, format: skillId:skillLevel such as "map:1"
	PlayerMessage string // string to display to player
	RoomMessage   string // string to display to room
	RoomId        int    // roomId to move player to
}
```

Replace with:

```go
type QuestReward struct {
	QuestId       string // new questId to give ( {id}-{step} format )
	Gold          int    // zero or more gold to give.
	ItemId        int    // itemId to give
	BuffId        int    // buffId to apply
	SkillInfo     string // skill to give, format: skillId:skillLevel such as "map:1"
	SpellId       string // spell to teach on quest completion
	PlayerMessage string // string to display to player
	RoomMessage   string // string to display to room
	RoomId        int    // roomId to move player to
}
```

- [ ] **Step 2: Add spell reward handling to quest update hook**

In `internal/hooks/Quest_HandleQuestUpdate.go`, add the spells import. Find:

```go
	"github.com/GoMudEngine/GoMud/internal/rooms"
```

Add after it:

```go
	"github.com/GoMudEngine/GoMud/internal/spells"
```

Then find the skill reward block ending around line 151:

```go
			}
		}
		// Move them to another room/area?
```

Add the spell reward block between them:

```go
			}
		}
		// Spell reward?
		if questInfo.Rewards.SpellId != "" {
			if questUser.Character.LearnSpell(questInfo.Rewards.SpellId) {
				if spellData := spells.GetSpell(questInfo.Rewards.SpellId); spellData != nil {
					questUser.SendText(fmt.Sprintf(
						`<ansi fg="magenta-bold">You have learned the spell: <ansi fg="cyan-bold">%s</ansi></ansi>`,
						spellData.Name))
				}
			}
		}
		// Move them to another room/area?
```

- [ ] **Step 3: Add spell reward to quest 12**

In `_datafiles/world/dogmud/quests/12-the_wardens_covenant.yaml`, find:

```yaml
rewards:
  playermessage: "Sylara's eyes shine with quiet pride. The steppe
    wind swirls around you like a greeting, and you feel a new
    connection to the land -- ancient, vast, and watchful. She
    presses a bundle of spirit fetishes into your hands."
  gold: 15
  itemid: 40031
```

Replace with:

```yaml
rewards:
  playermessage: "Sylara's eyes shine with quiet pride. The steppe
    wind swirls around you like a greeting, and you feel a new
    connection to the land -- ancient, vast, and watchful. She
    presses a bundle of spirit fetishes into your hands."
  gold: 15
  itemid: 40031
  spellid: summon-steppe-spirit
```

- [ ] **Step 4: Build and verify**

Run: `go build ./...`
Expected: Clean build, no errors.

- [ ] **Step 5: Commit**

```bash
git add internal/quests/quests.go \
        internal/hooks/Quest_HandleQuestUpdate.go \
        _datafiles/world/dogmud/quests/12-the_wardens_covenant.yaml
git commit -m "feat: add SpellId field to QuestReward for spell quest rewards

Quests can now teach spells on completion. The Warden's Covenant
(quest 12) now grants summon-steppe-spirit.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 10: Fetish Inventory Gating

**Files:**
- Modify: `_datafiles/world/dogmud/mobs/ironwind_steppe/scripts/241-windwarden_sylara.js:60-77`

- [ ] **Step 1: Add inventory check to fetish dispensing**

In `_datafiles/world/dogmud/mobs/ironwind_steppe/scripts/241-windwarden_sylara.js`, find lines 60-77 (the bonus and subsequent-ask blocks):

```javascript
    // First time after quest: give 4 bonus fetishes (quest reward gave 1)
    var bonusGiven = user.GetMiscCharacterData(BONUS_KEY);
    if (!bonusGiven || bonusGiven === '' || bonusGiven === '0') {
        user.GiveItem(FETISH_ITEM_ID);
        user.GiveItem(FETISH_ITEM_ID);
        user.GiveItem(FETISH_ITEM_ID);
        user.GiveItem(FETISH_ITEM_ID);
        user.SetMiscCharacterData(BONUS_KEY, '1');
        mob.Command('emote reaches into a pouch and produces several small bundles of grass and wolf fur.');
        mob.Command('say Take these. The spirits provided well this season. You will need them for the calling.', 2.0);
        return true;
    }

    // Subsequent asks: give 1 fetish
    user.GiveItem(FETISH_ITEM_ID);
    mob.Command('emote pulls a spirit fetish from her pouch and holds it out.');
    mob.Command('say For the calling. The steppe provides.', 2.0);
    return true;
```

Replace with:

```javascript
    // First time after quest: give 4 bonus fetishes (quest reward gave 1)
    var bonusGiven = user.GetMiscCharacterData(BONUS_KEY);
    if (!bonusGiven || bonusGiven === '' || bonusGiven === '0') {
        user.GiveItem(FETISH_ITEM_ID);
        user.GiveItem(FETISH_ITEM_ID);
        user.GiveItem(FETISH_ITEM_ID);
        user.GiveItem(FETISH_ITEM_ID);
        user.SetMiscCharacterData(BONUS_KEY, '1');
        mob.Command('emote reaches into a pouch and produces several small bundles of grass and wolf fur.');
        mob.Command('say Take these. The spirits provided well this season. You will need them for the calling.', 2.0);
        return true;
    }

    // Already carrying a fetish? Don't give another.
    if (user.HasItemId(FETISH_ITEM_ID)) {
        mob.Command('say You already carry a spirit fetish. Use it wisely.');
        return true;
    }

    // Subsequent asks: give 1 fetish
    user.GiveItem(FETISH_ITEM_ID);
    mob.Command('emote pulls a spirit fetish from her pouch and holds it out.');
    mob.Command('say For the calling. The steppe provides.', 2.0);
    return true;
```

Note: `HasItemId()` checks both backpack and equipped items by default (see `actor_func.go:530-544`), so this covers the worn-items case too.

- [ ] **Step 2: Commit**

```bash
git add _datafiles/world/dogmud/mobs/ironwind_steppe/scripts/241-windwarden_sylara.js
git commit -m "fix: gate fetish dispensing on inventory check

Sylara no longer gives unlimited spirit fetishes. If the player
already has one (backpack or equipped), she refuses with a message.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 11: Smoke Test

- [ ] **Step 1: Full build**

Run: `go build ./...`
Expected: Clean build, no errors.

- [ ] **Step 2: Run tests**

Run: `go test ./...`
Expected: All tests pass.

- [ ] **Step 3: Manual verification checklist**

Start the server and verify:
1. **Death loop**: Kill a character, confirm they arrive in Shadow Realm with no aggro, can use portal
2. **Fold Anchor**: Cast `fold-anchor`, confirm anchor is set. Move to another room, cast `fold-recall`, confirm teleport works
3. **Fold Recall with no anchor**: Cast `fold-recall` with no anchor set, confirm it fails gracefully on cast initiation (no conviction spent)
4. **Quest 12 spell reward**: Complete The Warden's Covenant, confirm `summon-steppe-spirit` is learned
5. **Fetish gating**: With a fetish in inventory, ask Sylara for another — confirm refusal. Drop the fetish, ask again — confirm she gives one
6. **Paired spells**: Learn fold-anchor via discovery — confirm fold-recall appears in spellbook too
