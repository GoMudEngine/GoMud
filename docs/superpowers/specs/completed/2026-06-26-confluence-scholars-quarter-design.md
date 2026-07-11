# The Confluence — District 4: The Scholars' Quarter (+ Quest 73) — Design

**Date:** 2026-06-26
**Status:** Approved (design phase)
**Umbrella:** `docs/superpowers/specs/2026-06-26-confluence-citywide-design.md` (district 4 of 10 — build order step 3)
**Predecessor:** District 3 Tri-Cross Square, merged — the **Hall of Records (6149)** is the seam this district opens.

## Purpose

The **Margin's home** — the quiet scholarly quarter behind the civic archive,
and **the Confluence's first quest district**. Everything seeded across the Long
Quay and the Square (Tallis's charts, the notation map, Savel's thread) pays off
here in **Quest 73 — The Margin Notation**: the player gathers the disagreeing
map records, the Margin synthesizes that there *was* a fourth water struck from
the record, and the thread points onward to the temple's oldest stonework (the
Q74 hook). Threshold-only — the *why* stays for the crash site.

## Scope & IDs

- **14 rooms, 6218–6231** (the umbrella's reserved block for the Scholars' Quarter).
- **Mobs/dialogue 9441–9448** (8).
- **Items 40137–40138** (a bookseller good + the Q73 reward item).
- **Quest 73** (next free; declared with steps + flags if needed).
- **Zone:** `The Confluence` (existing folder; no new zone-config).
- **Biome:** `city` throughout.
- **Factions:** most scholars are **`margin`** (the existing faction); the
  Bookseller + ambient students may be unfactioned `[humanoid]`.

## Seam — open the Hall of Records (6149)

Edit `_datafiles/world/dogmud/rooms/the_confluence/6149.yaml`:
- Add `west: {roomid: 6218}` to `exits:` (keep `east: 6143`).
- Lightly revise the description so a doorway/passage leads **west** from the
  records room into the Scholars' Quarter (the civic archive backs onto the
  scholars' own halls) — Savel already points the player "up to their quarter."

