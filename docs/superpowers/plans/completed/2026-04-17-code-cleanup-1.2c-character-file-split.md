# Code Cleanup 1.2c: `character.go` File Split — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Split `internal/characters/character.go` (3925 lines, 178 functions) into 10 new themed subject files + 5 extended existing siblings. `character.go` ends at ~400 lines holding only the `Character` struct, constructors, and a handful of core accessors.

**Architecture:** Pure file moves — no renames, no logic changes, no refactoring. Each commit relocates a cohesive cluster of methods from `character.go` into its destination file. Per-commit verification via `go build`, `go vet`, `go test ./internal/characters/...`. End-of-branch server-boot smoke confirms runtime state.

**Tech Stack:** Go 1.25.

**Spec:** `docs/superpowers/specs/2026-04-17-code-cleanup-1.2c-character-file-split-design.md`

**Branch:** `feature/stage-1.2c-character-file-split` off `development`.

**Pre-existing file inventory:**
- `aggro.go` already exists with `AggroType`/`SpellAggroInfo`/`Aggro` type definitions. Task 5 extends it.
- `cooldowns.go`, `charminfo.go`, `formattedname.go`, `worn.go` already exist. Tasks 2, 3, 4, 14 extend them.
- All other destination files (`migrations.go`, `spells.go`, `description.go`, `quests.go`, `buffs.go`, `resources.go`, `skills.go`, `combat.go`, `inventory.go`, `validate.go`) are new.

---

## Per-Task Mechanical Pattern

Every file-move task (Tasks 1–15) follows this shape. The per-task sections below only deviate from it where noted.

1. **Locate the functions** in `internal/characters/character.go` using `grep -n "^func ... <name>"`. Line numbers shift between tasks as earlier commits pull code out, so always grep by name rather than trusting a stale line number from the plan.
2. **If destination is a NEW file:** create it with:
   ```go
   package characters

   import (
   	// imports added here — start empty, let the Go compiler drive additions
   )
   ```
3. **Cut each function** (full declaration + body + any leading doc comment) from `character.go` and paste into the destination file. Preserve original source order when moving multiple functions (aids diff review).
4. **Resolve imports.** Run `go build ./internal/characters/...`. The compiler will flag:
   - Missing imports in the destination → add them.
   - Now-unused imports in `character.go` → remove them.
5. **Verify clean:**
   ```bash
   go build ./...
   go vet ./...
   go test ./internal/characters/...
   ```
   Any failure → fix inline before committing. The known pre-existing `internal/rooms TestRoom_AddTemporaryExit/duplicate_name_rejected` failure is acceptable.
6. **Commit** with the exact subject listed in that task plus the standard trailer:
   ```
   Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
   ```

Per the spec, any opportunistic fix or cleanup is **forbidden**. If you spot a bug, document it and move on.

---

## Task 0: Create feature branch

**Files:** none.

