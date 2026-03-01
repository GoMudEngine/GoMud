# DOGMud Test Coverage Audit — Stage 40.1

Generated: 2026-02-28
Baseline: `go test -coverprofile=coverage.out ./internal/...`

---

## Section A: Coverage Baseline Table

Sorted by tier, then coverage ascending (worst gaps first).

| Package | Coverage | Tests? | Tier |
|---------|----------|--------|------|
| hooks | 0.0% | No | 1 |
| items | 0.0% | No | 1 |
| mobs | 0.0% | No | 1 |
| spells | 0.0% | No | 1 |
| combat | 3.3% | Yes | 1 |
| characters | 27.3% | Yes | 1 |
| crafting | 57.0% | Yes | 1 |
| mutations | 52.1% | Yes | 1 |
| dice | 69.7% | Yes | 1 |
| buffs | 25.7% | Yes | 1 |
| dialogue | 0.0% | No | 2 |
| enchantments | 0.0% | No | 2 |
| quests | 0.0% | No | 2 |
| skills | 0.0% | No | 2 |
| species | 0.0% | No | 2 |
| rooms | 5.2% | Yes | 2 |
| audio | 0.0% | No | 3 |
| clans | — | No files | 3 |
| colorpatterns | 0.0% | No | 3 |
| configs | 3.3% | Yes | 3 |
| connections | 0.0% | No | 3 |
| conversations | 0.0% | No | 3 |
| devtools | 43.3% | Yes | 3 |
| events | 18.8% | Yes | 3 |
| exit | 0.0% | No | 3 |
| fileloader | 0.0% | No | 3 |
| flags | 0.0% | No | 3 |
| gamelock | 0.0% | No | 3 |
| gametime | 0.0% | No tests | 3 |
| inputhandlers | 3.7% | Yes | 3 |
| language | 78.2% | Yes | 3 |
| llm | 0.0% | No | 3 |
| mapper | 14.8% | Yes | 3 |
| markdown | 78.0% | Yes | 3 |
| migration | 0.0% | No | 3 |
| mobcommands | 0.0% | No | 3 |
| mudlog | 0.0% | No | 3 |
| mutators | 0.0% | No | 3 |
| parties | 0.0% | No | 3 |
| pets | 0.0% | No | 3 |
| plugins | 0.0% | No | 3 |
| prompt | 60.5% | Yes | 3 |
| rooms | 5.2% | Yes | 3 |
| scripting | 0.0% | No tests | 3 |
| statmods | 0.0% | No | 3 |
| stats | 0.0% | No | 3 |
| suggestions | 0.0% | No | 3 |
| templates | 16.2% | Yes | 3 |
| term | 0.0% | No | 3 |
| usercommands | 0.0% | No | 3 |
| users | 12.3% | Yes | 3 |
| util | 69.0% | Yes | 3 |
| uuid | 72.1% | Yes | 3 |
| version | 86.5% | Yes | 3 |
| web | 0.0% | No | 3 |
| worldevents | 0.0% | No | 3 |
| badinputtracker | 100.0% | Yes | 3 |

**Overall Tier 1 weighted average: ~24%** (critical gap).

---

## Section B: Package Tier Classification

### Tier 1 — Critical (Core Gameplay)

| Package | Role |
|---------|------|
| combat | Damage pipeline, grapple, defense resolution, AI |
| dice | All probabilistic rolls |
| characters | Stats, progression, mitigation, defense |
| mutations | Mutation effects, stacking, acquisition |
| crafting | Recipe matching, ingredient checks |
| spells | Spell lookup, cost calc, eligibility |
| items | Item comparison, damage calc, enchantments |
| mobs | Mob templates, AI idle/angry, relationships |
| hooks | Spell resolution, combat round helpers |
| buffs | Buff spec, stacking, duration |

### Tier 2 — Important (Gameplay Quality)

| Package | Role |
|---------|------|
| quests | Quest state tracking |
| dialogue | NPC conversation trees |
| enchantments | Item enchantment logic |
| skills | Skill definitions |
| species | Species traits, stat modifiers |
| rooms | Room properties, exits |

### Tier 3 — Utility / Infrastructure
Everything else: configs, events, templates, users, util, uuid,
language, markdown, mapper, prompt, inputhandlers, connections,
fileloader, web, etc.

---

## Section C: Coverage Targets

