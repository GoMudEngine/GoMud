# The Bloom Trail Questline (design)

**Status:** approved 2026-06-24 (brainstorming). The capital's marquee multi-district
questline — the interactive payoff of the Bloom Trail breadcrumbs placed across all 7
districts, built on the now-shipped **Bloom mechanic**
([[project-bloom-mechanic]] / the toxicity+Bloom merge `04f13230`). Quests-phase
enrichment (after economy-depth + the Bloom mechanic).

> **Push policy:** capital is built + prod-ready, push HELD by user. This merges to
> master; push stays held until the user calls it.

## 0. Scope (locked with user 2026-06-24)

A **two-quest chain**, opened by an **addict's plight** (environmental, no formal
quest-giver), with an **opt-in undercover dosing beat** and a **bittersweet expose**
ending (Deren is watched/untouchable — you build the case, the bloodline office
shields him). Uses EVERY placed Bloom-trail NPC + the Bloom mechanic + Ysolde's detox.

IN: 1 new named addict mob + dialogue; **2 chained quests** (66 The Addict's Plight,
67 The Bloom Trail) with **quest flags** for the undercover-vs-alternate branch; a few
**evidence items**; **dialogue additions** to the existing trail NPCs (Marn, Falk,
Wenna, Deren, Quill, Ysolde, Dock Constable) carrying their quest beats; **faction
consequences** on completion.

OUT: no new rooms (the trail uses built rooms); no Go (quest engine + the shipped
Bloom mechanic only); no killing Deren / no captive rescue (canon: untouchable +
captive gone); the bloodline_domestic office stays untouchable (theme).

**Locked decisions:** chain (both quests) · addict's-plight environmental hook ·
opt-in undercover beat (always skippable via an alternate path) · bittersweet expose
resolution.

## 1. IDs, factions, flags

