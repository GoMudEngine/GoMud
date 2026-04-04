# Mob/Player Progression Parity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate all progression asymmetries between mobs and players, unify howl/taunt, remove deprecated commands, and rename alchemy.

**Architecture:** Expand the Actor interface with 4 progression methods (OnSkillUse, OnStatUse, OnCriticalSuccess, OnCriticalFailure). Move skill progression calls from user/mob wrappers into the shared ExecuteX functions. Create three new shared actions (taunt, bite, hamstring). Add mob progression to the combat loop's basic attack functions.

**Tech Stack:** Go, YAML data files, JS mob scripts

**Spec:** `docs/superpowers/specs/2026-04-04-mob-player-parity-design.md`

---

### Task 1: Expand Actor Interface with Progression Methods

**Files:**
- Modify: `internal/actions/actor.go:12-47`
- Modify: `internal/actions/actor_user.go`
- Modify: `internal/actions/actor_mob.go`

- [ ] **Step 1: Add progression methods to Actor interface**

In `internal/actions/actor.go`, add these four methods inside the `Actor` interface (after the `AddBuff` method at line 46):

```go
	// OnSkillUse triggers skill progression (and the skill's governing stat).
	// UserActor calls Character.OnSkillUse(skill, userId) which internally
	// fires events.SkillUsed for quest tracking when userId > 0.
	// MobActor calls Character.OnSkillUse(skill, 0).
	OnSkillUse(skillName string) bool

	// OnStatUse triggers stat progression.
	OnStatUse(statName string) bool

	// OnCriticalSuccess records a critical hit for progression bonuses.
	OnCriticalSuccess(skillName string)

	// OnCriticalFailure records a fumble for progression tracking.
	OnCriticalFailure(skillName string)
```

- [ ] **Step 2: Implement on UserActor**

In `internal/actions/actor_user.go`, add after the `AddBuff` method (after line 63):

```go
func (a *UserActor) OnSkillUse(skillName string) bool {
	return a.User.Character.OnSkillUse(skillName, a.User.UserId)
}

func (a *UserActor) OnStatUse(statName string) bool {
	return a.User.Character.OnStatUse(statName, a.User.UserId)
}

func (a *UserActor) OnCriticalSuccess(skillName string) {
	a.User.Character.OnCriticalSuccess(skillName, a.User.UserId)
}

func (a *UserActor) OnCriticalFailure(skillName string) {
	a.User.Character.OnCriticalFailure(skillName, a.User.UserId)
}
```

- [ ] **Step 3: Implement on MobActor**

In `internal/actions/actor_mob.go`, add after the `AddBuff` method (after line 60):

```go
func (a *MobActor) OnSkillUse(skillName string) bool {
	return a.Mob.Character.OnSkillUse(skillName, 0)
}

func (a *MobActor) OnStatUse(statName string) bool {
	return a.Mob.Character.OnStatUse(statName, 0)
}

func (a *MobActor) OnCriticalSuccess(skillName string) {
	a.Mob.Character.OnCriticalSuccess(skillName, 0)
}

func (a *MobActor) OnCriticalFailure(skillName string) {
	a.Mob.Character.OnCriticalFailure(skillName, 0)
}
```

- [ ] **Step 4: Verify compilation**

Run: `go build ./internal/actions/...`
Expected: PASS (no compile errors)

- [ ] **Step 5: Commit**

```bash
git add internal/actions/actor.go internal/actions/actor_user.go internal/actions/actor_mob.go
git commit -m "feat: add OnSkillUse/OnStatUse/OnCriticalSuccess/OnCriticalFailure to Actor interface"
```

---

### Task 2: Add Mob Progression to Combat Loop

**Files:**
- Modify: `internal/combat/combat.go:131-183`

- [ ] **Step 1: Add trackMobAttackProgression helper**

In `internal/combat/combat.go`, add a new import for `actions` at the top, then add this helper function before `AttackMobVsPlayer` (before line 131):

Add to imports:
```go
"github.com/GoMudEngine/GoMud/internal/actions"
```

Add function:
```go
// trackMobAttackProgression mirrors the player progression calls in
// AttackPlayerVsMob / AttackPlayerVsPlayer for a mob attacker.
func trackMobAttackProgression(mob *mobs.Mob, room *rooms.Room, result AttackResult) {
	actor := &actions.MobActor{Mob: mob, Room: room}
	actor.OnStatUse("strength")
	actor.OnStatUse("dexterity")
	if result.Hit {
		combatSkill := string(mob.Character.GetCombatSkillTag())
		actor.OnSkillUse(combatSkill)
		if result.Crit {
			actor.OnCriticalSuccess(combatSkill)
		}
		// Dual-wield weapon-combat tracking (same logic as player path)
		if mob.Character.Equipment.Weapon.ItemId > 0 &&
			mob.Character.Equipment.Offhand.ItemId > 0 &&
			mob.Character.Equipment.Offhand.GetSpec().Type == items.Weapon {
			actor.OnSkillUse(string(skills.WeaponCombat))
		}
	} else if result.Fumble {
		combatSkill := string(mob.Character.GetCombatSkillTag())
		actor.OnCriticalFailure(combatSkill)
	}
}
```

- [ ] **Step 2: Add progression to AttackMobVsPlayer**

In `AttackMobVsPlayer` (line 131), add mob attacker progression after the existing defender dexterity tracking at line 151. Insert after `user.Character.OnStatUse("dexterity", user.UserId)`:

```go
	// Track progression for the attacking mob (mirrors player attacker logic)
	trackMobAttackProgression(mob, room, attackResult)
```

- [ ] **Step 3: Add progression to AttackMobVsMob**