| Tier | Target | Rationale |
|------|--------|-----------|
| Tier 1 | **95%** | Core gameplay — bugs here are game-breaking. Most logic is pure math or state transitions. |
| Tier 2 | **85%** | Important features. Some integration complexity but most logic is extractable. |
| Tier 3 | **60%** | Utility/infra — test pure logic; skip raw I/O wrappers and network code. |

### Per-Package Targets (Tier 1 Detail)

| Package | Current | Target | Gap | Notes |
|---------|---------|--------|-----|-------|
| dice | 69.7% | 95% | 25pp | Almost all pure functions. Add wrapper tests. |
| characters | 27.3% | 95% | 68pp | Progression, mitigation, defense all testable. 100+ tests exist as foundation. |
| combat | 3.3% | 95% | 92pp | Pipeline tested. Grapple + defense are pure state machines. |
| mutations | 52.1% | 95% | 43pp | seedRegistry pattern exists. Add remaining 16 effect getters. |
| crafting | 57.0% | 90% | 33pp | Mostly pure. 2 missing functions trivial to test. |
| buffs | 25.7% | 90% | 64pp | Spec lookup, stacking logic are pure. |
| spells | 0.0% | 90% | 90pp | ~95% is pure data + lookups. Easiest zero-to-covered. |
| items | 0.0% | 90% | 90pp | ~80% pure logic (matching, display, damage calc). |
| mobs | 0.0% | 75% | 75pp | ~40% pure today (relationships, idle, stat dist). Rest needs refactoring. |
| hooks | 0.0% | 60% | 60pp | Integration-heavy, but extractable pure logic can reach 60%. |

---

## Section D: Testability Barriers & Refactoring Options

### Barrier 1: Global Singleton Registries

**Affected:** hooks, mobs, usercommands, mobcommands

**Problem:** Functions call `mobs.GetInstance()`, `users.GetByUserId()`,
`rooms.LoadRoom()` directly — global lookups that require the entire
game state initialized.

**Already Solved By:** mutations and crafting packages use a
`seedRegistry()` pattern — manually populate the in-memory map with
test fixtures, bypassing file I/O entirely.

| Option | Effort | Pros | Cons |
|--------|--------|------|------|
| **A. seedRegistry pattern** | Low | No production code changes. Proven in mutations/crafting. | Global state; careful with parallel tests. |
| **B. Interface injection** | Medium | Clean separation. Proper mocking. | ~50+ call-site signature changes in hooks. |
| **C. Package-level test init** | Low | Simple setup per package. | Fragile if internals change. |

**Recommendation:** Option A immediately (proven pattern). Option B as
follow-up only if hooks needs 80%+.

---

### Barrier 2: Interleaved Logic and Side Effects

**Affected:** hooks (spell_resolution.go, NewRound_DoCombat_helpers.go)

**Problem:** Pure calculations (damage formulas, probability checks)
mixed with side effects (HP mutation, message sending, aggro setting)
in the same function body. Can't test math without triggering mutations.

**Example** — `resolveAgainstMob()` in spell_resolution.go:
- Lines 77–79: Pure opposed roll (testable)
- Lines 83–95: Backfire damage calc + HP mutation + message send (mixed)

| Option | Effort | Pros | Cons |
|--------|--------|------|------|
| **A. Extract pure helpers** | Low | No signature changes. Each helper independently testable. | Doesn't test orchestration. |
| **B. Return result structs** | Medium | Full orchestration testable. | Larger refactor; update all call sites. |
| **C. Command pattern** | High | Full undo/replay. | Over-engineered for a MUD. |

**Recommendation:** Option A immediately (extract ~10 pure helpers).
Option B for 3–4 most critical functions in a later stage.

---

### Barrier 3: Embedded RNG

**Affected:** hooks, mobs (AI decisions), combat (grapple)

**Problem:** `util.Rand()` and `dice.Roll()` scattered inside business
logic. Can't get deterministic test results.

| Option | Effort | Pros | Cons |
|--------|--------|------|------|
| **A. Separate chance calc from roll** | Low | Test probability function. Already used in `CalcConcentrationChance`. | Can't test success/failure branches. |
| **B. Inject RNG source** | Medium | Fully deterministic. | Changes function signatures throughout. |
| **C. Statistical testing** | Low | Already proven in dice_test.go (10k iterations). | Slower; can't test specific edges. |

**Recommendation:** Option A for most. Option C for end-to-end. Option B
only if deterministic replay needed.

---

### Barrier 4: File I/O in Constructors

**Affected:** mobs (`LoadMobInstance`), items (`LoadDataFiles`), spells
(`LoadSpellFiles`)

