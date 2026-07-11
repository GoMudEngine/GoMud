# Non-human Attacks — Phase 4: Beast Data Audit & Polish — design

**Date:** 2026-06-09
**Status:** Approved (brainstorming) — pending spec review → writing-plans
**Predecessors:** Phase 1 (natural-attack messaging), Phase 2 (anatomy-gated human moves + hamstring + bite retirement), Phase 3 (beast moveset: rake/maul/pounce/gore/drain/throttle) — all shipped to local master.

## Problem

Phase 3 shipped the beast moveset with a CURATED profile assignment (15 mobs) and the gating data was never fully audited. Four gaps remain:
1. **Monster-humanoids over-reach:** clawed/fanged humanoids (goblin, skeleton, vampire) have a beast `natural_attack` AND `arms`, so they currently use BOTH humanoid technique moves (grapple) AND beast moves (rake/pounce) — a tool-using goblin shouldn't pounce.
2. **Profile coverage is partial:** ~60 beast mobs (rodents, raptors, reptiles, serpents, bats, mustelids, …) still fall back to the generic `default` profile instead of fighting to an archetype.
3. **Inert combatcommands:** some mob YAMLs list combat commands their anatomy now forbids (e.g. the aberration `sump_dweller`'s `bash` — no arms).
4. **Under-used moves / thematic gaps:** `gore` has one species (boar); spectral undead (wraith/spectre) don't lifedrain though it fits perfectly.

## Decisions (from brainstorming 2026-06-09)

1. **Strip beast moves from monster-humanoids** — implemented via the `hands` distinction (below), not by removing their `natural_attack` (which would lose claw/bite messaging).
2. **Add two new AI profiles:** `skirmisher` (light fanged vermin) and `serpent` (legless fanged).
3. **Content retags:** `deer` slam→gore (+`horns`); `wraith` + `spectre` → `lifedrain` (drain).
4. **Do NOT build a per-mob `natural_attack` override** (YAGNI — the one need, vampire bite, was solved by `drain`).

## Architecture

### Workstream A — the `hands` refinement (the monster-humanoid fix) [CODE]

**Key insight:** `arms` does NOT separate beasts from humanoids — **bears have `arms` (forelimbs) but no `hands`**, and should still maul. The tool-using-humanoid trait is **`hands`**. Current `body_parts`:
- Tool-using humanoids: goblin `[arms, hands, legs, …]`, skeleton `[arms, hands, legs, …]`, vampire `[arms, hands, legs, …]`, human `[arms, hands, …]`.
- Beasts: bear `[arms, legs, …]` (NO hands), feline/canine/boar/etc. (no arms, no hands).

**Rule:** the six beast natural-weapon moves — `rake`, `maul`, `pounce`, `gore`, `throttle`, **and `hamstring`** (Phase 2) — additionally require **`!HasBodyPart("hands")`** (a true beast has no tool-using hands). Apply at all three sync points each (AI `CanUse*`, `Execute*` action entry, `CommandIsReady`) + drift rows, consistent with the Phase-3 pattern.

Effect:
- goblin/skeleton/vampire (have `hands`) → no rake/maul/pounce/gore/throttle/hamstring; they keep humanoid moves (grapple/bash, gated on `arms`) and their claw/bite BASIC-attack messaging (keys off `natural_attack`, unchanged).
- bear (no `hands`) → still mauls/rakes (beast); may also grapple (has `arms`) — an acceptable hybrid for a bear.
- feline/canine/etc. (no hands) → unaffected, still beast-move-capable.
- **`drain` is EXEMPT** — it is gated on the `LifeDrain` flag, not `natural_attack`/anatomy, so armed undead (vampire, wraith, spectre) still drain. (A drain is a supernatural leech, not a natural-weapon strike.)

This is the symmetric counterpart to Phase 2's "humanoid technique moves require `arms`": humanoid moves need arms; beast natural-weapon moves need no hands. `drain` sits outside both (capability-flag gated).

### Workstream B — profile breadth [CODE + DATA]

Add two profiles to `aiProfiles` (`internal/combat/ai.go`):
- **`skirmisher`** (small fanged vermin — rats, insects): light, no heavy finishers. e.g. `hamstring 35, trip 30, kick 20, maul 10` (low maul so they don't fight like wolves).
- **`serpent`** (legless fanged — snakes, worms): `maul 35, throttle 35` (strike + constrict); NO pounce/hamstring (anatomy gates them out anyway, but don't weight them).

Species → profile mapping (assign `aiprofile:` on the mobs of each beast species that lack a specialized profile):

| Species (natural_attack, legs?) | Profile |
|---|---|
| canine(bite,Y), reptile(bite,Y), mustelid(bite,Y) | `predator` |
| feline(claws,Y), bat(claws,Y), raptor(claws,Y), arachnid(bite,Y) | `ambush_predator` |
| boar(gore,Y), bear(claws,Y), deer(gore,Y after retag) | `brute` |
| rodent(bite,Y), insectoid(bite,Y) | `skirmisher` |
| serpent(bite,N), worm(bite,N) | `serpent` |
| slime(slam, no limbs), elementals(slam) | none — default (basic attacks only; correct) |
| goblin/skeleton (humanoid) | leave as-is (humanoid profiles; the `hands` rule excludes beast moves) |

Curated but BROAD: assign across the obvious beast mobs of each species (the audit lists them). Mobs left on `default` still work (default weights the beast moves + anatomy/`hands` gating) — assignment is specialization, not correctness.

### Workstream C — correctness audit [DATA, mostly confirmation]

Produce an audit table of all ~30 beast species: `natural_attack`, `body_parts`, and a PASS/FIX verdict against these invariants:
- `hands` present on exactly the tool-using humanoids (goblin/skeleton/vampire/human/…) and ABSENT on all beasts. (This is what makes Workstream A correct.)
- A fanged/clawed species has `mouth`; a `gore` species has `horns` (already load-validated); legged moves' species have `legs`.
- No quadruped/non-humanoid beast has stray `arms` (would let it grapple). (Bears legitimately have arms — flag-and-confirm, don't auto-strip.)
- `natural_attack` is thematically apt for the species.
Fix any mismatch in the species YAML; document every change + every confirmed-correct species in the audit table (committed in the plan's notes or a short doc).

### Workstream D — inert combatcommand cleanup [DATA]

Sweep every mob YAML `combatcommands`/`angrycommands` for a move the mob's anatomy now forbids (grapple/bash on no-arms; trip/kick on no-legs; a beast move on a `hands` humanoid or wrong identity). For each: remove the dead entry (it's a silent no-op today). Known case: `sump_dweller` (aberration, `[]` body_parts) lists `bash`. Report the full list found + removed.

### Workstream E — content retags [DATA]

- **deer (7):** `natural_attack` slam→gore, add `horns` to `body_parts` (required by the gore→horns load validation), assign `brute` to deer mobs. Antlered deer now charge/gore.
- **wraith (32) + spectre (33):** add `lifedrain: true`. They have `body_parts: []` + `grapple_immune` → no humanoid/beast natural-weapon moves; `drain` (LifeDrain-gated) becomes their signature. (Optional flavor: a natural_attack for their basic touch — leave generic unless trivial.) Assign no special profile (drain comes via `combatcommands` or default weighting); ensure the wraith/spectre mobs can actually reach `drain` (add `drain` to their `combatcommands` like the vampire, OR confirm the AI path selects it — the plan decides).

## Testing

- Workstream A: `CanUse{Rake,Maul,Pounce,Gore,Throttle,Hamstring}` returns false for a hands-bearing species (goblin/skeleton/vampire) and true for a no-hands beast (bear/feline) — at all three sync points; drift rows `*_hashands`. `drain` still true for a hands-bearing LifeDrain species (vampire).
- Workstream B: a `skirmisher`-profile rat selects light moves (hamstring/trip), never maul-dominant; a `serpent`-profile snake selects maul/throttle, never pounce.
- Retags: deer (gore+horns) passes the load validation + can gore; wraith (lifedrain) can drain.
- Boot: all species + mob YAML edits load (the gore→horns validation passes for deer; no inert-command removals break a mob).
- Smoke: a goblin grapples but does NOT pounce; a bear mauls; a rat skirmishes (hamstring/trip, not maul); a snake throttles; a wraith drains.

## Hardening / validation

- Optionally extend the species load validation: a species with a beast `natural_attack` (bite/claws) that ALSO has `hands` is a likely authoring smell (a tool-user with a beast natural attack) — but this is INTENTIONAL for monster-humanoids (goblin), so make it a WARN (log), not a panic. (Decide in the plan whether even a warn is worth it.)

## Out of scope

- Per-mob `natural_attack` override (YAGNI, decided).
- Non-beast audits (rarity-tier tagging, carried-item dropchance) — separate backlog items, not this sub-project.
- New beast moves (the moveset is complete); constrict/web/venom etc. are future ideas.
- Player-facing beast moves for beast-mutated players (still a separate follow-up).

## Risks / watch-items

- **The `hands` rule must be applied at ALL three sync points for all six moves** (Phase-2/3 defense-in-depth) — a missed site lets a goblin pounce via a direct command. Drift rows pin it.
- **`drain` must NOT get the `!hands` gate** — it's the one beast move armed creatures keep. Easy to over-apply; tests pin the vampire-drains case.
- **deer→gore needs `horns` in the SAME change** or the boot panics (gore→horns validation).
- Profile assignment breadth can sprawl; the audit table bounds it (one pass per species, list what changed).
