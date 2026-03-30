# Alchemy Rework Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Overhaul alchemy with witcher-style potions, item aging, toxicity system, skill-scaled potency, progression potions, and bottle tiers.

**Architecture:** Five phases: (1) Engine features — Item fields + toxicity + aging, (2) New materials + bottles, (3) New recipes + buffs + potion items, (4) Consumption rework — drink command with aging/toxicity, (5) Migration + content integration + help files. Each phase produces a compilable, testable state.

**Tech Stack:** Go, YAML data files, Go templates

**Spec:** `docs/superpowers/specs/2026-03-30-alchemy-rework-design.md`

---

## Phase 1: Engine Features

### Task 1: Item Fields — CraftedRound + CraftSkill

**Files:**
- Modify: `internal/items/items.go` — add fields to Item struct
- Modify: `internal/crafting/crafting.go` — stamp fields on craft completion
- Modify: `internal/hooks/NewRound_UserRoundTick.go` — stamp fields on craft completion

- [ ] **Step 1:** Add `CraftedRound uint64` and `CraftSkill int` to the Item struct in `items.go` (after `LastUsedRound`, with `yaml:"crafted_round,omitempty"` and `yaml:"craft_skill,omitempty"` tags).

- [ ] **Step 2:** In the crafting completion code in `NewRound_UserRoundTick.go`, after a successful craft creates the output item (`items.New(recipe.Output.ItemId)`), set `newItem.CraftedRound = util.GetRoundCount()` and `newItem.CraftSkill = user.Character.GetSkillLevel(skills.SkillTag(recipe.Skill))`.

- [ ] **Step 3:** Build and test: `go build ./... && go test ./...`

- [ ] **Step 4:** Commit: `feat: add CraftedRound and CraftSkill fields to Item`

---

### Task 2: Toxicity System — Character Fields + Decay

**Files:**
- Modify: `internal/characters/character.go` — add Toxicity field
- Modify: `internal/hooks/NewRound_UserRoundTick.go` or `NewRound_AutoHeal.go` — toxicity decay per tick
- Modify: `internal/configs/config.balance.go` — toxicity config knobs

- [ ] **Step 1:** Add to Character struct: `Toxicity float64 \`yaml:"toxicity,omitempty"\``

- [ ] **Step 2:** Add to Balance config struct:
```go
ToxicityDecayPerTick    ConfigFloat `yaml:"ToxicityDecayPerTick"`    // Points per regen tick (default 1.0)
ToxicityBaseMax         ConfigFloat `yaml:"ToxicityBaseMax"`         // Base max before vitality bonus (default 100)
ToxicityVitalityScale   ConfigFloat `yaml:"ToxicityVitalityScale"`   // Vitality divisor for max bonus (default 5)
```
With validation defaults in Validate().

- [ ] **Step 3:** Add `GetToxicityMax()` method to Character:
```go
func (c *Character) GetToxicityMax() float64 {
    bal := configs.GetBalanceConfig()
    return float64(bal.ToxicityBaseMax) + float64(c.Stats.Vitality.ValueAdj)/float64(bal.ToxicityVitalityScale)
}
```

- [ ] **Step 4:** Add toxicity decay to the regen tick loop (in `NewRound_AutoHeal.go` or equivalent). Each tick: `c.Toxicity -= decayRate; if c.Toxicity < 0 { c.Toxicity = 0 }`.

- [ ] **Step 5:** Add toxicity threshold stat penalties. Create a method `GetToxicityPenalties()` that returns stat multipliers based on threshold (0-50%: none, 50-75%: -10% regen/-10% Per, 75-90%: -20% regen/-10% Per/-10% Dex, 90-100%: -40% regen/-20% Per/-10% Dex). Wire these into stat calculation (similar to how prone penalties work).

- [ ] **Step 6:** Build and test.

- [ ] **Step 7:** Commit: `feat: toxicity system with decay and threshold penalties`