**Problem:** Object creation requires reading YAML files from disk.

| Option | Effort | Pros | Cons |
|--------|--------|------|------|
| **A. Construct test objects directly** | None | Already done in crafting (`makeItem()`). | Doesn't test loading code. |
| **B. Test fixture files** | Low | Go convention (`testdata/`). | Must maintain fixtures. |
| **C. `fs.FS` interface** | Medium | Clean. Tests use `fstest.MapFS`. | Loader signature changes. |

**Recommendation:** Option A for unit tests. Option B for data validation.
Option C deferred.

---

### Package Testability Summary

| Package | % Testable Today | After Refactoring | Key Refactoring |
|---------|-----------------|-------------------|-----------------|
| dice | 95% | 98% | Just add tests for wrapper functions |
| spells | 90% | 95% | seedRegistry for global map |
| crafting | 90% | 95% | Already well-structured |
| items | 85% | 95% | Construct ItemSpec directly in tests |
| characters | 80% | 95% | Character fixtures with equipped items |
| mutations | 75% | 95% | seedRegistry exists; add remaining getters |
| combat | 60% | 95% | Grapple = pure state machines; need char fixtures |
| buffs | 70% | 90% | seedRegistry for buff specs |
| mobs | 40% | 75% | seedRegistry + extract AI decision logic |
| hooks | 20% | 60% | Extract ~10 pure helpers + seedRegistry |

---

## Section E: Critical Functions Inventory

### Dice (5 functions) — all unit testable today

| Function | File | Pure | Notes |
|----------|------|------|-------|
| `RollStat` | dice.go:432 | Yes | Auto-scaled spread |
| `OpposedRollStat` | dice.go:443 | Yes | Contested stat checks |
| `StdDevFor` | dice.go:416 | Yes | Derives stdDev from RollSpread |
| `SuccessChance` | dice.go | Yes | Probability calculation |
| `OpposedSuccessChance` | dice.go | Yes | Two-stat probability |

### Combat / Calculations (3) — unit testable

| Function | File | Pure | Notes |
|----------|------|------|-------|
| `PowerRanking` | calculations.go:13 | Yes | Relative strength 0–1 |
| `ChanceToTame` | calculations.go:49 | Yes | Taming success % |
| `ChanceToSwitchTarget` | calculations.go:119 | Yes | Target-switch % |

### Combat / Grapple (7) — state machines, need char fixtures

| Function | File | Pure | Notes |
|----------|------|------|-------|
| `AttemptGrapple` | grapple.go:58 | No | Opposed roll, reads cooldowns |
| `CheckClinchProgression` | grapple.go:144 | No | Auto-check for clinch |
| `CheckGroundedEscape` | grapple.go:177 | No | Escape with armor mod |
| `AttemptSubmission` | grapple.go:276 | No | Finishing move |
| `IsThirdPartyAttack` | grapple.go:249 | Yes | Pure check |
| `ApplySubmissionFailure` | grapple.go | No | State mutation |
| `ApplySubmissionSuccess` | grapple.go | No | State mutation |

### Combat / AI (8) — mostly pure scoring

| Function | File | Pure | Notes |
|----------|------|------|-------|
| `CanUseBash` | ai.go:151 | Yes | Viability check |
| `CanUseTrip` | ai.go | Yes | Viability check |
| `CanUseKick` | ai.go | Yes | Viability check |
| `CanUseGrapple` | ai.go | Yes | Viability check |
| `CanUseCast` | ai.go | Yes | Viability check |
| `ScoreBash` | ai.go:213 | Yes | Score vs target |
| `ScoreTrip` | ai.go | Yes | Score vs target |
| `ScoreGrapple` | ai.go | Yes | Score vs target |

### Combat / Pipeline (5) — trivial unit tests

| Function | File | Pure | Notes |
|----------|------|------|-------|
| `SkillMultiplier` | damage_pipeline.go:21 | Yes | sqrt curve |
| `DamageScale` | damage_pipeline.go:46 | Yes | Per-channel scale |
| `CalcRawDamage` | damage_pipeline.go:69 | Yes | Unified formula |
| `ApplyMitigation` | damage_pipeline.go:101 | Yes | % reduction |
| `MitigationCap` | damage_pipeline.go:115 | Yes | Per-channel cap |
| `ResourceMultiplier` | damage_pipeline.go:83 | Yes | Depletion penalty |

### Combat / Defense (2) — needs character fixtures