In `AttackMobVsMob` (line 161), add mob attacker AND defender progression after the charmed-user tracking block (after line 180). Insert before the `return attackResult`:

```go
	// Track progression for both mobs
	trackMobAttackProgression(mobAtk, room, attackResult)
	// Defender dexterity (mirrors player defender tracking in AttackMobVsPlayer)
	mobDef.Character.OnStatUse("dexterity", 0)
```

- [ ] **Step 4: Verify compilation**

Run: `go build ./internal/combat/...`
Expected: PASS

- [ ] **Step 5: Run existing tests**

Run: `go test ./internal/combat/... -v`
Expected: All existing tests pass

- [ ] **Step 6: Commit**

```bash
git add internal/combat/combat.go
git commit -m "feat: add mob attacker progression to combat loop (parity with player attacks)"
```

---

### Task 3: Move Progression into Shared ExecuteBash

**Files:**
- Modify: `internal/actions/combat_bash.go:95-101`
- Modify: `internal/usercommands/bash.go:96-101`
- Modify: `internal/mobcommands/bash.go:30-31`

- [ ] **Step 1: Add OnSkillUse to ExecuteBash**

In `internal/actions/combat_bash.go`, add the skill import and progression call after the `RecordAndWait` call at line 100, before the return at line 102:

Add to imports:
```go
"github.com/GoMudEngine/GoMud/internal/skills"
```

Add after `RecordAndWait(...)` (line 100):
```go
	// Progression: weapon-combat on hit (moved from user/mob wrappers)
	if result.Hit {
		actor.OnSkillUse(string(skills.WeaponCombat))
	}
```

- [ ] **Step 2: Remove redundant progression from usercommands/bash.go**

Delete lines 96-101 in `internal/usercommands/bash.go`:
```go
	// Progress weapon combat skill.
	events.AddToQueue(events.SkillUsed{
		UserId:  user.UserId,
		Skill:   skills.WeaponCombat,
		Details: "bash",
	})
```

Remove unused imports `events` and `skills` if they are no longer used elsewhere in the file. (Check first — `skills` may still be used.)

- [ ] **Step 3: Remove redundant progression from mobcommands/bash.go**

Delete line 31 in `internal/mobcommands/bash.go`:
```go
	mob.Character.OnSkillUse(string(skills.WeaponCombat), 0)
```

Remove the `skills` import if no longer used in the file.

- [ ] **Step 4: Verify compilation**

Run: `go build ./internal/actions/... ./internal/usercommands/... ./internal/mobcommands/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/actions/combat_bash.go internal/usercommands/bash.go internal/mobcommands/bash.go
git commit -m "refactor: move bash OnSkillUse into shared ExecuteBash"
```

---

### Task 4: Move Progression into Shared ExecuteKick

**Files:**
- Modify: `internal/actions/combat_kick.go`
- Modify: `internal/usercommands/kick.go:216-228`
- Modify: `internal/mobcommands/kick.go:30-31`

- [ ] **Step 1: Add OnSkillUse to ExecuteKick**

In `internal/actions/combat_kick.go`, add the `skills` import and add after the `RecordAndWait` call (near the end of `ExecuteKick`, before the return):

```go
	// Progression: unarmed-combat on hit (moved from user/mob wrappers)
	if result.Hit {
		actor.OnSkillUse(string(skills.UnarmedCombat))
	}
```

- [ ] **Step 2: Remove redundant progression from usercommands/kick.go**

Delete lines 216-228 in `internal/usercommands/kick.go`:
```go
	// Progress unarmed combat skill.
	variantName := "kick"
	switch res.Variant {
	case actions.KickStomp:
		variantName = "stomp"
	case actions.KickKnee:
		variantName = "knee"
	}
	events.AddToQueue(events.SkillUsed{
		UserId:  user.UserId,
		Skill:   skills.UnarmedCombat,
		Details: variantName,
	})
```

Remove unused imports if applicable.

- [ ] **Step 3: Remove redundant progression from mobcommands/kick.go**

Delete line 31 in `internal/mobcommands/kick.go`:
```go
	mob.Character.OnSkillUse(string(skills.UnarmedCombat), 0)
```

Remove the `skills` import if no longer used.

- [ ] **Step 4: Verify compilation**

Run: `go build ./internal/actions/... ./internal/usercommands/... ./internal/mobcommands/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/actions/combat_kick.go internal/usercommands/kick.go internal/mobcommands/kick.go
git commit -m "refactor: move kick OnSkillUse into shared ExecuteKick"
```

---

### Task 5: Move Progression into Shared ExecuteTrip

**Files:**
- Modify: `internal/actions/combat_trip.go`
- Modify: `internal/usercommands/trip.go:133-142`
- Modify: `internal/mobcommands/trip.go:27-28`

- [ ] **Step 1: Add OnSkillUse to ExecuteTrip**

In `internal/actions/combat_trip.go`, add the `skills` import and add after the `RecordAndWait` call (before the return):

```go
	// Progression: unarmed-combat on hit (moved from user/mob wrappers)
	if result.Hit {
		actor.OnSkillUse(string(skills.UnarmedCombat))
	}
```

- [ ] **Step 2: Remove redundant progression from usercommands/trip.go**

Delete lines 133-142 in `internal/usercommands/trip.go`:
```go
	// Progress unarmed combat skill
	moveLabel := "trip"
	if hasTail {
		moveLabel = "tailsweep"
	}
	events.AddToQueue(events.SkillUsed{
		UserId:  user.UserId,
		Skill:   skills.UnarmedCombat,
		Details: moveLabel,
	})
```

Remove unused imports if applicable.

- [ ] **Step 3: Remove redundant progression from mobcommands/trip.go**

