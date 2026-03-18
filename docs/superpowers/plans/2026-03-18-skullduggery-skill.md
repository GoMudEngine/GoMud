# Skullduggery Skill Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development
> (if subagents available) or superpowers:executing-plans to implement this plan.
> Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Consolidate stealth/pickpocket/backstab into a single skullduggery
skill with 7 sub-commands: sneak, steal, plant, picklock, shadow, surprise
attack, defuse.

**Architecture:** Rename existing stealth skill to skullduggery, add config
knobs, rework sneak with opposed rolls, replace backstab with surprise attack
(extra hit from stealth), add steal/plant/shadow/defuse commands, add stealth
detection to room movement, gate picklock behind the new skill.

**Tech Stack:** Go, Go templates, JS (goja scripting), YAML data files

**Spec:** `docs/superpowers/specs/2026-03-18-skullduggery-skill-design.md`

---

## Chunk 1: Foundation (Skill Rename + Config + Aggro Rename)

### Task 1: Add Skullduggery Config Knobs

**Files:**
- Modify: `internal/configs/config.balance.go`

- [ ] **Step 1: Add skullduggery config section**

After the `// -- COMBAT: SPECIAL MOVES` section (around line 44), add:

```go
	// ── SKULLDUGGERY ─────────────────────────────────────────────────────────
	SneakFailCooldown              ConfigInt   `yaml:"SneakFailCooldown"`              // Rounds before sneak retry after failure (default 3)
	SurpriseAttackOffhandPenalty   ConfigFloat `yaml:"SurpriseAttackOffhandPenalty"`   // Hit penalty for offhand surprise attack (default 0.10)
	SurpriseAttackExtraArm1Penalty ConfigFloat `yaml:"SurpriseAttackExtraArm1Penalty"` // Hit penalty for extra arm 1 (default 0.25)
	SurpriseAttackExtraArm2Penalty ConfigFloat `yaml:"SurpriseAttackExtraArm2Penalty"` // Hit penalty for extra arm 2 (default 0.40)
	StealSkillMultiplier           ConfigFloat `yaml:"StealSkillMultiplier"`           // Tuning knob for steal/plant rolls (default 1.0)
	StealHiddenBonus               ConfigInt   `yaml:"StealHiddenBonus"`               // Bonus to attacker score when hidden (default 25)
	StealCooldown                  ConfigInt   `yaml:"StealCooldown"`                  // Steal/plant cooldown in real seconds (default 60)
	ShadowCooldown                 ConfigInt   `yaml:"ShadowCooldown"`                 // Rounds before re-shadowing (default 5)
```

- [ ] **Step 2: Add defaults in the init/default function**

Find where other defaults are set (search for `SpecialMoveCooldown` default)
and add defaults for all 8 new knobs matching the values above.

- [ ] **Step 3: Verify build**

Run: `go build ./...`

- [ ] **Step 4: Commit**

```bash
git add internal/configs/config.balance.go
git commit -m "feat: add skullduggery config knobs to balance config"
```

### Task 2: Rename Stealth Skill to Skullduggery

**Files:**
- Modify: `internal/skills/skills.go`

- [ ] **Step 1: Rename the skill tag**

Change line 39 from:
```go
Stealth   SkillTag = `stealth`    // Sneaking, hiding, avoiding detection
```
To:
```go
Skullduggery SkillTag = `skullduggery` // Sneaking, stealing, lockpicking, surprise attacks
```

- [ ] **Step 2: Update all references to Stealth in the file**

Replace all occurrences of `Stealth` with `Skullduggery` in:
- The rogue profession mapping (around line 70)
- `SkillPrimaryStats` map (stealth -> skullduggery, still dexterity)
- `SkillProgressionMultipliers` map

- [ ] **Step 3: Fix all compiler errors across the codebase**

Run `go build ./...` and fix every reference to `skills.Stealth` -> `skills.Skullduggery`.
Files that will need updating:
- `internal/usercommands/skill.stealth.go`
- `internal/usercommands/skill.stealth.pickpocket.go`
- `internal/mobcommands/sneak.go`
- Any other files referencing `skills.Stealth`

