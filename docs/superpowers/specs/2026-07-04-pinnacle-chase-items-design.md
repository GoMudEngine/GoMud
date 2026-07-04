# Pinnacle Chase Items — Design Spec

**Date:** 2026-07-04
**Status:** Approved design, pre-plan
**Depends on:** existing crafting/recipe system, enchantment reserve-pool
concept, mutation system (`ScourMutations`, weighted pools), worn-item buffs
(`WornBuffIds`), instance zones (Crash Site), trade layer (ferries/warehouses)

---

## 1. Purpose

Legendary best-in-slot (BIS) crafted items as the game's long-term
**gold/aspiration sink** — the top of the wealth-and-mastery pyramid. Nine
items, each gated by a questline, cross-disciplinary crafting mastery, and
ultra-rare reagents from existing endgame content.

**Calibration anchors (user-set):**

- All nine attainable by one character — no artificial cap. Cost and time are
  the gate, and they are deliberately egregious.
- Realistically only the god character (Megalomania) collects all nine; a
  determined player collects roughly half.
- Full set: ~320,000g + five craft skills at 65 + every craft skill at 50+.
- A one-lane determined player: 4-5 items, ~150-175k gold.
- Skill feasibility proven: the user has a prod character with 65+ skills
  (progression continues past the soft cap of 50; it slows sharply but never
  stops — verified in `internal/characters/progression.go` + tests).

## 2. Design Pillars

1. **Visible early, attainable late.** The artificer and her legend are
   discoverable by newbies; the items are years of play away. Aspiration
   requires visibility.
2. **Item identity = acquisition story.** Each item card includes its path.
   No stat sticks with fetch-lists bolted on.
3. **Primitives once, items as data.** ~4 small generic engine pieces; the
   nine items are YAML on top (Approach A, approved).
4. **The economy is the path.** Bulk mundane materials route demand through
   shops/ferries/warehouses; components are real crafted goods; gold fees are
   staged sinks.
5. **Personal mastery, not purchasable.** Wearer self-crafts, enforced via
   `MakerName` attribution on components at assembly.
6. **Quasi-legal flavor.** Some of these workings are not strictly
   sanctioned. Veyra operates out of the way for a reason.

## 3. The Artificer Frame

### 3.1 Veyra Coil-Tongue

A reclusive legendary artificer, sole keeper of the *convergence crafts* —
techniques fusing multiple disciplines into a single working. Non-combatant,
schedule-driven, full dialogue-tree treatment. Her dialogue carries the
quasi-legal tone: she asks few questions and answers fewer.

**Location:** an out-of-the-way workshop reachable via the Confluence (the
trade crossroads feeds her materials habit; the seclusion suits her
clientele). Final room placement chosen at plan time — either an unused
corner of an existing Confluence-adjacent zone or a 2-3 room annex off one.
No new zone.

### 3.2 Entry: "The Convergence" (intro quest)

One quest makes a player *known* to Veyra. Gate: show her a masterwork —
any item the player personally crafted at skill 50+ (checked via the item's
`MakerName` attribution + a live skill check). No hard quest prerequisites.
Players below the bar get flavor dialogue and a legend to chase.

Quest 77 (The Truth) is **not** required. Truth-knowers get variant
dialogue lines via quest flags — reward, not gate.

### 3.3 Per-item commissions

Each item is its own commission quest, identical structure:

1. **The Asking** — player asks about the item (discoverable triggers per
   dialogue SOP). Veyra names the full price: skill proof, reagent list,
   gold.
2. **The Gathering** — quest tracks the ultra-rare reagents + bulk mundane
   materials.
3. **The Proving** — supervised craft of the components at her workshop
   (existing multi-round crafting activity; each component recipe carries
   its own single-skill gate).
4. **The Forging** — gold in two stages (half at commission, half at
   completion — two sinks, partial progress still sinks). Final assembly
   performed by the player at her stations; output carries their
   `MakerName`.

**One commission active at a time** per player. Commission quests follow all
existing quest SOPs (end-token exclusion, quest/task triggers, dismissal
nodes, recovery nodes).

