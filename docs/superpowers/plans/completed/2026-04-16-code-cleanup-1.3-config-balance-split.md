# Code Cleanup 1.3: Config Balance File Split Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Split `internal/configs/config.balance.go` (949 lines) into 6 domain files by extracting `Validate()` into themed validator methods. `Balance` struct stays intact.

**Architecture:** Pure mechanical extraction. The struct stays in `config.balance.go`. Each domain gets a `validate<Domain>()` method in its own file. Main `Validate()` becomes a thin dispatcher. Go's package-level scope means all methods on `*Balance` are accessible regardless of file.

**Tech Stack:** Go, existing configs package

**Spec:** `docs/superpowers/specs/2026-04-16-code-cleanup-1.3-config-balance-split-design.md`

---

## File Structure

**Modified:** `internal/configs/config.balance.go`
- `Balance` struct definition (unchanged)
- `Validate()` — becomes a 6-line dispatcher
- `GetBalanceConfig()` accessor (unchanged)
- `GetStatProgressionMultiplier()` / `GetSkillProgressionMultiplier()` helpers (unchanged)

**New files (all in `internal/configs/`):**
- `config.balance.combat.go` — `validateCombat()`
- `config.balance.progression.go` — `validateProgression()`
- `config.balance.spells.go` — `validateSpells()`
- `config.balance.mobs.go` — `validateMobs()`
- `config.balance.shops.go` — `validateShops()`
- `config.balance.misc.go` — `validateMisc()`

---

## Task 1: Extract validateMisc()

**Files:**
- Create: `internal/configs/config.balance.misc.go`
- Modify: `internal/configs/config.balance.go`

Largest and most heterogeneous domain. Doing first while all the other section comments in `Validate()` are still intact as guideposts.

- [ ] **Step 1: Read config.balance.go**

Open `internal/configs/config.balance.go` and locate the `Validate()` method (starts around line 279, ends around line 918).

Identify every `if b.X <= 0 { b.X = default }` block that belongs to the MISC domain. Per the spec, these sections go in misc:
- REGEN RATES (player regen pct for health/stamina/conviction)
- STAMINA & CONVICTION (base pools, per-stat scaling)
- RESOURCE MAXIMUMS (StartingHealth, HealthPenaltyMax, StaminaPenaltyMax, ConvictionPenaltyMax, ResourcePenaltyCurve, HealthBase, StaminaBase, ConvictionBase, HealthPerStrength/Vitality, StaminaPerStrength/Vitality/Willpower, ConvictionPerWilCha)
- CHARACTER CREATION (StatRollMean, StatRollStdDev, StatRollMin, StatRollMax)
- SALVAGE (SalvageMinChance, SalvageMaxChance, SalvageSoftCap, SalvageGoldPerRound, SalvageMaxRounds)
- QUEST ENGINE (QuestChainDepthLimit, QuestLogLevel, QuestPerformanceWarnMs)
- WORLD EVENTS (WorldEventBufferSize)
- CARRY CAPACITY (CarryCapacityMultiplier)
- MIN DEFENSE/ATTACK CHANCE (MinDefenseChance, MinAttackHitChance)
- MOVEMENT (MovementBaseStaminaCost, MovementMaxStaminaCost)
- STAND (StandMinStamina, StandStaminaCost)
- GLOBAL DAMAGE (GlobalDamageMultiplier, HasteSwingMultiplier)
- SKILL MULTIPLIER (SkillMultiplierBase, SkillMultiplierMax)
- THIRD-PARTY GRAPPLE (ThirdPartyGrapplePenalty)
- UNARMED (UnarmedBaseDamage, UnarmedBaseVariance, UnarmedStrengthDivisor, UnarmedSkillDivisor, UnarmedSpeedMultiplier, UnarmedAttackStaminaCost)
- BASH/KICK/STOMP/KNEE/TRIP (damage percents, knockdown chances)
- SURPRISE ATTACK (offhand penalty, extra arm penalties)
- COUP DE GRACE (CoupDeGraceRounds)
- CLINCH (block/dodge/parry penalties)
- GROUNDED (block/dodge/parry/damage penalties)
- LOOT (LootBudgetScalar)
- INSTANCES (InstanceStatPoolCap)
- STORAGE FEES (StorageFeePerItem) — note: this could also go in shops; per spec keep in misc if the extraction is simpler, but put in shops if that's cleaner. Going with shops since it's money-related.

