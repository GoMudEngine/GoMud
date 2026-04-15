# Alchemy Rework — Design Spec (2026-03-30)

Complete overhaul of the alchemy system: witcher-style potions with
meaningful drawbacks, item aging lifecycle, toxicity system, skill-scaled
potency, progression potions with pool reservations, and bottle tiers.

---

## 1. Item Aging System

### New Item Fields

```go
CraftedRound   uint64  `yaml:"crafted_round,omitempty"`
CraftSkill     int     `yaml:"craft_skill,omitempty"`
```

`CraftedRound` records when the item was crafted (online round count).
`CraftSkill` records the brewer's alchemy skill at craft time.

### Aging Lifecycle

Potions age based on elapsed online rounds since `CraftedRound`. Five
phases, with thresholds defined per-recipe in the YAML:

| Phase | Potency Multiplier | Description |
|-------|-------------------|-------------|
| Fresh | 1.0x | Just brewed, base potency |
| Fermented | 1.15x | Developing complexity |
| Peak | 1.30x | Maximum potency |
| Declining | 1.30x → 0.5x (linear) | Past prime, losing strength |
| Spoiled | Harmful | Debuff or no effect |

Recipe YAML aging fields:
```yaml
aging:
  ferment_rounds: 1000
  peak_rounds: 5000
  decay_rounds: 15000
  spoil_rounds: 25000
```

### Bottle Tiers

Every potion recipe requires a bottle ingredient. The bottle type
applies an aging speed multiplier to all thresholds:

| Bottle | Cost | Aging Multiplier | Effect |
|--------|------|-----------------|--------|
| Clay Flask | 1g | 3.0x faster | Quick-use, disposable |
| Glass Vial | 3g | 1.0x (baseline) | Standard |
| Sealed Phial | 10g | 0.5x (half speed) | Mid-tier preservation |
| Crystalline Decanter | 30g | 0.25x (quarter speed) | Master-grade, long shelf life |

Higher alchemy skill at craft time also slows aging slightly:
`agingSpeed = bottleMultiplier × (1.0 - craftSkill/200)`. A skill 30
alchemist's potions age 15% slower on top of the bottle modifier.

### Skill-Based Detection

When examining a potion, the player's alchemy skill determines what
they learn about its age. No hard numbers are shown — descriptive
language only, per the project's no-raw-numbers convention.

| Alchemy Skill | Information Shown |
|---------------|-------------------|
| 0-5 | "a potion" (no age info) |
| 6-15 | Phase feel: "fresh", "this has aged nicely", "past its prime", "smells off" |
| 16-30 | Phase + direction: "still maturing", "nearing its peak", "beginning to fade" |
| 30+ | Exact phase + urgency: "at peak potency with plenty of time left", "at peak but won't last much longer", "declining — use it soon" |

### Spoiled Potions

Spoiled potions do NOT disappear. They remain in inventory and look
identical to fresh potions unless the player has alchemy skill 6+
(which shows them as "turned" in a dull color).

**Drinking a spoiled potion:**
- 3x the potion's normal toxicity hit
- Nausea debuff: -15% all stats for 30 rounds
- 10% chance (scaled by alchemy skill) to discover a new recipe —
  tasting something gone wrong teaches you about how ingredients
  interact. Base 10%, up to ~25% for master alchemists.

**Salvaging spoiled potions:**
- New recipe "Distill Remnants" (alchemy skill 5): break down a
  spoiled potion into 1-2 of its base ingredients. Not all ingredients
  recovered, but better than discarding.

### Inventory Display & Stacking

Potions have unique `CraftedRound` and `CraftSkill` values, so they
never truly stack. To avoid inventory clutter:

- **Cosmetic stacking:** Potions stack by ItemId in the display,
  showing count `(x5)`. When consumed, the oldest (closest to
  spoiling) is used first.