### 3.4 Skill gating via component recipes (no engine change)

There is no multi-skill recipe mechanic. Instead each pinnacle item's
recipe tree spans disciplines:

- **Component recipes** — each a normal single-skill recipe at minimum 50
  in its discipline (e.g. Hungering Guard = jewelcrafting 50).
- **Assembly recipe** — single-skill at minimum **65** in the item's primary
  discipline, consuming the components + ultra-rares + bulk materials.
- **Self-craft enforcement** — new recipe flag `require_own_components:
  true` on assembly recipes: every crafted-component ingredient must carry
  the assembler's `MakerName`, else the craft refuses with a message.
  (Small, targeted engine addition — much cheaper than multi-skill gating.)

## 4. Engine Primitives (new work)

### 4.1 Item proc system

New ItemSpec field:

```yaml
procs:
  - trigger: on_hit        # on_hit | on_kill | on_block | on_grapple | on_spell_hit
    chance: 25             # percent per trigger event
    cooldown_rounds: 10    # internal cooldown, 0 = none
    effect: lifesteal      # from a fixed Go effect registry
    params: { ratio: 0.35 }
```

Effect registry (Go, not scripts — these fire in per-round hot paths):
`lifesteal` (heal % of damage dealt), `steal_pool` (drain target pool →
refill wielder), `aoe_stun` (room force burst applying the existing Stunned
buff; excludes party members and non-combatants), `apply_condition` (via
existing `AddCondition`). Fired from the four existing
chokepoints: damage pipeline (on_hit / on_kill), defense resolution
(on_block), grapple resolution (on_grapple), spell resolution
(on_spell_hit). All proc messaging descriptive — no raw numbers. Global
`ItemProcsEnabled` kill switch.

### 4.2 Pool reservation on items

`reserve_health_pct` / `reserve_stamina_pct` / `reserve_conviction_pct` on
ItemSpec — while equipped, effective pool max reduced by that percentage.
Percentage, never flat (multipliers-over-flat convention). Mirrors the
enchantment system's existing `ReservePool` concept at the same pool-max
calculation sites so the two stay consistent.

### 4.3 Sentient item module

Voice definition per item: `_datafiles/world/dogmud/itemvoices/<itemid>.yaml`
— line pools keyed by event (`on_equip`, `on_kill`, `on_idle`,
`on_hunger_warning`, `on_taunt`, `on_grudge`, `on_unequip`). Engine tracks
simple per-item-instance state:

- **hunger** — rounds since the wielder's last kill (Blackrazor: escalating
  demand lines + drives the drain tick).
- **grudge** — last enemy to damage the bearer (Aegis: targeted insults).

Chatter paces like NPC conversations: cooldowns, max one line per round.
The Aegis's taunts are also mechanical — reuse the rhetoric/conviction
taunt path to pull mob aggro onto the bearer.

### 4.4 Targeted one-offs

- **Bandolier**: `preserves_contents` ItemSpec flag (aging clock frozen for
  potions inside — checked where `CalcEffectiveAgingSpeed` /
  `GetAgingPhase` read the item) + `ambient_potions` flag (while worn, each
  slotted potion's buff continuously refreshed at Peak 1.30x potency, zero
  toxicity).
- **Remort potion**: `drink.go` special case beside the Catalyst of
  Unmaking — `ScourMutations(0)` then ONE immediate grant from
  `GetWeightedPool` with a **rarity floor** (prune below rarity ~5), via
  `RollAcquisition`.
- **Necklace mutation tick**: worn-buff with slow `RoundInterval` → small
  chance per tick of a rare-biased `GrantRandomMutation` (weighted pool
  with rarity floor; conflicts/body-parts respected). Event text on grant
  ("something stirs beneath your skin...").
- **Manifestation statmod key**: `casting` exists; add a `manifestation`
  statmod key + its read site at manifestation checks (staff bonus).

Everything else already exists: `WornBuffIds` (worn buffs incl. permanent
haste), statmods (skills, `healthmax`, recovery), haste buffs, weight
reduction, bandolier capacity, drain/heal plumbing, mutation machinery.

