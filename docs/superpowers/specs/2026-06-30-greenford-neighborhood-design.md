# Greenford — District 4: Brennan's & Reth's Neighborhood — Design

*Spec date: 2026-06-30. District 4 of 5 (city-wide layer
`2026-06-30-greenford-citywide-design.md`; D1 `6b162857`, D2 `b8b14bcb`, D3
University in flight). **The Q75 payoff district — the most narratively loaded.**
The residential streets behind the college, where Brennan and the retired
surveyor **Reth** live. Q75 The Surveyor's Report COMPLETES here: with Brennan's
introduction, Reth gives his testimony — directions east + "it's not natural" —
and the marked map (40160). The back half of the split quest declared in D3.
**Build AFTER D3 merges** (D4 attaches at D3's 6308 + wires Q75's forward steps).*

## Role

Through the college's back gate: a quiet residential quarter of the upper town —
faculty houses, retired folk, a tea house, walled gardens. Two homes matter:
**Brennan's house (the blue door)** — examinable lore, a window into the scholar
who set the quest — and **Reth's cottage** at the far/quiet end, sparse and
orderly, where the Eastern Arc's directions finally come out. The texture is
lived-in and gentle; the weight is Reth.

- **Folder:** `greenford`. **Rooms:** 6309–6316 (8). **Mobs/dialogue:**
  9524–9529 (6). **Items:** 40160 (Reth's map — the Q75 payoff) + optional tea good.
- **Faction:** Reth is `[humanoid]` (retired, apart — not formally Margin); the
  Q75 completion awards **`margin` rep** (you carried the truth into the Margin's
  orbit). Residents `[humanoid]`.

## Geography & Seam

- **Seam:** D3's **6308 "The College Lane"** `{x:22,y:-83,z:2}` (the back-gate
  stub; exits `north→6307, west→6306`, no onward; has a `back gate` noun "beyond
  it the residential streets"). Add **`6308 south → 6309`** (out the back gate
  into the streets) and revise 6308's prose so the gate now leads onward (open).
- **Layout:** the residential quarter on **z=2**, SOUTH of the college grounds
  (y ≤ −84; clear of D2's z=1 footprint and D3's y=−80…−83). Builder finalizes a
  clean reciprocal, collision-free graph + cartcheck.

