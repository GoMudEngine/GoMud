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

### Stage 5.3: Rework Number of Attacks Formula
**Goal**: Replace the current attack count calculation with a formula based on: `Dexterity * weapon speed multiplier * encumbrance modifier * stamina modifier`, rounded to the nearest whole number each round.

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

## Phase 6: Conviction & Magic System

### Stage 6.1: Add Spell Schools as Tags
**Goal**: Implement DOG's 4 spell schools (Elemental, Enhancement, Mental, Vital).

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

### Stage 6.2: Replace Mana with Conviction Costs
**Goal**: Make spells cost Conviction instead of mana.

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

### Stage 7.1: Segmented Avoidance — Dodge, Parry, Block
**Goal**: Replace the single "attack roll vs defense roll" with a layered defense system: after the attack roll succeeds, the defender gets separate checks for dodge, parry, and block — each based on different skills and equipment.

**Design**:
- **Attack roll**: Attacker's combat skill + stat vs a base difficulty (not directly opposed by defender)
- **Dodge check**: Based on Unarmed Combat skill + Dexterity. Available to everyone. Encumbrance and armor reduce dodge chance. Costs stamina.
- **Parry check**: Based on Weapon Combat skill + weapon's parry rating. Only available if wielding a weapon. Two-handed weapons get a bonus. Costs stamina.
- **Block check**: Based on Weapon Combat skill + shield's block rating. Only available if wielding a shield. Two shields = high block chance but no attack. Costs stamina.
- Checks are rolled in order: dodge → parry → block. First success avoids the hit.
- Each defensive action has a stamina cost, creating a resource tradeoff for long fights.

**Changes**:
1. Add `ParryRating float64` to weapon item spec
2. Add `BlockRating float64` to shield item spec
3. Implement `RollDodge()`, `RollParry()`, `RollBlock()` functions in combat calculations
4. Refactor `AttackPlayerVsPlayer`/`AttackPlayerVsMob` to use layered defense
5. Generate distinct combat messages for each avoidance type ("You dodge...", "You parry...", "You block...")
6. Dual-shield support: if both hands hold shields, block chance stacks but no attacks possible

**Files to Modify** (~12 files, ~600 lines):
1. `internal/combat/calculations.go` — Dodge/parry/block formulas
2. `internal/combat/combat.go` — Refactor attack resolution
3. `internal/items/itemspec.go` — Add parry/block ratings
4. Weapon/shield YAML files — Set parry/block ratings
5. Combat message templates or inline strings
6. Test files

**Testing**:
- [ ] **Unit Tests**: Test each avoidance check independently
- [ ] **Manual Test**: Fight unarmed, verify dodge messages appear
- [ ] **Manual Test**: Fight with sword, verify parry messages appear
- [ ] **Manual Test**: Fight with shield, verify block messages appear
- [ ] **Manual Test**: Equip two shields, verify high block rate but no attacks
- [ ] **Balance Test**: Verify overall hit rate feels right (~50-70% of attacks land)

**Acceptance Criteria**:
- Three distinct avoidance types with separate rolls
- Each avoidance tied to appropriate skill and equipment
- Dual-shield is a viable defensive strategy
- Combat messages clearly indicate dodge vs parry vs block
- Stamina cost for each defensive action
- All tests pass

**Estimated Changes**: ~600 lines, 12 files

---

### Stage 7.2: Restrict Commands During Combat
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

### Stage 7.3: Unarmed Damage Scaling
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

### Stage 7.4: Target Switching in Combat
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

## Phase 8: Grappling & Advanced Unarmed Combat