## 5. The Nine Items

Numbers are starting values; tuned at playtest. Bills of materials are
three-tier: **ultra-rares** (new chase reagents) + **components**
(self-crafted, cross-discipline) + **bulk** (existing trade goods —
indicative lists below, matched to real `component_tag`s at plan time).
All new item/reagent IDs allocated via `id_inventory.py` at plan time.

### 5.1 The Vitalis Bandolier — belt · alchemy 65 · 35,000g

Mageblood-style. 4 potion slots (`bandolier_capacity: 4`). Contents never
age. While worn, each slotted potion is **continuously active at Peak
(1.30x) potency with zero toxicity**. Drawback: slotted potions can't be
drunk or removed without a long re-attunement (no mid-fight hot-swapping).

- Components: Reinforced Harness (tailoring 50), Preservation Runes
  (enchanting 50).
- Ultra-rares: **Chrysalis Filter-Membrane** (Core Guardian, Crash Site),
  **Still-Glass Rosette** (ultra-rare Stillwater Marsh forage).
- Bulk (indicative): cured hides ×20, silver ingots ×10, assorted alchemy
  reagents.

### 5.2 The Blackrazor — 2H weapon · blacksmithing 65 · 50,000g

Sentient obsidian-edged greatsword. `damage_multiplier` **3.5-4.0** — the
hardest-hitting weapon in the game by a wide margin (endgame numbers
account for gold-scaled instance BIS; a two-hander at this cost must be
worth it). On-hit lifesteal proc (100% chance, ~25% of damage dealt).

Costs: **25% health reservation** while wielded + the **hunger clock** — no
kill within ~50 rounds and it drains 1% max HP per round, escalating.
Sentient: kill praise, hunger-tier demands, sulks on unequip. Veyra forges
it reluctantly, after a blood-signed waiver.

- Components: Hungering Guard (jewelcrafting 50), Obsidian Edge-Resin
  (alchemy 50).
- Ultra-rares: **Void-Quenched Obsidian Core** (Warden-Prime), **the
  Sentinel's existing ~3% pinnacle material**, **Whisper of the Old White**
  (The Old White, NP Sewers).
- Bulk (indicative): iron ingots ×40, coal ×60, leather wrap ×10.

### 5.3 The Wayfarer's Bottomless Pack — back · tailoring 65 · 25,000g

99% weight reduction on contents, enormous capacity. No drawback — pure
quality-of-life; the "entry" pinnacle most players chase first.

- Components: Reinforced Frame (blacksmithing 50), Spatial Stitching
  (enchanting 50, consuming thread reclaimed via salvage 50).
- Ultra-rares: **Folded-Space Silk** (ultra-rare from sealed crates /
  caravan cargo — trade-layer tie-in), **Warden Chassis-Loom** (Crash
  Site).
- Bulk (indicative): silk bolts ×25, treated leather ×15.

### 5.4 The Aegis of Mockery — offhand shield · blacksmithing 65 · 40,000g