Delete line 28 in `internal/mobcommands/trip.go`:
```go
	mob.Character.OnSkillUse(string(skills.UnarmedCombat), 0)
```

Remove the `skills` import if no longer used.

- [ ] **Step 4: Verify compilation**

Run: `go build ./internal/actions/... ./internal/usercommands/... ./internal/mobcommands/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/actions/combat_trip.go internal/usercommands/trip.go internal/mobcommands/trip.go
git commit -m "refactor: move trip OnSkillUse into shared ExecuteTrip"
```

---

### Task 6: Move Progression into Shared ExecuteGrapple

**Files:**
- Modify: `internal/actions/combat_grapple.go`
- Modify: `internal/usercommands/grapple.go:119-124`
- Modify: `internal/mobcommands/grapple.go:26-27`

- [ ] **Step 1: Add OnSkillUse to ExecuteGrapple**

In `internal/actions/combat_grapple.go`, add the `skills` import and add after the `RecordAndWait` call (before the return):

```go
	// Progression: unarmed-combat on executed grapple (moved from user/mob wrappers)
	if result.Success {
		actor.OnSkillUse(string(skills.UnarmedCombat))
	}
```

Note: Grapple uses `result.Success` not `result.Hit` — check the existing code to confirm the field name.

- [ ] **Step 2: Remove redundant progression from usercommands/grapple.go**

Delete lines 119-124 in `internal/usercommands/grapple.go`:
```go
	// Progress unarmed combat skill
	events.AddToQueue(events.SkillUsed{
		UserId:  user.UserId,
		Skill:   skills.UnarmedCombat,
		Details: "grapple",
	})
```

Remove unused imports if applicable.

- [ ] **Step 3: Remove redundant progression from mobcommands/grapple.go**

Delete line 27 in `internal/mobcommands/grapple.go`:
```go
	mob.Character.OnSkillUse(string(skills.UnarmedCombat), 0)
```

Remove the `skills` import if no longer used.

- [ ] **Step 4: Verify compilation**

Run: `go build ./internal/actions/... ./internal/usercommands/... ./internal/mobcommands/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/actions/combat_grapple.go internal/usercommands/grapple.go internal/mobcommands/grapple.go
git commit -m "refactor: move grapple OnSkillUse into shared ExecuteGrapple"
```

---

### Task 7: Move Progression into Shared InitiateCast

**Files:**
- Modify: `internal/actions/cast.go:49-66`
- Modify: `internal/usercommands/skill.cast.go:228-237`

- [ ] **Step 1: Add OnSkillUse to InitiateCast**

In `internal/actions/cast.go`, the function already receives `actor Actor`. Add skill progression at the end of the function, just before the final `return` of the successful path. You need to determine the cast skill (spellcasting vs manifestation) inside the shared function.

Find where `Initiated` is set to `true` in the result. Add before the return:

```go
	// Progression: cast skill on successful initiation (moved from user wrapper)
	castSkill := string(skills.Spellcasting)
	if result.SpellInfo.School == "manifestation" {
		castSkill = string(skills.Manifestation)
	}
	actor.OnSkillUse(castSkill)
```

Note: Check how the spell school/manifestation flag is determined. The user wrapper uses `isManifestation` which may come from a different check. Read the full `InitiateCast` to find the right field — it may be `spellInfo.School`, `spellInfo.Type`, or a flag. Match the existing logic in `skill.cast.go`.

Add import for `skills` if not already present.

- [ ] **Step 2: Remove redundant progression from usercommands/skill.cast.go**

Delete lines 228-237 in `internal/usercommands/skill.cast.go`:
```go
	// 14. Announce and fire skill-used event for the relevant skill.
	castEventSkill := skills.Spellcasting
	if isManifestation {
		castEventSkill = skills.Manifestation
	}
	events.AddToQueue(events.SkillUsed{
		UserId:  user.UserId,
		Skill:   castEventSkill,
		Details: spellInfo.Name,
	})
```

Remove unused imports if applicable.

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/actions/... ./internal/usercommands/...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/actions/cast.go internal/usercommands/skill.cast.go
git commit -m "refactor: move cast OnSkillUse into shared InitiateCast"
```

---

### Task 8: Create Shared ExecuteTaunt

**Files:**
- Create: `internal/actions/combat_taunt.go`
- Modify: `internal/usercommands/taunt.go`
- Modify: `internal/mobcommands/howl.go`

- [ ] **Step 1: Create ExecuteTaunt shared action**

Create `internal/actions/combat_taunt.go`:

```go
package actions

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// TauntResult holds the outcome of a taunt/howl attempt.
type TauntResult struct {
	Target     AggroTarget
	Executed   bool
	OnCooldown bool
	NoTarget   bool
	Hit        bool
	Crit       bool
	Fumble     bool
	Damage     int
	DmgDesc    string
	SelfDamage int // conviction self-damage on fumble
}