> **Build-time note:** 6149 is a district-3 file the background world-critic is
> currently reading. Make this edit only after the testers report (or confirm
> they're done). Re-`cartcheck` after.

## Layout — 14 rooms

A central **E–W court** (the spine) running west from the Hall of Records, with
study rooms north and south, the Archive at the west end. Proposed grid — **build
assigns final coords + `cartcheck`s** against the Square and all prior. Exits
reciprocal as drawn.

| Room | Title | Coord | Mob | Role |
|------|-------|-------|-----|------|
| 6218 | The Scholars' Gate | {-8,-67,0} | 9447 | Entry (east→6149 Hall of Records); a porter/archivist; the quarter begins |
| 6219 | Inkwell Court | {-9,-67,0} | 9448 | The central scholarly court; an ambient student; benches, a dry fountain |
| 6220 | The Margin Hall | {-10,-67,0} | 9441 | The Margin's meeting hall; **the Q73 giver** (senior Margin scholar) |
| 6221 | The Great Archive | {-11,-67,0} | 9442 | The library / the Margin's source surveys; the Archivist; **Q73 synthesis happens here** |
| 6230 | The Deep Stacks | {-12,-67,0} | — | The oldest surveys, west end; **a locked/restricted case** (`look case` noun — foreshadows the temple's sealed records; no access) |
| 6222 | A Study Hall | {-8,-66,0} | — | Reading desks, a scholar's clutter (north) |
| 6223 | The Cartographers' Room | {-9,-66,0} | 9443 | A map study; a cartographer-scholar; instruments, draughting tables (north) |
| 6224 | A Lecture Room | {-10,-66,0} | — | Tiered benches, a slate (north) |
| 6225 | The Reading Room | {-11,-66,0} | — | Quiet stacks, locked reference cases (north) |
| 6226 | The Bookseller's | {-8,-68,0} | 9444 | The **Bookseller** (book vendor) + **the damaged chart** (a Q73 source — `look chart` noun) (south) |
| 6227 | Scholars' Lodgings | {-9,-68,0} | 9445 | Where the quarter's scholars sleep and argue; a scholar (south) |
| 6228 | The Copyists' Room | {-10,-68,0} | 9446 | Copy-work (ties to Tallis's trade); a copyist (south) |
| 6229 | The Quiet Garden | {-11,-68,0} | — | A contemplative court; a fig tree, a worn bench (south) |
| 6231 | The Garden Walk | {-12,-68,0} | — | A secluded study walk off the garden (a `look` orbital-mark seed on an old garden stone — restrained) |

Exit skeleton (build finalizes + cartchecks):
```
6218 e->6149(Hall of Records, the new seam) w->6219 n->6222 s->6226
6219 e->6218 w->6220 n->6223 s->6227
6220 e->6219 w->6221 n->6224 s->6228
6221 e->6220 w->6230 n->6225 s->6229
6230 e->6221
6222 s->6218
6223 s->6219
6224 s->6220
6225 s->6221
6226 n->6218
6227 n->6219
6228 n->6220
6229 n->6221 s->6231
6231 n->6229
```

## NPCs (9441–9448)

All `non_combatant: true`, `hostile: false`, `charm_immune: true`,
`speciesid: 1`, `level: 1`, `maxwander: 0`, `statpool ~30`. Non-vendors
`behavior_archetype: noncombat_passive`. **Unique names, Title Case.**
`groups: [humanoid, margin]` for the scholars; `[humanoid]` for the Bookseller +
ambient student.

| Mob | Role | Room | Notes |
|-----|------|------|-------|
| 9441 | The Q73 Giver — a senior Margin scholar | 6220 | An old cartographer-historian who has chased the four-waters question for decades (failing eyesight from a life of faded charts). **Grants Q73**; receives the evidence; delivers the synthesis + the temple/undercroft hook. `margin`. |
| 9442 | The Archivist | 6221 | Keeps the Margin's source surveys; helps lay the evidence side by side at the synthesis. `margin`. |
| 9443 | A Cartographer-Scholar | 6223 | Map-work; river-survey lore (canon directions). `margin`. |
| 9444 | The Bookseller | 6226 | **Vendor** (book good 40137); owns **the damaged chart** (Q73 source). `[humanoid]` (a tradesperson, Margin-sympathetic but not a member). |
| 9445 | A Margin Scholar | 6227 | The quarter's life; the open question, the temple's official account. `margin`. |
| 9446 | A Copyist | 6228 | Copy-work, the chart trade (a peer of Tallis). `margin`. |
| 9447 | The Gate-Porter | 6218 | Keeps the quarter's gate; directs visitors (ambient/short). `[humanoid]` or `margin`. |
| 9448 | A Student | 6219 | Ambient (idlecommands; the life of the court). `[humanoid]`. |

## Quest 73 — The Margin Notation

A **linear lore investigation** (no branch — the player's introduction to the
mystery; the allegiance choice is reserved for Q74). Giver: **9441** (Margin Hall).

**Steps:**
1. **start** — the giver frames the four-waters question and asks the player to
   examine the three independent records that disagree, then return.
2. **chart** — examined **the Bookseller's damaged chart** (6226, `look chart`
   room_interact, gated by the quest token).