---

### Task 3: Potion Aging Engine

**Files:**
- Create: `internal/items/aging.go` — aging phase calculation + descriptions
- Modify: `internal/usercommands/look.go` — show age info based on alchemy skill
- Modify: `internal/usercommands/inventory.go` — spoiled potions display separately

- [ ] **Step 1:** Create `internal/items/aging.go` with:

```go
// AgingPhase represents a potion's lifecycle phase
type AgingPhase int

const (
    PhaseFresh AgingPhase = iota
    PhaseFermented
    PhasePeak
    PhaseDeclining
    PhaseSpoiled
)

// AgingThresholds defines the round counts for each phase transition
type AgingThresholds struct {
    FermentRounds int `yaml:"ferment_rounds"`
    PeakRounds    int `yaml:"peak_rounds"`
    DecayRounds   int `yaml:"decay_rounds"`
    SpoilRounds   int `yaml:"spoil_rounds"`
}

// GetAgingPhase returns the current phase and potency multiplier
// for a potion based on elapsed rounds and bottle aging multiplier.
func GetAgingPhase(elapsedRounds uint64, thresholds AgingThresholds,
    bottleMultiplier float64) (AgingPhase, float64) {
    // Apply bottle multiplier to thresholds (higher = faster aging)
    ferment := float64(thresholds.FermentRounds) / bottleMultiplier
    peak := float64(thresholds.PeakRounds) / bottleMultiplier
    decay := float64(thresholds.DecayRounds) / bottleMultiplier
    spoil := float64(thresholds.SpoilRounds) / bottleMultiplier

    elapsed := float64(elapsedRounds)

    switch {
    case elapsed < ferment:
        return PhaseFresh, 1.0
    case elapsed < peak:
        return PhaseFermented, 1.15
    case elapsed < decay:
        return PhasePeak, 1.30
    case elapsed < spoil:
        // Linear decline from 1.30 to 0.5
        progress := (elapsed - decay) / (spoil - decay)
        return PhaseDeclining, 1.30 - (0.80 * progress)
    default:
        return PhaseSpoiled, 0.0
    }
}

// GetPhaseDescription returns descriptive text based on alchemy skill
func GetPhaseDescription(phase AgingPhase, alchemySkill int,
    elapsedRounds uint64, thresholds AgingThresholds,
    bottleMultiplier float64) string {
    // Skill 0-5: no info
    // Skill 6-15: general feel
    // Skill 16-30: phase + direction
    // Skill 30+: exact phase + urgency
    // (implement full switch logic per spec)
}
```

- [ ] **Step 2:** Add `AgingThresholds` field to ItemSpec:
```go
Aging AgingThresholds `yaml:"aging,omitempty"`
```

- [ ] **Step 3:** Add `BottleAgingMultiplier` field to ItemSpec:
```go
BottleAgingMultiplier float64 `yaml:"bottle_aging_multiplier,omitempty"`
```
Bottle items set this (clay flask = 3.0, glass vial = 1.0, sealed phial = 0.5, crystalline decanter = 0.25).

- [ ] **Step 4:** In `look.go`, when examining a potion item (has aging thresholds), compute the phase and show description based on the viewer's alchemy skill. Use `GetPhaseDescription()`.

- [ ] **Step 5:** In `inventory.go`, when building stacked display: potions stack by ItemId. If the player's alchemy skill is 6+, spoiled potions display separately with `(turned)` suffix in grey.

- [ ] **Step 6:** Build and test.

- [ ] **Step 7:** Commit: `feat: potion aging engine with phase calculation and skill-based detection`

---

### Task 4: Potion Bandolier (Belt Trade-off)

**Files:**
- Modify: `internal/characters/character.go` — PotionItems slice, StoreItem routing
- Modify: `internal/usercommands/inventory.go` — Potions section
- Modify: `internal/usercommands/drink.go` — consume from bandolier first
- Modify: `internal/characters/worn.go` — RemoveFromBody spills potions
- Create: item YAMLs for bandoliers
- Create: tailoring recipe YAMLs for bandoliers