// ExecuteTaunt performs the core conviction-damage taunt shared between player
// taunt and mob howl. Handles cooldown, target resolution, opposed roll,
// damage calculation, and progression. Callers handle all messaging.
func ExecuteTaunt(actor Actor) TauntResult {
	char := actor.GetCharacter()

	if char.Aggro == nil {
		return TauntResult{NoTarget: true}
	}

	// Cooldown check
	cfg := configs.GetBalanceConfig()
	cooldownStr := fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)
	if !char.Cooldowns.Try("special-move", cooldownStr) {
		return TauntResult{OnCooldown: true}
	}

	// Resolve target
	target := ResolveAggroTarget(char.Aggro)
	if !target.Found {
		return TauntResult{NoTarget: true}
	}

	// Conviction attack: Charisma + rhetoric vs Willpower + rhetoric
	skillWeight := float64(cfg.SkillWeight)
	attackerRhetoric := float64(char.GetSkillLevel(skills.Rhetoric)) * skillWeight
	attackScore := float64(char.Stats.Charisma.ValueAdj) + attackerRhetoric

	defenderRhetoric := float64(target.Char.GetSkillLevel(skills.Rhetoric)) * skillWeight
	defenseScore := float64(target.Char.Stats.Willpower.ValueAdj) + defenderRhetoric

	// Apply conviction depletion penalty
	cpPenalty := float64(cfg.ConvictionPenaltyMax)
	convMult := combat.ResourceMultiplier(char.Conviction, char.ConvictionMax.Value, cpPenalty)
	attackScore *= convMult

	// Opposed roll
	attackSuccess, _, atkRoll, _ := dice.OpposedRollStat(attackScore, defenseScore)

	// Determine source/target types for analytics
	sourceType := combat.User
	if !actor.IsPlayer() {
		sourceType = combat.Mob
	}
	targetType := combat.User
	if target.MobInstanceId > 0 {
		targetType = combat.Mob
	}

	// Fumble: self-conviction damage
	if atkRoll.ZScore <= -2.0 {
		selfDmg := char.Stats.Charisma.ValueAdj / 10
		if selfDmg < 1 {
			selfDmg = 1
		}
		char.Conviction -= selfDmg
		if char.Conviction < 0 {
			char.Conviction = 0
		}

		combat.RecordSpecialMove(sourceType, targetType, "taunt", false, 0,
			char, target.Char, util.GetRoundCount())

		// Progression fires even on fumble
		actor.OnSkillUse(string(skills.Rhetoric))

		if char.Aggro != nil {
			char.Aggro.RoundsWaiting = 1
		}

		return TauntResult{
			Target:     target,
			Executed:   true,
			Fumble:     true,
			SelfDamage: selfDmg,
		}
	}

	if attackSuccess {
		rawDmg := combat.CalcRawDamage(
			char.Stats.Charisma.ValueAdj,
			int(attackerRhetoric),
			0.5, // TauntBaseMult
			combat.ChannelConviction,
		)
		rawDmg *= convMult

		isCrit := atkRoll.ZScore >= 2.0

		var dmg int
		if isCrit {
			dmgRoll := dice.RollStat(rawDmg)
			dmg = int(math.Round(dmgRoll.Value))
		} else {
			mitigPct := target.Char.GetConvictionMitigation()
			cap := combat.MitigationCap(combat.ChannelConviction)
			dmgMean := combat.ApplyMitigation(rawDmg, mitigPct, cap)
			dmgRoll := dice.RollStat(dmgMean)
			dmg = int(math.Round(dmgRoll.Value))
		}
		if dmg < 1 {
			dmg = 1
		}

		target.Char.Conviction -= dmg
		if target.Char.Conviction < 0 {
			target.Char.Conviction = 0
		}

		convMaxRef := target.Char.ConvictionMax.Value
		if convMaxRef <= 0 {
			convMaxRef = 1
		}
		dmgDesc := combat.GetConvictionDamageDescription(dmg, convMaxRef)

		combat.RecordSpecialMove(sourceType, targetType, "taunt", true, dmg,
			char, target.Char, util.GetRoundCount())

		// Rhetoric crit received → charisma progression for defender (player only)
		if isCrit && target.UserId > 0 {
			target.Char.OnCritReceived("conviction", target.UserId)
		}

		actor.OnSkillUse(string(skills.Rhetoric))

		if char.Aggro != nil {
			char.Aggro.RoundsWaiting = 1
		}

		return TauntResult{
			Target:   target,
			Executed: true,
			Hit:      true,
			Crit:     isCrit,
			Damage:   dmg,
			DmgDesc:  dmgDesc,
		}
	}

	// Miss
	combat.RecordSpecialMove(sourceType, targetType, "taunt", false, 0,
		char, target.Char, util.GetRoundCount())

	actor.OnSkillUse(string(skills.Rhetoric))

	if char.Aggro != nil {
		char.Aggro.RoundsWaiting = 1
	}

	return TauntResult{
		Target:   target,
		Executed: true,
	}
}
```

Note: You'll need `"fmt"` in the imports for the cooldown string. Check if `OnCritReceived` exists on the Character type — the player taunt uses `targetPlayer.Character.OnCritReceived("conviction", targetPlayer.UserId)` at line 183. Verify the method signature matches.

- [ ] **Step 2: Rewrite usercommands/taunt.go as thin wrapper**

Replace the core logic in `internal/usercommands/taunt.go` with a call to `ExecuteTaunt`. Keep the out-of-combat target resolution (lines 22-41) and the `sendTauntMessages` helper function. The new structure:

```go
func Taunt(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Must be in combat or specify a target to use taunt
	if user.Character.Aggro == nil {
		if rest == "" {
			user.SendText("Taunt whom?")
			return true, nil
		}
		targetPId, targetMId := room.FindByName(rest)
		if targetPId == user.UserId {
			user.SendText("You can't taunt yourself.")
			return true, nil
		}
		if targetPId == 0 && targetMId == 0 {
			user.SendText("You don't see them here.")
			return true, nil
		}
		if targetMId > 0 {
			user.Character.SetAggro(0, targetMId, characters.DefaultAttack)
		} else {
			user.Character.SetAggro(targetPId, 0, characters.DefaultAttack)
		}
	}

	actor := &actions.UserActor{User: user, Room: room}
	res := actions.ExecuteTaunt(actor)

	if res.OnCooldown {
		user.SendText("You need a moment to recover before attempting another special move.")
		return true, nil
	}
	if res.NoTarget {
		user.SendText("Your target is gone!")
		return true, nil
	}

	targetName := res.Target.Name
	targetType := "mob"
	var targetPlayer *users.UserRecord
	if res.Target.UserId > 0 {
		targetType = "user"
		targetPlayer = users.GetByUserId(res.Target.UserId)
	}

	if res.Fumble {
		sendTauntMessages(combat.TauntFumble, "", user.Character.Name, targetName,
			"username", targetType, user, targetPlayer, room, res.Target.UserId)
	} else if res.Hit {
		intensity := combat.TauntHit
		if res.Crit {
			intensity = combat.TauntCritical
		}
		sendTauntMessages(intensity, res.DmgDesc, user.Character.Name, targetName,
			"username", targetType, user, targetPlayer, room, res.Target.UserId)
	} else {
		sendTauntMessages(combat.TauntMiss, "", user.Character.Name, targetName,
			"username", targetType, user, targetPlayer, room, res.Target.UserId)
	}

	return true, nil
}
```

Keep the existing `sendTauntMessages` function unchanged.

Remove now-unused imports: `"math"`, `"github.com/GoMudEngine/GoMud/internal/configs"`, `"github.com/GoMudEngine/GoMud/internal/dice"`, `"github.com/GoMudEngine/GoMud/internal/skills"`, `"github.com/GoMudEngine/GoMud/internal/util"`. Keep `"fmt"`, `events`, `actions`, `characters`, `combat`, `mobs`, `rooms`, `users`.

- [ ] **Step 3: Rewrite mobcommands/howl.go as thin wrapper**

Replace the contents of `internal/mobcommands/howl.go` with a thin wrapper:

```go
package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Howl is a mob conviction attack — delegates to shared ExecuteTaunt with
// mob-flavored messaging.
func Howl(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	if mob.Character.Aggro == nil {
		return true, nil
	}

	actor := &actions.MobActor{Mob: mob, Room: room}
	res := actions.ExecuteTaunt(actor)

	if !res.Executed {
		return true, nil
	}

	mobName := mob.Character.Name
	targetName := res.Target.Name

	var targetPlayer *users.UserRecord
	if res.Target.UserId > 0 {
		targetPlayer = users.GetByUserId(res.Target.UserId)
	}
	canSee := targetPlayer == nil || canSeeInDark(targetPlayer, room)

	if res.Fumble {
		sendAudioRoomText(room, mob,
			`Something lets out a pitiful howl that trails off weakly.`,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> lets out a pitiful howl that trails off weakly.`, mobName))
	} else if res.Hit {
		dmgDesc := res.DmgDesc
		if targetPlayer != nil {
			if canSee {
				targetPlayer.SendText(fmt.Sprintf(
					`<ansi fg="mobname">%s</ansi>'s menacing howl shakes your resolve! (<ansi fg="damage">%s</ansi>)`,
					mobName, dmgDesc))
			} else {
				targetPlayer.SendText(fmt.Sprintf(
					`A menacing howl shakes your resolve! (<ansi fg="damage">%s</ansi>)`, dmgDesc))
			}
		}
		sendAudioRoomText(room, mob,
			fmt.Sprintf(`Something lets out a bone-chilling howl at <ansi fg="username">%s</ansi>!`, targetName),
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> throws back its head and lets out a bone-chilling howl at <ansi fg="username">%s</ansi>!`, mobName, targetName))
	} else {
		// Miss
		if targetPlayer != nil {
			if canSee {
				targetPlayer.SendText(fmt.Sprintf(
					`<ansi fg="mobname">%s</ansi> howls, but you steel yourself against the sound.`, mobName))
			} else {
				targetPlayer.SendText(`Something howls, but you steel yourself against the sound.`)
			}
		}
		sendAudioRoomText(room, mob,
			fmt.Sprintf(`Something howls menacingly at <ansi fg="username">%s</ansi>, but it has no effect.`, targetName),
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> howls menacingly at <ansi fg="username">%s</ansi>, but it has no effect.`, mobName, targetName))
	}

	return true, nil
}
```

Note: `sendAudioRoomText` is a helper already available in the mobcommands package. Verify it exists by searching for it — it should be defined in a shared mob command helper file.

- [ ] **Step 4: Verify compilation**

Run: `go build ./internal/actions/... ./internal/usercommands/... ./internal/mobcommands/...`
Expected: PASS

- [ ] **Step 5: Run existing tests**

Run: `go test ./internal/actions/... -v`
Expected: All existing tests pass

- [ ] **Step 6: Commit**

```bash
git add internal/actions/combat_taunt.go internal/usercommands/taunt.go internal/mobcommands/howl.go
git commit -m "feat: unify taunt/howl via shared ExecuteTaunt action"
```

---

### Task 9: Create Shared ExecuteBite

**Files:**
- Create: `internal/actions/combat_bite.go`
- Modify: `internal/mobcommands/bite.go`

- [ ] **Step 1: Create ExecuteBite shared action**

Create `internal/actions/combat_bite.go`:

```go
package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// BiteResult holds the outcome of a bite attempt.
type BiteResult struct {
	Target      AggroTarget
	MoveResult  combat.SkillMoveResult
	Executed    bool
	OnCooldown  bool
	NoTarget    bool
	DrainAmount int // HP drained on hit
}