| Function | File | Pure | Notes |
|----------|------|------|-------|
| `runBestOfAllDefense` | combat_helpers.go:353 | No | Best-of-all resolution |
| `resolveDefenseOutcome` | combat_helpers.go | No | Defense outcome |

### Character / Mitigation (3) — needs equipped item fixtures

| Function | File | Pure | Notes |
|----------|------|------|-------|
| `GetPhysicalMitigation` | character.go:962 | Yes | Sum equipment armor |
| `GetMagicalMitigation` | character.go:996 | Yes | Sum magical resist |
| `GetConvictionMitigation` | character.go:1022 | Yes | Sum conviction resist |

### Character / Defense (3) — needs equipped item fixtures

| Function | File | Pure | Notes |
|----------|------|------|-------|
| `GetDefenseScore` | character.go:1664 | Yes | Dodge/Parry/Block score |
| `GetDefenseStaminaCost` | character.go:1703 | Yes | Cost per type |
| `DeductDefenseStamina` | character.go:1731 | No | Mutates stamina |

### Character / Progression (5) — needs config seeding

| Function | File | Pure | Notes |
|----------|------|------|-------|
| `CalculateProgressionChance` | progression.go:43 | Yes | Exponential decay |
| `CheckSkillProgression` | progression.go:67 | No | Rolls advancement |
| `CheckStatProgression` | progression.go:138 | No | Rolls advancement |
| `OnStatUse` | progression.go:204 | No | Tracks use |
| `OnSkillUse` | progression.go:218 | No | Tracks use |

### Mutations (16 untested getters) — unit testable with seedRegistry

| Function | Pure | Notes |
|----------|------|-------|
| `GetMutationLoad` | Yes | Total load calculation |
| `HasConflict` | Yes | Conflict detection |
| `GetStaminaRegenMultiplier` | Yes | Stamina regen effect |
| `GetNaturalWeaponBonus` | Yes | Unarmed bonus |
| `GetConvictionResistance` | Yes | Conviction resist |
| `HasMutationFlag` | Yes | Flag check |
| `GetConditionalHealthRegen` | Yes | Conditional regen |
| `GetHealthRegenMultiplier` | Yes | Regen multiplier |
| `GetDodgeModifier` | Yes | Dodge bonus/penalty |
| `GetDamageMultiplier` | Yes | Damage scaling |
| `GetMovementSpeedModifier` | Yes | Speed effect |
| `GetHealthRegen` | Yes | Base regen |
| `GetSkillProgressionMultiplier` | Yes | Skill gain modifier |
| `GetStatProgressionMultiplier` | Yes | Stat gain modifier |
| `HasMutation` | Yes | Ownership check |
| `GetMutationLevel` | Yes | Level query |

### Crafting (2 untested) — trivial

| Function | File | Pure | Notes |
|----------|------|------|-------|
| `FindTargetItem` | crafting.go:199 | Yes | Item matching |
| `CalcSuccessChance` | crafting.go:221 | Yes | Success % calc |

### Spells (7 untested) — all pure lookups

| Function | File | Pure | Notes |
|----------|------|------|-------|
| `FindSpell` | spells.go:107 | Yes | Fuzzy name search |
| `GetSpell` | spells.go:119 | Yes | By ID |
| `FindSpellByName` | spells.go:126 | Yes | Exact match |
| `GetAllSpells` | spells.go:147 | Yes | Full map |
| `MaxFoldsForSkill` | spells.go:223 | Yes | Fold cap |
| `GetEligibleSpells` | spells.go:249 | Yes | Filtered list |
| `GetTotalConvictionCost` | spells.go | Yes | Cost calc |

### Items (10 key untested) — mostly pure

| Function | Pure | Notes |
|----------|------|-------|
| `IsBetterThan` | Yes | Item comparison |
| `GetDiceRoll` | Yes | Weapon stats |
| `GetDistributionDamage` | Yes | Damage distribution |
| `GetDamage` | Yes | Damage struct |
| `GetDefense` | Yes | Defense value |
| `Equals` | Yes | Equality check |
| `HasAdjective` | Yes | Adjective match |
| `GetLongDescription` | Yes | Display text |
| `IsEnchanted` | Yes | Enchant check |
| `IsCursed` | Yes | Curse check |

### Hooks / Spell Resolution (3+ extractable) — needs refactoring

| Function | File | Pure | Notes |
|----------|------|------|-------|
| `spellDefenseValue` | spell_resolution.go:389 | Yes | Already pure |
| `checkConcentrationBreak` | combat_shared_helpers.go:94 | Yes | Already pure |
| `calcSpellDamageForCharacter` | combat_shared_helpers.go:27 | No | Needs extraction |

