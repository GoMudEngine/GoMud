# Mob Aliveness 6.1 — Stillwater Town-Flavor Pass (Design)

**Date:** 2026-06-03
**Chunk:** 6.1 (Polish phase) — first-zone benchmark
**Size:** L
**Depends on:** Phase 1 (relationships 1.6, knowledge 1.4, facts 1.7,
factions 1.2), Phase 3 (schedules 3.2, sleep 3.3, conversations 3.6)
**Status:** Design approved 2026-06-03

## Purpose

6.1 is the **first-zone benchmark** for the mob-aliveness framework: take the
substrate and routine layers built across Phases 1–5 and apply them
comprehensively to one real zone (Stillwater) to validate authorability and
catch what is hard to author before the broader content rollout (6.5).

This is a **content pass**, not an engine change. No new Go code, no new
schemas — only data files exercising existing systems. Any system gap the pass
exposes is logged as a followup, not fixed inline.

## Scope decisions (locked with user)

- **Ambition:** "Layered-by-fit" — apply each substrate layer to the NPCs where
  it earns its keep, mirroring how Thornwall was actually authored (selective,
  not every-NPC-every-layer). Sized L.
- **Layers in scope:** Relationships (1.6), Schedules (3.2), Knowledge + Facts
  (1.4/1.7), NPC↔NPC Conversations (3.6).
- **Layers out of scope:** Strategic goals (Phase 4) — deferred; faction
  membership — **already done** (`groups: stillwater_citizens` /
  `stillwater_guards` are wired and both faction definition files exist).
- **Schedules:** Full 8 anchors (richest-texture tier).
- **New rooms:** **None.** All rest/sleep schedule targets are existing rooms.
  See "Room consistency" below.

## Current state (verified 2026-06-03)

- 22 town NPCs (ids 333–356, monsters 330–332 excluded).
- Dialogue files already exist for 333–355 (broad coverage — the pre-substrate
  memory note claiming only 335/342 had dialogue is stale).
- Faction membership already wired via `groups:` tags.
- **Not yet present on any Stillwater NPC:** `relationships:`, `schedule_id:`,
  `knows_facts:`.
- Gossipers tagged: Neva (334), Hodder (343), Gyda (349).
- No Stillwater NPC↔NPC conversation pairs; conversation type-pools exist only
  for `friend` and `rival`.
- `facts.yaml` holds only a `test-mayor` seed fact.

## Sequencing approach

Author **layer-by-layer with a boot-validate gate between each**, in dependency
order:

1. Relationships → boot (validator warns on unknown ids; permissive).
2. Facts + knowledge → boot (undeclared-fact references are fine; facts.yaml is
   author-declared).
3. Schedules → wipe `mobs.instances/` + `rooms.instances/`, then boot
   (validator **panics** on coverage gaps / unreachable target rooms /
   unresolved `schedule_id`).
4. Conversations → boot.

Rationale: each layer's validators surface problems at startup; booting after
each keeps the blast radius small and panics localizable. Rejected
alternatives: NPC-by-NPC vertical slices and big-bang-then-boot both fight the
inherently cross-NPC nature of relationships/conversations.

---

## Section A — Relationship graph (1.6)

Authored in each mob's `relationships:` field (`to`, `type`, optional
`subtype`). Engine auto-mirrors symmetric types and the employer↔employee
asymmetric pair at load. Edges (author once per pair; mirror is automatic):

**Employment**
- Sigrid (333) `employer` → Neva (334)
- Seren (344) `employer` → Finn (345)

**Family — Voss cluster**
- Ulla (347) `family` Vella (355), subtype `sister-in-law`
  (Ulla = Elgar's widow; Vella = Elgar's sister)
- *Cross-zone, conditional:* Ulla (347) `family` Maren (Thornwall weaver),
  subtype `niece` — **only if a Maren mob exists.** Verify during authoring;
  drop silently if Maren is dialogue-only.

**Friendship (mentor/peer subtypes)**
- Hodder (343) `friend` Tov (336), subtype `mentor`
- Hodder (343) `friend` Luc (346), subtype `mentor`
- Drunn (335) `friend` Sigrid (333), subtype `regular`
- Ilsa (338) `friend` Vella (355), subtype `colleague`
- Gyda (349) `friend` Vella (355), subtype `neighbor`
- Brindle (337) `friend` Hodder (343)

**Rival (optional texture)**
- Sigrid (333) `rival` Wulf (341), subtype `petty` (inn-vs-store grumbling)

~10 edges across 12 of 22 NPCs — a connected social graph grounded in existing
lore, no invented ties.

---

## Section B — Schedules (3.2) — 8 anchors

Each anchor gets a `schedule_id:` field pointing to a new file under
`_datafiles/world/dogmud/schedules/stillwater/<id>.yaml`. Each schedule has 3–4
segments covering all 24h (validator panics on gaps), routes between an
existing work room and an existing rest room, and gates craft/sleep via
`activity:`. Per-segment idlecommand pools follow the Thornwall precedent
(emotes + occasional `say`).

| NPC | Work (day) | Evening | Night (`activity: sleeping`) |
|-----|-----------|---------|------------------------------|
| Sigrid 333 | Pike & Lantern bar 4103 | tavern, busier 4103 | loft 4104 |
| Neva 334 | Pike & Lantern service 4103 | tavern peak 4103 | loft 4104 |
| Brindle 337 | Smithy 4106 (`craft`) | tavern 4103 wind-down | smithy 4106 |
| Seren 344 | Temple 4123 (services AM/PM) | temple 4123 | temple 4123 |
| Arn 342 | Docks 4116 (ledger) | promenade 4113 dusk | docks 4116 |
| Ilsa 338 | Healer's Alcove 4125 (`craft`) | 4125 close | 4125 |
| Bram 348 | Watermill 4135 (`craft`) | 4135 bank wheel | 4135 |
| Vella 355 | Healer's Cottage 4136 (`craft`) | **Ulla's Parlor 4137** | 4136 |