// ExecuteBite performs a life-drain bite attack. Shared between mob bite
// and future player species-gated bite. Callers handle all messaging.
func ExecuteBite(actor Actor) BiteResult {
	char := actor.GetCharacter()

	if char.Aggro == nil {
		return BiteResult{NoTarget: true}
	}

	cfg := configs.GetBalanceConfig()
	cooldownStr := fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)
	if !char.Cooldowns.Try("special-move", cooldownStr) {
		return BiteResult{OnCooldown: true}
	}

	target := ResolveAggroTarget(char.Aggro)
	if !target.Found {
		return BiteResult{NoTarget: true}
	}

	result := combat.ExecuteSkillMove(combat.SkillMoveParams{
		Attacker:        char,
		Defender:        target.Char,
		AttackStat:      char.Stats.Strength.ValueAdj,
		AttackSkill:     char.GetSkillLevel(skills.UnarmedCombat),
		DefenseStat:     target.Char.Stats.Dexterity.ValueAdj,
		DefenseSkill:    target.Char.GetCombatSkillLevel(),
		DamagePercent:   0.60,
		KnockdownChance: 0,
		SkillRank:       char.GetSkillLevel(skills.UnarmedCombat),
		DamageStat:      char.Stats.Strength.ValueAdj,
	})

	drain := 0
	if result.Hit && result.Damage > 0 {
		drain = int(float64(result.Damage) * 0.50)
		char.Health += drain
		if char.Health > char.HealthMax.Value {
			char.Health = char.HealthMax.Value
		}
	}

	// Determine source/target types for analytics
	sourceType := combat.User
	if !actor.IsPlayer() {
		sourceType = combat.Mob
	}
	targetType := combat.User
	if target.MobInstanceId > 0 {
		targetType = combat.Mob
	}

	dmgRecorded := 0
	if result.Hit {
		dmgRecorded = result.Damage
	}
	RecordAndWait(char, "bite", sourceType, target.Char, targetType, result.Hit, dmgRecorded, util.GetRoundCount())

	// Progression
	if result.Hit {
		actor.OnSkillUse(string(skills.UnarmedCombat))
	}

	return BiteResult{
		Target:      target,
		MoveResult:  result,
		Executed:    true,
		DrainAmount: drain,
	}
}
```

- [ ] **Step 2: Rewrite mobcommands/bite.go as thin wrapper**

Replace the contents of `internal/mobcommands/bite.go`:

```go
package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Bite is a vampire-only special attack that deals moderate physical damage
// and heals the attacker for half the damage inflicted (life drain).
func Bite(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	if mob.Character.Aggro == nil {
		return true, nil
	}

	actor := &actions.MobActor{Mob: mob, Room: room}
	res := actions.ExecuteBite(actor)

	if !res.Executed {
		return true, nil
	}

	target := res.Target
	result := res.MoveResult
	mobName := mob.Character.Name
	dmgDesc := combat.GetDamageDescription(result.Damage, result.TargetMaxHP)

	var targetUser *users.UserRecord
	if target.UserId > 0 {
		targetUser = users.GetByUserId(target.UserId)
	}
	canSee := targetUser == nil || canSeeInDark(targetUser, room)

	if result.Hit {
		if targetUser != nil {
			if canSee {
				targetUser.SendText(fmt.Sprintf(
					`<ansi fg="mobname">%s</ansi> sinks its fangs into you, drawing strength from the wound! (<ansi fg="damage">%s</ansi> damage)`,
					mobName, dmgDesc))
			} else {
				targetUser.SendText(fmt.Sprintf(
					`Something sinks its fangs into you in the darkness, drawing strength from the wound! (<ansi fg="damage">%s</ansi> damage)`,
					dmgDesc))
			}
		}
		if result.Damage > 0 {
			sendRoomText(room,
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> sinks its fangs into <ansi fg="username">%s</ansi> and draws strength from the wound!`,
					mobName, target.Name),
				target.UserId)
		} else {
			sendRoomText(room,
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> sinks its fangs into <ansi fg="username">%s</ansi>!`,
					mobName, target.Name),
				target.UserId)
		}
	} else {
		if targetUser != nil {
			if canSee {
				targetUser.SendText(fmt.Sprintf(
					`<ansi fg="mobname">%s</ansi> snaps its fangs at you, but misses!`,
					mobName))
			} else {
				targetUser.SendText(`Something snaps its fangs at you in the darkness, but misses!`)
			}
		}
		sendRoomText(room,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> snaps its fangs at <ansi fg="username">%s</ansi>, but misses!`,
				mobName, target.Name),
			target.UserId)
	}

	return true, nil
}
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/actions/... ./internal/mobcommands/...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/actions/combat_bite.go internal/mobcommands/bite.go
git commit -m "feat: create shared ExecuteBite action with progression"
```

---

### Task 10: Create Shared ExecuteHamstring

**Files:**
- Create: `internal/actions/combat_hamstring.go`
- Modify: `internal/mobcommands/hamstring.go`

- [ ] **Step 1: Create ExecuteHamstring shared action**

Create `internal/actions/combat_hamstring.go`:

```go
package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// HamstringResult holds the outcome of a hamstring attempt.
type HamstringResult struct {
	Target     AggroTarget
	MoveResult combat.SkillMoveResult
	Executed   bool
	OnCooldown bool
	NoTarget   bool
	BleedDmg   int // per-tick bleed damage applied on hit
}

// ExecuteHamstring performs a bleed-inflicting attack. Shared between mob
// hamstring and future player species-gated hamstring. Callers handle messaging.
func ExecuteHamstring(actor Actor) HamstringResult {
	char := actor.GetCharacter()

	if char.Aggro == nil {
		return HamstringResult{NoTarget: true}
	}

	cfg := configs.GetBalanceConfig()
	cooldownStr := fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)
	if !char.Cooldowns.Try("special-move", cooldownStr) {
		return HamstringResult{OnCooldown: true}
	}

	target := ResolveAggroTarget(char.Aggro)
	if !target.Found {
		return HamstringResult{NoTarget: true}
	}

	result := combat.ExecuteSkillMove(combat.SkillMoveParams{
		Attacker:        char,
		Defender:        target.Char,
		AttackStat:      char.Stats.Dexterity.ValueAdj,
		AttackSkill:     char.GetSkillLevel(skills.UnarmedCombat),
		DefenseStat:     target.Char.Stats.Dexterity.ValueAdj,
		DefenseSkill:    target.Char.GetCombatSkillLevel(),
		DamagePercent:   float64(cfg.TripDamagePercent),
		KnockdownChance: 0,
		SkillRank:       char.GetSkillLevel(skills.UnarmedCombat),
		DamageStat:      char.Stats.Strength.ValueAdj,
	})

	bleedDmg := 0
	if result.Hit {
		bleedDmg = char.Stats.Strength.ValueAdj / 10
		if bleedDmg < 2 {
			bleedDmg = 2
		}
		target.Char.AddCondition(characters.ConditionBleeding, 5, float64(bleedDmg), "hamstring")
	}

	// Determine source/target types for analytics
	sourceType := combat.User
	if !actor.IsPlayer() {
		sourceType = combat.Mob
	}
	targetType := combat.User
	if target.MobInstanceId > 0 {
		targetType = combat.Mob
	}

	dmgRecorded := 0
	if result.Hit {
		dmgRecorded = result.Damage
	}
	combat.RecordSpecialMove(sourceType, targetType, "hamstring", result.Hit, dmgRecorded,
		char, target.Char, util.GetRoundCount())

	// Progression
	if result.Hit {
		actor.OnSkillUse(string(skills.UnarmedCombat))
	}

	if char.Aggro != nil {
		char.Aggro.RoundsWaiting = 1
	}

	return HamstringResult{
		Target:     target,
		MoveResult: result,
		Executed:   true,
		BleedDmg:   bleedDmg,
	}
}
```

- [ ] **Step 2: Rewrite mobcommands/hamstring.go as thin wrapper**

Replace the contents of `internal/mobcommands/hamstring.go`:

```go
package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Hamstring is a wolf physical attack that applies a bleed condition.
func Hamstring(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	if mob.Character.Aggro == nil {
		return true, nil
	}

	actor := &actions.MobActor{Mob: mob, Room: room}
	res := actions.ExecuteHamstring(actor)

	if !res.Executed {
		return true, nil
	}

	target := res.Target
	result := res.MoveResult
	mobName := mob.Character.Name

	var targetUser *users.UserRecord
	if target.UserId > 0 {
		targetUser = users.GetByUserId(target.UserId)
	}
	canSee := targetUser == nil || canSeeInDark(targetUser, room)

	if result.Hit {
		dmgDesc := combat.GetDamageDescription(result.Damage, result.TargetMaxHP)
		if targetUser != nil {
			if canSee {
				targetUser.SendText(fmt.Sprintf(
					`<ansi fg="mobname">%s</ansi> rakes its fangs across your legs, opening deep wounds! (<ansi fg="damage">%s</ansi> damage)`,
					mobName, dmgDesc))
			} else {
				targetUser.SendText(fmt.Sprintf(
					`Something rakes its fangs across your legs, opening deep wounds! (<ansi fg="damage">%s</ansi> damage)`,
					dmgDesc))
			}
		}
		sendRoomText(room,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> lunges low and rakes its fangs across <ansi fg="username">%s</ansi>'s legs!`,
				mobName, target.Name),
			target.UserId)
	} else {
		if targetUser != nil {
			if canSee {
				targetUser.SendText(fmt.Sprintf(
					`<ansi fg="mobname">%s</ansi> lunges at your legs, but you sidestep the attack!`, mobName))
			} else {
				targetUser.SendText(`Something lunges at your legs, but you sidestep the attack!`)
			}
		}
		sendRoomText(room,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> lunges at <ansi fg="username">%s</ansi>'s legs, but misses!`,
				mobName, target.Name),
			target.UserId)
	}

	return true, nil
}
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/actions/... ./internal/mobcommands/...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/actions/combat_hamstring.go internal/mobcommands/hamstring.go
git commit -m "feat: create shared ExecuteHamstring action with progression"
```

---

### Task 11: Remove Deprecated Commands (roar, throw, backstab)

**Files:**
- Delete: `internal/mobcommands/roar.go`
- Delete: `internal/mobcommands/throw.go`
- Delete: `internal/mobcommands/backstab.go`
- Modify: `internal/mobcommands/mobcommands.go:25-81`
- Modify: `_datafiles/world/dogmud/mobs/thornwall_city/scripts/272-chrysalis_phantom.js:33-38`
- Modify: `_datafiles/world/dogmud/keywords.yaml:222,229`

- [ ] **Step 1: Delete roar.go, throw.go, backstab.go**

```bash
rm internal/mobcommands/roar.go
rm internal/mobcommands/throw.go
rm internal/mobcommands/backstab.go
```

- [ ] **Step 2: Remove from command registry**

In `internal/mobcommands/mobcommands.go`, remove these three lines from the `mobCommands` map:

Remove:
```go
		"backstab":       {Backstab, false},