For now, update the references in existing files. The old files will be
replaced in later tasks but need to compile in the meantime.

- [ ] **Step 4: Verify build**

Run: `go build ./...`

- [ ] **Step 5: Run tests**

Run: `go test ./...`
Fix any test failures related to the rename.

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "refactor: rename stealth skill to skullduggery

Renames the skill tag, updates all references across the codebase.
Rogue profession now maps to {Skullduggery, WeaponCombat}."
```

### Task 3: Rename BackStab Aggro Type to SurpriseAttack

**Files:**
- Modify: `internal/characters/aggro.go`
- Modify: all files referencing `characters.BackStab`

- [ ] **Step 1: Rename the constant**

In `aggro.go`, change `BackStab` to `SurpriseAttack` in the iota enum.

- [ ] **Step 2: Fix all references**

Run `go build ./...` and fix every reference. Key files:
- `internal/combat/combat.go` (lines 313-317)
- `internal/combat/combat_helpers.go` (anywhere referencing BackStab)
- `internal/mobcommands/backstab.go`
- Any test files

- [ ] **Step 3: Update combat message prefix**

In `combat.go`, change the backstab message prefix from:
```go
attackMessagePrefix = `<ansi fg="magenta-bold">*[BACKSTAB]*</ansi> `
```
To:
```go
attackMessagePrefix = `<ansi fg="magenta-bold">*[SURPRISE ATTACK]*</ansi> `
```

- [ ] **Step 4: Verify build + tests**

Run: `go build ./...` then `go test ./...`

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "refactor: rename BackStab aggro type to SurpriseAttack

Renames the constant and updates all references. Combat message
prefix updated to *[SURPRISE ATTACK]*."
```

---

## Chunk 2: Sneak Rework

### Task 4: Create Sneak Command with Opposed Rolls

**Files:**
- Create: `internal/usercommands/skill.skullduggery.sneak.go`
- Delete: `internal/usercommands/skill.stealth.go`
- Modify: `internal/usercommands/usercommands.go` (update registration)

- [ ] **Step 1: Create the new sneak command file**

Write `internal/usercommands/skill.skullduggery.sneak.go`:

```go
package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Sneak(rest string, user *users.UserRecord,
	room *rooms.Room, flags events.EventFlag) (bool, error) {

	skillLevel := user.Character.GetSkillLevel(skills.Skullduggery)
	if skillLevel < 0 {
		return false, nil
	}

	if user.Character.HasBuffFlag(buffs.Hidden) {
		user.SendText("You're already hidden!")
		return true, nil
	}

	if user.Character.Aggro != nil {
		user.SendText("You can't do that while in combat!")
		return true, nil
	}

	cfg := configs.GetBalanceConfig()
	cooldownRounds := cfg.SneakFailCooldown.Get(3)
	if !user.Character.TryCooldown("skullduggery:sneak",
		fmt.Sprintf("%d rounds", cooldownRounds)) {
		user.SendText(fmt.Sprintf(
			"You need to wait %d more rounds before trying to hide again.",
			user.Character.GetCooldown("skullduggery:sneak")))
		return true, nil
	}

	// Compute sneak score
	sneakScore := float64(user.Character.Stats.Dexterity.ValueAdj) +
		combat.SkillMultiplier(skillLevel)*25.0

	// Get all hostile/neutral observers (exclude party members)
	var partyMemberIds map[int]bool
	if p := parties.Get(user.UserId); p != nil {
		partyMemberIds = make(map[int]bool)
		for _, uid := range p.GetMembers() {
			partyMemberIds[uid] = true
		}
	}

	spotted := false
	spotterName := ""

	// Check against players in room
	for _, pId := range room.GetPlayers() {
		if pId == user.UserId {
			continue
		}
		if partyMemberIds != nil && partyMemberIds[pId] {
			continue
		}
		p := users.GetByUserId(pId)
		if p == nil {
			continue
		}
		observerScore := float64(p.Character.Stats.Perception.ValueAdj) +
			combat.SkillMultiplier(
				p.Character.GetSkillLevel(skills.Search))*25.0
		success, _, _, _ := dice.OpposedRollStat(sneakScore, observerScore)
		if !success {
			spotted = true
			spotterName = p.Character.Name
			p.SendText(fmt.Sprintf(
				`<ansi fg="username">%s</ansi> tries to hide but you notice them.`,
				user.Character.Name))
			break
		}
	}

	// Check against mobs in room (if not already spotted)
	if !spotted {
		for _, mId := range room.GetMobs() {
			mob := mobs.GetInstance(mId)
			if mob == nil {
				continue
			}
			observerScore := float64(
				mob.Character.Stats.Perception.ValueAdj) +
				combat.SkillMultiplier(
					mob.Character.GetSkillLevel(skills.Search))*25.0
			success, _, _, _ := dice.OpposedRollStat(
				sneakScore, observerScore)
			if !success {
				spotted = true
				spotterName = mob.Character.Name
				break
			}
		}
	}

	// Skill progression on any roll
	if len(room.GetPlayers()) > 1 || len(room.GetMobs()) > 0 {
		user.Character.CheckSkillProgression(
			string(skills.Skullduggery), user.UserId, 1.0)
	}

	if spotted {
		user.SendText(fmt.Sprintf(
			"You try to blend into the shadows but %s notices you.",
			spotterName))
		return true, nil
	}

	// Success — on failure the cooldown was already consumed above,
	// but on success we clear it so there's no cooldown.
	user.Character.CancelCooldown("skullduggery:sneak")
	user.AddBuff(9, `skill`)
	user.SendText("You slip into the shadows.")

	events.AddToQueue(events.SkillUsed{
		UserId: user.UserId,
		Skill:  skills.Skullduggery,
		Details: `sneak`,
	})

	return true, nil
}
```

