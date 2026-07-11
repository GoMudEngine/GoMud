# Prod Player Migration to the Mutation Graph — Design

**Date:** 2026-07-11
**Status:** Design banked / **PARKED** — decisions locked, several knobs deliberately
left open (see §8). Blocked on content: needs the Tinkerer/Artificer crafter
cluster built first (§9) and the full cluster content authored (mutation Wave 6).
**Relates to:** [[project-mutation-graph-redesign]] (the engine + cluster content),
the `internal/migration/*.go` versioned per-user framework, `skillClusters`
(`internal/mutations/graph.go`).

---

## 1. Purpose & philosophy

Move existing prod players from the retired 41-flat-mutation system onto the new
mutation graph. Decision (user, 2026-07-11): **minimal disruption — a player
should wake up already specced to match how they've played**, as a natural
evolution, not a reset. Auto-derived from play history; **no player action
required** (no forced phial choice).

**We are NOT preserving the old power budget.** Instead we *define a new* mutation
budget (how many keystones / what rank a given investment level should have) and
then **retune the mutation-acquisition chance to match it** going forward. The
migration seeds players onto that new curve; the config tuning governs the curve
after. Those are two linked deliverables — the seed (this spec) and the
acquisition-rate retune (§8, parked).

---

## 2. The real population (data-grounded — pulled from prod 2026-07-11)

Profiled all 34 live accounts (`_archive/prod-users/`, gitignored, never
committed). **The naïve assumption — "many diverse combat builds to re-archetype"
— was wrong.** Reality:

- **~6 accounts have real investment.** Of those, the heaviest (fyttyn, Duard,
  Oriana, pruuk) are **crafter-generalists** — maxed crafting skills (alchemy,
  salvage, blacksmithing, enchanting, jewelcrafting, bartering in the 60–75
  range), each with **1 companion** and high skullduggery. Crafting maps to **no
  combat cluster**, so a skill-only classifier mis-brands them.
- **3 accounts have a clean archetype:** **meirok** → Manifester (2 companions +
  manifestation 48), **Deios** → Ethereal (deep spellbook, willpower training 88),
  **Saphira** → Ravener (unarmed 27 / weapon 18 brawler).
- **Megalomania (id 1) is the only `role: admin` account** — all skills ~100–103,
  set not earned.
- **24 accounts are test/newbie** (max skill ≤ level 3, names like `thisisatest`,
  `newbietest`, `fdasfa1231`, `aitester`). No meaningful signal.

Full per-account profile is reproducible via
`scratchpad/profile_prod_users.py` against `_archive/prod-users/`. Provisional
classification table in §4.

**Key derivation insight:** raw **skill level is already bias-corrected** — the
per-skill progression multiplier (weapon/unarmed 0.23, spellcasting 0.63,
manifestation 0.38, skullduggery 2.0, crafts 2.0–3.5) is *baked into the level
during progression*, so a heavily-spammed low-mult skill stays LOW level (Deios:
2,178 weapon swings → level 5). Therefore **classify on raw skill level, do NOT
re-multiply by the progression mult** (that double-corrects and over-amplifies
rare skills — the first pass wrongly branded 23/34 as Stalker off skullduggery×2.0).

---

## 3. Classification algorithm — threshold-gated, Generalist floor

A player is slotted into a combat/belief cluster **only if a signal clears a
threshold**; otherwise they fall to **Generalist** (the adaptable center —
Hollow Bones / Winged Flight path, no forced pole, widest drift access; a
"you kept your options open, so you keep them open" outcome, not a booby prize).

Signals, in priority order:

1. **Companions → Manifester** (strong). ≥2 companions **or** (≥1 companion AND
   manifestation skill above threshold) ⇒ Manifester. (meirok qualifies; a single
   dabbled companion alone does not — see crafter handling.)
2. **Spellbook depth + mental stat-training → Ethereal** (caster). A deep
   spellbook + high willpower training clears the caster bar (Deios).
3. **Dominant combat-cluster skill level** (raw), mapping via `skillClusters`:
   weapon→Colossus, unarmed→Ravener, ranged/skullduggery→Stalker, rhetoric→Zealot.
   Clears only if the level is a genuine standout above a floor (Saphira's unarmed
   27 clears; incidental skullduggery from a crafter does not — gated by §3.4).
4. **Crafting investment → Tinkerer/Artificer** (the new crafter cluster inside
   the Generalist path, §9). High summed crafting-skill investment with no clear
   combat standout ⇒ the crafter cluster. This is what catches fyttyn, Duard,
   Oriana, pruuk — their identity is *maker*, and skullduggery is incidental.
5. **Else → Generalist.** Covers undecided low-investment players and all 24
   test/newbie accounts.

Exact threshold values are **parked** (§8) — they'll be fit against the §4 real
accounts so each lands where it should, then sanity-checked.

---

## 4. Provisional per-account classification (to be finalized against thresholds)

| Account | Signal | → Cluster |
|---|---|---|
| meirok | 2 companions + manifestation 48 | **Manifester** |
| Deios | deep spellbook, willpower-train 88 | **Ethereal** |
| Saphira | unarmed 27 / weapon 18 | **Ravener** |
| fyttyn, Duard, Oriana, pruuk | maxed crafting, 1 companion, incidental skulldug | **Tinkerer/Artificer** (crafter cluster) |
| Megalomania | admin | **all-access** (§6) |
| 24 test/newbie + low-investment | no signal | **Generalist** |

