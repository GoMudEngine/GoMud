# Code Cleanup 1.2c: `character.go` File Split — Design Spec

## Goal

Split `internal/characters/character.go` (3925 lines, 178 functions)
into focused subject files following the package's existing convention.
After the split, `character.go` holds only the `Character` struct
definition, constructors, and a handful of core accessors (~400 lines).
Each extracted concern lives in either a new subject file or an
existing sibling that already owns that subject.

This is the follow-up to 1.2b, which decomposed three god-functions in
`character.go` but left the file at its full size. 1.2a (combat + spell
god-function refactor) still follows — keeping the combat code stable
for another cycle.

## Scope

**In scope:** 11 new files, 4 extended siblings, 1 trimmed
`character.go`. **Pure file moves** — no renames, no logic changes, no
refactoring beyond relocating functions and their required imports.

**Out of scope:**
- Any behavior change
- Any rename of existing functions or types
- Any inline cleanup (dead code, lint fixes, magic-number extraction)
- Splitting `admin.room.go` further — that was in 1.2b's scope and
  isn't coming back here
- `internal/usercommands/admin.room.go` — untouched by 1.2c

## Decisions Locked During Brainstorming

- **Moderate granularity (8–12 files, ~200–400 lines each).** Matches
  the existing sibling pattern where each file owns one cohesive
  concern at 150–500-line sizing.
- **Match existing naming: subject-based, no `character_` prefix.**
  Preserves the package's design. Preferred over scattering `character_*`
  files alongside the existing `cooldowns.go` / `charminfo.go` etc.
- **Merge into existing siblings when the subject already owns a
  file.** Character methods that obviously belong to `cooldowns`,
  `charminfo`, `formattedname`, or `worn` move into those files rather
  than creating duplicate homes.
- **`combat.go` combines defense + damage-dice methods.** Both are
  "combat-derived calculations" and belong together. This replaces a
  proposed separate `defense.go`.
- **Pure file moves only.** Any bug or improvement opportunity
  discovered during the moves is documented in a follow-up note, not
  fixed inline. Rationale: the commit's value proposition is
  "structurally identical" — adding logic changes forces reviewers to
  do semantic diffing on top of structural diffing, defeating the
  point.
- **Include a server-boot smoke test before marking complete.** Tests
  catch package-level issues; booting the server catches init-order,
  runtime-registration, and cross-package startup issues that unit
  tests don't exercise.

## Architecture

### Final file layout

**`character.go` keeps (~400 lines):**
- `Character` struct (unchanged)
- `New()` / `initAllSkills()` / `ensureAllSkills()` / `RollCharacterStats()`
- `SetUserId` / `GetUserId`
- `SetMiscData` / `GetMiscData` / `GetMiscDataKeys`
- `HasDiscovery` / `AddDiscovery`
- `SetSetting` / `GetSetting`
- `StatMod(statName string) int` (low-level helper used by many callers)

### New files

