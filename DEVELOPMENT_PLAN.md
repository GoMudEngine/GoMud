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
   - Bash: Good if target standing, mob has shield equipped, target health > 50%
   - Trip: Good if target standing, mob has decent dexterity, close combat
   - Kick: Good if target standing, mob unarmed or light weapon
   - Grapple: Good if both standing, mob has unarmed-combat skill, wants control
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
- +40 if wielding shield
- +20 if target health > 60%
- +15 if combat skill > 50
- -50 if target prone
- -100 if no shield equipped

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
- +30 if mob unarmed-combat > 40
- +20 if mob strength > target strength
- +15 if target health < 30% (finish them)
- -100 if already in grapple
- -50 if mob health < 20% (too risky)

**Submit** (only when grapple controller):
- Base: 40
- +40 if in dominant position (mounted, standing over prone)
- +20 if mob unarmed-combat > 60
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

**Status**: ✅ COMPLETED - NPC Combat AI implemented with 6 profiles, contextual move scoring, and YAML customization (10d908a)

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
| 8.8 | Auto-aggro reciprocal (hotfix) | ~10 | 2 | ✅ Complete (61fae51) |
| 8.9 | NPC combat AI for special moves | ~400 | 13 | ✅ Complete (10d908a) |
| **Total** | **Integrated grappling + AI** | **~2260** | **~68 unique** | **Phase 8 Complete** |

**Phase Completion**: After Stage 8.9, combat system will include:
- Position-based mechanics (Standing/Prone/Clinched/Grounded)
- Grappling integrated with striking
- Organic special moves (disarms, submissions, bash, trip, kick)
- Equipment-driven playstyles (shields for bash, etc.)
- Multi-combatant tactics
- Risk/reward grappling decisions
- **Intelligent NPC AI** using special moves contextually
- Customizable per-mob combat behavior via AI profiles

---

## Phase 9: Combat Presentation Overhaul

> **Goal**: Transform combat from a "you hit for X damage" number game into an immersive,
> descriptive experience inspired by the Evennia DOG combat system (see `combat_example_evennia.png`).
> Progression: Remove numbers → Expand narrative variety → Add skill-based messaging depth.

---

### Stage 9.1: Remove Damage Numbers — Clean Foundation ✅ COMPLETED
**Goal**: Remove numeric damage from all combat output, creating a clean foundation for narrative combat expansion.

**Problem Identified**:
- Initial approach (replacing numbers with HP% descriptions) created **double description issue**
- YAML tier verbs (CRUSHES, lands squarely) already describe attack intensity
- HP% damage descriptions (light wounds, critical injuries) describe damage result
- These conflict when attack intensity ≠ damage result (e.g., "CRUSHES... light wounds")
- Root cause: Two independent tier systems (attack strength vs actual damage %)

**Revised Approach**:
- **Remove `{damage}` token entirely** from YAML combat messages
- Keep existing weapon-specific verbs from YAML tiers (weak/normal/heavy/critical)
- Result: "Your fists CRITICALLY SMASHES training dummy!" (clean, immersive, no contradictions)
- Foundation ready for future narrative expansion via YAML pool expansion

**Changes**:
1. Remove `{damage}` token and surrounding text from all 8 weapon YAML files
2. Update special move commands (kick, bash, trip, submit) to remove numeric damage
3. Keep the `GetDamageDescription()` function for potential future use
4. Maintain existing YAML tier system (attack verbs already provide narrative)

**Files Modified** (10 files, ~160 lines):
1. ✅ `internal/combat/descriptions.go` — NEW: GetDamageDescription() function (kept for future)
2. ✅ `internal/combat/combat.go` — Removed damage from token replacement, pet messages, blocked damage
3. ✅ `_datafiles/world/dogmud/combat-messages/slashing.yaml` — Removed {damage} token
4. ✅ `_datafiles/world/dogmud/combat-messages/stabbing.yaml` — Removed {damage} token
5. ✅ `_datafiles/world/dogmud/combat-messages/bludgeoning.yaml` — Removed {damage} token
6. ✅ `_datafiles/world/dogmud/combat-messages/cleaving.yaml` — Removed {damage} token
7. ✅ `_datafiles/world/dogmud/combat-messages/whipping.yaml` — Removed {damage} token
8. ✅ `_datafiles/world/dogmud/combat-messages/claws.yaml` — Removed {damage} token
9. ✅ `_datafiles/world/dogmud/combat-messages/shooting.yaml` — Removed {damage} token
10. ✅ `_datafiles/world/dogmud/combat-messages/generic.yaml` — Removed {damage} token
11. ✅ `internal/usercommands/kick.go` — Removed numeric damage display
12. ✅ `internal/usercommands/bash.go` — Removed numeric damage display
13. ✅ `internal/usercommands/trip.go` — Removed numeric damage display
14. ✅ `internal/usercommands/submit.go` — Removed numeric damage display

**Testing**:
- ✅ **Manual Test**: Fight with various weapons, verify no numeric damage appears
- ✅ **Manual Test**: Use kick/bash/trip, verify no numeric damage appears
- ✅ **Manual Test**: Verify weapon-specific verbs still work (slash, smash, pierce, etc.)
- ✅ **Manual Test**: Verify critical hits show *** markers, fumbles show !!! markers
- ✅ **Manual Test**: Verify dodge/parry/block messages unchanged

**Acceptance Criteria**:
- ✅ Zero numeric damage in player-visible combat output
- ✅ Weapon-specific verbs from YAML work correctly
- ✅ All existing combat mechanics (criticals, fumbles, defense) work unchanged
- ✅ Clean foundation ready for narrative expansion
- ✅ All tests pass

**Estimated Changes**: ~160 lines, 14 files
**Actual Changes**: ~298 lines, 14 files
**Status**: ✅ COMPLETED
**Merge Commits**: 8769807 (YAML cleanup), 90b0b9b (special moves & combat system)

---

### Stage 9.2: Expand Combat Narrative — Message Pool Variety ✅ COMPLETED
**Merge Commit**: bcd0870
**Goal**: Expand YAML message pools from 3 variants per tier to 10-15 variants, adding footwork, feints, and positioning flavor.

**Key Insight**:
The existing YAML system is **perfectly designed** for this enhancement. No code changes needed—just expand the creative content in YAML files.

**Design**:
- Current: 3 message variants per tier (weak/normal/heavy/critical)
- Target: 10-15 message variants per tier
- Add narrative variety:
  - **Footwork**: "You circle left and slash...", "You sidestep and thrust..."
  - **Feints**: "Feinting high, you bring your blade down...", "You fake left, then strike..."
  - **Positioning**: "You press forward with a heavy slash...", "Dancing back, you riposte..."
  - **Momentum**: "Following through, you spin and slash...", "You capitalize on your opening..."

**Example Expansion** (slashing.yaml, normal tier):
```yaml
normal:
  together:
    toattacker:
    # Original 3 messages (keep these)
    - 'You connect with your {itemname}, wounding {target}!'
    - 'You slash {target} with your {itemname}!'
    - 'Your {itemname} cuts into {target}!'
    # New narrative variants (add ~10 more)
    - 'You circle left and slash at {target} with precision!'
    - 'Feinting high, you bring your {itemname} down in a swift arc!'
    - 'You step forward and deliver a sharp slash to {target}!'
    - 'Dancing around {target}, you slice with practiced ease!'
    - 'You press your advantage with a sweeping slash!'
    - 'Pivoting on your heel, you cut at {target}''s flank!'
    - 'Your {itemname} flashes as you strike with purpose!'
    - 'You exploit an opening and slash decisively!'
    - 'Following your momentum, you deliver a clean cut!'
    - 'You weave past {target}''s guard and slash!'
```

**Changes**:
1. Expand each weapon YAML file from ~130 lines to ~400 lines
2. Add 10-15 variants per tier (weak, normal, heavy, critical) × 3 perspectives (toattacker, todefender, toroom)
3. Maintain existing token system ({itemname}, {target}, {source}, etc.)
4. Zero code changes—existing `msgs.Together.ToAttacker.Get(msgSeed)` already picks randomly

**Files to Modify** (8 YAML files, ~2400 lines total):
1. `_datafiles/world/dogmud/combat-messages/slashing.yaml` — Expand from 132 to ~400 lines
2. `_datafiles/world/dogmud/combat-messages/stabbing.yaml` — Expand from 132 to ~400 lines
3. `_datafiles/world/dogmud/combat-messages/bludgeoning.yaml` — Expand from 132 to ~400 lines
4. `_datafiles/world/dogmud/combat-messages/cleaving.yaml` — Expand from 132 to ~400 lines
5. `_datafiles/world/dogmud/combat-messages/whipping.yaml` — Expand from 132 to ~400 lines
6. `_datafiles/world/dogmud/combat-messages/claws.yaml` — Expand from 132 to ~400 lines
7. `_datafiles/world/dogmud/combat-messages/shooting.yaml` — Expand from 132 to ~400 lines
8. `_datafiles/world/dogmud/combat-messages/generic.yaml` — Expand from 160 to ~400 lines

**Testing**:
- [ ] **Manual Test**: Fight 20+ rounds, verify message variety (minimal repetition)
- [ ] **Manual Test**: Verify footwork/feint/positioning messages appear naturally
- [ ] **Manual Test**: Verify all weapon types have expanded pools
- [ ] **Manual Test**: Verify toattacker/todefender/toroom perspectives all work
- [ ] **Manual Test**: Verify messages grammatically correct with all token replacements

**Acceptance Criteria**:
- Each tier has 10-15 message variants minimum
- Messages include footwork, feints, and positioning variety
- No repetition in typical 20-round combat
- All token replacements work correctly
- Messages feel natural and immersive
- Zero code changes required

**Estimated Effort**: 8-12 hours (creative writing)
**Estimated Changes**: ~2400 lines (YAML only), 8 files

---

### Stage 9.3: Defensive Action Narrative — Dodge/Parry/Block Messages ✅ COMPLETED
**Merge Commit**: 9ed3b69
**Goal**: Add YAML message pools for defensive actions (dodge, parry, block) with same narrative variety as attacks.

**Current State**:
- Dodges show generic: "You dodge the attack"
- Parries show generic: "You parry with your shield"
- Blocks show generic: "You block the attack"

**Target State**:
- Dodges: "You weave under the blow", "You sidestep gracefully", "You roll aside"
- Parries: "You deflect with your blade", "You catch the strike on your shield", "You redirect the blow"
- Blocks: "You absorb the impact", "You brace and block", "Your armor turns the strike"

**Design**:
- Create new YAML files for defensive actions
- Structure: Similar to attack messages (weak/normal/heavy tiers based on how close the attack was)
- Perspective messaging: todefender, toattacker, toroom
- Token system: {defender}, {attacker}, {weapon}, {defense_type}

**Files Created** (4 new files):
1. `internal/items/defensive_messages.go` — Defense message data structures and loader (90 lines)
2. `_datafiles/world/dogmud/defense-messages/dodge.yaml` — Dodge message pools (40+ variants)
3. `_datafiles/world/dogmud/defense-messages/parry.yaml` — Parry message pools (40+ variants)
4. `_datafiles/world/dogmud/defense-messages/block.yaml` — Block message pools (40+ variants)

**Files Modified** (2 files):
1. `internal/combat/combat.go` — Replaced generic defense messages with narrative variety system (~40 lines)
2. `internal/items/itemspec.go` — Added defensive message loading and new tokens (~10 lines)

**Testing**:
- [x] **Build Test**: Code compiles successfully
- [x] **Manual Test**: Get dodged, verify variety in dodge messages
- [x] **Manual Test**: Parry attacks, verify variety in parry messages
- [x] **Manual Test**: Block attacks, verify variety in block messages
- [x] **Manual Test**: Verify perspective messages (what you see vs what attacker sees)

**Acceptance Criteria**:
- ✅ Dodge/parry/block have 10-15 message variants each (achieved 40+ each across 3 intensities)
- ✅ Messages vary by how close the attack was (weak/normal/heavy based on z-score)
- ✅ Perspective messaging works (defender, attacker, observer)
- ✅ Token system supports {defender}, {attacker}, {weapon}
- ✅ Zero repetition in typical combat (manual testing confirmed)

**Actual Effort**: ~2 hours
**Actual Changes**: ~140 lines code, ~1200 lines YAML (4 new files, 2 modified)

---

### Stage 9.4: Contextual Combat Tokens — Stance, Position, Momentum ✅ COMPLETED
**Merge Commit**: 5cf780d

**Goal**: Add new context tokens ({stance}, {position}, {momentum}) to make messages more dynamic and situational.

**Design**:
- New tokens:
  - `{stance}` — "aggressive", "defensive", "balanced", "reckless" (based on combat behavior)
  - `{position}` — "prone", "standing", "elevated", "cornered" (based on combat position)
  - `{momentum}` — "on the offensive", "on the defensive", "pressured", "in control" (based on recent combat flow)

**Example Usage**:
```yaml
# Message variant that uses new tokens
- 'From your {stance} stance, you slash at {target}!'
- 'While {position}, you strike desperately at {target}!'
- 'You capitalize on your momentum, pressing {target} with a heavy blow!'
```

**Changes**:
1. Add stance calculation based on recent attack/defense balance
2. Add position tracking (already partially exists for prone)
3. Add momentum tracking based on recent hits/misses
4. Add new tokens to token replacement map in combat.go
5. Update YAML messages to use new tokens (subset of messages, not all)

**Files to Modify** (~4 files, ~200 lines):
1. `internal/combat/combat.go` — Calculate stance, position, momentum; add to token map
2. `internal/characters/character.go` — Add stance/momentum tracking fields
3. `_datafiles/world/dogmud/combat-messages/*.yaml` — Add token usage to subset of messages
4. Test files

