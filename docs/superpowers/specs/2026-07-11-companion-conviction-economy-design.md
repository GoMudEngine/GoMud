# Companion Conviction Economy — Design

**Date:** 2026-07-11
**Status:** Approved (design); implementation is a follow-on plan
**Supersedes:** the Wave 4c Brood-Sac respawn approach
(`docs/superpowers/plans/2026-07-11-chrysalis-content-wave4c-companions.md` — scrap/reshape).
**Relates to:** the Chrysalis mutation graph (the Manifester cluster) and the
Body↔Belief pole opposition.

---

## 1. Why

Today the number of companions a character may maintain is a hard count:
`GetMaxCompanions() = min(4, floor(manifestationSkill / 19))` (companions.go:130).
That is a free-with-skill entitlement — it creates no decision and doesn't care
how powerful the pet is. A Brood-Sac-style "always have one weak pet" bolt-on
is worse: nobody uses it once they have real summons.

**Redesign: companions RESERVE Conviction.** More powerful pets reserve more;
manifestation skill and a Manifester mutation each shave the cost. Since
Conviction also fuels spellcasting, this creates the real decision a summoner
archetype wants: **every pet you field is Conviction you can't spend on
`conviction-ward`, heals, or offense.** Pets vs. spells.

Two things make this fit the game we already have:

1. **The engine already reserves Conviction.** `GetPoolReservation("conviction",
   poolMax)` (validate.go:241) sums reservations from Chrysalis enchantments and
   pinnacle gear and subtracts them from usable Conviction. Companions become
   another contributor to that same total — reserved Conviction is already
   correctly "unavailable to cast with."
2. **It expresses the Body↔Belief pole opposition.** Conviction is the belief
   resource, and deep Body commitment already shrinks `ConvictionMax`. So a
   brute physically can't maintain a brood (their Conviction is choked), while a
   deep Manifester (belief pole, full Conviction, + the cost-reducing mutation)
   can field a menagerie *and* still cast. The companion economy becomes a
   direct expression of the belief pole.

---

## 2. Core model

- Each **active companion reserves Conviction**. The reservation persists while
  the companion lives and frees on its death/dismissal.
- **Reservation = `round(base × (1 − reduction))`**, floored at a minimum
  fraction of base.