---

## 5. Grant / re-bloom generosity

Calibration anchors (user, 2026-07-11) — **even the heaviest veterans do not get
apexes handed to them; they arrive *near* their ceiling:**

- **Duard-class** (the most-played crafter) ⇒ roughly the **lower tier of the
  crafter keystone / Generalist apex-line** (e.g. the entry into what would be
  Winged Flight's tier, or better, the crafter keystone equivalent). Real
  investment shows, but the top is still earned through play.
- **meirok** ⇒ Manifester keystones up to **just short of the apex** — a strong
  broodmaster start, apex still a goal.
- **Clear mid archetypes** (Deios, Saphira) ⇒ their cluster's entry + core at
  modest rank.
- **Generalist floor** (crafters below Duard-class, test/newbie) ⇒ the Generalist
  entry keystone(s) at rank 1; they grow through play.

Since progression is use-based, a modest seed is a *starting point they regrow*,
not a permanent nerf. Exact keystone counts + ranks per tier are **parked** (§8),
to be fit once the new budget curve is defined.

---

## 6. Admin handling — Megalomania → all-access

Gate on `role == "admin"` (Megalomania is the only one). Grant it **all clusters'
keystones / an affinity-bypass** so the admin can test the entire new system from
one character. Do **not** auto-classify it as a player, and do not leave it
stranded on retired mutations.

---

## 7. Migration mechanics (`internal/migration/0.14.0.go` — sketch)

Reuse the proven versioned per-user framework (mirror `0.13.0.go`):

1. Glob `_datafiles/world/dogmud/users/*.yaml`; skip `users.idx`.
2. For each: parse; read `role`, `character.skills`, `character.stats.*.training`,
   `character.spellbook`, `character.companions`, `character.mutations`,
   `character.mutationprogress`.
3. **Idempotency:** tag migrated users (e.g. a `mutation_migrated: true` marker or
   version stamp) so a re-run skips them. Critical — this rewrites live saves.
4. `role == "admin"` → all-access branch (§6).
5. Else classify (§3) → grant the seed (§5) into `character.mutations`, **wipe the
   retired-41 entries**, write back.
6. Log per-user (classification + grant), like the existing migrations.

**Safety (non-negotiable — this rewrites prod player data):**
- **Back up `users/` before the run** (the migration, or a pre-step, tars the dir).
- **Dry-run mode first** — log every intended classification+grant with zero
  writes, eyeball it against §4, THEN enable writes.
- Idempotent + version-stamped so a redeploy can't double-apply.
- Clean-break is intentional (retired mutations are discarded, not mapped).

---

## 8. Parked / open (resume after the crafter cluster + Wave 6 content)

- **The new mutation budget curve** — the target "at investment level X → N
  keystones totaling M ranks." Defines §5 precisely.
- **Retune `MobMutationRate` / player mutation-acquisition chance** to match that
  curve going forward (the second linked deliverable).
- **Exact §3 thresholds** and **§5 grant counts/ranks**, fit against the §4 real
  accounts.
- **Weights rebaseline** — parked with a conclusion: *the launch-tuned progression
  mults look broadly sane against real data (crafts 2.0–3.5 clearly let veterans
  reach 60–75; combat 0.23 is slow to LEVEL but that's the intended per-round
  throttle). A real rebaseline wants a per-skill effective-rate study
  (use-count ÷ level across veterans) — a balance pass, not migration work.*
- **⚠ NPC mutations — UNANSWERED, needs a decision (user-raised 2026-07-11).** Mobs
  currently use the retired-41 mutations: `tickMobMutationAcquisition` /
  `applyAcquiredMutation` (`internal/hooks/NewRound_MobRoundTick.go`) grant mob
  mutations, gated by the provisional `archetype_pull` table. This player migration
  wipes the old mutations for *players* — but says nothing about *mobs*. Two paths
  to weigh: **(a)** leave the old mutation system in place for NPCs (they keep using
  the retired-41 until we design a separate NPC migration), or **(b)** migrate NPCs
  onto the new graph too (re-curate `archetype_pull` → cluster tags, so mobs acquire
  new-graph mutations — this is already listed as a Wave 6 item). Blast radius: if
  players are on the new graph and mobs on the old, both systems must coexist
  cleanly (the engine is backward-compatible today, so (a) is *possible* — but it's
  two parallel mutation vocabularies to maintain). **Decide before the migration
  ships.**

---

## 9. Dependency & the side quest (doing now)

The classification routes crafter-generalists to a **Tinkerer/Artificer crafter
cluster that sits inside the Generalist path** — which **does not exist yet**. So
before this migration can run, we build that cluster (its own design → plan →
content). That's the immediate side quest; this migration spec is parked behind it
and behind the broader Wave 6 cluster authoring.

---

## 10. Future notes

- **Reframe player titles around the cluster matrix** (user idea, 2026-07-11).
  Current titles are derived differently; a future pass could base a player's
  title (or part of it) on their dominant cluster + depth — "Master Artificer,"
  "Broodmaster," "Winged Adept," etc. Revisit as its own feature.
- The **Weaver** ring-cluster stays the disruptor/control hybrid; the crafter
  identity is the *Generalist-path* Tinkerer/Artificer, distinct from Weaver.