Some fields mentioned above may actually belong to other domains (combat especially). Use judgment based on the section comment where each field's default is set in the current `Validate()`.

- [ ] **Step 2: Create config.balance.misc.go**

Create `internal/configs/config.balance.misc.go` with:

```go
package configs

// validateMisc sets defaults for fields that don't fit a specific
// domain: regen rates, resource pools, character creation, salvage,
// quest engine, world events, carry capacity, unarmed combat helpers,
// special-move damage percentages, clinch/grounded penalties, loot,
// instances.
func (b *Balance) validateMisc() {
    // ── REGEN RATES (fraction of pool max per tick) ────────
    if b.PlayerHealthRegenPct <= 0 {
        b.PlayerHealthRegenPct = 0.02
    }
    // ... continue with every misc default, copied verbatim ...
}
```

Copy each `if b.X <= 0 { b.X = default }` block verbatim from the current `Validate()`. Preserve the order they appear. Preserve the section comments (they're helpful for readers).

- [ ] **Step 3: Remove those blocks from Validate() in config.balance.go**

For every block you copied into `validateMisc()`, delete it from `Validate()`. Leave the section comments there for now — we'll clean them up in Task 7.

- [ ] **Step 4: Call validateMisc() from Validate()**

At the end of `Validate()`, add:

```go
b.validateMisc()
```

- [ ] **Step 5: Verify build**

```bash
cd /c/Users/Calabe\ Davis/workspace/DOGMud && go build ./... && go test ./internal/configs/... 2>&1 | tail -5
```

Expected: build clean, tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/configs/config.balance.go internal/configs/config.balance.misc.go
git commit -m "$(cat <<'EOF'
refactor(configs): extract validateMisc into config.balance.misc.go

Move miscellaneous default-setting logic (regen, resources, character
creation, salvage, quest engine, unarmed combat, special-move
percentages, clinch/grounded penalties, loot, instances) out of
config.balance.go. No behavior change.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Extract validateShops()

**Files:**
- Create: `internal/configs/config.balance.shops.go`
- Modify: `internal/configs/config.balance.go`

- [ ] **Step 1: Read config.balance.go Validate() method**

Find these sections:
- SHOP ECONOMY (ShopPriceFloor, ShopPriceCeiling, ShopBuyRatio, ShopAbundanceThreshold, ShopMaterialReserve, ShopGoldReserveRatio)
- BARTERING (BarterMaxDiscount, BarterMaxBonus)
- STORAGE FEES (StorageFeePerItem) — moved here from misc per Task 1 decision
- CRAFTER MOBS (CrafterEnabled, CrafterMaterialRestockRate, CrafterRareThreshold)
- CRAFTING (CraftingBaseSuccessChance, CraftingMinSuccessChance, CraftingMaxSuccessChance, CraftingSkillBonusPerLevel)
- CRAFT DIFFICULTY (CraftDifficultyProgressionScale)
- RECIPE DISCOVERY (RecipeDiscoveryBaseChance, RecipeDiscoveryDecayRate)

- [ ] **Step 2: Create config.balance.shops.go**

```go
package configs

// validateShops sets defaults for shop economy, bartering, storage
// fees, crafter mobs, crafting success chances, and recipe discovery.
func (b *Balance) validateShops() {
    // ── SHOP ECONOMY ────────────────────────────────────────
    if b.ShopPriceFloor <= 0 {
        b.ShopPriceFloor = 0.25
    }
    // ... continue with every shop default copied verbatim ...
}
```

Copy every default block for the domains listed above.

- [ ] **Step 3: Remove those blocks from Validate()**

- [ ] **Step 4: Call validateShops() from Validate()**

Add `b.validateShops()` after `b.validateMisc()`.

- [ ] **Step 5: Verify build**

```bash
cd /c/Users/Calabe\ Davis/workspace/DOGMud && go build ./... && go test ./internal/configs/... 2>&1 | tail -5
```

- [ ] **Step 6: Commit**

```bash
git add internal/configs/config.balance.go internal/configs/config.balance.shops.go
git commit -m "$(cat <<'EOF'
refactor(configs): extract validateShops into config.balance.shops.go

Move shop economy, bartering, storage fees, crafter mobs, crafting
success, and recipe discovery defaults out of config.balance.go.
No behavior change.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Extract validateSpells()

**Files:**
- Create: `internal/configs/config.balance.spells.go`
- Modify: `internal/configs/config.balance.go`

- [ ] **Step 1: Read config.balance.go Validate() method**

Find these sections:
- COMBAT: SPELL COSTS (SpellConvictionCostMultiplier, SpellHealthCostMultiplier, SelfCastProgressionMultiplier)
- SPELLCASTING (SpellConcentrationBase, SpellInitiationBase, SpellInitiationSkillFactor, SpellInitiationWillpowerDivisor, SpellFoldsSkillFactor, SpellDamageScale, SpellAttackSkillFactor, SpellAvoidanceDamageMultiplier, SpellDifficultyProgressionScale, SpellDiscoveryBaseChance, SpellDiscoveryDecayRate, SpellProficiencyCastsPerPoint)
- ENCHANTMENTS (EnchantMaxTier, EnchantRemovalPenaltyRounds, EnchantTierUpBaseChance, EnchantTierUsesBase, EnchantTierUsesScale)

- [ ] **Step 2: Create config.balance.spells.go**

```go
package configs

// validateSpells sets defaults for spell costs, spellcasting
// parameters (folds, initiation, concentration, damage scales,
// discovery), and enchantments.
func (b *Balance) validateSpells() {
    // ── COMBAT: SPELL COSTS ─────────────────────────────────
    if b.SpellConvictionCostMultiplier <= 0 {
        b.SpellConvictionCostMultiplier = 1.0
    }
    // ... continue with every spell default copied verbatim ...
}
```

- [ ] **Step 3: Remove those blocks from Validate()**

- [ ] **Step 4: Call validateSpells() from Validate()**

Add `b.validateSpells()` after the previous calls.

- [ ] **Step 5: Verify build**

```bash
cd /c/Users/Calabe\ Davis/workspace/DOGMud && go build ./... && go test ./internal/configs/... 2>&1 | tail -5
```

- [ ] **Step 6: Commit**

```bash
git add internal/configs/config.balance.go internal/configs/config.balance.spells.go
git commit -m "$(cat <<'EOF'
refactor(configs): extract validateSpells into config.balance.spells.go

Move spell costs, spellcasting parameters (folds, initiation,
concentration, damage scales, discovery), and enchantment defaults
out of config.balance.go. No behavior change.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Extract validateMobs()

**Files:**
- Create: `internal/configs/config.balance.mobs.go`
- Modify: `internal/configs/config.balance.go`

- [ ] **Step 1: Read config.balance.go Validate() method**

Find these sections:
- MOB AI (MobAIEnabled, CombatMemoryDuration, MobReactionDelayMin, MobReactionDelayMax, MobBTreeReactionBase, MobBTreeReactionPerceptionScale)
- MOB REGEN (MobHealthRegenPct, MobStaminaRegenPct, MobConvictionRegenPct)
- MOB DAMAGE (MobDamageMultiplier)
- MOB PROGRESSION (MobProgressionEnabled, MobProgressionRate, MobStatCap, MobSkillCap, MobSaveIntervalRounds, MobInstanceMaxAgeDays)
- MOB MUTATIONS (MobMutationEnabled, MobMutationRate)
- PACK SCALING (PackScalingEnabled, PackMaxSize, PackSurvivalRounds, PackScatterRounds, PackMaxBonus, PackBonusTrainingPts)
- PACK ROAMING (PackRoamingEnabled)
- GOSSIP (GossipIntervalRounds)
- MOON PHASES (MoonStatModMax)
- MANIFESTATION / COMPANION SCALING (ManifestStatScaleChaFactor, ManifestStatScaleSkillFactor)

- [ ] **Step 2: Create config.balance.mobs.go**

```go
package configs

// validateMobs sets defaults for mob AI, regen, progression,
// mutations, pack scaling/roaming, gossip, moon phases, and
// manifestation/companion scaling.
func (b *Balance) validateMobs() {
    // ── MOB AI ──────────────────────────────────────────────
    // MobAIEnabled is a ConfigBool; no zero-check needed for bools
    if b.CombatMemoryDuration <= 0 {
        b.CombatMemoryDuration = 300
    }
    // ... continue with every mob default copied verbatim ...
}
```

- [ ] **Step 3: Remove those blocks from Validate()**

- [ ] **Step 4: Call validateMobs() from Validate()**

Add `b.validateMobs()` after the previous calls.

- [ ] **Step 5: Verify build**

```bash
cd /c/Users/Calabe\ Davis/workspace/DOGMud && go build ./... && go test ./internal/configs/... 2>&1 | tail -5
```

- [ ] **Step 6: Commit**

```bash
git add internal/configs/config.balance.go internal/configs/config.balance.mobs.go
git commit -m "$(cat <<'EOF'
refactor(configs): extract validateMobs into config.balance.mobs.go

Move mob AI, regen, progression, mutations, pack scaling, gossip,
moon phases, and manifestation/companion defaults out of
config.balance.go. No behavior change.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Extract validateProgression()

**Files:**
- Create: `internal/configs/config.balance.progression.go`
- Modify: `internal/configs/config.balance.go`

- [ ] **Step 1: Read config.balance.go Validate() method**

Find these sections:
- PROGRESSION (SkillSoftCap, BaseProgressionChance, ProgressionDecayBelowCap, ProgressionDecayAboveCap, StatSoftCap, StatSoftCapThreshold, StatSoftCapMultiplier, UsesPerRank, SkillWeight, ProgressionDecayAboveCap, RegenProgressionBase, RegenProgressionCurve)
- PROGRESSION MULTIPLIERS (StatProgressionMultipliers, SkillProgressionMultipliers — these are maps, check for nil and populate defaults)
- MUTATIONS (MutationMaxCount, MutationMaxLevel, MutationBaseProgress, MutationProgressGainPerRound, MutationProgressScale, MutationDeepenChance, MutationLevel2Multiplier, MutationLevel3Multiplier, MutationLevel4Multiplier)

- [ ] **Step 2: Create config.balance.progression.go**

```go
package configs

// validateProgression sets defaults for skill and stat progression
// curves, per-stat/per-skill multipliers, regen-based progression,
// and mutation progression.
func (b *Balance) validateProgression() {
    // ── PROGRESSION ─────────────────────────────────────────
    if b.SkillSoftCap <= 0 {
        b.SkillSoftCap = 50
    }
    // ... continue with every progression default copied verbatim ...
}
```

Note: map fields like `StatProgressionMultipliers` may need nil checks or default population — copy verbatim from the current Validate().

- [ ] **Step 3: Remove those blocks from Validate()**

- [ ] **Step 4: Call validateProgression() from Validate()**

Add `b.validateProgression()` after the previous calls.

- [ ] **Step 5: Verify build**

```bash
cd /c/Users/Calabe\ Davis/workspace/DOGMud && go build ./... && go test ./internal/configs/... 2>&1 | tail -5
```

- [ ] **Step 6: Commit**

```bash
git add internal/configs/config.balance.go internal/configs/config.balance.progression.go
git commit -m "$(cat <<'EOF'
refactor(configs): extract validateProgression into config.balance.progression.go

Move skill/stat progression curves, per-stat/per-skill multipliers,
regen progression, and mutation progression defaults out of
config.balance.go. No behavior change.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Extract validateCombat()

**Files:**
- Create: `internal/configs/config.balance.combat.go`
- Modify: `internal/configs/config.balance.go`

Everything remaining in `Validate()` should be combat-related after Tasks 1-5.

- [ ] **Step 1: Read config.balance.go Validate() method**

Every remaining `if b.X <= 0 { ... }` block should belong to the COMBAT domain. Expected sections:
- ROLL SPREAD (RollSpread)
- COMBAT: DEFENSE COSTS (DodgeStaminaCost, ParryStaminaCost, BlockStaminaCost if present — check exact field names)
- COMBAT: DEFENSE EFFECTIVENESS (DodgeMultiplier, ParryMultiplier, BlockMultiplier, DodgeEffectiveness, ParryEffectiveness, BlockEffectiveness)
- COMBAT: PRONE & GRAPPLE (ProneAttackMultiplier, ProneVulnerabilityMultiplier, ProneDamagePenalty, ProneBlockPenalty, ProneDodgePenalty, ProneParryPenalty, plus any grapple fields)
- COMBAT: SPECIAL MOVES (SpecialMoveCooldown — actual move damage percentages went to misc)
- SKULLDUGGERY (ShadowCooldown, StealCooldown, StealHiddenBonus, StealSkillMultiplier, SneakFailCooldown)
- COMBAT: DARKNESS (DarknessCombatPenalty)
- COMBAT: MESSAGES (ConsistentAttackMessages)
- COMBAT: DAMAGE (MeleeDamageScale, SpellDamageScale — wait, SpellDamageScale went to spells; check here for MeleeDamageScale, RhetoricDamageScale, RhetoricAvoidanceDamageMultiplier, PhysicalMitigationCap, MagicalMitigationCap, ConvictionMitigationCap, per-channel damage scales)
- RESOURCE DEPLETION PENALTIES (ResourcePenaltyCurve, HealthPenaltyMax, StaminaPenaltyMax, ConvictionPenaltyMax) — wait, these went to misc per Task 1 decision; verify they're not duplicated
- TOXICITY (ToxicityBaseMax, ToxicityVitalityScale, ToxicityDecayPerTick)

If during Task 1 you put some of these in misc, they stay there. The goal is: after Task 6, `Validate()` has no `if b.X <= 0` blocks left, only the 6 domain dispatcher calls.

- [ ] **Step 2: Create config.balance.combat.go**

```go
package configs

// validateCombat sets defaults for combat rolls, defense costs and
// effectiveness, prone/grapple penalties, special moves, skullduggery,
// darkness, damage channels, mitigation caps, and toxicity.
func (b *Balance) validateCombat() {
    // ── ROLL SPREAD ─────────────────────────────────────────
    if b.RollSpread <= 0 {
        b.RollSpread = 0.15
    }
    // ... continue with every combat default copied verbatim ...
}
```

- [ ] **Step 3: Remove those blocks from Validate()**

At this point, `Validate()` should look like:

```go
func (b *Balance) Validate() {
    b.validateCombat()
    b.validateProgression()
    b.validateSpells()
    b.validateMobs()
    b.validateShops()
    b.validateMisc()
}
```

(~8 lines total, including braces)

- [ ] **Step 4: Verify build**

```bash
cd /c/Users/Calabe\ Davis/workspace/DOGMud && go build ./... && go test ./internal/configs/... 2>&1 | tail -5
```

- [ ] **Step 5: Commit**

```bash
git add internal/configs/config.balance.go internal/configs/config.balance.combat.go
git commit -m "$(cat <<'EOF'
refactor(configs): extract validateCombat into config.balance.combat.go

Move combat rolls, defense, prone/grapple, special moves,
skullduggery, darkness, damage channels, mitigation caps, and
toxicity defaults out of config.balance.go. Validate() is now
a thin dispatcher (~8 lines). No behavior change.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Final verification

**Files:** All 7 config.balance*.go files

- [ ] **Step 1: Verify final file structure**

```bash
cd /c/Users/Calabe\ Davis/workspace/DOGMud && ls internal/configs/config.balance*.go
```

Expected output (alphabetical order):
```
internal/configs/config.balance.combat.go
internal/configs/config.balance.go
internal/configs/config.balance.mobs.go
internal/configs/config.balance.misc.go
internal/configs/config.balance.progression.go
internal/configs/config.balance.shops.go
internal/configs/config.balance.spells.go
```

- [ ] **Step 2: Verify file sizes are reasonable**

```bash
wc -l internal/configs/config.balance*.go
```

Expected: main `config.balance.go` reduced from 949 to ~340 lines. Each domain file 80-250 lines.

- [ ] **Step 3: Verify Validate() is a thin dispatcher**

```bash
grep -A 10 "^func (b \*Balance) Validate() {" internal/configs/config.balance.go
```

Expected: 6 `b.validate<Domain>()` calls, nothing else.

- [ ] **Step 4: Verify each domain file has one method**

```bash
for f in internal/configs/config.balance.{combat,progression,spells,mobs,shops,misc}.go; do
    echo "=== $f ==="
    grep "^func (b \*Balance)" "$f"
done
```

Expected: one `validate<Domain>()` method per file.

- [ ] **Step 5: Final build and test**

```bash
cd /c/Users/Calabe\ Davis/workspace/DOGMud
go build ./...
go vet ./internal/configs/...
go test ./internal/configs/...
```

All three must succeed.

- [ ] **Step 6: Manual server smoke test**

```bash
go run .
```

Server must start without errors. Stop with Ctrl+C once you see "Server Ready".

- [ ] **Step 7: Spot-check that no default was lost**

```bash
# Count total default assignments across all split files
grep -c "b\." internal/configs/config.balance.combat.go \
  internal/configs/config.balance.progression.go \
  internal/configs/config.balance.spells.go \
  internal/configs/config.balance.mobs.go \
  internal/configs/config.balance.shops.go \
  internal/configs/config.balance.misc.go | \
  awk -F: '{sum += $2} END {print "Total b.X assignments in domain files:", sum}'
```

This should roughly match the count of `b.X` assignments in the original Validate() method (check git history: `git show <first-task-commit>^:internal/configs/config.balance.go | grep -c "b\."` to compare).

If counts differ significantly, some defaults may have been dropped. Review the git diff to find them.

- [ ] **Step 8: No separate commit needed**

If any fixes are required from the verification, fix them and amend the relevant task's commit. Otherwise, Task 7 produces no new commit — it's a verification checkpoint.

---

## Domain assignment reference

When in doubt about which domain a field belongs to, use this table:

| Field prefix / topic | Domain file |
|---|---|
| `RollSpread`, `Dodge*`, `Parry*`, `Block*`, `Prone*`, `Clinch*`, `Grounded*`, `Coup*`, `SpecialMove*`, `Shadow*`, `Steal*`, `Sneak*`, `Darkness*`, `Physical/Magical/ConvictionMitigation*`, `MeleeDamage*`, `RhetoricDamage*`, `RhetoricAvoidance*`, `Toxicity*`, `ConsistentAttackMessages` | combat |
| `SkillSoftCap`, `StatSoftCap*`, `BaseProgressionChance`, `ProgressionDecay*`, `StatProgressionMultipliers`, `SkillProgressionMultipliers`, `UsesPerRank`, `SkillWeight`, `Mutation*`, `RegenProgression*` | progression |
| `Spell*`, `Enchant*`, `SelfCastProgressionMultiplier` | spells |
| `Mob*`, `Pack*`, `Gossip*`, `Moon*`, `Manifest*` | mobs |
| `Shop*`, `Barter*`, `Storage*`, `Crafter*`, `Crafting*`, `Recipe*`, `CraftDifficulty*` | shops |
| `PlayerHealthRegenPct`, `PlayerStaminaRegenPct`, `PlayerConvictionRegenPct`, `Health*`/`Stamina*`/`Conviction*` base pools, `StartingHealth`, `StatRoll*`, `Salvage*`, `Quest*`, `WorldEvent*`, `CarryCapacity*`, `MinDefenseChance`, `MinAttackHitChance`, `Movement*`, `Stand*`, `GlobalDamage*`, `HasteSwing*`, `SkillMultiplier*`, `ThirdParty*`, `Unarmed*`, `Bash*`/`Kick*`/`Stomp*`/`Knee*`/`Trip*` damage percents, `SurpriseAttack*`, `Loot*`, `Instance*` | misc |

If a field could go in multiple domains, pick the one where the original section comment placed it. The goal is "one domain per field" — no field should be validated in multiple files.

---

## Important notes

- **Go can't split structs.** The `Balance` struct stays entirely in
  `config.balance.go`. Only the `Validate()` method gets split.
- **All methods on `*Balance` share the same receiver scope** regardless
  of which file they're defined in. This is Go's normal package-level
  behavior.
- **Don't add new validation logic.** If you find a field with no default
  being set, that's pre-existing — not your job to fix here. Note it
  as a concern in your commit message if you want.
- **Map field initialization** (like `StatProgressionMultipliers`) may
  look different from simple `if b.X <= 0` patterns — copy verbatim.
- **ConfigBool fields** (like `MobAIEnabled`) default to `false` and
  usually don't need zero-check blocks. Leave them out of validators.
