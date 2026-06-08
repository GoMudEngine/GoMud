# Mob Aliveness 6.5d — Roads Batch (light)

**Status:** Design approved — pending spec review → implementation plan
**Roadmap chunk:** 6.5d (roads/travel batch — final batch of the 6.5 content pass), Size S
**Date:** 2026-06-05
**Depends on:** 6.5a (factions — road bandits/wardens/caravan already tagged), 3.7 (inter-zone caravans/patrols)

## Goal

A light aliveness pass on the road zones: road-danger gossip voiced by the
travelling folk (thematically perfect — the bandits are co-located on the same
roads) plus a couple of obvious relationships, and a confirmation that the 3.7
caravan still resolves. Closes out the 6.5 content pass.

## Why this is small

- `world_road` — 0 mobs (empty).
- `dustwalk_road` — bandit + 2 beasts + road_warden_tessara (faction-tagged in 6.5a).
- `marches_spur_road` — peddler Malk, innkeeper Thessa, traveler Bren, old Edrin
  (a boss) + bandits + beasts.
- `north_road` — ~9 sentient road-folk (Corvin, Betta, Haral, Dessa, Tam, caravan
  master, farmer, woodcutter Hagen, lone traveler) + the bandit camp + the
  Bloodline agent + beasts.

Faction-tagging is done (6.5a). The genuine new content is road-danger gossip +
two relationships. No schedules, no conversations, no behavior changes. Data-only,
validated at boot.

## 1. Relationships

Author once (auto-mirror). `type` per the relationship's nature:

- **innkeeper_thessa 251 ↔ peddler_malk 250** — `friend` (subtype: `waystation`).
  The two regulars of the Marches Spur stop. Author on `251-innkeeper_thessa.yaml`.
- **caravan_master 281 → crew** — `employer` (subtype: `caravan`) to ketil 357,
  marta 358, lars 359 (the crew tagged `dustwalk_caravans` in 6.5a; the engine
  mirrors to `employee`). Author on `281-caravan_master.yaml`.

## 2. Facts + gossipers

Two road-scoped facts (distinct from the existing zone-scoped bandit facts:
`watchers-road-bandits`, `thornwall-road-bandits`, `ironwind-tribe-pressing`).
Append to `_datafiles/world/dogmud/facts.yaml`. Public/observable troubles — no
quest spoilers (gossip is additive, separate from quest dialogue).

| fact id | description | gossipers (`knows_facts` + `gossiper`) |
|---|---|---|
| `roads-bandit-peril` | The trade roads have grown dangerous; bandits ambush the lonely stretches, so folk travel in numbers. | road_warden_tessara 83, lone_traveler 329, woodcutter_hagen 328, traveler_bren 252, corvin 276, farmer 282 |
| `roads-caravans-guarded` | The caravans cross under hired guard now, running the gauntlet between the towns. | caravan_master 281, peddler_malk 250, innkeeper_thessa 251 |

Tags: `roads-bandit-peril` → `[roads, bandits, crisis]`; `roads-caravans-guarded`
→ `[roads, trade]`. `roads-bandit-peril` cross-ties to the 6.5a `bandits` faction
(the threat is literally in these zones).

## 3. Caravan / route verification (confirm, don't author)

No patrol files live under the road zone folders; the 3.7 inter-zone caravan runs
the north road via `caravan_master 281` + crew. 6.5d does NOT author new patrols —
it confirms at boot that the caravan and any inter-zone patrols still resolve
(they load via the caravan/patrol systems; a clean boot is the check). If the
caravan fails to resolve, that's a pre-existing 3.7 issue to log, not 6.5d scope.

## Validation

- **Boot test (hard gate):** relationships loader panics on unknown `to:` mob;
  facts loader panics on unresolved `knows_facts:` id. Clean boot = pass. Also
  confirm no caravan/patrol resolution error in the boot log.
- **In-game smoke** (may be deferred to user): approach the road-folk and confirm
  the danger facts surface as gossip; confirm the caravan still runs the north road.

## Out of scope

- `world_road` (empty), pure-beast scatter, the bandits themselves (hostile).
- old_edrin (boss) and the Bloodline agent (faction placeholder) — leave as-is.
- north_road "hamlet" full treatment (relationships/schedules for Corvin/Betta/
  Dessa/Tam) — deliberately NOT done; kept to gossip only (the user chose the
  lighter scope). Logged as a possible future deepening.
- Schedules / conversations / behavior changes.
- Authoring new patrol routes (3.7 owns that).

## Files touched (anticipated)

- Edit: `251-innkeeper_thessa.yaml`, `281-caravan_master.yaml` (relationships).
- Edit: `facts.yaml` (+2 facts).
- Edit: the 9 gossiper mob YAMLs (`knows_facts:` + `gossiper` group tag) —
  83, 329, 328, 252, 276, 282 (peril); 281, 250, 251 (caravans-guarded).
- Edit: roadmap (6.5d status; mark the 6.5 content pass complete).