- **Spoiled exception:** Potions the player can detect as spoiled
  (alchemy skill 6+) display separately as `healing salve (turned)`
  in grey. Below skill 6, spoiled potions stack with fresh ones
  (the player can't tell the difference).

### Potion Bandolier (Belt Trade-off)

A potion bandolier is a Belt-slot item that auto-routes potions into
a dedicated "Potions:" inventory section (same pattern as the component
bag). Players choose between a normal belt (stats) and a bandolier
(potion organization + quick access).

| Bandolier | Capacity | Source |
|-----------|----------|--------|
| Leather Bandolier | 6 potions | Tailoring recipe (skill 10) |
| Reinforced Bandolier | 12 potions | Tailoring recipe (skill 20) |

Potions in the bandolier are consumed first when using `drink` or
`use` commands. Removing the bandolier spills potions back to
backpack (same as component bag removal).

### Auction House Note

**IMPORTANT for future implementation:** When the auction house system
is built, potion listings MUST display the aging phase and quality
regardless of the buyer's alchemy skill. This prevents experienced
players from selling spoiled potions to newcomers who can't detect
the age. The auction UI should show: potion name, craft skill tier
(descriptive, not numeric), and current aging phase.

### CraftSkill Scaling

The brewer's alchemy skill at craft time affects the potion's quality:

- Potency multiplier: `1.0 + (craftSkill / 100)` — skill 20 = 1.20x
- Duration multiplier: `1.0 + (craftSkill / 100)` — same scaling
- Aging speed reduction: `1.0 - craftSkill/200` — skill 30 = 15% slower

Final duration formula:
```
finalDuration = baseDuration × (1.0 + craftSkill/100) × agingPhaseMultiplier
```

---

## 2. Toxicity System

### New Character Fields

```go
Toxicity    float64 `yaml:"toxicity,omitempty"`
ToxicityMax float64 `yaml:"-"` // Derived: 100 + vitality/5
```

### Mechanics

- Each potion has a `toxicity` value in its item spec
- Drinking a potion adds its toxicity to the character's total
- Toxicity decays: 1 point per regen tick (every 3 rounds)
- `ToxicityMax = 100 + Vitality/5`
- Exceeding max: potion rejected, not consumed

### Toxicity Threshold Effects

| Threshold | Effect |
|-----------|--------|
| 0-50% | No penalty |
| 50-75% | Nausea: -10% all regen, -10% Perception |
| 75-90% | Sweating: -20% regen, -10% Perception, -10% Dexterity |
| 90-100% | Severe: -40% regen, -20% Perception, -10% Dexterity |

Percentages are of the stat's adjusted value, applied as a condition
that updates dynamically as toxicity changes.

### Purging Draught

Clears ALL potion effects, pool reservations, and toxicity. Then:
- Applies -15 Vitality, Willpower, Charisma for 50 rounds
- 5% chance per stat to cause 1 point of permanent de-progression
- 60 toxicity cost (prevents immediately re-drinking)
- Requires alchemy skill 35

---

## 3. Pool Regen Potions (7 recipes)

| Potion | Skill | Pools | Per-Pool Rate | Toxicity | Base Duration |
|--------|-------|-------|--------------|----------|---------------|
| Healing Salve | 0 | HP | 100% | 8 | 300 |
| Stamina Tonic | 0 | SP | 100% | 8 | 300 |
| Conviction Draught | 3 | CP | 100% | 8 | 300 |
| Warrior's Brew | 12 | HP+SP | 88% each | 14 | 400 |
| Preacher's Tincture | 12 | HP+CP | 88% each | 14 | 400 |
| Windrunner Draught | 15 | SP+CP | 88% each | 14 | 400 |
| Elixir of Renewal | 25 | HP+SP+CP | 80% each | 22 | 500 |

Durations are base — multiplied by `(1.0 + craftSkill/100)` and aging
phase. Combo potions trade per-pool rate for lower toxicity and single
buff slot.

### Ingredients

| Potion | Ingredients |
|--------|------------|
| Healing Salve | healer's root ×2, bottle |
| Stamina Tonic | bitter thistle ×2, bottle |
| Conviction Draught | healer's root, bitter thistle, bottle |
| Warrior's Brew | healer's root ×2, bitter thistle, dustwalk herb, bottle |
| Preacher's Tincture | healer's root ×2, chrysalis core, bottle |
| Windrunner Draught | bitter thistle ×2, dustwalk herb, chrysalis core, bottle |
| Elixir of Renewal | healer's root ×2, bitter thistle, dustwalk herb, chrysalis core, bottle |

---

## 4. Combat & Utility Potions (10 recipes)

| Potion | Skill | Effect | Drawback | Toxicity | Base Duration |
|--------|-------|--------|----------|----------|---------------|
| Ironhide Brew | 8 | +15% phys mitigation | -10% Dexterity | 15 | 400 |
| Mindshield Elixir | 10 | +15% magical mitigation | -10% Strength | 15 | 400 |
| Veilguard Tonic | 10 | +15% conviction mitigation | -10% Perception | 15 | 400 |
| Stone Stomach | 8 | Poison immunity | -10% Dex, -5% Charisma | 12 | 500 |
| Cat's Eye Draught | 12 | Night vision + see-hidden | -10% Strength | 18 | 500 |
| Swiftfoot Essence | 15 | Haste + 15% Dexterity | -10% Vitality | 20 | 350 |
| Berserker Elixir | 18 | +20% Str, +10% dmg mult | -15% Dex, -10% Per | 25 | 350 |
| Silver Tongue Oil | 20 | +20% Cha, +15% rhetoric dmg | -15% Str, -10% Vit | 22 | 350 |
| Battle Trance | 25 | +10% all defense scores | Reserves 15% SP | 30 | 300 |
| Purging Draught | 35 | Clear all + toxicity | De-progression risk | 60 | Instant |

### Ingredients

| Potion | Key Ingredients |
|--------|----------------|
| Ironhide Brew | ironbark shaving ×2, healer's root, bottle |
| Mindshield Elixir | chrysalis core, moonpetal, bottle |
| Veilguard Tonic | chrysalis core, bitter thistle ×2, bottle |
| Stone Stomach | serpent venom sac, bitter thistle, bottle |
| Cat's Eye Draught | moonpetal ×2, dustwalk herb, bottle |
| Swiftfoot Essence | dustwalk herb ×2, moonpetal, bottle |
| Berserker Elixir | hive fragment, serpent venom sac, chrysalis core, bottle |
| Silver Tongue Oil | moonpetal, chrysalis core, dustwalk herb, bottle |
| Battle Trance | ironbark shaving, chrysalis core, moonpetal, bottle |
| Purging Draught | veilbloom petal, chrysalis core ×2, serpent venom sac, bottle |

---

## 5. Progression Potions (4 recipes)

| Potion | Skill | Effect | Pool Reservation | Toxicity | Base Duration |
|--------|-------|--------|-----------------|----------|---------------|
| Essence of Growth | 28 | +50% stat progression | 20% HP reserved | 60 | 800 |
| Savant's Infusion | 30 | +50% skill progression | 20% SP reserved | 60 | 800 |
| Mutagen Brew | 35 | +50% mutation progression | 15% HP + 15% CP | 80 | 600 |
| Chrysalis Catalyst | 40 | +30% rare mutation chance | 20% HP + 20% CP | 90 | 1000 |

### Ingredients

| Potion | Key Ingredients |
|--------|----------------|
| Essence of Growth | moonpetal ×2, chrysalis core, healer's root ×2, bottle |
| Savant's Infusion | moonpetal ×2, chrysalis core, dustwalk herb ×2, bottle |
| Mutagen Brew | veilbloom petal, chrysalis core ×2, hive fragment, bottle |
| Chrysalis Catalyst | veilbloom petal ×2, chrysalis core ×2, hive fragment, moonpetal, bottle |

---

## 6. Material Sourcing

### Existing Materials (update only)

| Material | ID | Source | Rarity |
|----------|----|--------|--------|
| healer's root | 40004 | Forage (wilderness), Voss | Common |
| bitter thistle | 40005 | Forage (wilderness), Voss | Common |
| small vial → glass vial | 40006 | Voss vendor (rename only) | Common |
| cloth strip | 40007 | Forage, Maren vendor | Common |
| dustwalk herb | 40009 | Forage (steppe), Voss | Uncommon |
| chrysalis core | 40010 | Mob drop (mutated creatures) | Rare |
| hive fragment | 40011 | Mob drop (hive area) | Rare |

### New Materials

| Material | ID | Source | Rarity | Forage % |
|----------|----|--------|--------|----------|
| Clay Flask | 40043 | Voss vendor | Common | — |
| Sealed Phial | 40044 | Jewelcrafting recipe | Uncommon | — |
| Crystalline Decanter | 40045 | Jewelcrafting recipe | Rare | — |
| Moonpetal | 40046 | Forage (night, wilderness) | Rare | ~1% |
| Veilbloom Petal | 40047 | Forage (steppe, very rare), boss drop | Very Rare | ~0.2% |
| Serpent Venom Sac | 40048 | Mob drop (river lurker, blind stalker) | Uncommon | — |
| Ironbark Shaving | 40049 | Forage (forest zones) | Uncommon | ~5% |

### Forage Rarity Tiers

| Tier | Chance per forage |
|------|-------------------|
| Common | ~15% in appropriate biome |
| Uncommon | ~5% in appropriate biome |
| Rare | ~1% in specific biome + conditions |
| Very Rare | ~0.2% in specific biome, or boss drop |

### Bottle Sourcing

- Clay Flask + Glass Vial: Voss vendor (cheap)
- Sealed Phial: new jewelcrafting recipe (copper wire + chrysalis
  shard + glass vial)
- Crystalline Decanter: advanced jewelcrafting recipe (gemstone +
  chrysalis shard + glass vial ×2)

---

## 7. Migration & Backwards Compatibility

### Player Inventory Migration

One-time `MigrateAlchemyPotions()` on character load:

| Old Potion (ID) | New Equivalent |
|-----------------|---------------|
| Healing Poultice (30010) | Healing Salve |
| Stamina Draught (30011) | Stamina Tonic |
| Conviction Draught (30012) | Conviction Draught (updated) |
| Minor Antidote (30028) | Stone Stomach |
| Clarity Tonic (30029) | Mindshield Elixir |
| Fire Resistance (30030) | Ironhide Brew |
| Greater Healing (30031) | Elixir of Renewal |
| Berserker Elixir (30032) | Berserker Elixir (updated) |

Migrated potions get `CraftedRound` set to current round,
`CraftSkill` set to 10, and aging phase set to peak. Old buff IDs
retired but left in data for backwards compat.

### Recipe Knowledge Migration

Players who knew old recipes auto-learn the new equivalents:

| Old Recipe | New Recipe |
|-----------|-----------|
| healing-poultice | healing-salve |
| stamina-draught | stamina-tonic |
| minor-antidote | stone-stomach |
| clarity-tonic | mindshield-elixir |
| fire-resistance-draught | ironhide-brew |
| greater-healing-poultice | elixir-of-renewal |
| berserker-elixir | berserker-elixir |

### Vendor Updates

Voss stock updated: clay flask, glass vial, healer's root, bitter
thistle, dustwalk herb (existing herbs stay).

---

## Summary of All Changes

| Category | Count | Details |
|----------|-------|---------|
| New Item fields | 2 | CraftedRound, CraftSkill |
| New Character fields | 2 | Toxicity, ToxicityMax |
| New alchemy recipes | 22 | 7 regen + 10 combat/utility + 4 progression + Distill Remnants |
| New buff definitions | ~21 | One per potion (new IDs) |
| New material items | 7 | 3 bottles + 4 ingredients |
| New jewelcrafting recipes | 2 | Sealed phial, crystalline decanter |
| New tailoring recipes | 2 | Leather bandolier, reinforced bandolier |
| New bandolier items | 2 | Belt-slot potion containers (6/12 capacity) |
| Retired recipes | 8 | Old alchemy recipes replaced |
| Migration functions | 2 | Inventory + recipe knowledge |

### Files Changed (estimated)

| File | Change |
|------|--------|
| `internal/items/items.go` | CraftedRound, CraftSkill fields on Item |
| `internal/characters/character.go` | Toxicity fields, PotionItems slice, migration |
| `internal/hooks/NewRound_UserRoundTick.go` | Toxicity decay per tick |
| `internal/usercommands/drink.go` (or use.go) | Toxicity check, aging potency, spoiled behavior |
| `internal/crafting/crafting.go` | Stamp CraftedRound/CraftSkill on output |
| `internal/usercommands/look.go` | Skill-based age detection for potions |
| `internal/usercommands/inventory.go` | Potions section, cosmetic stacking, spoiled display |
| `_datafiles/world/dogmud/recipes/alchemy/` | 22 new recipe YAMLs |
| `_datafiles/world/dogmud/recipes/tailoring/` | 2 bandolier recipes |
| `_datafiles/world/dogmud/buffs/` | ~21 new buff YAMLs |
| `_datafiles/world/dogmud/items/` | 7 materials + potion items + 2 bandoliers |
| `_datafiles/world/dogmud/recipes/jewelcrafting/` | 2 bottle recipes |
| `_datafiles/world/dogmud/mobs/` | Mob drop tables for new materials |
| `_datafiles/config.yaml` | Toxicity config knobs |
| `CLAUDE.md` | Alchemy system documentation |
| Forage tables | New material entries |

### CLAUDE.md Update Required

Add an "Alchemy & Potions" section covering:
- Item aging lifecycle (phases, bottle tiers, skill-based detection)
- Toxicity system (thresholds, %-based penalties, decay)
- CraftSkill/CraftedRound fields on items
- Potion bandolier (belt trade-off)
- Spoiled potion behavior (3x toxicity, 10% recipe discovery, salvage)
- **Auction house note:** when implemented, potion listings MUST display
  aging phase and quality regardless of buyer skill to prevent scamming