- **`base`** is authored **per summon type** (§3) — a fixed `conviction_reserve`
  on the summon/raise spell — **not** derived from the summoned creature's actual
  stat pool. This sidesteps the undead problem entirely: a golem raised from a
  huge corpse is stronger but costs the *same* as any golem, so the raised pool
  never affects price. (Nice side effect: raising from beefier corpses is a pure
  power gain for the same Conviction — a summoner's corpse-hunting reward.)
- **`reduction`** comes from manifestation skill + the Manifester mutation,
  capped (§4).
- Companions' total reservation is added into `GetPoolReservation("conviction",
  …)`. **The Conviction budget is the real limit:** a summon is refused if it
  would push total reservation past `ConvictionMax` (leaving a small casting
  floor). A generous **soft count ceiling** (§4) is only a backstop.

This replaces the `min(4, skill/19)` count formula.

---

## 3. Companion cost tiers

Each **summon type** carries its own fixed `conviction_reserve` (a field on its
summon/raise spell) — the cost is tied to *what you summoned*, not how powerful
the resulting creature happens to be. The corpse a golem is raised from changes
the golem's *power* but never its *cost*. The tier labels below are just
groupings of those per-type numbers.

| Tier | Summon types (each authored individually) | `conviction_reserve` (first-pass) |
|---|---|---|
| **Lesser** | Spirit wolf, conjured imp | **~350** |
| **Greater** | Magma elemental, standard golem | **~440** |
| **Elder** | Elite golem / high-end summon | **~735** |
| **Tier 4+ (RESERVED — speculative, §10)** | Support-caster ghost, lich, undead dragon | 900–1500+ (TBD) |

> The premium between tiers is intentionally **modest, not power-proportional**:
> a golem is ~6× a wolf's stat pool but only ~1.3× its cost, because the anchor
> targets (§5: newbie's wolf ≈75%, Meirok's 2 golems) mathematically require the
> dominant cost to be "supporting another body at all," with tier a small
> premium on top. A summoner's payoff is fielding *beefier* bodies at a
> *discount* (skill + mutation), plus raising from fat corpses — not a
> cheaper-per-power rate. Base numbers are the **tuning surface**; §5 fixes them,
> final values land in playtest.

---

## 4. Reduction & ceiling

```
manifReduction = min(0.55, manifestationSkill × 0.01)       # skill: up to 55% off, cap reached at manifestation 55
mutReduction   = min(0.24, manifesterMutationRank × 0.06)   # mutation: up to 24% off (rank 4)
reduction      = min(0.79, manifReduction + mutReduction)   # combined cap 79% (cost floors at 21% of base)
```

- **Skill caps at 55% off** — a maxed manifestation skill nearly halves cost but
  can't go further, so the **mutation is what pushes past it** (the last ~24%),
  which is exactly the "fully-archetyped unlocks the extra golems" fantasy.
- **Soft count ceiling** (backstop, not the primary limit): **5** base, raised to
  **7** by the Manifester apex. Prevents absurd swarms if the Conviction math is
  ever mis-tuned.

Config knobs (Balance): `CompanionReserveSkillPct` (0.01),
`CompanionReserveSkillCap` (0.55), `CompanionReserveMutPctPerRank` (0.06),
`CompanionReserveMutCap` (0.24), `CompanionReserveTotalCap` (0.79),
`CompanionSoftCap` (5), plus per-tier base costs in data.

---

## 5. Calibration table (real characters, first-pass knobs)

`ConvictionMax = 5 + (Wil + Cha)ᵥₐₗᵤₑ𝒶𝒹ⱼ × 2` (validate.go:96). Stats are
base+training (dashboard values). Reduction per §4; base costs per §3.

| Character | Manif | Mut | Wil / Cha | ConvMax | Reduction | Loadout | Reserved | % / left |
|---|---|---|---|---|---|---|---|---|
| **Newbie** (post-quest) | 5 | — | ~108 / ~108 | ~440 | 5% | 1 Spirit Wolf (350→333) | 333 | **76% / 24%** |
| **Martial dabbler** (Saphira) | 1 | — | 89 / 115 | ~413 | 1% | 1 Spirit Wolf (350→346) | 346 | **84% / 16%** |
| **Meirok** (geared, no companion mut) | 48 | — | 148 / 123 | ~547 | 48% | 2 Greater golems (440→229) | 458 | **84% / 16%** (3rd → 126%) |
| **Fully-archetyped** | 55 | r4 | ~135 / ~135 | ~600 | 79% | 5 Greater (440→92) | 462 | **77% / 23%** |
| " (alt loadout) | 55 | r4 | — | ~600 | 79% | 3 Elder (735→154) | 462 | **77% / 23%** |
| **Absolute unit** (grind + gear) | 65 | r4 | ~150+gear | ~850 | 79% | 5 Elder (735→154) | 770 | **91% / 9%** |

**Reading the table:**
- **Newbie** runs exactly one spirit wolf and keeps a sliver of Conviction — one
  self-buff or the odd bolt. A second wolf would blow the budget. ✓
- **Martial dabbler** *can* run a companion, but one pet consumes ~84% of their
  Conviction — they get a pet **or** their spells, not both. This is the honest
  cost of dabbling. And if that character is **deep Body pole**, the opposition
  has already shrunk their `ConvictionMax`, so a single wolf may not fit at all —
  companions are a belief-pole fantasy, enforced by the resource, not a rule.
- **Meirok** (no companion mutation yet) fields his 2 boss-golems at ~84% —
  near his ceiling, with a third well out of reach. That's the design working:
  a strong summoner *without* the companion mutation tops out at 2; the mutation
  (up to 24% off) is exactly what unlocks the 3rd+. Matches his live count today
  (`floor(48/19)=2`).
- **Fully-archetyped** picks: 3 Elder or 5 Greater, either way ~77% reserved,
  ~23% left for `conviction-ward` + a heal/offense — the mutation's ~24% off is
  what lets 3 golems become 5, or lets the Elders come out.
- **Absolute unit** stacks the beefiest Elders to the brim (~91%), casting only
  in emergencies — the "did nothing but summon" payoff.

> The rows are internally consistent under one set of knobs; the exact base
> costs / percentages are refined in playtest until every row reads true.

---

## 6. Manifester mutation remap

The Manifester cluster's companion mutations now key off this economy (this
supersedes Brood-Sac-as-respawn):

- **Core companion mutation** (the identity keystone): **reduces companion
  reservation cost per rank** (`CompanionReserveMutPctPerRank`, up to 24% at
  rank 4). This is what turns a competent summoner into a *broodmaster*.
- **Hive Mind**: raises the **soft count ceiling** (+ companion coordination — a
  later combat-behavior pass).
- **Brood Mother (apex):** max reservation reduction + the raised ceiling (7) +
  (optional) a rank-1 **floor companion** so a deep Manifester is never petless.
- **Symbiotic Bond:** buffs/regen bleed to companions — a separate passive, not
  part of the economy.

Full per-cluster authoring (effects, ranks, prereq spines) lands in the mutation
Wave 6 content pass, against the effect types this spec defines.

---

## 7. Integration

- **`GetMaxCompanions` → budget check.** Replace the count formula's role: the
  summon path (`resolveCompanionSummon`, companion_summon.go; and the btree
  `actSummonCompanion`) computes the pet's tier reservation and refuses if
  `currentConvictionReservation + petReservation > ConvictionMax − castingFloor`.
  Keep `CompanionSoftCap` as a hard backstop.
- **`GetPoolReservation("conviction", …)` extension.** Add a companion term: sum
  each live companion's stored reservation (persist the computed reservation on
  `CompanionInfo` at summon time so it doesn't drift when skill changes mid-fight).
- **`CompanionInfo` gains `ConvictionReserve int`** (snapshotted at summon).
- **Death/dismiss** frees the reservation automatically — `MobDeath_CompanionCleanup`
  already removes the companion; `GetPoolReservation` simply stops counting it.
- **Undead:** the raise spell carries its own `conviction_reserve` like any
  summon — no corpse-value math. The corpse only sets the raised creature's
  power (existing behavior), never its cost.

---

## 8. Migration / live impact

Modest. Per the prod dashboard, **only Meirok invests in manifestation** (48;
everyone else 0–1), so the live blast radius is one character — and the new model
reproduces his current count (~2 boss-golems). No wipe needed; the change is to
the summon gate + reservation sum. Flag in patch notes that manifestation now
*costs Conviction to field* (a buff for dedicated summoners, a real cost for
dabblers). No existing player loses companions they can still afford.

---

## 9. Out of scope / follow-on

- **Implementation plan** (its own doc): extend `GetPoolReservation`, add the
  budget gate + tier brackets + `CompanionInfo.ConvictionReserve`, the config
  knobs, and the calibration table as tests; author the Manifester mutation
  effect type (reservation reducer). Reuses the Wave 4c `spawnCharmedCompanion`
  extraction idea.
- **Balance finalization** in the post-build playtest (the §5 numbers).
- **Companion combat behavior** (Hive Mind coordination, Symbiotic Bond buffs).

## 10. Speculative future — higher companion tiers (note for later)

Leave the tier system **open-ended** so stronger companions slot in without
rework. Reserved design space (costs/how-to-obtain **TBD, do not build now**):

- **Support-caster ghost** — a healer/warder pet that casts *for* you (reserves
  a lot; changes the calculus from "more bodies" to "a second caster").
- **Lich** — a powerful undead spellcaster companion (elite content unlock).
- **Undead dragon** — a capstone single companion so expensive it *is* your whole
  Conviction budget (one dragon ≈ your entire brood).

These would sit at **Tier 4+** (base ~900–1500+), gated behind rare content
(late crash-site / boss corpses), and are exactly why the cost model is tiered +
budget-based rather than a hard count: a dragon that eats 100% of your Conviction
is a coherent build, not a special case. **Speculative only — revisit when the
Tier 1–3 economy is live and tuned.**

### 10b. Rejected (for now): per-pool reservation

Considered making companion *type* determine which pool it reserves — undead draw
**Health**, elementals draw **Conviction** — for a glass-cannon risk/reward feel.
**Rejected** because it breaks the load-bearing part of this design: a Body-pole
brute has *high* Health and *choked* Conviction, so undead-reserve-Health would
let the brute field an army *because* they went Body — inverting the pole
opposition the whole model expresses. It also forces a hard count cap back in
(two pools can't share one budget), and makes companions compete with gear
enchantments across both pools.

If the fragile-necromancer fantasy is wanted later, the safe form is a **deep
necromancer sub-branch mutation** that opts *part* of an undead's reservation
onto Health — a deliberate, earned, single-path choice — **not** a free unlock
for any low-Conviction martial. Note-for-later, do not build now.