- [ ] **Step 1:** Add `PotionItems []items.Item \`yaml:"potionitems,omitempty"\`` to Character struct.

- [ ] **Step 2:** Add `IsBandolier` bool and `BandolierCapacity` int to ItemSpec (or reuse BagCapacity since the pattern is the same as component bag).

- [ ] **Step 3:** In `StoreItem()`, add auto-routing: if item is a potion (type `potion` or has `buffids`) and a bandolier is equipped in Belt slot with capacity, route to PotionItems.

- [ ] **Step 4:** In `drink.go`, search PotionItems first (oldest first based on CraftedRound), then backpack.

- [ ] **Step 5:** In `inventory.go`, add "Potions:" display section using same stacking pattern as Components.

- [ ] **Step 6:** In RemoveFromBody for Belt slot, if the removed item was a bandolier, spill PotionItems to backpack.

- [ ] **Step 7:** Create bandolier item YAMLs:
- `_datafiles/world/dogmud/items/armor-20000/belt/20059-leather_bandolier.yaml` (capacity 6)
- `_datafiles/world/dogmud/items/armor-20000/belt/20060-reinforced_bandolier.yaml` (capacity 12)

- [ ] **Step 8:** Create tailoring recipes:
- `_datafiles/world/dogmud/recipes/tailoring/leather-bandolier.yaml` (skill 10)
- `_datafiles/world/dogmud/recipes/tailoring/reinforced-bandolier.yaml` (skill 20)

- [ ] **Step 9:** Build and test.

- [ ] **Step 10:** Commit: `feat: potion bandolier system with belt trade-off`

---

## Phase 2: Materials & Bottles

### Task 5: New Material Items

**Files:**
- Create: 7 new item YAMLs in `_datafiles/world/dogmud/items/materials-40000/`
- Modify: `_datafiles/world/dogmud/items/materials-40000/40006-small_vial.yaml` — rename to glass vial

- [ ] **Step 1:** Rename small vial: change `name: small vial` to `name: glass vial`, add `bottle_aging_multiplier: 1.0`.

- [ ] **Step 2:** Create 7 new material items (IDs 40043-40049):
- 40043: Clay Flask (`bottle_aging_multiplier: 3.0`, value 1, is_component true)
- 40044: Sealed Phial (`bottle_aging_multiplier: 0.5`, value 10, is_component true)
- 40045: Crystalline Decanter (`bottle_aging_multiplier: 0.25`, value 30, is_component true)
- 40046: Moonpetal (forage rare, value 8, is_component true)
- 40047: Veilbloom Petal (forage very rare / boss drop, value 25, is_component true)
- 40048: Serpent Venom Sac (mob drop, value 12, is_component true)
- 40049: Ironbark Shaving (forage uncommon, value 4, is_component true)

All with appropriate `component_tag` values.

- [ ] **Step 3:** Create 2 new jewelcrafting recipes:
- `sealed-phial.yaml` (skill 8): copper wire + chrysalis shard + glass vial → sealed phial
- `crystalline-decanter.yaml` (skill 20): gemstone + chrysalis shard + glass vial ×2 → crystalline decanter

- [ ] **Step 4:** Update Voss vendor stock: add clay flask (40043). Glass vial (40006) already sold.

- [ ] **Step 5:** Commit: `feat: new alchemy materials — bottles, moonpetal, veilbloom, venom sac, ironbark`

---

### Task 6: Forage Table Updates

**Files:**
- Modify: `internal/usercommands/skill.forage.go`

- [ ] **Step 1:** Add new materials to forage yield tables:

```go
// Add to existing biome yield maps:
"forest":    {...existing..., 40049, 40049},        // ironbark shaving (uncommon)
"land":      {...existing..., 40049},                // ironbark shaving (rare in open land)
"mountains": {...existing..., 40046},                // moonpetal (rare, appears at night check below)
"cave":      {...existing..., 40046},                // moonpetal (rare in caves)
```

- [ ] **Step 2:** Add night-only forage logic for moonpetal: check if the current game time is night. If so, moonpetal appears in the yield table for wilderness/mountains/cave biomes. If day, it's removed. This requires checking `gametime.IsNight()` or equivalent.

- [ ] **Step 3:** Add veilbloom petal as an extremely rare forage in steppe biome only:
```go
"steppe": {...existing..., 40047}, // veilbloom — only 1 entry = very rare
```

- [ ] **Step 4:** Add serpent venom sac to mob drop tables: update river lurker (87) and blind stalker (227) mob YAMLs to include itemid 40048 in their `items:` list.

- [ ] **Step 5:** Build and test.

- [ ] **Step 6:** Commit: `feat: forage tables + mob drops for new alchemy materials`

---

## Phase 3: Recipes, Buffs & Potion Items

### Task 7: New Buff Definitions (~21 buffs)

**Files:**
- Create: ~21 new buff YAMLs in `_datafiles/world/dogmud/buffs/`

- [ ] **Step 1:** Create buff YAMLs for all 21 potions. Each buff needs: buffid, name, description, triggerrate, triggercount, statmods (for potions with stat effects), flags (for utility effects like night-vision, see-hidden, haste, poison-immunity).

Start with the next available buff ID (52+ already used for chrysalis shell and veil sight, so start at 54).

**Buff IDs 54-74 (21 buffs):**

Pool regen buffs (7): healing-salve, stamina-tonic, conviction-draught, warriors-brew, preachers-tincture, windrunner-draught, elixir-of-renewal

Combat/utility buffs (10): ironhide-brew, mindshield-elixir, veilguard-tonic, stone-stomach, cats-eye-draught, swiftfoot-essence, berserker-elixir (new version), silver-tongue-oil, battle-trance, purging-draught-debuff

Progression buffs (4): essence-of-growth, savants-infusion, mutagen-brew, chrysalis-catalyst

Each buff should have:
- `triggercount` set to the base duration from the spec (300-1000)
- `triggerrate: 1 round`
- Appropriate statmods for drawbacks (e.g., berserker: `strength_multiplier: 0.20, dexterity_multiplier: -0.15, perception_multiplier: -0.10`)
- Flags as needed (`haste`, `see-hidden`, `night-vision`, `poison-immunity`)

For pool reservation buffs (progression potions), use the existing `reservepool` pattern from enchantments.

- [ ] **Step 2:** Create a spoiled-potion-nausea debuff (~buff 75): -15% all stats for 30 rounds.

- [ ] **Step 3:** Create purging-draught-debuff (~buff 76): -15 flat Vit/Wil/Cha for 50 rounds.

- [ ] **Step 4:** Build (verify buff YAMLs load without error).

- [ ] **Step 5:** Commit: `feat: 23 new buff definitions for alchemy rework`

---

### Task 8: New Potion Item Definitions

**Files:**
- Create: ~21 new potion item YAMLs in `_datafiles/world/dogmud/items/consumables-30000/`

- [ ] **Step 1:** Create potion item YAMLs for all 21 potions. Each needs: itemid, name, description, type (potion or object), subtype (drinkable or usable), uses (1), buffids (pointing to the new buff), value, weight, and the new `toxicity` field and `aging` thresholds.

Use item IDs starting at 30036 (30033-30035 already used for component bags).

Each potion YAML includes:
```yaml
toxicity: 8          # toxicity cost when consumed
aging:
  ferment_rounds: 1000
  peak_rounds: 5000
  decay_rounds: 15000
  spoil_rounds: 25000
```

Aging thresholds vary by potion tier (basic potions age faster/shorter lifecycle, progression potions age much slower/longer lifecycle).

