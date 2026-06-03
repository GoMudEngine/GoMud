# Mob Aliveness 6.3 — Thornwall Deepening (Design)

**Date:** 2026-06-03
**Chunk:** 6.3 (Polish phase) — per-zone tuning, second benchmark zone
**Size:** L
**Depends on:** 6.1 (Stillwater pass established the layered-by-fit pattern), Phase 1 substrate, Phase 3 routine layer
**Status:** Design approved 2026-06-03

## Purpose

6.3 applies the mob-aliveness framework to a **second zone** (Thornwall city) to
reveal the repeatable authoring pattern before the XL broad rollout (6.5), and
to **tune archetype/gossip defaults** based on what two zones teach us.
Thornwall is the dense *urban* contrast to Stillwater's fishing-village (6.1):
~25 social NPCs, a market, craft quarter, temple, bank, barracks, and a
caravan crew, plus a noir undercurrent (smuggling, a protection racket, a tithe
audit, a missing-person thread, the mayor's disgrace).

This is a **content pass + a small tuning pass** — mostly data files exercising
existing systems, plus evidence-driven adjustment of shared archetype/gossip
defaults. Any engine gap exposed is logged as a followup, not fixed inline
(except the targeted gossip-fact-tag investigation in §E, which may need a small
code fix).

## Scope decisions (locked with user)

- **Zone:** Thornwall only, deep pass. (Sanctum Basin explicitly excluded — it
  is being replaced by the queued newbie-area rework.)
- **Ambition:** Layered-by-fit (same as 6.1) — each substrate layer applied where
  it earns its keep, not exhaustively.
- **Flavor:** *Mix — public troubles only, NO quest spoilers.* Everyday townsfolk
  texture + rumor-level references to PUBLIC-knowledge troubles (mayor's
  disgrace, road bandits, "hard times"). Active quest threads stay OUT of casual
  gossip/conversation: Elara's whereabouts, the smuggling tunnels, the bribe
  ledger, Torvan's identity.
- **Layers:** Relationships (1.6), Schedules (3.2) +6 anchors, Knowledge+Facts
  (1.4/1.7), NPC↔NPC Conversations (3.6). Faction membership already wired
  (`thornwall_citizens`/`thornwall_guards` groups). Strategic goals (Phase 4) out
  of scope.
- **6.3-specific deliverables:** archetype-default tuning (§E) + before/after
  authoring notes (§F).

## Current state (verified 2026-06-03)

- ~25 social town NPCs (94–120, 248, 249, 357–359, 375–376) + dungeon mobs
  (skip). Dialogue exists for the quest-relevant ones.
- **Existing schedules (5):** tavern keeper Marek (96), barmaid Dal (117), smith
  Kerra (97), temple priest Olen (95), city guard (106).
- **Existing relationships:** only the 3 tavern regulars Fen (114), Gobb (115),
  Wrex (116) — friends.
- **Gossipers:** Fen, Gobb, Wrex.
- **Conversation pairs:** one — `116_117` (Wrex/Dal). Type-pools friend, rival,
  employer, employee, family all exist (6.1).
- **knows_facts:** none. `facts.yaml` has a `test-mayor` placeholder fact ("The
  Thornwall mayor has resigned in disgrace") — repurpose it.
- **Faction membership:** wired via `groups:`.

## Room map (for schedules — existing rooms only, no new rooms)

460 Gate Ward · 461–463 Main Street (W/Central/E) · 464 Market Square West ·
465 Market Square Center · 466 Market Square East · 468 Temple Interior ·
470 Smithy · 471 Apothecary Lane · 472 Drowning Post Tavern · 473 Guard
Barracks · 475 Back Alley East · 476 Records Office · 477 Residential Ward N ·
480 Tailor's Workshop · 481 Tavern Kitchen · 482 Jeweler's Workshop · 483
Enchanter's Circle · 484 The Back Corner · 510 Bank.

---

## Section A — Relationship graph (1.6)

Author ~13 edges (one side per edge; engine auto-mirrors), grounded in roles/
dialogue. Added to each mob's `relationships:` field.

**Employment**
- Marek (96) `employer` → Dal (117), subtype barmaid
- Marek (96) `employer` → Brynn (248), subtype cook
- Velk (94) `employer` → city_guard (106), subtype captain

**Caravan family/crew** *(substrate only — caravan travels, so NO pair-overrides;
conversations rarely fire mid-route)*
- Ketil (357) `family` → Lars (359), subtype son
- Ketil (357) `employer` → Marta (358), subtype guard

**Artisan-quarter colleagues**
- Tess (108) `friend` Vael (109), subtype colleague
- Voss (98) `friend` Vael (109), subtype colleague

**Civic**
- Olen (95) `friend` Pell (99), subtype "the audit"

**Street folk**
- Beggar (100) `friend` Performer (101), subtype street

**Tavern regulars** — keep the existing Fen/Gobb/Wrex friendships as-is (their
"old argument" stays *friend* flavor, NOT a conflicting rival edge). Optionally
add Dal (117) `friend` to one regular if it reads naturally; otherwise leave.

**Cross-zone:** Maren (113) ↔ Ulla (347) is already covered by Ulla's authored
edge (auto-mirrored) — leave as-is.

**Explicitly NOT authored:** any criminal tie (Torvan 249, Siv 104) — quest
territory and a spoiler risk.

## Section B — Schedules (3.2), +6 anchors (≈11 total)

Each new schedule is 24h, existing-rooms only, sleep-in-place or a reachable
existing room, with `activity:` gating craft/sleep and per-segment idlecommand
pools (Thornwall precedent). The headline is the **market square (464/465)
gaining day/night vendor life** — currently dead.

| NPC | Day (work) | Evening | Night |
|-----|-----------|---------|-------|
| Market merchant 102 | Market Square Center 465 | wind-down/close | sleep (in place or residential) |
| Food vendor 103 | Market Square West 464 (`craft` — is a cook) | close stall | sleep |
| Apothecary Voss 98 | Apothecary Lane 471 (`craft`) | close | sleep |
| Jeweler Tess 108 | Jeweler's Workshop 482 (`craft`) | close | sleep |
| Weaver Maren 113 | Tailor's Workshop 480 (`craft`) | close | sleep |
| Guard Captain Velk 94 | Barracks 473 + a **midday market-beat inspection at 465** (overlaps the city guard + market vendors) | barracks | sleep (barracks) |

Rest/sleep targets are existing rooms (sleep-in-place at the shop, or a
reachable residential/barracks room). Confirm each anchor's actual spawn room +
that consecutive-segment rooms are path-connected (the schedule validator panics
on unreachable `target_room`). Velk's market inspection is the cross-NPC beat
that also seeds conversation co-location (e.g. Velk + a market vendor).

## Section C — Knowledge + facts (1.4/1.7) — PUBLIC ONLY

Seed `facts.yaml` with **public-knowledge** Thornwall facts (repurpose
`test-mayor` into a real one). NO quest-spoiler facts.

| Fact id | Description | Tags |
|---------|-------------|------|
| `thornwall-mayor-disgraced` | The Thornwall mayor resigned in disgrace (rename of the existing `test-mayor` seed). | thornwall, politics |
| `thornwall-road-bandits` | The Thornwall–Stillwater road draws bandits; caravans run guarded. | thornwall, road |
| `thornwall-hard-times` | Taxes are heavy and "protection" money is grumbled about; trade is tight. | thornwall, hardship |
| `thornwall-steel-heritage` | Thornwall steel is an old guild craft, the technique passed hand to hand. | thornwall, craft |
| `thornwall-caravan-trade` | A regular caravan runs Thornwall↔Stillwater (lake-iron, pearls, goods). | thornwall, trade |

`thornwall-hard-times` is deliberately GENERIC (no racket specifics — that's
quest territory). Attach `knows_facts:` role-gated (not universal):
- mayor-disgraced: Pell 99, market merchant 102, Marek 96, the regulars
- road-bandits/caravan-trade: Ketil 357, Marta 358, Lars 359, market merchant
  102, food vendor 103, Velk 94
- hard-times: Marek 96 (taxed + racket-pressured), Siv 104, food vendor 103
- steel-heritage: Kerra 97, Maren 113, Tess 108

**Gossiper expansion:** add `gossiper` to Beggar (100) ("sees everything") and
Street Performer (101) ("reads the crowd"), alongside Fen/Gobb/Wrex.

## Section D — NPC↔NPC conversations (3.6)

Type-pools all exist (6.1). Add ~4–5 Thornwall **pair-overrides**, all swap-safe
(the 6.1 A/B-randomization lesson) and co-location-verified (the 6.1 dead-pair
lesson — both NPCs must share a room sometime):
- `96_117` Marek/Dal — both in the tavern (472) by schedule.
- `96_248` Marek/Brynn — Marek (472) and Brynn (kitchen 481); verify they
  co-locate (Marek's prep segment, or Brynn steps into the floor) — if they
  never share a room, give one a brief segment in the other's room (mirroring
  the 6.1 Hodder/Tov co-location fix) or drop this pair to type-pool only.
- `114_115` Fen/Gobb — both at the Back Corner (484).
- An artisan or civic pair: `108_109` Tess/Vael (both crafters; confirm
  co-location — adjacent workshops 482/483, may need a visit segment) OR `95_99`
  Olen/Pell (the audit; co-locate via a records-office visit). Pick whichever
  co-locates cleanly; author the other as substrate-only if not.

Keep the existing `116_117`. Author pairs role-agnostic ("A"/"B"), named
`{lower}_{higher}.yaml`.

## Section E — Archetype-default tuning (the 6.3 deliverable)

Evidence-driven, folding in the two 6.1 smoke followups
([[project_stillwater_6_1_smoke_followups]]):

1. **Investigate the gossip fact-tag gap.** In the 6.1 smoke, only crisis-tagged
   facts surfaced in gossip; lore/history-tagged facts did not. Check
   `buildGossipLine` / the `fact-default` template family (chunk 1.7) actually
   emits non-crisis facts (politics/road/craft/trade tags). If it's a real gap
   (e.g. the gossip pool filters by tag or the fact-default split starves
   non-event facts), fix it so Thornwall's history/heritage facts gossip too.
   This is the one place 6.3 may touch engine/template code.
2. **Widen idle-pool variety** where repetition shows (6.1 had Arn/Ilsa repeating
   a line) — author ≥3–4 distinct idlecommands per new Thornwall schedule
   segment, proactively.
3. **Tune conversation cadence if needed.** Thornwall packs more NPCs per room
   (square, tavern) than Stillwater, so `ConversationBaseChancePct` / cooldown
   may fire too often. Observe in the smoke, then adjust the knob(s) if the
   cadence is wrong (too chatty or too sparse). Knob change only if evidence
   warrants.

## Section F — Validation, smoke & before/after notes

- **Boot-validate after each layer** (relationships → facts/knowledge →
  schedules → conversations), watching for panics (schedule coverage/
  reachability/unresolved id) and relationship/fact warnings. Wipe
  `mobs.instances/` + `rooms.instances/` before the schedule smoke.
- **Manual in-game smoke deferred to user:** a Thornwall day/night walk
  confirming the 11 schedules move NPCs, the market square lives, gossipers
  spread the 5 facts (incl. the non-crisis ones, post-§E fix), and the new
  conversation pairs fire.
- **Before/after authoring notes** (the 6.3 brief's deliverable): after the
  pass, write a short note — what generalized cleanly from Stillwater (the
  layered-by-fit recipe, the type-pools, the co-location/swap-safety rules) and
  what was *harder* in a denser, quest-laden city (spoiler firewall, caravan
  mobs that travel, co-location in a bigger room graph). This de-risks and
  partially scripts the 6.5 broad rollout.

## Out of scope / followups

- Strategic goals (Phase 4) on Thornwall NPCs.
- New quests / any change to existing Thornwall quest content.
- Surfacing quest-secret lore (Elara, tunnels, bribe ledger, Torvan) in gossip.
- Criminal relationship edges (Torvan/Siv).
- Any engine gap beyond the targeted §E gossip-tag fix → log to MEMORY, don't fix
  inline.
- 6.2-style parity items already closed; 6.4 (performance review) consumes what
  this and 6.1 teach about substrate size/tick cost.