**Confirmed existing spawn rooms:** Ilsa 4125, Arn 4116, Bram 4135 (verified via
room spawninfo); the rest from the roster map. **All rest/sleep targets are
existing rooms** — no new rooms.

**Standout cross-NPC beat:** Vella's evening route to Ulla's Parlor (4137) puts
the two Voss-grief women in one room, which also gives the conversation layer a
live pair (`347_355`) to fire on.

**Reachability:** the schedule validator panics on unreachable `target_room`.
During authoring, confirm each work→rest pair is path-connected (e.g. 4103↔4104
loft exit, 4136↔4137). If a route is not connected, prefer sleep-in-place over
adding a room.

---

## Section C — Knowledge + Facts (1.4 / 1.7)

### C.1 Seed `facts.yaml`

Append 5 standing Stillwater facts (id / description / significance / tags /
status: active). `declared_round` set to a low/zero baseline so they read as
long-standing at boot.

| Fact id | Description | Tags |
|---------|-------------|------|
| `stillwater-lake-decline` | Boats return half-empty; nets shredded by something from the caves | stillwater, crisis |
| `stillwater-voss-death` | Elgar Voss lost to the lake 12 years ago; body never recovered | stillwater, history |
| `stillwater-spiral-motif` | Pre-Chrysalis spiral on temple, garden, chapel ruin, wardstones | stillwater, lore |
| `stillwater-cave-creatures` | Drowned hunters / skitter-shrimp pushed into the shallows | stillwater, crisis |
| `stillwater-pearl-divers-gone` | The black-pearl divers stopped coming; the water has "gone strange" | stillwater, history |

### C.2 Attach `knows_facts:` (role-gated, NOT universal)

The point of the knowledge model is that NPCs know only what their role/lore
supports:

- decline / cave-creatures: Arn 342, Hodder 343, Tov 336, Luc 346, Drunn 335
- voss-death: Ulla 347, Vella 355, Brindle 337, Hodder 343, Seren 344
- spiral-motif: Seren 344 (Vella 355 faintly)
- pearl-divers-gone: Kess 340, Hodder 343

### C.3 Gossiper expansion

Add `gossiper` to **Oswin (354)** (square beggar — hears everything) and
**Fenwick (353)** (pilgrim — brings outside news), per the original memory note.
With C.2 seeded, gossipers spread these facts via the existing gossip pipeline.

---

## Section D — NPC↔NPC Conversations (3.6)

### D.1 New generic type-pools

The pass uses relationship types that currently have no conversation pool.
Author role-agnostic generic pools (~6–10 exchanges each) under
`_datafiles/world/dogmud/conversations/types/`:

- `employer.yaml`
- `employee.yaml`
- `family.yaml`

(`friend.yaml` and `rival.yaml` already exist.)

### D.2 Stillwater pair overrides

Author town-specific pair files under
`_datafiles/world/dogmud/conversations/pairs/{lower}_{higher}.yaml`, role-agnostic
("A"/"B"), extending (not replacing) the type pool:

- `333_334` Sigrid/Neva — tavern banter, lake gossip
- `336_343` Tov/Hodder — nets, the old days, mentor cadence
- `338_355` Ilsa/Vella — two healers comparing methods
- `347_355` Ulla/Vella — the grief pair; gentle, never names Elgar outright;
  fires when Vella's schedule routes her to the parlor

### Voice guidance (applies to D and any idlecommand touch-ups)

Stillwater is the comfortable town that has had a quietly bad year. NPCs read
competent but slightly worn. Avoid jokey/upbeat voices except Pip (child) and
Sigrid behind the bar. Trigger discoverability SOP still applies to any new
dialogue triggers (every keyword must appear in a hint, NPC text, room noun, or
quest log).

---

## Section E — Room consistency (user constraint)

New rooms must keep the world map Cartesian-consistent: no two rooms in a zone
may share `coord:` x/y/z, and exits must be spatially reciprocal. **This pass
adds zero rooms** — every schedule rest/sleep target is an existing room, so
there is nothing to overlap.

Guard-rail for the contingency: if authoring reveals that a separate rest room
is clearly better for one NPC, prefer sleep-in-place instead. Only add a room if
truly unavoidable, and then verify (a) the chosen coords are unique within the
zone, (b) the coord delta matches the exit direction, and (c) the exit is
reciprocal. Default plan: no rooms added.

---

## Validation & smoke plan

- Boot after each layer (A→B→C→D), watching log for relationship-id warnings,
  schedule panics (coverage/reachability/unresolved id), and clean data-file
  load counts.
- **Before the schedule smoke:** `rm -rf mobs.instances/* rooms.instances/*`
  (stale instance saves shadow new `schedule_id`).
- Final manual in-game smoke **deferred to user** (per chunk-2.x precedent):
  walk Stillwater across a day/night cycle and confirm:
  - schedules move the 8 anchors between rooms and gate sleep/craft;
  - gossipers surface the 5 seeded facts;
  - the Ulla/Vella parlor scene fires in the evening segment;
  - relationship-keyed conversations appear between idle pairs.

## Out of scope / followups

- Strategic goals (Phase 4) on Stillwater NPCs — deferred.
- New quests — content pass only (leave quest bait, write no quests).
- Any engine/system gap exposed during authoring → log to MEMORY, do not fix
  inline.
- 6.2 (parity audit closeout) consumes what this pass teaches us.