**Total critical functions: ~75** (expanded from initial ~45 estimate
after thorough inventory).

---

## Section F: Existing Test Patterns to Reuse

### 1. seedRegistry() — Global Map Population
**Used in:** mutations_test.go, crafting_test.go
**Pattern:** Populate package-level map with hand-crafted test specs.
Call `Validate()` on each to migrate legacy fields. No disk I/O needed.
**Reuse for:** spells, items, mobs, buffs.

### 2. makeItem(tag) / buildOwned(pairs...) — Factory Helpers
**Used in:** crafting_test.go, mutations_test.go
**Pattern:** Lightweight constructors for test objects. `makeItem`
creates an `items.Item` with just a `ComponentTag`. `buildOwned` builds
`map[string]int` from alternating key-value pairs.
**Reuse for:** characters (with equipment), rooms, mobs.

### 3. testify/assert — Assertion Library
**Used in:** characters, buffs, util test files
**Pattern:** `assert.Equal(t, expected, actual)` for cleaner output.
**Standard:** Use `testify/assert` for all new tests.

### 4. Statistical Verification — N Iterations
**Used in:** dice_test.go (10k–100k iterations)
**Pattern:** Run many iterations, verify mean/stddev/percentiles within
tolerance. Also: empirical validation (compare calculated vs observed
probability with 3% tolerance).
**Reuse for:** progression, combat outcomes, grapple win rates.

### 5. Table-Driven Tests — Standard Go Pattern
**Used in:** nearly all existing test files
**Pattern:** `tests := []struct{...}{{...},{...}}` with `t.Run()`.
**Standard:** Mandate for all new tests.

### 6. Float Tolerance — Epsilon Comparisons
**Used in:** mutations, dice, combat pipeline tests
**Patterns:**
- Tight: `math.Abs(got-expected) < 1e-9` (exact math)
- Relative: `math.Abs(got-expected) < expected*0.05` (statistical)
- Absolute: `math.Abs(got-expected) < 0.01` (general purpose)
**Standard:** Use `math.Abs` with appropriate epsilon; never `==` for
floats.

### 7. Monotonicity / Curve Verification
**Used in:** damage_pipeline_test.go, progression_test.go
**Pattern:** Verify function is monotonically increasing/decreasing
without knowing exact values. Catches formula bugs.

### 8. Global State Save/Restore
**Used in:** buffspec_test.go
**Pattern:** `orig := globalVar; defer func() { globalVar = orig }()`
**Standard:** Always restore global state in tests that modify it.

---

## Section G: Stage 40.2–40.4 Checklist

### Stage 40.2 — Core Unit Tests (~55 functions)

