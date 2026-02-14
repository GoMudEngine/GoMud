# DOGMud Development Plan - Core Mechanics Conversion

## Overview

This plan breaks down the conversion of GoMud to DOGMud into small, testable increments. Each stage:
- Changes < 1000 lines of code
- Touches < 50 files
- Leaves the MUD in a working (if limited) state
- Includes manual and automated testing
- Can be committed independently

**Strategy**: Start with low-risk changes (stats, species names) and progressively build toward high-risk changes (combat, progression system).

---

## Phase 1: Foundation - Stat System Refactor ✅ COMPLETED

### Stage 1.1: Rename Stats (Minimal Risk) ✅ COMPLETED
**Goal**: Rename existing 6 stats to DOG's stat names without changing functionality.

**Changes**:
| Current | New DOG Name |
|---------|--------------|
| Strength | Strength (no change) |
| Speed | Dexterity |
| Smarts | Perception (swap with old Perception) |
| Vitality | Vitality (no change) |
| Mysticism | Willpower |
| Perception | Charisma |

**Files to Modify** (~20 files, ~300 lines):
1. `internal/stats/stats.go` - Rename struct fields and constants
2. `internal/characters/character.go` - Update references
3. `internal/combat/calculations.go` - Update combat formulas
4. `internal/usercommands/train.go` - Update training references
5. `_datafiles/config.yaml` - Update any stat references
6. All test files referencing stats

**Testing**:
- [ ] **Unit Tests**: Update `stats_test.go` to verify new names
- [ ] **Unit Tests**: Update `character_test.go` stat references
- [ ] **Manual Test**: Create new character, verify stats display correctly
- [ ] **Manual Test**: Level up, train stats, verify names show correctly
- [ ] **Manual Test**: Combat, verify damage calculations still work
- [ ] **Integration Test**: Full character creation → combat → level up cycle

**Acceptance Criteria**:
- All tests pass
- MUD starts and runs without errors
- Characters can be created with new stat names
- Combat works identically to before
- Stats can be trained with new names

**Estimated Changes**: ~300 lines, 20 files

---

### Stage 1.2: Add Secondary Stat Pools ✅ COMPLETED
**Goal**: Add Stamina and Conviction as secondary stats alongside existing Health/Mana.

**Changes**:
1. Add `Stamina`, `StaminaMax` to Character struct (temporary: calculated from Vitality)
2. Add `Conviction`, `ConvictionMax` to Character struct (temporary: calculated from Willpower + Charisma)
3. Keep existing `Health`, `Mana` (will rename later)
4. Add regeneration logic for Stamina/Conviction

**Files to Modify** (~15 files, ~400 lines):
1. `internal/characters/character.go` - Add new fields
2. `internal/stats/stats.go` - Add calculation functions
3. `internal/hooks/NewRound_AutoHeal.go` - Add stamina/conviction regen
4. `internal/usercommands/score.go` - Display new stats
5. Character YAML persistence - Add new fields
6. Test files

**Testing**:
- [ ] **Unit Tests**: `character_test.go` - Verify Stamina/Conviction calculations
- [ ] **Unit Tests**: Test regeneration formulas
- [ ] **Manual Test**: Create character, verify Stamina/Conviction appear in score
- [ ] **Manual Test**: Move around, verify Stamina doesn't change yet (not hooked to movement)
- [ ] **Manual Test**: Rest, verify Stamina/Conviction regenerate
- [ ] **Integration Test**: Character lifecycle with new stats

**Acceptance Criteria**:
- Stamina and Conviction display in score
- Values calculate correctly from primary stats
- Regeneration works
- Existing Health/Mana still functional
- All tests pass

**Estimated Changes**: ~400 lines, 15 files

---

### Stage 1.3: Remove Mana, Rename Health ✅ COMPLETED
**Goal**: Remove Mana system entirely. Health stays as-is (will work with Stamina/Conviction as the 3 pools).

**Changes**:
1. Remove all `Mana`, `ManaMax`, `ManaRecovery` references
2. Remove mana costs from spells (temporary - will add Conviction costs later)
3. Update all spell casting to not consume mana
4. Remove mana potions or convert to health potions

**Files to Modify** (~25 files, ~500 lines):
1. `internal/characters/character.go` - Remove mana fields
2. `internal/spells/spells.go` - Remove mana cost checks
3. `internal/items/itemspec.go` - Update potion types
4. `internal/usercommands/cast.go` - Remove mana deduction
5. `internal/hooks/NewRound_AutoHeal.go` - Remove mana regen
6. All spell YAML files - Remove mana costs (or set to 0)
7. Test files

**Testing**:
- [ ] **Unit Tests**: Verify no mana references remain
- [ ] **Manual Test**: Cast spells, verify they work without mana
- [ ] **Manual Test**: Drink potions, verify they still work
- [ ] **Manual Test**: Character creation/save/load without mana fields
- [ ] **Integration Test**: Full spell casting cycle
- [ ] **Regression Test**: Ensure combat/health still work

**Acceptance Criteria**:
- No mana references in code
- Spells cast successfully
- Character save/load works
- All tests pass
- MUD runs without errors

**Estimated Changes**: ~500 lines, 25 files

---

## Phase 2: Species System Refactor ✅ COMPLETED

### Stage 2.1: Rename Race to Species ✅ COMPLETED
**Goal**: Rename "Race" to "Species" throughout codebase (for clarity - players are human, NPCs/animals have species).

**Changes**:
1. Rename `internal/races/` directory to `internal/species/`
2. Rename `Race` struct to `Species`
3. Rename `RaceId` to `SpeciesId` throughout
4. Update all references in code, configs, YAML files
5. Keep functionality identical

**Files to Modify** (~40 files, ~800 lines):
1. `internal/races/races.go` → `internal/species/species.go`
2. `internal/characters/character.go` - Update field names
3. `internal/mobs/mobs.go` - Update references
4. All user commands referencing race
5. Config files
6. All YAML race files → species files
7. Test files

**Testing**:
- [ ] **Unit Tests**: Update all race tests to species tests
- [ ] **Manual Test**: Character creation with species selection
- [ ] **Manual Test**: Mob spawning with correct species
- [ ] **Manual Test**: Species-specific abilities still work
- [ ] **Integration Test**: Character lifecycle with species system
- [ ] **Grep Test**: `grep -r "race" --include="*.go"` should show minimal results (comments only)

**Acceptance Criteria**:
- All "race" references renamed to "species"
- Species selection works in character creation
- Mobs spawn with correct species
- All tests pass
- Directory structure updated

**Estimated Changes**: ~800 lines, 40 files

---

### Stage 2.2: Create "Human" Default Species ✅ COMPLETED
**Goal**: Create a single "Human" species for all players. Convert existing playable races to be NPC-only species.

