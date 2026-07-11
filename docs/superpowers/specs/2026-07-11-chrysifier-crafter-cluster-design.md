# Chrysifier — the Crafter Cluster — Design

**Date:** 2026-07-11
**Status:** Design (brainstorm) — for review, then plan → build.
**Relates to:** [[project-mutation-graph-redesign]] (mutation graph engine + cluster
content), the prod-migration design (`2026-07-11-mutation-migration-design.md` — the
migration classifies crafter-veterans *into* this cluster, so it must exist first),
the Wave-5 companion Conviction economy (Homomunculus reserves Conviction).

---

## 1. Concept

The **Chrysifier** is the crafter's playstyle cluster — "the self-made." You turned
your craft *inward*, reshaping your own body and works through the Chrysalis:
uncanny provisioning, a body that is itself a workshop, faith that makes your
craftsmanship real, and — at the apex — the power to forge a living, boss-tier
copy of yourself. The name ties the crafter identity straight into the Chrysalis
lore and reads well as a future title ("Master Chrysifier").

**Why it exists:** the real prod population's heaviest veterans are
crafter-generalists (fyttyn, Duard, Oriana, pruuk) with no combat cluster home,
and the mutation graph's Generalist center had no drift identity. The Chrysifier
gives crafting a genuine mutation home reachable **two ways** — the migration
classifies crafters into it, and ongoing crafting play drifts toward it.

---

## 2. Structure & placement

- **A real, crafting-fed cluster in the Generalist center.** Add crafting skills
  to `skillClusters` (`internal/mutations/graph.go`):
  `blacksmithing, alchemy, tailoring, cooking, jewelcrafting, enchanting, salvage,
  foraging → chrysifier`. This is the first cluster whose drift signal is *crafting
  use*, and it resolves the "Generalist center has no drift" gap for the crafter
  identity specifically.
- **No pole** (`pole: ""`). Crafters are versatile — no Body/Belief opposition
  choke. (Faithwrought is *flavored* toward belief but the cluster stays neutral so
  a crafter keeps their gear/pool intact.)
- **Coexists with the Winged Flight line** in the center — a versatile player could
  pursue Chrysifier, flight, or both.

---

## 3. Roster (entry → core → core → apex)

Prereq spine: **Provident Hands** (entry) → **Walking Chrysalis** + **Faithwrought**
(cores) → **Homunculus** (apex, requires both cores).

### 3.1 Provident Hands — entry
Your gathering and breaking-down is uncanny. **Foraging and salvage yield more, and
your crafts consume fewer materials.** Resource mastery — the foothold that makes a
maker self-sufficient.
- *Impl surface:* new consumer hooks in foraging yield, salvage roll/yield, and
  crafting material consumption. New effect types (e.g. `forage_yield_multiplier`,
  `salvage_yield_bonus`, `craft_material_discount`) + their consumers. **P1-style
  passives but with NEW consumers in three systems** (forage / salvage / craft).

### 3.2 Walking Chrysalis — core (signature)
Your body *is* the workshop. **Craft anything, anywhere — no station, no tools
required.** The killer convenience keystone; pure self-made fantasy.
- *Impl surface:* a `flag: portable-workshop` read wherever crafting checks for a
  required station/tool (locate the station/tool gate in the craft path and bypass
  it when the flag is present).

### 3.3 Faithwrought — core (pre-apex)
Your conviction in your craft makes it real. **Crafted items you make come out
better** (higher quality / potency), **and you haul far more** (the carry-capacity
benefit lives here). The one node that tints the Chrysifier toward belief — a
maker's faith.
- *Impl surface:* a crafting-quality hook (conviction and/or a flat boost lifting
  the crafted item's `CraftSkill`/quality at craft time) + a carry-capacity boost
  (statmod or a `carry_capacity_multiplier` effect feeding `CarryCapacity()`).

### 3.4 Homunculus — apex ★
You craft a living homunculus of yourself — a **crafted companion**, built rather
than summoned. The crafter's answer to Winged Flight; the capstone Duard climbs
toward.
- **Boss-tier copy.** Not a weak clone — a terrifying you-shaped monster.
- **Stat scaling ties to craft mastery:** homunculus **statpool = (sum of the
  player's crafting-skill levels) × `HomunculusCraftScale`** (≈ 4). For a maxed
  crafter (craft-sum ≈ 500) that's ≈ 2000 statpool — roughly a boss / ~3× a normal
  character, ~3 elder golems' worth of a single body. A novice crafter's homunculus
  is modest; the more you've crafted, the mightier the self you forge.
- **Reserves a LOT of Conviction** (Wave-5 companion economy): a high fixed
  reservation (`HomunculusConvictionReserve`, a Tier-4-scale value) so a Chrysifier
  can realistically field **only the one** — "having such wonderful times, all by
  themself." Not five-elder-golem strong; one big friend.
- *Impl surface:* **P8 companion** — reuse `CompanionInfo` + the companion economy
  (`ConvictionReserve` snapshot). Two design points for the plan: (a) how the
  homunculus is *created* (a `craft homunculus` command / recipe vs. auto-manifest
  while the apex is owned + a respawn tick like a brood), and (b) the crafted-stat
  scaling path (spawn the mob then override its stat pool from craft-sum × scale).

---

## 4. Config knobs (new)
- `HomunculusCraftScale` (≈ 4.0) — statpool = craftSum × this.
- `HomunculusConvictionReserve` (high, Tier-4 scale ≈ 900–1200) — the base reserve
  (reduced by the summoner's manifestation/mutation per the Wave-5 economy; a pure
  crafter has little reduction, so it stays expensive — usually their only pet).
- Per-node magnitudes (forage/salvage/craft-discount %, craft-quality boost, carry
  multiplier) — **deferred to the Wave-6 playtest** per the content-spec convention;
  ship sensible first-pass values.

---

## 5. Dependencies / order
1. This cluster (its own plan → build).
2. Unblocks the prod migration (crafters classify here).
3. Folds into Wave 6 cluster authoring (tags/prereqs/help/balance).

---

## 6. Open questions for the plan
- Homunculus creation trigger (command/recipe vs. auto-manifest + respawn tick).
- Does the homunculus **inherit** the player's look/name ("<name>'s Homunculus")
  and any of their equipment/skills, or purely a stat-scaled boss mob?
- Whether Provident Hands' three sub-effects are one keystone (as specced) or split
  (keep as one — three small consumers, one identity).
- Exact drift weighting for the many crafting skills feeding one cluster (so a
  crafter drifts here as fast as a fighter drifts to Colossus).