**Testing**:
- [ ] **Manual Test**: Fight aggressively, verify "aggressive" stance appears
- [ ] **Manual Test**: Fight while prone, verify "prone" position appears
- [ ] **Manual Test**: Land multiple hits, verify momentum messages trigger
- [ ] **Manual Test**: Verify tokens replace correctly in all message variants

**Acceptance Criteria**:
- Stance, position, momentum calculate correctly
- New tokens work in message system
- Messages feel contextual and responsive to combat flow
- No performance impact (O(1) calculations)
- All tests pass

**Estimated Effort**: 8-10 hours
**Estimated Changes**: ~200 lines code, ~400 lines YAML updates

---

### Stage 9.5: Skill-Tiered Message Pools — Beginner to Master ✅ COMPLETED
**Commit**: `39878e4` - feat: implement Stage 9.5 skill-tiered combat messages and vitals privacy

**Goal**: Gate advanced combat messages behind weapon skill levels, creating progression from clumsy beginner to elegant master.

**Design**:
- Three skill tiers: `beginner`, `expert`, `master`
- Message pools organized by skill tier within each attack tier
- Low skill = simple, clumsy descriptions
- High skill = elegant, tactical descriptions
- Message selection:
  - Beginner (skill 1-33): Only beginner pool
  - Expert (skill 34-66): Beginner + expert pool
  - Master (skill 67-100): Beginner + expert + master pool

**Example Structure**:
```yaml
normal:
  together:
    toattacker:
      beginner:  # Basic, simple attacks
        - 'You swing your {itemname} at {target}!'
        - 'You slash awkwardly at {target}!'
        - 'You strike at {target} with your {itemname}!'
      expert:  # Tactical, competent attacks
        - 'You feint left and slash at {target}!'
        - 'You circle and deliver a precise cut!'
        - 'You exploit an opening and strike!'
      master:  # Elegant, masterful attacks
        - 'Your blade flows like water as you execute a perfect slash!'
        - 'With practiced grace, you weave a deadly arc toward {target}!'
        - 'You read {target}''s guard and strike the perfect opening!'
```

**Message Selection Algorithm**:
```go
// Collect available message pools based on skill level
availablePools := []string{"beginner"}
if weaponSkill >= 34 {
    availablePools = append(availablePools, "expert")
}
if weaponSkill >= 67 {
    availablePools = append(availablePools, "master")
}

// Pick random pool, then random message from that pool
selectedPool := availablePools[rand.Intn(len(availablePools))]
message := messages[tier][perspective][selectedPool].Get(msgSeed)
```

**Benefits**:
- **Progression feel**: Players experience combat evolution as skills improve
- **Immersion**: Messages match character competence level
- **Replayability**: Combat feels different at different skill levels
- **Training incentive**: Players want to level weapon skills to see master messages

**Changes**:
1. Update YAML structure to nest messages under skill tiers
2. Add skill-based message pool selection to combat.go
3. Expand each tier with beginner/expert/master variants (5 messages each = 15 total per tier)
4. Update message loading to handle nested structure
5. **Full integration of Stage 9.4 tokens** ({stance}, {position}, {momentum}) across all weapon types and skill tiers

**Files to Modify** (~3 files, ~150 lines code + ~3200 lines YAML):
1. `internal/combat/combat.go` — Add skill-based pool selection (~50 lines)
2. `internal/items/messages.go` — Update YAML parsing for nested structure (~100 lines)
3. `_datafiles/world/dogmud/combat-messages/*.yaml` — Restructure all 8 files with skill tiers + integrate Stage 9.4 tokens throughout (~3200 lines)
4. Test files

**Note**: This stage will complete the integration of Stage 9.4's contextual combat tokens ({stance}, {position}, {momentum}) across all weapon types and skill tiers.

**Testing**:
- [ ] **Manual Test**: Fight at skill 1, verify only beginner messages appear
- [ ] **Manual Test**: Train to skill 50, verify beginner+expert messages appear
- [ ] **Manual Test**: Train to skill 80, verify all three pools appear
- [ ] **Manual Test**: Verify proper randomization across available pools
- [ ] **Manual Test**: Verify message quality matches skill tier

**Acceptance Criteria**:
- Message pools gated correctly by skill level
- Beginner messages feel simple/clumsy
- Expert messages feel tactical/competent
- Master messages feel elegant/masterful
- Smooth progression from one tier to next
- No performance impact
- All tests pass

**Estimated Effort**: 12-16 hours (8 hours code, 8 hours YAML restructuring/writing)
**Estimated Changes**: ~150 lines code, ~3200 lines YAML

---

### Stage 9.6: Health/Stamina/Conviction Bars in Prompt ✅ COMPLETED
**Commit**: `98ecfe6` - feat: merge Stage 9.6 vital bars in prompt

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

### Stage 9.7: Configurable Combat Prompt ✅ COMPLETED
**Commit**: `c7ec255` - feat: implement Stage 9.7 configurable combat prompt
**Fix commits**: `7c368a3` (add FightPrompt config field), `6244c25` (fix generic.yaml startup panic)

**Goal**: Let players configure what information appears in their combat prompt via per-element toggles.

**Implemented**:
- `set fprompt tog <element>` — toggle bars/pos/target/targethealth/targetpos/tank on/off (all ON by default)
- `set fprompt` (no args) — shows custom string (if set) plus toggle table with current state
- `set fprompt default` — resets custom string AND all toggles to defaults
- Toggle state stored in `ConfigOptions` as `fprompt-tog-<name>` (bool), persists across sessions
- Dynamic prompt assembled from enabled toggles, cached as `fprompt-default-compiled`; invalidated on toggle change
- New prompt tags: `{target}`, `{targethealth}`, `{targetpos}`, `{tank}`, `{tankpos}`, `{tankbar}`
- `targetHealthDesc`: healthy/bruised/wounded/badly wounded/near death/dead
- Updated `help/set-prompt.template` with full toggle documentation
- Added `FightPrompt` field to `TextFormats` config struct and `config.yaml`
- Fixed `generic.yaml` startup panic: added second message to every single-item tier in all 7 `separate:` sections (yaml.v3 quirk with single-item block sequences in struct context)

**Files Modified** (5 files):
1. `internal/users/userrecord.prompt.go` — helpers + new tags + dynamic fprompt cache
2. `internal/usercommands/set.go` — `tog` subcommand + improved zero-arg display + `default` clears toggles
3. `_datafiles/world/dogmud/templates/help/set-prompt.template` — full rewrite
4. `internal/configs/config.textformats.go` — added `FightPrompt` field
5. `_datafiles/world/dogmud/combat-messages/generic.yaml` — fixed single-item tier panic

---

### Stage 9.8: Combat Conditions System Refactor ✅ COMPLETED
**Merge commit**: 5877c00
**Fix commit**: 2f7e608 — removed position adjectives (prone/clinched/grounded) from combat name labels; position now shown exclusively via fight prompt {pos}/{targetpos} tags
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

## Phase 10: Skill System Generalization & Cleanup

> **Goal**: Create a clean, generalized foundation where every skill and spell has a
> single authoritative primary stat governing both rolls and progression. Remove all
> legacy GoMud skills that have no place in DOG. This is the prerequisite for all
> subsequent system work.

---

### Stage 10.1: Generalize Skill-Stat & Spell-Stat Association ✅ COMPLETED (merge ba598e7)
**Goal**: Add a `PrimaryStat` field to every skill and spell definition. Replace all hardcoded stat references in progression checks and roll calculations with a single lookup. Adding a new skill or spell in the future requires only a YAML entry — no manual wiring in Go.

**Current Problem**:
- `TrackStatUse("vitality")` and similar calls are manually written per skill in various code paths
- Adding a new skill requires editing progression.go, stat tracking hooks, and roll calculations individually
- Spells have no authoritative stat association

**Final Skill Set** (9 DOG core skills):

| Skill | Primary Stat |
|-------|-------------|
| weapon-combat | dexterity |
| unarmed-combat | dexterity |
| ranged-combat | perception |
| spellcasting | willpower |
| first-aid | perception |
| stealth | dexterity |
| tracking | perception |
| bartering | charisma |
| foraging | perception |

**Changes**:
1. Add `PrimaryStat string` field to skill definition struct in `internal/skills/skills.go`
2. Add `PrimaryStat string` field to spell definition struct in `internal/spells/spells.go`
3. Update `CheckSkillProgression()` to auto-call `TrackStatUse(skill.PrimaryStat)` — no more manual per-skill stat tracking
4. Update roll calculations to read `skill.PrimaryStat` rather than hardcoded stat names
5. Add `primary_stat` field to all skill YAML definitions
6. Add `primary_stat` field to all spell YAML definitions

**Files to Modify** (~8 files, ~150 lines):
1. `internal/skills/skills.go` — Add PrimaryStat field, set for all 9 skills
2. `internal/spells/spells.go` — Add PrimaryStat field
3. `internal/characters/progression.go` — Auto-track PrimaryStat in CheckSkillProgression
4. `internal/combat/calculations.go` — Read PrimaryStat from skill definition
5. `_datafiles/world/dogmud/skills/*.yaml` — Add primary_stat field
6. `_datafiles/world/dogmud/spells/*.yaml` — Add primary_stat field
7. Test files

**Testing**:
- [ ] **Unit Tests**: PrimaryStat loaded correctly for all 9 skills
- [ ] **Unit Tests**: CheckSkillProgression calls TrackStatUse with correct stat
- [ ] **Manual Test**: Use each skill, verify its primary stat receives progression calls
- [ ] **Regression Test**: `go test ./...` passes

**Acceptance Criteria**:
- Every skill and spell has PrimaryStat set in YAML
- CheckSkillProgression auto-tracks the correct stat — no hardcoded stat names anywhere
- Adding a new skill requires only YAML + use-trigger — no changes to progression.go
- All tests pass

**Estimated Changes**: ~150 lines, 8 files

---

### Stage 10.2: Skill Cleanup — Remove Legacy Skills ✅ COMPLETED (merge e9ebd4c)
**Goal**: Remove all legacy GoMud skills with no DOG equivalent. Implement backstab as a stealth-triggered combat bonus. Move pickpocket to the stealth skill. Convert map/inspect/search to free stat-check commands.

**Skills Removed**:

| Skill | Action | Absorbed By |
|-------|--------|-------------|
| psionics | Remove | spellcasting |
| brawling | Remove | unarmed-combat |
| cast | Stub only | New magic system (Phase 11) |
| enchant | Remove | spellcasting |
| protection | Remove | spellcasting |
| trading | Remove | bartering (alias) |
| track | Remove | tracking (alias) |
| dual-wield | Remove as skill | Equipment config — weapon-combat governs it |
| scribe | Remove | No DOG equivalent |
| portal | Remove | Spellcasting spell if needed later |
| peep | Remove | Doesn't fit DOG |
| skulduggery | Remove | backstab → stealth combat; pickpocket → stealth |
| tame | Remove | Spellcasting spell (Phase 11) |

**Free Commands** (no longer skill-gated):

| Old Skill | New Behavior |
|-----------|-------------|
| map | Available to everyone, no skill required |
| inspect | Perception stat check, no skill gate |
| search | Perception stat check, no skill gate |

**Backstab — Stealth Combat Bonus**:
- When a character has active stealth and initiates combat, their first attack is a backstab
- Applies to weapon-combat, unarmed-combat, or ranged-combat based on equipped weapon
- Roll: Dexterity + stealth skill vs target Perception
- On success: +50% damage and +20% hit chance on first strike, then stealth breaks
- Message: "Striking from the shadows, you catch {target} completely off guard!"

**Pickpocket → Stealth**:
- `pickpocket <target>` becomes a stealth sub-command
- Requires active stealth or nearby concealment
- Roll: Dexterity + stealth vs target Perception
- Success: steal one small random item from target's inventory

**Brawling Sub-Commands** (retire — covered by newer systems):
- `brawling disarm` — retire (disarms happen organically via crits, Stage 8.4)
- `brawling recover` — retire (`stand` command covers this)
- `brawling tackle` — retire (`grapple` command covers this, Stage 8.2)
- `brawling throw` — retire (grapple system covers this)

**Changes**:
1. Delete/stub: `skill.brawling.*.go`, `skill.dualwield.go`, `skill.enchant.go`, `skill.peep.go`, `skill.portal.go`, `skill.protection.*.go`, `skill.scribe.go`, `skill.tame.go`, `skill.skulduggery.bump.go`
2. Remove skill definitions from `internal/skills/skills.go`
3. Add backstab check in combat initiation (when stealth was active)
4. Create `internal/usercommands/skill.stealth.pickpocket.go` (moved from skulduggery)
5. Remove skill gate from map; convert inspect/search to Perception checks
6. Remove all legacy skills from YAML data files
7. Stub `cast` command: "You reach for the folds of reality, but the technique escapes you. [Coming soon]"

**Files to Modify** (~30 files, ~400 lines):
1. `internal/skills/skills.go` — Remove 13 legacy skill definitions
2. `internal/usercommands/skill.*.go` — Delete ~12 files, modify ~3
3. `internal/combat/combat.go` — Add backstab check on combat initiation
4. `internal/usercommands/skill.stealth.pickpocket.go` — New file
5. `internal/usercommands/map.go` — Remove skill gate
6. `internal/usercommands/skill.inspect.go` — Convert to Perception check
7. `internal/usercommands/skill.search.go` — Convert to Perception check
8. YAML skill/mob/item data files — Remove legacy skill references
9. Test files

**Testing**:
- [ ] **Unit Tests**: Verify exactly 9 DOG skills in skill registry
- [ ] **Manual Test**: Stealth then initiate combat — backstab bonus applies on first hit only
- [ ] **Manual Test**: Pickpocket a target while stealthed
- [ ] **Manual Test**: Map, inspect, search work with no skill requirement
- [ ] **Manual Test**: Dual-wield two weapons — still functions, governed by weapon-combat
- [ ] **Grep Test**: `grep -r "brawling\|skulduggery\|psionics\|tame" --include="*.go"` → zero results
- [ ] **Regression Test**: All combat, movement, and progression still work