**Note:** The `CancelCooldown` call on success is important — the cooldown
was consumed by `TryCooldown` at the top (to prevent spam), but on success
we clear it so the player isn't penalized. Verify that `CancelCooldown`
exists on the character — if not, simply move the `TryCooldown` check
to only apply on failure (set it after the spotted check).

- [ ] **Step 2: Delete old stealth file**

```bash
rm internal/usercommands/skill.stealth.go
```

- [ ] **Step 3: Update command registration**

In `usercommands.go`, find the `sneak` registration and ensure it points
to the new `Sneak` function. The function name hasn't changed, so this
may just work. Verify the old file's function was also named `Sneak`.

- [ ] **Step 4: Verify build + tests**

Run: `go build ./...` then `go test ./...`

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat: rework sneak command with opposed rolls

Sneak now rolls against each hostile/neutral observer in the room.
Empty rooms auto-succeed. Party members excluded from observer list.
Failure triggers a configurable cooldown."
```

### Task 5: Add Stealth Detection on Room Entry

**Files:**
- Modify: `internal/usercommands/go.go`

This task adds two detection checks to the movement handler:

1. When a hidden player enters a room — observers roll to spot them
2. When any player enters a room — they roll to spot hidden occupants

- [ ] **Step 1: Add helper function for stealth detection**

Create a helper function (can be in `go.go` or a new file
`internal/usercommands/stealth_detection.go`) that performs the
opposed roll check:

```go
// checkSteathDetection rolls each observer against the hidden player.
// Returns (spotted bool, spotterName string).
func checkStealthDetection(
	hiddenUser *users.UserRecord,
	room *rooms.Room,
	excludeUserIds map[int]bool,
) (bool, string) {
	sneakScore := float64(
		hiddenUser.Character.Stats.Dexterity.ValueAdj) +
		combat.SkillMultiplier(
			hiddenUser.Character.GetSkillLevel(
				skills.Skullduggery))*25.0

	// Check players
	for _, pId := range room.GetPlayers() {
		if pId == hiddenUser.UserId {
			continue
		}
		if excludeUserIds != nil && excludeUserIds[pId] {
			continue
		}
		p := users.GetByUserId(pId)
		if p == nil {
			continue
		}
		observerScore := float64(
			p.Character.Stats.Perception.ValueAdj) +
			combat.SkillMultiplier(
				p.Character.GetSkillLevel(skills.Search))*25.0
		success, _, _, _ := dice.OpposedRollStat(
			sneakScore, observerScore)
		if !success {
			p.SendText(fmt.Sprintf(
				`<ansi fg="username">%s</ansi> slips into the room but you notice them.`,
				hiddenUser.Character.Name))
			return true, p.Character.Name
		}
	}

	// Check mobs
	for _, mId := range room.GetMobs() {
		mob := mobs.GetInstance(mId)
		if mob == nil {
			continue
		}
		observerScore := float64(
			mob.Character.Stats.Perception.ValueAdj) +
			combat.SkillMultiplier(
				mob.Character.GetSkillLevel(skills.Search))*25.0
		success, _, _, _ := dice.OpposedRollStat(
			sneakScore, observerScore)
		if !success {
			return true, mob.Character.Name
		}
	}

	return false, ""
}
```

- [ ] **Step 2: Add detection check for hidden player entering a room**

In `go.go`, after the player has been moved to the destination room
but before mob aggro checks, add:

```go
if isSneaking {
	// Build party exclusion set
	var partyIds map[int]bool
	if p := parties.Get(user.UserId); p != nil {
		partyIds = make(map[int]bool)
		for _, uid := range p.GetMembers() {
			partyIds[uid] = true
		}
	}
	spotted, spotterName := checkStealthDetection(
		user, destRoom, partyIds)
	if spotted {
		user.Character.CancelBuffsWithFlag(buffs.Hidden)
		isSneaking = false
		user.SendText(fmt.Sprintf(
			"You slip into the room but %s notices you.",
			spotterName))
		// Normal arrival broadcast happens below
	}
}
```

- [ ] **Step 3: Add detection check for newcomer spotting hidden occupants**

After the player arrives in the room (whether hidden or not), add a
check where the arriving player rolls against each hidden player in
the room:

```go
// Newcomer tries to spot hidden occupants
if !isSneaking { // Only non-hidden arrivals spot people
	for _, pId := range destRoom.GetPlayers() {
		if pId == user.UserId {
			continue
		}
		hiddenP := users.GetByUserId(pId)
		if hiddenP == nil ||
			!hiddenP.Character.HasBuffFlag(buffs.Hidden) {
			continue
		}
		// Arriving player rolls to spot hidden occupant
		observerScore := float64(
			user.Character.Stats.Perception.ValueAdj) +
			combat.SkillMultiplier(
				user.Character.GetSkillLevel(skills.Search))*25.0
		hiddenScore := float64(
			hiddenP.Character.Stats.Dexterity.ValueAdj) +
			combat.SkillMultiplier(
				hiddenP.Character.GetSkillLevel(
					skills.Skullduggery))*25.0
		success, _, _, _ := dice.OpposedRollStat(
			observerScore, hiddenScore)
		if success {
			hiddenP.Character.CancelBuffsWithFlag(buffs.Hidden)
			hiddenP.SendText(fmt.Sprintf(
				"%s enters the room and notices you!",
				user.Character.Name))
			user.SendText(fmt.Sprintf(
				"You notice <ansi fg=\"username\">%s</ansi> lurking in the shadows.",
				hiddenP.Character.Name))
		}
	}
}
```

- [ ] **Step 4: Verify build + test manually**

Run: `go build ./...`

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat: add stealth detection on room entry

Hidden players are rolled against when entering rooms. Newcomers
also roll to spot hidden occupants. Party members excluded."
```

