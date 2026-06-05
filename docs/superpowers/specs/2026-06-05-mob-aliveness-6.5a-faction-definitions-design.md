# Mob Aliveness 6.5a — Faction Definitions

**Status:** Design — pending user review
**Roadmap chunk:** 6.5a (Phase 6 / Polish), Size M, precursor to the 6.5 content batches
**Date:** 2026-06-05
**Depends on:** 1.2 (faction system), 1.3 (crime/wanted). Must land BEFORE the
6.5 town/wilderness/road content batches (they tag against this roster).

## Goal

Author the remaining world factions on the 1.2/1.3 substrate so the 6.5 content
batches tag mobs against a settled roster + ally/enemy graph, rather than
inventing factions ad hoc. Establishes a coherent "civilization vs outlaw"
political map and reserves a slot for the Bloodline Agents (a future-major force).

## Mechanism recap (from 1.2)

- A faction is a YAML at `_datafiles/world/dogmud/factions/{faction_id}.yaml` with:
  `faction_id`, `display_name`, `description`, `default_rep` (int, the player's
  starting reputation), `allies: [faction_id...]`, `enemies: [faction_id...]`.
  Guard/justice factions additionally carry `holding_cell_room` + `release_room`.
- **Membership = the `faction_id` string appears in a mob's `groups:` list.**
  `factions.FactionsForMob(mob)` returns the subset of `mob.Groups` that match a
  registered faction. (`groups:` is a multi-purpose tag list — e.g. the Bloodline
  Agent is `groups: [humanoid]` today and becomes `[humanoid, bloodline_agents]`.)
- ally/enemy edges are **explicit on both sides** (not auto-mirrored — see warren
  ↔ thornwall_guards, each declares the other). The boot-time loader **panics on
  unresolved ally/enemy references**, so the pre-push boot test catches typos.

## Existing factions (5 — do not recreate, but edges get updated)

| faction_id | default_rep | current allies | current enemies |
|---|---|---|---|
| thornwall_guards | 0 | thornwall_citizens | warren ← **REMOVE** |
| thornwall_citizens | 0 | thornwall_guards | — |
| stillwater_guards | 0 | stillwater_citizens | — |
| stillwater_citizens | 0 | stillwater_guards | — |
| warren | −25 | — | thornwall_guards ← **REMOVE** |

(So "Stillwater militia & citizens" from the 6.5a brief already exist.)

**Correction to 1.2:** the mutual `warren ↔ thornwall_guards` enemy edge
mischaracterizes the Warren — see the warren section below; 6.5a removes it.

## New factions (8)

| faction_id | default_rep | role | bloc |
|---|---|---|---|
| bandits | −35 | raiders across wilderness + roads | outlaw |
| ironwind_tribe | −25 | territorial steppe goblin tribe (shaman-led) | outlaw |
| dustwalk_caravans | +10 | merchant trains on the trade roads | law bloc |
| road_wardens | 0 | road law / patrol escorts | law bloc |
| shopkeepers | 0 | chamber-of-commerce merchant solidarity | law bloc |
| ashwick_villagers | 0 | the farming village of Ashwick | law bloc |
| watchers_crossing | 0 | the waystation settlement at Watcher's Crossing | law bloc |
| bloodline_agents | 0 | shadow force (future-major) | **neutral placeholder** |

The two settlement factions came out of the 6.5b towns-batch NPC inventory:
Ashwick (~4 townsfolk) and Watcher's Crossing (~4 townsfolk) are micro-settlements
but get their own citizen factions for group cohesion, parity with
Stillwater/Thornwall. They have no guards of their own — law-bloc membership means
road_wardens + the town guards react to crimes against them.

## The ally/enemy graph

### Unified law bloc (mutually allied clique)

These nine factions are **all mutually allied** — wronging one ripples across
the whole "civilization" bloc, and this links the previously-islanded Thornwall
and Stillwater blocs:

`thornwall_guards`, `stillwater_guards`, `road_wardens`,
`thornwall_citizens`, `stillwater_citizens`, `dustwalk_caravans`, `shopkeepers`,
`ashwick_villagers`, `watchers_crossing`

Each lists the other eight in its `allies:`. (Verbose but explicit, as the schema
requires both-sided edges. This verbosity is the cost of the cohesive-bloc choice
and is intentional. Consider a brief authoring helper/comment block so the clique
stays in sync if edited.)

### Outlaw cluster (enemies of the law bloc)

- **bandits** — `enemies:` = the full law bloc (9); `allies:` = `[ironwind_tribe]`
  (loose, circumstantial).
- **ironwind_tribe** — `enemies:` = the law bloc (9); `allies:` = `[bandits]`
  (loose). Thematic note: ironwind goblins are remote steppe-dwellers, so the
  bandit alliance is "circumstantial enemy-of-civilization," not a coordinated
  pact.

### warren — NOT an outlaw faction (reframed)

The Warren are **not outlaws and not enemies of the law**. They are an insular
colony of small-folk who want to be left alone — *discriminated against and
looked down upon*, rather than actively doing harm like bandits. So:

- **Remove** the mutual `warren ↔ thornwall_guards` enemy edge (a 1.2
  mischaracterization). The Warren are mechanically **neutral** to the law bloc:
  no enemy edges, no ally edges.
- Keep `default_rep: -25` — this represents a stranger starting *mistrusted*
  ("surface-dwellers are mistrusted on sight" per their description) and earnable
  back via the Warren Compact (quest 2, `bump_rep: warren +30`). Mistrust, not
  hostility.
