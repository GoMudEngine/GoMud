# New Plymouth — Common Quarter (Design Spec)

*Date: 2026-06-20 · District 2 of 7 (built after the Docks). Where most of
the city actually lives.*

> **Parents:** master plan §8.2; city-wide layer §6 (Common anchors), §4
> (civic); `docs/new_plymouth_canon.md`. Built with the **7 gotchas** learned
> on the Docks (see `project-new-plymouth-build`): prose `": "` → `" — "`;
> faction membership via `groups:`; Title-Case mob names; valid `craft_support:`
> on shop mobs; `<ansi fg="itemname">`; schedules route via compass/up-down only;
> no faction `allies/enemies` forward-refs.
>
> **Scope:** ~25 rooms (staged A→B, smoke between), the 7 Common anchors, the
> ambient/transient crowd, the **Street Sweeper's Secret** questline, civic
> infrastructure, expansion stubs. The supply-runner wiring stays deferred.

---

## 0. Decisions locked
- Built **staged A (Carter's Rise + market + entry) → B (tenements + tannery +
  shadows)**, boot-smoke between.
- **Entry via the Docks:** the Docks' **Aldric's Public House (5514) → east →
  Common (5600)** — the dock crews who drink at Aldric's live here. The overland
  **East Gate (5471) stays a dead-end stub** (the Docks/outskirts box in its only
  northward room; its proper wiring is a future polish, not this build).
- Carter's Rise is the **civic heart** (well + flower market) — the Common's
  equivalent of the Docks' Long Quay.

---

## 1. The living rhythm
This is the working city's home. By day it empties toward the docks and the
other quarters for work; midday and evening it fills — the **well at Carter's
Rise** for water and gossip, the **cookshop row** and the **Brimming Bowl** for
food and drink, the **back-alley healer** for what the temple won't treat. The
sweeper sees everything; the tenement matron counts everyone; the pit roars
behind a locked gate. Children, laundry-lines, too many people in too little
space. Rooms exist to house and feed and water that crowd.

---

## 2. Geometry & room list
**Region:** x −16…−2, y 78…95, z 0. **Entry:** Docks **Aldric's 5514 (−17,82)
→ east → 5600 (−16,82)**. Rooms **5600–5624** (25). Exits to other districts
(Merchant north, etc.) are described stubs until those districts exist.

> Coords/exits below are an indicative layout guide — exact coords + reciprocal
> exits assigned and **`cartcheck`-validated at build** (the Common sits clear of
> the Docks at x ≤ −16... i.e. x from −16 east to −2). Update `coordinate_map.md`.

### Stage A — Carter's Rise, the market, the entry (5600–5612)
| Room | Name | Role / nouns |
|---|---|---|
| 5600 | Dockgate Street | entry from Aldric's (5514 e); the seam from docks to homes; *boundary stone, ballad-sheet* |
| 5601 | Lower Common Way | the main residential street climbing inland; *gutter-channel, doorstoops* |
| 5602 | Carter's Rise | **civic heart** — the public well + flower market on the rise; *the well, flower stalls* (the hollow-accusation scene site) |
| 5603 | The Flower Market | stalls around the Rise; *cut-flower buckets, a fortune-teller's mat* |
| 5604 | The Common Market | a cheaper, scrappier market than the Merchant Quarter; *barrow-stalls, a weighing-beam* |
| 5605 | Cookshop Row | street-food + the cookshop; *braziers, a shared trestle* |
| 5606 | The Brimming Bowl | the neighborhood tavern (exterior+door) — Renn Bowl's; *painted bowl sign, doorway* |
| 5607 | The Brimming Bowl Taproom | INTERIOR (enter/out) — loud, cheap, the conversation hub; *long benches, a warped bar* |
| 5608 | The Public Cistern | a covered rain-cistern + washing place; *cistern lid, washboards* |
| 5609 | The Bathhouse | cross-class mixing; *steam-room door, a tally-slate of tokens* |
| 5610 | Tinker's Lane | a side street of menders + a barber; *whetstone, a barber's stool* |
| 5611 | The Common Shrine | a small Chrysalis streetshrine; *offering bowl, worn step* |
| 5612 | Upper Common Way | the street climbing toward the (future) Merchant Quarter; **stub:** the way north to the Merchant Quarter (described, no exit) |

### Stage B — Tenements, tannery, the shadows (5613–5624)
| Room | Name | Role / nouns |
|---|---|---|
| 5613 | Tenement Row | the main housing; *washing lines, a shared stoop*; **stub:** condemned upper floors (barricaded stair, sounds above) |
| 5614 | Mother Brisk's Block | the tenement matron's building; *rent-board, a watchful window* |
| 5615 | The Back Court | a tight interior court between blocks; *a dry fountain, a children's chalk-game* |
| 5616 | The Tannery Street | the tannery trades (the Docks/Common seam smell); *soaking pits, stretched hides* |
| 5617 | The Tanner's Yard | Corwin-style tanner's workplace; *scraping-beams, a lime barrel* |
| 5618 | Sweeper's Corner | Coll the sweeper's haunt; *a gutter-grate, a hoard of odd finds* |
| 5619 | The Back-Alley | Ysolde's no-questions clinic (exterior); *a curtained door, a herb-bunch* |
| 5620 | Ysolde's Room | INTERIOR (enter/out) — the back-alley healer; *a cot, a shelf of unlabeled jars* |
| 5621 | The Fighting-Pit Gate | **stub:** a locked iron gate, a bouncer, cheering within (the pit itself = future) |
| 5622 | The Flophouse Steps | Garrick's haunt; *a dice-scratched step, a flophouse door* |
| 5623 | The Undercroft Stair | **Street Sweeper's Secret payoff** — a stair down to a pre-Founding cellar; *worn steps* (z−1 undercroft, 1 room) |
| 5624 | The Old Foundation | z−1 — pre-Founding stonework + the fragments; *gray-material seam, sealed niche* (Street Sweeper reward + lore) |

