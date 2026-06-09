# Phase 4 — Beast body_parts / natural_attack correctness audit (2026-06-09)

Verification sweep of every species with a `natural_attack` (Workstream C/D of
the Phase-4 spec). Result: **data is correct — no fixes required.** (deer→gore and
wraith/spectre→lifedrain are deliberate retags handled in Tasks 4 & 5, not audit
fixes.)

## Invariants checked
1. `hands` present ONLY on tool-using (monster-)humanoids; ABSENT on all beasts —
   this is what makes the Phase-4 "beast moves require `!hands`" rule correct.
2. fanged/clawed species have `mouth`.
3. `gore` species have `horns` (load-validated).
4. No non-bear quadruped beast has stray `arms`.

## Table (species with a natural_attack)

| species | natural_attack | hands? | horns? | verdict |
|---|---|---|---|---|
| rodent | bite | no | no | OK |
| feline | claws | no | no | OK |
| insectoid | bite | no | no | OK |
| fish | bite | no | no | OK (legless fanged → serpent-profile candidate) |
| carnivorous plant | bite | no | no | OK (legless fanged → serpent-profile candidate) |
| fungal colony | slam | no | no | OK (no limbs → basic only) |
| slime | slam | no | no | OK (no limbs → basic only) |
| arachnid | bite | no | no | OK |
| worm | bite | no | no | OK (legless fanged → serpent) |
| canine | bite | no | no | OK |
| reptile | bite | no | no | OK |
| bat | claws | no | no | OK |
| mustelid | bite | no | no | OK |
| bear | claws | no | no | OK — has `arms` (forelimbs) but NO `hands`, so it stays a mauler AND may grapple (intentional hybrid) |
| Skeleton | claws | YES | no | OK — humanoid; `hands` excludes it from beast moves (claws still drive basic-attack messaging) |
| Vampire | claws | YES | no | OK — humanoid; `hands` excludes beast moves; `drain` (LifeDrain) is its special |
| goblin | claws | YES | no | OK — humanoid; `hands` excludes beast moves |
| boar | gore | no | YES | OK |
| deer | slam | no | no | RETAG → gore+horns (Task 4) |
| serpent | bite | no | no | OK (legless → serpent profile) |
| raptor | claws | no | no | OK |
| Water/Magma/Sand Elemental | slam | no | no | OK (slam, no beast moves) |
| Earth/Ice Elemental | slam | no | no | OK — have `arms` (can grapple/bash, NaturalBash) but are `slam`, so no beast moves; intentional humanoid-elementals |
| Air/Fire/Storm/Smoke Elemental | slam | no | no | OK (empty body_parts → basic slam only) |

## Notes for downstream tasks
- **fish, carnivorous plant, worm, serpent** are legless fanged → assign `serpent` profile (Task 6).
- **deer** retag is Task 4; **wraith/spectre** lifedrain is Task 5.
- No species YAML required correction; the gating data shipped accurate in Phases 1–3.