- [ ] **Step 1: Verify you're on `development` and clean.**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git status
git branch --show-current
```

Expected: `development`. The working tree has pre-existing unrelated noise (`.claude/settings.local.json`, `internal/usercommands/_datafiles/feedback/*.txt`, a screenshot PNG). Do not stage or commit any of those in this branch.

- [ ] **Step 2: Create feature branch.**

```bash
git checkout -b feature/stage-1.2c-character-file-split
```

Expected: `Switched to a new branch 'feature/stage-1.2c-character-file-split'`.

---

## Task 1: Extract migrations → `migrations.go` (new)

**Files:**
- Create: `internal/characters/migrations.go`
- Modify: `internal/characters/character.go` (delete moved functions)

**Functions to move (9):**

- `(c *Character) MigratePairedSpells()`
- `(c *Character) MigrateNeckToBack()`
- `(c *Character) MigrateQuestSpells()`
- `(c *Character) MigrateDescriptionWrapping()`
- `(c *Character) MigrateAlchemyPotions()`
- `(c *Character) MigrateAlchemyRecipes()`
- `(c *Character) MigrateQuestFlags()`
- `(c *Character) MigrateLegacyPotions()`
- `(c *Character) MigrateChrysalisAidRemoved()`

- [ ] **Step 1: Confirm functions exist.**

```bash
grep -nE "^func \(c \*Character\) Migrate[A-Z]" internal/characters/character.go
```

Expected: 9 matches. If count differs, stop and investigate.

- [ ] **Step 2: Create `internal/characters/migrations.go`** with only `package characters` on line 1. Imports added via compiler feedback in Step 4.

- [ ] **Step 3: Move the 9 functions** (in source order) from `character.go` to `migrations.go`.

- [ ] **Step 4: Resolve imports.**

```bash
go build ./internal/characters/...
```

Add whatever imports the compiler flags as missing to `migrations.go` (likely `github.com/GoMudEngine/GoMud/internal/items`, possibly `github.com/GoMudEngine/GoMud/internal/buffs`, possibly `github.com/GoMudEngine/GoMud/internal/crafting`, possibly `github.com/GoMudEngine/GoMud/internal/enchantments`, possibly `github.com/GoMudEngine/GoMud/internal/mudlog`). Remove any now-unused imports from `character.go`.

- [ ] **Step 5: Verify clean.**

```bash
go build ./...
go vet ./...
go test ./internal/characters/...
```

- [ ] **Step 6: Commit.**

```bash
git add internal/characters/migrations.go internal/characters/character.go
git commit -m "$(cat <<'EOF'
refactor(characters): extract migrations to migrations.go

Move the 9 Migrate* functions out of character.go into their own file.
Pure file move — no logic changes.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Move cooldown + timer methods → existing `cooldowns.go`

**Files:**
- Modify: `internal/characters/cooldowns.go`
- Modify: `internal/characters/character.go`

**Functions to move (7):**

- `(c *Character) PruneCooldowns()`
- `(c *Character) GetCooldown(trackingTag string) int`
- `(c *Character) GetAllCooldowns() map[string]int`
- `(c *Character) TryCooldown(trackingTag string, cooldownTime string) bool`
- `(c *Character) TimerSet(name, period string)`
- `(c *Character) TimerExpired(name string) bool`
- `(c *Character) TimerExists(name string) bool`

- [ ] **Step 1: Confirm the 7 functions + baseline.**

```bash
grep -nE "^func \(c \*Character\) (PruneCooldowns|GetCooldown|GetAllCooldowns|TryCooldown|TimerSet|TimerExpired|TimerExists)\b" internal/characters/character.go
wc -l internal/characters/cooldowns.go
```

- [ ] **Step 2: Move the 7 functions** to the END of `cooldowns.go`.

- [ ] **Step 3: Resolve imports.**

- [ ] **Step 4: Verify clean.**

```bash
go build ./...
go vet ./...
go test ./internal/characters/...
```

- [ ] **Step 5: Commit.**

```bash
git add internal/characters/cooldowns.go internal/characters/character.go
git commit -m "$(cat <<'EOF'
refactor(characters): move cooldown + timer methods to cooldowns.go

Move 7 methods (PruneCooldowns, GetCooldown, GetAllCooldowns,
TryCooldown, TimerSet, TimerExpired, TimerExists) from character.go
into their subject file. Pure file move.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Move charm methods → existing `charminfo.go`

**Files:**
- Modify: `internal/characters/charminfo.go`
- Modify: `internal/characters/character.go`

**Functions to move (7):**

- `(c *Character) TrackCharmed(mobId int, add bool)`
- `(c *Character) GetCharmIds() []int`
- `(c *Character) Charm(userId int, rounds int, expireCommand string)`
- `(c *Character) GetCharmedUserId() int`
- `(c *Character) IsCharmed(userId ...int) bool`
- `(c *Character) RemoveCharm() int`
- `(c *Character) GetMaxCharmedCreatures() int`

- [ ] **Step 1: Confirm the 7 functions.**

```bash
grep -nE "^func \(c \*Character\) (TrackCharmed|GetCharmIds|Charm|GetCharmedUserId|IsCharmed|RemoveCharm|GetMaxCharmedCreatures)\b" internal/characters/character.go
```

Expected: 7 matches.

- [ ] **Step 2: Move the 7 functions** to the end of `charminfo.go`.

- [ ] **Step 3: Resolve imports.**

- [ ] **Step 4: Verify clean.**

```bash
go build ./...
go vet ./...
go test ./internal/characters/...
```

- [ ] **Step 5: Commit.**

```bash
git add internal/characters/charminfo.go internal/characters/character.go
git commit -m "$(cat <<'EOF'
refactor(characters): move charm methods to charminfo.go

Move 7 methods (TrackCharmed, GetCharmIds, Charm, GetCharmedUserId,
IsCharmed, RemoveCharm, GetMaxCharmedCreatures) from character.go
into their subject file. Pure file move.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Move name methods → existing `formattedname.go`

**Files:**
- Modify: `internal/characters/formattedname.go`
- Modify: `internal/characters/character.go`

**Functions to move (5):**

- `(c *Character) GetMobName(viewingUserId int, renderFlags ...NameRenderFlag) FormattedName`
- `(c *Character) GetMobNameIndexed(viewingUserId int, dupIndex int, renderFlags ...NameRenderFlag) FormattedName`
- `(c *Character) GetPlayerName(viewingUserId int, renderFlags ...NameRenderFlag) FormattedName`
- `(c *Character) GetCharacterName(ansi bool) string`
- `(c *Character) getFormattedName(viewingUserId int, uType string, renderFlags ...NameRenderFlag) FormattedName`

- [ ] **Step 1: Confirm the 5 functions.**

```bash
grep -nE "^func \(c \*Character\) (GetMobName|GetMobNameIndexed|GetPlayerName|GetCharacterName|getFormattedName)\b" internal/characters/character.go
```

Expected: 5 matches.

- [ ] **Step 2: Move the 5 functions** to the end of `formattedname.go`.

- [ ] **Step 3: Resolve imports.**

- [ ] **Step 4: Verify clean.**

```bash
go build ./...
go vet ./...
go test ./internal/characters/...
```

- [ ] **Step 5: Commit.**

```bash
git add internal/characters/formattedname.go internal/characters/character.go
git commit -m "$(cat <<'EOF'
refactor(characters): move name methods to formattedname.go

Move 5 methods (GetMobName, GetMobNameIndexed, GetPlayerName,
GetCharacterName, getFormattedName) from character.go into their
subject file. Pure file move.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Move aggro methods → existing `aggro.go`

**Files:**
- Modify: `internal/characters/aggro.go`
- Modify: `internal/characters/character.go`

**Context:** `aggro.go` already exists holding `AggroType`, `SpellAggroInfo`, and `Aggro` type definitions. We're extending it with the Character methods.

**Functions to move (6):**

- `(c *Character) SetAggroRemote(exitName string, userId int, mobInstanceId int, aggroType AggroType, roundsWaitTime ...int)`
- `(c *Character) SetAggro(userId int, mobInstanceId int, aggroType AggroType, roundsWaitTime ...int)`
- `(c *Character) EndAggro()`
- `(c *Character) ClearGrappleState()`
- `(c *Character) IsAggro(targetUserId int, targetMobInstanceId int) bool`
- `(c *Character) TrackPlayerDamage(userId int, damageAmt int)`

- [ ] **Step 1: Confirm the 6 functions.**

```bash
grep -nE "^func \(c \*Character\) (SetAggroRemote|SetAggro|EndAggro|ClearGrappleState|IsAggro|TrackPlayerDamage)\b" internal/characters/character.go
```

Expected: 6 matches.

- [ ] **Step 2: Move the 6 functions** to the end of `aggro.go`.

- [ ] **Step 3: Resolve imports** in both files.

- [ ] **Step 4: Verify clean.**

```bash
go build ./...
go vet ./...
go test ./internal/characters/...
```

- [ ] **Step 5: Commit.**

```bash
git add internal/characters/aggro.go internal/characters/character.go
git commit -m "$(cat <<'EOF'
refactor(characters): move aggro methods to aggro.go

Move 6 methods (SetAggroRemote, SetAggro, EndAggro, ClearGrappleState,
IsAggro, TrackPlayerDamage) from character.go into the existing
aggro.go (which previously held only the AggroType + Aggro type
definitions). Pure file move.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Extract spells → `spells.go` (new)

**Files:**
- Create: `internal/characters/spells.go`
- Modify: `internal/characters/character.go`

**Functions to move (12):**

- `(c *Character) GetSpells() map[string]int`
- `(c *Character) IsCasting() bool`
- `(c *Character) IsCrafting() bool`
- `(c *Character) HasSpell(spellName string) bool`
- `(c *Character) DisableSpell(spellName string) bool`
- `(c *Character) EnableSpell(spellName string) bool`
- `(c *Character) TrackSpellCast(spellName string) bool`
- `(c *Character) LearnSpell(spellName string) bool`
- `(c *Character) GetBaseCastSuccessChance(spellId string) int`
- `(c *Character) SetCast(roundsWaitTime int, sInfo SpellAggroInfo)`
- `(c *Character) HasRecipe(recipeId string) bool`
- `(c *Character) LearnRecipe(recipeId string) bool`

- [ ] **Step 1: Confirm the 12 functions.**

```bash
grep -nE "^func \(c \*Character\) (GetSpells|IsCasting|IsCrafting|HasSpell|DisableSpell|EnableSpell|TrackSpellCast|LearnSpell|GetBaseCastSuccessChance|SetCast|HasRecipe|LearnRecipe)\b" internal/characters/character.go
```

Expected: 12 matches.

- [ ] **Step 2: Create `internal/characters/spells.go`** with `package characters` header.

- [ ] **Step 3: Move the 12 functions** in source order.

- [ ] **Step 4: Resolve imports.**

Likely: `github.com/GoMudEngine/GoMud/internal/spells` (possibly — check whether methods actually reference the package). `SetCast` references `SpellAggroInfo` which lives in `aggro.go` (same package, no import needed).

- [ ] **Step 5: Verify clean.**

```bash
go build ./...
go vet ./...
go test ./internal/characters/...
```

- [ ] **Step 6: Commit.**

```bash
git add internal/characters/spells.go internal/characters/character.go
git commit -m "$(cat <<'EOF'
refactor(characters): extract spells to spells.go

Move 12 methods (GetSpells, IsCasting, IsCrafting, HasSpell,
DisableSpell, EnableSpell, TrackSpellCast, LearnSpell,
GetBaseCastSuccessChance, SetCast, HasRecipe, LearnRecipe) out of
character.go. Grouping recipes with spells because both are
"learned abilities" the Character accumulates. Pure file move.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Extract description → `description.go` (new)

**Files:**
- Create: `internal/characters/description.go`
- Modify: `internal/characters/character.go`

**Functions to move (9):**

- `(c *Character) GetDescription() string`
- `(c *Character) GetMutationVisuals() string`
- `(c *Character) GetHealthAppearance() string`
- `(c *Character) CacheDescription()`
- `(c *Character) HasAdjective(adj string) bool`
- `(c *Character) SetAdjective(adj string, addToList bool)`
- `(c *Character) GetAdjectives() []string`
- `(c *Character) Species() string`
- `(c *Character) BarterPrice(startPrice int) int`

- [ ] **Step 1: Confirm the 9 functions.**

```bash
grep -nE "^func \(c \*Character\) (GetDescription|GetMutationVisuals|GetHealthAppearance|CacheDescription|HasAdjective|SetAdjective|GetAdjectives|Species|BarterPrice)\b" internal/characters/character.go
```

Expected: 9 matches.

- [ ] **Step 2: Create `internal/characters/description.go`** with `package characters` header.

- [ ] **Step 3: Move the 9 functions.**

- [ ] **Step 4: Resolve imports.**

- [ ] **Step 5: Verify clean.**

```bash
go build ./...
go vet ./...
go test ./internal/characters/...
```

- [ ] **Step 6: Commit.**

```bash
git add internal/characters/description.go internal/characters/character.go
git commit -m "$(cat <<'EOF'
refactor(characters): extract description to description.go

Move 9 methods (GetDescription, GetMutationVisuals, GetHealthAppearance,
CacheDescription, HasAdjective, SetAdjective, GetAdjectives, Species,
BarterPrice) out of character.go. Pure file move.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: Extract quests → `quests.go` (new)

**Files:**
- Create: `internal/characters/quests.go`
- Modify: `internal/characters/character.go`

**Functions to move (12):**

- `(c *Character) IsQuestDone(questToken string) bool`
- `(c *Character) HasQuest(questToken string) bool`
- `(c *Character) GetQuestProgress() map[int]string`
- `(c *Character) GiveQuestToken(questToken string) bool`
- `(c *Character) ClearQuestToken(questToken string)`
- `(c *Character) SetQuestFlag(key, value string)`
- `(c *Character) GetQuestFlag(key string) string`
- `(c *Character) HasQuestFlag(key string) bool`
- `(c *Character) ClearQuestFlag(key string)`
- `(c *Character) RememberRoom(roomId int)`
- `(c *Character) GetMemoryCapacity() int`
- `(c *Character) GetMapSprawlCapacity() int`

- [ ] **Step 1: Confirm the 12 functions.**

```bash
grep -nE "^func \(c \*Character\) (IsQuestDone|HasQuest|GetQuestProgress|GiveQuestToken|ClearQuestToken|SetQuestFlag|GetQuestFlag|HasQuestFlag|ClearQuestFlag|RememberRoom|GetMemoryCapacity|GetMapSprawlCapacity)\b" internal/characters/character.go
```

Expected: 12 matches.

- [ ] **Step 2: Create `internal/characters/quests.go`** with `package characters` header.

- [ ] **Step 3: Move the 12 functions.**

- [ ] **Step 4: Resolve imports.**

- [ ] **Step 5: Verify clean.**

```bash
go build ./...
go vet ./...
go test ./internal/characters/...
```

- [ ] **Step 6: Commit.**

```bash
git add internal/characters/quests.go internal/characters/character.go
git commit -m "$(cat <<'EOF'
refactor(characters): extract quests to quests.go

Move 12 methods (IsQuestDone, HasQuest, GetQuestProgress,
GiveQuestToken, ClearQuestToken, SetQuestFlag, GetQuestFlag,
HasQuestFlag, ClearQuestFlag, RememberRoom, GetMemoryCapacity,
GetMapSprawlCapacity) out of character.go. Pure file move.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 9: Extract buffs → `buffs.go` (new)

**Files:**
- Create: `internal/characters/buffs.go`
- Modify: `internal/characters/character.go`

**Functions to move (13):**

- `(c *Character) HasBuffFlag(buffFlag buffs.Flag) bool`
- `(c *Character) HasFlagFromAnySource(buffFlag buffs.Flag) bool`
- `(c *Character) CancelBuffsWithFlag(buffFlag buffs.Flag) bool`
- `(c *Character) HasBuff(buffId int) bool`
- `(c *Character) AddBuff(buffId int, isPermanent bool) error`
- `(c *Character) AddBuffScaled(buffId int, durationMult float64) error`
- `(c *Character) TrackBuffStarted(buffId int)`
- `(c *Character) GetBuffs(buffId ...int) []*buffs.Buff`
- `(c *Character) RemoveBuff(buffId int)`
- `(c *Character) SetPermaBuffs(buffIds []int)`
- `(c *Character) RemovePermaBuff(buffId int)`
- `(c *Character) reapplyPermabuffs(removedItems ...items.Item)`
- `(c *Character) IsDisabled() bool`

- [ ] **Step 1: Confirm the 13 functions.**

```bash
grep -nE "^func \(c \*Character\) (HasBuffFlag|HasFlagFromAnySource|CancelBuffsWithFlag|HasBuff|AddBuff|AddBuffScaled|TrackBuffStarted|GetBuffs|RemoveBuff|SetPermaBuffs|RemovePermaBuff|reapplyPermabuffs|IsDisabled)\b" internal/characters/character.go
```

Expected: 13 matches.

- [ ] **Step 2: Create `internal/characters/buffs.go`** with `package characters` header.

- [ ] **Step 3: Move the 13 functions.**

- [ ] **Step 4: Resolve imports.**

Likely: `github.com/GoMudEngine/GoMud/internal/buffs`, `github.com/GoMudEngine/GoMud/internal/items`.

- [ ] **Step 5: Verify clean.**

```bash
go build ./...
go vet ./...
go test ./internal/characters/...
```

- [ ] **Step 6: Commit.**

```bash
git add internal/characters/buffs.go internal/characters/character.go
git commit -m "$(cat <<'EOF'
refactor(characters): extract buffs to buffs.go

Move 13 methods (HasBuffFlag, HasFlagFromAnySource, CancelBuffsWithFlag,
HasBuff, AddBuff, AddBuffScaled, TrackBuffStarted, GetBuffs, RemoveBuff,
SetPermaBuffs, RemovePermaBuff, reapplyPermabuffs, IsDisabled) out of
character.go. Pure file move.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 10: Extract resources → `resources.go` (new)

**Files:**
- Create: `internal/characters/resources.go`
- Modify: `internal/characters/character.go`

**Functions to move (16):**

- `(c *Character) DeductActionPoints(amount int) bool`
- `(c *Character) DeductStamina(amount int) bool`
- `(c *Character) GetMovementStaminaCost(terrainMultiplier float64) int`
- `(c *Character) GetAttackStaminaCost() int`
- `(c *Character) DeductAttackStamina() int`
- `(c *Character) MovementCost() int`
- `(c *Character) HealthPerRound() int`
- `(c *Character) StaminaPerRound() int`
- `(c *Character) ConvictionPerRound() int`
- `(c *Character) ApplyHealthChange(healthChange int) int`
- `(c *Character) Heal(hp int) int`
- `(c *Character) GetToxicityMax() float64`
- `(c *Character) AddToxicity(amount float64) bool`
- `(c *Character) GetToxicityPenalties() (float64, float64, float64)`
- `(c *Character) GetDefenseStaminaCost(defenseType string) int`
- `(c *Character) DeductDefenseStamina(defenseType string) bool`

- [ ] **Step 1: Confirm the 16 functions.**

```bash
grep -nE "^func \(c \*Character\) (DeductActionPoints|DeductStamina|GetMovementStaminaCost|GetAttackStaminaCost|DeductAttackStamina|MovementCost|HealthPerRound|StaminaPerRound|ConvictionPerRound|ApplyHealthChange|Heal|GetToxicityMax|AddToxicity|GetToxicityPenalties|GetDefenseStaminaCost|DeductDefenseStamina)\b" internal/characters/character.go
```

Expected: 16 matches.

- [ ] **Step 2: Create `internal/characters/resources.go`** with `package characters` header.

- [ ] **Step 3: Move the 16 functions.**

- [ ] **Step 4: Resolve imports.**

Likely: `github.com/GoMudEngine/GoMud/internal/configs`, `github.com/GoMudEngine/GoMud/internal/stats`, possibly `math`.

- [ ] **Step 5: Verify clean.**

```bash
go build ./...
go vet ./...
go test ./internal/characters/...
```

- [ ] **Step 6: Commit.**

```bash
git add internal/characters/resources.go internal/characters/character.go
git commit -m "$(cat <<'EOF'
refactor(characters): extract resources to resources.go

Move 16 methods covering HP/Stamina/Conviction/ActionPoints pools +
per-round regen + toxicity out of character.go. Pure file move.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 11: Extract skills → `skills.go` (new)

**Files:**
- Create: `internal/characters/skills.go`
- Modify: `internal/characters/character.go`

**Functions to move (15):**

- `(c *Character) GetSkills() map[string]int`
- `(c *Character) SetSkill(skillName string, level int)`
- `(c *Character) TrainSkill(skillName string, targetLevel ...int) int`
- `(c *Character) GetSkillLevel(skillName skills.SkillTag) int`
- `(c *Character) GetSkillLevelCost(currentLevel int) int`
- `(c *Character) IncreaseSkill(skillName string) bool`
- `(c *Character) GetAllSkillRanks() map[string]int`
- `(c *Character) GetTotalSkillRanks() int`
- `(c *Character) GetCombatSkillTag() skills.SkillTag`
- `(c *Character) GetCombatSkillLevel() int`
- `(c *Character) GetModifiedAttackCount(baseAttacks int, weaponSpeed float64, isOffhand bool) int`
- `(c *Character) IncreaseStat(statName string, amount int) bool`
- `(c *Character) GetStatValue(statName string) int`
- `(c *Character) AttemptRecovery(statValue int) (bool, bool)`
- `(c *Character) KnowsFirstAid() bool`

- [ ] **Step 1: Confirm the 15 functions.**

```bash
grep -nE "^func \(c \*Character\) (GetSkills|SetSkill|TrainSkill|GetSkillLevel|GetSkillLevelCost|IncreaseSkill|GetAllSkillRanks|GetTotalSkillRanks|GetCombatSkillTag|GetCombatSkillLevel|GetModifiedAttackCount|IncreaseStat|GetStatValue|AttemptRecovery|KnowsFirstAid)\b" internal/characters/character.go
```

Expected: 15 matches.

- [ ] **Step 2: Create `internal/characters/skills.go`** with `package characters` header.

- [ ] **Step 3: Move the 15 functions.**

- [ ] **Step 4: Resolve imports.**

Likely: `github.com/GoMudEngine/GoMud/internal/skills`, `github.com/GoMudEngine/GoMud/internal/stats`, `github.com/GoMudEngine/GoMud/internal/configs`, `math`.

- [ ] **Step 5: Verify clean.**

```bash
go build ./...
go vet ./...
go test ./internal/characters/...
```

- [ ] **Step 6: Commit.**

```bash
git add internal/characters/skills.go internal/characters/character.go
git commit -m "$(cat <<'EOF'
refactor(characters): extract skills to skills.go

Move 15 methods (skill getters/setters/training + combat skill tag +
stat value + recovery + first-aid knowledge check) out of character.go.
Pure file move.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 12: Extract combat → `combat.go` (new)

**Files:**
- Create: `internal/characters/combat.go`
- Modify: `internal/characters/character.go`

**Functions to move (9):**

- `(c *Character) GetDefense() int`
- `(c *Character) GetPhysicalMitigation() float64`
- `(c *Character) GetMagicalMitigation() float64`
- `(c *Character) GetConvictionMitigation() float64`
- `(c *Character) GetDefenseSequence() []string`
- `(c *Character) GetDefenseScore(defenseType string) float64`
- `(c *Character) GetDefaultDiceRoll() (attacks int, dCount int, dSides int, bonus int, buffOnCrit []int)`
- `(c *Character) GetDefaultDistributionDamage() (attacks int, baseDamage float64, variance float64, buffOnCrit []int)`
- `(c *Character) CalculateUnarmedDamage() (baseDamage float64, variance float64)`

- [ ] **Step 1: Confirm the 9 functions.**

```bash
grep -nE "^func \(c \*Character\) (GetDefense|GetPhysicalMitigation|GetMagicalMitigation|GetConvictionMitigation|GetDefenseSequence|GetDefenseScore|GetDefaultDiceRoll|GetDefaultDistributionDamage|CalculateUnarmedDamage)\b" internal/characters/character.go
```

Expected: 9 matches.

- [ ] **Step 2: Create `internal/characters/combat.go`** with `package characters` header.

- [ ] **Step 3: Move the 9 functions.**

- [ ] **Step 4: Resolve imports.**

- [ ] **Step 5: Verify clean.**

```bash
go build ./...
go vet ./...
go test ./internal/characters/...
```

- [ ] **Step 6: Commit.**

```bash
git add internal/characters/combat.go internal/characters/character.go
git commit -m "$(cat <<'EOF'
refactor(characters): extract combat to combat.go

Move 9 combat-derived calculation methods out of character.go:
defense + mitigation + defense sequence/score + damage dice getters +
unarmed damage. Pure file move.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 13: Extract inventory → `inventory.go` (new)

Largest single extract. Take care.

**Files:**
- Create: `internal/characters/inventory.go`
- Modify: `internal/characters/character.go`

**Functions to move (22):**

- `(c *Character) CarryCapacity() float64`
- `(c *Character) GetCarriedWeight() float64`
- `(c *Character) StoreItem(i items.Item) bool`
- `(c *Character) RemoveItem(i items.Item) bool`
- `(c *Character) UpdateItem(originalItm items.Item, replacement items.Item) bool`
- `(c *Character) UseItem(i items.Item) int`
- `(c *Character) FindInPotions(itemName string) (items.Item, bool)`
- `(c *Character) UseItemFromPotions(i items.Item) int`
- `(c *Character) FindInComponents(itemName string) (items.Item, bool)`
- `(c *Character) FindInBackpack(itemName string) (items.Item, bool)`
- `(c *Character) FindOnBody(itemName string) (items.Item, bool)`
- `(c *Character) FindItem(itemName string) (items.Item, string, bool)`
- `(c *Character) GetAllBackpackItems() []items.Item`
- `(c *Character) GetAllCarriedItems() []items.Item`
- `(c *Character) GetRandomItem() (items.Item, bool)`
- `(c *Character) SortComponentItems() int`
- `(c *Character) SortPotionItems() int`
- `(c *Character) FindKeyInBackpack(lockId string) (items.Item, bool)`
- `(c *Character) HasKey(lockId string, difficulty int) (hasKey bool, hasSequence bool)`
- `(c *Character) KeyCount() int`
- `(c *Character) GetKey(lockId string) string`
- `(c *Character) SetKey(lockId string, sequence string)`

**Note:** `HandsRequired` does NOT move here — it moves to `worn.go` in Task 14 since it's an equipment-reasoning helper.

- [ ] **Step 1: Confirm the 22 functions.**

```bash
grep -nE "^func \(c \*Character\) (CarryCapacity|GetCarriedWeight|StoreItem|RemoveItem|UpdateItem|UseItem|FindInPotions|UseItemFromPotions|FindInComponents|FindInBackpack|FindOnBody|FindItem|GetAllBackpackItems|GetAllCarriedItems|GetRandomItem|SortComponentItems|SortPotionItems|FindKeyInBackpack|HasKey|KeyCount|GetKey|SetKey)\b" internal/characters/character.go
```

Expected: 22 matches.

- [ ] **Step 2: Create `internal/characters/inventory.go`** with `package characters` header.

- [ ] **Step 3: Move the 22 functions** in source order.

- [ ] **Step 4: Resolve imports.**

Likely: `github.com/GoMudEngine/GoMud/internal/items`, `github.com/GoMudEngine/GoMud/internal/configs`, `strings`, possibly `github.com/GoMudEngine/GoMud/internal/util`.

- [ ] **Step 5: Verify clean.**

```bash
go build ./...
go vet ./...
go test ./internal/characters/...
```

- [ ] **Step 6: Commit.**

```bash
git add internal/characters/inventory.go internal/characters/character.go
git commit -m "$(cat <<'EOF'
refactor(characters): extract inventory to inventory.go

Move 22 methods covering carry capacity, backpack storage + retrieval,
item use, potion/component/backpack/on-body lookup, random-item pull,
sort helpers, and key-ring management out of character.go.
Pure file move.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 14: Move equipment methods → existing `worn.go`

**Files:**
- Modify: `internal/characters/worn.go`
- Modify: `internal/characters/character.go`

**Functions to move (12):**

- `(c *Character) Wear(i items.Item) (returnItems []items.Item, newItemWorn bool, failureReason string)`
- `(c *Character) wearWeaponOrShield(i items.Item, spec items.ItemSpec, iHandsRequired int, canDualWield bool) (returnItems []items.Item, newItemWorn bool, failureReason string)`
- `(c *Character) wearArmorSlot(i items.Item, spec items.ItemSpec) (returnItems []items.Item, newItemWorn bool, failureReason string)`
- `(c *Character) RemoveFromBody(i items.Item) bool`
- `(c *Character) Uncurse() []items.Item`
- `(c *Character) HasShield() bool`
- `(c *Character) IsDualWielding() bool`
- `(c *Character) IsUnarmed() bool`
- `(c *Character) IsUnarmedStyle() bool`
- `(c *Character) GetAllWornItems() []items.Item`
- `(c *Character) HandsRequired(i items.Item) int`
- `(c *Character) GetGearValue() int`

- [ ] **Step 1: Confirm the 12 functions in character.go + baseline for worn.go.**

```bash
grep -nE "^func \(c \*Character\) (Wear|wearWeaponOrShield|wearArmorSlot|RemoveFromBody|Uncurse|HasShield|IsDualWielding|IsUnarmed|IsUnarmedStyle|GetAllWornItems|HandsRequired|GetGearValue)\b" internal/characters/character.go
wc -l internal/characters/worn.go
```

Expected: 12 matches; baseline ~420 lines on `worn.go`.

- [ ] **Step 2: Move the 12 functions** to the end of `worn.go`.

- [ ] **Step 3: Resolve imports** in both `worn.go` (likely no additions needed — it already imports `items`) and `character.go` (remove unused).

- [ ] **Step 4: Verify clean.**

```bash
go build ./...
go vet ./...
go test ./internal/characters/...
```

- [ ] **Step 5: Commit.**

```bash
git add internal/characters/worn.go internal/characters/character.go
git commit -m "$(cat <<'EOF'
refactor(characters): move equipment methods to worn.go

Move 12 methods (Wear + 2 wear helpers from 1.2b, RemoveFromBody,
Uncurse, HasShield, IsDualWielding, IsUnarmed, IsUnarmedStyle,
GetAllWornItems, HandsRequired, GetGearValue) from character.go into
their subject file. Pure file move.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 15: Extract validate → `validate.go` (new)

Biggest logical cluster. Depends on the validate helpers extracted in 1.2b + the core RecalculateStats pipeline.

**Files:**
- Create: `internal/characters/validate.go`
- Modify: `internal/characters/character.go`

**Functions to move (10):**

- `(c *Character) Validate(recalcPermaBuffs ...bool) error`
- `(c *Character) validateSkillMigrations()`
- `(c *Character) validatePoolClamps()`
- `(c *Character) validateEquipmentItems()`
- `(c *Character) validateDisabledSlotsForSpecies()`
- `(c *Character) validateMutationSlots()`
- `(c *Character) RecalculateStats()`
- `(c *Character) GetPoolReservation(pool string, poolMax int) int`
- `(c *Character) CanDualWield() bool`
- `CombatSkillTagForItem(weapon items.Item) skills.SkillTag` (package-level, NOT a method)

**Note:** `StatMod` stays in `character.go` (it's a low-level helper used by many callers across the package).

- [ ] **Step 1: Confirm the 10 functions.**

```bash
grep -nE "^func \(c \*Character\) (Validate|validateSkillMigrations|validatePoolClamps|validateEquipmentItems|validateDisabledSlotsForSpecies|validateMutationSlots|RecalculateStats|GetPoolReservation|CanDualWield)\b" internal/characters/character.go
grep -nE "^func CombatSkillTagForItem\b" internal/characters/character.go
```

Expected: 9 method matches + 1 package-level function match.

- [ ] **Step 2: Create `internal/characters/validate.go`** with `package characters` header.

- [ ] **Step 3: Move the 10 functions.**

Preserve source order for readability.

- [ ] **Step 4: Resolve imports.**

Likely: `github.com/GoMudEngine/GoMud/internal/buffs`, `github.com/GoMudEngine/GoMud/internal/configs`, `github.com/GoMudEngine/GoMud/internal/crafting`, `github.com/GoMudEngine/GoMud/internal/items`, `github.com/GoMudEngine/GoMud/internal/mudlog`, `github.com/GoMudEngine/GoMud/internal/mutations`, `github.com/GoMudEngine/GoMud/internal/skills`, `github.com/GoMudEngine/GoMud/internal/species`, `github.com/GoMudEngine/GoMud/internal/stats`, `github.com/GoMudEngine/GoMud/internal/statmods`, `github.com/GoMudEngine/GoMud/internal/events`, `math`, `time`.

Let the compiler drive the exact list.

- [ ] **Step 5: Verify clean.**

```bash
go build ./...
go vet ./...
go test ./internal/characters/...
```

Expected: ALL 13 of the 18 Task-1-from-1.2b characterization tests still PASS. If any fail, the move changed a semantic (e.g., a receiver variable name collision). Revert and try again.

- [ ] **Step 6: Commit.**

```bash
git add internal/characters/validate.go internal/characters/character.go
git commit -m "$(cat <<'EOF'
refactor(characters): extract validate to validate.go

Move 10 functions (Validate + 5 validate* helpers + RecalculateStats +
GetPoolReservation + CanDualWield + CombatSkillTagForItem) out of
character.go. StatMod stays in character.go as a shared low-level
helper. Pure file move.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 16: End-of-branch verification + mark complete

**Files:**
- Modify: `docs/superpowers/code_cleanup_stage_1_overview.md`

- [ ] **Step 1: Verify the `character.go` size target.**

```bash
wc -l internal/characters/character.go
```

Expected: ≤500 lines (target ~400). If >600, investigate before proceeding: what's still in there that should have moved? Make one more extraction commit if needed, then re-run this step.

- [ ] **Step 2: Run full-project tests.**

```bash
go test ./...
```

Expected: PASS. The known pre-existing `internal/rooms TestRoom_AddTemporaryExit/duplicate_name_rejected` failure is acceptable; everything else must be green.

- [ ] **Step 3: Boot the server.**

```bash
go run .
```

Wait for the "ready" / "listening" log line. Watch for any panic or init error. If anything panics, stop and fix before marking complete.

- [ ] **Step 4: Connect and run `look`.**

In another terminal (or via `telnet localhost <port>`):
- Log in as any character.
- Run `look`.
- Confirm a room description renders.

Ctrl+C the server after the smoke succeeds.

- [ ] **Step 5: Flip status in the overview.**

Edit `docs/superpowers/code_cleanup_stage_1_overview.md`. Find:

```
| 1.2c | `character.go` File Split | 3h | Low | Planning |
```

Change `Planning` to `Complete`.

- [ ] **Step 6: Commit.**

```bash
git add docs/superpowers/code_cleanup_stage_1_overview.md
git commit -m "$(cat <<'EOF'
docs: mark code cleanup 1.2c complete

character.go split from 3925 lines into themed subject files across
15 pure-move commits. No behavior change.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

- [ ] **Step 7: Confirm branch state.**

```bash
git log --oneline feature/stage-1.2c-character-file-split ^development
```

Expected: 16 commits, in the order of Tasks 1–16.

---

## Merge to development

Only after all 16 commits pass and the user has reviewed the branch.

```bash
git checkout development
git merge --no-ff feature/stage-1.2c-character-file-split -m "$(cat <<'EOF'
merge: stage 1.2c character.go file split

Split the 3925-line character.go into 10 new themed subject files
(migrations, spells, description, quests, buffs, resources, skills,
combat, inventory, validate) plus 5 extended existing siblings
(cooldowns, charminfo, formattedname, aggro, worn). character.go
shrunk to ~400 lines, holding only the struct definition,
constructors, and core ID/misc-data accessors.

Pure file moves — zero behavior change.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

Do NOT push to prod yet — bundle with 1.2a (combat + spell god-functions) or push solo once confidence is high.