---

## Chunk 3: Surprise Attack

### Task 6: Implement Surprise Attack in attack.go

**Files:**
- Modify: `internal/usercommands/attack.go`
- Modify: `internal/combat/combat.go` (or `combat_helpers.go`)

The surprise attack fires as an extra hit BEFORE normal combat begins,
while the hidden buff is still active.

- [ ] **Step 1: Add surprise attack check in attack.go**

In `attack.go`, after the target is resolved but BEFORE `SetAggro` is
called (around line 220 for mobs, line 276 for players), add:

```go
// Check for surprise attack from stealth
if user.Character.HasBuffFlag(buffs.Hidden) {
	cfg := configs.GetBalanceConfig()
	if user.Character.TryCooldown("special-move",
		fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown.Get(5))) {

		executeSurpriseAttack(user, room, attackMobInstanceId, 0)
	}
}
```

For player targets, use `executeSurpriseAttack(user, room, 0, attackPlayerId)`.

- [ ] **Step 2: Create executeSurpriseAttack function**

This can go in a new file `internal/usercommands/surprise_attack.go`
or at the bottom of `attack.go`. It should:

1. Get all equipped weapons (same enumeration as combat round)
2. For each weapon, compute damage:
   - Use the physical damage pipeline: `combat.CalcRawDamage(stat, skillRank, itemMult, combat.Physical)`
   - Apply surprise multiplier: `max(1.0, (dex + skillRank) / 100.0)`
   - Apply hit penalty per weapon index (from config)
   - Roll to hit (with penalty applied). On miss, skip damage.
   - On hit: treat as crit (bypass mitigation or apply crit logic)
