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

## Phase 1: Foundation - Stat System Refactor

### Stage 1.1: Rename Stats (Minimal Risk)
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

### Stage 1.2: Add Secondary Stat Pools
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

### Stage 1.3: Remove Mana, Rename Health
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

## Phase 2: Species System Refactor

### Stage 2.1: Rename Race to Species
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

### Stage 2.2: Create "Human" Default Species
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

### Stage 3.1: Add Skill-Based Progression Hooks (Preparation)
**Goal**: Create the infrastructure for skill-use progression without removing levels yet.

**Changes**:
1. Add `SkillProgression` tracking to Character:
   - `SkillUseCount map[string]int` - Tracks skill usage
   - `StatUseCount map[string]int` - Tracks stat checks
2. Add progression trigger functions (don't activate yet):
   - `OnCriticalSuccess(skill string)`
   - `OnCriticalFailure(skill string)`
   - `OnSkillUse(skill string)`
3. Hook into existing combat/skill use (passive logging only)
4. Add config for progression rates (disabled by default)

**Files to Modify** (~10 files, ~300 lines):
1. `internal/characters/character.go` - Add new fields/methods
2. `internal/configs/config.gameplay.go` - Add progression config
3. `internal/combat/combat.go` - Add hooks for crit success/fail
4. `internal/skills/skills.go` - Add usage tracking hooks
5. Test files

**Testing**:
- [ ] **Unit Tests**: Test usage tracking increments correctly
- [ ] **Manual Test**: Use skills, verify usage counts increment
- [ ] **Manual Test**: Combat with crits, verify tracking works
- [ ] **Manual Test**: Verify existing level system still works
- [ ] **Data Test**: Check character save files include new fields

**Acceptance Criteria**:
- Usage tracking works
- No impact on existing level system
- New fields save/load correctly
- All tests pass

**Estimated Changes**: ~300 lines, 10 files

---

### Stage 3.2: Implement Skill Progression Formula
**Goal**: Create the actual progression logic (still not activated).

**Changes**:
1. Implement progression chance calculation:
   - Exponential difficulty curve
   - Easier below soft cap (50 for skills, 100 for stats)
   - Very hard above soft cap
2. Implement progression roll on triggers:
   - Critical success/failure
   - Low health/stamina/conviction events
   - Notable deeds (quest completion, first kill of mob type)
3. Add progression notifications (messages only, no actual stat changes)
4. Add config flag `UseSkillProgression bool` (default: false)

**Files to Modify** (~12 files, ~500 lines):
1. `internal/characters/progression.go` - New file with progression logic
2. `internal/characters/character.go` - Integrate progression checks
3. `internal/combat/combat.go` - Trigger on combat events
4. `internal/configs/config.gameplay.go` - Add enable flag
5. Test files with progression formula tests

**Testing**:
- [ ] **Unit Tests**: Test progression chance calculations
- [ ] **Unit Tests**: Test exponential curve (easy→hard)
- [ ] **Manual Test**: Enable progression, verify messages appear
- [ ] **Manual Test**: Verify no actual stat/skill changes (messages only)
- [ ] **Math Test**: Verify curve matches design (50% chance at low levels, 5% at high)

**Acceptance Criteria**:
- Progression formula implemented
- Messages show when progression would occur
- No actual changes to stats/skills yet
- Config flag controls system
- All tests pass

**Estimated Changes**: ~500 lines, 12 files

---

### Stage 3.3: Dual Progression Mode (Levels + Skills)
**Goal**: Run both systems in parallel - existing level system + new skill progression (opt-in via config).

**Changes**:
1. Enable actual stat/skill increases from progression system
2. Keep level system active
3. Add config `DualProgressionMode bool` (default: true)
4. Characters can:
   - Gain levels (traditional)
   - Gain skill ranks from use (new)
   - Gain stat points from use (new)
5. Both systems work, neither interferes

**Files to Modify** (~15 files, ~400 lines):
1. `internal/characters/progression.go` - Enable actual increases
2. `internal/characters/character.go` - Manage both systems
3. `internal/usercommands/score.go` - Display both level and skill ranks
4. `internal/configs/config.gameplay.go` - Add dual mode flag
5. Test files for both systems

**Testing**:
- [ ] **Unit Tests**: Test skill increases work
- [ ] **Unit Tests**: Test stat increases work
- [ ] **Manual Test**: Level up (old system), verify it works
- [ ] **Manual Test**: Use skills, verify they increase (new system)
- [ ] **Manual Test**: Both systems visible in score
- [ ] **Integration Test**: Character gains levels AND skill ranks
- [ ] **Balance Test**: Verify progression rates feel appropriate

**Acceptance Criteria**:
- Both systems functional
- No conflicts between systems
- Config flag controls dual mode
- Score displays both level and skill ranks
- All tests pass

**Estimated Changes**: ~400 lines, 15 files

---

### Stage 3.4: Decouple Combat from Levels
**Goal**: Make combat use skill ranks instead of levels for calculations.

**Changes**:
1. Update combat calculations to use skill ranks:
   - Attack bonus from `Melee Combat` skill instead of level
   - Defense bonus from `Defense/Evasion` skill instead of level
   - Damage from Strength + weapon + skill
2. Keep level visible but non-functional in combat
3. Add backwards compatibility for NPCs without skills (use level * 5 as skill)

**Files to Modify** (~10 files, ~600 lines):
1. `internal/combat/calculations.go` - Major refactor of combat math
2. `internal/combat/combat.go` - Update attack functions
3. `internal/mobs/mobs.go` - Add skill defaults for NPCs
4. `internal/characters/character.go` - Add helper methods
5. Combat test files

**Testing**:
- [ ] **Unit Tests**: Test new combat calculations
- [ ] **Manual Test**: Combat at level 1 with high skill vs low skill
- [ ] **Manual Test**: Combat with NPCs (verify backward compat)
- [ ] **Manual Test**: Verify damage formulas feel balanced
- [ ] **Balance Test**: Compare old vs new combat results
- [ ] **Regression Test**: Ensure combat is still functional

**Acceptance Criteria**:
- Combat uses skills, not levels
- NPCs without skills still functional
- Combat balance feels appropriate
- All tests pass
- No combat bugs

**Estimated Changes**: ~600 lines, 10 files

---

### Stage 3.5: Remove Level from Progression (Skills Only)
**Goal**: Stop awarding levels. Progression is 100% skill-based now.

**Changes**:
1. Set `DualProgressionMode: false` as default
2. Remove level-up on XP gain
3. Remove XP system entirely (or keep for legacy NPCs)
4. Remove training point system (tied to levels)
5. Hide level from UI (or show as decorative only)
6. Convert NPC levels to skill equivalents

**Files to Modify** (~20 files, ~700 lines):
1. `internal/characters/character.go` - Remove level-up logic
2. `internal/usercommands/experience.go` - Remove or deprecate
3. `internal/usercommands/train.go` - Remove or refactor for skill-based training
4. `internal/usercommands/score.go` - Remove level display
5. `internal/mobs/mobs.go` - Convert levels to skills
6. All NPC YAML files - Set default skills based on old levels
7. Test files

**Testing**:
- [ ] **Unit Tests**: Verify level-up doesn't happen
- [ ] **Unit Tests**: Verify XP doesn't accumulate
- [ ] **Manual Test**: Play for extended period, verify no levels gained
- [ ] **Manual Test**: Skills increase from use
- [ ] **Manual Test**: Combat, crafting, all systems use skills
- [ ] **Integration Test**: Full character progression lifecycle (create → skill up → combat)
- [ ] **Migration Test**: Load old characters, verify level→skill conversion

**Acceptance Criteria**:
- No levels gained
- Skill progression is only progression
- All systems functional without levels
- Old characters convert successfully
- All tests pass

**Estimated Changes**: ~700 lines, 20 files

---

## Phase 4: Distribution-Based Combat

### Stage 4.1: Add Distribution Rolling (Parallel System)
**Goal**: Implement DOG's distribution-based rolling without removing existing dice rolls.

**Changes**:
1. Create new `internal/rolling/distribution.go` module:
   - `RollDistribution(mean, stdDevPercent float64) float64`
   - `IsCritSuccess(roll, mean, stdDev float64) bool`
   - `IsCritFailure(roll, mean, stdDev float64) bool`
2. Add config flag `UseDistributionRolling bool` (default: false)
3. Create parallel combat functions using distribution rolling
4. Don't modify existing dice-based combat yet

**Files to Modify** (~8 files, ~300 lines):
1. `internal/rolling/distribution.go` - New file
2. `internal/combat/combat_distribution.go` - New file (parallel combat)
3. `internal/configs/config.gameplay.go` - Add config flag
4. Test files for distribution rolling

**Testing**:
- [ ] **Unit Tests**: Test distribution rolling produces correct mean/stddev
- [ ] **Unit Tests**: Test critical detection (2 std devs)
- [ ] **Statistical Test**: Generate 10,000 rolls, verify distribution shape
- [ ] **Manual Test**: Enable flag, test combat with distribution
- [ ] **Manual Test**: Disable flag, verify dice combat still works

**Acceptance Criteria**:
- Distribution rolling implemented
- Statistical properties verified
- Critical success/failure detection works
- Config flag controls which system is used
- All tests pass

**Estimated Changes**: ~300 lines, 8 files

---

### Stage 4.2: Replace Dice Combat with Distribution Combat
**Goal**: Completely replace dice-based combat with distribution-based combat.

**Changes**:
1. Replace all combat calculations:
   - Attack roll: `mean = Dexterity + Melee Combat skill, stddev = 15% of mean`
   - Defense roll: `mean = Dexterity + Defense skill, stddev = 15% of mean`
   - Compare rolls (higher wins)
   - Critical on ±2 stddev
2. Update damage calculation to use distributions
3. Remove all dice roll references from combat
4. Set `UseDistributionRolling: true` as default

**Files to Modify** (~15 files, ~800 lines):
1. `internal/combat/combat.go` - Replace attack functions
2. `internal/combat/calculations.go` - Replace all formulas
3. `internal/items/itemspec.go` - Remove dice notation from weapons (use multipliers)
4. All weapon YAML files - Convert dice to damage multipliers
5. Test files

**Testing**:
- [ ] **Unit Tests**: Test all new combat calculations
- [ ] **Unit Tests**: Test critical hit/miss detection
- [ ] **Manual Test**: Extensive combat testing (easy/medium/hard enemies)
- [ ] **Manual Test**: Verify damage feels balanced
- [ ] **Balance Test**: Compare statistical expected damage vs old system
- [ ] **Regression Test**: Ensure no combat bugs

**Acceptance Criteria**:
- All combat uses distribution rolling
- No dice notation remains
- Combat balance feels appropriate
- Crits happen at expected rate (~5%)
- All tests pass

**Estimated Changes**: ~800 lines, 15 files

---

## Phase 5: Movement & Stamina System

### Stage 5.1: Connect Stamina to Movement
**Goal**: Make movement consume stamina.

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
- [ ] **Unit Tests**: Test stamina cost calculations
- [ ] **Manual Test**: Move around, verify stamina decreases
- [ ] **Manual Test**: Move while heavily encumbered, verify higher cost
- [ ] **Manual Test**: Try to move with 0 stamina, verify blocked
- [ ] **Manual Test**: Rest, verify stamina regenerates
- [ ] **Balance Test**: Verify stamina costs feel appropriate

**Acceptance Criteria**:
- Movement costs stamina
- Cannot move with insufficient stamina
- Costs scale with terrain and encumbrance
- Regeneration works correctly
- All tests pass

**Estimated Changes**: ~400 lines, 12 files

---

### Stage 5.2: Stamina in Combat
**Goal**: Make combat actions consume stamina.

**Changes**:
1. Each attack costs stamina (scaled by weapon weight/speed)
2. Blocking/dodging costs stamina
3. Fleeing costs stamina
4. Out-of-stamina penalty in combat (can't attack/defend well)
5. Stamina regenerates slowly during combat

**Files to Modify** (~10 files, ~500 lines):
1. `internal/combat/combat.go` - Add stamina costs to attacks
2. `internal/combat/calculations.go` - Add out-of-stamina penalties
3. `internal/characters/character.go` - Add stamina calculation methods
4. `internal/hooks/NewRound_DoCombat.go` - Update combat round logic
5. Test files

**Testing**:
- [ ] **Unit Tests**: Test stamina costs for attacks
- [ ] **Manual Test**: Long combat, verify stamina decreases
- [ ] **Manual Test**: Run out of stamina in combat, verify penalties
- [ ] **Manual Test**: Flee, verify stamina cost
- [ ] **Balance Test**: Verify combat stamina drain feels appropriate

**Acceptance Criteria**:
- Combat consumes stamina
- Out-of-stamina penalties apply
- Stamina costs feel balanced
- All tests pass

**Estimated Changes**: ~500 lines, 10 files

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
Each phase (1-6) must include:
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
- **Stage 3.4**: Decouple Combat from Levels (combat refactor)
- **Stage 3.5**: Remove Level System (major breaking change)
- **Stage 4.2**: Replace Dice with Distribution (combat refactor)

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

| Phase | Stages | Estimated Hours |
|-------|--------|-----------------|
| Phase 1: Stats | 3 stages | 12 hours |
| Phase 2: Species | 2 stages | 8 hours |
| Phase 3: Remove Levels | 5 stages | 20 hours |
| Phase 4: Distribution Combat | 2 stages | 8 hours |
| Phase 5: Stamina | 2 stages | 8 hours |
| Phase 6: Conviction | 2 stages | 8 hours |
| **Total** | **16 stages** | **64 hours** |

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
- [ ] All 6 phases complete
- [ ] Core DOGMud mechanics functional
- [ ] Migration path from GoMud verified
- [ ] Ready for world building and content creation

---

## Next Steps After Core Mechanics

Once core mechanics (Phases 1-6) are complete:
1. **Phase 7**: Tutorial Area (Sanctum Basin)
2. **Phase 8**: Mutation System
3. **Phase 9**: Economy & Crafting
4. **Phase 10**: World Building (Cities, NPCs)

These phases will be detailed in a separate plan once core mechanics are stable.

---

## Notes

- Each stage is designed to be completable in 1-2 work sessions
- Stages build on each other - complete in order
- Don't skip testing stages (critical for stability)
- If a stage exceeds scope (1000 lines/50 files), break it into sub-stages
- Prioritize keeping the MUD playable at each stage
- Document any deviations from the plan in commit messages

---

**Last Updated**: 2026-02-05
**Status**: Ready to Begin
**Current Stage**: Not Started