- The "discriminated against / looked down upon" theme is carried by **flavor**
  (NPC dialogue, low NPC→player opinion, townsfolk attitudes) and by the negative
  default_rep — **not** by a faction enemy edge. The faction allies/enemies model
  is binary (hostile vs not); there's no "dislikes-but-tolerates" tier, and
  forcing an enemy edge would make guards attack Warren on sight, which is wrong.

**Behavior-change note:** removing the enemy edge means killing a Warren member no
longer triggers thornwall_guards-enemy reactions / rep mechanics that keyed off
that edge. This is the intended correction (the world should not treat the Warren
as criminals), but flag it at merge so it's a conscious change, not a silent one.
warren is NOT added to the outlaw cluster.

### bloodline_agents — neutral placeholder

`default_rep: 0`, `allies: []`, `enemies: []`. The faction exists and the lone
agent is tagged, but its political alignment is deliberately uncommitted until the
Bloodline storyline is designed ("major force later").

## Member tagging (world-wide; batches then consume the tags)

6.5a owns all faction definitions + the graph + member tagging. Add the
`faction_id` to each member mob's `groups:` list. Anchor members identified:

- **bandits:** `80-dustwalk_bandit`, `253-road_bandit`, `254-bandit_leader`,
  `283-bandit_lookout`, `284-bandit_fighter`, `285-bandit_caster`, `286-soren`
  (bandit-camp boss), `90-thornwall_highwayman`, `105-thornwall_thug`. Implementer
  scans all wilderness/road zones for remaining bandit-type mobs.
- **ironwind_tribe:** the goblins — `217-goblin_scout`, `218-goblin_scrapper`,
  `219-goblin_shaman` (the shaman the brief named). The steppe **wildlife/wolves
  are NOT faction members** (they get pack *relationships* in the wilderness
  batch, not faction membership).
- **dustwalk_caravans:** `281-caravan_master` + caravan crew/escort mobs (the
  `caravan` group members on the trade roads).
- **road_wardens:** `83-road_warden_tessara` + road patrol/escort mobs.
  (`241-windwarden_sylara` is a quest boss — exclude unless intended.)
- **shopkeepers:** all merchant mobs that own a shop (`Shop`/`ShopInventory`),
  e.g. `337-smith_brindle`, `338-apothecary_ilsa`, `339-weaver_edda`,
  `340-pearl_carver_kess`, plus Thornwall + Ashwick merchants. Implementer
  enumerates by "has a shop."
- **ashwick_villagers:** `259-delia`, `260-deacon_ferris`, `261-farmer_hesta`,
  `262-the_forager` (the ashwick wildlife — fox/wolf/hawk/chicken/mouse — are NOT
  members). `262-the_forager` likely runs the forager archetype; faction
  membership is independent of behavior.
- **watchers_crossing:** `84-innkeeper_tolva`, `86-toll_collector_harn`, plus
  `85-merchant_brecca` and `88-traveling_merchant` as **dual members**
  (`groups: [..., watchers_crossing, shopkeepers]`). `87-river_lurker` is wildlife.
- **bloodline_agents:** `287-bloodline_agent` (the lone member).
- **thornwall_outskirts** (not a town — light treatment): tag `89-farmer_dorn` →
  `thornwall_citizens`, `92-city_gate_guard` → `thornwall_guards`,
  `90-thornwall_highwayman` → `bandits` (`91-crop_pest` is wildlife). No
  settlement faction; handled as light faction-tagging, not a town pass.

Member enumeration is finalized at implementation via a per-zone scan; the anchors
above are the confirmed seeds.

## default_rep rationale

- bandits −35 / ironwind_tribe −25: start disliked (outlaws); bandits a touch
  lower as active raiders. (warren is −25 for parity.)
- dustwalk_caravans +10: friendly merchants, mildly positive to strangers.
- road_wardens 0 / shopkeepers 0 / bloodline_agents 0: neutral — earned, not given.

## Out of scope

- **bounty_hunters faction** — deferred (5.2 functions without it; author later if
  a concrete need appears).
- **Bloodline Agents' political graph** — deferred to the Bloodline storyline.
- **road_wardens justice integration** (holding_cell_room/release_room + arrest) —
  omitted; wardens fight bandits rather than jail players in v1. Add if road
  justice is later wired.
- **Per-faction quests** — separate content chunk.
- **Relationship edges / schedules / conversations** — those are the 6.5 content
  batches; 6.5a is factions only.

## Validation

- `go build` clean (no code changes expected — pure data — but the faction loader
  runs at boot).
- **Boot test is the key gate:** the faction loader panics on unresolved
  ally/enemy references, so a clean boot past data-file load proves the graph is
  internally consistent. Watch for `factions.LoadDataFiles()` style load logs
  without panic.
- Spot-check in-game with `faction list` / `faction show <slug>` (admin) and
  confirm a tagged mob resolves via `FactionsForMob` (e.g. attack a bandit, see
  the bandit-faction rep hit; verify a law-bloc member reacts).

## Files touched

- New: `_datafiles/world/dogmud/factions/{bandits,ironwind_tribe,dustwalk_caravans,road_wardens,shopkeepers,ashwick_villagers,watchers_crossing,bloodline_agents}.yaml` (8).
- Edit: the 5 existing faction YAMLs — extend `allies:`/`enemies:` for the law-bloc
  clique + warren decision.
- Edit: member mob YAMLs — append faction ids to `groups:` (anchors above + the
  per-zone scan results).
- Edit: `internal/factions/context.md` if the faction count / graph notes are
  documented there.

## Implementation note

This is content (YAML) authoring — a good fit for the CLAUDE.md parallel
content-creation pattern, but factions are few and interdependent (the graph), so
a single focused pass is simplest. No ID-block allocation needed (faction_ids are
slugs, not numeric).
