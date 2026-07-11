# New Plymouth — District 6: The Noble Quarter (design)

**Status:** approved 2026-06-22 (brainstorming). Build sequence item #6 of the
capital (Docks ✅ → Common ✅ → Crafting ✅ → Merchant ✅ → Temple ✅ → **Noble** →
Old Quarter). Pulls anchors from the city-wide layer
(`2026-06-20-new-plymouth-citywide-design.md` §6 #36–42).

> **PUSH POLICY (user, 2026-06-20):** Do NOT push New Plymouth to prod until the
> WHOLE capital is built and tests well. Accumulate locally; no mid-build push.

## 0. Scope (locked with user 2026-06-22)

IN: rooms + mobs + dialogue + **faction MEMBERSHIP only (no new faction)** + the
**gallery cipher resolved as discoverable lore** + the **Bloom Trail Noble beat**
(Wenna) + **anchor schedules** + the **Doryn↔Garrick** cross-district relationship
edge + a **Dobb supply branch to Aurel**.

OUT (deferred): the district quest (the Gallery Cipher is resolved as LORE, not a
quest — no quest-engine work); the **Palace** (Doryn's gate is an impassable
endgame stub); the gated estate lane beyond Skell (impassable stub); the Old
Quarter / the full pre-Founding climax (the buried city + Gritta — Old Quarter
build); any forced respawn or new mechanics.

## 1. Zone, IDs, coordinates

- **Zone folder:** `new_plymouth_noble` (underscores; `ConvertForFilename("New Plymouth Noble")`).
- **Rooms:** 6000–6019 (20 — a smaller district). **Mobs:** 9368–9378 (7 anchors
  9368–9374 + 4 ambient 9375–9378). **Dialogue:** by mobid. **Items/Quests:** none.
- **Coordinates — REAL unified frame.** Noble climbs NORTH off the Merchant
  Processional: Merchant **5803 (Processional North, at −14/90)** → north → 6000.
  The open ground north of the built footprint (Merchant max y≈91, Crafting max
  y=92 only at x≤−17) is free; Noble occupies roughly x[−16..−4], y[91..99] —
  solved collision-free at build time, `cartcheck errors=0`.

Role (city-wide §2): *elite homes; bloodline admin; the watched streets.*

### Connections
- **Primary entry:** Merchant **5803 → north → 6000** (the Processional climbs into
  the Noble Quarter) + reciprocal `south`. (5803's existing exits south→5802,
  east→5804 stay intact.)
- **Deferred prose-stubs** (gotcha #5): the **Palace gate** (Doryn's gatehouse —
  impassable, endgame); the **gated estate lane** beyond Skell (impassable); any
  further-north exits. Interiors compass/vertical only (gotcha #10).

## 2. Layout & connections

Climbing the Processional into a chilly, surveilled elite quarter. Clusters:
- **The watched streets** — the Processional approach (6000) + a surveilled plaza
  (the elite quarter's observed atmosphere; a sentry, a noble passer-by).
- **The Administrative Office** — Steward Caldwin (the bloodline's will; Horst's
  superior-side contact).
- **The Art Gallery** — Keeper Lysha + the gallery hall where the **cipher resolves
  as lore** (the paintings read against the approved script).
- **The atelier** — Modiste Aurel (clothier; a Dobb supply stop).
- **The gated estate lane** — Porter Skell's gatehouse (turns all away — stub) +
  Wenna's servants' garret (the **Bloom delivery-house** breadcrumb).
- **The Palace gatehouse** — Guard-Captain Doryn (the Palace gate — endgame stub;
  the Doryn↔Garrick beat).
- Guide Ferrol's flat / tour route threads the quarter.

## 3. Anchors (7 — city-wide §6 #36–42), full life-sheets

Each: home + work room, ≥3 discoverable first-person dialogue topics, the unique
mutation in the description, faction via `groups:`, a 24h `schedule_id` (Stage D).
Most Noble anchors have **no shop** (admin/guard/servant are not vendors); Aurel
(clothier) and possibly Lysha (gallery — likely no shop) are the only candidates.

| # | Mob | Name | Mutation | Role | Shop / `craft_support` | `groups` |
|---|-----|------|----------|------|------------------------|----------|
| 36 | 9368 | **Steward Caldwin** | none (uninfected pride) | bloodline functionary, the Admin Office | none | humanoid, bloodline_domestic |
| 37 | 9369 | **Wenna the Servant** | flinches with a faint fear-light | nervous servant; the Bloom delivery-house secret; terrified of Caldwin | none | humanoid, np_commonfolk |
| 38 | 9370 | **Guide Ferrol** | a rehearsed smile that never reaches the eyes | tour guide (approved history) | none | humanoid, bloodline_domestic |
| 39 | 9371 | **Keeper Lysha** | color-true eyes (reads old pigments) | art-gallery keeper; the cipher resolution | none | humanoid, cooperage_circle |
| 40 | 9372 | **Porter Skell** | an immovable stance | liveried porter, the gated estate lane (stub) | none | humanoid, bloodline_domestic |
| 41 | 9373 | **Modiste Aurel** | fingers that read fabric quality blind | high-end clothier; cloth from Nessa; a Dobb stop | `tailoring` shop (cloth/garment goods — reuse VERIFIED item ids: thread 40012, cloth strip 40007 as the runner-deliverable materials + 1-2 finished garments if a valid id exists) | humanoid |
| 42 | 9374 | **Guard-Captain Doryn** | a parade-perfect physique | ceremonial palace guard, the gatehouse (Palace stub) | none | humanoid, bloodline_domestic |

Plus **~4 ambient** (a liveried footman, a noble passer-by, a watched servant, a
gatehouse sentry) — `non_combatant` (mobs 9375–9378).

**Authoring gotchas:** YAML `": "` → `" — "`; Title-Case names; `ConvertForFilename`
filenames; faction via `groups:`; noun ansi `fg="itemname"` (post-grep);
sentence-leading noun tags lowercase; interiors compass/vertical; avoid
prefix-shadowing triggers; only verified item ids in Aurel's shop.

## 4. Factions — membership only

No new faction YAML. Noble anchors join EXISTING factions via `groups:`:
`bloodline_domestic` (Caldwin 9368, Ferrol 9370, Skell 9372, Doryn 9374),
`np_commonfolk` (Wenna 9369 — working up in Noble), `cooperage_circle` (Lysha 9371
— quietly). Aurel (9373) is faction-neutral. (All these factions already exist.)

## 5. The gallery cipher — resolved as discoverable lore

Keeper Lysha's gallery is the payoff of Dross's Temple breadcrumb ("the cipher's
answer is in the Noble Quarter gallery"). Discoverable content — Lysha's dialogue
+ the **gallery paintings** noun(s) read against **Ferrol's approved-history
script** — reveals the city's **suppressed pre-Founding history**: the eight-pointed
/ orbital symbol, an older settlement the bloodline's founding narrative overwrites.
Lysha (color-true eyes, reads the old pigments; a `cooperage_circle` sympathizer)
confirms what Dross suspected, what Ept kept seeing, and what Orin's pre-survey
maps showed — the threads converge here into a real revelation. **No quest.** The
FULL convergence (the buried city, Gritta, the canal) still waits for the Old
Quarter. Ferrol is the counterpoint — his approved script actively contradicts the
gallery, and a player who's heard both sees the suppression in action.

## 6. Bloom Trail Noble beat + Doryn↔Garrick + Dobb to Aurel

- **Bloom beat (Wenna):** a discoverable dialogue/noun where the terrified servant
  lets slip the **Noble delivery-house** — a guarded address (an estate on the
  gated lane) where unmarked goods arrive at odd hours under Caldwin's eye —
  pointing onward toward the Old Quarter (advancing from Falk's Lintel St beat).
  Content-only, unresolved; Wenna is frightened, so it's oblique.
- **Doryn↔Garrick relationship edge:** add `relationships:` to BOTH mobs (the
  schema is a list of `{to: <mobid>, type, subtype}`): Doryn (9374) → `{to: 9324,
  type: friend, subtype: old_comrade}`; and edit Garrick (`9324-garrick_one_hand`,
  Common) → `{to: 9374, type: friend, subtype: old_comrade}`. Plus dialogue
  cross-references both ways (Doryn speaks warmly of his old pit-comrade Garrick in
  the Common; Garrick optionally gains a line — keep the Garrick edit minimal).
- **Dobb to Aurel:** append ONE waypoint to `np_docks_runner_circuit.yaml` — from
  the temple gate (5903, the current last waypoint) the strict loop returns to the
  depot; insert **Aurel's atelier** (the Noble waypoint) appropriately so Dobb
  delivers cloth strip (40007) / thread (40012) — both already in the manifest. The
  path must be pathto-reachable across the Merchant→Noble seam (5803→6000). Noble
  is an off-spine branch; validate the loop still loads with no home-fallback.
  Aurel must pre-declare those materials as StockEntries (her shop block).

## 7. Anchor schedules (Stage D)

24h-contiguous schedules at `schedules/new_plymouth_noble/` (validators panic on
gaps/unreachable). Beats: Caldwin's office hours (the Admin Office); Ferrol's tour
circuits (gallery → plaza → flat); the gallery hours (Lysha); the gatehouse
watches (Doryn, Skell); Wenna's furtive servant routine (garret → errands, avoiding
Caldwin); Aurel's atelier. All `pathto` targets within the built Noble rooms.

## 8. Build staging — feeds ONE plan (each stage boot-verified)

> **Pre-smoke ritual** (CLAUDE.md SOP): wipe instance saves before each boot;
> boot-poll for `ERROR:.*PANIC`/`fatal error:`, not bare "panic" (gotcha #8).

- **Stage A — the Processional approach + watched streets (rooms):** 6000 + the
  plaza + the entry; wire Merchant 5803 `north`. `cartcheck`-clean.
- **Stage B — cluster rooms:** Admin Office, the gallery, the atelier, the gated
  lane + Wenna's garret, the Palace gatehouse, Ferrol's flat. Boot-verify (the
  Palace gate + gated lane are impassable prose-stubs).
- **Stage C — population:** 7 anchors + ambient + dialogue + Aurel's shop +
  faction MEMBERSHIP (groups) + room spawns.
- **Stage D — gallery-cipher lore + Bloom beat + Doryn↔Garrick + schedules + Dobb
  to Aurel:** §5/§6/§7. Boot-verify (schedule + patrol validators; the relationship
  edges load). Supply smoke if practical.
- **Stage E — district harness playtest** (`/playtest local feature-tester`) →
  report → merge `--no-ff` (hold push) → update memory.

## 9. Definition of done

- 20 rooms (6000–6019) boot-clean, `cartcheck` errors=0, the Merchant→Noble entry
  works.
- 7 anchors + ambient live; the Palace gate + gated estate lane impassable;
  dialogue ≥3 topics each.
- Faction membership correct (bloodline_domestic / np_commonfolk / cooperage_circle).
- The gallery cipher pays off as discoverable lore (Lysha + the paintings vs.
  Ferrol's script); Dross's Temple breadcrumb resolves here.
- Wenna's Bloom delivery-house breadcrumb in place (points toward the Old Quarter).
- Doryn↔Garrick relationship edge + dialogue cross-refs live.
- Dobb's circuit reaches Aurel; her materials deliverable.
- Anchor schedules pass validators.
- Harness-playtested; report committed. Merge to master, hold push.

## 10. Honored gotchas checklist

#2 faction refs (membership only — all referenced factions already exist) · #5 no
exits to unbuilt rooms (Palace/gated-lane/Old-Quarter stubs) · #6 shop mobs need
`craft_support:` (Aurel) · #7 `fg="itemname"` ansi · #8 boot-poll `ERROR:.*PANIC` ·
#10 cardinal/vertical interiors + REAL unified-frame coords · Plan-2 supply rules
(Aurel pre-declares deliverable materials; validate the off-spine loop) ·
sentence-leading noun tags lowercase · no prefix-shadowing triggers · only verified
item ids · YAML `": "` → `" — "` · Title-Case names.
