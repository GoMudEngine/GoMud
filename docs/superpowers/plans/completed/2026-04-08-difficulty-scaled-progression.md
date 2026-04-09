# Difficulty-Scaled Skill Progression — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Scale skill progression chance with action difficulty so hard spells/crafts reward more than trivial ones, remove double spell progression, and guard against empty-room AoE cheese.

**Architecture:** Add `OnSkillUseScaled(skill, userId, bonus)` to Character, wire spell difficulty and recipe skill_minimum through existing progression paths, add three config knobs. Remove initiation-time spell progression. Guard AoE/self-cast at resolution.

**Tech Stack:** Go, YAML config

---

### Task 1: Add Config Knobs

**Files:**
- Modify: `internal/configs/config.balance.go:193-202` (struct fields)
- Modify: `internal/configs/config.balance.go:720-747` (Validate defaults)
- Modify: `_datafiles/config.yaml` (add entries near existing spellcasting section)

- [ ] **Step 1: Add struct fields to Balance**

In `internal/configs/config.balance.go`, add three new fields after the
existing `SpellProficiencyCastsPerPoint` line (around line 202):

```go
	SpellDifficultyProgressionScale ConfigFloat `yaml:"SpellDifficultyProgressionScale"` // Per-point spell difficulty bonus to skill progression (default 0.01)
	CraftDifficultyProgressionScale ConfigFloat `yaml:"CraftDifficultyProgressionScale"` // Per-point recipe skill_minimum bonus to skill progression (default 0.02)
	SelfCastProgressionMultiplier   ConfigFloat `yaml:"SelfCastProgressionMultiplier"`   // Progression multiplier when spell only targets self (default 0.5)
```

- [ ] **Step 2: Add Validate defaults**

In `internal/configs/config.balance.go`, in the `Validate()` method after
the `SpellProficiencyCastsPerPoint` default block (around line 747), add:

```go
	if b.SpellDifficultyProgressionScale <= 0 {
		b.SpellDifficultyProgressionScale = 0.01
	}
	if b.CraftDifficultyProgressionScale <= 0 {
		b.CraftDifficultyProgressionScale = 0.02
	}
	if b.SelfCastProgressionMultiplier <= 0 {
		b.SelfCastProgressionMultiplier = 0.5
	}
```

- [ ] **Step 3: Add config.yaml entries**

In `_datafiles/config.yaml`, after the `SpellProficiencyCastsPerPoint`
entry in the Balance section, add:

```yaml
  # ── DIFFICULTY-SCALED PROGRESSION ──────────────────────────────────────────
  # Harder spells and recipes give proportionally more skill progression.
  # Formula (spells):  bonusMultiplier = 1.0 + difficulty * Scale
  # Formula (crafts):  bonusMultiplier = 1.0 + skill_minimum * Scale
  SpellDifficultyProgressionScale: 0.01   # difficulty 75 → 1.75x progression
  CraftDifficultyProgressionScale: 0.02   # skill_minimum 50 → 2.0x progression
  SelfCastProgressionMultiplier: 0.5      # self-only buffs get half progression
```

- [ ] **Step 4: Build and verify**

Run: `go build ./...`
Expected: clean build, no errors

- [ ] **Step 5: Commit**

```bash
git add internal/configs/config.balance.go _datafiles/config.yaml
git commit -m "feat: add difficulty-scaled progression config knobs

Three new Balance config knobs:
- SpellDifficultyProgressionScale (default 0.01)
- CraftDifficultyProgressionScale (default 0.02)
- SelfCastProgressionMultiplier (default 0.5)

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: Add OnSkillUseScaled Method

**Files:**
- Modify: `internal/characters/progression.go:223-250`
- Test: `internal/characters/progression_test.go`

- [ ] **Step 1: Write the test**

Add to `internal/characters/progression_test.go`:

```go
func TestOnSkillUseScaled_PassesBonusMultiplier(t *testing.T) {
	// OnSkillUseScaled should exist and accept a bonus multiplier.
	// We can't easily test the full progression pipeline in a unit test
	// (it depends on configs and RNG), but we can verify the method exists
	// and delegates properly by checking it compiles and doesn't panic.
	c := Character{
		Skills:      map[string]int{string(skills.Spellcasting): 5},
		SkillUseCount: map[string]int{},
		StatUseCount:  map[string]int{},
	}
	// Should not panic — just exercises the code path
	c.OnSkillUseScaled(string(skills.Spellcasting), 0, 1.5)
}