- [ ] **Step 2:** Add `Toxicity int` field to ItemSpec if not already present:
```go
Toxicity int `yaml:"toxicity,omitempty"` // Toxicity cost when consumed
```

- [ ] **Step 3:** Build and verify all items load.

- [ ] **Step 4:** Commit: `feat: 21 new potion item definitions with aging thresholds`

---

### Task 9: New Alchemy Recipe Definitions

**Files:**
- Create: 22 recipe YAMLs in `_datafiles/world/dogmud/recipes/alchemy/`
- Delete or archive: 8 old alchemy recipe YAMLs

- [ ] **Step 1:** Remove (or rename with `.old` suffix) the 8 existing alchemy recipes: healing-poultice, stamina-draught, conviction-draught, minor-antidote, clarity-tonic, fire-resistance-draught, greater-healing-poultice, berserker-elixir.

- [ ] **Step 2:** Create 22 new recipe YAMLs. Each recipe has skill, skill_minimum, station (alchemy_bench), time_rounds, ingredients (with component_tag references including the bottle), and output (item_id pointing to new potion items).

IMPORTANT: The bottle ingredient should use a generic `bottle` component_tag. All 4 bottle types must share this tag. Update the bottle item YAMLs to have `component_tag: bottle`. The player chooses which bottle to use by having the desired bottle type in inventory — the crafting system consumes the first matching `bottle` tag it finds. The `bottle_aging_multiplier` from the consumed bottle gets baked into the output potion item.

- [ ] **Step 3:** Create the "Distill Remnants" recipe (skill 5): takes a spoiled potion as input, produces 1-2 random base ingredients. This needs special handling — the input is not a component_tag but a spoiled potion item. Implement as a special recipe type or use an `enchant_type`-style target system.

- [ ] **Step 4:** Build and verify.

- [ ] **Step 5:** Commit: `feat: 22 new alchemy recipes replacing old system`

---

## Phase 4: Consumption Rework

### Task 10: Drink Command — Aging + Toxicity Integration

**Files:**
- Modify: `internal/usercommands/drink.go`

- [ ] **Step 1:** Before applying buffs, check toxicity: if `user.Character.Toxicity + itemSpec.Toxicity > user.Character.GetToxicityMax()`, reject with "Your body rejects the potion — too much toxicity."

- [ ] **Step 2:** Compute aging phase and potency. Get `CraftedRound` from the item, compute elapsed rounds, get aging thresholds from ItemSpec, get bottle multiplier, call `GetAgingPhase()`. Apply potency multiplier to the buff's duration.

- [ ] **Step 3:** Handle spoiled potions: if phase is `PhaseSpoiled`:
- Apply 3x toxicity
- Apply nausea debuff (buff 75)
- Roll 10% + (alchemySkill × 0.5)% chance for recipe discovery
- Skip normal buff application
- Send descriptive message: "The potion has gone bad! You retch as the foul liquid burns your throat."

- [ ] **Step 4:** For non-spoiled potions: apply toxicity, apply buff with scaled duration, send messages showing aging quality ("The potion is at its peak — you feel its full potency." vs "The potion tastes a bit stale — its effects are diminished.").

- [ ] **Step 5:** Apply CraftSkill potency scaling: multiply buff effects by `1.0 + craftSkill/100`.

- [ ] **Step 6:** Build and test.

- [ ] **Step 7:** Commit: `feat: drink command with aging potency, toxicity check, spoiled handling`

---

### Task 11: Crafting Integration — Bottle Selection + Aging Stamp

**Files:**
- Modify: `internal/hooks/NewRound_UserRoundTick.go` — stamp aging data on crafted potions
- Modify: `internal/crafting/crafting.go` — bottle handling

- [ ] **Step 1:** When an alchemy recipe completes and produces a potion, find which bottle was consumed (check the consumed ingredients for the `bottle` tag). Read the bottle's `BottleAgingMultiplier` from its spec. Store it on the output potion item (add a field or use MiscData).

