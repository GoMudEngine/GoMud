# The Confluence — District 6b: The Undercroft (Q74 climax)

**Date:** 2026-06-27
**Status:** Approved (design phase)
**Umbrella:** `docs/superpowers/specs/2026-06-26-confluence-citywide-design.md` (§3 row 7, §5, §6)
**Predecessor:** District 6a — Cloisters & Archive, merged `be0aa8d3`. Q74 sits
in-progress at the `74-descent` step; the descent stairhead (6199) has a described
`down` stub this build wires.
**This is the climax** — the payoff for the pre-Founding mystery seeded across New
Plymouth, the Southern Road, and the whole Confluence.

## 1. Concept

The descent beneath the temple island — three levels down through pre-Founding
construction the temple was unknowingly built on. The architecture sheds its temple
character as the player descends, until the **sealed chamber** at the bottom holds the
**orbital "face": the old sky rendered with four rings** — the fourth water/moon that
was lost. The **threshold reveal**: there *was* a fourth; the sky changed; the temple
sits on a suppressed pre-Founding record; the faith is a reinterpretation. The player
leaves understanding *something is missing* and is pointed *onward* — never the why.

**LORE BOUNDARY (absolute):** threshold only. The undercroft proves there was a
fourth and that the temple suppressed/reinterpreted it. It NEVER states the why — the
crash, the gray material, the mutation link. **That is reserved for the crash-site
zone (roadmap #22).** No numerology lectures; environmental + a few placed beats.

## 2. Geography (18 rooms, 6200–6217, z−1…−3, beneath the island)

Wire **6199 `down` → 6200** (z−1). The undercroft lies beneath the temple island
(roughly under x+3..+9, the island footprint), descending z−1 → z−2 → z−3. Within a
level, exits are cardinal; between levels, `up`/`down` with stacked coords (z steps).
**Exact coords are the build target and MUST be `cartcheck`-verified (mode=panic).**
The undercroft occupies negative-z space no other district uses; keep its x,y within
the island footprint and verify no collision with z0 island rooms (different z = no
collision, but the down/up exits must be reciprocal and the deltas clean).

- **z−1 (6200–6205, 6rm) — the Upper Undercroft.** Where the temple's own foundations
  meet older work; an early-Keepers crypt; storerooms; the seam from the stairhead.
  The transition: temple-built masonry giving way, course by course, to something
  older. Cool, dripping, lamplit.
- **z−2 (6206–6211, 6rm) — the Old Halls.** Pre-Founding construction proper; you are
  now at the level of the joining waters (the great hall is far above). The
  architecture is unmistakably not the temple's — a different geometry, a different
  hand. **The ward/guardian (§4) is here, barring the stair down.** A water-gallery
  where the three channels run close.
- **z−3 (6212–6217, 6rm) — the Deep / the Sealed Chamber.** The deepest level: an
  antechamber, the sealed door, and the **chamber of the orbital face** (the climax),
  plus 1–2 atmosphere rooms. Q74 completes from here.

## 3. NPCs (mobs 9471+)

A dungeon, not a populated district — presences, not a roster:
- **The Ward-Construct guardian** (z−2, §4) — the one combatant.
- **Optional:** Aldric or Brother Cael may descend partway as a *voice at the
  threshold* (a scheduled/placed presence on z−1 who will not go lower — reinforcing
  that even the senior Keepers stop here), giving the climax a human anchor. Keep
  light; mostly the undercroft speaks through stone and inscription.
- A couple of ambient non-threats if useful (e.g., blind cave fauna in the
  water-gallery), threshold-neutral.

## 4. The guardian / wards

**One pre-Founding ward-construct** on z−2, barring the stair to z−3.

- **Strength (locked, tune from playtest):** `statpool: ~100`, `archetype: fighting`
  — the toughest single mob in the Confluence, below the combat-zone raid elites
  (120–130). Calibrated so a developed Q73→Q74 character can win and an underprepared
  one should flee. (Reference: the Q70 Canal Lurker optional fight is statpool 30;
  Ironwind/Pothole elites are 120–130.)
- **Profile:** tanky + slow + telegraphed — high **vitality** + high
  **physical_mitigation** (pre-Founding stone: lots of HP, soaks hits), low
  **dexterity** (swings slow/inaccurately, few attacks/round). Moderate **strength**
  damage. **No ranged, no spells.**
- **Optional by design:** it *guards*, it does not *hunt* — low/no aggro,
  `maxwander: 0`, `hostile: false` (rouses when the player forces the descent / attacks
  / passes the threshold, not on sight). A careful or Keeper-sanctioned approach can
  slip past, so players who came for the reveal aren't hard-gated by a fight.
- A construct, not a Keeper: **`non_combatant: false`** (attackable), `groups:` not
  keepers (the old site's own ward). May apply a "chill of the deep" debuff buff
  (94+) on hit — optional; only if it earns its keep.

## 5. Q74 — completion (the second half)

6a left Q74 in-progress at `74-descent` (the full step skeleton — start, ledger,
record, survey, descent, **reveal, end** — is already declared in
`quests/74-the_undercroft.yaml`; the `74-allegiance` flag is declared with values
`[margin, keepers]`). 6b wires the back half:

1. **Descend:** 6199 `down` → 6200. The descent satisfies the `74-descent` hint;
   exploring downward is the gameplay. (No new token needed to descend — the player
   already holds `74-descent`.)
2. **The reveal (z−3 sealed chamber):** a quest-gated `room_interact` on the orbital
   **`orbital-face`** noun (`look orbital-face`, gated `has: [74-descent]`,
   `missing: [74-reveal]`) → **grant `74-reveal`** + the threshold-reveal send_text
   (the four-ringed old sky; the fourth that was lost; the temple built atop the
   suppressed record). Supporting inscription/relief nouns nearby (threshold-only).
   An ungated fallback for a player who somehow looks without the quest.
3. **The allegiance choice → completion (soft flag, by report-target):** after
   `74-reveal`, the player returns up and reports. Two completion nodes, each gated
   `questRequired: ["74-reveal"]`, `questExcluded: ["74-end"]`, each `grantsQuest:
   "74-end"`:
   - **To Aldric** (cloisters, 9464): `setsQuestFlag {74-allegiance: keepers}` + a
     keeper-rep bump → "you keep the Keepers' confidence."
   - **To the Margin** (the portico scholar 9454, or Quist 9441 in the Scholars'
     Quarter): `setsQuestFlag {74-allegiance: margin}` + a margin-rep bump → "you
     carry the truth out."
   Both complete Q74 identically except the flag + which faction's rep moves — a soft
   arc, not a content fork. (Rep mechanism: prefer the quest-engine `bump_rep` action
   or dialogue equivalent used in Q67; pin the exact mechanism in the plan after
   verifying against a shipped example.)
4. **Rewards (on `74-end`):** finalize the Q74 `rewards:` block — gold + a
   flag-neutral undercroft keepsake item (~40142, e.g. a rubbing/sketch of the old
   sky, `not_salable`) + the `playermessage` that points *onward* (the answer is not
   here; the thread continues — never naming the crash). Faction rep is delivered by
   the report node (flag-specific), not the static reward block.

**Reference the shipped Q73/Q74 patterns** (`quests/73-the_margin_notation.yaml`,
`quests/74-the_undercroft.yaml`) and the verified conventions: flag key is **bare**
(`allegiance`) in the declaration, referenced `74-allegiance`; completion/rewards fire
only on the step named `end`; a trigger may only grant a declared step; grant nodes
exclude the end token + carry `quest`/`task` triggers; `room_interact` nouns are
ansi-highlighted hyphenated tokens with matching keys.

## 6. Items / buffs

- **~1 item (40142):** the undercroft keepsake / old-sky rubbing (Q74 reward,
  `not_salable`). Possibly a second atmosphere item if warranted; keep minimal.
- **Buffs (94+):** only if the guardian applies a debuff (optional "chill of the
  deep"). No buffs if not needed.

## 7. Schedules

The undercroft is largely unscheduled (a dungeon). If Aldric/Cael appear as a
threshold voice on z−1, give a simple presence (a placed spawn or a light schedule
segment) — but do not pull Aldric out of the cloisters in a way that breaks his 6a
Q74 turn-in/permission availability. Simplest: a *separate* threshold presence, or no
descended Keeper at all (decide at build; lean toward keeping it spare).

## 8. Process & verification

Same proven cycle, with the fullest harness check yet:
1. Subagent-driven build: wire 6199 `down`; 18 rooms (3 levels); the guardian; the
   reveal + completion dialogue/quest wiring; the reward item; (optional buff).
2. **Boot** `ValidateZoneConsistency` mode=panic + `cartcheck the_confluence` clean
   (reciprocal up/down across z-levels, no collisions, quest loads, flags validated,
   no "unknown step" panic now that reveal/end are granted).
3. **World-critic + feel pass** (mandatory): direction/level canon (down is deeper;
   the waters are at z−2; the reveal is z−3); dialogue node-shadowing (the two
   completion nodes gated correctly; no double-grant); **lore-boundary (the single
   most important check here — the reveal must stay threshold-only, never the why)**;
   casing; the guardian's strength *feel* (beatable + avoidable).
4. **Full Q74 end-to-end harness-verify:** fresh Q73→Q74 chain → descend → guardian
   (fight + verify it's beatable, and verify the slip-past) → `look orbital-face`
   (74-reveal) → report to Aldric (74-allegiance=keepers, 74-end, keeper rep, reward)
   AND, on a second run, report to the Margin (74-allegiance=margin, 74-end, margin
   rep) → confirm Q74 completes (100%) both ways with the correct flag/rep.

## 9. Out of scope

- The why (crash, gray material, mutation link) — reserved for the crash-site zone.
- The outer quarters (Craftsmen's/Residential + East Gate) — the next/last district(s).
- Any post-Q74 content hanging off the allegiance flag beyond the soft rep/flag (the
  flag is recorded for future use; no new content forks on it here).

## 10. World impact

World grows by 18 rooms + a few mobs. **The Confluence's spine is complete** — the
whole island (public temple + inner cloisters + the undercroft) and Q74, the keystone
quest, fully playable end-to-end. Only the outer residential/gate quarters remain to
finish the city.