func TestOnSkillUse_DelegatesToScaled(t *testing.T) {
	c := Character{
		Skills:      map[string]int{string(skills.Spellcasting): 5},
		SkillUseCount: map[string]int{},
		StatUseCount:  map[string]int{},
	}
	// OnSkillUse should work exactly as before (delegates with bonus=1.0)
	c.OnSkillUse(string(skills.Spellcasting), 0)
	if c.SkillUseCount[string(skills.Spellcasting)] != 1 {
		t.Errorf("Expected use count 1, got %d", c.SkillUseCount[string(skills.Spellcasting)])
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd internal/characters && go test -run "TestOnSkillUseScaled|TestOnSkillUse_Delegates" -v`
Expected: FAIL — `OnSkillUseScaled` does not exist yet

- [ ] **Step 3: Implement OnSkillUseScaled**

In `internal/characters/progression.go`, rename the current `OnSkillUse`
body into a new `OnSkillUseScaled` method and make `OnSkillUse` delegate:

Replace the existing `OnSkillUse` method (lines 223-250) with:

```go
// OnSkillUse is called whenever a character uses a skill in gameplay.
// Tracks usage and, if progression is enabled, rolls for skill advancement.
// Also auto-tracks and progresses the skill's primary governing stat.
// Returns true if the skill actually increased.
func (c *Character) OnSkillUse(skillName string, userId int) bool {
	return c.OnSkillUseScaled(skillName, userId, 1.0)
}

// OnSkillUseScaled is like OnSkillUse but accepts a bonus multiplier that
// scales the progression chance. Used for difficulty-scaled progression
// where harder spells/crafts reward proportionally more skill growth.
func (c *Character) OnSkillUseScaled(skillName string, userId int, bonusMultiplier float64) bool {
	c.TrackSkillUse(skillName)
	mudlog.Debug("Progression", "event", "skill_use", "skill", skillName, "bonus", fmt.Sprintf("%.2f", bonusMultiplier), "character", c.Name)

	gained := false
	if configs.GetGamePlayConfig().UseSkillProgression {
		gained = c.CheckSkillProgression(skillName, userId, bonusMultiplier)
	}

	// Auto-track and progress the skill's primary governing stat
	if primaryStat := skills.GetSkillPrimaryStat(skillName); primaryStat != "" {
		c.OnStatUse(primaryStat, userId)
	}

	// Emit SkillUsed event for quest engine and other listeners
	if userId > 0 {
		events.AddToQueue(events.SkillUsed{
			UserId: userId,
			Skill:  skills.SkillTag(skillName),
		})
	}

	return gained
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd internal/characters && go test -run "TestOnSkillUseScaled|TestOnSkillUse_Delegates" -v`
Expected: PASS

- [ ] **Step 5: Build full project**

Run: `go build ./...`
Expected: clean build (all existing callers of `OnSkillUse` unchanged)

- [ ] **Step 6: Commit**

```bash
git add internal/characters/progression.go internal/characters/progression_test.go
git commit -m "feat: add OnSkillUseScaled for difficulty-based progression

OnSkillUseScaled accepts a bonus multiplier that flows into
CheckSkillProgression. OnSkillUse delegates with 1.0 so all
existing callsites are unchanged.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: Remove Double Spell Progression

**Files:**
- Modify: `internal/actions/cast.go:289-294`

- [ ] **Step 1: Remove the initiation-time OnSkillUse call**

In `internal/actions/cast.go`, delete lines 289-294 (the block that calls
`actor.OnSkillUse(castSkill)` at the end of `InitiateCast`):

Remove this block:
```go
	// Progression: cast skill on successful initiation (covers both players and mobs).
	castSkill := string(skills.Spellcasting)
	if spellInfo.HasSchool(spells.SchoolManifestation) {
		castSkill = string(skills.Manifestation)
	}
	actor.OnSkillUse(castSkill)
```

This leaves progression to fire only at resolution time in
`NewRound_DoCombat_helpers.go`.

- [ ] **Step 2: Build and verify**

Run: `go build ./...`
Expected: clean build. The `skills` import in cast.go may become unused —
remove it if the compiler complains.

- [ ] **Step 3: Run cast tests**

Run: `cd internal/actions && go test -v -count=1`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/actions/cast.go
git commit -m "fix: remove double spell progression trigger

Spells were firing OnSkillUse at both initiation and resolution,
giving 2x progression per cast. Now only fires at resolution when
the spell actually completes.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: Wire Spell Difficulty Into Resolution Progression

**Files:**
- Modify: `internal/hooks/NewRound_DoCombat_helpers.go:277-284` (player spell resolution)
- Modify: `internal/hooks/NewRound_DoCombat_helpers.go:378-385` (mob spell resolution)

- [ ] **Step 1: Calculate spell difficulty multiplier for player path**

In `internal/hooks/NewRound_DoCombat_helpers.go`, replace the player
spell resolution progression block (lines 277-284):

Replace:
```go
		// Fire progression for the correct skill based on spell school
		if spellData != nil && spellData.HasSchool(spells.SchoolManifestation) {
			user.Character.OnSkillUse(string(skills.Manifestation), userId)
			user.Character.OnStatUse("charisma", userId)
		} else {
			user.Character.OnSkillUse(string(skills.Spellcasting), userId)
			user.Character.OnStatUse("willpower", userId)
		}
```

With:
```go
		// Fire progression for the correct skill based on spell school.
		// Difficulty scaling: harder spells give proportionally more progression.
		spellBonus := 1.0
		if spellData != nil {
			bal := configs.GetBalanceConfig()
			spellBonus = 1.0 + float64(spellData.Difficulty)*float64(bal.SpellDifficultyProgressionScale)

			// Self-cast penalty: HelpSingle targeting only self gets reduced progression
			if spellData.Type == spells.HelpSingle &&
				len(cs.TargetMobInstanceIds) == 0 &&
				len(cs.TargetUserIds) == 1 && cs.TargetUserIds[0] == userId {
				spellBonus *= float64(bal.SelfCastProgressionMultiplier)
			}

			// AoE guard: HarmArea/HarmMulti with no targets hit skips progression
			if (spellData.Type == spells.HarmArea || spellData.Type == spells.HarmMulti) &&
				len(cs.TargetUserIds) == 0 && len(cs.TargetMobInstanceIds) == 0 {
				spellBonus = 0
			}
		}

		if spellBonus > 0 {
			if spellData != nil && spellData.HasSchool(spells.SchoolManifestation) {
				user.Character.OnSkillUseScaled(string(skills.Manifestation), userId, spellBonus)
				user.Character.OnStatUse("charisma", userId)
			} else {
				user.Character.OnSkillUseScaled(string(skills.Spellcasting), userId, spellBonus)
				user.Character.OnStatUse("willpower", userId)
			}
		}
```

- [ ] **Step 2: Calculate spell difficulty multiplier for mob path**

In `internal/hooks/NewRound_DoCombat_helpers.go`, replace the mob spell
resolution progression block (lines 378-385):

Replace:
```go
		// Stage 38.3: Mob spellcasting progression — route to correct skill
		if spellData != nil && spellData.HasSchool(spells.SchoolManifestation) {
			mob.Character.OnSkillUse(string(skills.Manifestation), 0)
			mob.Character.OnStatUse("charisma", 0)
		} else {
			mob.Character.OnSkillUse(string(skills.Spellcasting), 0)
			mob.Character.OnStatUse("willpower", 0)
		}
```

With:
```go
		// Stage 38.3: Mob spellcasting progression — difficulty-scaled
		spellBonus := 1.0
		if spellData != nil {
			bal := configs.GetBalanceConfig()
			spellBonus = 1.0 + float64(spellData.Difficulty)*float64(bal.SpellDifficultyProgressionScale)
		}
		if spellData != nil && spellData.HasSchool(spells.SchoolManifestation) {
			mob.Character.OnSkillUseScaled(string(skills.Manifestation), 0, spellBonus)
			mob.Character.OnStatUse("charisma", 0)
		} else {
			mob.Character.OnSkillUseScaled(string(skills.Spellcasting), 0, spellBonus)
			mob.Character.OnStatUse("willpower", 0)
		}
```

Note: Mobs don't need the self-cast penalty or AoE guard — they cast
tactically via AI, not for grinding.

- [ ] **Step 3: Verify configs import exists**

Check that `internal/hooks/NewRound_DoCombat_helpers.go` already imports
`configs` and `spells` packages. Both should already be imported — verify
by building.

- [ ] **Step 4: Build and verify**

Run: `go build ./...`
Expected: clean build

- [ ] **Step 5: Commit**

```bash
git add internal/hooks/NewRound_DoCombat_helpers.go
git commit -m "feat: wire spell difficulty into progression at resolution

Harder spells give proportionally more skill progression via
SpellDifficultyProgressionScale config. Self-cast HelpSingle spells
get SelfCastProgressionMultiplier penalty. Empty-room HarmArea/
HarmMulti spells skip progression entirely.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: Wire Craft Difficulty Into Progression

**Files:**
- Modify: `internal/hooks/NewRound_UserRoundTick.go:342` (player craft completion)
- Modify: `internal/hooks/NewRound_MobRoundTick.go:354` (mob multi-round craft)
- Modify: `internal/mobcommands/craft.go:51` (mob immediate-complete)

- [ ] **Step 1: Player craft completion — difficulty scaling**

In `internal/hooks/NewRound_UserRoundTick.go`, replace line 342:

Replace:
```go
									user.Character.OnSkillUse(recipe.Skill, user.UserId)
```

With:
```go
									craftBonus := 1.0 + float64(recipe.SkillMinimum)*float64(configs.GetBalanceConfig().CraftDifficultyProgressionScale)
									user.Character.OnSkillUseScaled(recipe.Skill, user.UserId, craftBonus)
```

Verify that `configs` is already imported in this file (it should be —
search for existing `configs.Get` calls).

- [ ] **Step 2: Mob multi-round craft — difficulty scaling**

In `internal/hooks/NewRound_MobRoundTick.go`, replace line 354:

Replace:
```go
							mob.Character.OnSkillUse(recipe.Skill, 0)
```

With:
```go
							craftBonus := 1.0 + float64(recipe.SkillMinimum)*float64(configs.GetBalanceConfig().CraftDifficultyProgressionScale)
							mob.Character.OnSkillUseScaled(recipe.Skill, 0, craftBonus)
```

- [ ] **Step 3: Mob immediate-complete — difficulty scaling**

In `internal/mobcommands/craft.go`, replace line 51:

Replace:
```go
		mob.Character.OnSkillUse(result.SkillName, 0)
```

With:
```go
		craftBonus := 1.0 + float64(result.SkillMinimum)*float64(configs.GetBalanceConfig().CraftDifficultyProgressionScale)
		mob.Character.OnSkillUseScaled(result.SkillName, 0, craftBonus)
```

Add `"github.com/GoMudEngine/GoMud/internal/configs"` to the import block
in `internal/mobcommands/craft.go` if not already present.

- [ ] **Step 4: Build and verify**

Run: `go build ./...`
Expected: clean build

- [ ] **Step 5: Commit**

```bash
git add internal/hooks/NewRound_UserRoundTick.go internal/hooks/NewRound_MobRoundTick.go internal/mobcommands/craft.go
git commit -m "feat: wire recipe skill_minimum into craft progression

Higher skill_minimum recipes give proportionally more skill
progression via CraftDifficultyProgressionScale config knob.
Applied to player crafts, mob multi-round crafts, and mob
immediate-complete crafts. Salvage unchanged.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 6: Final Build + Full Test Suite

**Files:** None (verification only)

- [ ] **Step 1: Full build**

Run: `go build ./...`
Expected: clean build

- [ ] **Step 2: Run all tests**

Run: `go test ./... -count=1 -timeout 120s`
Expected: all PASS

- [ ] **Step 3: Verify config loads correctly**

Run: `go test ./internal/configs/ -v -count=1`
Expected: PASS — new config fields load with defaults

- [ ] **Step 4: Verify progression tests**

Run: `go test ./internal/characters/ -v -run "Progression|SkillUse" -count=1`
Expected: PASS — OnSkillUseScaled tests pass, existing tests unchanged

- [ ] **Step 5: Spot-check spell difficulty values**

Run a quick grep to verify key spells have sensible difficulty values:
```bash
grep -r "difficulty:" _datafiles/world/dogmud/spells/ | sort -t: -k3 -n
```
Expected: range from 0 (utility) to 75+ (hard combat spells). Flag any
combat spells with difficulty 0 that should probably be higher — but do
NOT change them in this task (that's tuning, not implementation).