**Acceptance Criteria**:
- Exactly 9 skills remain: weapon-combat, unarmed-combat, ranged-combat, spellcasting, first-aid, stealth, tracking, bartering, foraging
- No references to removed skills in any .go files
- Backstab applies for all three combat skill types when entering from stealth
- Map/inspect/search work for all players regardless of skills
- Dual-wield functions as equipment configuration
- All tests pass

**Estimated Changes**: ~400 lines, 30 files

---

## Phase 11: Magic System Rework — Fold-Based Casting

> **Lore**: Magic on Gaius is the physical manifestation of belief. The caster
> bisects their inner vision of reality — each "fold" doubling the intensity of
> their intention — until the accumulated conviction reshapes the world.
>
> **Design Philosophy**: Casting is slow, interruptible, resource-intensive, and
> devastatingly powerful when completed. Glass cannon archetype: a completed spell
> hits harder than 6–10 physical attacks, but the caster is completely vulnerable
> during preparation. Interrupting a spellcast is a valid and rewarding tactic.

---

### Stage 11.1: Disconnect Old Casting System ✅ COMPLETED (merged with Stage 11.2, merge 6c13581)
**Goal**: Remove the old mana/cast-skill casting execution flow cleanly. Preserve spell data structures (YAML schemas and loading code) for reuse in Stage 11.4.

**Changes**:
1. Remove casting execution from `skill.cast.go` — replace with informative stub
2. Remove old conviction-deduction-on-cast logic (conviction system stays; new casting uses it)
3. Preserve `internal/spells/spells.go` spell data loading — YAML schemas will be reused
4. Add placeholder fields to spell struct: `BaseFolds int`, `TargetDefenseType string` (populated in Stage 11.4)
5. Delete `skill.enchant.go`, `skill.protection.*.go` if not already removed in Stage 10.2
6. Stub `cast` message: "You reach for the folds of reality, but the technique escapes you."

**Files to Modify** (~5 files, ~100 lines):
1. `internal/usercommands/cast.go` — Replace with stub
2. `internal/spells/spells.go` — Add placeholder fields, keep data loading intact
3. Test files

**Acceptance Criteria**:
- Old casting system no longer executes
- Spell data loads cleanly with placeholder fields
- `cast` gives a meaningful stub message
- No compilation errors, all tests pass

**Estimated Changes**: ~100 lines, 5 files

---

### Stage 11.2: Fold Engine Core ✅ COMPLETED (merge 6c13581)
**Goal**: Implement the multi-round fold-based casting state machine. Round 1 initiates casting and calculates folds needed. Subsequent rounds accumulate folds until the spell resolves.

**Fold Mechanics**:
```
Round 1 (Initiation):
  - Willpower check (difficulty: base spell difficulty)
  - Fail: no casting begins, cooldown applied
  - Pass: calculate folds_needed
      folds_needed = base_folds × ceil(target_defense / 100)
      rounded up to next power of 2: 2, 4, 8, 16, 32...
  - Announce: "You gather your will and form an image of [spell outcome]."

Subsequent Rounds (Folding):
  - folds_per_round = max(1, round((Perception + spellcasting_skill) / 100))
  - Announce: "You fold your inner vision X times. You now hold Y folds."
  - When folds_accumulated >= folds_needed → resolve spell (Stage 11.4)

Cancellation:
  - Player uses a non-casting command, or types `cancel`
  - Concentration check fails (Stage 11.3)
  - Partial conviction cost proportional to folds already completed
```

**Character State** (not persisted — `yaml:"-"`):
```go
type CastingState struct {
    SpellId          string
    FoldsNeeded      int
    FoldsAccumulated int
    FoldsPerRound    int
    PowerSource      string  // "sunlight", "flame", "health"
    TargetId         int
    ConvictionSpent  int
}
```

**Changes**:
1. Create `internal/combat/casting.go` — CastingState struct, fold calculation, initiation logic
2. Add `CastingState *CastingState` to Character struct (yaml:"-")
3. New `cast <spell> [target]` command — initiates casting, sets CastingState
4. New `cancel` command — cancels active casting with partial conviction cost
5. Update `NewRound_DoCombat` hook — process one fold step per round for casting characters
6. Extend combat restriction system (Stage 7.2) to block other commands while casting

**Files to Modify** (~7 files, ~400 lines):
1. `internal/combat/casting.go` — New file: fold engine
2. `internal/characters/character.go` — Add CastingState field
3. `internal/usercommands/cast.go` — Full command (replaces stub)
4. `internal/usercommands/cancel.go` — New command
5. `internal/hooks/NewRound_DoCombat.go` — Process folds per round
6. `internal/usercommands/` — Extend combat restriction for active casting
7. Test files

**Testing**:
- [ ] **Unit Tests**: Fold calculation (base × defense multiplier, power-of-2 rounding)
- [ ] **Unit Tests**: Folds per round at various Perception/skill levels
- [ ] **Manual Test**: Cast a spell — verify fold progress messages across rounds
- [ ] **Manual Test**: Cancel mid-cast — verify partial conviction cost
- [ ] **Manual Test**: Try to move while casting — verify blocked
- [ ] **Regression Test**: `go test ./...` passes

**Acceptance Criteria**:
- Multi-round casting works end-to-end
- Folds needed calculates correctly (power-of-2 rounding)
- Folds per round scales with Perception and spellcasting skill
- Cancelling costs partial conviction proportional to progress
- Other commands blocked during active casting
- All tests pass

**Estimated Changes**: ~400 lines, 7 files

---

### ⚠️ Testing Note: New Characters Need Starting Spells
New characters currently have no spells in their spellbook, making it impossible to test the magic system manually. Before Stage 11.4 testing, grant new characters a small set of starter spells (e.g., `heal`, `throw-stone`, `fire-bolt`) directly in the character creation flow or via the `New()` function in `internal/characters/character.go`. These can be stripped out later — or reframed as spells the player learns during the tutorial area (Stage 16+). Track this as a prerequisite for Stage 11.4 manual verification.

---

### Stage 11.3: Concentration Mechanics ✅ COMPLETED (merge d5bf9d6)
**Goal**: Implement the three power sources that affect fold accumulation speed. Add concentration checks when the caster is hit during casting.

**Power Sources**:

| Source | Detection | Folds/Round Multiplier | Cost |
|--------|-----------|----------------------|------|
| Sunlight | Outdoors room flag + daytime | 1.0× (baseline) | Conviction per fold |
| Flame | Torch/lantern equipped, or fire present in room | 1.5× | Conviction per fold |
| Health | Always available | 2.0× | HP per fold (configurable) |

- Specify via `cast <spell> [target] with <source>` syntax
- If unspecified, priority: flame > sunlight > health

**Concentration Mechanics**:
- When caster takes damage while casting: Willpower vs (damage / maxHealth × 100) check
- Pass: "You maintain your concentration despite the blow."
- Fail: casting cancelled, partial conviction kept, 1-round casting initiation penalty (`ConditionConcentrationBreak`)
- Being knocked Prone: automatic concentration failure

**Changes**:
1. Add `DetectPowerSource(char, room) string` to `casting.go`
2. Apply source multiplier to fold-per-round calculation
3. Add HP deduction for health-source casting
4. Trigger concentration check in combat damage path
5. Add `ConditionConcentrationBreak` to conditions system
6. Parse `with <source>` in cast command

**Files to Modify** (~5 files, ~250 lines):
1. `internal/combat/casting.go` — Source detection and HP cost logic
2. `internal/combat/conditions.go` — Add ConditionConcentrationBreak
3. `internal/hooks/NewRound_DoCombat.go` — Trigger concentration check on damage dealt to caster
4. `internal/usercommands/cast.go` — Parse `with <source>` syntax
5. Test files

**Testing**:
- [ ] **Manual Test**: Cast outdoors at midday — verify 1.0× sunlight multiplier
- [ ] **Manual Test**: Cast holding a torch — verify 1.5× flame multiplier (faster folds)
- [ ] **Manual Test**: Cast with health source — verify HP deducted each fold
- [ ] **Manual Test**: Take a hit while casting — verify concentration check fires
- [ ] **Manual Test**: Get knocked prone while casting — automatic cancellation

**Acceptance Criteria**:
- All three sources detected and multiplier applied correctly
- Health source costs HP instead of conviction per fold
- Concentration checks fire on every hit during casting
- Prone automatically breaks concentration
- All tests pass

**Estimated Changes**: ~250 lines, 5 files

---

### Stage 11.4: Core Spells & Components ✅ COMPLETED (merge 80045f3)
**Goal**: Port and rewrite core spells to the fold system. Implement component checking before casting begins. Add spell resolution with critical outcomes.

**Core Spell Set**:

| Spell | School | Base Folds | Target Defense | Component | Effect |
|-------|--------|-----------|----------------|-----------|--------|
| Throw Stone | Elemental | 4 | Physical | stone item in inventory | High physical damage |
| Fire Bolt | Elemental | 8 | Physical | None | High fire damage |
| Heal | Vital | 4 | None (self/ally) | None | Restore health |
| Minor Shield | Enhancement | 4 | None (self) | None | +defense for N rounds |
| Stun | Mental | 8 | Mental (Willpower) | None | Stun condition on target |
| Blind | Mental | 8 | Mental (Willpower) | None | Blind condition on target |
| Tame | Vital | 16 | Mental (animal) | None | Animal becomes passive follower |
| Fireball | Elemental (AOE) | 16 | Highest physical defense in room | None | Area fire damage |

**AOE Rule**: For AOE spells, folds are calculated against the highest defense value in the room.

**Component System**:
- Components checked at cast initiation, before any folds begin
- Required item must be in inventory, tagged with matching `component_tag`
- Consumed on successful spell resolution

**Spell Resolution** (when folds reach target):
1. Final spellcasting roll vs target defense (physical or mental)
2. Success: full effect applied
3. Failure: fizzles, partial conviction cost kept
4. Crit success (z > 2.0): double effect magnitude
5. Crit failure (z < -2.0): spell backfires (self-damage or wild effect)

**Changes**:
1. Update `casting.go` with spell resolution logic
2. Rewrite all spell YAML files with `base_folds`, `target_defense_type`, `component_tag`, `effect_type`, `effect_magnitude`
3. Update `internal/spells/spells.go` struct with new fields
4. Add `ComponentTag []string` to ItemSpec for component items
5. Implement AOE defense calculation (scan room for highest defense)
6. Connect spell effects to existing condition/damage/heal systems

**Files to Modify** (~12 files, ~500 lines):
1. `internal/combat/casting.go` — Spell resolution logic
2. `_datafiles/world/dogmud/spells/*.yaml` — Full rewrite of all spell definitions
3. `internal/spells/spells.go` — Update spell struct with new fields
4. `internal/items/itemspec.go` — Add ComponentTag field
5. Test files

**Testing**:
- [ ] **Manual Test**: Throw Stone — requires stone in inventory, multi-round, big damage
- [ ] **Manual Test**: Heal self and an ally
- [ ] **Manual Test**: Fireball — AOE hits all targets, folds use highest defense
- [ ] **Manual Test**: Tame an animal mob — becomes passive
- [ ] **Manual Test**: Crit success doubles effect; crit failure causes backfire
- [ ] **Balance Test**: Completed spell hits harder than 6–10 equivalent physical attacks

**Acceptance Criteria**:
- All 8 core spells functional under the fold system
- Components checked before casting begins, consumed on resolution
- AOE uses highest room defense for fold calculation
- Crit success and failure both trigger correctly
- Taming spell works on animal-type mobs
- All tests pass

**Estimated Changes**: ~500 lines, 12 files

---

### Hotfix: Proportional stdDev & Physical Armor Formula ✅ COMPLETED (merge 0eb9d38)
### Refactor: RollSpread Master Knob ✅ COMPLETED (merge a66eee4)
**Goal**: Fix two systematic errors introduced in Stage 11.4 and present throughout existing combat code.

**Fixes**:
1. All `dice.Roll` / `dice.OpposedRoll` calls used flat `15.0` stdDev — replaced with `dice.StdDevFor(mean)` = `mean * 0.15` (floor 1.0) across 11 files
2. `spell_resolution.go` physical defense incorrectly included `Vitality.ValueAdj` — removed; physical defense = equipment `DamageReduction` + `ConditionShield` only
3. Added `dice.StdDevFor()` helper to `internal/dice/dice.go`
4. Updated `CLAUDE.md` with accurate rules: use-based progression (no levels/XP), proportional stdDev, 3-source armor model, file naming conventions

**Files changed**: `dice.go`, `combat.go`, `grapple.go`, `character.go`, `spell_resolution.go`, `bash/kick/trip.go` (user + mob), `CLAUDE.md`

---

### Hotfix: Descriptive Messages, Minor Shield Status Display & Duration ✅ COMPLETED (merge ce3ba22)
**Goal**: Polish the Stage 11.4 spell resolution output and fix a Minor Shield bug.

**Fixes**:
1. All raw numeric values removed from player-facing spell messages — damage uses `combat.GetDamageDescription()`, healing uses new `combat.GetHealDescription()`, shield and backfire use descriptive text
2. Minor Shield armor bonus now included in `GetDefense()` so it shows in the `status` command's Armor line
3. Minor Shield duration changed from hardcoded `10` to `10 + round(skillLevel/5)` — scales with spellcasting investment
4. Added `CLAUDE.md` rule: never display raw numbers in player messages; use descriptive language

**Files changed**: `character.go` (GetDefense), `spell_resolution.go` (messages + duration), `combat/descriptions.go` (GetHealDescription), `CLAUDE.md`

---

### Hotfix: Spell Help Files ✅ COMPLETED (merge 34efec5)
**Goal**: Update stale help templates that still described old skill-based mechanics.