| File | Methods | Approx lines |
|------|---------|--------------|
| `migrations.go` | All 9 `Migrate*` funcs | ~250 |
| `spells.go` | `GetSpells`, `IsCasting`, `IsCrafting`, `HasSpell`, `DisableSpell`, `EnableSpell`, `TrackSpellCast`, `LearnSpell`, `GetBaseCastSuccessChance`, `SetCast`, `HasRecipe`, `LearnRecipe` (12) | ~220 |
| `inventory.go` | `CarryCapacity`, `GetCarriedWeight`, `StoreItem`, `RemoveItem`, `UpdateItem`, `UseItem`, `FindInPotions`, `UseItemFromPotions`, `FindInComponents`, `FindInBackpack`, `FindOnBody`, `FindItem`, `GetAllBackpackItems`, `GetAllCarriedItems`, `GetRandomItem`, `SortComponentItems`, `SortPotionItems`, `HandsRequired`, key-ring methods (`FindKeyInBackpack`, `HasKey`, `KeyCount`, `GetKey`, `SetKey`) | ~430 |
| `resources.go` | `DeductActionPoints`, `DeductStamina`, `GetMovementStaminaCost`, `GetAttackStaminaCost`, `DeductAttackStamina`, `MovementCost`, `HealthPerRound`, `StaminaPerRound`, `ConvictionPerRound`, `ApplyHealthChange`, `Heal`, `GetToxicityMax`, `AddToxicity`, `GetToxicityPenalties`, `GetDefenseStaminaCost`, `DeductDefenseStamina` | ~250 |
| `skills.go` | `GetSkills`, `SetSkill`, `TrainSkill`, `GetSkillLevel`, `GetSkillLevelCost`, `IncreaseSkill`, `GetAllSkillRanks`, `GetTotalSkillRanks`, `GetCombatSkillTag`, `GetCombatSkillLevel`, `GetModifiedAttackCount`, `IncreaseStat`, `GetStatValue`, `AttemptRecovery`, `KnowsFirstAid` | ~280 |
| `combat.go` | `GetDefense`, `GetPhysicalMitigation`, `GetMagicalMitigation`, `GetConvictionMitigation`, `GetDefenseSequence`, `GetDefenseScore`, `GetDefaultDiceRoll`, `GetDefaultDistributionDamage`, `CalculateUnarmedDamage` | ~230 |
| `quests.go` | `IsQuestDone`, `HasQuest`, `GetQuestProgress`, `GiveQuestToken`, `ClearQuestToken`, `SetQuestFlag`, `GetQuestFlag`, `HasQuestFlag`, `ClearQuestFlag`, `RememberRoom`, `GetMemoryCapacity`, `GetMapSprawlCapacity` | ~200 |
| `buffs.go` | `HasBuffFlag`, `HasFlagFromAnySource`, `CancelBuffsWithFlag`, `HasBuff`, `AddBuff`, `AddBuffScaled`, `TrackBuffStarted`, `GetBuffs`, `RemoveBuff`, `SetPermaBuffs`, `RemovePermaBuff`, `reapplyPermabuffs`, `IsDisabled` | ~200 |
| `aggro.go` | `SetAggroRemote`, `SetAggro`, `EndAggro`, `ClearGrappleState`, `IsAggro`, `TrackPlayerDamage` | ~120 |
| `description.go` | `GetDescription`, `GetMutationVisuals`, `GetHealthAppearance`, `CacheDescription`, `HasAdjective`, `SetAdjective`, `GetAdjectives`, `Species`, `BarterPrice` | ~160 |
| `validate.go` | `Validate`, `validateSkillMigrations`, `validatePoolClamps`, `validateEquipmentItems`, `validateDisabledSlotsForSpecies`, `validateMutationSlots`, `RecalculateStats`, `GetPoolReservation`, `CanDualWield`, `CombatSkillTagForItem` | ~440 |

### Extended siblings

| File | Methods added | Approx additional lines |
|------|---------------|-------------------------|
| `cooldowns.go` | `PruneCooldowns`, `GetCooldown`, `GetAllCooldowns`, `TryCooldown`, `TimerSet`, `TimerExpired`, `TimerExists` | ~100 |
| `charminfo.go` | `TrackCharmed`, `GetCharmIds`, `Charm`, `GetCharmedUserId`, `IsCharmed`, `RemoveCharm`, `GetMaxCharmedCreatures` | ~80 |
| `formattedname.go` | `GetMobName`, `GetMobNameIndexed`, `GetPlayerName`, `GetCharacterName`, `getFormattedName` | ~140 |
| `worn.go` | `Wear`, `wearWeaponOrShield`, `wearArmorSlot`, `RemoveFromBody`, `Uncurse`, `HasShield`, `IsDualWielding`, `IsUnarmed`, `IsUnarmedStyle`, `GetAllWornItems`, `HandsRequired`, `GetGearValue` | ~330 |