> **Inspiration**: The Evennia-based DOG combat system (see `combat_example_evennia.png` and
> [Evennia discussion #2411](https://github.com/evennia/evennia/discussions/2411)).
> Unarmed combat should include grappling as a stamina-based win condition, not just
> punching for health damage.

### Stage 8.1: Grappling System — Stamina Damage Attacks
**Goal**: Add grappling as a combat action that damages the opponent's stamina instead of (or in addition to) health. This creates an alternate win condition — exhaust your opponent.

**Design**:
- `grapple <target>` initiates a grapple attempt (Unarmed Combat skill check vs target's Unarmed Combat or Dexterity)
- Successful grapple applies the **Grappled** condition to the target
- While grappled, each round the grappler deals **stamina damage** (based on Strength + Unarmed Combat)
- Grappler can also deal minor health damage (chokes, joint locks)
- Target can attempt to **escape** each round (Unarmed Combat + Strength check)
- When a target's stamina hits 0 while grappled, they **yield** (combat ends, they're at the grappler's mercy)

**Changes**:
1. Add `grapple` command
2. Add `Grappled` condition flag to character state
3. Add grapple resolution logic (initiate, maintain, escape)
4. Add stamina damage calculation for grapples
5. Add yield/submission mechanic when stamina reaches 0 while grappled
6. Grapple-specific combat messages ("You lock Guard in a rear naked choke...", "Guard tries to escape your hold...")

**Files to Modify** (~10 files, ~500 lines):
1. `internal/usercommands/grapple.go` — New command
2. `internal/combat/grapple.go` — Grapple resolution logic
3. `internal/combat/combat.go` — Integrate grapple into combat round
4. `internal/characters/character.go` — Grappled condition flag
5. `internal/hooks/NewRound_DoCombat.go` — Grapple round processing
6. Test files

**Testing**:
- [ ] **Manual Test**: Initiate grapple, verify grapple messages
- [ ] **Manual Test**: Verify stamina damage while grappled
- [ ] **Manual Test**: Escape from grapple, verify escape messages
- [ ] **Manual Test**: Exhaust opponent's stamina, verify yield
- [ ] **Balance Test**: Grappling should be viable but not dominant

**Acceptance Criteria**:
- Grapple is a distinct combat action with its own command
- Grappled condition has real mechanical effects
- Stamina damage creates an alternate win condition
- Escape mechanics prevent grapple from being inescapable
- All tests pass

**Estimated Changes**: ~500 lines, 10 files

---

### Stage 8.2: Grappled Condition Effects
**Goal**: Make the Grappled condition have meaningful combat penalties for the target (and some drawbacks for the grappler).

**Grappled Target Penalties**:
- Cannot use weapons (hands are occupied)
- Cannot dodge (movement restricted)
- Cannot flee (pinned)
- Reduced attack count (struggling)
- Can only use Unarmed Combat to attack or attempt escape
- Can still cast spells (if they have the concentration)

**Grappler Drawbacks**:
- Cannot use weapons (hands occupied with the hold)
- Cannot dodge (committed to the grapple)
- Vulnerable to attacks from third parties (focused on hold)
- Reduced perception (tunnel vision on the grapple)

**Changes**:
1. Apply combat penalties when `Grappled` flag is set
2. Restrict weapon use for both grappler and target
3. Disable dodge/flee for grappled target
4. Reduce grappler's defenses against third-party attacks
5. Add messaging for attempted actions while grappled ("You can't swing your sword while grappled!")

**Files to Modify** (~8 files, ~300 lines):
1. `internal/combat/combat.go` — Apply grapple penalties to attack/defense
2. `internal/combat/calculations.go` — Grapple modifiers
3. `internal/usercommands/flee.go` — Block while grappled
4. `internal/usercommands/attack.go` — Restrict weapon attacks while grappled
5. Test files

**Testing**:
- [ ] **Manual Test**: While grappled, try to use a weapon — verify blocked
- [ ] **Manual Test**: While grappled, try to flee — verify blocked
- [ ] **Manual Test**: Attack a grappler as a third party — verify they're easier to hit
- [ ] **Manual Test**: While grappling, verify reduced defense
- [ ] **Balance Test**: Grapple penalties are meaningful but not instant death

**Acceptance Criteria**:
- Grappled targets have significant but fair penalties
- Grapplers have meaningful drawbacks (not free advantage)
- Third-party intervention is effective against grapplers
- Clear messaging for all restricted actions
- All tests pass

**Estimated Changes**: ~300 lines, 8 files

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
- **Stage 7.1**: Segmented Avoidance (major combat refactor — dodge/parry/block replaces single defense roll)
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
| Phase 5: Stamina & Attacks | 3 stages (5.1–5.3) | 16 hours | 5.1–5.2 Complete |
| Phase 6: Conviction & Magic | 2 stages (6.1–6.2) | 8 hours | Not Started |
| Phase 7: Defense & Combat | 4 stages (7.1–7.4) | 20 hours | Not Started |
| Phase 8: Grappling | 2 stages (8.1–8.2) | 12 hours | Not Started |
| Phase 9: Combat Presentation | 3 stages (9.1–9.3) | 16 hours | Not Started |
| Phase 10: Balance Pass | 1 stage (10.1) | 8 hours | Not Started |
| **Total** | **37 stages** | **155 hours** | |

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

Once core mechanics (Phases 1-10) are complete:
1. **Phase 11**: Tutorial Area (Sanctum Basin)
2. **Phase 12**: Mutation System
3. **Phase 13**: Economy & Crafting
4. **Phase 14**: World Building (Cities, NPCs)

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

## Notes

- Each stage is designed to be completable in 1-2 work sessions
- Stages build on each other - complete in order
- Don't skip testing stages (critical for stability)
- If a stage exceeds scope (1000 lines/50 files), break it into sub-stages
- Prioritize keeping the MUD playable at each stage
- Document any deviations from the plan in commit messages

---

**Last Updated**: 2026-02-12
**Status**: In Progress
**Current Stage**: 5.3 — Rework Number of Attacks Formula (next up)