**Fixes**:
1. `spell.template` — generic fallback now shows Base Folds, conditional Resisted-by (physical/mental), and conditional Requires (component tag); covers all spells without custom templates
2. `heal.template` — rewritten: describes fold-based healing spell, removes cooldown/level references
3. `tame.template` — rewritten: removes old level-cap/learned-creature text; describes opposed mental roll, animal-only restriction, 24h charm, backfire risk
4. `minor-shield.template` — new file: explains armor bonus formula, skill-scaled duration, status display integration

**Files changed**: `spell.template`, `heal.template`, `tame.template`, `minor-shield.template` (new)

---

### Stage 11.5: Combat Integration & NPC Caster AI ✅ COMPLETED (merge 10bc02c)
**Goal**: Fully integrate casting into the combat action economy. Update NPC AI with a caster archetype that uses the fold system.

**Changes**:
1. Casting initiation shares cooldown slot with special attacks (bash/trip/kick) — cannot start casting the same round a special move was used, and vice versa
2. Add `caster` AI profile to `internal/combat/ai.go`:
   - Prefers initiating casts over physical attacks when health is high
   - Evaluates whether folds can plausibly complete before likely death
   - Prioritizes defensive actions when being hit mid-cast (tries to maintain concentration)
3. Add `{casting}` prompt tag: shows `Casting: Throw Stone [4/8 folds]` when actively casting
4. Mob fold casting — mobs use CastingState instead of legacy SpellCast/onMagic path
5. Conviction regen for mobs in NewRound_AutoHeal
6. Startland Practice Arena (room 3) + Caster's Alcove (room 4) with apprentice mage (mob 3)

**Post-completion balance pass** (commit 6056681):
- Spell conviction costs increased ~8x (tame/aidskill unchanged)
- Harm spell effect_magnitude increased 10x (mm, fire-bolt, fireball, sparks, throw-stone)
- `heal` reworked from flat instant heal to `ConditionRegen`: 50 HP/tick every 3 rounds for `max(1, skillLevel/10)` ticks; works in combat
- `ConditionRegen` added to `characters/conditions.go`; processed in `NewRound_AutoHeal` for players and mobs
- `mobcommands/cast.go` gates mob casting with the same `special-move` cooldown as players

**Files changed**: `internal/combat/ai.go`, `internal/hooks/NewRound_DoCombat.go`, `internal/hooks/NewRound_AutoHeal.go`, `internal/hooks/spell_resolution.go`, `internal/mobcommands/cast.go`, `internal/usercommands/skill.cast.go`, `internal/users/userrecord.prompt.go`, `internal/characters/conditions.go`, all spell YAMLs, startland rooms 2–4, apprentice mage mob

---

## Phase 12: Mutations

> **Lore**: The Chrysalis infects all humans on Gaius. Over time the symbiosis
> deepens, and beliefs made physical manifest as mutations. They are not chosen —
> they emerge. And they always come with a cost.
>
> **Design**: Every mutation has a pro and a con. The system is fully data-driven:
> adding new mutations requires only a YAML file, zero Go code changes.

---

### Stage 12.1: Mutation Framework & First 10 Mutations
**Status**: ✅ COMPLETED — merge commit ebc22e0

**Goal**: Build the mutation data system, acquisition mechanics, and implement 10 starter mutations with pro/con effects.

**Data Structure**:
```go
type Mutation struct {
    Id          string          // YAML key: "fast-reflexes"
    Name        string          // Display: "Fast Reflexes"
    Description string          // Flavor text
    Pro         MutationEffect
    Con         MutationEffect
    Rarity      int             // 1–10, affects acquisition weight (higher = rarer)
    Visual      string          // Added to character's look description
}

type MutationEffect struct {
    Type   string  // "stat_multiplier", "stat_flat", "natural_armor", "natural_weapon", etc.
    Target string  // Stat name or system target
    Value  float64
}
```

**First 10 Mutations**:

| Mutation | Pro | Con | Visual |
|----------|-----|-----|--------|
| Fast Reflexes | +10% Dexterity | -5% Strength | "moves with uncanny speed" |
| Tough Skin | Natural armor +5 | -5% Dexterity | "skin has a leathery, scaled texture" |
| Dense Muscles | +15% Strength for damage | +10% stamina cost per action | "unnaturally thick musculature" |
| Clawed Hands | Natural claw weapon, +10% grapple | -20% Bartering effectiveness | "fingers end in curved claws" |
| Keen Eyes | +15% Perception | Penalty in bright outdoor light | "eyes catch light in an inhuman way" |
| Iron Constitution | +20% max health | -15% stamina regen rate | "a dense, heavyset frame" |
| Hollow Bones | -20% encumbrance penalty | -15% max health | "lighter than expected when touched" |
| Adrenaline Surge | At <25% health: +20% damage and speed | Post-combat exhaustion (-50% stamina for 5 rounds) | "veins visibly pulse under the skin" |
| Thermal Regulation | Fire/cold damage reduced 15% | None (minor mutation) | "skin radiates unusual warmth" |
| Pheromone Glands | +15% Charisma for Bartering | Predatory mobs more likely to aggro you | "a faint musk that most find strangely compelling" |

**Acquisition Mechanics**:
- `MutationProgress float64` on Character (`yaml:"-"`) — increments each combat round during play
- When MutationProgress crosses a threshold: roll weighted by Rarity against mutation pool
- Cannot acquire duplicates; maximum mutations configurable (default 5)
- On acquisition: announce to player, apply effects permanently to character

**Commands**:
- `mutations` — list active mutations with descriptions and current level
- Mutations section added to `score` output

**Files to Create/Modify** (~10 files, ~450 lines):
1. `internal/mutations/mutations.go` — New package: struct, registry, effect application
2. `internal/mutations/mutations_test.go`
3. `internal/characters/character.go` — Add `Mutations map[string]int` (ID→level), `MutationProgress float64`
4. `internal/hooks/NewRound_UserRoundTick.go` — Tick MutationProgress, trigger acquisition check
5. `internal/usercommands/mutations.go` — New command
6. `internal/usercommands/score.go` — Add mutations section
7. `_datafiles/world/dogmud/mutations/*.yaml` — 10 mutation definition files
8. Test files

**Testing**:
- [ ] **Unit Tests**: Effect application for each effect type (stat_multiplier, natural_weapon, etc.)
- [ ] **Unit Tests**: Acquisition roll weighting by Rarity
- [ ] **Manual Test**: Play through combat — mutation eventually acquired
- [ ] **Manual Test**: `mutations` command lists active mutations correctly
- [ ] **Manual Test**: Clawed Hands gives natural weapon in unarmed combat
- [ ] **Manual Test**: Adrenaline Surge triggers correctly at low health
- [ ] **Manual Test**: Tough Skin reduces incoming damage

**Acceptance Criteria**:
- 10 mutations defined with correct pro/con effects
- Acquisition fires automatically during normal gameplay
- Adding a new mutation requires only a YAML file — zero Go code changes
- `mutations` command and score section display active mutations
- All tests pass

**Estimated Changes**: ~450 lines, 10 files

---

### Stage 12.2: Mutation Deepening & Visual Integration ✅ COMPLETED (107c187)
**Goal**: Allow mutations to strengthen over time (Level 1–3). Integrate mutation visuals into character descriptions.

**Mutation Deepening**:
- Each acquired mutation has a `Level int` (1–3) stored in `Mutations map[string]int`
- Same MutationProgress mechanic triggers level-up for existing mutations (threshold higher for each level)
- Level 2 and 3 scale both pro and con effect values proportionally
- Example: Tough Skin L1: +5 armor / -5% Dex; L3: +15 armor / -15% Dex

**Visual Integration**:
- `look <player>` includes mutation Visual strings naturally in their description
- Multiple mutations stack their visual descriptions
- `score` shows mutation name + current level

**Configuration Preparation** (for Phase 14):
- `MutationBaseRate` and `MutationMaxCount` surfaced as named constants ready to be wired into balance config

**Files to Modify** (~5 files, ~200 lines):
1. `internal/mutations/mutations.go` — Add deepening logic using Level field
2. `internal/characters/character.go` — Level already in map[string]int from Stage 12.1
3. `internal/hooks/NewRound_UserRoundTick.go` — Add deepening check alongside acquisition
4. `_datafiles/world/dogmud/mutations/*.yaml` — Add level 2/3 effect values per mutation
5. Test files

**Acceptance Criteria**:
- Mutations deepen from Level 1 → 2 → 3 over extended play
- Level 3 mutations are noticeably more impactful (both pro and con)
- Mutation visuals appear in `look` descriptions
- All tests pass

**Estimated Changes**: ~200 lines, 5 files

---

## Phase 13: Basic Crafting

> **Goal**: A crafting system that is functional, extensible, and tutorial-ready.
> Each crafting skill has at least two working recipes. Adding new recipes later
> requires only a YAML file — zero Go code changes.

---

### Stage 13.1: Crafting Framework ✅ COMPLETED (merge: b04e22e)
**Goal**: Build the core crafting system: new crafting skills, recipe data structure, crafting station support, and the `craft` command.

**New Crafting Skills** (added to the 9 DOG core skills — 11 total):

| Skill | Primary Stat | Covers |
|-------|-------------|--------|
| blacksmithing | Strength | Metal weapons, armor, tools |
| alchemy | Perception | Potions, salves, medicines from gathered herbs |

**Recipe YAML Format** (`_datafiles/world/dogmud/recipes/<skill>/<recipe>.yaml`):
```yaml
id: iron-dagger
name: Iron Dagger
skill: blacksmithing
skill_minimum: 10
station: forge
time_rounds: 3
ingredients:
  - item_tag: iron-ingot
    quantity: 1
  - item_tag: leather-strip
    quantity: 1
output:
  item_id: iron-dagger
  quantity: 1
success_message: "You hammer the iron into a sharp blade and wrap the handle tightly."
failure_message: "The metal cracks from uneven heating. The materials are ruined."
```

**Craft Command**:
- `craft list` — show all recipes learnable/known at current skill levels
- `craft <recipe name>` — attempt craft if at correct station with required materials
- Crafting takes N rounds (interruptible if attacked, like casting)

**Crafting Stations**:
- Room YAML gets optional `station` flag: `forge`, `alchemy_bench`, `workbench`
- Recipes requiring a station check the current room's station type

**Files to Create/Modify** (~10 files, ~450 lines):
1. `internal/crafting/crafting.go` — New package: recipe loading, checks, execution
2. `internal/crafting/crafting_test.go`
3. `internal/skills/skills.go` — Add Blacksmithing and Alchemy definitions
4. `internal/characters/character.go` — Add CraftingState (similar to CastingState)
5. `internal/usercommands/craft.go` — New command
6. `internal/rooms/rooms.go` — Add Station field to room struct
7. `_datafiles/world/dogmud/recipes/` — New directory
8. Test files

**Acceptance Criteria**:
- `craft list` shows available recipes for character's current skill levels
- Crafting executes over multiple rounds, interruptible by combat
- Recipes require correct station type and materials in inventory
- Failed crafts consume materials (lower chance at low skill)
- Adding a new recipe requires only a YAML file — zero Go code changes
- All tests pass

**Estimated Changes**: ~450 lines, 10 files

---

### Stage 13.2: First Recipes & Foraging Integration ✅ COMPLETED (merge: 9edb418)
**Goal**: Implement at least two recipes per crafting skill. Connect Foraging output to crafting as the primary material source. Ensure recipes are tutorial-ready.

**First Recipes**:

| Skill | Recipe | Ingredients | Output |
|-------|--------|-------------|--------|
| Blacksmithing | Iron Dagger | 1 iron ingot, 1 leather strip | iron dagger (weapon) |
| Blacksmithing | Iron Shield | 2 iron ingots, 1 wooden plank | iron shield (off-hand) |
| Alchemy | Healing Poultice | 2 healer's root, 1 cloth strip | healing poultice (consumable, restores health) |
| Alchemy | Stamina Draught | 2 bitter thistle, 1 small vial | stamina draught (consumable, restores stamina) |

**Foraging → Materials**:
- Foraging now yields typed material items tagged for crafting use:
  - Forest/field: healer's root, bitter thistle, cloth fiber
  - Rocky terrain: iron ore (smelted at forge into iron ingot)
  - General drops: leather strip (looted from animals), small vial (purchasable)
- Each material item has a `component_tag` matching recipe ingredient tags

**Tutorial Integration**:
- Forge room and alchemy bench rooms flagged in tutorial area plan (Phase 16)
- Each recipe teachable via NPC `teach <recipe>` interaction
- New help file: `help crafting`

**Files to Create/Modify** (~12 files, ~300 lines):
1. `_datafiles/world/dogmud/recipes/blacksmithing/` — 2 recipe YAML files
2. `_datafiles/world/dogmud/recipes/alchemy/` — 2 recipe YAML files
3. `_datafiles/world/dogmud/items/materials/` — Material item definitions
4. `internal/usercommands/forage.go` — Update to yield typed material items
5. `_datafiles/world/dogmud/templates/help/crafting.template` — New help file
6. Test files

**Acceptance Criteria**:
- All 4 recipes work end-to-end (forage → craft → usable item)
- Foraging yields correctly-tagged materials
- All recipes teachable by an NPC
- All tests pass

**Estimated Changes**: ~300 lines, 12 files

---

## Phase 14: Balance Configuration

> **Why here**: All core systems now exist — combat, magic, mutations, crafting,
> and progression. We know exactly what needs to be tunable. Building this
> config earlier would have required constant revision as new systems were added.
>
> **Design**: One YAML file, well-organized with section headers. Big knobs at
> the top of each section for broad adjustments; granular per-element knobs
> below. Heavily commented. New systems add a new section — existing sections
> are never restructured, keeping the file stable and easy to reason about.

---

### Stage 14.1: Central Balance Configuration File
**Goal**: Create `balance.yaml` — a single comprehensive config file covering all balance-relevant constants across all systems. Audit the codebase for hardcoded values and surface them as config entries.