Note: `HandsRequired` appears in both `inventory.go` and `worn.go` in
the tables above; it ultimately lives in ONE place. Final home: `worn.go`
(it's an equipment-reasoning helper, not a backpack accessor).
`inventory.go` does NOT own `HandsRequired`.

## Execution Order (15 commits)

One commit per file move. Simpler → complex ordering so early commits
establish the pattern before tackling the bigger extracts.

| # | Commit subject | Target | Size |
|---|----------------|--------|------|
| 1 | `refactor(characters): extract migrations to migrations.go` | new | 9 funcs |
| 2 | `refactor(characters): move cooldown+timer methods to cooldowns.go` | extended | 7 funcs |
| 3 | `refactor(characters): move charm methods to charminfo.go` | extended | 7 funcs |
| 4 | `refactor(characters): move name methods to formattedname.go` | extended | 5 funcs |
| 5 | `refactor(characters): extract aggro to aggro.go` | new | 6 funcs |
| 6 | `refactor(characters): extract description to description.go` | new | 9 funcs |
| 7 | `refactor(characters): extract quests to quests.go` | new | 12 funcs |
| 8 | `refactor(characters): extract buffs to buffs.go` | new | 13 funcs |
| 9 | `refactor(characters): extract resources to resources.go` | new | 16 funcs |
| 10 | `refactor(characters): extract skills to skills.go` | new | 15 funcs |
| 11 | `refactor(characters): extract combat to combat.go` | new | 9 funcs |
| 12 | `refactor(characters): extract inventory to inventory.go` | new | ~22 funcs |
| 13 | `refactor(characters): move equipment methods to worn.go` | extended | 12 funcs |
| 14 | `refactor(characters): extract validate to validate.go` | new | 10 funcs |
| 15 | `docs: mark code cleanup 1.2c complete` | overview | — |

**Rationale for this order:**

- Simplest extracts first. `migrations.go` (9 fully self-contained funcs)
  is the warm-up — gets the "create new file, add package header,
  import block, paste functions, trim unused imports in source,
  build/vet/test" rhythm going.
- Existing-sibling extensions early (2–4) confirm the "paste into
  existing file" pattern before cascading into 11 new files.
- `inventory.go` (biggest single extract) runs late (12) to benefit
  from practice on the smaller extracts.
- `validate.go` is last (14) because its dependencies — `RecalculateStats`,
  `validate*` helpers, `StatMod`, `GetPoolReservation`, `CanDualWield`,
  `CombatSkillTagForItem` — should be in their final homes when the
  move happens. `StatMod` stays in `character.go`; `CombatSkillTagForItem`
  is a package-level function (not a method) and moves with the
  `validate.go` cluster since it's referenced there.

## Per-Commit Verification

```bash
go build ./...
go vet ./...
go test ./internal/characters/...
```

Expected: clean after every commit. Any failure = fix before
committing (no `--no-verify`). If the compiler flags an unused import
after the move, clean it up in the same commit. If the compiler flags a
missing import, add it to the destination file's import block.

## End-of-Branch Verification

After commit 14, before the docs-complete commit (15):

```bash
go test ./...     # full project suite — catches cross-package fallout
go run .          # boot server
```

Server-boot smoke:
1. Start the server.
2. Watch logs for panics / init errors.
3. Connect as a test character.
4. Run `look` — confirm the world loaded.
5. Ctrl+C.

Any failure at any step → stop, diagnose, fix in a new commit before
proceeding to commit 15.

## Risk Register

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Circular import introduced by extracting to a themed file | Low | Pure file moves within a single package don't create cross-package cycles. Intra-package files share the package namespace — no import needed. |
| A moved function depends on an unexported name (`initAllSkills`, `ensureAllSkills`, `startingZone`, etc.) that stays in `character.go` | Low | These are package-level identifiers — still accessible from sibling files. Go compiler catches any missed reference. |
| An import used only by a moved function lingers in `character.go` | Low | `go vet` / goimports flags unused imports; clean them up in the same commit. |
| Post-move tests fail due to a behavior change introduced during the move | Very Low | Pure move policy prohibits semantic changes. If a test fails, the cause is a typo in the copy-paste — fix it, don't relax the test. |
| `character.go` still ends up large because the keeper set wasn't trimmed far enough | Medium | Post-split size target is ~400 lines; actual count audited in commit 14. If >600 lines, investigate what else can be extracted before marking complete. |
| Server fails to boot after the branch lands | Low | End-of-branch smoke test specifically catches this. Abort merge if it fails. |

## Success Criteria

- `character.go` reduced to ≤500 lines (target ~400).
- 11 new themed files exist, each 120–450 lines.
- 4 existing sibling files extended with their relevant Character methods.
- `go build ./...`, `go vet ./...`, `go test ./...` all clean (minus pre-existing `internal/rooms` unrelated failure).
- Server boots with no new panics / init errors.
- `look` renders a room correctly.
- No player-visible behavior change.