Best block stats in the game. Continuously insults enemies — flavor lines
AND a mechanical rhetoric-path taunt pulling mob aggro onto the bearer (the
tank item). **On-block proc: force-burst AoE stun** (~10%, 20-round
internal cooldown, 2-round stun, party-safe). Tracks a **grudge** against
the last enemy to hurt its bearer and gets personal ("Your mother smells of
elderberries, and your footwork embarrasses her further").

- Components: Voice-Amber Housing (jewelcrafting 50), Resonance Lacquer
  (alchemy 50).
- Ultra-rares: **Resonant Vox-Core** (Core Guardian), **Mockingbird Amber**
  (ultra-rare Ironwind Steppe forage).
- Bulk (indicative): steel ingots ×35, oak planks ×10.

### 5.5 The Thornwall Harness — body · tailoring 65 · 30,000g

Single body-slot item (no set mechanics). Heavy physical mitigation; when
the wearer is in a grapple (either direction), **on-grapple proc: severe
bleed** on the opponent (Strength-scaled magnitude, well above maul's).
The grappler's pinnacle.

- Components: Barbed Spike-Plates (blacksmithing 50), Anti-Corrosion Quench
  (alchemy 50).
- Ultra-rares: **Ironwood Thorn-Heart** (ultra-rare Fernway/steppe forage),
  **Scab-Chitin Plate** (new rare drop slot on an existing endgame mob —
  placement at plan time).
- Bulk (indicative): cured hides ×30, iron ingots ×25.

### 5.6 The Seething Prism — neck · jewelcrafting 65 · 40,000g

A living crystal that feeds on its wearer: **15% reservation on all three
pools**. In exchange, a slow worn-buff tick with a chance to grant a
**rare-biased mutation** (rarity-floored weighted pool). The only mutation
source outside the Chrysalis — deeply quasi-legal.

- Components: Containment Lattice (enchanting 50), Nutrient Suspension
  (alchemy 50).
- Ultra-rares: **Seed-Crystal of the Breach** (Crash Site), **Bloom-
  Saturated Geode** (ultra-rare deep-zone mining/forage).
- Bulk (indicative): gold ingots ×8, cut gems ×6.

### 5.7 The Zephyr Treads — feet · tailoring 65 · 25,000g

Large stamina-pool statmod + **permanent haste** while worn (worn-buff
equivalent of the haste potion's Peak effect). The other entry pinnacle;
no drawback, the cost is the point.

- Components: Quicksilver Soles (alchemy 50), Windlace Bindings
  (enchanting 50).
- Ultra-rares: **Stormfront Residue** (ultra-rare weather-event forage —
  weather-system tie-in), **Gale-Sinew of the Steppe** (new rare drop on an
  existing Ironwind apex predator).
- Bulk (indicative): treated leather ×20, silver thread ×12.

### 5.8 The Staff of the Hollow Choir — 2H caster weapon · enchanting 65 · 45,000g

Staff subtype, `spell_damage_multiplier` **3.5-4.0** (vs the current 1.6
staff ceiling) — truly endgame caster numbers, same reasoning as the
Blackrazor. Statmod bonuses to **spellcasting (`casting`) and
manifestation** effectiveness. Signature: **on-spell-hit proc steals
conviction** — drains target CP into the wielder (ratio tuned to sustain a
rotation, not infinite free casting).

- Components: Conductor Core (blacksmithing 50), Choir-Focus Gems
  (jewelcrafting 50).
- Ultra-rares: **Hollowed Voice-Box** (Core Guardian), **Chorus-Shard**
  (ultra-rare, Confluence region — deliberately near Veyra).
- Bulk (indicative): silver ingots ×15, cut gems ×8, hardwood stave.

### 5.9 The Phial of Second Birth — consumable · alchemy 65 · 30,000g per phial

The remort potion. Drinking it scours **all** mutations to species base,
then immediately grants **one** mutation from a pool pruned of everything
below rarity ~5. The mutation min-maxer's dice — and the system's only
**repeatable** sink: every build re-roll costs the full craft again.

- Components: Crystalline Decanter (existing item 40045), Reduction Base
  (cooking 50 — cooking's pinnacle moment).
- Ultra-rares: **Unmaking Distillate** (Core Guardian — sister reagent to
  the Catalyst of Unmaking), **First-Bloom Nectar** (ultra-rare deep-marsh
  botanical forage).
- Bulk (indicative): rare culinary + alchemy reagents.

## 6. Economics Summary

| Item | Primary 65 | Secondaries 50 | Gold |
|---|---|---|---|
| Vitalis Bandolier | alchemy | tailoring, enchanting | 35k |
| The Blackrazor | blacksmithing | jewelcrafting, alchemy | 50k |
| Wayfarer's Pack | tailoring | blacksmithing, enchanting, salvage | 25k |
| Aegis of Mockery | blacksmithing | jewelcrafting, alchemy | 40k |
| Thornwall Harness | tailoring | blacksmithing, alchemy | 30k |
| Seething Prism | jewelcrafting | enchanting, alchemy | 40k |
| Zephyr Treads | tailoring | alchemy, enchanting | 25k |
| Hollow Choir Staff | enchanting | blacksmithing, jewelcrafting | 45k |
| Phial of Second Birth | alchemy | cooking, (jewelcrafting) | 30k/use |

- Full set ≈ **320k gold** + alchemy/blacksmithing/tailoring/jewelcrafting/
  enchanting at 65 + all craft skills at 50+.
- Every craft skill appears somewhere (cooking and salvage as secondaries)
  — no mastery track is dead weight.
- Gold is staged (half at commission, half at forging).
- Bulk materials pull demand through the shop/ferry/warehouse economy.

## 7. Reagent Placement (18 new ultra-rare items + 1 existing)

| Source | Reagents |
|---|---|
| Crash Site — Core Guardian | Chrysalis Filter-Membrane, Resonant Vox-Core, Hollowed Voice-Box, Unmaking Distillate, Seed-Crystal of the Breach |
| Crash Site — Warden-Prime | Void-Quenched Obsidian Core, Warden Chassis-Loom |
| The Sentinel (Eastern Highlands) | existing ~3% pinnacle material |
| The Old White (NP Sewers) | Whisper of the Old White |
| Ultra-rare forage tables (existing zones) | Still-Glass Rosette, Mockingbird Amber, Ironwood Thorn-Heart, Bloom-Saturated Geode, First-Bloom Nectar, Chorus-Shard |
| Weather events | Stormfront Residue |
| Trade layer (sealed crates / caravans) | Folded-Space Silk |
| New rare slots on existing endgame mobs | Scab-Chitin Plate, Gale-Sinew of the Steppe |

Core Guardian carries five reagents on **separate low-chance rolls** — the
instance stays the hub without any single run feeling obligatory. Drop
rates start in the **2-5% band** (Sentinel precedent); tune at playtest.
Existing zones only — no new zones in this project.

## 8. Balance Guardrails

1. **The full-stack character is the design ceiling.** Blackrazor + Prism +
   Treads + Harness = -40% health reservation alongside the power; the
   reservations are the stacking brake. The stacked build is playtested
   explicitly (reference: the user's 65+-skill prod character).
2. **One commission at a time** at Veyra's.
3. **Kill switches**: `PinnacleItemsEnabled` (global) + `ItemProcsEnabled`,
   following the `FerriesEnabled` precedent — prod misbehavior is a config
   flip, not a rollback.
4. **Not tradeable in practice**: `require_own_components` +
   `MakerName`-bound assembly mean the sink cannot be bypassed by a resale
   market.
5. **Proc internal cooldowns** prevent degenerate loops (stun-lock chains,
   CP-steal cascading).
6. **Blackrazor/Choir are the explicit power ceiling** — future gold-scaled
   instance tuning calibrates against a wielder as the top predator.

## 9. Build Order

Primitives → items (YAML, verified via admin `give`) → reagents seeded into
existing zones → Veyra + commission questline + recipes last. Every layer
lands on something already testable.

## 10. Testing

- **Unit**: proc registry (each effect + cooldown + chance), pool
  reservation math, hunger clock, remort roll distribution (copy the
  existing `TestRerollBonus_ShiftsDistributionTowardRare` pattern),
  `require_own_components` enforcement.
- **Boot**: full data-file load, panics on schema/ID errors (pre-push SOP).
- **Harness playtests**: one feature-tester run per item covering the full
  commission loop (ask → gather → prove → forge → wield → effect
  verification); a stacked-build feel test; a Veyra dialogue/discovery run
  at low skill (visible-early check).
- Admin `give` shortcuts the grind for item-behavior testing before the
  acquisition path is wired.

## 11. Open Items (resolved at plan time)

- Veyra's exact rooms (Confluence-adjacent, out of the way).
- Real `component_tag` matches for the bulk material lists.
- ID allocation for ~19 reagents + 9 items + components + buffs + quests
  (`id_inventory.py`).
- Which existing endgame mobs carry Scab-Chitin Plate and Gale-Sinew.
- Exact drop percentages, proc numbers, reservation percentages — starting
  values above, tuned via playtest.