**Changes**:
1. Create `0-human.yaml` species file (if doesn't exist) with:
   - Neutral alignment
   - Base stats all at DOG's mean (100)
   - No special abilities
   - Selectable = true (only selectable species)
2. Mark all other species as `Selectable: false`
3. Update character creation to default to Human (no species choice for players)
4. Keep all animal/monster species for NPCs

**Files to Modify** (~10 files, ~200 lines):
1. `_datafiles/world/default/species/0-human.yaml` - Create/update
2. All other species YAML files - Set `selectable: false`
3. `internal/usercommands/character.go` - Default to human species
4. Remove species selection UI from character creation
5. Test files

**Testing**:
- [ ] **Unit Tests**: Test human species loads correctly
- [ ] **Manual Test**: Character creation auto-assigns human species
- [ ] **Manual Test**: Verify cannot select other species
- [ ] **Manual Test**: NPCs/mobs still spawn with their species
- [ ] **Manual Test**: Load old characters, verify species conversion

**Acceptance Criteria**:
- All new players are human
- Cannot select non-human species during creation
- NPCs retain their species
- Old characters convert gracefully
- All tests pass

**Estimated Changes**: ~200 lines, 10 files

---

## Phase 3: Remove Level System (High Risk - Take Carefully)

### Stage 3.1: Add Skill-Based Progression Hooks (Preparation) ✅ COMPLETED
**Goal**: Create the infrastructure for skill-use progression without removing levels yet.
**Status**: Merged into `development` — see commit history for details.

---

### Stage 3.2: Implement Skill Progression Formula ✅ COMPLETED
**Goal**: Create the actual progression logic.
**Status**: Merged into `development` — see commit history for details.

---

### Stage 3.3: Dual Progression Mode (Levels + Skills) ✅ COMPLETED
**Goal**: Run both systems in parallel - existing level system + new skill progression.
**Status**: Merged into `development` — see commit history for details.

---

### Stage 3.4: Decouple Combat from Levels ✅ COMPLETED
**Goal**: Make combat use skill ranks instead of levels for calculations.
**Status**: Merged into `development` — commit `f23260b`.

---

### Stage 3.5: Remove Level from Player Progression ✅ COMPLETED
**Goal**: Stop awarding levels to players. Progression is skill-based.
**Status**: Merged into `development` — commit `f020034`.

**What was done**:
- `LevelUp()` disabled (returns false)
- Combat formulas decoupled from Level (use `GetCombatSkillLevel()` instead)
- Level still exists in Character struct but is no longer advanced

**What remains** (addressed in Stages 3.6–3.9 below):
- Mobs still use Level for stat initialization (`AutoTrain()`, `StatPoints = Level`)
- `Experience` field still populated on mob creation
- Only 15 legacy GoMud skills exist; no DOG-specific combat skills yet
- Documentation still describes old GoMud systems

---

### Stage 3.6: Remove Level/XP from Mobs (Mechanical Parity) ✅ COMPLETED
**Goal**: Make mobs define their power through stats and skills directly, not derived from a level field. Players and mobs become truly mechanically identical.
**Status**: Merged into `development` — commit `f65a9d4`.

**Changes**:
1. Remove `Level`-based stat initialization from mob creation:
   - Remove `StatPoints = Level` and `AutoTrain()` from `NewMobById()`
   - Remove `Experience = XPTNL()` initialization
2. Mob YAML files define stats directly (already partially supported via `stats:` block)
3. Mob YAML files define skills directly (add `skills:` block if not present)
4. Remove or deprecate `forceLevel` parameter from `NewMobById()`
5. Ensure mob stat recalculation works without Level input
6. Keep `Level` field in Character struct for now (zero value / cosmetic) to avoid breaking serialization

**Files to Modify** (~15 files, ~400 lines):
1. `internal/mobs/mobs.go` - Remove level-based initialization from `NewMobById()`
2. `internal/characters/character.go` - Ensure `Recalculate()` works with Level=0
3. `internal/stats/stats.go` - Ensure stat calculation doesn't require Level
4. `_datafiles/world/default/mobs/**/*.yaml` - Verify/update stat and skill definitions
5. `internal/combat/` - Remove any remaining Level fallbacks (e.g., `GetCombatSkillLevel()` Level fallback)
6. Test files

**Testing**:
- [ ] **Unit Tests**: Mob creation without level produces correct stats
- [ ] **Manual Test**: Spawn mobs, verify stats match YAML definitions
- [ ] **Manual Test**: Combat with mobs still works and feels balanced
- [ ] **Manual Test**: Shopkeeper and persistent NPCs still function
- [ ] **Regression Test**: `go test ./...` passes

**Acceptance Criteria**:
- Mobs initialize from stats/skills in YAML, not from Level
- No `AutoTrain()` or `StatPoints = Level` in mob creation path
- Combat works identically (mob difficulty unchanged)
- Players and mobs use the same stat/skill resolution — no special cases
- All tests pass

**Estimated Changes**: ~400 lines, 15 files

---

### Stage 3.7: Add Core Combat & Magic Skills ✅ COMPLETED
**Goal**: Replace the legacy GoMud skill set with DOG's core combat and magic skills so combat has proper skill references instead of fallbacks.
**Status**: Merged into `development` — commit `453367c`.

**New Skills (5 total — minimal set covering all combat/magic mechanics)**:

| Skill | Covers | Replaces |
|-------|--------|----------|
| **Weapon Combat** | Melee attack & defense with weapons, parrying, weapon techniques | Brawling (partially), DualWield |
| **Unarmed Combat** | Fist/body attacks & defense, grappling, martial arts | Brawling |
| **Ranged Combat** | Bows, crossbows, thrown weapons — attack & defense at range | (new) |
| **Spellcasting** | All magic — elemental, enhancement, vital schools — offense & defense | Cast, Enchant, Scribe |
| **Psionics** | Mental powers — telepathy, telekinesis, illusion, charm — offense & defense | (new) |

**Design Notes**:
- Each skill covers both offense and defense in its domain (no separate Defense skill)
- Opposed rolls: attacker's combat skill vs defender's combat skill (e.g., Weapon Combat vs Weapon Combat, or Weapon Combat vs Unarmed Combat)
- Magic defense against physical: Spellcasting/Psionics can serve as defense (e.g., magical shield, mental dodge)
- Skills use the existing 1-4 level system for now; progression rework comes in Phase 4+
- Legacy skills (Map, Search, Track, Skulduggery, etc.) remain untouched — they still work

**Changes**:
1. Add 5 new skill definitions to `internal/skills/skills.go`
2. Update combat resolution to reference new skills instead of Brawling fallback
3. Remove `GetCombatSkillLevel()` Level-based fallback (mobs now have real skills from 3.6)
4. Add skill YAML data for the new skills
5. Update mob YAML files to use new combat/magic skills where appropriate

**Files to Modify** (~15 files, ~500 lines):
1. `internal/skills/skills.go` - Add new skill definitions
2. `internal/combat/combat.go` - Use new skill names in resolution
3. `internal/combat/calculations.go` - Update formulas for new skill references
4. `_datafiles/` - Skill training data, mob skill assignments
5. Test files

**Testing**:
- [ ] **Unit Tests**: New skills load and resolve correctly
- [ ] **Manual Test**: Combat using Weapon Combat / Unarmed Combat / Ranged Combat
- [ ] **Manual Test**: Spellcasting and Psionics function in combat
- [ ] **Manual Test**: Mobs with different combat skills fight appropriately
- [ ] **Balance Test**: Combat with new skill references feels comparable to before
- [ ] **Regression Test**: Legacy skills (Map, Search, etc.) still work

**Acceptance Criteria**:
- 5 new combat/magic skills defined and functional
- Combat uses real skills — no Level fallbacks anywhere
- Mobs have appropriate combat skills assigned
- Legacy utility skills unaffected
- All tests pass

**Estimated Changes**: ~500 lines, 15 files

---

### Stage 3.8: Add Foundational Non-Combat Skills ✅ COMPLETED
**Goal**: Add a minimal set of non-combat skills so other systems have real skill references. Keep the list small — we can always expand later.
**Status**: Merged into `development` — commit `e6d7cf1`.

**New Skills (5 total — skeleton covering core non-combat mechanics)**:

| Skill | Covers | Replaces/Maps To |
|-------|--------|-----------------|
| **First Aid** | Healing others, treating wounds, stabilizing the dying | (new) |
| **Stealth** | Sneaking, hiding, avoiding detection | Skulduggery (partially) |
| **Tracking** | Finding creatures/players, reading trails, navigation | Track |
| **Bartering** | Trade prices, negotiation, appraisal | Trading |
| **Foraging** | Gathering resources — herbs, wood, ore, food | (new) |

**Design Notes**:
- These 5 + the 5 combat skills from 3.7 = 10 core DOG skills
- Existing legacy skills that overlap (Track, Trading, Skulduggery) should be mapped/aliased to the new names
- Legacy skills without a DOG equivalent (Map, Portal, Search, Peep, Inspect, Tame, DualWield) remain as-is for now
- Crafting skills (Blacksmithing, Alchemy, etc.) deferred to a later phase when the crafting system is built

**Changes**:
1. Add 5 new skill definitions
2. Map/alias legacy skills to new DOG skills where appropriate
3. Ensure the `score` command displays new skills
4. Add skill YAML data

**Files to Modify** (~10 files, ~300 lines):
1. `internal/skills/skills.go` - Add new skill definitions
2. `internal/usercommands/score.go` - Display new skills
3. `_datafiles/` - Skill data files
4. Test files

**Testing**:
- [ ] **Manual Test**: New skills appear in character score
- [ ] **Manual Test**: Skills can be trained at appropriate locations
- [ ] **Manual Test**: Legacy skills still function
- [ ] **Regression Test**: `go test ./...` passes

**Acceptance Criteria**:
- 5 new non-combat skills defined
- Legacy skill aliases work correctly
- Score displays all skills
- All tests pass

**Estimated Changes**: ~300 lines, 10 files

---

### Stage 3.9: Update Documentation ✅ COMPLETED
**Goal**: Bring all documentation in sync with the actual codebase state after Stages 3.1–3.8.

**Changes**:
1. Update `DEVELOPMENT_PLAN.md`:
   - Mark all completed stages
   - Update "Current Stage" and "Status" fields
   - Update timeline estimates
2. Update `internal/stats/context.md`:
   - Replace old stat names (Speed, Smarts, Mysticism) with current names
   - Remove level-based progression documentation
   - Document skill-based stat resolution
3. Update `internal/mobs/context.md`:
   - Remove level-based mob creation documentation
   - Document stats/skills-based mob initialization
4. Update `internal/skills/context.md`:
   - Document new DOG skill set (10 core skills)
   - Document legacy skill mapping
5. Update `internal/combat/context.md`:
   - Document skill-based combat resolution
   - Remove level references
6. Update `internal/characters/context.md`:
   - Remove level/XP progression documentation
   - Document skill-based progression
7. Scan remaining `context.md` files for stale Level/XP/Mana references and update as needed

**Files to Modify** (~10-20 context.md files):
1. `internal/stats/context.md`
2. `internal/mobs/context.md`
3. `internal/skills/context.md`
4. `internal/combat/context.md`
5. `internal/characters/context.md`
6. Other context.md files with stale references
7. `DEVELOPMENT_PLAN.md` (final status update)

**Testing**:
- [ ] **Grep Test**: Search all context.md files for "level" / "Level" / "XP" / "experience" — verify references are accurate or removed
- [ ] **Review**: Each updated file accurately describes the current code behavior

**Acceptance Criteria**:
- All context.md files reflect current codebase state
- No stale references to level-based progression, old stat names, or mana
- Dev plan accurately tracks completed vs. pending work
- Documentation is useful as a reference for future development

**Estimated Changes**: ~20 files, documentation only

---

## Phase 4: Distribution-Based Combat

> **Note (2026-02-07):** The original plan had two stages (4.1 and 4.2). Stage 4.1's
> distribution rolling library was implemented early as a side quest and already exists
> in `internal/dice/dice.go` with full test coverage (`dice_test.go`). This includes:
> `Roll()`, `OpposedRoll()`, `DifficultyCheck()`, `CriticalCheck()`, `RollDamage()`,
> `normalCDF()`, `inverseNormalCDF()`, and more. Character creation already uses it
> via `RollCharacterStats()`. Stages 4.1 and 4.2 are therefore collapsed into a single
> stage that wires the existing `dice` package into combat.

### Stage 4.1: ~~Add Distribution Rolling~~ ✅ COMPLETED (side quest)
**Goal**: ~~Implement DOG's distribution-based rolling without removing existing dice rolls.~~
**Status**: Already implemented in `internal/dice/dice.go` during earlier development.

The `dice` package provides:
- `Roll(mean, stdDev)` → `RollResult` with Value, ZScore, Percentile
- `OpposedRoll(atkStat, defStat, stdDev)` → contested checks
- `DifficultyCheck(stat, difficulty, stdDev)` → skill checks
- `CriticalCheck(result, critThreshold, fumbleThreshold)` → crit/fumble detection
- `RollDamage(baseDamage, variance, minDamage)` → damage rolls
- `RollWithCriticals(mean, stdDev, critThresh, fumbleThresh)` → combined roll+crit
- Full statistical math: `normalCDF`, `inverseNormalCDF`, `SuccessChance`, `OpposedSuccessChance`
- Tests in `dice_test.go` including `TestRollDistribution`

No config flag needed — we will wire the `dice` package directly into combat in Stage 4.2.

---

### Stage 4.2: Replace Dice Combat with Distribution Combat ✅ COMPLETED (merge: 3892439)
**Goal**: Replace all legacy `util.RollDice()` / `util.Rand()` combat calculations with the existing `dice` package's distribution-based rolling.

**Changes**:
1. Replace all combat calculations in `internal/combat/`:
   - Hit chance: Use `dice.OpposedRoll(atkDex + combatSkill, defDex + combatSkill, stdDev)`
   - Critical hits: Use `dice.CriticalCheck()` with z-score thresholds (±2σ ≈ ~2.5% each)
   - Damage: Use `dice.RollDamage(baseDamage, variance, minDamage)` instead of `util.RollDice()`
   - Attack count: Derive from Dexterity differential using distribution math
2. Update weapon items to use base damage + variance instead of dice notation (e.g., `2d6+3`)
3. Convert weapon YAML files from dice notation to `baseDamage` / `variance` fields
4. Remove old `util.RollDice()` calls from combat path

**Files to Modify** (~15 files, ~800 lines):
1. `internal/combat/combat.go` - Replace attack functions to use `dice` package
2. `internal/combat/calculations.go` - Replace hit/crit/damage formulas
3. `internal/items/itemspec.go` - Add baseDamage/variance fields alongside or replacing dice notation
4. All weapon YAML files - Convert dice to damage multipliers
5. Test files

**Testing**:
- [ ] **Unit Tests**: Test all new combat calculations
- [ ] **Unit Tests**: Test critical hit/miss detection via z-score
- [ ] **Manual Test**: Extensive combat testing (easy/medium/hard enemies)
- [ ] **Manual Test**: Verify damage feels balanced
- [ ] **Balance Test**: Compare statistical expected damage vs old system
- [ ] **Regression Test**: Ensure no combat bugs, `go test ./...` passes

**Acceptance Criteria**:
- All combat uses `dice` package distribution rolling
- No `util.RollDice()` in combat path
- Combat balance feels appropriate
- Crits happen at expected rate (~5% total: ~2.5% crit success + ~2.5% crit fail)
- All tests pass

**Estimated Changes**: ~800 lines, 15 files

---

### Stage 4.3: Combat Test Content ✅ COMPLETED (merge: a9ac95d)
**Goal**: Create a set of test rooms, mobs, and gear specifically tuned for playtesting the new distribution-based combat system. This gives us controlled scenarios to verify balance, crit rates, and damage curves.

**Changes**:
1. Create a small test zone (3–5 rooms) accessible from an existing area
   - Training yard / combat arena theme
   - Rooms connected linearly for easy navigation
2. Create test mobs at varying difficulty tiers:
   - **Training Dummy** — Very low stats, stands still. For testing basic hit/damage output
   - **Sparring Partner** — Moderate stats, balanced. For testing opposed rolls
   - **Arena Champion** — High stats, tough fight. For testing hard encounters and crit impact
   - Each mob uses the StatPool system (no levels), with stats tuned to exercise different combat paths
3. Create test weapons covering different combat skill routes:
   - A basic sword (weapon-combat)
   - A staff or fist wraps (unarmed-combat)
   - A bow (ranged-combat)
   - Each with `baseDamage` / `variance` fields (distribution-friendly format from 4.2)
4. Create basic armor set for survivability testing
   - Simple chest piece and shield with stat modifiers

**Files to Create/Modify** (~8 files, ~200 lines):
1. `_datafiles/rooms/test_arena/` — New room YAML files (3–5 rooms)
2. `_datafiles/mobs/test_arena/` — New mob YAML files (3 mobs)
3. `_datafiles/items/weapons/test/` — Test weapon YAML files (3 weapons)
4. `_datafiles/items/armor/test/` — Test armor YAML files (1–2 items)
5. Room exit connections to link test zone to an existing area

**Testing**:
- [ ] **Manual Test**: Navigate to test zone, verify all rooms load
- [ ] **Manual Test**: Fight each mob tier, verify combat feels balanced
- [ ] **Manual Test**: Equip each test weapon, verify correct combat skill routing
- [ ] **Manual Test**: Verify crit/fumble rates feel reasonable over many fights
- [ ] **Balance Test**: Easy mob should be trivial, medium should be fair, hard should be dangerous
- [ ] **Regression Test**: `go build ./...` and `go test ./...` pass

**Acceptance Criteria**:
- Test zone accessible in-game
- Three difficulty tiers of mobs exercising the distribution combat system
- Three weapon types covering weapon-combat, unarmed-combat, ranged-combat skills
- All content loads without errors
- Combat balance feels appropriate for the new system

**Estimated Changes**: ~200 lines, 8 files

---

### Stage 4.4: Bypass Newbie Tutorial for New Characters ✅ COMPLETED (merge: 6a831c7)
**Goal**: New characters spawn directly in Town Square (Room 1) instead of the void (Room -1), skipping the tutorial flow entirely. The tutorial code stays in place but is disconnected — no characters will enter the void state that triggers it. This removes friction when creating test characters during development.

**How the tutorial currently works:**
1. New characters start with `RoomId = -1` (void) — defined as `StartingRoomId` in `character.go`
2. The `start` command detects the void state and runs name selection → tutorial prompt → tutorial rooms (900-903)
3. A newbie guide NPC (mob 38) spawns for players with Level ≤ 5 via `RoomChange_SpawnGuide` hook

**Changes:**
1. Change `StartingRoomId` from `-1` to `0` in `internal/characters/character.go`
   - Room 0 means "use the configured StartRoom" — the room manager resolves this to Room 1 (Town Square)
   - The `start` command's void-detection (`RoomId == -1`) never fires, so the tutorial is bypassed
2. Disable the newbie guide spawn hook — since DOGMud doesn't use levels, the Level ≤ 5 check is already broken. Add an early return or config flag to prevent guide spawning entirely.

**What stays intact (not deleted):**
- Tutorial rooms (900-903) and their YAML files
- The `start` command code (just won't be triggered)
- Guide mob YAML (mob 38)
- All ephemeral room logic

**Files to Modify** (~3 files, ~10 lines):
1. `internal/characters/character.go` — Change `StartingRoomId` constant
2. `internal/hooks/RoomChange_SpawnGuide.go` — Add early return to disable guide spawning

**Testing**:
- [ ] **Manual Test**: Create new character, verify spawns in Town Square (not void)
- [ ] **Manual Test**: Verify `start` command is not triggered
- [ ] **Manual Test**: Verify guide NPC does not spawn
- [ ] **Regression Test**: `go build ./...` and `go test ./...` pass

**Acceptance Criteria**:
- New characters land directly in Town Square
- No tutorial prompt or void state
- Guide NPC disabled
- Tutorial code remains in codebase (not deleted)

**Estimated Changes**: ~10 lines, 2-3 files

---

## Phase 4b: Progression Bug Fixes & Tuning

> **Context (2026-02-12):** Playtesting after Stage 4.4 revealed several progression
> and combat issues. These are bug fixes and tuning passes that should land before
> building new systems on top of a broken foundation.

### Stage 4.5: Fix Stat Progression Triggers ✅ COMPLETED (merge: 3cda200)
**Goal**: Fix why only Vitality appears to increase — other stats lack proper usage triggers and there's a stale stat name reference.

**Known Issues**:
1. `TrackStatUse("mysticism")` is called during spell casting, but the stat was renamed to `willpower` in Stage 1.1 — so willpower never progresses from spellcasting
2. `perception` and `charisma` have **no usage triggers anywhere** — they never get `TrackStatUse` calls, so they can never progress via usage
3. `strength` and `dexterity` are tracked during combat attacks but only on the attacker's turn — if you're mainly defending/tanking, neither progresses
4. Vitality works because it has a special `OnLowResource("health", "vitality")` trigger at <25% health — the other stats lack equivalent resource-based triggers

**Changes**:
1. Fix `TrackStatUse("mysticism")` → `TrackStatUse("willpower")` in all spell/magic paths
2. Add stat usage triggers for missing stats:
   - **Perception**: Track on successful searches, tracking, noticing hidden mobs/players, ranged combat
   - **Charisma**: Track on bartering, conversations with NPCs, taunts, group commands
   - **Strength**: Also track when dealing melee damage (not just attacking), carrying heavy loads
   - **Dexterity**: Also track when dodging/avoiding attacks (defender side)
3. Add `OnLowResource` equivalents for stamina → strength/dexterity, conviction → willpower/charisma
4. Verify all 6 stats can actually progress through normal gameplay

**Files to Modify** (~10 files, ~200 lines):
1. `internal/combat/combat.go` — Fix "mysticism" reference, add defender stat tracking
2. `internal/hooks/NewRound_DoCombat.go` — Fix spell stat tracking
3. `internal/characters/progression.go` — Add low-stamina → strength, low-conviction → willpower triggers
4. `internal/usercommands/search.go`, `track.go`, etc. — Add perception tracking
5. `internal/usercommands/buy.go`, `sell.go` — Add charisma tracking
6. Test files

**Testing**:
- [ ] **Manual Test**: Fight mobs, verify strength/dexterity messages appear
- [ ] **Manual Test**: Cast spells, verify willpower increases (not "mysticism")
- [ ] **Manual Test**: Search/track, verify perception increases
- [ ] **Manual Test**: Buy/sell items, verify charisma increases
- [ ] **Manual Test**: Get low stamina, verify strength/dexterity progression triggers
- [ ] **Regression Test**: `go test ./...` passes

**Acceptance Criteria**:
- All 6 primary stats have at least one usage-based progression path
- No references to old stat name "mysticism" remain in progression code
- Low-resource triggers exist for stamina and conviction (not just health)
- All tests pass

**Estimated Changes**: ~200 lines, 10 files

---

### Stage 4.6: Tune Skill Progression — Soft Cap Fix & Per-Skill Multipliers ✅ COMPLETED (merge: 605cf31)
**Goal**: Fix the skill soft cap so skills don't blow past 100+ trivially, and add a per-skill `ProgressionMultiplier` so frequently-used skills (like unarmed combat) can be slowed down relative to rarely-used skills (like tracking).

**Known Issues**:
1. `SkillSoftCap = 50` virtual ranks with `UsesPerRank = 10` means 500 total uses to reach the "soft cap" — but skills like unarmed combat fire multiple times per combat round, so you can blow through this in a single session
2. The progression chance at rank 0 is 50%, which is extremely generous — skills increase almost every other use at the start
3. All skills progress at the same rate regardless of how frequently they're used, making combat skills vastly easier to level than utility skills

**Changes**:
1. Add `ProgressionMultiplier float64` field to the skill definition struct
   - Default: `1.0`
   - Combat skills (weapon-combat, unarmed-combat, ranged-combat): `0.3` — these fire many times per round
   - Utility skills (tracking, bartering, foraging): `2.0` — these fire rarely
   - Magic skills (spellcasting, psionics): `0.5`
2. Apply the multiplier in `CheckSkillProgression()` to scale the progression chance
3. Tune the soft cap constants:
   - Consider raising `UsesPerRank` from 10 to 25 (more uses needed per virtual rank)
   - Or lower the base chance from 50% to 30%
   - Or both — test and find the sweet spot
4. Add the multiplier to skill YAML data files
5. Ensure skills above the soft cap still progress, just very slowly

**Files to Modify** (~8 files, ~200 lines):
1. `internal/skills/skills.go` — Add `ProgressionMultiplier` field
2. `internal/characters/progression.go` — Apply multiplier in progression check
3. `_datafiles/` skill YAML files — Set multipliers per skill
4. `internal/characters/progression_test.go` — Update tests for new multiplier
5. Test files

**Testing**:
- [ ] **Unit Tests**: Verify multiplier affects progression chance correctly
- [ ] **Manual Test**: Fight mobs extensively, verify unarmed combat doesn't skyrocket
- [ ] **Manual Test**: Use utility skills, verify they progress at a reasonable rate
- [ ] **Manual Test**: Verify skills still progress above soft cap (just slowly)
- [ ] **Balance Test**: Compare progression rates across skill types

**Acceptance Criteria**:
- Per-skill multipliers are configurable in YAML
- Combat skills progress slower per-use than utility skills
- Skill progression feels achievable but not trivial
- Soft cap creates a real slowdown, not just a minor speed bump
- All tests pass

**Estimated Changes**: ~200 lines, 8 files

---

### Stage 4.7: Tune Critical Hit/Failure Rates & Messaging ✅ COMPLETED
**Goal**: Fix the perception that critical successes happen far more often than critical failures, and improve crit messaging clarity.

**Known Issues**:
1. Crits may be happening at correct statistical rates but the messaging may be asymmetric (crit successes produce flashy text, crit failures are subtle or missing)
2. The z-score thresholds in `dice.CriticalCheck()` may need adjustment
3. Need to audit all combat paths to ensure both crit success AND crit failure are reported consistently

**Changes**:
1. Audit `dice.CriticalCheck()` thresholds and verify the actual statistical rates match the intended ~2.5% each
2. Add logging/counters to track actual crit rates during testing
3. Review all combat message paths — ensure crit failures produce equally visible messaging as crit successes
4. Standardize crit messaging format (e.g., colored prefix tags for both)
5. Consider adding a brief explanation to crit messages so players understand what happened

**Files to Modify** (~5 files, ~150 lines):
1. `internal/dice/dice.go` — Audit/adjust thresholds
2. `internal/combat/combat.go` — Audit crit messaging paths
3. `internal/combat/calculations.go` — Verify crit application
4. Test files

**Testing**:
- [ ] **Unit Tests**: Statistical test — run 10,000 rolls, verify crit rates within tolerance
- [ ] **Manual Test**: Extended combat, verify crit successes and failures appear at roughly equal rates
- [ ] **Manual Test**: Verify both crit success and crit failure messages are clearly visible
- [ ] **Regression Test**: `go test ./...` passes

**Acceptance Criteria**:
- Crit success and crit failure rates are statistically balanced (~2.5% each)
- Both produce clearly visible, distinct combat messages
- Players can easily tell when a crit success vs crit failure occurs
- All tests pass

**Estimated Changes**: ~150 lines, 5 files

**Merge commit**: `698011d`

#### Side Quest: Arena Balance Tuning ✅ COMPLETED
After Stage 4.7, manual testing revealed that arena mobs were too weak for new players (who start with ~100 stats and all skills). Applied the following balance changes:
- **Human species**: Base stats raised from 1 → 100 (affects all human NPCs; player stats are rolled 70-130 so unaffected)
- **Training dummy**: Weak stats (10) but high vitality (200) — punching bag for testing damage
- **Sparring partner/archer**: Slightly above average (Training +10-20, skills ~40 in primaries)
- **Arena champion**: Strong (Training +40-50 for ~150 raw stats, skills 100 in weapon/unarmed combat)
- All arena mobs switched from random `statpool` to explicit `training` values for deterministic stats

---

### Stage 4.8: Remove Player Guide ✅ COMPLETED
**Goal**: Since levels have been removed, the player guide (which was level-oriented) is no longer relevant. Remove all dead code and data files.

**Changes**:
1. Deleted `internal/hooks/RoomChange_SpawnGuide.go` — dead code (early return since Stage 4.4)
2. Deleted `internal/hooks/LevelUp_CheckGuide.go` — `CheckGuideBySkillRanks()` defined but never called
3. Removed `SpawnGuide` listener registration and stale guide dismissal comment from `internal/hooks/hooks.go`
4. Deleted `_datafiles/world/dogmud/mobs/startland/38-player_guide.yaml` — guide mob data
5. Deleted `_datafiles/world/dogmud/templates/help/guide.template` — guide help template

**Testing**:
- [x] `go build ./...` — compiles cleanly
- [x] `go test ./...` — all tests pass

---

## Phase 5: Movement & Stamina System

### Stage 5.1: Connect Stamina to Movement ✅ COMPLETED
**Goal**: Make movement consume stamina.
**Status**: Merged into `development` — commit `dc34e0c`.

**Changes**:
1. Update movement commands to deduct stamina:
   - Calculate stamina cost based on terrain, encumbrance
   - Flat terrain, unencumbered = 2 stamina
   - Scale up to 20 stamina for rough terrain + heavy encumbrance
2. Add stamina checks before movement (prevent if insufficient)
3. Add "tired" messages when low stamina
4. Update stamina regeneration (current: 2 game hours to full)

**Files to Modify** (~12 files, ~400 lines):
1. `internal/usercommands/move.go` - Add stamina deduction
2. `internal/characters/character.go` - Add stamina cost calculation
3. `internal/rooms/rooms.go` - Add terrain difficulty data
4. `internal/configs/config.gameplay.go` - Add stamina cost config
5. Test files

**Testing**:
- [x] **Unit Tests**: Test stamina cost calculations
- [x] **Manual Test**: Move around, verify stamina decreases
- [x] **Manual Test**: Move while heavily encumbered, verify higher cost
- [x] **Manual Test**: Try to move with 0 stamina, verify blocked
- [x] **Manual Test**: Rest, verify stamina regenerates
- [x] **Balance Test**: Verify stamina costs feel appropriate

**Acceptance Criteria**:
- Movement costs stamina
- Cannot move with insufficient stamina
- Costs scale with terrain and encumbrance
- Regeneration works correctly
- All tests pass

**Estimated Changes**: ~400 lines, 12 files

---

### Stage 5.2: Stamina in Combat ✅ COMPLETED
**Goal**: Make combat actions consume stamina. Attacks, dodges, and blocks all cost stamina. Running out of stamina in combat has real consequences.
**Status**: Merged into `development` — commit `5366e2b`.

**Changes**:
1. Each attack costs stamina (scaled by weapon weight/speed)
2. Blocking/dodging costs stamina
3. Fleeing costs stamina
4. Out-of-stamina penalty in combat (reduced attack count, can't dodge/block effectively)
5. Stamina regenerates slowly during combat (much slower than out of combat)

**Files to Modify** (~10 files, ~500 lines):
1. `internal/combat/combat.go` - Add stamina costs to attacks
2. `internal/combat/calculations.go` - Add out-of-stamina penalties
3. `internal/characters/character.go` - Add stamina calculation methods
4. `internal/hooks/NewRound_DoCombat.go` - Update combat round logic
5. Test files

**Testing**:
- [x] **Unit Tests**: Test stamina costs for attacks
- [x] **Manual Test**: Long combat, verify stamina decreases
- [x] **Manual Test**: Run out of stamina in combat, verify penalties
- [x] **Manual Test**: Flee, verify stamina cost
- [x] **Balance Test**: Verify combat stamina drain feels appropriate

**Acceptance Criteria**:
- Combat consumes stamina
- Out-of-stamina penalties apply
- Stamina costs feel balanced
- All tests pass

**Estimated Changes**: ~500 lines, 10 files

---

### Stage 5.3: Rework Number of Attacks Formula ✅ COMPLETED
**Goal**: Replace the current attack count calculation with a formula based on: `Dexterity * weapon speed multiplier * encumbrance modifier * stamina modifier`, rounded to the nearest whole number each round.
**Status**: Merged into `development` — commit `288774b`.

**Changes**:
1. Add `SpeedMultiplier float64` field to weapon item spec (fast daggers ~1.5, slow greataxes ~0.5, fists ~1.2)
2. Calculate encumbrance modifier from carried weight vs capacity (1.0 at light load → 0.5 at max load)
3. Calculate stamina modifier (1.0 at full stamina → 0.5 when winded, 0.25 when exhausted)
4. Formula: `attacks = round(baseDexAttacks * weaponSpeed * encumbrance * staminaMod)`
   - `baseDexAttacks` derived from Dexterity stat (e.g., Dex/50 gives ~2 attacks at 100 Dex)
   - Minimum 1 attack per round
5. Update weapon YAML files with speed multipliers
6. Dual wielding: calculate for each weapon separately

**Files to Modify** (~10 files, ~300 lines):
1. `internal/combat/calculations.go` — New attack count formula
2. `internal/items/itemspec.go` — Add `SpeedMultiplier` field
3. `internal/characters/character.go` — Encumbrance ratio helper
4. Weapon YAML files — Add speed multipliers
5. Test files

**Testing**:
- [ ] **Unit Tests**: Test attack count formula with various inputs
- [ ] **Manual Test**: Equip fast vs slow weapons, verify attack count differs
- [ ] **Manual Test**: Get encumbered, verify fewer attacks
- [ ] **Manual Test**: Get winded, verify fewer attacks
- [ ] **Balance Test**: Verify attack counts feel right (1-4 typical range)

**Acceptance Criteria**:
- Attack count scales with Dexterity, weapon speed, encumbrance, and stamina
- Fast weapons give more attacks, slow weapons fewer
- Being encumbered or winded reduces attacks
- Minimum 1 attack per round
- All tests pass

**Estimated Changes**: ~300 lines, 10 files

---

### Stage 5.4: Weight-Based Encumbrance ✅ COMPLETED
**Goal**: Replace item-count based encumbrance with realistic weight-based system and display encumbrance in character sheet.
**Status**: Merged into `development` — commit `69eec06`.

**Current State**: Encumbrance uses item count (1 dagger = 1 greatsword = "1 item"). This is simple but unrealistic.

**Changes**:
1. Add `Weight float64` field to ItemSpec
2. Update CarryCapacity to return weight capacity (e.g., Strength × 3)
3. Add GetCarriedWeight() method to calculate total weight
4. Change encumbrance calculations to use total carried weight
5. Update movement stamina cost formula to use weight ratio
6. Update combat attack count penalty to use weight ratio
7. **Add encumbrance display to status command** - Show current weight, capacity, and encumbrance level
8. Add weight values to all item YAML files
   - Light weapons: 2-5 lbs (daggers, shortswords)
   - Medium weapons: 6-12 lbs (longswords, maces)
   - Heavy weapons: 15-25 lbs (greatswords, warhammers)
   - Armor: varies by type
   - Consumables: 0.5-2 lbs

**Files to Modify** (~52 files, ~250 lines):
1. `internal/items/itemspec.go` - Add Weight field, GetWeight() method
2. `internal/characters/character.go` - Update CarryCapacity, add GetCarriedWeight
3. `internal/characters/character.go` - Update GetMovementStaminaCost to use weight
4. `internal/combat/combat.go` - Update encumbrance penalty to use weight
5. `internal/usercommands/status.go` - Add encumbrance display
6. `_datafiles/world/dogmud/templates/character/status.template` - Add encumbrance section
7. All item YAML files - Add weight values
8. Test files

**Testing**:
- [ ] **Unit Tests**: Test weight calculations, GetCarriedWeight
- [ ] **Manual Test**: Check status command shows encumbrance
- [ ] **Manual Test**: Carry light items vs heavy items, verify different stamina costs
- [ ] **Manual Test**: Carry greatswords vs daggers, verify combat penalties differ
- [ ] **Balance Test**: Verify weights feel realistic

**Acceptance Criteria**:
- Items have weight values
- Encumbrance based on weight vs strength
- Heavier items more encumbering than lighter items
- Status command displays current weight/capacity/encumbrance level
- All tests pass

**Estimated Changes**: ~250 lines, 52 files

---

## Phase 6: Conviction & Magic System

### Stage 6.1: Add Spell Schools as Tags ✅ COMPLETED
**Goal**: Implement DOG's 4 spell schools (Elemental, Enhancement, Mental, Vital).
**Status**: Merged into `development` — commit `6b65be6`.

**Changes**:
1. Add `Schools []string` field to spell data (can have multiple tags)
2. Create 4 school categories
3. Tag existing spells with appropriate schools
4. Keep existing spell functionality unchanged

**Files to Modify** (~20 files, ~300 lines):
1. `internal/spells/spells.go` - Add school field
2. All spell YAML files - Add school tags
3. `internal/usercommands/cast.go` - Display school info
4. Test files

**Testing**:
- [ ] **Unit Tests**: Verify spell school tagging
- [ ] **Manual Test**: Cast spells, verify schools display
- [ ] **Manual Test**: View spellbook, verify schools shown
- [ ] **Data Test**: Verify all spells have at least one school

**Acceptance Criteria**:
- All spells tagged with schools
- Schools display correctly
- Spells still functional
- All tests pass

**Estimated Changes**: ~300 lines, 20 files

---

### Stage 6.2: Replace Mana with Conviction Costs ✅ COMPLETED
**Goal**: Make spells cost Conviction instead of mana.
**Status**: Merged into `development` — commit `6b65be6`.

**Note**: Core conviction cost system was implemented in Stage 1.3 (commit `5eb0bdf`). This stage added:
- Optional Health costs for life-force magic (HealthCost field)
- Spell cost scaling configuration (SpellConvictionCostMultiplier, SpellHealthCostMultiplier)
- Health cost display in spells command
- Example implementation in healall.yaml

**Changes**:
1. Add `ConvictionCost int` to spell data
2. Add conviction checks before casting
3. Deduct conviction on cast
4. Some spells may also cost Health (life-force magic)
5. Add config for spell cost scaling

**Files to Modify** (~12 files, ~400 lines):
1. `internal/spells/spells.go` - Add conviction cost
2. `internal/usercommands/cast.go` - Check/deduct conviction
3. All spell YAML files - Set conviction costs
4. `internal/characters/character.go` - Add conviction deduction methods
5. Test files

**Testing**:
- [ ] **Unit Tests**: Test conviction cost checks
- [ ] **Manual Test**: Cast spells, verify conviction decreases
- [ ] **Manual Test**: Try to cast with insufficient conviction
- [ ] **Manual Test**: Verify conviction regenerates
- [ ] **Balance Test**: Verify spell costs feel appropriate

**Acceptance Criteria**:
- Spells cost conviction
- Cannot cast without sufficient conviction
- Conviction costs balanced
- All tests pass

**Estimated Changes**: ~400 lines, 12 files

---

## Phase 7: Defense Overhaul & Combat Restrictions

### Stage 7.1: Segmented Avoidance — Dodge, Parry, Block ✅ COMPLETED
**Merge commit**: cd146e5
**Goal**: Replace the single "attack roll vs defense roll" with a layered defense system: after the attack roll succeeds, the defender gets separate checks for dodge, parry, and block — each based on different skills and equipment.

**Design**:
- **Attack roll**: Attacker's combat skill + stat vs a base difficulty (not directly opposed by defender)
- **Defensive checks** are equipment-dependent, rolled in order until one succeeds:
  - **Dodge check**: Based on Unarmed Combat skill + Dexterity. Available to everyone. Encumbrance and armor reduce dodge chance. Costs stamina.
  - **Parry check**: Based on Weapon Combat skill + weapon's parry rating. Only available if wielding a weapon. Two-handed weapons get a bonus. Costs stamina.
  - **Block check**: Based on Weapon Combat skill + shield's block rating. Only available if wielding a shield. Costs stamina.
- **Equipment configurations determine defensive sequence** (first success avoids the hit):
  - **Unarmed**: dodge only
  - **Weapon + Shield**: dodge → parry → block
  - **Dual wielding** (2+ weapons): dodge → parry main hand → parry off hand (extensible for 4/6-limbed characters)
  - **Single weapon, no shield**: dodge → parry
  - **Two shields**: dodge → block main hand → block off hand (can still attack via shield bash - manual action with cooldown will be added in a later stage, but normal attack commands should work with shields in this stage)
- **Parry and block** can use the same underlying mechanic, just with different output text for success/fail/crit success/crit failure
- Each defensive action has a stamina cost, creating a resource tradeoff for long fights.

**Changes**:
1. Add `ParryRating float64` to weapon item spec
2. Add `BlockRating float64` to shield item spec
3. Add config multipliers for defensive tuning:
   - `DodgeMultiplier float64` (default: 0.9)
   - `ParryMultiplier float64` (default: 0.9)
   - `BlockMultiplier float64` (default: 0.9)
   - Applied to final success chance for each defensive roll
4. Implement `RollDodge()`, `RollParry()`, `RollBlock()` functions in combat calculations (parry and block can share logic with different output text)
5. Implement equipment-aware defensive sequence logic:
   - Detect what is equipped in main/off hands
   - Determine which defensive checks are available (unarmed, weapon, shield, dual-wield, dual-shield)
   - Execute checks in the correct order for that configuration
6. Refactor `AttackPlayerVsPlayer`/`AttackPlayerVsMob` to use layered defense with equipment-based sequences
7. Generate distinct combat messages for each avoidance type ("You dodge...", "You parry...", "You block...")
8. Add extensible support for multi-limbed characters (3+ weapons/shields) in dual-wield defensive logic

**Files to Modify** (~13 files, ~650 lines):
1. `internal/combat/calculations.go` — Dodge/parry/block formulas with multiplier support
2. `internal/combat/combat.go` — Refactor attack resolution
3. `internal/items/itemspec.go` — Add parry/block ratings
4. `internal/configs/config.gameplay.go` — Add dodge/parry/block multiplier configs
5. `_datafiles/config.yaml` — Set default multiplier values (0.9 each)
6. Weapon/shield YAML files — Set parry/block ratings
7. Combat message templates or inline strings
8. Test files

**Testing**:
- [ ] **Unit Tests**: Test each avoidance check independently (dodge, parry, block)
- [ ] **Unit Tests**: Test that multipliers correctly affect success rates (0.9 = 90% of base chance)
- [ ] **Unit Tests**: Test equipment detection and defensive sequence determination for all configurations
- [ ] **Manual Test**: Fight unarmed, verify dodge-only messages (no parry/block)
- [ ] **Manual Test**: Fight with single weapon (no shield), verify dodge → parry sequence
- [ ] **Manual Test**: Fight with weapon + shield, verify dodge → parry → block sequence
- [ ] **Manual Test**: Fight dual-wielding, verify dodge → parry main → parry off sequence
- [ ] **Manual Test**: Equip two shields, verify dodge → block main → block off sequence
- [ ] **Manual Test**: With two shields, verify can still attack via normal attack command (bash)
- [ ] **Manual Test**: Adjust multipliers in config, verify defensive rates change accordingly
- [ ] **Balance Test**: Verify overall hit rate feels right (~50-70% of attacks land) across all configurations

**Acceptance Criteria**:
- Three distinct avoidance types with separate rolls (dodge, parry, block)
- Each avoidance tied to appropriate skill and equipment
- Config multipliers (DodgeMultiplier, ParryMultiplier, BlockMultiplier) allow tuning without code changes
- Default multiplier values (0.9) applied correctly to all defensive rolls
- All equipment configurations work correctly:
  - Unarmed: dodge only
  - Weapon only: dodge → parry
  - Weapon + shield: dodge → parry → block
  - Dual wield: dodge → parry main → parry off
  - Two shields: dodge → block main → block off
- Dual-shield users can still attack (normal attack = bash)
- System is extensible for 4/6-limbed characters (additional parry/block checks)
- Parry and block share underlying mechanic but have distinct output text
- Combat messages clearly indicate dodge vs parry vs block
- Stamina cost for each defensive action
- All tests pass

**Estimated Changes**: ~650 lines, 13 files

---

### Stage 7.2: Restrict Commands During Combat ✅ COMPLETED
**Merge commit**: de5394a
**Goal**: Prevent players from performing non-combat actions during combat that would be unrealistic or exploitable (e.g., equipping armor, picking up items, crafting).

**Changes**:
1. Add an `AllowedInCombat bool` flag (or a disallow list) to command definitions
2. Commands to **restrict** during combat:
   - `get` / `drop` (picking up / dropping items, especially equipment)
   - `wear` / `remove` (equipping / unequipping armor)
   - `wield` / `unwield` (changing weapons — or add a "weapon swap" action that costs a turn)
   - Any crafting commands
   - `rest` / `sleep`
3. Commands that **remain available** during combat:
   - `attack`, `flee`, `cast`, `look`, `say`, `shout`, `score`, `inventory`
   - `who`, `help`, `quit` (with combat logout timer)
4. Players who try a restricted command get a message: "You can't do that while fighting!"

**Files to Modify** (~8 files, ~200 lines):
1. `internal/usercommands/` — Add combat check to restricted commands
2. `internal/connections/` or main input loop — Add combat state check before dispatch
3. Possibly a centralized "command flags" system

**Testing**:
- [ ] **Manual Test**: Start combat, try to equip armor — verify blocked
- [ ] **Manual Test**: Start combat, try to pick up items — verify blocked
- [ ] **Manual Test**: Start combat, verify attack/flee/cast/look still work
- [ ] **Regression Test**: Out of combat, all commands still work normally

**Acceptance Criteria**:
- Restricted commands are blocked during combat with clear messaging
- Combat-relevant commands still work
- No false positives (commands wrongly blocked out of combat)
- All tests pass

**Estimated Changes**: ~200 lines, 8 files

---

### Stage 7.3: Unarmed Damage Scaling ✅ COMPLETED
**Merge commit**: 1794919
**Goal**: Make unarmed damage scale meaningfully with Unarmed Combat skill and Strength, so training unarmed feels rewarding.

**Changes**:
1. Unarmed base damage formula: `baseDamage = 2 + (Strength / 25) + (UnarmedSkill / 10)`
   - Untrained (skill 0, avg strength): ~6 damage
   - Trained (skill 50, high strength): ~15 damage
   - Master (skill 100, max strength): ~25+ damage
2. Unarmed attacks get bonus attack speed (fists are fast) — reflected in the Stage 5.3 speed multiplier
3. Unarmed critical hits get descriptive flavor text (roundhouse kick, elbow strike, etc.)
4. Unarmed should be a viable combat style, competitive with basic weapons

**Files to Modify** (~5 files, ~200 lines):
1. `internal/combat/calculations.go` — Unarmed damage formula
2. `internal/combat/combat.go` — Unarmed crit flavor text
3. Test files

**Testing**:
- [ ] **Unit Tests**: Test unarmed damage at various skill/strength levels
- [ ] **Manual Test**: Fight unarmed at low skill, verify low damage
- [ ] **Manual Test**: Train unarmed, verify damage increases noticeably
- [ ] **Balance Test**: Compare unarmed damage to basic weapon damage at equivalent skill

**Acceptance Criteria**:
- Unarmed damage scales clearly with skill and strength
- Unarmed is competitive with basic weapons when trained
- Flavor text makes unarmed combat feel distinct
- All tests pass

**Estimated Changes**: ~200 lines, 5 files

---

### Stage 7.4: Target Switching in Combat ✅ COMPLETED
**Merge Commit**: `dad9a71`
**Goal**: Allow players (and mobs) to switch combat targets during a fight, with a skill check determining success.

**Changes**:
1. Add `target <name>` command usable during combat to switch who you're attacking
2. Switching targets costs a round's worth of attacks (you spend the round repositioning)
3. Success chance based on combat skill + Dexterity — higher skill = smoother transition
4. Failed switch: you still attack your original target this round
5. Mobs with higher combat skill can also switch targets (e.g., target the healer)
6. Later (Phase 9+): positioning system will add more depth to target switching

**Files to Modify** (~6 files, ~250 lines):
1. `internal/usercommands/target.go` — New command (or modify `attack`)
2. `internal/combat/combat.go` — Target switch logic
3. `internal/combat/calculations.go` — Switch success roll
4. `internal/hooks/NewRound_DoCombat.go` — Mob target switching AI
5. Test files

**Testing**:
- [ ] **Manual Test**: Enter combat, switch targets mid-fight
- [ ] **Manual Test**: Verify round cost when switching
- [ ] **Manual Test**: Verify mobs sometimes switch targets
- [ ] **Regression Test**: Single-target combat still works normally

**Acceptance Criteria**:
- Players can switch targets during combat
- Switching has a cost (lost round) and skill check
- Mobs can intelligently switch targets
- All tests pass

**Estimated Changes**: ~250 lines, 6 files

---

### Stage 7.5: Combat-Specific Commands with Cooldowns ✅ COMPLETED
**Merge commit**: 53a6ca0 (includes comprehensive documentation)
**Goal**: Add special combat-only commands with cooldowns that provide tactical options beyond basic attacks. These include maneuvers like shield bash, trip, and kick that can apply status conditions to opponents.

**Design**:
- **Combat-only commands** — these commands are only available during active combat:
  - `bash` / `shield bash` — Uses shield to bash opponent, chance to knock them down (requires shield equipped)
  - `trip` — Attempt to trip opponent using Unarmed Combat or Weapon Combat
  - `kick` — Quick kick attack, lower damage but can knock down
- **Cooldown system** — each special move has a cooldown period (measured in combat rounds):
  - Shield bash: 3 rounds
  - Trip: 2 rounds
  - Kick: 2 rounds
- **Knockdown condition** — successful maneuvers have a chance to apply "Prone" condition:
  - **Success chance**: Opposed roll (attacker's skill + Strength vs defender's Dexterity + combat skill)
  - **Duration**: Target can attempt to stand up each round (costs stamina, uses their action for that round)
  - **Effects while Prone**:
    - Reduced attack success rate (-30%)
    - Reduced dodge/parry effectiveness (-50%)
    - Vulnerable to attacks from standing opponents (+20% to hit)
    - Cannot move or flee
    - Can still attack (at penalty) or attempt to stand
- **Spell interruption** — being knocked prone has a chance to interrupt spellcasting in progress (if casting time > 1 round)
- **Damage** — special moves deal reduced damage compared to normal attacks, but the tactical advantage of knockdown is the real benefit

**Changes**:
1. Create cooldown tracking system:
   - Add `CombatCooldowns map[string]int` to character state (tracks rounds until each move is available)
   - Decrement all cooldowns each combat round
   - Check cooldown before allowing special move
2. Add `Prone` condition flag to character state
3. Implement three new combat commands:
   - `internal/usercommands/bash.go` — Shield bash (requires shield)
   - `internal/usercommands/trip.go` — Trip attempt
   - `internal/usercommands/kick.go` — Kick attack
4. Add `stand` command to recover from Prone condition
5. Apply Prone condition effects to combat calculations:
   - Attack penalties in `internal/combat/calculations.go`
   - Defense penalties (dodge/parry/block)
   - Attack bonus for attackers vs prone targets
6. Add spell interruption logic when knocked prone during cast
7. Display cooldown status in combat prompt or via `cooldowns` command
8. Add config values for cooldown durations and prone effect magnitudes

**Files to Modify** (~12 files, ~600 lines):
1. `internal/characters/character.go` — Add `CombatCooldowns` map, `Prone` condition flag
2. `internal/usercommands/bash.go` — New command (shield bash)
3. `internal/usercommands/trip.go` — New command (trip)
4. `internal/usercommands/kick.go` — New command (kick)
5. `internal/usercommands/stand.go` — New command (stand up from prone)
6. `internal/combat/combat.go` — Integrate special moves, cooldown processing
7. `internal/combat/calculations.go` — Prone condition modifiers
8. `internal/spells/spells.go` — Spell interruption logic
9. `internal/hooks/NewRound_DoCombat.go` — Cooldown decrements, prone recovery attempts
10. `internal/configs/config.gameplay.go` — Add cooldown and prone effect configs
11. Test files

**Testing**:
- [ ] **Unit Tests**: Test cooldown tracking (set, decrement, check)
- [ ] **Unit Tests**: Test prone condition application and effects
- [ ] **Manual Test**: Use shield bash, verify cooldown prevents immediate reuse
- [ ] **Manual Test**: Trip opponent, verify prone condition applied
- [ ] **Manual Test**: Kick in combat, verify damage + knockdown chance
- [ ] **Manual Test**: While prone, verify attack/defense penalties apply
- [ ] **Manual Test**: Use stand command to recover from prone
- [ ] **Manual Test**: Knock down a spellcaster mid-cast, verify spell interruption
- [ ] **Manual Test**: Verify commands only work during combat (blocked outside combat)
- [ ] **Balance Test**: Verify special moves are tactically useful but not overpowered
- [ ] **Balance Test**: Verify prone condition is significant but recoverable

**Acceptance Criteria**:
- Three special combat commands (bash, trip, kick) functional with cooldowns
- Cooldown system prevents spam, displays remaining rounds
- Prone condition has meaningful combat effects
- Stand command allows recovery from prone
- Special moves only available during combat
- Spell interruption works when knocked prone while casting
- Config values allow tuning without code changes
- All tests pass

**Estimated Changes**: ~600 lines, 12 files

---

## Phase 8: Combat Positions & Integrated Grappling System

> **Design Philosophy**: Create an integrated combat position system that naturally weaves together striking, grappling, and ground fighting. Positions (Standing/Prone/Clinched/Grounded) affect attack speed, crit chance, and defense. Grappling flows organically from equipment, crits, and player choices rather than being a separate minigame.
>
> **Core Principles**:
> - **Gear-driven playstyles**: Weapon and armor choices naturally influence grappling effectiveness (daggers = agile grapplers, greatswords = terrible grapplers, heavy armor = hard to escape from ground)
> - **Simplified positions**: 4 states instead of complex body positioning (Standing, Prone, Clinched, Grounded)
> - **Organic special moves**: Disarms and submissions emerge from critical successes rather than explicit commands
> - **Risk/reward mechanics**: Grappling is powerful but risky (vulnerable to third parties, failure penalties)
> - **Z-score integration**: Uses existing crit system (z > 2.0 for crits, z < -2.0 for fumbles)

---

### Stage 8.1: Combat Position System (Replaces Prone Boolean) ✅ COMPLETED
**Commit**: `ace56e2` - feat: implement Stage 8.1 - Combat Position System

**Goal**: Replace the `Prone` boolean flag with a unified `CombatPosition` enum system. Migrate existing bash/trip mechanics to use positions. This lays the foundation for integrated grappling without changing combat behavior.

**Design**:
Four combat positions:
- **Standing**: Default state, on feet, normal combat
- **Prone**: Knocked down (bash/trip), scrambling to stand, vulnerable to grappling
- **Clinched**: Standing grapple, reduced attack speed, automatic control checks
- **Grounded**: Ground grapple, very slow attacks, controller has advantage

**Position Transitions** (Stage 8.1 scope):
```
Standing ──[bash/trip]──> Prone ──[recovery]──> Standing
         (existing mechanics, just using new position system)
```

**Changes**:
1. Add `CombatPosition` enum type with 4 constants
2. Replace `Prone bool` with `CombatPosition CombatPositionType` in Character struct
3. Rename `ProneRoundsRemaining` → `PositionRoundsMin` (more general name)
4. Add `GrappleControllerId int` field (preparation for 8.2, not used yet)
5. Migrate bash/trip commands: set `CombatPosition = PositionProne` instead of `Prone = true`
6. Update all prone penalty checks: `if c.Prone` → `if c.CombatPosition == PositionProne`
7. Update recovery logic to check position
8. Update character adjectives: "prone" only when position is Prone
9. Display position in `score` command

**Files to Modify** (~8 files, ~200 lines):
1. `internal/characters/character.go` — Replace Prone fields with CombatPosition
2. `internal/characters/combatposition.go` — New file: position enum and constants
3. `internal/combat/combat.go` — Update all `if c.Prone` checks to use CombatPosition
4. `internal/usercommands/bash.go` — Set CombatPosition instead of Prone flag
5. `internal/usercommands/trip.go` — Set CombatPosition instead of Prone flag
6. `internal/usercommands/score.go` — Display combat position
7. Character recovery logic — Check CombatPosition == PositionProne
8. Test files

**Testing**:
- [x] **Unit Tests**: Position enum values and transitions
- [x] **Manual Test**: Bash opponent, verify they show as "prone" in adjectives
- [x] **Manual Test**: Verify recovery still works (prone → standing after rounds)
- [x] **Manual Test**: Verify prone attack/defense penalties still apply
- [x] **Manual Test**: Position shows in `look` command and `{pos}` prompt tag
- [x] **Regression Test**: All Stage 7.5 prone functionality still works identically
- [x] **Integration Test**: Bash → wait → recover → verify standing
- [x] **Bug Fix**: Fixed cooldowns not persisting (changed to pointer receivers)

**Acceptance Criteria**:
- Prone system works identically to before (zero behavior change)
- All bash/trip tests pass
- Position displays in score and adjectives
- Recovery mechanics unchanged
- All existing tests pass
- Code is cleaner (no boolean + rounds tracking, just position state)

**Estimated Changes**: ~200 lines, 8 files

---

### Stage 8.2: Grapple Command & Basic Position Transitions ✅ COMPLETED
**Commit**: `ed58029` - Merge feature/stage-8.2-grapple-command into development

**Goal**: Add `grapple` command with cooldown. Successful grapples transition positions (Standing→Clinched, Prone→Grounded). Grapplers and grappled opponents are marked by controller tracking.

**Design**:
- `grapple <target>` initiates opposed check (Dex + Combat Skill + weapon modifier vs Dex + Combat Skill)
- **Huge advantage when grappling prone targets**: Defender at -70% if prone (nasty!)
- Success: `Standing → Clinched` OR `Prone → Grounded` (direct, skip Clinched)
- 5-round cooldown on grapple attempts
- `GrappleControllerId` tracks who initiated/controls the grapple

**Grapple Calculation**:
```go
attackScore = attacker.Dex + attacker.CombatSkill + weapon.GrappleModifier
defenseScore = defender.Dex + defender.CombatSkill

// Position modifiers
if defender.CombatPosition == PositionProne {
    defenseScore *= 0.3  // -70% defense when already down (brutal!)
}
if attacker.CombatPosition == PositionProne {
    attackScore *= 0.5   // -50% offense when attacking from ground
}

// Opposed z-score roll
success = OpposedRoll(attackScore, defenseScore)
```

**Changes**:
1. Add `grapple <target>` user command with cooldown check
2. Create `internal/combat/grapple.go` with grapple calculation logic
3. Add `GrappleModifier float64` field to `ItemSpec` (weapon property)
4. Position transitions: Standing→Clinched, Prone→Grounded
5. Set `GrappleControllerId` to track who initiated the grapple
6. Add 5-round cooldown: `character.Cooldowns["grapple"] = 5`
7. Grapple success/failure messaging
8. Update a few weapon YAMLs with test grapple modifiers

**Files to Modify** (~10 files, ~300 lines):
1. `internal/usercommands/grapple.go` — New command
2. `internal/combat/grapple.go` — New file: grapple calculations and transitions
3. `internal/items/itemspec.go` — Add GrappleModifier field
4. `internal/characters/character.go` — GrappleControllerId field (added in 8.1)
5. Weapon YAML files — Add grapple_modifier to 3-4 test weapons
6. Test files

**Testing**:
- [ ] **Manual Test**: Grapple from Standing, verify transition to Clinched
- [ ] **Manual Test**: Grapple a prone target, verify transition to Grounded (and it's easy!)
- [ ] **Manual Test**: Try to grapple while on cooldown, verify blocked
- [ ] **Manual Test**: Grapple with dagger vs greatsword, verify modifier difference
- [ ] **Balance Test**: Prone targets should be MUCH easier to grapple

**Acceptance Criteria**:
- Grapple command functional with cooldown
- Position transitions work (Standing→Clinched, Prone→Grounded)
- Prone targets are very vulnerable to grappling
- Weapon modifiers apply
- Clear messaging for success/failure
- All tests pass

**Estimated Changes**: ~300 lines, 10 files

---

### Stage 8.3: Position-Based Attack Speed & Auto-Progression ✅ COMPLETED
**Commit**: `170e9ec` - Merge feature/stage-8.3-position-combat-effects into development

**Goal**: Make positions affect combat speed. Add automatic control checks (Clinched→Grounded) and escape checks (Grounded→Standing) each round.

**Design**:
**Attack Speed Multipliers**:
```
Standing: 1.0x (normal)
Prone:    0.5x (existing penalty, now formalized by position)
Clinched: 0.6x (both fighters locked up, slow strikes)
Grounded: 0.3x (both fighters very limited movement)
```

**Crit Chance Modifiers**:
```
Grounded controller: +10% crit (dominant position)
Grounded controlled: -10% crit (defensive position)
Clinched controller: +5% crit (slight advantage)
```

**Auto-Progression Each Round**:
- **Clinched**: Automatic control check (Str + Combat Skill opposed roll)
  - Success → Transition to Grounded (controller maintains control)
  - Failure → Break apart, both return to Standing
- **Grounded**: Automatic escape attempt by controlled fighter
  - Success → Both return to Standing (scramble up together)
  - Failure → Remain Grounded

**Changes**:
1. Apply attack speed multipliers in `calculateCombat()` based on `CombatPosition`
2. Apply crit threshold modifiers based on position and controller status
3. Add automatic control check logic in `NewRound_DoCombat` hook
4. Add automatic escape check logic in `NewRound_DoCombat` hook
5. Control/escape calculations use Strength + Combat Skill opposed rolls
6. Position transition messaging ("You drive Guard to the ground!", "You break free!")

**Files to Modify** (~10 files, ~350 lines):
1. `internal/combat/combat.go` — Apply position attack speed multipliers
2. `internal/combat/combat.go` — Apply position crit modifiers
3. `internal/combat/grapple.go` — Add ControlCheck() and EscapeCheck() functions
4. `internal/hooks/NewRound_DoCombat.go` — Call auto-checks for Clinched/Grounded positions
5. `internal/characters/character.go` — Helper methods for position checks
6. Test files

**Testing**:
- [ ] **Manual Test**: Enter Clinched, verify slower attacks (0.6x speed)
- [ ] **Manual Test**: Stay Clinched multiple rounds, verify auto control check
- [ ] **Manual Test**: Successful control check, verify transition to Grounded
- [ ] **Manual Test**: Enter Grounded, verify very slow attacks (0.3x speed)
- [ ] **Manual Test**: Grounded controlled fighter auto-escapes eventually
- [ ] **Manual Test**: Verify controller has higher crit chance when Grounded
- [ ] **Balance Test**: Typical grapple sequence (Clinched 1-2 rounds → Grounded 2-4 rounds)

**Acceptance Criteria**:
- Attack speed reduced appropriately by position
- Clinched fighters automatically progress to Grounded or break apart
- Grounded fighters have escape opportunities each round
- Crit modifiers make controller position advantageous
- Fights don't get permanently stuck in grapples
- All tests pass

**Estimated Changes**: ~350 lines, 10 files

---

### Stage 8.4: Crit Outcomes (Disarm, Dodge Opportunities) ✅ COMPLETED
**Merge Commit**: 7d17921 (2026-02-13)
**Goal**: Add organic special moves triggered by critical successes: disarms from grapple/parry crits, grapple opportunities from dodge crits.

**Design**:
**Disarm Mechanics**:
1. **Grapple Crit Disarm** (z > 2.0 while Clinched/Grounded):
   - 15% chance to disarm opponent
   - Weapon drops to room floor
   - Fallback to unarmed combat (secondary weapon system deferred)
   - Message: "You wrench the iron longsword from Guard's grip! It clatters to the floor."

2. **Parry Crit Disarm** (z > 2.0 on successful parry):
   - 10% chance to disarm attacker
   - Represents a perfect parry that twists weapon away
   - Message: "You parry and twist, disarming Guard!"

**Dodge Crit → Grapple Opportunity**:
- On dodge with z > 2.0: Set `GrappleOpportunity = true` (1-round flag)
- **Only if grapple cooldown is available** (prevents spam)
- Next grapple attempt gets +15% bonus
- Flag expires after 1 round if not used
- Message: "You slip inside Guard's guard! [Grapple opportunity]"

**Disarm Fallback** (simplified for now):
- Disarmed → equip unarmed (fall back to fists/claws)
- Secondary weapon system deferred to later stage
- Certain weapons can be flagged `disarm_immune` (natural weapons, magical bindings)

**Changes**:
1. Detect z-score > 2.0 in grapple checks, trigger disarm chance
2. Detect z-score > 2.0 in parry defense, trigger disarm chance
3. Detect z-score > 2.0 in dodge defense, set GrappleOpportunity flag (if cooldown available)
4. Add `GrappleOpportunity bool` flag to Character (expires each round)
5. Apply +15% bonus to grapple checks when opportunity flag is set
6. Disarm logic: drop weapon to room, switch to unarmed
7. Add messaging for all special outcomes

**Files to Modify** (~12 files, ~350 lines):
1. `internal/combat/combat.go` — Detect parry/dodge crits, trigger outcomes
2. `internal/combat/grapple.go` — Detect grapple crits, trigger disarm, apply opportunity bonus
3. `internal/characters/character.go` — Add GrappleOpportunity flag, disarm helper
4. `internal/items/item.go` — Drop weapon to room logic
5. `internal/hooks/NewRound_DoCombat.go` — Expire GrappleOpportunity flag each round
6. Test files

**Testing**:
- [x] **Manual Test**: Get grapple crit, verify occasional disarm (debug logging active)
- [x] **Manual Test**: Get parry crit, verify occasional disarm (debug logging active)
- [x] **Manual Test**: Disarmed fighter switches to unarmed combat
- [x] **Manual Test**: Crit dodge, verify grapple opportunity message (if cooldown up)
- [x] **Manual Test**: Use grapple opportunity, verify +15% bonus
- [x] **Manual Test**: Ignore opportunity, verify it expires next round
- [x] **Balance Test**: Special outcomes are rare but impactful (z > 2.0 = ~2.3% of rolls)

**Acceptance Criteria**:
- ✅ Disarms occur organically from crits (not explicit command)
- ✅ Grapple opportunities create reactive tactical choices
- ✅ Disarmed fighters can continue fighting (unarmed)
- ✅ Opportunity window is short (1 round) and requires cooldown
- ✅ Clear messaging for all special outcomes
- ✅ All tests pass
- ✅ Debug logging added for testing crit detection

**Actual Changes**: ~270 lines, 8 files (6 modified + 1 new + 1 test)

---

### Stage 8.5: Multi-Combatant Grappling Penalties ✅ COMPLETED
**Commit**: `97085f2` - feat: implement Stage 8.5 third-party grapple vulnerability
**Goal**: Make grappling risky in group combat. Both grappler and grappled are vulnerable to third-party attacks.

**Design**:
**Vulnerability When Clinched/Grounded**:
- Both fighters: -30% defense against third-party attacks
- Both fighters: Cannot parry/dodge attacks from others (too focused on grapple)
- Attacks from third parties bypass layered defense system (auto-hit against static defense)

**Pile-On Bonus** (optional):
- Additional attackers get +20% hit chance against Grounded targets
- Simulates multiple people attacking a downed opponent

**Messaging**:
- "The goblin is too entangled with you to defend against the orc's attack!"
- "You're too focused on holding the guard down to dodge the bandit's strike!"

**Changes**:
1. In `calculateCombat()`: Check if target is Clinched/Grounded
2. If attacker is NOT the grapple controller/controlled: apply third-party penalties
3. Disable dodge/parry defense for entangled fighters (vs third parties only)
4. Apply -30% defense modifier against third-party attacks
5. Optional: Apply +20% attack bonus for attackers hitting Grounded targets
6. Add third-party vulnerability messaging

**Files to Modify** (~8 files, ~200 lines):
1. `internal/combat/combat.go` — Detect third-party attacks, apply penalties
2. `internal/combat/calculations.go` — Third-party attack modifiers
3. `internal/combat/grapple.go` — Helper to check if attack is third-party
4. Test files

**Testing**:
- [x] **Manual Test**: 3-combatant fight, grapple one opponent
- [x] **Manual Test**: Third party attacks grappler, verify easier to hit
- [x] **Manual Test**: Third party attacks grappled target, verify easier to hit
- [x] **Manual Test**: Verify defense messages ("too entangled to dodge")
- [x] **Balance Test**: Grappling in group combat is high-risk

**Acceptance Criteria**:
- Both grappler and grappled are vulnerable to third parties
- Defense penalties are meaningful (grappling in groups is risky)
- Clear messaging explains why defense failed
- Tactical choice: grapple in 1v1 good, in groups risky
- All tests pass

**Estimated Changes**: ~200 lines, 8 files

---

### Stage 8.6: Submission Command & Failed Grapple Penalties ✅ COMPLETED
**Goal**: Add high-risk submission finishing move. Add meaningful penalties for failed grapple attempts (risk/reward).

**Merge Commit**: e829213

**Design**:
**Submission Mechanic**:
- `submit` command (only available when Grounded as controller)
- **Requires grapple cooldown available** (prevents grapple→submit spam)
- High-threshold opposed check (harder than normal grapple)
- Success: Opponent chooses [yield] or [resist]
  - Yield: Combat ends, they're at your mercy
  - Resist: Take 2x damage, make escape check, likely break free
- Failure: Opponent auto-escapes to Standing, you fall Prone (overcommitted)
- Sets 5-round cooldown (burned whether success or fail)
- Message: "You overcommit to the armbar and Guard scrambles free, leaving you prone!"

**Failed Grapple Penalties** (risk/reward):
```
Grapple outcomes:
  Success (z > 0.5):     Clinched (or Grounded if target was Prone)
  Failure (z < 0.5):     -15% defense next round (exposed, off-balance)
  Crit Failure (z < -2.0): Fall Prone, opponent gets reversal opportunity
```

**Reversal Opportunity**:
- On crit failed grapple: Defender can immediately attempt grapple (if cooldown up)
- Represents capitalizing on attacker's mistake
- Message: "Guard capitalizes on your failed takedown and grabs you!"

**Changes**:
1. Add `submit` user command with position check (must be Grounded controller)
2. Check grapple cooldown available
3. High-threshold submission check logic
4. Opponent choice prompt: [yield] or [resist]
5. Failure consequence: Opponent escapes, attacker falls Prone
6. Add failed grapple defense penalty (-15% next round)
7. Add crit failed grapple: attacker falls Prone, defender gets reversal
8. Set 5-round cooldown on submit attempt
9. Add to combat help file

**Files to Modify** (~10 files, ~300 lines):
1. `internal/usercommands/submit.go` — New command
2. `internal/combat/grapple.go` — Submission logic, failure penalties, reversals
3. `internal/characters/character.go` — Track defense penalty flag
4. Help files — Add grapple/submit documentation
5. Test files

**Testing**:
- [ ] **Manual Test**: Submit while Grounded as controller, verify check
- [ ] **Manual Test**: Succeed submit, verify opponent gets yield/resist choice
- [ ] **Manual Test**: Opponent resists, verify 2x damage + escape
- [ ] **Manual Test**: Fail submit, verify auto-escape and you fall Prone
- [ ] **Manual Test**: Crit fail grapple, verify you fall Prone
- [ ] **Manual Test**: Crit fail grapple, opponent capitalizes with reversal
- [ ] **Manual Test**: Try to submit when cooldown not ready, verify blocked
- [ ] **Balance Test**: Submit is powerful but very risky

**Acceptance Criteria**:
- Submit is a high-stakes finishing move
- Failed grapples have meaningful penalties (not free attempts)
- Crit failures create dramatic reversals
- Cooldown prevents grapple→submit spam
- Clear messaging for all outcomes
- All tests pass

**Estimated Changes**: ~300 lines, 10 files

---

### Stage 8.7: Weapon & Armor Grapple Modifiers (Data-Driven)
**Goal**: Add grapple modifiers to weapon and armor data files. Equipment naturally dictates grappling effectiveness.

**Design**:
**Weapon Grapple Modifiers**:
```yaml
# Agile, close-quarters weapons
Unarmed/Claws:  1.3  (natural grapplers)
Dagger:         1.2  (easy to maintain grip)
Shortsword:     1.0  (neutral)

# Unwieldy weapons
Longsword:      0.7  (harder to grapple)
Greatsword:     0.4  (terrible for grappling)
Polearm:        0.3  (completely unwieldy)

# Special
Net:            0.8  (hands full, but see below)
```

**Net Special Mechanic** (deferred):
- Treat net as throwing weapon for now
- Entangle condition deferred to later stage (8.8 or Phase 9)
- Future: Net crit → "Entangled" condition (like Prone but requires cutting free)

**Armor Escape Modifiers**:
```yaml
# Affects escape checks when Grounded
Light/None:  +1.0  (agile, easy to stand)
Medium:       0.0  (neutral)
Heavy/Plate: -2.0  (historically accurate - stuck like a turtle!)
```

**Changes**:
1. `GrappleModifier` field already added in Stage 8.2 to ItemSpec
2. Add `EscapeModifier` field to armor ItemSpec
3. Update ~15 weapon YAML files with grapple_modifier values
4. Update ~5 armor YAML files with escape_modifier values
5. Apply armor escape modifier in escape checks
6. Net deferred (mark as throwing weapon, entangle mechanic later)

**Files to Modify** (~25 files, ~150 lines):
1. `internal/items/itemspec.go` — Add EscapeModifier field for armor
2. `internal/combat/grapple.go` — Use armor escape modifier in checks
3. Weapon YAML files — Add grapple_modifier (~15 files, 2-3 lines each)
4. Armor YAML files — Add escape_modifier (~5 files, 2-3 lines each)
5. Test files

**Testing**:
- [ ] **Manual Test**: Grapple with dagger, verify easier than greatsword
- [ ] **Manual Test**: Try to escape in plate armor, verify harder than leather
- [ ] **Manual Test**: Grapple opponent in heavy armor, verify they struggle to escape
- [ ] **Balance Test**: Equipment creates meaningful playstyle differences

**Acceptance Criteria**:
- Weapon modifiers create distinct grappling effectiveness
- Armor modifiers make heavy armor a liability on the ground
- Equipment choices have strategic grappling implications
- Data-driven (easy to tune via YAML)
- All tests pass

**Estimated Changes**: ~150 lines, 25 files

---

## Phase 8 Summary

| Stage | Focus | Lines | Files | Status |
|-------|-------|-------|-------|--------|
| 8.1 | Position system (replace Prone boolean) | ~200 | 10 | ✅ Complete (ace56e2) |
| 8.2 | Grapple command + transitions | ~300 | 10 | ✅ Complete (170e9ec) |
| 8.3 | Attack speed + auto-progression | ~350 | 10 | ✅ Complete (3f53cac) |
| 8.4 | Crit outcomes (disarm, opportunities) | ~350 | 12 | ✅ Complete (7d17921) |
| 8.5 | Multi-combatant penalties | ~200 | 8 | ✅ Complete (97085f2) |
| 8.6 | Submissions + failure penalties | ~300 | 10 | ✅ Complete (e829213) |
| 8.7 | Weapon/armor modifiers (data) | ~150 | 25 | ✅ Complete (737be35) |
| 8.8 | **HOTFIX**: Auto-aggro when attacked | ~50 | 3 | **KNOWN BUG** |
| **Total** | **Integrated grappling system** | **~1900** | **~61 unique** | **Phase 8** |

**Phase Completion**: After Stage 8.7, combat system includes:
- Position-based mechanics (Standing/Prone/Clinched/Grounded)
- Grappling integrated with striking
- Organic special moves (disarms, submissions)
- Equipment-driven playstyles
- Multi-combatant tactics
- Risk/reward grappling decisions

### Stage 8.8: Hotfix - Auto-Aggro When Attacked (KNOWN BUG)
**Goal**: Fix player not auto-aggroing back when attacked by respawned/spawned mobs

**Bug Description**:
When a mob respawns in the same room as a player (or otherwise becomes aggro with a player):
- Mob.Character.Aggro is set → mob attacks player
- Player.Character.Aggro is NOT set → player doesn't attack back
- Player takes damage without fighting back
- Special move commands (grapple, etc.) fail: "You must be in combat!"
- Player must manually type `attack <mob>` to start fighting

**Root Cause**:
Asymmetric aggro state - mob spawning/aggro logic sets mob's aggro but doesn't set player's aggro reciprocally.

**Fix Required**:
1. When mob sets aggro on player (in spawn/idle/patrol logic), also set player's aggro on mob
2. Check mob spawn code, idle action code, and aggro-setting code
3. Ensure reciprocal aggro: if A attacks B, then B should auto-attack A back
4. May need to check `handleMobCombat()` or mob AI routines

**Files to Check**:
1. `internal/mobs/mobs.go` - Spawn/respawn logic
2. `internal/mobs/*.go` - Idle/patrol/aggro logic
3. `internal/hooks/NewRound_*.go` - Round processing
4. Anywhere `mob.Character.SetAggro()` is called without reciprocal player aggro

**Priority**: High - breaks combat flow, very noticeable to players

**Testing**:
- [x] Wait for mob to respawn in same room → verify auto-aggro back
- [x] Hostile mob enters room → verify auto-aggro back
- [x] Special moves work without manual attack command

**Status**: ✅ COMPLETED - Added reciprocal aggro in `NewRound_DoCombat.go:932-935`

---

### Stage 8.9: NPC Combat AI - Special Moves
**Goal**: Implement intelligent NPC AI for using special combat moves (bash, trip, kick, grapple, submit) with generic defaults and per-mob customization

**Current State**:
- Mobs have `CombatCommands []string` field for random combat actions
- `ActivityLevel` (1-100%) determines action frequency
- Target switching AI exists for skilled mobs (combat skill >= 30)
- No AI for choosing special moves contextually

**Design Philosophy**:
- **Default AI**: Smart, context-aware decision making for all NPCs
- **Configurable**: YAML-based customization per mob type
- **Skill-scaled**: Higher combat skills = better tactical decisions
- **Context-aware**: Considers health, position, grapple state, weapon type

**Architecture**:

1. **New File**: `internal/combat/ai.go`
   - `ChooseSpecialMove(mob, target) string` - Main AI decision function
   - `EvaluateMoveViability(mob, target, moveType) int` - Score each move (0-100)
   - `GetMovePreferences(mob) map[string]int` - Get mob's move weights

2. **New Mob Fields** (optional, for customization):
   ```go
   // Combat AI configuration
   AIProfile string // "defensive", "aggressive", "grappler", "brawler", "tactical"
   MovePreferences map[string]int // Custom weights: {"bash": 30, "grapple": 50, "kick": 20}
   SpecialMoveChance int // Base % chance to attempt special move (default: 30%)
   ```

3. **YAML Configuration Example**:
   ```yaml
   mobid: 15
   aiprofile: "grappler"  # Prefers grappling moves
   specialmovechance: 50   # 50% chance to use special moves
   movepreferences:
     grapple: 60
     submit: 40
     bash: 10
   ```

**AI Decision Flow**:

```
Each combat round:
1. Check ActivityLevel (existing mechanic)
2. Roll against SpecialMoveChance (default 30%, scaled by combat skill)
3. If yes, evaluate all viable special moves:
   - Bash: Good if target standing, mob has bludgeon weapon, target health > 50%
   - Trip: Good if target standing, mob has decent dexterity, close combat
   - Kick: Good if target standing, mob unarmed or light weapon
   - Grapple: Good if both standing, mob has wrestling skill, wants control
   - Submit: Only if controller in dominant grapple position
   - Escape: Only if grappled and in bad position
4. Score each move based on:
   - Viability (can it be used now?)
   - Effectiveness (mob stats/skills favor it?)
   - Tactical value (current combat state)
   - Preference weight (from AI profile/YAML)
5. Weight scores by preferences, pick highest
6. Execute move command
7. If no special move chosen, use regular attack/CombatCommands
```

**AI Profiles** (Default Behavior Templates):

- **`default`**: Balanced mix, 30% special move chance
  - Standing: bash 25%, trip 20%, kick 15%, grapple 40%
  - Grappled: escape 60%, submit 40%

- **`aggressive`**: High damage, rushes fights, 40% special move chance
  - Prefers: bash 40%, kick 30%, trip 20%, grapple 10%
  - Tends toward damage over control

- **`defensive`**: Cautious, uses control moves, 25% special move chance
  - Prefers: trip 35%, grapple 35%, bash 20%, kick 10%
  - Higher threshold for submission

- **`grappler`**: Wrestling specialist, 50% special move chance
  - Prefers: grapple 60%, submit 30%, trip 10%
  - Aggressive position progression

- **`brawler`**: Unarmed specialist, 45% special move chance
  - Prefers: kick 40%, trip 30%, grapple 20%, bash 10%
  - Favors unarmed moves

- **`tactical`**: Smart fighter, adapts to situation, 35% special move chance
  - High variance based on target state
  - Uses bash on healthy targets, submit on grappled, trip on low-dex targets

**Move Viability Scoring** (0-100 scale):

**Bash**:
- Base: 50
- +30 if wielding bludgeon weapon
- +20 if target health > 60%
- +15 if combat skill > 50
- -50 if target prone
- -100 if no weapon or wrong damage type

**Trip**:
- Base: 40
- +25 if mob dexterity > 14
- +20 if target dexterity < 10
- +15 if mob has unarmed-combat skill > 40
- -100 if target already prone
- -50 if in grapple

**Kick**:
- Base: 45
- +30 if unarmed or light weapon
- +20 if mob unarmed-combat > 50
- +15 if mob strength > 14
- -100 if target prone
- -30 if wielding heavy weapon

**Grapple**:
- Base: 50
- +30 if mob wrestling > 40
- +20 if mob strength > target strength
- +15 if target health < 30% (finish them)
- -100 if already in grapple
- -50 if mob health < 20% (too risky)

**Submit** (only when grapple controller):
- Base: 40
- +40 if in dominant position (mounted, standing over prone)
- +20 if mob wrestling > 60
- +15 if target health < 40%
- -100 if not controller
- -100 if not in grapple

**Escape** (only when grappled and not controller):
- Base: 60
- +30 if mob health < 30% (desperate)
- +20 if mob strength > target strength
- +15 if mob dexterity > 14
- -100 if controller
- -40 if in favorable position

**Implementation Steps**:

1. **Create `internal/combat/ai.go`**:
   - Define AI profiles (constants/structs)
   - Implement `ChooseSpecialMove()` function
   - Implement scoring functions for each move type
   - Load preferences from mob YAML

2. **Update `internal/mobs/mobs.go`**:
   - Add optional AI fields to Mob struct
   - Default `AIProfile = "default"`
   - Default `SpecialMoveChance = 30`

3. **Integrate into `internal/hooks/NewRound_DoCombat.go`**:
   - In mob combat loop (around line 869-898, existing combat command logic)
   - Before checking `CombatCommands`, call `ai.ChooseSpecialMove()`
   - If special move chosen, execute it
   - Otherwise fall back to existing CombatCommands or standard attack

4. **YAML Loader Support** (if needed):
   - Update YAML parsing to load `aiprofile`, `specialmovechance`, `movepreferences`
   - Gracefully handle missing fields (use defaults)

5. **Testing**:
   - Create test mobs with different AI profiles
   - Verify moves are chosen contextually
   - Ensure customization via YAML works
   - Test that defaults apply when no profile specified

**Files to Modify**:
- `internal/combat/ai.go` (NEW)
- `internal/mobs/mobs.go` (add AI fields)
- `internal/hooks/NewRound_DoCombat.go` (integrate AI)
- `_datafiles/world/dogmud/mobs/*.yaml` (optional, for testing custom AI)

**Testing Checklist**:
- [ ] Default AI profile works for mobs without YAML config
- [ ] Aggressive AI prefers bash/kick over grapple
- [ ] Grappler AI initiates and maintains grapples
- [ ] Defensive AI uses trip/control moves appropriately
- [ ] Tactical AI adapts to combat state (health, position)
- [ ] Custom YAML preferences override defaults
- [ ] AI respects move viability (no bash when prone, no submit when not controller)
- [ ] ActivityLevel still gates AI decisions
- [ ] Combat skill scales decision quality (low skill = simpler tactics)

**Expected Behavior After Implementation**:
- Guards use defensive tactics (trip, control)
- Brawlers/thugs use kicks and bashes
- Wrestlers/grapplers attempt submissions
- Skilled fighters adapt to combat flow
- Low-skill mobs mostly use basic attacks with occasional special moves
- Boss mobs can be configured with custom move sets

---

## Phase 9: Combat Presentation Overhaul

> **Goal**: Transform combat from a "you hit for X damage" number game into an immersive,
> descriptive experience. Inspired by the Evennia DOG combat system
> (see `combat_example_evennia.png`). No raw damage numbers shown to players.

### Stage 9.1: Descriptive Damage Text — Remove Numbers
**Goal**: Replace all numeric damage output ("You hit Guard for 4 damage") with descriptive text based on damage severity and attack type.

**Design**:
- Damage ranges map to description tiers:
  - **Negligible** (0-5% max HP): "with little effect", "barely scratching"
  - **Light** (5-15%): colored "scratched", "nicked"
  - **Moderate** (15-30%): colored "wounded"
  - **Heavy** (30-50%): colored "seriously wounded"
  - **Severe** (50-75%): colored "critically wounded"
  - **Devastating** (75%+): colored "obliterated", "brutally wounded"
- Attack descriptions vary by weapon/unarmed type:
  - Sword: "slash", "uppercut slash", "thrust"
  - Unarmed: "jab", "roundhouse kick", "elbow strike", "shoulder check"
  - Bow: "arrow strikes", "bolt pierces"
- Messages differ by perspective (attacker, defender, observer) like the Evennia system
- Color coding: orange/red for hits, cyan for dodges, green for blocks (with text prefixes for screen readers)

**Changes**:
1. Create a damage description system with tier lookups
2. Create attack flavor text dictionaries per weapon type / unarmed
3. Replace all `fmt.Sprintf("you hit for %d damage")` with descriptive text calls
4. Add perspective-based messaging (1st person, 2nd person, 3rd person observer)
5. Add color coding with screen-reader-friendly prefixes

**Files to Modify** (~10 files, ~600 lines):
1. `internal/combat/messages.go` — New file: damage descriptions, attack flavor text
2. `internal/combat/combat.go` — Replace numeric damage output
3. `internal/combat/calculations.go` — Add description tier calculation
4. Weapon YAML or combat data — Attack flavor text per weapon type
5. Test files

**Testing**:
- [ ] **Manual Test**: Fight with sword, verify descriptive slash/thrust messages
- [ ] **Manual Test**: Fight unarmed, verify kick/punch/elbow messages
- [ ] **Manual Test**: Verify no raw damage numbers appear anywhere in combat output
- [ ] **Manual Test**: Verify color coding (orange hits, cyan dodges, green blocks)
- [ ] **Manual Test**: Verify screen reader mode still conveys the same information

**Acceptance Criteria**:
- Zero numeric damage in player-visible combat output
- Descriptive text clearly conveys damage severity via color and wording
- Attack descriptions vary by weapon type
- Screen reader accessibility maintained
- All tests pass

**Estimated Changes**: ~600 lines, 10 files

---

### Stage 9.2: Health/Stamina/Conviction Bars in Prompt
**Goal**: Replace numeric HP/Stamina/Conviction in the player prompt with colored progress bars, matching the Evennia DOG style (see `combat_example_evennia.png`).

**Design**:
- Format: `Health:[████████░░░░] Stamina:[████████████░] Conviction:[████████████████]`
- Color gradient: green (>60%) → yellow (30-60%) → red (<30%)
- Bars scale proportionally to max value
- Width configurable (default ~16 characters per bar)
- Shown every round during combat, and on `score`/`status` outside combat

**Changes**:
1. Create bar rendering function: `RenderBar(current, max, width) string`
2. Integrate bars into the combat round prompt
3. Update the prompt template to use bars instead of numbers
4. Ensure proper ANSI color support and screen reader fallback (e.g., "Health: 75%")

**Files to Modify** (~6 files, ~250 lines):
1. `internal/templates/` or `internal/term/` — Bar rendering utility
2. `internal/prompt/` — Update prompt generation
3. Prompt template files — Use bar format
4. `internal/usercommands/score.go` — Optional bar display
5. Test files

**Testing**:
- [ ] **Unit Tests**: Test bar rendering at various fill levels
- [ ] **Manual Test**: Take damage, verify bar shrinks and changes color
- [ ] **Manual Test**: Verify bars display correctly in combat prompt
- [ ] **Manual Test**: Verify screen reader mode shows percentages instead of bars

**Acceptance Criteria**:
- Colored progress bars for all three resource pools
- Color changes based on fill level
- Screen reader fallback
- Bars update in real-time during combat
- All tests pass

**Estimated Changes**: ~250 lines, 6 files

---

### Stage 9.3: Configurable Combat Prompt
**Goal**: Let players configure what information appears in their combat prompt. Options include: current tank, current target, relative health of target, and the resource bars from 9.2.

**Design**:
- `prompt set` command to toggle prompt elements:
  - `tank` — Show who is currently tanking (taking hits) in your group
  - `target` — Show who you are currently attacking
  - `targethealth` — Show relative health of your target (descriptive, not numeric: "healthy", "wounded", "near death")
  - `bars` — Toggle resource bars on/off (default: on)
  - `compact` — Single-line vs multi-line prompt
- Settings saved to character data, persist across sessions
- Default prompt shows: resource bars + target name + target condition
- Example: `Health:[████████░░░░] Stamina:[████████████░] | Target: Guard (wounded) | Tank: You`

**Changes**:
1. Add `PromptConfig` struct to character data with toggle fields
2. Add `prompt` command to configure settings
3. Update prompt rendering to conditionally include elements
4. Add target condition descriptions ("healthy", "bruised", "wounded", "critical", "near death")
5. Add tank detection for group play

**Files to Modify** (~8 files, ~350 lines):
1. `internal/characters/character.go` — Add `PromptConfig` struct
2. `internal/usercommands/prompt.go` — New command
3. `internal/prompt/` — Conditional prompt rendering
4. Prompt template files — Support configurable sections
5. Test files

**Testing**:
- [ ] **Manual Test**: Configure prompt to show target, verify it appears
- [ ] **Manual Test**: Toggle tank display on/off
- [ ] **Manual Test**: Verify target health description updates as target takes damage
- [ ] **Manual Test**: Save/quit/reload, verify prompt settings persist

**Acceptance Criteria**:
- All prompt elements are independently toggleable
- Settings persist across sessions
- Target health shown as descriptive text (not numbers)
- Default prompt is sensible for new players
- All tests pass

**Estimated Changes**: ~350 lines, 8 files

---

### Stage 9.4: Combat Conditions System Refactor
**Goal**: Consolidate scattered boolean flags (RecoveryPenaltyThisRound, DefensePenaltyNextRound, etc.) into a unified, configurable combat conditions system. Make it easy to add new temporary combat states.

**Problem**:
As combat has evolved through Stages 7-8, we've added individual boolean flags for various temporary states:
- `RecoveryPenaltyThisRound` (Stage 7.5: prone recovery)
- `DefensePenaltyNextRound` (Stage 8.6: failed grapple)
- `IsGrappleController` (Stage 8.3: grapple state tracking)
- Various timer-based states (GrappleOpportunity, etc.)

Each flag requires:
- Field in Character struct
- Manual clearing in round tick hooks (both user and mob)
- Individual application logic scattered across combat.go

This is fragile, verbose, and error-prone. Adding a new temporary state requires touching 4+ files.

**Solution**: Unified Conditions System
```go
type CombatCondition struct {
    Type        ConditionType  // enum: RecoveryPenalty, DefensePenalty, GrappleOpportunity, etc.
    Duration    int            // rounds remaining (0 = permanent until cleared)
    Magnitude   float64        // effect strength (e.g., -0.15 for defense penalty)
    Source      string         // what caused it (for debugging/messages)
    Metadata    map[string]any // flexible data for condition-specific state
}

type ConditionType int
const (
    ConditionRecoveryPenalty ConditionType = iota  // -1 attack this round
    ConditionDefensePenalty                         // -15% defense next round
    ConditionGrappleOpportunity                     // +15% grapple bonus
    ConditionGrappleController                      // is the grapple controller
    // Future: Stunned, Dazed, Bleeding, Entangled, etc.
)

// Character struct changes
type Character struct {
    // ... existing fields ...
    Conditions []CombatCondition `yaml:"-"` // Active combat conditions
}
```

**API Design**:
```go
// Simple, consistent API
char.AddCondition(ConditionDefensePenalty, 1, -0.15, "failed grapple")
char.HasCondition(ConditionRecoveryPenalty) bool
char.GetConditionMagnitude(ConditionDefensePenalty) float64
char.RemoveCondition(ConditionGrappleOpportunity)
char.TickConditions() // Called once in round tick, decrements all durations
```

**Changes**:
1. Create `internal/combat/conditions.go` — Condition types, struct, management functions
2. Add `Conditions []CombatCondition` field to Character struct
3. Replace `RecoveryPenaltyThisRound` with `ConditionRecoveryPenalty`
4. Replace `DefensePenaltyNextRound` with `ConditionDefensePenalty`
5. Replace `IsGrappleController` with `ConditionGrappleController`
6. Migrate grapple opportunity from Timers to Conditions
7. Update combat.go to check conditions instead of individual flags
8. Replace scattered flag clearing in hooks with single `TickConditions()` call
9. Add condition display to `score`/`effects` command

**Files to Modify** (~12 files, ~400 lines):
1. `internal/combat/conditions.go` — New file, core conditions system
2. `internal/characters/character.go` — Add Conditions field, remove old flags
3. `internal/combat/combat.go` — Check conditions instead of flags
4. `internal/combat/grapple.go` — Use condition API instead of flags
5. `internal/combat/criteffects.go` — Migrate grapple opportunity to conditions
6. `internal/usercommands/stand.go` — Check recovery penalty via condition
7. `internal/usercommands/grapple.go` — Apply conditions instead of flags
8. `internal/hooks/NewRound_UserRoundTick.go` — Replace flag clearing with TickConditions()
9. `internal/hooks/NewRound_MobRoundTick.go` — Replace flag clearing with TickConditions()
10. `internal/usercommands/score.go` — Display active conditions
11. Test files

**Migration Strategy**:
1. Implement conditions system alongside existing flags (both work in parallel)
2. Migrate one flag at a time (RecoveryPenalty → DefensePenalty → GrappleController → GrappleOpportunity)
3. Test thoroughly after each migration
4. Remove old flags only after all are migrated and tested
5. Clean up scattered clearing logic from round ticks

**Testing**:
- [ ] **Unit Tests**: Condition add/remove/tick logic
- [ ] **Manual Test**: Prone recovery, verify RecoveryPenalty condition applies and clears
- [ ] **Manual Test**: Failed grapple, verify DefensePenalty condition applies and clears
- [ ] **Manual Test**: Successful grapple, verify GrappleController condition tracks correctly
- [ ] **Manual Test**: Dodge crit, verify GrappleOpportunity condition applies and clears after use
- [ ] **Manual Test**: Multiple conditions active simultaneously, verify they stack correctly
- [ ] **Manual Test**: Save/reload, verify conditions don't persist (yaml:"-" flag)
- [ ] **Integration Test**: Full combat with multiple condition interactions

**Acceptance Criteria**:
- All existing combat flags migrated to conditions system
- Single `TickConditions()` call replaces scattered flag clearing
- Adding new temporary combat states requires only:
  1. Add ConditionType enum value
  2. Add application logic where state triggers
  3. Add effect logic in combat calculations
- No more manual hook updates for new flags
- Conditions display in `score` or `effects` command
- All existing combat mechanics work identically to before
- All tests pass

**Benefits**:
- **Maintainability**: One system instead of scattered flags
- **Discoverability**: Players can see active conditions via `effects`
- **Extensibility**: Easy to add Stunned, Dazed, Bleeding, Entangled, etc. in future
- **Debuggability**: Conditions track their source ("failed grapple", "dodge crit", etc.)
- **Consistency**: All temporary combat states managed the same way

**Estimated Changes**: ~400 lines, 12 files

---

## Phase 10: Combat Balance Pass

> **Note**: This phase deliberately comes after all mechanical combat changes are in place.
> Tuning numbers before the systems are built leads to rework.

### Stage 10.1: Combat Speed & Balance Tuning
**Goal**: Speed up combat to reach resolution faster. Currently fights drag on too long. Adjust damage curves, defense rates, and health pools so typical fights against equal opponents resolve in 5-10 rounds, not 20+.

**Changes**:
1. Audit and tune: base damage values, armor reduction, health pools
2. Tune avoidance rates (dodge/parry/block) so ~40-60% of attacks land
3. Tune stamina drain so fighters get winded in ~8-12 rounds
4. Tune critical hit damage multiplier (crits should meaningfully accelerate combat)
5. Ensure mob difficulty tiers feel distinct:
   - Easy mobs: 2-4 rounds
   - Medium mobs: 5-8 rounds
   - Hard mobs: 10-15 rounds (and dangerous)
6. Document the tuning values and rationale

**Files to Modify** (~8 files, ~300 lines):
1. `internal/combat/calculations.go` — Damage/defense tuning constants
2. `internal/combat/combat.go` — Round timing
3. Mob YAML files — Stat adjustments
4. Weapon/armor YAML files — Damage/defense adjustments
5. Config files — Tuning constants

**Testing**:
- [ ] **Manual Test**: Fight all test arena mobs, time the fights
- [ ] **Manual Test**: Verify easy/medium/hard feel appropriately different
- [ ] **Balance Test**: Record average rounds to kill for each mob tier
- [ ] **Balance Test**: Verify player death is possible but not frequent against appropriate mobs

**Acceptance Criteria**:
- Fights resolve in a satisfying number of rounds
- Difficulty tiers are distinct and feel right
- Combat is dangerous enough to be exciting, not a grind
- All tests pass

**Estimated Changes**: ~300 lines, 8 files

---

## Phase 11: LLM Integration & AI NPCs

> **Context**: Enable external LLM agents to interact with the MUD via structured data, and create a tiered AI NPC system that balances quality, performance, and cost. This phase uses a pragmatic approach: enhance existing GMCP for LLM agent access, build rule-based NPCs for most characters, and optionally add LLM-powered NPCs for special cases.

### Stage 11.1: GMCP Enhancement for Full State Coverage
**Goal**: Expand the existing GMCP (Generic Mud Communication Protocol) support to provide complete, structured game state data. This enables external LLM agents to "play" the MUD programmatically and provides rich context for AI decision-making.

**Current State**: GoMud already has basic GMCP support. This stage expands it to cover all game state.

**Changes**:
1. Audit existing GMCP modules and identify gaps
2. Add comprehensive GMCP modules:
   - `gmcp.room.full` — Complete room data (description, entities, exits, terrain, biome)
   - `gmcp.room.entities` — All mobs/players in room with stats visible to player
   - `gmcp.combat.state` — Ongoing combat state (participants, rounds, current target)
   - `gmcp.combat.result` — Structured combat round results (hit/miss, damage tier, defenses used)
   - `gmcp.character.full` — Complete character state (stats, skills, conditions, encumbrance)
   - `gmcp.character.equipment` — Equipped items with stats
   - `gmcp.character.inventory` — Backpack contents
   - `gmcp.commands.available` — Context-aware list of valid commands (excludes unavailable due to combat, cooldowns, etc.)
   - `gmcp.world.time` — Game time, weather, season
   - `gmcp.world.quests` — Active quests and objectives
3. Add GMCP event notifications:
   - `gmcp.event.combat_start`, `gmcp.event.combat_end`
   - `gmcp.event.condition_applied`, `gmcp.event.condition_removed`
   - `gmcp.event.cooldown_ready`
   - `gmcp.event.skill_progress`, `gmcp.event.stat_progress`
4. Create GMCP toggle in character settings (some players may want to disable)
5. Add GMCP documentation for LLM agent developers

**Files to Modify** (~12 files, ~600 lines):
1. `internal/gmcp/gmcp.go` — Core GMCP module registration and dispatch
2. `internal/gmcp/room.go` — Room state modules
3. `internal/gmcp/combat.go` — Combat state modules (new file)
4. `internal/gmcp/character.go` — Character state modules
5. `internal/gmcp/commands.go` — Available commands module (new file)
6. `internal/gmcp/world.go` — World state modules (new file)
7. `internal/combat/combat.go` — Send GMCP combat updates
8. `internal/hooks/` — Send GMCP event notifications from various hooks
9. `internal/characters/character.go` — GMCP preference flag
10. Documentation: `docs/GMCP_REFERENCE.md` (new file)
11. Documentation: `docs/LLM_AGENT_GUIDE.md` (new file)
12. Test files

**Testing**:
- [ ] **Unit Tests**: Test each GMCP module serialization
- [ ] **Integration Test**: Connect with GMCP-enabled client, verify all modules populate
- [ ] **Manual Test**: Walk through various game scenarios (combat, movement, quests) and verify GMCP updates
- [ ] **Agent Test**: Create simple Python LLM agent that can navigate and fight using only GMCP data
- [ ] **Regression Test**: Verify GMCP-disabled clients still work normally

**Acceptance Criteria**:
- All major game state is exposed via GMCP modules
- GMCP data is accurate and updates in real-time
- External agents can fully interact with the MUD using GMCP
- Documentation allows third-party LLM agent development
- No performance impact for non-GMCP clients
- All tests pass

**Estimated Changes**: ~600 lines, 12 files

---

### Stage 11.2: Rule-Based NPC Dialogue Framework
**Goal**: Create a flexible, fast, zero-cost dialogue system for 99% of NPCs using pattern matching, dialogue trees, and scripted responses. This is the foundation for making NPCs feel alive without any LLM costs or performance overhead.

**Design**:
- **Pattern-based responses**: NPC recognizes keywords/phrases and responds appropriately
- **Dialogue trees**: Branch based on previous conversation, quest state, player reputation
- **Context awareness**: NPCs know about room, recent events, player state
- **Emotion/mood system**: NPCs have moods that affect responses (friendly, hostile, afraid, etc.)
- **Memory**: NPCs remember recent interactions with specific players

**Example NPC Configuration** (YAML):
```yaml
npcId: 1001
name: "Merchant Talia"
personality: "friendly, greedy"
defaultMood: "welcoming"

dialoguePatterns:
  - pattern: "(hello|hi|greet|hey)"
    responses:
      - "Welcome, traveler! Care to see my wares?"
      - "Ah, a new face! What brings you to my shop?"
    mood: "welcoming"

  - pattern: "(price|cost|expensive|cheap)"
    responses:
      - "My prices are fair! The roads are dangerous, after all."
      - "You won't find better deals in the wasteland."
    mood: "defensive"

  - pattern: "(temple|priest|sanctum)"
    responses:
      - "The temple? Head east through the market, you can't miss it."
    requirements:
      questComplete: ["intro_quest"]

dialogueTree:
  root:
    text: "Hello, stranger."
    options:
      - text: "What do you sell?"
        goto: "shop_intro"
      - text: "Tell me about the temple."
        goto: "temple_info"
        requires:
          questActive: ["find_temple"]

memory:
  rememberFor: "24h"
  maxInteractions: 10
```

**Changes**:
1. Create NPC dialogue data structure (patterns, trees, responses)
2. Create dialogue engine:
   - Pattern matching with regex
   - Tree navigation with requirements checking
   - Response selection (random, weighted, contextual)
3. Add conversation memory system (per-player, time-limited)
4. Add mood/emotion system affecting responses
5. Create YAML dialogue file format
6. Add conversation commands: `talk <npc>`, `ask <npc> about <topic>`, `tell <npc> <message>`
7. Add NPC response delays (1-2 seconds) for realism
8. Create example dialogues for common NPC types (merchant, guard, quest-giver)

**Files to Modify** (~15 files, ~800 lines):
1. `internal/npcs/dialogue/` — New package for dialogue system
2. `internal/npcs/dialogue/pattern.go` — Pattern matching engine
3. `internal/npcs/dialogue/tree.go` — Dialogue tree navigation
4. `internal/npcs/dialogue/memory.go` — Conversation memory
5. `internal/npcs/dialogue/mood.go` — Mood/emotion system
6. `internal/mobs/mobs.go` — Add dialogue data to mob struct
7. `internal/usercommands/talk.go` — New command (aliases: ask, tell, converse)
8. `internal/fileloader/` — Add dialogue file loading
9. `_datafiles/dialogues/` — Example dialogue files
10. Test files

**Testing**:
- [ ] **Unit Tests**: Pattern matching, tree navigation, memory system
- [ ] **Manual Test**: Converse with multiple NPCs, verify responses
- [ ] **Manual Test**: Verify mood changes affect responses
- [ ] **Manual Test**: Verify NPCs remember previous conversations
- [ ] **Manual Test**: Verify quest/state requirements work
- [ ] **Performance Test**: 100 NPCs in same room responding simultaneously

**Acceptance Criteria**:
- NPCs respond intelligently to player input using patterns
- Dialogue trees support branching conversations
- NPCs have distinct personalities via response variations
- Memory system allows contextual follow-up conversations
- Zero performance impact (all processing is trivial regex/lookups)
- Easy to author new NPC dialogues via YAML
- All tests pass

**Estimated Changes**: ~800 lines, 15 files

---

### Stage 11.3: Local LLM Integration (Optional)
**Goal**: Add support for 5-10 special NPCs powered by locally-hosted small language models (Llama 3.2 3B, Phi-3, Mistral 7B). These NPCs provide more dynamic, emergent conversations than rule-based NPCs while avoiding API costs.

**Note**: This stage is **optional** and requires:
- A separate machine or VPS to run the LLM service
- ~8GB RAM minimum for 3B models, 16GB for 7B models
- ollama, llama.cpp, or similar inference server

**Design**:
- **Async architecture**: NPC sends request to LLM service, continues other activities while waiting
- **Separate process**: LLM runs on different machine/container, MUD server communicates via HTTP
- **Heavy caching**: Same questions get cached responses (with minor variations)
- **Rate limiting**: Max 1 response per NPC every 5-10 seconds
- **Fallback**: If LLM service is down, fall back to rule-based responses

**LLM Service Architecture**:
```
MUD Server ──(HTTP POST)──> LLM Service (ollama/llama.cpp)
                             - Runs on separate machine/container
                             - Model: Llama 3.2 3B or Phi-3 Mini
                             - Response time: 0.5-2 seconds

MUD Server ←─(JSON response)── LLM Service
```

**Changes**:
1. Create LLM client package:
   - HTTP client to ollama/llama.cpp API
   - Async request/response handling
   - Connection pooling and timeouts
2. Add NPC type: `LLMControlledNPC`
   - Has personality prompt template
   - Has conversation history (rolling window)
   - Has response cache
3. Add response caching system:
   - Exact match cache (question → answer)
   - Semantic similarity cache (optional, uses embeddings)
   - Time-based cache expiration (1 hour)
4. Add rate limiting per NPC:
   - Cooldown between responses
   - Max responses per hour
5. Add configuration for LLM service endpoint
6. Add fallback to rule-based responses if LLM unavailable
7. Add debug mode to log LLM prompts/responses

**Example NPC Configuration** (YAML):
```yaml
npcId: 2001
name: "Oracle of the Wastes"
type: "llm_controlled"
llmModel: "llama3.2-3b"

personality: |
  You are the Oracle of the Wastes, an ancient seer who speaks in cryptic riddles.
  You know about the mutations, the old world, and the sanctum.
  You are helpful but mysterious. Keep responses under 100 words.

context: |
  The player is in a post-apocalyptic wasteland. Mutations are common.
  The Sanctum is a safe haven. The player is seeking answers.

rateLimits:
  minDelay: 10s
  maxPerHour: 20

cache:
  enabled: true
  ttl: 1h
  maxSize: 1000

fallback:
  useRuleBasedOnFailure: true
  patterns:
    - pattern: "(hello|greet)"
      response: "The sands whisper your name, traveler..."
```

**Files to Modify** (~12 files, ~700 lines):
1. `internal/npcs/llm/` — New package for LLM integration
2. `internal/npcs/llm/client.go` — HTTP client for LLM service
3. `internal/npcs/llm/cache.go` — Response caching
4. `internal/npcs/llm/ratelimit.go` — Rate limiting
5. `internal/mobs/mobs.go` — Add LLM NPC type
6. `internal/mobs/ai.go` — LLM NPC behavior (new file)
7. `internal/configs/config.llm.go` — LLM configuration (new file)
8. `_datafiles/config.yaml` — Add LLM service endpoint config
9. `_datafiles/npcs/llm_npcs/` — Example LLM NPC definitions
10. Documentation: `docs/LLM_NPC_SETUP.md` (new file)
11. Test files

**Testing**:
- [ ] **Integration Test**: Mock LLM service, verify requests/responses
- [ ] **Manual Test**: Converse with LLM NPC, verify responses are contextual
- [ ] **Manual Test**: Verify caching reduces duplicate LLM calls
- [ ] **Manual Test**: Verify rate limiting prevents spam
- [ ] **Manual Test**: Verify fallback works when LLM service is down
- [ ] **Performance Test**: Multiple players talking to LLM NPC simultaneously
- [ ] **Load Test**: Verify MUD remains responsive with slow LLM responses

**Acceptance Criteria**:
- LLM NPCs respond with contextual, dynamic dialogue
- Async architecture prevents MUD blocking
- Caching reduces LLM calls by 80%+
- Rate limiting prevents abuse
- Graceful fallback when LLM unavailable
- Clear documentation for deploying LLM service
- All tests pass

**Estimated Changes**: ~700 lines, 12 files

---

### Stage 11.4: Cloud API Integration (Optional)
**Goal**: Add support for 1-2 "legendary" NPCs powered by cloud LLM APIs (OpenAI, Anthropic). These provide the highest quality conversations but have API costs, so they're reserved for critical story NPCs.

**Note**: This stage is **optional** and **costly**. Only use for NPCs where quality is paramount (main quest hub, final boss, etc.). Budget $5-50/month depending on player traffic.

**Design**:
- Same architecture as Stage 11.3, but calls OpenAI/Anthropic instead of local service
- **Aggressive cost mitigation**:
  - Heavy caching (99% cache hit rate target)
  - Very strict rate limiting (1 response per 30 seconds)
  - Budget alerts (notify admin if costs exceed threshold)
  - Automatic shutdown if budget exceeded
- **Quality over quantity**: 1-2 NPCs total, not more

**Changes**:
1. Add cloud API clients:
   - OpenAI client (using official SDK)
   - Anthropic client (using official SDK)
2. Add cost tracking and budget enforcement:
   - Track API calls and estimated costs
   - Alert when approaching budget limit
   - Auto-disable NPCs if budget exceeded
3. Add even heavier caching than Stage 11.3:
   - Persistent cache (survives server restart)
   - Semantic similarity matching for cache hits
4. Add NPC type: `CloudLLMNPC`
5. Add configuration for API keys and budgets

**Example NPC Configuration** (YAML):
```yaml
npcId: 3001
name: "The Sanctum Keeper"
type: "cloud_llm"
provider: "anthropic"
model: "claude-3-haiku-20240307"

personality: |
  You are the Sanctum Keeper, guardian of the last bastion of civilization.
  You guide newcomers, assign quests, and know the history of the wasteland.
  You are wise, patient, and slightly melancholic about the old world.
  Keep responses under 150 words.

context: |
  The Sanctum is a walled settlement protecting survivors.
  Outside is a harsh wasteland filled with mutated creatures.
  The player is a newcomer seeking refuge and purpose.

budget:
  maxCostPerDay: 5.00    # USD
  alertThreshold: 4.00
  autoDisableAt: 5.50

rateLimits:
  minDelay: 30s           # Very strict for cloud APIs
  maxPerHour: 10
  maxPerDay: 50

cache:
  enabled: true
  persistent: true
  ttl: 24h
  semanticMatching: true
  similarityThreshold: 0.90
```

**Files to Modify** (~10 files, ~600 lines):
1. `internal/npcs/llm/openai.go` — OpenAI client (new file)
2. `internal/npcs/llm/anthropic.go` — Anthropic client (new file)
3. `internal/npcs/llm/cost.go` — Cost tracking and budget enforcement (new file)
4. `internal/npcs/llm/cache.go` — Enhance with persistent storage and semantic matching
5. `internal/mobs/mobs.go` — Add CloudLLM NPC type
6. `internal/configs/config.llm.go` — Add cloud API configuration
7. `_datafiles/config.yaml` — Add API keys and budget settings
8. `_datafiles/npcs/cloud_npcs/` — Example cloud NPC definitions
9. Documentation: `docs/CLOUD_LLM_SETUP.md` (new file)
10. Test files

**Testing**:
- [ ] **Integration Test**: Mock cloud APIs, verify requests/responses
- [ ] **Manual Test**: Converse with cloud NPC, verify high-quality responses
- [ ] **Manual Test**: Verify budget tracking and alerts work
- [ ] **Manual Test**: Verify auto-disable when budget exceeded
- [ ] **Manual Test**: Verify persistent caching survives server restart
- [ ] **Cost Test**: Monitor actual API costs over 24 hours with test traffic

**Acceptance Criteria**:
- Cloud NPCs provide very high quality, contextual dialogue
- Cost tracking accurately estimates API spend
- Budget enforcement prevents runaway costs
- Cache hit rate >95%
- Persistent cache survives restarts
- Auto-disable protects against cost overruns
- Clear documentation for API key setup
- All tests pass

**Estimated Changes**: ~600 lines, 10 files

---

## Testing Strategy

### Manual Testing Checklist (Run After Each Stage)
- [ ] MUD starts without errors
- [ ] Character creation works
- [ ] Character save/load works
- [ ] Movement works
- [ ] Combat works
- [ ] Skills/spells work (if applicable)
- [ ] No crashes during 10-minute play session

### Unit Test Requirements
Each stage must include:
- Unit tests for new functions
- Update existing tests that break
- Aim for 70%+ code coverage on modified files

### Integration Test Requirements
Each phase (1-10) must include:
- Full character lifecycle test (create → play → save → load)
- Combat integration test
- Progression integration test (where applicable)

### Regression Test Requirements
Before each git commit:
- Run full test suite: `go test ./...`
- No test failures allowed
- No new compiler warnings

---

## Git Workflow (Per Stage)

### Branch Naming
- `feature/stage-1.1-rename-stats`
- `feature/stage-2.1-rename-race-to-species`
- etc.

### Commit Process (Per Stage)
1. Create feature branch from `development`
2. Implement stage
3. Write/update tests
4. Manual testing
5. Run full test suite
6. Commit with conventional commit message:
   ```
   feat: [stage X.Y] Brief description

   - Detailed change 1
   - Detailed change 2
   - Testing: describe testing done

   Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
   ```
7. Merge to `development` with `--no-ff`
8. Test after merge
9. Move to next stage

---

## Risk Mitigation

### High-Risk Stages (Extra Care Required)
- **Stage 3.4**: ✅ Decouple Combat from Levels (combat refactor)
- **Stage 3.5**: ✅ Remove Level System (major breaking change)
- **Stage 4.2**: ✅ Replace Dice with Distribution (combat refactor) — merge 3892439
- **Stage 7.1**: ✅ Segmented Avoidance (major combat refactor — dodge/parry/block replaces single defense roll) — merge cd146e5
- **Stage 8.1**: Grappling System (new combat subsystem, many interaction points)
- **Stage 9.1**: Descriptive Damage Text (touches all combat output — high regression risk)

### Backup Strategy
Before each high-risk stage:
1. Tag current state: `git tag pre-stage-X.Y`
2. Create backup branch: `git branch backup-YYYY-MM-DD`
3. Test rollback plan

### Rollback Plan
If a stage breaks the MUD:
1. `git rebase --abort` (if in progress)
2. `git reset --hard origin/development`
3. Investigate issue
4. Re-attempt with fixes

---

## Estimated Timeline

Assuming ~4 hours per stage (implement + test):

| Phase | Stages | Estimated Hours | Status |
|-------|--------|-----------------|--------|
| Phase 1: Stats | 3 stages (1.1–1.3) | 12 hours | **Complete** |
| Phase 2: Species | 2 stages (2.1–2.2) | 8 hours | **Complete** |
| Phase 3: Remove Levels | 9 stages (3.1–3.9) | 36 hours | **Complete** |
| Phase 4: Distribution Combat | 4 stages (4.1–4.4) | 7 hours | **Complete** |
| Phase 4b: Progression Fixes | 4 stages (4.5–4.8) | 12 hours | **Complete** |
| Phase 5: Stamina & Attacks | 4 stages (5.1–5.4) | 20 hours | **Complete** |
| Phase 6: Conviction & Magic | 2 stages (6.1–6.2) | 8 hours | **Complete** |
| Phase 7: Defense & Combat | 5 stages (7.1–7.5) | 26 hours | **Complete** |
| Phase 8: Grappling | 5 stages (8.1–8.5) | 24 hours | **8.1–8.5 Complete** |
| Phase 9: Combat Presentation | 4 stages (9.1–9.4) | 22 hours | Not Started |
| Phase 10: Balance Pass | 1 stage (10.1) | 8 hours | Not Started |
| Phase 11: LLM Integration | 4 stages (11.1–11.4) | 35 hours | Not Started |
| **Total** | **43 stages** | **202 hours** | |

**Note**: Timeline is rough estimate. Adjust based on actual progress.

---

## Success Metrics

### Per Stage
- [ ] All tests pass
- [ ] MUD runs without errors
- [ ] Stage features work as designed
- [ ] No regression in existing features

### Per Phase
- [ ] Integration tests pass
- [ ] Manual playtesting confirms features work
- [ ] Code committed to development branch
- [ ] Documentation updated

### Overall
- [ ] All 10 phases complete
- [ ] Core DOGMud mechanics functional
- [ ] Combat is descriptive, immersive, and balanced
- [ ] Migration path from GoMud verified
- [ ] Ready for world building and content creation

---

## Next Steps After Core Mechanics

Once core mechanics (Phases 1-11) are complete:
1. **Phase 12**: Tutorial Area (Sanctum Basin)
2. **Phase 13**: Mutation System
3. **Phase 14**: Economy & Crafting
4. **Phase 15**: World Building (Cities, NPCs)

These phases will be detailed in a separate plan once core mechanics are stable.

---

## Issue Traceability

Issues discovered during 2026-02-12 playtest session, mapped to stages:

| # | Issue | Stage(s) |
|---|-------|----------|
| 1 | Stats other than Vitality don't increase | 4.5 |
| 2 | Attack count formula needs rework | 5.3 |
| 3 | Skill soft cap not working; need per-skill multipliers | 4.6 |
| 4 | Unarmed damage doesn't scale; needs grappling | 7.3, 8.1, 8.2 |
| 5 | Combat takes too long | 10.1 |
| 6 | Crit success/failure rate imbalance or messaging | 4.7 |
| 7 | Commands like equipping armor should be disabled in combat | 7.2 |
| 8 | Defense should be dodge/parry/block, not single roll | 7.1 |
| 9 | Configurable prompt (tank, target, health) | 9.3 |
| 10 | Target switching in combat | 7.4 |
| 11 | Remove player guide (levels are gone) | 4.8 |
| 12 | Descriptive text instead of damage numbers; resource bars | 9.1, 9.2 |

---

## Hotfixes & Bug Fixes

Critical bugs fixed outside of formal stage development:

### 2026-02-13: Fumble Detection Bug (commits 9b191c2, 2ff9c32)
**Issue**: Arena Champion and all high-skill NPCs were fumbling 30%+ of attacks instead of the intended 2.5%.

**Root Causes**:
1. Fumbles were detected based on defender's dodge roll success (opposed roll), not attacker's raw performance
2. Skill advantage calculation made fumbleThreshold positive for high-skill attackers, causing massive fumble rates

**Fixes**:
- Added initial attack roll before defense sequence for fumble detection
- Fumbles now based on attacker's raw z-score (≤ -2.0)
- Fumble threshold is now fixed at -2.0 (~2.5% chance) regardless of skill difference
- Only crit threshold scales with skill advantage (as intended)

**Result**: Combat balance restored. High-skill NPCs no longer fumble constantly. Stamina depletion now drives combat dynamics as designed.

---

## Notes

- Each stage is designed to be completable in 1-2 work sessions
- Stages build on each other - complete in order
- Don't skip testing stages (critical for stability)
- If a stage exceeds scope (1000 lines/50 files), break it into sub-stages
- Prioritize keeping the MUD playable at each stage
- Document any deviations from the plan in commit messages

---

**Last Updated**: 2026-02-14
**Status**: In Progress
**Current Stage**: 8.8 Complete — Auto-aggro hotfix applied, ready for Stage 8.9 (NPC Combat AI)