**Config File Structure** (`_datafiles/world/dogmud/balance.yaml`):
```yaml
# =============================================================================
# DOGMud Balance Configuration
# =============================================================================
# BIG KNOBS: adjust these first — they scale entire systems
# SMALL KNOBS: per-element tuning after big knobs feel right
# New systems: add a new section below, do not modify existing sections
# =============================================================================

# -----------------------------------------------------------------------------
# COMBAT
# -----------------------------------------------------------------------------
combat:
  # Big knobs
  global_damage_multiplier: 1.0        # Scale all damage up/down
  global_defense_multiplier: 1.0       # Scale all avoidance rates
  global_stamina_drain_multiplier: 1.0 # Scale all stamina costs in combat
  # Defense rates
  dodge_base_multiplier: 0.9
  parry_base_multiplier: 0.9
  block_base_multiplier: 0.9
  # Critical hits
  crit_success_threshold: 2.0          # z-score: above = crit success (~2.3%)
  crit_failure_threshold: -2.0         # z-score: below = crit failure (~2.3%)
  crit_damage_multiplier: 2.0
  fumble_self_damage_multiplier: 0.25
  # Stamina costs per action
  attack_stamina_cost: 5
  dodge_stamina_cost: 3
  parry_stamina_cost: 3
  block_stamina_cost: 2
  # Grapple
  grapple_prone_defense_penalty: 0.3
  grapple_third_party_penalty: 0.3

# -----------------------------------------------------------------------------
# PROGRESSION
# -----------------------------------------------------------------------------
progression:
  # Big knobs
  global_skill_progression_multiplier: 1.0
  global_stat_progression_multiplier: 1.0
  # Skill soft cap behaviour
  skill_uses_per_virtual_rank: 25
  skill_base_progression_chance: 0.30
  skill_soft_cap: 50
  # Per-skill multipliers (override global; higher = progresses faster per use)
  skill_multipliers:
    weapon-combat: 0.3      # Fires many times per round — slowed down
    unarmed-combat: 0.3
    ranged-combat: 0.3
    spellcasting: 0.5
    first-aid: 2.0          # Used rarely — sped up
    stealth: 1.0
    tracking: 2.0
    bartering: 2.0
    foraging: 2.0
    blacksmithing: 2.0
    alchemy: 2.0
  # Stat progression
  stat_uses_to_progress: 50
  stat_natural_max: 200
  # Per-stat rates (relative to each other)
  stat_progression_rates:
    strength: 1.0
    dexterity: 1.0
    perception: 1.0
    vitality: 1.0
    willpower: 1.0
    charisma: 1.0

# -----------------------------------------------------------------------------
# MAGIC
# -----------------------------------------------------------------------------
magic:
  # Big knobs
  global_spell_damage_multiplier: 1.0
  global_fold_cost_multiplier: 1.0     # Scale folds needed for all spells
  # Power source fold-per-round multipliers
  source_sunlight_multiplier: 1.0
  source_flame_multiplier: 1.5
  source_health_multiplier: 2.0
  source_health_hp_cost_per_fold: 5    # HP cost when using health as source
  # Concentration
  concentration_check_base_difficulty: 50
  concentration_failure_penalty_rounds: 1
  # Spell resolution
  spell_crit_damage_multiplier: 2.0
  spell_backfire_self_damage_pct: 0.25 # Fraction of spell damage on backfire

# -----------------------------------------------------------------------------
# MUTATIONS
# -----------------------------------------------------------------------------
mutations:
  # Big knobs
  global_acquisition_rate_multiplier: 1.0   # Scale acquisition speed globally
  global_deepening_rate_multiplier: 1.0     # Scale how fast mutations level up
  max_mutations_per_character: 5
  # Per-mutation rarity weights defined in individual mutation YAML files

# -----------------------------------------------------------------------------
# CRAFTING
# -----------------------------------------------------------------------------
crafting:
  global_success_rate_multiplier: 1.0
  consume_materials_on_failure: true        # If false, materials returned on fail
  output_quality_variance: 0.1             # ±10% variance in crafted item stats

# -----------------------------------------------------------------------------
# STAMINA
# -----------------------------------------------------------------------------
stamina:
  move_flat_terrain_cost: 2
  move_rough_terrain_multiplier: 2.5
  move_encumbrance_multiplier: 1.5
  out_of_combat_regen_per_round: 5
  in_combat_regen_per_round: 1

# -----------------------------------------------------------------------------
# ECONOMY
# -----------------------------------------------------------------------------
economy:
  barter_price_variance: 0.20              # ±20% price swing from bartering
  shop_markup_multiplier: 1.5              # Shop prices vs base item value
```

**Implementation**:
1. Create `_datafiles/world/dogmud/balance.yaml` with the structure above
2. Create `internal/configs/config.balance.go` — load and expose all balance values
3. Audit all hardcoded constants across combat, magic, progression, mutations, crafting
4. Replace each hardcoded value with a reference to the loaded balance config
5. Verify every key in balance.yaml has an explanatory comment

**Files to Modify** (~18 files, ~350 lines):
1. `internal/configs/config.balance.go` — New file: BalanceConfig struct + loader
2. `_datafiles/world/dogmud/balance.yaml` — New file
3. `internal/combat/calculations.go` — Replace hardcoded constants
4. `internal/characters/progression.go` — Replace hardcoded constants
5. `internal/combat/casting.go` — Replace hardcoded constants
6. `internal/mutations/mutations.go` — Replace hardcoded constants
7. `internal/crafting/crafting.go` — Replace hardcoded constants
8. Other files with hardcoded balance values discovered during audit

**Testing**:
- [ ] **Unit Tests**: Config loads correctly, all fields populate with expected defaults
- [ ] **Manual Test**: Set `global_damage_multiplier: 2.0` — verify noticeably faster fights
- [ ] **Manual Test**: Set `global_skill_progression_multiplier: 5.0` — verify faster skill gain
- [ ] **Grep Test**: Search codebase for remaining hardcoded balance constants
- [ ] **Regression Test**: Default config values produce identical gameplay behavior as before

**Acceptance Criteria**:
- All balance constants sourced from `balance.yaml`
- No hardcoded balance values remain in Go files
- Every config key has an explanatory comment in the YAML
- Adding a new configurable value requires only: field in struct + line in YAML
- Default values produce the same gameplay as before Phase 14
- All tests pass

**Estimated Changes**: ~350 lines, 18 files

---

## Phase 15: Dev Tools

> **Goal**: Give developers (human and AI) the tools to build zones efficiently.
> Zone consistency checking catches silent broken-exit bugs before they reach
> players. The grid generator scaffolds a zone in seconds. The JSON API makes
> all tools callable by AI agents building zones from prompts.

---

### ✅ Stage 15.1: Zone Consistency Checker & Grid Generator — COMPLETED (merge: 55465f3)
**Goal**: Admin commands to verify cardinal exit consistency in a zone, and to generate a rectangular grid of rooms with automatic bidirectional exits.

**Zone Consistency Checker** (`devtool check <zone>`):
- Scans all rooms in the named zone directory
- For each exit in each room: verifies the destination room has the matching reverse exit
- Reports: missing reverse exits, orphan rooms (no in/out connections), dead exits (invalid IDs)
- Example output:
  ```
  Zone: merchants-quarter — 20 rooms scanned
  ✓ Room 100–102: all exits consistent
  ✗ Room 103 (north → 104): Room 104 missing south exit
  ✗ Room 107: orphan — no connections in or out
  2 issues found.
  ```

**Grid Generator** (`devtool makezone <zone_name> <width> <height>`):
- Creates `_datafiles/world/dogmud/rooms/<zone_name>/` directory
- Creates W×H rooms with sequential auto-assigned IDs
- Room names: `"<zone_name> - Room 1"`, `"<zone_name> - Room 2"`, etc.
- Default description: `"A room in <zone_name>. This area has not yet been described."`
- Automatic bidirectional cardinal exits connecting the grid (N/S/E/W)
- Summary: `"Created 20 rooms in 'merchants-quarter' (4×5 grid). IDs 100–119."`

**Files to Create/Modify** (~6 files, ~350 lines):
1. `internal/usercommands/devtool.go` — New command with subcommand dispatch
2. `internal/devtools/consistency.go` — Zone consistency checker
3. `internal/devtools/gridgen.go` — Grid room generator
4. `internal/rooms/rooms.go` — Room YAML serializer (write room to file)
5. Test files

**Acceptance Criteria**:
- `devtool check <zone>` accurately reports all exit inconsistencies
- `devtool makezone <name> <W> <H>` creates a working grid zone
- Generated zones load without errors and pass their own consistency check
- Dev tools require admin flag — not accessible to regular players
- All tests pass

**Estimated Changes**: ~350 lines, 6 files

---

### ✅ Stage 15.2: Zone Linking & AI-Callable JSON API — COMPLETED (merge: 55465f3)
**Goal**: Tool for linking rooms across zones. Expose all dev tools via structured JSON input/output so AI agents can build zones programmatically.

**Zone Linking** (`devtool linkzones <zoneA>/<roomA_id> <direction> <zoneB>/<roomB_id>`):
- Creates bidirectional exit between a room in zone A and a room in zone B
- Validates both rooms exist before creating links
- Runs consistency check on both rooms after linking
- Example: `devtool linkzones startland/1 east merchants-quarter/100`

**AI-Callable JSON API**:
All dev tools accept a JSON command via `devtool json <json_string>` for AI agent access:
```json
{"action": "makezone", "params": {"name": "merchants-quarter", "width": 4, "height": 5}}
```
```json
{"action": "linkzones", "params": {"zone_a": "startland", "room_a": 1, "direction": "east", "zone_b": "merchants-quarter", "room_b": 100}}
```
```json
{"action": "check", "params": {"zone": "merchants-quarter"}}
```

Response format:
```json
{"success": true, "action": "makezone", "result": {"rooms_created": 20, "first_room_id": 100, "last_room_id": 119}}
```

Adding a new dev tool requires only registering one action handler in the dispatch table — no API structural changes.

**Files to Create/Modify** (~5 files, ~300 lines):
1. `internal/devtools/api.go` — New file: JSON dispatch layer
2. `internal/devtools/linkzones.go` — Zone linking logic
3. `internal/usercommands/devtool.go` — Add `devtool json <json>` input mode
4. `docs/DEVTOOLS_API.md` — JSON API reference for AI agent developers
5. Test files

**Acceptance Criteria**:
- Zone linking creates correct bidirectional exits between zones with validation
- All dev tools callable via `devtool json <json>` input
- JSON responses structured consistently (`success`, `action`, `result`/`error`)
- New tools require only one handler registration — no API restructuring
- All tests pass

**Estimated Changes**: ~300 lines, 5 files

---

## Phase 16: Tutorial Area (Sanctum Basin)

> **Goal**: Build the starting area using the Phase 15 dev tools. The tutorial
> teaches all core mechanics: movement, combat, magic, crafting, mutations.
> Players leave with a functional character and an understanding of all core
> systems before entering the open world.

---

### ✅ Stage 16.1: Sanctum Basin Zone Layout — COMPLETED
**Goal**: Create the Sanctum Basin zone skeleton, room files, mob definitions, world-road placeholder, and wire up spawn point.

**Implemented**:
- 20 room YAMLs (101–120) + 20 JS stubs in `_datafiles/world/dogmud/rooms/sanctum-basin/`
- `zone-config.yaml`: name=Sanctum Basin, spawn roomid=107
- World-road placeholder zone (room 201) linked south from World Gate (103)
- 12 mob YAMLs in `_datafiles/world/dogmud/mobs/sanctum-basin/` (IDs 50–56 trainers, 65–69 combat)
- Deleted old tutorial zone (`rooms/tutorial/`, `rooms.instances/tutorial/`, `quests/0-tutorial.yaml`)
- Updated `config.yaml`: `StartRoom: 107`, `TutorialRooms: []`
- Exit design: Basin Gate (102) south→World Gate (103) locked difficulty 999; World Gate south→201 (world-road)

---

### ✅ Stage 16.2: Tutorial Quest Flow & NPC Dialogue — COMPLETED
**Goal**: Wire up the Sanctum Trials quest flow with 6-station NPC dialogue and Basin Gate lock/unlock.

**Implemented**:
- Quest YAML: `quests/1-sanctum_tutorial.yaml` (questid 1, steps: arrive/combat/crafting/alchemy/wilderness/magic/graduate)
- 7 JS room scripts: 113 (Greeter/intro), 114 (Combat Trainer/dummy), 109 (Blacksmith), 111 (Alchemist), 106 (Wilderness Guide), 116 (Elder/LearnSpell), 102 (Basin Warden/gate lock)
- Basin Gate relocks after each player exits; unlocks per-player on `1-graduate`
- All trainers deliver world lore consistent with Gaius/Chrysalis/Fold canon

---

### ✅ Stage 16.3: Tutorial Playtesting Polish — COMPLETED
**Goal**: Post-playtesting fixes and content polish across the Sanctum Basin tutorial zone.