- [ ] **Step 2:** The output potion's `CraftedRound` and `CraftSkill` are already stamped from Task 1. Verify the aging multiplier from the bottle is also stored.

- [ ] **Step 3:** Build and test.

- [ ] **Step 4:** Commit: `feat: bottle aging multiplier stamped on crafted potions`

---

## Phase 5: Migration + Content + Docs

### Task 12: Player Migration

**Files:**
- Modify: `internal/characters/character.go` — migration function
- Modify: `internal/users/users.go` — wire into LoadUser

- [ ] **Step 1:** Create `MigrateAlchemyPotions()` on Character. For each old potion item in Items and ComponentItems:
- Map old item ID to new item ID per the spec's migration table
- Set `CraftedRound` to current round count
- Set `CraftSkill` to 10
- Set aging to peak phase (store a flag or set CraftedRound to a value that puts it mid-peak)

- [ ] **Step 2:** Create `MigrateAlchemyRecipes()` on Character. For each old recipe in KnownRecipes:
- Add the new equivalent recipe per the spec's mapping
- Keep the old recipe knowledge (doesn't hurt)

- [ ] **Step 3:** Wire both into LoadUser alongside existing migrations.

- [ ] **Step 4:** Build and test.

- [ ] **Step 5:** Commit: `feat: one-time migration for existing potions and recipe knowledge`

---

### Task 13: Help Files + Documentation

**Files:**
- Create: 22 help templates for new recipes
- Create: help templates for new materials (bottles, moonpetal, etc.)
- Update: `CLAUDE.md` with Alchemy & Potions section
- Update: `PATCH_NOTES.md`
- Update: hints.yaml with alchemy tips

- [ ] **Step 1:** Create help files for all 22 new alchemy recipes. Follow the existing format. Include ingredients, skill threshold, station. Describe effects in words, not numbers.

- [ ] **Step 2:** Create help files for new materials: clay-flask, sealed-phial, crystalline-decanter, moonpetal, veilbloom-petal.

- [ ] **Step 3:** Create help file for toxicity: `help toxicity` explaining the system.

- [ ] **Step 4:** Update CLAUDE.md with "Alchemy & Potions" section covering aging, toxicity, CraftSkill/CraftedRound, bandolier, spoiled potions, and the auction house note.

- [ ] **Step 5:** Update PATCH_NOTES.md.

- [ ] **Step 6:** Add 3-5 alchemy tips to hints.yaml.

- [ ] **Step 7:** Commit: `docs: help files and documentation for alchemy rework`

---

### Task 14: Smoke Test

- [ ] **Step 1:** Full build: `go build ./...`

- [ ] **Step 2:** Run all tests: `go test ./...` (including help file completeness tests)

- [ ] **Step 3:** Manual verification checklist:
1. Craft a healing salve with a clay flask — verify CraftedRound and CraftSkill stamped
2. Craft same recipe with a crystalline decanter — verify different aging speed
3. Wait several rounds, examine potion — verify age description changes over time
4. Drink a potion — verify toxicity increases, buff applied
5. Drink until near toxicity max — verify threshold penalties kick in
6. Try to exceed toxicity max — verify rejection
7. Wait for toxicity to decay — verify it drops over time
8. Examine a spoiled potion at various alchemy skill levels — verify detection tiers
9. Drink a spoiled potion — verify 3x toxicity + nausea + recipe discovery roll
10. Use Distill Remnants on a spoiled potion — verify ingredient recovery
11. Equip bandolier — verify potions auto-route
12. Remove bandolier — verify potions spill to backpack
13. Verify old potions in existing player inventory are migrated correctly
14. Verify old recipe knowledge is migrated
15. Verify all new materials are obtainable (forage moonpetal at night, buy clay flask from Voss, etc.)