3. **scrivener** — examined **Tallis's charts** (Long Quay **6131**, `look
   charts` room_interact). *(cross-district wiring)*
4. **map** — examined **the municipal notation map** (Tri-Cross Square **6143**,
   `look <map-noun>` room_interact). *(cross-district wiring)*
5. **synthesize** — returned to the giver/Archive; the three laid side by side.
6. **end** — the giver's synthesis: *there was a fourth, struck from the record
   over generations; the temple sits on what it suppressed.* Points the player to
   the **temple's oldest stonework / the undercroft** (the Q74 hook — actionable
   when the temple districts build). 

> Step order puts the in-quarter source (the bookseller) first, then the two
> the player has already walked past (Tallis, the hall map) — `hint` text guides
> each. If the engine supports an unordered "examine all three" gather step,
> that's acceptable too; otherwise sequential steps as above.

**Rewards** (reward-block keys are NO-underscore — `playermessage`/`gold`/`itemid`/
`rep_faction`/`rep_amount`): gold (~30), **`rep_faction: margin`** (+15), and
**`itemid: 40138`** — *A Fair Copy of the Compiled Survey* (a lore item the player
carries; threads toward Q74). A `playermessage` delivering the threshold beat.

**Quest gotchas (SOP — carry into the build):**
- The **giver's grant dialogue node** needs `grantsQuest: "73-start"`,
  `questExcluded: ["73-start", "73-end"]` (BOTH tokens), and `"quest"` + `"task"`
  in its `triggers`/`keywords` (so `ask <giver> quest` works).
- **room_interact** fires on `look <highlighted-noun>` (the exact ansi-highlighted
  hyphenated noun) and is **gated** by the quest token — without Q73 active those
  `look`s just show the existing lore noun (no regression). At build, **verify the
  source nouns are ansi-highlighted** (6226 chart, 6131 charts, 6143 map); if a
  seeded noun isn't highlighted, highlight it (a small edit to that room).
- A quest trigger may only **grant a DECLARED step token** — declare every step
  id used; final completion grants the `end` step.
- Item **40138** (40xxx) lives in `items/materials-40000/` regardless of type.
- If any quest **flag** is used, declare it in the quest YAML (undeclared flag →
  panic). This quest is linear; likely **no flags** needed.

## Cross-district wiring (built here, edits earlier districts)

Done at build time (after the background testers finish reading those files):
- **6131 (Long Quay, Tallis's stall):** ensure `the charts` is an ansi-highlighted
  noun; the Q73 `scrivener` step room_interacts on `look charts` here.
- **6143 (Tri-Cross Square, Municipal Hall):** ensure the map noun is highlighted;
  the Q73 `map` step room_interacts on `look <map-noun>` here.
- **Savel (9435) / Tallis (9431) dialogue:** optionally add a line acknowledging
  the Margin's work once Q73 is active (nice-to-have, not required).
- No changes to those rooms' non-quest behavior — the triggers are quest-gated.

## Mystery — the Margin's home (threshold-only)

The most *overt* investigation site, but still bounded: the Archive's open
question, the source surveys, the synthesis that there *was* a fourth. The
**Deep Stacks' locked case (6230)** and a restrained **garden-stone orbital mark
(6231)** keep the motif present. The *why* (crash, mutation link) is **never
stated** — the giver explicitly points onward (the temple, then beyond). Keep the
Margin scholarly and restrained, not conspiratorial.

## Build approach

Standard pipeline, branch `feature/confluence-scholars-quarter`:
1. Open the 6149 seam (after testers done).
2. Rooms 6218–6231 (`city` biome; final coords + exits, cartchecked; quote colon
   idlemessages; Title-Case mob names).
3. Items 40137 (book) + 40138 (Q73 reward, lore item).
4. Mobs 9441–9448 + dialogue; the giver's grant node (Q73 SOP); the Bookseller
   vendor; the `margin` groups.
5. **Quest 73 YAML** (`quests/73-the_margin_notation.yaml`): steps + rewards +
   the room_interact triggers (model on `quests/69-the_gallery_cipher.yaml` /
   `70-the_pre_founding_web.yaml`). Wire the cross-district source nouns.
6. Smoke test: wipe instances, boot clean (no panics, `ValidateZoneConsistency
   errors=0 mode=panic`, `quests.LoadDataFiles` +1), `cartcheck the_confluence`
   clean. **Harness/admin quest test** (`questtoken` admin for reliable mechanic
   checks — the harness adapter is flaky per the NP-questline lesson): grant Q73,
   examine the three sources (`look` in 6226/6131/6143), return, complete; verify
   reward + `margin` rep.
7. Merge `--no-ff`; update `ZONE_EXPANSION.md` (Confluence 4/10, Q73 done).

World after this district: **40 zones / 1070 → 1084 rooms** (+14).

## Out of scope

- Districts 5–10 (the Processional/Temple/Undercroft/outer quarters).
- Q74 (the Undercroft quest — the temple districts; Q73 only plants its hook).
- The crash-site reveal (reserved).