**Implemented**:
- **StartRoom 107→113**: New characters now spawn directly in Academy Hall so the Awakening Rite triggers immediately on first login
- **Ceremony lock system** (`113.js`): Exits lock when a new player enters Academy Hall; Priest intercepts movement commands with "The Rite is not yet complete"; `onIdle` tick counter unlocks after ~30s; `onLoad` guarantees exits open on server restart
- **Mosaic map item** (`13-mosaic_map.yaml`): Floor mosaic in Academy Hall (item 13); `look mosaic` renders ASCII world map of the Windward Marches; pickup intercepted with flavor message; Priest closing line now mentions mosaic
- **Cave final trial** (`120.js`, `102.js`): Added `cave` quest step — player must defeat the Aberrant Chrysalis boss before the Basin Warden will open the gate; `onIdle` boss-death detection mirrors the training dummy pattern; `1-cave` token replaces `1-magic` as the graduation check
- **Fold lore rewrite** (`116.js`): Elder Saris now explains Fold casting via bifurcation model — form an image, split to two, four, eight; each doubling is a fold; harder spells require more folds; Witnesses affect fold-pressure ceiling. Removed premature cave reference. Added `spells` command prompt.
- **Wilderness Guide** (`106.js`): Fen now explicitly prompts `forage`, `track`, and `sneak` commands with in-line explanations
- **5× regen in Sanctum Basin** (`NewRound_AutoHeal.go`): Rooms 101–120 grant 5× HP/stamina regen to reduce frustration during tutorial combat
- **Starter spell cleanup** (`character.go`): Removed `tame`, `fire-bolt`, and `fireball` from default spellbook; `illum` is now earned via Elder Saris during tutorial
- **`mutation` command alias** (`keywords.yaml`): `mutation` now resolves to `mutations`
- **Quest file rename**: `1-sanctum_tutorial.yaml` → `1-the_sanctum_trials.yaml`; mob files renamed to match full NPC names (chrysalis_priest, blacksmith_korvath, etc.)
- **World Road renumber**: Room 201 → 2001 to avoid future ID conflicts
- **Various NPC dialogue polish**: Korvath forge history, Adela trade gossip, 103.yaml abandoned pack, 105.yaml cookfire smoke/Witnesses idle messages, 120.js Aberrant lore on entry

---

## Phase 17: Moon Phase System

### Stage 17.1: Moon Phase Global Emoter ✅ COMPLETED

**Merge commit**: see `development` branch history

**Goal**: Broadcast atmospheric messages to all players when any of the three Witnesses cross a phase boundary — mirroring the existing sunrise/sunset mechanism exactly.

**Mechanism** (same two-file hook pattern as `NewRound_CheckNewDay` → `DayNightCycle` → `NotifySunriseSunset`):

1. `NewRound_CheckMoonPhase.go` (new) — listens on `NewRound`; for each moon computes `phaseRound = currentRound % cycleRounds` for the current and previous round; if any moon crossed a full-moon (50%) or new-moon (0%) threshold, fires a `MoonPhase` event.
2. New `MoonPhase` event type in `eventtypes.go`:
   ```go
   type MoonPhase struct {
       MoonName  string // "Swiftmoon" | "The Wanderer" | "The Eye"
       PhaseName string // "new" | "full"
       IsFull    bool
       IsNew     bool
   }
   ```
3. `MoonPhase_BroadcastEmote.go` (new) — listens on `MoonPhase`; selects the appropriate template by moon+phase; broadcasts via `events.Broadcast` to all players.
4. `hooks.go` — register both new listeners.

**Phase boundaries announced**: new moon and full moon only (2 events × 3 moons = up to 6 announcements per longest cycle). Quarter moons deferred.

**Moon cycle lengths** (hardcoded constants, match world.md lore):
```
Swiftmoon:    4.7  × RoundsPerDay rounds per cycle
The Wanderer: 10.6 × RoundsPerDay rounds per cycle
The Eye:      21.1 × RoundsPerDay rounds per cycle
```

**Timing note**: Round count is persisted to disk (`SaveRoundCount`/`LoadRoundCount`) and reloads on restart. The hook compares round N−1 to round N as normal — no triple-fire risk on reboot and no staggering needed.

**Templates** (6 files, `_datafiles/templates/generic/`):

| File | Text |
|------|------|
| `moon_swiftmoon_new` | Swiftmoon dims and vanishes. It will be back before you miss it. |
| `moon_swiftmoon_full` | Swiftmoon is full tonight — brief and bright. It will not linger. |
| `moon_wanderer_new` | The Wanderer has gone dark. Its absence is its own kind of presence. |
| `moon_wanderer_full` | The Wanderer reaches its apex: a pale disc, slow and deliberate overhead. |
| `moon_eye_new` | The Eye has closed. The old records note these nights without comment. |
| `moon_eye_full` | The Eye is fully open tonight. Those who study the Fold grow quiet. |

**Files to create/modify** (~5 files, ~120 lines):
1. `internal/hooks/NewRound_CheckMoonPhase.go` (new)
2. `internal/hooks/MoonPhase_BroadcastEmote.go` (new)
3. `internal/events/eventtypes.go` — add `MoonPhase` event type
4. `internal/hooks/hooks.go` — register listeners
5. `_datafiles/templates/generic/moon_*.ext` — 6 template files

**Gameplay hook compatibility**: Because `MoonPhase` is a typed event on the bus, any future system registers independently — e.g. a Fold pressure hook does `events.RegisterListener(events.MoonPhase{}, FoldPressureHandler)` and checks `evt.MoonName` / `evt.IsFull`. Zero coupling to the emoter. Planned future hook: The Eye full → raise spell fold ceiling; all three new simultaneously → minimum ceiling (wires into Stage 11.2 Fold engine).

**Testing**:
- Use `devtool settime` (or equivalent) to advance game time to a known moon boundary; verify broadcast fires with correct text
- Verify no double-fire when two moons cross a boundary in the same round
- Verify server restart does not produce spurious phase events

---

### Stage 17.2: Moon Phase Gameplay Effects (Fold Pressure) ✅ COMPLETED

**Merge commit**: see `development` branch history

**Goal**: Wire the Witnesses into actual gameplay — spell fold ceilings, mutation probability, and Aberrant aggression all shift with the combined moon phase state. This makes the flavor introduced in 17.1 mechanically real.

**Core concept — continuous Fold pressure**:
Rather than only reacting to phase-transition events, introduce a queryable `GetFoldPressure() float64` (range 0.0–1.0) that is computed from the current phase of all three moons simultaneously. Each moon contributes via a cosine curve:

```
contribution = (1 - cos(2π × phasePercent)) / 2
  → 0.0 at new moon, 1.0 at full moon, 0.5 at quarters
```

Weighted sum (The Eye is the largest and slowest — it dominates):
```
FoldPressure = 0.20 × swiftContrib + 0.30 × wandererContrib + 0.50 × eyeContrib
```

This gives a smooth 0.0–1.0 range with meaningful variation. The function lives in `internal/gametime/` alongside the existing day/night helpers and is cheap to call any time.

**Effect 1 — Spell fold ceiling** (wires into Stage 11.2 Fold engine):
```
foldCeiling = 4 + int(FoldPressure × 4)   // range: 4 (all dark) to 8 (Eye full)
```
The Fold engine checks `gametime.GetFoldPressure()` when evaluating whether a cast attempt exceeds the caster's coherence limit. Failure message should be flavor, not a number: "The Fold will not hold that many folds tonight."

**Effect 2 — Mutation trigger probability**:
```
mutationMult = 0.5 + FoldPressure          // range: 0.5× (all dark) to 1.5× (Eye full)
```
Applied as a multiplier to the existing mutation chance roll in `CheckMutationTrigger()` or equivalent. High pressure nights make the body more susceptible to Becoming.

**Effect 3 — Aberrant combat stats** (optional, lower priority):
When spawning or engaging in combat in cave/dungeon biomes, Aberrant mobs check pressure and apply a scaled stat bonus:
```
aberrantBonus = int(FoldPressure × 10)    // 0–10 points on attack/defense
```
This is wired in via mob combat hooks or spawn scripts, not hardcoded per mob.

**New files/modifications** (~4 files, ~100 lines):
1. `internal/gametime/moonphase.go` (new) — `GetFoldPressure()` function; moon cycle constants; phase computation
2. `internal/scripting/` or relevant casting file — read `GetFoldPressure()` to cap fold ceiling during spell resolution
3. Mutation trigger location (TBD — wherever `RollMutation` / `CheckMutationTrigger` lives) — apply pressure multiplier
4. Optional: mob spawn/combat hook for Aberrant bonus

**Design notes**:
- `GetFoldPressure()` is pure math on round count — no state, no event dependency, always current
- The `MoonPhase` event from Stage 17.1 is *not* required for this to work — the pressure value is always computable. Stage 17.1 events are for flavor only; 17.2 reads the raw value.
- No player-visible numbers. Effects described in flavor terms only (see CLAUDE.md player-facing messages rule)
- Fold pressure is a good candidate for a `devtool` query so admins can check the current value without doing the math manually

**Testing**:
- `devtool pressure` (or similar) reports current value and each moon's contribution
- Advance time to Eye-full alignment, attempt a high-fold spell — should succeed where it failed at minimum pressure
- Advance time to all-new-moon state, verify mutation rate is observably lower over many rolls
- Verify Aberrant in Boss Cave is measurably harder on high-pressure nights

---

## Stage 18: Remove Numerical References — Immersive Descriptions ✅ COMPLETED

**Goal**: Audit every player-visible message surface and replace all raw numbers with qualitative descriptive language. Introduces a data-driven casting message system (YAML file) so future atmosphere can be added without code changes.

### Substage 18.1 — Scripting API + Spell JS Scripts ✅ COMPLETED
- Added `UtilGetDamageDescription` and `UtilGetHealDescription` to scripting API (`internal/scripting/util_func.go`)
- Updated `heal.js`, `healall.js`, `mm.js`, `sparks.js` to use qualitative descriptions instead of raw HP numbers
- Updated spell YAML descriptions (`heal.yaml`, `healall.yaml`, `mm.yaml`, `sparks.yaml`) to remove dice notation

### Substage 18.2 — Casting System Message Cleanup ✅ COMPLETED
- Created `_datafiles/world/dogmud/casting-messages.yaml` — varied atmospheric casting messages
- Created `internal/spells/casting_messages.go` — YAML loader + `GetCastMessage()` function
- Replaced fold counts, conviction numbers, dice rolls, and cooldown rounds in `internal/usercommands/skill.cast.go`

### Substage 18.3 — Special Move Cooldown Cleanup ✅ COMPLETED
- Removed round-count cooldown messages from `bash.go`, `kick.go`, `trip.go`, `grapple.go`, `submit.go`
- Removed DEBUG z-score block from `grapple.go`

### Substage 18.4 — Informational Command + Status Screen Cleanup ✅ COMPLETED
- Added 4 qualitative helpers to `internal/combat/descriptions.go`: `GetConvictionCostDescription`, `GetWaitRoundsDescription`, `GetCastCountDescription`, `GetSuccessChanceDescription`
- Added 5 template functions to `internal/templates/templatesfunctions.go`: `statQuality`, `vitalQuality`, `armorQuality`, `mutationLevel`, `durationQuality`
- Updated `internal/usercommands/spells.go` — qualitative cost/wait/familiarity/reliability columns
- Updated `internal/usercommands/status.go` — stat training feedback shows tier name not numbers
- Updated `status.template` — stats show tier words, HP/ST/CV show qualitative state, armor shows tier word, mutations show minor/moderate/major
- Updated `conditions.template` — buff duration shows qualitative phrase

**Deliberate numeric exemptions**: Gold and Bank on status screen; Lives counter (permadeath)

---

## Phase 19: LLM Integration & AI NPCs

> **Foundation now in place**: All core systems are stable, the dev tools JSON API
> provides a programmatic zone-building interface, and the balance config allows
> external tuning. LLM agents can now interact with a complete, well-structured game.

The stage structure and full design details for Phase 18 carry forward from the original Phase 11 plan, renumbered as Stages 18.1–18.4:

- **Stage 18.1**: GMCP Enhancement for Full State Coverage
- **Stage 18.2**: Rule-Based NPC Dialogue Framework
- **Stage 18.3**: Local LLM Integration (Optional — requires separate LLM service)
- **Stage 18.4**: LLM-as-Builder Pipeline (Offline Content Generation)

*(See original Stage 11.1–11.4 entries above for full design, file lists, and acceptance criteria.)*

---

### Stage 18.1: GMCP Enhancement for Full State Coverage ✅ COMPLETED
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

### Stage 18.2: Rule-Based NPC Dialogue Framework ✅ COMPLETED (a1cdc58)
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

### Stage 18.3: Local LLM Integration (Optional) ✅ COMPLETED (86c7e48) — post-completion fixes applied
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

### Stage 18.4: LLM-as-Builder Pipeline (Offline Content Generation) ✅ COMPLETED
**Goal**: Create a human-in-the-loop pipeline for using an LLM to assist with world building — generating rooms, mobs, items, and other data files in the correct YAML format. The LLM works offline as a content drafting tool; all output is reviewed by the designer before being dropped into the game. No engine changes are required.

**Design philosophy**:
- The LLM is a *drafting assistant*, not an autonomous builder. The designer is always the editor and gatekeeper.
- `world.md` is mandatory context for every generation prompt — it anchors tone, lore, and aesthetic.
- Existing data files serve as few-shot examples so the LLM produces correctly-structured output.
- The pipeline is a set of prompt templates and schema docs, not engine code.

**Deliverables**:
1. **Schema reference documents** — concise human- and LLM-readable descriptions of every data file format the LLM might generate:
   - `docs/schemas/room.md` — room YAML fields, terrain/biome values, exit format
   - `docs/schemas/mob.md` — mob YAML fields, species IDs, stat ranges, script hooks
   - `docs/schemas/item.md` — item YAML fields, slot names, damage types, rarity
   - `docs/schemas/spell.md` — spell YAML fields + matching `.js` stub contract
   - `docs/schemas/buff.md` — buff YAML fields, filename convention (`{id}-{name}.yaml`)
2. **Prompt templates** — reusable system + user prompt pairs for each content type:
   - `docs/prompts/new_room.md` — generates a room YAML + any connecting exit edits needed
   - `docs/prompts/new_mob.md` — generates a mob YAML + optional script stub
   - `docs/prompts/new_item.md` — generates an item YAML
   - `docs/prompts/zone_sketch.md` — given a zone concept, generates a room list + rough map before generating individual rooms