3. Send messages: `*[SURPRISE ATTACK]* You strike <target> from the shadows!`
4. Consume `special-move` cooldown (already done in step 1)
5. Do NOT cancel hidden buff here — let normal combat initiation handle it

**Important:** The exact damage calculation should follow the existing
`calcHitDamage` pattern in `combat_helpers.go`. Read that code carefully
before implementing. The key difference is the surprise multiplier and
per-weapon hit penalties.

- [ ] **Step 3: Add party auto-assist surprise attacks**

In the existing auto-assist block in `attack.go` (lines 205-218 for
mobs), before calling `partyUser.Command("attack #...")`, check if
the party member is hidden and has special-move cooldown available.
If so, execute their surprise attack first, then let auto-assist
proceed normally.

```go
// Party surprise attacks
if partyUser.Character.HasBuffFlag(buffs.Hidden) {
	if partyUser.Character.TryCooldown("special-move",
		fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown.Get(5))) {
		executeSurpriseAttack(partyUser, room,
			attackMobInstanceId, 0)
	}
}
```

- [ ] **Step 4: Update mob backstab command**

Rename `internal/mobcommands/backstab.go` to use the new aggro type
and message. The mob version can stay simpler (uses the existing
combat loop crit path via `SurpriseAttack` aggro type) since mobs
don't need the extra-hit-on-top approach.

- [ ] **Step 5: Verify build + tests**

Run: `go build ./...` then `go test ./...`

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "feat: implement surprise attack from stealth

Extra crit hit when attacking from hidden state. Swings all weapons
with stacking hit penalties. Party auto-assist triggers surprise
attacks for hidden party members."
```

---

## Chunk 4: Steal & Plant

### Task 7: Create Steal Command

**Files:**
- Create: `internal/usercommands/skill.skullduggery.steal.go`
- Delete: `internal/usercommands/skill.stealth.pickpocket.go`
- Modify: `internal/usercommands/usercommands.go`

- [ ] **Step 1: Create steal command**

Write `internal/usercommands/skill.skullduggery.steal.go`. Key logic:

1. Require skullduggery rank 1+
2. Not in combat check
3. Cooldown check: `TryCooldown("skullduggery:steal", StealCooldown)`
4. Find target by name (mob only — reject players)
5. Compute attacker score:
   `(Dex + SkillMultiplier(rank) * 25) * StealSkillMultiplier`
   If hidden: `+ StealHiddenBonus`
6. Opposed roll vs target Perception
7. Success: steal gold/items (reuse pickpocket loot logic)
8. Failure: cancel hidden buff, target attacks
9. Trigger skill progression

Also handle container steal path:
- Find container by name
- If empty room: auto-success
- If observers: opposed roll vs highest Perception
- Success: take item from container
- Failure: cancel hidden, observers notice

- [ ] **Step 2: Delete old pickpocket file**

```bash
rm internal/usercommands/skill.stealth.pickpocket.go
```

- [ ] **Step 3: Update command registration**

In `usercommands.go`:
- Remove `pickpocket` registration
- Add `steal` registration pointing to new `Steal` function

- [ ] **Step 4: Verify build + tests**

Run: `go build ./...` then `go test ./...`
Fix any test references to the old Pickpocket function.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat: add steal command replacing pickpocket

Roll-based stealing from NPCs and containers. Uses opposed
Dex+skill vs Perception. Hidden bonus applies. No PvP stealing."
```

