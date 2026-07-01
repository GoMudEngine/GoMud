# The Disc Questline (Quest 76) — design

*Date: 2026-07-01. Sub-project 1 of #22 (the Crash Site Interior). The on-ramp: how a player obtains the usable disc key that opens the crash-site door.*
*Reference: `docs/superpowers/specs/2026-07-01-crash-site-interior-design.md` (the finale this gates).*

## Purpose

Turn the scattered disc/symbol lore into a playable questline that ends with the
player holding a **usable disc key** for the crash-site door. It weaves existing
threads — the disc in Pothole Coulee, the orbital symbol, Brennan/Reth in
Greenford, and **Maren's father** (the scholar who went east and never returned) —
into one arc, and it works for **every** character regardless of when or where
they were created.

## The connection (locked)

The carved **MAREN** at the crash-site father's camp (Eastern Highlands 6365) is
her father keeping his lost daughter's name in mind. **Maren's father** went east
years ago, obsessed with the buried thing; he studied how to *wake* the disc.
Maren herself (the Ashwick herbalist, Chrysalis-touched, "left because she had
to") stays **offstage** — her established story is untouched. (The unrelated
Thornwall "Weaver Maren" is a name collision and is NOT involved.)

## The arc (Quest 76 — 4 steps)

**Gate:** available after **Greenford Q75** (Reth's testimony — the crash-site
directions). Q75 tells you *where*; Quest 76 gets you *in*.

1. **The inert disc (obtain — universal).** Pothole Coulee **5343 "The Reliquary"**
   already describes the disc under the floor. Make it a **per-player takeable
   grant**: search reveals it, and taking/interacting grants *your own* **Inert
   Disc** item (per-player, repeatable — NOT a single shared world item). Any
   character can get one: newbie-era players grab it in passing ("had it all
   along"); everyone else travels back (Pothole Coulee is reachable overland via
   Ironwind Steppe) — a thematic pilgrimage to where it began. The Inert Disc does
   nothing on its own.