3. **Generation checklist** (`docs/CONTENT_GENERATION_GUIDE.md`) — step-by-step workflow:
   - Load `world.md` + relevant schema doc as system context
   - Provide 2-3 existing files as few-shot examples
   - Describe the desired content
   - Review LLM output against schema and world feel
   - Verify filename matches `ConvertForFilename()` convention
   - Check for stale instance saves if editing existing zones
   - Drop file, restart server, smoke test in-game

**What this stage explicitly does NOT do**:
- No engine code changes
- No live API calls from the MUD server
- No autonomous generation without human review

**Files to Create** (~8 documentation files, ~0 lines of Go):
1. `docs/schemas/room.md`
2. `docs/schemas/mob.md`
3. `docs/schemas/item.md`
4. `docs/schemas/spell.md`
5. `docs/schemas/buff.md`
6. `docs/prompts/new_room.md`
7. `docs/prompts/new_mob.md`
8. `docs/prompts/new_item.md`
9. `docs/prompts/zone_sketch.md`
10. `docs/CONTENT_GENERATION_GUIDE.md`

**Testing**:
- [ ] **Schema test**: Run each schema doc + `world.md` as context, ask LLM to generate a sample file, verify output is valid YAML that the engine accepts on startup
- [ ] **Prompt test**: Generate one room, one mob, one item end-to-end using the prompt templates; drop files into a test zone and verify no startup panic
- [ ] **Filename test**: Verify generated filenames match `ConvertForFilename()` output before every file drop

**Acceptance Criteria**:
- Each schema doc covers all required fields with valid value ranges
- Prompt templates reliably produce engine-loadable YAML in one or two iterations
- Generation guide covers the full workflow including instance-save gotchas
- A new zone room can be generated, reviewed, placed, and tested in under 15 minutes
- No engine startup panics from generated files

**Estimated Changes**: ~10 documentation files, minimal Go

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
| Phase 9: Combat Presentation | 8 stages (9.1–9.8) | ~40 hours | **9.1–9.8 Complete** |
| Phase 10: Skill System Cleanup | 2 stages (10.1–10.2) | 12 hours | 10.1–10.2 Complete |
| Phase 11: Magic Rework | 5 stages (11.1–11.5) | 30 hours | **11.1–11.5 Complete** |
| Phase 12: Mutations | 2 stages (12.1–12.2) | 16 hours | **12.1–12.2 Complete** |
| Phase 13: Basic Crafting | 2 stages (13.1–13.2) | 16 hours | **13.1–13.2 Complete** |
| Phase 14: Balance Config | 1 stage (14.1) | 8 hours | **14.1 Complete** |
| Phase 15: Dev Tools | 2 stages (15.1–15.2) | 12 hours | **15.1–15.2 Complete** |
| Phase 16: Tutorial Area | 2 stages (16.1–16.2) | 30 hours | **16.1–16.2 Complete** |
| Phase 17: LLM Integration | 4 stages (17.1–17.4) | 35 hours | **17.1–17.4 Complete** |
| Phase 18: Immersive Descriptions | 4 stages (18.1–18.4) | 24 hours | **18.1–18.4 Complete** |
| Phase 19: Hotfixes & Polish | 1 stage (19.1) | 4 hours | Not Started |
| Phase 20: Death Penalties | 1 stage (20.1) | 6 hours | Not Started |
| Phase 21: Autoscaling Removal | 1 stage (21.1) | 4 hours | Not Started |
| Phase 22: AI Connection Limits | 1 stage (22.1) | 6 hours | Not Started |
| Phase 23: Content — Major City & Road | 5 stages (23.1–23.5) | 40 hours | Not Started |
| Phase 24: Expanded Mutations | 2 stages (24.1–24.2) | 16 hours | Not Started |
| Phase 25: Expanded Spells | 2 stages (25.1–25.2) | 16 hours | Not Started |
| Phase 26: NPC Species Variety | 2 stages (26.1–26.2) | 12 hours | Not Started |
| Phase 27: Dialogue–Quest Integration | 2 stages (27.1–27.2) | 16 hours | Not Started |
| Phase 28: LLM Tutorial Enhancement | 1 stage (28.1) | 8 hours | Not Started |
| **Total** | **~77 stages** | **~533 hours** | |

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
- [ ] All 17 phases complete
- [ ] Core DOGMud mechanics functional
- [ ] Combat is descriptive, immersive, and balanced
- [ ] Magic system is distinctive and working
- [ ] Mutations, crafting, and progression feel cohesive
- [ ] Tutorial area teaches all core systems
- [ ] Dev tools enable AI-assisted zone building
- [ ] Ready for world expansion and content creation

---

## Phase 19: Post-Stage-18 Hotfixes & Polish

### Stage 19.1: Bug Fixes from Playtesting

**Goal**: Fix small bugs and polish issues discovered after Stage 18 completion.

**Bug 1 — `SkillSoftCap` compiler error in `progression_test.go`**:
The test file references `SkillSoftCap` as a bare package-level identifier, but it was moved to
`configs.Balance.SkillSoftCap` during the balance config refactor. The test needs a local constant
or needs to pull the value from the config.

**Files**: `internal/characters/progression_test.go`

**Bug 2 — Mutation visual text uses first-person pronouns**:
All 10 mutation YAML files have `visual:` fields written in second-person ("Your skin has taken on...",
"Your fingers end in..."). `GetMutationVisuals()` appends these to the character description template,
which is shown to *anyone* looking at the character — not just the character themselves. Fix all
`visual:` fields to use third-person phrasing ("Rough, leathery skin covers their body.").

**Files**: All `_datafiles/world/dogmud/mutations/*.yaml` (10 files)

**Bug 3 — Tutorial trainer dialogue fires before player attacks**:
In `sanctum_basin/114.js`, the trainer's "I may have understated..." comment fires on the first
idle tick after the dummy spawns, regardless of whether the player has actually engaged in combat.
Fix: add a check for dummy HP being lower than max (i.e., the player has actually hit it) before
triggering the encouraging dialogue.

**Files**: `_datafiles/world/dogmud/rooms/sanctum_basin/114.js`

**Bug 4 — Tutorial wilderness guide room — tracks/forage too subtle**:
The wilderness guide room (106) hints at foraging and tracking via idle messages, but it's too
easy to miss. Add clearer room description text and/or have the guide NPC proactively mention
tracking and foraging through a dialogue prompt or arrival message.

**Files**: `_datafiles/world/dogmud/rooms/sanctum_basin/106.yaml`

**Testing**:
- [ ] `go test ./internal/characters/...` passes (SkillSoftCap fix)
- [ ] Look at a mutated mob — description uses third-person ("Their skin..." not "Your skin...")
- [ ] Enter tutorial combat room without attacking — trainer does NOT comment prematurely
- [ ] Enter wilderness guide room — tracks/foraging are mentioned prominently

---

## Phase 20: Death Penalty Tuning

### Stage 20.1: Meaningful Death Consequences for a Level-Free World

**Goal**: Tune the existing death penalty system for DOGMud's level-free, use-based progression.
The engine already has death penalty infrastructure (`suicide.go`, `config.gameplay.go`) but the
XP penalty mode is meaningless since levels/XP were removed. Design penalties that create tension
without being punishing enough to drive players away.

**Changes**:
1. **Remove XP-based penalty mode** — `XPPenalty: "level"` and percentage modes reference a system
   that no longer exists. Remove or replace with stat/skill-based consequences.
2. **Stat decay on death** — on death, reduce 1–2 random stats by a small amount (e.g., 1–3 points
   raw value). This fits the use-based model: you lose a bit of hard-earned progress.
3. **Skill rust on death** — reduce 1–2 random skills by a small amount of virtual ranks.
   Skills used more recently are less likely to decay (use recency weighting).
4. **Respawn debuff** — apply a temporary "Death's Shadow" debuff on respawn that reduces all stats
   by a percentage for N rounds. Creates a recovery period.
5. **Equipment drop tuning** — `EquipmentDropChance` already exists. Set a reasonable default
   (e.g., 10–20%) so death has tangible item risk.
6. **Configurable via balance config** — all death penalty values should be tunable in
   `_datafiles/config.yaml` under `GamePlay.DeathPenalty`.

**Files to Modify** (~8 files, ~300 lines):
1. `internal/configs/config.gameplay.go` — add/update death penalty config fields
2. `internal/usercommands/suicide.go` — implement stat/skill decay logic
3. `internal/hooks/` — death hook for applying respawn debuff
4. `_datafiles/config.yaml` — default penalty values
5. `_datafiles/world/dogmud/buffs/` — new "Death's Shadow" debuff YAML

**Testing**:
- [ ] Die and respawn — verify stat decay message appears
- [ ] Die and respawn — verify skill rust message appears
- [ ] Die and respawn — verify Death's Shadow debuff applies
- [ ] Verify equipment drop chance works at configured rate
- [ ] Verify all penalty values are configurable and validate correctly

---

## Phase 21: Remove Autoscaling

### Stage 21.1: Remove Zone Autoscaling System

**Goal**: Remove the zone-level mob autoscaling system entirely. Mob difficulty should be set
explicitly per-mob via `statpool` in the mob YAML or per-spawn via `statpool`/`statpoolmod` in
room spawn info. The zone-wide `autoscale` config creates unpredictable difficulty variance
that's hard to balance and obscures the intended difficulty curve.

**Changes**:
1. **Remove `MobAutoScale` from `ZoneConfig`** — delete the struct field and `GenerateRandomStatPool()`
2. **Remove autoscale fallback in room spawn logic** — in `rooms.go`, the spawn code checks
   `zConfig.MobAutoScale.Minimum > 0` and overrides mob stat pools. Remove this fallback.
3. **Audit all zone configs** — check each `zone-config.yaml` for `autoscale:` entries and remove them.
   Ensure every mob that relied on autoscaling has an explicit `statpool` in its mob YAML.
4. **Update documentation** — remove autoscale references from schemas and content generation docs.

**Files to Modify** (~6 files, ~80 lines):
1. `internal/rooms/zoneconfig.go` — remove `MobAutoScale` struct and `GenerateRandomStatPool()`
2. `internal/rooms/rooms.go` — remove autoscale fallback in spawn logic (~lines 601–621)
3. `_datafiles/world/dogmud/rooms/*/zone-config.yaml` — remove `autoscale:` entries
4. `docs/schemas/` — update zone config schema if documented

**Testing**:
- [ ] Server starts without errors after removing autoscale configs
- [ ] Mobs spawn with their YAML-defined `statpool` values
- [ ] No zone config YAML contains `autoscale:` entries
- [ ] `go test ./internal/rooms/...` passes

---

## Phase 22: AI vs Human Connection Limits

### Stage 22.1: Separate Connection Pools for AI and Human Players

**Goal**: Prevent AI player connections from consuming slots meant for human players. Currently,
`MaxTelnetConnections` in `main.go` applies a single cap to all connections. Split this into
separate pools so AI testing/play can scale independently.

**Changes**:
1. **New config fields** — add `MaxHumanConnections` and `MaxAIConnections` to the server config.
   Keep `MaxTelnetConnections` as a hard global ceiling.
2. **Identify AI connections** — AI players connect via the LLM integration pipeline. Tag these
   connections at handshake time (e.g., via a login flag or a dedicated AI login command).
3. **Enforce separate limits** — in the connection acceptance loop (`main.go`), check the connection
   type against its respective pool limit before accepting.
4. **Status reporting** — update admin `who` or server status to show human vs AI connection counts.

**Files to Modify** (~5 files, ~150 lines):
1. `main.go` — split connection limit check
2. `internal/configs/` — add new config fields
3. `_datafiles/config.yaml` — default values
4. `internal/connections/` — tag connection type

**Testing**:
- [ ] Human connections are accepted up to `MaxHumanConnections`
- [ ] AI connections are accepted up to `MaxAIConnections`
- [ ] Neither pool can exceed the global `MaxTelnetConnections` ceiling
- [ ] Admin `who` shows connection type breakdown

---

## Phase 23: Content — Major City & Road from School

### Stage 23.1: Zone Sketch — The Road to Thornwall

**Goal**: Design the overland route from Sanctum Basin (tutorial area) to the first major city,
Thornwall. This is a linear-ish path with 2–3 small zones along the way, each with distinct
biomes and level-appropriate encounters for newly graduated characters.

**Changes**:
1. Use `/zone-sketch` to plan:
   - **Dustwalk Road** — 8–12 rooms, open grassland/scrubland connecting the school to the city.
     Encounters: bandits, wild dogs, scavenger birds. Foraging opportunities (herbs, berries).
   - **Watcher's Crossing** — 5–8 rooms, a small waystation/bridge area. Friendly NPC merchants,
     a small inn for resting. Minor quest hooks (missing caravan, bridge troll).
   - **Thornwall Outskirts** — 6–10 rooms, farmland and outskirts approaching the city walls.
     Encounters: livestock, farm pests, the occasional highwayman.
2. Document room adjacency maps and mob/item lists for each zone.

**Deliverables**: Zone sketch documents with room lists, adjacency, mob rosters, item drops.

### Stage 23.2: Build Dustwalk Road Zone

**Goal**: Generate all rooms, mobs, and items for the Dustwalk Road zone using `/new-room`,
`/new-mob`, and `/new-item` commands.

**Changes**:
1. Create zone folder `_datafiles/world/dogmud/rooms/dustwalk_road/`
2. Create zone-config.yaml (no autoscale — explicit stat pools per mob)
3. Generate 8–12 room YAML files with descriptions, exits, idle messages
4. Generate 3–5 mob YAML files (wild dogs, scavenger birds, bandits)
5. Generate 2–3 item YAML files (roadside loot, bandit drops)
6. Wire exits between rooms and connect to Sanctum Basin exit

### Stage 23.3: Build Watcher's Crossing Zone

**Goal**: Generate the waystation zone — a rest stop with merchants, an inn, and minor quest hooks.