- **Quests:** **66** (The Addict's Plight), **67** (The Bloom Trail). Q67 gated on
  Q66 completion. (Next-free quest id is 66; the `1000000` in the tree is a junk
  artifact, ignore.)
- **Mob:** **9392** — the named addict (the anchor). Spawn in an accessible early
  room (Docks waterfront or Common — see §2).
- **Items (evidence + quest props):** **40110+** — a small set (see §4): e.g. a
  spent wax wrapper (evidence), a Bloom buyer's-token / cover-purchase prop, the
  collated **case-file** the player assembles. (Reuse the wafer 40108 / Ysolde's
  Purge 40109 from the mechanic — do NOT recreate.)
- **Factions (existing):** `bloom_trade` (rep ↓ on expose), `np_dockfolk` /
  `np_commonfolk` (rep ↑), `bloodline_domestic` (the shield — referenced, untouched).
  No new faction.
- **Quest flags (declare in the quest YAML — undeclared refs PANIC):**
  - `66-addict-fate` — values `[saved, lost]` (the addict's outcome; default path =
    saved). Set in Q66/Q67.
  - `67-entry` — values `[undercover, evidence]` (which way the player got past Marn's
    back room: dosed Bloom vs the alternate evidence/skullduggery path).

## 2. The named addict (mob 9392) — the anchor

A named Bloom addict, the human face of the trade and the chain's emotional center.
Distinct from the ambient Bloom-addled wanderers (Docks 9317 / OQ 9383) — this one has
a name, a history, and a fate. Life-sheet:
- **Name:** a Title-Case name (collision-checked at build), e.g. *Teels*, *Hask*, or
  similar — a former dockworker or craftsman the Bloom hollowed out.
- **Where:** an accessible early room — **recommend the Docks waterfront** (near the
  Pilings Haunt breadcrumb / Marn) so the hook is reachable soon after arrival; OR
  Common. `non_combatant: true` (a victim, not a threat).
- **Mutation:** a Bloom-exposure mark (faint copper veining beginning — echoes Deren),
  underscoring that Bloom is mutating them.
- **Dialogue:** craving/lucid alternating; in lucid moments, the plea that pulls the
  player in (the Q66 hook); names where the Bloom comes from only vaguely ("a clean
  shop, a draper" → Marn). The Q66 trigger originates HERE (the plight), not a giver.
- **Schedule:** light — haunts the waterfront/an alley; `activity: sleeping`-ish
  stupor segments (optional, keep simple).

## 3. Quest 66 — "The Addict's Plight" (the personal hook)

**Spine:** encounter the addict → get them to Ysolde → detox + the call to find the
source. Steps (quest schema = steps with id/description/hint):

| step id | what happens | gate / trigger |
|---------|--------------|----------------|
| `start` | The player finds the addict (9392) mid-craving and, talking to them, agrees to help. **Granted via the addict's dialogue** (`grantsQuest: 66-start`; include `"quest"`/`"task"` triggers + `questExcluded: [66-start, 66-end]` per SOPs). | addict dialogue node |
| `escort` | The addict can't make it alone — the player must get them to **Ysolde** (Common 9323). (Implement as: tell the addict to seek Ysolde, then **talk to Ysolde** about the addict — a dialogue/flag step; OR a simple "bring an item from the addict to Ysolde" item_give if escort is awkward in-engine. Prefer the item_give pattern — give Ysolde the addict's wax-wrapper token to advance, mirroring the Dock Rat/Street Sweeper report pattern, since `grantsQuest` doesn't fire `quest_granted`.) | Ysolde dialogue / item_give |
| `detox` | Ysolde administers the detox (gives the player insight into the cure + Ysolde's Purge 40109 as a dispensable). She states plainly: **they will relapse unless the source is stopped.** This is the bridge to Q67. | Ysolde dialogue |
| `end` | The addict is stabilized (for now); the player knows the detox exists and that the trade must be traced to its source. | auto |

**Reward (Q66):** modest gold; `np_commonfolk` + `np_dockfolk` rep; Ysolde's detox
knowledge (narratively — she'll dispense the Purge 40109 on `ask ysolde detox`, which
the mechanic build already wired). **Completing Q66 unlocks Q67** (Q67 `questRequired:
66-end` on its grant node).

## 4. Quest 67 — "The Bloom Trail" (the expose, bittersweet)

**Spine:** follow the trail to the source, gather evidence, confront Deren (untouchable),
report to the Dock Constable → the trade is disrupted but Deren is shielded. The
multi-district trail uses the BUILT breadcrumbs as quest nodes.

| step id | district | node | content |
|---------|----------|------|---------|
| `start` | — | granted by Ysolde (or the addict) after Q66 | "Find where the Bloom comes from." `questRequired: 66-end`; `questExcluded: [67-start, 67-end]`; `"quest"`/`"task"` triggers. |
| `front` | Docks | **Marn the Draper (9305)** | Marn is the clean-fronted dealer. To get into his back room you must earn trust. **THE BRANCH:** |
| | | — **undercover (opt-in):** sample **Bloom (wafer 40108)** to be taken for a buyer — experience the high/crash (the mechanic); set flag `67-entry=undercover`. Ysolde's detox (40109) is the out. | flag + the mechanic |
| | | — **evidence/alt (skip dosing):** present the wax-wrapper evidence / pass a skullduggery-or-search check / a bribe to get in without dosing; set flag `67-entry=evidence`. | flag |
| `lintel` | Merchant | **Falk the Auctioneer (9345)** | `ask falk about property/lintel` → the 215 Lintel Street address (the existing breadcrumb becomes the next pointer). |
| `delivery` | Noble | **Wenna (9369)** | The terrified servant's delivery-house slip — confirms the guarded address + that goods arrive under the bloodline office's eye. |
| `source` | Old Quarter | **215 Lintel St crime scene (6028) + Deren (9379)** | Examine the production-room evidence (apparatus/drain/pallet — built nouns). **Confront Deren** (the built untouchable scene — he's exposed but the watch/bloodline office have eyes; you cannot take him). Pick up the final evidence. |
| `witness` | Old Quarter | **Quill (9380)** | Quill's oblique testimony of the traffic — the witness statement that completes the case. |
| `report` | Docks | **Dock Constable (9316)** | Hand over the assembled **case-file** (evidence item). The Constable acts on what she can — the operation is **disrupted** — but tells you plainly the bloodline office shields Deren himself: a partial win. Set `66-addict-fate` outcome. |
| `end` | — | resolution | The trade disrupted, the truth known, the named addict (Q66) gets a closing beat (§6). Some powers in this city you can reach, and cannot touch. |

**Evidence items (40110+):** a small assembled set — e.g. **40110 a spent wax wrapper**
(picked up early, the physical Bloom trace), **40111 the case-file** (the player
assembles/receives it as evidence accrues, given to the Constable at `report`).
Keep it to ~2 items; reuse room nouns for the rest (the apparatus, tally-marks, the
address are read in-room, not carried). Use **item_give triggers** for the report step
(give.go gotcha: the Constable keeping the case-file is correct; no return needed).

**Reward (Q67):** good gold; **faction shifts** — `np_dockfolk`/constabulary rep ↑,
`bloom_trade` rep ↓ (VERIFY the rep-down mechanism — `rep_faction` is a single
positive grant in the schema; a negative/second faction may need the quest engine's
`set_flag`/action path or a `rep_faction` with negative value — pin at plan time); a
meaningful **item or title** (e.g. a "the trade's enemy" token or a useful reward
item — reuse a verified item id); the emotional closure.

## 5. Dialogue additions to existing trail NPCs

The breadcrumbs are currently LORE; the quest makes them interactive nodes. Each gets
quest-aware dialogue (gated by `questRequired`/`questFlagRequired`), preserving their
existing lore dialogue for non-quest players:
- **Marn (9305):** the back-room gate + the two-path branch (undercover dose vs
  evidence/skullduggery). Sets `67-entry`.
- **Falk (9345):** the existing `property/lintel` pointer becomes the `lintel` step
  advance.
- **Wenna (9369):** the delivery-house slip becomes the `delivery` step (frightened,
  oblique).
- **Deren (9379):** his built confrontation gains a quest-aware beat (the player
  arrives WITH the case half-built; he's still untouchable).
- **Quill (9380):** his traffic-witness beat becomes the `witness` step (testimony).
- **Ysolde (9323):** the Q66 detox step + the Q67 grant + bittersweet framing.
- **Dock Constable (9316):** the `report` step — receives the case-file, disrupts the
  trade, names the bloodline shield.

All per dialogue SOPs: 1st-person NPC text, narrator hints, `"quest"`/`"task"`
triggers on quest nodes, `questExcluded` end-tokens on every `grantsQuest`,
`questRequired` (not `requires`) for gating.

## 6. The named addict's fate (§ branch)

Flag `66-addict-fate` (`saved`/`lost`):
- **Default = saved.** Good-faith completion (Q66 detox + Q67 disrupting the source)
  → the addict is stabilized/recovering at the `end`; a hopeful closing beat (the one
  life the player could actually touch).
- **`lost` (optional darker branch):** if the player never completes Q67 / abandons
  the trail for a long time (or a deliberate wrong choice if one exists), a later
  visit finds the addict relapsed/gone — the cost of the untouchable source. KEEP THIS
  LIGHT for v1: the canonical path saves them; the `lost` branch is a stretch goal —
  if it complicates the quest engine wiring, ship saved-only and defer the branch.

## 7. Build staging (feeds ONE plan; each stage boot-verified)

> Pre-smoke ritual: wipe instance saves; boot-poll `ERROR:.*PANIC`/`fatal error:`,
> not bare "panic". Quests panic at startup on undeclared flags / bad triggers / missing
> end-token exclusions — the boot test catches these.

- **Stage A — the addict mob (9392) + dialogue + spawn + the Q66 hook.** Boot-verify.
- **Stage B — Quest 66 (The Addict's Plight)** YAML (steps, flags, rewards) + Ysolde's
  Q66 dialogue (escort/detox via item_give) + the addict's `grantsQuest`. Boot-verify
  + harness-test Q66 end-to-end.
- **Stage C — evidence items (40110+) + Quest 67 (The Bloom Trail)** YAML (steps,
  flags, the entry branch, rewards) + the Q67 grant (gated on 66-end). Boot-verify.
- **Stage D — the trail dialogue additions** (Marn branch, Falk, Wenna, Deren, Quill,
  Constable report) wiring each step + the faction consequences. Boot-verify.
- **Stage E — the addict's-fate closure** (§6, saved path; lost as stretch).
- **Stage F — harness playtest** the full chain (both quests, both entry paths,
  confirm Deren untouchable, the bittersweet report, the addict closure) → fix → merge
  `--no-ff` (hold push) → update memory.

## 8. Definition of done

- Quests 66 + 67 load (no undeclared-flag/trigger panic); 67 gated on 66.
- The named addict (9392) lives, gives Q66 via the plight; Ysolde detox + the source
  call bridge to Q67.
- The trail plays through all built breadcrumbs (Marn→Falk→Wenna→215 Lintel→Quill→
  Constable) as quest steps; both entry paths (undercover dose vs evidence) work + set
  `67-entry`.
- The undercover beat uses the real Bloom mechanic (dose→high→crash) with Ysolde's
  detox as the out; the alternate path needs no dosing.
- Deren stays untouchable; the report disrupts the trade but the bloodline office
  shields him (bittersweet); faction reps shift; the addict gets a closing beat.
- Re-grant prevention (end-token exclusions), `"quest"`/`"task"` triggers, `item_give`
  for report steps all per SOP. Harness-playtested; report committed. Merge, hold push.

## 9. Honored gotchas

Quest reward YAML keys are tag-less/no-underscore for itemid/skillinfo/playermessage
(snake_case silently no-ops — VERIFY `rep_faction` + any negative-rep mechanism at
plan time) · every `grantsQuest` includes the `{id}-end` token in `questExcluded` ·
quest-granting nodes include `"quest"`+`"task"` triggers · `grantsQuest` does NOT fire
`quest_granted` → use `item_give` triggers for report/auto-complete steps (Dock Rat /
Street Sweeper pattern) · give.go transfers the item before handlers fire (the
Constable keeping the case-file is correct) · prefer `questRequired`/`questFlagRequired`
over `requires` · declare ALL quest flags with allowed values (undeclared = panic) ·
dialogue 1st-person / hints narrator / discoverable triggers · Title-Case mob name
(collision-checked) · only verified item ids in rewards · no `": "` in YAML values.