### Task 8: Create Plant Command

**Files:**
- Create: `internal/usercommands/skill.skullduggery.plant.go`
- Modify: `internal/usercommands/usercommands.go`

- [ ] **Step 1: Create plant command**

Write `internal/usercommands/skill.skullduggery.plant.go`. Mirrors
steal but in reverse — moves item from backpack to target/container.

Same roll formula, same config knobs, shares steal cooldown.

Key differences from steal:
- Syntax: `plant <item> <target>` — need to parse both item and target
- Item must be in player's backpack
- On success: item transfers to target inventory or container
- Error messages: "Plant what?", "Plant on whom?", "You don't have that."

- [ ] **Step 2: Register command**

Add `plant` to `usercommands.go`.

- [ ] **Step 3: Verify build**

Run: `go build ./...`

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "feat: add plant command for slipping items onto NPCs

Mirror of steal — places backpack items into NPC inventory or
containers unnoticed. Same roll formula and cooldown."
```

---

## Chunk 5: Shadow + Picklock Gate + Defuse Stub

### Task 9: Create Shadow Command

**Files:**
- Create: `internal/usercommands/skill.skullduggery.shadow.go`
- Modify: `internal/usercommands/go.go` (shadow follow on move)
- Modify: `internal/usercommands/usercommands.go`

- [ ] **Step 1: Create shadow command**

Write `internal/usercommands/skill.skullduggery.shadow.go`:

1. Require skullduggery rank 2+, must be hidden, not in combat
2. Find target by name in room
3. Store shadow state on character via `SetMiscData("shadow-target", targetName)`
   (similar pattern to tracking)
4. Apply a shadow buff or just use misc data to track state
5. Message: "You begin shadowing \<target\>."

- [ ] **Step 2: Add shadow follow logic to go.go**

In `go.go`, in the party follow section (or after it), add shadow
follow logic:

When any player/mob moves, check if anyone in the room has them as
a shadow target. If so, the shadower auto-follows:

```go
// Shadow follow — check if anyone is shadowing the mover
for _, pId := range room.GetPlayers() {
	if pId == user.UserId {
		continue
	}
	shadowP := users.GetByUserId(pId)
	if shadowP == nil {
		continue
	}
	shadowTarget, _ := shadowP.Character.GetMiscData(
		"shadow-target").(string)
	if shadowTarget == user.Character.Name {
		// Auto-follow + detection checks
		// ... (move shadower, run room-entry detection,
		//      run target-specific detection)
	}
}
```

The detection checks use the same `checkStealthDetection` helper from
Task 5, plus a target-specific roll.

- [ ] **Step 3: Handle shadow stop and shadow end conditions**

Add `shadow stop` handling in the shadow command.
In the movement/follow code, end shadow if:
- Room detection fails (spotted)
- Target enters inaccessible room
- Target dies (check in combat death hooks)
- Target logs off

Apply `ShadowCooldown` on any end condition.

- [ ] **Step 4: Register command + verify build**

Add `shadow` to `usercommands.go`.
Run: `go build ./...`

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat: add shadow command for tailing targets between rooms

Automatically follows target while hidden. Room-entry detection and
target-specific detection on each transition. Configurable cooldown."
```

### Task 10: Gate Picklock Behind Skullduggery + Add Defuse Stub

**Files:**
- Modify: `internal/usercommands/picklock.go`
- Create: `internal/usercommands/skill.skullduggery.defuse.go`
- Modify: `internal/usercommands/usercommands.go`

- [ ] **Step 1: Add skill gate to picklock**

At the top of the `Picklock` function in `picklock.go`, add:

```go
skillLevel := user.Character.GetSkillLevel(skills.Skullduggery)
if skillLevel < 1 {
	return false, nil
}
```

- [ ] **Step 2: Add skill progression to picklock**

After the "Successfully picked the lock!" message blocks (there are
two — one for keyring match and one for fresh solve), add:

```go
user.Character.CheckSkillProgression(
	string(skills.Skullduggery), user.UserId, 1.0)
```

- [ ] **Step 3: Create defuse stub**

Write `internal/usercommands/skill.skullduggery.defuse.go`:

```go
package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Defuse(rest string, user *users.UserRecord,
	room *rooms.Room, flags events.EventFlag) (bool, error) {

	skillLevel := user.Character.GetSkillLevel(skills.Skullduggery)
	if skillLevel < 3 {
		return false, nil
	}

	if rest == "" {
		user.SendText("Defuse what?")
		return true, nil
	}

	user.SendText("You don't detect any traps here.")
	return true, nil
}
```

- [ ] **Step 4: Register defuse command + verify build**

Add `defuse` to `usercommands.go`.
Run: `go build ./...`

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat: gate picklock behind skullduggery, add defuse stub

Picklock now requires skullduggery rank 1+ and triggers skill
progression. Defuse stub added for future trap system (rank 3+)."
```

---

## Chunk 6: Help Files, Hints, Cleanup, Final Verification

### Task 11: Create Help Files

**Files:**
- Create: `_datafiles/world/dogmud/templates/help/skullduggery.template`
- Create: `_datafiles/world/dogmud/templates/help/sneak.template`
- Create: `_datafiles/world/dogmud/templates/help/steal.template`
- Create: `_datafiles/world/dogmud/templates/help/plant.template`
- Create: `_datafiles/world/dogmud/templates/help/shadow.template`
- Create: `_datafiles/world/dogmud/templates/help/defuse.template`
- Modify: `_datafiles/world/dogmud/templates/help/picklock.template` (if exists)
- Delete: any `stealth.template` help files

- [ ] **Step 1: Create skullduggery overview help**

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="skill">skullduggery</ansi> (skill)

The <ansi fg="skill">skullduggery</ansi> skill governs all manner
of underhanded arts: hiding, stealing, tailing, lockpicking, and
striking from the shadows.

<ansi fg="yellow">Sub-commands:</ansi>

  <ansi fg="command">sneak</ansi>       - Slip into the shadows
  <ansi fg="command">steal</ansi>       - Take from NPCs or containers
  <ansi fg="command">plant</ansi>       - Slip items onto NPCs or containers
  <ansi fg="command">picklock</ansi>    - Pick locks on doors and containers
  <ansi fg="command">shadow</ansi>      - Tail someone between rooms unseen
  <ansi fg="command">defuse</ansi>      - Disable traps (coming soon)

Attacking from stealth triggers a
<ansi fg="magenta-bold">surprise attack</ansi> automatically if your
special move cooldown is ready.

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help sneak</ansi>, <ansi fg="command">help steal</ansi>, <ansi fg="command">help shadow</ansi>
```

- [ ] **Step 2: Create individual help files**

Create help templates for sneak, steal, plant, shadow, defuse
following the same format pattern. Keep each under 80 chars wide.

- [ ] **Step 3: Update picklock help**

If a picklock help file exists, add a note:
"Requires <ansi fg="skill">skullduggery</ansi> rank 1."

- [ ] **Step 4: Delete old stealth help files**

```bash
rm _datafiles/world/*/templates/help/stealth.template 2>/dev/null
```

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "docs: add help files for skullduggery and sub-commands"
```

### Task 12: Update Hints and MOTD

**Files:**
- Modify: `_datafiles/world/dogmud/hints.yaml`
- Modify: `_datafiles/config.yaml` (MOTD)

- [ ] **Step 1: Add skullduggery tips to hints.yaml**

Add the 5 tips from the spec. Remove or update any tips referencing
the old `stealth` skill name.

- [ ] **Step 2: Update MOTD**

Update the MOTD in `config.yaml` to mention the skullduggery skill.

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/hints.yaml _datafiles/config.yaml
git commit -m "docs: add skullduggery broadcast tips, update MOTD"
```

### Task 13: Update Patch Notes + Final Verification

**Files:**
- Modify: `PATCH_NOTES.md`

- [ ] **Step 1: Add patch notes**

Add a dated section covering the skullduggery consolidation.

- [ ] **Step 2: Full build**

Run: `go build ./...`

- [ ] **Step 3: Full test suite**

Run: `go test ./...`
Fix any remaining failures.

- [ ] **Step 4: Commit**

```bash
git add PATCH_NOTES.md
git commit -m "docs: add skullduggery patch notes"
```