**Changes**:
1. Create zone folder and zone-config
2. Generate 5–8 rooms (bridge, inn, merchant stall, crossroads)
3. Generate 2–3 friendly NPCs (innkeeper, merchant, guard) with dialogue trees
4. Generate 1–2 hostile mobs for optional encounters (bridge troll, river creature)
5. Wire exits to Dustwalk Road

### Stage 23.4: Build Thornwall Outskirts Zone

**Goal**: Generate the farmland approach to Thornwall.

**Changes**:
1. Create zone folder and zone-config
2. Generate 6–10 rooms (farmhouses, fields, dirt roads, city gate approach)
3. Generate 3–4 mobs (farm pests, livestock, highwayman)
4. Wire exits to Watcher's Crossing and Thornwall city entrance

### Stage 23.5: Build Thornwall City (Core)

**Goal**: Generate the core of Thornwall — a walled city with essential services and exploration.
This is a major content milestone. Focus on the city skeleton; additional city content can be
added incrementally.

**Changes**:
1. Create zone folder `_datafiles/world/dogmud/rooms/thornwall/`
2. Generate 15–25 rooms:
   - City gate, main street, market square
   - Tavern/inn, blacksmith, apothecary, temple
   - Guard barracks, city hall, residential streets
   - Back alleys, sewer entrance (future dungeon hook)
3. Generate 8–12 NPCs:
   - Merchants (blacksmith, apothecary, general goods)
   - Quest-givers (guard captain, temple priest, tavern keeper)
   - Ambient NPCs (townsfolk, beggars, street performers)
4. Generate city-appropriate items (city goods, quest items)
5. Wire all exits and connect to Thornwall Outskirts

**Testing** (all of Phase 23):
- [ ] Server starts with all new zones loaded
- [ ] Player can walk from Sanctum Basin → Dustwalk Road → Watcher's Crossing → Thornwall Outskirts → Thornwall
- [ ] All mobs spawn and are fightable
- [ ] All merchants are functional
- [ ] No broken exits or missing room references
- [ ] Mini-map renders correctly for all new zones

---

## Phase 24: Expanded Mutations

### Stage 24.1: New Mutations & Conflict Resolution

**Goal**: Expand the mutation pool from 10 to 25+ and implement a conflict resolution system
for diametrically opposed mutations (e.g., Dense Muscles vs Hollow Bones).

**Changes**:
1. **Mutation conflict system** — add a `conflicts` field to mutation YAML specs. When a character
   would gain a mutation that conflicts with an existing one, either block it or replace the
   existing mutation (configurable). Display a message explaining the conflict.
2. **New mutations** (15+ new entries):
   - Physical: Elongated Limbs, Thick Hide, Venomous Bite, Regenerative Tissue, Stone Skin
   - Sensory: Echolocation, Thermal Vision, Tremorsense, Heightened Smell
   - Mental: Hivemind Echo, Precognitive Flashes, Psychic Resistance
   - Metabolic: Cold-Blooded, Photosynthetic Skin, Rapid Metabolism
3. **Fix visual text** — ensure all new mutations use third-person `visual:` fields
   (this may already be fixed in Stage 19.1)

**Files to Modify** (~20 files, ~500 lines):
1. `internal/mutations/mutations.go` — add conflict checking logic
2. `internal/characters/character.go` — call conflict check on mutation gain
3. `_datafiles/world/dogmud/mutations/` — 15+ new YAML files, update existing 10 with `conflicts:`
4. Test files

### Stage 24.2: NPC Mutations

**Goal**: Allow NPCs/mobs to spawn with mutations, making encounters more varied and surprising.

**Changes**:
1. **Mob mutation field** — add `mutations:` list to mob YAML spec. Mobs spawn with these
   mutations active, gaining their stat bonuses and visual descriptors.
2. **Random mutation chance** — optionally add a `mutationchance:` field to mob YAML. On spawn,
   the mob has a percentage chance to gain 1 random mutation from a configured pool.
3. **Visual integration** — mutated mobs show mutation visuals in their description
   (already handled by `GetMutationVisuals()` on Character).

**Files to Modify** (~5 files, ~150 lines):
1. `internal/mobs/mobs.go` — apply mutations on spawn
2. `_datafiles/world/dogmud/mobs/` — add mutations to select mob YAML files
3. Mob YAML schema docs

**Testing** (all of Phase 24):
- [ ] Conflicting mutations are blocked/replaced with a message
- [ ] All 25+ mutations load without errors
- [ ] NPCs spawn with configured mutations
- [ ] Random mutation chance works correctly
- [ ] Mutated mob descriptions show mutation visuals

---

## Phase 25: Expanded Spells

### Stage 25.1: New Elemental & Enhancement Spells

**Goal**: Expand the spell library from 14 to 25+. Focus on Elemental and Enhancement schools
first, adding offensive variety and utility.

**Changes**:
1. **Elemental spells** (4–5 new):
   - Ice Shard — single-target cold damage, chance to slow
   - Chain Lightning — hits primary target + 1–2 nearby enemies (reduced damage)
   - Earth Tremor — area effect, chance to knock prone
   - Flame Wall — room-area damage over time
2. **Enhancement spells** (4–5 new):
   - Iron Skin — temporary natural armor boost (self)
   - Haste — temporary Dexterity boost (self/other)
   - Bull's Strength — temporary Strength boost
   - Cat's Grace — temporary dodge chance boost
   - Fortify — temporary Vitality boost
3. Each spell needs a `.yaml` definition and `.js` script stub.
4. Balance conviction costs against existing spells.

**Files to Create**: 8–10 new spell YAML + JS pairs in `_datafiles/world/dogmud/spells/`

### Stage 25.2: New Mental & Vital Spells

**Goal**: Add Mental and Vital school spells for crowd control, debuffs, and healing variety.

**Changes**:
1. **Mental spells** (3–4 new):
   - Confuse — target attacks randomly (friend or foe) for N rounds
   - Fear — target flees the room (opposed Willpower check)
   - Mind Spike — mental damage, bypasses physical armor
   - Dominate — short-duration charm (very high conviction cost)
2. **Vital spells** (3–4 new):
   - Regenerate — heal-over-time buff
   - Purify — remove poison and disease conditions
   - Life Drain — damage target, heal self for portion of damage
   - Mass Heal — heal all friendly characters in room (reduced per-target)
3. Each spell needs `.yaml` + `.js` files.
4. Ensure conviction costs scale appropriately.

**Testing** (all of Phase 25):
- [ ] All new spells load without errors
- [ ] Each spell can be cast and produces expected effects
- [ ] Conviction costs are balanced (no free infinite casting)
- [ ] Buff/debuff durations and magnitudes feel right
- [ ] Spell descriptions use immersive language (no raw numbers)

---

## Phase 26: NPC Species Variety

### Stage 26.1: Animal & Creature Species

**Goal**: Add NPC-only species for animals and creatures to populate the world with diverse
wildlife. Currently only 6 species exist (human, ghostly spirit, rodent, dummy, bat, orb).
These are NOT playable — they're for mob variety.

**Changes**:
1. **New animal species** (8–10):
   - Wolf, Bear, Boar, Deer, Snake, Eagle/Hawk, Spider, Fish
   - Each with appropriate stat modifiers (e.g., wolves: high Dexterity/Perception,
     bears: high Strength/Vitality)
   - Size categories affecting combat (small creatures are harder to hit, large ones hit harder)
2. **Creature species** (3–5):
   - Troll (regeneration trait), Goblin (pack tactics), Elemental (magic resistance),
     Undead (poison immunity), Insectoid (natural armor)
3. Each species needs a YAML file in `_datafiles/world/dogmud/species/`

### Stage 26.2: Species Traits & Combat Integration

**Goal**: Make species matter mechanically — species-specific traits that affect combat,
movement, and interaction.

**Changes**:
1. **Natural weapons** — some species have built-in attacks (wolf bite, bear claw, snake venom).
   These override unarmed damage with species-appropriate values.
2. **Natural armor** — species like trolls, insectoids get innate damage reduction.
3. **Movement traits** — flying species can access aerial rooms, aquatic species can swim.
4. **Resistances/vulnerabilities** — undead resist poison but are vulnerable to fire, etc.

**Files to Modify**:
1. `_datafiles/world/dogmud/species/` — 11–15 new YAML files
2. `internal/species/` — trait application logic
3. `internal/combat/` — species trait integration in damage/defense calculations

**Testing** (all of Phase 26):
- [ ] All new species load without errors
- [ ] Mobs using new species display correct descriptions
- [ ] Natural weapons/armor apply correctly in combat
- [ ] Species traits (resistances, vulnerabilities) work as expected

---

## Phase 27: Dialogue–Quest Integration

### Stage 27.1: Quest-Gated Dialogue Options

**Goal**: Allow dialogue tree options to be conditionally shown/hidden based on quest state.
Currently the dialogue tree system and quest system operate independently. This stage connects
them so NPCs can offer different dialogue based on what quests the player has active, completed,
or not yet started.

**Changes**:
1. **Quest condition in dialogue nodes** — add `questRequired` and `questExcluded` fields to
   dialogue option YAML. Options only appear if the player has (or doesn't have) the specified
   quest flag.
2. **Quest state checks** — when building the dialogue option list for a player, filter options
   based on the player's quest flags.
3. **Quest grant from dialogue** — add a `grantsQuest` field to dialogue options. Selecting the
   option gives the player a quest flag.
4. **Quest completion dialogue** — add `requiresItem` field. If the player has the required item,
   the option appears and selecting it consumes the item and advances the quest.

**Files to Modify** (~6 files, ~200 lines):
1. `internal/dialogues/` — add quest condition fields to dialogue structs
2. `internal/hooks/` — filter dialogue options by quest state
3. `_datafiles/world/dogmud/dialogues/` — update existing NPC dialogues with quest hooks
4. Test files

### Stage 27.2: NPC Memory & Quest State Responses

**Goal**: NPCs remember player quest progress and respond contextually. A quest-giver who sent
you on a mission should acknowledge your progress when you return.

**Changes**:
1. **Greeting variation** — NPC greetings change based on active quest state:
   - No quest: default greeting
   - Quest active: "Have you found the missing caravan yet?"
   - Quest complete: "You've done well. Here is your reward."
2. **LLM context injection** — when the LLM dialogue system is active, inject quest state into
   the NPC's context so LLM-generated responses are quest-aware.
3. **Reward delivery** — NPCs can give items, gold, or stat bonuses as quest rewards through
   dialogue completion nodes.

**Files to Modify** (~5 files, ~200 lines):
1. `internal/dialogues/` — greeting variation logic
2. `internal/llm/` — quest state context injection
3. `_datafiles/world/dogmud/dialogues/` — example quest dialogues

**Testing** (all of Phase 27):
- [ ] Dialogue options appear/hide correctly based on quest flags
- [ ] Selecting a quest-grant dialogue option gives the quest flag
- [ ] NPC greetings change based on quest state
- [ ] LLM responses reference quest context when available
- [ ] Quest rewards are delivered through dialogue

---

## Phase 28: LLM Tutorial Enhancement

### Stage 28.1: Dynamic Tutorial NPC Responses

**Goal**: Enhance tutorial area mobs with deeper LLM integration so they respond dynamically
to player actions, offer contextual hints, and create a more engaging onboarding experience.

**Changes**:
1. **Contextual hint system** — tutorial NPCs detect what the player has and hasn't done
   (attacked dummy, used skills, tried foraging, etc.) and offer LLM-generated hints for
   things the player hasn't tried yet.
2. **Adaptive difficulty coaching** — if a player is struggling (dying repeatedly, low HP),
   tutorial NPCs offer encouragement and tactical advice via LLM.
3. **Personality depth** — give each tutorial NPC a richer personality prompt so LLM responses
   feel distinct (the gruff trainer vs the patient wilderness guide vs the mysterious mage).
4. **Conversation memory** — tutorial NPCs remember what they've already told the player
   within a session, avoiding repetitive advice.

**Files to Modify** (~8 files, ~300 lines):
1. `_datafiles/world/dogmud/rooms/sanctum_basin/*.js` — enhanced tutorial room scripts
2. `_datafiles/world/dogmud/mobs/sanctum_basin/*.yaml` — richer NPC personality prompts
3. `internal/llm/` — conversation memory for tutorial context
4. `internal/hooks/` — player action tracking for hint system

**Testing**:
- [ ] Tutorial NPCs respond differently based on player progress
- [ ] Struggling players receive helpful hints
- [ ] NPCs don't repeat the same advice within a session
- [ ] Each tutorial NPC has a distinct personality in LLM responses
- [ ] Tutorial flow feels natural and engaging

---

## Future Expansion (Not Yet Scheduled)

These are longer-term goals to be detailed when the above phases are complete:

1. **Economy depth** — markets, supply/demand, trade routes, player economy
2. **Extended crafting** — additional skills (tailoring, woodworking), hundreds of recipes
3. **Faction & reputation system** — NPC factions that remember player actions
4. **Quest system expansion** — multi-stage quests with choices and consequences
5. **Additional world zones** — more cities, dungeons, wilderness areas
6. **PvP systems** — arenas, dueling, faction warfare

---

## Issue Traceability

Issues discovered during 2026-02-12 playtest session, mapped to stages:

| # | Issue | Stage(s) |
|---|-------|----------|
| 1 | Stats other than Vitality don't increase | 4.5 |
| 2 | Attack count formula needs rework | 5.3 |
| 3 | Skill soft cap not working; need per-skill multipliers | 4.6 |
| 4 | Unarmed damage doesn't scale; needs grappling | 7.3, 8.1, 8.2 |
| 5 | Combat takes too long | 14.1 (balance config knobs) |
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

**Last Updated**: 2026-02-23
**Status**: In Progress
**Current Stage**: Phases 1–18 complete. Next: Stage 19.1 (post-stage-18 hotfixes — SkillSoftCap test fix, mutation visual text, tutorial dialogue timing, wilderness guide room polish), then Phase 20 (death penalty tuning) through Phase 28 (LLM tutorial enhancement).
