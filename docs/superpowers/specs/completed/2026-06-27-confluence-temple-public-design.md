# The Confluence — District 5b: The Temple of Confluence (public)

**Date:** 2026-06-27
**Status:** Approved (design phase)
**Umbrella:** `docs/superpowers/specs/2026-06-26-confluence-citywide-design.md` (§3 row 5)
**Predecessor:** District 5a — The Processional, merged `6db34753` (2026-06-27). This
is the second half of city-wide build-order step 4.
**Successor:** District 6 — Cloisters & Archive + the Undercroft (Q74). This district
sets up that climax but grants no quest.

## 1. Concept

The worship heart on the central island — an axial temple entered through the
Processional portico's now-open great doors. The player moves down a nave to the
**crossing / great hall where the three rivers join in a single channel below the
floor** (the payoff of Wess the pilgrim's River Road line, 9414), past three
river-chapels and a reliquary to the sanctuary, and sees — but cannot pass — the
**warded inner door to the cloisters**. The Keepers' official account reaches its
fullest, most beautiful expression here; to a Q73-completed player the cracks show
(the oldest stone underfoot at the crossing, the relic resting on pre-Founding
stone, a disused fourth alcove no one remembers consecrating).

**Lore boundary (threshold only, carried from the city-wide layer §5):** the temple
may intensify the sense that it was built FOR this confluence, atop older work, and
that the cloisters and "what lies beneath" are closed to the public. It NEVER states
the why (crash, gray material, mutation link). One understated forgotten-fourth beat
(the disused alcove); no numerology lectures.

## 2. Entry / seam (revises 5a)

The 5a portico (6167) currently stubs the great doors shut and redirects pilgrims to
"the island's far side" — a dead-end the 5a feel-review flagged. **This district
resolves it:** the great doors stand open for worship by day and become the public
entrance.

- **Revise `_datafiles/world/dogmud/rooms/the_confluence/6167.yaml`:** add
  `south → 6168`; rewrite the `great-doors` noun and the relevant description line so
  the doors stand open onto the narthex by day (remove the "directed to the far side"
  redirect entirely). Keep the four-ring `weathered-symbol` above them unchanged.
- The temple occupies the island's **west half (x +2 → +5)**. The warded inner
  threshold (6183) stubs **east** toward the future Cloisters (district 6, island
  east half ~x+9) — leave x ≥ +6 clear for that build.

## 3. Coordinate frame & collision discipline

- Island, around `{+3, −74…−79, 0}` (the portico is the existing anchor at
  `{3,−74,0}`); the temple extends **south** (y more negative) and branches E/W.
- One proper vertical: a gallery above the nave/crossing (z+1).
- **Exact coords below are the build target and MUST be `cartcheck`-verified
  (mode=panic) against all built districts before commit.** No rooms occupy the
  island interior yet; keep everything within x [+2,+5] (plus 6183 at x+5) so the
  Cloisters slot in east. South of y−79 stays clear.

## 4. Rooms (6168–6183, 16) — axial temple floor plan

| roomid | title | coord (x,y,z) | exits | role |
|--------|-------|---------------|-------|------|
| 6167 | The Temple Portico | 3,-74,0 | **add S→6168** | (5a) doors now open by day |
| 6168 | The Narthex | 3,-75,0 | N→6167, S→6169, W→6180 | vestibule; the hush at the threshold |
| 6169 | The Public Nave | 3,-76,0 | N→6168, S→6172, W→6170, E→6171, U→6181 | the long worship hall, central axis |
| 6170 | The West Aisle | 2,-76,0 | E→6169, S→6174 | side aisle; candle-racks |
| 6171 | The East Aisle | 4,-76,0 | W→6169, S→6175 | side aisle; pilgrims |
| 6172 | The Crossing — Great Hall over the Waters | 3,-77,0 | N→6169, S→6173, W→6174, E→6175 | **centerpiece**: grate over the joined channel |
| 6173 | The Sanctuary | 3,-78,0 | N→6172, S→6178, W→6176, E→6177 | altar + sacred fire |
| 6174 | Chapel of the Aldren | 2,-77,0 | E→6172, N→6170, S→6176 | north-river chapel |
| 6175 | Chapel of the Brenn | 4,-77,0 | W→6172, N→6171, S→6177, E→6183 | east-river chapel |
| 6176 | Chapel of the Solt | 2,-78,0 | N→6174, E→6173, S→6179 | southwest-river chapel |
| 6177 | The Disused Alcove | 4,-78,0 | N→6175, W→6173 | walled-over fourth space (storage); the understated forgotten-fourth beat |
| 6178 | The Reliquary | 3,-79,0 | N→6173, W→6179 | relics; oldest rests on pre-Founding stone |
| 6179 | The Sacristy | 2,-79,0 | E→6178, N→6176 | working vestry (sacristan) |
| 6180 | The Almonry | 2,-75,0 | E→6168 | alms/candles for the poor; daily temple life |
| 6181 | The Gallery | 3,-76,1 | D→6169, S→6182 | upper level (proper vertical) |
| 6182 | The Gallery Walk | 3,-77,1 | N→6181 | overlooks the crossing/waters from above |
| 6183 | The Inner Threshold | 5,-77,0 | W→6175, **E→stub (future Cloisters)** | warded door/stair to the cloisters; Keeper turns the public back; **Q74 lead-in** |

Room count: 16 (6168–6183). Biome `city` throughout (matching the island/temple
stone), consistent with 5a. Every exit above is reciprocal — verify each pair at
build. The gallery (6181/6182) is the only vertical (z+1, stacked over 6169/6172).

## 5. NPCs (mobs/dialogue 9457–9463, ~7)

All clergy `groups: [..., keepers]`, `non_combatant: true`. Senior Keepers
(Aldric / Brother Cael / Prioress Crane) are **reserved for the Cloisters (district
6)** — this district uses public/mid-level clergy only.

| mobid | name (Title Case) | groups | where | role |
|-------|-------------------|--------|-------|------|
| 9457 | The Officiant | [humanoid, keepers] | 6172 Crossing | **anchor**; leads the public rite over the waters; the fullest official account (the temple built for the confluence; the symbol = Chrysalis truth); a **Q73-gated aside** (loyal-but-aware); points to the senior Keepers beyond — **grants nothing**. |
| 9458 | The Sacristan | [humanoid, keepers] | 6178 Reliquary | tends relics; lore on the oldest relic + the pre-Founding stone it rests on (threshold-only); optional small Q73 aside. |
| 9459 | The Threshold Warden | [humanoid, keepers] | 6183 Inner Threshold | gates the inner door; turns the public back ("the inner precincts, and what lies beneath, are for the Keepers"); the **visible Q74 threshold**. |
| 9460 | The Almoner | [humanoid, keepers] | 6180 Almonry | alms/candles; warm, charitable daily life. |
| 9461 | A Kneeling Worshipper | [humanoid] | 6172 Crossing | ambient, noncombat_passive; kneels at the grate over the waters. |
| 9462 | A Pilgrim at Prayer | [humanoid] | 6174 Chapel | ambient, noncombat_passive; lights candles at a river-chapel. |
| 9463 | A Temple Visitor | [humanoid] | 6169 Nave | ambient, noncombat_passive; takes in the nave. |

No Margin presence inside the temple (the Margin scholar stays at the portico, 5a).
**No quest** (Q74 is district 6).

## 6. Items

Minimal — reuse the 5a devotional goods. **0–2 new** (40142+ only if a printed
liturgy / Keepers' seal-token genuinely earns a slot). The relics, the grate, and
the altar are **lore nouns**, not items.

*Nice-to-have (optional, light):* the feel-review noted votive offerings bought in
5a have nowhere to be used. The crossing/sanctuary may carry an **offering brazier /
censer noun** whose description acknowledges pilgrims leaving candles and wreaths —
**flavor only, not a mechanic** (no item-consumption wiring this build). A real
"use a votive at the altar" interaction is deferred.

## 7. Q74 seeding (no grant)

Threshold-only escalation, paying off across the island:
1. **The Crossing grate (6172)** — the three rivers join in one channel directly
   below the floor; the floor here is the oldest stone in the temple; the sense the
   temple was placed *for* this point.
2. **The Reliquary (6178)** — the oldest relic rests on visibly pre-Founding stone,
   older than the temple around it.
3. **The Disused Alcove (6177)** — a walled-over fourth chapel-space, repurposed;
   "no one now remembers what it was consecrated to." One understated beat.
4. **The Inner Threshold (6183)** — the warded door/stair the public cannot pass;
   the Warden names the senior Keepers and "what lies beneath" as closed. The clear,
   visible Q74 lead-in.
5. **Q73 echo** — the Officiant (and optionally Sacristan) give Q73-completed players
   a knowing-but-loyal aside, gated on the Q73 end token (`questRequired: ["73-end"]`).

No crash, no gray material, no mutation link.

## 8. Schedules (anchor NPCs)

Follow the established pattern (day-at-post + night, findability-preserving). Likely:
the Officiant (rites by day at the crossing), the Sacristan (reliquary/sacristy), the
Threshold Warden (always at the door), the Almoner (almonry by day). Keep ambient
worshippers unscheduled. Author so the world-critic/feel checks pass.

## 9. Process & verification

Same as 5a:
1. Subagent-driven build (revise 6167; rooms; mobs/dialogue; schedules; items if any).
2. **Boot** `ValidateZoneConsistency` mode=panic + `cartcheck the_confluence` clean
   (reciprocal exits, no collisions, the gallery vertical, the 6183 east stub).
   Instance-wipe before smoke per SOP.
3. **Polish pass (mandatory):** world-critic + feel-tester over 6167–6183. Recurring
   traps to check explicitly:
   - **River/compass directions** — Aldren N, Brenn E, Solt SW; combined water spills
     SW; Scholars' Quarter upriver/N. Double-check every direction word and that the
     three chapels match their rivers' source directions.
   - **Dialogue node-shadowing** — `strings.Contains(topic, trigger)`; gated nodes
     FIRST; questRequired/questExcluded as LISTS; **no grantsQuest anywhere**.
   - Mob names Title Case; idlemessages with colon-space quoted; multi-word nouns
     hyphenated (token + key); descriptions/nouns with colons use `>` block scalars;
     exits carry **no `kind:` field** (mapper-derived).
4. No quest to harness-verify; verify the 6167↔6168 entry, the Officiant/Sacristan/
   Warden dialogue (incl. Q73-gated asides), the grate/relic/alcove nouns, the
   gallery vertical, and the 6183 warded stub.

## 10. Out of scope

- Q74, the Cloisters, the Undercroft, and the senior Keepers — district 6.
- The working barge transit (deferred city-wide).
- A real "use a votive offering at the altar" mechanic (deferred nice-to-have).
- The east-stub destination rooms (built in district 6).

## 11. World impact

World grows by 16 rooms and ~7 mobs. The Confluence reaches 5 full districts + the
temple island's public half (~74 of its ~152 rooms), with the warded inner threshold
positioned for the Cloisters/Undercroft climax build.