```

Remove:
```go
		"roar":           {Roar, false},
```

Remove:
```go
		"throw":  {Throw, false},
```

- [ ] **Step 3: Fix chrysalis phantom script**

In `_datafiles/world/dogmud/mobs/thornwall_city/scripts/272-chrysalis_phantom.js`, replace `backstab` with `attack` at line 37:

Change:
```js
            mob.Command('backstab');
```
To:
```js
            mob.Command('attack');
```

Also update the comment at line 33 from "backstab" to "attack":

Change:
```js
    // If not fighting and players are in the room, backstab
```
To:
```js
    // If not fighting and players are in the room, surprise attack
```

- [ ] **Step 4: Remove keyword aliases**

In `_datafiles/world/dogmud/keywords.yaml`, remove lines 222 and 229:

Remove:
```yaml
  throw:              ['toss']
```

Remove:
```yaml
  backstab:           ['bs']
```

- [ ] **Step 5: Verify compilation**

Run: `go build ./internal/mobcommands/...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add -u internal/mobcommands/roar.go internal/mobcommands/throw.go internal/mobcommands/backstab.go
git add internal/mobcommands/mobcommands.go
git add _datafiles/world/dogmud/mobs/thornwall_city/scripts/272-chrysalis_phantom.js
git add _datafiles/world/dogmud/keywords.yaml
git commit -m "feat: remove deprecated mob commands (roar, throw, backstab)"
```

---

### Task 12: Rename alchemy → selljunk

**Files:**
- Delete: `internal/mobcommands/alchemy.go`
- Create: `internal/mobcommands/selljunk.go`
- Modify: `internal/mobcommands/mobcommands.go`

- [ ] **Step 1: Create selljunk.go**

Create `internal/mobcommands/selljunk.go` with the same logic as alchemy.go but renamed:

```go
package mobcommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Selljunk converts mob inventory items into gold (1 coin per item).
// Formerly named "alchemy" — renamed for clarity.
func Selljunk(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if args[0] == "random" {
		if len(mob.Character.Items) > 0 {
			matchItem := mob.Character.Items[util.Rand(len(mob.Character.Items))]
			Selljunk(matchItem.Name(), mob, room)
		}
		return true, nil
	}

	if args[0] == "all" {
		iCopies := []items.Item{}
		for _, item := range mob.Character.Items {
			iCopies = append(iCopies, item)
		}
		for _, item := range iCopies {
			Selljunk(item.Name(), mob, room)
		}
		return true, nil
	}

	matchItem, found := mob.Character.FindInBackpack(rest)
	if found {
		mob.Character.RemoveItem(matchItem)

		events.AddToQueue(events.ItemOwnership{
			MobInstanceId: mob.InstanceId,
			Item:          matchItem,
			Gained:        false,
		})

		mob.Character.Gold += 1
		room.SendText(
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> chants softly. Their <ansi fg="item">%s</ansi> slowly levitates in the air, trembles briefly and then in a flash of light becomes a gold coin!`, mob.Character.Name, matchItem.DisplayName()))
	}

	return true, nil
}
```

- [ ] **Step 2: Delete alchemy.go**

```bash
rm internal/mobcommands/alchemy.go
```

- [ ] **Step 3: Update command registry**

In `internal/mobcommands/mobcommands.go`, replace the alchemy entry:

Change:
```go
		"alchemy":        {Alchemy, false},
```
To:
```go
		"selljunk":       {Selljunk, false},
```

- [ ] **Step 4: Verify compilation**

Run: `go build ./internal/mobcommands/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add -u internal/mobcommands/alchemy.go
git add internal/mobcommands/selljunk.go internal/mobcommands/mobcommands.go
git commit -m "refactor: rename mob alchemy command to selljunk"
```

---

### Task 13: Update Divergences.go

**Files:**
- Modify: `internal/actions/divergences.go:115-149`

- [ ] **Step 1: Update mobOnlyCommands map**

In `internal/actions/divergences.go`, replace the `mobOnlyCommands` var block (lines 115-149) with:

```go
// mobOnlyCommands lists mob commands that intentionally have no user equivalent.
//
// Future work candidates (not mob-only forever):
//   - bite: shared action (ExecuteBite), future player species-gated ability
//   - hamstring: shared action (ExecuteHamstring), future player species-gated ability
var mobOnlyCommands = map[string]string{
	// --- Mob AI behaviours ---
	"aid":            "mob-ai",
	"befriend":       "mob-ai",
	"bite":           "mob-ai: shared ExecuteBite action, future player ability",
	"callforhelp":    "mob-ai",
	"charge":         "mob-ai",
	"consume":        "mob-ai",
	"converse":       "mob-ai",
	"despawn":        "mob-ai",
	"givequest":      "mob-ai",
	"hamstring":      "mob-ai: shared ExecuteHamstring action, future player ability",
	"howl":           "mob-ai: shared ExecuteTaunt action with mob flavor text",
	"lookforaid":     "mob-ai",
	"lookfortrouble": "mob-ai",
	"pathto":         "mob-ai",
	"portal":         "mob-ai",
	"replyto":        "mob-ai",
	"sayto":          "mob-ai",
	"saytoonly":      "mob-ai",
	"selljunk":       "mob-ai: converts inventory items to gold",
	"wander":         "mob-ai",
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/actions/...`
Expected: PASS

- [ ] **Step 3: Run parity audit**

Run: `go test ./internal/actions/... -v`
Expected: All tests pass

- [ ] **Step 4: Commit**

```bash
git add internal/actions/divergences.go
git commit -m "refactor: update divergences.go for parity changes"
```

---

### Task 14: Full Build and Test Verification

**Files:** None (verification only)

- [ ] **Step 1: Full build**

Run: `go build ./...`
Expected: PASS with no errors

- [ ] **Step 2: Run all tests**

Run: `go test ./... 2>&1 | head -100`
Expected: All tests pass

- [ ] **Step 3: Check for import cleanup**

Run: `goimports -l ./internal/actions/ ./internal/usercommands/ ./internal/mobcommands/ ./internal/combat/`

If any files are listed, fix the imports. Common issues:
- `events` or `skills` imports left in user/mob wrappers after removing progression calls
- Missing `fmt` import in new files

- [ ] **Step 4: Final commit (if any fixups needed)**

```bash
git add -A
git commit -m "fix: import cleanup after parity refactor"
```