| # | Function | Package | Tier | Type | Status |
|---|----------|---------|------|------|--------|
| 1 | RollStat | dice | 1 | unit | TODO |
| 2 | OpposedRollStat | dice | 1 | unit | TODO |
| 3 | StdDevFor | dice | 1 | unit | TODO |
| 4 | SuccessChance | dice | 1 | unit | TODO |
| 5 | OpposedSuccessChance | dice | 1 | unit | TODO |
| 6 | PowerRanking | combat | 1 | unit | TODO |
| 7 | ChanceToTame | combat | 1 | unit | TODO |
| 8 | ChanceToSwitchTarget | combat | 1 | unit | TODO |
| 9 | SkillMultiplier | combat | 1 | unit | TODO |
| 10 | DamageScale | combat | 1 | unit | TODO |
| 11 | CalcRawDamage | combat | 1 | unit | TODO |
| 12 | ApplyMitigation | combat | 1 | unit | TODO |
| 13 | MitigationCap | combat | 1 | unit | TODO |
| 14 | ResourceMultiplier | combat | 1 | unit | TODO |
| 15 | IsThirdPartyAttack | combat | 1 | unit | TODO |
| 16 | CanUseBash | combat | 1 | unit | TODO |
| 17 | CanUseTrip | combat | 1 | unit | TODO |
| 18 | CanUseKick | combat | 1 | unit | TODO |
| 19 | CanUseGrapple | combat | 1 | unit | TODO |
| 20 | CanUseCast | combat | 1 | unit | TODO |
| 21 | ScoreBash | combat | 1 | unit | TODO |
| 22 | ScoreTrip | combat | 1 | unit | TODO |
| 23 | ScoreGrapple | combat | 1 | unit | TODO |
| 24 | GetPhysicalMitigation | characters | 1 | unit | TODO |
| 25 | GetMagicalMitigation | characters | 1 | unit | TODO |
| 26 | GetConvictionMitigation | characters | 1 | unit | TODO |
| 27 | GetDefenseScore | characters | 1 | unit | TODO |
| 28 | GetDefenseStaminaCost | characters | 1 | unit | TODO |
| 29 | CalculateProgressionChance | characters | 1 | unit | TODO |
| 30 | GetMutationLoad | mutations | 1 | unit | TODO |
| 31 | HasConflict | mutations | 1 | unit | TODO |
| 32 | GetStaminaRegenMultiplier | mutations | 1 | unit | TODO |
| 33 | GetNaturalWeaponBonus | mutations | 1 | unit | TODO |
| 34 | GetConvictionResistance | mutations | 1 | unit | TODO |
| 35 | HasMutationFlag | mutations | 1 | unit | TODO |
| 36 | GetConditionalHealthRegen | mutations | 1 | unit | TODO |
| 37 | GetHealthRegenMultiplier | mutations | 1 | unit | TODO |
| 38 | GetDodgeModifier | mutations | 1 | unit | TODO |
| 39 | GetDamageMultiplier | mutations | 1 | unit | TODO |
| 40 | GetMovementSpeedModifier | mutations | 1 | unit | TODO |
| 41 | GetHealthRegen | mutations | 1 | unit | TODO |
| 42 | GetSkillProgressionMult | mutations | 1 | unit | TODO |
| 43 | GetStatProgressionMult | mutations | 1 | unit | TODO |
| 44 | HasMutation | mutations | 1 | unit | TODO |
| 45 | GetMutationLevel | mutations | 1 | unit | TODO |
| 46 | FindTargetItem | crafting | 1 | unit | TODO |
| 47 | CalcSuccessChance | crafting | 1 | unit | TODO |
| 48 | FindSpell / GetSpell | spells | 1 | unit | TODO |
| 49 | MaxFoldsForSkill | spells | 1 | unit | TODO |
| 50 | GetTotalConvictionCost | spells | 1 | unit | TODO |
| 51 | IsBetterThan | items | 1 | unit | TODO |
| 52 | GetDiceRoll | items | 1 | unit | TODO |
| 53 | GetDistributionDamage | items | 1 | unit | TODO |
| 54 | Equals | items | 1 | unit | TODO |
| 55 | HasAdjective | items | 1 | unit | TODO |

### Stage 40.3 — Integration & Scenario Tests (~12 scenarios)

| # | Scenario | Packages | Type | Status |
|---|----------|----------|------|--------|
| 1 | Full melee attack round | combat+characters | integration | TODO |
| 2 | Spell cast → resolve → damage | hooks+spells+combat | integration | TODO |
| 3 | Grapple sequence (clinch→ground→submit) | combat | integration | TODO |
| 4 | Defense resolution (best-of-all) | combat+characters | integration | TODO |
| 5 | Resource depletion → penalty curve | combat+characters | integration | TODO |
| 6 | Skill progression over N uses | characters | statistical | TODO |
| 7 | Stat progression over N uses | characters | statistical | TODO |
| 8 | Crafting loop (check → consume → result) | crafting+items | integration | TODO |
| 9 | Mutation acquisition + stacking | mutations | integration | TODO |
| 10 | Buff application + expiry | buffs+characters | integration | TODO |
| 11 | Item comparison chain | items | integration | TODO |
| 12 | Mob AI move selection | combat | integration | TODO |

### Stage 40.4 — Regression, Refactoring & CI (~10 items)

| # | Task | Type | Status |
|---|------|------|--------|
| 1 | Extract pure helpers from hooks (Barrier 2 Option A) | refactor | TODO |
| 2 | Add hooks unit tests for extracted helpers | test | TODO |
| 3 | seedRegistry for spells package | test infra | TODO |
| 4 | seedRegistry for items package | test infra | TODO |
| 5 | seedRegistry for mobs package | test infra | TODO |
| 6 | Regression test: damage pipeline edge cases | test | TODO |
| 7 | Regression test: mitigation cap enforcement | test | TODO |
| 8 | Regression test: defense floor (MinDefenseChance) | test | TODO |
| 9 | CI coverage gate (fail if Tier 1 < 90%) | CI | TODO |
| 10 | Smoke test: `go build && go test ./...` in CI | CI | TODO |