---

## 3. The 7 anchors (mobs 9319–9325; `np_commonfolk` faction — CREATE it)
Full life-sheets at build (home/work, mutation, ≥3 dialogue topics, schedule
deferred to the schedules pass). `np_commonfolk` faction def (default_rep +10,
**`allies: []` `enemies: []`** — no forward-refs; the np_dockfolk ally edge gets
added later).
1. **Tova (9319)** *(canon)* — landlady; faintly luminous freckles. Runs a
   rooming house (place at Mother Brisk's Block or a dedicated room). **Holds
   "the group that left" lore.** Topics: rooms/lodgers, the old seminary, the
   ones who left.
2. **Coll the Sweeper (9320)** — hears through stone; street-sweeper who knows
   everything. **Gives the Street Sweeper's Secret quest.** Topics: gossip, the
   odd fragments he finds, where everyone is.
3. **Marda (9321)** — hands that never burn; cookshop keeper (5605). `craft_support:
   cooking`. **Sister of Halvard (Crafting smith — cross-district, built later).**
4. **Renn Bowl (9322)** — a carrying voice; the Brimming Bowl keeper (5607).
   `craft_support: cooking`. **Brews ale sold to Bressa's Salt Cellar (Docks).**
5. **Ysolde (9323)** — empathic (feels others' pain); back-alley healer (5620).
   `bloom_trade`-adjacent (treats addicts). Topics: healing, no questions, the
   Bloom-sick (Bloom Trail echo).
6. **Garrick One-Hand (9324)** — a stone-hard forearm; retired soldier, pit
   fixture (5622). **Old comrade of Guard-Captain Doryn (Noble — built later).**
7. **Mother Brisk (9325)** — counts perfectly; tenement matron (5614). Wary of
   the bloodline tax collector. Topics: rent, her tenants, the condemned floors.

**Ambient/transient (9326–9330):** common-folk (laundresses, errand-kids, a
street-barber, a tanner, a flower-seller, a dice-player), on pooled idlecommands.

---

## 4. Civic infrastructure (on the Common's streets)
Carter's Rise **well + flower market** (the heart), the **Brimming Bowl** tavern,
**Cookshop Row**, the **bathhouse**, the **public cistern**, the **streetshrine**,
the **fighting-pit gate** (stub). These are where common-folk routines cross.

---

## 5. Questline — The Street Sweeper's Secret (quest 65)
Coll the sweeper has been finding odd fragments of old gray material in the
gutters. **3 breadcrumbs:** Coll mentions it; a child plays with one in the
street (room flavor); the back-alley healer keeps one as a curiosity (5620 noun).
**2 resolutions:** (a) collect 3 fragments (room_interact on gutter-grates /
finds across the quarter) and bring them to Coll → he leads you to the
**undercroft stair (5623)**; (b) follow Coll's hint straight to the undercroft.
Reward: access to the small **pre-Founding undercroft (5624)** + lore + gold +
`np_commonfolk` rep. Quest item: a "lump of gray material" (item 40101). Mirror
the Dock Rat quest's `room_interact`/`item_give`/`quest_granted` structure.

---

## 6. Expansion stubs
Condemned tenement upper floors (5613 — barricaded stair, habitation sounds);
the **fighting pit** (5621 — locked gate, bouncer "by invitation"); the way north
to the **Merchant Quarter** (5612 — described, future); a **river gate** in the
wall (a Common edge room — chained flood-gate, future river travel).

---

## 7. ID allocation
Rooms 5600–5624 (Stage A 5600–5612, Stage B 5613–5624, incl. z−1 5623/5624).
Mobs 9319–9330. Items 40101+ (the gray-material fragment). Quest 65. Dialogue from
the next free. Faction `np_commonfolk` (new).

---

## 8. Build sequence
1. **Stage A rooms** (5600–5612 + the Aldric's 5514→east seam) → cartcheck-clean,
   boot-smoke.
2. **Stage A population** (Tova, Coll, Marda, Renn Bowl + ambient + `np_commonfolk`
   faction; the Carter's Rise / market / tavern crowd) → smoke.
3. **Stage B rooms** (5613–5624 incl. the z−1 undercroft) → smoke.
4. **Stage B population** (Ysolde, Garrick, Mother Brisk + tannery/pit ambient) → smoke.
5. **The Street Sweeper's Secret quest** (65) + harness test.
6. **Anchor schedules** (compass/up-down routing only) + district smoke.
Then later: backfill the np_dockfolk↔np_commonfolk ally edge; wire the supply
runner once more districts exist.