| Room | Title | role |
|------|-------|------|
| 6309 | Back Gate Lane | from 6308; the streets begin |
| 6310 | College Row | a residential street (faculty houses) |
| 6311 | The Blue Door | **Brennan's house** — examinable lore (no NPC; he's at the college by day) |
| 6312 | The Tea House | a small social spot (a tea-house keeper, cooking vendor) |
| 6313 | The Walled Garden | a green / quiet court |
| 6314 | The Quiet End | the lane toward Reth's, residents thin out |
| 6315 | Reth's Cottage | **Q75 PAYOFF — Reth + the testimony** (sparse, orderly) |
| 6316 | The Garden Wall | a quiet terminus (the neighborhood's far edge) |

(6309 entry-hub; the homes/tea house off the row; 6314→6315 the quiet approach to
Reth; 6316 a soft terminus — NOT a stub to D5 [D5 West Outskirts attaches off the
town/university west edge, its own spec]. No onward stub needed unless the
builder finds a natural one — keep it a closed, lived-in quarter.)

**Geography note (carry into the build):** Brennan (D3) says Reth is "the north
end of the town" / the quiet residential streets. Frame D4 as **the residential
quarter behind the college**, with Reth's cottage at its far/quiet end — keep the
in-game directions ("through the back gate, the quiet end of the streets") so the
player follows the hint, not a strict compass. If Brennan's D3 line reads
awkwardly against the south-of-college placement, lightly reword it during the D4
build to "the residential streets behind the college, the quiet end."

## NPCs (mobs 9524–9529: 6)

Canonical Title-Case names, `ConvertForFilename` filenames, ambient
`noncombat_passive`, unique mutations (cross-check vs the WHOLE Greenford roster —
D1+D2+D3 = 21 mutations so far), ≥3 topics, idle behaviors, voice rules (**every
hint word routes**; **`|` literal blocks for ALL long NPC text**), NO undeclared
quest fields.

| mob | name | room | role |
|-----|------|------|------|
| 9524 | **Reth** (named) | 6315 | **Q75 PAYOFF.** Retired surveyor, `[humanoid]`. Reluctant/unsettled; turns away cold callers, but **with Brennan's introduction (`75-intro`)** relents and gives the testimony: directions + landmarks east (the lightning-split cairn, the route into the highlands) + **"it's not natural"** (he saw the exposed thing — metal, no seam, not landscape) — but **NEVER what it is or what's inside** (threshold-only; the revelation is the crash-site zone #22). Hands you his **marked field-map (40160)**. |
| 9525 | (named) The Tea-House Keeper | 6312 | **cooking vendor**; neighborhood gossip, who lives where, Reth "keeps to himself," Brennan "always at his maps" (reinforces the thread) |
| 9526 | A Neighbor | 6310 | dialogue; residential color; the college, the quiet streets |
| 9527 | A Gardener | 6313 | ambient; the walled garden, the season |
| 9528 | A Retired Functionary | 6314 | dialogue; an old resident near Reth's end — knew Reth "before he went quiet" (light lore, no answers) |
| 9529 | A Resident | 6309/6310 | ambient; daily-life color |

Reth is the one heavy NPC — reluctant, careful, threshold-only. Everyone else is
warm residential texture (the tea-house keeper gently points the player toward
Reth/Brennan).

## Quest 75 — COMPLETION (wire the back half: testimony + end)

**Reth gates on `75-intro`** (Brennan's introduction). The 3 city-wide
resolution paths converge here — primarily **Brennan's intro** (the through-line
D3 delivers); Reth may also relent for a player who found his field notes
(research path) — but keep the core gate `75-intro`.

**Mechanism (resolve against Q74's reveal+completion at build):**
- **Testimony delivery** — a `room_interact` in Reth's cottage (6315) on a
  hyphenated noun (e.g. **`field-notes`** / **`survey-map`**), gated
  `has:['75-intro'] missing:['75-testimony']` → `grant: 75-testimony` +
  `give_item: 40160` (the marked map) + `send_text` (the directions + landmarks +
  "it's not natural", threshold-only). Reth's DIALOGUE (gated `75-intro`) sets it
  up ("look at my old field-map, there, on the table"). Ungated fallback = a
  retired man's papers, nothing.
- **Completion** — grant `75-end` + `bump_rep {faction: margin, +N}` on the
  completion. Resolve the exact carrier at build (one trigger doing
  `grant 75-testimony` + `give_item` + then a Reth dialogue node
  `questRequired:['75-testimony'] grantsQuest:'75-end'` for the parting words;
  OR a single room_interact granting through `75-end` with `bump_rep` if the
  engine allows the chain). **Rewards fire on `75-end`** (the quest's
  `rewards.playermessage` onward beat — already declared in D3's quest YAML).
  Margin rep via `bump_rep` in the completing trigger (Q74 pattern).
- **Quest-build SOPs:** `room_interact` noun = ansi-highlighted HYPHENATED token
  + matching key; a trigger may only `grant` a DECLARED step (75-testimony,
  75-end both declared in D3); Reth's gated dialogue nodes FIRST; `questRequired`
  is a LIST. The **40160 item is `not_salable`** (reward item).

**Mystery boundary (LOCKED):** Reth gives DIRECTIONS + "it's not natural" only.
NEVER what it is / inside / the symbol's meaning / numerology. The revelation is
the crash-site interior (#22). The thread points "east, into the broken highland
country" (the reward onward beat).

## Items

- **40160** — Reth's marked field-map / survey notes. `not_salable`. The Q75
  payoff (crash-site directions, a tangible takeaway). Created here.
- Optional **40161** — a tea-house cooking good (`cooking`) if the tea house
  vendors. Reuse existing goods otherwise.

## Brennan's House (6311 — lore, no NPC)

The **blue door** (the city-wide detail). Examinable: maps visible through the
window, the clutter of a scholar who lives his work, a sense of the man behind
the quest. NO NPC (Brennan's at the college by day). A quiet character beat — and
if the city-wide "Empty Cottage"-style hidden-item idea is wanted, a small
examinable detail here can reward the observant (optional; keep mundane).

## Supporting quest (OPTIONAL — user's call)

The city-wide allowed "1–2 supporting district quests" total; none built in
D1–D3. **Default: D4 has NO supporting quest** — Q75 completion + the rich
residential lore (Brennan's house, Reth's reluctance, the tea-house gossip) carry
it. A small optional one could be added (e.g. a neighbor's errand that eases
Reth's trust, or a "what happened to Reth" lore-gather) — **flag for the user; cut
by default to keep D4 focused on the payoff.** Any supporting quest = Q76.

## Build conventions & validation

Full Greenford convention/gotcha list + quest SOPs (as D3). Per-district SOP:
`id_inventory` → author → wipe instances → clean boot (`ValidateZoneConsistency
errors=0 mode=panic`, **no "trigger grants unknown step"**, ValidateAllFlags OK) →
`cartcheck greenford` → **world-critic + harness feel-test (Q75 END-TO-END,
across D3+D4):** start at Brennan (D3) → archive survey → intro → **go to Reth
(D4)** → testimony room_interact → get the map (40160) → **Q75 COMPLETES**
(rewards + margin rep) → confirm `quests` shows it done + the onward beat. Use
`questtoken`/`questtoken flags` for reliable checks (harness flaky on multi-step).
Update `docs/ZONE_EXPANSION.md` row 19 (4/5) + memory → merge `--no-ff`.

## Out of scope

D5 West Outskirts (the West Road / NP-loop stub — its own spec, attaches off the
town/university west edge, not D4); any crash-site/symbol EXPLANATION (#22).