2. **Brennan connects it (quest start).** At Greenford University, post-Q75,
   `ask brennan` about the disc / the symbol → he recognizes it as a **key, but
   inert**, and names the scholar who went east from Ashwick to wake it and never
   returned (**Maren's father**). Grants `76-start`. **Universality branch:** if
   the player does NOT have the Inert Disc, Brennan directs them to **retrieve it
   from the Reliquary in Pothole Coulee** (explicit in-dialogue directions) — so
   the disc is discoverable *through the quest*, not only via the newbie funnel.
   → go to Ashwick, find the father's research.
3. **The father's journal (Ashwick 4023 "Maren's Cottage").** The abandoned
   cottage holds his **journal** — a gated `room_interact` (examine the journal)
   that reveals: his obsession with the buried thing, the disc's nature, the
   orbital symbol, and **the attunement method** (how to wake the disc). Grants
   `76-journal`. Lore-rich but threshold-only re the buried thing's true nature
   (the revelation stays for #22).
4. **The attunement ritual (Ashwick 4023).** With `76-journal` + the Inert Disc, a
   gated `room_interact` at the cottage (following his method) performs the
   attunement → **consumes the Inert Disc**, grants the **Attuned Disc** (the
   usable key) + `76-end`. The quiet emotional close: the daughter's name in his
   work, the man who kept her in mind while chasing this. Reward beat + the
   Attuned Disc.

**Payoff:** the **Attuned Disc** is the key the Threshold-Keeper needs at the
crash-site door (the #22 entry). This questline is complete on its own (a
satisfying arc: curiosity → the father's legacy → the key) even before #22 exists.

## The reckoning-bone — lore thread + optional enrichment (NOT a gate)

The reckoning-bone (item 30066) shares the symbol "at its deepest end" and is a
beautiful thematic tie — BUT it is the reward of **quest 48** (Pothole Coulee's
Standing Stones), so requiring it would re-introduce the same universality problem
(pre-Pothole-Coulee players lack it). Therefore:
- The **journal + Brennan reference the reckoning-bone** (the symbol connection —
  the "reckoning" is the record, the disc is the key it describes). Lore only.
- **If the player holds the reckoning-bone** (did quest 48), the journal / ritual
  gains an **extra optional lore beat** (the two symbols meeting) — a reward for
  the mystery-followers.
- The ritual **requires only the Inert Disc + the journal step** — never the
  physical bone. Universality preserved.

## The disc-key mechanic (locked)

- **Inert Disc** (new item): granted per-player at Pothole Coulee 5343; inert
  (a curiosity; no mechanical effect). `not_salable` (a story key).
- **Attuned Disc** (new item): granted by the attunement ritual; the usable key
  for the crash-site door / the Threshold-Keeper (#22). `not_salable`, and ideally
  flagged so it can't be lost/dropped-and-stuck (a recovery path — see SOP).
- **Attunement** = a gated `room_interact` at Ashwick 4023 that consumes the Inert
  Disc and grants the Attuned Disc (per the give.go/quest-engine patterns used
  across the Eastern Arc: `room_interact` fires on `examine`/`look`).

## Files (for the plan)

- Modify `rooms/pothole_coulee/5343.yaml` — the disc becomes a per-player takeable
  grant (Inert Disc); keep the existing evocative noun prose.
- Modify `rooms/ashwick/4023.yaml` (Maren's Cottage) — add the father's journal
  (`room_interact` → `76-journal`) + the attunement `room_interact` (gated
  `has:76-journal` + Inert Disc → consume Inert Disc, grant Attuned Disc +
  `76-end`), + the optional reckoning-bone lore beat.
- Modify `dialogue/greenford/9516.yaml` (Brennan) — the `76-start` quest-grant node
  (post-Q75 gated) + the "retrieve the disc from Pothole Coulee" directions branch.
  Follow the Quest NPC Dialogue SOP (`"quest"`/`"task"` triggers) + the re-grant
  prevention SOP (questExcluded incl. `76-end`).
- New items: **Inert Disc**, **Attuned Disc** (both `not_salable`); the father's
  journal is a room noun/interact, not necessarily an item.
- New quest `quests/76-*.yaml` — 4 steps (start/journal/attune/end), declared
  skeleton, flags if needed.

## Prereqs / gating

- **Q75 (Greenford)** = hard prereq (Brennan's grant is gated on Q75 complete).
- The **Inert Disc** = required for the ritual (universal via the per-player grant
  + Brennan's directions).
- **Quest 48 / the reckoning-bone** = NOT required (optional enrichment only).
- Q76 is endgame-tier (post-Q75), but has **no combat/level gate itself** — it's an
  investigation/travel quest; the danger is the crash-site arc it unlocks.

## SOP / gotchas (banked from the arc)

- **Quest NPC dialogue:** `"quest"`+`"task"` in the grant node's triggers; gated
  grant nodes FIRST under `tree.nodes` (short-trigger substring-shadowing); no
  semicolons in text; `|` blocks for long text; `room_interact` fires on
  `examine`/`look` (write hints as "examine the journal").
- **Re-grant prevention:** every `grantsQuest` node's `questExcluded` includes the
  end token (`76-end`).
- **Quest item recovery:** the Attuned Disc is pivotal — if it can be lost, provide
  a recovery path (Brennan re-issues, or the ritual is re-runnable if the Inert
  Disc is re-fetched). Design the ritual so a player who lost the Attuned Disc can
  re-fetch an Inert Disc (Pothole Coulee) and re-attune. (Avoids a bricked finale.)
- **Universality is the core constraint:** every required component (the Inert Disc)
  is obtainable by any character via a reachable, repeatable, quest-directed path.

## What this questline deliberately is NOT

- Not the crash-site interior (that is #22 — this only produces the key).
- Not a Maren-onstage quest (she stays offstage; her story is untouched).
- Not gated on newbie-era play or on quest 48 (universality).
- Not the revelation (threshold-only; the truth stays for #22's records).
